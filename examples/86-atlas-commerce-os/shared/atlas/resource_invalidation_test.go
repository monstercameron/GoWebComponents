package atlas

import (
	"reflect"
	"testing"
)

func TestMutationRoutePrefixesUsesSharedInvalidationMatrix(t *testing.T) {
	got := MutationRoutePrefixes("/app/purchase-orders/po-1042", "purchase-order-created")
	want := []string{
		"/app/purchase-orders",
		"/app/receiving",
		"/app/dashboard",
		"/app/inventory",
		"/app/warehouses",
		"/shop",
		"/warehouses",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MutationRoutePrefixes() = %#v, want %#v", got, want)
	}
}

func TestMutationRequestPrefixesCoversInternalAndPublicFamilies(t *testing.T) {
	got := MutationRequestPrefixes([]string{"/app/settings", "/shop", "/warehouses"})
	want := []string{
		"/api/app/settings",
		"/api/app/saved-views",
		"/api/public/catalog",
		"/api/public/products",
		"/api/public/warehouses",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MutationRequestPrefixes() = %#v, want %#v", got, want)
	}
}

func TestMutationRoutePrefixesFallsBackToCurrentPath(t *testing.T) {
	got := MutationRoutePrefixes("/app/inventory/frame-desk", "unknown-notice")
	want := []string{"/app/inventory/frame-desk"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MutationRoutePrefixes() = %#v, want %#v", got, want)
	}
}
