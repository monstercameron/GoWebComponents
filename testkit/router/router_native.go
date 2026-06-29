//go:build !js || !wasm

package routertest

import (
	"net/url"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/testkit/render"
)

// Fixture wraps one router plus a rendered route fixture.
type Fixture struct{}

// Inspection mirrors the route details exposed by the wasm fixture.
type Inspection struct {
	Path   string
	Query  url.Values
	Params map[string]string
}

var nativeRouterFatal = func(parseTb testing.TB) {
	parseTb.Helper()
	parseTb.Fatal("testkit/router requires js/wasm tests; run go test with a js/wasm executor such as .\\tools\\go_js_wasm_exec.bat on Windows")
}

func NewHash(parseTb testing.TB, parseOptions ...any) *Fixture {
	parseTb.Helper()
	nativeRouterFatal(parseTb)
	return nil
}

func NewHistory(parseTb testing.TB, parseOptions ...any) *Fixture {
	parseTb.Helper()
	nativeRouterFatal(parseTb)
	return nil
}

func (parseF *Fixture) Register(parsePath string, parseComponent any, parseOptions ...any) {
}
func (parseF *Fixture) SetPath(parsePath string)                                    {}
func (parseF *Fixture) Render()                                                     {}
func (parseF *Fixture) Navigate(parsePath string)                                   {}
func (parseF *Fixture) Replace(parsePath string)                                    {}
func (parseF *Fixture) Inspect() Inspection                                         { return Inspection{} }
func (parseF *Fixture) Path() string                                                { return "" }
func (parseF *Fixture) Query() url.Values                                           { return nil }
func (parseF *Fixture) Params() map[string]string                                   { return nil }
func (parseF *Fixture) Router() any                                                 { return nil }
func (parseF *Fixture) ByID(parseId string) *render.QueryNode                       { return nil }
func (parseF *Fixture) ByText(parseText string) *render.QueryNode                   { return nil }
func (parseF *Fixture) ByRole(parseRole string, parseName string) *render.QueryNode { return nil }
func (parseF *Fixture) AllByRole(parseRole string) []*render.QueryNode              { return nil }
func (parseF *Fixture) ByLabel(parseLabel string) *render.QueryNode                 { return nil }
func (parseF *Fixture) ByDescription(parseDescription string) *render.QueryNode     { return nil }
func (parseF *Fixture) ByLiveRegion(parsePoliteness string, parseText string) *render.QueryNode {
	return nil
}
func (parseF *Fixture) ApplyByRole(parseRole string, parseName string) *render.QueryNode { return nil }
func (parseF *Fixture) ApplyByLabel(parseLabel string) *render.QueryNode                 { return nil }
func (parseF *Fixture) ApplyByDescription(parseDescription string) *render.QueryNode     { return nil }
func (parseF *Fixture) ApplyByLiveRegion(parsePoliteness string, parseText string) *render.QueryNode {
	return nil
}
func (parseF *Fixture) DispatchByID(parseId string, parseProperty string, parseEvent render.Event) {}
func (parseF *Fixture) ClickByID(parseId string)                                                   {}
func (parseF *Fixture) InputByID(parseId string, parseValue string)                                {}
func (parseF *Fixture) ChangeByID(parseId string, parseValue string)                               {}
func (parseF *Fixture) SubmitByID(parseId string)                                                  {}
func (parseF *Fixture) Text() string                                                               { return "" }
func (parseF *Fixture) Cleanup()                                                                   {}
