# Bill Splitter — HTMX (Go backend)

HTMX is **server-driven hypermedia**, not a client reactive framework, so the
honest implementation is a small server that holds state and returns re-rendered
HTML fragments. Here the backend is Go (`net/http`), which keeps it in this repo's
toolchain and lets it compile-verify.

## Run (from this directory)

```powershell
go run .
# → http://127.0.0.1:8099
```

The server reads the canonical `../../shared/styles.css` and serves it at `/styles.css`.

## How it maps to SPEC

| SPEC concept | HTMX realization |
|---|---|
| local + shared state | **all server-side** (`state` struct); no client framework |
| an interaction | `hx-post` to an endpoint that mutates state and returns the morphed `<main>` |
| reactivity | `hx-swap="morph:outerHTML"` via the idiomorph extension (preserves input focus) |
| shared toggles | `theme` / `roundUp` are server state flipped by `/toggle/*` endpoints |

## Notes / comparison

- This is the **paradigm outlier**: no client state, no VDOM/signals. The contrast
  with the wasm and client-JS apps is the point — htmx trades a client runtime for a
  network round-trip per interaction.
- State is a single in-memory value (one user) to keep the demo small; a real app
  would scope it per session/cookie.
- Compile-verified (`go build`) in this repo.
