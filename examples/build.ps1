# Build script for GoWebComponents examples
# Builds WASM binaries for blog and counter examples

Write-Host "🚀 Building GoWebComponents WASM Examples..." -ForegroundColor Cyan

# Set WASM build environment
$env:GOOS = 'js'
$env:GOARCH = 'wasm'

# Build Blog Landing Page
Write-Host "`n📝 Building Blog Landing Page..." -ForegroundColor Yellow
Set-Location "$PSScriptRoot\blog"
go build -o ..\static\bin\blog.wasm main.go blog_landing_page.go
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Blog Landing Page built successfully -> static/bin/blog.wasm" -ForegroundColor Green
} else {
    Write-Host "❌ Blog Landing Page build failed" -ForegroundColor Red
    exit 1
}

# Build Click Counter
Write-Host "`n🖱️ Building Click Counter..." -ForegroundColor Yellow
Set-Location "$PSScriptRoot\counter"
go build -o ..\static\bin\counter.wasm main.go click_counter.go
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Click Counter built successfully -> static/bin/counter.wasm" -ForegroundColor Green
} else {
    Write-Host "❌ Click Counter build failed" -ForegroundColor Red
    exit 1
}

# Return to examples directory
Set-Location $PSScriptRoot

# Display file sizes
Write-Host "`n📊 Build Summary:" -ForegroundColor Cyan
Get-Item static\bin\blog.wasm, static\bin\counter.wasm | Format-Table Name, @{Label="Size (KB)"; Expression={[math]::Round($_.Length/1KB, 2)}} -AutoSize

Write-Host "`n🎉 All examples built successfully!" -ForegroundColor Green
