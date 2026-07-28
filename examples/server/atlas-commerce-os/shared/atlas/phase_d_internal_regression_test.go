package atlas

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func TestPhaseDInternalRoutesRenderStableShellAndWorkflows(parseT *testing.T) {
	parseInventoryRows := sampleInventoryRows()
	parseProducts := sampleProductAdminCards()
	parseWarehouses := sampleWarehouseOpsRecords()
	parseComments := sampleCommentRecords()
	parseTransfers := sampleTransferRecords()
	parseOrders := samplePurchaseOrders()
	parseReceiving := sampleReceivingRecords()

	parseCases := []struct {
		name       string
		path       string
		page       any
		wantTokens []string
	}{
		{
			name: "dashboard", path: RouteDashboard,
			page: dashboardPage{Summary: sampleSummary("Dashboard pulse"), Alerts: 3, Transfers: parseTransfers, Receiving: parseReceiving, Comments: parseComments, Orders: parseOrders},
			// "Action cluster" named the LAYOUT device, and the four shortcuts under it were
			// numbered ACTION 1-4 as if they were a sequence. Both are gone; the durable
			// contract is that the dashboard still offers the shortcut and still names the
			// queue that needs attention.
			wantTokens: []string{"Quick actions", "Open low-stock lanes", "Pending comments"},
		},
		{
			name: "products", path: "/app/products",
			page:       productCMSPageData{Items: parseProducts, Total: len(parseProducts), Editable: true},
			wantTokens: []string{"Product merchandising flows", "Create product", "Edit product"},
		},
		{
			name: "inventory", path: RouteInventory,
			page:       inventoryCMSPage{Summary: sampleSummary("Inventory shell"), Items: parseInventoryRows},
			wantTokens: []string{"Inventory", "Saved views", "Promise"},
		},
		{
			name: "sku-detail", path: RouteSKUDetail,
			page:       inventoryDetailPage{SKU: "frame-desk", Title: "Frame Desk", Rows: parseInventoryRows},
			wantTokens: []string{"Frame Desk", "On hand by hub", "Flagged lanes"},
		},
		{
			name: "warehouse-ops", path: RouteWarehouseOps,
			page:       warehouseOpsList{Summary: sampleSummary("Warehouse operations"), Items: parseWarehouses},
			wantTokens: []string{"Warehouse operations", "Facility table", "Open a roster"},
		},
		{
			name: "warehouse-detail", path: RouteWarehouseDetail,
			page:       warehouseInventoryDetailPage{Summary: sampleSummary("Facility detail"), Warehouse: parseWarehouses[0], Inventory: parseInventoryRows, Orders: parseOrders},
			wantTokens: []string{"Facility detail", "Items in this hub", "Facility actions"},
		},
		{
			name: "warehouse-item", path: RouteWarehouseItemDetail,
			page:       warehouseInventoryItemDetailPage{Warehouse: parseWarehouses[0], Item: parseInventoryRows[0], Product: parseProducts[0], Network: parseInventoryRows, Orders: parseOrders},
			wantTokens: []string{"Warehouse item", "Warehouse item table", "Update marketing copy"},
		},
		{
			name: "transfers", path: RouteTransfers,
			page:       transferList{Items: parseTransfers},
			wantTokens: []string{"Transfers", "Create transfer", "Lanes in flight"},
		},
		{
			name: "transfer-detail", path: RouteTransferDetail,
			page:       transferDetailPage{Transfer: parseTransfers[0], Lines: []transferLineRecord{{ID: "tr-line-1", TransferID: parseTransfers[0].ID, ProductSKU: "frame-desk", Quantity: 5}}},
			wantTokens: []string{"Transfer workspace", "Transfer lines", "Back to transfers"},
		},
		{
			name: "purchase-orders", path: RoutePurchaseOrders,
			page:       purchaseOrderList{Summary: sampleSummary("Purchase-order watch"), Items: parseOrders},
			wantTokens: []string{"Vendor order table", "Orders on the way", "po-1042"},
		},
		{
			name: "purchase-order-detail", path: RoutePurchaseOrderDetail,
			page:       purchaseOrderDetailPage{Order: parseOrders[0], Lines: []purchaseOrderLineRecord{{ID: "po-line-1", PurchaseOrderID: parseOrders[0].ID, ProductSKU: "frame-desk", Quantity: 12, ETA: "Thu 09:30", Status: "submitted"}}},
			wantTokens: []string{"Purchase-order workspace", "What is on the order", "Line-item context"},
		},
		{
			name: "receiving", path: RouteReceiving,
			page:       receivingList{Items: parseReceiving},
			wantTokens: []string{"Receiving table", "Sessions to close", "rcv-illinois-001"},
		},
		{
			name: "receiving-detail", path: RouteReceivingSessionDetail,
			page:       receivingDetailPage{Session: parseReceiving[0], Lines: []receivingLineRecord{{ID: "rcv-line-1", ReceivingSessionID: parseReceiving[0].ID, ProductSKU: "frame-desk", ExpectedQuantity: 8, ActualQuantity: 7, DiscrepancyReason: "supplier short"}}},
			wantTokens: []string{"Receiving", "Discrepancy", "Reconcile"},
		},
		{
			name: "comments", path: RouteComments,
			page:       commentList{Summary: sampleSummary("Buyer inbox"), Items: parseComments},
			wantTokens: []string{"Buyer inbox", "moderation", "Bulk"},
		},
		{
			name: "comments-moderation", path: RouteCommentsModeration,
			page:       commentList{Summary: sampleSummary("Buyer inbox moderation"), Items: parseComments},
			wantTokens: []string{"Moderation", "Back to full inbox", "Approve"},
		},
		{
			name: "settings", path: RouteSettings,
			page:       settingsPage{Summary: sampleSummary("Shell settings")},
			wantTokens: []string{"Settings sub-routes", "Theme", "Saved-view"},
		},
	}

	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT2 *testing.T) {
			parseMarkup := renderAtlasComponentForTest(parseT2, func() ui.Node {
				return App(samplePayloadForRoute(parseCase.path, parseCase.page, nil))
			})
			// The shell contract is now the console RAIL, not the old top navigation.
			// "Workspace nav" was the label on the mobile drawer button that opened a
			// duplicate copy of the nav; design.ConsoleRail replaces both the four-box
			// desktop nav and that drawer with one element that restacks itself, so the
			// durable assertions are the rail's navigation landmark and its plate. Every
			// route label below is unchanged - the rail still lists all nine routes.
			for _, parseToken := range []string{"atlas-shell-root", `aria-label="Operator console"`, "Atlas Commerce OS console", "Dashboard", "Inventory", "Products", "Warehouses", "Transfers", "Purchase Orders", "Receiving", "Comments", "Settings"} {
				if !strings.Contains(parseMarkup, parseToken) {
					parseT2.Fatalf("internal shell for %s missing %q", parseCase.path, parseToken)
				}
			}
			for _, parseToken := range parseCase.wantTokens {
				if !strings.Contains(parseMarkup, parseToken) {
					parseT2.Fatalf("internal route %s missing workflow token %q", parseCase.path, parseToken)
				}
			}
		})
	}
}

func TestPhaseDLocalePreferenceRecoveryAndMotionContracts(parseT *testing.T) {
	parseLocalePayload := samplePayloadForRoute(RouteSettingsLocale, settingsPage{Summary: sampleSummary("Locale settings")}, nil)
	parseLocalePayload.Preferences = PreferencesState{Theme: "dark", Locale: "ar", Density: "compact", DefaultWarehouse: "new-jersey-hub"}
	parseLocalePayload.I18n = DefaultI18n("ar")
	parseLocalePayload.Theme = ThemeState{Mode: "dark"}

	parseLocaleMarkup := renderAtlasComponentForTest(parseT, func() ui.Node {
		return App(parseLocalePayload)
	})
	for _, parseToken := range []string{"Locale defaults", "Current locale", "ar", "Direction", "rtl", "Supported locales", "name=\"locale\"", "name=\"density\"", "name=\"default_warehouse_id\""} {
		if !strings.Contains(parseLocaleMarkup, parseToken) {
			parseT.Fatalf("locale settings markup missing %q", parseToken)
		}
	}

	parseRecoveryPayload := samplePayloadForRoute("/app/recovery", recoveryPage{
		Title:         "Atlas internal recovery",
		Message:       "The requested Atlas internal route is not part of the current demo route set.",
		RecoveryHref:  RouteDashboard,
		RecoveryLabel: "Back to dashboard",
		Detail:        "Return to the operator dashboard and keep the internal shell context intact.",
	}, nil)
	parseRecoveryPayload.Route.Screen = "recovery"
	parseRecoveryPayload.Route.Surface = "internal"
	parseRecoveryMarkup := renderAtlasComponentForTest(parseT, func() ui.Node {
		return App(parseRecoveryPayload)
	})
	for _, parseToken := range []string{"Atlas internal recovery", "Back to dashboard", `aria-label="Operator console"`, "atlas-shell-root"} {
		if !strings.Contains(parseRecoveryMarkup, parseToken) {
			parseT.Fatalf("internal recovery markup missing %q", parseToken)
		}
	}

	_, parseFile, _, parseOK := runtime.Caller(0)
	if !parseOK {
		parseT.Fatal("could not resolve test file path")
	}
	parseShellPath := filepath.Join(filepath.Dir(parseFile), "..", "..", "client", "atlas-commerce-os.html")
	parseShellBytes, parseErr := os.ReadFile(parseShellPath)
	if parseErr != nil {
		parseT.Fatalf("read Atlas shell template: %v", parseErr)
	}
	parseShell := string(parseShellBytes)

	// The shell template is now only what must exist BEFORE Go runs: the mount
	// point, the portal host, and the boot skeleton. Styling moved into the typed
	// CSS design system (shared/design), so this file no longer carries the RTL
	// and reduced-motion rules it used to.
	//
	// These two tokens are still asserted here because they are structural, not
	// stylistic: the overlay root is the DOM node portals mount into, and losing
	// it breaks every dialog and sheet with no compile error to warn you.
	for _, parseToken := range []string{`id="atlas-overlay-root"`, `data-atlas-portal-host="overlays"`} {
		if !strings.Contains(parseShell, parseToken) {
			parseT.Fatalf("Atlas shell template missing %q", parseToken)
		}
	}

	// The RTL and reduced-motion contracts this test used to assert as CSS text in
	// the template above have NOT been dropped — they moved. Asserting them where
	// they now live is the point of re-pointing rather than deleting: a test that
	// pins an implementation's ADDRESS fails when the implementation moves, which
	// tells you nothing about whether the behaviour survived.
	//
	// Reduced motion is emitted by the design system's own motion helper, and
	// shared/design has TestEveryTransitionCarriesAReducedMotionOverride, which is
	// a stronger guard than a substring check: it counts emitted transitions and
	// requires each to carry an override, so a new animated primitive cannot ship
	// without one.
	//
	// Direction is emitted by the server as a dir attribute on <html> driven by
	// Payload.I18n.Direction, covered by the server package's locale contract
	// tests. What this test can still usefully assert is that the payload carries
	// a direction at all, since a blank one silently yields an LTR document for an
	// RTL locale — which is exactly the defect that shipped.
	if parseDir := strings.TrimSpace(LocaleDirection("ar")); parseDir != "rtl" {
		parseT.Fatalf("Arabic must resolve to direction rtl so the document can set dir; got %q", parseDir)
	}
	if parseDir := strings.TrimSpace(LocaleDirection("en")); parseDir != "ltr" {
		parseT.Fatalf("English must resolve to direction ltr; got %q", parseDir)
	}
}

func TestPhaseDCachePayloadAndInteractionContracts(parseT *testing.T) {
	parseNotices := map[string]string{
		"product-updated":        "/app/products",
		"inventory-updated":      "/app/dashboard",
		"transfer-created":       "/app/dashboard",
		"purchase-order-created": "/app/dashboard",
		"receiving-reconciled":   "/app/dashboard",
		"comment-moderated":      "/app/dashboard",
		"preferences-saved":      "/app/dashboard",
		"saved-views-imported":   "/app/dashboard",
	}
	for parseNotice, parseWantPrefix := range parseNotices {
		parseT.Run(parseNotice, func(parseT2 *testing.T) {
			parseRoutePrefixes := MutationRoutePrefixes(RouteDashboard, parseNotice)
			if len(parseRoutePrefixes) == 0 || !containsAtlasPrefix(parseRoutePrefixes, parseWantPrefix) {
				parseT2.Fatalf("notice %q did not invalidate %s context: %#v", parseNotice, parseWantPrefix, parseRoutePrefixes)
			}
			parseRequestPrefixes := MutationRequestPrefixes(parseRoutePrefixes)
			if len(parseRequestPrefixes) == 0 {
				parseT2.Fatalf("notice %q did not map to request invalidation prefixes", parseNotice)
			}
		})
	}

	for _, parsePath := range []string{RouteDashboard, "/app/products", RouteInventory, RouteSKUDetail, RouteWarehouseOps, RouteWarehouseItemDetail, RouteTransfers, RouteTransferDetail, RoutePurchaseOrders, RoutePurchaseOrderDetail, RouteReceiving, RouteReceivingSessionDetail, RouteComments, RouteCommentsModeration, RouteSettings, RouteSettingsLocale} {
		if parseURL := StartupRequestURL(parsePath, nil); strings.TrimSpace(parseURL) == "" {
			parseT.Fatalf("internal route %q missing startup payload URL", parsePath)
		}
	}

	parsePayload := samplePayloadForRoute(RouteDashboard, dashboardPage{Summary: sampleSummary("Dashboard pulse"), Alerts: 2, Transfers: sampleTransferRecords(), Receiving: sampleReceivingRecords(), Comments: sampleCommentRecords(), Orders: samplePurchaseOrders()}, nil)
	parsePayload.Requests["dashboard"] = Request{Method: "GET", URL: "/api/app/dashboard", Status: 200, Data: map[string]any{"page": "duplicate-page", "alerts": []any{"one"}}}
	parseRoundTrip := PayloadFromSSRBootstrap(parsePayload.ToSSRBootstrap())
	if _, parseDuplicated := parseRoundTrip.Requests["dashboard"].Data["page"]; parseDuplicated {
		parseT.Fatalf("request payload kept duplicate route page data: %#v", parseRoundTrip.Requests["dashboard"].Data)
	}
	if _, parseKept := parseRoundTrip.Requests["dashboard"].Data["alerts"]; !parseKept {
		parseT.Fatalf("request payload lost non-duplicate panel data: %#v", parseRoundTrip.Requests["dashboard"].Data)
	}
}
