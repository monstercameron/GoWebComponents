package router

import (
	"net/url"
	"strings"
	"testing"
)

func TestRouteContractAdditionalBranches(parseT *testing.T) {
	parseRootContract, parseErr := DefineRoute(" # ")
	if parseErr != nil {
		parseT.Fatalf("DefineRoute(#) error = %v", parseErr)
	}
	if parseRootContract.Pattern() != "/" {
		parseT.Fatalf("root contract Pattern() = %q, want /", parseRootContract.Pattern())
	}
	if parsePath, parseErr2 := parseRootContract.Path(nil); parseErr2 != nil || parsePath != "/" {
		parseT.Fatalf("root contract Path(nil) = %q, %v; want /, nil", parsePath, parseErr2)
	}
	if parseHref, parseErr3 := parseRootContract.HrefFor(nil, nil); parseErr3 != nil || parseHref != "/" {
		parseT.Fatalf("root contract HrefFor(nil,nil) = %q, %v; want /, nil", parseHref, parseErr3)
	}
	if parseNames := parseRootContract.ParamNames(); parseNames != nil {
		parseT.Fatalf("root contract ParamNames() = %#v, want nil", parseNames)
	}

	parsePatternCases := map[string]string{
		"users/:id":          "/users/:id",
		"#/users/:id?page=2": "/users/:id",
		"/users/:id/":        "/users/:id",
		"#":                  "/",
	}
	for parseInput, parseWant := range parsePatternCases {
		if parseGot := normalizeRouteContractPattern(parseInput); parseGot != parseWant {
			parseT.Fatalf("normalizeRouteContractPattern(%q) = %q, want %q", parseInput, parseGot, parseWant)
		}
	}

	for _, parsePattern := range []string{"", "*", "/users/:", "/docs/*/detail"} {
		if _, parseErr4 := DefineRoute(parsePattern); parseErr4 == nil {
			parseT.Fatalf("DefineRoute(%q) error = nil, want validation failure", parsePattern)
		}
	}

	func() {
		defer func() {
			if parseRecovered := recover(); parseRecovered == nil {
				parseT.Fatal("MustDefineRoute() did not panic for invalid pattern")
			}
		}()
		_ = MustDefineRoute("*")
	}()

	parseUninitialized := RouteContract{}
	if _, parseErr5 := parseUninitialized.Path(map[string]string{"id": "42"}); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "not initialized") {
		parseT.Fatalf("Path(uninitialized) error = %v, want not initialized", parseErr5)
	}

	parseContract := MustDefineRoute("/users/:id")
	if _, parseErr6 := parseContract.Path(map[string]string{" ": "42"}); parseErr6 == nil || !strings.Contains(parseErr6.Error(), "does not define params") {
		parseT.Fatalf("Path(blank key) error = %v, want unexpected param error", parseErr6)
	}

	func() {
		defer func() {
			if parseRecovered2 := recover(); parseRecovered2 == nil {
				parseT.Fatal("MustPath() did not panic for missing param")
			}
		}()
		_ = parseContract.MustPath(nil)
	}()

	func() {
		defer func() {
			if parseRecovered3 := recover(); parseRecovered3 == nil {
				parseT.Fatal("MustHrefFor() did not panic for missing param")
			}
		}()
		_ = parseContract.MustHrefFor(nil, nil)
	}()

	if routeParamsFromProvider(nil) != nil {
		parseT.Fatal("routeParamsFromProvider(nil) should return nil")
	}
	if routeQueryFromProvider(nil) != nil {
		parseT.Fatal("routeQueryFromProvider(nil) should return nil")
	}

	parseHref2, parseErr := parseContract.Href(map[string]string{"id": "42"}, url.Values{"tab": {"settings"}})
	if parseErr != nil {
		parseT.Fatalf("Href() error = %v", parseErr)
	}
	if parseHref2 != "/users/42?tab=settings" {
		parseT.Fatalf("Href() = %q, want /users/42?tab=settings", parseHref2)
	}
}
