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
	params := appRouter.UseParams()
	query := appRouter.UseQuery()
	return html.Div(html.Props{ID: "product-route"},
		html.Text(fmt.Sprintf("product:%s|sort:%s", params.Get("sku"), query.Get("sort"))),
	)
}

func homeRouteExample(_ appRouter.Attrs) *appRouter.Element {
	return html.Div(html.Props{ID: "home-route"}, html.Text("home"))
}

func TestConsumerRouterPattern_HashFixture(t *testing.T) {
	fixture := routertest.NewHash(t)
	fixture.Register("/", homeRouteExample)
	fixture.Register("/products/:sku", productRouteExample)
	fixture.SetPath("/products/sku-42?sort=price")
	fixture.Render()

	if got := fixture.Params()["sku"]; got != "sku-42" {
		t.Fatalf("expected route param to be available, got %q", got)
	}
	if got := fixture.Query().Get("sort"); got != "price" {
		t.Fatalf("expected query param to be available, got %q", got)
	}
	if got := fixture.Path(); got != "/products/sku-42" {
		t.Fatalf("expected current path /products/sku-42, got %q", got)
	}
}
