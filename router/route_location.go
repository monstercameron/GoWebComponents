package router

// Location is a snapshot of the active route used by the reactive UseRoute /
// UseLocation hooks (G6). Unlike InspectCurrentRoute (a non-reactive read), a
// component that calls UseRoute subscribes to navigation and re-renders when the
// location changes — so memoized chrome (active-nav highlight, breadcrumb) stays
// in sync without threading the path down as a prop.
type Location struct {
	// Path is the current logical route path (no query string).
	Path string
	// Query is the encoded query string (without the leading '?').
	Query string
	// Params are the params captured by the matched route pattern.
	Params map[string]string
}

// Param returns the captured route param for key, or "" if absent.
func (parseLoc Location) Param(parseKey string) string {
	if parseLoc.Params == nil {
		return ""
	}
	return parseLoc.Params[parseKey]
}
