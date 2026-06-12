//go:build js && wasm

package routertest

import (
	"net/url"
	"testing"

	appRouter "github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/test/browser"
	"github.com/monstercameron/GoWebComponents/testkit/render"
	"github.com/monstercameron/GoWebComponents/ui"
)

// Fixture wraps one router plus a rendered route fixture.
type Fixture struct {
	tb                     testing.TB
	router                 *appRouter.Router
	render                 *render.Fixture
	env                    *browser.Environment
	isHash                 bool
	cleaned                bool
	renderCurrentComponent func() ui.Node
	renderVersion          int
}

// NewHash creates a hash-router test fixture.
func NewHash(parseTb testing.TB, parseOptions ...appRouter.RouterOptions) *Fixture {
	parseTb.Helper()
	parseFixture := &Fixture{
		tb:     parseTb,
		env:    browser.Install(parseTb, browser.Options{HashRouting: true}),
		render: render.New(parseTb),
		isHash: true,
	}
	parseFixture.router = appRouter.NewHashRouter(parseOptions...)
	parseFixture.renderCurrentComponent = func() ui.Node {
		return parseFixture.router.Current()
	}
	parseTb.Cleanup(func() {
		parseFixture.Cleanup()
	})
	return parseFixture
}

// NewHistory creates a history-router test fixture.
func NewHistory(parseTb testing.TB, parseOptions ...appRouter.RouterOptions) *Fixture {
	parseTb.Helper()
	parseFixture := &Fixture{
		tb:     parseTb,
		env:    browser.Install(parseTb),
		render: render.New(parseTb),
		isHash: false,
	}
	if len(parseOptions) > 0 {
		parseFixture.router = appRouter.NewHistoryRouter(parseOptions[0])
	} else {
		parseFixture.router = appRouter.NewHistoryRouter(appRouter.RouterOptions{})
	}
	parseFixture.renderCurrentComponent = func() ui.Node {
		return parseFixture.router.Current()
	}
	parseTb.Cleanup(func() {
		parseFixture.Cleanup()
	})
	return parseFixture
}

// Register adds a route to the underlying router.
func (parseF *Fixture) Register(parsePath string, parseComponent interface{}, parseOptions ...appRouter.Options) {
	parseF.requireActive()
	parseF.router.Register(parsePath, parseComponent, parseOptions...)
}

// SetPath updates the current browser location without treating it as a navigation action.
func (parseF *Fixture) SetPath(parsePath string) {
	parseF.requireActive()
	parseF.env.SetPath(parsePath, parseF.isHash)
	if parseF.router != nil {
		parseF.Render()
	}
}

// Render renders the current route into the delegated render fixture.
func (parseF *Fixture) Render() {
	parseF.tb.Helper()
	parseF.requireActive()
	parseF.render.Render(ui.CreateElement(parseF.renderCurrentComponent, parseF.renderVersion))
	parseF.renderVersion++
}

// Navigate performs router navigation and rerenders the current route.
func (parseF *Fixture) Navigate(parsePath string) {
	parseF.tb.Helper()
	parseF.requireActive()
	if parseF.isHash {
		parseF.router.Navigate(parsePath)
	} else {
		parseF.env.SetPath(parsePath, false)
	}
	parseF.Render()
}

// Replace performs replace-style router navigation and rerenders the current route.
func (parseF *Fixture) Replace(parsePath string) {
	parseF.tb.Helper()
	parseF.requireActive()
	if parseF.isHash {
		parseF.router.NavigateReplace(parsePath)
	} else {
		parseF.env.SetPath(parsePath, false)
	}
	parseF.Render()
}

// Inspect returns the current public route inspection snapshot.
func (parseF *Fixture) Inspect() appRouter.RouteInspection {
	parseF.requireActive()
	parseF.router.Current()
	return appRouter.InspectCurrentRoute()
}

// Path returns the current route path.
func (parseF *Fixture) Path() string {
	return parseF.Inspect().Path
}

// Query returns the current query values.
func (parseF *Fixture) Query() url.Values {
	return parseF.Inspect().Query
}

// Params returns the current route params.
func (parseF *Fixture) Params() map[string]string {
	return parseF.Inspect().Params
}

// Router exposes the underlying router for advanced assertions.
func (parseF *Fixture) Router() *appRouter.Router {
	return parseF.router
}

// ByID delegates to the rendered route fixture.
func (parseF *Fixture) ByID(parseId string) *render.QueryNode {
	if parseF == nil || parseF.render == nil {
		return nil
	}
	return parseF.render.ByID(parseId)
}

// ByText delegates to the rendered route fixture.
func (parseF *Fixture) ByText(parseText string) *render.QueryNode {
	if parseF == nil || parseF.render == nil {
		return nil
	}
	return parseF.render.ByText(parseText)
}

// ByRole delegates role-first route queries to the rendered route fixture.
func (parseF *Fixture) ByRole(parseRole string, parseName string) *render.QueryNode {
	if parseF == nil || parseF.render == nil {
		return nil
	}
	return parseF.render.ByRole(parseRole, parseName)
}

// AllByRole delegates role collection queries to the rendered route fixture.
func (parseF *Fixture) AllByRole(parseRole string) []*render.QueryNode {
	if parseF == nil || parseF.render == nil {
		return nil
	}
	return parseF.render.AllByRole(parseRole)
}

// ByLabel delegates accessibility label queries to the rendered route fixture.
func (parseF *Fixture) ByLabel(parseLabel string) *render.QueryNode {
	if parseF == nil || parseF.render == nil {
		return nil
	}
	return parseF.render.ByLabel(parseLabel)
}

// ByDescription delegates accessibility description queries to the rendered route fixture.
func (parseF *Fixture) ByDescription(parseDescription string) *render.QueryNode {
	if parseF == nil || parseF.render == nil {
		return nil
	}
	return parseF.render.ByDescription(parseDescription)
}

// ByLiveRegion delegates live-region queries to the rendered route fixture.
func (parseF *Fixture) ByLiveRegion(parsePoliteness string, parseText string) *render.QueryNode {
	if parseF == nil || parseF.render == nil {
		return nil
	}
	return parseF.render.ByLiveRegion(parsePoliteness, parseText)
}

// ApplyByRole delegates role assertions to the rendered route fixture.
func (parseF *Fixture) ApplyByRole(parseRole string, parseName string) *render.QueryNode {
	if parseF == nil || parseF.render == nil {
		return nil
	}
	return parseF.render.ApplyByRole(parseRole, parseName)
}

// ApplyByLabel delegates label assertions to the rendered route fixture.
func (parseF *Fixture) ApplyByLabel(parseLabel string) *render.QueryNode {
	if parseF == nil || parseF.render == nil {
		return nil
	}
	return parseF.render.ApplyByLabel(parseLabel)
}

// ApplyByDescription delegates description assertions to the rendered route fixture.
func (parseF *Fixture) ApplyByDescription(parseDescription string) *render.QueryNode {
	if parseF == nil || parseF.render == nil {
		return nil
	}
	return parseF.render.ApplyByDescription(parseDescription)
}

// ApplyByLiveRegion delegates live-region assertions to the rendered route fixture.
func (parseF *Fixture) ApplyByLiveRegion(parsePoliteness string, parseText string) *render.QueryNode {
	if parseF == nil || parseF.render == nil {
		return nil
	}
	return parseF.render.ApplyByLiveRegion(parsePoliteness, parseText)
}

// DispatchByID delegates synthetic event dispatch to the rendered route fixture.
func (parseF *Fixture) DispatchByID(parseId string, parseProperty string, parseEvent render.Event) {
	if parseF == nil || parseF.render == nil {
		return
	}
	parseF.render.DispatchByID(parseId, parseProperty, parseEvent)
}

// ClickByID delegates click dispatch to the rendered route fixture.
func (parseF *Fixture) ClickByID(parseId string) {
	if parseF == nil || parseF.render == nil {
		return
	}
	parseF.render.ClickByID(parseId)
}

// InputByID delegates input dispatch to the rendered route fixture.
func (parseF *Fixture) InputByID(parseId string, parseValue string) {
	if parseF == nil || parseF.render == nil {
		return
	}
	parseF.render.InputByID(parseId, parseValue)
}

// ChangeByID delegates change dispatch to the rendered route fixture.
func (parseF *Fixture) ChangeByID(parseId string, parseValue string) {
	if parseF == nil || parseF.render == nil {
		return
	}
	parseF.render.ChangeByID(parseId, parseValue)
}

// SubmitByID delegates submit dispatch to the rendered route fixture.
func (parseF *Fixture) SubmitByID(parseId string) {
	if parseF == nil || parseF.render == nil {
		return
	}
	parseF.render.SubmitByID(parseId)
}

// Text returns the full rendered route text.
func (parseF *Fixture) Text() string {
	if parseF == nil || parseF.render == nil {
		return ""
	}
	return parseF.render.Text()
}

// Cleanup releases the delegated fixture and restores browser globals.
func (parseF *Fixture) Cleanup() {
	if parseF == nil || parseF.cleaned {
		return
	}
	parseF.cleaned = true
	if parseF.render != nil {
		parseF.render.Cleanup()
		parseF.render = nil
	}
	if parseF.env != nil {
		parseF.env.Restore()
		parseF.env = nil
	}
	parseF.router = nil
}

func (parseF *Fixture) requireActive() {
	if parseF == nil || parseF.cleaned || parseF.router == nil || parseF.render == nil || parseF.env == nil {
		parseF.tb.Fatal("router fixture is no longer active")
	}
}
