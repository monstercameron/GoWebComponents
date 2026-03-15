//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const (
	coreListSize       = 40
	contentCardCount   = 12
	deepTreeDepth      = 60
	hookComponentCount = 40
	hooksPerComponent  = 20
	primeLimit         = 10000
)

type ContentCardData struct {
	ID      int
	Title   string
	Summary string
	Status  string
	Meta    string
	Tags    []string
}

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

type ContentCardProps struct {
	Item ContentCardData
}

func ContentCard(props ContentCardProps) ui.Node {
	tagChildren := make([]ui.Node, 0, len(props.Item.Tags))
	for _, tag := range props.Item.Tags {
		tagChildren = append(tagChildren,
			html.Span(html.Props{Class: "content-tag"}, html.Text(tag)),
		)
	}

	return html.Article(html.Props{Class: "content-card"},
		html.Div(html.Props{Class: "content-card-header"},
			html.H2(html.Props{Class: "content-title"}, html.Text(props.Item.Title)),
			html.Span(html.Props{Class: "content-status"}, html.Text(props.Item.Status)),
		),
		html.P(html.Props{Class: "content-summary"}, html.Text(props.Item.Summary)),
		html.Div(html.Props{Class: "content-meta"},
			html.Span(html.Props{Class: "content-meta-text"}, html.Text(props.Item.Meta)),
		),
		html.Div(html.Props{Class: "content-tags"}, tagChildren...),
	)
}

func buildCoreItems() []string {
	items := make([]string, coreListSize)
	for i := 0; i < coreListSize; i++ {
		items[i] = "Item " + strconv.Itoa(i)
	}
	return items
}

func buildContentItems() []ContentCardData {
	items := make([]ContentCardData, contentCardCount)
	for i := 0; i < contentCardCount; i++ {
		items[i] = ContentCardData{
			ID:      i,
			Title:   "Article " + strconv.Itoa(i),
			Summary: "This benchmark card exercises regular app rendering with nested content blocks.",
			Status:  "draft",
			Meta:    "Section " + strconv.Itoa((i%3)+1),
			Tags: []string{
				"perf",
				"bench",
				"card-" + strconv.Itoa(i%4),
			},
		}
	}
	return items
}

func BenchmarkApp() ui.Node {
	coreItems := ui.UseState([]string{})
	contentItems := ui.UseState([]ContentCardData{})
	view := ui.UseState("core")
	lastRenderTime := ui.UseState("")
	computeResult := ui.UseState("")

	ui.UseEffect(func() func() {
		lastRenderTime.Set(time.Now().Format(time.RFC3339Nano))
		return nil
	}, coreItems.Get(), contentItems.Get(), view.Get())

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
		view.Set("core")
		coreItems.Set(buildCoreItems())
		contentItems.Set([]ContentCardData{})
	})

	renderContent := ui.UseEvent(func() {
		view.Set("content")
		contentItems.Set(buildContentItems())
		coreItems.Set([]string{})
	})

	renderDeep := ui.UseEvent(func() {
		view.Set("deep")
		coreItems.Set([]string{})
		contentItems.Set([]ContentCardData{})
	})

	renderHooks := ui.UseEvent(func() {
		view.Set("hooks")
		coreItems.Set([]string{})
		contentItems.Set([]ContentCardData{})
	})

	clearList := ui.UseEvent(func() {
		view.Set("core")
		coreItems.Set([]string{})
	})

	clearContent := ui.UseEvent(func() {
		view.Set("content")
		contentItems.Set([]ContentCardData{})
	})

	updateList := ui.UseEvent(func() {
		coreItems.Update(func(current []string) []string {
			newItems := make([]string, len(current))
			for i, item := range current {
				newItems[i] = item + " (Updated)"
			}
			return newItems
		})
	})

	updateContent := ui.UseEvent(func() {
		contentItems.Update(func(current []ContentCardData) []ContentCardData {
			next := make([]ContentCardData, len(current))
			for i, item := range current {
				tags := make([]string, len(item.Tags))
				copy(tags, item.Tags)
				next[i] = ContentCardData{
					ID:      item.ID,
					Title:   item.Title + " (Updated)",
					Summary: item.Summary + " Updated with fresh content.",
					Status:  "live",
					Meta:    item.Meta + " / refreshed",
					Tags:    tags,
				}
			}
			return next
		})
	})

	var content ui.Node
	switch view.Get() {
	case "deep":
		content = ui.CreateElement(DeepTree, DeepTreeProps{Depth: deepTreeDepth})
	case "content":
		children := make([]ui.Node, 0, len(contentItems.Get()))
		for _, item := range contentItems.Get() {
			children = append(children, ui.CreateElement(ContentCard, ContentCardProps{Item: item}))
		}
		content = html.Div(html.Props{ID: "content-container"}, children...)
	case "hooks":
		children := make([]ui.Node, 0, hookComponentCount)
		for i := 0; i < hookComponentCount; i++ {
			children = append(children, ui.CreateElement(ManyHooks))
		}
		content = html.Div(html.Props{ID: "hooks-container"}, children...)
	default:
		children := make([]ui.Node, 0, len(coreItems.Get()))
		for _, item := range coreItems.Get() {
			children = append(children, html.Div(html.Props{Class: "core-list-item"}, html.Text(item)))
		}
		content = html.Div(html.Props{ID: "core-list-container"}, children...)
	}

	return html.Div(html.Props{ID: "app"},
		html.H1(html.Props{}, html.Text("Benchmark App")),
		html.Div(html.Props{ID: "controls"},
			html.Button(html.Props{ID: "btn-render", OnClick: renderList}, html.Text("Render Core Items")),
			html.Button(html.Props{ID: "btn-update", OnClick: updateList}, html.Text("Update Core Items")),
			html.Button(html.Props{ID: "btn-clear", OnClick: clearList}, html.Text("Clear Core Items")),
			html.Button(html.Props{ID: "btn-content-render", OnClick: renderContent}, html.Text("Render Content Cards")),
			html.Button(html.Props{ID: "btn-content-update", OnClick: updateContent}, html.Text("Update Content Cards")),
			html.Button(html.Props{ID: "btn-content-clear", OnClick: clearContent}, html.Text("Clear Content Cards")),
			html.Button(html.Props{ID: "btn-deep", OnClick: renderDeep}, html.Text("Render Deep Tree (60)")),
			html.Button(html.Props{ID: "btn-hooks", OnClick: renderHooks}, html.Text("Render 40 Components w/ 60 Hooks")),
			html.Button(html.Props{ID: "btn-compute", OnClick: computePrimes}, html.Text("Compute Primes (10k)")),
		),
		html.Div(html.Props{ID: "metrics"},
			html.P(html.Props{ID: "last-render"}, html.Text("Last Render: "+lastRenderTime.Get())),
			html.P(html.Props{ID: "core-count"}, html.Text("Core Count: "+strconv.Itoa(len(coreItems.Get())))),
			html.P(html.Props{ID: "content-count"}, html.Text("Content Count: "+strconv.Itoa(len(contentItems.Get())))),
			html.P(html.Props{ID: "compute-result"}, html.Text(computeResult.Get())),
		),
		html.Div(html.Props{ID: "container"}, content),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(BenchmarkApp), "#root")
	select {}
}
