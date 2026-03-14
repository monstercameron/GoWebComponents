@echo off
"C:\Program Files\Go\bin\go.exe" env GOROOT >nul
node "C:\Program Files\Go\lib\wasm\wasm_exec_node.js" %*