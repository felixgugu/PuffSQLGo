package mssql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"puffsql/internal/model"
)

func (s *Server) Execute(
	ctx context.Context,
	database *string,
	statement string,
	maxRows *int,
	timeoutSeconds *int,
	onSPID func(int),
	onPhase func(string),
) model.QueryResult {
	started := time.Now()
	result := model.QueryResult{
		ResultSets: []model.ResultSet{},
		Messages:   []model.QueryMessage{},
	}
	if timeoutSeconds != nil && *timeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*timeoutSeconds)*time.Second)
		defer cancel()
	}

	phase(onPhase, "open_conn")
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		result.Messages = append(result.Messages, errorMessage(err, nil))
		result.ExecutionTime = elapsedMillis(started)
		return result
	}
	defer conn.Close()

	phase(onPhase, "spid")
	var spid int
	if err := conn.QueryRowContext(ctx, "SELECT @@SPID").Scan(&spid); err == nil && onSPID != nil {
		onSPID(spid)
	}

	batches := SplitSQLBatches(statement)
	firstDrop := ""
	for _, batch := range batches {
		if name, ok := DetectDropDatabase(batch.SQL); ok {
			firstDrop = name
			break
		}
	}

	target := s.Database()
	if database != nil && strings.TrimSpace(*database) != "" {
		target = strings.TrimSpace(*database)
	}
	phase(onPhase, "use")
	if target != "" {
		if _, err := conn.ExecContext(ctx, "USE "+QuoteIdentifier(target)); err != nil {
			// A script that drops its own selected database must still run when that database is
			// already gone (dropped externally or by an earlier session).
			if firstDrop == "" || !strings.EqualFold(target, firstDrop) {
				result.Messages = append(result.Messages, errorMessage(err, nil))
				result.ExecutionTime = elapsedMillis(started)
				return result
			}
			if _, fallbackErr := conn.ExecContext(ctx, "USE master"); fallbackErr != nil {
				result.Messages = append(result.Messages, errorMessage(err, nil))
				result.ExecutionTime = elapsedMillis(started)
				return result
			}
			s.SetDatabase("master")
		} else {
			s.SetDatabase(target)
		}
	}

	executedBatches := 0
	idleReleased := false
	defer func() {
		if idleReleased {
			s.resumeIdleConnections()
		}
	}()
	for batchIndex := 0; batchIndex < len(batches); batchIndex++ {
		batch := batches[batchIndex]
		for repeat := 0; repeat < batch.RepeatCount; repeat++ {
			if ctx.Err() != nil {
				result.Messages = append(result.Messages, model.QueryMessage{
					Level:     "warning",
					Message:   "Query was cancelled by user",
					Timestamp: time.Now().Format(time.RFC3339),
				})
				result.ExecutionTime = elapsedMillis(started)
				return result
			}

			dropTarget, isDrop := DetectDropDatabase(batch.SQL)
			execCtx := ctx
			var cancelDrop context.CancelFunc
			if isDrop {
				// A DROP DATABASE must not run from inside the database it removes, otherwise SQL
				// Server reports error 3702 because this session is still using it.
				phase(onPhase, "drop_database")
				if _, err := conn.ExecContext(ctx, "USE master"); err != nil {
					result.Messages = append(result.Messages, errorMessage(err, &batch.StartLine))
					result.ExecutionTime = elapsedMillis(started)
					return result
				}
				s.SetDatabase("master")
				if !idleReleased {
					// The application pool may still hold idle sessions inside the target
					// database. Release them before waiting for the exclusive drop lock and keep
					// them released until the drop finishes, so a metadata query that ends in the
					// meantime cannot park another session in the target database.
					s.pauseIdleConnections()
					idleReleased = true
				}
				execCtx, cancelDrop = context.WithTimeout(ctx, dropDatabaseTimeout)
			}

			phase(onPhase, fmt.Sprintf("batch%d", batchIndex+1))
			rows, err := conn.QueryContext(execCtx, batch.SQL)
			if err != nil {
				if cancelDrop != nil {
					cancelDrop()
				}
				if isDrop && errors.Is(err, context.DeadlineExceeded) {
					err = fmt.Errorf(
						"DROP DATABASE [%s] did not finish before the timeout; another session may still be using the database: %w",
						dropTarget, err,
					)
				}
				phase(onPhase, "failed")
				result.Messages = append(result.Messages, errorMessage(err, &batch.StartLine))
				result.ExecutionTime = elapsedMillis(started)
				return result
			}
			if err := collectRows(rows, maxRows, &result); err != nil {
				_ = rows.Close()
				if cancelDrop != nil {
					cancelDrop()
				}
				phase(onPhase, "failed")
				result.Messages = append(result.Messages, errorMessage(err, &batch.StartLine))
				result.ExecutionTime = elapsedMillis(started)
				return result
			}
			_ = rows.Close()
			if cancelDrop != nil {
				cancelDrop()
			}
			executedBatches++
			phase(onPhase, fmt.Sprintf("batch%d_done", batchIndex+1))
			if isDrop {
				s.SetDatabase("master")
				phase(onPhase, "drop_database_done")
			}
			if used := DetectUseDatabase(batch.SQL); used != "" {
				// Never stay pointed at a database this batch just dropped.
				if isDrop && strings.EqualFold(used, dropTarget) {
					s.SetDatabase("master")
				} else {
					s.SetDatabase(used)
				}
			}
		}
	}

	phase(onPhase, "done")
	var rows uint64
	hasTruncated := false
	for _, set := range result.ResultSets {
		rows += uint64(set.RowCount)
		hasTruncated = hasTruncated || set.IsTruncated
	}
	result.AffectedRows = rows
	if hasTruncated && maxRows != nil {
		result.Messages = append(result.Messages, model.QueryMessage{
			Level: "warning",
			Message: fmt.Sprintf(
				"查詢結果已達最大限制 %d 筆，其餘資料已自動截斷以保護系統效能。", *maxRows,
			),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}
	if executedBatches > 1 {
		result.Messages = append(result.Messages, model.QueryMessage{
			Level: "info",
			Message: fmt.Sprintf(
				"Query completed successfully. Executed %d batch(es), %d result set(s), %d rows returned.",
				executedBatches, len(result.ResultSets), rows,
			),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	} else {
		result.Messages = append(result.Messages, model.QueryMessage{
			Level: "info",
			Message: fmt.Sprintf(
				"Query completed successfully. %d result set(s), %d rows returned.",
				len(result.ResultSets), rows,
			),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}
	result.ExecutionTime = elapsedMillis(started)
	return result
}

func collectRows(rows *sql.Rows, maxRows *int, result *model.QueryResult) error {
	for {
		columns, err := rows.Columns()
		if err != nil {
			return err
		}
		if len(columns) > 0 {
			set, err := readResultSet(rows, columns, maxRows)
			if err != nil {
				return err
			}
			result.ResultSets = append(result.ResultSets, set)
		} else {
			for rows.Next() {
				// Drain non-tabular result sets so the driver can reach the next response.
			}
			if err := rows.Err(); err != nil {
				return err
			}
		}
		if !rows.NextResultSet() {
			break
		}
	}
	return rows.Err()
}

func readResultSet(rows *sql.Rows, names []string, maxRows *int) (model.ResultSet, error) {
	types, err := rows.ColumnTypes()
	if err != nil {
		return model.ResultSet{}, err
	}
	limit := math.MaxInt
	if maxRows != nil && *maxRows >= 0 {
		limit = *maxRows
	}
	set := model.ResultSet{
		Columns: make([]model.ColumnDef, len(names)),
		Rows:    [][]any{},
	}
	for i, name := range names {
		dataType := "unknown"
		if i < len(types) {
			dataType = types[i].DatabaseTypeName()
		}
		set.Columns[i] = model.ColumnDef{Name: name, DataType: dataType, Nullable: true, Ordinal: i}
	}

	scans := make([]any, len(names))
	values := make([]any, len(names))
	for i := range scans {
		scans[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(scans...); err != nil {
			return model.ResultSet{}, err
		}
		set.TotalCount++
		if len(set.Rows) >= limit {
			set.IsTruncated = true
			continue
		}
		row := make([]any, len(values))
		for i, value := range values {
			typeName := ""
			if i < len(types) {
				typeName = types[i].DatabaseTypeName()
			}
			row[i] = normalizeValue(value, typeName)
		}
		set.Rows = append(set.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return model.ResultSet{}, err
	}
	set.RowCount = len(set.Rows)
	return set, nil
}

func normalizeValue(value any, dataType string) any {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case time.Time:
		return formatTimeValue(typed, dataType)
	case []byte:
		switch strings.ToUpper(dataType) {
		case "UNIQUEIDENTIFIER":
			if guid, ok := formatUniqueIdentifier(typed); ok {
				return guid
			}
			return model.BinaryCell{Type: "binary", Length: len(typed)}
		case "BINARY", "VARBINARY", "IMAGE", "TIMESTAMP", "ROWVERSION":
			return model.BinaryCell{Type: "binary", Length: len(typed)}
		default:
			return string(typed)
		}
	case int64:
		return typed
	case int32:
		return typed
	case int16:
		return typed
	case uint64:
		if typed > math.MaxInt64 {
			return strconv.FormatUint(typed, 10)
		}
		return int64(typed)
	case float64:
		return typed
	case float32:
		return float64(typed)
	default:
		return typed
	}
}

// formatTimeValue renders a scanned SQL Server date/time value as text for the grid.
// datetimeoffset keeps its UTC offset (for example "2024-05-06 07:08:09 +08:00"); datetime,
// datetime2, smalldatetime, date and time carry no zone information, so no offset is appended.
// The "-07:00" token is a Go layout placeholder, not a literal: it prints "+08:00", "+00:00",
// "-05:00", ... depending on the zone attached to the scanned time.
func formatTimeValue(value time.Time, dataType string) string {
	const layout = "2006-01-02 15:04:05.999999999"
	if isDateTimeOffset(dataType) {
		return value.Format(layout + " -07:00")
	}
	return value.Format(layout)
}

// isDateTimeOffset accepts both the bare driver type name ("DATETIMEOFFSET") and the declared
// form that includes the fractional-seconds scale ("datetimeoffset(7)").
func isDateTimeOffset(dataType string) bool {
	kind := strings.ToUpper(strings.TrimSpace(dataType))
	return kind == "DATETIMEOFFSET" || strings.HasPrefix(kind, "DATETIMEOFFSET(")
}

// SQL Server sends uniqueidentifier values in its mixed-endian storage order. Convert the first
// three groups before formatting so the grid shows the same GUID as SQL Server Management Studio.
func formatUniqueIdentifier(value []byte) (string, bool) {
	if len(value) != 16 {
		return "", false
	}

	var guid [16]byte
	copy(guid[:], value)
	for _, bounds := range [][2]int{{0, 4}, {4, 6}, {6, 8}} {
		for left, right := bounds[0], bounds[1]-1; left < right; left, right = left+1, right-1 {
			guid[left], guid[right] = guid[right], guid[left]
		}
	}

	return fmt.Sprintf(
		"%X-%X-%X-%X-%X",
		guid[0:4], guid[4:6], guid[6:8], guid[8:10], guid[10:],
	), true
}

func errorMessage(err error, startLine *int) model.QueryMessage {
	code := serverErrorCode(err)
	var line *int
	if startLine != nil {
		value := *startLine
		type lineNumbered interface{ SQLErrorLineNo() int32 }
		var lineErr lineNumbered
		if errors.As(err, &lineErr) && lineErr.SQLErrorLineNo() > 0 {
			value += int(lineErr.SQLErrorLineNo()) - 1
		}
		line = &value
	}
	level := "error"
	message := err.Error()
	if errors.Is(err, context.Canceled) {
		level = "warning"
		message = "Query was cancelled by user"
		line = nil
	}
	return model.QueryMessage{
		Level:      level,
		Message:    message,
		Code:       code,
		LineNumber: line,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
}

func serverErrorCode(err error) *int {
	type numbered interface{ SQLErrorNumber() int32 }
	var numberedErr numbered
	if errors.As(err, &numberedErr) {
		value := int(numberedErr.SQLErrorNumber())
		return &value
	}
	return nil
}

func elapsedMillis(start time.Time) uint64 {
	elapsed := time.Since(start)
	if elapsed < 0 {
		return 0
	}
	return uint64(elapsed.Milliseconds())
}

func phase(onPhase func(string), name string) {
	if onPhase != nil {
		onPhase(name)
	}
}
