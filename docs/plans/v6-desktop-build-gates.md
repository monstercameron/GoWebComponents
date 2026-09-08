# Desktop build mode and native feature gates

Status: implemented and undergoing final acceptance, 2026-09-08.
Typed APIs and host policy now cover file dialogs, clipboard, messages, windows,
screens, menus, persistence and report export. `gwc_desktop` source selection,
`desktop.IsDesktopBuild()`, native target/features command plumbing and isolated
web startup are implemented. See [current evidence](v6-completion-verification.md).
The historical design discussion below distinguishes policy from source exclusion.

## Post-implementation decision

Keep the two mechanisms, but do not make the portable SDK require a desktop tag:

1. **Host-enforced runtime capability** is the normal workflow. Shared code calls
   OpenFile directly and handles unavailable; optional UI calls Supports. Require
   is useful for preflight or a larger desktop-only action, not mandatory boilerplate
   before every SDK operation. Direct native service calls are checked again.
2. **Opt-in source exclusion** is for application files that truly must not ship
   in web output. Use js/wasm + gwc_desktop for desktop frontend files and Windows
   constraints for native implementations. Runtime if-statements do not exclude
   platform imports or enforce host policy. Do not gate all of desktop behind a
   build tag: it must remain importable in web/SSR and injectable in tests.

Proof: the file host has immutable explicit opt-in; the native lab's
--file-dialogs=false rejects SDK, legacy counter and API Lab picker routes even
when called directly. Root Wasm dependency inspection contains no Wails imports.
The same resolved policy now gates menus, clipboard and export; export additionally
requires its own writing privilege and file-dialog availability.

## Findings from the current implementation

- Both web and desktop frontend code target `GOOS=js GOARCH=wasm`. A frontend
  `windows` constraint is therefore false even inside the Windows WebView.
- `desktop.Connect()` already returns a typed unavailable error without a host;
  `Client.GetCapabilities()` discovers platform, registered methods and topics.
- `gwc desktop init/doctor/build/dev/package` already exist. Build delegates to
  the isolated example's helper, which currently builds a fixed desktop bundle.
- APIService.Run multiplexes actions behind `api.run`. Removing a single
  JavaScript binding cannot disable just clipboard or file pickers.
- Native menus, keyboard bindings, CounterService.OpenFile and ExportReport are
  additional entry points. A gate only on APIService.Run would be incomplete.
- `flags` provides browser-visible rollout decisions, not native permissions.
- Root and isolated host modules already prevent native Wails imports from
  becoming an ordinary web/SSR dependency. Preserve this separation.

## Options and recommendation

| Mechanism | Useful for | Does not provide |
| --- | --- | --- |
| Go build constraints | Excluding source files and platform imports | Runtime permission checks |
| Small client capability API | Shared UI, optional controls and explicit errors | Code exclusion or host authorization |
| Host-owned feature policy | Preventing disabled native side effects | Protection from arbitrary compromised native code |
| Existing UI feature flags | UX rollout and experiments | Authority to enable native APIs |
| Linker variables | Build identity and diagnostics | General compile-time file selection |

Recommend one mode tag, small typed capabilities and host enforcement. Avoid a
separate build tag for every checkbox: that multiplies test combinations. Do not
invent function annotations that stock Go does not enforce.

## Minimal author-facing interface

### Strict desktop-only source

```go
//go:build js && wasm && gwc_desktop

package main

// renderNativeTools renders UI that is excluded from ordinary web builds.
func renderNativeTools() ui.Node { /* desktop controls */ }
```

Provide the same signature in a companion file constrained by
`js && wasm && !gwc_desktop` when shared code calls it; return web UI or no node.
Otherwise register the desktop-only route in a tagged file so no web caller
references the excluded symbol. Use `windows` only for actual native Windows
implementation files, optionally combined with `gwc_desktop`.

Name frontend files `tools_desktop.go` and `tools_web.go` with explicit tags;
`_desktop` and `_web` are not automatic Go filename constraints. Do not name a
Wasm file `tools_windows.go` merely because it calls the Windows host.

### Shared code with runtime requirements

Keep existing Connect as the explicit desktop-host requirement. Add just two
client methods plus typed feature names initially:

```go
parseClient, parseErr := desktop.Connect() // existing; requires a live bridge
if parseErr != nil {
    return parseErr
}
if parseErr = parseClient.Require(desktop.FileDialogs); parseErr != nil {
    return parseErr
}
// Perform the existing typed desktop.Call here.
```

`Client.Supports(features ...Feature) bool` is the convenience predicate for UI.
`Client.Require(features ...Feature) error` preserves detailed unavailable,
disabled, invalid-protocol or unsupported-feature errors. Zero features means
require the host itself. Unknown features fail closed. Neither helper should
silently run a fallback or swallow an operation failure. Calls still check host
policy because a frontend preflight is not authority.

Avoid `desktop.Only(func(){...})` initially: callback guards still compile imports,
obscure errors/return values, and invite hooks inside conditional callbacks.
Keep hooks at component top level; gate child components or actions instead.

## Feature policy

Start with explicit groups: file dialogs, message dialogs, clipboard read,
clipboard write, native menus, window controls, screens, persistent storage and
report export. Exact public names are subject to API review. Report export is a
separate file-writing privilege, not implicitly enabled by selecting a save path.

Effective permission is the intersection of compiled host support, local host
configuration and platform availability. UI rollout flags can further hide a
feature but cannot grant it. New projects default to no optional OS features;
the API Lab explicitly opts into its test features. Do not silently change its
current behavior during migration.

Resolve a single validated policy at host startup, advertise its effective
features, and enforce it before any native side effect. Menu callbacks, legacy
  services, generated-binding direct calls and export paths must obey it too.
Disable/unregister menus where possible and retain server-side checks. Reject
unknown feature names rather than silently accepting a typo.

Build-time feature selection initially controls the immutable host policy, not
guaranteed linker removal of every Windows implementation. Strict exclusion of
a specific implementation is a separate tagged-package requirement if needed.
This is application policy, not an OS sandbox or defense against arbitrary
malicious code already inside the native process.
It gates the host APIs described here, not browser-native user gestures such as
copy/paste in an editable field or an ordinary HTML file input. The clipboard
feature controls explicit programmatic SDK access; it is not a system clipboard
lock. Native menu composition remains an explicitly enabled host escape hatch.

## Developer commands

```text
gwc build -target web -root <app>
gwc build -target desktop -root <app> -features file-dialogs,window-controls
gwc dev -target desktop -root <app> -features none
gwc test -target desktop -root <app> -features none
gwc desktop doctor -root <app> -json
```

- Existing `gwc build/dev` defaults remain web. Existing `gwc desktop build/dev`
  remain aliases for the desktop path. Reuse existing package command semantics.
- `-features` is desktop-only; reject it for web instead of implying it grants
  browser-native privileges. `none` is explicit and mutually exclusive with names.
- The API Lab explicitly defaults to `all`; CLI feature lists set the immutable
  compiled ceiling for an invocation. Runtime `--features` may only narrow it.
  Zero-value SDK policy denies optional features. No global environment mutation.
- Web: ordinary Wasm, no desktop tag, no Wails bootstrap/imports, separate output.
- Desktop: Wasm with `gwc_desktop`; Windows host with the same mode/policy; generate
  bindings against the same policy/build inputs. Compile packages, not arbitrary
  single-file lists that bypass companion-file selection.
- Dev rebuild/restart retains selected mode/features. Packaging and JSON build
  reports include resolved target/features and validate bundle/host consistency.
- `test -target desktop` selects the opt-in native lane. The normal `all` lane
  does not silently run Windows jobs. Unsupported environments are not passes.
- Reject accidental inherited desktop mode tags on explicit web builds; keep
  host and web asset output trees separate so stale privileged bootstrap cannot
  leak into a web artifact.

## Implementation sequence and acceptance criteria

1. Add typed feature contract, client predicates/errors and host-policy resolver
   with native and real Wasm unit tests. Preserve existing Connect behavior.
2. Enforce policy across every lab service/menu/legacy/export entry point; return
   only effective capabilities. Test disabled calls cause no OS side effect,
   including direct generated binding routes bypassing UI predicates.
3. Add explicit mode tag to the build helper, paired frontend entry/registration
   files, independent web bootstrap and output paths. Test browser launch without
   Wails and desktop launch with the bridge; tagged desktop-only fixture must be
   absent from web file selection/dependency graph.
4. Add unified commands, metadata validation, help, dev restart propagation,
   package manifests and fresh-scaffold tests. Reject unknown target/features.
5. Matrix: web, desktop-none, desktop-selected, API Lab full policy, SSR/native
   contracts, missing bridge and forged frontend feature claims. Existing smoke
   must adapt expected cases to policy instead of requiring disabled APIs.

Source: installed `go help buildconstraint`, plus the official
[Go command build constraints](https://pkg.go.dev/cmd/go#hdr-Build_constraints).
