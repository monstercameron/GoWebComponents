# Wasm Build Experiments

This page defines the intended experiment matrix for wasm build and release optimization work.

Use it when comparing build flags, artifact size, startup cost, compatibility settings, and compression choices so release decisions stay evidence-driven instead of anecdotal.

## Current Status

- The repo already ships real wasm experiment helpers for phase timing, compression comparison, cache-topology comparison, manifest comparison, and Go toolchain comparison.
- This page is the current policy layer over those scripts and over the representative benchmark targets used by the repo.
- The release experiment story now feeds directly into the shipped launcher and manifest contract rather than living as isolated notes.

## Experiment Matrix

The intended comparison matrix should use a stable set of representative variants.

The baseline matrix is:

- plain `go build`
- release-style build with `-trimpath`
- release-style build with `-trimpath -buildvcs=false`
- stripped release build with `-trimpath -buildvcs=false -ldflags="-s -w"`
- compressed delivery variants for emitted release artifacts
- post-link optimized variants when a candidate tool such as `wasm-opt` is available

Each experiment should record:

- exact build flags
- toolchain version
- target package
- emitted artifact hashes and sizes
- any post-processing or compression steps applied after `go build`

## Representative Target Set

Build experiments should run against a stable target set instead of one convenient package.

The current canonical targets are:

- small feature target: `./examples/21-ui-render`
- routed mid-sized target: `./examples/56-browser-router`
- large showcase target: `./examples/86-atlas-commerce-os/client`

Why these targets are the current baseline:

- `21-ui-render` keeps a minimal mount path in the matrix so flag and cache overhead are visible on a small package
- `56-browser-router` adds route registration and browser-router behavior without the full server-backed Atlas surface
- `86-atlas-commerce-os/client` exercises the largest production-shaped wasm hydration target currently shipped in the repo

Selection rules for future target changes:

- prefer long-lived examples with stable teaching intent over temporary experiments
- keep one target per size tier unless a second target is needed to explain a materially different result
- update the target list only when a replacement better represents the same tier or the current target is removed

The machine-readable copy of this target set should live beside build tooling so experiment scripts and CI jobs can consume the same package list.

## Cold And Warm Build-Time Measurement

Build-speed experiments should distinguish first build from cached rebuilds.

The intended timing set is:

- cold build from a cleared build cache
- warm build with no source changes
- small-edit rebuild after a targeted source change in a representative package

Recommended measurement rules:

- keep package targets fixed across runs
- record host OS, CPU, and Go toolchain version with the timing results
- separate pure build time from any later compression or packaging steps
- prefer repeated runs and saved outputs over one-off shell timing

## Cache Strategy Effects

Build-speed experiments should record cache topology explicitly because local machines and CI workers do not behave the same way.

The current PowerShell helper for this pass is `tools/compare-wasm-build-cache.ps1`.

Its current comparison shape is:

- shared build-cache cold run with a reused module cache
- shared build-cache warm run with the same reused module cache
- shared build-cache small-edit rebuild with the same warmed caches
- isolated build-cache run with the normal module cache still available
- CI-style cold run with fresh build and module caches
- CI-style warm run after those isolated caches have been hydrated once
- CI-style small-edit rebuild after those isolated caches have been hydrated once

The intended use is to keep local guidance, CI expectations, and future cache-tuning work tied to measured `GOCACHE` and `GOMODCACHE` behavior instead of one unqualified build-time number.

## CI-Friendly Comparison

Saved experiment manifests should be compared in a way that automation can fail fast on material regressions.

The current PowerShell helper for this pass is `tools/compare-wasm-experiment.ps1`.

Its current comparison shape is:

- flatten numeric metrics from the saved JSON manifests produced by the current wasm experiment helpers
- treat timing metrics such as `*_ms` and `module_download_ms` as regression-budgeted time costs
- treat artifact `bytes` as size metrics with a separate threshold
- emit a machine-readable comparison summary and exit non-zero when the candidate exceeds the configured regression budget

The intended use is to compare a saved baseline against a candidate run in CI before a new build flag, cache strategy, compression path, or release default is documented as accepted.

## Phase-Attributed Timing

Build-loop timing should be recorded as separate phases, not one wall-clock number.

The current phase breakdown is:

- `go_build_ms` for the `go build` phase
- `gzip_ms` and `brotli_ms` for individual compression steps when enabled
- `compression_total_ms` for total post-processing and compression time
- `serve_reload_ms` when a browser or dev-server probe provides reload timing for the same build artifact
- `total_wall_ms` for overall elapsed time across the measured steps

The current PowerShell helper for this shape is `tools/measure-wasm-build.ps1`.

Use it to emit a JSON manifest that keeps phase timing beside artifact size and hash records so later build-flag or cache experiments can compare the same package across repeated runs.

## Size And Startup Tradeoff Harness

Artifact size is not the same thing as user-perceived startup cost.

The intended harness should capture:

- raw `.wasm` size
- compressed delivery size
- browser download cost under representative network conditions
- instantiate or compile time
- time to first meaningful interaction or hydrated route readiness

The release manifest emitted by the current helper is one input to that harness, not the whole experiment by itself.

The current browser-side startup probes use Playwright against real example entrypoints:

- `examples/tests/startup-experiments.spec.ts` for the small and routed mid-sized static targets
- `examples/tests/86-atlas-commerce-os-startup.spec.ts` for the large Atlas SSR target

Those probes are intended to record ready-to-interact timing, network-idle timing when available, first client interaction timing, and browser resource timing entries for emitted `.wasm` assets.

## Post-Processing And Compression Comparison

Post-processing comparisons should record both supported and unavailable variants explicitly.

The current PowerShell helper for this pass is `tools/compare-wasm-compression.ps1`.

Its current comparison shape is:

- plain raw wasm output
- stripped raw wasm output
- stripped wasm with gzip and brotli delivery sidecars when either the PowerShell runtime or the repo's Node-based fallback can emit them
- optimized wasm output via `wasm-opt`, resolved from `PATH` or the `binaryen` npm package through `npx`
- explicit environment notes only when neither compression nor optimizer fallback can be resolved

That keeps the experiment history honest on hosts where an optimizer or compression path still cannot be resolved, while allowing the repo to complete the comparison matrix on standard Node-equipped contributor machines.

## Toolchain Regression Tracking

Go version upgrades should be measured before the documented baseline changes.

The current PowerShell helper for this pass is `tools/compare-wasm-go-toolchain.ps1`.

Its current comparison shape is:

- run the same package through `tools/measure-wasm-build.ps1` with an explicit baseline Go executable and candidate Go executable
- keep each toolchain's build manifest in a stable output directory
- compare the saved manifests with `tools/compare-wasm-experiment.ps1`
- emit a small summary file that records the two toolchain versions together with the comparison result

The intended use is to compare the currently pinned Go toolchain against a candidate upgrade before release docs or CI baselines are updated.

## Build-Flag Tradeoff Evaluation

Build flags should be measured across more than one dimension.

The intended tradeoff table for any candidate flag set is:

- build time
- raw artifact size
- compressed artifact size
- startup or instantiate time
- runtime benchmark deltas where relevant
- debugging cost or metadata loss

This keeps the project from assuming that a smaller binary is always better if compile times, startup behavior, or debuggability regress materially.

## `GOWASM` Feature Toggle Policy

`GOWASM` feature toggles should be treated as compatibility experiments, not hidden defaults.

The intended policy is:

- record which `GOWASM` feature set is active during each experiment
- evaluate whether a toggle changes browser compatibility, output size, or startup behavior for the documented support matrix
- do not standardize on a non-default toggle unless the compatibility tradeoff is explicit and documented

If a toggle becomes part of the recommended release path later, that choice should be reflected in browser-support and release-engineering docs, not only in CI scripts.

## Accepted And Rejected Results

Keep a short experiment record beside the matrix so future release changes have local context.

Current evidence-backed outcomes from the scripted measurements in this repo are:

- accepted release baseline: `-trimpath -buildvcs=false -ldflags="-s -w"`
	On `./examples/21-ui-render`, the stripped release build emitted `4575365` raw bytes versus `4680861` raw bytes for plain `go build`, so the stripped profile remains the documented release starting point.
- accepted delivery sidecar: gzip compression for release artifacts
	On the same target, the stripped release artifact compressed to `1260895` gzip bytes, and the repo's current release helper already emits that sidecar by default.
- accepted release packaging default: gzip plus Brotli sidecars in the launcher-owned release path
	The current `gwc release` contract emits both sidecars by default and records them in `wasm-release-manifest.json`, so experiment work now evaluates whether that default should change, not whether Brotli exists at all.
- rejected as a release default: plain unstripped `go build`
	The current measurements show a larger artifact with no delivery-sidecar advantage, so plain output is still treated as a debug-oriented build path rather than the release recommendation.
- rejected as an inner-loop expectation: CI-style cold-cache timings
	The cache comparison for `./examples/21-ui-render` showed `18233` ms of module download time and `6554` ms of compile time for the clean CI-style pass, versus `127` ms `go_build_ms` for the warmed shared-cache rebuild, so CI-cold numbers should not be used to describe local incremental workflow quality.
- not accepted yet: `wasm-opt` post-processing in the release workflow
	`tools/compare-wasm-compression.ps1` now measures `wasm-opt` output through `PATH` or `npx --package binaryen`, but the release workflow still does not promote it to the default until the saved comparison data justifies the extra post-processing step.
- not accepted yet: a narrower Brotli-only or optimizer-coupled serving policy
	The current launcher default is to emit gzip plus Brotli sidecars together. Further experiment work can still justify changing that delivery policy, but the open question is about policy refinement rather than Brotli availability.

Update this section when a new flag set, cache policy, toolchain, or post-processing step becomes accepted, rejected, or explicitly deferred.

## Current Boundary

This document defines the intended experiment model only.

It currently covers real scripted comparison flows, but it is still not the same thing as the release contract itself.

It does not yet claim:

- automated experiment execution for every variant in CI
- accepted post-link optimizer tooling
- complete startup-cost coverage for representative releases

Those remain separate backlog work.
