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

func (f *Fixture) Render(root interface{})                              {}
func (f *Fixture) Rerender(root interface{})                            {}
func (f *Fixture) Flush()                                               {}
func (f *Fixture) FlushTimers()                                         {}
func (f *Fixture) Stabilize()                                           {}
func (f *Fixture) Cleanup()                                             {}
func (f *Fixture) Container() *QueryNode                                { return nil }
func (f *Fixture) Target() any                                          { return nil }
func (f *Fixture) ByRole(role string, name string) *QueryNode           { return nil }
func (f *Fixture) AllByRole(role string) []*QueryNode                   { return nil }
func (f *Fixture) ByID(id string) *QueryNode                            { return nil }
func (f *Fixture) ByText(text string) *QueryNode                        { return nil }
func (f *Fixture) AllByTag(tag string) []*QueryNode                     { return nil }
func (f *Fixture) Text() string                                         { return "" }
func (f *Fixture) DispatchByID(id string, property string, event Event) {}
func (f *Fixture) ClickByID(id string)                                  {}
func (f *Fixture) InputByID(id string, value string)                    {}
func (f *Fixture) ChangeByID(id string, value string)                   {}
func (f *Fixture) SubmitByID(id string)                                 {}

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
