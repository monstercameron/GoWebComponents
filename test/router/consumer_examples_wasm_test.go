//go:build js && wasm
// +build js,wasm

package routertest_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	appRouter "github.com/monstercameron/GoWebComponents/router"
	routertest "github.com/monstercameron/GoWebComponents/test/router"
)

func productRouteExample(_ appRouter.Attrs) *appRouter.Element {
	parseParams := appRouter.UseParams()
	parseQuery := appRouter.UseQuery()
	return html.Div(html.Props{ID: "product-route"},
		html.Text(fmt.Sprintf("product:%s|sort:%s", parseParams.Get("sku"), parseQuery.Get("sort"))),
	)
}

func homeRouteExample(_ appRouter.Attrs) *appRouter.Element {
	return html.Div(html.Props{ID: "home-route"}, html.Text("home"))
}

func TestConsumerRouterPattern_HashFixture(parseT *testing.T) {
	parseFixture := routertest.NewHash(parseT)
	parseFixture.Register("/", homeRouteExample)
	parseFixture.Register("/products/:sku", productRouteExample)
	parseFixture.SetPath("/products/sku-42?sort=price")
	parseFixture.Render()

	if parseGot := parseFixture.Params()["sku"]; parseGot != "sku-42" {
		parseT.Fatalf("expected route param to be available, got %q", parseGot)
	}
	if parseGot2 := parseFixture.Query().Get("sort"); parseGot2 != "price" {
		parseT.Fatalf("expected query param to be available, got %q", parseGot2)
	}
	if parseGot3 := parseFixture.Path(); parseGot3 != "/products/sku-42" {
		parseT.Fatalf("expected current path /products/sku-42, got %q", parseGot3)
	}
}
