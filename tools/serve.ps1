# Simple HTTP server for testing
# Serves the static directory on port 8080

Write-Host "Starting HTTP server on http://localhost:8080" -ForegroundColor Green
Write-Host "Serving from: examples/static" -ForegroundColor Yellow
Write-Host "Press Ctrl+C to stop" -ForegroundColor Cyan

$StaticPath = Join-Path $PSScriptRoot "..\examples\static"

if (-not (Test-Path $StaticPath)) {
    Write-Host "Error: Static directory not found at $StaticPath" -ForegroundColor Red
    exit 1
}

# Check if Python is available
$pythonCmd = Get-Command python -ErrorAction SilentlyContinue
if ($null -eq $pythonCmd) {
    $pythonCmd = Get-Command python3 -ErrorAction SilentlyContinue
}

if ($null -ne $pythonCmd) {
    Push-Location $StaticPath
    & $pythonCmd.Source -m http.server 8080 --bind 127.0.0.1
    Pop-Location
} else {
    Write-Host "Python not found. Install Python or use an alternative HTTP server." -ForegroundColor Red
    Write-Host "Alternative: Install http-server via npm: npm install -g http-server" -ForegroundColor Yellow
    exit 1
}
