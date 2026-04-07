# Plugin Framework Plan

Status: draft for review with a partial implementation now landed in `internal/pluginruntime`, `devtools`, `internal/runtime/plugininterposer.go`, `fetch/plugininterposer.go`, `router/plugininterposer.go`, and `internal/runtime2/plugininterposer.go`

Audience:

- framework maintainers
- companion-package authors
- future first-party plugin authors

This document proposes a new core plugin framework for GoWebComponents that is generic enough to support multiple deep plugin categories while using `devtools` as the first serious plugin consumer.

It remains a design plan first, but parts of the kernel and the initial devtools integration have now been implemented. Where the implementation is narrower than this document, the code should be treated as the current source of truth.

## Why This Exists

The current `plugin` package is an explicit, application-owned companion host. That is useful for app-level extension and example integrations, but it is not the right abstraction for plugins that need deep, live, framework-owned access to runtime state, browser interaction, lifecycle hooks, and command surfaces.

The current gaps are:

- the existing `plugin.Host` is callback-oriented and app-owned rather than core-owned
- the current devtools bridge snapshots host contributions instead of resolving them live
- there is no generic kernel for trusted framework plugins
- there is no strong fault boundary that prevents plugin panics from cascading into framework behavior
- there is no typed service model for deep read and command access into runtime, DOM, router, fetch, SSR, forms, workers, storage, or diagnostics
- there is no central API for high-volume devtools data such as event streams, network timelines, DOM snapshots, or replay artifacts

This plan proposes a new plugin kernel that addresses those gaps without collapsing all extension points into one weak, untyped API.

## Core Goals

- support one generic plugin framework that can host multiple plugin categories
- allow trusted plugins to reach deeply into framework subsystems through typed service APIs
- make plugin calls non-fatal to the core framework
- keep deep plugin access strongly shaped and auditable
- let `devtools` become the first kernel plugin without making it a one-off special case
- preserve a distinction between app-owned companion extensions and core-owned framework plugins
- support additive rollout instead of a flag day rewrite
- support high-signal devtools workflows such as full DOM inspection, event tracing, request before-and-after inspection, worker traffic, storage observation, and replay

## Non-Goals

- no immediate commitment to a public semver-stable deep plugin API
- no arbitrary plugin mutation of framework internals
- no raw `internal/*` pointers crossing the plugin boundary
- no dynamic code loading or external marketplace story in the first pass
- no promise that third-party plugins will be supported before the SPI proves itself with first-party consumers
- no requirement that the first-pass devtools plugin matches Chrome DevTools feature-for-feature

## Current State

Today there are effectively two extension stories:

- app-owned companion extensions via `plugin.Host`
- app-owned devtools state injection via `devtools.SetExtensionSections(...)` and `devtools.SetErrorOverlayActions(...)`

Those are useful, but they are not sufficient for deep framework plugins because:

- they are not owned by core runtime bootstrap
- they rely on ad hoc callback registration
- they do not provide a generic service lookup model
- they do not enforce plugin isolation centrally
- they do not define plugin lifecycle, health, or quarantine rules
- they do not expose enough typed data for richer devtools use cases such as DOM, event, or network inspection

## Proposed Architecture

The plugin system should have four layers.

### 1. Plugin Kernel

The kernel owns:

- plugin manifest validation
- capability and version checks
- startup and shutdown lifecycle
- service lookup
- contribution registration
- health tracking
- panic recovery
- plugin disable and quarantine behavior

This should be core-owned, not app-owned.

Suggested package placement for the first pass:

- `internal/pluginruntime`

Possible subpackages later:

- `internal/pluginruntime/kernel`
- `internal/pluginruntime/services`
- `internal/pluginruntime/contrib`

### 2. Stable Plugin Service Contracts

The kernel exposes typed, read-focused and command-focused service interfaces for framework subsystems.

Initial service families:

- runtime inspection
- DOM inspection and DOM commands
- event stream inspection
- network and async-data inspection
- cache storage and asset inspection
- router inspection and commands
- diagnostics stream
- profiling stream
- SSR and hydration inspection
- forms inspection and commands
- worker inspection and message tracing
- storage inspection and observation
- overlay and accessibility inspection
- logging and structured event sink
- capture, timeline, and replay

These service contracts are the only approved way for deep plugins to interact with the framework.

### 3. Interposer Layer

The interposer layer sits between the stable plugin service contracts and the unstable implementation details of the active runtime, platform adapter, router implementation, fetch/cache machinery, and service worker or asset stack.

The interposer layer owns:

- adapting runtime-specific data into stable plugin snapshots
- translating plugin commands into runtime-specific operations
- hiding `runtime` versus `runtime2` differences
- hiding browser platform adapter churn
- enforcing capability, ownership, and policy checks before commands reach implementation code
- normalizing event, network, asset, and diagnostics records across implementation variants

Core design rule:

- plugins talk to service contracts
- the kernel talks to interposers
- interposers talk to implementation details

Plugins should never know whether the active backing implementation is:

- `internal/runtime`
- `internal/runtime2`
- one DOM adapter or another
- one fetch/cache implementation or another

That is the interposer's job.

### 4. Typed Contribution Contracts

Plugins do not mutate core by reaching into internals directly. They register typed contributions that the owning subsystem resolves.

Initial contribution families:

- devtools views
- devtools panels
- devtools sections
- devtools actions
- devtools timeline lanes
- devtools inspectors
- theme and style contributions
- DOM patch contributions
- security inspection contributions
- security policy contributions
- cache policy contributions
- asset management contributions
- route guards
- route observers
- request observers
- form validators
- head contributions
- bootstrap contributions

The lifecycle is generic. The services are typed. The contributions are subsystem-specific.

That split is the main design rule.

## Trust Model

The framework should distinguish between trust levels even if the first pass only enables one.

Suggested trust levels:

- `first_party`
- `trusted_internal`
- `untrusted`

First-pass recommendation:

- implement `first_party`
- optionally allow `trusted_internal`
- reject `untrusted` until the SPI, sandbox, and compatibility rules are mature

Why trust matters:

- deep read access still reveals sensitive runtime state
- command APIs can affect navigation, cache, hydration recovery, or user-visible app state
- event and network tracing can expose secrets if the service layer is not careful
- security plugins may be allowed to observe or enforce policy around requests, storage, and exports
- panic isolation is necessary but not sufficient for safe plugin behavior

## Plugin Packaging, Discovery, And Enablement

The architecture needs an explicit operational story for how plugins get into the kernel.

First-pass recommendation:

- plugins are compiled and linked into the app or framework build
- core bootstrap passes an explicit registration list into the kernel
- plugin enablement is controlled by manifest ID and startup configuration
- plugin disablement is possible per ID without recompiling unrelated services
- no filesystem or network auto-discovery in the first pass

Suggested registration shape:

```go
type PluginRegistration struct {
	Manifest Manifest
	Create   func() Plugin
}

type KernelBootstrapOptions struct {
	Plugins          []PluginRegistration
	EnabledPluginIDs []string
	DisabledPluginIDs []string
}
```

Operational rules:

- the kernel should log why a plugin was enabled, skipped, or rejected
- duplicate plugin IDs should fail bootstrap deterministically
- disabled plugins should not start, subscribe, or register contributions
- startup configuration should be able to disable one bad plugin without disabling the kernel

## Distribution And Approval Policy

The document also needs to say what distribution model is actually in scope.

Recommended policy:

| Trust Level | Delivery Model | Process Model | First-Pass Status |
| --- | --- | --- | --- |
| `first_party` | compiled into framework or app build | in-process | supported |
| `trusted_internal` | compiled into controlled internal builds | in-process | optional |
| `untrusted` | not supported yet | not applicable | rejected |

Explicit non-goals for the first pass:

- marketplace-style plugin installation
- runtime download and execution of plugin code
- out-of-process plugin RPC
- user-supplied plugin binaries

That keeps the initial kernel aligned with the trust and interposer model instead of pretending it is already a public plugin marketplace.

## Plugin Manifest

The manifest should be stronger than the current companion manifest because the kernel owns compatibility and health.

Suggested fields:

```go
type Manifest struct {
	ID                string
	Version           string
	APIVersion        string
	Description       string
	Provider          string
	Trust             TrustLevel
	RequiredServices  []ServiceKey
	OptionalServices  []ServiceKey
	ContributionKinds []ContributionKind
	ActivationPolicy  ActivationPolicy
	DependsOn         []string
	StartPriority     int
	Experimental      bool
}
```

Intent of the key fields:

- `ID`: stable plugin identity
- `Version`: plugin implementation version
- `APIVersion`: plugin-kernel compatibility contract
- `Trust`: what level of framework access the plugin may request
- `RequiredServices`: services that must resolve for startup to succeed
- `OptionalServices`: services that may be absent on one backend or one host
- `ContributionKinds`: contribution families this plugin may register
- `ActivationPolicy`: whether this plugin is boot, view, session, or opportunistic
- `DependsOn`: future dependency graph support
- `StartPriority`: deterministic ordering where ordering matters

## Generic Plugin API

The kernel should define a small lifecycle interface.

Suggested shape:

```go
type Plugin interface {
	Manifest() Manifest
	StartPlugin(getContext PluginContext) (PluginHandle, error)
}

type PluginHandle interface {
	StopPlugin(getContext context.Context) error
}
```

The context is where the real power lives.

```go
type PluginContext interface {
	GetKernelInfo() KernelInfo
	GetServiceResolver() ServiceResolver
	GetDiagnosticsService() DiagnosticsService
	RegisterContribution(getRegistration ContributionRegistration) error
	RegisterCleanup(getCleanup CleanupFunc)
	ReportHealth(getReport HealthReport)
}
```

Important constraints:

- plugin startup should be the only place that registers contributions
- contributions should be immutable registrations, not arbitrary mutation hooks
- long-running work should use explicit subscriptions or background sessions that core can stop

### Core API Design Rules

The API should be designed around a few hard rules so it stays flexible without becoming vague.

- lifecycle is small and typed
- services are stable, typed, and capability-gated
- contributions are declarative registrations with typed execution methods
- commands are explicit and auditable
- snapshots and streams carry metadata about freshness, truncation, and source backend
- optional richer capabilities are negotiated rather than assumed

### Base Kernel Types

The kernel needs a few common types that keep the service APIs coherent.

```go
type ServiceKey string

type ContributionKind string

type ExecutionClass string

const (
	ExecutionClassHot        ExecutionClass = "hot"
	ExecutionClassWarm       ExecutionClass = "warm"
	ExecutionClassBackground ExecutionClass = "background"
)

type ActivationKind string

const (
	ActivationKindBoot          ActivationKind = "boot"
	ActivationKindView          ActivationKind = "view"
	ActivationKindSession       ActivationKind = "session"
	ActivationKindOpportunistic ActivationKind = "opportunistic"
)

type ActivationPolicy struct {
	Kind      ActivationKind
	ViewID    string
	SessionID string
}

type KernelInfo struct {
	KernelVersion  string
	BackendID      string
	BackendVersion string
	APIVersion     string
}

type QueryBudget struct {
	MaxItems      int
	MaxBytes      int
	Timeout       time.Duration
	AllowSampling bool
}

type SnapshotMeta struct {
	BackendID  string
	ObservedAt time.Time
	Truncated  bool
	Partial    bool
	Cursor     string
}

type Subscription interface {
	StopSubscription(getContext context.Context) error
}

type CleanupFunc func(getContext context.Context) error
```

### Service Resolver

The kernel should expose typed service access for common cases and still leave room for optional service families over time.

```go
type ServiceResolver interface {
	GetRuntimeInspectService() RuntimeInspectService
	GetDOMService() (DOMService, bool)
	GetStyleService() (StyleService, bool)
	GetEventService() (EventService, bool)
	GetRouterService() (RouterService, bool)
	GetNetworkService() (NetworkService, bool)
	GetAssetService() (AssetService, bool)
	GetSSRInspectService() (SSRInspectService, bool)
	GetFormsService() (FormsService, bool)
	GetWorkerService() (WorkerService, bool)
	GetStorageService() (StorageService, bool)
	GetSurfaceService() (SurfaceService, bool)
	GetCaptureService() (CaptureService, bool)
	GetSecurityService() (SecurityService, bool)
	HasService(getKey ServiceKey) bool
}
```

This keeps core services ergonomic while preserving the optional-capability model.

## Interposer Model

The interposer model is the main architectural mechanism that keeps the plugin layer clean while runtime internals remain in flux.

The plugin API should stabilize around service contracts and contribution contracts.

The implementation-facing side should stay internal and replaceable.

### Why The Interposer Exists

The repo currently has active architectural movement around `internal/runtime` and `internal/runtime2`.

That means the plugin framework should not:

- bind directly to runtime structs from either implementation
- leak runtime-owned node, scheduler, or commit internals into plugin APIs
- force plugins to care which runtime is active
- freeze unstable runtime details just because plugins need deep access

The interposer layer solves that by acting as a compatibility membrane.

### Interposer Responsibilities

The interposer layer should:

- gather implementation-specific data
- convert it into stable plugin-facing snapshots
- route plugin commands into the active implementation
- reject commands that violate ownership, capability, or policy rules
- normalize errors, events, and diagnostics
- absorb implementation churn so plugin code and kernel logic stay stable

### Interposer Boundaries

Stable side:

- plugin manifests
- kernel lifecycle
- service contracts
- contribution contracts
- health and diagnostics model

Unstable side:

- runtime tree internals
- scheduler and commit internals
- DOM bridge details
- router implementation details
- fetch/cache implementation details
- service worker and asset plumbing
- worker and cross-tab transport details

### Interposer Shape

One clean option is:

```go
type InterposerRegistry interface {
	GetRuntimeInterposer() RuntimeInterposer
	GetDOMInterposer() DOMInterposer
	GetEventInterposer() EventInterposer
	GetNetworkInterposer() NetworkInterposer
	GetAssetInterposer() AssetInterposer
	GetSecurityInterposer() SecurityInterposer
}
```

Each interposer then implements the stable service contract on top of implementation-specific data sources.

Example:

```go
type RuntimeInterposer interface {
	GetRuntimeSnapshot() RuntimeSnapshot
	GetSelectedRuntimeNode(getPath string) (RuntimeNodeSnapshot, error)
	SubscribeRuntimeDiagnostics(getHandler func(Diagnostic)) Subscription
	SubscribeRuntimeProfiling(getHandler func(ProfilingEvent)) Subscription
}
```

The service contract exposed to plugins can be identical to the interposer interface or wrapped by a thin kernel-owned facade. The important point is that the backing implementation remains swappable.

### Runtime 1 And Runtime 2 Strategy

The first-pass kernel should assume that multiple implementation backends may exist.

Recommended approach:

- `internal/runtime` implements one set of interposers
- `internal/runtime2` implements another set of interposers
- runtime selection happens inside core bootstrap
- the kernel receives one resolved interposer registry and stays agnostic

This allows plugin APIs to remain stable while runtime implementation details continue to evolve.

### Interposer Rules

- interposers are internal-only
- interposers may depend on unstable runtime details
- interposers must return stable plugin-facing shapes
- interposers must not leak raw implementation pointers
- interposers must be replaceable without changing plugin code

## Devtools Use Cases That Must Shape The API

The plugin kernel should not be designed around the current compact devtools panel only. It should be designed around a broader set of devtools workflows.

The first-pass devtools plan should support these use cases at the planning and API layer even if some land in later phases.

### 1. Runtime Tree Inspection

Questions devtools should answer:

- what components rendered
- why did they rerender
- what hooks are attached
- what reactive sources or updates triggered this branch
- what subtree is hot

API consequences:

- runtime snapshot service
- profiling service
- render-cause metadata
- selected-node inspection API

### 2. Full DOM Inspection

Questions devtools should answer:

- what actual DOM tree exists now
- how does it differ from the framework component tree
- what attributes, classes, and text content were committed
- what overlays or portals exist outside the main tree

API consequences:

- separate DOM service, not just component-tree inspection
- DOM node identity and parent-child topology
- portal and overlay ownership metadata
- command API for inspect, highlight, and scroll-into-view

### 3. DOM Manipulation, Theming, And CSS Adaptation

Questions plugins should be able to answer:

- can a plugin detect the effective visual theme of the current DOM
- can a plugin scan computed styles, classes, attributes, and CSS variables
- can a plugin apply a dynamic dark mode or light mode treatment
- can a plugin restyle third-party or legacy DOM that does not follow framework theme tokens
- can a plugin rewrite DOM styling in a reversible and bounded way

Example plugin classes this should support:

- dynamic dark mode and light mode adapters
- high-contrast accessibility mode adapters
- reduced-motion visual adapters
- enterprise branding or white-label theming adapters
- DOM cleanup or normalization plugins for hostile third-party embeds

API consequences:

- DOM read access is not enough; the kernel needs a bounded DOM mutation model
- the framework should prefer style and attribute patch operations over arbitrary DOM ownership transfer
- plugins need reversible patch handles so changes can be removed cleanly
- plugins need access to computed style summaries, CSS variable state, and selector-scoped targeting
- the kernel must define what kinds of DOM mutation are allowed, rejected, or reserved to core

### 4. Event Interception And Relaying

Questions devtools should answer:

- what native and framework events fired
- in what order did capture, target, and bubble happen
- which handler prevented default or stopped propagation
- which component or DOM node handled the event
- can those events be relayed to another devtools UI or recorder

API consequences:

- event observation service
- bounded event ring buffer
- per-event metadata
- replay-safe serialization
- optional relay sink for remote or detached devtools consumers

### 5. Network Before-And-After Inspection

Questions devtools should answer:

- what request was about to be sent
- what cache key was used
- what request headers or payload summary existed before send
- what response status, timing, and body summary came back
- what retries, failures, or revalidations happened

API consequences:

- network lifecycle events before request dispatch, after response, after decode, after error
- request and response summaries with redaction rules
- explicit separation of metadata and sensitive body capture
- timing and correlation IDs

### 6. Router And Loader Inspection

Questions devtools should answer:

- what route is active
- what loaders are pending or failed
- what redirects happened
- what guard decisions ran
- what revalidation or retry actions are possible

API consequences:

- router inspection service
- loader timeline events
- command API for retry and revalidate
- route transition history

### 7. Hydration And SSR Inspection

Questions devtools should answer:

- what bootstrap payloads were present
- what hydration mismatches happened
- which nodes were discarded or recovered
- what boundary failed and why

API consequences:

- SSR and hydration services
- bootstrap summary and size reporting
- mismatch event stream
- hydration replay artifacts

### 8. Cache, State, And Data Flow Inspection

Questions devtools should answer:

- what async cache entries exist
- who owns them
- what resource is stale or failed
- how did state mutate over time

API consequences:

- fetch and state inspection services
- cache lifecycle events
- mutation timeline entries
- optional time-travel or replay markers

### 9. Caching, Asset Delivery, And Release Management

Questions plugins should be able to answer:

- what browser cache, cache storage, and framework cache layers exist
- what service-worker-managed assets are installed, stale, missing, or oversized
- what release manifest or asset plan is active
- what preload, prefetch, or cache warming decisions were made
- what asset invalidation or version drift exists across tabs, workers, or service workers
- what CDN, compression, or content-hash issues are degrading delivery

Example plugin classes this should support:

- asset budget and oversized-bundle analyzers
- stale asset or manifest drift detectors
- cache warming and prefetch strategy plugins
- service worker asset-audit plugins
- static release and rollback diagnostics plugins

API consequences:

- async-data cache inspection is not enough; the kernel needs an explicit asset and cache-storage service
- the framework should expose release-manifest, cache-storage, preload, and invalidation data as first-class snapshots and events
- asset plugins need typed commands for safe invalidation, refresh, or reconcile flows
- asset policy plugins should be able to annotate or block unsafe release transitions through typed decisions

### 10. Worker, Channel, And Multi-Client Inspection

Questions devtools should answer:

- what workers are running
- what messages crossed channels
- what cross-tab events were published
- what peer compatibility or authority state exists

API consequences:

- worker service
- channel and transport trace service
- multi-client event stream
- correlation IDs across worker and window boundaries

### 11. Storage And Persistence Inspection

Questions devtools should answer:

- what local or session storage keys changed
- what persistent snapshot keys were read or written
- what offline replay queue is pending

API consequences:

- storage service
- storage mutation observation
- offline queue inspection
- redaction policy for stored values

### 12. Overlay, Portal, And Accessibility Inspection

Questions devtools should answer:

- what overlays are mounted
- what focus trap or overlay stack exists
- what aria contracts are violated
- what live-region or announcer events fired

API consequences:

- overlay service
- accessibility inspection service
- focus-path and active-element service
- command API for highlight and focus jump

### 13. Error Recovery And Actionability

Questions devtools should answer:

- what failures are active
- what is the first remediation step
- what actions can safely be triggered from the UI

API consequences:

- diagnostics service
- typed overlay actions
- command validation and authorization
- plugin fault visibility

### 14. Security Inspection, Policy, And Guardrails

Questions plugins should be able to answer:

- is the app exposing sensitive data through diagnostics, logs, storage, DOM, or network summaries
- are there unsafe request destinations, headers, query params, or payload shapes
- are support exports and replay captures properly redacted
- are security-relevant browser conditions present, such as mixed content, untrusted embeds, or missing protective metadata
- should the framework block, warn, or annotate a risky operation before it proceeds

Example plugin classes this should support:

- sensitive-data leak detectors for logs, DOM, storage, and exports
- request policy plugins that enforce allowed destinations, headers, or query rules
- support-bundle redaction policy plugins
- CSP, iframe, or embed-audit plugins
- auth and session guard plugins that annotate risky route transitions or form submissions

API consequences:

- security needs both inspection and policy services
- some security plugins are advisory and some are enforcement-oriented
- policy evaluation must be explicit, typed, and auditable
- security plugins should be able to produce diagnostics, block decisions, and redaction decisions without direct arbitrary mutation

### 15. Recording, Replay, And Support Capture

Questions devtools should answer:

- can we capture a timeline around the failure
- can we replay state locally
- can we emit a support-safe artifact

API consequences:

- capture and replay service
- timeline session API
- redacted export paths
- event, network, and DOM summaries suitable for replay

## Typed Service API

Deep plugins need to interact with the framework, but through a strong API that constrains behavior.

The service pattern should separate:

- read services
- command services
- event services

Implementation note:

- the kernel should construct these services from interposers, not by reaching directly into runtime or platform internals

### Service Design Pattern

Every service family should follow the same shape:

- inspect methods return immutable snapshots or records
- observe methods return bounded subscriptions
- command methods run explicit, validated actions
- all three should carry enough metadata for budgets, freshness, and diagnostics

One clean base pattern is:

```go
type QueryOptions struct {
	Cursor string
	Budget QueryBudget
}

type CommandOptions struct {
	Reason   string
	AuditTag string
}
```

The concrete service families can extend those option shapes with subsystem-specific filters.

### Runtime Service

Suggested shape:

```go
type RuntimeInspectService interface {
	GetRuntimeSnapshot(getOptions RuntimeSnapshotOptions) (RuntimeSnapshot, SnapshotMeta, error)
	GetSelectedRuntimeNode(getNodeID string) (RuntimeNodeSnapshot, SnapshotMeta, error)
	SubscribeRuntimeDiagnostics(getOptions DiagnosticSubscriptionOptions, getHandler func(Diagnostic)) (Subscription, error)
	SubscribeRuntimeProfiling(getOptions ProfilingSubscriptionOptions, getHandler func(ProfilingEvent)) (Subscription, error)
}
```

### DOM Service

Suggested shape:

```go
type DOMService interface {
	GetDOMSnapshot(getOptions DOMSnapshotOptions) (DOMSnapshot, SnapshotMeta, error)
	GetDOMNode(getNodeID string) (DOMNodeSnapshot, SnapshotMeta, error)
	GetComputedStyle(getNodeID string, getOptions StyleQueryOptions) (StyleSnapshot, SnapshotMeta, error)
	SubscribeDOMMutations(getOptions DOMMutationSubscriptionOptions, getHandler func(DOMMutationRecord)) (Subscription, error)
	RunHighlightNode(getNodeID string, getOptions HighlightOptions) error
	RunScrollNodeIntoView(getNodeID string) error
	RunApplyPatch(getPatch DOMPatch, getOptions CommandOptions) (PatchHandle, error)
	RunRemovePatch(getHandleID string, getOptions CommandOptions) error
}
```

### Style And Theme Service

Suggested shape:

```go
type StyleService interface {
	GetThemeSnapshot() (ThemeSnapshot, SnapshotMeta, error)
	GetCSSVariables(getScope ScopeSelector) (map[string]string, SnapshotMeta, error)
	RunApplyThemePatch(getPatch ThemePatch, getOptions CommandOptions) (PatchHandle, error)
	RunApplyStylesheetPatch(getPatch StylesheetPatch, getOptions CommandOptions) (PatchHandle, error)
	RunRemovePatch(getHandleID string, getOptions CommandOptions) error
}
```

The intent is to support plugins such as dynamic dark-mode adapters without giving them unconstrained ownership of the DOM.

The preferred mutation shapes are:

- CSS variable overrides
- scoped class additions and removals
- attribute patches
- stylesheet patch insertion and removal
- bounded text or inline-style patches only where explicitly allowed

The preferred non-goals are:

- arbitrary raw JavaScript execution by plugins
- direct mutation of framework-owned reconciliation internals
- permanent takeover of element ownership from the renderer

### Event Service

Suggested shape:

```go
type EventService interface {
	GetRecentEvents(getOptions EventQueryOptions) ([]EventRecord, SnapshotMeta, error)
	SubscribeEvents(getOptions EventSubscriptionOptions, getHandler func(EventRecord)) (Subscription, error)
	RunEventRelay(getRecord EventRecord, getTarget RelayTarget, getOptions CommandOptions) error
}
```

The event record should eventually carry:

- event kind and type
- timestamp
- capture, target, and bubble phase markers
- DOM node and component ownership
- default-prevented and propagation-stopped flags
- summarized event payload

### Router Service

Suggested shape:

```go
type RouterService interface {
	GetRouteSnapshot() (RouteSnapshot, SnapshotMeta, error)
	GetRouteHistory(getOptions RouteHistoryOptions) ([]RouteTransition, SnapshotMeta, error)
	RunNavigate(getPath string, getOptions NavigateOptions) error
	RunRetryLoader(getKey string, getOptions CommandOptions) error
	RunRevalidateRoute(getOptions CommandOptions) error
	SubscribeRouteTransitions(getOptions RouteTransitionSubscriptionOptions, getHandler func(RouteTransition)) (Subscription, error)
}
```

### Network And Async-Data Service

Suggested shape:

```go
type NetworkService interface {
	GetRecentTransactions(getOptions NetworkQueryOptions) ([]NetworkTransaction, SnapshotMeta, error)
	SubscribeNetworkTransactions(getOptions NetworkSubscriptionOptions, getHandler func(NetworkTransactionEvent)) (Subscription, error)
	GetCacheSnapshot(getOptions CacheSnapshotOptions) ([]CacheEntry, SnapshotMeta, error)
	RunClearCacheKey(getKey string, getOptions CommandOptions) error
	RunRevalidateCacheKey(getKey string, getOptions CommandOptions) error
}
```

The network event model should support these phases:

- request prepared
- request about to send
- response headers received
- response body summarized
- decode completed
- request failed
- cache hit or miss

This is the planning hook that supports before-and-after API visibility.

### Asset And Cache Service

Suggested shape:

```go
type AssetService interface {
	GetAssetSnapshot() (AssetSnapshot, SnapshotMeta, error)
	GetReleaseManifest() (ReleaseManifestSnapshot, SnapshotMeta, error)
	GetCacheStorageSnapshot(getOptions CacheStorageQueryOptions) (CacheStorageSnapshot, SnapshotMeta, error)
	SubscribeAssetEvents(getOptions AssetSubscriptionOptions, getHandler func(AssetEvent)) (Subscription, error)
	RunInvalidateAsset(getAssetKey string, getOptions CommandOptions) error
	RunRefreshReleaseManifest(getOptions CommandOptions) error
	RunWarmAssets(getPlan AssetWarmPlan, getOptions CommandOptions) error
}
```

The asset event model should support at least:

- manifest loaded
- manifest mismatch detected
- asset cache hit or miss
- service worker asset install and activate
- stale asset detected
- preload or prefetch queued
- asset budget warning

This is the planning hook that supports service-worker, cache-storage, preload, and release-management plugins.

### SSR and Hydration Service

Suggested shape:

```go
type SSRInspectService interface {
	GetBootstrapSnapshot() (BootstrapSnapshot, SnapshotMeta, error)
	GetHydrationSnapshot() (HydrationSnapshot, SnapshotMeta, error)
	SubscribeHydrationEvents(getOptions HydrationSubscriptionOptions, getHandler func(HydrationEvent)) (Subscription, error)
}
```

### Forms Service

Suggested shape:

```go
type FormsService interface {
	GetFormSnapshot(getOptions FormSnapshotOptions) ([]FormSnapshot, SnapshotMeta, error)
	RunValidateForm(getFormID string, getOptions CommandOptions) ([]ValidationIssue, error)
	SubscribeFormEvents(getOptions FormEventSubscriptionOptions, getHandler func(FormEvent)) (Subscription, error)
}
```

### Worker And Coordination Service

Suggested shape:

```go
type WorkerService interface {
	GetWorkerSnapshot() (WorkerSnapshot, SnapshotMeta, error)
	SubscribeWorkerMessages(getOptions WorkerMessageSubscriptionOptions, getHandler func(WorkerMessageRecord)) (Subscription, error)
	SubscribeCoordinationEvents(getOptions CoordinationSubscriptionOptions, getHandler func(CoordinationEvent)) (Subscription, error)
}
```

### Storage Service

Suggested shape:

```go
type StorageService interface {
	GetStorageSnapshot(getOptions StorageSnapshotOptions) (StorageSnapshot, SnapshotMeta, error)
	SubscribeStorageEvents(getOptions StorageSubscriptionOptions, getHandler func(StorageEvent)) (Subscription, error)
}
```

### Overlay And Accessibility Service

Suggested shape:

```go
type SurfaceService interface {
	GetOverlaySnapshot() (OverlaySnapshot, SnapshotMeta, error)
	GetAccessibilitySnapshot() (AccessibilitySnapshot, SnapshotMeta, error)
	RunHighlightFocusPath(getOptions CommandOptions) error
	SubscribeSurfaceEvents(getOptions SurfaceSubscriptionOptions, getHandler func(SurfaceEvent)) (Subscription, error)
}
```

### Capture And Replay Service

Suggested shape:

```go
type CaptureService interface {
	StartCaptureSession(getOptions CaptureOptions) (CaptureSession, error)
	GetReplayCatalog(getOptions ReplayCatalogOptions) ([]ReplayArtifact, SnapshotMeta, error)
	RunReplayArtifact(getArtifactID string, getOptions CommandOptions) error
}
```

### Diagnostics Service

Suggested shape:

```go
type DiagnosticsService interface {
	PublishPluginDiagnostic(getReport DiagnosticReport)
	SubscribeFrameworkDiagnostics(getOptions DiagnosticSubscriptionOptions, getHandler func(DiagnosticReport)) (Subscription, error)
}
```

### Security Service

Suggested shape:

```go
type SecurityService interface {
	GetSecuritySnapshot() (SecuritySnapshot, SnapshotMeta, error)
	RunEvaluateRequestPolicy(getRequest SecurityRequestContext) (SecurityDecision, error)
	RunEvaluateExportPolicy(getArtifact SecurityArtifactContext) (SecurityDecision, error)
	RunEvaluateDOMPolicy(getPatch DOMPatch) (SecurityDecision, error)
	RunEvaluateStoragePolicy(getContext StorageWriteContext) (SecurityDecision, error)
	RunEvaluateRoutePolicy(getContext RouteSecurityContext) (SecurityDecision, error)
	SubscribeSecurityEvents(getOptions SecuritySubscriptionOptions, getHandler func(SecurityEvent)) (Subscription, error)
}
```

The intent is to support both advisory and enforcement-oriented plugins without giving them blanket control over framework behavior.

The security decision model should support outcomes such as:

- allow
- warn
- redact
- require-confirmation
- block

Security-oriented evaluation contexts should be explicit and typed for:

- outbound requests
- support bundle and replay exports
- DOM and theme patches
- storage writes
- route transitions where session or auth policy matters

### Strong Service Rules

- read methods return cloned or immutable snapshots
- command methods are explicit and named by intent
- services do not return raw mutable framework pointers
- services do not expose internal runtime registries directly
- subscriptions return owned cleanup handles
- high-volume event streams must support filtering, sampling, limits, or bounded buffers
- request and event payload capture must support redaction and size budgets
- before-and-after network events should expose summaries first, with richer body capture behind explicit opt-in
- DOM mutation commands should return patch handles so the kernel can revert, expire, or quarantine plugin-owned changes
- style and DOM patch operations should be bounded by policy rather than allowing arbitrary node replacement
- security policy evaluation should happen against typed contexts, not arbitrary plugin-provided code injection points
- snapshot-returning methods should also expose metadata about truncation, freshness, and backend source
- subscription-returning methods should accept filters and budget hints rather than forcing the kernel to stream everything
- command methods should accept audit context so user-initiated and plugin-initiated mutations can be distinguished later

## Contribution API

The contribution API should stay typed and subsystem-owned.

Generic marker:

```go
type Contribution interface {
	GetContributionMeta() ContributionMeta
}

type ContributionMeta struct {
	ContributionID string
	Kind           ContributionKind
	ExecutionClass ExecutionClass
	ActivationPolicy ActivationPolicy
	OrderKey       string
}

type ContributionRegistration struct {
	Contribution Contribution
	Scope        ContributionScope
}
```

Contribution metadata is what lets the kernel apply ordering, activation, budget policy, and quarantine behavior consistently across different plugin families.

### Devtools Contribution

`devtools` is the first serious use case and should drive the shape.

The initial contribution model should go beyond sections and actions.

Suggested interfaces:

```go
type DevtoolsViewContribution interface {
	Contribution
	GetDevtoolsViews(getContext DevtoolsContext) ([]DevtoolsViewDescriptor, error)
}

type DevtoolsPanelContribution interface {
	Contribution
	GetDevtoolsPanels(getContext DevtoolsContext) ([]DevtoolsPanelDescriptor, error)
}

type DevtoolsSectionContribution interface {
	Contribution
	GetDevtoolsSections(getContext DevtoolsContext) ([]devtools.ExtensionSection, error)
}

type DevtoolsActionContribution interface {
	Contribution
	GetDevtoolsActions(getContext DevtoolsContext) ([]devtools.ErrorOverlayAction, error)
}

type DevtoolsTimelineContribution interface {
	Contribution
	GetTimelineLanes(getContext DevtoolsContext) ([]TimelineLaneDescriptor, error)
}

type DevtoolsInspectorContribution interface {
	Contribution
	GetInspectors(getContext DevtoolsContext) ([]InspectorDescriptor, error)
}
```

Key rule:

- these methods are live providers, not one-time setup callbacks

That avoids the current static bridge problem and keeps the devtools UI reactive to framework state.

### Router Contribution

Suggested interface:

```go
type RouterContribution interface {
	Contribution
	EvaluateRoute(getContext RouteEvaluationContext) (GuardDecision, error)
	ObserveNavigation(getContext NavigationObservationContext) error
}
```

### Forms Contribution

Suggested interface:

```go
type FormsContribution interface {
	Contribution
	ValidateSubmission(getContext FormValidationContext) ([]ValidationIssue, error)
	ObserveSubmission(getContext FormObservationContext) error
}
```

### Head Contribution

Suggested interface:

```go
type HeadContribution interface {
	Contribution
	GetHeadNodes(getContext HeadContext) ([]ui.Node, error)
}
```

### Theme Contribution

Suggested interface:

```go
type ThemeContribution interface {
	Contribution
	GetThemePatches(getContext ThemeContext) ([]ThemePatch, error)
}
```

This is the contribution family for plugins such as dynamic dark mode, high-contrast mode, branding overlays, or scoped CSS adaptation.

### DOM Patch Contribution

Suggested interface:

```go
type DOMPatchContribution interface {
	Contribution
	GetDOMPatches(getContext DOMPatchContext) ([]DOMPatch, error)
}
```

This is intentionally stronger than pure inspection and intentionally weaker than arbitrary DOM takeover.

The expected use cases are:

- applying reversible style patches
- patching classes or attributes on legacy nodes
- adapting third-party embed containers
- adding instrumentation markers for devtools or testing

### Security Inspection Contribution

Suggested interface:

```go
type SecurityInspectionContribution interface {
	Contribution
	GetSecurityFindings(getContext SecurityInspectionContext) ([]SecurityFinding, error)
}
```

This is the contribution family for plugins that detect or annotate security risks without directly enforcing policy.

### Security Policy Contribution

Suggested interface:

```go
type SecurityPolicyContribution interface {
	Contribution
	EvaluateRequest(getContext SecurityRequestContext) (SecurityDecision, error)
	EvaluateExport(getContext SecurityArtifactContext) (SecurityDecision, error)
	EvaluateDOMPatch(getContext DOMPatchSecurityContext) (SecurityDecision, error)
	EvaluateStorageWrite(getContext StorageWriteContext) (SecurityDecision, error)
}
```

This is the contribution family for plugins that can warn, redact, require confirmation, or block specific operations through typed policy hooks.

### Cache Policy Contribution

Suggested interface:

```go
type CachePolicyContribution interface {
	Contribution
	EvaluateCacheOperation(getContext CacheOperationContext) (CacheDecision, error)
}
```

This is the contribution family for plugins that can annotate, tune, or block risky cache operations through typed policy evaluation.

### Asset Management Contribution

Suggested interface:

```go
type AssetManagementContribution interface {
	Contribution
	GetAssetFindings(getContext AssetContext) ([]AssetFinding, error)
	EvaluateReleaseTransition(getContext ReleaseTransitionContext) (AssetDecision, error)
}
```

This is the contribution family for plugins that inspect asset health, release manifests, service worker state, preload strategies, or stale-asset drift and can surface diagnostics or policy decisions.

### Context Objects For Contributions

Contribution methods should receive typed context objects rather than raw subsystem snapshots only.

That keeps the contract extensible because each family can grow context metadata, feature flags, or service handles without changing the top-level lifecycle.

Suggested examples:

```go
type DevtoolsContext interface {
	GetSnapshot() DevtoolsSnapshot
	GetViewID() string
	GetSelectedNodeID() string
	GetQueryBudget() QueryBudget
	GetServices() ServiceResolver
}

type ThemeContext interface {
	GetSnapshot() ThemeSnapshot
	GetScope() ScopeSelector
	GetQueryBudget() QueryBudget
	GetServices() ServiceResolver
}

type RouteEvaluationContext interface {
	GetRouteRequest() RouteRequest
	GetRouteSnapshot() RouteSnapshot
	GetReason() string
	GetServices() ServiceResolver
}
```

This pattern should apply across contribution families.

### Recommended Plugin Usage Pattern

The API should encourage one consistent implementation pattern:

- plugin startup resolves required and optional services
- plugin startup registers immutable contribution values
- contribution methods stay live and compute from typed contexts
- background subscriptions are explicit and registered for cleanup
- commands flow back through services rather than direct internal mutation

One plausible first-pass devtools plugin looks like:

```go
type NoopPluginHandle struct{}

func (getHandle NoopPluginHandle) StopPlugin(getContext context.Context) error {
	return nil
}

type DevtoolsPlugin struct{}

func (getPlugin DevtoolsPlugin) Manifest() Manifest {
	return Manifest{
		ID:                "gwc.devtools",
		Version:           "0.1.0",
		APIVersion:        "v1alpha1",
		Trust:             TrustLevelFirstParty,
		RequiredServices:  []ServiceKey{"runtime.inspect", "router", "network", "diagnostics"},
		OptionalServices:  []ServiceKey{"dom", "event", "asset", "security"},
		ContributionKinds: []ContributionKind{"devtools.panel", "devtools.section", "devtools.timeline"},
		ActivationPolicy:  ActivationPolicy{Kind: ActivationKindBoot},
	}
}

func (getPlugin DevtoolsPlugin) StartPlugin(getContext PluginContext) (PluginHandle, error) {
	getServices := getContext.GetServiceResolver()
	getRuntime := getServices.GetRuntimeInspectService()
	getRouter, _ := getServices.GetRouterService()
	getNetwork, _ := getServices.GetNetworkService()

	if getErr := getContext.RegisterContribution(ContributionRegistration{
		Contribution: DevtoolsOverviewContribution{
			Runtime: getRuntime,
			Router:  getRouter,
			Network: getNetwork,
		},
	}); getErr != nil {
		return nil, getErr
	}

	return NoopPluginHandle{}, nil
}
```

The important point is that the plugin registers a stable contribution object once and the contribution computes live values later from service state and contribution context.

That keeps the lifecycle simple, keeps service lookup centralized, and avoids reintroducing one-shot callback bridges that freeze dynamic framework state.

## Panic Isolation And Fault Containment

This is a hard requirement.

Every plugin entrypoint must run behind a guarded execution wrapper.

Suggested kernel rule:

- `StartPlugin`
- `StopPlugin`
- every contribution read
- every contribution action
- every observer callback
- every subscription callback delivery

must be wrapped with `recover`.

Suggested helper shape:

```go
func runPluginCall(parsePluginID string, parseOp string, parseCall func() error) (parseErr error)
```

Kernel behavior on panic:

- recover the panic
- emit a framework diagnostic with plugin ID and operation name
- mark the plugin unhealthy
- quarantine the plugin contribution for the rest of the session or until explicit reset
- keep the framework running

Kernel behavior on repeated ordinary errors:

- maintain an error counter per plugin
- degrade or quarantine after a threshold if the plugin keeps failing hot paths

Hot-path protection rules:

- no plugin panic may unwind into router, render, hydration, or fetch core flows
- plugin failures downgrade plugin behavior, not framework behavior

## Plugin Health Model

The kernel should track:

- `starting`
- `healthy`
- `degraded`
- `quarantined`
- `stopped`

Suggested reasons:

- startup error
- panic during contribution read
- panic during command execution
- panic during subscription callback
- repeated transient errors
- incompatible API version
- exceeded event or network stream budget

Devtools should surface plugin health so failures are visible during development.

## Devtools First-Pass Plan

`devtools` should be the first plugin built on the kernel, but the framework must treat it as one plugin type among others.

### Devtools Plugin Responsibilities

- resolve runtime snapshots through typed services
- provide live sections for runtime, route, cache, hydration, profiling, and diagnostics
- provide full DOM inspection separate from runtime-tree inspection
- expose event-stream inspection and relay
- expose request and response timelines with before-and-after lifecycle visibility
- surface worker, coordination, and storage events
- provide live overlay actions through explicit command services
- surface plugin health and kernel diagnostics
- expose a plugin-inspection section so kernel status is debuggable
- support recording and replay-oriented artifact capture

### What Changes In Devtools

Replace the current static extension seam with live contribution resolution.

Current rough behavior:

- app or companion code sets extension sections and overlay actions globally
- `ApplyHostExtensions` snapshots host contributions once
- `SnapshotNow()` focuses on runtime tree, route, cache, profiling, diagnostics, and a few companion-owned areas

Proposed behavior:

- `devtools.Panel(...)` asks the kernel for active devtools contributions on each render or poll
- `devtools.ErrorOverlay(...)` asks the kernel for active actions against the current issue set
- a richer devtools shell can request specialized views such as DOM, network, events, or workers on demand
- contributions are merged, not replaced
- plugin health and faults appear as first-class diagnostics
- event, network, and DOM data can be streamed or snapshotted depending on the view

### Devtools Snapshot Shape

The devtools snapshot should grow a plugin/kernel section.

Suggested additions:

- plugin health list
- contribution counts by kind
- quarantined plugin diagnostics
- kernel API version
- DOM summary and selected-node metadata
- recent event summary
- recent network transaction summary
- worker and storage summary
- overlay and accessibility summary

Not every view needs every field in one monolithic snapshot. The important planning rule is that devtools must support both:

- compact summary snapshots
- richer on-demand inspectors and timeline queries

### Devtools API Families

The devtools package will likely need clearer families instead of one compact panel-only surface.

Possible long-term families:

- summary and snapshot APIs
- inspector APIs
- timeline APIs
- capture and replay APIs
- command APIs
- plugin health APIs

That does not require public stabilization yet, but the internal design should leave room for it.

## Full Devtools Data Model To Plan For

The devtools plugin framework should be able to express at least these data categories:

- component tree
- full DOM tree
- computed style and theme state
- selected node mapping between runtime and DOM
- hook state and rerender causes
- diagnostics and structured logs
- route transitions and loader history
- request and response lifecycle summaries
- cache and state mutation summaries
- cache storage, release manifests, and asset budget findings
- worker and channel message traces
- local and session storage mutations
- overlay stack and active focus path
- accessibility findings
- security findings and policy decisions
- capture sessions and replay artifacts
- plugin health and plugin fault history

## Canonical Built-In Types

The service and contribution APIs need a minimum shared vocabulary so implementations do not invent incompatible meanings.

### Shared Decision Types

```go
type DecisionKind string

const (
	DecisionKindAllow              DecisionKind = "allow"
	DecisionKindWarn               DecisionKind = "warn"
	DecisionKindRedact             DecisionKind = "redact"
	DecisionKindRequireConfirmation DecisionKind = "require_confirmation"
	DecisionKindBlock              DecisionKind = "block"
)

type PolicyDecision struct {
	Kind                   DecisionKind
	Reason                 string
	DiagnosticCode         string
	ShouldAudit            bool
	RequiresConfirmation   bool
	RedactionPlan          RedactionPlan
}
```

The first-pass policy families should use this shape or a thin wrapper over it:

- `SecurityDecision`
- `CacheDecision`
- `AssetDecision`
- `GuardDecision`

### Patch And Mutation Types

```go
type PatchHandle struct {
	ID         string
	PluginID   string
	Kind       string
	AppliedAt  time.Time
	ExpiresAt  time.Time
}

type DOMPatch struct {
	ID          string
	Scope       ScopeSelector
	Operations  []DOMPatchOperation
	Reversible  bool
	Reason      string
}

type ThemePatch struct {
	ID          string
	Scope       ScopeSelector
	Variables   map[string]string
	Classes     []ClassPatch
	Stylesheets []StylesheetPatch
	Reversible  bool
	Reason      string
}
```

### Snapshot Summary Types

```go
type RuntimeSnapshot struct {
	Meta             SnapshotMeta
	Roots            []RuntimeNodeSnapshot
	SelectedNodeID   string
	HotPathSummaries []HotPathSummary
}

type DOMSnapshot struct {
	Meta           SnapshotMeta
	Roots          []DOMNodeSnapshot
	SelectedNodeID string
	OverlayNodeIDs []string
}

type AssetSnapshot struct {
	Meta              SnapshotMeta
	ActiveReleaseID   string
	AssetEntries      []AssetEntrySummary
	CacheSummaries    []AssetCacheSummary
	DriftFindings     []AssetDriftFinding
}
```

### Scope And Registration Types

```go
type ContributionScope struct {
	AppID        string
	RoutePattern string
	ViewID       string
}

type ScopeSelector struct {
	NodeIDs    []string
	Selectors  []string
	RouteIDs   []string
}
```

These are intentionally summary-oriented. Richer backend detail should be added through optional capabilities, not by inflating every baseline type.

## Built-In Service And Contribution Registry

The manifest model needs a canonical registry for built-in service keys and contribution kinds.

### Built-In Service Keys

| Service Key | Owner | v1alpha1 Status | Notes |
| --- | --- | --- | --- |
| `runtime.inspect` | runtime facade over interposers | required | always available for kernel plugins |
| `diagnostics` | kernel | required | health and plugin diagnostics |
| `router` | router interposer | required | route summary and loader commands |
| `network` | fetch interposer | required | request lifecycle and async-data cache |
| `dom` | platform or runtime interposer | optional | full DOM inspection and mutation stream |
| `style` | platform or ui interposer | optional | theme and stylesheet patching |
| `event` | platform interposer | optional | event stream and relay |
| `asset` | asset and service-worker interposer | optional | manifest, cache storage, warmup, invalidation |
| `ssr.inspect` | runtime or platform interposer | optional | bootstrap and hydration details |
| `forms` | forms owner | optional | validation and submit observation |
| `worker` | platform interposer | optional | workers, channels, and coordination |
| `storage` | platform interposer | optional | storage inspection and mutation observation |
| `surface` | ui or platform interposer | optional | overlays, focus, accessibility |
| `capture` | kernel plus service owners | optional | recording and replay |
| `security` | security facade | optional | advisory and enforcement policy checks |

### Built-In Contribution Kinds

| Contribution Kind | Owner | v1alpha1 Status | Notes |
| --- | --- | --- | --- |
| `devtools.view` | devtools | required | live devtools view descriptors |
| `devtools.panel` | devtools | required | top-level panel descriptors |
| `devtools.section` | devtools | required | compact summary sections |
| `devtools.action` | devtools | required | overlay or action surfaces |
| `devtools.timeline` | devtools | optional | timeline lanes and timeline summaries |
| `devtools.inspector` | devtools | optional | selected-node or selected-resource inspectors |
| `router.guard` | router | optional | typed route evaluation |
| `router.observe` | router | optional | navigation observation |
| `forms.validate` | forms | optional | form validation contributions |
| `head.nodes` | head owner | optional | head node composition |
| `theme.patch` | ui or platform | optional | reversible theme contributions |
| `dom.patch` | ui or platform | optional | bounded DOM patch contributions |
| `security.inspect` | security owner | optional | findings and annotations |
| `security.policy` | security owner | optional | typed allow, warn, redact, block decisions |
| `cache.policy` | fetch owner | optional | cache operation policy |
| `asset.manage` | asset owner | optional | asset findings and release-transition policy |

## DOM Mutation Policy

If the plugin kernel is going to support DOM-manipulation plugins, it needs an explicit mutation policy.

The kernel should classify DOM operations into three tiers.

### Tier 1: Safe Style Adaptation

Allowed by default for trusted plugins:

- CSS variable overrides
- scoped stylesheet injection
- class additions and removals
- reversible attribute patching on approved attributes
- highlight and inspection overlays

This tier is what should power dynamic dark mode, light mode, high-contrast, and branding adapters.

### Tier 2: Bounded Structural Assistance

Allowed only where the service API explicitly approves it:

- adding framework-external inspector markers
- inserting temporary wrapper elements for overlays or inspection affordances
- patching third-party embed containers that core does not own

This tier should require stronger ownership checks.

### Tier 3: Restricted Structural Mutation

Not allowed as a general plugin capability:

- replacing framework-owned subtrees
- mutating reconciliation-critical DOM structure behind core's back
- direct takeover of event ownership
- arbitrary script execution

If a plugin needs Tier 3 power, that should be a deliberate framework feature, not a default plugin affordance.

### Dynamic Theme Plugin Guidance

A dynamic dark mode or light mode plugin should prefer this order:

1. inspect existing theme state and CSS variables
2. apply token and variable overrides first
3. add scoped stylesheet patches second
4. use class and attribute adaptation third
5. avoid direct inline-style rewriting unless no safer path exists

This keeps theming plugins compatible with framework ownership, SSR, and future design-token work.

This does not mean the first release must expose every one of these in the default UI. It means the planning and API layer must not block them.

## Command Semantics Matrix

The command surface should be normative, not hand-wavy, for the first pass.

| Command Family | Atomicity | Idempotency | Rollback Or Cleanup | Audit Requirement | Default Failure Behavior |
| --- | --- | --- | --- | --- | --- |
| `RunRetryLoader` | best-effort | not strictly idempotent across time; repeated retries are allowed | none; loader state remains framework-owned | required | return error and preserve current route state |
| `RunRevalidateRoute` | best-effort | effectively idempotent per route epoch | none; route state remains framework-owned | required | return error and record diagnostic |
| `RunNavigate` | atomic from plugin point of view once accepted | not idempotent | router owns rollback through normal navigation failure semantics | required | reject before dispatch if policy blocks |
| `RunClearCacheKey` | atomic per cache key | idempotent when key is already absent | none | required | return error for invalid scope, not partial clear |
| `RunRevalidateCacheKey` | best-effort | effectively idempotent per current entry version | none | required | leave current cache entry intact on failure |
| `RunInvalidateAsset` | atomic per asset key or manifest scope | idempotent when target is already invalid | none | required | reject invalid scope and keep caches unchanged |
| `RunRefreshReleaseManifest` | best-effort | idempotent for unchanged manifest source | none | required | keep previous manifest active and emit diagnostic |
| `RunWarmAssets` | best-effort across plan items | idempotent per resolved plan item | no rollback; warming is additive | required | partial completion is allowed but must be reported |
| `RunApplyThemePatch` | atomic per patch handle | idempotent by patch ID | explicit remove through patch handle | required | reject patch and leave prior styles intact |
| `RunApplyPatch` | atomic per accepted patch handle | idempotent by patch ID | explicit remove through patch handle | required | reject patch and leave DOM untouched |
| `RunRemovePatch` | atomic per patch handle | idempotent when handle is already absent | final cleanup action | required | succeed if already removed or expired |
| `RunReplayArtifact` | best-effort session start | not idempotent | replay session stop or reset | required | fail without mutating live runtime state |

Normative rules:

- commands may not partially mutate multiple unrelated scopes and then return success
- commands that support partial completion must report partial completion explicitly
- every accepted command should carry plugin ID, reason, and audit tag into diagnostics
- a command family without clear rollback or cleanup semantics should stay internal or be deferred

## Separation From The Existing `plugin` Package

This plan should not silently redefine the current companion API.

Recommended position:

- keep `plugin` as an app-owned companion host
- add the new kernel as a separate framework-owned system

Possible long-term outcomes:

- keep both systems permanently because they solve different problems
- add adapter bridges later if that becomes genuinely useful

The important point is that the deep plugin kernel should not masquerade as a small evolution of the current companion host.

## Proposed Package Layout

First-pass internal layout:

- `internal/pluginruntime/doc.go`
- `internal/pluginruntime/kernel.go`
- `internal/pluginruntime/manifest.go`
- `internal/pluginruntime/context.go`
- `internal/pluginruntime/health.go`
- `internal/pluginruntime/recovery.go`
- `internal/pluginruntime/services.go`
- `internal/pluginruntime/contributions.go`
- `internal/pluginruntime/streams.go`
- `internal/pluginruntime/capture.go`
- `internal/pluginruntime/interposer.go`
- `internal/pluginruntime/interposers/registry.go`

First-pass devtools integration points:

- `devtools/plugin_bridge.go`
- `devtools/plugin_views.go`
- `devtools/plugin_sections.go`
- `devtools/plugin_actions.go`
- `devtools/plugin_timeline.go`
- `devtools/plugin_inspectors.go`

Potential service-owner packages later:

- `internal/runtime/plugininterposer.go`
- `internal/runtime2/plugininterposer.go`
- `internal/platform/plugininterposer.go`
- `router/plugininterposer.go`
- `fetch/plugininterposer.go`
- `ui/plugininterposer.go`

The service-owner and interposer split can stay internal until the SPI matures.

## Subsystem Ownership Matrix

The document should explicitly assign ownership so service drift does not become a kernel problem.

| Surface | Stable Contract Owner | Interposer Owner | Primary Test Owner |
| --- | --- | --- | --- |
| kernel lifecycle and health | `internal/pluginruntime` | `internal/pluginruntime` | `internal/pluginruntime` |
| runtime inspect | runtime facade | `internal/runtime` and `internal/runtime2` | runtime owners plus kernel integration tests |
| router | router owner | router owner | router owner |
| network and async-data | fetch owner | fetch owner | fetch owner |
| dom | platform or ui owner | platform or runtime bridge owner | platform or ui owner |
| style and theme | ui or platform owner | ui or platform owner | ui or platform owner |
| event | platform owner | platform owner | platform owner |
| asset and cache storage | asset or service-worker owner | asset owner | asset owner |
| forms | forms owner | forms owner | forms owner |
| storage | platform owner | platform owner | platform owner |
| surface and accessibility | ui owner | ui or platform owner | ui owner |
| security | security owner | security facade and policy owner | security owner |
| capture and replay | kernel plus service owners | mixed | kernel integration tests plus service owners |
| devtools contribution consumption | devtools owner | devtools plus kernel bridge | devtools owner |

## Lifecycle Plan

### Boot

- core constructs the plugin kernel during runtime bootstrap
- core resolves the active interposer registry from the selected runtime and platform stack
- core registers built-in framework services
- core registers first-party plugins
- kernel starts plugins in deterministic order
- kernel records plugin health

### Runtime

- subsystems query active contributions when needed
- plugin calls run behind guarded wrappers
- plugin service calls are routed through interposers into the active implementation
- plugin diagnostics are surfaced through devtools and logging
- stream-backed services feed bounded buffers, subscriptions, or capture sessions

### Shutdown

- kernel stops plugins in reverse start order
- plugin stop failures are recorded and joined, but do not block core shutdown
- active capture and stream sessions are closed

## Normative Sequence Flows

The document should define a few end-to-end flows so the API and lifecycle are not interpreted differently by each subsystem.

### 1. Boot And Activation

1. core selects the active runtime and builds the interposer registry
2. core passes built-in plugin registrations and enablement config into the kernel
3. kernel validates manifest IDs, API version, trust level, and service availability
4. kernel starts boot-activated plugins in resolved order
5. kernel records health and publishes startup diagnostics

### 2. Devtools Section Resolution

1. devtools requests active `devtools.section` contributions from the kernel
2. kernel filters by activation policy and health
3. kernel builds a typed `DevtoolsContext`
4. kernel invokes contribution methods behind guarded wrappers
5. kernel merges ordered results and omits quarantined failures
6. devtools renders the merged result and any plugin health diagnostics

### 3. Request Policy Before Network Send

1. framework prepares the outbound request summary
2. kernel constructs a `SecurityRequestContext`
3. active security policy contributions are evaluated in resolved order
4. kernel combines decisions using policy precedence rules
5. framework either sends, warns, redacts, requires confirmation, or blocks
6. diagnostics and audit records capture the decision path

### 4. Theme Patch Apply And Remove

1. a theme plugin resolves style state through `StyleService`
2. the plugin emits a reversible `ThemePatch`
3. kernel validates scope, ownership, and security policy
4. interposer applies the patch atomically and returns a `PatchHandle`
5. the patch remains attributable to one plugin ID
6. plugin shutdown, quarantine, or explicit remove calls `RunRemovePatch`

### 5. Plugin Panic And Quarantine

1. kernel invokes one contribution method
2. the method panics
3. guarded execution recovers the panic and emits a diagnostic
4. kernel marks the plugin `degraded` or `quarantined`
5. the active subsystem continues without that contribution result
6. later calls skip the quarantined contribution until reset or restart

## Ordering Rules

The kernel must define deterministic ordering rules because contributions such as guards and observers can be order-sensitive.

Suggested ordering:

- sort by explicit `StartPriority`
- tie-break by plugin ID

Subsystem rule examples:

- router guards evaluate in resolved plugin order
- devtools sections append in resolved plugin order
- devtools timeline lanes append in resolved plugin order
- head contributions merge in resolved plugin order with subsystem-specific precedence rules

## Performance Contract

The plugin framework should define a performance contract as explicitly as it defines trust and safety.

The architecture should optimize for:

- low startup overhead
- bounded hot-path latency
- bounded memory growth
- incremental updates instead of full re-snapshot by default
- explicit backpressure instead of unbounded queues
- graceful degradation under load

Core performance rule:

- no plugin capability should be admitted without a defensible cost model

Every stable service family and contribution family should declare:

- its default execution class
- its latency and memory budget model
- its freshness model
- its retention model
- its degradation behavior under overload
- its normalization cost model across supported backends

## Execution Classes

The kernel should classify plugin work into execution classes.

### Hot Path

Examples:

- route guard evaluation
- request-policy decisions before dispatch
- DOM patch-policy evaluation before apply
- export-policy evaluation before release of an artifact

Rules:

- must be synchronous
- must have tight latency budgets
- should avoid allocations where possible
- should avoid full snapshots
- should be quarantined quickly on repeated slowdown or failure

### Warm Path

Examples:

- panel summary refresh
- route history summary
- cache and asset findings
- selected-node inspector refresh

Rules:

- may do moderate work
- should prefer cached or incremental data
- should avoid recomputing whole-world views on every poll

### Background Path

Examples:

- full DOM scans
- timeline recording
- replay artifact generation
- asset-budget analysis
- security leak scanning across stored artifacts

Rules:

- should run off hot paths
- should use bounded worker capacity
- should support cancellation
- should degrade cleanly under pressure

## Scheduling, Budgets, And Backpressure

The kernel should own an explicit scheduling and backpressure model.

### Scheduling Rules

- inline execution only for hot-path hooks that must return immediately
- bounded worker pools for background analysis
- optional browser worker offload where the interposer can support it
- no unbounded goroutine fan-out per plugin

### Budget Types

The kernel should track budgets for:

- startup latency
- per-hook latency
- event throughput
- queue depth
- memory retention
- snapshot size
- artifact export size

### Overload Behavior

When a plugin or service exceeds budget, the kernel should prefer:

- sample
- coalesce
- drop oldest
- degrade detail
- disable one contribution
- quarantine the plugin

It should not prefer:

- unbounded buffering
- hot-path blocking
- global framework slowdown just to preserve plugin fidelity

## Initial Operational Defaults

The first implementation should ship with explicit default limits instead of leaving every service owner to guess.

Recommended v1alpha1 defaults:

| Budget Or Limit | Initial Default |
| --- | --- |
| plugin startup soft budget | 25 ms per plugin |
| plugin startup hard failure threshold | 100 ms per plugin before degrade or quarantine review |
| hot-path hook soft budget | 2 ms |
| hot-path hook hard threshold | 5 ms |
| warm-path query soft budget | 25 ms |
| warm-path query hard threshold | 100 ms |
| background worker concurrency | 2 workers per plugin, 4 global by default |
| default event ring size | 1000 records or 2 MiB |
| default network ring size | 500 transactions or 4 MiB |
| default diagnostics ring size | 500 records |
| default DOM summary snapshot cap | 10,000 nodes or 4 MiB |
| default asset history retention | 200 events or 2 MiB |
| default replay artifact export cap | 25 MiB before explicit override |

These are not permanent guarantees. They are starting defaults that benchmarks and real plugin behavior should tune over time.

Default redaction and truncation rules:

- request and response bodies are summarized by default
- event payloads are summarized by default
- DOM text capture should truncate large text nodes by default
- storage values should expose key, kind, and size before value detail
- every truncated snapshot should mark `SnapshotMeta.Truncated`

## Pull Versus Push Data Model

The framework should be explicit about when to use snapshots versus subscriptions.

Recommended default:

- poll cheap summary snapshots
- fetch expensive detail on demand
- subscribe only to bounded streams with clear retention rules

This is especially important for:

- DOM inspection
- event streams
- network timelines
- worker and multi-client traffic
- asset and storage mutation feeds

## Incremental Data And Caching Model

Full rebuilds should be the exception, not the default.

The kernel and interposers should prefer:

- stable object IDs
- dirty-region updates
- diff or patch payloads
- selected-node detail queries
- windowed timeline queries
- cached derived plugin outputs with scoped invalidation

Practical examples:

- DOM interposers should expose incremental node changes where possible
- route and asset history should be queryable by window or cursor
- plugin findings derived from large snapshots should be cached and invalidated by affected scope

Cache invalidation should be explicit and scoped:

- invalidate by object ID, subtree, route, request key, asset manifest, or capture session
- avoid full-plugin cache resets when one affected scope can be identified
- let interposers surface coarse invalidation when a backend cannot support finer granularity cheaply

## Command Transaction Model

Any command that mutates framework-visible state should have explicit semantics.

Examples:

- loader retry
- cache invalidation
- release-manifest refresh
- theme patch application
- DOM patch application

Each command family should define:

- atomic versus best-effort behavior
- idempotency expectations
- rollback or cleanup path
- ownership checks
- conflict handling
- audit and diagnostics behavior

Commands that do not have a defensible transaction model should not be exposed.

## Memory And Retention Policy

The framework should define retention rules for high-volume plugin data.

Retention-controlled data includes:

- event rings
- network transaction buffers
- DOM snapshots
- worker message traces
- security findings
- replay artifacts
- asset and release history

Recommended rules:

- bounded ring buffers for recent activity
- max artifact size limits
- eviction by age and size
- explicit export for large or long-lived artifacts
- redaction before retention where feasible, not only before export

## Lazy Activation And Capability Negotiation

Not every plugin should start eagerly or request every service at boot.

The kernel should support:

- lazy plugin activation by feature or view
- optional capability negotiation
- feature probing through interposers
- graceful fallback when one backend cannot provide one service efficiently

Examples:

- full DOM inspection may activate only when the inspector view opens
- replay generation may activate only during a capture session
- runtime2 may support one richer stream that runtime1 cannot, while still satisfying the same stable summary contract

Activation policy should distinguish:

- boot-required capabilities that must exist before app runtime proceeds
- view-activated capabilities that start when a tool or panel opens
- session-activated capabilities that start only for a named recording or capture window
- opportunistic capabilities that should disappear cleanly when a backend cannot provide them within budget

## Cross-Runtime Normalization Strategy

The interposer layer should normalize only what must be stable for plugin consumers and should not force maximum-detail normalization on every backend in every path.

Normalization rules:

- normalize summaries and command semantics at the stable service boundary
- allow richer backend-specific capture detail behind optional capabilities
- avoid forcing `runtime` and `runtime2` to construct identical heavy snapshots on hot paths
- prefer backend-native efficient representations internally as long as the stable plugin contract remains coherent
- make normalization cost visible in benchmarks and compatibility review

Practical examples:

- both runtimes should expose the same route summary contract even if one backend can expose a richer transition timeline
- both runtimes should expose the same minimum DOM inspector summary contract even if only one backend can stream fine-grained mutation diffs efficiently
- a service should expose optional richer detail through feature negotiation instead of inflating the baseline contract for every backend

## Extensibility Admission Criteria

To keep the architecture scalable, new service and contribution families should meet explicit admission criteria.

A new family should require:

- at least one real consumer with a second plausible consumer
- a clear ownership boundary
- a typed cost model
- a safe interposer story
- testability and benchmarkability
- a defined downgrade story when a backend cannot provide it fully

This keeps the kernel from turning into a bag of one-off hooks.

## Security And Data Exposure Rules

The plugin framework must assume that deep plugins can see a lot.

Rules for the first pass:

- plugin access is only for trusted plugins
- browser-visible devtools remains a public data channel
- services must explicitly redact or omit data that should never be exposed
- command services must validate inputs and target ownership
- high-volume capture such as bodies, event payloads, and DOM snapshots must use opt-in scope and explicit size budgets
- DOM and theme patch services must enforce an ownership and mutation policy before applying changes
- security policy contributions must not become hidden, implicit policy engines; their decisions need visibility in diagnostics or devtools
- asset and cache plugins should work through typed invalidation and refresh commands rather than ad hoc cache clearing across unrelated scopes

Specific example:

- a devtools plugin may inspect cache metadata, route loaders, and diagnostics
- it should not automatically receive raw secrets, cookies, or server-only policy data
- network services should expose redacted headers and summarized body metadata by default
- event services should expose safe payload summaries by default rather than arbitrary deep object graphs
- theme and dark-mode plugins should apply reversible patch handles rather than uncontrolled inline rewrites across the entire page
- security plugins should be able to block unsafe request or export flows through typed decisions, but the framework should log those decisions with enough context to audit them
- asset plugins should be able to detect stale or mismatched release artifacts and request safe invalidation or refresh, but not silently purge unrelated caches without typed scope checks

## Compatibility Plan

This should begin as an internal SPI.

Recommended stages:

1. internal SPI for first-party use only
2. second plugin category proves generic shape
3. compatibility review and cleanup
4. public preview if the shape remains coherent
5. supported public API only after multiple real consumers exist

Compatibility rules even for the internal phase:

- version the kernel API
- fail plugin startup on incompatible `APIVersion`
- make incompatibility visible in devtools and logs
- avoid leaking raw internal structs so compatibility remains negotiable
- allow `runtime` and `runtime2` interposers to evolve independently behind the same stable plugin-facing contracts
- allow one backend to expose richer internals while still satisfying the same minimum stable plugin-facing contract

## Testing Plan

The plugin framework needs stronger tests than the current callback host because plugin isolation is a core correctness rule.

Required test families:

- manifest validation
- kernel startup and shutdown ordering
- interposer selection and wiring for different implementation backends
- service registration and lookup
- contribution registration and ordering
- execution-class enforcement and hot-path budget checks
- service-family declaration coverage for execution class, budget, freshness, retention, and degradation rules
- panic recovery during startup
- panic recovery during contribution reads
- panic recovery during contribution actions
- panic recovery during subscription callback delivery
- plugin quarantine after failure
- command API validation
- command transaction, rollback, and idempotency checks
- cloned read snapshots across services
- interposer snapshot normalization across runtime variants
- incremental snapshot and diff behavior
- plugin result cache invalidation behavior
- lazy activation and capability-negotiation behavior
- retention and bounded-buffer eviction behavior
- bounded event and network stream behavior
- cross-runtime optional-capability downgrade behavior
- redaction enforcement for sensitive payloads
- capture-session lifecycle and cleanup
- DOM patch lifecycle, rollback, and ownership enforcement
- theme patch application and cleanup
- security finding generation and policy decision auditing
- request-policy allow, warn, redact, confirmation, and block branches
- export redaction and security policy enforcement
- asset manifest, cache-storage, and service-worker state inspection
- asset invalidation, warming, and release-transition policy branches
- devtools live contribution resolution
- mixed plugin and non-plugin devtools composition

Required first-pass devtools scenarios:

- one healthy devtools plugin contributes live sections
- one healthy devtools plugin contributes a live DOM inspector
- one healthy devtools plugin observes before-and-after network lifecycle events
- one healthy devtools plugin observes and relays framework event records
- one healthy theme plugin applies reversible dark-mode patches through the style service
- one healthy security plugin flags sensitive values in logs, storage, or export previews
- one healthy security policy plugin blocks an unsafe request or export through a typed decision path
- one healthy asset plugin detects stale release-manifest or cache-storage drift and surfaces findings
- one healthy asset policy plugin triggers safe asset invalidation or warmup through typed commands
- one expensive inspector activates lazily and does not impact cold-start or hot-path budgets before activation
- one high-volume stream exceeds budget and degrades by sampling or coalescing instead of stalling the app
- one devtools plugin panics while resolving one section and is quarantined
- one devtools subscription callback panics and is quarantined without breaking the app
- overlay actions continue to work after a different plugin is quarantined
- app-owned devtools sections still compose with kernel-provided sections where intended

## v1alpha1 Minimum Shippable Scope

The first pass needs a hard boundary so implementation does not sprawl.

Required kernel features:

- manifest validation and explicit plugin registration
- enable or disable by plugin ID
- typed service resolver
- contribution registration and ordering
- panic recovery and quarantine
- health reporting and diagnostics
- execution classes, default budgets, and bounded subscriptions

Required service families:

- `runtime.inspect`
- `diagnostics`
- `router`
- `network`

Required devtools contribution kinds:

- `devtools.panel`
- `devtools.section`
- `devtools.action`

Required first-pass commands:

- `RunRetryLoader`
- `RunRevalidateRoute`
- `RunClearCacheKey`
- `RunRevalidateCacheKey`

Required first-pass devtools capabilities:

- runtime summary
- route and loader summary
- network before-and-after summary
- diagnostics and plugin health
- live contribution composition

Explicitly deferred unless already cheap behind one backend:

- full DOM mutation subscriptions
- event relay to remote sinks
- capture and replay artifact generation
- service-worker asset warmup policy
- hard-blocking security policy outside tightly scoped request and export checks
- public plugin packaging or runtime-installed plugins

## Rollout Plan

### Phase 0: Design And Approval

- review this plan
- resolve boundary questions
- choose internal package names
- choose initial trust and compatibility policy
- choose the minimum first-pass devtools use cases that the API layer must unblock

### Phase 1: Kernel Skeleton

- implement manifest, context, kernel lifecycle, health, and recovery wrappers
- implement interposer registry resolution and stable service facades
- define execution classes, budgets, and scheduler rules
- define the required service-family metadata for budgets, freshness, retention, and degradation
- add no-op service registration and contribution registration
- add focused unit tests

### Phase 2: Core Service Wiring

- expose runtime, DOM, router, fetch, diagnostics, hydration, profiling, worker, and storage interposers to the kernel
- expose style and theme patch services with ownership enforcement
- expose security inspection and policy services with typed evaluation contexts
- expose asset, cache-storage, and release-manifest services with typed invalidation and warmup commands
- keep these services internal
- validate snapshot cloning, bounded streams, command paths, and runtime1/runtime2 normalization
- validate minimum stable summaries plus optional richer capability negotiation where one backend can expose more detail efficiently

### Phase 3: Performance Foundations

- implement bounded buffers, retention rules, and lazy activation support
- implement incremental or diff-oriented snapshot paths where the interposers can support them
- add plugin result caching with scoped invalidation
- add normalization benchmarks so one backend does not inherit another backend's heavy path accidentally
- add focused performance benchmarks and overload tests

### Phase 4: Devtools As First Plugin

- implement first-party devtools plugin against the kernel
- replace static `ApplyHostExtensions` behavior for kernel-managed devtools contributions
- add at least one DOM inspector path
- add at least one event-trace path
- add at least one network before-and-after inspection path
- surface plugin health in devtools

### Phase 5: Capture And Replay Expansion

- add capture-session APIs
- connect bug bundle, support bundle, and replay flows to kernel-backed services where appropriate
- validate redaction and export rules

### Phase 6: Second Plugin Category

- pick one additional real plugin family
- likely candidates for the second proving consumer:
- router policy plugin
- head plugin
- forms validation plugin

The second real consumer is necessary before treating the kernel design as settled.

### Phase 7: Migration And Public API Review

- decide what stays internal
- decide whether the current `plugin` package remains companion-only
- decide whether any adapter path is worth adding

## Migration Guidance

Short-term:

- keep existing `plugin.Host` and existing devtools APIs working
- do not force apps to adopt the kernel

Medium-term:

- stop extending the current callback host for deep framework scenarios
- move first-party deep integrations onto the kernel

Long-term:

- decide whether to deprecate any overlapping companion-only seams once kernel-backed paths prove superior

## Benchmark And Regression Strategy

The plugin framework should ship with a benchmark and regression strategy from the start.

At minimum, measure:

- kernel startup overhead per enabled plugin
- service lookup cost
- route-guard latency under load
- snapshot construction cost by size
- incremental update cost versus full snapshot cost
- event-stream throughput and degradation behavior
- network timeline overhead
- DOM inspector query cost
- asset and security policy evaluation latency
- runtime1 versus runtime2 normalization cost

The benchmark strategy should include:

- microbenchmarks for hot-path operations
- bounded-load stress tests for streams and retention
- regression thresholds for startup and steady-state overhead
- comparative measurements across `runtime` and `runtime2`
- separate measurements for baseline summary paths versus optional richer capability paths
- alerting when a stable contract begins to require full heavy snapshot normalization on a hot or warm path

## Open Review Questions

1. Should the deep plugin kernel remain internal permanently, or is a public SPI a likely goal?
2. Should trusted plugin execution be limited to first-party code in the first release?
3. Should devtools be implemented as one plugin or as a plugin plus built-in core views?
4. Should route guards remain app-owned in the current companion host, or also gain a kernel-backed path?
5. Should kernel contributions compose with app-owned companion contributions by default in devtools, or stay separate?
6. Should plugin subscriptions be allowed in the first pass, or should all plugin reads be poll-based snapshots only?
7. How much of the full DOM, event, and network surface must land in the first devtools plugin versus later phases?
8. Should theme and DOM-patch plugins be first-class contribution families in the first kernel release?
9. Which security policy hooks should be advisory only versus hard-blocking in the first release?
10. Should cache and asset-management plugins be first-class contribution families in the first kernel release?
11. Should the stable plugin service layer map one-to-one onto interposer interfaces, or should the kernel own a separate facade over interposers?
12. Which capabilities are allowed to start eagerly in the first release, and which must stay lazy by policy?
13. What are the first concrete latency, retention, and overload budgets for hot, warm, and background execution classes?
14. How much optional richer backend detail should the stable contract expose in `runtime2` without pressuring `runtime` into expensive normalization?
15. What subsystem should be the second proving consumer after devtools?

## Recommended First Decisions

If we want the smallest coherent first implementation, the recommended answers are:

- keep the kernel internal for now
- allow first-party trusted plugins only
- implement devtools as one kernel plugin
- keep the current companion `plugin` package for app-owned extension
- compose devtools contributions rather than replacing state
- allow both poll-based reads and explicit subscriptions, but only through owned cleanup handles
- make full DOM inspection, event tracing, and network before-and-after lifecycle support first-class API goals now even if their UI lands incrementally
- support style and theme patch plugins through reversible patch handles rather than arbitrary DOM mutation
- support security inspection and typed policy decisions through explicit audit-friendly service hooks
- support cache and asset-management plugins through explicit manifest, cache-storage, invalidation, and warmup APIs
- keep expensive capabilities lazy by default unless a concrete boot-time need proves otherwise
- require every service family to declare execution class, budget, freshness, retention, and degradation behavior before it lands
- hide `runtime` and `runtime2` churn behind interposers so plugin-facing contracts stay stable
- normalize minimum stable summaries across runtimes and gate richer backend detail behind optional capability negotiation
- use a router policy or forms plugin as the second proving consumer

## Review Focus

The key review questions for this draft are:

- is the kernel versus companion split correct
- is the service-and-contribution split the right abstraction boundary
- is the panic isolation model strict enough
- does the service layer cover the real devtools use cases we care about
- does the security model give enough power for real policy plugins without hiding framework behavior behind opaque hooks
- does the asset and cache service layer cover service-worker, release-manifest, and invalidation workflows well enough
- is the interposer boundary in the right place to absorb `runtime` and `runtime2` churn without freezing unstable internals
- are the execution classes, budgets, retention rules, and overload behaviors concrete enough to keep this fast in practice
- does the normalization strategy preserve a stable contract without forcing runtime-wide heavy snapshot work
- is `devtools` the right first consumer
- which subsystem should be the second proof point
