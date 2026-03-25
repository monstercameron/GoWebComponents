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

func TestInternalProductCRUDJSONMutations(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	csrfToken, csrfCookie := loadCSRFFromPage(t, server, "/app/products")

	createReq := httptest.NewRequest(http.MethodPost, "/api/app/products", strings.NewReader(`{
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
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Origin", "http://example.com")
	createReq.Header.Set(csrfHeaderName, csrfToken)
	createReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	createReq.AddCookie(csrfCookie)
	createRes := httptest.NewRecorder()

	server.routes().ServeHTTP(createRes, createReq)

	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, createRes.Code)
	}
	if location := createRes.Header().Get("Location"); location != "" {
		t.Fatalf("expected no redirect location for JSON create, got %q", location)
	}
	var created serverdb.ProductAdminRecord
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if created.Slug != "task-lamp-json" || created.WarehouseID != "illinois-hub" || created.PriceCents != 45900 {
		t.Fatalf("unexpected create response: %+v", created)
	}
	persistedCreate, err := server.store.ProductAdminBySlug(context.Background(), "task-lamp-json")
	if err != nil {
		t.Fatalf("load created json product: %v", err)
	}
	if persistedCreate.Title != "Task Lamp JSON" || persistedCreate.WarehouseID != "illinois-hub" {
		t.Fatalf("unexpected persisted create state: %+v", persistedCreate)
	}

	updateReq := httptest.NewRequest(http.MethodPost, "/api/app/products/task-lamp-json/update", strings.NewReader(`{
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
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Origin", "http://example.com")
	updateReq.Header.Set(csrfHeaderName, csrfToken)
	updateReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	updateReq.AddCookie(csrfCookie)
	updateRes := httptest.NewRecorder()

	server.routes().ServeHTTP(updateRes, updateReq)

	if updateRes.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, updateRes.Code)
	}
	if location := updateRes.Header().Get("Location"); location != "" {
		t.Fatalf("expected no redirect location for JSON update, got %q", location)
	}
	var updated serverdb.ProductAdminRecord
	if err := json.Unmarshal(updateRes.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal update response: %v", err)
	}
	if updated.Slug != "task-lamp-json-pro" || updated.WarehouseID != "new-jersey-hub" || updated.Available != 2 || updated.PriceCents != 55900 {
		t.Fatalf("unexpected update response: %+v", updated)
	}
	persistedUpdate, err := server.store.ProductAdminBySlug(context.Background(), "task-lamp-json-pro")
	if err != nil {
		t.Fatalf("load updated json product: %v", err)
	}
	if persistedUpdate.Title != "Task Lamp JSON Pro" || persistedUpdate.WarehouseID != "new-jersey-hub" || persistedUpdate.Available != 2 {
		t.Fatalf("unexpected persisted update state: %+v", persistedUpdate)
	}

	deleteReq := httptest.NewRequest(http.MethodPost, "/api/app/products/task-lamp-json-pro/delete", nil)
	deleteReq.Header.Set("Content-Type", "application/json")
	deleteReq.Header.Set("Origin", "http://example.com")
	deleteReq.Header.Set(csrfHeaderName, csrfToken)
	deleteReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	deleteReq.AddCookie(csrfCookie)
	deleteRes := httptest.NewRecorder()

	server.routes().ServeHTTP(deleteRes, deleteReq)

	if deleteRes.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, deleteRes.Code)
	}
	if location := deleteRes.Header().Get("Location"); location != "" {
		t.Fatalf("expected no redirect location for JSON delete, got %q", location)
	}
	var deleted map[string]bool
	if err := json.Unmarshal(deleteRes.Body.Bytes(), &deleted); err != nil {
		t.Fatalf("unmarshal delete response: %v", err)
	}
	if !deleted["ok"] {
		t.Fatalf("unexpected delete response: %+v", deleted)
	}
	if _, err := server.store.ProductAdminBySlug(context.Background(), "task-lamp-json-pro"); err == nil {
		t.Fatal("expected deleted JSON product lookup to fail")
	}
}
