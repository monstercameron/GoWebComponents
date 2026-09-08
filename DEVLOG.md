# Development log

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
