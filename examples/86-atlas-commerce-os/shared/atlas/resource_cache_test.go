package atlas

import (
	"net/url"
	"testing"
)

func TestRoutePayloadResourceKeyFiltersNoticeAndBootstrapQuery(t *testing.T) {
	query := url.Values{}
	query.Set("warehouse", "east")
	query.Set("atlas_notice", "inventory-updated")
	query.Set("atlas_bootstrap", "external")

	key := RoutePayloadResourceKey("/app/inventory", query)
	if got, want := key, "atlas:route:/app/inventory?warehouse=east"; got != want {
		t.Fatalf("RoutePayloadResourceKey() = %q, want %q", got, want)
	}
}

func TestRoutePayloadResourceKeyNormalizesQueryOrderForNavigationReuse(t *testing.T) {
	left := url.Values{}
	left.Add("warehouse", "east")
	left.Add("status", "healthy")

	right := url.Values{}
	right.Add("status", "healthy")
	right.Add("warehouse", "east")

	if got, want := RoutePayloadResourceKey("/app/inventory", left), RoutePayloadResourceKey("/app/inventory", right); got != want {
		t.Fatalf("expected query order to normalize for repeat navigation, got %q and %q", got, want)
	}
}

func TestPayloadResourceKeysUseBootstrapFallbackData(t *testing.T) {
	payload := Payload{
		Route: RouteBootstrap{
			Path:  "/shop/widget",
			Query: map[string][]string{"view": {"full"}, "atlas_notice": {"comment-submitted"}},
		},
		Data: map[string]any{
			"page": map[string]any{"slug": "widget"},
		},
		Requests: map[string]Request{
			"page": {
				Method: "GET",
				URL:    "/api/atlas/route-data?path=%2Fshop%2Fwidget",
				Status: 200,
			},
			"related": {
				Method: "GET",
				URL:    "/api/public/products/widget/related-products",
				Status: 200,
				Data: map[string]any{
					"items": []any{"alt-widget"},
				},
			},
		},
	}

	keys := PayloadResourceKeys(payload)
	expected := []string{
		"atlas:route:/shop/widget?view=full",
		"atlas:request:/api/atlas/route-data?path=%2Fshop%2Fwidget::page",
		"atlas:request:/api/public/products/widget/related-products::items",
	}
	for _, key := range expected {
		if _, ok := keys[key]; !ok {
			t.Fatalf("expected payload resource keys to include %q", key)
		}
	}
	if _, ok := keys["atlas:route:/shop/widget?atlas_notice=comment-submitted&view=full"]; ok {
		t.Fatalf("expected notice query to be filtered from route cache key")
	}
}
