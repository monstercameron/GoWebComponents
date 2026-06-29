package atlas

import (
	"strings"
	"testing"
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
	parseMarkup := renderAtlasMarkupForTest(parseT, App(parsePayload))
	for _, parseNeedle := range []string{
		publicHeaderShellClass(),
		publicCatalogControlShellClass(),
		publicCatalogCardClass(),
		publicFormControlClass(),
		"Browse Atlas",
		"Modern workspace systems",
	} {
		if !strings.Contains(parseMarkup, parseNeedle) {
			parseT.Fatalf("expected catalog shell markup to contain %q", parseNeedle)
		}
	}
	if strings.Count(parseMarkup, "<header") != 1 {
		parseT.Fatalf("expected exactly one public header, got markup %q", parseMarkup)
	}
}

func TestAtlasRenderedVisualPrimitivesPreserveProductAndInternalTableParity(parseT *testing.T) {
	parseT.Parallel()

	parseProduct := sampleProductCards()[0]
	parseProductMarkup := renderAtlasMarkupForTest(parseT, publicProductHeroCard(parseProduct))
	for _, parseNeedle := range []string{
		publicHeroSurfaceClass(),
		publicSignalPillClass(),
		parseProduct.Title,
		formatPrice(parseProduct.PriceCents),
	} {
		if !strings.Contains(parseProductMarkup, parseNeedle) {
			parseT.Fatalf("expected product hero markup to contain %q", parseNeedle)
		}
	}

	parseTableMarkup := renderAtlasMarkupForTest(parseT, productCMSTableShell(sampleProductAdminCards()))
	for _, parseNeedle := range []string{
		internalSurfaceCardClass(),
		internalTableContainerClass(),
		internalTableHeaderCellClass(),
		internalTableRowClass(),
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

func atlasVisualRulesContain(parseNeedle string) bool {
	for _, parseRule := range atlasVisualEfficiencyRules() {
		if strings.Contains(parseRule, parseNeedle) {
			return true
		}
	}
	return false
}
