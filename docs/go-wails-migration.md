# Rust/Tauri to Go/Wails Migration

## Completed

- Wails v2 application host, Windows frameless chrome and non-Windows native decorations.
- Typed Wails command facade for connection, query, schema, import, template, SQL-folder and log
  operations.
- SQL Server connectivity through `github.com/microsoft/go-mssqldb`.
- GO batch splitting, multi-result-set conversion, row limits and query-context cancellation.
- KILL-based cancellation through a separate SQL Server connection.
- OS keychain credential storage with legacy SQLight service migration.
- Atomic connection profile persistence with legacy `%LOCALAPPDATA%\SQLight` migration.
- Database, table, column, foreign-key and full table-object metadata.
- Transactional TSV import with progress events.
- Query/AI logs and custom template management.
- Vue service bridge changed from Tauri `invoke`/events to generated Wails bindings/events.

## Verification

The following gates currently pass:

```powershell
go test ./...
go vet ./...
wails generate module
Set-Location frontend
npm run typecheck
npm test
npm run build
Set-Location ..
wails build -trimpath
powershell -ExecutionPolicy Bypass -File .\build-portable.ps1
```

The Wails production executable was launched successfully. The UI, custom window controls,
settings dialog, About identity, Wails-to-Go IPC error propagation and clean shutdown were
verified interactively.

`npm audit --omit=dev` reports zero production vulnerabilities. Tailwind CSS 3's development-time
glob/watch dependency still carries npm advisories whose offered fix is a Tailwind 4 migration;
those packages are not included in the Wails application bundle.

Live verification against the local test SQL Server now passes through both the Go connection
manager and the exported Wails `App` facade:

- SQL authentication as `sa` succeeds.
- `TestDB1` reports 8 visible databases and 15 tables.
- A two-batch `GO` script returns two result sets.
- Columns, full table objects and schema metadata load successfully.
- A `WAITFOR DELAY` query is cancelled through its request context; the cancelled TDS session is
  evicted from the pool before `KILL`, and the next command reconnects lazily.

## Live Smoke Test

Run the opted-in live checks against a configured test profile:

```powershell
$env:PUFFSQL_LIVE = "1"
$env:PUFFSQL_TEST_PROFILE = "<connection id>"
go test ./internal/connection -run TestLiveSQLServerSmoke -count=1 -v
go test . -run TestLiveAppFacade -count=1 -v
```

The tests perform read-only probes, multi-result-set execution, cancellation, database discovery
and table-object metadata checks.
