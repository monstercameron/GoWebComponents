//go:build js && wasm
// +build js,wasm

package router

import (
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/internal/platform/jsdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

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
var cleanupMu sync.Mutex
var cleanupHandlers []js.Func

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

// registerCleanup registers a js.Func to be released when the page unloads.
// All registered handlers are released together by a single beforeunload listener.
func registerCleanup(parseHandler js.Func) {
	cleanupMu.Lock()
	cleanupHandlers = append(cleanupHandlers, parseHandler)
	cleanupMu.Unlock()

	cleanupOnce.Do(func() {
		parseWindow := js.Global().Get("window")
		if !parseWindow.Truthy() || !parseWindow.Get("addEventListener").Truthy() {
			return
		}
		// Register a single unload listener that releases all accumulated handlers.
		parseUnload := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			defer runtime.RecoverContainedPanic("router", "registerCleanup callback")
			cleanupMu.Lock()
			parseAll := cleanupHandlers
			cleanupHandlers = nil
			cleanupMu.Unlock()
			for _, parseH := range parseAll {
				parseH.Release()
			}
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
