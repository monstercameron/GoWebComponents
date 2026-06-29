package atlas

import (
	"reflect"
	"testing"
)

func TestMutationRoutePrefixesUsesSharedInvalidationMatrix(parseT *testing.T) {
	parseGot := MutationRoutePrefixes("/app/purchase-orders/po-1042", "purchase-order-created")
	parseWant := []string{
		"/app/purchase-orders",
		"/app/receiving",
		"/app/dashboard",
		"/app/inventory",
		"/app/warehouses",
		"/shop",
		"/warehouses",
	}
	if !reflect.DeepEqual(parseGot, parseWant) {
		parseT.Fatalf("MutationRoutePrefixes() = %#v, want %#v", parseGot, parseWant)
	}
}

func TestMutationRequestPrefixesCoversInternalAndPublicFamilies(parseT *testing.T) {
	parseGot := MutationRequestPrefixes([]string{"/app/settings", "/shop", "/warehouses"})
	parseWant := []string{
		"/api/app/settings",
		"/api/app/saved-views",
		"/api/public/catalog",
		"/api/public/products",
		"/api/public/warehouses",
	}
	if !reflect.DeepEqual(parseGot, parseWant) {
		parseT.Fatalf("MutationRequestPrefixes() = %#v, want %#v", parseGot, parseWant)
	}
}

func TestMutationRoutePrefixesFallsBackToCurrentPath(parseT *testing.T) {
	parseGot := MutationRoutePrefixes("/app/inventory/frame-desk", "unknown-notice")
	parseWant := []string{"/app/inventory/frame-desk"}
	if !reflect.DeepEqual(parseGot, parseWant) {
		parseT.Fatalf("MutationRoutePrefixes() = %#v, want %#v", parseGot, parseWant)
	}
}

func TestMutationInvalidationMatrixCoversEveryNoticeFamily(parseT *testing.T) {
	parseExpectedPublicNotices := map[string]struct{}{
		"comment-submitted":          {},
		"comment-moderated":          {},
		"comments-bulk-moderated":    {},
		"inventory-updated":          {},
		"product-created":            {},
		"product-deleted":            {},
		"product-updated":            {},
		"purchase-order-created":     {},
		"quote-request-submitted":    {},
		"restock-request-submitted":  {},
	}
	parseExpectedDashboardNotices := map[string]struct{}{
		"comment-submitted":       {},
		"comment-moderated":       {},
		"comments-bulk-moderated": {},
		"inventory-updated":       {},
		"preferences-saved":       {},
		"purchase-order-created":  {},
		"purchase-order-updated":  {},
		"receiving-reconciled":    {},
		"saved-view-created":      {},
		"saved-views-imported":    {},
		"threshold-updated":       {},
		"transfer-created":        {},
	}

	for parseNotice, parsePrefixes := range mutationInvalidationRules {
		parseT.Run(parseNotice, func(parseT2 *testing.T) {
			if len(parsePrefixes) == 0 {
				parseT2.Fatalf("notice %q has no route invalidation targets", parseNotice)
			}
			parseRequestPrefixes := MutationRequestPrefixes(parsePrefixes)
			if len(parseRequestPrefixes) == 0 {
				parseT2.Fatalf("notice %q has no request invalidation targets for %#v", parseNotice, parsePrefixes)
			}
			if _, parseNeedsPublic := parseExpectedPublicNotices[parseNotice]; parseNeedsPublic && !containsAtlasPrefix(parsePrefixes, "/shop") && !containsAtlasPrefix(parsePrefixes, "/warehouses") {
				parseT2.Fatalf("notice %q should invalidate at least one public route family, got %#v", parseNotice, parsePrefixes)
			}
			if _, parseNeedsDashboard := parseExpectedDashboardNotices[parseNotice]; parseNeedsDashboard && !containsAtlasPrefix(parsePrefixes, "/app/dashboard") {
				parseT2.Fatalf("notice %q should invalidate dashboard summaries, got %#v", parseNotice, parsePrefixes)
			}
		})
	}
}

func containsAtlasPrefix(parsePrefixes []string, parseNeedle string) bool {
	for _, parsePrefix := range parsePrefixes {
		if parsePrefix == parseNeedle {
			return true
		}
	}
	return false
}
