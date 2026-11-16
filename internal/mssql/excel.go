package mssql

import (
	"context"
	"database/sql"

	"puffsql/internal/model"
	"puffsql/internal/sqlshared"
)

// ExportDatabaseSchemaExcel writes one schema-only worksheet per base table.
func (s *Server) ExportDatabaseSchemaExcel(
	ctx context.Context,
	database string,
	path string,
	progress func(model.ExcelExportProgress),
) (model.ExcelExportResult, error) {
	return sqlshared.ExportDatabaseSchemaExcel(ctx, s, database, path, progress)
}

// ExportDatabaseTablesExcel writes row data for explicitly selected base tables only.
func (s *Server) ExportDatabaseTablesExcel(
	ctx context.Context,
	database string,
	refs []model.ExcelTableRef,
	path string,
	progress func(model.ExcelExportProgress),
) (model.ExcelExportResult, error) {
	return sqlshared.ExportDatabaseTablesExcel(ctx, s, database, refs, path, progress)
}

// OpenTableRows streams one table on a pinned connection so the USE applies to the SELECT.
func (s *Server) OpenTableRows(
	ctx context.Context,
	database string,
	table model.TableItem,
) (*sql.Rows, func(), error) {
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return nil, nil, err
	}
	release := func() { _ = conn.Close() }
	if _, err := conn.ExecContext(ctx, "USE "+QuoteIdentifier(database)); err != nil {
		release()
		return nil, nil, err
	}
	rows, err := conn.QueryContext(ctx, "SELECT * FROM "+QualifiedTable(table.Schema, table.Name))
	if err != nil {
		release()
		return nil, nil, err
	}
	return rows, release, nil
}

func (s *Server) NormalizeValue(value any, typeName string) any {
	return normalizeValue(value, typeName)
}

func (s *Server) TableLabel(table model.TableItem) string {
	return table.Schema + "." + table.Name
}
