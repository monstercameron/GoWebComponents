# Windows API coverage (pinned Wails v3.1.0 adapter)

This checklist maps the current `desktop` contract and `desktop/wails` Windows
implementation to the pinned Wails `v3.1.0` application API (forked on
`monstercameron/wails` branch `gwc/v3.1.0` with scoped menu-ownership and
popup-command fixes). “Implemented” means an adapter exists; native
WebView2/manual verification remains pending unless called out. “Pending” means
an API is not yet implemented, not that Windows does not support it.

## Implemented portable families

| Family | Contract and Windows mapping | Status / exact remaining risk |
|---|---|---|
| File dialogs | `OpenFile`, `OpenFiles`, `OpenDirectory`, `OpenDirectories`, `SaveFile` → `FileDialogBackend.SelectPaths` → Wails v3.1.0 open/save dialogs (including multi-folder `NewSelectMultipleFoldersDialog`) | Implemented. Picker cancellation/error and missing caller are unit-tested; real five-mode/filter/cancel smoke is pending. Wails Windows CFD does not expose `ShowHiddenFiles` or `ButtonText`; both are rejected explicitly rather than silently dropped. |
| Clipboard | `WriteClipboard`, `ReadClipboard` → `Clipboard.SetText/Text` | Implemented. Unicode/empty and real round-trip smoke pending. |
| Messages | `ShowMessage` → Wails v3.1.0 info/warning/error (`MessageBox`, `Ok` only) and `question` (`TaskDialogIndirect` with 1–3 custom buttons plus arbitrary present default/cancel roles) | Implemented. Custom question button labels, defaults, and cancel roles are validated before native work. Info/warning/error remain `Ok`-only to match Windows `MessageBox`. Real modal interaction remains pending. |
| Window controls | `ControlWindow` → `info`, `show`, `hide`, `close`, `reload`, `snap-assist`, title, screen, position/bounds, center, size constraints, minimize/maximize/restore/fullscreen, focus, topmost, resizable/frameless, menu bar, background color, caption-button state, flash, content protection, `set-ignore-mouse-events`, `set-enabled`, full `[0.25, 5.0]` zoom (including sub-1.0 zoom in v3.1.0); `WindowInfo` includes `ignoreMouseEvents`, `screenId`, and `borderSizes` | Implemented broad caller-owned surface. `force-reload` is explicitly rejected as `unavailable` because Wails Windows `forceReload` is a no-op; `set-fullscreen-button-state` likewise returns `unavailable`. Raw `ExecJS`, `SetURL`, `SetHTML`, DevTools, and native `HWND` handles remain intentionally excluded. |
| Printing | `PrintWindow` / `desktop.window.print` → `Window.Print` | Implemented behind separate `WindowPrinting` feature gate. Real printer/WebView smoke pending. |
| Screens | `ListScreens` includes work area, physical bounds/work area, rotation; `ScreenGeometry` covers nearest screen and DIP/physical point/rect conversion | Implemented. Multi-monitor/DPI smoke pending. |
| Child windows | Template-restricted `CreateChildWindow`, list, inspect, and bounded `ControlChildWindow` (`show`, `hide`, `close`, `focus`, `minimize`, `unminimize`, `maximize`, `unmaximize`, `restore`, `center`, `resize`, `set-title`, `set-bounds`); host `WindowTemplate` supports size constraints, `Frameless`, `AlwaysOnTop`, and DWM `Backdrop` (`none`/`auto`/`mica`/`acrylic`/`tabbed`) | Implemented only when `NewNativeBackendWithWindowTemplates` is used; default backend deliberately does not advertise `ChildWindows`. Routes remain same-origin root-relative templates; raw URL navigation is rejected. Two-window ownership/lifecycle smoke pending. |
| Menus/context menus | `ReplaceMenu`, install/show/remove context menu, menu-selection topic → Wails menu/context-menu APIs | Implemented with per-window accelerator/HMENU/HBITMAP ownership and DPI-scaled client coordinates. Wails roles are not exposed by the portable schema; verify icon/lifecycle cleanup visually. |
| Tray | `ConfigureTray` (`upsert` with in-place updates, `show`, `hide`, `position-window`, `remove`), tray event topic → Wails v3.1.0 system tray (`SetIcon`, `SetDarkModeIcon`, PNG + ICO support, `SetLabel` tooltip fallback, `SetTooltip`, `PositionWindow`) | Implemented. Real tray interaction, taskbar theme switching, and teardown smoke pending. Template-icon and icon-position remain no-ops on Windows and are not advertised. |
| Global shortcuts | `ConfigureShortcut`, shortcut event topic → Wails global shortcut manager | Implemented. Registration conflict/unregister/restart smoke pending. |
| Window events/drop | `SubscribeWindowEvents`, `SubscribeWindowDrops` → Wails v3.1.0 window events (`focused`, `blurred`, `moved`, `resized`, `shown`, `hidden`, `fullscreen`, `unfullscreen`, `minimized`, `unminimized`, `maximized`, `unmaximized`, `restored`, `dpi-changed`, `zoom-changed`, `closing`); optional file-drop hook | Implemented. Verify close/resize/DPI/zoom and dropped-file payloads in a real WebView. |
| System environment | `InspectSystemEnvironment` → Wails dark-mode, accent color, and OS metadata (`osName`, `osVersion`, `osBranding` from `Env.Info()`) | Implemented. Theme/accent values need Windows visual verification. |
| External URLs/file manager | `OpenExternalURL`, `RevealPath` → Wails browser/File Explorer | Implemented with URL/path validation in portable layer. User-authorized real launch/reveal test pending. |
| Autostart | `Get/Enable/DisableAutostart` → Wails Windows Run-key autostart | Implemented and explicitly exposed through `Autostart` feature. This is user-authorized native state mutation; verification must inspect/restore the registration. |

## Deliberate boundaries and omissions

| Pinned Wails family | Classification | Scope decision |
|---|---|---|
| Application construction, `Run`, main loop, shutdown wiring, service registration, generated binding installation | Host-only | Native host startup/configuration; not frontend APIs. |
| Raw `ExecJS`, native `HWND` handles, unrestricted `SetURL`/`SetHTML`, arbitrary child-window URLs | Host-only/safety boundary | Adapter intentionally uses registered same-origin templates and caller-window context. Do not add a raw execution/navigation escape hatch. |
| Production `OpenDevTools` | Host-only/dev-only | Dev-only behavior must not be advertised as a production frontend capability. |
| Windows `forceReload`, `setFullscreenButtonState`, tray `SetTemplateIcon`/`SetIconPosition` | Windows no-op in Wails v3.1.0 | Adapter explicitly returns `CodeUnavailable` or omits these fields so Windows capability claims stay honest. |
| Unrestricted application-wide WindowManager enumeration | Safety boundary | Caller-owned child windows provide full bounded control without letting one window enumerate or mutate unrelated windows. |
| File dialog `ShowHiddenFiles` and `ButtonText` | Platform limitation | Wails Windows CFD does not expose these knobs; adapter rejects them with `CodeInvalid` instead of silently ignoring caller intent. |
| Single-instance mutex/event policy | Host-only | Process startup policy; no frontend method. |
| Updater download/install, signing, packaging, WebView2 runtime installation | Host-only | Deployment/installation responsibilities, not desktop frontend capabilities. |
| Durable storage (`storage.*`), report export | Host service | Contracts exist in `desktop`; host owns persistence and export policy (demonstrated in `examples/desktop/wails-counter`). |

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
