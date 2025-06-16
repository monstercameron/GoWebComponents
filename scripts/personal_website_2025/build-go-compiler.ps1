# PowerShell script to build Go compiler for Personal Website 2025
$ErrorActionPreference = "Stop"

Write-Host "🔨 Building Personal Website 2025 Go compiler..." -ForegroundColor Cyan

# Check if Go is installed
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "❌ Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}

# Create output directory
if (-not (Test-Path "static/bin")) {
    New-Item -ItemType Directory -Path "static/bin" -Force | Out-Null
}

# Build Go compiler to WASM
Write-Host "📦 Compiling Go compiler to WASM..." -ForegroundColor Yellow
$env:GOOS = "js"
$env:GOARCH = "wasm"

try {
    go build -ldflags="-s -w" -o "static/bin/personal-website-2025-go-compiler.wasm" "./cmd/wasm-compiler"
    
    # Check if build was successful
    if (Test-Path "static/bin/personal-website-2025-go-compiler.wasm") {
        $size = (Get-Item "static/bin/personal-website-2025-go-compiler.wasm").Length
        $sizeKB = [math]::Round($size / 1024, 2)
        Write-Host "✅ Compiler built successfully: static/bin/personal-website-2025-go-compiler.wasm ($sizeKB KB)" -ForegroundColor Green
    } else {
        Write-Host "❌ Build failed - output file not found" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "❌ Build failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
} 