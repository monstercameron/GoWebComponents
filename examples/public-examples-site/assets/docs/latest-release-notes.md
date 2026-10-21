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

## Unreleased — v6.1.0 in progress

### Added

- Typed, separately gated Windows APIs for expanded window controls, printing,
  screen geometry, runtime/context menus, tray, global shortcuts, template-owned
  child windows, window events and opt-in drops, environment inspection,
  external URLs, file-manager reveal and autostart.
- Expanded GWC-built Windows API Lab controls and typed event reporting;
  capability-filtered native bindings keep web/Wasm code independent of Wails.

### Fixed

- Windows menu resource ownership, shortcut isolation and cleanup, radio
  reselection and context selection dispatch through a tested, pinned Wails fork.
- Child-window close/publication races and tray-handler UI-thread publication.
- Cross-target Windows path validation, feature/method mapping, request validation
  and explicit rejection of unsupported native options.
- Tester duplicate dispatch, invalid window requests, save-cancellation wording,
  child-template selection and misleading success outcomes on API errors.

### Verification and release status

Native and CI-style framework Wasm suites passed with documented skips; the
rebuilt EXE passed 26 WebView2 smoke checks. Selected native UI checks passed,
including autostart/shortcut testing with restoration. The final tester fixes
still require a rebuilt EXE and visual replay; remaining manual acceptance and
release checks are open. Windows only is targeted. Wails remains beta.17 plus
fork commit `0965e9db574e7b14a447b0ae2bf7ed36a5406462`; the separate adapter is
not independently published. No v6.1.0 tag or production deployment exists yet.
See `docs/plans/v6.1-windows-api-parity.md` and its manual evidence record.
