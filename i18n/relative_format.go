package i18n

import (
	"fmt"
	"math"
	"time"
)

// FormatRelativeTime formats parseValue relative to parseBase as a human-readable
// string in the language of parseLocale (e.g. "3 days ago", "in 2 hours").
func FormatRelativeTime(parseLocale string, parseValue time.Time, parseBase time.Time) string {
	parseDelta := parseValue.Sub(parseBase)
	parseIsFuture := parseDelta > 0
	parseAbsDelta := parseDelta
	if parseAbsDelta < 0 {
		parseAbsDelta = -parseAbsDelta
		// -MinInt64 overflows back to a negative: time.Sub saturates for spans over ~292 years
		// (the zero time.Time against now), so clamp to the largest representable span.
		if parseAbsDelta < 0 {
			parseAbsDelta = time.Duration(math.MaxInt64)
		}
	}

	// Pick the largest fitting unit and compute n.
	parseU := getRelativeUnit(parseAbsDelta)

	parseFamily := localeFamily(NormalizeLocale(parseLocale))
	parseN := parseU.n
	parseKey := parseU.key

	switch parseFamily {
	case "fr":
		parseUnitStr := parseFrUnit(parseKey, parseN)
		if parseIsFuture {
			return fmt.Sprintf("dans %d %s", parseN, parseUnitStr)
		}
		return fmt.Sprintf("il y a %d %s", parseN, parseUnitStr)

	case "ja":
		parseUnitStr := parseJaUnit(parseKey)
		if parseIsFuture {
			return fmt.Sprintf("%d%s後", parseN, parseUnitStr)
		}
		return fmt.Sprintf("%d%s前", parseN, parseUnitStr)

	case "ar":
		parseCategory := pluralCategoryForLocale(parseLocale, float64(parseN))
		parseUnitStr := parseArUnit(parseKey, parseCategory)
		if parseIsFuture {
			return fmt.Sprintf("بعد %d %s", parseN, parseUnitStr)
		}
		return fmt.Sprintf("منذ %d %s", parseN, parseUnitStr)

	default: // en and everything else
		parseCategory := pluralCategoryForLocale(parseLocale, float64(parseN))
		parseUnitStr := parseEnUnit(parseKey, parseCategory)
		if parseIsFuture {
			return fmt.Sprintf("in %d %s", parseN, parseUnitStr)
		}
		return fmt.Sprintf("%d %s ago", parseN, parseUnitStr)
	}
}

// relativeUnit is one unit key ("minute") with its rounded count.
type relativeUnit struct {
	key string
	n   int
}

// getRelativeUnit picks the unit and rounded count for one absolute span. The unit is chosen from
// the ROUNDED count, so 59.7 seconds reads "1 minute" rather than "60 seconds", and 350 days reads
// "1 year" rather than "12 months".
func getRelativeUnit(parseAbs time.Duration) relativeUnit {
	if parseSeconds := int(math.Round(parseAbs.Seconds())); parseSeconds < 60 {
		return relativeUnit{"second", parseSeconds}
	}
	if parseMinutes := int(math.Round(parseAbs.Minutes())); parseMinutes < 60 {
		return relativeUnit{"minute", parseMinutes}
	}
	if parseHours := int(math.Round(parseAbs.Hours())); parseHours < 24 {
		return relativeUnit{"hour", parseHours}
	}
	if parseDays := int(math.Round(parseAbs.Hours() / 24)); parseDays < 7 {
		return relativeUnit{"day", parseDays}
	}
	if parseAbs < 30*24*time.Hour {
		return relativeUnit{"week", int(math.Round(parseAbs.Hours() / (24 * 7)))}
	}
	if parseMonths := int(math.Round(parseAbs.Hours() / (24 * 30))); parseMonths < 12 {
		return relativeUnit{"month", parseMonths}
	}
	return relativeUnit{"year", max(1, int(math.Round(parseAbs.Hours()/(24*365))))}
}

// parseEnUnit returns the English singular or plural form for the given unit key.
func parseEnUnit(parseKey string, parseCategory PluralCategory) string {
	parseIsSingular := parseCategory == PluralOne
	switch parseKey {
	case "second":
		if parseIsSingular {
			return "second"
		}
		return "seconds"
	case "minute":
		if parseIsSingular {
			return "minute"
		}
		return "minutes"
	case "hour":
		if parseIsSingular {
			return "hour"
		}
		return "hours"
	case "day":
		if parseIsSingular {
			return "day"
		}
		return "days"
	case "week":
		if parseIsSingular {
			return "week"
		}
		return "weeks"
	case "month":
		if parseIsSingular {
			return "month"
		}
		return "months"
	case "year":
		if parseIsSingular {
			return "year"
		}
		return "years"
	default:
		return parseKey
	}
}

// parseFrUnit returns the French unit form for the given unit key and count.
func parseFrUnit(parseKey string, parseN int) string {
	// fr plural: 0 or 1 → singular (PluralOne), else plural (PluralOther)
	parseCategory := pluralCategoryForLocale("fr", float64(parseN))
	parseIsSingular := parseCategory == PluralOne
	switch parseKey {
	case "second":
		if parseIsSingular {
			return "seconde"
		}
		return "secondes"
	case "minute":
		if parseIsSingular {
			return "minute"
		}
		return "minutes"
	case "hour":
		if parseIsSingular {
			return "heure"
		}
		return "heures"
	case "day":
		if parseIsSingular {
			return "jour"
		}
		return "jours"
	case "week":
		if parseIsSingular {
			return "semaine"
		}
		return "semaines"
	case "month":
		return "mois" // invariant
	case "year":
		if parseIsSingular {
			return "an"
		}
		return "ans"
	default:
		return parseKey
	}
}

// parseJaUnit returns the Japanese unit string (no plural, no spaces).
func parseJaUnit(parseKey string) string {
	switch parseKey {
	case "second":
		return "秒"
	case "minute":
		return "分"
	case "hour":
		return "時間"
	case "day":
		return "日"
	case "week":
		return "週間"
	case "month":
		return "か月"
	case "year":
		return "年"
	default:
		return parseKey
	}
}

// parseArUnit returns the Arabic unit form for the given unit key and plural category.
func parseArUnit(parseKey string, parseCategory PluralCategory) string {
	type parseArForms struct {
		one, two, few, many, other string
	}
	parseUnits := map[string]parseArForms{
		"second": {"ثانية", "ثانيتان", "ثوان", "ثوان", "ثوان"},
		"minute": {"دقيقة", "دقيقتان", "دقائق", "دقائق", "دقائق"},
		"hour":   {"ساعة", "ساعتان", "ساعات", "ساعات", "ساعات"},
		"day":    {"يوم", "يومان", "أيام", "أيام", "أيام"},
		"week":   {"أسبوع", "أسبوعان", "أسابيع", "أسابيع", "أسابيع"},
		"month":  {"شهر", "شهران", "أشهر", "أشهر", "أشهر"},
		"year":   {"سنة", "سنتان", "سنوات", "سنوات", "سنوات"},
	}
	parseForms, parseOk := parseUnits[parseKey]
	if !parseOk {
		return parseKey
	}
	switch parseCategory {
	case PluralOne:
		return parseForms.one
	case PluralTwo:
		return parseForms.two
	case PluralFew:
		return parseForms.few
	case PluralMany:
		return parseForms.many
	default:
		return parseForms.other
	}
}

// FormatList joins parseItems into a locale-aware list string using the
// appropriate conjunction for parseLocale (e.g. "a, b, and c" for en).
func FormatList(parseLocale string, parseItems []string) string {
	switch len(parseItems) {
	case 0:
		return ""
	case 1:
		return parseItems[0]
	}
	return joinLocaleList(localeFamily(NormalizeLocale(parseLocale)), parseItems)
}
