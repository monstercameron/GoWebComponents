//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderPricingShell renders the full standalone pricing page navigated to via the SW router.
func renderPricingShell(parseIntl i18n.Runtime, _ appViewState) ui.Node {
	n := marketingI18nNamespace
	return Div(
		Class("relative min-h-screen text-[#f0f0f8] antialiased page-bg"),
		renderPageBackground(),
		renderPricingHeader(parseIntl),
		Main(
			ID(pricingTopSectionID),
			Class("relative z-10"),
			Div(
				Class("mx-auto w-[min(1200px,calc(100%-24px))] pt-4 sm:w-[min(1200px,calc(100%-32px))] sm:pt-5 lg:w-[min(1200px,calc(100%-40px))]"),
				renderJourneyProgressBand(parseBuildMarketingJourneyStage(marketingPricingRoute)),
			),
			renderPricingHero(parseIntl),
			renderPricingPlans(parseIntl),
			renderPricingCompare(parseIntl),
			renderPricingFAQ(parseIntl),
			renderPricingContact(parseIntl),
		),
		renderMarketingFooter(
			parseIntl,
			renderFooterColumn(parseIntl.T(n, "footer.col.pricing"),
				renderFooterLink(parseIntl.T(n, "footer.link.starter"), "#plans"),
				renderFooterLink(parseIntl.T(n, "footer.link.team"), "#plans"),
				renderFooterLink(parseIntl.T(n, "footer.link.enterprise"), "#plans"),
				renderFooterLink(parseIntl.T(n, "nav.compare"), "#compare"),
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

// renderPricingHeader renders the pricing page top bar with brand, in-page nav, and app CTA.
func renderPricingHeader(parseIntl i18n.Runtime) ui.Node {
	n := marketingI18nNamespace
	return renderMarketingHeader(
		parseIntl,
		marketingPricingRoute,
		Tag("nav",
			Class("hidden items-center gap-6 lg:flex"),
			A(Class("text-sm text-[#8a8a9a] transition hover:text-[#f0f0f8]"), Href("#plans"), Text(parseIntl.T(n, "nav.plans"))),
			A(Class("text-sm text-[#8a8a9a] transition hover:text-[#f0f0f8]"), Href("#compare"), Text(parseIntl.T(n, "nav.compare"))),
			A(Class("text-sm text-[#8a8a9a] transition hover:text-[#f0f0f8]"), Href("#faq"), Text(parseIntl.T(n, "nav.faq"))),
		),
		renderLanguageSelector(parseIntl),
		renderMarketingHeaderAction(parseIntl.T(n, "header.logIn"), authLandingRoute, false, true),
		renderMarketingHeaderAction(parseIntl.T(n, "header.signUp"), marketingSignupRoute, false, false),
		renderMarketingHeaderAction(parseIntl.T(n, "header.openApp"), chatRouteRoot, true, false),
	)
}

// renderPricingHero renders the pricing hero: headline copy left, three stat cards right.
func renderPricingHero(parseIntl i18n.Runtime) ui.Node {
	n := marketingI18nNamespace
	return Section(
		Class("pb-14 pt-4 sm:pb-18 sm:pt-6 md:pb-20 md:pt-8 lg:pb-24 lg:pt-12"),
		Div(
			Class("mx-auto grid w-[min(1200px,calc(100%-24px))] gap-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-12 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[.95fr_1.05fr] lg:items-end lg:gap-14"),
			// left: headline + body + CTA row
			Div(
				Class("max-w-[640px] fade-up"),
				renderSectionEyebrow(parseIntl.T(n, "pricing.hero.eyebrow")),
				H1(Class("font-display mt-4 text-4xl font-bold leading-tight tracking-[-0.03em] text-[#f0f0f8] sm:text-5xl md:text-6xl"), Text(parseIntl.T(n, "pricing.hero.headline"))),
				P(Class("mt-5 max-w-[52ch] text-base leading-7 text-[#8a8a9a] sm:text-lg sm:leading-8"), Text(parseIntl.T(n, "pricing.hero.body"))),
				Div(
					Class("mt-8 flex flex-wrap gap-3"),
					renderCtaPrimary(parseIntl.T(n, "pricing.hero.primaryCta"), "#plans"),
					renderCtaSecondary(parseIntl.T(n, "pricing.hero.secondaryCta"), "#compare"),
				),
			),
			// right: three stat cards
			Div(
				Class("grid gap-4 sm:grid-cols-3 sm:gap-5 fade-up fade-up-d1"),
				renderPricingStatCard(parseIntl.T(n, "pricing.hero.stat1.title"), parseIntl.T(n, "pricing.hero.stat1.body")),
				renderPricingStatCard(parseIntl.T(n, "pricing.hero.stat2.title"), parseIntl.T(n, "pricing.hero.stat2.body")),
				renderPricingStatCard(parseIntl.T(n, "pricing.hero.stat3.title"), parseIntl.T(n, "pricing.hero.stat3.body")),
			),
		),
	)
}

// renderPricingStatCard renders a single stat cell in the pricing hero.
func renderPricingStatCard(renderTitle, renderBody string) ui.Node {
	return Div(
		Class("rounded-2xl border border-white/[0.06] bg-[#111118] px-5 py-6"),
		Div(Class("text-base font-semibold text-[#f0f0f8]"), Text(renderTitle)),
		P(Class("mt-2 text-sm leading-6 text-[#8a8a9a]"), Text(renderBody)),
	)
}

// renderPricingPlans renders the three plan cards: Starter, Team (featured), Enterprise.
func renderPricingPlans(parseIntl i18n.Runtime) ui.Node {
	n := marketingI18nNamespace
	type pricingPlan struct {
		prefix      string
		ctaRoute    string
		featureKeys []string
		featured    bool
		enterprise  bool
	}
	return Section(
		ID("plans"),
		Class("pb-16 sm:pb-20 md:pb-24"),
		Div(
			Class("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				Class("mb-10 max-w-[640px] scroll-reveal"),
				renderSectionEyebrow(parseIntl.T(n, "pricing.plans.eyebrow")),
				H2(Class("font-display mt-4 text-3xl font-bold tracking-[-0.04em] text-[#f0f0f8] sm:text-4xl"), Text(parseIntl.T(n, "pricing.plans.h2"))),
			),
			Div(
				Class("grid gap-4 sm:gap-5 lg:grid-cols-3 scroll-reveal scroll-reveal-d1"),
				Map([]pricingPlan{
					{"pricing.starter", marketingSignupRoute, []string{"0", "1", "2", "3", "4"}, false, false},
					{"pricing.team", marketingSignupRoute, []string{"0", "1", "2", "3", "4", "5"}, true, false},
					{"pricing.enterprise", "mailto:sales@relaydesk.com", []string{"0", "1", "2", "3", "4", "5"}, false, true},
				}, func(parseP pricingPlan) ui.Node {
					parseCardClass := "rounded-2xl border border-white/[0.06] bg-[#111118] px-5 py-7 sm:px-6 sm:py-8"
					if parseP.featured {
						parseCardClass = "pricing-card-featured rounded-2xl border bg-[#111118] px-5 py-7 sm:px-6 sm:py-8"
					} else if parseP.enterprise {
						parseCardClass = "pricing-card-enterprise rounded-2xl bg-[#111118] px-5 py-7 sm:px-6 sm:py-8"
					}
					parseBadgeText := parseIntl.T(n, parseP.prefix+".badge")
					var buildBadge ui.Node
					if parseBadgeText != "" {
						buildBadge = Div(Class("mb-4 inline-flex rounded-full bg-[#00d9ff]/10 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.14em] text-[#00d9ff]"), Text(parseBadgeText))
					}
					parseSuffixText := parseIntl.T(n, parseP.prefix+".suffix")
					var buildSuffix ui.Node
					if parseSuffixText != "" {
						buildSuffix = Span(Class("font-mono-tech text-lg text-[#8a8a9a]"), Text(parseSuffixText))
					}
					parsePriceClass := "font-mono-tech mt-4 text-4xl font-bold tracking-[-0.04em] text-[#f0f0f8] sm:text-5xl"
					if parseP.featured {
						parsePriceClass = "font-mono-tech mt-4 text-4xl font-bold tracking-[-0.04em] text-[#00d9ff] sm:text-5xl"
					}
					return Article(
						Class(parseCardClass),
						buildBadge,
						Div(Class("text-[11px] font-semibold uppercase tracking-[0.14em] text-[#8a8a9a]"), Text(parseIntl.T(n, parseP.prefix+".tone"))),
						Div(Class("mt-1 text-lg font-bold text-[#f0f0f8]"), Text(parseIntl.T(n, parseP.prefix+".name"))),
						Div(
							Class(parsePriceClass),
							Text(parseIntl.T(n, parseP.prefix+".price")),
							buildSuffix,
						),
						P(Class("mt-4 text-sm leading-6 text-[#8a8a9a]"), Text(parseIntl.T(n, parseP.prefix+".description"))),
						Ul(
							Class("mt-6 space-y-2.5 text-sm text-[#8a8a9a]"),
							Map(parseP.featureKeys, func(parseIdx string) ui.Node {
								return Li(
									Class("flex items-start gap-3"),
									Span(Class("mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#00d9ff]"), nil),
									Span(Class("text-[#f0f0f8]"), Text(parseIntl.T(n, parseP.prefix+".feature."+parseIdx))),
								)
							}),
						),
						Div(
							Class("mt-7"),
							renderCtaPrimary(parseIntl.T(n, parseP.prefix+".cta"), parseP.ctaRoute),
						),
					)
				}),
			),
		),
	)
}

// renderPricingCompare renders the feature comparison table (8 capability rows).
func renderPricingCompare(parseIntl i18n.Runtime) ui.Node {
	n := marketingI18nNamespace
	compareRowKeys := []string{"compare.workspace", "compare.collab", "compare.admin", "compare.billing", "compare.support", "compare.compliance"}

	buildGridArgs := make([]interface{}, 0, 4+len(compareRowKeys)*4+1)
	buildGridArgs = append(buildGridArgs, Class("min-w-[640px] overflow-hidden rounded-2xl border border-white/[0.06] grid grid-cols-4 gap-px bg-white/[0.03] text-sm"))
	for _, buildColKey := range []string{"compare.col.capability", "compare.col.starter", "compare.col.team", "compare.col.enterprise"} {
		buildGridArgs = append(buildGridArgs, Div(Class("bg-[#111118] px-4 py-3.5 text-[11px] font-semibold uppercase tracking-[0.12em] text-[#8a8a9a]"), Text(parseIntl.T(n, buildColKey))))
	}
	for i, buildRowKey := range compareRowKeys {
		parseBg := "bg-[#0d0d14]"
		if i%2 == 0 {
			parseBg = "bg-[#111118]"
		}
		buildGridArgs = append(buildGridArgs,
			Div(Class(parseBg+" px-4 py-3.5 text-sm font-medium text-[#f0f0f8]"), Text(parseIntl.T(n, buildRowKey+".label"))),
			Div(Class(parseBg+" px-4 py-3.5 text-sm text-[#8a8a9a]"), Text(parseIntl.T(n, buildRowKey+".starter"))),
			Div(Class(parseBg+" px-4 py-3.5 text-sm text-[#8a8a9a]"), Text(parseIntl.T(n, buildRowKey+".team"))),
			Div(Class(parseBg+" px-4 py-3.5 text-sm text-[#8a8a9a]"), Text(parseIntl.T(n, buildRowKey+".enterprise"))),
		)
	}

	return Section(
		ID("compare"),
		Class("pb-16 sm:pb-20 md:pb-24"),
		Div(
			Class("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				Class("rounded-2xl border border-white/[0.06] bg-[#111118] px-5 py-7 sm:px-8 sm:py-8 md:px-10 md:py-10 scroll-reveal"),
				renderSectionEyebrow(parseIntl.T(n, "compare.eyebrow")),
				H2(Class("font-display mt-4 text-2xl font-bold tracking-[-0.04em] text-[#f0f0f8] sm:text-3xl"), Text(parseIntl.T(n, "compare.h2"))),
				Div(
					Class("mt-8 overflow-x-auto"),
					Div(buildGridArgs...),
				),
			),
		),
	)
}

// renderPricingFAQ renders five objection-handler FAQ cards in a two-column layout.
func renderPricingFAQ(parseIntl i18n.Runtime) ui.Node {
	n := marketingI18nNamespace
	faqKeys := []string{"faq.q1", "faq.q2", "faq.q3", "faq.q4", "faq.q5"}
	return Section(
		ID("faq"),
		Class("pb-16 sm:pb-20 md:pb-24"),
		Div(
			Class("mx-auto grid w-[min(1200px,calc(100%-24px))] gap-8 sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[.78fr_1.22fr] lg:gap-12"),
			Div(
				Class("scroll-reveal"),
				renderSectionEyebrow(parseIntl.T(n, "faq.eyebrow")),
				H2(Class("font-display mt-4 text-3xl font-bold tracking-[-0.04em] text-[#f0f0f8] sm:text-4xl"), Text(parseIntl.T(n, "faq.h2"))),
				P(Class("mt-4 text-sm leading-7 text-[#8a8a9a]"), Text(parseIntl.T(n, "faq.body"))),
			),
			Div(
				Class("grid gap-4 scroll-reveal scroll-reveal-d1"),
				Map(faqKeys, func(parseK string) ui.Node {
					return Div(
						Class("rounded-2xl border border-white/[0.06] bg-[#111118] px-5 py-6 sm:px-6"),
						Div(Class("text-base font-semibold text-[#f0f0f8]"), Text(parseIntl.T(n, parseK+".q"))),
						P(Class("mt-3 text-sm leading-6 text-[#8a8a9a]"), Text(parseIntl.T(n, parseK+".a"))),
					)
				}),
			),
		),
	)
}

// renderPricingContact renders the sales-contact CTA at the bottom of the pricing page.
func renderPricingContact(parseIntl i18n.Runtime) ui.Node {
	n := marketingI18nNamespace
	return Section(
		ID("contact"),
		Class("pb-20 sm:pb-24 md:pb-28"),
		Div(
			Class("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				Class("rounded-2xl border border-[#00d9ff]/20 bg-[#111118] px-6 py-8 sm:px-8 sm:py-10 md:px-10 md:py-12 scroll-reveal"),
				Div(
					Class("max-w-[640px]"),
					renderSectionEyebrow(parseIntl.T(n, "contact.eyebrow")),
					H2(Class("font-display mt-4 text-3xl font-bold tracking-[-0.04em] text-[#f0f0f8] sm:text-4xl"), Text(parseIntl.T(n, "contact.h2"))),
					P(Class("mt-4 max-w-[52ch] text-base leading-7 text-[#8a8a9a] sm:leading-8"), Text(parseIntl.T(n, "contact.body"))),
					Div(
						Class("mt-8 flex flex-col gap-3 sm:flex-row"),
						A(
							Class("inline-flex items-center justify-center rounded-full bg-[#00d9ff] px-6 py-3.5 text-sm font-semibold text-[#050508] transition hover:-translate-y-[1px]"),
							Href("mailto:sales@relaydesk.com"),
							Text(parseIntl.T(n, "contact.emailSales")),
						),
						A(
							Class("inline-flex items-center justify-center rounded-full border border-white/[0.12] bg-transparent px-6 py-3.5 text-sm font-semibold text-[#f0f0f8] transition hover:bg-white/[0.06]"),
							Href("mailto:hello@relaydesk.com"),
							Text(parseIntl.T(n, "contact.emailTeam")),
						),
					),
				),
			),
		),
	)
}
