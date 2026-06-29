# Browser Compiler Example

Location: `examples/public/browser-compiler/`

This example explores running a browser-hosted compilation workflow and associated UI around GoWebComponents.

## Current Status

- This remains an experimental example, not a normal baseline app path.
- It uses a browser-hosted compiler asset pipeline and a static `index.html` entrypoint rather than the simpler `main.go` pattern used by most examples.
- It may generate large local package archives under `static/pkg/`, and those generated artifacts should stay out of git.

## Status

- Experimental example
- Not part of the core runtime path
- May generate large local build artifacts under `static/pkg/`

## Serve It

Before serving the example, build the browser compiler assets once:

```powershell
Set-Location .\examples\public\browser-compiler
.\build-compiler.ps1
```

That script builds the browser-hosted compiler/linker binaries, copies `wasm_exec.js`,
copies the `js/wasm` standard-library archives into `static/pkg/js_wasm/`, and
generates `static/pkg/index.json`.

Use the repo dev server from the repo root:

```powershell
go run ./tools/gwc examples
```

Then open:

- `http://127.0.0.1:8090/examples/public/browser-compiler/`

## Local Generated Output

This example can generate content under:

- `examples/public/browser-compiler/static/pkg/`

That directory contains generated package archives and metadata. It is ignored and should not be committed.

## Notes

- Older docs referenced `go run ../../tools/serve.ps1`. That was incorrect and stale.
- This example is best treated as a contained experiment around browser-hosted compilation, not as the recommended path for module loading or code generation in normal apps. For the current importer workflow, see `go run ./tools/gwc import ...` in `tools/README.md`.
- If you are cleaning the repo, verify that generated package archives under `static/pkg/` are not added back to git.
