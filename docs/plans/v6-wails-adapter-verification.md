# V6 Wails D2 adapter verification

Recorded 2026-09-08 by Astra after the initial adapter implementation. This
extends [the D1 evidence](v6-wails-verification.md); the unresolved OS interaction
checks from that record remain unresolved. It does not certify D3/D4 work being
implemented concurrently.

## Verified behavior

The optional `desktop` package exposes an injectable synchronous transport,
typed JSON calls, a protocol/capability check, context cancellation/deadlines,
and latest-value native event subscriptions. Its native Connect stub reports
unavailable. Both native and js/wasm dependency inventories contain no Wails
package. Native Wails services stay inside the isolated example module.

The external embedded desktop.js adapter owns the original cancellable request
handle. Go polls JSON envelopes and never gives a Go js.Func to an outstanding
service promise. Context cancellation releases the request and invokes the
pinned Wails handle's cancel method. The actual native smoke separately reads
host Active/Cancelled counters after a UI cancel and after route unmount; both
operations stop RunWork in the host, rather than merely stop waiting in the UI.

Default Go call timeout is 30 seconds; a shorter caller deadline wins. Each
transport caps pending request and subscription registries at 256 by default
(configurable from 1 to 4096). Event bursts keep one latest pending payload per
subscription. Tests inspect registry counts after cancellation, late resolution,
late rejection, close, quotas, unsubscribe, and reentrant host callbacks.

## Defects found and corrected

| Finding | Correction / regression evidence |
| --- | --- |
| Cancellation triggered inside Poll could still return its completed value. The new native regression returned nil error before the fix. | Check context again immediately after Poll and before starting transport work. Both native and Wasm common tests pass. |
| Numeric arguments above JavaScript's safe range reached Start and silently lost precision. The new native regression observed the unsafe payload reaching the transport. | Scan marshaled argument numbers before crossing JS; return encode error for unsafe magnitudes. JS response/event serialization rejects non-finite and unsafe integer values with decode error. Wide identifiers remain strings. |
| A rejected object with a throwing message getter caused an unhandled promise rejection that terminated the Node Wasm test process. | Error normalization handles getters/coercion failures and always returns a diagnostic string. Rejection and synchronous exceptions now remain typed remote errors. |
| Inspection found that a service could close the transport synchronously, after which start still inserted a request. Event registration could similarly close before returning its unsubscribe handle. | Recheck closed state after both host calls; consume/cancel the returned promise or call the newly returned unsubscribe without retaining a registry entry. Dedicated real-JS tests pass. |
| A throwing cancel-property getter could interrupt bulk cleanup. | Access and invocation are both guarded; owned entries are removed before attempting cancellation. Other requests/listeners still close. |
| User event-handler panic escaped the subscription goroutine. | Use the existing framework panic-containment boundary with diagnostics; unsubscribe still runs. Tests deliberately log GWC-RUNTIME-PANIC-ASYNC and confirm release. |
| Initial general contract tests were native-only; the event test cancelled before observing any event. | Common tests now run on both targets. Real-module Wasm tests observe delivery, 100-event burst coalescing, typed decode failure, and no delivery after unsubscribe. |
| The native success reporter required only the original six D1 checks. | Require all twelve D2 checks, including backend cancellation, unmount cancellation, adapter cleanup, MIME and CSP. |

Legacy uncancellable wrapper exports were removed from the example bootstrap;
the UI now consistently uses the desktop adapter. README documents cancellation,
coalescing, numeric constraints and remaining manual gates.

## Validation commands and results

Repository-root commands unless marked example module. Target overrides below
were set only in individual PowerShell processes; no persisted Go setting was
changed. Environment remains Go 1.26.3, Windows amd64 execution on ARM hardware,
CGO disabled; WebView2 152.0.4191.66. Node Wasm tests use Node 26.2.0.

| Command | Result |
| --- | --- |
| `go test ./desktop -count=1` | Final focused rerun exit 0. General call/event tests also execute on Wasm. Concurrent storage tests ran incidentally; their presence is not D4 acceptance. |
| `$env:GOOS='js'; $env:GOARCH='wasm'; $env:CGO_ENABLED='0'; go test -exec C:/Users/mreca/Desktop/GoWebComponents/tools/go_js_wasm_exec.bat ./desktop -count=1 -v` | Exit 0. Uses the actual embedded ES module, real JS promises, typed wire round trips, missing/malformed bootstrap, remote rejection/throw, cyclic/unsafe responses, hostile error getters, cancel-before-call, in-flight cancellation, deadlines, never-settling calls, late resolve/reject, request/subscription quotas, close, reentrant close, event burst/decode/unsubscribe, local-topic separation, and cleanup exceptions. |
| `go vet ./desktop` on native and js/wasm | Exit 0. |
| `go list -deps ./desktop` on native and js/wasm, filtered for wailsapp | No Wails dependencies on either target. |
| Example module `go test ./...` | Exit 0. |
| Example module `go vet ./...` | Exit 0. |
| Example module `go test ./internal/services -run '^TestRunWork' -count=1 -v` | Exit 0. Work becomes active, cancellation terminates it within the bounded test, and final Active=0/Cancelled=1. Nil, pre-cancelled and expired contexts do not start work. |
| Example module `go run ./tools/build` | Exit 0, generates pinned bindings and embeds the reusable desktop.js source. |
| Example module `bin/wails-counter.exe --smoke-test` | Exit 0 with all twelve required checks. |
| Example module `bin/wails-counter.exe --smoke-test --smoke-fault streaming` | Exit 0 with the same twelve checks plus wasm-fallback. |
| Example module missing-binding and missing-wasm smoke faults | Both expected exit 1; persistent visible-boot-error and native-controls-unavailable evidence retained. |
| `git diff --check` | Exit 0; normal Windows LF/CRLF advisory messages only. |

Actual native normal smoke report:

```json
{"ok":true,"checks":["dom","local-counter","native-call","native-error","native-event","backend-cancellation","unmount-cancellation","adapter-cleanup","routing","route-remount","wasm-mime","csp"],"error":""}
```

## Limits and retained diagnostics

- The first new native precision/cancellation tests failed before their fixes.
  The first strengthened Wasm run terminated on the hostile Error.message
  getter; the corrected run passed. These failures were not weakened away.
- One native test run printed package PASS but then hit the previously observed
  Windows temporary desktop.test.exe unlink access denial. A separate final
  `go test ./desktop -count=1` exited 0.
- Direct js/wasm golangci-lint initially found an unchecked deferred Cancel
  result (corrected as explicitly best-effort cleanup) and an always-false nil
  comparison in concurrently authored storage_test.go. The latter was reported
  to its owner and corrected there. The final direct js/wasm
  `golangci-lint run --timeout 60s ./desktop` exited 0 with no issues.
- The package uses the existing framework's structured panic diagnostic rather
  than concealing handler failures. The intentional panic test therefore emits
  a diagnostic even though its cleanup assertions pass.
- WebView2 continues to emit shutdown unregister-class warning 1412 despite
  correct success/failure exit codes. Host and fallback invocations terminate;
  no interactive title-bar close claim is inferred from programmatic shutdown.
- Capabilities describe the installed JS protocol and explicit application
  allowlist. The configured hostVersion string is not an authenticated native
  version handshake or an authorization boundary. Applications must explicitly
  register/validate their privileged host methods.
- Bounded resources here mean adapter-owned registries, latest-event queues,
  Go polling goroutines and absence of Go callbacks retained by promises.
  JavaScript cannot detach a then-handler from an externally retained promise.
  A hostile host that retains arbitrarily many never-settling promises can retain
  their JS continuations; this is not a measured total-heap bound. Late settlement
  does not restore registry state or call released Go callbacks.
- Cancellation is cooperative. It cannot undo completed writes, force an OS
  picker to close, or guarantee termination of host code that ignores context.
- Real picker selection/cancellation, physical keyboard/IME, resize, DPI,
  disconnected startup and title-bar close remain manual. No macOS/Linux or
  packaging/signing claim is added by these Windows tests.

Technical D2 acceptance is established for unavailable behavior, typed calls and
errors, cooperative native cancellation, bounded owned cleanup and event lifecycle.
Root backlog dependency checkboxes still depend on the outstanding D1 OS gates;
the user authorized progressing independent implementation without pretending
those gates had passed.
