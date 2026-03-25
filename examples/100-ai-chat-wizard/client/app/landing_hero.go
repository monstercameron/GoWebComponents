//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderLandingHeroSection renders the full-width hero: headline copy on the left, demo chat card on the right.
func renderLandingHeroSection(parsePage string) ui.Node {
	parseEyebrow := "Moody \u00b7 modern \u00b7 business-ready"
	parseHeadline := "The AI workspace people understand in one glance."
	parseBody := "RelayDesk takes the power of an advanced chat system and turns it into something calm, clear, and easy to trust. It feels premium, but it sells on simplicity."
	parsePrimaryLabel := "See the product"
	parseSecondaryLabel := "Why teams buy it"
	parseSecondaryRoute := marketingCapabilitiesRoute

	switch parsePage {
	case landingPageCapabilities:
		parseEyebrow = "What makes it sell"
		parseHeadline = "It looks sharp, but the win is usability."
		parseBody = "RelayDesk is not trying to impress buyers with technical complexity. It makes advanced AI feel organized, premium, and commercially useful."
		parsePrimaryLabel = "See the product"
		parseSecondaryLabel = "See pricing"
		parseSecondaryRoute = marketingPricingRoute
	case landingPagePricing:
		parseEyebrow = "Simple pricing"
		parseHeadline = "Package it like a business tool, not a science experiment."
		parseBody = "Choose the tier that fits your team today. Scale without rebuilding workflows every quarter."
		parsePrimaryLabel = "Get started"
		parseSecondaryLabel = "See solutions"
		parseSecondaryRoute = marketingCapabilitiesRoute
	}

	return Section(
		Class("pb-16 pt-4 sm:pb-20 sm:pt-6 md:pb-24 md:pt-8 lg:pb-28 lg:pt-12"),
		Div(
			Class("mx-auto grid w-[min(1200px,calc(100%-24px))] items-start gap-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-12 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[1.02fr_.98fr] lg:gap-12 xl:gap-16"),
			// left: headline, body, CTAs, stat cells
			Div(
				Class("max-w-[700px] pt-2 sm:pt-4"),
				renderMarketingHeroHeading(
					parseEyebrow,
					parseHeadline,
					parseBody,
					parseLandingActionButton(parsePrimaryLabel, chatRouteRoot, true),
					parseLandingActionButton(parseSecondaryLabel, parseSecondaryRoute, false),
				),
				Div(
					Class("mt-10 grid gap-6 sm:mt-12 sm:grid-cols-3 sm:gap-8 lg:mt-14"),
					renderLandingHeroStat("Clear", "Structured answers and obvious next steps."),
					renderLandingHeroStat("Calm", "Low-noise UI that does not intimidate normal users."),
					renderLandingHeroStat("Flexible", "Sell it to ops, support, research, or internal teams."),
				),
			),
			// right: mock chat demo card
			renderLandingDemoCard(),
		),
	)
}

// renderLandingHeroStat renders a single headline/body stat cell shown below the hero copy.
func renderLandingHeroStat(parseTitle, parseBody string) ui.Node {
	return Div(
		Div(Class("text-2xl font-semibold tracking-[-0.04em] text-white sm:text-3xl"), Text(parseTitle)),
		P(Class("mt-2 max-w-[28ch] text-sm leading-6 text-[#b8c2d9]"), Text(parseBody)),
	)
}

// renderLandingDemoCard renders the mock chat UI placed beside the hero headline.
func renderLandingDemoCard() ui.Node {
	return Div(
		ID("demo"),
		Class("relative order-first lg:order-none lg:pt-2"),
		// ambient glow orbs behind the card
		Div(Class("absolute -left-4 top-12 h-20 w-20 rounded-full bg-[#8b5cf6]/12 blur-3xl sm:-left-6 sm:top-16 sm:h-24 sm:w-24"), nil),
		Div(Class("absolute -right-4 top-2 h-24 w-24 rounded-full bg-[#ec4899]/12 blur-3xl sm:-right-6 sm:top-4 sm:h-28 sm:w-28"), nil),
		// outer card shell with gradient border glow
		Div(
			Class("relative overflow-hidden rounded-[24px] bg-[#1b2133]/80 p-3 shadow-[0_24px_80px_rgba(16,20,36,.24)] sm:rounded-[30px] sm:p-4 lg:rounded-[36px]"),
			Div(Class("absolute inset-0 bg-[linear-gradient(135deg,rgba(139,92,246,.24),rgba(236,72,153,.18))] opacity-40"), nil),
			// inner card
			Div(
				Class("relative rounded-[20px] bg-[#20273b]/78 p-4 sm:rounded-[24px] sm:p-5 lg:rounded-[30px]"),
				// card header: brand + status badges
				Div(
					Class("flex flex-wrap items-start justify-between gap-3 sm:gap-4"),
					Div(
						Class("flex min-w-0 items-center gap-3"),
						Div(Class("grid h-9 w-9 shrink-0 place-items-center rounded-2xl bg-[#8b5cf6] text-sm font-black text-[#1a1330] sm:h-10 sm:w-10"), Text("RD")),
						Div(
							Class("min-w-0"),
							Div(Class("truncate text-sm font-semibold text-white"), Text("RelayDesk")),
							Div(Class("truncate text-[10px] uppercase tracking-[0.16em] text-[#b8c2d9] sm:text-[11px] sm:tracking-[0.18em]"), Text("Decision workspace")),
						),
					),
					Div(
						Class("hidden items-center gap-2 md:flex"),
						Div(Class("rounded-full bg-white/6 px-3 py-2 text-[11px] uppercase tracking-[0.18em] text-[#dfe6f7]"), Text("Best fit")),
						Div(Class("rounded-full bg-white/6 px-3 py-2 text-[11px] uppercase tracking-[0.18em] text-[#dfe6f7]"), Text("Reasoning on")),
					),
				),
				// messages thread
				Div(
					Class("mt-5 space-y-3 sm:mt-6 sm:space-y-4"),
					// AI response bubble
					Div(
						Class("max-w-full rounded-[22px] bg-[#272f46]/74 px-4 py-4 sm:max-w-[90%] sm:rounded-[28px] sm:px-5 sm:py-5"),
						Div(
							Class("flex items-start gap-3"),
							Div(Class("mt-0.5 grid h-8 w-8 shrink-0 place-items-center rounded-full bg-[#8b5cf6] text-[10px] font-black text-[#1a1330] sm:h-9 sm:w-9 sm:text-xs"), Text("RD")),
							Div(
								P(Class("text-base font-medium leading-7 text-white sm:text-lg sm:leading-8"), Text("Which vendor will be easiest for my team to adopt?")),
								P(Class("mt-2 text-sm leading-6 text-[#e6ebf8]/92 sm:mt-3 sm:leading-7"), Text("I will compare rollout friction, training burden, pricing clarity, and day-one usability, then give you the safest option first.")),
							),
						),
					),
					// user reply bubble
					Div(
						Class("flex justify-end"),
						Div(Class("max-w-[85%] rounded-[18px] bg-[#352746]/74 px-4 py-3 text-sm text-white sm:max-w-[68%] sm:rounded-[22px]"), Text("Keep it simple. My team is not technical.")),
					),
					// metric chips
					Div(
						Class("grid gap-3 sm:grid-cols-3"),
						renderLandingDemoChip("Setup", "Guided rollout"),
						renderLandingDemoChip("Output", "Plain language"),
						renderLandingDemoChip("Value", "Same-day clarity"),
					),
					// composer row
					Div(
						Class("rounded-[18px] bg-[#232b41]/74 px-4 py-4 sm:rounded-[24px]"),
						Div(
							Class("flex items-center gap-3 text-sm text-[#b8c2d9]"),
							Span(Class("min-w-0 flex-1 text-[#dfe6f7]"), Text("Ask about your workflow, vendors, queue, docs, or next step...")),
							Span(Class("grid h-9 w-9 shrink-0 place-items-center rounded-full bg-white text-[#1a1330] sm:h-10 sm:w-10"), Text("up")),
						),
					),
				),
			),
		),
	)
}

// renderLandingDemoChip renders a small label/value chip inside the demo card.
func renderLandingDemoChip(parseLabel, parseValue string) ui.Node {
	return Div(
		Class("rounded-[18px] bg-white/[0.03] px-4 py-4 sm:rounded-[24px]"),
		Div(Class("text-[10px] uppercase tracking-[0.16em] text-[#b8c2d9] sm:text-[11px] sm:tracking-[0.18em]"), Text(parseLabel)),
		Div(Class("mt-2 text-base font-semibold text-white"), Text(parseValue)),
	)
}
