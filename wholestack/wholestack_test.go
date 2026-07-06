package wholestack_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/monstercameron/GoWebComponents/v4/serverfn"
	"github.com/monstercameron/GoWebComponents/v4/wholestack"
)

type pingReq struct{}
type pingResp struct {
	Pong string `json:"pong"`
}

func appAssets() fstest.MapFS {
	return fstest.MapFS{
		"index.html":   {Data: []byte("<!doctype html><title>App</title>")},
		"app.wasm":     {Data: []byte("\x00asm fake module")},
		"wasm_exec.js": {Data: []byte("// wasm glue")},
	}
}

// TestWholeStackServesAssetsAndServerFunctions is the FC5 guarantee: a single handler serves
// the embedded app shell, its static assets, AND a //gwc:server function — the whole stack
// from one binary.
func TestWholeStackServesAssetsAndServerFunctions(parseT *testing.T) {
	parseHandler := wholestack.Handler(wholestack.Options{
		Assets: appAssets(),
		RegisterServerFns: func(parseMux *http.ServeMux) {
			serverfn.Handle(parseMux, "Ping", func(parseCtx context.Context, parseReq pingReq) (pingResp, error) {
				return pingResp{Pong: "pong"}, nil
			})
		},
	})
	parseServer := httptest.NewServer(parseHandler)
	defer parseServer.Close()
	serverfn.Configure(parseServer.URL)
	defer serverfn.Configure("")

	// 1. The app shell at "/".
	parseBody, parseCT := get(parseT, parseServer.URL+"/")
	if !contains(parseBody, "<title>App</title>") {
		parseT.Fatalf("root should serve the index shell, got %q", parseBody)
	}
	if parseCT != "text/html; charset=utf-8" {
		parseT.Fatalf("index should be served as html, got %q", parseCT)
	}

	// 2. A static asset.
	if parseBody, _ := get(parseT, parseServer.URL+"/wasm_exec.js"); !contains(parseBody, "wasm glue") {
		parseT.Fatalf("static asset not served, got %q", parseBody)
	}

	// 3. The server function — same binary, same handler.
	parseResp, parseErr := serverfn.Call[pingReq, pingResp](context.Background(), "Ping", pingReq{})
	if parseErr != nil || parseResp.Pong != "pong" {
		parseT.Fatalf("server function over the whole-stack handler failed: %+v err=%v", parseResp, parseErr)
	}
}

// TestSPAFallbackServesShellForClientRoutes proves a deep client-route path falls back to
// the index shell (so client-side routing deep-links work).
func TestSPAFallbackServesShellForClientRoutes(parseT *testing.T) {
	parseHandler := wholestack.Handler(wholestack.Options{Assets: appAssets()})
	parseServer := httptest.NewServer(parseHandler)
	defer parseServer.Close()

	parseBody, _ := get(parseT, parseServer.URL+"/dashboard/settings")
	if !contains(parseBody, "<title>App</title>") {
		parseT.Fatalf("unknown client route should fall back to the shell, got %q", parseBody)
	}
}

// TestHandlerNilAssetsReturns500 pins that a handler built with no Assets serves
// a clean 500 rather than panicking on a nil fs.FS on the first request.
func TestHandlerNilAssetsReturns500(parseT *testing.T) {
	parseHandler := wholestack.Handler(wholestack.Options{})
	parseServer := httptest.NewServer(parseHandler)
	defer parseServer.Close()

	parseResp, parseErr := http.Get(parseServer.URL + "/")
	if parseErr != nil {
		parseT.Fatalf("GET failed: %v", parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusInternalServerError {
		parseT.Fatalf("expected 500 for nil Assets, got %d", parseResp.StatusCode)
	}
}

// TestSPAFallbackDisabled404s proves disabling the fallback yields a 404 for unknown paths.
func TestSPAFallbackDisabled404s(parseT *testing.T) {
	parseHandler := wholestack.Handler(wholestack.Options{Assets: appAssets(), DisableSPAFallback: true})
	parseServer := httptest.NewServer(parseHandler)
	defer parseServer.Close()

	parseResp, parseErr := http.Get(parseServer.URL + "/unknown")
	if parseErr != nil {
		parseT.Fatalf("GET failed: %v", parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusNotFound {
		parseT.Fatalf("expected 404 with fallback disabled, got %d", parseResp.StatusCode)
	}
}

func get(parseT *testing.T, parseURL string) (string, string) {
	parseT.Helper()
	parseResp, parseErr := http.Get(parseURL)
	if parseErr != nil {
		parseT.Fatalf("GET %s: %v", parseURL, parseErr)
	}
	defer parseResp.Body.Close()
	parseBody, _ := io.ReadAll(parseResp.Body)
	return string(parseBody), parseResp.Header.Get("Content-Type")
}

func contains(parseHaystack, parseNeedle string) bool {
	return strings.Contains(parseHaystack, parseNeedle)
}
