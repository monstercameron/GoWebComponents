# Wasm Build Experiments

This page defines the intended experiment matrix for wasm build and release optimization work.

Use it when comparing build flags, artifact size, startup cost, compatibility settings, and compression choices so release decisions stay evidence-driven instead of anecdotal.

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

## Size And Startup Tradeoff Harness

Artifact size is not the same thing as user-perceived startup cost.

The intended harness should capture:

- raw `.wasm` size
- compressed delivery size
- browser download cost under representative network conditions
- instantiate or compile time
- time to first meaningful interaction or hydrated route readiness

The release manifest emitted by the current helper is one input to that harness, not the whole experiment by itself.

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

## Current Boundary

This document defines the intended experiment model only.

It does not yet claim:

- automated experiment execution for every variant in CI
- accepted post-link optimizer tooling
- complete startup-cost coverage for representative releases

Those remain separate backlog work.
