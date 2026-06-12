package i18n

import (
	"testing"
	"time"
)

// baseTime is a fixed reference point used across all relative-time tests.
var baseTime = time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

func TestFormatRelativeTime_EnBoundaryRounding(t *testing.T) {
	tests := []struct {
		parseName   string
		parseOffset time.Duration
		parseWant   string
	}{
		{"59s past", -59 * time.Second, "59 seconds ago"},
		{"60s past", -60 * time.Second, "1 minute ago"},
		{"23h past", -23 * time.Hour, "23 hours ago"},
		{"24h past", -24 * time.Hour, "1 day ago"},
	}
	for _, parseTT := range tests {
		t.Run(parseTT.parseName, func(t *testing.T) {
			parseGot := FormatRelativeTime("en", baseTime.Add(parseTT.parseOffset), baseTime)
			if parseGot != parseTT.parseWant {
				t.Errorf("FormatRelativeTime(%q) = %q; want %q", parseTT.parseName, parseGot, parseTT.parseWant)
			}
		})
	}
}

func TestFormatRelativeTime_EnPluralInteraction(t *testing.T) {
	tests := []struct {
		parseName   string
		parseOffset time.Duration
		parseWant   string
	}{
		{"1 day past", -24 * time.Hour, "1 day ago"},
		{"2 days past", -48 * time.Hour, "2 days ago"},
		{"0 seconds (same time)", 0, "0 seconds ago"},
	}
	for _, parseTT := range tests {
		t.Run(parseTT.parseName, func(t *testing.T) {
			parseGot := FormatRelativeTime("en", baseTime.Add(parseTT.parseOffset), baseTime)
			if parseGot != parseTT.parseWant {
				t.Errorf("FormatRelativeTime(%q) = %q; want %q", parseTT.parseName, parseGot, parseTT.parseWant)
			}
		})
	}
}

func TestFormatRelativeTime_FrPlural(t *testing.T) {
	tests := []struct {
		parseName   string
		parseOffset time.Duration
		parseWant   string
	}{
		{"1 jour past", -24 * time.Hour, "il y a 1 jour"},
		{"2 jours past", -48 * time.Hour, "il y a 2 jours"},
	}
	for _, parseTT := range tests {
		t.Run(parseTT.parseName, func(t *testing.T) {
			parseGot := FormatRelativeTime("fr", baseTime.Add(parseTT.parseOffset), baseTime)
			if parseGot != parseTT.parseWant {
				t.Errorf("FormatRelativeTime(%q) = %q; want %q", parseTT.parseName, parseGot, parseTT.parseWant)
			}
		})
	}
}

func TestFormatRelativeTime_JaNoPluralNoSpaces(t *testing.T) {
	parseGot := FormatRelativeTime("ja", baseTime.Add(-3*24*time.Hour), baseTime)
	parseWant := "3日前"
	if parseGot != parseWant {
		t.Errorf("FormatRelativeTime ja 3 days past = %q; want %q", parseGot, parseWant)
	}
}

func TestFormatRelativeTime_ArPastContainsMindhu(t *testing.T) {
	// 5 days past → plural category few (5 % 100 = 5, which is 3–10)
	parseGot := FormatRelativeTime("ar", baseTime.Add(-5*24*time.Hour), baseTime)
	if len(parseGot) == 0 {
		t.Fatal("FormatRelativeTime ar returned empty string")
	}
	// Must contain the Arabic "منذ" (past marker)
	parseWantPrefix := "منذ"
	if !containsString(parseGot, parseWantPrefix) {
		t.Errorf("FormatRelativeTime ar past = %q; want prefix %q", parseGot, parseWantPrefix)
	}
	// Must contain the few-form of "day" = أيام
	if !containsString(parseGot, "أيام") {
		t.Errorf("FormatRelativeTime ar 5 days past = %q; want unit أيام", parseGot)
	}
}

func TestFormatRelativeTime_ArPluralUnit(t *testing.T) {
	tests := []struct {
		parseName     string
		parseDays     int
		parseWantUnit string
	}{
		{"1 day (one)", 1, "يوم"},
		{"2 days (two)", 2, "يومان"},
		{"3 days (few)", 3, "أيام"},
	}
	for _, parseTT := range tests {
		t.Run(parseTT.parseName, func(t *testing.T) {
			parseOffset := time.Duration(-parseTT.parseDays) * 24 * time.Hour
			parseGot := FormatRelativeTime("ar", baseTime.Add(parseOffset), baseTime)
			if !containsString(parseGot, parseTT.parseWantUnit) {
				t.Errorf("FormatRelativeTime ar %s = %q; want unit %q", parseTT.parseName, parseGot, parseTT.parseWantUnit)
			}
		})
	}
}

func TestFormatRelativeTime_FutureVsPast(t *testing.T) {
	tests := []struct {
		parseLocale string
		parsePast   string
		parseFuture string
		parseOffset time.Duration
	}{
		{"en", "2 hours ago", "in 2 hours", 2 * time.Hour},
		{"fr", "il y a 2 heures", "dans 2 heures", 2 * time.Hour},
		{"ja", "2時間前", "2時間後", 2 * time.Hour},
		{"ar", "منذ", "بعد", 2 * time.Hour},
	}
	for _, parseTT := range tests {
		t.Run(parseTT.parseLocale+"_past", func(t *testing.T) {
			parseGot := FormatRelativeTime(parseTT.parseLocale, baseTime.Add(-parseTT.parseOffset), baseTime)
			if !containsString(parseGot, parseTT.parsePast) {
				t.Errorf("past %s = %q; want to contain %q", parseTT.parseLocale, parseGot, parseTT.parsePast)
			}
		})
		t.Run(parseTT.parseLocale+"_future", func(t *testing.T) {
			parseGot := FormatRelativeTime(parseTT.parseLocale, baseTime.Add(parseTT.parseOffset), baseTime)
			if !containsString(parseGot, parseTT.parseFuture) {
				t.Errorf("future %s = %q; want to contain %q", parseTT.parseLocale, parseGot, parseTT.parseFuture)
			}
		})
	}
}

func TestFormatList_En(t *testing.T) {
	tests := []struct {
		parseName  string
		parseItems []string
		parseWant  string
	}{
		{"0 items", []string{}, ""},
		{"1 item", []string{"alpha"}, "alpha"},
		{"2 items", []string{"alpha", "beta"}, "alpha and beta"},
		{"3 items", []string{"alpha", "beta", "gamma"}, "alpha, beta, and gamma"},
	}
	for _, parseTT := range tests {
		t.Run(parseTT.parseName, func(t *testing.T) {
			parseGot := FormatList("en", parseTT.parseItems)
			if parseGot != parseTT.parseWant {
				t.Errorf("FormatList en %s = %q; want %q", parseTT.parseName, parseGot, parseTT.parseWant)
			}
		})
	}
}

func TestFormatList_Fr(t *testing.T) {
	tests := []struct {
		parseName  string
		parseItems []string
		parseWant  string
	}{
		{"0 items", []string{}, ""},
		{"1 item", []string{"alpha"}, "alpha"},
		{"2 items", []string{"alpha", "beta"}, "alpha et beta"},
		{"3 items", []string{"alpha", "beta", "gamma"}, "alpha, beta et gamma"},
	}
	for _, parseTT := range tests {
		t.Run(parseTT.parseName, func(t *testing.T) {
			parseGot := FormatList("fr", parseTT.parseItems)
			if parseGot != parseTT.parseWant {
				t.Errorf("FormatList fr %s = %q; want %q", parseTT.parseName, parseGot, parseTT.parseWant)
			}
		})
	}
}

func TestFormatList_Ja(t *testing.T) {
	tests := []struct {
		parseName  string
		parseItems []string
		parseWant  string
	}{
		{"0 items", []string{}, ""},
		{"1 item", []string{"alpha"}, "alpha"},
		{"2 items", []string{"alpha", "beta"}, "alpha、beta"},
		{"3 items", []string{"alpha", "beta", "gamma"}, "alpha、beta、gamma"},
	}
	for _, parseTT := range tests {
		t.Run(parseTT.parseName, func(t *testing.T) {
			parseGot := FormatList("ja", parseTT.parseItems)
			if parseGot != parseTT.parseWant {
				t.Errorf("FormatList ja %s = %q; want %q", parseTT.parseName, parseGot, parseTT.parseWant)
			}
		})
	}
}

func TestFormatList_Ar(t *testing.T) {
	tests := []struct {
		parseName  string
		parseItems []string
		parseWant  string
	}{
		{"0 items", []string{}, ""},
		{"1 item", []string{"alpha"}, "alpha"},
		{"2 items", []string{"alpha", "beta"}, "alpha و beta"},
		{"3 items", []string{"alpha", "beta", "gamma"}, "alpha، beta، و gamma"},
	}
	for _, parseTT := range tests {
		t.Run(parseTT.parseName, func(t *testing.T) {
			parseGot := FormatList("ar", parseTT.parseItems)
			if parseGot != parseTT.parseWant {
				t.Errorf("FormatList ar %s = %q; want %q", parseTT.parseName, parseGot, parseTT.parseWant)
			}
		})
	}
}

// containsString reports whether s contains substr.
func containsString(parseS, parseSubstr string) bool {
	return len(parseSubstr) == 0 || (len(parseS) >= len(parseSubstr) && stringContains(parseS, parseSubstr))
}

func stringContains(parseS, parseSubstr string) bool {
	for parseI := 0; parseI <= len(parseS)-len(parseSubstr); parseI++ {
		if parseS[parseI:parseI+len(parseSubstr)] == parseSubstr {
			return true
		}
	}
	return false
}
