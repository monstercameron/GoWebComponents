# golden-linked-client

Golden output for the contributor-linked scaffold.

- Author: Golden Test
- Version: 1.0.0
- Preset: minimal-client

## Selected Capabilities

- `ui`
- `html`

The full generated capability contract lives in `FEATURE_MATRIX.md`.

## Starter Output Rules

- keep generated files small, readable, and conventionally organized
- treat this scaffold as disposable starter code; edit or replace it when the app shape changes
- keep app code on public framework packages instead of importing repo-internal build tooling
- avoid coupling to monorepo-only paths so this project can live independently

## Run

From the GoWebComponents repo root:

```powershell
go run ./tools/gwc dev -app "//generated//golden-linked-client//main.go" -root "//generated//golden-linked-client" -html "//generated//golden-linked-client//index.html" -wasm "main.wasm"
```

## Verify

From the generated project directory:

```powershell
go test ./...
```
