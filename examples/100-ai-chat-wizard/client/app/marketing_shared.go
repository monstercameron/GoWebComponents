//go:build js && wasm

package app

import (
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderLandingFooterColumn renders a titled footer link column.
func renderLandingFooterColumn(title string, links ...ui.Node) ui.Node {
	ulArgs := make([]interface{}, 0, len(links)+1)
	ulArgs = append(ulArgs, Class("mt-4 space-y-3 text-sm text-[#b8c2d9] sm:mt-5"))
	for _, l := range links {
		ulArgs = append(ulArgs, l)
	}
	return Div(
		Div(Class("text-[10px] font-semibold uppercase tracking-[0.16em] text-[#dfe6f7] sm:text-[11px] sm:tracking-[0.18em]"), Text(title)),
		Ul(ulArgs...),
	)
}

// renderLandingFooterLink renders a single footer nav link as a list item.
func renderLandingFooterLink(label, href string) ui.Node {
	return Li(A(Class("transition hover:text-white"), Href(href), Text(label)))
}

// landingNavigateHandler returns a click handler that performs client-side router navigation,
// respecting modifier keys and default-prevented events so browser behaviour is preserved.
func landingNavigateHandler(targetPath string) func(ui.Event) {
	normalizedTarget := strings.TrimSpace(targetPath)
	return func(e ui.Event) {
		jsEvent := e.JSValue()
		if jsEvent.Truthy() {
			if jsEvent.Get("defaultPrevented").Bool() {
				return
			}
			if jsEvent.Get("button").Int() != 0 {
				return
			}
			if jsEvent.Get("metaKey").Bool() || jsEvent.Get("ctrlKey").Bool() || jsEvent.Get("shiftKey").Bool() || jsEvent.Get("altKey").Bool() {
				return
			}
		}

		e.PreventDefault()
		if normalizedTarget == "" {
			return
		}
		currentPath := strings.TrimSpace(router.GetCurrentPath())
		if currentPath == normalizedTarget {
			return
		}
		// treat /home and authLandingRoute as the same destination to avoid a redundant navigation
		if normalizedTarget == authLandingRoute && currentPath == marketingHomeRoute {
			return
		}
		router.Navigate(normalizedTarget)
	}
}
