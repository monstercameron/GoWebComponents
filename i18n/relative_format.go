package i18n

import (
	"fmt"
	"math"
	"strings"
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
	}

	// Pick the largest fitting unit and compute n.
	type parseUnit struct {
		key string
		n   int
	}
	var parseU parseUnit
	switch {
	case parseAbsDelta < 60*time.Second:
		parseU = parseUnit{"second", int(math.Round(parseAbsDelta.Seconds()))}
	case parseAbsDelta < 60*time.Minute:
		parseU = parseUnit{"minute", int(math.Round(parseAbsDelta.Minutes()))}
	case parseAbsDelta < 24*time.Hour:
		parseU = parseUnit{"hour", int(math.Round(parseAbsDelta.Hours()))}
	case parseAbsDelta < 7*24*time.Hour:
		parseU = parseUnit{"day", int(math.Round(parseAbsDelta.Hours() / 24))}
	case parseAbsDelta < 30*24*time.Hour:
		parseU = parseUnit{"week", int(math.Round(parseAbsDelta.Hours() / (24 * 7)))}
	case parseAbsDelta < 365*24*time.Hour:
		parseU = parseUnit{"month", int(math.Round(parseAbsDelta.Hours() / (24 * 30)))}
	default:
		parseU = parseUnit{"year", int(math.Round(parseAbsDelta.Hours() / (24 * 365)))}
	}

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
	parseFamily := localeFamily(NormalizeLocale(parseLocale))
	parseCount := len(parseItems)

	switch parseCount {
	case 0:
		return ""
	case 1:
		return parseItems[0]
	}

	switch parseFamily {
	case "fr":
		if parseCount == 2 {
			return parseItems[0] + " et " + parseItems[1]
		}
		return strings.Join(parseItems[:parseCount-1], ", ") + " et " + parseItems[parseCount-1]

	case "ja":
		return strings.Join(parseItems, "、")

	case "ar":
		if parseCount == 2 {
			return parseItems[0] + " و " + parseItems[1]
		}
		return strings.Join(parseItems[:parseCount-1], "، ") + "، و " + parseItems[parseCount-1]

	default: // en
		if parseCount == 2 {
			return parseItems[0] + " and " + parseItems[1]
		}
		return strings.Join(parseItems[:parseCount-1], ", ") + ", and " + parseItems[parseCount-1]
	}
}
