package i18n

import (
	"strings"
	"time"

	"golang.org/x/text/language"
)

// dateLocale holds the month names and per-style layouts of one language.
//
// Layout tokens are {d} {dd} {M} {MM} {MMM} {MMMM} {yyyy}. MMMM and MMM are
// the full and abbreviated month names as they appear next to a day number
// (genitive for ru, pl and cs), which is what every layout here needs.
type dateLocale struct {
	monthsLong  [12]string
	monthsShort [12]string
	short       string
	medium      string
	long        string
}

var dateLocales = map[string]dateLocale{
	"fr": {
		monthsLong:  [12]string{"janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"},
		monthsShort: [12]string{"janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."},
		short:       "{dd}/{MM}/{yyyy}", medium: "{d} {MMM} {yyyy}", long: "{d} {MMMM} {yyyy}",
	},
	"de": {
		monthsLong:  [12]string{"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"},
		monthsShort: [12]string{"Jan.", "Feb.", "März", "Apr.", "Mai", "Juni", "Juli", "Aug.", "Sept.", "Okt.", "Nov.", "Dez."},
		short:       "{dd}.{MM}.{yyyy}", medium: "{d}. {MMM} {yyyy}", long: "{d}. {MMMM} {yyyy}",
	},
	"es": {
		monthsLong:  [12]string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"},
		monthsShort: [12]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sept", "oct", "nov", "dic"},
		short:       "{dd}/{MM}/{yyyy}", medium: "{d} {MMM} {yyyy}", long: "{d} de {MMMM} de {yyyy}",
	},
	"pt": {
		monthsLong:  [12]string{"janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"},
		monthsShort: [12]string{"jan.", "fev.", "mar.", "abr.", "mai.", "jun.", "jul.", "ago.", "set.", "out.", "nov.", "dez."},
		short:       "{dd}/{MM}/{yyyy}", medium: "{d} de {MMM} de {yyyy}", long: "{d} de {MMMM} de {yyyy}",
	},
	"it": {
		monthsLong:  [12]string{"gennaio", "febbraio", "marzo", "aprile", "maggio", "giugno", "luglio", "agosto", "settembre", "ottobre", "novembre", "dicembre"},
		monthsShort: [12]string{"gen", "feb", "mar", "apr", "mag", "giu", "lug", "ago", "set", "ott", "nov", "dic"},
		short:       "{dd}/{MM}/{yyyy}", medium: "{d} {MMM} {yyyy}", long: "{d} {MMMM} {yyyy}",
	},
	"nl": {
		monthsLong:  [12]string{"januari", "februari", "maart", "april", "mei", "juni", "juli", "augustus", "september", "oktober", "november", "december"},
		monthsShort: [12]string{"jan", "feb", "mrt", "apr", "mei", "jun", "jul", "aug", "sep", "okt", "nov", "dec"},
		short:       "{dd}-{MM}-{yyyy}", medium: "{d} {MMM} {yyyy}", long: "{d} {MMMM} {yyyy}",
	},
	"ru": {
		monthsLong:  [12]string{"января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"},
		monthsShort: [12]string{"янв.", "февр.", "мар.", "апр.", "мая", "июн.", "июл.", "авг.", "сент.", "окт.", "нояб.", "дек."},
		short:       "{dd}.{MM}.{yyyy}", medium: "{d} {MMM} {yyyy} г.", long: "{d} {MMMM} {yyyy} г.",
	},
	"pl": {
		monthsLong:  [12]string{"stycznia", "lutego", "marca", "kwietnia", "maja", "czerwca", "lipca", "sierpnia", "września", "października", "listopada", "grudnia"},
		monthsShort: [12]string{"sty", "lut", "mar", "kwi", "maj", "cze", "lip", "sie", "wrz", "paź", "lis", "gru"},
		short:       "{dd}.{MM}.{yyyy}", medium: "{d} {MMM} {yyyy}", long: "{d} {MMMM} {yyyy}",
	},
	"cs": {
		monthsLong:  [12]string{"ledna", "února", "března", "dubna", "května", "června", "července", "srpna", "září", "října", "listopadu", "prosince"},
		monthsShort: [12]string{"led", "úno", "bře", "dub", "kvě", "čvn", "čvc", "srp", "zář", "říj", "lis", "pro"},
		short:       "{dd}.{MM}.{yyyy}", medium: "{d}. {M}. {yyyy}", long: "{d}. {MMMM} {yyyy}",
	},
	"tr": {
		monthsLong:  [12]string{"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran", "Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık"},
		monthsShort: [12]string{"Oca", "Şub", "Mar", "Nis", "May", "Haz", "Tem", "Ağu", "Eyl", "Eki", "Kas", "Ara"},
		short:       "{dd}.{MM}.{yyyy}", medium: "{d} {MMM} {yyyy}", long: "{d} {MMMM} {yyyy}",
	},
	"ja": {
		short: "{yyyy}/{MM}/{dd}", medium: "{yyyy}年{M}月{d}日", long: "{yyyy}年{M}月{d}日",
	},
	"zh": {
		short: "{yyyy}/{MM}/{dd}", medium: "{yyyy}年{M}月{d}日", long: "{yyyy}年{M}月{d}日",
	},
	"ko": {
		short: "{yyyy}. {MM}. {dd}.", medium: "{yyyy}. {M}. {d}.", long: "{yyyy}년 {M}월 {d}일",
	},
	"ar": {
		monthsLong:  [12]string{"يناير", "فبراير", "مارس", "أبريل", "مايو", "يونيو", "يوليو", "أغسطس", "سبتمبر", "أكتوبر", "نوفمبر", "ديسمبر"},
		monthsShort: [12]string{"يناير", "فبراير", "مارس", "أبريل", "مايو", "يونيو", "يوليو", "أغسطس", "سبتمبر", "أكتوبر", "نوفمبر", "ديسمبر"},
		short:       "{dd}/{MM}/{yyyy}", medium: "{dd}/{MM}/{yyyy}", long: "{d} {MMMM} {yyyy}",
	},
}

// formatLocaleDate renders parseValue for a language in dateLocales; the
// second result is false when the language has no table and the caller should
// use the English layouts.
func formatLocaleDate(parseFamily string, parseValue time.Time, parseStyle DateStyle) (string, bool) {
	parseData, parseOk := dateLocales[parseFamily]
	if !parseOk {
		return "", false
	}
	parseLayout := parseData.medium
	switch parseStyle {
	case DateStyleShort:
		parseLayout = parseData.short
	case DateStyleLong:
		parseLayout = parseData.long
	}
	parseMonth := int(parseValue.Month()) - 1
	return strings.NewReplacer(
		"{yyyy}", parseValue.Format("2006"),
		"{MMMM}", parseData.monthsLong[parseMonth],
		"{MMM}", parseData.monthsShort[parseMonth],
		"{MM}", parseValue.Format("01"),
		"{M}", parseValue.Format("1"),
		"{dd}", parseValue.Format("02"),
		"{d}", parseValue.Format("2"),
	).Replace(parseLayout), true
}

// listPattern holds the separators used to join a list in one language.
type listPattern struct {
	sep  string // between items before the last one
	pair string // between exactly two items
	last string // before the final item of three or more
}

var listPatterns = map[string]listPattern{
	"fr": {", ", " et ", " et "},
	"de": {", ", " und ", " und "},
	"es": {", ", " y ", " y "},
	"pt": {", ", " e ", " e "},
	"it": {", ", " e ", " e "},
	"nl": {", ", " en ", " en "},
	"ru": {", ", " и ", " и "},
	"pl": {", ", " i ", " i "},
	"cs": {", ", " a ", " a "},
	"tr": {", ", " ve ", " ve "},
	"ja": {"、", "、", "、"},
	"zh": {"、", "和", "和"},
	"ko": {", ", " 및 ", " 및 "},
	"ar": {"، ", " و ", "، و "},
}

// englishListPattern is the fallback for every language without a table.
var englishListPattern = listPattern{", ", " and ", ", and "}

// joinLocaleList joins parseItems (two or more) with the pattern of parseFamily.
func joinLocaleList(parseFamily string, parseItems []string) string {
	parsePattern, parseOk := listPatterns[parseFamily]
	if !parseOk {
		parsePattern = englishListPattern
	}
	parseCount := len(parseItems)
	if parseCount == 2 {
		return parseItems[0] + parsePattern.pair + parseItems[1]
	}
	return strings.Join(parseItems[:parseCount-1], parsePattern.sep) + parsePattern.last + parseItems[parseCount-1]
}

// isRouteLocaleSegment reports whether parseSegment is a well-formed locale
// of the shape ll, ll-RR, ll-Script or ll-Script-RR, used when a route has no
// SupportedLocales list to check against. A three-letter language must carry
// a script or region so that path segments like "api" or "app" (valid ISO
// 639-3 codes) are not mistaken for locales.
func isRouteLocaleSegment(parseSegment string) bool {
	parseParts := strings.Split(parseSegment, "-")
	parseLang := parseParts[0]
	if !isLetters(parseLang, 2, 3) {
		return false
	}
	if len(parseLang) == 3 && len(parseParts) == 1 {
		return false
	}
	parseRest := parseParts[1:]
	if len(parseRest) > 0 && isLetters(parseRest[0], 4, 4) {
		parseRest = parseRest[1:]
	}
	if len(parseRest) > 1 {
		return false
	}
	if len(parseRest) == 1 && !isLetters(parseRest[0], 2, 2) && !isDigits(parseRest[0], 3) {
		return false
	}
	_, parseErr := language.Parse(parseSegment)
	return parseErr == nil
}

// isLetters reports whether parseValue is ASCII letters with length in [parseMin, parseMax].
func isLetters(parseValue string, parseMin int, parseMax int) bool {
	if len(parseValue) < parseMin || len(parseValue) > parseMax {
		return false
	}
	for parseIndex := 0; parseIndex < len(parseValue); parseIndex++ {
		parseChar := parseValue[parseIndex] | 0x20
		if parseChar < 'a' || parseChar > 'z' {
			return false
		}
	}
	return true
}

// isDigits reports whether parseValue is exactly parseLength ASCII digits.
func isDigits(parseValue string, parseLength int) bool {
	if len(parseValue) != parseLength {
		return false
	}
	for parseIndex := 0; parseIndex < len(parseValue); parseIndex++ {
		if parseValue[parseIndex] < '0' || parseValue[parseIndex] > '9' {
			return false
		}
	}
	return true
}

// escapeMessageNamespace escapes backslash and dot so a namespace survives
// the "namespace.key" SSR key encoding.
func escapeMessageNamespace(parseNamespace string) string {
	if !strings.ContainsAny(parseNamespace, `\.`) {
		return parseNamespace
	}
	parseNamespace = strings.ReplaceAll(parseNamespace, `\`, `\\`)
	return strings.ReplaceAll(parseNamespace, ".", `\.`)
}

// joinCombinedMessageKey builds the SSR bootstrap key for a namespace and key.
func joinCombinedMessageKey(parseNamespace string, parseKey string) string {
	return escapeMessageNamespace(parseNamespace) + "." + parseKey
}
