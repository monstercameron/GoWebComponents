package atlas

import (
	"net/url"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/shared/repository"
)

func TestPayloadBootstrapRoundTripAndRequestDedup(t *testing.T) {
	payload := Payload{
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

	bootstrap := payload.ToSSRBootstrap()
	got := PayloadFromSSRBootstrap(bootstrap)
	if got.Route.Path != payload.Route.Path || got.Route.Screen != payload.Route.Screen {
		t.Fatalf("expected route to survive bootstrap round-trip, got %#v", got.Route)
	}
	if got.CSRF != payload.CSRF || got.User == nil || got.User.ID != payload.User.ID {
		t.Fatalf("expected auth context to survive bootstrap round-trip, got %#v", got)
	}
	request, ok := got.Requests["catalog"]
	if !ok {
		t.Fatalf("expected request payload to survive bootstrap round-trip, got %#v", got.Requests)
	}
	if _, duplicated := request.Data["page"]; duplicated {
		t.Fatalf("expected duplicate route-data keys to be filtered, got %#v", request.Data)
	}
	if _, kept := request.Data["items"]; !kept {
		t.Fatalf("expected non-duplicate request data to be preserved, got %#v", request.Data)
	}

	minimal := PayloadFromSSRBootstrap(Payload{}.ToSSRBootstrap())
	if strings.TrimSpace(minimal.I18n.Locale) == "" || strings.TrimSpace(minimal.I18n.Direction) == "" {
		t.Fatalf("expected fallback i18n defaults, got %#v", minimal.I18n)
	}
}

func TestBootstrapAndLegacyHelpersCoverRouteVariants(t *testing.T) {
	if got := SupportedLocales(); len(got) < 3 {
		t.Fatalf("expected supported locales to include baseline locales, got %#v", got)
	}
	if got := LocaleDirection("ar"); got != "rtl" {
		t.Fatalf("expected arabic locale direction rtl, got %q", got)
	}
	if got := LocaleDirection("en"); got != "ltr" {
		t.Fatalf("expected english locale direction ltr, got %q", got)
	}

	for _, path := range []string{
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
		meta := MetadataForPath(path)
		if strings.TrimSpace(meta.Title) == "" || strings.TrimSpace(meta.Canonical) == "" {
			t.Fatalf("expected metadata title and canonical for %q, got %#v", path, meta)
		}
	}

	if route, ok := RouteBootstrapForPath(RouteCatalog); !ok || route.Path != RouteCatalog {
		t.Fatalf("expected route bootstrap for catalog, got ok=%t route=%#v", ok, route)
	}
	if _, ok := RouteBootstrapForPath("/missing"); ok {
		t.Fatal("expected unknown route to skip route bootstrap lookup")
	}

	values := BuildCatalogQueryValues("desk", "desks", "illinois-hub", "warehouse", 2)
	if values.Get("q") != "desk" || values.Get("category") != "desks" || values.Get("warehouse") != "illinois-hub" || values.Get("sort") != "warehouse" || values.Get("page") != "2" {
		t.Fatalf("unexpected catalog query values: %s", values.Encode())
	}
	if page := CatalogPageValue(" 0 "); page != 1 {
		t.Fatalf("expected invalid page to clamp to 1, got %d", page)
	}

	startupCases := map[string]string{
		RouteCatalog:                                             "/api/public/catalog",
		RouteCatalog + "/frame-desk":                            "/api/public/products/frame-desk",
		RouteWarehouses:                                          "/api/public/warehouses",
		RouteWarehouseAvailability:                               "/api/public/warehouses/new-jersey-hub/availability/frame-desk",
		RouteWarehousePublicDetail:                               "/api/public/warehouses/new-jersey-hub",
		RouteDashboard:                                           "/api/app/dashboard",
		RouteInventory:                                           "/api/app/inventory",
		RouteSKUThresholdHistory:                                 "/api/app/inventory/frame-desk/threshold-panel",
		RouteWarehouseOps:                                        "/api/app/warehouses",
		RouteWarehouseItemDetail:                                 "/api/app/warehouses/illinois-hub/items/frame-desk",
		RouteTransfers:                                           "/api/app/transfers",
		RouteTransferDetail:                                      "/api/app/transfers/tr-2048",
		RoutePurchaseOrders:                                      "/api/app/purchase-orders",
		RoutePurchaseOrderDetail:                                 "/api/app/purchase-orders/po-1042",
		RouteReceiving:                                           "/api/app/receiving",
		RouteReceivingSessionDetail:                              "/api/app/receiving/illinois-accessories-042",
		RouteComments:                                            "/api/app/comments",
		RouteSettings:                                            "/api/app/settings",
	}
	for path, want := range startupCases {
		if got := StartupRequestURL(path, url.Values{}); got != want {
			t.Fatalf("StartupRequestURL(%q) = %q, want %q", path, got, want)
		}
	}
	if got := StartupRequestURL("/unknown", url.Values{}); got != "" {
		t.Fatalf("expected unknown startup path to return empty request URL, got %q", got)
	}
}

func TestLegacyDecisionAndDerivedStateHelpers(t *testing.T) {
	for _, finish := range []string{"Graphite oak", "Drift ash", "Walnut ember"} {
		hero := ResolveAtlasProductHeroState("frame-desk", finish)
		if strings.TrimSpace(hero.Label) == "" || strings.TrimSpace(hero.WarehousePromise) == "" {
			t.Fatalf("expected product hero copy for finish %q, got %#v", finish, hero)
		}
	}
	lanes := ResolveAtlasPromiseLanes("frame-desk")
	if len(lanes) != 3 || !strings.Contains(lanes[0].Href, "frame-desk") {
		t.Fatalf("expected three promise lanes with product-specific href, got %#v", lanes)
	}

	for _, scenario := range []struct {
		savedView string
		warehouse string
	}{
		{"East coast shortages", "new-jersey-hub"},
		{"Low stock triage", "illinois-hub"},
		{"Balanced", "nevada-hub"},
	} {
		summary := InventorySummaryFor(scenario.savedView, scenario.warehouse)
		if summary.TotalAvailable == 0 || strings.TrimSpace(summary.StatusLabel) == "" || strings.TrimSpace(summary.SuggestedAction) == "" {
			t.Fatalf("expected inventory summary copy for %+v, got %#v", scenario, summary)
		}
	}
	for _, status := range []string{"approved", "flagged", "rejected", "pending"} {
		scenario := ModerationScenarioForStatus(status)
		if strings.TrimSpace(scenario.Record) == "" || strings.TrimSpace(scenario.Decision) == "" {
			t.Fatalf("expected moderation scenario for %q, got %#v", status, scenario)
		}
	}
	for _, action := range []string{"approve", "reject", "flag", "unknown"} {
		outcome := ModerationDecisionForAction(action)
		if strings.TrimSpace(outcome.Status) == "" {
			t.Fatalf("expected moderation decision outcome for %q, got %#v", action, outcome)
		}
	}
	if got := WarehouseLabel("new-jersey-hub"); got != "New Jersey Hub" {
		t.Fatalf("expected known warehouse label to expand, got %q", got)
	}
	if got := WarehouseLabel("custom-lane"); got != "custom lane" {
		t.Fatalf("expected unknown warehouse label to normalize hyphens, got %q", got)
	}

	repositoryRows := []repository.InventoryRow{
		{SKU: "frame-desk", Available: 3, Inbound: 4, ReorderUnits: 2, Status: "critical"},
		{SKU: "frame-desk", Available: 7, Inbound: 0, ReorderUnits: 0, Status: "balanced"},
		{SKU: "studio-console", Available: 2, Inbound: 1, ReorderUnits: 5, Status: "promise_risk"},
	}
	rollup := InventoryRollupFromRepositoryRows(repositoryRows)
	if rollup.VisibleLanes != 3 || rollup.SKUCount != 2 || rollup.RiskLanes < 2 || rollup.SKUsWithInbound != 2 {
		t.Fatalf("unexpected inventory rollup summary: %#v", rollup)
	}

	rows := []inventoryRow{
		{SKU: "frame-desk", Title: "Frame Desk", WarehouseName: "New Jersey Hub", Available: 3, Inbound: 2, ReorderUnits: 2, Status: "critical", UpdatedAt: "2026-03-01"},
		{SKU: "frame-desk", Title: "Frame Desk", WarehouseName: "Illinois Hub", Available: 5, Inbound: 1, ReorderUnits: 0, Status: "balanced", UpdatedAt: "2026-03-02"},
		{SKU: "studio-console", Title: "Studio Console", WarehouseName: "Nevada Hub", Available: 7, Inbound: 0, ReorderUnits: 0, Status: "promise_risk", UpdatedAt: "2026-03-03"},
	}
	_ = inventoryRollupFromRows(rows)
	summaries := inventorySummaryCards(rows)
	if len(summaries) != 2 || summaries[0].SKU != "frame-desk" {
		t.Fatalf("expected grouped inventory summary cards sorted by title, got %#v", summaries)
	}

	for _, status := range []string{"in_stock", "low_stock", "unknown"} {
		if promise := catalogPromiseCopy(status); strings.TrimSpace(promise) == "" {
			t.Fatalf("expected catalog promise copy for status %q", status)
		}
		actionLabel, actionDetail := catalogActionPlan(status)
		if strings.TrimSpace(actionLabel) == "" || strings.TrimSpace(actionDetail) == "" {
			t.Fatalf("expected action plan copy for status %q", status)
		}
		if support := productSupportCue(status); strings.TrimSpace(support) == "" {
			t.Fatalf("expected product support cue for status %q", status)
		}
		if motion := productBuyingMotion(status); strings.TrimSpace(motion) == "" {
			t.Fatalf("expected product buying motion for status %q", status)
		}
		planTitle, planDetail, bullets := productSupportPlan(status)
		if strings.TrimSpace(planTitle) == "" || strings.TrimSpace(planDetail) == "" || len(bullets) == 0 {
			t.Fatalf("expected product support plan for status %q", status)
		}
	}
	if copy := catalogEditorialCopy(productCard{}); strings.TrimSpace(copy) == "" {
		t.Fatal("expected fallback editorial copy to be non-empty")
	}
	if copy := catalogEditorialCopy(productCard{SEODescription: "SEO copy"}); copy != "SEO copy" {
		t.Fatalf("expected SEO description to win editorial copy fallback, got %q", copy)
	}
	for _, category := range []string{"desks", "storage", "lighting", "bundles", "other"} {
		if cue := productCategoryCue(category); strings.TrimSpace(cue) == "" {
			t.Fatalf("expected non-empty product category cue for %q", category)
		}
	}
	for _, service := range []string{"priority", "two-day", "standard"} {
		if tone := warehouseServiceTone(service); strings.TrimSpace(tone) == "" {
			t.Fatalf("expected non-empty warehouse service tone for %q", service)
		}
	}
	for _, region := range []string{"west coast", "midwest", "east coast", "other"} {
		if cue := warehouseRegionCue(region); strings.TrimSpace(cue) == "" {
			t.Fatalf("expected non-empty warehouse region cue for %q", region)
		}
	}
	if story := availabilityStoryCopy(10, 2); strings.TrimSpace(story) == "" {
		t.Fatal("expected availability story copy to be non-empty")
	}
	title, detail := availabilitySupportPlan(1, 0)
	if strings.TrimSpace(title) == "" || strings.TrimSpace(detail) == "" {
		t.Fatal("expected availability support plan to return copy")
	}
}
