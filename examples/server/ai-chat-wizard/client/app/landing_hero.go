//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/i18n"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// renderLandingHeroSection renders the full-width hero: headline copy on the left, demo chat card on the right.
func renderLandingHeroSection(parseIntl i18n.Runtime, parsePage string) ui.Node {
	n := marketingI18nNamespace
	// select the hero key prefix and routes for the current page variant
	var parseHeroKey, parsePrimaryRoute, parseSecondaryRoute string
	switch parsePage {
	case landingPageCapabilities:
		parseHeroKey = "hero.capabilities."
		parsePrimaryRoute = chatRouteRoot
		parseSecondaryRoute = marketingPricingRoute
	case landingPagePricing:
		parseHeroKey = "hero.pricing."
		parsePrimaryRoute = chatRouteRoot
		parseSecondaryRoute = marketingHomeRoute
	default:
		parseHeroKey = "hero.home."
		parsePrimaryRoute = chatRouteRoot
		parseSecondaryRoute = marketingPricingRoute
	}

	return Section(
		ClassStr("pb-16 pt-4 sm:pb-20 sm:pt-6 md:pb-24 md:pt-8 lg:pb-28 lg:pt-12"),
		Div(
			ClassStr("mx-auto grid w-[min(1200px,calc(100%-24px))] items-center gap-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-12 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[1.05fr_.95fr] lg:gap-16 xl:gap-20"),
			// left: headline, body, CTAs, metric strip
			Div(
				ClassStr("max-w-[640px] pt-2 sm:pt-4 fade-up"),
				Img(
					Src(brandLogoURL),
					Attr("alt", "RelayDesk"),
					ClassStr("mb-6 h-14 w-auto sm:h-16"),
				),
				renderSectionEyebrow(parseIntl.T(n, parseHeroKey+"eyebrow")),
				H1(
					ClassStr("hero-gradient-text font-display mt-4 text-5xl font-bold leading-[1.08] tracking-[-0.03em] sm:text-6xl md:text-7xl lg:text-[5rem]"),
					Text(parseIntl.T(n, parseHeroKey+"headline")),
				),
				P(ClassStr("mt-6 max-w-[52ch] text-base leading-7 text-[#9b9bb1] sm:text-lg sm:leading-8"), Text(parseIntl.T(n, parseHeroKey+"body"))),
				Div(
					ClassStr("mt-8 flex flex-wrap items-center gap-3 sm:mt-10 sm:gap-4"),
					renderCtaPrimary(parseIntl.T(n, parseHeroKey+"primaryCta"), parsePrimaryRoute),
					renderCtaSecondary(parseIntl.T(n, parseHeroKey+"secondaryCta"), parseSecondaryRoute),
				),
				// metric strip
				Div(
					ClassStr("mt-10 flex flex-wrap items-center gap-6 border-t border-white/[0.06] pt-8 sm:mt-12 sm:gap-8 sm:pt-10"),
					renderLandingHeroMetric(parseIntl.T(n, "metric.responseTime.value"), parseIntl.T(n, "metric.responseTime.label")),
					renderLandingHeroMetric(parseIntl.T(n, "metric.models.value"), parseIntl.T(n, "metric.models.label")),
					Div(
						ClassStr("flex items-center gap-2"),
						renderStatusDot(),
						Div(
							ClassStr("font-mono-tech text-xs text-[#9b9bb1]"),
							Span(ClassStr("font-semibold text-[#4ade80]"), Text(parseIntl.T(n, "metric.uptime.value"))),
							Text(" "+parseIntl.T(n, "metric.uptime.label")),
						),
					),
				),
			),
			// right: mock chat demo card
			renderLandingDemoCard(parseIntl),
		),
	)
}

// renderLandingHeroMetric renders a single metric cell in the hero stat strip.
func renderLandingHeroMetric(parseValue, parseLabel string) ui.Node {
	return Div(
		ClassStr("font-mono-tech"),
		Div(ClassStr("text-sm font-semibold text-[#f0f0f8]"), Text(parseValue)),
		Div(ClassStr("mt-0.5 text-[11px] uppercase tracking-[0.12em] text-[#9b9bb1]"), Text(parseLabel)),
	)
}

// renderLandingDemoCard renders the mock chat UI placed beside the hero headline.
func renderLandingDemoCard(parseIntl i18n.Runtime) ui.Node {
	n := marketingI18nNamespace
	return Div(
		ID("demo"),
		ClassStr("relative order-first lg:order-none fade-up fade-up-d1"),
		// subtle cyan glow behind card
		Div(ClassStr("absolute -inset-4 rounded-[48px] bg-[#8e7bff]/[0.04] blur-2xl"), nil),
		// card shell
		Div(
			ClassStr("relative overflow-hidden rounded-[24px] border border-white/[0.08] bg-[#13131e] p-4 shadow-[0_32px_80px_rgba(0,0,0,.48)] sm:rounded-[28px] sm:p-5"),
			// card header: brand + model badge
			Div(
				ClassStr("flex flex-wrap items-center justify-between gap-3"),
				Div(
					ClassStr("flex items-center gap-3"),
					renderBrandMark(),
					Div(
						Div(ClassStr("text-sm font-semibold text-[#f0f0f8]"), Text(parseIntl.T(n, "demo.brandLabel"))),
						Div(ClassStr("font-mono-tech text-[10px] text-[#4ade80]"), Text(parseIntl.T(n, "demo.model"))),
					),
				),
				Div(
					ClassStr("hidden items-center gap-1.5 md:flex"),
					renderStatusDot(),
					Span(ClassStr("font-mono-tech text-[11px] text-[#9b9bb1]"), Text(parseIntl.T(n, "demo.live"))),
				),
			),
			// messages thread
			Div(
				ClassStr("mt-5 space-y-3 sm:mt-6"),
				// AI response bubble
				Div(
					ClassStr("max-w-[90%] rounded-2xl bg-[#0e0e17] px-4 py-4 sm:px-5"),
					Div(
						ClassStr("flex items-start gap-3"),
						Div(ClassStr("mt-0.5 grid h-7 w-7 shrink-0 place-items-center rounded-full bg-[#8e7bff] text-[10px] font-black text-[#070710] sm:h-8 sm:w-8"), Text("RD")),
						Div(
							P(ClassStr("text-sm font-medium leading-6 text-[#f0f0f8] sm:text-base sm:leading-7"), Text(parseIntl.T(n, "demo.question"))),
							P(ClassStr("mt-2 text-sm leading-6 text-[#9b9bb1]"), Text(parseIntl.T(n, "demo.answer"))),
						),
					),
				),
				// user reply
				Div(
					ClassStr("flex justify-end"),
					Div(ClassStr("max-w-[70%] rounded-2xl bg-[#8e7bff]/[0.08] px-4 py-3 text-sm text-[#f0f0f8]"), Text(parseIntl.T(n, "demo.reply"))),
				),
				// metric chips
				Div(
					ClassStr("grid gap-2 sm:grid-cols-3"),
					renderLandingDemoChip(parseIntl.T(n, "demo.chip.setup.label"), parseIntl.T(n, "demo.chip.setup.value")),
					renderLandingDemoChip(parseIntl.T(n, "demo.chip.output.label"), parseIntl.T(n, "demo.chip.output.value")),
					renderLandingDemoChip(parseIntl.T(n, "demo.chip.value.label"), parseIntl.T(n, "demo.chip.value.value")),
				),
				// composer row
				Div(
					ClassStr("rounded-2xl border border-white/[0.06] bg-[#0e0e17] px-4 py-3"),
					Div(
						ClassStr("flex items-center gap-3 text-sm text-[#9b9bb1]"),
						Span(ClassStr("min-w-0 flex-1 text-[#9b9bb1]"),
							Text(parseIntl.T(n, "demo.composer.prompt")),
							Span(ClassStr("inline-block h-4 w-px animate-pulse bg-[#8e7bff] align-middle"), nil),
						),
						Span(ClassStr("grid h-8 w-8 shrink-0 place-items-center rounded-full bg-[#8e7bff] text-[#070710] sm:h-9 sm:w-9"), Text("\u2191")),
					),
				),
			),
		),
	)
}

// renderLandingDemoChip renders a small label/value chip inside the demo card.
func renderLandingDemoChip(parseLabel, parseValue string) ui.Node {
	return Div(
		ClassStr("rounded-xl bg-[#0e0e17] px-3 py-3"),
		Div(ClassStr("font-mono-tech text-[10px] uppercase tracking-[0.14em] text-[#9b9bb1]"), Text(parseLabel)),
		Div(ClassStr("mt-1.5 text-sm font-medium text-[#f0f0f8]"), Text(parseValue)),
	)
}
