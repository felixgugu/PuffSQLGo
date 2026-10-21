param(
    [string]$OutDir = (Join-Path $PSScriptRoot "dist-portable")
)

$ErrorActionPreference = "Stop"
$ProjectRoot = $PSScriptRoot
$BuildOutput = Join-Path $ProjectRoot "build\bin\PuffSQL.exe"
$PackageDir = Join-Path $OutDir "PuffSQL-Portable"
$ZipPath = Join-Path $OutDir "PuffSQL-Portable.zip"

Push-Location $ProjectRoot
try {
    # Do not clean build/bin: a developer may be running `wails dev`, which owns
    # PuffSQL-dev.exe. Wails overwrites only the requested production binary.
    & wails build -trimpath
    if ($LASTEXITCODE -ne 0) {
        throw "wails build failed with exit code $LASTEXITCODE"
    }
    if (-not (Test-Path -LiteralPath $BuildOutput)) {
        throw "Expected build output was not found: $BuildOutput"
    }

    New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
    if (Test-Path -LiteralPath $PackageDir) {
        Remove-Item -LiteralPath $PackageDir -Recurse -Force
    }
    New-Item -ItemType Directory -Force -Path $PackageDir | Out-Null
    Copy-Item -LiteralPath $BuildOutput -Destination (Join-Path $PackageDir "PuffSQL.exe")
    Copy-Item -LiteralPath (Join-Path $ProjectRoot "README.md") -Destination $PackageDir
    Copy-Item -LiteralPath (Join-Path $ProjectRoot "sql_custom_templates.json") -Destination $PackageDir

    if (Test-Path -LiteralPath $ZipPath) {
        Remove-Item -LiteralPath $ZipPath -Force
    }
    Compress-Archive -Path (Join-Path $PackageDir "*") -DestinationPath $ZipPath -CompressionLevel Optimal
    Write-Host "Portable package: $ZipPath"
}
finally {
    Pop-Location
}
