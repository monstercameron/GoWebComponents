package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	serverauth "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/auth"
)

// TestInventoryManagementFlowIntegration verifies filtered inventory load, saved-view persistence, threshold edit, and post-mutation list continuity.
func TestInventoryManagementFlowIntegration(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCSRFTok, parseCSRFCookie := loadCSRFFromPage(parseT, parseServer, "/app/inventory")

	parseInventoryReq := httptest.NewRequest(http.MethodGet, "/api/app/inventory?warehouse=new-jersey-hub&status=promise_risk&sort=updated&q=frame", nil)
	parseInventoryReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseInventoryRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseInventoryRes, parseInventoryReq)
	if parseInventoryRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseInventoryRes.Code)
	}
	var parseInventoryPayload struct {
		Items []struct {
			SKU         string `json:"sku"`
			WarehouseID string `json:"warehouseId"`
			Status      string `json:"status"`
		} `json:"items"`
	}
	if parseErr := json.Unmarshal(parseInventoryRes.Body.Bytes(), &parseInventoryPayload); parseErr != nil {
		parseT.Fatalf("decode inventory payload: %v", parseErr)
	}
	if len(parseInventoryPayload.Items) == 0 {
		parseT.Fatal("expected filtered inventory items")
	}
	for _, parseItem := range parseInventoryPayload.Items {
		if parseItem.WarehouseID != "new-jersey-hub" {
			parseT.Fatalf("expected warehouse filter to hold, got %+v", parseItem)
		}
		if parseItem.Status != "promise_risk" {
			parseT.Fatalf("expected status filter to hold, got %+v", parseItem)
		}
	}

	parseSavedViewForm := url.Values{
		"name":           {"Integration inventory flow"},
		"scope":          {"inventory"},
		"filters_json":   {`{"warehouse":"new-jersey-hub","status":"promise_risk","q":"frame"}`},
		"sort_key":       {"updated"},
		"sort_direction": {"desc"},
		"density":        {"compact"},
		"warehouse_id":   {"new-jersey-hub"},
		"csrf_token":     {parseCSRFTok},
	}
	parseSavedViewReq := httptest.NewRequest(http.MethodPost, "/api/app/saved-views", strings.NewReader(parseSavedViewForm.Encode()))
	parseSavedViewReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseSavedViewReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseSavedViewReq.AddCookie(parseCSRFCookie)
	parseSavedViewReq.Header.Set("Referer", "http://example.com/app/inventory")
	parseSavedViewRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseSavedViewRes, parseSavedViewReq)
	if parseSavedViewRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d body=%q", http.StatusSeeOther, parseSavedViewRes.Code, parseSavedViewRes.Body.String())
	}

	parseHistoryBeforeReq := httptest.NewRequest(http.MethodGet, "/api/app/inventory/frame-desk/threshold-history", nil)
	parseHistoryBeforeReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseHistoryBeforeRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseHistoryBeforeRes, parseHistoryBeforeReq)
	if parseHistoryBeforeRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseHistoryBeforeRes.Code)
	}
	var parseHistoryBefore struct {
		Items []map[string]any `json:"items"`
	}
	if parseErr := json.Unmarshal(parseHistoryBeforeRes.Body.Bytes(), &parseHistoryBefore); parseErr != nil {
		parseT.Fatalf("decode threshold history before payload: %v", parseErr)
	}

	parseThresholdForm := url.Values{
		"warehouse_id":  {"new-jersey-hub"},
		"reorder_point": {"24"},
		"safety_stock":  {"12"},
		"return_path":   {"/app/inventory"},
		"csrf_token":    {parseCSRFTok},
	}
	parseThresholdReq := httptest.NewRequest(http.MethodPost, "/api/app/inventory/frame-desk/threshold", strings.NewReader(parseThresholdForm.Encode()))
	parseThresholdReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseThresholdReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseThresholdReq.AddCookie(parseCSRFCookie)
	parseThresholdReq.Header.Set("Referer", "http://example.com/app/inventory/frame-desk")
	parseThresholdRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseThresholdRes, parseThresholdReq)
	if parseThresholdRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d body=%q", http.StatusSeeOther, parseThresholdRes.Code, parseThresholdRes.Body.String())
	}

	parseHistoryAfterReq := httptest.NewRequest(http.MethodGet, "/api/app/inventory/frame-desk/threshold-history", nil)
	parseHistoryAfterReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseHistoryAfterRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseHistoryAfterRes, parseHistoryAfterReq)
	if parseHistoryAfterRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseHistoryAfterRes.Code)
	}
	var parseHistoryAfter struct {
		Items []map[string]any `json:"items"`
	}
	if parseErr := json.Unmarshal(parseHistoryAfterRes.Body.Bytes(), &parseHistoryAfter); parseErr != nil {
		parseT.Fatalf("decode threshold history after payload: %v", parseErr)
	}
	if len(parseHistoryAfter.Items) < len(parseHistoryBefore.Items)+1 {
		parseT.Fatalf("expected threshold history growth, before=%d after=%d", len(parseHistoryBefore.Items), len(parseHistoryAfter.Items))
	}

	parseInventoryReloadReq := httptest.NewRequest(http.MethodGet, "/api/app/inventory?warehouse=new-jersey-hub&status=promise_risk&sort=updated&q=frame", nil)
	parseInventoryReloadReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseInventoryReloadRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseInventoryReloadRes, parseInventoryReloadReq)
	if parseInventoryReloadRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseInventoryReloadRes.Code)
	}
	var parseInventoryReload struct {
		Items []map[string]any `json:"items"`
	}
	if parseErr := json.Unmarshal(parseInventoryReloadRes.Body.Bytes(), &parseInventoryReload); parseErr != nil {
		parseT.Fatalf("decode inventory reload payload: %v", parseErr)
	}
	if len(parseInventoryReload.Items) == 0 {
		parseT.Fatal("expected filtered inventory to remain queryable after threshold mutation")
	}
}
