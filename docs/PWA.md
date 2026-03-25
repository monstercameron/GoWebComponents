# PWA and Offline App Support

This page defines the current Progressive Web App and offline-application support boundary for GoWebComponents.

Use it when deciding how web app manifests, service workers, offline caches, mutation replay, installability, and deployment concerns should fit around the current runtime and browser-support model.

## At A Glance

- The `pwa` package already ships a real companion surface for manifests, installability observation, explicit service-worker registration, release-manifest-driven asset planning, Cache Storage coordination, and structured PWA diagnostics.
- PWA support in this repo is integration-oriented: the framework provides explicit helpers, but applications still own service-worker code, route fallback policy, retention policy, and deployment rollout.
- Durable offline writes are first-class through `fetch.OpenMutationQueue(...)`, while broader PWA state can be inspected through `pwa.InspectDiagnostics(...)`.
- Use [OFFLINE_MUTATIONS.md](OFFLINE_MUTATIONS.md) for replay semantics, [ASSETS.md](ASSETS.md) for cache and manifest alignment, and [BROWSER_SUPPORT.md](BROWSER_SUPPORT.md) for capability-gated behavior.

## Quick API Chooser

Use this route when the question is about:

- web app manifest generation and validation: `pwa.Manifest`, `Manifest.Validate()`, `pwa.MarshalManifestJSON(...)`
- install prompts and browser installability state: `pwa.ObserveInstallability(...)`
- explicit service-worker registration, waiting-worker control, and controller-change reload behavior: `pwa.RegisterServiceWorker(...)`
- aligning cache namespaces and precache entries with a wasm release record: `pwa.ParseWasmReleaseManifestJSON(...)`, `pwa.BuildServiceWorkerAssetPlan(...)`, `pwa.BuildCacheStoragePlan(...)`
- applying or inspecting Cache Storage state from application code: `pwa.OpenCacheStorageManager()`
- one structured snapshot across manifest, installability, service worker, Cache Storage, and offline queue state: `pwa.InspectDiagnostics(...)`

## Current Shipped Slice

What is already real in this repo today:

- manifest normalization, validation, and JSON marshaling
- installability observation and explicit prompt ownership
- explicit service-worker registration plus lifecycle inspection helpers
- release-manifest parsing and service-worker asset planning helpers
- Cache Storage planning, sync, and inspection helpers
- structured PWA diagnostics snapshots, including offline mutation queue adaptation

What remains deliberately application-owned:

- the service-worker implementation itself
- route-level offline fallback policy
- security and retention decisions for durable browser data
- rollout decisions such as refresh prompting, logout purge behavior, and cross-tab replay ownership

## Example Shape

```go
manifest := pwa.Manifest{
	Name:      "Atlas",
	ShortName: "Atlas",
	StartURL:  "/app/",
	Scope:     "/app/",
	Display:   pwa.ManifestDisplayStandalone,
	Icons: []pwa.ManifestImage{{
		Src:   "/assets/icon-192.png",
		Sizes: "192x192",
		Type:  "image/png",
	}},
}

manifestJSON, err := pwa.MarshalManifestJSON(manifest)
if err != nil {
	return err
}

registration, err := pwa.RegisterServiceWorker(ctx, pwa.ServiceWorkerOptions{
	URL:   "/sw.js",
	Scope: "/app/",
})
if err != nil {
	return err
}

snapshot, err := pwa.InspectDiagnostics(ctx, pwa.DiagnosticsOptions{})
if err != nil {
	return err
}

_ = manifestJSON
_ = registration.Snapshot()
_ = snapshot
```

That example is intentionally narrow: the framework owns the helpers and typed snapshots, while the application still owns the actual service-worker script, precache rules, and offline route policy.

## Support Boundary

The PWA scope is integration-oriented, not a hidden runtime feature.

The current project direction is:

- document clear integration points for manifests, service workers, offline caching, and installability
- keep the core rendering and hydration runtime independent from service-worker lifecycle logic
- treat offline mutation replay, cache invalidation, and static asset delivery as related but distinct concerns

The framework does not claim:

- a built-in service-worker runtime
- automatic app-manifest generation from app code
- one-click offline enablement for every app

Applications should choose PWA features deliberately instead of assuming every GoWebComponents app is installable or offline-ready by default.

## Web App Manifest Boundary

Installability metadata belongs in a manifest pipeline, not in the rendering runtime itself.

The current model is:

- applications provide a web app manifest with app name, icons, start URL, display mode, theme color, and related metadata
- manifest generation or templating may live in companion tooling rather than in core rendering APIs; the first shipped helper surface now lives in the `pwa` package through `pwa.Manifest` plus `pwa.MarshalManifestJSON(...)`
- SSR, prerender, or static-hosted HTML should reference the manifest explicitly through ordinary head markup

The project may still grow additional manifest tooling later, but the current contract already includes the shipped manifest helper surface rather than only a future boundary.

Durable browser persistence now has a defined first-party seam through `interop.OpenPersistentStore(...)` plus the cache, queue, and state helpers layered on top of it. The current storage model is intentionally narrow: one logical feature store maps to one IndexedDB object store with string keys and JSON-friendly string payloads, database-version bumps own schema creation, blocked upgrades surface structured diagnostics, quota failures stay typed, and corruption recovery is opt-in so reconstructible caches can reset safely without silently deleting mutation or state data.

## Installability Helpers

Installability should stay explicit as well.

The current first-party surface is:

- `pwa.ObserveInstallability(...)` for browser-side installability observation
- manifest validation through `pwa.Manifest.Validate()` or the state exposed by `ObserveInstallability(...)`
- `InstallabilityManager.Prompt(...)` for explicit `beforeinstallprompt` ownership
- `InstallabilityManager.Subscribe(...)` for `beforeinstallprompt` and `appinstalled` state updates

The helper reports installability state, manifest validation results, prompt availability, installed status, and user-visible reasons the browser has not exposed installation yet. It does not force installation UI or guess hidden browser heuristics beyond the events and secure-context signals the platform actually exposes.

## Service-Worker Integration Story

Service workers should be treated as an application-owned deployment concern with documented framework touchpoints.

The current integration story is:

- register the service worker from explicit client bootstrap code, not from hidden framework side effects
- use `pwa.RegisterServiceWorker(...)` when you want a first-party helper for explicit registration, waiting-worker inspection, `skipWaiting` signaling, and controller-change refresh coordination without burying ownership inside unrelated runtime code
- parse `wasm-release-manifest.json` through `pwa.ParseWasmReleaseManifestJSON(...)` and derive cache namespaces plus precache inputs through `pwa.BuildServiceWorkerAssetPlan(...)` so service-worker revisions track the same release record as the emitted wasm artifact
- precache shell assets, wasm artifacts, and other immutable build outputs through service-worker tooling or application-owned registration logic
- keep cache naming and versioning aligned with the same asset manifest and hashing rules documented in [ASSETS.md](ASSETS.md)
- keep service-worker scope, update flow, and fallback behavior explicit in the application instead of coupling them to route rendering

This keeps the runtime small while still making PWA integration a first-class documented path.

## Offline Caching Strategies

Offline behavior should separate shell delivery, static assets, data reads, and mutations.

The current recommended cache split is:

- app shell and route HTML: cache for repeat visits and offline fallback only when the deployment model supports serving those documents safely
- immutable static assets: cache aggressively by hashed filename
- route or API data: cache deliberately with app-owned freshness rules instead of assuming offline persistence is always correct
- queued writes: use the offline mutation queue where durable replay is needed, rather than conflating write replay with read caching

Recommended model:

- use immutable asset caching for wasm, JS helpers, CSS, and fingerprinted media
- use explicit offline fallback documents or cached shells for routes that should still open without network
- prefer revalidation-aware read caching over silent indefinite API caching
- use `fetch.OpenMutationQueue(...)` for durable browser-side write replay when offline writes matter

The current first-party Cache Storage companion surface is:

- `pwa.BuildCacheStoragePlan(...)` for release-scoped cache namespaces and explicit per-entry strategy assignment
- `pwa.OpenCacheStorageManager()` plus `Sync(...)` and `Inspect(...)` for applying and inspecting browser Cache Storage state

The current recommended default strategy split is:

- shell HTML and offline fallback documents: `network-first` or `stale-while-revalidate`, depending on how aggressively the application wants repeat-visit freshness
- hashed wasm, JS helpers, CSS, and fingerprinted media: `cache-first`
- non-hashed or correctness-sensitive route data: not part of this helper surface; keep those under app-owned fetch or service-worker logic instead of pretending Cache Storage alone solves data freshness

## Diagnostics And Devtools Visibility

PWA state should be inspectable through structured snapshots, not inferred from ad hoc browser-console output.

The current first-party diagnostics surface is:

- `pwa.InspectDiagnostics(...)` for one structured snapshot covering manifest validation, installability state, service-worker lifecycle, Cache Storage contents, offline queue summary, and browser storage-pressure signals
- `pwa.MutationQueueDiagnosticsSource(...)` for adapting `fetch.MutationQueue` into that snapshot without coupling the generic diagnostics API directly to wasm-only queue types

The current snapshot is intended to answer:

- whether the manifest is currently valid for installability
- whether install prompts are available or the app is already installed
- whether a controller, waiting worker, or active worker is present and under which scope
- which versioned Cache Storage namespace and entries are currently resident
- how many offline mutations are queued, retrying, or dead-lettered
- how much browser storage is currently used, how much quota remains, whether persistence has been granted, and how much of the reported usage is attributed to IndexedDB or Cache Storage when the browser exposes those details

Applications may surface this snapshot through their own diagnostics drawer, devtools panel, or support bundle flow. The framework keeps the data structured and typed; applications decide how aggressively to poll it and where to render it.

## Offline Route Opening And Fallbacks

Offline route behavior should be explicit per route family.

The route-opening rules should be:

- routes that can render safely from a cached shell plus durable client data may open offline through an application-owned fallback document or shell route
- routes that require authoritative online data for correctness should fail closed with an explicit offline-unavailable experience instead of rendering misleading stale content
- deep links should resolve through the same router ownership model as online opens; the service worker may provide a shell or fallback document, but route policy still belongs to the application
- fallback documents should preserve enough bootstrap or route context to show a useful offline state, retry action, and navigation back to known-openable surfaces
- applications should distinguish “offline-openable with stale data” from “offline-openable empty shell” and from “online required” rather than treating every route as one generic offline case

Recommended split:

- shell-capable routes: marketing pages, settings shells, draft-centric workflows, and other surfaces that can recover from cached shell plus later revalidation
- partial offline routes: routes that may open the shell and local state but must mark some panels or actions unavailable until the network returns
- online-required routes: privileged or correctness-sensitive flows where stale or partial rendering would be misleading or unsafe

## Update And Invalidation Semantics

PWA update behavior must keep old clients from running half-old asset graphs.

The current rules should be:

- version shell assets, wasm artifacts, JS helpers, and CSS through the same hashed-asset or manifest strategy used outside the service worker
- treat HTML documents and manifest-like entrypoints as short-lived so they can point clients at the newest immutable assets
- when a new service worker activates, make stale caches disposable as one versioned unit instead of mixing assets across releases
- if an update changes the active wasm or JS bootstrap set, the application should prefer prompting for refresh or safe reload rather than silently continuing on a mismatched graph

Offline caches should be invalidated by release version or manifest revision, not by ad hoc per-file guessing.

## Durable Data Security And Retention

Durable offline data needs explicit retention rules, not just storage APIs.

The current rules should be:

- persist only data that is already acceptable in browser-visible storage and developer tools
- treat read caches as reconstructible copies; they may be reset aggressively on corruption, logout, user switch, or trust-boundary changes
- treat offline mutation queues and durable state snapshots as app-owned data with explicit purge semantics; do not silently preserve them across account changes unless the application has confirmed that is correct for the current user and device
- do not persist bearer tokens, session secrets, CSRF secrets, raw cookies, or server-only policy results in caches, queues, or snapshots
- attach `MaxAge`, queue cleanup, or application-owned purge workflows to every durable store that should not survive indefinitely
- make logout, session-expiry, and user-switch flows clear durable data immediately when that data should no longer be readable on the device
- prefer deriving volatile auth headers or privileged replay context at execution time instead of storing them in durable payloads

Recommended application split:

- safe to persist: public route data, user-visible drafts that are intentionally recoverable, presentation preferences, and other JSON-shaped values already acceptable in client storage
- usually not safe to persist: secrets, privileged moderation context, private internal policy decisions, or anything whose mere presence on disk is a security issue
- requires explicit product policy: queued writes or snapshots that may contain user-created but sensitive business data; these need domain-specific retention windows, logout purge behavior, and potentially encryption owned by the application

## Cross-Tab Replay And Cache Coordination

Durable offline work should not be replayed by every open tab at once.

The coordination rules should be:

- use one logical cross-tab topic for offline replay ownership and one for cache invalidation or replay results when an application opens multiple tabs for the same surface
- elect one active replay owner at a time; other tabs may observe queue state and results, but should not stampede the same queued writes concurrently
- treat logout, session expiry, and trust-boundary narrowing as immediately authoritative across tabs; apply those messages before any further replay or cache reuse
- broadcast cache invalidation and replay outcomes after authoritative success so sibling tabs can refresh or dispose their own local state deliberately
- when tabs disagree about freshness, prefer the narrowest safe action such as invalidation or reload over optimistic cross-tab replacement of richer local state

The existing `interop.OpenCrossTabChannel(...)` surface is the intended transport for this policy. The framework now defines the ownership model; applications still decide which queue, cache keys, and auth or route transitions should be coordinated.

## Production Deployment Guidance

PWA deployments add operational requirements beyond ordinary static hosting.

Recommended rules:

- serve over HTTPS for service-worker registration and installability
- ensure service-worker scope matches the intended route base path
- keep cache headers aligned with the asset strategy: immutable for hashed assets, short-lived for HTML and manifest-like entrypoints
- validate CDN or reverse-proxy behavior so service-worker scripts, manifests, and offline fallbacks are not cached incorrectly
- treat failed service-worker rollout, stale caches, and offline fallback loops as release risks, not as edge cases

PWA deployments should be validated together with:

- [ASSETS.md](ASSETS.md)
- [BROWSER_SUPPORT.md](BROWSER_SUPPORT.md)
- [OFFLINE_MUTATIONS.md](OFFLINE_MUTATIONS.md)

## Current Offline-First Reference Slice

The repo does not currently hide all offline-first behavior inside one magic starter. The current supported reference shape is a small coordinated slice across the focused PWA examples.

Use the current trio as one coherent reference topology:

- `examples/97-pwa-installability`: manifest wiring, installability observation, explicit service-worker registration, and update-prompt ownership
- `examples/97-pwa-offline-cache`: shell warmup, offline fallback documents, durable queued writes, replay, conflict-aware replay handling, Background Sync fallback, and structured PWA diagnostics
- `examples/97-pwa-multi-client`: multi-tab ownership, cache invalidation fanout, peer discovery, and shared service-worker or cache visibility across sovereign wasm tabs

Treat them as one intended product shape:

- one installable shell with an explicit manifest and service-worker scope
- one offline-capable route family that can open from a cached shell and durable local state
- one authoritative mutation queue with explicit replay and conflict handling
- one cross-tab ownership channel for replay leadership, invalidation, logout propagation, and support diagnostics
- one purge path that clears queue, cache, and app-owned retained state when the user or trust boundary changes

This is the current offline-first reference app shape for the repo even though it is still presented as three focused examples instead of one monolithic demo.

## Offline-First Sync Model

The framework should now be read as supporting one explicit offline-first sync model beyond raw queue replay, even though applications still own domain payloads and final authority decisions.

Recommended model layers:

- local optimistic state in atoms, cached resources, or app-owned view state
- durable mutation intent in `fetch.OpenMutationQueue(...)`
- reconstructible read state in Cache Storage, persistent fetch cache, or typed bootstrap seeds
- reconnect and replay ownership coordinated across tabs through `interop.OpenCrossTabChannel(...)`
- one support-facing diagnostics surface through `pwa.InspectDiagnostics(...)`

Recommended lifecycle:

1. accept a local user intent into app-owned optimistic state
2. enqueue the authoritative server write into the durable mutation queue
3. mark the affected entity or workflow as locally pending instead of pretending the server already accepted it
4. replay from one elected owner when connectivity and auth state allow it
5. on success, invalidate or refresh reconstructible read state and clear the pending sync marker
6. on failure, branch into retry, deferred replay, conflict review, or purge depending on policy and trust boundary

Recommended ownership rules:

- the queue owns durability, ordering, retry timing, and dead-letter visibility
- the application owns idempotency keys, auth attachment, optimistic UI, merge strategy, and user-visible repair workflows
- Cache Storage and persistent read caches stay reconstructible; they should not become the source of truth for pending writes
- cross-tab coordination owns replay leadership and invalidation fanout, not hidden state replication

This keeps the current offline story coherent: queue for intent durability, caches for reconstructible reads, cross-tab channels for leadership, and diagnostics for supportability.

## Conflict Resolution And Operator Recovery

Durable replay is not production-ready if every conflict collapses into "retry later" or "show one generic error."

Recommended conflict inputs:

- an idempotency or deduplication key that lets the server recognize repeated intent safely
- the local revision or base version the user last edited against
- an authoritative server conflict response that distinguishes duplicate acceptance, stale base revision, permission loss, and validation failure
- enough app-owned metadata to tell whether a queued intent is still current or has already been superseded by a newer local draft

Recommended durable-replay branches:

- duplicate already accepted: mark the mutation successful locally and invalidate affected read state
- stale or conflicting revision: move the mutation into conflict review instead of silently retrying forever
- superseded local intent: retire the older queued mutation in favor of the newer local draft and keep that decision visible in diagnostics
- permission or trust-boundary loss: block replay, purge or quarantine the entry as product policy requires, and surface a user-visible re-auth or ownership message
- validation failure: move the entry into an operator or user repair path instead of treating it as transient network retry

Recommended operator-recovery model:

- keep conflict records visible through diagnostics and app-owned review screens
- show the local payload summary, the remote revision or reason, and the next available action
- support deliberate actions such as retry unchanged, discard local change, requeue with merge, or open a repair workflow
- record the resolution path so later diagnostics can explain why a queued mutation disappeared, was retried, or was replaced

## Merge Policies And Reconnect Reconciliation

Reconnect should not mean "replay everything blindly, then hope the server sorts it out." The policy needs to be explicit before queued writes resume.

Recommended reconciliation order:

1. re-establish auth and trust-boundary state
2. elect one replay owner if more than one tab is open
3. refresh any lightweight authoritative revision or lock metadata needed to classify queued mutations safely
4. apply the configured merge or conflict policy per entity or mutation family
5. replay only the mutations that remain valid under that policy
6. invalidate or refresh reconstructible reads after authoritative outcomes land

Recommended policy families:

- `server-wins`: discard or quarantine the local mutation when the server revision already moved and local overwrite is not allowed
- `client-wins`: reissue the local mutation intentionally, usually only when the server treats the client write as authoritative and idempotent
- `revision-aware reject`: stop replay and surface conflict review when the local base revision is stale
- `field-level merge`: synthesize a new mutation draft from compatible local and remote fields, then replay that explicit merged payload
- `operator-reviewed`: hold replay until a human picks discard, merge, or overwrite

Recommended scoping rules:

- choose policy per mutation family or entity type, not one global repo-wide default
- keep merge execution application-owned even when the framework documents the policy names and reconnect order
- prefer reject or operator-reviewed flows over silent field-level merge unless the domain already has a defensible merge contract

## Per-Entity Sync Health

Offline-capable products need more than one queue size. They need a small shared vocabulary for the sync health of each tracked entity, draft, row, or form.

Recommended entity states:

- `clean`: local and authoritative state are aligned
- `pending`: a local intent exists and is queued but not currently replaying
- `replaying`: a replay owner is actively attempting the queued mutation
- `conflicted`: replay stopped because authoritative state, validation, or policy now requires review
- `blocked`: replay is intentionally paused because auth, permissions, trust boundary, or operator policy does not allow execution
- `stale`: local read state is usable but known to need refresh or invalidation before it should be trusted as current

Recommended inspection shape per entity:

- stable entity or draft id
- current sync state
- last attempted replay time and last authoritative outcome
- owning queue or mutation family
- replay owner identity when multi-tab leadership matters
- conflict or block reason when the state is not `clean`

Recommended usage:

- surface the state at row, record, or form scope instead of collapsing everything into one global banner
- let queue-level diagnostics roll up counts from these entity states, not replace them
- prefer showing `stale` separately from `pending` so read freshness and write durability are not conflated

## Conflict-Oriented UI Workflows

Offline-first apps should treat conflict and replay repair as first-class workflows, not as raw queue dumps.

Recommended UI patterns:

- conflict banner: a lightweight summary when one or more entities moved into `conflicted` or `blocked`
- per-record repair screen: a focused view that compares local intent, authoritative state, and the available next actions
- replay review queue: an operator-facing list of queued or dead-lettered mutations that need approval, discard, or merge
- reconnect summary: a short post-reconnect report that explains what replayed cleanly, what stayed blocked, and what now needs review

Recommended interaction rules:

- keep the queue transport and the UI workflow separate; the queue stores intent, but the app chooses how and where to render repair actions
- link UI actions back to explicit queue outcomes such as discard, requeue with merge, retry, or open a record-level editor
- show conflict reasons in domain language when possible, while preserving the structured technical cause for diagnostics and support
- prefer narrow per-record repair over one giant modal whenever only a few entities are conflicted

## Long-Lived Offline Data Operations

Durable offline support reaches production readiness only when long-lived storage and replay backlogs have an explicit operating model.

Recommended operational rules:

- monitor storage pressure through `pwa.InspectDiagnostics(...)` and degrade before the browser starts evicting data unpredictably
- expire stale queued writes and repair records by product-owned age windows instead of letting abandoned intents accumulate forever
- treat replay backlogs after long disconnects as a controlled recovery event: re-check auth, refresh lightweight authoritative metadata, and then resume replay under the documented merge policy
- purge queue, cache, and durable state immediately on logout, user switch, tenant switch, or narrower trust-boundary changes when the retained data is no longer safe for the active identity
- treat corruption recovery for reconstructible caches differently from mutation queues; cache reset may be automatic, but queue or repair-state deletion should remain explicit and observable

Recommended support checks after long offline periods:

- can the app distinguish transient backlog growth from permanently blocked replay
- can support inspect which queue or entity states are consuming the most retained storage
- can the product tell the user whether local data was replayed, discarded, expired, or purged because the identity boundary changed
- can the app re-warm reconstructible reads after a purge without pretending durable writes were preserved

## Current Boundary

This document defines the current integration contract only.

It does not yet claim:

- a built-in service-worker runtime or precache strategy

Those remain separate backlog work.

## Review Checklist

- does the page clearly separate shipped `pwa` helpers from the still application-owned service-worker and route-policy logic
- are cache, queue, and retention recommendations grounded in current public APIs instead of generic PWA advice
- does the page keep durable offline writes, read caching, and installability as separate concerns rather than collapsing them into one feature
- are security, logout purge, and cross-tab replay ownership treated as explicit product decisions rather than automatic framework behavior
