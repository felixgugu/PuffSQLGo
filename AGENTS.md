# PuffSQL Go/Wails Architecture

## Product

PuffSQL is a lightweight desktop client for Microsoft SQL Server (T-SQL) and MySQL. The shipping
application is a Go 1.25+ backend hosted by Wails v2 with a Vue 3/TypeScript frontend.

## Layout

```text
.
├── main.go, app*.go              Wails entry point and Go command facade
├── internal/
│   ├── applog/                   redacted query and AI diagnostics logs
│   ├── connection/               profile lifecycle, keychain and query registry
│   ├── dbengine/                 Server interface every database engine implements
│   ├── mssql/                    SQL Server: TDS client, GO batches, metadata, transactional import
│   ├── mysql/                    MySQL: statement splitting, information_schema metadata
│   ├── sqlshared/                engine-neutral result streaming and Excel writers
│   ├── model/                    JSON contracts shared with Vue
│   └── storage/                  application data and atomic profile persistence
├── frontend/
│   ├── src/                      Vue 3 UI, Pinia stores and service boundary
│   │   └── dialects/             per-engine SQL dialect (quoting, literals, capabilities)
│   ├── tests/                    frontend contract/unit tests
│   └── wailsjs/                  generated Wails bindings (do not hand-edit)
├── build/                        Wails icon and build assets
└── wails.json                    Wails build configuration
```

## Data Flow

1. Vue calls `services/api.ts`; that module alone translates command-shaped calls to typed Wails
   bindings in `frontend/wailsjs/go/main/App`.
2. The Go `App` facade validates arguments and delegates to `internal/connection` or file/log
   services.
3. `internal/connection` opens the engine named by the profile (`mssql` or `mysql`) and only talks
   to it through `dbengine.Server`. SQL Server traffic goes through
   `github.com/microsoft/go-mssqldb` with `GO` split in Go; MySQL goes through
   `github.com/go-sql-driver/mysql` with statements split on `;` / `DELIMITER`. Result sets stay
   streamed through `database/sql`, and cancellation calls the active query context plus `KILL`
   (`KILL QUERY` for MySQL).
4. Passwords live only in the OS keychain. `connections.json` contains endpoint metadata but never
   credentials.

## Engineering Rules

- Keep frontend components independent of generated bindings; route desktop calls through
  `src/services/`.
- Keep SQL protocol and metadata logic in the engine packages (`internal/mssql`,
  `internal/mysql`); Wails methods remain thin and never import an engine package.
- Frontend SQL text goes through `src/dialects`; never hard-code `[name]` / `N'..'` quoting.
  There is no global "current dialect": pass the dialect of the connection the SQL targets
  (`connectionStore.getDialectFor(id)`, `workspaceStore.getTabDialect(tabId)`). Helpers in
  `utils/` default to SQL Server only so T-SQL-only generators and tests stay unchanged.
  Features an engine lacks are hidden through the dialect's `capabilities`.
- MySQL has no schema level: the backend reports the database name as the schema.
- Prefer parameterized SQL. Dynamic identifiers and catalog literals must be escaped.
- TSV imports run in one transaction. Any failed row rolls the whole import back.
- Never log passwords, API keys, or query result rows.
- Keep files near 400 lines and split by responsibility when practical.
- Regenerate bindings with `wails generate module` after changing public Go method signatures.

## Verification

```powershell
go test ./...
go vet ./...
wails generate module
Set-Location frontend
npm run typecheck
npm test
npm run build
Set-Location ..
wails build
```

The grid stack is Tabulator 6; Monaco remains the SQL editor.
