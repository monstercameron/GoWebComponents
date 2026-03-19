//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type indexRequest struct {
	Text  string `json:"text"`
	Query string `json:"query"`
}

type indexProgress struct {
	Percent int    `json:"percent"`
	Stage   string `json:"stage"`
}

type termCount struct {
	Term  string `json:"term"`
	Count int    `json:"count"`
}

type indexResult struct {
	TotalWords  int         `json:"totalWords"`
	UniqueWords int         `json:"uniqueWords"`
	Query       string      `json:"query"`
	QueryHits   int         `json:"queryHits"`
	TopTerms    []termCount `json:"topTerms"`
}

const defaultWorkerCorpus = `Atlas routes should stay responsive even when the app is crunching a dense inventory draft, tokenizing copy, or building a fast local search index for a content-heavy screen.

GoWebComponents can keep UI state on the main thread while a dedicated worker handles the heavier parsing and counting pass. The worker reports progress, the page keeps rendering, and the user can cancel or rerun the job without freezing the rest of the interface.

This sample text is intentionally repetitive so the top-term list becomes obvious after indexing. Atlas atlas atlas worker worker router router inventory inventory search search search status status status.`

func workerTextIndexExample() ui.Node {
	text := ui.UseState(defaultWorkerCorpus)
	query := ui.UseState("atlas")
	task := ui.UseWorkerTask[indexRequest, indexProgress, indexResult](interop.WorkerOptions{
		URL:   "./text-index-worker.js",
		Ready: true,
		Name:  "text-index",
	}, "build-index")

	setText := ui.UseEvent(func(e ui.Event) {
		text.Set(e.GetValue())
	})
	setQuery := ui.UseEvent(func(e ui.Event) {
		query.Set(strings.TrimSpace(e.GetValue()))
	})
	start := ui.UseEvent(func() {
		task.Start(indexRequest{Text: text.Get(), Query: query.Get()})
	})
	cancel := ui.UseEvent(func() { task.Cancel() })
	reset := ui.UseEvent(func() {
		task.Cancel()
		text.Set(defaultWorkerCorpus)
		query.Set("atlas")
	})

	state := task.Get()
	status := "Ready to build a worker-side index."
	if state.Running {
		status = "Worker indexing is in flight."
	} else if state.Cancelled {
		status = "Worker indexing was cancelled."
	} else if state.Error != nil {
		status = state.Error.Error()
	} else if state.Ready {
		status = "Worker index completed."
	}

	progressValue := "0%"
	progressStage := "idle"
	if state.ProgressReady {
		progressValue = fmt.Sprintf("%d%%", state.Progress.Percent)
		progressStage = state.Progress.Stage
	}

	topTerms := []ui.Node{
		html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm text-slate-300"}, html.Text("Run the worker to see the top terms.")),
	}
	if state.Ready && len(state.Value.TopTerms) > 0 {
		topTerms = make([]ui.Node, 0, len(state.Value.TopTerms))
		for _, term := range state.Value.TopTerms {
			topTerms = append(topTerms,
				html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm text-slate-300"},
					html.Text(fmt.Sprintf("%s · %d", term.Term, term.Count)),
				),
			)
		}
	}

	return shared.ExamplePage(
		"Worker Text Index",
		"ui.UseWorkerTask and interop.NewWorker",
		"Move a token-counting index build onto a dedicated worker so the page can keep rendering, report progress, and cancel a heavy CPU-bound pass cleanly.",
		shared.ExamplePanel("Worker-backed indexing",
			html.Label(html.Props{For: "worker-index-text", Class: "text-sm font-semibold text-slate-200"}, html.Text("Corpus")),
			html.Textarea(html.Props{
				ID:          "worker-index-text",
				Rows:        12,
				Value:       text.Get(),
				OnInput:     setText,
				Placeholder: "Paste a large text block",
				Class:       "mt-2 w-full rounded-[1.35rem] border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100",
			}),
			html.Div(html.Props{Class: "mt-4 grid gap-4 md:grid-cols-[minmax(0,1fr)_220px]"},
				html.Div(html.Props{},
					html.Label(html.Props{For: "worker-index-query", Class: "text-sm font-semibold text-slate-200"}, html.Text("Search term")),
					html.Input(html.Props{
						ID:          "worker-index-query",
						Value:       query.Get(),
						OnInput:     setQuery,
						Placeholder: "atlas",
						Class:       "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100",
					}),
				),
				html.Div(html.Props{Class: "flex flex-wrap items-end gap-3"},
					shared.ExampleButton("Build index", start),
					shared.ExampleButton("Cancel", cancel),
					shared.ExampleButton("Reset sample", reset),
				),
			),
		),
		shared.ExamplePanel("Worker state",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-5"},
				shared.ExampleStat("Running", fmt.Sprintf("%t", state.Running)),
				shared.ExampleStat("Progress", progressValue),
				shared.ExampleStat("Stage", progressStage),
				shared.ExampleStat("Unique words", fmt.Sprintf("%d", state.Value.UniqueWords)),
				shared.ExampleStat("Query hits", fmt.Sprintf("%d", state.Value.QueryHits)),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(status)),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, topTerms...),
		),
		shared.ExamplePanel("Integration shape",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("The component only owns local UI state and calls ui.UseWorkerTask(...). Worker creation, request correlation, progress routing, cancellation, and teardown stay behind the public helper instead of leaking into the page.")),
			shared.ExampleCode(
				`task := ui.UseWorkerTask[indexRequest, indexProgress, indexResult](interop.WorkerOptions{URL: "./text-index-worker.js", Ready: true}, "build-index")`,
				`task.Start(indexRequest{Text: corpus, Query: query})`,
				`task.Cancel()`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(workerTextIndexExample), "#app")
	select {}
}
