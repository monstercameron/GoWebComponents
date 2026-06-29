package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func gzipHandlerBody(t *testing.T, parseRec *httptest.ResponseRecorder) string {
	t.Helper()
	parseReader, parseErr := gzip.NewReader(bytes.NewReader(parseRec.Body.Bytes()))
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	defer parseReader.Close()
	parseData, parseErr := io.ReadAll(parseReader)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	return string(parseData)
}

func TestGzipWASMHandlerPassThrough(t *testing.T) {
	parseNextHits := 0
	parseHandler := newGzipWASMHandler(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseNextHits++
		parseW.Header().Set("X-Passthrough", "yes")
		_, _ = parseW.Write([]byte("plain"))
	}))

	for _, parseReq := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/app.js", nil),
		httptest.NewRequest(http.MethodGet, "/app.wasm", nil),
		httptest.NewRequest(http.MethodGet, "/app.wasm", nil),
	} {
		if strings.HasSuffix(parseReq.URL.Path, ".wasm") {
			parseReq.Header.Set("Accept-Encoding", "gzip")
			parseReq.Header.Set("Range", "bytes=0-1")
		}
		parseRec := httptest.NewRecorder()
		parseHandler.ServeHTTP(parseRec, parseReq)
		if parseRec.Header().Get("Content-Encoding") != "" {
			t.Fatalf("pass-through response unexpectedly compressed: headers=%v", parseRec.Header())
		}
		if parseRec.Body.String() != "plain" {
			t.Fatalf("pass-through body = %q, want plain", parseRec.Body.String())
		}
	}
	if parseNextHits != 3 {
		t.Fatalf("next hits = %d, want 3", parseNextHits)
	}
}

func TestGzipWASMHandlerPassesThroughNonOK(t *testing.T) {
	parseHandler := newGzipWASMHandler(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.Header().Set("X-Error", "missing")
		parseW.WriteHeader(http.StatusNotFound)
		_, _ = parseW.Write([]byte("missing wasm"))
	}))
	parseReq := httptest.NewRequest(http.MethodGet, "/app.wasm", nil)
	parseReq.Header.Set("Accept-Encoding", "gzip")
	parseRec := httptest.NewRecorder()

	parseHandler.ServeHTTP(parseRec, parseReq)

	if parseRec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", parseRec.Code)
	}
	if parseRec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("non-OK response unexpectedly compressed: headers=%v", parseRec.Header())
	}
	if parseRec.Header().Get("X-Error") != "missing" || parseRec.Body.String() != "missing wasm" {
		t.Fatalf("non-OK response not passed through: headers=%v body=%q", parseRec.Header(), parseRec.Body.String())
	}
}

func TestGzipWASMHandlerCompressesAndInvalidatesCache(t *testing.T) {
	parseBody := "wasm-v1-" + strings.Repeat("abcdef", 64)
	parseLastModified := "Mon, 01 Jan 2024 00:00:00 GMT"
	parseOriginHits := 0
	parseHandler := newGzipWASMHandler(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseOriginHits++
		parseW.Header().Set("Last-Modified", parseLastModified)
		parseW.Header().Set("Content-Length", strconv.Itoa(len(parseBody)))
		_, _ = parseW.Write([]byte(parseBody))
	}))

	parseCapture := func() *httptest.ResponseRecorder {
		parseReq := httptest.NewRequest(http.MethodGet, "/app.wasm", nil)
		parseReq.Header.Set("Accept-Encoding", "br, gzip")
		parseRec := httptest.NewRecorder()
		parseHandler.ServeHTTP(parseRec, parseReq)
		return parseRec
	}

	parseFirst := parseCapture()
	if parseFirst.Header().Get("Content-Type") != "application/wasm" {
		t.Fatalf("content type = %q, want application/wasm", parseFirst.Header().Get("Content-Type"))
	}
	if parseFirst.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("encoding = %q, want gzip", parseFirst.Header().Get("Content-Encoding"))
	}
	if parseFirst.Header().Get("Last-Modified") != parseLastModified {
		t.Fatalf("last-modified = %q, want %q", parseFirst.Header().Get("Last-Modified"), parseLastModified)
	}
	if parseDecoded := gzipHandlerBody(t, parseFirst); parseDecoded != parseBody {
		t.Fatalf("decoded body = %q, want %q", parseDecoded, parseBody)
	}
	if parseLength := parseFirst.Header().Get("Content-Length"); parseLength != strconv.Itoa(parseFirst.Body.Len()) {
		t.Fatalf("content length = %q, want %d", parseLength, parseFirst.Body.Len())
	}

	parseSecond := parseCapture()
	if !bytes.Equal(parseFirst.Body.Bytes(), parseSecond.Body.Bytes()) {
		t.Fatal("same validator should reuse cached gzip bytes")
	}

	parseBody = "wasm-v2-" + strings.Repeat("ghijkl", 80)
	parseThird := parseCapture()
	if bytes.Equal(parseSecond.Body.Bytes(), parseThird.Body.Bytes()) {
		t.Fatal("changed body length should invalidate cached gzip bytes")
	}
	if parseDecoded := gzipHandlerBody(t, parseThird); parseDecoded != parseBody {
		t.Fatalf("decoded invalidated body = %q, want %q", parseDecoded, parseBody)
	}
	if parseOriginHits != 3 {
		t.Fatalf("origin hits = %d, want 3 probes", parseOriginHits)
	}
}
