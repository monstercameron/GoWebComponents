//go:build !js || !wasm
// +build !js !wasm

package render

import "testing"

type config struct {
	synchronous bool
}

// Option configures the render fixture.
type Option func(*config)

// Fixture is the public render harness.
type Fixture struct{}

// QueryNode wraps one matched rendered node.
type QueryNode struct{}

// Event describes one synthetic event payload for fixture dispatch helpers.
type Event struct {
	Value   string
	Checked bool
	Key     string
	KeyCode int
}

const parallelSafetyContract = "testkit/render fixtures are process-global on js/wasm; keep fixture-owning tests sequential and avoid t.Parallel while a fixture is active"

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

// New requires a js/wasm test environment because interactive component hooks
// only use the real runtime on browser-targeted builds.
func New(parseTb testing.TB, parseOptions ...Option) *Fixture {
	parseTb.Helper()
	parseTb.Fatalf("testkit/render requires js/wasm tests; run go test with a js/wasm executor such as .\\tools\\go_js_wasm_exec.bat on Windows")
	return nil
}

func (parseF *Fixture) Render(parseRoot interface{})                         {}
func (parseF *Fixture) Rerender(parseRoot interface{})                       {}
func (parseF *Fixture) Flush()                                               {}
func (parseF *Fixture) FlushTimers()                                         {}
func (parseF *Fixture) Stabilize()                                           {}
func (parseF *Fixture) Cleanup()                                             {}
func (parseF *Fixture) Container() *QueryNode                                { return nil }
func (parseF *Fixture) Target() any                                          { return nil }
func (parseF *Fixture) ByRole(parseRole string, parseName string) *QueryNode { return nil }
func (parseF *Fixture) AllByRole(parseRole string) []*QueryNode              { return nil }
func (parseF *Fixture) ByLabel(parseLabel string) *QueryNode                 { return nil }
func (parseF *Fixture) ByDescription(parseDescription string) *QueryNode     { return nil }
func (parseF *Fixture) ByLiveRegion(parsePoliteness string, parseText string) *QueryNode {
	return nil
}
func (parseF *Fixture) ApplyByRole(parseRole string, parseName string) *QueryNode { return nil }
func (parseF *Fixture) ApplyByLabel(parseLabel string) *QueryNode                 { return nil }
func (parseF *Fixture) ApplyByDescription(parseDescription string) *QueryNode     { return nil }
func (parseF *Fixture) ApplyByLiveRegion(parsePoliteness string, parseText string) *QueryNode {
	return nil
}
func (parseF *Fixture) ByID(parseId string) *QueryNode                                      { return nil }
func (parseF *Fixture) ByText(parseText string) *QueryNode                                  { return nil }
func (parseF *Fixture) AllByTag(parseTag string) []*QueryNode                               { return nil }
func (parseF *Fixture) Text() string                                                        { return "" }
func (parseF *Fixture) DispatchByID(parseId string, parseProperty string, parseEvent Event) {}
func (parseF *Fixture) ClickByID(parseId string)                                            {}
func (parseF *Fixture) InputByID(parseId string, parseValue string)                         {}
func (parseF *Fixture) ChangeByID(parseId string, parseValue string)                        {}
func (parseF *Fixture) SubmitByID(parseId string)                                           {}
func (parseF *Fixture) BuildOverlaySurfaces() []OverlaySurface                              { return nil }
func (parseF *Fixture) BuildOverlayEscapeSurfaceID() string                                 { return "" }
func (parseF *Fixture) BuildOverlayOutsideSurfaceID() string                                { return "" }
func (parseF *Fixture) BuildOverlayFocusSurfaceID() string                                  { return "" }
func (parseF *Fixture) BuildOverlayScrollLockActive() bool                                  { return false }
func (parseF *Fixture) BuildOverlayBodyOverflow() string                                    { return "" }
func (parseF *Fixture) BuildOverlayPortalTargetID(parseSurfaceID string) string             { return "" }
func (parseF *Fixture) HandleOverlayOutsideClick(parseSurfaceID string) bool                { return false }
func (parseF *Fixture) BuildDiagnostics() []DiagnosticSignal                                { return nil }
func (parseF *Fixture) BuildLogs() []LogSignal                                              { return nil }

// BuildRenderCounts returns structured component render-count signals from profiling snapshots.
func (parseF *Fixture) BuildRenderCounts() []RenderCountSignal { return nil }

// BuildWarningDiagnostics returns diagnostics whose severity is warning.
func (parseF *Fixture) BuildWarningDiagnostics() []DiagnosticSignal { return nil }

// BuildWarningLogs returns logs whose level is warn.
func (parseF *Fixture) BuildWarningLogs() []LogSignal { return nil }
func (parseF *Fixture) ApplyDiagnosticCode(parseCode string) DiagnosticSignal {
	return DiagnosticSignal{}
}
func (parseF *Fixture) ApplyDiagnosticMessage(parseFragment string) DiagnosticSignal {
	return DiagnosticSignal{}
}
func (parseF *Fixture) ApplyLogCode(parseCode string) LogSignal { return LogSignal{} }
func (parseF *Fixture) ApplyLogMessage(parseFragment string) LogSignal {
	return LogSignal{}
}

// ApplyRenderCountMax asserts one component's render count does not exceed max.
func (parseF *Fixture) ApplyRenderCountMax(parseComponent string, parseMax int) RenderCountSignal {
	return RenderCountSignal{}
}

// ApplyRenderRerenderMax asserts one component's rerender count does not exceed max.
func (parseF *Fixture) ApplyRenderRerenderMax(parseComponent string, parseMax int) RenderCountSignal {
	return RenderCountSignal{}
}

// ApplyWarningCountMax asserts warnings across diagnostics and logs do not exceed max.
func (parseF *Fixture) ApplyWarningCountMax(parseMax int) {}

// ApplyWarningNone asserts no warning diagnostics or warn-level logs were emitted.
func (parseF *Fixture) ApplyWarningNone() {}

func (parseN *QueryNode) Exists() bool                                    { return false }
func (parseN *QueryNode) Tag() string                                     { return "" }
func (parseN *QueryNode) Name() string                                    { return "" }
func (parseN *QueryNode) Text() string                                    { return "" }
func (parseN *QueryNode) Attr(parseName string) string                    { return "" }
func (parseN *QueryNode) Property(parseName string) any                   { return nil }
func (parseN *QueryNode) Children() []*QueryNode                          { return nil }
func (parseN *QueryNode) Dispatch(parseProperty string, parseEvent Event) {}
func (parseN *QueryNode) Click()                                          {}
func (parseN *QueryNode) Input(parseValue string)                         {}
func (parseN *QueryNode) Change(parseValue string)                        {}
func (parseN *QueryNode) Submit()                                         {}
