//go:build !js || !wasm
// +build !js !wasm

package routertest

import (
	"net/url"
	"testing"

	"github.com/monstercameron/GoWebComponents/testkit/render"
)

// Fixture wraps one router plus a rendered route fixture.
type Fixture struct{}

// Inspection mirrors the route details exposed by the wasm fixture.
type Inspection struct {
	Path   string
	Query  url.Values
	Params map[string]string
}

func NewHash(tb testing.TB, options ...interface{}) *Fixture {
	tb.Helper()
	tb.Fatalf("testkit/router requires js/wasm tests; run go test with a js/wasm executor such as .\\tools\\go_js_wasm_exec.bat on Windows")
	return nil
}

func NewHistory(tb testing.TB, options ...interface{}) *Fixture {
	tb.Helper()
	tb.Fatalf("testkit/router requires js/wasm tests; run go test with a js/wasm executor such as .\\tools\\go_js_wasm_exec.bat on Windows")
	return nil
}

func (f *Fixture) Register(path string, component interface{}, options ...interface{}) {}
func (f *Fixture) SetPath(path string)                                                 {}
func (f *Fixture) Render()                                                             {}
func (f *Fixture) Navigate(path string)                                                {}
func (f *Fixture) Replace(path string)                                                 {}
func (f *Fixture) Inspect() Inspection                                                 { return Inspection{} }
func (f *Fixture) Path() string                                                        { return "" }
func (f *Fixture) Query() url.Values                                                   { return nil }
func (f *Fixture) Params() map[string]string                                           { return nil }
func (f *Fixture) Router() any                                                         { return nil }
func (f *Fixture) ByID(id string) *render.QueryNode                                    { return nil }
func (f *Fixture) ByText(text string) *render.QueryNode                                { return nil }
func (f *Fixture) ByRole(role string, name string) *render.QueryNode                   { return nil }
func (f *Fixture) AllByRole(role string) []*render.QueryNode                           { return nil }
func (f *Fixture) ByLabel(label string) *render.QueryNode                              { return nil }
func (f *Fixture) ByDescription(description string) *render.QueryNode                  { return nil }
func (f *Fixture) ByLiveRegion(politeness string, text string) *render.QueryNode       { return nil }
func (f *Fixture) ApplyByRole(role string, name string) *render.QueryNode              { return nil }
func (f *Fixture) ApplyByLabel(label string) *render.QueryNode                         { return nil }
func (f *Fixture) ApplyByDescription(description string) *render.QueryNode             { return nil }
func (f *Fixture) ApplyByLiveRegion(politeness string, text string) *render.QueryNode  { return nil }
func (f *Fixture) DispatchByID(id string, property string, event render.Event)         {}
func (f *Fixture) ClickByID(id string)                                                  {}
func (f *Fixture) InputByID(id string, value string)                                    {}
func (f *Fixture) ChangeByID(id string, value string)                                   {}
func (f *Fixture) SubmitByID(id string)                                                 {}
func (f *Fixture) Text() string                                                        { return "" }
func (f *Fixture) Cleanup()                                                            {}
