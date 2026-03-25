//go:build js && wasm
// +build js,wasm

package render

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

type config struct {
	synchronous bool
}

// Option configures the render fixture.
type Option func(*config)

// Event describes one synthetic event payload for fixture dispatch helpers.
type Event struct {
	Value   string
	Checked bool
	Key     string
	KeyCode int
}

// OverlaySurface describes one rendered overlay surface snapshot.
type OverlaySurface struct {
	SurfaceID           string
	Kind                string
	Depth               int
	HandlesEscape       bool
	HandlesOutsideClick bool
	TrapFocusOwner      bool
	IsModal             bool
	PortalTargetID      string
}

// DiagnosticSignal describes one runtime diagnostic entry exposed by the fixture.
type DiagnosticSignal struct {
	Source         string
	Severity       string
	Classification string
	Code           string
	Message        string
	Count          int
	Path           string
	Recoverable    bool
	TopFrame       string
	Consequence    string
	ComponentStack []string
	Fields         map[string]string
}

// LogSignal describes one buffered runtime log entry exposed by the fixture.
type LogSignal struct {
	Domain         string
	Level          string
	Classification string
	Code           string
	Message        string
	Timestamp      string
	CorrelationID  string
	Recoverable    bool
	TopFrame       string
	Consequence    string
	Fields         map[string]string
}

// RenderCountSignal describes one component render-count profile entry.
type RenderCountSignal struct {
	Name                    string
	Path                    string
	RenderCount             int
	RerenderCount           int
	LastTrigger             string
	TotalRenderDurationNs   int64
	AverageRenderDurationNs int64
}

var fixtureGate = make(chan struct{}, 1)

const parallelSafetyContract = "testkit/render fixtures are process-global on js/wasm; keep fixture-owning tests sequential and avoid t.Parallel while a fixture is active"

// ParallelSafetyContract returns the explicit js/wasm fixture parallel-safety contract.
func ParallelSafetyContract() string {
	return parallelSafetyContract
}

// WithQueuedScheduler configures the harness to queue work until Flush is called.
func WithQueuedScheduler() Option {
	return func(parseCfg *config) {
		parseCfg.synchronous = false
	}
}

// Fixture owns one mock DOM container, scheduler, and configured global runtime.
//
// The current harness serializes fixture ownership because the runtime hook
// surface is global on js/wasm builds.
type Fixture struct {
	tb        testing.TB
	adapter   *mockdom.MockDOMAdapter
	scheduler *mockdom.MockScheduler
	container *mockdom.MockDOMNode
	cleaned   bool
	unlock    sync.Once
}

// QueryNode wraps one matched rendered node.
type QueryNode struct {
	fixture *Fixture
	node    *mockdom.MockDOMNode
}

// New creates a controlled render fixture for js/wasm tests.
func New(parseTb testing.TB, parseOptions ...Option) *Fixture {
	parseTb.Helper()
	parseCfg := config{synchronous: true}
	for _, parseOption := range parseOptions {
		if parseOption != nil {
			parseOption(&parseCfg)
		}
	}

	applyFixtureOwnership(parseTb)
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseScheduler := mockdom.NewMockScheduler(parseCfg.synchronous)
	parseContainer, _ := parseAdapter.CreateElement("div").(*mockdom.MockDOMNode)
	runtime.InitGlobalRuntime(runtime.Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
		Reset:      true,
	})
	runtime.ClearDiagnostics()
	runtime.ClearLogs()

	parseFixture := &Fixture{
		tb:        parseTb,
		adapter:   parseAdapter,
		scheduler: parseScheduler,
		container: parseContainer,
	}
	parseTb.Cleanup(func() {
		parseFixture.Cleanup()
	})
	return parseFixture
}

// Render mounts a UI tree into the fixture container.
func (parseF *Fixture) Render(parseRoot ui.Node) {
	parseF.tb.Helper()
	parseF.requireActive()
	if parseErr := runtime.GetGlobalRuntime().RenderInto(parseF.container, parseRoot); parseErr != nil {
		parseF.tb.Fatalf("render fixture failed to mount root: %v", parseErr)
	}
	parseF.Flush()
}

// Rerender replaces the current tree with a new root.
func (parseF *Fixture) Rerender(parseRoot ui.Node) {
	parseF.Render(parseRoot)
}

// Flush drains queued scheduler work until the fixture settles.
func (parseF *Fixture) Flush() {
	parseF.tb.Helper()
	parseF.requireActive()
	parseF.scheduler.FlushAll()
}

// FlushTimers drains queued timeout work and any follow-up render work.
func (parseF *Fixture) FlushTimers() {
	parseF.tb.Helper()
	parseF.requireActive()
	parseF.scheduler.FlushTimeouts()
	parseF.scheduler.FlushAll()
}

// Stabilize drains pending scheduled work until the fixture settles.
func (parseF *Fixture) Stabilize() {
	parseF.Flush()
}

// Cleanup releases fixture ownership and clears buffered diagnostics.
func (parseF *Fixture) Cleanup() {
	if parseF == nil || parseF.cleaned {
		return
	}
	parseF.cleaned = true
	runtime.ClearDiagnostics()
	runtime.ClearLogs()
	parseF.container = nil
	parseF.adapter = nil
	parseF.scheduler = nil
	parseF.unlock.Do(func() {
		clearFixtureOwnership()
	})
}

// Container returns the fixture root container.
func (parseF *Fixture) Container() *QueryNode {
	if parseF == nil || parseF.container == nil {
		return nil
	}
	return &QueryNode{fixture: parseF, node: parseF.container}
}

// Target returns the explicit DOM target owned by the fixture.
func (parseF *Fixture) Target() any {
	if parseF == nil || parseF.container == nil {
		return nil
	}
	return parseF.container
}

// ByRole returns the first node whose computed role matches, optionally filtered by accessible name.
func (parseF *Fixture) ByRole(parseRole string, parseName string) *QueryNode {
	if parseF == nil || parseF.container == nil {
		return nil
	}
	parseWantRole := normalizeText(parseRole)
	parseWantName := normalizeText(parseName)
	return parseF.wrap(findNode(parseF.container, func(parseNode *mockdom.MockDOMNode) bool {
		if parseNode == nil || normalizeText(nodeRole(parseNode)) != parseWantRole {
			return false
		}
		if parseWantName == "" {
			return true
		}
		return normalizeText(accessibleName(parseF.container, parseNode)) == parseWantName
	}))
}

// AllByRole returns all nodes whose computed role matches.
func (parseF *Fixture) AllByRole(parseRole string) []*QueryNode {
	if parseF == nil || parseF.container == nil {
		return nil
	}
	parseWantRole := normalizeText(parseRole)
	parseMatches := collectNodes(parseF.container, func(parseNode *mockdom.MockDOMNode) bool {
		return normalizeText(nodeRole(parseNode)) == parseWantRole
	})
	parseResult := make([]*QueryNode, 0, len(parseMatches))
	for _, parseMatch := range parseMatches {
		parseResult = append(parseResult, parseF.wrap(parseMatch))
	}
	return parseResult
}

// ByLabel returns the first interactive node whose accessible label matches.
func (parseF *Fixture) ByLabel(parseLabel string) *QueryNode {
	if parseF == nil || parseF.container == nil {
		return nil
	}
	parseLabel := normalizeText(parseLabel)
	return parseF.wrap(findNode(parseF.container, func(parseNode *mockdom.MockDOMNode) bool {
		if parseNode == nil || nodeRole(parseNode) == "" {
			return false
		}
		return normalizeText(accessibleName(parseF.container, parseNode)) == parseLabel
	}))
}

// ByDescription returns the first interactive node whose accessible description matches.
func (parseF *Fixture) ByDescription(parseDescription string) *QueryNode {
	if parseF == nil || parseF.container == nil {
		return nil
	}
	parseDescription := normalizeText(parseDescription)
	return parseF.wrap(findNode(parseF.container, func(parseNode *mockdom.MockDOMNode) bool {
		if parseNode == nil || nodeRole(parseNode) == "" {
			return false
		}
		return normalizeText(accessibleDescription(parseF.container, parseNode)) == parseDescription
	}))
}

// ByLiveRegion returns the first live-region node matching politeness and optional text.
func (parseF *Fixture) ByLiveRegion(parsePoliteness string, parseText string) *QueryNode {
	if parseF == nil || parseF.container == nil {
		return nil
	}
	parsePoliteness := normalizeText(parsePoliteness)
	parseText := normalizeText(parseText)
	return parseF.wrap(findNode(parseF.container, func(parseNode *mockdom.MockDOMNode) bool {
		parseLive := normalizeText(nodeLivePoliteness(parseNode))
		if parseLive == "" {
			return false
		}
		if parsePoliteness != "" && parseLive != parsePoliteness {
			return false
		}
		if parseText == "" {
			return true
		}
		return normalizeText(nodeText(parseNode)) == parseText
	}))
}

// ApplyByRole asserts one role query match and returns the node.
func (parseF *Fixture) ApplyByRole(parseRole string, parseName string) *QueryNode {
	parseF.tb.Helper()
	parseMatch := parseF.ByRole(parseRole, parseName)
	if parseMatch == nil {
		parseF.tb.Fatalf("render fixture could not find role=%q name=%q", parseRole, parseName)
	}
	return parseMatch
}

// ApplyByLabel asserts one label query match and returns the node.
func (parseF *Fixture) ApplyByLabel(parseLabel string) *QueryNode {
	parseF.tb.Helper()
	parseMatch := parseF.ByLabel(parseLabel)
	if parseMatch == nil {
		parseF.tb.Fatalf("render fixture could not find label=%q", parseLabel)
	}
	return parseMatch
}

// ApplyByDescription asserts one description query match and returns the node.
func (parseF *Fixture) ApplyByDescription(parseDescription string) *QueryNode {
	parseF.tb.Helper()
	parseMatch := parseF.ByDescription(parseDescription)
	if parseMatch == nil {
		parseF.tb.Fatalf("render fixture could not find description=%q", parseDescription)
	}
	return parseMatch
}

// ApplyByLiveRegion asserts one live-region query match and returns the node.
func (parseF *Fixture) ApplyByLiveRegion(parsePoliteness string, parseText string) *QueryNode {
	parseF.tb.Helper()
	parseMatch := parseF.ByLiveRegion(parsePoliteness, parseText)
	if parseMatch == nil {
		parseF.tb.Fatalf("render fixture could not find live region politeness=%q text=%q", parsePoliteness, parseText)
	}
	return parseMatch
}

// ByID returns the first node with the requested id attribute.
func (parseF *Fixture) ByID(parseId string) *QueryNode {
	if parseF == nil || parseF.container == nil {
		return nil
	}
	return parseF.wrap(findNode(parseF.container, func(parseNode *mockdom.MockDOMNode) bool {
		return parseNode.Attrs["id"] == parseId
	}))
}

// ByText returns the first node whose full text content matches the provided text.
func (parseF *Fixture) ByText(parseText string) *QueryNode {
	if parseF == nil || parseF.container == nil {
		return nil
	}
	parseWant := normalizeText(parseText)
	return parseF.wrap(findNode(parseF.container, func(parseNode *mockdom.MockDOMNode) bool {
		return normalizeText(nodeText(parseNode)) == parseWant
	}))
}

// AllByTag returns all nodes matching the requested tag name.
func (parseF *Fixture) AllByTag(parseTag string) []*QueryNode {
	if parseF == nil || parseF.container == nil {
		return nil
	}
	parseMatches := collectNodes(parseF.container, func(parseNode *mockdom.MockDOMNode) bool {
		return strings.EqualFold(parseNode.Tag, parseTag)
	})
	parseResult := make([]*QueryNode, 0, len(parseMatches))
	for _, parseMatch := range parseMatches {
		parseResult = append(parseResult, parseF.wrap(parseMatch))
	}
	return parseResult
}

// Text returns the full fixture container text.
func (parseF *Fixture) Text() string {
	if parseF == nil || parseF.container == nil {
		return ""
	}
	return nodeText(parseF.container)
}

// DispatchByID invokes one handler property on the matched node and settles the fixture.
func (parseF *Fixture) DispatchByID(parseId string, parseProperty string, parseEvent Event) {
	parseF.tb.Helper()
	parseNode := parseF.ByID(parseId)
	if parseNode == nil {
		parseF.tb.Fatalf("render fixture could not find node with id %q", parseId)
	}
	parseNode.Dispatch(parseProperty, parseEvent)
}

// ClickByID invokes the matched node's `onclick` handler and settles the fixture.
func (parseF *Fixture) ClickByID(parseId string) {
	parseF.DispatchByID(parseId, "onclick", Event{})
}

// InputByID updates the matched node value, invokes `oninput`, and settles the fixture.
func (parseF *Fixture) InputByID(parseId string, parseValue string) {
	parseF.DispatchByID(parseId, "oninput", Event{Value: parseValue})
}

// ChangeByID updates the matched node value, invokes `onchange`, and settles the fixture.
func (parseF *Fixture) ChangeByID(parseId string, parseValue string) {
	parseF.DispatchByID(parseId, "onchange", Event{Value: parseValue})
}

// SubmitByID invokes the matched node's `onsubmit` handler and settles the fixture.
func (parseF *Fixture) SubmitByID(parseId string) {
	parseF.DispatchByID(parseId, "onsubmit", Event{})
}

// BuildOverlaySurfaces returns all rendered overlay surfaces sorted by depth.
func (parseF *Fixture) BuildOverlaySurfaces() []OverlaySurface {
	if parseF == nil || parseF.container == nil {
		return nil
	}
	parseMatches := collectNodes(parseF.container, func(parseNode *mockdom.MockDOMNode) bool {
		return strings.TrimSpace(parseNode.Attrs["data-overlay-kind"]) != ""
	})
	if len(parseMatches) == 0 {
		return nil
	}
	parseSurfaces := make([]OverlaySurface, 0, len(parseMatches))
	for _, parseNode := range parseMatches {
		parseSurfaces = append(parseSurfaces, OverlaySurface{
			SurfaceID:           strings.TrimSpace(parseNode.Attrs["id"]),
			Kind:                strings.TrimSpace(parseNode.Attrs["data-overlay-kind"]),
			Depth:               buildOverlayInt(parseNode.Attrs["data-overlay-depth"]),
			HandlesEscape:       buildOverlayBool(parseNode.Attrs["data-overlay-handles-escape"]),
			HandlesOutsideClick: buildOverlayBool(parseNode.Attrs["data-overlay-handles-outside"]),
			TrapFocusOwner:      buildOverlayBool(parseNode.Attrs["data-overlay-trap-owner"]),
			IsModal:             strings.EqualFold(strings.TrimSpace(parseNode.Attrs["aria-modal"]), "true"),
			PortalTargetID:      buildOverlayPortalTargetID(parseNode),
		})
	}
	sort.SliceStable(parseSurfaces, func(parseLeft, parseRight int) bool {
		if parseSurfaces[parseLeft].Depth == parseSurfaces[parseRight].Depth {
			return parseSurfaces[parseLeft].SurfaceID < parseSurfaces[parseRight].SurfaceID
		}
		return parseSurfaces[parseLeft].Depth < parseSurfaces[parseRight].Depth
	})
	return parseSurfaces
}

// BuildOverlayEscapeSurfaceID returns the topmost escape-handling overlay surface id.
func (parseF *Fixture) BuildOverlayEscapeSurfaceID() string {
	return buildOverlayOwnerSurfaceID(parseF.BuildOverlaySurfaces(), func(parseSurface OverlaySurface) bool {
		return parseSurface.HandlesEscape
	})
}

// BuildOverlayOutsideSurfaceID returns the topmost outside-click-handling overlay surface id.
func (parseF *Fixture) BuildOverlayOutsideSurfaceID() string {
	return buildOverlayOwnerSurfaceID(parseF.BuildOverlaySurfaces(), func(parseSurface OverlaySurface) bool {
		return parseSurface.HandlesOutsideClick
	})
}

// BuildOverlayFocusSurfaceID returns the topmost trap-focus owner overlay surface id.
func (parseF *Fixture) BuildOverlayFocusSurfaceID() string {
	return buildOverlayOwnerSurfaceID(parseF.BuildOverlaySurfaces(), func(parseSurface OverlaySurface) bool {
		return parseSurface.TrapFocusOwner
	})
}

// BuildOverlayScrollLockActive reports whether overlay-driven scroll lock is active.
func (parseF *Fixture) BuildOverlayScrollLockActive() bool {
	parseOverflow := strings.TrimSpace(parseF.BuildOverlayBodyOverflow())
	if strings.EqualFold(parseOverflow, "hidden") {
		return true
	}
	for _, parseSurface := range parseF.BuildOverlaySurfaces() {
		if parseSurface.IsModal {
			return true
		}
	}
	return false
}

// BuildOverlayBodyOverflow reports the current browser document body overflow style.
func (parseF *Fixture) BuildOverlayBodyOverflow() string {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return ""
	}
	parseBody := parseDocument.Get("body")
	if !parseBody.Truthy() {
		return ""
	}
	parseStyle := parseBody.Get("style")
	if !parseStyle.Truthy() {
		return ""
	}
	return strings.TrimSpace(parseStyle.Get("overflow").String())
}

// BuildOverlayPortalTargetID resolves one overlay surface to the nearest ancestor id.
func (parseF *Fixture) BuildOverlayPortalTargetID(parseSurfaceID string) string {
	if parseF == nil || parseF.container == nil {
		return ""
	}
	parseSurface := findNode(parseF.container, func(parseNode *mockdom.MockDOMNode) bool {
		return strings.TrimSpace(parseNode.Attrs["id"]) == strings.TrimSpace(parseSurfaceID) && strings.TrimSpace(parseNode.Attrs["data-overlay-kind"]) != ""
	})
	if parseSurface == nil {
		return ""
	}
	return buildOverlayPortalTargetID(parseSurface)
}

// HandleOverlayOutsideClick dispatches one outside-click dismissal through the overlay backdrop.
func (parseF *Fixture) HandleOverlayOutsideClick(parseSurfaceID string) bool {
	parseF.tb.Helper()
	parseF.requireActive()
	parseSurface := findNode(parseF.container, func(parseNode *mockdom.MockDOMNode) bool {
		return strings.TrimSpace(parseNode.Attrs["id"]) == strings.TrimSpace(parseSurfaceID) && strings.TrimSpace(parseNode.Attrs["data-overlay-kind"]) != ""
	})
	if parseSurface == nil || parseSurface.Parent == nil {
		return false
	}
	if parseSurface.Parent.Props["onclick"] == nil {
		return false
	}
	parseF.dispatch(parseSurface.Parent, "onclick", Event{})
	return true
}

// BuildDiagnostics returns structured runtime diagnostics captured for this fixture run.
func (parseF *Fixture) BuildDiagnostics() []DiagnosticSignal {
	parseDiagnostics := runtime.GetDiagnostics()
	if len(parseDiagnostics) == 0 {
		return nil
	}
	parseSignals := make([]DiagnosticSignal, 0, len(parseDiagnostics))
	for _, parseDiagnostic := range parseDiagnostics {
		parseSignals = append(parseSignals, DiagnosticSignal{
			Source:         parseDiagnostic.Source,
			Severity:       string(parseDiagnostic.Severity),
			Classification: string(parseDiagnostic.Classification),
			Code:           parseDiagnostic.Code,
			Message:        parseDiagnostic.Message,
			Count:          parseDiagnostic.Count,
			Path:           parseDiagnostic.Path,
			Recoverable:    parseDiagnostic.Recoverable,
			TopFrame:       parseDiagnostic.TopFrame,
			Consequence:    parseDiagnostic.Consequence,
			ComponentStack: append([]string(nil), parseDiagnostic.ComponentStack...),
			Fields:         cloneSignalFields(parseDiagnostic.Fields),
		})
	}
	return parseSignals
}

// BuildLogs returns buffered runtime logs captured for this fixture run.
func (parseF *Fixture) BuildLogs() []LogSignal {
	parseLogs := runtime.GetLogs()
	if len(parseLogs) == 0 {
		return nil
	}
	parseSignals := make([]LogSignal, 0, len(parseLogs))
	for _, parseLog := range parseLogs {
		parseSignals = append(parseSignals, LogSignal{
			Domain:         parseLog.Domain,
			Level:          string(parseLog.Level),
			Classification: string(parseLog.Classification),
			Code:           parseLog.Code,
			Message:        parseLog.Message,
			Timestamp:      parseLog.Timestamp,
			CorrelationID:  parseLog.CorrelationID,
			Recoverable:    parseLog.Recoverable,
			TopFrame:       parseLog.TopFrame,
			Consequence:    parseLog.Consequence,
			Fields:         cloneSignalFields(parseLog.Fields),
		})
	}
	return parseSignals
}

// BuildRenderCounts returns structured component render-count signals from profiling snapshots.
func (parseF *Fixture) BuildRenderCounts() []RenderCountSignal {
	parseSnapshot := runtime.GetGlobalRuntime().Inspect()
	parseTraces := parseSnapshot.Profiling.ComponentRenders
	if len(parseTraces) == 0 {
		return nil
	}
	parseSignals := make([]RenderCountSignal, 0, len(parseTraces))
	for _, parseTrace := range parseTraces {
		parseSignals = append(parseSignals, RenderCountSignal{
			Name:                    parseTrace.Name,
			Path:                    parseTrace.Path,
			RenderCount:             parseTrace.RenderCount,
			RerenderCount:           parseTrace.RerenderCount,
			LastTrigger:             parseTrace.LastTrigger,
			TotalRenderDurationNs:   parseTrace.TotalRenderDurationNs,
			AverageRenderDurationNs: parseTrace.AverageRenderDurationNs,
		})
	}
	sort.SliceStable(parseSignals, func(parseLeft, parseRight int) bool {
		if parseSignals[parseLeft].RenderCount == parseSignals[parseRight].RenderCount {
			return parseSignals[parseLeft].Path < parseSignals[parseRight].Path
		}
		return parseSignals[parseLeft].RenderCount > parseSignals[parseRight].RenderCount
	})
	return parseSignals
}

// BuildWarningDiagnostics returns diagnostics whose severity is warning.
func (parseF *Fixture) BuildWarningDiagnostics() []DiagnosticSignal {
	parseDiagnostics := parseF.BuildDiagnostics()
	if len(parseDiagnostics) == 0 {
		return nil
	}
	parseWarnings := make([]DiagnosticSignal, 0, len(parseDiagnostics))
	for _, parseDiagnostic := range parseDiagnostics {
		if strings.EqualFold(strings.TrimSpace(parseDiagnostic.Severity), "warning") {
			parseWarnings = append(parseWarnings, parseDiagnostic)
		}
	}
	return parseWarnings
}

// BuildWarningLogs returns logs whose level is warn.
func (parseF *Fixture) BuildWarningLogs() []LogSignal {
	parseLogs := parseF.BuildLogs()
	if len(parseLogs) == 0 {
		return nil
	}
	parseWarnings := make([]LogSignal, 0, len(parseLogs))
	for _, parseLog := range parseLogs {
		if strings.EqualFold(strings.TrimSpace(parseLog.Level), "warn") {
			parseWarnings = append(parseWarnings, parseLog)
		}
	}
	return parseWarnings
}

// ApplyDiagnosticCode asserts that one diagnostic with the requested code exists.
func (parseF *Fixture) ApplyDiagnosticCode(parseCode string) DiagnosticSignal {
	parseF.tb.Helper()
	parseCode := strings.TrimSpace(parseCode)
	for _, parseDiagnostic := range parseF.BuildDiagnostics() {
		if strings.TrimSpace(parseDiagnostic.Code) == parseCode {
			return parseDiagnostic
		}
	}
	parseF.tb.Fatalf("render fixture could not find diagnostic code %q", parseCode)
	return DiagnosticSignal{}
}

// ApplyDiagnosticMessage asserts that one diagnostic message contains the provided fragment.
func (parseF *Fixture) ApplyDiagnosticMessage(parseFragment string) DiagnosticSignal {
	parseF.tb.Helper()
	parseFragment := strings.TrimSpace(parseFragment)
	for _, parseDiagnostic := range parseF.BuildDiagnostics() {
		if strings.Contains(parseDiagnostic.Message, parseFragment) {
			return parseDiagnostic
		}
	}
	parseF.tb.Fatalf("render fixture could not find diagnostic message fragment %q", parseFragment)
	return DiagnosticSignal{}
}

// ApplyLogCode asserts that one buffered log with the requested code exists.
func (parseF *Fixture) ApplyLogCode(parseCode string) LogSignal {
	parseF.tb.Helper()
	parseCode := strings.TrimSpace(parseCode)
	for _, parseLog := range parseF.BuildLogs() {
		if strings.TrimSpace(parseLog.Code) == parseCode {
			return parseLog
		}
	}
	parseF.tb.Fatalf("render fixture could not find log code %q", parseCode)
	return LogSignal{}
}

// ApplyLogMessage asserts that one buffered log message contains the provided fragment.
func (parseF *Fixture) ApplyLogMessage(parseFragment string) LogSignal {
	parseF.tb.Helper()
	parseFragment := strings.TrimSpace(parseFragment)
	for _, parseLog := range parseF.BuildLogs() {
		if strings.Contains(parseLog.Message, parseFragment) {
			return parseLog
		}
	}
	parseF.tb.Fatalf("render fixture could not find log message fragment %q", parseFragment)
	return LogSignal{}
}

// ApplyRenderCountMax asserts one component's render count does not exceed max.
func (parseF *Fixture) ApplyRenderCountMax(parseComponent string, parseMax int) RenderCountSignal {
	parseF.tb.Helper()
	parseSignal := applyRenderCountSignal(parseF.BuildRenderCounts(), parseComponent)
	if strings.TrimSpace(parseSignal.Name) == "" && strings.TrimSpace(parseSignal.Path) == "" {
		parseF.tb.Fatalf("render fixture could not resolve render-count signal for component %q", strings.TrimSpace(parseComponent))
	}
	if parseSignal.RenderCount > parseMax {
		parseF.tb.Fatalf("render fixture component %q exceeded max render count %d with %d renders (path=%q)", strings.TrimSpace(parseComponent), parseMax, parseSignal.RenderCount, parseSignal.Path)
	}
	return parseSignal
}

// ApplyRenderRerenderMax asserts one component's rerender count does not exceed max.
func (parseF *Fixture) ApplyRenderRerenderMax(parseComponent string, parseMax int) RenderCountSignal {
	parseF.tb.Helper()
	parseSignal := applyRenderCountSignal(parseF.BuildRenderCounts(), parseComponent)
	if strings.TrimSpace(parseSignal.Name) == "" && strings.TrimSpace(parseSignal.Path) == "" {
		parseF.tb.Fatalf("render fixture could not resolve rerender signal for component %q", strings.TrimSpace(parseComponent))
	}
	if parseSignal.RerenderCount > parseMax {
		parseF.tb.Fatalf("render fixture component %q exceeded max rerender count %d with %d rerenders (path=%q)", strings.TrimSpace(parseComponent), parseMax, parseSignal.RerenderCount, parseSignal.Path)
	}
	return parseSignal
}

// ApplyWarningCountMax asserts warnings across diagnostics and logs do not exceed max.
func (parseF *Fixture) ApplyWarningCountMax(parseMax int) {
	parseF.tb.Helper()
	parseWarningDiagnostics := parseF.BuildWarningDiagnostics()
	parseWarningLogs := parseF.BuildWarningLogs()
	parseWarningCount := len(parseWarningDiagnostics) + len(parseWarningLogs)
	if parseWarningCount > parseMax {
		parseF.tb.Fatalf("render fixture warning count %d exceeds max %d (diagnostics=%d logs=%d)", parseWarningCount, parseMax, len(parseWarningDiagnostics), len(parseWarningLogs))
	}
}

// ApplyWarningNone asserts no warning diagnostics or warn-level logs were emitted.
func (parseF *Fixture) ApplyWarningNone() {
	parseF.tb.Helper()
	parseF.ApplyWarningCountMax(0)
}

// Exists reports whether the node wrapper points at a real node.
func (parseN *QueryNode) Exists() bool {
	return parseN != nil && parseN.node != nil
}

// Tag returns the node tag name.
func (parseN *QueryNode) Tag() string {
	if parseN == nil || parseN.node == nil {
		return ""
	}
	return parseN.node.Tag
}

// Name returns the node accessible name used by role-based queries.
func (parseN *QueryNode) Name() string {
	if parseN == nil || parseN.node == nil || parseN.fixture == nil || parseN.fixture.container == nil {
		return ""
	}
	return accessibleName(parseN.fixture.container, parseN.node)
}

// Text returns the full text content beneath the node.
func (parseN *QueryNode) Text() string {
	if parseN == nil || parseN.node == nil {
		return ""
	}
	return nodeText(parseN.node)
}

// Attr returns one attribute value.
func (parseN *QueryNode) Attr(parseName string) string {
	if parseN == nil || parseN.node == nil {
		return ""
	}
	return parseN.node.Attrs[parseName]
}

// NodeID returns the stable mock-DOM node id for identity-sensitive assertions.
func (parseN *QueryNode) NodeID() int {
	if parseN == nil || parseN.node == nil {
		return 0
	}
	return parseN.node.ID
}

// Property returns one raw property value from the rendered node.
func (parseN *QueryNode) Property(parseName string) any {
	if parseN == nil || parseN.node == nil {
		return nil
	}
	return parseN.node.Props[parseName]
}

// Children returns wrapped child nodes.
func (parseN *QueryNode) Children() []*QueryNode {
	if parseN == nil || parseN.node == nil || len(parseN.node.Children) == 0 {
		return nil
	}
	parseChildren := make([]*QueryNode, 0, len(parseN.node.Children))
	for _, parseChild := range parseN.node.Children {
		parseChildren = append(parseChildren, parseN.fixture.wrap(parseChild))
	}
	return parseChildren
}

// Dispatch invokes one handler property on the current node and settles the fixture.
func (parseN *QueryNode) Dispatch(parseProperty string, parseEvent Event) {
	if parseN == nil || parseN.node == nil || parseN.fixture == nil {
		return
	}
	parseN.fixture.dispatch(parseN.node, parseProperty, parseEvent)
}

// Click invokes the current node's `onclick` handler and settles the fixture.
func (parseN *QueryNode) Click() {
	parseN.Dispatch("onclick", Event{})
}

// Input updates the current node value, invokes `oninput`, and settles the fixture.
func (parseN *QueryNode) Input(parseValue string) {
	parseN.Dispatch("oninput", Event{Value: parseValue})
}

// Change updates the current node value, invokes `onchange`, and settles the fixture.
func (parseN *QueryNode) Change(parseValue string) {
	parseN.Dispatch("onchange", Event{Value: parseValue})
}

// Submit invokes the current node's `onsubmit` handler and settles the fixture.
func (parseN *QueryNode) Submit() {
	parseN.Dispatch("onsubmit", Event{})
}

func (parseF *Fixture) requireActive() {
	if parseF == nil || parseF.cleaned || parseF.container == nil || parseF.scheduler == nil {
		parseF.tb.Fatal("render fixture is no longer active")
	}
}

func (parseF *Fixture) wrap(parseNode *mockdom.MockDOMNode) *QueryNode {
	if parseNode == nil {
		return nil
	}
	return &QueryNode{fixture: parseF, node: parseNode}
}

func (parseF *Fixture) dispatch(parseNode *mockdom.MockDOMNode, parseProperty string, parseEvent Event) {
	parseF.tb.Helper()
	parseF.requireActive()
	if parseNode == nil {
		parseF.tb.Fatalf("render fixture cannot dispatch %q on a nil node", parseProperty)
	}
	parseHandler := parseNode.Props[parseProperty]
	if parseHandler == nil {
		parseF.tb.Fatalf("render fixture node %q does not expose handler property %q", parseNode.Attrs["id"], parseProperty)
	}
	parseEvent.apply(parseNode)
	parseSynthetic := parseEvent.syntheticValue()
	parseSyntheticGoEvent := runtime.NewGoEvent(parseSynthetic)

	switch parseTyped := parseHandler.(type) {
	case func():
		parseTyped()
	case func(string):
		parseTyped(parseEvent.Value)
	case func(js.Value):
		parseTyped(parseSynthetic)
	case func(js.Value) error:
		if parseErr := parseTyped(parseSynthetic); parseErr != nil {
			parseF.tb.Fatalf("render fixture handler %q returned error: %v", parseProperty, parseErr)
		}
	case func(runtime.GoEvent):
		parseTyped(parseSyntheticGoEvent)
	case func(runtime.GoEvent) error:
		if parseErr2 := parseTyped(parseSyntheticGoEvent); parseErr2 != nil {
			parseF.tb.Fatalf("render fixture handler %q returned error: %v", parseProperty, parseErr2)
		}
	case func() error:
		if parseErr3 := parseTyped(); parseErr3 != nil {
			parseF.tb.Fatalf("render fixture handler %q returned error: %v", parseProperty, parseErr3)
		}
	default:
		parseF.tb.Fatalf("render fixture does not know how to dispatch handler property %q with type %T", parseProperty, parseHandler)
	}
	if parseProperty == "onchange" || parseProperty == "oninput" || parseProperty == "onsubmit" || parseProperty == "onclick" {
		parseF.Stabilize()
	}
}

func findNode(parseNode *mockdom.MockDOMNode, parseMatch func(*mockdom.MockDOMNode) bool) *mockdom.MockDOMNode {
	if parseNode == nil {
		return nil
	}
	if parseMatch(parseNode) {
		return parseNode
	}
	for _, parseChild := range parseNode.Children {
		if parseFound := findNode(parseChild, parseMatch); parseFound != nil {
			return parseFound
		}
	}
	return nil
}

func collectNodes(parseNode *mockdom.MockDOMNode, parseMatch func(*mockdom.MockDOMNode) bool) []*mockdom.MockDOMNode {
	if parseNode == nil {
		return nil
	}
	parseResult := make([]*mockdom.MockDOMNode, 0)
	if parseMatch(parseNode) {
		parseResult = append(parseResult, parseNode)
	}
	for _, parseChild := range parseNode.Children {
		parseResult = append(parseResult, collectNodes(parseChild, parseMatch)...)
	}
	return parseResult
}

func nodeText(parseNode *mockdom.MockDOMNode) string {
	if parseNode == nil {
		return ""
	}
	if parseNode.Tag == "#text" {
		return parseNode.TextContent
	}
	parseParts := make([]string, 0, len(parseNode.Children)+1)
	if strings.TrimSpace(parseNode.TextContent) != "" {
		parseParts = append(parseParts, strings.TrimSpace(parseNode.TextContent))
	}
	for _, parseChild := range parseNode.Children {
		parseText := strings.TrimSpace(nodeText(parseChild))
		if parseText != "" {
			parseParts = append(parseParts, parseText)
		}
	}
	return strings.Join(parseParts, " ")
}

func normalizeText(parseValue string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(parseValue)), " ")
}

func nodeRole(parseNode *mockdom.MockDOMNode) string {
	if parseNode == nil {
		return ""
	}
	if parseRole := strings.TrimSpace(parseNode.Attrs["role"]); parseRole != "" {
		return parseRole
	}
	switch strings.ToLower(strings.TrimSpace(parseNode.Tag)) {
	case "button":
		return "button"
	case "a":
		if strings.TrimSpace(parseNode.Attrs["href"]) != "" {
			return "link"
		}
	case "textarea":
		return "textbox"
	case "select":
		return "combobox"
	case "img":
		return "img"
	case "form":
		return "form"
	case "input":
		switch strings.ToLower(strings.TrimSpace(parseNode.Attrs["type"])) {
		case "button", "submit", "reset":
			return "button"
		case "checkbox":
			return "checkbox"
		case "radio":
			return "radio"
		case "range":
			return "slider"
		case "email", "password", "search", "tel", "text", "url", "":
			return "textbox"
		}
	}
	return ""
}

func accessibleName(parseRoot *mockdom.MockDOMNode, parseNode *mockdom.MockDOMNode) string {
	if parseNode == nil {
		return ""
	}
	if parseLabel := normalizeText(parseNode.Attrs["aria-label"]); parseLabel != "" {
		return parseLabel
	}
	if parseRefs := normalizeText(parseNode.Attrs["aria-labelledby"]); parseRefs != "" {
		parseParts := make([]string, 0)
		for _, parseRef := range strings.Fields(parseRefs) {
			if parseTarget := findNode(parseRoot, func(candidate *mockdom.MockDOMNode) bool {
				return candidate != nil && candidate.Attrs["id"] == parseRef
			}); parseTarget != nil {
				parseText := normalizeText(nodeText(parseTarget))
				if parseText != "" {
					parseParts = append(parseParts, parseText)
				}
			}
		}
		if len(parseParts) > 0 {
			return strings.Join(parseParts, " ")
		}
	}
	return normalizeText(nodeText(parseNode))
}

// accessibleDescription resolves the accessible description for one node.
func accessibleDescription(parseRoot *mockdom.MockDOMNode, parseNode *mockdom.MockDOMNode) string {
	if parseNode == nil {
		return ""
	}
	if parseDescription := normalizeText(parseNode.Attrs["aria-description"]); parseDescription != "" {
		return parseDescription
	}
	if parseRefs := normalizeText(parseNode.Attrs["aria-describedby"]); parseRefs != "" {
		parseParts := make([]string, 0)
		for _, parseRef := range strings.Fields(parseRefs) {
			parseTarget := findNode(parseRoot, func(parseCandidate *mockdom.MockDOMNode) bool {
				return parseCandidate != nil && parseCandidate.Attrs["id"] == parseRef
			})
			if parseTarget == nil {
				continue
			}
			parseText := normalizeText(nodeText(parseTarget))
			if parseText != "" {
				parseParts = append(parseParts, parseText)
			}
		}
		if len(parseParts) > 0 {
			return strings.Join(parseParts, " ")
		}
	}
	return ""
}

// nodeLivePoliteness resolves the live-region politeness for one node.
func nodeLivePoliteness(parseNode *mockdom.MockDOMNode) string {
	if parseNode == nil {
		return ""
	}
	if parseLive := normalizeText(parseNode.Attrs["aria-live"]); parseLive != "" {
		if parseLive == "off" {
			return ""
		}
		return parseLive
	}
	switch normalizeText(nodeRole(parseNode)) {
	case "status", "log":
		return "polite"
	case "alert":
		return "assertive"
	}
	return ""
}

func (parseE Event) apply(parseNode *mockdom.MockDOMNode) {
	if parseNode == nil {
		return
	}
	if parseE.Value != "" || parseNode.Tag == "input" || parseNode.Tag == "textarea" || parseNode.Tag == "select" {
		parseNode.Attrs["value"] = parseE.Value
		parseNode.Props["value"] = parseE.Value
	}
	parseNode.Props["checked"] = parseE.Checked
	if parseE.Checked {
		parseNode.Attrs["checked"] = ""
	} else {
		delete(parseNode.Attrs, "checked")
	}
}

func (parseE Event) syntheticValue() js.Value {
	parseObject := js.Global().Get("Object")
	parseTarget := parseObject.New()
	parseTarget.Set("value", parseE.Value)
	parseTarget.Set("checked", parseE.Checked)
	parseEvent := parseObject.New()
	parseEvent.Set("target", parseTarget)
	parseEvent.Set("key", parseE.Key)
	parseEvent.Set("keyCode", parseE.KeyCode)
	return parseEvent
}

// buildOverlayBool parses one overlay boolean attribute value.
func buildOverlayBool(parseValue string) bool {
	return strings.EqualFold(strings.TrimSpace(parseValue), "true")
}

// buildOverlayInt parses one overlay integer attribute value.
func buildOverlayInt(parseValue string) int {
	parseParsed, parseErr := strconv.Atoi(strings.TrimSpace(parseValue))
	if parseErr != nil {
		return 0
	}
	return parseParsed
}

// buildOverlayOwnerSurfaceID resolves the topmost overlay id for one ownership selector.
func buildOverlayOwnerSurfaceID(parseSurfaces []OverlaySurface, parseMatch func(OverlaySurface) bool) string {
	for parseIndex := len(parseSurfaces) - 1; parseIndex >= 0; parseIndex-- {
		if parseMatch(parseSurfaces[parseIndex]) {
			return parseSurfaces[parseIndex].SurfaceID
		}
	}
	return ""
}

// buildOverlayPortalTargetID resolves the nearest ancestor id for one overlay node.
func buildOverlayPortalTargetID(parseSurface *mockdom.MockDOMNode) string {
	if parseSurface == nil {
		return ""
	}
	for parseParent := parseSurface.Parent; parseParent != nil; parseParent = parseParent.Parent {
		parseID := strings.TrimSpace(parseParent.Attrs["id"])
		if parseID != "" {
			return parseID
		}
	}
	return ""
}

// cloneSignalFields clones one log or diagnostic field map.
func cloneSignalFields(parseValues map[string]string) map[string]string {
	if len(parseValues) == 0 {
		return map[string]string{}
	}
	parseCloned := make(map[string]string, len(parseValues))
	for parseKey, parseValue := range parseValues {
		parseCloned[parseKey] = parseValue
	}
	return parseCloned
}

// applyRenderCountSignal resolves one render-count signal by component name/path.
func applyRenderCountSignal(parseSignals []RenderCountSignal, parseComponent string) RenderCountSignal {
	parseComponent := strings.TrimSpace(parseComponent)
	if len(parseSignals) == 0 {
		return RenderCountSignal{}
	}
	if parseComponent == "" {
		return parseSignals[0]
	}
	for _, parseSignal := range parseSignals {
		if parseSignal.Name == parseComponent || parseSignal.Path == parseComponent || strings.Contains(parseSignal.Path, parseComponent) {
			return parseSignal
		}
	}
	return RenderCountSignal{}
}

// applyFixtureOwnership claims exclusive fixture ownership for this test process.
func applyFixtureOwnership(parseTb testing.TB) {
	parseTb.Helper()
	select {
	case fixtureGate <- struct{}{}:
	default:
		parseTb.Fatalf("render fixture ownership contention: %s", ParallelSafetyContract())
	}
}

// clearFixtureOwnership releases exclusive fixture ownership for this test process.
func clearFixtureOwnership() {
	select {
	case <-fixtureGate:
	default:
	}
}
