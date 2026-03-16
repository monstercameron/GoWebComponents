package devtools

import "time"

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Diagnostic describes a runtime diagnostic entry surfaced in devtools.
type Diagnostic struct {
	Source   string
	Severity Severity
	Message  string
	Count    int
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

// Branch describes one hot subtree in profiling output.
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

// Stats summarizes the current inspected runtime tree.
type Stats struct {
	TotalFibers     int
	DirtyFibers     int
	ComponentFibers int
	HostFibers      int
	TextFibers      int
	HookEntries     int
	Effects         int
}

// Profiling summarizes runtime profiling counters and hot branches.
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

// Snapshot is the top-level devtools inspection payload.
type Snapshot struct {
	Route       Route
	Tree        *Node
	Stats       Stats
	Profiling   Profiling
	Diagnostics []Diagnostic
}

// PanelProps configures the embeddable devtools panel.
type PanelProps struct {
	Title           string
	InitiallyOpen   bool
	RefreshInterval time.Duration
	MaxDepth        int
}
