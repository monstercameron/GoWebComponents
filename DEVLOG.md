# Development log

## 2026-09-08 — v6.1 Windows API expansion (unreleased; paused for today)

- Expanded the Wails-free desktop contracts and separately gated Windows adapter:
  window controls/printing, screen geometry, runtime/context menus, tray, global
  shortcuts, template-owned child windows, events/drop opt-in, environment,
  external URLs, file-manager reveal, and autostart. Host-only configuration and
  unsafe execution/navigation remain outside the portable frontend contract.
- Sol/Luna handled bounded implementation and test work; Astra reviewed ownership,
  lifecycle, threading and native semantics. Resolved menu handle/bitmap/callback
  leaks, accelerator collisions, radio reselection, popup command dispatch,
  child-close races and tray callback publication. Unsupported Windows request
  values now fail explicitly rather than silently succeeding.
- With explicit authorization, created and pushed the Wails fork patch
  `0965e9db574e7b14a447b0ae2bf7ed36a5406462` to `monstercameron/wails`.
  Updated the local submodule URL/revision and build-tool pin. Astra cleared
  the patch for build/pinning; this is not release acceptance.
- Go 1.26.6 verification: root native 151 passing packages / 6,497 passing test
  events / four actual skips; CI-style framework Wasm 28 packages / 2,211 events /
  seven expected runtime skips. Native skips were two optional Atlas performance
  checks and two symlink-permission checks. Wasm excludes examples, third-party
  modules and native API-baseline tests. Counts are not coverage percentages.
- Focused desktop/build-tool, adapter, frontend Wasm, service and JavaScript
  tests passed; adapter vet passed. Fork application/Win32 tests and repeated
  ownership regressions passed. The clean pinned EXE built and passed all 26
  actual WebView2 smoke checks.
- Native UI checks verified runtime title/menu replacement, menu/context events,
  radio reselection, single/multiple-file and directory selections, save cancel,
  and an information dialog. Autostart was verified against the Run key and
  restored to no registration; the test global shortcut fired and was removed.
- Manual testing caught save-cancel wording, a child template mismatch and
  misleading success prefixes on errors. Fixes and regression tests pass; the
  running EXE predates those last tester fixes and needs rebuilding/rechecking.
- Windows Security's `gwc.test.exe` firewall prompt interrupted UI testing;
  no security setting was changed. Tray interaction remains unverified and its
  fixture remains in the running lab. Further UI checks and final release
  verification remain open. Stopped at the maintainer's request for today.
- GWC changes remain uncommitted; no v6.1 tag or production deployment was made.
  Unrelated Atlas working data and `tools/uicodegen/` remain untouched.

See [acceptance tracker](docs/plans/v6.1-windows-api-parity.md) and
[manual evidence](docs/plans/v6.1-windows-manual-verification.md).

## 2026-09-08 — v6.0.0 release preparation

- At the maintainer's direction, paused further example migration and prepared
  a `/v6` semantic-import migration, main-branch promotion, release tag and
  GitHub Pages publication. Publication is not complete until remote workflows
  succeed; this entry records preparation rather than a deployment claim.
- Luna checked the example build matrix and fresh Windows desktop smoke probes.
  Sol ran native/Wasm/hydration checks and repaired the server-interactive
  hydration defect, a WebSocket test race and chat fixture omissions.
- The uncached root native replay passed 151 packages; Wasm passed 29 packages,
  hydration five, and core race checks 19. JavaScript and doc-sample compile
  checks passed. Explicit skips and complementary checks are documented.
- The complete browser sweep found chat-example failures before work was paused.
  These remain disclosed, not reclassified as passing. Atlas's diagnostic gzip
  measurement also misses the historical size target.
- Local Atlas working data and unrelated `tools/uicodegen` work are excluded
  from the release commit. No claim of complete example migration is made.
- Refined the README to lead with web and desktop app development, separate
  quick starts, Windows-first support and optional native dependencies.
- Updated release verification to check the exact candidate rather than
  `@latest`. Go 1.26.6 removes the reachable standard-library findings observed
  with the older local toolchain; no security suppressions were added.
- Post-migration native, nested-module, Wasm/hydration, root browser and focused
  role-guard checks passed with documented conditional skips. Go 1.26.6 desktop
  smoke and production Pages/site builds passed. Astra cleared its bounded
  release-review findings; remote publication remains the next step.

See [regression verification](docs/plans/v6-regression-verification.md) and
[paused example migration](docs/plans/v6-examples-migration.md).

## 2026-09-08 — v6 Windows-first desktop and framework checkpoint

- Added a portable desktop API with an isolated Wails adapter and pinned Wails
  submodule. Web builds do not import the native adapter.
- Added desktop build-mode gates, capability ceilings and runtime checks, CLI
  build/dev/test support, and a Windows API Lab for native integrations.
- Fixed realtime lifecycle races, concurrent native SSR hook ownership, native
  state locking, and framework test-harness reliability on Windows and Wasm.
- Completed the defined framework/system verification matrix, independently
  reviewed by Astra: 4,517 native passing test/subtest results, 4,517 race results
  in the final per-package union, 2,169 Wasm results, and 1,005 browser results.
  Adapter, developer-tool, JavaScript and conditional checks also passed.
- The original broad race run had Windows executable-cleanup failures; retained
  successful package reruns resolve them. The original failed log is preserved.
  Two Windows symlink checks passed in an explicitly authorized elevated run;
  no Windows security settings were changed.

See [framework/system verification](docs/plans/v6-framework-system-tests-20260908.md)
and [independent review](docs/plans/v6-astra-framework-verification.md) for scope,
commands, evidence locations, exclusions, and conditional-test accounting.
Counts include subtests and overlap across lanes; they are not code coverage.

This is a development checkpoint, not a v6 release or a claim that every example
or native visual acceptance test passes. Application repair work remains paused;
manual native visual release acceptance remains unverified. Unrelated existing
working-tree changes and paused application edits are not part of this commit.
No module-major-version bump, release tag, or push is included.
