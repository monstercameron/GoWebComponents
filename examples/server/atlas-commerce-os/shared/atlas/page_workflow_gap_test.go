package atlas

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

// TestAtlasNestedPanelAndRouteHelperBranches covers nested panel wrappers and route helper output branches.
func TestAtlasNestedPanelAndRouteHelperBranches(buildT *testing.T) {
	buildWarehousePayload := samplePayloadForRoute(RouteWarehouseDetail, warehouseInventoryDetailPage{
		Summary:   sampleSummary("Facility detail"),
		Warehouse: sampleWarehouseOpsRecords()[0],
		Inventory: sampleInventoryRows(),
		Orders:    samplePurchaseOrders(),
		Filters: map[string]string{
			"status": "promise_risk",
		},
	}, nil)
	buildWarehouseMarkup := renderAtlasMarkupForTest(buildT, warehouseOpsNestedPanelNode(buildWarehousePayload))
	for _, buildNeedle := range []string{
		"Warehouse items",
		"Urgent actions",
		"Add warehouse item",
	} {
		if !strings.Contains(buildWarehouseMarkup, buildNeedle) {
			buildT.Fatalf("expected warehouse nested panel markup to contain %q", buildNeedle)
		}
	}

	buildWarehouseContentMarkup := renderAtlasMarkupForTest(buildT, warehouseOpsDetailContent(buildWarehousePayload))
	if !strings.Contains(buildWarehouseContentMarkup, "Warehouse items") {
		buildT.Fatalf("expected warehouse detail content markup to contain warehouse table, got %q", buildWarehouseContentMarkup)
	}

	buildOrderPage := buildAtlasPurchaseOrderDetailPage()
	buildOrderPayload := samplePayloadForRoute(RoutePurchaseOrderDetail, buildOrderPage, nil)
	buildOrderMarkup := renderAtlasMarkupForTest(buildT, purchaseOrdersNestedPanelNode(buildOrderPayload))
	for _, buildNeedle := range []string{
		buildOrderPage.Order.ID,
		"Purchase-order route refresh",
		"Update order status",
	} {
		if !strings.Contains(buildOrderMarkup, buildNeedle) {
			buildT.Fatalf("expected purchase-order nested panel markup to contain %q", buildNeedle)
		}
	}

	buildRevalidationMarkup := renderAtlasMarkupForTest(buildT, routeRevalidationCard("Reload order", "Reload the current order data."))
	for _, buildNeedle := range []string{
		"Reload order",
		"Reload the current order data.",
		"Refresh route data",
	} {
		if !strings.Contains(buildRevalidationMarkup, buildNeedle) {
			buildT.Fatalf("expected route revalidation markup to contain %q", buildNeedle)
		}
	}

	if settingsPanelNode(samplePayloadForRoute(RouteSettings, settingsPage{}, nil)) != nil {
		buildT.Fatal("expected settings root route to avoid nested settings panel")
	}

	buildLocaleMarkup := renderAtlasMarkupForTest(buildT, SettingsNestedPanel(samplePayloadForRoute(RouteSettingsLocale, settingsPage{}, nil)))
	for _, buildNeedle := range []string{
		"Locale defaults",
		"Current locale",
		"Supported locales",
	} {
		if !strings.Contains(buildLocaleMarkup, buildNeedle) {
			buildT.Fatalf("expected locale panel markup to contain %q", buildNeedle)
		}
	}

	buildWorkspaceMarkup := renderAtlasMarkupForTest(buildT, settingsPanelNode(samplePayloadForRoute(RouteSettingsWorkspaceDefaults, settingsPage{}, nil)))
	for _, buildNeedle := range []string{
		"Workspace defaults",
		"Default warehouse",
		"Low stock triage",
	} {
		if !strings.Contains(buildWorkspaceMarkup, buildNeedle) {
			buildT.Fatalf("expected workspace-defaults panel markup to contain %q", buildNeedle)
		}
	}

	if buildHref := commentsModerationHref(""); buildHref != RouteComments {
		buildT.Fatalf("commentsModerationHref(empty) = %q, want %q", buildHref, RouteComments)
	}
	if buildHref := commentsModerationHref("flagged"); buildHref != RouteComments+"/moderation/flagged" {
		buildT.Fatalf("commentsModerationHref(flagged) = %q", buildHref)
	}
	if buildLabel := formatCommentStatusLabel(""); buildLabel != "Unknown" {
		buildT.Fatalf("formatCommentStatusLabel(empty) = %q", buildLabel)
	}
	if buildLabel := formatCommentStatusLabel("in_review"); buildLabel != "in review" {
		buildT.Fatalf("formatCommentStatusLabel(in_review) = %q", buildLabel)
	}
}

// TestAtlasWorkflowAndPreviewBranches covers workflow-state helpers, stats panels, and settings preview branches.
func TestAtlasWorkflowAndPreviewBranches(buildT *testing.T) {
	buildReviewState := normalizeReceivingResolutionWorkflowState(receivingResolutionWorkflowState{
		Status:             "review",
		DiscrepancySummary: "Dock discrepancy is under review.",
	})
	if buildReviewState.Stage != "discrepancy-review" {
		buildT.Fatalf("normalizeReceivingResolutionWorkflowState(review) = %#v", buildReviewState)
	}

	buildNeedsNoteState := reduceReceivingResolutionWorkflowState(receivingResolutionWorkflowState{}, receivingResolutionWorkflowAction{Field: "status", Value: "closed"})
	if buildNeedsNoteState.Stage != "closeout-needs-note" || buildNeedsNoteState.LastEditedField != "status" {
		buildT.Fatalf("reduceReceivingResolutionWorkflowState(status) = %#v", buildNeedsNoteState)
	}

	buildReadyState := reduceReceivingResolutionWorkflowState(buildNeedsNoteState, receivingResolutionWorkflowAction{Field: "discrepancy_summary", Value: "Supplier short documented."})
	if buildReadyState.Stage != "closeout-ready" || buildReadyState.LastEditedField != "discrepancy_summary" {
		buildT.Fatalf("reduceReceivingResolutionWorkflowState(discrepancy_summary) = %#v", buildReadyState)
	}

	buildNeedsNoteMarkup := renderAtlasMarkupForTest(buildT, receivingResolutionSummaryCard(buildNeedsNoteState))
	for _, buildNeedle := range []string{
		"Closeout needs note",
		"Last updated field: status.",
	} {
		if !strings.Contains(buildNeedsNoteMarkup, buildNeedle) {
			buildT.Fatalf("expected receiving summary markup to contain %q", buildNeedle)
		}
	}

	buildReviewMarkup := renderAtlasMarkupForTest(buildT, receivingResolutionSummaryCard(buildReviewState))
	if !strings.Contains(buildReviewMarkup, "Discrepancy review") {
		buildT.Fatalf("expected discrepancy-review summary markup, got %q", buildReviewMarkup)
	}

	buildReceivingForm := ui.UseForm(receivingFormState{
		Status:             "closed",
		DiscrepancySummary: "Dock note",
	})
	buildReceivingWorkflow := ui.UseReducer(reduceReceivingResolutionWorkflowState, buildReadyState)
	buildInputMarkup := renderAtlasMarkupForTest(buildT, receivingWorkflowInputWithValue("status", "Status", "closed", "Status", buildReceivingForm, buildReceivingWorkflow))
	if !strings.Contains(buildInputMarkup, `name="status"`) || !strings.Contains(buildInputMarkup, `value="closed"`) {
		buildT.Fatalf("unexpected receiving workflow input markup %q", buildInputMarkup)
	}

	buildTextareaMarkup := renderAtlasMarkupForTest(buildT, receivingWorkflowTextareaWithValue("discrepancy_summary", "Discrepancy summary", "Dock note", "DiscrepancySummary", buildReceivingForm, buildReceivingWorkflow))
	if !strings.Contains(buildTextareaMarkup, "Dock note") || !strings.Contains(buildTextareaMarkup, "Discrepancy summary") {
		buildT.Fatalf("unexpected receiving workflow textarea markup %q", buildTextareaMarkup)
	}

	buildOrderStatsMarkup := renderAtlasMarkupForTest(buildT, purchaseOrderDetailStatsContent(buildAtlasPurchaseOrderDetailPage(), true, true, "panel lagged", ui.Handler{}))
	for _, buildNeedle := range []string{
		"Refreshing panel...",
		"Refreshing the purchase-order side panel while the current snapshot stays visible.",
		"panel lagged",
	} {
		if !strings.Contains(buildOrderStatsMarkup, buildNeedle) {
			buildT.Fatalf("expected purchase-order stats markup to contain %q", buildNeedle)
		}
	}

	buildReceivingStatsMarkup := renderAtlasMarkupForTest(buildT, receivingDetailStatsContent(buildAtlasReceivingDetailPage(), true, true, "receiving lagged", ui.Handler{}))
	for _, buildNeedle := range []string{
		"Refreshing panel...",
		"Refreshing the receiving side panel while the current snapshot stays visible.",
		"receiving lagged",
	} {
		if !strings.Contains(buildReceivingStatsMarkup, buildNeedle) {
			buildT.Fatalf("expected receiving stats markup to contain %q", buildNeedle)
		}
	}

	buildTransferMarkup := renderAtlasMarkupForTest(buildT, savedViewTransferCard(samplePayloadForRoute(RouteSettings, settingsPage{}, nil)))
	for _, buildNeedle := range []string{
		"Export saved views",
		"Import saved views",
		"Low stock triage",
	} {
		if !strings.Contains(buildTransferMarkup, buildNeedle) {
			buildT.Fatalf("expected saved-view transfer markup to contain %q", buildNeedle)
		}
	}

	buildComfortableMarkup := renderAtlasMarkupForTest(buildT, preferenceDensityPreviewCard("comfortable", false))
	if !strings.Contains(buildComfortableMarkup, "Comfortable") || !strings.Contains(buildComfortableMarkup, "increases whitespace") {
		buildT.Fatalf("unexpected comfortable density preview markup %q", buildComfortableMarkup)
	}

	buildPendingMarkup := renderAtlasMarkupForTest(buildT, preferenceDensityPreviewCard("", true))
	if !strings.Contains(buildPendingMarkup, "Compact") || !strings.Contains(buildPendingMarkup, "Applying the next density preview") {
		buildT.Fatalf("unexpected pending density preview markup %q", buildPendingMarkup)
	}
}

// TestAtlasPublicActionAndMapBranches covers public action-card and warehouse-map branches.
func TestAtlasPublicActionAndMapBranches(buildT *testing.T) {
	buildAvailableProduct := sampleProductCards()[0]
	buildAvailableProduct.Status = "available"
	buildAvailableMarkup := renderAtlasMarkupForTest(buildT, productPrimaryActionForm(buildAvailableProduct, Payload{CSRF: "atlas-csrf"}))
	if !strings.Contains(buildAvailableMarkup, "Request pricing") || !strings.Contains(buildAvailableMarkup, "/quote-requests") {
		buildT.Fatalf("unexpected available product action markup %q", buildAvailableMarkup)
	}

	buildLowStockProduct := sampleProductCards()[0]
	buildLowStockProduct.Status = "low_stock"
	buildLowStockMarkup := renderAtlasMarkupForTest(buildT, productPrimaryActionForm(buildLowStockProduct, Payload{CSRF: "atlas-csrf"}))
	if !strings.Contains(buildLowStockMarkup, "Reserve availability") || !strings.Contains(buildLowStockMarkup, "/restock-requests") {
		buildT.Fatalf("unexpected low-stock product action markup %q", buildLowStockMarkup)
	}

	buildNotifyProduct := sampleProductCards()[0]
	buildNotifyProduct.Status = "discontinued"
	buildNotifyMarkup := renderAtlasMarkupForTest(buildT, productPrimaryActionForm(buildNotifyProduct, Payload{CSRF: "atlas-csrf"}))
	if !strings.Contains(buildNotifyMarkup, "Notify me when available") || !strings.Contains(buildNotifyMarkup, "Notify me") {
		buildT.Fatalf("unexpected notify product action markup %q", buildNotifyMarkup)
	}

	buildRegionalMarkup := renderAtlasMarkupForTest(buildT, productSecondaryActionCard(buildAvailableProduct))
	if !strings.Contains(buildRegionalMarkup, "Check delivery for your region") || !strings.Contains(buildRegionalMarkup, "See regional delivery options") {
		buildT.Fatalf("unexpected regional secondary action markup %q", buildRegionalMarkup)
	}

	buildAlternativeMarkup := renderAtlasMarkupForTest(buildT, productSecondaryActionCard(buildNotifyProduct))
	if !strings.Contains(buildAlternativeMarkup, "See similar options") || !strings.Contains(buildAlternativeMarkup, "Browse alternatives") {
		buildT.Fatalf("unexpected alternative secondary action markup %q", buildAlternativeMarkup)
	}

	buildCriticalFill, buildCriticalStroke := formatWarehouseMapNodeTone(warehouseOpsRecord{RiskCount: 4})
	if buildCriticalFill != "fill-rose-400/25" || buildCriticalStroke != "stroke-rose-300" {
		buildT.Fatalf("formatWarehouseMapNodeTone(critical) = %q %q", buildCriticalFill, buildCriticalStroke)
	}

	buildWatchFill, buildWatchStroke := formatWarehouseMapNodeTone(warehouseOpsRecord{RiskCount: 1})
	if buildWatchFill != "fill-amber-300/25" || buildWatchStroke != "stroke-amber-200" {
		buildT.Fatalf("formatWarehouseMapNodeTone(watch) = %q %q", buildWatchFill, buildWatchStroke)
	}

	buildBalancedFill, buildBalancedStroke := formatWarehouseMapNodeTone(warehouseOpsRecord{})
	if buildBalancedFill != "fill-cyan-300/20" || buildBalancedStroke != "stroke-cyan-200" {
		buildT.Fatalf("formatWarehouseMapNodeTone(default) = %q %q", buildBalancedFill, buildBalancedStroke)
	}

	buildEmptyMapMarkup := renderAtlasMarkupForTest(buildT, renderWarehouseMapCard(nil))
	if !strings.Contains(buildEmptyMapMarkup, "No warehouse geometry to render.") {
		buildT.Fatalf("unexpected empty warehouse map markup %q", buildEmptyMapMarkup)
	}

	buildMapMarkup := renderAtlasMarkupForTest(buildT, renderWarehouseMapCard(sampleWarehouseOpsRecords()))
	for _, buildNeedle := range []string{
		"Warehouse transfer map",
		"Atlas warehouse transfer map centered on",
		sampleWarehouseOpsRecords()[0].Name,
	} {
		if !strings.Contains(buildMapMarkup, buildNeedle) {
			buildT.Fatalf("expected warehouse map markup to contain %q", buildNeedle)
		}
	}
}

// buildAtlasPurchaseOrderDetailPage builds a representative purchase-order detail page for direct render tests.
func buildAtlasPurchaseOrderDetailPage() purchaseOrderDetailPage {
	return purchaseOrderDetailPage{
		Order: samplePurchaseOrders()[0],
		Lines: []purchaseOrderLineRecord{
			{ID: "po-line-1", PurchaseOrderID: "po-1042", ProductSKU: "frame-desk", Quantity: 12},
		},
	}
}

// buildAtlasReceivingDetailPage builds a representative receiving detail page for direct render tests.
func buildAtlasReceivingDetailPage() receivingDetailPage {
	return receivingDetailPage{
		Session: sampleReceivingRecords()[0],
		Lines: []receivingLineRecord{
			{ID: "rcv-line-1", ReceivingSessionID: "rcv-illinois-001", ProductSKU: "frame-desk", ExpectedQuantity: 8, ActualQuantity: 7, DiscrepancyReason: "supplier short"},
		},
	}
}
