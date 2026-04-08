//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"encoding/json"
	"strings"
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/fetch"
	gwchtml "github.com/monstercameron/GoWebComponents/html"
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/testkit/render"
	"github.com/monstercameron/GoWebComponents/ui"
)

func testArticleItem() docsItem {
	return docsItem{
		ID:       1,
		Title:    "Start With GoWebComponents",
		Status:   statusStable,
		Module:   moduleCore,
		Type:     kindConcept,
		Level:    levelBeginner,
		Tags:     []string{"getting-started", "architecture"},
		Blurb:    "A guided overview of the package model and rendering surface.",
		ReadTime: "8 min",
		Content: docsContent{
			Kind:       contentKindArticle,
			Callout:    "Start here before wiring larger examples.",
			SourcePath: "assets/docs/start-here.md",
			Code:       "func main() { ui.Render(app, \"#app\") }",
			Sections: []docsSection{{
				Heading:    "Core model",
				Paragraphs: []string{"Components are plain Go functions.", "State and effects stay colocated with the component tree."},
			}},
		},
	}
}

func testArticleMarkdown() string {
	return "# Start Here\n\nThis is the first paragraph with a [guide](troubleshooting.md).\n\n## Checklist\n\n- First\n- Second\n\n```go\nfunc main() {}\n```\n"
}

func testAPIItem() docsItem {
	return docsItem{
		ID:       2,
		Title:    "UseCachedResource",
		Status:   statusExperimental,
		Module:   moduleData,
		Type:     kindAPI,
		Level:    levelCore,
		Tags:     []string{"cache", "fetch"},
		Blurb:    "Typed cached resources with stale-while-revalidate semantics.",
		ReadTime: readTimeReference,
		Content: docsContent{
			Kind:      contentKindAPI,
			Signature: "func UseCachedResource[T any](key string, loader func(context.Context) (T, error), options ...CacheOptions) CachedResource[T]",
			Summary:   "Caches async values by key and exposes loading, stale, and reload state.",
			Returns:   "CachedResource[T] with Get, Reload, Cancel, Invalidate, Set, and Update.",
			Example:   "users := fetch.UseCachedResource(\"users\", loader)",
			Notes:     []string{"Use a stable cache key per resource.", "Combine with stale timestamps for background refreshes."},
			Params: []docsParam{{
				Name:        "key",
				Type:        "string",
				Required:    "yes",
				Description: "Stable cache key for the resource entry.",
			}, {
				Name:        "loader",
				Type:        "func(context.Context) (T, error)",
				Required:    "yes",
				Description: "Loader invoked when the resource becomes stale or is reloaded.",
			}},
		},
	}
}

func testHTMLAPIItem() docsItem {
	parseItem := testAPIItem()
	parseItem.ID = 6
	parseItem.Title = "RenderToString"
	parseItem.Content.SourcePath = "assets/docs/public-api-reference.html"
	parseItem.Content.AnchorID = "core-rendering"
	parseItem.Content.Example = ""
	return parseItem
}

func testGroupedAPIItem() docsItem {
	return docsItem{
		ID:         59,
		Title:      "Core Rendering Primitives",
		Status:     statusStable,
		Module:     moduleRendering,
		Type:       kindAPI,
		Level:      levelCore,
		Tags:       []string{"ui", "render", "dom"},
		SearchTags: []string{"CreateElement", "RenderInto", "RenderToString"},
		Blurb:      "Render Go component trees, create nodes, and move output into the DOM or string output paths.",
		ReadTime:   readTimeReference,
		Content: docsContent{
			Kind:       contentKindAPI,
			AnchorID:   "core-rendering",
			SourcePath: "assets/docs/public-api-reference.html",
			Summary:    "This group is the base rendering surface for composing nodes, creating host elements, and turning component trees into HTML or live DOM output.",
		},
	}
}

func testExampleItem() docsItem {
	return docsItem{
		ID:       3,
		Title:    "Go Counter Demo",
		Status:   statusStable,
		Module:   moduleState,
		Type:     kindExample,
		Level:    levelIntermediate,
		Tags:     []string{"state", "events"},
		Blurb:    "Interactive counter demo that exercises reactive state updates.",
		ReadTime: readTimeInteractive,
		Content: docsContent{
			Kind:        contentKindCounter,
			Description: "Update local state from Go event handlers and inspect the resulting UI tone.",
			Tips:        []string{"Use hooks for local state.", "Prefer stable event handlers for repeated renders."},
		},
	}
}

func testEmbeddedExampleItem() docsItem {
	parseItem := testExampleItem()
	parseItem.ID = 73
	parseItem.Title = "Counter Example"
	parseItem.Content.EmbedPath = "assets/bins/counter.wasm"
	parseItem.Content.PreviewPath = "assets/examples/counter/index.html"
	parseItem.Content.SourcePath = "assets/code/counter/main.go"
	parseItem.Content.Code = "fallback"
	return parseItem
}

func testSourceFirstExampleItem() docsItem {
	parseItem := testExampleItem()
	parseItem.ID = 74
	parseItem.Title = "Browser Router"
	parseItem.Content.Kind = contentKindExample
	parseItem.Content.SourcePath = "assets/code/browser-router/main.go"
	parseItem.Content.Description = "This example depends on routing or other browser-global behavior, so the catalog keeps it source-first."
	parseItem.Content.Tips = []string{
		"Review the mirrored main.go source from the public examples tree.",
		"Run the example package directly when you need the full standalone browser flow.",
	}
	parseItem.Content.Code = "package main\nfunc main() {}"
	return parseItem
}

func testDeprecatedExampleItem() docsItem {
	return docsItem{
		ID:       4,
		Title:    "Atlas Commerce OS",
		Status:   statusDeprecated,
		Module:   moduleCommerce,
		Type:     kindExample,
		Level:    levelAdvanced,
		Tags:     []string{"atlas", "commerce", "ssr"},
		Blurb:    "The production-shaped reference app for SSR, hydration, routes, mutations, and reviewer-facing docs.",
		ReadTime: "15 min",
		Content: docsContent{
			Kind:        contentKindCounter,
			Description: "A richer example surface used for reviewer walkthroughs.",
			Tips:        []string{"Exercise real routes.", "Observe hydration and mutations together."},
		},
	}
}

func testUnknownLevelItem() docsItem {
	return docsItem{
		ID:       5,
		Title:    "Zeta Unknown",
		Status:   statusStable,
		Module:   modulePlugins,
		Type:     kindConcept,
		Level:    "Unknown",
		Tags:     []string{"edge"},
		Blurb:    "Used to verify unknown levels sink to the bottom during sorting.",
		ReadTime: "3 min",
		Content:  docsContent{Kind: contentKindArticle, Callout: "Unknown level content.", Code: "// noop", Sections: []docsSection{{Heading: "Unknown", Paragraphs: []string{"Unknown level items sort last."}}}},
	}
}

func testCatalog() docsCatalog {
	return docsCatalog{
		Modules:  []string{allFilterValue, moduleCore, moduleData, moduleState, moduleCommerce, moduleRendering},
		Statuses: []string{allFilterValue, statusStable, statusExperimental, statusDeprecated},
		Levels:   []string{allFilterValue, levelBeginner, levelCore, levelIntermediate, levelAdvanced},
		Filters:  []string{filterAll, kindConcept, kindAPI, kindExample},
		SortOptions: []sortOption{{
			Value: sortRelevance,
			Label: labelRelevance,
		}, {
			Value: sortAlpha,
			Label: labelAlpha,
		}, {
			Value: sortLevel,
			Label: labelLevel,
		}},
		Items: []docsItem{testArticleItem(), testAPIItem(), testExampleItem(), testDeprecatedExampleItem(), testGroupedAPIItem()},
	}
}

func mustMarshalCatalog(parseT *testing.T, parseCatalog docsCatalog) string {
	parseT.Helper()
	parsePayload, parseErr := json.Marshal(parseCatalog)
	if parseErr != nil {
		parseT.Fatalf("json.Marshal catalog failed: %v", parseErr)
	}
	return string(parsePayload)
}

func installMockFetchText(parseT *testing.T, parsePayload string, parseStatus int, parseStatusText string) {
	parseT.Helper()
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parsePromiseCtor := parseGlobal.Get("Promise")
	parsePrevFetch := parseGlobal.Get("fetch")

	parseHeaders := parseObjectCtor.New()
	parseHeadersForEach := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return nil
	})
	parseHeaders.Set("forEach", parseHeadersForEach)

	parseResponse := parseObjectCtor.New()
	parseResponse.Set("status", parseStatus)
	parseResponse.Set("statusText", parseStatusText)
	parseResponse.Set("headers", parseHeaders)
	parseTextFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		return parsePromiseCtor.Call("resolve", parsePayload)
	})
	parseResponse.Set("text", parseTextFn)

	parseFetchFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		return parsePromiseCtor.Call("resolve", parseResponse)
	})

	parseGlobal.Set("fetch", parseFetchFn)
	parseT.Cleanup(func() {
		parseGlobal.Set("fetch", parsePrevFetch)
		parseFetchFn.Release()
		parseTextFn.Release()
		parseHeadersForEach.Release()
	})
}

func installMockDocumentBaseURI(parseT *testing.T, parseHref string) {
	parseT.Helper()
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parsePrevDocument := parseGlobal.Get("document")
	parsePrevWindow := parseGlobal.Get("window")

	parseDocument := parseObjectCtor.New()
	parseDocument.Set("baseURI", parseHref)

	parseLocation := parseObjectCtor.New()
	parseLocation.Set("href", parseHref)
	parseWindow := parseObjectCtor.New()
	parseWindow.Set("location", parseLocation)

	parseGlobal.Set("document", parseDocument)
	parseGlobal.Set("window", parseWindow)
	parseT.Cleanup(func() {
		parseGlobal.Set("document", parsePrevDocument)
		parseGlobal.Set("window", parsePrevWindow)
	})
}

func TestDecodeCatalogJSONValidation(parseT *testing.T) {
	parseValid := testCatalog()
	parseDecoded, parseErr := decodeCatalogJSON([]byte(mustMarshalCatalog(parseT, parseValid)))
	if parseErr != nil {
		parseT.Fatalf("decodeCatalogJSON returned error: %v", parseErr)
	}
	if len(parseDecoded.Items) != len(parseValid.Items) || parseDecoded.Items[0].Title != parseValid.Items[0].Title {
		parseT.Fatalf("decoded catalog did not preserve items: %+v", parseDecoded)
	}

	parseTests := []struct {
		name    string
		mutate  func(*docsCatalog)
		message string
	}{
		{name: "missing modules", mutate: func(parseCatalog2 *docsCatalog) { parseCatalog2.Modules = nil }, message: "catalog.json must define modules"},
		{name: "missing statuses", mutate: func(parseCatalog3 *docsCatalog) { parseCatalog3.Statuses = nil }, message: "catalog.json must define statuses"},
		{name: "missing levels", mutate: func(parseCatalog4 *docsCatalog) { parseCatalog4.Levels = nil }, message: "catalog.json must define levels"},
		{name: "missing filters", mutate: func(parseCatalog5 *docsCatalog) { parseCatalog5.Filters = nil }, message: "catalog.json must define filters"},
		{name: "missing sort options", mutate: func(parseCatalog6 *docsCatalog) { parseCatalog6.SortOptions = nil }, message: "catalog.json must define sortOptions"},
		{name: "missing items", mutate: func(parseCatalog7 *docsCatalog) { parseCatalog7.Items = nil }, message: "catalog.json must define at least one item"},
		{name: "item type missing from filters", mutate: func(parseCatalog8 *docsCatalog) {
			parseCatalog8.Filters = []string{filterAll, kindConcept, kindExample}
		}, message: "catalog.json item 2 type \"API\" must appear in filters"},
		{name: "item status missing from statuses", mutate: func(parseCatalog9 *docsCatalog) {
			parseCatalog9.Statuses = []string{allFilterValue, statusStable, statusDeprecated}
		}, message: "catalog.json item 2 status \"experimental\" must appear in statuses"},
		{name: "item level missing from levels", mutate: func(parseCatalog10 *docsCatalog) {
			parseCatalog10.Levels = []string{allFilterValue, levelBeginner, levelIntermediate, levelAdvanced}
		}, message: "catalog.json item 2 level \"Core\" must appear in levels"},
		{name: "item module missing from modules", mutate: func(parseCatalog11 *docsCatalog) {
			parseCatalog11.Modules = []string{allFilterValue, moduleCore, moduleState, moduleCommerce, moduleRendering}
		}, message: "catalog.json item 2 module \"data\" must appear in modules"},
	}

	for _, parseTestCase := range parseTests {
		parseT.Run(parseTestCase.name, func(parseT2 *testing.T) {
			parseCatalog := testCatalog()
			parseTestCase.mutate(&parseCatalog)
			_, parseErr2 := decodeCatalogJSON([]byte(mustMarshalCatalog(parseT2, parseCatalog)))
			if parseErr2 == nil || !strings.Contains(parseErr2.Error(), parseTestCase.message) {
				parseT2.Fatalf("expected %q, got %v", parseTestCase.message, parseErr2)
			}
		})
	}

	if _, parseErr3 := decodeCatalogJSON([]byte("{")); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "invalid catalog.json") {
		parseT.Fatalf("expected invalid catalog json error, got %v", parseErr3)
	}
}

func TestDecodeFetchedCatalogAndHelpers(parseT *testing.T) {
	parseDecoded, parseErr := decodeFetchedCatalog(mustMarshalCatalog(parseT, testCatalog()))
	if parseErr != nil {
		parseT.Fatalf("decodeFetchedCatalog returned error: %v", parseErr)
	}
	if len(parseDecoded.Items) != 5 {
		parseT.Fatalf("expected five decoded items, got %d", len(parseDecoded.Items))
	}
	if parseGot := countItemsByType(parseDecoded.Items, kindAPI); parseGot != 2 {
		parseT.Fatalf("expected two api items from catalog, got %d", parseGot)
	}
	if _, parseErr2 := decodeFetchedCatalog([]byte("nope")); parseErr2 == nil || parseErr2.Error() != "catalog response must be text" {
		parseT.Fatalf("expected text payload error, got %v", parseErr2)
	}
	if parseGot2 := catalogCacheKey("/docs/catalog.json"); parseGot2 != catalogCacheKeyPrefix+"/docs/catalog.json" {
		parseT.Fatalf("unexpected cache key: %q", parseGot2)
	}
	if parseGot3 := normalizeLowercase("  AtLaS  "); parseGot3 != "atlas" {
		parseT.Fatalf("unexpected normalized query: %q", parseGot3)
	}
	if parseGot4 := filteredItemsSignature([]docsItem{testArticleItem(), testAPIItem(), testExampleItem()}); parseGot4 != "1,2,3" {
		parseT.Fatalf("unexpected filtered signature: %q", parseGot4)
	}
	if parseItem, parseOk := findSelectedItem(testCatalog().Items, 3); !parseOk || parseItem.Title != "Go Counter Demo" {
		parseT.Fatalf("expected selected item lookup to succeed, got %+v ok=%v", parseItem, parseOk)
	}
	if _, parseOk2 := findSelectedItem(testCatalog().Items, 999); parseOk2 {
		parseT.Fatal("expected missing selected item lookup to fail")
	}
	if parseGot5 := countItemsByType(testCatalog().Items, kindExample); parseGot5 != 2 {
		parseT.Fatalf("expected two examples, got %d", parseGot5)
	}
	if !isGroupedAPIItem(testGroupedAPIItem()) {
		parseT.Fatal("expected grouped api item helper to be recognized")
	}
	if parseGot6 := apiReferenceDocumentURL(testGroupedAPIItem()); parseGot6 != "assets/docs/public-api-reference.html#core-rendering" && !strings.Contains(parseGot6, "assets/docs/public-api-reference.html#core-rendering") {
		parseT.Fatalf("unexpected grouped api document url: %q", parseGot6)
	}
	if parseGot7 := getContentKindLabel(testArticleItem()); parseGot7 != contentKindLabelArticle {
		parseT.Fatalf("unexpected content kind label: %q", parseGot7)
	}
	if parseGot8 := getContentKindLabel(testAPIItem()); parseGot8 != contentKindLabelAPI {
		parseT.Fatalf("unexpected API kind label: %q", parseGot8)
	}
	if parseGot9 := getContentKindLabel(testExampleItem()); parseGot9 != contentKindLabelDemo {
		parseT.Fatalf("unexpected example kind label: %q", parseGot9)
	}
	parseUnknown := testExampleItem()
	parseUnknown.Content.Kind = "mystery"
	if parseGot10 := getContentKindLabel(parseUnknown); parseGot10 != contentKindLabelUnknown {
		parseT.Fatalf("unexpected unknown kind label: %q", parseGot10)
	}
	if parseGot11 := statusBadgeClass(statusStable); !strings.Contains(parseGot11, "emerald") {
		parseT.Fatalf("unexpected stable badge class: %q", parseGot11)
	}
	if parseGot12 := statusBadgeClass(statusExperimental); !strings.Contains(parseGot12, "amber") {
		parseT.Fatalf("unexpected experimental badge class: %q", parseGot12)
	}
	if parseGot13 := statusBadgeClass(statusDeprecated); !strings.Contains(parseGot13, "rose") {
		parseT.Fatalf("unexpected deprecated badge class: %q", parseGot13)
	}
	if parseGot14 := typeBadgeClass(kindConcept); !strings.Contains(parseGot14, "cyan") {
		parseT.Fatalf("unexpected concept badge class: %q", parseGot14)
	}
	if parseGot15 := typeBadgeClass(kindAPI); !strings.Contains(parseGot15, "violet") {
		parseT.Fatalf("unexpected API badge class: %q", parseGot15)
	}
	if parseGot16 := typeBadgeClass(kindExample); !strings.Contains(parseGot16, "emerald") {
		parseT.Fatalf("unexpected example badge class: %q", parseGot16)
	}
}

func TestLoadMarkdownResourceAndRenderDocument(parseT *testing.T) {
	installMockDocumentBaseURI(parseT, "https://example.test/examples/public-examples-site/index.html")
	installMockFetchText(parseT, testArticleMarkdown(), 200, "OK")
	parseMarkdownURL := docsSourceURL("assets/docs/start-here.md")
	parseLoaded, parseErr := loadMarkdownResource(context.Background(), parseMarkdownURL)
	if parseErr != nil {
		parseT.Fatalf("loadMarkdownResource returned error: %v", parseErr)
	}
	if parseLoaded != testArticleMarkdown() {
		parseT.Fatalf("unexpected markdown payload: %q", parseLoaded)
	}

	parseMarkup, renderErr := ui.RenderToString(Div(Class("space-y-4"), gwchtml.RenderMarkdown(testArticleMarkdown(), markdownRenderOptions("assets/docs/start-here.md"))))
	if renderErr != nil {
		parseT.Fatalf("RenderMarkdown returned error: %v", renderErr)
	}
	for _, parseSnippet := range []string{"<h1", "Start Here", "<h2", "Checklist", "<li", "First", "func main() {}", "assets/docs/troubleshooting.md"} {
		if !strings.Contains(parseMarkup, parseSnippet) {
			parseT.Fatalf("expected markdown markup to contain %q, got %s", parseSnippet, parseMarkup)
		}
	}
}

func TestFilterAndSortItems(parseT *testing.T) {
	parseItems := append(testCatalog().Items, testUnknownLevelItem())

	if parseGot := filterItems(parseItems, "atlas", filterAll, allFilterValue, allFilterValue, allFilterValue); len(parseGot) != 1 || parseGot[0].Title != "Atlas Commerce OS" {
		parseT.Fatalf("expected atlas query to return one item, got %+v", parseGot)
	}
	if parseGot2 := filterItems(parseItems, "", kindExample, statusDeprecated, levelAdvanced, moduleCommerce); len(parseGot2) != 1 || parseGot2[0].Title != "Atlas Commerce OS" {
		parseT.Fatalf("expected combined filters to return atlas item, got %+v", parseGot2)
	}
	if parseGot3 := filterItems(parseItems, "state", kindExample, statusStable, allFilterValue, moduleState); len(parseGot3) != 1 || parseGot3[0].Title != "Go Counter Demo" {
		parseT.Fatalf("expected state example filter to return counter demo, got %+v", parseGot3)
	}
	if parseGot4 := filterItems(parseItems, "renderinto", kindAPI, allFilterValue, allFilterValue, allFilterValue); len(parseGot4) != 1 || parseGot4[0].Title != "Core Rendering Primitives" {
		parseT.Fatalf("expected hidden search tags to match grouped api item, got %+v", parseGot4)
	}
	if parseGot5 := filterItems(parseItems, "missing", filterAll, allFilterValue, allFilterValue, allFilterValue); len(parseGot5) != 0 {
		parseT.Fatalf("expected no query matches, got %+v", parseGot5)
	}

	parseAlphaSorted := sortItems(parseItems, sortAlpha)
	if parseAlphaSorted[0].Title != "Atlas Commerce OS" || parseAlphaSorted[len(parseAlphaSorted)-1].Title != "Zeta Unknown" {
		parseT.Fatalf("unexpected alpha sort order: %+v", parseAlphaSorted)
	}

	parseLevelSorted := sortItems(parseItems, sortLevel)
	if parseLevelSorted[0].Level != levelBeginner || parseLevelSorted[1].Level != levelCore || parseLevelSorted[2].Level != levelIntermediate || parseLevelSorted[3].Level != levelAdvanced || parseLevelSorted[4].Level != "Unknown" {
		parseT.Fatalf("unexpected level sort order: %+v", parseLevelSorted)
	}

	parseRelevanceSorted := sortItems(parseItems, sortRelevance)
	if parseRelevanceSorted[0].ID != parseItems[0].ID || parseRelevanceSorted[len(parseRelevanceSorted)-1].ID != parseItems[len(parseItems)-1].ID {
		parseT.Fatalf("expected relevance sort to preserve input order, got %+v", parseRelevanceSorted)
	}
}

func TestOptionRenderingAndDirectSliceExpansion(parseT *testing.T) {
	parseOptions := renderOptionNodes([]string{"one", "two"})
	parseSortOptions := renderSortOptionNodes([]sortOption{{Value: sortAlpha, Label: labelAlpha}})
	parseMarkup, parseErr := ui.RenderToString(Div(Class("stack"), append(parseOptions, parseSortOptions...)))
	if parseErr != nil {
		parseT.Fatalf("direct slice expansion render failed: %v", parseErr)
	}
	for _, parseSnippet := range []string{"<div class=\"stack\"", "<option value=\"one\">one</option>", "<option value=\"two\">two</option>", labelAlpha} {
		if !strings.Contains(parseMarkup, parseSnippet) {
			parseT.Fatalf("expected markup to contain %q, got %s", parseSnippet, parseMarkup)
		}
	}
}

func TestStaticRenderHelpersProduceExpectedMarkup(parseT *testing.T) {
	parseParameterMarkup, parseErr := ui.RenderToString(renderParameterTable(testAPIItem().Content.Params))
	if parseErr != nil {
		parseT.Fatalf("renderParameterTable returned error: %v", parseErr)
	}
	if !strings.Contains(parseParameterMarkup, "Stable cache key for the resource entry.") || !strings.Contains(parseParameterMarkup, "func(context.Context) (T, error)") {
		parseT.Fatalf("parameter table markup missing expected content: %s", parseParameterMarkup)
	}

	parseFetchStateMarkup, parseErr := ui.RenderToString(renderCatalogFetchState("Loading examples", "Requesting the runnable example catalog before the gallery renders.", "Retry request", ui.Handler{}))
	if parseErr != nil {
		parseT.Fatalf("renderCatalogFetchState returned error: %v", parseErr)
	}
	for _, parseSnippet := range []string{"Loading examples", "Retry request", "Fetching examples", "GoWebComponents example gallery"} {
		if !strings.Contains(parseFetchStateMarkup, parseSnippet) {
			parseT.Fatalf("fetch state markup missing %q: %s", parseSnippet, parseFetchStateMarkup)
		}
	}

	parseHeroMarkup, parseErr := ui.RenderToString(ui.CreateElement(renderCatalogHero, catalogHeroProps{
		OnBrowseExamples:  ui.Handler{},
		OnInspectAPIs:     ui.Handler{},
		TotalItems:        4,
		ExampleCount:      4,
		APICount:          2,
		StableCount:       2,
		ExperimentalCount: 2,
		ModuleCount:       3,
	}))
	if parseErr != nil {
		parseT.Fatalf("renderCatalogHero returned error: %v", parseErr)
	}
	for _, parseSnippet2 := range []string{"Runnable Go + WASM examples with the concepts up front.", buttonBrowseExamples, buttonInspectPackageAPIs, labelCatalogEntries, "Experimental builds"} {
		if !strings.Contains(parseHeroMarkup, parseSnippet2) {
			parseT.Fatalf("hero markup missing %q: %s", parseSnippet2, parseHeroMarkup)
		}
	}

	parseSidebarMarkup, parseErr := ui.RenderToString(ui.CreateElement(renderCatalogSidebar, catalogSidebarProps{
		SearchQuery:          "atlas",
		ResultCount:          1,
		HasActiveFilters:     true,
		Statuses:             testCatalog().Statuses,
		Levels:               testCatalog().Levels,
		Modules:              testCatalog().Modules,
		SortOptions:          testCatalog().SortOptions,
		FilterButtons:        []ui.Node{renderItemCard(testDeprecatedExampleItem(), true, ui.Handler{})},
		SelectedStatusFilter: statusDeprecated,
		SelectedLevelFilter:  levelAdvanced,
		SelectedModuleFilter: moduleCommerce,
		SelectedSortOrder:    sortLevel,
		ItemNodes:            []ui.Node{renderItemCard(testDeprecatedExampleItem(), true, ui.Handler{})},
		OnSearchInput:        ui.Handler{},
		OnStatusChange:       ui.Handler{},
		OnLevelChange:        ui.Handler{},
		OnModuleChange:       ui.Handler{},
		OnSortChange:         ui.Handler{},
		OnResetFilters:       ui.Handler{},
	}))
	if parseErr != nil {
		parseT.Fatalf("renderCatalogSidebar returned error: %v", parseErr)
	}
	for _, parseSnippet3 := range []string{"1 results", "Status", "Difficulty", "Module", "Sort", "Atlas Commerce OS", buttonResetFilters} {
		if !strings.Contains(parseSidebarMarkup, parseSnippet3) {
			parseT.Fatalf("sidebar markup missing %q: %s", parseSnippet3, parseSidebarMarkup)
		}
	}

	parseDefaultSidebarMarkup, parseErr := ui.RenderToString(ui.CreateElement(renderCatalogSidebar, catalogSidebarProps{
		SearchQuery:          "",
		ResultCount:          5,
		HasActiveFilters:     false,
		Statuses:             testCatalog().Statuses,
		Levels:               testCatalog().Levels,
		Modules:              testCatalog().Modules,
		SortOptions:          testCatalog().SortOptions,
		FilterButtons:        []ui.Node{renderItemCard(testGroupedAPIItem(), false, ui.Handler{})},
		SelectedStatusFilter: allFilterValue,
		SelectedLevelFilter:  allFilterValue,
		SelectedModuleFilter: allFilterValue,
		SelectedSortOrder:    sortRelevance,
		ItemNodes:            []ui.Node{renderItemCard(testGroupedAPIItem(), false, ui.Handler{})},
		OnSearchInput:        ui.Handler{},
		OnStatusChange:       ui.Handler{},
		OnLevelChange:        ui.Handler{},
		OnModuleChange:       ui.Handler{},
		OnSortChange:         ui.Handler{},
		OnResetFilters:       ui.Handler{},
	}))
	if parseErr != nil {
		parseT.Fatalf("renderCatalogSidebar default returned error: %v", parseErr)
	}
	for _, parseSnippet4 := range []string{buttonResetFilters, "disabled"} {
		if !strings.Contains(parseDefaultSidebarMarkup, parseSnippet4) {
			parseT.Fatalf("default sidebar markup missing %q: %s", parseSnippet4, parseDefaultSidebarMarkup)
		}
	}

	parseGroupedCardMarkup, parseErr := ui.RenderToString(renderItemCard(testGroupedAPIItem(), true, ui.Handler{}))
	if parseErr != nil {
		parseT.Fatalf("renderItemCard grouped api returned error: %v", parseErr)
	}
	for _, parseSnippet5 := range []string{"<button", "Core Rendering Primitives", kindAPI} {
		if !strings.Contains(parseGroupedCardMarkup, parseSnippet5) {
			parseT.Fatalf("grouped api card missing %q: %s", parseSnippet5, parseGroupedCardMarkup)
		}
	}
	if strings.Contains(parseGroupedCardMarkup, "href=\"#core-rendering\"") {
		parseT.Fatalf("grouped api card should not render a hash href: %s", parseGroupedCardMarkup)
	}
}

func TestRenderDetailPanelAndDisplaySurfaceStates(parseT *testing.T) {
	parseEmptyMarkup, parseErr := ui.RenderToString(ui.CreateElement(renderDetailPanel, detailPanelProps{}))
	if parseErr != nil {
		parseT.Fatalf("renderDetailPanel empty returned error: %v", parseErr)
	}
	for _, parseSnippet := range []string{labelNothingSelected, messageAdjustFilters, `id="demo"`} {
		if !strings.Contains(parseEmptyMarkup, parseSnippet) {
			parseT.Fatalf("empty detail panel missing %q: %s", parseSnippet, parseEmptyMarkup)
		}
	}

	parseSelectedMarkup, parseErr := ui.RenderToString(ui.CreateElement(renderDetailPanel, detailPanelProps{SelectedItem: testAPIItem(), HasSelectedItem: true}))
	if parseErr != nil {
		parseT.Fatalf("renderDetailPanel selected returned error: %v", parseErr)
	}
	for _, parseSnippet2 := range []string{"UseCachedResource", "Typed cached resources with stale-while-revalidate semantics.", contentKindLabelAPI} {
		if !strings.Contains(parseSelectedMarkup, parseSnippet2) {
			parseT.Fatalf("selected detail panel missing %q: %s", parseSnippet2, parseSelectedMarkup)
		}
	}

	parseArticleMarkup, parseErr := ui.RenderToString(renderDisplaySurface(contentPanelProps{Item: testArticleItem(), MarkdownBody: testArticleMarkdown(), MarkdownReady: true}, true))
	if parseErr != nil {
		parseT.Fatalf("renderDisplaySurface article returned error: %v", parseErr)
	}
	if !strings.Contains(parseArticleMarkup, labelConceptArticle) || !strings.Contains(parseArticleMarkup, labelRenderedMarkdown) || !strings.Contains(parseArticleMarkup, "assets/docs/start-here.md") || !strings.Contains(parseArticleMarkup, "Start Here") {
		parseT.Fatalf("article surface missing expected content: %s", parseArticleMarkup)
	}

	parseApiMarkup, parseErr := ui.RenderToString(renderDisplaySurface(contentPanelProps{Item: testAPIItem()}, true))
	if parseErr != nil {
		parseT.Fatalf("renderDisplaySurface api returned error: %v", parseErr)
	}
	if !strings.Contains(parseApiMarkup, labelAPIReference) || !strings.Contains(parseApiMarkup, labelParameters) || !strings.Contains(parseApiMarkup, labelReturns) {
		parseT.Fatalf("api surface missing expected content: %s", parseApiMarkup)
	}
	if !strings.Contains(parseApiMarkup, "users := fetch.UseCachedResource") {
		parseT.Fatalf("api surface should render plain-text usage example by default: %s", parseApiMarkup)
	}

	parseHtmlAPIMarkup, parseErr := ui.RenderToString(renderDisplaySurface(contentPanelProps{Item: testHTMLAPIItem(), MarkdownBody: `<section id="core-rendering"><h2>Core Rendering Primitives</h2></section>`, MarkdownReady: true}, true))
	if parseErr != nil {
		parseT.Fatalf("renderDisplaySurface html api returned error: %v", parseErr)
	}
	for _, parseSnippet3 := range []string{"api-usage-example-fragment", labelUsageExample} {
		if !strings.Contains(parseHtmlAPIMarkup, parseSnippet3) {
			parseT.Fatalf("html api usage example missing %q: %s", parseSnippet3, parseHtmlAPIMarkup)
		}
	}

	parseGroupedAPIMarkup, parseErr := ui.RenderToString(renderDisplaySurface(contentPanelProps{Item: testGroupedAPIItem()}, true))
	if parseErr != nil {
		parseT.Fatalf("renderDisplaySurface grouped api returned error: %v", parseErr)
	}
	for _, parseSnippet4 := range []string{"api-reference-fragment", "Catalog-owned grouped API reference document", "fetches the fragment into this surface", labelReferenceSearch, messageReferenceSearch} {
		if !strings.Contains(parseGroupedAPIMarkup, parseSnippet4) {
			parseT.Fatalf("grouped api surface missing %q: %s", parseSnippet4, parseGroupedAPIMarkup)
		}
	}

	parseInjectedGroupedAPIMarkup, parseErr := ui.RenderToString(renderDisplaySurface(contentPanelProps{Item: testGroupedAPIItem(), MarkdownBody: `<section id="core-rendering"><h2>Core Rendering Primitives</h2></section>`}, true))
	if parseErr != nil {
		parseT.Fatalf("renderDisplaySurface grouped api with html returned error: %v", parseErr)
	}
	for _, parseSnippet5 := range []string{"api-reference-fragment", labelAPIReference} {
		if !strings.Contains(parseInjectedGroupedAPIMarkup, parseSnippet5) {
			parseT.Fatalf("grouped api injected markup missing %q: %s", parseSnippet5, parseInjectedGroupedAPIMarkup)
		}
	}

	parseEmbeddedExampleMarkup, parseErr := ui.RenderToString(renderDisplaySurface(contentPanelProps{
		Item:          testEmbeddedExampleItem(),
		MarkdownBody:  "func Counter() ui.Node {\n  return Button()\n}",
		MarkdownReady: true,
	}, true))
	if parseErr != nil {
		parseT.Fatalf("renderDisplaySurface embedded example returned error: %v", parseErr)
	}
	for _, parseSnippet6 := range []string{labelExampleSource, "Go + hooks + typed HTML", "func Counter() ui.Node"} {
		if !strings.Contains(parseEmbeddedExampleMarkup, parseSnippet6) {
			parseT.Fatalf("embedded example surface missing %q: %s", parseSnippet6, parseEmbeddedExampleMarkup)
		}
	}
	for _, parseSnippet7 := range []string{"iframe", "assets/examples/counter/index.html", "Open standalone"} {
		if !strings.Contains(parseEmbeddedExampleMarkup, parseSnippet7) {
			parseT.Fatalf("embedded example preview missing %q: %s", parseSnippet7, parseEmbeddedExampleMarkup)
		}
	}

	parseSourceFirstMarkup, parseErr := ui.RenderToString(renderDisplaySurface(contentPanelProps{
		Item:          testSourceFirstExampleItem(),
		MarkdownBody:  "package main\n\nfunc main() {\n    // router setup\n}\n",
		MarkdownReady: true,
	}, true))
	if parseErr != nil {
		parseT.Fatalf("renderDisplaySurface source-first example returned error: %v", parseErr)
	}
	for _, parseSnippet := range []string{labelSourceFirstExample, "Browser Router", "router setup"} {
		if !strings.Contains(parseSourceFirstMarkup, parseSnippet) {
			parseT.Fatalf("source-first example surface missing %q: %s", parseSnippet, parseSourceFirstMarkup)
		}
	}

	parsePlaceholderMarkup, parseErr := ui.RenderToString(renderDisplaySurface(contentPanelProps{}, false))
	if parseErr != nil {
		parseT.Fatalf("renderDisplaySurface placeholder returned error: %v", parseErr)
	}
	if !strings.Contains(parsePlaceholderMarkup, messageNothingSelected) {
		parseT.Fatalf("placeholder surface missing empty-state message: %s", parsePlaceholderMarkup)
	}
}

func TestRenderCounterExampleInteractions(parseT *testing.T) {
	parseFixture := render.New(parseT)
	parseFixture.Render(ui.CreateElement(renderCounterExample, contentPanelProps{Item: testExampleItem()}))

	if !strings.Contains(parseFixture.Text(), labelStateTonePrefix+toneReady) {
		parseT.Fatalf("expected initial ready tone, got %q", parseFixture.Text())
	}

	parseIncrement := parseFixture.ByRole("button", buttonIncrement)
	if parseIncrement == nil {
		parseT.Fatal("expected increment button")
	}
	parseIncrement.Click()
	if !strings.Contains(parseFixture.Text(), labelStateTonePrefix+tonePositive) {
		parseT.Fatalf("expected positive tone after increment, got %q", parseFixture.Text())
	}

	parseDecrement := parseFixture.ByRole("button", buttonDecrement)
	if parseDecrement == nil {
		parseT.Fatal("expected decrement button")
	}
	parseDecrement.Click()
	parseDecrement.Click()
	if !strings.Contains(parseFixture.Text(), labelStateTonePrefix+toneNegative) {
		parseT.Fatalf("expected negative tone after decrement, got %q", parseFixture.Text())
	}

	reset := parseFixture.ByRole("button", buttonReset)
	if reset == nil {
		parseT.Fatal("expected reset button")
	}
	reset.Click()
	if !strings.Contains(parseFixture.Text(), labelStateTonePrefix+toneReady) {
		parseT.Fatalf("expected ready tone after reset, got %q", parseFixture.Text())
	}
}

func TestCatalogDataURLResolvesAgainstDocumentBaseURI(parseT *testing.T) {
	installMockDocumentBaseURI(parseT, "https://example.test/examples/public-examples-site/index.html")
	if parseGot := catalogDataURL(); parseGot != "https://example.test/examples/public-examples-site/assets/data/catalog.json" {
		parseT.Fatalf("unexpected catalog data url: %q", parseGot)
	}
}

func TestLoadCatalogResourceFetchesAndValidatesCatalog(parseT *testing.T) {
	parsePayload := mustMarshalCatalog(parseT, testCatalog())
	installMockFetchText(parseT, parsePayload, 200, "OK")

	parseCatalog, parseErr := loadCatalogResource(context.Background(), "https://example.test/assets/data/catalog.json")
	if parseErr != nil {
		parseT.Fatalf("loadCatalogResource returned error: %v", parseErr)
	}
	if len(parseCatalog.Items) != 5 || parseCatalog.Items[0].Title != "Start With GoWebComponents" {
		parseT.Fatalf("unexpected loaded catalog: %+v", parseCatalog)
	}
	if !containsString(parseCatalog.Filters, kindAPI) {
		parseT.Fatalf("expected loaded catalog filters to include %q, got %+v", kindAPI, parseCatalog.Filters)
	}
	if !containsString(parseCatalog.Filters, kindExample) {
		parseT.Fatalf("expected loaded catalog filters to include %q, got %+v", kindExample, parseCatalog.Filters)
	}
	if parseGot := countItemsByType(parseCatalog.Items, kindExample); parseGot != 2 {
		parseT.Fatalf("expected loaded catalog to preserve example count, got %d", parseGot)
	}
	cacheKey := catalogCacheKey("https://example.test/assets/data/catalog.json")
	fetch.DisposeResource(cacheKey)
}

func TestLoadCatalogResourcePropagatesErrors(parseT *testing.T) {
	installMockFetchText(parseT, "{", 200, "OK")
	if _, parseErr := loadCatalogResource(context.Background(), "https://example.test/assets/data/catalog.json"); parseErr == nil || !strings.Contains(parseErr.Error(), "invalid catalog.json") {
		parseT.Fatalf("expected decode error, got %v", parseErr)
	}

	installMockFetchText(parseT, "boom", 500, "Internal Server Error")
	if _, parseErr2 := loadCatalogResource(context.Background(), "https://example.test/assets/data/catalog.json"); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "request failed with status 500") {
		parseT.Fatalf("expected http error, got %v", parseErr2)
	}

	parseCancelled, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	installMockFetchText(parseT, mustMarshalCatalog(parseT, testCatalog()), 200, "OK")
	if _, parseErr3 := loadCatalogResource(parseCancelled, "https://example.test/assets/data/catalog.json"); parseErr3 != context.Canceled {
		parseT.Fatalf("expected context cancellation, got %v", parseErr3)
	}
}

func containsString(parseValues []string, parseExpected string) bool {
	for _, parseValue := range parseValues {
		if parseValue == parseExpected {
			return true
		}
	}
	return false
}
