// Package dbengine defines the contract every supported database engine implements. The
// connection manager and the Wails facade depend on this interface only, never on a concrete
// engine package.
package dbengine

import (
	"context"
	"errors"
	"strings"

	"puffsql/internal/model"
)

const (
	MSSQL = "mssql"
	MySQL = "mysql"
)

// ErrUnsupported is returned by an engine for a feature it does not implement.
var ErrUnsupported = errors.New("this feature is not supported for this database engine")

// Server is one open connection pool to a database server.
type Server interface {
	Ping(ctx context.Context) error
	Close() error
	// Database is the database the application last selected on this pool.
	Database() string

	// Execute runs a script. onSession reports the server-side session id of the pinned
	// connection so the query can be killed; onPhase receives diagnostic phase names.
	Execute(
		ctx context.Context,
		database *string,
		statement string,
		maxRows *int,
		timeoutSeconds *int,
		onSession func(int),
		onPhase func(string),
	) model.QueryResult
	// KillSession stops the statement running on another session of the same server.
	KillSession(ctx context.Context, sessionID int) error

	GetDatabases(ctx context.Context) ([]model.DatabaseItem, error)
	GetTables(ctx context.Context, database, schema *string) ([]model.TableItem, error)
	GetColumns(ctx context.Context, database *string, schema, table string) ([]model.ColumnItem, error)
	GetTableObjects(ctx context.Context, database *string, schema, table string) (model.TableObjects, error)
	GetDatabaseSchema(ctx context.Context, database *string) ([]model.TableSchema, error)
	GetForeignKeys(ctx context.Context, database *string, schema, table *string) ([]model.ForeignKeyItem, error)
	SwitchDatabase(ctx context.Context, database string) error

	PreviewColumnChange(ctx context.Context, req model.ColumnChangeRequest) (model.ColumnChangePreview, error)
	ApplyColumnChange(ctx context.Context, req model.ColumnChangeRequest, fingerprint string) (model.ColumnChangeResult, error)

	GetImportCapabilities(ctx context.Context, database *string, schema, table string) (model.ImportCapabilities, error)
	ImportRows(
		ctx context.Context,
		database *string,
		schema, table string,
		providedColumns []string,
		rows []model.ImportRowPayload,
		manualIdentity bool,
		progress func(processed, total int),
	) model.ImportResult

	ExportDatabaseTablesExcel(
		ctx context.Context,
		database string,
		refs []model.ExcelTableRef,
		path string,
		progress func(model.ExcelExportProgress),
	) (model.ExcelExportResult, error)
	ExportDatabaseSchemaExcel(
		ctx context.Context,
		database string,
		path string,
		progress func(model.ExcelExportProgress),
	) (model.ExcelExportResult, error)
}

// Normalize maps a stored engine name to a known engine. Profiles written before engines
// existed have an empty engine and are SQL Server.
func Normalize(engine string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "", MSSQL:
		return MSSQL, nil
	case MySQL:
		return MySQL, nil
	default:
		return "", errors.New("unsupported database engine: " + engine)
	}
}

// ApplyDefaults fills the port and initial database an engine expects when the profile leaves
// them empty. MySQL has no mandatory initial database.
func ApplyDefaults(profile *model.ConnectionProfile) {
	validPort := profile.Port > 0 && profile.Port <= 65535
	switch profile.Engine {
	case MySQL:
		if !validPort {
			profile.Port = 3306
		}
	default:
		if !validPort {
			profile.Port = 1433
		}
		if profile.Database == "" {
			profile.Database = "master"
		}
	}
}
