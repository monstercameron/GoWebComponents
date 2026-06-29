//go:build js && wasm

package app

import (
	"strings"

	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/i18n"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// ── Atoms ────────────────────────────────────────────────────────────────────

// renderPageBackground renders the shared full-screen background used by all marketing and auth pages.
// A hexagonal SVG grid sits at 3% opacity over a deep obsidian base with a single subtle cyan radial.
func renderPageBackground() ui.Node {
	return Div(
		ClassStr("pointer-events-none fixed inset-0 -z-10 page-bg overflow-hidden"),
	)
}

// renderBrandMark renders the RelayDesk chat icon badge.
func renderBrandMark() ui.Node {
	return Img(
		Src(brandChatIconURL),
		Attr("alt", appBrandName),
		ClassStr("h-10 w-10 shrink-0 rounded-xl object-cover sm:h-11 sm:w-11"),
	)
}

// renderHeroBadge renders a pill eyebrow label above hero headlines.
func renderHeroBadge(renderText string) ui.Node {
	return Div(
		ClassStr("mb-5 inline-flex items-center gap-2 rounded-full border border-white/[0.07] bg-white/[0.04] px-3 py-1.5 text-[10px] font-medium uppercase tracking-[0.22em] text-[#9b9bb1] sm:mb-7 sm:px-4 sm:text-[11px]"),
		Text(renderText),
	)
}

// renderSectionEyebrow renders the uppercase tracking label used above section H2s.
func renderSectionEyebrow(renderText string) ui.Node {
	return Div(
		ClassStr("text-[10px] font-semibold uppercase tracking-[0.22em] text-[#8e7bff]/70 sm:text-[11px]"),
		Text(renderText),
	)
}

// renderSectionDivider renders a thin ruled horizontal line with a centered label — instrument-panel style.
func renderSectionDivider(renderLabel string) ui.Node {
	return Div(
		ClassStr("mx-auto w-[min(1200px,calc(100%-24px))] py-6 sm:w-[min(1200px,calc(100%-32px))] lg:w-[min(1200px,calc(100%-40px))]"),
		Div(ClassStr("section-divider"), Text(renderLabel)),
	)
}

// renderStatusDot renders a pulsing green live-status dot.
func renderStatusDot() ui.Node {
	return Span(ClassStr("status-dot-live"), nil)
}

// renderMonoLabel renders a technical detail string in JetBrains Mono with an acid-green tint.
func renderMonoLabel(renderText string) ui.Node {
	return Span(
		ClassStr("font-mono-tech text-[#4ade80] text-xs tracking-tight"),
		Text(renderText),
	)
}

// renderCtaPrimary renders the primary CTA button — electric cyan fill, black text.
func renderCtaPrimary(renderLabel, renderTargetPath string) ui.Node {
	return A(
		ClassStr("cta-btn inline-flex items-center justify-center rounded-full bg-[#8e7bff] px-5 py-3 text-sm font-semibold text-black transition hover:-translate-y-[1px] hover:bg-[#a99bff] sm:px-6 sm:py-3.5"),
		Href(renderTargetPath),
		OnClick(parseLandingNavigateHandler(renderTargetPath)),
		Text(renderLabel),
	)
}

// renderCtaSecondary renders the secondary ghost CTA button — glass/outline style.
func renderCtaSecondary(renderLabel, renderTargetPath string) ui.Node {
	return A(
		ClassStr("inline-flex items-center justify-center rounded-full border border-white/[0.12] bg-white/[0.04] px-5 py-3 text-sm font-medium text-[#f0f0f8] transition hover:border-white/20 hover:bg-white/[0.08] sm:px-6 sm:py-3.5"),
		Href(renderTargetPath),
		OnClick(parseLandingNavigateHandler(renderTargetPath)),
		Text(renderLabel),
	)
}

// renderNavLink renders a router-aware nav link styled active when it matches the current path.
func renderNavLink(renderCurrentPath, renderTargetPath, renderLabel string) ui.Node {
	isActive := strings.TrimSpace(renderCurrentPath) == renderTargetPath ||
		(renderTargetPath == authLoginRoute && strings.TrimSpace(renderCurrentPath) == marketingHomeRoute)
	return A(
		ClassStr(ClassNames(
			"nav-link text-xs font-medium uppercase tracking-[0.16em] transition",
			When(isActive, "text-[#f0f0f8]"),
			When(!isActive, "text-[#9b9bb1] hover:text-[#f0f0f8]"),
		)),
		Href(renderTargetPath),
		OnClick(parseLandingNavigateHandler(renderTargetPath)),
		Text(renderLabel),
	)
}

// renderFooterLink renders a single footer nav anchor as a list item.
// Links to page routes (starting with /) use SPA navigation to avoid a full-page reload.
func renderFooterLink(renderLabel, renderHref string) ui.Node {
	renderLinkArgs := []interface{}{
		ClassStr("transition hover:text-[#f0f0f8]"),
		Href(renderHref),
		Text(renderLabel),
	}
	if strings.HasPrefix(renderHref, "/") {
		renderLinkArgs = append(renderLinkArgs, OnClick(parseLandingNavigateHandler(renderHref)))
	}
	return Li(
		A(renderLinkArgs...),
	)
}

// ── Molecules ────────────────────────────────────────────────────────────────

// renderMarketingHeader renders the sticky top bar shared across all marketing pages.
func renderMarketingHeader(parseIntl i18n.Runtime, renderCurrentPath string, renderNav ui.Node, renderActions ...ui.Node) ui.Node {
	renderActionArgs := make([]interface{}, 0, len(renderActions)+1)
	renderActionArgs = append(renderActionArgs, ClassStr("flex items-center gap-2 sm:gap-3 md:ml-auto"))
	for _, renderAction := range renderActions {
		renderActionArgs = append(renderActionArgs, renderAction)
	}
	return Header(
		ClassStr("sticky top-0 z-20 border-b border-white/[0.05] bg-[#070710]/80 backdrop-blur-sm"),
		Div(
			ClassStr("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-wrap items-center justify-between gap-4 py-4 sm:w-[min(1200px,calc(100%-32px))] sm:py-5 lg:w-[min(1200px,calc(100%-40px))] lg:flex-nowrap"),
			renderMarketingBrand(parseIntl, renderCurrentPath),
			renderNav,
			Div(renderActionArgs...),
		),
	)
}

// renderMarketingBrand renders the RelayDesk brand mark + wordmark, linking to home.
func renderMarketingBrand(parseIntl i18n.Runtime, renderCurrentPath string) ui.Node {
	_ = renderCurrentPath
	n := marketingI18nNamespace
	return A(
		ClassStr("flex min-w-0 items-center gap-3"),
		Href(marketingHomeRoute),
		OnClick(parseLandingNavigateHandler(marketingHomeRoute)),
		renderBrandMark(),
		Div(
			ClassStr("min-w-0"),
			Div(ClassStr("font-display truncate text-[14px] font-semibold tracking-[-0.02em] text-[#f0f0f8] sm:text-[15px]"), Text(parseIntl.T(n, "brand.name"))),
			Div(ClassStr("truncate text-[9px] uppercase tracking-[0.22em] text-[#9b9bb1] sm:text-[10px]"), Text(parseIntl.T(n, "brand.tagline"))),
		),
	)
}

// renderMarketingFooter renders the minimal two-row footer: brand+blurb left, columns right.
func renderMarketingFooter(parseIntl i18n.Runtime, renderColumns ...ui.Node) ui.Node {
	n := marketingI18nNamespace
	renderColArgs := make([]interface{}, 0, len(renderColumns)+1)
	renderColArgs = append(renderColArgs, ClassStr("flex flex-wrap gap-8 sm:gap-12 lg:gap-16"))
	for _, renderCol := range renderColumns {
		renderColArgs = append(renderColArgs, renderCol)
	}
	return Tag("footer",
		ClassStr("relative z-10 border-t border-white/[0.05] bg-[#070710]"),
		// main row: brand + columns
		Div(
			ClassStr("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-col gap-10 py-10 sm:w-[min(1200px,calc(100%-32px))] sm:flex-row sm:items-start sm:justify-between sm:py-12 lg:w-[min(1200px,calc(100%-40px))]"),
			// brand blurb
			Div(
				ClassStr("max-w-[26ch] shrink-0"),
				Div(
					ClassStr("flex items-center gap-3"),
					renderBrandMark(),
					Div(ClassStr("font-display text-[14px] font-semibold tracking-[-0.02em] text-[#f0f0f8]"), Text(parseIntl.T(n, "brand.name"))),
				),
				P(ClassStr("mt-4 text-sm leading-6 text-[#9b9bb1]"), Text(parseIntl.T(n, "brand.blurb"))),
			),
			Div(renderColArgs...),
		),
		// copyright bar
		Div(
			ClassStr("border-t border-white/[0.04]"),
			Div(
				ClassStr("mx-auto flex w-[min(1200px,calc(100%-24px))] flex-col gap-3 py-4 text-xs text-[#9b9bb1] sm:w-[min(1200px,calc(100%-32px))] sm:flex-row sm:items-center sm:justify-between lg:w-[min(1200px,calc(100%-40px))]"),
				Div(Text(parseIntl.T(n, "footer.copyright"))),
				Div(
					ClassStr("flex flex-wrap items-center gap-4"),
					A(ClassStr("transition hover:text-[#f0f0f8]"), Href("#"), Text(parseIntl.T(n, "footer.privacy"))),
					A(ClassStr("transition hover:text-[#f0f0f8]"), Href("#"), Text(parseIntl.T(n, "footer.terms"))),
					A(ClassStr("transition hover:text-[#f0f0f8]"), Href("#"), Text(parseIntl.T(n, "footer.status"))),
				),
			),
		),
	)
}

// renderFooterColumn renders a titled footer link column.
func renderFooterColumn(renderTitle string, renderLinks ...ui.Node) ui.Node {
	renderUlArgs := make([]interface{}, 0, len(renderLinks)+1)
	renderUlArgs = append(renderUlArgs, ClassStr("mt-3 space-y-2.5 text-sm text-[#9b9bb1] sm:mt-4"))
	for _, renderLink := range renderLinks {
		renderUlArgs = append(renderUlArgs, renderLink)
	}
	return Div(
		Div(ClassStr("text-[10px] font-semibold uppercase tracking-[0.2em] text-[#f0f0f8]/60 sm:text-[11px]"), Text(renderTitle)),
		Ul(renderUlArgs...),
	)
}

// renderMarketingHeroHeading renders the shared eyebrow, headline, body, and CTA row used by marketing heroes.
func renderMarketingHeroHeading(renderEyebrow, renderHeadline, renderBody string, renderActions ...ui.Node) ui.Node {
	renderArgs := make([]interface{}, 0, len(renderActions)+1)
	renderArgs = append(renderArgs, ClassStr("mt-8 flex flex-col gap-3 sm:mt-9 sm:flex-row"))
	for _, renderAction := range renderActions {
		renderArgs = append(renderArgs, renderAction)
	}
	return Div(
		renderHeroBadge(renderEyebrow),
		H1(
			ClassStr("font-display max-w-none text-5xl font-bold leading-[0.92] tracking-[-0.055em] text-[#f0f0f8] sm:text-6xl md:text-7xl xl:text-8xl"),
			Text(renderHeadline),
		),
		P(
			ClassStr("mt-5 max-w-[56ch] text-base leading-7 text-[#9b9bb1] sm:mt-6 sm:text-lg sm:leading-8"),
			Text(renderBody),
		),
		Div(renderArgs...),
	)
}

// renderStandardFooterColumns returns the shared Product/Company/Resources footer columns
// used across all marketing and auth pages.
func renderStandardFooterColumns(parseIntl i18n.Runtime) []ui.Node {
	n := marketingI18nNamespace
	return []ui.Node{
		renderFooterColumn(parseIntl.T(n, "footer.col.product"),
			renderFooterLink(parseIntl.T(n, "footer.link.overview"), marketingHomeRoute),
			renderFooterLink(parseIntl.T(n, "footer.link.pricing"), marketingPricingRoute),
			renderFooterLink(parseIntl.T(n, "footer.link.signup"), marketingSignupRoute),
		),
		renderFooterColumn(parseIntl.T(n, "footer.col.company"),
			renderFooterLink(parseIntl.T(n, "footer.link.about"), marketingAboutRoute),
			renderFooterLink(parseIntl.T(n, "footer.link.contact"), marketingContactRoute),
		),
		renderFooterColumn(parseIntl.T(n, "footer.col.resources"),
			renderFooterLink(parseIntl.T(n, "footer.link.security"), marketingSecurityRoute),
			renderFooterLink(parseIntl.T(n, "footer.status"), marketingStatusRoute),
			renderFooterLink(parseIntl.T(n, "footer.privacy"), marketingPrivacyRoute),
			renderFooterLink(parseIntl.T(n, "footer.terms"), marketingTermsRoute),
		),
	}
}

// renderMarketingHeaderBrand renders the brand treatment, optionally as a link.
func renderMarketingHeaderBrand(parseIntl i18n.Runtime, renderSubtitle, renderTargetPath string) ui.Node {
	n := marketingI18nNamespace
	if strings.TrimSpace(renderTargetPath) == "" {
		return Div(
			ClassStr("flex min-w-0 items-center gap-3"),
			renderBrandMark(),
			Div(
				ClassStr("min-w-0"),
				Div(ClassStr("font-display truncate text-[14px] font-semibold tracking-[-0.02em] text-[#f0f0f8] sm:text-[15px]"), Text(parseIntl.T(n, "brand.name"))),
				Div(ClassStr("truncate text-[9px] uppercase tracking-[0.22em] text-[#9b9bb1] sm:text-[10px]"), Text(renderSubtitle)),
			),
		)
	}
	return A(
		ClassStr("flex min-w-0 items-center gap-3"),
		Href(renderTargetPath),
		OnClick(parseLandingNavigateHandler(renderTargetPath)),
		renderBrandMark(),
		Div(
			ClassStr("min-w-0"),
			Div(ClassStr("font-display truncate text-[14px] font-semibold tracking-[-0.02em] text-[#f0f0f8] sm:text-[15px]"), Text(parseIntl.T(n, "brand.name"))),
			Div(ClassStr("truncate text-[9px] uppercase tracking-[0.22em] text-[#9b9bb1] sm:text-[10px]"), Text(renderSubtitle)),
		),
	)
}

// renderMarketingHeaderAction delegates to renderCtaPrimary or renderCtaSecondary.
func renderMarketingHeaderAction(renderLabel, renderTargetPath string, isPrimary, isHiddenOnSmall bool) ui.Node {
	if isPrimary {
		return A(
			ClassStr("inline-flex items-center justify-center rounded-full bg-[#8e7bff] px-4 py-2.5 text-sm font-semibold text-black transition hover:-translate-y-[1px] hover:bg-[#a99bff] sm:px-5"),
			Href(renderTargetPath),
			OnClick(parseLandingNavigateHandler(renderTargetPath)),
			Text(renderLabel),
		)
	}
	return A(
		ClassStr(ClassNames(
			"inline-flex items-center justify-center rounded-full border border-white/[0.10] bg-white/[0.04] px-4 py-2.5 text-sm font-medium text-[#f0f0f8] transition hover:border-white/20 hover:bg-white/[0.08] sm:px-5",
			When(isHiddenOnSmall, "hidden sm:inline-flex"),
		)),
		Href(renderTargetPath),
		OnClick(parseLandingNavigateHandler(renderTargetPath)),
		Text(renderLabel),
	)
}

// parseLandingNavLink delegates to renderNavLink.
func parseLandingNavLink(parseCurrentPath, parseTargetPath, parseLabel string) ui.Node {
	return renderNavLink(parseCurrentPath, parseTargetPath, parseLabel)
}

// parseLandingActionButton renders a rounded-pill CTA button — delegates to new atom.
func parseLandingActionButton(parseLabel, parseTargetPath string, isPrimary bool) ui.Node {
	if isPrimary {
		return renderCtaPrimary(parseLabel, parseTargetPath)
	}
	return renderCtaSecondary(parseLabel, parseTargetPath)
}

// parseLandingNavigateHandler returns a click handler that performs client-side router navigation,
// respecting modifier keys and default-prevented events so browser behaviour is preserved.
func parseLandingNavigateHandler(parseRouteTargetPath string) func(ui.Event) {
	parseRouteNormalizedTarget := strings.TrimSpace(parseRouteTargetPath)
	return func(parseE ui.Event) {
		parseRouteJSEvent := parseE.JSValue()
		if parseRouteJSEvent.Truthy() {
			if parseRouteJSEvent.Get("defaultPrevented").Bool() {
				return
			}
			if parseRouteJSEvent.Get("button").Int() != 0 {
				return
			}
			if parseRouteJSEvent.Get("metaKey").Bool() || parseRouteJSEvent.Get("ctrlKey").Bool() || parseRouteJSEvent.Get("shiftKey").Bool() || parseRouteJSEvent.Get("altKey").Bool() {
				return
			}
		}

		parseE.PreventDefault()
		if !shouldNavigateLandingRoute(strings.TrimSpace(router.GetCurrentPath()), parseRouteNormalizedTarget) {
			return
		}
		router.Navigate(parseRouteNormalizedTarget)
	}
}

// shouldNavigateLandingRoute returns whether a landing CTA should trigger a client-side route change.
func shouldNavigateLandingRoute(parseCurrentPath, parseTargetPath string) bool {
	parseCurrentPath = strings.TrimSpace(parseCurrentPath)
	parseTargetPath = strings.TrimSpace(parseTargetPath)
	if parseTargetPath == "" {
		return false
	}
	return parseCurrentPath != parseTargetPath
}

// renderLanguageSelector renders a compact locale <select> for the marketing navigation bar.
// Selecting a locale calls parseIntl.SetLocale which persists the choice and triggers a re-render.
func renderLanguageSelector(parseIntl i18n.Runtime) ui.Node {
	parseCurrentLocale := parseIntl.Locale()
	n := marketingI18nNamespace
	parseOptionNodes := make([]ui.Node, 0, len(availableLocales))
	for _, parseLocale := range availableLocales {
		parseOptionNodes = append(parseOptionNodes, Option(
			Value(parseLocale.ID),
			SelectedIf(parseLocale.ID == parseCurrentLocale),
			Text(parseLocale.NativeLabel),
		))
	}
	return Select(
		Attr("aria-label", parseIntl.T(n, "nav.language")),
		Attr("title", parseIntl.T(n, "nav.language")),
		Value(parseCurrentLocale),
		OnChange(func(parseE ui.Event) {
			parseIntl.SetLocale(parseE.GetValue())
		}),
		ClassStr("h-8 cursor-pointer rounded-lg border border-white/[0.08] bg-transparent px-2 text-sm text-[#9b9bb1] outline-none transition hover:border-white/[0.12] hover:text-[#f0f0f8]"),
		parseOptionNodes,
	)
}
