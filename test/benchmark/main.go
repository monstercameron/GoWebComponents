//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
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

func DeepTree(parseProps DeepTreeProps) ui.Node {
	if parseProps.Depth <= 0 {
		return html.Div(html.Props{Class: "leaf", ID: "deep-leaf"}, html.Text("Leaf"))
	}

	return html.Div(
		html.Props{Class: "node"},
		ui.CreateElement(DeepTree, DeepTreeProps{Depth: parseProps.Depth - 1}),
	)
}

func ManyHooks() ui.Node {
	for parseI := 0; parseI < hooksPerComponent; parseI++ {
		ui.UseState(parseI)
		ui.UseEffect(func() func() { return nil })
		ui.UseMemo(func() int { return parseI * 2 }, parseI)
	}

	return html.Div(html.Props{Class: "hook-node"}, html.Text("Hooks"))
}

type ContentCardProps struct {
	Item ContentCardData
}

func ContentCard(parseProps ContentCardProps) ui.Node {
	parseTagChildren := make([]ui.Node, 0, len(parseProps.Item.Tags))
	for _, parseTag := range parseProps.Item.Tags {
		parseTagChildren = append(parseTagChildren,
			html.Span(html.Props{Class: "content-tag"}, html.Text(parseTag)),
		)
	}

	return html.Article(html.Props{Class: "content-card"},
		html.Div(html.Props{Class: "content-card-header"},
			html.H2(html.Props{Class: "content-title"}, html.Text(parseProps.Item.Title)),
			html.Span(html.Props{Class: "content-status"}, html.Text(parseProps.Item.Status)),
		),
		html.P(html.Props{Class: "content-summary"}, html.Text(parseProps.Item.Summary)),
		html.Div(html.Props{Class: "content-meta"},
			html.Span(html.Props{Class: "content-meta-text"}, html.Text(parseProps.Item.Meta)),
		),
		html.Div(html.Props{Class: "content-tags"}, parseTagChildren...),
	)
}

func buildCoreItems() []string {
	parseItems := make([]string, coreListSize)
	for parseI := 0; parseI < coreListSize; parseI++ {
		parseItems[parseI] = "Item " + strconv.Itoa(parseI)
	}
	return parseItems
}

func buildContentItems() []ContentCardData {
	parseItems := make([]ContentCardData, contentCardCount)
	for parseI := 0; parseI < contentCardCount; parseI++ {
		parseItems[parseI] = ContentCardData{
			ID:      parseI,
			Title:   "Article " + strconv.Itoa(parseI),
			Summary: "This benchmark card exercises regular app rendering with nested content blocks.",
			Status:  "draft",
			Meta:    "Section " + strconv.Itoa((parseI%3)+1),
			Tags: []string{
				"perf",
				"bench",
				"card-" + strconv.Itoa(parseI%4),
			},
		}
	}
	return parseItems
}

func BenchmarkApp() ui.Node {
	parseCoreItems := ui.UseState([]string{})
	parseContentItems := ui.UseState([]ContentCardData{})
	parseView := ui.UseState("core")
	parseLastRenderTime := ui.UseState("")
	parseComputeResult := ui.UseState("")

	ui.UseEffect(func() func() {
		parseLastRenderTime.Set(time.Now().Format(time.RFC3339Nano))
		return nil
	}, parseCoreItems.Get(), parseContentItems.Get(), parseView.Get())

	parseComputePrimes := ui.UseEvent(func() {
		parseStart := time.Now()
		parseCount := 0
		for parseI := 2; parseI < primeLimit; parseI++ {
			isPrime := true
			for parseJ := 2; parseJ*parseJ <= parseI; parseJ++ {
				if parseI%parseJ == 0 {
					isPrime = false
					break
				}
			}
			if isPrime {
				parseCount++
			}
		}
		parseDuration := time.Since(parseStart)
		parseComputeResult.Set(fmt.Sprintf("Found %d primes in %dms", parseCount, parseDuration.Milliseconds()))
	})

	renderList := ui.UseEvent(func() {
		parseView.Set("core")
		parseCoreItems.Set(buildCoreItems())
		parseContentItems.Set([]ContentCardData{})
	})

	renderContent := ui.UseEvent(func() {
		parseView.Set("content")
		parseContentItems.Set(buildContentItems())
		parseCoreItems.Set([]string{})
	})

	renderDeep := ui.UseEvent(func() {
		parseView.Set("deep")
		parseCoreItems.Set([]string{})
		parseContentItems.Set([]ContentCardData{})
	})

	renderHooks := ui.UseEvent(func() {
		parseView.Set("hooks")
		parseCoreItems.Set([]string{})
		parseContentItems.Set([]ContentCardData{})
	})

	clearList := ui.UseEvent(func() {
		parseView.Set("core")
		parseCoreItems.Set([]string{})
	})

	clearContent := ui.UseEvent(func() {
		parseView.Set("content")
		parseContentItems.Set([]ContentCardData{})
	})

	parseUpdateList := ui.UseEvent(func() {
		parseCoreItems.Update(func(parseCurrent []string) []string {
			parseNewItems := make([]string, len(parseCurrent))
			for parseI2, parseItem := range parseCurrent {
				parseNewItems[parseI2] = parseItem + " (Updated)"
			}
			return parseNewItems
		})
	})

	parseUpdateContent := ui.UseEvent(func() {
		parseContentItems.Update(func(parseCurrent2 []ContentCardData) []ContentCardData {
			parseNext := make([]ContentCardData, len(parseCurrent2))
			for parseI3, parseItem2 := range parseCurrent2 {
				parseTags := make([]string, len(parseItem2.Tags))
				copy(parseTags, parseItem2.Tags)
				parseNext[parseI3] = ContentCardData{
					ID:      parseItem2.ID,
					Title:   parseItem2.Title + " (Updated)",
					Summary: parseItem2.Summary + " Updated with fresh content.",
					Status:  "live",
					Meta:    parseItem2.Meta + " / refreshed",
					Tags:    parseTags,
				}
			}
			return parseNext
		})
	})

	var parseContent ui.Node
	switch parseView.Get() {
	case "deep":
		parseContent = ui.CreateElement(DeepTree, DeepTreeProps{Depth: deepTreeDepth})
	case "content":
		parseChildren := make([]ui.Node, 0, len(parseContentItems.Get()))
		for _, parseItem3 := range parseContentItems.Get() {
			parseChildren = append(parseChildren, ui.CreateElement(ContentCard, ContentCardProps{Item: parseItem3}))
		}
		parseContent = html.Div(html.Props{ID: "content-container"}, parseChildren...)
	case "hooks":
		parseChildren2 := make([]ui.Node, 0, hookComponentCount)
		for parseI4 := 0; parseI4 < hookComponentCount; parseI4++ {
			parseChildren2 = append(parseChildren2, ui.CreateElement(ManyHooks))
		}
		parseContent = html.Div(html.Props{ID: "hooks-container"}, parseChildren2...)
	default:
		parseChildren3 := make([]ui.Node, 0, len(parseCoreItems.Get()))
		for _, parseItem4 := range parseCoreItems.Get() {
			parseChildren3 = append(parseChildren3, html.Div(html.Props{Class: "core-list-item"}, html.Text(parseItem4)))
		}
		parseContent = html.Div(html.Props{ID: "core-list-container"}, parseChildren3...)
	}

	return html.Div(html.Props{ID: "app"},
		html.H1(html.Props{}, html.Text("Benchmark App")),
		html.Div(html.Props{ID: "controls"},
			html.Button(html.Props{ID: "btn-render", OnClick: renderList}, html.Text("Render Core Items")),
			html.Button(html.Props{ID: "btn-update", OnClick: parseUpdateList}, html.Text("Update Core Items")),
			html.Button(html.Props{ID: "btn-clear", OnClick: clearList}, html.Text("Clear Core Items")),
			html.Button(html.Props{ID: "btn-content-render", OnClick: renderContent}, html.Text("Render Content Cards")),
			html.Button(html.Props{ID: "btn-content-update", OnClick: parseUpdateContent}, html.Text("Update Content Cards")),
			html.Button(html.Props{ID: "btn-content-clear", OnClick: clearContent}, html.Text("Clear Content Cards")),
			html.Button(html.Props{ID: "btn-deep", OnClick: renderDeep}, html.Text("Render Deep Tree (60)")),
			html.Button(html.Props{ID: "btn-hooks", OnClick: renderHooks}, html.Text("Render 40 Components w/ 60 Hooks")),
			html.Button(html.Props{ID: "btn-compute", OnClick: parseComputePrimes}, html.Text("Compute Primes (10k)")),
		),
		html.Div(html.Props{ID: "metrics"},
			html.P(html.Props{ID: "last-render"}, html.Text("Last Render: "+parseLastRenderTime.Get())),
			html.P(html.Props{ID: "core-count"}, html.Text("Core Count: "+strconv.Itoa(len(parseCoreItems.Get())))),
			html.P(html.Props{ID: "content-count"}, html.Text("Content Count: "+strconv.Itoa(len(parseContentItems.Get())))),
			html.P(html.Props{ID: "compute-result"}, html.Text(parseComputeResult.Get())),
		),
		html.Div(html.Props{ID: "container"}, parseContent),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(BenchmarkApp), "#root")
	select {}
}
