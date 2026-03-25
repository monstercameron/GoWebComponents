//go:build js && wasm
// +build js,wasm

package routertest

import (
	"net/url"
	"testing"

	appRouter "github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/test/browser"
	"github.com/monstercameron/GoWebComponents/testkit/render"
)

// Fixture wraps one router plus a rendered route fixture.
type Fixture struct {
	tb      testing.TB
	router  *appRouter.Router
	render  *render.Fixture
	env     *browser.Environment
	isHash  bool
	cleaned bool
}

// NewHash creates a hash-router test fixture.
func NewHash(tb testing.TB, options ...appRouter.RouterOptions) *Fixture {
	tb.Helper()
	fixture := &Fixture{
		tb:     tb,
		env:    browser.Install(tb, browser.Options{HashRouting: true}),
		render: render.New(tb),
		isHash: true,
	}
	fixture.router = appRouter.NewHashRouter(options...)
	tb.Cleanup(func() {
		fixture.Cleanup()
	})
	return fixture
}

// NewHistory creates a history-router test fixture.
func NewHistory(tb testing.TB, options ...appRouter.RouterOptions) *Fixture {
	tb.Helper()
	fixture := &Fixture{
		tb:     tb,
		env:    browser.Install(tb),
		render: render.New(tb),
		isHash: false,
	}
	if len(options) > 0 {
		fixture.router = appRouter.NewHistoryRouter(options[0])
	} else {
		fixture.router = appRouter.NewHistoryRouter(appRouter.RouterOptions{})
	}
	tb.Cleanup(func() {
		fixture.Cleanup()
	})
	return fixture
}

// Register adds a route to the underlying router.
func (f *Fixture) Register(path string, component interface{}, options ...appRouter.Options) {
	f.requireActive()
	f.router.Register(path, component, options...)
}

// SetPath updates the current browser location without treating it as a navigation action.
func (f *Fixture) SetPath(path string) {
	f.requireActive()
	f.env.SetPath(path, f.isHash)
	if f.router != nil {
		f.Render()
	}
}

// Render renders the current route into the delegated render fixture.
func (f *Fixture) Render() {
	f.tb.Helper()
	f.requireActive()
	f.render.Render(f.router.Current())
}

// Navigate performs router navigation and rerenders the current route.
func (f *Fixture) Navigate(path string) {
	f.tb.Helper()
	f.requireActive()
	if f.isHash {
		f.router.Navigate(path)
	} else {
		f.env.SetPath(path, false)
	}
	f.Render()
}

// Replace performs replace-style router navigation and rerenders the current route.
func (f *Fixture) Replace(path string) {
	f.tb.Helper()
	f.requireActive()
	if f.isHash {
		f.router.NavigateReplace(path)
	} else {
		f.env.SetPath(path, false)
	}
	f.Render()
}

// Inspect returns the current public route inspection snapshot.
func (f *Fixture) Inspect() appRouter.RouteInspection {
	f.requireActive()
	f.router.Current()
	return appRouter.InspectCurrentRoute()
}

// Path returns the current route path.
func (f *Fixture) Path() string {
	return f.Inspect().Path
}

// Query returns the current query values.
func (f *Fixture) Query() url.Values {
	return f.Inspect().Query
}

// Params returns the current route params.
func (f *Fixture) Params() map[string]string {
	return f.Inspect().Params
}

// Router exposes the underlying router for advanced assertions.
func (f *Fixture) Router() *appRouter.Router {
	return f.router
}

// ByID delegates to the rendered route fixture.
func (f *Fixture) ByID(id string) *render.QueryNode {
	if f == nil || f.render == nil {
		return nil
	}
	return f.render.ByID(id)
}

// ByText delegates to the rendered route fixture.
func (f *Fixture) ByText(text string) *render.QueryNode {
	if f == nil || f.render == nil {
		return nil
	}
	return f.render.ByText(text)
}

// ByRole delegates role-first route queries to the rendered route fixture.
func (f *Fixture) ByRole(role string, name string) *render.QueryNode {
	if f == nil || f.render == nil {
		return nil
	}
	return f.render.ByRole(role, name)
}

// AllByRole delegates role collection queries to the rendered route fixture.
func (f *Fixture) AllByRole(role string) []*render.QueryNode {
	if f == nil || f.render == nil {
		return nil
	}
	return f.render.AllByRole(role)
}

// ByLabel delegates accessibility label queries to the rendered route fixture.
func (f *Fixture) ByLabel(label string) *render.QueryNode {
	if f == nil || f.render == nil {
		return nil
	}
	return f.render.ByLabel(label)
}

// ByDescription delegates accessibility description queries to the rendered route fixture.
func (f *Fixture) ByDescription(description string) *render.QueryNode {
	if f == nil || f.render == nil {
		return nil
	}
	return f.render.ByDescription(description)
}

// ByLiveRegion delegates live-region queries to the rendered route fixture.
func (f *Fixture) ByLiveRegion(politeness string, text string) *render.QueryNode {
	if f == nil || f.render == nil {
		return nil
	}
	return f.render.ByLiveRegion(politeness, text)
}

// ApplyByRole delegates role assertions to the rendered route fixture.
func (f *Fixture) ApplyByRole(role string, name string) *render.QueryNode {
	if f == nil || f.render == nil {
		return nil
	}
	return f.render.ApplyByRole(role, name)
}

// ApplyByLabel delegates label assertions to the rendered route fixture.
func (f *Fixture) ApplyByLabel(label string) *render.QueryNode {
	if f == nil || f.render == nil {
		return nil
	}
	return f.render.ApplyByLabel(label)
}

// ApplyByDescription delegates description assertions to the rendered route fixture.
func (f *Fixture) ApplyByDescription(description string) *render.QueryNode {
	if f == nil || f.render == nil {
		return nil
	}
	return f.render.ApplyByDescription(description)
}

// ApplyByLiveRegion delegates live-region assertions to the rendered route fixture.
func (f *Fixture) ApplyByLiveRegion(politeness string, text string) *render.QueryNode {
	if f == nil || f.render == nil {
		return nil
	}
	return f.render.ApplyByLiveRegion(politeness, text)
}

// DispatchByID delegates synthetic event dispatch to the rendered route fixture.
func (f *Fixture) DispatchByID(id string, property string, event render.Event) {
	if f == nil || f.render == nil {
		return
	}
	f.render.DispatchByID(id, property, event)
}

// ClickByID delegates click dispatch to the rendered route fixture.
func (f *Fixture) ClickByID(id string) {
	if f == nil || f.render == nil {
		return
	}
	f.render.ClickByID(id)
}

// InputByID delegates input dispatch to the rendered route fixture.
func (f *Fixture) InputByID(id string, value string) {
	if f == nil || f.render == nil {
		return
	}
	f.render.InputByID(id, value)
}

// ChangeByID delegates change dispatch to the rendered route fixture.
func (f *Fixture) ChangeByID(id string, value string) {
	if f == nil || f.render == nil {
		return
	}
	f.render.ChangeByID(id, value)
}

// SubmitByID delegates submit dispatch to the rendered route fixture.
func (f *Fixture) SubmitByID(id string) {
	if f == nil || f.render == nil {
		return
	}
	f.render.SubmitByID(id)
}

// Text returns the full rendered route text.
func (f *Fixture) Text() string {
	if f == nil || f.render == nil {
		return ""
	}
	return f.render.Text()
}

// Cleanup releases the delegated fixture and restores browser globals.
func (f *Fixture) Cleanup() {
	if f == nil || f.cleaned {
		return
	}
	f.cleaned = true
	if f.render != nil {
		f.render.Cleanup()
		f.render = nil
	}
	if f.env != nil {
		f.env.Restore()
		f.env = nil
	}
	f.router = nil
}

func (f *Fixture) requireActive() {
	if f == nil || f.cleaned || f.router == nil || f.render == nil || f.env == nil {
		f.tb.Fatal("router fixture is no longer active")
	}
}
