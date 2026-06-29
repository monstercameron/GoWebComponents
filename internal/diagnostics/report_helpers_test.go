package diagnostics

import (
	"strings"
	"testing"
)

func TestBuildFallbacksWhenPathAndAppFramesAreUnavailable(parseT *testing.T) {
	parseReport := Build(Options{
		Summary:       "",
		Code:          "GWC-DIAG-HELPER",
		Headline:      "diagnostics helper fallback",
		Path:          "",
		Runtime:       "runtime consequence",
		Next:          "next action",
		Docs:          "docs/link",
		SkipFunctions: []string{"TestBuildFallbacksWhenPathAndAppFramesAreUnavailable", "testing.tRunner"},
	})

	if parseReport.Summary != "error without message" {
		parseT.Fatalf("expected summary fallback, got %q", parseReport.Summary)
	}
	if parseReport.Error != parseReport.Summary {
		parseT.Fatalf("expected error to mirror summary fallback, got %+v", parseReport)
	}
	if parseReport.Where == "" {
		parseT.Fatalf("expected where fallback to headline/path, got %+v", parseReport)
	}
	if parseReport.Path == "" {
		parseT.Fatalf("expected path fallback to where, got %+v", parseReport)
	}
}

func TestFormattedHeaderVariants(parseT *testing.T) {
	parseWithCodeAndHeadline := Report{
		Summary:  "boom",
		Code:     "GWC-CODE",
		Headline: "headline",
		Where:    "where",
		Path:     "path",
		Error:    "boom",
		Runtime:  "runtime",
		Next:     "next",
		Docs:     "docs",
	}
	if parseFormatted := parseWithCodeAndHeadline.Formatted(); !strings.Contains(parseFormatted, "[GWC-CODE] headline") {
		parseT.Fatalf("expected code+headline header, got %q", parseFormatted)
	}

	parseWithCodeOnly := parseWithCodeAndHeadline
	parseWithCodeOnly.Headline = ""
	if parseFormatted2 := parseWithCodeOnly.Formatted(); !strings.Contains(parseFormatted2, "[GWC-CODE]") || strings.Contains(parseFormatted2, "[GWC-CODE] headline") {
		parseT.Fatalf("expected code-only header variant, got %q", parseFormatted2)
	}

	parseWithHeadlineOnly := parseWithCodeAndHeadline
	parseWithHeadlineOnly.Code = ""
	if parseFormatted3 := parseWithHeadlineOnly.Formatted(); !strings.Contains(parseFormatted3, "headline") || strings.Contains(parseFormatted3, "[GWC-CODE]") {
		parseT.Fatalf("expected headline-only header variant, got %q", parseFormatted3)
	}
}

func TestClassifyFrameCoversAppFrameworkAndPlatformBranches(parseT *testing.T) {
	if parseGot := classifyFrame(frame{Function: "pkg.TestSomething", File: "/tmp/value.go"}); parseGot != "app" {
		parseT.Fatalf("expected test function to classify as app, got %q", parseGot)
	}
	if parseGot2 := classifyFrame(frame{Function: frameworkModulePath + "/router.Navigate", File: "/tmp/value.go"}); parseGot2 != "framework" {
		parseT.Fatalf("expected framework function classification, got %q", parseGot2)
	}
	if parseGot3 := classifyFrame(frame{Function: frameworkModulePath + "/examples/demo.main", File: "/tmp/value.go"}); parseGot3 != "app" {
		parseT.Fatalf("expected framework examples function classification to map to app, got %q", parseGot3)
	}
	if parseGot4 := classifyFrame(frame{Function: "user.main", File: "/work/" + frameworkWorkspaceName + "/router/router.go"}); parseGot4 != "framework" {
		parseT.Fatalf("expected workspace framework file classification, got %q", parseGot4)
	}
	if parseGot5 := classifyFrame(frame{Function: "user.main", File: "/work/" + frameworkWorkspaceName + "/examples/demo/main.go"}); parseGot5 != "app" {
		parseT.Fatalf("expected workspace examples file classification to map to app, got %q", parseGot5)
	}
	if parseGot6 := classifyFrame(frame{Function: "runtime.goexit", File: "/usr/local/go/src/runtime/proc.go"}); parseGot6 != "platform" {
		parseT.Fatalf("expected runtime frame classification, got %q", parseGot6)
	}
	if parseGot7 := classifyFrame(frame{Function: "testing.tRunner", File: "/usr/local/go/src/testing/testing.go"}); parseGot7 != "platform" {
		parseT.Fatalf("expected testing frame classification, got %q", parseGot7)
	}
}

func TestFrameFormattingHelpers(parseT *testing.T) {
	if parseGot := sanitizeFunction(" pkg.fn(value) "); parseGot != "pkg.fn" {
		parseT.Fatalf("expected function sanitize to strip args and whitespace, got %q", parseGot)
	}
	if parseGot2 := sanitizeFunction("   "); parseGot2 != "" {
		parseT.Fatalf("expected blank function sanitize to stay blank, got %q", parseGot2)
	}

	if parseGot3 := shortenFilePath("/tmp/" + frameworkWorkspaceName + "/router/contracts.go"); parseGot3 != "router/contracts.go" {
		parseT.Fatalf("expected workspace marker file shortening, got %q", parseGot3)
	}
	if parseGot4 := shortenFilePath("a/b"); parseGot4 != "a/b" {
		parseT.Fatalf("expected short path passthrough, got %q", parseGot4)
	}
	if parseGot5 := shortenFilePath("/a/b/c/d/e.go"); parseGot5 != "c/d/e.go" {
		parseT.Fatalf("expected tail-only path shortening, got %q", parseGot5)
	}

	if parseGot6 := formatFrame(frame{Function: "pkg.fn", File: "", Line: 12}); parseGot6 != "pkg.fn" {
		parseT.Fatalf("expected formatFrame to return function when location missing, got %q", parseGot6)
	}
	if parseGot7 := formatFrame(frame{Function: "pkg.fn", File: "/a/b/c.go", Line: 0}); !strings.Contains(parseGot7, "pkg.fn at a/b/c.go") {
		parseT.Fatalf("expected formatFrame without line number, got %q", parseGot7)
	}

	parseValues := appendFrame(nil, frame{})
	if len(parseValues) != 0 {
		parseT.Fatalf("expected appendFrame to skip empty formatted frame, got %+v", parseValues)
	}
	parseValues = appendFrame(parseValues, frame{Function: "pkg.fn", File: "/a/b/c.go", Line: 9})
	if len(parseValues) != 1 || !strings.Contains(parseValues[0], "pkg.fn") {
		parseT.Fatalf("expected appendFrame to append formatted frame, got %+v", parseValues)
	}
}
