# Windows desktop compatibility

Status: experimental Windows-first integration, 2026-09-08. The verified target
is Windows amd64 on this Windows 11 ARM machine, Go 1.26.3 and WebView2
152.0.4191.66, with Wails v3.0.0-beta.17 pinned as a source submodule. This is
not a Windows ARM64 binary certification or a macOS/Linux support claim.

## Evidence and remaining acceptance

| Capability | Current evidence | Release limitation |
| --- | --- | --- |
| Bundled Wasm and CSS | Actual native smoke renders and interacts with the example; MIME and buffered fallback checked | Disconnected cold launch still needs manual verification |
| Hash routing | Repeated Counter/About unmount/remount passes | Other application routes require their own coverage |
| Native services | Typed success/error, handshake and cooperative cancellation pass | Native methods must still validate application-specific inputs |
| Native events | Progress and bounded subscription cleanup pass | Latest-value delivery is not a lossless event log |
| Independent windows | Actual two-window smoke shares saved state while counters remain local | General multi-window stress and lifecycle certification remains open |
| Durable state | Native bounded JSON service, version guards and process-reopen tests | Not a multi-process database; one host owns the file lock |
| File picker | Real single/Unicode/multiple selection, Cancel and second-window caller verified in API Lab | Directory/save-path observed on earlier snapshot; remaining cancellation variants pending |
| Keyboard and focus | Native F8, Ctrl+Shift+K, standard edit context menu and message focus recovery observed | Bulk text entry duplication unresolved; tab order and direct paste pending |
| Accessibility and IME | Standard HTML controls used | Screen-reader and IME checks pending |
| Fonts, DPI and resize | Real screen enumeration at 150%; maximize/restore/fullscreen/Escape observed | Physical scaling, font fallback and fixed-size resize checks pending |
| Clipboard and external links | No desktop-specific support assertion | Explicit application policy and native tests needed |
| Workers and browser storage | No native compatibility assertion | Probe independently; desktop saved state uses native services |
| Content policy | Native CSP smoke and exact-host/origin/referrer guard tests pass | Full hostile-navigation certification pending; privileged-document JS remains privileged |
| Packaging | Unsigned executable/ZIP workflow with manifest and hashes | No signed installer, update policy or clean-machine certification |
| Shutdown | Automated smoke and manual last-titlebar-close exit; fixture cleanup and durable export verified | WebView2 warning 1412 is retained in evidence |

PWA installation and service workers are not required for desktop startup.
Desktop support does not change existing browser/SSR defaults. A Linux portable
Go race-test workflow is not a Linux native-WebView support claim, and a checked-in
workflow is not evidence of a successful hosted CI run.

## Manual Windows acceptance recipe

Build using the [desktop reference](../REFERENCE_MANUAL/16-windows-desktop.md),
then run the executable with `--two-windows`:

1. Confirm local increments affect only their own window and saved increments
   appear in both. Close both windows, reopen, and confirm saved state survives.
2. Open a file, select a harmless file, then reopen the picker and cancel. Confirm
   the app remains usable and the dialog belongs to the invoking window.
3. Navigate by keyboard, resize both windows, change display scaling, and check
   visible focus, labels and layout. Test the intended IME and screen reader.
4. Start native work, cancel it, start again and change route. Confirm no late UI
   update. Close using the title bar and confirm the host exits.
5. Disconnect the network after dependency installation/build, launch the packaged
   executable, and repeat core interactions. Reconnect after testing.

Record the Windows build, WebView2 version, binary hash and each result. Do not
check the corresponding release TODOs merely because this recipe exists.

See [adapter evidence](v6-wails-adapter-verification.md),
[Windows API manual evidence](v6-wails-windows-api-manual.md),
[final integration verification](v6-wails-integration-verification.md),
[initial native evidence](v6-wails-verification.md) and the
[authoritative backlog](../../todos.md#v6-wails-desktop-integration).
