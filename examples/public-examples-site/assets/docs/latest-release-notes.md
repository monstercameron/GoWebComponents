# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v7.0.0-rc.1 — release candidate (pending final release gates)

This candidate packages the v7 module-path migration and the API, tooling,
hydration, router, telemetry, and error-reporting changes recorded in the v7
development log. It is intended for explicit RC testing with Go 1.27;
the stable `v7.0.0` tag has not been published.

### Upgrade notes

- Change consumer imports and the module requirement from `/v6` to `/v7`.
- Use Go 1.27.3 or newer and follow [`docs/MIGRATION_V7.md`](docs/MIGRATION_V7.md)
  for typed input handlers, opt-in Markdown registration, router options,
  tracing headers, reconnect defaults, and the `kvstate` identifier migration.
- Nested tools, browser tests, telemetry SDK tests, and dependency-heavy
  examples are separate modules; run their checks from their module directory.

### Verification and release status

This is a documentation and packaging checkpoint for the RC. An isolated
CashFlux `/v7` migration against exact source `5a2ce15b` passed its Wasm
build, full short Go suite, and production browser suite (27 passed, 2 skipped,
2 timing-sensitive retries). Go 1.27 required regenerating CashFlux's eight
icon fixtures; the same mismatch occurs on its v5 source with Go 1.27.
Repeated v5/v7 route timings did not show a consistent v7 slowdown. The
final-runtime validation-only release workflow passed its Windows build and
test gates. Dedicated fair v6/v7 production comparisons completed; native was
6.11% faster by geometric mean, and Wasm was 0.12% faster overall.
Published-RC `go get` consumer verification remains a release gate. The
Example 201 browser comparison stayed within its
per-scenario noise limit but missed the positive geomean target, and the
transition-list Wasm benchmark exceeded the 15% per-entry limit. The
maintainer accepted both measured costs for v7 on 2026-10-21; see
`docs/benchmarks/V7-RELEASE-EVIDENCE.md`. Windows v6.1 manual acceptance is
deferred; v7 may release before v6.1 after its own gates pass.

### Documentation

- Migration guide: [`docs/MIGRATION_V7.md`](../../../../docs/MIGRATION_V7.md)
- Versioning policy: [`../../../../VERSIONING.md`](../../../../VERSIONING.md)
