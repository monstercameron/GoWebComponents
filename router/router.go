//go:build js && wasm
// +build js,wasm

package router

import (
	"context"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/internal/platform/jsdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// Component is a type alias for component functions used in routing.
type Attrs = map[string]interface{}
type Element = runtime.Element
type Component = func(Attrs) *Element

// Options represents configuration for individual routes.
// Placeholder for future per-route settings (e.g., titles, guards).
type Options struct {
	Title        string
	Redirect     string
	Description  string
	CanonicalURL string
	BeforeEnter  GuardFunc
	BeforeLeave  LeaveGuardFunc
	Loader       LoaderFunc
	Loading      interface{}
	Error        interface{}
}

// RouterOptions configures router defaults.
type RouterOptions struct {
	DefaultRoute string
}

// Router manages routes and navigation for single-page applications.
type Router struct {
	routes         map[string]routeFactory
	routeOptions   map[string]Options
	patterns       []routePattern
	defaultRoute   string
	notFound       routeFactory
	notFoundOption Options
	targetSelector string
	targetElement  js.Value
	listening      bool
	routerType     string // "hash" or "history"
	loaderState    loaderState
	metadataState  routeMetadataState
}

type routeMetadataState struct {
	baseTitleCaptured bool
	baseTitle         string
	titleManaged      bool
}

type Navigator struct {
	navigate func(string)
	replace  func(string)
}

type Revalidator struct {
	revalidate func()
	loading    func() bool
}

type Query struct {
	values url.Values
}

type SearchParams struct {
	values   url.Values
	navigate func(url.Values)
	replace  func(url.Values)
}

type Params struct {
	values map[string]string
}

type RouteInspection struct {
	Path    string
	Query   url.Values
	Params  map[string]string
	Loading bool
}

type RouteContext struct {
	Path   string
	Params Params
	Query  Query
}

type GuardResult struct {
	Redirect string
	Blocked  bool
	Reason   string
}

type GuardFunc func(RouteContext) GuardResult
type LeaveGuardFunc func(current RouteContext, next RouteContext) GuardResult

type resolvedRoute struct {
	path    string
	params  map[string]string
	option  Options
	factory routeFactory
	found   bool
}

type LoaderFunc func(context.Context, RouteContext) (Attrs, error)

type routeFactory func(Attrs) *Element
type routePattern struct {
	pattern string
	factory routeFactory
	options Options
}

type loaderState struct {
	mu      sync.Mutex
	key     string
	pending bool
	data    Attrs
	err     error
	cancel  context.CancelFunc
	version int
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || trimmed == "#" {
		return "/"
	}
	if trimmed == "*" {
		return "*"
	}

	trimmed = strings.TrimPrefix(trimmed, "#")
	if idx := strings.Index(trimmed, "?"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	if len(trimmed) > 1 {
		trimmed = strings.TrimRight(trimmed, "/")
		if trimmed == "" {
			return "/"
		}
	}
	return trimmed
}

func normalizeNavigationTarget(target string) string {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" || trimmed == "#" {
		return "/"
	}

	trimmed = strings.TrimPrefix(trimmed, "#")
	query := ""
	if idx := strings.Index(trimmed, "?"); idx >= 0 {
		query = trimmed[idx:]
		trimmed = trimmed[:idx]
	}

	normalized := normalizePath(trimmed)
	if query == "?" {
		query = ""
	}
	return normalized + query
}

var initialized bool
var currentParams = map[string]string{}
var currentRouteData Attrs

// NewHashRouter creates a hash-based router that reads from window.location.hash.
func NewHashRouter(options ...RouterOptions) *Router {
	cfg := RouterOptions{DefaultRoute: "/"}
	if len(options) > 0 {
		cfg = options[0]
	}
	cfg.DefaultRoute = normalizePath(cfg.DefaultRoute)

	return &Router{
		routes:       make(map[string]routeFactory),
		routeOptions: make(map[string]Options),
		patterns:     []routePattern{},
		defaultRoute: cfg.DefaultRoute,
		routerType:   "hash",
	}
}

// NewRouter creates a history-based router using the HTML5 History API.
// This router uses window.location.pathname instead of hash fragments.
// Requires server to redirect all routes to the app's entry point.
func NewRouter(options RouterOptions) *Router {
	// Use provided options or defaults
	if options.DefaultRoute == "" {
		options.DefaultRoute = "/"
	}
	options.DefaultRoute = normalizePath(options.DefaultRoute)

	router := &Router{
		routes:       make(map[string]routeFactory),
		routeOptions: make(map[string]Options),
		patterns:     []routePattern{},
		defaultRoute: options.DefaultRoute,
		routerType:   "history",
	}

	// Setup browser sync for history-based navigation
	router.setupHistoryListener()

	return router
}

// setupHistoryListener sets up History API listeners for popstate events
func (r *Router) setupHistoryListener() {
	window := js.Global().Get("window")
	if !window.Truthy() || !window.Get("addEventListener").Truthy() {
		return
	}

	// Handler for browser back/forward buttons
	popstateHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		r.renderCurrentRoute()
		return nil
	})

	window.Call("addEventListener", "popstate", popstateHandler)

	// Clean up on unload
	registerCleanup(popstateHandler)
}

// GoRegisterRoute registers a route on the router instance.
func (r *Router) GoRegisterRoute(path string, component interface{}, options ...Options) {
	r.Register(path, component, options...)
}

// Register registers a route using either a component function or a static node.
func (r *Router) Register(path string, component interface{}, options ...Options) {
	if r.routes == nil {
		r.routes = make(map[string]routeFactory)
	}
	if r.routeOptions == nil {
		r.routeOptions = make(map[string]Options)
	}
	if r.patterns == nil {
		r.patterns = []routePattern{}
	}

	normalize := normalizePath(path)
	factory := makeRouteFactory(component)
	option := Options{}
	if len(options) > 0 {
		option = options[0]
	}

	if normalize == "*" {
		if r.notFound != nil {
			runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "replacing existing catch-all route registration for *")
		}
		r.notFound = factory
		r.notFoundOption = option
		return
	}

	if isPatternRoute(normalize) {
		for index, pattern := range r.patterns {
			if pattern.pattern == normalize {
				runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "replacing existing pattern route registration for "+normalize)
				r.patterns[index] = routePattern{pattern: normalize, factory: factory, options: option}
				return
			}
		}
		r.patterns = append(r.patterns, routePattern{pattern: normalize, factory: factory, options: option})
		return
	}

	if _, exists := r.routes[normalize]; exists {
		runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "replacing existing route registration for "+normalize)
	}
	r.routes[normalize] = factory
	r.routeOptions[normalize] = option
}

// GoGetRoute returns the element for the current route.
func (r *Router) GoGetRoute() *Element {
	return r.Current()
}

// Current returns the current route element.
func (r *Router) Current() *Element {
	globalRouter = r
	path := r.GetCurrentRouterPath()
	if path == "" {
		path = r.defaultRoute
	}
	query := getCurrentQueryValues()
	queryKey := query.Encode()
	resolved := r.resolveRoute(path)
	if resolved.found {
		currentParams = copyParams(resolved.params)
		return r.renderResolvedRoute(resolved.path, resolved.factory, resolved.option, resolved.params, query, queryKey)
	}

	r.cancelLoaderIfActive()
	currentRouteData = nil

	return runtime.Div(nil, runtime.Text("Route not found"))
}

// Mount renders the router into a DOM node selected by CSS selector and wires hashchange listeners.
func (r *Router) Mount(selector string) {
	globalRouter = r
	r.targetSelector = selector
	r.targetElement = js.Null()
	r.renderCurrentRoute()
	r.ensureListener()
}

// HydrateMount binds the router to an already-hydrated DOM target and only
// wires future route updates/listeners without forcing an immediate rerender.
func (r *Router) HydrateMount(selector string) {
	globalRouter = r
	r.targetSelector = selector
	r.targetElement = js.Null()
	r.ensureListener()
}

// MountElement renders the router into an existing DOM element reference.
func (r *Router) MountElement(elem js.Value) {
	globalRouter = r
	r.targetElement = elem
	r.targetSelector = ""
	r.renderCurrentRoute()
	r.ensureListener()
}

// HydrateMountElement binds the router to an already-hydrated DOM element and
// only wires future route updates/listeners without forcing an immediate rerender.
func (r *Router) HydrateMountElement(elem js.Value) {
	globalRouter = r
	r.targetElement = elem
	r.targetSelector = ""
	r.ensureListener()
}

// RevalidateCurrentRoute clears the cached result for the current route loader and runs it again.
func (r *Router) RevalidateCurrentRoute() {
	r.cancelLoaderIfActive()
	r.renderCurrentRoute()
}

// IsRouteLoading reports whether the current route loader is pending.
func (r *Router) IsRouteLoading() bool {
	r.loaderState.mu.Lock()
	defer r.loaderState.mu.Unlock()
	return r.loaderState.pending
}

func (r *Router) renderCurrentRoute() {
	ensureInitialized()
	rt := runtime.GetGlobalRuntime()
	routeElement := r.GoGetRoute()
	switch {
	case r.targetSelector != "":
		rt.RenderTo(r.targetSelector, routeElement)
	case r.targetElement.Truthy():
		rt.Render(routeElement, jsdom.NewWASMDOMNode(r.targetElement))
	}
}

func (r *Router) ensureListener() {
	if r.listening {
		return
	}
	window := js.Global().Get("window")
	if !window.Truthy() || !window.Get("addEventListener").Truthy() {
		return
	}
	r.listening = true

	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		r.renderCurrentRoute()
		return nil
	})
	window.Call("addEventListener", "hashchange", handler)
	registerCleanup(handler)
}

// RegisterRoute registers a route with the global router.
func RegisterRoute(path string, component interface{}, options ...Options) {
	GetRouter().Register(path, component, options...)
}

// GetCurrentRouterPath returns the current path from a router instance based on its type.
func (r *Router) GetCurrentRouterPath() string {
	loc := getLocationValue()
	if !loc.Truthy() {
		return "/"
	}

	if r.routerType == "history" {
		// For history router, use pathname
		return normalizePath(loc.Get("pathname").String())
	}

	// For hash router, use hash fragment
	return normalizePath(loc.Get("hash").String())
}

// Navigate navigates to a path using the appropriate method for this router type.
func (r *Router) Navigate(path string) {
	normalized, ok := r.evaluateNavigation(normalizeNavigationTarget(path))
	if !ok {
		return
	}
	if r.routerType == "history" {
		// For history router, use pushState
		history := getHistoryValue()
		if history.Truthy() && history.Get("pushState").Truthy() {
			history.Call("pushState", nil, "", normalized)
		} else if loc := getLocationValue(); loc.Truthy() {
			loc.Set("pathname", normalized)
		}
		r.renderCurrentRoute()
	} else {
		// Hash routers re-render through the hashchange listener.
		if loc := getLocationValue(); loc.Truthy() {
			loc.Set("hash", normalized)
		}
	}
}

// NavigateReplace replaces the current history entry using the appropriate method for this router type.
func (r *Router) NavigateReplace(path string) {
	normalized, ok := r.evaluateNavigation(normalizeNavigationTarget(path))
	if !ok {
		return
	}
	if r.routerType == "history" {
		// For history router, use replaceState
		history := getHistoryValue()
		if history.Truthy() && history.Get("replaceState").Truthy() {
			history.Call("replaceState", nil, "", normalized)
		} else if loc := getLocationValue(); loc.Truthy() {
			loc.Set("pathname", normalized)
		}
		r.renderCurrentRoute()
	} else {
		// Hash routers re-render through the hashchange listener.
		loc := getLocationValue()
		if loc.Truthy() {
			if loc.Get("replace").Truthy() {
				loc.Call("replace", "#"+strings.TrimPrefix(normalized, "#"))
			} else {
				loc.Set("hash", normalized)
			}
		}
	}
}

// Navigate updates the URL using the appropriate method for the current router.
func Navigate(path string) {
	GetRouter().Navigate(path)
}

// NavigateReplace replaces the current history entry using the appropriate method for the current router.
func NavigateReplace(path string) {
	GetRouter().NavigateReplace(path)
}

// RevalidateCurrentRoute clears the current route loader result and runs the route again.
func RevalidateCurrentRoute() {
	GetRouter().RevalidateCurrentRoute()
}

// GetCurrentPath returns the current route path from the global router.
func GetCurrentPath() string {
	return GetRouter().GetCurrentRouterPath()
}

// InspectCurrentRoute returns the current path, query, params, and loading state.
func InspectCurrentRoute() RouteInspection {
	return RouteInspection{
		Path:    GetCurrentPath(),
		Query:   copyQueryValues(getCurrentQueryValues()),
		Params:  copyParams(currentParams),
		Loading: GetRouter().IsRouteLoading(),
	}
}

// UseNavigate returns a typed navigation handle for the global router.
func UseNavigate() Navigator {
	return Navigator{
		navigate: Navigate,
		replace:  NavigateReplace,
	}
}

// UseRevalidator returns a handle for manually revalidating the current route loader.
func UseRevalidator() Revalidator {
	return Revalidator{
		revalidate: RevalidateCurrentRoute,
		loading: func() bool {
			return GetRouter().IsRouteLoading()
		},
	}
}

// Navigate pushes a new route onto the history stack.
func (n Navigator) Navigate(path string) {
	if n.navigate != nil {
		n.navigate(path)
	}
}

// Replace replaces the current route entry.
func (n Navigator) Replace(path string) {
	if n.replace != nil {
		n.replace(path)
	}
}

// Revalidate forces the current route loader to run again.
func (r Revalidator) Revalidate() {
	if r.revalidate != nil {
		r.revalidate()
	}
}

// Loading reports whether the current route loader is still pending.
func (r Revalidator) Loading() bool {
	if r.loading == nil {
		return false
	}
	return r.loading()
}

// UseQuery returns the current URL query values.
//
// For hash routers it reads the `?` portion after the hash path, for example
// `#/search?q=golang`. For history routers it also supports `window.location.search`.
func UseQuery() Query {
	return Query{values: getCurrentQueryValues()}
}

// UseSearchParams returns the current query values together with update helpers
// that preserve the active route path while changing the query string.
func UseSearchParams() SearchParams {
	values := copyQueryValues(getCurrentQueryValues())
	return SearchParams{
		values: values,
		navigate: func(next url.Values) {
			Navigate(buildPathWithQuery(GetCurrentPath(), next))
		},
		replace: func(next url.Values) {
			NavigateReplace(buildPathWithQuery(GetCurrentPath(), next))
		},
	}
}

// UseParams returns the params captured by the current matched route pattern.
func UseParams() Params {
	return Params{values: copyParams(currentParams)}
}

// UseRouteData returns loader-provided route data for the current matched route.
func UseRouteData() Attrs {
	return copyAttrs(currentRouteData)
}

// Get returns the first value for a query key or an empty string.
func (q Query) Get(key string) string {
	if q.values == nil {
		return ""
	}
	return q.values.Get(key)
}

// Has reports whether a query key is present.
func (q Query) Has(key string) bool {
	if q.values == nil {
		return false
	}
	_, ok := q.values[key]
	return ok
}

// Values returns a copy of the parsed query values.
func (q Query) Values() url.Values {
	clone := make(url.Values, len(q.values))
	for key, values := range q.values {
		clone[key] = append([]string(nil), values...)
	}
	return clone
}

// Encode serializes the current query values using net/url encoding.
func (q Query) Encode() string {
	if q.values == nil {
		return ""
	}
	return q.values.Encode()
}

// Get returns the first value for a query key or an empty string.
func (s SearchParams) Get(key string) string {
	return Query{values: s.values}.Get(key)
}

// Has reports whether a query key is present.
func (s SearchParams) Has(key string) bool {
	return Query{values: s.values}.Has(key)
}

// Values returns a copy of the current search params.
func (s SearchParams) Values() url.Values {
	return Query{values: s.values}.Values()
}

// Encode serializes the current search params using net/url encoding.
func (s SearchParams) Encode() string {
	return Query{values: s.values}.Encode()
}

// Set pushes a navigation update with key assigned to value.
func (s SearchParams) Set(key, value string) {
	if s.navigate == nil {
		return
	}
	next := s.Values()
	next.Set(key, value)
	s.navigate(next)
}

// Delete pushes a navigation update with key removed.
func (s SearchParams) Delete(key string) {
	if s.navigate == nil {
		return
	}
	next := s.Values()
	next.Del(key)
	s.navigate(next)
}

// Replace updates the current history entry with key assigned to value.
func (s SearchParams) Replace(key, value string) {
	if s.replace == nil {
		return
	}
	next := s.Values()
	next.Set(key, value)
	s.replace(next)
}

// Navigate applies the provided query values as a pushed navigation update.
func (s SearchParams) Navigate(values url.Values) {
	if s.navigate != nil {
		s.navigate(copyQueryValues(values))
	}
}

// ReplaceAll replaces the current history entry with the provided query values.
func (s SearchParams) ReplaceAll(values url.Values) {
	if s.replace != nil {
		s.replace(copyQueryValues(values))
	}
}

// Get returns the captured route param for key or an empty string.
func (p Params) Get(key string) string {
	if p.values == nil {
		return ""
	}
	return p.values[key]
}

// Has reports whether the route param key exists.
func (p Params) Has(key string) bool {
	if p.values == nil {
		return false
	}
	_, ok := p.values[key]
	return ok
}

// Values returns a copy of the current route params.
func (p Params) Values() map[string]string {
	return copyParams(p.values)
}

// Int parses the route param for key as an int.
func (p Params) Int(key string) (int, bool) {
	raw := p.Get(key)
	if raw == "" {
		return 0, false
	}

	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, false
	}
	return value, true
}

// Bool parses the route param for key as a bool.
func (p Params) Bool(key string) (bool, bool) {
	raw := p.Get(key)
	if raw == "" {
		return false, false
	}

	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false, false
	}
	return value, true
}

// GetRoute returns the component for the current route as an Element.
func GetRoute() *Element {
	return GetRouter().Current()
}

// GetRouter returns the global router instance.
func GetRouter() *Router {
	return globalRouter
}

// RouteWithElement renders a route directly to a DOM element and sets up hash listening.
func RouteWithElement(path string, elemRef js.Value) {
	Navigate(path)
	globalRouter.MountElement(elemRef)
}

var globalRouter = NewHashRouter()
var cleanupOnce sync.Once

func ensureInitialized() {
	if initialized {
		return
	}

	runtime.InitGlobalRuntime(runtime.Config{
		DOMAdapter:   jsdom.NewWASMDOMAdapter(),
		EventAdapter: jsdom.NewWASMEventAdapter(),
		Scheduler:    jsdom.NewWASMScheduler(),
		BrowserState: jsdom.NewWASMBrowserState(),
	})
	initialized = true
}

func registerCleanup(handler js.Func) {
	cleanupOnce.Do(func() {
		window := js.Global().Get("window")
		if !window.Truthy() || !window.Get("addEventListener").Truthy() {
			return
		}
		// Register unload listener to release the hashchange handler to avoid leaks in hot reload.
		unload := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			handler.Release()
			return nil
		})
		window.Call("addEventListener", "beforeunload", unload)
	})
}

func makeRouteFactory(component interface{}) routeFactory {
	if component == nil {
		runtime.ReportDiagnostic("router", runtime.DiagnosticError, "route component cannot be nil")
		panic("router: component cannot be nil")
	}

	if element, ok := component.(*Element); ok {
		return func(_ Attrs) *Element {
			return element
		}
	}

	value := reflect.ValueOf(component)
	if !value.IsValid() || value.Kind() != reflect.Func {
		runtime.ReportDiagnostic("router", runtime.DiagnosticError, "unsupported route component type")
		panic("router: unsupported component type")
	}

	typ := value.Type()
	if typ.NumOut() != 1 {
		runtime.ReportDiagnostic("router", runtime.DiagnosticError, "route component must return exactly one element")
		panic("router: route component must return one element")
	}

	return func(attrs Attrs) *Element {
		args := []reflect.Value{}
		if typ.NumIn() == 1 {
			arg := reflect.Zero(typ.In(0))
			if attrs != nil {
				provided := reflect.ValueOf(attrs)
				switch {
				case provided.IsValid() && provided.Type() == typ.In(0):
					arg = provided
				case provided.IsValid() && provided.Type().AssignableTo(typ.In(0)):
					arg = provided
				case provided.IsValid() && provided.Type().ConvertibleTo(typ.In(0)):
					arg = provided.Convert(typ.In(0))
				}
			}
			args = append(args, arg)
		}

		results := value.Call(args)
		if len(results) == 0 || !results[0].IsValid() || results[0].IsNil() {
			return nil
		}

		element, _ := results[0].Interface().(*Element)
		return element
	}
}

func getCurrentQueryValues() url.Values {
	loc := getLocationValue()
	if !loc.Truthy() {
		return url.Values{}
	}

	raw := strings.TrimPrefix(loc.Get("search").String(), "?")
	if raw == "" {
		hash := strings.TrimPrefix(loc.Get("hash").String(), "#")
		if idx := strings.Index(hash, "?"); idx >= 0 && idx+1 < len(hash) {
			raw = hash[idx+1:]
		}
	}

	if raw == "" {
		return url.Values{}
	}

	values, err := url.ParseQuery(raw)
	if err != nil {
		return url.Values{}
	}
	return values
}

func buildPathWithQuery(path string, values url.Values) string {
	normalized := normalizePath(path)
	encoded := copyQueryValues(values).Encode()
	if encoded == "" {
		return normalized
	}
	return normalized + "?" + encoded
}

func getLocationValue() js.Value {
	window := js.Global().Get("window")
	if window.Truthy() {
		loc := window.Get("location")
		if loc.Truthy() {
			return loc
		}
	}
	return js.Global().Get("location")
}

func getHistoryValue() js.Value {
	window := js.Global().Get("window")
	if window.Truthy() {
		history := window.Get("history")
		if history.Truthy() {
			return history
		}
	}
	return js.Global().Get("history")
}

func (r *Router) renderResolvedRoute(path string, factory routeFactory, option Options, params map[string]string, query url.Values, queryKey string) *Element {
	if blocked := r.applyBeforeEnterGuard(path, option, params, query); blocked != nil {
		return blocked
	}
	if redirected := r.applyRouteOptions(path, option, query); redirected != nil {
		return redirected
	}

	baseProps := copyParamsToAttrs(params)
	if option.Loader == nil {
		r.cancelLoaderIfActive()
		currentRouteData = nil
		return factory(baseProps)
	}

	loaderKey := buildLoaderKey(path, queryKey)
	state := r.ensureLoaderResult(loaderKey, option.Loader, RouteContext{
		Path:   path,
		Params: Params{values: copyParams(params)},
		Query:  Query{values: copyQueryValues(query)},
	})

	if state.pending {
		currentRouteData = nil
		return renderRouteFallback(option.Loading, mergeAttrs(baseProps, Attrs{"path": path, "loading": true}))
	}
	if state.err != nil {
		currentRouteData = nil
		return renderRouteError(option.Error, state.err, mergeAttrs(baseProps, Attrs{"path": path, "error": state.err.Error()}))
	}

	currentRouteData = copyAttrs(state.data)
	return factory(mergeAttrs(baseProps, state.data))
}

func (r *Router) resolveRoute(path string) resolvedRoute {
	if comp, ok := r.routes[path]; ok {
		return resolvedRoute{path: path, params: map[string]string{}, option: r.routeOptions[path], factory: comp, found: true}
	}
	if comp, params, ok := r.matchPattern(path); ok {
		_, option := r.matchOptions(path)
		return resolvedRoute{path: path, params: copyParams(params), option: option, factory: comp, found: true}
	}
	if r.notFound != nil {
		return resolvedRoute{path: path, params: map[string]string{}, option: r.notFoundOption, factory: r.notFound, found: true}
	}
	if r.defaultRoute != "" {
		if comp, ok := r.routes[r.defaultRoute]; ok {
			return resolvedRoute{path: r.defaultRoute, params: map[string]string{}, option: r.routeOptions[r.defaultRoute], factory: comp, found: true}
		}
	}
	return resolvedRoute{}
}

func (r *Router) routeContext(path string, params map[string]string, query url.Values) RouteContext {
	return RouteContext{
		Path:   path,
		Params: Params{values: copyParams(params)},
		Query:  Query{values: copyQueryValues(query)},
	}
}

func (r *Router) applyBeforeEnterGuard(path string, option Options, params map[string]string, query url.Values) *Element {
	if option.BeforeEnter == nil {
		return nil
	}
	result := option.BeforeEnter(r.routeContext(path, params, query))
	if target := strings.TrimSpace(result.Redirect); target != "" {
		normalized := normalizeNavigationTarget(target)
		if normalized == buildPathWithQuery(path, query) {
			runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "ignoring route before-enter redirect loop for "+normalized)
			return nil
		}
		r.replaceLocation(normalized)
		return r.Current()
	}
	if !result.Blocked {
		return nil
	}
	currentRouteData = nil
	r.cancelLoaderIfActive()
	message := strings.TrimSpace(result.Reason)
	if message == "" {
		message = "Navigation blocked"
	}
	return runtime.Div(nil, runtime.Text(message))
}

func (r *Router) applyRouteOptions(path string, option Options, query url.Values) *Element {
	r.applyRouteMetadata(option)
	if option.Redirect == "" {
		return nil
	}

	currentTarget := buildPathWithQuery(path, query)
	redirectTarget := normalizeNavigationTarget(option.Redirect)
	if redirectTarget == currentTarget {
		runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "ignoring route redirect loop for "+redirectTarget)
		return nil
	}

	r.replaceLocation(redirectTarget)
	return r.Current()
}

func (r *Router) applyRouteMetadata(option Options) {
	applyRouteTitle(&r.metadataState, option.Title)
	applyRouteMetaTag("description", option.Description)
	applyRouteCanonical(option.CanonicalURL)
}

func applyRouteTitle(state *routeMetadataState, title string) {
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		return
	}
	if state != nil && !state.baseTitleCaptured {
		state.baseTitleCaptured = true
		state.baseTitle = doc.Get("title").String()
	}

	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		if state != nil && state.titleManaged {
			doc.Set("title", state.baseTitle)
			state.titleManaged = false
		}
		return
	}

	doc.Set("title", trimmed)
	if state != nil {
		state.titleManaged = true
	}
}

func applyRouteMetaTag(name, content string) {
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		return
	}
	head := getHeadElement(doc)
	if !head.Truthy() {
		return
	}
	element := doc.Call("querySelector", `meta[name="`+name+`"]`)
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		if element.Truthy() && element.Get("remove").Truthy() {
			element.Call("remove")
		}
		return
	}
	if !element.Truthy() {
		element = doc.Call("createElement", "meta")
		element.Call("setAttribute", "name", name)
		head.Call("appendChild", element)
	}
	element.Call("setAttribute", "content", trimmed)
}

func applyRouteCanonical(href string) {
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		return
	}
	head := getHeadElement(doc)
	if !head.Truthy() {
		return
	}
	element := doc.Call("querySelector", `link[rel="canonical"]`)
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		if element.Truthy() && element.Get("remove").Truthy() {
			element.Call("remove")
		}
		return
	}
	if !element.Truthy() {
		element = doc.Call("createElement", "link")
		element.Call("setAttribute", "rel", "canonical")
		head.Call("appendChild", element)
	}
	element.Call("setAttribute", "href", trimmed)
}

func getHeadElement(doc js.Value) js.Value {
	head := doc.Get("head")
	if head.Truthy() {
		return head
	}
	return doc.Call("querySelector", "head")
}

func (r *Router) evaluateNavigation(target string) (string, bool) {
	currentPath := r.GetCurrentRouterPath()
	currentQuery := getCurrentQueryValues()
	currentResolved := r.resolveRoute(currentPath)

	nextTarget := target
	for steps := 0; steps < 4; steps++ {
		nextPath, nextQuery := parseNavigationTarget(nextTarget)
		nextResolved := r.resolveRoute(nextPath)
		nextCtx := r.routeContext(nextPath, nextResolved.params, nextQuery)

		if steps == 0 && currentResolved.found && currentResolved.option.BeforeLeave != nil {
			result := currentResolved.option.BeforeLeave(r.routeContext(currentPath, currentResolved.params, currentQuery), nextCtx)
			if redirect := strings.TrimSpace(result.Redirect); redirect != "" {
				nextTarget = normalizeNavigationTarget(redirect)
				continue
			}
			if result.Blocked {
				return "", false
			}
		}

		if nextResolved.found && nextResolved.option.BeforeEnter != nil {
			result := nextResolved.option.BeforeEnter(r.routeContext(nextPath, nextResolved.params, nextQuery))
			if redirect := strings.TrimSpace(result.Redirect); redirect != "" {
				nextTarget = normalizeNavigationTarget(redirect)
				continue
			}
			if result.Blocked {
				return "", false
			}
		}

		return nextTarget, true
	}

	runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "navigation guard redirect loop detected for "+target)
	return "", false
}

func parseNavigationTarget(target string) (string, url.Values) {
	normalized := normalizeNavigationTarget(target)
	path := normalized
	query := url.Values{}
	if idx := strings.Index(normalized, "?"); idx >= 0 {
		path = normalized[:idx]
		parsed, err := url.ParseQuery(normalized[idx+1:])
		if err == nil {
			query = parsed
		}
	}
	return path, query
}

func AllowNavigation() GuardResult {
	return GuardResult{}
}

func BlockNavigation(reason string) GuardResult {
	return GuardResult{Blocked: true, Reason: reason}
}

func RedirectNavigation(path string) GuardResult {
	return GuardResult{Redirect: path}
}

func (r *Router) replaceLocation(target string) {
	if r.routerType == "history" {
		history := getHistoryValue()
		if history.Truthy() && history.Get("replaceState").Truthy() {
			history.Call("replaceState", nil, "", target)
			return
		}
		if loc := getLocationValue(); loc.Truthy() {
			if idx := strings.Index(target, "?"); idx >= 0 {
				loc.Set("pathname", target[:idx])
				loc.Set("search", target[idx:])
			} else {
				loc.Set("pathname", target)
				loc.Set("search", "")
			}
		}
		return
	}

	loc := getLocationValue()
	if !loc.Truthy() {
		return
	}
	loc.Set("hash", "#"+strings.TrimPrefix(target, "#"))
}

func (r *Router) ensureLoaderResult(key string, loader LoaderFunc, routeCtx RouteContext) struct {
	pending bool
	data    Attrs
	err     error
} {
	r.loaderState.mu.Lock()
	if r.loaderState.key == key {
		state := struct {
			pending bool
			data    Attrs
			err     error
		}{
			pending: r.loaderState.pending,
			data:    copyAttrs(r.loaderState.data),
			err:     r.loaderState.err,
		}
		r.loaderState.mu.Unlock()
		return state
	}

	if r.loaderState.cancel != nil {
		r.loaderState.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.loaderState.version++
	version := r.loaderState.version
	r.loaderState.key = key
	r.loaderState.pending = true
	r.loaderState.data = nil
	r.loaderState.err = nil
	r.loaderState.cancel = cancel
	r.loaderState.mu.Unlock()

	go func() {
		data, err := loader(ctx, routeCtx)

		r.loaderState.mu.Lock()
		defer r.loaderState.mu.Unlock()
		if ctx.Err() != nil || version != r.loaderState.version || key != r.loaderState.key {
			return
		}
		r.loaderState.pending = false
		r.loaderState.data = copyAttrs(data)
		r.loaderState.err = err
		r.loaderState.cancel = nil

		go func() {
			doc := js.Global().Get("document")
			elem := js.Global().Get("Element")
			if doc.IsUndefined() || doc.IsNull() || elem.IsUndefined() || elem.IsNull() {
				return
			}
			r.renderCurrentRoute()
		}()
	}()

	return struct {
		pending bool
		data    Attrs
		err     error
	}{pending: true}
}

func (r *Router) cancelLoaderIfActive() {
	r.loaderState.mu.Lock()
	defer r.loaderState.mu.Unlock()
	if r.loaderState.cancel != nil {
		r.loaderState.cancel()
		r.loaderState.cancel = nil
	}
	r.loaderState.key = ""
	r.loaderState.pending = false
	r.loaderState.data = nil
	r.loaderState.err = nil
	r.loaderState.version++
}

func buildLoaderKey(path, queryKey string) string {
	if queryKey == "" {
		return path
	}
	return path + "?" + queryKey
}

func renderRouteFallback(component interface{}, props Attrs) *Element {
	if component != nil {
		return makeRouteFactory(component)(props)
	}
	return runtime.Div(nil, runtime.Text("Loading route..."))
}

func renderRouteError(component interface{}, err error, props Attrs) *Element {
	if component != nil {
		return makeRouteFactory(component)(props)
	}
	message := "Route load failed"
	if err != nil {
		message = err.Error()
	}
	return runtime.Div(nil, runtime.Text(message))
}

func isPatternRoute(path string) bool {
	return strings.Contains(path, ":") || (strings.HasSuffix(path, "*") && path != "*")
}

func (r *Router) matchPattern(path string) (routeFactory, map[string]string, bool) {
	for _, pattern := range r.patterns {
		if params, ok := matchRoutePattern(pattern.pattern, path); ok {
			return pattern.factory, params, true
		}
	}
	return nil, nil, false
}

func (r *Router) matchOptions(path string) (map[string]string, Options) {
	for _, pattern := range r.patterns {
		if params, ok := matchRoutePattern(pattern.pattern, path); ok {
			return params, pattern.options
		}
	}
	return nil, Options{}
}

func matchRoutePattern(pattern, path string) (map[string]string, bool) {
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		prefix = strings.TrimSuffix(prefix, "/")
		if prefix == "" {
			return map[string]string{}, true
		}
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return map[string]string{}, true
		}
		return nil, false
	}

	patternParts := splitPath(pattern)
	pathParts := splitPath(path)
	if len(patternParts) != len(pathParts) {
		return nil, false
	}

	params := make(map[string]string)
	for index, part := range patternParts {
		candidate := pathParts[index]
		if strings.HasPrefix(part, ":") {
			name := strings.TrimPrefix(part, ":")
			if name == "" || strings.ContainsAny(name, "?*") {
				return nil, false
			}
			decoded, err := url.PathUnescape(candidate)
			if err != nil || decoded == "" {
				return nil, false
			}
			params[name] = decoded
			continue
		}
		if part != candidate {
			return nil, false
		}
	}

	return params, true
}

func splitPath(path string) []string {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

func copyParams(params map[string]string) map[string]string {
	if len(params) == 0 {
		return map[string]string{}
	}
	clone := make(map[string]string, len(params))
	for key, value := range params {
		clone[key] = value
	}
	return clone
}

func copyParamsToAttrs(params map[string]string) Attrs {
	if len(params) == 0 {
		return nil
	}
	attrs := make(Attrs, len(params))
	for key, value := range params {
		attrs[key] = value
	}
	return attrs
}

func copyAttrs(attrs Attrs) Attrs {
	if len(attrs) == 0 {
		return nil
	}
	clone := make(Attrs, len(attrs))
	for key, value := range attrs {
		clone[key] = value
	}
	return clone
}

func copyQueryValues(values url.Values) url.Values {
	clone := make(url.Values, len(values))
	for key, value := range values {
		clone[key] = append([]string(nil), value...)
	}
	return clone
}

func mergeAttrs(base Attrs, extra Attrs) Attrs {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	merged := make(Attrs, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}
