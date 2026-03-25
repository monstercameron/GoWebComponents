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
	return func(cfg *config) {
		cfg.synchronous = false
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
func New(tb testing.TB, options ...Option) *Fixture {
	tb.Helper()
	cfg := config{synchronous: true}
	for _, option := range options {
		if option != nil {
			option(&cfg)
		}
	}

	applyFixtureOwnership(tb)
	adapter := mockdom.NewMockDOMAdapter()
	scheduler := mockdom.NewMockScheduler(cfg.synchronous)
	container, _ := adapter.CreateElement("div").(*mockdom.MockDOMNode)
	runtime.InitGlobalRuntime(runtime.Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
		Reset:      true,
	})
	runtime.ClearDiagnostics()
	runtime.ClearLogs()

	fixture := &Fixture{
		tb:        tb,
		adapter:   adapter,
		scheduler: scheduler,
		container: container,
	}
	tb.Cleanup(func() {
		fixture.Cleanup()
	})
	return fixture
}

// Render mounts a UI tree into the fixture container.
func (f *Fixture) Render(root ui.Node) {
	f.tb.Helper()
	f.requireActive()
	if err := runtime.GetGlobalRuntime().RenderInto(f.container, root); err != nil {
		f.tb.Fatalf("render fixture failed to mount root: %v", err)
	}
	f.Flush()
}

// Rerender replaces the current tree with a new root.
func (f *Fixture) Rerender(root ui.Node) {
	f.Render(root)
}

// Flush drains queued scheduler work until the fixture settles.
func (f *Fixture) Flush() {
	f.tb.Helper()
	f.requireActive()
	f.scheduler.FlushAll()
}

// FlushTimers drains queued timeout work and any follow-up render work.
func (f *Fixture) FlushTimers() {
	f.tb.Helper()
	f.requireActive()
	f.scheduler.FlushTimeouts()
	f.scheduler.FlushAll()
}

// Stabilize drains pending scheduled work until the fixture settles.
func (f *Fixture) Stabilize() {
	f.Flush()
}

// Cleanup releases fixture ownership and clears buffered diagnostics.
func (f *Fixture) Cleanup() {
	if f == nil || f.cleaned {
		return
	}
	f.cleaned = true
	runtime.ClearDiagnostics()
	runtime.ClearLogs()
	f.container = nil
	f.adapter = nil
	f.scheduler = nil
	f.unlock.Do(func() {
		clearFixtureOwnership()
	})
}

// Container returns the fixture root container.
func (f *Fixture) Container() *QueryNode {
	if f == nil || f.container == nil {
		return nil
	}
	return &QueryNode{fixture: f, node: f.container}
}

// Target returns the explicit DOM target owned by the fixture.
func (f *Fixture) Target() any {
	if f == nil || f.container == nil {
		return nil
	}
	return f.container
}

// ByRole returns the first node whose computed role matches, optionally filtered by accessible name.
func (f *Fixture) ByRole(role string, name string) *QueryNode {
	if f == nil || f.container == nil {
		return nil
	}
	wantRole := normalizeText(role)
	wantName := normalizeText(name)
	return f.wrap(findNode(f.container, func(node *mockdom.MockDOMNode) bool {
		if node == nil || normalizeText(nodeRole(node)) != wantRole {
			return false
		}
		if wantName == "" {
			return true
		}
		return normalizeText(accessibleName(f.container, node)) == wantName
	}))
}

// AllByRole returns all nodes whose computed role matches.
func (f *Fixture) AllByRole(role string) []*QueryNode {
	if f == nil || f.container == nil {
		return nil
	}
	wantRole := normalizeText(role)
	matches := collectNodes(f.container, func(node *mockdom.MockDOMNode) bool {
		return normalizeText(nodeRole(node)) == wantRole
	})
	result := make([]*QueryNode, 0, len(matches))
	for _, match := range matches {
		result = append(result, f.wrap(match))
	}
	return result
}

// ByLabel returns the first interactive node whose accessible label matches.
func (f *Fixture) ByLabel(label string) *QueryNode {
	if f == nil || f.container == nil {
		return nil
	}
	parseLabel := normalizeText(label)
	return f.wrap(findNode(f.container, func(parseNode *mockdom.MockDOMNode) bool {
		if parseNode == nil || nodeRole(parseNode) == "" {
			return false
		}
		return normalizeText(accessibleName(f.container, parseNode)) == parseLabel
	}))
}

// ByDescription returns the first interactive node whose accessible description matches.
func (f *Fixture) ByDescription(description string) *QueryNode {
	if f == nil || f.container == nil {
		return nil
	}
	parseDescription := normalizeText(description)
	return f.wrap(findNode(f.container, func(parseNode *mockdom.MockDOMNode) bool {
		if parseNode == nil || nodeRole(parseNode) == "" {
			return false
		}
		return normalizeText(accessibleDescription(f.container, parseNode)) == parseDescription
	}))
}

// ByLiveRegion returns the first live-region node matching politeness and optional text.
func (f *Fixture) ByLiveRegion(politeness string, text string) *QueryNode {
	if f == nil || f.container == nil {
		return nil
	}
	parsePoliteness := normalizeText(politeness)
	parseText := normalizeText(text)
	return f.wrap(findNode(f.container, func(parseNode *mockdom.MockDOMNode) bool {
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
func (f *Fixture) ApplyByRole(role string, name string) *QueryNode {
	f.tb.Helper()
	parseMatch := f.ByRole(role, name)
	if parseMatch == nil {
		f.tb.Fatalf("render fixture could not find role=%q name=%q", role, name)
	}
	return parseMatch
}

// ApplyByLabel asserts one label query match and returns the node.
func (f *Fixture) ApplyByLabel(label string) *QueryNode {
	f.tb.Helper()
	parseMatch := f.ByLabel(label)
	if parseMatch == nil {
		f.tb.Fatalf("render fixture could not find label=%q", label)
	}
	return parseMatch
}

// ApplyByDescription asserts one description query match and returns the node.
func (f *Fixture) ApplyByDescription(description string) *QueryNode {
	f.tb.Helper()
	parseMatch := f.ByDescription(description)
	if parseMatch == nil {
		f.tb.Fatalf("render fixture could not find description=%q", description)
	}
	return parseMatch
}

// ApplyByLiveRegion asserts one live-region query match and returns the node.
func (f *Fixture) ApplyByLiveRegion(politeness string, text string) *QueryNode {
	f.tb.Helper()
	parseMatch := f.ByLiveRegion(politeness, text)
	if parseMatch == nil {
		f.tb.Fatalf("render fixture could not find live region politeness=%q text=%q", politeness, text)
	}
	return parseMatch
}

// ByID returns the first node with the requested id attribute.
func (f *Fixture) ByID(id string) *QueryNode {
	if f == nil || f.container == nil {
		return nil
	}
	return f.wrap(findNode(f.container, func(node *mockdom.MockDOMNode) bool {
		return node.Attrs["id"] == id
	}))
}

// ByText returns the first node whose full text content matches the provided text.
func (f *Fixture) ByText(text string) *QueryNode {
	if f == nil || f.container == nil {
		return nil
	}
	want := normalizeText(text)
	return f.wrap(findNode(f.container, func(node *mockdom.MockDOMNode) bool {
		return normalizeText(nodeText(node)) == want
	}))
}

// AllByTag returns all nodes matching the requested tag name.
func (f *Fixture) AllByTag(tag string) []*QueryNode {
	if f == nil || f.container == nil {
		return nil
	}
	matches := collectNodes(f.container, func(node *mockdom.MockDOMNode) bool {
		return strings.EqualFold(node.Tag, tag)
	})
	result := make([]*QueryNode, 0, len(matches))
	for _, match := range matches {
		result = append(result, f.wrap(match))
	}
	return result
}

// Text returns the full fixture container text.
func (f *Fixture) Text() string {
	if f == nil || f.container == nil {
		return ""
	}
	return nodeText(f.container)
}

// DispatchByID invokes one handler property on the matched node and settles the fixture.
func (f *Fixture) DispatchByID(id string, property string, event Event) {
	f.tb.Helper()
	node := f.ByID(id)
	if node == nil {
		f.tb.Fatalf("render fixture could not find node with id %q", id)
	}
	node.Dispatch(property, event)
}

// ClickByID invokes the matched node's `onclick` handler and settles the fixture.
func (f *Fixture) ClickByID(id string) {
	f.DispatchByID(id, "onclick", Event{})
}

// InputByID updates the matched node value, invokes `oninput`, and settles the fixture.
func (f *Fixture) InputByID(id string, value string) {
	f.DispatchByID(id, "oninput", Event{Value: value})
}

// ChangeByID updates the matched node value, invokes `onchange`, and settles the fixture.
func (f *Fixture) ChangeByID(id string, value string) {
	f.DispatchByID(id, "onchange", Event{Value: value})
}

// SubmitByID invokes the matched node's `onsubmit` handler and settles the fixture.
func (f *Fixture) SubmitByID(id string) {
	f.DispatchByID(id, "onsubmit", Event{})
}

// BuildOverlaySurfaces returns all rendered overlay surfaces sorted by depth.
func (f *Fixture) BuildOverlaySurfaces() []OverlaySurface {
	if f == nil || f.container == nil {
		return nil
	}
	parseMatches := collectNodes(f.container, func(parseNode *mockdom.MockDOMNode) bool {
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
func (f *Fixture) BuildOverlayEscapeSurfaceID() string {
	return buildOverlayOwnerSurfaceID(f.BuildOverlaySurfaces(), func(parseSurface OverlaySurface) bool {
		return parseSurface.HandlesEscape
	})
}

// BuildOverlayOutsideSurfaceID returns the topmost outside-click-handling overlay surface id.
func (f *Fixture) BuildOverlayOutsideSurfaceID() string {
	return buildOverlayOwnerSurfaceID(f.BuildOverlaySurfaces(), func(parseSurface OverlaySurface) bool {
		return parseSurface.HandlesOutsideClick
	})
}

// BuildOverlayFocusSurfaceID returns the topmost trap-focus owner overlay surface id.
func (f *Fixture) BuildOverlayFocusSurfaceID() string {
	return buildOverlayOwnerSurfaceID(f.BuildOverlaySurfaces(), func(parseSurface OverlaySurface) bool {
		return parseSurface.TrapFocusOwner
	})
}

// BuildOverlayScrollLockActive reports whether overlay-driven scroll lock is active.
func (f *Fixture) BuildOverlayScrollLockActive() bool {
	parseOverflow := strings.TrimSpace(f.BuildOverlayBodyOverflow())
	if strings.EqualFold(parseOverflow, "hidden") {
		return true
	}
	for _, parseSurface := range f.BuildOverlaySurfaces() {
		if parseSurface.IsModal {
			return true
		}
	}
	return false
}

// BuildOverlayBodyOverflow reports the current browser document body overflow style.
func (f *Fixture) BuildOverlayBodyOverflow() string {
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
func (f *Fixture) BuildOverlayPortalTargetID(surfaceID string) string {
	if f == nil || f.container == nil {
		return ""
	}
	parseSurface := findNode(f.container, func(parseNode *mockdom.MockDOMNode) bool {
		return strings.TrimSpace(parseNode.Attrs["id"]) == strings.TrimSpace(surfaceID) && strings.TrimSpace(parseNode.Attrs["data-overlay-kind"]) != ""
	})
	if parseSurface == nil {
		return ""
	}
	return buildOverlayPortalTargetID(parseSurface)
}

// HandleOverlayOutsideClick dispatches one outside-click dismissal through the overlay backdrop.
func (f *Fixture) HandleOverlayOutsideClick(surfaceID string) bool {
	f.tb.Helper()
	f.requireActive()
	parseSurface := findNode(f.container, func(parseNode *mockdom.MockDOMNode) bool {
		return strings.TrimSpace(parseNode.Attrs["id"]) == strings.TrimSpace(surfaceID) && strings.TrimSpace(parseNode.Attrs["data-overlay-kind"]) != ""
	})
	if parseSurface == nil || parseSurface.Parent == nil {
		return false
	}
	if parseSurface.Parent.Props["onclick"] == nil {
		return false
	}
	f.dispatch(parseSurface.Parent, "onclick", Event{})
	return true
}

// BuildDiagnostics returns structured runtime diagnostics captured for this fixture run.
func (f *Fixture) BuildDiagnostics() []DiagnosticSignal {
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
func (f *Fixture) BuildLogs() []LogSignal {
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
func (f *Fixture) BuildRenderCounts() []RenderCountSignal {
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
func (f *Fixture) BuildWarningDiagnostics() []DiagnosticSignal {
	parseDiagnostics := f.BuildDiagnostics()
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
func (f *Fixture) BuildWarningLogs() []LogSignal {
	parseLogs := f.BuildLogs()
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
func (f *Fixture) ApplyDiagnosticCode(code string) DiagnosticSignal {
	f.tb.Helper()
	parseCode := strings.TrimSpace(code)
	for _, parseDiagnostic := range f.BuildDiagnostics() {
		if strings.TrimSpace(parseDiagnostic.Code) == parseCode {
			return parseDiagnostic
		}
	}
	f.tb.Fatalf("render fixture could not find diagnostic code %q", parseCode)
	return DiagnosticSignal{}
}

// ApplyDiagnosticMessage asserts that one diagnostic message contains the provided fragment.
func (f *Fixture) ApplyDiagnosticMessage(fragment string) DiagnosticSignal {
	f.tb.Helper()
	parseFragment := strings.TrimSpace(fragment)
	for _, parseDiagnostic := range f.BuildDiagnostics() {
		if strings.Contains(parseDiagnostic.Message, parseFragment) {
			return parseDiagnostic
		}
	}
	f.tb.Fatalf("render fixture could not find diagnostic message fragment %q", parseFragment)
	return DiagnosticSignal{}
}

// ApplyLogCode asserts that one buffered log with the requested code exists.
func (f *Fixture) ApplyLogCode(code string) LogSignal {
	f.tb.Helper()
	parseCode := strings.TrimSpace(code)
	for _, parseLog := range f.BuildLogs() {
		if strings.TrimSpace(parseLog.Code) == parseCode {
			return parseLog
		}
	}
	f.tb.Fatalf("render fixture could not find log code %q", parseCode)
	return LogSignal{}
}

// ApplyLogMessage asserts that one buffered log message contains the provided fragment.
func (f *Fixture) ApplyLogMessage(fragment string) LogSignal {
	f.tb.Helper()
	parseFragment := strings.TrimSpace(fragment)
	for _, parseLog := range f.BuildLogs() {
		if strings.Contains(parseLog.Message, parseFragment) {
			return parseLog
		}
	}
	f.tb.Fatalf("render fixture could not find log message fragment %q", parseFragment)
	return LogSignal{}
}

// ApplyRenderCountMax asserts one component's render count does not exceed max.
func (f *Fixture) ApplyRenderCountMax(component string, max int) RenderCountSignal {
	f.tb.Helper()
	parseSignal := applyRenderCountSignal(f.BuildRenderCounts(), component)
	if parseSignal.RenderCount > max {
		f.tb.Fatalf("render fixture component %q exceeded max render count %d with %d renders (path=%q)", strings.TrimSpace(component), max, parseSignal.RenderCount, parseSignal.Path)
	}
	return parseSignal
}

// ApplyRenderRerenderMax asserts one component's rerender count does not exceed max.
func (f *Fixture) ApplyRenderRerenderMax(component string, max int) RenderCountSignal {
	f.tb.Helper()
	parseSignal := applyRenderCountSignal(f.BuildRenderCounts(), component)
	if parseSignal.RerenderCount > max {
		f.tb.Fatalf("render fixture component %q exceeded max rerender count %d with %d rerenders (path=%q)", strings.TrimSpace(component), max, parseSignal.RerenderCount, parseSignal.Path)
	}
	return parseSignal
}

// ApplyWarningCountMax asserts warnings across diagnostics and logs do not exceed max.
func (f *Fixture) ApplyWarningCountMax(max int) {
	f.tb.Helper()
	parseWarningDiagnostics := f.BuildWarningDiagnostics()
	parseWarningLogs := f.BuildWarningLogs()
	parseWarningCount := len(parseWarningDiagnostics) + len(parseWarningLogs)
	if parseWarningCount > max {
		f.tb.Fatalf("render fixture warning count %d exceeds max %d (diagnostics=%d logs=%d)", parseWarningCount, max, len(parseWarningDiagnostics), len(parseWarningLogs))
	}
}

// ApplyWarningNone asserts no warning diagnostics or warn-level logs were emitted.
func (f *Fixture) ApplyWarningNone() {
	f.tb.Helper()
	f.ApplyWarningCountMax(0)
}

// Exists reports whether the node wrapper points at a real node.
func (n *QueryNode) Exists() bool {
	return n != nil && n.node != nil
}

// Tag returns the node tag name.
func (n *QueryNode) Tag() string {
	if n == nil || n.node == nil {
		return ""
	}
	return n.node.Tag
}

// Name returns the node accessible name used by role-based queries.
func (n *QueryNode) Name() string {
	if n == nil || n.node == nil || n.fixture == nil || n.fixture.container == nil {
		return ""
	}
	return accessibleName(n.fixture.container, n.node)
}

// Text returns the full text content beneath the node.
func (n *QueryNode) Text() string {
	if n == nil || n.node == nil {
		return ""
	}
	return nodeText(n.node)
}

// Attr returns one attribute value.
func (n *QueryNode) Attr(name string) string {
	if n == nil || n.node == nil {
		return ""
	}
	return n.node.Attrs[name]
}

// NodeID returns the stable mock-DOM node id for identity-sensitive assertions.
func (n *QueryNode) NodeID() int {
	if n == nil || n.node == nil {
		return 0
	}
	return n.node.ID
}

// Property returns one raw property value from the rendered node.
func (n *QueryNode) Property(name string) any {
	if n == nil || n.node == nil {
		return nil
	}
	return n.node.Props[name]
}

// Children returns wrapped child nodes.
func (n *QueryNode) Children() []*QueryNode {
	if n == nil || n.node == nil || len(n.node.Children) == 0 {
		return nil
	}
	children := make([]*QueryNode, 0, len(n.node.Children))
	for _, child := range n.node.Children {
		children = append(children, n.fixture.wrap(child))
	}
	return children
}

// Dispatch invokes one handler property on the current node and settles the fixture.
func (n *QueryNode) Dispatch(property string, event Event) {
	if n == nil || n.node == nil || n.fixture == nil {
		return
	}
	n.fixture.dispatch(n.node, property, event)
}

// Click invokes the current node's `onclick` handler and settles the fixture.
func (n *QueryNode) Click() {
	n.Dispatch("onclick", Event{})
}

// Input updates the current node value, invokes `oninput`, and settles the fixture.
func (n *QueryNode) Input(value string) {
	n.Dispatch("oninput", Event{Value: value})
}

// Change updates the current node value, invokes `onchange`, and settles the fixture.
func (n *QueryNode) Change(value string) {
	n.Dispatch("onchange", Event{Value: value})
}

// Submit invokes the current node's `onsubmit` handler and settles the fixture.
func (n *QueryNode) Submit() {
	n.Dispatch("onsubmit", Event{})
}

func (f *Fixture) requireActive() {
	if f == nil || f.cleaned || f.container == nil || f.scheduler == nil {
		f.tb.Fatal("render fixture is no longer active")
	}
}

func (f *Fixture) wrap(node *mockdom.MockDOMNode) *QueryNode {
	if node == nil {
		return nil
	}
	return &QueryNode{fixture: f, node: node}
}

func (f *Fixture) dispatch(node *mockdom.MockDOMNode, property string, event Event) {
	f.tb.Helper()
	f.requireActive()
	if node == nil {
		f.tb.Fatalf("render fixture cannot dispatch %q on a nil node", property)
	}
	handler := node.Props[property]
	if handler == nil {
		f.tb.Fatalf("render fixture node %q does not expose handler property %q", node.Attrs["id"], property)
	}
	event.apply(node)
	synthetic := event.syntheticValue()
	syntheticGoEvent := runtime.NewGoEvent(synthetic)

	switch typed := handler.(type) {
	case func():
		typed()
	case func(string):
		typed(event.Value)
	case func(js.Value):
		typed(synthetic)
	case func(js.Value) error:
		if err := typed(synthetic); err != nil {
			f.tb.Fatalf("render fixture handler %q returned error: %v", property, err)
		}
	case func(runtime.GoEvent):
		typed(syntheticGoEvent)
	case func(runtime.GoEvent) error:
		if err := typed(syntheticGoEvent); err != nil {
			f.tb.Fatalf("render fixture handler %q returned error: %v", property, err)
		}
	case func() error:
		if err := typed(); err != nil {
			f.tb.Fatalf("render fixture handler %q returned error: %v", property, err)
		}
	default:
		f.tb.Fatalf("render fixture does not know how to dispatch handler property %q with type %T", property, handler)
	}
	if property == "onchange" || property == "oninput" || property == "onsubmit" || property == "onclick" {
		f.Stabilize()
	}
}

func findNode(node *mockdom.MockDOMNode, match func(*mockdom.MockDOMNode) bool) *mockdom.MockDOMNode {
	if node == nil {
		return nil
	}
	if match(node) {
		return node
	}
	for _, child := range node.Children {
		if found := findNode(child, match); found != nil {
			return found
		}
	}
	return nil
}

func collectNodes(node *mockdom.MockDOMNode, match func(*mockdom.MockDOMNode) bool) []*mockdom.MockDOMNode {
	if node == nil {
		return nil
	}
	result := make([]*mockdom.MockDOMNode, 0)
	if match(node) {
		result = append(result, node)
	}
	for _, child := range node.Children {
		result = append(result, collectNodes(child, match)...)
	}
	return result
}

func nodeText(node *mockdom.MockDOMNode) string {
	if node == nil {
		return ""
	}
	if node.Tag == "#text" {
		return node.TextContent
	}
	parts := make([]string, 0, len(node.Children)+1)
	if strings.TrimSpace(node.TextContent) != "" {
		parts = append(parts, strings.TrimSpace(node.TextContent))
	}
	for _, child := range node.Children {
		text := strings.TrimSpace(nodeText(child))
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func nodeRole(node *mockdom.MockDOMNode) string {
	if node == nil {
		return ""
	}
	if role := strings.TrimSpace(node.Attrs["role"]); role != "" {
		return role
	}
	switch strings.ToLower(strings.TrimSpace(node.Tag)) {
	case "button":
		return "button"
	case "a":
		if strings.TrimSpace(node.Attrs["href"]) != "" {
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
		switch strings.ToLower(strings.TrimSpace(node.Attrs["type"])) {
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

func accessibleName(root *mockdom.MockDOMNode, node *mockdom.MockDOMNode) string {
	if node == nil {
		return ""
	}
	if label := normalizeText(node.Attrs["aria-label"]); label != "" {
		return label
	}
	if refs := normalizeText(node.Attrs["aria-labelledby"]); refs != "" {
		parts := make([]string, 0)
		for _, ref := range strings.Fields(refs) {
			if target := findNode(root, func(candidate *mockdom.MockDOMNode) bool {
				return candidate != nil && candidate.Attrs["id"] == ref
			}); target != nil {
				text := normalizeText(nodeText(target))
				if text != "" {
					parts = append(parts, text)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	}
	return normalizeText(nodeText(node))
}

// accessibleDescription resolves the accessible description for one node.
func accessibleDescription(root *mockdom.MockDOMNode, node *mockdom.MockDOMNode) string {
	if node == nil {
		return ""
	}
	if description := normalizeText(node.Attrs["aria-description"]); description != "" {
		return description
	}
	if refs := normalizeText(node.Attrs["aria-describedby"]); refs != "" {
		parseParts := make([]string, 0)
		for _, parseRef := range strings.Fields(refs) {
			parseTarget := findNode(root, func(parseCandidate *mockdom.MockDOMNode) bool {
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
func nodeLivePoliteness(node *mockdom.MockDOMNode) string {
	if node == nil {
		return ""
	}
	if parseLive := normalizeText(node.Attrs["aria-live"]); parseLive != "" {
		if parseLive == "off" {
			return ""
		}
		return parseLive
	}
	switch normalizeText(nodeRole(node)) {
	case "status", "log":
		return "polite"
	case "alert":
		return "assertive"
	}
	return ""
}

func (e Event) apply(node *mockdom.MockDOMNode) {
	if node == nil {
		return
	}
	if e.Value != "" || node.Tag == "input" || node.Tag == "textarea" || node.Tag == "select" {
		node.Attrs["value"] = e.Value
		node.Props["value"] = e.Value
	}
	node.Props["checked"] = e.Checked
	if e.Checked {
		node.Attrs["checked"] = ""
	} else {
		delete(node.Attrs, "checked")
	}
}

func (e Event) syntheticValue() js.Value {
	object := js.Global().Get("Object")
	target := object.New()
	target.Set("value", e.Value)
	target.Set("checked", e.Checked)
	event := object.New()
	event.Set("target", target)
	event.Set("key", e.Key)
	event.Set("keyCode", e.KeyCode)
	return event
}

// buildOverlayBool parses one overlay boolean attribute value.
func buildOverlayBool(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

// buildOverlayInt parses one overlay integer attribute value.
func buildOverlayInt(value string) int {
	parseParsed, parseErr := strconv.Atoi(strings.TrimSpace(value))
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
func cloneSignalFields(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

// applyRenderCountSignal resolves one render-count signal by component name/path.
func applyRenderCountSignal(signals []RenderCountSignal, component string) RenderCountSignal {
	parseComponent := strings.TrimSpace(component)
	if len(signals) == 0 {
		return RenderCountSignal{}
	}
	if parseComponent == "" {
		return signals[0]
	}
	for _, parseSignal := range signals {
		if parseSignal.Name == parseComponent || parseSignal.Path == parseComponent || strings.Contains(parseSignal.Path, parseComponent) {
			return parseSignal
		}
	}
	return RenderCountSignal{}
}

// applyFixtureOwnership claims exclusive fixture ownership for this test process.
func applyFixtureOwnership(tb testing.TB) {
	tb.Helper()
	select {
	case fixtureGate <- struct{}{}:
	default:
		tb.Fatalf("render fixture ownership contention: %s", ParallelSafetyContract())
	}
}

// clearFixtureOwnership releases exclusive fixture ownership for this test process.
func clearFixtureOwnership() {
	select {
	case <-fixtureGate:
	default:
	}
}
