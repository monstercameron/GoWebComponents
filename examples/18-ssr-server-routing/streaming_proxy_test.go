//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type bufferingProxyRecorder struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func newBufferingProxyRecorder() *bufferingProxyRecorder {
	return &bufferingProxyRecorder{header: make(http.Header)}
}

func (parseR *bufferingProxyRecorder) Header() http.Header {
	return parseR.header
}

func (parseR *bufferingProxyRecorder) WriteHeader(parseStatusCode int) {
	if parseR.status == 0 {
		parseR.status = parseStatusCode
	}
}

func (parseR *bufferingProxyRecorder) Write(parseData []byte) (int, error) {
	if parseR.status == 0 {
		parseR.status = http.StatusOK
	}
	return parseR.body.Write(parseData)
}

func (parseR *bufferingProxyRecorder) Flush() {}

type gzipBufferingProxyRecorder struct {
	header http.Header
	body   bytes.Buffer
	writer *gzip.Writer
	status int
}

func newGzipBufferingProxyRecorder() *gzipBufferingProxyRecorder {
	parseRecorder := &gzipBufferingProxyRecorder{header: make(http.Header)}
	parseRecorder.writer = gzip.NewWriter(&parseRecorder.body)
	parseRecorder.header.Set("Content-Encoding", "gzip")
	return parseRecorder
}

func (parseR *gzipBufferingProxyRecorder) Header() http.Header {
	return parseR.header
}

func (parseR *gzipBufferingProxyRecorder) WriteHeader(parseStatusCode int) {
	if parseR.status == 0 {
		parseR.status = parseStatusCode
	}
}

func (parseR *gzipBufferingProxyRecorder) Write(parseData []byte) (int, error) {
	if parseR.status == 0 {
		parseR.status = http.StatusOK
	}
	return parseR.writer.Write(parseData)
}

func (parseR *gzipBufferingProxyRecorder) Flush() {}

func (parseR *gzipBufferingProxyRecorder) Close() error {
	return parseR.writer.Close()
}

func (parseR *gzipBufferingProxyRecorder) DecompressedBody(parseT *testing.T) string {
	parseT.Helper()
	if parseErr := parseR.Close(); parseErr != nil {
		parseT.Fatalf("close gzip writer: %v", parseErr)
	}
	parseReader, parseErr2 := gzip.NewReader(bytes.NewReader(parseR.body.Bytes()))
	if parseErr2 != nil {
		parseT.Fatalf("new gzip reader: %v", parseErr2)
	}
	defer parseReader.Close()
	parseData, parseErr2 := io.ReadAll(parseReader)
	if parseErr2 != nil {
		parseT.Fatalf("read gzip body: %v", parseErr2)
	}
	return string(parseData)
}

func TestStreamedPageCompletesThroughBufferingProxyFixture(parseT *testing.T) {
	parseServer := testServer(parseT)
	parseReq := httptest.NewRequest(http.MethodGet, "/docs/ssr?tab=loader", nil)
	parseRes := newBufferingProxyRecorder()

	parseServer.handlePage(parseRes, parseReq)
	if parseRes.status != http.StatusOK {
		parseT.Fatalf("expected OK status, got %d", parseRes.status)
	}
	parseBody := parseRes.body.String()
	parseChecks := []string{
		"Streaming nested docs panel...",
		"Streamed docs insights ready",
		"target.outerHTML=",
		"ssr-server-routing.wasm",
	}
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseBody, parseCheck) {
			parseT.Fatalf("expected buffered proxy body to contain %q, got %q", parseCheck, parseBody)
		}
	}
}

func TestStreamedPageCompletesThroughGzipBufferingFixture(parseT *testing.T) {
	parseServer := testServer(parseT)
	parseReq := httptest.NewRequest(http.MethodGet, "/docs/ssr?tab=loader&stream=nested", nil)
	parseRes := newGzipBufferingProxyRecorder()

	parseServer.handlePage(parseRes, parseReq)
	if parseRes.status != http.StatusOK {
		parseT.Fatalf("expected OK status, got %d", parseRes.status)
	}
	parseBody := parseRes.DecompressedBody(parseT)
	parseChecks := []string{
		"Nested streamed layout ready",
		"Outer layout",
		"Nested child panel",
		"target.outerHTML=",
	}
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseBody, parseCheck) {
			parseT.Fatalf("expected gzip-buffered proxy body to contain %q, got %q", parseCheck, parseBody)
		}
	}
}
