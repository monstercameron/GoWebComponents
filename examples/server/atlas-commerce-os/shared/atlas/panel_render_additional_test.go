package atlas

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// atlasNativeRenderMu serializes native renders across this package's tests, and
// every test render must go through renderAtlasNodeForTest to take it.
//
// WHY: the runtime tracks the component it is currently rendering in PACKAGE
// globals - currentFiber and currentFiberOwnerGoroutineID in
// internal/runtime/reconciler.go - which ui.RenderToString sets and restores
// around each component (internal/runtime/ssr.go: withSSRHookFiber). That makes
// native server rendering single-flight per PROCESS, not per call. Two
// t.Parallel() tests rendering at the same moment overwrite each other's current
// fiber, and the loser either reads another component's hook slots or trips
// GWC-RUNTIME-HOOK-THREADING.
//
// This never surfaced before because the native hook surface was stubbed: no
// render claimed a fiber, so concurrent renders could not collide. Making the
// hooks real exposed a genuine property of the runtime, so the tests hold a lock
// rather than paper over it. Everything in a test outside the render still runs
// in parallel.
var atlasNativeRenderMu sync.Mutex

// renderAtlasNodeForTest is the (markup, error) render entry point for this
// package's tests. Call it instead of ui.RenderToString directly so the render is
// serialized; see atlasNativeRenderMu.
func renderAtlasNodeForTest(parseNode ui.Node) (string, error) {
	atlasNativeRenderMu.Lock()
	defer atlasNativeRenderMu.Unlock()
	return ui.RenderToString(parseNode)
}

// renderAtlasMarkupForTest renders one already-built Atlas node to markup.
//
// Use it only for node BUILDERS - helpers that assemble markup from plain
// arguments and call no hooks. If the helper calls a hook (directly or through
// something like currentShellPresentationState -> useAtlasAtom), building the
// node at the call site runs that hook outside any render pass and panics with
// GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT. Use renderAtlasComponentForTest instead.
func renderAtlasMarkupForTest(parseT *testing.T, parseNode ui.Node) string {
	parseT.Helper()
	parseMarkup, parseErr := renderAtlasNodeForTest(parseNode)
	if parseErr != nil {
		parseT.Fatalf("RenderToString(): %v", parseErr)
	}
	return parseMarkup
}

// renderAtlasComponentForTest renders parseRender AS a component, so anything it
// calls runs on the render fiber the runtime installs for it.
//
// This is the test-side counterpart of what client/main.go must do:
// ui.CreateElement(component, props) rather than component(props). It exists as a
// closure-taking helper because many Atlas view functions take several arguments
// (payload plus a page struct) and so cannot be passed to ui.CreateElement's
// single-props form directly.
//
// Prefer this whenever the function under test touches state. It is the only
// shape in which a native render is evidence about the browser.
func renderAtlasComponentForTest(parseT *testing.T, parseRender func() ui.Node) string {
	parseT.Helper()
	parseMarkup, parseErr := renderAtlasNodeForTest(ui.CreateElement(parseRender))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(component): %v", parseErr)
	}
	return parseMarkup
}

// TestWarehouseOpsPanelsRenderWorkspaceAndItem verifies nested warehouse panel rendering paths.
func TestWarehouseOpsPanelsRenderWorkspaceAndItem(parseT *testing.T) {
	parseWarehouses := sampleWarehouseOpsRecords()
	parseInventory := sampleInventoryRows()
	parseOrders := samplePurchaseOrders()
	parseProducts := sampleProductAdminCards()

	parseDetailPayload := samplePayloadForRoute(RouteWarehouseDetail, warehouseInventoryDetailPage{
		Summary:   sampleSummary("Facility detail"),
		Warehouse: parseWarehouses[0],
		Inventory: parseInventory,
		Orders:    parseOrders,
		Filters: map[string]string{
			"status": "promise_risk",
		},
	}, nil)
	parseDetailMarkup := renderAtlasMarkupForTest(parseT, WarehouseOpsDetailPanel(parseDetailPayload))
	for _, parseNeedle := range []string{
		parseWarehouses[0].Name,
		"Items in this hub",
		"Needs action",
	} {
		if !strings.Contains(parseDetailMarkup, parseNeedle) {
			parseT.Fatalf("expected warehouse detail markup to contain %q", parseNeedle)
		}
	}

	parseItemPayload := samplePayloadForRoute(RouteWarehouseItemDetail, warehouseInventoryItemDetailPage{
		Warehouse: parseWarehouses[0],
		Item:      parseInventory[0],
		Product:   parseProducts[0],
		Network:   parseInventory,
		Orders:    parseOrders,
		Filters: map[string]string{
			"status": "all",
			"sort":   "available",
		},
	}, nil)
	parseItemMarkup := renderAtlasMarkupForTest(parseT, WarehouseOpsItemPanel(parseItemPayload))
	for _, parseNeedle := range []string{
		"Nested warehouse item workspace",
		parseProducts[0].Title,
		"Related replenishment orders",
	} {
		if !strings.Contains(parseItemMarkup, parseNeedle) {
			parseT.Fatalf("expected warehouse item markup to contain %q", parseNeedle)
		}
	}
}

// TestDetailPanelsAndStatsIslandsRenderFallbacks verifies direct detail content and fallback side-panel helpers.
func TestDetailPanelsAndStatsIslandsRenderFallbacks(parseT *testing.T) {
	parseOrderPage := purchaseOrderDetailPage{
		Order: samplePurchaseOrders()[0],
		Lines: []purchaseOrderLineRecord{
			{ID: "po-line-1", PurchaseOrderID: "po-1042", ProductSKU: "frame-desk", Quantity: 12, ETA: "Thu 09:30", Status: "submitted"},
		},
	}
	parseOrderMarkup := renderAtlasComponentForTest(parseT, func() ui.Node {
		return purchaseOrderDetailContent(samplePayloadForRoute(RoutePurchaseOrderDetail, parseOrderPage, nil))
	})
	for _, parseNeedle := range []string{
		parseOrderPage.Order.VendorName,
		"Back to PO table",
		"Line-item context",
	} {
		if !strings.Contains(parseOrderMarkup, parseNeedle) {
			parseT.Fatalf("expected purchase-order detail markup to contain %q", parseNeedle)
		}
	}

	parseOrderStatsMarkup := renderAtlasComponentForTest(parseT, func() ui.Node {
		return purchaseOrderDetailStatsIsland(Payload{}, parseOrderPage)
	})
	if !strings.Contains(parseOrderStatsMarkup, parseOrderPage.Order.WarehouseName) || !strings.Contains(parseOrderStatsMarkup, parseOrderPage.Order.ETA) {
		parseT.Fatalf("unexpected purchase-order stats markup %q", parseOrderStatsMarkup)
	}

	parseReceivingPage := receivingDetailPage{
		Session: sampleReceivingRecords()[0],
		Lines: []receivingLineRecord{
			{ID: "rcv-line-1", ReceivingSessionID: "rcv-illinois-001", ProductSKU: "frame-desk", ExpectedQuantity: 8, ActualQuantity: 7, DiscrepancyReason: "supplier short"},
		},
	}
	parseReceivingStatsMarkup := renderAtlasComponentForTest(parseT, func() ui.Node {
		return receivingDetailStatsIsland(Payload{}, parseReceivingPage)
	})
	if !strings.Contains(parseReceivingStatsMarkup, parseReceivingPage.Session.WarehouseID) || !strings.Contains(parseReceivingStatsMarkup, parseReceivingPage.Session.Status) {
		parseT.Fatalf("unexpected receiving stats markup %q", parseReceivingStatsMarkup)
	}

	parseErrorMarkup := renderAtlasMarkupForTest(parseT, detailRailErrorIsland("Purchase-order side panel", errors.New("panel fetch failed"), func() {}, ui.Text("fallback detail")))
	for _, parseNeedle := range []string{
		"Purchase-order side panel",
		"panel fetch failed",
		"Retry panel",
		"fallback detail",
	} {
		if !strings.Contains(parseErrorMarkup, parseNeedle) {
			parseT.Fatalf("expected detail rail error markup to contain %q", parseNeedle)
		}
	}
}

// TestCommentsAndAvailabilityPanelsRenderVariants verifies route-local comment panels and public action variants.
func TestCommentsAndAvailabilityPanelsRenderVariants(parseT *testing.T) {
	parseComments := sampleCommentRecords()

	parseModerationMarkup := renderAtlasMarkupForTest(parseT, CommentsNestedPanel(samplePayloadForRoute(RouteComments+"/moderation/pending", commentList{
		Summary: sampleSummary("Buyer inbox moderation"),
		Items:   parseComments,
	}, nil)))
	for _, parseNeedle := range []string{
		"Moderation queue",
		"Review bulk action",
		"Back to full inbox",
	} {
		if !strings.Contains(parseModerationMarkup, parseNeedle) {
			parseT.Fatalf("expected moderation panel markup to contain %q", parseNeedle)
		}
	}

	parseRecordMarkup := renderAtlasMarkupForTest(parseT, CommentsNestedPanel(samplePayloadForRoute(RouteComments+"/"+parseComments[0].ID, commentList{
		Summary: sampleSummary("Buyer inbox detail"),
		Items:   parseComments,
	}, nil)))
	for _, parseNeedle := range []string{
		"Selected buyer record",
		parseComments[0].Subject,
		"Open status filter",
	} {
		if !strings.Contains(parseRecordMarkup, parseNeedle) {
			parseT.Fatalf("expected record panel markup to contain %q", parseNeedle)
		}
	}

	parseAvailableMarkup := renderAtlasMarkupForTest(parseT, availabilityPrimaryActionForm(availabilityPage{
		Warehouse: sampleWarehouseCards()[0],
		Product:   sampleProductCards()[0],
		Available: 2,
		Inbound:   0,
	}, Payload{CSRF: "csrf-token"}))
	if !strings.Contains(parseAvailableMarkup, "Request pricing for this region") || !strings.Contains(parseAvailableMarkup, "Start quote") {
		parseT.Fatalf("unexpected available action markup %q", parseAvailableMarkup)
	}

	parseInboundMarkup := renderAtlasMarkupForTest(parseT, availabilityPrimaryActionForm(availabilityPage{
		Warehouse: sampleWarehouseCards()[0],
		Product:   sampleProductCards()[0],
		Available: 0,
		Inbound:   3,
	}, Payload{CSRF: "csrf-token"}))
	if !strings.Contains(parseInboundMarkup, "Reserve upcoming availability") || !strings.Contains(parseInboundMarkup, "Reserve availability") {
		parseT.Fatalf("unexpected inbound action markup %q", parseInboundMarkup)
	}

	parseNotifyMarkup := renderAtlasMarkupForTest(parseT, availabilityPrimaryActionForm(availabilityPage{
		Warehouse: sampleWarehouseCards()[0],
		Product:   sampleProductCards()[0],
		Available: 0,
		Inbound:   0,
	}, Payload{CSRF: "csrf-token"}))
	if !strings.Contains(parseNotifyMarkup, "Notify me when available") || !strings.Contains(parseNotifyMarkup, "Notify me") {
		parseT.Fatalf("unexpected notify action markup %q", parseNotifyMarkup)
	}
}

// TestSavedViewValidationAndSummaryHelpersRender verifies worker-state cards and list-node render helpers.
func TestSavedViewValidationAndSummaryHelpersRender(parseT *testing.T) {
	parseErrorMarkup := renderAtlasMarkupForTest(parseT, savedViewImportValidationCard(atlasWorkerTaskState[savedViewImportValidationProgress, savedViewImportValidationResult]{
		Error: errors.New("worker crashed"),
	}))
	if !strings.Contains(parseErrorMarkup, "Worker validation failed") || !strings.Contains(parseErrorMarkup, "worker crashed") {
		parseT.Fatalf("unexpected worker error markup %q", parseErrorMarkup)
	}

	parseRunningMarkup := renderAtlasMarkupForTest(parseT, savedViewImportValidationCard(atlasWorkerTaskState[savedViewImportValidationProgress, savedViewImportValidationResult]{
		Running:       true,
		ProgressReady: true,
		Progress: savedViewImportValidationProgress{
			Percent: 40,
			Stage:   "reading payload",
		},
	}))
	if !strings.Contains(parseRunningMarkup, "40% complete") || !strings.Contains(parseRunningMarkup, "reading payload") {
		parseT.Fatalf("unexpected worker running markup %q", parseRunningMarkup)
	}

	parseReadyMarkup := renderAtlasMarkupForTest(parseT, savedViewImportValidationCard(atlasWorkerTaskState[savedViewImportValidationProgress, savedViewImportValidationResult]{
		Ready: true,
		Value: savedViewImportValidationResult{
			Valid:        false,
			ItemCount:    3,
			ScopeCount:   2,
			Scopes:       []string{"inventory", "warehouse"},
			Preview:      []string{"Low stock triage"},
			InvalidCount: 1,
			Warnings:     []string{"Unknown density value"},
		},
	}))
	for _, parseNeedle := range []string{
		"Payload needs fixes before import.",
		"Low stock triage",
		"Unknown density value",
	} {
		if !strings.Contains(parseReadyMarkup, parseNeedle) {
			parseT.Fatalf("expected worker ready markup to contain %q", parseNeedle)
		}
	}

	parseIdleMarkup := renderAtlasMarkupForTest(parseT, savedViewImportValidationCard(atlasWorkerTaskState[savedViewImportValidationProgress, savedViewImportValidationResult]{}))
	if !strings.Contains(parseIdleMarkup, "Atlas will validate the saved-view import payload") {
		parseT.Fatalf("unexpected idle worker markup %q", parseIdleMarkup)
	}

	parseCommentMarkup := renderAtlasMarkupForTest(parseT, ui.Fragment(commentNodes(sampleCommentRecords())...))
	if !strings.Contains(parseCommentMarkup, sampleCommentRecords()[0].Subject) || !strings.Contains(parseCommentMarkup, sampleCommentRecords()[1].AuthorName) {
		parseT.Fatalf("unexpected comment nodes markup %q", parseCommentMarkup)
	}

	parseTransferMarkup := renderAtlasMarkupForTest(parseT, ui.Fragment(transferNodes(sampleTransferRecords())...))
	if !strings.Contains(parseTransferMarkup, sampleTransferRecords()[0].SourceWarehouseID) || !strings.Contains(parseTransferMarkup, sampleTransferRecords()[1].Reason) {
		parseT.Fatalf("unexpected transfer nodes markup %q", parseTransferMarkup)
	}

	parseReceivingMarkup := renderAtlasMarkupForTest(parseT, ui.Fragment(receivingNodes(sampleReceivingRecords())...))
	if !strings.Contains(parseReceivingMarkup, sampleReceivingRecords()[0].ID) || !strings.Contains(parseReceivingMarkup, sampleReceivingRecords()[1].DiscrepancySummary) {
		parseT.Fatalf("unexpected receiving nodes markup %q", parseReceivingMarkup)
	}

	parseResolutionMarkup := renderAtlasMarkupForTest(parseT, receivingResolutionSummaryCard(normalizeReceivingResolutionWorkflowState(receivingResolutionWorkflowState{
		Status:             "closed",
		DiscrepancySummary: "Supplier short documented.",
		LastEditedField:    "discrepancy_summary",
	})))
	for _, parseNeedle := range []string{
		"Closeout ready",
		"Last updated field: discrepancy summary.",
	} {
		if !strings.Contains(parseResolutionMarkup, parseNeedle) {
			parseT.Fatalf("expected resolution markup to contain %q", parseNeedle)
		}
	}
}
