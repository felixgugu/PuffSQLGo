$ErrorActionPreference = 'Stop'

$projectRoot = Split-Path -Parent $PSScriptRoot
if (-not $env:PUFFSHELL_SCRIPT_DIR) {
    $env:PUFFSHELL_SCRIPT_DIR = $projectRoot
}

$goDirectory = 'C:\Program Files\Go\bin'
if (Test-Path -LiteralPath (Join-Path $goDirectory 'go.exe')) {
    $env:PATH = "$goDirectory;$env:PATH"
}

$wailsExecutable = $null
$wails = Get-Command 'wails.exe' -ErrorAction SilentlyContinue
if ($wails) {
    $wailsExecutable = $wails.Source
}
else {
    $fallback = Join-Path $env:USERPROFILE 'go\bin\wails.exe'
    if (Test-Path -LiteralPath $fallback) {
        $wailsExecutable = $fallback
    }
}
if (-not $wailsExecutable) {
    throw '找不到 Wails CLI。請先執行：go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0'
}

Set-Location -LiteralPath $projectRoot
& $wailsExecutable dev
exit $LASTEXITCODE
