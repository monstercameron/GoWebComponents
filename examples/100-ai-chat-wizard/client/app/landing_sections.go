//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderLandingProductSection renders the #product grid of feature cards beneath the hero.
func renderLandingProductSection(page string) ui.Node {
	return Section(
		ID("product"),
		Class("pb-16 sm:pb-20 md:pb-24"),
		Div(
			Class("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				Class("grid gap-8 lg:grid-cols-[.82fr_1.18fr] lg:gap-14"),
				// left: section intro copy
				Div(
					Div(Class("text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:text-[11px] sm:tracking-[0.18em]"), Text("What makes it sell")),
					H2(Class("mt-3 max-w-none text-2xl font-semibold leading-tight tracking-[-0.045em] text-white sm:max-w-[12ch] sm:text-3xl md:text-4xl"), Text("It looks sharp, but the win is usability.")),
					P(Class("mt-4 max-w-[46ch] text-base leading-7 text-[#dfe6f7]/88 sm:leading-8"), Text("RelayDesk is not trying to impress buyers with technical complexity. It makes advanced AI feel organized, premium, and commercially useful.")),
				),
				// right: 2x2 feature cards
				renderLandingProductCards(page),
			),
		),
	)
}

// renderLandingProductCards builds the 2x2 feature card grid, varying content by page variant.
func renderLandingProductCards(page string) ui.Node {
	type card struct {
		eyebrow      string
		eyebrowClass string
		title        string
		body         string
	}

	cards := []card{
		{"Low cognitive load", "text-[#8b5cf6]", "Easy to understand", "The interface explains itself quickly, which makes demos land faster and adoption friction lower."},
		{"Premium mood", "text-[#ec4899]", "Dark, calm, modern", "It feels high-end and intriguing without drifting into flashy consumer-app noise."},
		{"Business surface", "text-[#8b5cf6]", "Made to be sold", "Position it as a team copilot, decision console, knowledge layer, or workflow assistant."},
		{"Strong core", "text-[#ec4899]", "Go-native underneath", "You keep the engineering leverage of the GWC stack without making that the burden of the marketing story."},
	}

	if page == landingPageCapabilities {
		cards = []card{
			{"Intent detection", "text-[#8b5cf6]", "Classify instantly", "Requests land in the right queue the moment they arrive, with no manual triage."},
			{"Smart drafting", "text-[#ec4899]", "On-brand answers", "RelayDesk writes in your voice using approved guidance and live context."},
			{"Human takeover", "text-[#8b5cf6]", "Escalation with context", "Escalated threads include a summary, sentiment read, and the recommended next move."},
			{"Performance visibility", "text-[#ec4899]", "One decision view", "See response quality, queue pressure, and automation impact without switching tools."},
		}
	}

	return Div(
		Class("grid gap-4 sm:gap-5 md:grid-cols-2"),
		Map(cards, func(c card) ui.Node {
			return Article(
				Class("rounded-[24px] bg-[linear-gradient(180deg,rgba(255,255,255,.14),rgba(255,255,255,.06))] px-5 py-6 shadow-[inset_0_1px_0_rgba(255,255,255,.04)] sm:rounded-[30px] sm:px-7 sm:py-8"),
				Div(Class("text-sm font-semibold "+c.eyebrowClass), Text(c.eyebrow)),
				H3(Class("mt-3 text-xl font-semibold tracking-[-0.03em] text-white sm:text-2xl"), Text(c.title)),
				P(Class("mt-3 text-sm leading-7 text-[#b8c2d9]"), Text(c.body)),
			)
		}),
	)
}

// renderLandingWhySection renders the #why split: a large left card and three stacked proof cards on the right.
func renderLandingWhySection(_ string) ui.Node {
	type proofCard struct {
		title string
		body  string
	}

	proof := []proofCard{
		{"Faster adoption", "Clean interaction patterns mean less explanation, less training, and a shorter path to value."},
		{"Higher trust", "Readable output helps users accept and defend the recommendations they act on."},
		{"Broader sales surface", "The same product can support multiple buyer narratives without a full rebuild of the experience."},
	}

	return Section(
		ID("why"),
		Class("pb-16 sm:pb-20 md:pb-24"),
		Div(
			Class("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				Class("grid gap-4 sm:gap-5 lg:grid-cols-[1.08fr_.92fr]"),
				// left: primary message card
				Div(
					Class("rounded-[26px] bg-[linear-gradient(180deg,rgba(255,255,255,.14),rgba(255,255,255,.06))] px-5 py-6 sm:rounded-[34px] sm:px-8 sm:py-9 md:px-10 md:py-10"),
					Div(Class("text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:text-[11px] sm:tracking-[0.18em]"), Text("Why teams buy")),
					H2(Class("mt-4 max-w-none text-3xl font-semibold leading-tight tracking-[-0.05em] text-white sm:max-w-[13ch] sm:text-4xl"), Text("You are not selling AI. You are selling reduced confusion.")),
					P(Class("mt-5 max-w-[58ch] text-base leading-7 text-[#dfe6f7]/88 sm:leading-8"), Text("That is the wedge. The product feels contemporary and premium, but the business value is simple: faster decisions, clearer recommendations, less operational drag.")),
				),
				// right: stacked proof cards
				Div(
					Class("grid gap-4 sm:gap-5"),
					Map(proof, func(c proofCard) ui.Node {
						return Div(
							Class("rounded-[24px] bg-[linear-gradient(180deg,rgba(255,255,255,.14),rgba(255,255,255,.06))] px-5 py-6 sm:rounded-[30px] sm:px-7 sm:py-8"),
							Div(Class("text-3xl font-semibold tracking-[-0.04em] text-white sm:text-4xl"), Text(c.title)),
							P(Class("mt-3 text-sm leading-7 text-[#b8c2d9]"), Text(c.body)),
						)
					}),
				),
			),
		),
	)
}

// renderLandingPricingSection renders the #pricing tier grid with Starter, Team, and Custom cards.
func renderLandingPricingSection(_ string) ui.Node {
	type pricingCard struct {
		name     string
		price    string
		suffix   string
		body     string
		featured bool
		badge    string
	}

	cards := []pricingCard{
		{"Starter", "$39", "/mo", "For solo operators and tiny teams that want a polished AI workspace.", false, ""},
		{"Team", "$149", "/mo", "For small business teams that want shared value fast and a cleaner decision workflow.", true, "Best launch tier"},
		{"Custom", "Talk to us", "", "For agencies, internal tools, and branded deployments shaped around a specific workflow.", false, ""},
	}

	return Section(
		ID("pricing"),
		Class("pb-20 sm:pb-24 md:pb-28"),
		Div(
			Class("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				Class("mb-8 max-w-[720px]"),
				Div(Class("text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:text-[11px] sm:tracking-[0.18em]"), Text("Simple pricing")),
				H2(Class("mt-3 text-2xl font-semibold tracking-[-0.04em] text-white sm:text-3xl md:text-4xl"), Text("Package it like a business tool, not a science experiment.")),
			),
			Div(
				Class("grid gap-4 sm:gap-5 lg:grid-cols-3"),
				Map(cards, func(c pricingCard) ui.Node {
					bg := "bg-[linear-gradient(180deg,rgba(255,255,255,.14),rgba(255,255,255,.06))]"
					bodyColor := "text-[#b8c2d9]"
					if c.featured {
						bg = "bg-[linear-gradient(180deg,rgba(139,92,246,.24),rgba(255,255,255,.10))]"
						bodyColor = "text-[#e6ebf8]/92"
					}
					var badge ui.Node
					if c.badge != "" {
						badge = Div(Class("mb-4 inline-flex rounded-full bg-[#8b5cf6]/12 px-3 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:text-[11px] sm:tracking-[0.18em]"), Text(c.badge))
					}
					var suffix ui.Node
					if c.suffix != "" {
						suffix = Span(Class("text-lg text-[#b8c2d9]"), Text(c.suffix))
					}
					return Div(
						Class("rounded-[24px] px-5 py-6 sm:rounded-[30px] sm:px-7 sm:py-8 "+bg),
						badge,
						Div(Class("text-sm font-semibold text-[#dfe6f7]"), Text(c.name)),
						Div(
							Class("mt-4 text-4xl font-semibold tracking-[-0.04em] text-white sm:text-5xl"),
							Text(c.price),
							suffix,
						),
						P(Class("mt-4 text-sm leading-7 "+bodyColor), Text(c.body)),
					)
				}),
			),
		),
	)
}
