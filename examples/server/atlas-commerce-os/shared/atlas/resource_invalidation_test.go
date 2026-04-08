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
