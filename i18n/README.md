# GWC | I18n Library

```text
  ____ ____      __
 / ___|\\ \\ \\    / /
| |  _ \\ \\ \\\\ /\\ / /
| |_| | \\ V  V /
 \\____|  \\_/\\_/
GoWebComponents (GWC)
```

## High-Level Overview

The `i18n` library provides translation primitives and locale-aware text helpers for GWC apps.

## Public APIs

### `github.com/monstercameron/GoWebComponents/i18n` (`package i18n`)
- Functions: `BundleFromSSRBootstrap`, `DefaultLocale`, `Direction`, `DirectionForLocale`, `FallbackLocale`, `FormatDate`, `FormatNumber`, `Get`, `Locale`, `Locales`, `NewBundle`, `NormalizeLocale`, `PrefixPath`, `Provider`, `Register`, `RegisterNamespace`, `ResolvePath`, `Set`, `SetLocale`, `SupportedLocales`, `T`, `ToSSRBootstrap`, `Translate`, `UseI18n`, `UseLocale`
- Types: `Arguments`, `Bundle`, `BundleOptions`, `Catalog`, `DateOptions`, `DateStyle`, `Direction`, `LocaleOptions`, `LocaleState`, `Message`, `MissingHandler`, `NamespaceCatalog`, `NumberOptions`, `PluralCategory`, `ProviderProps`, `ResolvedPath`, `RouteOptions`, `Runtime`, `SSRBootstrapOptions`
- Variables: _none_
- Constants: `DateStyleLong`, `DateStyleMedium`, `DateStyleShort`, `DirectionAuto`, `DirectionLTR`, `DirectionRTL`, `PluralFew`, `PluralMany`, `PluralOne`, `PluralOther`, `PluralTwo`, `PluralZero`

## Subfiles And Purpose

- `doc.go` - Package-level Go documentation
- `helpers_more_test.go` - Tests for helpers_more behavior
- `i18n.go` - Core implementation for i18n
- `i18n_additional_test.go` - Tests for i18n_additional behavior
- `i18n_benchmark_test.go` - Tests for i18n_benchmark behavior
- `i18n_test.go` - Tests for i18n behavior
- `locale_native.go` - Native (non-WASM) implementation for locale
- `locale_wasm.go` - WebAssembly-specific implementation for locale
- `runtime_core_test.go` - Tests for runtime_core behavior

## ASCII File List

```text
i18n/
|-- doc.go
|-- helpers_more_test.go
|-- i18n.go
|-- i18n_additional_test.go
|-- i18n_benchmark_test.go
|-- i18n_test.go
|-- locale_native.go
|-- locale_wasm.go
\-- runtime_core_test.go
```
