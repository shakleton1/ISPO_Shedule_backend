Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Set-Location (Resolve-Path "$PSScriptRoot\..")

$goPath = (go env GOPATH)
$lint = Join-Path $goPath 'bin\golangci-lint.exe'

if (-not (Test-Path $lint)) {
  Write-Host "Installing golangci-lint..." -ForegroundColor Yellow
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2
}

Write-Host "Using golangci-lint: $lint" -ForegroundColor DarkGray
& $lint version

# Local lint can be RAM-hungry on Windows; keep it lightweight by default.
$env:GOMAXPROCS = '1'
$mode = $env:ISPO_LINT_MODE
if ([string]::IsNullOrWhiteSpace($mode)) { $mode = 'ci' }

$timeout = $env:ISPO_LINT_TIMEOUT
if ([string]::IsNullOrWhiteSpace($timeout)) { $timeout = '5m' }

if ($mode -eq 'ci') {
  Write-Host "Skipping local golangci-lint (CI-first mode)." -ForegroundColor Yellow
  Write-Host "Run lint on GitHub Actions by pushing a branch / opening a PR, or by manual workflow run." -ForegroundColor Yellow
  Write-Host "To lint locally anyway: set ISPO_LINT_MODE=fast (light) or ISPO_LINT_MODE=full (heavy)." -ForegroundColor DarkGray
  exit 0
}

if ($mode -eq 'full') {
  Write-Host "Running golangci-lint (full, may be heavy)..." -ForegroundColor Cyan
  & $lint run ./... --timeout=$timeout --modules-download-mode=readonly --print-resources-usage --verbose
} else {
  Write-Host "Running golangci-lint (fast, low-memory)..." -ForegroundColor Cyan
  & $lint run ./... --timeout=$timeout --modules-download-mode=readonly --print-resources-usage --tests=false --exclude-dirs '(^|\\)\\.venv($|\\)' --concurrency 1 --fast --verbose
}
