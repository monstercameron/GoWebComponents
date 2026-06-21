package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type examplesPublicSiteCatalog struct {
	Modules     []string                        `json:"modules"`
	Statuses    []string                        `json:"statuses"`
	Levels      []string                        `json:"levels"`
	Filters     []string                        `json:"filters"`
	SortOptions []examplesPublicSiteSortOption  `json:"sortOptions"`
	Items       []examplesPublicSiteCatalogItem `json:"items"`
}

type examplesPublicSiteSortOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type examplesPublicSiteCatalogItem struct {
	ID         int                              `json:"id"`
	Title      string                           `json:"title"`
	Status     string                           `json:"status"`
	Module     string                           `json:"module"`
	Type       string                           `json:"type"`
	Level      string                           `json:"level"`
	Tags       []string                         `json:"tags"`
	SearchTags []string                         `json:"searchTags,omitempty"`
	Blurb      string                           `json:"blurb"`
	ReadTime   string                           `json:"readTime"`
	Content    examplesPublicSiteCatalogContent `json:"content"`
}

type examplesPublicSiteCatalogContent struct {
	Kind        string                       `json:"kind"`
	AnchorID    string                       `json:"anchorID,omitempty"`
	EmbedPath   string                       `json:"embedPath,omitempty"`
	PreviewPath string                       `json:"previewPath,omitempty"`
	Sections    []examplesPublicSiteSection  `json:"sections,omitempty"`
	Callout     string                       `json:"callout,omitempty"`
	SourcePath  string                       `json:"sourcePath,omitempty"`
	Code        string                       `json:"code,omitempty"`
	Signature   string                       `json:"signature,omitempty"`
	Summary     string                       `json:"summary,omitempty"`
	Params      []examplesPublicSiteDocParam `json:"params,omitempty"`
	Returns     string                       `json:"returns,omitempty"`
	Example     string                       `json:"example,omitempty"`
	Notes       []string                     `json:"notes,omitempty"`
	Description string                       `json:"description,omitempty"`
	Tips        []string                     `json:"tips,omitempty"`
}

type examplesPublicSiteSection struct {
	Heading    string   `json:"heading"`
	Paragraphs []string `json:"paragraphs"`
}

type examplesPublicSiteDocParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    string `json:"required"`
	Description string `json:"description"`
}

var examplesPublicSiteSourceOnlySlugs = map[string]struct{}{
	"progressive-web-app-installability": {},
	"progressive-web-app-multi-client":   {},
	"progressive-web-app-offline-cache":  {},
	"worker-text-index":                  {},
}

// syncExamplesPublicSiteCatalog rebuilds the public examples site catalog entries for every top-level public example.
func syncExamplesPublicSiteCatalog(parseL launcher) error {
	parseCatalogPath := filepath.Join(parseL.examplesDir, "public-examples-site", "assets", "data", "catalog.json")
	parseCatalog, parseErr := loadExamplesPublicSiteCatalog(parseCatalogPath)
	if parseErr != nil {
		return parseErr
	}

	parseRetainedItems := make([]examplesPublicSiteCatalogItem, 0, len(parseCatalog.Items))
	parseNextID := 1
	for _, parseItem := range parseCatalog.Items {
		if parseItem.Type == "Example" {
			continue
		}
		parseRetainedItems = append(parseRetainedItems, parseItem)
		if parseItem.ID >= parseNextID {
			parseNextID = parseItem.ID + 1
		}
	}

	parseExampleItems, parseErr := buildExamplesPublicSiteCatalogItems(filepath.Join(parseL.examplesDir, "public"), parseNextID)
	if parseErr != nil {
		return parseErr
	}
	parseCatalog.Items = append(parseRetainedItems, parseExampleItems...)
	return writeExamplesPublicSiteCatalog(parseCatalogPath, parseCatalog)
}

// loadExamplesPublicSiteCatalog loads the current catalog file or returns a default base catalog when the file does not exist yet.
func loadExamplesPublicSiteCatalog(parseCatalogPath string) (examplesPublicSiteCatalog, error) {
	parsePayload, parseErr := os.ReadFile(parseCatalogPath)
	if parseErr != nil {
		if os.IsNotExist(parseErr) {
			return buildDefaultExamplesPublicSiteCatalog(), nil
		}
		return examplesPublicSiteCatalog{}, fmt.Errorf("read public examples catalog: %w", parseErr)
	}

	var parseCatalog examplesPublicSiteCatalog
	if parseErr := json.Unmarshal(parsePayload, &parseCatalog); parseErr != nil {
		return examplesPublicSiteCatalog{}, fmt.Errorf("decode public examples catalog: %w", parseErr)
	}
	if len(parseCatalog.Modules) == 0 || len(parseCatalog.Statuses) == 0 || len(parseCatalog.Levels) == 0 || len(parseCatalog.Filters) == 0 || len(parseCatalog.SortOptions) == 0 {
		return buildDefaultExamplesPublicSiteCatalog(), nil
	}
	return parseCatalog, nil
}

// buildDefaultExamplesPublicSiteCatalog returns the non-example catalog defaults used when no checked-in catalog exists yet.
func buildDefaultExamplesPublicSiteCatalog() examplesPublicSiteCatalog {
	return examplesPublicSiteCatalog{
		Modules:  []string{"all", "commerce", "core", "data", "ecosystem", "forms", "interop", "platform", "plugins", "rendering", "router", "state"},
		Statuses: []string{"all", "experimental", "stable"},
		Levels:   []string{"all", "Advanced", "Beginner", "Core", "Intermediate"},
		Filters:  []string{"All", "Concept", "API", "Example"},
		SortOptions: []examplesPublicSiteSortOption{
			{Value: "relevance", Label: "Relevance"},
			{Value: "alpha", Label: "A-Z"},
			{Value: "level", Label: "Level"},
		},
		Items: []examplesPublicSiteCatalogItem{},
	}
}

// writeExamplesPublicSiteCatalog writes the generated catalog with stable indentation for review and diffs.
func writeExamplesPublicSiteCatalog(parseCatalogPath string, parseCatalog examplesPublicSiteCatalog) error {
	if parseErr := os.MkdirAll(filepath.Dir(parseCatalogPath), 0755); parseErr != nil {
		return fmt.Errorf("create public examples catalog directory: %w", parseErr)
	}
	parsePayload, parseErr := json.MarshalIndent(parseCatalog, "", "    ")
	if parseErr != nil {
		return fmt.Errorf("encode public examples catalog: %w", parseErr)
	}
	parsePayload = append(parsePayload, '\n')
	if parseErr := os.WriteFile(parseCatalogPath, parsePayload, 0644); parseErr != nil {
		return fmt.Errorf("write public examples catalog: %w", parseErr)
	}
	return nil
}

// buildExamplesPublicSiteCatalogItems derives one catalog entry per top-level public wasm example directory.
func buildExamplesPublicSiteCatalogItems(parsePublicRoot string, parseStartingID int) ([]examplesPublicSiteCatalogItem, error) {
	parseDirectories, parseErr := listExamplesPublicSiteDirectories(parsePublicRoot)
	if parseErr != nil {
		return nil, parseErr
	}

	parseItems := make([]examplesPublicSiteCatalogItem, 0, len(parseDirectories))
	parseNextID := parseStartingID
	for _, parseDirectory := range parseDirectories {
		parseItem, parseErr := buildExamplesPublicSiteCatalogItem(parseDirectory, parseNextID)
		if parseErr != nil {
			return nil, parseErr
		}
		parseItems = append(parseItems, parseItem)
		parseNextID++
	}
	return parseItems, nil
}

// listExamplesPublicSiteDirectories returns every top-level public example directory that exposes a wasm entrypoint.
func listExamplesPublicSiteDirectories(parsePublicRoot string) ([]string, error) {
	parseEntries, parseErr := os.ReadDir(parsePublicRoot)
	if parseErr != nil {
		return nil, fmt.Errorf("read public examples directory: %w", parseErr)
	}

	parseDirectories := make([]string, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		if !parseEntry.IsDir() {
			continue
		}
		parseExampleDir := filepath.Join(parsePublicRoot, parseEntry.Name())
		parseMainPath := filepath.Join(parseExampleDir, "main.go")
		parseInfo, parseErr := os.Stat(parseMainPath)
		if parseErr != nil {
			if os.IsNotExist(parseErr) {
				continue
			}
			return nil, fmt.Errorf("inspect public example entrypoint: %w", parseErr)
		}
		if parseInfo.IsDir() {
			continue
		}
		parseDirectories = append(parseDirectories, parseExampleDir)
	}
	sort.Strings(parseDirectories)
	return parseDirectories, nil
}

// buildExamplesPublicSiteCatalogItem derives one generated catalog entry from a public example directory.
func buildExamplesPublicSiteCatalogItem(parseExampleDir string, parseID int) (examplesPublicSiteCatalogItem, error) {
	parseSlug := filepath.Base(parseExampleDir)
	parseTitle := buildExamplesPublicSiteTitle(parseSlug)
	parseSummary := readExamplesPublicSiteSummary(parseExampleDir)
	isParseEmbeddable := shouldEmbedExamplesPublicSiteItem(parseSlug)

	return examplesPublicSiteCatalogItem{
		ID:         parseID,
		Title:      parseTitle,
		Status:     getExamplesPublicSiteStatus(parseSlug),
		Module:     getExamplesPublicSiteModule(parseSlug),
		Type:       "Example",
		Level:      getExamplesPublicSiteLevel(parseSlug),
		Tags:       buildExamplesPublicSiteTags(parseSlug),
		SearchTags: buildExamplesPublicSiteSearchTags(parseSlug, parseTitle),
		Blurb:      buildExamplesPublicSiteBlurb(parseTitle, parseSummary),
		ReadTime:   "Interactive",
		Content: examplesPublicSiteCatalogContent{
			Kind:        "example",
			EmbedPath:   buildExamplesPublicSiteEmbedPath(parseSlug, isParseEmbeddable),
			PreviewPath: buildExamplesPublicSitePreviewPath(parseSlug, isParseEmbeddable),
			SourcePath:  buildExamplesPublicSiteAssetPath("assets", "code", parseSlug, "main.go"),
			Description: buildExamplesPublicSiteDescription(parseTitle, parseSummary, isParseEmbeddable),
			Tips:        buildExamplesPublicSiteTips(isParseEmbeddable),
		},
	}, nil
}

// readExamplesPublicSiteSummary extracts the first prose paragraph from an example README when one exists.
func readExamplesPublicSiteSummary(parseExampleDir string) string {
	parseReadmePath := filepath.Join(parseExampleDir, "README.md")
	parsePayload, parseErr := os.ReadFile(parseReadmePath)
	if parseErr != nil {
		return ""
	}

	parseLines := strings.Split(string(parsePayload), "\n")
	parseParagraph := make([]string, 0, 4)
	isParseCodeFence := false
	for _, parseLine := range parseLines {
		parseTrimmed := strings.TrimSpace(parseLine)
		if strings.HasPrefix(parseTrimmed, "```") {
			isParseCodeFence = !isParseCodeFence
			continue
		}
		if isParseCodeFence {
			continue
		}
		if parseTrimmed == "" {
			if len(parseParagraph) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(parseTrimmed, "#") || strings.HasPrefix(parseTrimmed, "- ") || strings.HasPrefix(parseTrimmed, "* ") || strings.HasPrefix(parseTrimmed, "Location:") || strings.HasPrefix(parseTrimmed, "**") {
			continue
		}
		parseParagraph = append(parseParagraph, strings.ReplaceAll(strings.ReplaceAll(parseTrimmed, "`", ""), "**", ""))
	}
	return strings.TrimSpace(strings.Join(parseParagraph, " "))
}

// buildExamplesPublicSiteTitle converts one example slug into a readable catalog title.
func buildExamplesPublicSiteTitle(parseSlug string) string {
	parseTokens := strings.Split(strings.TrimSpace(parseSlug), "-")
	parseParts := make([]string, 0, len(parseTokens))
	for _, parseToken := range parseTokens {
		parseToken = strings.TrimSpace(parseToken)
		if parseToken == "" {
			continue
		}
		parseParts = append(parseParts, buildExamplesPublicSiteTokenLabel(parseToken))
	}
	if len(parseParts) == 0 {
		return "Untitled Example"
	}
	return strings.Join(parseParts, " ")
}

// buildExamplesPublicSiteTokenLabel normalizes common acronym tokens before they are title-cased for the catalog.
func buildExamplesPublicSiteTokenLabel(parseToken string) string {
	switch strings.ToLower(strings.TrimSpace(parseToken)) {
	case "api":
		return "API"
	case "css":
		return "CSS"
	case "dom":
		return "DOM"
	case "html":
		return "HTML"
	case "id":
		return "ID"
	case "omi":
		return "OMI"
	case "pwa":
		return "PWA"
	case "ssr":
		return "SSR"
	case "ui":
		return "UI"
	case "wasm":
		return "Wasm"
	default:
		if parseToken == "" {
			return ""
		}
		return strings.ToUpper(parseToken[:1]) + strings.ToLower(parseToken[1:])
	}
}

// buildExamplesPublicSiteBlurb chooses one concise sidebar summary for the generated example catalog card.
func buildExamplesPublicSiteBlurb(parseTitle, parseSummary string) string {
	if strings.TrimSpace(parseSummary) != "" {
		return strings.TrimSpace(parseSummary)
	}
	return fmt.Sprintf("Interactive example for %s with mirrored Go source and a generated wasm build.", parseTitle)
}

// buildExamplesPublicSiteDescription chooses the longer example-panel summary shown alongside the source or live demo.
func buildExamplesPublicSiteDescription(parseTitle, parseSummary string, isParseEmbeddable bool) string {
	if isParseEmbeddable {
		if strings.TrimSpace(parseSummary) != "" {
			return strings.TrimSpace(parseSummary)
		}
		return fmt.Sprintf("Run the live %s demo and inspect the mirrored Go source side by side.", parseTitle)
	}
	return fmt.Sprintf("%s builds as a standalone wasm example, but it depends on routing, hydration, browser-global state, or extra document scaffolding. The docs site keeps it as a source-first reference instead of forcing a fragile live mount.", parseTitle)
}

// buildExamplesPublicSiteTips returns the teaching bullets shown for source-first examples in the catalog.
func buildExamplesPublicSiteTips(isParseEmbeddable bool) []string {
	if isParseEmbeddable {
		return []string{
			"Use the live panel to connect the rendered output to the mirrored Go source.",
			"Open the example package directly when you want to inspect the same binary outside the docs shell.",
		}
	}
	return []string{
		"This example is kept source-first because it relies on standalone document scaffolding or browser-global behavior.",
		"Use the mirrored source here, then run the standalone example package when you need the full environment.",
	}
}

// buildExamplesPublicSiteTags derives a compact tag set from the example slug.
func buildExamplesPublicSiteTags(parseSlug string) []string {
	parseTokens := strings.Split(strings.TrimSpace(parseSlug), "-")
	parseSeen := map[string]struct{}{}
	parseTags := make([]string, 0, len(parseTokens))
	for _, parseToken := range parseTokens {
		parseToken = strings.ToLower(strings.TrimSpace(parseToken))
		if parseToken == "" {
			continue
		}
		if _, parseOk := parseSeen[parseToken]; parseOk {
			continue
		}
		parseSeen[parseToken] = struct{}{}
		parseTags = append(parseTags, parseToken)
	}
	return parseTags
}

// buildExamplesPublicSiteSearchTags expands the catalog search surface with both title and slug forms.
func buildExamplesPublicSiteSearchTags(parseSlug, parseTitle string) []string {
	parseValues := append(buildExamplesPublicSiteTags(parseSlug), strings.ToLower(parseTitle), strings.ReplaceAll(parseSlug, "-", " "))
	parseSeen := map[string]struct{}{}
	parseTags := make([]string, 0, len(parseValues))
	for _, parseValue := range parseValues {
		parseValue = strings.TrimSpace(parseValue)
		if parseValue == "" {
			continue
		}
		if _, parseOk := parseSeen[parseValue]; parseOk {
			continue
		}
		parseSeen[parseValue] = struct{}{}
		parseTags = append(parseTags, parseValue)
	}
	return parseTags
}

// buildExamplesPublicSiteEmbedPath returns one embedded wasm asset path when the example is safe to mount inside the docs shell.
func buildExamplesPublicSiteEmbedPath(parseSlug string, isParseEmbeddable bool) string {
	if !isParseEmbeddable {
		return ""
	}
	return buildExamplesPublicSiteAssetPath("assets", "bins", parseSlug+".wasm")
}

// buildExamplesPublicSitePreviewPath returns one isolated preview page when the example is safe to mount from the public examples site.
func buildExamplesPublicSitePreviewPath(parseSlug string, isParseEmbeddable bool) string {
	if !isParseEmbeddable {
		return ""
	}
	return buildExamplesPublicSiteAssetPath("assets", "examples", parseSlug, "index.html")
}

// buildExamplesPublicSiteAssetPath joins one public-site asset path using slash separators for JSON output.
func buildExamplesPublicSiteAssetPath(parseParts ...string) string {
	return strings.ReplaceAll(filepath.Join(parseParts...), "\\", "/")
}

// getExamplesPublicSiteModule infers the most relevant module filter bucket for one generated example entry.
func getExamplesPublicSiteModule(parseSlug string) string {
	switch {
	case hasExamplesPublicSiteKeyword(parseSlug, "form", "input", "todo"):
		return "forms"
	case hasExamplesPublicSiteKeyword(parseSlug, "route", "router", "navigation", "locale"):
		return "router"
	case hasExamplesPublicSiteKeyword(parseSlug, "fetch", "resource", "cache", "snapshot", "sqlite", "persist", "kvstate"):
		return "data"
	case hasExamplesPublicSiteKeyword(parseSlug, "plugin"):
		return "plugins"
	case hasExamplesPublicSiteKeyword(parseSlug, "multi", "browser", "worker", "interop", "window", "tab"):
		return "interop"
	case hasExamplesPublicSiteKeyword(parseSlug, "render", "portal", "overlay", "html", "fragment", "hydration", "virtualized", "css"):
		return "rendering"
	case strings.HasPrefix(parseSlug, "use-") || hasExamplesPublicSiteKeyword(parseSlug, "counter", "toggle", "calculator", "state", "atom", "reducer", "context", "goroutines"):
		return "state"
	case hasExamplesPublicSiteKeyword(parseSlug, "portfolio", "blog", "omi"):
		return "ecosystem"
	case hasExamplesPublicSiteKeyword(parseSlug, "pwa", "devtools", "reload", "component"):
		return "platform"
	default:
		return "core"
	}
}

// getExamplesPublicSiteLevel infers the learning level label used by the public examples catalog.
func getExamplesPublicSiteLevel(parseSlug string) string {
	switch {
	case strings.HasPrefix(parseSlug, "use-") || hasExamplesPublicSiteKeyword(parseSlug, "counter", "toggle", "text", "calculator", "fragment", "create", "html-tag", "ui-render"):
		return "Beginner"
	case hasExamplesPublicSiteKeyword(parseSlug, "server", "static", "multi", "pwa", "worker", "plugin", "compiler", "auth", "virtualized", "portfolio"):
		return "Advanced"
	case hasExamplesPublicSiteKeyword(parseSlug, "overlay", "portal", "form", "fetch", "snapshot", "devtools", "browser", "error", "async", "hot", "locale", "route", "router", "css", "sqlite", "persist", "kvstate", "raw-html", "global-events"):
		return "Intermediate"
	default:
		return "Core"
	}
}

// getExamplesPublicSiteStatus marks the highest-risk example surfaces as experimental and everything else as stable.
func getExamplesPublicSiteStatus(parseSlug string) string {
	if hasExamplesPublicSiteKeyword(parseSlug, "server", "static", "multi", "pwa", "plugin", "compiler", "worker", "hot", "devtools", "code") {
		return "experimental"
	}
	return "stable"
}

// shouldEmbedExamplesPublicSiteItem reports whether one example is safe to mount inside the docs shell instead of remaining source-first.
func shouldEmbedExamplesPublicSiteItem(parseSlug string) bool {
	_, isParseBlocked := examplesPublicSiteSourceOnlySlugs[strings.TrimSpace(parseSlug)]
	return !isParseBlocked
}

// hasExamplesPublicSiteKeyword reports whether the slug contains any of the supplied keyword fragments.
func hasExamplesPublicSiteKeyword(parseSlug string, parseKeywords ...string) bool {
	parseNormalizedSlug := strings.ToLower(strings.TrimSpace(parseSlug))
	for _, parseKeyword := range parseKeywords {
		parseKeyword = strings.ToLower(strings.TrimSpace(parseKeyword))
		if parseKeyword == "" {
			continue
		}
		if strings.Contains(parseNormalizedSlug, parseKeyword) {
			return true
		}
	}
	return false
}
