//go:build js && wasm

package app

import (
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderLandingFooterColumn renders a titled footer link column.
func renderLandingFooterColumn(renderTitle string, renderLinks ...ui.Node) ui.Node {
	renderUlArgs := make([]interface{}, 0, len(renderLinks)+1)
	renderUlArgs = append(renderUlArgs, Class("mt-4 space-y-3 text-sm text-[#b8c2d9] sm:mt-5"))
	for _, renderLink := range renderLinks {
		renderUlArgs = append(renderUlArgs, renderLink)
	}
	return Div(
		Div(Class("text-[10px] font-semibold uppercase tracking-[0.16em] text-[#dfe6f7] sm:text-[11px] sm:tracking-[0.18em]"), Text(renderTitle)),
		Ul(renderUlArgs...),
	)
}

// renderMarketingHeaderShell renders the shared outer header layout for marketing pages.
func renderMarketingHeaderShell(renderBrand, renderNav, renderActions ui.Node) ui.Node {
	return Header(
		Class("relative z-20"),
		Div(
			Class("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-wrap items-center justify-between gap-4 py-5 sm:w-[min(1200px,calc(100%-32px))] sm:py-6 lg:w-[min(1200px,calc(100%-40px))] lg:flex-nowrap lg:py-7"),
			renderBrand,
			renderNav,
			renderActions,
		),
	)
}

// renderMarketingHeaderBrand renders the shared RelayDesk brand treatment for marketing page headers.
func renderMarketingHeaderBrand(renderSubtitle, renderTargetPath string) ui.Node {
	if strings.TrimSpace(renderTargetPath) == "" {
		return Div(
			Class("flex min-w-0 items-center gap-3 sm:gap-4"),
			Div(
				Class("grid h-10 w-10 shrink-0 place-items-center rounded-2xl bg-[linear-gradient(135deg,#c4b5fd_0%,#f9a8d4_100%)] text-sm font-black text-[#1a1330] sm:h-11 sm:w-11"),
				Text("RD"),
			),
			Div(
				Class("min-w-0"),
				Div(Class("truncate text-[14px] font-semibold tracking-[-0.01em] sm:text-[15px]"), Text("RelayDesk")),
				Div(Class("truncate text-[10px] uppercase tracking-[0.16em] text-[#b8c2d9] sm:text-[11px] sm:tracking-[0.18em]"), Text(renderSubtitle)),
			),
		)
	}
	return A(
		Class("flex min-w-0 items-center gap-3 sm:gap-4"),
		Href(renderTargetPath),
		OnClick(landingNavigateHandler(renderTargetPath)),
		Div(
			Class("grid h-10 w-10 shrink-0 place-items-center rounded-2xl bg-[linear-gradient(135deg,#c4b5fd_0%,#f9a8d4_100%)] text-sm font-black text-[#1a1330] sm:h-11 sm:w-11"),
			Text("RD"),
		),
		Div(
			Class("min-w-0"),
			Div(Class("truncate text-[14px] font-semibold tracking-[-0.01em] sm:text-[15px]"), Text("RelayDesk")),
			Div(Class("truncate text-[10px] uppercase tracking-[0.16em] text-[#b8c2d9] sm:text-[11px] sm:tracking-[0.18em]"), Text(renderSubtitle)),
		),
	)
}

// renderMarketingHeaderAction renders a shared header CTA link for marketing pages.
func renderMarketingHeaderAction(renderLabel, renderTargetPath string, isPrimary, isHiddenOnSmall bool) ui.Node {
	return A(
		Class(ClassNames(
			"inline-flex items-center justify-center rounded-full px-4 py-2.5 text-sm transition sm:px-5",
			When(isPrimary, "flex-1 bg-white font-semibold text-[#1a1330] hover:-translate-y-[1px] sm:flex-none"),
			When(!isPrimary, "bg-white/10 font-medium text-[#dfe6f7] hover:bg-white/15"),
			When(isHiddenOnSmall, "hidden sm:inline-flex"),
		)),
		Href(renderTargetPath),
		OnClick(landingNavigateHandler(renderTargetPath)),
		Text(renderLabel),
	)
}

// renderMarketingHeroHeading renders the shared eyebrow, headline, body, and CTA row used by marketing heroes.
func renderMarketingHeroHeading(renderEyebrow, renderHeadline, renderBody string, renderActions ...ui.Node) ui.Node {
	renderArgs := make([]interface{}, 0, len(renderActions)+1)
	renderArgs = append(renderArgs, Class("mt-8 flex flex-col gap-3 sm:mt-9 sm:flex-row"))
	for _, renderAction := range renderActions {
		renderArgs = append(renderArgs, renderAction)
	}

	return Div(
		Div(
			Class("mb-5 inline-flex rounded-full bg-white/10 px-3 py-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:mb-7 sm:px-4 sm:text-[11px] sm:tracking-[0.18em]"),
			Text(renderEyebrow),
		),
		H1(
			Class("max-w-none text-4xl font-semibold leading-[0.95] tracking-[-0.055em] text-white sm:max-w-[11ch] sm:text-5xl md:text-6xl xl:text-7xl"),
			Text(renderHeadline),
		),
		P(
			Class("mt-5 max-w-[58ch] text-base leading-7 text-[#e6ebf8]/92 sm:mt-6 sm:text-lg sm:leading-8 lg:text-xl"),
			Text(renderBody),
		),
		Div(renderArgs...),
	)
}

// renderLandingFooterLink renders a single footer nav link as a list item.
func renderLandingFooterLink(renderLabel, renderHref string) ui.Node {
	return Li(A(Class("transition hover:text-white"), Href(renderHref), Text(renderLabel)))
}

// landingNavigateHandler returns a click handler that performs client-side router navigation,
// respecting modifier keys and default-prevented events so browser behaviour is preserved.
func landingNavigateHandler(routeTargetPath string) func(ui.Event) {
	routeNormalizedTarget := strings.TrimSpace(routeTargetPath)
	return func(e ui.Event) {
		routeJSEvent := e.JSValue()
		if routeJSEvent.Truthy() {
			if routeJSEvent.Get("defaultPrevented").Bool() {
				return
			}
			if routeJSEvent.Get("button").Int() != 0 {
				return
			}
			if routeJSEvent.Get("metaKey").Bool() || routeJSEvent.Get("ctrlKey").Bool() || routeJSEvent.Get("shiftKey").Bool() || routeJSEvent.Get("altKey").Bool() {
				return
			}
		}

		e.PreventDefault()
		if routeNormalizedTarget == "" {
			return
		}
		routeCurrentPath := strings.TrimSpace(router.GetCurrentPath())
		if routeCurrentPath == routeNormalizedTarget {
			return
		}
		// treat /home and authLandingRoute as the same destination to avoid a redundant navigation
		if routeNormalizedTarget == authLandingRoute && routeCurrentPath == marketingHomeRoute {
			return
		}
		router.Navigate(routeNormalizedTarget)
	}
}
