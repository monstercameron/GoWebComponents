//go:build js && wasm

package router

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/deprecation"
	"github.com/monstercameron/GoWebComponents/v4/internal/platform/jsdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
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
	Chunk            RouteChunk
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
	routes                 map[string]routeFactory
	routeOptions           map[string]Options
	patterns               []routePattern
	defaultRoute           string
	notFound               routeFactory
	notFoundOption         Options
	targetSelector         string
	targetElement          js.Value
	listening              bool
	disposed               bool   // set when the router is unmounted; stale loader goroutines check this flag
	routerType             string // "hash" or "history"
	loaderState            loaderState
	chunkState             routeChunkState
	metadataState          routeMetadataState
	guardState             navigationGuardState
	debugState             routeDebugState
	renderCurrentComponent func() *Element
	isRenderApplyGuards    bool
	renderVersion          int
	// viewTransitions, when true (the default), wraps each navigation's DOM swap in the
	// browser View Transitions API so route changes animate; it auto-skips under
	// prefers-reduced-motion. focusManagement, when true (the default), moves keyboard focus
	// to the new route's content after navigation so screen-reader and keyboard users are not
	// stranded on the old page (D4). Both are opt-out via SetViewTransitions/SetFocusManagement.
	viewTransitions bool
	focusManagement bool
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

type routeChunkState struct {
	mu      sync.Mutex
	entries map[string]*routeChunkEntry
	active  map[string]struct{}
}

type loaderEntry struct {
	pending bool
	data    Attrs
	err     error
	cancel  context.CancelFunc
	version int
}

type routeChunkEntry struct {
	pending bool
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

// splitNavigationTarget separates a navigation target into path/query and fragment components.
func splitNavigationTarget(parseTarget string) (string, string) {
	parseTrimmed := strings.TrimSpace(parseTarget)
	if parseTrimmed == "" {
		return "", ""
	}
	if parseIdx := strings.Index(parseTrimmed, "#"); parseIdx >= 0 {
		return parseTrimmed[:parseIdx], parseTrimmed[parseIdx:]
	}
	return parseTrimmed, ""
}

// normalizeHistoryFragment converts a history-router fragment target into a canonical hash string.
func normalizeHistoryFragment(parseFragment string) string {
	parseTrimmed := strings.TrimSpace(parseFragment)
	if parseTrimmed == "" || parseTrimmed == "#" {
		return ""
	}
	if strings.HasPrefix(parseTrimmed, "#") {
		return parseTrimmed
	}
	return "#" + strings.TrimPrefix(parseTrimmed, "#")
}

// buildCurrentHistoryTarget returns the current history-router path plus search string.
func buildCurrentHistoryTarget() string {
	parseLoc := getLocationValue()
	if !parseLoc.Truthy() {
		return rootRoutePath
	}
	parsePath := normalizePath(parseLoc.Get("pathname").String())
	parseSearch := strings.TrimSpace(parseLoc.Get("search").String())
	if parseSearch == "?" {
		parseSearch = ""
	}
	return parsePath + parseSearch
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

	parseRouter := &Router{
		routes:          make(map[string]routeFactory),
		routeOptions:    make(map[string]Options),
		patterns:        []routePattern{},
		defaultRoute:    parseCfg.DefaultRoute,
		routerType:      routerTypeHash,
		viewTransitions: true,
		focusManagement: true,
		loaderState: loaderState{
			entries: make(map[string]*loaderEntry),
			active:  make(map[string]struct{}),
		},
		chunkState: routeChunkState{
			entries: make(map[string]*routeChunkEntry),
			active:  make(map[string]struct{}),
		},
	}
	return parseRouter
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
		routes:          make(map[string]routeFactory),
		routeOptions:    make(map[string]Options),
		patterns:        []routePattern{},
		defaultRoute:    parseCfg.DefaultRoute,
		routerType:      routerTypeHistory,
		viewTransitions: true,
		focusManagement: true,
		loaderState: loaderState{
			entries: make(map[string]*loaderEntry),
			active:  make(map[string]struct{}),
		},
		chunkState: routeChunkState{
			entries: make(map[string]*routeChunkEntry),
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
		defer runtime.RecoverContainedPanic("router", "setupHistoryListener callback")
		parseR.renderCurrentRoute(true)
		return nil
	})

	parseWindow.Call("addEventListener", browserEventPop, parsePopstateHandler)
	parseHashchangeHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer runtime.RecoverContainedPanic("router", "setupHistoryListener callback")
		// History routers still need hashchange rerenders for in-page fragment navigation.
		// Without this, links like "#faq" update the URL but shared route components never
		// see a new location snapshot, which breaks anchor state and browser back/forward.
		parseR.renderCurrentRoute(true)
		return nil
	})
	parseWindow.Call("addEventListener", browserEventHash, parseHashchangeHandler)

	// Clean up on unload
	registerCleanup(parsePopstateHandler)
	registerCleanup(parseHashchangeHandler)
}

// GoRegisterRoute registers a route on the router instance.
//
// Deprecated: Use Register instead. GoRegisterRoute exists for compatibility with
// earlier API consumers and delegates directly to Register.
func (parseR *Router) GoRegisterRoute(parsePath string, parseComponent interface{}, parseOptions ...Options) {
	deprecation.Warn("router.Router.GoRegisterRoute", "router.Router.Register")
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
//
// Deprecated: Use Current instead. GoGetRoute exists for compatibility with
// earlier API consumers and delegates directly to Current.
func (parseR *Router) GoGetRoute() *Element {
	deprecation.Warn("router.Router.GoGetRoute", "router.Router.Current")
	return parseR.Current()
}

// Current returns the current route element.
func (parseR *Router) Current() *Element {
	return parseR.currentElement(true)
}

// setGlobalRouter updates the global router, marking the previous one as disposed
// so any in-flight loader goroutines on it do not trigger stale re-renders (#46).
func setGlobalRouter(parseNew *Router) {
	if globalRouter != nil && globalRouter != parseNew {
		globalRouter.disposed = true
	}
	globalRouter = parseNew
}

// maxRedirectDepth is the maximum number of chained redirects before aborting
// to prevent infinite redirect loops when a redirect target itself redirects.
const maxRedirectDepth = 5

// currentElement is an internal router helper.
func (parseR *Router) currentElement(isApplyGuards bool) *Element {
	return parseR.currentElementWithDepth(isApplyGuards, 0)
}

// currentElementWithDepth is an internal router helper that tracks redirect depth
// to prevent infinite loops when a redirect target itself has a redirect or guard redirect.
func (parseR *Router) currentElementWithDepth(isApplyGuards bool, parseRedirectDepth int) *Element {
	setGlobalRouter(parseR)
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
			parseRendered := parseR.renderResolvedRouteStackWithDepth(parseResolved.routes, parseQuery, parseQueryKey, true, parseCtx, parseAttemptID, parseRedirectDepth)
			if parseCtx.Err() != nil || !parseR.guardAttemptActive(parseAttemptID) {
				return nil
			}
			return parseRendered
		}
		return parseR.renderResolvedRouteStackWithDepth(parseResolved.routes, parseQuery, parseQueryKey, false, nil, 0, parseRedirectDepth)
	}

	parseR.cancelLoaderIfActive()
	currentRouteData = nil
	currentRouteOutlet = nil

	return runtime.Div(nil, runtime.Text(routeNotFoundText))
}

// Mount renders the router into a DOM node selected by CSS selector and wires hashchange listeners.
func (parseR *Router) Mount(parseSelector string) {
	setGlobalRouter(parseR)
	parseR.targetSelector = parseSelector
	parseR.targetElement = js.Null()
	parseR.renderCurrentRoute(true)
	parseR.ensureListener()
}

// HydrateMount binds the router to an already-hydrated DOM target and only
// wires future route updates/listeners without forcing an immediate rerender.
func (parseR *Router) HydrateMount(parseSelector string) {
	setGlobalRouter(parseR)
	parseR.targetSelector = parseSelector
	parseR.targetElement = js.Null()
	parseR.ensureListener()
}

// MountElement renders the router into an existing DOM element reference.
func (parseR *Router) MountElement(parseElem js.Value) {
	setGlobalRouter(parseR)
	parseR.targetElement = parseElem
	parseR.targetSelector = ""
	parseR.renderCurrentRoute(true)
	parseR.ensureListener()
}

// HydrateMountElement binds the router to an already-hydrated DOM element and
// only wires future route updates/listeners without forcing an immediate rerender.
func (parseR *Router) HydrateMountElement(parseElem js.Value) {
	setGlobalRouter(parseR)
	parseR.targetElement = parseElem
	parseR.targetSelector = ""
	parseR.ensureListener()
}

// Revalidate clears the cached result for the current route loader and runs it again.
func (parseR *Router) Revalidate() {
	parseR.cancelLoaderIfActive()
	parseR.renderCurrentRoute(false)
}

// RetryLoader clears one active loader entry by key and renders the current route again.
func (parseR *Router) RetryLoader(parseKey string) error {
	parseTrimmed := strings.TrimSpace(parseKey)
	if parseTrimmed == "" {
		return fmt.Errorf("router: loader key is required")
	}
	parseR.loaderState.mu.Lock()
	parseEntry := parseR.loaderState.entries[parseTrimmed]
	_, hasActive := parseR.loaderState.active[parseTrimmed]
	if parseEntry == nil && !hasActive {
		parseR.loaderState.mu.Unlock()
		runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "route loader retry failed because the loader key was not active: "+parseTrimmed)
		return fmt.Errorf("router: loader key %q is not active", parseTrimmed)
	}
	if parseEntry != nil && parseEntry.cancel != nil {
		parseEntry.cancel()
	}
	delete(parseR.loaderState.entries, parseTrimmed)
	parseR.loaderState.mu.Unlock()
	parseR.renderCurrentRoute(false)
	return nil
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
	var parseRouteElement *Element
	var parseRouteRecovered interface{}
	func() {
		defer func() {
			parseRouteRecovered = recover()
		}()
		parseRouteElement = parseR.currentElement(isApplyGuards)
	}()
	if parseRouteRecovered != nil {
		if !strings.Contains(fmt.Sprint(parseRouteRecovered), "called outside component context") {
			panic(parseRouteRecovered)
		}
		if parseR.renderCurrentComponent == nil {
			parseR.renderCurrentComponent = func() *Element {
				return parseR.currentElement(parseR.isRenderApplyGuards)
			}
		}
		parseR.isRenderApplyGuards = isApplyGuards
		parseRouteElement = runtime.CreateElement(parseR.renderCurrentComponent, map[string]interface{}{
			"version": parseR.renderVersion,
		})
		parseR.renderVersion++
	}
	if parseRouteElement == nil {
		return
	}
	// Publish the active location so UseRoute/UseLocation subscribers (e.g.
	// memoized chrome) re-render on navigation (G6). Done before the render so a
	// subscribed fiber marked dirty here is re-rendered in this same pass.
	publishLocation()
	// Wrap the DOM swap in a view transition (auto-skipped under reduced motion) and move focus
	// to the new route content afterward, both built-in defaults (FA5 + D4).
	parseR.withRouteTransition(func() {
		switch {
		case parseR.targetSelector != "":
			parseRt.RenderTo(parseR.targetSelector, parseRouteElement)
		case parseR.targetElement.Truthy():
			parseRt.Render(parseRouteElement, jsdom.NewWASMDOMNode(parseR.targetElement))
		}
		parseR.focusRouteContent()
	})
}

// SetViewTransitions enables or disables the automatic View Transitions animation on
// navigation. It is on by default; disable it to opt a router out of route-change animation.
func (parseR *Router) SetViewTransitions(parseEnabled bool) {
	parseR.viewTransitions = parseEnabled
}

// SetFocusManagement enables or disables automatic focus movement to the new route's content
// after navigation. It is on by default; disable it only if the app manages focus itself.
func (parseR *Router) SetFocusManagement(parseEnabled bool) {
	parseR.focusManagement = parseEnabled
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

	// History routers already register both popstate and hashchange listeners in
	// setupHistoryListener (called from NewHistoryRouter). Adding another hashchange
	// listener here would cause double re-renders on every hash navigation.
	if parseR.routerType == routerTypeHistory {
		return
	}

	parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer runtime.RecoverContainedPanic("router", "ensureListener callback")
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

// GetCurrentPath returns the current route path from this router instance. It mirrors the
// package-level GetCurrentPath and is the preferred name over GetCurrentRouterPath.
func (parseR *Router) GetCurrentPath() string {
	return parseR.GetCurrentRouterPath()
}

// GetCurrentRouterPath returns the current path from a router instance based on its type.
//
// Deprecated: use GetCurrentPath, which mirrors the package-level GetCurrentPath name.
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

// buildNavigationTarget normalizes a navigation target for the active router type.
func (parseR *Router) buildNavigationTarget(parseTarget string) string {
	if parseR.routerType != routerTypeHistory {
		return normalizeNavigationTarget(parseTarget)
	}
	parseTrimmed := strings.TrimSpace(parseTarget)
	if parseTrimmed == "" {
		return rootRoutePath
	}
	if strings.HasPrefix(parseTrimmed, "#") {
		// Preserve hash-only targets against the current history entry instead of treating
		// "#plans" as a route path. That regression caused history-router apps to lose
		// in-page anchor behavior on click and on browser back/forward.
		return buildCurrentHistoryTarget() + normalizeHistoryFragment(parseTrimmed)
	}
	parsePathTarget, parseFragment := splitNavigationTarget(parseTrimmed)
	// Keep fragments attached to full history-router targets as well. This is the core
	// behavior apps expect from links like "/pricing?plan=team#faq".
	return normalizeNavigationTarget(parsePathTarget) + normalizeHistoryFragment(parseFragment)
}

// Navigate navigates to a path using the appropriate method for this router type.
func (parseR *Router) Navigate(parsePath string) {
	parseCtx, parseAttemptID := parseR.beginGuardAttempt()
	defer parseR.finishGuardAttempt(parseAttemptID)

	parseNormalized, parseOk := parseR.evaluateNavigationWithAttempt(parseCtx, parseAttemptID, parseR.buildNavigationTarget(parsePath))
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

	parseNormalized, parseOk := parseR.evaluateNavigationWithAttempt(parseCtx, parseAttemptID, parseR.buildNavigationTarget(parsePath))
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

// RetryLoader clears one loader entry by key and re-runs the current route render.
func RetryLoader(parseKey string) error {
	return GetRouter().RetryLoader(parseKey)
}

// GetCurrentPath returns the current route path from the global router.
func GetCurrentPath() string {
	return GetRouter().GetCurrentRouterPath()
}
