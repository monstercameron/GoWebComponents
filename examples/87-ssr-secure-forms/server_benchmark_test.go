//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func BenchmarkRenderSecureFormsDocument(b *testing.B) {
	state := newPageState("token-123", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		document, err := renderDocument(state)
		if err != nil {
			b.Fatal(err)
		}
		if len(document) == 0 {
			b.Fatal("expected document")
		}
	}
}

func BenchmarkHandleMultipartUploadSuccess(b *testing.B) {
	server := newTestServer()
	csrfToken, csrfCookie := loadBenchmarkCSRF(b, server)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("csrf_token", csrfToken)
		_ = writer.WriteField("label", "Benchmark board")
		part, err := writer.CreateFormFile("asset", "board.png")
		if err != nil {
			b.Fatal(err)
		}
		if _, err := part.Write([]byte("\x89PNG\r\n\x1a\nfakepng")); err != nil {
			b.Fatal(err)
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
			b.Fatalf("expected redirect, got %d", res.Code)
		}
	}
}

func loadBenchmarkCSRF(b *testing.B, server *secureFormsServer) (string, *http.Cookie) {
	b.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "example.com"
	res := httptest.NewRecorder()
	server.handleIndex(res, req)
	body := res.Body.String()
	match := regexp.MustCompile(`name="csrf_token"[^>]*value="([^"]+)"`).FindStringSubmatch(body)
	if len(match) != 2 {
		b.Fatalf("expected csrf token in body, got %q", body)
	}
	for _, cookie := range res.Result().Cookies() {
		if cookie.Name == secureFormsCSRFCookie {
			return match[1], cookie
		}
	}
	b.Fatal("expected csrf cookie")
	return "", nil
}
