# Go-based example server wrapper.
# Serves examples and static assets with stable MIME handling for WASM.

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot

Write-Host "Starting Go example server on http://127.0.0.1:8090" -ForegroundColor Green
Write-Host "Examples index: http://127.0.0.1:8090/examples/" -ForegroundColor Cyan
Write-Host "Counter page:   http://127.0.0.1:8090/examples/01-counter/" -ForegroundColor Cyan
Write-Host "Press Ctrl+C to stop" -ForegroundColor Yellow

Push-Location $RepoRoot
go run ./tools/gwc examples -host 127.0.0.1 -port 8090
$exitCode = $LASTEXITCODE
Pop-Location
exit $exitCode
