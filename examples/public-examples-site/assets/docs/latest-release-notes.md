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

This is a documentation and packaging checkpoint for the RC. The fair v6/v7
production benchmark, CashFlux consumer acceptance, the complete release
workflow, and final `go get` verification remain release gates. Windows v6.1
manual acceptance is deferred by the release plan; that deferral does not make
the v7 stable tag available. No `v7.0.0-rc.1` or `v7.0.0` tag has been
published.

### Documentation

- Migration guide: [`docs/MIGRATION_V7.md`](../../../../docs/MIGRATION_V7.md)
- Versioning policy: [`../../../../VERSIONING.md`](../../../../VERSIONING.md)
