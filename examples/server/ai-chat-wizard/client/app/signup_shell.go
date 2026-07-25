//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/i18n"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// renderSignupShell renders the dedicated /signup marketing page.
// It mounts the full account-creation form inside a two-column marketing layout so
// the page feels like a proper destination rather than a toggled variant of the login page.
func renderSignupShell(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController) ui.Node {
	setAuthDocumentTitle(parseIntl, authModeSignup)
	n := marketingI18nNamespace
	return Div(
		ClassStr("relative min-h-screen text-[#f0f0f8] antialiased page-bg"),
		renderPageBackground(),
		// marketing header — Log In link only; no "Open App" primary so the signup form is the CTA
		renderMarketingHeader(
			parseIntl,
			marketingSignupRoute,
			Tag("nav",
				ClassStr("hidden items-center gap-6 lg:flex"),
				renderNavLink(marketingSignupRoute, marketingHomeRoute, parseIntl.T(n, "nav.product")),
				renderNavLink(marketingSignupRoute, marketingPricingRoute, parseIntl.T(n, "nav.pricing")),
			),
			renderLanguageSelector(parseIntl),
			renderMarketingHeaderAction(parseIntl.T(n, "header.logIn"), authLoginRoute, false, false),
		),
		Main(
			ClassStr("relative z-10"),
			Div(
				ClassStr("mx-auto w-[min(1200px,calc(100%-24px))] pt-4 sm:w-[min(1200px,calc(100%-32px))] sm:pt-5 lg:w-[min(1200px,calc(100%-40px))]"),
				renderJourneyProgressBand(parseBuildMarketingJourneyStage(marketingSignupRoute)),
			),
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
		ClassStr("pb-16 pt-4 sm:pb-20 sm:pt-6 md:pb-24 md:pt-8 lg:pb-28 lg:pt-12"),
		Div(
			ClassStr("mx-auto grid w-[min(1200px,calc(100%-24px))] items-center gap-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-12 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[.95fr_1.05fr] lg:gap-14"),
			// left — hero copy and stat cards
			Div(
				ClassStr("max-w-[640px] fade-up"),
				// brand logo above headline
				Img(
					Src(brandLogoURL),
					Attr("alt", appBrandName),
					ClassStr("mb-6 h-12 w-auto sm:h-14"),
				),
				Div(
					ClassStr("mb-5 inline-flex rounded-full bg-white/10 px-3 py-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:mb-7 sm:px-4 sm:text-[11px] sm:tracking-[0.18em]"),
					Text(parseIntl.T(c, "auth.signupBadge")),
				),
				H1(
					ClassStr("hero-gradient-text font-display max-w-none text-4xl font-bold leading-[1.02] tracking-[-0.045em] sm:text-5xl md:max-w-[11ch] md:text-6xl xl:text-7xl"),
					Text(parseIntl.T(c, "auth.signupHeroTitle")),
				),
				P(
					ClassStr("mt-5 max-w-[56ch] text-base leading-7 text-[#9b9bb1] sm:mt-6 sm:text-lg sm:leading-8"),
					Text(parseIntl.T(c, "auth.signupHeroBody")),
				),
				// stat / proof cards
				Div(
					ClassStr("mt-8 grid gap-4 sm:grid-cols-3 sm:gap-5 scroll-reveal"),
					Map(parseStatCards, func(parseCard []string) ui.Node {
						return Div(
							ClassStr("rounded-2xl border border-white/[0.06] bg-[#13131e] px-5 py-6"),
							Div(ClassStr("text-lg font-semibold tracking-[-0.04em] text-[#f0f0f8] sm:text-xl"), Text(parseCard[0])),
							P(ClassStr("mt-2 text-sm leading-6 text-[#9b9bb1]"), Text(parseCard[1])),
						)
					}),
				),
				// metric strip
				Div(
					ClassStr("mt-10 flex flex-wrap items-center gap-6 border-t border-white/[0.06] pt-8 sm:mt-12 sm:gap-8 sm:pt-10"),
					renderLandingHeroMetric(parseIntl.T(marketingI18nNamespace, "metric.responseTime.value"), parseIntl.T(marketingI18nNamespace, "metric.responseTime.label")),
					renderLandingHeroMetric(parseIntl.T(marketingI18nNamespace, "metric.models.value"), parseIntl.T(marketingI18nNamespace, "metric.models.label")),
					Div(
						ClassStr("flex items-center gap-2"),
						renderStatusDot(),
						Div(
							ClassStr("font-mono-tech text-xs text-[#9b9bb1]"),
							Span(ClassStr("font-semibold text-[#4ade80]"), Text(parseIntl.T(marketingI18nNamespace, "metric.uptime.value"))),
							Text(" "+parseIntl.T(marketingI18nNamespace, "metric.uptime.label")),
						),
					),
				),
				// what happens after you sign up
				renderSignupAfterSignupExplainer(),
			),
			// right — signup form card
			Div(
				ClassStr("mx-auto w-full max-w-[520px] fade-up fade-up-d1"),
				renderAuthFormCard(parseIntl, parseView, parseAuth, true),
			),
		),
	)
}

// renderSignupAfterSignupExplainer renders a compact 4-step sequence that sets expectations
// for what happens immediately after an account is created.
func renderSignupAfterSignupExplainer() ui.Node {
	type parseStep struct {
		parseNum   string
		parseTitle string
		parseBody  string
	}
	parseSteps := []parseStep{
		{"01", "Create workspace", "Name your workspace. One workspace per account on the Starter plan."},
		{"02", "Confirm account", "Check your inbox for a confirmation link. Required before the first AI turn."},
		{"03", "Land in chat", "Your workspace opens directly in the chat view. No dashboard tour. Start working."},
		{"04", "Invite team later", "Seat invites are in Settings. You can start solo and add collaborators any time."},
	}
	return Div(
		ClassStr("mt-10 border-t border-white/[0.06] pt-8 sm:mt-12 sm:pt-10"),
		Div(ClassStr("mb-4 text-[10px] uppercase tracking-[0.18em] text-white/30"), Text("What happens next")),
		Div(
			ClassStr("grid grid-cols-2 gap-3 sm:gap-4"),
			Map(parseSteps, func(parseS parseStep) ui.Node {
				return Div(
					ClassStr("rounded-xl border border-white/[0.06] bg-white/[0.025] px-4 py-3"),
					Div(ClassStr("font-mono-tech mb-1 text-[10px] text-[#8e7bff]/60"), Text(parseS.parseNum)),
					Div(ClassStr("text-xs font-semibold text-[#f0f0f8]"), Text(parseS.parseTitle)),
					P(ClassStr("mt-1 text-[11px] leading-4 text-[#9b9bb1]"), Text(parseS.parseBody)),
				)
			}),
		),
	)
}
