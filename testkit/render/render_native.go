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

// New requires a js/wasm test environment because interactive component hooks
// only use the real runtime on browser-targeted builds.
func New(tb testing.TB, options ...Option) *Fixture {
	tb.Helper()
	tb.Fatalf("testkit/render requires js/wasm tests; run go test with a js/wasm executor such as .\\tools\\go_js_wasm_exec.bat on Windows")
	return nil
}

func (f *Fixture) Render(root interface{})                     {}
func (f *Fixture) Rerender(root interface{})                   {}
func (f *Fixture) Flush()                                      {}
func (f *Fixture) FlushTimers()                                {}
func (f *Fixture) Stabilize()                                  {}
func (f *Fixture) Cleanup()                                    {}
func (f *Fixture) Container() *QueryNode                       { return nil }
func (f *Fixture) Target() any                                 { return nil }
func (f *Fixture) ByRole(role string, name string) *QueryNode  { return nil }
func (f *Fixture) AllByRole(role string) []*QueryNode          { return nil }
func (f *Fixture) ByLabel(label string) *QueryNode             { return nil }
func (f *Fixture) ByDescription(description string) *QueryNode { return nil }
func (f *Fixture) ByLiveRegion(politeness string, text string) *QueryNode {
	return nil
}
func (f *Fixture) ApplyByRole(role string, name string) *QueryNode  { return nil }
func (f *Fixture) ApplyByLabel(label string) *QueryNode             { return nil }
func (f *Fixture) ApplyByDescription(description string) *QueryNode { return nil }
func (f *Fixture) ApplyByLiveRegion(politeness string, text string) *QueryNode {
	return nil
}
func (f *Fixture) ByID(id string) *QueryNode                            { return nil }
func (f *Fixture) ByText(text string) *QueryNode                        { return nil }
func (f *Fixture) AllByTag(tag string) []*QueryNode                     { return nil }
func (f *Fixture) Text() string                                         { return "" }
func (f *Fixture) DispatchByID(id string, property string, event Event) {}
func (f *Fixture) ClickByID(id string)                                  {}
func (f *Fixture) InputByID(id string, value string)                    {}
func (f *Fixture) ChangeByID(id string, value string)                   {}
func (f *Fixture) SubmitByID(id string)                                 {}
func (f *Fixture) BuildOverlaySurfaces() []OverlaySurface               { return nil }
func (f *Fixture) BuildOverlayEscapeSurfaceID() string                  { return "" }
func (f *Fixture) BuildOverlayOutsideSurfaceID() string                 { return "" }
func (f *Fixture) BuildOverlayFocusSurfaceID() string                   { return "" }
func (f *Fixture) BuildOverlayScrollLockActive() bool                   { return false }
func (f *Fixture) BuildOverlayBodyOverflow() string                     { return "" }
func (f *Fixture) BuildOverlayPortalTargetID(surfaceID string) string   { return "" }
func (f *Fixture) HandleOverlayOutsideClick(surfaceID string) bool      { return false }
func (f *Fixture) BuildDiagnostics() []DiagnosticSignal                 { return nil }
func (f *Fixture) BuildLogs() []LogSignal                               { return nil }
func (f *Fixture) ApplyDiagnosticCode(code string) DiagnosticSignal     { return DiagnosticSignal{} }
func (f *Fixture) ApplyDiagnosticMessage(fragment string) DiagnosticSignal {
	return DiagnosticSignal{}
}
func (f *Fixture) ApplyLogCode(code string) LogSignal { return LogSignal{} }
func (f *Fixture) ApplyLogMessage(fragment string) LogSignal {
	return LogSignal{}
}

func (n *QueryNode) Exists() bool                          { return false }
func (n *QueryNode) Tag() string                           { return "" }
func (n *QueryNode) Name() string                          { return "" }
func (n *QueryNode) Text() string                          { return "" }
func (n *QueryNode) Attr(name string) string               { return "" }
func (n *QueryNode) Property(name string) any              { return nil }
func (n *QueryNode) Children() []*QueryNode                { return nil }
func (n *QueryNode) Dispatch(property string, event Event) {}
func (n *QueryNode) Click()                                {}
func (n *QueryNode) Input(value string)                    {}
func (n *QueryNode) Change(value string)                   {}
func (n *QueryNode) Submit()                               {}
