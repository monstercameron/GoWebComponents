//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/monstercameron/GoWebComponents/fetch"
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

var log = logging.New("example-0")

// getContentKindLabel maps an item content kind to the UI label shown in the detail header.
func getContentKindLabel(item docsItem) string {
	switch item.Content.Kind {
	case contentKindArticle:
		return contentKindLabelArticle
	case contentKindAPI:
		return contentKindLabelAPI
	case contentKindCounter:
		return contentKindLabelDemo
	default:
		return contentKindLabelUnknown
	}
}

// statusBadgeClass returns the badge styling for the current stability state.
func statusBadgeClass(status string) string {
	switch status {
	case statusStable:
		return "border-emerald-400/20 bg-emerald-400/10 text-emerald-200"
	case statusExperimental:
		return "border-amber-400/20 bg-amber-400/10 text-amber-200"
	default:
		return "border-rose-400/20 bg-rose-400/10 text-rose-200"
	}
}

// typeBadgeClass returns the badge styling for the catalog item type.
func typeBadgeClass(kind string) string {
	switch kind {
	case kindConcept:
		return "bg-cyan-500/15 text-cyan-200 border-cyan-400/25"
	case kindAPI:
		return "bg-violet-500/15 text-violet-200 border-violet-400/25"
	default:
		return "bg-emerald-500/15 text-emerald-200 border-emerald-400/25"
	}
}

// renderLoadingSpinner renders a lightweight shell spinner for async catalog loading states.
func renderLoadingSpinner() ui.Node {
	return Div(Class("inline-flex h-12 w-12 items-center justify-center rounded-full border border-cyan-300/20 bg-cyan-400/10"),
		Span(Class("h-6 w-6 animate-spin rounded-full border-2 border-cyan-200/30 border-t-cyan-200")),
	)
}

// renderStatCard renders a fixed-value metric card in the hero header.
func renderStatCard(value, label string) ui.Node {
	return Div(Class("rounded-3xl border border-white/10 bg-white/5 p-3 shadow-lg shadow-black/10"),
		Div(Class("text-xl font-semibold text-white"), Text(value)),
		Div(Class("mt-1 text-[11px] uppercase tracking-[0.18em] text-slate-400"), Text(label)),
	)
}

// renderCatalogFetchState renders the catalog loading and error shell while the JSON request settles.
func renderCatalogFetchState(title, message, buttonLabel string, onRetry ui.Handler) ui.Node {
	return Div(Class("min-h-screen bg-[radial-gradient(circle_at_top_left,rgba(59,130,246,0.20),transparent_28%),radial-gradient(circle_at_top_right,rgba(168,85,247,0.18),transparent_24%),linear-gradient(180deg,#07111f_0%,#091427_40%,#0b1020_100%)] text-slate-100"),
		Div(Class("mx-auto flex min-h-screen max-w-3xl items-center justify-center px-4 py-6"),
			Div(Class("w-full rounded-[24px] border border-white/10 bg-white/5 p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"),
				Div(Class("flex items-center justify-between gap-4"),
					renderLoadingSpinner(),
					Div(Class("rounded-full border border-cyan-300/20 bg-cyan-400/10 px-3 py-1 text-[11px] uppercase tracking-[0.18em] text-cyan-100"), Text("Fetching catalog")),
				),
				Div(Class("text-xs uppercase tracking-[0.18em] text-cyan-200"), Text("Example 0 data pipeline")),
				H1(Class("mt-3 text-3xl font-semibold tracking-tight text-white"), Text(title)),
				P(Class("mt-3 text-sm leading-7 text-slate-300"), Text(message)),
				Div(Class("mt-5 flex flex-wrap gap-3"),
					Button(Type("button"), OnClick(onRetry), Class("cursor-pointer rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-2 text-sm font-medium text-cyan-100 transition hover:-translate-y-0.5 hover:bg-cyan-400/20 active:translate-y-0"), Text(buttonLabel)),
				),
			),
		),
	)
}

// renderCountStatCard renders an integer metric card in the hero header.
func renderCountStatCard(value int, label string) ui.Node {
	return Div(Class("rounded-3xl border border-white/10 bg-white/5 p-3 shadow-lg shadow-black/10"),
		Div(Class("text-xl font-semibold text-white"), Textf("%d", value)),
		Div(Class("mt-1 text-[11px] uppercase tracking-[0.18em] text-slate-400"), Text(label)),
	)
}

// renderOptionNodes converts plain string values into select option nodes.
func renderOptionNodes(values []string) []ui.Node {
	return Map(values, func(value string) ui.Node {
		return Option(Value(value), Text(value))
	})
}

// renderSortOptionNodes converts sort options into select option nodes.
func renderSortOptionNodes(values []sortOption) []ui.Node {
	return Map(values, func(value sortOption) ui.Node {
		return Option(Value(value.Value), Text(value.Label))
	})
}

// renderItemCard renders a single searchable catalog item in the sidebar list.
func renderItemCard(item docsItem, isActive bool, onSelect ui.Handler) ui.Node {
	cardClass := ClassNames(
		"group block w-full cursor-pointer rounded-[22px] border p-3 text-left transition duration-200",
		When(isActive, "border-cyan-300/35 bg-cyan-400/10 shadow-xl shadow-cyan-950/25"),
		When(!isActive, "border-white/10 bg-white/[0.04] hover:-translate-y-0.5 hover:border-white/20 hover:bg-white/[0.08]"),
	)
	tagNodes := append([]ui.Node{
		Span(Class(ClassNames("rounded-full border px-2 py-1 text-[10px] uppercase tracking-[0.14em]", statusBadgeClass(item.Status))), Text(item.Status)),
		Span(Class("rounded-full border border-white/10 bg-black/20 px-2 py-1 text-[10px] uppercase tracking-[0.14em] text-slate-400"), Text(item.Level)),
		Span(Class("rounded-full border border-white/10 bg-white/5 px-2 py-1 text-[10px] text-slate-300"), Text(item.Module)),
		Span(Class("rounded-full border border-white/10 bg-white/5 px-2 py-1 text-[10px] text-slate-300"), Text(item.ReadTime)),
	}, Map(item.Tags, func(tag string) ui.Node {
		return Span(Class("rounded-full border border-white/10 bg-white/5 px-2 py-1 text-[10px] text-slate-300"), Text("#"+tag))
	})...)
	cardBody := Fragment(
		Div(Class("flex items-start justify-between gap-2"),
			Div(Class("min-w-0 flex-1"),
				Div(Class("text-sm font-medium text-white"), Text(item.Title)),
				Div(Class("mt-1 line-clamp-2 text-xs leading-5 text-slate-400"), Text(item.Blurb)),
			),
			Span(Class(ClassNames("shrink-0 rounded-full border px-2 py-1 text-[10px] font-medium uppercase tracking-[0.14em]", typeBadgeClass(item.Type))), Text(item.Type)),
		),
		Div(Class("mt-3 flex flex-wrap items-center gap-1"), tagNodes),
	)
	if isGroupedAPIItem(item) {
		return Button(Type("button"), OnClick(onSelect), Class(cardClass), cardBody)
	}
	return Button(Type("button"), OnClick(onSelect), Class(cardClass), cardBody)
}

// renderConceptArticle renders long-form concept content for article-style entries.
func renderConceptArticle(panelProps contentPanelProps) ui.Node {
	sectionNodes := make([]ui.Node, 0, len(panelProps.Item.Content.Sections)+3)
	if panelProps.Item.Content.Callout != "" {
		sectionNodes = append(sectionNodes,
			Div(Class("rounded-[22px] border border-cyan-400/20 bg-cyan-400/10 p-4 text-sm leading-7 text-cyan-50"), Text(panelProps.Item.Content.Callout)),
		)
	}
	sectionNodes = append(sectionNodes, Map(panelProps.Item.Content.Sections, func(section docsSection) ui.Node {
		paragraphs := Map(section.Paragraphs, func(paragraph string) ui.Node {
			return P(Text(paragraph))
		})
		return Article(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4"),
			H3(Class("text-lg font-semibold text-white"), Text(section.Heading)),
			Div(Class("mt-3 space-y-3 text-sm leading-7 text-slate-300"), paragraphs),
		)
	})...)
	if panelProps.Item.Content.SourcePath != "" {
		sectionNodes = append(sectionNodes,
			Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4"),
				Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelSourceDocument)),
				Div(Class("mt-3 rounded-xl border border-white/10 bg-black/15 px-3 py-2 text-sm text-cyan-100"), Text(panelProps.Item.Content.SourcePath)),
			),
		)
	}
	sectionNodes = append(sectionNodes, renderMarkdownState(panelProps))
	if panelProps.Item.Content.Code != "" && !panelProps.MarkdownReady {
		sectionNodes = append(sectionNodes,
			Div(Class("rounded-[20px] border border-white/10 bg-[#06101d] p-4"),
				Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelExampleMarkdown)),
				Pre(Class("mt-3 overflow-x-auto text-sm leading-6 text-cyan-100"), Code(Text(panelProps.Item.Content.Code))),
			),
		)
	}
	if len(sectionNodes) == 0 {
		sectionNodes = append(sectionNodes,
			Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4 text-sm leading-7 text-slate-300"), Text(panelProps.Item.Blurb)),
		)
	}
	return Div(Class("flex min-h-full flex-col rounded-[22px] border border-white/10 bg-slate-950/35 p-4 shadow-inner shadow-black/20"),
		Div(Class("border-b border-white/10 pb-3"),
			Div(Class("text-sm font-medium text-white"), Text(labelConceptArticle)),
			Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelMarkdownWriteup)),
		),
		Div(Class("mt-4 flex flex-1 flex-col gap-4"), sectionNodes),
	)
}

// renderParameterTable renders the API parameter reference grid.
func renderParameterTable(params []docsParam) ui.Node {
	rows := Map(params, func(param docsParam) ui.Node {
		return Tr(Class("border-b border-white/5 align-top text-slate-300 last:border-b-0"),
			Td(Class("py-3 pr-4 font-medium text-white"), Text(param.Name)),
			Td(Class("py-3 pr-4 text-cyan-200"), Text(param.Type)),
			Td(Class("py-3 pr-4 uppercase"), Text(param.Required)),
			Td(Class("py-3"), Text(param.Description)),
		)
	})
	return Table(Class("min-w-full text-left text-sm"),
		Thead(
			Tr(Class("border-b border-white/10 text-slate-400"),
				Th(Class("pb-2 pr-4 font-medium"), Text("Name")),
				Th(Class("pb-2 pr-4 font-medium"), Text("Type")),
				Th(Class("pb-2 pr-4 font-medium"), Text("Required")),
				Th(Class("pb-2 font-medium"), Text("Description")),
			),
		),
		Tbody(rows),
	)
}

// renderAPIReference renders the structured API reference panel.
func renderAPIReference(panelProps contentPanelProps) ui.Node {
	if isGroupedAPIItem(panelProps.Item) {
		return renderGroupedAPIReference(panelProps)
	}
	hasHTMLUsageExample := panelProps.Item.Content.SourcePath != ""
	if hasHTMLUsageExample {
		scrollToDemoAnchor(panelProps.Item.Content.AnchorID, panelProps.MarkdownReady, panelProps.MarkdownBody)
	}
	noteNodes := Map(panelProps.Item.Content.Notes, func(note string) ui.Node {
		return Li(Text(note))
	})
	return Div(Class("flex min-h-full flex-col rounded-[22px] border border-white/10 bg-slate-950/35 p-4 shadow-inner shadow-black/20"),
		Div(Class("border-b border-white/10 pb-3"),
			Div(Class("text-sm font-medium text-white"), Text(labelAPIReference)),
			Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelStructuredDocs)),
		),
		Div(Class("mt-4 flex flex-1 flex-col gap-4"),
			Div(Class("rounded-[22px] border border-violet-400/20 bg-violet-400/10 p-4"),
				Div(Class("text-xs uppercase tracking-[0.18em] text-violet-200"), Text(labelSignature)),
				Pre(Class("mt-3 overflow-x-auto text-sm leading-6 text-violet-50"), Code(Text(panelProps.Item.Content.Signature))),
				P(Class("mt-3 text-sm leading-7 text-slate-200"), Text(panelProps.Item.Content.Summary)),
			),
			Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4"),
				Div(Class("text-sm font-medium text-white"), Text(labelParameters)),
				Div(Class("mt-4 overflow-x-auto"), renderParameterTable(panelProps.Item.Content.Params)),
			),
			Div(Class("grid gap-3 md:grid-cols-2"),
				Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4"),
					Div(Class("text-sm font-medium text-white"), Text(labelReturns)),
					Div(Class("mt-3 rounded-xl border border-white/10 bg-black/15 px-3 py-2 text-sm text-emerald-200"), Text(panelProps.Item.Content.Returns)),
				),
				Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4"),
					Div(Class("text-sm font-medium text-white"), Text(labelNotes)),
					Ul(Class("mt-3 space-y-2 text-sm leading-6 text-slate-300"), noteNodes),
				),
			),
			Div(Class("rounded-[20px] border border-white/10 bg-[#06101d] p-4"),
				Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelUsageExample)),
				ui.If(hasHTMLUsageExample,
					func() ui.Node {
						return ui.Match().
							When(panelProps.MarkdownLoading && !panelProps.MarkdownReady, func() ui.Node {
								return Div(Class("mt-3 rounded-xl border border-white/10 bg-black/15 px-3 py-4 text-sm text-slate-300"), Text(messageDocLoading))
							}).
							When(panelProps.MarkdownError != "", func() ui.Node {
								return Div(Class("mt-3 rounded-xl border border-rose-400/20 bg-rose-400/10 px-3 py-4 text-sm text-rose-100"), Text(panelProps.MarkdownError))
							}).
							Default(func() ui.Node {
								return Div(Class("mt-3 rounded-xl border border-white/10 bg-black/15 p-3"),
									renderInjectedHTMLFragment("api-usage-example-fragment", panelProps.MarkdownBody),
								)
							})
					},
					func() ui.Node {
						return Pre(Class("mt-3 overflow-x-auto text-sm leading-6 text-cyan-100"), Code(Text(panelProps.Item.Content.Example)))
					},
				),
			),
		),
	)
}

// renderCounterExample renders the interactive demo panel used by example entries.
func renderCounterExample(panelProps contentPanelProps) ui.Node {
	counterValue := ui.UseState(0)
	ui.UseEffect(func() func() {
		// Reset the demo whenever the user switches to a different catalog item.
		counterValue.Set(0)
		log.Info("counter reset", map[string]interface{}{"itemID": panelProps.Item.ID})
		return nil
	}, panelProps.Item.ID)
	decrementCount := ui.UseEvent(func() {
		nextValue := counterValue.Get() - 1
		counterValue.Set(nextValue)
		log.Info("counter decremented", map[string]interface{}{"itemID": panelProps.Item.ID, "next": nextValue})
	})
	incrementCount := ui.UseEvent(func() {
		nextValue := counterValue.Get() + 1
		counterValue.Set(nextValue)
		log.Info("counter incremented", map[string]interface{}{"itemID": panelProps.Item.ID, "next": nextValue})
	})
	resetCount := ui.UseEvent(func() {
		counterValue.Set(0)
		log.Info("counter manually reset", map[string]interface{}{"itemID": panelProps.Item.ID})
	})
	stateToneLabel := toneReady
	if counterValue.Get() > 0 {
		stateToneLabel = tonePositive
	} else if counterValue.Get() < 0 {
		stateToneLabel = toneNegative
	}
	tipNodes := Map(panelProps.Item.Content.Tips, func(tip string) ui.Node {
		return Li(Text(tip))
	})
	controlButtonClass := "rounded-2xl border border-white/15 bg-black/20 px-4 py-2 text-sm font-medium text-white transition hover:bg-black/30"
	return Div(Class("flex min-h-full flex-col rounded-[22px] border border-white/10 bg-slate-950/35 p-4 shadow-inner shadow-black/20"),
		Div(Class("border-b border-white/10 pb-3"),
			Div(Class("text-sm font-medium text-white"), Text(labelInteractiveExample)),
			Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelReactiveDemoSurface)),
		),
		Div(Class("mt-4 grid flex-1 gap-3 lg:grid-cols-[minmax(0,1.4fr)_280px]"),
			Div(Class("rounded-[22px] border border-emerald-400/20 bg-emerald-400/10 p-5"),
				Div(Class("text-xs uppercase tracking-[0.18em] text-emerald-200"), Text(labelLiveWidget)),
				Div(Class("mt-4 text-5xl font-semibold tracking-tight text-white"), Textf("%d", counterValue.Get())),
				Div(Class("mt-2 text-sm text-emerald-50/90"), Text(labelStateTonePrefix+stateToneLabel)),
				Div(Class("mt-5 flex flex-wrap gap-2"),
					Button(Type("button"), OnClick(decrementCount), Class(controlButtonClass), Text(buttonDecrement)),
					Button(Type("button"), OnClick(incrementCount), Class(controlButtonClass), Text(buttonIncrement)),
					Button(Type("button"), OnClick(resetCount), Class(controlButtonClass), Text(buttonReset)),
				),
				P(Class("mt-5 text-sm leading-7 text-emerald-50/90"), Text(panelProps.Item.Content.Description)),
			),
			Div(Class("space-y-3"),
				Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4"),
					Div(Class("text-sm font-medium text-white"), Text(labelWhyThisMatters)),
					Ul(Class("mt-3 space-y-2 text-sm leading-6 text-slate-300"), tipNodes),
				),
				Div(Class("rounded-[20px] border border-white/10 bg-[#06101d] p-4"),
					Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelExampleSource)),
					Pre(Class("mt-3 overflow-x-auto text-sm leading-6 text-cyan-100"), Code(Text("const [count, setCount] = useState(0);\n\n<button onClick={() => setCount((prev) => prev + 1)}>\n  Increment\n</button>"))),
				),
			),
		),
	)
}

// renderDisplaySurface chooses the appropriate detail renderer for the selected item.
func renderDisplaySurface(panelProps contentPanelProps, hasSelectedItem bool) ui.Node {
	return ui.If(!hasSelectedItem,
		func() ui.Node {
			return Div(Class("flex min-h-full items-center justify-center rounded-[22px] border border-dashed border-white/10 bg-black/10 p-8 text-sm text-slate-400"), Text(messageNothingSelected))
		},
		func() ui.Node {
			return ui.Match().
				When(panelProps.Item.Content.Kind == contentKindArticle, func() ui.Node {
					return renderConceptArticle(panelProps)
				}).
				When(panelProps.Item.Content.Kind == contentKindAPI, func() ui.Node {
					return renderAPIReference(panelProps)
				}).
				Default(func() ui.Node {
					return ui.Component(renderCounterExample, contentPanelProps{Item: panelProps.Item})
				})
		},
	)
}

// renderCatalogHero renders the top summary banner and primary CTA actions.
func renderCatalogHero(props catalogHeroProps) ui.Node {
	return Header(Class("relative overflow-hidden rounded-[24px] border border-white/10 bg-white/5 p-5 shadow-2xl shadow-black/30 backdrop-blur-xl sm:p-6"),
		Div(Class("absolute inset-0 bg-[linear-gradient(135deg,rgba(255,255,255,0.08),transparent_35%,rgba(255,255,255,0.03))]")),
		Div(Class("relative flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between"),
			Div(Class("max-w-3xl space-y-3"),
				Div(Class("inline-flex items-center gap-2 rounded-full border border-cyan-400/30 bg-cyan-400/10 px-3 py-1 text-xs font-medium uppercase tracking-[0.24em] text-cyan-200"),
					Span(Class("h-2 w-2 rounded-full bg-cyan-300")),
					Text("Docs • APIs • Demos"),
				),
				Div(Class("space-y-2"),
					H1(Class("text-3xl font-semibold tracking-tight text-white sm:text-4xl lg:text-5xl"), Text("GoWebComponents docs, APIs, and live wasm examples")),
					P(Class("max-w-2xl text-sm leading-6 text-slate-300 sm:text-base sm:leading-7"), Text("Concept guides explain the package model, API panes stay copy-pasteable for Go authors, and example panels behave like the real GoWebComponents catalog instead of static mockups.")),
				),
				Div(Class("flex flex-wrap gap-2"),
					Button(Type("button"), OnClick(props.OnBrowseExamples), Class("cursor-pointer rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-2 text-sm font-medium text-cyan-100 transition hover:-translate-y-0.5 hover:bg-cyan-400/20 active:translate-y-0"), Text(buttonBrowseExamples)),
					Button(Type("button"), OnClick(props.OnInspectAPIs), Class("cursor-pointer rounded-2xl border border-white/10 bg-white/5 px-4 py-2 text-sm font-medium text-slate-100 transition hover:-translate-y-0.5 hover:bg-white/10 active:translate-y-0"), Text(buttonInspectPackageAPIs)),
				),
			),
			Div(Class("grid grid-cols-2 gap-2 sm:gap-3 lg:w-[320px]"),
				renderCountStatCard(props.TotalItems, labelCatalogEntries),
				renderCountStatCard(props.ExampleCount, labelLiveExamples),
				renderCountStatCard(props.APICount, labelAPIPanes),
				renderStatCard("Go + WASM", labelSingleStackUI),
			),
		),
	)
}

// renderCatalogSidebar renders the search, filters, and result list.
func renderCatalogSidebar(props catalogSidebarProps) ui.Node {
	return Section(Class("flex min-h-[420px] flex-col rounded-[24px] border border-white/10 bg-white/5 backdrop-blur-xl lg:sticky lg:top-3 lg:h-[calc(100vh-1.5rem)] lg:w-[34%] xl:w-[31%]"),
		Div(Class("sticky top-0 z-10 border-b border-white/10 bg-slate-950/60 p-3 backdrop-blur-xl sm:p-4"),
			Div(Class("flex flex-col gap-2"),
				Div(Class("flex flex-col gap-2 sm:flex-row sm:items-center"),
					Div(Class("relative flex-1"),
						Input(Value(props.SearchQuery), OnInput(props.OnSearchInput), Placeholder("Search concepts, APIs, examples..."), Class("w-full rounded-xl border border-white/10 bg-slate-950/40 px-3 py-2 text-sm text-white outline-none placeholder:text-slate-500 transition focus:border-cyan-300/40 focus:bg-slate-950/60")),
					),
					Div(Class("flex items-center gap-2 self-start sm:self-auto"),
						Div(Class("text-xs uppercase tracking-[0.18em] text-slate-400"), Textf("%d results", props.ResultCount)),
						Button(
							Type("button"),
							OnClick(props.OnResetFilters),
							Disabled(!props.HasActiveFilters),
							Class(ClassNames(
								"rounded-xl border px-3 py-2 text-[11px] font-medium uppercase tracking-[0.16em] transition",
								When(props.HasActiveFilters, "cursor-pointer border-white/10 bg-white/5 text-slate-200 hover:bg-white/10 hover:text-white"),
								When(!props.HasActiveFilters, "cursor-not-allowed border-white/5 bg-white/[0.03] text-slate-500"),
							)),
							Text(buttonResetFilters),
						),
					),
				),
				Div(Class("flex flex-wrap gap-2"), props.FilterButtons),
				Div(Class("grid grid-cols-2 gap-2 xl:grid-cols-4"),
					Label(Class("flex flex-col gap-1 text-[11px] uppercase tracking-[0.16em] text-slate-500"),
						Span(Text("Status")),
						Select(Value(props.SelectedStatusFilter), OnChange(props.OnStatusChange), Class("rounded-xl border border-white/10 bg-slate-950/50 px-3 py-2 text-xs text-slate-100 outline-none"), renderOptionNodes(props.Statuses)),
					),
					Label(Class("flex flex-col gap-1 text-[11px] uppercase tracking-[0.16em] text-slate-500"),
						Span(Text("Difficulty")),
						Select(Value(props.SelectedLevelFilter), OnChange(props.OnLevelChange), Class("rounded-xl border border-white/10 bg-slate-950/50 px-3 py-2 text-xs text-slate-100 outline-none"), renderOptionNodes(props.Levels)),
					),
					Label(Class("flex flex-col gap-1 text-[11px] uppercase tracking-[0.16em] text-slate-500"),
						Span(Text("Module")),
						Select(Value(props.SelectedModuleFilter), OnChange(props.OnModuleChange), Class("rounded-xl border border-white/10 bg-slate-950/50 px-3 py-2 text-xs text-slate-100 outline-none"), renderOptionNodes(props.Modules)),
					),
					Label(Class("flex flex-col gap-1 text-[11px] uppercase tracking-[0.16em] text-slate-500"),
						Span(Text("Sort")),
						Select(Value(props.SelectedSortOrder), OnChange(props.OnSortChange), Class("rounded-xl border border-white/10 bg-slate-950/50 px-3 py-2 text-xs text-slate-100 outline-none"), renderSortOptionNodes(props.SortOptions)),
					),
				),
			),
		),
		Div(Class("min-h-0 flex-1 overflow-y-auto p-2 sm:p-3"),
			Div(Class("space-y-2"), props.ItemNodes),
		),
	)
}

// renderDetailPanel renders the selected item header and its detail surface.
func renderDetailPanel(props detailPanelProps) ui.Node {
	return Section(Class("flex min-h-[420px] flex-1 flex-col rounded-[24px] border border-white/10 bg-white/5 backdrop-blur-xl lg:sticky lg:top-3 lg:h-[calc(100vh-1.5rem)]"),
		Div(Class("sticky top-0 z-10 border-b border-white/10 bg-slate-950/60 p-4 backdrop-blur-xl sm:p-5"),
			IfElse(props.HasSelectedItem,
				Fragment(
					Div(Class("flex flex-wrap items-center gap-3"),
						Span(Class(ClassNames("rounded-full border px-3 py-1 text-xs font-medium uppercase tracking-[0.2em]", typeBadgeClass(props.SelectedItem.Type))), Text(props.SelectedItem.Type)),
						Span(Class("rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs uppercase tracking-[0.2em] text-slate-400"), Text(props.SelectedItem.Level)),
						Span(Class("rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs uppercase tracking-[0.2em] text-slate-400"), Text(getContentKindLabel(props.SelectedItem))),
					),
					H2(Class("mt-4 text-2xl font-semibold tracking-tight text-white sm:text-3xl"), Text(props.SelectedItem.Title)),
					P(Class("mt-3 max-w-3xl text-sm leading-7 text-slate-300 sm:text-base"), Text(props.SelectedItem.Blurb)),
				),
				Fragment(
					H2(Class("text-2xl font-semibold tracking-tight text-white sm:text-3xl"), Text(labelNothingSelected)),
					P(Class("mt-3 max-w-3xl text-sm leading-7 text-slate-300 sm:text-base"), Text(messageAdjustFilters)),
				),
			),
		),
		Div(ID("demo"), Class("min-h-0 flex-1 overflow-y-auto p-3 sm:p-4"), renderDisplaySurface(contentPanelProps{Item: props.SelectedItem, MarkdownBody: props.MarkdownBody, MarkdownLoading: props.MarkdownLoading, MarkdownReady: props.MarkdownReady, MarkdownError: props.MarkdownError, OnRetryMarkdown: props.OnRetryMarkdown}, props.HasSelectedItem)),
	)
}

// renderDocsDemosSite wires together the search state, filters, selection state, and layout.
func renderDocsDemosSite() ui.Node {
	catalogURL := catalogDataURL()
	catalogResource := fetch.UseCachedResource(catalogCacheKey(catalogURL), func(ctx context.Context) (docsCatalog, error) {
		return loadCatalogResource(ctx, catalogURL)
	}, fetch.CacheOptions{StaleAfter: 45 * time.Second})
	catalogRequest := catalogResource.Get()
	retryCatalogLoad := ui.UseEvent(func() {
		log.Info("catalog refetch requested", map[string]interface{}{"url": catalogURL})
		catalogResource.Reload()
	})
	ui.UseEffect(func() func() {
		switch {
		case catalogRequest.Loading:
			log.Info("catalog cache loading", map[string]interface{}{"url": catalogURL, "stale": catalogRequest.Stale, "ready": catalogRequest.Ready})
		case catalogRequest.Error != nil:
			log.Error("catalog cache failed", map[string]interface{}{"url": catalogURL, "error": catalogRequest.Error.Error()})
		case catalogRequest.Ready:
			log.Info("catalog cache ready", map[string]interface{}{
				"url":         catalogURL,
				"modules":     len(catalogRequest.Value.Modules),
				"statuses":    len(catalogRequest.Value.Statuses),
				"levels":      len(catalogRequest.Value.Levels),
				"filters":     len(catalogRequest.Value.Filters),
				"sortOptions": len(catalogRequest.Value.SortOptions),
				"items":       len(catalogRequest.Value.Items),
				"stale":       catalogRequest.Stale,
			})
		}
		return nil
	}, catalogRequest.Loading, catalogRequest.Ready, catalogRequest.Stale, fmt.Sprint(catalogRequest.Error), len(catalogRequest.Value.Items))

	searchQuery := ui.UseState("")
	deferredSearchQuery := ui.UseDeferredValue(searchQuery.Get())
	activeTypeFilter := ui.UseState(filterAll)
	selectedStatusFilter := ui.UseState(allFilterValue)
	selectedLevelFilter := ui.UseState(allFilterValue)
	selectedModuleFilter := ui.UseState(allFilterValue)
	selectedSortOrder := ui.UseState(sortRelevance)
	selectedItemID := ui.UseState(0)

	updateSearchQuery := ui.UseEvent(func(event ui.InputEvent) {
		searchQuery.Set(event.GetValue())
		log.Info("search query updated", map[string]interface{}{"query": event.GetValue()})
	})
	updateStatusFilter := ui.UseEvent(func(event ui.ChangeEvent) {
		selectedStatusFilter.Set(event.GetValue())
		log.Info("status filter updated", map[string]interface{}{"status": event.GetValue()})
	})
	updateLevelFilter := ui.UseEvent(func(event ui.ChangeEvent) {
		selectedLevelFilter.Set(event.GetValue())
		log.Info("level filter updated", map[string]interface{}{"level": event.GetValue()})
	})
	updateModuleFilter := ui.UseEvent(func(event ui.ChangeEvent) {
		selectedModuleFilter.Set(event.GetValue())
		log.Info("module filter updated", map[string]interface{}{"module": event.GetValue()})
	})
	updateSortOrder := ui.UseEvent(func(event ui.ChangeEvent) {
		selectedSortOrder.Set(event.GetValue())
		log.Info("sort order updated", map[string]interface{}{"sort": event.GetValue()})
	})
	resetFilters := ui.UseEvent(func() {
		activeTypeFilter.Set(filterAll)
		selectedStatusFilter.Set(allFilterValue)
		selectedLevelFilter.Set(allFilterValue)
		selectedModuleFilter.Set(allFilterValue)
		selectedSortOrder.Set(sortRelevance)
		log.Info("filters reset", map[string]interface{}{
			"type":   filterAll,
			"status": allFilterValue,
			"level":  allFilterValue,
			"module": allFilterValue,
			"sort":   sortRelevance,
		})
	})
	hasActiveFilters := activeTypeFilter.Get() != filterAll || selectedStatusFilter.Get() != allFilterValue || selectedLevelFilter.Get() != allFilterValue || selectedModuleFilter.Get() != allFilterValue || selectedSortOrder.Get() != sortRelevance

	filteredItems := ui.UseMemo(func() []docsItem {
		return sortItems(filterItems(catalogRequest.Value.Items, deferredSearchQuery, activeTypeFilter.Get(), selectedStatusFilter.Get(), selectedLevelFilter.Get(), selectedModuleFilter.Get()), selectedSortOrder.Get())
	}, len(catalogRequest.Value.Items), deferredSearchQuery, activeTypeFilter.Get(), selectedStatusFilter.Get(), selectedLevelFilter.Get(), selectedModuleFilter.Get(), selectedSortOrder.Get())
	ui.UseEffect(func() func() {
		log.Info("filters applied", map[string]interface{}{
			"query":       searchQuery.Get(),
			"deferred":    deferredSearchQuery,
			"type":        activeTypeFilter.Get(),
			"status":      selectedStatusFilter.Get(),
			"level":       selectedLevelFilter.Get(),
			"module":      selectedModuleFilter.Get(),
			"sort":        selectedSortOrder.Get(),
			"resultCount": len(filteredItems),
		})
		return nil
	}, searchQuery.Get(), deferredSearchQuery, activeTypeFilter.Get(), selectedStatusFilter.Get(), selectedLevelFilter.Get(), selectedModuleFilter.Get(), selectedSortOrder.Get(), filteredItemsSignature(filteredItems))
	ui.UseEffect(func() func() {
		// Keep selection aligned with the filtered list so the detail panel never points at a stale item.
		if len(filteredItems) == 0 {
			selectedItemID.Set(0)
			log.Info("selection cleared", map[string]interface{}{"reason": "no filtered items"})
			return nil
		}
		currentSelectedID := selectedItemID.Get()
		for _, item := range filteredItems {
			if item.ID == currentSelectedID {
				return nil
			}
		}
		selectedItemID.Set(filteredItems[0].ID)
		log.Info("selection repaired", map[string]interface{}{"selectedItemID": filteredItems[0].ID})
		return nil
	}, filteredItemsSignature(filteredItems), selectedItemID.Get())

	selectedCatalogItem, hasSelectedItem := findSelectedItem(filteredItems, selectedItemID.Get())
	selectedDocURL := ""
	if hasSelectedItem && selectedCatalogItem.Content.SourcePath != "" && (selectedCatalogItem.Content.Kind == contentKindArticle || selectedCatalogItem.Content.Kind == contentKindAPI) {
		selectedDocURL = docsSourceURL(selectedCatalogItem.Content.SourcePath)
	}
	markdownResource := fetch.UseCachedResource(markdownCacheKey(selectedDocURL), func(ctx context.Context) (string, error) {
		return loadMarkdownResource(ctx, selectedDocURL)
	}, fetch.CacheOptions{StaleAfter: 2 * time.Minute})
	markdownRequest := markdownResource.Get()
	retryMarkdownLoad := ui.UseEvent(func() {
		if selectedDocURL == "" {
			return
		}
		log.Info("markdown refetch requested", map[string]interface{}{"url": selectedDocURL, "sourcePath": selectedCatalogItem.Content.SourcePath})
		markdownResource.Reload()
	})
	ui.UseEffect(func() func() {
		if selectedDocURL == "" {
			return nil
		}
		switch {
		case markdownRequest.Loading:
			log.Info("markdown cache loading", map[string]interface{}{"url": selectedDocURL, "sourcePath": selectedCatalogItem.Content.SourcePath, "ready": markdownRequest.Ready, "stale": markdownRequest.Stale})
		case markdownRequest.Error != nil:
			log.Error("markdown cache failed", map[string]interface{}{"url": selectedDocURL, "sourcePath": selectedCatalogItem.Content.SourcePath, "error": markdownRequest.Error.Error()})
		case markdownRequest.Ready:
			log.Info("markdown cache ready", map[string]interface{}{"url": selectedDocURL, "sourcePath": selectedCatalogItem.Content.SourcePath, "bytes": len(markdownRequest.Value)})
		}
		return nil
	}, selectedDocURL, markdownRequest.Loading, markdownRequest.Ready, markdownRequest.Stale, errorString(markdownRequest.Error), len(markdownRequest.Value), selectedItemID.Get())
	browseExamples := ui.UseEvent(func() {
		activeTypeFilter.Set(kindExample)
		log.Info("quick filter selected", map[string]interface{}{"type": kindExample})
	})
	inspectPackageAPIs := ui.UseEvent(func() {
		activeTypeFilter.Set(kindAPI)
		log.Info("quick filter selected", map[string]interface{}{"type": kindAPI})
	})
	ui.UseEffect(func() func() {
		if !hasSelectedItem {
			return nil
		}
		log.Info("detail item selected", map[string]interface{}{
			"itemID": selectedCatalogItem.ID,
			"title":  selectedCatalogItem.Title,
			"type":   selectedCatalogItem.Type,
		})
		return nil
	}, selectedItemID.Get(), filteredItemsSignature(filteredItems))

	filterButtons := Map(catalogRequest.Value.Filters, func(filter string) ui.Node {
		filterValue := filter
		isActiveFilter := filterValue == activeTypeFilter.Get()
		buttonClass := ClassNames(
			"cursor-pointer rounded-xl border px-2.5 py-1.5 text-xs transition",
			When(isActiveFilter, "border-cyan-300/40 bg-cyan-400/15 text-cyan-100 shadow-lg shadow-cyan-900/20"),
			When(!isActiveFilter, "border-white/10 bg-white/5 text-slate-300 hover:bg-white/10 hover:text-white"),
		)
		return Button(Type("button"), OnClick(ui.UseEvent(func() {
			activeTypeFilter.Set(filterValue)
			log.Info("type filter updated", map[string]interface{}{"type": filterValue})
		})), Class(buttonClass), Text(filterValue))
	})

	itemNodes := Map(filteredItems, func(item docsItem) ui.Node {
		selectedCardID := item.ID
		return renderItemCard(item, hasSelectedItem && selectedCatalogItem.ID == item.ID, ui.UseEvent(func() {
			selectedItemID.Set(selectedCardID)
			log.Info("catalog card clicked", map[string]interface{}{"itemID": selectedCardID, "title": item.Title})
		}))
	})
	itemNodes = append(itemNodes, If(len(filteredItems) == 0,
		Div(Class("rounded-[22px] border border-dashed border-white/10 bg-black/10 p-6 text-center text-sm text-slate-400"), Text(messageNoMatches)),
	))

	catalogErrorMessage := ""
	if catalogRequest.Error != nil {
		catalogErrorMessage = catalogRequest.Error.Error()
	}

	return ui.Match().
		When(!catalogRequest.Ready && catalogRequest.Error == nil, func() ui.Node {
			return renderCatalogFetchState("Loading catalog", "Requesting the example catalog JSON before the docs surface renders.", "Retry request", retryCatalogLoad)
		}).
		When(catalogRequest.Error != nil && !catalogRequest.Ready, func() ui.Node {
			return renderCatalogFetchState("Catalog request failed", catalogErrorMessage, "Retry request", retryCatalogLoad)
		}).
		Default(func() ui.Node {
			return Div(Class("min-h-screen bg-[radial-gradient(circle_at_top_left,rgba(59,130,246,0.20),transparent_28%),radial-gradient(circle_at_top_right,rgba(168,85,247,0.18),transparent_24%),linear-gradient(180deg,#07111f_0%,#091427_40%,#0b1020_100%)] text-slate-100"),
				Div(Class("mx-auto flex min-h-screen max-w-7xl flex-col px-3 py-3 sm:px-4 sm:py-4 lg:px-5"),
					ui.Component(renderCatalogHero, catalogHeroProps{OnBrowseExamples: browseExamples, OnInspectAPIs: inspectPackageAPIs, TotalItems: len(catalogRequest.Value.Items), ExampleCount: countItemsByType(catalogRequest.Value.Items, kindExample), APICount: countItemsByType(catalogRequest.Value.Items, kindAPI)}),
					Main(Class("mt-3 flex flex-1 flex-col gap-3 lg:min-h-0 lg:flex-row"),
						ui.Component(renderCatalogSidebar, catalogSidebarProps{SearchQuery: searchQuery.Get(), ResultCount: len(filteredItems), HasActiveFilters: hasActiveFilters, Statuses: catalogRequest.Value.Statuses, Levels: catalogRequest.Value.Levels, Modules: catalogRequest.Value.Modules, SortOptions: catalogRequest.Value.SortOptions, FilterButtons: filterButtons, SelectedStatusFilter: selectedStatusFilter.Get(), SelectedLevelFilter: selectedLevelFilter.Get(), SelectedModuleFilter: selectedModuleFilter.Get(), SelectedSortOrder: selectedSortOrder.Get(), ItemNodes: itemNodes, OnSearchInput: updateSearchQuery, OnStatusChange: updateStatusFilter, OnLevelChange: updateLevelFilter, OnModuleChange: updateModuleFilter, OnSortChange: updateSortOrder, OnResetFilters: resetFilters}),
						ui.Component(renderDetailPanel, detailPanelProps{SelectedItem: selectedCatalogItem, HasSelectedItem: hasSelectedItem, MarkdownBody: markdownRequest.Value, MarkdownLoading: markdownRequest.Loading, MarkdownReady: markdownRequest.Ready, MarkdownError: errorString(markdownRequest.Error), OnRetryMarkdown: retryMarkdownLoad}),
					),
				),
			)
		})
}

func main() {
	defer func() {
		if recovered := recover(); recovered != nil {
			_, _ = runtime.FinalizeUnhandledPanicContext("runtime", runtime.PanicPhaseStartup, "main", "#app", nil, recovered)
		}
	}()

	logging.AttachBrowserConsole(logging.BrowserConsoleOptions{Scope: "example-0"})
	utils.DisableAllDebug()
	ui.Render(ui.Component(renderDocsDemosSite), "#app")
	utils.WaitForever()
}
