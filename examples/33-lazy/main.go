//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func lazyExample() ui.Node {
	version := ui.UseState(1)
	next := ui.UseEvent(func() { version.Update(func(prev int) int { return prev + 1 }) })

	card := ui.CreateElement(ui.Lazy, ui.LazyProps{
		Loader: func(ctx context.Context) (ui.Node, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(350 * time.Millisecond):
			}

			return html.Div(html.Props{Class: "rounded-2xl border border-emerald-400/30 bg-emerald-400/10 p-6 text-emerald-50"},
				html.H3(html.Props{Class: "text-2xl font-bold"}, html.Text(fmt.Sprintf("Lazy payload %d", version.Get()))),
				html.P(html.Props{Class: "mt-3"}, html.Text("The subtree was resolved asynchronously through ui.Lazy.")),
			), nil
		},
		Dependencies: []interface{}{version.Get()},
		Delay:        50 * time.Millisecond,
		Fallback:     html.Div(html.Props{Class: "rounded-2xl border border-cyan-400/30 bg-cyan-400/10 p-6 text-cyan-50"}, html.Text("Resolving lazy subtree...")),
		ErrorFallback: func(err error) ui.Node {
			return html.Div(html.Props{Class: "rounded-2xl border border-red-400/30 bg-red-400/10 p-6 text-red-50"}, html.Text(err.Error()))
		},
	})

	return shared.ExamplePage(
		"ui.Lazy",
		"Resolve a subtree asynchronously through AsyncBoundary semantics",
		"Lazy is a convenience layer over async node loading plus fallback rendering. Changing the dependency state causes the loader to resolve a fresh subtree.",
		shared.ExamplePanel("Lazy subtree",
			html.Div(html.Props{Class: "mt-3 flex gap-3"}, shared.ExampleButton("Load next payload", next)),
			html.Div(html.Props{Class: "mt-6"}, card),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(lazyExample), "#app")
	select {}
}
