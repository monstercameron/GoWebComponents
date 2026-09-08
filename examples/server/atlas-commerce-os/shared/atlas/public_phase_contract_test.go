package atlas

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func TestAtlasPublicPhaseShellCatalogLanguageAndAccessibilityContracts(parseT *testing.T) {
	parsePayload := samplePayloadForRoute(RouteCatalog, catalogPage{
		Items:    sampleProductCards(),
		Total:    len(sampleProductCards()),
		Page:     1,
		PageSize: 12,
		Query: catalogQueryState{
			Search:    "frame",
			Category:  "desks",
			Warehouse: "new-jersey-hub",
			Sort:      "featured",
		},
		Focus: "Featured Atlas systems",
	}, nil)
	parsePayload.I18n = DefaultI18n("fr")
	parsePayload.Preferences.Locale = "fr"
	parsePayload.Route.Query = map[string][]string{
		"q":      {"frame"},
		"locale": {"fr"},
	}

	// component + props, not App(payload): App's hooks need a render fiber.
	parseMarkup, parseErr := renderAtlasNodeForTest(ui.CreateElement(App, parsePayload))
	if parseErr != nil {
		parseT.Fatalf("render public catalog app: %v", parseErr)
	}
	for _, parseNeedle := range []string{
		`id="atlas-shell-root"`,
		`id="atlas-overlay-root"`,
		`radial-gradient(circle_at_top,rgba(245,158,11`,
		"Accueil",
		"Boutique",
		"Entrepots",
		"Langue",
		`hreflang="fr"`,
		`aria-current="true"`,
		`href="/shop?locale=fr&amp;q=frame"`,
		// The catalog route now ships ONE page head and one manifest. The old
		// needles here were the second and third hero ("Curated modular workspace
		// catalog", "Modern workspace systems, organized for quick decisions."),
		// both of which were deleted; these three assert the replacements — the
		// page-head sentence, the manifest's list semantics, and the mono SKU
		// column that proves rows render as manifest lines rather than cards.
		"Filter the manifest, then compare price and availability line by line.",
		`role="list"`,
		"FRAME-DESK",
		"Search",
		"Category",
		"Warehouse",
		"Sort",
		"Frame Desk",
	} {
		if !strings.Contains(parseMarkup, parseNeedle) {
			parseT.Fatalf("expected public catalog markup to contain %q, got %q", parseNeedle, parseMarkup)
		}
	}
}

func TestAtlasPublicPhaseProductWarehouseCacheInteractionAndRecoveryContracts(parseT *testing.T) {
	parseProducts := sampleProductCards()
	parseWarehouses := sampleWarehouseCards()
	parsePayload := samplePayloadForRoute(RouteProduct, productDetailPage{
		Product:  parseProducts[0],
		Comments: sampleCommentRecords(),
	}, nil)

	parseProductMarkup := renderAtlasComponentForTest(parseT, func() ui.Node {
		return renderProductContent(productDetailPage{
			Product:  parseProducts[0],
			Comments: sampleCommentRecords(),
		}, parsePayload)
	})
	for _, parseNeedle := range []string{
		// Was "Why this product page is easier to use" — a three-card strip in which
		// the page graded its own UX to the buyer. Deleted; the price label is the
		// product hero's real content and a better route marker.
		publicStartingAtLabel,
		"Open buying drawer",
		"Reserve upcoming availability",
		`name="csrf_token"`,
		"Related systems",
		"Repeat-open visits can reuse the cached related-product list",
		"Customer reviews and questions",
		"Regional promise lanes",
		"Atlas defers this secondary lane module until after hydration",
	} {
		if !strings.Contains(parseProductMarkup, parseNeedle) {
			parseT.Fatalf("expected product markup to contain %q, got %q", parseNeedle, parseProductMarkup)
		}
	}

	parseDirectoryMarkup := renderAtlasMarkupForTest(parseT, renderWarehouseDirectoryContent(warehouseDirectoryPage{Items: parseWarehouses}))
	for _, parseNeedle := range []string{
		"Atlas delivery regions",
		// Was "Regional commerce board" (the label on a stat block whose "%d stocked
		// units" / "%d inbound units" summed fields /api/public/warehouses does not
		// return, so they printed 0 in production) and "Browse all systems" (a
		// duplicate of the page head's own catalog action). Both deleted. These two
		// assert what replaced them: the directory's real heading, and the hub code
		// that proves each row carries the hub's identity.
		"Pick the hub that serves your site",
		"NJ-HUB",
		"New Jersey Hub",
		"Open warehouse route",
	} {
		if !strings.Contains(parseDirectoryMarkup, parseNeedle) {
			parseT.Fatalf("expected warehouse directory markup to contain %q, got %q", parseNeedle, parseDirectoryMarkup)
		}
	}

	parseDetailMarkup := renderAtlasComponentForTest(parseT, func() ui.Node {
		return renderWarehouseDetailContent(warehouseDetailPage{
			Warehouse: parseWarehouses[0],
			Products:  parseProducts,
		}, parsePayload)
	})
	for _, parseNeedle := range []string{
		"Regional availability picks",
		"Repeat-open visits can reuse this side snapshot",
		"Open stocked systems that fit this regional route.",
	} {
		if !strings.Contains(parseDetailMarkup, parseNeedle) {
			parseT.Fatalf("expected warehouse detail markup to contain %q, got %q", parseNeedle, parseDetailMarkup)
		}
	}

	parseAvailabilityMarkup := renderAtlasMarkupForTest(parseT, renderAvailabilityContent(availabilityPage{
		Warehouse: parseWarehouses[0],
		Product:   parseProducts[0],
		Available: 0,
		Inbound:   3,
		Status:    "promise_risk",
	}, parsePayload))
	for _, parseNeedle := range []string{
		"Warehouse-specific promise",
		"Reserve upcoming availability",
		"Ask about delivery or fit",
		// Rewritten support point. The old sentence ("Use reserve capture to hold
		// buyer intent against the inbound recovery window for this specific hub.")
		// was written for the team building the page, not for a buyer.
		"Reserve now and you hold your place in the next inbound batch for this hub.",
	} {
		if !strings.Contains(parseAvailabilityMarkup, parseNeedle) {
			parseT.Fatalf("expected availability markup to contain %q, got %q", parseNeedle, parseAvailabilityMarkup)
		}
	}

	parseRecoveryPayload := samplePayloadForRoute("/missing-public-route", recoveryPage{
		Title:         "Atlas Route Not Found",
		Message:       "The requested Atlas route could not be resolved.",
		RecoveryHref:  RouteCatalog,
		RecoveryLabel: "Back to shop",
	}, nil)
	parseRecoveryPayload.Route.Screen = "recovery"
	parseRecoveryPayload.Route.Surface = "public"
	parseRecoveryMarkup, parseErr := renderAtlasNodeForTest(ui.CreateElement(App, parseRecoveryPayload))
	if parseErr != nil {
		parseT.Fatalf("render public recovery app: %v", parseErr)
	}
	if !strings.Contains(parseRecoveryMarkup, "Back to shop") || !strings.Contains(parseRecoveryMarkup, RouteCatalog) {
		parseT.Fatalf("expected public recovery affordance, got %q", parseRecoveryMarkup)
	}
}
