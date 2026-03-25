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

func TestInternalProductCRUDWarehouseReturnRedirects(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCsrfToken, parseCsrfCookie := loadCSRFFromPage(parseT, parseServer, "/app/products")

	parseCreateForm := url.Values{
		"csrf_token":          {parseCsrfToken},
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
	if parseLocation := parseCreateRes.Header().Get("Location"); !strings.Contains(parseLocation, "/app/warehouses/illinois-hub/items/task-lamp-return") || !strings.Contains(parseLocation, "product-created") {
		parseT.Fatalf("expected warehouse item redirect for create, got %q", parseLocation)
	}

	parseUpdateForm := url.Values{
		"csrf_token":           {parseCsrfToken},
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
	parseUpdateReq := httptest.NewRequest(http.MethodPost, "/api/app/products/task-lamp-return/update", strings.NewReader(parseUpdateForm.Encode()))
	parseUpdateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseUpdateReq.Header.Set("Origin", "http://example.com")
	parseUpdateReq.Header.Set("Referer", "http://example.com/app/products/task-lamp-return")
	parseUpdateReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseUpdateReq.AddCookie(parseCsrfCookie)
	parseUpdateRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseUpdateRes, parseUpdateReq)

	if parseUpdateRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseUpdateRes.Code)
	}
	if parseLocation2 := parseUpdateRes.Header().Get("Location"); !strings.Contains(parseLocation2, "/app/warehouses/new-jersey-hub/items/task-lamp-return") || !strings.Contains(parseLocation2, "product-updated") {
		parseT.Fatalf("expected warehouse item redirect for update, got %q", parseLocation2)
	}

	parseDeleteForm := url.Values{
		"csrf_token":          {parseCsrfToken},
		"return_warehouse_id": {"new-jersey-hub"},
	}
	parseDeleteReq := httptest.NewRequest(http.MethodPost, "/api/app/products/task-lamp-return-pro/delete", strings.NewReader(parseDeleteForm.Encode()))
	parseDeleteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseDeleteReq.Header.Set("Origin", "http://example.com")
	parseDeleteReq.Header.Set("Referer", "http://example.com/app/products/task-lamp-return-pro")
	parseDeleteReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseDeleteReq.AddCookie(parseCsrfCookie)
	parseDeleteRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseDeleteRes, parseDeleteReq)

	if parseDeleteRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d", http.StatusSeeOther, parseDeleteRes.Code)
	}
	if parseLocation3 := parseDeleteRes.Header().Get("Location"); !strings.Contains(parseLocation3, "/app/warehouses/new-jersey-hub") || !strings.Contains(parseLocation3, "product-deleted") {
		parseT.Fatalf("expected warehouse redirect for delete, got %q", parseLocation3)
	}
	if _, parseErr := parseServer.store.ProductAdminBySlug(context.Background(), "task-lamp-return-pro"); parseErr == nil {
		parseT.Fatal("expected deleted product lookup to fail")
	}
}

func TestInternalProductCreateValidationError(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCsrfToken, parseCsrfCookie := loadCSRFFromPage(parseT, parseServer, "/app/products")

	parseForm := url.Values{
		"csrf_token":   {parseCsrfToken},
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
	parseReq := httptest.NewRequest(http.MethodPost, "/api/app/products", strings.NewReader(parseForm.Encode()))
	parseReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseReq.Header.Set("Origin", "http://example.com")
	parseReq.Header.Set("Referer", "http://example.com/app/products")
	parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseReq.AddCookie(parseCsrfCookie)
	parseRes := httptest.NewRecorder()

	parseServer.routes().ServeHTTP(parseRes, parseReq)

	if parseRes.Code != http.StatusBadRequest {
		parseT.Fatalf("expected %d, got %d", http.StatusBadRequest, parseRes.Code)
	}
	var parsePayload struct {
		Error   string            `json:"error"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields"`
	}
	if parseErr := json.Unmarshal(parseRes.Body.Bytes(), &parsePayload); parseErr != nil {
		parseT.Fatalf("unmarshal validation payload: %v", parseErr)
	}
	if parsePayload.Error != "invalid_product_request" {
		parseT.Fatalf("unexpected error payload: %+v", parsePayload)
	}
	if parsePayload.Fields["title"] == "" {
		parseT.Fatalf("expected title validation message, got %+v", parsePayload.Fields)
	}
	if _, parseErr2 := parseServer.store.ProductAdminBySlug(context.Background(), "task-lamp-invalid"); parseErr2 == nil {
		parseT.Fatal("expected invalid product create to leave no persisted record")
	}
}
