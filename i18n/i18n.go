package i18n

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/ui"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type Direction string

const (
	DirectionLTR  Direction = "ltr"
	DirectionRTL  Direction = "rtl"
	DirectionAuto Direction = "auto"
)

type PluralCategory string

const (
	PluralZero  PluralCategory = "zero"
	PluralOne   PluralCategory = "one"
	PluralTwo   PluralCategory = "two"
	PluralFew   PluralCategory = "few"
	PluralMany  PluralCategory = "many"
	PluralOther PluralCategory = "other"
)

type Message struct {
	Text      string
	PluralArg string
	Plural    map[PluralCategory]string
	SelectArg string
	Select    map[string]string
	Default   string
}

type NamespaceCatalog map[string]Message
type Catalog map[string]NamespaceCatalog
type Arguments map[string]interface{}
type MissingHandler func(locale string, namespace string, key string) string

type BundleOptions struct {
	DefaultLocale  string
	FallbackLocale string
	OnMissing      MissingHandler
}

type Bundle struct {
	defaultLocale  string
	fallbackLocale string
	onMissing      MissingHandler
	catalogs       map[string]Catalog
}

type LocaleOptions struct {
	InitialLocale    string
	SupportedLocales []string
	FallbackLocale   string
	PersistenceKey   string
	DetectBrowser    bool
	OnChange         func(string)
}

type LocaleState struct {
	get       func() string
	set       func(string)
	direction func() Direction
	supported func() []string
	fallback  func() string
}

type ProviderProps struct {
	Locale         LocaleState
	CurrentLocale  string
	FallbackLocale string
	Bundle         *Bundle
	Child          ui.Node
	Children       []ui.Node
}

type Runtime struct {
	locale         func() string
	setLocale      func(string)
	direction      func() Direction
	fallbackLocale string
	bundle         *Bundle
}

type NumberOptions struct {
	MaximumFractionDigits int
}

type DateStyle string

const (
	DateStyleShort  DateStyle = "short"
	DateStyleMedium DateStyle = "medium"
	DateStyleLong   DateStyle = "long"
)

type DateOptions struct {
	Style    DateStyle
	Location *time.Location
}

type RouteOptions struct {
	SupportedLocales  []string
	DefaultLocale     string
	OmitDefaultPrefix bool
}

type ResolvedPath struct {
	Locale        string
	BasePath      string
	LocalizedPath string
	PrefixPresent bool
}

type SSRBootstrapOptions struct {
	Locale            string
	FallbackLocale    string
	Direction         Direction
	IncludeLocales    []string
	IncludeNamespaces []string
}

var runtimeContext = ui.CreateContext(Runtime{bundle: NewBundle()})

func (s LocaleState) Get() string {
	if s.get == nil {
		return ""
	}
	return s.get()
}

func (s LocaleState) Set(locale string) {
	if s.set != nil {
		s.set(locale)
	}
}

func (s LocaleState) Direction() Direction {
	if s.direction == nil {
		return DirectionLTR
	}
	return s.direction()
}

func (s LocaleState) SupportedLocales() []string {
	if s.supported == nil {
		return nil
	}
	return s.supported()
}

func (s LocaleState) FallbackLocale() string {
	if s.fallback == nil {
		return ""
	}
	return s.fallback()
}

// NewBundle creates a new i18n Bundle with the given options.
func NewBundle(options ...BundleOptions) *Bundle {
	resolved := BundleOptions{}
	if len(options) > 0 {
		resolved = options[0]
	}
	return &Bundle{
		defaultLocale:  NormalizeLocale(resolved.DefaultLocale),
		fallbackLocale: NormalizeLocale(resolved.FallbackLocale),
		onMissing:      resolved.OnMissing,
		catalogs:       map[string]Catalog{},
	}
}

func (b *Bundle) Register(locale string, catalog Catalog) {
	if b == nil {
		return
	}
	key := NormalizeLocale(locale)
	if key == "" {
		return
	}
	if b.catalogs == nil {
		b.catalogs = map[string]Catalog{}
	}
	if _, ok := b.catalogs[key]; !ok {
		b.catalogs[key] = Catalog{}
	}
	for namespace, entries := range catalog {
		if _, ok := b.catalogs[key][namespace]; !ok {
			b.catalogs[key][namespace] = NamespaceCatalog{}
		}
		for messageKey, entry := range entries {
			b.catalogs[key][namespace][messageKey] = cloneMessage(entry)
		}
	}
	if b.defaultLocale == "" {
		b.defaultLocale = key
	}
	if b.fallbackLocale == "" {
		b.fallbackLocale = key
	}
}

func (b *Bundle) RegisterNamespace(locale string, namespace string, entries NamespaceCatalog) {
	b.Register(locale, Catalog{namespace: entries})
}

func (b *Bundle) DefaultLocale() string {
	if b == nil {
		return ""
	}
	return b.defaultLocale
}

func (b *Bundle) FallbackLocale() string {
	if b == nil {
		return ""
	}
	return b.fallbackLocale
}

func (b *Bundle) Locales() []string {
	if b == nil {
		return nil
	}
	locales := make([]string, 0, len(b.catalogs))
	for locale := range b.catalogs {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	return locales
}

// Provider renders an i18n runtime context provider around its children.
func Provider(props ProviderProps) ui.Node {
	bundle := props.Bundle
	if bundle == nil {
		bundle = NewBundle()
	}
	localeHandle := props.Locale
	current := props.CurrentLocale
	if localeHandle.get != nil {
		current = localeHandle.Get()
	}
	if current == "" {
		current = bundle.DefaultLocale()
	}
	fallback := props.FallbackLocale
	if localeHandle.fallback != nil {
		fallback = localeHandle.FallbackLocale()
	}
	if fallback == "" {
		fallback = bundle.FallbackLocale()
	}
	runtime := Runtime{
		locale: func() string {
			if localeHandle.get != nil {
				return localeHandle.Get()
			}
			return current
		},
		setLocale: func(next string) {
			if localeHandle.set != nil {
				localeHandle.Set(next)
			}
		},
		direction: func() Direction {
			if localeHandle.direction != nil {
				return localeHandle.Direction()
			}
			return DirectionForLocale(current)
		},
		fallbackLocale: fallback,
		bundle:         bundle,
	}
	children := make([]ui.Node, 0, len(props.Children)+1)
	if props.Child != nil {
		children = append(children, props.Child)
	}
	children = append(children, props.Children...)
	return ui.CreateElement(runtimeContext.Provider, ui.ContextProviderProps[Runtime]{
		Value:    runtime,
		Children: children,
	})
}

// UseI18n returns the i18n Runtime from the nearest Provider ancestor.
func UseI18n() Runtime {
	resolved := ui.UseContext(runtimeContext)
	if resolved.bundle == nil {
		resolved.bundle = NewBundle()
	}
	return resolved
}

func (r Runtime) Locale() string {
	if r.locale == nil {
		return ""
	}
	return r.locale()
}

func (r Runtime) SetLocale(locale string) {
	if r.setLocale != nil {
		r.setLocale(locale)
	}
}

func (r Runtime) Direction() Direction {
	if r.direction == nil {
		return DirectionLTR
	}
	return r.direction()
}

func (r Runtime) T(namespace string, key string, args ...Arguments) string {
	resolvedArgs := Arguments{}
	if len(args) > 0 {
		resolvedArgs = args[0]
	}
	if r.bundle == nil {
		return defaultMissingText(r.Locale(), namespace, key)
	}
	return r.bundle.Translate(r.Locale(), namespace, key, resolvedArgs, r.fallbackLocale)
}

func (r Runtime) FormatNumber(value float64, options ...NumberOptions) string {
	return FormatNumber(r.Locale(), value, options...)
}

func (r Runtime) FormatDate(value time.Time, options ...DateOptions) string {
	return FormatDate(r.Locale(), value, options...)
}

func (r Runtime) PrefixPath(path string, options ...RouteOptions) string {
	resolved := RouteOptions{}
	if len(options) > 0 {
		resolved = options[0]
	}
	if resolved.DefaultLocale == "" {
		resolved.DefaultLocale = r.fallbackLocale
	}
	return PrefixPath(r.Locale(), path, resolved)
}

func (b *Bundle) Translate(locale string, namespace string, key string, args Arguments, fallbackLocale string) string {
	if b == nil {
		return defaultMissingText(locale, namespace, key)
	}
	entry, ok := b.lookup(locale, namespace, key, fallbackLocale)
	if !ok {
		if b.onMissing != nil {
			return b.onMissing(locale, namespace, key)
		}
		return defaultMissingText(locale, namespace, key)
	}
	template := resolveTemplate(locale, entry, args)
	return interpolateTemplate(template, args)
}

// NormalizeLocale parses and canonicalizes a BCP 47 locale tag.
func NormalizeLocale(raw string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(raw, "_", "-"))
	if trimmed == "" {
		return ""
	}
	parsed, err := language.Parse(trimmed)
	if err != nil {
		return trimmed
	}
	return parsed.String()
}

// DirectionForLocale returns DirectionRTL for right-to-left locales, otherwise DirectionLTR.
func DirectionForLocale(locale string) Direction {
	primary := strings.ToLower(primaryLanguage(NormalizeLocale(locale)))
	switch primary {
	case "ar", "fa", "he", "ur", "ps", "sd", "ku":
		return DirectionRTL
	default:
		return DirectionLTR
	}
}

// FormatNumber formats value as a locale-aware number string.
func FormatNumber(locale string, value float64, options ...NumberOptions) string {
	resolved := NumberOptions{MaximumFractionDigits: -1}
	if len(options) > 0 {
		resolved = options[0]
	}
	if resolved.MaximumFractionDigits < 0 {
		if value == float64(int64(value)) {
			resolved.MaximumFractionDigits = 0
		} else {
			resolved.MaximumFractionDigits = 2
		}
	}
	printer := message.NewPrinter(language.Make(fallbackString(NormalizeLocale(locale), "en")))
	format := "%0." + strconv.Itoa(resolved.MaximumFractionDigits) + "f"
	return printer.Sprintf(format, value)
}

// FormatDate formats value as a locale-aware date string.
func FormatDate(locale string, value time.Time, options ...DateOptions) string {
	resolved := DateOptions{Style: DateStyleMedium}
	if len(options) > 0 {
		resolved = options[0]
	}
	if resolved.Location != nil {
		value = value.In(resolved.Location)
	}
	switch localeFamily(locale) {
	case "fr":
		switch resolved.Style {
		case DateStyleShort:
			return value.Format("02/01/2006")
		case DateStyleLong:
			return value.Format("2 January 2006")
		default:
			return value.Format("2 Jan 2006")
		}
	case "de":
		switch resolved.Style {
		case DateStyleShort:
			return value.Format("02.01.2006")
		case DateStyleLong:
			return value.Format("2. January 2006")
		default:
			return value.Format("2. Jan 2006")
		}
	case "ar":
		switch resolved.Style {
		case DateStyleLong:
			return value.Format("02 Jan 2006")
		default:
			return value.Format("02/01/2006")
		}
	default:
		switch resolved.Style {
		case DateStyleShort:
			return value.Format("01/02/2006")
		case DateStyleLong:
			return value.Format("January 2, 2006")
		default:
			return value.Format("Jan 2, 2006")
		}
	}
}

// ResolvePath extracts the locale prefix from path and returns routing metadata.
func ResolvePath(path string, options RouteOptions) ResolvedPath {
	normalizedPath, suffix := splitPathAndQuery(path)
	resolved := normalizeRouteOptions(options)
	trimmed := strings.Trim(strings.TrimPrefix(normalizedPath, "/"), " ")
	segments := []string{}
	if trimmed != "" {
		segments = strings.Split(trimmed, "/")
	}
	locale := resolved.DefaultLocale
	prefixPresent := false
	basePath := normalizedPath
	if len(segments) > 0 {
		candidate := NormalizeLocale(segments[0])
		if localeAllowed(candidate, resolved.SupportedLocales) {
			locale = candidate
			prefixPresent = true
			remaining := strings.Join(segments[1:], "/")
			if remaining == "" {
				basePath = "/"
			} else {
				basePath = "/" + remaining
			}
		}
	}
	if locale == "" {
		locale = resolved.DefaultLocale
	}
	localizedPath := PrefixPath(locale, basePath+suffix, resolved)
	return ResolvedPath{
		Locale:        locale,
		BasePath:      basePath,
		LocalizedPath: localizedPath,
		PrefixPresent: prefixPresent,
	}
}

// PrefixPath prepends the locale prefix to path according to the route options.
func PrefixPath(locale string, path string, options RouteOptions) string {
	resolved := normalizeRouteOptions(options)
	localized := normalizeLeadingPath(path)
	if localized == "" {
		localized = "/"
	}
	resolvedLocale := chooseSupportedLocale(locale, resolved.SupportedLocales, resolved.DefaultLocale)
	if resolved.OmitDefaultPrefix && resolvedLocale == resolved.DefaultLocale {
		return localized
	}
	if localized == "/" {
		return "/" + resolvedLocale
	}
	return "/" + resolvedLocale + localized
}

func (b *Bundle) ToSSRBootstrap(options SSRBootstrapOptions) ui.SSRI18nBootstrap {
	if b == nil {
		return ui.SSRI18nBootstrap{}
	}
	locale := chooseSupportedLocale(options.Locale, b.Locales(), fallbackString(options.FallbackLocale, b.fallbackLocale))
	includeLocales := options.IncludeLocales
	if len(includeLocales) == 0 {
		includeLocales = []string{locale, fallbackString(options.FallbackLocale, b.fallbackLocale)}
	}
	includeNamespaces := make(map[string]struct{}, len(options.IncludeNamespaces))
	for _, namespace := range options.IncludeNamespaces {
		includeNamespaces[namespace] = struct{}{}
	}
	messages := map[string]map[string]ui.SSRI18nMessage{}
	for _, candidate := range normalizeLocales(includeLocales) {
		catalog, ok := b.catalogs[candidate]
		if !ok {
			continue
		}
		if _, exists := messages[candidate]; !exists {
			messages[candidate] = map[string]ui.SSRI18nMessage{}
		}
		for namespace, entries := range catalog {
			if len(includeNamespaces) > 0 {
				if _, ok := includeNamespaces[namespace]; !ok {
					continue
				}
			}
			for key, entry := range entries {
				messages[candidate][namespace+"."+key] = ui.SSRI18nMessage{
					Text:      entry.Text,
					PluralArg: entry.PluralArg,
					SelectArg: entry.SelectArg,
					Default:   entry.Default,
					Plural:    pluralToRaw(entry.Plural),
					Select:    cloneSelect(entry.Select),
				}
			}
		}
	}
	direction := options.Direction
	if direction == "" {
		direction = DirectionForLocale(locale)
	}
	return ui.SSRI18nBootstrap{
		Locale:         locale,
		FallbackLocale: fallbackString(options.FallbackLocale, b.fallbackLocale),
		Direction:      string(direction),
		Messages:       messages,
	}
}

// BundleFromSSRBootstrap reconstructs a Bundle from an SSR bootstrap payload.
func BundleFromSSRBootstrap(payload ui.SSRI18nBootstrap) *Bundle {
	bundle := NewBundle(BundleOptions{DefaultLocale: payload.Locale, FallbackLocale: payload.FallbackLocale})
	for locale, entries := range payload.Messages {
		catalog := Catalog{}
		for combinedKey, entry := range entries {
			namespace, messageKey := splitCombinedMessageKey(combinedKey)
			if _, ok := catalog[namespace]; !ok {
				catalog[namespace] = NamespaceCatalog{}
			}
			catalog[namespace][messageKey] = Message{
				Text:      entry.Text,
				PluralArg: entry.PluralArg,
				SelectArg: entry.SelectArg,
				Default:   entry.Default,
				Plural:    rawToPlural(entry.Plural),
				Select:    cloneSelect(entry.Select),
			}
		}
		bundle.Register(locale, catalog)
	}
	return bundle
}

func (b *Bundle) lookup(locale string, namespace string, key string, fallbackLocale string) (Message, bool) {
	for _, candidate := range localeCandidates(locale, fallbackString(fallbackLocale, b.fallbackLocale), b.defaultLocale) {
		catalog, ok := b.catalogs[candidate]
		if !ok {
			continue
		}
		entries, ok := catalog[namespace]
		if !ok {
			continue
		}
		entry, ok := entries[key]
		if ok {
			return cloneMessage(entry), true
		}
	}
	return Message{}, false
}

func resolveTemplate(locale string, entry Message, args Arguments) string {
	if len(entry.Select) > 0 {
		selectorKey := fallbackString(entry.SelectArg, "select")
		selector := strings.TrimSpace(fmt.Sprint(args[selectorKey]))
		if template, ok := entry.Select[selector]; ok {
			return template
		}
		if template, ok := entry.Select["other"]; ok {
			return template
		}
		if entry.Default != "" {
			return entry.Default
		}
	}
	if len(entry.Plural) > 0 {
		pluralKey := fallbackString(entry.PluralArg, "count")
		value := numericArgument(args[pluralKey])
		category := pluralCategoryForLocale(locale, value)
		if template, ok := entry.Plural[category]; ok {
			return template
		}
		if template, ok := entry.Plural[PluralOther]; ok {
			return template
		}
		if entry.Default != "" {
			return entry.Default
		}
	}
	if entry.Text != "" {
		return entry.Text
	}
	return entry.Default
}

func interpolateTemplate(template string, args Arguments) string {
	resolved := template
	for key, value := range args {
		resolved = strings.ReplaceAll(resolved, "{"+key+"}", stringifyArgument(value))
	}
	return resolved
}

func stringifyArgument(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case time.Time:
		return typed.Format(time.RFC3339)
	default:
		return fmt.Sprint(value)
	}
}

func numericArgument(value interface{}) float64 {
	switch typed := value.(type) {
	case int:
		return float64(typed)
	case int8:
		return float64(typed)
	case int16:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case uint:
		return float64(typed)
	case uint8:
		return float64(typed)
	case uint16:
		return float64(typed)
	case uint32:
		return float64(typed)
	case uint64:
		return float64(typed)
	case float32:
		return float64(typed)
	case float64:
		return typed
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func pluralCategoryForLocale(locale string, value float64) PluralCategory {
	primary := localeFamily(locale)
	abs := value
	if abs < 0 {
		abs = -abs
	}
	integer := int(abs)
	mod10 := integer % 10
	mod100 := integer % 100
	switch primary {
	case "ar":
		switch {
		case integer == 0:
			return PluralZero
		case integer == 1:
			return PluralOne
		case integer == 2:
			return PluralTwo
		case mod100 >= 3 && mod100 <= 10:
			return PluralFew
		case mod100 >= 11 && mod100 <= 99:
			return PluralMany
		default:
			return PluralOther
		}
	case "fr", "pt":
		if integer == 0 || integer == 1 {
			return PluralOne
		}
		return PluralOther
	case "ru", "uk":
		switch {
		case mod10 == 1 && mod100 != 11:
			return PluralOne
		case mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14):
			return PluralFew
		case mod10 == 0 || (mod10 >= 5 && mod10 <= 9) || (mod100 >= 11 && mod100 <= 14):
			return PluralMany
		default:
			return PluralOther
		}
	case "pl":
		switch {
		case integer == 1:
			return PluralOne
		case mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14):
			return PluralFew
		case integer != 1 && (mod10 == 0 || mod10 == 1 || mod10 >= 5 || (mod100 >= 12 && mod100 <= 14)):
			return PluralMany
		default:
			return PluralOther
		}
	case "cs", "sk":
		switch {
		case integer == 1:
			return PluralOne
		case integer >= 2 && integer <= 4:
			return PluralFew
		default:
			return PluralOther
		}
	case "ja", "ko", "zh", "th", "vi", "tr":
		return PluralOther
	default:
		if integer == 1 {
			return PluralOne
		}
		return PluralOther
	}
}

func localeCandidates(locale string, fallbackLocale string, defaultLocale string) []string {
	candidates := []string{}
	for _, raw := range []string{locale, primaryLanguage(locale), fallbackLocale, primaryLanguage(fallbackLocale), defaultLocale, primaryLanguage(defaultLocale)} {
		normalized := NormalizeLocale(raw)
		if normalized == "" {
			continue
		}
		already := false
		for _, existing := range candidates {
			if existing == normalized {
				already = true
				break
			}
		}
		if !already {
			candidates = append(candidates, normalized)
		}
	}
	return candidates
}

func localeFamily(locale string) string {
	return strings.ToLower(primaryLanguage(locale))
}

func primaryLanguage(locale string) string {
	normalized := NormalizeLocale(locale)
	if normalized == "" {
		return ""
	}
	parts := strings.Split(normalized, "-")
	return parts[0]
}

func normalizeLocales(locales []string) []string {
	result := make([]string, 0, len(locales))
	seen := map[string]struct{}{}
	for _, locale := range locales {
		normalized := NormalizeLocale(locale)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func localeAllowed(locale string, supported []string) bool {
	if locale == "" {
		return false
	}
	if len(supported) == 0 {
		return true
	}
	primary := primaryLanguage(locale)
	for _, candidate := range normalizeLocales(supported) {
		if candidate == locale || primaryLanguage(candidate) == primary {
			return true
		}
	}
	return false
}

func chooseSupportedLocale(locale string, supported []string, fallbackLocale string) string {
	normalized := NormalizeLocale(locale)
	if localeAllowed(normalized, supported) {
		return normalized
	}
	primary := primaryLanguage(normalized)
	for _, candidate := range normalizeLocales(supported) {
		if primaryLanguage(candidate) == primary {
			return candidate
		}
	}
	return fallbackString(fallbackLocale, normalized)
}

func normalizeRouteOptions(options RouteOptions) RouteOptions {
	options.DefaultLocale = NormalizeLocale(options.DefaultLocale)
	options.SupportedLocales = normalizeLocales(options.SupportedLocales)
	if options.DefaultLocale == "" && len(options.SupportedLocales) > 0 {
		options.DefaultLocale = options.SupportedLocales[0]
	}
	return options
}

func splitPathAndQuery(path string) (string, string) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/", ""
	}
	parts := strings.SplitN(trimmed, "?", 2)
	base := normalizeLeadingPath(parts[0])
	if len(parts) == 2 {
		return base, "?" + parts[1]
	}
	return base, ""
}

func normalizeLeadingPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	if len(trimmed) > 1 {
		trimmed = strings.TrimRight(trimmed, "/")
	}
	if trimmed == "" {
		return "/"
	}
	return trimmed
}

func fallbackString(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return NormalizeLocale(value)
	}
	return NormalizeLocale(fallback)
}

func defaultMissingText(locale string, namespace string, key string) string {
	if namespace == "" {
		return key
	}
	return namespace + "." + key
}

func cloneMessage(message Message) Message {
	return Message{
		Text:      message.Text,
		PluralArg: message.PluralArg,
		Plural:    clonePlural(message.Plural),
		SelectArg: message.SelectArg,
		Select:    cloneSelect(message.Select),
		Default:   message.Default,
	}
}

func clonePlural(source map[PluralCategory]string) map[PluralCategory]string {
	if len(source) == 0 {
		return nil
	}
	clone := make(map[PluralCategory]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func pluralToRaw(source map[PluralCategory]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[string(key)] = value
	}
	return clone
}

func rawToPlural(source map[string]string) map[PluralCategory]string {
	if len(source) == 0 {
		return nil
	}
	clone := make(map[PluralCategory]string, len(source))
	for key, value := range source {
		clone[PluralCategory(key)] = value
	}
	return clone
}

func cloneSelect(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func splitCombinedMessageKey(value string) (string, string) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) == 1 {
		return "default", parts[0]
	}
	return parts[0], parts[1]
}
