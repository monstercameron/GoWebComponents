//go:build js && wasm

package app

import "testing"

func TestParseResolvePricingFragmentScroll(parseT *testing.T) {
	parseTests := []struct {
		name         string
		currentPath  string
		hash         string
		wantTargetID string
		wantTop      bool
	}{
		{
			name:         "pricing faq hash resolves faq section",
			currentPath:  marketingPricingRoute,
			hash:         "#faq",
			wantTargetID: "faq",
			wantTop:      false,
		},
		{
			name:         "pricing without hash resets to top",
			currentPath:  marketingPricingRoute,
			hash:         "",
			wantTargetID: "",
			wantTop:      true,
		},
		{
			name:         "unknown pricing hash is ignored",
			currentPath:  marketingPricingRoute,
			hash:         "#missing",
			wantTargetID: "",
			wantTop:      false,
		},
		{
			name:         "non pricing routes ignore pricing fragments",
			currentPath:  marketingHomeRoute,
			hash:         "#faq",
			wantTargetID: "",
			wantTop:      false,
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			parseGotTargetID, parseGotTop := parseResolvePricingFragmentScroll(parseTest.currentPath, parseTest.hash)
			if parseGotTargetID != parseTest.wantTargetID || parseGotTop != parseTest.wantTop {
				parseT2.Fatalf("parseResolvePricingFragmentScroll(%q, %q) = (%q, %v), want (%q, %v)", parseTest.currentPath, parseTest.hash, parseGotTargetID, parseGotTop, parseTest.wantTargetID, parseTest.wantTop)
			}
		})
	}
}
