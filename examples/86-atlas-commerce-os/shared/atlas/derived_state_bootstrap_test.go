package atlas

import (
	"net/url"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/shared/repository"
)

func TestPayloadBootstrapRoundTripAndRequestDedup(parseT *testing.T) {
	parsePayload := Payload{
		Route: RouteBootstrap{
			Path:      RouteCatalog,
			Query:     map[string][]string{"q": {"desk"}, "atlas_notice": {"saved"}},
			Surface:   "public",
			Screen:    "catalog",
			Title:     "Atlas Shop",
			Canonical: RouteCatalog,
		},
		Preferences: PreferencesState{Theme: "dark", Locale: "en", Density: "compact", DefaultWarehouse: "illinois-hub"},
		I18n:        I18nState{Locale: "en", Direction: "ltr"},
		Theme:       ThemeState{Mode: "dark"},
		Data: map[string]any{
			"page": map[string]any{"slug": "frame-desk"},
		},
		Requests: map[string]Request{
			"catalog": {
				Method: "GET",
				URL:    "/api/public/catalog",
				Status: 200,
				Data: map[string]any{
					"page":  "duplicate",
					"items": []any{"frame-desk"},
				},
			},
		},
		SavedViews: []SavedViewPayload{{Name: "Low stock triage", Scope: "inventory"}},
		CSRF:       "csrf-token",
		User:       &UserSession{ID: "u-1", DisplayName: "Atlas Demo", Role: "admin"},
	}

	parseBootstrap := parsePayload.ToSSRBootstrap()
	parseGot := PayloadFromSSRBootstrap(parseBootstrap)
	if parseGot.Route.Path != parsePayload.Route.Path || parseGot.Route.Screen != parsePayload.Route.Screen {
		parseT.Fatalf("expected route to survive bootstrap round-trip, got %#v", parseGot.Route)
	}
	if parseGot.CSRF != parsePayload.CSRF || parseGot.User == nil || parseGot.User.ID != parsePayload.User.ID {
		parseT.Fatalf("expected auth context to survive bootstrap round-trip, got %#v", parseGot)
	}
	parseRequest, parseOk := parseGot.Requests["catalog"]
	if !parseOk {
		parseT.Fatalf("expected request payload to survive bootstrap round-trip, got %#v", parseGot.Requests)
	}
	if _, parseDuplicated := parseRequest.Data["page"]; parseDuplicated {
		parseT.Fatalf("expected duplicate route-data keys to be filtered, got %#v", parseRequest.Data)
	}
	if _, parseKept := parseRequest.Data["items"]; !parseKept {
		parseT.Fatalf("expected non-duplicate request data to be preserved, got %#v", parseRequest.Data)
	}

	parseMinimal := PayloadFromSSRBootstrap(Payload{}.ToSSRBootstrap())
	if strings.TrimSpace(parseMinimal.I18n.Locale) == "" || strings.TrimSpace(parseMinimal.I18n.Direction) == "" {
		parseT.Fatalf("expected fallback i18n defaults, got %#v", parseMinimal.I18n)
	}
}

func TestBootstrapAndLegacyHelpersCoverRouteVariants(parseT *testing.T) {
	if parseGot := SupportedLocales(); len(parseGot) < 3 {
		parseT.Fatalf("expected supported locales to include baseline locales, got %#v", parseGot)
	}
	if parseGot2 := LocaleDirection("ar"); parseGot2 != "rtl" {
		parseT.Fatalf("expected arabic locale direction rtl, got %q", parseGot2)
	}
	if parseGot3 := LocaleDirection("en"); parseGot3 != "ltr" {
		parseT.Fatalf("expected english locale direction ltr, got %q", parseGot3)
	}

	for _, parsePath := range []string{
		RouteLanding,
		RouteCatalog,
		RouteProduct,
		RouteWarehouses,
		RouteWarehouseAvailability,
		RouteDashboard,
		RouteInventory,
		RouteSKUThresholdHistory,
		RouteWarehouseOps,
		RouteTransferDetail,
		RoutePurchaseOrders,
		RouteReceivingSessionDetail,
		RouteComments,
		RouteSettings,
		"/unknown",
	} {
		parseMeta := MetadataForPath(parsePath)
		if strings.TrimSpace(parseMeta.Title) == "" || strings.TrimSpace(parseMeta.Canonical) == "" {
			parseT.Fatalf("expected metadata title and canonical for %q, got %#v", parsePath, parseMeta)
		}
	}

	if parseRoute, parseOk := RouteBootstrapForPath(RouteCatalog); !parseOk || parseRoute.Path != RouteCatalog {
		parseT.Fatalf("expected route bootstrap for catalog, got ok=%t route=%#v", parseOk, parseRoute)
	}
	if _, parseOk2 := RouteBootstrapForPath("/missing"); parseOk2 {
		parseT.Fatal("expected unknown route to skip route bootstrap lookup")
	}

	parseValues := BuildCatalogQueryValues("desk", "desks", "illinois-hub", "warehouse", 2)
	if parseValues.Get("q") != "desk" || parseValues.Get("category") != "desks" || parseValues.Get("warehouse") != "illinois-hub" || parseValues.Get("sort") != "warehouse" || parseValues.Get("page") != "2" {
		parseT.Fatalf("unexpected catalog query values: %s", parseValues.Encode())
	}
	if parsePage := CatalogPageValue(" 0 "); parsePage != 1 {
		parseT.Fatalf("expected invalid page to clamp to 1, got %d", parsePage)
	}

	parseStartupCases := map[string]string{
		RouteCatalog:                 "/api/public/catalog",
		RouteCatalog + "/frame-desk": "/api/public/products/frame-desk",
		RouteWarehouses:              "/api/public/warehouses",
		RouteWarehouseAvailability:   "/api/public/warehouses/new-jersey-hub/availability/frame-desk",
		RouteWarehousePublicDetail:   "/api/public/warehouses/new-jersey-hub",
		RouteDashboard:               "/api/app/dashboard",
		RouteInventory:               "/api/app/inventory",
		RouteSKUThresholdHistory:     "/api/app/inventory/frame-desk/threshold-panel",
		RouteWarehouseOps:            "/api/app/warehouses",
		RouteWarehouseItemDetail:     "/api/app/warehouses/illinois-hub/items/frame-desk",
		RouteTransfers:               "/api/app/transfers",
		RouteTransferDetail:          "/api/app/transfers/tr-2048",
		RoutePurchaseOrders:          "/api/app/purchase-orders",
		RoutePurchaseOrderDetail:     "/api/app/purchase-orders/po-1042",
		RouteReceiving:               "/api/app/receiving",
		RouteReceivingSessionDetail:  "/api/app/receiving/illinois-accessories-042",
		RouteComments:                "/api/app/comments",
		RouteSettings:                "/api/app/settings",
	}
	for parsePath2, parseWant := range parseStartupCases {
		if parseGot4 := StartupRequestURL(parsePath2, url.Values{}); parseGot4 != parseWant {
			parseT.Fatalf("StartupRequestURL(%q) = %q, want %q", parsePath2, parseGot4, parseWant)
		}
	}
	if parseGot5 := StartupRequestURL("/unknown", url.Values{}); parseGot5 != "" {
		parseT.Fatalf("expected unknown startup path to return empty request URL, got %q", parseGot5)
	}
}

func TestLegacyDecisionAndDerivedStateHelpers(parseT *testing.T) {
	for _, parseFinish := range []string{"Graphite oak", "Drift ash", "Walnut ember"} {
		parseHero := ResolveAtlasProductHeroState("frame-desk", parseFinish)
		if strings.TrimSpace(parseHero.Label) == "" || strings.TrimSpace(parseHero.WarehousePromise) == "" {
			parseT.Fatalf("expected product hero copy for finish %q, got %#v", parseFinish, parseHero)
		}
	}
	parseLanes := ResolveAtlasPromiseLanes("frame-desk")
	if len(parseLanes) != 3 || !strings.Contains(parseLanes[0].Href, "frame-desk") {
		parseT.Fatalf("expected three promise lanes with product-specific href, got %#v", parseLanes)
	}

	for _, parseScenario := range []struct {
		savedView string
		warehouse string
	}{
		{"East coast shortages", "new-jersey-hub"},
		{"Low stock triage", "illinois-hub"},
		{"Balanced", "nevada-hub"},
	} {
		parseSummary := InventorySummaryFor(parseScenario.savedView, parseScenario.warehouse)
		if parseSummary.TotalAvailable == 0 || strings.TrimSpace(parseSummary.StatusLabel) == "" || strings.TrimSpace(parseSummary.SuggestedAction) == "" {
			parseT.Fatalf("expected inventory summary copy for %+v, got %#v", parseScenario, parseSummary)
		}
	}
	for _, parseStatus := range []string{"approved", "flagged", "rejected", "pending"} {
		parseScenario2 := ModerationScenarioForStatus(parseStatus)
		if strings.TrimSpace(parseScenario2.Record) == "" || strings.TrimSpace(parseScenario2.Decision) == "" {
			parseT.Fatalf("expected moderation scenario for %q, got %#v", parseStatus, parseScenario2)
		}
	}
	for _, parseAction := range []string{"approve", "reject", "flag", "unknown"} {
		parseOutcome := ModerationDecisionForAction(parseAction)
		if strings.TrimSpace(parseOutcome.Status) == "" {
			parseT.Fatalf("expected moderation decision outcome for %q, got %#v", parseAction, parseOutcome)
		}
	}
	if parseGot := WarehouseLabel("new-jersey-hub"); parseGot != "New Jersey Hub" {
		parseT.Fatalf("expected known warehouse label to expand, got %q", parseGot)
	}
	if parseGot2 := WarehouseLabel("custom-lane"); parseGot2 != "custom lane" {
		parseT.Fatalf("expected unknown warehouse label to normalize hyphens, got %q", parseGot2)
	}

	parseRepositoryRows := []repository.InventoryRow{
		{SKU: "frame-desk", Available: 3, Inbound: 4, ReorderUnits: 2, Status: "critical"},
		{SKU: "frame-desk", Available: 7, Inbound: 0, ReorderUnits: 0, Status: "balanced"},
		{SKU: "studio-console", Available: 2, Inbound: 1, ReorderUnits: 5, Status: "promise_risk"},
	}
	parseRollup := InventoryRollupFromRepositoryRows(parseRepositoryRows)
	if parseRollup.VisibleLanes != 3 || parseRollup.SKUCount != 2 || parseRollup.RiskLanes < 2 || parseRollup.SKUsWithInbound != 2 {
		parseT.Fatalf("unexpected inventory rollup summary: %#v", parseRollup)
	}

	parseRows := []inventoryRow{
		{SKU: "frame-desk", Title: "Frame Desk", WarehouseName: "New Jersey Hub", Available: 3, Inbound: 2, ReorderUnits: 2, Status: "critical", UpdatedAt: "2026-03-01"},
		{SKU: "frame-desk", Title: "Frame Desk", WarehouseName: "Illinois Hub", Available: 5, Inbound: 1, ReorderUnits: 0, Status: "balanced", UpdatedAt: "2026-03-02"},
		{SKU: "studio-console", Title: "Studio Console", WarehouseName: "Nevada Hub", Available: 7, Inbound: 0, ReorderUnits: 0, Status: "promise_risk", UpdatedAt: "2026-03-03"},
	}
	_ = inventoryRollupFromRows(parseRows)
	parseSummaries := inventorySummaryCards(parseRows)
	if len(parseSummaries) != 2 || parseSummaries[0].SKU != "frame-desk" {
		parseT.Fatalf("expected grouped inventory summary cards sorted by title, got %#v", parseSummaries)
	}

	for _, parseStatus2 := range []string{"in_stock", "low_stock", "unknown"} {
		if parsePromise := catalogPromiseCopy(parseStatus2); strings.TrimSpace(parsePromise) == "" {
			parseT.Fatalf("expected catalog promise copy for status %q", parseStatus2)
		}
		parseActionLabel, parseActionDetail := catalogActionPlan(parseStatus2)
		if strings.TrimSpace(parseActionLabel) == "" || strings.TrimSpace(parseActionDetail) == "" {
			parseT.Fatalf("expected action plan copy for status %q", parseStatus2)
		}
		if parseSupport := productSupportCue(parseStatus2); strings.TrimSpace(parseSupport) == "" {
			parseT.Fatalf("expected product support cue for status %q", parseStatus2)
		}
		if parseMotion := productBuyingMotion(parseStatus2); strings.TrimSpace(parseMotion) == "" {
			parseT.Fatalf("expected product buying motion for status %q", parseStatus2)
		}
		parsePlanTitle, parsePlanDetail, parseBullets := productSupportPlan(parseStatus2)
		if strings.TrimSpace(parsePlanTitle) == "" || strings.TrimSpace(parsePlanDetail) == "" || len(parseBullets) == 0 {
			parseT.Fatalf("expected product support plan for status %q", parseStatus2)
		}
	}
	if parseCopy := catalogEditorialCopy(productCard{}); strings.TrimSpace(parseCopy) == "" {
		parseT.Fatal("expected fallback editorial copy to be non-empty")
	}
	if parseCopy2 := catalogEditorialCopy(productCard{SEODescription: "SEO copy"}); parseCopy2 != "SEO copy" {
		parseT.Fatalf("expected SEO description to win editorial copy fallback, got %q", parseCopy2)
	}
	for _, parseCategory := range []string{"desks", "storage", "lighting", "bundles", "other"} {
		if parseCue := productCategoryCue(parseCategory); strings.TrimSpace(parseCue) == "" {
			parseT.Fatalf("expected non-empty product category cue for %q", parseCategory)
		}
	}
	for _, parseService := range []string{"priority", "two-day", "standard"} {
		if parseTone := warehouseServiceTone(parseService); strings.TrimSpace(parseTone) == "" {
			parseT.Fatalf("expected non-empty warehouse service tone for %q", parseService)
		}
	}
	for _, parseRegion := range []string{"west coast", "midwest", "east coast", "other"} {
		if parseCue2 := warehouseRegionCue(parseRegion); strings.TrimSpace(parseCue2) == "" {
			parseT.Fatalf("expected non-empty warehouse region cue for %q", parseRegion)
		}
	}
	if parseStory := availabilityStoryCopy(10, 2); strings.TrimSpace(parseStory) == "" {
		parseT.Fatal("expected availability story copy to be non-empty")
	}
	parseTitle, parseDetail := availabilitySupportPlan(1, 0)
	if strings.TrimSpace(parseTitle) == "" || strings.TrimSpace(parseDetail) == "" {
		parseT.Fatal("expected availability support plan to return copy")
	}
}
