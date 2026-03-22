# Build and serve test app
Write-Host "Building test WASM app..." -ForegroundColor Green

$TestAppDir = Join-Path $PSScriptRoot "testapp"
$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$OutputDir = Join-Path $RepoRoot "bin\test\testapp"
$OutputWasm = Join-Path $OutputDir "main.wasm"

# Build WASM
$env:GOOS = "js"
$env:GOARCH = "wasm"

Push-Location $TestAppDir
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
go build -o $OutputWasm .
Pop-Location

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Build successful: $OutputWasm" -ForegroundColor Green
    Write-Host "`nStarting test server on http://localhost:8081" -ForegroundColor Cyan
    Write-Host "Press Ctrl+C to stop" -ForegroundColor Yellow
    
    Push-Location $TestAppDir
    python -m http.server 8081
    Pop-Location
} else {
    Write-Host "❌ Build failed" -ForegroundColor Red
    exit 1
}

# Cleanup env vars
Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
