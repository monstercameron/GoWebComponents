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
	"time"

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
	debugState     routeDebugState
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

type routeDebugState struct {
	mu           sync.Mutex
	lastRedirect RouteRedirectInspection
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
	Path         string
	Query        url.Values
	Params       map[string]string
	Loading      bool
	Stack        []RouteStackInspection
	Loaders      []RouteLoaderInspection
	LastRedirect RouteRedirectInspection
	Metadata     Metadata
}

type RouteStackInspection struct {
	ID             string
	Path           string
	Params         map[string]string
	HasLoader      bool
	HasBeforeEnter bool
	HasBeforeLeave bool
	Metadata       Metadata
}

type RouteLoaderInspection struct {
	Key     string
	Path    string
	Pending bool
	HasData bool
	Error   string
}

type RouteRedirectInspection struct {
	Cause string
	From  string
	To    string
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

// normalizePath is an internal router helper.
func normalizePath(parsePath string) string {
	parseTrimmed := strings.TrimSpace(parsePath)
	if parseTrimmed == "" || parseTrimmed == "#" {
		return rootRoutePath
	}
	if parseTrimmed == catchAllRoutePath {
		return catchAllRoutePath
	}

	parseTrimmed = strings.TrimPrefix(parseTrimmed, "#")
	if parseIdx := strings.Index(parseTrimmed, "?"); parseIdx >= 0 {
		parseTrimmed = parseTrimmed[:parseIdx]
	}
	if parseTrimmed == "" {
		return rootRoutePath
	}
	if !strings.HasPrefix(parseTrimmed, "/") {
		parseTrimmed = rootRoutePath + parseTrimmed
	}
	if len(parseTrimmed) > 1 {
		parseTrimmed = strings.TrimRight(parseTrimmed, "/")
		if parseTrimmed == "" {
			return rootRoutePath
		}
	}
	return parseTrimmed
}

// normalizeNavigationTarget is an internal router helper.
func normalizeNavigationTarget(parseTarget string) string {
	parseTrimmed := strings.TrimSpace(parseTarget)
	if parseTrimmed == "" || parseTrimmed == "#" {
		return rootRoutePath
	}

	parseTrimmed = strings.TrimPrefix(parseTrimmed, "#")
	parseQuery := ""
	if parseIdx := strings.Index(parseTrimmed, "?"); parseIdx >= 0 {
		parseQuery = parseTrimmed[parseIdx:]
		parseTrimmed = parseTrimmed[:parseIdx]
	}

	parseNormalized := normalizePath(parseTrimmed)
	if parseQuery == "?" {
		parseQuery = ""
	}
	return parseNormalized + parseQuery
}

var routerRuntimeInitialized bool
var currentParams = map[string]string{}
var currentRouteData Attrs
var currentRouteOutlet *Element

// NewHashRouter creates a hash-based router that reads from window.location.hash.
func NewHashRouter(parseOptions ...RouterOptions) *Router {
	parseCfg := RouterOptions{DefaultRoute: rootRoutePath}
	if len(parseOptions) > 0 {
		parseCfg = parseOptions[0]
	}
	parseCfg.DefaultRoute = normalizePath(parseCfg.DefaultRoute)

	return &Router{
		routes:       make(map[string]routeFactory),
		routeOptions: make(map[string]Options),
		patterns:     []routePattern{},
		defaultRoute: parseCfg.DefaultRoute,
		routerType:   routerTypeHash,
		loaderState: loaderState{
			entries: make(map[string]*loaderEntry),
			active:  make(map[string]struct{}),
		},
	}
}

// NewHistoryRouter creates a history-based router using the HTML5 History API.
// This router uses window.location.pathname instead of hash fragments.
// Requires server to redirect all routes to the app's entry point.
func NewHistoryRouter(parseOptions ...RouterOptions) *Router {
	// Use provided options or defaults
	var parseCfg RouterOptions
	if len(parseOptions) > 0 {
		parseCfg = parseOptions[0]
	}
	if parseCfg.DefaultRoute == "" {
		parseCfg.DefaultRoute = rootRoutePath
	}
	parseCfg.DefaultRoute = normalizePath(parseCfg.DefaultRoute)

	parseRouter := &Router{
		routes:       make(map[string]routeFactory),
		routeOptions: make(map[string]Options),
		patterns:     []routePattern{},
		defaultRoute: parseCfg.DefaultRoute,
		routerType:   routerTypeHistory,
		loaderState: loaderState{
			entries: make(map[string]*loaderEntry),
			active:  make(map[string]struct{}),
		},
	}

	// Setup browser sync for history-based navigation
	parseRouter.setupHistoryListener()

	return parseRouter
}

// setupHistoryListener sets up History API listeners for popstate events
func (parseR *Router) setupHistoryListener() {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() || !parseWindow.Get("addEventListener").Truthy() {
		return
	}

	// Handler for browser back/forward buttons
	parsePopstateHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseR.renderCurrentRoute(true)
		return nil
	})

	parseWindow.Call("addEventListener", browserEventPop, parsePopstateHandler)

	// Clean up on unload
	registerCleanup(parsePopstateHandler)
}

// GoRegisterRoute registers a route on the router instance.
func (parseR *Router) GoRegisterRoute(parsePath string, parseComponent interface{}, parseOptions ...Options) {
	parseR.Register(parsePath, parseComponent, parseOptions...)
}

// Register registers a route using either a component function or a static node.
func (parseR *Router) Register(parsePath string, parseComponent interface{}, parseOptions ...Options) {
	if parseR.routes == nil {
		parseR.routes = make(map[string]routeFactory)
	}
	if parseR.routeOptions == nil {
		parseR.routeOptions = make(map[string]Options)
	}
	if parseR.patterns == nil {
		parseR.patterns = []routePattern{}
	}

	parseNormalize := normalizePath(parsePath)
	parseFactory := makeRouteFactory(parseComponent)
	parseOption := Options{}
	if len(parseOptions) > 0 {
		parseOption = parseOptions[0]
	}

	if parseNormalize == catchAllRoutePath {
		if parseR.notFound != nil {
			runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "replacing existing catch-all route registration for *")
		}
		parseR.notFound = parseFactory
		parseR.notFoundOption = parseOption
		return
	}

	if isPatternRoute(parseNormalize) {
		for parseIndex, parsePattern := range parseR.patterns {
			if parsePattern.pattern == parseNormalize {
				runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "replacing existing pattern route registration for "+parseNormalize)
				parseR.patterns[parseIndex] = routePattern{pattern: parseNormalize, factory: parseFactory, options: parseOption}
				return
			}
		}
		parseR.patterns = append(parseR.patterns, routePattern{pattern: parseNormalize, factory: parseFactory, options: parseOption})
		return
	}

	if _, parseExists := parseR.routes[parseNormalize]; parseExists {
		runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "replacing existing route registration for "+parseNormalize)
	}
	parseR.routes[parseNormalize] = parseFactory
	parseR.routeOptions[parseNormalize] = parseOption
}

// GoGetRoute returns the element for the current route.
func (parseR *Router) GoGetRoute() *Element {
	return parseR.Current()
}

// Current returns the current route element.
func (parseR *Router) Current() *Element {
	return parseR.currentElement(true)
}

// currentElement is an internal router helper.
func (parseR *Router) currentElement(isApplyGuards bool) *Element {
	globalRouter = parseR
	parsePath := parseR.GetCurrentRouterPath()
	if parsePath == "" {
		parsePath = parseR.defaultRoute
	}
	parseQuery := getCurrentQueryValues()
	parseQueryKey := parseQuery.Encode()
	parseResolved := parseR.resolveRouteStack(parsePath)
	if parseResolved.found {
		parseLeaf := parseResolved.routes[len(parseResolved.routes)-1]
		currentParams = copyParams(parseLeaf.params)
		if isApplyGuards {
			parseCtx, parseAttemptID := parseR.beginGuardAttempt()
			defer parseR.finishGuardAttempt(parseAttemptID)
			parseRendered := parseR.renderResolvedRouteStack(parseResolved.routes, parseQuery, parseQueryKey, true, parseCtx, parseAttemptID)
			if parseCtx.Err() != nil || !parseR.guardAttemptActive(parseAttemptID) {
				return nil
			}
			return parseRendered
		}
		return parseR.renderResolvedRouteStack(parseResolved.routes, parseQuery, parseQueryKey, false, nil, 0)
	}

	parseR.cancelLoaderIfActive()
	currentRouteData = nil
	currentRouteOutlet = nil

	return runtime.Div(nil, runtime.Text(routeNotFoundText))
}

// Mount renders the router into a DOM node selected by CSS selector and wires hashchange listeners.
func (parseR *Router) Mount(parseSelector string) {
	globalRouter = parseR
	parseR.targetSelector = parseSelector
	parseR.targetElement = js.Null()
	parseR.renderCurrentRoute(true)
	parseR.ensureListener()
}

// HydrateMount binds the router to an already-hydrated DOM target and only
// wires future route updates/listeners without forcing an immediate rerender.
func (parseR *Router) HydrateMount(parseSelector string) {
	globalRouter = parseR
	parseR.targetSelector = parseSelector
	parseR.targetElement = js.Null()
	parseR.ensureListener()
}

// MountElement renders the router into an existing DOM element reference.
func (parseR *Router) MountElement(parseElem js.Value) {
	globalRouter = parseR
	parseR.targetElement = parseElem
	parseR.targetSelector = ""
	parseR.renderCurrentRoute(true)
	parseR.ensureListener()
}

// HydrateMountElement binds the router to an already-hydrated DOM element and
// only wires future route updates/listeners without forcing an immediate rerender.
func (parseR *Router) HydrateMountElement(parseElem js.Value) {
	globalRouter = parseR
	parseR.targetElement = parseElem
	parseR.targetSelector = ""
	parseR.ensureListener()
}

// Revalidate clears the cached result for the current route loader and runs it again.
func (parseR *Router) Revalidate() {
	parseR.cancelLoaderIfActive()
	parseR.renderCurrentRoute(false)
}

// IsLoading reports whether the current route loader is pending.
func (parseR *Router) IsLoading() bool {
	parseR.loaderState.mu.Lock()
	defer parseR.loaderState.mu.Unlock()
	for parseKey := range parseR.loaderState.active {
		parseEntry := parseR.loaderState.entries[parseKey]
		if parseEntry != nil && parseEntry.pending {
			return true
		}
	}
	return false
}

// renderCurrentRoute is an internal router helper.
func (parseR *Router) renderCurrentRoute(isApplyGuards bool) {
	parseStart := time.Now()
	parsePath := parseR.GetCurrentRouterPath()
	runtime.RecordStartupRouteContext(parsePath)
	defer func() {
		runtime.ReportProfilingEvent("router", "route.lifecycle", "finish", parsePath, time.Since(parseStart).Nanoseconds(), map[string]string{
			"apply_guards": strconv.FormatBool(isApplyGuards),
			"kind":         parseR.routerType,
			"loading":      strconv.FormatBool(parseR.IsLoading()),
		})
	}()
	ensureInitialized()
	parseRt := runtime.GetGlobalRuntime()
	parseRouteElement := parseR.currentElement(isApplyGuards)
	if parseRouteElement == nil {
		return
	}
	switch {
	case parseR.targetSelector != "":
		parseRt.RenderTo(parseR.targetSelector, parseRouteElement)
	case parseR.targetElement.Truthy():
		parseRt.Render(parseRouteElement, jsdom.NewWASMDOMNode(parseR.targetElement))
	}
}

// ensureListener is an internal router helper.
func (parseR *Router) ensureListener() {
	if parseR.listening {
		return
	}
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() || !parseWindow.Get("addEventListener").Truthy() {
		return
	}
	parseR.listening = true

	parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseR.renderCurrentRoute(true)
		return nil
	})
	parseWindow.Call("addEventListener", browserEventHash, parseHandler)
	registerCleanup(parseHandler)
}

// RegisterRoute registers a route with the global router.
func RegisterRoute(parsePath string, parseComponent interface{}, parseOptions ...Options) {
	GetRouter().Register(parsePath, parseComponent, parseOptions...)
}

// GetCurrentRouterPath returns the current path from a router instance based on its type.
func (parseR *Router) GetCurrentRouterPath() string {
	parseLoc := getLocationValue()
	if !parseLoc.Truthy() {
		return rootRoutePath
	}

	if parseR.routerType == routerTypeHistory {
		// For history router, use pathname
		return normalizePath(parseLoc.Get("pathname").String())
	}

	// For hash router, use hash fragment
	return normalizePath(parseLoc.Get("hash").String())
}

// Navigate navigates to a path using the appropriate method for this router type.
func (parseR *Router) Navigate(parsePath string) {
	parseCtx, parseAttemptID := parseR.beginGuardAttempt()
	defer parseR.finishGuardAttempt(parseAttemptID)

	parseNormalized, parseOk := parseR.evaluateNavigationWithAttempt(parseCtx, parseAttemptID, normalizeNavigationTarget(parsePath))
	if !parseOk {
		return
	}
	if parseCtx.Err() != nil || !parseR.guardAttemptActive(parseAttemptID) {
		return
	}
	runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "navigation started", "", map[string]string{
		"target": parseNormalized,
		"mode":   "push",
		"kind":   parseR.routerType,
	})
	runtime.ReportProfilingEvent("router", "navigation", "start", parseNormalized, 0, map[string]string{
		"mode": "push",
		"kind": parseR.routerType,
	})
	if parseR.routerType == routerTypeHistory {
		// For history router, use pushState
		parseHistory := getHistoryValue()
		if parseHistory.Truthy() && parseHistory.Get("pushState").Truthy() {
			parseHistory.Call("pushState", nil, "", parseNormalized)
		} else if parseLoc := getLocationValue(); parseLoc.Truthy() {
			parseLoc.Set("pathname", parseNormalized)
		}
	} else {
		// Hash routers re-render through the hashchange listener.
		if parseLoc2 := getLocationValue(); parseLoc2.Truthy() {
			parseLoc2.Set("hash", parseNormalized)
		}
	}
	if parseR.routerType == routerTypeHistory {
		if parseCtx.Err() != nil || !parseR.guardAttemptActive(parseAttemptID) {
			return
		}
		parseR.renderCurrentRoute(false)
	}
}

// NavigateReplace replaces the current history entry using the appropriate method for this router type.
func (parseR *Router) NavigateReplace(parsePath string) {
	parseCtx, parseAttemptID := parseR.beginGuardAttempt()
	defer parseR.finishGuardAttempt(parseAttemptID)

	parseNormalized, parseOk := parseR.evaluateNavigationWithAttempt(parseCtx, parseAttemptID, normalizeNavigationTarget(parsePath))
	if !parseOk {
		return
	}
	if parseCtx.Err() != nil || !parseR.guardAttemptActive(parseAttemptID) {
		return
	}
	runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "navigation started", "", map[string]string{
		"target": parseNormalized,
		"mode":   "replace",
		"kind":   parseR.routerType,
	})
	runtime.ReportProfilingEvent("router", "navigation", "start", parseNormalized, 0, map[string]string{
		"mode": "replace",
		"kind": parseR.routerType,
	})
	if parseR.routerType == routerTypeHistory {
		// For history router, use replaceState
		parseHistory := getHistoryValue()
		if parseHistory.Truthy() && parseHistory.Get("replaceState").Truthy() {
			parseHistory.Call("replaceState", nil, "", parseNormalized)
		} else if parseLoc := getLocationValue(); parseLoc.Truthy() {
			parseLoc.Set("pathname", parseNormalized)
		}
	} else {
		// Hash routers re-render through the hashchange listener.
		parseLoc2 := getLocationValue()
		if parseLoc2.Truthy() {
			if parseLoc2.Get("replace").Truthy() {
				parseLoc2.Call("replace", "#"+strings.TrimPrefix(parseNormalized, "#"))
			} else {
				parseLoc2.Set("hash", parseNormalized)
			}
		}
	}
	if parseR.routerType == routerTypeHistory {
		if parseCtx.Err() != nil || !parseR.guardAttemptActive(parseAttemptID) {
			return
		}
		parseR.renderCurrentRoute(false)
	}
}

// Navigate updates the URL using the appropriate method for the current router.
func Navigate(parsePath string) {
	GetRouter().Navigate(parsePath)
}

// NavigateReplace replaces the current history entry using the appropriate method for the current router.
func NavigateReplace(parsePath string) {
	GetRouter().NavigateReplace(parsePath)
}

// Revalidate clears the current route loader result and runs the route again.
func Revalidate() {
	GetRouter().Revalidate()
}

// GetCurrentPath returns the current route path from the global router.
func GetCurrentPath() string {
	return GetRouter().GetCurrentRouterPath()
}

// InspectCurrentRoute returns the current path, query, params, and loading state.
func InspectCurrentRoute() RouteInspection {
	return GetRouter().inspectCurrentRoute()
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
		revalidate: Revalidate,
		loading: func() bool {
			return GetRouter().IsLoading()
		},
	}
}

// Navigate pushes a new route onto the history stack.
func (parseN Navigator) Navigate(parsePath string) {
	if parseN.navigate != nil {
		parseN.navigate(parsePath)
	}
}

// Replace replaces the current route entry.
func (parseN Navigator) Replace(parsePath string) {
	if parseN.replace != nil {
		parseN.replace(parsePath)
	}
}

// Revalidate forces the current route loader to run again.
func (parseR Revalidator) Revalidate() {
	if parseR.revalidate != nil {
		parseR.revalidate()
	}
}

// Loading reports whether the current route loader is still pending.
func (parseR Revalidator) Loading() bool {
	if parseR.loading == nil {
		return false
	}
	return parseR.loading()
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
	parseValues := copyQueryValues(getCurrentQueryValues())
	return SearchParams{
		values: parseValues,
		navigate: func(parseNext url.Values) {
			Navigate(buildPathWithQuery(GetCurrentPath(), parseNext))
		},
		replace: func(parseNext2 url.Values) {
			NavigateReplace(buildPathWithQuery(GetCurrentPath(), parseNext2))
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

// GetOutlet returns the child route element for the current layout route, if one exists.
func GetOutlet() *Element {
	return currentRouteOutlet
}

// Get returns the first value for a query key or an empty string.
func (parseQ Query) Get(parseKey string) string {
	if parseQ.values == nil {
		return ""
	}
	return parseQ.values.Get(parseKey)
}

// Has reports whether a query key is present.
func (parseQ Query) Has(parseKey string) bool {
	if parseQ.values == nil {
		return false
	}
	_, parseOk := parseQ.values[parseKey]
	return parseOk
}

// Values returns a copy of the parsed query values.
func (parseQ Query) Values() url.Values {
	parseClone := make(url.Values, len(parseQ.values))
	for parseKey, parseValues := range parseQ.values {
		parseClone[parseKey] = append([]string(nil), parseValues...)
	}
	return parseClone
}

// Encode serializes the current query values using net/url encoding.
func (parseQ Query) Encode() string {
	if parseQ.values == nil {
		return ""
	}
	return parseQ.values.Encode()
}

// Get returns the first value for a query key or an empty string.
func (parseS SearchParams) Get(parseKey string) string {
	return Query{values: parseS.values}.Get(parseKey)
}

// Has reports whether a query key is present.
func (parseS SearchParams) Has(parseKey string) bool {
	return Query{values: parseS.values}.Has(parseKey)
}

// Values returns a copy of the current search params.
func (parseS SearchParams) Values() url.Values {
	return Query{values: parseS.values}.Values()
}

// Encode serializes the current search params using net/url encoding.
func (parseS SearchParams) Encode() string {
	return Query{values: parseS.values}.Encode()
}

// Set pushes a navigation update with key assigned to value.
func (parseS SearchParams) Set(parseKey, parseValue string) {
	if parseS.navigate == nil {
		return
	}
	parseNext := parseS.Values()
	parseNext.Set(parseKey, parseValue)
	parseS.navigate(parseNext)
}

// Delete pushes a navigation update with key removed.
func (parseS SearchParams) Delete(parseKey string) {
	if parseS.navigate == nil {
		return
	}
	parseNext := parseS.Values()
	parseNext.Del(parseKey)
	parseS.navigate(parseNext)
}

// Replace updates the current history entry with key assigned to value.
func (parseS SearchParams) Replace(parseKey, parseValue string) {
	if parseS.replace == nil {
		return
	}
	parseNext := parseS.Values()
	parseNext.Set(parseKey, parseValue)
	parseS.replace(parseNext)
}

// Navigate applies the provided query values as a pushed navigation update.
func (parseS SearchParams) Navigate(parseValues url.Values) {
	if parseS.navigate != nil {
		parseS.navigate(copyQueryValues(parseValues))
	}
}

// ReplaceAll replaces the current history entry with the provided query values.
func (parseS SearchParams) ReplaceAll(parseValues url.Values) {
	if parseS.replace != nil {
		parseS.replace(copyQueryValues(parseValues))
	}
}

// Get returns the captured route param for key or an empty string.
func (parseP Params) Get(parseKey string) string {
	if parseP.values == nil {
		return ""
	}
	return parseP.values[parseKey]
}

// Has reports whether the route param key exists.
func (parseP Params) Has(parseKey string) bool {
	if parseP.values == nil {
		return false
	}
	_, parseOk := parseP.values[parseKey]
	return parseOk
}

// Values returns a copy of the current route params.
func (parseP Params) Values() map[string]string {
	return copyParams(parseP.values)
}

// Int parses the route param for key as an int.
func (parseP Params) Int(parseKey string) (int, bool) {
	parseRaw := parseP.Get(parseKey)
	if parseRaw == "" {
		return 0, false
	}

	parseValue, parseErr := strconv.Atoi(strings.TrimSpace(parseRaw))
	if parseErr != nil {
		return 0, false
	}
	return parseValue, true
}

// Bool parses the route param for key as a bool.
func (parseP Params) Bool(parseKey string) (bool, bool) {
	parseRaw := parseP.Get(parseKey)
	if parseRaw == "" {
		return false, false
	}

	parseValue, parseErr := strconv.ParseBool(strings.TrimSpace(parseRaw))
	if parseErr != nil {
		return false, false
	}
	return parseValue, true
}

// GetRoute returns the component for the current route as an Element.
func GetRoute() *Element {
	return GetRouter().Current()
}

// GetRouter returns the global router instance.
func GetRouter() *Router {
	return globalRouter
}

// RegisterElementRoute renders a route directly to a DOM element and sets up hash listening.
func RegisterElementRoute(parsePath string, parseElemRef js.Value) {
	Navigate(parsePath)
	globalRouter.MountElement(parseElemRef)
}

var globalRouter = NewHashRouter()
var cleanupOnce sync.Once

// ensureInitialized is an internal router helper.
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

// registerCleanup is an internal router helper.
func registerCleanup(parseHandler js.Func) {
	cleanupOnce.Do(func() {
		parseWindow := js.Global().Get("window")
		if !parseWindow.Truthy() || !parseWindow.Get("addEventListener").Truthy() {
			return
		}
		// Register unload listener to release the hashchange handler to avoid leaks in hot reload.
		parseUnload := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseHandler.Release()
			return nil
		})
		parseWindow.Call("addEventListener", "beforeunload", parseUnload)
	})
}

// makeRouteFactory is an internal router helper.
func makeRouteFactory(parseComponent interface{}) routeFactory {
	if parseComponent == nil {
		panic(runtime.ActionableFrameworkPanic(runtime.ActionablePanicOptions{
			Source:  "router",
			Subject: "router.Register",
			Message: "route component cannot be nil",
			Path:    "router.Register",
		}))
	}

	if parseElement, parseOk := parseComponent.(*Element); parseOk {
		return func(_ Attrs) *Element {
			return parseElement
		}
	}

	parseValue := reflect.ValueOf(parseComponent)
	if !parseValue.IsValid() || parseValue.Kind() != reflect.Func {
		panic(runtime.ActionableFrameworkPanic(runtime.ActionablePanicOptions{
			Source:  "router",
			Subject: "router.Register",
			Message: "unsupported route component type",
			Path:    "router.Register",
		}))
	}

	parseTyp := parseValue.Type()
	if parseTyp.NumOut() != 1 {
		panic(runtime.ActionableFrameworkPanic(runtime.ActionablePanicOptions{
			Source:  "router",
			Subject: "router.Register",
			Message: "route component must return exactly one element",
			Path:    "router.Register",
		}))
	}

	return func(parseAttrs Attrs) *Element {
		parseArgs := []reflect.Value{}
		if parseTyp.NumIn() == 1 {
			parseArg := reflect.Zero(parseTyp.In(0))
			if parseAttrs != nil {
				parseProvided := reflect.ValueOf(parseAttrs)
				switch {
				case parseProvided.IsValid() && parseProvided.Type() == parseTyp.In(0):
					parseArg = parseProvided
				case parseProvided.IsValid() && parseProvided.Type().AssignableTo(parseTyp.In(0)):
					parseArg = parseProvided
				case parseProvided.IsValid() && parseProvided.Type().ConvertibleTo(parseTyp.In(0)):
					parseArg = parseProvided.Convert(parseTyp.In(0))
				}
			}
			parseArgs = append(parseArgs, parseArg)
		}

		parseResults := parseValue.Call(parseArgs)
		if len(parseResults) == 0 || !parseResults[0].IsValid() || parseResults[0].IsNil() {
			return nil
		}

		parseElement2, _ := parseResults[0].Interface().(*Element)
		return parseElement2
	}
}

// getCurrentQueryValues is an internal router helper.
func getCurrentQueryValues() url.Values {
	parseLoc := getLocationValue()
	if !parseLoc.Truthy() {
		return url.Values{}
	}

	parseRaw := strings.TrimPrefix(parseLoc.Get("search").String(), "?")
	if parseRaw == "" {
		parseHash := strings.TrimPrefix(parseLoc.Get("hash").String(), "#")
		if parseIdx := strings.Index(parseHash, "?"); parseIdx >= 0 && parseIdx+1 < len(parseHash) {
			parseRaw = parseHash[parseIdx+1:]
		}
	}

	if parseRaw == "" {
		return url.Values{}
	}

	parseValues, parseErr := url.ParseQuery(parseRaw)
	if parseErr != nil {
		return url.Values{}
	}
	return parseValues
}

// buildPathWithQuery is an internal router helper.
func buildPathWithQuery(parsePath string, parseValues url.Values) string {
	parseNormalized := normalizePath(parsePath)
	parseEncoded := copyQueryValues(parseValues).Encode()
	if parseEncoded == "" {
		return parseNormalized
	}
	return parseNormalized + "?" + parseEncoded
}

// getLocationValue is an internal router helper.
func getLocationValue() js.Value {
	parseWindow := js.Global().Get("window")
	if parseWindow.Truthy() {
		parseLoc := parseWindow.Get("location")
		if parseLoc.Truthy() {
			return parseLoc
		}
	}
	return js.Global().Get("location")
}

// getHistoryValue is an internal router helper.
func getHistoryValue() js.Value {
	parseWindow := js.Global().Get("window")
	if parseWindow.Truthy() {
		parseHistory := parseWindow.Get("history")
		if parseHistory.Truthy() {
			return parseHistory
		}
	}
	return js.Global().Get("history")
}

// renderResolvedRouteStack is an internal router helper.
func (parseR *Router) renderResolvedRouteStack(parseRoutes []resolvedRoute, parseQuery url.Values, parseQueryKey string, isApplyGuards bool, parseGuardCtx context.Context, parseAttemptID uint64) *Element {
	parseLoaderKeys := make([]string, 0, len(parseRoutes))
	for _, parseRoute := range parseRoutes {
		if parseRoute.option.Loader != nil {
			parseLoaderKeys = append(parseLoaderKeys, buildLoaderKey(parseRoute.id, parseRoute.path, parseQueryKey))
		}
	}
	parseR.prepareLoaderState(parseLoaderKeys)
	return parseR.renderRouteLevel(parseRoutes, 0, parseQuery, parseQueryKey, isApplyGuards, parseGuardCtx, parseAttemptID)
}

// renderRouteLevel is an internal router helper.
func (parseR *Router) renderRouteLevel(parseRoutes []resolvedRoute, parseIndex int, parseQuery url.Values, parseQueryKey string, isApplyGuards bool, parseGuardCtx context.Context, parseAttemptID uint64) *Element {
	parseMatch := parseRoutes[parseIndex]
	if isApplyGuards {
		if parseBlocked := parseR.applyBeforeEnterGuard(parseMatch.path, parseMatch.option, parseMatch.params, parseQuery, parseGuardCtx, parseAttemptID); parseBlocked != nil {
			return parseBlocked
		}
	}
	if parseRedirected := parseR.applyRouteOptions(parseMatch.path, parseMatch.option, parseQuery); parseRedirected != nil {
		return parseRedirected
	}

	parseBaseProps := copyParamsToAttrs(parseMatch.params)
	parseData := Attrs(nil)
	if parseMatch.option.Loader != nil {
		parseLoaderKey := buildLoaderKey(parseMatch.id, parseMatch.path, parseQueryKey)
		parseState := parseR.ensureLoaderResult(parseLoaderKey, parseMatch.option.Loader, RouteContext{
			Path:   parseMatch.path,
			Params: Params{values: copyParams(parseMatch.params)},
			Query:  Query{values: copyQueryValues(parseQuery)},
		})

		if parseState.pending {
			currentRouteData = nil
			currentRouteOutlet = nil
			return renderRouteFallback(parseMatch.option.Loading, mergeAttrs(parseBaseProps, Attrs{"path": parseMatch.path, "loading": true}))
		}
		if parseState.err != nil {
			currentRouteData = nil
			currentRouteOutlet = nil
			return renderRouteError(parseMatch.option.Error, parseState.err, mergeAttrs(parseBaseProps, Attrs{"path": parseMatch.path, "error": parseState.err.Error()}))
		}

		parseData = copyAttrs(parseState.data)
		parseBaseProps = mergeAttrs(parseBaseProps, parseData)
	}

	var parseOutlet *Element
	if parseIndex+1 < len(parseRoutes) {
		parseOutlet = parseR.renderRouteLevel(parseRoutes, parseIndex+1, parseQuery, parseQueryKey, isApplyGuards, parseGuardCtx, parseAttemptID)
	}

	parsePrevParams, parsePrevData, parsePrevOutlet := withRouteRenderContext(parseMatch.params, parseData, parseOutlet)
	defer restoreRouteRenderContext(parsePrevParams, parsePrevData, parsePrevOutlet)

	return parseMatch.factory(parseBaseProps)
}

// withRouteRenderContext is an internal router helper.
func withRouteRenderContext(parseParams map[string]string, parseData Attrs, parseOutlet *Element) (map[string]string, Attrs, *Element) {
	parsePrevParams := currentParams
	parsePrevData := currentRouteData
	parsePrevOutlet := currentRouteOutlet
	currentParams = copyParams(parseParams)
	currentRouteData = copyAttrs(parseData)
	currentRouteOutlet = parseOutlet
	return parsePrevParams, parsePrevData, parsePrevOutlet
}

// restoreRouteRenderContext is an internal router helper.
func restoreRouteRenderContext(parseParams map[string]string, parseData Attrs, parseOutlet *Element) {
	currentParams = parseParams
	currentRouteData = parseData
	currentRouteOutlet = parseOutlet
}

// resolveRouteStack is an internal router helper.
func (parseR *Router) resolveRouteStack(parsePath string) resolvedRouteStack {
	parseLeaf := parseR.resolveRoute(parsePath)
	if !parseLeaf.found {
		return resolvedRouteStack{}
	}

	if parseLeaf.id == defaultRoutePrefix+parseR.defaultRoute {
		return resolvedRouteStack{routes: []resolvedRoute{parseLeaf}, found: true}
	}

	parsePrefixes := expandPathPrefixes(parsePath)
	parseRoutes := make([]resolvedRoute, 0, len(parsePrefixes)+1)
	parseSeen := map[string]struct{}{}
	for _, parsePrefix := range parsePrefixes {
		if parseComp, parseOk := parseR.routes[parsePrefix]; parseOk {
			parseOption := parseR.routeOptions[parsePrefix]
			parseId := routeIDExact(parsePrefix)
			if parseOption.Layout && parseId != parseLeaf.id {
				parseRoutes = append(parseRoutes, resolvedRoute{id: parseId, path: parsePrefix, params: map[string]string{}, option: parseOption, factory: parseComp, found: true})
				parseSeen[parseId] = struct{}{}
			}
		}

		for _, parsePattern := range parseR.patterns {
			if !parsePattern.options.Layout {
				continue
			}
			parseId2 := routeIDPattern(parsePattern.pattern)
			if parseId2 == parseLeaf.id {
				continue
			}
			if _, parseExists := parseSeen[parseId2]; parseExists {
				continue
			}
			parseParams, parseOk2 := matchRoutePattern(parsePattern.pattern, parsePrefix)
			if !parseOk2 {
				continue
			}
			parseRoutes = append(parseRoutes, resolvedRoute{id: parseId2, path: parsePrefix, params: copyParams(parseParams), option: parsePattern.options, factory: parsePattern.factory, found: true})
			parseSeen[parseId2] = struct{}{}
		}
	}

	parseRoutes = append(parseRoutes, parseLeaf)
	return resolvedRouteStack{routes: parseRoutes, found: true}
}

// resolveRoute is an internal router helper.
func (parseR *Router) resolveRoute(parsePath string) resolvedRoute {
	if parseComp, parseOk := parseR.routes[parsePath]; parseOk {
		return resolvedRoute{id: routeIDExact(parsePath), path: parsePath, params: map[string]string{}, option: parseR.routeOptions[parsePath], factory: parseComp, found: true}
	}
	if parseComp2, parseParams, parseOption, parsePattern, parseOk2 := parseR.matchPattern(parsePath); parseOk2 {
		return resolvedRoute{id: routeIDPattern(parsePattern), path: parsePath, params: copyParams(parseParams), option: parseOption, factory: parseComp2, found: true}
	}
	if parseR.notFound != nil {
		return resolvedRoute{id: routeIDNotFound(), path: parsePath, params: map[string]string{}, option: parseR.notFoundOption, factory: parseR.notFound, found: true}
	}
	if parseR.defaultRoute != "" {
		if parseComp3, parseOk3 := parseR.routes[parseR.defaultRoute]; parseOk3 {
			return resolvedRoute{id: defaultRoutePrefix + parseR.defaultRoute, path: parseR.defaultRoute, params: map[string]string{}, option: parseR.routeOptions[parseR.defaultRoute], factory: parseComp3, found: true}
		}
	}
	return resolvedRoute{}
}

// routeContext is an internal router helper.
func (parseR *Router) routeContext(parsePath string, parseParams map[string]string, parseQuery url.Values) RouteContext {
	return RouteContext{
		Path:   parsePath,
		Params: Params{values: copyParams(parseParams)},
		Query:  Query{values: copyQueryValues(parseQuery)},
	}
}

// applyBeforeEnterGuard is an internal router helper.
func (parseR *Router) applyBeforeEnterGuard(parsePath string, parseOption Options, parseParams map[string]string, parseQuery url.Values, parseGuardCtx context.Context, parseAttemptID uint64) *Element {
	if parseOption.BeforeEnter == nil && parseOption.BeforeEnterAsync == nil {
		return nil
	}
	parseRouteCtx := parseR.routeContext(parsePath, parseParams, parseQuery)
	parseDecision := guardDecisionAllowed()
	if parseOption.BeforeEnter != nil {
		parseDecision = guardDecisionFromResult(parseOption.BeforeEnter(parseRouteCtx))
	}
	if !parseDecision.Blocked && parseDecision.Redirect == "" && parseOption.BeforeEnterAsync != nil {
		if parseGuardCtx == nil {
			parseGuardCtx = context.Background()
		}
		parseDecision = parseOption.BeforeEnterAsync(parseGuardCtx, parseRouteCtx)
	}
	if (parseGuardCtx != nil && parseGuardCtx.Err() != nil) || (parseAttemptID != 0 && !parseR.guardAttemptActive(parseAttemptID)) {
		return nil
	}
	if parseTarget := strings.TrimSpace(parseDecision.Redirect); parseTarget != "" {
		parseNormalized := normalizeNavigationTarget(parseTarget)
		if parseNormalized == buildPathWithQuery(parsePath, parseQuery) {
			runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "ignoring route before-enter redirect loop for "+parseNormalized)
			return nil
		}
		parseR.recordRedirectDebug("before-enter", parsePath, parseNormalized)
		runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "before-enter redirected navigation", "", map[string]string{
			"from": parsePath,
			"to":   parseNormalized,
		})
		runtime.ReportProfilingEvent("router", "guard.before_enter", "redirect", parsePath, 0, map[string]string{
			"to": parseNormalized,
		})
		parseR.replaceLocation(parseNormalized)
		return parseR.currentElement(false)
	}
	if !parseDecision.Blocked && !parseDecision.Denied {
		return nil
	}
	currentRouteData = nil
	parseR.cancelLoaderIfActive()
	parseMessage := strings.TrimSpace(parseDecision.Reason)
	if parseMessage == "" {
		parseMessage = navigationBlocked
	}
	parseProps := mergeAttrs(copyParamsToAttrs(parseParams), Attrs{
		"path":         parsePath,
		"reason":       parseMessage,
		"blocked":      parseDecision.Blocked,
		"denied":       parseDecision.Denied,
		"retryable":    parseDecision.Retryable,
		"authorizing":  parseDecision.Retryable,
		"unauthorized": parseDecision.Denied,
	})
	if parseDecision.Retryable {
		if parseOption.Authorizing != nil {
			return renderRouteGuardState(parseOption.Authorizing, parseProps)
		}
		if parseOption.GuardPending != nil {
			return renderRouteGuardState(parseOption.GuardPending, parseProps)
		}
	}
	if parseDecision.Denied && parseOption.Unauthorized != nil {
		return renderRouteGuardState(parseOption.Unauthorized, parseProps)
	}
	if parseDecision.Blocked && parseOption.GuardPending != nil {
		return renderRouteGuardState(parseOption.GuardPending, parseProps)
	}
	runtime.ReportLogWithFields("router", runtime.LogWarn, runtime.DiagnosticRecovered, "before-enter blocked navigation", "", map[string]string{
		"path":   parsePath,
		"reason": parseMessage,
	})
	runtime.ReportProfilingEvent("router", "guard.before_enter", "blocked", parsePath, 0, map[string]string{
		"reason": parseMessage,
	})
	return runtime.Div(nil, runtime.Text(parseMessage))
}

// applyRouteOptions is an internal router helper.
func (parseR *Router) applyRouteOptions(parsePath string, parseOption Options, parseQuery url.Values) *Element {
	parseR.applyRouteMetadata(parseOption)
	if parseOption.Redirect == "" {
		return nil
	}

	parseCurrentTarget := buildPathWithQuery(parsePath, parseQuery)
	parseRedirectTarget := normalizeNavigationTarget(parseOption.Redirect)
	if parseRedirectTarget == parseCurrentTarget {
		runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "ignoring route redirect loop for "+parseRedirectTarget)
		return nil
	}

	runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "route redirect applied", "", map[string]string{
		"from": parseCurrentTarget,
		"to":   parseRedirectTarget,
	})
	parseR.recordRedirectDebug("route-option", parseCurrentTarget, parseRedirectTarget)
	parseR.replaceLocation(parseRedirectTarget)
	return parseR.currentElement(false)
}

// applyRouteMetadata is an internal router helper.
func (parseR *Router) applyRouteMetadata(parseOption Options) {
	applyRouteTitle(&parseR.metadataState, parseOption.Title)
	applyRouteMetaTag("description", parseOption.Description)
	applyRouteCanonical(parseOption.CanonicalURL)
}

// applyRouteTitle is an internal router helper.
func applyRouteTitle(parseState *routeMetadataState, parseTitle string) {
	parseDoc := js.Global().Get("document")
	if !parseDoc.Truthy() {
		return
	}
	initializeRouteMetadataState(parseState, parseDoc)

	parseTrimmed := strings.TrimSpace(parseTitle)
	if parseTrimmed == "" {
		if parseState != nil && parseState.titleManaged {
			if parseTitleElement := ensureManagedTitleElement(parseDoc, false); parseTitleElement.Truthy() {
				if strings.TrimSpace(parseState.baseTitle) == "" {
					removeElement(parseTitleElement)
				} else {
					parseTitleElement.Set("textContent", parseState.baseTitle)
					if parseTitleElement.Get("removeAttribute").Truthy() {
						parseTitleElement.Call("removeAttribute", managedMetadataAttr)
					}
				}
			}
			parseDoc.Set("title", parseState.baseTitle)
			parseState.titleManaged = false
		}
		return
	}

	parseTitleElement2 := ensureManagedTitleElement(parseDoc, true)
	if parseTitleElement2.Truthy() {
		parseTitleElement2.Set("textContent", parseTrimmed)
		parseTitleElement2.Call("setAttribute", managedMetadataAttr, managedMetadataValue)
	}
	parseDoc.Set("title", parseTrimmed)
	if parseState != nil {
		parseState.titleManaged = true
	}
}

// applyRouteMetaTag is an internal router helper.
func applyRouteMetaTag(parseName, parseContent string) {
	parseDoc := js.Global().Get("document")
	if !parseDoc.Truthy() {
		return
	}
	parseHead := getHeadElement(parseDoc)
	if !parseHead.Truthy() {
		return
	}
	parseElement := ensureManagedHeadElement(parseDoc, parseHead, `meta[name="`+parseName+`"]`, "meta", func(parseNode js.Value) {
		setElementAttribute(parseNode, "name", parseName)
	})
	parseTrimmed := strings.TrimSpace(parseContent)
	if parseTrimmed == "" {
		if parseElement.Truthy() {
			removeElement(parseElement)
		}
		return
	}
	if !parseElement.Truthy() {
		return
	}
	setElementAttribute(parseElement, managedMetadataAttr, managedMetadataValue)
	setElementAttribute(parseElement, "content", parseTrimmed)
}

// applyRouteCanonical is an internal router helper.
func applyRouteCanonical(parseHref string) {
	parseDoc := js.Global().Get("document")
	if !parseDoc.Truthy() {
		return
	}
	parseHead := getHeadElement(parseDoc)
	if !parseHead.Truthy() {
		return
	}
	parseElement := ensureManagedHeadElement(parseDoc, parseHead, `link[rel="canonical"]`, "link", func(parseNode js.Value) {
		setElementAttribute(parseNode, "rel", "canonical")
	})
	parseTrimmed := strings.TrimSpace(parseHref)
	if parseTrimmed == "" {
		if parseElement.Truthy() {
			removeElement(parseElement)
		}
		return
	}
	if !parseElement.Truthy() {
		return
	}
	setElementAttribute(parseElement, managedMetadataAttr, managedMetadataValue)
	setElementAttribute(parseElement, "href", parseTrimmed)
}

// initializeRouteMetadataState is an internal router helper.
func initializeRouteMetadataState(parseState *routeMetadataState, parseDoc js.Value) {
	if parseState == nil || parseState.baseTitleCaptured {
		return
	}
	parseState.baseTitleCaptured = true
	parseState.baseTitle = parseDoc.Get("title").String()
	if parseTitle := findManagedHeadElement(parseDoc, `title[`+managedMetadataAttr+`="`+managedMetadataValue+`"]`); parseTitle.Truthy() {
		parseState.baseTitle = ""
		parseState.titleManaged = true
	}
}

// ensureManagedTitleElement is an internal router helper.
func ensureManagedTitleElement(parseDoc js.Value, isCreate bool) js.Value {
	if parseElement := findManagedHeadElement(parseDoc, `title[`+managedMetadataAttr+`="`+managedMetadataValue+`"]`); parseElement.Truthy() {
		return parseElement
	}
	parseTitles := querySelectorAll(parseDoc, "title")
	if len(parseTitles) == 1 {
		setElementAttribute(parseTitles[0], managedMetadataAttr, managedMetadataValue)
		return parseTitles[0]
	}
	if !isCreate {
		return js.Null()
	}
	parseHead := getHeadElement(parseDoc)
	if !parseHead.Truthy() {
		return js.Null()
	}
	parseCreateElement := parseDoc.Get("createElement")
	if parseCreateElement.IsUndefined() || parseCreateElement.IsNull() || !parseCreateElement.Truthy() {
		return js.Null()
	}
	parseAppendChild := parseHead.Get("appendChild")
	if parseAppendChild.IsUndefined() || parseAppendChild.IsNull() || !parseAppendChild.Truthy() {
		return js.Null()
	}
	parseElement2 := parseDoc.Call("createElement", "title")
	setElementAttribute(parseElement2, managedMetadataAttr, managedMetadataValue)
	appendChildElement(parseHead, parseElement2)
	return parseElement2
}

// ensureManagedHeadElement is an internal router helper.
func ensureManagedHeadElement(parseDoc js.Value, parseHead js.Value, parseSelector string, parseTag string, parseInitialize func(js.Value)) js.Value {
	parseManagedSelector := parseSelector + `[` + managedMetadataAttr + `="` + managedMetadataValue + `"]`
	if parseElement := findManagedHeadElement(parseDoc, parseManagedSelector); parseElement.Truthy() {
		return parseElement
	}
	parseMatches := querySelectorAll(parseDoc, parseSelector)
	if len(parseMatches) == 1 {
		setElementAttribute(parseMatches[0], managedMetadataAttr, managedMetadataValue)
		if parseInitialize != nil {
			parseInitialize(parseMatches[0])
		}
		return parseMatches[0]
	}
	parseCreateElement := parseDoc.Get("createElement")
	if parseCreateElement.IsUndefined() || parseCreateElement.IsNull() || !parseCreateElement.Truthy() {
		return js.Null()
	}
	parseAppendChild := parseHead.Get("appendChild")
	if parseAppendChild.IsUndefined() || parseAppendChild.IsNull() || !parseAppendChild.Truthy() {
		return js.Null()
	}
	parseElement2 := parseDoc.Call("createElement", parseTag)
	setElementAttribute(parseElement2, managedMetadataAttr, managedMetadataValue)
	if parseInitialize != nil {
		parseInitialize(parseElement2)
	}
	appendChildElement(parseHead, parseElement2)
	return parseElement2
}

// findManagedHeadElement is an internal router helper.
func findManagedHeadElement(parseDoc js.Value, parseSelector string) js.Value {
	parseMatches := querySelectorAll(parseDoc, parseSelector)
	if len(parseMatches) == 0 {
		return js.Null()
	}
	for _, parseExtra := range parseMatches[1:] {
		removeElement(parseExtra)
	}
	return parseMatches[0]
}

// querySelectorAll is an internal router helper.
func querySelectorAll(parseDoc js.Value, parseSelector string) []js.Value {
	if parseDoc.IsUndefined() || parseDoc.IsNull() || !parseDoc.Truthy() {
		return nil
	}
	parseQueryAll := parseDoc.Get("querySelectorAll")
	if parseQueryAll.IsUndefined() || parseQueryAll.IsNull() || !parseQueryAll.Truthy() {
		return nil
	}
	parseList := parseDoc.Call("querySelectorAll", parseSelector)
	if parseList.IsUndefined() || parseList.IsNull() || !parseList.Truthy() {
		return nil
	}
	parseLength := parseList.Get("length").Int()
	if parseLength == 0 {
		return nil
	}
	parseMatches := make([]js.Value, 0, parseLength)
	for parseIndex := 0; parseIndex < parseLength; parseIndex++ {
		parseNode := parseList.Call("item", parseIndex)
		if !parseNode.Truthy() {
			parseNode = parseList.Index(parseIndex)
		}
		if parseNode.Truthy() {
			parseMatches = append(parseMatches, parseNode)
		}
	}
	return parseMatches
}

// removeElement is an internal router helper.
func removeElement(parseNode js.Value) {
	if !parseNode.Truthy() {
		return
	}
	if parseNode.Get("remove").Truthy() {
		parseNode.Call("remove")
		return
	}
	parseParent := parseNode.Get("parentNode")
	if parseParent.Truthy() {
		parseParent.Call("removeChild", parseNode)
	}
}

// getHeadElement is an internal router helper.
func getHeadElement(parseDoc js.Value) js.Value {
	if parseDoc.IsUndefined() || parseDoc.IsNull() || !parseDoc.Truthy() {
		return js.Null()
	}
	parseHead := parseDoc.Get("head")
	if parseHead.Truthy() {
		return parseHead
	}
	parseQuery := parseDoc.Get("querySelector")
	if parseQuery.IsUndefined() || parseQuery.IsNull() || !parseQuery.Truthy() {
		return js.Null()
	}
	return parseDoc.Call("querySelector", "head")
}

// setElementAttribute is an internal router helper.
func setElementAttribute(parseNode js.Value, parseName, parseValue string) bool {
	if parseNode.IsUndefined() || parseNode.IsNull() || !parseNode.Truthy() {
		return false
	}
	parseMethod := parseNode.Get("setAttribute")
	if parseMethod.IsUndefined() || parseMethod.IsNull() || !parseMethod.Truthy() {
		return false
	}
	parseNode.Call("setAttribute", parseName, parseValue)
	return true
}

// appendChildElement is an internal router helper.
func appendChildElement(parseParent js.Value, parseChild js.Value) bool {
	if parseParent.IsUndefined() || parseParent.IsNull() || !parseParent.Truthy() {
		return false
	}
	parseMethod := parseParent.Get("appendChild")
	if parseMethod.IsUndefined() || parseMethod.IsNull() || !parseMethod.Truthy() {
		return false
	}
	parseParent.Call("appendChild", parseChild)
	return true
}

// evaluateNavigation is an internal router helper.
func (parseR *Router) evaluateNavigation(parseTarget string) (string, bool) {
	parseCtx, parseAttemptID := parseR.beginGuardAttempt()
	defer parseR.finishGuardAttempt(parseAttemptID)
	return parseR.evaluateNavigationWithAttempt(parseCtx, parseAttemptID, parseTarget)
}

// evaluateNavigationWithAttempt is an internal router helper.
func (parseR *Router) evaluateNavigationWithAttempt(parseCtx context.Context, parseAttemptID uint64, parseTarget string) (string, bool) {
	parseCurrentPath := parseR.GetCurrentRouterPath()
	parseCurrentQuery := getCurrentQueryValues()
	parseCurrentResolved := parseR.resolveRouteStack(parseCurrentPath)

	parseNextTarget := parseTarget
	for parseSteps := 0; parseSteps < 4; parseSteps++ {
		if parseCtx.Err() != nil || !parseR.guardAttemptActive(parseAttemptID) {
			return "", false
		}
		parseNextPath, parseNextQuery := parseNavigationTarget(parseNextTarget)
		parseNextResolved := parseR.resolveRouteStack(parseNextPath)
		parseNextLeaf := resolvedRoute{}
		if parseNextResolved.found {
			parseNextLeaf = parseNextResolved.routes[len(parseNextResolved.routes)-1]
		}
		parseNextCtx := parseR.routeContext(parseNextPath, parseNextLeaf.params, parseNextQuery)
		isParseRedirected := false

		if parseSteps == 0 && parseCurrentResolved.found {
			for parseIndex := len(parseCurrentResolved.routes) - 1; parseIndex >= 0; parseIndex-- {
				parseCurrentRoute := parseCurrentResolved.routes[parseIndex]
				if parseCurrentRoute.option.BeforeLeave == nil && parseCurrentRoute.option.BeforeLeaveAsync == nil {
					continue
				}
				parseDecision := guardDecisionAllowed()
				parseCurrentCtx := parseR.routeContext(parseCurrentRoute.path, parseCurrentRoute.params, parseCurrentQuery)
				if parseCurrentRoute.option.BeforeLeave != nil {
					parseDecision = guardDecisionFromResult(parseCurrentRoute.option.BeforeLeave(parseCurrentCtx, parseNextCtx))
				}
				if !parseDecision.Blocked && parseDecision.Redirect == "" && parseCurrentRoute.option.BeforeLeaveAsync != nil {
					parseDecision = parseCurrentRoute.option.BeforeLeaveAsync(parseCtx, parseCurrentCtx, parseNextCtx)
				}
				if parseCtx.Err() != nil || !parseR.guardAttemptActive(parseAttemptID) {
					return "", false
				}
				if parseRedirect := strings.TrimSpace(parseDecision.Redirect); parseRedirect != "" {
					parseNextTarget = normalizeNavigationTarget(parseRedirect)
					parseR.recordRedirectDebug("before-leave", parseCurrentRoute.path, parseNextTarget)
					runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "before-leave redirected navigation", "", map[string]string{
						"from": parseCurrentRoute.path,
						"to":   parseNextTarget,
					})
					isParseRedirected = true
					break
				}
				if parseDecision.Blocked || parseDecision.Denied {
					runtime.ReportLogWithFields("router", runtime.LogWarn, runtime.DiagnosticRecovered, "before-leave blocked navigation", "", map[string]string{
						"from": parseCurrentRoute.path,
						"to":   parseNextPath,
					})
					return "", false
				}
			}
			if isParseRedirected {
				continue
			}
		}

		if parseNextResolved.found {
			for _, parseNextRoute := range parseNextResolved.routes {
				if parseNextRoute.option.BeforeEnter == nil && parseNextRoute.option.BeforeEnterAsync == nil {
					continue
				}
				parseDecision2 := guardDecisionAllowed()
				parseNextRouteCtx := parseR.routeContext(parseNextRoute.path, parseNextRoute.params, parseNextQuery)
				if parseNextRoute.option.BeforeEnter != nil {
					parseDecision2 = guardDecisionFromResult(parseNextRoute.option.BeforeEnter(parseNextRouteCtx))
				}
				if !parseDecision2.Blocked && parseDecision2.Redirect == "" && parseNextRoute.option.BeforeEnterAsync != nil {
					parseDecision2 = parseNextRoute.option.BeforeEnterAsync(parseCtx, parseNextRouteCtx)
				}
				if parseCtx.Err() != nil || !parseR.guardAttemptActive(parseAttemptID) {
					return "", false
				}
				if parseRedirect2 := strings.TrimSpace(parseDecision2.Redirect); parseRedirect2 != "" {
					parseNextTarget = normalizeNavigationTarget(parseRedirect2)
					parseR.recordRedirectDebug("before-enter", parseNextPath, parseNextTarget)
					runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "before-enter redirected navigation", "", map[string]string{
						"from": parseNextPath,
						"to":   parseNextTarget,
					})
					isParseRedirected = true
					break
				}
				if parseDecision2.Blocked || parseDecision2.Denied {
					runtime.ReportLogWithFields("router", runtime.LogWarn, runtime.DiagnosticRecovered, "before-enter blocked navigation", "", map[string]string{
						"path": parseNextPath,
					})
					return "", false
				}
			}
			if isParseRedirected {
				continue
			}
		}

		return parseNextTarget, true
	}

	runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "navigation guard redirect loop detected for "+parseTarget)
	return "", false
}

// parseNavigationTarget is an internal router helper.
func parseNavigationTarget(parseTarget string) (string, url.Values) {
	parseNormalized := normalizeNavigationTarget(parseTarget)
	parsePath := parseNormalized
	parseQuery := url.Values{}
	if parseIdx := strings.Index(parseNormalized, "?"); parseIdx >= 0 {
		parsePath = parseNormalized[:parseIdx]
		parseParsed, parseErr := url.ParseQuery(parseNormalized[parseIdx+1:])
		if parseErr == nil {
			parseQuery = parseParsed
		}
	}
	return parsePath, parseQuery
}

// PreserveReturnTo normalizes an internal path plus query values into a bounded
// return-target payload suitable for auth or re-auth redirects.
func PreserveReturnTo(parsePath string, parseQuery url.Values) string {
	parseTrimmed := strings.TrimSpace(parsePath)
	if parseTrimmed == "" {
		return rootRoutePath
	}
	if parseParsed, parseErr := url.Parse(parseTrimmed); parseErr == nil {
		if parseParsed.IsAbs() || parseParsed.Host != "" {
			return rootRoutePath
		}
	}
	if strings.HasPrefix(parseTrimmed, "//") {
		return rootRoutePath
	}
	parseTarget := buildPathWithQuery(parseTrimmed, copyQueryValues(parseQuery))
	if strings.TrimSpace(parseTarget) == "" || len(parseTarget) > maxReturnToLength {
		return rootRoutePath
	}
	return parseTarget
}

// ReadReturnTo reads an internal return-target query value and falls back when
// the value is empty, oversized, or external.
func ReadReturnTo(parseQuery url.Values, parseFallback string) string {
	parseFallbackTarget := normalizeNavigationTarget(parseFallback)
	parseRaw := strings.TrimSpace(copyQueryValues(parseQuery).Get(ReturnToParam))
	if parseRaw == "" || len(parseRaw) > maxReturnToLength {
		return parseFallbackTarget
	}
	parseParsed, parseErr := url.Parse(parseRaw)
	if parseErr != nil || parseParsed.IsAbs() || parseParsed.Host != "" || strings.HasPrefix(parseRaw, "//") {
		return parseFallbackTarget
	}
	return normalizeNavigationTarget(parseRaw)
}

// AllowNavigation permits the pending navigation.
func AllowNavigation() GuardResult {
	return GuardResult{}
}

// BlockNavigation blocks the pending navigation with a reason.
func BlockNavigation(parseReason string) GuardResult {
	return GuardResult{Blocked: true, Reason: parseReason}
}

// RedirectNavigation redirects the pending navigation to path.
func RedirectNavigation(parsePath string) GuardResult {
	return GuardResult{Redirect: parsePath}
}

// guardDecisionFromResult is an internal router helper.
func guardDecisionFromResult(parseResult GuardResult) GuardDecision {
	return GuardDecision{
		Redirect: parseResult.Redirect,
		Blocked:  parseResult.Blocked,
		Reason:   parseResult.Reason,
		Denied:   parseResult.Blocked,
	}
}

// guardDecisionAllowed is an internal router helper.
func guardDecisionAllowed() GuardDecision {
	return GuardDecision{}
}

// beginGuardAttempt is an internal router helper.
func (parseR *Router) beginGuardAttempt() (context.Context, uint64) {
	parseR.guardState.mu.Lock()
	if parseR.guardState.cancel != nil {
		parseR.guardState.cancel()
	}
	parseR.guardState.seq++
	parseId := parseR.guardState.seq
	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseR.guardState.active = parseId
	parseR.guardState.cancel = parseCancel
	parseR.guardState.mu.Unlock()
	return parseCtx, parseId
}

// finishGuardAttempt is an internal router helper.
func (parseR *Router) finishGuardAttempt(parseId uint64) {
	parseR.guardState.mu.Lock()
	defer parseR.guardState.mu.Unlock()
	if parseR.guardState.active != parseId {
		return
	}
	if parseR.guardState.cancel != nil {
		parseR.guardState.cancel()
	}
	parseR.guardState.active = 0
	parseR.guardState.cancel = nil
}

// guardAttemptActive is an internal router helper.
func (parseR *Router) guardAttemptActive(parseId uint64) bool {
	parseR.guardState.mu.Lock()
	defer parseR.guardState.mu.Unlock()
	return parseR.guardState.active == parseId && parseId != 0
}

// replaceLocation is an internal router helper.
func (parseR *Router) replaceLocation(parseTarget string) {
	if parseR.routerType == "history" {
		parseHistory := getHistoryValue()
		if parseHistory.Truthy() && parseHistory.Get("replaceState").Truthy() {
			parseHistory.Call("replaceState", nil, "", parseTarget)
			return
		}
		if parseLoc := getLocationValue(); parseLoc.Truthy() {
			if parseIdx := strings.Index(parseTarget, "?"); parseIdx >= 0 {
				parseLoc.Set("pathname", parseTarget[:parseIdx])
				parseLoc.Set("search", parseTarget[parseIdx:])
			} else {
				parseLoc.Set("pathname", parseTarget)
				parseLoc.Set("search", "")
			}
		}
		return
	}

	parseLoc2 := getLocationValue()
	if !parseLoc2.Truthy() {
		return
	}
	parseLoc2.Set("hash", "#"+strings.TrimPrefix(parseTarget, "#"))
}

// ensureLoaderResult is an internal router helper.
func (parseR *Router) ensureLoaderResult(parseKey string, parseLoader LoaderFunc, parseRouteCtx RouteContext) struct {
	pending bool
	data    Attrs
	err     error
} {
	parseR.loaderState.mu.Lock()
	parseEntry := parseR.loaderState.entries[parseKey]
	if parseEntry != nil {
		parseState := struct {
			pending bool
			data    Attrs
			err     error
		}{
			pending: parseEntry.pending,
			data:    copyAttrs(parseEntry.data),
			err:     parseEntry.err,
		}
		parseR.loaderState.mu.Unlock()
		return parseState
	}

	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseEntry = &loaderEntry{
		pending: true,
		cancel:  parseCancel,
	}
	parseEntry.version++
	parseVersion := parseEntry.version
	parseR.loaderState.entries[parseKey] = parseEntry
	parseR.loaderState.mu.Unlock()

	runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "route loader started", "", map[string]string{
		"key":  parseKey,
		"path": parseRouteCtx.Path,
	})
	runtime.ReportProfilingEvent("router", "loader", "start", parseRouteCtx.Path, 0, map[string]string{
		"key": parseKey,
	})

	go func() {
		parseLoaderStarted := time.Now()
		defer func() {
			if parseRecovered := recover(); parseRecovered != nil {
				parseR.loaderState.mu.Lock()
				parseCurrent := parseR.loaderState.entries[parseKey]
				if parseCurrent != nil && parseCurrent == parseEntry && parseVersion == parseCurrent.version {
					parseCurrent.pending = false
					parseCurrent.cancel = nil
				}
				parseR.loaderState.mu.Unlock()
				runtime.ReportProfilingEvent("router", "loader", "panic", parseRouteCtx.Path, time.Since(parseLoaderStarted).Nanoseconds(), map[string]string{
					"key": parseKey,
				})
				if _, parseSuppressed := runtime.FinalizeUnhandledPanicContext("router", runtime.PanicPhaseLoader, "route loader", parseRouteCtx.Path, nil, parseRecovered); parseSuppressed {
					return
				}
			}
		}()

		parseData, parseErr := parseLoader(parseCtx, parseRouteCtx)
		parseDurationNs := time.Since(parseLoaderStarted).Nanoseconds()

		parseR.loaderState.mu.Lock()
		parseCurrent2 := parseR.loaderState.entries[parseKey]
		if parseCtx.Err() != nil || parseCurrent2 == nil || parseCurrent2 != parseEntry || parseVersion != parseCurrent2.version {
			parseR.loaderState.mu.Unlock()
			runtime.ReportProfilingEvent("router", "loader", "cancelled", parseRouteCtx.Path, parseDurationNs, map[string]string{
				"key": parseKey,
			})
			return
		}
		parseCurrent2.pending = false
		parseCurrent2.data = copyAttrs(parseData)
		parseCurrent2.err = parseErr
		parseCurrent2.cancel = nil
		if parseErr != nil {
			runtime.ReportLogWithFields("router", runtime.LogError, runtime.DiagnosticCorrectness, "route loader failed", "", map[string]string{
				"key":   parseKey,
				"path":  parseRouteCtx.Path,
				"error": parseErr.Error(),
			})
			runtime.ReportProfilingEvent("router", "loader", "error", parseRouteCtx.Path, parseDurationNs, map[string]string{
				"key":   parseKey,
				"error": parseErr.Error(),
			})
		} else {
			runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "route loader resolved", "", map[string]string{
				"key":  parseKey,
				"path": parseRouteCtx.Path,
			})
			runtime.ReportProfilingEvent("router", "loader", "finish", parseRouteCtx.Path, parseDurationNs, map[string]string{
				"key": parseKey,
			})
		}
		parseR.loaderState.mu.Unlock()

		parseDoc := js.Global().Get("document")
		parseElem := js.Global().Get("Element")
		if parseDoc.IsUndefined() || parseDoc.IsNull() || parseElem.IsUndefined() || parseElem.IsNull() {
			return
		}
		parseR.renderCurrentRoute(false)
	}()

	return struct {
		pending bool
		data    Attrs
		err     error
	}{pending: true}
}

// cancelLoaderIfActive is an internal router helper.
func (parseR *Router) cancelLoaderIfActive() {
	parseR.loaderState.mu.Lock()
	defer parseR.loaderState.mu.Unlock()
	for parseKey, parseEntry := range parseR.loaderState.entries {
		if parseEntry != nil && parseEntry.cancel != nil {
			parseEntry.cancel()
		}
		delete(parseR.loaderState.entries, parseKey)
	}
	parseR.loaderState.active = make(map[string]struct{})
}

// buildLoaderKey is an internal router helper.
func buildLoaderKey(parseRouteID, parseResolvedPath, parseQueryKey string) string {
	parseBase := parseRouteID + "@" + normalizePath(parseResolvedPath)
	if parseQueryKey == "" {
		return parseBase
	}
	return parseBase + "?" + parseQueryKey
}

// inspectCurrentRoute is an internal router helper.
func (parseR *Router) inspectCurrentRoute() RouteInspection {
	parseQuery := copyQueryValues(getCurrentQueryValues())
	parsePath := GetCurrentPath()
	parseResolved := parseR.resolveRouteStack(parsePath)
	return RouteInspection{
		Path:         parsePath,
		Query:        parseQuery,
		Params:       copyParams(currentParams),
		Loading:      parseR.IsLoading(),
		Stack:        inspectRouteStack(parseResolved.routes),
		Loaders:      parseR.inspectRouteLoaders(parseResolved.routes, parseQuery.Encode()),
		LastRedirect: parseR.lastRedirectInspection(),
		Metadata:     inspectRouteMetadata(parseResolved.routes),
	}
}

// inspectRouteStack is an internal router helper.
func inspectRouteStack(parseRoutes []resolvedRoute) []RouteStackInspection {
	if len(parseRoutes) == 0 {
		return nil
	}
	parseStack := make([]RouteStackInspection, 0, len(parseRoutes))
	for _, parseRoute := range parseRoutes {
		parseStack = append(parseStack, RouteStackInspection{
			ID:             parseRoute.id,
			Path:           parseRoute.path,
			Params:         copyParams(parseRoute.params),
			HasLoader:      parseRoute.option.Loader != nil,
			HasBeforeEnter: parseRoute.option.BeforeEnter != nil || parseRoute.option.BeforeEnterAsync != nil,
			HasBeforeLeave: parseRoute.option.BeforeLeave != nil || parseRoute.option.BeforeLeaveAsync != nil,
			Metadata: Metadata{
				Title:        parseRoute.option.Title,
				Description:  parseRoute.option.Description,
				CanonicalURL: parseRoute.option.CanonicalURL,
			},
		})
	}
	return parseStack
}

// inspectRouteMetadata is an internal router helper.
func inspectRouteMetadata(parseRoutes []resolvedRoute) Metadata {
	if len(parseRoutes) == 0 {
		return Metadata{}
	}
	parseLeaf := parseRoutes[len(parseRoutes)-1]
	return Metadata{
		Title:        parseLeaf.option.Title,
		Description:  parseLeaf.option.Description,
		CanonicalURL: parseLeaf.option.CanonicalURL,
	}
}

// inspectRouteLoaders is an internal router helper.
func (parseR *Router) inspectRouteLoaders(parseRoutes []resolvedRoute, parseQueryKey string) []RouteLoaderInspection {
	if len(parseRoutes) == 0 {
		return nil
	}
	parseR.loaderState.mu.Lock()
	defer parseR.loaderState.mu.Unlock()
	parseLoaders := make([]RouteLoaderInspection, 0, len(parseRoutes))
	for _, parseRoute := range parseRoutes {
		if parseRoute.option.Loader == nil {
			continue
		}
		parseKey := buildLoaderKey(parseRoute.id, parseRoute.path, parseQueryKey)
		parseEntry := parseR.loaderState.entries[parseKey]
		parseInspection := RouteLoaderInspection{
			Key:  parseKey,
			Path: parseRoute.path,
		}
		if parseEntry != nil {
			parseInspection.Pending = parseEntry.pending
			parseInspection.HasData = len(parseEntry.data) > 0
			if parseEntry.err != nil {
				parseInspection.Error = parseEntry.err.Error()
			}
		}
		parseLoaders = append(parseLoaders, parseInspection)
	}
	return parseLoaders
}

// recordRedirectDebug is an internal router helper.
func (parseR *Router) recordRedirectDebug(parseCause, parseFrom, parseTo string) {
	parseR.debugState.mu.Lock()
	defer parseR.debugState.mu.Unlock()
	parseR.debugState.lastRedirect = RouteRedirectInspection{
		Cause: strings.TrimSpace(parseCause),
		From:  strings.TrimSpace(parseFrom),
		To:    strings.TrimSpace(parseTo),
	}
}

// lastRedirectInspection is an internal router helper.
func (parseR *Router) lastRedirectInspection() RouteRedirectInspection {
	parseR.debugState.mu.Lock()
	defer parseR.debugState.mu.Unlock()
	return parseR.debugState.lastRedirect
}

// renderRouteFallback is an internal router helper.
func renderRouteFallback(parseComponent interface{}, parseProps Attrs) *Element {
	if parseComponent != nil {
		return makeRouteFactory(parseComponent)(parseProps)
	}
	return runtime.Div(nil, runtime.Text(routeLoadingText))
}

// renderRouteError is an internal router helper.
func renderRouteError(parseComponent interface{}, parseErr error, parseProps Attrs) *Element {
	if parseComponent != nil {
		return makeRouteFactory(parseComponent)(parseProps)
	}
	parseMessage := "Route load failed"
	if parseErr != nil {
		parseMessage = parseErr.Error()
	}
	return runtime.Div(nil, runtime.Text(parseMessage))
}

// renderRouteGuardState is an internal router helper.
func renderRouteGuardState(parseComponent interface{}, parseProps Attrs) *Element {
	if parseComponent != nil {
		return makeRouteFactory(parseComponent)(parseProps)
	}
	parseMessage, _ := parseProps["reason"].(string)
	if strings.TrimSpace(parseMessage) == "" {
		parseMessage = navigationBlocked
	}
	return runtime.Div(nil, runtime.Text(parseMessage))
}

// isPatternRoute is an internal router helper.
func isPatternRoute(parsePath string) bool {
	return strings.Contains(parsePath, ":") || (strings.HasSuffix(parsePath, "*") && parsePath != "*")
}

// matchPattern is an internal router helper.
func (parseR *Router) matchPattern(parsePath string) (routeFactory, map[string]string, Options, string, bool) {
	for _, parsePattern := range parseR.patterns {
		if parseParams, parseOk := matchRoutePattern(parsePattern.pattern, parsePath); parseOk {
			return parsePattern.factory, parseParams, parsePattern.options, parsePattern.pattern, true
		}
	}
	return nil, nil, Options{}, "", false
}

// prepareLoaderState is an internal router helper.
func (parseR *Router) prepareLoaderState(parseActiveKeys []string) {
	parseR.loaderState.mu.Lock()
	defer parseR.loaderState.mu.Unlock()
	parseNextActive := make(map[string]struct{}, len(parseActiveKeys))
	for _, parseKey := range parseActiveKeys {
		parseNextActive[parseKey] = struct{}{}
	}
	for parseKey2, parseEntry := range parseR.loaderState.entries {
		if _, parseKeep := parseNextActive[parseKey2]; parseKeep {
			continue
		}
		if parseEntry != nil && parseEntry.cancel != nil {
			parseEntry.cancel()
		}
		delete(parseR.loaderState.entries, parseKey2)
	}
	parseR.loaderState.active = parseNextActive
}

// routeIDExact is an internal router helper.
func routeIDExact(parsePath string) string {
	return "exact:" + parsePath
}

// routeIDPattern is an internal router helper.
func routeIDPattern(parsePattern string) string {
	return "pattern:" + parsePattern
}

// routeIDNotFound is an internal router helper.
func routeIDNotFound() string {
	return "notfound:*"
}

// expandPathPrefixes is an internal router helper.
func expandPathPrefixes(parsePath string) []string {
	parseParts := splitPath(parsePath)
	if len(parseParts) == 0 {
		return []string{"/"}
	}
	parsePrefixes := make([]string, 0, len(parseParts)+1)
	parsePrefixes = append(parsePrefixes, "/")
	for parseIndex := range parseParts {
		parsePrefixes = append(parsePrefixes, "/"+strings.Join(parseParts[:parseIndex+1], "/"))
	}
	return parsePrefixes
}

// matchRoutePattern is an internal router helper.
func matchRoutePattern(parsePattern, parsePath string) (map[string]string, bool) {
	if strings.HasSuffix(parsePattern, "*") {
		parsePrefix := strings.TrimSuffix(parsePattern, "*")
		parsePrefix = strings.TrimSuffix(parsePrefix, "/")
		if parsePrefix == "" {
			return map[string]string{}, true
		}
		if parsePath == parsePrefix || strings.HasPrefix(parsePath, parsePrefix+"/") {
			return map[string]string{}, true
		}
		return nil, false
	}

	parsePatternParts := splitPath(parsePattern)
	parsePathParts := splitPath(parsePath)
	if len(parsePatternParts) != len(parsePathParts) {
		return nil, false
	}

	parseParams := make(map[string]string)
	for parseIndex, parsePart := range parsePatternParts {
		parseCandidate := parsePathParts[parseIndex]
		if strings.HasPrefix(parsePart, ":") {
			parseName := strings.TrimPrefix(parsePart, ":")
			if parseName == "" || strings.ContainsAny(parseName, "?*") {
				return nil, false
			}
			parseDecoded, parseErr := url.PathUnescape(parseCandidate)
			if parseErr != nil || parseDecoded == "" {
				return nil, false
			}
			parseParams[parseName] = parseDecoded
			continue
		}
		if parsePart != parseCandidate {
			return nil, false
		}
	}

	return parseParams, true
}

// splitPath is an internal router helper.
func splitPath(parsePath string) []string {
	parseTrimmed := strings.Trim(strings.TrimSpace(parsePath), "/")
	if parseTrimmed == "" {
		return []string{}
	}
	return strings.Split(parseTrimmed, "/")
}

// copyParams is an internal router helper.
func copyParams(parseParams map[string]string) map[string]string {
	if len(parseParams) == 0 {
		return map[string]string{}
	}
	parseClone := make(map[string]string, len(parseParams))
	for parseKey, parseValue := range parseParams {
		parseClone[parseKey] = parseValue
	}
	return parseClone
}

// copyParamsToAttrs is an internal router helper.
func copyParamsToAttrs(parseParams map[string]string) Attrs {
	if len(parseParams) == 0 {
		return nil
	}
	parseAttrs := make(Attrs, len(parseParams))
	for parseKey, parseValue := range parseParams {
		parseAttrs[parseKey] = parseValue
	}
	return parseAttrs
}

// copyAttrs is an internal router helper.
func copyAttrs(parseAttrs Attrs) Attrs {
	if len(parseAttrs) == 0 {
		return nil
	}
	parseClone := make(Attrs, len(parseAttrs))
	for parseKey, parseValue := range parseAttrs {
		parseClone[parseKey] = parseValue
	}
	return parseClone
}

// copyQueryValues is an internal router helper.
func copyQueryValues(parseValues url.Values) url.Values {
	parseClone := make(url.Values, len(parseValues))
	for parseKey, parseValue := range parseValues {
		parseClone[parseKey] = append([]string(nil), parseValue...)
	}
	return parseClone
}

// mergeAttrs is an internal router helper.
func mergeAttrs(parseBase Attrs, parseExtra Attrs) Attrs {
	if len(parseBase) == 0 && len(parseExtra) == 0 {
		return nil
	}
	parseMerged := make(Attrs, len(parseBase)+len(parseExtra))
	for parseKey, parseValue := range parseBase {
		parseMerged[parseKey] = parseValue
	}
	for parseKey2, parseValue2 := range parseExtra {
		parseMerged[parseKey2] = parseValue2
	}
	return parseMerged
}
