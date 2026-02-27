Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Set-Location (Resolve-Path "$PSScriptRoot\..")

go test ./...
