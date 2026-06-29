package atlas

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

// TestInventoryThresholdHistoryPanelRendersRouteOverlay verifies the SKU threshold workflow renders as a route-owned overlay sheet.
func TestInventoryThresholdHistoryPanelRendersRouteOverlay(parseT *testing.T) {
	parseT.Parallel()

	parseMarkup, parseErr := ui.RenderToString(inventoryThresholdHistoryPanel(
		inventoryThresholdHistoryPanelPage{
			SKU: "frame-desk",
			Items: []inventoryThresholdHistoryItem{
				{
					ID:           "th-1",
					ProductSKU:   "frame-desk",
					WarehouseID:  "new-jersey-hub",
					ReorderPoint: 18,
					SafetyStock:  9,
					ActorName:    "Ops lead",
					Summary:      "Threshold adjustment",
					Detail:       "Raised reorder protection.",
					CreatedAt:    "2026-03-25T10:00:00Z",
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
					Priority:                 "promise_risk",
					Reason:                   "Protect east-coast lane commitments.",
				},
			},
		},
		inventoryDetailPage{SKU: "frame-desk", Title: "Frame Desk"},
	))
	if parseErr != nil {
		parseT.Fatalf("inventoryThresholdHistoryPanel render failed: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "Route overlay") || !strings.Contains(parseMarkup, "Threshold history") {
		parseT.Fatalf("expected threshold overlay heading copy, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, `data-atlas-route-overlay="threshold-history"`) {
		parseT.Fatalf("expected threshold route-overlay marker, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "Transfer recommendations") {
		parseT.Fatalf("expected transfer recommendation section, got %q", parseMarkup)
	}
}

// TestAtlasConfirmationDialogRendersWorkflowVariants verifies transfer, receiving, and moderation confirmation dialogs mount with expected copy.
func TestAtlasConfirmationDialogRendersWorkflowVariants(parseT *testing.T) {
	parseT.Parallel()

	parseTransferMarkup, parseErr := ui.RenderToString(atlasConfirmationDialog(
		true,
		"atlas-transfer-confirm",
		"Confirm transfer plan",
		"Review transfer details before submit.",
		"Create transfer",
		nil,
		html.P(html.Props{}, html.Text("Transfer body")),
	))
	if parseErr != nil {
		parseT.Fatalf("transfer dialog render failed: %v", parseErr)
	}
	if !strings.Contains(parseTransferMarkup, "Confirm transfer plan") || !strings.Contains(parseTransferMarkup, "Create transfer") {
		parseT.Fatalf("expected transfer confirmation copy, got %q", parseTransferMarkup)
	}
	for _, parseNeedle := range []string{`id="atlas-transfer-confirm-title"`, `id="atlas-transfer-confirm-description"`, `id="atlas-transfer-confirm-confirm"`, "Cancel"} {
		if !strings.Contains(parseTransferMarkup, parseNeedle) {
			parseT.Fatalf("expected transfer confirmation markup to contain %q, got %q", parseNeedle, parseTransferMarkup)
		}
	}

	parseReceivingMarkup, parseErr := ui.RenderToString(atlasConfirmationDialog(
		true,
		"atlas-receiving-confirm",
		"Confirm receiving closeout",
		"Review discrepancy before closeout.",
		"Close session",
		nil,
		html.P(html.Props{}, html.Text("Receiving body")),
	))
	if parseErr != nil {
		parseT.Fatalf("receiving dialog render failed: %v", parseErr)
	}
	if !strings.Contains(parseReceivingMarkup, "Confirm receiving closeout") || !strings.Contains(parseReceivingMarkup, "Close session") {
		parseT.Fatalf("expected receiving confirmation copy, got %q", parseReceivingMarkup)
	}

	parseModerationMarkup, parseErr := ui.RenderToString(atlasConfirmationDialog(
		true,
		"atlas-comment-review-confirm",
		"Confirm buyer review",
		"Confirm moderation action before submit.",
		"Apply review",
		nil,
		html.P(html.Props{}, html.Text("Moderation body")),
	))
	if parseErr != nil {
		parseT.Fatalf("moderation dialog render failed: %v", parseErr)
	}
	if !strings.Contains(parseModerationMarkup, "Confirm buyer review") || !strings.Contains(parseModerationMarkup, "Apply review") {
		parseT.Fatalf("expected moderation confirmation copy, got %q", parseModerationMarkup)
	}
}

// TestAtlasDismissibleSheetRendersSideSheet verifies the shared side-sheet wrapper renders child content in open state.
func TestAtlasDismissibleSheetRendersSideSheet(parseT *testing.T) {
	parseT.Parallel()

	parseMarkup, parseErr := ui.RenderToString(atlasDismissibleSheet(
		true,
		"atlas-sheet",
		"atlas-sheet-title",
		"atlas-sheet-description",
		"#atlas-sheet-close",
		nil,
		html.Div(html.Props{},
			html.P(html.Props{ID: "atlas-sheet-title"}, html.Text("Sheet title")),
			html.P(html.Props{ID: "atlas-sheet-description"}, html.Text("Sheet description")),
			html.Button(html.Props{ID: "atlas-sheet-close", Type: "button"}, html.Text("Close")),
		),
	))
	if parseErr != nil {
		parseT.Fatalf("atlasDismissibleSheet render failed: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "Sheet title") || !strings.Contains(parseMarkup, "Sheet description") {
		parseT.Fatalf("expected side-sheet content in markup, got %q", parseMarkup)
	}
	for _, parseNeedle := range []string{`id="atlas-sheet-title"`, `id="atlas-sheet-description"`, `id="atlas-sheet-close"`} {
		if !strings.Contains(parseMarkup, parseNeedle) {
			parseT.Fatalf("expected side-sheet markup to contain %q, got %q", parseNeedle, parseMarkup)
		}
	}
}

func TestAtlasOverlayTargetUsesSharedPortalRoot(parseT *testing.T) {
	parseT.Parallel()

	parseTarget := atlasOverlayTarget()
	if parseTarget.Selector != "#"+atlasOverlayRootID {
		parseT.Fatalf("expected shared overlay portal target %q, got %q", "#"+atlasOverlayRootID, parseTarget.Selector)
	}
}
