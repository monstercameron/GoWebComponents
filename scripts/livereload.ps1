# Live reload script for Go WASM development
# This script watches for .go file changes and automatically rebuilds the WASM binary

Write-Host "Starting Go WASM Live Reload..." -ForegroundColor Green
Write-Host "Building from project root..." -ForegroundColor Blue

# Navigate to the livereload directory
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$LiveReloadPath = Join-Path $ScriptDir "livereload"
Set-Location $LiveReloadPath

# Run the live reloader
go run livereload.go 