package runtime2_test

import (
	"slices"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestGetRenderAllowedHostTagsIncludesExpectedDisplayTags verifies first-slice display tags are explicitly listed.
func TestGetRenderAllowedHostTagsIncludesExpectedDisplayTags(parseT *testing.T) {
	getAllowedHostTags := runtime2.GetRenderAllowedHostTags()
	if len(getAllowedHostTags) == 0 {
		parseT.Fatal("expected explicit allowed host-tag set to be non-empty")
	}
	parseExpectedHostTags := []string{"div", "section", "span", "ul", "li"}
	for _, parseExpectedHostTag := range parseExpectedHostTags {
		if !slices.Contains(getAllowedHostTags, parseExpectedHostTag) {
			parseT.Fatalf("expected allowed host-tag set to contain %q, got %+v", parseExpectedHostTag, getAllowedHostTags)
		}
	}
}

// TestGetRenderAllowedHostTagsExcludesNonDisplayTags verifies unsafe or non-display tags are not first-slice defaults.
func TestGetRenderAllowedHostTagsExcludesNonDisplayTags(parseT *testing.T) {
	getAllowedHostTags := runtime2.GetRenderAllowedHostTags()
	parseRejectedHostTags := []string{"script", "style", "iframe"}
	for _, parseRejectedHostTag := range parseRejectedHostTags {
		if slices.Contains(getAllowedHostTags, parseRejectedHostTag) {
			parseT.Fatalf("did not expect allowed host-tag set to contain %q", parseRejectedHostTag)
		}
	}
}

// TestGetRenderAllowedHostTagsReturnsStableSortedSet verifies the returned host-tag set is stable and sorted.
func TestGetRenderAllowedHostTagsReturnsStableSortedSet(parseT *testing.T) {
	getAllowedHostTags := runtime2.GetRenderAllowedHostTags()
	if !slices.IsSorted(getAllowedHostTags) {
		parseT.Fatalf("expected allowed host-tag set to be sorted, got %+v", getAllowedHostTags)
	}
	getAllowedHostTagsCopy := runtime2.GetRenderAllowedHostTags()
	if !slices.Equal(getAllowedHostTags, getAllowedHostTagsCopy) {
		parseT.Fatalf("expected stable allowed host-tag set, first=%+v second=%+v", getAllowedHostTags, getAllowedHostTagsCopy)
	}
}
