//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"path"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/fetch"
	gwchtml "github.com/monstercameron/GoWebComponents/v6/html"
	. "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

const repoBlobBase = "https://github.com/monstercameron/GoWebComponents/blob/master/"

// useCatalog loads the catalog manifest once and caches it across routes.
func useCatalog() fetch.CachedResourceState[catalogManifest] {
	parseResource := fetch.UseCachedResource("site-catalog", func(parseCtx context.Context) (catalogManifest, error) {
		parseResult := <-fetch.Fetch("public-examples-site/assets/data/catalog.json", fetch.Options{})
		if parseResult.Err != nil {
			return catalogManifest{}, parseResult.Err
		}
		parseBody, _ := parseResult.Data.(string)
		return parseCatalogJSON(parseBody)
	})
	return parseResource.Get()
}

// useDocumentText loads one repo-served text document with caching.
func useDocumentText(parseURL string) fetch.CachedResourceState[string] {
	parseResource := fetch.UseCachedResource("site-doc:"+parseURL, func(parseCtx context.Context) (string, error) {
		parseResult := <-fetch.Fetch(parseURL, fetch.Options{})
		if parseResult.Err != nil {
			return "", parseResult.Err
		}
		parseBody, _ := parseResult.Data.(string)
		return parseBody, nil
	})
	return parseResource.Get()
}

// renderLoading shows the shared loading state.
func renderLoading(parseWhat string) ui.Node {
	return Div(ClassStr("boot-state"), Span(ClassStr("spinner")), Textf("loading %s...", parseWhat))
}

// renderLoadError shows the shared fetch error state.
func renderLoadError(parseWhat string, parseErr error) ui.Node {
	return Div(ClassStr("boot-state"), Textf("failed to load %s: %v", parseWhat, parseErr))
}

// ---------- landing ----------

func renderLandingPage() ui.Node {
	parseCatalog := useCatalog()
	parseExampleCount := 103
	if !parseCatalog.Loading && parseCatalog.Error == nil {
		parseExampleCount = len(itemsOfType(parseCatalog.Value, "Example"))
	}

	return Main(ClassStr("container"),
		Section(ClassStr("hero"),
			H1(Text("Go. In the "), Span(ClassStr("hero-accent"), Text("browser")), Text(".")),
			P(ClassStr("hero-sub"),
				Text("Build React-class web UIs in pure Go — fine-grained reactivity, server-side rendering with hydration, and a type system you already trust. No JavaScript toolchain required.")),
			Div(ClassStr("hero-ctas"),
				A(ClassStr("button-primary"), Href("#/learn/01-getting-started"), Text("Get started")),
				A(ClassStr("button-secondary"), Href("#/examples"), Textf("Browse %d live examples", parseExampleCount)),
			),
			Div(ClassStr("hero-demo"),
				Div(ClassStr("demo-pane"),
					Div(ClassStr("demo-pane-title"), Text("counter/main.go")),
					Pre(Code(nodesToArgs(highlightSource(heroSource, "go"))...)),
				),
				Div(ClassStr("demo-pane"),
					Div(ClassStr("demo-pane-title"), Span(ClassStr("live-dot")), Text(" live — running in wasm")),
					Iframe(Src("public-examples-site/assets/examples/counter/index.html"),
						Attr("title", "Live counter example"), Attr("loading", "lazy")),
				),
			),
			Div(ClassStr("stats-strip"),
				renderStat("6/6", "paint benchmarks won vs React", "browser-measured update scenarios"),
				renderStat("~1.4 MB", "over the wire", "production wasm, brotli compressed"),
				renderStat("304 ms", "edit-to-browser reload", "dev server rebuild + fine-grained hot reload"),
				renderStat("0 allocs", "steady-state reconcile", "keyed list diff hot path"),
			),
		),

		H2(ClassStr("section-title"), Text("Why teams pick it")),
		P(ClassStr("section-sub"), Text("One language for backend and frontend, with the engineering posture of a systems runtime — not a script loader.")),
		Div(ClassStr("feature-grid"),
			renderFeature("Fine-grained reactivity", "Solid-style atoms update exactly the DOM nodes that depend on them. Components re-render only when their own state changes.", "#/learn/06-state-and-reactivity", "State & reactivity"),
			renderFeature("SSR + hydration", "Render on the server, ship real HTML, hydrate in place with identity reuse and per-subtree mismatch recovery.", "#/learn/09-ssr-and-hydration", "SSR & hydration"),
			renderFeature("Crash containment", "A panicking component produces a structured, agent-readable console report — and the page keeps running. Goroutines, callbacks, and renders are all guarded.", "#/learn/12-devtools-testing-and-observability", "Observability"),
			renderFeature("A real dev loop", "File watching, incremental wasm rebuilds, CSS swap without reload, and component-state-preserving hot reload — measured at 304 ms end to end.", "#/learn/02-gwc-workflows", "Workflows"),
			renderFeature("Router, forms, i18n", "Nested routes with loaders and guards, controlled forms with validation and CSRF, locale routing and message catalogs — in the box.", "#/learn/08-routing", "Routing"),
			renderFeature("Testable by design", "A public testkit renders components headlessly, drives hooks, and asserts SSR output. This site is itself a GWC app — every page you're reading is Go.", "#/learn/12-devtools-testing-and-observability", "Testing"),
		),

		H2(ClassStr("section-title"), Text("Learn it in an afternoon")),
		P(ClassStr("section-sub"), Text("A guided sixteen-chapter manual takes you from first render to scaling patterns — compiled into this very wasm app.")),
		Div(ClassStr("hero-ctas"),
			A(ClassStr("button-primary"), Href("#/learn"), Text("Open the manual")),
			A(ClassStr("button-secondary"), Href("#/api"), Text("API reference")),
		),
	)
}

func renderStat(parseValue string, parseLabel string, parseNote string) ui.Node {
	return Div(ClassStr("stat-card"),
		Div(ClassStr("stat-value"), Text(parseValue)),
		Div(ClassStr("stat-label"), Text(parseLabel)),
		Div(ClassStr("stat-note"), Text(parseNote)),
	)
}

func renderFeature(parseTitle string, parseBody string, parseHref string, parseLinkLabel string) ui.Node {
	return Div(ClassStr("feature-card"),
		H3(Text(parseTitle)),
		P(Text(parseBody)),
		A(ClassStr("feature-link"), Href(parseHref), Textf("%s →", parseLinkLabel)),
	)
}

// ---------- learn ----------

func renderLearnIndex() ui.Node {
	parseChapters := siteChapters
	parseCatalog := useCatalog()

	parseChapterCards := []interface{}{ClassStr("card-grid")}
	for parseIndex, parseChapter := range parseChapters {
		parseChapterCards = append(parseChapterCards, Div(ClassStr("catalog-card"),
			H3(A(Href("#/learn/"+parseChapter.Slug), Textf("%02d · %s", parseIndex+1, chapterShortTitle(parseChapter)))),
			P(Text(parseChapter.Summary)),
		))
	}

	parseConceptSection := []ui.Node{}
	if parseCatalog.Loading {
		parseConceptSection = append(parseConceptSection, renderLoading("concept guides"))
	} else if parseCatalog.Error != nil {
		parseConceptSection = append(parseConceptSection, renderLoadError("concept guides", parseCatalog.Error))
	} else {
		parseConceptCards := []interface{}{ClassStr("card-grid")}
		for _, parseConcept := range itemsOfType(parseCatalog.Value, "Concept") {
			parseConceptCards = append(parseConceptCards, Div(ClassStr("catalog-card"),
				H3(A(Href("#/concepts/"+slugify(parseConcept.Title)), Text(parseConcept.Title))),
				P(Text(parseConcept.Blurb)),
				Div(ClassStr("card-meta"),
					Span(ClassStr("chip chip-level"), Text(parseConcept.Level)),
					Span(ClassStr("chip"), Text(parseConcept.Module)),
				),
			))
		}
		parseConceptSection = append(parseConceptSection, Div(parseConceptCards...))
	}

	return Main(ClassStr("container"),
		H1(ClassStr("page-title"), Text("Learn")),
		P(ClassStr("page-sub"), Text("Start with the guided manual — sixteen chapters from first render to large-codebase patterns. Deep-dive concept guides cover individual subsystems.")),
		H2(ClassStr("section-title"), Text("The manual")),
		Div(parseChapterCards...),
		Fragment(nodesToArgs(append([]ui.Node{H2(ClassStr("section-title"), Text("Concept guides")),
			P(ClassStr("section-sub"), Text("Focused documents on one subsystem at a time."))}, parseConceptSection...))...),
	)
}

// markdownOptionsForChapter wires chapter cross-links onto hash routes.
func markdownOptionsForChapter(parseSlug string) gwchtml.MarkdownRenderOptions {
	return gwchtml.MarkdownRenderOptions{
		SourcePath: parseSlug + ".md",
		ResolveHref: func(parseSourcePath, parseDestination string) string {
			parseTrimmed := strings.TrimSpace(parseDestination)
			if strings.HasPrefix(parseTrimmed, "http") || strings.HasPrefix(parseTrimmed, "#") {
				return parseTrimmed
			}
			parseBase, parseFragment, _ := strings.Cut(parseTrimmed, "#")
			if strings.HasSuffix(parseBase, ".md") && !strings.Contains(parseBase, "/") {
				parseResolved := "#/learn/" + strings.TrimSuffix(strings.TrimPrefix(parseBase, "./"), ".md")
				_ = parseFragment
				return parseResolved
			}
			return repoBlobBase + path.Clean(path.Join("docs/REFERENCE_MANUAL", parseBase))
		},
	}
}

func renderChapterPage(parseSlug string) ui.Node {
	parseChapters := siteChapters
	parseIndex := -1
	for parseI, parseChapter := range parseChapters {
		if parseChapter.Slug == parseSlug {
			parseIndex = parseI
			break
		}
	}
	if parseIndex < 0 {
		return renderNotFound()
	}
	parseChapter := parseChapters[parseIndex]
	parseBody := gwchtml.RenderMarkdown(parseChapter.Markdown, markdownOptionsForChapter(parseChapter.Slug))

	var parsePagerPrev ui.Node = Span()
	if parseIndex > 0 {
		parsePrevious := parseChapters[parseIndex-1]
		parsePagerPrev = A(Href("#/learn/"+parsePrevious.Slug),
			Span(ClassStr("pager-label"), Text("Previous")), Text(parsePrevious.Title))
	}
	var parsePagerNext ui.Node = Span()
	if parseIndex < len(parseChapters)-1 {
		parseNext := parseChapters[parseIndex+1]
		parsePagerNext = A(ClassStr("pager-next"), Href("#/learn/"+parseNext.Slug),
			Span(ClassStr("pager-label"), Text("Next")), Text(parseNext.Title))
	}

	return Main(ClassStr("container"),
		Div(ClassStr("learn-layout"),
			renderLearnSidebar(parseChapters, parseChapter.Slug),
			Div(
				Article(append([]interface{}{ClassStr("prose")}, nodesToArgs(parseBody)...)...),
				Div(ClassStr("chapter-pager"), parsePagerPrev, parsePagerNext),
			),
		),
	)
}

func renderLearnSidebar(parseChapters []chapter, parseActiveSlug string) ui.Node {
	parseLinks := []interface{}{ClassStr("learn-sidebar"),
		Div(ClassStr("sidebar-group"), Text("Reference manual"))}
	for _, parseChapter := range parseChapters {
		parseAttrs := []interface{}{Href("#/learn/" + parseChapter.Slug), Text(chapterShortTitle(parseChapter))}
		if parseChapter.Slug == parseActiveSlug {
			parseAttrs = append(parseAttrs, Attr("data-active", "true"))
		}
		parseLinks = append(parseLinks, A(parseAttrs...))
	}
	return Aside(parseLinks...)
}

// ---------- concepts ----------

func renderConceptPage(parseSlug string) ui.Node {
	parseCatalog := useCatalog()
	if parseCatalog.Loading {
		return Main(ClassStr("container"), renderLoading("concept guide"))
	}
	if parseCatalog.Error != nil {
		return Main(ClassStr("container"), renderLoadError("concept guide", parseCatalog.Error))
	}

	var parseConcept catalogItem
	isParseFound := false
	parseConceptSlugsByDoc := map[string]string{}
	for _, parseItem := range itemsOfType(parseCatalog.Value, "Concept") {
		if strings.HasSuffix(parseItem.Content.SourcePath, ".md") {
			parseConceptSlugsByDoc[path.Base(parseItem.Content.SourcePath)] = slugify(parseItem.Title)
		}
		if slugify(parseItem.Title) == parseSlug {
			parseConcept = parseItem
			isParseFound = true
		}
	}
	if !isParseFound {
		return renderNotFound()
	}

	var parseBody []ui.Node
	if strings.HasSuffix(parseConcept.Content.SourcePath, ".md") {
		parseDoc := useDocumentText("public-examples-site/" + parseConcept.Content.SourcePath)
		switch {
		case parseDoc.Loading:
			return Main(ClassStr("container"), renderLoading(parseConcept.Title))
		case parseDoc.Error != nil:
			return Main(ClassStr("container"), renderLoadError(parseConcept.Title, parseDoc.Error))
		default:
			parseBody = gwchtml.RenderMarkdown(parseDoc.Value, gwchtml.MarkdownRenderOptions{
				SourcePath: parseConcept.Content.SourcePath,
				ResolveHref: func(parseSourcePath, parseDestination string) string {
					return resolveConceptHref(parseDestination, parseConceptSlugsByDoc)
				},
			})
		}
	} else {
		parseBody = append(parseBody, H1(Text(parseConcept.Title)))
		for _, parseSection := range parseConcept.Content.Sections {
			if parseSection.Heading != "" {
				parseBody = append(parseBody, H2(Text(parseSection.Heading)))
			}
			for _, parseParagraph := range parseSection.Paragraphs {
				parseBody = append(parseBody, P(Text(parseParagraph)))
			}
		}
	}

	return Main(ClassStr("container"),
		Article(append([]interface{}{ClassStr("prose"), Style(map[string]string{"padding-top": "40px"})}, nodesToArgs(parseBody)...)...),
	)
}

func resolveConceptHref(parseDestination string, parseConceptSlugsByDoc map[string]string) string {
	parseTrimmed := strings.TrimSpace(parseDestination)
	if parseTrimmed == "" || strings.HasPrefix(parseTrimmed, "http") || strings.HasPrefix(parseTrimmed, "#") {
		return parseTrimmed
	}
	parseBase, _, _ := strings.Cut(parseTrimmed, "#")
	if strings.HasSuffix(parseBase, ".md") {
		if parseSlug, isParseKnown := parseConceptSlugsByDoc[path.Base(parseBase)]; isParseKnown {
			return "#/concepts/" + parseSlug
		}
	}
	return repoBlobBase + path.Clean(path.Join("examples/public-examples-site/assets/docs", parseBase))
}

// ---------- examples gallery ----------

func renderExamplesGallery() ui.Node {
	parseCatalog := useCatalog()
	parseFilter := ui.UseState("all")

	if parseCatalog.Loading {
		return Main(ClassStr("container"), renderLoading("example catalog"))
	}
	if parseCatalog.Error != nil {
		return Main(ClassStr("container"), renderLoadError("example catalog", parseCatalog.Error))
	}
	parseExamples := itemsOfType(parseCatalog.Value, "Example")

	parseModuleSet := map[string]bool{}
	for _, parseExample := range parseExamples {
		parseModuleSet[parseExample.Module] = true
	}
	parseActiveFilter := parseFilter.Get()
	parseFilterChip := func(parseLabel string, parseValue string) ui.Node {
		parseAttrs := []interface{}{ClassStr("filter-chip"), Attr("type", "button"), Text(parseLabel),
			OnClick(func() { parseFilter.Set(parseValue) })}
		if parseActiveFilter == parseValue {
			parseAttrs = append(parseAttrs, Attr("data-active", "true"))
		}
		return Button(parseAttrs...)
	}
	parseFilterChips := []interface{}{ClassStr("filter-bar"), parseFilterChip("All", "all")}
	for _, parseModule := range sortedKeys(parseModuleSet) {
		parseFilterChips = append(parseFilterChips, parseFilterChip(parseModule, parseModule))
	}

	parseCards := []interface{}{ClassStr("card-grid")}
	parseVisible := 0
	for _, parseExample := range parseExamples {
		if parseActiveFilter != "all" && parseExample.Module != parseActiveFilter {
			continue
		}
		parseVisible++
		parseActions := []interface{}{ClassStr("card-actions")}
		if parseExample.Content.PreviewPath != "" {
			parseActions = append(parseActions,
				A(Href("public-examples-site/"+parseExample.Content.PreviewPath), Attr("target", "_blank"), Text("Run live →")))
		}
		if parseExample.Content.SourcePath != "" {
			parseActions = append(parseActions,
				A(Href("#/source/"+exampleSlug(parseExample)), Text("Source")))
		}
		parseCards = append(parseCards, Div(ClassStr("catalog-card"),
			H3(Text(parseExample.Title)),
			P(Text(parseExample.Blurb)),
			Div(ClassStr("card-meta"),
				Span(ClassStr("chip chip-level"), Text(parseExample.Level)),
				Span(ClassStr("chip"), Text(parseExample.Module)),
			),
			Div(parseActions...),
		))
	}

	return Main(ClassStr("container"),
		H1(ClassStr("page-title"), Textf("%d live examples", parseVisible)),
		P(ClassStr("page-sub"), Text("Every example runs real wasm in your browser and ships with its mirrored Go source. Filter by subsystem, run anything, copy everything.")),
		Div(parseFilterChips...),
		Div(parseCards...),
	)
}

// ---------- API index ----------

func renderAPIIndex() ui.Node {
	parseCatalog := useCatalog()
	if parseCatalog.Loading {
		return Main(ClassStr("container"), renderLoading("API reference"))
	}
	if parseCatalog.Error != nil {
		return Main(ClassStr("container"), renderLoadError("API reference", parseCatalog.Error))
	}

	parseCards := []interface{}{ClassStr("card-grid")}
	for _, parseItem := range itemsOfType(parseCatalog.Value, "API") {
		parseHref := "public-examples-site/" + parseItem.Content.SourcePath
		if parseItem.Content.AnchorID != "" {
			parseHref += "#" + parseItem.Content.AnchorID
		}
		parseTagLine := strings.Join(parseItem.SearchTags, " · ")
		if len(parseTagLine) > 110 {
			parseTagLine = parseTagLine[:107] + "..."
		}
		parseCards = append(parseCards, Div(ClassStr("catalog-card"),
			H3(A(Href(parseHref), Attr("target", "_blank"), Text(parseItem.Title))),
			P(Text(parseItem.Content.Summary)),
			Div(ClassStr("card-meta"), Span(ClassStr("chip"), Text(parseTagLine))),
		))
	}

	return Main(ClassStr("container"),
		H1(ClassStr("page-title"), Text("API reference")),
		P(ClassStr("page-sub"), Text("The public surface grouped by subsystem. Each group links into the full generated reference with signatures and parameter tables.")),
		Div(parseCards...),
	)
}

// ---------- source viewer ----------

func renderSourceViewer(parseSlug string) ui.Node {
	parseCatalog := useCatalog()
	if parseCatalog.Loading {
		return Main(ClassStr("container"), renderLoading("source"))
	}
	if parseCatalog.Error != nil {
		return Main(ClassStr("container"), renderLoadError("source", parseCatalog.Error))
	}

	var parseExample catalogItem
	isParseFound := false
	for _, parseItem := range itemsOfType(parseCatalog.Value, "Example") {
		if exampleSlug(parseItem) == parseSlug {
			parseExample = parseItem
			isParseFound = true
			break
		}
	}
	if !isParseFound || parseExample.Content.SourcePath == "" {
		return renderNotFound()
	}

	parseSourceURL := "public-examples-site/" + parseExample.Content.SourcePath
	parseSource := useDocumentText(parseSourceURL)
	if parseSource.Loading {
		return Main(ClassStr("container"), renderLoading(parseExample.Title+" source"))
	}
	if parseSource.Error != nil {
		return Main(ClassStr("container"), renderLoadError(parseExample.Title+" source", parseSource.Error))
	}

	parseActions := []interface{}{ClassStr("source-actions"),
		renderCopyButton(parseSource.Value)}
	if parseExample.Content.PreviewPath != "" {
		parseActions = append(parseActions,
			A(Href("public-examples-site/"+parseExample.Content.PreviewPath), Attr("target", "_blank"), Text("Run live →")))
	}
	parseActions = append(parseActions, A(Href("#/examples"), Text("All examples")))

	return Main(ClassStr("container"),
		Div(ClassStr("source-header"),
			H1(Text(parseExample.Title)),
			Span(ClassStr("source-path"), Text(path.Base(parseExample.Content.SourcePath))),
			Div(parseActions...),
		),
		P(ClassStr("page-sub"), Text(parseExample.Content.Description)),
		Div(ClassStr("source-pane"),
			Pre(Code(nodesToArgs(highlightSource(parseSource.Value, "go"))...)),
		),
	)
}

// ---------- shared bits ----------

func renderNotFound() ui.Node {
	return Main(ClassStr("container"),
		Div(ClassStr("boot-state"), Text("404 — nothing routed here. ")),
		Div(ClassStr("hero-ctas"), Style(map[string]string{"justify-content": "center", "padding-bottom": "64px"}),
			A(ClassStr("button-secondary"), Href("#/"), Text("Back home"))),
	)
}

// nodesToArgs adapts a ui.Node slice for variadic shorthand constructors.
func nodesToArgs(parseNodes []ui.Node) []interface{} {
	parseArgs := make([]interface{}, len(parseNodes))
	for parseIndex, parseNode := range parseNodes {
		parseArgs[parseIndex] = parseNode
	}
	return parseArgs
}

// sortedKeys returns map keys in sorted order.
func sortedKeys(parseSet map[string]bool) []string {
	parseKeys := make([]string, 0, len(parseSet))
	for parseKey := range parseSet {
		parseKeys = append(parseKeys, parseKey)
	}
	for parseA := 0; parseA < len(parseKeys); parseA++ {
		for parseB := parseA + 1; parseB < len(parseKeys); parseB++ {
			if parseKeys[parseB] < parseKeys[parseA] {
				parseKeys[parseA], parseKeys[parseB] = parseKeys[parseB], parseKeys[parseA]
			}
		}
	}
	return parseKeys
}
