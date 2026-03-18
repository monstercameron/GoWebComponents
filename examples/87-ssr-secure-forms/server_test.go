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

func TestIndexRendersCSRFAndMultipartForm(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()

	server.handleIndex(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected OK, got %d", res.Code)
	}
	body := res.Body.String()
	checks := []string{
		`name="csrf_token"`,
		`enctype="multipart/form-data"`,
		`Request pricing`,
		`Upload asset`,
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected body to contain %q, got %q", check, body)
		}
	}
	if len(res.Result().Cookies()) == 0 {
		t.Fatal("expected csrf cookie")
	}
}

func TestQuoteValidationRoundTripPreservesValues(t *testing.T) {
	server := newTestServer()
	csrfToken, csrfCookie := loadCSRF(t, server)
	form := strings.NewReader("csrf_token=" + csrfToken + "&name=&email=buyer%40example.com&company=Atlas+Studio&timeline=quarter&notes=Need+review")
	req := httptest.NewRequest(http.MethodPost, "/quote", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	req.Host = "example.com"
	req.AddCookie(csrfCookie)
	res := httptest.NewRecorder()

	server.handleQuote(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected validation error status, got %d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "Name is required.") || !strings.Contains(body, `value="buyer@example.com"`) || !strings.Contains(body, `value="Atlas Studio"`) {
		t.Fatalf("expected validation round-trip body, got %q", body)
	}
}

func TestQuoteSuccessRedirectsAfterSubmit(t *testing.T) {
	server := newTestServer()
	csrfToken, csrfCookie := loadCSRF(t, server)
	form := strings.NewReader("csrf_token=" + csrfToken + "&name=Ada+Buyer&email=buyer%40example.com&company=Atlas+Studio&timeline=30_days&notes=Ship+quote")
	req := httptest.NewRequest(http.MethodPost, "/quote", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	req.Host = "example.com"
	req.AddCookie(csrfCookie)
	res := httptest.NewRecorder()

	server.handleQuote(res, req)
	if res.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", res.Code)
	}
	if location := res.Header().Get("Location"); !strings.Contains(location, "notice=") {
		t.Fatalf("expected redirect notice, got %q", location)
	}
}

func TestUploadValidationRoundTripPreservesLabel(t *testing.T) {
	server := newTestServer()
	csrfToken, csrfCookie := loadCSRF(t, server)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("csrf_token", csrfToken)
	_ = writer.WriteField("label", "Warehouse board")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "http://example.com")
	req.Host = "example.com"
	req.AddCookie(csrfCookie)
	res := httptest.NewRecorder()

	server.handleUpload(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected upload validation error, got %d", res.Code)
	}
	if got := res.Body.String(); !strings.Contains(got, "Choose a PNG or JPEG file.") || !strings.Contains(got, `value="Warehouse board"`) {
		t.Fatalf("expected upload validation round-trip, got %q", got)
	}
}

func TestUploadSuccessRedirectsAfterMultipartSubmit(t *testing.T) {
	server := newTestServer()
	csrfToken, csrfCookie := loadCSRF(t, server)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("csrf_token", csrfToken)
	_ = writer.WriteField("label", "Warehouse board")
	part, err := writer.CreateFormFile("asset", "board.png")
	if err != nil {
		t.Fatalf("unexpected multipart file creation error: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewBufferString("\x89PNG\r\n\x1a\nfakepng")); err != nil {
		t.Fatalf("unexpected multipart write error: %v", err)
	}
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "http://example.com")
	req.Host = "example.com"
	req.AddCookie(csrfCookie)
	res := httptest.NewRecorder()

	server.handleUpload(res, req)
	if res.Code != http.StatusSeeOther {
		t.Fatalf("expected upload redirect, got %d", res.Code)
	}
	location := res.Header().Get("Location")
	if !strings.Contains(location, "asset=board.png") || !strings.Contains(location, "notice=") {
		t.Fatalf("expected asset redirect summary, got %q", location)
	}
}

func TestPostRejectsMissingCSRFTokens(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/quote", strings.NewReader("name=Ada"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	req.Host = "example.com"
	res := httptest.NewRecorder()

	server.handleQuote(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "csrf") {
		t.Fatalf("expected csrf failure body, got %q", res.Body.String())
	}
}

func loadCSRF(t *testing.T, server *secureFormsServer) (string, *http.Cookie) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "example.com"
	res := httptest.NewRecorder()
	server.handleIndex(res, req)
	body := res.Body.String()
	match := regexp.MustCompile(`name="csrf_token"[^>]*value="([^"]+)"`).FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatalf("expected csrf token in body, got %q", body)
	}
	for _, cookie := range res.Result().Cookies() {
		if cookie.Name == secureFormsCSRFCookie {
			return match[1], cookie
		}
	}
	t.Fatal("expected csrf cookie")
	return "", nil
}
