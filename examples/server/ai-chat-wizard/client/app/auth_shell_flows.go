//go:build js && wasm

package app

import (
	"fmt"

	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/i18n"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// renderAuthResetShell renders the password-reset request page.
func renderAuthResetShell(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController) ui.Node {
	return Div(
		ClassStr("relative min-h-screen text-[#f0f0f8] antialiased page-bg"),
		renderPageBackground(),
		renderAuthResetHeader(parseIntl),
		Main(
			ClassStr("relative z-10"),
			renderAuthResetBody(parseIntl, parseView, parseAuth),
		),
		renderMarketingFooter(parseIntl, renderStandardFooterColumns(parseIntl)...),
	)
}

// renderAuthResetHeader renders the header bar for the password reset page.
func renderAuthResetHeader(parseIntl i18n.Runtime) ui.Node {
	c := chatI18nNamespace
	return Header(
		ClassStr("relative z-20"),
		Div(
			ClassStr("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-wrap items-center justify-between gap-4 py-5 sm:w-[min(1200px,calc(100%-32px))] sm:py-6 lg:w-[min(1200px,calc(100%-40px))] lg:flex-nowrap lg:py-7"),
			// brand
			A(
				ClassStr("flex min-w-0 items-center gap-3 sm:gap-4"),
				Href(marketingHomeRoute),
				OnClick(parseLandingNavigateHandler(marketingHomeRoute)),
				Img(
					Src(brandChatIconURL),
					Attr("alt", appBrandName),
					ClassStr("h-10 w-10 shrink-0 rounded-xl object-cover sm:h-11 sm:w-11"),
				),
				Div(
					ClassStr("min-w-0"),
					Div(ClassStr("truncate text-[14px] font-semibold tracking-[-0.01em] sm:text-[15px]"), Text(parseIntl.T(c, "auth.loadingBrand"))),
					Div(ClassStr("truncate text-[10px] uppercase tracking-[0.16em] text-[#b4b8d0] sm:text-[11px] sm:tracking-[0.18em]"), Text(parseIntl.T(c, "auth.passwordReset"))),
				),
			),
			// actions
			Div(
				ClassStr("flex w-full items-center gap-2 sm:gap-3 md:w-auto"),
				renderLanguageSelector(parseIntl),
				A(
					ClassStr("hidden rounded-full bg-white/10 px-4 py-2 text-sm font-medium text-[#e0e3f2] transition hover:bg-white/15 sm:inline-flex"),
					Href(authLoginRoute),
					OnClick(parseLandingNavigateHandler(authLoginRoute)),
					Text(parseIntl.T(c, "auth.logIn")),
				),
				A(
					ClassStr("inline-flex flex-1 items-center justify-center rounded-full bg-white px-4 py-2.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px] sm:flex-none sm:px-5"),
					Href(chatRouteRoot),
					OnClick(parseLandingNavigateHandler(chatRouteRoot)),
					Text(parseIntl.T(c, "auth.openApp")),
				),
			),
		),
	)
}

// renderAuthResetBody renders the two-column hero + reset form section.
func renderAuthResetBody(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController) ui.Node {
	c := chatI18nNamespace
	parseStatCards := [][]string{
		{parseIntl.T(c, "auth.resetStat1Title"), parseIntl.T(c, "auth.resetStat1Body")},
		{parseIntl.T(c, "auth.resetStat2Title"), parseIntl.T(c, "auth.resetStat2Body")},
		{parseIntl.T(c, "auth.resetStat3Title"), parseIntl.T(c, "auth.resetStat3Body")},
	}

	return Section(
		ClassStr("pb-16 pt-4 sm:pb-20 sm:pt-6 md:pb-24 md:pt-8 lg:pb-28 lg:pt-12"),
		Div(
			ClassStr("mx-auto grid w-[min(1200px,calc(100%-24px))] items-center gap-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-12 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[.95fr_1.05fr] lg:gap-14"),
			// left: hero copy + stat cards
			Div(
				ClassStr("max-w-[640px]"),
				Div(ClassStr("mb-5 inline-flex rounded-full bg-white/10 px-3 py-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:mb-7 sm:px-4 sm:text-[11px] sm:tracking-[0.18em]"),
					Text(parseIntl.T(c, "auth.resetBadge")),
				),
				H1(ClassStr("max-w-none text-4xl font-semibold leading-[0.95] tracking-[-0.055em] text-white sm:text-5xl md:max-w-[11ch] md:text-6xl xl:text-7xl"),
					Text(parseIntl.T(c, "auth.resetHeroTitle")),
				),
				P(ClassStr("mt-5 max-w-[56ch] text-base leading-7 text-[#e6ebf8]/92 sm:mt-6 sm:text-lg sm:leading-8 lg:text-xl"),
					Text(parseIntl.T(c, "auth.resetHeroBody")),
				),
				Div(
					ClassStr("mt-8 grid gap-4 sm:grid-cols-3 sm:gap-5"),
					Map(parseStatCards, func(parseCard []string) ui.Node {
						return Div(
							ClassStr("rounded-2xl border border-white/[0.06] bg-[#13131e] px-5 py-6"),
							Div(ClassStr("text-lg font-semibold tracking-[-0.04em] text-white sm:text-xl"), Text(parseCard[0])),
							P(ClassStr("mt-2 text-sm leading-6 text-[#b4b8d0]"), Text(parseCard[1])),
						)
					}),
				),
			),
			// right: reset form card (mounted as a component to enable usestate for sent-success)
			Div(
				ClassStr("mx-auto w-full max-w-[520px]"),
				ui.Component(renderAuthResetFormCardComponent, authResetFormCardProps{Intl: parseIntl, View: parseView, Auth: parseAuth}),
			),
		),
	)
}

// authResetFormCardProps are the props for the reset-form component.
type authResetFormCardProps struct {
	Intl i18n.Runtime
	View appViewState
	Auth authSessionController
}

// renderAuthResetFormCardComponent is the component entry point for the reset form card.
// It owns local "sent" state so the form can show a success confirmation without a route change.
func renderAuthResetFormCardComponent(parseProps authResetFormCardProps) ui.Node {
	parseIntl := parseProps.Intl
	parseView := parseProps.View
	parseAuth := parseProps.Auth
	c := chatI18nNamespace

	// Local sent state — true once the user clicks Send and the button fires.
	isParseSent := ui.UseState(false)

	handleSend := ui.UseEvent(func() {
		if isParseSent.Get() {
			return
		}
		// Mark sent optimistically; backend reset-link delivery is async / not yet wired.
		isParseSent.Set(true)
	})

	// Success state — show a confirmation card.
	if isParseSent.Get() {
		return Div(
			ClassStr("rounded-2xl border border-white/[0.06] bg-[#13131e] px-5 py-6 sm:px-8 sm:py-8"),
			Div(
				ClassStr("mb-6"),
				H2(ClassStr("text-2xl font-semibold tracking-[-0.04em] text-white sm:text-3xl"), Text(parseIntl.T(c, "auth.resetSentTitle"))),
				P(ClassStr("mt-3 text-sm leading-7 text-[#b4b8d0]"),
					Text(parseIntl.T(c, "auth.resetSentBody")),
				),
			),
			Div(
				ClassStr("rounded-[18px] border border-white/[0.06] bg-white/5 px-4 py-3 text-sm text-[#e0e3f2]"),
				Text(parseView.AuthEmail),
			),
			Div(
				ClassStr("mt-6 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b4b8d0]"),
				Text(parseIntl.T(c, "auth.resetRemembered")),
				A(
					ClassStr("font-medium text-white transition hover:text-[#f5f7fb]"),
					Href(authLoginRoute),
					OnClick(parseAuth.HandleModeToggle),
					Text(parseIntl.T(c, "auth.backToLogIn")),
				),
			),
		)
	}

	// Normal reset request form.
	return Div(
		ClassStr("rounded-2xl border border-white/[0.06] bg-[#13131e] px-5 py-6 sm:px-8 sm:py-8"),
		// form header
		Div(
			ClassStr("mb-6"),
			Div(ClassStr("text-sm font-semibold text-[#e0e3f2]"), Text(parseIntl.T(c, "auth.resetFormSub"))),
			H2(ClassStr("mt-2 text-3xl font-semibold tracking-[-0.04em] text-white sm:text-4xl"), Text(parseIntl.T(c, "auth.passwordReset"))),
			P(ClassStr("mt-3 text-sm leading-7 text-[#b4b8d0]"),
				Text(parseIntl.T(c, "auth.resetFormBody")),
			),
		),
		// email field
		Div(
			ClassStr("space-y-5"),
			Div(
				Tag("label",
					ClassStr("mb-2 block text-sm font-medium text-[#e0e3f2]"),
					For(idAuthEmailInput),
					Text(parseIntl.T(c, "auth.email")),
				),
				Input(
					ID(idAuthEmailInput),
					Type("email"),
					ClassStr("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b4b8d0] outline-none transition focus:bg-white/15"),
					Placeholder(parseIntl.T(c, "auth.companyEmailPlaceholder")),
					Value(parseView.AuthEmail),
					OnInput(parseAuth.HandleEmailInput),
				),
			),
			// error banner
			If(parseView.AuthError != "",
				Div(ID("auth-reset-error-banner"), ClassStr("flex flex-col gap-1 rounded-[18px] border border-red-400/20 bg-red-500/10 px-4 py-3 text-sm leading-6 text-red-100"),
					Text(parseUserErrorMessage(parseView.AuthError)),
					renderSupportIDChip(parseUserErrorRequestID(parseView.AuthError)),
				),
			),
			// submit
			Button(
				ClassStr(ClassNames(
					"inline-flex w-full items-center justify-center rounded-full bg-white px-5 py-3.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px]",
					When(parseView.AuthSubmitting, "cursor-progress opacity-70"),
				)),
				DisabledIf(parseView.AuthSubmitting || !parseView.GRPCReady),
				OnClick(handleSend),
				Text(parseIntl.T(c, "auth.sendResetLink")),
			),
		),
		// back to login
		Div(
			ClassStr("mt-6 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b4b8d0]"),
			Text(parseIntl.T(c, "auth.resetRemembered")),
			A(
				ClassStr("font-medium text-white transition hover:text-[#f5f7fb]"),
				Href(authLoginRoute),
				OnClick(parseAuth.HandleModeToggle),
				Text(parseIntl.T(c, "auth.backToLogIn")),
			),
		),
		// create account
		Div(
			ClassStr("mt-4 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b4b8d0]"),
			Text(parseIntl.T(c, "auth.resetNoAccount")),
			A(
				ClassStr("font-medium text-white transition hover:text-[#f5f7fb]"),
				Href(marketingSignupRoute),
				OnClick(parseLandingNavigateHandler(marketingSignupRoute)),
				Text(parseIntl.T(c, "auth.createOne")),
			),
		),
	)
}

// renderAuthUpdatePasswordShell renders the update-password page for account security settings.
func renderAuthUpdatePasswordShell(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController) ui.Node {
	return Div(
		ClassStr("relative min-h-screen text-[#f0f0f8] antialiased page-bg"),
		renderPageBackground(),
		renderAuthUpdatePasswordHeader(parseIntl),
		Main(
			ClassStr("relative z-10"),
			renderAuthUpdatePasswordBody(parseIntl, parseView, parseAuth),
		),
		renderMarketingFooter(parseIntl, renderStandardFooterColumns(parseIntl)...),
	)
}

// renderAuthUpdatePasswordHeader renders the header bar for the update-password page.
func renderAuthUpdatePasswordHeader(parseIntl i18n.Runtime) ui.Node {
	c := chatI18nNamespace
	return Header(
		ClassStr("relative z-20"),
		Div(
			ClassStr("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-wrap items-center justify-between gap-4 py-5 sm:w-[min(1200px,calc(100%-32px))] sm:py-6 lg:w-[min(1200px,calc(100%-40px))] lg:flex-nowrap lg:py-7"),
			A(
				ClassStr("flex min-w-0 items-center gap-3 sm:gap-4"),
				Href(marketingHomeRoute),
				OnClick(parseLandingNavigateHandler(marketingHomeRoute)),
				Img(
					Src(brandChatIconURL),
					Attr("alt", appBrandName),
					ClassStr("h-10 w-10 shrink-0 rounded-xl object-cover sm:h-11 sm:w-11"),
				),
				Div(
					ClassStr("min-w-0"),
					Div(ClassStr("truncate text-[14px] font-semibold tracking-[-0.01em] sm:text-[15px]"), Text(parseIntl.T(c, "auth.loadingBrand"))),
					Div(ClassStr("truncate text-[10px] uppercase tracking-[0.16em] text-[#b4b8d0] sm:text-[11px] sm:tracking-[0.18em]"), Text(parseIntl.T(c, "auth.updatePassword"))),
				),
			),
			Div(
				ClassStr("flex w-full items-center gap-2 sm:gap-3 md:w-auto"),
				renderLanguageSelector(parseIntl),
				A(
					ClassStr("hidden rounded-full bg-white/10 px-4 py-2 text-sm font-medium text-[#e0e3f2] transition hover:bg-white/15 sm:inline-flex"),
					Href(authLoginRoute),
					OnClick(parseLandingNavigateHandler(authLoginRoute)),
					Text(parseIntl.T(c, "auth.logIn")),
				),
				A(
					ClassStr("inline-flex flex-1 items-center justify-center rounded-full bg-white px-4 py-2.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px] sm:flex-none sm:px-5"),
					Href(chatRouteRoot),
					OnClick(parseLandingNavigateHandler(chatRouteRoot)),
					Text(parseIntl.T(c, "auth.openApp")),
				),
			),
		),
	)
}

// renderAuthUpdatePasswordBody renders the two-column hero + update-password form section.
func renderAuthUpdatePasswordBody(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController) ui.Node {
	c := chatI18nNamespace
	parseStatCards := [][]string{
		{parseIntl.T(c, "auth.updatePasswordStat1Title"), parseIntl.T(c, "auth.updatePasswordStat1Body")},
		{parseIntl.T(c, "auth.updatePasswordStat2Title"), parseIntl.T(c, "auth.updatePasswordStat2Body")},
		{parseIntl.T(c, "auth.updatePasswordStat3Title"), parseIntl.T(c, "auth.updatePasswordStat3Body")},
	}

	return Section(
		ClassStr("pb-16 pt-4 sm:pb-20 sm:pt-6 md:pb-24 md:pt-8 lg:pb-28 lg:pt-12"),
		Div(
			ClassStr("mx-auto grid w-[min(1200px,calc(100%-24px))] items-center gap-10 sm:w-[min(1200px,calc(100%-32px))] sm:gap-12 lg:w-[min(1200px,calc(100%-40px))] lg:grid-cols-[.95fr_1.05fr] lg:gap-14"),
			Div(
				ClassStr("max-w-[640px]"),
				Div(ClassStr("mb-5 inline-flex rounded-full bg-white/10 px-3 py-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8b5cf6] sm:mb-7 sm:px-4 sm:text-[11px] sm:tracking-[0.18em]"),
					Text(parseIntl.T(c, "auth.updatePasswordBadge")),
				),
				H1(ClassStr("max-w-none text-4xl font-semibold leading-[0.95] tracking-[-0.055em] text-white sm:text-5xl md:max-w-[11ch] md:text-6xl xl:text-7xl"),
					Text(parseIntl.T(c, "auth.updatePasswordHeroTitle")),
				),
				P(ClassStr("mt-5 max-w-[56ch] text-base leading-7 text-[#e6ebf8]/92 sm:mt-6 sm:text-lg sm:leading-8 lg:text-xl"),
					Text(parseIntl.T(c, "auth.updatePasswordHeroBody")),
				),
				Div(
					ClassStr("mt-8 grid gap-4 sm:grid-cols-3 sm:gap-5"),
					Map(parseStatCards, func(parseCard []string) ui.Node {
						return Div(
							ClassStr("rounded-2xl border border-white/[0.06] bg-[#13131e] px-5 py-6"),
							Div(ClassStr("text-lg font-semibold tracking-[-0.04em] text-white sm:text-xl"), Text(parseCard[0])),
							P(ClassStr("mt-2 text-sm leading-6 text-[#b4b8d0]"), Text(parseCard[1])),
						)
					}),
				),
			),
			Div(
				ClassStr("mx-auto w-full max-w-[520px]"),
				renderAuthUpdatePasswordFormCard(parseIntl, parseView, parseAuth),
			),
		),
	)
}

// renderAuthUpdatePasswordFormCard renders the glass form for changing an account password.
func renderAuthUpdatePasswordFormCard(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController) ui.Node {
	c := chatI18nNamespace
	return Div(
		ClassStr("rounded-2xl border border-white/[0.06] bg-[#13131e] px-5 py-6 sm:px-8 sm:py-8"),
		Div(
			ClassStr("mb-6"),
			Div(ClassStr("text-sm font-semibold text-[#e0e3f2]"), Text(parseIntl.T(c, "auth.updatePasswordFormSub"))),
			H2(ClassStr("mt-2 text-3xl font-semibold tracking-[-0.04em] text-white sm:text-4xl"), Text(parseIntl.T(c, "auth.updatePassword"))),
			P(ClassStr("mt-3 text-sm leading-7 text-[#b4b8d0]"),
				Text(parseIntl.T(c, "auth.updatePasswordFormBody")),
			),
		),
		Div(
			ClassStr("space-y-5"),
			Div(
				Tag("label",
					ClassStr("mb-2 block text-sm font-medium text-[#e0e3f2]"),
					For(idAuthPasswordInput),
					Text(parseIntl.T(c, "auth.currentPassword")),
				),
				Input(
					ID(idAuthPasswordInput),
					Type("password"),
					ClassStr("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b4b8d0] outline-none transition focus:bg-white/15"),
					Placeholder(parseIntl.T(c, "auth.currentPasswordPlaceholder")),
					OnInput(parseAuth.HandlePasswordInput),
				),
			),
			Div(
				Tag("label",
					ClassStr("mb-2 block text-sm font-medium text-[#e0e3f2]"),
					Text(parseIntl.T(c, "auth.newPassword")),
				),
				Input(
					Type("password"),
					ClassStr("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b4b8d0] outline-none transition focus:bg-white/15"),
					Placeholder(parseIntl.T(c, "auth.newPasswordPlaceholder")),
				),
			),
			Div(
				Tag("label",
					ClassStr("mb-2 block text-sm font-medium text-[#e0e3f2]"),
					Text(parseIntl.T(c, "auth.confirmNewPassword")),
				),
				Input(
					Type("password"),
					ClassStr("w-full rounded-[18px] bg-white/10 px-4 py-3.5 text-sm text-white placeholder:text-[#b4b8d0] outline-none transition focus:bg-white/15"),
					Placeholder(parseIntl.T(c, "auth.confirmNewPasswordPlaceholder")),
				),
			),
			// error banner
			If(parseView.AuthError != "",
				Div(ID("auth-update-password-error-banner"), ClassStr("flex flex-col gap-1 rounded-[18px] border border-red-400/20 bg-red-500/10 px-4 py-3 text-sm leading-6 text-red-100"),
					Text(parseUserErrorMessage(parseView.AuthError)),
					renderSupportIDChip(parseUserErrorRequestID(parseView.AuthError)),
				),
			),
			Button(
				ClassStr(ClassNames(
					"inline-flex w-full items-center justify-center rounded-full bg-white px-5 py-3.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px]",
					When(parseView.AuthSubmitting, "cursor-progress opacity-70"),
				)),
				DisabledIf(parseView.AuthSubmitting || !parseView.GRPCReady),
				OnClick(parseLandingNavigateHandler(chatRouteRoot)),
				Text(parseIntl.T(c, "auth.updatePasswordSubmit")),
			),
		),
		Div(
			ClassStr("mt-6 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b4b8d0]"),
			Text(parseIntl.T(c, "auth.updatePasswordHelp")),
			A(
				ClassStr("font-medium text-white transition hover:text-[#f5f7fb]"),
				Href(authLoginRoute),
				OnClick(parseAuth.HandleModeToggle),
				Text(parseIntl.T(c, "auth.returnToLogIn")),
			),
		),
		Div(
			ClassStr("mt-4 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b4b8d0]"),
			Text(parseIntl.T(c, "auth.noAccount")),
			A(
				ClassStr("font-medium text-white transition hover:text-[#f5f7fb]"),
				Href(marketingSignupRoute),
				OnClick(parseLandingNavigateHandler(marketingSignupRoute)),
				Text(parseIntl.T(c, "auth.createOne")),
			),
		),
	)
}

// authVerifyEmailShellProps are the props for the email verification shell.
type authVerifyEmailShellProps struct {
	Intl i18n.Runtime
	View appViewState
	Auth authSessionController
}

// renderAuthVerifyEmailShell renders the email-verification pending page.
func renderAuthVerifyEmailShell(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController) ui.Node {
	return Div(
		ClassStr("relative min-h-screen text-[#f0f0f8] antialiased page-bg"),
		renderPageBackground(),
		renderAuthVerifyEmailHeader(parseIntl),
		Main(
			ClassStr("relative z-10"),
			Div(
				ClassStr("mx-auto flex w-[min(1200px,calc(100%-24px))] items-center justify-center pb-20 pt-16 sm:w-[min(1200px,calc(100%-32px))] sm:pb-24 sm:pt-20 lg:w-[min(1200px,calc(100%-40px))]"),
				ui.Component(renderAuthVerifyEmailCardComponent, authVerifyEmailShellProps{Intl: parseIntl, View: parseView, Auth: parseAuth}),
			),
		),
		renderMarketingFooter(parseIntl, renderStandardFooterColumns(parseIntl)...),
	)
}

// renderAuthVerifyEmailHeader renders the header bar for the email verification page.
func renderAuthVerifyEmailHeader(parseIntl i18n.Runtime) ui.Node {
	c := chatI18nNamespace
	return Header(
		ClassStr("relative z-20"),
		Div(
			ClassStr("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-wrap items-center justify-between gap-4 py-5 sm:w-[min(1200px,calc(100%-32px))] sm:py-6 lg:w-[min(1200px,calc(100%-40px))] lg:flex-nowrap lg:py-7"),
			A(
				ClassStr("flex min-w-0 items-center gap-3 sm:gap-4"),
				Href(marketingHomeRoute),
				OnClick(parseLandingNavigateHandler(marketingHomeRoute)),
				Img(
					Src(brandChatIconURL),
					Attr("alt", appBrandName),
					ClassStr("h-10 w-10 shrink-0 rounded-xl object-cover sm:h-11 sm:w-11"),
				),
				Div(
					ClassStr("min-w-0"),
					Div(ClassStr("truncate text-[14px] font-semibold tracking-[-0.01em] sm:text-[15px]"), Text(parseIntl.T(c, "auth.loadingBrand"))),
					Div(ClassStr("truncate text-[10px] uppercase tracking-[0.16em] text-[#b4b8d0] sm:text-[11px] sm:tracking-[0.18em]"), Text(parseIntl.T(c, "auth.verifyEmailTitle"))),
				),
			),
			Div(
				ClassStr("flex w-full items-center gap-2 sm:gap-3 md:w-auto"),
				renderLanguageSelector(parseIntl),
				A(
					ClassStr("hidden rounded-full bg-white/10 px-4 py-2 text-sm font-medium text-[#e0e3f2] transition hover:bg-white/15 sm:inline-flex"),
					Href(authLoginRoute),
					OnClick(parseLandingNavigateHandler(authLoginRoute)),
					Text(parseIntl.T(c, "auth.logIn")),
				),
			),
		),
	)
}

// renderAuthVerifyEmailCardComponent is the hook-enabled component for the verification card.
// It owns local resent state so the resend button can show a confirmation without a route change.
func renderAuthVerifyEmailCardComponent(parseProps authVerifyEmailShellProps) ui.Node {
	parseIntl := parseProps.Intl
	parseView := parseProps.View
	parseAuth := parseProps.Auth
	c := chatI18nNamespace

	// Local resent state — true once the user clicks Resend.
	isParseResent := ui.UseState(false)

	handleResend := ui.UseEvent(func() {
		// Resend delivery is async / not yet wired to an RPC — optimistic UI only.
		isParseResent.Set(true)
	})

	parseEmailDisplay := parseView.AuthEmail
	if parseEmailDisplay == "" {
		parseEmailDisplay = parseView.SessionEmail
	}

	return Div(
		ClassStr("mx-auto w-full max-w-[520px]"),
		Div(
			ClassStr("rounded-2xl border border-white/[0.06] bg-[#13131e] px-5 py-6 sm:px-8 sm:py-8"),
			// icon badge
			Div(
				ClassStr("mb-6 flex flex-col items-start gap-3"),
				Div(ClassStr("flex h-12 w-12 items-center justify-center rounded-2xl border border-white/10 bg-white/5 text-2xl"), Text("✉️")),
				Div(
					H2(ClassStr("text-2xl font-semibold tracking-[-0.04em] text-white sm:text-3xl"), Text(parseIntl.T(c, "auth.verifyEmailHeading"))),
					P(ClassStr("mt-2 text-sm leading-7 text-[#b4b8d0]"), Text(parseIntl.T(c, "auth.verifyEmailBody"))),
				),
			),
			// email address chip
			If(parseEmailDisplay != "",
				Div(
					ClassStr("mb-5 rounded-[18px] border border-white/[0.06] bg-white/5 px-4 py-3 text-sm text-[#e0e3f2]"),
					Text(parseEmailDisplay),
				),
			),
			// resent confirmation banner
			If(isParseResent.Get(),
				Div(ClassStr("mb-5 flex flex-col gap-1 rounded-[18px] border border-emerald-400/20 bg-emerald-500/10 px-4 py-3 text-sm leading-6 text-emerald-100"),
					Text(parseIntl.T(c, "auth.verifyEmailResentConfirm")),
				),
			),
			// error banner (e.g. expired / already-used token state)
			If(parseView.AuthError != "",
				Div(ID("auth-verify-error-banner"), ClassStr("mb-5 flex flex-col gap-1 rounded-[18px] border border-red-400/20 bg-red-500/10 px-4 py-3 text-sm leading-6 text-red-100"),
					Text(parseUserErrorMessage(parseView.AuthError)),
					renderSupportIDChip(parseUserErrorRequestID(parseView.AuthError)),
				),
			),
			// actions
			Div(
				ClassStr("flex flex-col gap-3"),
				// resend button — shown while not yet resent
				If(!isParseResent.Get(),
					Button(
						ClassStr("inline-flex w-full items-center justify-center rounded-full bg-white/10 px-5 py-3 text-sm font-medium text-[#e0e3f2] transition hover:bg-white/20"),
						OnClick(handleResend),
						Text(parseIntl.T(c, "auth.verifyEmailResend")),
					),
				),
				// continue to app
				A(
					ClassStr("inline-flex w-full items-center justify-center rounded-full bg-white px-5 py-3.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px]"),
					Href(chatRouteRoot),
					OnClick(parseLandingNavigateHandler(chatRouteRoot)),
					Text(parseIntl.T(c, "auth.verifyEmailContinue")),
				),
			),
			// back to login
			Div(
				ClassStr("mt-6 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b4b8d0]"),
				Text(parseIntl.T(c, "auth.resetRemembered")),
				A(
					ClassStr("font-medium text-white transition hover:text-[#f5f7fb]"),
					Href(authLoginRoute),
					OnClick(parseAuth.HandleModeToggle),
					Text(parseIntl.T(c, "auth.backToLogIn")),
				),
			),
		),
	)
}

// authExternalFailureKind maps a raw failure reason to a structured display tuple.
type authExternalFailureKind struct {
	icon    string
	heading string
	body    string
}

// parseExternalFailureKind resolves the display copy for a given failure reason code.
// Reason codes are short slugs stored in AuthError (e.g. "consent_denied").
func parseExternalFailureKind(parseIntl i18n.Runtime, parseReason string) authExternalFailureKind {
	c := chatI18nNamespace
	switch parseReason {
	case "consent_denied":
		return authExternalFailureKind{
			icon:    "🚫",
			heading: parseIntl.T(c, "auth.extAuthConsentDeniedHeading"),
			body:    parseIntl.T(c, "auth.extAuthConsentDeniedBody"),
		}
	case "expired_state":
		return authExternalFailureKind{
			icon:    "⏳",
			heading: parseIntl.T(c, "auth.extAuthExpiredHeading"),
			body:    parseIntl.T(c, "auth.extAuthExpiredBody"),
		}
	case "policy_mismatch":
		return authExternalFailureKind{
			icon:    "🔒",
			heading: parseIntl.T(c, "auth.extAuthPolicyHeading"),
			body:    parseIntl.T(c, "auth.extAuthPolicyBody"),
		}
	case "sso_required":
		return authExternalFailureKind{
			icon:    "🏢",
			heading: parseIntl.T(c, "auth.extAuthSSORequiredHeading"),
			body:    parseIntl.T(c, "auth.extAuthSSORequiredBody"),
		}
	default:
		return authExternalFailureKind{
			icon:    "⚠️",
			heading: parseIntl.T(c, "auth.extAuthGenericHeading"),
			body:    fmt.Sprintf("%s %s", parseIntl.T(c, "auth.extAuthGenericBody"), parseUserErrorRequestID(parseReason)),
		}
	}
}

// renderAuthExternalFailureShell renders a standalone page for external-auth callback failures.
func renderAuthExternalFailureShell(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController) ui.Node {
	return Div(
		ClassStr("relative min-h-screen text-[#f0f0f8] antialiased page-bg"),
		renderPageBackground(),
		renderAuthVerifyEmailHeader(parseIntl),
		Main(
			ClassStr("relative z-10"),
			Div(
				ClassStr("mx-auto flex w-[min(1200px,calc(100%-24px))] items-center justify-center pb-20 pt-16 sm:w-[min(1200px,calc(100%-32px))] sm:pb-24 sm:pt-20 lg:w-[min(1200px,calc(100%-40px))]"),
				renderAuthExternalFailureCard(parseIntl, parseView, parseAuth),
			),
		),
		renderMarketingFooter(parseIntl, renderStandardFooterColumns(parseIntl)...),
	)
}

// renderAuthExternalFailureCard renders the external-auth error card, selecting
// copy from the failure reason stored in AuthError.
func renderAuthExternalFailureCard(parseIntl i18n.Runtime, parseView appViewState, parseAuth authSessionController) ui.Node {
	c := chatI18nNamespace
	parseKind := parseExternalFailureKind(parseIntl, parseView.AuthError)

	return Div(
		ClassStr("mx-auto w-full max-w-[520px]"),
		Div(
			ClassStr("rounded-2xl border border-white/[0.06] bg-[#13131e] px-5 py-6 sm:px-8 sm:py-8"),
			// icon + heading
			Div(
				ClassStr("mb-6 flex flex-col items-start gap-3"),
				Div(ClassStr("flex h-12 w-12 items-center justify-center rounded-2xl border border-white/10 bg-white/5 text-2xl"), Text(parseKind.icon)),
				Div(
					H2(ClassStr("text-2xl font-semibold tracking-[-0.04em] text-white sm:text-3xl"), Text(parseKind.heading)),
					P(ClassStr("mt-2 text-sm leading-7 text-[#b4b8d0]"), Text(parseKind.body)),
				),
			),
			// support ID chip if the error string contains a request ID token
			If(parseUserErrorRequestID(parseView.AuthError) != "",
				Div(ClassStr("mb-5"),
					renderSupportIDChip(parseUserErrorRequestID(parseView.AuthError)),
				),
			),
			// actions
			Div(
				ClassStr("flex flex-col gap-3"),
				// retry with the original provider
				Button(
					ClassStr("inline-flex w-full items-center justify-center rounded-full bg-white/10 px-5 py-3 text-sm font-medium text-[#e0e3f2] transition hover:bg-white/20"),
					OnClick(parseAuth.HandleModeToggle),
					Text(parseIntl.T(c, "auth.extAuthTryAgain")),
				),
				// always-available password login fallback
				A(
					ClassStr("inline-flex w-full items-center justify-center rounded-full bg-white px-5 py-3.5 text-sm font-semibold text-[#1a1330] transition hover:-translate-y-[1px]"),
					Href(authLoginRoute),
					OnClick(parseLandingNavigateHandler(authLoginRoute)),
					Text(parseIntl.T(c, "auth.extAuthUsePassword")),
				),
			),
			// back to login footer
			Div(
				ClassStr("mt-6 rounded-[22px] bg-white/5 px-4 py-4 text-sm text-[#b4b8d0]"),
				Text(parseIntl.T(c, "auth.extAuthLoginFooter")),
				A(
					ClassStr("font-medium text-white transition hover:text-[#f5f7fb]"),
					Href(authLoginRoute),
					OnClick(parseAuth.HandleModeToggle),
					Text(parseIntl.T(c, "auth.backToLogIn")),
				),
			),
		),
	)
}
