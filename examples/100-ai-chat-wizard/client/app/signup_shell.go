//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderSignupShell renders the dedicated /signup marketing page.
// It mounts the full account-creation form inside a two-column marketing layout so
// the page feels like a proper destination rather than a toggled variant of the login page.
func renderSignupShell(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController) ui.Node {
	setAuthDocumentTitle(parseIntl, authModeSignup)
	n := marketingI18nNamespace
	return Div(
		Class("relative min-h-screen text-[#f0f0f8] antialiased page-bg"),
		renderPageBackground(),
		// marketing header — Log In link only; no "Open App" primary so the signup form is the CTA
		renderMarketingHeader(
			parseIntl,
			marketingSignupRoute,
			Tag("nav",
				Class("hidden items-center gap-6 lg:flex"),
				renderNavLink(marketingSignupRoute, marketingHomeRoute, parseIntl.T(n, "nav.product")),
				renderNavLink(marketingSignupRoute, marketingPricingRoute, parseIntl.T(n, "nav.pricing")),
			),
			renderMarketingHeaderAction(parseIntl.T(n, "header.logIn"), authLandingRoute, false, true),
		),
		Main(
			Class("relative z-10"),
			renderSignupBody(parseIntl, parseView, parseAuth),
		),
		renderMarketingFooter(parseIntl, renderStandardFooterColumns(parseIntl)...),
	)
}

// renderSignupBody renders the two-column hero + form section for the /signup page.
// Left column: brand logo, headline, body copy, and stat cards with social proof.
// Right column: the account creation form card.
func renderSignupBody(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController) ui.Node {
	c := chatI18nNamespace
	parseStatCards := [][]string{
		{parseIntl.T(c, "auth.signupStat1Title"), parseIntl.T(c, "auth.signupStat1Body")},
		{parseIntl.T(c, "auth.signupStat2Title"), parseIntl.T(c, "auth.signupStat2Body")},
		{parseIntl.T(c, "auth.signupStat3Title"), parseIntl.T(c, "auth.signupStat3Body")},
	}
	return Section(
		Class("pb-16 pt-4 sm:pb-20 sm:pt-6 md:pb-24 md:pt-8 lg:pb-28 lg:pt-12"),
		Div(
			Class("mx-auto grid w-[min(1200px,calc(100%-24px))] items-center gap-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-12 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[.95fr_1.05fr] lg:gap-14"),
			// left — hero copy and stat cards
			Div(
				Class("max-w-[640px] fade-up"),
				// brand logo above headline
				Img(
					Src(brandLogoURL),
					Attr("alt", appBrandName),
					Class("mb-6 h-12 w-auto sm:h-14"),
				),
				Div(
					Class("mb-5 inline-flex rounded-full bg-white/10 px-3 py-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:mb-7 sm:px-4 sm:text-[11px] sm:tracking-[0.18em]"),
					Text(parseIntl.T(c, "auth.signupBadge")),
				),
				H1(
					Class("hero-gradient-text font-display max-w-none text-4xl font-bold leading-[1.02] tracking-[-0.045em] sm:text-5xl md:max-w-[11ch] md:text-6xl xl:text-7xl"),
					Text(parseIntl.T(c, "auth.signupHeroTitle")),
				),
				P(
					Class("mt-5 max-w-[56ch] text-base leading-7 text-[#8a8a9a] sm:mt-6 sm:text-lg sm:leading-8"),
					Text(parseIntl.T(c, "auth.signupHeroBody")),
				),
				// stat / proof cards
				Div(
					Class("mt-8 grid gap-4 sm:grid-cols-3 sm:gap-5 scroll-reveal"),
					Map(parseStatCards, func(parseCard []string) ui.Node {
						return Div(
							Class("rounded-2xl border border-white/[0.06] bg-[#111118] px-5 py-6"),
							Div(Class("text-lg font-semibold tracking-[-0.04em] text-[#f0f0f8] sm:text-xl"), Text(parseCard[0])),
							P(Class("mt-2 text-sm leading-6 text-[#8a8a9a]"), Text(parseCard[1])),
						)
					}),
				),
				// metric strip
				Div(
					Class("mt-10 flex flex-wrap items-center gap-6 border-t border-white/[0.06] pt-8 sm:mt-12 sm:gap-8 sm:pt-10"),
					renderLandingHeroMetric(parseIntl.T(marketingI18nNamespace, "metric.responseTime.value"), parseIntl.T(marketingI18nNamespace, "metric.responseTime.label")),
					renderLandingHeroMetric(parseIntl.T(marketingI18nNamespace, "metric.models.value"), parseIntl.T(marketingI18nNamespace, "metric.models.label")),
					Div(
						Class("flex items-center gap-2"),
						renderStatusDot(),
						Div(
							Class("font-mono-tech text-xs text-[#8a8a9a]"),
							Span(Class("font-semibold text-[#4ade80]"), Text(parseIntl.T(marketingI18nNamespace, "metric.uptime.value"))),
							Text(" "+parseIntl.T(marketingI18nNamespace, "metric.uptime.label")),
						),
					),
				),
			),
			// right — signup form card
			Div(
				Class("mx-auto w-full max-w-[520px] fade-up fade-up-d1"),
				renderAuthFormCard(parseIntl, parseView, parseAuth, true),
			),
		),
	)
}
