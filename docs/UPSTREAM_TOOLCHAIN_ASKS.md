# Upstream Go/wasm toolchain asks (tracked)

Some DevX dimensions are capped by the Go/wasm toolchain itself, not by GoWebComponents.
Per the devx-maxxing "platform-honest anchor" policy, the framework ships everything
achievable in its own code and **tracks the irreducible upstream constraint here** so the
honest ceiling is recorded rather than hidden. This file is that record.

## A2 — Time-to-first-render: cold golden-path time

- **What GWC ships:** prebuilt cross-platform `gwc` binaries (no from-source CLI build —
  `release.yml`, 6 targets incl. `windows/arm64`); a hot-reload-capable dev loop
  (`gwc dev`); curated idiomatic starters; and **published, CI-gated wasm cold/warm build
  timings** (`docs/benchmarks/build-times.json`, `build-times.yml`).
- **Irreducible upstream cost:** on a *clean OS* the first run is dominated by installing
  the Go toolchain and the cold module fetch — neither of which a UI framework can remove.
  A full clean-machine `<2 min` therefore depends on the Go toolchain's install/fetch
  speed (upstream) or on a framework-hosted **CDN-cached starter tarball / module proxy**
  (the recorded stretch route to literal 10; carries a baked-artifact maintenance cost).
- **Status:** platform-honest 10 met (prebuilt binaries + published timings + warm path);
  literal-10 stretch (CDN starter) is deferred, recorded here.

## C2 — Build speed: incremental wasm linking

- **What GWC ships:** `gwc warm` persistent build-cache daemon; `gwc buildreport`
  ("what rebuilt and why" from the Go build action graph); and the published, CI-gated
  cold/warm wasm timing pair with a ratio gate proving the cache delivers.
- **Irreducible upstream cost:** the Go linker performs a **full-world link on every wasm
  build** — there is no incremental wasm linker in the Go toolchain. The measured cold
  wasm build (~9 s for the counter app) is that full relink; the warm rebuild (~0.4 s) is
  the Go build cache serving unchanged packages. A genuinely *incremental* wasm link is a
  Go-compiler/linker capability that does not exist today.
- **Status:** platform-honest 10 met (daemon + report + gated published timings); literal
  10 (sub-second cold relink) requires an upstream Go incremental-wasm-linker, tracked here.

## C4 — Debugging: browser source maps from Go/wasm

- **What GWC ships:** an installable browser DevTools extension (live tree / props-state /
  commit profiling), a snapshot time-travel engine, and a `SetWASMStackFrameMapper` hook
  (`internal/runtime/panic_source_map.go`) for browser-stack ↔ Go-symbol correlation.
- **Irreducible upstream cost:** Go's `js/wasm` output does not emit a browser-consumable
  `SourceMap v3`; there is no production DWARF→source-map tool for Go/wasm. Native
  source-map debugging is a Go toolchain capability that does not exist today.
- **Status:** platform-honest 10 met (extension + time-travel + symbol mapper); literal 10
  (native source maps) requires upstream Go support, tracked here.

### Stack-correlation workaround (use until native source maps exist)

Because the browser only sees wasm frames, GWC exposes a **frame mapper** so a panic stack can
be translated back to application-owned Go locations. Install one at startup:

```go gwc:build
//go:build js && wasm

package main

import "github.com/monstercameron/GoWebComponents/v6/ui"

func main() {
	// Map an observed wasm/browser panic frame back to a higher-signal Go location.
	// Source the mapping from your build's symbol/DWARF dump, a //go:generate manifest,
	// or a hand-maintained table for the hot paths you care about.
	ui.SetWASMStackFrameMapper(func(frame ui.WASMStackFrame) (ui.WASMStackFrame, bool) {
		// Translate a known wasm frame to your Go source. Source this from a symbol table
		// you ship (e.g. generated from the build's DWARF) keyed by frame.Function.
		if frame.Function == "main.handleClick" {
			return ui.WASMStackFrame{Function: "handleClick", File: "app/handlers.go", Line: 42}, true
		}
		return frame, false // fall through to the raw frame
	})
	// ... ui.Run(...) etc.
}
```

The runtime threads every panic frame through this mapper (`translateWASMStackFrame`), so the
in-page error overlay and the devtools panel show your Go function/file/line instead of an opaque
wasm offset. This is the documented bridge until the Go toolchain emits browser source maps
natively (the upstream ask above), at which point the mapper becomes unnecessary.

---

These are the only three dimensions whose literal-10 gap is attributable to the host
toolchain. Everything else in the devx-maxxing rubric is in-scope framework work. When the
Go toolchain gains incremental wasm linking or wasm source maps, revisit C2/C4 for the
literal-10 rung.
