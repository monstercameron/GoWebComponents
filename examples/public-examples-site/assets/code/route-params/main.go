//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
)

func paramsPageView() ui.Node {
	parseNav := router.UseNavigate()
	parseParams := router.UseParams()
	parseUserID, hasID := parseParams.Int("id")
	parseEnabled, hasEnabled := parseParams.Bool("enabled")

	return shared.ExamplePage(
		"router.UseParams",
		"Read captured route parameters with typed helpers",
		"Params expose raw strings plus typed parsing helpers, which makes path-based detail routes easier to read without manual string splitting.",
		shared.ExamplePanel("Captured params",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("User 7 / enabled", ui.UseEvent(func() { parseNav.Navigate("/users/7/true") })),
				shared.ExampleButton("User 18 / disabled", ui.UseEvent(func() { parseNav.Navigate("/users/18/false") })),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("id", parseParams.Get("id")),
				shared.ExampleStat("enabled", parseParams.Get("enabled")),
				shared.ExampleStat("Int parsed", fmt.Sprintf("%t", hasID)),
				shared.ExampleStat("Bool parsed", fmt.Sprintf("%t", hasEnabled)),
			),
			html.Div(html.Props{Class: "mt-4 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("ID value", fmt.Sprintf("%d", parseUserID)),
				shared.ExampleStat("Enabled value", fmt.Sprintf("%t", parseEnabled)),
			),
		),
	)
}

func paramsPage(router.Attrs) *router.Element {
	return ui.CreateElement(paramsPageView)
}

func main() {
	utils.DisableAllDebug()
	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/users/7/true"})
	parseR.Register("/users/:id/:enabled", paramsPage)
	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
