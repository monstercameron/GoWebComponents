package main

import (
	"testing"
)

// TestA11yLintFlagsViolations proves the static accessibility linter catches the core defects —
// missing alt, no accessible name, an unlabeled control, missing lang, positive tabindex — while
// NOT flagging the accessible equivalents (no false positives).
func TestA11yLintFlagsViolations(parseT *testing.T) {
	parseDir := parseT.TempDir()
	mustWrite(parseT, parseDir, "bad.html", `<!doctype html>
<html>
<body>
  <img src="logo.png">
  <button></button>
  <a href="/next"></a>
  <input type="text" id="email">
  <div tabindex="3">trap</div>
</body>
</html>`)
	mustWrite(parseT, parseDir, "good.html", `<!doctype html>
<html lang="en">
<body>
  <img src="logo.png" alt="Company logo">
  <button>Save</button>
  <a href="/next" aria-label="Next page"><img src="arrow.png" alt=""></a>
  <label for="email2">Email</label>
  <input type="text" id="email2">
  <label>Name <input type="text"></label>
  <input type="hidden" name="csrf">
  <div tabindex="0">ok</div>
</body>
</html>`)

	parseIssues, parseErr := collectLintA11yRuleIssues(parseDir, []string{"./..."})
	if parseErr != nil {
		parseT.Fatalf("collectLintA11yRuleIssues: %v", parseErr)
	}

	parseBySymbol := map[string]int{}
	for _, parseIssue := range parseIssues {
		if parseIssue.Linter != a11yLinterName {
			parseT.Fatalf("unexpected linter label %q", parseIssue.Linter)
		}
		parseBySymbol[parseIssue.Symbol]++
	}

	// bad.html: img-alt, interactive-name (button), interactive-name (a), control-label,
	// tabindex-positive. good.html: html (no lang) on bad.html only — good.html has lang.
	parseWant := map[string]int{
		"a11y/img-alt":           1,
		"a11y/interactive-name":  2,
		"a11y/control-label":     1,
		"a11y/tabindex-positive": 1,
		"a11y/html-lang":         1, // only bad.html's <html> lacks lang
	}
	for parseSymbol, parseCount := range parseWant {
		if parseBySymbol[parseSymbol] != parseCount {
			parseT.Fatalf("rule %s: expected %d, got %d (all: %#v)", parseSymbol, parseCount, parseBySymbol[parseSymbol], parseBySymbol)
		}
	}
}

// TestA11yLintCarriesSymbolAndLocation proves each finding names the rule (Symbol) and a source
// line, so an editor can surface a named, located quick-fix.
func TestA11yLintCarriesSymbolAndLocation(parseT *testing.T) {
	parseDir := parseT.TempDir()
	mustWrite(parseT, parseDir, "page.html", "<html lang=\"en\">\n<body>\n  <img src=\"x.png\">\n</body>\n</html>")
	parseIssues, parseErr := collectLintA11yRuleIssues(parseDir, []string{"./..."})
	if parseErr != nil {
		parseT.Fatalf("collectLintA11yRuleIssues: %v", parseErr)
	}
	if len(parseIssues) != 1 {
		parseT.Fatalf("expected exactly one issue (img-alt), got %d: %#v", len(parseIssues), parseIssues)
	}
	parseIssue := parseIssues[0]
	if parseIssue.Symbol != "a11y/img-alt" {
		parseT.Fatalf("expected a11y/img-alt symbol, got %q", parseIssue.Symbol)
	}
	if parseIssue.Line != 3 {
		parseT.Fatalf("expected the <img> on line 3, got line %d", parseIssue.Line)
	}
	if parseIssue.Path != "page.html" {
		parseT.Fatalf("expected relative path page.html, got %q", parseIssue.Path)
	}
}
