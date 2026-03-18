package main

import (
	"context"
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
		{path: "/app/products", title: "Atlas Product CMS"},
		{path: "/app/products/frame-desk", title: "Atlas Product Editor"},
		{path: "/app/inventory", title: "Atlas Inventory"},
		{path: "/app/inventory/frame-desk", title: "Atlas SKU Detail"},
		{path: "/app/warehouses", title: "Atlas Warehouses Internal"},
		{path: "/app/warehouses/new-jersey-hub", title: "Atlas Warehouse Detail"},
		{path: "/app/transfers", title: "Atlas Transfers"},
		{path: "/app/transfers/tr-seed-001", title: "Atlas Transfer Detail"},
		{path: "/app/purchase-orders", title: "Atlas Purchase Orders"},
		{path: "/app/purchase-orders/po-1042", title: "Atlas Purchase Order Detail"},
		{path: "/app/receiving", title: "Atlas Receiving"},
		{path: "/app/receiving/rcv-illinois-001", title: "Atlas Receiving Session"},
		{path: "/app/comments", title: "Atlas Buyer Follow-Up"},
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
			if !strings.Contains(body, tc.path) {
				t.Fatalf("expected bootstrap payload to include route path %q", tc.path)
			}
			if tc.path == "/app/inventory/frame-desk" && strings.Contains(body, "/shop?q=frame-desk") {
				t.Fatalf("expected inventory detail to stay in internal workflows, got %q", body)
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
