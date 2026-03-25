package router

import (
	"net/url"
	"reflect"
	"strconv"
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
