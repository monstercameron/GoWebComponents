//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/i18n"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// renderPricingShell renders the full standalone pricing page navigated to via the SW router.
func renderPricingShell(parseIntl i18n.Runtime, _ appViewState) ui.Node {
	n := marketingI18nNamespace
	return Div(
		ClassStr("relative min-h-screen text-[#f0f0f8] antialiased page-bg"),
		renderPageBackground(),
		renderPricingHeader(parseIntl),
		Main(
			ID(pricingTopSectionID),
			ClassStr("relative z-10"),
			Div(
				ClassStr("mx-auto w-[min(1200px,calc(100%-24px))] pt-4 sm:w-[min(1200px,calc(100%-32px))] sm:pt-5 lg:w-[min(1200px,calc(100%-40px))]"),
				renderJourneyProgressBand(parseBuildMarketingJourneyStage(marketingPricingRoute)),
			),
			renderPricingHero(parseIntl),
			renderPricingPlans(parseIntl),
			renderPricingCompare(parseIntl),
			renderPricingFAQ(parseIntl),
			renderPricingTrustBand(parseIntl),
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
			ClassStr("hidden items-center gap-6 lg:flex"),
			renderNavLink(marketingPricingRoute, marketingHomeRoute, parseIntl.T(n, "nav.product")),
			renderNavLink(marketingPricingRoute, marketingPricingRoute, parseIntl.T(n, "nav.pricing")),
		),
		renderLanguageSelector(parseIntl),
		renderMarketingHeaderAction(parseIntl.T(n, "header.logIn"), authLoginRoute, false, true),
		renderMarketingHeaderAction(parseIntl.T(n, "header.signUp"), marketingSignupRoute, false, false),
		renderMarketingHeaderAction(parseIntl.T(n, "header.openApp"), chatRouteRoot, true, false),
	)
}

// renderPricingHero renders the pricing hero: headline copy left, three stat cards right.
func renderPricingHero(parseIntl i18n.Runtime) ui.Node {
	n := marketingI18nNamespace
	return Section(
		ClassStr("pb-14 pt-4 sm:pb-18 sm:pt-6 md:pb-20 md:pt-8 lg:pb-24 lg:pt-12"),
		Div(
			ClassStr("mx-auto grid w-[min(1200px,calc(100%-24px))] gap-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-12 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[.95fr_1.05fr] lg:items-end lg:gap-14"),
			// left: headline + body + CTA row
			Div(
				ClassStr("max-w-[640px] fade-up"),
				renderSectionEyebrow(parseIntl.T(n, "pricing.hero.eyebrow")),
				H1(ClassStr("font-display mt-4 text-4xl font-bold leading-tight tracking-[-0.03em] text-[#f0f0f8] sm:text-5xl md:text-6xl"), Text(parseIntl.T(n, "pricing.hero.headline"))),
				P(ClassStr("mt-5 max-w-[52ch] text-base leading-7 text-[#9b9bb1] sm:text-lg sm:leading-8"), Text(parseIntl.T(n, "pricing.hero.body"))),
				Div(
					ClassStr("mt-8 flex flex-wrap gap-3"),
					renderCtaPrimary(parseIntl.T(n, "pricing.hero.primaryCta"), "#plans"),
					renderCtaSecondary(parseIntl.T(n, "pricing.hero.secondaryCta"), "#compare"),
				),
			),
			// right: three stat cards
			Div(
				ClassStr("grid gap-4 sm:grid-cols-3 sm:gap-5 fade-up fade-up-d1"),
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
		ClassStr("rounded-2xl border border-white/[0.06] bg-[#13131e] px-5 py-6"),
		Div(ClassStr("text-base font-semibold text-[#f0f0f8]"), Text(renderTitle)),
		P(ClassStr("mt-2 text-sm leading-6 text-[#9b9bb1]"), Text(renderBody)),
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
		ClassStr("pb-16 sm:pb-20 md:pb-24"),
		Div(
			ClassStr("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				ClassStr("mb-10 max-w-[640px] scroll-reveal"),
				renderSectionEyebrow(parseIntl.T(n, "pricing.plans.eyebrow")),
				H2(ClassStr("font-display mt-4 text-3xl font-bold tracking-[-0.04em] text-[#f0f0f8] sm:text-4xl"), Text(parseIntl.T(n, "pricing.plans.h2"))),
			),
			Div(
				ClassStr("grid gap-4 sm:gap-5 lg:grid-cols-3 scroll-reveal scroll-reveal-d1"),
				Map([]pricingPlan{
					{"pricing.starter", marketingSignupRoute, []string{"0", "1", "2", "3", "4"}, false, false},
					{"pricing.team", marketingSignupRoute, []string{"0", "1", "2", "3", "4", "5"}, true, false},
					{"pricing.enterprise", "mailto:sales@relaydesk.com", []string{"0", "1", "2", "3", "4", "5"}, false, true},
				}, func(parseP pricingPlan) ui.Node {
					parseCardClass := "rounded-2xl border border-white/[0.06] bg-[#13131e] px-5 py-7 sm:px-6 sm:py-8"
					if parseP.featured {
						parseCardClass = "pricing-card-featured rounded-2xl border bg-[#13131e] px-5 py-7 sm:px-6 sm:py-8"
					} else if parseP.enterprise {
						parseCardClass = "pricing-card-enterprise rounded-2xl bg-[#13131e] px-5 py-7 sm:px-6 sm:py-8"
					}
					parseBadgeText := parseIntl.T(n, parseP.prefix+".badge")
					var buildBadge ui.Node
					if parseBadgeText != "" {
						buildBadge = Div(ClassStr("mb-4 inline-flex rounded-full bg-[#8e7bff]/10 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.14em] text-[#8e7bff]"), Text(parseBadgeText))
					}
					parseSuffixText := parseIntl.T(n, parseP.prefix+".suffix")
					var buildSuffix ui.Node
					if parseSuffixText != "" {
						buildSuffix = Span(ClassStr("font-mono-tech text-lg text-[#9b9bb1]"), Text(parseSuffixText))
					}
					parsePriceClass := "font-mono-tech mt-4 text-4xl font-bold tracking-[-0.04em] text-[#f0f0f8] sm:text-5xl"
					if parseP.featured {
						parsePriceClass = "font-mono-tech mt-4 text-4xl font-bold tracking-[-0.04em] text-[#8e7bff] sm:text-5xl"
					}
					return Article(
						ClassStr(parseCardClass),
						buildBadge,
						Div(ClassStr("text-[11px] font-semibold uppercase tracking-[0.14em] text-[#9b9bb1]"), Text(parseIntl.T(n, parseP.prefix+".tone"))),
						Div(ClassStr("mt-1 text-lg font-bold text-[#f0f0f8]"), Text(parseIntl.T(n, parseP.prefix+".name"))),
						Div(
							ClassStr(parsePriceClass),
							Text(parseIntl.T(n, parseP.prefix+".price")),
							buildSuffix,
						),
						P(ClassStr("mt-4 text-sm leading-6 text-[#9b9bb1]"), Text(parseIntl.T(n, parseP.prefix+".description"))),
						Ul(
							ClassStr("mt-6 space-y-2.5 text-sm text-[#9b9bb1]"),
							Map(parseP.featureKeys, func(parseIdx string) ui.Node {
								return Li(
									ClassStr("flex items-start gap-3"),
									Span(ClassStr("mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[#8e7bff]"), nil),
									Span(ClassStr("text-[#f0f0f8]"), Text(parseIntl.T(n, parseP.prefix+".feature."+parseIdx))),
								)
							}),
						),
						Div(
							ClassStr("mt-7"),
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
	buildGridArgs = append(buildGridArgs, ClassStr("min-w-[640px] overflow-hidden rounded-2xl border border-white/[0.06] grid grid-cols-4 gap-px bg-white/[0.03] text-sm"))
	for _, buildColKey := range []string{"compare.col.capability", "compare.col.starter", "compare.col.team", "compare.col.enterprise"} {
		buildGridArgs = append(buildGridArgs, Div(ClassStr("bg-[#13131e] px-4 py-3.5 text-[11px] font-semibold uppercase tracking-[0.12em] text-[#9b9bb1]"), Text(parseIntl.T(n, buildColKey))))
	}
	for i, buildRowKey := range compareRowKeys {
		parseBg := "bg-[#0e0e17]"
		if i%2 == 0 {
			parseBg = "bg-[#13131e]"
		}
		buildGridArgs = append(buildGridArgs,
			Div(ClassStr(parseBg+" px-4 py-3.5 text-sm font-medium text-[#f0f0f8]"), Text(parseIntl.T(n, buildRowKey+".label"))),
			Div(ClassStr(parseBg+" px-4 py-3.5 text-sm text-[#9b9bb1]"), Text(parseIntl.T(n, buildRowKey+".starter"))),
			Div(ClassStr(parseBg+" px-4 py-3.5 text-sm text-[#9b9bb1]"), Text(parseIntl.T(n, buildRowKey+".team"))),
			Div(ClassStr(parseBg+" px-4 py-3.5 text-sm text-[#9b9bb1]"), Text(parseIntl.T(n, buildRowKey+".enterprise"))),
		)
	}

	return Section(
		ID("compare"),
		ClassStr("pb-16 sm:pb-20 md:pb-24"),
		Div(
			ClassStr("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				ClassStr("rounded-2xl border border-white/[0.06] bg-[#13131e] px-5 py-7 sm:px-8 sm:py-8 md:px-10 md:py-10 scroll-reveal"),
				renderSectionEyebrow(parseIntl.T(n, "compare.eyebrow")),
				H2(ClassStr("font-display mt-4 text-2xl font-bold tracking-[-0.04em] text-[#f0f0f8] sm:text-3xl"), Text(parseIntl.T(n, "compare.h2"))),
				Div(
					ClassStr("mt-8 overflow-x-auto"),
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
		ClassStr("pb-16 sm:pb-20 md:pb-24"),
		Div(
			ClassStr("mx-auto grid w-[min(1200px,calc(100%-24px))] gap-8 sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[.78fr_1.22fr] lg:gap-12"),
			Div(
				ClassStr("scroll-reveal"),
				renderSectionEyebrow(parseIntl.T(n, "faq.eyebrow")),
				H2(ClassStr("font-display mt-4 text-3xl font-bold tracking-[-0.04em] text-[#f0f0f8] sm:text-4xl"), Text(parseIntl.T(n, "faq.h2"))),
				P(ClassStr("mt-4 text-sm leading-7 text-[#9b9bb1]"), Text(parseIntl.T(n, "faq.body"))),
			),
			Div(
				ClassStr("grid gap-4 scroll-reveal scroll-reveal-d1"),
				Map(faqKeys, func(parseK string) ui.Node {
					return Div(
						ClassStr("rounded-2xl border border-white/[0.06] bg-[#13131e] px-5 py-6 sm:px-6"),
						Div(ClassStr("text-base font-semibold text-[#f0f0f8]"), Text(parseIntl.T(n, parseK+".q"))),
						P(ClassStr("mt-3 text-sm leading-6 text-[#9b9bb1]"), Text(parseIntl.T(n, parseK+".a"))),
					)
				}),
			),
		),
	)
}

// renderPricingContact renders the sales-contact CTA at the bottom of the pricing page.
// renderPricingTrustBand renders a "what a serious buyer does next" escalation band with four CTA groups.
func renderPricingTrustBand(parseIntl i18n.Runtime) ui.Node {
	type parseCTAGroup struct {
		parseLabel   string
		parseCaption string
		parseHref    string
		parseAction  string
	}
	parseGroups := []parseCTAGroup{
		{"Talk to sales", "We can walk through workspace setup, billing fit, and admin requirements before rollout.", "mailto:sales@relaydesk.com", "Email sales →"},
		{"Ask about onboarding", "Our team covers seat provisioning, workspace setup, and first-run guidance for every plan tier.", "mailto:hello@relaydesk.com", "Email team →"},
		{"Review security", "Read our data handling policy, encryption posture, and audit controls before you commit.", marketingSecurityRoute, "Security docs →"},
		{"Check status", "See current uptime, recent incidents, and scheduled maintenance across all services.", marketingStatusRoute, "Status page →"},
	}
	return Section(
		ClassStr("pb-16 sm:pb-20"),
		Div(
			ClassStr("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				ClassStr("mb-8 text-center"),
				renderSectionEyebrow("What a serious buyer does next"),
				H2(ClassStr("font-display mt-4 text-2xl font-bold tracking-[-0.03em] text-[#f0f0f8] sm:text-3xl"), Text("Evaluate with confidence")),
				P(ClassStr("mt-3 text-sm leading-6 text-[#9b9bb1]"), Text("No commitment required. Every path below gives you real answers before rollout.")),
			),
			Div(
				ClassStr("grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4"),
				Map(parseGroups, func(parseG parseCTAGroup) ui.Node {
					return Div(
						ClassStr("rounded-2xl border border-white/[0.08] bg-white/[0.03] p-5 flex flex-col gap-3 scroll-reveal"),
						Div(ClassStr("text-xs font-semibold uppercase tracking-[0.14em] text-[#8e7bff]"), Text(parseG.parseLabel)),
						P(ClassStr("flex-1 text-xs leading-5 text-[#9b9bb1]"), Text(parseG.parseCaption)),
						A(
							ClassStr("inline-flex items-center text-xs font-medium text-[#f0f0f8] hover:text-[#8e7bff] transition-colors"),
							Href(parseG.parseHref),
							Text(parseG.parseAction),
						),
					)
				}),
			),
		),
	)
}

// renderPricingContact renders the contact / reach-out section at the bottom of the pricing page.
func renderPricingContact(parseIntl i18n.Runtime) ui.Node {
	n := marketingI18nNamespace
	return Section(
		ID("contact"),
		ClassStr("pb-20 sm:pb-24 md:pb-28"),
		Div(
			ClassStr("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				ClassStr("rounded-2xl border border-[#8e7bff]/20 bg-[#13131e] px-6 py-8 sm:px-8 sm:py-10 md:px-10 md:py-12 scroll-reveal"),
				Div(
					ClassStr("max-w-[640px]"),
					renderSectionEyebrow(parseIntl.T(n, "contact.eyebrow")),
					H2(ClassStr("font-display mt-4 text-3xl font-bold tracking-[-0.04em] text-[#f0f0f8] sm:text-4xl"), Text(parseIntl.T(n, "contact.h2"))),
					P(ClassStr("mt-4 max-w-[52ch] text-base leading-7 text-[#9b9bb1] sm:leading-8"), Text(parseIntl.T(n, "contact.body"))),
					Div(
						ClassStr("mt-8 flex flex-col gap-3 sm:flex-row"),
						A(
							ClassStr("inline-flex items-center justify-center rounded-full bg-[#8e7bff] px-6 py-3.5 text-sm font-semibold text-[#070710] transition hover:-translate-y-[1px]"),
							Href("mailto:sales@relaydesk.com"),
							Text(parseIntl.T(n, "contact.emailSales")),
						),
						A(
							ClassStr("inline-flex items-center justify-center rounded-full border border-white/[0.12] bg-transparent px-6 py-3.5 text-sm font-semibold text-[#f0f0f8] transition hover:bg-white/[0.06]"),
							Href("mailto:hello@relaydesk.com"),
							Text(parseIntl.T(n, "contact.emailTeam")),
						),
					),
				),
			),
		),
	)
}
