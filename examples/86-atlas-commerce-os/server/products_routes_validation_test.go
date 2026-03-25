package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	serverauth "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/auth"
)

func TestInternalProductCRUDWarehouseReturnRedirects(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	csrfToken, csrfCookie := loadCSRFFromPage(t, server, "/app/products")

	createForm := url.Values{
		"csrf_token":          {csrfToken},
		"sku":                 {"task-lamp-return"},
		"slug":                {"task-lamp-return"},
		"title":               {"Task Lamp Return"},
		"category":            {"lighting"},
		"price_cents":         {"45900"},
		"status":              {"in_stock"},
		"finish":              {"Warm brass"},
		"summary":             {"Focused task lighting for compact workstation layouts."},
		"details":             {"A narrow-footprint task lamp built for workstation clusters and studio benches."},
		"seo_title":           {"Atlas Task Lamp Return"},
		"seo_description":     {"Task lighting with a compact footprint and warehouse-backed fulfillment."},
		"warehouse_id":        {"illinois-hub"},
		"return_warehouse_id": {"illinois-hub"},
		"available":           {"11"},
		"inbound":             {"4"},
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
	if location := createRes.Header().Get("Location"); !strings.Contains(location, "/app/warehouses/illinois-hub/items/task-lamp-return") || !strings.Contains(location, "product-created") {
		t.Fatalf("expected warehouse item redirect for create, got %q", location)
	}

	updateForm := url.Values{
		"csrf_token":           {csrfToken},
		"sku":                  {"task-lamp-return"},
		"slug":                 {"task-lamp-return-pro"},
		"title":                {"Task Lamp Return Pro"},
		"category":             {"lighting"},
		"price_cents":          {"55900"},
		"status":               {"low_stock"},
		"finish":               {"Brushed bronze"},
		"summary":              {"Premium task lighting for focused studio work."},
		"details":              {"Updated premium lighting copy for the Atlas catalog CMS."},
		"current_warehouse_id": {"illinois-hub"},
		"warehouse_id":         {"new-jersey-hub"},
		"return_warehouse_id":  {"new-jersey-hub"},
		"available":            {"2"},
		"inbound":              {"8"},
	}
	updateReq := httptest.NewRequest(http.MethodPost, "/api/app/products/task-lamp-return/update", strings.NewReader(updateForm.Encode()))
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateReq.Header.Set("Origin", "http://example.com")
	updateReq.Header.Set("Referer", "http://example.com/app/products/task-lamp-return")
	updateReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	updateReq.AddCookie(csrfCookie)
	updateRes := httptest.NewRecorder()

	server.routes().ServeHTTP(updateRes, updateReq)

	if updateRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, updateRes.Code)
	}
	if location := updateRes.Header().Get("Location"); !strings.Contains(location, "/app/warehouses/new-jersey-hub/items/task-lamp-return") || !strings.Contains(location, "product-updated") {
		t.Fatalf("expected warehouse item redirect for update, got %q", location)
	}

	deleteForm := url.Values{
		"csrf_token":          {csrfToken},
		"return_warehouse_id": {"new-jersey-hub"},
	}
	deleteReq := httptest.NewRequest(http.MethodPost, "/api/app/products/task-lamp-return-pro/delete", strings.NewReader(deleteForm.Encode()))
	deleteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deleteReq.Header.Set("Origin", "http://example.com")
	deleteReq.Header.Set("Referer", "http://example.com/app/products/task-lamp-return-pro")
	deleteReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	deleteReq.AddCookie(csrfCookie)
	deleteRes := httptest.NewRecorder()

	server.routes().ServeHTTP(deleteRes, deleteReq)

	if deleteRes.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, deleteRes.Code)
	}
	if location := deleteRes.Header().Get("Location"); !strings.Contains(location, "/app/warehouses/new-jersey-hub") || !strings.Contains(location, "product-deleted") {
		t.Fatalf("expected warehouse redirect for delete, got %q", location)
	}
	if _, err := server.store.ProductAdminBySlug(context.Background(), "task-lamp-return-pro"); err == nil {
		t.Fatal("expected deleted product lookup to fail")
	}
}

func TestInternalProductCreateValidationError(t *testing.T) {
	server, cleanup := newTestAtlasServer(t)
	defer cleanup()

	csrfToken, csrfCookie := loadCSRFFromPage(t, server, "/app/products")

	form := url.Values{
		"csrf_token":   {csrfToken},
		"sku":          {"task-lamp-invalid"},
		"slug":         {"task-lamp-invalid"},
		"title":        {""},
		"category":     {"lighting"},
		"price_cents":  {"45900"},
		"status":       {"in_stock"},
		"summary":      {"Focused task lighting for compact workstation layouts."},
		"warehouse_id": {"illinois-hub"},
		"available":    {"11"},
		"inbound":      {"4"},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/app/products", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Referer", "http://example.com/app/products")
	req.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	req.AddCookie(csrfCookie)
	res := httptest.NewRecorder()

	server.routes().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, res.Code)
	}
	var payload struct {
		Error   string            `json:"error"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal validation payload: %v", err)
	}
	if payload.Error != "invalid_product_request" {
		t.Fatalf("unexpected error payload: %+v", payload)
	}
	if payload.Fields["title"] == "" {
		t.Fatalf("expected title validation message, got %+v", payload.Fields)
	}
	if _, err := server.store.ProductAdminBySlug(context.Background(), "task-lamp-invalid"); err == nil {
		t.Fatal("expected invalid product create to leave no persisted record")
	}
}
