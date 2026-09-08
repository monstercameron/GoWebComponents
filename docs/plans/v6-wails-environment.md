# V6 Wails environment and browser baseline

Recorded 2026-09-08 on Windows ARM64 hardware with the repository checkout at
`C:\Users\mreca\Desktop\GoWebComponents`. This is evidence for WAILS-002 and
WAILS-003; it does not change the global Go environment, the root module, or
the Wails source checkout.

## Windows prerequisites

| Probe | Result |
| --- | --- |
| Go | `go version go1.26.3 windows/arm64` |
| Go target environment | `go env` reports `GOOS=windows`, `GOARCH=amd64`, `CGO_ENABLED=0`, `CC=gcc`, `CXX=g++`; these values are already persisted in `C:\Users\mreca\AppData\Roaming\go\env` and were not changed by this work |
| WebView2 | Installed machine-wide; registry `HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}` reports `pv=152.0.4191.66`; `msedgewebview2.exe` has file/product version `152.0.4191.66` |
| Wails CLI | `Get-Command wails3` finds no command on PATH; the example build invokes the pinned CLI using `go run github.com/wailsapp/wails/v3/cmd/wails3` and the example's local module replacement. |
| Native compilers | `gcc`, `g++`, `cl`, `msbuild`, and `make` are not on `PATH`; `clang` is available at `C:\gwcbin\clang.exe`. The isolated host already compiles with the pinned Wails Windows path and `CGO_ENABLED=0`, so these PATH absences are observations, not blockers. |
| Node tooling | `node` and `npm` are available under `C:\Program Files\nodejs` |

The installed WebView2 runtime is sufficient for a Windows WebView probe. A
global `wails3` installation is not required: the example build invokes the CLI
through its pinned Go dependency. The host's Windows build path uses `CGO_ENABLED=0`, and
the isolated host already compiles; no global installation or environment
change was attempted. The `wails3` command name should be used for v3 probes,
not the legacy `wails` name.

## Isolated module resolution

The proposed module now exists at
`examples/desktop/wails-counter/` (created by the desktop example work). Its
`go.mod` declares:

```text
module example.com/gwc-wails-counter
go 1.26.0
require github.com/monstercameron/GoWebComponents/v5 v5.0.0
require github.com/wailsapp/wails/v3 v3.0.0-beta.17
replace github.com/monstercameron/GoWebComponents/v5 => ../../..
replace github.com/wailsapp/wails/v3 => ../../../third_party/wails/v3
```

The Wails source replacement resolves to revision
`5bce785eb1efbd121ef2e1cb2588eb50b9c59068`, described by its checkout as
`v3.0.0-beta.17`. From the isolated module:

```text
go list -m all       # exit 0; includes both local replacements
go test ./...        # exit 0; assets, cmd/desktop, contracts, services
```

The root `go.mod` still has no Wails requirement. The isolated module imports
the currently declared GWC path (`github.com/monstercameron/GoWebComponents/v5`)
and uses a local replacement rather than assuming root replacements propagate.

## Existing browser counter baseline

Build command from the repository root:

```text
go run ./tools/gwc build -app .\examples\public\counter\main.go \
  -root .\examples\public\counter -json
```

Result: success; output is
`examples/public/counter/bin/main.wasm`, 6,647,102 bytes, SHA-256
`7b312e1724b217a72d75617f844567a26bb086333f3288083330d9b4d4acb627`.
The development profile is `js/wasm`, `ldflags=-w`; the build report emitted a
regexp-link size warning (~290 kB). A live `gwc dev` run reported a successful
Wasm build in `861.8094ms` (raw artifact 6.34 MB, gzip transfer 2.07 MB).

Browser behavior was exercised with the repository's Playwright-Go tests:

```text
go run ./tools/gwc test -lane browser -json
```

Result: pass, `TestMainSuite` in 20.614s (components, integration, and state
subtests). The focused cross-browser example run was:

```text
go test -tags playwrightgo ./test/playwrightgo/examples \
  -run TestCrossBrowserConformance -v
```

The counter startup/render/increment scenario passed in all three engines:
Chromium (0.84s), Firefox (3.62s), and WebKit (1.89s). The complete matrix
was not green because the unrelated WebKit `storage/snapshot-storage` case
lost its temporary catalog server (`Could not connect to server`); the run
also ended with a Windows temporary test executable unlink `Access is denied`.
That failure is retained as a limitation, not treated as a counter failure.

The standalone `gwc dev` livereload server served its status endpoint and
`main.wasm`, but had no generated HTML route at `/` (HTTP 404). Therefore the
startup timing above is the repeatable browser-suite measurement, not a claim
that the raw livereload root is a complete packaged application.

## Remaining risk / next step

WAILS-002 environment and module-isolation evidence is available. WAILS-003 browser
counter behavior and artifact baseline are recorded. These probes preceded the
complete desktop example. Later actual-WebView and build evidence belongs in
the [Astra verification report](v6-wails-verification.md), including the exact
native target and remaining manual/platform limitations.
