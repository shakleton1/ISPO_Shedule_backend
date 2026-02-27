Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Set-Location (Resolve-Path "$PSScriptRoot\..")

docker compose down
