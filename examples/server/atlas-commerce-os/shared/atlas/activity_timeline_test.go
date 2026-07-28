package atlas

import (
	"strings"
	"testing"
)

// TestDashboardActivityItemsGroupAndLimit verifies activity entries are grouped by source and capped per source for scanability.
func TestDashboardActivityItemsGroupAndLimit(parseT *testing.T) {
	parseT.Parallel()

	parsePage := dashboardPage{
		Comments: []commentRecord{
			{ID: "c1", Subject: "Comment 1", Body: "Body 1", Status: "pending"},
			{ID: "c2", Subject: "Comment 2", Body: "Body 2", Status: "approved"},
			{ID: "c3", Subject: "Comment 3", Body: "Body 3", Status: "flagged"},
		},
		Transfers: []transferRecord{
			{ID: "t1", SourceWarehouseID: "a", DestinationWarehouse: "b", Status: "draft", Reason: "R1"},
			{ID: "t2", SourceWarehouseID: "a", DestinationWarehouse: "c", Status: "submitted", Reason: "R2"},
			{ID: "t3", SourceWarehouseID: "a", DestinationWarehouse: "d", Status: "approved", Reason: "R3"},
		},
		Receiving: []receivingRecord{
			{ID: "r1", Status: "open", DiscrepancySummary: "D1"},
			{ID: "r2", Status: "in_review", DiscrepancySummary: "D2"},
			{ID: "r3", Status: "closed", DiscrepancySummary: "D3"},
		},
		Orders: []purchaseOrderRecord{
			{ID: "po1", VendorName: "V1", Status: "submitted", PriorityNote: "P1"},
			{ID: "po2", VendorName: "V2", Status: "approved", PriorityNote: "P2"},
			{ID: "po3", VendorName: "V3", Status: "on_hold", PriorityNote: "P3"},
		},
	}
	parseItems := dashboardActivityItems(parsePage)
	if len(parseItems) != 8 {
		parseT.Fatalf("expected 8 grouped activity items (2 per source), got %d", len(parseItems))
	}
	if parseItems[0].Kicker != "Moderation" || parseItems[2].Kicker != "Transfer" || parseItems[4].Kicker != "Receiving" || parseItems[6].Kicker != "Purchase order" {
		parseT.Fatalf("unexpected activity grouping order: %#v", parseItems)
	}
}

// TestDashboardActivityFeedRendersStatusesAndEmptyState verifies status/meta rendering and empty-state fallback for the activity feed card.
func TestDashboardActivityFeedRendersStatusesAndEmptyState(parseT *testing.T) {
	parseT.Parallel()

	parseFilledMarkup, parseErr := renderAtlasNodeForTest(dashboardActivityFeed(dashboardPage{
		Comments:  sampleCommentRecords(),
		Transfers: sampleTransferRecords(),
		Receiving: sampleReceivingRecords(),
		Orders:    samplePurchaseOrders(),
	}))
	if parseErr != nil {
		parseT.Fatalf("dashboardActivityFeed(filled) render failed: %v", parseErr)
	}
	if !strings.Contains(parseFilledMarkup, "Activity feed") {
		parseT.Fatalf("expected activity feed heading, got %q", parseFilledMarkup)
	}
	if !strings.Contains(parseFilledMarkup, "Moderation") || !strings.Contains(parseFilledMarkup, "Transfer") || !strings.Contains(parseFilledMarkup, "Receiving") {
		parseT.Fatalf("expected grouped activity kickers, got %q", parseFilledMarkup)
	}

	parseEmptyMarkup, parseErr := renderAtlasNodeForTest(dashboardActivityFeed(dashboardPage{}))
	if parseErr != nil {
		parseT.Fatalf("dashboardActivityFeed(empty) render failed: %v", parseErr)
	}
	// The empty state is now a row INSIDE the manifest table, so the feed keeps its
	// header and bottom rule instead of collapsing to a bare sentence. The copy says
	// what will fill the queue rather than reporting the absence as a status.
	if !strings.Contains(parseEmptyMarkup, "Nothing has moved yet.") {
		parseT.Fatalf("expected activity empty state, got %q", parseEmptyMarkup)
	}
}

// TestThresholdHistoryRowsRenderTimestampsAndNotes verifies timeline rows surface timestamp, status chips, and operator note detail.
func TestThresholdHistoryRowsRenderTimestampsAndNotes(parseT *testing.T) {
	parseT.Parallel()

	parseMarkup, parseErr := renderAtlasNodeForTest(inventoryThresholdHistoryPanel(
		inventoryThresholdHistoryPanelPage{
			SKU: "frame-desk",
			Items: []inventoryThresholdHistoryItem{
				{
					ID:           "timeline-1",
					ProductSKU:   "frame-desk",
					WarehouseID:  "new-jersey-hub",
					ReorderPoint: 18,
					SafetyStock:  9,
					ActorName:    "Atlas Ops",
					Summary:      "Threshold tuned",
					Detail:       "Raised safety stock after dock-side discrepancy.",
					CreatedAt:    "2026-03-25T09:30:00Z",
				},
			},
		},
		inventoryDetailPage{SKU: "frame-desk", Title: "Frame Desk"},
	))
	if parseErr != nil {
		parseT.Fatalf("inventoryThresholdHistoryPanel timeline render failed: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "Threshold changes") {
		parseT.Fatalf("expected threshold timeline section, got %q", parseMarkup)
	}
	// The history is a table now, so the actor is a cell under a "By" column rather than
	// a sentence reading "Updated by Atlas Ops" on every card, and the two threshold
	// figures are cells under "Reorder" and "Safety" instead of repeating their own
	// labels once per row. Same facts, one label each instead of one label per entry.
	if !strings.Contains(parseMarkup, "2026-03-25T09:30:00Z") || !strings.Contains(parseMarkup, ">Atlas Ops<") {
		parseT.Fatalf("expected timeline timestamp and actor cell, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, ">Reorder<") || !strings.Contains(parseMarkup, ">Safety<") {
		parseT.Fatalf("expected threshold audit columns, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, ">18<") || !strings.Contains(parseMarkup, ">9<") {
		parseT.Fatalf("expected threshold audit values, got %q", parseMarkup)
	}
}
