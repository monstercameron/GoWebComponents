# Windows API Lab manual verification — 2026-09-08

Status: implemented and manually exercised; not complete Windows API certification.
The operator authorized temporary control of the lab window. Tests used the real
Windows/WebView2 application and synthetic fixture files, not simulated dialogs.
Luna implementation was followed by Astra review and independent automated checks.

## Build and evidence identity

- Target: Windows amd64, on the Windows 11 ARM development machine; WebView2
  152.0.4191.66; pinned Wails v3.0.0-beta.17.
- Final executable: `examples/desktop/wails-counter/bin/wails-counter.exe`.
- SHA256: `3297271CD7C26F1636808D43DFFC89E020DFA276F0689187036C537F8D4B3932`.
- Native UI export: `bin/windows-api-manual-20260908.json` (ignored local artifact).
- Export SHA256: `7F5F7CD50D985C6B6CD882C18695B0A4CB4D6F205B32AEA344F3E74AE334454D`.
- Export contains 13 results, including one explicit observed picker verdict.
  Completed callbacks are not automatically visual passes. Export completion
  itself occurs after the saved snapshot and is not included in that snapshot.
- The export remained readable with the same hash after last-window shutdown;
  the host process exited and its owned temporary fixture directory was removed.

## Actual observations

| Surface | Observation | Build |
| --- | --- | --- |
| Custom context menu | Right-click opened native menu; command returned context=tester-surface; checkbox checkmark persisted; Radio Beta replaced Alpha; submenu callback worked; disabled item stayed inactive; Escape dismissed | Initial snapshot |
| Unicode file | Supplied sample-unicode-雪.txt selected and returned intact | Initial and final |
| Multiple files | Two checkmarks and two quoted names visible; both paths returned completed | Final |
| Picker cancellation | Native Cancel returned cancelled, not error; app usable afterward | Final |
| Directory picker | Selected synthetic fixture directory and returned correct path | Initial snapshot |
| Save-path picker | Returned JSON path; read-only filesystem check confirmed no file was written | Initial snapshot |
| Native messages | Information OK returned Ok; question No and Yes both returned completed with correct answer; app input recovered | Final |
| Fullscreen | Entire screen used, titlebar absent; Escape returned to normal window | Final |
| Maximize/restore | Native bounds changed visibly; restore returned original bounds | Final |
| Screens | One primary display; scale 1.5, logical 1920x1200, physical 2880x1800 | Final |
| F8 | Native shortcut record with window ID 1 | Final |
| Ctrl+Shift+K | Native application-menu callback appeared in report | Final |
| Standard edit context menu | Native edit menu, not custom menu; Select all selected actual input text | Final |
| Ordinary keys | Single a replaced selection once; b produced exactly ab | Final |
| Second window | Created through native Tests menu; its Unicode picker returned window ID 2; closing second window preserved primary | Final |
| Manual verdict/export | Explicit picker observed-pass and note appeared in shared report; native save dialog exported durable JSON | Final |
| Titlebar close | Closing last window removed all lab windows and terminated host | Initial and final |

Initial snapshot SHA256:
`E912B583A1C4868A2A2C25608AAE84D8E68DBE4BC1A9F2421DE8410398F251DF`.
Initial-only observations are deliberately not represented as fresh final-binary
retests. The final smoke also exercises context-target configuration, tester
routing, window RPC and report delivery, but does not visually click native menus.

## Issues found and disposition

1. **Cancel classification fixed.** Pinned upstream emits the private leaf error
   `cancelled by user`. The example now narrowly classifies that exact leaf as
   cancellation, including supported wrapping, without swallowing real failures.
   Final real picker Cancel verified the fix.
2. **Interactive wait policy fixed.** Initial multi-file inspection exceeded the
   default 30-second bridge wait; OS picker remained open and returned a cancelled
   call. Added bounded `desktop.CallWithTimeout`, preserving the ordinary 30-second
   default and earlier caller deadlines. Only tester pickers/messages/export use
   five minutes. Final multi-file selection completed. Automated native and real
   Wasm tests verify explicit/invalid/earlier deadlines and cancellation cleanup;
   a deliberate full five-minute timeout was not manually waited out.
3. **Pinned message labels corrected.** Windows uses fixed MB_OK/MB_YESNO buttons
   and callback strings Ok/Yes/No, not arbitrary custom labels. Both question paths
   and information dismissal were verified in the final binary.
4. **Bulk text duplication remains unresolved.** One computer-use bulk entry of
   `API Lab café 雪 123` appeared twice. A second bulk entry duplicated the manual
   note; this duplication is deliberately preserved in exported evidence. Native
   save-dialog filename entry was not duplicated. Discrete a/b keypresses worked.
   There is no own-code input handler on the uncontrolled keyboard test input.
   A plausible pinned-upstream path resets an accelerator's handled flag to false
   after its callback (`internal/webview2/pkg/edge/chromium.go`, AcceleratorKeyPressed),
   while the Edit role also implements paste. A duplicate paste is a hypothesis,
   not proven root cause: tool entry implementation and direct Ctrl+V with an
   approved synthetic clipboard still need isolation. No upstream patch was made.

## Explicitly unverified

Clipboard write/read and copy/paste were not explicitly exercised; replacing the
clipboard was not separately approved. Bulk entry is not an IME-composition pass.
Screen readers, tab-order certification, physical multi-display/DPI changes,
resize-to-720x520, every picker cancellation variant, export-overwrite UI flow,
and disconnected cold launch remain manual follow-ups. Export no-overwrite,
path bounds, caller resolution, and menu definitions have automated coverage;
the symlink test was skipped because this machine lacks the required privilege.

See the [tester runbook](../../examples/desktop/wails-counter/WINDOWS_API_TESTER.md)
and [independent automated verification](v6-wails-integration-verification.md).
