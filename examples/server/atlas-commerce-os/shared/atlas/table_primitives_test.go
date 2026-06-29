package atlas

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// TestInventoryQueueTableRendersEmptyState verifies the dense table emits a clear empty-state row.
func TestInventoryQueueTableRendersEmptyState(parseT *testing.T) {
	parseT.Parallel()

	parseMarkup, parseErr := ui.RenderToString(inventoryQueueTable(nil))
	if parseErr != nil {
		parseT.Fatalf("inventoryQueueTable(empty) render failed: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "Dense queue table") {
		parseT.Fatalf("expected queue table heading, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "No inventory rows match the current filter set.") {
		parseT.Fatalf("expected queue table empty-state copy, got %q", parseMarkup)
	}
}

// TestInventoryQueueTableRendersRowsAndActions verifies core table columns and row actions render with real data.
func TestInventoryQueueTableRendersRowsAndActions(parseT *testing.T) {
	parseT.Parallel()

	parseRows := inventorySummaryCards(sampleInventoryRows())
	parseMarkup, parseErr := ui.RenderToString(inventoryQueueTable(parseRows))
	if parseErr != nil {
		parseT.Fatalf("inventoryQueueTable(rows) render failed: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "Available") || !strings.Contains(parseMarkup, "Inbound") {
		parseT.Fatalf("expected core queue columns in markup, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "Open SKU") || !strings.Contains(parseMarkup, "Open warehouse ops") {
		parseT.Fatalf("expected row action links in markup, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, parseRows[0].SKU) {
		parseT.Fatalf("expected first SKU %q in markup", parseRows[0].SKU)
	}
}

// TestWarehouseItemNetworkTableRendersSortableHeaders verifies sortable header links include sort and direction query keys.
func TestWarehouseItemNetworkTableRendersSortableHeaders(parseT *testing.T) {
	parseT.Parallel()

	parseMarkup, parseErr := ui.RenderToString(warehouseItemNetworkTable(
		"/app/warehouses/new-jersey-hub/items/frame-desk",
		map[string]string{"sort": "available", "dir": "desc"},
		"new-jersey-hub",
		sampleInventoryRows(),
	))
	if parseErr != nil {
		parseT.Fatalf("warehouseItemNetworkTable render failed: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "Warehouse item table") {
		parseT.Fatalf("expected table heading, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "sort=warehouse") || !strings.Contains(parseMarkup, "dir=asc") {
		parseT.Fatalf("expected sortable warehouse header query parameters, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "sort=available") {
		parseT.Fatalf("expected available sort header query parameter, got %q", parseMarkup)
	}
}

// TestInventoryOperationsRailRendersFilterAndSavedViewBadges verifies filter-chip and saved-view badge copy in the rail.
func TestInventoryOperationsRailRendersFilterAndSavedViewBadges(parseT *testing.T) {
	parseT.Parallel()

	parseMarkup, parseErr := ui.RenderToString(inventoryOperationsRail(internalShellState{
		SummaryLabel:    "Risk lanes",
		SummaryValue:    "4",
		ActiveFilters:   []string{"Status: promise_risk", "Warehouse: new-jersey-hub"},
		ActiveSavedView: "Low stock triage",
		WorkspaceStats: []pageSummaryItem{
			{Label: "Visible SKUs", Value: "12"},
		},
	}))
	if parseErr != nil {
		parseT.Fatalf("inventoryOperationsRail render failed: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "Status: promise_risk") || !strings.Contains(parseMarkup, "Warehouse: new-jersey-hub") {
		parseT.Fatalf("expected filter chips in rail markup, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "Active view") || !strings.Contains(parseMarkup, "Low stock triage") {
		parseT.Fatalf("expected saved-view badge copy in rail markup, got %q", parseMarkup)
	}
}

// TestCommentsRouteRendersBulkModerationAffordance verifies the table route keeps the bulk moderation affordance visible.
func TestCommentsRouteRendersBulkModerationAffordance(parseT *testing.T) {
	parseT.Parallel()

	parsePayload := samplePayloadForRoute(RouteComments, commentList{
		Summary: sampleSummary("Buyer inbox"),
		Items:   sampleCommentRecords(),
	}, nil)
	parseMarkup, parseErr := ui.RenderToString(App(parsePayload))
	if parseErr != nil {
		parseT.Fatalf("comments route render failed: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "Buyer question table") {
		parseT.Fatalf("expected comments table heading, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "/api/app/comments/bulk-moderate") || !strings.Contains(parseMarkup, "Review bulk action") {
		parseT.Fatalf("expected bulk moderation affordance, got %q", parseMarkup)
	}
}
