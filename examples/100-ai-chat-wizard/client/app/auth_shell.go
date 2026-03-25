//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderAuthLoadingShell keeps the root route stable while the gRPC bridge and
// persisted auth token are being resolved.
func renderAuthLoadingShell(intl i18n.Runtime, view appViewState) ui.Node {
	status := intl.T(chatI18nNamespace, "auth.connecting")
	if view.GRPCReady {
		status = intl.T(chatI18nNamespace, "auth.resolving")
	}
	return Div(
		Class("flex min-h-screen w-full items-center justify-center bg-[radial-gradient(circle_at_top,_rgba(25,195,125,0.18),_transparent_42%),linear-gradient(180deg,#151515_0%,#212121_100%)] px-6"),
		Div(
			Class("w-full max-w-md rounded-3xl border border-white/10 bg-[#171717]/90 p-8 shadow-[0_24px_80px_rgba(0,0,0,0.45)] backdrop-blur-xl"),
			Div(Class("mb-5 flex items-center gap-3"),
				Div(Class("flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-[#19c37d] to-[#0ea47e] text-sm font-semibold text-[#052516]"), Text(assistantBadgeText)),
				Div(Class("min-w-0"),
					P(Class("text-lg font-semibold tracking-tight text-white"), Text(appBrandName)),
					P(Class("text-sm text-white/45"), Text(appVersion)),
				),
			),
			P(Class("text-sm leading-6 text-white/70"), Text(status)),
		),
	)
}

// renderAuthShell renders the experimental login/signup flow entirely inside
// the GWC app so the server only serves the shell and gRPC bridge.
func renderAuthShell(intl i18n.Runtime, view appViewState, auth authSessionController) ui.Node {
	isSignup := view.AuthMode == authModeSignup
	submitLabel := intl.T(chatI18nNamespace, "auth.signIn")
	heading := intl.T(chatI18nNamespace, "auth.loginTitle")
	body := intl.T(chatI18nNamespace, "auth.loginBody")
	switchLabel := intl.T(chatI18nNamespace, "auth.switchToSignup")
	if isSignup {
		submitLabel = intl.T(chatI18nNamespace, "auth.createAccount")
		heading = intl.T(chatI18nNamespace, "auth.signupTitle")
		body = intl.T(chatI18nNamespace, "auth.signupBody")
		switchLabel = intl.T(chatI18nNamespace, "auth.switchToLogin")
	}
	return Div(
		Class("flex min-h-screen w-full items-center justify-center bg-[radial-gradient(circle_at_top,_rgba(25,195,125,0.22),_transparent_42%),linear-gradient(180deg,#151515_0%,#212121_100%)] px-6 py-12"),
		Div(
			Class("grid w-full max-w-5xl gap-6 lg:grid-cols-[1.15fr_0.85fr]"),
			Div(
				Class("hidden rounded-[2rem] border border-white/10 bg-[#171717]/85 p-10 shadow-[0_24px_80px_rgba(0,0,0,0.4)] backdrop-blur-xl lg:flex lg:flex-col lg:justify-between"),
				Div(
					Div(Class("mb-6 flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-[#19c37d] to-[#0ea47e] text-sm font-semibold text-[#052516]"), Text(assistantBadgeText)),
					H1(Class("max-w-lg text-4xl font-semibold tracking-tight text-white"), Text(intl.T(chatI18nNamespace, "auth.heroTitle"))),
					P(Class("mt-4 max-w-xl text-base leading-7 text-white/60"), Text(intl.T(chatI18nNamespace, "auth.heroBody"))),
				),
				Div(Class("rounded-2xl border border-white/8 bg-white/[0.03] p-5"),
					P(Class("text-xs font-medium uppercase tracking-[0.24em] text-[#7ee6ba]"), Text(intl.T(chatI18nNamespace, "auth.grpcBadge"))),
					P(Class("mt-2 text-sm leading-6 text-white/60"), Text(intl.T(chatI18nNamespace, "auth.grpcBody"))),
				),
			),
			Div(
				Class("rounded-[2rem] border border-white/10 bg-[#171717]/92 p-8 shadow-[0_24px_80px_rgba(0,0,0,0.45)] backdrop-blur-xl"),
				Div(Class("mb-8"),
					P(Class("text-xs font-medium uppercase tracking-[0.24em] text-[#7ee6ba]"), Text(intl.T(chatI18nNamespace, "auth.badge"))),
					H2(Class("mt-3 text-3xl font-semibold tracking-tight text-white"), Text(heading)),
					P(Class("mt-3 text-sm leading-6 text-white/60"), Text(body)),
				),
				Div(Class("flex flex-col gap-4"),
					If(isSignup,
						Div(Class("flex flex-col gap-1.5"),
							Label(Class("text-xs font-medium uppercase tracking-[0.18em] text-white/45"), Text(intl.T(chatI18nNamespace, "auth.displayName"))),
							Input(
								ID(idAuthNameInput),
								Type("text"),
								Class("w-full rounded-2xl border border-white/12 bg-[#232323] px-4 py-3 text-sm text-white outline-none transition-colors focus:border-[#19c37d]/50"),
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
							Class("w-full rounded-2xl border border-white/12 bg-[#232323] px-4 py-3 text-sm text-white outline-none transition-colors focus:border-[#19c37d]/50"),
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
							Class("w-full rounded-2xl border border-white/12 bg-[#232323] px-4 py-3 text-sm text-white outline-none transition-colors focus:border-[#19c37d]/50"),
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
							When(!view.AuthSubmitting, "hover:bg-[#27d889]"),
							When(view.AuthSubmitting, "cursor-progress opacity-80"),
						)),
						DisabledIf(view.AuthSubmitting || !view.GRPCReady),
						OnClick(auth.HandleSubmit),
						Text(submitLabel),
					),
					Button(
						Class("w-full rounded-2xl border border-white/10 px-4 py-3 text-sm text-white/70 transition-colors hover:bg-white/5 hover:text-white"),
						OnClick(auth.HandleModeToggle),
						Text(switchLabel),
					),
				),
			),
		),
	)
}
