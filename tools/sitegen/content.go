package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// catalogItem mirrors one entry of the public examples catalog manifest.
type catalogItem struct {
	ID         int             `json:"id"`
	Title      string          `json:"title"`
	Status     string          `json:"status"`
	Module     string          `json:"module"`
	Type       string          `json:"type"`
	Level      string          `json:"level"`
	Tags       []string        `json:"tags"`
	SearchTags []string        `json:"searchTags"`
	Blurb      string          `json:"blurb"`
	ReadTime   string          `json:"readTime"`
	Content    catalogContent  `json:"content"`
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

// chapter is one reference-manual chapter loaded from docs/REFERENCE_MANUAL.
type chapter struct {
	Slug     string
	Title    string
	Summary  string
	Markdown string
}

// searchEntry is one record of the client-side search index.
type searchEntry struct {
	Title   string `json:"title"`
	Href    string `json:"href"`
	Kind    string `json:"kind"`
	Tags    string `json:"tags,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}

var slugCleanupPattern = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a title into a stable URL slug.
func slugify(parseTitle string) string {
	parseSlug := slugCleanupPattern.ReplaceAllString(strings.ToLower(parseTitle), "-")
	return strings.Trim(parseSlug, "-")
}

// loadCatalog reads the generated public-examples catalog manifest.
func loadCatalog(parseRepoRoot string) (catalogManifest, error) {
	parsePath := filepath.Join(parseRepoRoot, "examples", "public-examples-site", "assets", "data", "catalog.json")
	parseRaw, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return catalogManifest{}, fmt.Errorf("sitegen: read catalog manifest: %w", parseErr)
	}
	var parseManifest catalogManifest
	if parseErr2 := json.Unmarshal(parseRaw, &parseManifest); parseErr2 != nil {
		return catalogManifest{}, fmt.Errorf("sitegen: decode catalog manifest: %w", parseErr2)
	}
	return parseManifest, nil
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

// loadChapters reads the numbered reference-manual chapters in order.
func loadChapters(parseRepoRoot string) ([]chapter, error) {
	parseManualDir := filepath.Join(parseRepoRoot, "docs", "REFERENCE_MANUAL")
	parseEntries, parseErr := os.ReadDir(parseManualDir)
	if parseErr != nil {
		return nil, fmt.Errorf("sitegen: read reference manual dir: %w", parseErr)
	}

	var parseChapters []chapter
	for _, parseEntry := range parseEntries {
		parseName := parseEntry.Name()
		if parseEntry.IsDir() || !strings.HasSuffix(parseName, ".md") || strings.EqualFold(parseName, "README.md") {
			continue
		}
		parseRaw, parseReadErr := os.ReadFile(filepath.Join(parseManualDir, parseName))
		if parseReadErr != nil {
			return nil, fmt.Errorf("sitegen: read chapter %s: %w", parseName, parseReadErr)
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
	return parseChapters, nil
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

// loadHeroSource returns the curated counter snippet for the hero pane. The
// real example component is ~95 lines; the hero teaches the shape, and the
// live pane next to it runs the full example.
func loadHeroSource(parseRepoRoot string) string {
	_ = parseRepoRoot
	return heroFallbackSource
}

const heroFallbackSource = `func Counter() ui.Node {
	parseCount := ui.UseState(0)

	return Div(Class("counter"),
		H1(Textf("Count: %d", parseCount.Get())),
		Button(Text("+"), OnClick(func() {
			parseCount.Set(parseCount.Get() + 1)
		})),
	)
}`
