//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
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

func loadCachedFeed(ctx context.Context) ([]string, error) {
	select {
	case <-time.After(700 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	revision := nextCachedFeedRevision()
	return []string{
		fmt.Sprintf("Revision %d", revision),
		fmt.Sprintf("Loaded at %s", time.Now().Format("15:04:05")),
		"Shared key lets multiple components see the same cached data.",
	}, nil
}

func renderFeedList(items []string) []ui.Node {
	children := make([]ui.Node, 0, len(items))
	for _, item := range items {
		children = append(children, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-slate-200"}, html.Text(item)))
	}
	return children
}

func cachedControlPanel() ui.Node {
	resource := fetch.UseCachedResource(cachedKey, loadCachedFeed, fetch.CacheOptions{StaleAfter: 5 * time.Second})
	state := resource.Get()
	reload := ui.UseEvent(func() { resource.Reload() })
	invalidate := ui.UseEvent(func() { resource.Invalidate() })
	optimistic := ui.UseEvent(func() {
		resource.Update(func(previous []string) []string {
			next := append([]string{}, previous...)
			next = append([]string{"Optimistic local item"}, next...)
			return next
		})
	})

	status := "Idle"
	if state.Loading && !state.Ready {
		status = "Initial load"
	} else if state.Loading && state.Ready {
		status = "Refreshing"
	} else if state.Stale {
		status = "Stale"
	} else if state.Ready {
		status = "Ready"
	}

	updatedAt := "-"
	if !state.UpdatedAt.IsZero() {
		updatedAt = state.UpdatedAt.Format("15:04:05")
	}

	return shared.ExamplePanel("Cache owner",
		html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
			shared.ExampleButton("Reload", reload),
			shared.ExampleButton("Invalidate", invalidate),
			shared.ExampleButton("Optimistic update", optimistic),
		),
		html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
			shared.ExampleStat("Status", status),
			shared.ExampleStat("Ready", fmt.Sprintf("%t", state.Ready)),
			shared.ExampleStat("Updated", updatedAt),
		),
	)
}

func cachedViewerPanel() ui.Node {
	resource := fetch.UseCachedResource(cachedKey, loadCachedFeed, fetch.CacheOptions{StaleAfter: 5 * time.Second})
	state := resource.Get()

	items := []ui.Node{html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-slate-400"}, html.Text("Waiting for cache data..."))}
	if len(state.Value) > 0 {
		items = renderFeedList(state.Value)
	}

	return shared.ExamplePanel("Shared consumer",
		html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("This second component uses the same cache key and sees the same value, stale state, and optimistic updates.")),
		html.Ul(html.Props{Class: "mt-6 grid gap-3"}, items...),
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