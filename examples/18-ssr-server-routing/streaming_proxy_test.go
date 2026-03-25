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

func (r *bufferingProxyRecorder) Header() http.Header {
	return r.header
}

func (r *bufferingProxyRecorder) WriteHeader(statusCode int) {
	if r.status == 0 {
		r.status = statusCode
	}
}

func (r *bufferingProxyRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(data)
}

func (r *bufferingProxyRecorder) Flush() {}

type gzipBufferingProxyRecorder struct {
	header http.Header
	body   bytes.Buffer
	writer *gzip.Writer
	status int
}

func newGzipBufferingProxyRecorder() *gzipBufferingProxyRecorder {
	recorder := &gzipBufferingProxyRecorder{header: make(http.Header)}
	recorder.writer = gzip.NewWriter(&recorder.body)
	recorder.header.Set("Content-Encoding", "gzip")
	return recorder
}

func (r *gzipBufferingProxyRecorder) Header() http.Header {
	return r.header
}

func (r *gzipBufferingProxyRecorder) WriteHeader(statusCode int) {
	if r.status == 0 {
		r.status = statusCode
	}
}

func (r *gzipBufferingProxyRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.writer.Write(data)
}

func (r *gzipBufferingProxyRecorder) Flush() {}

func (r *gzipBufferingProxyRecorder) Close() error {
	return r.writer.Close()
}

func (r *gzipBufferingProxyRecorder) DecompressedBody(t *testing.T) string {
	t.Helper()
	if err := r.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(r.body.Bytes()))
	if err != nil {
		t.Fatalf("new gzip reader: %v", err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read gzip body: %v", err)
	}
	return string(data)
}

func TestStreamedPageCompletesThroughBufferingProxyFixture(t *testing.T) {
	server := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/docs/ssr?tab=loader", nil)
	res := newBufferingProxyRecorder()

	server.handlePage(res, req)
	if res.status != http.StatusOK {
		t.Fatalf("expected OK status, got %d", res.status)
	}
	body := res.body.String()
	checks := []string{
		"Streaming nested docs panel...",
		"Streamed docs insights ready",
		"target.outerHTML=",
		"ssr-server-routing.wasm",
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected buffered proxy body to contain %q, got %q", check, body)
		}
	}
}

func TestStreamedPageCompletesThroughGzipBufferingFixture(t *testing.T) {
	server := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/docs/ssr?tab=loader&stream=nested", nil)
	res := newGzipBufferingProxyRecorder()

	server.handlePage(res, req)
	if res.status != http.StatusOK {
		t.Fatalf("expected OK status, got %d", res.status)
	}
	body := res.DecompressedBody(t)
	checks := []string{
		"Nested streamed layout ready",
		"Outer layout",
		"Nested child panel",
		"target.outerHTML=",
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected gzip-buffered proxy body to contain %q, got %q", check, body)
		}
	}
}
