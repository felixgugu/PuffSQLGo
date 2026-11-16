package mysql

import (
	"context"
	"database/sql"

	"puffsql/internal/dbengine"
	"puffsql/internal/model"
	"puffsql/internal/sqlshared"
)

// The column editor and the TSV import are not implemented for MySQL yet. The UI hides them for
// MySQL connections; these methods are the backstop if they are called anyway.

func (s *Server) PreviewColumnChange(
	context.Context, model.ColumnChangeRequest,
) (model.ColumnChangePreview, error) {
	return model.ColumnChangePreview{}, dbengine.ErrUnsupported
}

func (s *Server) ApplyColumnChange(
	context.Context, model.ColumnChangeRequest, string,
) (model.ColumnChangeResult, error) {
	return model.ColumnChangeResult{}, dbengine.ErrUnsupported
}

func (s *Server) GetImportCapabilities(
	context.Context, *string, string, string,
) (model.ImportCapabilities, error) {
	reason := dbengine.ErrUnsupported.Error()
	return model.ImportCapabilities{DisabledReason: &reason}, nil
}

func (s *Server) ImportRows(
	_ context.Context,
	_ *string,
	_, _ string,
	_ []string,
	_ []model.ImportRowPayload,
	_ bool,
	_ func(processed, total int),
) model.ImportResult {
	return model.ImportResult{
		RolledBack: true,
		Errors:     []model.ImportRowError{{Message: dbengine.ErrUnsupported.Error()}},
	}
}

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

// OpenTableRows streams one table through a fully qualified name, so no USE is needed.
func (s *Server) OpenTableRows(
	ctx context.Context,
	database string,
	table model.TableItem,
) (*sql.Rows, func(), error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT * FROM "+QualifiedTable(database, table.Name))
	if err != nil {
		return nil, nil, err
	}
	return rows, func() {}, nil
}

func (s *Server) NormalizeValue(value any, typeName string) any {
	return normalizeValue(value, typeName)
}

func (s *Server) TableLabel(table model.TableItem) string {
	return table.Name
}
