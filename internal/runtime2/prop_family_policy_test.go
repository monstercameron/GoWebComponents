package runtime2_test

import (
	"slices"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestGetRenderAllowedPropFamiliesIncludesExpectedFirstSliceFamilies verifies first-slice prop families are explicitly listed.
func TestGetRenderAllowedPropFamiliesIncludesExpectedFirstSliceFamilies(parseT *testing.T) {
	getAllowedPropFamilies := runtime2.GetRenderAllowedPropFamilies()
	if len(getAllowedPropFamilies) == 0 {
		parseT.Fatal("expected explicit allowed prop-family set to be non-empty")
	}
	parseExpectedPropFamilies := []string{"class", "style", "aria", "data"}
	for _, parseExpectedPropFamily := range parseExpectedPropFamilies {
		if !slices.Contains(getAllowedPropFamilies, parseExpectedPropFamily) {
			parseT.Fatalf("expected allowed prop-family set to contain %q, got %+v", parseExpectedPropFamily, getAllowedPropFamilies)
		}
	}
}

// TestGetRenderAllowedPropFamiliesExcludesInteractiveFamilies verifies interactive prop families are not first-slice defaults.
func TestGetRenderAllowedPropFamiliesExcludesInteractiveFamilies(parseT *testing.T) {
	getAllowedPropFamilies := runtime2.GetRenderAllowedPropFamilies()
	parseRejectedPropFamilies := []string{"event", "ref", "portal", "interop"}
	for _, parseRejectedPropFamily := range parseRejectedPropFamilies {
		if slices.Contains(getAllowedPropFamilies, parseRejectedPropFamily) {
			parseT.Fatalf("did not expect allowed prop-family set to contain %q", parseRejectedPropFamily)
		}
	}
}

// TestGetRenderAllowedPropFamiliesReturnsStableSortedSet verifies the returned prop-family set is stable and sorted.
func TestGetRenderAllowedPropFamiliesReturnsStableSortedSet(parseT *testing.T) {
	getAllowedPropFamilies := runtime2.GetRenderAllowedPropFamilies()
	if !slices.IsSorted(getAllowedPropFamilies) {
		parseT.Fatalf("expected allowed prop-family set to be sorted, got %+v", getAllowedPropFamilies)
	}
	getAllowedPropFamiliesCopy := runtime2.GetRenderAllowedPropFamilies()
	if !slices.Equal(getAllowedPropFamilies, getAllowedPropFamiliesCopy) {
		parseT.Fatalf("expected stable allowed prop-family set, first=%+v second=%+v", getAllowedPropFamilies, getAllowedPropFamiliesCopy)
	}
}
