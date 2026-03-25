package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	serverauth "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/auth"
	serverdb "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/db"
)

func TestInternalProductCRUDJSONMutations(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCsrfToken, parseCsrfCookie := loadCSRFFromPage(parseT, parseServer, "/app/products")

	parseCreateReq := httptest.NewRequest(http.MethodPost, "/api/app/products", strings.NewReader(`{
		"sku":"task-lamp-json",
		"slug":"task-lamp-json",
		"title":"Task Lamp JSON",
		"category":"lighting",
		"price_cents":45900,
		"status":"in_stock",
		"finish":"Warm brass",
		"summary":"Focused task lighting for compact workstation layouts.",
		"details":"A narrow-footprint task lamp built for workstation clusters and studio benches.",
		"seo_title":"Atlas Task Lamp JSON",
		"seo_description":"Task lighting with a compact footprint and warehouse-backed fulfillment.",
		"warehouse_id":"illinois-hub",
		"available":11,
		"inbound":4
	}`))
	parseCreateReq.Header.Set("Content-Type", "application/json")
	parseCreateReq.Header.Set("Origin", "http://example.com")
	parseCreateReq.Header.Set(csrfHeaderName, parseCsrfToken)
	parseCreateReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseCreateReq.AddCookie(parseCsrfCookie)
	parseCreateRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseCreateRes, parseCreateReq)

	if parseCreateRes.Code != http.StatusCreated {
		parseT.Fatalf("expected %d, got %d", http.StatusCreated, parseCreateRes.Code)
	}
	if parseLocation := parseCreateRes.Header().Get("Location"); parseLocation != "" {
		parseT.Fatalf("expected no redirect location for JSON create, got %q", parseLocation)
	}
	var parseCreated serverdb.ProductAdminRecord
	if parseErr := json.Unmarshal(parseCreateRes.Body.Bytes(), &parseCreated); parseErr != nil {
		parseT.Fatalf("unmarshal create response: %v", parseErr)
	}
	if parseCreated.Slug != "task-lamp-json" || parseCreated.WarehouseID != "illinois-hub" || parseCreated.PriceCents != 45900 {
		parseT.Fatalf("unexpected create response: %+v", parseCreated)
	}
	parsePersistedCreate, parseErr2 := parseServer.store.ProductAdminBySlug(context.Background(), "task-lamp-json")
	if parseErr2 != nil {
		parseT.Fatalf("load created json product: %v", parseErr2)
	}
	if parsePersistedCreate.Title != "Task Lamp JSON" || parsePersistedCreate.WarehouseID != "illinois-hub" {
		parseT.Fatalf("unexpected persisted create state: %+v", parsePersistedCreate)
	}

	parseUpdateReq := httptest.NewRequest(http.MethodPost, "/api/app/products/task-lamp-json/update", strings.NewReader(`{
		"sku":"task-lamp-json",
		"slug":"task-lamp-json-pro",
		"title":"Task Lamp JSON Pro",
		"category":"lighting",
		"price_cents":55900,
		"status":"low_stock",
		"finish":"Brushed bronze",
		"summary":"Premium task lighting for focused studio work.",
		"details":"Updated premium lighting copy for the Atlas catalog CMS.",
		"seo_title":"Atlas Task Lamp JSON Pro",
		"seo_description":"Premium task lighting with constrained inventory and managed restock cues.",
		"current_warehouse_id":"illinois-hub",
		"warehouse_id":"new-jersey-hub",
		"available":2,
		"inbound":8
	}`))
	parseUpdateReq.Header.Set("Content-Type", "application/json")
	parseUpdateReq.Header.Set("Origin", "http://example.com")
	parseUpdateReq.Header.Set(csrfHeaderName, parseCsrfToken)
	parseUpdateReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseUpdateReq.AddCookie(parseCsrfCookie)
	parseUpdateRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseUpdateRes, parseUpdateReq)

	if parseUpdateRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseUpdateRes.Code)
	}
	if parseLocation2 := parseUpdateRes.Header().Get("Location"); parseLocation2 != "" {
		parseT.Fatalf("expected no redirect location for JSON update, got %q", parseLocation2)
	}
	var parseUpdated serverdb.ProductAdminRecord
	if parseErr3 := json.Unmarshal(parseUpdateRes.Body.Bytes(), &parseUpdated); parseErr3 != nil {
		parseT.Fatalf("unmarshal update response: %v", parseErr3)
	}
	if parseUpdated.Slug != "task-lamp-json-pro" || parseUpdated.WarehouseID != "new-jersey-hub" || parseUpdated.Available != 2 || parseUpdated.PriceCents != 55900 {
		parseT.Fatalf("unexpected update response: %+v", parseUpdated)
	}
	parsePersistedUpdate, parseErr2 := parseServer.store.ProductAdminBySlug(context.Background(), "task-lamp-json-pro")
	if parseErr2 != nil {
		parseT.Fatalf("load updated json product: %v", parseErr2)
	}
	if parsePersistedUpdate.Title != "Task Lamp JSON Pro" || parsePersistedUpdate.WarehouseID != "new-jersey-hub" || parsePersistedUpdate.Available != 2 {
		parseT.Fatalf("unexpected persisted update state: %+v", parsePersistedUpdate)
	}

	parseDeleteReq := httptest.NewRequest(http.MethodPost, "/api/app/products/task-lamp-json-pro/delete", nil)
	parseDeleteReq.Header.Set("Content-Type", "application/json")
	parseDeleteReq.Header.Set("Origin", "http://example.com")
	parseDeleteReq.Header.Set(csrfHeaderName, parseCsrfToken)
	parseDeleteReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseDeleteReq.AddCookie(parseCsrfCookie)
	parseDeleteRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseDeleteRes, parseDeleteReq)

	if parseDeleteRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseDeleteRes.Code)
	}
	if parseLocation3 := parseDeleteRes.Header().Get("Location"); parseLocation3 != "" {
		parseT.Fatalf("expected no redirect location for JSON delete, got %q", parseLocation3)
	}
	var parseDeleted map[string]bool
	if parseErr4 := json.Unmarshal(parseDeleteRes.Body.Bytes(), &parseDeleted); parseErr4 != nil {
		parseT.Fatalf("unmarshal delete response: %v", parseErr4)
	}
	if !parseDeleted["ok"] {
		parseT.Fatalf("unexpected delete response: %+v", parseDeleted)
	}
	if _, parseErr5 := parseServer.store.ProductAdminBySlug(context.Background(), "task-lamp-json-pro"); parseErr5 == nil {
		parseT.Fatal("expected deleted JSON product lookup to fail")
	}
}
