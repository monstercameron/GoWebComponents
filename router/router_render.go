//go:build js && wasm

package router

import (
	"context"
	"net/url"
	"strings"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

func (parseR *Router) renderResolvedRouteStack(parseRoutes []resolvedRoute, parseQuery url.Values, parseQueryKey string, isApplyGuards bool, parseGuardCtx context.Context, parseAttemptID uint64) *Element {
	return parseR.renderResolvedRouteStackWithDepth(parseRoutes, parseQuery, parseQueryKey, isApplyGuards, parseGuardCtx, parseAttemptID, 0)
}

// renderResolvedRouteStackWithDepth is an internal router helper that tracks redirect
// depth to prevent infinite redirect loops (#44/#45).
func (parseR *Router) renderResolvedRouteStackWithDepth(parseRoutes []resolvedRoute, parseQuery url.Values, parseQueryKey string, isApplyGuards bool, parseGuardCtx context.Context, parseAttemptID uint64, parseRedirectDepth int) *Element {
	parseLoaderKeys := make([]string, 0, len(parseRoutes))
	parseChunkKeys := make([]string, 0, len(parseRoutes))
	for _, parseRoute := range parseRoutes {
		if parseRoute.option.Loader != nil {
			parseLoaderKeys = append(parseLoaderKeys, buildLoaderKey(parseRoute.id, parseRoute.path, parseQueryKey))
		}
		if !parseRoute.option.Chunk.empty() {
			parseChunkKeys = append(parseChunkKeys, parseRoute.option.Chunk.key(parseRoute.id))
		}
	}
	parseR.prepareLoaderState(parseLoaderKeys)
	parseR.prepareChunkState(parseChunkKeys)
	// Apply document metadata ONCE for the whole matched stack, with the
	// deepest non-empty value winning per field. Applying it per level (as it
	// used to) made a leaf's empty Title/Description/Canonical reset the
	// document back to the base value, silently erasing a layout route's
	// metadata on every normal layout+outlet navigation.
	parseR.applyRouteMetadata(mergeRouteMetadata(parseRoutes))
	return parseR.renderRouteLevel(parseRoutes, 0, parseQuery, parseQueryKey, isApplyGuards, parseGuardCtx, parseAttemptID, parseRedirectDepth)
}

// mergeRouteMetadata collapses the matched route stack into the effective
// document metadata: for Title, Description, and CanonicalURL the deepest
// (most specific) route that sets a non-empty value wins, and a field no route
// sets stays empty so applyRouteMetadata resets it to the captured base.
func mergeRouteMetadata(parseRoutes []resolvedRoute) Options {
	var parseMerged Options
	for _, parseRoute := range parseRoutes {
		if strings.TrimSpace(parseRoute.option.Title) != "" {
			parseMerged.Title = parseRoute.option.Title
		}
		if strings.TrimSpace(parseRoute.option.Description) != "" {
			parseMerged.Description = parseRoute.option.Description
		}
		if strings.TrimSpace(parseRoute.option.CanonicalURL) != "" {
			parseMerged.CanonicalURL = parseRoute.option.CanonicalURL
		}
	}
	return parseMerged
}

// renderRouteLevel is an internal router helper.
func (parseR *Router) renderRouteLevel(parseRoutes []resolvedRoute, parseIndex int, parseQuery url.Values, parseQueryKey string, isApplyGuards bool, parseGuardCtx context.Context, parseAttemptID uint64, parseRedirectDepth int) *Element {
	parseMatch := parseRoutes[parseIndex]
	if isApplyGuards {
		if parseBlocked := parseR.applyBeforeEnterGuard(parseMatch.path, parseMatch.option, parseMatch.params, parseQuery, parseGuardCtx, parseAttemptID, parseRedirectDepth); parseBlocked != nil {
			return parseBlocked
		}
		// Propagate guard cancellation: if a sibling or ancestor cancelled the attempt,
		// abort mid-recursion rather than rendering a partial layout (#55).
		if (parseGuardCtx != nil && parseGuardCtx.Err() != nil) || (parseAttemptID != 0 && !parseR.guardAttemptActive(parseAttemptID)) {
			return nil
		}
	}
	if parseRedirected := parseR.applyRouteOptions(parseMatch.path, parseMatch.option, parseQuery, isApplyGuards, parseGuardCtx, parseAttemptID, parseRedirectDepth); parseRedirected != nil {
		return parseRedirected
	}

	parseBaseProps := copyParamsToAttrs(parseMatch.params)
	if !parseMatch.option.Chunk.empty() {
		parseChunkKey := parseMatch.option.Chunk.key(parseMatch.id)
		parseState := parseR.ensureRouteChunkResult(parseChunkKey, parseMatch.option.Chunk, RouteContext{
			Path:   parseMatch.path,
			Params: Params{values: copyParams(parseMatch.params)},
			Query:  Query{values: copyQueryValues(parseQuery)},
		})
		if parseState.pending {
			currentRouteData = nil
			currentRouteOutlet = nil
			return renderRouteFallback(parseMatch.option.Chunk.loadingComponent(parseMatch.option), mergeAttrs(parseBaseProps, Attrs{"path": parseMatch.path, "loading": true, "chunk": true}))
		}
		if parseState.err != nil {
			currentRouteData = nil
			currentRouteOutlet = nil
			return renderRouteError(parseMatch.option.Chunk.errorComponent(parseMatch.option), parseState.err, mergeAttrs(parseBaseProps, Attrs{"path": parseMatch.path, "error": parseState.err.Error(), "chunk": true}))
		}
	}
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
		parseOutlet = parseR.renderRouteLevel(parseRoutes, parseIndex+1, parseQuery, parseQueryKey, isApplyGuards, parseGuardCtx, parseAttemptID, parseRedirectDepth)
		// Propagate cancellation from nested levels (#55): if the child render aborted
		// due to a guard cancellation, the whole stack should abort too.
		if isApplyGuards && parseOutlet == nil && parseIndex+1 < len(parseRoutes) {
			if (parseGuardCtx != nil && parseGuardCtx.Err() != nil) || (parseAttemptID != 0 && !parseR.guardAttemptActive(parseAttemptID)) {
				return nil
			}
		}
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
		// The request path did not match, so we are falling back to the default
		// route. Wrap it in the SAME layout stack it gets when navigated to directly
		// — resolve by the default route's OWN path (which exact-matches and so does
		// not re-enter this branch, terminating the recursion) rather than rendering
		// it bare. Without this, a fallback render skipped all parent Layout routes.
		if parseStack := parseR.resolveRouteStack(parseR.defaultRoute); parseStack.found {
			return parseStack
		}
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
func (parseR *Router) applyBeforeEnterGuard(parsePath string, parseOption Options, parseParams map[string]string, parseQuery url.Values, parseGuardCtx context.Context, parseAttemptID uint64, parseRedirectDepth int) *Element {
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
		// Abort guard-redirect chains that exceed the depth limit (e.g. two guards
		// redirecting to each other) instead of recursing forever.
		if parseRedirectDepth >= maxRedirectDepth {
			runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "redirect depth limit reached, aborting before-enter redirect from "+parsePath+" to "+parseNormalized)
			return nil
		}
		parseR.replaceLocation(parseNormalized)
		// Evaluate guards on the redirect target (#45) WITHIN the same guard attempt:
		// starting a new attempt via currentElement would cancel the outer attempt's
		// context, making the caller discard the rendered redirect target.
		return parseR.renderRedirectTarget(true, parseGuardCtx, parseAttemptID, parseRedirectDepth+1)
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

// applyRouteOptions is an internal router helper. Metadata is applied once for
// the whole stack in renderResolvedRouteStackWithDepth (see mergeRouteMetadata),
// so this handles only per-level redirect resolution.
func (parseR *Router) applyRouteOptions(parsePath string, parseOption Options, parseQuery url.Values, isApplyGuards bool, parseGuardCtx context.Context, parseAttemptID uint64, parseRedirectDepth int) *Element {
	if parseOption.Redirect == "" {
		return nil
	}

	parseCurrentTarget := buildPathWithQuery(parsePath, parseQuery)
	parseRedirectTarget := normalizeNavigationTarget(parseOption.Redirect)
	if parseRedirectTarget == parseCurrentTarget {
		runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "ignoring route redirect loop for "+parseRedirectTarget)
		return nil
	}

	// Abort redirect chains that exceed the depth limit to prevent infinite loops.
	if parseRedirectDepth >= maxRedirectDepth {
		runtime.ReportDiagnostic("router", runtime.DiagnosticWarning, "redirect depth limit reached, aborting redirect from "+parseCurrentTarget+" to "+parseRedirectTarget)
		return nil
	}

	runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "route redirect applied", "", map[string]string{
		"from": parseCurrentTarget,
		"to":   parseRedirectTarget,
	})
	parseR.recordRedirectDebug("route-option", parseCurrentTarget, parseRedirectTarget)
	parseR.replaceLocation(parseRedirectTarget)
	// Evaluate guards on the redirect target (#44) WITHIN the caller's guard attempt:
	// starting a new attempt via currentElement would cancel the outer attempt's
	// context, making the caller discard the rendered redirect target.
	return parseR.renderRedirectTarget(isApplyGuards, parseGuardCtx, parseAttemptID, parseRedirectDepth+1)
}

// renderRedirectTarget re-resolves the (already replaced) location and renders the
// resulting route stack within the SAME guard attempt as the caller. It mirrors the
// resolution logic of currentElementWithDepth but deliberately does not call
// beginGuardAttempt, which would cancel the outer attempt's context (#44/#45).
func (parseR *Router) renderRedirectTarget(isApplyGuards bool, parseGuardCtx context.Context, parseAttemptID uint64, parseRedirectDepth int) *Element {
	parsePath := parseR.GetCurrentRouterPath()
	if parsePath == "" {
		parsePath = parseR.defaultRoute
	}
	parseQuery := getCurrentQueryValues()
	parseQueryKey := parseQuery.Encode()
	parseResolved := parseR.resolveRouteStack(parsePath)
	if !parseResolved.found {
		parseR.cancelLoaderIfActive()
		currentRouteData = nil
		currentRouteOutlet = nil
		return runtime.Div(nil, runtime.Text(routeNotFoundText))
	}
	parseLeaf := parseResolved.routes[len(parseResolved.routes)-1]
	currentParams = copyParams(parseLeaf.params)
	return parseR.renderResolvedRouteStackWithDepth(parseResolved.routes, parseQuery, parseQueryKey, isApplyGuards, parseGuardCtx, parseAttemptID, parseRedirectDepth)
}
