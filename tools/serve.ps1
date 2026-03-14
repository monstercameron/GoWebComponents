# Express-based dev server wrapper.
# Serves examples and static assets with stable MIME handling for WASM.

$ErrorActionPreference = "Stop"

$ServerDir = Join-Path $PSScriptRoot "dev-server"

if (-not (Test-Path $ServerDir)) {
    Write-Host "Error: dev server directory not found at $ServerDir" -ForegroundColor Red
    exit 1
}

$nodeCmd = Get-Command node -ErrorAction SilentlyContinue
$npmCmd = Get-Command npm -ErrorAction SilentlyContinue

if ($null -eq $nodeCmd -or $null -eq $npmCmd) {
    Write-Host "Error: Node.js + npm are required to run tools/dev-server." -ForegroundColor Red
    exit 1
}

Write-Host "Installing dev-server dependencies..." -ForegroundColor Yellow
Push-Location $ServerDir
npm install
if ($LASTEXITCODE -ne 0) {
    Pop-Location
    Write-Host "Error: npm install failed in $ServerDir" -ForegroundColor Red
    exit 1
}

Write-Host "Starting Express dev server on http://127.0.0.1:8090" -ForegroundColor Green
Write-Host "Examples index: http://127.0.0.1:8090/examples/static/index.html" -ForegroundColor Cyan
Write-Host "Counter page:   http://127.0.0.1:8090/examples/01-counter/counter.html" -ForegroundColor Cyan
Write-Host "Press Ctrl+C to stop" -ForegroundColor Yellow

npm start
$exitCode = $LASTEXITCODE
Pop-Location
exit $exitCode
