//go:build !js || !wasm

package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// TestRenderToStringRequestReportBuildsDiagnostic verifies request diagnostics metadata.
func TestRenderToStringRequestReportBuildsDiagnostic(parseT *testing.T) {
	parseErr := errors.New("render failed")
	parseReport := renderToStringRequestReport("/pricing", parseErr)

	if parseReport.Summary != "render failed" {
		parseT.Fatalf("expected summary to match error, got %#v", parseReport)
	}
	if parseReport.Code != "GWC-EXAMPLE-SERVER-REQUEST" {
		parseT.Fatalf("unexpected report code %#v", parseReport)
	}
	if parseReport.Path != "/pricing" || parseReport.Docs != renderToStringRequestDocs {
		parseT.Fatalf("unexpected request report metadata %#v", parseReport)
	}
}

// TestRenderCardAndHeadMarkupRender verifies the example server render primitives.
func TestRenderCardAndHeadMarkupRender(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(renderCard())
	if parseErr != nil {
		parseT.Fatalf("RenderToString renderCard: %v", parseErr)
	}
	for _, parseNeedle := range []string{
		"Server render",
		"ui.RenderToString generated this card",
		"The server never mounted a browser runtime.",
	} {
		if !strings.Contains(parseMarkup, parseNeedle) {
			parseT.Fatalf("expected card markup to contain %q", parseNeedle)
		}
	}

	parseHeadMarkup, parseErr := renderHeadMarkup()
	if parseErr != nil {
		parseT.Fatalf("renderHeadMarkup: %v", parseErr)
	}
	for _, parseNeedle := range []string{
		"ui.RenderToString",
		"Minimal prerender-style HTML generation",
		"schema.org",
	} {
		if !strings.Contains(parseHeadMarkup, parseNeedle) {
			parseT.Fatalf("expected head markup to contain %q", parseNeedle)
		}
	}
}

// TestHandleRenderToStringWritesHTML verifies the request handler success response.
func TestHandleRenderToStringWritesHTML(parseT *testing.T) {
	parseRecorder := httptest.NewRecorder()
	parseRequest := httptest.NewRequest(http.MethodGet, "/", nil)

	handleRenderToString(parseRecorder, parseRequest)

	if parseRecorder.Code != http.StatusOK {
		parseT.Fatalf("expected handler status 200, got %d", parseRecorder.Code)
	}
	if parseGot := parseRecorder.Header().Get("Content-Type"); parseGot != "text/html; charset=utf-8" {
		parseT.Fatalf("unexpected content type %q", parseGot)
	}
	for _, parseNeedle := range []string{
		"<!DOCTYPE html>",
		"ui.RenderToString",
		"Returned HTML string",
		"Rendered preview",
		"Server render",
	} {
		if !strings.Contains(parseRecorder.Body.String(), parseNeedle) {
			parseT.Fatalf("expected handler body to contain %q", parseNeedle)
		}
	}
}
