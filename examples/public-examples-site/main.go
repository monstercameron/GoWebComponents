//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/fetch"
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

var log = logging.New("public-examples-site")

// getContentKindLabel maps an item content kind to the UI label shown in the detail header.
func getContentKindLabel(parseItem docsItem) string {
	switch parseItem.Content.Kind {
	case contentKindArticle:
		return contentKindLabelArticle
	case contentKindAPI:
		return contentKindLabelAPI
	case contentKindExample:
		return contentKindLabelDemo
	case contentKindCounter:
		return contentKindLabelDemo
	default:
		return contentKindLabelUnknown
	}
}

// statusBadgeClass returns the badge styling for the current stability state.
func statusBadgeClass(parseStatus string) string {
	switch parseStatus {
	case statusStable:
		return "border-emerald-400/20 bg-emerald-400/10 text-emerald-200"
	case statusExperimental:
		return "border-amber-400/20 bg-amber-400/10 text-amber-200"
	default:
		return "border-rose-400/20 bg-rose-400/10 text-rose-200"
	}
}

// typeBadgeClass returns the badge styling for the catalog item type.
func typeBadgeClass(parseKind string) string {
	switch parseKind {
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
func renderStatCard(parseValue, parseLabel string) ui.Node {
	return Div(Class("rounded-3xl border border-white/10 bg-white/5 p-3 shadow-lg shadow-black/10"),
		Div(Class("text-xl font-semibold text-white"), Text(parseValue)),
		Div(Class("mt-1 text-[11px] uppercase tracking-[0.18em] text-slate-400"), Text(parseLabel)),
	)
}

// renderCatalogFetchState renders the catalog loading and error shell while the JSON request settles.
func renderCatalogFetchState(parseTitle, parseMessage, parseButtonLabel string, parseOnRetry ui.Handler) ui.Node {
	return Div(Class("min-h-screen bg-[radial-gradient(circle_at_top_left,rgba(34,211,238,0.14),transparent_24%),radial-gradient(circle_at_top_right,rgba(251,191,36,0.10),transparent_22%),linear-gradient(180deg,#08111d_0%,#0d1726_44%,#0c1320_100%)] text-slate-100"),
		Div(Class("mx-auto flex min-h-screen max-w-3xl items-center justify-center px-4 py-6"),
			Div(Class("w-full rounded-[28px] border border-white/10 bg-slate-950/55 p-6 shadow-2xl shadow-black/35 backdrop-blur-xl"),
				Div(Class("flex items-center justify-between gap-4"),
					renderLoadingSpinner(),
					Div(Class("rounded-full border border-cyan-300/20 bg-cyan-400/10 px-3 py-1 text-[11px] uppercase tracking-[0.18em] text-cyan-100"), Text("Fetching examples")),
				),
				Div(Class("text-xs uppercase tracking-[0.18em] text-cyan-200"), Text("GoWebComponents example gallery")),
				H1(Class("mt-3 text-3xl font-semibold tracking-tight text-white"), Text(parseTitle)),
				P(Class("mt-3 text-sm leading-7 text-slate-300"), Text(parseMessage)),
				Div(Class("mt-5 flex flex-wrap gap-3"),
					Button(Type("button"), OnClick(parseOnRetry), Class("cursor-pointer rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-2 text-sm font-medium text-cyan-100 transition hover:-translate-y-0.5 hover:bg-cyan-400/20 active:translate-y-0"), Text(parseButtonLabel)),
				),
			),
		),
	)
}

// renderCountStatCard renders an integer metric card in the hero header.
func renderCountStatCard(parseValue int, parseLabel string) ui.Node {
	return Div(Class("rounded-3xl border border-white/10 bg-white/5 p-3 shadow-lg shadow-black/10"),
		Div(Class("text-xl font-semibold text-white"), Textf("%d", parseValue)),
		Div(Class("mt-1 text-[11px] uppercase tracking-[0.18em] text-slate-400"), Text(parseLabel)),
	)
}

// renderOptionNodes converts plain string values into select option nodes.
func renderOptionNodes(parseValues []string) []ui.Node {
	return Map(parseValues, func(parseValue string) ui.Node {
		return Option(Value(parseValue), Text(parseValue))
	})
}

// renderSortOptionNodes converts sort options into select option nodes.
func renderSortOptionNodes(parseValues []sortOption) []ui.Node {
	return Map(parseValues, func(parseValue sortOption) ui.Node {
		return Option(Value(parseValue.Value), Text(parseValue.Label))
	})
}

func catalogResultAnnouncement(parseResultCount int, parseQuery string) string {
	parseNoun := "examples"
	if parseResultCount == 1 {
		parseNoun = "example"
	}
	parseTrimmed := strings.TrimSpace(parseQuery)
	if parseTrimmed == "" {
		return fmt.Sprintf("%d %s available.", parseResultCount, parseNoun)
	}
	return fmt.Sprintf("%d %s match %q.", parseResultCount, parseNoun, parseTrimmed)
}

func firstCatalogItems(parseItems []docsItem, parseLimit int) []docsItem {
	if parseLimit <= 0 || len(parseItems) == 0 {
		return nil
	}
	if len(parseItems) <= parseLimit {
		return parseItems
	}
	return parseItems[:parseLimit]
}

// renderItemCard renders a single searchable catalog item in the sidebar list.
func renderItemCard(parseItem docsItem, isActive bool, parseOnSelect ui.Handler) ui.Node {
	parseConceptLabels := buildExampleConceptLabels(parseItem)
	if len(parseConceptLabels) > 3 {
		parseConceptLabels = parseConceptLabels[:3]
	}
	parseCardClass := ClassNames(
		"group block w-full cursor-pointer rounded-[24px] border p-4 text-left transition duration-200",
		When(isActive, "border-cyan-300/35 bg-cyan-400/10 shadow-xl shadow-cyan-950/25"),
		When(!isActive, "border-white/10 bg-slate-950/35 hover:-translate-y-0.5 hover:border-cyan-300/20 hover:bg-slate-900/60"),
	)
	parseTagNodes := []ui.Node{
		Span(Class(ClassNames("rounded-full border px-2 py-1 text-[10px] uppercase tracking-[0.14em]", statusBadgeClass(parseItem.Status))), Text(parseItem.Status)),
		Span(Class("rounded-full border border-white/10 bg-black/20 px-2 py-1 text-[10px] uppercase tracking-[0.14em] text-slate-400"), Text(parseItem.Level)),
		Span(Class("rounded-full border border-white/10 bg-white/5 px-2 py-1 text-[10px] uppercase tracking-[0.14em] text-slate-300"), Text(formatExampleConceptLabel(parseItem.Module))),
	}
	parseCardBody := Fragment(
		Div(Class("flex items-start justify-between gap-3"),
			Div(Class("min-w-0 flex-1"),
				Div(Class("text-sm font-semibold tracking-tight text-white"), Text(parseItem.Title)),
				Div(Class("mt-2 line-clamp-3 text-sm leading-6 text-slate-300"), Text(getExampleFocusSummary(parseItem))),
			),
			Span(Class("shrink-0 rounded-full border border-cyan-300/20 bg-cyan-400/10 px-2.5 py-1 text-[10px] font-medium uppercase tracking-[0.16em] text-cyan-100"), Text(parseItem.ReadTime)),
		),
		Div(Class("mt-3 flex flex-wrap items-center gap-1.5"), parseTagNodes),
		If(len(parseConceptLabels) > 0,
			Div(Class("mt-3 flex flex-wrap gap-1.5"), renderExampleConceptChipNodes(parseConceptLabels)),
		),
	)
	if isGroupedAPIItem(parseItem) {
		return Button(Type("button"), OnClick(parseOnSelect), Class(parseCardClass), parseCardBody)
	}
	return Button(Type("button"), OnClick(parseOnSelect), Class(parseCardClass), parseCardBody)
}

// renderConceptArticle renders long-form concept content for article-style entries.
func renderConceptArticle(parsePanelProps contentPanelProps) ui.Node {
	parseSectionNodes := make([]ui.Node, 0, len(parsePanelProps.Item.Content.Sections)+3)
	if parsePanelProps.Item.Content.Callout != "" {
		parseSectionNodes = append(parseSectionNodes,
			Div(Class("rounded-[22px] border border-cyan-400/20 bg-cyan-400/10 p-4 text-sm leading-7 text-cyan-50"), Text(parsePanelProps.Item.Content.Callout)),
		)
	}
	parseSectionNodes = append(parseSectionNodes, Map(parsePanelProps.Item.Content.Sections, func(parseSection docsSection) ui.Node {
		parseParagraphs := Map(parseSection.Paragraphs, func(parseParagraph string) ui.Node {
			return P(Text(parseParagraph))
		})
		return Article(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4"),
			H3(Class("text-lg font-semibold text-white"), Text(parseSection.Heading)),
			Div(Class("mt-3 space-y-3 text-sm leading-7 text-slate-300"), parseParagraphs),
		)
	})...)
	if parsePanelProps.Item.Content.SourcePath != "" {
		parseSectionNodes = append(parseSectionNodes,
			Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4"),
				Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelSourceDocument)),
				Div(Class("mt-3 rounded-xl border border-white/10 bg-black/15 px-3 py-2 text-sm text-cyan-100"), Text(parsePanelProps.Item.Content.SourcePath)),
			),
		)
	}
	parseSectionNodes = append(parseSectionNodes, renderMarkdownState(parsePanelProps))
	if parsePanelProps.Item.Content.Code != "" && !parsePanelProps.MarkdownReady {
		parseSectionNodes = append(parseSectionNodes,
			Div(Class("rounded-[20px] border border-white/10 bg-[#06101d] p-4"),
				Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelExampleMarkdown)),
				Pre(Class("mt-3 overflow-x-auto text-sm leading-6 text-cyan-100"), Code(Text(parsePanelProps.Item.Content.Code))),
			),
		)
	}
	if len(parseSectionNodes) == 0 {
		parseSectionNodes = append(parseSectionNodes,
			Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4 text-sm leading-7 text-slate-300"), Text(parsePanelProps.Item.Blurb)),
		)
	}
	return Div(Class("min-w-0 flex min-h-full flex-col rounded-[22px] border border-white/10 bg-slate-950/35 p-4 shadow-inner shadow-black/20"),
		Div(Class("border-b border-white/10 pb-3"),
			Div(Class("text-sm font-medium text-white"), Text(labelConceptArticle)),
			Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelMarkdownWriteup)),
		),
		Div(Class("mt-4 flex flex-1 flex-col gap-4"), parseSectionNodes),
	)
}

// renderParameterTable renders the API parameter reference grid.
func renderParameterTable(parseParams []docsParam) ui.Node {
	parseRows := Map(parseParams, func(parseParam docsParam) ui.Node {
		return Tr(Class("border-b border-white/5 align-top text-slate-300 last:border-b-0"),
			Td(Class("py-3 pr-4 font-medium text-white"), Text(parseParam.Name)),
			Td(Class("py-3 pr-4 text-cyan-200"), Text(parseParam.Type)),
			Td(Class("py-3 pr-4 uppercase"), Text(parseParam.Required)),
			Td(Class("py-3"), Text(parseParam.Description)),
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
		Tbody(parseRows),
	)
}

// renderAPIReference renders the structured API reference panel.
func renderAPIReference(parsePanelProps contentPanelProps) ui.Node {
	if isGroupedAPIItem(parsePanelProps.Item) {
		return renderGroupedAPIReference(parsePanelProps)
	}
	hasHTMLUsageExample := parsePanelProps.Item.Content.SourcePath != ""
	if hasHTMLUsageExample {
		scrollToDemoAnchor(parsePanelProps.Item.Content.AnchorID, parsePanelProps.MarkdownReady, parsePanelProps.MarkdownBody)
	}
	parseNoteNodes := Map(parsePanelProps.Item.Content.Notes, func(parseNote string) ui.Node {
		return Li(Text(parseNote))
	})
	return Div(Class("min-w-0 flex min-h-full flex-col rounded-[22px] border border-white/10 bg-slate-950/35 p-4 shadow-inner shadow-black/20"),
		Div(Class("border-b border-white/10 pb-3"),
			Div(Class("text-sm font-medium text-white"), Text(labelAPIReference)),
			Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelStructuredDocs)),
		),
		Div(Class("mt-4 flex flex-1 flex-col gap-4"),
			Div(Class("rounded-[22px] border border-violet-400/20 bg-violet-400/10 p-4"),
				Div(Class("text-xs uppercase tracking-[0.18em] text-violet-200"), Text(labelSignature)),
				Pre(Class("mt-3 overflow-x-auto text-sm leading-6 text-violet-50"), Code(Text(parsePanelProps.Item.Content.Signature))),
				P(Class("mt-3 text-sm leading-7 text-slate-200"), Text(parsePanelProps.Item.Content.Summary)),
			),
			Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4"),
				Div(Class("text-sm font-medium text-white"), Text(labelParameters)),
				Div(Class("mt-4 overflow-x-auto"), renderParameterTable(parsePanelProps.Item.Content.Params)),
			),
			Div(Class("grid gap-3 md:grid-cols-2"),
				Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4"),
					Div(Class("text-sm font-medium text-white"), Text(labelReturns)),
					Div(Class("mt-3 rounded-xl border border-white/10 bg-black/15 px-3 py-2 text-sm text-emerald-200"), Text(parsePanelProps.Item.Content.Returns)),
				),
				Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4"),
					Div(Class("text-sm font-medium text-white"), Text(labelNotes)),
					Ul(Class("mt-3 space-y-2 text-sm leading-6 text-slate-300"), parseNoteNodes),
				),
			),
			Div(Class("rounded-[20px] border border-white/10 bg-[#06101d] p-4"),
				Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelUsageExample)),
				ui.If(hasHTMLUsageExample,
					func() ui.Node {
						return ui.Match().
							When(parsePanelProps.MarkdownLoading && !parsePanelProps.MarkdownReady, func() ui.Node {
								return Div(Class("mt-3 rounded-xl border border-white/10 bg-black/15 px-3 py-4 text-sm text-slate-300"), Text(messageDocLoading))
							}).
							When(parsePanelProps.MarkdownError != "", func() ui.Node {
								return Div(Class("mt-3 rounded-xl border border-rose-400/20 bg-rose-400/10 px-3 py-4 text-sm text-rose-100"), Text(parsePanelProps.MarkdownError))
							}).
							Default(func() ui.Node {
								return Div(Class("mt-3 rounded-xl border border-white/10 bg-black/15 p-3"),
									renderInjectedHTMLFragment("api-usage-example-fragment", parsePanelProps.MarkdownBody),
								)
							})
					},
					func() ui.Node {
						return Pre(Class("mt-3 overflow-x-auto text-sm leading-6 text-cyan-100"), Code(Text(parsePanelProps.Item.Content.Example)))
					},
				),
			),
		),
	)
}

// renderCounterExample renders the interactive demo panel used by example entries.
func renderCounterExample(parsePanelProps contentPanelProps) ui.Node {
	if parsePanelProps.Item.Content.PreviewPath != "" {
		return renderEmbeddedExample(parsePanelProps)
	}
	if parsePanelProps.Item.Content.Kind == contentKindExample || parsePanelProps.Item.Content.SourcePath != "" || parsePanelProps.Item.Content.Code != "" || parsePanelProps.Item.Content.EmbedPath != "" {
		return renderSourceOnlyExample(parsePanelProps)
	}
	parseCounterValue := ui.UseState(0)
	ui.UseEffect(func() func() {
		// Reset the demo whenever the user switches to a different catalog item.
		parseCounterValue.Set(0)
		if isPageDebugLoggingEnabled() {
			log.Info("counter reset", map[string]interface{}{"itemID": parsePanelProps.Item.ID})
		}
		return nil
	}, parsePanelProps.Item.ID)
	parseDecrementCount := ui.UseEvent(func() {
		parseNextValue := parseCounterValue.Get() - 1
		parseCounterValue.Set(parseNextValue)
		if isPageDebugLoggingEnabled() {
			log.Info("counter decremented", map[string]interface{}{"itemID": parsePanelProps.Item.ID, "next": parseNextValue})
		}
	})
	parseIncrementCount := ui.UseEvent(func() {
		parseNextValue2 := parseCounterValue.Get() + 1
		parseCounterValue.Set(parseNextValue2)
		if isPageDebugLoggingEnabled() {
			log.Info("counter incremented", map[string]interface{}{"itemID": parsePanelProps.Item.ID, "next": parseNextValue2})
		}
	})
	resetCount := ui.UseEvent(func() {
		parseCounterValue.Set(0)
		if isPageDebugLoggingEnabled() {
			log.Info("counter manually reset", map[string]interface{}{"itemID": parsePanelProps.Item.ID})
		}
	})
	parseStateToneLabel := toneReady
	if parseCounterValue.Get() > 0 {
		parseStateToneLabel = tonePositive
	} else if parseCounterValue.Get() < 0 {
		parseStateToneLabel = toneNegative
	}
	parseTipNodes := Map(parsePanelProps.Item.Content.Tips, func(parseTip string) ui.Node {
		return Li(Text(parseTip))
	})
	parseControlButtonClass := "rounded-2xl border border-white/15 bg-black/20 px-4 py-2 text-sm font-medium text-white transition hover:bg-black/30"
	return Div(Class("min-w-0 flex min-h-full flex-col rounded-[22px] border border-white/10 bg-slate-950/35 p-4 shadow-inner shadow-black/20"),
		Div(Class("border-b border-white/10 pb-3"),
			Div(Class("text-sm font-medium text-white"), Text(labelInteractiveExample)),
			Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelReactiveDemoSurface)),
		),
		Div(Class("mt-4 grid flex-1 gap-3 lg:grid-cols-[minmax(0,1.4fr)_280px]"),
			Div(Class("rounded-[22px] border border-emerald-400/20 bg-emerald-400/10 p-5"),
				Div(Class("text-xs uppercase tracking-[0.18em] text-emerald-200"), Text(labelLiveWidget)),
				Div(Class("mt-4 text-5xl font-semibold tracking-tight text-white"), Textf("%d", parseCounterValue.Get())),
				Div(Class("mt-2 text-sm text-emerald-50/90"), Text(labelStateTonePrefix+parseStateToneLabel)),
				Div(Class("mt-5 flex flex-wrap gap-2"),
					Button(Type("button"), OnClick(parseDecrementCount), Class(parseControlButtonClass), Text(buttonDecrement)),
					Button(Type("button"), OnClick(parseIncrementCount), Class(parseControlButtonClass), Text(buttonIncrement)),
					Button(Type("button"), OnClick(resetCount), Class(parseControlButtonClass), Text(buttonReset)),
				),
				P(Class("mt-5 text-sm leading-7 text-emerald-50/90"), Text(parsePanelProps.Item.Content.Description)),
			),
			Div(Class("space-y-3"),
				Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4"),
					Div(Class("text-sm font-medium text-white"), Text(labelWhyThisMatters)),
					Ul(Class("mt-3 space-y-2 text-sm leading-6 text-slate-300"), parseTipNodes),
				),
				Div(Class("rounded-[20px] border border-white/10 bg-[#06101d] p-4"),
					Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelExampleSource)),
					Pre(Class("mt-3 overflow-x-auto text-sm leading-6 text-cyan-100"), Code(Text("const [count, setCount] = useState(0);\n\n<button onClick={() => setCount((prev) => prev + 1)}>\n  Increment\n</button>"))),
				),
			),
		),
	)
}

// renderSourceOnlyExample renders one example card when the docs shell should show the code and guidance instead of live-mounting the wasm binary.
func renderSourceOnlyExample(parsePanelProps contentPanelProps) ui.Node {
	parseLearningPoints := Map(buildExampleLearningPoints(parsePanelProps.Item), func(parsePoint string) ui.Node {
		return Li(Text(parsePoint))
	})
	parseConceptNodes := renderExampleConceptChipNodes(buildExampleConceptLabels(parsePanelProps.Item))
	parseDescription := strings.TrimSpace(parsePanelProps.Item.Content.Description)
	if parseDescription == "" {
		parseDescription = messageSourceFirstDemo
	}
	return Div(Class("min-w-0 flex min-h-full flex-col rounded-[26px] border border-white/10 bg-slate-950/45 p-4 shadow-inner shadow-black/25"),
		Div(Class("border-b border-white/10 pb-3"),
			Div(Class("text-sm font-medium text-white"), Text(labelSourceFirstExample)),
			Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text("Concept and source")),
		),
		Div(Class("mt-4 grid flex-1 gap-3 lg:grid-cols-[minmax(0,0.9fr)_minmax(320px,1.1fr)]"),
			Div(Class("space-y-3"),
				Div(Class("rounded-[22px] border border-amber-400/20 bg-amber-400/10 p-5"),
					Div(Class("text-xs uppercase tracking-[0.18em] text-amber-200"), Text(labelConceptFocus)),
					H3(Class("mt-3 text-2xl font-semibold tracking-tight text-white"), Text(parsePanelProps.Item.Title)),
					P(Class("mt-4 text-sm leading-7 text-slate-100"), Text(parseDescription)),
					P(Class("mt-4 text-sm leading-7 text-slate-300"), Text(getExampleFocusSummary(parsePanelProps.Item))),
					If(len(parseConceptNodes) > 0,
						Div(Class("mt-5 flex flex-wrap gap-2"), parseConceptNodes),
					),
				),
				Div(Class("rounded-[22px] border border-white/10 bg-slate-900/55 p-5"),
					Div(Class("text-xs uppercase tracking-[0.18em] text-slate-400"), Text(labelStudyPrompts)),
					Ul(Class("mt-4 space-y-3 text-sm leading-6 text-slate-200"), parseLearningPoints),
				),
			),
			Div(Class("min-w-0 space-y-2"),
				IfElse(parsePanelProps.IsSourceVisible,
					Div(Class("space-y-2"),
						Div(Class("flex justify-end"),
							Button(Type("button"), OnClick(parsePanelProps.OnHideSource), Class("cursor-pointer rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-xs font-medium uppercase tracking-[0.16em] text-slate-200 transition hover:border-cyan-300/30 hover:text-cyan-100"), Text(buttonHideSource)),
						),
						renderExampleSourceNode(parsePanelProps),
					),
					Div(Class("rounded-[24px] border border-white/10 bg-slate-950/50 p-5"),
						Div(Class("flex flex-wrap items-center justify-between gap-3"),
							Div(Class("min-w-0"),
								Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelExampleSource)),
								P(Class("mt-3 text-sm leading-7 text-slate-300"), Text(messageSourceDeferred)),
							),
							Button(Type("button"), OnClick(parsePanelProps.OnShowSource), Class("cursor-pointer rounded-2xl border border-cyan-300/25 bg-cyan-400/12 px-4 py-2 text-sm font-medium text-cyan-100 transition hover:bg-cyan-400/18"), Text(buttonShowSource)),
						),
					),
				),
			),
		),
	)
}

// renderExampleSourceNode resolves the mirrored source file for one example and renders it as a code panel.
func renderExampleSourceNode(parsePanelProps contentPanelProps) ui.Node {
	parseSourceURL := docsSourceURL(parsePanelProps.Item.Content.SourcePath)
	parseSourceBody := parsePanelProps.MarkdownBody
	parseSourceLoading := parsePanelProps.MarkdownLoading
	parseSourceReady := parsePanelProps.MarkdownReady
	parseSourceError := parsePanelProps.MarkdownError
	if parseSourceURL != "" && parseSourceBody == "" && !parseSourceLoading && !parseSourceReady && parseSourceError == "" {
		parseSourceResource := fetch.UseCachedResource(markdownCacheKey(parseSourceURL), func(parseCtx context.Context) (string, error) {
			return loadMarkdownResource(parseCtx, parseSourceURL)
		}, fetch.CacheOptions{StaleAfter: 2 * time.Minute})
		parseSourceRequest := parseSourceResource.Get()
		parseSourceBody = parseSourceRequest.Value
		parseSourceLoading = parseSourceRequest.Loading
		parseSourceReady = parseSourceRequest.Ready
		parseSourceError = errorString(parseSourceRequest.Error)
	}
	return ui.Match().
		When(parseSourceLoading && !parseSourceReady, func() ui.Node {
			return Div(Class("rounded-[24px] border border-white/10 bg-slate-950/50 px-4 py-5 text-sm text-slate-300"), Text(messageDocLoading))
		}).
		When(parseSourceError != "", func() ui.Node {
			return Div(Class("rounded-[24px] border border-rose-400/20 bg-rose-400/10 px-4 py-5 text-sm text-rose-100"), Text(parseSourceError))
		}).
		When(parseSourceReady && parseSourceBody != "", func() ui.Node {
			return renderSourceSnippetCard(labelExampleSource, "Go + hooks + typed HTML", parseSourceBody)
		}).
		Default(func() ui.Node {
			return renderSourceSnippetCard(labelExampleSource, "Go + hooks + typed HTML", parsePanelProps.Item.Content.Code)
		})
}

func renderEmbeddedExample(parsePanelProps contentPanelProps) ui.Node {
	parsePreviewURL := previewExampleURL(parsePanelProps.Item.Content.PreviewPath)
	parseConceptNodes := renderExampleConceptChipNodes(buildExampleConceptLabels(parsePanelProps.Item))
	parseLearningPoints := Map(buildExampleLearningPoints(parsePanelProps.Item), func(parsePoint string) ui.Node {
		return Li(Text(parsePoint))
	})
	return Div(Class("min-w-0 flex min-h-full flex-col rounded-[26px] border border-white/10 bg-slate-950/45 p-3 shadow-inner shadow-black/25"),
		Div(Class("flex flex-wrap items-center justify-between gap-2 border-b border-white/10 pb-2"),
			Div(Class("text-sm font-medium text-white"), Text(labelLivePreview)),
			Div(Class("flex flex-wrap items-center gap-2"),
				Div(Class("text-[11px] uppercase tracking-[0.18em] text-slate-500"), Text(labelPreviewMode)),
				If(parsePreviewURL != "",
					A(
						Href(parsePreviewURL),
						Attr("target", "_blank"),
						Attr("rel", "noreferrer noopener"),
						Class("rounded-full border border-white/10 bg-white/5 px-3 py-1 text-[11px] uppercase tracking-[0.16em] text-slate-300 transition hover:border-cyan-300/30 hover:text-cyan-100"),
						Text(labelStandalonePreview),
					),
				),
			),
		),
		Div(Class("mt-3 min-w-0 grid flex-1 gap-3 lg:grid-cols-[minmax(0,1.15fr)_minmax(320px,0.85fr)]"),
			Div(Class("min-w-0 space-y-2"),
				Div(Class("overflow-hidden rounded-[20px] border border-white/10 bg-[#081420] shadow-inner shadow-black/20"),
					IfElse(parsePanelProps.IsPreviewLoaded && parsePreviewURL != "",
						Iframe(
							Src(parsePreviewURL),
							Title(parsePanelProps.Item.Title+" example preview"),
							Class("min-h-[420px] w-full border-0 bg-transparent"),
							Attr("loading", "lazy"),
							Attr("allow", "clipboard-read; clipboard-write"),
						),
						Div(Class("flex min-h-[420px] items-center justify-center px-4 py-6 text-sm text-slate-400"),
							Div(Class("max-w-md rounded-[22px] border border-white/10 bg-white/[0.04] p-5 text-left"),
								Div(Class("text-xs uppercase tracking-[0.18em] text-cyan-200"), Text(labelPreviewPaused)),
								P(Class("mt-3 leading-7 text-slate-300"), Text(messagePreviewDeferred)),
								IfElse(parsePreviewURL != "",
									Div(Class("mt-5 flex flex-wrap gap-2"),
										Button(Type("button"), OnClick(parsePanelProps.OnLoadPreview), Class("cursor-pointer rounded-2xl border border-cyan-300/25 bg-cyan-400/12 px-4 py-2 text-sm font-medium text-cyan-100 transition hover:bg-cyan-400/18"), Text(buttonLoadPreview)),
										A(
											Href(parsePreviewURL),
											Attr("target", "_blank"),
											Attr("rel", "noreferrer noopener"),
											Class("rounded-2xl border border-white/10 bg-white/5 px-4 py-2 text-sm font-medium text-slate-200 transition hover:border-cyan-300/30 hover:text-cyan-100"),
											Text(labelStandalonePreview),
										),
									),
									P(Class("mt-5 leading-7 text-slate-400"), Text("Preview host is unavailable for this example.")),
								),
							),
						),
					),
				),
			),
			Div(Class("min-w-0 space-y-2"),
				Div(Class("rounded-[22px] border border-cyan-300/15 bg-slate-900/55 p-5"),
					Div(Class("text-xs uppercase tracking-[0.18em] text-cyan-200"), Text(labelConceptFocus)),
					H3(Class("mt-3 text-2xl font-semibold tracking-tight text-white"), Text(parsePanelProps.Item.Title)),
					P(Class("mt-4 text-sm leading-7 text-slate-200"), Text(getExampleFocusSummary(parsePanelProps.Item))),
					If(len(parseConceptNodes) > 0,
						Div(Class("mt-4 flex flex-wrap gap-2"), parseConceptNodes),
					),
				),
				Div(Class("rounded-[22px] border border-white/10 bg-slate-900/55 p-5"),
					Div(Class("text-xs uppercase tracking-[0.18em] text-slate-400"), Text(labelStudyPrompts)),
					Ul(Class("mt-4 space-y-3 text-sm leading-6 text-slate-200"), parseLearningPoints),
				),
				IfElse(parsePanelProps.IsSourceVisible,
					Div(Class("space-y-2"),
						Div(Class("flex justify-end"),
							Button(Type("button"), OnClick(parsePanelProps.OnHideSource), Class("cursor-pointer rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-xs font-medium uppercase tracking-[0.16em] text-slate-200 transition hover:border-cyan-300/30 hover:text-cyan-100"), Text(buttonHideSource)),
						),
						renderExampleSourceNode(parsePanelProps),
					),
					Div(Class("rounded-[24px] border border-white/10 bg-slate-950/50 p-5"),
						Div(Class("flex flex-wrap items-center justify-between gap-3"),
							Div(Class("min-w-0"),
								Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelExampleSource)),
								P(Class("mt-3 text-sm leading-7 text-slate-300"), Text(messageSourceDeferred)),
							),
							Button(Type("button"), OnClick(parsePanelProps.OnShowSource), Class("cursor-pointer rounded-2xl border border-cyan-300/25 bg-cyan-400/12 px-4 py-2 text-sm font-medium text-cyan-100 transition hover:bg-cyan-400/18"), Text(buttonShowSource)),
						),
					),
				),
			),
		),
	)
}

// renderDisplaySurface chooses the appropriate detail renderer for the selected item.
func renderDisplaySurface(parsePanelProps contentPanelProps, hasSelectedItem bool) ui.Node {
	return ui.If(!hasSelectedItem,
		func() ui.Node {
			return Div(Class("flex min-h-full items-center justify-center rounded-[22px] border border-dashed border-white/10 bg-black/10 p-8 text-sm text-slate-400"), Text(messageNothingSelected))
		},
		func() ui.Node {
			return ui.Match().
				When(parsePanelProps.Item.Content.Kind == contentKindArticle, func() ui.Node {
					return renderConceptArticle(parsePanelProps)
				}).
				When(parsePanelProps.Item.Content.Kind == contentKindAPI, func() ui.Node {
					return renderAPIReference(parsePanelProps)
				}).
				Default(func() ui.Node {
					return ui.Component(renderCounterExample, parsePanelProps)
				})
		},
	)
}

// renderCatalogHero renders the top summary banner and primary CTA actions.
func renderCatalogHero(parseProps catalogHeroProps) ui.Node {
	parseRepoURL := "https://github.com/monstercameron/GoWebComponents"

	return Header(Class("relative overflow-hidden rounded-[28px] border border-white/10 bg-slate-950/55 p-4 shadow-2xl shadow-black/35 backdrop-blur-xl sm:p-5"),
		Div(Class("absolute inset-0 bg-[linear-gradient(135deg,rgba(255,255,255,0.05),transparent_38%,rgba(34,211,238,0.06))]")),
		Div(Class("pointer-events-none absolute -left-10 top-0 h-40 w-40 rounded-full bg-cyan-400/12 blur-3xl")),
		Div(Class("pointer-events-none absolute right-0 top-8 h-40 w-40 rounded-full bg-amber-300/10 blur-3xl")),
		Div(Class("relative"),
			Div(Class("min-w-0"),
				Div(Class("flex flex-wrap items-center gap-2"),
					Div(Class("inline-flex items-center gap-2 rounded-full border border-cyan-400/30 bg-cyan-400/10 px-3 py-1 text-xs font-medium uppercase tracking-[0.24em] text-cyan-200"),
						Span(Class("h-2 w-2 rounded-full bg-cyan-300")),
						Text("GoWebComponents Example Lab"),
					),
					A(
						FromProps(Props{Href: parseRepoURL, Target: "_blank", Rel: "noreferrer noopener", Class: "inline-flex items-center rounded-full border border-white/10 bg-slate-950/35 px-3 py-1 font-mono text-[11px] text-slate-300 transition hover:border-cyan-300/30 hover:text-cyan-100"}),
						Text("github.com/monstercameron/GoWebComponents"),
					),
				),
				Div(Class("mt-4 grid gap-4 lg:grid-cols-2 lg:gap-5 lg:items-stretch"),
					Div(Class("space-y-4 lg:min-w-0"),
						Div(Class("space-y-3"),
							H1(Class("max-w-3xl text-3xl font-semibold tracking-tight text-white sm:text-4xl lg:text-5xl"), Text("Runnable Go + WASM examples with the concepts up front.")),
							P(Class("max-w-3xl text-sm leading-6 text-slate-300 sm:text-base sm:leading-7"), Text("This gallery is example-first. Each item pairs a live runtime surface with the exact Go source that produced it, so the page elements, state flow, and browser behavior stay easier to study.")),
							P(Class("max-w-2xl text-xs uppercase tracking-[0.22em] text-slate-400 sm:text-[13px]"), Text("Live preview, concept framing, and mirrored source in one place.")),
						),
						Div(Class("flex flex-wrap gap-2 text-[11px] uppercase tracking-[0.18em] text-slate-300 sm:text-xs"),
							Span(Class("rounded-full border border-cyan-300/20 bg-cyan-400/10 px-3 py-1.5 text-cyan-100"), Text("Typed HTML builders")),
							Span(Class("rounded-full border border-white/10 bg-white/5 px-3 py-1.5"), Text("Hooks + state")),
							Span(Class("rounded-full border border-white/10 bg-white/5 px-3 py-1.5"), Text("Router + data")),
							Span(Class("rounded-full border border-white/10 bg-white/5 px-3 py-1.5"), Text("SSR + hydration")),
						),
						Div(Class("flex flex-wrap gap-2"),
							Button(Type("button"), OnClick(parseProps.OnBrowseExamples), Class("cursor-pointer rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-2 text-sm font-medium text-cyan-100 transition hover:-translate-y-0.5 hover:bg-cyan-400/20 active:translate-y-0"), Text(buttonBrowseExamples)),
							Button(Type("button"), OnClick(parseProps.OnInspectAPIs), Class("cursor-pointer rounded-2xl border border-white/10 bg-white/5 px-4 py-2 text-sm font-medium text-slate-100 transition hover:-translate-y-0.5 hover:bg-white/10 active:translate-y-0"), Text(buttonInspectPackageAPIs)),
						),
					),
					Div(Class("space-y-4 lg:min-w-0"),
						renderHeroCodeSnippet(),
						Div(Class("grid grid-cols-2 gap-2 sm:gap-3"),
							renderCountStatCard(parseProps.TotalItems, labelCatalogEntries),
							renderCountStatCard(parseProps.StableCount, labelLiveExamples),
							renderCountStatCard(parseProps.ExperimentalCount, "Experimental builds"),
							renderCountStatCard(parseProps.ModuleCount, "Module families"),
						),
					),
				),
			),
		),
	)
}

// renderCatalogSidebar renders the search, filters, and result list.
func renderCatalogSidebar(parseProps catalogSidebarProps) ui.Node {
	return Section(Class("min-w-0 flex min-h-[420px] flex-col rounded-[24px] border border-white/10 bg-slate-950/45 backdrop-blur-xl lg:sticky lg:top-3 lg:h-[calc(100vh-1.5rem)] lg:w-[34%] lg:flex-none xl:w-[31%]"),
		Div(Class("sticky top-0 z-10 border-b border-white/10 bg-slate-950/75 p-2.5 backdrop-blur-xl sm:p-3"),
			Div(Class("flex flex-col gap-2"),
				Div(Class("space-y-1"),
					Div(Class("text-xs uppercase tracking-[0.18em] text-cyan-200"), Text("Examples index")),
					P(Class("text-sm leading-6 text-slate-400"), Text("Search runnable examples by concept, module, or difficulty.")),
				),
				Div(Class("flex flex-col gap-2 sm:flex-row sm:items-center"),
					Div(Class("relative flex-1"),
						Input(ID("catalog-search-input"), Value(parseProps.SearchQuery), OnInput(parseProps.OnSearchInput), Placeholder("Search examples, concepts, modules..."), Aria("label", labelSearchDialog), Aria("describedby", parseProps.ResultSummaryID), Class("w-full rounded-xl border border-white/10 bg-slate-950/40 px-3 py-2 text-sm text-white outline-none placeholder:text-slate-500 transition focus:border-cyan-300/40 focus:bg-slate-950/60")),
					),
					Div(Class("flex items-center gap-2 self-start sm:self-auto"),
						Div(ID(parseProps.ResultSummaryID), Class("text-xs uppercase tracking-[0.18em] text-slate-400"), Aria("live", "polite"), Aria("atomic", "true"), Textf("%d results", parseProps.ResultCount)),
						Button(
							Type("button"),
							OnClick(parseProps.OnOpenSearchDialog),
							Class("cursor-pointer rounded-xl border border-cyan-300/25 bg-cyan-400/10 px-3 py-2 text-[11px] font-medium uppercase tracking-[0.16em] text-cyan-100 transition hover:bg-cyan-400/15 hover:text-white"),
							Text(buttonOpenSearchDialog),
						),
						Button(
							Type("button"),
							OnClick(parseProps.OnResetFilters),
							Disabled(!parseProps.HasActiveFilters),
							Class(ClassNames(
								"rounded-xl border px-3 py-2 text-[11px] font-medium uppercase tracking-[0.16em] transition",
								When(parseProps.HasActiveFilters, "cursor-pointer border-white/10 bg-white/5 text-slate-200 hover:bg-white/10 hover:text-white"),
								When(!parseProps.HasActiveFilters, "cursor-not-allowed border-white/5 bg-white/[0.03] text-slate-500"),
							)),
							Text(buttonResetFilters),
						),
					),
				),
				Div(Class("grid grid-cols-2 gap-2 xl:grid-cols-4"),
					Label(Class("flex flex-col gap-1 text-[11px] uppercase tracking-[0.16em] text-slate-500"),
						Span(Text("Status")),
						Select(Value(parseProps.SelectedStatusFilter), OnChange(parseProps.OnStatusChange), Class("rounded-xl border border-white/10 bg-slate-950/50 px-3 py-2 text-xs text-slate-100 outline-none"), renderOptionNodes(parseProps.Statuses)),
					),
					Label(Class("flex flex-col gap-1 text-[11px] uppercase tracking-[0.16em] text-slate-500"),
						Span(Text("Difficulty")),
						Select(Value(parseProps.SelectedLevelFilter), OnChange(parseProps.OnLevelChange), Class("rounded-xl border border-white/10 bg-slate-950/50 px-3 py-2 text-xs text-slate-100 outline-none"), renderOptionNodes(parseProps.Levels)),
					),
					Label(Class("flex flex-col gap-1 text-[11px] uppercase tracking-[0.16em] text-slate-500"),
						Span(Text("Module")),
						Select(Value(parseProps.SelectedModuleFilter), OnChange(parseProps.OnModuleChange), Class("rounded-xl border border-white/10 bg-slate-950/50 px-3 py-2 text-xs text-slate-100 outline-none"), renderOptionNodes(parseProps.Modules)),
					),
					Label(Class("flex flex-col gap-1 text-[11px] uppercase tracking-[0.16em] text-slate-500"),
						Span(Text("Sort")),
						Select(Value(parseProps.SelectedSortOrder), OnChange(parseProps.OnSortChange), Class("rounded-xl border border-white/10 bg-slate-950/50 px-3 py-2 text-xs text-slate-100 outline-none"), renderSortOptionNodes(parseProps.SortOptions)),
					),
				),
			),
		),
		Div(Class("scrollbar-stable min-h-0 flex-1 overflow-y-auto p-1.5 sm:p-2"),
			Div(Class("space-y-2"), parseProps.ItemNodes),
		),
	)
}

func renderCatalogSearchDialogContent(parseProps catalogSearchDialogProps) ui.Node {
	return Div(Class("flex max-h-[min(84vh,42rem)] flex-col gap-4"),
		Div(Class("flex items-start justify-between gap-4"),
			Div(Class("min-w-0"),
				H2(ID("catalog-search-dialog-title"), Class("text-2xl font-semibold tracking-tight text-white"), Text(labelSearchDialog)),
				P(ID("catalog-search-dialog-description"), Class("mt-2 text-sm leading-6 text-slate-300"), Text(messageSearchDialog)),
			),
			Button(Type("button"), ID("catalog-search-dialog-close"), OnClick(parseProps.OnClose), Aria("label", buttonCloseSearchDialog), Class("shrink-0 rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-slate-200 transition hover:bg-white/10 hover:text-white"), Text("Close")),
		),
		Input(ID("catalog-search-dialog-input"), Value(parseProps.SearchQuery), OnInput(parseProps.OnSearchInput), Placeholder("Search examples, concepts, modules..."), Aria("label", labelSearchDialog), Aria("describedby", parseProps.ResultSummaryID+" catalog-search-dialog-description"), Class("w-full rounded-xl border border-white/10 bg-slate-950/70 px-3 py-2 text-sm text-white outline-none placeholder:text-slate-500 transition focus:border-cyan-300/50 focus:bg-slate-950")),
		Div(Class("flex flex-wrap items-center justify-between gap-3"),
			Div(ID("catalog-search-dialog-summary"), Class("text-xs uppercase tracking-[0.18em] text-slate-400"), Aria("live", "polite"), Aria("atomic", "true"), Text(catalogResultAnnouncement(parseProps.ResultCount, parseProps.SearchQuery))),
			Button(
				Type("button"),
				OnClick(parseProps.OnResetFilters),
				Disabled(!parseProps.HasActiveFilters),
				Class(ClassNames(
					"rounded-xl border px-3 py-2 text-[11px] font-medium uppercase tracking-[0.16em] transition",
					When(parseProps.HasActiveFilters, "cursor-pointer border-white/10 bg-white/5 text-slate-200 hover:bg-white/10 hover:text-white"),
					When(!parseProps.HasActiveFilters, "cursor-not-allowed border-white/5 bg-white/[0.03] text-slate-500"),
				)),
				Text(buttonResetFilters),
			),
		),
		Div(Class("min-h-0 overflow-y-auto rounded-[22px] border border-white/10 bg-black/10 p-2"),
			IfElse(len(parseProps.ItemNodes) > 0,
				Div(Class("space-y-2"), parseProps.ItemNodes),
				Div(Class("p-6 text-center text-sm text-slate-400"), Text(messageNoMatches)),
			),
		),
	)
}

func catalogSearchOverlayProps(parseProps catalogSearchDialogProps, parseContent ui.Node) ui.AccessibleOverlayProps {
	return ui.AccessibleOverlayProps{
		Open:                  parseProps.Open,
		SurfaceID:             "catalog-search-dialog",
		Role:                  "dialog",
		Modal:                 true,
		LabelledBy:            "catalog-search-dialog-title",
		DescribedBy:           "catalog-search-dialog-description",
		InitialFocusSelector:  "#catalog-search-dialog-input",
		FallbackFocusSelector: "#catalog-search-dialog-close",
		RestoreFocus:          true,
		TrapFocus:             true,
		CloseOnEscape:         true,
		CloseOnOutsideClick:   true,
		LockScroll:            true,
		BackgroundInert:       true,
		Backdrop:              true,
		BaseZIndex:            80,
		BackdropClass:         "fixed inset-0 bg-slate-950/75 backdrop-blur-sm",
		SurfaceClass:          "fixed left-1/2 top-1/2 w-[min(44rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-[28px] border border-white/10 bg-slate-950 p-4 shadow-2xl shadow-black/50 sm:p-5",
		Child:                 parseContent,
		OnDismiss:             parseProps.OnDismiss,
	}
}

func renderCatalogSearchDialog(parseProps catalogSearchDialogProps) ui.Node {
	return ui.Component(ui.AccessibleOverlay, catalogSearchOverlayProps(parseProps, renderCatalogSearchDialogContent(parseProps)))
}

// renderDetailPanel renders the selected item header and its detail surface.
func renderDetailPanel(parseProps detailPanelProps) ui.Node {
	parseConceptNodes := renderExampleConceptChipNodes(buildExampleConceptLabels(parseProps.SelectedItem))
	parsePrimaryBadge := labelLivePreview
	if parseProps.SelectedItem.Type != kindExample && parseProps.SelectedItem.Content.Kind != contentKindExample && parseProps.SelectedItem.Content.Kind != contentKindCounter {
		parsePrimaryBadge = getContentKindLabel(parseProps.SelectedItem)
	}
	return Section(Class("min-w-0 flex min-h-[420px] flex-1 flex-col rounded-[24px] border border-white/10 bg-slate-950/45 backdrop-blur-xl lg:sticky lg:top-3 lg:h-[calc(100vh-1.5rem)]"),
		Div(Class("sticky top-0 z-10 border-b border-white/10 bg-slate-950/75 p-3 backdrop-blur-xl sm:p-4"),
			IfElse(parseProps.HasSelectedItem,
				Fragment(
					Div(Class("flex flex-wrap items-center gap-3"),
						Span(Class(ClassNames("rounded-full border px-3 py-1 text-xs font-medium uppercase tracking-[0.2em]", typeBadgeClass(parseProps.SelectedItem.Type))), Text(parsePrimaryBadge)),
						Span(Class("rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs uppercase tracking-[0.2em] text-slate-400"), Text(parseProps.SelectedItem.Level)),
						Span(Class("rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs uppercase tracking-[0.2em] text-slate-400"), Text(formatExampleConceptLabel(parseProps.SelectedItem.Module))),
						Span(Class(ClassNames("rounded-full border px-3 py-1 text-xs uppercase tracking-[0.2em]", statusBadgeClass(parseProps.SelectedItem.Status))), Text(parseProps.SelectedItem.Status)),
					),
					H2(Class("mt-4 text-2xl font-semibold tracking-tight text-white sm:text-3xl"), Text(parseProps.SelectedItem.Title)),
					P(Class("mt-3 max-w-3xl text-sm leading-7 text-slate-300 sm:text-base"), Text(getExampleFocusSummary(parseProps.SelectedItem))),
					If(len(parseConceptNodes) > 0,
						Div(Class("mt-4 flex flex-wrap gap-2"), parseConceptNodes),
					),
				),
				Fragment(
					H2(Class("text-2xl font-semibold tracking-tight text-white sm:text-3xl"), Text(labelNothingSelected)),
					P(Class("mt-3 max-w-3xl text-sm leading-7 text-slate-300 sm:text-base"), Text(messageAdjustFilters)),
				),
			),
		),
		Div(ID("demo"), Class("scrollbar-stable min-w-0 min-h-0 flex-1 overflow-y-auto p-2 sm:p-3"), renderDisplaySurface(contentPanelProps{Item: parseProps.SelectedItem, IsPreviewLoaded: parseProps.IsPreviewLoaded, IsSourceVisible: parseProps.IsSourceVisible, MarkdownBody: parseProps.MarkdownBody, MarkdownLoading: parseProps.MarkdownLoading, MarkdownReady: parseProps.MarkdownReady, MarkdownError: parseProps.MarkdownError, OnLoadPreview: parseProps.OnLoadPreview, OnHideSource: parseProps.OnHideSource, OnRetryMarkdown: parseProps.OnRetryMarkdown, OnShowSource: parseProps.OnShowSource}, parseProps.HasSelectedItem)),
	)
}

// renderDocsDemosSite wires together the search state, filters, selection state, and layout.
func renderDocsDemosSite() ui.Node {
	parseCatalogURL := catalogDataURL()
	parseCatalogResource := fetch.UseCachedResource(catalogCacheKey(parseCatalogURL), func(parseCtx context.Context) (docsCatalog, error) {
		return loadCatalogResource(parseCtx, parseCatalogURL)
	}, fetch.CacheOptions{StaleAfter: 45 * time.Second})
	parseCatalogRequest := parseCatalogResource.Get()
	isDebugLoggingEnabled := isPageDebugLoggingEnabled()
	parseAnnouncer := ui.UseAnnouncer()
	parseSearchDialogOpen := ui.UseState(false)
	parseRetryCatalogLoad := ui.UseEvent(func() {
		if isDebugLoggingEnabled {
			log.Info("catalog refetch requested", map[string]interface{}{"url": parseCatalogURL})
		}
		parseCatalogResource.Reload()
	})
	ui.UseEffect(func() func() {
		switch {
		case parseCatalogRequest.Loading:
			if isDebugLoggingEnabled {
				log.Info("catalog cache loading", map[string]interface{}{"url": parseCatalogURL, "stale": parseCatalogRequest.Stale, "ready": parseCatalogRequest.Ready})
			}
		case parseCatalogRequest.Error != nil:
			log.Error("catalog cache failed", map[string]interface{}{"url": parseCatalogURL, "error": parseCatalogRequest.Error.Error()})
		case parseCatalogRequest.Ready:
			if isDebugLoggingEnabled {
				log.Info("catalog cache ready", map[string]interface{}{
					"url":         parseCatalogURL,
					"modules":     len(parseCatalogRequest.Value.Modules),
					"statuses":    len(parseCatalogRequest.Value.Statuses),
					"levels":      len(parseCatalogRequest.Value.Levels),
					"filters":     len(parseCatalogRequest.Value.Filters),
					"sortOptions": len(parseCatalogRequest.Value.SortOptions),
					"items":       len(parseCatalogRequest.Value.Items),
					"stale":       parseCatalogRequest.Stale,
				})
			}
		}
		return nil
	}, parseCatalogRequest.Loading, parseCatalogRequest.Ready, parseCatalogRequest.Stale, fmt.Sprint(parseCatalogRequest.Error), len(parseCatalogRequest.Value.Items))

	parseSearchQuery := ui.UseState("")
	parseDeferredSearchQuery := ui.UseDeferredValue(parseSearchQuery.Get())
	parseActiveTypeFilter := ui.UseState(kindExample)
	parseSelectedStatusFilter := ui.UseState(allFilterValue)
	parseSelectedLevelFilter := ui.UseState(allFilterValue)
	parseSelectedModuleFilter := ui.UseState(allFilterValue)
	parseSelectedSortOrder := ui.UseState(sortRelevance)
	parseSelectedItemID := ui.UseState(0)
	parseAnchorScrollItemID := ui.UseState(0)
	parseAnchorScrollRequestID := ui.UseState(0)
	parseLoadedPreviewItemID := ui.UseState(0)
	parseVisibleSourceItemID := ui.UseState(0)

	parseUpdateSearchQuery := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseSearchQuery.Set(parseEvent.GetValue())
		if isDebugLoggingEnabled {
			log.Info("search query updated", map[string]interface{}{"query": parseEvent.GetValue()})
		}
	})
	parseUpdateStatusFilter := ui.UseEvent(func(parseEvent2 ui.ChangeEvent) {
		parseSelectedStatusFilter.Set(parseEvent2.GetValue())
		if isDebugLoggingEnabled {
			log.Info("status filter updated", map[string]interface{}{"status": parseEvent2.GetValue()})
		}
	})
	parseUpdateLevelFilter := ui.UseEvent(func(parseEvent3 ui.ChangeEvent) {
		parseSelectedLevelFilter.Set(parseEvent3.GetValue())
		if isDebugLoggingEnabled {
			log.Info("level filter updated", map[string]interface{}{"level": parseEvent3.GetValue()})
		}
	})
	parseUpdateModuleFilter := ui.UseEvent(func(parseEvent4 ui.ChangeEvent) {
		parseSelectedModuleFilter.Set(parseEvent4.GetValue())
		if isDebugLoggingEnabled {
			log.Info("module filter updated", map[string]interface{}{"module": parseEvent4.GetValue()})
		}
	})
	parseUpdateSortOrder := ui.UseEvent(func(parseEvent5 ui.ChangeEvent) {
		parseSelectedSortOrder.Set(parseEvent5.GetValue())
		if isDebugLoggingEnabled {
			log.Info("sort order updated", map[string]interface{}{"sort": parseEvent5.GetValue()})
		}
	})
	resetFilters := ui.UseEvent(func() {
		parseActiveTypeFilter.Set(kindExample)
		parseSelectedStatusFilter.Set(allFilterValue)
		parseSelectedLevelFilter.Set(allFilterValue)
		parseSelectedModuleFilter.Set(allFilterValue)
		parseSelectedSortOrder.Set(sortRelevance)
		parseSelectedItemID.Set(0)
		parseLoadedPreviewItemID.Set(0)
		parseVisibleSourceItemID.Set(0)
		if isDebugLoggingEnabled {
			log.Info("filters reset", map[string]interface{}{
				"type":   kindExample,
				"status": allFilterValue,
				"level":  allFilterValue,
				"module": allFilterValue,
				"sort":   sortRelevance,
			})
		}
	})
	hasActiveFilters := parseSelectedStatusFilter.Get() != allFilterValue || parseSelectedLevelFilter.Get() != allFilterValue || parseSelectedModuleFilter.Get() != allFilterValue || parseSelectedSortOrder.Get() != sortRelevance

	parseExampleItems := ui.UseMemo(func() []docsItem {
		return getExampleCatalogItems(parseCatalogRequest.Value.Items)
	}, len(parseCatalogRequest.Value.Items))
	parseVisibleStatuses := ui.UseMemo(func() []string {
		return getVisibleCatalogValues(parseCatalogRequest.Value.Statuses, parseExampleItems, func(parseItem docsItem) string { return parseItem.Status })
	}, len(parseCatalogRequest.Value.Statuses), len(parseExampleItems))
	parseVisibleLevels := ui.UseMemo(func() []string {
		return getVisibleCatalogValues(parseCatalogRequest.Value.Levels, parseExampleItems, func(parseItem docsItem) string { return parseItem.Level })
	}, len(parseCatalogRequest.Value.Levels), len(parseExampleItems))
	parseVisibleModules := ui.UseMemo(func() []string {
		return getVisibleCatalogValues(parseCatalogRequest.Value.Modules, parseExampleItems, func(parseItem docsItem) string { return parseItem.Module })
	}, len(parseCatalogRequest.Value.Modules), len(parseExampleItems))
	parseFilteredItems := ui.UseMemo(func() []docsItem {
		return sortItems(filterItems(parseExampleItems, parseDeferredSearchQuery, parseActiveTypeFilter.Get(), parseSelectedStatusFilter.Get(), parseSelectedLevelFilter.Get(), parseSelectedModuleFilter.Get()), parseSelectedSortOrder.Get())
	}, len(parseExampleItems), parseDeferredSearchQuery, parseActiveTypeFilter.Get(), parseSelectedStatusFilter.Get(), parseSelectedLevelFilter.Get(), parseSelectedModuleFilter.Get(), parseSelectedSortOrder.Get())
	ui.UseEffect(func() func() {
		if parseCatalogRequest.Ready {
			parseAnnouncer.Polite(catalogResultAnnouncement(len(parseFilteredItems), parseDeferredSearchQuery))
		}
		return nil
	}, parseCatalogRequest.Ready, len(parseFilteredItems), parseDeferredSearchQuery, parseSelectedStatusFilter.Get(), parseSelectedLevelFilter.Get(), parseSelectedModuleFilter.Get(), parseSelectedSortOrder.Get())
	ui.UseEffect(func() func() {
		if isDebugLoggingEnabled {
			log.Info("filters applied", map[string]interface{}{
				"query":       parseSearchQuery.Get(),
				"deferred":    parseDeferredSearchQuery,
				"type":        parseActiveTypeFilter.Get(),
				"status":      parseSelectedStatusFilter.Get(),
				"level":       parseSelectedLevelFilter.Get(),
				"module":      parseSelectedModuleFilter.Get(),
				"sort":        parseSelectedSortOrder.Get(),
				"resultCount": len(parseFilteredItems),
			})
		}
		return nil
	}, parseSearchQuery.Get(), parseDeferredSearchQuery, parseActiveTypeFilter.Get(), parseSelectedStatusFilter.Get(), parseSelectedLevelFilter.Get(), parseSelectedModuleFilter.Get(), parseSelectedSortOrder.Get(), filteredItemsSignature(parseFilteredItems))
	ui.UseEffect(func() func() {
		// Keep selection aligned with the filtered list so the detail panel never points at a stale item.
		if len(parseFilteredItems) == 0 {
			parseSelectedItemID.Set(0)
			parseAnchorScrollItemID.Set(0)
			parseLoadedPreviewItemID.Set(0)
			parseVisibleSourceItemID.Set(0)
			clearRequestedDemoAnchor()
			if isDebugLoggingEnabled {
				log.Info("selection cleared", map[string]interface{}{"reason": "no filtered items"})
			}
			return nil
		}
		parseCurrentSelectedID := parseSelectedItemID.Get()
		if parseCurrentSelectedID == 0 {
			return nil
		}
		for _, parseItem := range parseFilteredItems {
			if parseItem.ID == parseCurrentSelectedID {
				return nil
			}
		}
		parseSelectedItemID.Set(parseFilteredItems[0].ID)
		parseAnchorScrollItemID.Set(0)
		clearRequestedDemoAnchor()
		if isDebugLoggingEnabled {
			log.Info("selection repaired", map[string]interface{}{"selectedItemID": parseFilteredItems[0].ID})
		}
		return nil
	}, filteredItemsSignature(parseFilteredItems), parseSelectedItemID.Get(), isDebugLoggingEnabled)

	parseSelectedCatalogItem, hasSelectedItem := findSelectedItem(parseFilteredItems, parseSelectedItemID.Get())
	parseSelectedDocURL := ""
	if hasSelectedItem && parseSelectedCatalogItem.Content.SourcePath != "" && (parseSelectedCatalogItem.Content.Kind == contentKindArticle || parseSelectedCatalogItem.Content.Kind == contentKindAPI) {
		parseSelectedDocURL = docsSourceURL(parseSelectedCatalogItem.Content.SourcePath)
	}
	parseMarkdownResource := fetch.UseCachedResource(markdownCacheKey(parseSelectedDocURL), func(parseCtx2 context.Context) (string, error) {
		return loadMarkdownResource(parseCtx2, parseSelectedDocURL)
	}, fetch.CacheOptions{StaleAfter: 2 * time.Minute})
	parseMarkdownRequest := parseMarkdownResource.Get()
	parseRetryMarkdownLoad := ui.UseEvent(func() {
		if parseSelectedDocURL == "" {
			return
		}
		if isDebugLoggingEnabled {
			log.Info("markdown refetch requested", map[string]interface{}{"url": parseSelectedDocURL, "sourcePath": parseSelectedCatalogItem.Content.SourcePath})
		}
		parseMarkdownResource.Reload()
	})
	ui.UseEffect(func() func() {
		if parseSelectedDocURL == "" {
			return nil
		}
		switch {
		case parseMarkdownRequest.Loading:
			if isDebugLoggingEnabled {
				log.Info("markdown cache loading", map[string]interface{}{"url": parseSelectedDocURL, "sourcePath": parseSelectedCatalogItem.Content.SourcePath, "ready": parseMarkdownRequest.Ready, "stale": parseMarkdownRequest.Stale})
			}
		case parseMarkdownRequest.Error != nil:
			log.Error("markdown cache failed", map[string]interface{}{"url": parseSelectedDocURL, "sourcePath": parseSelectedCatalogItem.Content.SourcePath, "error": parseMarkdownRequest.Error.Error()})
		case parseMarkdownRequest.Ready:
			if isDebugLoggingEnabled {
				log.Info("markdown cache ready", map[string]interface{}{"url": parseSelectedDocURL, "sourcePath": parseSelectedCatalogItem.Content.SourcePath, "bytes": len(parseMarkdownRequest.Value)})
			}
		}
		return nil
	}, parseSelectedDocURL, parseMarkdownRequest.Loading, parseMarkdownRequest.Ready, parseMarkdownRequest.Stale, errorString(parseMarkdownRequest.Error), len(parseMarkdownRequest.Value), parseSelectedItemID.Get())
	parseBrowseExamples := ui.UseEvent(func() {
		parseActiveTypeFilter.Set(kindExample)
		parseSearchQuery.Set("")
		parseSelectedStatusFilter.Set(allFilterValue)
		parseSelectedLevelFilter.Set(allFilterValue)
		parseSelectedModuleFilter.Set(allFilterValue)
		parseSelectedSortOrder.Set(sortRelevance)
		parseSelectedItemID.Set(getCatalogItemIDByTitle(parseExampleItems, "Counter"))
		parseLoadedPreviewItemID.Set(0)
		parseVisibleSourceItemID.Set(0)
		clearRequestedDemoAnchor()
		if isDebugLoggingEnabled {
			log.Info("quick filter selected", map[string]interface{}{"type": kindExample})
		}
	})
	parseInspectPackageAPIs := ui.UseEvent(func() {
		parseActiveTypeFilter.Set(kindExample)
		parseSelectedLevelFilter.Set(levelAdvanced)
		parseSelectedStatusFilter.Set(allFilterValue)
		parseSelectedModuleFilter.Set(allFilterValue)
		parseSelectedSortOrder.Set(sortLevel)
		parseSelectedItemID.Set(0)
		parseLoadedPreviewItemID.Set(0)
		parseVisibleSourceItemID.Set(0)
		clearRequestedDemoAnchor()
		if isDebugLoggingEnabled {
			log.Info("quick filter selected", map[string]interface{}{"type": kindExample, "level": levelAdvanced})
		}
	})
	parseLoadSelectedPreview := ui.UseEvent(func() {
		if !hasSelectedItem {
			return
		}
		parseLoadedPreviewItemID.Set(parseSelectedCatalogItem.ID)
		if isDebugLoggingEnabled {
			log.Info("preview requested", map[string]interface{}{"itemID": parseSelectedCatalogItem.ID, "title": parseSelectedCatalogItem.Title})
		}
	})
	parseShowSelectedSource := ui.UseEvent(func() {
		if !hasSelectedItem {
			return
		}
		parseVisibleSourceItemID.Set(parseSelectedCatalogItem.ID)
		if isDebugLoggingEnabled {
			log.Info("source revealed", map[string]interface{}{"itemID": parseSelectedCatalogItem.ID, "title": parseSelectedCatalogItem.Title})
		}
	})
	parseHideSelectedSource := ui.UseEvent(func() {
		if !hasSelectedItem {
			return
		}
		parseVisibleSourceItemID.Set(0)
		if isDebugLoggingEnabled {
			log.Info("source hidden", map[string]interface{}{"itemID": parseSelectedCatalogItem.ID, "title": parseSelectedCatalogItem.Title})
		}
	})
	parseDismissSearchDialog := func() {
		parseSearchDialogOpen.Set(false)
		parseAnnouncer.Polite("Search dialog closed.")
	}
	parseOpenSearchDialog := ui.UseEvent(func() {
		parseSearchDialogOpen.Set(true)
		parseAnnouncer.Polite("Search dialog opened. " + catalogResultAnnouncement(len(parseFilteredItems), parseDeferredSearchQuery))
	})
	parseCloseSearchDialog := ui.UseEvent(func() {
		parseDismissSearchDialog()
	})
	ui.UseEffect(func() func() {
		if !hasSelectedItem {
			return nil
		}
		if isDebugLoggingEnabled {
			log.Info("detail item selected", map[string]interface{}{
				"itemID": parseSelectedCatalogItem.ID,
				"title":  parseSelectedCatalogItem.Title,
				"type":   parseSelectedCatalogItem.Type,
			})
		}
		return nil
	}, parseSelectedItemID.Get(), filteredItemsSignature(parseFilteredItems))

	parseSelectCatalogItem := func(parseItem docsItem) {
		parseSelectedItemID.Set(parseItem.ID)
		parseAnchorScrollItemID.Set(parseItem.ID)
		parseAnchorScrollRequestID.Set(parseAnchorScrollRequestID.Get() + 1)
		if parseItem.Content.AnchorID != "" {
			setRequestedDemoAnchor(parseItem.Content.AnchorID)
		} else {
			clearRequestedDemoAnchor()
		}
		if isDebugLoggingEnabled {
			log.Info("catalog card clicked", map[string]interface{}{"itemID": parseItem.ID, "title": parseItem.Title})
		}
	}
	parseItemNodes := Map(parseFilteredItems, func(parseItem2 docsItem) ui.Node {
		return renderItemCard(parseItem2, hasSelectedItem && parseSelectedCatalogItem.ID == parseItem2.ID, ui.UseEvent(func() {
			parseSelectCatalogItem(parseItem2)
		}))
	})
	parseItemNodes = append(parseItemNodes, If(len(parseFilteredItems) == 0,
		Div(Class("rounded-[22px] border border-dashed border-white/10 bg-black/10 p-6 text-center text-sm text-slate-400"), Text(messageNoMatches)),
	))
	parseDialogItemNodes := Map(firstCatalogItems(parseFilteredItems, 6), func(parseItem3 docsItem) ui.Node {
		return renderItemCard(parseItem3, hasSelectedItem && parseSelectedCatalogItem.ID == parseItem3.ID, ui.UseEvent(func() {
			parseSelectCatalogItem(parseItem3)
			parseSearchDialogOpen.Set(false)
			parseAnnouncer.Polite(parseItem3.Title + " selected. " + catalogResultAnnouncement(len(parseFilteredItems), parseDeferredSearchQuery))
		}))
	})

	parseCatalogErrorMessage := ""
	if parseCatalogRequest.Error != nil {
		parseCatalogErrorMessage = parseCatalogRequest.Error.Error()
	}

	return ui.Match().
		When(!parseCatalogRequest.Ready && parseCatalogRequest.Error == nil, func() ui.Node {
			return renderCatalogFetchState("Loading examples", "Requesting the runnable example catalog before the gallery renders.", "Retry request", parseRetryCatalogLoad)
		}).
		When(parseCatalogRequest.Error != nil && !parseCatalogRequest.Ready, func() ui.Node {
			return renderCatalogFetchState("Catalog request failed", parseCatalogErrorMessage, "Retry request", parseRetryCatalogLoad)
		}).
		Default(func() ui.Node {
			parseAnchorScrollID := 0
			if hasSelectedItem && parseAnchorScrollItemID.Get() == parseSelectedCatalogItem.ID {
				parseAnchorScrollID = parseAnchorScrollRequestID.Get()
			}
			return Div(Class("min-h-screen bg-[radial-gradient(circle_at_top_left,rgba(34,211,238,0.14),transparent_24%),radial-gradient(circle_at_top_right,rgba(251,191,36,0.10),transparent_22%),linear-gradient(180deg,#08111d_0%,#0d1726_44%,#0c1320_100%)] text-slate-100"),
				parseAnnouncer.Region(),
				Div(Class("mx-auto flex min-h-screen w-full max-w-[96rem] flex-col px-2 py-2 sm:px-3 sm:py-3 lg:px-4"),
					ui.Component(renderCatalogHero, catalogHeroProps{
						OnBrowseExamples:  parseBrowseExamples,
						OnInspectAPIs:     parseInspectPackageAPIs,
						TotalItems:        len(parseExampleItems),
						ExampleCount:      len(parseExampleItems),
						APICount:          getStatusItemCount(parseExampleItems, statusExperimental),
						StableCount:       getStatusItemCount(parseExampleItems, statusStable),
						ExperimentalCount: getStatusItemCount(parseExampleItems, statusExperimental),
						ModuleCount:       getDistinctModuleCount(parseExampleItems),
					}),
					Main(Class("mt-2 flex w-full flex-1 flex-col gap-2 lg:min-h-0 lg:flex-row"),
						ui.Component(renderCatalogSidebar, catalogSidebarProps{SearchQuery: parseSearchQuery.Get(), ResultCount: len(parseFilteredItems), ResultSummaryID: catalogSearchResultSummaryID, HasActiveFilters: hasActiveFilters, Statuses: parseVisibleStatuses, Levels: parseVisibleLevels, Modules: parseVisibleModules, SortOptions: parseCatalogRequest.Value.SortOptions, SelectedStatusFilter: parseSelectedStatusFilter.Get(), SelectedLevelFilter: parseSelectedLevelFilter.Get(), SelectedModuleFilter: parseSelectedModuleFilter.Get(), SelectedSortOrder: parseSelectedSortOrder.Get(), ItemNodes: parseItemNodes, OnSearchInput: parseUpdateSearchQuery, OnStatusChange: parseUpdateStatusFilter, OnLevelChange: parseUpdateLevelFilter, OnModuleChange: parseUpdateModuleFilter, OnSortChange: parseUpdateSortOrder, OnResetFilters: resetFilters, OnOpenSearchDialog: parseOpenSearchDialog}),
						ui.Component(renderDetailPanel, detailPanelProps{SelectedItem: parseSelectedCatalogItem, HasSelectedItem: hasSelectedItem, IsPreviewLoaded: hasSelectedItem && parseLoadedPreviewItemID.Get() == parseSelectedCatalogItem.ID, IsSourceVisible: hasSelectedItem && parseVisibleSourceItemID.Get() == parseSelectedCatalogItem.ID, MarkdownBody: parseMarkdownRequest.Value, MarkdownLoading: parseMarkdownRequest.Loading, MarkdownReady: parseMarkdownRequest.Ready, MarkdownError: errorString(parseMarkdownRequest.Error), AnchorScrollID: parseAnchorScrollID, OnLoadPreview: parseLoadSelectedPreview, OnHideSource: parseHideSelectedSource, OnRetryMarkdown: parseRetryMarkdownLoad, OnShowSource: parseShowSelectedSource}),
					),
				),
				ui.Component(renderCatalogSearchDialog, catalogSearchDialogProps{Open: parseSearchDialogOpen.Get(), SearchQuery: parseSearchQuery.Get(), ResultCount: len(parseFilteredItems), ResultSummaryID: catalogSearchResultSummaryID, HasActiveFilters: hasActiveFilters, ItemNodes: parseDialogItemNodes, OnSearchInput: parseUpdateSearchQuery, OnResetFilters: resetFilters, OnClose: parseCloseSearchDialog, OnDismiss: parseDismissSearchDialog}),
			)
		})
}

func main() {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			_, _ = runtime.FinalizeUnhandledPanicContext("runtime", runtime.PanicPhaseStartup, "main", "#app", nil, parseRecovered)
		}
	}()

	utils.DisableAllDebug()
	if isPageDebugLoggingEnabled() {
		logging.AttachBrowserConsole(logging.BrowserConsoleOptions{Scope: "public-examples-site"})
	}
	ui.Render(ui.Component(renderDocsDemosSite), "#app")
	utils.WaitForever()
}
