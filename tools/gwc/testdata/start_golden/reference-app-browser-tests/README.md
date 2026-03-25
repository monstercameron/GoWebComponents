# golden-reference-app

Golden output for the reference app scaffold.

- Author: Golden Test
- Version: 1.0.0
- Preset: reference-app

## Selected Capabilities

- `router`
- `forms`
- `fetch`
- `state`
- `browser-tests`

The full generated capability contract lives in `FEATURE_MATRIX.md`.

## Starter Output Rules

- keep generated files small, readable, and conventionally organized
- treat this scaffold as disposable starter code; edit or replace it when the app shape changes
- keep app code on public framework packages instead of importing repo-internal build tooling
- avoid coupling to monorepo-only paths so this project can live independently

## Run

From the GoWebComponents repo root:

```powershell
go run ./tools/gwc dev -app "//generated//golden-reference-app//main.go" -root "//generated//golden-reference-app" -html "//generated//golden-reference-app//index.html" -wasm "bin/main.wasm"
```

## Verify

From the generated project directory:

```powershell
go test ./...
```

For browser tests, start from `test/playwrightgo/smoke_test.go` and run:

```powershell
go test -tags playwrightgo ./test/playwrightgo -run TestMainSuite -v
```
