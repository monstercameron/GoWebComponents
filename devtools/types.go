package devtools

import "time"

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type Classification string

// Diagnostic describes a runtime diagnostic entry surfaced in devtools.
type Diagnostic struct {
	Source         string
	Severity       Severity
	Classification Classification
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

type LogLevel string

type Log struct {
	Domain         string
	Level          LogLevel
	Classification Classification
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

// Hook describes one hook entry captured for a component node.
type Hook struct {
	Slot         int
	Kind         string
	Value        string
	Dependencies string
	Status       string
}

// Node describes one component or host node in the inspected runtime tree.
type Node struct {
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
	Hooks             []Hook
	Children          []Node
}

// Branch describes one hot subtree in profiling output.
type Branch struct {
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

type ProfilingPhaseTotals struct {
	RenderDurationNs  int64
	DiffDurationNs    int64
	CommitDurationNs  int64
	EffectDurationNs  int64
	CleanupDurationNs int64
}

type ProfilingEvent struct {
	Domain        string
	Name          string
	Phase         string
	Target        string
	CorrelationID string
	DurationNs    int64
	Timestamp     string
	Fields        map[string]string
}

type ComponentRenderTrace struct {
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

type FlamegraphFrame struct {
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

type StartupProfiling struct {
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
	RouteBudgets               []RouteStartupBudget
}

type RouteStartupBudget struct {
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

type HydrationDebug struct {
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

type BoundaryInspection struct {
	Entries []Boundary
}

type Boundary struct {
	Name          string
	Kind          string
	Direction     string
	Transport     string
	Encoding      string
	Scope         string
	Target        string
	Status        string
	CorrelationID string
	SizeBytes     int
	InlineBytes   int
	BinaryBytes   int
	Notes         []string
	Redacted      []string
	Downgraded    []string
	Rejected      []string
}

type Coordination struct {
	Workers         []WorkerJob
	SyncEvents      []SyncEvent
	Replay          []ReplayEntry
	QueueEntries    []SyncQueueEntry
	SyncHealth      []SyncHealthEntry
	Reconnect       ReconnectStatus
	Conflict        ConflictState
	LastReplayError string
}

type WorkerJob struct {
	Name        string
	URL         string
	Kind        string
	Status      string
	RequestID   string
	Progress    string
	Result      string
	Error       string
	Running     bool
	Ready       bool
	Cancelled   bool
	Correlation string
}

type SyncEvent struct {
	Channel     string
	Transport   string
	Direction   string
	Topic       string
	Target      string
	Status      string
	Error       string
	Correlation string
	Timestamp   time.Time
}

type ReplayEntry struct {
	ID            string
	Kind          string
	Method        string
	URL           string
	Owner         string
	State         string
	LastError     string
	Attempts      int
	MaxAttempts   int
	NextAttemptAt time.Time
	UpdatedAt     time.Time
}

type SyncQueueEntry struct {
	ID            string
	Entity        string
	Operation     string
	Owner         string
	State         string
	URL           string
	Attempts      int
	MaxAttempts   int
	LastError     string
	QueuedAt      time.Time
	UpdatedAt     time.Time
}

type SyncHealthEntry struct {
	Entity     string
	Owner      string
	Status     string
	Version    string
	PendingOps int
	LastSyncAt time.Time
	LastError  string
}

type ReconnectStatus struct {
	State       string
	Transport   string
	Attempts    int
	MaxAttempts int
	NextRetryAt time.Time
	LastChange  time.Time
	IsConnected bool
}

type ConflictState struct {
	Entity     string
	Owner      string
	Status     string
	Strategy   string
	DetectedAt time.Time
	LastError  string
}

type ExtensionSection struct {
	Name    string
	Summary map[string]string
	Lines   []string
}

// Stats summarizes the current inspected runtime tree.
type Stats struct {
	TotalFibers       int
	DirtyFibers       int
	ComponentFibers   int
	HostFibers        int
	TextFibers        int
	FineGrainedFibers int
	HookEntries       int
	Effects           int
}

// Profiling summarizes runtime profiling counters and hot branches.
type Profiling struct {
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
	PhaseTotals                      ProfilingPhaseTotals
	ComponentRenders                 []ComponentRenderTrace
	RecentEvents                     []ProfilingEvent
	FlamegraphFrames                 []FlamegraphFrame
	Startup                          StartupProfiling
	HotBranches                      []Branch
}
type Route struct {
	Path         string
	Query        map[string][]string
	Params       map[string]string
	Loading      bool
	Stack        []RouteStack
	Loaders      []RouteLoader
	LastRedirect RouteRedirect
	Metadata     RouteMetadata
}

type RouteStack struct {
	ID             string
	Path           string
	Params         map[string]string
	HasLoader      bool
	HasBeforeEnter bool
	HasBeforeLeave bool
	Metadata       RouteMetadata
}

type RouteLoader struct {
	Key     string
	Path    string
	Pending bool
	HasData bool
	Error   string
}

type RouteRedirect struct {
	Cause string
	From  string
	To    string
}

type RouteMetadata struct {
	Title        string
	Description  string
	CanonicalURL string
}

type CacheEntry struct {
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

type MultiClientPeer struct {
	ID              string
	App             string
	Surface         string
	Role            string
	State           string
	LeaseDeadline   time.Time
	LastSeen        time.Time
	ProtocolVersion string
	Encodings       []string
	Topics          []string
	Compatible      bool
}

type MultiClientTraffic struct {
	Direction     string
	Kind          string
	Topic         string
	PeerID        string
	CorrelationID string
	LatencyMs     int
	Timestamp     time.Time
	Failed        bool
}

type MultiClientFailure struct {
	Op        string
	Topic     string
	Target    string
	Code      string
	Message   string
	Timestamp time.Time
}

type MultiClient struct {
	Enabled           bool
	LocalPeerID       string
	ResolvedTransport string
	AuthorityView     map[string]string
	Peers             []MultiClientPeer
	RecentTraffic     []MultiClientTraffic
	FailedPublishes   []MultiClientFailure
}

// Snapshot is the top-level devtools inspection payload.
type Snapshot struct {
	Route        Route
	Cache        []CacheEntry
	MultiClient  MultiClient
	Boundaries   BoundaryInspection
	Coordination Coordination
	Kernel       KernelSnapshot
	Extensions   []ExtensionSection
	Tree         *Node
	Stats        Stats
	Profiling    Profiling
	Hydration    HydrationDebug
	Diagnostics  []Diagnostic
	Logs         []Log
}

// SnapshotComparison summarizes how two inspection snapshots differ.
type SnapshotComparison struct {
	Equal               bool
	ChangedSections     []string
	PreviousFingerprint string
	CurrentFingerprint  string
	PreviousSize        int
	CurrentSize         int
}

// PanelProps configures the embeddable devtools panel.
type PanelProps struct {
	Title           string
	InitiallyOpen   bool
	RefreshInterval time.Duration
	MaxDepth        int
}

// ErrorOverlayProps configures the embeddable in-browser error overlay.
type ErrorOverlayProps struct {
	Title           string
	RefreshInterval time.Duration
	MaxItems        int
}

// ErrorOverlayIssue is one actionable failure surfaced by the browser overlay.
type ErrorOverlayIssue struct {
	Severity Severity
	Source   string
	Code     string
	Message  string
	TopFrame string
	Path     string
	Docs     string
}

// ErrorOverlayActionContext is passed to one overlay recovery action handler.
type ErrorOverlayActionContext struct {
	Snapshot Snapshot
	Issue    ErrorOverlayIssue
}

// ErrorOverlayAction describes one app-owned recovery action for overlay issues.
type ErrorOverlayAction struct {
	Label        string
	MatchCodes   []string
	MatchSources []string
	Run          func(ErrorOverlayActionContext)
}

// TraceCapture stores one labeled devtools snapshot for export, comparison, and replay.
type TraceCapture struct {
	Label      string
	CapturedAt string
	Snapshot   Snapshot
}

// BugCaptureBundle packages one local debugging artifact for later replay or attachment.
type BugCaptureBundle struct {
	Version    int
	Label      string
	CapturedAt string
	Trace      TraceCapture
}

// SupportDiagnosticBundle packages one redacted debugging artifact for support workflows.
type SupportDiagnosticBundle struct {
	Version    int
	Sanitized  bool
	Label      string
	CapturedAt string
	Trace      TraceCapture
}

// KernelSnapshot is the top-level devtools view of plugin kernel state.
type KernelSnapshot struct {
	APIVersion string
	Plugins    []KernelPluginHealth
	Events     []KernelPluginEvent
}

// KernelPluginHealth stores one plugin health summary.
type KernelPluginHealth struct {
	ID         string
	State      string
	Reason     string
	Message    string
	ErrorCount int
}

// KernelPluginEvent stores one plugin diagnostic summary.
type KernelPluginEvent struct {
	PluginID  string
	Operation string
	Reason    string
	Message   string
	When      time.Time
}
