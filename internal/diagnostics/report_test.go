package diagnostics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildFormatsUnifiedContract(t *testing.T) {
	report := Build(Options{
		Summary:  "render failed",
		Code:     "GWC-EXAMPLE-SERVER-REQUEST",
		Headline: "server failure in renderExample",
		Path:     "/products",
		Runtime:  "this request failed before HTML could be returned.",
		Next:     "Inspect the SSR render path and bootstrap inputs for this request.",
		Docs:     "ACTIONABLE_ERRORS.md#gwc-example-server-request",
	})

	formatted := report.Formatted()
	if !strings.Contains(formatted, "[GWC-EXAMPLE-SERVER-REQUEST] server failure in renderExample") ||
		!strings.Contains(formatted, "where: ") ||
		!strings.Contains(formatted, "path: /products") ||
		!strings.Contains(formatted, "error: render failed") ||
		!strings.Contains(formatted, "runtime: this request failed before HTML could be returned.") ||
		!strings.Contains(formatted, "next: Inspect the SSR render path and bootstrap inputs for this request.") ||
		!strings.Contains(formatted, "docs: ACTIONABLE_ERRORS.md#gwc-example-server-request") ||
		!strings.Contains(formatted, "stack:\napp:") {
		t.Fatalf("expected unified report contract, got %q", formatted)
	}
	if strings.Contains(formatted, "(0x") {
		t.Fatalf("expected sanitized stack output, got %q", formatted)
	}
	if report.Where == "" {
		t.Fatal("expected top frame to be populated")
	}
}

func TestWriteHTTPErrorWritesStructuredBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	report := Build(Options{
		Summary:  "build failed",
		Code:     "GWC-TOOL-LIVERELOAD",
		Headline: "tool failure in livereload.handleHTML",
		Path:     "/",
		Runtime:  "the request returned HTTP 500 and live reload could not inject the client script.",
		Next:     "Verify the livereload client script path and rebuild the dev server.",
		Docs:     "ACTIONABLE_ERRORS.md#gwc-tool-livereload",
	})

	WriteHTTPError(recorder, http.StatusInternalServerError, report)

	response := recorder.Result()
	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.StatusCode)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "GWC-TOOL-LIVERELOAD") || !strings.Contains(body, "runtime: the request returned HTTP 500") {
		t.Fatalf("expected structured response body, got %q", body)
	}
	if contentType := response.Header.Get("Content-Type"); contentType != "text/plain; charset=utf-8" {
		t.Fatalf("expected text response content type, got %q", contentType)
	}
	if !strings.Contains(recorder.Body.String(), report.Formatted()) {
		t.Fatalf("expected response body to match formatted report, got %q", recorder.Body.String())
	}
	if !strings.Contains(report.Formatted(), "where: ") {
		t.Fatalf("expected formatted report to include where line, got %q", report.Formatted())
	}
	if report.Where == "" {
		t.Fatal("expected where to be populated")
	}
	if !strings.Contains(report.Formatted(), "app:") {
		t.Fatalf("expected app stack section, got %q", report.Formatted())
	}
	if report.Error != "build failed" {
		t.Fatalf("expected error line to mirror summary, got %+v", report)
	}
	if report.Code != "GWC-TOOL-LIVERELOAD" {
		t.Fatalf("expected stable code, got %+v", report)
	}
	if !strings.Contains(report.Formatted(), "docs: ACTIONABLE_ERRORS.md#gwc-tool-livereload") {
		t.Fatalf("expected docs line, got %q", report.Formatted())
	}
	if !strings.Contains(report.Formatted(), "next: Verify the livereload client script path and rebuild the dev server.") {
		t.Fatalf("expected next guidance, got %q", report.Formatted())
	}
	if !strings.Contains(report.Formatted(), "path: /") {
		t.Fatalf("expected path line, got %q", report.Formatted())
	}
	if report.Runtime == "" {
		t.Fatal("expected runtime consequence")
	}
	if report.Next == "" {
		t.Fatal("expected remediation guidance")
	}
	if report.Docs == "" {
		t.Fatal("expected docs anchor")
	}
	if report.Headline == "" {
		t.Fatal("expected headline")
	}
	if report.Summary == "" {
		t.Fatal("expected summary")
	}
	if report.Path == "" {
		t.Fatal("expected path")
	}
	if len(report.AppFrames) == 0 {
		t.Fatal("expected app frames")
	}
	if recorder.Body.Len() == 0 {
		t.Fatal("expected response body")
	}
	if response.Header.Get("Content-Type") == "" {
		t.Fatal("expected content type header")
	}
	if report.Formatted() == "" {
		t.Fatal("expected formatted report")
	}
	if !strings.Contains(report.Formatted(), "error: build failed") {
		t.Fatalf("expected error line, got %q", report.Formatted())
	}
	if !strings.Contains(report.Formatted(), "stack:") {
		t.Fatalf("expected stack section, got %q", report.Formatted())
	}
	if !strings.Contains(report.Formatted(), report.Where) {
		t.Fatalf("expected where value in report, got %q", report.Formatted())
	}
	if report.Where == report.Path && report.Where != "/" {
		t.Fatalf("expected where to prefer a stack frame, got %+v", report)
	}
	if response.Status != "500 Internal Server Error" {
		t.Fatalf("expected recorder status, got %q", response.Status)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "stack:") {
		t.Fatalf("expected stack details in response body, got %q", body)
	}
	if !strings.Contains(report.Formatted(), "[GWC-TOOL-LIVERELOAD]") {
		t.Fatalf("expected code line, got %q", report.Formatted())
	}
	if len(report.FrameworkFrames) == 0 && len(report.PlatformFrames) == 0 {
		return
	}
	if !strings.Contains(report.Formatted(), "framework: GWC") && !strings.Contains(report.Formatted(), "platform: GOLANG") {
		t.Fatalf("expected grouped stack labels when extra sections exist, got %q", report.Formatted())
	}
	if len(report.AppFrames) > visibleAppFrameLimit+1 {
		t.Fatalf("expected app frames to be limited, got %+v", report.AppFrames)
	}
	if strings.Contains(report.Formatted(), "github.com/monstercameron/GoWebComponents/internal/diagnostics.Build") {
		t.Fatalf("expected internal builder frames to be hidden, got %q", report.Formatted())
	}
	if !strings.Contains(recorder.Body.String(), "docs:") {
		t.Fatalf("expected docs line in response body, got %q", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "next:") {
		t.Fatalf("expected next line in response body, got %q", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "runtime:") {
		t.Fatalf("expected runtime line in response body, got %q", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "where:") {
		t.Fatalf("expected where line in response body, got %q", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "path: /") {
		t.Fatalf("expected path line in response body, got %q", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "error: build failed") {
		t.Fatalf("expected error line in response body, got %q", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "[GWC-TOOL-LIVERELOAD] tool failure in livereload.handleHTML") {
		t.Fatalf("expected headline in response body, got %q", recorder.Body.String())
	}
	if !strings.HasPrefix(recorder.Body.String(), "build failed\n") {
		t.Fatalf("expected summary first, got %q", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "(0x") {
		t.Fatalf("expected sanitized body, got %q", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "internal/diagnostics.WriteHTTPError") {
		t.Fatalf("expected internal helper frames to be hidden, got %q", recorder.Body.String())
	}
	if report.Where == "" || report.Path == "" || report.Runtime == "" || report.Next == "" || report.Docs == "" {
		t.Fatalf("expected report metadata to stay populated, got %+v", report)
	}
	if !strings.Contains(report.Formatted(), "tool failure in livereload.handleHTML") {
		t.Fatalf("expected stable headline, got %q", report.Formatted())
	}
	if report.Summary != "build failed" {
		t.Fatalf("expected summary round-trip, got %+v", report)
	}
	if report.Headline != "tool failure in livereload.handleHTML" {
		t.Fatalf("expected headline round-trip, got %+v", report)
	}
	if report.Docs != "ACTIONABLE_ERRORS.md#gwc-tool-livereload" {
		t.Fatalf("expected docs round-trip, got %+v", report)
	}
	if report.Path != "/" {
		t.Fatalf("expected path round-trip, got %+v", report)
	}
	if report.Error != report.Summary {
		t.Fatalf("expected error to mirror summary, got %+v", report)
	}
	if response.ContentLength != 0 {
		return
	}
	if recorder.Body.Len() <= 0 {
		t.Fatal("expected non-empty body")
	}
	if report.Formatted() == "build failed" {
		t.Fatalf("expected full contract, got %q", report.Formatted())
	}
	if !strings.Contains(recorder.Body.String(), "app:") {
		t.Fatalf("expected app frames in response body, got %q", recorder.Body.String())
	}
	if !strings.Contains(report.Formatted(), report.AppFrames[0]) {
		t.Fatalf("expected first app frame to appear, got %q", report.Formatted())
	}
	if report.Where == "/" {
		t.Fatalf("expected where to use stack frame when available, got %+v", report)
	}
}
