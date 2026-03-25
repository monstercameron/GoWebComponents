package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	serverauth "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/auth"
	serverdb "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/db"
)

func TestAppRootRedirectsToDashboard(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/app", nil)
	res := httptest.NewRecorder()

	server.routes().ServeHTTP(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("expected %d, got %d", http.StatusFound, res.Code)
	}
	if location := res.Header().Get("Location"); location != "/app/dashboard" {
		t.Fatalf("expected redirect to /app/dashboard, got %q", location)
	}
}

func TestInternalRouteRedirectsToMockSignInWithoutSession(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
	res := httptest.NewRecorder()

	server.routes().ServeHTTP(res, req)

	if res.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, res.Code)
	}
	if location := res.Header().Get("Location"); !strings.Contains(location, "/auth/mock-sign-in?next=%2Fapp%2Fdashboard") {
		t.Fatalf("expected mock sign-in redirect, got %q", location)
	}
}

func TestInternalAPIRequiresMockSignInRecovery(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/app/preferences", nil)
	res := httptest.NewRecorder()

	server.routes().ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "mock_sign_in_required") {
		t.Fatalf("expected mock_sign_in_required payload, got %q", body)
	}
	if !strings.Contains(body, "/auth/mock-sign-in") {
		t.Fatalf("expected mock sign-in recovery url, got %q", body)
	}
}

func TestMockSignInSetsCookieAndRedirects(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	body := strings.NewReader("role=warehouse_supervisor&next=%2Fapp%2Fwarehouses")
	req := httptest.NewRequest(http.MethodPost, "/auth/mock-sign-in", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	server.routes().ServeHTTP(res, req)

	if res.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, res.Code)
	}
	if location := res.Header().Get("Location"); !strings.Contains(location, "/app/warehouses") {
		t.Fatalf("expected internal redirect, got %q", location)
	}
	var found bool
	for _, cookie := range res.Result().Cookies() {
		if cookie.Name == serverauth.MockSessionCookieName && cookie.Value == "warehouse_supervisor" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected mock session cookie in sign-in response")
	}
}

func TestInternalSSRRoutes(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	tests := []struct {
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
		{path: "/app/settings", title: "Atlas Settings"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
			res := httptest.NewRecorder()

			server.routes().ServeHTTP(res, req)

			if res.Code != http.StatusOK {
				t.Fatalf("expected %d, got %d", http.StatusOK, res.Code)
			}
			body := res.Body.String()
			if !strings.Contains(body, tc.title) {
				t.Fatalf("expected body to contain %q", tc.title)
			}
			if !strings.Contains(body, `<div id="app"></div>`) {
				t.Fatalf("expected wasm app shell in response body, got %q", body)
			}
			if !strings.Contains(body, `id="__ATLAS_BOOTSTRAP__"`) {
				t.Fatalf("expected bootstrap script in response body, got %q", body)
			}
			for _, snippet := range []string{
				`<title data-gwc-router-managed="true">`,
				`<meta name="description"`,
				`<link rel="canonical"`,
			} {
				if !strings.Contains(body, snippet) {
					t.Fatalf("expected router-managed metadata snippet %q, got %q", snippet, body)
				}
			}
			if !strings.Contains(body, tc.path) {
				t.Fatalf("expected bootstrap payload to include route path %q", tc.path)
			}
			if !strings.Contains(body, `"description":"`) {
				t.Fatalf("expected bootstrap payload to include route description, got %q", body)
			}
			if !strings.Contains(body, `"canonical":"`+tc.path+`"`) {
				t.Fatalf("expected bootstrap payload to include canonical %q, got %q", tc.path, body)
			}
			if tc.path == "/app/dashboard" && !strings.Contains(body, "/api/app/dashboard") {
				t.Fatalf("expected dashboard startup request to use /api/app/dashboard, got %q", body)
			}
			if tc.path == "/app/settings" && !strings.Contains(body, "/api/app/settings") {
				t.Fatalf("expected settings startup request to use /api/app/settings, got %q", body)
			}
			if tc.path == "/app/inventory/frame-desk" && strings.Contains(body, "/shop?q=frame-desk") {
				t.Fatalf("expected inventory detail to stay in internal workflows, got %q", body)
			}
			if tc.path == "/app/inventory/frame-desk/threshold-history" && !strings.Contains(body, `"overlay":`) {
				t.Fatalf("expected threshold-history route bootstrap to include overlay data, got %q", body)
			}
		})
	}
}

func TestWarehouseItemDirectEntryBootstrapsParentAndChildData(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/app/warehouses/new-jersey-hub/items/frame-desk?status=promise_risk", nil)
	req.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	res := httptest.NewRecorder()

	server.routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "/api/app/warehouses/new-jersey-hub?status=promise_risk") {
		t.Fatalf("expected nested warehouse item bootstrap to include parent warehouse request, got %q", body)
	}
	if !strings.Contains(body, "/api/app/warehouses/new-jersey-hub/items/frame-desk?status=promise_risk") {
		t.Fatalf("expected nested warehouse item bootstrap to include child item request, got %q", body)
	}
}

func TestInventoryThresholdHistoryExternalBootstrapMode(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/app/inventory/frame-desk/threshold-history?atlas_bootstrap=external", nil)
	req.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	res := httptest.NewRecorder()

	server.routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, res.Code)
	}
	if mode := res.Header().Get("X-Atlas-Bootstrap-Mode"); mode != "external" {
		t.Fatalf("expected external bootstrap mode, got %q", mode)
	}
	body := res.Body.String()
	if !strings.Contains(body, `id="__ATLAS_BOOTSTRAP_REF__"`) {
		t.Fatalf("expected bootstrap reference script, got %q", body)
	}
	if strings.Contains(body, `id="__ATLAS_BOOTSTRAP__"`) {
		t.Fatalf("expected inline bootstrap script to be omitted in external mode, got %q", body)
	}
	if !strings.Contains(body, `/__atlas/bootstrap.json?`) {
		t.Fatalf("expected external bootstrap endpoint reference, got %q", body)
	}
}

func TestUnsupportedRouteFallsBackToInlineBootstrapMode(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/app/dashboard?atlas_bootstrap=external", nil)
	req.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	res := httptest.NewRecorder()

	server.routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, res.Code)
	}
	if mode := res.Header().Get("X-Atlas-Bootstrap-Mode"); mode != "inline" {
		t.Fatalf("expected unsupported route to stay inline, got %q", mode)
	}
	if !strings.Contains(res.Body.String(), `id="__ATLAS_BOOTSTRAP__"`) {
		t.Fatalf("expected inline bootstrap script for unsupported route, got %q", res.Body.String())
	}
}

func TestExternalBootstrapEndpointReturnsThresholdHistoryPayload(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	routeQuery := url.QueryEscape("atlas_bootstrap=external")
	req := httptest.NewRequest(http.MethodGet, "/__atlas/bootstrap.json?path=%2Fapp%2Finventory%2Fframe-desk%2Fthreshold-history&route_query="+routeQuery, nil)
	req.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	res := httptest.NewRecorder()

	server.routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, res.Code)
	}
	body := res.Body.String()
	for _, expected := range []string{
		`"path":"/app/inventory/frame-desk/threshold-history"`,
		`"screen":"sku-threshold-history"`,
		`"overlay"`,
		`"/api/app/inventory/frame-desk/threshold-panel"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected external bootstrap payload to contain %q, got %q", expected, body)
		}
	}
}

func TestPublicSSRRoutes(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	tests := []struct {
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

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			res := httptest.NewRecorder()

			server.routes().ServeHTTP(res, req)

			if res.Code != http.StatusOK {
				t.Fatalf("expected %d, got %d", http.StatusOK, res.Code)
			}
			body := res.Body.String()
			if !strings.Contains(body, tc.title) {
				t.Fatalf("expected body to contain %q", tc.title)
			}
			if !strings.Contains(body, `<div id="app"></div>`) {
				t.Fatalf("expected wasm app shell in response body, got %q", body)
			}
			if !strings.Contains(body, `id="__ATLAS_BOOTSTRAP__"`) {
				t.Fatalf("expected bootstrap script in response body, got %q", body)
			}
			for _, snippet := range []string{
				`<title data-gwc-router-managed="true">`,
				`<meta name="description"`,
				`<link rel="canonical"`,
			} {
				if !strings.Contains(body, snippet) {
					t.Fatalf("expected router-managed metadata snippet %q, got %q", snippet, body)
				}
			}
			if !strings.Contains(body, `"description":"`) {
				t.Fatalf("expected bootstrap payload to include route description, got %q", body)
			}
			if !strings.Contains(body, `"canonical":"`+tc.path+`"`) {
				t.Fatalf("expected bootstrap payload to include canonical %q, got %q", tc.path, body)
			}
			if tc.path == "/warehouses" && !strings.Contains(body, "/api/public/warehouses") {
				t.Fatalf("expected warehouses startup request to use /api/public/warehouses, got %q", body)
			}
		})
	}
}

func TestInternalMutationRejectsMissingCSRFTokens(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	body := strings.NewReader("theme=light&locale=fr&density=comfortable&default_warehouse_id=illinois-hub")
	req := httptest.NewRequest(http.MethodPost, "/api/app/preferences", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	req.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	res := httptest.NewRecorder()

	server.routes().ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, res.Code)
	}
	if !strings.Contains(res.Body.String(), "csrf") {
		t.Fatalf("expected csrf error payload, got %q", res.Body.String())
	}
}

func TestPublicMutationValidationReturnsFieldErrors(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	csrfToken, csrfCookie := loadCSRFFromPage(t, server, "/shop/frame-desk")
	form := url.Values{
		"csrf_token": {csrfToken},
		"email":      {"not-an-email"},
		"quantity":   {"0"},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/public/products/frame-desk/quote-requests", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Referer", "http://example.com/shop/frame-desk")
	req.AddCookie(csrfCookie)
	res := httptest.NewRecorder()

	server.routes().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, res.Code)
	}
	body := res.Body.String()
	for _, expected := range []string{"fields", "requester_name", "company_name", "email", "quantity"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected validation response to contain %q, got %q", expected, body)
		}
	}
}

func TestInternalPurchaseOrderAndReceivingMutationsWithCSRFTokens(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	csrfToken, csrfCookie := loadCSRFFromPage(t, server, "/app/purchase-orders/po-1042")

	purchaseOrderForm := url.Values{
		"csrf_token": {csrfToken},
		"status":     {"approved"},
		"note":       {"Validated by Atlas operations."},
	}
	purchaseOrderReq := httptest.NewRequest(http.MethodPost, "/api/app/purchase-orders/po-1042/status", strings.NewReader(purchaseOrderForm.Encode()))
	purchaseOrderReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	purchaseOrderReq.Header.Set("Origin", "http://example.com")
	purchaseOrderReq.Header.Set("Referer", "http://example.com/app/purchase-orders/po-1042")
	purchaseOrderReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	purchaseOrderReq.AddCookie(csrfCookie)
	purchaseOrderRes := httptest.NewRecorder()

	server.routes().ServeHTTP(purchaseOrderRes, purchaseOrderReq)

	if purchaseOrderRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, purchaseOrderRes.Code)
	}
	if location := purchaseOrderRes.Header().Get("Location"); !strings.Contains(location, "purchase-order-updated") {
		t.Fatalf("expected purchase-order-updated redirect notice, got %q", location)
	}
	purchaseOrder, err := server.store.PurchaseOrderByID(context.Background(), "po-1042")
	if err != nil {
		t.Fatalf("load purchase order: %v", err)
	}
	if purchaseOrder.Status != "approved" {
		t.Fatalf("expected purchase order status approved, got %q", purchaseOrder.Status)
	}

	receivingForm := url.Values{
		"csrf_token":          {csrfToken},
		"status":              {"closed"},
		"discrepancy_summary": {"SSR reconcile complete."},
	}
	receivingReq := httptest.NewRequest(http.MethodPost, "/api/app/receiving/rcv-illinois-001/reconcile", strings.NewReader(receivingForm.Encode()))
	receivingReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	receivingReq.Header.Set("Origin", "http://example.com")
	receivingReq.Header.Set("Referer", "http://example.com/app/receiving/rcv-illinois-001")
	receivingReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	receivingReq.AddCookie(csrfCookie)
	receivingRes := httptest.NewRecorder()

	server.routes().ServeHTTP(receivingRes, receivingReq)

	if receivingRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, receivingRes.Code)
	}
	if location := receivingRes.Header().Get("Location"); !strings.Contains(location, "receiving-reconciled") {
		t.Fatalf("expected receiving-reconciled redirect notice, got %q", location)
	}
	receivingDetail, err := server.store.ReceivingDetail(context.Background(), "rcv-illinois-001")
	if err != nil {
		t.Fatalf("load receiving detail: %v", err)
	}
	if receivingDetail.Session.Status != "closed" {
		t.Fatalf("expected receiving status closed, got %q", receivingDetail.Session.Status)
	}
	if receivingDetail.Session.DiscrepancySummary != "SSR reconcile complete." {
		t.Fatalf("expected updated discrepancy summary, got %q", receivingDetail.Session.DiscrepancySummary)
	}
}

func TestInternalProductCRUDWithCSRFTokens(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	csrfToken, csrfCookie := loadCSRFFromPage(t, server, "/app/products")

	createForm := url.Values{
		"csrf_token":      {csrfToken},
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
	createReq := httptest.NewRequest(http.MethodPost, "/api/app/products", strings.NewReader(createForm.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createReq.Header.Set("Origin", "http://example.com")
	createReq.Header.Set("Referer", "http://example.com/app/products")
	createReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	createReq.AddCookie(csrfCookie)
	createRes := httptest.NewRecorder()

	server.routes().ServeHTTP(createRes, createReq)

	if createRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, createRes.Code)
	}
	if location := createRes.Header().Get("Location"); !strings.Contains(location, "product-created") || !strings.Contains(location, "/app/products/task-lamp") {
		t.Fatalf("expected product-created redirect, got %q", location)
	}
	created, err := server.store.ProductAdminBySlug(context.Background(), "task-lamp")
	if err != nil {
		t.Fatalf("load created product: %v", err)
	}
	if created.Title != "Task Lamp" {
		t.Fatalf("expected created product title Task Lamp, got %q", created.Title)
	}
	if created.WarehouseID != "illinois-hub" {
		t.Fatalf("expected created warehouse illinois-hub, got %q", created.WarehouseID)
	}

	updateForm := url.Values{
		"csrf_token":           {csrfToken},
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
	updateReq := httptest.NewRequest(http.MethodPost, "/api/app/products/task-lamp/update", strings.NewReader(updateForm.Encode()))
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateReq.Header.Set("Origin", "http://example.com")
	updateReq.Header.Set("Referer", "http://example.com/app/products/task-lamp")
	updateReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	updateReq.AddCookie(csrfCookie)
	updateRes := httptest.NewRecorder()

	server.routes().ServeHTTP(updateRes, updateReq)

	if updateRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, updateRes.Code)
	}
	if location := updateRes.Header().Get("Location"); !strings.Contains(location, "product-updated") || !strings.Contains(location, "/app/products/task-lamp-pro") {
		t.Fatalf("expected product-updated redirect, got %q", location)
	}
	updated, err := server.store.ProductAdminBySlug(context.Background(), "task-lamp-pro")
	if err != nil {
		t.Fatalf("load updated product: %v", err)
	}
	if updated.Title != "Task Lamp Pro" {
		t.Fatalf("expected updated product title Task Lamp Pro, got %q", updated.Title)
	}
	if updated.WarehouseID != "new-jersey-hub" {
		t.Fatalf("expected moved warehouse new-jersey-hub, got %q", updated.WarehouseID)
	}
	if updated.Available != 2 {
		t.Fatalf("expected available units 2, got %d", updated.Available)
	}

	deleteForm := url.Values{"csrf_token": {csrfToken}}
	deleteReq := httptest.NewRequest(http.MethodPost, "/api/app/products/task-lamp-pro/delete", strings.NewReader(deleteForm.Encode()))
	deleteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deleteReq.Header.Set("Origin", "http://example.com")
	deleteReq.Header.Set("Referer", "http://example.com/app/products/task-lamp-pro")
	deleteReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	deleteReq.AddCookie(csrfCookie)
	deleteRes := httptest.NewRecorder()

	server.routes().ServeHTTP(deleteRes, deleteReq)

	if deleteRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, deleteRes.Code)
	}
	if location := deleteRes.Header().Get("Location"); !strings.Contains(location, "product-deleted") || !strings.Contains(location, "/app/products") {
		t.Fatalf("expected product-deleted redirect, got %q", location)
	}
	if _, err := server.store.ProductAdminBySlug(context.Background(), "task-lamp-pro"); err == nil {
		t.Fatal("expected deleted product lookup to fail")
	}
}

func TestInternalInventoryAndPurchaseOrderCreateWithCSRFTokens(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	csrfToken, csrfCookie := loadCSRFFromPage(t, server, "/app/inventory/frame-desk")

	updateForm := url.Values{
		"csrf_token":    {csrfToken},
		"warehouse_id":  {"new-jersey-hub"},
		"on_hand":       {"10"},
		"reserved":      {"2"},
		"damaged":       {"1"},
		"inbound":       {"6"},
		"reorder_point": {"14"},
		"safety_stock":  {"5"},
		"status":        {"recovery"},
	}
	updateReq := httptest.NewRequest(http.MethodPost, "/api/app/inventory/frame-desk/update", strings.NewReader(updateForm.Encode()))
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateReq.Header.Set("Origin", "http://example.com")
	updateReq.Header.Set("Referer", "http://example.com/app/inventory/frame-desk")
	updateReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	updateReq.AddCookie(csrfCookie)
	updateRes := httptest.NewRecorder()

	server.routes().ServeHTTP(updateRes, updateReq)

	if updateRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, updateRes.Code)
	}
	if location := updateRes.Header().Get("Location"); !strings.Contains(location, "inventory-updated") || !strings.Contains(location, "/app/inventory/frame-desk") {
		t.Fatalf("expected inventory-updated redirect, got %q", location)
	}
	rows, err := server.store.InventoryRowsBySKU(context.Background(), "frame-desk")
	if err != nil {
		t.Fatalf("load inventory rows: %v", err)
	}
	var updated bool
	for _, row := range rows {
		if row.WarehouseID != "new-jersey-hub" {
			continue
		}
		updated = true
		if row.Available != 7 {
			t.Fatalf("expected recomputed available units 7, got %d", row.Available)
		}
		if row.Inbound != 6 {
			t.Fatalf("expected inbound units 6, got %d", row.Inbound)
		}
		if row.Status != "recovery" {
			t.Fatalf("expected recovery status, got %q", row.Status)
		}
		if row.ReorderPoint != 14 {
			t.Fatalf("expected reorder point 14, got %d", row.ReorderPoint)
		}
	}
	if !updated {
		t.Fatal("expected updated new-jersey-hub inventory lane")
	}

	orderForm := url.Values{
		"csrf_token":    {csrfToken},
		"vendor_name":   {"Northline Fabrication"},
		"warehouse_id":  {"new-jersey-hub"},
		"product_sku":   {"frame-desk"},
		"quantity":      {"9"},
		"eta":           {"Wed 08:00"},
		"priority_note": {"Top up east coast lane"},
		"status":        {"submitted"},
	}
	orderReq := httptest.NewRequest(http.MethodPost, "/api/app/purchase-orders", strings.NewReader(orderForm.Encode()))
	orderReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	orderReq.Header.Set("Origin", "http://example.com")
	orderReq.Header.Set("Referer", "http://example.com/app/inventory/frame-desk")
	orderReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	orderReq.AddCookie(csrfCookie)
	orderRes := httptest.NewRecorder()

	server.routes().ServeHTTP(orderRes, orderReq)

	if orderRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, orderRes.Code)
	}
	if location := orderRes.Header().Get("Location"); !strings.Contains(location, "purchase-order-created") || !strings.Contains(location, "/app/purchase-orders/") {
		t.Fatalf("expected purchase-order-created redirect, got %q", location)
	}
	orders, err := server.store.PurchaseOrdersByWarehouse(context.Background(), "new-jersey-hub")
	if err != nil {
		t.Fatalf("load purchase orders by warehouse: %v", err)
	}
	if len(orders) == 0 {
		t.Fatal("expected at least one purchase order for new-jersey-hub")
	}
	if orders[0].PriorityNote != "Top up east coast lane" {
		t.Fatalf("expected newest order priority note to match, got %q", orders[0].PriorityNote)
	}
	rows, err = server.store.InventoryRowsBySKU(context.Background(), "frame-desk")
	if err != nil {
		t.Fatalf("reload inventory rows: %v", err)
	}
	for _, row := range rows {
		if row.WarehouseID == "new-jersey-hub" && row.Inbound != 15 {
			t.Fatalf("expected inbound to increase to 15 after order create, got %d", row.Inbound)
		}
	}
}

func TestInternalBulkModerationAndSavedViewTransfer(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	csrfToken, csrfCookie := loadCSRFFromPage(t, server, "/app/comments")
	comments, err := server.store.Comments(context.Background(), "")
	if err != nil {
		t.Fatalf("load comments: %v", err)
	}
	if len(comments) < 2 {
		t.Fatalf("expected at least 2 comments, got %d", len(comments))
	}
	ids := []string{comments[0].ID, comments[1].ID}

	bulkForm := url.Values{
		"csrf_token": {csrfToken},
		"ids":        {strings.Join(ids, ",")},
		"status":     {"approved"},
		"reason":     {"Bulk review from SSR test."},
	}
	bulkReq := httptest.NewRequest(http.MethodPost, "/api/app/comments/bulk-moderate", strings.NewReader(bulkForm.Encode()))
	bulkReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	bulkReq.Header.Set("Origin", "http://example.com")
	bulkReq.Header.Set("Referer", "http://example.com/app/comments")
	bulkReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	bulkReq.AddCookie(csrfCookie)
	bulkRes := httptest.NewRecorder()

	server.routes().ServeHTTP(bulkRes, bulkReq)

	if bulkRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, bulkRes.Code)
	}
	if location := bulkRes.Header().Get("Location"); !strings.Contains(location, "comments-bulk-moderated") {
		t.Fatalf("expected comments-bulk-moderated redirect notice, got %q", location)
	}
	updatedComments, err := server.store.Comments(context.Background(), "")
	if err != nil {
		t.Fatalf("reload comments: %v", err)
	}
	for _, id := range ids {
		var found bool
		for _, item := range updatedComments {
			if item.ID != id {
				continue
			}
			found = true
			if item.Status != "approved" {
				t.Fatalf("expected comment %q to move into approved status, got %q", id, item.Status)
			}
			if item.ModerationReason != "Bulk review from SSR test." {
				t.Fatalf("expected bulk moderation reason to persist for %q, got %q", id, item.ModerationReason)
			}
		}
		if !found {
			t.Fatalf("expected comment %q to move into approved status", id)
		}
	}

	exportReq := httptest.NewRequest(http.MethodGet, "/api/app/saved-views/export", nil)
	exportReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	exportRes := httptest.NewRecorder()

	server.routes().ServeHTTP(exportRes, exportReq)

	if exportRes.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, exportRes.Code)
	}
	var exportPayload struct {
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
	if err := json.Unmarshal(exportRes.Body.Bytes(), &exportPayload); err != nil {
		t.Fatalf("decode saved-view export payload: %v", err)
	}
	if len(exportPayload.Items) == 0 {
		t.Fatal("expected saved-view export payload to include at least one item")
	}

	importBody := `{"items":[{"name":"SSR imported triage","scope":"inventory","sortKey":"updated","sortDirection":"desc","density":"compact","warehouseId":"illinois-hub","filtersJSON":"{\"status\":\"promise_risk\"}"}]}`
	importForm := url.Values{
		"csrf_token": {csrfToken},
		"views_json": {importBody},
	}
	importReq := httptest.NewRequest(http.MethodPost, "/api/app/saved-views/import", strings.NewReader(importForm.Encode()))
	importReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	importReq.Header.Set("Origin", "http://example.com")
	importReq.Header.Set("Referer", "http://example.com/app/settings")
	importReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	importReq.AddCookie(csrfCookie)
	importRes := httptest.NewRecorder()

	server.routes().ServeHTTP(importRes, importReq)

	if importRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, importRes.Code)
	}
	if location := importRes.Header().Get("Location"); !strings.Contains(location, "saved-views-imported") {
		t.Fatalf("expected saved-views-imported redirect notice, got %q", location)
	}
	savedViews, err := server.store.SavedViewsByOwner(context.Background(), "demo-operator")
	if err != nil {
		t.Fatalf("reload saved views: %v", err)
	}
	var imported bool
	for _, view := range savedViews {
		if view.Name == "SSR imported triage" {
			imported = true
		}
	}
	if !imported {
		t.Fatal("expected imported saved view to be persisted for demo-operator")
	}
}

func TestDirectEntrySSRUsesFreshBootstrapAfterPreferenceSave(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	csrfToken, csrfCookie := loadCSRFFromPage(t, server, "/app/settings")
	saveForm := url.Values{
		"csrf_token":           {csrfToken},
		"theme":                {"light"},
		"locale":               {"ar"},
		"density":              {"comfortable"},
		"default_warehouse_id": {"illinois-hub"},
	}
	saveReq := httptest.NewRequest(http.MethodPost, "/api/app/preferences", strings.NewReader(saveForm.Encode()))
	saveReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	saveReq.Header.Set("Origin", "http://example.com")
	saveReq.Header.Set("Referer", "http://example.com/app/settings")
	saveReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	saveReq.AddCookie(csrfCookie)
	saveRes := httptest.NewRecorder()

	server.routes().ServeHTTP(saveRes, saveReq)

	if saveRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, saveRes.Code)
	}
	if location := saveRes.Header().Get("Location"); !strings.Contains(location, "preferences-saved") {
		t.Fatalf("expected preferences-saved redirect notice, got %q", location)
	}

	directReq := httptest.NewRequest(http.MethodGet, "/app/inventory", nil)
	directReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	directRes := httptest.NewRecorder()

	server.routes().ServeHTTP(directRes, directReq)

	if directRes.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, directRes.Code)
	}
	body := directRes.Body.String()
	for _, expected := range []string{
		`<html lang="ar" class="atlas-theme-light atlas-density-comfortable"`,
		`"locale":"ar"`,
		`"direction":"rtl"`,
		`"defaultWarehouse":"illinois-hub"`,
		`"theme":{"mode":"light"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected direct-entry SSR body to contain %q, got %q", expected, body)
		}
	}
}

func TestSSRBootstrapOmitsDuplicatedRequestPageData(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/app/inventory", nil)
	req.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	res := httptest.NewRecorder()

	server.routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, `"requests":{"page":{"method":"GET","url":"/api/app/inventory","status":200}}`) {
		t.Fatalf("expected bootstrap request metadata without duplicated page payload, got %q", body)
	}
	if strings.Contains(body, `"/api/app/inventory","status":200,"data":{"page":`) {
		t.Fatalf("expected duplicated request page payload to be trimmed from bootstrap, got %q", body)
	}
}

func TestPublicAndInternalJSONAPIsReturnData(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	publicCases := []struct {
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
	for _, tc := range publicCases {
		t.Run("public:"+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			res := httptest.NewRecorder()
			server.routes().ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("expected %d, got %d", http.StatusOK, res.Code)
			}
			if !strings.Contains(strings.ToLower(res.Header().Get("Content-Type")), "application/json") {
				t.Fatalf("expected JSON content type, got %q", res.Header().Get("Content-Type"))
			}
			if !strings.Contains(res.Body.String(), tc.snippet) {
				t.Fatalf("expected %q in payload, got %q", tc.snippet, res.Body.String())
			}
		})
	}

	internalCases := []struct {
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
	for _, tc := range internalCases {
		t.Run("internal:"+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
			res := httptest.NewRecorder()
			server.routes().ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("expected %d, got %d", http.StatusOK, res.Code)
			}
			if !strings.Contains(strings.ToLower(res.Header().Get("Content-Type")), "application/json") {
				t.Fatalf("expected JSON content type, got %q", res.Header().Get("Content-Type"))
			}
			if !strings.Contains(res.Body.String(), tc.snippet) {
				t.Fatalf("expected %q in payload, got %q", tc.snippet, res.Body.String())
			}
		})
	}
}

func TestAdditionalMutationSuccessPaths(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	publicCSRFToken, publicCSRFCookie := loadCSRFFromPage(t, server, "/shop/frame-desk")

	commentForm := url.Values{
		"csrf_token":  {publicCSRFToken},
		"author_name": {"Cam"},
		"reaction":    {"up"},
		"subject":     {"Shipping update"},
		"body":        {"Could you share the warehouse ETA details?"},
	}
	commentReq := httptest.NewRequest(http.MethodPost, "/api/public/products/frame-desk/comments", strings.NewReader(commentForm.Encode()))
	commentReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	commentReq.Header.Set("Origin", "http://example.com")
	commentReq.Header.Set("Referer", "http://example.com/shop/frame-desk")
	commentReq.AddCookie(publicCSRFCookie)
	commentRes := httptest.NewRecorder()
	server.routes().ServeHTTP(commentRes, commentReq)
	if commentRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, commentRes.Code)
	}
	if location := commentRes.Header().Get("Location"); !strings.Contains(location, "comment-submitted") {
		t.Fatalf("expected comment-submitted redirect notice, got %q", location)
	}

	quoteForm := url.Values{
		"csrf_token":     {publicCSRFToken},
		"requester_name": {"Cam"},
		"company_name":   {"Atlas QA"},
		"email":          {"cam@example.com"},
		"quantity":       {"3"},
		"note":           {"Need a quick quote for design review."},
	}
	quoteReq := httptest.NewRequest(http.MethodPost, "/api/public/products/frame-desk/quote-requests", strings.NewReader(quoteForm.Encode()))
	quoteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	quoteReq.Header.Set("Origin", "http://example.com")
	quoteReq.Header.Set("Referer", "http://example.com/shop/frame-desk")
	quoteReq.AddCookie(publicCSRFCookie)
	quoteRes := httptest.NewRecorder()
	server.routes().ServeHTTP(quoteRes, quoteReq)
	if quoteRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, quoteRes.Code)
	}
	if location := quoteRes.Header().Get("Location"); !strings.Contains(location, "quote-request-submitted") {
		t.Fatalf("expected quote-request-submitted redirect notice, got %q", location)
	}

	restockForm := url.Values{
		"csrf_token":             {publicCSRFToken},
		"email":                  {"cam@example.com"},
		"preferred_warehouse_id": {"new-jersey-hub"},
	}
	restockReq := httptest.NewRequest(http.MethodPost, "/api/public/products/frame-desk/restock-requests", strings.NewReader(restockForm.Encode()))
	restockReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	restockReq.Header.Set("Origin", "http://example.com")
	restockReq.Header.Set("Referer", "http://example.com/shop/frame-desk")
	restockReq.AddCookie(publicCSRFCookie)
	restockRes := httptest.NewRecorder()
	server.routes().ServeHTTP(restockRes, restockReq)
	if restockRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, restockRes.Code)
	}
	if location := restockRes.Header().Get("Location"); !strings.Contains(location, "restock-request-submitted") {
		t.Fatalf("expected restock-request-submitted redirect notice, got %q", location)
	}

	internalCSRFToken, internalCSRFCookie := loadCSRFFromPage(t, server, "/app/comments")
	comments, err := server.store.Comments(context.Background(), "")
	if err != nil {
		t.Fatalf("load comments: %v", err)
	}
	if len(comments) == 0 {
		t.Fatal("expected seeded comments")
	}

	moderateForm := url.Values{
		"csrf_token": {internalCSRFToken},
		"status":     {"flagged"},
		"reason":     {"Escalated by QA scenario."},
	}
	moderateReq := httptest.NewRequest(http.MethodPost, "/api/app/comments/"+comments[0].ID+"/moderate", strings.NewReader(moderateForm.Encode()))
	moderateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	moderateReq.Header.Set("Origin", "http://example.com")
	moderateReq.Header.Set("Referer", "http://example.com/app/comments")
	moderateReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	moderateReq.AddCookie(internalCSRFCookie)
	moderateRes := httptest.NewRecorder()
	server.routes().ServeHTTP(moderateRes, moderateReq)
	if moderateRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, moderateRes.Code)
	}
	if location := moderateRes.Header().Get("Location"); !strings.Contains(location, "comment-moderated") {
		t.Fatalf("expected comment-moderated redirect notice, got %q", location)
	}

	thresholdForm := url.Values{
		"csrf_token":    {internalCSRFToken},
		"warehouse_id":  {"new-jersey-hub"},
		"reorder_point": {"13"},
		"safety_stock":  {"4"},
	}
	thresholdReq := httptest.NewRequest(http.MethodPost, "/api/app/inventory/frame-desk/threshold", strings.NewReader(thresholdForm.Encode()))
	thresholdReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	thresholdReq.Header.Set("Origin", "http://example.com")
	thresholdReq.Header.Set("Referer", "http://example.com/app/inventory/frame-desk")
	thresholdReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	thresholdReq.AddCookie(internalCSRFCookie)
	thresholdRes := httptest.NewRecorder()
	server.routes().ServeHTTP(thresholdRes, thresholdReq)
	if thresholdRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, thresholdRes.Code)
	}
	if location := thresholdRes.Header().Get("Location"); !strings.Contains(location, "threshold-updated") {
		t.Fatalf("expected threshold-updated redirect notice, got %q", location)
	}

	savedViewForm := url.Values{
		"csrf_token":     {internalCSRFToken},
		"name":           {"Ops triage"},
		"scope":          {"inventory"},
		"filters_json":   {`{"status":"critical"}`},
		"sort_key":       {"updated"},
		"sort_direction": {"desc"},
		"density":        {"compact"},
		"warehouse_id":   {"illinois-hub"},
	}
	savedViewReq := httptest.NewRequest(http.MethodPost, "/api/app/saved-views", strings.NewReader(savedViewForm.Encode()))
	savedViewReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	savedViewReq.Header.Set("Origin", "http://example.com")
	savedViewReq.Header.Set("Referer", "http://example.com/app/settings")
	savedViewReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	savedViewReq.AddCookie(internalCSRFCookie)
	savedViewRes := httptest.NewRecorder()
	server.routes().ServeHTTP(savedViewRes, savedViewReq)
	if savedViewRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, savedViewRes.Code)
	}
	if location := savedViewRes.Header().Get("Location"); !strings.Contains(location, "saved-view-created") {
		t.Fatalf("expected saved-view-created redirect notice, got %q", location)
	}

	transferForm := url.Values{
		"csrf_token":              {internalCSRFToken},
		"source_warehouse_id":     {"illinois-hub"},
		"destination_warehouse_id": {"new-jersey-hub"},
		"reason":                  {"Rebalance due to east-coast demand."},
		"recommended_by":          {"ops_lead"},
	}
	transferReq := httptest.NewRequest(http.MethodPost, "/api/app/transfers", strings.NewReader(transferForm.Encode()))
	transferReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	transferReq.Header.Set("Origin", "http://example.com")
	transferReq.Header.Set("Referer", "http://example.com/app/transfers")
	transferReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	transferReq.AddCookie(internalCSRFCookie)
	transferRes := httptest.NewRecorder()
	server.routes().ServeHTTP(transferRes, transferReq)
	if transferRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, transferRes.Code)
	}
	if location := transferRes.Header().Get("Location"); !strings.Contains(location, "transfer-created") {
		t.Fatalf("expected transfer-created redirect notice, got %q", location)
	}
}

func TestRecoveryAndBootstrapErrorPaths(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	mockSignInPageReq := httptest.NewRequest(http.MethodGet, "/auth/mock-sign-in?next=%2Fapp%2Fdashboard", nil)
	mockSignInPageRes := httptest.NewRecorder()
	server.routes().ServeHTTP(mockSignInPageRes, mockSignInPageReq)
	if mockSignInPageRes.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, mockSignInPageRes.Code)
	}
	if !strings.Contains(mockSignInPageRes.Body.String(), "Atlas Mock Sign In") {
		t.Fatalf("expected mock sign-in page content, got %q", mockSignInPageRes.Body.String())
	}

	mockSignOutReq := httptest.NewRequest(http.MethodPost, "/auth/mock-sign-out", nil)
	mockSignOutReq.Header.Set("Accept", "application/json")
	mockSignOutRes := httptest.NewRecorder()
	server.routes().ServeHTTP(mockSignOutRes, mockSignOutReq)
	if mockSignOutRes.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, mockSignOutRes.Code)
	}
	if !strings.Contains(mockSignOutRes.Body.String(), `"ok":true`) {
		t.Fatalf("expected mock sign-out JSON payload, got %q", mockSignOutRes.Body.String())
	}

	publicRecoveryReq := httptest.NewRequest(http.MethodGet, "/unknown-public-route", nil)
	publicRecoveryRes := httptest.NewRecorder()
	server.routes().ServeHTTP(publicRecoveryRes, publicRecoveryReq)
	if publicRecoveryRes.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, publicRecoveryRes.Code)
	}
	if !strings.Contains(publicRecoveryRes.Body.String(), "Back to shop") {
		t.Fatalf("expected public recovery content, got %q", publicRecoveryRes.Body.String())
	}

	internalRecoveryNoSessionReq := httptest.NewRequest(http.MethodGet, "/app/unknown-route", nil)
	internalRecoveryNoSessionRes := httptest.NewRecorder()
	server.routes().ServeHTTP(internalRecoveryNoSessionRes, internalRecoveryNoSessionReq)
	if internalRecoveryNoSessionRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, internalRecoveryNoSessionRes.Code)
	}
	if location := internalRecoveryNoSessionRes.Header().Get("Location"); !strings.Contains(location, "/auth/mock-sign-in") {
		t.Fatalf("expected internal recovery without session to redirect to mock sign-in, got %q", location)
	}

	internalRecoveryReq := httptest.NewRequest(http.MethodGet, "/app/unknown-route", nil)
	internalRecoveryReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	internalRecoveryRes := httptest.NewRecorder()
	server.routes().ServeHTTP(internalRecoveryRes, internalRecoveryReq)
	if internalRecoveryRes.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, internalRecoveryRes.Code)
	}
	if !strings.Contains(internalRecoveryRes.Body.String(), "Back to dashboard") {
		t.Fatalf("expected internal recovery content, got %q", internalRecoveryRes.Body.String())
	}

	unsupportedReq := httptest.NewRequest(http.MethodGet, "/__atlas/bootstrap.json?path=%2Fapp%2Fdashboard", nil)
	unsupportedRes := httptest.NewRecorder()
	server.routes().ServeHTTP(unsupportedRes, unsupportedReq)
	if unsupportedRes.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, unsupportedRes.Code)
	}
	if !strings.Contains(unsupportedRes.Body.String(), "bootstrap_route_unsupported") {
		t.Fatalf("expected bootstrap_route_unsupported payload, got %q", unsupportedRes.Body.String())
	}

	invalidQueryReq := httptest.NewRequest(http.MethodGet, "/__atlas/bootstrap.json?path=%2Fapp%2Finventory%2Fframe-desk%2Fthreshold-history&route_query=%25zz", nil)
	invalidQueryRes := httptest.NewRecorder()
	server.routes().ServeHTTP(invalidQueryRes, invalidQueryReq)
	if invalidQueryRes.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, invalidQueryRes.Code)
	}
	if !strings.Contains(invalidQueryRes.Body.String(), "bootstrap_query_invalid") {
		t.Fatalf("expected bootstrap_query_invalid payload, got %q", invalidQueryRes.Body.String())
	}

	missingSessionReq := httptest.NewRequest(http.MethodGet, "/__atlas/bootstrap.json?path=%2Fapp%2Finventory%2Fframe-desk%2Fthreshold-history", nil)
	missingSessionRes := httptest.NewRecorder()
	server.routes().ServeHTTP(missingSessionRes, missingSessionReq)
	if missingSessionRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, missingSessionRes.Code)
	}
	if location := missingSessionRes.Header().Get("Location"); !strings.Contains(location, "/auth/mock-sign-in") {
		t.Fatalf("expected missing session redirect to mock sign-in, got %q", location)
	}

	notFoundReq := httptest.NewRequest(http.MethodGet, "/__atlas/bootstrap.json?path=%2Fapp%2Finventory%2Fnot-a-real-sku%2Fthreshold-history", nil)
	notFoundReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	notFoundRes := httptest.NewRecorder()
	server.routes().ServeHTTP(notFoundRes, notFoundReq)
	if notFoundRes.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, notFoundRes.Code)
	}
	if !strings.Contains(notFoundRes.Body.String(), "Atlas Route Not Found") {
		t.Fatalf("expected not-found recovery page payload, got %q", notFoundRes.Body.String())
	}
}

func loadCSRFFromPage(t *testing.T, server *atlasServer, path string) (string, *http.Cookie) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Host", "example.com")
	req.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	res := httptest.NewRecorder()

	server.routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, res.Code)
	}
	bodyBytes, err := io.ReadAll(res.Result().Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	tokenMatch := regexp.MustCompile(`"csrf":"([^"]+)"`).FindStringSubmatch(string(bodyBytes))
	if len(tokenMatch) != 2 {
		t.Fatalf("expected csrf token in bootstrap payload, got %q", string(bodyBytes))
	}
	token := tokenMatch[1]
	if token == "" {
		t.Fatalf("expected csrf token capture, got %q", string(bodyBytes))
	}
	for _, cookie := range res.Result().Cookies() {
		if cookie.Name == csrfCookieName {
			return token, cookie
		}
	}
	t.Fatal("expected csrf cookie in response")
	return "", nil
}

func newTestAtlasServer(t *testing.T) (*atlasServer, func()) {
	t.Helper()

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.SQLitePath = filepath.Join(t.TempDir(), "atlas-commerce-os-test.db")

	database, err := serverdb.Open(context.Background(), cfg.SQLitePath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := serverdb.Migrate(context.Background(), database, cfg.MigrationsDir, cfg.FallbackSchema); err != nil {
		database.Close()
		t.Fatalf("migrate sqlite: %v", err)
	}
	if err := serverdb.Seed(context.Background(), database); err != nil {
		database.Close()
		t.Fatalf("seed sqlite: %v", err)
	}

	server := newAtlasServer(cfg, serverdb.NewStore(database), serverauth.NewMockSessionManager())
	cleanup := func() {
		_ = database.Close()
	}
	return server, cleanup
}
