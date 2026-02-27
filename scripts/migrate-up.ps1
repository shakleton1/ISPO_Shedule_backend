Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Set-Location (Resolve-Path "$PSScriptRoot\..")

$goPath = (go env GOPATH)
$goose = Join-Path $goPath 'bin\goose.exe'

if (-not (Test-Path $goose)) {
  Write-Host "Installing goose..." -ForegroundColor Yellow
  go install github.com/pressly/goose/v3/cmd/goose@latest
}

$env:GOOSE_DRIVER = "postgres"
if (-not $env:GOOSE_DBSTRING) {
  $env:GOOSE_DBSTRING = "host=localhost port=5432 user=postgres password=postgres dbname=ispo_schedule sslmode=disable"
}

& $goose -dir .\db\migrations up
