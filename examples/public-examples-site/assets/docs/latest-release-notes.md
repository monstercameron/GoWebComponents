# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v7.0.0 — stable

GoWebComponents v7 moves the public module to
`github.com/monstercameron/GoWebComponents/v7` and raises the Go floor to
1.27. It includes the API, tooling, hydration, router, telemetry, error
reporting, and dependency-graph changes described in the v7 development log.

### Upgrade notes

- Change the module requirement and Go imports from `/v6` to `/v7`, then run
  `go mod tidy`. Pin `v7.0.0` for the stable release.
- Follow [`docs/MIGRATION_V7.md`](../../../../docs/MIGRATION_V7.md) for typed handlers,
  opt-in Markdown registration, router options, tracing headers, reconnect
  defaults, and the `kvstate` identifier migration.
- Invoke the CLI from its nested `tools/` module; browser tests, telemetry SDK
  tests, and dependency-heavy examples also live in separate modules.

### Verification and accepted performance costs

The [`v7.0.0-rc.1` release workflow](https://github.com/monstercameron/GoWebComponents/actions/runs/39595468422)
passed its Windows build, tests, browser suites, vulnerability scan, SBOM,
reproducible build, and candidate consumer guard. A fresh external module
fetched the published RC and compiled public packages natively and for
js/wasm. The exact-source CashFlux migration passed its full short Go suite,
Wasm build, and production browser suite; pinned GoGRPCBridge stress and P0
browser health probes passed.

The dedicated v6/v7 production comparison matched 179 native entries with a
6.11% faster v7 geometric mean and 148 Wasm entries with a 0.12% faster v7
geometric mean. `TransitionListRefresh250` was 22.72% slower on Wasm against
the 15% per-entry limit. The React 19 Example 201 browser comparison had no
scenario beyond its +30% noise limit, but its final geometric mean was 4.27%
slower. The maintainer reviewed and accepted both measured costs for this
release on 2026-10-21. Raw runs and the size caveats are recorded in
[`docs/benchmarks/V7-RELEASE-EVIDENCE.md`](../../../../docs/benchmarks/V7-RELEASE-EVIDENCE.md).

Windows v6.1 manual acceptance remains deferred; v7 is released first by
maintainer decision. The v6 line retains its support window under
[`VERSIONING.md`](../../../../VERSIONING.md).
