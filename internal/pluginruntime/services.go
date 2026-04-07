package pluginruntime

import (
	"fmt"
	"sort"
	"time"
)

// ServiceRegistration stores one built-in service value for kernel bootstrap.
type ServiceRegistration struct {
	Key   ServiceKey
	Value any
}

// RuntimeSnapshot stores one normalized runtime inspection snapshot.
type RuntimeSnapshot struct {
	Meta        SnapshotMeta
	Root        *RuntimeNode
	Stats       RuntimeStats
	Profiling   RuntimeProfiling
	Hydration   RuntimeHydration
	Diagnostics []RuntimeDiagnostic
	Logs        []RuntimeLog
}

// RuntimeNode stores one normalized runtime tree node.
type RuntimeNode struct {
	Name              string
	Path              string
	Kind              string
	Dirty             bool
	NeedsUpdate       bool
	FineGrained       bool
	ReactiveSource    string
	UpdateOrigin      string
	EffectCount       int
	HookCount         int
	Signature         string
	RenderDurationNs  int64
	DiffDurationNs    int64
	CommitDurationNs  int64
	EffectDurationNs  int64
	CleanupDurationNs int64
	SelfDurationNs    int64
	SubtreeDurationNs int64
	Hooks             []RuntimeHook
	Children          []RuntimeNode
}

// RuntimeHook stores one normalized hook inspection value.
type RuntimeHook struct {
	Slot         int
	Kind         string
	Value        string
	Dependencies string
	Status       string
}

// RuntimeStats stores one normalized runtime stats snapshot.
type RuntimeStats struct {
	TotalFibers       int
	DirtyFibers       int
	ComponentFibers   int
	HostFibers        int
	TextFibers        int
	FineGrainedFibers int
	HookEntries       int
	Effects           int
}

// RuntimeProfiling stores one normalized runtime profiling snapshot.
type RuntimeProfiling struct {
	RenderCalls                      int
	ScheduledRootUpdates             int
	ScheduledFiberMarks              int
	ScheduledGranularMarks           int
	WorkLoopPasses                   int
	ProcessedUnits                   int
	CommitCount                      int
	FineGrainedCommits               int
	FineGrainedDescendantHostCommits int
	FineGrainedDescendantTextCommits int
	EffectExecutions                 int
	CleanupExecutions                int
	LastRenderDurationNs             int64
	LastCommitDurationNs             int64
	LastEffectDurationNs             int64
	LastCleanupDurationNs            int64
	PhaseTotals                      RuntimeProfilingPhaseTotals
	ComponentRenders                 []RuntimeComponentRenderTrace
	RecentEvents                     []RuntimeProfilingEvent
	FlamegraphFrames                 []RuntimeFlamegraphFrame
	Startup                          RuntimeStartupProfiling
	HotBranches                      []RuntimeBranch
}

// RuntimeProfilingPhaseTotals stores attributed profiling phase time totals.
type RuntimeProfilingPhaseTotals struct {
	RenderDurationNs  int64
	DiffDurationNs    int64
	CommitDurationNs  int64
	EffectDurationNs  int64
	CleanupDurationNs int64
}

// RuntimeComponentRenderTrace stores per-component render activity.
type RuntimeComponentRenderTrace struct {
	Name                    string
	Path                    string
	RenderCount             int
	RerenderCount           int
	LastTrigger             string
	LastRenderDurationNs    int64
	TotalRenderDurationNs   int64
	AverageRenderDurationNs int64
	LastRenderedAt          string
	TriggerCounts           map[string]int
}

// RuntimeProfilingEvent stores one profiling event record.
type RuntimeProfilingEvent struct {
	Domain        string
	Name          string
	Phase         string
	Target        string
	CorrelationID string
	DurationNs    int64
	Timestamp     string
	Fields        map[string]string
}

// RuntimeFlamegraphFrame stores one normalized flamegraph frame.
type RuntimeFlamegraphFrame struct {
	Name              string
	Kind              string
	Path              string
	Depth             int
	StartNs           int64
	DurationNs        int64
	SelfDurationNs    int64
	RenderDurationNs  int64
	DiffDurationNs    int64
	CommitDurationNs  int64
	EffectDurationNs  int64
	CleanupDurationNs int64
}

// RuntimeStartupProfiling stores startup and hydration milestones.
type RuntimeStartupProfiling struct {
	Mode                       string
	StartedAt                  string
	BootstrapReadDurationNs    int64
	WASMTransferBytes          int64
	WASMDecodedBytes           int64
	BootstrapDecodedBytes      int64
	CacheWarmupDurationNs      int64
	ServiceWorkerOverheadNs    int64
	InitialRouteDataBytes      int64
	HydrationDurationNs        int64
	StartupCommitDurationNs    int64
	FirstInteractionDurationNs int64
	FirstInteractionCaptured   bool
	FirstInteractionEvent      string
	RouteBudgets               []RuntimeRouteStartupBudget
}

// RuntimeRouteStartupBudget stores one grouped startup budget sample.
type RuntimeRouteStartupBudget struct {
	RouteFamily                       string
	LastRoutePath                     string
	SampleCount                       int
	AverageBootstrapReadDurationNs    int64
	AverageWASMTransferBytes          int64
	AverageWASMDecodedBytes           int64
	AverageBootstrapDecodedBytes      int64
	AverageCacheWarmupDurationNs      int64
	AverageServiceWorkerOverheadNs    int64
	AverageInitialRouteDataBytes      int64
	AverageHydrationDurationNs        int64
	AverageStartupCommitDurationNs    int64
	AverageFirstInteractionDurationNs int64
}

// RuntimeBranch stores one normalized hot branch.
type RuntimeBranch struct {
	Name              string
	Kind              string
	Path              string
	RenderDurationNs  int64
	DiffDurationNs    int64
	CommitDurationNs  int64
	EffectDurationNs  int64
	CleanupDurationNs int64
	SelfDurationNs    int64
	SubtreeDurationNs int64
}

// RuntimeHydration stores one normalized hydration summary.
type RuntimeHydration struct {
	CorrelationID        string
	StartedAt            string
	FinishedAt           string
	DurationNs           int64
	ExistingDOMNodeCount int
	FallbackCount        int
	MismatchCount        int
	DiscardedNodeCount   int
	Strict               bool
	Failed               bool
	Failure              string
	RecentMessages       []string
}

// RuntimeDiagnostic stores one normalized runtime diagnostic.
type RuntimeDiagnostic struct {
	Source         string
	Severity       string
	Classification string
	Code           string
	Docs           string
	Remediation    string
	Recoverable    bool
	TopFrame       string
	Consequence    string
	Message        string
	Count          int
	Path           string
	ComponentStack []string
	Fields         map[string]string
}

// RuntimeLog stores one normalized runtime log entry.
type RuntimeLog struct {
	Domain         string
	Level          string
	Classification string
	Code           string
	Docs           string
	Remediation    string
	Recoverable    bool
	TopFrame       string
	Consequence    string
	Message        string
	Timestamp      string
	CorrelationID  string
	Fields         map[string]string
}

// RouteSnapshot stores one normalized current-route snapshot.
type RouteSnapshot struct {
	Meta         SnapshotMeta
	Path         string
	Query        map[string][]string
	Params       map[string]string
	Loading      bool
	Stack        []RouteStackEntry
	Loaders      []RouteLoaderEntry
	LastRedirect RouteRedirect
	Metadata     RouteMetadata
}

// RouteStackEntry stores one route stack entry.
type RouteStackEntry struct {
	ID             string
	Path           string
	Params         map[string]string
	HasLoader      bool
	HasBeforeEnter bool
	HasBeforeLeave bool
	Metadata       RouteMetadata
}

// RouteLoaderEntry stores one current route loader entry.
type RouteLoaderEntry struct {
	Key     string
	Path    string
	Pending bool
	HasData bool
	Error   string
}

// RouteRedirect stores one last-redirect summary.
type RouteRedirect struct {
	Cause string
	From  string
	To    string
}

// RouteMetadata stores one normalized route metadata summary.
type RouteMetadata struct {
	Title        string
	Description  string
	CanonicalURL string
}

// FetchSnapshot stores one normalized fetch-cache summary.
type FetchSnapshot struct {
	Meta    SnapshotMeta
	Entries []FetchCacheEntry
}

// FetchCacheEntry stores one normalized cache entry.
type FetchCacheEntry struct {
	Key             string
	Loading         bool
	Ready           bool
	Stale           bool
	LastError       string
	UpdatedAt       time.Time
	LastLoaded      time.Time
	SubscriberCount int
	OwnerPaths      []string
	ResumePolicy    string
}

// DiagnosticsSnapshot stores aggregated runtime and plugin diagnostics.
type DiagnosticsSnapshot struct {
	Meta             SnapshotMeta
	Runtime          []RuntimeDiagnostic
	PluginReports    []HealthReport
	PluginEvents     []PluginDiagnostic
	KernelAPIVersion string
}

// RuntimeInspectionService exposes normalized runtime inspection reads.
type RuntimeInspectionService interface {
	GetRuntimeSnapshot(QueryBudget) (RuntimeSnapshot, error)
}

// RouteService exposes normalized route inspection and command APIs.
type RouteService interface {
	GetRouteSnapshot(QueryBudget) (RouteSnapshot, error)
	NavigateRoute(string, CommandOptions) error
	RevalidateRoute(CommandOptions) error
	RetryRouteLoader(string, CommandOptions) error
}

// FetchService exposes normalized fetch-cache inspection and command APIs.
type FetchService interface {
	GetFetchSnapshot(QueryBudget) (FetchSnapshot, error)
	ClearFetchEntry(string, CommandOptions) error
	RevalidateFetchEntry(string, CommandOptions) error
}

// DiagnosticsService exposes aggregated diagnostic reads.
type DiagnosticsService interface {
	GetDiagnosticsSnapshot(QueryBudget) (DiagnosticsSnapshot, error)
}

// DOMNodeSnapshot stores one normalized DOM node summary.
type DOMNodeSnapshot struct {
	ID         string
	Tag        string
	Text       string
	Attributes map[string]string
	Children   []DOMNodeSnapshot
}

// DOMSnapshot stores one normalized DOM inspection result.
type DOMSnapshot struct {
	Meta SnapshotMeta
	Root *DOMNodeSnapshot
}

// DOMService exposes bounded DOM inspection and focus helpers.
type DOMService interface {
	GetDOMSnapshot(QueryBudget) (DOMSnapshot, error)
	HighlightNode(string, CommandOptions) error
	ScrollNodeIntoView(string, CommandOptions) error
}

// StyleSnapshot stores one style inspection summary.
type StyleSnapshot struct {
	Meta       SnapshotMeta
	Variables  map[string]string
	Stylesheet []string
}

// StylePatchHandle stores one reversible style patch handle.
type StylePatchHandle interface {
	RemoveStylePatch() error
}

// StyleService exposes style inspection and reversible patch APIs.
type StyleService interface {
	GetStyleSnapshot(QueryBudget) (StyleSnapshot, error)
	ApplyStyleVariables(map[string]string, CommandOptions) (StylePatchHandle, error)
}

// EventRecord stores one bounded normalized event record.
type EventRecord struct {
	Type       string
	Phase      string
	Target     string
	Current    string
	OccurredAt time.Time
}

// EventSnapshot stores one bounded event ring snapshot.
type EventSnapshot struct {
	Meta   SnapshotMeta
	Events []EventRecord
}

// EventService exposes event observation reads.
type EventService interface {
	GetEventSnapshot(QueryBudget) (EventSnapshot, error)
}

// AssetEntry stores one normalized asset/cache summary entry.
type AssetEntry struct {
	Key    string
	Status string
	Bytes  int64
}

// AssetSnapshot stores one normalized asset-management summary.
type AssetSnapshot struct {
	Meta    SnapshotMeta
	Entries []AssetEntry
}

// AssetService exposes asset inspection and invalidation commands.
type AssetService interface {
	GetAssetSnapshot(QueryBudget) (AssetSnapshot, error)
	RefreshAssets(CommandOptions) error
}

// SecurityFinding stores one normalized security finding.
type SecurityFinding struct {
	Code     string
	Message  string
	Severity string
}

// SecuritySnapshot stores one normalized security summary.
type SecuritySnapshot struct {
	Meta     SnapshotMeta
	Findings []SecurityFinding
}

// SecurityService exposes advisory security inspection.
type SecurityService interface {
	GetSecuritySnapshot(QueryBudget) (SecuritySnapshot, error)
}

// CaptureSnapshot stores one normalized capture/export summary.
type CaptureSnapshot struct {
	Meta     SnapshotMeta
	Sessions []string
}

// CaptureService exposes capture and replay lifecycle reads.
type CaptureService interface {
	GetCaptureSnapshot(QueryBudget) (CaptureSnapshot, error)
}

// Runtime2MetaSnapshot stores optional runtime2 capability metadata.
type Runtime2MetaSnapshot struct {
	Meta         SnapshotMeta
	Capabilities map[string]bool
	Regions      []Runtime2RegionSnapshot
	Diagnostics  []Runtime2Diagnostic
}

// Runtime2RegionSnapshot stores one normalized runtime2 region-status summary.
type Runtime2RegionSnapshot struct {
	RegionInstanceID            string
	RegionMode                  string
	AssignedWorkerShard         string
	RendererID                  string
	Epoch                       uint64
	IsHydrationComplete         bool
	HasHydratedShellAnchor      bool
	HasPostHydrationAttached    bool
	LastSnapshotVersion         uint64
	LastDispatchedVersion       uint64
	LastCommittedVersion        uint64
	TransportTier               string
	HasSnapshotDowngrade        bool
	SnapshotDowngradePath       string
	SnapshotDowngradeReason     string
	HasPatchDowngrade           bool
	PatchDowngradePath          string
	PatchDowngradeReason        string
	DroppedStalePatchCount      uint64
	IgnoredStaleDiagnosticCount uint64
	RepairTriggeredRemountCount uint64
	FallbackReason              string
	DispatchToPatchReadyNs      uint64
	DispatchToCommitNs          uint64
	PatchReadyToCommitNs        uint64
	DiagnosticCount             int
}

// Runtime2Diagnostic stores one normalized runtime2 diagnostic record.
type Runtime2Diagnostic struct {
	RegionInstanceID string
	Type             string
	Text             string
	TransportTier    string
	ShardID          string
	FallbackDomain   string
	FallbackReason   string
	DowngradePath    string
	DowngradeReason  string
	QueueNs          uint64
	RenderNs         uint64
	SnapshotBytes    uint64
	PatchBytes       uint64
}

// Runtime2MetaService exposes optional runtime2 capability metadata.
type Runtime2MetaService interface {
	GetRuntime2MetaSnapshot(QueryBudget) (Runtime2MetaSnapshot, error)
}

// ResolveServiceAs resolves one service and casts it to the requested type.
func ResolveServiceAs[T any](parseKernel *Kernel, parseKey ServiceKey) (T, error) {
	var buildZero T
	if parseKernel == nil {
		return buildZero, fmt.Errorf("pluginruntime: kernel is unavailable")
	}
	getService, hasService := parseKernel.ResolveService(parseKey)
	if !hasService {
		return buildZero, fmt.Errorf("pluginruntime: service %q is unavailable", parseKey)
	}
	getTyped, hasTyped := getService.(T)
	if !hasTyped {
		return buildZero, fmt.Errorf("pluginruntime: service %q has unexpected type %T", parseKey, getService)
	}
	return getTyped, nil
}

// cloneServiceRegistrations returns one cloned service-registration slice.
func cloneServiceRegistrations(parseRegistrations []ServiceRegistration) []ServiceRegistration {
	if len(parseRegistrations) == 0 {
		return nil
	}
	buildClone := append([]ServiceRegistration(nil), parseRegistrations...)
	sort.SliceStable(buildClone, func(parseI int, parseJ int) bool {
		return buildClone[parseI].Key < buildClone[parseJ].Key
	})
	return buildClone
}
