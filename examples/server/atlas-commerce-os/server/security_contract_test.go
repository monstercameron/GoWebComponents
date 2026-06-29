package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	serverauth "github.com/monstercameron/GoWebComponents/v4/examples/server/atlas-commerce-os/server/auth"
)

func TestAtlasFormPagesExposeCSRFBootstrap(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCases := []struct {
		name     string
		path     string
		internal bool
	}{
		{name: "public product actions", path: "/shop/frame-desk"},
		{name: "products editor", path: "/app/products", internal: true},
		{name: "inventory threshold", path: "/app/inventory/frame-desk", internal: true},
		{name: "comments moderation", path: "/app/comments", internal: true},
		{name: "purchase order detail", path: "/app/purchase-orders/po-1042", internal: true},
		{name: "receiving detail", path: "/app/receiving/rcv-illinois-001", internal: true},
		{name: "settings preferences", path: "/app/settings", internal: true},
	}

	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT2 *testing.T) {
			parseReq := httptest.NewRequest(http.MethodGet, parseCase.path, nil)
			if parseCase.internal {
				parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
			}
			parseRes := httptest.NewRecorder()

			parseServer.routes().ServeHTTP(parseRes, parseReq)

			if parseRes.Code != http.StatusOK {
				parseT2.Fatalf("expected %d, got %d: %s", http.StatusOK, parseRes.Code, parseRes.Body.String())
			}
			parseBody := parseRes.Body.String()
			if !strings.Contains(parseBody, `"csrf":"`) {
				parseT2.Fatalf("expected %s to include csrf in bootstrap payload", parseCase.path)
			}
		})
	}
}

func TestAtlasWriteEndpointsRejectWrongOrigin(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parsePublicToken, parsePublicCookie := loadCSRFFromPage(parseT, parseServer, "/shop/frame-desk")
	parseInternalToken, parseInternalCookie := loadCSRFFromPage(parseT, parseServer, "/app/settings")

	parseCases := []struct {
		name     string
		method   string
		path     string
		form     url.Values
		internal bool
	}{
		{name: "public comment", method: http.MethodPost, path: "/api/public/products/frame-desk/comments", form: url.Values{"csrf_token": {parsePublicToken}}},
		{name: "public quote", method: http.MethodPost, path: "/api/public/products/frame-desk/quote-requests", form: url.Values{"csrf_token": {parsePublicToken}}},
		{name: "public restock", method: http.MethodPost, path: "/api/public/products/frame-desk/restock-requests", form: url.Values{"csrf_token": {parsePublicToken}}},
		{name: "comment moderate", method: http.MethodPost, path: "/api/app/comments/cmt-seed-desk-pending/moderate", form: url.Values{"csrf_token": {parseInternalToken}}, internal: true},
		{name: "bulk comment moderate", method: http.MethodPost, path: "/api/app/comments/bulk-moderate", form: url.Values{"csrf_token": {parseInternalToken}}, internal: true},
		{name: "product create", method: http.MethodPost, path: "/api/app/products", form: url.Values{"csrf_token": {parseInternalToken}}, internal: true},
		{name: "product update", method: http.MethodPost, path: "/api/app/products/frame-desk/update", form: url.Values{"csrf_token": {parseInternalToken}}, internal: true},
		{name: "product delete", method: http.MethodPost, path: "/api/app/products/frame-desk/delete", form: url.Values{"csrf_token": {parseInternalToken}}, internal: true},
		{name: "inventory update", method: http.MethodPost, path: "/api/app/inventory/frame-desk/update", form: url.Values{"csrf_token": {parseInternalToken}}, internal: true},
		{name: "threshold update", method: http.MethodPost, path: "/api/app/inventory/frame-desk/threshold", form: url.Values{"csrf_token": {parseInternalToken}}, internal: true},
		{name: "preferences post", method: http.MethodPost, path: "/api/app/preferences", form: url.Values{"csrf_token": {parseInternalToken}}, internal: true},
		{name: "preferences put", method: http.MethodPut, path: "/api/app/preferences", form: url.Values{"csrf_token": {parseInternalToken}}, internal: true},
		{name: "saved view create", method: http.MethodPost, path: "/api/app/saved-views", form: url.Values{"csrf_token": {parseInternalToken}}, internal: true},
		{name: "saved view import", method: http.MethodPost, path: "/api/app/saved-views/import", form: url.Values{"csrf_token": {parseInternalToken}}, internal: true},
		{name: "transfer create", method: http.MethodPost, path: "/api/app/transfers", form: url.Values{"csrf_token": {parseInternalToken}}, internal: true},
		{name: "purchase order create", method: http.MethodPost, path: "/api/app/purchase-orders", form: url.Values{"csrf_token": {parseInternalToken}}, internal: true},
		{name: "purchase order status", method: http.MethodPost, path: "/api/app/purchase-orders/po-1042/status", form: url.Values{"csrf_token": {parseInternalToken}}, internal: true},
		{name: "receiving reconcile", method: http.MethodPost, path: "/api/app/receiving/rcv-illinois-001/reconcile", form: url.Values{"csrf_token": {parseInternalToken}}, internal: true},
	}

	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT2 *testing.T) {
			parseReq := httptest.NewRequest(parseCase.method, parseCase.path, strings.NewReader(parseCase.form.Encode()))
			parseReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			parseReq.Header.Set("Origin", "https://evil.example")
			if parseCase.internal {
				parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
				parseReq.AddCookie(parseInternalCookie)
			} else {
				parseReq.AddCookie(parsePublicCookie)
			}
			parseRes := httptest.NewRecorder()

			parseServer.routes().ServeHTTP(parseRes, parseReq)

			if parseRes.Code != http.StatusForbidden {
				parseT2.Fatalf("expected %d, got %d: %s", http.StatusForbidden, parseRes.Code, parseRes.Body.String())
			}
			if !strings.Contains(parseRes.Body.String(), "csrf_same_origin_failed") {
				parseT2.Fatalf("expected same-origin csrf failure payload, got %q", parseRes.Body.String())
			}
		})
	}
}

func TestAtlasMockRolesRenderDistinctShellContext(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseRoles := map[string]string{
		"inventory_manager":    "new-jersey-hub",
		"warehouse_supervisor": "illinois-hub",
		"ops_lead":             "nevada-hub",
	}

	for parseRole, parseWarehouse := range parseRoles {
		parseT.Run(parseRole, func(parseT2 *testing.T) {
			parseReq := httptest.NewRequest(http.MethodGet, "/app/settings", nil)
			parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: parseRole})
			parseRes := httptest.NewRecorder()

			parseServer.routes().ServeHTTP(parseRes, parseReq)

			if parseRes.Code != http.StatusOK {
				parseT2.Fatalf("expected %d, got %d: %s", http.StatusOK, parseRes.Code, parseRes.Body.String())
			}
			parseBody := parseRes.Body.String()
			if !strings.Contains(parseBody, `"role":"`+parseRole+`"`) {
				parseT2.Fatalf("expected bootstrap role %q, got %q", parseRole, parseBody)
			}
			if !strings.Contains(parseBody, `"defaultWarehouse":"`+parseWarehouse+`"`) {
				parseT2.Fatalf("expected role default warehouse %q, got %q", parseWarehouse, parseBody)
			}
		})
	}
}
