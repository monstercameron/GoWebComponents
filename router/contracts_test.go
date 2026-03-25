package router

import (
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type testUserRouteParams struct {
	ID string
}

func (parseP testUserRouteParams) RouteParams() map[string]string {
	return map[string]string{"id": parseP.ID}
}

type testUserRouteQuery struct {
	Tab   string
	Page  int
	Draft bool
}

func (parseQ testUserRouteQuery) RouteQuery() url.Values {
	parseValues := url.Values{}
	if parseQ.Tab != "" {
		parseValues.Set("tab", parseQ.Tab)
	}
	if parseQ.Page > 0 {
		parseValues.Set("page", strconv.Itoa(parseQ.Page))
	}
	if parseQ.Draft {
		parseValues.Set("draft", "true")
	}
	return parseValues
}

func TestDefineRouteExposesPatternAndParamNames(parseT *testing.T) {
	parseContract, parseErr := DefineRoute("users/:id/orders/:orderID")
	if parseErr != nil {
		parseT.Fatalf("expected route contract, got error: %v", parseErr)
	}
	if parseGot := parseContract.Pattern(); parseGot != "/users/:id/orders/:orderID" {
		parseT.Fatalf("expected normalized pattern, got %q", parseGot)
	}
	if parseGot2 := parseContract.ParamNames(); !reflect.DeepEqual(parseGot2, []string{"id", "orderID"}) {
		parseT.Fatalf("expected param names [id orderID], got %#v", parseGot2)
	}
}

func TestRouteContractPathBuildsAndEscapesSegments(parseT *testing.T) {
	parseContract := MustDefineRoute("/users/:id/orders/:orderID")
	parsePath, parseErr := parseContract.Path(map[string]string{
		"id":      "ada lovelace",
		"orderID": "PO/42",
	})
	if parseErr != nil {
		parseT.Fatalf("expected path build to succeed, got %v", parseErr)
	}
	if parsePath != "/users/ada%20lovelace/orders/PO%2F42" {
		parseT.Fatalf("expected escaped route path, got %q", parsePath)
	}
}

func TestRouteContractPathRejectsMissingAndUnexpectedParams(parseT *testing.T) {
	parseContract := MustDefineRoute("/users/:id")

	if _, parseErr := parseContract.Path(nil); parseErr == nil {
		parseT.Fatal("expected missing param error")
	}
	if _, parseErr2 := parseContract.Path(map[string]string{"id": "42", "slug": "ada"}); parseErr2 == nil {
		parseT.Fatal("expected extra param error")
	}
}

func TestRouteContractHrefForTypedProviders(parseT *testing.T) {
	parseContract := MustDefineRoute("/users/:id")
	parseHref, parseErr := parseContract.HrefFor(testUserRouteParams{ID: "42"}, testUserRouteQuery{
		Tab:   "billing",
		Page:  3,
		Draft: true,
	})
	if parseErr != nil {
		parseT.Fatalf("expected typed href build to succeed, got %v", parseErr)
	}
	if parseHref != "/users/42?draft=true&page=3&tab=billing" {
		parseT.Fatalf("expected typed href, got %q", parseHref)
	}
}

func TestDefineRouteRejectsCatchAllAndDuplicateParams(parseT *testing.T) {
	if _, parseErr := DefineRoute("*"); parseErr == nil {
		parseT.Fatal("expected catch-all contract definition to fail")
	}
	if _, parseErr2 := DefineRoute("/users/:id/orders/:id"); parseErr2 == nil {
		parseT.Fatal("expected duplicate params to fail")
	}
}

func TestRouteContractMustHelpersAndProviderHelpers(parseT *testing.T) {
	parseContract := MustDefineRoute("/users/:id")

	if parseGot := parseContract.MustPath(map[string]string{"id": "42"}); parseGot != "/users/42" {
		parseT.Fatalf("expected MustPath to build /users/42, got %q", parseGot)
	}
	if parseGot2 := parseContract.MustHref(map[string]string{"id": "42"}, url.Values{"tab": {"profile"}}); parseGot2 != "/users/42?tab=profile" {
		parseT.Fatalf("expected MustHref to encode query, got %q", parseGot2)
	}
	if parseGot3 := parseContract.MustPathFor(testUserRouteParams{ID: "42"}); parseGot3 != "/users/42" {
		parseT.Fatalf("expected MustPathFor to build /users/42, got %q", parseGot3)
	}
	if parseGot4 := parseContract.MustHrefFor(testUserRouteParams{ID: "42"}, testUserRouteQuery{Tab: "billing"}); parseGot4 != "/users/42?tab=billing" {
		parseT.Fatalf("expected MustHrefFor to build typed href, got %q", parseGot4)
	}
	if _, parseErr := parseContract.PathFor(nil); parseErr == nil {
		parseT.Fatal("expected PathFor(nil) to fail missing required params")
	}

	parseRoot := MustDefineRoute("/")
	if parseGot5, parseErr2 := parseRoot.PathFor(nil); parseErr2 != nil || parseGot5 != "/" {
		parseT.Fatalf("expected root PathFor(nil) to succeed, got path=%q err=%v", parseGot5, parseErr2)
	}
	if parseGot6, parseErr3 := parseRoot.HrefFor(nil, nil); parseErr3 != nil || parseGot6 != "/" {
		parseT.Fatalf("expected root HrefFor(nil,nil) to succeed, got href=%q err=%v", parseGot6, parseErr3)
	}
	if routeParamsFromProvider(nil) != nil {
		parseT.Fatal("expected nil route params provider to return nil map")
	}
	if routeQueryFromProvider(nil) != nil {
		parseT.Fatal("expected nil route query provider to return nil query")
	}
}

func TestRouteContractMustHelpersPanicOnInvalidInputs(parseT *testing.T) {
	parseAssertPanic := func(parseName string, parseFn func(), parseWant string) {
		parseT.Helper()
		defer func() {
			parseRecovered := recover()
			if parseRecovered == nil {
				parseT.Fatalf("%s: expected panic", parseName)
			}
			if parseWant == "" {
				return
			}
			if !strings.Contains(parseRecovered.(error).Error(), parseWant) {
				parseT.Fatalf("%s: expected panic to contain %q, got %v", parseName, parseWant, parseRecovered)
			}
		}()
		parseFn()
	}

	parseAssertPanic("MustDefineRoute", func() {
		_ = MustDefineRoute("*")
	}, "catch-all")
	parseAssertPanic("MustPath", func() {
		_ = MustDefineRoute("/users/:id").MustPath(nil)
	}, "requires non-empty param")
	parseAssertPanic("MustHref", func() {
		_ = MustDefineRoute("/users/:id").MustHref(nil, nil)
	}, "requires non-empty param")
	parseAssertPanic("MustPathFor", func() {
		_ = MustDefineRoute("/users/:id").MustPathFor(nil)
	}, "requires non-empty param")
	parseAssertPanic("MustHrefFor", func() {
		_ = MustDefineRoute("/users/:id").MustHrefFor(nil, nil)
	}, "requires non-empty param")
}

func TestDefineRouteValidationAndPatternNormalizationEdges(parseT *testing.T) {
	for _, parseTc := range []struct {
		name    string
		pattern string
		want    string
	}{
		{name: "blank hash normalizes to root", pattern: "#", want: "/"},
		{name: "trim and strip query", pattern: " users/:id ?tab=a", want: "/users/:id "},
		{name: "leading slash added", pattern: "users/:id", want: "/users/:id"},
		{name: "trailing slash removed", pattern: "/users/:id/", want: "/users/:id"},
	} {
		parseT.Run(parseTc.name, func(parseT2 *testing.T) {
			if parseGot := normalizeRouteContractPattern(parseTc.pattern); parseGot != parseTc.want {
				parseT2.Fatalf("normalizeRouteContractPattern(%q) = %q, want %q", parseTc.pattern, parseGot, parseTc.want)
			}
		})
	}

	for _, parsePattern := range []string{
		"",
		"/users//id",
		"/users/*",
		"/users/:",
		"/users/or*ders",
	} {
		if _, parseErr := DefineRoute(parsePattern); parseErr == nil {
			parseT.Fatalf("expected DefineRoute(%q) to fail validation", parsePattern)
		}
	}
}
