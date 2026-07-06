# Code Audit Devnotes — file-by-file bug/perf pass (2026-07-05 →)

Granular module-by-module audit. Each section lists what was found, what was
fixed, and what was deliberately left alone (with the reason), so a later
reader can tell "reviewed and clean" apart from "not yet reviewed".

## Module 1: internal/platform (jsdom + mockdom) — DONE

### Fixed
- **jsdom/adapters.go — template fast path skipped URL sanitization**
  (security). `buildHostElementHTMLInto` HTML-escaped attribute values but
  never ran them through `runtime.SanitizeURLAttributeValue`, so a
  `href="javascript:..."` survived on exactly the lane whose comment promised
  sanitizer parity with `SetAttribute`/SSR. Now sanitized before escaping.
  Pinned by `TestBuildHostElementHTMLSanitizesURLAttributes` (wasm).
- **jsdom/adapters.go — `CreatePreparedElement` returned a null node when the
  template parse dropped the element** (html/head/body/frame are removed from
  template content by the HTML parser). Now falls through to the direct
  `createElement` + `BatchSetAttributes` path instead.
- **jsdom/adapters.go — localStorage exceptions crashed the app.**
  `setItem` throws on quota-exceeded / private-mode; syscall/js surfaces that
  as a Go panic. `SetItem` now recovers and returns the error it always
  promised; `GetItem` treats a throw as absent; `RemoveItem` is best-effort.
- **jsdom/attr_batch.go — JS-side node registry grew without bound.** One
  WeakRef slot per ever-registered node, never recycled → slow leak in
  long-lived apps with churning lists. Added a FinalizationRegistry-backed id
  free-list (feature-detected; old browsers keep prior behavior). Flush loop
  already tolerated null slots.
- **mockdom/dom.go — three browser-fidelity divergences** (this package's
  contract is "trustworthy stand-in"):
  - `GetParent` returned a typed-nil `*MockDOMNode` in the interface (jsdom
    returns untyped nil) → `== nil` checks behaved differently native vs browser.
  - `InsertBefore` detached the new node BEFORE checking the reference was a
    child; a bad ref lost the node entirely. Now guards first and no-ops,
    matching jsdom.
  - `SetTextContent` kept existing children; the browser's textContent setter
    replaces all children, so later `textContent` reads concatenated stale
    descendants. Now detaches children.
  - Also: `AssertOperation` now rejects negative indexes.

### Reviewed, left alone (reasons)
- `WrapFunction` default case returns a no-op js.Func for unknown handler
  types — silent, but changing it to panic/report is a behavior change beyond
  this pass's scope.
- `appendDOMChildren` probes the multi-arg `append()` capability once globally;
  in principle a first probe against a non-ParentNode would mis-cache, but the
  reconciler only passes elements/fragments as parents.
- mockdom `QuerySelector` scans `nodeMap` in map order (nondeterministic on
  multiple matches) — documented as the test-subset selector; tests use ids.
- mockdom `SetInnerHTML` only clears children for the empty string — documented
  mock limitation; non-empty innerHTML is not parsed.
- mockdom subtree.go parses fragments in a `div` context, so table-scoped tags
  (`tr`, `td`) unwrap differently than the browser template element. The
  runtime's serializer gates which tags reach this path; revisit only if
  serialized mounts ever include table internals.
- MockScheduler `FlushAll` can spin if a goroutine perpetually reschedules —
  acceptable in a test scheduler.

### Verification
- `go test ./internal/platform/... -count=1` — ok
- `GOOS=js GOARCH=wasm go test ./internal/platform/jsdom/` under node — ok
- Dependent suites re-run: `./internal/runtime/...`, `./test/render/...` — ok

## Module 2: html (+ html/shorthand) — DONE

### Fixed
- **html.go — unbounded attr-name intern caches.** `dataAttrNameCache` /
  `ariaAttrNameCache` intern user-supplied data-/aria- attribute NAMES with no
  size bound; an app generating dynamic names (`data-idx-<n>`) leaked entries
  forever. Now a size-capped table (4096 entries via LoadOrStore + atomic
  counter); past the cap new names just pay the concat.
- **shorthand.go — dead code.** Local `mergeStringMap` (and its `maps` import)
  was unreferenced since mergeProps started delegating to html.MergeProps.
- **shorthand_more_test.go — wasm-illegal test (pre-existing FAIL).**
  `TestEventHandlerReexportsEmitNativePropsAndPassive` constructs On* handlers
  outside a component; ui.UseEvent is a hook and panics on the wasm build
  (native stubs masked it — the package had never run green under wasm).
  RenderToString can't host it either in the bare-node wasm env (no DOM
  adapter). The emission logic is platform-independent → test now skips on
  GOOS=js with the rationale in a comment; native carries the coverage.
- **markdown_fuzz_test.go / rawhtml_fuzz_test.go — wasm-on-Windows FAIL
  (pre-existing).** Seed-corpus loading dies under the node runner
  (`O_DIRECTORY is not supported on Windows`); fuzzing is unsupported on
  js/wasm anyway. Both files now build-constrained `!(js && wasm)`.

### Reviewed, left alone (reasons)
- `runtimeEventProps` scans all 30 handler fields per Tag call — survived the
  perf loops with profiling; single-scan design is deliberate.
- `Debounce`/`Throttle` create fresh timer state per render — documented API
  shape; changing it is a design decision, not a cleanup.
- `Repeat` aliases the same node pointer N times — documented semantics.
- markdown.go scheme sanitizer verified against obfuscation (embedded control
  chars, mixed case) — already fuzz-tested; image nodes reuse Classes.Link
  (cosmetic, no Image class exists).
- raw_html.go: attribute values flow into Raw and are URL-sanitized at the
  adapter layer; RawHTMLUnsafe is deliberately alarming and greppable.

### Verification
- `go test ./html/... -count=1` — ok (native)
- `GOOS=js GOARCH=wasm go test ./html/...` under node — ok (first time this
  package is fully green under wasm)
- Dependents re-run: `./test/render/...`, `./test/hooks/...`,
  `./internal/runtime/...` — ok

## Module 3: internal/runtime — chunk 1 of ~6 (small core files) — DONE

Files reviewed: state.go, events.go, transition.go, error_boundary.go,
async_boundary.go, context.go, signature.go, memory_hygiene.go,
panic_guard.go, component_type.go, types.go, interfaces.go.

### Fixed
- **component_type.go — lost-update race in `SetImplementationRenderer`.**
  The load-modify-store on the `atomic.Value` let two concurrent swaps drop
  one update. Added a writer-side mutex (`setMu`); `Render` stays lock-free.

### Reviewed, left alone (reasons)
- state.go: generation-guarded derived recompute and notify-dedup verified
  correct; `RestoreSnapshot` not recomputing derived atoms is fine because a
  snapshot already contains the derived values as an internally consistent set.
- `markSubtreeNeedsUpdate` walks siblings by design; its single call site
  passes `.child`, so semantics are correct.
- `subscribeAsyncBoundary` goroutine blocks forever if a Suspension's Done
  never closes — leak-by-contract on the caller.
- ssr.go:791 "\ Finding #57" seen in an earlier grep was display mangling; the
  source is a normal `//` comment.

### Verification
- `go test ./internal/runtime/... -count=1` — ok; wasm build ok;
  `./test/render/...` — ok

## Module 3: internal/runtime — chunk 2 (SSR + hydration) — DONE

Files reviewed: ssr.go, ssr_stream.go, hydration.go, hydration_metrics.go.

### Fixed
- **hydration.go — browser hydration silently discarded server DOM (MAJOR).**
  The browser adapter's GetProperty returns raw `js.Value`; hydration's
  `normalizeHydrationInt` has no js.Value case, so `domNodeType` returned 0 in
  a real browser, the first node match failed, and every hydration pass fell
  back to full client rendering — while native tests passed because mockdom
  returns Go ints. Also: absent properties (js undefined) stringified to
  garbage, producing false attribute-mismatch diagnostics. Fix: build-tag pair
  `normalizeForeignPropertyValue` (wasm unwraps string/number/bool,
  undefined/null → nil; native no-op) applied at hydration's four GetProperty
  read sites. Pinned by `TestHydrationNormalizesJSValueProperties` (wasm).
  Follow-up task #37: browser e2e asserting FallbackCount==0 — the coverage
  hole that let this ship.
- **ssr.go — error boundary leaked partial markup on child panic.** Children
  rendered directly into the output builder, so a panic mid-subtree left
  unclosed tags ahead of the fallback (the async boundary already used a
  scratch builder). Pinned by
  `TestRenderToStringErrorBoundaryDoesNotLeakPartialMarkupOnPanic`.
- **ssr_stream.go — same partial-markup leak in the stream shell, plus
  orphaned pending boundaries.** Scratch builder + rollback of
  `pending`/`nextBoundaryID` registered by discarded children (their markers
  died with the scratch output; streaming their patches targeted nothing).
  Pinned by `TestRenderToStreamErrorBoundaryDoesNotLeakPartialMarkupOnPanic`.

### Reviewed, left alone (reasons)
- SSR attr-name validation, URL-scheme sanitizer (narrow denylist; data: is
  deliberately allowed for img src), select/textarea controlled-value
  serialization — verified correct; compact options can never carry a `value`
  attribute (the compact lane disqualifies Value), so optionMatchValue's text
  fallback is sound.
- Stream boundary patch script: boundary id is json.Marshal-ed with Go's
  default HTML escaping, so `</script>` breakout via a hostile prefix is not
  possible.
- Nested suspension inside a resolved boundary chunk renders its fallback
  permanently (no recursive streaming) — known design limitation.

### Verification
- `go test ./internal/runtime/... ./test/render/... ./test/ssr/... -count=1` — ok
- Full wasm runtime suite under node — ok

## Module 3: internal/runtime — chunk 3a (reconciler_elements.go) — DONE

### Fixed
- **cloneChildFibers dropped `fiber.key` (and `portalUnresolved`) in the
  bailout clone (MAJOR).** The subtree-dirty pass-through path (clean ancestor,
  dirty descendant) rewrites each child fiber via a struct literal that omitted
  `key`. Fast-lane keyed rows carry their key ONLY on that field (props is
  nil), and the keyed reconciler treats key==""+props==nil as unkeyed — so any
  deep update inside a keyed list destroyed row identity for the NEXT reorder:
  DOM nodes were positionally reused, moving input state/focus/animations onto
  the wrong rows. Map-lane rows were immune (key lives in the copied props
  map), which is why existing keyed tests never caught it. Pinned by
  `TestKeyedFastLaneRowsSurviveBailoutCloneThenReorder` (verified red without
  the fix, green with it). `portalUnresolved` (retry-on-next-commit) now also
  survives the clone.

### Framework sharp edge confirmed while writing the pin test
- Loop-created CLOSURE components all resolve to one per-code-pointer handle
  whose implementation is the last closure registered — all rows render the
  final iteration's captured environment. This is the documented G1 gotcha
  (MapKeyedComponent is the sanctioned pattern), but the failure is silent;
  worth a future diagnostic when a handle's implementation is swapped
  mid-render-pass.

### Verification
- `./internal/runtime/... ./test/render/... ./test/hooks/...` — ok; wasm build ok

## Module 3: internal/runtime — chunk 3b (runtime.go, runtime_controls.go, html.go) — DONE

No defects found. runtime.go's config/adapter plumbing is defensive throughout;
runtime_controls.go's replay/lane state documents its schedulerMu contract at
each access; html.go is trivial constructors. Noted only: replay-buffer
overflow shifts the slice per event (O(n), dev-only recording) and
fiberPathIndexes is quadratic in sibling count (also dev-only) — both fine.

## Module 3: internal/runtime — chunk 4 (hooks.go, hooks_fetch.go, scheduler.go) — DONE

### Fixed
- **hooks_fetch.go — refetch's loading update targeted the stale creation
  fiber.** All the async completion callbacks schedule against
  `fetches[idx].fiber` (re-pointed every render); the initial loading-mark
  still used the closure's creation fiber, which can be a detached generation
  by the time refetch runs — the loading state could miss its re-render. Now
  targets the tracked fiber like the callbacks.
- **scheduler.go — dead `sync.Once` ritual removed** from EnqueueUI (empty
  Do body, no initialization to guard).

### Reviewed, left alone (reasons)
- hooks.go: accessor caching, slot growth, effect epochs, and fastEqual's
  slice-identity semantics (alias/prefix edges) all verified correct; the
  rejected-optimization NOTEs (interface-identity fast path) are accurate.
- GoUseFetch's then/catch js.Func release pattern is single-settle-safe;
  a never-settling promise leaks both funcs by contract.
- Fiber dirty-flag writes outside schedulerMu are the documented
  single-goroutine-per-render design; not a cleanup-pass change.
- Windows note: the wasm exec .bat breaks on `-run 'A|B'` alternation (cmd
  pipe parsing) — reports FAIL at ~0.03s with "'C:' is not recognized".
  Use separate runs or the full suite.

### Verification
- `./internal/runtime/... ./test/render/...` native — ok; full wasm runtime
  suite under node — ok (10.1s)

## Module 3: internal/runtime — chunk 5a (reconciler_commit.go) — DONE

### Fixed
- **Queued effects re-ran on bailout-reused fibers (MAJOR).** `runFiberEffects`
  never consumed `fiber.effects` after running them, and fibers below a bailout
  boundary are REUSED (same object) across commits. Any commit where no
  re-rendered component queued effects takes the full-tree `runEffects`
  fallback (`tracksPendingEffects` stays false), which re-executed every stale
  queued effect: mount effects re-fired on unrelated updates, and their prior
  cleanups were overwritten WITHOUT being called (leaked subscriptions,
  doubled listeners). Latent because effect-heavy passes take the pending-
  fiber path, which skips reused fibers. Reproduced empirically (runs=3 after
  two unrelated counter bumps, want 1) with the effect one component level
  below the clone line — effects ON the first level are safe because
  cloneChildFibers zeroes the field. Fix: `runFiberEffects` now truncates the
  queue after running (deferred, panic-safe). Pinned by
  `TestMountEffectDoesNotRerunAfterBailoutReuse`.

### Reviewed, left alone (reasons)
- LIS-based committed-child-order repair (patience algorithm), placement
  batching gates, serialized-run integration, portal retry
  (portalUnresolved), passive-listener add/remove transitions, and the
  attr-update batch Begin/End pairing in commitRoot all verified correct.
- `runCleanups` deliberately not walking the argument's sibling chain is
  correct and well-documented (shared Hooks with live alternates).

### Verification
- Native: `./internal/runtime/... ./test/render/... ./test/hooks/...` — ok
- wasm under node: `./internal/runtime/ ./test/render/` — ok

## Module 3: internal/runtime — chunk 5b (hot_reload.go) — DONE

No changes. Snapshot capture/restore (selective + legacy), signature prefix
compatibility, JSON coercion fallback, and renderFunctionComponent's
render-phase convergence + effect dedup all verified. Known minor gap (logged,
not fixed): a panic during a render-phase CONVERGENCE re-run bypasses
error-boundary recovery (only the first attempt wraps the render in the
boundary recover); the work-loop containment still keeps the page alive.
renderFunctionComponent resets fiber.effects at render start, which composes
correctly with chunk 5a's consume-on-run truncation.

## Module 3: internal/runtime — chunk 6 (reconciler.go, reconciler_commit_subtree.go) — DONE

### Fixed
- **reconciler_commit_subtree.go — serialized mounts wrote the tag name raw
  into parsed HTML.** jsdom's CreatePreparedElement validates tag names before
  building HTML, but the serialized-mount path bypasses it: a hostile or
  malformed tag broke the positional bind zip (silent subtree corruption) or
  injected markup. Added `isSafeSerializedMountTag` (letters/digits/hyphen)
  mirroring the adapter-side guard; ineligible tags fall back to per-node
  mounting.

### Reviewed, left alone (reasons)
- reconciler.go: keyed reconciliation's duplicate-key fallback routing and
  deletion accounting (every old fiber tracked in exactly one structure)
  verified; the in-order fast path validates the whole run before mutating any
  chain links; buildUpdatedFiber copies `key` from the ELEMENT (correct — the
  bailout-clone fix in chunk 3a was the only key-dropping site).
- Non-Element children (e.g. a raw int) skip reconciliation without consuming
  the positional old fiber — only reachable with malformed input; strings are
  normalized to TEXT_ELEMENTs at construction.
- Text containing \r binds against parser-normalized \n — render-equivalent,
  self-corrects on first update.
- Note: gopls emitted phantom "serializedMountRoots undefined" diagnostics
  during the edit; real compile is clean (recurring stale-metadata pattern
  this session).

### Verification
- `./internal/runtime/... ./test/render/...` — ok (incl. serialized-mount pins)

## Module 3: internal/runtime — chunk 7 (diagnostics/inspect/agent tail, ~35 files) — DONE

Swept by a sequential Sonnet reviewer (read-only findings), each finding
verified against source before fixing.

### Fixed
- **strict_diagnostics.go — `RecoverableOnly: false` was unexpressible.** A
  zero-value coercion forced the flag to true on every ConfigureStrictDiagnostics
  call, so escalate-everything mode silently behaved as recoverable-only.
  Coercion removed; the branch test now covers BOTH flag directions (it had
  codified the coercion).
- **agent_detached.go — detachedRuntimes registry grew without bound.** One
  full Runtime per agent-supplied selector, never evicted (unmount only
  cleared DOM). A nil render now evicts the entry; a fresh mount to the same
  selector gets a new runtime (state was discarded anyway). Contract test
  updated to assert eviction.
- **inspect.go — containsHydrationFold was O(n²)** on messages dense in one
  case of 'h' (per-position IndexByte for the absent case letter scanned the
  full remainder). Rewritten as a single byte scan.
- **inspect.go — previewValue could split a UTF-8 rune** at the fixed byte-45
  cut, rendering escaped garbage in hook-value previews. Cut now backs off
  continuation bytes.
- **hooks_fetch_stub.go — vestigial dead locals removed.**

### Reviewed clean (by Sonnet, spot-verified)
panic_report.go + console wasm/stub pair, panic_artifact_metadata,
panic_source_map, diagnostic_metadata, inspect_reporting (ring buffers
correctly capped), plugininterposer, agent_read/write + wasm/native pairs
(bridge validates hook-slot bounds; values coerce via JSON round-trip; no raw
string reaches a DOM API), dom_ref, events_pointer, event_wrap pairs,
event_wrap_cell, ready, hook_threading pair, hooks_func_validate pair,
commit_timing pair, passive_event, shim/shim_native (wasm-only globals are
intentional, not a pair mismatch).

### Noted, left alone
- agent_write.go waitForAgentWriteReady busy-polls schedulerMu at 1ms up to
  2s — bounded, agent-only path; a cond-var refactor is not worth the churn.

### Verification
- `./internal/runtime/... ./agentbridge/... ./test/render/...` — ok
- Full wasm runtime suite under node — ok

## Module 4: ui package (94 production files) — DONE

Core (ui.go, ui_native.go, component_handle_shared.go) reviewed directly;
tail (~90 files) swept by a sequential Sonnet reviewer, findings verified.

### Fixed
- **ui_native.go — native `UseContext` panicked UnsupportedOnServer despite
  full runtime SSR context support** (vestigial from before SSR hooks).
  Components using context were un-SSR-able. Now mirrors the wasm
  implementation; test proves a provider value resolves through native SSR.
- **defer_blocks.go — UseAsyncDefer's loader goroutine had no panic
  containment** (every sibling async hook has it): a panicking loader killed
  the whole wasm program. Added RecoverContainedPanic.
- **bootstrap_wasm.go — ReadBootstrapReference hung forever + leaked 3
  js.Funcs when any promise callback panicked** (containment only logged;
  nothing resolved the blocking channel, so the deferred Releases never ran).
  Callbacks now recover into the result channel.
- **files_wasm.go — PickFile leaked one js.Func per cancelled dialog**
  (cancel doesn't fire "change") plus the FileReader load func on read
  errors. Added a "cancel" listener and a reader "error" listener with
  once-release guards; read failures now report a diagnostic.
- **overlay_runtime_wasm.go — conditional UseId() shifted the hook sequence**
  in UseOverlayStack and Overlay whenever the caller's ID prop toggled
  between empty and provided. UseId now called unconditionally.
- **Pre-existing wasm test failures fixed**: preference-hook tests called
  Use* hooks at test top level (skip on js, covered natively);
  FuzzCSSEscape build-constrained off js/wasm (Windows corpus loading).

### Known failing (tracked, not fixed here)
- TestParallelRegionClickEventBridgeCommitsWorkerPatch (wasm): worker region
  RenderIR missing the bridged click render — task #40, investigate with the
  runtime2/worker module.

### Systemic finding
- **CI never RUNS wasm tests** (only builds GOOS=js targets) — the root cause
  behind every "how did this ship" wasm bug in this audit, including broken
  hydration. Task #39: add a wasm test job.

### Reviewed clean
ui.go, component_handle_shared.go (G1 aliasing mechanism confirmed — handle
identity is the code pointer, implementation swapped per registration; task
#38 tracks the diagnostic), and ~80 tail files listed clean by the reviewer
(form state machines, ssr_transfer envelope parsing, parallel-region bridges,
overlay focus/inert logic, timers, spring loop termination, bounded caches).

### Verification
- `go test ./ui/... -count=1` — ok (native)
- wasm ui suite under node: all green except the tracked #40 failure

## Module 5: state/events/scheduler/kvstate/query/flags/telemetry/textutil/deprecation — DONE

state/ reviewed directly; the other eight packages swept by a sequential
Sonnet reviewer, every finding verified before fixing.

### Fixed
- **kvstate/engine.go — engine cache keyed by Name only (HIGH).** A second
  Table under the same Name silently reused the first table's engine (rows
  written to the wrong table); a different Durability was silently ignored.
  Now one shared DB per Name with per-table engines; a Durability conflict is
  an explicit error.
- **events/events.go — topic registry never released entries (HIGH).** The
  documented "clears the registry entry when the last subscriber leaves" was
  never implemented: dynamic topic names leaked topicState + the retained
  lastValue forever. Implemented race-safely (dead flag + CompareAndDelete +
  retry in Subscribe/Publish); WithReplayLast doc updated for the
  replay-after-full-unsubscribe consequence.
- **query/query.go — no eviction path (HIGH) + last-to-finish-wins races
  (MEDIUM) + silent type-mismatch (MEDIUM).** Added Evict/EvictPrefix/
  EvictAll (Invalidate keeps retaining values by design); added a per-entry
  generation counter so background fetches and optimistic mutations commit
  only when no newer write landed mid-flight (rollback can no longer clobber
  a concurrent mutation's success); a cached value of the wrong concrete type
  now reports StatusError instead of Success-with-zero-value. runFetch is
  evict-tolerant. API baseline regenerated for the new Evict methods.
- **kvstate — decode errors silently swallowed (MEDIUM).** Hydrate and
  cross-tab watch paths kept the initial value with Err()==nil on schema
  drift/corrupted rows; all four sites now surface the decode error. Also:
  cross-tab hub now drops empty inner maps (slow leak), dead nil check
  removed.
- **textutil — byte-index capitalization corrupted multi-byte leading runes**
  ("über" → invalid UTF-8). Rune-based upperFirstRune.
- **events/typed_publish.go — dead codecMu removed.**
- **state/autotrack.go — documented the discovery-pass concurrency contract**
  (cross-goroutine reads during discovery mis-attribute deps; serializing
  would deadlock the supported nested case).

### Noted, left alone
- Triplicated manual itoa in scheduler/telemetry (cosmetic duplication).
- kvstate registries are keyed by developer-chosen names by design; no cap.

### Clean
state/ (signal/global-atom/atom-key semantics and footguns all documented),
flags/, deprecation/, query/devtools, kvstate backend/codec/conflict/
strategy/transfer/unload (sanitizeIdent guards SQL identifiers), events
introspect.

### Verification
- All nine package suites — ok; ui + test/render downstream — ok;
  whole-repo build unchanged (pre-existing wasm-main example messages only)

## Module 6 (part 1): router / css / head / a11y / anim — DONE

sanitize.go reviewed directly (clean allowlist sanitizer). Sonnet swept
router/css/head/a11y/anim; every finding verified before fixing.

### Fixed
- **css/sink_wasm.go — DOM sink not thread-safe (HIGH).** registerAndEmit
  releases the registry lock before Emit; the wasm domSink then did an
  unlocked read-modify-write of the <style> textContent (and unlocked lazy
  element cache), so concurrent emits could lose CSS or duplicate the style
  element. Added a mutex mirroring the native bufferSink.
- **router — nested-route metadata erased by empty child fields (HIGH).**
  Metadata was applied per level root→leaf; a leaf's empty Title/Description/
  Canonical reset the document back to base, silently wiping a layout route's
  metadata on every normal layout+outlet navigation. Now merged once over the
  matched stack (deepest non-empty wins) and applied a single time.
- **router — popstate/hashchange handlers ignored the disposed flag (HIGH).**
  A replaced Router (hot-reload, remount, hash<->history switch, test harness)
  left the old router's window listeners attached and rendering stale state
  against a reused target, racing the live router. Handlers now no-op when
  disposed. (Listener removal itself deferred — no per-router dispose hook.)
- **router/route_chunk.go — js.Func release-then-invoke crash race (MED).** On
  ctx cancel the load/error funcs were Released while still attached; a late
  event invoking a released func panics. Now removeEventListener before the
  deferred Release.
- **css/rule.go — at-rule "||" delimiter collided (MED).** An at-rule value
  containing "||" merged unrelated rule groups and mis-split nesting on render.
  Switched to the ASCII unit separator (illegal in CSS at-rules).
- **css/value.go — Hex accepted illegal digit counts (LOW).** Hex("12345")
  emitted the browser-dropped "#12345"; now only 3/4/6/8-digit forms pass,
  else black.

### Deferred to tasks (verified real, larger changes)
- #41 router pattern-route specificity precedence (first-registered-wins makes
  some patterns unreachable silently).
- #42 css wasm Inject/Reset/Harvest parity with native (wasm Inject bypasses
  the managed style element; Reset is a wasm no-op).
- #43 css unbounded registry/cache under per-render dynamic classes.

### Reviewed clean
router match/loader-lifecycle (generation-counter prevents stale-loader
commits; %2F segment-smuggling rejected; bounded loader/chunk caches), css
utility/theme/prop/selector/variant/dynamic/preflight/hardenCSS (author-trust
escape hatches are intentional, NUL-first filter-bypass guard correct),
head (JSON-LD + attribute escaping handles </script>, U+2028/2029), a11y and
anim (pure Go, no js/timers).

### Verification
- Native: css/router/head/sanitize/a11y/anim — ok
- wasm build of router/css/head — ok; css wasm suite under node — ok

## Module 6 (part 2): fetch / interop / i18n / logging / validate — DONE

Sonnet swept ~14k lines (interop is the codebase's largest js.Value surface);
every finding verified before fixing.

### Fixed
- **fetch/fetch_wasm.go — synchronous fetch()/XHR throw hung the caller
  forever (HIGH).** Browser fetch() and xhr.open()/send() throw SYNCHRONOUSLY
  for realistic inputs (GET with body, invalid URL, bad header/method); the
  goroutine's plain RecoverContainedPanic logged and swallowed, leaving the
  result channel unwritten/unclosed and `<-Fetch(...)` / `range Upload(...)`
  blocked forever. Both now settle the channel with a terminal error on any
  synchronous throw (Fetch: settle-guarded recover; Upload: finalize machinery
  hoisted above all throwable calls).
- **fetch/realtime_wasm.go — WebSocket/EventSource listeners had no panic
  containment (HIGH).** A panic anywhere downstream of an incoming message
  (malformed server payload, re-render bug) killed the page on every socket
  event. Added RecoverContainedPanic to the per-message listener.
- **logging/browser_console_wasm.go — document-wide interaction listeners had
  no panic containment (HIGH).** click/submit/error/popstate/... handlers and
  the mount MutationObserver could crash the whole wasm program on any panic.
  Added a self-contained recover (logging has no runtime dependency) reporting
  to console.error.
- **fetch/mutation_queue.go — lost durable mutation under concurrency (MED).**
  Enqueue/Remove/ReplayWithOptions did unguarded load→mutate→save; two
  concurrent calls dropped one writer's mutation. MutationQueue is a value type
  shared by copy, so a mutex field wouldn't protect copies — added a
  package-level per-storage-key lock.
- **i18n interpolateTemplate — map-order-dependent placeholder
  cross-contamination (MED).** Sequential ReplaceAll re-scanned substituted
  values, so an arg value containing another key's placeholder interpolated
  nondeterministically. Single-pass strings.NewReplacer.
- **Unbounded caches bounded (MED):** i18n localeCandidateCache (keyed by
  request-controlled locale strings) and interop intlFormatterCache (keyed by
  per-user locale/currency/timezone) now size-capped; past the cap they
  recompute instead of growing.
- **interop/cookie.go — cookie name not escaped (LOW).** A name with `;`/`=`/
  CR/LF could inject a Set-Cookie directive; now strips those illegal-in-name
  bytes (valid names unchanged, so reads still match).

### Reviewed clean
Essentially all of interop core/wrappers (js.Value accessors consistently
Type()/Truthy()-guarded; js.Func Released once with on*-nulled-before-release),
fetch pure-Go resilience/cache/realtime state machine, validate (depth-bounded,
no js), most of i18n/logging.

### Noted, left alone
- interop Value.Bool/Int/Float are unguarded primitives (no exploitable call
  site today; latent footgun for future callers).
- fetch cache.go / query_cache.go registries need app-driven Sweep/DisposeAfter
  (documented escape hatch).
- i18n/locale_wasm.go hand-rolls localStorage/navigator access instead of going
  through interop (works, but bypasses the abstraction).
- Pre-existing wasm test panic in logging (Value.Get on string, test-double
  only) → task #44.

### Verification
- Native: fetch/i18n/logging/interop/validate — ok
- wasm under node: i18n, interop — ok; logging has the one pre-existing
  test-double failure (task #44)

## Module 7: internal/runtime2 (98-file worker runtime) — DONE

Swept by FOUR parallel Sonnet reviewers (Cam's 2026-07-05 instruction to run
4-at-a-time) over disjoint groups: binary/transport decode, patch/render-IR,
host/worker adapters, scheduler/coordinator. Every finding verified against
source before fixing; stash-compare confirmed zero new test failures.

### Fixed (all pinned by new tests)
- **binary_source_value.go — unbounded decode recursion → worker stack
  overflow (HIGH, reachable).** Decoders bounded items-per-level but never
  depth; a crafted deeply-nested payload from untrusted worker bytes overflowed
  the Go stack (fatal, unrecoverable) from every binary-envelope entry point.
  Threaded a depth budget (64). Pin: TestParseBinarySourceValueRejectsDeepNesting.
- **dom_commit.go — cyclic node graph → infinite recursion crash (HIGH,
  reachable).** A replace-subtree op can carry mutually-referencing children;
  a later remove-node drove parseDeleteNodeIDs (post-order delete) into
  unbounded recursion, and hasCommitRemoveRootAncestor could loop on a parent
  cycle. Fixed: delete-before-recurse (map = visited-set) + bounded ancestor
  walk. Pins: TestParseDeleteNodeIDsTerminatesOnCyclicGraph,
  TestHasCommitRemoveRootAncestorTerminatesOnCyclicParents.
- **render_style_value.go — CSS injection via map-form style value (MEDIUM,
  security).** {"color":"red; position:fixed"} injected extra declarations;
  now rejects ';' '{' '}' and control bytes. Pin:
  TestFormatRenderStyleMapRejectsInjection.
- **host_control_dispatcher.go — ControlKindRestart forgot to reset the patch
  version watermark + idempotency tracker (HIGH landmine, dormant).** Would
  silently drop every post-restart patch (frozen region); now matches the four
  other reset paths.

### Verified FALSE POSITIVE (no fix)
- recovery_coordinator.go "unbounded leak" — the coordinator is per-adapter
  (one region each), maps are O(1) and freed with the adapter. Left unchanged.

### #40 root cause localized (NOT in runtime2)
Traced the failing test's exact version numbers through the whole runtime2
commit path — every gate passes. Defect is upstream in the worker re-render
(ui/parallel_region_worker_bridge.go event-slot branch): the worker RenderIR
is missing the clicked render. Task #40 updated.

### Deferred to tasks (real, but regression-risky or dormant): #45, #46, #47.

### Verification
- `go test ./internal/runtime2/ ./ui/... ./test/render/...` — ok
- wasm: build ok; 15 pre-existing failures (O_DIRECTORY corpus + capability
  env mismatch, #39/#44) — IDENTICAL count with/without my changes
  (stash-compared) → zero regressions.

## Module 8: tools/gwc, agentbridge, devtools, pwa, livereload, telemetryredaction — DONE

Reviewed by 4 parallel Sonnet readers over disjoint file groups; every finding
re-verified against source before acting.

FIXED (all pinned where behavior changed):
- agentbridge/commands_render.go — agent-supplied render tree was UNBOUNDED
  (unlike bridge.snapshot's MaxDepth/MaxNodes). Added depth (32) + node-budget
  (2000) caps to renderBuildElement → prevents resource exhaustion / huge DOM
  mount from untrusted agent input. Pin: TestRenderBuildElementBoundsDepthAndNodeCount.
- agentbridge/commands_write.go — writeHandleMount had no cap on mounted roots;
  an agent could mount unique ids forever. Added writeMaxMountedRoots=256 guard.
- agentbridge/commands_control.go — controlHandleWaitFor timeout was unclamped;
  RunLoop dispatches synchronously on one goroutine, so a huge timeoutMs blocked
  every other frame (self-DoS). Clamped to 30s.
- devtools/support_bundle.go — sanitizeSnapshotForSupport redacted every section
  EXCEPT Kernel; plugin-supplied KernelPluginHealth/Event freeform strings could
  leak secrets/PII in the ONE bundle meant to leave the machine. Added
  sanitizeKernelForSupport.
- tools/gwc/tailwind.go — downloadTailwindBinaryFile fail-OPEN (checksum manifest
  unreachable, or asset missing from it) was SILENT: a downloaded executable
  installed with no integrity check. Kept fail-open (older releases ship no
  sha256sums.txt) but made it a loud stderr warning. Mismatch still fails closed.
- tools/gwc/examples.go — renderExamplesListingHTML reflected the ?q= query param
  UNESCAPED into the meta line (reflected XSS; %q is not HTML-safe) and wrote
  filesystem-derived link Name/Href unescaped. Now escapeHTML on all three. Pin:
  TestRenderExamplesListingHTMLEscapesReflectedInput.
- tools/gwc/deploy.go — manifest artifact paths were filepath.Join'd with no
  containment check → a crafted manifest could read files outside the artifact
  dir and (filesystem adapter) write them outside the target (traversal/zip-slip).
  Added ensureDeployPathContained. Pin: TestRunDeployRejectsPathTraversalArtifact.
- tools/livereload/livereload.go — lastBuildStatus was written under mutex in
  triggerBuild but read/written UNLOCKED in sendCurrentBuildStatus,
  checkCurrentBuildState (its own goroutine), and triggerBuild's mkdir-error
  branch → data race. Now uniformly mutex-guarded.

DEFERRED (tracked tasks, regression-risk / owner-decision):
- #48 plugin panic isolation at Host dispatch (failure-semantics change, ~11 sites)
- #49 pwa UpdatePrompt.Apply ReloadOnControllerChange handle leak (low sev)
- #50 telemetryredaction struct-walk policy (out of JSON-like contract; reflect risk)

Tests: tools/gwc, agentbridge, devtools, pwa, telemetryredaction, livereload all
green natively. (-race unavailable on windows/arm64; livereload fix verified by
uniform-locking reasoning.) NOTE: tools/livereload is a SEPARATE go module.

## Module 9: sanitize / serverfn / servercomponents / prerender / plugin / pluginruntime / localfirst / db/sqlite (security-critical core) — DONE

4 parallel Sonnet reviewers over disjoint groups; every finding re-verified
against source (the sanitize reviewer also verified empirically). This module was
security-dense — many findings were genuine but design/behavior-level and were
DEFERRED to tracked tasks rather than applied unilaterally.

FIXED (all pinned):
- serverfn/serverfn.go — (a) CRITICAL: io.ReadAll on the request body was uncapped
  (trust boundary → memory-exhaustion DoS); added http.MaxBytesReader + configurable
  SetMaxRequestBytes (default 10 MiB) → 413. (b) MED: no panic recovery around the
  user fn (a nil/index panic on attacker-shaped input reset the connection); added
  defer-recover → 500. (c) MED: json.Encoder.Encode error was discarded — a marshal
  failure left a misleading empty 200; marshal-to-buffer first → 500 on failure.
  Pins: TestHandleRejectsOversizedBody, TestHandleRecoversPanic. (API baseline
  regenerated for the new SetMaxRequestBytes export.)
- prerender/export.go — normalizeRoutePath rejected only non-'/' prefixes, not '..'
  segments; a route path is turned into an on-disk file path (buildTarget →
  filepath.Join → os.WriteFile), so '/../../x' escaped the output dir (traversal).
  Added '..'-segment rejection. Pin: TestExportRejectsPathTraversalRoute.
- internal/pluginruntime/kernel.go — getEvents diagnostic log grew without bound
  (each entry carries a full diagnostics.Report) over a long-lived process that
  boots/closes kernels. Capped to the most recent 512 (maxKernelDiagnostics).
- db/sqlite/sqlite.go — applyPragmas built "PRAGMA k=v" by string formatting (the
  ONE non-parameterized SQL path); pragma keys/values from config could inject SQL.
  Added isValidPragmaKey/isValidPragmaValue (identifier / signed-int allowlist).
  Pin: TestPragmaValidationRejectsInjection.
- sanitize/sanitize.go — (a) HIGH: <template> was neither dropped nor allow-listed,
  so x/net/html (which doesn't model template's inert .content) let its children
  unwrap into live out-of-context output (mutation-XSS precondition); added
  "template" to dropWithContents. (b) MED: URL attrs were validated on a control-
  char-stripped copy but the RAW value was emitted; now emit the same cleaned
  string that was validated. (c) MED: colspan/rowspan were allow-listed by name
  only — oversized/negative values (rendering-hang vector) now validated to small
  positive ints. (d) doc: Policy immutability-once-shared contract + fail-closed
  size/depth behavior documented. Pins: TestSanitizeDropsTemplateSubtree,
  TestSanitizeValidatesSpanAttributes, TestSanitizeEmitsCleanedURL.

DEFERRED (tracked tasks — design/behavior-semantics, owner sign-off or careful
concurrency work):
- #52 serverfn CSRF/Content-Type + error-message disclosure (statusForError's plain-
  error passthrough is INTENTIONAL/tested — a breaking change).
- #53 plugin Host hardening: Host has ZERO synchronization (CRITICAL data race),
  Manifest.Requires not enforced per-plugin, no callback timeout (hang), DecorateCacheKey
  unbounded amplification, reentrant-Register rollback corruption, BootstrapData shallow
  clone — interdependent, touch the same dispatch sites as #48.
- #54 pluginruntime ResolveService per-plugin service-allowlist enforcement.
- #55 localfirst/db: Flush/Tx torn-write (CRITICAL wasm), tombstone/history GC,
  gwcMemoryDBs eviction, Text/Counter thread-safety asymmetry, Text.ordered O(n²).

Reviewer-confirmed CLEAN (no action): LWW/PN-counter/presence convergence, AES-256-GCM
encryption, Exec/Query parameterization, tx commit/rollback, sanitize scheme-evasion
corpus (js:/data:/vbscript: via tab/ZWSP/entities/case), dropWithContents precedence
over custom Policy, pluginruntime Kernel locking + panic isolation, cloneManifest deep
copies, servercomponents/prerender SSR escaping.

Tests: all touched packages green natively + wasm-build clean. Full native suite:
zero regressions. Pre-existing wasm-only failures unchanged (serverfn api_baseline
O_DIRECTORY dir-read #44; servercomponents server-node test renders a wasm placeholder
by design #39) — proven pre-existing (servercomponents is unmodified by this module).

## Module 10: virtualization / hotreload / diagnostics / timetravel / agentui / wholestack / workbench — DONE

4 parallel Sonnet reviewers; every finding re-verified against source. This module
skewed toward delicate perf/design findings that were DEFERRED with detailed tasks
(the fix risk exceeds the audit's safe-edit envelope).

FIXED (all pinned where a native test could exercise it):
- agentui/agentui.go — HIGH: Registry.specs (a plain map) had NO synchronization;
  concurrent Register vs Validate/Render/Catalog is a FATAL, un-recoverable Go
  "concurrent map read and map write" (worse than a normal panic — kills the
  process). Added sync.RWMutex + a lookup() helper that copies the spec under the
  read lock so the lock is never held across user Render callbacks/recursion. Also
  guarded render() against a nil-Render spec (degrade to empty node, not panic).
  Pins: TestRegistryConcurrentRegisterAndRead, TestRenderToleratesNilRenderSpec.
  (agentui api_baseline regenerated for the internal mu field.)
- wholestack/wholestack.go — HIGH: nil Options.Assets (a valid API-only build)
  caused a nil fs.FS panic on the first request (both http.FileServer and
  serveFile). Handler now serves a clear 500 for non-server-fn paths when Assets
  is nil. MED: ListenAndServe used bare http.ListenAndServe (no timeouts →
  Slowloris FD/goroutine exhaustion for the "one-line production entry point");
  now builds an http.Server with ReadHeaderTimeout/ReadTimeout/IdleTimeout (no
  WriteTimeout, so slow large-wasm downloads aren't cut off). Pin:
  TestHandlerNilAssetsReturns500.
- hotreload/hotreload_wasm.go — MED: installBridge partial failure left enabled=true
  while no bridge existed (callers gate dev behavior on Enabled()); now
  enabled=bridgeInstalled. MED: hotReloadDiagnostics grew unboundedly and re-
  filtered the full list on every devtools poll; capped at 32 (like activity).
- diagnostics/diagnostics.go — doc: Emit's comment claimed "registered listeners"
  that don't exist; corrected to describe the stderr-only behavior.

DEFERRED (tracked tasks — delicate perf/design or owner-decision):
- #57 hotreload restore-wedge (needs transient-vs-permanent error distinction; an
  existing test pins the current skip-clear behavior) + package-global sync.
- #58 virtualization CRITICAL perf: O(N) key rebuild + ~500KB keySignature alloc on
  EVERY scroll frame (defeats virtualization); + spacer/row rounding drift;
  + unbounded restorationStore; + duplicate-ItemKey corruption; + NaN guards.
- #59 timetravel snapshot aliasing for reference-typed T (documented but unenforced
  & untested — undo silently no-ops if the caller mutates a stored ref).
- #60 diagnostics WriteHTTPError leaks stack traces/source paths/internal error text
  (and OS username via shortenFilePath) to the HTTP client — needs a debug gate.
- #61 apidump extractor accuracy: silently drops generic type-params & const values
  (FALSE PASS on real breaking changes) and includes unexported fields/param names
  (FALSE FAIL noise) — high-blast-radius infra fix (regenerates all baselines).

Reviewer-confirmed CLEAN: ComputeViewportState range math, renderRows O(visible),
timetravel History locking + capacity eviction, devpanel, hotreload schema/migration
atomicity, diagnostics concurrency/format-strings/nil-safety, apidump traversal
safety, agentui XSS/recursion-bounds, workbench/gallery/fixtures.

Tests: touched packages green natively + wasm (hotreload). Full native suite: zero
regressions.

## Module 11 (FINAL): testkit / CI tools / build tooling / test harnesses + doc tools — DONE

4 parallel Sonnet reviewers over the CI/harness/codegen tier; every finding
re-verified. This tier's worst outcome is a CI gate or test util that SILENTLY
PASSES when it shouldn't — several such were found and the contained ones fixed.

FIXED (pinned where testable):
- testkit/hooks/hooks_wasm.go — CRITICAL (shipped test util): RenderHook.Rerender()
  re-declared the host component closure on EVERY render, so the reconciler's
  identity check (which includes the closure environment pointer) saw a different
  component each time → full remount that WIPED all UseState/UseRef/UseReducer
  state and re-fired mount effects. A hook state-persistence test would get a false
  pass/fail. Fixed by capturing the host function ONCE + reusing it with a changing
  Tick prop. Pin: TestRenderHookPreservesStateAcrossRerender.
- testkit/render/render_signals_wasm.go — MED-HIGH: applyRenderCountSignal matched
  substring before exact, and signals are sorted by render count desc, so
  ApplyRenderCountMax("Button") could silently check "IconButton"'s budget. Now
  two-pass: exact Name/Path first, substring only as fallback.
- tools/changelogcheck — HIGH false-PASS (this gate blocks tag pushes in release.yml):
  containsVersionToken matched the version ANYWHERE in a "## " header, so
  "## Migration notes for 3.0.46 users" satisfied the gate with no real entry. Now
  anchored to the header's leading identifier. Pin: TestHasEntryRejectsProseMention.
- tools/api_compat_guard — CRITICAL false-PASS: an empty/"{}" baseline made
  compareBaselines iterate zero times and "pass" even if every symbol were deleted.
  readBaseline now rejects len(Targets)==0.
- tools/sbom — HIGH: locally-replaced (in-tree) modules were emitted with fabricated
  pkg:golang PURLs at Go's placeholder pseudo-version, misrepresenting provenance in
  the SBOM. Now resolves Replace: versioned replaces use the real path@version,
  local-path replaces are skipped (in-tree source, not a fetchable package).
- tools/devtools-extension/pack — MED: manifest version was concatenated into the
  output path unvalidated (traversal) and the zip was written non-atomically
  (a mid-package failure truncated a good artifact). Added a path-safe-version
  check + temp-file-then-rename.
- tools/hookcheck — MED false-PASS: unparseable files (a genuine syntax error, since
  go/parser ignores build tags) were silently dropped; now logged to stderr.
- tools/sitegen — LOW: nil os.Stat result dereferenced for a size print; use the
  bytes already in hand.
- tools/runnerconfig — removed a dead IsAbs("bin") branch; GetArtifactNamespace now
  falls back on a ".." base (would have escaped the artifact root via Join).

DEFERRED (tracked tasks — design-level / CI-engineering / risk of surfacing existing
violations):
- #63 api_compat_guard signature-blind identity (misses param/return/field/generic
  changes) + no package-drift detection.
- #64 testkit remaining traps (loader-attempt positional merge, warning-log dedup
  undercount, dispatch auto-flush whitelist, ssr full-parse vs fragment, nodeText
  whitespace).
- #65 doclint flag-scan gaps (inline code + `--flag`) + errorcodes SELF-REFERENTIAL
  drift guard (can't detect a silently-dropped code).
- #66 reprobuild reproducible-build gate verifies the WRONG artifact (a wasm example,
  not the shipped native dist/gwc_* binaries).

Reviewer-confirmed CLEAN: test/{hooks,render,router,ssr} are pure re-export facades;
test/browser global-JS-env is properly t.Cleanup-scoped (safe while not parallel);
docs/capabilities deterministic + independently-checked; testkit state isolation
(render.New Reset:true) + fixtureGate; changelogcheck LatestEntry/CheckFile; hookcheck
core loop/conditional detection; runnerconfig precedence; sbom decode/sort/marshal;
sitegen path safety; reprobuild comparison logic. runnerconfig FS.Getwd is a dead
testability seam (blank-root fallback uses ambient cwd via filepath.Abs) — noted, low.

## Final: tools/agenthub (separate module — dev WebSocket hub) — DONE

The last package audited, completing full-framework coverage. Security-and-
concurrency-sensitive (per-run token, loopback+origin gating, many concurrent WS
sessions). Reviewer confirmed the core is SOUND: constant-time token compare,
unspoofable RemoteAddr loopback check, WS Origin validation, correct single-
writer/reader discipline, pendingAcks locking, bounded ring buffers, recording-
chain cycle guard.

FIXED (pinned): the WS conn had no SetReadLimit (unbounded frame allocation → a
token-holding peer could OOM the box) — now 16 MiB; the plain-HTTP API routes
enforced loopback+token but NOT Origin (a token-leak-contingent CSRF gap, unlike
the WS path) — authorizeAPI now enforces originAllowed for parity and caps request
bodies at 4 MiB. Pin: TestCrossOriginAPIRequestRefused.

DEFERRED: #67 (WS ping/pong keepalive + whole-session read deadline to stop a
dead-peer goroutine/memory leak; bound the append-only hub.sessions history; wire
the dead StateClosed state).

Full agenthub module suite: zero regressions.

---

AUDIT COMPLETE: all framework packages (Modules 1–11 + agenthub) reviewed
file-by-file. ~40 fixes shipped and pinned across the run; ~31 design/owner-
decision findings deferred to tracked tasks (#37–#67).

## Post-audit backlog implementation (deferred fixes, one at a time)

With the file-by-file audit complete, working the deferred backlog — contained,
non-breaking, non-concurrency-model fixes first; owner-decision/risky items still
gated on sign-off.

- #48 DONE — plugin Host panic isolation. Added PluginPanicHandler (overridable,
  defaults to stderr — observe, don't swallow) + generic pluginCall/pluginRun
  recover helpers, and wrapped all 11 dispatch callback sites (route guard, nav +
  request + submit observers, cache-key decorator, panel/section/action/head/
  bootstrap providers, form validator). A panicking third-party plugin now degrades
  (guard→allow, decorator→passthrough, provider→skip) instead of crashing the host.
  Pin: TestHostIsolatesPanickingPluginCallbacks. Native+wasm green. (#53's Host
  hardening still owns the remaining CRITICAL: Host has no mutex — the snapshot-
  under-lock rewrite composes with these panic wrappers.)
- #49 DONE — pwa UpdatePrompt.Apply reload-handle leak. Apply discarded the
  ReloadOnControllerChange subscription; on a SkipWaiting failure (no
  controllerchange ever fires) the listener leaked. Now retained on the prompt,
  cancelled on SkipWaiting error and on re-Apply (idempotent), and cleaned up in
  Stop(). Pin: TestUpdatePromptApplyCancelsReloadOnSkipWaitingFailure. Native+wasm
  green.
- NOTE: a full-suite gate surfaced FLAKY failures in tools/gwc's vuln-scan +
  build-variant tests (live advisory data / missing wasm-opt+brotli); re-runs pass,
  unrelated to any audit change. Tracked as #68.
- #54 ATTEMPTED + REVERTED — pluginruntime per-plugin service-allowlist
  enforcement. Rejecting undeclared ResolveService breaks two existing tests that
  resolve services without declaring them → the unrestricted behavior is relied
  upon; enforcing is a coordinated breaking change (owner decision). Reverted per
  the tests-must-pass guard; #54 stays deferred with the finding recorded.
- #53 (CRITICAL race) DONE — plugin.Host had ZERO synchronization. Added registerMu
  (serializes Register/Close across the Setup callback) + stateMu RWMutex guarding
  all mutable fields. Every Add* takes stateMu.Lock; every dispatch snapshots its
  provider slice under RLock and invokes callbacks OUTSIDE the lock (composes with
  the #48 panic wrappers → no reentrant-lock deadlock). SetValue/Value/Plugins/
  snapshot/rollback/hasPlugin guarded; capabilities immutable. Pin:
  TestHostConcurrentRegisterAndDispatchNoDeadlock. Native+wasm green. (User opted to
  tackle the CRITICALs.) #53's HIGH/MED sub-items (per-plugin capability enforcement,
  callback timeout, DecorateCacheKey length cap, reentrant Register, BootstrapData
  deep clone) remain on #53.
- #55 (CRITICAL torn write) DONE — db/sqlite DB.Flush snapshots the raw VFS image
  bypassing the sql.DB pool, unsynchronized with Tx. Flush now acquires the single
  pooled connection (SetMaxOpenConns(1)) before snapshotting, so a concurrent flush
  blocks until any in-flight Tx commits/rolls back — no mid-transaction (torn,
  journal-less) image is ever persisted. Localized to shared DB.Flush; native flush
  is nil (early return, unaffected). Pin: TestFlushSerializesWithOpenTransaction
  (native, fake flush + open Tx holding the conn). Native+wasm green. #55's remaining
  HIGH/MED items (tombstone GC, gwcMemoryDBs eviction, Text/Counter thread-safety,
  Text.ordered O(n²)) stay on #55.
- #58 (contained part) DONE — virtualization keySignature did fmt.Sprintf("%q", keys),
  allocating a full quoted copy of every key on every scroll-frame render (multi-
  hundred-KB per frame for large lists). Replaced with an allocation-free FNV-1a fold,
  same change-detection contract. Pin: TestKeySignatureChangeDetection. Native green;
  the one failing wasm test (TestListWASMRestoresViewportAndPersistsSnapshots) fails
  IDENTICALLY at HEAD (pre-existing #39-class wasm-on-Windows restoration failure), not
  a regression. #58's CRITICAL CORE (collectItemKeys O(N) walk per scroll frame) needs
  a caller-supplied Revision field on ListProps (API addition) or a scroll rAF-throttle
  (needs browser benchmarking) — both owner decisions, stay on #58.

CRITICALs status (user opted to tackle them): #53 Host race FULLY fixed; #55 db torn
write FULLY fixed; #58 virtualization — dominant per-frame allocation fixed, deeper
O(N) memoization needs an API/behavior decision.

## Module 12: DEEP-DIVE re-audit of the core runtime — HIGHLY PRODUCTIVE

4 Sonnet reviewers did a granular SECOND pass on the reconciler, hooks, runtime2
worker/patch, and scheduler — subsystems originally audited in bulk 169/383-file
chunks. The deep pass found MULTIPLE genuine CRITICALs the bulk pass missed,
validating the deeper look.

FIXED (contained, pinned):
- runtime2/binary_source_value.go — the MAP decoder called the PUBLIC
  ParseBinarySourceValue (resetting the depth counter to 0), so a map-nested-in-map
  payload bypassed the maxBinarySourceValueDepth stack-overflow guard I added in
  Module 7 (the list path was correct). Now threads parseDepth+1. Pin:
  TestParseBinarySourceValueRejectsDeepMapNesting.
- runtime2/patch_keyed_move_op.go — keyed-move DestinationIndex validator accepted
  == siblingCount but the applier's ceiling is siblingCount-1 (it removes the node
  before reinserting) → guaranteed rollback+fallback. Tightened to >= siblingCount.
- runtime/hooks.go — fastEqual's float64/float32 dep comparison used == so NaN != NaN
  → a NaN dependency re-ran the effect/memo every render forever. Added Object.is
  NaN==NaN semantics. Pin: TestFastEqualTreatsNaNAsEqual.

DEFERRED (CRITICAL/HIGH architectural/concurrency/lifecycle — detailed designs in
tasks; these are the real payload of the deep pass):
- #70 reconciler CRITICAL — child-order DOM repair skipped when the reordering fiber
  is DOM-less (a component/Fragment returning a keyed list directly): rows update in
  place but are NEVER physically reordered → silent visual desync. (+O(n²) match-build
  needs a DOMNode identity-key; +dead pending-effect path; +ref republish.)
- #71 hooks CRITICAL — currentFiber is a package-level global; concurrent SSR
  (concurrent HTTP handlers) races → hook state bleeds between requests, silent in
  production builds (guard compiled out). +empty-deps UseEffect runs-every-render
  parity bug; +layout/passive effect ordering only per-fiber not tree-wide; +cleanup
  runs during render not commit.
- #72 runtime2 2×CRITICAL — DOM node-index never cleared on worker-restart recovery →
  ID reuse collides with the stale tree (repair loop or silently spliced DOM); +patch
  idempotency ledger recorded BEFORE op validation → a legitimately-retried patch is
  permanently dropped, region wedged/desynced.
- #73 scheduler CRITICAL — fiber dirty-flag fields mutated from atom-writer/fetch
  goroutines with NO synchronization (schedulerMu guards only unrelated bookkeeping) →
  native data race, lost updates; +EnqueueUI hand-off never drained (dead + wrong-
  goroutine fallback); +every DOM event forces a sync full flush (no discrete/
  continuous split → scroll/drag jank).

Reviewers confirmed the keyed-reconciliation core, bailout/subtree-reuse, commit
ordering, slot bookkeeping, functional-setState batching, UseAtom, binary transport
bounds-checking, and registry locking are otherwise SOLID.

## Module 14: DEEP-DIVE — router / css / fetch-interop / parallel-region bridge

4 Sonnet reviewers. Headline: the parallel-region reviewer ROOT-CAUSED and I FIXED
the long-standing #40 failure.

FIXED (contained, pinned/verified):
- ui/parallel_region_worker_bridge.go — #40 ROOT CAUSE + FIX. buildParallelRegionWorker
  NodeOutput read only Element.Children, ignoring internal/runtime's direct-text fast
  lane (a host element's single text child is folded onto Element.TextContent, leaving
  Children EMPTY). So a <div>{label}</div> region's text never entered the worker
  RenderIR → text changes never diffed → no patch committed (and it was broader than
  the click — text was invisible to the worker end-to-end). Now re-materializes
  TextContent as a synthetic text child. TestParallelRegionClickEventBridgeCommitsWorker
  Patch RUNS + PASSES; full ui native+wasm green. #40 CLOSED.
- css — CSS-injection: AttrEq/ClassSel/AttrSel/DataTheme spliced caller strings into
  selectors unsanitized, and the process-global never-reset registry made one injection
  a PERSISTENT global rule (defacement/DoS across every later SSR response). Added
  cssStringEscape (attr values) + cssIdentSanitize (names). Also RGB/RGBA now clamp
  channels to [0,255] (matched alpha). Pins: TestSelectorConstructorsPreventInjection,
  TestRGBClampsChannels.
- router — url.ParseQuery returns valid pairs PLUS an error, but parseNavigationTarget /
  getCurrentQueryValues discarded ALL params on any error → one malformed param silently
  dropped every valid one. Now keeps the partial result. Pin (wasm):
  TestParseNavigationTargetKeepsPartialQuery.

DEFERRED (detailed tasks):
- #82 router — BeforeLeave bypassed by browser back/forward (user loses unsaved edits);
  DefaultRoute fallback skips layout; popstate+hashchange double-render; listener leak.
- #83 parallel-region — silent failure surfacing (unsupported nodes / func-props →
  swallowed encode error → region stuck); NOTE: no real Web Worker exists yet (sync
  same-goroutine call); #44's parallel-region-panic part likely moot post-#40.
- #84 css — FNV-1a hash collisions silently drop CSS (no canonical verification);
  O(N²) wasm sink; emitGlobal NUL-key collision; SeedFromDocument trust.
- #85 fetch/interop — InfiniteQuery/GoUseFetch stale-result races; EventSource tears
  down on transient blips (loses Last-Event-ID resume); Fetch has no cancellation;
  MutationQueue.Replay deadlocks on a hung executor; swallowed Remove → double-replay.

## Module 3 (internal/runtime) COMPLETE — summary
Three MAJOR fixes (browser hydration silently discarding server DOM; keyed
fast-lane row identity lost after bailout clone; stale queued effects re-run
on bailout-reused fibers), two SSR partial-markup leaks, one lost-update race,
one serialized-mount tag-injection hardening, one unexpressible-config fix,
one unbounded agent-bridge registry, plus perf/cosmetic/dead-code cleanups.
All pinned by new tests where behavior changed.

## Module 15 (deep-dive: html / head+i18n+logging / interop / small libs) COMPLETE
FIXED + PINNED (contained, ship-safe):
- html CustomElement XSS sink — Properties{innerHTML/outerHTML/insertAdjacentHTML}
  were forwarded verbatim as `__gwc_prop__:<name>`, letting a component assign
  attacker markup to element.innerHTML and bypass the framework's text-escaping.
  Added isUnsafeCustomElementProperty denylist (case/space-insensitive); the guard
  drops sink props, safe props still pass. Pins: html/custom_element_xss_test.go.
- i18n Bundle concurrent-map crash — Register mutated the catalog map while
  lookup/Locales/ToSSRBootstrap read it, no lock → fatal "concurrent map
  read/write" under SSR + live registration. Added mu sync.RWMutex: Register
  write-locks; lookup/Locales/ToSSRBootstrap read-lock (no nesting — Locales
  releases before ToSSRBootstrap's own read).
- validate numeric rules false-pass through pointers — min/max/gte/… silently
  no-op'd on *int/*float64 fields (measureLen/numericValue saw a Pointer kind and
  returned "not applicable" → rule skipped → invalid data passed). Added
  derefValidateValue (deref pointer/interface, nil → skip). Pins:
  validate/pointer_numeric_test.go (*int -5 invalid, 21 valid, nil skips).

DEFERRED (detailed tasks — need API/policy decisions or careful concurrency work):
- #87 interop js-bridge [CRITICAL] — Value.Get/Set/Delete panic on null receiver;
  Call/Invoke swallow JS exceptions (unnamed-return+defer-recover → zero Value, no
  error); jsValueToGo no cycle detection; WorkerPool.Close abandons in-flight;
  awaitValue leak on never-settling promises; int64 float64-bridge precision loss.
- #88 logging [HIGH] — normalizeLogValue doesn't walk struct fields (embedded
  secrets bypass redaction); browser-console sink emits PII unredacted; message
  string itself never redacted. Tie to #50.
- #89 head/i18n [MED] — head.Merge emits duplicate meta/link tags; i18n/extract
  misses Namespace.T(...) forms; catalog uncapped.
- #90 small-libs [CRITICAL] — kvstate version race (non-atomic compare-then-store,
  ties to #76); BindAtom subscription leak; sanitizeIdent accepts leading digit;
  telemetry no redaction hook (#50); flags bucketing hash bias.
- #91 html [MED] — markdown no nesting depth cap (stack-overflow DoS); DataAttr/Data
  same-key collision silent last-writer-wins; reflection Props merge per-call cost.
Gate: full library tree (`go list ./... | grep -v /examples/`) GREEN. (examples/
native build failures are pre-existing wasm-only mains, unrelated.)

## Module 15 backlog follow-ups (post-completion, 2026-07-05)
FIXED + PINNED:
- i18n/extract missed the Namespace.T(key) call shape — rt.NS("ns").T("key") has
  the namespace bound in the receiver, so the extractor (which read arg0/arg1 as
  namespace/key) misread the key as the namespace and dropped it from the
  completeness check → false "complete" against real code. Added
  parseNamespaceReceiver to detect the inline .NS("literal").T(...) chain and pull
  the namespace from the receiver; non-literal keys on a Namespace receiver are now
  surfaced as Dynamic instead of silently dropped. Kept narrow (receiver must be a
  .NS("literal") call) so unrelated one-arg .T methods aren't misclassified. The
  variable-bound Namespace form (ns := rt.NS(...); ns.T(...)) still needs
  type/dataflow info — documented limitation, surfaced as a skipped 1-arg .T.
  Pin: i18n/extract/namespace_receiver_test.go.
CORRECTED (not-a-bug): head.Merge "duplicate tags" is default-append BY DESIGN —
  MergeOptions.Replace{Alternates,ResourceHints,JSONLD,Extras} are the opt-outs.
  Changing the default would break TestMergeAppliesRouteDefaultsOverridesAndReplacementRules
  and the documented contract. Downgraded on #89; optional DedupExact flag only.
DEFERRED (need owner/registry decisions, NOT loop drive-bys):
- #76 state atom atomic functional-update — confirmed real; fix needs either
  lock-across-user-updater (reentrant-deadlock risk) or a per-atom generation
  counter on the store (touches every atom access path). Design recorded on #76.
- #89 i18n catalog entry cap — DoS-hardening policy choice, deferred.

## Module 15 backlog follow-ups #2 (kvstate + flags, 2026-07-05)
FIXED + PINNED (kvstate/hardening_test.go):
- kvstate sanitizeIdent accepted a leading digit — the result is interpolated
  UNQUOTED into DDL/DML, and "123abc" is an invalid unquoted SQL identifier
  (CREATE TABLE syntax error). Now prefixes a leading digit with '_'.
- kvstate BindAtom cross-tab subscription leak — subscribeCrossTab already returned
  an unsubscribe func that BindAtom discarded, so the subscription + captured
  engine/atom leaked for the process lifetime. Now captured and torn down on
  context cancellation (skipped for non-cancellable Done()==nil contexts).
- kvstate version lost-update (partial) — the write path did getVersion()+1 then a
  separate setVersion() inside a per-Set goroutine, so two concurrent writers could
  claim the same version and clobber each other. Added atomic nextVersion() and
  moved version assignment to be SYNCHRONOUS in parseSet → distinct, call-ordered
  versions. RESIDUAL: engine.Save is an unconditional upsert, so out-of-order write
  goroutines can still land a lower version last; full fix = version-conditional
  Save (engine.go SQL change), deferred on #90.
CORRECTED (won't-fix): flags getBucket "modulo bias" is negligible (~2e-8 at
  modulo=100) and correcting it would reshuffle every subject's bucket assignment,
  breaking rollout stability across versions. Stable fnv%modulo is the right call.
Gate: full library tree (minus examples/) GREEN.

## Module 15 backlog follow-ups #3 (html markdown DoS, 2026-07-05)
FIXED + PINNED (html/markdown_depth_test.go):
- RenderMarkdown recursive AST walk had no depth cap. goldmark does not bound
  blockquote/list nesting, so input like "> " repeated tens of thousands of times
  yields an AST deep enough to overflow the recursive render stack (crash/DoS).
  Threaded a parseDepth through renderMarkdownBlocks/Block/Inlines/Inline/Table/
  TableRow with const maxMarkdownRenderDepth=512 (far above any real document);
  over-depth blocks truncate with a visible marker, inlines return nil. Verified
  native + wasm (GOOS=js go vet) build clean. Deep-nesting pin returns instead of
  overflowing; shallow-nesting pin confirms real docs are not clipped.
DEFERRED on #91: DataAttr/Data same-key collision (silent last-writer-wins) and
  the per-call reflection Props merge cost — lower severity, batch separately.
- events #77 (topic registry / replay ordering): re-framed after review — the
  "publish-only leak" is actually TESTED publish-before-subscribe replay behavior
  (TestLateSubscriberWithReplayLast); genuine residual is unbounded unique
  subscriberless topic names (needs an LRU/TTL policy). WithReplayLast stale-after-
  fresh is a real MED but the fix needs reentrancy-safe delivery serialization.
  Both deferred with designs on #77.

## Module 13 backlog follow-up (jsdom text adapters, 2026-07-05)
FIXED + PINNED (internal/platform/jsdom/adapters_contract_wasm_test.go, wasm):
- WASMDOMAdapter.SetTextContent / GetTextContent nil-node panic — a typed-nil
  (*WASMDOMNode) passes the type assertion but nil-derefs on .value, and a
  null/undefined-valued node panics on .Set/.Get ("not an object"). Detached nodes
  are common on the render/commit path, so either crashed the whole app. Now guard
  with the existing nil-receiver-safe IsNull() idiom (matches Focus/callMethod).
  Pin runs typed-nil + js.Null + js.Undefined nodes through both without panic and
  confirms a concrete node still round-trips. Full jsdom wasm suite GREEN.
DEFERRED on #80: the "WrapHandler js.Func leak" is NOT a devtools-local bug and NOT
  a drive-by. Traced: ui.WrapHandler -> BuildDOMWrappedFunctionIfReadyGlobal ->
  WASMDOMAdapter.WrapFunction eagerly mints a js.FuncOf PER CALL (per render). The
  tracked CreateEventHandler/ReleaseEventHandler adapter path has NO callers in
  internal/runtime, so handler props are committed as raw js.Func properties — the
  open question (does the reconciler Release the old js.Func on prop-change/unmount?)
  is a core, framework-wide, wasm-only reconciler-commit investigation. Needs a
  js.Func-count-instrumented wasm test to confirm before any fix.

## Module 10 backlog follow-up (timetravel snapshot contract, 2026-07-05)
RESOLVED-AS-DESIGNED + PINNED (timetravel/snapshot_contract_test.go):
- #59 "snapshot-copy contract for reference-typed state" is NOT a defect. The
  package documents value-semantics as the deliberate contract (State held by
  value; "use value types or copy on Record"). Generic deep-copy enforcement is
  infeasible in Go and contradicts the pure/value design. Added pins: value-type
  snapshots stay independent after the caller mutates its copy (guards a refactor
  that accidentally stores a shared reference), and the intentional pointer-alias
  behavior is locked. Suite GREEN.
- ENHANCEMENT deferred on #59: an additive, opt-in clone hook (NewWithClone) for
  reference-type users — non-breaking (nil clone = current behavior) but requires
  updating TestPublicAPIBaseline, so it's an owner-decided change, not a drive-by.

## Module 10 backlog follow-up (diagnostics HTTP disclosure, 2026-07-05)
CONFIRMED (#60): WriteHTTPError writes Formatted() to the HTTP client, including
  full debug.Stack() frames with ABSOLUTE build paths (leaking the OS username +
  server filesystem layout) + where:/path:/runtime: lines. Real disclosure on a
  public error path; it's the error handler in every examples/server/*.
NOT FLIPPED (deliberate + test-pinned): TestWriteHTTPErrorWritesStructuredBody
  asserts the client body CONTAINS Formatted() (stack/where/runtime). Full detail
  is fine for a trusted/dev diagnostics endpoint but a leak on a public path — a
  deployment-context POLICY call, so per revert-if-tests-break I did NOT silently
  flip the default + rewrite its pinning test.
DONE (additive, non-breaking) + PINNED (diagnostics/TestFormattedPublicExcludesStackAndPaths):
  added Report.FormattedPublic() rendering only app-authored client-safe fields
  (summary, code/headline, next, docs), omitting where/path/error/runtime + all
  stack frames. Left a security NOTE on WriteHTTPError. RECOMMENDED owner fix: gated
  verbose mode (default client-safe via FormattedPublic, opt-in full via a flag),
  updating the pinning test to the verbose path. Building block is ready.

## Module 10 backlog follow-up (hotreload globals, 2026-07-05)
RESOLVED-AS-NON-ISSUE + DOCUMENTED (#57 sync part): the hotreload package globals
  live only in hotreload_wasm.go (//go:build js,wasm). wasm is single-threaded/
  cooperative and that file has NO goroutines/channels/awaits — every accessor
  (Install/Disable + JS bridge callbacks) touches the globals synchronously on the
  one main goroutine with no yield between read and write, so no data race. Adding a
  sync.Mutex would be cargo-cult false-safety on a hot path. Added a doc-comment
  block stating the single-threaded invariant + when it breaks. Native+wasm build
  GREEN. (The other half of #57, the silent restore-wedge, stays deferred — its fix
  conflicts with a test that pins skip-clear-on-malformed as intentional.)

## Module 4 backlog follow-up (css sink API parity, 2026-07-05)
FIXED + PINNED (#42): css sink public API diverged by target — StyleBlock/
  HarvestedClasses native-only, SeedFromDocument wasm-only — so portable code
  calling any failed to compile on the other target. Added native SeedFromDocument()
  (no-op), wasm HarvestedClasses() (registeredClasses() shared accessor) + wasm
  StyleBlock() (native-format, html.EscapeString'd attr). Both targets now expose the
  identical 7-func set. Pinned by css/parity_test.go — build-tag-free typed-value
  references that fail to compile if any func is target-only. Native + wasm GREEN.
  (Existing wasm CriticalCSS empty-attr left as-is; attr consistency in #84.)

## Module 12 backlog follow-up (runtime2 idempotency monotonicity, 2026-07-05)
FIXED + PINNED (#46): HandlePatchIdempotency deduped by version + rejected
  same-version conflicts but had NO monotonicity guard — a never-seen version below
  one already applied in the epoch (stale/reordered worker-bridge delivery) was
  applied, regressing the DOM with an older cumulative diff. Added a per-epoch
  high-water mark (getMaxPatchVersion): a new version < max is dropped idempotently
  (apply=false, no error); forward progress advances it; epoch change resets it.
  Pins: patch_idempotency_monotonicity_test.go. Full runtime2 suite GREEN.

## Module 12 backlog follow-up (runtime2 patch-identity verify, 2026-07-05)
CAPABILITY + PINNED (#45): patch_identity (deterministic hash of parts) is carried
  in the binary payload but TRUSTED on decode — a body corrupted/tampered across the
  worker postMessage boundary with an intact identity slips through. Added
  VerifyPatchStreamIdentity(raw): recompute from decoded parts, reject on mismatch,
  accept empty identity. Pinned (patch_identity_verify_test.go): canonical round-trip
  verifies (no false-reject), tampered body rejected, empty accepted. Suite GREEN.
NOT WIRED into ParseBinaryPatchPayload (deliberate): not all carried identities are
  canonical (hand-set ones would be recomputed-and-rejected → broken render), and
  buildPatchStreamIdentityFromParts branches on ops!=nil. Mandatory wiring must be
  scoped to a boundary proven to only receive canonical patches (the worker bridge),
  opt-in. Capability is ready.

## Module 4 backlog follow-up (router unreachable-route warning, 2026-07-05)
DONE + PINNED (#41 warning part): matchPattern returns the first registration-order
  match, so a pattern registered after a broader shadowing one is unreachable
  (/users/* or /users/:id before /users/new). Register now warns (ReportDiagnostic)
  when an earlier pattern shadows a newly-registered one. Subsumption logic in
  route_shadow.go is BUILD-TAG-FREE (matcher is js,wasm only) → natively unit-tested:
  patternShadows = earlier matches concrete-instance-of-later (params/wildcard →
  NUL sentinel tokens). Pinned by TestPatternShadows (11 cases). Native + wasm green.
NOT DONE (deliberate): specificity-sort precedence would break the registration-order
  contract (risky behavior change); the warning gives the safety signal without it.

## Module 15 backlog follow-up (html DataAttr/Data collision, 2026-07-05)
FIXED + PINNED (#91): Props.DataAttr (zero-alloc single data-*) and the Props.Data
  map could both target the same data-<name>. The fast-lane slice builder
  (toRuntimeCompactProps) appended BOTH → a DUPLICATE data-foo attribute, while the
  map builder let Data silently overwrite DataAttr — the two build paths disagreed.
  Fixed the slice path to skip DataAttr when Data holds the same key (Data wins, no
  duplicate), matching the map path. Non-colliding DataAttr still emitted. Pins:
  html/data_collision_test.go. Native + wasm GREEN. (#91 markdown depth cap done
  earlier; only the reflection-Props-merge perf item remains, deferred.)

## Module 15 backlog follow-up (logging struct redaction, 2026-07-05)
FIXED + PINNED (#88 struct blind spot, HIGH): normalizeLogValue collapsed a plain
  struct to fmt.Sprint — a string redaction couldn't decompose, so a struct with a
  Password/Token field leaked in full (fmt.Sprint even dumps unexported fields).
  Added a reflect.Struct case → buildStructValue: exported fields → map[string]any
  (json-tag keys, json:"-" opt-out, unexported skipped), recursively normalized, so
  RedactTelemetryValue matches sensitive field names. maxLogValueDepth=32 guard for
  cyclic structs. Scoped to the logging package (feeds telemetryredaction clean
  maps) — does NOT preempt #50's raw-struct policy. Pins: struct_redaction_test.go.
  Full library gate GREEN.
DEFERRED on #88: browser-console sink may emit raw fields (confirm it routes through
  RedactTelemetryValue); log MESSAGE string itself is never redacted (free-text
  redaction is a lossy policy call). Both need owner decisions.

## Module 10 backlog follow-up (virtualization restorationStore, 2026-07-05)
FIXED + PINNED (#58 unbounded-store part): restorationStore.snapshots had no delete
  — it grew one entry per unique (possibly dynamic) list ID forever; restoration
  must survive unmount so entries can't be dropped on unmount. Added a bounded LRU
  (maxRestorationSnapshots=256 + order slice + touchAndEvictLocked from both write
  sites); re-store moves to most-recent, oldest evicted past cap. Pinned by
  TestRestorationStoreIsBoundedLRU + ...ReStoreDoesNotGrowOrder. Suite GREEN.
DEFERRED on #58: (a) O(N)-per-scroll collectItemKeys rebuild — needs framework
  memoization on items-identity (slices aren't UseMemo-comparable); real perf item,
  not a safe drive-by. (b) anchor rounding drift — restore snaps scrollTop to
  anchorIndex*rowHeight, losing intra-row offset; fix changes the snapshot format.

## Module 12 backlog follow-up (tools/gwc test determinism, 2026-07-05)
#68 ADDRESSED: "vuln-scan flaky" was a non-issue (vuln_test.go parses a FIXED
  govulncheck stream — pure/deterministic). The real culprit was the build-variant
  test TestAgentBridgeReleaseArtifactHasNoAgentStrings (full `go build -tags
  production` js/wasm — transient toolchain failures = the observed flake); now
  testing.Short()-gated. Also gated writeAgenticReadFixture (runs `go mod tidy`,
  network-dependent) at the helper so all 4 fixture tests skip in -short. `go test
  -short ./tools/gwc` GREEN with the heavy tests skipped; still enforced in full CI.

## Module 11 backlog follow-up (api_compat_guard signature-aware, 2026-07-05)
FIXED + PINNED (#63 signature part): the compat guard recorded symbols NAME-ONLY
  (func Foo / field T.F / interface T.M), so a breaking signature or field-type
  change kept the same key and was INVISIBLE — a false-pass in the gate whose job
  is catching breaking changes. Added additive signature entries (funcsig/methodsig/
  fieldtype/interfacesig via normalizeSignature(exprString), generic-aware); name
  entries retained for removal-detection. Baseline regenerated: diff PURELY additive
  (0 real removals, +2179 sig entries); guard clean against it. Pinned by
  TestScanPackageCapturesSignaturesForBreakingChanges. Suite GREEN.
STILL OPEN on #63 (package drift): compareBaselines walks the baseline's packages,
  so a narrower -packages run than the baseline flags un-scanned packages as all-
  missing (false-fail), and a baseline with fewer packages than the scan silently
  skips extras. Fix = record+enforce scan scope, or derive scope from the baseline.

## Module 11 backlog follow-up (testkit traps re-examined, 2026-07-05)
#64 RE-EXAMINED — the four named traps are correct-by-design:
- loader-merge: defensive positional merge w/ bounds guard, cloned+mutex'd.
- ssr-parse: collectStructuredSnapshot walks the WHOLE tree → placement-agnostic;
  head tags found wherever the parser relocates them. ADDED pin
  TestStructuredSnapshotCollectsMetadataAnywhereInTree (metadata buried deep in
  body content) to lock the walk-everything guarantee vs a future head-only walk.
- dispatch: native !js stubs look like a false-pass but render.New() fatals
  natively, so no *Fixture is obtainable → stubs unreachable. (Can't pin: testing.TB
  is unmockable externally.)
- warning-dedup: excludes diagnostic-mirrored logs INTENTIONALLY (avoids
  double-count) — prevents a false-FAIL, not a false-pass.
No live defect; one robustness pin added. Any remainder is wasm-only helper
behavior needing the wasm executor, not a native drive-by.

## Module 11 backlog follow-up (doclint + errorcodes guards, 2026-07-05)
FIXED + PINNED (#65), both parts:
- errorcodes drift guard was SELF-REFERENTIAL: TestErrorCodeReferenceIsGenerated
  compares the committed page to RenderPage(ExtractCodes(...)) — both from the same
  extractor — so a regex bug dropping a code omits it from both sides and passes.
  Added TestExtractorMatchesIndependentLiteralScan: an INDEPENDENT naive scan of
  every quoted "GWC-<UPPER>" literal (minus the intentionally-skipped generic
  GWC-RUNTIME-PANIC) must all appear in ExtractCodes output. Two methods must agree.
- doclint ExtractKnownGwcFlags missed the *Var family (.StringVar/.BoolVar/… name is
  2nd arg; \.Var\( doesn't match `.StringVar(`) and .Uint64 — latent (gwc uses none)
  but a future StringVar flag would be flagged unknown in every doc citing it.
  Added typedVarFlagPattern + Uint64; additive to the known set (only removes false
  positives). Pinned by TestExtractKnownGwcFlagsFindsTypedVarAndUint64.

## Module 11 backlog follow-up (reprobuild wrong-artifact, 2026-07-05)
FIXED + PINNED (#66) — DONE: BuildWasmSHA built with only -trimpath -ldflags "-s -w",
  but the real release profile (tools/gwc release_build.go "release") also sets
  -buildvcs=false and -tags production. Missing -tags production compiled the DEV
  code paths (a different binary than ships); missing -buildvcs=false left VCS
  stamping (nondeterminism the release strips) — so the gate verified an artifact
  that is not what ships. Fixed via releaseWasmBuildArgs() matching the release
  profile; pinned by TestReleaseWasmBuildArgsMatchReleaseProfile (flag presence) so
  a revert can't silently reintroduce the gap. Real TestReproducibleWasmBuild now
  builds the counter twice with production+buildvcs flags → byte-identical. GREEN.

## Module 11 backlog follow-up (agenthub WS keepalive, 2026-07-05)
FIXED + PINNED (#67 keepalive part) — tools/agenthub/keepalive_test.go:
- After the hello handshake the read deadline was CLEARED, so a half-open peer left
  ReadMessage blocked forever → leaked session goroutine + stale hub.sessions entry.
  Added a ping/pong keepalive (agentPongWait/agentPingPeriod/agentWriteWait tunable
  vars): post-handshake arms a pongWait read deadline + pong handler that extends
  it; runFrameLoop pings on a ticker (gorilla WriteControl is concurrency-safe with
  the writeMu'd WriteMessage path) and refreshes the deadline on each inbound frame.
  Pins: unresponsive peer dropped within a few pongWait windows; responsive peer
  stays Active across 5x pongWait. agenthub is its own module (tools/agenthub/go.mod)
  — build/test from inside it. Suite GREEN.
STILL OPEN on #67 (history cap): per-session history is already bounded (event/log
  rings 512, recording ring 256), but hub.sessions is append-only — crashed/reloaded
  predecessors are retained for crash-report linkage and never pruned, so repeated
  hot-reloads grow it without bound. Fix = cap retained terminal sessions to the last
  K, pruning under hub.mu with care for ListSessions/predecessor/crash-report lookups.

## Module 11 backlog follow-up (apidump extractor accuracy, 2026-07-05)
FIXED + PINNED (#61) — internal/apidump/apidump_test.go (the tool had ZERO tests):
- The API-baseline extractor rendered exported vars/consts as NAME ONLY
  (`var Default` / `const StatusIdle`), dropping the type — so a BREAKING type
  change to an exported var/const (`var Default *Config` -> `*OtherConfig`,
  retyping an enum const) collapsed to the same line and slid past the gate
  silently, contradicting the package's "a breaking change can never land silently"
  guarantee. renderGen's ValueSpec branch now appends the EXPLICIT type when
  present; untyped decls (Type==nil, e.g. `var ErrX = ...` / untyped const) stay
  name-only (conservative — no go/types inference). Pins cover typed-capture,
  untyped-stays-bare, and the before/after var-type-change detection.
- Blast radius verified CONTAINED: only two goldens regenerated (anim, query);
  diffs are PURELY type additions to enum consts/typed easing vars
  (const StatusIdle Status, var Linear Easing, ...) — no API lines removed or
  reordered. Full library gate GREEN. NOTE: in an `iota` block only the first
  member carries the explicit type in the AST, so it anchors the block's type;
  later members stay name-only (acceptable — the block's type is still guarded).
