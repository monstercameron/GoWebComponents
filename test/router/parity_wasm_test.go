//go:build js && wasm
// +build js,wasm

package routertest_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	appRouter "github.com/monstercameron/GoWebComponents/router"
	routertest "github.com/monstercameron/GoWebComponents/test/router"
	base "github.com/monstercameron/GoWebComponents/testkit/router"
)

func parityRoute(_ appRouter.Attrs) *appRouter.Element {
	params := appRouter.UseParams()
	query := appRouter.UseQuery()
	return html.Div(html.Props{ID: "parity-route"},
		html.Text(fmt.Sprintf("item:%s|tab:%s", params.Get("id"), query.Get("tab"))),
	)
}

func parityHome(_ appRouter.Attrs) *appRouter.Element {
	return html.Div(html.Props{ID: "home-route"}, html.Text("home"))
}

func TestPreferredRouterWrappersMatchCompatibilityAliasBehavior(t *testing.T) {
	preferred := routertest.NewHash(t)
	preferred.Register("/", parityHome)
	preferred.Register("/items/:id", parityRoute)
	preferred.SetPath("/items/42?tab=history")
	preferredText := preferred.ByID("parity-route").Text()
	preferredPath := preferred.Path()
	preferredQuery := preferred.Query().Get("tab")
	preferred.Cleanup()

	compat := base.NewHash(t)
	compat.Register("/", parityHome)
	compat.Register("/items/:id", parityRoute)
	compat.SetPath("/items/42?tab=history")
	compatText := compat.ByID("parity-route").Text()
	compatPath := compat.Path()
	compatQuery := compat.Query().Get("tab")
	compat.Cleanup()

	if preferredText != compatText || preferredPath != compatPath || preferredQuery != compatQuery {
		t.Fatalf("expected preferred wrapper and compatibility alias to match, got preferred=(%q,%q,%q) compat=(%q,%q,%q)", preferredText, preferredPath, preferredQuery, compatText, compatPath, compatQuery)
	}
}
