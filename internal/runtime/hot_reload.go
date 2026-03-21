package runtime

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
)

// HotReloadSnapshot captures the component-local state that can be restored
// after a compatible hot reload.
type HotReloadSnapshot struct {
	Components []HotReloadComponentSnapshot `json:"components,omitempty"`
}

// HotReloadRestorePlan controls whether hot-reload restore should preserve all
// compatible component snapshots or selectively drop changed subtrees.
type HotReloadRestorePlan struct {
	Selective         bool
	ChangedIdentities []string
}

// HotReloadRestoreDecision reports whether selective restore was applied or the
// runtime fell back to the legacy full compatible restore path.
type HotReloadRestoreDecision struct {
	Strategy     string
	UnsafeReason string
}

// HotReloadComponentSnapshot stores the serializable hook slots for a single
// function component render in preorder traversal order.
type HotReloadComponentSnapshot struct {
	Signature     ComponentSignature       `json:"signature"`
	Path          string                   `json:"path,omitempty"`
	IdentityTrail []string                 `json:"identityTrail,omitempty"`
	States        []interface{}            `json:"states,omitempty"`
	Memos         []HotReloadMemoSnapshot  `json:"memos,omitempty"`
	Refs          []interface{}            `json:"refs,omitempty"`
	IDs           []string                 `json:"ids,omitempty"`
	Fetches       []HotReloadFetchSnapshot `json:"fetches,omitempty"`
}

// HotReloadFetchSnapshot stores the serializable portion of a fetch hook.
type HotReloadFetchSnapshot struct {
	URL   string     `json:"url"`
	State FetchState `json:"state"`
}

var hotReloadSerializableKinds = map[string]struct{}{
	"state": {},
	"memo":  {},
	"ref":   {},
	"id":    {},
	"fetch": {},
}

func (rt *Runtime) CaptureHotReloadSnapshot() HotReloadSnapshot {
	if rt == nil || rt.currentRoot == nil {
		return HotReloadSnapshot{}
	}

	snapshot := HotReloadSnapshot{}
	captureHotReloadComponentSnapshots(rt.currentRoot, &snapshot.Components)
	return snapshot
}

func (rt *Runtime) RestoreHotReloadSnapshot(snapshot HotReloadSnapshot) {
	_ = rt.RestoreHotReloadSnapshotWithPlan(snapshot, HotReloadRestorePlan{})
}

func (rt *Runtime) RestoreHotReloadSnapshotWithPlan(snapshot HotReloadSnapshot, plan HotReloadRestorePlan) HotReloadRestoreDecision {
	if rt == nil {
		return HotReloadRestoreDecision{}
	}

	if len(snapshot.Components) == 0 {
		rt.pendingHotReloadComponents = nil
		rt.pendingHotReloadIndex = 0
		if rt.pendingHotReloadByPath != nil {
			clear(rt.pendingHotReloadByPath)
		}
		rt.pendingHotReloadSelective = false
		return HotReloadRestoreDecision{}
	}

	decision := HotReloadRestoreDecision{}
	if plan.Selective {
		decision = rt.restoreSelectiveHotReloadSnapshot(snapshot, plan)
		if decision.Strategy == "selective" {
			return decision
		}
	}

	rt.pendingHotReloadComponents = append(rt.pendingHotReloadComponents[:0], snapshot.Components...)
	rt.pendingHotReloadIndex = 0
	if rt.pendingHotReloadByPath != nil {
		clear(rt.pendingHotReloadByPath)
	}
	rt.pendingHotReloadSelective = false
	if decision.Strategy == "" {
		decision.Strategy = "legacy"
	}
	return decision
}

func (rt *Runtime) HasPendingHotReloadSnapshot() bool {
	if rt == nil {
		return false
	}
	if rt.pendingHotReloadSelective {
		return len(rt.pendingHotReloadByPath) > 0
	}
	return rt.pendingHotReloadIndex < len(rt.pendingHotReloadComponents)
}

func (rt *Runtime) nextHotReloadComponentSnapshot() *HotReloadComponentSnapshot {
	if rt == nil || rt.pendingHotReloadIndex >= len(rt.pendingHotReloadComponents) {
		return nil
	}

	snapshot := rt.pendingHotReloadComponents[rt.pendingHotReloadIndex]
	rt.pendingHotReloadIndex++
	return &snapshot
}

func (rt *Runtime) matchingHotReloadComponentSnapshot(fiber *Fiber) *HotReloadComponentSnapshot {
	if rt == nil {
		return nil
	}
	if !rt.pendingHotReloadSelective {
		return rt.nextHotReloadComponentSnapshot()
	}
	if rt.pendingHotReloadByPath == nil {
		return nil
	}
	path := hotReloadFiberPath(fiber)
	if strings.TrimSpace(path) == "" {
		return nil
	}
	snapshot, ok := rt.pendingHotReloadByPath[path]
	if !ok {
		return nil
	}
	delete(rt.pendingHotReloadByPath, path)
	return &snapshot
}

func (rt *Runtime) restoreSelectiveHotReloadSnapshot(snapshot HotReloadSnapshot, plan HotReloadRestorePlan) HotReloadRestoreDecision {
	changed := make(map[string]struct{}, len(plan.ChangedIdentities))
	for _, identity := range plan.ChangedIdentities {
		identity = strings.TrimSpace(identity)
		if identity != "" {
			changed[identity] = struct{}{}
		}
	}
	if len(changed) == 0 {
		return HotReloadRestoreDecision{Strategy: "legacy", UnsafeReason: "selective restore requested without changed component identities"}
	}
	if rt.pendingHotReloadByPath == nil {
		rt.pendingHotReloadByPath = make(map[string]HotReloadComponentSnapshot, len(snapshot.Components))
	} else {
		clear(rt.pendingHotReloadByPath)
	}

	for _, component := range snapshot.Components {
		path := strings.TrimSpace(component.Path)
		if path == "" {
			clear(rt.pendingHotReloadByPath)
			return HotReloadRestoreDecision{Strategy: "legacy", UnsafeReason: "saved snapshot is missing stable component paths"}
		}
		if _, exists := rt.pendingHotReloadByPath[path]; exists {
			clear(rt.pendingHotReloadByPath)
			return HotReloadRestoreDecision{Strategy: "legacy", UnsafeReason: "saved snapshot contains duplicate component paths"}
		}
		if hotReloadSnapshotTouchesChangedIdentity(component, changed) {
			continue
		}
		rt.pendingHotReloadByPath[path] = component
	}

	rt.pendingHotReloadComponents = nil
	rt.pendingHotReloadIndex = 0
	rt.pendingHotReloadSelective = true
	return HotReloadRestoreDecision{Strategy: "selective"}
}

func captureHotReloadComponentSnapshots(fiber *Fiber, snapshots *[]HotReloadComponentSnapshot) {
	if fiber == nil {
		return
	}

	if kind, _ := describeFiber(fiber); kind == "component" {
		if snapshot := captureHotReloadComponentSnapshot(fiber); snapshot != nil {
			*snapshots = append(*snapshots, *snapshot)
		}
	}

	for child := fiber.child; child != nil; child = child.sibling {
		captureHotReloadComponentSnapshots(child, snapshots)
	}
}

func captureHotReloadComponentSnapshot(fiber *Fiber) *HotReloadComponentSnapshot {
	if fiber == nil {
		return nil
	}

	signature := buildComponentSignature(fiber, fiber.hooks)
	if signature == nil {
		return nil
	}

	snapshot := &HotReloadComponentSnapshot{
		Signature:     *signature,
		Path:          hotReloadFiberPath(fiber),
		IdentityTrail: hotReloadIdentityTrail(fiber),
	}
	if fiber.hooks != nil {
		if len(fiber.hooks.states) > 0 {
			snapshot.States = make([]interface{}, 0, len(fiber.hooks.states)/2)
			for index := 0; index+1 < len(fiber.hooks.states); index += 2 {
				snapshot.States = append(snapshot.States, normalizeHotReloadValue(fiber.hooks.states[index]))
			}
		}
		if len(fiber.hooks.memos) > 0 {
			snapshot.Memos = make([]HotReloadMemoSnapshot, 0, len(fiber.hooks.memos))
			for _, memo := range fiber.hooks.memos {
				snapshot.Memos = append(snapshot.Memos, HotReloadMemoSnapshot{
					Value: normalizeHotReloadValue(memo.value),
					Deps:  normalizeHotReloadDeps(memo.deps),
				})
			}
		}
		if len(fiber.hooks.refs) > 0 {
			snapshot.Refs = make([]interface{}, 0, len(fiber.hooks.refs))
			for _, ref := range fiber.hooks.refs {
				if ref == nil {
					snapshot.Refs = append(snapshot.Refs, nil)
					continue
				}
				snapshot.Refs = append(snapshot.Refs, normalizeHotReloadValue(ref.Current))
			}
		}
		if len(fiber.hooks.ids) > 0 {
			snapshot.IDs = append(snapshot.IDs, fiber.hooks.ids...)
		}
		if len(fiber.hooks.fetches) > 0 {
			snapshot.Fetches = make([]HotReloadFetchSnapshot, 0, len(fiber.hooks.fetches))
			for _, fetch := range fiber.hooks.fetches {
				snapshot.Fetches = append(snapshot.Fetches, HotReloadFetchSnapshot{
					URL:   fetch.url,
					State: normalizeHotReloadFetchState(fetch.state),
				})
			}
		}
	}

	return snapshot
}

func hotReloadSnapshotTouchesChangedIdentity(snapshot HotReloadComponentSnapshot, changed map[string]struct{}) bool {
	if len(changed) == 0 {
		return false
	}
	for _, identity := range snapshot.IdentityTrail {
		if _, ok := changed[strings.TrimSpace(identity)]; ok {
			return true
		}
	}
	return false
}

func hotReloadIdentityTrail(fiber *Fiber) []string {
	if fiber == nil {
		return nil
	}
	trail := make([]string, 0, 8)
	for current := fiber; current != nil; current = current.parent {
		signature := buildComponentSignature(current, nil)
		if signature == nil {
			continue
		}
		identity := strings.TrimSpace(signature.identityKey())
		if identity == "" {
			continue
		}
		trail = append(trail, identity)
	}
	for left, right := 0, len(trail)-1; left < right; left, right = left+1, right-1 {
		trail[left], trail[right] = trail[right], trail[left]
	}
	return trail
}

func hotReloadFiberPath(fiber *Fiber) string {
	if fiber == nil {
		return ""
	}
	segments := make([]string, 0, 8)
	for current := fiber; current != nil && current.parent != nil; current = current.parent {
		segment := hotReloadFiberPathSegment(current)
		if segment == "" {
			continue
		}
		segments = append(segments, segment)
	}
	for left, right := 0, len(segments)-1; left < right; left, right = left+1, right-1 {
		segments[left], segments[right] = segments[right], segments[left]
	}
	return strings.Join(segments, "/")
}

func hotReloadFiberPathSegment(fiber *Fiber) string {
	if fiber == nil || fiber.parent == nil {
		return ""
	}
	if key := hotReloadFiberKeySegment(fiber); key != "" {
		return key
	}

	index := 0
	for sibling := fiber.parent.child; sibling != nil && sibling != fiber; sibling = sibling.sibling {
		index++
	}
	if signature := buildComponentSignature(fiber, nil); signature != nil {
		identity := strings.TrimSpace(signature.identityKey())
		if identity != "" {
			return fmt.Sprintf("%s@%d", identity, index)
		}
	}
	kind, name := describeFiber(fiber)
	if strings.TrimSpace(name) == "" {
		name = kind
	}
	return fmt.Sprintf("%s@%d", name, index)
}

func hotReloadFiberKeySegment(fiber *Fiber) string {
	if fiber == nil || fiber.props == nil {
		return ""
	}
	key, ok := fiber.props["key"]
	if !ok || key == nil {
		return ""
	}
	return "key:" + fmt.Sprint(key)
}

func normalizeHotReloadFetchState(state FetchState) FetchState {
	state.Data = normalizeHotReloadValue(state.Data)
	if strings.TrimSpace(state.Error) == "" {
		state.Error = ""
	}
	return state
}

func normalizeHotReloadValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		normalized := make(map[string]interface{}, len(typed))
		for key, nested := range typed {
			normalized[key] = normalizeHotReloadValue(nested)
		}
		return normalized
	case []interface{}:
		normalized := make([]interface{}, len(typed))
		for index, nested := range typed {
			normalized[index] = normalizeHotReloadValue(nested)
		}
		return normalized
	case float64:
		if math.Trunc(typed) == typed {
			return int(typed)
		}
		return typed
	default:
		return value
	}
}

func normalizeHotReloadDeps(deps []interface{}) []interface{} {
	if len(deps) == 0 {
		return nil
	}

	normalized := make([]interface{}, len(deps))
	for index, dep := range deps {
		normalized[index] = normalizeHotReloadValue(dep)
	}
	return normalized
}

func coerceHotReloadValue(value interface{}, targetType reflect.Type) (interface{}, bool) {
	if targetType == nil {
		return value, true
	}

	if value == nil {
		return reflect.Zero(targetType).Interface(), true
	}

	valueType := reflect.TypeOf(value)
	if valueType.AssignableTo(targetType) {
		return value, true
	}
	if valueType.ConvertibleTo(targetType) {
		return reflect.ValueOf(value).Convert(targetType).Interface(), true
	}
	if targetType.Kind() == reflect.Interface && targetType.NumMethod() == 0 {
		return value, true
	}

	data, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}

	var target reflect.Value
	if targetType.Kind() == reflect.Pointer {
		target = reflect.New(targetType.Elem())
	} else {
		target = reflect.New(targetType)
	}

	if err := json.Unmarshal(data, target.Interface()); err != nil {
		return nil, false
	}

	if targetType.Kind() == reflect.Pointer {
		return target.Interface(), true
	}

	return target.Elem().Interface(), true
}

// NormalizeHotReloadValue exposes the hot-reload value normalizer for bridge
// code that receives JSON-decoded snapshots.
func NormalizeHotReloadValue(value interface{}) interface{} {
	return normalizeHotReloadValue(value)
}

func componentSnapshotCompatible(snapshot *HotReloadComponentSnapshot, fiber *Fiber) bool {
	if snapshot == nil || fiber == nil {
		return false
	}

	current := buildComponentSignature(fiber, nil)
	if current == nil {
		return false
	}

	if snapshot.Signature.Kind != current.Kind {
		return false
	}
	if snapshot.Signature.identityKey() != current.identityKey() {
		return false
	}
	if snapshot.Signature.Key != current.Key {
		return false
	}
	return true
}

func componentSnapshotFullyCompatible(snapshot *HotReloadComponentSnapshot, fiber *Fiber) bool {
	if !componentSnapshotCompatible(snapshot, fiber) {
		return false
	}

	current := buildComponentSignature(fiber, fiber.hooks)
	if current == nil {
		return false
	}

	return snapshot.Signature.CompatibleWith(*current)
}

func componentSnapshotSerializableCompatible(snapshot *HotReloadComponentSnapshot, fiber *Fiber) bool {
	if !componentSnapshotCompatible(snapshot, fiber) {
		return false
	}

	current := buildComponentSignature(fiber, fiber.hooks)
	if current == nil {
		return false
	}

	previousSerializable := filterHotReloadSerializableKinds(snapshot.Signature.HookKinds)
	currentSerializable := filterHotReloadSerializableKinds(current.HookKinds)
	return hotReloadHookKindPrefixCompatible(previousSerializable, currentSerializable)
}

func filterHotReloadSerializableKinds(kinds []string) []string {
	if len(kinds) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		if _, ok := hotReloadSerializableKinds[kind]; ok {
			filtered = append(filtered, kind)
		}
	}
	return filtered
}

func hotReloadHookKindPrefixCompatible(previous, current []string) bool {
	if len(previous) > len(current) {
		previous, current = current, previous
	}
	for index, kind := range previous {
		if current[index] != kind {
			return false
		}
	}
	return true
}

func reportHotReloadFallbackDiagnostic(snapshot *HotReloadComponentSnapshot, fiber *Fiber) {
	if snapshot == nil || fiber == nil {
		return
	}

	label := hotReloadComponentLabel(snapshot, fiber)
	ReportDiagnosticWithContext(
		"runtime",
		DiagnosticWarning,
		fmt.Sprintf("hot reload fell back to remount for %s: %s", label, hotReloadFallbackReason(snapshot, fiber)),
		diagnosticPathForFiber(fiber),
		diagnosticComponentStack(fiber),
	)
}

func hotReloadComponentLabel(snapshot *HotReloadComponentSnapshot, fiber *Fiber) string {
	if current := buildComponentSignature(fiber, nil); current != nil {
		if name := strings.TrimSpace(current.Name); name != "" {
			return name
		}
	}
	if snapshot != nil {
		if name := strings.TrimSpace(snapshot.Signature.Name); name != "" {
			return name
		}
		if qualified := strings.TrimSpace(snapshot.Signature.QualifiedName); qualified != "" {
			return qualified
		}
	}
	return "component"
}

func hotReloadFallbackReason(snapshot *HotReloadComponentSnapshot, fiber *Fiber) string {
	if snapshot == nil || fiber == nil {
		return "saved state could not be matched"
	}

	current := buildComponentSignature(fiber, nil)
	if current == nil {
		return "current component identity could not be inspected"
	}

	if snapshot.Signature.Kind != current.Kind {
		return fmt.Sprintf("component kind changed from %q to %q", snapshot.Signature.Kind, current.Kind)
	}
	if snapshot.Signature.identityKey() != current.identityKey() {
		return fmt.Sprintf("component identity changed from %q to %q", snapshot.Signature.identityKey(), current.identityKey())
	}
	if snapshot.Signature.Key != current.Key {
		return fmt.Sprintf("component key changed from %q to %q", snapshot.Signature.Key, current.Key)
	}

	current = buildComponentSignature(fiber, fiber.hooks)
	if current == nil {
		return "current hook signature could not be inspected"
	}
	if len(snapshot.Signature.HookKinds) != len(current.HookKinds) {
		return fmt.Sprintf(
			"hook count changed from %d to %d (%s -> %s)",
			len(snapshot.Signature.HookKinds),
			len(current.HookKinds),
			hotReloadHookKindsSummary(snapshot.Signature.HookKinds),
			hotReloadHookKindsSummary(current.HookKinds),
		)
	}
	for index, kind := range snapshot.Signature.HookKinds {
		if current.HookKinds[index] != kind {
			return fmt.Sprintf(
				"hook order changed from %s to %s",
				hotReloadHookKindsSummary(snapshot.Signature.HookKinds),
				hotReloadHookKindsSummary(current.HookKinds),
			)
		}
	}

	return fmt.Sprintf("component signature changed from %s to %s", snapshot.Signature.Summary(), current.Summary())
}

func hotReloadHookKindsSummary(kinds []string) string {
	if len(kinds) == 0 {
		return "no hooks"
	}
	return strings.Join(kinds, " > ")
}

func releaseHookResources(hooks *Hooks) {
	if hooks == nil {
		return
	}

	for index := range hooks.funcs {
		handler := hooks.funcs[index]
		if releasable, ok := handler.wrapper.(interface{ Release() }); ok {
			releasable.Release()
		}
		hooks.funcs[index] = funcHandlerValue{}
	}
}

func (rt *Runtime) PrepareForHotReload() {
	if rt == nil || rt.currentRoot == nil {
		return
	}

	rt.runCleanups(rt.currentRoot)
	prepareFiberForHotReload(rt, rt.currentRoot)
	rt.wipRoot = nil
	rt.nextUnitOfWork = nil
	if rt.deletions != nil {
		rt.deletions = rt.deletions[:0]
	}
	rt.updateScheduled = false
	rt.pendingBoundaryRecovery = false
}

func prepareFiberForHotReload(rt *Runtime, fiber *Fiber) {
	if fiber == nil {
		return
	}

	if fiber.hooks != nil {
		reportHotReloadRestartActivity(fiber)
		releaseHookResources(fiber.hooks)
		for index := range fiber.hooks.cleanups {
			fiber.hooks.cleanups[index] = nil
		}
		fiber.hooks.hotReloadRestore = nil
	}
	if rt != nil {
		rt.CleanupAtomSubscriptions(fiber)
	}
	fiber.effects = nil
	fiber.eventCallbacks = nil

	prepareFiberForHotReload(rt, fiber.child)
	prepareFiberForHotReload(rt, fiber.sibling)
}

func reportHotReloadRestartActivity(fiber *Fiber) {
	if fiber == nil || fiber.hooks == nil {
		return
	}

	for _, fetch := range fiber.hooks.fetches {
		if !fetch.state.Loading {
			continue
		}

		fields := map[string]string{}
		if path := strings.TrimSpace(diagnosticPathForFiber(fiber)); path != "" {
			fields["path"] = path
		}
		if url := strings.TrimSpace(fetch.url); url != "" {
			fields["url"] = url
		}

		ReportLogWithFields(
			"hotreload",
			LogInfo,
			DiagnosticInformational,
			"pending fetch will restart on hot reload",
			"",
			fields,
		)
	}
}

func (hooks *Hooks) restoreStateValue(index int) (interface{}, bool) {
	if hooks == nil || hooks.hotReloadRestore == nil {
		return nil, false
	}
	if index < 0 || index >= len(hooks.hotReloadRestore.States) {
		return nil, false
	}
	return normalizeHotReloadValue(hooks.hotReloadRestore.States[index]), true
}

func (hooks *Hooks) restoreMemoValue(index int) (interface{}, []interface{}, bool) {
	if hooks == nil || hooks.hotReloadRestore == nil {
		return nil, nil, false
	}
	if index < 0 || index >= len(hooks.hotReloadRestore.Memos) {
		return nil, nil, false
	}

	snapshot := hooks.hotReloadRestore.Memos[index]
	return normalizeHotReloadValue(snapshot.Value), normalizeHotReloadDeps(snapshot.Deps), true
}

func (hooks *Hooks) restoreRefValue(index int) (interface{}, bool) {
	if hooks == nil || hooks.hotReloadRestore == nil {
		return nil, false
	}
	if index < 0 || index >= len(hooks.hotReloadRestore.Refs) {
		return nil, false
	}
	return normalizeHotReloadValue(hooks.hotReloadRestore.Refs[index]), true
}

func (hooks *Hooks) restoreIDValue(index int) (string, bool) {
	if hooks == nil || hooks.hotReloadRestore == nil {
		return "", false
	}
	if index < 0 || index >= len(hooks.hotReloadRestore.IDs) {
		return "", false
	}
	if id := strings.TrimSpace(hooks.hotReloadRestore.IDs[index]); id != "" {
		return id, true
	}
	return "", false
}

func (hooks *Hooks) restoreFetchValue(index int, url string) (FetchState, bool) {
	if hooks == nil || hooks.hotReloadRestore == nil {
		return FetchState{}, false
	}
	if index < 0 || index >= len(hooks.hotReloadRestore.Fetches) {
		return FetchState{}, false
	}
	snapshot := hooks.hotReloadRestore.Fetches[index]
	if strings.TrimSpace(snapshot.URL) != "" && snapshot.URL != url {
		return FetchState{}, false
	}
	snapshot.State.Data = normalizeHotReloadValue(snapshot.State.Data)
	return snapshot.State, true
}

func (rt *Runtime) renderFunctionComponent(fiber *Fiber) (*Element, bool, *Fiber) {
	var restore *HotReloadComponentSnapshot
	if snapshot := rt.matchingHotReloadComponentSnapshot(fiber); snapshot != nil {
		if componentSnapshotCompatible(snapshot, fiber) {
			restore = snapshot
		} else {
			reportHotReloadFallbackDiagnostic(snapshot, fiber)
		}
	}

	for attempt := 0; attempt < 2; attempt++ {
		currentFiber = fiber
		if attempt == 0 && restore != nil {
			fiber.hooks = &Hooks{owner: fiber, hotReloadRestore: restore}
		} else if fiber.alternate != nil && fiber.alternate.hooks != nil {
			fiber.hooks = fiber.alternate.hooks
			fiber.hooks.owner = fiber

			// Prepare for new render
			fiber.hooks.index = 0
			fiber.hooks.stateIndex = 0
			fiber.hooks.depIndex = 0
			fiber.hooks.memoIndex = 0
			fiber.hooks.callbackIndex = 0
			fiber.hooks.refIndex = 0
			fiber.hooks.idIndex = 0
			fiber.hooks.fetchIndex = 0
			fiber.hooks.funcIndex = 0
			fiber.hooks.atomIndex = 0
			fiber.hooks.cleanupIndex = 0
			fiber.hooks.signature = fiber.hooks.signature[:0]
			fiber.hooks.hotReloadRestore = nil
		} else {
			fiber.hooks = &Hooks{owner: fiber}
		}

		if fiber.effects != nil {
			fiber.effects = fiber.effects[:0]
		} else {
			fiber.effects = make([]Effect, 0)
		}

		var element *Element
		var handledPanic bool
		var nextFromBoundary *Fiber
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					var handled bool
					nextFromBoundary, handled = rt.recoverBoundaryError(fiber.parent, recovered, boundaryPhaseRender)
					if !handled {
						panic(reportUnhandledPanic(fiber, boundaryPhaseRender, recovered))
					}
					handledPanic = true
				}
			}()

			if fn, ok := fiber.typeOf.(func() *Element); ok {
				element = fn()
			} else if fn, ok := fiber.typeOf.(func(map[string]interface{}) *Element); ok {
				element = fn(fiber.props)
			} else if fn, ok := fiber.typeOf.(func(Attrs) *Element); ok {
				element = fn(Attrs(fiber.props))
			} else if component, ok := fiber.typeOf.(*ComponentType); ok {
				element = component.Render(fiber.props)
			}
		}()

		if handledPanic {
			return nil, true, nextFromBoundary
		}

		if attempt == 0 && restore != nil {
			if !componentSnapshotSerializableCompatible(restore, fiber) {
				reportHotReloadFallbackDiagnostic(restore, fiber)
				releaseHookResources(fiber.hooks)
				restore = nil
				fiber.hooks = nil
				continue
			}
		}

		if fiber.hooks != nil {
			fiber.hooks.hotReloadRestore = nil
		}
		return element, false, nil
	}

	return nil, false, nil
}
