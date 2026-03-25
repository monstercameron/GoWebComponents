//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderPricingShell renders the full standalone pricing page navigated to via the SW router.
func renderPricingShell(_ appViewState) ui.Node {
	return Div(
		Class("relative min-h-screen bg-[radial-gradient(circle_at_12%_10%,rgba(139,92,246,.18),transparent_24%),radial-gradient(circle_at_88%_14%,rgba(236,72,153,.16),transparent_26%),linear-gradient(180deg,#121726_0%,#171c2d_48%,#1b2135_100%)] text-[#f5f7fb] antialiased"),
		// three ambient glow orbs: top-left purple, top-right pink, bottom-center purple
		Div(
			Class("pointer-events-none fixed inset-0 overflow-hidden"),
			Div(Class("absolute left-[6%] top-[6%] h-40 w-40 rounded-full bg-[#8b5cf6]/12 blur-3xl sm:h-56 sm:w-56 lg:h-64 lg:w-64"), nil),
			Div(Class("absolute right-[8%] top-[10%] h-44 w-44 rounded-full bg-[#ec4899]/12 blur-3xl sm:h-60 sm:w-60 lg:h-72 lg:w-72"), nil),
			Div(Class("absolute bottom-[8%] left-1/2 h-52 w-52 -translate-x-1/2 rounded-full bg-[#8b5cf6]/10 blur-3xl sm:h-72 sm:w-72"), nil),
		),
		renderPricingHeader(),
		Main(
			Class("relative z-10"),
			renderPricingHero(),
			renderPricingPlans(),
			renderPricingCompare(),
			renderPricingFAQ(),
			renderPricingContact(),
		),
		renderPricingFooter(),
	)
}

// renderPricingHeader renders the pricing page top bar with brand, in-page nav, and app CTA.
func renderPricingHeader() ui.Node {
	return renderMarketingHeaderShell(
		renderMarketingHeaderBrand("Pricing", authLandingRoute),
		Tag("nav",
			Class("hidden items-center gap-5 lg:flex xl:gap-8"),
			A(Class("text-sm text-[#b8c2d9] transition hover:text-white"), Href("#plans"), Text("Plans")),
			A(Class("text-sm text-[#b8c2d9] transition hover:text-white"), Href("#compare"), Text("Compare")),
			A(Class("text-sm text-[#b8c2d9] transition hover:text-white"), Href("#faq"), Text("FAQ")),
		),
		Div(
			Class("flex w-full items-center gap-2 sm:gap-3 md:w-auto"),
			renderMarketingHeaderAction("Open chat", chatRouteRoot, false, true),
			renderMarketingHeaderAction("Open app", chatRouteRoot, true, false),
		),
	)
}

// renderPricingHero renders the pricing hero: headline copy left, three stat cards right.
func renderPricingHero() ui.Node {
	return Section(
		Class("pb-14 pt-4 sm:pb-18 sm:pt-6 md:pb-20 md:pt-8 lg:pb-24 lg:pt-12"),
		Div(
			Class("mx-auto grid w-[min(1200px,calc(100%-24px))] gap-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-12 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[.95fr_1.05fr] lg:items-end lg:gap-14"),
			// left: headline + body + CTA row
			Div(
				Class("max-w-[720px]"),
				renderMarketingHeroHeading(
					"Premium pricing \u00b7 simple packaging",
					"Clear plans for teams that want AI without the mess.",
					"RelayDesk is priced like a business tool, not a science experiment. Start small, grow into shared usage, and move into a tailored deployment when the workflow proves out.",
					A(Class("inline-flex items-center justify-center rounded-full bg-white px-5 py-3 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px] sm:px-6 sm:py-3.5"), Href("#plans"), Text("Explore plans")),
					A(Class("inline-flex items-center justify-center rounded-full bg-white/10 px-5 py-3 text-sm font-semibold text-white transition hover:bg-white/15 sm:px-6 sm:py-3.5"), Href("#compare"), Text("Compare features")),
				),
			),
			// right: three glass stat cards
			Div(
				Class("grid gap-4 sm:grid-cols-3 sm:gap-5"),
				renderPricingStatCard("Fast adoption", "Clear packaging reduces decision drag."),
				renderPricingStatCard("Low friction", "Start with a simple plan and expand later."),
				renderPricingStatCard("Enterprise path", "Move into governance and tailored workflows."),
			),
		),
	)
}

// renderPricingStatCard renders a single glass stat cell in the pricing hero.
func renderPricingStatCard(renderTitle, renderBody string) ui.Node {
	return Div(
		Class("rounded-[24px] bg-[linear-gradient(180deg,rgba(255,255,255,.14),rgba(255,255,255,.06))] px-5 py-6 sm:rounded-[28px]"),
		Div(Class("text-xl font-semibold tracking-[-0.04em] text-white sm:text-2xl"), Text(renderTitle)),
		P(Class("mt-2 text-sm leading-6 text-[#b8c2d9]"), Text(renderBody)),
	)
}

// renderPricingPlans renders the three plan cards: Starter, Team (featured), Enterprise.
func renderPricingPlans() ui.Node {
	type plan struct {
		name, price, suffix, tone, description, cta, badge string
		features                                           []string
		featured                                           bool
	}

	buildPlans := []plan{
		{
			name: "Starter", price: "$39", suffix: "/mo", tone: "Best for individuals",
			description: "For solo operators and tiny teams that want a polished AI workspace without heavy setup.",
			cta:         "Start free",
			features:    []string{"1 workspace", "Core chat and guided answers", "Basic saved workflows", "Email support"},
		},
		{
			name: "Team", price: "$149", suffix: "/mo", tone: "Most popular", badge: "Best launch tier",
			description: "For small business teams that want shared value fast, cleaner decision workflows, and a more premium operating surface.",
			cta:         "Book a demo", featured: true,
			features: []string{"Up to 15 seats", "Shared workspaces", "Admin controls", "Templates and presets", "Priority support"},
		},
		{
			name: "Enterprise", price: "Custom", suffix: "", tone: "For larger orgs",
			description: "For internal platforms, agencies, and business units that need branded experiences, governance, and deeper workflow fit.",
			cta:         "Talk to sales",
			features:    []string{"Custom seat counts", "Advanced governance", "Private deployment options", "Custom workflows", "Dedicated support"},
		},
	}

	return Section(
		ID("plans"),
		Class("pb-16 sm:pb-20 md:pb-24"),
		Div(
			Class("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				Class("mb-8 max-w-[720px]"),
				Div(Class("text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:text-[11px] sm:tracking-[0.18em]"), Text("Plans")),
				H2(Class("mt-3 text-2xl font-semibold tracking-[-0.04em] text-white sm:text-3xl md:text-4xl"), Text("Choose the surface that matches your stage.")),
			),
			Div(
				Class("grid gap-4 sm:gap-5 lg:grid-cols-3"),
				Map(buildPlans, func(parseP plan) ui.Node {
					buildBg := "bg-[linear-gradient(180deg,rgba(255,255,255,.14),rgba(255,255,255,.06))]"
					buildBodyColor := "text-[#b8c2d9]"
					buildCTAClass := "bg-white/10 text-white hover:bg-white/15"
					if parseP.featured {
						buildBg = "bg-[linear-gradient(180deg,rgba(139,92,246,.24),rgba(255,255,255,.10))]"
						buildBodyColor = "text-[#e6ebf8]/92"
						buildCTAClass = "bg-white text-[#1a1330]"
					}

					var buildBadge ui.Node
					if parseP.badge != "" {
						buildBadge = Div(Class("mb-4 inline-flex rounded-full bg-[#8b5cf6]/12 px-3 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:text-[11px] sm:tracking-[0.18em]"), Text(parseP.badge))
					}

					var buildSuffix ui.Node
					if parseP.suffix != "" {
						buildSuffix = Span(Class("text-lg text-[#b8c2d9]"), Text(parseP.suffix))
					}

					// build Ul args with the class prop followed by each Li
					buildUlArgs := make([]interface{}, 0, len(parseP.features)+1)
					buildUlArgs = append(buildUlArgs, Class("mt-6 space-y-3 text-sm text-[#dfe6f7]"))
					for _, buildFeature := range parseP.features {
						buildUlArgs = append(buildUlArgs, Li(
							Class("flex items-start gap-3"),
							Span(Class("mt-1 h-2 w-2 shrink-0 rounded-full bg-[#8b5cf6]"), nil),
							Span(Text(buildFeature)),
						))
					}

					return Article(
						Class("rounded-[24px] px-5 py-6 sm:rounded-[30px] sm:px-7 sm:py-8 "+buildBg),
						buildBadge,
						Div(Class("mt-4 text-sm font-semibold text-[#dfe6f7]"), Text(parseP.name)),
						Div(Class("mt-1 text-sm text-[#b8c2d9]"), Text(parseP.tone)),
						Div(
							Class("mt-5 text-4xl font-semibold tracking-[-0.04em] text-white sm:text-5xl"),
							Text(parseP.price),
							buildSuffix,
						),
						P(Class("mt-4 text-sm leading-7 "+buildBodyColor), Text(parseP.description)),
						Ul(buildUlArgs...),
						// CTA navigates into the chat app via the SW router
						A(
							Class("mt-8 inline-flex w-full items-center justify-center rounded-full px-5 py-3 text-sm font-semibold transition hover:-translate-y-[1px] "+buildCTAClass),
							Href(chatRouteRoot),
							OnClick(parseLandingNavigateHandler(chatRouteRoot)),
							Text(parseP.cta),
						),
					)
				}),
			),
		),
	)
}

// renderPricingCompare renders the 4-column capability comparison table.
func renderPricingCompare() ui.Node {
	type row struct{ label, starter, team, enterprise string }
	buildRows := []row{
		{"Shared workspace", "\u2014", "Included", "Included"},
		{"Admin controls", "Basic", "Included", "Advanced"},
		{"Workflow presets", "Basic", "Expanded", "Tailored"},
		{"Deployment model", "Hosted", "Hosted", "Custom"},
	}

	// flatten headers + data into a single CSS grid
	buildGridArgs := make([]interface{}, 0, 4+len(buildRows)*4+1)
	buildGridArgs = append(buildGridArgs, Class("min-w-[680px] overflow-hidden rounded-[22px] bg-white/5 grid grid-cols-4 gap-px text-sm"))
	for _, buildHeader := range []string{"Capability", "Starter", "Team", "Enterprise"} {
		buildGridArgs = append(buildGridArgs, Div(Class("bg-white/5 px-4 py-4 font-medium text-[#dfe6f7]"), Text(buildHeader)))
	}
	for _, buildRow := range buildRows {
		buildGridArgs = append(buildGridArgs,
			Div(Class("bg-white/[0.06] px-4 py-4 text-sm text-[#dfe6f7]"), Text(buildRow.label)),
			Div(Class("bg-white/[0.03] px-4 py-4 text-sm text-[#b8c2d9]"), Text(buildRow.starter)),
			Div(Class("bg-white/[0.03] px-4 py-4 text-sm text-[#b8c2d9]"), Text(buildRow.team)),
			Div(Class("bg-white/[0.03] px-4 py-4 text-sm text-[#b8c2d9]"), Text(buildRow.enterprise)),
		)
	}

	return Section(
		ID("compare"),
		Class("pb-16 sm:pb-20 md:pb-24"),
		Div(
			Class("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				Class("rounded-[28px] bg-[linear-gradient(180deg,rgba(255,255,255,.14),rgba(255,255,255,.06))] px-5 py-6 sm:px-8 sm:py-8 md:px-10 md:py-10"),
				Div(Class("text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:text-[11px] sm:tracking-[0.18em]"), Text("Compare")),
				H2(Class("mt-3 text-2xl font-semibold tracking-[-0.04em] text-white sm:text-3xl"), Text("What changes as you move up.")),
				Div(
					Class("mt-8 overflow-x-auto"),
					Div(buildGridArgs...),
				),
			),
		),
	)
}

// renderPricingFAQ renders the FAQ two-column section.
func renderPricingFAQ() ui.Node {
	type faq struct{ q, a string }
	buildFaqs := []faq{
		{
			q: "Can we start small and upgrade later?",
			a: "Yes. The pricing is designed to let smaller teams start with low friction and move into shared or enterprise plans as the workflow hardens.",
		},
		{
			q: "Is this built for non-technical teams?",
			a: "Yes. The product story and interface are intentionally designed to be easier to understand than typical model-heavy AI tools.",
		},
		{
			q: "Do you support internal business workflows?",
			a: "Yes. RelayDesk can be positioned as a decision console, internal knowledge layer, support assistant, or workflow surface.",
		},
	}

	return Section(
		ID("faq"),
		Class("pb-16 sm:pb-20 md:pb-24"),
		Div(
			Class("mx-auto grid w-[min(1200px,calc(100%-24px))] gap-6 sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[.8fr_1.2fr] lg:gap-10"),
			Div(
				Div(Class("text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:text-[11px] sm:tracking-[0.18em]"), Text("FAQ")),
				H2(Class("mt-3 text-3xl font-semibold tracking-[-0.05em] text-white sm:text-4xl"), Text("Questions buyers usually ask first.")),
			),
			Div(
				Class("grid gap-4"),
				Map(buildFaqs, func(parseF faq) ui.Node {
					return Div(
						Class("rounded-[24px] bg-[linear-gradient(180deg,rgba(255,255,255,.14),rgba(255,255,255,.06))] px-5 py-6 sm:px-6"),
						Div(Class("text-lg font-semibold text-white"), Text(parseF.q)),
						P(Class("mt-3 text-sm leading-7 text-[#b8c2d9]"), Text(parseF.a)),
					)
				}),
			),
		),
	)
}

// renderPricingContact renders the gradient sales-contact CTA at the bottom of the pricing page.
func renderPricingContact() ui.Node {
	return Section(
		ID("contact"),
		Class("pb-20 sm:pb-24 md:pb-28"),
		Div(
			Class("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				Class("rounded-[30px] bg-[linear-gradient(135deg,rgba(139,92,246,.24),rgba(236,72,153,.18))] px-6 py-8 sm:px-8 sm:py-10 md:px-10 md:py-12"),
				Div(
					Class("max-w-[720px]"),
					Div(Class("text-[10px] font-semibold uppercase tracking-[0.16em] text-[#1a1330] sm:text-[11px] sm:tracking-[0.18em]"), Text("Contact sales")),
					H2(Class("mt-3 text-3xl font-semibold tracking-[-0.05em] text-white sm:text-4xl md:text-5xl"), Text("Need a more tailored plan for your workflow?")),
					P(Class("mt-4 max-w-[56ch] text-base leading-7 text-[#f7f2ff]/92 sm:text-lg sm:leading-8"), Text("RelayDesk can be shaped into an internal AI surface, decision console, or workflow assistant for the teams you already have.")),
					Div(
						Class("mt-7 flex flex-col gap-3 sm:flex-row"),
						A(
							Class("inline-flex items-center justify-center rounded-full bg-white px-6 py-3.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px]"),
							Href(chatRouteRoot),
							OnClick(parseLandingNavigateHandler(chatRouteRoot)),
							Text("Book a sales call"),
						),
						A(
							Class("inline-flex items-center justify-center rounded-full bg-white/15 px-6 py-3.5 text-sm font-semibold text-white transition hover:bg-white/20"),
							Href(chatRouteRoot),
							OnClick(parseLandingNavigateHandler(chatRouteRoot)),
							Text("Email the team"),
						),
					),
				),
			),
		),
	)
}

// renderPricingFooter renders the pricing-page footer with Pricing, Company, and Legal columns.
func renderPricingFooter() ui.Node {
	return Tag("footer",
		Class("relative z-10 bg-transparent"),
		Div(
			Class("mx-auto grid w-[min(1200px,calc(100%-24px))] gap-8 py-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-10 sm:py-12 md:grid-cols-2 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[1.2fr_.8fr_.8fr_.8fr] lg:gap-12 lg:py-14"),
			// brand blurb
			Div(
				Class("max-w-[34ch] md:col-span-2 lg:col-span-1"),
				Div(
					Class("flex items-center gap-4"),
					Div(Class("grid h-10 w-10 place-items-center rounded-2xl bg-[linear-gradient(135deg,#c4b5fd_0%,#f9a8d4_100%)] text-sm font-black text-[#1a1330] sm:h-11 sm:w-11"), Text("RD")),
					Div(
						Div(Class("text-[14px] font-semibold tracking-[-0.01em] text-white sm:text-[15px]"), Text("RelayDesk")),
						Div(Class("text-[10px] uppercase tracking-[0.16em] text-[#b8c2d9] sm:text-[11px] sm:tracking-[0.18em]"), Text("Enterprise AI workspace")),
					),
				),
				P(Class("mt-5 text-sm leading-7 text-[#b8c2d9]"), Text("RelayDesk helps teams ask better questions, get clearer answers, and move work forward with less confusion.")),
			),
			renderLandingFooterColumn("Pricing",
				renderLandingFooterLink("Starter", "#plans"),
				renderLandingFooterLink("Team", "#plans"),
				renderLandingFooterLink("Enterprise", "#plans"),
				renderLandingFooterLink("Compare", "#compare"),
			),
			renderLandingFooterColumn("Company",
				renderLandingFooterLink("About", "#"),
				renderLandingFooterLink("Customers", "#"),
				renderLandingFooterLink("Security", "#"),
				renderLandingFooterLink("Contact", "#contact"),
			),
			renderLandingFooterColumn("Legal",
				renderLandingFooterLink("Privacy", "#"),
				renderLandingFooterLink("Terms", "#"),
				renderLandingFooterLink("Status", "#"),
				renderLandingFooterLink("Support", "#"),
			),
		),
		Div(
			Div(
				Class("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-col gap-3 py-5 text-xs text-[#b8c2d9] sm:w-[min(1200px,calc(100%-32px))] sm:gap-4 sm:py-6 md:flex-row md:items-center md:justify-between lg:w-[min(1200px,calc(100%-40px))]"),
				Div(Text("\u00a9 2026 RelayDesk, Inc. All rights reserved.")),
				Div(
					Class("flex flex-wrap items-center gap-4 sm:gap-5"),
					A(Class("transition hover:text-white"), Href("#"), Text("Privacy Policy")),
					A(Class("transition hover:text-white"), Href("#"), Text("Terms of Service")),
					A(Class("transition hover:text-white"), Href("#"), Text("Status")),
				),
			),
		),
	)
}
