//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/hotreload"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type cacheDemoRecord struct {
	Label    string `json:"label"`
	Revision int    `json:"revision"`
	Source   string `json:"source"`
}

var trustCounter int32 = 1
var swrCounter int32 = 1
var alwaysCounter int32 = 1

func loadTrust(parseCtx context.Context) (cacheDemoRecord, error) {
	select {
	case <-parseCtx.Done():
		return cacheDemoRecord{}, parseCtx.Err()
	case <-time.After(60 * time.Millisecond):
	}
	return cacheDemoRecord{
		Label:    "Authoritative client refresh after trust-once seed",
		Revision: int(atomic.AddInt32(&trustCounter, 1)),
		Source:   "client loader",
	}, nil
}

func loadStale(parseCtx context.Context) (cacheDemoRecord, error) {
	select {
	case <-parseCtx.Done():
		return cacheDemoRecord{}, parseCtx.Err()
	case <-time.After(80 * time.Millisecond):
	}
	return cacheDemoRecord{
		Label:    "Bootstrap entry was stale, so the client revalidated in the background",
		Revision: int(atomic.AddInt32(&swrCounter, 1)),
		Source:   "client loader",
	}, nil
}

func loadAlways(parseCtx context.Context) (cacheDemoRecord, error) {
	select {
	case <-parseCtx.Done():
		return cacheDemoRecord{}, parseCtx.Err()
	case <-time.After(90 * time.Millisecond):
	}
	return cacheDemoRecord{
		Label:    "Resume policy forced an immediate authoritative client load",
		Revision: int(atomic.AddInt32(&alwaysCounter, 1)),
		Source:   "client loader",
	}, nil
}

func cacheCard(parseTitle string, parsePolicy fetch.CacheResumePolicy, parseSummary string, parseKey string, parseLoader func(context.Context) (cacheDemoRecord, error)) ui.Node {
	parseResource := fetch.UseCachedResource(parseKey, parseLoader, fetch.CacheOptions{
		StaleAfter:   time.Minute,
		MaxAge:       5 * time.Minute,
		DisposeAfter: 10 * time.Minute,
	})
	parseState := parseResource.Get()

	parseStatus := "Idle"
	if parseState.Loading && !parseState.Ready {
		parseStatus = "Cold loading"
	} else if parseState.Loading && parseState.Ready {
		parseStatus = "Background revalidating"
	} else if parseState.Stale {
		parseStatus = "Seeded stale snapshot"
	} else if parseState.Ready {
		parseStatus = "Ready"
	}
	if parseState.Error != nil {
		parseStatus = parseState.Error.Error()
	}

	parseReload := ui.UseEvent(func() { parseResource.Reload() })
	parseInvalidate := ui.UseEvent(func() { parseResource.Invalidate() })
	parseDispose := ui.UseEvent(func() { parseResource.Dispose() })

	parseLabel := "No cached value"
	parseRevision := "0"
	parseSource := "none"
	if parseState.Ready {
		parseLabel = parseState.Value.Label
		parseRevision = fmt.Sprintf("%d", parseState.Value.Revision)
		parseSource = emptyFallback(parseState.Value.Source, "bootstrap")
	}

	return html.Div(html.Props{Class: "rounded-[1.6rem] border border-white/10 bg-slate-950/70 p-6 shadow-[0_18px_48px_rgba(2,6,23,0.35)]"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.3em] text-cyan-300"}, html.Text(string(parsePolicy))),
		html.H2(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text(parseTitle)),
		html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text(parseSummary)),
		html.Div(html.Props{Class: "mt-5 grid gap-3 md:grid-cols-2"},
			statBlock("Revision", parseRevision),
			statBlock("Source", parseSource),
			statBlock("Ready", fmt.Sprintf("%t", parseState.Ready)),
			statBlock("Stale", fmt.Sprintf("%t", parseState.Stale)),
		),
		html.P(html.Props{Class: "mt-5 text-sm leading-7 text-slate-300"}, html.Text("Status: "+parseStatus)),
		html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-400"}, html.Text(parseLabel)),
		html.Div(html.Props{Class: "mt-5 flex flex-wrap gap-3"},
			actionButton("Reload", parseReload),
			actionButton("Invalidate", parseInvalidate),
			actionButton("Dispose", parseDispose),
		),
	)
}

func statBlock(parseLabel, parseValue string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "mt-3 text-lg font-black text-white"}, html.Text(parseValue)),
	)
}

func actionButton(parseLabel string, parseHandler ui.Handler) ui.Node {
	return html.Button(html.Props{
		OnClick: parseHandler,
		Class:   "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80",
	}, html.Text(parseLabel))
}

func app() ui.Node {
	return html.Div(html.Props{Class: "min-h-screen bg-[#08111d] text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-6xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/80 p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("SSR shared-cache bootstrap")),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text("Seed fetch cache state before hydration")),
				html.P(html.Props{Class: "mt-4 max-w-4xl text-lg leading-8 text-slate-300"}, html.Text("This page restores shared-cache entries from ui.SSRBootstrap.Data before hydration. The three cards show trust-once, stale-while-revalidate, and always-refetch resume behavior against the same public fetch cache surface.")),
				html.Div(html.Props{Class: "mt-8 grid gap-6 lg:grid-cols-3"},
					cacheCard(
						"Trust bootstrap once",
						fetch.CacheResumeTrustOnce,
						"Use the seeded value as fresh data until explicit reload, invalidation, or normal stale timing says otherwise.",
						"catalog:trust",
						loadTrust,
					),
					cacheCard(
						"Stale while revalidate",
						fetch.CacheResumeStaleWhileRevalidate,
						"Start with the server snapshot, but mark it stale because the embedded timestamp is already beyond the allowed freshness window.",
						"catalog:swr",
						loadStale,
					),
					cacheCard(
						"Always refetch",
						fetch.CacheResumeAlwaysRefetch,
						"Keep the seed visible for first paint, but force an authoritative client pass on the first subscriber regardless of freshness.",
						"catalog:always",
						loadAlways,
					),
				),
				html.Div(html.Props{Class: "mt-8 rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Security and serialization")),
					html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Bootstrap cache entries should stay JSON-shaped, omit secrets, and remain small enough for the first HTML response. The shared cache only improves startup and reuse; it does not make client data authoritative.")),
				),
			),
		),
	)
}

func emptyFallback(parseValue, parseFallback string) string {
	if strings.TrimSpace(parseValue) == "" {
		return parseFallback
	}
	return parseValue
}

func main() {
	utils.DisableAllDebug()
	hotreload.Enable()

	parsePayload, parseErr := ui.ReadBootstrapScript("")
	if parseErr == nil {
		_ = fetch.RestoreCacheBootstrap(parsePayload)
		_, _ = exampleboot.ApplyExampleHydration(ui.CreateElement(app), ui.HydrationOptions{Bootstrap: parsePayload})
		exampleboot.WaitExampleRuntime()
		return
	}

	exampleboot.RenderExampleRoot(ui.CreateElement(app))
	exampleboot.WaitExampleRuntime()
}
