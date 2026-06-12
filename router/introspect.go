//go:build js && wasm

package router

import "sort"

// RegisteredRoutes returns a sorted, de-duplicated slice of all registered
// route path strings for the given Router. Only exact-path routes stored in
// the routes map are included; catch-all ("*") and dynamic-pattern routes
// (those stored in patterns) are excluded because they are not discrete
// navigable paths. A nil receiver returns a non-nil empty slice.
func (parseR *Router) RegisteredRoutes() []string {
	if parseR == nil {
		return []string{}
	}
	parseSeen := make(map[string]struct{}, len(parseR.routes))
	parseResult := make([]string, 0, len(parseR.routes))
	for parsePath := range parseR.routes {
		if _, parseAlready := parseSeen[parsePath]; parseAlready {
			continue
		}
		parseSeen[parsePath] = struct{}{}
		parseResult = append(parseResult, parsePath)
	}
	sort.Strings(parseResult)
	return parseResult
}

// RegisteredRoutes returns a sorted, de-duplicated slice of all registered
// route path strings from the global router. It delegates to
// GetRouter().RegisteredRoutes() and returns a non-nil empty slice when no
// routes are registered or the global router is nil.
func RegisteredRoutes() []string {
	parseRouter := GetRouter()
	if parseRouter == nil {
		return []string{}
	}
	return parseRouter.RegisteredRoutes()
}
