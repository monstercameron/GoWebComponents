# Wasm Build Profiles and Release Engineering

This page defines the intended wasm build-profile and release-engineering model for GoWebComponents applications.

Use it when deciding how local development builds, CI verification builds, benchmark builds, and production release builds should differ, and which production-oriented build flags are the default safe baseline.

## Build Profiles

The intended wasm build profiles are explicit, not one generic `go build` command reused for every job.

The current canonical profiles are:

- development build:
  optimized for local iteration, easier debugging, and compatibility with local dev servers
- CI verification build:
  optimized for deterministic validation that the target package still builds and boots under the documented release-like settings
- benchmark build:
  optimized for repeatable measurement with the flags and artifact shape clearly recorded beside the benchmark output
- production release build:
  optimized for deployable size, stable paths, and predictable runtime behavior in staging or production hosting

Recommended differences by profile:

- development builds may preserve fuller debug metadata and simpler artifact naming
- CI verification builds should prove that the chosen production-oriented flags still compile and pass smoke checks
- benchmark builds should record the exact flags and compression context used, so size and startup numbers stay attributable
- production builds should use the documented release baseline consistently instead of ad hoc per-app flag choices

## Recommended Production Build Flags

The intended baseline production command is conservative and reproducible.

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
- production flag choices should be paired with a separate debug-friendly build path, not used blindly for every local workflow
- if a team needs richer postmortem debugging, it should keep a parallel debug artifact policy instead of weakening the release baseline accidentally

## Debug Info And Metadata Policy

The intended release policy is:

- production artifacts default toward stripped debug information
- debug-friendly builds may preserve symbols or richer metadata, but that should be explicit and separate from the normal release artifact
- build metadata, VCS details, and artifact provenance should be preserved in release notes or external build records when operationally needed, not necessarily embedded into every shipped wasm binary

Recommended rule:

- keep deployed browser artifacts small and predictable
- keep debugging and provenance information in release automation, CI records, or dedicated debug artifacts when those needs exist

## Size Budgets And Regression Tracking

Release wasm work should have explicit artifact-size expectations.

The intended policy is:

- track raw `.wasm` size plus compressed delivery sizes
- set budgets per release target or representative example instead of relying on anecdotal size impressions
- fail release-oriented verification when agreed budgets are exceeded

The PowerShell release helper now supports optional budget enforcement through a simple JSON file with keys such as:

- `raw_bytes`
- `gzip_bytes`
- `brotli_bytes`

That gives the repo one concrete path for turning size expectations into a pass/fail release check.

## Artifact Size Reporting And Comparison

Every release-style build should produce attributable size records.

The intended reporting shape is:

- raw `.wasm` size
- gzip sidecar size
- brotli sidecar size when that sidecar is emitted
- sha256 hashes for each emitted artifact

The release helper now emits `wasm-release-manifest.json` containing those size and hash records so builds can be compared mechanically instead of by ad hoc shell output.

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

This document defines the intended build-profile and flag baseline only.

It does not yet claim:

- guaranteed brotli sidecar generation on every host runtime
- post-link optimizer integration such as `wasm-opt`
- first-class startup-cost measurement for representative releases

Those remain separate backlog work.

For the broader comparison matrix behind future release-engineering decisions, see [BUILD_EXPERIMENTS.md](BUILD_EXPERIMENTS.md).
