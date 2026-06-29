//go:build !(js && wasm)

package css_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/css"
)

// TestGlobalRulesParticipateInSeedSuppression proves css.Global / css.Root rules
// are listed in the SSR StyleBlock's data-gwc-css key set and are suppressed by
// Seed on hydration — so they are not re-injected client-side (no FOUC / dup).
func TestGlobalRulesParticipateInSeedSuppression(parseT *testing.T) {
	css.Reset()
	css.Global("body", css.Raw("margin", "0"))
	css.Root(css.Raw("--accent", "#fff"))
	parseSheet := css.New(css.Raw("color", "red"))

	parseBlock := css.StyleBlock()
	if !strings.Contains(parseBlock, "body{margin:0") ||
		!strings.Contains(parseBlock, ":root{") ||
		!strings.Contains(parseBlock, "."+string(parseSheet)) {
		parseT.Fatalf("StyleBlock missing global/root/hashed rule:\n%s", parseBlock)
	}

	parseKeys := strings.Fields(regexp.MustCompile(`data-gwc-css="([^"]*)"`).FindStringSubmatch(parseBlock)[1])
	if len(parseKeys) != 3 {
		parseT.Fatalf("expected 3 seed keys (2 global + 1 hashed), got %v", parseKeys)
	}

	// Client hydration: seed, then re-emit the SAME rules — nothing should inject.
	css.Reset()
	css.Seed(parseKeys...)
	css.Global("body", css.Raw("margin", "0"))
	css.Root(css.Raw("--accent", "#fff"))
	css.New(css.Raw("color", "red"))
	if parseOut := css.Harvest(); parseOut != "" {
		parseT.Fatalf("global/root rules re-injected after Seed (FOUC/dup): %q", parseOut)
	}
}
