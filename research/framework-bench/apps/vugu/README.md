# Bill Splitter — Vugu (Go → wasm)

Single-file `.vugu` component (HTML template + a Go `<script>` block), compiled to
wasm by Vugu's generator. Implements [../../SPEC.md](../../SPEC.md).

```bash
go install github.com/vugu/vugu/cmd/vgrun@latest
go mod tidy
vgrun -port 8101 .     # generates wasm + serves; reads root.vugu
```

Link the canonical `../../shared/styles.css` from the generated `index.html`.

**Status:** to-spec scaffold, not build-verified in this repo (Vugu toolchain + deps
are fetched on `go mod tidy`). Vugu has no separate shared-state primitive for a
single root, so `theme`/`roundUp` are root fields.
