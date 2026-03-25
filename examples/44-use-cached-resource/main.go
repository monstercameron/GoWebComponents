//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const cachedKey = "catalog-fetch-shared-feed"

var (
	cachedFeedMu       sync.Mutex
	cachedFeedRevision int
)

func nextCachedFeedRevision() int {
	cachedFeedMu.Lock()
	defer cachedFeedMu.Unlock()
	cachedFeedRevision++
	return cachedFeedRevision
}

func loadCachedFeed(parseCtx context.Context) ([]string, error) {
	select {
	case <-time.After(700 * time.Millisecond):
	case <-parseCtx.Done():
		return nil, parseCtx.Err()
	}

	parseRevision := nextCachedFeedRevision()
	return []string{
		fmt.Sprintf("Revision %d", parseRevision),
		fmt.Sprintf("Loaded at %s", time.Now().Format("15:04:05")),
		"Shared key lets multiple components see the same cached data.",
	}, nil
}

func renderFeedList(parseItems []string) []ui.Node {
	parseChildren := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseChildren = append(parseChildren, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-slate-200"}, html.Text(parseItem)))
	}
	return parseChildren
}

func cachedControlPanel() ui.Node {
	parseResource := fetch.UseCachedResource(cachedKey, loadCachedFeed, fetch.CacheOptions{StaleAfter: 5 * time.Second})
	parseState := parseResource.Get()
	parseReload := ui.UseEvent(func() { parseResource.Reload() })
	parseInvalidate := ui.UseEvent(func() { parseResource.Invalidate() })
	parseOptimistic := ui.UseEvent(func() {
		parseResource.Update(func(parsePrevious []string) []string {
			parseNext := append([]string{}, parsePrevious...)
			parseNext = append([]string{"Optimistic local item"}, parseNext...)
			return parseNext
		})
	})

	parseStatus := "Idle"
	if parseState.Loading && !parseState.Ready {
		parseStatus = "Initial load"
	} else if parseState.Loading && parseState.Ready {
		parseStatus = "Refreshing"
	} else if parseState.Stale {
		parseStatus = "Stale"
	} else if parseState.Ready {
		parseStatus = "Ready"
	}

	parseUpdatedAt := "-"
	if !parseState.UpdatedAt.IsZero() {
		parseUpdatedAt = parseState.UpdatedAt.Format("15:04:05")
	}

	return shared.ExamplePanel("Cache owner",
		html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
			shared.ExampleButton("Reload", parseReload),
			shared.ExampleButton("Invalidate", parseInvalidate),
			shared.ExampleButton("Optimistic update", parseOptimistic),
		),
		html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
			shared.ExampleStat("Status", parseStatus),
			shared.ExampleStat("Ready", fmt.Sprintf("%t", parseState.Ready)),
			shared.ExampleStat("Updated", parseUpdatedAt),
		),
	)
}

func cachedViewerPanel() ui.Node {
	parseResource := fetch.UseCachedResource(cachedKey, loadCachedFeed, fetch.CacheOptions{StaleAfter: 5 * time.Second})
	parseState := parseResource.Get()

	parseItems := []ui.Node{html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-slate-400"}, html.Text("Waiting for cache data..."))}
	if len(parseState.Value) > 0 {
		parseItems = renderFeedList(parseState.Value)
	}

	return shared.ExamplePanel("Shared consumer",
		html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("This second component uses the same cache key and sees the same value, stale state, and optimistic updates.")),
		html.Ul(html.Props{Class: "mt-6 grid gap-3"}, parseItems...),
	)
}

func useCachedResourceExample() ui.Node {
	return shared.ExamplePage(
		"fetch.UseCachedResource",
		"Share async data with stale-aware cache state",
		"Cached resources deduplicate work by key, keep last-ready values visible during refreshes, and let callers push optimistic changes before revalidation.",
		ui.CreateElement(cachedControlPanel),
		ui.CreateElement(cachedViewerPanel),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useCachedResourceExample), "#app")
	select {}
}
