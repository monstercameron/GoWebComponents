# Development log

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
