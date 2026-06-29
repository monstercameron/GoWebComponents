package atlas

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/ui"
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

	parseMarkup, parseErr := ui.RenderToString(App(parsePayload))
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
		"Curated modular workspace catalog",
		"Modern workspace systems, organized for quick decisions.",
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

	parseProductMarkup := renderAtlasMarkupForTest(parseT, renderProductContent(productDetailPage{
		Product:  parseProducts[0],
		Comments: sampleCommentRecords(),
	}, parsePayload))
	for _, parseNeedle := range []string{
		"Why this product page is easier to use",
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
		"Regional commerce board",
		"Browse all systems",
		"New Jersey Hub",
		"Open warehouse route",
	} {
		if !strings.Contains(parseDirectoryMarkup, parseNeedle) {
			parseT.Fatalf("expected warehouse directory markup to contain %q, got %q", parseNeedle, parseDirectoryMarkup)
		}
	}

	parseDetailMarkup := renderAtlasMarkupForTest(parseT, renderWarehouseDetailContent(warehouseDetailPage{
		Warehouse: parseWarehouses[0],
		Products:  parseProducts,
	}, parsePayload))
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
		"Use reserve capture to hold buyer intent against the inbound recovery window for this specific hub.",
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
	parseRecoveryMarkup, parseErr := ui.RenderToString(App(parseRecoveryPayload))
	if parseErr != nil {
		parseT.Fatalf("render public recovery app: %v", parseErr)
	}
	if !strings.Contains(parseRecoveryMarkup, "Back to shop") || !strings.Contains(parseRecoveryMarkup, RouteCatalog) {
		parseT.Fatalf("expected public recovery affordance, got %q", parseRecoveryMarkup)
	}
}
