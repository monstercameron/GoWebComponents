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

func (p testUserRouteParams) RouteParams() map[string]string {
	return map[string]string{"id": p.ID}
}

type testUserRouteQuery struct {
	Tab   string
	Page  int
	Draft bool
}

func (q testUserRouteQuery) RouteQuery() url.Values {
	values := url.Values{}
	if q.Tab != "" {
		values.Set("tab", q.Tab)
	}
	if q.Page > 0 {
		values.Set("page", strconv.Itoa(q.Page))
	}
	if q.Draft {
		values.Set("draft", "true")
	}
	return values
}

func TestDefineRouteExposesPatternAndParamNames(t *testing.T) {
	contract, err := DefineRoute("users/:id/orders/:orderID")
	if err != nil {
		t.Fatalf("expected route contract, got error: %v", err)
	}
	if got := contract.Pattern(); got != "/users/:id/orders/:orderID" {
		t.Fatalf("expected normalized pattern, got %q", got)
	}
	if got := contract.ParamNames(); !reflect.DeepEqual(got, []string{"id", "orderID"}) {
		t.Fatalf("expected param names [id orderID], got %#v", got)
	}
}

func TestRouteContractPathBuildsAndEscapesSegments(t *testing.T) {
	contract := MustDefineRoute("/users/:id/orders/:orderID")
	path, err := contract.Path(map[string]string{
		"id":      "ada lovelace",
		"orderID": "PO/42",
	})
	if err != nil {
		t.Fatalf("expected path build to succeed, got %v", err)
	}
	if path != "/users/ada%20lovelace/orders/PO%2F42" {
		t.Fatalf("expected escaped route path, got %q", path)
	}
}

func TestRouteContractPathRejectsMissingAndUnexpectedParams(t *testing.T) {
	contract := MustDefineRoute("/users/:id")

	if _, err := contract.Path(nil); err == nil {
		t.Fatal("expected missing param error")
	}
	if _, err := contract.Path(map[string]string{"id": "42", "slug": "ada"}); err == nil {
		t.Fatal("expected extra param error")
	}
}

func TestRouteContractHrefForTypedProviders(t *testing.T) {
	contract := MustDefineRoute("/users/:id")
	href, err := contract.HrefFor(testUserRouteParams{ID: "42"}, testUserRouteQuery{
		Tab:   "billing",
		Page:  3,
		Draft: true,
	})
	if err != nil {
		t.Fatalf("expected typed href build to succeed, got %v", err)
	}
	if href != "/users/42?draft=true&page=3&tab=billing" {
		t.Fatalf("expected typed href, got %q", href)
	}
}

func TestDefineRouteRejectsCatchAllAndDuplicateParams(t *testing.T) {
	if _, err := DefineRoute("*"); err == nil {
		t.Fatal("expected catch-all contract definition to fail")
	}
	if _, err := DefineRoute("/users/:id/orders/:id"); err == nil {
		t.Fatal("expected duplicate params to fail")
	}
}

func TestRouteContractMustHelpersAndProviderHelpers(t *testing.T) {
	contract := MustDefineRoute("/users/:id")

	if got := contract.MustPath(map[string]string{"id": "42"}); got != "/users/42" {
		t.Fatalf("expected MustPath to build /users/42, got %q", got)
	}
	if got := contract.MustHref(map[string]string{"id": "42"}, url.Values{"tab": {"profile"}}); got != "/users/42?tab=profile" {
		t.Fatalf("expected MustHref to encode query, got %q", got)
	}
	if got := contract.MustPathFor(testUserRouteParams{ID: "42"}); got != "/users/42" {
		t.Fatalf("expected MustPathFor to build /users/42, got %q", got)
	}
	if got := contract.MustHrefFor(testUserRouteParams{ID: "42"}, testUserRouteQuery{Tab: "billing"}); got != "/users/42?tab=billing" {
		t.Fatalf("expected MustHrefFor to build typed href, got %q", got)
	}
	if _, err := contract.PathFor(nil); err == nil {
		t.Fatal("expected PathFor(nil) to fail missing required params")
	}

	root := MustDefineRoute("/")
	if got, err := root.PathFor(nil); err != nil || got != "/" {
		t.Fatalf("expected root PathFor(nil) to succeed, got path=%q err=%v", got, err)
	}
	if got, err := root.HrefFor(nil, nil); err != nil || got != "/" {
		t.Fatalf("expected root HrefFor(nil,nil) to succeed, got href=%q err=%v", got, err)
	}
	if routeParamsFromProvider(nil) != nil {
		t.Fatal("expected nil route params provider to return nil map")
	}
	if routeQueryFromProvider(nil) != nil {
		t.Fatal("expected nil route query provider to return nil query")
	}
}

func TestRouteContractMustHelpersPanicOnInvalidInputs(t *testing.T) {
	assertPanic := func(name string, fn func(), want string) {
		t.Helper()
		defer func() {
			recovered := recover()
			if recovered == nil {
				t.Fatalf("%s: expected panic", name)
			}
			if want == "" {
				return
			}
			if !strings.Contains(recovered.(error).Error(), want) {
				t.Fatalf("%s: expected panic to contain %q, got %v", name, want, recovered)
			}
		}()
		fn()
	}

	assertPanic("MustDefineRoute", func() {
		_ = MustDefineRoute("*")
	}, "catch-all")
	assertPanic("MustPath", func() {
		_ = MustDefineRoute("/users/:id").MustPath(nil)
	}, "requires non-empty param")
	assertPanic("MustHref", func() {
		_ = MustDefineRoute("/users/:id").MustHref(nil, nil)
	}, "requires non-empty param")
	assertPanic("MustPathFor", func() {
		_ = MustDefineRoute("/users/:id").MustPathFor(nil)
	}, "requires non-empty param")
	assertPanic("MustHrefFor", func() {
		_ = MustDefineRoute("/users/:id").MustHrefFor(nil, nil)
	}, "requires non-empty param")
}

func TestDefineRouteValidationAndPatternNormalizationEdges(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pattern string
		want    string
	}{
		{name: "blank hash normalizes to root", pattern: "#", want: "/"},
		{name: "trim and strip query", pattern: " users/:id ?tab=a", want: "/users/:id "},
		{name: "leading slash added", pattern: "users/:id", want: "/users/:id"},
		{name: "trailing slash removed", pattern: "/users/:id/", want: "/users/:id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeRouteContractPattern(tc.pattern); got != tc.want {
				t.Fatalf("normalizeRouteContractPattern(%q) = %q, want %q", tc.pattern, got, tc.want)
			}
		})
	}

	for _, pattern := range []string{
		"",
		"/users//id",
		"/users/*",
		"/users/:",
		"/users/or*ders",
	} {
		if _, err := DefineRoute(pattern); err == nil {
			t.Fatalf("expected DefineRoute(%q) to fail validation", pattern)
		}
	}
}
