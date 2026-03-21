//go:build js && wasm
// +build js,wasm

package routertest

import (
	"net/url"
	"strings"
	"testing"

	"syscall/js"

	appRouter "github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/testkit/render"
)

type browserEnv struct {
	prevWindow   js.Value
	prevLocation js.Value
	prevHistory  js.Value
	release      []js.Func
}

// Fixture wraps one router plus a rendered route fixture.
type Fixture struct {
	tb      testing.TB
	router  *appRouter.Router
	render  *render.Fixture
	env     *browserEnv
	isHash  bool
	cleaned bool
}

// NewHash creates a hash-router test fixture.
func NewHash(tb testing.TB, options ...appRouter.RouterOptions) *Fixture {
	tb.Helper()
	fixture := &Fixture{
		tb:     tb,
		env:    installBrowserEnv(tb),
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
		env:    installBrowserEnv(tb),
		render: render.New(tb),
		isHash: false,
	}
	if len(options) > 0 {
		fixture.router = appRouter.NewRouter(options[0])
	} else {
		fixture.router = appRouter.NewRouter(appRouter.RouterOptions{})
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
	f.env.setPath(path, f.isHash)
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
		f.env.setPath(path, false)
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
		f.env.setPath(path, false)
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
		f.env.restore()
		f.env = nil
	}
	f.router = nil
}

func (f *Fixture) requireActive() {
	if f == nil || f.cleaned || f.router == nil || f.render == nil || f.env == nil {
		f.tb.Fatal("router fixture is no longer active")
	}
}

func installBrowserEnv(tb testing.TB) *browserEnv {
	tb.Helper()
	global := js.Global()
	objectCtor := global.Get("Object")
	prevWindow := global.Get("window")
	prevLocation := global.Get("location")
	prevHistory := global.Get("history")

	location := objectCtor.New()
	location.Set("hash", "#/")
	location.Set("pathname", "/")
	location.Set("search", "")

	replaceFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		target := args[0].String()
		if strings.HasPrefix(target, "#") {
			location.Set("hash", target)
			return nil
		}
		pathname, search := splitPathSearch(target)
		location.Set("pathname", pathname)
		location.Set("search", search)
		return nil
	})
	location.Set("replace", replaceFn)

	history := objectCtor.New()
	pushStateFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 3 {
			return nil
		}
		pathname, search := splitPathSearch(args[2].String())
		location.Set("pathname", pathname)
		location.Set("search", search)
		return nil
	})
	replaceStateFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 3 {
			return nil
		}
		pathname, search := splitPathSearch(args[2].String())
		location.Set("pathname", pathname)
		location.Set("search", search)
		return nil
	})
	history.Set("pushState", pushStateFn)
	history.Set("replaceState", replaceStateFn)

	window := objectCtor.New()
	addEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	removeEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	window.Set("addEventListener", addEventListenerFn)
	window.Set("removeEventListener", removeEventListenerFn)
	window.Set("location", location)
	window.Set("history", history)

	global.Set("window", window)
	global.Set("location", location)
	global.Set("history", history)

	env := &browserEnv{
		prevWindow:   prevWindow,
		prevLocation: prevLocation,
		prevHistory:  prevHistory,
		release:      []js.Func{replaceFn, pushStateFn, replaceStateFn, addEventListenerFn, removeEventListenerFn},
	}
	tb.Cleanup(func() {
		env.restore()
	})
	return env
}

func (e *browserEnv) restore() {
	if e == nil {
		return
	}
	global := js.Global()
	if e.prevWindow.Truthy() || !e.prevWindow.IsUndefined() {
		global.Set("window", e.prevWindow)
	}
	if e.prevLocation.Truthy() || !e.prevLocation.IsUndefined() {
		global.Set("location", e.prevLocation)
	}
	if e.prevHistory.Truthy() || !e.prevHistory.IsUndefined() {
		global.Set("history", e.prevHistory)
	}
	for _, fn := range e.release {
		fn.Release()
	}
	e.release = nil
}

func (e *browserEnv) setPath(path string, isHash bool) {
	if e == nil {
		return
	}
	location := js.Global().Get("location")
	if isHash {
		location.Set("hash", "#"+strings.TrimPrefix(path, "#"))
		return
	}
	pathname, search := splitPathSearch(path)
	location.Set("pathname", pathname)
	location.Set("search", search)
}

func splitPathSearch(target string) (string, string) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return "/", ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + strings.TrimPrefix(trimmed, "#")
	}
	if index := strings.Index(trimmed, "?"); index >= 0 {
		return trimmed[:index], trimmed[index:]
	}
	return trimmed, ""
}
