# Wasm Build Profiles and Release Engineering

This page defines the current wasm build-profile and release-engineering model for GoWebComponents applications.

Use it when deciding how local development builds, source-debug builds, CI verification builds, benchmark builds, and production release builds should differ, and which production-oriented build flags are the default safe baseline.

## Current Status

- The repo already ships a real Go-native launcher path for wasm build and release work through `gwc build` and `gwc release`.
- `gwc release` emits a release-profile wasm artifact, defaults to gzip plus Brotli sidecars, writes `wasm-release-manifest.json`, supports optional budget enforcement, and can emit machine-readable JSON summaries.
- That release manifest is also the shipped handoff into the `pwa` package through `pwa.ParseWasmReleaseManifestJSON(...)`, `pwa.BuildServiceWorkerAssetPlan(...)`, and `pwa.BuildCacheStoragePlan(...)`.

## Build Profiles

The current wasm build profiles are explicit, not one generic `go build` command reused for every job.

The current canonical profiles are:

- development build:
  optimized for local iteration and compatibility with local dev servers
- source-debug build:
  optimized for browser crash and stack-frame correlation with untrimmed local paths and disabled compiler optimization
- CI verification build:
  optimized for deterministic validation that the target package still builds and boots under the documented release-like settings
- benchmark build:
  optimized for repeatable measurement with the flags and artifact shape clearly recorded beside the benchmark output
- production release build:
  optimized for deployable size, stable paths, and predictable runtime behavior in staging or production hosting
- TinyGo build:
  optimized for constrained leaf apps that can compile with TinyGo's `wasm` target and should be evaluated for smaller shipped artifacts

The repo now exposes the baseline build-profile runner through:

```powershell
go run ./tools/gwc build -app .\path\to\main.go -profile ci
```

That command currently covers single-target wasm builds and JSON build summaries. The standard profiles use the Go toolchain with `GOOS=js GOARCH=wasm`; `-profile debug` adds `-gcflags=all=-N -l` without `-trimpath` or strip flags; `-profile tinygo` uses `tinygo build -target=wasm -opt=z -tags production` and fails early with an install hint when TinyGo is not on `PATH`.

The repo now also exposes the baseline release packager through:

```powershell
go run ./tools/gwc release -app .\path\to\main.go -out-dir .\bin\wasm-release
```

That release command emits the raw release-profile wasm artifact, supports `-compression none|gzip|brotli|gzip+brotli`, defaults to gzip plus Brotli sidecars, writes `wasm-release-manifest.json`, applies optional budget enforcement, and emits a JSON release summary for automation.

Recommended differences by profile:

- development builds should stay fast and small enough for the inner loop
- debug builds should preserve untrimmed local paths and record `gcflags` in build or release summaries
- CI verification builds should prove that the chosen production-oriented flags still compile and pass smoke checks
- benchmark builds should record the exact flags and compression context used, so size and startup numbers stay attributable
- production builds should use the documented release baseline consistently instead of ad hoc per-app flag choices
- TinyGo builds should start as a compatibility spike for leaf apps; unsupported runtime, reflection, or package dependencies should fail the build instead of silently weakening application behavior

## Recommended Production Build Flags

The current baseline production command is conservative and reproducible.

Recommended starting point:

```powershell
$env:GOOS = "js"
$env:GOARCH = "wasm"
go build -trimpath -ldflags="-s -w" -o app.wasm ./path/to/app
```

Why this baseline:

- `-trimpath` reduces machine-specific path leakage and helps reproducibility
- `-ldflags="-s -w"` strips symbol and DWARF data that are usually unnecessary in deployed browser artifacts
- ordinary `go build` stays the compiler front door so the build remains explainable and compatible with existing toolchain expectations

Tradeoffs:

- stripped artifacts are harder to debug in-browser
- production flag choices should be paired with `gwc build -profile debug` for browser investigation, not used blindly for every local workflow
- if a team needs richer postmortem debugging, it should keep a parallel debug artifact policy instead of weakening the release baseline accidentally

Debug profile starting point:

```powershell
go run ./tools/gwc build -app .\path\to\main.go -profile debug -out .\bin\debug\app.wasm
```

Use this when you need symbolized wasm stack frames, untrimmed local paths, and less surprising browser pauses while correlating DevTools output or crash reports to Go source. Current Go `js/wasm` builds do not emit browser source maps or `.debug_*` DWARF sections, so this is a symbolized-stack workflow rather than full browser source stepping.

TinyGo profile starting point:

```powershell
go run ./tools/gwc build -app .\path\to\main.go -profile tinygo
```

Use it when the app is intentionally small and its dependencies are known to compile under TinyGo. Keep the standard Go release profile as the compatibility baseline until CI proves the TinyGo artifact boots and passes the same smoke checks.

## Debug Info And Metadata Policy

The current release policy is:

- production artifacts default toward stripped debug information
- debug-friendly builds may preserve symbols or richer metadata, but that should be explicit and separate from the normal release artifact
- build metadata, VCS details, and artifact provenance should be preserved in release notes or external build records when operationally needed, not necessarily embedded into every shipped wasm binary

Recommended rule:

- keep deployed browser artifacts small and predictable
- keep debugging and provenance information in release automation, CI records, or dedicated debug artifacts when those needs exist

## Size Budgets And Regression Tracking

Release wasm work should have explicit artifact-size expectations.

The current policy is:

- track raw `.wasm` size plus compressed delivery sizes
- set budgets per release target or representative example instead of relying on anecdotal size impressions
- fail release-oriented verification when agreed budgets are exceeded

The PowerShell release helper now supports optional budget enforcement through a simple JSON file with keys such as:

- `raw_bytes`
- `gzip_bytes`
- `brotli_bytes`

That gives the repo one concrete path for turning size expectations into a pass/fail release check.

The current release manifest and planning tests live in `tools/gwc/release_test.go`, `pwa/release_manifest_test.go`, and `pwa/cache_storage_test.go`, so the release record and the downstream PWA planning contract are validated together.

## Artifact Size Reporting And Comparison

Every release-style build should produce attributable size records.

The current reporting shape is:

- raw `.wasm` size
- gzip sidecar size
- brotli sidecar size when that sidecar is emitted
- sha256 hashes for each emitted artifact

The PowerShell helper and `gwc release` now emit `wasm-release-manifest.json` containing those size and hash records so builds can be compared mechanically instead of by ad hoc shell output.

That same manifest is now also the first-class input for service-worker release planning through `pwa.ParseWasmReleaseManifestJSON(...)` and `pwa.BuildServiceWorkerAssetPlan(...)`, so cache namespaces and safe-reload revision checks can reuse the shipped release record instead of maintaining a second wasm-specific PWA manifest.

## Reproducible Release-Build Guidance

Release builds should be reproducible enough to explain what was shipped.

Recommended rules:

- pin the Go toolchain version in CI and release automation
- use one documented release command path instead of per-developer flag variations
- prefer `-trimpath` and `-buildvcs=false` for reproducible release artifacts unless a deliberate provenance requirement says otherwise
- record the emitted artifact hashes and sizes in `wasm-release-manifest.json`
- treat release manifests, hashes, and build flags as part of the release record, not as incidental console output

## Build-Output Conventions

The intended release-output directory is predictable and self-describing.

The current convention is:

- one release output directory per target
- the primary wasm artifact at the configured binary name, defaulting to `app.wasm`
- sidecars at `app.wasm.gz` and, when available, `app.wasm.br`
- one `wasm-release-manifest.json` file describing emitted artifacts, relative paths, sizes, and hashes

This gives static hosts, SSR servers, and deployment automation one stable place to read optimized artifact metadata.

## Current Boundary

This document defines the current build-profile and release-packaging baseline.

The current Go-native launcher boundary is:

- `gwc build` for single-target js/wasm artifacts and JSON build summaries
- `gwc release` for raw wasm plus gzip and Brotli packaging, manifest emission, optional budgets, and JSON release summaries
- `pwa.ParseWasmReleaseManifestJSON(...)`, `pwa.BuildServiceWorkerAssetPlan(...)`, and `pwa.BuildCacheStoragePlan(...)` for turning the release manifest into cache and service-worker planning inputs

It does not yet claim:

- post-link optimizer integration such as `wasm-opt`
- first-class startup-cost measurement for representative releases
- full end-to-end release verification beyond artifact emission, manifest consistency, and the test coverage already in the launcher and `pwa` packages

Those remain separate backlog work.

For the broader comparison matrix behind future release-engineering decisions, see [BUILD_EXPERIMENTS.md](wasm-build-experiments.md).
