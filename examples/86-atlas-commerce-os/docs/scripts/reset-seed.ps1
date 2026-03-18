param(
    [string]$DbPath = "..\data\atlas-commerce-os.db"
)

Write-Host "Atlas Commerce OS seed reset scaffold" -ForegroundColor Cyan

if (Test-Path $DbPath) {
    Remove-Item $DbPath -Force
    Write-Host "Removed existing database: $DbPath" -ForegroundColor Green
} else {
    Write-Host "No database file found at $DbPath" -ForegroundColor Yellow
}

Write-Host "Next step: rebuild the local seed database once persistence is implemented." -ForegroundColor Gray