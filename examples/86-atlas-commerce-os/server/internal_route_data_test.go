package main

import (
	"testing"

	serverdb "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/db"
	"github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/shared/repository"
)

func TestInternalRouteDataCountAndFallbackHelpers(t *testing.T) {
	comments := []serverdb.CommentRecord{
		{Status: " pending "},
		{Status: "FLAGGED"},
		{Status: "approved"},
	}
	if got := countCommentsByStatus(comments, "pending"); got != 1 {
		t.Fatalf("countCommentsByStatus(pending) = %d, want 1", got)
	}
	if got := countCommentsByStatus(comments, "flagged"); got != 1 {
		t.Fatalf("countCommentsByStatus(flagged) = %d, want 1", got)
	}

	transfers := []serverdb.TransferRecord{
		{Status: "submitted"},
		{Status: " approved "},
		{Status: "submitted"},
	}
	if got := countTransfersByStatus(transfers, "submitted"); got != 2 {
		t.Fatalf("countTransfersByStatus(submitted) = %d, want 2", got)
	}

	receiving := []serverdb.ReceivingSessionRecord{
		{Status: "open"},
		{Status: " closed "},
		{Status: "review"},
	}
	if got := countOpenReceivingSessions(receiving); got != 2 {
		t.Fatalf("countOpenReceivingSessions() = %d, want 2", got)
	}

	warehousePressure := []serverdb.WarehousePressureRecord{
		{ID: "nj", Name: "New Jersey", RiskCount: 2, Inbound: 4},
		{ID: "nv", Name: "", RiskCount: 5, Inbound: 6},
	}
	if got := countWarehouseRiskLanes(warehousePressure); got != 7 {
		t.Fatalf("countWarehouseRiskLanes() = %d, want 7", got)
	}
	if got := countWarehouseInbound(warehousePressure); got != 10 {
		t.Fatalf("countWarehouseInbound() = %d, want 10", got)
	}
	if got := fallbackWarehousePressure(nil); got != "No active lane" {
		t.Fatalf("fallbackWarehousePressure(nil) = %q, want No active lane", got)
	}
	if got := fallbackWarehousePressure(warehousePressure); got != "nv" {
		t.Fatalf("fallbackWarehousePressure() = %q, want nv", got)
	}

	inventory := []repository.InventoryRow{
		{WeeklyUnits: 7, ReorderUnits: 0, MarketPressure: "stable"},
		{WeeklyUnits: 5, ReorderUnits: 3, MarketPressure: "steady"},
		{WeeklyUnits: 6, ReorderUnits: 0, MarketPressure: " hot market "},
	}
	if got := countWarehouseDemand(inventory); got != 18 {
		t.Fatalf("countWarehouseDemand() = %d, want 18", got)
	}
	if got := countWarehouseUrgentItems(inventory); got != 2 {
		t.Fatalf("countWarehouseUrgentItems() = %d, want 2", got)
	}

	orders := []serverdb.PurchaseOrderRecord{
		{WarehouseID: "nj", WarehouseName: "New Jersey Hub", Status: "submitted"},
		{WarehouseID: "nj", WarehouseName: "New Jersey Hub", Status: "approved"},
		{WarehouseID: "il", WarehouseName: "Illinois Hub", Status: "submitted"},
	}
	if got := countPurchaseOrdersByStatus(orders, "submitted"); got != 2 {
		t.Fatalf("countPurchaseOrdersByStatus(submitted) = %d, want 2", got)
	}
	if got := fallbackPurchaseOrderWarehouse(nil); got != "No active inbound lane" {
		t.Fatalf("fallbackPurchaseOrderWarehouse(nil) = %q, want No active inbound lane", got)
	}
	if got := fallbackPurchaseOrderWarehouse(orders); got != "New Jersey Hub" {
		t.Fatalf("fallbackPurchaseOrderWarehouse() = %q, want New Jersey Hub", got)
	}
}

func TestInternalRouteDataSummaryBuilders(t *testing.T) {
	comments := []serverdb.CommentRecord{
		{Status: "pending"},
		{Status: "flagged"},
		{Status: "approved"},
	}
	transfers := []serverdb.TransferRecord{
		{Status: "submitted"},
		{Status: "approved"},
	}
	receiving := []serverdb.ReceivingSessionRecord{
		{Status: "open"},
		{Status: "closed"},
	}
	orders := []serverdb.PurchaseOrderRecord{
		{WarehouseID: "nj", WarehouseName: "New Jersey Hub", Status: "submitted"},
		{WarehouseID: "nj", WarehouseName: "New Jersey Hub", Status: "approved"},
		{WarehouseID: "il", WarehouseName: "Illinois Hub", Status: "on_hold"},
	}
	inventory := []repository.InventoryRow{
		{SKU: "frame-desk", WarehouseID: "new-jersey-hub", Available: 2, Inbound: 1, ReorderUnits: 5, Status: "promise_risk", WeeklyUnits: 8},
		{SKU: "cable-bridge", WarehouseID: "new-jersey-hub", Available: 20, Inbound: 0, ReorderUnits: 0, Status: "balanced", WeeklyUnits: 14},
	}

	dashboard := buildDashboardSummary(comments, transfers, receiving, orders)
	if dashboard.Headline != "Demand and operations overview" || len(dashboard.Items) != 4 {
		t.Fatalf("buildDashboardSummary() = %+v", dashboard)
	}
	if dashboard.Items[0].Value != "3 queued" {
		t.Fatalf("unexpected dashboard buyer inbox value: %+v", dashboard.Items[0])
	}

	inventorySummary := buildInventorySummary(inventory)
	if inventorySummary.Headline != "Inventory pressure baseline" || len(inventorySummary.Items) != 4 {
		t.Fatalf("buildInventorySummary() = %+v", inventorySummary)
	}

	warehouseOps := buildWarehouseOpsSummary([]serverdb.WarehousePressureRecord{
		{ID: "nj", Name: "New Jersey Hub", RiskCount: 2, Inbound: 4},
		{ID: "nv", Name: "", RiskCount: 5, Inbound: 6},
	})
	if warehouseOps.Items[3].Value != "nv" {
		t.Fatalf("unexpected warehouse priority lane: %+v", warehouseOps.Items[3])
	}

	warehouseDetail := buildWarehouseDetailSummary(inventory, orders[:2], "new-jersey-hub")
	if warehouseDetail.Headline != "Warehouse route baseline" || warehouseDetail.Items[2].Value != "2" {
		t.Fatalf("buildWarehouseDetailSummary() = %+v", warehouseDetail)
	}

	purchaseOrders := buildPurchaseOrderSummary(orders)
	if purchaseOrders.Headline != "Vendor replenishment baseline" || purchaseOrders.Items[3].Value != "New Jersey Hub" {
		t.Fatalf("buildPurchaseOrderSummary() = %+v", purchaseOrders)
	}

	commentSummary := buildCommentSummary(comments)
	if commentSummary.Headline != "Buyer inbox baseline" || commentSummary.Items[1].Value != "1" || commentSummary.Items[2].Value != "1" {
		t.Fatalf("buildCommentSummary() = %+v", commentSummary)
	}

	settings := buildSettingsSummary(serverdb.PreferencesRecord{}, []repository.SavedView{{Name: "Low stock"}})
	if settings.Headline != "Workspace preferences and transfer tools" || settings.Items[0].Value != "dark" || settings.Items[1].Value != "en" || settings.Items[2].Value != "1" || settings.Items[3].Value != "new-jersey-hub" {
		t.Fatalf("buildSettingsSummary() = %+v", settings)
	}
}
