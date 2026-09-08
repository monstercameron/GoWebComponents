# 107 Server-Interactive POC

This example is a narrow proof-of-concept for server-interactive rendering:

- the server owns state and renders the initial GWC view
- the Wasm client hydrates that view and sends small action payloads
- the server streams updated state snapshots over SSE for GWC to reconcile

Build the Wasm client from the repository root:

```powershell
$env:GOOS = "js"
$env:GOARCH = "wasm"
go build -o ./examples/server/server-interactive-proof-of-concept/client.wasm ./examples/server/server-interactive-proof-of-concept
Remove-Item Env:GOOS
Remove-Item Env:GOARCH
```

Then run the native server:

```powershell
go run ./examples/server/server-interactive-proof-of-concept
```

Then open `http://127.0.0.1:8180`.
