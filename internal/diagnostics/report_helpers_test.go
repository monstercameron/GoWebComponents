package diagnostics

import (
	"strings"
	"testing"
)

func TestBuildFallbacksWhenPathAndAppFramesAreUnavailable(t *testing.T) {
	report := Build(Options{
		Summary:       "",
		Code:          "GWC-DIAG-HELPER",
		Headline:      "diagnostics helper fallback",
		Path:          "",
		Runtime:       "runtime consequence",
		Next:          "next action",
		Docs:          "docs/link",
		SkipFunctions: []string{"TestBuildFallbacksWhenPathAndAppFramesAreUnavailable", "testing.tRunner"},
	})

	if report.Summary != "error without message" {
		t.Fatalf("expected summary fallback, got %q", report.Summary)
	}
	if report.Error != report.Summary {
		t.Fatalf("expected error to mirror summary fallback, got %+v", report)
	}
	if report.Where == "" {
		t.Fatalf("expected where fallback to headline/path, got %+v", report)
	}
	if report.Path == "" {
		t.Fatalf("expected path fallback to where, got %+v", report)
	}
}

func TestFormattedHeaderVariants(t *testing.T) {
	withCodeAndHeadline := Report{
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
	if formatted := withCodeAndHeadline.Formatted(); !strings.Contains(formatted, "[GWC-CODE] headline") {
		t.Fatalf("expected code+headline header, got %q", formatted)
	}

	withCodeOnly := withCodeAndHeadline
	withCodeOnly.Headline = ""
	if formatted := withCodeOnly.Formatted(); !strings.Contains(formatted, "[GWC-CODE]") || strings.Contains(formatted, "[GWC-CODE] headline") {
		t.Fatalf("expected code-only header variant, got %q", formatted)
	}

	withHeadlineOnly := withCodeAndHeadline
	withHeadlineOnly.Code = ""
	if formatted := withHeadlineOnly.Formatted(); !strings.Contains(formatted, "headline") || strings.Contains(formatted, "[GWC-CODE]") {
		t.Fatalf("expected headline-only header variant, got %q", formatted)
	}
}

func TestClassifyFrameCoversAppFrameworkAndPlatformBranches(t *testing.T) {
	if got := classifyFrame(frame{Function: "pkg.TestSomething", File: "/tmp/value.go"}); got != "app" {
		t.Fatalf("expected test function to classify as app, got %q", got)
	}
	if got := classifyFrame(frame{Function: frameworkModulePath + "/router.Navigate", File: "/tmp/value.go"}); got != "framework" {
		t.Fatalf("expected framework function classification, got %q", got)
	}
	if got := classifyFrame(frame{Function: frameworkModulePath + "/examples/demo.main", File: "/tmp/value.go"}); got != "app" {
		t.Fatalf("expected framework examples function classification to map to app, got %q", got)
	}
	if got := classifyFrame(frame{Function: "user.main", File: "/work/" + frameworkWorkspaceName + "/router/router.go"}); got != "framework" {
		t.Fatalf("expected workspace framework file classification, got %q", got)
	}
	if got := classifyFrame(frame{Function: "user.main", File: "/work/" + frameworkWorkspaceName + "/examples/demo/main.go"}); got != "app" {
		t.Fatalf("expected workspace examples file classification to map to app, got %q", got)
	}
	if got := classifyFrame(frame{Function: "runtime.goexit", File: "/usr/local/go/src/runtime/proc.go"}); got != "platform" {
		t.Fatalf("expected runtime frame classification, got %q", got)
	}
	if got := classifyFrame(frame{Function: "testing.tRunner", File: "/usr/local/go/src/testing/testing.go"}); got != "platform" {
		t.Fatalf("expected testing frame classification, got %q", got)
	}
}

func TestFrameFormattingHelpers(t *testing.T) {
	if got := sanitizeFunction(" pkg.fn(value) "); got != "pkg.fn" {
		t.Fatalf("expected function sanitize to strip args and whitespace, got %q", got)
	}
	if got := sanitizeFunction("   "); got != "" {
		t.Fatalf("expected blank function sanitize to stay blank, got %q", got)
	}

	if got := shortenFilePath("/tmp/" + frameworkWorkspaceName + "/router/contracts.go"); got != "router/contracts.go" {
		t.Fatalf("expected workspace marker file shortening, got %q", got)
	}
	if got := shortenFilePath("a/b"); got != "a/b" {
		t.Fatalf("expected short path passthrough, got %q", got)
	}
	if got := shortenFilePath("/a/b/c/d/e.go"); got != "c/d/e.go" {
		t.Fatalf("expected tail-only path shortening, got %q", got)
	}

	if got := formatFrame(frame{Function: "pkg.fn", File: "", Line: 12}); got != "pkg.fn" {
		t.Fatalf("expected formatFrame to return function when location missing, got %q", got)
	}
	if got := formatFrame(frame{Function: "pkg.fn", File: "/a/b/c.go", Line: 0}); !strings.Contains(got, "pkg.fn at a/b/c.go") {
		t.Fatalf("expected formatFrame without line number, got %q", got)
	}

	values := appendFrame(nil, frame{})
	if len(values) != 0 {
		t.Fatalf("expected appendFrame to skip empty formatted frame, got %+v", values)
	}
	values = appendFrame(values, frame{Function: "pkg.fn", File: "/a/b/c.go", Line: 9})
	if len(values) != 1 || !strings.Contains(values[0], "pkg.fn") {
		t.Fatalf("expected appendFrame to append formatted frame, got %+v", values)
	}
}

