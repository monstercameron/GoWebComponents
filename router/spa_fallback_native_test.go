//go:build !js || !wasm

package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSPAFallbackHandlerServesIndexAndAssets(parseT *testing.T) {
	parseDir := parseT.TempDir()
	parseIndex := filepath.Join(parseDir, "index.html")
	if parseErr := os.WriteFile(parseIndex, []byte("INDEX"), 0o644); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseDir, "app.js"), []byte("JS"), 0o644); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseHandler := SPAFallbackHandler(parseIndex, parseDir, "/app/")

	parseCases := []struct {
		name, method, target string
		status               int
		body                 string
	}{
		{"deep link boots index", http.MethodGet, "/app/users/9", 200, "INDEX"},
		{"base root", http.MethodGet, "/app", 200, "INDEX"},
		{"static asset", http.MethodGet, "/app/app.js", 200, "JS"},
		{"missing asset is 404", http.MethodGet, "/app/missing.js", 404, ""},
		{"outside base is 404", http.MethodGet, "/other/x", 404, ""},
		{"traversal stays inside", http.MethodGet, "/app/../../etc/passwd", 200, "INDEX"},
		{"post rejected", http.MethodPost, "/app/x", 405, ""},
	}
	for _, parseCase := range parseCases {
		parseRec := httptest.NewRecorder()
		parseHandler.ServeHTTP(parseRec, httptest.NewRequest(parseCase.method, parseCase.target, nil))
		if parseRec.Code != parseCase.status {
			parseT.Errorf("%s: status = %d, want %d", parseCase.name, parseRec.Code, parseCase.status)
			continue
		}
		if parseCase.body != "" && parseRec.Body.String() != parseCase.body {
			parseT.Errorf("%s: body = %q, want %q", parseCase.name, parseRec.Body.String(), parseCase.body)
		}
	}
}
