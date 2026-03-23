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
	item := testAPIItem()
	item.ID = 6
	item.Title = "RenderToString"
	item.Content.SourcePath = "assets/docs/public-api-reference.html"
	item.Content.AnchorID = "core-rendering"
	item.Content.Example = ""
	return item
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
	item := testExampleItem()
	item.ID = 73
	item.Title = "Counter Example"
	item.Content.EmbedPath = "assets/examples/01-counter-host.html"
	item.Content.SourcePath = "assets/code/example-1/counter.go"
	item.Content.Code = "fallback"
	return item
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

func mustMarshalCatalog(t *testing.T, catalog docsCatalog) string {
	t.Helper()
	payload, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("json.Marshal catalog failed: %v", err)
	}
	return string(payload)
}

func installMockFetchText(t *testing.T, payload string, status int, statusText string) {
	t.Helper()
	global := js.Global()
	objectCtor := global.Get("Object")
	promiseCtor := global.Get("Promise")
	prevFetch := global.Get("fetch")

	headers := objectCtor.New()
	headersForEach := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return nil
	})
	headers.Set("forEach", headersForEach)

	response := objectCtor.New()
	response.Set("status", status)
	response.Set("statusText", statusText)
	response.Set("headers", headers)
	textFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return promiseCtor.Call("resolve", payload)
	})
	response.Set("text", textFn)

	fetchFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return promiseCtor.Call("resolve", response)
	})

	global.Set("fetch", fetchFn)
	t.Cleanup(func() {
		global.Set("fetch", prevFetch)
		fetchFn.Release()
		textFn.Release()
		headersForEach.Release()
	})
}

func installMockDocumentBaseURI(t *testing.T, href string) {
	t.Helper()
	global := js.Global()
	objectCtor := global.Get("Object")
	prevDocument := global.Get("document")
	prevWindow := global.Get("window")

	document := objectCtor.New()
	document.Set("baseURI", href)

	location := objectCtor.New()
	location.Set("href", href)
	window := objectCtor.New()
	window.Set("location", location)

	global.Set("document", document)
	global.Set("window", window)
	t.Cleanup(func() {
		global.Set("document", prevDocument)
		global.Set("window", prevWindow)
	})
}

func TestDecodeCatalogJSONValidation(t *testing.T) {
	valid := testCatalog()
	decoded, err := decodeCatalogJSON([]byte(mustMarshalCatalog(t, valid)))
	if err != nil {
		t.Fatalf("decodeCatalogJSON returned error: %v", err)
	}
	if len(decoded.Items) != len(valid.Items) || decoded.Items[0].Title != valid.Items[0].Title {
		t.Fatalf("decoded catalog did not preserve items: %+v", decoded)
	}

	tests := []struct {
		name    string
		mutate  func(*docsCatalog)
		message string
	}{
		{name: "missing modules", mutate: func(catalog *docsCatalog) { catalog.Modules = nil }, message: "catalog.json must define modules"},
		{name: "missing statuses", mutate: func(catalog *docsCatalog) { catalog.Statuses = nil }, message: "catalog.json must define statuses"},
		{name: "missing levels", mutate: func(catalog *docsCatalog) { catalog.Levels = nil }, message: "catalog.json must define levels"},
		{name: "missing filters", mutate: func(catalog *docsCatalog) { catalog.Filters = nil }, message: "catalog.json must define filters"},
		{name: "missing sort options", mutate: func(catalog *docsCatalog) { catalog.SortOptions = nil }, message: "catalog.json must define sortOptions"},
		{name: "missing items", mutate: func(catalog *docsCatalog) { catalog.Items = nil }, message: "catalog.json must define at least one item"},
		{name: "item type missing from filters", mutate: func(catalog *docsCatalog) { catalog.Filters = []string{filterAll, kindConcept, kindExample} }, message: "catalog.json item 2 type \"API\" must appear in filters"},
		{name: "item status missing from statuses", mutate: func(catalog *docsCatalog) {
			catalog.Statuses = []string{allFilterValue, statusStable, statusDeprecated}
		}, message: "catalog.json item 2 status \"experimental\" must appear in statuses"},
		{name: "item level missing from levels", mutate: func(catalog *docsCatalog) {
			catalog.Levels = []string{allFilterValue, levelBeginner, levelIntermediate, levelAdvanced}
		}, message: "catalog.json item 2 level \"Core\" must appear in levels"},
		{name: "item module missing from modules", mutate: func(catalog *docsCatalog) {
			catalog.Modules = []string{allFilterValue, moduleCore, moduleState, moduleCommerce, moduleRendering}
		}, message: "catalog.json item 2 module \"data\" must appear in modules"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			catalog := testCatalog()
			testCase.mutate(&catalog)
			_, err := decodeCatalogJSON([]byte(mustMarshalCatalog(t, catalog)))
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("expected %q, got %v", testCase.message, err)
			}
		})
	}

	if _, err := decodeCatalogJSON([]byte("{")); err == nil || !strings.Contains(err.Error(), "invalid catalog.json") {
		t.Fatalf("expected invalid catalog json error, got %v", err)
	}
}

func TestDecodeFetchedCatalogAndHelpers(t *testing.T) {
	decoded, err := decodeFetchedCatalog(mustMarshalCatalog(t, testCatalog()))
	if err != nil {
		t.Fatalf("decodeFetchedCatalog returned error: %v", err)
	}
	if len(decoded.Items) != 5 {
		t.Fatalf("expected five decoded items, got %d", len(decoded.Items))
	}
	if got := countItemsByType(decoded.Items, kindAPI); got != 2 {
		t.Fatalf("expected two api items from catalog, got %d", got)
	}
	if _, err := decodeFetchedCatalog([]byte("nope")); err == nil || err.Error() != "catalog response must be text" {
		t.Fatalf("expected text payload error, got %v", err)
	}
	if got := catalogCacheKey("/docs/catalog.json"); got != catalogCacheKeyPrefix+"/docs/catalog.json" {
		t.Fatalf("unexpected cache key: %q", got)
	}
	if got := normalizeLowercase("  AtLaS  "); got != "atlas" {
		t.Fatalf("unexpected normalized query: %q", got)
	}
	if got := filteredItemsSignature([]docsItem{testArticleItem(), testAPIItem(), testExampleItem()}); got != "1,2,3" {
		t.Fatalf("unexpected filtered signature: %q", got)
	}
	if item, ok := findSelectedItem(testCatalog().Items, 3); !ok || item.Title != "Go Counter Demo" {
		t.Fatalf("expected selected item lookup to succeed, got %+v ok=%v", item, ok)
	}
	if _, ok := findSelectedItem(testCatalog().Items, 999); ok {
		t.Fatal("expected missing selected item lookup to fail")
	}
	if got := countItemsByType(testCatalog().Items, kindExample); got != 2 {
		t.Fatalf("expected two examples, got %d", got)
	}
	if !isGroupedAPIItem(testGroupedAPIItem()) {
		t.Fatal("expected grouped api item helper to be recognized")
	}
	if got := apiReferenceDocumentURL(testGroupedAPIItem()); got != "assets/docs/public-api-reference.html#core-rendering" && !strings.Contains(got, "assets/docs/public-api-reference.html#core-rendering") {
		t.Fatalf("unexpected grouped api document url: %q", got)
	}
	if got := getContentKindLabel(testArticleItem()); got != contentKindLabelArticle {
		t.Fatalf("unexpected content kind label: %q", got)
	}
	if got := getContentKindLabel(testAPIItem()); got != contentKindLabelAPI {
		t.Fatalf("unexpected API kind label: %q", got)
	}
	if got := getContentKindLabel(testExampleItem()); got != contentKindLabelDemo {
		t.Fatalf("unexpected example kind label: %q", got)
	}
	unknown := testExampleItem()
	unknown.Content.Kind = "mystery"
	if got := getContentKindLabel(unknown); got != contentKindLabelUnknown {
		t.Fatalf("unexpected unknown kind label: %q", got)
	}
	if got := statusBadgeClass(statusStable); !strings.Contains(got, "emerald") {
		t.Fatalf("unexpected stable badge class: %q", got)
	}
	if got := statusBadgeClass(statusExperimental); !strings.Contains(got, "amber") {
		t.Fatalf("unexpected experimental badge class: %q", got)
	}
	if got := statusBadgeClass(statusDeprecated); !strings.Contains(got, "rose") {
		t.Fatalf("unexpected deprecated badge class: %q", got)
	}
	if got := typeBadgeClass(kindConcept); !strings.Contains(got, "cyan") {
		t.Fatalf("unexpected concept badge class: %q", got)
	}
	if got := typeBadgeClass(kindAPI); !strings.Contains(got, "violet") {
		t.Fatalf("unexpected API badge class: %q", got)
	}
	if got := typeBadgeClass(kindExample); !strings.Contains(got, "emerald") {
		t.Fatalf("unexpected example badge class: %q", got)
	}
}

func TestLoadMarkdownResourceAndRenderDocument(t *testing.T) {
	installMockDocumentBaseURI(t, "https://example.test/examples/00-example-0/example-0.html")
	installMockFetchText(t, testArticleMarkdown(), 200, "OK")
	markdownURL := docsSourceURL("assets/docs/start-here.md")
	loaded, err := loadMarkdownResource(context.Background(), markdownURL)
	if err != nil {
		t.Fatalf("loadMarkdownResource returned error: %v", err)
	}
	if loaded != testArticleMarkdown() {
		t.Fatalf("unexpected markdown payload: %q", loaded)
	}

	markup, renderErr := ui.RenderToString(Div(Class("space-y-4"), gwchtml.RenderMarkdown(testArticleMarkdown(), markdownRenderOptions("assets/docs/start-here.md"))))
	if renderErr != nil {
		t.Fatalf("RenderMarkdown returned error: %v", renderErr)
	}
	for _, snippet := range []string{"<h1", "Start Here", "<h2", "Checklist", "<li", "First", "func main() {}", "assets/docs/troubleshooting.md"} {
		if !strings.Contains(markup, snippet) {
			t.Fatalf("expected markdown markup to contain %q, got %s", snippet, markup)
		}
	}
}

func TestFilterAndSortItems(t *testing.T) {
	items := append(testCatalog().Items, testUnknownLevelItem())

	if got := filterItems(items, "atlas", filterAll, allFilterValue, allFilterValue, allFilterValue); len(got) != 1 || got[0].Title != "Atlas Commerce OS" {
		t.Fatalf("expected atlas query to return one item, got %+v", got)
	}
	if got := filterItems(items, "", kindExample, statusDeprecated, levelAdvanced, moduleCommerce); len(got) != 1 || got[0].Title != "Atlas Commerce OS" {
		t.Fatalf("expected combined filters to return atlas item, got %+v", got)
	}
	if got := filterItems(items, "state", kindExample, statusStable, allFilterValue, moduleState); len(got) != 1 || got[0].Title != "Go Counter Demo" {
		t.Fatalf("expected state example filter to return counter demo, got %+v", got)
	}
	if got := filterItems(items, "renderinto", kindAPI, allFilterValue, allFilterValue, allFilterValue); len(got) != 1 || got[0].Title != "Core Rendering Primitives" {
		t.Fatalf("expected hidden search tags to match grouped api item, got %+v", got)
	}
	if got := filterItems(items, "missing", filterAll, allFilterValue, allFilterValue, allFilterValue); len(got) != 0 {
		t.Fatalf("expected no query matches, got %+v", got)
	}

	alphaSorted := sortItems(items, sortAlpha)
	if alphaSorted[0].Title != "Atlas Commerce OS" || alphaSorted[len(alphaSorted)-1].Title != "Zeta Unknown" {
		t.Fatalf("unexpected alpha sort order: %+v", alphaSorted)
	}

	levelSorted := sortItems(items, sortLevel)
	if levelSorted[0].Level != levelBeginner || levelSorted[1].Level != levelCore || levelSorted[2].Level != levelIntermediate || levelSorted[3].Level != levelAdvanced || levelSorted[4].Level != "Unknown" {
		t.Fatalf("unexpected level sort order: %+v", levelSorted)
	}

	relevanceSorted := sortItems(items, sortRelevance)
	if relevanceSorted[0].ID != items[0].ID || relevanceSorted[len(relevanceSorted)-1].ID != items[len(items)-1].ID {
		t.Fatalf("expected relevance sort to preserve input order, got %+v", relevanceSorted)
	}
}

func TestOptionRenderingAndDirectSliceExpansion(t *testing.T) {
	options := renderOptionNodes([]string{"one", "two"})
	sortOptions := renderSortOptionNodes([]sortOption{{Value: sortAlpha, Label: labelAlpha}})
	markup, err := ui.RenderToString(Div(Class("stack"), append(options, sortOptions...)))
	if err != nil {
		t.Fatalf("direct slice expansion render failed: %v", err)
	}
	for _, snippet := range []string{"<div class=\"stack\"", "<option value=\"one\">one</option>", "<option value=\"two\">two</option>", labelAlpha} {
		if !strings.Contains(markup, snippet) {
			t.Fatalf("expected markup to contain %q, got %s", snippet, markup)
		}
	}
}

func TestStaticRenderHelpersProduceExpectedMarkup(t *testing.T) {
	parameterMarkup, err := ui.RenderToString(renderParameterTable(testAPIItem().Content.Params))
	if err != nil {
		t.Fatalf("renderParameterTable returned error: %v", err)
	}
	if !strings.Contains(parameterMarkup, "Stable cache key for the resource entry.") || !strings.Contains(parameterMarkup, "func(context.Context) (T, error)") {
		t.Fatalf("parameter table markup missing expected content: %s", parameterMarkup)
	}

	fetchStateMarkup, err := ui.RenderToString(renderCatalogFetchState("Loading catalog", "Requesting the example catalog JSON before the docs surface renders.", "Retry request", ui.Handler{}))
	if err != nil {
		t.Fatalf("renderCatalogFetchState returned error: %v", err)
	}
	for _, snippet := range []string{"Loading catalog", "Retry request", "Fetching catalog", "Example 0 data pipeline"} {
		if !strings.Contains(fetchStateMarkup, snippet) {
			t.Fatalf("fetch state markup missing %q: %s", snippet, fetchStateMarkup)
		}
	}

	heroMarkup, err := ui.RenderToString(ui.CreateElement(renderCatalogHero, catalogHeroProps{
		OnBrowseExamples: ui.Handler{},
		OnInspectAPIs:    ui.Handler{},
		TotalItems:       4,
		ExampleCount:     2,
		APICount:         1,
	}))
	if err != nil {
		t.Fatalf("renderCatalogHero returned error: %v", err)
	}
	for _, snippet := range []string{"GoWebComponents docs, APIs, and live wasm examples", buttonBrowseExamples, buttonInspectPackageAPIs, labelCatalogEntries, "Go + WASM"} {
		if !strings.Contains(heroMarkup, snippet) {
			t.Fatalf("hero markup missing %q: %s", snippet, heroMarkup)
		}
	}

	sidebarMarkup, err := ui.RenderToString(ui.CreateElement(renderCatalogSidebar, catalogSidebarProps{
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
	if err != nil {
		t.Fatalf("renderCatalogSidebar returned error: %v", err)
	}
	for _, snippet := range []string{"1 results", "Status", "Difficulty", "Module", "Sort", "Atlas Commerce OS", buttonResetFilters} {
		if !strings.Contains(sidebarMarkup, snippet) {
			t.Fatalf("sidebar markup missing %q: %s", snippet, sidebarMarkup)
		}
	}

	defaultSidebarMarkup, err := ui.RenderToString(ui.CreateElement(renderCatalogSidebar, catalogSidebarProps{
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
	if err != nil {
		t.Fatalf("renderCatalogSidebar default returned error: %v", err)
	}
	for _, snippet := range []string{buttonResetFilters, "disabled"} {
		if !strings.Contains(defaultSidebarMarkup, snippet) {
			t.Fatalf("default sidebar markup missing %q: %s", snippet, defaultSidebarMarkup)
		}
	}

	groupedCardMarkup, err := ui.RenderToString(renderItemCard(testGroupedAPIItem(), true, ui.Handler{}))
	if err != nil {
		t.Fatalf("renderItemCard grouped api returned error: %v", err)
	}
	for _, snippet := range []string{"<button", "Core Rendering Primitives", kindAPI} {
		if !strings.Contains(groupedCardMarkup, snippet) {
			t.Fatalf("grouped api card missing %q: %s", snippet, groupedCardMarkup)
		}
	}
	if strings.Contains(groupedCardMarkup, "href=\"#core-rendering\"") {
		t.Fatalf("grouped api card should not render a hash href: %s", groupedCardMarkup)
	}
}

func TestRenderDetailPanelAndDisplaySurfaceStates(t *testing.T) {
	emptyMarkup, err := ui.RenderToString(ui.CreateElement(renderDetailPanel, detailPanelProps{}))
	if err != nil {
		t.Fatalf("renderDetailPanel empty returned error: %v", err)
	}
	for _, snippet := range []string{labelNothingSelected, messageAdjustFilters, `id="demo"`} {
		if !strings.Contains(emptyMarkup, snippet) {
			t.Fatalf("empty detail panel missing %q: %s", snippet, emptyMarkup)
		}
	}

	selectedMarkup, err := ui.RenderToString(ui.CreateElement(renderDetailPanel, detailPanelProps{SelectedItem: testAPIItem(), HasSelectedItem: true}))
	if err != nil {
		t.Fatalf("renderDetailPanel selected returned error: %v", err)
	}
	for _, snippet := range []string{"UseCachedResource", "Typed cached resources with stale-while-revalidate semantics.", contentKindLabelAPI} {
		if !strings.Contains(selectedMarkup, snippet) {
			t.Fatalf("selected detail panel missing %q: %s", snippet, selectedMarkup)
		}
	}

	articleMarkup, err := ui.RenderToString(renderDisplaySurface(contentPanelProps{Item: testArticleItem(), MarkdownBody: testArticleMarkdown(), MarkdownReady: true}, true))
	if err != nil {
		t.Fatalf("renderDisplaySurface article returned error: %v", err)
	}
	if !strings.Contains(articleMarkup, labelConceptArticle) || !strings.Contains(articleMarkup, labelRenderedMarkdown) || !strings.Contains(articleMarkup, "assets/docs/start-here.md") || !strings.Contains(articleMarkup, "Start Here") {
		t.Fatalf("article surface missing expected content: %s", articleMarkup)
	}

	apiMarkup, err := ui.RenderToString(renderDisplaySurface(contentPanelProps{Item: testAPIItem()}, true))
	if err != nil {
		t.Fatalf("renderDisplaySurface api returned error: %v", err)
	}
	if !strings.Contains(apiMarkup, labelAPIReference) || !strings.Contains(apiMarkup, labelParameters) || !strings.Contains(apiMarkup, labelReturns) {
		t.Fatalf("api surface missing expected content: %s", apiMarkup)
	}
	if !strings.Contains(apiMarkup, "users := fetch.UseCachedResource") {
		t.Fatalf("api surface should render plain-text usage example by default: %s", apiMarkup)
	}

	htmlAPIMarkup, err := ui.RenderToString(renderDisplaySurface(contentPanelProps{Item: testHTMLAPIItem(), MarkdownBody: `<section id="core-rendering"><h2>Core Rendering Primitives</h2></section>`, MarkdownReady: true}, true))
	if err != nil {
		t.Fatalf("renderDisplaySurface html api returned error: %v", err)
	}
	for _, snippet := range []string{"api-usage-example-fragment", labelUsageExample} {
		if !strings.Contains(htmlAPIMarkup, snippet) {
			t.Fatalf("html api usage example missing %q: %s", snippet, htmlAPIMarkup)
		}
	}

	groupedAPIMarkup, err := ui.RenderToString(renderDisplaySurface(contentPanelProps{Item: testGroupedAPIItem()}, true))
	if err != nil {
		t.Fatalf("renderDisplaySurface grouped api returned error: %v", err)
	}
	for _, snippet := range []string{"api-reference-fragment", "Catalog-owned grouped API reference document", "fetches the fragment into this surface", labelReferenceSearch, messageReferenceSearch} {
		if !strings.Contains(groupedAPIMarkup, snippet) {
			t.Fatalf("grouped api surface missing %q: %s", snippet, groupedAPIMarkup)
		}
	}

	injectedGroupedAPIMarkup, err := ui.RenderToString(renderDisplaySurface(contentPanelProps{Item: testGroupedAPIItem(), MarkdownBody: `<section id="core-rendering"><h2>Core Rendering Primitives</h2></section>`}, true))
	if err != nil {
		t.Fatalf("renderDisplaySurface grouped api with html returned error: %v", err)
	}
	for _, snippet := range []string{"api-reference-fragment", labelAPIReference} {
		if !strings.Contains(injectedGroupedAPIMarkup, snippet) {
			t.Fatalf("grouped api injected markup missing %q: %s", snippet, injectedGroupedAPIMarkup)
		}
	}

	embeddedExampleMarkup, err := ui.RenderToString(renderDisplaySurface(contentPanelProps{
		Item:          testEmbeddedExampleItem(),
		MarkdownBody:  "func Counter() ui.Node {\n  return Button()\n}",
		MarkdownReady: true,
	}, true))
	if err != nil {
		t.Fatalf("renderDisplaySurface embedded example returned error: %v", err)
	}
	for _, snippet := range []string{labelExampleSource, "Go + hooks + typed HTML", "func Counter() ui.Node"} {
		if !strings.Contains(embeddedExampleMarkup, snippet) {
			t.Fatalf("embedded example surface missing %q: %s", snippet, embeddedExampleMarkup)
		}
	}

	placeholderMarkup, err := ui.RenderToString(renderDisplaySurface(contentPanelProps{}, false))
	if err != nil {
		t.Fatalf("renderDisplaySurface placeholder returned error: %v", err)
	}
	if !strings.Contains(placeholderMarkup, messageNothingSelected) {
		t.Fatalf("placeholder surface missing empty-state message: %s", placeholderMarkup)
	}
}

func TestRenderCounterExampleInteractions(t *testing.T) {
	fixture := render.New(t)
	fixture.Render(ui.CreateElement(renderCounterExample, contentPanelProps{Item: testExampleItem()}))

	if !strings.Contains(fixture.Text(), labelStateTonePrefix+toneReady) {
		t.Fatalf("expected initial ready tone, got %q", fixture.Text())
	}

	increment := fixture.ByRole("button", buttonIncrement)
	if increment == nil {
		t.Fatal("expected increment button")
	}
	increment.Click()
	if !strings.Contains(fixture.Text(), labelStateTonePrefix+tonePositive) {
		t.Fatalf("expected positive tone after increment, got %q", fixture.Text())
	}

	decrement := fixture.ByRole("button", buttonDecrement)
	if decrement == nil {
		t.Fatal("expected decrement button")
	}
	decrement.Click()
	decrement.Click()
	if !strings.Contains(fixture.Text(), labelStateTonePrefix+toneNegative) {
		t.Fatalf("expected negative tone after decrement, got %q", fixture.Text())
	}

	reset := fixture.ByRole("button", buttonReset)
	if reset == nil {
		t.Fatal("expected reset button")
	}
	reset.Click()
	if !strings.Contains(fixture.Text(), labelStateTonePrefix+toneReady) {
		t.Fatalf("expected ready tone after reset, got %q", fixture.Text())
	}
}

func TestCatalogDataURLResolvesAgainstDocumentBaseURI(t *testing.T) {
	installMockDocumentBaseURI(t, "https://example.test/examples/00-example-0/example-0.html")
	if got := catalogDataURL(); got != "https://example.test/examples/00-example-0/assets/data/catalog.json" {
		t.Fatalf("unexpected catalog data url: %q", got)
	}
}

func TestLoadCatalogResourceFetchesAndValidatesCatalog(t *testing.T) {
	payload := mustMarshalCatalog(t, testCatalog())
	installMockFetchText(t, payload, 200, "OK")

	catalog, err := loadCatalogResource(context.Background(), "https://example.test/assets/data/catalog.json")
	if err != nil {
		t.Fatalf("loadCatalogResource returned error: %v", err)
	}
	if len(catalog.Items) != 5 || catalog.Items[0].Title != "Start With GoWebComponents" {
		t.Fatalf("unexpected loaded catalog: %+v", catalog)
	}
	if !containsString(catalog.Filters, kindAPI) {
		t.Fatalf("expected loaded catalog filters to include %q, got %+v", kindAPI, catalog.Filters)
	}
	if !containsString(catalog.Filters, kindExample) {
		t.Fatalf("expected loaded catalog filters to include %q, got %+v", kindExample, catalog.Filters)
	}
	if got := countItemsByType(catalog.Items, kindExample); got != 2 {
		t.Fatalf("expected loaded catalog to preserve example count, got %d", got)
	}
	cacheKey := catalogCacheKey("https://example.test/assets/data/catalog.json")
	fetch.DisposeResource(cacheKey)
}

func TestLoadCatalogResourcePropagatesErrors(t *testing.T) {
	installMockFetchText(t, "{", 200, "OK")
	if _, err := loadCatalogResource(context.Background(), "https://example.test/assets/data/catalog.json"); err == nil || !strings.Contains(err.Error(), "invalid catalog.json") {
		t.Fatalf("expected decode error, got %v", err)
	}

	installMockFetchText(t, "boom", 500, "Internal Server Error")
	if _, err := loadCatalogResource(context.Background(), "https://example.test/assets/data/catalog.json"); err == nil || !strings.Contains(err.Error(), "request failed with status 500") {
		t.Fatalf("expected http error, got %v", err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	installMockFetchText(t, mustMarshalCatalog(t, testCatalog()), 200, "OK")
	if _, err := loadCatalogResource(cancelled, "https://example.test/assets/data/catalog.json"); err != context.Canceled {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
