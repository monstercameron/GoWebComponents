//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

// isInfoLandingPage reports whether parsePage is one of the six static info pages.
func isInfoLandingPage(parsePage string) bool {
	switch parsePage {
	case landingPageAbout, landingPageContact, landingPagePrivacy,
		landingPageTerms, landingPageSecurity, landingPageStatus:
		return true
	}
	return false
}

// renderInfoShell renders the shared chrome (header + footer) around whichever
// info page body renderPage selects.
func renderInfoShell(parseIntl i18n.Runtime, parsePage string) ui.Node {
	n := marketingI18nNamespace
	return Div(
		Class("relative min-h-screen text-[#f0f0f8] antialiased page-bg"),
		renderPageBackground(),
		renderMarketingHeader(
			parseIntl,
			"",
			Tag("nav",
				Class("hidden items-center gap-6 lg:flex"),
				renderNavLink("", marketingHomeRoute, parseIntl.T(n, "nav.product")),
				renderNavLink("", marketingPricingRoute, parseIntl.T(n, "nav.pricing")),
			),
			renderLanguageSelector(parseIntl),
			renderMarketingHeaderAction(parseIntl.T(n, "header.logIn"), authLandingRoute, false, true),
			renderMarketingHeaderAction(parseIntl.T(n, "header.signUp"), marketingSignupRoute, false, false),
			renderMarketingHeaderAction(parseIntl.T(n, "header.openApp"), chatRouteRoot, true, false),
		),
		Main(
			Class("relative z-10"),
			renderInfoBody(parseIntl, parsePage),
		),
		renderMarketingFooter(
			parseIntl,
			renderFooterColumn(parseIntl.T(n, "footer.col.product"),
				renderFooterLink(parseIntl.T(n, "footer.link.home"), marketingHomeRoute),
				renderFooterLink(parseIntl.T(n, "footer.link.pricing"), marketingPricingRoute),
				renderFooterLink(parseIntl.T(n, "footer.link.signup"), marketingSignupRoute),
			),
			renderFooterColumn(parseIntl.T(n, "footer.col.company"),
				renderFooterLink(parseIntl.T(n, "footer.link.about"), marketingAboutRoute),
				renderFooterLink(parseIntl.T(n, "footer.link.contact"), marketingContactRoute),
			),
			renderFooterColumn(parseIntl.T(n, "footer.col.legal"),
				renderFooterLink(parseIntl.T(n, "footer.privacy"), marketingPrivacyRoute),
				renderFooterLink(parseIntl.T(n, "footer.terms"), marketingTermsRoute),
				renderFooterLink(parseIntl.T(n, "footer.link.security"), marketingSecurityRoute),
				renderFooterLink(parseIntl.T(n, "footer.status"), marketingStatusRoute),
			),
		),
	)
}

// renderInfoBody dispatches to the correct info page body based on parsePage.
func renderInfoBody(parseIntl i18n.Runtime, parsePage string) ui.Node {
	switch parsePage {
	case landingPageAbout:
		return renderInfoAbout(parseIntl)
	case landingPageContact:
		return renderInfoContact(parseIntl)
	case landingPagePrivacy:
		return renderInfoPrivacy(parseIntl)
	case landingPageTerms:
		return renderInfoTerms(parseIntl)
	case landingPageSecurity:
		return renderInfoSecurity(parseIntl)
	case landingPageStatus:
		return renderInfoStatus(parseIntl)
	}
	return renderInfoAbout(parseIntl)
}

// renderInfoAbout renders the /about page body.
func renderInfoAbout(parseIntl i18n.Runtime) ui.Node {
	n := marketingI18nNamespace
	return Div(
		Class("mx-auto w-[min(760px,calc(100%-24px))] py-20 sm:py-28"),
		renderSectionEyebrow(parseIntl.T(n, "about.eyebrow")),
		H1(
			Class("mt-6 font-display text-4xl font-bold leading-tight tracking-tight text-[#f0f0f8] sm:text-5xl"),
			Text(parseIntl.T(n, "about.headline")),
		),
		P(
			Class("mt-6 text-base leading-7 text-[#8a8a9a] sm:text-lg sm:leading-8"),
			Text(parseIntl.T(n, "about.body")),
		),
		Div(
			Class("mt-10 flex flex-col gap-3 sm:flex-row"),
			renderCtaPrimary(parseIntl.T(n, "about.cta.pricing"), marketingPricingRoute),
			renderCtaSecondary(parseIntl.T(n, "about.cta.start"), marketingSignupRoute),
		),
	)
}

// renderInfoContact renders the /contact page body.
func renderInfoContact(parseIntl i18n.Runtime) ui.Node {
	n := marketingI18nNamespace
	return Div(
		Class("mx-auto w-[min(900px,calc(100%-24px))] py-20 sm:py-28"),
		renderSectionEyebrow(parseIntl.T(n, "info.contact.eyebrow")),
		H1(
			Class("mt-6 font-display text-4xl font-bold leading-tight tracking-tight text-[#f0f0f8] sm:text-5xl"),
			Text(parseIntl.T(n, "info.contact.headline")),
		),
		P(
			Class("mt-6 text-base leading-7 text-[#8a8a9a] sm:text-lg sm:leading-8"),
			Text(parseIntl.T(n, "info.contact.body")),
		),
		Div(
			Class("mt-12 grid grid-cols-1 gap-8 sm:grid-cols-2"),
			renderInfoContactCard(
				parseIntl.T(n, "info.contact.sales.title"),
				parseIntl.T(n, "info.contact.sales.body"),
				parseIntl.T(n, "info.contact.sales.cta"),
			),
			renderInfoContactCard(
				parseIntl.T(n, "info.contact.support.title"),
				parseIntl.T(n, "info.contact.support.body"),
				parseIntl.T(n, "info.contact.support.cta"),
			),
			renderInfoContactCard(
				parseIntl.T(n, "info.contact.general.title"),
				parseIntl.T(n, "info.contact.general.body"),
				parseIntl.T(n, "info.contact.general.cta"),
			),
			renderInfoContactCard(
				parseIntl.T(n, "info.contact.response.title"),
				parseIntl.T(n, "info.contact.response.body"),
				"",
			),
		),
	)
}

// renderInfoContactCard renders a single contact category card.
func renderInfoContactCard(renderTitle, renderBody, renderEmail string) ui.Node {
	emailNode := ui.Node(nil)
	if renderEmail != "" {
		emailNode = P(
			Class("mt-3 text-sm font-medium text-[#7c6fcd]"),
			Text(renderEmail),
		)
	}
	return Div(
		Class("rounded-2xl border border-[#2a2a3d] bg-[#13131f] p-6"),
		H3(
			Class("text-base font-semibold text-[#f0f0f8]"),
			Text(renderTitle),
		),
		P(
			Class("mt-2 text-sm leading-6 text-[#8a8a9a]"),
			Text(renderBody),
		),
		emailNode,
	)
}

// renderInfoPrivacy renders the /privacy page body.
func renderInfoPrivacy(parseIntl i18n.Runtime) ui.Node {
	n := marketingI18nNamespace
	return Div(
		Class("mx-auto w-[min(760px,calc(100%-24px))] py-20 sm:py-28"),
		renderSectionEyebrow(parseIntl.T(n, "info.privacy.eyebrow")),
		H1(
			Class("mt-6 font-display text-4xl font-bold leading-tight tracking-tight text-[#f0f0f8] sm:text-5xl"),
			Text(parseIntl.T(n, "info.privacy.headline")),
		),
		P(
			Class("mt-6 text-base leading-7 text-[#8a8a9a] sm:text-lg sm:leading-8"),
			Text(parseIntl.T(n, "info.privacy.body")),
		),
		Div(
			Class("mt-12 space-y-8"),
			renderInfoSection(parseIntl.T(n, "info.privacy.collect.title"), parseIntl.T(n, "info.privacy.collect.body")),
			renderInfoSection(parseIntl.T(n, "info.privacy.use.title"), parseIntl.T(n, "info.privacy.use.body")),
			renderInfoSection(parseIntl.T(n, "info.privacy.retention.title"), parseIntl.T(n, "info.privacy.retention.body")),
			renderInfoSection(parseIntl.T(n, "info.privacy.contact.title"), parseIntl.T(n, "info.privacy.contact.body")),
		),
	)
}

// renderInfoTerms renders the /terms page body.
func renderInfoTerms(parseIntl i18n.Runtime) ui.Node {
	n := marketingI18nNamespace
	return Div(
		Class("mx-auto w-[min(760px,calc(100%-24px))] py-20 sm:py-28"),
		renderSectionEyebrow(parseIntl.T(n, "info.terms.eyebrow")),
		H1(
			Class("mt-6 font-display text-4xl font-bold leading-tight tracking-tight text-[#f0f0f8] sm:text-5xl"),
			Text(parseIntl.T(n, "info.terms.headline")),
		),
		P(
			Class("mt-6 text-base leading-7 text-[#8a8a9a] sm:text-lg sm:leading-8"),
			Text(parseIntl.T(n, "info.terms.body")),
		),
		Div(
			Class("mt-12 space-y-8"),
			renderInfoSection(parseIntl.T(n, "info.terms.account.title"), parseIntl.T(n, "info.terms.account.body")),
			renderInfoSection(parseIntl.T(n, "info.terms.use.title"), parseIntl.T(n, "info.terms.use.body")),
			renderInfoSection(parseIntl.T(n, "info.terms.billing.title"), parseIntl.T(n, "info.terms.billing.body")),
			renderInfoSection(parseIntl.T(n, "info.terms.availability.title"), parseIntl.T(n, "info.terms.availability.body")),
			renderInfoSection(parseIntl.T(n, "info.terms.contact.title"), parseIntl.T(n, "info.terms.contact.body")),
		),
	)
}

// renderInfoSecurity renders the /security page body.
func renderInfoSecurity(parseIntl i18n.Runtime) ui.Node {
	n := marketingI18nNamespace
	return Div(
		Class("mx-auto w-[min(760px,calc(100%-24px))] py-20 sm:py-28"),
		renderSectionEyebrow(parseIntl.T(n, "info.security.eyebrow")),
		H1(
			Class("mt-6 font-display text-4xl font-bold leading-tight tracking-tight text-[#f0f0f8] sm:text-5xl"),
			Text(parseIntl.T(n, "info.security.headline")),
		),
		P(
			Class("mt-6 text-base leading-7 text-[#8a8a9a] sm:text-lg sm:leading-8"),
			Text(parseIntl.T(n, "info.security.body")),
		),
		Div(
			Class("mt-12 space-y-8"),
			renderInfoSection(parseIntl.T(n, "info.security.access.title"), parseIntl.T(n, "info.security.access.body")),
			renderInfoSection(parseIntl.T(n, "info.security.data.title"), parseIntl.T(n, "info.security.data.body")),
			renderInfoSection(parseIntl.T(n, "info.security.retention.title"), parseIntl.T(n, "info.security.retention.body")),
			renderInfoSection(parseIntl.T(n, "info.security.ops.title"), parseIntl.T(n, "info.security.ops.body")),
			renderInfoSection(parseIntl.T(n, "info.security.contact.title"), parseIntl.T(n, "info.security.contact.body")),
		),
	)
}

// renderInfoStatus renders the /status page body.
func renderInfoStatus(parseIntl i18n.Runtime) ui.Node {
	n := marketingI18nNamespace
	return Div(
		Class("mx-auto w-[min(760px,calc(100%-24px))] py-20 sm:py-28"),
		renderSectionEyebrow(parseIntl.T(n, "info.status.eyebrow")),
		H1(
			Class("mt-6 font-display text-4xl font-bold leading-tight tracking-tight text-[#f0f0f8] sm:text-5xl"),
			Text(parseIntl.T(n, "info.status.headline")),
		),
		P(
			Class("mt-6 text-base leading-7 text-[#8a8a9a] sm:text-lg sm:leading-8"),
			Text(parseIntl.T(n, "info.status.body")),
		),
		Div(
			Class("mt-12 space-y-8"),
			renderInfoStatusSection(
				parseIntl.T(n, "info.status.current.title"),
				parseIntl.T(n, "info.status.current.value"),
				true,
			),
			renderInfoStatusSection(
				parseIntl.T(n, "info.status.incidents.title"),
				parseIntl.T(n, "info.status.incidents.none"),
				false,
			),
			renderInfoStatusSection(
				parseIntl.T(n, "info.status.history.title"),
				parseIntl.T(n, "info.status.history.none"),
				false,
			),
			renderInfoStatusSection(
				parseIntl.T(n, "info.status.maintenance.title"),
				parseIntl.T(n, "info.status.maintenance.none"),
				false,
			),
		),
	)
}

// renderInfoSection renders a titled text block used by privacy, terms, and security pages.
func renderInfoSection(renderTitle, renderBody string) ui.Node {
	return Div(
		Class("border-l-2 border-[#2a2a3d] pl-5"),
		H3(
			Class("text-base font-semibold text-[#f0f0f8]"),
			Text(renderTitle),
		),
		P(
			Class("mt-2 text-sm leading-6 text-[#8a8a9a]"),
			Text(renderBody),
		),
	)
}

// renderInfoStatusSection renders a titled status row, optionally with an operational indicator.
func renderInfoStatusSection(renderTitle, renderValue string, isOperational bool) ui.Node {
	indicatorClass := "inline-block h-2 w-2 rounded-full bg-[#4a4a6a]"
	valueClass := "text-sm leading-6 text-[#8a8a9a]"
	if isOperational {
		indicatorClass = "inline-block h-2 w-2 rounded-full bg-emerald-400"
		valueClass = "text-sm font-medium leading-6 text-emerald-400"
	}
	return Div(
		Class("flex items-start justify-between gap-4 rounded-xl border border-[#2a2a3d] bg-[#13131f] px-5 py-4"),
		H3(
			Class("text-sm font-semibold text-[#f0f0f8]"),
			Text(renderTitle),
		),
		Div(
			Class("flex items-center gap-2"),
			Span(Class(indicatorClass)),
			Span(Class(valueClass), Text(renderValue)),
		),
	)
}
