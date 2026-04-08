//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func newTestServer() *secureFormsServer {
	return &secureFormsServer{}
}

func TestIndexRendersCSRFAndMultipartForm(parseT *testing.T) {
	parseServer := newTestServer()
	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseRes := httptest.NewRecorder()

	parseServer.handleIndex(parseRes, parseReq)
	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected OK, got %d", parseRes.Code)
	}
	parseBody := parseRes.Body.String()
	parseChecks := []string{
		`name="csrf_token"`,
		`enctype="multipart/form-data"`,
		`Request pricing`,
		`Upload asset`,
	}
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseBody, parseCheck) {
			parseT.Fatalf("expected body to contain %q, got %q", parseCheck, parseBody)
		}
	}
	if len(parseRes.Result().Cookies()) == 0 {
		parseT.Fatal("expected csrf cookie")
	}
}

func TestQuoteValidationRoundTripPreservesValues(parseT *testing.T) {
	parseServer := newTestServer()
	parseCsrfToken, parseCsrfCookie := loadCSRF(parseT, parseServer)
	parseForm := strings.NewReader("csrf_token=" + parseCsrfToken + "&name=&email=buyer%40example.com&company=Atlas+Studio&timeline=quarter&notes=Need+review")
	parseReq := httptest.NewRequest(http.MethodPost, "/quote", parseForm)
	parseReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseReq.Header.Set("Origin", "http://example.com")
	parseReq.Host = "example.com"
	parseReq.AddCookie(parseCsrfCookie)
	parseRes := httptest.NewRecorder()

	parseServer.handleQuote(parseRes, parseReq)
	if parseRes.Code != http.StatusBadRequest {
		parseT.Fatalf("expected validation error status, got %d", parseRes.Code)
	}
	parseBody := parseRes.Body.String()
	if !strings.Contains(parseBody, "Name is required.") || !strings.Contains(parseBody, `value="buyer@example.com"`) || !strings.Contains(parseBody, `value="Atlas Studio"`) {
		parseT.Fatalf("expected validation round-trip body, got %q", parseBody)
	}
}

func TestQuoteSuccessRedirectsAfterSubmit(parseT *testing.T) {
	parseServer := newTestServer()
	parseCsrfToken, parseCsrfCookie := loadCSRF(parseT, parseServer)
	parseForm := strings.NewReader("csrf_token=" + parseCsrfToken + "&name=Ada+Buyer&email=buyer%40example.com&company=Atlas+Studio&timeline=30_days&notes=Ship+quote")
	parseReq := httptest.NewRequest(http.MethodPost, "/quote", parseForm)
	parseReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseReq.Header.Set("Origin", "http://example.com")
	parseReq.Host = "example.com"
	parseReq.AddCookie(parseCsrfCookie)
	parseRes := httptest.NewRecorder()

	parseServer.handleQuote(parseRes, parseReq)
	if parseRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected redirect, got %d", parseRes.Code)
	}
	if parseLocation := parseRes.Header().Get("Location"); !strings.Contains(parseLocation, "notice=") {
		parseT.Fatalf("expected redirect notice, got %q", parseLocation)
	}
}

func TestUploadValidationRoundTripPreservesLabel(parseT *testing.T) {
	parseServer := newTestServer()
	parseCsrfToken, parseCsrfCookie := loadCSRF(parseT, parseServer)
	parseBody := &bytes.Buffer{}
	parseWriter := multipart.NewWriter(parseBody)
	_ = parseWriter.WriteField("csrf_token", parseCsrfToken)
	_ = parseWriter.WriteField("label", "Warehouse board")
	_ = parseWriter.Close()

	parseReq := httptest.NewRequest(http.MethodPost, "/upload", parseBody)
	parseReq.Header.Set("Content-Type", parseWriter.FormDataContentType())
	parseReq.Header.Set("Origin", "http://example.com")
	parseReq.Host = "example.com"
	parseReq.AddCookie(parseCsrfCookie)
	parseRes := httptest.NewRecorder()

	parseServer.handleUpload(parseRes, parseReq)
	if parseRes.Code != http.StatusBadRequest {
		parseT.Fatalf("expected upload validation error, got %d", parseRes.Code)
	}
	if parseGot := parseRes.Body.String(); !strings.Contains(parseGot, "Choose a PNG or JPEG file.") || !strings.Contains(parseGot, `value="Warehouse board"`) {
		parseT.Fatalf("expected upload validation round-trip, got %q", parseGot)
	}
}

func TestUploadSuccessRedirectsAfterMultipartSubmit(parseT *testing.T) {
	parseServer := newTestServer()
	parseCsrfToken, parseCsrfCookie := loadCSRF(parseT, parseServer)
	parseBody := &bytes.Buffer{}
	parseWriter := multipart.NewWriter(parseBody)
	_ = parseWriter.WriteField("csrf_token", parseCsrfToken)
	_ = parseWriter.WriteField("label", "Warehouse board")
	parsePart, parseErr := parseWriter.CreateFormFile("asset", "board.png")
	if parseErr != nil {
		parseT.Fatalf("unexpected multipart file creation error: %v", parseErr)
	}
	if _, parseErr2 := io.Copy(parsePart, bytes.NewBufferString("\x89PNG\r\n\x1a\nfakepng")); parseErr2 != nil {
		parseT.Fatalf("unexpected multipart write error: %v", parseErr2)
	}
	_ = parseWriter.Close()

	parseReq := httptest.NewRequest(http.MethodPost, "/upload", parseBody)
	parseReq.Header.Set("Content-Type", parseWriter.FormDataContentType())
	parseReq.Header.Set("Origin", "http://example.com")
	parseReq.Host = "example.com"
	parseReq.AddCookie(parseCsrfCookie)
	parseRes := httptest.NewRecorder()

	parseServer.handleUpload(parseRes, parseReq)
	if parseRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected upload redirect, got %d", parseRes.Code)
	}
	parseLocation := parseRes.Header().Get("Location")
	if !strings.Contains(parseLocation, "asset=board.png") || !strings.Contains(parseLocation, "notice=") {
		parseT.Fatalf("expected asset redirect summary, got %q", parseLocation)
	}
}

func TestPostRejectsMissingCSRFTokens(parseT *testing.T) {
	parseServer := newTestServer()
	parseReq := httptest.NewRequest(http.MethodPost, "/quote", strings.NewReader("name=Ada"))
	parseReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseReq.Header.Set("Origin", "http://example.com")
	parseReq.Host = "example.com"
	parseRes := httptest.NewRecorder()

	parseServer.handleQuote(parseRes, parseReq)
	if parseRes.Code != http.StatusForbidden {
		parseT.Fatalf("expected forbidden, got %d", parseRes.Code)
	}
	if !strings.Contains(parseRes.Body.String(), "csrf") {
		parseT.Fatalf("expected csrf failure body, got %q", parseRes.Body.String())
	}
}

func loadCSRF(parseT *testing.T, parseServer *secureFormsServer) (string, *http.Cookie) {
	parseT.Helper()
	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseReq.Host = "example.com"
	parseRes := httptest.NewRecorder()
	parseServer.handleIndex(parseRes, parseReq)
	parseBody := parseRes.Body.String()
	parseMatch := regexp.MustCompile(`name="csrf_token"[^>]*value="([^"]+)"`).FindStringSubmatch(parseBody)
	if len(parseMatch) != 2 {
		parseT.Fatalf("expected csrf token in body, got %q", parseBody)
	}
	for _, parseCookie := range parseRes.Result().Cookies() {
		if parseCookie.Name == secureFormsCSRFCookie {
			return parseMatch[1], parseCookie
		}
	}
	parseT.Fatal("expected csrf cookie")
	return "", nil
}
