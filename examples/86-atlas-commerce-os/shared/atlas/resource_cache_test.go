package atlas

import (
	"net/url"
	"testing"
)

func TestRoutePayloadResourceKeyFiltersNoticeAndBootstrapQuery(parseT *testing.T) {
	parseQuery := url.Values{}
	parseQuery.Set("warehouse", "east")
	parseQuery.Set("atlas_notice", "inventory-updated")
	parseQuery.Set("atlas_bootstrap", "external")

	parseKey := RoutePayloadResourceKey("/app/inventory", parseQuery)
	if parseGot, parseWant := parseKey, "atlas:route:/app/inventory?warehouse=east"; parseGot != parseWant {
		parseT.Fatalf("RoutePayloadResourceKey() = %q, want %q", parseGot, parseWant)
	}
}

func TestRoutePayloadResourceKeyNormalizesQueryOrderForNavigationReuse(parseT *testing.T) {
	parseLeft := url.Values{}
	parseLeft.Add("warehouse", "east")
	parseLeft.Add("status", "healthy")

	parseRight := url.Values{}
	parseRight.Add("status", "healthy")
	parseRight.Add("warehouse", "east")

	if parseGot, parseWant := RoutePayloadResourceKey("/app/inventory", parseLeft), RoutePayloadResourceKey("/app/inventory", parseRight); parseGot != parseWant {
		parseT.Fatalf("expected query order to normalize for repeat navigation, got %q and %q", parseGot, parseWant)
	}
}

func TestPayloadResourceKeysUseBootstrapFallbackData(parseT *testing.T) {
	parsePayload := Payload{
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

	parseKeys := PayloadResourceKeys(parsePayload)
	parseExpected := []string{
		"atlas:route:/shop/widget?view=full",
		"atlas:request:/api/atlas/route-data?path=%2Fshop%2Fwidget::page",
		"atlas:request:/api/public/products/widget/related-products::items",
	}
	for _, parseKey := range parseExpected {
		if _, parseOk := parseKeys[parseKey]; !parseOk {
			parseT.Fatalf("expected payload resource keys to include %q", parseKey)
		}
	}
	if _, parseOk2 := parseKeys["atlas:route:/shop/widget?atlas_notice=comment-submitted&view=full"]; parseOk2 {
		parseT.Fatalf("expected notice query to be filtered from route cache key")
	}
}
