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
const landingPageAbout = "about"
const landingPageContact = "contact"
const landingPagePrivacy = "privacy"
const landingPageTerms = "terms"
const landingPageSecurity = "security"
const landingPageStatus = "status"

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
	case landingPageAbout:
		parseTitle = parseIntl.T(n, "page.title.about")
	case landingPageContact:
		parseTitle = parseIntl.T(n, "page.title.contact")
	case landingPagePrivacy:
		parseTitle = parseIntl.T(n, "page.title.privacy")
	case landingPageTerms:
		parseTitle = parseIntl.T(n, "page.title.terms")
	case landingPageSecurity:
		parseTitle = parseIntl.T(n, "page.title.security")
	case landingPageStatus:
		parseTitle = parseIntl.T(n, "page.title.status")
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
	if isInfoLandingPage(parsePage) {
		return renderInfoShell(parseIntl, parsePage)
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
			renderLanguageSelector(parseIntl),
			renderMarketingHeaderAction(parseIntl.T(n, "header.logIn"), authLandingRoute, false, true),
			renderMarketingHeaderAction(parseIntl.T(n, "header.signUp"), marketingSignupRoute, false, false),
			renderMarketingHeaderAction(parseIntl.T(n, "header.openApp"), chatRouteRoot, true, false),
		),
		Main(
			Class("relative z-10"),
			Div(
				Class("mx-auto w-[min(1200px,calc(100%-24px))] pt-4 sm:w-[min(1200px,calc(100%-32px))] sm:pt-5 lg:w-[min(1200px,calc(100%-40px))]"),
				renderJourneyProgressBand(parseBuildMarketingJourneyStage(parseView.CurrentPath)),
			),
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
				renderFooterLink(parseIntl.T(n, "footer.link.home"), marketingHomeRoute),
				renderFooterLink(parseIntl.T(n, "footer.link.pricing"), marketingPricingRoute),
				renderFooterLink(parseIntl.T(n, "footer.link.signup"), marketingSignupRoute),
			),
			renderFooterColumn(parseIntl.T(n, "footer.col.company"),
				renderFooterLink(parseIntl.T(n, "footer.link.about"), "/about"),
				renderFooterLink(parseIntl.T(n, "footer.link.contact"), "/contact"),
			),
			renderFooterColumn(parseIntl.T(n, "footer.col.legal"),
				renderFooterLink(parseIntl.T(n, "footer.privacy"), "/privacy"),
				renderFooterLink(parseIntl.T(n, "footer.terms"), "/terms"),
				renderFooterLink(parseIntl.T(n, "footer.link.security"), "/security"),
				renderFooterLink(parseIntl.T(n, "footer.status"), "/status"),
			),
		),
	)
}

// parseLandingPageForPath maps a URL path to the landing page variant constant.
func parseLandingPageForPath(parsePath string) string {
	switch strings.TrimSpace(parsePath) {
	case marketingCapabilitiesRoute:
		return landingPageCapabilities
	case marketingPricingRoute, marketingPlansRoute:
		return landingPagePricing
	case marketingSignupRoute:
		return landingPageSignup
	case marketingHomeRoute, authLandingRoute:
		return landingPageHome
	case marketingAboutRoute:
		return landingPageAbout
	case marketingContactRoute:
		return landingPageContact
	case marketingPrivacyRoute:
		return landingPagePrivacy
	case marketingTermsRoute:
		return landingPageTerms
	case marketingSecurityRoute:
		return landingPageSecurity
	case marketingStatusRoute:
		return landingPageStatus
	default:
		return landingPageHome
	}
}
