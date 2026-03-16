@echo off
for /f "delims=" %%I in ('go env GOROOT') do set GOROOT_PATH=%%I
node "%GOROOT_PATH%\lib\wasm\wasm_exec_node.js" %*
