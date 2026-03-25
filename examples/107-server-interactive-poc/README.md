# 107 Server-Interactive POC

This example is a narrow proof-of-concept for server-interactive rendering:

- server owns state and HTML rendering
- browser sends small action payloads
- server streams updated snapshots over SSE

Run:

```powershell
go run ./examples/107-server-interactive-poc
```

Then open `http://127.0.0.1:8180`.
