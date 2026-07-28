package atlas

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/examples/server/atlas-commerce-os/shared/design"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func TestAtlasVisualPrimitiveClassesCoverParitySurfaces(parseT *testing.T) {
	parseT.Parallel()

	parseLayoutCases := map[string]string{
		"public root":   atlasRootSurfaceClass(atlasVisualSurfacePublic),
		"internal root": atlasRootSurfaceClass(atlasVisualSurfaceInternal),
		"public main":   atlasMainShellClass(atlasVisualSurfacePublic),
		"internal main": atlasMainShellClass(atlasVisualSurfaceInternal),
	}
	for parseName, parseClass := range parseLayoutCases {
		if strings.TrimSpace(parseClass) == "" {
			parseT.Fatalf("%s primitive returned an empty class", parseName)
		}
	}
	parseSurfaceCases := map[string]string{
		"public header":     publicHeaderShellClass(),
		"public hero":       publicHeroSurfaceClass(),
		"public glass card": publicGlassCardClass(),
		"public metric":     publicMetricSurfaceClass(),
		"catalog card":      publicCatalogCardClass(),
		"catalog controls":  publicCatalogControlShellClass(),
		"form control":      publicFormControlClass(),
		"signal pill":       publicSignalPillClass(),
		"internal hero":     internalHeroSurfaceClass(),
		"internal card":     internalSurfaceCardClass(),
		"internal table":    internalTableContainerClass(),
	}
	for parseName, parseClass := range parseSurfaceCases {
		if strings.TrimSpace(parseClass) == "" {
			parseT.Fatalf("%s primitive returned an empty class", parseName)
		}
		if !strings.Contains(parseClass, "border") {
			parseT.Fatalf("%s primitive should preserve the parity border contract, got %q", parseName, parseClass)
		}
	}
	for _, parseNeedle := range []string{"heavy gradients", "semantic table", "reduced-motion", "shared class helpers"} {
		if !atlasVisualRulesContain(parseNeedle) {
			parseT.Fatalf("expected visual efficiency rules to mention %q", parseNeedle)
		}
	}
}

func TestAtlasRenderedVisualPrimitivesPreservePublicParityShell(parseT *testing.T) {
	parseT.Parallel()

	parsePayload := samplePayloadForRoute(RouteCatalog, catalogPage{
		Items: sampleProductCards(),
		Query: catalogQueryState{Sort: "warehouse"},
	}, nil)
	parseMarkup := renderAtlasComponentForTest(parseT, func() ui.Node {
		return App(parsePayload)
	})
	// publicCatalogCardClass() and "Modern workspace systems" are gone from this
	// list on purpose: the storefront catalog no longer renders product CARDS or a
	// second "Catalog overview" hero. /shop is now one page head plus one
	// design.Catalog manifest, so the parity contract this test guards moved with
	// it — the filter shell and its controls still come from the shared helpers,
	// and the manifest is asserted by its list semantics and a real product row.
	for _, parseNeedle := range []string{
		publicHeaderShellClass(),
		publicCatalogControlShellClass(),
		publicFormControlClass(),
		"Browse Atlas",
		`role="list"`,
		"Frame Desk",
	} {
		if !strings.Contains(parseMarkup, parseNeedle) {
			parseT.Fatalf("expected catalog shell markup to contain %q", parseNeedle)
		}
	}
	// And the disease itself must not come back: no product card frames on /shop.
	if strings.Contains(parseMarkup, publicCatalogCardClass()) {
		parseT.Fatalf("the storefront catalog is a manifest, not a card grid; found the catalog card class in %q", parseMarkup)
	}
	if strings.Count(parseMarkup, "<header") != 1 {
		parseT.Fatalf("expected exactly one public header, got markup %q", parseMarkup)
	}
}

func TestAtlasRenderedVisualPrimitivesPreserveProductAndInternalTableParity(parseT *testing.T) {
	parseT.Parallel()

	parseProduct := sampleProductCards()[0]
	// The product hero is a design.Surface now, so the two utility-class needles
	// (publicHeroSurfaceClass, publicSignalPillClass) are gone — and so is the
	// product TITLE, deliberately: the route's own <h1> already names the product,
	// and the hero repeating it was one of the three stacked heroes this pass
	// removed. What the hero must still carry is the identity code, the price and
	// the availability chip, which is what these needles pin.
	parseProductMarkup := renderAtlasMarkupForTest(parseT, publicProductHeroCard(parseProduct))
	for _, parseNeedle := range []string{
		atlasSKUCode(parseProduct.SKU),
		strings.ToUpper(parseProduct.Category),
		formatPrice(parseProduct.PriceCents),
		publicStartingAtLabel,
	} {
		if !strings.Contains(parseProductMarkup, parseNeedle) {
			parseT.Fatalf("expected product hero markup to contain %q", parseNeedle)
		}
	}

	// The catalog table used to be pinned to the four internal*Class() Tailwind strings
	// — a gradient card wrapping a gradient scroll container wrapping the table. It now
	// folds design-system bundles instead, so the contract this test protects is the
	// same one stated differently: the table still bleeds to a single flush surface,
	// still lives in a scroll region, and still emits semantic table markup with real
	// numeric cells rather than divs pretending to be rows.
	parseTableMarkup := renderAtlasMarkupForTest(parseT, productCMSTableShell(sampleProductAdminCards()))
	for _, parseNeedle := range []string{
		design.Class(design.SurfaceFlush()),
		design.Class(design.TableScroll()),
		design.Class(design.Table()),
		design.Class(design.NumericCell()),
		"<table",
		"<thead",
		"<tbody",
	} {
		if !strings.Contains(parseTableMarkup, parseNeedle) {
			parseT.Fatalf("expected internal product table markup to contain %q", parseNeedle)
		}
	}
	if strings.Contains(parseTableMarkup, "<div><tr") || strings.Contains(parseTableMarkup, "<tbody><div") {
		parseT.Fatalf("internal table parity contract should not wrap rows in divs: %q", parseTableMarkup)
	}
}

// TestInventorySurfacesEmitDesignSystemMarkup pins the conversion of the inventory,
// warehouse and catalog surfaces onto the design system by asserting on the EMITTED
// markup rather than by reading the call sites.
//
// It covers the six regions that are built entirely inside inventory_cms.go and
// products_cms.go — no page.go helper renders into them — so "no legacy utility class
// survives here" is a statement this test can actually make. Two things are checked:
//
//  1. Every queue is a real design.Table inside a single flush surface with a
//     keyboard-reachable scroll region, and its numeric columns carry NumericCell.
//  2. None of the retired Tailwind vocabulary appears: no slate/cyan/emerald/amber/rose
//     palette classes, no arbitrary radii, no gradient or shadow utilities. Those are
//     the exact strings the old markup was made of, and they are how a regression would
//     look if someone pasted a class string back in.
func TestInventorySurfacesEmitDesignSystemMarkup(parseT *testing.T) {
	parseT.Parallel()

	parseRows := sampleInventoryRows()
	parseSurfaces := map[string]ui.Node{
		"inventory queue":    inventoryQueueTable(inventorySummaryCards(parseRows)),
		"on hand by hub":     skuLaneRosterTable(parseRows),
		"hub item roster":    warehouseDetailInventoryTable("illinois-hub", parseRows),
		"network comparison": warehouseItemNetworkTable("/app/warehouses/illinois-hub/items/frame-desk", map[string]string{"sort": "cover", "dir": "asc"}, "illinois-hub", parseRows),
		"catalog table":      productCMSTableShell(sampleProductAdminCards()),
		"lane editor":        inventoryLaneEditorCardWithOptions(parseRows[0], Payload{CSRF: "atlas-csrf"}, "/app/inventory/frame-desk"),
	}

	// The retired vocabulary. Each prefix is a family the design system replaced with a
	// token, a role or a tone; a hit means a call site went back to picking a value.
	parseRetired := []string{
		"bg-slate-", "text-slate-", "border-slate-",
		"text-cyan-", "border-cyan-", "bg-cyan-",
		"emerald-", "amber-", "rose-",
		"linear-gradient(", "shadow-[", "rounded-[", "tracking-[",
	}

	for parseName, parseNode := range parseSurfaces {
		parseMarkup := renderAtlasMarkupForTest(parseT, parseNode)
		for _, parseNeedle := range parseRetired {
			if strings.Contains(parseMarkup, parseNeedle) {
				parseT.Fatalf("%s markup still contains retired utility class %q: %s", parseName, parseNeedle, parseMarkup)
			}
		}
	}

	for _, parseName := range []string{"inventory queue", "on hand by hub", "hub item roster", "network comparison", "catalog table"} {
		parseMarkup := renderAtlasMarkupForTest(parseT, parseSurfaces[parseName])
		for _, parseNeedle := range []string{
			design.Class(design.SurfaceFlush()),
			design.Class(design.Table()),
			design.Class(design.NumericCell()),
			`role="region"`,
			`tabIndex="0"`,
		} {
			if !strings.Contains(parseMarkup, parseNeedle) {
				parseT.Fatalf("%s markup missing %q: %s", parseName, parseNeedle, parseMarkup)
			}
		}
	}
}

// TestInventoryStatusTonesFollowDesignSystemJudgement pins the two mappings that a
// future contributor is most likely to "fix" back the wrong way.
//
// low_stock and promise_risk are NOT the same tone as critical. Being under a reorder
// point is what reorder points are for, and a promise that may slip is a lane a human
// should look at — not a lane that has already failed. Only genuine failure reaches the
// one filled tone, which is what keeps oxide meaning something in a 40-row queue.
func TestInventoryStatusTonesFollowDesignSystemJudgement(parseT *testing.T) {
	parseT.Parallel()

	parseCases := map[string]design.StatusTone{
		"balanced":     design.ToneVerified,
		"in_stock":     design.ToneVerified,
		"low_stock":    design.ToneNeutral,
		"draft":        design.ToneNeutral,
		"archived":     design.ToneNeutral,
		"promise_risk": design.TonePending,
		"recovery":     design.TonePending,
		"critical":     design.ToneException,
	}
	for parseStatus, parseWant := range parseCases {
		if parseGot := inventoryStatusTone(parseStatus); parseGot != parseWant {
			parseT.Fatalf("inventoryStatusTone(%q) = %v, want %v", parseStatus, parseGot, parseWant)
		}
	}

	// A count of flagged lanes is the one numeric column that earns the filled tone,
	// and only when it is non-zero: an empty risk column is the absence of a finding,
	// not a finding that passed.
	if parseGot := atlasRiskCountTone(0); parseGot != design.ToneNeutral {
		parseT.Fatalf("atlasRiskCountTone(0) = %v, want neutral", parseGot)
	}
	if parseGot := atlasRiskCountTone(2); parseGot != design.ToneException {
		parseT.Fatalf("atlasRiskCountTone(2) = %v, want exception", parseGot)
	}
	if parseGot := atlasCoverTone(0); parseGot != design.ToneException {
		parseT.Fatalf("atlasCoverTone(0) = %v, want exception", parseGot)
	}
	if parseGot := atlasCoverTone(3); parseGot != design.TonePending {
		parseT.Fatalf("atlasCoverTone(3) = %v, want pending", parseGot)
	}
	if parseGot := atlasCoverTone(21); parseGot != design.ToneNeutral {
		parseT.Fatalf("atlasCoverTone(21) = %v, want neutral", parseGot)
	}
}

func atlasVisualRulesContain(parseNeedle string) bool {
	for _, parseRule := range atlasVisualEfficiencyRules() {
		if strings.Contains(parseRule, parseNeedle) {
			return true
		}
	}
	return false
}
