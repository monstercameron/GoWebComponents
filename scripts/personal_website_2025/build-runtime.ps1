# PowerShell script to build GoWebComponents runtime for Personal Website 2025
$ErrorActionPreference = "Stop"

Write-Host "🔨 Building Personal Website 2025 runtime..." -ForegroundColor Cyan

# Check if Go is installed
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "❌ Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}

# Create output directory
if (-not (Test-Path "static/bin")) {
    New-Item -ItemType Directory -Path "static/bin" -Force | Out-Null
}

# Build main GoWebComponents runtime
Write-Host "📦 Compiling main runtime to WASM..." -ForegroundColor Yellow
$env:GOOS = "js"
$env:GOARCH = "wasm"

try {
    go build -ldflags="-s -w" -o "static/bin/main.wasm" "./"
    
    # Check if build was successful
    if (Test-Path "static/bin/main.wasm") {
        $size = (Get-Item "static/bin/main.wasm").Length
        $sizeKB = [math]::Round($size / 1024, 2)
        Write-Host "✅ Runtime built successfully: static/bin/main.wasm ($sizeKB KB)" -ForegroundColor Green
    } else {
        Write-Host "❌ Runtime build failed - output file not found" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "❌ Runtime build failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
} 