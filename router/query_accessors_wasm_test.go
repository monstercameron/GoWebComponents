//go:build js && wasm

package router

import (
	"net/url"
	"testing"
)

// TestQueryIntBoolMirrorParams proves Query/SearchParams gained the typed Int/Bool accessors
// that Params already had — so route params and query params read the same way, and an absent
// or unparseable key reports ok=false rather than a silent zero.
func TestQueryIntBoolMirrorParams(parseT *testing.T) {
	parseQuery := Query{values: url.Values{"page": {"7"}, "live": {"true"}, "bad": {"x"}}}

	if parseN, parseOk := parseQuery.Int("page"); !parseOk || parseN != 7 {
		parseT.Fatalf("Query.Int(page) = %d,%v want 7,true", parseN, parseOk)
	}
	if parseB, parseOk := parseQuery.Bool("live"); !parseOk || !parseB {
		parseT.Fatalf("Query.Bool(live) = %v,%v want true,true", parseB, parseOk)
	}
	if _, parseOk := parseQuery.Int("missing"); parseOk {
		parseT.Fatal("Query.Int(missing) must report ok=false")
	}
	if _, parseOk := parseQuery.Int("bad"); parseOk {
		parseT.Fatal("Query.Int(bad) must report ok=false for an unparseable value")
	}
	if _, parseOk := parseQuery.Bool("bad"); parseOk {
		parseT.Fatal("Query.Bool(bad) must report ok=false for an unparseable value")
	}

	// SearchParams delegates to the same logic.
	parseSearch := SearchParams{values: url.Values{"n": {"3"}}}
	if parseN, parseOk := parseSearch.Int("n"); !parseOk || parseN != 3 {
		parseT.Fatalf("SearchParams.Int(n) = %d,%v want 3,true", parseN, parseOk)
	}
}
