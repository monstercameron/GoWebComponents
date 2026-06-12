package main

import (
	"strings"
	"testing"
)

func TestCompilePlaygroundSnippetRendersAllowedNodes(parseT *testing.T) {
	parseSource := `package main

func App() ui.Node {
	return Div(H1("Launch"), P(Text("Ready")), Button("Ship"))
}
`
	parseResult := compilePlaygroundSnippet(parseSource)
	if !parseResult.OK {
		parseT.Fatalf("expected snippet to compile, got diagnostic %#v", parseResult.Diagnostic)
	}
	parseHTML := playgroundSandboxHTML(parseResult)
	for _, parseExpected := range []string{`<h1>Launch</h1>`, `<p>Ready</p>`, `<button>Ship</button>`, `class="playground-root"`} {
		if !strings.Contains(parseHTML, parseExpected) {
			parseT.Fatalf("sandbox HTML missing %q: %s", parseExpected, parseHTML)
		}
	}
}

func TestCompilePlaygroundSnippetEscapesRenderedText(parseT *testing.T) {
	parseResult := compilePlaygroundSnippet(`package main

func App() ui.Node {
	return Div("<script>alert(1)</script>")
}
`)
	if !parseResult.OK {
		parseT.Fatalf("expected snippet to compile, got diagnostic %#v", parseResult.Diagnostic)
	}
	parseHTML := playgroundSandboxHTML(parseResult)
	if strings.Contains(parseHTML, "<script>alert") {
		parseT.Fatalf("sandbox HTML did not escape script text: %s", parseHTML)
	}
	if !strings.Contains(parseHTML, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		parseT.Fatalf("sandbox HTML missing escaped script text: %s", parseHTML)
	}
}

func TestCompilePlaygroundSnippetReportsSyntaxDiagnostic(parseT *testing.T) {
	parseResult := compilePlaygroundSnippet(`package main

func App() ui.Node {
	return Div(
}
`)
	if parseResult.OK {
		parseT.Fatal("expected syntax diagnostic")
	}
	if parseResult.Diagnostic.Code != "GWC-PLAYGROUND-SYNTAX" {
		parseT.Fatalf("diagnostic code = %q, want syntax", parseResult.Diagnostic.Code)
	}
	if parseResult.Diagnostic.Line == 0 {
		parseT.Fatalf("expected source line in diagnostic, got %#v", parseResult.Diagnostic)
	}
}

func TestCompilePlaygroundSnippetReportsUnsupportedDiagnostic(parseT *testing.T) {
	parseResult := compilePlaygroundSnippet(`package main

func App() ui.Node {
	return Script("alert(1)")
}
`)
	if parseResult.OK {
		parseT.Fatal("expected unsupported diagnostic")
	}
	if parseResult.Diagnostic.Code != "GWC-PLAYGROUND-UNSUPPORTED" {
		parseT.Fatalf("diagnostic code = %q, want unsupported", parseResult.Diagnostic.Code)
	}
	if !strings.Contains(parseResult.Diagnostic.Message, "Script") {
		parseT.Fatalf("diagnostic message should name unsupported call, got %q", parseResult.Diagnostic.Message)
	}
}

func TestPlaygroundSourceURLRoundTrip(parseT *testing.T) {
	parseSource := `package main

func App() ui.Node { return H1("Shared") }
`
	parseEncoded := encodePlaygroundSource(parseSource)
	parseDecoded, parseFound, parseErr := decodePlaygroundSource(parseEncoded)
	if parseErr != nil {
		parseT.Fatalf("decode source: %v", parseErr)
	}
	if !parseFound || parseDecoded != parseSource {
		parseT.Fatalf("round trip = (%q, %t), want original source and found", parseDecoded, parseFound)
	}
	parseShareURL := buildPlaygroundShareURL("https://example.test/gallery?v=1", parseSource)
	if !strings.Contains(parseShareURL, "snippet="+parseEncoded) {
		parseT.Fatalf("share URL missing encoded snippet: %s", parseShareURL)
	}
	if !strings.HasSuffix(parseShareURL, "#playground") {
		parseT.Fatalf("share URL should target playground anchor: %s", parseShareURL)
	}
}

func TestBuildPlaygroundShareURLDropsDefaultSnippet(parseT *testing.T) {
	parseShareURL := buildPlaygroundShareURL("https://example.test/gallery?snippet=stale&theme=dark#old", defaultPlaygroundSnippet)
	if strings.Contains(parseShareURL, "snippet=") {
		parseT.Fatalf("default source should remove snippet parameter: %s", parseShareURL)
	}
	if !strings.Contains(parseShareURL, "theme=dark") {
		parseT.Fatalf("share URL should preserve unrelated query parameters: %s", parseShareURL)
	}
	if !strings.HasSuffix(parseShareURL, "#playground") {
		parseT.Fatalf("share URL should target playground anchor: %s", parseShareURL)
	}
}
