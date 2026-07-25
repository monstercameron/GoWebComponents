//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

func lazyExample() ui.Node {
	parseVersion := ui.UseState(1)
	parseNext := ui.UseEvent(func() { parseVersion.Update(func(parsePrev int) int { return parsePrev + 1 }) })

	parseCard := ui.CreateElement(ui.Lazy, ui.LazyProps{
		Loader: func(parseCtx context.Context) (ui.Node, error) {
			select {
			case <-parseCtx.Done():
				return nil, parseCtx.Err()
			case <-time.After(350 * time.Millisecond):
			}

			return html.Div(html.Props{Class: "rounded-2xl border border-emerald-400/30 bg-emerald-400/10 p-6 text-emerald-50"},
				html.H3(html.Props{Class: "text-2xl font-bold"}, html.Text(fmt.Sprintf("Lazy payload %d", parseVersion.Get()))),
				html.P(html.Props{Class: "mt-3"}, html.Text("The subtree was resolved asynchronously through ui.Lazy.")),
			), nil
		},
		Dependencies: []interface{}{parseVersion.Get()},
		Delay:        50 * time.Millisecond,
		Fallback:     html.Div(html.Props{Class: "rounded-2xl border border-cyan-400/30 bg-cyan-400/10 p-6 text-cyan-50"}, html.Text("Resolving lazy subtree...")),
		ErrorFallback: func(parseErr error) ui.Node {
			return html.Div(html.Props{Class: "rounded-2xl border border-red-400/30 bg-red-400/10 p-6 text-red-50"}, html.Text(parseErr.Error()))
		},
	})

	return shared.ExamplePage(
		"ui.Lazy",
		"Resolve a subtree asynchronously through AsyncBoundary semantics",
		"Lazy is a convenience layer over async node loading plus fallback rendering. Changing the dependency state causes the loader to resolve a fresh subtree.",
		shared.ExamplePanel("Lazy subtree",
			html.Div(html.Props{Class: "mt-3 flex gap-3"}, shared.ExampleButton("Load next payload", parseNext)),
			html.Div(html.Props{Class: "mt-6"}, parseCard),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(lazyExample))
	exampleboot.WaitExampleRuntime()
}
