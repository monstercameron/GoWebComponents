package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	serverauth "github.com/monstercameron/GoWebComponents/v4/examples/server/atlas-commerce-os/server/auth"
	"github.com/monstercameron/GoWebComponents/v4/examples/server/atlas-commerce-os/shared/repository"
)

// TestAtlasServerClosedDatabaseAPIReadErrors covers API read handlers after the backing store is closed.
func TestAtlasServerClosedDatabaseAPIReadErrors(buildT *testing.T) {
	buildServer, buildCleanup := newTestAtlasServer(buildT)
	buildCleanup()

	buildCases := []struct {
		getPath    string
		getStatus  int
		getSnippet string
		hasSession bool
	}{
		{getPath: "/api/public/catalog", getStatus: http.StatusInternalServerError, getSnippet: "catalog_query_failed"},
		{getPath: "/api/public/products/frame-desk", getStatus: http.StatusNotFound, getSnippet: "product_not_found"},
		{getPath: "/api/public/products/frame-desk/comments", getStatus: http.StatusNotFound, getSnippet: "product_comments_not_found"},
		{getPath: "/api/public/products/frame-desk/related-products", getStatus: http.StatusNotFound, getSnippet: "related_products_not_found"},
		{getPath: "/api/public/warehouses", getStatus: http.StatusInternalServerError, getSnippet: "warehouse_query_failed"},
		{getPath: "/api/public/warehouses/new-jersey-hub", getStatus: http.StatusNotFound, getSnippet: "warehouse_not_found"},
		{getPath: "/api/public/warehouses/new-jersey-hub/availability/frame-desk", getStatus: http.StatusNotFound, getSnippet: "availability_not_found"},
		{getPath: "/api/app/dashboard", getStatus: http.StatusInternalServerError, getSnippet: "dashboard_query_failed", hasSession: true},
		{getPath: "/api/app/preferences", getStatus: http.StatusInternalServerError, getSnippet: "preferences_query_failed", hasSession: true},
		{getPath: "/api/app/settings", getStatus: http.StatusInternalServerError, getSnippet: "settings_query_failed", hasSession: true},
		{getPath: "/api/app/saved-views", getStatus: http.StatusInternalServerError, getSnippet: "saved_views_query_failed", hasSession: true},
		{getPath: "/api/app/saved-views/export", getStatus: http.StatusInternalServerError, getSnippet: "saved_views_export_failed", hasSession: true},
		{getPath: "/api/app/comments", getStatus: http.StatusInternalServerError, getSnippet: "comment_query_failed", hasSession: true},
		{getPath: "/api/app/products", getStatus: http.StatusInternalServerError, getSnippet: "product_admin_query_failed", hasSession: true},
		{getPath: "/api/app/products/frame-desk", getStatus: http.StatusNotFound, getSnippet: "product_admin_not_found", hasSession: true},
		{getPath: "/api/app/inventory", getStatus: http.StatusInternalServerError, getSnippet: "inventory_query_failed", hasSession: true},
		{getPath: "/api/app/inventory/frame-desk", getStatus: http.StatusNotFound, getSnippet: "inventory_item_not_found", hasSession: true},
		{getPath: "/api/app/inventory/frame-desk/threshold-panel", getStatus: http.StatusInternalServerError, getSnippet: "threshold_panel_query_failed", hasSession: true},
		{getPath: "/api/app/inventory/frame-desk/threshold-history", getStatus: http.StatusInternalServerError, getSnippet: "threshold_history_query_failed", hasSession: true},
		{getPath: "/api/app/inventory/frame-desk/transfer-recommendations", getStatus: http.StatusInternalServerError, getSnippet: "transfer_recommendations_query_failed", hasSession: true},
		{getPath: "/api/app/warehouses", getStatus: http.StatusInternalServerError, getSnippet: "warehouse_pressure_query_failed", hasSession: true},
		{getPath: "/api/app/warehouses/new-jersey-hub", getStatus: http.StatusNotFound, getSnippet: "warehouse_detail_not_found", hasSession: true},
		{getPath: "/api/app/warehouses/new-jersey-hub/items/frame-desk", getStatus: http.StatusNotFound, getSnippet: "warehouse_item_not_found", hasSession: true},
		{getPath: "/api/app/transfers", getStatus: http.StatusInternalServerError, getSnippet: "transfer_query_failed", hasSession: true},
		{getPath: "/api/app/transfers/tr-seed-001", getStatus: http.StatusNotFound, getSnippet: "transfer_detail_not_found", hasSession: true},
		{getPath: "/api/app/receiving", getStatus: http.StatusInternalServerError, getSnippet: "receiving_query_failed", hasSession: true},
		{getPath: "/api/app/receiving/rcv-illinois-001", getStatus: http.StatusNotFound, getSnippet: "receiving_detail_not_found", hasSession: true},
		{getPath: "/api/app/purchase-orders", getStatus: http.StatusInternalServerError, getSnippet: "purchase_orders_query_failed", hasSession: true},
		{getPath: "/api/app/purchase-orders/po-1042", getStatus: http.StatusNotFound, getSnippet: "purchase_order_detail_not_found", hasSession: true},
	}

	for _, buildCase := range buildCases {
		buildT.Run(strings.NewReplacer("/", "_", "-", "_").Replace(strings.TrimPrefix(buildCase.getPath, "/")), func(buildT *testing.T) {
			buildReq := httptest.NewRequest(http.MethodGet, buildCase.getPath, nil)
			if buildCase.hasSession {
				buildReq.AddCookie(buildAtlasSessionCookie())
			}
			buildRes := httptest.NewRecorder()

			buildServer.routes().ServeHTTP(buildRes, buildReq)

			if buildRes.Code != buildCase.getStatus {
				buildT.Fatalf("expected %d, got %d for %s", buildCase.getStatus, buildRes.Code, buildCase.getPath)
			}
			if buildBody := buildRes.Body.String(); !strings.Contains(buildBody, buildCase.getSnippet) {
				buildT.Fatalf("expected response for %s to contain %q, got %q", buildCase.getPath, buildCase.getSnippet, buildBody)
			}
		})
	}
}

// TestAtlasServerClosedDatabasePageErrors covers page handlers after the backing store is closed.
func TestAtlasServerClosedDatabasePageErrors(buildT *testing.T) {
	buildServer, buildCleanup := newTestAtlasServer(buildT)
	buildCleanup()

	buildCases := []struct {
		getPath    string
		getStatus  int
		getSnippet string
		hasSession bool
	}{
		{getPath: "/shop", getStatus: http.StatusInternalServerError, getSnippet: "catalog_query_failed"},
		{getPath: "/shop/frame-desk", getStatus: http.StatusNotFound, getSnippet: "Atlas Route Not Found"},
		{getPath: "/warehouses", getStatus: http.StatusInternalServerError, getSnippet: "Atlas Delivery Regions Unavailable"},
		{getPath: "/warehouses/new-jersey-hub", getStatus: http.StatusNotFound, getSnippet: "Atlas Route Not Found"},
		{getPath: "/warehouses/new-jersey-hub/availability/frame-desk", getStatus: http.StatusNotFound, getSnippet: "Atlas Route Not Found"},
		{getPath: "/app/dashboard", getStatus: http.StatusInternalServerError, getSnippet: "dashboard_query_failed", hasSession: true},
		{getPath: "/app/products/frame-desk", getStatus: http.StatusNotFound, getSnippet: "Atlas Internal Route Not Found", hasSession: true},
		{getPath: "/app/inventory", getStatus: http.StatusInternalServerError, getSnippet: "inventory_query_failed", hasSession: true},
		{getPath: "/app/inventory/frame-desk", getStatus: http.StatusNotFound, getSnippet: "Atlas Internal Route Not Found", hasSession: true},
		{getPath: "/app/inventory/frame-desk/threshold-history", getStatus: http.StatusNotFound, getSnippet: "Atlas Internal Route Not Found", hasSession: true},
		{getPath: "/app/warehouses", getStatus: http.StatusInternalServerError, getSnippet: "warehouse_pressure_query_failed", hasSession: true},
		{getPath: "/app/warehouses/new-jersey-hub", getStatus: http.StatusInternalServerError, getSnippet: "warehouse_pressure_query_failed", hasSession: true},
		{getPath: "/app/warehouses/new-jersey-hub/items/frame-desk", getStatus: http.StatusInternalServerError, getSnippet: "warehouse_pressure_query_failed", hasSession: true},
		{getPath: "/app/transfers", getStatus: http.StatusInternalServerError, getSnippet: "transfer_query_failed", hasSession: true},
		{getPath: "/app/transfers/tr-seed-001", getStatus: http.StatusNotFound, getSnippet: "Atlas Internal Route Not Found", hasSession: true},
		{getPath: "/app/purchase-orders", getStatus: http.StatusInternalServerError, getSnippet: "purchase_orders_query_failed", hasSession: true},
		{getPath: "/app/purchase-orders/po-1042", getStatus: http.StatusInternalServerError, getSnippet: "purchase_orders_query_failed", hasSession: true},
		{getPath: "/app/receiving", getStatus: http.StatusInternalServerError, getSnippet: "receiving_query_failed", hasSession: true},
		{getPath: "/app/receiving/rcv-illinois-001", getStatus: http.StatusNotFound, getSnippet: "Atlas Internal Route Not Found", hasSession: true},
		{getPath: "/app/comments", getStatus: http.StatusInternalServerError, getSnippet: "comment_query_failed", hasSession: true},
		{getPath: "/app/comments/moderation/pending", getStatus: http.StatusInternalServerError, getSnippet: "comment_query_failed", hasSession: true},
		{getPath: "/app/comments/cmt-seed-studio-console-flagged", getStatus: http.StatusInternalServerError, getSnippet: "comment_query_failed", hasSession: true},
		{getPath: "/app/settings", getStatus: http.StatusInternalServerError, getSnippet: "settings_query_failed", hasSession: true},
		{getPath: "/app/settings/appearance", getStatus: http.StatusInternalServerError, getSnippet: "settings_query_failed", hasSession: true},
		{getPath: "/app/settings/locale", getStatus: http.StatusInternalServerError, getSnippet: "settings_query_failed", hasSession: true},
		{getPath: "/app/settings/workspace-defaults", getStatus: http.StatusInternalServerError, getSnippet: "settings_query_failed", hasSession: true},
	}

	for _, buildCase := range buildCases {
		buildT.Run(strings.NewReplacer("/", "_", "-", "_").Replace(strings.TrimPrefix(buildCase.getPath, "/")), func(buildT *testing.T) {
			buildReq := httptest.NewRequest(http.MethodGet, buildCase.getPath, nil)
			if buildCase.hasSession {
				buildReq.AddCookie(buildAtlasSessionCookie())
			}
			buildRes := httptest.NewRecorder()

			buildServer.routes().ServeHTTP(buildRes, buildReq)

			if buildRes.Code != buildCase.getStatus {
				buildT.Fatalf("expected %d, got %d for %s", buildCase.getStatus, buildRes.Code, buildCase.getPath)
			}
			if buildBody := buildRes.Body.String(); !strings.Contains(buildBody, buildCase.getSnippet) {
				buildT.Fatalf("expected response for %s to contain %q, got %q", buildCase.getPath, buildCase.getSnippet, buildBody)
			}
		})
	}
}

// TestAtlasServerClosedDatabaseMutationErrors covers mutation handlers after CSRF is established and the store is closed.
func TestAtlasServerClosedDatabaseMutationErrors(buildT *testing.T) {
	buildServer, buildCleanup := newTestAtlasServer(buildT)

	buildPublicToken, buildPublicCookie := loadCSRFFromPage(buildT, buildServer, "/shop/frame-desk")
	buildInternalToken, buildInternalCookie := loadCSRFFromPage(buildT, buildServer, "/app/comments")
	buildCleanup()

	buildImportDocument := buildSavedViewTransferDocument([]repository.SavedView{{
		Name:          "Ops triage",
		Scope:         "inventory",
		SortKey:       "updated",
		SortDirection: "desc",
		Density:       "compact",
		WarehouseID:   "new-jersey-hub",
		FiltersJSON:   `{"status":"critical"}`,
	}})

	buildCases := []struct {
		getMethod       string
		getPath         string
		getForm         url.Values
		getReferer      string
		getStatus       int
		getSnippet      string
		hasSession      bool
		hasPublicCSRF   bool
		hasInternalCSRF bool
	}{
		{
			getMethod:     http.MethodPost,
			getPath:       "/api/public/products/frame-desk/comments",
			getForm:       url.Values{"csrf_token": {buildPublicToken}, "author_name": {"Cam"}, "reaction": {"up"}, "subject": {"Shipping update"}, "body": {"Could you share the warehouse ETA details?"}},
			getReferer:    "/shop/frame-desk",
			getStatus:     http.StatusNotFound,
			getSnippet:    "product_not_found",
			hasPublicCSRF: true,
		},
		{
			getMethod:     http.MethodPost,
			getPath:       "/api/public/products/frame-desk/quote-requests",
			getForm:       url.Values{"csrf_token": {buildPublicToken}, "requester_name": {"Cam"}, "company_name": {"Atlas QA"}, "email": {"cam@example.com"}, "quantity": {"3"}, "note": {"Need a quick quote for design review."}},
			getReferer:    "/shop/frame-desk",
			getStatus:     http.StatusNotFound,
			getSnippet:    "product_not_found",
			hasPublicCSRF: true,
		},
		{
			getMethod:     http.MethodPost,
			getPath:       "/api/public/products/frame-desk/restock-requests",
			getForm:       url.Values{"csrf_token": {buildPublicToken}, "email": {"cam@example.com"}, "preferred_warehouse_id": {"new-jersey-hub"}},
			getReferer:    "/shop/frame-desk",
			getStatus:     http.StatusNotFound,
			getSnippet:    "product_not_found",
			hasPublicCSRF: true,
		},
		{
			getMethod:       http.MethodPost,
			getPath:         "/api/app/comments/cmt-seed-studio-console-flagged/moderate",
			getForm:         url.Values{"csrf_token": {buildInternalToken}, "status": {"flagged"}, "reason": {"Escalated by QA scenario."}},
			getReferer:      "/app/comments",
			getStatus:       http.StatusBadRequest,
			getSnippet:      "moderation_failed",
			hasSession:      true,
			hasInternalCSRF: true,
		},
		{
			getMethod:       http.MethodPost,
			getPath:         "/api/app/comments/bulk-moderate",
			getForm:         url.Values{"csrf_token": {buildInternalToken}, "ids": {"cmt-seed-studio-console-flagged,cmt-seed-desk-pending"}, "status": {"approved"}, "reason": {"Bulk QA review."}},
			getReferer:      "/app/comments",
			getStatus:       http.StatusBadRequest,
			getSnippet:      "bulk_moderation_failed",
			hasSession:      true,
			hasInternalCSRF: true,
		},
		{
			getMethod:       http.MethodPost,
			getPath:         "/api/app/inventory/frame-desk/threshold",
			getForm:         url.Values{"csrf_token": {buildInternalToken}, "warehouse_id": {"new-jersey-hub"}, "reorder_point": {"13"}, "safety_stock": {"4"}},
			getReferer:      "/app/inventory/frame-desk",
			getStatus:       http.StatusBadRequest,
			getSnippet:      "threshold_update_failed",
			hasSession:      true,
			hasInternalCSRF: true,
		},
		{
			getMethod:       http.MethodPost,
			getPath:         "/api/app/inventory/frame-desk/update",
			getForm:         url.Values{"csrf_token": {buildInternalToken}, "warehouse_id": {"new-jersey-hub"}, "on_hand": {"10"}, "reserved": {"2"}, "damaged": {"1"}, "inbound": {"6"}, "reorder_point": {"14"}, "safety_stock": {"5"}, "status": {"recovery"}},
			getReferer:      "/app/inventory/frame-desk",
			getStatus:       http.StatusBadRequest,
			getSnippet:      "inventory_update_failed",
			hasSession:      true,
			hasInternalCSRF: true,
		},
		{
			getMethod:       http.MethodPost,
			getPath:         "/api/app/preferences",
			getForm:         url.Values{"csrf_token": {buildInternalToken}, "theme": {"light"}, "locale": {"en"}, "density": {"compact"}, "default_warehouse_id": {"new-jersey-hub"}},
			getReferer:      "/app/settings",
			getStatus:       http.StatusBadRequest,
			getSnippet:      "preferences_save_failed",
			hasSession:      true,
			hasInternalCSRF: true,
		},
		{
			getMethod:       http.MethodPut,
			getPath:         "/api/app/preferences",
			getForm:         url.Values{"csrf_token": {buildInternalToken}, "theme": {"light"}, "locale": {"en"}, "density": {"compact"}, "default_warehouse_id": {"new-jersey-hub"}},
			getReferer:      "/app/settings/appearance",
			getStatus:       http.StatusBadRequest,
			getSnippet:      "preferences_save_failed",
			hasSession:      true,
			hasInternalCSRF: true,
		},
		{
			getMethod:       http.MethodPost,
			getPath:         "/api/app/saved-views",
			getForm:         url.Values{"csrf_token": {buildInternalToken}, "name": {"Ops triage"}, "scope": {"inventory"}, "filters_json": {`{"status":"critical"}`}, "sort_key": {"updated"}, "sort_direction": {"desc"}, "density": {"compact"}, "warehouse_id": {"illinois-hub"}},
			getReferer:      "/app/settings",
			getStatus:       http.StatusBadRequest,
			getSnippet:      "saved_view_create_failed",
			hasSession:      true,
			hasInternalCSRF: true,
		},
		{
			getMethod:       http.MethodPost,
			getPath:         "/api/app/saved-views/import",
			getForm:         url.Values{"csrf_token": {buildInternalToken}, "views_json": {stringifySavedViewTransferDocument(buildImportDocument)}},
			getReferer:      "/app/settings",
			getStatus:       http.StatusBadRequest,
			getSnippet:      "saved_view_import_failed",
			hasSession:      true,
			hasInternalCSRF: true,
		},
		{
			getMethod:       http.MethodPost,
			getPath:         "/api/app/transfers",
			getForm:         url.Values{"csrf_token": {buildInternalToken}, "source_warehouse_id": {"illinois-hub"}, "destination_warehouse_id": {"new-jersey-hub"}, "reason": {"Rebalance due to east-coast demand."}, "recommended_by": {"ops_lead"}},
			getReferer:      "/app/transfers",
			getStatus:       http.StatusBadRequest,
			getSnippet:      "transfer_create_failed",
			hasSession:      true,
			hasInternalCSRF: true,
		},
		{
			getMethod:       http.MethodPost,
			getPath:         "/api/app/receiving/rcv-illinois-001/reconcile",
			getForm:         url.Values{"csrf_token": {buildInternalToken}, "status": {"closed"}, "discrepancy_summary": {"Closeout completed with QA discrepancy notes."}},
			getReferer:      "/app/receiving/rcv-illinois-001",
			getStatus:       http.StatusBadRequest,
			getSnippet:      "receiving_reconcile_failed",
			hasSession:      true,
			hasInternalCSRF: true,
		},
		{
			getMethod:       http.MethodPost,
			getPath:         "/api/app/purchase-orders/po-1042/status",
			getForm:         url.Values{"csrf_token": {buildInternalToken}, "status": {"approved"}},
			getReferer:      "/app/purchase-orders/po-1042",
			getStatus:       http.StatusBadRequest,
			getSnippet:      "purchase_order_status_failed",
			hasSession:      true,
			hasInternalCSRF: true,
		},
		{
			getMethod:       http.MethodPost,
			getPath:         "/api/app/purchase-orders",
			getForm:         url.Values{"csrf_token": {buildInternalToken}, "vendor_name": {"Northline Fabrication"}, "warehouse_id": {"new-jersey-hub"}, "product_sku": {"frame-desk"}, "quantity": {"9"}, "eta": {"Wed 08:00"}, "priority_note": {"Top up east coast lane"}, "status": {"submitted"}},
			getReferer:      "/app/purchase-orders",
			getStatus:       http.StatusBadRequest,
			getSnippet:      "purchase_order_create_failed",
			hasSession:      true,
			hasInternalCSRF: true,
		},
		{
			getMethod:       http.MethodPost,
			getPath:         "/api/app/products",
			getForm:         url.Values{"csrf_token": {buildInternalToken}, "sku": {"atlas-prototype-desk"}, "slug": {"atlas-prototype-desk"}, "title": {"Atlas Prototype Desk"}, "category": {"workspace"}, "price_cents": {"129900"}, "status": {"launch-ready"}, "summary": {"Prototype desk for QA coverage."}, "warehouse_id": {"new-jersey-hub"}, "available": {"6"}, "inbound": {"2"}},
			getReferer:      "/app/products",
			getStatus:       http.StatusBadRequest,
			getSnippet:      "product_create_failed",
			hasSession:      true,
			hasInternalCSRF: true,
		},
		{
			getMethod:       http.MethodPost,
			getPath:         "/api/app/products/frame-desk/update",
			getForm:         url.Values{"csrf_token": {buildInternalToken}, "sku": {"frame-desk"}, "slug": {"frame-desk"}, "title": {"Frame Desk"}, "category": {"workspace"}, "price_cents": {"199900"}, "status": {"active"}, "summary": {"Updated QA summary."}, "warehouse_id": {"new-jersey-hub"}, "available": {"7"}, "inbound": {"3"}},
			getReferer:      "/app/products/frame-desk",
			getStatus:       http.StatusBadRequest,
			getSnippet:      "product_update_failed",
			hasSession:      true,
			hasInternalCSRF: true,
		},
		{
			getMethod:       http.MethodPost,
			getPath:         "/api/app/products/frame-desk/delete",
			getForm:         url.Values{"csrf_token": {buildInternalToken}},
			getReferer:      "/app/products/frame-desk",
			getStatus:       http.StatusBadRequest,
			getSnippet:      "product_delete_failed",
			hasSession:      true,
			hasInternalCSRF: true,
		},
	}

	for _, buildCase := range buildCases {
		buildT.Run(strings.NewReplacer("/", "_", "-", "_").Replace(strings.TrimPrefix(buildCase.getPath, "/")), func(buildT *testing.T) {
			buildCookies := []*http.Cookie{}
			if buildCase.hasSession {
				buildCookies = append(buildCookies, buildAtlasSessionCookie())
			}
			if buildCase.hasPublicCSRF {
				buildCookies = append(buildCookies, buildPublicCookie)
			}
			if buildCase.hasInternalCSRF {
				buildCookies = append(buildCookies, buildInternalCookie)
			}
			buildReq := buildAtlasFormRequest(buildCase.getMethod, buildCase.getPath, buildCase.getForm, buildCase.getReferer, buildCookies...)
			buildRes := httptest.NewRecorder()

			buildServer.routes().ServeHTTP(buildRes, buildReq)

			if buildRes.Code != buildCase.getStatus {
				buildT.Fatalf("expected %d, got %d for %s", buildCase.getStatus, buildRes.Code, buildCase.getPath)
			}
			if buildBody := buildRes.Body.String(); !strings.Contains(buildBody, buildCase.getSnippet) {
				buildT.Fatalf("expected response for %s to contain %q, got %q", buildCase.getPath, buildCase.getSnippet, buildBody)
			}
		})
	}
}

// TestAtlasServerMockSignInAdditionalBranches covers the remaining mock sign-in branches.
func TestAtlasServerMockSignInAdditionalBranches(buildT *testing.T) {
	buildServer, buildCleanup := newTestAtlasServer(buildT)
	defer buildCleanup()

	buildRedirectReq := httptest.NewRequest(http.MethodGet, "/auth/mock-sign-in?next=%2Fapp%2Finventory", nil)
	buildRedirectReq.AddCookie(buildAtlasSessionCookie())
	buildRedirectRes := httptest.NewRecorder()

	buildServer.routes().ServeHTTP(buildRedirectRes, buildRedirectReq)

	if buildRedirectRes.Code != http.StatusSeeOther {
		buildT.Fatalf("expected %d, got %d", http.StatusSeeOther, buildRedirectRes.Code)
	}
	if buildLocation := buildRedirectRes.Header().Get("Location"); buildLocation != "/app/inventory" {
		buildT.Fatalf("expected redirect to /app/inventory, got %q", buildLocation)
	}

	buildHTMLReq := buildAtlasFormRequest(http.MethodPost, "/auth/mock-sign-in", url.Values{"role": {"not-a-role"}, "next": {"/app/settings"}}, "/auth/mock-sign-in?next=%2Fapp%2Fsettings")
	buildHTMLRes := httptest.NewRecorder()

	buildServer.routes().ServeHTTP(buildHTMLRes, buildHTMLReq)

	if buildHTMLRes.Code != http.StatusSeeOther {
		buildT.Fatalf("expected %d, got %d", http.StatusSeeOther, buildHTMLRes.Code)
	}
	if buildLocation := buildHTMLRes.Header().Get("Location"); !strings.Contains(buildLocation, "invalid-mock-role") || !strings.Contains(buildLocation, "/auth/mock-sign-in") {
		buildT.Fatalf("expected invalid role redirect, got %q", buildLocation)
	}

	buildJSONReq := httptest.NewRequest(http.MethodPost, "/auth/mock-sign-in", strings.NewReader(`{"role":"not-a-role","next":"/app/settings"}`))
	buildJSONReq.Header.Set("Content-Type", "application/json")
	buildJSONRes := httptest.NewRecorder()

	buildServer.routes().ServeHTTP(buildJSONRes, buildJSONReq)

	if buildJSONRes.Code != http.StatusBadRequest {
		buildT.Fatalf("expected %d, got %d", http.StatusBadRequest, buildJSONRes.Code)
	}
	if buildBody := buildJSONRes.Body.String(); !strings.Contains(buildBody, "invalid_mock_sign_in") {
		buildT.Fatalf("expected invalid_mock_sign_in payload, got %q", buildBody)
	}
}

// buildAtlasSessionCookie builds the default internal session cookie used by Atlas server tests.
func buildAtlasSessionCookie() *http.Cookie {
	return &http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"}
}

// buildAtlasFormRequest builds a same-origin Atlas form request with the provided cookies attached.
func buildAtlasFormRequest(buildMethod string, buildPath string, buildForm url.Values, buildReferer string, buildCookies ...*http.Cookie) *http.Request {
	buildReq := httptest.NewRequest(buildMethod, buildPath, strings.NewReader(buildForm.Encode()))
	buildReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	buildReq.Header.Set("Origin", "http://example.com")
	buildReq.Header.Set("Referer", "http://example.com"+buildReferer)
	for _, buildCookie := range buildCookies {
		buildReq.AddCookie(buildCookie)
	}
	return buildReq
}

// stringifySavedViewTransferDocument serializes a saved-view transfer document for import-request tests.
func stringifySavedViewTransferDocument(buildDocument savedViewTransferDocument) string {
	buildItems := make([]string, 0, len(buildDocument.Items))
	for _, buildItem := range buildDocument.Items {
		buildItems = append(buildItems, `{"name":"`+buildItem.Name+`","scope":"`+buildItem.Scope+`","sortKey":"`+buildItem.SortKey+`","sortDirection":"`+buildItem.SortDirection+`","density":"`+buildItem.Density+`","warehouseId":"`+buildItem.WarehouseID+`","filtersJSON":"`+strings.ReplaceAll(buildItem.FiltersJSON, `"`, `\"`)+`"}`)
	}
	return `{"items":[` + strings.Join(buildItems, ",") + `]}`
}
