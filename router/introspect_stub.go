//go:build !js || !wasm

package router

// RegisteredRoutes returns a non-nil empty route list outside the browser
// runtime, where the active router table (and the Router type) are not
// available. The *Router method lives only in the js/wasm build alongside the
// type itself.
func RegisteredRoutes() []string {
	return []string{}
}
