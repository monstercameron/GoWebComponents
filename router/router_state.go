//go:build js && wasm

package router

import (
	"context"
	"net/url"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

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
		// url.ParseQuery returns the successfully-parsed pairs PLUS the first error
		// — keep the partial result so one malformed param (e.g. a bad %-escape)
		// doesn't silently discard every valid param alongside it.
		parseParsed, _ := url.ParseQuery(parseNormalized[parseIdx+1:])
		if len(parseParsed) > 0 {
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
		// Skip re-render if this router has been unmounted/replaced since the
		// loader was launched (finding #46: stale goroutine guard).
		if parseR.disposed {
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
	parseR.cancelRouteChunksIfActive()
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
