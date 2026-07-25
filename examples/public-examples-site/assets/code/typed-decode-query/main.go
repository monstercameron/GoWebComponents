//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"net/url"

	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/router"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// listParams is decoded from a query string by tag, then validated by the same struct tags — one
// typed read instead of scattered q.Get + strconv + bounds checks.
type listParams struct {
	Page int    `query:"page" validate:"gte=1"`
	Sort string `query:"sort" validate:"oneof=new top"`
}

// decodeQueryExample demonstrates router.DecodeQuery: edit the raw query string and watch it decode
// into a typed, validated struct (or surface a validation error for a bad value).
func decodeQueryExample() ui.Node {
	parseRaw := ui.UseState("page=2&sort=top")

	parseValues, parseParseErr := url.ParseQuery(parseRaw.Get())
	parseResult := ""
	if parseParseErr != nil {
		parseResult = "malformed query string: " + parseParseErr.Error()
	} else if parseParams, parseErr := router.DecodeQuery[listParams](parseValues); parseErr != nil {
		parseResult = "validation failed: " + parseErr.Error()
	} else {
		parseResult = fmt.Sprintf("Page=%d  Sort=%q", parseParams.Page, parseParams.Sort)
	}

	return shared.ExamplePage(
		"router.DecodeQuery",
		"Typed, validated search params from a struct",
		"DecodeQuery reads query:\"...\" tags into a typed struct and validates it with the same validate:\"...\" tags. Try page=0 (fails gte=1) or sort=hot (fails oneof=new top) to see a validation error instead of a silent miss.",
		shared.ExamplePanel("Decode a query string",
			html.Label(html.Props{Class: "mt-3 block text-sm text-slate-300"}, html.Text("Query string")),
			html.Input(html.PropsOf(
				html.Class("mt-1 w-full rounded-lg border border-slate-600 bg-slate-900 px-3 py-2 font-mono text-slate-100"),
				html.Bind(parseRaw),
			)),
			html.P(html.Props{Class: "mt-6 text-lg text-slate-100"}, html.Text(parseResult)),
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(decodeQueryExample))
	exampleboot.WaitExampleRuntime()
}
