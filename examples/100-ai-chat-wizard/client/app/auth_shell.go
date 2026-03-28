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

// setAuthDocumentTitle updates the browser tab title for auth and workspace-entry routes.
func setAuthDocumentTitle(parseIntl i18n.Runtime, parseAuthMode string) {
	parseDoc := js.Global().Get("document")
	if !parseDoc.Truthy() {
		return
	}
	c := chatI18nNamespace
	parseTitle := parseIntl.T(c, "auth.loadingBrand") + " - AI Chat Workspace"
	switch parseAuthMode {
	case authModeSignup:
		parseTitle = parseIntl.T(c, "auth.signUp") + " - " + parseIntl.T(c, "auth.loadingBrand")
	case authModeReset:
		parseTitle = parseIntl.T(c, "auth.forgotPassword") + " - " + parseIntl.T(c, "auth.loadingBrand")
	case authModeUpdatePassword:
		parseTitle = parseIntl.T(c, "auth.updatePassword") + " - " + parseIntl.T(c, "auth.loadingBrand")
	}
	parseDoc.Set("title", parseTitle)
}

// renderAuthLoadingShell keeps the root route stable while the gRPC bridge and
// persisted auth token are being resolved. Must be mounted via ui.Component.
func renderAuthLoadingShell(parseProps authLoadingShellProps) ui.Node {
	parseProgress := ui.UseState(0)
	parsePhase := ui.UseState("loading")
	parseIntl := i18n.UseI18n()
	c := chatI18nNamespace

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
					Class("absolute inset-y-0 left-0 rounded-full bg-[#00d9ff] transition-all duration-300"),
					Style(map[string]string{"width": fmt.Sprintf("%d%%", parsePct)}),
				),
			),
			Div(
				Class("flex items-center justify-between text-sm text-white/35"),
				Span(Text(parseIntl.T(c, "auth.loadingProgress"))),
				Span(Class("text-[11px] uppercase tracking-[0.18em]"), Text(parseIntl.T(c, "auth.loadingPleaseWait"))),
			),
		)
	} else {
		parseBody = Div(
			Class("flex min-h-[84px] items-center justify-between"),
			Div(
				Div(Class("text-sm text-white/[0.38]"), Text(parseIntl.T(c, "auth.loadingFinalizing"))),
				Div(Class("mt-2 text-base font-medium text-white/[0.88]"), Text(parseIntl.T(c, "auth.loadingAlmostReady"))),
			),
			Div(
				Class("relative h-10 w-10"),
				Div(Class("absolute inset-0 rounded-full border border-white/[0.08]"), nil),
				Div(Class("absolute inset-0 animate-spin rounded-full border-2 border-transparent border-t-[#00d9ff]"), nil),
			),
		)
	}

	return Div(
		Class("flex min-h-screen items-center justify-center bg-[#050508] px-6 text-[#f0f0f8]"),
		Div(
			Class("w-full max-w-lg rounded-2xl border border-white/[0.06] bg-[#111118] p-8 shadow-[0_20px_80px_rgba(0,0,0,0.72)]"),
			Div(
				Class("mb-8 flex items-center justify-between"),
				Div(
					Div(Class("text-[10px] uppercase tracking-[0.35em] text-white/35"), Text(parseIntl.T(c, "auth.loadingBrand"))),
					Div(Class("mt-3 text-xl font-medium tracking-tight text-white/90"), Text(parseIntl.T(c, "auth.loadingLabel"))),
				),
				Div(Class("text-sm tabular-nums text-white/40"), Text(parseDisplayPct)),
			),
			parseBody,
		),
	)
}

// renderAuthShell renders the login/signup/reset/update-password page matching the RelayDesk brand.
func renderAuthShell(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController) ui.Node {
	setAuthDocumentTitle(parseIntl, parseView.AuthMode)
	if parseView.AuthMode == authModeUpdatePassword {
		return renderAuthUpdatePasswordShell(parseIntl, parseAuth)
	}
	if parseView.AuthMode == authModeReset {
		return renderAuthResetShell(parseIntl, parseAuth)
	}
	isSignup := parseView.AuthMode == authModeSignup
	return Div(
		Class("relative min-h-screen text-[#f0f0f8] antialiased page-bg"),
		renderPageBackground(),
		renderAuthHeader(parseIntl, isSignup),
		Main(
			Class("relative z-10"),
			Div(
				Class("mx-auto w-[min(1200px,calc(100%-24px))] pt-4 sm:w-[min(1200px,calc(100%-32px))] sm:pt-5 lg:w-[min(1200px,calc(100%-40px))]"),
				renderJourneyProgressBand(parseBuildAuthJourneyStage(isSignup)),
			),
			renderAuthBody(parseIntl, parseView, parseAuth, isSignup),
		),
		renderMarketingFooter(parseIntl, renderStandardFooterColumns(parseIntl)...),
	)
}

// renderAuthHeader renders the top bar with brand, signup/login toggle, and open-app CTA.
func renderAuthHeader(parseIntl i18n.Runtime, isSignup bool) ui.Node {
	c := chatI18nNamespace
	parseSubtitleText := parseIntl.T(c, "auth.logIn")
	if isSignup {
		parseSubtitleText = parseIntl.T(c, "auth.signUp")
	}
	parseSignupHref := marketingSignupRoute
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
				Img(
					Src(brandChatIconURL),
					Attr("alt", appBrandName),
					Class("h-10 w-10 shrink-0 rounded-xl object-cover sm:h-11 sm:w-11"),
				),
				Div(
					Class("min-w-0"),
					Div(Class("truncate text-[14px] font-semibold tracking-[-0.01em] sm:text-[15px]"), Text(parseIntl.T(c, "auth.loadingBrand"))),
					Div(Class("truncate text-[10px] uppercase tracking-[0.16em] text-[#b8c2d9] sm:text-[11px] sm:tracking-[0.18em]"), Text(parseSubtitleText)),
				),
			),
			// actions
			Div(
				Class("flex w-full items-center gap-2 sm:gap-3 md:w-auto"),
				renderLanguageSelector(parseIntl),
				If(!isSignup,
					A(
						Class("hidden rounded-full bg-white/10 px-4 py-2 text-sm font-medium text-[#dfe6f7] transition hover:bg-white/15 sm:inline-flex"),
						Href(parseSignupHref),
						OnClick(parseLandingNavigateHandler(parseSignupHref)),
						Text(parseIntl.T(c, "auth.signUp")),
					),
				),
				If(isSignup,
					A(
						Class("hidden rounded-full bg-white/10 px-4 py-2 text-sm font-medium text-[#dfe6f7] transition hover:bg-white/15 sm:inline-flex"),
						Href(parseLoginHref),
						OnClick(parseLandingNavigateHandler(parseLoginHref)),
						Text(parseIntl.T(c, "auth.logIn")),
					),
				),
				A(
					Class("inline-flex flex-1 items-center justify-center rounded-full bg-white px-4 py-2.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px] sm:flex-none sm:px-5"),
					Href(chatRouteRoot),
					OnClick(parseLandingNavigateHandler(chatRouteRoot)),
					Text(parseIntl.T(c, "auth.openApp")),
				),
			),
		),
	)
}

// renderAuthBody renders the two-column hero + form section.
func renderAuthBody(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController, isSignup bool) ui.Node {
	c := chatI18nNamespace
	parseHeadingText := parseIntl.T(c, "auth.loginHeroTitle")
	parseBodyText := parseIntl.T(c, "auth.loginHeroBody")
	parseBadgeText := parseIntl.T(c, "auth.loginBadge")
	if isSignup {
		parseHeadingText = parseIntl.T(c, "auth.signupHeroTitle")
		parseBodyText = parseIntl.T(c, "auth.signupHeroBody")
		parseBadgeText = parseIntl.T(c, "auth.signupBadge")
	}

	parseStatCards := [][]string{
		{parseIntl.T(c, "auth.loginStat1Title"), parseIntl.T(c, "auth.loginStat1Body")},
		{parseIntl.T(c, "auth.loginStat2Title"), parseIntl.T(c, "auth.loginStat2Body")},
		{parseIntl.T(c, "auth.loginStat3Title"), parseIntl.T(c, "auth.loginStat3Body")},
	}
	if isSignup {
		parseStatCards = [][]string{
			{parseIntl.T(c, "auth.signupStat1Title"), parseIntl.T(c, "auth.signupStat1Body")},
			{parseIntl.T(c, "auth.signupStat2Title"), parseIntl.T(c, "auth.signupStat2Body")},
			{parseIntl.T(c, "auth.signupStat3Title"), parseIntl.T(c, "auth.signupStat3Body")},
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
							Class("rounded-2xl border border-white/[0.06] bg-[#111118] px-5 py-6"),
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
	c := chatI18nNamespace
	parseHeadingText := parseIntl.T(c, "auth.loginFormTitle")
	parseFormSubLabel := parseIntl.T(c, "auth.loginFormSub")
	parseSubText := parseIntl.T(c, "auth.loginFormBody")
	parseSubmitLabel := parseIntl.T(c, "auth.signIn")
	parseSwitchText := parseIntl.T(c, "auth.loginFormSwitchText")
	parseSwitchLinkText := parseIntl.T(c, "auth.loginFormSwitchLink")
	parseSwitchHref := marketingSignupRoute
	parseEmailLabel := parseIntl.T(c, "auth.email")
	parsePasswordPlaceholder := parseIntl.T(c, "auth.passwordPlaceholder")
	if isSignup {
		parseHeadingText = parseIntl.T(c, "auth.signupFormTitle")
		parseFormSubLabel = parseIntl.T(c, "auth.signupFormSub")
		parseSubText = parseIntl.T(c, "auth.signupFormBody")
		parseSubmitLabel = parseIntl.T(c, "auth.createAccount")
		parseSwitchText = parseIntl.T(c, "auth.signupFormSwitchText")
		parseSwitchLinkText = parseIntl.T(c, "auth.signupFormSwitchLink")
		parseSwitchHref = authLandingRoute
		parseEmailLabel = parseIntl.T(c, "auth.workEmail")
		parsePasswordPlaceholder = parseIntl.T(c, "auth.createPasswordPlaceholder")
	}

	return Div(
		Class("rounded-2xl border border-white/[0.06] bg-[#111118] px-5 py-6 sm:px-8 sm:py-8"),
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
						Text(parseIntl.T(c, "auth.fullName")),
					),
					Input(
						ID(idAuthNameInput),
						Type("text"),
						Class("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b8c2d9] outline-none transition focus:bg-white/15"),
						Placeholder(parseIntl.T(c, "auth.fullNamePlaceholder")),
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
						Text(parseIntl.T(c, "auth.password")),
					),
					If(!isSignup,
						A(
							Class("text-sm text-[#b8c2d9] transition hover:text-white"),
							Href("#"),
							OnClick(parseAuth.HandleForgotPassword),
							Text(parseIntl.T(c, "auth.forgotPassword")),
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
						Text(parseIntl.T(c, "auth.confirmPassword")),
					),
					Input(
						Type("password"),
						Class("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b8c2d9] outline-none transition focus:bg-white/15"),
						Placeholder(parseIntl.T(c, "auth.confirmPasswordPlaceholder")),
					),
				),
			),
			// terms agreement (signup only)
			If(isSignup,
				Tag("label",
					Class("flex items-start gap-3 text-sm text-[#dfe6f7]"),
					Input(Type("checkbox"), Class("mt-1 h-4 w-4 rounded bg-white/10")),
					Span(
						Text(parseIntl.T(c, "auth.termsText")),
						A(Class("text-white transition hover:text-[#f5f7fb]"), Href("#"), Text(parseIntl.T(c, "auth.termsLink"))),
						Text(parseIntl.T(c, "auth.termsAnd")),
						A(Class("text-white transition hover:text-[#f5f7fb]"), Href("#"), Text(parseIntl.T(c, "auth.privacyPolicy"))),
						Text(parseIntl.T(c, "auth.termsEnd")),
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
				OnClick(parseLandingNavigateHandler(parseSwitchHref)),
				Text(parseSwitchLinkText),
			),
		),
	)
}

// renderAuthResetShell renders the password-reset request page.
func renderAuthResetShell(parseIntl i18n.Runtime, parseAuth authSessionController) ui.Node {
	return Div(
		Class("relative min-h-screen text-[#f0f0f8] antialiased page-bg"),
		renderPageBackground(),
		renderAuthResetHeader(parseIntl),
		Main(
			Class("relative z-10"),
			renderAuthResetBody(parseIntl, parseAuth),
		),
		renderMarketingFooter(parseIntl, renderStandardFooterColumns(parseIntl)...),
	)
}

// renderAuthResetHeader renders the header bar for the password reset page.
func renderAuthResetHeader(parseIntl i18n.Runtime) ui.Node {
	c := chatI18nNamespace
	return Header(
		Class("relative z-20"),
		Div(
			Class("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-wrap items-center justify-between gap-4 py-5 sm:w-[min(1200px,calc(100%-32px))] sm:py-6 lg:w-[min(1200px,calc(100%-40px))] lg:flex-nowrap lg:py-7"),
			// brand
			A(
				Class("flex min-w-0 items-center gap-3 sm:gap-4"),
				Href(marketingHomeRoute),
				OnClick(parseLandingNavigateHandler(marketingHomeRoute)),
				Img(
					Src(brandChatIconURL),
					Attr("alt", appBrandName),
					Class("h-10 w-10 shrink-0 rounded-xl object-cover sm:h-11 sm:w-11"),
				),
				Div(
					Class("min-w-0"),
					Div(Class("truncate text-[14px] font-semibold tracking-[-0.01em] sm:text-[15px]"), Text(parseIntl.T(c, "auth.loadingBrand"))),
					Div(Class("truncate text-[10px] uppercase tracking-[0.16em] text-[#b8c2d9] sm:text-[11px] sm:tracking-[0.18em]"), Text(parseIntl.T(c, "auth.passwordReset"))),
				),
			),
			// actions
			Div(
				Class("flex w-full items-center gap-2 sm:gap-3 md:w-auto"),
				renderLanguageSelector(parseIntl),
				A(
					Class("hidden rounded-full bg-white/10 px-4 py-2 text-sm font-medium text-[#dfe6f7] transition hover:bg-white/15 sm:inline-flex"),
					Href(authLandingRoute),
					OnClick(parseLandingNavigateHandler(authLandingRoute)),
					Text(parseIntl.T(c, "auth.logIn")),
				),
				A(
					Class("inline-flex flex-1 items-center justify-center rounded-full bg-white px-4 py-2.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px] sm:flex-none sm:px-5"),
					Href(chatRouteRoot),
					OnClick(parseLandingNavigateHandler(chatRouteRoot)),
					Text(parseIntl.T(c, "auth.openApp")),
				),
			),
		),
	)
}

// renderAuthResetBody renders the two-column hero + reset form section.
func renderAuthResetBody(parseIntl i18n.Runtime, parseAuth authSessionController) ui.Node {
	c := chatI18nNamespace
	parseStatCards := [][]string{
		{parseIntl.T(c, "auth.resetStat1Title"), parseIntl.T(c, "auth.resetStat1Body")},
		{parseIntl.T(c, "auth.resetStat2Title"), parseIntl.T(c, "auth.resetStat2Body")},
		{parseIntl.T(c, "auth.resetStat3Title"), parseIntl.T(c, "auth.resetStat3Body")},
	}

	return Section(
		Class("pb-16 pt-4 sm:pb-20 sm:pt-6 md:pb-24 md:pt-8 lg:pb-28 lg:pt-12"),
		Div(
			Class("mx-auto grid w-[min(1200px,calc(100%-24px))] items-center gap-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-12 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[.95fr_1.05fr] lg:gap-14"),
			// left: hero copy + stat cards
			Div(
				Class("max-w-[640px]"),
				Div(Class("mb-5 inline-flex rounded-full bg-white/10 px-3 py-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:mb-7 sm:px-4 sm:text-[11px] sm:tracking-[0.18em]"),
					Text(parseIntl.T(c, "auth.resetBadge")),
				),
				H1(Class("max-w-none text-4xl font-semibold leading-[0.95] tracking-[-0.055em] text-white sm:text-5xl md:max-w-[11ch] md:text-6xl xl:text-7xl"),
					Text(parseIntl.T(c, "auth.resetHeroTitle")),
				),
				P(Class("mt-5 max-w-[56ch] text-base leading-7 text-[#e6ebf8]/92 sm:mt-6 sm:text-lg sm:leading-8 lg:text-xl"),
					Text(parseIntl.T(c, "auth.resetHeroBody")),
				),
				Div(
					Class("mt-8 grid gap-4 sm:grid-cols-3 sm:gap-5"),
					Map(parseStatCards, func(parseCard []string) ui.Node {
						return Div(
							Class("rounded-2xl border border-white/[0.06] bg-[#111118] px-5 py-6"),
							Div(Class("text-lg font-semibold tracking-[-0.04em] text-white sm:text-xl"), Text(parseCard[0])),
							P(Class("mt-2 text-sm leading-6 text-[#b8c2d9]"), Text(parseCard[1])),
						)
					}),
				),
			),
			// right: reset form card
			Div(
				Class("mx-auto w-full max-w-[520px]"),
				renderAuthResetFormCard(parseIntl, parseAuth),
			),
		),
	)
}

// renderAuthResetFormCard renders the glass email-submission form for password recovery.
func renderAuthResetFormCard(parseIntl i18n.Runtime, parseAuth authSessionController) ui.Node {
	c := chatI18nNamespace
	return Div(
		Class("rounded-2xl border border-white/[0.06] bg-[#111118] px-5 py-6 sm:px-8 sm:py-8"),
		// form header
		Div(
			Class("mb-6"),
			Div(Class("text-sm font-semibold text-[#dfe6f7]"), Text(parseIntl.T(c, "auth.resetFormSub"))),
			H2(Class("mt-2 text-3xl font-semibold tracking-[-0.04em] text-white sm:text-4xl"), Text(parseIntl.T(c, "auth.passwordReset"))),
			P(Class("mt-3 text-sm leading-7 text-[#b8c2d9]"),
				Text(parseIntl.T(c, "auth.resetFormBody")),
			),
		),
		// email field
		Div(
			Class("space-y-5"),
			Div(
				Tag("label",
					Class("mb-2 block text-sm font-medium text-[#dfe6f7]"),
					For(idAuthEmailInput),
					Text(parseIntl.T(c, "auth.email")),
				),
				Input(
					ID(idAuthEmailInput),
					Type("email"),
					Class("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b8c2d9] outline-none transition focus:bg-white/15"),
					Placeholder(parseIntl.T(c, "auth.companyEmailPlaceholder")),
					OnInput(parseAuth.HandleEmailInput),
				),
			),
			// submit
			Button(
				Class("inline-flex w-full items-center justify-center rounded-full bg-white px-5 py-3.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px]"),
				OnClick(parseAuth.HandleModeToggle),
				Text(parseIntl.T(c, "auth.sendResetLink")),
			),
		),
		// back to login
		Div(
			Class("mt-6 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b8c2d9]"),
			Text(parseIntl.T(c, "auth.resetRemembered")),
			A(
				Class("font-medium text-white transition hover:text-[#f5f7fb]"),
				Href(authLandingRoute),
				OnClick(parseAuth.HandleModeToggle),
				Text(parseIntl.T(c, "auth.backToLogIn")),
			),
		),
		// create account
		Div(
			Class("mt-4 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b8c2d9]"),
			Text(parseIntl.T(c, "auth.resetNoAccount")),
			A(
				Class("font-medium text-white transition hover:text-[#f5f7fb]"),
				Href(marketingSignupRoute),
				OnClick(parseLandingNavigateHandler(marketingSignupRoute)),
				Text(parseIntl.T(c, "auth.createOne")),
			),
		),
	)
}

// renderAuthUpdatePasswordShell renders the update-password page for account security settings.
func renderAuthUpdatePasswordShell(parseIntl i18n.Runtime, parseAuth authSessionController) ui.Node {
	return Div(
		Class("relative min-h-screen text-[#f0f0f8] antialiased page-bg"),
		renderPageBackground(),
		renderAuthUpdatePasswordHeader(parseIntl),
		Main(
			Class("relative z-10"),
			renderAuthUpdatePasswordBody(parseIntl, parseAuth),
		),
		renderMarketingFooter(parseIntl, renderStandardFooterColumns(parseIntl)...),
	)
}

// renderAuthUpdatePasswordHeader renders the header bar for the update-password page.
func renderAuthUpdatePasswordHeader(parseIntl i18n.Runtime) ui.Node {
	c := chatI18nNamespace
	return Header(
		Class("relative z-20"),
		Div(
			Class("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-wrap items-center justify-between gap-4 py-5 sm:w-[min(1200px,calc(100%-32px))] sm:py-6 lg:w-[min(1200px,calc(100%-40px))] lg:flex-nowrap lg:py-7"),
			A(
				Class("flex min-w-0 items-center gap-3 sm:gap-4"),
				Href(marketingHomeRoute),
				OnClick(parseLandingNavigateHandler(marketingHomeRoute)),
				Img(
					Src(brandChatIconURL),
					Attr("alt", appBrandName),
					Class("h-10 w-10 shrink-0 rounded-xl object-cover sm:h-11 sm:w-11"),
				),
				Div(
					Class("min-w-0"),
					Div(Class("truncate text-[14px] font-semibold tracking-[-0.01em] sm:text-[15px]"), Text(parseIntl.T(c, "auth.loadingBrand"))),
					Div(Class("truncate text-[10px] uppercase tracking-[0.16em] text-[#b8c2d9] sm:text-[11px] sm:tracking-[0.18em]"), Text(parseIntl.T(c, "auth.updatePassword"))),
				),
			),
			Div(
				Class("flex w-full items-center gap-2 sm:gap-3 md:w-auto"),
				renderLanguageSelector(parseIntl),
				A(
					Class("hidden rounded-full bg-white/10 px-4 py-2 text-sm font-medium text-[#dfe6f7] transition hover:bg-white/15 sm:inline-flex"),
					Href(authLandingRoute),
					OnClick(parseLandingNavigateHandler(authLandingRoute)),
					Text(parseIntl.T(c, "auth.logIn")),
				),
				A(
					Class("inline-flex flex-1 items-center justify-center rounded-full bg-white px-4 py-2.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px] sm:flex-none sm:px-5"),
					Href(chatRouteRoot),
					OnClick(parseLandingNavigateHandler(chatRouteRoot)),
					Text(parseIntl.T(c, "auth.openApp")),
				),
			),
		),
	)
}

// renderAuthUpdatePasswordBody renders the two-column hero + update-password form section.
func renderAuthUpdatePasswordBody(parseIntl i18n.Runtime, parseAuth authSessionController) ui.Node {
	c := chatI18nNamespace
	parseStatCards := [][]string{
		{parseIntl.T(c, "auth.updatePasswordStat1Title"), parseIntl.T(c, "auth.updatePasswordStat1Body")},
		{parseIntl.T(c, "auth.updatePasswordStat2Title"), parseIntl.T(c, "auth.updatePasswordStat2Body")},
		{parseIntl.T(c, "auth.updatePasswordStat3Title"), parseIntl.T(c, "auth.updatePasswordStat3Body")},
	}

	return Section(
		Class("pb-16 pt-4 sm:pb-20 sm:pt-6 md:pb-24 md:pt-8 lg:pb-28 lg:pt-12"),
		Div(
			Class("mx-auto grid w-[min(1200px,calc(100%-24px))] items-center gap-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-12 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[.95fr_1.05fr] lg:gap-14"),
			Div(
				Class("max-w-[640px]"),
				Div(Class("mb-5 inline-flex rounded-full bg-white/10 px-3 py-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:mb-7 sm:px-4 sm:text-[11px] sm:tracking-[0.18em]"),
					Text(parseIntl.T(c, "auth.updatePasswordBadge")),
				),
				H1(Class("max-w-none text-4xl font-semibold leading-[0.95] tracking-[-0.055em] text-white sm:text-5xl md:max-w-[11ch] md:text-6xl xl:text-7xl"),
					Text(parseIntl.T(c, "auth.updatePasswordHeroTitle")),
				),
				P(Class("mt-5 max-w-[56ch] text-base leading-7 text-[#e6ebf8]/92 sm:mt-6 sm:text-lg sm:leading-8 lg:text-xl"),
					Text(parseIntl.T(c, "auth.updatePasswordHeroBody")),
				),
				Div(
					Class("mt-8 grid gap-4 sm:grid-cols-3 sm:gap-5"),
					Map(parseStatCards, func(parseCard []string) ui.Node {
						return Div(
							Class("rounded-2xl border border-white/[0.06] bg-[#111118] px-5 py-6"),
							Div(Class("text-lg font-semibold tracking-[-0.04em] text-white sm:text-xl"), Text(parseCard[0])),
							P(Class("mt-2 text-sm leading-6 text-[#b8c2d9]"), Text(parseCard[1])),
						)
					}),
				),
			),
			Div(
				Class("mx-auto w-full max-w-[520px]"),
				renderAuthUpdatePasswordFormCard(parseIntl, parseAuth),
			),
		),
	)
}

// renderAuthUpdatePasswordFormCard renders the glass form for changing an account password.
func renderAuthUpdatePasswordFormCard(parseIntl i18n.Runtime, parseAuth authSessionController) ui.Node {
	c := chatI18nNamespace
	return Div(
		Class("rounded-2xl border border-white/[0.06] bg-[#111118] px-5 py-6 sm:px-8 sm:py-8"),
		Div(
			Class("mb-6"),
			Div(Class("text-sm font-semibold text-[#dfe6f7]"), Text(parseIntl.T(c, "auth.updatePasswordFormSub"))),
			H2(Class("mt-2 text-3xl font-semibold tracking-[-0.04em] text-white sm:text-4xl"), Text(parseIntl.T(c, "auth.updatePassword"))),
			P(Class("mt-3 text-sm leading-7 text-[#b8c2d9]"),
				Text(parseIntl.T(c, "auth.updatePasswordFormBody")),
			),
		),
		Div(
			Class("space-y-5"),
			Div(
				Tag("label",
					Class("mb-2 block text-sm font-medium text-[#dfe6f7]"),
					For(idAuthPasswordInput),
					Text(parseIntl.T(c, "auth.currentPassword")),
				),
				Input(
					ID(idAuthPasswordInput),
					Type("password"),
					Class("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b8c2d9] outline-none transition focus:bg-white/15"),
					Placeholder(parseIntl.T(c, "auth.currentPasswordPlaceholder")),
					OnInput(parseAuth.HandlePasswordInput),
				),
			),
			Div(
				Tag("label",
					Class("mb-2 block text-sm font-medium text-[#dfe6f7]"),
					Text(parseIntl.T(c, "auth.newPassword")),
				),
				Input(
					Type("password"),
					Class("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b8c2d9] outline-none transition focus:bg-white/15"),
					Placeholder(parseIntl.T(c, "auth.newPasswordPlaceholder")),
				),
			),
			Div(
				Tag("label",
					Class("mb-2 block text-sm font-medium text-[#dfe6f7]"),
					Text(parseIntl.T(c, "auth.confirmNewPassword")),
				),
				Input(
					Type("password"),
					Class("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b8c2d9] outline-none transition focus:bg-white/15"),
					Placeholder(parseIntl.T(c, "auth.confirmNewPasswordPlaceholder")),
				),
			),
			Button(
				Class("inline-flex w-full items-center justify-center rounded-full bg-white px-5 py-3.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px]"),
				OnClick(parseLandingNavigateHandler(chatRouteRoot)),
				Text(parseIntl.T(c, "auth.updatePasswordSubmit")),
			),
		),
		Div(
			Class("mt-6 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b8c2d9]"),
			Text(parseIntl.T(c, "auth.updatePasswordHelp")),
			A(
				Class("font-medium text-white transition hover:text-[#f5f7fb]"),
				Href(authLandingRoute),
				OnClick(parseAuth.HandleModeToggle),
				Text(parseIntl.T(c, "auth.returnToLogIn")),
			),
		),
		Div(
			Class("mt-4 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b8c2d9]"),
			Text(parseIntl.T(c, "auth.noAccount")),
			A(
				Class("font-medium text-white transition hover:text-[#f5f7fb]"),
				Href(marketingSignupRoute),
				OnClick(parseLandingNavigateHandler(marketingSignupRoute)),
				Text(parseIntl.T(c, "auth.createOne")),
			),
		),
	)
}
