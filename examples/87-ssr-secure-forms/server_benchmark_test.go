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

func BenchmarkRenderSecureFormsDocument(parseB *testing.B) {
	parseState := newPageState("token-123", nil)
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseDocument, parseErr := renderDocument(parseState)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		if len(parseDocument) == 0 {
			parseB.Fatal("expected document")
		}
	}
}

func BenchmarkHandleMultipartUploadSuccess(parseB *testing.B) {
	parseServer := newTestServer()
	parseCsrfToken, parseCsrfCookie := loadBenchmarkCSRF(parseB, parseServer)
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseBody := &bytes.Buffer{}
		parseWriter := multipart.NewWriter(parseBody)
		_ = parseWriter.WriteField("csrf_token", parseCsrfToken)
		_ = parseWriter.WriteField("label", "Benchmark board")
		parsePart, parseErr := parseWriter.CreateFormFile("asset", "board.png")
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		if _, parseErr2 := parsePart.Write([]byte("\x89PNG\r\n\x1a\nfakepng")); parseErr2 != nil {
			parseB.Fatal(parseErr2)
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
			parseB.Fatalf("expected redirect, got %d", parseRes.Code)
		}
	}
}

func loadBenchmarkCSRF(parseB *testing.B, parseServer *secureFormsServer) (string, *http.Cookie) {
	parseB.Helper()
	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseReq.Host = "example.com"
	parseRes := httptest.NewRecorder()
	parseServer.handleIndex(parseRes, parseReq)
	parseBody := parseRes.Body.String()
	parseMatch := regexp.MustCompile(`name="csrf_token"[^>]*value="([^"]+)"`).FindStringSubmatch(parseBody)
	if len(parseMatch) != 2 {
		parseB.Fatalf("expected csrf token in body, got %q", parseBody)
	}
	for _, parseCookie := range parseRes.Result().Cookies() {
		if parseCookie.Name == secureFormsCSRFCookie {
			return parseMatch[1], parseCookie
		}
	}
	parseB.Fatal("expected csrf cookie")
	return "", nil
}
