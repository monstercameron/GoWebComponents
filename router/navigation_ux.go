//go:build js && wasm

package router

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

// withRouteTransition runs apply (the route's DOM swap) inside the browser View Transitions API
// so navigation animates between the old and new route, when the router has view transitions
// enabled (the default), the API exists, and the user has NOT requested reduced motion.
// Otherwise it calls apply directly. This makes animated route changes a built-in default rather
// than something every caller must wire by hand (FA5).
func (parseR *Router) withRouteTransition(parseApply func()) {
	if parseApply == nil {
		return
	}
	parseDocument := js.Global().Get("document")
	if !parseR.viewTransitions || !parseDocument.Truthy() {
		parseApply()
		return
	}
	parseStart := parseDocument.Get("startViewTransition")
	if parseStart.Type() != js.TypeFunction || routerPrefersReducedMotion() {
		parseApply()
		return
	}
	var parseCallback js.Func
	parseCallback = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		defer runtime.RecoverContainedPanic("router", "view transition apply")
		defer parseCallback.Release()
		parseApply()
		return nil
	})
	parseDocument.Call("startViewTransition", parseCallback)
}

// focusRouteContent moves keyboard focus to the new route's content after navigation, so screen
// reader and keyboard users land on the freshly rendered page instead of being stranded where
// the old page's focus was (D4). It honors an explicit focus target ([autofocus] or
// [data-route-focus]); otherwise it focuses the route container itself, making it programmatically
// focusable with tabindex=-1 when needed. No-op when focus management is disabled.
func (parseR *Router) focusRouteContent() {
	if !parseR.focusManagement {
		return
	}
	parseContainer := parseR.routeContainerElement()
	if !parseContainer.Truthy() {
		return
	}
	parseTarget := firstTruthyQuery(parseContainer, "[autofocus]", "[data-route-focus]")
	if !parseTarget.Truthy() {
		parseTarget = parseContainer
	}
	if !parseTarget.Get("hasAttribute").Truthy() || !parseTarget.Call("hasAttribute", "tabindex").Bool() {
		if parseTarget.Get("setAttribute").Truthy() {
			parseTarget.Call("setAttribute", "tabindex", "-1")
		}
	}
	if parseTarget.Get("focus").Truthy() {
		parseFocusOptions := js.Global().Get("Object").New()
		parseFocusOptions.Set("preventScroll", false)
		parseTarget.Call("focus", parseFocusOptions)
	}
}

// routeContainerElement resolves the element the router renders into.
func (parseR *Router) routeContainerElement() js.Value {
	if parseR.targetElement.Truthy() {
		return parseR.targetElement
	}
	if parseR.targetSelector != "" {
		parseDocument := js.Global().Get("document")
		if parseDocument.Truthy() && parseDocument.Get("querySelector").Truthy() {
			return parseDocument.Call("querySelector", parseR.targetSelector)
		}
	}
	return js.Value{}
}

// firstTruthyQuery returns the first element matching any of the selectors within root.
func firstTruthyQuery(parseRoot js.Value, parseSelectors ...string) js.Value {
	if !parseRoot.Truthy() || !parseRoot.Get("querySelector").Truthy() {
		return js.Value{}
	}
	for _, parseSelector := range parseSelectors {
		parseMatch := parseRoot.Call("querySelector", parseSelector)
		if parseMatch.Truthy() {
			return parseMatch
		}
	}
	return js.Value{}
}

// routerPrefersReducedMotion reports whether the user has requested reduced motion, so the
// router can skip the navigation animation.
func routerPrefersReducedMotion() bool {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() || !parseWindow.Get("matchMedia").Truthy() {
		return false
	}
	parseQuery := parseWindow.Call("matchMedia", "(prefers-reduced-motion: reduce)")
	if !parseQuery.Truthy() {
		return false
	}
	return parseQuery.Get("matches").Truthy()
}
