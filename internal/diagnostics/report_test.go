//go:build !js

package diagnostics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildFormatsUnifiedContract(parseT *testing.T) {
	parseReport := Build(Options{
		Summary:  "render failed",
		Code:     "GWC-EXAMPLE-SERVER-REQUEST",
		Headline: "server failure in renderExample",
		Path:     "/products",
		Runtime:  "this request failed before HTML could be returned.",
		Next:     "Inspect the SSR render path and bootstrap inputs for this request.",
		Docs:     "ACTIONABLE_ERRORS.md#gwc-example-server-request",
	})

	parseFormatted := parseReport.Formatted()
	if !strings.Contains(parseFormatted, "[GWC-EXAMPLE-SERVER-REQUEST] server failure in renderExample") ||
		!strings.Contains(parseFormatted, "where: ") ||
		!strings.Contains(parseFormatted, "path: /products") ||
		!strings.Contains(parseFormatted, "error: render failed") ||
		!strings.Contains(parseFormatted, "runtime: this request failed before HTML could be returned.") ||
		!strings.Contains(parseFormatted, "next: Inspect the SSR render path and bootstrap inputs for this request.") ||
		!strings.Contains(parseFormatted, "docs: ACTIONABLE_ERRORS.md#gwc-example-server-request") ||
		!strings.Contains(parseFormatted, "stack:\napp:") {
		parseT.Fatalf("expected unified report contract, got %q", parseFormatted)
	}
	if strings.Contains(parseFormatted, "(0x") {
		parseT.Fatalf("expected sanitized stack output, got %q", parseFormatted)
	}
	if parseReport.Where == "" {
		parseT.Fatal("expected top frame to be populated")
	}
}

func TestWriteHTTPErrorWritesStructuredBody(parseT *testing.T) {
	parseRecorder := httptest.NewRecorder()
	parseReport := Build(Options{
		Summary:  "build failed",
		Code:     "GWC-TOOL-LIVERELOAD",
		Headline: "tool failure in livereload.handleHTML",
		Path:     "/",
		Runtime:  "the request returned HTTP 500 and live reload could not inject the client script.",
		Next:     "Verify the livereload client script path and rebuild the dev server.",
		Docs:     "ACTIONABLE_ERRORS.md#gwc-tool-livereload",
	})

	WriteHTTPError(parseRecorder, http.StatusInternalServerError, parseReport)

	parseResponse := parseRecorder.Result()
	if parseResponse.StatusCode != http.StatusInternalServerError {
		parseT.Fatalf("expected status 500, got %d", parseResponse.StatusCode)
	}
	// #60: the default HTTP body is the CLIENT-SAFE subset — code/next/docs — and
	// must NOT disclose the stack, build paths, or where/path/error/runtime detail.
	parseBody := parseRecorder.Body.String()
	if !strings.Contains(parseBody, "GWC-TOOL-LIVERELOAD") {
		parseT.Fatalf("expected client-safe body to include the error code, got %q", parseBody)
	}
	for _, parseLeak := range []string{"stack:", "app:", "where: ", "runtime: ", "error: "} {
		if strings.Contains(parseBody, parseLeak) {
			parseT.Fatalf("client HTTP body leaked internal diagnostic detail %q: %q", parseLeak, parseBody)
		}
	}
	if parseBody != parseReport.FormattedPublic() {
		parseT.Fatalf("expected body to equal FormattedPublic(), got %q", parseBody)
	}
	if parseContentType := parseResponse.Header.Get("Content-Type"); parseContentType != "text/plain; charset=utf-8" {
		parseT.Fatalf("expected text response content type, got %q", parseContentType)
	}
	if !strings.Contains(parseReport.Formatted(), "where: ") {
		parseT.Fatalf("expected formatted report to include where line, got %q", parseReport.Formatted())
	}
	if parseReport.Where == "" {
		parseT.Fatal("expected where to be populated")
	}
	if !strings.Contains(parseReport.Formatted(), "app:") {
		parseT.Fatalf("expected app stack section, got %q", parseReport.Formatted())
	}
	if parseReport.Error != "build failed" {
		parseT.Fatalf("expected error line to mirror summary, got %+v", parseReport)
	}
	if parseReport.Code != "GWC-TOOL-LIVERELOAD" {
		parseT.Fatalf("expected stable code, got %+v", parseReport)
	}
	if !strings.Contains(parseReport.Formatted(), "docs: ACTIONABLE_ERRORS.md#gwc-tool-livereload") {
		parseT.Fatalf("expected docs line, got %q", parseReport.Formatted())
	}
	if !strings.Contains(parseReport.Formatted(), "next: Verify the livereload client script path and rebuild the dev server.") {
		parseT.Fatalf("expected next guidance, got %q", parseReport.Formatted())
	}
	if !strings.Contains(parseReport.Formatted(), "path: /") {
		parseT.Fatalf("expected path line, got %q", parseReport.Formatted())
	}
	if parseReport.Runtime == "" {
		parseT.Fatal("expected runtime consequence")
	}
	if parseReport.Next == "" {
		parseT.Fatal("expected remediation guidance")
	}
	if parseReport.Docs == "" {
		parseT.Fatal("expected docs anchor")
	}
	if parseReport.Headline == "" {
		parseT.Fatal("expected headline")
	}
	if parseReport.Summary == "" {
		parseT.Fatal("expected summary")
	}
	if parseReport.Path == "" {
		parseT.Fatal("expected path")
	}
	if len(parseReport.AppFrames) == 0 {
		parseT.Fatal("expected app frames")
	}
	if parseRecorder.Body.Len() == 0 {
		parseT.Fatal("expected response body")
	}
	if parseResponse.Header.Get("Content-Type") == "" {
		parseT.Fatal("expected content type header")
	}
	if parseReport.Formatted() == "" {
		parseT.Fatal("expected formatted report")
	}
	if !strings.Contains(parseReport.Formatted(), "error: build failed") {
		parseT.Fatalf("expected error line, got %q", parseReport.Formatted())
	}
	if !strings.Contains(parseReport.Formatted(), "stack:") {
		parseT.Fatalf("expected stack section, got %q", parseReport.Formatted())
	}
	if !strings.Contains(parseReport.Formatted(), parseReport.Where) {
		parseT.Fatalf("expected where value in report, got %q", parseReport.Formatted())
	}
	if parseReport.Where == parseReport.Path && parseReport.Where != "/" {
		parseT.Fatalf("expected where to prefer a stack frame, got %+v", parseReport)
	}
	if parseResponse.Status != "500 Internal Server Error" {
		parseT.Fatalf("expected recorder status, got %q", parseResponse.Status)
	}
	if !strings.Contains(parseReport.Formatted(), "[GWC-TOOL-LIVERELOAD]") {
		parseT.Fatalf("expected code line, got %q", parseReport.Formatted())
	}
	if len(parseReport.FrameworkFrames) == 0 && len(parseReport.PlatformFrames) == 0 {
		return
	}
	if !strings.Contains(parseReport.Formatted(), "framework: GWC") && !strings.Contains(parseReport.Formatted(), "platform: GOLANG") {
		parseT.Fatalf("expected grouped stack labels when extra sections exist, got %q", parseReport.Formatted())
	}
	if len(parseReport.AppFrames) > visibleAppFrameLimit+1 {
		parseT.Fatalf("expected app frames to be limited, got %+v", parseReport.AppFrames)
	}
	if strings.Contains(parseReport.Formatted(), "github.com/monstercameron/GoWebComponents/v4/internal/diagnostics.Build") {
		parseT.Fatalf("expected internal builder frames to be hidden, got %q", parseReport.Formatted())
	}
	if !strings.Contains(parseRecorder.Body.String(), "docs:") {
		parseT.Fatalf("expected docs line in response body, got %q", parseRecorder.Body.String())
	}
	if !strings.Contains(parseRecorder.Body.String(), "next:") {
		parseT.Fatalf("expected next line in response body, got %q", parseRecorder.Body.String())
	}
	if !strings.Contains(parseRecorder.Body.String(), "[GWC-TOOL-LIVERELOAD] tool failure in livereload.handleHTML") {
		parseT.Fatalf("expected headline in response body, got %q", parseRecorder.Body.String())
	}
	if !strings.HasPrefix(parseRecorder.Body.String(), "build failed\n") {
		parseT.Fatalf("expected summary first, got %q", parseRecorder.Body.String())
	}
	if strings.Contains(parseRecorder.Body.String(), "(0x") {
		parseT.Fatalf("expected sanitized body, got %q", parseRecorder.Body.String())
	}
	if strings.Contains(parseRecorder.Body.String(), "internal/diagnostics.WriteHTTPError") {
		parseT.Fatalf("expected internal helper frames to be hidden, got %q", parseRecorder.Body.String())
	}
	if parseReport.Where == "" || parseReport.Path == "" || parseReport.Runtime == "" || parseReport.Next == "" || parseReport.Docs == "" {
		parseT.Fatalf("expected report metadata to stay populated, got %+v", parseReport)
	}
	if !strings.Contains(parseReport.Formatted(), "tool failure in livereload.handleHTML") {
		parseT.Fatalf("expected stable headline, got %q", parseReport.Formatted())
	}
	if parseReport.Summary != "build failed" {
		parseT.Fatalf("expected summary round-trip, got %+v", parseReport)
	}
	if parseReport.Headline != "tool failure in livereload.handleHTML" {
		parseT.Fatalf("expected headline round-trip, got %+v", parseReport)
	}
	if parseReport.Docs != "ACTIONABLE_ERRORS.md#gwc-tool-livereload" {
		parseT.Fatalf("expected docs round-trip, got %+v", parseReport)
	}
	if parseReport.Path != "/" {
		parseT.Fatalf("expected path round-trip, got %+v", parseReport)
	}
	if parseReport.Error != parseReport.Summary {
		parseT.Fatalf("expected error to mirror summary, got %+v", parseReport)
	}
	if parseResponse.ContentLength != 0 {
		return
	}
	if parseRecorder.Body.Len() <= 0 {
		parseT.Fatal("expected non-empty body")
	}
	if parseReport.Formatted() == "build failed" {
		parseT.Fatalf("expected full contract, got %q", parseReport.Formatted())
	}
	if !strings.Contains(parseReport.Formatted(), parseReport.AppFrames[0]) {
		parseT.Fatalf("expected first app frame to appear, got %q", parseReport.Formatted())
	}
	if parseReport.Where == "/" {
		parseT.Fatalf("expected where to use stack frame when available, got %+v", parseReport)
	}
}

// TestWriteHTTPErrorVerboseWritesFullReport pins the #60 opt-in: the verbose
// variant intentionally writes the FULL Formatted() report (stack + where/path/
// error/runtime) for trusted/dev endpoints, while the default WriteHTTPError does
// not. Guards that both halves of the gate stay wired.
func TestWriteHTTPErrorVerboseWritesFullReport(parseT *testing.T) {
	parseReport := Build(Options{
		Summary:  "build failed",
		Code:     "GWC-TOOL-LIVERELOAD",
		Headline: "tool failure",
		Path:     "/",
		Runtime:  "the request returned HTTP 500.",
		Next:     "Rebuild.",
		Docs:     "ACTIONABLE_ERRORS.md#x",
	})

	parseVerbose := httptest.NewRecorder()
	WriteHTTPErrorVerbose(parseVerbose, http.StatusInternalServerError, parseReport)
	parseVerboseBody := parseVerbose.Body.String()
	if parseVerboseBody != parseReport.Formatted() {
		parseT.Fatalf("verbose body must equal the full Formatted() report, got %q", parseVerboseBody)
	}
	for _, parseWant := range []string{"stack:", "runtime: ", "where: "} {
		if !strings.Contains(parseVerboseBody, parseWant) {
			parseT.Fatalf("verbose body must include %q for a trusted endpoint, got %q", parseWant, parseVerboseBody)
		}
	}

	// The default variant on the same report must remain client-safe.
	parseDefault := httptest.NewRecorder()
	WriteHTTPError(parseDefault, http.StatusInternalServerError, parseReport)
	if strings.Contains(parseDefault.Body.String(), "stack:") {
		parseT.Fatalf("default WriteHTTPError must not leak the stack, got %q", parseDefault.Body.String())
	}
}
