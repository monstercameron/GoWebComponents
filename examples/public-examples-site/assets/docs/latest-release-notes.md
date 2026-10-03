# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

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
