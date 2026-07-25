package sanitize_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/sanitize"
)

// xssMarkers are substrings that must never appear in sanitized output.
var xssMarkers = []string{
	"<script", "javascript:", "onerror", "onload",
	"<iframe", "<svg", "<style", "data:text/html", "vbscript:", " style=",
}

// TestXSSCorpus verifies that known XSS payloads produce output free of every
// xssMarkers entry.
func TestXSSCorpus(t *testing.T) {
	parseInputs := []struct {
		parseName  string
		parseInput string
	}{
		{"script tag", `<script>alert(1)</script>`},
		{"img onerror", `<img src=x onerror=alert(1)>`},
		{"a javascript href", `<a href="javascript:alert(1)">x</a>`},
		{"a tab-embedded javascript", `<a href="java` + "\t" + `script:alert(1)">x</a>`},
		{"a zero-width javascript href", "<a href=\"java\u200Bscript:alert(1)\">x</a>"},
		{"a unicode line-separator javascript href", "<a href=\"java\u2028script:alert(1)\">x</a>"},
		{"a unicode paragraph-separator javascript href", "<a href=\"java\u2029script:alert(1)\">x</a>"},
		{"svg onload", `<svg/onload=alert(1)>`},
		{"iframe javascript src", `<iframe src=javascript:alert(1)>`},
		{"div style expression", `<div style="x:expression(alert(1))">`},
		{"a data-text/html href", `<a href="data:text/html,<script>alert(1)</script>">click</a>`},
		{"a vbscript href", `<a href="vbscript:msgbox(1)">v</a>`},
		{"nested script in p", `<div><p><script>alert(1)</script></p></div>`},
	}

	for _, parseCase := range parseInputs {
		t.Run(parseCase.parseName, func(t *testing.T) {
			parseOutput := sanitize.Sanitize(parseCase.parseInput)
			parseLower := strings.ToLower(parseOutput)
			for _, parseMarker := range xssMarkers {
				if strings.Contains(parseLower, parseMarker) {
					t.Errorf("output contains XSS marker %q\ninput:  %s\noutput: %s",
						parseMarker, parseCase.parseInput, parseOutput)
				}
			}
		})
	}
}

func TestEmptyInputContract(t *testing.T) {
	parseCases := map[string]string{
		"empty":      "",
		"whitespace": "   ",
		"text":       "hello",
	}
	parseWants := map[string]string{
		"empty":      "",
		"whitespace": "",
		"text":       "hello",
	}
	for parseName, parseInput := range parseCases {
		t.Run(parseName, func(t *testing.T) {
			if parseGot := sanitize.Sanitize(parseInput); parseGot != parseWants[parseName] {
				t.Fatalf("Sanitize(%q) = %q, want %q", parseInput, parseGot, parseWants[parseName])
			}
		})
	}
}

// TestPositiveSurvival checks that safe markup survives sanitization intact.
func TestPositiveSurvival(t *testing.T) {
	t.Run("bold and paragraph preserved", func(t *testing.T) {
		parseOut := sanitize.Sanitize(`<p>Hello <strong>world</strong></p>`)
		if !strings.Contains(parseOut, "<p>") {
			t.Errorf("expected <p> in output; got: %s", parseOut)
		}
		if !strings.Contains(parseOut, "<strong>") {
			t.Errorf("expected <strong> in output; got: %s", parseOut)
		}
	})

	t.Run("https href kept", func(t *testing.T) {
		parseOut := sanitize.Sanitize(`<a href="https://example.test">ok</a>`)
		if !strings.Contains(parseOut, `href="https://example.test"`) {
			t.Errorf("expected href to be kept; got: %s", parseOut)
		}
	})

	t.Run("relative href kept", func(t *testing.T) {
		parseOut := sanitize.Sanitize(`<a href="/rel">r</a>`)
		if !strings.Contains(parseOut, `href="/rel"`) {
			t.Errorf("expected relative href; got: %s", parseOut)
		}
	})

	t.Run("fragment href kept", func(t *testing.T) {
		parseOut := sanitize.Sanitize(`<a href="#frag">f</a>`)
		if !strings.Contains(parseOut, `href="#frag"`) {
			t.Errorf("expected fragment href; got: %s", parseOut)
		}
	})

	t.Run("img src and alt kept", func(t *testing.T) {
		parseOut := sanitize.Sanitize(`<img src="https://x/y.png" alt="a">`)
		if !strings.Contains(parseOut, `src="https://x/y.png"`) {
			t.Errorf("expected src kept; got: %s", parseOut)
		}
		if !strings.Contains(parseOut, `alt="a"`) {
			t.Errorf("expected alt kept; got: %s", parseOut)
		}
	})

	t.Run("table round-trip", func(t *testing.T) {
		parseOut := sanitize.Sanitize(`<table><tr><td>cell</td></tr></table>`)
		for _, parseTag := range []string{"<table>", "<tr>", "<td>", "cell", "</td>", "</tr>", "</table>"} {
			if !strings.Contains(parseOut, parseTag) {
				t.Errorf("missing %q in table output; got: %s", parseTag, parseOut)
			}
		}
	})

	t.Run("escaped text content preserved", func(t *testing.T) {
		// The browser-parsed text of "&lt;" is "<"; re-serialized it should be "&lt;".
		parseOut := sanitize.Sanitize(`<p>1 &lt; 2 &amp; ok</p>`)
		if !strings.Contains(parseOut, "1 &lt; 2 &amp; ok") {
			t.Errorf("expected escaped text preserved; got: %s", parseOut)
		}
	})
}

// TestAllowlistRoundTrip exercises custom policies.
func TestAllowlistRoundTrip(t *testing.T) {
	t.Run("data scheme allowed by custom policy keeps src", func(t *testing.T) {
		parsePolicy := sanitize.DefaultPolicy()
		parsePolicy.AllowedURLSchemes["data"] = true
		parseOut := parsePolicy.Sanitize(`<img src="data:image/png;base64,AAA" alt="x">`)
		if !strings.Contains(parseOut, `src="data:image/png;base64,AAA"`) {
			t.Errorf("expected data: src to be kept; got: %s", parseOut)
		}
	})

	t.Run("img removed from AllowedTags drops img element", func(t *testing.T) {
		parsePolicy := sanitize.DefaultPolicy()
		delete(parsePolicy.AllowedTags, "img")
		parseOut := parsePolicy.Sanitize(`<img src="https://x/y.png" alt="a">`)
		if strings.Contains(parseOut, "<img") {
			t.Errorf("expected img to be dropped; got: %s", parseOut)
		}
	})
}

// TestStability verifies that sanitizing the same input twice produces
// identical output (idempotency under re-sanitization).
func TestStability(t *testing.T) {
	parseInput := `<div><p>Hello <strong>world</strong></p><a href="https://example.test">link</a><script>evil()</script></div>`
	parseFirst := sanitize.Sanitize(parseInput)
	parseSecond := sanitize.Sanitize(parseFirst)
	if parseFirst != parseSecond {
		t.Errorf("sanitize not idempotent:\nfirst:  %s\nsecond: %s", parseFirst, parseSecond)
	}
}

// BenchmarkSanitizeLargeDocument measures throughput on a ~50 KB document
// composed of safe, repeated HTML snippets.
func BenchmarkSanitizeLargeDocument(b *testing.B) {
	parseSnippet := `<div><p>Hello <strong>world</strong></p><a href="https://example.test">link</a></div>`
	var parseBuf strings.Builder
	for range 500 {
		parseBuf.WriteString(parseSnippet)
	}
	parseDoc := parseBuf.String()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sanitize.Sanitize(parseDoc)
	}
}
