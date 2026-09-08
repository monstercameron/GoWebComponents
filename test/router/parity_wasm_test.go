//go:build js && wasm

package routertest_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	appRouter "github.com/monstercameron/GoWebComponents/v6/router"
	routertest "github.com/monstercameron/GoWebComponents/v6/test/router"
	base "github.com/monstercameron/GoWebComponents/v6/testkit/router"
)

func parityRoute(_ appRouter.Attrs) *appRouter.Element {
	parseParams := appRouter.UseParams()
	parseQuery := appRouter.UseQuery()
	return html.Div(html.Props{ID: "parity-route"},
		html.Text(fmt.Sprintf("item:%s|tab:%s", parseParams.Get("id"), parseQuery.Get("tab"))),
	)
}

func parityHome(_ appRouter.Attrs) *appRouter.Element {
	return html.Div(html.Props{ID: "home-route"}, html.Text("home"))
}

func TestPreferredRouterWrappersMatchCompatibilityAliasBehavior(parseT *testing.T) {
	parsePreferred := routertest.NewHash(parseT)
	parsePreferred.Register("/", parityHome)
	parsePreferred.Register("/items/:id", parityRoute)
	parsePreferred.SetPath("/items/42?tab=history")
	parsePreferredText := parsePreferred.ByID("parity-route").Text()
	parsePreferredPath := parsePreferred.Path()
	parsePreferredQuery := parsePreferred.Query().Get("tab")
	parsePreferred.Cleanup()

	parseCompat := base.NewHash(parseT)
	parseCompat.Register("/", parityHome)
	parseCompat.Register("/items/:id", parityRoute)
	parseCompat.SetPath("/items/42?tab=history")
	parseCompatText := parseCompat.ByID("parity-route").Text()
	parseCompatPath := parseCompat.Path()
	parseCompatQuery := parseCompat.Query().Get("tab")
	parseCompat.Cleanup()

	if parsePreferredText != parseCompatText || parsePreferredPath != parseCompatPath || parsePreferredQuery != parseCompatQuery {
		parseT.Fatalf("expected preferred wrapper and compatibility alias to match, got preferred=(%q,%q,%q) compat=(%q,%q,%q)", parsePreferredText, parsePreferredPath, parsePreferredQuery, parseCompatText, parseCompatPath, parseCompatQuery)
	}
}
