package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDesktopOriginPolicy tests exact-host and native-endpoint initiation checks.
func TestDesktopOriginPolicy(parseT *testing.T) {
	for _, parseCase := range []struct {
		name, url, origin, referer string
		status                     int
	}{
		{"local document", "http://wails.localhost/", "", "", 200},
		{"local runtime", "http://wails.localhost/wails/runtime", "", "http://wails.localhost/?smoke=1", 200},
		{"local origin", "http://wails.localhost/wails/runtime", "http://wails.localhost", "", 200},
		{"missing initiator", "http://wails.localhost/wails/runtime", "", "", 403},
		{"foreign origin", "http://wails.localhost/wails/runtime", "https://example.invalid", "http://wails.localhost/", 403},
		{"foreign referrer", "http://wails.localhost/wails/runtime", "", "https://example.invalid/", 403},
		{"prefix hostname", "http://wails.localhost.example.invalid/wails/runtime", "", "http://wails.localhost/", 403},
		{"port", "http://wails.localhost:9999/wails/runtime", "", "http://wails.localhost/", 403},
		{"opaque origin", "http://wails.localhost/wails/runtime", "null", "", 403},
		{"credentials", "http://wails.localhost/wails/runtime", "http://name@wails.localhost", "", 403},
	} {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			parseRequest := httptest.NewRequest(http.MethodGet, parseCase.url, nil)
			parseRequest.Header.Set("Origin", parseCase.origin)
			parseRequest.Header.Set("Referer", parseCase.referer)
			parseResponse := httptest.NewRecorder()
			guardDesktopAssets(http.HandlerFunc(func(parseWriter http.ResponseWriter, _ *http.Request) { parseWriter.WriteHeader(http.StatusOK) })).ServeHTTP(parseResponse, parseRequest)
			if parseResponse.Code != parseCase.status {
				parseT.Fatalf("status=%d want=%d", parseResponse.Code, parseCase.status)
			}
			if parseResponse.Header().Get("Content-Security-Policy") != desktopContentPolicy {
				parseT.Fatal("missing enforced content policy")
			}
		})
	}
}
