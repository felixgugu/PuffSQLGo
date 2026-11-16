package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"puffsql/internal/model"
	"puffsql/internal/sqlshared"
)

const (
	errUnknownDatabase  = 1049
	errQueryInterrupted = 1317
)

// Execute runs a script on one pinned connection, statement by statement.
//
// A pooled MySQL connection keeps an open transaction when it is released, and a later
// execution cannot reach it again. Execute therefore always rolls back whatever the script left
// uncommitted before releasing the connection: a transaction must be committed by the same
// execution that started it.
func (s *Server) Execute(
	ctx context.Context,
	database *string,
	statement string,
	maxRows *int,
	timeoutSeconds *int,
	onSession func(int),
	onPhase func(string),
) model.QueryResult {
	started := time.Now()
	result := model.QueryResult{
		ResultSets: []model.ResultSet{},
		Messages:   []model.QueryMessage{},
	}
	fail := func(err error, line *int) model.QueryResult {
		result.Messages = append(result.Messages, errorMessage(ctx, err, line))
		result.ExecutionTime = sqlshared.ElapsedMillis(started)
		return result
	}
	if timeoutSeconds != nil && *timeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*timeoutSeconds)*time.Second)
		defer cancel()
	}

	phase(onPhase, "open_conn")
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return fail(err, nil)
	}
	defer conn.Close()
	defer rollbackOpenTransaction(conn)

	phase(onPhase, "session")
	var sessionID int
	if err := conn.QueryRowContext(ctx, "SELECT CONNECTION_ID()").Scan(&sessionID); err == nil && onSession != nil {
		onSession(sessionID)
	}

	phase(onPhase, "use")
	if target := s.resolveDatabase(database); target != "" {
		if _, err := conn.ExecContext(ctx, "USE "+QuoteIdentifier(target)); err != nil {
			// The selected database may have been dropped (possibly by the previous script).
			// Run without a default database; unqualified statements then fail on their own.
			if serverErrorNumber(err) != errUnknownDatabase {
				return fail(err, nil)
			}
			s.SetDatabase("")
			result.Messages = append(result.Messages, model.QueryMessage{
				Level:     "warning",
				Message:   fmt.Sprintf("Database '%s' no longer exists; running without a default database.", target),
				Timestamp: time.Now().Format(time.RFC3339),
			})
		} else {
			s.SetDatabase(target)
		}
	}

	statements := SplitStatements(statement)
	for index, item := range statements {
		if ctx.Err() != nil {
			result.Messages = append(result.Messages, sqlshared.CancelledMessage())
			result.ExecutionTime = sqlshared.ElapsedMillis(started)
			return result
		}
		phase(onPhase, fmt.Sprintf("batch%d", index+1))
		line := item.StartLine
		rows, err := conn.QueryContext(ctx, item.SQL)
		if err != nil {
			phase(onPhase, "failed")
			return fail(err, &line)
		}
		err = sqlshared.CollectRows(rows, maxRows, &result, normalizeValue)
		_ = rows.Close()
		if err != nil {
			phase(onPhase, "failed")
			return fail(err, &line)
		}
		phase(onPhase, fmt.Sprintf("batch%d_done", index+1))
		if used := DetectUseDatabase(item.SQL); used != "" {
			s.SetDatabase(used)
		}
	}

	phase(onPhase, "done")
	sqlshared.AppendCompletion(&result, len(statements), maxRows, started)
	return result
}

// rollbackOpenTransaction discards a transaction the script left open. It uses its own short
// deadline because the query context is usually cancelled or expired on the failure paths.
func rollbackOpenTransaction(conn *sql.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _ = conn.ExecContext(ctx, "ROLLBACK")
}

func errorMessage(ctx context.Context, err error, startLine *int) model.QueryMessage {
	number := serverErrorNumber(err)
	cancelled := errors.Is(err, context.Canceled) ||
		(number == errQueryInterrupted && errors.Is(ctx.Err(), context.Canceled))
	if cancelled {
		return sqlshared.CancelledMessage()
	}
	message := model.QueryMessage{
		Level:      "error",
		Message:    err.Error(),
		LineNumber: startLine,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
	if number != 0 {
		message.Code = &number
		// The driver prefixes "Error 1064 (42000): "; the code is reported separately.
		var serverErr *driver.MySQLError
		if errors.As(err, &serverErr) && strings.TrimSpace(serverErr.Message) != "" {
			message.Message = serverErr.Message
		}
	}
	return message
}

func serverErrorNumber(err error) int {
	var serverErr *driver.MySQLError
	if errors.As(err, &serverErr) {
		return int(serverErr.Number)
	}
	return 0
}

func phase(onPhase func(string), name string) {
	if onPhase != nil {
		onPhase(name)
	}
}
