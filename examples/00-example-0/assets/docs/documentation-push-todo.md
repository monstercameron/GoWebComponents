# Documentation Push TODO

## At A Glance

This document is the documentation-program backlog for the repo.

It exists to coordinate three related tasks:

- reviewing the full documentation surface
- reviewing the public API surface against that documentation
- closing gaps between package docs, guides, examples, and cross-links

Treat it as a working coverage and prioritization artifact, not as end-user documentation.

## Current Focus

The core question behind this backlog is not just "do docs exist?" It is:

- do the right docs exist for the supported public API surface?
- can an adopter find the relevant API from guides and examples without reading source?
- are package docs, policy docs, and workflow docs aligned enough to support a production user journey?

That means this file is partly an audit list and partly a release-readiness checklist for documentation quality.

## How To Use This Backlog

Work through the file in this order:

- use the checklist to define the documentation push outcome
- use the documentation-files inventory to review coverage across guides, READMEs, and example docs
- use the public API inventory to verify each supported package surface is documented and cross-linked

When items are completed, this document should reflect the real documentation state rather than staying a speculative wishlist.

## Current Status

- The project-level `docs/*.md` review pass and their example-0 mirror pages have been refreshed against the current runtime, tooling, and launcher workflow.
- The examples documentation pass has already refreshed `examples/README.md`, `examples/01-counter/README.md`, `examples/13-browser-compiler/README.md`, `examples/16-devtools/README.md`, `examples/17-ssr-routing/README.md`, `examples/18-ssr-server-routing/README.md`, `examples/19-nested-routes/README.md`, and `examples/20-portals/README.md`.
- This backlog remains open because the full package README audit, complete public API coverage mapping, and remaining example/package cross-links are not finished yet.

## Checklist

- [x] Refresh the project-level `docs/*.md` narratives and the example-0 mirror pages.
- [x] Refresh the current example README set already covered by the documentation push.
- [ ] Review every repository documentation file listed below.
- [ ] Review every public package API listed below.
- [ ] Mark which docs already cover each public API.
- [ ] Add or update missing API documentation.
- [ ] Add cross-links from package docs and guides to the relevant APIs.

## Documentation Files

- AGENTS.md
- CHANGELOG.md
- README.md
- devtools/README.md
- docs/ACCESSIBILITY.md
- docs/ACTIONABLE_ERRORS.md
- docs/ADOPTION.md
- docs/API_POLICY.md
- docs/ASSETS.md
- docs/ATLAS_COMMERCE_OS_TODO.md
- docs/BUILD_EXPERIMENTS.md
- docs/BROWSER_SUPPORT.md
- docs/CACHE.md
- docs/CODE_SPLITTING.md
- docs/COMPARISONS.md
- docs/COMPILER_ASSISTED_FEATURES.md
- docs/CONFIGURATION.md
- docs/CROSS_TAB.md
- docs/CUSTOM_ELEMENTS.md
- docs/DEPLOYMENT_TARGETS.md
- docs/ECOSYSTEM.md
- docs/ERROR_BOUNDARIES.md
- docs/FINE_GRAINED_REACTIVITY.md
- docs/FORMS.md
- docs/FRAMEWORK_SCOPE.md
- docs/HEAD_MANAGEMENT.md
- docs/HOT_RELOAD.md
- docs/HYDRATION.md
- docs/I18N.md
- docs/INTEROP.md
- docs/LOGGING.md
- docs/MIGRATIONS.md
- docs/MULTI_CLIENTS.md
- docs/MULTI_SURFACE.md
- docs/OBSERVABILITY.md
- docs/OFFLINE_MUTATIONS.md
- docs/ONBOARDING.md
- docs/OVERLAYS.md
- docs/PERFORMANCE.md
- docs/PRERENDER.md
- docs/PRODUCTION_CORRECTNESS.md
- docs/PROJECT_STRUCTURE.md
- docs/PWA.md
- docs/README.md
- docs/REFERENCE_MAP.md
- docs/ROUTER_AUTH.md
- docs/SCHEDULING.md
- docs/SECURITY.md
- docs/SERVER_INTEGRATION.md
- docs/SERVER_INTERACTIVE.md
- docs/START_HERE.md
- docs/STATE_TRANSFER.md
- docs/STREAMING_SSR.md
- docs/TESTING.md
- docs/TODO.md
- docs/TROUBLESHOOTING.md
- docs/WALKTHROUGHS.md
- docs/WASM_RELEASES.md
- docs/WORKERS.md
- docs/WORKFLOWS.md
- examples/01-counter/README.md
- examples/13-browser-compiler/README.md
- examples/16-devtools/README.md
- examples/17-ssr-routing/README.md
- examples/18-ssr-server-routing/README.md
- examples/19-nested-routes/README.md
- examples/20-portals/README.md
- examples/86-atlas-commerce-os/README.md
- examples/86-atlas-commerce-os/docs/ATLAS_COMMERCE_OS_TODO.md
- examples/86-atlas-commerce-os/docs/ATLAS_ROUTING_DATA_BASELINE.md
- examples/86-atlas-commerce-os/docs/LAYOUT_MAP.md
- examples/86-atlas-commerce-os/docs/README.md
- examples/86-atlas-commerce-os/server/README.md
- examples/87-ssr-secure-forms/README.md
- examples/MANUAL_TESTING.md
- examples/README.md
- fetch/README.md
- interop/README.md
- internal/README.md
- router/README.md
- state/README.md
- test/README.md
- tools/livereload/README.md
- tools/README.md
- utils/README.md

## Public API Inventory

Public package set, based on the current README and API policy:

- Stable: ui, html, state, fetch, router
- Supported companion: devtools, head
- Experimental: plugin, hotreload

### ui

Constants: AnnouncementAssertive, AnnouncementPolite, CurrentSSRBootstrapVersion, CurrentSSRStateUpdateVersion, DefaultBootstrapReferenceScriptID, DefaultBootstrapScriptID, DefaultCSRFFormFieldName, DefaultCSRFHeaderName, DefaultRouteBootstrapPayloadKey, OverlayKindCustom, OverlayKindDialog, OverlayKindMenu, OverlayKindPopover, OverlayKindSheet, OverlayKindTooltip, SSRBootstrapFormatCBOR, SSRBootstrapFormatJSON, SSRPayloadEncodingBinary, SSRPayloadEncodingCBOR, SSRPayloadEncodingJSON, SSRPayloadEncodingText, SSRPayloadEncodingTimeRFC3339, SSRPayloadEncodingTimeUnixNano, SSRPayloadKindCacheSeed, SSRPayloadKindData, SSRPayloadKindFormDefaults, SSRPayloadKindRouteData, SSRPayloadKindSessionHint, SSRPayloadReuseClientOwned, SSRPayloadReuseRevalidateAfterResume, SSRPayloadReuseTrustOnFirstResume, SSRPayloadScopeApp, SSRPayloadScopeRoute, SSRPayloadScopeSubtree

Variables: ErrorBoundary

Functions: AccessibleOverlay, AddStateUpdatePayload, AnalyzeSSRBootstrapSize, ApplySSRStateUpdate, AsyncBoundary, CreateContext, CreateElement, DefaultSSRBootstrapBudget, ExtractFiles, FileFromJSValue, Fragment, HotReloadBoundary, Hydrate, HydrateInto, InspectBootstrapPayloads, Lazy, MarshalSSRBootstrap, MarshalSSRBootstrapBinary, MarshalSSRBootstrapBinaryObserved, MarshalSSRBootstrapObserved, MarshalSSRStateUpdateBinary, MarshalSSRStateUpdateText, NewCSRFToken, ObserveSSR, Overlay, Portal, RawHandler, ReactiveRegion, ReadBootstrapPayload, ReadBootstrapReference, ReadBootstrapReferenceScript, ReadBootstrapScript, ReadCacheBootstrapSeed, ReadFormBootstrapDefaults, ReadRouteBootstrapData, ReadSessionBootstrapHint, RegisterBootstrapPayload, RegisterCacheBootstrapSeed, RegisterFormBootstrapDefaults, RegisterRouteBootstrapData, RegisterSessionBootstrapHint, Render, RenderBootstrapReferenceScript, RenderBootstrapScript, RenderBootstrapScriptObserved, RenderInto, RenderToString, RenderToStringObserved, StartTransition, Text, UnmarshalSSRBootstrap, UnmarshalSSRBootstrapBinary, UnmarshalSSRBootstrapReference, UnmarshalSSRStateUpdateBinary, UnmarshalSSRStateUpdateText, UnsupportedOnServer, UseAnnouncer, UseCallback, UseChannel, UseCompositeNavigation, UseContext, UseDebounced, UseDeferredValue, UseEffect, UseEvent, UseFocusManager, UseFocusTrap, UseForm, UseId, UseLazyNode, UseMemo, UseOverlayStack, UsePrevious, UseReducer, UseRef, UseState, UseTask, UseThrottled, UseTransition, UseWorkerTask

Types:

- AccessibleOverlayProps [struct]: AnchorSelector, AppRootSelector, Backdrop, BackdropClass, BackdropStyle, BackgroundInert, BaseZIndex, Child, Children, CloseOnEscape, CloseOnOutsideClick, DescribedBy, FallbackFocusSelector, InitialFocusSelector, Kind, LabelledBy, LockScroll, Modal, OnDismiss, Open, Positioning, RestoreFocus, Role, SurfaceClass, SurfaceID, SurfaceStyle, Target, TrapFocus
- AnnouncementMode [defined]
- Announcer [struct]: methods Announce, Assertive, AssertiveID, Clear, Polite, PoliteID, Region
- AsyncBoundaryProps [struct]: Content, Delay, Error, ErrorFallback, Fallback, Pending, Timeout, TimeoutFallback
- CSRFToken [struct]: FormFieldName, HeaderName, Value; methods FormField, Header
- ChangeEvent [defined]
- Channel [struct]: methods Closed, Get, Ok
- CompositeItem [struct]: Disabled, ID, Text
- CompositeNavigation [struct]: methods ActiveDescendant, ActiveID, ActiveIndex, IsActive, MoveEnd, MoveHome, MoveNext, MovePrevious, OnKeyDown, SetActive, TabIndex
- CompositeNavigationOptions [struct]: InitialIndex, Loop, Orientation
- Context [struct]: Provider
- ContextProvider [struct]
- ContextProviderProps [struct]: Child, Children, Value
- Debounced [struct]: methods Get, Pending
- Element [defined]
- ErrorBoundaryProps [struct]: Child, Children, ErrorFallback, Fallback, OnError, ResetKeys
- Event [defined]: methods GetKey, GetKeyCode, GetValue, IsChecked, PreventDefault, StopPropagation
- FieldErrors [defined]
- FieldStatus [struct]: Dirty, Error, Name, Pending, Touched
- File [struct]: methods JSValue, LastModified, Name, Size, Type
- FocusEvent [defined]
- FocusManager [struct]: methods FocusByID, FocusFirst, FocusFirstError, FocusSelector, RememberActive, Restore
- FocusOptions [struct]: PreventScroll
- FocusTrapOptions [struct]: Active, ContainerSelector, FallbackFocusSelector, InitialFocusSelector, RestoreFocus
- Form [struct]: methods ApplyServerErrors, Dirty, DirtyAny, Error, Errors, FieldMessage, FieldStatus, FormError, Get, HasErrors, HasFieldError, IntentPending, Reset, Set, SetErrors, SetField, SetFormError, SetSubmitIntent, Submit, SubmitError, SubmitIntent, SubmitWithIntent, Submitted, Submitting, Touch, Touched, TouchedAny, Update, Validate, ValidateAsync, ValidateIntent, Validated, Validating
- FormEvent [defined]
- Handler [struct]: methods Value
- HotReloadBoundaryProps [struct]: Child, Children, ResetKeys
- HydrationOptions [struct]: Bootstrap, BootstrapRef, Observability, ReferenceScriptID, ScriptID, Strict
- InputEvent [defined]
- KeyboardEvent [defined]
- LazyNode [struct]: methods Cancel, Get, Reload
- LazyNodeState [struct]: Error, Loading, Node, Ready
- LazyProps [struct]: Delay, Dependencies, ErrorFallback, Fallback, Loader, Timeout, TimeoutFallback
- MouseEvent [defined]
- Node [defined]
- OverlayKind [defined]
- OverlayProps [struct]: AnchorSelector, AppRootSelector, Backdrop, BackdropClass, BackdropStyle, BackgroundInert, BaseZIndex, Child, Children, CloseOnEscape, CloseOnOutsideClick, DescribedBy, FallbackFocusSelector, InitialFocusSelector, Kind, LabelledBy, LockScroll, Modal, OnDismiss, Open, Positioning, RestoreFocus, Role, SurfaceClass, SurfaceID, SurfaceStyle, Target, TrapFocus
- OverlayStack [struct]: BackdropZIndex, Depth, HandlesEscape, HandlesOutsideClick, ID, IsTop, Kind, LayerCount, SurfaceZIndex, TrapFocusActive
- OverlayStackOptions [struct]: BaseZIndex, CloseOnEscape, CloseOnOutsideClick, ID, Kind, Open, TrapFocus
- PortalProps [struct]: Child, Children, Target
- PortalTarget [struct]: Node, Selector
- Previous [struct]: methods Get, Ok
- ReactiveSource [interface]: ReactiveRegionSourceIDs
- Reducer [struct]: methods Dispatch, Get
- Ref [struct]: methods Get, Set
- SSRBootstrap [struct]: Atoms, Data, I18n, IDSeed, Route, Version
- SSRBootstrapBudget [struct]: BinaryErrorBytes, BinaryWarnBytes, InlineErrorBytes, InlineWarnBytes, SidecarErrorBytes, SidecarWarnBytes
- SSRBootstrapMetrics [struct]: Format, PayloadBytes, ScriptBytes
- SSRBootstrapReference [struct]: Format, URL, Version
- SSRBootstrapSizeReport [struct]: BinaryPayloadBytes, Errors, InlineScriptBytes, JSONPayloadBytes, Payloads, Recommendation, Version, Warnings
- SSRHydrationMetrics [struct]: DiscardedNodeCount, Duration, DurationNs, ExistingDOMNodeCount, Failed, Failure, FallbackCount, FinishedAt, MismatchCount, StartedAt, Strict
- SSRI18nBootstrap [struct]: Direction, FallbackLocale, Locale, Messages
- SSRI18nMessage [struct]: Default, Plural, PluralArg, Select, SelectArg, Text
- SSRObservabilityOptions [struct]: CorrelationID, OnEvent
- SSRObservation [struct]: Bootstrap, CorrelationID, Domain, Hydration, Name, Phase, Render, Timestamp
- SSRPayloadEncoding [defined]
- SSRPayloadEnvelope [struct]: Binary, Encoding, JSON, Kind, ReusePolicy, Revision, Scope, Target, Text, Version
- SSRPayloadFilter [struct]: Kind, Scope, Target
- SSRPayloadKind [defined]
- SSRPayloadMetadata [struct]: Encoding, Key, Kind, Legacy, ReusePolicy, Revision, Scope, Target, Version
- SSRPayloadOptions [struct]: Encoding, Kind, ReusePolicy, Revision, Scope, Target
- SSRPayloadReusePolicy [defined]
- SSRPayloadScope [defined]
- SSRPayloadValue [struct]: Encoding, Key, Kind, ReusePolicy, Revision, Scope, Target, Value
- SSRRenderMetrics [struct]: Duration, DurationNs
- SSRRouteBootstrap [struct]: Params, Path, Query
- SSRStateUpdate [struct]: CorrelationID, Deletes, Scope, Target, Upserts, Version
- ServerFormErrors [struct]: Error, Fields, Message; methods FormMessage
- State [struct]: methods Get, Set, Update
- Task [struct]: methods Cancel, Get, Start
- TaskState [struct]: Cancelled, Error, Ready, Running, Started, Value
- Throttled [struct]: methods Get, Pending
- Transition [struct]: methods Pending, Start
- WorkerTask [struct]: methods Cancel, Get, Start
- WorkerTaskState [struct]: Cancelled, Error, Progress, ProgressReady, Ready, Running, Started, Value

### html

Functions: A, Article, Aside, Blockquote, Br, Button, Code, CustomElement, DNSPrefetch, Dialog, Div, Em, Fieldset, Footer, Form, Fragment, H1, H2, H3, H4, H5, H6, Header, HiddenInput, Hr, Img, Input, Label, Legend, Li, Link, Main, ModulePreload, Nav, Option, P, Pre, Preconnect, Prefetch, Preload, Section, Select, Small, Span, Strong, Tag, Text, Textarea, Time, Ul

Types:

- CustomElementProps [struct]: Attributes, Presence, Properties, Props
- Props [struct]: Accept, Action, Alt, Aria, As, AutoComplete, AutoFocus, Checked, Class, Cols, Data, Disabled, EncType, For, Hidden, Href, ID, Key, Max, Method, Min, Multiple, Name, OnBlur, OnChange, OnClick, OnFocus, OnInput, OnKeyDown, OnKeyUp, OnSubmit, Placeholder, Raw, ReadOnly, Rel, Required, Role, Rows, Selected, Slot, Src, Step, Style, Target, Title, Type, Value

### state

Constants: LocalStorage, SessionStorage

Functions: ExportSnapshot, ImportSnapshot, LoadPersistentSnapshot, LoadSnapshot, MarshalSnapshotJSON, RestorePersistentSnapshot, RestoreSnapshot, SavePersistentSnapshot, SaveSnapshot, Select, UnmarshalSnapshotJSON, UseAtom, UseComputed, UseDerived

Types:

- Atom [struct]: methods Get, ReactiveRegionSourceIDs, Set, Text, Update
- Computed [struct]: methods Get
- Derived [struct]: methods Get, ReactiveRegionSourceIDs, Text
- Element [defined]
- PersistentSnapshotOptions [struct]: DatabaseName, DeleteOnCorruption, FallbackBackend, FallbackResolver, StoreName, StoreResolver
- Snapshot [defined]: methods Select
- StorageArea [defined]

### fetch

Constants: CacheBootstrapDataKey, CacheResumeAlwaysRefetch, CacheResumeStaleWhileRevalidate, CacheResumeTrustOnce, MutationDead, MutationQueued, MutationResolutionDead, MutationResolutionRemove, MutationResolutionReplace, MutationResolutionRetry, MutationRetrying

Functions: AsMutationConflictError, ConfigurePersistentCache, DisposeResource, Fetch, InspectCachedResources, InvalidateResource, IsMutationConflict, LoadCached, MutationConflictOf, NewMutationConflict, OpenMutationQueue, RestoreCacheBootstrap, ReturnChannel, SweepCachedResources, Upload, UseCachedResource, UseFetch, UseResource

Types:

- AsyncResource [struct]: methods Cancel, Get, Reload
- CacheBootstrap [struct]: Entries
- CacheBootstrapEntry [struct]: Key, ResumePolicy, StaleAfter, UpdatedAt, Value
- CacheOptions [struct]: DisposeAfter, MaxAge, Persist, StaleAfter
- CacheResumePolicy [defined]
- CachedResource [struct]: methods Cancel, Dispose, Get, Invalidate, Reload, Set, Update
- CachedResourceInspection [struct]: Key, LastError, LastLoaded, Loading, Ready, ResumePolicy, Stale, SubscriberCount, UpdatedAt
- CachedResourceState [struct]: Error, Loading, Ready, Stale, UpdatedAt, Value
- HTTPError [struct]: Body, Headers, Status, StatusText; methods Error
- MultipartBody [struct]: Fields, Files
- MultipartFile [struct]: FieldName, File, Filename
- MutationConflict [struct]: Code, Fields, LocalVersion, Message, RemoteVersion
- MutationConflictError [struct]: Conflict, Err; methods Error, Unwrap
- MutationConflictHandler [defined]
- MutationConflictResolution [struct]: Action, Draft, Message
- MutationDraft [struct]: Body, DedupKey, Headers, ID, Kind, Metadata, Method, URL
- MutationExecutor [defined]
- MutationQueue [struct]: methods Clear, Enqueue, List, Remove, Replay, ReplayWithOptions
- MutationQueueOptions [struct]: BaseDelay, DeleteOnCorruption, MaxAttempts, MaxDelay, Now, StorageKey, StorageResolver, StoreResolver
- MutationReplayOptions [struct]: ConflictHandler
- MutationReplayReport [struct]: Conflicts, DeadLetters, Deferred, Remaining, Resolved, Retried, Succeeded
- MutationResolutionAction [defined]
- MutationState [defined]
- Options [struct]: Body, Headers, Method
- PersistentCacheOptions [struct]: DatabaseName, FallbackBackend, FallbackResolver, StoreName, StoreResolver
- QueuedMutation [struct]: Attempts, Body, CreatedAt, DedupKey, Headers, ID, Kind, LastError, MaxAttempts, Metadata, Method, NextAttemptAt, State, URL, UpdatedAt
- Resource [struct]: methods Get, Refetch
- ResourceState [struct]: Error, Loading, Ready, Value
- Result [struct]: Data, Err, Headers, Status; methods DecodeJSON, Text
- State [defined]
- UploadUpdate [struct]: Done, LengthComputable, Loaded, Result, Total

### router

Constants: ReturnToParam

Functions: AllowNavigation, BlockNavigation, GetCurrentPath, GetRoute, GetRouter, InspectCurrentRoute, MetadataNode, Navigate, NavigateReplace, NewHashRouter, NewRouter, Outlet, PreserveReturnTo, ReadReturnTo, RedirectNavigation, RegisterRoute, RevalidateCurrentRoute, RouteWithElement, UseNavigate, UseParams, UseQuery, UseRevalidator, UseRouteData, UseSearchParams

Types:

- AsyncGuardFunc [defined]
- AsyncLeaveGuardFunc [defined]
- Attrs [defined]
- Component [defined]
- Element [defined]
- GuardDecision [struct]: Blocked, Denied, Reason, Redirect, Retryable
- GuardFunc [defined]
- GuardResult [struct]: Blocked, Reason, Redirect
- LeaveGuardFunc [defined]
- LoaderFunc [defined]
- Metadata [struct]: CanonicalURL, Description, Title
- Navigator [struct]: methods Navigate, Replace
- Options [struct]: Authorizing, BeforeEnter, BeforeEnterAsync, BeforeLeave, BeforeLeaveAsync, CanonicalURL, Description, Error, GuardPending, Layout, Loader, Loading, Redirect, Title, Unauthorized
- Params [struct]: methods Bool, Get, Has, Int, Values
- Query [struct]: methods Encode, Get, Has, Values
- Revalidator [struct]: methods Loading, Revalidate
- RouteContext [struct]: Params, Path, Query
- RouteInspection [struct]: Loading, Params, Path, Query
- Router [struct]: methods Current, GetCurrentRouterPath, GoGetRoute, GoRegisterRoute, HydrateMount, HydrateMountElement, IsRouteLoading, Mount, MountElement, Navigate, NavigateReplace, Register, RevalidateCurrentRoute
- RouterOptions [struct]: DefaultRoute
- SearchParams [struct]: methods Delete, Encode, Get, Has, Navigate, Replace, ReplaceAll, Set, Values

### devtools

Constants: SeverityError, SeverityInfo, SeverityWarning

Functions: CompareSnapshots, ExportSnapshotJSON, InspectMultiClient, Panel, ResetMultiClientInspection, SetMultiClientInspection, SnapshotNow, UseSnapshot

Types:

- Branch [struct]: CleanupDurationNs, CommitDurationNs, EffectDurationNs, Kind, Name, Path, SelfDurationNs, SubtreeDurationNs
- CacheEntry [struct]: Key, LastError, LastLoaded, Loading, Ready, ResumePolicy, Stale, SubscriberCount, UpdatedAt
- Classification [defined]
- Diagnostic [struct]: Classification, Code, ComponentStack, Consequence, Count, Docs, Message, Path, Recoverable, Remediation, Severity, Source, TopFrame
- Hook [struct]: Kind, Value
- Log [struct]: Classification, Code, Consequence, CorrelationID, Docs, Domain, Fields, Level, Message, Recoverable, Remediation, Timestamp, TopFrame
- LogLevel [defined]
- MultiClient [struct]: AuthorityView, Enabled, FailedPublishes, LocalPeerID, Peers, RecentTraffic, ResolvedTransport
- MultiClientFailure [struct]: Code, Message, Op, Target, Timestamp, Topic
- MultiClientPeer [struct]: App, Compatible, Encodings, ID, LastSeen, LeaseDeadline, ProtocolVersion, Role, State, Surface, Topics
- MultiClientTraffic [struct]: CorrelationID, Direction, Failed, Kind, LatencyMs, PeerID, Timestamp, Topic
- Node [struct]: Children, CleanupDurationNs, CommitDurationNs, Dirty, EffectCount, EffectDurationNs, FineGrained, HookCount, Hooks, Kind, Name, NeedsUpdate, ReactiveSource, SelfDurationNs, Signature, SubtreeDurationNs, UpdateOrigin
- PanelProps [struct]: InitiallyOpen, MaxDepth, RefreshInterval, Title
- Profiling [struct]: CleanupExecutions, CommitCount, EffectExecutions, FineGrainedCommits, FineGrainedDescendantHostCommits, FineGrainedDescendantTextCommits, HotBranches, LastCleanupDurationNs, LastCommitDurationNs, LastEffectDurationNs, LastRenderDurationNs, ProcessedUnits, RenderCalls, ScheduledFiberMarks, ScheduledGranularMarks, ScheduledRootUpdates, WorkLoopPasses
- Route [struct]: Loading, Params, Path, Query
- Severity [defined]
- Snapshot [struct]: Cache, Diagnostics, Logs, MultiClient, Profiling, Route, Stats, Tree
- SnapshotComparison [struct]: ChangedSections, CurrentFingerprint, CurrentSize, Equal, PreviousFingerprint, PreviousSize
- Stats [struct]: ComponentFibers, DirtyFibers, Effects, FineGrainedFibers, HookEntries, HostFibers, TextFibers, TotalFibers

### head

Functions: Compose, LinkRel, MetaName, MetaProperty, OpenGraph, Robots, SocialTags, Twitter

Types:

- SocialMetadata [struct]: Description, ImageURL, Title, TwitterCard, Type, URL

### plugin

Constants: CapabilityAsyncData, CapabilityDevtools, CapabilityForms, CapabilityRouter, CapabilitySSR, GuardAllow, GuardBlock, GuardRedirect, TierExperimental, TierInternal, TierStable, TierSupportedCompanion

Functions: Allow, Block, Define, NewHost, Redirect

Types:

- BootstrapPayload [struct]: Data, Namespace
- BootstrapProvider [defined]
- CacheKeyDecorator [defined]
- Capability [defined]
- CleanupFunc [defined]
- DefineFunc [defined]
- FormSubmission [struct]: ID, Intent, Values
- FormValidator [defined]
- GuardDecision [struct]: Outcome, Reason, Redirect
- GuardOutcome [defined]
- HeadProvider [defined]
- Host [struct]: methods AddBootstrapProvider, AddCacheKeyDecorator, AddFormValidator, AddHeadProvider, AddNavigationObserver, AddPanelProvider, AddRequestObserver, AddRouteGuard, AddSubmitObserver, BootstrapData, Capabilities, Close, DecorateCacheKey, EvaluateRoute, HeadNodes, NotifyNavigation, NotifyRequest, NotifySubmit, Panels, Plugins, Register, SetValue, ValidateForm, Value
- HostOptions [struct]: Capabilities
- Manifest [struct]: Description, ID, Requires, Tier, Version
- NavigationEvent [struct]: Path, Source
- NavigationObserver [defined]
- Panel [struct]: ID, Summary, Title
- PanelProvider [defined]
- Plugin [interface]: Manifest, Setup
- RequestEvent [struct]: Key, Phase, Source
- RequestObserver [defined]
- RouteGuard [defined]
- RouteRequest [struct]: Path, Tags
- SubmitObserver [defined]
- Tier [defined]
- ValidationIssue [struct]: Field, Message

### hotreload

Functions: Configure, Disable, Enable, Enabled, ExportSnapshot, ImportSnapshot, Prepare

Types:

- Config [struct]: AtomIDs, ResetKey