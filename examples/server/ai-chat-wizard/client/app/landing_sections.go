//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/i18n"
	"github.com/monstercameron/GoWebComponents/v5/ui"
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
		ClassStr("pb-16 sm:pb-20 md:pb-24"),
		Div(
			ClassStr("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				ClassStr("grid gap-10 lg:grid-cols-[.80fr_1.20fr] lg:gap-16"),
				// left: section intro copy
				Div(
					ClassStr("scroll-reveal"),
					renderSectionEyebrow(parseIntl.T(n, parseEyebrowKey)),
					H2(ClassStr("font-display mt-4 max-w-[12ch] text-3xl font-bold leading-tight tracking-[-0.04em] text-[#f0f0f8] sm:text-4xl"), Text(parseIntl.T(n, parseH2Key))),
					P(ClassStr("mt-4 max-w-[46ch] text-base leading-7 text-[#9b9bb1] sm:leading-8"), Text(parseIntl.T(n, parseBodyKey))),
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
		ClassStr("grid gap-4 sm:gap-5 md:grid-cols-2 scroll-reveal scroll-reveal-d1"),
		Map(parseCards, func(parseC productCard) ui.Node {
			return Article(
				ClassStr("feature-card rounded-2xl bg-[#13131e] border border-white/[0.06] px-5 py-6 sm:px-6 sm:py-7"),
				Div(ClassStr("text-2xl"), Text(parseC.icon)),
				H3(ClassStr("mt-3 text-base font-semibold text-[#f0f0f8] sm:text-lg"), Text(parseIntl.T(n, parseC.prefix+".title"))),
				P(ClassStr("mt-2 text-sm leading-6 text-[#9b9bb1]"), Text(parseIntl.T(n, parseC.prefix+".body"))),
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
		ClassStr("pb-16 sm:pb-20 md:pb-24"),
		Div(
			ClassStr("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				ClassStr("grid gap-4 sm:gap-5 lg:grid-cols-[1.10fr_.90fr]"),
				// left: pull-quote card
				Div(
					ClassStr("rounded-2xl border border-white/[0.06] bg-[#13131e] px-6 py-8 sm:px-8 sm:py-10 scroll-reveal"),
					renderSectionEyebrow(parseIntl.T(n, "why.eyebrow")),
					P(
						ClassStr("font-display mt-6 text-2xl font-bold italic leading-snug tracking-[-0.02em] text-[#f0f0f8] sm:text-3xl md:text-4xl"),
						Text(parseIntl.T(n, "why.quote")),
					),
					P(ClassStr("mt-6 max-w-[52ch] text-base leading-7 text-[#9b9bb1] sm:leading-8"), Text(parseIntl.T(n, "why.body"))),
					Div(
						ClassStr("mt-8 flex flex-wrap gap-3"),
						renderCtaPrimary(parseIntl.T(n, "why.primaryCta"), chatRouteRoot),
						renderCtaSecondary(parseIntl.T(n, "why.secondaryCta"), marketingPricingRoute),
					),
				),
				// right: stacked proof cards anchored to real operational surfaces
				Div(
					ClassStr("grid gap-4 sm:gap-5 scroll-reveal scroll-reveal-d1"),
					Map(parseProofKeys, func(parseK string) ui.Node {
						return Div(
							ClassStr("rounded-2xl border border-white/[0.06] bg-[#13131e] px-5 py-6 sm:px-6"),
							// surface label — small-cap eyebrow, not a decorative metric number
							Div(ClassStr("mb-2 inline-flex rounded-full border border-[#8e7bff]/20 bg-[#8e7bff]/8 px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-[0.18em] text-[#8e7bff]/80"), Text(parseIntl.T(n, parseK+".number"))),
							Div(ClassStr("text-base font-semibold text-[#f0f0f8]"), Text(parseIntl.T(n, parseK+".title"))),
							P(ClassStr("mt-2 text-sm leading-6 text-[#9b9bb1]"), Text(parseIntl.T(n, parseK+".body"))),
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
		ClassStr("pb-20 sm:pb-24 md:pb-28"),
		Div(
			ClassStr("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				ClassStr("mb-10 max-w-[640px] scroll-reveal"),
				renderSectionEyebrow(parseIntl.T(n, "pricing.section.eyebrow")),
				H2(ClassStr("font-display mt-4 text-3xl font-bold tracking-[-0.04em] text-[#f0f0f8] sm:text-4xl"), Text(parseIntl.T(n, "pricing.section.h2"))),
				P(ClassStr("mt-4 text-base leading-7 text-[#9b9bb1]"), Text(parseIntl.T(n, "pricing.section.body"))),
			),
			Div(
				ClassStr("grid gap-4 sm:gap-5 lg:grid-cols-3 scroll-reveal scroll-reveal-d1"),
				Map(parseTiers, func(parseT pricingTier) ui.Node {
					parseCardClass := "rounded-2xl border border-white/[0.06] bg-[#13131e] px-5 py-7 sm:px-6 sm:py-8"
					if parseT.featured {
						parseCardClass = "pricing-card-featured rounded-2xl border bg-[#13131e] px-5 py-7 sm:px-6 sm:py-8"
					} else if parseT.enterprise {
						parseCardClass = "pricing-card-enterprise rounded-2xl bg-[#13131e] px-5 py-7 sm:px-6 sm:py-8"
					}
					parseBadgeText := parseIntl.T(n, parseT.prefix+".badge")
					var parseBadge ui.Node
					if parseBadgeText != "" {
						parseBadge = Div(ClassStr("mb-4 inline-flex rounded-full bg-[#8e7bff]/10 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.14em] text-[#8e7bff]"), Text(parseBadgeText))
					}
					parseSuffixText := parseIntl.T(n, parseT.prefix+".suffix")
					var parseSuffix ui.Node
					if parseSuffixText != "" {
						parseSuffix = Span(ClassStr("font-mono-tech text-lg text-[#9b9bb1]"), Text(parseSuffixText))
					}
					parsePriceClass := "font-mono-tech mt-4 text-4xl font-bold tracking-[-0.04em] text-[#f0f0f8] sm:text-5xl"
					if parseT.featured {
						parsePriceClass = "font-mono-tech mt-4 text-4xl font-bold tracking-[-0.04em] text-[#8e7bff] sm:text-5xl"
					}
					return Div(
						ClassStr(parseCardClass),
						parseBadge,
						Div(ClassStr("text-sm font-semibold uppercase tracking-[0.12em] text-[#9b9bb1]"), Text(parseIntl.T(n, parseT.prefix+".name"))),
						Div(
							ClassStr(parsePriceClass),
							Text(parseIntl.T(n, parseT.prefix+".price")),
							parseSuffix,
						),
						P(ClassStr("mt-4 text-sm leading-6 text-[#9b9bb1]"), Text(parseIntl.T(n, parseT.prefix+".body"))),
						Div(
							ClassStr("mt-6"),
							renderCtaPrimary(parseIntl.T(n, parseT.prefix+".cta"), parseT.ctaRoute),
						),
					)
				}),
			),
		),
	)
}

// renderLandingPersonaBand renders a trio of who-this-is-for cards with a before/after framing
// for each named operator persona.
func renderLandingPersonaBand() ui.Node {
	type personaCard struct {
		role, before, after string
	}
	parseCards := []personaCard{
		{
			role:   "Support lead drowning in repeat questions",
			before: "Scattered prompts, no shared history, no audit trail.",
			after:  "One workspace routing repeated questions to grounded answers with per-thread usage visibility.",
		},
		{
			role:   "Ops manager juggling too many disconnected tools",
			before: "Manual handoffs, billing gaps, no clear who-changed-what.",
			after:  "Admin controls, audit log, billing visibility, and usage review from a single dashboard.",
		},
		{
			role:   "Internal knowledge team keeping answers consistent",
			before: "Different teammates getting different answers from the same prompt.",
			after:  "Shared workspace defaults, model routing, and admin oversight of every chat send.",
		},
	}
	return Section(
		ClassStr("pb-16 sm:pb-20 md:pb-24"),
		Div(
			ClassStr("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				ClassStr("mb-8 sm:mb-10 scroll-reveal"),
				renderSectionEyebrow("Who this is for"),
				H2(ClassStr("font-display mt-4 max-w-[24ch] text-3xl font-bold leading-tight tracking-[-0.04em] text-[#f0f0f8] sm:text-4xl"),
					Text("Built for operators, not just users"),
				),
				P(ClassStr("mt-4 max-w-[54ch] text-base leading-7 text-[#9b9bb1] sm:leading-8"),
					Text("If you run a team that depends on consistent, accountable AI answers, RelayDesk was designed for your exact situation."),
				),
			),
			Div(
				ClassStr("grid gap-4 sm:gap-5 lg:grid-cols-3 scroll-reveal scroll-reveal-d1"),
				Map(parseCards, func(parseC personaCard) ui.Node {
					return Div(
						ClassStr("rounded-2xl border border-white/[0.06] bg-[#13131e] px-6 py-7"),
						P(ClassStr("mb-5 text-sm font-semibold leading-snug text-[#f0f0f8]"), Text(parseC.role)),
						Div(
							ClassStr("space-y-3"),
							Div(
								ClassStr("rounded-xl border border-red-400/12 bg-red-500/5 px-4 py-3"),
								Div(ClassStr("mb-1 text-[10px] font-semibold uppercase tracking-[0.18em] text-red-400/70"), Text("Before")),
								P(ClassStr("text-sm leading-6 text-[#9b9bb1]"), Text(parseC.before)),
							),
							Div(
								ClassStr("rounded-xl border border-[#8e7bff]/12 bg-[#8e7bff]/5 px-4 py-3"),
								Div(ClassStr("mb-1 text-[10px] font-semibold uppercase tracking-[0.18em] text-[#8e7bff]/70"), Text("After")),
								P(ClassStr("text-sm leading-6 text-[#9b9bb1]"), Text(parseC.after)),
							),
						),
					)
				}),
			),
		),
	)
}

// renderLandingHowItWorksSection explains the product loop in four concrete steps anchored to real shipped surfaces.
func renderLandingHowItWorksSection() ui.Node {
	type workStep struct {
		num, title, body string
	}
	parseSteps := []workStep{
		{
			num:   "01",
			title: "Connect your workspace",
			body:  "Set model routing, workspace defaults, and team access in Settings. Every chat inherits these choices without manual setup per thread.",
		},
		{
			num:   "02",
			title: "Start a workflow-backed chat",
			body:  "Open a thread, pick your model, and send. The system logs every turn with cost, model ID, and latency from the first message.",
		},
		{
			num:   "03",
			title: "Route to the right model",
			body:  "Admins pick which providers are active, set fallback order, and review routing cost from the Providers dashboard slice.",
		},
		{
			num:   "04",
			title: "Review usage and admin actions",
			body:  "Every billing event, session, and admin mutation is visible in Business, Customers, and Ops. Nothing is hidden between dashboards.",
		},
	}
	return Section(
		ClassStr("pb-16 sm:pb-20 md:pb-24"),
		Div(
			ClassStr("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				ClassStr("mb-8 sm:mb-10 scroll-reveal"),
				renderSectionEyebrow("How it works"),
				H2(ClassStr("font-display mt-4 max-w-[22ch] text-3xl font-bold leading-tight tracking-[-0.04em] text-[#f0f0f8] sm:text-4xl"),
					Text("The product loop, in four steps"),
				),
			),
			Div(
				ClassStr("grid gap-4 sm:gap-5 sm:grid-cols-2 lg:grid-cols-4 scroll-reveal scroll-reveal-d1"),
				Map(parseSteps, func(parseS workStep) ui.Node {
					return Div(
						ClassStr("rounded-2xl border border-white/[0.06] bg-[#13131e] p-5 sm:p-6"),
						Div(ClassStr("font-mono-tech mb-4 text-[2.5rem] font-bold leading-none tracking-tight text-white/10"), Text(parseS.num)),
						P(ClassStr("text-base font-semibold text-[#f0f0f8]"), Text(parseS.title)),
						P(ClassStr("mt-2 text-sm leading-6 text-[#9b9bb1]"), Text(parseS.body)),
					)
				}),
			),
		),
	)
}

// renderLandingPreviewCluster renders three side-by-side product interface panels showing the chat,
// settings, and dashboard surfaces as markup-based mockups. Panels are truthful to the shipped product.
func renderLandingPreviewCluster() ui.Node {
	return Section(
		ClassStr("pb-16 sm:pb-20 md:pb-24"),
		Div(
			ClassStr("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				ClassStr("mb-8 sm:mb-10 scroll-reveal"),
				renderSectionEyebrow("See it in action"),
				H2(ClassStr("font-display mt-4 max-w-[24ch] text-3xl font-bold leading-tight tracking-[-0.04em] text-[#f0f0f8] sm:text-4xl"),
					Text("The product as software"),
				),
				P(ClassStr("mt-4 max-w-[52ch] text-base leading-7 text-[#9b9bb1] sm:leading-8"),
					Text("Three surfaces ship together: a workspace chat session, a settings and billing panel, and an admin dashboard. Each one is functional and live in this example."),
				),
			),
			Div(
				ClassStr("grid gap-4 sm:gap-5 lg:grid-cols-3 scroll-reveal scroll-reveal-d1"),
				renderLandingPreviewPanel("Chat workspace", "Thread view with model selector, streaming reply, cost logged per turn.", renderChatPreviewMockup()),
				renderLandingPreviewPanel("Settings & billing", "Workspace defaults, model preference, display name, and session billing summary.", renderSettingsPreviewMockup()),
				renderLandingPreviewPanel("Admin dashboard", "Business, customers, chats, providers, and ops from one admin surface.", renderDashboardPreviewMockup()),
			),
		),
	)
}

// renderLandingPreviewPanel wraps one mock-UI panel with a window bar and a captioned label below.
func renderLandingPreviewPanel(parseTitle, parseCaption string, parseContent ui.Node) ui.Node {
	return Div(
		ClassStr("flex flex-col"),
		Div(
			ClassStr("overflow-hidden rounded-2xl border border-white/[0.08] bg-[#0e0e14]"),
			Div(
				ClassStr("flex items-center gap-1.5 border-b border-white/[0.06] px-3 py-2.5"),
				Div(ClassStr("h-1.5 w-1.5 rounded-full bg-white/15")),
				Div(ClassStr("h-1.5 w-1.5 rounded-full bg-white/10")),
				Div(ClassStr("h-1.5 w-1.5 rounded-full bg-white/8")),
				Div(ClassStr("ml-3 h-3.5 flex-1 rounded-full bg-white/5")),
			),
			parseContent,
		),
		Div(
			ClassStr("mt-3 px-1"),
			P(ClassStr("text-sm font-semibold text-[#f0f0f8]"), Text(parseTitle)),
			P(ClassStr("mt-1 text-xs leading-5 text-[#9b9bb1]"), Text(parseCaption)),
		),
	)
}

// renderChatPreviewMockup renders a markup-based chat interface mock for the preview cluster.
func renderChatPreviewMockup() ui.Node {
	return Div(
		ClassStr("flex gap-3 p-4"),
		Div(
			ClassStr("w-20 shrink-0"),
			Div(ClassStr("mb-2 h-5 rounded-lg bg-white/5")),
			Div(ClassStr("space-y-1"),
				Div(ClassStr("h-6 rounded-lg bg-[#181830]")),
				Div(ClassStr("h-6 rounded-lg bg-white/[0.03]")),
				Div(ClassStr("h-6 rounded-lg bg-white/[0.03]")),
			),
		),
		Div(
			ClassStr("flex min-w-0 flex-1 flex-col gap-3"),
			Div(ClassStr("ml-auto w-4/5 rounded-2xl border border-[#8fffd8]/10 bg-[#181830] px-3 py-2"),
				Div(ClassStr("mb-1 h-1.5 w-3/4 rounded bg-white/20")),
				Div(ClassStr("h-1.5 w-1/2 rounded bg-white/12")),
			),
			Div(ClassStr("w-full rounded-[1.25rem] border border-[#8e7bff]/10 bg-[#13141f] px-3 py-3"),
				Div(ClassStr("mb-2 flex items-center gap-2"),
					Div(ClassStr("h-4 w-4 rounded-full border border-[#8e7bff]/30 bg-[#8e7bff]/15")),
					Div(ClassStr("h-1.5 w-12 rounded bg-[#8e7bff]/25")),
				),
				Div(ClassStr("space-y-1.5"),
					Div(ClassStr("h-1.5 w-full rounded bg-white/15")),
					Div(ClassStr("h-1.5 w-5/6 rounded bg-white/12")),
					Div(ClassStr("h-1.5 w-4/6 rounded bg-white/10")),
				),
			),
			Div(ClassStr("mt-1 flex items-center gap-2 rounded-[1.5rem] border border-white/10 bg-white/[0.04] px-3 py-2"),
				Div(ClassStr("h-1.5 flex-1 rounded bg-white/10")),
				Div(ClassStr("h-5 w-5 rounded-full bg-white/90")),
			),
		),
	)
}

// renderSettingsPreviewMockup renders a markup-based settings panel mock for the preview cluster.
func renderSettingsPreviewMockup() ui.Node {
	parseRows := []string{"Display name", "Email", "Model", "Thinking effort", "Billing"}
	return Div(
		ClassStr("space-y-2 p-4"),
		Div(ClassStr("mb-3 h-4 w-20 rounded-lg bg-white/8")),
		Map(parseRows, func(parseRow string) ui.Node {
			return Div(
				ClassStr("flex items-center justify-between rounded-xl border border-white/[0.06] px-3 py-2"),
				Div(ClassStr("text-[11px] text-white/45"), Text(parseRow)),
				Div(ClassStr("h-1.5 w-14 rounded bg-white/12")),
			)
		}),
	)
}

// renderDashboardPreviewMockup renders a markup-based admin dashboard mock for the preview cluster.
func renderDashboardPreviewMockup() ui.Node {
	parseTiles := []string{"Business", "Customers", "Chats", "Providers", "Ops"}
	return Div(
		ClassStr("p-4"),
		Div(ClassStr("mb-3 flex items-center gap-2"),
			Div(ClassStr("h-4 w-16 rounded-lg bg-white/8")),
			Div(ClassStr("ml-auto h-3.5 w-14 rounded-full border border-white/10 bg-white/[0.04]")),
		),
		Div(
			ClassStr("grid grid-cols-2 gap-2"),
			Map(parseTiles, func(parseTile string) ui.Node {
				return Div(
					ClassStr("rounded-xl border border-white/8 bg-white/[0.03] px-3 py-3"),
					Div(ClassStr("text-[11px] text-white/50 mb-1.5"), Text(parseTile)),
					Div(ClassStr("h-1.5 w-full rounded bg-white/10")),
					Div(ClassStr("mt-1 h-1.5 w-4/5 rounded bg-white/8")),
				)
			}),
		),
	)
}

type frameworkCallout struct {
	Label     string
	Desc      string
	Href      string
	LinkLabel string
}

func parseBuildFrameworkCallouts() []frameworkCallout {
	const parseRepo = "https://github.com/monstercameron/GoWebComponents/blob/main/"
	return []frameworkCallout{
		{
			Label:     "SSR bootstrap",
			Desc:      "Server-owned shell HTML plus JS loader. WASM mounts on top.",
			Href:      parseRepo + "examples/server/ai-chat-wizard/docs/PUBLIC_ROUTE_DELIVERY.md",
			LinkLabel: "Docs",
		},
		{
			Label:     "Typed routes",
			Desc:      "One Go WASM binary owns every public, auth, and workspace route.",
			Href:      parseRepo + "examples/server/ai-chat-wizard/client/app/routes.go",
			LinkLabel: "Source",
		},
		{
			Label:     "Streaming chat",
			Desc:      "gRPC ChatChunk deltas apply incrementally through GoGRPCBridge.",
			Href:      parseRepo + "examples/server/ai-chat-wizard/docs/CHAT_REQUEST_LIFECYCLE.md",
			LinkLabel: "Docs",
		},
		{
			Label:     "Worker tasks",
			Desc:      "Background WASM worker offloads markdown and render metadata.",
			Href:      parseRepo + "examples/server/ai-chat-wizard/client/backgroundworker",
			LinkLabel: "Source",
		},
		{
			Label:     "Cross-tab sync",
			Desc:      "Persisted preferences and state atoms sync across browser tabs.",
			Href:      parseRepo + "examples/server/ai-chat-wizard/docs/AUTHENTICATED_SHELL.md",
			LinkLabel: "Docs",
		},
		{
			Label:     "Server functions",
			Desc:      "Typed RPCs cover auth, chat, settings, admin, and superuser flows.",
			Href:      parseRepo + "examples/server/ai-chat-wizard/server/app/server.go",
			LinkLabel: "Source",
		},
	}
}

// renderLandingFrameworkCalloutStrip renders a compact GWC framework signal strip for framework-learner audiences.
// It names the live GWC patterns used in RelayDesk and links each one to the README anchor or source.
func renderLandingFrameworkCalloutStrip() ui.Node {
	parseCallouts := parseBuildFrameworkCallouts()
	return Section(
		ClassStr("pb-12 sm:pb-14"),
		Div(
			ClassStr("mx-auto w-[min(1200px,calc(100%-24px))] sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
			Div(
				ClassStr("rounded-2xl border border-white/[0.06] bg-white/[0.02] px-5 py-6 sm:px-7 sm:py-8"),
				Div(
					ClassStr("mb-5 flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between"),
					Div(
						Div(ClassStr("text-[10px] uppercase tracking-[0.18em] text-white/30"), Text("Built with GoWebComponents")),
						P(ClassStr("mt-1 text-sm font-medium text-white/60"), Text("Live framework patterns in this example")),
					),
					A(
						Href("https://github.com/monstercameron/GoWebComponents"),
						Target("_blank"),
						Rel("noreferrer"),
						ClassStr("mt-3 inline-flex items-center text-xs font-medium text-[#8e7bff] hover:text-white transition-colors sm:mt-0"),
						Text("GWC source →"),
					),
				),
				Div(
					ClassStr("grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6"),
					Map(parseCallouts, func(parseC frameworkCallout) ui.Node {
						return A(
							Href(parseC.Href),
							Target("_blank"),
							Rel("noreferrer"),
							ClassStr("block rounded-xl border border-white/[0.06] bg-white/[0.025] px-3 py-3 no-underline transition-colors hover:border-[#8e7bff]/35 hover:bg-[#8e7bff]/[0.05]"),
							Div(ClassStr("mb-1 text-[11px] font-semibold text-[#8e7bff]/80"), Text(parseC.Label)),
							P(ClassStr("text-[10px] leading-4 text-white/35"), Text(parseC.Desc)),
							Span(ClassStr("mt-2 inline-flex text-[10px] font-medium text-white/30"), Text(parseC.LinkLabel+" ->")),
						)
					}),
				),
			),
		),
	)
}
