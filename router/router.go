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

const (
	rootRoutePath      = "/"
	catchAllRoutePath  = "*"
	routerTypeHash     = "hash"
	routerTypeHistory  = "history"
	browserEventHash   = "hashchange"
	browserEventPop    = "popstate"
	routeNotFoundText  = "Route not found"
	routeLoadingText   = "Loading route..."
	navigationBlocked  = "Navigation blocked"
	defaultRoutePrefix = "default:"
	ReturnToParam      = "return_to"
	maxReturnToLength  = 2048
)

// Component is a type alias for component functions used in routing.
type Attrs = map[string]interface{}
type Element = runtime.Element
type Component = func(Attrs) *Element

// Options represents configuration for individual routes.
// Placeholder for future per-route settings (e.g., titles, guards).
type Options struct {
	Title            string
	Redirect         string
	Description      string
	CanonicalURL     string
	Layout           bool
	BeforeEnter      GuardFunc
	BeforeLeave      LeaveGuardFunc
	BeforeEnterAsync AsyncGuardFunc
	BeforeLeaveAsync AsyncLeaveGuardFunc
	GuardPending     interface{}
	Unauthorized     interface{}
	Authorizing      interface{}
	Loader           LoaderFunc
	Loading          interface{}
	Error            interface{}
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
	guardState     navigationGuardState
}

type navigationGuardState struct {
	mu     sync.Mutex
	seq    uint64
	active uint64
	cancel context.CancelFunc
}

type routeMetadataState struct {
	baseTitleCaptured bool
	baseTitle         string
	titleManaged      bool
}

// Navigator provides push and replace navigation helpers.
type Navigator struct {
	navigate func(string)
	replace  func(string)
}

// Revalidator exposes route-loader revalidation and loading state.
type Revalidator struct {
	revalidate func()
	loading    func() bool
}

// Query provides read-only access to parsed query values.
type Query struct {
	values url.Values
}

// SearchParams provides query values plus navigation helpers that preserve the current path.
type SearchParams struct {
	values   url.Values
	navigate func(url.Values)
	replace  func(url.Values)
}

// Params provides read-only access to matched route params.
type Params struct {
	values map[string]string
}

// RouteInspection summarizes the currently active route.
type RouteInspection struct {
	Path    string
	Query   url.Values
	Params  map[string]string
	Loading bool
}

// RouteContext describes the path, params, and query for a route evaluation.
type RouteContext struct {
	Path   string
	Params Params
	Query  Query
}

// GuardResult describes the outcome of a navigation guard.
type GuardResult struct {
	Redirect string
	Blocked  bool
	Reason   string
}

// GuardDecision describes the outcome of an async navigation guard.
type GuardDecision struct {
	Redirect  string
	Blocked   bool
	Reason    string
	Retryable bool
	Denied    bool
}

// GuardFunc decides whether navigation into a route should proceed.
type GuardFunc func(RouteContext) GuardResult

// LeaveGuardFunc decides whether navigation away from a route should proceed.
type LeaveGuardFunc func(current RouteContext, next RouteContext) GuardResult

// AsyncGuardFunc decides whether navigation into a route should proceed.
type AsyncGuardFunc func(context.Context, RouteContext) GuardDecision

// AsyncLeaveGuardFunc decides whether navigation away from a route should proceed.
type AsyncLeaveGuardFunc func(context.Context, RouteContext, RouteContext) GuardDecision

type resolvedRoute struct {
	id      string
	path    string
	params  map[string]string
	option  Options
	factory routeFactory
	found   bool
}

type resolvedRouteStack struct {
	routes []resolvedRoute
	found  bool
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
	entries map[string]*loaderEntry
	active  map[string]struct{}
}

type loaderEntry struct {
	pending bool
	data    Attrs
	err     error
	cancel  context.CancelFunc
	version int
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || trimmed == "#" {
		return rootRoutePath
	}
	if trimmed == catchAllRoutePath {
		return catchAllRoutePath
	}

	trimmed = strings.TrimPrefix(trimmed, "#")
	if idx := strings.Index(trimmed, "?"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	if trimmed == "" {
		return rootRoutePath
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = rootRoutePath + trimmed
	}
	if len(trimmed) > 1 {
		trimmed = strings.TrimRight(trimmed, "/")
		if trimmed == "" {
			return rootRoutePath
		}
	}
	return trimmed
}

func normalizeNavigationTarget(target string) string {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" || trimmed == "#" {
		return rootRoutePath
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

var routerRuntimeInitialized bool
var currentParams = map[string]string{}
var currentRouteData Attrs
var currentRouteOutlet *Element

// NewHashRouter creates a hash-based router that reads from window.location.hash.
func NewHashRouter(options ...RouterOptions) *Router {
	cfg := RouterOptions{DefaultRoute: rootRoutePath}
	if len(options) > 0 {
		cfg = options[0]
	}
	cfg.DefaultRoute = normalizePath(cfg.DefaultRoute)

	return &Router{
		routes:       make(map[string]routeFactory),
		routeOptions: make(map[string]Options),
		patterns:     []routePattern{},
		defaultRoute: cfg.DefaultRoute,
		routerType:   routerTypeHash,
		loaderState: loaderState{
			entries: make(map[string]*loaderEntry),
			active:  make(map[string]struct{}),
		},
	}
}

// NewRouter creates a history-based router using the HTML5 History API.
// This router uses window.location.pathname instead of hash fragments.
// Requires server to redirect all routes to the app's entry point.
func NewRouter(options RouterOptions) *Router {
	// Use provided options or defaults
	if options.DefaultRoute == "" {
		options.DefaultRoute = rootRoutePath
	}
	options.DefaultRoute = normalizePath(options.DefaultRoute)

	router := &Router{
		routes:       make(map[string]routeFactory),
		routeOptions: make(map[string]Options),
		patterns:     []routePattern{},
		defaultRoute: options.DefaultRoute,
		routerType:   routerTypeHistory,
		loaderState: loaderState{
			entries: make(map[string]*loaderEntry),
			active:  make(map[string]struct{}),
		},
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
		r.renderCurrentRoute(true)
		return nil
	})

	window.Call("addEventListener", browserEventPop, popstateHandler)

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

	if normalize == catchAllRoutePath {
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
	return r.currentElement(true)
}

func (r *Router) currentElement(applyGuards bool) *Element {
	globalRouter = r
	path := r.GetCurrentRouterPath()
	if path == "" {
		path = r.defaultRoute
	}
	query := getCurrentQueryValues()
	queryKey := query.Encode()
	resolved := r.resolveRouteStack(path)
	if resolved.found {
		leaf := resolved.routes[len(resolved.routes)-1]
		currentParams = copyParams(leaf.params)
		if applyGuards {
			ctx, attemptID := r.beginGuardAttempt()
			defer r.finishGuardAttempt(attemptID)
			rendered := r.renderResolvedRouteStack(resolved.routes, query, queryKey, true, ctx, attemptID)
			if ctx.Err() != nil || !r.guardAttemptActive(attemptID) {
				return nil
			}
			return rendered
		}
		return r.renderResolvedRouteStack(resolved.routes, query, queryKey, false, nil, 0)
	}

	r.cancelLoaderIfActive()
	currentRouteData = nil
	currentRouteOutlet = nil

	return runtime.Div(nil, runtime.Text(routeNotFoundText))
}

// Mount renders the router into a DOM node selected by CSS selector and wires hashchange listeners.
func (r *Router) Mount(selector string) {
	globalRouter = r
	r.targetSelector = selector
	r.targetElement = js.Null()
	r.renderCurrentRoute(true)
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
	r.renderCurrentRoute(true)
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
	r.renderCurrentRoute(false)
}

// IsRouteLoading reports whether the current route loader is pending.
func (r *Router) IsRouteLoading() bool {
	r.loaderState.mu.Lock()
	defer r.loaderState.mu.Unlock()
	for key := range r.loaderState.active {
		entry := r.loaderState.entries[key]
		if entry != nil && entry.pending {
			return true
		}
	}
	return false
}

func (r *Router) renderCurrentRoute(applyGuards bool) {
	ensureInitialized()
	rt := runtime.GetGlobalRuntime()
	routeElement := r.currentElement(applyGuards)
	if routeElement == nil {
		return
	}
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
		r.renderCurrentRoute(true)
		return nil
	})
	window.Call("addEventListener", browserEventHash, handler)
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
		return rootRoutePath
	}

	if r.routerType == routerTypeHistory {
		// For history router, use pathname
		return normalizePath(loc.Get("pathname").String())
	}

	// For hash router, use hash fragment
	return normalizePath(loc.Get("hash").String())
}

// Navigate navigates to a path using the appropriate method for this router type.
func (r *Router) Navigate(path string) {
	ctx, attemptID := r.beginGuardAttempt()
	defer r.finishGuardAttempt(attemptID)

	normalized, ok := r.evaluateNavigationWithAttempt(ctx, attemptID, normalizeNavigationTarget(path))
	if !ok {
		return
	}
	if ctx.Err() != nil || !r.guardAttemptActive(attemptID) {
		return
	}
	runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "navigation started", "", map[string]string{
		"target": normalized,
		"mode":   "push",
		"kind":   r.routerType,
	})
	if r.routerType == routerTypeHistory {
		// For history router, use pushState
		history := getHistoryValue()
		if history.Truthy() && history.Get("pushState").Truthy() {
			history.Call("pushState", nil, "", normalized)
		} else if loc := getLocationValue(); loc.Truthy() {
			loc.Set("pathname", normalized)
		}
	} else {
		// Hash routers re-render through the hashchange listener.
		if loc := getLocationValue(); loc.Truthy() {
			loc.Set("hash", normalized)
		}
	}
	if r.routerType == routerTypeHistory {
		if ctx.Err() != nil || !r.guardAttemptActive(attemptID) {
			return
		}
		r.renderCurrentRoute(false)
	}
}

// NavigateReplace replaces the current history entry using the appropriate method for this router type.
func (r *Router) NavigateReplace(path string) {
	ctx, attemptID := r.beginGuardAttempt()
	defer r.finishGuardAttempt(attemptID)

	normalized, ok := r.evaluateNavigationWithAttempt(ctx, attemptID, normalizeNavigationTarget(path))
	if !ok {
		return
	}
	if ctx.Err() != nil || !r.guardAttemptActive(attemptID) {
		return
	}
	runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "navigation started", "", map[string]string{
		"target": normalized,
		"mode":   "replace",
		"kind":   r.routerType,
	})
	if r.routerType == routerTypeHistory {
		// For history router, use replaceState
		history := getHistoryValue()
		if history.Truthy() && history.Get("replaceState").Truthy() {
			history.Call("replaceState", nil, "", normalized)
		} else if loc := getLocationValue(); loc.Truthy() {
			loc.Set("pathname", normalized)
		}
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
	if r.routerType == routerTypeHistory {
		if ctx.Err() != nil || !r.guardAttemptActive(attemptID) {
			return
		}
		r.renderCurrentRoute(false)
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

// Outlet returns the child route element for the current layout route, if one exists.
func Outlet() *Element {
	return currentRouteOutlet
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
	if routerRuntimeInitialized {
		return
	}

	runtime.InitGlobalRuntime(runtime.Config{
		DOMAdapter:   jsdom.NewWASMDOMAdapter(),
		EventAdapter: jsdom.NewWASMEventAdapter(),
		Scheduler:    jsdom.NewWASMScheduler(),
		BrowserState: jsdom.NewWASMBrowserState(),
	})
	routerRuntimeInitialized = true
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

func (r *Router) renderResolvedRouteStack(routes []resolvedRoute, query url.Values, queryKey string, applyGuards bool, guardCtx context.Context, attemptID uint64) *Element {
	loaderKeys := make([]string, 0, len(routes))
	for _, route := range routes {
		if route.option.Loader != nil {
			loaderKeys = append(loaderKeys, buildLoaderKey(route.id, route.path, queryKey))
		}
	}
	r.prepareLoaderState(loaderKeys)
	return r.renderRouteLevel(routes, 0, query, queryKey, applyGuards, guardCtx, attemptID)
}

func (r *Router) renderRouteLevel(routes []resolvedRoute, index int, query url.Values, queryKey string, applyGuards bool, guardCtx context.Context, attemptID uint64) *Element {
	match := routes[index]
	if applyGuards {
		if blocked := r.applyBeforeEnterGuard(match.path, match.option, match.params, query, guardCtx, attemptID); blocked != nil {
			return blocked
		}
	}
	if redirected := r.applyRouteOptions(match.path, match.option, query); redirected != nil {
		return redirected
	}

	baseProps := copyParamsToAttrs(match.params)
	data := Attrs(nil)
	if match.option.Loader != nil {
		loaderKey := buildLoaderKey(match.id, match.path, queryKey)
		state := r.ensureLoaderResult(loaderKey, match.option.Loader, RouteContext{
			Path:   match.path,
			Params: Params{values: copyParams(match.params)},
			Query:  Query{values: copyQueryValues(query)},
		})

		if state.pending {
			currentRouteData = nil
			currentRouteOutlet = nil
			return renderRouteFallback(match.option.Loading, mergeAttrs(baseProps, Attrs{"path": match.path, "loading": true}))
		}
		if state.err != nil {
			currentRouteData = nil
			currentRouteOutlet = nil
			return renderRouteError(match.option.Error, state.err, mergeAttrs(baseProps, Attrs{"path": match.path, "error": state.err.Error()}))
		}

		data = copyAttrs(state.data)
		baseProps = mergeAttrs(baseProps, data)
	}

	var outlet *Element
	if index+1 < len(routes) {
		outlet = r.renderRouteLevel(routes, index+1, query, queryKey, applyGuards, guardCtx, attemptID)
	}

	prevParams, prevData, prevOutlet := withRouteRenderContext(match.params, data, outlet)
	defer restoreRouteRenderContext(prevParams, prevData, prevOutlet)

	return match.factory(baseProps)
}

func withRouteRenderContext(params map[string]string, data Attrs, outlet *Element) (map[string]string, Attrs, *Element) {
	prevParams := currentParams
	prevData := currentRouteData
	prevOutlet := currentRouteOutlet
	currentParams = copyParams(params)
	currentRouteData = copyAttrs(data)
	currentRouteOutlet = outlet
	return prevParams, prevData, prevOutlet
}

func restoreRouteRenderContext(params map[string]string, data Attrs, outlet *Element) {
	currentParams = params
	currentRouteData = data
	currentRouteOutlet = outlet
}

func (r *Router) resolveRouteStack(path string) resolvedRouteStack {
	leaf := r.resolveRoute(path)
	if !leaf.found {
		return resolvedRouteStack{}
	}

	if leaf.id == defaultRoutePrefix+r.defaultRoute {
		return resolvedRouteStack{routes: []resolvedRoute{leaf}, found: true}
	}

	prefixes := expandPathPrefixes(path)
	routes := make([]resolvedRoute, 0, len(prefixes)+1)
	seen := map[string]struct{}{}
	for _, prefix := range prefixes {
		if comp, ok := r.routes[prefix]; ok {
			option := r.routeOptions[prefix]
			id := routeIDExact(prefix)
			if option.Layout && id != leaf.id {
				routes = append(routes, resolvedRoute{id: id, path: prefix, params: map[string]string{}, option: option, factory: comp, found: true})
				seen[id] = struct{}{}
			}
		}

		for _, pattern := range r.patterns {
			if !pattern.options.Layout {
				continue
			}
			id := routeIDPattern(pattern.pattern)
			if id == leaf.id {
				continue
			}
			if _, exists := seen[id]; exists {
				continue
			}
			params, ok := matchRoutePattern(pattern.pattern, prefix)
			if !ok {
				continue
			}
			routes = append(routes, resolvedRoute{id: id, path: prefix, params: copyParams(params), option: pattern.options, factory: pattern.factory, found: true})
			seen[id] = struct{}{}
		}
	}

	routes = append(routes, leaf)
	return resolvedRouteStack{routes: routes, found: true}
}

func (r *Router) resolveRoute(path string) resolvedRoute {
	if comp, ok := r.routes[path]; ok {
		return resolvedRoute{id: routeIDExact(path), path: path, params: map[string]string{}, option: r.routeOptions[path], factory: comp, found: true}
	}
	if comp, params, option, pattern, ok := r.matchPattern(path); ok {
		return resolvedRoute{id: routeIDPattern(pattern), path: path, params: copyParams(params), option: option, factory: comp, found: true}
	}
	if r.notFound != nil {
		return resolvedRoute{id: routeIDNotFound(), path: path, params: map[string]string{}, option: r.notFoundOption, factory: r.notFound, found: true}
	}
	if r.defaultRoute != "" {
		if comp, ok := r.routes[r.defaultRoute]; ok {
			return resolvedRoute{id: defaultRoutePrefix + r.defaultRoute, path: r.defaultRoute, params: map[string]string{}, option: r.routeOptions[r.defaultRoute], factory: comp, found: true}
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

func (r *Router) applyBeforeEnterGuard(path string, option Options, params map[string]string, query url.Values, guardCtx context.Context, attemptID uint64) *Element {
	if option.BeforeEnter == nil && option.BeforeEnterAsync == nil {
		return nil
	}
	routeCtx := r.routeContext(path, params, query)
	decision := guardDecisionAllowed()
	if option.BeforeEnter != nil {
		decision = guardDecisionFromResult(option.BeforeEnter(routeCtx))
	}
	if !decision.Blocked && decision.Redirect == "" && option.BeforeEnterAsync != nil {
		if guardCtx == nil {
			guardCtx = context.Background()
		}
		decision = option.BeforeEnterAsync(guardCtx, routeCtx)
	}
	if (guardCtx != nil && guardCtx.Err() != nil) || (attemptID != 0 && !r.guardAttemptActive(attemptID)) {
		return nil
	}
	if target := strings.TrimSpace(decision.Redirect); target != "" {
		normalized := normalizeNavigationTarget(target)
		if normalized == buildPathWithQuery(path, query) {
			runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "ignoring route before-enter redirect loop for "+normalized)
			return nil
		}
		runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "before-enter redirected navigation", "", map[string]string{
			"from": path,
			"to":   normalized,
		})
		r.replaceLocation(normalized)
		return r.currentElement(false)
	}
	if !decision.Blocked && !decision.Denied {
		return nil
	}
	currentRouteData = nil
	r.cancelLoaderIfActive()
	message := strings.TrimSpace(decision.Reason)
	if message == "" {
		message = navigationBlocked
	}
	runtime.ReportLogWithFields("router", runtime.LogWarn, runtime.DiagnosticRecovered, "before-enter blocked navigation", "", map[string]string{
		"path":   path,
		"reason": message,
	})
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

	runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "route redirect applied", "", map[string]string{
		"from": currentTarget,
		"to":   redirectTarget,
	})
	r.replaceLocation(redirectTarget)
	return r.currentElement(false)
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
	initializeRouteMetadataState(state, doc)

	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		if state != nil && state.titleManaged {
			if titleElement := ensureManagedTitleElement(doc, false); titleElement.Truthy() {
				if strings.TrimSpace(state.baseTitle) == "" {
					removeElement(titleElement)
				} else {
					titleElement.Set("textContent", state.baseTitle)
					if titleElement.Get("removeAttribute").Truthy() {
						titleElement.Call("removeAttribute", managedMetadataAttr)
					}
				}
			}
			doc.Set("title", state.baseTitle)
			state.titleManaged = false
		}
		return
	}

	titleElement := ensureManagedTitleElement(doc, true)
	if titleElement.Truthy() {
		titleElement.Set("textContent", trimmed)
		titleElement.Call("setAttribute", managedMetadataAttr, managedMetadataValue)
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
	element := ensureManagedHeadElement(doc, head, `meta[name="`+name+`"]`, "meta", func(node js.Value) {
		node.Call("setAttribute", "name", name)
	})
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		if element.Truthy() {
			removeElement(element)
		}
		return
	}
	if !element.Truthy() {
		return
	}
	element.Call("setAttribute", managedMetadataAttr, managedMetadataValue)
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
	element := ensureManagedHeadElement(doc, head, `link[rel="canonical"]`, "link", func(node js.Value) {
		node.Call("setAttribute", "rel", "canonical")
	})
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		if element.Truthy() {
			removeElement(element)
		}
		return
	}
	if !element.Truthy() {
		return
	}
	element.Call("setAttribute", managedMetadataAttr, managedMetadataValue)
	element.Call("setAttribute", "href", trimmed)
}

func initializeRouteMetadataState(state *routeMetadataState, doc js.Value) {
	if state == nil || state.baseTitleCaptured {
		return
	}
	state.baseTitleCaptured = true
	state.baseTitle = doc.Get("title").String()
	if title := findManagedHeadElement(doc, `title[`+managedMetadataAttr+`="`+managedMetadataValue+`"]`); title.Truthy() {
		state.baseTitle = ""
		state.titleManaged = true
	}
}

func ensureManagedTitleElement(doc js.Value, create bool) js.Value {
	if element := findManagedHeadElement(doc, `title[`+managedMetadataAttr+`="`+managedMetadataValue+`"]`); element.Truthy() {
		return element
	}
	titles := querySelectorAll(doc, "title")
	if len(titles) == 1 {
		titles[0].Call("setAttribute", managedMetadataAttr, managedMetadataValue)
		return titles[0]
	}
	if !create {
		return js.Null()
	}
	head := getHeadElement(doc)
	if !head.Truthy() {
		return js.Null()
	}
	element := doc.Call("createElement", "title")
	element.Call("setAttribute", managedMetadataAttr, managedMetadataValue)
	head.Call("appendChild", element)
	return element
}

func ensureManagedHeadElement(doc js.Value, head js.Value, selector string, tag string, initialize func(js.Value)) js.Value {
	managedSelector := selector + `[` + managedMetadataAttr + `="` + managedMetadataValue + `"]`
	if element := findManagedHeadElement(doc, managedSelector); element.Truthy() {
		return element
	}
	matches := querySelectorAll(doc, selector)
	if len(matches) == 1 {
		matches[0].Call("setAttribute", managedMetadataAttr, managedMetadataValue)
		if initialize != nil {
			initialize(matches[0])
		}
		return matches[0]
	}
	element := doc.Call("createElement", tag)
	element.Call("setAttribute", managedMetadataAttr, managedMetadataValue)
	if initialize != nil {
		initialize(element)
	}
	head.Call("appendChild", element)
	return element
}

func findManagedHeadElement(doc js.Value, selector string) js.Value {
	matches := querySelectorAll(doc, selector)
	if len(matches) == 0 {
		return js.Null()
	}
	for _, extra := range matches[1:] {
		removeElement(extra)
	}
	return matches[0]
}

func querySelectorAll(doc js.Value, selector string) []js.Value {
	list := doc.Call("querySelectorAll", selector)
	length := list.Get("length").Int()
	if length == 0 {
		return nil
	}
	matches := make([]js.Value, 0, length)
	for index := 0; index < length; index++ {
		node := list.Call("item", index)
		if !node.Truthy() {
			node = list.Index(index)
		}
		if node.Truthy() {
			matches = append(matches, node)
		}
	}
	return matches
}

func removeElement(node js.Value) {
	if !node.Truthy() {
		return
	}
	if node.Get("remove").Truthy() {
		node.Call("remove")
		return
	}
	parent := node.Get("parentNode")
	if parent.Truthy() {
		parent.Call("removeChild", node)
	}
}

func getHeadElement(doc js.Value) js.Value {
	head := doc.Get("head")
	if head.Truthy() {
		return head
	}
	return doc.Call("querySelector", "head")
}

func (r *Router) evaluateNavigation(target string) (string, bool) {
	ctx, attemptID := r.beginGuardAttempt()
	defer r.finishGuardAttempt(attemptID)
	return r.evaluateNavigationWithAttempt(ctx, attemptID, target)
}

func (r *Router) evaluateNavigationWithAttempt(ctx context.Context, attemptID uint64, target string) (string, bool) {
	currentPath := r.GetCurrentRouterPath()
	currentQuery := getCurrentQueryValues()
	currentResolved := r.resolveRouteStack(currentPath)

	nextTarget := target
	for steps := 0; steps < 4; steps++ {
		if ctx.Err() != nil || !r.guardAttemptActive(attemptID) {
			return "", false
		}
		nextPath, nextQuery := parseNavigationTarget(nextTarget)
		nextResolved := r.resolveRouteStack(nextPath)
		nextLeaf := resolvedRoute{}
		if nextResolved.found {
			nextLeaf = nextResolved.routes[len(nextResolved.routes)-1]
		}
		nextCtx := r.routeContext(nextPath, nextLeaf.params, nextQuery)
		redirected := false

		if steps == 0 && currentResolved.found {
			for index := len(currentResolved.routes) - 1; index >= 0; index-- {
				currentRoute := currentResolved.routes[index]
				if currentRoute.option.BeforeLeave == nil && currentRoute.option.BeforeLeaveAsync == nil {
					continue
				}
				decision := guardDecisionAllowed()
				currentCtx := r.routeContext(currentRoute.path, currentRoute.params, currentQuery)
				if currentRoute.option.BeforeLeave != nil {
					decision = guardDecisionFromResult(currentRoute.option.BeforeLeave(currentCtx, nextCtx))
				}
				if !decision.Blocked && decision.Redirect == "" && currentRoute.option.BeforeLeaveAsync != nil {
					decision = currentRoute.option.BeforeLeaveAsync(ctx, currentCtx, nextCtx)
				}
				if ctx.Err() != nil || !r.guardAttemptActive(attemptID) {
					return "", false
				}
				if redirect := strings.TrimSpace(decision.Redirect); redirect != "" {
					nextTarget = normalizeNavigationTarget(redirect)
					runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "before-leave redirected navigation", "", map[string]string{
						"from": currentRoute.path,
						"to":   nextTarget,
					})
					redirected = true
					break
				}
				if decision.Blocked || decision.Denied {
					runtime.ReportLogWithFields("router", runtime.LogWarn, runtime.DiagnosticRecovered, "before-leave blocked navigation", "", map[string]string{
						"from": currentRoute.path,
						"to":   nextPath,
					})
					return "", false
				}
			}
			if redirected {
				continue
			}
		}

		if nextResolved.found {
			for _, nextRoute := range nextResolved.routes {
				if nextRoute.option.BeforeEnter == nil && nextRoute.option.BeforeEnterAsync == nil {
					continue
				}
				decision := guardDecisionAllowed()
				nextRouteCtx := r.routeContext(nextRoute.path, nextRoute.params, nextQuery)
				if nextRoute.option.BeforeEnter != nil {
					decision = guardDecisionFromResult(nextRoute.option.BeforeEnter(nextRouteCtx))
				}
				if !decision.Blocked && decision.Redirect == "" && nextRoute.option.BeforeEnterAsync != nil {
					decision = nextRoute.option.BeforeEnterAsync(ctx, nextRouteCtx)
				}
				if ctx.Err() != nil || !r.guardAttemptActive(attemptID) {
					return "", false
				}
				if redirect := strings.TrimSpace(decision.Redirect); redirect != "" {
					nextTarget = normalizeNavigationTarget(redirect)
					runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "before-enter redirected navigation", "", map[string]string{
						"from": nextPath,
						"to":   nextTarget,
					})
					redirected = true
					break
				}
				if decision.Blocked || decision.Denied {
					runtime.ReportLogWithFields("router", runtime.LogWarn, runtime.DiagnosticRecovered, "before-enter blocked navigation", "", map[string]string{
						"path": nextPath,
					})
					return "", false
				}
			}
			if redirected {
				continue
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

// PreserveReturnTo normalizes an internal path plus query values into a bounded
// return-target payload suitable for auth or re-auth redirects.
func PreserveReturnTo(path string, query url.Values) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return rootRoutePath
	}
	if parsed, err := url.Parse(trimmed); err == nil {
		if parsed.IsAbs() || parsed.Host != "" {
			return rootRoutePath
		}
	}
	if strings.HasPrefix(trimmed, "//") {
		return rootRoutePath
	}
	target := buildPathWithQuery(trimmed, copyQueryValues(query))
	if strings.TrimSpace(target) == "" || len(target) > maxReturnToLength {
		return rootRoutePath
	}
	return target
}

// ReadReturnTo reads an internal return-target query value and falls back when
// the value is empty, oversized, or external.
func ReadReturnTo(query url.Values, fallback string) string {
	fallbackTarget := normalizeNavigationTarget(fallback)
	raw := strings.TrimSpace(copyQueryValues(query).Get(ReturnToParam))
	if raw == "" || len(raw) > maxReturnToLength {
		return fallbackTarget
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(raw, "//") {
		return fallbackTarget
	}
	return normalizeNavigationTarget(raw)
}

// AllowNavigation permits the pending navigation.
func AllowNavigation() GuardResult {
	return GuardResult{}
}

// BlockNavigation blocks the pending navigation with a reason.
func BlockNavigation(reason string) GuardResult {
	return GuardResult{Blocked: true, Reason: reason}
}

// RedirectNavigation redirects the pending navigation to path.
func RedirectNavigation(path string) GuardResult {
	return GuardResult{Redirect: path}
}

func guardDecisionFromResult(result GuardResult) GuardDecision {
	return GuardDecision{
		Redirect: result.Redirect,
		Blocked:  result.Blocked,
		Reason:   result.Reason,
		Denied:   result.Blocked,
	}
}

func guardDecisionAllowed() GuardDecision {
	return GuardDecision{}
}

func guardDecisionBlocked(reason string) GuardDecision {
	return GuardDecision{Blocked: true, Reason: reason, Denied: true}
}

func guardDecisionRedirect(path string) GuardDecision {
	return GuardDecision{Redirect: path}
}

func (r *Router) beginGuardAttempt() (context.Context, uint64) {
	r.guardState.mu.Lock()
	if r.guardState.cancel != nil {
		r.guardState.cancel()
	}
	r.guardState.seq++
	id := r.guardState.seq
	ctx, cancel := context.WithCancel(context.Background())
	r.guardState.active = id
	r.guardState.cancel = cancel
	r.guardState.mu.Unlock()
	return ctx, id
}

func (r *Router) finishGuardAttempt(id uint64) {
	r.guardState.mu.Lock()
	defer r.guardState.mu.Unlock()
	if r.guardState.active != id {
		return
	}
	if r.guardState.cancel != nil {
		r.guardState.cancel()
	}
	r.guardState.active = 0
	r.guardState.cancel = nil
}

func (r *Router) guardAttemptActive(id uint64) bool {
	r.guardState.mu.Lock()
	defer r.guardState.mu.Unlock()
	return r.guardState.active == id && id != 0
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
	entry := r.loaderState.entries[key]
	if entry != nil {
		state := struct {
			pending bool
			data    Attrs
			err     error
		}{
			pending: entry.pending,
			data:    copyAttrs(entry.data),
			err:     entry.err,
		}
		r.loaderState.mu.Unlock()
		return state
	}

	ctx, cancel := context.WithCancel(context.Background())
	entry = &loaderEntry{
		pending: true,
		cancel:  cancel,
	}
	entry.version++
	version := entry.version
	r.loaderState.entries[key] = entry
	r.loaderState.mu.Unlock()

	runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "route loader started", "", map[string]string{
		"key":  key,
		"path": routeCtx.Path,
	})

	go func() {
		data, err := loader(ctx, routeCtx)

		r.loaderState.mu.Lock()
		defer r.loaderState.mu.Unlock()
		current := r.loaderState.entries[key]
		if ctx.Err() != nil || current == nil || current != entry || version != current.version {
			return
		}
		current.pending = false
		current.data = copyAttrs(data)
		current.err = err
		current.cancel = nil
		if err != nil {
			runtime.ReportLogWithFields("router", runtime.LogError, runtime.DiagnosticCorrectness, "route loader failed", "", map[string]string{
				"key":   key,
				"path":  routeCtx.Path,
				"error": err.Error(),
			})
		} else {
			runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "route loader resolved", "", map[string]string{
				"key":  key,
				"path": routeCtx.Path,
			})
		}

		go func() {
			doc := js.Global().Get("document")
			elem := js.Global().Get("Element")
			if doc.IsUndefined() || doc.IsNull() || elem.IsUndefined() || elem.IsNull() {
				return
			}
			r.renderCurrentRoute(false)
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
	for key, entry := range r.loaderState.entries {
		if entry != nil && entry.cancel != nil {
			entry.cancel()
		}
		delete(r.loaderState.entries, key)
	}
	r.loaderState.active = make(map[string]struct{})
}

func buildLoaderKey(routeID, resolvedPath, queryKey string) string {
	base := routeID + "@" + normalizePath(resolvedPath)
	if queryKey == "" {
		return base
	}
	return base + "?" + queryKey
}

func renderRouteFallback(component interface{}, props Attrs) *Element {
	if component != nil {
		return makeRouteFactory(component)(props)
	}
	return runtime.Div(nil, runtime.Text(routeLoadingText))
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

func (r *Router) matchPattern(path string) (routeFactory, map[string]string, Options, string, bool) {
	for _, pattern := range r.patterns {
		if params, ok := matchRoutePattern(pattern.pattern, path); ok {
			return pattern.factory, params, pattern.options, pattern.pattern, true
		}
	}
	return nil, nil, Options{}, "", false
}

func (r *Router) prepareLoaderState(activeKeys []string) {
	r.loaderState.mu.Lock()
	defer r.loaderState.mu.Unlock()
	nextActive := make(map[string]struct{}, len(activeKeys))
	for _, key := range activeKeys {
		nextActive[key] = struct{}{}
	}
	for key, entry := range r.loaderState.entries {
		if _, keep := nextActive[key]; keep {
			continue
		}
		if entry != nil && entry.cancel != nil {
			entry.cancel()
		}
		delete(r.loaderState.entries, key)
	}
	r.loaderState.active = nextActive
}

func routeIDExact(path string) string {
	return "exact:" + path
}

func routeIDPattern(pattern string) string {
	return "pattern:" + pattern
}

func routeIDNotFound() string {
	return "notfound:*"
}

func expandPathPrefixes(path string) []string {
	parts := splitPath(path)
	if len(parts) == 0 {
		return []string{"/"}
	}
	prefixes := make([]string, 0, len(parts)+1)
	prefixes = append(prefixes, "/")
	for index := range parts {
		prefixes = append(prefixes, "/"+strings.Join(parts[:index+1], "/"))
	}
	return prefixes
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
