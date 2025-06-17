# Live reload server for Go WASM development
Write-Host "Starting Go WASM Live Reload Server..." -ForegroundColor Green
Write-Host "Building from project root..." -ForegroundColor Yellow
Write-Host "Server will be available at http://localhost:8080" -ForegroundColor Cyan

# Store original directory to restore on exit
$OriginalDir = Get-Location
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# Add trap to restore directory on Ctrl+C or script exit
trap {
    Write-Host "`nRestoring original directory..." -ForegroundColor Yellow
    if ($OriginalDir) {
        Set-Location $OriginalDir
    }
    exit
}

# Navigate to livereload directory
$LiveReloadDir = Join-Path $ScriptDir "livereload"
Set-Location $LiveReloadDir

$BinaryPath = ".\livereload.exe"

Write-Host "Building live reload server..." -ForegroundColor Yellow
go build -o livereload.exe .
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!" -ForegroundColor Red
    if ($OriginalDir) {
        Set-Location $OriginalDir
    }
    exit 1
}
Write-Host "Build completed!" -ForegroundColor Green

Write-Host "Starting live reload server..." -ForegroundColor Green
try {
    & $BinaryPath
}
finally {
    Write-Host "Restoring original directory..." -ForegroundColor Yellow
    if ($OriginalDir) {
        Set-Location $OriginalDir
    }
} 