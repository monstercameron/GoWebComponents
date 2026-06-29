package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// gzipWASMHandler serves .wasm responses gzip-compressed with an in-memory
// cache keyed by path+ETag/Length, so each artifact build pays compression
// once.  Multi-megabyte wasm artifacts compress ~4-5x, which converts a
// network-bound cold start into a compute-bound one on real connections.
type gzipWASMHandler struct {
	parseNext http.Handler
	parseMu   sync.Mutex
	parseKey  map[string]string // request path -> cache validator
	parseBody map[string][]byte // request path -> gzip bytes
}

func newGzipWASMHandler(parseNext http.Handler) http.Handler {
	return &gzipWASMHandler{
		parseNext: parseNext,
		parseKey:  map[string]string{},
		parseBody: map[string][]byte{},
	}
}

// gzipCaptureWriter buffers the downstream response so it can be compressed.
type gzipCaptureWriter struct {
	parseHeader http.Header
	parseStatus int
	parseBuf    bytes.Buffer
}

func (parseW *gzipCaptureWriter) Header() http.Header { return parseW.parseHeader }
func (parseW *gzipCaptureWriter) WriteHeader(parseStatus int) {
	parseW.parseStatus = parseStatus
}
func (parseW *gzipCaptureWriter) Write(parseData []byte) (int, error) {
	if parseW.parseStatus == 0 {
		parseW.parseStatus = http.StatusOK
	}
	return parseW.parseBuf.Write(parseData)
}

func (parseH *gzipWASMHandler) ServeHTTP(parseW http.ResponseWriter, parseR *http.Request) {
	if !strings.HasSuffix(parseR.URL.Path, ".wasm") ||
		!strings.Contains(parseR.Header.Get("Accept-Encoding"), "gzip") ||
		parseR.Header.Get("Range") != "" {
		parseH.parseNext.ServeHTTP(parseW, parseR)
		return
	}

	// Probe freshness cheaply with a captured response; FileServer fills
	// Last-Modified which doubles as the cache validator.
	parseCapture := &gzipCaptureWriter{parseHeader: http.Header{}}
	parseH.parseNext.ServeHTTP(parseCapture, parseR)
	if parseCapture.parseStatus != http.StatusOK {
		// Pass through non-200s (404, 304 revalidations, errors) untouched.
		for parseName, parseValues := range parseCapture.parseHeader {
			for _, parseValue := range parseValues {
				parseW.Header().Add(parseName, parseValue)
			}
		}
		parseW.WriteHeader(parseCapture.parseStatus)
		_, _ = parseW.Write(parseCapture.parseBuf.Bytes())
		return
	}

	parseValidator := parseCapture.parseHeader.Get("Last-Modified") + "/" + fmt.Sprintf("%d", parseCapture.parseBuf.Len())
	parseH.parseMu.Lock()
	parseCompressed, parseHit := parseH.parseBody[parseR.URL.Path]
	if !parseHit || parseH.parseKey[parseR.URL.Path] != parseValidator {
		var parseOut bytes.Buffer
		parseWriter, _ := gzip.NewWriterLevel(&parseOut, gzip.BestSpeed)
		_, _ = parseWriter.Write(parseCapture.parseBuf.Bytes())
		_ = parseWriter.Close()
		parseCompressed = parseOut.Bytes()
		parseH.parseBody[parseR.URL.Path] = parseCompressed
		parseH.parseKey[parseR.URL.Path] = parseValidator
	}
	parseH.parseMu.Unlock()

	parseW.Header().Set("Content-Type", "application/wasm")
	parseW.Header().Set("Content-Encoding", "gzip")
	parseW.Header().Set("Content-Length", fmt.Sprintf("%d", len(parseCompressed)))
	if parseLastModified := parseCapture.parseHeader.Get("Last-Modified"); parseLastModified != "" {
		parseW.Header().Set("Last-Modified", parseLastModified)
	}
	_, _ = parseW.Write(parseCompressed)
}
