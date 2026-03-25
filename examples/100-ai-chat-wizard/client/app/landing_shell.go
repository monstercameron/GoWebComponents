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

func renderLandingShell(intl i18n.Runtime, view appViewState, auth authSessionController) ui.Node {
	isSignup := view.AuthMode == authModeSignup
	submitLabel := intl.T(chatI18nNamespace, "auth.signIn")
	formTitle := intl.T(chatI18nNamespace, "auth.loginTitle")
	formBody := intl.T(chatI18nNamespace, "auth.loginBody")
	switchLabel := intl.T(chatI18nNamespace, "auth.switchToSignup")
	if isSignup {
		submitLabel = intl.T(chatI18nNamespace, "auth.createAccount")
		formTitle = intl.T(chatI18nNamespace, "auth.signupTitle")
		formBody = intl.T(chatI18nNamespace, "auth.signupBody")
		switchLabel = intl.T(chatI18nNamespace, "auth.switchToLogin")
	}

	page := landingPageForPath(view.CurrentPath)

	return Div(
		Class("min-h-screen w-full overflow-y-auto bg-[radial-gradient(circle_at_top,_rgba(25,195,125,0.18),_transparent_24%),radial-gradient(circle_at_80%_10%,_rgba(245,158,11,0.12),_transparent_20%),linear-gradient(180deg,#0e1513_0%,#111827_38%,#09110f_100%)] text-white"),
		Header(
			Class("sticky top-0 z-20 border-b border-white/8 bg-[#08100f]/72 backdrop-blur-xl"),
			Div(
				Class("mx-auto flex w-full max-w-7xl items-center justify-between gap-4 px-6 py-4"),
				Div(Class("flex items-center gap-3"),
					Div(Class("flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-[#19c37d] via-[#8df5cf] to-[#fbbf24] text-sm font-black tracking-[0.18em] text-[#04110c] shadow-[0_18px_40px_rgba(25,195,125,0.24)]"), Text(assistantBadgeText)),
					Div(
						P(Class("text-sm font-semibold tracking-[0.22em] text-white/90"), Text(appBrandName)),
						P(Class("text-xs uppercase tracking-[0.26em] text-white/35"), Text("Go/WASM workspace")),
					),
				),
				Div(Class("hidden items-center gap-6 md:flex"),
					landingNavLink(view.CurrentPath, authLandingRoute, "Home"),
					landingNavLink(view.CurrentPath, marketingCapabilitiesRoute, "Capabilities"),
					landingNavLink(view.CurrentPath, marketingPricingRoute, "Pricing"),
				),
			),
		),
		Main(
			Class("mx-auto flex w-full max-w-7xl flex-col gap-16 px-6 py-10 pb-20"),
			Section(
				Class("grid items-stretch gap-8 lg:grid-cols-[1.08fr_0.92fr]"),
				renderLandingHero(page),
				renderLandingAuthCard(intl, view, auth, isSignup, submitLabel, formTitle, formBody, switchLabel),
			),
			renderLandingBody(page),
			Section(
				Class("rounded-[2.1rem] border border-[#8df5cf]/14 bg-[linear-gradient(135deg,rgba(12,17,16,0.9),rgba(9,38,28,0.82))] p-8 shadow-[0_24px_80px_rgba(0,0,0,0.34)] sm:p-10"),
				P(Class("text-xs font-semibold uppercase tracking-[0.26em] text-[#8df5cf]"), Text("Next")),
				H2(Class("mt-3 text-3xl font-semibold tracking-tight text-white"), Text("Separate marketing routes now run on the browser router.")),
				P(Class("mt-4 max-w-3xl text-sm leading-7 text-white/62 sm:text-base"), Text("Each page is now addressable as its own route. Use the links below or jump straight to chat when you are ready.")),
				Div(Class("mt-7 flex flex-wrap gap-3"),
					A(
						Class("rounded-2xl bg-white px-5 py-3 text-sm font-semibold text-[#0b1712] transition-colors hover:bg-[#d9fff0]"),
						Href(authLandingRoute),
						OnClick(landingNavigateHandler(authLandingRoute)),
						Text("Sign in"),
					),
					A(
						Class("rounded-2xl border border-white/12 px-5 py-3 text-sm font-semibold text-white/82 transition-colors hover:bg-white/5"),
						Href(marketingCapabilitiesRoute),
						OnClick(landingNavigateHandler(marketingCapabilitiesRoute)),
						Text("Capabilities"),
					),
					A(
						Class("rounded-2xl border border-white/12 px-5 py-3 text-sm font-semibold text-white/82 transition-colors hover:bg-white/5"),
						Href(marketingPricingRoute),
						OnClick(landingNavigateHandler(marketingPricingRoute)),
						Text("Pricing"),
					),
				),
			),
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
			"text-sm transition-colors",
			When(isActive, "text-white"),
			When(!isActive, "text-white/60 hover:text-white"),
		)),
		Href(targetPath),
		OnClick(landingNavigateHandler(targetPath)),
		Text(label),
	)
}

func renderLandingHero(page string) ui.Node {
	eyebrow := "Multi-route shell"
	title := "The homepage and marketing pages now run as separate browser routes."
	body := "RelayDesk keeps marketing, auth, and the private chat workspace in one Go/WASM client while still letting each public page have its own URL."
	primaryLabel := "Open capabilities"
	primaryHref := marketingCapabilitiesRoute
	secondaryLabel := "View pricing"
	secondaryHref := marketingPricingRoute
	metricA := landingMetricCard("100%", "GWC-owned UI", "Public pages, auth, and workspace are rendered from the same runtime.")
	metricB := landingMetricCard("gRPC", "Live tunnel", "Sign in, session refresh, and chat stay on one transport path.")
	metricC := landingMetricCard("SQL", "Model catalog", "Provider, model, and pricing data remain runtime-backed.")

	switch page {
	case landingPageCapabilities:
		eyebrow = "Capabilities route"
		title = "Capabilities has its own route and no longer depends on hash fragments."
		body = "This page focuses on shipped runtime behavior: auth, provider switching, routed thread history, and canvas collaboration."
		primaryLabel = "View pricing"
		primaryHref = marketingPricingRoute
		secondaryLabel = "Back home"
		secondaryHref = authLandingRoute
	case landingPagePricing:
		eyebrow = "Pricing route"
		title = "Pricing is now a first-class browser route."
		body = "Use this page for plan framing and value communication, then move into auth and chat from a stable top-level URL."
		primaryLabel = "View capabilities"
		primaryHref = marketingCapabilitiesRoute
		secondaryLabel = "Back home"
		secondaryHref = authLandingRoute
	}

	return Div(
		Class("relative overflow-hidden rounded-[2.25rem] border border-white/10 bg-[linear-gradient(145deg,rgba(11,24,20,0.96),rgba(17,24,39,0.88))] p-8 shadow-[0_30px_90px_rgba(0,0,0,0.45)] sm:p-10"),
		Div(Class("absolute inset-x-8 top-0 h-px bg-gradient-to-r from-transparent via-[#8df5cf]/60 to-transparent")),
		Div(Class("inline-flex items-center gap-2 rounded-full border border-[#8df5cf]/18 bg-[#8df5cf]/8 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.28em] text-[#9cf9d6]"),
			Span(Class("h-2 w-2 rounded-full bg-[#19c37d] shadow-[0_0_16px_rgba(25,195,125,0.8)]")),
			Text(eyebrow),
		),
		H1(Class("mt-6 max-w-4xl text-5xl font-black leading-[0.92] tracking-[-0.05em] text-white sm:text-6xl"),
			Text(title),
		),
		P(Class("mt-6 max-w-2xl text-base leading-8 text-white/65 sm:text-lg"), Text(body)),
		Div(Class("mt-8 flex flex-wrap gap-3"),
			A(
				Class("rounded-2xl bg-[#19c37d] px-5 py-3 text-sm font-semibold text-[#052516] transition-colors hover:bg-[#31de90]"),
				Href(primaryHref),
				OnClick(landingNavigateHandler(primaryHref)),
				Text(primaryLabel),
			),
			A(
				Class("rounded-2xl border border-white/10 px-5 py-3 text-sm font-semibold text-white/82 transition-colors hover:bg-white/5"),
				Href(secondaryHref),
				OnClick(landingNavigateHandler(secondaryHref)),
				Text(secondaryLabel),
			),
		),
		Div(Class("mt-10 grid gap-4 sm:grid-cols-3"),
			metricA,
			metricB,
			metricC,
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

func renderLandingAuthCard(intl i18n.Runtime, view appViewState, auth authSessionController, isSignup bool, submitLabel, formTitle, formBody, switchLabel string) ui.Node {
	return Div(
		Class("rounded-[2.25rem] border border-white/10 bg-[linear-gradient(180deg,rgba(12,17,16,0.94),rgba(17,24,39,0.92))] p-8 shadow-[0_30px_90px_rgba(0,0,0,0.45)] sm:p-9"),
		P(Class("text-xs font-semibold uppercase tracking-[0.24em] text-[#8df5cf]"), Text("Workspace access")),
		H2(Class("mt-3 text-3xl font-semibold tracking-tight text-white"), Text(formTitle)),
		P(Class("mt-3 text-sm leading-6 text-white/58"), Text(formBody)),
		Div(Class("mt-7 flex flex-col gap-4"),
			If(isSignup,
				Div(Class("flex flex-col gap-1.5"),
					Label(Class("text-xs font-medium uppercase tracking-[0.18em] text-white/45"), Text(intl.T(chatI18nNamespace, "auth.displayName"))),
					Input(
						ID(idAuthNameInput),
						Type("text"),
						Class("w-full rounded-2xl border border-white/12 bg-[#202b28] px-4 py-3 text-sm text-white outline-none transition-colors focus:border-[#19c37d]/55"),
						Placeholder(intl.T(chatI18nNamespace, "auth.displayNamePlaceholder")),
						Value(view.AuthDisplayName),
						OnInput(auth.HandleDisplayNameInput),
					),
				),
			),
			Div(Class("flex flex-col gap-1.5"),
				Label(Class("text-xs font-medium uppercase tracking-[0.18em] text-white/45"), Text(intl.T(chatI18nNamespace, "auth.email"))),
				Input(
					ID(idAuthEmailInput),
					Type("email"),
					Class("w-full rounded-2xl border border-white/12 bg-[#202b28] px-4 py-3 text-sm text-white outline-none transition-colors focus:border-[#19c37d]/55"),
					Placeholder(intl.T(chatI18nNamespace, "auth.emailPlaceholder")),
					Value(view.AuthEmail),
					OnInput(auth.HandleEmailInput),
				),
			),
			Div(Class("flex flex-col gap-1.5"),
				Label(Class("text-xs font-medium uppercase tracking-[0.18em] text-white/45"), Text(intl.T(chatI18nNamespace, "auth.password"))),
				Input(
					ID(idAuthPasswordInput),
					Type("password"),
					Class("w-full rounded-2xl border border-white/12 bg-[#202b28] px-4 py-3 text-sm text-white outline-none transition-colors focus:border-[#19c37d]/55"),
					Placeholder(intl.T(chatI18nNamespace, "auth.passwordPlaceholder")),
					Value(view.AuthPassword),
					OnInput(auth.HandlePasswordInput),
					OnKeyDown(auth.HandlePasswordKey),
				),
				P(Class("text-xs leading-5 text-white/35"), Text(intl.T(chatI18nNamespace, "auth.passwordHelp"))),
			),
			If(view.AuthError != "",
				Div(Class("rounded-2xl border border-red-400/20 bg-red-500/10 px-4 py-3 text-sm leading-6 text-red-100"), Text(view.AuthError)),
			),
			Button(
				Class(ClassNames(
					"mt-1 w-full rounded-2xl bg-[#19c37d] px-4 py-3 text-sm font-semibold text-[#052516] transition-colors",
					When(!view.AuthSubmitting, "hover:bg-[#31de90]"),
					When(view.AuthSubmitting, "cursor-progress opacity-80"),
				)),
				DisabledIf(view.AuthSubmitting || !view.GRPCReady),
				OnClick(auth.HandleSubmit),
				Text(submitLabel),
			),
			Button(
				Class("w-full rounded-2xl border border-white/10 px-4 py-3 text-sm text-white/72 transition-colors hover:bg-white/5 hover:text-white"),
				OnClick(auth.HandleModeToggle),
				Text(switchLabel),
			),
		),
	)
}

func renderLandingBody(page string) ui.Node {
	switch page {
	case landingPageCapabilities:
		return Section(
			Class("grid gap-5 lg:grid-cols-3"),
			landingFeatureCard("Runtime-owned auth", "Sessions, logout, and refresh all stay in the same wasm surface, so route state and UI state stay aligned."),
			landingFeatureCard("Provider switching", "OpenAI, Anthropic, and Cerebras live behind one SQL-backed model catalog with capability-aware dropdowns."),
			landingFeatureCard("Canvas + thread flow", "Deep thread routes, canvas workspaces, and the chat shell share one router instead of bouncing through detached pages."),
		)
	case landingPagePricing:
		return Section(
			Class("grid gap-5 lg:grid-cols-[1.1fr_0.9fr_0.9fr]"),
			landingTierCard("Build", "For local iteration", "Run the wasm client, seed dev accounts, and test provider switching without leaving the app shell."),
			landingTierCard("Operate", "For long-lived threads", "Conversation history, settings, and model preferences persist per user instead of leaking across a shared dev session."),
			landingTierCard("Extend", "For product experiments", "The same client shell can carry landing routes, auth, routed threads, and canvas editing without reintroducing server HTML."),
		)
	default:
		return Fragment(
			Section(
				Class("grid gap-5 lg:grid-cols-3"),
				landingFeatureCard("Runtime-owned auth", "Sessions, logout, and refresh all stay in the same wasm surface, so route state and UI state stay aligned."),
				landingFeatureCard("Provider switching", "OpenAI, Anthropic, and Cerebras live behind one SQL-backed model catalog with capability-aware dropdowns."),
				landingFeatureCard("Canvas + thread flow", "Deep thread routes, canvas workspaces, and the chat shell share one router instead of bouncing through detached pages."),
			),
			Section(
				Class("grid gap-5 lg:grid-cols-[1.1fr_0.9fr_0.9fr]"),
				landingTierCard("Build", "For local iteration", "Run the wasm client, seed dev accounts, and test provider switching without leaving the app shell."),
				landingTierCard("Operate", "For long-lived threads", "Conversation history, settings, and model preferences persist per user instead of leaking across a shared dev session."),
				landingTierCard("Extend", "For product experiments", "The same client shell can carry landing routes, auth, routed threads, and canvas editing without reintroducing server HTML."),
			),
		)
	}
}

func landingMetricCard(value, label, body string) ui.Node {
	return Div(
		Class("rounded-3xl border border-white/10 bg-white/[0.045] p-5 backdrop-blur-sm"),
		P(Class("text-3xl font-black tracking-[-0.05em] text-white"), Text(value)),
		P(Class("mt-2 text-sm font-semibold uppercase tracking-[0.18em] text-[#8df5cf]"), Text(label)),
		P(Class("mt-3 text-sm leading-6 text-white/56"), Text(body)),
	)
}

func landingFeatureCard(title, body string) ui.Node {
	return Article(
		Class("rounded-[1.8rem] border border-white/10 bg-[linear-gradient(180deg,rgba(255,255,255,0.04),rgba(255,255,255,0.02))] p-6 shadow-[0_16px_45px_rgba(0,0,0,0.24)]"),
		Div(Class("flex h-10 w-10 items-center justify-center rounded-2xl bg-[#19c37d]/14 text-[#8df5cf]"), Text("o")),
		H3(Class("mt-5 text-xl font-semibold tracking-tight text-white"), Text(title)),
		P(Class("mt-3 text-sm leading-7 text-white/58"), Text(body)),
	)
}

func landingTierCard(title, eyebrow, body string) ui.Node {
	return Article(
		Class("rounded-[1.9rem] border border-white/10 bg-[linear-gradient(180deg,rgba(12,17,16,0.88),rgba(17,24,39,0.82))] p-6 shadow-[0_18px_50px_rgba(0,0,0,0.28)]"),
		P(Class("text-xs font-semibold uppercase tracking-[0.24em] text-[#8df5cf]"), Text(eyebrow)),
		H3(Class("mt-3 text-2xl font-semibold tracking-tight text-white"), Text(title)),
		P(Class("mt-4 text-sm leading-7 text-white/58"), Text(body)),
	)
}
