//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	manualdata "github.com/monstercameron/GoWebComponents/v6/docs/REFERENCE_MANUAL"
)

// catalogItem mirrors one entry of the public examples catalog manifest.
type catalogItem struct {
	ID         int            `json:"id"`
	Title      string         `json:"title"`
	Status     string         `json:"status"`
	Module     string         `json:"module"`
	Type       string         `json:"type"`
	Level      string         `json:"level"`
	Tags       []string       `json:"tags"`
	SearchTags []string       `json:"searchTags"`
	Blurb      string         `json:"blurb"`
	ReadTime   string         `json:"readTime"`
	Content    catalogContent `json:"content"`
}

type catalogContent struct {
	Kind        string           `json:"kind"`
	AnchorID    string           `json:"anchorID"`
	Summary     string           `json:"summary"`
	Description string           `json:"description"`
	EmbedPath   string           `json:"embedPath"`
	PreviewPath string           `json:"previewPath"`
	SourcePath  string           `json:"sourcePath"`
	Sections    []catalogSection `json:"sections"`
}

type catalogSection struct {
	Heading    string   `json:"heading"`
	Paragraphs []string `json:"paragraphs"`
}

type catalogManifest struct {
	Items []catalogItem `json:"items"`
}

// chapter is one reference-manual chapter from the embedded manual data.
type chapter struct {
	Slug     string
	Title    string
	Summary  string
	Markdown string
}

var slugCleanupPattern = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a title into a stable URL slug.
func slugify(parseTitle string) string {
	return strings.Trim(slugCleanupPattern.ReplaceAllString(strings.ToLower(parseTitle), "-"), "-")
}

// parseCatalogJSON decodes the catalog manifest body.
func parseCatalogJSON(parseBody string) (catalogManifest, error) {
	var parseManifest catalogManifest
	parseErr := json.Unmarshal([]byte(parseBody), &parseManifest)
	return parseManifest, parseErr
}

// itemsOfType filters catalog items by type label, sorted by title.
func itemsOfType(parseManifest catalogManifest, parseType string) []catalogItem {
	var parseItems []catalogItem
	for _, parseItem := range parseManifest.Items {
		if parseItem.Type == parseType {
			parseItems = append(parseItems, parseItem)
		}
	}
	sort.Slice(parseItems, func(parseA, parseB int) bool {
		return parseItems[parseA].Title < parseItems[parseB].Title
	})
	return parseItems
}

// exampleSlug derives the example directory slug from its mirrored source path.
func exampleSlug(parseItem catalogItem) string {
	parseParts := strings.Split(strings.Trim(parseItem.Content.SourcePath, "/"), "/")
	if len(parseParts) >= 2 {
		return parseParts[len(parseParts)-2]
	}
	return slugify(parseItem.Title)
}

// loadChapters reads the embedded reference-manual chapters in order.
func loadChapters() []chapter {
	parseEntries, parseErr := manualdata.Chapters.ReadDir(".")
	if parseErr != nil {
		return nil
	}
	var parseChapters []chapter
	for _, parseEntry := range parseEntries {
		parseName := parseEntry.Name()
		if !strings.HasSuffix(parseName, ".md") || strings.EqualFold(parseName, "README.md") {
			continue
		}
		parseRaw, parseReadErr := manualdata.Chapters.ReadFile(parseName)
		if parseReadErr != nil {
			continue
		}
		parseMarkdown := string(parseRaw)
		parseChapters = append(parseChapters, chapter{
			Slug:     strings.TrimSuffix(parseName, ".md"),
			Title:    firstMarkdownHeading(parseMarkdown, parseName),
			Summary:  firstMarkdownParagraph(parseMarkdown),
			Markdown: parseMarkdown,
		})
	}
	sort.Slice(parseChapters, func(parseA, parseB int) bool {
		return parseChapters[parseA].Slug < parseChapters[parseB].Slug
	})
	return parseChapters
}

// firstMarkdownHeading returns the first H1 text, falling back to the filename.
func firstMarkdownHeading(parseMarkdown string, parseFallback string) string {
	for _, parseLine := range strings.Split(parseMarkdown, "\n") {
		parseTrimmed := strings.TrimSpace(parseLine)
		if strings.HasPrefix(parseTrimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(parseTrimmed, "# "))
		}
	}
	return strings.TrimSuffix(parseFallback, ".md")
}

// firstMarkdownParagraph returns the first body paragraph for use as a summary.
func firstMarkdownParagraph(parseMarkdown string) string {
	for _, parseBlock := range strings.Split(strings.ReplaceAll(parseMarkdown, "\r\n", "\n"), "\n\n") {
		parseTrimmed := strings.TrimSpace(parseBlock)
		if parseTrimmed == "" || strings.HasPrefix(parseTrimmed, "#") ||
			strings.HasPrefix(parseTrimmed, "```") || strings.HasPrefix(parseTrimmed, "-") ||
			strings.HasPrefix(parseTrimmed, "|") || strings.HasPrefix(parseTrimmed, ">") {
			continue
		}
		parseFlattened := strings.Join(strings.Fields(parseTrimmed), " ")
		if len(parseFlattened) > 180 {
			parseFlattened = parseFlattened[:177] + "..."
		}
		return parseFlattened
	}
	return ""
}

// chapterShortTitle strips numeric prefixes for sidebar display.
func chapterShortTitle(parseChapter chapter) string {
	parseTitle := parseChapter.Title
	if parseCut := strings.Index(parseTitle, ": "); parseCut > 0 && parseCut < 14 {
		parseTitle = parseTitle[parseCut+2:]
	}
	return parseTitle
}

// heroSource is the curated counter snippet shown beside the live demo.
const heroSource = `func Counter() ui.Node {
	parseCount := ui.UseState(0)

	return Div(Class("counter"),
		H1(Textf("Count: %d", parseCount.Get())),
		Button(Text("+"), OnClick(func() {
			parseCount.Set(parseCount.Get() + 1)
		})),
	)
}`
