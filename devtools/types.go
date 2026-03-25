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
	Kind  string
	Value string
}

// Node describes one component or host node in the inspected runtime tree.
type Node struct {
	Name              string
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
	HydrationDurationNs        int64
	StartupCommitDurationNs    int64
	FirstInteractionDurationNs int64
	FirstInteractionCaptured   bool
	FirstInteractionEvent      string
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
	Path    string
	Query   map[string][]string
	Params  map[string]string
	Loading bool
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
	Route       Route
	Cache       []CacheEntry
	MultiClient MultiClient
	Tree        *Node
	Stats       Stats
	Profiling   Profiling
	Diagnostics []Diagnostic
	Logs        []Log
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
