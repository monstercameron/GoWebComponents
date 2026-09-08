package i18n

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/ui"
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
type Arguments map[string]any
type MissingHandler func(locale string, namespace string, key string) string

type BundleOptions struct {
	DefaultLocale  string
	FallbackLocale string
	OnMissing      MissingHandler
}

type Bundle struct {
	// mu guards catalogs (and the default/fallback locale fields written during
	// Register). A *Bundle is commonly shared across concurrent SSR requests (and
	// LazyBundle.EnsureLocale, which promises concurrency-safety, mutates it), so
	// an unsynchronized Register racing a lookup is a FATAL, unrecoverable Go
	// "concurrent map read and map write" — never held across a user callback.
	mu             sync.RWMutex
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

func (parseS LocaleState) Get() string {
	if parseS.get == nil {
		return ""
	}
	return parseS.get()
}

func (parseS LocaleState) Set(parseLocale string) {
	if parseS.set != nil {
		parseS.set(parseLocale)
	}
}

func (parseS LocaleState) Direction() Direction {
	if parseS.direction == nil {
		return DirectionLTR
	}
	return parseS.direction()
}

func (parseS LocaleState) SupportedLocales() []string {
	if parseS.supported == nil {
		return nil
	}
	return parseS.supported()
}

func (parseS LocaleState) FallbackLocale() string {
	if parseS.fallback == nil {
		return ""
	}
	return parseS.fallback()
}

// NewBundle creates a new i18n Bundle with the given options.
func NewBundle(parseOptions ...BundleOptions) *Bundle {
	parseResolved := BundleOptions{}
	if len(parseOptions) > 0 {
		parseResolved = parseOptions[0]
	}
	return &Bundle{
		defaultLocale:  NormalizeLocale(parseResolved.DefaultLocale),
		fallbackLocale: NormalizeLocale(parseResolved.FallbackLocale),
		onMissing:      parseResolved.OnMissing,
		catalogs:       map[string]Catalog{},
	}
}

func (parseB *Bundle) Register(parseLocale string, parseCatalog Catalog) {
	if parseB == nil {
		return
	}
	parseKey := NormalizeLocale(parseLocale)
	if parseKey == "" {
		return
	}
	parseB.mu.Lock()
	defer parseB.mu.Unlock()
	if parseB.catalogs == nil {
		parseB.catalogs = map[string]Catalog{}
	}
	if _, parseOk := parseB.catalogs[parseKey]; !parseOk {
		parseB.catalogs[parseKey] = Catalog{}
	}
	for parseNamespace, parseEntries := range parseCatalog {
		if _, parseOk2 := parseB.catalogs[parseKey][parseNamespace]; !parseOk2 {
			parseB.catalogs[parseKey][parseNamespace] = NamespaceCatalog{}
		}
		for parseMessageKey, parseEntry := range parseEntries {
			parseB.catalogs[parseKey][parseNamespace][parseMessageKey] = cloneMessage(parseEntry)
		}
	}
	if parseB.defaultLocale == "" {
		parseB.defaultLocale = parseKey
	}
	if parseB.fallbackLocale == "" {
		parseB.fallbackLocale = parseKey
	}
}

func (parseB *Bundle) RegisterNamespace(parseLocale string, parseNamespace string, parseEntries NamespaceCatalog) {
	parseB.Register(parseLocale, Catalog{parseNamespace: parseEntries})
}

func (parseB *Bundle) DefaultLocale() string {
	if parseB == nil {
		return ""
	}
	return parseB.defaultLocale
}

func (parseB *Bundle) FallbackLocale() string {
	if parseB == nil {
		return ""
	}
	return parseB.fallbackLocale
}

func (parseB *Bundle) Locales() []string {
	if parseB == nil {
		return nil
	}
	parseB.mu.RLock()
	parseLocales := make([]string, 0, len(parseB.catalogs))
	for parseLocale := range parseB.catalogs {
		parseLocales = append(parseLocales, parseLocale)
	}
	parseB.mu.RUnlock()
	sort.Strings(parseLocales)
	return parseLocales
}

// Provider renders an i18n runtime context provider around its children.
func Provider(parseProps ProviderProps) ui.Node {
	parseBundle := parseProps.Bundle
	if parseBundle == nil {
		parseBundle = NewBundle()
	}
	parseLocaleHandle := parseProps.Locale
	parseCurrent := parseProps.CurrentLocale
	if parseLocaleHandle.get != nil {
		parseCurrent = parseLocaleHandle.Get()
	}
	if parseCurrent == "" {
		parseCurrent = parseBundle.DefaultLocale()
	}
	parseFallback := parseProps.FallbackLocale
	if parseLocaleHandle.fallback != nil {
		parseFallback = parseLocaleHandle.FallbackLocale()
	}
	if parseFallback == "" {
		parseFallback = parseBundle.FallbackLocale()
	}
	parseRuntime := Runtime{
		locale: func() string {
			if parseLocaleHandle.get != nil {
				return parseLocaleHandle.Get()
			}
			return parseCurrent
		},
		setLocale: func(parseNext string) {
			if parseLocaleHandle.set != nil {
				parseLocaleHandle.Set(parseNext)
			}
		},
		direction: func() Direction {
			if parseLocaleHandle.direction != nil {
				return parseLocaleHandle.Direction()
			}
			return DirectionForLocale(parseCurrent)
		},
		fallbackLocale: parseFallback,
		bundle:         parseBundle,
	}
	parseChildren := make([]ui.Node, 0, len(parseProps.Children)+1)
	if parseProps.Child != nil {
		parseChildren = append(parseChildren, parseProps.Child)
	}
	parseChildren = append(parseChildren, parseProps.Children...)
	return ui.CreateElement(runtimeContext.Provider, ui.ContextProviderProps[Runtime]{
		Value:    parseRuntime,
		Children: parseChildren,
	})
}

// UseI18n returns the i18n Runtime from the nearest Provider ancestor.
func UseI18n() Runtime {
	parseResolved := ui.UseContext(runtimeContext)
	if parseResolved.bundle == nil {
		parseResolved.bundle = NewBundle()
	}
	return parseResolved
}

func (parseR Runtime) Locale() string {
	if parseR.locale == nil {
		return ""
	}
	return parseR.locale()
}

func (parseR Runtime) SetLocale(parseLocale string) {
	if parseR.setLocale != nil {
		parseR.setLocale(parseLocale)
	}
}

func (parseR Runtime) Direction() Direction {
	if parseR.direction == nil {
		return DirectionLTR
	}
	return parseR.direction()
}

func (parseR Runtime) T(parseNamespace string, parseKey string, parseArgs ...Arguments) string {
	parseResolvedArgs := Arguments{}
	if len(parseArgs) > 0 {
		parseResolvedArgs = parseArgs[0]
	}
	if parseR.bundle == nil {
		return defaultMissingText(parseR.Locale(), parseNamespace, parseKey)
	}
	return parseR.bundle.Translate(parseR.Locale(), parseNamespace, parseKey, parseResolvedArgs, parseR.fallbackLocale)
}

// Namespace binds a Runtime to a single namespace so its translations are looked up by key
// alone. It removes the per-call footgun of T(namespace, key, …) where swapping or forgetting the
// positional namespace silently misses; the namespace is fixed once at the NS() call site.
type Namespace struct {
	runtime   Runtime
	namespace string
}

// NS returns a Namespace handle bound to namespace, so a component that translates many keys in
// the same namespace writes t := r.NS("checkout"); t.T("title"); t.T("total", args) instead of
// repeating the namespace (and risking a typo) at every call.
func (parseR Runtime) NS(parseNamespace string) Namespace {
	return Namespace{runtime: parseR, namespace: parseNamespace}
}

// T translates key within the bound namespace, with the same args/fallback semantics as Runtime.T.
func (parseN Namespace) T(parseKey string, parseArgs ...Arguments) string {
	return parseN.runtime.T(parseN.namespace, parseKey, parseArgs...)
}

// Name returns the bound namespace.
func (parseN Namespace) Name() string { return parseN.namespace }

func (parseR Runtime) FormatNumber(parseValue float64, parseOptions ...NumberOptions) string {
	return FormatNumber(parseR.Locale(), parseValue, parseOptions...)
}

func (parseR Runtime) FormatDate(parseValue time.Time, parseOptions ...DateOptions) string {
	return FormatDate(parseR.Locale(), parseValue, parseOptions...)
}

func (parseR Runtime) PrefixPath(parsePath string, parseOptions ...RouteOptions) string {
	parseResolved := RouteOptions{}
	if len(parseOptions) > 0 {
		parseResolved = parseOptions[0]
	}
	if parseResolved.DefaultLocale == "" {
		parseResolved.DefaultLocale = parseR.fallbackLocale
	}
	return PrefixPath(parseR.Locale(), parsePath, parseResolved)
}

func (parseB *Bundle) Translate(parseLocale string, parseNamespace string, parseKey string, parseArgs Arguments, parseFallbackLocale string) string {
	if parseB == nil {
		return defaultMissingText(parseLocale, parseNamespace, parseKey)
	}
	parseEntry, parseOk := parseB.lookup(parseLocale, parseNamespace, parseKey, parseFallbackLocale)
	if !parseOk {
		if parseB.onMissing != nil {
			return parseB.onMissing(parseLocale, parseNamespace, parseKey)
		}
		return defaultMissingText(parseLocale, parseNamespace, parseKey)
	}
	parseTemplate := resolveTemplate(parseLocale, parseEntry, parseArgs)
	return interpolateTemplate(parseTemplate, parseArgs)
}

// NormalizeLocale parses and canonicalizes a BCP 47 locale tag.
func NormalizeLocale(parseRaw string) string {
	parseTrimmed := strings.TrimSpace(strings.ReplaceAll(parseRaw, "_", "-"))
	if parseTrimmed == "" {
		return ""
	}
	parseParsed, parseErr := language.Parse(parseTrimmed)
	if parseErr != nil {
		return parseTrimmed
	}
	return parseParsed.String()
}

// DirectionForLocale returns DirectionRTL for right-to-left locales, otherwise DirectionLTR.
func DirectionForLocale(parseLocale string) Direction {
	parsePrimary := strings.ToLower(primaryLanguage(NormalizeLocale(parseLocale)))
	switch parsePrimary {
	case "ar", "fa", "he", "ur", "ps", "sd", "ku":
		return DirectionRTL
	default:
		return DirectionLTR
	}
}

// FormatNumber formats value as a locale-aware number string.
func FormatNumber(parseLocale string, parseValue float64, parseOptions ...NumberOptions) string {
	parseResolved := NumberOptions{MaximumFractionDigits: -1}
	if len(parseOptions) > 0 {
		parseResolved = parseOptions[0]
	}
	if parseResolved.MaximumFractionDigits < 0 {
		if parseValue == float64(int64(parseValue)) {
			parseResolved.MaximumFractionDigits = 0
		} else {
			parseResolved.MaximumFractionDigits = 2
		}
	}
	parsePrinter := message.NewPrinter(language.Make(fallbackString(NormalizeLocale(parseLocale), "en")))
	format := "%0." + strconv.Itoa(parseResolved.MaximumFractionDigits) + "f"
	return parsePrinter.Sprintf(format, parseValue)
}

// FormatDate formats value as a locale-aware date string.
func FormatDate(parseLocale string, parseValue time.Time, parseOptions ...DateOptions) string {
	parseResolved := DateOptions{Style: DateStyleMedium}
	if len(parseOptions) > 0 {
		parseResolved = parseOptions[0]
	}
	if parseResolved.Location != nil {
		parseValue = parseValue.In(parseResolved.Location)
	}
	switch localeFamily(parseLocale) {
	case "fr":
		switch parseResolved.Style {
		case DateStyleShort:
			return parseValue.Format("02/01/2006")
		case DateStyleLong:
			return parseValue.Format("2 January 2006")
		default:
			return parseValue.Format("2 Jan 2006")
		}
	case "de":
		switch parseResolved.Style {
		case DateStyleShort:
			return parseValue.Format("02.01.2006")
		case DateStyleLong:
			return parseValue.Format("2. January 2006")
		default:
			return parseValue.Format("2. Jan 2006")
		}
	case "ar":
		switch parseResolved.Style {
		case DateStyleLong:
			return parseValue.Format("02 Jan 2006")
		default:
			return parseValue.Format("02/01/2006")
		}
	default:
		switch parseResolved.Style {
		case DateStyleShort:
			return parseValue.Format("01/02/2006")
		case DateStyleLong:
			return parseValue.Format("January 2, 2006")
		default:
			return parseValue.Format("Jan 2, 2006")
		}
	}
}

// ResolvePath extracts the locale prefix from path and returns routing metadata.
func ResolvePath(parsePath string, parseOptions RouteOptions) ResolvedPath {
	parseNormalizedPath, parseSuffix := splitPathAndQuery(parsePath)
	parseResolved := normalizeRouteOptions(parseOptions)
	parseTrimmed := strings.Trim(strings.TrimPrefix(parseNormalizedPath, "/"), " ")
	parseSegments := []string{}
	if parseTrimmed != "" {
		parseSegments = strings.Split(parseTrimmed, "/")
	}
	parseLocale := parseResolved.DefaultLocale
	isParsePrefixPresent := false
	parseBasePath := parseNormalizedPath
	if len(parseSegments) > 0 {
		parseCandidate := NormalizeLocale(parseSegments[0])
		if localeAllowed(parseCandidate, parseResolved.SupportedLocales) {
			parseLocale = chooseRoutePrefixLocale(parseCandidate, parseResolved.SupportedLocales, parseResolved.DefaultLocale)
			isParsePrefixPresent = true
			parseRemaining := strings.Join(parseSegments[1:], "/")
			if parseRemaining == "" {
				parseBasePath = "/"
			} else {
				parseBasePath = "/" + parseRemaining
			}
		}
	}
	if parseLocale == "" {
		parseLocale = parseResolved.DefaultLocale
	}
	parseLocalizedPath := PrefixPath(parseLocale, parseBasePath+parseSuffix, parseResolved)
	return ResolvedPath{
		Locale:        parseLocale,
		BasePath:      parseBasePath,
		LocalizedPath: parseLocalizedPath,
		PrefixPresent: isParsePrefixPresent,
	}
}

// PrefixPath prepends the locale prefix to path according to the route options.
func PrefixPath(parseLocale string, parsePath string, parseOptions RouteOptions) string {
	parseResolved := normalizeRouteOptions(parseOptions)
	parseLocalized := normalizeLeadingPath(parsePath)
	if parseLocalized == "" {
		parseLocalized = "/"
	}
	parseResolvedLocale := chooseSupportedLocale(parseLocale, parseResolved.SupportedLocales, parseResolved.DefaultLocale)
	if parseResolved.OmitDefaultPrefix && parseResolvedLocale == parseResolved.DefaultLocale {
		return parseLocalized
	}
	if parseLocalized == "/" {
		return "/" + parseResolvedLocale
	}
	return "/" + parseResolvedLocale + parseLocalized
}

func (parseB *Bundle) ToSSRBootstrap(parseOptions SSRBootstrapOptions) ui.SSRI18nBootstrap {
	if parseB == nil {
		return ui.SSRI18nBootstrap{}
	}
	parseLocale := chooseSupportedLocale(parseOptions.Locale, parseB.Locales(), fallbackString(parseOptions.FallbackLocale, parseB.fallbackLocale))
	parseIncludeLocales := parseOptions.IncludeLocales
	if len(parseIncludeLocales) == 0 {
		parseIncludeLocales = []string{parseLocale, fallbackString(parseOptions.FallbackLocale, parseB.fallbackLocale)}
	}
	parseIncludeNamespaces := make(map[string]struct{}, len(parseOptions.IncludeNamespaces))
	for _, parseNamespace := range parseOptions.IncludeNamespaces {
		parseIncludeNamespaces[parseNamespace] = struct{}{}
	}
	parseMessages := map[string]map[string]ui.SSRI18nMessage{}
	// RLock the direct catalog iteration. Locales() above already locked+released,
	// so there is no nested-lock here.
	parseB.mu.RLock()
	defer parseB.mu.RUnlock()
	for _, parseCandidate := range normalizeLocales(parseIncludeLocales) {
		parseCatalog, parseOk := parseB.catalogs[parseCandidate]
		if !parseOk {
			continue
		}
		if _, parseExists := parseMessages[parseCandidate]; !parseExists {
			parseMessages[parseCandidate] = map[string]ui.SSRI18nMessage{}
		}
		for parseNamespace2, parseEntries := range parseCatalog {
			if len(parseIncludeNamespaces) > 0 {
				if _, parseOk2 := parseIncludeNamespaces[parseNamespace2]; !parseOk2 {
					continue
				}
			}
			for parseKey, parseEntry := range parseEntries {
				parseMessages[parseCandidate][parseNamespace2+"."+parseKey] = ui.SSRI18nMessage{
					Text:      parseEntry.Text,
					PluralArg: parseEntry.PluralArg,
					SelectArg: parseEntry.SelectArg,
					Default:   parseEntry.Default,
					Plural:    pluralToRaw(parseEntry.Plural),
					Select:    cloneSelect(parseEntry.Select),
				}
			}
		}
	}
	parseDirection := parseOptions.Direction
	if parseDirection == "" {
		parseDirection = DirectionForLocale(parseLocale)
	}
	return ui.SSRI18nBootstrap{
		Locale:         parseLocale,
		FallbackLocale: fallbackString(parseOptions.FallbackLocale, parseB.fallbackLocale),
		Direction:      string(parseDirection),
		Messages:       parseMessages,
	}
}

// BundleFromSSRBootstrap reconstructs a Bundle from an SSR bootstrap payload.
func BundleFromSSRBootstrap(parsePayload ui.SSRI18nBootstrap) *Bundle {
	parseBundle := NewBundle(BundleOptions{DefaultLocale: parsePayload.Locale, FallbackLocale: parsePayload.FallbackLocale})
	for parseLocale, parseEntries := range parsePayload.Messages {
		parseCatalog := Catalog{}
		for parseCombinedKey, parseEntry := range parseEntries {
			parseNamespace, parseMessageKey := splitCombinedMessageKey(parseCombinedKey)
			if _, parseOk := parseCatalog[parseNamespace]; !parseOk {
				parseCatalog[parseNamespace] = NamespaceCatalog{}
			}
			parseCatalog[parseNamespace][parseMessageKey] = Message{
				Text:      parseEntry.Text,
				PluralArg: parseEntry.PluralArg,
				SelectArg: parseEntry.SelectArg,
				Default:   parseEntry.Default,
				Plural:    rawToPlural(parseEntry.Plural),
				Select:    cloneSelect(parseEntry.Select),
			}
		}
		parseBundle.Register(parseLocale, parseCatalog)
	}
	return parseBundle
}

func (parseB *Bundle) lookup(parseLocale string, parseNamespace string, parseKey string, parseFallbackLocale string) (Message, bool) {
	parseB.mu.RLock()
	defer parseB.mu.RUnlock()
	for _, parseCandidate := range localeCandidates(parseLocale, fallbackString(parseFallbackLocale, parseB.fallbackLocale), parseB.defaultLocale) {
		parseCatalog, parseOk := parseB.catalogs[parseCandidate]
		if !parseOk {
			continue
		}
		parseEntries, parseOk := parseCatalog[parseNamespace]
		if !parseOk {
			continue
		}
		parseEntry, parseOk := parseEntries[parseKey]
		if parseOk {
			return cloneMessage(parseEntry), true
		}
	}
	return Message{}, false
}

func resolveTemplate(parseLocale string, parseEntry Message, parseArgs Arguments) string {
	if len(parseEntry.Select) > 0 {
		parseSelectorKey := fallbackString(parseEntry.SelectArg, "select")
		parseSelector := strings.TrimSpace(fmt.Sprint(parseArgs[parseSelectorKey]))
		if parseTemplate, parseOk := parseEntry.Select[parseSelector]; parseOk {
			return parseTemplate
		}
		if parseTemplate2, parseOk2 := parseEntry.Select["other"]; parseOk2 {
			return parseTemplate2
		}
		if parseEntry.Default != "" {
			return parseEntry.Default
		}
	}
	if len(parseEntry.Plural) > 0 {
		parsePluralKey := fallbackString(parseEntry.PluralArg, "count")
		parseValue := numericArgument(parseArgs[parsePluralKey])
		parseCategory := pluralCategoryForLocale(parseLocale, parseValue)
		if parseTemplate3, parseOk3 := parseEntry.Plural[parseCategory]; parseOk3 {
			return parseTemplate3
		}
		if parseTemplate4, parseOk4 := parseEntry.Plural[PluralOther]; parseOk4 {
			return parseTemplate4
		}
		if parseEntry.Default != "" {
			return parseEntry.Default
		}
	}
	if parseEntry.Text != "" {
		return parseEntry.Text
	}
	return parseEntry.Default
}

func interpolateTemplate(parseTemplate string, parseArgs Arguments) string {
	if len(parseArgs) == 0 {
		return parseTemplate
	}
	// Single pass via a Replacer: sequential ReplaceAll re-scanned each
	// substituted value, so an argument whose value contained another key's
	// placeholder (e.g. name="{count}") interpolated nondeterministically with
	// Go's randomized map order. A Replacer never re-scans inserted text.
	parsePairs := make([]string, 0, len(parseArgs)*2)
	for parseKey, parseValue := range parseArgs {
		parsePairs = append(parsePairs, "{"+parseKey+"}", stringifyArgument(parseValue))
	}
	return strings.NewReplacer(parsePairs...).Replace(parseTemplate)
}

func stringifyArgument(parseValue any) string {
	switch parseTyped := parseValue.(type) {
	case string:
		return parseTyped
	case fmt.Stringer:
		return parseTyped.String()
	default:
		return fmt.Sprint(parseValue)
	}
}

func numericArgument(parseValue any) float64 {
	switch parseTyped := parseValue.(type) {
	case int:
		return float64(parseTyped)
	case int8:
		return float64(parseTyped)
	case int16:
		return float64(parseTyped)
	case int32:
		return float64(parseTyped)
	case int64:
		return float64(parseTyped)
	case uint:
		return float64(parseTyped)
	case uint8:
		return float64(parseTyped)
	case uint16:
		return float64(parseTyped)
	case uint32:
		return float64(parseTyped)
	case uint64:
		return float64(parseTyped)
	case float32:
		return float64(parseTyped)
	case float64:
		return parseTyped
	case string:
		parseParsed, parseErr := strconv.ParseFloat(strings.TrimSpace(parseTyped), 64)
		if parseErr == nil {
			return parseParsed
		}
	}
	return 0
}

func pluralCategoryForLocale(parseLocale string, parseValue float64) PluralCategory {
	parsePrimary := localeFamily(parseLocale)
	parseAbs := parseValue
	if parseAbs < 0 {
		parseAbs = -parseAbs
	}
	parseInteger := int(parseAbs)
	parseMod10 := parseInteger % 10
	parseMod100 := parseInteger % 100
	switch parsePrimary {
	case "ar":
		switch {
		case parseInteger == 0:
			return PluralZero
		case parseInteger == 1:
			return PluralOne
		case parseInteger == 2:
			return PluralTwo
		case parseMod100 >= 3 && parseMod100 <= 10:
			return PluralFew
		case parseMod100 >= 11 && parseMod100 <= 99:
			return PluralMany
		default:
			return PluralOther
		}
	case "fr", "pt":
		if parseInteger == 0 || parseInteger == 1 {
			return PluralOne
		}
		return PluralOther
	case "ru", "uk":
		switch {
		case parseMod10 == 1 && parseMod100 != 11:
			return PluralOne
		case parseMod10 >= 2 && parseMod10 <= 4 && (parseMod100 < 12 || parseMod100 > 14):
			return PluralFew
		case parseMod10 == 0 || (parseMod10 >= 5 && parseMod10 <= 9) || (parseMod100 >= 11 && parseMod100 <= 14):
			return PluralMany
		default:
			return PluralOther
		}
	case "pl":
		switch {
		case parseInteger == 1:
			return PluralOne
		case parseMod10 >= 2 && parseMod10 <= 4 && (parseMod100 < 12 || parseMod100 > 14):
			return PluralFew
		case parseInteger != 1 && (parseMod10 == 0 || parseMod10 == 1 || parseMod10 >= 5 || (parseMod100 >= 12 && parseMod100 <= 14)):
			return PluralMany
		default:
			return PluralOther
		}
	case "cs", "sk":
		switch {
		case parseInteger == 1:
			return PluralOne
		case parseInteger >= 2 && parseInteger <= 4:
			return PluralFew
		default:
			return PluralOther
		}
	case "ja", "ko", "zh", "th", "vi", "tr":
		return PluralOther
	default:
		if parseInteger == 1 {
			return PluralOne
		}
		return PluralOther
	}
}

// localeCandidateCache memoizes resolved candidate chains: NormalizeLocale
// re-parses BCP-47 tags through x/text on every call (~75% of Translate
// allocations), and apps use a handful of locale combinations at most.
//
// The key is the raw (un-normalized) locale, which is often derived from a URL
// segment or Accept-Language — i.e. request-controlled — so the cache is
// size-capped: past the cap, misses still resolve correctly, they just skip
// caching. The cap is far above any real app's locale count.
var (
	localeCandidateCache sync.Map // "locale|fallback|default" -> []string
	localeCandidateCount atomic.Int32
)

const maxLocaleCandidateEntries = 4096

func localeCandidates(parseLocale string, parseFallbackLocale string, parseDefaultLocale string) []string {
	parseCacheKey := parseLocale + "|" + parseFallbackLocale + "|" + parseDefaultLocale
	if parseCached, parseOk := localeCandidateCache.Load(parseCacheKey); parseOk {
		return parseCached.([]string)
	}
	parseResolved := buildLocaleCandidates(parseLocale, parseFallbackLocale, parseDefaultLocale)
	if localeCandidateCount.Load() < maxLocaleCandidateEntries {
		if _, parseLoaded := localeCandidateCache.LoadOrStore(parseCacheKey, parseResolved); !parseLoaded {
			localeCandidateCount.Add(1)
		}
	}
	return parseResolved
}

func buildLocaleCandidates(parseLocale string, parseFallbackLocale string, parseDefaultLocale string) []string {
	parseCandidates := []string{}
	for _, parseRaw := range []string{parseLocale, primaryLanguage(parseLocale), parseFallbackLocale, primaryLanguage(parseFallbackLocale), parseDefaultLocale, primaryLanguage(parseDefaultLocale)} {
		parseNormalized := NormalizeLocale(parseRaw)
		if parseNormalized == "" {
			continue
		}
		isParseAlready := slices.Contains(parseCandidates, parseNormalized)
		if !isParseAlready {
			parseCandidates = append(parseCandidates, parseNormalized)
		}
	}
	return parseCandidates
}

func localeFamily(parseLocale string) string {
	return strings.ToLower(primaryLanguage(parseLocale))
}

func primaryLanguage(parseLocale string) string {
	parseNormalized := NormalizeLocale(parseLocale)
	if parseNormalized == "" {
		return ""
	}
	parseParts := strings.Split(parseNormalized, "-")
	return parseParts[0]
}

func normalizeLocales(parseLocales []string) []string {
	parseResult := make([]string, 0, len(parseLocales))
	parseSeen := map[string]struct{}{}
	for _, parseLocale := range parseLocales {
		parseNormalized := NormalizeLocale(parseLocale)
		if parseNormalized == "" {
			continue
		}
		if _, parseOk := parseSeen[parseNormalized]; parseOk {
			continue
		}
		parseSeen[parseNormalized] = struct{}{}
		parseResult = append(parseResult, parseNormalized)
	}
	return parseResult
}

func localeAllowed(parseLocale string, parseSupported []string) bool {
	if parseLocale == "" {
		return false
	}
	if len(parseSupported) == 0 {
		return true
	}
	parsePrimary := primaryLanguage(parseLocale)
	for _, parseCandidate := range normalizeLocales(parseSupported) {
		if parseCandidate == parseLocale || primaryLanguage(parseCandidate) == parsePrimary {
			return true
		}
	}
	return false
}

func chooseSupportedLocale(parseLocale string, parseSupported []string, parseFallbackLocale string) string {
	parseNormalized := NormalizeLocale(parseLocale)
	if localeAllowed(parseNormalized, parseSupported) {
		return parseNormalized
	}
	parsePrimary := primaryLanguage(parseNormalized)
	for _, parseCandidate := range normalizeLocales(parseSupported) {
		if primaryLanguage(parseCandidate) == parsePrimary {
			return parseCandidate
		}
	}
	return fallbackString(parseFallbackLocale, parseNormalized)
}

func chooseRoutePrefixLocale(parseLocale string, parseSupported []string, parseFallbackLocale string) string {
	parseNormalized := NormalizeLocale(parseLocale)
	if parseNormalized == "" {
		return NormalizeLocale(parseFallbackLocale)
	}
	if len(parseSupported) == 0 {
		return parseNormalized
	}
	parseSupportedLocales := normalizeLocales(parseSupported)
	for _, parseCandidate := range parseSupportedLocales {
		if parseCandidate == parseNormalized {
			return parseCandidate
		}
	}
	parsePrimary := primaryLanguage(parseNormalized)
	for _, parseCandidate := range parseSupportedLocales {
		if primaryLanguage(parseCandidate) == parsePrimary {
			return parseCandidate
		}
	}
	return fallbackString(parseFallbackLocale, parseNormalized)
}

func normalizeRouteOptions(parseOptions RouteOptions) RouteOptions {
	parseOptions.DefaultLocale = NormalizeLocale(parseOptions.DefaultLocale)
	parseOptions.SupportedLocales = normalizeLocales(parseOptions.SupportedLocales)
	if parseOptions.DefaultLocale == "" && len(parseOptions.SupportedLocales) > 0 {
		parseOptions.DefaultLocale = parseOptions.SupportedLocales[0]
	}
	return parseOptions
}

func splitPathAndQuery(parsePath string) (string, string) {
	parseTrimmed := strings.TrimSpace(parsePath)
	if parseTrimmed == "" {
		return "/", ""
	}
	parseParts := strings.SplitN(parseTrimmed, "?", 2)
	parseBase := normalizeLeadingPath(parseParts[0])
	if len(parseParts) == 2 {
		return parseBase, "?" + parseParts[1]
	}
	return parseBase, ""
}

func normalizeLeadingPath(parsePath string) string {
	parseTrimmed := strings.TrimSpace(parsePath)
	if parseTrimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(parseTrimmed, "/") {
		parseTrimmed = "/" + parseTrimmed
	}
	if len(parseTrimmed) > 1 {
		parseTrimmed = strings.TrimRight(parseTrimmed, "/")
	}
	if parseTrimmed == "" {
		return "/"
	}
	return parseTrimmed
}

func fallbackString(parseValue string, parseFallback string) string {
	if strings.TrimSpace(parseValue) != "" {
		return NormalizeLocale(parseValue)
	}
	return NormalizeLocale(parseFallback)
}

func defaultMissingText(_ string, parseNamespace string, parseKey string) string {
	if parseNamespace == "" {
		return parseKey
	}
	return parseNamespace + "." + parseKey
}

func cloneMessage(parseMessage Message) Message {
	return Message{
		Text:      parseMessage.Text,
		PluralArg: parseMessage.PluralArg,
		Plural:    clonePlural(parseMessage.Plural),
		SelectArg: parseMessage.SelectArg,
		Select:    cloneSelect(parseMessage.Select),
		Default:   parseMessage.Default,
	}
}

func clonePlural(parseSource map[PluralCategory]string) map[PluralCategory]string {
	if len(parseSource) == 0 {
		return nil
	}
	parseClone := make(map[PluralCategory]string, len(parseSource))
	maps.Copy(parseClone, parseSource)
	return parseClone
}

func pluralToRaw(parseSource map[PluralCategory]string) map[string]string {
	if len(parseSource) == 0 {
		return nil
	}
	parseClone := make(map[string]string, len(parseSource))
	for parseKey, parseValue := range parseSource {
		parseClone[string(parseKey)] = parseValue
	}
	return parseClone
}

func rawToPlural(parseSource map[string]string) map[PluralCategory]string {
	if len(parseSource) == 0 {
		return nil
	}
	parseClone := make(map[PluralCategory]string, len(parseSource))
	for parseKey, parseValue := range parseSource {
		parseClone[PluralCategory(parseKey)] = parseValue
	}
	return parseClone
}

func cloneSelect(parseSource map[string]string) map[string]string {
	if len(parseSource) == 0 {
		return nil
	}
	parseClone := make(map[string]string, len(parseSource))
	maps.Copy(parseClone, parseSource)
	return parseClone
}

func splitCombinedMessageKey(parseValue string) (string, string) {
	parseParts := strings.SplitN(parseValue, ".", 2)
	if len(parseParts) == 1 {
		return "default", parseParts[0]
	}
	return parseParts[0], parseParts[1]
}
