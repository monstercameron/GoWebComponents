# Set environment variables for WebAssembly build
$env:GOOS = "js"
$env:GOARCH = "wasm"

# Build the WebAssembly binary
go build -o static/bin/main.wasm

# Reset environment variables to default (optional)
Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue

Write-Host "✅ WebAssembly build completed: static/bin/main.wasm" -ForegroundColor Green 