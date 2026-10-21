# PuffSQL

**A lightweight Microsoft SQL Server desktop client built with Go and Wails.**

This repository is the Go/Wails implementation of PuffSQL. It keeps the mature Vue 3 interface
from the original desktop client while replacing the Rust/Tauri host with a Go backend and Wails
v2. It supports SQL Server 2012 through current Azure SQL Database connections.

## Highlights

- Native, small desktop host powered by Go and the operating system WebView.
- Monaco SQL editor with completion, snippets, formatting and statement execution.
- Explorer metadata for databases, tables, views, columns, constraints, indexes and triggers.
- Streaming result sets with configurable row limits and Tabulator virtualization.
- Query cancellation through request context and SQL Server `KILL <spid>`.
- Transactional TSV import with row/column validation and progress events.
- OS keychain password storage; connection profiles never contain plaintext passwords.
- Query/message history, execution plans, statistics analysis, ER diagrams and SQL templates.
- Chinese and English UI, light/dark themes, and custom window chrome.

## Requirements

- Go 1.25 or newer
- Wails CLI 2.17 or newer
- Node.js 20 or newer
- Windows 10/11 WebView2, or the corresponding Wails dependencies on macOS/Linux

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0
wails doctor
```

## Development

Run the desktop application with live frontend reload:

```powershell
wails dev
```

Run the Vue application in a browser with the built-in mock backend:

```powershell
Set-Location frontend
npm install
npm run dev
```

The opt-in live smoke test uses the first saved profile and its OS keychain password:

```powershell
$env:PUFFSQL_LIVE = "1"
$env:PUFFSQL_TEST_PROFILE = "<connection id>"
go test ./internal/connection -run TestLiveSQLServerSmoke -count=1 -v
```

## Verification

```powershell
go test ./...
go vet ./...
wails generate module
Set-Location frontend
npm run typecheck
npm test
npm run build
```

Build the desktop executable:

```powershell
wails build
```

The Windows binary is written to `build/bin/PuffSQL.exe`.

Create a portable ZIP:

```powershell
powershell -ExecutionPolicy Bypass -File .\build-portable.ps1
```

Application data is stored in `%LOCALAPPDATA%\PuffSQL`. Existing `%LOCALAPPDATA%\SQLight`
profiles and legacy keychain entries are migrated automatically.
