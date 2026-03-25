//go:build js && wasm

package app

import (
	"fmt"
	"syscall/js"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

type authLoadingShellProps struct {
	View appViewState
}

// renderAuthLoadingShell keeps the root route stable while the gRPC bridge and
// persisted auth token are being resolved. Must be mounted via ui.Component.
func renderAuthLoadingShell(parseProps authLoadingShellProps) ui.Node {
	parseProgress := ui.UseState(0)
	parsePhase := ui.UseState("loading")

	// Loading phase: increment progress by 1 every 38 ms, then pause 450 ms before spinning.
	ui.UseEffect(func() func() {
		if parsePhase.Get() != "loading" {
			return nil
		}
		parseG := js.Global()
		isParseDone := false
		var parseIntervalID js.Value
		parseCb := js.FuncOf(func(_ js.Value, _ []js.Value) interface{} {
			if isParseDone {
				return nil
			}
			parseProgress.Update(func(parsePrev int) int {
				parseNext := parsePrev + 1
				if parseNext >= 100 {
					parseG.Call("clearInterval", parseIntervalID)
					// switch to spinner after a short pause
					parseTransition := js.FuncOf(func(_ js.Value, _ []js.Value) interface{} {
						if !isParseDone {
							parsePhase.Set("spinning")
							parseProgress.Set(0)
						}
						return nil
					})
					parseG.Call("setTimeout", parseTransition, 450)
					return 100
				}
				return parseNext
			})
			return nil
		})
		parseIntervalID = parseG.Call("setInterval", parseCb, 38)
		return func() {
			isParseDone = true
			parseG.Call("clearInterval", parseIntervalID)
			parseCb.Release()
		}
	}, parsePhase.Get())

	// Spinning phase: stay for 2.2 s then restart the loading bar.
	ui.UseEffect(func() func() {
		if parsePhase.Get() != "spinning" {
			return nil
		}
		parseG2 := js.Global()
		isParseDone2 := false
		parseRestart := js.FuncOf(func(_ js.Value, _ []js.Value) interface{} {
			if !isParseDone2 {
				parsePhase.Set("loading")
				parseProgress.Set(0)
			}
			return nil
		})
		parseTimerID := parseG2.Call("setTimeout", parseRestart, 2200)
		return func() {
			isParseDone2 = true
			parseG2.Call("clearTimeout", parseTimerID)
			parseRestart.Release()
		}
	}, parsePhase.Get())

	parsePct := parseProgress.Get()
	isLoading := parsePhase.Get() == "loading"

	parseDisplayPct := fmt.Sprintf("%d%%", parsePct)
	if !isLoading {
		parseDisplayPct = "100%"
	}

	var parseBody ui.Node
	if isLoading {
		parseBody = Div(
			Class("space-y-4"),
			Div(
				Class("relative h-[2px] overflow-hidden rounded-full bg-white/[0.08]"),
				Div(
					Class("absolute inset-y-0 left-0 rounded-full bg-gradient-to-r from-cyan-400/70 via-cyan-300 to-sky-300 transition-all duration-300"),
					Style(map[string]string{"width": fmt.Sprintf("%d%%", parsePct)}),
				),
			),
			Div(
				Class("flex items-center justify-between text-sm text-white/35"),
				Span(Text("Loading")),
				Span(Class("text-[11px] uppercase tracking-[0.18em]"), Text("Please wait")),
			),
		)
	} else {
		parseBody = Div(
			Class("flex min-h-[84px] items-center justify-between"),
			Div(
				Div(Class("text-sm text-white/[0.38]"), Text("Finalizing")),
				Div(Class("mt-2 text-base font-medium text-white/[0.88]"), Text("Almost ready")),
			),
			Div(
				Class("relative h-10 w-10"),
				Div(Class("absolute inset-0 rounded-full border border-white/[0.08]"), nil),
				Div(Class("absolute inset-0 animate-spin rounded-full border-2 border-transparent border-t-cyan-300/80 border-r-cyan-200/60"), nil),
			),
		)
	}

	return Div(
		Class("flex min-h-screen items-center justify-center bg-[#020617] px-6 text-white"),
		Div(
			Class("w-full max-w-lg rounded-[32px] border border-white/[0.06] bg-white/[0.02] p-8 shadow-[0_20px_80px_rgba(0,0,0,0.55)] backdrop-blur-md"),
			Div(
				Class("mb-8 flex items-center justify-between"),
				Div(
					Div(Class("text-[10px] uppercase tracking-[0.35em] text-white/35"), Text("RelayDesk")),
					Div(Class("mt-3 text-xl font-medium tracking-tight text-white/90"), Text("Preparing interface")),
				),
				Div(Class("text-sm tabular-nums text-white/40"), Text(parseDisplayPct)),
			),
			parseBody,
		),
	)
}

// renderAuthShell renders the login/signup/reset/update-password page matching the RelayDesk brand.
func renderAuthShell(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController) ui.Node {
	if parseView.AuthMode == authModeUpdatePassword {
		return renderAuthUpdatePasswordShell(parseAuth)
	}
	if parseView.AuthMode == authModeReset {
		return renderAuthResetShell(parseAuth)
	}
	isSignup := parseView.AuthMode == authModeSignup
	return Div(
		Class("relative min-h-screen bg-[radial-gradient(circle_at_12%_10%,rgba(139,92,246,.18),transparent_24%),radial-gradient(circle_at_88%_14%,rgba(236,72,153,.16),transparent_26%),linear-gradient(180deg,#121726_0%,#171c2d_48%,#1b2135_100%)] text-[#f5f7fb] antialiased"),
		// ambient glow orbs
		Div(
			Class("pointer-events-none fixed inset-0 overflow-hidden"),
			Div(Class("absolute left-[6%] top-[6%] h-40 w-40 rounded-full bg-[#8b5cf6]/12 blur-3xl sm:h-56 sm:w-56 lg:h-64 lg:w-64"), nil),
			Div(Class("absolute right-[8%] top-[10%] h-44 w-44 rounded-full bg-[#ec4899]/12 blur-3xl sm:h-60 sm:w-60 lg:h-72 lg:w-72"), nil),
			Div(Class("absolute bottom-[8%] left-1/2 h-52 w-52 -translate-x-1/2 rounded-full bg-[#8b5cf6]/10 blur-3xl sm:h-72 sm:w-72"), nil),
		),
		renderAuthHeader(isSignup),
		Main(
			Class("relative z-10"),
			renderAuthBody(parseIntl, parseView, parseAuth, isSignup),
		),
		renderAuthFooter(),
	)
}

// renderAuthHeader renders the top bar with brand, signup/login toggle, and open-app CTA.
func renderAuthHeader(isSignup bool) ui.Node {
	parseSubtitleText := "Log in"
	if isSignup {
		parseSubtitleText = "Sign up"
	}
	parseSignupHref := authLandingRoute + "?mode=signup"
	parseLoginHref := authLandingRoute

	return Header(
		Class("relative z-20"),
		Div(
			Class("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-wrap items-center justify-between gap-4 py-5 sm:w-[min(1200px,calc(100%-32px))] sm:py-6 lg:w-[min(1200px,calc(100%-40px))] lg:flex-nowrap lg:py-7"),
			// brand
			A(
				Class("flex min-w-0 items-center gap-3 sm:gap-4"),
				Href(marketingHomeRoute),
				OnClick(parseLandingNavigateHandler(marketingHomeRoute)),
				Div(Class("grid h-10 w-10 shrink-0 place-items-center rounded-2xl bg-[linear-gradient(135deg,#c4b5fd_0%,#f9a8d4_100%)] text-sm font-black text-[#1a1330] sm:h-11 sm:w-11"), Text("RD")),
				Div(
					Class("min-w-0"),
					Div(Class("truncate text-[14px] font-semibold tracking-[-0.01em] sm:text-[15px]"), Text("RelayDesk")),
					Div(Class("truncate text-[10px] uppercase tracking-[0.16em] text-[#b8c2d9] sm:text-[11px] sm:tracking-[0.18em]"), Text(parseSubtitleText)),
				),
			),
			// actions
			Div(
				Class("flex w-full items-center gap-2 sm:gap-3 md:w-auto"),
				If(!isSignup,
					A(
						Class("hidden rounded-full bg-white/10 px-4 py-2 text-sm font-medium text-[#dfe6f7] transition hover:bg-white/15 sm:inline-flex"),
						Href(parseSignupHref),
						Text("Sign up"),
					),
				),
				If(isSignup,
					A(
						Class("hidden rounded-full bg-white/10 px-4 py-2 text-sm font-medium text-[#dfe6f7] transition hover:bg-white/15 sm:inline-flex"),
						Href(parseLoginHref),
						Text("Log in"),
					),
				),
				A(
					Class("inline-flex flex-1 items-center justify-center rounded-full bg-white px-4 py-2.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px] sm:flex-none sm:px-5"),
					Href(chatRouteRoot),
					OnClick(parseLandingNavigateHandler(chatRouteRoot)),
					Text("Open app"),
				),
			),
		),
	)
}

// renderAuthBody renders the two-column hero + form section.
func renderAuthBody(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController, isSignup bool) ui.Node {
	parseHeadingText := "Log in and get back to work."
	parseBodyText := "Access your chats, saved workflows, and team workspace from one clean entry point."
	parseBadgeText := "Secure access · clean entry"
	if isSignup {
		parseHeadingText = "Create your account and get started fast."
		parseBodyText = "Join RelayDesk with a clean signup flow built for real product onboarding, then move straight into your workspace."
		parseBadgeText = "Simple onboarding · premium entry"
	}

	parseStatCards := [][]string{
		{"Fast re-entry", "Jump back into your existing workspace without extra noise."},
		{"Secure flow", "A simple login surface designed for real product use."},
		{"Team ready", "Built for individual and shared workspace access."},
	}
	if isSignup {
		parseStatCards = [][]string{
			{"Quick setup", "Create an account and enter the product without extra complexity."},
			{"Clean onboarding", "A signup surface that feels premium without getting in the way."},
			{"Ready for teams", "Start solo and grow into shared workspace access later."},
		}
	}

	return Section(
		Class("pb-16 pt-4 sm:pb-20 sm:pt-6 md:pb-24 md:pt-8 lg:pb-28 lg:pt-12"),
		Div(
			Class("mx-auto grid w-[min(1200px,calc(100%-24px))] items-center gap-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-12 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[.95fr_1.05fr] lg:gap-14"),
			// left — hero copy + stat cards
			Div(
				Class("max-w-[640px]"),
				Div(Class("mb-5 inline-flex rounded-full bg-white/10 px-3 py-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:mb-7 sm:px-4 sm:text-[11px] sm:tracking-[0.18em]"),
					Text(parseBadgeText),
				),
				H1(Class("max-w-none text-4xl font-semibold leading-[0.95] tracking-[-0.055em] text-white sm:text-5xl md:max-w-[11ch] md:text-6xl xl:text-7xl"),
					Text(parseHeadingText),
				),
				P(Class("mt-5 max-w-[56ch] text-base leading-7 text-[#e6ebf8]/90 sm:mt-6 sm:text-lg sm:leading-8 lg:text-xl"),
					Text(parseBodyText),
				),
				Div(
					Class("mt-8 grid gap-4 sm:grid-cols-3 sm:gap-5"),
					Map(parseStatCards, func(parseCard []string) ui.Node {
						return Div(
							Class("rounded-[24px] bg-[linear-gradient(180deg,rgba(255,255,255,.14),rgba(255,255,255,.06))] px-5 py-6 sm:rounded-[28px]"),
							Div(Class("text-lg font-semibold tracking-[-0.04em] text-white sm:text-xl"), Text(parseCard[0])),
							P(Class("mt-2 text-sm leading-6 text-[#b8c2d9]"), Text(parseCard[1])),
						)
					}),
				),
			),
			// right — form card
			Div(
				Class("mx-auto w-full max-w-[520px]"),
				renderAuthFormCard(parseIntl, parseView, parseAuth, isSignup),
			),
		),
	)
}

// renderAuthFormCard renders the glass login/signup form.
func renderAuthFormCard(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController, isSignup bool) ui.Node {
	parseHeadingText := "Log in"
	parseFormSubLabel := "Welcome back"
	parseSubText := "Enter your email and password to continue into RelayDesk."
	parseSubmitLabel := parseIntl.T(chatI18nNamespace, "auth.signIn")
	parseSwitchText := "New to RelayDesk?"
	parseSwitchLinkText := "Create an account"
	parseSwitchHref := authLandingRoute + "?mode=signup"
	parseEmailLabel := "Email"
	parsePasswordPlaceholder := parseIntl.T(chatI18nNamespace, "auth.passwordPlaceholder")
	if isSignup {
		parseHeadingText = "Sign up"
		parseFormSubLabel = "Create your account"
		parseSubText = "Enter your details to create a RelayDesk account and continue into the app."
		parseSubmitLabel = parseIntl.T(chatI18nNamespace, "auth.createAccount")
		parseSwitchText = "Already have an account?"
		parseSwitchLinkText = "Log in"
		parseSwitchHref = authLandingRoute
		parseEmailLabel = "Work email"
		parsePasswordPlaceholder = "Create a password"
	}

	return Div(
		Class("rounded-[28px] bg-[linear-gradient(180deg,rgba(255,255,255,.14),rgba(255,255,255,.06))] px-5 py-6 sm:rounded-[32px] sm:px-8 sm:py-8"),
		// form header
		Div(
			Class("mb-6"),
			Div(Class("text-sm font-semibold text-[#dfe6f7]"), Text(parseFormSubLabel)),
			H2(Class("mt-2 text-3xl font-semibold tracking-[-0.04em] text-white sm:text-4xl"), Text(parseHeadingText)),
			P(Class("mt-3 text-sm leading-7 text-[#b8c2d9]"), Text(parseSubText)),
		),
		// fields
		Div(
			Class("space-y-5"),
			// full name (signup only)
			If(isSignup,
				Div(
					Tag("label",
						Class("mb-2 block text-sm font-medium text-[#dfe6f7]"),
						For(idAuthNameInput),
						Text("Full name"),
					),
					Input(
						ID(idAuthNameInput),
						Type("text"),
						Class("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b8c2d9] outline-none transition focus:bg-white/15"),
						Placeholder("Jane Doe"),
						Value(parseView.AuthDisplayName),
						OnInput(parseAuth.HandleDisplayNameInput),
					),
				),
			),
			// email
			Div(
				Tag("label",
					Class("mb-2 block text-sm font-medium text-[#dfe6f7]"),
					For(idAuthEmailInput),
					Text(parseEmailLabel),
				),
				Input(
					ID(idAuthEmailInput),
					Type("email"),
					Class("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b8c2d9] outline-none transition focus:bg-white/15"),
					Placeholder(parseIntl.T(chatI18nNamespace, "auth.emailPlaceholder")),
					Value(parseView.AuthEmail),
					OnInput(parseAuth.HandleEmailInput),
				),
			),
			// password
			Div(
				Div(
					Class("mb-2 flex items-center justify-between gap-3"),
					Tag("label",
						Class("block text-sm font-medium text-[#dfe6f7]"),
						For(idAuthPasswordInput),
						Text("Password"),
					),
					If(!isSignup,
						A(
							Class("text-sm text-[#b8c2d9] transition hover:text-white"),
							Href("#"),
							OnClick(parseAuth.HandleForgotPassword),
							Text("Forgot password?"),
						),
					),
				),
				Input(
					ID(idAuthPasswordInput),
					Type("password"),
					Class("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b8c2d9] outline-none transition focus:bg-white/15"),
					Placeholder(parsePasswordPlaceholder),
					Value(parseView.AuthPassword),
					OnInput(parseAuth.HandlePasswordInput),
					OnKeyDown(parseAuth.HandlePasswordKey),
				),
			),
			// confirm password (signup only — visual field; validation is server-side)
			If(isSignup,
				Div(
					Tag("label",
						Class("mb-2 block text-sm font-medium text-[#dfe6f7]"),
						Text("Confirm password"),
					),
					Input(
						Type("password"),
						Class("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b8c2d9] outline-none transition focus:bg-white/15"),
						Placeholder("Confirm your password"),
					),
				),
			),
			// terms agreement (signup only)
			If(isSignup,
				Tag("label",
					Class("flex items-start gap-3 text-sm text-[#dfe6f7]"),
					Input(Type("checkbox"), Class("mt-1 h-4 w-4 rounded bg-white/10")),
					Span(
						Text("I agree to the "),
						A(Class("text-white transition hover:text-[#f5f7fb]"), Href("#"), Text("Terms")),
						Text(" and "),
						A(Class("text-white transition hover:text-[#f5f7fb]"), Href("#"), Text("Privacy Policy")),
						Text("."),
					),
				),
			),
			// error banner
			If(parseView.AuthError != "",
				Div(Class("rounded-[18px] border border-red-400/20 bg-red-500/10 px-4 py-3 text-sm leading-6 text-red-100"), Text(parseView.AuthError)),
			),
			// submit
			Button(
				Class(ClassNames(
					"inline-flex w-full items-center justify-center rounded-full bg-white px-5 py-3.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px]",
					When(parseView.AuthSubmitting, "cursor-progress opacity-70"),
				)),
				DisabledIf(parseView.AuthSubmitting || !parseView.GRPCReady),
				OnClick(parseAuth.HandleSubmit),
				Text(parseSubmitLabel),
			),
		),
		// mode switch footer
		Div(
			Class("mt-6 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b8c2d9]"),
			Text(parseSwitchText+" "),
			A(
				Class("font-medium text-white transition hover:text-[#f5f7fb]"),
				Href(parseSwitchHref),
				OnClick(parseAuth.HandleModeToggle),
				Text(parseSwitchLinkText),
			),
		),
	)
}

// renderAuthResetShell renders the password-reset request page.
func renderAuthResetShell(parseAuth authSessionController) ui.Node {
	return Div(
		Class("relative min-h-screen bg-[radial-gradient(circle_at_12%_10%,rgba(139,92,246,.18),transparent_24%),radial-gradient(circle_at_88%_14%,rgba(236,72,153,.16),transparent_26%),linear-gradient(180deg,#121726_0%,#171c2d_48%,#1b2135_100%)] text-[#f5f7fb] antialiased"),
		// ambient glow orbs
		Div(
			Class("pointer-events-none fixed inset-0 overflow-hidden"),
			Div(Class("absolute left-[6%] top-[6%] h-40 w-40 rounded-full bg-[#8b5cf6]/12 blur-3xl sm:h-56 sm:w-56 lg:h-64 lg:w-64"), nil),
			Div(Class("absolute right-[8%] top-[10%] h-44 w-44 rounded-full bg-[#ec4899]/12 blur-3xl sm:h-60 sm:w-60 lg:h-72 lg:w-72"), nil),
			Div(Class("absolute bottom-[8%] left-1/2 h-52 w-52 -translate-x-1/2 rounded-full bg-[#8b5cf6]/10 blur-3xl sm:h-72 sm:w-72"), nil),
		),
		renderAuthResetHeader(),
		Main(
			Class("relative z-10"),
			renderAuthResetBody(parseAuth),
		),
		renderAuthFooter(),
	)
}

// renderAuthResetHeader renders the header bar for the password reset page.
func renderAuthResetHeader() ui.Node {
	return Header(
		Class("relative z-20"),
		Div(
			Class("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-wrap items-center justify-between gap-4 py-5 sm:w-[min(1200px,calc(100%-32px))] sm:py-6 lg:w-[min(1200px,calc(100%-40px))] lg:flex-nowrap lg:py-7"),
			// brand
			A(
				Class("flex min-w-0 items-center gap-3 sm:gap-4"),
				Href(marketingHomeRoute),
				OnClick(parseLandingNavigateHandler(marketingHomeRoute)),
				Div(Class("grid h-10 w-10 shrink-0 place-items-center rounded-2xl bg-[linear-gradient(135deg,#c4b5fd_0%,#f9a8d4_100%)] text-sm font-black text-[#1a1330] sm:h-11 sm:w-11"), Text("RD")),
				Div(
					Class("min-w-0"),
					Div(Class("truncate text-[14px] font-semibold tracking-[-0.01em] sm:text-[15px]"), Text("RelayDesk")),
					Div(Class("truncate text-[10px] uppercase tracking-[0.16em] text-[#b8c2d9] sm:text-[11px] sm:tracking-[0.18em]"), Text("Password reset")),
				),
			),
			// actions
			Div(
				Class("flex w-full items-center gap-2 sm:gap-3 md:w-auto"),
				A(
					Class("hidden rounded-full bg-white/10 px-4 py-2 text-sm font-medium text-[#dfe6f7] transition hover:bg-white/15 sm:inline-flex"),
					Href(authLandingRoute),
					Text("Log in"),
				),
				A(
					Class("inline-flex flex-1 items-center justify-center rounded-full bg-white px-4 py-2.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px] sm:flex-none sm:px-5"),
					Href(chatRouteRoot),
					OnClick(parseLandingNavigateHandler(chatRouteRoot)),
					Text("Open app"),
				),
			),
		),
	)
}

// renderAuthResetBody renders the two-column hero + reset form section.
func renderAuthResetBody(parseAuth authSessionController) ui.Node {
	parseStatCards := [][]string{
		{"Quick recovery", "Reset access without digging through a cluttered auth flow."},
		{"Secure flow", "A simple recovery surface designed for trusted account access."},
		{"Back to work", "Get the reset link, update your password, and return to your workspace."},
	}

	return Section(
		Class("pb-16 pt-4 sm:pb-20 sm:pt-6 md:pb-24 md:pt-8 lg:pb-28 lg:pt-12"),
		Div(
			Class("mx-auto grid w-[min(1200px,calc(100%-24px))] items-center gap-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-12 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[.95fr_1.05fr] lg:gap-14"),
			// left: hero copy + stat cards
			Div(
				Class("max-w-[640px]"),
				Div(Class("mb-5 inline-flex rounded-full bg-white/10 px-3 py-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:mb-7 sm:px-4 sm:text-[11px] sm:tracking-[0.18em]"),
					Text("Secure recovery · low friction"),
				),
				H1(Class("max-w-none text-4xl font-semibold leading-[0.95] tracking-[-0.055em] text-white sm:text-5xl md:max-w-[11ch] md:text-6xl xl:text-7xl"),
					Text("Reset your password and get back in."),
				),
				P(Class("mt-5 max-w-[56ch] text-base leading-7 text-[#e6ebf8]/92 sm:mt-6 sm:text-lg sm:leading-8 lg:text-xl"),
					Text("Recover access with a clean reset flow built for real product use. Enter your email and we'll send a secure reset link."),
				),
				Div(
					Class("mt-8 grid gap-4 sm:grid-cols-3 sm:gap-5"),
					Map(parseStatCards, func(parseCard []string) ui.Node {
						return Div(
							Class("rounded-[24px] bg-[linear-gradient(180deg,rgba(255,255,255,.14),rgba(255,255,255,.06))] px-5 py-6 sm:rounded-[28px]"),
							Div(Class("text-lg font-semibold tracking-[-0.04em] text-white sm:text-xl"), Text(parseCard[0])),
							P(Class("mt-2 text-sm leading-6 text-[#b8c2d9]"), Text(parseCard[1])),
						)
					}),
				),
			),
			// right: reset form card
			Div(
				Class("mx-auto w-full max-w-[520px]"),
				renderAuthResetFormCard(parseAuth),
			),
		),
	)
}

// renderAuthResetFormCard renders the glass email-submission form for password recovery.
func renderAuthResetFormCard(parseAuth authSessionController) ui.Node {
	return Div(
		Class("rounded-[28px] bg-[linear-gradient(180deg,rgba(255,255,255,.14),rgba(255,255,255,.06))] px-5 py-6 sm:rounded-[32px] sm:px-8 sm:py-8"),
		// form header
		Div(
			Class("mb-6"),
			Div(Class("text-sm font-semibold text-[#dfe6f7]"), Text("Recover your account")),
			H2(Class("mt-2 text-3xl font-semibold tracking-[-0.04em] text-white sm:text-4xl"), Text("Password reset")),
			P(Class("mt-3 text-sm leading-7 text-[#b8c2d9]"),
				Text("Enter the email tied to your RelayDesk account and we'll send you a reset link."),
			),
		),
		// email field
		Div(
			Class("space-y-5"),
			Div(
				Tag("label",
					Class("mb-2 block text-sm font-medium text-[#dfe6f7]"),
					For(idAuthEmailInput),
					Text("Email"),
				),
				Input(
					ID(idAuthEmailInput),
					Type("email"),
					Class("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b8c2d9] outline-none transition focus:bg-white/15"),
					Placeholder("you@company.com"),
					OnInput(parseAuth.HandleEmailInput),
				),
			),
			// submit
			Button(
				Class("inline-flex w-full items-center justify-center rounded-full bg-white px-5 py-3.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px]"),
				OnClick(parseAuth.HandleModeToggle),
				Text("Send reset link"),
			),
		),
		// back to login
		Div(
			Class("mt-6 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b8c2d9]"),
			Text("Remembered your password? "),
			A(
				Class("font-medium text-white transition hover:text-[#f5f7fb]"),
				Href(authLandingRoute),
				OnClick(parseAuth.HandleModeToggle),
				Text("Back to log in"),
			),
		),
		// create account
		Div(
			Class("mt-4 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b8c2d9]"),
			Text("Need a new account? "),
			A(
				Class("font-medium text-white transition hover:text-[#f5f7fb]"),
				Href(authLandingRoute+"?mode=signup"),
				OnClick(parseAuth.HandleModeToggle),
				Text("Create one"),
			),
		),
	)
}

// renderAuthUpdatePasswordShell renders the update-password page for account security settings.
func renderAuthUpdatePasswordShell(parseAuth authSessionController) ui.Node {
	return Div(
		Class("relative min-h-screen bg-[radial-gradient(circle_at_12%_10%,rgba(139,92,246,.18),transparent_24%),radial-gradient(circle_at_88%_14%,rgba(236,72,153,.16),transparent_26%),linear-gradient(180deg,#121726_0%,#171c2d_48%,#1b2135_100%)] text-[#f5f7fb] antialiased"),
		Div(
			Class("pointer-events-none fixed inset-0 overflow-hidden"),
			Div(Class("absolute left-[6%] top-[6%] h-40 w-40 rounded-full bg-[#8b5cf6]/12 blur-3xl sm:h-56 sm:w-56 lg:h-64 lg:w-64"), nil),
			Div(Class("absolute right-[8%] top-[10%] h-44 w-44 rounded-full bg-[#ec4899]/12 blur-3xl sm:h-60 sm:w-60 lg:h-72 lg:w-72"), nil),
			Div(Class("absolute bottom-[8%] left-1/2 h-52 w-52 -translate-x-1/2 rounded-full bg-[#8b5cf6]/10 blur-3xl sm:h-72 sm:w-72"), nil),
		),
		renderAuthUpdatePasswordHeader(),
		Main(
			Class("relative z-10"),
			renderAuthUpdatePasswordBody(parseAuth),
		),
		renderAuthFooter(),
	)
}

// renderAuthUpdatePasswordHeader renders the header bar for the update-password page.
func renderAuthUpdatePasswordHeader() ui.Node {
	return Header(
		Class("relative z-20"),
		Div(
			Class("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-wrap items-center justify-between gap-4 py-5 sm:w-[min(1200px,calc(100%-32px))] sm:py-6 lg:w-[min(1200px,calc(100%-40px))] lg:flex-nowrap lg:py-7"),
			A(
				Class("flex min-w-0 items-center gap-3 sm:gap-4"),
				Href(marketingHomeRoute),
				OnClick(parseLandingNavigateHandler(marketingHomeRoute)),
				Div(Class("grid h-10 w-10 shrink-0 place-items-center rounded-2xl bg-[linear-gradient(135deg,#c4b5fd_0%,#f9a8d4_100%)] text-sm font-black text-[#1a1330] sm:h-11 sm:w-11"), Text("RD")),
				Div(
					Class("min-w-0"),
					Div(Class("truncate text-[14px] font-semibold tracking-[-0.01em] sm:text-[15px]"), Text("RelayDesk")),
					Div(Class("truncate text-[10px] uppercase tracking-[0.16em] text-[#b8c2d9] sm:text-[11px] sm:tracking-[0.18em]"), Text("Update password")),
				),
			),
			Div(
				Class("flex w-full items-center gap-2 sm:gap-3 md:w-auto"),
				A(
					Class("hidden rounded-full bg-white/10 px-4 py-2 text-sm font-medium text-[#dfe6f7] transition hover:bg-white/15 sm:inline-flex"),
					Href(authLandingRoute),
					Text("Log in"),
				),
				A(
					Class("inline-flex flex-1 items-center justify-center rounded-full bg-white px-4 py-2.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px] sm:flex-none sm:px-5"),
					Href(chatRouteRoot),
					OnClick(parseLandingNavigateHandler(chatRouteRoot)),
					Text("Open app"),
				),
			),
		),
	)
}

// renderAuthUpdatePasswordBody renders the two-column hero + update-password form section.
func renderAuthUpdatePasswordBody(parseAuth authSessionController) ui.Node {
	parseStatCards := [][]string{
		{"Secure update", "Verify your current password before setting a new one."},
		{"Fast flow", "A simple form that gets you back into the product quickly."},
		{"Account control", "Built for real settings and account management screens."},
	}

	return Section(
		Class("pb-16 pt-4 sm:pb-20 sm:pt-6 md:pb-24 md:pt-8 lg:pb-28 lg:pt-12"),
		Div(
			Class("mx-auto grid w-[min(1200px,calc(100%-24px))] items-center gap-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-12 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[.95fr_1.05fr] lg:gap-14"),
			Div(
				Class("max-w-[640px]"),
				Div(Class("mb-5 inline-flex rounded-full bg-white/10 px-3 py-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:mb-7 sm:px-4 sm:text-[11px] sm:tracking-[0.18em]"),
					Text("Secure update · low friction"),
				),
				H1(Class("max-w-none text-4xl font-semibold leading-[0.95] tracking-[-0.055em] text-white sm:text-5xl md:max-w-[11ch] md:text-6xl xl:text-7xl"),
					Text("Change your password and keep moving."),
				),
				P(Class("mt-5 max-w-[56ch] text-base leading-7 text-[#e6ebf8]/92 sm:mt-6 sm:text-lg sm:leading-8 lg:text-xl"),
					Text("Update your existing password with a clean account settings flow designed for fast, secure access management."),
				),
				Div(
					Class("mt-8 grid gap-4 sm:grid-cols-3 sm:gap-5"),
					Map(parseStatCards, func(parseCard []string) ui.Node {
						return Div(
							Class("rounded-[24px] bg-[linear-gradient(180deg,rgba(255,255,255,.14),rgba(255,255,255,.06))] px-5 py-6 sm:rounded-[28px]"),
							Div(Class("text-lg font-semibold tracking-[-0.04em] text-white sm:text-xl"), Text(parseCard[0])),
							P(Class("mt-2 text-sm leading-6 text-[#b8c2d9]"), Text(parseCard[1])),
						)
					}),
				),
			),
			Div(
				Class("mx-auto w-full max-w-[520px]"),
				renderAuthUpdatePasswordFormCard(parseAuth),
			),
		),
	)
}

// renderAuthUpdatePasswordFormCard renders the glass form for changing an account password.
func renderAuthUpdatePasswordFormCard(parseAuth authSessionController) ui.Node {
	return Div(
		Class("rounded-[28px] bg-[linear-gradient(180deg,rgba(255,255,255,.14),rgba(255,255,255,.06))] px-5 py-6 sm:rounded-[32px] sm:px-8 sm:py-8"),
		Div(
			Class("mb-6"),
			Div(Class("text-sm font-semibold text-[#dfe6f7]"), Text("Account security")),
			H2(Class("mt-2 text-3xl font-semibold tracking-[-0.04em] text-white sm:text-4xl"), Text("Update password")),
			P(Class("mt-3 text-sm leading-7 text-[#b8c2d9]"),
				Text("Enter your current password, then choose a new one for your RelayDesk account."),
			),
		),
		Div(
			Class("space-y-5"),
			Div(
				Tag("label",
					Class("mb-2 block text-sm font-medium text-[#dfe6f7]"),
					For(idAuthPasswordInput),
					Text("Current password"),
				),
				Input(
					ID(idAuthPasswordInput),
					Type("password"),
					Class("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b8c2d9] outline-none transition focus:bg-white/15"),
					Placeholder("Enter your current password"),
					OnInput(parseAuth.HandlePasswordInput),
				),
			),
			Div(
				Tag("label",
					Class("mb-2 block text-sm font-medium text-[#dfe6f7]"),
					Text("New password"),
				),
				Input(
					Type("password"),
					Class("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b8c2d9] outline-none transition focus:bg-white/15"),
					Placeholder("Create a new password"),
				),
			),
			Div(
				Tag("label",
					Class("mb-2 block text-sm font-medium text-[#dfe6f7]"),
					Text("Confirm new password"),
				),
				Input(
					Type("password"),
					Class("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b8c2d9] outline-none transition focus:bg-white/15"),
					Placeholder("Confirm your new password"),
				),
			),
			Button(
				Class("inline-flex w-full items-center justify-center rounded-full bg-white px-5 py-3.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px]"),
				OnClick(parseLandingNavigateHandler(chatRouteRoot)),
				Text("Update password"),
			),
		),
		Div(
			Class("mt-6 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b8c2d9]"),
			Text("Need help instead? "),
			A(
				Class("font-medium text-white transition hover:text-[#f5f7fb]"),
				Href(authLandingRoute),
				OnClick(parseAuth.HandleModeToggle),
				Text("Return to log in"),
			),
		),
		Div(
			Class("mt-4 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b8c2d9]"),
			Text("Don't have an account? "),
			A(
				Class("font-medium text-white transition hover:text-[#f5f7fb]"),
				Href(authLandingRoute+"?mode=signup"),
				OnClick(parseAuth.HandleModeToggle),
				Text("Create one"),
			),
		),
	)
}

func renderAuthFooter() ui.Node {
	return Tag("footer",
		Class("relative z-10"),
		Div(
			Class("mx-auto grid w-[min(1200px,calc(100%-24px))] gap-8 py-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-10 sm:py-12 md:grid-cols-2 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[1.2fr_.8fr_.8fr_.8fr] lg:gap-12 lg:py-14"),
			// brand column
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
			renderLandingFooterColumn("Auth",
				renderLandingFooterLink("Sign up", authLandingRoute+"?mode=signup"),
				renderLandingFooterLink("Log in", authLandingRoute),
				renderLandingFooterLink("Open app", chatRouteRoot),
				renderLandingFooterLink("Support", "#"),
			),
			renderLandingFooterColumn("Company",
				renderLandingFooterLink("About", "#"),
				renderLandingFooterLink("Customers", "#"),
				renderLandingFooterLink("Security", "#"),
				renderLandingFooterLink("Contact", "#"),
			),
			renderLandingFooterColumn("Legal",
				renderLandingFooterLink("Privacy", "#"),
				renderLandingFooterLink("Terms", "#"),
				renderLandingFooterLink("Status", "#"),
				renderLandingFooterLink("Support", "#"),
			),
		),
		Div(
			Class("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-col gap-3 py-5 text-xs text-[#b8c2d9] sm:w-[min(1200px,calc(100%-32px))] sm:gap-4 sm:py-6 md:flex-row md:items-center md:justify-between lg:w-[min(1200px,calc(100%-40px))]"),
			Div(Text("© 2026 RelayDesk, Inc. All rights reserved.")),
			Div(
				Class("flex flex-wrap items-center gap-4 sm:gap-5"),
				A(Class("transition hover:text-white"), Href("#"), Text("Privacy Policy")),
				A(Class("transition hover:text-white"), Href("#"), Text("Terms of Service")),
				A(Class("transition hover:text-white"), Href("#"), Text("Status")),
			),
		),
	)
}
