package render

import (
	"strconv"
	"testing"

	"github.com/monstercameron/GoWebComponents/v7/html"
	"github.com/monstercameron/GoWebComponents/v7/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v7/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v7/ui"
)

const mirrorContentCardCount = 12

type mirrorContentCard struct {
	GetID      int
	GetTitle   string
	GetSummary string
	GetStatus  string
	GetMeta    string
	GetTags    []string
}

type mirrorContentCardProps struct {
	GetItem         mirrorContentCard
	GetRefreshToken int
}

func buildMirrorContentCards() []mirrorContentCard {
	getItems := make([]mirrorContentCard, mirrorContentCardCount)
	for parseIndex := range getItems {
		getItems[parseIndex] = mirrorContentCard{
			GetID: parseIndex, GetTitle: "Article " + strconv.Itoa(parseIndex),
			GetSummary: "This benchmark card exercises browser-side render and update work with nested content blocks.",
			GetStatus:  "draft", GetMeta: "Section " + strconv.Itoa((parseIndex%3)+1),
			GetTags: []string{"perf", "browser", "card-" + strconv.Itoa(parseIndex%4)},
		}
	}
	return getItems
}

func renderMirrorContentCard(parseProps mirrorContentCardProps) ui.Node {
	getTagNodes := make([]ui.Node, 0, len(parseProps.GetItem.GetTags))
	for _, getTag := range parseProps.GetItem.GetTags {
		getTagNodes = append(getTagNodes, html.Span(html.Props{Class: "benchmark-content-tag rounded-full border border-white/10 px-2 py-1 text-[11px] uppercase tracking-[0.14em] text-slate-300"}, html.Text(getTag)))
	}
	return html.Article(html.Props{
		Class:     "benchmark-content-card rounded-2xl border border-white/10 bg-white/[0.04] p-4 shadow-lg shadow-black/20",
		DataAttrs: []html.DataAttribute{{Name: "refresh-token", Value: strconv.Itoa(parseProps.GetRefreshToken)}, {Name: "card-id", Value: strconv.Itoa(parseProps.GetItem.GetID)}},
	},
		html.Div(html.Props{Class: "flex items-center justify-between gap-3"},
			html.H2(html.Props{Class: "benchmark-content-title text-base font-semibold text-white"}, html.Text(parseProps.GetItem.GetTitle)),
			html.Span(html.Props{Class: "benchmark-content-status rounded-full border border-cyan-400/20 bg-cyan-400/10 px-2 py-1 text-[11px] uppercase tracking-[0.16em] text-cyan-100"}, html.Text(parseProps.GetItem.GetStatus))),
		html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"}, html.Text(parseProps.GetItem.GetSummary)),
		html.P(html.Props{Class: "benchmark-content-meta mt-3 text-xs uppercase tracking-[0.16em] text-slate-400"}, html.Text(parseProps.GetItem.GetMeta)),
		html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-2"}, getTagNodes...),
	)
}

func newMirrorContentRefreshHost(parseB *testing.B) (func(int), *mockdom.MockScheduler) {
	parseB.Helper()
	getAdapter := mockdom.NewMockDOMAdapter()
	getScheduler := mockdom.NewMockScheduler(false)
	getRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: getAdapter, Scheduler: getScheduler, Reset: true})
	getRoot := getAdapter.CreateElement("div")
	getItems := buildMirrorContentCards()
	getTypedCard := ui.Typed(renderMirrorContentCard)
	var storeSetToken func(any)
	getComponent := func() ui.Node {
		getToken, parseSetToken := runtime.GoUseState(getRuntime, 0)
		storeSetToken = parseSetToken
		getCardNodes := make([]ui.Node, 0, len(getItems))
		for _, getItem := range getItems {
			getCardNodes = append(getCardNodes, getTypedCard(mirrorContentCardProps{GetItem: getItem, GetRefreshToken: getToken()}))
		}
		return html.Div(html.Props{ID: "content-container", Class: "grid gap-4 lg:grid-cols-2", DataAttr: html.DataAttribute{Name: "refresh-token", Value: strconv.Itoa(getToken())}}, getCardNodes...)
	}
	if parseErr := getRuntime.RenderInto(getRoot, ui.CreateElement(getComponent)); parseErr != nil {
		parseB.Fatalf("RenderInto: %v", parseErr)
	}
	getScheduler.FlushAll()
	return func(parseToken int) { storeSetToken(parseToken) }, getScheduler
}

// BenchmarkMirrorContentRefresh12 mirrors Example 201's content-refresh: the
// stable 12-card payload is rebuilt through typed components with a new token.
func BenchmarkMirrorContentRefresh12(parseB *testing.B) {
	getSetToken, getScheduler := newMirrorContentRefreshHost(parseB)
	parseB.ReportAllocs()
	for parseI := 0; parseB.Loop(); parseI++ {
		getSetToken(parseI + 1)
		getScheduler.FlushAll()
	}
}
