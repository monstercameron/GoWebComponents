package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

func DeepTree(props dom.Attrs) *dom.Element {
	depth := props["depth"].(int)
	if depth <= 0 {
		return dom.Div(dom.ClassIdProps("leaf", "deep-leaf"), dom.Text("Leaf"))
	}
	return dom.Div(dom.ClassProps("node"), dom.CreateElement(DeepTree, dom.Attrs{"depth": depth - 1}))
}

func ManyHooks(props dom.Attrs) *dom.Element {
	// Simulate a component with many hooks
	for i := 0; i < 50; i++ {
		hooks.UseState(i)
		hooks.UseEffect(func() func() { return nil })
		hooks.UseMemo(func() interface{} { return i * 2 }, i)
	}
	return dom.Div(dom.ClassProps("hook-node"), dom.Text("Hooks"))
}

func BenchmarkApp(props dom.Attrs) *dom.Element {
	items, setItems := hooks.UseState([]string{})
	view, setView := hooks.UseState("list") // list, deep, hooks
	lastRenderTime, setLastRenderTime := hooks.UseState("")
	computeResult, setComputeResult := hooks.UseState("")

	// Measure render time
	hooks.UseEffect(func() func() {
		setLastRenderTime(time.Now().Format(time.RFC3339Nano))
		return nil
	}, items(), view())

	computePrimes := hooks.GoUseFunc(func() {
		start := time.Now()
		count := 0
		for i := 2; i < 200000; i++ {
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
		setComputeResult(fmt.Sprintf("Found %d primes in %dms", count, duration.Milliseconds()))
	})

	renderList := hooks.GoUseFunc(func() {
		setView("list")
		newItems := make([]string, 1000)
		for i := 0; i < 1000; i++ {
			newItems[i] = "Item " + strconv.Itoa(i)
		}
		setItems(newItems)
	})

	renderDeep := hooks.GoUseFunc(func() {
		setView("deep")
		setItems([]string{})
	})

	renderHooks := hooks.GoUseFunc(func() {
		setView("hooks")
		setItems([]string{})
	})

	clearList := hooks.GoUseFunc(func() {
		setView("list")
		setItems([]string{})
	})

	updateList := hooks.GoUseFunc(func() {
		current := items()
		newItems := make([]string, len(current))
		for i, item := range current {
			newItems[i] = item + " (Updated)"
		}
		setItems(newItems)
	})

	return dom.Div(dom.Attrs{"id": "app"},
		dom.H1(nil, dom.Text("Benchmark App")),
		dom.Div(dom.Attrs{"id": "controls"},
			dom.Button(dom.Attrs{"id": "btn-render", "onclick": renderList}, dom.Text("Render 1000 Items")),
			dom.Button(dom.Attrs{"id": "btn-update", "onclick": updateList}, dom.Text("Update Items")),
			dom.Button(dom.Attrs{"id": "btn-clear", "onclick": clearList}, dom.Text("Clear List")),
			dom.Button(dom.Attrs{"id": "btn-deep", "onclick": renderDeep}, dom.Text("Render Deep Tree (500)")),
			dom.Button(dom.Attrs{"id": "btn-hooks", "onclick": renderHooks}, dom.Text("Render 100 Components w/ 150 Hooks")),
			dom.Button(dom.Attrs{"id": "btn-compute", "onclick": computePrimes}, dom.Text("Compute Primes (200k)")),
		),
		dom.Div(dom.Attrs{"id": "metrics"},
			dom.P(dom.Attrs{"id": "last-render"}, dom.Text("Last Render: "+lastRenderTime())),
			dom.P(dom.Attrs{"id": "item-count"}, dom.Text("Count: "+strconv.Itoa(len(items())))),
			dom.P(dom.Attrs{"id": "compute-result"}, dom.Text(computeResult())),
		),
		dom.Div(dom.Attrs{"id": "container"},
			func() *dom.Element {
				if view() == "deep" {
					return dom.CreateElement(DeepTree, dom.Attrs{"depth": 100})
				} else if view() == "hooks" {
					var children []interface{}
					for i := 0; i < 100; i++ {
						children = append(children, dom.CreateElement(ManyHooks, nil))
					}
					return dom.Div(dom.IdProps("hooks-container"), children...)
				} else {
					var children []interface{}
					for _, item := range items() {
						children = append(children, dom.Div(dom.Attrs{"class": "list-item"}, dom.Text(item)))
					}
					return dom.Div(dom.IdProps("list-container"), children...)
				}
			}(),
		),
	)
}

func main() {
	render.To(dom.CreateElement(BenchmarkApp, nil), "body")
	select {}
}
