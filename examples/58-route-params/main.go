//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func paramsPageView() ui.Node {
	nav := router.UseNavigate()
	params := router.UseParams()
	userID, hasID := params.Int("id")
	enabled, hasEnabled := params.Bool("enabled")

	return shared.ExamplePage(
		"router.UseParams",
		"Read captured route parameters with typed helpers",
		"Params expose raw strings plus typed parsing helpers, which makes path-based detail routes easier to read without manual string splitting.",
		shared.ExamplePanel("Captured params",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("User 7 / enabled", ui.UseEvent(func() { nav.Navigate("/users/7/true") })),
				shared.ExampleButton("User 18 / disabled", ui.UseEvent(func() { nav.Navigate("/users/18/false") })),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("id", params.Get("id")),
				shared.ExampleStat("enabled", params.Get("enabled")),
				shared.ExampleStat("Int parsed", fmt.Sprintf("%t", hasID)),
				shared.ExampleStat("Bool parsed", fmt.Sprintf("%t", hasEnabled)),
			),
			html.Div(html.Props{Class: "mt-4 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("ID value", fmt.Sprintf("%d", userID)),
				shared.ExampleStat("Enabled value", fmt.Sprintf("%t", enabled)),
			),
		),
	)
}

func paramsPage(router.Attrs) *router.Element {
	return ui.CreateElement(paramsPageView)
}

func main() {
	utils.DisableAllDebug()
	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/users/7/true"})
	r.Register("/users/:id/:enabled", paramsPage)
	r.Mount("#app")
	select {}
}
