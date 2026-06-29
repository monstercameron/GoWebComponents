package i18n_test

import (
	"fmt"
	"time"

	"github.com/monstercameron/GoWebComponents/i18n"
)

// ExampleFormatRelativeTime formats a timestamp relative to a base time.
func ExampleFormatRelativeTime() {
	parseBase := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	parsePast := parseBase.Add(-3 * 24 * time.Hour)
	fmt.Println(i18n.FormatRelativeTime("en", parsePast, parseBase))
	fmt.Println(i18n.FormatRelativeTime("fr", parsePast, parseBase))
	// Output:
	// 3 days ago
	// il y a 3 jours
}

// ExampleFormatList joins items with the locale's list conjunction.
func ExampleFormatList() {
	fmt.Println(i18n.FormatList("en", []string{"alpha", "beta", "gamma"}))
	// Output:
	// alpha, beta, and gamma
}
