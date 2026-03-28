//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderLandingProductSection renders the #product grid of feature cards beneath the hero.
func renderLandingProductSection(parseIntl i18n.Runtime, parsePage string) ui.Node {
	n := marketingI18nNamespace
	parseEyebrowKey := "product.home.eyebrow"
	parseH2Key := "product.home.h2"
	parseBodyKey := "product.home.body"
	if parsePage == landingPageCapabilities {
		parseEyebrowKey = "product.capabilities.eyebrow"
		parseH2Key = "product.capabilities.h2"
		parseBodyKey = "product.capabilities.body"
	}
	return Section(
		ID("product"),
		Class("pb-16 sm:pb-20 md:pb-24"),
		Div(
			Class("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
				Div(
					Class("grid gap-10 lg:grid-cols-[.80fr_1.20fr] lg:gap-16"),
					// left: section intro copy
					Div(
						Class("scroll-reveal"),
					renderSectionEyebrow(parseIntl.T(n, parseEyebrowKey)),
					H2(Class("font-display mt-4 max-w-[12ch] text-3xl font-bold leading-tight tracking-[-0.04em] text-[#f0f0f8] sm:text-4xl"), Text(parseIntl.T(n, parseH2Key))),
					P(Class("mt-4 max-w-[46ch] text-base leading-7 text-[#8a8a9a] sm:leading-8"), Text(parseIntl.T(n, parseBodyKey))),
				),
				// right: 2x2 feature cards
				renderLandingProductCards(parseIntl, parsePage),
			),
		),
	)
}

// renderLandingProductCards builds the 2x2 feature card grid, varying content by page variant.
func renderLandingProductCards(parseIntl i18n.Runtime, parsePage string) ui.Node {
	n := marketingI18nNamespace
	type productCard struct {
		icon   string
		prefix string
	}
	parseCards := []productCard{
		{"\u26a1", "product.home.card.responses"},
		{"\U0001F4C4", "product.home.card.docqa"},
		{"\U0001F4CA", "product.home.card.decisionlog"},
		{"\U0001F512", "product.home.card.enterprise"},
	}
	if parsePage == landingPageCapabilities {
		parseCards = []productCard{
			{"\U0001F9E0", "product.capabilities.card.routing"},
			{"\u270d", "product.capabilities.card.drafting"},
			{"\U0001F504", "product.capabilities.card.escalation"},
			{"\U0001F4CC", "product.capabilities.card.usage"},
		}
	}
	return Div(
		Class("grid gap-4 sm:gap-5 md:grid-cols-2 scroll-reveal scroll-reveal-d1"),
		Map(parseCards, func(parseC productCard) ui.Node {
			return Article(
				Class("feature-card rounded-2xl bg-[#111118] border border-white/[0.06] px-5 py-6 sm:px-6 sm:py-7"),
				Div(Class("text-2xl"), Text(parseC.icon)),
				H3(Class("mt-3 text-base font-semibold text-[#f0f0f8] sm:text-lg"), Text(parseIntl.T(n, parseC.prefix+".title"))),
				P(Class("mt-2 text-sm leading-6 text-[#8a8a9a]"), Text(parseIntl.T(n, parseC.prefix+".body"))),
			)
		}),
	)
}

// renderLandingWhySection renders the #why split: a large left pull-quote and three stacked proof cards on the right.
func renderLandingWhySection(parseIntl i18n.Runtime, _ string) ui.Node {
	n := marketingI18nNamespace
	parseProofKeys := []string{"why.proof1", "why.proof2", "why.proof3"}
	return Section(
		ID("why"),
		Class("pb-16 sm:pb-20 md:pb-24"),
		Div(
			Class("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				Class("grid gap-4 sm:gap-5 lg:grid-cols-[1.10fr_.90fr]"),
				// left: pull-quote card
				Div(
					Class("rounded-2xl border border-white/[0.06] bg-[#111118] px-6 py-8 sm:px-8 sm:py-10 scroll-reveal"),
					renderSectionEyebrow(parseIntl.T(n, "why.eyebrow")),
					P(
						Class("font-display mt-6 text-2xl font-bold italic leading-snug tracking-[-0.02em] text-[#f0f0f8] sm:text-3xl md:text-4xl"),
						Text(parseIntl.T(n, "why.quote")),
					),
					P(Class("mt-6 max-w-[52ch] text-base leading-7 text-[#8a8a9a] sm:leading-8"), Text(parseIntl.T(n, "why.body"))),
					Div(
						Class("mt-8 flex flex-wrap gap-3"),
						renderCtaPrimary(parseIntl.T(n, "why.primaryCta"), chatRouteRoot),
						renderCtaSecondary(parseIntl.T(n, "why.secondaryCta"), marketingPricingRoute),
					),
				),
				// right: stacked proof cards
				Div(
					Class("grid gap-4 sm:gap-5 scroll-reveal scroll-reveal-d1"),
					Map(parseProofKeys, func(parseK string) ui.Node {
						return Div(
							Class("rounded-2xl border border-white/[0.06] bg-[#111118] px-5 py-6 sm:px-6"),
							Div(Class("metric-glow font-mono-tech text-3xl font-bold text-[#00d9ff] sm:text-4xl"), Text(parseIntl.T(n, parseK+".number"))),
							Div(Class("mt-2 text-base font-semibold text-[#f0f0f8]"), Text(parseIntl.T(n, parseK+".title"))),
							P(Class("mt-2 text-sm leading-6 text-[#8a8a9a]"), Text(parseIntl.T(n, parseK+".body"))),
						)
					}),
				),
			),
		),
	)
}

// renderLandingPricingSection renders the #pricing tier grid with Starter, Team, and Enterprise cards.
func renderLandingPricingSection(parseIntl i18n.Runtime, _ string) ui.Node {
	n := marketingI18nNamespace
	type pricingTier struct {
		prefix     string
		ctaRoute   string
		featured   bool
		enterprise bool
	}
	parseTiers := []pricingTier{
		{"pricing.starter", marketingSignupRoute, false, false},
		{"pricing.team", marketingSignupRoute, true, false},
		{"pricing.enterprise", "mailto:sales@relaydesk.com", false, true},
	}
	return Section(
		ID("pricing"),
		Class("pb-20 sm:pb-24 md:pb-28"),
		Div(
			Class("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				Class("mb-10 max-w-[640px] scroll-reveal"),
				renderSectionEyebrow(parseIntl.T(n, "pricing.section.eyebrow")),
				H2(Class("font-display mt-4 text-3xl font-bold tracking-[-0.04em] text-[#f0f0f8] sm:text-4xl"), Text(parseIntl.T(n, "pricing.section.h2"))),
				P(Class("mt-4 text-base leading-7 text-[#8a8a9a]"), Text(parseIntl.T(n, "pricing.section.body"))),
			),
			Div(
				Class("grid gap-4 sm:gap-5 lg:grid-cols-3 scroll-reveal scroll-reveal-d1"),
				Map(parseTiers, func(parseT pricingTier) ui.Node {
					parseCardClass := "rounded-2xl border border-white/[0.06] bg-[#111118] px-5 py-7 sm:px-6 sm:py-8"
					if parseT.featured {
						parseCardClass = "pricing-card-featured rounded-2xl border bg-[#111118] px-5 py-7 sm:px-6 sm:py-8"
					} else if parseT.enterprise {
						parseCardClass = "pricing-card-enterprise rounded-2xl bg-[#111118] px-5 py-7 sm:px-6 sm:py-8"
					}
					parseBadgeText := parseIntl.T(n, parseT.prefix+".badge")
					var parseBadge ui.Node
					if parseBadgeText != "" {
						parseBadge = Div(Class("mb-4 inline-flex rounded-full bg-[#00d9ff]/10 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.14em] text-[#00d9ff]"), Text(parseBadgeText))
					}
					parseSuffixText := parseIntl.T(n, parseT.prefix+".suffix")
					var parseSuffix ui.Node
					if parseSuffixText != "" {
						parseSuffix = Span(Class("font-mono-tech text-lg text-[#8a8a9a]"), Text(parseSuffixText))
					}
					parsePriceClass := "font-mono-tech mt-4 text-4xl font-bold tracking-[-0.04em] text-[#f0f0f8] sm:text-5xl"
					if parseT.featured {
						parsePriceClass = "font-mono-tech mt-4 text-4xl font-bold tracking-[-0.04em] text-[#00d9ff] sm:text-5xl"
					}
					return Div(
						Class(parseCardClass),
						parseBadge,
						Div(Class("text-sm font-semibold uppercase tracking-[0.12em] text-[#8a8a9a]"), Text(parseIntl.T(n, parseT.prefix+".name"))),
						Div(
							Class(parsePriceClass),
							Text(parseIntl.T(n, parseT.prefix+".price")),
							parseSuffix,
						),
						P(Class("mt-4 text-sm leading-6 text-[#8a8a9a]"), Text(parseIntl.T(n, parseT.prefix+".body"))),
						Div(
							Class("mt-6"),
							renderCtaPrimary(parseIntl.T(n, parseT.prefix+".cta"), parseT.ctaRoute),
						),
					)
				}),
			),
		),
	)
}
