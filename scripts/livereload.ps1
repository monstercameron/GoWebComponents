# Live reload server for Go WASM development
# This script starts an HTTP server with WebSocket live reload functionality
# Features:
# - Serves static files and index.html with injected live reload script
# - Watches for .go file changes and automatically rebuilds the WASM binary
# - WebSocket communication for real-time build status and hot reload
# - State preservation during hot reloads

Write-Host "🚀 Starting Go WASM Live Reload Server..." -ForegroundColor Green
Write-Host "📦 Building from project root..." -ForegroundColor Yellow
Write-Host "🌐 Server will be available at http://localhost:8080" -ForegroundColor Cyan

# Get the directory of this script
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location "$ScriptDir\livereload"

# Check if binary exists, build if not
$BinaryPath = ".\livereload.exe"
if (-not (Test-Path $BinaryPath)) {
    Write-Host "🔨 Building live reload server..." -ForegroundColor Yellow
    go build -o livereload.exe .
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Build failed!" -ForegroundColor Red
        exit 1
    }
    Write-Host "✅ Build completed!" -ForegroundColor Green
}

# Run the live reload server
Write-Host "🚀 Starting live reload server..." -ForegroundColor Green
& $BinaryPath 