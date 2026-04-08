# 107 Server-Interactive POC

This example is a narrow proof-of-concept for server-interactive rendering:

- server owns state and HTML rendering
- browser sends small action payloads
- server streams updated snapshots over SSE

Run:

```powershell
go run ./examples/server/server-interactive-proof-of-concept
```

Then open `http://127.0.0.1:8180`.
