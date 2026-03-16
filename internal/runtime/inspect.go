package runtime

import (
	"fmt"
	"reflect"
	goRuntime "runtime"
	"sort"
	"strings"
	"sync"
)

type DiagnosticSeverity string

const (
	DiagnosticInfo    DiagnosticSeverity = "info"
	DiagnosticWarning DiagnosticSeverity = "warning"
	DiagnosticError   DiagnosticSeverity = "error"
)

type Diagnostic struct {
	Source   string
	Severity DiagnosticSeverity
	Message  string
	Count    int
}

type HookSnapshot struct {
	Kind  string
	Value string
}

type FiberSnapshot struct {
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
	Hooks             []HookSnapshot
	Children          []FiberSnapshot
}

type HotBranchSnapshot struct {
	Name              string
	Kind              string
	Path              string
	CommitDurationNs  int64
	EffectDurationNs  int64
	CleanupDurationNs int64
	SelfDurationNs    int64
	SubtreeDurationNs int64
}

type InspectionStats struct {
	TotalFibers     int
	DirtyFibers     int
	ComponentFibers int
	HostFibers      int
	TextFibers      int
	HookEntries     int
	Effects         int
}

type ProfilingSnapshot struct {
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
	HotBranches           []HotBranchSnapshot
}

type InspectionSnapshot struct {
	Root        *FiberSnapshot
	Stats       InspectionStats
	Profiling   ProfilingSnapshot
	Diagnostics []Diagnostic
}

var (
	diagnosticsMu   sync.Mutex
	diagnosticIndex = map[string]int{}
	diagnostics     []Diagnostic
)

func ReportDiagnostic(source string, severity DiagnosticSeverity, message string) {
	trimmedSource := strings.TrimSpace(source)
	if trimmedSource == "" {
		trimmedSource = "runtime"
	}
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return
	}

	key := string(severity) + "|" + trimmedSource + "|" + trimmedMessage

	diagnosticsMu.Lock()
	defer diagnosticsMu.Unlock()
	if index, ok := diagnosticIndex[key]; ok {
		diagnostics[index].Count++
		return
	}

	diagnosticIndex[key] = len(diagnostics)
	diagnostics = append(diagnostics, Diagnostic{
		Source:   trimmedSource,
		Severity: severity,
		Message:  trimmedMessage,
		Count:    1,
	})
}

func GetDiagnostics() []Diagnostic {
	diagnosticsMu.Lock()
	defer diagnosticsMu.Unlock()
	clone := make([]Diagnostic, len(diagnostics))
	copy(clone, diagnostics)
	return clone
}

func ClearDiagnostics() {
	diagnosticsMu.Lock()
	defer diagnosticsMu.Unlock()
	diagnosticIndex = map[string]int{}
	diagnostics = nil
}

func (rt *Runtime) Inspect() InspectionSnapshot {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()

	snapshot := InspectionSnapshot{
		Diagnostics: GetDiagnostics(),
	}
	if rt == nil || rt.currentRoot == nil {
		return snapshot
	}

	root, stats := inspectFiberTree(rt.currentRoot)
	snapshot.Root = root
	snapshot.Stats = stats
	snapshot.Profiling = ProfilingSnapshot{
		RenderCalls:           rt.profiling.renderCalls,
		ScheduledRootUpdates:  rt.profiling.scheduledRootUpdates,
		ScheduledFiberMarks:   rt.profiling.scheduledFiberMarks,
		WorkLoopPasses:        rt.profiling.workLoopPasses,
		ProcessedUnits:        rt.profiling.processedUnits,
		CommitCount:           rt.profiling.commitCount,
		EffectExecutions:      rt.profiling.effectExecutions,
		CleanupExecutions:     rt.profiling.cleanupExecutions,
		LastRenderDurationNs:  rt.profiling.lastRenderDurationNs,
		LastCommitDurationNs:  rt.profiling.lastCommitDurationNs,
		LastEffectDurationNs:  rt.profiling.lastEffectDurationNs,
		LastCleanupDurationNs: rt.profiling.lastCleanupDurationNs,
		HotBranches:           collectHotBranches(root, 5),
	}
	return snapshot
}

func inspectFiberTree(fiber *Fiber) (*FiberSnapshot, InspectionStats) {
	if fiber == nil {
		return nil, InspectionStats{}
	}

	kind, name := describeFiber(fiber)
	hooks := inspectHooks(fiber.hooks)
	node := &FiberSnapshot{
		Name:              name,
		Kind:              kind,
		Dirty:             fiber.dirty,
		NeedsUpdate:       fiber.needsUpdate,
		EffectCount:       len(fiber.effects),
		HookCount:         len(hooks),
		CommitDurationNs:  fiber.commitDurationNs,
		EffectDurationNs:  fiber.effectDurationNs,
		CleanupDurationNs: fiber.cleanupDurationNs,
		Hooks:             hooks,
	}
	node.SelfDurationNs = node.CommitDurationNs + node.EffectDurationNs + node.CleanupDurationNs

	stats := InspectionStats{
		TotalFibers: 1,
		HookEntries: len(hooks),
		Effects:     len(fiber.effects),
	}
	if fiber.dirty || fiber.needsUpdate {
		stats.DirtyFibers++
	}

	childSubtreeDurationNs := int64(0)

	switch kind {
	case "component":
		stats.ComponentFibers++
	case "host", "root":
		stats.HostFibers++
	case "text":
		stats.TextFibers++
	}

	for child := fiber.child; child != nil; child = child.sibling {
		childSnapshot, childStats := inspectFiberTree(child)
		if childSnapshot != nil {
			node.Children = append(node.Children, *childSnapshot)
			childSubtreeDurationNs += childSnapshot.SubtreeDurationNs
		}
		stats.TotalFibers += childStats.TotalFibers
		stats.DirtyFibers += childStats.DirtyFibers
		stats.ComponentFibers += childStats.ComponentFibers
		stats.HostFibers += childStats.HostFibers
		stats.TextFibers += childStats.TextFibers
		stats.HookEntries += childStats.HookEntries
		stats.Effects += childStats.Effects
	}
	node.SubtreeDurationNs = node.SelfDurationNs + childSubtreeDurationNs

	return node, stats
}

func collectHotBranches(root *FiberSnapshot, limit int) []HotBranchSnapshot {
	if root == nil || limit <= 0 {
		return nil
	}

	branches := make([]HotBranchSnapshot, 0, limit)
	var walk func(node *FiberSnapshot, path []string)
	walk = func(node *FiberSnapshot, path []string) {
		if node == nil {
			return
		}

		nextPath := append(append([]string(nil), path...), node.Name)
		if node.Kind != "root" && node.SubtreeDurationNs > 0 {
			branches = append(branches, HotBranchSnapshot{
				Name:              node.Name,
				Kind:              node.Kind,
				Path:              strings.Join(nextPath, " > "),
				CommitDurationNs:  node.CommitDurationNs,
				EffectDurationNs:  node.EffectDurationNs,
				CleanupDurationNs: node.CleanupDurationNs,
				SelfDurationNs:    node.SelfDurationNs,
				SubtreeDurationNs: node.SubtreeDurationNs,
			})
		}

		for index := range node.Children {
			walk(&node.Children[index], nextPath)
		}
	}

	walk(root, nil)
	sort.SliceStable(branches, func(i, j int) bool {
		if branches[i].SubtreeDurationNs == branches[j].SubtreeDurationNs {
			return branches[i].SelfDurationNs > branches[j].SelfDurationNs
		}
		return branches[i].SubtreeDurationNs > branches[j].SubtreeDurationNs
	})
	if len(branches) > limit {
		branches = branches[:limit]
	}
	return branches
}

func describeFiber(fiber *Fiber) (string, string) {
	if fiber == nil {
		return "unknown", "unknown"
	}

	switch value := fiber.typeOf.(type) {
	case string:
		switch value {
		case "ROOT":
			return "root", "ROOT"
		case "TEXT_ELEMENT":
			return "text", previewValue(fiber.textContent)
		default:
			return "host", value
		}
	default:
		return "component", describeCallable(value)
	}
}

func describeCallable(value interface{}) string {
	rv := reflect.ValueOf(value)
	if rv.IsValid() && rv.Kind() == reflect.Func {
		if fn := goRuntime.FuncForPC(rv.Pointer()); fn != nil {
			name := fn.Name()
			if index := strings.LastIndex(name, "/"); index >= 0 {
				name = name[index+1:]
			}
			if index := strings.LastIndex(name, "."); index >= 0 {
				name = name[index+1:]
			}
			return name
		}
	}
	if value == nil {
		return "nil"
	}
	return reflect.TypeOf(value).String()
}

func inspectHooks(hooks *Hooks) []HookSnapshot {
	if hooks == nil {
		return nil
	}

	result := make([]HookSnapshot, 0, len(hooks.states)/2+len(hooks.memos)+len(hooks.refs)+len(hooks.ids)+len(hooks.fetches)+len(hooks.atoms)+len(hooks.callbacks)+len(hooks.deps))
	for index := 0; index+1 < len(hooks.states); index += 2 {
		result = append(result, HookSnapshot{Kind: "state", Value: previewValue(hooks.states[index])})
	}
	for _, memo := range hooks.memos {
		result = append(result, HookSnapshot{Kind: "memo", Value: previewValue(memo.value)})
	}
	for _, ref := range hooks.refs {
		if ref == nil {
			result = append(result, HookSnapshot{Kind: "ref", Value: "<nil>"})
			continue
		}
		result = append(result, HookSnapshot{Kind: "ref", Value: previewValue(ref.Current)})
	}
	for _, id := range hooks.ids {
		result = append(result, HookSnapshot{Kind: "id", Value: id})
	}
	for _, atom := range hooks.atoms {
		result = append(result, HookSnapshot{Kind: "atom", Value: atom})
	}
	for _, callback := range hooks.callbacks {
		result = append(result, HookSnapshot{Kind: "callback", Value: describeCallable(callback.fn)})
	}
	for _, fetch := range hooks.fetches {
		result = append(result, HookSnapshot{Kind: "fetch", Value: fmt.Sprintf("url=%q loading=%t error=%q", fetch.url, fetch.state.Loading, fetch.state.Error)})
	}
	for _, deps := range hooks.deps {
		result = append(result, HookSnapshot{Kind: "effect", Value: fmt.Sprintf("deps=%d", len(deps))})
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Kind < result[j].Kind
	})
	return result
}

func previewValue(value interface{}) string {
	if value == nil {
		return "<nil>"
	}

	switch typed := value.(type) {
	case string:
		trimmed := typed
		if len(trimmed) > 48 {
			trimmed = trimmed[:45] + "..."
		}
		return fmt.Sprintf("%q", trimmed)
	case fmt.Stringer:
		return typed.String()
	case error:
		return typed.Error()
	}

	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return "<invalid>"
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return fmt.Sprintf("%s(len=%d)", rv.Type(), rv.Len())
	case reflect.Map:
		return fmt.Sprintf("%s(len=%d)", rv.Type(), rv.Len())
	case reflect.Struct:
		return rv.Type().String()
	case reflect.Pointer:
		if rv.IsNil() {
			return fmt.Sprintf("%s(nil)", rv.Type())
		}
		return fmt.Sprintf("%s", rv.Type())
	case reflect.Func:
		return describeCallable(value)
	default:
		return fmt.Sprintf("%v", value)
	}
}
