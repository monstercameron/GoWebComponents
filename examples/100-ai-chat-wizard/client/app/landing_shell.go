//go:build js && wasm

package app

import (
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

const landingPageHome = "home"
const landingPageCapabilities = "capabilities"
const landingPagePricing = "pricing"

func renderLandingShell(_ i18n.Runtime, view appViewState, _ authSessionController) ui.Node {
	page := landingPageForPath(view.CurrentPath)
	return Div(
		Class("rd2-root min-h-screen w-full overflow-x-hidden overflow-y-auto text-white"),
		Div(Class("rd2-grid-overlay"), nil),
		Div(Class("rd2-light rd2-light-a"), nil),
		Div(Class("rd2-light rd2-light-b"), nil),
		Div(Class("rd2-light rd2-light-c"), nil),
		Header(
			Class("rd2-header sticky top-0 z-40 border-b border-white/10"),
			Div(
				Class("rd2-shell mx-auto flex w-full max-w-[84rem] items-center justify-between gap-4 px-6 py-4"),
				Div(
					Class("flex items-center gap-3"),
					Div(
						Class("rd2-brand-icon"),
						Text("R"),
					),
					Div(
						P(Class("text-[0.98rem] font-semibold tracking-[0.07em] text-white"), Text("RelayDesk")),
						P(Class("text-[0.62rem] uppercase tracking-[0.28em] text-white/52"), Text("Chat service")),
					),
				),
				Tag("nav",
					Class("hidden items-center gap-7 md:flex"),
					landingNavLink(view.CurrentPath, authLandingRoute, "Home"),
					landingNavLink(view.CurrentPath, marketingCapabilitiesRoute, "Solutions"),
					landingNavLink(view.CurrentPath, marketingPricingRoute, "Pricing"),
				),
				Div(
					Class("flex items-center gap-2"),
					landingActionButton("Launch chat", chatRouteRoot, false),
				),
			),
		),
		Main(
			Class("rd2-shell relative z-10 mx-auto flex w-full max-w-[84rem] flex-col gap-7 px-6 pb-20 pt-10"),
			Section(
				Class("grid items-stretch gap-5 lg:grid-cols-[1.15fr_0.85fr]"),
				renderRD2Hero(page),
				renderRD2QueueCard(page),
			),
			renderRD2Middle(page),
			renderRD2FinalCTA(page),
		),
	)
}

func landingPageForPath(path string) string {
	switch strings.TrimSpace(path) {
	case marketingCapabilitiesRoute:
		return landingPageCapabilities
	case marketingPricingRoute:
		return landingPagePricing
	case marketingHomeRoute, authLandingRoute:
		return landingPageHome
	default:
		return landingPageHome
	}
}

func landingNavLink(currentPath, targetPath, label string) ui.Node {
	isActive := strings.TrimSpace(currentPath) == targetPath || (targetPath == authLandingRoute && strings.TrimSpace(currentPath) == marketingHomeRoute)
	return A(
		Class(ClassNames(
			"text-sm font-medium tracking-[0.1em] transition-colors",
			When(isActive, "text-white"),
			When(!isActive, "text-white/58 hover:text-white"),
		)),
		Href(targetPath),
		OnClick(landingNavigateHandler(targetPath)),
		Text(label),
	)
}

func renderRD2Hero(page string) ui.Node {
	eyebrow := "Customer support acceleration"
	titleLead := "Ship faster replies without burning out your support team."
	titleAccent := "RelayDesk handles the first response, routing, and context handoff."
	body := "Customers get immediate answers. Agents get cleaner escalations. Leaders get a predictable path to lower resolution cost and higher CSAT."
	secondaryLabel := "See solutions"
	secondaryRoute := marketingCapabilitiesRoute

	switch page {
	case landingPageCapabilities:
		eyebrow = "What RelayDesk does"
		titleLead = "One service for triage, drafting, and human handoff."
		titleAccent = "Every conversation lands in the right lane with the right context."
		body = "RelayDesk classifies intent, prioritizes urgency, and drafts policy-safe responses while your team stays in control of final outcomes."
		secondaryLabel = "See pricing"
		secondaryRoute = marketingPricingRoute
	case landingPagePricing:
		eyebrow = "Simple pricing"
		titleLead = "Pay for outcomes, not tool sprawl."
		titleAccent = "RelayDesk scales with your queue volume and SLA targets."
		body = "Choose a plan that fits your support load today, then scale without rebuilding your workflows every quarter."
		secondaryLabel = "See solutions"
		secondaryRoute = marketingCapabilitiesRoute
	}

	return Article(
		Class("rd2-card rd2-hero-card rd2-reveal"),
		Span(Class("rd2-pill"), Text(eyebrow)),
		H1(
			Class("rd2-hero-title mt-6"),
			Span(Text(titleLead+" ")),
			Span(Class("rd2-hero-accent"), Text(titleAccent)),
		),
		P(
			Class("mt-6 max-w-[48rem] text-[1.02rem] leading-8 text-white/68"),
			Text(body),
		),
		Div(
			Class("mt-8 flex flex-wrap gap-3"),
			landingActionButton("Start with RelayDesk", chatRouteRoot, true),
			landingActionButton(secondaryLabel, secondaryRoute, false),
		),
		Div(
			Class("mt-8 grid gap-3 sm:grid-cols-3"),
			rd2MiniMetric("87%", "Auto-resolved before agent handoff"),
			rd2MiniMetric("< 2m", "Median first response time"),
			rd2MiniMetric("-38%", "Average ticket handling cost"),
		),
	)
}

func renderRD2QueueCard(page string) ui.Node {
	title := "Live queue pulse"
	body := "RelayDesk keeps the support floor moving by routing incoming requests by intent and urgency."
	laneA := "Billing & refunds"
	laneB := "Product setup"
	laneC := "Priority incident"

	switch page {
	case landingPageCapabilities:
		title = "Operational control deck"
		body = "Automation runs first-pass coverage while specialists step in only where judgment actually matters."
		laneA = "Policy-safe auto replies"
		laneB = "Account-based routing"
		laneC = "Escalation with context summary"
	case landingPagePricing:
		title = "Volume to value"
		body = "See how RelayDesk handles growth spikes without forcing overnight hiring cycles."
		laneA = "Off-hours queue coverage"
		laneB = "Team workload balancing"
		laneC = "SLA breach prevention"
	}

	return Article(
		Class("rd2-card rd2-queue-card rd2-reveal rd2-reveal-d1"),
		Div(
			Class("flex items-start justify-between gap-4"),
			Div(
				H2(Class("text-2xl font-semibold tracking-tight text-white"), Text(title)),
				P(Class("mt-2 text-sm leading-7 text-white/60"), Text(body)),
			),
			Span(Class("rd2-live-pill"), Text("Live")),
		),
		Div(Class("rd2-lane rd2-lane-a mt-6"),
			Span(Class("rd2-lane-dot"), nil),
			Div(
				P(Class("text-[0.72rem] uppercase tracking-[0.2em] text-white/45"), Text("Lane A")),
				P(Class("mt-1 text-sm font-medium text-white/90"), Text(laneA)),
			),
		),
		Div(Class("rd2-lane rd2-lane-b"),
			Span(Class("rd2-lane-dot"), nil),
			Div(
				P(Class("text-[0.72rem] uppercase tracking-[0.2em] text-white/45"), Text("Lane B")),
				P(Class("mt-1 text-sm font-medium text-white/90"), Text(laneB)),
			),
		),
		Div(Class("rd2-lane rd2-lane-c"),
			Span(Class("rd2-lane-dot"), nil),
			Div(
				P(Class("text-[0.72rem] uppercase tracking-[0.2em] text-white/45"), Text("Lane C")),
				P(Class("mt-1 text-sm font-medium text-white/90"), Text(laneC)),
			),
		),
		Div(
			Class("rd2-queue-foot mt-6"),
			Div(Class("rd2-queue-bar"), Div(Class("rd2-queue-bar-fill"), nil)),
			P(Class("mt-2 text-xs text-white/45"), Text("Queue stabilization trend over last 6 hours")),
		),
	)
}

func renderRD2Middle(page string) ui.Node {
	if page == landingPagePricing {
		return Section(
			Class("grid gap-4 lg:grid-cols-3"),
			rd2PlanCard("Starter", "$149/mo", "For teams starting with AI-assisted first response.", []string{
				"Up to 5,000 conversations/month",
				"Intent routing + response drafting",
				"Email and web chat coverage",
			}, false),
			rd2PlanCard("Growth", "$499/mo", "For support orgs with strict SLA and mixed queues.", []string{
				"Up to 30,000 conversations/month",
				"VIP routing and escalation rules",
				"Team performance analytics",
			}, true),
			rd2PlanCard("Scale", "Custom", "For enterprise operations with complex support workflows.", []string{
				"Unlimited volume",
				"Custom governance controls",
				"Dedicated success and onboarding",
			}, false),
		)
	}
	if page == landingPageCapabilities {
		return Section(
			Class("grid gap-4 lg:grid-cols-4"),
			rd2FeatureCard("Intent detection", "Requests are classified instantly so each conversation starts in the right queue."),
			rd2FeatureCard("Smart drafting", "RelayDesk generates answers in your brand voice using approved support guidance."),
			rd2FeatureCard("Human takeover", "Escalated threads include summary, customer sentiment, and recommended next action."),
			rd2FeatureCard("Performance visibility", "Track response quality, queue pressure, and automation impact in one view."),
		)
	}
	return Fragment(
		Section(
			Class("grid gap-4 lg:grid-cols-3"),
			rd2FeatureCard("Deflect repetitive tickets", "Automate common questions so your team can focus on nuanced, high-value conversations."),
			rd2FeatureCard("Protect customer experience", "Keep response quality consistent across peak volume and after-hours support."),
			rd2FeatureCard("Scale confidently", "Grow support capacity without multiplying headcount or adding brittle tooling."),
		),
		Section(
			Class("grid gap-4 lg:grid-cols-[1.2fr_0.8fr]"),
			rd2StoryCard("Support leaders get predictable operations", "RelayDesk turns queue chaos into clean lanes with clear ownership and measurable outcomes."),
			rd2StoryCard("Agents stay in control", "AI accelerates every handoff, but final customer decisions remain with your team."),
		),
	)
}

func renderRD2FinalCTA(page string) ui.Node {
	title := "Ready to run support with RelayDesk?"
	body := "Launch your workspace, connect channels, and start converting queue pressure into faster resolution."
	secondaryLabel := "See pricing"
	secondaryRoute := marketingPricingRoute
	if page == landingPagePricing {
		secondaryLabel = "See solutions"
		secondaryRoute = marketingCapabilitiesRoute
	}

	return Section(
		Class("rd2-card rd2-cta-card rd2-reveal rd2-reveal-d2"),
		H2(Class("text-3xl font-semibold tracking-tight text-white sm:text-4xl"), Text(title)),
		P(Class("mt-3 max-w-3xl text-base leading-8 text-white/64"), Text(body)),
		Div(
			Class("mt-7 flex flex-wrap gap-3"),
			landingActionButton("Launch RelayDesk", chatRouteRoot, true),
			landingActionButton(secondaryLabel, secondaryRoute, false),
		),
	)
}

func rd2MiniMetric(value, label string) ui.Node {
	return Div(
		Class("rd2-mini-metric"),
		P(Class("rd2-mini-value"), Text(value)),
		P(Class("rd2-mini-label"), Text(label)),
	)
}

func rd2FeatureCard(title, body string) ui.Node {
	return Article(
		Class("rd2-soft-card rd2-reveal"),
		H3(Class("text-xl font-semibold tracking-tight text-white"), Text(title)),
		P(Class("mt-3 text-sm leading-7 text-white/64"), Text(body)),
	)
}

func rd2StoryCard(title, body string) ui.Node {
	return Article(
		Class("rd2-soft-card rd2-reveal"),
		P(Class("text-[0.7rem] uppercase tracking-[0.24em] text-white/45"), Text("Outcome")),
		H3(Class("mt-2 text-xl font-semibold tracking-tight text-white"), Text(title)),
		P(Class("mt-3 text-sm leading-7 text-white/66"), Text(body)),
	)
}

func rd2PlanCard(name, price, body string, items []string, featured bool) ui.Node {
	return Article(
		Class(ClassNames(
			"rd2-soft-card rd2-reveal",
			When(featured, "rd2-plan-featured"),
		)),
		P(Class("text-[0.68rem] uppercase tracking-[0.24em] text-white/46"), Text(name)),
		P(Class("mt-3 text-4xl font-semibold tracking-tight text-white"), Text(price)),
		P(Class("mt-3 text-sm leading-7 text-white/62"), Text(body)),
		Ul(
			Class("mt-4 flex list-none flex-col gap-2 p-0"),
			Map(items, func(item string) ui.Node {
				return Li(
					Class("rd2-bullet"),
					Span(Class("rd2-bullet-dot"), nil),
					Span(Text(item)),
				)
			}),
		),
	)
}

func landingActionButton(label, targetPath string, primary bool) ui.Node {
	return A(
		Class(ClassNames(
			"rd2-btn",
			When(primary, "rd2-btn-primary"),
			When(!primary, "rd2-btn-secondary"),
		)),
		Href(targetPath),
		OnClick(landingNavigateHandler(targetPath)),
		Text(label),
	)
}

func landingNavigateHandler(targetPath string) func(ui.Event) {
	normalizedTarget := strings.TrimSpace(targetPath)
	return func(e ui.Event) {
		jsEvent := e.JSValue()
		if jsEvent.Truthy() {
			if jsEvent.Get("defaultPrevented").Bool() {
				return
			}
			if jsEvent.Get("button").Int() != 0 {
				return
			}
			if jsEvent.Get("metaKey").Bool() || jsEvent.Get("ctrlKey").Bool() || jsEvent.Get("shiftKey").Bool() || jsEvent.Get("altKey").Bool() {
				return
			}
		}

		e.PreventDefault()
		if normalizedTarget == "" {
			return
		}
		currentPath := strings.TrimSpace(router.GetCurrentPath())
		if currentPath == normalizedTarget {
			return
		}
		if normalizedTarget == authLandingRoute && currentPath == marketingHomeRoute {
			return
		}
		router.Navigate(normalizedTarget)
	}
}
