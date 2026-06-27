# Bill Splitter — GoWebComponents

The study **anchor**. Go compiled to WebAssembly; the whole UI (markup, state,
styling) is Go. Implements [../../SPEC.md](../../SPEC.md).

## Run (from the repo root)

```powershell
go run ./tools/gwc dev `
  -app .\research\framework-bench\apps\gwc\main.go `
  -root .\research\framework-bench\apps\gwc `
  -html .\research\framework-bench\apps\gwc\index.html `
  -wasm .\research\framework-bench\apps\gwc\dist\main.wasm
```

Or build the wasm directly:

```powershell
$env:GOOS="js"; $env:GOARCH="wasm"
go build -o .\research\framework-bench\apps\gwc\dist\main.wasm .\research\framework-bench\apps\gwc
```

`dist/` is git-ignored.

## What it exercises

| Surface | Used for |
|---|---|
| `html/shorthand` | every tag + `Map` / `MapKeyed` / `If` control flow |
| `css/u` (typed utilities) | the entire design — no class strings, no Tailwind toolchain; the sink auto-injects `<style id="gwc-css">` |
| `ui.UseState` | local `bill`, `tipPercent`, `people` |
| `ui.UseEvent` | input / click handlers |
| `ui.CreateElement` | component composition with typed props (`Header`, `Inputs`, `PresetButton`, `Results`, `Breakdown`, `Footer`) |
| `state.UseAtom` | **shared** `theme` + `roundUp`, read/written across Header, Results, Footer |
| `ui.Run` | one-line mount + keep-alive |

## Notes / comparison

- **Styling is generated, not shared.** Unlike the other apps (which import the
  canonical `shared/styles.css`), GWC reproduces the same design through typed,
  compile-checked `css/u` utilities resolved against the css theme. That divergence
  is intentional and is itself a comparison dimension: a typo in a utility is a
  compile error, and there is no separate CSS build step.
- **Shared state** uses keyed atoms (`bs.theme`, `bs.roundUp`); any component that
  calls `useTheme()` / `useRoundUp()` subscribes and re-renders on change.
- Compile-verified for both `GOOS=js GOARCH=wasm` and native in this repo's CI lanes.
