package atlas

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// TestAtlasInventoryRenderBranchesCoverOverlaysAndFilters verifies inventory renderers, reducers, and filter helpers.
func TestAtlasInventoryRenderBranchesCoverOverlaysAndFilters(parseT *testing.T) {
	parseRows := append([]inventoryRow(nil), sampleInventoryRows()...)
	parseRows = append(parseRows, inventoryRow{
		ID:             "inv-lamp-nevada",
		SKU:            "focus-lamp",
		Slug:           "focus-lamp",
		Title:          "Focus Lamp",
		Category:       "lighting",
		PriceCents:     49900,
		ProductStatus:  "in_stock",
		WarehouseID:    "nevada-hub",
		WarehouseName:  "Nevada Hub",
		Available:      1,
		Inbound:        9,
		CoverDays:      4,
		Status:         "critical",
		WeeklyUnits:    14,
		WeeklyRevenue:  698600,
		ReorderUnits:   6,
		MarketPressure: "Hot market",
		MarketSignal:   "Lighting demand accelerated after showroom launch.",
		UpdatedAt:      "2026-03-26T12:00:00Z",
	})

	parseApproved := reducePurchaseOrderWorkflowState(defaultPurchaseOrderWorkflowState("frame-desk", "new-jersey-hub", RouteInventory), purchaseOrderWorkflowAction{Field: "status", Value: "approved"})
	if parseApproved.Stage != "inbound-confirmed" || parseApproved.LastEditedField != "status" {
		parseT.Fatalf("reducePurchaseOrderWorkflowState(approved) = %#v", parseApproved)
	}
	parseBlocked := reducePurchaseOrderWorkflowState(defaultPurchaseOrderWorkflowState("frame-desk", "new-jersey-hub", RouteInventory), purchaseOrderWorkflowAction{Field: "status", Value: "on_hold"})
	if parseBlocked.Stage != "blocked" {
		parseT.Fatalf("reducePurchaseOrderWorkflowState(on_hold) = %#v", parseBlocked)
	}
	parseDraft := reducePurchaseOrderWorkflowState(defaultPurchaseOrderWorkflowState("frame-desk", "new-jersey-hub", RouteInventory), purchaseOrderWorkflowAction{Field: "priority_note", Value: ""})
	parseDraft = reducePurchaseOrderWorkflowState(parseDraft, purchaseOrderWorkflowAction{Field: "status", Value: "draft"})
	if parseDraft.Stage != "draft-needs-brief" {
		parseT.Fatalf("reducePurchaseOrderWorkflowState(draft-needs-brief) = %#v", parseDraft)
	}
	parseQuantity := reducePurchaseOrderWorkflowState(defaultPurchaseOrderWorkflowState("frame-desk", "new-jersey-hub", RouteInventory), purchaseOrderWorkflowAction{Field: "quantity", Value: "0"})
	if parseQuantity.Stage != "needs-quantity" {
		parseT.Fatalf("reducePurchaseOrderWorkflowState(needs-quantity) = %#v", parseQuantity)
	}

	parseAvailable := filterWarehouseInventoryRows(parseRows, atlasListFilterState{Sort: "available"})
	if len(parseAvailable) != len(parseRows) || parseAvailable[0].Available != 1 {
		parseT.Fatalf("filterWarehouseInventoryRows(available) = %#v", parseAvailable)
	}
	parseDemand := filterWarehouseInventoryRows(parseRows, atlasListFilterState{Sort: "demand"})
	if len(parseDemand) == 0 || parseDemand[0].WeeklyUnits != 14 {
		parseT.Fatalf("filterWarehouseInventoryRows(demand) = %#v", parseDemand)
	}
	parseRevenue := filterWarehouseInventoryRows(parseRows, atlasListFilterState{Sort: "revenue"})
	if len(parseRevenue) == 0 || parseRevenue[0].WeeklyRevenue != 1899000 {
		parseT.Fatalf("filterWarehouseInventoryRows(revenue) = %#v", parseRevenue)
	}
	parseFiltered := filterWarehouseInventoryRows(parseRows, atlasListFilterState{Query: "lamp", Status: "critical"})
	if len(parseFiltered) != 1 || parseFiltered[0].SKU != "focus-lamp" {
		parseT.Fatalf("filterWarehouseInventoryRows(filtered) = %#v", parseFiltered)
	}

	parseQueueMarkup := renderAtlasMarkupForTest(parseT, inventoryQueueRowCells(inventorySummaryCard{
		SKU:           "frame-desk",
		Title:         "Frame Desk",
		Status:        "promise_risk",
		PrimaryLane:   "New Jersey Hub",
		Available:     3,
		Inbound:       4,
		RiskLaneCount: 2,
		LastUpdated:   "2026-03-25T00:00:00Z",
	}))
	for _, parseNeedle := range []string{"frame-desk", "New Jersey Hub", "promise risk"} {
		if !strings.Contains(parseQueueMarkup, parseNeedle) {
			parseT.Fatalf("expected inventory queue row markup to contain %q", parseNeedle)
		}
	}

	parseInventoryPayload := samplePayloadForRoute(RouteInventory, inventoryCMSPage{
		Summary: sampleSummary("Inventory shell"),
		Items:   parseRows,
		Filters: map[string]string{"q": "frame", "status": "promise_risk", "sort": "available"},
	}, nil)
	parseInventoryMarkup := renderAtlasComponentForTest(parseT, func() ui.Node {
		return inventoryCMSContent(parseInventoryPayload)
	})
	for _, parseNeedle := range []string{"Inventory queue", "Saved views", "Lane pressure"} {
		if !strings.Contains(parseInventoryMarkup, parseNeedle) {
			parseT.Fatalf("expected inventory CMS markup to contain %q", parseNeedle)
		}
	}

	if InventoryThresholdHistoryOverlay(Payload{}) != nil {
		parseT.Fatal("expected empty threshold overlay payload to render nil")
	}
	parseOverlayPayload := samplePayloadForRoute(RouteSKUThresholdHistory, inventoryDetailPage{
		SKU:   "frame-desk",
		Title: "Frame Desk",
		Rows:  parseRows,
	}, inventoryThresholdHistoryPanelPage{
		SKU: "frame-desk",
		Items: []inventoryThresholdHistoryItem{
			{
				ID:           "th-001",
				ProductSKU:   "frame-desk",
				WarehouseID:  "new-jersey-hub",
				ReorderPoint: 18,
				SafetyStock:  9,
				ActorName:    "Ops lead",
				Summary:      "Threshold tuned for launch demand",
				Detail:       "Raised reorder protection after east-coast surge.",
				CreatedAt:    "2026-03-25T00:00:00Z",
			},
		},
		Recommendations: []inventoryTransferRecommendationItem{
			{
				ProductSKU:               "frame-desk",
				SourceWarehouseID:        "nevada-hub",
				SourceWarehouseName:      "Nevada Hub",
				DestinationWarehouseID:   "new-jersey-hub",
				DestinationWarehouseName: "New Jersey Hub",
				Quantity:                 4,
				Priority:                 "Promise recovery",
				Reason:                   "Shift stock to protect east-coast commitments.",
			},
		},
	})
	parseOverlayMarkup := renderAtlasMarkupForTest(parseT, InventoryThresholdHistoryOverlay(parseOverlayPayload))
	for _, parseNeedle := range []string{"Threshold history", "Transfer recommendations", "Threshold changes"} {
		if !strings.Contains(parseOverlayMarkup, parseNeedle) {
			parseT.Fatalf("expected threshold overlay markup to contain %q", parseNeedle)
		}
	}

	parseWarehousePayload := samplePayloadForRoute(RouteWarehouseDetail, warehouseInventoryDetailPage{
		Summary:   sampleSummary("Facility detail"),
		Warehouse: sampleWarehouseOpsRecords()[0],
		Inventory: parseRows,
		Orders:    samplePurchaseOrders(),
		Filters: map[string]string{
			"status": "promise_risk",
			"sort":   "available",
		},
	}, nil)
	parseWarehouseMarkup := renderAtlasComponentForTest(parseT, func() ui.Node {
		return warehouseInventoryDetailContent(parseWarehousePayload)
	})
	for _, parseNeedle := range []string{"Filter items in this hub", "Recent purchase orders", "Items in this hub"} {
		if !strings.Contains(parseWarehouseMarkup, parseNeedle) {
			parseT.Fatalf("expected warehouse inventory detail markup to contain %q", parseNeedle)
		}
	}

	parseItemPayload := samplePayloadForRoute(RouteWarehouseItemDetail, warehouseInventoryItemDetailPage{
		Warehouse: sampleWarehouseOpsRecords()[0],
		Item:      parseRows[0],
		Product:   sampleProductAdminCards()[0],
		Network:   parseRows,
		Orders:    nil,
		Filters:   nil,
	}, nil)
	parseItemMarkup := renderAtlasMarkupForTest(parseT, warehouseOpsItemContent(parseItemPayload))
	for _, parseNeedle := range []string{"Market and sales readout", "No open replenishment orders are tied to this item", "Related replenishment orders"} {
		if !strings.Contains(parseItemMarkup, parseNeedle) {
			parseT.Fatalf("expected warehouse item markup to contain %q", parseNeedle)
		}
	}

	parseEditorMarkup := renderAtlasMarkupForTest(parseT, inventoryLaneEditorCardWithOptions(parseRows[0], Payload{CSRF: "atlas-csrf"}, "/app/warehouses/"+parseRows[0].WarehouseID))
	for _, parseNeedle := range []string{"Save lane", `name="return_path"`, parseRows[0].WarehouseID} {
		if !strings.Contains(parseEditorMarkup, parseNeedle) {
			parseT.Fatalf("expected inventory lane editor markup to contain %q", parseNeedle)
		}
	}

	parseFilterForm := ui.UseForm(atlasListFilterState{Query: "desk", Status: "promise_risk", Sort: "revenue"})
	parseFilterMarkup := renderAtlasMarkupForTest(parseT, warehouseInventoryFilterForm("illinois-hub", parseFilterForm, true, ui.UseEvent(func(ui.FormEvent) {}), atlasTransition{}))
	for _, parseNeedle := range []string{"Filter items in this hub", "Updating...", "Search items"} {
		if !strings.Contains(parseFilterMarkup, parseNeedle) {
			parseT.Fatalf("expected warehouse filter form markup to contain %q", parseNeedle)
		}
	}
}

// TestAtlasPublicRenderBranchesCoverFeedbackAndFallbacks verifies public feedback, related-products, and input helper branches.
func TestAtlasPublicRenderBranchesCoverFeedbackAndFallbacks(parseT *testing.T) {
	parseProduct := sampleProductCards()[0]
	parseComments := sampleCommentRecords()

	parseFeedbackMarkup := renderAtlasComponentForTest(parseT, func() ui.Node {
		return publicProductFeedbackSection(parseProduct, parseComments, Payload{CSRF: "atlas-csrf"})
	})
	for _, parseNeedle := range []string{"What buyers are asking before they commit.", "Share your review or question", parseComments[0].Subject} {
		if !strings.Contains(parseFeedbackMarkup, parseNeedle) {
			parseT.Fatalf("expected public feedback markup to contain %q", parseNeedle)
		}
	}

	parseErrorMarkup := renderAtlasMarkupForTest(parseT, publicProductPromiseLanesError(parseProduct, errors.New("lane panel failed"), func() {}))
	for _, parseNeedle := range []string{"Atlas could not load the lane panel.", "Retry lane panel", parseProduct.Title} {
		if !strings.Contains(parseErrorMarkup, parseNeedle) {
			parseT.Fatalf("expected promise-lanes error markup to contain %q", parseNeedle)
		}
	}

	parseRelatedCardMarkup := renderAtlasComponentForTest(parseT, func() ui.Node {
		return publicRelatedProductsCard(parseProduct)
	})
	for _, parseNeedle := range []string{"Related systems", "Open adjacent Atlas systems without leaving the same buying context.", "Repeat-open visits can reuse the cached related-product list"} {
		if !strings.Contains(parseRelatedCardMarkup, parseNeedle) {
			parseT.Fatalf("expected related-products card markup to contain %q", parseNeedle)
		}
	}

	parseRelatedNodesMarkup := renderAtlasMarkupForTest(parseT, ui.Fragment(publicRelatedProductNodes([]relatedProductRecord{
		{
			SKU:           "studio-console",
			Slug:          "studio-console",
			Title:         "Studio Console",
			Category:      "storage",
			Summary:       "Creative-team console.",
			WarehouseID:   "illinois-hub",
			WarehouseName: "Illinois Hub",
			Reason:        "Pairs with modular desk rollouts.",
		},
	})...))
	for _, parseNeedle := range []string{"Studio Console", "Pairs with modular desk rollouts.", "Illinois Hub"} {
		if !strings.Contains(parseRelatedNodesMarkup, parseNeedle) {
			parseT.Fatalf("expected related-product node markup to contain %q", parseNeedle)
		}
	}

	parseInputMarkup := renderAtlasMarkupForTest(parseT, publicCatalogInput("q", "Search", "desk"))
	if !strings.Contains(parseInputMarkup, "Search") || !strings.Contains(parseInputMarkup, `value="desk"`) {
		parseT.Fatalf("unexpected public catalog input markup %q", parseInputMarkup)
	}
	parseSelectMarkup := renderAtlasMarkupForTest(parseT, publicCatalogSelect("sort", "Sort", "", []optionItem{{Value: "", Label: "Featured"}, {Value: "warehouse", Label: "Warehouse"}}))
	if !strings.Contains(parseSelectMarkup, "<select") || !strings.Contains(parseSelectMarkup, "Featured") {
		parseT.Fatalf("unexpected public catalog select markup %q", parseSelectMarkup)
	}
	parseNumberMarkup := renderAtlasMarkupForTest(parseT, cmsNumberInput("quantity", "Quantity", "12"))
	if !strings.Contains(parseNumberMarkup, `type="number"`) || !strings.Contains(parseNumberMarkup, "Quantity") {
		parseT.Fatalf("unexpected cms number input markup %q", parseNumberMarkup)
	}
	parseTextareaMarkup := renderAtlasMarkupForTest(parseT, cmsTextarea("details", "Details", "Atlas detail copy"))
	if !strings.Contains(parseTextareaMarkup, "<textarea") || !strings.Contains(parseTextareaMarkup, "Atlas detail copy") {
		parseT.Fatalf("unexpected cms textarea markup %q", parseTextareaMarkup)
	}

	if _, _, parseErr := submitPublicComment(parseProduct.Slug, publicCommentFormState{
		AuthorName: "Atlas Buyer",
		Reaction:   "up",
		Subject:    "Setup confidence",
		Body:       "Delivery details answered the main questions.",
	}, "atlas-csrf"); parseErr == nil || !strings.Contains(parseErr.Error(), "fetch API unavailable in this environment") {
		// The expected text moved from an Atlas-local string to the framework's own
		// message: native atlasFetch now delegates to fetch.Fetch instead of
		// hand-writing "fetch unavailable in native atlas build". One source of
		// truth for "there is no browser fetch here" - and if native HTTP ever
		// becomes supported, Atlas stops claiming otherwise for free.
		parseT.Fatalf("expected native submitPublicComment to fail through atlasFetch, got %v", parseErr)
	}

	if _, parseErr := fetchPublicProductComments(context.Background(), parseProduct.Slug); parseErr == nil {
		parseT.Fatal("expected fetchPublicProductComments to fail in native tests")
	}
	if _, parseErr := fetchPublicRelatedProducts(context.Background(), parseProduct.Slug); parseErr == nil {
		parseT.Fatal("expected fetchPublicRelatedProducts to fail in native tests")
	}
}
