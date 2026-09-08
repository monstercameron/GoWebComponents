# Windows-first v6 completion verification

Date: 2026-09-08. Branch: `v6`. This is an execution record, not a release announcement.

## Acceptance contract

The portable SDK remains importable in web, SSR and unit-test builds without
Wails. Native implementation dependencies live in the separate `desktop/wails`
module. Shared UI uses typed client methods and `Supports`/`Require`.

Two complementary gates are retained:

- `//go:build js && wasm && gwc_desktop` excludes an application source file
  from web output; its `!gwc_desktop` companion supplies any shared signature.
- The native host intersects its compiled feature ceiling, runtime allowlist
  and backend support before performing a privileged operation. Hiding a button
  or removing a JavaScript method is not authorization.

`desktop.IsDesktopBuild()` is a compile-time target query, not a permission check.
Feature selection is application policy, not an operating-system sandbox.

## Current execution evidence

- Native build-mode tests pass with and without `-tags gwc_desktop`.
- JavaScript transport lifecycle/capability tests: 6 passed.
- Real Chromium `TestDesktopLabWebE2E` passes (15.193 seconds) against a fresh
  temporary scaffold built through `gwc build -target web`, not the helper alone.
  The `assets/web` lab renders, disables native controls, preserves Unicode
  text, navigates to the working local counter, installs no native transport,
  makes no Wails/bindings requests, and emits no uncaught page errors.
  The resulting full-page screenshot was visually inspected with no layout overflow.
  This test first reproduced the missing metadata-aware CLI dispatch and passed
  after that dispatch was corrected.
  Final rerun after the agent-mode CLI fixes also passed (9.783 seconds).
- Astra's real `TestDesktopWebDevIntegration` passed (8.657 seconds): a fresh
  scaffold serves HTML/CSS/bootstrap/Wasm without native imports or source-root
  exposure, and editing frontend Go outside the served directory changes the
  served Wasm hash. Deliberately inherited desktop tags are excluded from both
  initial build and watcher rebuild. The test reaps its owned server process tree.
  Extending this to `dev -agent` reproduced inherited-tag leakage and an idle
  WebSocket panic. Shared child-environment sanitation and context-driven socket
  closure fix those failures. The 1.2-second idle regression preserves exactly
  one connection and cancellation reaps the reader within one second; real
  ordinary and agent-mode web dev both passed (16.257 seconds combined).
  Final strengthened rerun passed (15.299 seconds including idle regression):
  watcher acceptance requires both a changed hash and the unique edited text in
  served Wasm, after initial compilation completes. No owned dev processes remain.
- Final post-fix `go test ./tools/gwc -count=1` passed (100.528 seconds), followed
  by clean `go vet ./tools/gwc`. Astra's independent report records the exact
  commands. Root module files remain unchanged and the pinned Wails tree is clean.
- Retained `bin/v6-services.test.exe -test.run=TestAPIEditMenuSingleShortcutOwner`
  passes. Clickable native Edit roles remain; WebView2 alone owns editing keys.
  This removes the duplicate handler path without modifying the pinned submodule.
- The broad root suite encountered Windows temporary executable cleanup failures
  (`Access is denied`). Targeted hookcheck rerun passed; the subsequent full gwc
  run passed (103.325 seconds). The final uncached full-module run exited 1 only
  in the SSR-oracle fixture and hookcheck CLI cleanup: their test work passed,
  then Windows denied deletion of temporary executables. The serial root rerun
  `go test -p 1 ./...` exited zero (gwc 99.627 seconds, hookcheck 2.437 seconds;
  unchanged packages reused passing cache). The later first-class web CLI fix
  receives its own focused validation and real browser rerun.

## Native artifacts and feature matrix

Fresh contributor scaffold: `bin/v6-release-20260908`.
Unsigned executable: `bin/v6-release-20260908/bin/wails-counter.exe`.
SHA256: `2c7db74f8c686debd3c5f2afa83055904f62a64aedcd0ef4f368ab84b8ed5255`.
ZIP SHA256: `f4a798f47cb45b72bfd23e7f1d3bfc8877b129db68d732c6c80a4a589c48cbad`.
Both match the generated manifest. Building standalone web assets afterward did
not change the executable or ZIP hashes. The Wails submodule remains clean/pinned.

| Actual execution | Result |
| --- | --- |
| Fresh scaffold native package, compiled `all` | Pass, native smoke required before packaging |
| Final artifact runtime `none` | Pass, 20 checks including native denial |
| Final artifact runtime `window-controls,screens` | Pass, 20 checks |
| Final artifact `--file-dialogs=false` compatibility override | Pass, 22 checks; direct legacy routes denied |
| Separate host compiled `window-controls, screens`, runtime `all` | Pass; runtime cannot add excluded features |
| Separate host compiled `none`, runtime `all` | Pass; no optional native capabilities granted |
| Missing generated binding | Expected exit 1, visible boot error and unavailable controls |
| Missing Wasm | Expected exit 1, visible boot error and unavailable controls |
| Buffered Wasm fallback | Final artifact isolated rerun passed 23 checks; an earlier concurrent-load attempt timed out and is not counted as a pass |

The native SDK smoke checks screen enumeration and malformed clipboard/message/
window requests without opening unattended dialogs or reading/writing the user's
clipboard. Positive clipboard and OS-dialog interaction are not implied by these
checks. Typed fake-backend and real Wasm contract tests cover their wire behavior.

The final integration review fixed host storage creation while disabled, false
menu capabilities, client-side UTF-8 repair before validation, editing shortcut
double ownership, the window-only Escape keymap, observer smoke state coalescing,
double-encoded native smoke arguments, mutating dev dry-run behavior, ignored
desktop CLI flags, watch target selection, and inherited build-tag leakage.

## Actual computer-use attempt

Launched the previous SDK artifact
`bin/desktop-sdk-final-20260908/bin/wails-counter.exe` using the Windows
computer-use skill. The helper returned the lab accessibility tree but a black
screenshot. Input activation failed with `failed to activate captured window`.
Fresh window discovery and one activation retry returned the same error.
Per the skill's recovery boundary, UI input stopped. No current-build visual
pass is claimed from this attempt. An unlocked, targetable interactive Windows
session is required to finish the requested visual acceptance.

Existing manual observations, including their exact older build identities,
remain in `v6-wails-windows-api-manual.md`; they are not fresh final-build tests.
