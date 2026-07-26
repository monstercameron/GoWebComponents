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
	States        []any                    `json:"states,omitempty"`
	Memos         []HotReloadMemoSnapshot  `json:"memos,omitempty"`
	Refs          []any                    `json:"refs,omitempty"`
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

// CaptureHotReloadSnapshot is an internal hot-reload helper.
func (parseRt *Runtime) CaptureHotReloadSnapshot() HotReloadSnapshot {
	if parseRt == nil || parseRt.currentRoot == nil {
		return HotReloadSnapshot{}
	}

	parseSnapshot := HotReloadSnapshot{}
	captureHotReloadComponentSnapshots(parseRt.currentRoot, &parseSnapshot.Components)
	return parseSnapshot
}

// RestoreHotReloadSnapshot is an internal hot-reload helper.
func (parseRt *Runtime) RestoreHotReloadSnapshot(parseSnapshot HotReloadSnapshot) {
	_ = parseRt.RestoreHotReloadSnapshotWithPlan(parseSnapshot, HotReloadRestorePlan{})
}

// RestoreHotReloadSnapshotWithPlan is an internal hot-reload helper.
func (parseRt *Runtime) RestoreHotReloadSnapshotWithPlan(parseSnapshot HotReloadSnapshot, parsePlan HotReloadRestorePlan) HotReloadRestoreDecision {
	if parseRt == nil {
		return HotReloadRestoreDecision{}
	}

	if len(parseSnapshot.Components) == 0 {
		parseRt.pendingHotReloadComponents = nil
		parseRt.pendingHotReloadIndex = 0
		if parseRt.pendingHotReloadByPath != nil {
			clear(parseRt.pendingHotReloadByPath)
		}
		parseRt.pendingHotReloadSelective = false
		return HotReloadRestoreDecision{}
	}

	parseDecision := HotReloadRestoreDecision{}
	if parsePlan.Selective {
		parseDecision = parseRt.restoreSelectiveHotReloadSnapshot(parseSnapshot, parsePlan)
		if parseDecision.Strategy == "selective" {
			return parseDecision
		}
	}

	// Overwrite in place, then release whatever the previous, longer snapshot
	// left past the new length.
	parseRt.pendingHotReloadComponents = append(parseRt.pendingHotReloadComponents[:0], parseSnapshot.Components...)
	clear(parseRt.pendingHotReloadComponents[len(parseRt.pendingHotReloadComponents):cap(parseRt.pendingHotReloadComponents)])
	parseRt.pendingHotReloadIndex = 0
	if parseRt.pendingHotReloadByPath != nil {
		clear(parseRt.pendingHotReloadByPath)
	}
	parseRt.pendingHotReloadSelective = false
	if parseDecision.Strategy == "" {
		parseDecision.Strategy = "legacy"
	}
	return parseDecision
}

// HasPendingHotReloadSnapshot is an internal hot-reload helper.
func (parseRt *Runtime) HasPendingHotReloadSnapshot() bool {
	if parseRt == nil {
		return false
	}
	if parseRt.pendingHotReloadSelective {
		return len(parseRt.pendingHotReloadByPath) > 0
	}
	return parseRt.pendingHotReloadIndex < len(parseRt.pendingHotReloadComponents)
}

// nextHotReloadComponentSnapshot is an internal hot-reload helper.
func (parseRt *Runtime) nextHotReloadComponentSnapshot() *HotReloadComponentSnapshot {
	if parseRt == nil || parseRt.pendingHotReloadIndex >= len(parseRt.pendingHotReloadComponents) {
		return nil
	}

	parseSnapshot := parseRt.pendingHotReloadComponents[parseRt.pendingHotReloadIndex]
	parseRt.pendingHotReloadIndex++
	return &parseSnapshot
}

// matchingHotReloadComponentSnapshot is an internal hot-reload helper.
func (parseRt *Runtime) matchingHotReloadComponentSnapshot(parseFiber *Fiber) *HotReloadComponentSnapshot {
	if parseRt == nil {
		return nil
	}
	if !parseRt.pendingHotReloadSelective {
		return parseRt.nextHotReloadComponentSnapshot()
	}
	if parseRt.pendingHotReloadByPath == nil {
		return nil
	}
	parsePath := hotReloadFiberPath(parseFiber)
	if strings.TrimSpace(parsePath) == "" {
		return nil
	}
	parseSnapshot, parseOk := parseRt.pendingHotReloadByPath[parsePath]
	if !parseOk {
		return nil
	}
	delete(parseRt.pendingHotReloadByPath, parsePath)
	return &parseSnapshot
}

// restoreSelectiveHotReloadSnapshot is an internal hot-reload helper.
func (parseRt *Runtime) restoreSelectiveHotReloadSnapshot(parseSnapshot HotReloadSnapshot, parsePlan HotReloadRestorePlan) HotReloadRestoreDecision {
	parseChanged := make(map[string]struct{}, len(parsePlan.ChangedIdentities))
	for _, parseIdentity := range parsePlan.ChangedIdentities {
		parseIdentity = strings.TrimSpace(parseIdentity)
		if parseIdentity != "" {
			parseChanged[parseIdentity] = struct{}{}
		}
	}
	if len(parseChanged) == 0 {
		return HotReloadRestoreDecision{Strategy: "legacy", UnsafeReason: "selective restore requested without changed component identities"}
	}
	if parseRt.pendingHotReloadByPath == nil {
		parseRt.pendingHotReloadByPath = make(map[string]HotReloadComponentSnapshot, len(parseSnapshot.Components))
	} else {
		clear(parseRt.pendingHotReloadByPath)
	}

	for _, parseComponent := range parseSnapshot.Components {
		parsePath := strings.TrimSpace(parseComponent.Path)
		if parsePath == "" {
			clear(parseRt.pendingHotReloadByPath)
			return HotReloadRestoreDecision{Strategy: "legacy", UnsafeReason: "saved snapshot is missing stable component paths"}
		}
		if _, parseExists := parseRt.pendingHotReloadByPath[parsePath]; parseExists {
			clear(parseRt.pendingHotReloadByPath)
			return HotReloadRestoreDecision{Strategy: "legacy", UnsafeReason: "saved snapshot contains duplicate component paths"}
		}
		if hotReloadSnapshotTouchesChangedIdentity(parseComponent, parseChanged) {
			continue
		}
		parseRt.pendingHotReloadByPath[parsePath] = parseComponent
	}

	parseRt.pendingHotReloadComponents = nil
	parseRt.pendingHotReloadIndex = 0
	parseRt.pendingHotReloadSelective = true
	return HotReloadRestoreDecision{Strategy: "selective"}
}

// captureHotReloadComponentSnapshots is an internal hot-reload helper.
// It uses an explicit stack to avoid unbounded recursion on deep fiber trees.
func captureHotReloadComponentSnapshots(parseFiber *Fiber, parseSnapshots *[]HotReloadComponentSnapshot) {
	if parseFiber == nil {
		return
	}

	parseStack := make([]*Fiber, 0, 32)
	parseStack = append(parseStack, parseFiber)
	for len(parseStack) > 0 {
		parseCurrent := parseStack[len(parseStack)-1]
		parseStack = parseStack[:len(parseStack)-1]
		if parseCurrent == nil {
			continue
		}

		if parseKind, _ := describeFiber(parseCurrent); parseKind == "component" {
			if parseSnapshot := captureHotReloadComponentSnapshot(parseCurrent); parseSnapshot != nil {
				*parseSnapshots = append(*parseSnapshots, *parseSnapshot)
			}
		}

		// Push siblings then child so children are processed before siblings
		// (preserving the original preorder traversal order).
		for parseSibling := parseCurrent.child; parseSibling != nil; parseSibling = parseSibling.sibling {
			parseStack = append(parseStack, parseSibling)
		}
		// Reverse the siblings we just pushed so they come out left-to-right.
		parseChildStart := len(parseStack) - countFiberSiblings(parseCurrent.child)
		for parseLeft, parseRight := parseChildStart, len(parseStack)-1; parseLeft < parseRight; parseLeft, parseRight = parseLeft+1, parseRight-1 {
			parseStack[parseLeft], parseStack[parseRight] = parseStack[parseRight], parseStack[parseLeft]
		}
	}
}

// countFiberSiblings counts the number of direct children of a fiber.
func countFiberSiblings(parseFirst *Fiber) int {
	parseCount := 0
	for parseSibling := parseFirst; parseSibling != nil; parseSibling = parseSibling.sibling {
		parseCount++
	}
	return parseCount
}

// captureHotReloadComponentSnapshot is an internal hot-reload helper.
func captureHotReloadComponentSnapshot(parseFiber *Fiber) *HotReloadComponentSnapshot {
	if parseFiber == nil {
		return nil
	}

	parseSignature := buildComponentSignature(parseFiber, parseFiber.hooks)
	if parseSignature == nil {
		return nil
	}

	parseSnapshot := &HotReloadComponentSnapshot{
		Signature:     *parseSignature,
		Path:          hotReloadFiberPath(parseFiber),
		IdentityTrail: hotReloadIdentityTrail(parseFiber),
	}
	if parseFiber.hooks != nil {
		if len(parseFiber.hooks.states) > 0 {
			parseSnapshot.States = make([]any, 0, len(parseFiber.hooks.states)/2)
			for parseIndex := 0; parseIndex+1 < len(parseFiber.hooks.states); parseIndex += 2 {
				parseSnapshot.States = append(parseSnapshot.States, normalizeHotReloadValue(parseFiber.hooks.states[parseIndex]))
			}
		}
		if len(parseFiber.hooks.memos) > 0 {
			parseSnapshot.Memos = make([]HotReloadMemoSnapshot, 0, len(parseFiber.hooks.memos))
			for _, parseMemo := range parseFiber.hooks.memos {
				parseSnapshot.Memos = append(parseSnapshot.Memos, HotReloadMemoSnapshot{
					Value: normalizeHotReloadValue(parseMemo.value),
					Deps:  normalizeHotReloadDeps(parseMemo.deps),
				})
			}
		}
		if len(parseFiber.hooks.refs) > 0 {
			parseSnapshot.Refs = make([]any, 0, len(parseFiber.hooks.refs))
			for _, parseRef := range parseFiber.hooks.refs {
				if parseRef == nil {
					parseSnapshot.Refs = append(parseSnapshot.Refs, nil)
					continue
				}
				parseSnapshot.Refs = append(parseSnapshot.Refs, normalizeHotReloadValue(parseRef.Current))
			}
		}
		if len(parseFiber.hooks.ids) > 0 {
			parseSnapshot.IDs = append(parseSnapshot.IDs, parseFiber.hooks.ids...)
		}
		if len(parseFiber.hooks.fetches) > 0 {
			parseSnapshot.Fetches = make([]HotReloadFetchSnapshot, 0, len(parseFiber.hooks.fetches))
			for _, parseFetch := range parseFiber.hooks.fetches {
				parseSnapshot.Fetches = append(parseSnapshot.Fetches, HotReloadFetchSnapshot{
					URL:   parseFetch.url,
					State: normalizeHotReloadFetchState(parseFetch.state),
				})
			}
		}
	}

	return parseSnapshot
}

// hotReloadSnapshotTouchesChangedIdentity is an internal hot-reload helper.
func hotReloadSnapshotTouchesChangedIdentity(parseSnapshot HotReloadComponentSnapshot, parseChanged map[string]struct{}) bool {
	if len(parseChanged) == 0 {
		return false
	}
	for _, parseIdentity := range parseSnapshot.IdentityTrail {
		if _, parseOk := parseChanged[strings.TrimSpace(parseIdentity)]; parseOk {
			return true
		}
	}
	return false
}

// hotReloadIdentityTrail is an internal hot-reload helper.
func hotReloadIdentityTrail(parseFiber *Fiber) []string {
	if parseFiber == nil {
		return nil
	}
	parseTrail := make([]string, 0, 8)
	for parseCurrent := parseFiber; parseCurrent != nil; parseCurrent = parseCurrent.parent {
		parseSignature := buildComponentSignature(parseCurrent, nil)
		if parseSignature == nil {
			continue
		}
		parseIdentity := strings.TrimSpace(parseSignature.identityKey())
		if parseIdentity == "" {
			continue
		}
		parseTrail = append(parseTrail, parseIdentity)
	}
	for parseLeft, parseRight := 0, len(parseTrail)-1; parseLeft < parseRight; parseLeft, parseRight = parseLeft+1, parseRight-1 {
		parseTrail[parseLeft], parseTrail[parseRight] = parseTrail[parseRight], parseTrail[parseLeft]
	}
	return parseTrail
}

// hotReloadFiberPath is an internal hot-reload helper.
func hotReloadFiberPath(parseFiber *Fiber) string {
	if parseFiber == nil {
		return ""
	}
	parseSegments := make([]string, 0, 8)
	for parseCurrent := parseFiber; parseCurrent != nil && parseCurrent.parent != nil; parseCurrent = parseCurrent.parent {
		parseSegment := hotReloadFiberPathSegment(parseCurrent)
		if parseSegment == "" {
			continue
		}
		parseSegments = append(parseSegments, parseSegment)
	}
	for parseLeft, parseRight := 0, len(parseSegments)-1; parseLeft < parseRight; parseLeft, parseRight = parseLeft+1, parseRight-1 {
		parseSegments[parseLeft], parseSegments[parseRight] = parseSegments[parseRight], parseSegments[parseLeft]
	}
	return strings.Join(parseSegments, "/")
}

// hotReloadFiberPathSegment is an internal hot-reload helper.
func hotReloadFiberPathSegment(parseFiber *Fiber) string {
	if parseFiber == nil || parseFiber.parent == nil {
		return ""
	}
	if parseKey := hotReloadFiberKeySegment(parseFiber); parseKey != "" {
		return parseKey
	}

	parseIndex := 0
	for parseSibling := parseFiber.parent.child; parseSibling != nil && parseSibling != parseFiber; parseSibling = parseSibling.sibling {
		parseIndex++
	}
	if parseSignature := buildComponentSignature(parseFiber, nil); parseSignature != nil {
		parseIdentity := strings.TrimSpace(parseSignature.identityKey())
		if parseIdentity != "" {
			return fmt.Sprintf("%s@%d", parseIdentity, parseIndex)
		}
	}
	parseKind, parseName := describeFiber(parseFiber)
	if strings.TrimSpace(parseName) == "" {
		parseName = parseKind
	}
	return fmt.Sprintf("%s@%d", parseName, parseIndex)
}

// hotReloadFiberKeySegment is an internal hot-reload helper.
func hotReloadFiberKeySegment(parseFiber *Fiber) string {
	if parseFiber == nil {
		return ""
	}
	if parseFiber.key != "" {
		return "key:" + parseFiber.key
	}
	if parseFiber.props == nil {
		return ""
	}
	parseKey, parseOk := parseFiber.props["key"]
	if !parseOk || parseKey == nil {
		return ""
	}
	return "key:" + fmt.Sprint(parseKey)
}

// normalizeHotReloadFetchState is an internal hot-reload helper.
func normalizeHotReloadFetchState(parseState FetchState) FetchState {
	parseState.Data = normalizeHotReloadValue(parseState.Data)
	if strings.TrimSpace(parseState.Error) == "" {
		parseState.Error = ""
	}
	return parseState
}

// normalizeHotReloadValue is an internal hot-reload helper.
func normalizeHotReloadValue(parseValue any) any {
	switch parseTyped := parseValue.(type) {
	case map[string]any:
		parseNormalized := make(map[string]any, len(parseTyped))
		for parseKey, parseNested := range parseTyped {
			parseNormalized[parseKey] = normalizeHotReloadValue(parseNested)
		}
		return parseNormalized
	case []any:
		parseNormalized2 := make([]any, len(parseTyped))
		for parseIndex, parseNested2 := range parseTyped {
			parseNormalized2[parseIndex] = normalizeHotReloadValue(parseNested2)
		}
		return parseNormalized2
	case float64:
		if math.Trunc(parseTyped) == parseTyped {
			return int(parseTyped)
		}
		return parseTyped
	default:
		return parseValue
	}
}

// normalizeHotReloadDeps is an internal hot-reload helper.
func normalizeHotReloadDeps(parseDeps []any) []any {
	if len(parseDeps) == 0 {
		return nil
	}

	parseNormalized := make([]any, len(parseDeps))
	for parseIndex, parseDep := range parseDeps {
		parseNormalized[parseIndex] = normalizeHotReloadValue(parseDep)
	}
	return parseNormalized
}

// coerceHotReloadValue is an internal hot-reload helper.
func coerceHotReloadValue(parseValue any, parseTargetType reflect.Type) (any, bool) {
	if parseTargetType == nil {
		return parseValue, true
	}

	if parseValue == nil {
		return reflect.Zero(parseTargetType).Interface(), true
	}

	parseValueType := reflect.TypeOf(parseValue)
	if parseValueType.AssignableTo(parseTargetType) {
		return parseValue, true
	}
	if parseValueType.ConvertibleTo(parseTargetType) {
		return reflect.ValueOf(parseValue).Convert(parseTargetType).Interface(), true
	}
	if parseTargetType.Kind() == reflect.Interface && parseTargetType.NumMethod() == 0 {
		return parseValue, true
	}

	parseData, parseErr := json.Marshal(parseValue)
	if parseErr != nil {
		return nil, false
	}

	var parseTarget reflect.Value
	if parseTargetType.Kind() == reflect.Pointer {
		parseTarget = reflect.New(parseTargetType.Elem())
	} else {
		parseTarget = reflect.New(parseTargetType)
	}

	if parseErr2 := json.Unmarshal(parseData, parseTarget.Interface()); parseErr2 != nil {
		return nil, false
	}

	if parseTargetType.Kind() == reflect.Pointer {
		return parseTarget.Interface(), true
	}

	return parseTarget.Elem().Interface(), true
}

// NormalizeHotReloadValue exposes the hot-reload value normalizer for bridge
// code that receives JSON-decoded snapshots.
func NormalizeHotReloadValue(parseValue any) any {
	return normalizeHotReloadValue(parseValue)
}

// componentSnapshotCompatible is an internal hot-reload helper.
func componentSnapshotCompatible(parseSnapshot *HotReloadComponentSnapshot, parseFiber *Fiber) bool {
	if parseSnapshot == nil || parseFiber == nil {
		return false
	}

	parseCurrent := buildComponentSignature(parseFiber, nil)
	if parseCurrent == nil {
		return false
	}

	if parseSnapshot.Signature.Kind != parseCurrent.Kind {
		return false
	}
	if parseSnapshot.Signature.identityKey() != parseCurrent.identityKey() {
		return false
	}
	if parseSnapshot.Signature.Key != parseCurrent.Key {
		return false
	}
	return true
}

// componentSnapshotFullyCompatible is an internal hot-reload helper.
func componentSnapshotFullyCompatible(parseSnapshot *HotReloadComponentSnapshot, parseFiber *Fiber) bool {
	if !componentSnapshotCompatible(parseSnapshot, parseFiber) {
		return false
	}

	parseCurrent := buildComponentSignature(parseFiber, parseFiber.hooks)
	if parseCurrent == nil {
		return false
	}

	return parseSnapshot.Signature.CompatibleWith(*parseCurrent)
}

// componentSnapshotSerializableCompatible is an internal hot-reload helper.
func componentSnapshotSerializableCompatible(parseSnapshot *HotReloadComponentSnapshot, parseFiber *Fiber) bool {
	if !componentSnapshotCompatible(parseSnapshot, parseFiber) {
		return false
	}

	parseCurrent := buildComponentSignature(parseFiber, parseFiber.hooks)
	if parseCurrent == nil {
		return false
	}

	parsePreviousSerializable := filterHotReloadSerializableKinds(parseSnapshot.Signature.HookKinds)
	parseCurrentSerializable := filterHotReloadSerializableKinds(parseCurrent.HookKinds)
	return hotReloadHookKindPrefixCompatible(parsePreviousSerializable, parseCurrentSerializable)
}

// filterHotReloadSerializableKinds is an internal hot-reload helper.
func filterHotReloadSerializableKinds(parseKinds []string) []string {
	if len(parseKinds) == 0 {
		return nil
	}
	parseFiltered := make([]string, 0, len(parseKinds))
	for _, parseKind := range parseKinds {
		if _, parseOk := hotReloadSerializableKinds[parseKind]; parseOk {
			parseFiltered = append(parseFiltered, parseKind)
		}
	}
	return parseFiltered
}

// hotReloadHookKindPrefixCompatible is an internal hot-reload helper.
// It returns true only when the snapshot (parsePrevious) is a prefix of the
// current hook list (parseCurrent), i.e. the current component has at least as
// many serializable hooks as the snapshot and the leading kinds all match.
// This ensures that if hooks were removed the restore is rejected, preventing
// stale snapshot slots from being mapped into wrong hook positions.
func hotReloadHookKindPrefixCompatible(parsePrevious, parseCurrent []string) bool {
	if len(parsePrevious) > len(parseCurrent) {
		return false
	}
	for parseIndex, parseKind := range parsePrevious {
		if parseCurrent[parseIndex] != parseKind {
			return false
		}
	}
	return true
}

// reportHotReloadFallbackDiagnostic is an internal hot-reload helper.
func reportHotReloadFallbackDiagnostic(parseSnapshot *HotReloadComponentSnapshot, parseFiber *Fiber) {
	if parseSnapshot == nil || parseFiber == nil {
		return
	}

	parseLabel := hotReloadComponentLabel(parseSnapshot, parseFiber)
	ReportDiagnosticWithContext(
		"runtime",
		DiagnosticWarning,
		fmt.Sprintf("hot reload fell back to remount for %s: %s", parseLabel, hotReloadFallbackReason(parseSnapshot, parseFiber)),
		diagnosticPathForFiber(parseFiber),
		diagnosticComponentStack(parseFiber),
	)
}

// hotReloadComponentLabel is an internal hot-reload helper.
func hotReloadComponentLabel(parseSnapshot *HotReloadComponentSnapshot, parseFiber *Fiber) string {
	if parseCurrent := buildComponentSignature(parseFiber, nil); parseCurrent != nil {
		if parseName := strings.TrimSpace(parseCurrent.Name); parseName != "" {
			return parseName
		}
	}
	if parseSnapshot != nil {
		if parseName2 := strings.TrimSpace(parseSnapshot.Signature.Name); parseName2 != "" {
			return parseName2
		}
		if parseQualified := strings.TrimSpace(parseSnapshot.Signature.QualifiedName); parseQualified != "" {
			return parseQualified
		}
	}
	return "component"
}

// hotReloadFallbackReason is an internal hot-reload helper.
func hotReloadFallbackReason(parseSnapshot *HotReloadComponentSnapshot, parseFiber *Fiber) string {
	if parseSnapshot == nil || parseFiber == nil {
		return "saved state could not be matched"
	}

	parseCurrent := buildComponentSignature(parseFiber, nil)
	if parseCurrent == nil {
		return "current component identity could not be inspected"
	}

	if parseSnapshot.Signature.Kind != parseCurrent.Kind {
		return fmt.Sprintf("component kind changed from %q to %q", parseSnapshot.Signature.Kind, parseCurrent.Kind)
	}
	if parseSnapshot.Signature.identityKey() != parseCurrent.identityKey() {
		return fmt.Sprintf("component identity changed from %q to %q", parseSnapshot.Signature.identityKey(), parseCurrent.identityKey())
	}
	if parseSnapshot.Signature.Key != parseCurrent.Key {
		return fmt.Sprintf("component key changed from %q to %q", parseSnapshot.Signature.Key, parseCurrent.Key)
	}

	parseCurrent = buildComponentSignature(parseFiber, parseFiber.hooks)
	if parseCurrent == nil {
		return "current hook signature could not be inspected"
	}
	if len(parseSnapshot.Signature.HookKinds) != len(parseCurrent.HookKinds) {
		return fmt.Sprintf(
			"hook count changed from %d to %d (%s -> %s)",
			len(parseSnapshot.Signature.HookKinds),
			len(parseCurrent.HookKinds),
			hotReloadHookKindsSummary(parseSnapshot.Signature.HookKinds),
			hotReloadHookKindsSummary(parseCurrent.HookKinds),
		)
	}
	for parseIndex, parseKind := range parseSnapshot.Signature.HookKinds {
		if parseCurrent.HookKinds[parseIndex] != parseKind {
			return fmt.Sprintf(
				"hook order changed from %s to %s",
				hotReloadHookKindsSummary(parseSnapshot.Signature.HookKinds),
				hotReloadHookKindsSummary(parseCurrent.HookKinds),
			)
		}
	}

	return fmt.Sprintf("component signature changed from %s to %s", parseSnapshot.Signature.Summary(), parseCurrent.Summary())
}

// hotReloadHookKindsSummary is an internal hot-reload helper.
func hotReloadHookKindsSummary(parseKinds []string) string {
	if len(parseKinds) == 0 {
		return "no hooks"
	}
	return strings.Join(parseKinds, " > ")
}

// releaseHookResources is an internal hot-reload helper.
func releaseHookResources(parseHooks *Hooks) {
	if parseHooks == nil {
		return
	}

	for parseIndex := range parseHooks.funcs {
		parseHandler := parseHooks.funcs[parseIndex]
		if parseReleasable, parseOk := parseHandler.wrapper.(interface{ Release() }); parseOk {
			parseReleasable.Release()
		}
		parseHooks.funcs[parseIndex] = funcHandlerValue{}
	}
}

// PrepareForHotReload is an internal hot-reload helper.
func (parseRt *Runtime) PrepareForHotReload() {
	if parseRt == nil || parseRt.currentRoot == nil {
		return
	}

	parseRt.runCleanups(parseRt.currentRoot)
	prepareFiberForHotReload(parseRt, parseRt.currentRoot)
	parseRt.wipRoot = nil
	parseRt.nextUnitOfWork = nil
	if parseRt.deletions != nil {
		clear(parseRt.deletions)
		parseRt.deletions = parseRt.deletions[:0]
	}
	parseRt.updateScheduled = false
	parseRt.pendingBoundaryRecovery = false
}

// prepareFiberForHotReload is an internal hot-reload helper.
func prepareFiberForHotReload(parseRt *Runtime, parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}

	if parseFiber.hooks != nil {
		reportHotReloadRestartActivity(parseFiber)
		releaseHookResources(parseFiber.hooks)
		for parseIndex := range parseFiber.hooks.cleanups {
			parseFiber.hooks.cleanups[parseIndex] = nil
		}
		parseFiber.hooks.hotReloadRestore = nil
	}
	if parseRt != nil {
		parseRt.CleanupAtomSubscriptions(parseFiber)
	}
	parseFiber.effects = nil
	parseFiber.eventCallbacks = nil

	prepareFiberForHotReload(parseRt, parseFiber.child)
	prepareFiberForHotReload(parseRt, parseFiber.sibling)
}

// reportHotReloadRestartActivity is an internal hot-reload helper.
func reportHotReloadRestartActivity(parseFiber *Fiber) {
	if parseFiber == nil || parseFiber.hooks == nil {
		return
	}

	for _, parseFetch := range parseFiber.hooks.fetches {
		if !parseFetch.state.Loading {
			continue
		}

		parseFields := map[string]string{}
		if parsePath := strings.TrimSpace(diagnosticPathForFiber(parseFiber)); parsePath != "" {
			parseFields["path"] = parsePath
		}
		if parseUrl := strings.TrimSpace(parseFetch.url); parseUrl != "" {
			parseFields["url"] = parseUrl
		}

		ReportLogWithFields(
			"hotreload",
			LogInfo,
			DiagnosticInformational,
			"pending fetch will restart on hot reload",
			"",
			parseFields,
		)
	}
}

// restoreStateValue is an internal hot-reload helper.
func (parseHooks *Hooks) restoreStateValue(parseIndex int) (any, bool) {
	if parseHooks == nil || parseHooks.hotReloadRestore == nil {
		return nil, false
	}
	if parseIndex < 0 || parseIndex >= len(parseHooks.hotReloadRestore.States) {
		return nil, false
	}
	return normalizeHotReloadValue(parseHooks.hotReloadRestore.States[parseIndex]), true
}

// restoreMemoValue is an internal hot-reload helper.
func (parseHooks *Hooks) restoreMemoValue(parseIndex int) (any, []any, bool) {
	if parseHooks == nil || parseHooks.hotReloadRestore == nil {
		return nil, nil, false
	}
	if parseIndex < 0 || parseIndex >= len(parseHooks.hotReloadRestore.Memos) {
		return nil, nil, false
	}

	parseSnapshot := parseHooks.hotReloadRestore.Memos[parseIndex]
	return normalizeHotReloadValue(parseSnapshot.Value), normalizeHotReloadDeps(parseSnapshot.Deps), true
}

// restoreRefValue is an internal hot-reload helper.
func (parseHooks *Hooks) restoreRefValue(parseIndex int) (any, bool) {
	if parseHooks == nil || parseHooks.hotReloadRestore == nil {
		return nil, false
	}
	if parseIndex < 0 || parseIndex >= len(parseHooks.hotReloadRestore.Refs) {
		return nil, false
	}
	return normalizeHotReloadValue(parseHooks.hotReloadRestore.Refs[parseIndex]), true
}

// restoreIDValue is an internal hot-reload helper.
func (parseHooks *Hooks) restoreIDValue(parseIndex int) (string, bool) {
	if parseHooks == nil || parseHooks.hotReloadRestore == nil {
		return "", false
	}
	if parseIndex < 0 || parseIndex >= len(parseHooks.hotReloadRestore.IDs) {
		return "", false
	}
	if parseId := strings.TrimSpace(parseHooks.hotReloadRestore.IDs[parseIndex]); parseId != "" {
		return parseId, true
	}
	return "", false
}

// restoreFetchValue is an internal hot-reload helper.
func (parseHooks *Hooks) restoreFetchValue(parseIndex int, parseUrl string) (FetchState, bool) {
	if parseHooks == nil || parseHooks.hotReloadRestore == nil {
		return FetchState{}, false
	}
	if parseIndex < 0 || parseIndex >= len(parseHooks.hotReloadRestore.Fetches) {
		return FetchState{}, false
	}
	parseSnapshot := parseHooks.hotReloadRestore.Fetches[parseIndex]
	if strings.TrimSpace(parseSnapshot.URL) != "" && parseSnapshot.URL != parseUrl {
		return FetchState{}, false
	}
	parseSnapshot.State.Data = normalizeHotReloadValue(parseSnapshot.State.Data)
	return parseSnapshot.State, true
}

// renderFunctionComponent is an internal hot-reload helper.
func (parseRt *Runtime) renderFunctionComponent(parseFiber *Fiber) (*Element, bool, *Fiber) {
	if parseRt != nil && parseFiber != nil {
		parseFiber.ownerRuntime = parseRt
	}
	var parseRestore *HotReloadComponentSnapshot
	if parseSnapshot := parseRt.matchingHotReloadComponentSnapshot(parseFiber); parseSnapshot != nil {
		if componentSnapshotCompatible(parseSnapshot, parseFiber) {
			parseRestore = parseSnapshot
		} else {
			reportHotReloadFallbackDiagnostic(parseSnapshot, parseFiber)
		}
	}

	for parseAttempt := range 2 {
		if parseAttempt == 0 && parseRestore == nil {
			parseRt.strictPreviewRender(parseFiber)
		}
		setCurrentFiberOwned(parseFiber, parseRt.renderPassOwnerID())
		parseFiber.renderDurationNs = 0
		if parseAttempt == 0 && parseRestore != nil {
			parseFiber.hooks = &Hooks{owner: parseFiber, hotReloadRestore: parseRestore}
		} else if parseFiber.alternate != nil && parseFiber.alternate.hooks != nil {
			parseFiber.hooks = parseFiber.alternate.hooks
			parseFiber.hooks.owner = parseFiber

			// Prepare for new render
			parseFiber.hooks.index = 0
			parseFiber.hooks.stateIndex = 0
			parseFiber.hooks.depIndex = 0
			parseFiber.hooks.memoIndex = 0
			parseFiber.hooks.callbackIndex = 0
			parseFiber.hooks.refIndex = 0
			parseFiber.hooks.idIndex = 0
			parseFiber.hooks.fetchIndex = 0
			parseFiber.hooks.funcIndex = 0
			parseFiber.hooks.atomIndex = 0
			parseFiber.hooks.cleanupIndex = 0
			parseFiber.hooks.signature = parseFiber.hooks.signature[:0]
			parseFiber.hooks.hotReloadRestore = nil
		} else {
			parseFiber.hooks = &Hooks{owner: parseFiber}
		}

		if parseFiber.effects != nil {
			// clear() before the reslice: an Effect holds a closure over the
			// pre-reload component, which is exactly what a reload is replacing.
			clear(parseFiber.effects)
			parseFiber.effects = parseFiber.effects[:0]
		} else {
			parseFiber.effects = make([]Effect, 0)
		}

		var parseElement *Element
		var isHandledPanic bool
		var parseNextFromBoundary *Fiber
		renderStart := commitTimingStart()
		parseRt.activeRenderFiber = parseFiber
		func() {
			defer SetCurrentFiber(nil)
			defer func() { parseRt.activeRenderFiber = nil }()
			defer func() {
				if parseRecovered := recover(); parseRecovered != nil {
					var isHandled bool
					if parseSuspension, parseOk := AsSuspension(parseRecovered); parseOk {
						parseNextFromBoundary, isHandled = parseRt.recoverAsyncBoundarySuspension(parseFiber.parent, parseSuspension)
						if !isHandled {
							panic(markUnhandledPanic(parseFiber, boundaryPhaseRender, parseRecovered))
						}
						isHandledPanic = true
						return
					}
					if panicPhaseMayRecoverWithBoundary(PanicPhaseRender) {
						parseNextFromBoundary, isHandled = parseRt.recoverBoundaryError(parseFiber.parent, parseRecovered, boundaryPhaseRender)
					}
					if !isHandled {
						panic(markUnhandledPanic(parseFiber, boundaryPhaseRender, parseRecovered))
					}
					isHandledPanic = true
				}
			}()

			if parseFn, parseOk := parseFiber.typeOf.(func() *Element); parseOk {
				parseElement = parseFn()
			} else if parseFn2, parseOk2 := parseFiber.typeOf.(func(map[string]any) *Element); parseOk2 {
				parseElement = parseFn2(parseFiber.props)
			} else if parseFn3, parseOk3 := parseFiber.typeOf.(func(Attrs) *Element); parseOk3 {
				parseElement = parseFn3(Attrs(parseFiber.props))
			} else if parseComponent, parseOk4 := parseFiber.typeOf.(*ComponentType); parseOk4 {
				parseElement = parseComponent.Render(parseFiber.props)
			}
		}()
		renderDurationNs := commitTimingSinceNs(renderStart)
		parseFiber.renderDurationNs = renderDurationNs

		if isHandledPanic {
			return nil, true, parseNextFromBoundary
		}

		// Render-phase convergence: if the component updated its own state during
		// this render (parseFiber.renderPhaseUpdate, set by the hook setter), re-run
		// it with the new state until it stops updating — bounded like React's
		// re-render limit — so the committed output reflects the final state instead
		// of a half-rendered intermediate. The hook indices are reset per iteration
		// but the deps/cleanups/effects arrays persist; effects accumulate across
		// iterations (the deps-equality check makes a stable effect register only
		// once, while a deps-changing effect registers per iteration), so they are
		// de-duplicated by cleanup index afterwards to commit exactly the final
		// render's effect per slot. NOTE: do not clear parseFiber.effects here —
		// clearing it loses an effect whose deps did not change across iterations.
		parseConverged := false
		for parseRPIters := 0; parseFiber.renderPhaseUpdate; parseRPIters++ {
			parseConverged = true
			parseFiber.renderPhaseUpdate = false
			if parseRPIters >= 25 {
				ReportDiagnosticWithContext("runtime", DiagnosticError,
					"render-phase state updates did not converge after 25 attempts; likely an unconditional setState during render",
					diagnosticPathForFiber(parseFiber), diagnosticComponentStack(parseFiber))
				break
			}
			if parseFiber.hooks != nil {
				parseFiber.hooks.index = 0
				parseFiber.hooks.stateIndex = 0
				parseFiber.hooks.depIndex = 0
				parseFiber.hooks.memoIndex = 0
				parseFiber.hooks.callbackIndex = 0
				parseFiber.hooks.refIndex = 0
				parseFiber.hooks.idIndex = 0
				parseFiber.hooks.fetchIndex = 0
				parseFiber.hooks.funcIndex = 0
				parseFiber.hooks.atomIndex = 0
				parseFiber.hooks.cleanupIndex = 0
				parseFiber.hooks.signature = parseFiber.hooks.signature[:0]
			}
			setCurrentFiberOwned(parseFiber, parseRt.renderPassOwnerID())
			parseRt.activeRenderFiber = parseFiber
			func() {
				defer SetCurrentFiber(nil)
				defer func() { parseRt.activeRenderFiber = nil }()
				switch parseFn := parseFiber.typeOf.(type) {
				case func() *Element:
					parseElement = parseFn()
				case func(map[string]any) *Element:
					parseElement = parseFn(parseFiber.props)
				case func(Attrs) *Element:
					parseElement = parseFn(Attrs(parseFiber.props))
				case *ComponentType:
					parseElement = parseFn.Render(parseFiber.props)
				}
			}()
		}
		if parseConverged {
			parseFiber.effects = dedupRenderPhaseEffects(parseFiber.effects)
		}

		if parseAttempt == 0 && parseRestore != nil {
			if !componentSnapshotSerializableCompatible(parseRestore, parseFiber) {
				reportHotReloadFallbackDiagnostic(parseRestore, parseFiber)
				releaseHookResources(parseFiber.hooks)
				parseRestore = nil
				parseFiber.hooks = nil
				continue
			}
		}

		if parseFiber.hooks != nil {
			parseFiber.hooks.hotReloadRestore = nil
		}
		parseRt.recordComponentRenderTrace(parseFiber, renderDurationNs)
		parseRt.queuePendingEffectFiber(parseFiber)
		return parseElement, false, nil
	}

	return nil, false, nil
}

// dedupRenderPhaseEffects collapses the effects accumulated across render-phase
// convergence iterations to exactly one per cleanup-index slot — the last one,
// i.e. the effect produced by the final converged render — preserving
// declaration (cleanup-index) order. Without this, an effect whose deps change as
// state converges would be registered (and run) once per iteration.
func dedupRenderPhaseEffects(parseEffects []Effect) []Effect {
	if len(parseEffects) <= 1 {
		return parseEffects
	}
	parseLastBySlot := make(map[int]Effect, len(parseEffects))
	parseMaxSlot := -1
	for _, parseEffect := range parseEffects {
		parseLastBySlot[parseEffect.CleanupIndex] = parseEffect
		if parseEffect.CleanupIndex > parseMaxSlot {
			parseMaxSlot = parseEffect.CleanupIndex
		}
	}
	if len(parseLastBySlot) == len(parseEffects) {
		// No duplicates (every slot appeared once); keep the original order.
		return parseEffects
	}
	parseOut := parseEffects[:0]
	for parseSlot := 0; parseSlot <= parseMaxSlot; parseSlot++ {
		if parseEffect, parseOk := parseLastBySlot[parseSlot]; parseOk {
			parseOut = append(parseOut, parseEffect)
		}
	}
	// Filtering in place leaves the superseded duplicates — and their closures —
	// in the slots past the new length.
	clear(parseEffects[len(parseOut):])
	return parseOut
}
