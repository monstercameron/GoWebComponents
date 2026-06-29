package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/examples/server/atlas-commerce-os/shared/repository"
)

// TestServerLoggingHelperBranches covers direct logging helper and extraction branches.
func TestServerLoggingHelperBranches(parseT *testing.T) {
	parseHeaders := http.Header{}
	parseHeaders.Set("Location", "/app/products/prod-123?atlas_notice=product-saved")
	if parseResourceID := extractServerResourceID(parseHeaders, "/app/products/"); parseResourceID != "prod-123" {
		parseT.Fatalf("expected extracted resource id prod-123, got %q", parseResourceID)
	}
	parseHeaders.Set("Location", "/app/products/prod-123/edit")
	if parseResourceID := extractServerResourceID(parseHeaders, "/app/products/"); parseResourceID != "" {
		parseT.Fatalf("expected nested resource path to be rejected, got %q", parseResourceID)
	}
	parseHeaders.Set("Location", "/app/receiving/rcv-123?atlas_notice=receiving-reconciled")
	if parseResultState := extractServerResultState(parseHeaders); parseResultState != "receiving-reconciled" {
		parseT.Fatalf("expected result state receiving-reconciled, got %q", parseResultState)
	}
	parseHeaders.Set("Location", "://bad-location")
	if parseResultState := extractServerResultState(parseHeaders); parseResultState != "" {
		parseT.Fatalf("expected invalid location to suppress result state, got %q", parseResultState)
	}

	parseServer := &atlasServer{cfg: config{LogsEnabled: true}}
	parseLogBuffer := captureServerLogBuffer(parseT)

	parseServer.logServerRequestEvent(httptest.NewRequest(http.MethodGet, "/assets/logo.svg", nil), http.StatusOK, -5*time.Millisecond, http.Header{})
	parseServer.logServerRequestEvent(httptest.NewRequest(http.MethodDelete, "/api/app/unknown", nil), http.StatusNoContent, 5*time.Millisecond, http.Header{})
	if strings.TrimSpace(parseLogBuffer.String()) != "" {
		parseT.Fatalf("expected skipped helper branches to emit no logs, got %q", parseLogBuffer.String())
	}

	parseMutationHeaders := http.Header{}
	parseMutationHeaders.Set("Location", "/app/products/prod-123?atlas_notice=product-saved")
	parseServer.logServerRequestEvent(httptest.NewRequest(http.MethodPost, "/api/app/products", nil), http.StatusSeeOther, 25*time.Millisecond, parseMutationHeaders)
	parseEntry := decodeServerLogEntryByEvent(parseT, parseLogBuffer, "mutation.result")
	if parseType := strings.TrimSpace(parseEntry["mutation_type"].(string)); parseType != "internal_product_create" {
		parseT.Fatalf("expected internal_product_create mutation type, got %#v", parseEntry["mutation_type"])
	}
	if parseMutationID := strings.TrimSpace(parseEntry["mutation_id"].(string)); parseMutationID != "prod-123" {
		parseT.Fatalf("expected mutation id prod-123, got %#v", parseEntry["mutation_id"])
	}
	if parseResultState := strings.TrimSpace(parseEntry["result_state"].(string)); parseResultState != "product-saved" {
		parseT.Fatalf("expected product-saved result state, got %#v", parseEntry["result_state"])
	}

	parseLogBuffer.Reset()
	parseServer.logServerRequestEvent(httptest.NewRequest(http.MethodPatch, "/api/app/unknown", nil), http.StatusBadRequest, 10*time.Millisecond, http.Header{})
	parseUnknownEntry := decodeServerLogEntryByEvent(parseT, parseLogBuffer, "mutation.result")
	if parseType := strings.TrimSpace(parseUnknownEntry["mutation_type"].(string)); parseType != "mutation_unknown" {
		parseT.Fatalf("expected unknown mutation result log fields, got %#v", parseUnknownEntry)
	}

	parseServer.writeServerLogEvent("bad.event", map[string]any{"invalid": func() {}})
	if !strings.Contains(parseLogBuffer.String(), "server_log_encode_failed") {
		parseT.Fatalf("expected marshal failure fallback log, got %q", parseLogBuffer.String())
	}
}

// TestInventoryCMSHelperBranches covers inventory sort, label, and filter helper branches.
func TestInventoryCMSHelperBranches(parseT *testing.T) {
	if parseLabel := fallbackWarehouseLabel(repository.InventoryRow{WarehouseID: "nj"}); parseLabel != "nj" {
		parseT.Fatalf("expected warehouse id fallback label, got %q", parseLabel)
	}

	parseItemRows := []repository.InventoryRow{
		{Title: "Desk", WarehouseID: "zulu", WarehouseName: "", Available: 5, CoverDays: 12, Inbound: 2, WeeklyUnits: 7, RegionalShare: 3, UpdatedAt: "2026-03-24T10:00:00Z"},
		{Title: "Arm", WarehouseID: "alpha", WarehouseName: "Atlanta", Available: 8, CoverDays: 5, Inbound: 4, WeeklyUnits: 9, RegionalShare: 4, UpdatedAt: "2026-03-25T10:00:00Z"},
	}
	sortWarehouseItemRows(parseItemRows, "warehouse", "")
	if parseItemRows[0].WarehouseID != "alpha" {
		parseT.Fatalf("expected warehouse sort to default ascending, got %+v", parseItemRows)
	}
	sortWarehouseItemRows(parseItemRows, "available", "")
	if parseItemRows[0].Available != 8 {
		parseT.Fatalf("expected available sort to default descending, got %+v", parseItemRows)
	}
	sortWarehouseItemRows(parseItemRows, "cover", "")
	if parseItemRows[0].CoverDays != 5 {
		parseT.Fatalf("expected cover sort to default ascending, got %+v", parseItemRows)
	}
	if parseDirection := normalizeWarehouseItemDirection("updated", ""); parseDirection != "desc" {
		parseT.Fatalf("expected updated direction fallback desc, got %q", parseDirection)
	}

	parseInventoryRows := []repository.InventoryRow{
		{Title: "Desk", Available: 9, DemandScore: 10, WeeklyRevenue: 500, UpdatedAt: "2026-03-24T10:00:00Z"},
		{Title: "Arm", Available: 4, DemandScore: 20, WeeklyRevenue: 900, UpdatedAt: "2026-03-25T10:00:00Z"},
	}
	sortWarehouseInventoryRows(parseInventoryRows, "available")
	if parseInventoryRows[0].Available != 4 {
		parseT.Fatalf("expected available inventory sort ascending, got %+v", parseInventoryRows)
	}
	sortWarehouseInventoryRows(parseInventoryRows, "demand")
	if parseInventoryRows[0].DemandScore != 20 {
		parseT.Fatalf("expected demand inventory sort descending, got %+v", parseInventoryRows)
	}
	sortWarehouseInventoryRows(parseInventoryRows, "revenue")
	if parseInventoryRows[0].WeeklyRevenue != 900 {
		parseT.Fatalf("expected revenue inventory sort descending, got %+v", parseInventoryRows)
	}
	sortWarehouseInventoryRows(parseInventoryRows, "updated")
	if parseInventoryRows[0].UpdatedAt != "2026-03-25T10:00:00Z" {
		parseT.Fatalf("expected default inventory sort by updated descending, got %+v", parseInventoryRows)
	}

	parseFiltered := filterWarehouseInventoryRows([]repository.InventoryRow{
		{SKU: "DESK-001", Title: "Desk", Category: "workspace", MarketPressure: "hot lane", Status: "risk"},
		{SKU: "LAMP-001", Title: "Lamp", Category: "lighting", MarketPressure: "steady", Status: "balanced"},
	}, map[string]string{"q": "desk", "status": "risk"})
	if len(parseFiltered) != 1 || parseFiltered[0].SKU != "DESK-001" {
		parseT.Fatalf("expected one filtered warehouse row, got %+v", parseFiltered)
	}
}

// TestServerRouteHelperBranches covers remaining pure route and snippet helper branches.
func TestServerRouteHelperBranches(parseT *testing.T) {
	if parseSnippet := wasmRuntimeSnippet(false); !strings.Contains(parseSnippet, "without hydration") {
		parseT.Fatalf("expected missing-wasm warning snippet, got %q", parseSnippet)
	}
	if parseSnippet := wasmRuntimeSnippet(true); !strings.Contains(parseSnippet, "instantiateStreaming") {
		parseT.Fatalf("expected hydrated wasm snippet, got %q", parseSnippet)
	}

	if parseLabel := roleLabel("ops_lead"); parseLabel != "Operations Lead" {
		parseT.Fatalf("expected ops_lead label, got %q", parseLabel)
	}
	if parseLabel := roleLabel("custom_role"); parseLabel != "custom role" {
		parseT.Fatalf("expected fallback role label, got %q", parseLabel)
	}
	if parseDescription := roleDescription("warehouse_supervisor"); !strings.Contains(parseDescription, "Warehouse-focused operator role") {
		parseT.Fatalf("expected warehouse supervisor description, got %q", parseDescription)
	}
	if parseDescription := roleDescription("custom_role"); parseDescription != "Mock Atlas internal role." {
		parseT.Fatalf("expected fallback role description, got %q", parseDescription)
	}

	if parsePath := sanitizeNextPath(""); parsePath != "/app/dashboard" {
		parseT.Fatalf("expected empty next path fallback, got %q", parsePath)
	}
	if parsePath := sanitizeNextPath("//evil.example"); parsePath != "/app/dashboard" {
		parseT.Fatalf("expected double-slash next path rejection, got %q", parsePath)
	}
	if parsePath := sanitizeNextPath("/%zz"); parsePath != "/app/dashboard" {
		parseT.Fatalf("expected invalid next path rejection, got %q", parsePath)
	}
	if parsePath := sanitizeNextPath("/app/inventory?sku=desk-001"); parsePath != "/app/inventory?sku=desk-001" {
		parseT.Fatalf("expected safe next path to pass through, got %q", parsePath)
	}
}
