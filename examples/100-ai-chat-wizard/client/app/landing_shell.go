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
const landingPageSignup = "signup"

// setLandingDocumentTitle updates the browser tab title to match the current marketing page.
func setLandingDocumentTitle(parseIntl i18n.Runtime, parsePage string) {
	parseDoc := js.Global().Get("document")
	if !parseDoc.Truthy() {
		return
	}
	n := marketingI18nNamespace
	var parseTitle string
	switch parsePage {
	case landingPagePricing:
		parseTitle = parseIntl.T(n, "page.title.pricing")
	case landingPageCapabilities:
		parseTitle = parseIntl.T(n, "page.title.capabilities")
	default:
		parseTitle = parseIntl.T(n, "page.title.home")
	}
	parseDoc.Set("title", parseTitle)
}

// renderLandingShell renders the full marketing landing page with header, content sections, and footer.
// For the pricing route it delegates to renderPricingShell, which has its own richer layout.
// For the signup route it delegates to renderSignupShell.
func renderLandingShell(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController) ui.Node {
	n := marketingI18nNamespace
	parsePage := parseLandingPageForPath(parseView.CurrentPath)
	if parsePage == landingPageSignup {
		return renderSignupShell(parseIntl, parseView, parseAuth)
	}
	// keep the browser tab title in sync with whichever marketing page is active
	setLandingDocumentTitle(parseIntl, parsePage)
	if parsePage == landingPagePricing {
		return renderPricingShell(parseIntl, parseView)
	}
	return Div(
		Class("relative min-h-screen text-[#f0f0f8] antialiased page-bg"),
		renderPageBackground(),
		renderMarketingHeader(
			parseIntl,
			parseView.CurrentPath,
			Tag("nav",
				Class("hidden items-center gap-6 lg:flex"),
				renderNavLink(parseView.CurrentPath, marketingHomeRoute, parseIntl.T(n, "nav.product")),
				renderNavLink(parseView.CurrentPath, marketingPricingRoute, parseIntl.T(n, "nav.pricing")),
			),
			renderMarketingHeaderAction(parseIntl.T(n, "header.logIn"), authLandingRoute, false, true),
			renderMarketingHeaderAction(parseIntl.T(n, "header.signUp"), marketingSignupRoute, false, false),
			renderMarketingHeaderAction(parseIntl.T(n, "header.openApp"), chatRouteRoot, true, false),
		),
		Main(
			Class("relative z-10"),
			renderLandingHeroSection(parseIntl, parsePage),
			renderSectionDivider(parseIntl.T(n, "divider.capabilities")),
			renderLandingProductSection(parseIntl, parsePage),
			renderSectionDivider(parseIntl.T(n, "divider.whyTeamsBuy")),
			renderLandingWhySection(parseIntl, parsePage),
			renderSectionDivider(parseIntl.T(n, "divider.pricing")),
			renderLandingPricingSection(parseIntl, parsePage),
		),
		renderMarketingFooter(
			parseIntl,
			renderFooterColumn(parseIntl.T(n, "footer.col.product"),
				renderFooterLink(parseIntl.T(n, "footer.link.overview"), "#product"),
				renderFooterLink(parseIntl.T(n, "footer.link.demo"), "#demo"),
				renderFooterLink(parseIntl.T(n, "footer.link.pricing"), marketingPricingRoute),
				renderFooterLink(parseIntl.T(n, "footer.link.capabilities"), marketingCapabilitiesRoute),
			),
			renderFooterColumn(parseIntl.T(n, "footer.col.company"),
				renderFooterLink(parseIntl.T(n, "footer.link.about"), "#"),
				renderFooterLink(parseIntl.T(n, "footer.link.customers"), "#"),
				renderFooterLink(parseIntl.T(n, "footer.link.careers"), "#"),
				renderFooterLink(parseIntl.T(n, "footer.link.contact"), "#"),
			),
			renderFooterColumn(parseIntl.T(n, "footer.col.resources"),
				renderFooterLink(parseIntl.T(n, "footer.link.documentation"), "#"),
				renderFooterLink(parseIntl.T(n, "footer.link.security"), "#"),
				renderFooterLink(parseIntl.T(n, "footer.privacy"), "#"),
				renderFooterLink(parseIntl.T(n, "footer.terms"), "#"),
			),
		),
	)
}

// renderLandingHeader renders the top header bar with brand, nav, and CTA buttons.
// Kept for any direct call sites; delegates to renderMarketingHeader.
func renderLandingHeader(parseIntl i18n.Runtime, parseCurrentPath string) ui.Node {
	n := marketingI18nNamespace
	return renderMarketingHeader(
		parseIntl,
		parseCurrentPath,
		Tag("nav",
			Class("hidden items-center gap-6 lg:flex"),
			renderNavLink(parseCurrentPath, marketingHomeRoute, parseIntl.T(n, "nav.product")),
			renderNavLink(parseCurrentPath, marketingPricingRoute, parseIntl.T(n, "nav.pricing")),
		),
		renderMarketingHeaderAction(parseIntl.T(n, "header.logIn"), authLandingRoute, false, true),
		renderMarketingHeaderAction(parseIntl.T(n, "header.signUp"), marketingSignupRoute, false, false),
		renderMarketingHeaderAction(parseIntl.T(n, "header.openApp"), chatRouteRoot, true, false),
	)
}

// parseLandingPageForPath maps a URL path to the landing page variant constant.
func parseLandingPageForPath(parsePath string) string {
	switch strings.TrimSpace(parsePath) {
	case marketingCapabilitiesRoute:
		return landingPageCapabilities
	case marketingPricingRoute:
		return landingPagePricing
	case marketingSignupRoute:
		return landingPageSignup
	case marketingHomeRoute, authLandingRoute:
		return landingPageHome
	default:
		return landingPageHome
	}
}
