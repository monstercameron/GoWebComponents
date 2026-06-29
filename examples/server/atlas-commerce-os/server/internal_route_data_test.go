package main

import (
	"testing"

	serverdb "github.com/monstercameron/GoWebComponents/examples/server/atlas-commerce-os/server/db"
	"github.com/monstercameron/GoWebComponents/examples/server/atlas-commerce-os/shared/repository"
)

func TestInternalRouteDataCountAndFallbackHelpers(parseT *testing.T) {
	parseComments := []serverdb.CommentRecord{
		{Status: " pending "},
		{Status: "FLAGGED"},
		{Status: "approved"},
	}
	if parseGot := countCommentsByStatus(parseComments, "pending"); parseGot != 1 {
		parseT.Fatalf("countCommentsByStatus(pending) = %d, want 1", parseGot)
	}
	if parseGot2 := countCommentsByStatus(parseComments, "flagged"); parseGot2 != 1 {
		parseT.Fatalf("countCommentsByStatus(flagged) = %d, want 1", parseGot2)
	}

	parseTransfers := []serverdb.TransferRecord{
		{Status: "submitted"},
		{Status: " approved "},
		{Status: "submitted"},
	}
	if parseGot3 := countTransfersByStatus(parseTransfers, "submitted"); parseGot3 != 2 {
		parseT.Fatalf("countTransfersByStatus(submitted) = %d, want 2", parseGot3)
	}

	parseReceiving := []serverdb.ReceivingSessionRecord{
		{Status: "open"},
		{Status: " closed "},
		{Status: "review"},
	}
	if parseGot4 := countOpenReceivingSessions(parseReceiving); parseGot4 != 2 {
		parseT.Fatalf("countOpenReceivingSessions() = %d, want 2", parseGot4)
	}

	parseWarehousePressure := []serverdb.WarehousePressureRecord{
		{ID: "nj", Name: "New Jersey", RiskCount: 2, Inbound: 4},
		{ID: "nv", Name: "", RiskCount: 5, Inbound: 6},
	}
	if parseGot5 := countWarehouseRiskLanes(parseWarehousePressure); parseGot5 != 7 {
		parseT.Fatalf("countWarehouseRiskLanes() = %d, want 7", parseGot5)
	}
	if parseGot6 := countWarehouseInbound(parseWarehousePressure); parseGot6 != 10 {
		parseT.Fatalf("countWarehouseInbound() = %d, want 10", parseGot6)
	}
	if parseGot7 := fallbackWarehousePressure(nil); parseGot7 != "No active lane" {
		parseT.Fatalf("fallbackWarehousePressure(nil) = %q, want No active lane", parseGot7)
	}
	if parseGot8 := fallbackWarehousePressure(parseWarehousePressure); parseGot8 != "nv" {
		parseT.Fatalf("fallbackWarehousePressure() = %q, want nv", parseGot8)
	}

	parseInventory := []repository.InventoryRow{
		{WeeklyUnits: 7, ReorderUnits: 0, MarketPressure: "stable"},
		{WeeklyUnits: 5, ReorderUnits: 3, MarketPressure: "steady"},
		{WeeklyUnits: 6, ReorderUnits: 0, MarketPressure: " hot market "},
	}
	if parseGot9 := countWarehouseDemand(parseInventory); parseGot9 != 18 {
		parseT.Fatalf("countWarehouseDemand() = %d, want 18", parseGot9)
	}
	if parseGot10 := countWarehouseUrgentItems(parseInventory); parseGot10 != 2 {
		parseT.Fatalf("countWarehouseUrgentItems() = %d, want 2", parseGot10)
	}

	parseOrders := []serverdb.PurchaseOrderRecord{
		{WarehouseID: "nj", WarehouseName: "New Jersey Hub", Status: "submitted"},
		{WarehouseID: "nj", WarehouseName: "New Jersey Hub", Status: "approved"},
		{WarehouseID: "il", WarehouseName: "Illinois Hub", Status: "submitted"},
	}
	if parseGot11 := countPurchaseOrdersByStatus(parseOrders, "submitted"); parseGot11 != 2 {
		parseT.Fatalf("countPurchaseOrdersByStatus(submitted) = %d, want 2", parseGot11)
	}
	if parseGot12 := fallbackPurchaseOrderWarehouse(nil); parseGot12 != "No active inbound lane" {
		parseT.Fatalf("fallbackPurchaseOrderWarehouse(nil) = %q, want No active inbound lane", parseGot12)
	}
	if parseGot13 := fallbackPurchaseOrderWarehouse(parseOrders); parseGot13 != "New Jersey Hub" {
		parseT.Fatalf("fallbackPurchaseOrderWarehouse() = %q, want New Jersey Hub", parseGot13)
	}
}

func TestInternalRouteDataSummaryBuilders(parseT *testing.T) {
	parseComments := []serverdb.CommentRecord{
		{Status: "pending"},
		{Status: "flagged"},
		{Status: "approved"},
	}
	parseTransfers := []serverdb.TransferRecord{
		{Status: "submitted"},
		{Status: "approved"},
	}
	parseReceiving := []serverdb.ReceivingSessionRecord{
		{Status: "open"},
		{Status: "closed"},
	}
	parseOrders := []serverdb.PurchaseOrderRecord{
		{WarehouseID: "nj", WarehouseName: "New Jersey Hub", Status: "submitted"},
		{WarehouseID: "nj", WarehouseName: "New Jersey Hub", Status: "approved"},
		{WarehouseID: "il", WarehouseName: "Illinois Hub", Status: "on_hold"},
	}
	parseInventory := []repository.InventoryRow{
		{SKU: "frame-desk", WarehouseID: "new-jersey-hub", Available: 2, Inbound: 1, ReorderUnits: 5, Status: "promise_risk", WeeklyUnits: 8},
		{SKU: "cable-bridge", WarehouseID: "new-jersey-hub", Available: 20, Inbound: 0, ReorderUnits: 0, Status: "balanced", WeeklyUnits: 14},
	}

	parseDashboard := buildDashboardSummary(parseComments, parseTransfers, parseReceiving, parseOrders)
	if parseDashboard.Headline != "Demand and operations overview" || len(parseDashboard.Items) != 4 {
		parseT.Fatalf("buildDashboardSummary() = %+v", parseDashboard)
	}
	if parseDashboard.Items[0].Value != "3 queued" {
		parseT.Fatalf("unexpected dashboard buyer inbox value: %+v", parseDashboard.Items[0])
	}

	parseInventorySummary := buildInventorySummary(parseInventory)
	if parseInventorySummary.Headline != "Inventory pressure baseline" || len(parseInventorySummary.Items) != 4 {
		parseT.Fatalf("buildInventorySummary() = %+v", parseInventorySummary)
	}

	parseWarehouseOps := buildWarehouseOpsSummary([]serverdb.WarehousePressureRecord{
		{ID: "nj", Name: "New Jersey Hub", RiskCount: 2, Inbound: 4},
		{ID: "nv", Name: "", RiskCount: 5, Inbound: 6},
	})
	if parseWarehouseOps.Items[3].Value != "nv" {
		parseT.Fatalf("unexpected warehouse priority lane: %+v", parseWarehouseOps.Items[3])
	}

	parseWarehouseDetail := buildWarehouseDetailSummary(parseInventory, parseOrders[:2], "new-jersey-hub")
	if parseWarehouseDetail.Headline != "Warehouse route baseline" || parseWarehouseDetail.Items[2].Value != "2" {
		parseT.Fatalf("buildWarehouseDetailSummary() = %+v", parseWarehouseDetail)
	}

	parsePurchaseOrders := buildPurchaseOrderSummary(parseOrders)
	if parsePurchaseOrders.Headline != "Vendor replenishment baseline" || parsePurchaseOrders.Items[3].Value != "New Jersey Hub" {
		parseT.Fatalf("buildPurchaseOrderSummary() = %+v", parsePurchaseOrders)
	}

	parseCommentSummary := buildCommentSummary(parseComments)
	if parseCommentSummary.Headline != "Buyer inbox baseline" || parseCommentSummary.Items[1].Value != "1" || parseCommentSummary.Items[2].Value != "1" {
		parseT.Fatalf("buildCommentSummary() = %+v", parseCommentSummary)
	}

	parseSettings := buildSettingsSummary(serverdb.PreferencesRecord{}, []repository.SavedView{{Name: "Low stock"}})
	if parseSettings.Headline != "Workspace preferences and transfer tools" || parseSettings.Items[0].Value != "dark" || parseSettings.Items[1].Value != "en" || parseSettings.Items[2].Value != "1" || parseSettings.Items[3].Value != "new-jersey-hub" {
		parseT.Fatalf("buildSettingsSummary() = %+v", parseSettings)
	}
}
