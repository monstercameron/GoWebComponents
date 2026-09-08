package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	serverauth "github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/server/auth"
)

const (
	atlasBootstrapWarnBytes = 140 * 1024
	atlasBootstrapFailBytes = 220 * 1024
	atlasLoaderSmokeBudget  = 2 * time.Second
)

var atlasBootstrapScriptRE = regexp.MustCompile(`(?s)<script[^>]+id="__ATLAS_BOOTSTRAP__"[^>]*>(.*?)</script>`)

func TestAtlasBootstrapPayloadBudgetSmoke(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCases := []struct {
		name     string
		path     string
		internal bool
	}{
		{name: "public landing", path: "/"},
		{name: "public catalog", path: "/shop"},
		{name: "public product detail", path: "/shop/frame-desk"},
		{name: "public warehouse detail", path: "/warehouses/new-jersey-hub"},
		{name: "public availability", path: "/warehouses/new-jersey-hub/availability/frame-desk"},
		{name: "internal dashboard", path: "/app/dashboard", internal: true},
		{name: "internal inventory", path: "/app/inventory", internal: true},
		{name: "internal product detail", path: "/app/products/frame-desk", internal: true},
		{name: "internal warehouse detail", path: "/app/warehouses/new-jersey-hub", internal: true},
		{name: "internal purchase order", path: "/app/purchase-orders/po-1042", internal: true},
		{name: "internal receiving detail", path: "/app/receiving/rcv-illinois-001", internal: true},
	}

	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT2 *testing.T) {
			parseBody, parseElapsed := exerciseAtlasRouteForBudget(parseT2, parseServer, parseCase.path, parseCase.internal)
			parsePayload := extractAtlasBootstrapPayloadForBudget(parseT2, parseBody)
			if len(parsePayload) > atlasBootstrapFailBytes {
				parseT2.Fatalf("bootstrap payload for %s is %d bytes, over fail budget %d", parseCase.path, len(parsePayload), atlasBootstrapFailBytes)
			}
			if len(parsePayload) > atlasBootstrapWarnBytes {
				parseT2.Logf("bootstrap payload for %s is %d bytes, over warning budget %d", parseCase.path, len(parsePayload), atlasBootstrapWarnBytes)
			}
			if parseElapsed > atlasLoaderSmokeBudget {
				parseT2.Fatalf("route %s took %s, over loader smoke budget %s", parseCase.path, parseElapsed, atlasLoaderSmokeBudget)
			}
		})
	}
}

func TestAtlasLoaderLatencySmokeForDenseRoutes(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCases := []struct {
		name string
		path string
	}{
		{name: "dashboard", path: "/api/app/dashboard"},
		{name: "inventory", path: "/api/app/inventory"},
		{name: "product detail", path: "/api/app/products/frame-desk"},
		{name: "warehouse detail", path: "/api/app/warehouses/new-jersey-hub"},
		{name: "purchase order detail", path: "/api/app/purchase-orders/po-1042"},
		{name: "receiving detail", path: "/api/app/receiving/rcv-illinois-001"},
	}

	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT2 *testing.T) {
			parseReq := httptest.NewRequest(http.MethodGet, parseCase.path, nil)
			parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
			parseRes := httptest.NewRecorder()
			parseStarted := time.Now()

			parseServer.routes().ServeHTTP(parseRes, parseReq)

			parseElapsed := time.Since(parseStarted)
			if parseRes.Code != http.StatusOK {
				parseT2.Fatalf("expected %d, got %d: %s", http.StatusOK, parseRes.Code, parseRes.Body.String())
			}
			if parseElapsed > atlasLoaderSmokeBudget {
				parseT2.Fatalf("loader %s took %s, over smoke budget %s", parseCase.path, parseElapsed, atlasLoaderSmokeBudget)
			}
			parseBody := strings.TrimSpace(parseRes.Body.String())
			if parseBody == "" || parseBody == "null" {
				parseT2.Fatalf("loader %s returned empty payload", parseCase.path)
			}
		})
	}
}

func exerciseAtlasRouteForBudget(parseT *testing.T, parseServer *atlasServer, parsePath string, parseInternal bool) (string, time.Duration) {
	parseT.Helper()

	parseReq := httptest.NewRequest(http.MethodGet, parsePath, nil)
	if parseInternal {
		parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	}
	parseRes := httptest.NewRecorder()
	parseStarted := time.Now()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	parseElapsed := time.Since(parseStarted)
	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d: %s", http.StatusOK, parseRes.Code, parseRes.Body.String())
	}
	return parseRes.Body.String(), parseElapsed
}

func extractAtlasBootstrapPayloadForBudget(parseT *testing.T, parseBody string) string {
	parseT.Helper()

	parseMatch := atlasBootstrapScriptRE.FindStringSubmatch(parseBody)
	if len(parseMatch) != 2 {
		parseT.Fatalf("expected inline Atlas bootstrap payload in response")
	}
	parsePayload := strings.TrimSpace(parseMatch[1])
	if parsePayload == "" {
		parseT.Fatalf("expected non-empty Atlas bootstrap payload")
	}
	return parsePayload
}
