# Examples

Location: `examples/`

This directory contains the framework examples and shared static assets.

## Serve Examples

From the repo root:

```powershell
npm run dev:examples
```

Then open:

- `http://127.0.0.1:8090/examples`
- `http://127.0.0.1:8090/examples/01-counter/counter.html`

The `/examples` page is generated from the actual folder structure.

## Example List

Current example directories include:

- `01-counter`
- `02-text-input`
- `03-toggle`
- `04-form`
- `05-todo-basic`
- `06-todo-advanced`
- `07-goroutines`
- `08-fetch`
- `09-atoms`
- `10-advanced-form`
- `11-blog`
- `12-portfolio-site`
- `13-browser-compiler`
- `14-omi`
- `15-calculator`
- `16-devtools`
- `17-ssr-routing`
- `18-ssr-server-routing`

## Shared Static Assets

`examples/static/` contains assets shared across example pages, including:

- `script/wasm_exec.js`
- shared CSS
- shared images
- common HTML entrypoints

Generated wasm binaries for examples belong under `examples/static/bin/` and should not be committed unless there is a deliberate reason to do so.

## Running Example Browser Tests

The `examples/` directory has its own Playwright setup for example-oriented testing.

If you are working on the main framework regression suites, use `test/` instead. If you are specifically validating example pages, use the example-local Playwright config.

## Notes

- Older docs referenced ad hoc live reload commands as the primary example workflow. The primary documented path is now the Express server in `tools/dev-server/`.
- The browser-compiler example may generate a large local package archive tree under `examples/13-browser-compiler/static/pkg/`. That output is ignored and should stay out of git.
