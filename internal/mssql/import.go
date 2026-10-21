package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"puffsql/internal/model"
)

func (s *Server) GetImportCapabilities(
	ctx context.Context,
	database *string,
	schema, table string,
) (model.ImportCapabilities, error) {
	columns, err := s.GetColumns(ctx, database, schema, table)
	if err != nil {
		return model.ImportCapabilities{}, err
	}
	writable := writableColumns(columns)
	var identity *string
	for _, column := range writable {
		if column.IsIdentity {
			name := column.Name
			identity = &name
			break
		}
	}

	qualified := QualifiedTable(schema, table)
	statement := fmt.Sprintf(
		"SELECT CAST(ISNULL(HAS_PERMS_BY_NAME(%s, 'OBJECT', 'ALTER'), 0) AS INT), "+
			"CAST(ISNULL(CAST(SERVERPROPERTY('EngineEdition') AS INT), 0) AS INT)",
		sqlString(qualified),
	)
	var canAlter int
	var edition int
	if err := s.queryOne(ctx, database, statement, &canAlter, &edition); err != nil {
		return model.ImportCapabilities{}, err
	}

	capabilities := model.ImportCapabilities{
		IdentityColumn:         identity,
		WritableColumnCount:    len(writable),
		CanAlterTable:          canAlter != 0,
		EngineEdition:          edition,
		SupportsIdentityInsert: edition != 6,
	}
	switch {
	case identity == nil:
		capabilities.DisabledReason = optionalText("此資料表沒有識別欄位 (IDENTITY)")
	case edition == 6:
		capabilities.DisabledReason = optionalText("此資料庫引擎不支援 SET IDENTITY_INSERT")
	case canAlter == 0:
		capabilities.DisabledReason = optionalText(
			fmt.Sprintf("目前使用者沒有 %s 的 ALTER 權限，無法使用 SET IDENTITY_INSERT", qualified),
		)
	}
	return capabilities, nil
}

func (s *Server) ImportRows(
	ctx context.Context,
	database *string,
	schema, table string,
	providedColumns []string,
	rows []model.ImportRowPayload,
	manualIdentity bool,
	progress func(processed, total int),
) model.ImportResult {
	started := time.Now()
	result := model.ImportResult{Errors: []model.ImportRowError{}}
	columns, err := s.GetColumns(ctx, database, schema, table)
	if err != nil {
		result.Errors = append(result.Errors, rowError(0, nil, err.Error(), nil))
		result.RolledBack = true
		result.ExecutionTime = elapsedMillis(started)
		return result
	}
	expected := writableColumns(columns)
	if len(expected) == 0 {
		result.Errors = append(result.Errors, rowError(0, nil, "找不到可寫入的資料表欄位", nil))
		result.RolledBack = true
		result.ExecutionTime = elapsedMillis(started)
		return result
	}
	if !sameColumns(providedColumns, expected) {
		result.Errors = append(result.Errors, rowError(
			0, nil, "匯入欄位清單與資料表目前定義不符，請關閉後重新開啟匯入視窗", nil,
		))
		result.RolledBack = true
		result.ExecutionTime = elapsedMillis(started)
		return result
	}
	if len(rows) == 0 {
		result.ExecutionTime = elapsedMillis(started)
		return result
	}

	target := s.Database()
	if database != nil && strings.TrimSpace(*database) != "" {
		target = strings.TrimSpace(*database)
	}
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		result.Errors = append(result.Errors, rowError(0, nil, err.Error(), nil))
		result.RolledBack = true
		result.ExecutionTime = elapsedMillis(started)
		return result
	}
	defer conn.Close()
	if target != "" {
		if _, err := conn.ExecContext(ctx, "USE "+QuoteIdentifier(target)); err != nil {
			result.Errors = append(result.Errors, rowError(0, nil, err.Error(), nil))
			result.RolledBack = true
			result.ExecutionTime = elapsedMillis(started)
			return result
		}
		s.SetDatabase(target)
	}

	qualified := QualifiedTable(schema, table)
	identityInsert := manualIdentity && hasIdentity(expected)
	if manualIdentity && !identityInsert {
		result.Errors = append(result.Errors, rowError(
			0, nil, "此資料表沒有可手動寫入的 IDENTITY 欄位", nil,
		))
		result.RolledBack = true
		result.ExecutionTime = elapsedMillis(started)
		return result
	}
	if identityInsert {
		if _, err := conn.ExecContext(ctx, "SET IDENTITY_INSERT "+qualified+" ON"); err != nil {
			result.Errors = append(result.Errors, rowError(0, nil, err.Error(), nil))
			result.RolledBack = true
			result.ExecutionTime = elapsedMillis(started)
			return result
		}
		defer conn.ExecContext(context.Background(), "SET IDENTITY_INSERT "+qualified+" OFF")
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		result.Errors = append(result.Errors, rowError(0, nil, err.Error(), nil))
		result.RolledBack = true
		result.ExecutionTime = elapsedMillis(started)
		return result
	}
	statement, args := insertStatement(qualified, expected)
	stmt, err := tx.PrepareContext(ctx, statement)
	if err != nil {
		_ = tx.Rollback()
		result.Errors = append(result.Errors, rowError(0, nil, err.Error(), nil))
		result.RolledBack = true
		result.ExecutionTime = elapsedMillis(started)
		return result
	}
	defer stmt.Close()

	for index, row := range rows {
		if ctx.Err() != nil {
			_ = tx.Rollback()
			result.Errors = append(result.Errors, rowError(row.Line, nil, ctx.Err().Error(), nil))
			result.RolledBack = true
			result.ExecutionTime = elapsedMillis(started)
			return result
		}
		if len(row.Values) != len(expected) {
			_ = tx.Rollback()
			result.Errors = append(result.Errors, rowError(
				row.Line, nil,
				fmt.Sprintf("欄位數不符：預期 %d 欄，實際 %d 欄", len(expected), len(row.Values)), nil,
			))
			result.RolledBack = true
			result.ExecutionTime = elapsedMillis(started)
			return result
		}
		values, nullIndex, err := importArgs(row.Values)
		if err != nil {
			_ = tx.Rollback()
			result.Errors = append(result.Errors, rowError(row.Line, &expected[nullIndex].Name, err.Error(), nil))
			result.RolledBack = true
			result.ExecutionTime = elapsedMillis(started)
			return result
		}
		if _, err := stmt.ExecContext(ctx, values...); err != nil {
			_ = tx.Rollback()
			column := failingColumn(expected, err)
			result.Errors = append(result.Errors, rowError(row.Line, column, err.Error(), serverErrorCode(err)))
			result.RolledBack = true
			result.ExecutionTime = elapsedMillis(started)
			return result
		}
		if progress != nil {
			progress(index+1, len(rows))
		}
	}

	if err := tx.Commit(); err != nil {
		result.Errors = append(result.Errors, rowError(0, nil, err.Error(), serverErrorCode(err)))
		result.RolledBack = true
		result.ExecutionTime = elapsedMillis(started)
		return result
	}
	_ = args
	result.InsertedCount = uint64(len(rows))
	result.ExecutionTime = elapsedMillis(started)
	return result
}

func (s *Server) queryOne(ctx context.Context, database *string, statement string, dest ...any) error {
	rows, err := s.query(ctx, database, statement)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	return rows.Scan(dest...)
}

func writableColumns(columns []model.ColumnItem) []model.ColumnItem {
	result := make([]model.ColumnItem, 0, len(columns))
	for _, column := range columns {
		if !column.IsComputed && !column.IsRowVersion {
			result = append(result, column)
		}
	}
	return result
}

func sameColumns(provided []string, expected []model.ColumnItem) bool {
	if len(provided) != len(expected) {
		return false
	}
	for i := range expected {
		if !strings.EqualFold(strings.TrimSpace(provided[i]), expected[i].Name) {
			return false
		}
	}
	return true
}

func hasIdentity(columns []model.ColumnItem) bool {
	for _, column := range columns {
		if column.IsIdentity {
			return true
		}
	}
	return false
}

func insertStatement(qualified string, columns []model.ColumnItem) (string, []any) {
	names := make([]string, len(columns))
	parameters := make([]string, len(columns))
	args := make([]any, len(columns))
	for i, column := range columns {
		names[i] = QuoteIdentifier(column.Name)
		parameters[i] = fmt.Sprintf("@p%d", i+1)
		args[i] = any(nil)
	}
	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		qualified, strings.Join(names, ", "), strings.Join(parameters, ", "),
	), args
}

func importArgs(values []*string) ([]any, int, error) {
	args := make([]any, len(values))
	for i, value := range values {
		if value == nil {
			args[i] = nil
			continue
		}
		if strings.ContainsAny(*value, "\r\n") {
			return nil, i, fmt.Errorf("值不可包含換行字元")
		}
		args[i] = *value
	}
	return args, 0, nil
}

func failingColumn(columns []model.ColumnItem, err error) *string {
	message := strings.ToLower(err.Error())
	for _, column := range columns {
		if strings.Contains(message, strings.ToLower(column.Name)) {
			name := column.Name
			return &name
		}
	}
	return nil
}

func rowError(line int, column *string, message string, code *int) model.ImportRowError {
	return model.ImportRowError{Line: line, Column: column, Message: message, ServerCode: code}
}
