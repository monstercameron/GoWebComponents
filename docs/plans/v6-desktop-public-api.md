# Desktop public API implementation checkpoint

Implemented (2026-09-08): file dialogs, explicit text clipboard operations,
native information/questions, caller-owned window controls and screen enumeration
use the replaceable adapter boundary. These are narrow common workflows, not a
mirror of every Wails API. Native host composition remains an explicit escape hatch.
Current completion evidence is in [v6 verification](v6-completion-verification.md).

1. Add Wails-free options/results, a backend interface, an explicit immutable
   host opt-in and typed frontend workflows for open one/many, folder and save path.
2. Implement the backend in a separately versionable `desktop/wails` Go module.
   Keep the current pinned module version; local replacements are contributor-only.
3. Register the reusable host in the real example, consume it through the SDK,
   and preserve the lab's native observations and legacy binding compatibility.
4. Test portable contracts with a fake backend/transport, native adapter failure
   cases, real Wasm, isolated host build and native WebView smoke.
5. Revisit gates after evidence: compile-time file selection is independent of
   host-enforced feature availability; do not promise that hiding UI gates APIs.

Do not expose Wails types in shared application signatures. Do not wrap unrelated
Wails APIs speculatively. Preserve existing web/SSR behavior and uncommitted work.

## Implementation

- `desktop/files.go`: portable options/results; OpenFile, OpenFiles,
  OpenDirectory and SaveFile (path-only), both convenience functions and injected
  client methods. Supports/Require derive effective file support from the host.
- `FileDialogHost`: immutable explicit opt-in, input/output bounds, portable
  errors and direct-binding enforcement. Default/nil/disabled hosts deny access.
- `desktop/wails`: isolated Go module with pinned Windows implementation,
  caller-window ownership and a narrow upstream cancellation compatibility shim.
  Unsupported platforms return no backend and advertise no support.
- Counter frontend uses the typed API. Legacy CounterService and four lab picker
  actions share the same host. `--file-dialogs=false` disables all these paths.
- Scaffold replacements and generated bootstrap mapping include the adapter.
  SDK binding imports use the existing visible boot-error path, not a top-level
  import failure that would prevent boot diagnostics from running.

## Verification

Commands run with process-local GOOS/GOARCH; no persisted toolchain settings changed:

- Root Windows `go test ./desktop ./tools/gwc -count=1` and vet: pass
  (full CLI suite 100.234s).
- Root real Wasm `go test -exec tools/go_js_wasm_exec.bat ./desktop -count=1`:
  pass, including SDK calls through actual desktop.js and Unicode result decoding.
- Adapter module Windows tests/vet and real Wasm unsupported-platform test: pass.
- Example native services/host tests/vet and real Wasm frontend tests: pass.
- Scoped desktop native/Wasm and adapter native golangci-lint: pass.
- `go list -deps ./desktop` under js/wasm: no Wails or adapter dependencies.
- Fresh CLI init/build: pass, including absolute contributor replacements.
- Native WebView smoke: enabled 21 checks; disabled 21 checks, including direct
  SDK, counter and API Lab bypass attempts; streaming fallback 22 checks.
- Missing-binding fault: expected exit 1 with visible-boot-error and
  native-controls-unavailable (checked before the final filter-validation-only change).
- git diff --check: pass; root go.mod/go.sum and Wails submodule source untouched.

Final verification artifact:
`bin/desktop-sdk-final-20260908/bin/wails-counter.exe`.
SHA256 `C5CE9AB252F7F54DAB18183A8122CF8A8C75A0BE84453211890DE16CFFF02113`.
Windows amd64, Go 1.26.3, Wails beta.17, WebView2 152.0.4191.66.
The known WebView2 shutdown warning 1412 remains visible; smoke exits zero.

Native smoke deliberately does not open interactive dialogs or touch clipboard.
The positive picker mapping uses the same pinned Wails APIs previously manually
exercised, but this extraction has not received a new manual dialog-selection pass.
The previously open example executable was not overwritten or closed; the fresh
verification artifact is separate. Close the old host before launching this one
normally because the demo's persistent store is single-host locked.

## Gate decision after implementation

Use host-enforced runtime capabilities for ordinary portable code. A direct
OpenFile call already checks availability; Require is optional preflight and
Supports controls UX. Do not require redundant checks at every call site.

Retain `gwc_desktop` as the proposed opt-in **application source** exclusion
mechanism, not as a requirement for importing the SDK. The build/dev target flags
and full feature manifest remain unimplemented; see the revised
[gate proposal](v6-desktop-build-gates.md). A runtime predicate cannot exclude
imports, and a build tag cannot authorize a native operation.

Residuals: adapter release versioning is not published; broader native capability
policy and the prior bulk-entry duplication issue remain open. This checkpoint
does not certify all Wails APIs or a production desktop release.
