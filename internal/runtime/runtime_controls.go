package runtime

import (
	"fmt"
	"strings"
)

// UpdateLane names one runtime scheduling priority. Lower values run with
// larger work slices and win coalescing when multiple update causes arrive
// before a pending render commits.
type UpdateLane uint8

const (
	UpdateLaneSync UpdateLane = iota + 1
	UpdateLaneInput
	UpdateLaneDefault
	UpdateLaneTransition
	UpdateLaneBackground
)

// String returns the stable diagnostic label for one update lane.
func (parseLane UpdateLane) String() string {
	switch parseLane {
	case UpdateLaneSync:
		return "sync"
	case UpdateLaneInput:
		return "input"
	case UpdateLaneDefault:
		return "default"
	case UpdateLaneTransition:
		return "transition"
	case UpdateLaneBackground:
		return "background"
	default:
		return "default"
	}
}

// maxUnitsPerSlice adjusts reconciliation chunk size by priority lane.
func (parseLane UpdateLane) maxUnitsPerSlice(parseBase int) int {
	if parseBase <= 0 {
		parseBase = 1
	}
	switch parseLane {
	case UpdateLaneSync:
		return parseBase * 2
	case UpdateLaneInput:
		return parseBase
	case UpdateLaneTransition:
		if parseBase/2 < 1 {
			return 1
		}
		return parseBase / 2
	case UpdateLaneBackground:
		if parseBase/4 < 1 {
			return 1
		}
		return parseBase / 4
	default:
		return parseBase
	}
}

// RuntimeLimits bounds internal queues so long-lived applications fail soft
// instead of accumulating unbounded framework state.
type RuntimeLimits struct {
	MaxPendingEffectFibers int
	MaxQueuedUpdates       int
	MaxReplayEvents        int
	MaxDiagnostics         int
	MaxProfilingEvents     int
	MaxLogEntries          int
}

const (
	defaultMaxPendingEffectFibers = 1024
	defaultMaxQueuedUpdates       = 4096
	defaultMaxReplayEvents        = 2048
)

// withDefaults returns bounded defaults while preserving explicit positive overrides.
func (parseLimits RuntimeLimits) withDefaults() RuntimeLimits {
	if parseLimits.MaxPendingEffectFibers <= 0 {
		parseLimits.MaxPendingEffectFibers = defaultMaxPendingEffectFibers
	}
	if parseLimits.MaxQueuedUpdates <= 0 {
		parseLimits.MaxQueuedUpdates = defaultMaxQueuedUpdates
	}
	if parseLimits.MaxReplayEvents <= 0 {
		parseLimits.MaxReplayEvents = defaultMaxReplayEvents
	}
	return parseLimits
}

// applyGlobalRuntimeLimits applies process-wide buffers that predate runtime instances.
func applyGlobalRuntimeLimits(parseLimits RuntimeLimits) {
	if parseLimits.MaxDiagnostics > 0 {
		diagnosticsMu.Lock()
		maxDiagnosticEntries = parseLimits.MaxDiagnostics
		trimDiagnosticsLocked(maxDiagnosticEntries)
		diagnosticsMu.Unlock()
	}
}

type runtimeSchedulerState struct {
	pendingLane      UpdateLane
	currentLane      UpdateLane
	maxQueuedUpdates int
	enqueuedUpdates  int
	coalescedUpdates int
	// coalescedAtLimit counts how many updates arrived at the queue limit and
	// were merged into the pending render.
	//
	// Formerly droppedBackpressure, which was wrong in the way that matters: an
	// update is a request to re-render, re-render requests are idempotent, and
	// collapsing N of them produces the same frame. Nothing is dropped. A
	// counter named for data loss makes every reader investigate a loss that
	// never happened. See BudgetCoalesce.
	// updateArrivedInFlight records that an update was scheduled while a pass
	// was already walking the tree.
	//
	// Such an update cannot be served by the running pass — it may have visited
	// those fibers already — so the pass owes a follow-up. Without this the
	// update was folded into pendingLane and then nothing ran it: the fibers
	// stayed dirty, no pass was scheduled, and the tree simply stopped updating.
	//
	// The window is nearly closed without time-slicing, because a pass usually
	// completes inside one task. A frame budget opens it at every yield, which
	// is what froze the Example 201 core-* scenarios — the work loop ran two
	// passes, committed twice, and stopped with nothing rendered.
	updateArrivedInFlight bool
	coalescedAtLimit      int
	interruptedWork       int
	lastBackpressureLane  UpdateLane
	// lanes carries per-lane pending state and deferral deadlines (v5 P2.2).
	lanes laneState
}

func (parseState *runtimeSchedulerState) ensureDefaults(parseLimits RuntimeLimits) {
	if parseState == nil {
		return
	}
	parseLimits = parseLimits.withDefaults()
	parseState.maxQueuedUpdates = parseLimits.MaxQueuedUpdates
}

func (parseState *runtimeSchedulerState) beginScheduledLocked(parseLane UpdateLane) {
	if parseState == nil {
		return
	}
	parseState.pendingLane = normalizeUpdateLane(parseLane)
	parseState.currentLane = normalizeUpdateLane(parseLane)
	parseState.enqueuedUpdates++
	parseState.coalescedUpdates = 0
}

func (parseState *runtimeSchedulerState) finishScheduledLocked() {
	if parseState == nil {
		return
	}
	parseState.pendingLane = 0
	parseState.currentLane = 0
	parseState.coalescedUpdates = 0
}

func (parseRt *Runtime) coalesceScheduledUpdateLocked(parseLane UpdateLane) {
	if parseRt == nil {
		return
	}
	parseRt.schedulerState.ensureDefaults(parseRt.limits)
	parseRt.schedulerState.coalescedUpdates++
	if parseRt.schedulerState.pendingLane == 0 || normalizeUpdateLane(parseLane) < parseRt.schedulerState.pendingLane {
		parseRt.schedulerState.pendingLane = normalizeUpdateLane(parseLane)
	}
	// `nextUnitOfWork != wipRoot` is the "pass has actually started consuming
	// work" test (the same one FlushScheduledDiscreteWork uses). A pass that is
	// merely SCHEDULED still has nextUnitOfWork == wipRoot, and an update
	// arriving in that window needs no interrupt at all — the pass has not
	// visited anything yet, so it picks the new work up naturally. Treating
	// that window as an interrupt is what made the old rebuild look harmless
	// most of the time while being destructive in the narrow case that matters.
	isPassInFlight := parseRt.nextUnitOfWork != nil && parseRt.nextUnitOfWork != parseRt.wipRoot
	if isPassInFlight {
		// Owed regardless of lane. The interrupt branch below handles the case
		// where the new work is MORE urgent than the running pass; this handles
		// every case, including the common one where it is the same lane and the
		// old code did nothing at all.
		parseRt.schedulerState.updateArrivedInFlight = true
	}
	if isPassInFlight && parseRt.currentRoot != nil && parseRt.schedulerState.currentLane != 0 && normalizeUpdateLane(parseLane) < parseRt.schedulerState.currentLane {
		parseRt.schedulerState.interruptedWork++
		parseRt.schedulerState.currentLane = normalizeUpdateLane(parseLane)
		// v5 P2.5 (T12): mark, finish the pass, start the higher lane after the
		// commit — do NOT rebuild the work-in-progress root here.
		//
		// The old path called rebuildWIPRootForInterruptLocked, which reaches
		// acquireWorkInProgress(currentRoot). Because currentRoot.alternate IS
		// the in-flight wipRoot in the stable two-fiber cycle, that does
		// `*reused = Fiber{}` — zeroing the very root the work loop is walking,
		// clearing its child pointer. Landing between slices was survivable
		// (a clean restart); landing during performUnitOfWork was not: the loop
		// writes its own return value back over nextUnitOfWork, discarding the
		// restart, then commitRoot finds wipRoot.child == nil, commits nothing,
		// and installs a childless currentRoot. The next pass then diffs
		// against an empty tree and re-places DOM that is already mounted.
		//
		// Deferring costs at most one pass of latency for the higher lane and
		// removes the failure mode entirely.
		parseRt.pendingInterruptLane = normalizeUpdateLane(parseLane)
	}
	if parseRt.schedulerState.maxQueuedUpdates > 0 && parseRt.schedulerState.coalescedUpdates > parseRt.schedulerState.maxQueuedUpdates {
		parseRt.schedulerState.coalescedAtLimit++
		parseRt.schedulerState.lastBackpressureLane = normalizeUpdateLane(parseLane)
		parseRt.schedulerState.coalescedUpdates = parseRt.schedulerState.maxQueuedUpdates
		// R6: reported on the transition into the coalescing state, not on every
		// update that arrives while there. A burst of ten thousand updates past
		// the limit produced ten thousand identical warnings, whose formatting
		// cost landed on the render thread during the exact burst the warning
		// was about.
		if parseRt.schedulerState.coalescedAtLimit == 1 {
			ReportDiagnostic("runtime", DiagnosticWarning, "scheduled update queue reached its limit; further updates coalesce into the pending render (no updates are lost)")
		}
	}
}

func normalizeUpdateLane(parseLane UpdateLane) UpdateLane {
	switch parseLane {
	case UpdateLaneSync, UpdateLaneInput, UpdateLaneDefault, UpdateLaneTransition, UpdateLaneBackground:
		return parseLane
	default:
		return UpdateLaneDefault
	}
}

func laneForUpdateOrigin(parseOrigin string) UpdateLane {
	parseOrigin = strings.ToLower(strings.TrimSpace(parseOrigin))
	switch {
	case strings.Contains(parseOrigin, "sync"), strings.Contains(parseOrigin, "hydrate"), strings.Contains(parseOrigin, "render"):
		return UpdateLaneSync
	case strings.Contains(parseOrigin, "input"), strings.Contains(parseOrigin, "event"), strings.Contains(parseOrigin, "local-state"):
		return UpdateLaneInput
	case strings.Contains(parseOrigin, "transition"):
		return UpdateLaneTransition
	case strings.Contains(parseOrigin, "idle"), strings.Contains(parseOrigin, "background"):
		return UpdateLaneBackground
	default:
		return UpdateLaneDefault
	}
}

// interruptRestartIsSafe records that the P2.5 mark-and-defer path is in
// effect: an interrupt never mutates the root the work loop is walking.
//
// P1.2's real frame deadline asserts this before enabling itself, because live
// time-slicing multiplies the mid-pass yields that made T12 reachable — shipping
// the improvement without the fix would increase exposure to a bug that blanks
// the screen. Keeping the constant next to the code it describes means reverting
// P2.5 forces flipping it, so the gate cannot silently rot.
const interruptRestartIsSafe = true

// NOTE(v5 P2.5): rebuildWIPRootForInterruptLocked was REMOVED, not disabled.
//
// It rebuilt the work-in-progress root mid-pass on a higher-lane interrupt via
// acquireWorkInProgress(currentRoot). Because currentRoot.alternate IS the
// in-flight wipRoot in the two-fiber cycle, that zeroed the live root in place
// and cleared its child chain; when it landed inside performUnitOfWork the work
// loop overwrote the restart with its own return value, and commitRoot then
// installed a childless currentRoot — blanking the tree and causing the next
// pass to re-place already-mounted DOM.
//
// coalesceScheduledUpdateLocked now records pendingInterruptLane instead, and
// commitRoot starts the higher lane once the tree is consistent. Do not
// reintroduce an in-place rebuild: any restart that mutates the root the work
// loop is currently walking has this failure mode.

type SchedulerSnapshot struct {
	PendingLane      string
	CurrentLane      string
	EnqueuedUpdates  int
	CoalescedUpdates int
	// CoalescedAtLimit counts updates that arrived once the queue was full and
	// were merged into the pending render. NOT dropped — it was named
	// DroppedUpdates, and nothing is lost: re-render requests are idempotent.
	CoalescedAtLimit int
	InterruptedWork  int
	Backpressure     bool
}

// SchedulerSnapshot returns priority-lane and backpressure counters for diagnostics.
func (parseRt *Runtime) SchedulerSnapshot() SchedulerSnapshot {
	if parseRt == nil {
		return SchedulerSnapshot{}
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	return SchedulerSnapshot{
		PendingLane:      parseRt.schedulerState.pendingLane.String(),
		CurrentLane:      parseRt.schedulerState.currentLane.String(),
		EnqueuedUpdates:  parseRt.schedulerState.enqueuedUpdates,
		CoalescedUpdates: parseRt.schedulerState.coalescedUpdates,
		CoalescedAtLimit: parseRt.schedulerState.coalescedAtLimit,
		InterruptedWork:  parseRt.schedulerState.interruptedWork,
		Backpressure:     parseRt.schedulerState.coalescedAtLimit > 0,
	}
}

// StrictModeOptions enables development-time runtime checks.
type StrictModeOptions struct {
	Enabled                   bool
	DoubleRender              bool
	WarnSetStateDuringRender  bool
	RequireEffectCleanup      bool
	PanicOnViolation          bool
	ViolationDiagnosticSource string
}

func (parseOptions StrictModeOptions) withDefaults() StrictModeOptions {
	if !parseOptions.Enabled {
		return parseOptions
	}
	if parseOptions.ViolationDiagnosticSource == "" {
		parseOptions.ViolationDiagnosticSource = "runtime"
	}
	if !parseOptions.DoubleRender && !parseOptions.WarnSetStateDuringRender && !parseOptions.RequireEffectCleanup {
		parseOptions.DoubleRender = true
		parseOptions.WarnSetStateDuringRender = true
		parseOptions.RequireEffectCleanup = true
	}
	return parseOptions
}

func (parseRt *Runtime) reportStrictSetStateDuringRender(parseFiber *Fiber, parseHook string) {
	if parseRt == nil || !parseRt.strictMode.Enabled || !parseRt.strictMode.WarnSetStateDuringRender || GetCurrentFiber() == nil {
		return
	}
	parseMessage := fmt.Sprintf("strict mode detected %s state update during render", strings.TrimSpace(parseHook))
	parseRt.reportStrictViolation(parseFiber, parseMessage)
}

func (parseRt *Runtime) checkStrictEffectCleanupSymmetry(parseFiber *Fiber, parseCleanupIndex int, hasCleanup bool) {
	if parseRt == nil || !parseRt.strictMode.Enabled || !parseRt.strictMode.RequireEffectCleanup || parseFiber == nil || parseCleanupIndex < 0 {
		return
	}
	if parseFiber.hooks == nil {
		return
	}
	if parseCleanupIndex >= len(parseFiber.hooks.effectSeen) {
		parseNext := make([]bool, parseCleanupIndex+1)
		copy(parseNext, parseFiber.hooks.effectSeen)
		parseFiber.hooks.effectSeen = parseNext
	}
	if parseCleanupIndex >= len(parseFiber.hooks.effectHadCleanup) {
		parseNext := make([]bool, parseCleanupIndex+1)
		copy(parseNext, parseFiber.hooks.effectHadCleanup)
		parseFiber.hooks.effectHadCleanup = parseNext
	}
	if parseFiber.hooks.effectSeen[parseCleanupIndex] && parseFiber.hooks.effectHadCleanup[parseCleanupIndex] != hasCleanup {
		parseRt.reportStrictViolation(parseFiber, "strict mode detected an effect cleanup contract changing between renders")
	}
	parseFiber.hooks.effectSeen[parseCleanupIndex] = true
	parseFiber.hooks.effectHadCleanup[parseCleanupIndex] = hasCleanup
}

func (parseRt *Runtime) strictPreviewRender(parseFiber *Fiber) {
	if parseRt == nil || !parseRt.strictMode.Enabled || !parseRt.strictMode.DoubleRender || parseFiber == nil {
		return
	}
	parsePreviewHooks := &Hooks{owner: parseFiber}
	parsePrevHooks := parseFiber.hooks
	parseFiber.hooks = parsePreviewHooks
	setCurrentFiberOwned(parseFiber, parseRt.renderPassOwnerID())
	defer SetCurrentFiber(nil)
	defer func() {
		parseFiber.hooks = parsePrevHooks
		if parseRecovered := recover(); parseRecovered != nil {
			parseRt.reportStrictViolation(parseFiber, fmt.Sprintf("strict mode preview render panicked: %v", parseRecovered))
		}
		releaseHookResources(parsePreviewHooks)
	}()
	_, _ = renderFunctionComponentElement(parseFiber)
}

func renderFunctionComponentElement(parseFiber *Fiber) (*Element, bool) {
	if parseFiber == nil {
		return nil, false
	}
	if parseFn, parseOk := parseFiber.typeOf.(func() *Element); parseOk {
		return parseFn(), true
	}
	if parseFn, parseOk := parseFiber.typeOf.(func(map[string]any) *Element); parseOk {
		return parseFn(parseFiber.props), true
	}
	if parseFn, parseOk := parseFiber.typeOf.(func(Attrs) *Element); parseOk {
		return parseFn(Attrs(parseFiber.props)), true
	}
	if parseComponent, parseOk := parseFiber.typeOf.(*ComponentType); parseOk {
		return parseComponent.Render(parseFiber.props), true
	}
	return nil, false
}

func (parseRt *Runtime) reportStrictViolation(parseFiber *Fiber, parseMessage string) {
	parseSource := strings.TrimSpace(parseRt.strictMode.ViolationDiagnosticSource)
	if parseSource == "" {
		parseSource = "runtime"
	}
	ReportDiagnosticWithContext(parseSource, DiagnosticWarning, parseMessage, diagnosticPathForFiber(parseFiber), diagnosticComponentStack(parseFiber))
	if parseRt.strictMode.PanicOnViolation {
		panic(ActionableFrameworkPanic(ActionablePanicOptions{
			Source:         parseSource,
			Subject:        "strict mode",
			Message:        parseMessage,
			Path:           diagnosticPathForFiber(parseFiber),
			ComponentStack: diagnosticComponentStack(parseFiber),
			Consequence:    "strict mode turns this development-time runtime violation into a hard failure.",
		}))
	}
}

type replayUpdateKind string

const (
	replayUpdateKindRoot     replayUpdateKind = "root"
	replayUpdateKindFiber    replayUpdateKind = "fiber"
	replayUpdateKindGranular replayUpdateKind = "granular"
)

// ReplayEvent captures one deterministic scheduling event.
type ReplayEvent struct {
	Kind    string
	Path    []int
	Origin  string
	Lane    string
	Seq     int
	Dropped bool
}

type runtimeReplayState struct {
	recording bool
	events    []ReplayEvent
	nextSeq   int
	limit     int
	dropped   int
}

// StartReplayRecording clears and starts the runtime update replay buffer. It
// takes schedulerMu because the replay buffer is also written by
// recordReplayUpdate under that lock — accessing it unlocked is a data race.
func (parseRt *Runtime) StartReplayRecording() {
	if parseRt == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	parseRt.replay.limit = parseRt.limits.withDefaults().MaxReplayEvents
	parseRt.replay.events = parseRt.replay.events[:0]
	parseRt.replay.nextSeq = 0
	parseRt.replay.dropped = 0
	parseRt.replay.recording = true
}

// StopReplayRecording stops recording and returns a stable copy of captured events.
func (parseRt *Runtime) StopReplayRecording() []ReplayEvent {
	if parseRt == nil {
		return nil
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	parseRt.replay.recording = false
	return parseRt.replayEventsLocked()
}

// ReplayEvents returns a copy of captured deterministic scheduling events.
func (parseRt *Runtime) ReplayEvents() []ReplayEvent {
	if parseRt == nil {
		return nil
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	return parseRt.replayEventsLocked()
}

// replayEventsLocked copies the captured events; callers must hold schedulerMu.
func (parseRt *Runtime) replayEventsLocked() []ReplayEvent {
	if len(parseRt.replay.events) == 0 {
		return nil
	}
	parseEvents := make([]ReplayEvent, len(parseRt.replay.events))
	for parseIndex, parseEvent := range parseRt.replay.events {
		parseEvents[parseIndex] = parseEvent
		parseEvents[parseIndex].Path = append([]int(nil), parseEvent.Path...)
	}
	return parseEvents
}

func (parseRt *Runtime) recordReplayUpdate(parseKind replayUpdateKind, parsePath []int, parseOrigin string, parseLane UpdateLane) {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	parseRt.recordReplayUpdateLocked(parseKind, parsePath, parseOrigin, parseLane)
}

func (parseRt *Runtime) recordReplayUpdateLocked(parseKind replayUpdateKind, parsePath []int, parseOrigin string, parseLane UpdateLane) {
	if parseRt == nil || !parseRt.replay.recording {
		return
	}
	parseLimit := parseRt.replay.limit
	if parseLimit <= 0 {
		parseLimit = parseRt.limits.withDefaults().MaxReplayEvents
	}
	parseRt.replay.nextSeq++
	parseEvent := ReplayEvent{
		Kind:   string(parseKind),
		Path:   append([]int(nil), parsePath...),
		Origin: strings.TrimSpace(parseOrigin),
		Lane:   normalizeUpdateLane(parseLane).String(),
		Seq:    parseRt.replay.nextSeq,
	}
	if parseLimit > 0 && len(parseRt.replay.events) >= parseLimit {
		parseRt.replay.dropped++
		parseRt.replay.events = append(parseRt.replay.events[1:], parseEvent)
		parseRt.replay.events[len(parseRt.replay.events)-1].Dropped = true
		return
	}
	parseRt.replay.events = append(parseRt.replay.events, parseEvent)
}

func fiberPathIndexes(parseFiber *Fiber) []int {
	if parseFiber == nil {
		return nil
	}
	parseReversed := make([]int, 0, 8)
	for parseCursor := parseFiber; parseCursor != nil; parseCursor = parseCursor.parent {
		if parseCursor.parent == nil {
			break
		}
		parseIndex := 0
		for parseSibling := previousSibling(parseCursor); parseSibling != nil; parseSibling = previousSibling(parseSibling) {
			parseIndex++
		}
		parseReversed = append(parseReversed, parseIndex)
	}
	for parseLeft, parseRight := 0, len(parseReversed)-1; parseLeft < parseRight; parseLeft, parseRight = parseLeft+1, parseRight-1 {
		parseReversed[parseLeft], parseReversed[parseRight] = parseReversed[parseRight], parseReversed[parseLeft]
	}
	return parseReversed
}

func previousSibling(parseFiber *Fiber) *Fiber {
	if parseFiber == nil || parseFiber.parent == nil {
		return nil
	}
	for parseCursor := parseFiber.parent.child; parseCursor != nil; parseCursor = parseCursor.sibling {
		if parseCursor.sibling == parseFiber {
			return parseCursor
		}
	}
	return nil
}

// BeginReplayCapture is a compatibility alias for StartReplayRecording.
func (parseRt *Runtime) BeginReplayCapture() {
	parseRt.StartReplayRecording()
}

// EndReplayCapture is a compatibility alias for StopReplayRecording.
func (parseRt *Runtime) EndReplayCapture() []ReplayEvent {
	return parseRt.StopReplayRecording()
}

// ReplayUpdates replays captured scheduling events against the current fiber tree.
func (parseRt *Runtime) ReplayUpdates(parseEvents []ReplayEvent) {
	if parseRt == nil {
		return
	}
	// Pause recording around the replay so re-applied updates are not
	// re-captured. The flag is shared with recordReplayUpdate under
	// schedulerMu, so guard each access with it — but do NOT hold the lock
	// across the schedule* calls below, which acquire schedulerMu themselves.
	schedulerMu.Lock()
	wasRecording := parseRt.replay.recording
	parseRt.replay.recording = false
	schedulerMu.Unlock()
	defer func() {
		schedulerMu.Lock()
		parseRt.replay.recording = wasRecording
		schedulerMu.Unlock()
	}()
	for _, parseEvent := range parseEvents {
		switch replayUpdateKind(parseEvent.Kind) {
		case replayUpdateKindRoot:
			parseRt.scheduleUpdateWithLane(parseReplayLane(parseEvent.Lane), false)
		case replayUpdateKindFiber:
			if parseFiber := parseRt.fiberAtPath(parseEvent.Path); parseFiber != nil {
				parseRt.ScheduleUpdateForFiberWithOrigin(parseFiber, parseEvent.Origin)
			}
		case replayUpdateKindGranular:
			if parseFiber := parseRt.fiberAtPath(parseEvent.Path); parseFiber != nil {
				parseRt.ScheduleGranularUpdateForFiberWithOrigin(parseFiber, parseEvent.Origin)
			}
		}
	}
}

func parseReplayLane(parseLane string) UpdateLane {
	switch parseLane {
	case "sync":
		return UpdateLaneSync
	case "input":
		return UpdateLaneInput
	case "transition":
		return UpdateLaneTransition
	case "background":
		return UpdateLaneBackground
	default:
		return UpdateLaneDefault
	}
}

func (parseRt *Runtime) fiberAtPath(parsePath []int) *Fiber {
	if parseRt == nil || parseRt.currentRoot == nil {
		return nil
	}
	parseFiber := parseRt.currentRoot
	for _, parseIndex := range parsePath {
		parseFiber = parseFiber.child
		for parseStep := 0; parseStep < parseIndex && parseFiber != nil; parseStep++ {
			parseFiber = parseFiber.sibling
		}
		if parseFiber == nil {
			return nil
		}
	}
	return parseFiber
}
