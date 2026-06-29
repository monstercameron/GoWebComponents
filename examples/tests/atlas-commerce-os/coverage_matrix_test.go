package atlascommerceostests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type parseCoverageMatrix struct {
	Version          int                  `json:"version"`
	RouteMatrix      []parseCoverageRoute `json:"route_matrix"`
	FeatureMatrix    []parseFeatureEntry  `json:"feature_matrix"`
	QualityRules     []string             `json:"quality_rules"`
	ReleaseChecklist []string             `json:"release_checklist"`
}

type parseCoverageRoute struct {
	Route           string   `json:"route"`
	Surface         string   `json:"surface"`
	Unit            []string `json:"unit"`
	Component       []string `json:"component"`
	Integration     []string `json:"integration"`
	SSR             []string `json:"ssr"`
	Hydration       []string `json:"hydration"`
	Accessibility   []string `json:"accessibility"`
	PlaywrightStory []string `json:"playwright_story"`
}

type parseFeatureEntry struct {
	ID    string   `json:"id"`
	Tests []string `json:"tests"`
}

func TestAtlasCoverageMatrixReferencesExistingTestsAndStories(parseT *testing.T) {
	parseMatrix := loadCoverageMatrix(parseT)
	if parseMatrix.Version != 1 {
		parseT.Fatalf("expected coverage matrix version 1, got %d", parseMatrix.Version)
	}
	parseTests := collectAtlasTestFunctions(parseT)
	parseStories := collectManifestStoryIDs(loadManifest(parseT))

	assertRequiredIDs(parseT, "coverage route", collectCoverageRouteIDs(parseMatrix.RouteMatrix), []string{
		"/",
		"/shop",
		"/shop/:slug",
		"/warehouses",
		"/warehouses/:slug",
		"/warehouses/:slug/availability/:productSlug",
		"/app/dashboard",
		"/app/products",
		"/app/products/:slug",
		"/app/inventory",
		"/app/inventory/:sku",
		"/app/warehouses",
		"/app/warehouses/:warehouseId",
		"/app/warehouses/:warehouseId/items/:sku",
		"/app/transfers",
		"/app/transfers/:id",
		"/app/purchase-orders",
		"/app/purchase-orders/:id",
		"/app/receiving",
		"/app/receiving/:id",
		"/app/comments",
		"/app/settings",
	})
	for _, parseRoute := range parseMatrix.RouteMatrix {
		if parseRoute.Surface != "public" && parseRoute.Surface != "internal" {
			parseT.Fatalf("route %s has unsupported surface %q", parseRoute.Route, parseRoute.Surface)
		}
		assertTestRefsExist(parseT, parseTests, parseRoute.Route+" unit", parseRoute.Unit)
		assertTestRefsExist(parseT, parseTests, parseRoute.Route+" component", parseRoute.Component)
		assertTestRefsExist(parseT, parseTests, parseRoute.Route+" integration", parseRoute.Integration)
		assertTestRefsExist(parseT, parseTests, parseRoute.Route+" SSR", parseRoute.SSR)
		assertTestRefsExist(parseT, parseTests, parseRoute.Route+" hydration", parseRoute.Hydration)
		assertTestRefsExist(parseT, parseTests, parseRoute.Route+" accessibility", parseRoute.Accessibility)
		assertStoryRefsExist(parseT, parseStories, parseRoute.Route+" playwright story", parseRoute.PlaywrightStory)
	}

	assertRequiredIDs(parseT, "feature matrix", collectFeatureIDs(parseMatrix.FeatureMatrix), []string{
		"route-metadata-layout-bootstrap",
		"locale-theme-density-preferences",
		"csrf-origin-write-security",
		"filter-sort-pagination-saved-views",
		"copy-derived-state-toast-validation",
		"repository-migrations-seed-reads-writes",
		"rendered-shell-card-table-form-overlay-recovery",
		"public-catalog-product-forms-recovery",
		"internal-products-inventory-logistics-moderation-settings",
		"ssr-navigation-revalidation-recovery",
		"accessibility-keyboard-motion-contrast",
		"localization-rtl-formatting",
	})
	for _, parseFeature := range parseMatrix.FeatureMatrix {
		assertTestRefsExist(parseT, parseTests, parseFeature.ID, parseFeature.Tests)
	}
	assertRequiredIDs(parseT, "quality rule", mapFromSlice(parseMatrix.QualityRules), []string{
		"new-surface-requires-unit-rendered-and-flow-assertion",
		"route-family-requires-ssr-hydration-and-reload-coverage",
		"write-flow-requires-server-hydrated-and-no-js-form-coverage",
		"overlay-requires-keyboard-focus-dismiss-and-route-resume-stories",
		"loader-cache-path-requires-invalidation-revalidation-stale-and-retry-stories",
		"release-checklist-verifies-required-test-layers-for-touched-surfaces",
	})
	assertRequiredIDs(parseT, "release checklist command", mapFromSlice(parseMatrix.ReleaseChecklist), []string{
		"go test ./examples/server/atlas-commerce-os/shared/atlas",
		"go test ./examples/server/atlas-commerce-os/server",
		"go test ./examples/tests/atlas-commerce-os",
	})
}

func loadCoverageMatrix(parseT *testing.T) parseCoverageMatrix {
	parseT.Helper()
	parseBytes, parseErr := os.ReadFile("coverage_matrix.json")
	if parseErr != nil {
		parseT.Fatalf("read coverage_matrix.json: %v", parseErr)
	}
	var parseMatrix parseCoverageMatrix
	if parseErr := json.Unmarshal(parseBytes, &parseMatrix); parseErr != nil {
		parseT.Fatalf("parse coverage_matrix.json: %v", parseErr)
	}
	return parseMatrix
}

func collectAtlasTestFunctions(parseT *testing.T) map[string]bool {
	parseT.Helper()
	parseRoot := filepath.Clean(filepath.Join("..", "..", "server", "atlas-commerce-os"))
	parsePattern := regexp.MustCompile(`func\s+(Test[A-Za-z0-9_]+)\s*\(`)
	parseTests := map[string]bool{}
	parseErr := filepath.WalkDir(parseRoot, func(parsePath string, parseEntry os.DirEntry, parseErr error) error {
		if parseErr != nil {
			return parseErr
		}
		if parseEntry.IsDir() || !strings.HasSuffix(parsePath, "_test.go") {
			return nil
		}
		parseBytes, parseReadErr := os.ReadFile(parsePath)
		if parseReadErr != nil {
			return parseReadErr
		}
		for _, parseMatch := range parsePattern.FindAllStringSubmatch(string(parseBytes), -1) {
			parseTests[parseMatch[1]] = true
		}
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("collect Atlas test functions: %v", parseErr)
	}
	if len(parseTests) == 0 {
		parseT.Fatal("expected Atlas test function inventory to be non-empty")
	}
	return parseTests
}

func collectManifestStoryIDs(parseManifest parseManifest) map[string]bool {
	parseIDs := map[string]bool{}
	for _, parseBucket := range parseManifest.Buckets {
		for _, parseStory := range parseBucket.Stories {
			parseIDs[parseStory.ID] = true
		}
	}
	for _, parseStory := range parseManifest.ManualStories {
		parseIDs[parseStory] = true
	}
	for _, parseTrack := range parseManifest.E2ETracks {
		parseIDs[parseTrack.ID] = true
		for _, parseStory := range parseTrack.Stories {
			parseIDs[parseStory] = true
		}
	}
	for _, parseStory := range parseManifest.FailureRecoveryStories {
		parseIDs[parseStory.ID] = true
	}
	return parseIDs
}

func collectCoverageRouteIDs(parseRoutes []parseCoverageRoute) map[string]bool {
	parseIDs := map[string]bool{}
	for _, parseRoute := range parseRoutes {
		parseIDs[parseRoute.Route] = true
	}
	return parseIDs
}

func collectFeatureIDs(parseFeatures []parseFeatureEntry) map[string]bool {
	parseIDs := map[string]bool{}
	for _, parseFeature := range parseFeatures {
		parseIDs[parseFeature.ID] = true
	}
	return parseIDs
}

func assertTestRefsExist(parseT *testing.T, parseTests map[string]bool, parseLabel string, parseRefs []string) {
	parseT.Helper()
	if len(parseRefs) == 0 {
		parseT.Fatalf("%s must reference at least one test", parseLabel)
	}
	for _, parseRef := range parseRefs {
		if !parseTests[parseRef] {
			parseT.Fatalf("%s references missing test %q", parseLabel, parseRef)
		}
	}
}

func assertStoryRefsExist(parseT *testing.T, parseStories map[string]bool, parseLabel string, parseRefs []string) {
	parseT.Helper()
	if len(parseRefs) == 0 {
		parseT.Fatalf("%s must reference at least one story", parseLabel)
	}
	for _, parseRef := range parseRefs {
		if !parseStories[parseRef] {
			parseT.Fatalf("%s references missing manifest story %q", parseLabel, parseRef)
		}
	}
}
