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

// TestTransferCreationFlowIntegration verifies recommendation lookup, transfer submission, persistence, and refreshed queue visibility.
func TestTransferCreationFlowIntegration(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseRecommendationsReq := httptest.NewRequest(http.MethodGet, "/api/app/inventory/frame-desk/transfer-recommendations", nil)
	parseRecommendationsReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseRecommendationsRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseRecommendationsRes, parseRecommendationsReq)
	if parseRecommendationsRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseRecommendationsRes.Code)
	}
	var parseRecommendationsPayload struct {
		Items []struct {
			SourceWarehouseID      string `json:"sourceWarehouseId"`
			DestinationWarehouseID string `json:"destinationWarehouseId"`
			Reason                 string `json:"reason"`
		} `json:"items"`
	}
	if parseErr := json.Unmarshal(parseRecommendationsRes.Body.Bytes(), &parseRecommendationsPayload); parseErr != nil {
		parseT.Fatalf("decode transfer recommendations payload: %v", parseErr)
	}
	if len(parseRecommendationsPayload.Items) == 0 {
		parseT.Fatal("expected at least one transfer recommendation for frame-desk")
	}
	if parseRecommendationsPayload.Items[0].SourceWarehouseID == "" || parseRecommendationsPayload.Items[0].DestinationWarehouseID == "" {
		parseT.Fatalf("expected recommendation source/destination IDs, got %+v", parseRecommendationsPayload.Items[0])
	}

	parseTransfersBeforeReq := httptest.NewRequest(http.MethodGet, "/api/app/transfers", nil)
	parseTransfersBeforeReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseTransfersBeforeRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseTransfersBeforeRes, parseTransfersBeforeReq)
	if parseTransfersBeforeRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseTransfersBeforeRes.Code)
	}
	var parseTransfersBefore struct {
		Items []map[string]any `json:"items"`
	}
	if parseErr := json.Unmarshal(parseTransfersBeforeRes.Body.Bytes(), &parseTransfersBefore); parseErr != nil {
		parseT.Fatalf("decode transfers before payload: %v", parseErr)
	}

	parseCSRFTok, parseCSRFCookie := loadCSRFFromPage(parseT, parseServer, "/app/transfers")
	parseTransferForm := url.Values{
		"source_warehouse_id":      {"nevada-hub"},
		"destination_warehouse_id": {"new-jersey-hub"},
		"reason":                   {"Integration transfer for queue refresh check."},
		"recommended_by":           {"Integration suite"},
		"csrf_token":               {parseCSRFTok},
	}
	parseCreateReq := httptest.NewRequest(http.MethodPost, "/api/app/transfers", strings.NewReader(parseTransferForm.Encode()))
	parseCreateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseCreateReq.Header.Set("Referer", "http://example.com/app/transfers")
	parseCreateReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseCreateReq.AddCookie(parseCSRFCookie)
	parseCreateRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseCreateRes, parseCreateReq)
	if parseCreateRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d body=%q", http.StatusSeeOther, parseCreateRes.Code, parseCreateRes.Body.String())
	}

	parseTransfersAfterReq := httptest.NewRequest(http.MethodGet, "/api/app/transfers", nil)
	parseTransfersAfterReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseTransfersAfterRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseTransfersAfterRes, parseTransfersAfterReq)
	if parseTransfersAfterRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseTransfersAfterRes.Code)
	}
	var parseTransfersAfter struct {
		Items []map[string]any `json:"items"`
	}
	if parseErr := json.Unmarshal(parseTransfersAfterRes.Body.Bytes(), &parseTransfersAfter); parseErr != nil {
		parseT.Fatalf("decode transfers after payload: %v", parseErr)
	}
	if len(parseTransfersAfter.Items) < len(parseTransfersBefore.Items)+1 {
		parseT.Fatalf("expected transfer queue growth, before=%d after=%d", len(parseTransfersBefore.Items), len(parseTransfersAfter.Items))
	}
}
