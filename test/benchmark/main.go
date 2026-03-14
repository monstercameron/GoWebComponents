//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	listSize           = 10
	deepTreeDepth      = 60
	hookComponentCount = 40
	hooksPerComponent  = 20
	primeLimit         = 10000
)

type DeepTreeProps struct {
	Depth int
}

func DeepTree(props DeepTreeProps) ui.Node {
	if props.Depth <= 0 {
		return html.Div(html.Props{Class: "leaf", ID: "deep-leaf"}, html.Text("Leaf"))
	}

	return html.Div(
		html.Props{Class: "node"},
		ui.CreateElement(DeepTree, DeepTreeProps{Depth: props.Depth - 1}),
	)
}

func ManyHooks() ui.Node {
	for i := 0; i < hooksPerComponent; i++ {
		ui.UseState(i)
		ui.UseEffect(func() func() { return nil })
		ui.UseMemo(func() int { return i * 2 }, i)
	}

	return html.Div(html.Props{Class: "hook-node"}, html.Text("Hooks"))
}

func BenchmarkApp() ui.Node {
	items := ui.UseState([]string{})
	view := ui.UseState("list")
	lastRenderTime := ui.UseState("")
	computeResult := ui.UseState("")

	ui.UseEffect(func() func() {
		lastRenderTime.Set(time.Now().Format(time.RFC3339Nano))
		return nil
	}, items.Get(), view.Get())

	computePrimes := ui.UseEvent(func() {
		start := time.Now()
		count := 0
		for i := 2; i < primeLimit; i++ {
			isPrime := true
			for j := 2; j*j <= i; j++ {
				if i%j == 0 {
					isPrime = false
					break
				}
			}
			if isPrime {
				count++
			}
		}
		duration := time.Since(start)
		computeResult.Set(fmt.Sprintf("Found %d primes in %dms", count, duration.Milliseconds()))
	})

	renderList := ui.UseEvent(func() {
		view.Set("list")
		newItems := make([]string, listSize)
		for i := 0; i < listSize; i++ {
			newItems[i] = "Item " + strconv.Itoa(i)
		}
		items.Set(newItems)
	})

	renderDeep := ui.UseEvent(func() {
		view.Set("deep")
		items.Set([]string{})
	})

	renderHooks := ui.UseEvent(func() {
		view.Set("hooks")
		items.Set([]string{})
	})

	clearList := ui.UseEvent(func() {
		view.Set("list")
		items.Set([]string{})
	})

	updateList := ui.UseEvent(func() {
		current := items.Get()
		newItems := make([]string, len(current))
		for i, item := range current {
			newItems[i] = item + " (Updated)"
		}
		items.Set(newItems)
	})

	var content ui.Node
	switch view.Get() {
	case "deep":
		content = ui.CreateElement(DeepTree, DeepTreeProps{Depth: deepTreeDepth})
	case "hooks":
		children := make([]ui.Node, 0, hookComponentCount)
		for i := 0; i < hookComponentCount; i++ {
			children = append(children, ui.CreateElement(ManyHooks))
		}
		content = html.Div(html.Props{ID: "hooks-container"}, children...)
	default:
		children := make([]ui.Node, 0, len(items.Get()))
		for _, item := range items.Get() {
			children = append(children, html.Div(html.Props{Class: "list-item"}, html.Text(item)))
		}
		content = html.Div(html.Props{ID: "list-container"}, children...)
	}

	return html.Div(html.Props{ID: "app"},
		html.H1(html.Props{}, html.Text("Benchmark App")),
		html.Div(html.Props{ID: "controls"},
			html.Button(html.Props{ID: "btn-render", OnClick: renderList}, html.Text("Render 10 Items")),
			html.Button(html.Props{ID: "btn-update", OnClick: updateList}, html.Text("Update Items")),
			html.Button(html.Props{ID: "btn-clear", OnClick: clearList}, html.Text("Clear List")),
			html.Button(html.Props{ID: "btn-deep", OnClick: renderDeep}, html.Text("Render Deep Tree (60)")),
			html.Button(html.Props{ID: "btn-hooks", OnClick: renderHooks}, html.Text("Render 40 Components w/ 60 Hooks")),
			html.Button(html.Props{ID: "btn-compute", OnClick: computePrimes}, html.Text("Compute Primes (10k)")),
		),
		html.Div(html.Props{ID: "metrics"},
			html.P(html.Props{ID: "last-render"}, html.Text("Last Render: "+lastRenderTime.Get())),
			html.P(html.Props{ID: "item-count"}, html.Text("Count: "+strconv.Itoa(len(items.Get())))),
			html.P(html.Props{ID: "compute-result"}, html.Text(computeResult.Get())),
		),
		html.Div(html.Props{ID: "container"}, content),
	)
}

func main() {
	ui.Render(ui.CreateElement(BenchmarkApp), "body")
	select {}
}
