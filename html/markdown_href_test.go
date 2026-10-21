package html

import "testing"

func TestSanitizeMarkdownHrefDropsDangerousSchemes(parseT *testing.T) {
	parseDropped := []string{
		"javascript:alert(1)",
		"JavaScript:alert(1)",
		"  javascript:alert(1)",
		"java\tscript:alert(1)",
		"java\nscript:alert(1)",
		"jAvAsCrIpT:alert(1)",
		"vbscript:msgbox(1)",
		"data:text/html,<script>alert(1)</script>",
		"data:image/png;base64,AAAA",
		"\x01javascript:alert(1)",
	}
	for _, parseCase := range parseDropped {
		if parseGot := SanitizeMarkdownHref(parseCase, nil); parseGot != "" {
			parseT.Fatalf("expected %q to be dropped, got %q", parseCase, parseGot)
		}
	}

	parseKept := []string{
		"https://example.test/docs",
		"http://example.test",
		"mailto:dev@example.test",
		"guide.md#hydration",
		"./relative/page",
		"/absolute/path",
		"#section",
		"//cdn.example.test/asset.js",
	}
	for _, parseCase := range parseKept {
		if parseGot := SanitizeMarkdownHref(parseCase, nil); parseGot != parseCase {
			parseT.Fatalf("expected %q to survive unchanged, got %q", parseCase, parseGot)
		}
	}

	// A configurable allowlist round-trips: opting data in keeps inline images.
	if parseGot := SanitizeMarkdownHref("data:image/png;base64,AAAA", []string{"http", "https", "data"}); parseGot == "" {
		parseT.Fatal("expected data: to survive when explicitly allowed")
	}
	// And a custom allowlist still drops what it does not list.
	if parseGot := SanitizeMarkdownHref("https://example.test", []string{"mailto"}); parseGot != "" {
		parseT.Fatal("expected https to be dropped when only mailto is allowed")
	}
}

func TestResolveMarkdownHrefResolvesRelativeDestinations(parseT *testing.T) {
	parseResolved := ResolveMarkdownHref("assets/docs/start-here.md", "troubleshooting.md#hydration")
	if parseResolved != "assets/docs/troubleshooting.md#hydration" {
		parseT.Fatalf("expected relative href resolution, got %q", parseResolved)
	}

	parseAbsolute := ResolveMarkdownHref("assets/docs/start-here.md", "https://example.test/docs")
	if parseAbsolute != "https://example.test/docs" {
		parseT.Fatalf("expected absolute href passthrough, got %q", parseAbsolute)
	}
}

func TestResolveMarkdownHrefEdgeCases(parseT *testing.T) {
	if parseGot := ResolveMarkdownHref("docs/start.md", ""); parseGot != "" {
		parseT.Fatalf("expected empty destination to resolve empty, got %q", parseGot)
	}
	if parseGot2 := ResolveMarkdownHref("docs/start.md", "%zz"); parseGot2 != "%zz" {
		parseT.Fatalf("expected invalid URL destination passthrough, got %q", parseGot2)
	}
	if parseGot3 := ResolveMarkdownHref("docs/start.md", "#section"); parseGot3 != "#section" {
		parseT.Fatalf("expected fragment destination passthrough, got %q", parseGot3)
	}
	if parseGot4 := ResolveMarkdownHref("docs/start.md", "/guide"); parseGot4 != "/guide" {
		parseT.Fatalf("expected absolute path destination passthrough, got %q", parseGot4)
	}
	if parseGot5 := ResolveMarkdownHref("", "guide.md"); parseGot5 != "guide.md" {
		parseT.Fatalf("expected blank source path passthrough, got %q", parseGot5)
	}
	if parseGot6 := ResolveMarkdownHref("https://example.test/docs/start.md", "guide.md?q=1"); parseGot6 != "https://example.test/docs/guide.md?q=1" {
		parseT.Fatalf("expected absolute source path resolution, got %q", parseGot6)
	}
}
