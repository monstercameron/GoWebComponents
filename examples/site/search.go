//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"sort"
	"strings"

	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/interop"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// searchEntry is one searchable record built from chapters and the catalog.
type searchEntry struct {
	Title   string
	Href    string
	Kind    string
	Tags    string
	Snippet string
}

// buildSearchEntries flattens chapters and catalog items into search records.
func buildSearchEntries(parseManifest catalogManifest) []searchEntry {
	var parseEntries []searchEntry
	for _, parseChapter := range siteChapters {
		parseEntries = append(parseEntries, searchEntry{
			Title:   parseChapter.Title,
			Href:    "#/learn/" + parseChapter.Slug,
			Kind:    "Manual",
			Snippet: parseChapter.Summary,
		})
	}
	for _, parseItem := range parseManifest.Items {
		parseEntry := searchEntry{
			Title:   parseItem.Title,
			Kind:    parseItem.Type,
			Tags:    strings.ToLower(strings.Join(append(append([]string{}, parseItem.Tags...), parseItem.SearchTags...), " ")),
			Snippet: parseItem.Blurb,
		}
		switch parseItem.Type {
		case "Concept":
			parseEntry.Href = "#/concepts/" + slugify(parseItem.Title)
		case "Example":
			parseEntry.Href = "#/source/" + exampleSlug(parseItem)
		case "API":
			parseEntry.Href = "public-examples-site/" + parseItem.Content.SourcePath
			if parseItem.Content.AnchorID != "" {
				parseEntry.Href += "#" + parseItem.Content.AnchorID
			}
		default:
			continue
		}
		parseEntries = append(parseEntries, parseEntry)
	}
	return parseEntries
}

// scoreSearchEntry ranks one entry against the query terms; zero means skip.
func scoreSearchEntry(parseEntry searchEntry, parseTerms []string) int {
	parseHaystack := strings.ToLower(parseEntry.Title + " " + parseEntry.Tags + " " + parseEntry.Snippet)
	parseTitle := strings.ToLower(parseEntry.Title)
	parseScore := 0
	for _, parseTerm := range parseTerms {
		if !strings.Contains(parseHaystack, parseTerm) {
			return 0
		}
		if strings.Contains(parseTitle, parseTerm) {
			parseScore += 3
			if strings.HasPrefix(parseTitle, parseTerm) {
				parseScore += 2
			}
		} else {
			parseScore++
		}
	}
	return parseScore
}

// renderSearchModal renders the search overlay; entirely Go-driven state.
func renderSearchModal(parseOpen ui.State[bool]) ui.Node {
	parseQuery := ui.UseState("")
	parseCatalog := useCatalog()

	if !parseOpen.Get() {
		return Fragment()
	}

	var parseResults []searchEntry
	parseTerms := strings.Fields(strings.ToLower(parseQuery.Get()))
	if len(parseTerms) > 0 && !parseCatalog.Loading && parseCatalog.Error == nil {
		parseEntries := buildSearchEntries(parseCatalog.Value)
		type scoredEntry struct {
			entry searchEntry
			score int
		}
		var parseScored []scoredEntry
		for _, parseEntry := range parseEntries {
			if parseScore := scoreSearchEntry(parseEntry, parseTerms); parseScore > 0 {
				parseScored = append(parseScored, scoredEntry{entry: parseEntry, score: parseScore})
			}
		}
		sort.SliceStable(parseScored, func(parseA, parseB int) bool {
			return parseScored[parseA].score > parseScored[parseB].score
		})
		if len(parseScored) > 12 {
			parseScored = parseScored[:12]
		}
		for _, parseItem := range parseScored {
			parseResults = append(parseResults, parseItem.entry)
		}
	}

	parseResultNodes := []interface{}{ID("search-results"), ClassStr("search-results")}
	if len(parseResults) == 0 {
		parseEmptyText := "Type to search docs, examples, and APIs."
		if parseQuery.Get() != "" {
			parseEmptyText = "No matches."
		}
		parseResultNodes = append(parseResultNodes, Div(ClassStr("search-empty"), Text(parseEmptyText)))
	}
	for _, parseResult := range parseResults {
		parseHref := parseResult.Href
		parseResultNodes = append(parseResultNodes, A(ClassStr("search-result"), Href(parseHref),
			OnClick(func() { parseOpen.Set(false) }),
			Span(ClassStr("result-kind"), Text(parseResult.Kind)),
			Text(parseResult.Title),
			Span(ClassStr("result-snippet"), Text(parseResult.Snippet)),
		))
	}

	return Div(ID("search-overlay"), ClassStr("search-overlay"), Attr("data-open", "true"),
		OnClick(func() { parseOpen.Set(false) }),
		Div(ClassStr("search-panel"), OnClick(Stop(func() {})),
			Input(ID("search-input"), ClassStr("search-input"),
				Attr("type", "search"), Placeholder("Search docs, examples, APIs..."),
				Attr("autocomplete", "off"), Attr("spellcheck", "false"), AutoFocus(),
				Value(parseQuery.Get()),
				OnInput(func(parseValue string) { parseQuery.Set(parseValue) }),
				OnKeyDown(func(parseEvent ui.KeyboardEvent) {
					if parseEvent.GetKey() == "Escape" {
						parseOpen.Set(false)
					}
				}),
			),
			Div(parseResultNodes...),
		),
	)
}

// renderCopyButton copies text to the clipboard through the Go interop layer.
func renderCopyButton(parseText string) ui.Node {
	parseLabel := ui.UseState("Copy")
	return Button(ClassStr("button-secondary"), Attr("type", "button"), Text(parseLabel.Get()),
		OnClick(func() {
			ui.SafeGo("copy source to clipboard", func() {
				parseClipboard, parseErr := interop.GetClipboard()
				if parseErr != nil {
					parseLabel.Set("Unavailable")
					return
				}
				if parseErr2 := parseClipboard.WriteText(context.Background(), parseText); parseErr2 != nil {
					parseLabel.Set("Failed")
					return
				}
				parseLabel.Set("Copied")
			})
		}),
	)
}
