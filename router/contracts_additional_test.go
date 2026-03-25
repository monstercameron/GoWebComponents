package router

import (
	"net/url"
	"strings"
	"testing"
)

func TestRouteContractAdditionalBranches(t *testing.T) {
	rootContract, err := DefineRoute(" # ")
	if err != nil {
		t.Fatalf("DefineRoute(#) error = %v", err)
	}
	if rootContract.Pattern() != "/" {
		t.Fatalf("root contract Pattern() = %q, want /", rootContract.Pattern())
	}
	if path, err := rootContract.Path(nil); err != nil || path != "/" {
		t.Fatalf("root contract Path(nil) = %q, %v; want /, nil", path, err)
	}
	if href, err := rootContract.HrefFor(nil, nil); err != nil || href != "/" {
		t.Fatalf("root contract HrefFor(nil,nil) = %q, %v; want /, nil", href, err)
	}
	if names := rootContract.ParamNames(); names != nil {
		t.Fatalf("root contract ParamNames() = %#v, want nil", names)
	}

	patternCases := map[string]string{
		"users/:id":          "/users/:id",
		"#/users/:id?page=2": "/users/:id",
		"/users/:id/":        "/users/:id",
		"#":                  "/",
	}
	for input, want := range patternCases {
		if got := normalizeRouteContractPattern(input); got != want {
			t.Fatalf("normalizeRouteContractPattern(%q) = %q, want %q", input, got, want)
		}
	}

	for _, pattern := range []string{"", "*", "/users/:", "/docs/*/detail"} {
		if _, err := DefineRoute(pattern); err == nil {
			t.Fatalf("DefineRoute(%q) error = nil, want validation failure", pattern)
		}
	}

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("MustDefineRoute() did not panic for invalid pattern")
			}
		}()
		_ = MustDefineRoute("*")
	}()

	uninitialized := RouteContract{}
	if _, err := uninitialized.Path(map[string]string{"id": "42"}); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("Path(uninitialized) error = %v, want not initialized", err)
	}

	contract := MustDefineRoute("/users/:id")
	if _, err := contract.Path(map[string]string{" ": "42"}); err == nil || !strings.Contains(err.Error(), "does not define params") {
		t.Fatalf("Path(blank key) error = %v, want unexpected param error", err)
	}

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("MustPath() did not panic for missing param")
			}
		}()
		_ = contract.MustPath(nil)
	}()

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("MustHrefFor() did not panic for missing param")
			}
		}()
		_ = contract.MustHrefFor(nil, nil)
	}()

	if routeParamsFromProvider(nil) != nil {
		t.Fatal("routeParamsFromProvider(nil) should return nil")
	}
	if routeQueryFromProvider(nil) != nil {
		t.Fatal("routeQueryFromProvider(nil) should return nil")
	}

	href, err := contract.Href(map[string]string{"id": "42"}, url.Values{"tab": {"settings"}})
	if err != nil {
		t.Fatalf("Href() error = %v", err)
	}
	if href != "/users/42?tab=settings" {
		t.Fatalf("Href() = %q, want /users/42?tab=settings", href)
	}
}
