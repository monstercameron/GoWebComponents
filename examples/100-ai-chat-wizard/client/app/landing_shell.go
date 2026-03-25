//go:build js && wasm

package app

import (
	"strings"
	"syscall/js"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

const landingPageHome = "home"
const landingPageCapabilities = "capabilities"
const landingPagePricing = "pricing"

// setLandingDocumentTitle updates the browser tab title to match the current marketing page.
func setLandingDocumentTitle(page string) {
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		return
	}
	var title string
	switch page {
	case landingPagePricing:
		title = "RelayDesk – Pricing"
	case landingPageCapabilities:
		title = "RelayDesk – Capabilities"
	default:
		title = "RelayDesk – AI Chat Workspace"
	}
	doc.Set("title", title)
}

// renderLandingShell renders the full marketing landing page with header, content sections, and footer.
// For the pricing route it delegates to renderPricingShell, which has its own richer layout.
func renderLandingShell(_ i18n.Runtime, view appViewState, _ authSessionController) ui.Node {
	page := landingPageForPath(view.CurrentPath)
	// keep the browser tab title in sync with whichever marketing page is active
	setLandingDocumentTitle(page)
	if page == landingPagePricing {
		return renderPricingShell(view)
	}
	return Div(
		// dark gradient background with purple/pink radial glows
		Class("relative min-h-screen bg-[radial-gradient(circle_at_12%_10%,rgba(139,92,246,.18),transparent_24%),radial-gradient(circle_at_88%_14%,rgba(236,72,153,.16),transparent_26%),linear-gradient(180deg,#121726_0%,#171c2d_48%,#1b2135_100%)] text-[#f5f7fb] antialiased"),
		Div(
			Class("pointer-events-none fixed inset-0 overflow-hidden"),
			Div(Class("absolute left-[6%] top-[6%] h-40 w-40 rounded-full bg-[#8b5cf6]/12 blur-3xl sm:h-56 sm:w-56 lg:h-64 lg:w-64"), nil),
			Div(Class("absolute right-[8%] top-[10%] h-44 w-44 rounded-full bg-[#ec4899]/12 blur-3xl sm:h-60 sm:w-60 lg:h-72 lg:w-72"), nil),
		),
		renderLandingHeader(view.CurrentPath),
		Main(
			Class("relative z-10"),
			renderLandingHeroSection(page),
			renderLandingProductSection(page),
			renderLandingWhySection(page),
			renderLandingPricingSection(page),
		),
		renderLandingFooter(),
	)
}

// renderLandingHeader renders the top header bar with brand, nav, and CTA buttons.
func renderLandingHeader(currentPath string) ui.Node {
	return Header(
		Class("relative z-20"),
		Div(
			Class("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-wrap items-center justify-between gap-4 py-5 sm:w-[min(1200px,calc(100%-32px))] sm:py-6 lg:w-[min(1200px,calc(100%-40px))] lg:flex-nowrap lg:py-7"),
			// brand
			Div(
				Class("flex min-w-0 items-center gap-3 sm:gap-4"),
				Div(
					Class("grid h-10 w-10 shrink-0 place-items-center rounded-2xl bg-[linear-gradient(135deg,#c4b5fd_0%,#f9a8d4_100%)] text-sm font-black text-[#1a1330] sm:h-11 sm:w-11"),
					Text("RD"),
				),
				Div(
					Class("min-w-0"),
					Div(Class("truncate text-[14px] font-semibold tracking-[-0.01em] sm:text-[15px]"), Text("RelayDesk")),
					Div(Class("truncate text-[10px] uppercase tracking-[0.16em] text-[#b8c2d9] sm:text-[11px] sm:tracking-[0.18em]"), Text("Clear AI for real work")),
				),
			),
			// nav — router links to home and pricing
			Tag("nav",
				Class("hidden items-center gap-5 lg:flex xl:gap-8"),
				landingNavLink(currentPath, marketingHomeRoute, "Product"),
				landingNavLink(currentPath, marketingPricingRoute, "Pricing"),
			),
			// actions
			Div(
				Class("flex w-full items-center gap-2 sm:gap-3 md:w-auto"),
				A(
					Class("hidden rounded-full bg-white/10 px-4 py-2 text-sm font-medium text-[#dfe6f7] transition hover:bg-white/15 sm:inline-flex"),
					Href(authLandingRoute),
					Text("Log in"),
				),
				A(
					Class("inline-flex flex-1 items-center justify-center rounded-full bg-white px-4 py-2.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px] sm:flex-none sm:px-5"),
					Href(chatRouteRoot),
					OnClick(landingNavigateHandler(chatRouteRoot)),
					Text("Open chat"),
				),
			),
		),
	)
}

// landingPageForPath maps a URL path to the landing page variant constant.
func landingPageForPath(path string) string {
	switch strings.TrimSpace(path) {
	case marketingCapabilitiesRoute:
		return landingPageCapabilities
	case marketingPricingRoute:
		return landingPagePricing
	case marketingHomeRoute, authLandingRoute:
		return landingPageHome
	default:
		return landingPageHome
	}
}

// landingNavLink renders a router-aware nav link, styling it active when its route matches the current path.
func landingNavLink(currentPath, targetPath, label string) ui.Node {
	isActive := strings.TrimSpace(currentPath) == targetPath ||
		(targetPath == authLandingRoute && strings.TrimSpace(currentPath) == marketingHomeRoute)
	return A(
		Class(ClassNames(
			"text-sm transition",
			When(isActive, "text-white"),
			When(!isActive, "text-[#b8c2d9] hover:text-white"),
		)),
		Href(targetPath),
		OnClick(landingNavigateHandler(targetPath)),
		Text(label),
	)
}

// landingActionButton renders a rounded-pill CTA button, primary (white fill) or secondary (glass).
func landingActionButton(label, targetPath string, primary bool) ui.Node {
	return A(
		Class(ClassNames(
			"inline-flex items-center justify-center rounded-full px-5 py-3 text-sm font-semibold transition sm:px-6 sm:py-3.5",
			When(primary, "bg-white text-[#1a1330] hover:-translate-y-[1px]"),
			When(!primary, "bg-white/10 text-white hover:bg-white/15"),
		)),
		Href(targetPath),
		OnClick(landingNavigateHandler(targetPath)),
		Text(label),
	)
}

// renderLandingFooter renders the site footer with the brand blurb, link columns, and copyright bar.
func renderLandingFooter() ui.Node {
	return Tag("footer",
		Class("relative z-10 bg-transparent"),
		// main footer grid: brand column + three link columns
		Div(
			Class("mx-auto grid w-[min(1200px,calc(100%-24px))] gap-8 py-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-10 sm:py-12 md:grid-cols-2 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[1.2fr_.8fr_.8fr_.8fr] lg:gap-12 lg:py-14"),
			// brand blurb
			Div(
				Class("max-w-[34ch] md:col-span-2 lg:col-span-1"),
				Div(
					Class("flex items-center gap-4"),
					Div(Class("grid h-10 w-10 place-items-center rounded-2xl bg-[linear-gradient(135deg,#c4b5fd_0%,#f9a8d4_100%)] text-sm font-black text-[#1a1330] sm:h-11 sm:w-11"), Text("RD")),
					Div(
						Div(Class("text-[14px] font-semibold tracking-[-0.01em] text-white sm:text-[15px]"), Text("RelayDesk")),
						Div(Class("text-[10px] uppercase tracking-[0.16em] text-[#b8c2d9] sm:text-[11px] sm:tracking-[0.18em]"), Text("Enterprise AI workspace")),
					),
				),
				P(Class("mt-5 text-sm leading-7 text-[#b8c2d9]"), Text("RelayDesk helps teams ask better questions, get clearer answers, and move work forward with less confusion.")),
			),
			renderLandingFooterColumn("Product",
				renderLandingFooterLink("Overview", "#product"),
				renderLandingFooterLink("Demo", "#demo"),
				renderLandingFooterLink("Pricing", "#pricing"),
				renderLandingFooterLink("Use Cases", "#why"),
			),
			renderLandingFooterColumn("Company",
				renderLandingFooterLink("About", "#"),
				renderLandingFooterLink("Customers", "#"),
				renderLandingFooterLink("Careers", "#"),
				renderLandingFooterLink("Contact", "#"),
			),
			renderLandingFooterColumn("Resources",
				renderLandingFooterLink("Documentation", "#"),
				renderLandingFooterLink("Security", "#"),
				renderLandingFooterLink("Privacy", "#"),
				renderLandingFooterLink("Terms", "#"),
			),
		),
		// copyright bar
		Div(
			Div(
				Class("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-col gap-3 py-5 text-xs text-[#b8c2d9] sm:w-[min(1200px,calc(100%-32px))] sm:gap-4 sm:py-6 md:flex-row md:items-center md:justify-between lg:w-[min(1200px,calc(100%-40px))]"),
				Div(Text("(c) 2026 RelayDesk, Inc. All rights reserved.")),
				Div(
					Class("flex flex-wrap items-center gap-4 sm:gap-5"),
					A(Class("transition hover:text-white"), Href("#"), Text("Privacy Policy")),
					A(Class("transition hover:text-white"), Href("#"), Text("Terms of Service")),
					A(Class("transition hover:text-white"), Href("#"), Text("Status")),
				),
			),
		),
	)
}
