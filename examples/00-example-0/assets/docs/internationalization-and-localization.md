# Internationalization and Localization

This document defines the current first-class i18n surface for GoWebComponents.

## At A Glance

- Use `i18n.UseLocale(...)` when locale belongs in the component tree and can change at runtime.
- Use an `i18n.Bundle` when translation lookup, fallback behavior, and SSR message transfer should be deterministic.
- Use `i18n.Provider(...)` plus `i18n.UseI18n()` when components need one runtime surface for translation, formatting, locale switching, direction, and path helpers.
- Keep translation file formats, remote catalog loading, locale-domain routing, and automatic document-level mutation in application code unless the repo ships those concerns explicitly.

## Quick API Chooser

Use this rule of thumb:

- choose `UseLocale(...)` when you need runtime locale state, persistence, browser detection, or user-driven switching
- choose `Bundle.Register(...)` and `Bundle.RegisterNamespace(...)` when you want first-party message catalogs with fallback semantics instead of ad hoc nested maps
- choose `UseI18n()` inside components when the component needs `T(...)`, formatting helpers, direction, or locale-prefixed path generation
- choose `ToSSRBootstrap(...)` and `BundleFromSSRBootstrap(...)` when SSR and hydration must start from the same locale and message subset

## Scope

The shipped i18n scope now includes:

- locale context and runtime switching
- message catalog registration and lookup by locale and namespace
- interpolation, pluralization, and select-style branching
- locale-aware number and date formatting helpers
- SSR bootstrap transfer for locale data and the initial message subset
- locale-prefix path helpers for applications that want URL-driven locale selection
- direction helpers for RTL-aware application shells

The framework does not own translation file formats, remote catalog fetching, locale-domain routing, or automatic document-level `lang` and `dir` mutation. Those stay application-owned so the public surface stays small and predictable.

## Example Shape

```go
bundle := i18n.NewBundle(i18n.BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
bundle.Register("en", i18n.Catalog{
	"checkout": {
		"headline": i18n.Message{Text: "Review your order"},
		"items":    i18n.Message{PluralArg: "count", Plural: map[i18n.PluralCategory]string{i18n.PluralOne: "{count} item", i18n.PluralOther: "{count} items"}},
	},
})

locale := i18n.UseLocale(i18n.LocaleOptions{
	InitialLocale:    "en",
	SupportedLocales: []string{"en", "fr"},
	FallbackLocale:   "en",
	PersistenceKey:   "checkout-locale",
})

return i18n.Provider(i18n.ProviderProps{
	Locale: locale,
	Bundle: bundle,
	Child: ui.CreateElement(func() ui.Node {
		intl := i18n.UseI18n()
		return html.Div(html.Props{Raw: map[string]interface{}{"lang": intl.Locale(), "dir": string(intl.Direction())}},
			html.H1(html.Props{}, html.Text(intl.T("checkout", "headline"))),
			html.P(html.Props{}, html.Text(intl.T("checkout", "items", i18n.Arguments{"count": 3}))),
			html.P(html.Props{}, html.Text(intl.FormatNumber(12540.75))),
		)
	}),
})
```

This is the intended model: locale state, bundle lookup, formatting, and direction stay explicit and composable inside ordinary component code.

## Public API

The public package is `github.com/monstercameron/GoWebComponents/i18n`.

Current entrypoints:

- `i18n.NewBundle(...)`
- `(*i18n.Bundle).Register(...)`
- `(*i18n.Bundle).RegisterNamespace(...)`
- `(*i18n.Bundle).Translate(...)`
- `i18n.UseLocale(...)`
- `i18n.Provider(...)`
- `i18n.UseI18n()`
- `i18n.FormatNumber(...)`, `i18n.FormatDate(...)`
- `i18n.DirectionForLocale(...)`
- `i18n.PrefixPath(...)`, `i18n.ResolvePath(...)`
- `(*i18n.Bundle).ToSSRBootstrap(...)`
- `i18n.BundleFromSSRBootstrap(...)`

Use `i18n.UseLocale(...)` when the active locale belongs in the component tree and may change at runtime. Use a `Bundle` when the application needs deterministic message lookup and fallback behavior instead of ad hoc nested maps.

## Locale State and Message Lookup

`i18n.UseLocale(...)` returns a stable locale handle with the current locale, a setter, direction information, supported locales, and the configured fallback locale.

`i18n.Provider(...)` exposes that locale handle and a `Bundle` to descendant components. `i18n.UseI18n()` then gives components the current locale, translation lookup through `T(...)`, route-prefix helpers, and formatting helpers.

Catalog organization is locale, namespace, then message key. Missing-message behavior is deterministic:

1. exact locale
2. base locale fallback, such as `fr-CA` to `fr`
3. configured fallback locale
4. bundle default locale
5. missing handler or `namespace.key`

## Formatting and Pluralization

Message entries support plain text with placeholder interpolation like `{name}`, plural messages keyed by CLDR-style categories, and select-style branching keyed by arbitrary string selectors.

Use `FormatNumber(...)` and `FormatDate(...)` when a value should change with locale rather than being hard-coded once and reused everywhere as a string.

The current date helper intentionally stays small. If an application needs richer calendar behavior or timezone policy, keep that concern in application code and pass the final values into components.

## SSR and Hydration

`ui.SSRBootstrap` now includes `I18n ui.SSRI18nBootstrap`.

That payload can transfer:

- active locale
- fallback locale
- resolved direction
- the initial message subset needed for hydration

Recommended flow:

1. Build a bundle on the server.
2. Trim it to the initial locale and any fallback locale with `ToSSRBootstrap(...)`.
3. Include the payload in `ui.SSRBootstrap`.
4. Hydrate on the client.
5. Rebuild the initial bundle with `BundleFromSSRBootstrap(...)`.

This keeps the first hydrated render aligned with the server-rendered language instead of forcing an immediate duplicate catalog fetch.

## Routing and Content Loading

Locale-aware routing is intentionally a companion concern, not router-owned global policy.

Current boundary:

- `router` owns route matching, navigation, loaders, guards, and hydration-aware mount behavior
- `i18n` owns locale normalization plus helpers for prefixing or resolving locale-aware paths
- application loaders own locale-specific content selection

Use `PrefixPath(...)` when generating locale-prefixed links, and `ResolvePath(...)` inside loaders or navigation code when the locale should be derived from the current path.

The framework does not enforce locale prefixes, locale domains, or locale metadata conventions globally. That policy remains application-owned.

## RTL and Directionality

`DirectionForLocale(...)` and `Runtime.Direction()` provide the current direction.

Recommended application behavior:

- set `dir` and `lang` on the highest shell you own for locale-scoped sections
- use explicit component-level `dir` overrides only for mixed-direction subtrees
- avoid hard-coded left or right assumptions in spacing, iconography, and layout copy

The framework does not automatically rewrite layout or CSS for RTL. Directionality remains an input to application markup and styles.

## Examples

Use these examples as the current end-to-end i18n reference:

- `examples/83-locale-switcher`
- `examples/84-ssr-i18n-bootstrap`
- `examples/85-locale-routing`

## Review Checklist

- does the app keep locale state, message lookup, and routing policy clearly separated instead of merging them into one opaque abstraction
- are message catalogs registered through `Bundle` APIs instead of scattered nested map reads throughout components
- does SSR transfer only the initial locale and message subset needed for hydration instead of shipping unnecessary catalog data
- are `lang` and `dir` applied intentionally in the owned shell markup rather than assumed to mutate automatically
- do routing examples treat locale prefixes as application policy layered on top of `router`, not as an implicit router global