package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	serverauth "github.com/monstercameron/GoWebComponents/examples/server/atlas-commerce-os/server/auth"
	serverdb "github.com/monstercameron/GoWebComponents/examples/server/atlas-commerce-os/server/db"
)

func TestAppRootRedirectsToDashboard(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseReq := httptest.NewRequest(http.MethodGet, "/app", nil)
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusFound {
		parseT.Fatalf("expected %d, got %d", http.StatusFound, parseRes.Code)
	}
	if parseLocation := parseRes.Header().Get("Location"); parseLocation != "/app/dashboard" {
		parseT.Fatalf("expected redirect to /app/dashboard, got %q", parseLocation)
	}
}

func TestInternalRouteRedirectsToMockSignInWithoutSession(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseReq := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d: %s", http.StatusSeeOther, parseRes.Code, parseRes.Body.String())
	}
	if parseLocation := parseRes.Header().Get("Location"); !strings.Contains(parseLocation, "/auth/mock-sign-in?next=%2Fapp%2Fdashboard") {
		parseT.Fatalf("expected mock sign-in redirect, got %q", parseLocation)
	}
}

func TestInternalAPIRequiresMockSignInRecovery(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseReq := httptest.NewRequest(http.MethodGet, "/api/app/preferences", nil)
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusUnauthorized {
		parseT.Fatalf("expected %d, got %d", http.StatusUnauthorized, parseRes.Code)
	}
	parseBody := parseRes.Body.String()
	if !strings.Contains(parseBody, "mock_sign_in_required") {
		parseT.Fatalf("expected mock_sign_in_required payload, got %q", parseBody)
	}
	if !strings.Contains(parseBody, "/auth/mock-sign-in") {
		parseT.Fatalf("expected mock sign-in recovery url, got %q", parseBody)
	}
}

func TestMockSignInSetsCookieAndRedirects(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseBody := strings.NewReader("role=warehouse_supervisor&next=%2Fapp%2Fwarehouses")
	parseReq := httptest.NewRequest(http.MethodPost, "/auth/mock-sign-in", parseBody)
	parseReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseRes.Code)
	}
	if parseLocation := parseRes.Header().Get("Location"); !strings.Contains(parseLocation, "/app/warehouses") {
		parseT.Fatalf("expected internal redirect, got %q", parseLocation)
	}
	var isFound bool
	for _, parseCookie := range parseRes.Result().Cookies() {
		if parseCookie.Name == serverauth.MockSessionCookieName && parseCookie.Value == "warehouse_supervisor" {
			isFound = true
		}
	}
	if !isFound {
		parseT.Fatal("expected mock session cookie in sign-in response")
	}
}

func TestInternalSSRRoutes(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseTests := []struct {
		path  string
		title string
	}{
		{path: "/app/products", title: "Atlas Product Merchandising"},
		{path: "/app/products/frame-desk", title: "Atlas Product Editor"},
		{path: "/app/inventory", title: "Atlas Inventory"},
		{path: "/app/inventory/frame-desk", title: "Atlas SKU Detail"},
		{path: "/app/inventory/frame-desk/threshold-history", title: "Atlas Threshold History"},
		{path: "/app/warehouses", title: "Atlas Warehouse Operations"},
		{path: "/app/warehouses/new-jersey-hub", title: "Atlas Warehouse Detail"},
		{path: "/app/warehouses/new-jersey-hub/items/frame-desk", title: "Atlas Warehouse Item"},
		{path: "/app/transfers", title: "Atlas Transfers"},
		{path: "/app/transfers/tr-seed-001", title: "Atlas Transfer Detail"},
		{path: "/app/purchase-orders", title: "Atlas Purchase Orders"},
		{path: "/app/purchase-orders/po-1042", title: "Atlas Purchase Order Detail"},
		{path: "/app/receiving", title: "Atlas Receiving"},
		{path: "/app/receiving/rcv-illinois-001", title: "Atlas Receiving Session"},
		{path: "/app/comments", title: "Atlas Buyer Inbox"},
		{path: "/app/comments/moderation/pending", title: "Atlas Buyer Inbox Moderation"},
		{path: "/app/comments/cmt-seed-studio-console-flagged", title: "Atlas Buyer Comment Detail"},
		{path: "/app/settings", title: "Atlas Settings"},
		{path: "/app/settings/appearance", title: "Atlas Settings Appearance"},
		{path: "/app/settings/locale", title: "Atlas Settings Locale"},
		{path: "/app/settings/workspace-defaults", title: "Atlas Settings Workspace Defaults"},
	}

	for _, parseTc := range parseTests {
		parseT.Run(parseTc.path, func(parseT2 *testing.T) {
			parseReq := httptest.NewRequest(http.MethodGet, parseTc.path, nil)
			parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
			parseRes := httptest.NewRecorder()

			parseServer.routes().ServeHTTP(parseRes, parseReq)

			if parseRes.Code != http.StatusOK {
				parseT2.Fatalf("expected %d, got %d", http.StatusOK, parseRes.Code)
			}
			parseBody := parseRes.Body.String()
			if !strings.Contains(parseBody, parseTc.title) {
				parseT2.Fatalf("expected body to contain %q", parseTc.title)
			}
			if !strings.Contains(parseBody, `<div id="app"></div>`) {
				parseT2.Fatalf("expected wasm app shell in response body, got %q", parseBody)
			}
			if !strings.Contains(parseBody, `id="__ATLAS_BOOTSTRAP__"`) {
				parseT2.Fatalf("expected bootstrap script in response body, got %q", parseBody)
			}
			for _, parseSnippet := range []string{
				`<title data-gwc-router-managed="true">`,
				`<meta name="description"`,
				`<link rel="canonical"`,
			} {
				if !strings.Contains(parseBody, parseSnippet) {
					parseT2.Fatalf("expected router-managed metadata snippet %q, got %q", parseSnippet, parseBody)
				}
			}
			if !strings.Contains(parseBody, parseTc.path) {
				parseT2.Fatalf("expected bootstrap payload to include route path %q", parseTc.path)
			}
			if !strings.Contains(parseBody, `"description":"`) {
				parseT2.Fatalf("expected bootstrap payload to include route description, got %q", parseBody)
			}
			if !strings.Contains(parseBody, `"canonical":"`+parseTc.path+`"`) {
				parseT2.Fatalf("expected bootstrap payload to include canonical %q, got %q", parseTc.path, parseBody)
			}
			if parseTc.path == "/app/dashboard" && !strings.Contains(parseBody, "/api/app/dashboard") {
				parseT2.Fatalf("expected dashboard startup request to use /api/app/dashboard, got %q", parseBody)
			}
			if strings.HasPrefix(parseTc.path, "/app/settings") && !strings.Contains(parseBody, "/api/app/settings") {
				parseT2.Fatalf("expected settings startup request to use /api/app/settings, got %q", parseBody)
			}
			if parseTc.path == "/app/comments/moderation/pending" && !strings.Contains(parseBody, "/api/app/comments?status=pending") {
				parseT2.Fatalf("expected comments moderation startup request to include status filter, got %q", parseBody)
			}
			if parseTc.path == "/app/comments/cmt-seed-studio-console-flagged" && !strings.Contains(parseBody, "/api/app/comments") {
				parseT2.Fatalf("expected comment detail startup request to use /api/app/comments, got %q", parseBody)
			}
			if parseTc.path == "/app/inventory/frame-desk" && strings.Contains(parseBody, "/shop?q=frame-desk") {
				parseT2.Fatalf("expected inventory detail to stay in internal workflows, got %q", parseBody)
			}
			if parseTc.path == "/app/inventory/frame-desk/threshold-history" && !strings.Contains(parseBody, `"overlay":`) {
				parseT2.Fatalf("expected threshold-history route bootstrap to include overlay data, got %q", parseBody)
			}
		})
	}
}

func TestWarehouseItemDirectEntryBootstrapsParentAndChildData(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseReq := httptest.NewRequest(http.MethodGet, "/app/warehouses/new-jersey-hub/items/frame-desk?status=promise_risk", nil)
	parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseRes.Code)
	}
	parseBody := parseRes.Body.String()
	if !strings.Contains(parseBody, "/api/app/warehouses") {
		parseT.Fatalf("expected nested warehouse item bootstrap to include parent warehouse list request, got %q", parseBody)
	}
	if !strings.Contains(parseBody, "/api/app/warehouses/new-jersey-hub?status=promise_risk") {
		parseT.Fatalf("expected nested warehouse item bootstrap to include warehouse detail request, got %q", parseBody)
	}
	if !strings.Contains(parseBody, "/api/app/warehouses/new-jersey-hub/items/frame-desk?status=promise_risk") {
		parseT.Fatalf("expected nested warehouse item bootstrap to include child item request, got %q", parseBody)
	}
}

func TestWarehouseDetailDirectEntryBootstrapsParentAndDetailData(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseReq := httptest.NewRequest(http.MethodGet, "/app/warehouses/new-jersey-hub?status=promise_risk", nil)
	parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseRes.Code)
	}
	parseBody := parseRes.Body.String()
	if !strings.Contains(parseBody, "/api/app/warehouses") {
		parseT.Fatalf("expected nested warehouse detail bootstrap to include parent warehouse list request, got %q", parseBody)
	}
	if !strings.Contains(parseBody, "/api/app/warehouses/new-jersey-hub?status=promise_risk") {
		parseT.Fatalf("expected nested warehouse detail bootstrap to include child warehouse detail request, got %q", parseBody)
	}
}

func TestPurchaseOrderDetailDirectEntryBootstrapsParentAndDetailData(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseReq := httptest.NewRequest(http.MethodGet, "/app/purchase-orders/po-1042", nil)
	parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseRes.Code)
	}
	parseBody := parseRes.Body.String()
	if !strings.Contains(parseBody, `"url":"/api/app/purchase-orders"`) {
		parseT.Fatalf("expected nested purchase-order detail bootstrap to include parent order-list request, got %q", parseBody)
	}
	if !strings.Contains(parseBody, "/api/app/purchase-orders/po-1042") {
		parseT.Fatalf("expected nested purchase-order detail bootstrap to include child order-detail request, got %q", parseBody)
	}
}

func TestInventoryThresholdHistoryExternalBootstrapMode(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseReq := httptest.NewRequest(http.MethodGet, "/app/inventory/frame-desk/threshold-history?atlas_bootstrap=external", nil)
	parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseRes.Code)
	}
	if parseMode := parseRes.Header().Get("X-Atlas-Bootstrap-Mode"); parseMode != "external" {
		parseT.Fatalf("expected external bootstrap mode, got %q", parseMode)
	}
	parseBody := parseRes.Body.String()
	if !strings.Contains(parseBody, `id="__ATLAS_BOOTSTRAP_REF__"`) {
		parseT.Fatalf("expected bootstrap reference script, got %q", parseBody)
	}
	if strings.Contains(parseBody, `id="__ATLAS_BOOTSTRAP__"`) {
		parseT.Fatalf("expected inline bootstrap script to be omitted in external mode, got %q", parseBody)
	}
	if !strings.Contains(parseBody, `/__atlas/bootstrap.json?`) {
		parseT.Fatalf("expected external bootstrap endpoint reference, got %q", parseBody)
	}
}

func TestUnsupportedRouteFallsBackToInlineBootstrapMode(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseReq := httptest.NewRequest(http.MethodGet, "/app/dashboard?atlas_bootstrap=external", nil)
	parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseRes.Code)
	}
	if parseMode := parseRes.Header().Get("X-Atlas-Bootstrap-Mode"); parseMode != "inline" {
		parseT.Fatalf("expected unsupported route to stay inline, got %q", parseMode)
	}
	if !strings.Contains(parseRes.Body.String(), `id="__ATLAS_BOOTSTRAP__"`) {
		parseT.Fatalf("expected inline bootstrap script for unsupported route, got %q", parseRes.Body.String())
	}
}

func TestExternalBootstrapEndpointReturnsThresholdHistoryPayload(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseRouteQuery := url.QueryEscape("atlas_bootstrap=external")
	parseReq := httptest.NewRequest(http.MethodGet, "/__atlas/bootstrap.json?path=%2Fapp%2Finventory%2Fframe-desk%2Fthreshold-history&route_query="+parseRouteQuery, nil)
	parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseRes.Code)
	}
	parseBody := parseRes.Body.String()
	for _, parseExpected := range []string{
		`"path":"/app/inventory/frame-desk/threshold-history"`,
		`"screen":"sku-threshold-history"`,
		`"overlay"`,
		`"/api/app/inventory/frame-desk/threshold-panel"`,
	} {
		if !strings.Contains(parseBody, parseExpected) {
			parseT.Fatalf("expected external bootstrap payload to contain %q, got %q", parseExpected, parseBody)
		}
	}
}

func TestPublicSSRRoutes(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseTests := []struct {
		path  string
		title string
	}{
		{path: "/", title: "Atlas Commerce OS"},
		{path: "/shop", title: "Atlas Shop"},
		{path: "/shop/frame-desk", title: "Atlas Frame Desk"},
		{path: "/warehouses", title: "Atlas Delivery Regions"},
		{path: "/warehouses/new-jersey-hub", title: "Atlas Warehouse Detail"},
		{path: "/warehouses/new-jersey-hub/availability/frame-desk", title: "Atlas Warehouse Availability"},
	}

	for _, parseTc := range parseTests {
		parseT.Run(parseTc.path, func(parseT2 *testing.T) {
			parseReq := httptest.NewRequest(http.MethodGet, parseTc.path, nil)
			parseRes := httptest.NewRecorder()

			parseServer.routes().ServeHTTP(parseRes, parseReq)

			if parseRes.Code != http.StatusOK {
				parseT2.Fatalf("expected %d, got %d", http.StatusOK, parseRes.Code)
			}
			parseBody := parseRes.Body.String()
			if !strings.Contains(parseBody, parseTc.title) {
				parseT2.Fatalf("expected body to contain %q", parseTc.title)
			}
			if !strings.Contains(parseBody, `<div id="app"></div>`) {
				parseT2.Fatalf("expected wasm app shell in response body, got %q", parseBody)
			}
			if !strings.Contains(parseBody, `id="__ATLAS_BOOTSTRAP__"`) {
				parseT2.Fatalf("expected bootstrap script in response body, got %q", parseBody)
			}
			for _, parseSnippet := range []string{
				`<title data-gwc-router-managed="true">`,
				`<meta name="description"`,
				`<link rel="canonical"`,
			} {
				if !strings.Contains(parseBody, parseSnippet) {
					parseT2.Fatalf("expected router-managed metadata snippet %q, got %q", parseSnippet, parseBody)
				}
			}
			if !strings.Contains(parseBody, `"description":"`) {
				parseT2.Fatalf("expected bootstrap payload to include route description, got %q", parseBody)
			}
			if !strings.Contains(parseBody, `"canonical":"`+parseTc.path+`"`) {
				parseT2.Fatalf("expected bootstrap payload to include canonical %q, got %q", parseTc.path, parseBody)
			}
			if parseTc.path == "/warehouses" && !strings.Contains(parseBody, "/api/public/warehouses") {
				parseT2.Fatalf("expected warehouses startup request to use /api/public/warehouses, got %q", parseBody)
			}
		})
	}
}

func TestPublicSSRLocaleQueryControlsDocumentAndBootstrap(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseReq := httptest.NewRequest(http.MethodGet, "/shop?locale=ar&q=desk", nil)
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseRes.Code)
	}
	parseBody := parseRes.Body.String()
	for _, parseExpected := range []string{
		`<html lang="ar"`,
		`"locale":"ar"`,
		`"direction":"rtl"`,
		`"locale":["ar"]`,
		`"q":["desk"]`,
	} {
		if !strings.Contains(parseBody, parseExpected) {
			parseT.Fatalf("expected public locale SSR body to contain %q, got %q", parseExpected, parseBody)
		}
	}

	parseFallbackReq := httptest.NewRequest(http.MethodGet, "/shop?locale=zz", nil)
	parseFallbackRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseFallbackRes, parseFallbackReq)
	if parseFallbackRes.Code != http.StatusOK {
		parseT.Fatalf("expected fallback locale route status %d, got %d", http.StatusOK, parseFallbackRes.Code)
	}
	if !strings.Contains(parseFallbackRes.Body.String(), `<html lang="en"`) {
		parseT.Fatalf("expected unsupported public locale to fall back to en, got %q", parseFallbackRes.Body.String())
	}
}

func TestInternalMutationRejectsMissingCSRFTokens(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseBody := strings.NewReader("theme=light&locale=fr&density=comfortable&default_warehouse_id=illinois-hub")
	parseReq := httptest.NewRequest(http.MethodPost, "/api/app/preferences", parseBody)
	parseReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseReq.Header.Set("Origin", "http://example.com")
	parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusForbidden {
		parseT.Fatalf("expected %d, got %d", http.StatusForbidden, parseRes.Code)
	}
	if !strings.Contains(parseRes.Body.String(), "csrf") {
		parseT.Fatalf("expected csrf error payload, got %q", parseRes.Body.String())
	}
}

func TestPublicMutationValidationReturnsFieldErrors(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCsrfToken, parseCsrfCookie := loadCSRFFromPage(parseT, parseServer, "/shop/frame-desk")
	parseForm := url.Values{
		"csrf_token": {parseCsrfToken},
		"email":      {"not-an-email"},
		"quantity":   {"0"},
	}
	parseReq := httptest.NewRequest(http.MethodPost, "/api/public/products/frame-desk/quote-requests", strings.NewReader(parseForm.Encode()))
	parseReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseReq.Header.Set("Origin", "http://example.com")
	parseReq.Header.Set("Referer", "http://example.com/shop/frame-desk")
	parseReq.AddCookie(parseCsrfCookie)
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusBadRequest {
		parseT.Fatalf("expected %d, got %d", http.StatusBadRequest, parseRes.Code)
	}
	parseBody := parseRes.Body.String()
	for _, parseExpected := range []string{"fields", "requester_name", "company_name", "email", "quantity"} {
		if !strings.Contains(parseBody, parseExpected) {
			parseT.Fatalf("expected validation response to contain %q, got %q", parseExpected, parseBody)
		}
	}
}

func TestInternalPurchaseOrderAndReceivingMutationsWithCSRFTokens(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCsrfToken, parseCsrfCookie := loadCSRFFromPage(parseT, parseServer, "/app/purchase-orders/po-1042")

	parsePurchaseOrderForm := url.Values{
		"csrf_token": {parseCsrfToken},
		"status":     {"approved"},
		"note":       {"Validated by Atlas operations."},
	}
	parsePurchaseOrderReq := httptest.NewRequest(http.MethodPost, "/api/app/purchase-orders/po-1042/status", strings.NewReader(parsePurchaseOrderForm.Encode()))
	parsePurchaseOrderReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parsePurchaseOrderReq.Header.Set("Origin", "http://example.com")
	parsePurchaseOrderReq.Header.Set("Referer", "http://example.com/app/purchase-orders/po-1042")
	parsePurchaseOrderReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parsePurchaseOrderReq.AddCookie(parseCsrfCookie)
	parsePurchaseOrderRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parsePurchaseOrderRes, parsePurchaseOrderReq)

	if parsePurchaseOrderRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parsePurchaseOrderRes.Code)
	}
	if parseLocation := parsePurchaseOrderRes.Header().Get("Location"); !strings.Contains(parseLocation, "purchase-order-updated") {
		parseT.Fatalf("expected purchase-order-updated redirect notice, got %q", parseLocation)
	}
	parsePurchaseOrder, parseErr := parseServer.store.PurchaseOrderByID(context.Background(), "po-1042")
	if parseErr != nil {
		parseT.Fatalf("load purchase order: %v", parseErr)
	}
	if parsePurchaseOrder.Status != "approved" {
		parseT.Fatalf("expected purchase order status approved, got %q", parsePurchaseOrder.Status)
	}

	parseReceivingForm := url.Values{
		"csrf_token":          {parseCsrfToken},
		"status":              {"closed"},
		"discrepancy_summary": {"SSR reconcile complete."},
	}
	parseReceivingReq := httptest.NewRequest(http.MethodPost, "/api/app/receiving/rcv-illinois-001/reconcile", strings.NewReader(parseReceivingForm.Encode()))
	parseReceivingReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseReceivingReq.Header.Set("Origin", "http://example.com")
	parseReceivingReq.Header.Set("Referer", "http://example.com/app/receiving/rcv-illinois-001")
	parseReceivingReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseReceivingReq.AddCookie(parseCsrfCookie)
	parseReceivingRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseReceivingRes, parseReceivingReq)

	if parseReceivingRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseReceivingRes.Code)
	}
	if parseLocation2 := parseReceivingRes.Header().Get("Location"); !strings.Contains(parseLocation2, "receiving-reconciled") {
		parseT.Fatalf("expected receiving-reconciled redirect notice, got %q", parseLocation2)
	}
	parseReceivingDetail, parseErr := parseServer.store.ReceivingDetail(context.Background(), "rcv-illinois-001")
	if parseErr != nil {
		parseT.Fatalf("load receiving detail: %v", parseErr)
	}
	if parseReceivingDetail.Session.Status != "closed" {
		parseT.Fatalf("expected receiving status closed, got %q", parseReceivingDetail.Session.Status)
	}
	if parseReceivingDetail.Session.DiscrepancySummary != "SSR reconcile complete." {
		parseT.Fatalf("expected updated discrepancy summary, got %q", parseReceivingDetail.Session.DiscrepancySummary)
	}
}

func TestInternalReceivingAttachmentMutationWithCSRFTokens(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCsrfToken, parseCsrfCookie := loadCSRFFromPage(parseT, parseServer, "/app/receiving/rcv-illinois-001")
	parseBody := &bytes.Buffer{}
	parseWriter := multipart.NewWriter(parseBody)
	if parseErr := parseWriter.WriteField("csrf_token", parseCsrfToken); parseErr != nil {
		parseT.Fatalf("write csrf multipart field: %v", parseErr)
	}
	if parseErr := parseWriter.WriteField("note", "Dock evidence attached for discrepancy review."); parseErr != nil {
		parseT.Fatalf("write note multipart field: %v", parseErr)
	}
	parseAttachment, parseErr := parseWriter.CreateFormFile("attachment", "dock-evidence.txt")
	if parseErr != nil {
		parseT.Fatalf("create attachment field: %v", parseErr)
	}
	if _, parseErr = parseAttachment.Write([]byte("dock-door photo reference")); parseErr != nil {
		parseT.Fatalf("write attachment body: %v", parseErr)
	}
	if parseErr := parseWriter.Close(); parseErr != nil {
		parseT.Fatalf("close multipart writer: %v", parseErr)
	}

	parseReq := httptest.NewRequest(http.MethodPost, "/api/app/receiving/rcv-illinois-001/attachments", parseBody)
	parseReq.Header.Set("Content-Type", parseWriter.FormDataContentType())
	parseReq.Header.Set("Origin", "http://example.com")
	parseReq.Header.Set("Referer", "http://example.com/app/receiving/rcv-illinois-001")
	parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseReq.AddCookie(parseCsrfCookie)
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseRes.Code)
	}
	if parseLocation := parseRes.Header().Get("Location"); !strings.Contains(parseLocation, "receiving-attachment-added") {
		parseT.Fatalf("expected receiving-attachment-added redirect notice, got %q", parseLocation)
	}
}

func TestInternalProductCRUDWithCSRFTokens(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCsrfToken, parseCsrfCookie := loadCSRFFromPage(parseT, parseServer, "/app/products")

	parseCreateForm := url.Values{
		"csrf_token":      {parseCsrfToken},
		"sku":             {"task-lamp"},
		"slug":            {"task-lamp"},
		"title":           {"Task Lamp"},
		"category":        {"lighting"},
		"price_cents":     {"45900"},
		"status":          {"in_stock"},
		"finish":          {"Warm brass"},
		"summary":         {"Focused task lighting for compact workstation layouts."},
		"details":         {"A narrow-footprint task lamp built for workstation clusters and studio benches."},
		"seo_title":       {"Atlas Task Lamp"},
		"seo_description": {"Task lighting with a compact footprint and warehouse-backed fulfillment."},
		"warehouse_id":    {"illinois-hub"},
		"available":       {"11"},
		"inbound":         {"4"},
	}
	parseCreateReq := httptest.NewRequest(http.MethodPost, "/api/app/products", strings.NewReader(parseCreateForm.Encode()))
	parseCreateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseCreateReq.Header.Set("Origin", "http://example.com")
	parseCreateReq.Header.Set("Referer", "http://example.com/app/products")
	parseCreateReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseCreateReq.AddCookie(parseCsrfCookie)
	parseCreateRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseCreateRes, parseCreateReq)

	if parseCreateRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseCreateRes.Code)
	}
	if parseLocation := parseCreateRes.Header().Get("Location"); !strings.Contains(parseLocation, "product-created") || !strings.Contains(parseLocation, "/app/products/task-lamp") {
		parseT.Fatalf("expected product-created redirect, got %q", parseLocation)
	}
	parseCreated, parseErr := parseServer.store.ProductAdminBySlug(context.Background(), "task-lamp")
	if parseErr != nil {
		parseT.Fatalf("load created product: %v", parseErr)
	}
	if parseCreated.Title != "Task Lamp" {
		parseT.Fatalf("expected created product title Task Lamp, got %q", parseCreated.Title)
	}
	if parseCreated.WarehouseID != "illinois-hub" {
		parseT.Fatalf("expected created warehouse illinois-hub, got %q", parseCreated.WarehouseID)
	}

	parseUpdateForm := url.Values{
		"csrf_token":           {parseCsrfToken},
		"sku":                  {"task-lamp"},
		"slug":                 {"task-lamp-pro"},
		"title":                {"Task Lamp Pro"},
		"category":             {"lighting"},
		"price_cents":          {"55900"},
		"status":               {"low_stock"},
		"finish":               {"Brushed bronze"},
		"summary":              {"Premium task lighting for focused studio work."},
		"details":              {"Updated premium lighting copy for the Atlas catalog CMS."},
		"seo_title":            {"Atlas Task Lamp Pro"},
		"seo_description":      {"Premium task lighting with constrained inventory and managed restock cues."},
		"current_warehouse_id": {"illinois-hub"},
		"warehouse_id":         {"new-jersey-hub"},
		"available":            {"2"},
		"inbound":              {"8"},
	}
	parseUpdateReq := httptest.NewRequest(http.MethodPost, "/api/app/products/task-lamp/update", strings.NewReader(parseUpdateForm.Encode()))
	parseUpdateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseUpdateReq.Header.Set("Origin", "http://example.com")
	parseUpdateReq.Header.Set("Referer", "http://example.com/app/products/task-lamp")
	parseUpdateReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseUpdateReq.AddCookie(parseCsrfCookie)
	parseUpdateRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseUpdateRes, parseUpdateReq)

	if parseUpdateRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseUpdateRes.Code)
	}
	if parseLocation2 := parseUpdateRes.Header().Get("Location"); !strings.Contains(parseLocation2, "product-updated") || !strings.Contains(parseLocation2, "/app/products/task-lamp-pro") {
		parseT.Fatalf("expected product-updated redirect, got %q", parseLocation2)
	}
	parseUpdated, parseErr := parseServer.store.ProductAdminBySlug(context.Background(), "task-lamp-pro")
	if parseErr != nil {
		parseT.Fatalf("load updated product: %v", parseErr)
	}
	if parseUpdated.Title != "Task Lamp Pro" {
		parseT.Fatalf("expected updated product title Task Lamp Pro, got %q", parseUpdated.Title)
	}
	if parseUpdated.WarehouseID != "new-jersey-hub" {
		parseT.Fatalf("expected moved warehouse new-jersey-hub, got %q", parseUpdated.WarehouseID)
	}
	if parseUpdated.Available != 2 {
		parseT.Fatalf("expected available units 2, got %d", parseUpdated.Available)
	}

	parseDeleteForm := url.Values{"csrf_token": {parseCsrfToken}}
	parseDeleteReq := httptest.NewRequest(http.MethodPost, "/api/app/products/task-lamp-pro/delete", strings.NewReader(parseDeleteForm.Encode()))
	parseDeleteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseDeleteReq.Header.Set("Origin", "http://example.com")
	parseDeleteReq.Header.Set("Referer", "http://example.com/app/products/task-lamp-pro")
	parseDeleteReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseDeleteReq.AddCookie(parseCsrfCookie)
	parseDeleteRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseDeleteRes, parseDeleteReq)

	if parseDeleteRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseDeleteRes.Code)
	}
	if parseLocation3 := parseDeleteRes.Header().Get("Location"); !strings.Contains(parseLocation3, "product-deleted") || !strings.Contains(parseLocation3, "/app/products") {
		parseT.Fatalf("expected product-deleted redirect, got %q", parseLocation3)
	}
	if _, parseErr2 := parseServer.store.ProductAdminBySlug(context.Background(), "task-lamp-pro"); parseErr2 == nil {
		parseT.Fatal("expected deleted product lookup to fail")
	}
}

func TestInternalInventoryAndPurchaseOrderCreateWithCSRFTokens(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCsrfToken, parseCsrfCookie := loadCSRFFromPage(parseT, parseServer, "/app/inventory/frame-desk")

	parseUpdateForm := url.Values{
		"csrf_token":    {parseCsrfToken},
		"warehouse_id":  {"new-jersey-hub"},
		"on_hand":       {"10"},
		"reserved":      {"2"},
		"damaged":       {"1"},
		"inbound":       {"6"},
		"reorder_point": {"14"},
		"safety_stock":  {"5"},
		"status":        {"recovery"},
	}
	parseUpdateReq := httptest.NewRequest(http.MethodPost, "/api/app/inventory/frame-desk/update", strings.NewReader(parseUpdateForm.Encode()))
	parseUpdateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseUpdateReq.Header.Set("Origin", "http://example.com")
	parseUpdateReq.Header.Set("Referer", "http://example.com/app/inventory/frame-desk")
	parseUpdateReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseUpdateReq.AddCookie(parseCsrfCookie)
	parseUpdateRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseUpdateRes, parseUpdateReq)

	if parseUpdateRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseUpdateRes.Code)
	}
	if parseLocation := parseUpdateRes.Header().Get("Location"); !strings.Contains(parseLocation, "inventory-updated") || !strings.Contains(parseLocation, "/app/inventory/frame-desk") {
		parseT.Fatalf("expected inventory-updated redirect, got %q", parseLocation)
	}
	parseRows, parseErr := parseServer.store.InventoryRowsBySKU(context.Background(), "frame-desk")
	if parseErr != nil {
		parseT.Fatalf("load inventory rows: %v", parseErr)
	}
	var isUpdated bool
	for _, parseRow := range parseRows {
		if parseRow.WarehouseID != "new-jersey-hub" {
			continue
		}
		isUpdated = true
		if parseRow.Available != 7 {
			parseT.Fatalf("expected recomputed available units 7, got %d", parseRow.Available)
		}
		if parseRow.Inbound != 6 {
			parseT.Fatalf("expected inbound units 6, got %d", parseRow.Inbound)
		}
		if parseRow.Status != "recovery" {
			parseT.Fatalf("expected recovery status, got %q", parseRow.Status)
		}
		if parseRow.ReorderPoint != 14 {
			parseT.Fatalf("expected reorder point 14, got %d", parseRow.ReorderPoint)
		}
	}
	if !isUpdated {
		parseT.Fatal("expected updated new-jersey-hub inventory lane")
	}

	parseOrderForm := url.Values{
		"csrf_token":    {parseCsrfToken},
		"vendor_name":   {"Northline Fabrication"},
		"warehouse_id":  {"new-jersey-hub"},
		"product_sku":   {"frame-desk"},
		"quantity":      {"9"},
		"eta":           {"Wed 08:00"},
		"priority_note": {"Top up east coast lane"},
		"status":        {"submitted"},
	}
	parseOrderReq := httptest.NewRequest(http.MethodPost, "/api/app/purchase-orders", strings.NewReader(parseOrderForm.Encode()))
	parseOrderReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseOrderReq.Header.Set("Origin", "http://example.com")
	parseOrderReq.Header.Set("Referer", "http://example.com/app/inventory/frame-desk")
	parseOrderReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseOrderReq.AddCookie(parseCsrfCookie)
	parseOrderRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseOrderRes, parseOrderReq)

	if parseOrderRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseOrderRes.Code)
	}
	if parseLocation2 := parseOrderRes.Header().Get("Location"); !strings.Contains(parseLocation2, "purchase-order-created") || !strings.Contains(parseLocation2, "/app/purchase-orders/") {
		parseT.Fatalf("expected purchase-order-created redirect, got %q", parseLocation2)
	}
	parseOrders, parseErr := parseServer.store.PurchaseOrdersByWarehouse(context.Background(), "new-jersey-hub")
	if parseErr != nil {
		parseT.Fatalf("load purchase orders by warehouse: %v", parseErr)
	}
	if len(parseOrders) == 0 {
		parseT.Fatal("expected at least one purchase order for new-jersey-hub")
	}
	if parseOrders[0].PriorityNote != "Top up east coast lane" {
		parseT.Fatalf("expected newest order priority note to match, got %q", parseOrders[0].PriorityNote)
	}
	parseRows, parseErr = parseServer.store.InventoryRowsBySKU(context.Background(), "frame-desk")
	if parseErr != nil {
		parseT.Fatalf("reload inventory rows: %v", parseErr)
	}
	for _, parseRow2 := range parseRows {
		if parseRow2.WarehouseID == "new-jersey-hub" && parseRow2.Inbound != 15 {
			parseT.Fatalf("expected inbound to increase to 15 after order create, got %d", parseRow2.Inbound)
		}
	}
}

func TestInternalBulkModerationAndSavedViewTransfer(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCsrfToken, parseCsrfCookie := loadCSRFFromPage(parseT, parseServer, "/app/comments")
	parseComments, parseErr := parseServer.store.Comments(context.Background(), "")
	if parseErr != nil {
		parseT.Fatalf("load comments: %v", parseErr)
	}
	if len(parseComments) < 2 {
		parseT.Fatalf("expected at least 2 comments, got %d", len(parseComments))
	}
	parseIds := []string{parseComments[0].ID, parseComments[1].ID}

	parseBulkForm := url.Values{
		"csrf_token": {parseCsrfToken},
		"ids":        {strings.Join(parseIds, ",")},
		"status":     {"approved"},
		"reason":     {"Bulk review from SSR test."},
	}
	parseBulkReq := httptest.NewRequest(http.MethodPost, "/api/app/comments/bulk-moderate", strings.NewReader(parseBulkForm.Encode()))
	parseBulkReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseBulkReq.Header.Set("Origin", "http://example.com")
	parseBulkReq.Header.Set("Referer", "http://example.com/app/comments")
	parseBulkReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseBulkReq.AddCookie(parseCsrfCookie)
	parseBulkRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseBulkRes, parseBulkReq)

	if parseBulkRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseBulkRes.Code)
	}
	if parseLocation := parseBulkRes.Header().Get("Location"); !strings.Contains(parseLocation, "comments-bulk-moderated") {
		parseT.Fatalf("expected comments-bulk-moderated redirect notice, got %q", parseLocation)
	}
	parseUpdatedComments, parseErr := parseServer.store.Comments(context.Background(), "")
	if parseErr != nil {
		parseT.Fatalf("reload comments: %v", parseErr)
	}
	for _, parseId := range parseIds {
		var isFound bool
		for _, parseItem := range parseUpdatedComments {
			if parseItem.ID != parseId {
				continue
			}
			isFound = true
			if parseItem.Status != "approved" {
				parseT.Fatalf("expected comment %q to move into approved status, got %q", parseId, parseItem.Status)
			}
			if parseItem.ModerationReason != "Bulk review from SSR test." {
				parseT.Fatalf("expected bulk moderation reason to persist for %q, got %q", parseId, parseItem.ModerationReason)
			}
		}
		if !isFound {
			parseT.Fatalf("expected comment %q to move into approved status", parseId)
		}
	}

	parseExportReq := httptest.NewRequest(http.MethodGet, "/api/app/saved-views/export", nil)
	parseExportReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseExportRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseExportRes, parseExportReq)

	if parseExportRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseExportRes.Code)
	}
	var parseExportPayload struct {
		Items []struct {
			Name          string `json:"name"`
			Scope         string `json:"scope"`
			SortKey       string `json:"sortKey"`
			SortDirection string `json:"sortDirection"`
			Density       string `json:"density"`
			WarehouseID   string `json:"warehouseId"`
			FiltersJSON   string `json:"filtersJSON"`
		} `json:"items"`
	}
	if parseErr2 := json.Unmarshal(parseExportRes.Body.Bytes(), &parseExportPayload); parseErr2 != nil {
		parseT.Fatalf("decode saved-view export payload: %v", parseErr2)
	}
	if len(parseExportPayload.Items) == 0 {
		parseT.Fatal("expected saved-view export payload to include at least one item")
	}

	parseImportBody := `{"items":[{"name":"SSR imported triage","scope":"inventory","sortKey":"updated","sortDirection":"desc","density":"compact","warehouseId":"illinois-hub","filtersJSON":"{\"status\":\"promise_risk\"}"}]}`
	parseImportForm := url.Values{
		"csrf_token": {parseCsrfToken},
		"views_json": {parseImportBody},
	}
	parseImportReq := httptest.NewRequest(http.MethodPost, "/api/app/saved-views/import", strings.NewReader(parseImportForm.Encode()))
	parseImportReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseImportReq.Header.Set("Origin", "http://example.com")
	parseImportReq.Header.Set("Referer", "http://example.com/app/settings")
	parseImportReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseImportReq.AddCookie(parseCsrfCookie)
	parseImportRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseImportRes, parseImportReq)

	if parseImportRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseImportRes.Code)
	}
	if parseLocation2 := parseImportRes.Header().Get("Location"); !strings.Contains(parseLocation2, "saved-views-imported") {
		parseT.Fatalf("expected saved-views-imported redirect notice, got %q", parseLocation2)
	}
	parseSavedViews, parseErr := parseServer.store.SavedViewsByOwner(context.Background(), "demo-operator")
	if parseErr != nil {
		parseT.Fatalf("reload saved views: %v", parseErr)
	}
	var isImported bool
	for _, parseView := range parseSavedViews {
		if parseView.Name == "SSR imported triage" {
			isImported = true
		}
	}
	if !isImported {
		parseT.Fatal("expected imported saved view to be persisted for demo-operator")
	}
}

func TestDirectEntrySSRUsesFreshBootstrapAfterPreferenceSave(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCsrfToken, parseCsrfCookie := loadCSRFFromPage(parseT, parseServer, "/app/settings")
	parseSaveForm := url.Values{
		"csrf_token":           {parseCsrfToken},
		"theme":                {"light"},
		"locale":               {"ar"},
		"density":              {"comfortable"},
		"default_warehouse_id": {"illinois-hub"},
	}
	parseSaveReq := httptest.NewRequest(http.MethodPost, "/api/app/preferences", strings.NewReader(parseSaveForm.Encode()))
	parseSaveReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseSaveReq.Header.Set("Origin", "http://example.com")
	parseSaveReq.Header.Set("Referer", "http://example.com/app/settings")
	parseSaveReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseSaveReq.AddCookie(parseCsrfCookie)
	parseSaveRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseSaveRes, parseSaveReq)

	if parseSaveRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseSaveRes.Code)
	}
	if parseLocation := parseSaveRes.Header().Get("Location"); !strings.Contains(parseLocation, "preferences-saved") {
		parseT.Fatalf("expected preferences-saved redirect notice, got %q", parseLocation)
	}

	parseDirectReq := httptest.NewRequest(http.MethodGet, "/app/inventory", nil)
	parseDirectReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseDirectRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseDirectRes, parseDirectReq)

	if parseDirectRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseDirectRes.Code)
	}
	parseBody := parseDirectRes.Body.String()
	for _, parseExpected := range []string{
		`<html lang="ar" class="atlas-theme-light atlas-density-comfortable"`,
		`"locale":"ar"`,
		`"direction":"rtl"`,
		`"defaultWarehouse":"illinois-hub"`,
		`"theme":{"mode":"light"`,
	} {
		if !strings.Contains(parseBody, parseExpected) {
			parseT.Fatalf("expected direct-entry SSR body to contain %q, got %q", parseExpected, parseBody)
		}
	}
}

func TestSSRBootstrapOmitsDuplicatedRequestPageData(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseReq := httptest.NewRequest(http.MethodGet, "/app/inventory", nil)
	parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseRes.Code)
	}
	parseBody := parseRes.Body.String()
	if !strings.Contains(parseBody, `"requests":{"page":{"method":"GET","url":"/api/app/inventory","status":200}}`) {
		parseT.Fatalf("expected bootstrap request metadata without duplicated page payload, got %q", parseBody)
	}
	if strings.Contains(parseBody, `"/api/app/inventory","status":200,"data":{"page":`) {
		parseT.Fatalf("expected duplicated request page payload to be trimmed from bootstrap, got %q", parseBody)
	}
}

func TestPublicAndInternalJSONAPIsReturnData(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parsePublicCases := []struct {
		path    string
		snippet string
	}{
		{path: "/healthz", snippet: `"ok":true`},
		{path: "/api/public/catalog?page=1&sort=featured", snippet: `"items"`},
		{path: "/api/public/products/frame-desk", snippet: `"product"`},
		{path: "/api/public/products/frame-desk/comments", snippet: `"items"`},
		{path: "/api/public/products/frame-desk/related-products", snippet: `"items"`},
		{path: "/api/public/warehouses", snippet: `"items"`},
		{path: "/api/public/warehouses/new-jersey-hub", snippet: `"slug":"new-jersey-hub"`},
		{path: "/api/public/warehouses/new-jersey-hub/availability/frame-desk", snippet: `"warehouse":{"id":"new-jersey-hub"`},
	}
	for _, parseTc := range parsePublicCases {
		parseT.Run("public:"+parseTc.path, func(parseT2 *testing.T) {
			parseReq := httptest.NewRequest(http.MethodGet, parseTc.path, nil)
			parseRes := httptest.NewRecorder()
			parseServer.routes().ServeHTTP(parseRes, parseReq)
			if parseRes.Code != http.StatusOK {
				parseT2.Fatalf("expected %d, got %d", http.StatusOK, parseRes.Code)
			}
			if !strings.Contains(strings.ToLower(parseRes.Header().Get("Content-Type")), "application/json") {
				parseT2.Fatalf("expected JSON content type, got %q", parseRes.Header().Get("Content-Type"))
			}
			if !strings.Contains(parseRes.Body.String(), parseTc.snippet) {
				parseT2.Fatalf("expected %q in payload, got %q", parseTc.snippet, parseRes.Body.String())
			}
		})
	}

	parseInternalCases := []struct {
		path    string
		snippet string
	}{
		{path: "/api/app/bootstrap?path=/app/dashboard", snippet: `"bootstrap"`},
		{path: "/api/app/dashboard", snippet: `"summary"`},
		{path: "/api/app/preferences", snippet: `"ownerId":"demo-operator"`},
		{path: "/api/app/settings", snippet: `"summary"`},
		{path: "/api/app/saved-views", snippet: `"items"`},
		{path: "/api/app/comments", snippet: `"items"`},
		{path: "/api/app/products", snippet: `"items"`},
		{path: "/api/app/products/frame-desk", snippet: `"sku":"frame-desk"`},
		{path: "/api/app/inventory", snippet: `"items"`},
		{path: "/api/app/inventory/frame-desk", snippet: `"sku":"frame-desk"`},
		{path: "/api/app/inventory/frame-desk/threshold-panel", snippet: `"recommendations"`},
		{path: "/api/app/inventory/frame-desk/threshold-history", snippet: `"items"`},
		{path: "/api/app/inventory/frame-desk/transfer-recommendations", snippet: `"items"`},
		{path: "/api/app/warehouses", snippet: `"items"`},
		{path: "/api/app/warehouses/new-jersey-hub", snippet: `"warehouse"`},
		{path: "/api/app/warehouses/new-jersey-hub/items/frame-desk", snippet: `"item"`},
		{path: "/api/app/transfers", snippet: `"items"`},
		{path: "/api/app/transfers/tr-seed-001", snippet: `"transfer"`},
		{path: "/api/app/receiving", snippet: `"items"`},
		{path: "/api/app/receiving/rcv-illinois-001", snippet: `"session"`},
		{path: "/api/app/purchase-orders", snippet: `"items"`},
		{path: "/api/app/purchase-orders/po-1042", snippet: `"order"`},
	}
	for _, parseTc2 := range parseInternalCases {
		parseT.Run("internal:"+parseTc2.path, func(parseT3 *testing.T) {
			parseReq2 := httptest.NewRequest(http.MethodGet, parseTc2.path, nil)
			parseReq2.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
			parseRes2 := httptest.NewRecorder()
			parseServer.routes().ServeHTTP(parseRes2, parseReq2)
			if parseRes2.Code != http.StatusOK {
				parseT3.Fatalf("expected %d, got %d", http.StatusOK, parseRes2.Code)
			}
			if !strings.Contains(strings.ToLower(parseRes2.Header().Get("Content-Type")), "application/json") {
				parseT3.Fatalf("expected JSON content type, got %q", parseRes2.Header().Get("Content-Type"))
			}
			if !strings.Contains(parseRes2.Body.String(), parseTc2.snippet) {
				parseT3.Fatalf("expected %q in payload, got %q", parseTc2.snippet, parseRes2.Body.String())
			}
		})
	}
}

func TestAdditionalMutationSuccessPaths(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parsePublicCSRFToken, parsePublicCSRFCookie := loadCSRFFromPage(parseT, parseServer, "/shop/frame-desk")

	parseCommentForm := url.Values{
		"csrf_token":  {parsePublicCSRFToken},
		"author_name": {"Cam"},
		"reaction":    {"up"},
		"subject":     {"Shipping update"},
		"body":        {"Could you share the warehouse ETA details?"},
	}
	parseCommentReq := httptest.NewRequest(http.MethodPost, "/api/public/products/frame-desk/comments", strings.NewReader(parseCommentForm.Encode()))
	parseCommentReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseCommentReq.Header.Set("Origin", "http://example.com")
	parseCommentReq.Header.Set("Referer", "http://example.com/shop/frame-desk")
	parseCommentReq.AddCookie(parsePublicCSRFCookie)
	parseCommentRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseCommentRes, parseCommentReq)
	if parseCommentRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseCommentRes.Code)
	}
	if parseLocation := parseCommentRes.Header().Get("Location"); !strings.Contains(parseLocation, "comment-submitted") {
		parseT.Fatalf("expected comment-submitted redirect notice, got %q", parseLocation)
	}

	parseQuoteForm := url.Values{
		"csrf_token":     {parsePublicCSRFToken},
		"requester_name": {"Cam"},
		"company_name":   {"Atlas QA"},
		"email":          {"cam@example.com"},
		"quantity":       {"3"},
		"note":           {"Need a quick quote for design review."},
	}
	parseQuoteReq := httptest.NewRequest(http.MethodPost, "/api/public/products/frame-desk/quote-requests", strings.NewReader(parseQuoteForm.Encode()))
	parseQuoteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseQuoteReq.Header.Set("Origin", "http://example.com")
	parseQuoteReq.Header.Set("Referer", "http://example.com/shop/frame-desk")
	parseQuoteReq.AddCookie(parsePublicCSRFCookie)
	parseQuoteRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseQuoteRes, parseQuoteReq)
	if parseQuoteRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseQuoteRes.Code)
	}
	if parseLocation2 := parseQuoteRes.Header().Get("Location"); !strings.Contains(parseLocation2, "quote-request-submitted") {
		parseT.Fatalf("expected quote-request-submitted redirect notice, got %q", parseLocation2)
	}

	parseRestockForm := url.Values{
		"csrf_token":             {parsePublicCSRFToken},
		"email":                  {"cam@example.com"},
		"preferred_warehouse_id": {"new-jersey-hub"},
	}
	parseRestockReq := httptest.NewRequest(http.MethodPost, "/api/public/products/frame-desk/restock-requests", strings.NewReader(parseRestockForm.Encode()))
	parseRestockReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseRestockReq.Header.Set("Origin", "http://example.com")
	parseRestockReq.Header.Set("Referer", "http://example.com/shop/frame-desk")
	parseRestockReq.AddCookie(parsePublicCSRFCookie)
	parseRestockRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseRestockRes, parseRestockReq)
	if parseRestockRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseRestockRes.Code)
	}
	if parseLocation3 := parseRestockRes.Header().Get("Location"); !strings.Contains(parseLocation3, "restock-request-submitted") {
		parseT.Fatalf("expected restock-request-submitted redirect notice, got %q", parseLocation3)
	}

	parseInternalCSRFToken, parseInternalCSRFCookie := loadCSRFFromPage(parseT, parseServer, "/app/comments")
	parseComments, parseErr := parseServer.store.Comments(context.Background(), "")
	if parseErr != nil {
		parseT.Fatalf("load comments: %v", parseErr)
	}
	if len(parseComments) == 0 {
		parseT.Fatal("expected seeded comments")
	}

	parseModerateForm := url.Values{
		"csrf_token": {parseInternalCSRFToken},
		"status":     {"flagged"},
		"reason":     {"Escalated by QA scenario."},
	}
	parseModerateReq := httptest.NewRequest(http.MethodPost, "/api/app/comments/"+parseComments[0].ID+"/moderate", strings.NewReader(parseModerateForm.Encode()))
	parseModerateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseModerateReq.Header.Set("Origin", "http://example.com")
	parseModerateReq.Header.Set("Referer", "http://example.com/app/comments")
	parseModerateReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseModerateReq.AddCookie(parseInternalCSRFCookie)
	parseModerateRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseModerateRes, parseModerateReq)
	if parseModerateRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseModerateRes.Code)
	}
	if parseLocation4 := parseModerateRes.Header().Get("Location"); !strings.Contains(parseLocation4, "comment-moderated") {
		parseT.Fatalf("expected comment-moderated redirect notice, got %q", parseLocation4)
	}

	parseThresholdForm := url.Values{
		"csrf_token":    {parseInternalCSRFToken},
		"warehouse_id":  {"new-jersey-hub"},
		"reorder_point": {"13"},
		"safety_stock":  {"4"},
	}
	parseThresholdReq := httptest.NewRequest(http.MethodPost, "/api/app/inventory/frame-desk/threshold", strings.NewReader(parseThresholdForm.Encode()))
	parseThresholdReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseThresholdReq.Header.Set("Origin", "http://example.com")
	parseThresholdReq.Header.Set("Referer", "http://example.com/app/inventory/frame-desk")
	parseThresholdReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseThresholdReq.AddCookie(parseInternalCSRFCookie)
	parseThresholdRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseThresholdRes, parseThresholdReq)
	if parseThresholdRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseThresholdRes.Code)
	}
	if parseLocation5 := parseThresholdRes.Header().Get("Location"); !strings.Contains(parseLocation5, "threshold-updated") {
		parseT.Fatalf("expected threshold-updated redirect notice, got %q", parseLocation5)
	}

	parseSavedViewForm := url.Values{
		"csrf_token":     {parseInternalCSRFToken},
		"name":           {"Ops triage"},
		"scope":          {"inventory"},
		"filters_json":   {`{"status":"critical"}`},
		"sort_key":       {"updated"},
		"sort_direction": {"desc"},
		"density":        {"compact"},
		"warehouse_id":   {"illinois-hub"},
	}
	parseSavedViewReq := httptest.NewRequest(http.MethodPost, "/api/app/saved-views", strings.NewReader(parseSavedViewForm.Encode()))
	parseSavedViewReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseSavedViewReq.Header.Set("Origin", "http://example.com")
	parseSavedViewReq.Header.Set("Referer", "http://example.com/app/settings")
	parseSavedViewReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseSavedViewReq.AddCookie(parseInternalCSRFCookie)
	parseSavedViewRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseSavedViewRes, parseSavedViewReq)
	if parseSavedViewRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseSavedViewRes.Code)
	}
	if parseLocation6 := parseSavedViewRes.Header().Get("Location"); !strings.Contains(parseLocation6, "saved-view-created") {
		parseT.Fatalf("expected saved-view-created redirect notice, got %q", parseLocation6)
	}

	parseTransferForm := url.Values{
		"csrf_token":               {parseInternalCSRFToken},
		"source_warehouse_id":      {"illinois-hub"},
		"destination_warehouse_id": {"new-jersey-hub"},
		"reason":                   {"Rebalance due to east-coast demand."},
		"recommended_by":           {"ops_lead"},
	}
	parseTransferReq := httptest.NewRequest(http.MethodPost, "/api/app/transfers", strings.NewReader(parseTransferForm.Encode()))
	parseTransferReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseTransferReq.Header.Set("Origin", "http://example.com")
	parseTransferReq.Header.Set("Referer", "http://example.com/app/transfers")
	parseTransferReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseTransferReq.AddCookie(parseInternalCSRFCookie)
	parseTransferRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseTransferRes, parseTransferReq)
	if parseTransferRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseTransferRes.Code)
	}
	if parseLocation7 := parseTransferRes.Header().Get("Location"); !strings.Contains(parseLocation7, "transfer-created") {
		parseT.Fatalf("expected transfer-created redirect notice, got %q", parseLocation7)
	}
}

func TestRecoveryAndBootstrapErrorPaths(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseMockSignInPageReq := httptest.NewRequest(http.MethodGet, "/auth/mock-sign-in?next=%2Fapp%2Fdashboard", nil)
	parseMockSignInPageRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseMockSignInPageRes, parseMockSignInPageReq)
	if parseMockSignInPageRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseMockSignInPageRes.Code)
	}
	if !strings.Contains(parseMockSignInPageRes.Body.String(), "Atlas Mock Sign In") {
		parseT.Fatalf("expected mock sign-in page content, got %q", parseMockSignInPageRes.Body.String())
	}

	parseMockSignOutReq := httptest.NewRequest(http.MethodPost, "/auth/mock-sign-out", nil)
	parseMockSignOutReq.Header.Set("Accept", "application/json")
	parseMockSignOutRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseMockSignOutRes, parseMockSignOutReq)
	if parseMockSignOutRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseMockSignOutRes.Code)
	}
	if !strings.Contains(parseMockSignOutRes.Body.String(), `"ok":true`) {
		parseT.Fatalf("expected mock sign-out JSON payload, got %q", parseMockSignOutRes.Body.String())
	}

	parsePublicRecoveryReq := httptest.NewRequest(http.MethodGet, "/unknown-public-route", nil)
	parsePublicRecoveryRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parsePublicRecoveryRes, parsePublicRecoveryReq)
	if parsePublicRecoveryRes.Code != http.StatusNotFound {
		parseT.Fatalf("expected %d, got %d", http.StatusNotFound, parsePublicRecoveryRes.Code)
	}
	if !strings.Contains(parsePublicRecoveryRes.Body.String(), "Back to shop") {
		parseT.Fatalf("expected public recovery content, got %q", parsePublicRecoveryRes.Body.String())
	}

	parseInternalRecoveryNoSessionReq := httptest.NewRequest(http.MethodGet, "/app/unknown-route", nil)
	parseInternalRecoveryNoSessionRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseInternalRecoveryNoSessionRes, parseInternalRecoveryNoSessionReq)
	if parseInternalRecoveryNoSessionRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseInternalRecoveryNoSessionRes.Code)
	}
	if parseLocation := parseInternalRecoveryNoSessionRes.Header().Get("Location"); !strings.Contains(parseLocation, "/auth/mock-sign-in") {
		parseT.Fatalf("expected internal recovery without session to redirect to mock sign-in, got %q", parseLocation)
	}

	parseInternalRecoveryReq := httptest.NewRequest(http.MethodGet, "/app/unknown-route", nil)
	parseInternalRecoveryReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseInternalRecoveryRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseInternalRecoveryRes, parseInternalRecoveryReq)
	if parseInternalRecoveryRes.Code != http.StatusNotFound {
		parseT.Fatalf("expected %d, got %d", http.StatusNotFound, parseInternalRecoveryRes.Code)
	}
	if !strings.Contains(parseInternalRecoveryRes.Body.String(), "Back to dashboard") {
		parseT.Fatalf("expected internal recovery content, got %q", parseInternalRecoveryRes.Body.String())
	}

	parseUnsupportedReq := httptest.NewRequest(http.MethodGet, "/__atlas/bootstrap.json?path=%2Fapp%2Fdashboard", nil)
	parseUnsupportedRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseUnsupportedRes, parseUnsupportedReq)
	if parseUnsupportedRes.Code != http.StatusBadRequest {
		parseT.Fatalf("expected %d, got %d", http.StatusBadRequest, parseUnsupportedRes.Code)
	}
	if !strings.Contains(parseUnsupportedRes.Body.String(), "bootstrap_route_unsupported") {
		parseT.Fatalf("expected bootstrap_route_unsupported payload, got %q", parseUnsupportedRes.Body.String())
	}

	parseInvalidQueryReq := httptest.NewRequest(http.MethodGet, "/__atlas/bootstrap.json?path=%2Fapp%2Finventory%2Fframe-desk%2Fthreshold-history&route_query=%25zz", nil)
	parseInvalidQueryRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseInvalidQueryRes, parseInvalidQueryReq)
	if parseInvalidQueryRes.Code != http.StatusBadRequest {
		parseT.Fatalf("expected %d, got %d", http.StatusBadRequest, parseInvalidQueryRes.Code)
	}
	if !strings.Contains(parseInvalidQueryRes.Body.String(), "bootstrap_query_invalid") {
		parseT.Fatalf("expected bootstrap_query_invalid payload, got %q", parseInvalidQueryRes.Body.String())
	}

	parseMissingSessionReq := httptest.NewRequest(http.MethodGet, "/__atlas/bootstrap.json?path=%2Fapp%2Finventory%2Fframe-desk%2Fthreshold-history", nil)
	parseMissingSessionRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseMissingSessionRes, parseMissingSessionReq)
	if parseMissingSessionRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseMissingSessionRes.Code)
	}
	if parseLocation2 := parseMissingSessionRes.Header().Get("Location"); !strings.Contains(parseLocation2, "/auth/mock-sign-in") {
		parseT.Fatalf("expected missing session redirect to mock sign-in, got %q", parseLocation2)
	}

	parseNotFoundReq := httptest.NewRequest(http.MethodGet, "/__atlas/bootstrap.json?path=%2Fapp%2Finventory%2Fnot-a-real-sku%2Fthreshold-history", nil)
	parseNotFoundReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseNotFoundRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseNotFoundRes, parseNotFoundReq)
	if parseNotFoundRes.Code != http.StatusNotFound {
		parseT.Fatalf("expected %d, got %d", http.StatusNotFound, parseNotFoundRes.Code)
	}
	if !strings.Contains(parseNotFoundRes.Body.String(), "Atlas Route Not Found") {
		parseT.Fatalf("expected not-found recovery page payload, got %q", parseNotFoundRes.Body.String())
	}
}

func loadCSRFFromPage(parseT *testing.T, parseServer *atlasServer, parsePath string) (string, *http.Cookie) {
	parseT.Helper()

	parseReq := httptest.NewRequest(http.MethodGet, parsePath, nil)
	parseReq.Header.Set("Host", "example.com")
	parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseRes.Code)
	}
	parseBodyBytes, parseErr := io.ReadAll(parseRes.Result().Body)
	if parseErr != nil {
		parseT.Fatalf("read response body: %v", parseErr)
	}
	parseTokenMatch := regexp.MustCompile(`"csrf":"([^"]+)"`).FindStringSubmatch(string(parseBodyBytes))
	if len(parseTokenMatch) != 2 {
		parseT.Fatalf("expected csrf token in bootstrap payload, got %q", string(parseBodyBytes))
	}
	parseToken := parseTokenMatch[1]
	if parseToken == "" {
		parseT.Fatalf("expected csrf token capture, got %q", string(parseBodyBytes))
	}
	for _, parseCookie := range parseRes.Result().Cookies() {
		if parseCookie.Name == csrfCookieName {
			return parseToken, parseCookie
		}
	}
	parseT.Fatal("expected csrf cookie in response")
	return "", nil
}

func newTestAtlasServer(parseT *testing.T) (*atlasServer, func()) {
	parseT.Helper()

	parseCfg, parseErr := loadConfig()
	if parseErr != nil {
		parseT.Fatalf("load config: %v", parseErr)
	}
	parseCfg.SQLitePath = filepath.Join(parseT.TempDir(), "atlas-commerce-os-test.db")

	parseDatabase, parseErr := serverdb.Open(context.Background(), parseCfg.SQLitePath)
	if parseErr != nil {
		parseT.Fatalf("open sqlite: %v", parseErr)
	}
	if parseErr2 := serverdb.Migrate(context.Background(), parseDatabase, parseCfg.MigrationsDir, parseCfg.FallbackSchema); parseErr2 != nil {
		parseDatabase.Close()
		parseT.Fatalf("migrate sqlite: %v", parseErr2)
	}
	if parseErr3 := serverdb.Seed(context.Background(), parseDatabase); parseErr3 != nil {
		parseDatabase.Close()
		parseT.Fatalf("seed sqlite: %v", parseErr3)
	}

	parseServer := newAtlasServer(parseCfg, serverdb.NewStore(parseDatabase), serverauth.NewMockSessionManager())
	parseCleanup := func() {
		_ = parseDatabase.Close()
	}
	return parseServer, parseCleanup
}
