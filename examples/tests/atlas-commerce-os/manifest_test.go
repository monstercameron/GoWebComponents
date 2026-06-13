package atlascommerceostests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type parseManifest struct {
	Version                int                         `json:"version"`
	OwnerScope             string                      `json:"owner_scope"`
	RunOrder               []string                    `json:"run_order"`
	Buckets                []parseBucket               `json:"buckets"`
	Helpers                []parseHelper               `json:"helpers"`
	ManualStories          []string                    `json:"manual_stories"`
	E2ETracks              []parseE2ETrack             `json:"e2e_tracks"`
	FailureRecoveryStories []parseFailureRecoveryStory `json:"failure_recovery_stories"`
	ScreenshotConventions  parseScreenshotConventions  `json:"screenshot_conventions"`
	ScreenshotCheckpoints  []string                    `json:"screenshot_checkpoints"`
}

type parseBucket struct {
	ID             string       `json:"id"`
	Directory      string       `json:"directory"`
	Type           string       `json:"type"`
	Purpose        string       `json:"purpose"`
	Specs          []string     `json:"specs"`
	Stories        []parseStory `json:"stories"`
	ReferenceFiles []string     `json:"reference_files"`
}

type parseStory struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Routes      []string `json:"routes"`
	Assertions  []string `json:"assertions"`
	Checkpoints []string `json:"checkpoints"`
}

type parseHelper struct {
	ID      string `json:"id"`
	Purpose string `json:"purpose"`
}

type parseE2ETrack struct {
	ID      string   `json:"id"`
	Stories []string `json:"stories"`
}

type parseFailureRecoveryStory struct {
	ID         string   `json:"id"`
	Routes     []string `json:"routes"`
	Assertions []string `json:"assertions"`
}

type parseScreenshotConventions struct {
	Directory    string   `json:"directory"`
	Pattern      string   `json:"pattern"`
	Surfaces     []string `json:"surfaces"`
	Themes       []string `json:"themes"`
	Viewports    []string `json:"viewports"`
	Locales      []string `json:"locales"`
	RefreshOrder []string `json:"refresh_order"`
}

// TestAtlasCommerceOSManifest validates the planning skeleton used by future Atlas Playwright suites.
func TestAtlasCommerceOSManifest(parseT *testing.T) {
	parseManifest := loadManifest(parseT)
	if parseManifest.Version != 1 {
		parseT.Fatalf("expected manifest version 1, got %d", parseManifest.Version)
	}
	if strings.Contains(strings.ToLower(parseManifest.OwnerScope), "example 100") {
		parseT.Fatalf("manifest scope must not include Example 100")
	}

	assertRequiredIDs(parseT, "bucket", collectBucketIDs(parseManifest.Buckets), []string{
		"buyer-flow",
		"operator-flow",
		"design-parity",
	})
	assertRequiredIDs(parseT, "helper", collectHelperIDs(parseManifest.Helpers), []string{
		"buyer-navigation",
		"operator-navigation",
		"parity-landmarks",
		"screenshot-capture",
		"route-shell-stability",
	})
	assertEqualOrder(parseT, parseManifest.RunOrder, []string{"buyer-flow", "operator-flow", "design-parity"})
	assertEqualOrder(parseT, parseManifest.ScreenshotConventions.RefreshOrder, []string{"buyer-flow", "operator-flow", "design-parity"})

	for _, parseBucket := range parseManifest.Buckets {
		assertBucketValid(parseT, parseBucket)
	}
	assertRequiredIDs(parseT, "manual story", mapFromSlice(parseManifest.ManualStories), []string{
		"manual-public-browse-path",
		"manual-public-quote-submit",
		"manual-public-restock-submit",
		"manual-public-comment-submit",
		"manual-public-mobile-nav",
		"manual-internal-sign-in-recovery",
		"manual-product-editor-unsaved-save",
		"manual-inventory-threshold-edit",
		"manual-warehouse-item-context",
		"manual-transfer-detail-decision",
		"manual-purchase-order-decision-refresh",
		"manual-receiving-discrepancy-reconcile",
		"manual-comments-moderation-toast",
		"manual-settings-preference-persistence",
		"manual-diagnostics-review-mode",
	})
	assertRequiredIDs(parseT, "E2E track", collectTrackIDs(parseManifest.E2ETracks), []string{
		"buyer-flow-e2e",
		"operator-flow-e2e",
	})
	for _, parseTrack := range parseManifest.E2ETracks {
		if len(parseTrack.Stories) == 0 {
			parseT.Fatalf("E2E track %s must name stories", parseTrack.ID)
		}
	}
	assertRequiredIDs(parseT, "failure story", collectFailureIDs(parseManifest.FailureRecoveryStories), []string{
		"recovery-not-found",
		"recovery-server-error",
		"recovery-failed-write",
		"recovery-network-interruption",
		"recovery-duplicate-submit-timeout",
		"recovery-invalid-input-state",
		"recovery-async-navigation-race",
	})
	for _, parseStory := range parseManifest.FailureRecoveryStories {
		if len(parseStory.Routes) == 0 || len(parseStory.Assertions) == 0 {
			parseT.Fatalf("failure story %s must include routes and assertions", parseStory.ID)
		}
	}
	assertScreenshotConventionsValid(parseT, parseManifest)
}

// loadManifest reads the Atlas test manifest from the current package directory.
func loadManifest(parseT *testing.T) parseManifest {
	parseT.Helper()
	parseBytes, parseErr := os.ReadFile("manifest.json")
	if parseErr != nil {
		parseT.Fatalf("read manifest.json: %v", parseErr)
	}
	var parseManifest parseManifest
	if parseErr := json.Unmarshal(parseBytes, &parseManifest); parseErr != nil {
		parseT.Fatalf("parse manifest.json: %v", parseErr)
	}
	return parseManifest
}

// assertBucketValid checks common bucket fields and local directory existence.
func assertBucketValid(parseT *testing.T, parseBucket parseBucket) {
	parseT.Helper()
	if parseBucket.ID == "" || parseBucket.Directory == "" || parseBucket.Purpose == "" {
		parseT.Fatalf("bucket has empty id, directory, or purpose: %+v", parseBucket)
	}
	if _, parseErr := os.Stat(parseBucket.Directory); parseErr != nil {
		parseT.Fatalf("bucket %s directory %s missing: %v", parseBucket.ID, parseBucket.Directory, parseErr)
	}
	if len(parseBucket.Specs) == 0 {
		parseT.Fatalf("bucket %s must name future specs", parseBucket.ID)
	}
	for _, parseSpec := range parseBucket.Specs {
		parseSpecPath := filepath.Join(parseBucket.Directory, parseSpec+".spec.md")
		if _, parseErr := os.Stat(parseSpecPath); parseErr != nil {
			parseT.Fatalf("bucket %s spec skeleton %s missing: %v", parseBucket.ID, parseSpecPath, parseErr)
		}
	}
	if len(parseBucket.Stories) == 0 {
		parseT.Fatalf("bucket %s must name stories", parseBucket.ID)
	}
	for _, parseStory := range parseBucket.Stories {
		if parseStory.ID == "" || parseStory.Title == "" {
			parseT.Fatalf("bucket %s has story with empty id or title", parseBucket.ID)
		}
		if len(parseStory.Routes) == 0 {
			parseT.Fatalf("bucket %s story %s must include routes", parseBucket.ID, parseStory.ID)
		}
		if len(parseStory.Assertions) == 0 {
			parseT.Fatalf("bucket %s story %s must include assertions", parseBucket.ID, parseStory.ID)
		}
	}
	for _, parseReference := range parseBucket.ReferenceFiles {
		if _, parseErr := os.Stat(filepath.Clean(parseReference)); parseErr != nil {
			parseT.Fatalf("bucket %s reference file %s missing: %v", parseBucket.ID, parseReference, parseErr)
		}
	}
}

// assertScreenshotConventionsValid verifies the capture matrix has the required Atlas review dimensions.
func assertScreenshotConventionsValid(parseT *testing.T, parseManifest parseManifest) {
	parseT.Helper()
	parseConventions := parseManifest.ScreenshotConventions
	if !strings.Contains(parseConventions.Pattern, "<surface>") || !strings.Contains(parseConventions.Pattern, "<route>") {
		parseT.Fatalf("screenshot pattern must include surface and route placeholders: %s", parseConventions.Pattern)
	}
	assertRequiredIDs(parseT, "screenshot surface", mapFromSlice(parseConventions.Surfaces), []string{"public", "internal", "recovery", "parity"})
	assertRequiredIDs(parseT, "screenshot theme", mapFromSlice(parseConventions.Themes), []string{"light", "dark"})
	assertRequiredIDs(parseT, "screenshot viewport", mapFromSlice(parseConventions.Viewports), []string{"desktop", "mobile"})
	assertRequiredIDs(parseT, "screenshot locale", mapFromSlice(parseConventions.Locales), []string{"en", "fr", "ar"})
	assertRequiredIDs(parseT, "screenshot checkpoint", mapFromSlice(parseManifest.ScreenshotCheckpoints), []string{
		"public-landing-start",
		"public-catalog-browsing",
		"public-product-decision",
		"public-warehouse-availability",
		"public-post-submit-success",
		"operator-dashboard-start",
		"operator-product-editor",
		"operator-inventory-triage",
		"operator-warehouse-detail",
		"operator-po-receiving",
		"operator-settings-persistence",
		"empty-state",
		"no-results-state",
		"recovery-page",
		"validation-error",
		"success-state",
		"overlay-open",
		"locale-fr-public",
		"locale-ar-internal",
		"parity-public-start",
		"parity-public-midpoint",
		"parity-public-end",
		"parity-operator-start",
		"parity-operator-midpoint",
		"parity-operator-end",
	})
}

// assertRequiredIDs fails when a required id is absent.
func assertRequiredIDs(parseT *testing.T, parseLabel string, parseActual map[string]bool, parseRequired []string) {
	parseT.Helper()
	for _, parseID := range parseRequired {
		if !parseActual[parseID] {
			parseT.Fatalf("missing %s id %q", parseLabel, parseID)
		}
	}
}

// assertEqualOrder checks exact ordering for layered browser review.
func assertEqualOrder(parseT *testing.T, parseActual []string, parseExpected []string) {
	parseT.Helper()
	if len(parseActual) != len(parseExpected) {
		parseT.Fatalf("expected order %v, got %v", parseExpected, parseActual)
	}
	for parseIndex := range parseExpected {
		if parseActual[parseIndex] != parseExpected[parseIndex] {
			parseT.Fatalf("expected order %v, got %v", parseExpected, parseActual)
		}
	}
}

// collectBucketIDs returns bucket ids as a set.
func collectBucketIDs(parseBuckets []parseBucket) map[string]bool {
	parseIDs := make(map[string]bool, len(parseBuckets))
	for _, parseBucket := range parseBuckets {
		parseIDs[parseBucket.ID] = true
	}
	return parseIDs
}

// collectHelperIDs returns helper ids as a set.
func collectHelperIDs(parseHelpers []parseHelper) map[string]bool {
	parseIDs := make(map[string]bool, len(parseHelpers))
	for _, parseHelper := range parseHelpers {
		parseIDs[parseHelper.ID] = true
	}
	return parseIDs
}

// collectTrackIDs returns E2E track ids as a set.
func collectTrackIDs(parseTracks []parseE2ETrack) map[string]bool {
	parseIDs := make(map[string]bool, len(parseTracks))
	for _, parseTrack := range parseTracks {
		parseIDs[parseTrack.ID] = true
	}
	return parseIDs
}

// collectFailureIDs returns failure story ids as a set.
func collectFailureIDs(parseStories []parseFailureRecoveryStory) map[string]bool {
	parseIDs := make(map[string]bool, len(parseStories))
	for _, parseStory := range parseStories {
		parseIDs[parseStory.ID] = true
	}
	return parseIDs
}

// mapFromSlice returns string values as a set.
func mapFromSlice(parseValues []string) map[string]bool {
	parseMapped := make(map[string]bool, len(parseValues))
	for _, parseValue := range parseValues {
		parseMapped[parseValue] = true
	}
	return parseMapped
}
