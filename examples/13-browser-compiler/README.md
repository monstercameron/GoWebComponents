# Browser Compiler Example

Location: `examples/13-browser-compiler/`

This example explores running a browser-hosted compilation workflow and associated UI around GoWebComponents.

## Status

- Experimental example
- Not part of the core runtime path
- May generate large local build artifacts under `static/pkg/`

## Serve It

Use the repo dev server from the repo root:

```powershell
npm run dev:examples
```

Then open:

- `http://127.0.0.1:8090/examples/13-browser-compiler/`

## Local Generated Output

This example can generate content under:

- `examples/13-browser-compiler/static/pkg/`

That directory contains generated package archives and metadata. It is ignored and should not be committed.

## Notes

- Older docs referenced `go run ../../tools/serve.ps1`. That was incorrect and stale.
- If you are cleaning the repo, verify that generated package archives under `static/pkg/` are not added back to git.
