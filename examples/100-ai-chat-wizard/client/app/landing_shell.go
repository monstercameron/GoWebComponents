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
		Class("landing-root min-h-screen w-full overflow-x-hidden overflow-y-auto text-white"),
		Div(Class("landing-backdrop-grid"), nil),
		Div(Class("landing-orb landing-orb-a"), nil),
		Div(Class("landing-orb landing-orb-b"), nil),
		Div(Class("landing-orb landing-orb-c"), nil),
		Header(
			Class("landing-shell landing-header sticky top-0 z-30 border-b border-white/10"),
			Div(
				Class("mx-auto flex w-full max-w-[82rem] items-center justify-between gap-4 px-6 py-4"),
				Div(
					Class("flex items-center gap-3"),
					Div(
						Class("landing-brand-badge"),
						Text("RD"),
					),
					Div(
						P(Class("text-[0.96rem] font-semibold tracking-[0.08em] text-white"), Text("RelayDesk")),
						P(Class("text-[0.64rem] uppercase tracking-[0.28em] text-white/50"), Text("Chat Service")),
					),
				),
				Div(
					Class("hidden items-center gap-7 md:flex"),
					landingNavLink(view.CurrentPath, authLandingRoute, "Home"),
					landingNavLink(view.CurrentPath, marketingCapabilitiesRoute, "Solutions"),
					landingNavLink(view.CurrentPath, marketingPricingRoute, "Plans"),
				),
				Div(
					Class("flex items-center gap-2"),
					landingActionButton("Launch chat", chatRouteRoot, false),
				),
			),
		),
		Main(
			Class("landing-shell relative z-10 mx-auto flex w-full max-w-[82rem] flex-col gap-12 px-6 pb-20 pt-10"),
			Section(
				Class("grid items-stretch gap-8 lg:grid-cols-[1.06fr_0.94fr]"),
				renderLandingHero(page),
				renderLandingShowcase(page),
			),
			renderLandingBody(page),
			renderLandingFinalCTA(page),
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

func renderLandingHero(page string) ui.Node {
	eyebrow := "RelayDesk chat service"
	titleLead := "Support teams close more conversations in less time."
	titleAccent := "AI agents that answer, route, and hand off with context."
	body := "RelayDesk gives your customers instant replies while your team stays in control. Every message keeps its history, priority, and ownership in one clean workspace."
	primaryLabel := "Start free in RelayDesk"
	primaryRoute := chatRouteRoot
	secondaryLabel := "Explore solutions"
	secondaryRoute := marketingCapabilitiesRoute

	switch page {
	case landingPageCapabilities:
		eyebrow = "Solution highlights"
		titleLead = "Every inbox gets an expert copilot."
		titleAccent = "Fast replies, clean escalations, and zero context loss."
		body = "RelayDesk triages incoming questions, drafts responses with your brand voice, and routes edge cases to the right teammate before SLAs slip."
		secondaryLabel = "View plans"
		secondaryRoute = marketingPricingRoute
	case landingPagePricing:
		eyebrow = "Simple plans"
		titleLead = "Pricing that scales with your support volume."
		titleAccent = "No hidden seats, no mystery overages."
		body = "Pick the plan that matches your queue size and response targets. Every tier includes AI routing, smart drafting, and full human handoff."
		secondaryLabel = "See solutions"
		secondaryRoute = marketingCapabilitiesRoute
	}

	return Div(
		Class("landing-glass-card landing-reveal"),
		Div(
			Class("landing-pill"),
			Text(eyebrow),
		),
		H1(
			Class("landing-hero-title mt-6 max-w-3xl"),
			Span(Text(titleLead+" ")),
			Span(Class("landing-hero-emphasis"), Text(titleAccent)),
		),
		P(
			Class("mt-6 max-w-2xl text-[1.03rem] leading-8 text-white/68"),
			Text(body),
		),
		Div(
			Class("mt-8 flex flex-wrap items-center gap-3"),
			landingActionButton(primaryLabel, primaryRoute, true),
			landingActionButton(secondaryLabel, secondaryRoute, false),
		),
		Div(
			Class("mt-8 flex flex-wrap gap-2.5"),
			Span(Class("landing-chip"), Text("24/7 AI response coverage")),
			Span(Class("landing-chip"), Text("Smart intent routing")),
			Span(Class("landing-chip"), Text("Human takeover in one click")),
		),
	)
}

func renderLandingShowcase(page string) ui.Node {
	cardTitle := "Live queue impact"
	cardBody := "See how RelayDesk handles urgent requests while keeping your team focused on high-value conversations."
	topMetricValue := "84%"
	topMetricLabel := "Handled before escalation"
	secondaryMetricValue := "< 2 min"
	secondaryMetricLabel := "Median first response"
	feedItems := []string{
		"Billing question auto-resolved with policy-safe answer.",
		"Enterprise setup request routed to the named account pod.",
		"Priority outage ticket escalated with timeline summary.",
	}

	switch page {
	case landingPageCapabilities:
		cardTitle = "Automation + control"
		cardBody = "RelayDesk balances speed and oversight so your team can trust every AI-generated message."
		topMetricValue = "99.2%"
		topMetricLabel = "Policy-compliant drafts"
		secondaryMetricValue = "3x"
		secondaryMetricLabel = "Faster routing decisions"
		feedItems = []string{
			"VIP customer detected and assigned to dedicated support lane.",
			"Refund request answered with approved policy language.",
			"Technical question enriched with context before handoff.",
		}
	case landingPagePricing:
		cardTitle = "Predictable growth"
		cardBody = "Increase coverage without multiplying headcount or adding brittle workflow tools."
		topMetricValue = "2.4x"
		topMetricLabel = "More conversations per rep"
		secondaryMetricValue = "-41%"
		secondaryMetricLabel = "Lower resolution cost"
		feedItems = []string{
			"Night queue covered without adding overnight staffing.",
			"Recurring questions shifted to automated resolution.",
			"Agents spend more time on renewals and retention.",
		}
	}

	return Article(
		Class("landing-glass-card landing-reveal landing-reveal-delay-1"),
		Div(
			Class("flex items-start justify-between gap-4"),
			Div(
				P(Class("text-[0.72rem] uppercase tracking-[0.26em] text-white/44"), Text("Dashboard")),
				H2(Class("mt-2 text-2xl font-semibold tracking-tight text-white"), Text(cardTitle)),
			),
			Span(Class("landing-live-dot"), Text("Live")),
		),
		P(
			Class("mt-3 text-sm leading-7 text-white/60"),
			Text(cardBody),
		),
		Div(
			Class("mt-6 grid gap-3 sm:grid-cols-2"),
			landingKPI(topMetricValue, topMetricLabel),
			landingKPI(secondaryMetricValue, secondaryMetricLabel),
		),
		Ul(
			Class("mt-6 flex list-none flex-col gap-2.5 p-0"),
			Map(feedItems, func(item string) ui.Node {
				return Li(
					Class("landing-feed-item"),
					Span(Class("landing-feed-pulse"), nil),
					Span(Text(item)),
				)
			}),
		),
	)
}

func renderLandingBody(page string) ui.Node {
	switch page {
	case landingPageCapabilities:
		return Fragment(
			Section(
				Class("grid gap-4 lg:grid-cols-3"),
				landingFeatureCard("Intent routing", "Incoming chats are classified instantly so each request lands in the right queue."),
				landingFeatureCard("Voice-safe drafting", "RelayDesk writes in your tone and only uses approved policy and product context."),
				landingFeatureCard("Agent assist", "Human reps see summaries, suggested replies, and next-step prompts before they type."),
				landingFeatureCard("Escalation paths", "High-risk conversations move to specialists with full thread context attached."),
				landingFeatureCard("Priority workflows", "VIP and renewal accounts trigger dedicated handling rules automatically."),
				landingFeatureCard("Insight loops", "Recurring issues surface as themes so support leaders can tune flows weekly."),
			),
			Section(
				Class("grid gap-4 lg:grid-cols-2"),
				landingStoryCard("For support leads", "Track queue pressure in real time, control which issues are automated, and coach with clearer handoff history."),
				landingStoryCard("For operations teams", "Tune routing and policies in one place without rebuilding your stack or retraining agents every sprint."),
			),
		)
	case landingPagePricing:
		return Fragment(
			Section(
				Class("grid gap-4 lg:grid-cols-3"),
				landingPlanCard("Starter", "$149/mo", "For growing teams that need immediate coverage and consistent reply quality.", []string{
					"Up to 5,000 monthly conversations",
					"AI triage and first-response drafting",
					"Email and chat channel support",
				}, false),
				landingPlanCard("Growth", "$499/mo", "For teams running multi-channel support with strict response windows.", []string{
					"Up to 30,000 monthly conversations",
					"Advanced routing and VIP workflows",
					"Team analytics and SLA dashboards",
				}, true),
				landingPlanCard("Scale", "Custom", "For enterprise operations that need custom controls and dedicated success support.", []string{
					"Unlimited conversation volume",
					"Custom policy and security controls",
					"Dedicated onboarding and QBRs",
				}, false),
			),
			Section(
				Class("grid gap-4 lg:grid-cols-3"),
				landingStoryCard("No hidden usage traps", "You always see conversation volume, automation rates, and forecasted spend before the billing cycle closes."),
				landingStoryCard("Fast onboarding", "Most teams ship their first live RelayDesk flows in days, not quarters."),
				landingStoryCard("Human-first design", "AI helps at speed, but your agents stay in control of final responses and customer outcomes."),
			),
		)
	default:
		return Fragment(
			Section(
				Class("grid gap-4 lg:grid-cols-3"),
				landingFeatureCard("Resolve faster", "AI drafts complete answers from your approved knowledge so customers wait less."),
				landingFeatureCard("Route smarter", "Intent and urgency scoring direct each thread to the right queue before backlog grows."),
				landingFeatureCard("Handoff cleanly", "When humans step in, they inherit a structured summary, not a messy transcript."),
			),
			Section(
				Class("grid gap-4 lg:grid-cols-3"),
				landingFlowCard("1", "Connect channels", "Bring web chat and support inboxes into one RelayDesk queue."),
				landingFlowCard("2", "Set policy guardrails", "Define tone, escalation rules, and approved answer boundaries."),
				landingFlowCard("3", "Go live with confidence", "Track outcomes, tune prompts, and scale without service drops."),
			),
			Section(
				Class("grid gap-4 lg:grid-cols-2"),
				landingStoryCard("Support teams report faster turnarounds", "\"We cleared our weekend backlog before lunch on Monday and still improved quality scores.\""),
				landingStoryCard("Leadership gets clear performance signals", "\"RelayDesk showed exactly where automation worked and where human coaching moved the needle.\""),
			),
		)
	}
}

func renderLandingFinalCTA(page string) ui.Node {
	title := "Ready to run customer support at RelayDesk speed?"
	body := "Launch your workspace, connect channels, and start resolving more requests with less manual drag."
	secondaryLabel := "See plans"
	secondaryRoute := marketingPricingRoute
	if page == landingPagePricing {
		secondaryLabel = "See solutions"
		secondaryRoute = marketingCapabilitiesRoute
	}

	return Section(
		Class("landing-glass-card landing-reveal landing-reveal-delay-2"),
		H2(Class("text-3xl font-semibold tracking-tight text-white sm:text-4xl"), Text(title)),
		P(Class("mt-3 max-w-3xl text-base leading-8 text-white/64"), Text(body)),
		Div(
			Class("mt-7 flex flex-wrap gap-3"),
			landingActionButton("Launch RelayDesk", chatRouteRoot, true),
			landingActionButton(secondaryLabel, secondaryRoute, false),
		),
	)
}

func landingActionButton(label, targetPath string, primary bool) ui.Node {
	return A(
		Class(ClassNames(
			"landing-btn",
			When(primary, "landing-btn-primary"),
			When(!primary, "landing-btn-secondary"),
		)),
		Href(targetPath),
		OnClick(landingNavigateHandler(targetPath)),
		Text(label),
	)
}

func landingKPI(value, label string) ui.Node {
	return Div(
		Class("landing-kpi-card"),
		P(Class("landing-kpi-value"), Text(value)),
		P(Class("landing-kpi-label"), Text(label)),
	)
}

func landingFeatureCard(title, body string) ui.Node {
	return Article(
		Class("landing-soft-card landing-reveal"),
		P(Class("text-[0.68rem] uppercase tracking-[0.25em] text-white/40"), Text("Feature")),
		H3(Class("mt-2 text-xl font-semibold tracking-tight text-white"), Text(title)),
		P(Class("mt-3 text-sm leading-7 text-white/62"), Text(body)),
	)
}

func landingFlowCard(step, title, body string) ui.Node {
	return Article(
		Class("landing-soft-card landing-reveal"),
		Div(
			Class("flex items-center gap-3"),
			Span(Class("landing-step-dot"), Text(step)),
			H3(Class("text-lg font-semibold text-white"), Text(title)),
		),
		P(Class("mt-3 text-sm leading-7 text-white/62"), Text(body)),
	)
}

func landingStoryCard(title, body string) ui.Node {
	return Article(
		Class("landing-soft-card landing-reveal"),
		H3(Class("text-lg font-semibold tracking-tight text-white"), Text(title)),
		P(Class("mt-3 text-sm leading-7 text-white/66"), Text(body)),
	)
}

func landingPlanCard(name, price, body string, bullets []string, highlighted bool) ui.Node {
	return Article(
		Class(ClassNames(
			"landing-soft-card landing-reveal",
			When(highlighted, "landing-plan-highlight"),
		)),
		P(Class("text-[0.7rem] uppercase tracking-[0.26em] text-white/44"), Text(name)),
		P(Class("mt-3 text-4xl font-semibold tracking-tight text-white"), Text(price)),
		P(Class("mt-3 text-sm leading-7 text-white/66"), Text(body)),
		Ul(
			Class("mt-4 flex list-none flex-col gap-2 p-0"),
			Map(bullets, func(item string) ui.Node {
				return Li(
					Class("landing-feed-item"),
					Span(Class("landing-feed-pulse"), nil),
					Span(Text(item)),
				)
			}),
		),
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
