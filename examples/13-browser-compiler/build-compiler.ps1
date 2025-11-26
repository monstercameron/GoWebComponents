$env:GOOS = 'js'
$env:GOARCH = 'wasm'

$outputDir = Join-Path $PSScriptRoot "static\bin"
if (-not (Test-Path $outputDir)) {
    New-Item -ItemType Directory -Path $outputDir -Force | Out-Null
}

Write-Host "Building Go Compiler (cmd/compile)..."
go build -o (Join-Path $outputDir "compile.wasm") cmd/compile

Write-Host "Building Go Linker (cmd/link)..."
go build -o (Join-Path $outputDir "link.wasm") cmd/link

Write-Host "Copying wasm_exec.js..."
$goroot = go env GOROOT
# Try lib/wasm first (newer Go versions?)
$wasmExecPath = Join-Path $goroot "lib\wasm\wasm_exec.js"
if (-not (Test-Path $wasmExecPath)) {
    $wasmExecPath = Join-Path $goroot "misc\wasm\wasm_exec.js"
}
Copy-Item $wasmExecPath (Join-Path $PSScriptRoot "script\wasm_exec.js") -Force

Write-Host "Done."
