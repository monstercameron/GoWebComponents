package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestLLMSHandlerContentNegotiation proves the llms docs are served over HTTP with the
// content type negotiated from Accept (markdown when asked, plain otherwise) — closing the
// FB3 'file-only' gap so agents can fetch docs over HTTP.
func TestLLMSHandlerContentNegotiation(parseT *testing.T) {
	parseServer := httptest.NewServer(LLMSHandler("# Docs\nhello"))
	defer parseServer.Close()

	parseCases := map[string]string{
		"text/markdown": "text/markdown; charset=utf-8",
		"text/plain":    "text/plain; charset=utf-8",
		"":              "text/plain; charset=utf-8",
	}
	for parseAccept, parseWantCT := range parseCases {
		parseReq, _ := http.NewRequest(http.MethodGet, parseServer.URL, nil)
		if parseAccept != "" {
			parseReq.Header.Set("Accept", parseAccept)
		}
		parseResp, parseErr := http.DefaultClient.Do(parseReq)
		if parseErr != nil {
			parseT.Fatalf("GET (Accept=%q): %v", parseAccept, parseErr)
		}
		parseBody, _ := io.ReadAll(parseResp.Body)
		parseResp.Body.Close()
		if parseGot := parseResp.Header.Get("Content-Type"); parseGot != parseWantCT {
			parseT.Fatalf("Accept=%q: Content-Type=%q, want %q", parseAccept, parseGot, parseWantCT)
		}
		if string(parseBody) != "# Docs\nhello" {
			parseT.Fatalf("unexpected body: %q", parseBody)
		}
	}
}
