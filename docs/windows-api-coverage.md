# Windows API coverage (pinned Wails v3 adapter)

This checklist maps the current `desktop` contract and `desktop/wails` Windows
implementation to the pinned Wails application API. “Implemented” means an
adapter exists; native WebView2/manual verification remains pending unless
called out. “Pending” means an API is not yet implemented, not that Windows
does not support it.

## Implemented portable families

| Family | Contract and Windows mapping | Status / exact remaining risk |
|---|---|---|
| File dialogs | `OpenFile`, `OpenFiles`, `OpenDirectory`, `SaveFile` → `FileDialogBackend.SelectPaths` → Wails open/save dialogs | Implemented. Picker cancellation/error and missing caller are unit-tested; real four-mode/filter/cancel smoke is pending. Wails options not represented by the contract include hidden-file display, create-directory, aliases, and button text. |
| Clipboard | `WriteClipboard`, `ReadClipboard` → `Clipboard.SetText/Text` | Implemented. Unicode/empty and real round-trip smoke pending. |
| Messages | `ShowMessage` → Wails info/question/warning/error dialogs | Implemented with Windows restrictions: info/warning/error are `Ok` only; question is `Yes`/`No`; question has no cancel button. Custom labels and arbitrary cancel/default combinations are rejected before native work. Real modal interaction remains pending. |
| Window controls | `ControlWindow` → title, screen, position/bounds, center, size constraints, minimize/maximize/restore/fullscreen, focus, topmost, resizable/frameless, menu bar, background, caption-button state, flash, content protection, zoom | Implemented broad subset. `WindowRequest` still intentionally omits URL/HTML, close, reload, force-reload, `ExecJS`, DevTools, drag APIs, native handles, and ignore-mouse controls. Caller routing and state round-trip smoke pending. |
| Printing | `PrintWindow` / `desktop.window.print` → `Window.Print` | Implemented behind separate `WindowPrinting` feature gate. Real printer/WebView smoke pending. |
| Screens | `ListScreens` now includes work area, physical bounds/work area, rotation; `ScreenGeometry` covers nearest screen and DIP/physical point/rect conversion | Implemented. Multi-monitor/DPI smoke pending. |
| Child windows | Template-restricted `CreateChildWindow`, list, inspect, show/hide/close; Wails `Window.NewWithOptions` | Implemented only when `NewNativeBackendWithWindowTemplates` is used; default backend deliberately does not advertise `ChildWindows`. Routes are same-origin root-relative templates; raw URL navigation is rejected. Two-window ownership/lifecycle smoke pending. |
| Menus/context menus | `ReplaceMenu`, install/show/remove context menu, menu-selection topic → Wails menu/context-menu APIs | Implemented. Declarative labels, enabled/hidden, separators, checkbox/radio, shortcuts, bitmap, and selection events are mapped. Wails roles are not exposed by the portable schema; verify icon/role behavior and lifecycle cleanup. |
| Tray | `ConfigureTray`, tray event topic → Wails system tray/menu | Implemented. Real tray interaction and teardown pending. Pinned Wails Windows `SetLabel` and template-icon paths are no-ops; do not claim those visual features. |
| Global shortcuts | `ConfigureShortcut`, shortcut event topic → Wails global shortcut manager | Implemented. Registration conflict/unregister/restart smoke pending. |
| Window events/drop | `SubscribeWindowEvents`, `SubscribeWindowDrops` → Wails window events; optional file-drop hook | Implemented. Verify close/navigation/resize and dropped-file payloads in a real WebView. |
| System environment | `InspectSystemEnvironment` → Wails dark-mode/accent APIs | Implemented. Theme/accent values need Windows visual verification. |
| External URLs/file manager | `OpenExternalURL`, `RevealPath` → Wails browser/File Explorer | Implemented with URL/path validation in portable layer. User-authorized real launch/reveal test pending. |
| Autostart | `Get/Enable/DisableAutostart` → Wails Windows Run-key autostart | Implemented and explicitly exposed through `Autostart` feature. This is user-authorized native state mutation; verification must inspect/restore the registration. |

## Deliberate boundaries and omissions

| Pinned Wails family | Classification | Scope decision |
|---|---|---|
| Application construction, `Run`, main loop, shutdown wiring, service registration, generated binding installation | Host-only | Native host startup/configuration; not frontend APIs. |
| Raw `ExecJS`, native handles, unrestricted URL navigation, arbitrary child-window URLs | Host-only/safety boundary | Adapter intentionally uses registered same-origin templates and caller-window context. Do not add a raw execution/navigation escape hatch. |
| Full WindowManager enumeration/current-window API | Pending gap | Child-window ownership exists, but there is no unrestricted application-wide window manager API. |
| Wails window close/reload/force-reload/HTML, DevTools, ignore mouse, drag processing, non-client hit testing, backdrop/mask, redraw internals, snap assist | Pending gap | Not in current portable contract. Production `openDevTools` is itself a deliberate no-op; dev-only behavior must not be advertised as production support. |
| Dialog hidden files/create directories/aliases/button text and custom message labels | Platform limitation | Wails exposes some knobs, but this Windows adapter rejects or omits them; document as unsupported contract fields, not generic Windows inability. |
| Wails menu roles/services and some native menu presentation | Pending/limited | Portable menu schema intentionally omits roles. Bitmap/shortcut/hidden state are mapped; role/icon/lifecycle visual verification remains. |
| Tray template icon/label behavior | No-op/platform-only | Pinned Windows backend contains no-op methods for these paths. |
| Single-instance mutex/event policy | Host-only | Process startup policy; no frontend method. |
| Updater download/install, signing, packaging, WebView2 runtime installation | Host-only | Deployment/installation responsibilities, not desktop frontend capabilities. |
| Durable storage (`storage.*`), report export | Host service | Contracts exist, but no Wails implementation in this adapter; host owns persistence/export policy. |

## Verification gate

1. Run Windows-targeted adapter tests and `go vet`; run the nested Wails module,
   not only root `go test ./desktop`.
2. Assert feature/method allowlists for default and template-enabled backends;
   especially ensure child windows are absent by default and autostart/tray/
   shortcut/menu features are explicit.
3. Perform real WebView2 smoke tests for dialogs, all picker modes, clipboard,
   window caller ownership/state, printing, screens/DPI geometry, child-window
   lifecycle, menus, tray, shortcuts, events/drop, URL/reveal, environment,
   and authorized autostart mutation.
4. Keep the omissions above explicit in release notes; do not turn runtime
   gaps or pinned no-ops into a blanket “Windows supported” claim.
