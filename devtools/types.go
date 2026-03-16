package devtools

import "time"

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type Diagnostic struct {
	Source   string
	Severity Severity
	Message  string
	Count    int
}

type Hook struct {
	Kind  string
	Value string
}

type Node struct {
	Name              string
	Kind              string
	Dirty             bool
	NeedsUpdate       bool
	EffectCount       int
	HookCount         int
	CommitDurationNs  int64
	EffectDurationNs  int64
	CleanupDurationNs int64
	SelfDurationNs    int64
	SubtreeDurationNs int64
	Hooks             []Hook
	Children          []Node
}

type Branch struct {
	Name              string
	Kind              string
	Path              string
	CommitDurationNs  int64
	EffectDurationNs  int64
	CleanupDurationNs int64
	SelfDurationNs    int64
	SubtreeDurationNs int64
}

type Stats struct {
	TotalFibers     int
	DirtyFibers     int
	ComponentFibers int
	HostFibers      int
	TextFibers      int
	HookEntries     int
	Effects         int
}

type Profiling struct {
	RenderCalls           int
	ScheduledRootUpdates  int
	ScheduledFiberMarks   int
	WorkLoopPasses        int
	ProcessedUnits        int
	CommitCount           int
	EffectExecutions      int
	CleanupExecutions     int
	LastRenderDurationNs  int64
	LastCommitDurationNs  int64
	LastEffectDurationNs  int64
	LastCleanupDurationNs int64
	HotBranches           []Branch
}
type Route struct {
	Path    string
	Query   map[string][]string
	Params  map[string]string
	Loading bool
}

type Snapshot struct {
	Route       Route
	Tree        *Node
	Stats       Stats
	Profiling   Profiling
	Diagnostics []Diagnostic
}

type PanelProps struct {
	Title           string
	InitiallyOpen   bool
	RefreshInterval time.Duration
	MaxDepth        int
}
