//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/fetch"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

// catalogDataURL resolves the catalog asset relative to the current document URL.
func catalogDataURL() string {
	return appendPageVersion(utils.ResolveDocumentURL(catalogDataRelativeURL))
}

// docsSourceURL resolves a markdown source asset relative to the current document URL.
func docsSourceURL(parseSourcePath string) string {
	if parseSourcePath == "" {
		return ""
	}
	return appendPageVersion(utils.ResolveDocumentURL(parseSourcePath))
}

func embeddedExampleURL(parseSourcePath string) string {
	if parseSourcePath == "" {
		return ""
	}
	return appendPageVersion(utils.ResolveDocumentURL(parseSourcePath))
}

// previewExampleURL resolves a staged example preview page relative to the current document URL.
func previewExampleURL(parseSourcePath string) string {
	if parseSourcePath == "" {
		return ""
	}
	return appendPageVersion(utils.ResolveDocumentURL(parseSourcePath))
}

func appendPageVersion(parseResolvedURL string) string {
	if strings.TrimSpace(parseResolvedURL) == "" {
		return ""
	}
	parseWindow := js.Global().Get("window")
	if parseWindow.IsUndefined() || parseWindow.IsNull() {
		return parseResolvedURL
	}
	parseSearch := parseWindow.Get("location").Get("search")
	if parseSearch.IsUndefined() || parseSearch.IsNull() {
		return parseResolvedURL
	}
	parsePageParams := js.Global().Get("URLSearchParams").New(parseSearch.String())
	parseRawPageVersion := parsePageParams.Call("get", "v")
	if parseRawPageVersion.IsUndefined() || parseRawPageVersion.IsNull() {
		return parseResolvedURL
	}
	parsePageVersion := strings.TrimSpace(parseRawPageVersion.String())
	if parsePageVersion == "null" || parsePageVersion == "undefined" || parsePageVersion == "<null>" || parsePageVersion == "<undefined>" {
		return parseResolvedURL
	}
	if parsePageVersion == "" {
		return parseResolvedURL
	}
	parseParsedURL, parseErr := url.Parse(parseResolvedURL)
	if parseErr != nil {
		return parseResolvedURL
	}
	parseQuery := parseParsedURL.Query()
	parseQuery.Set("v", parsePageVersion)
	parseParsedURL.RawQuery = parseQuery.Encode()
	return parseParsedURL.String()
}

// getPageQueryValue resolves one query parameter from window.location.search.
func getPageQueryValue(parseName string) string {
	parseTrimmedName := strings.TrimSpace(parseName)
	if parseTrimmedName == "" {
		return ""
	}
	parseWindow := js.Global().Get("window")
	if parseWindow.IsUndefined() || parseWindow.IsNull() {
		return ""
	}
	parseLocation := parseWindow.Get("location")
	if parseLocation.IsUndefined() || parseLocation.IsNull() {
		return ""
	}
	parseSearch := parseLocation.Get("search")
	if parseSearch.IsUndefined() || parseSearch.IsNull() {
		return ""
	}
	parseParams := js.Global().Get("URLSearchParams")
	if parseParams.Type() != js.TypeFunction {
		return ""
	}
	parsePageParams := parseParams.New(parseSearch.String())
	parseValue := parsePageParams.Call("get", parseTrimmedName)
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return ""
	}
	return strings.TrimSpace(parseValue.String())
}

// isPageDebugLoggingEnabled reports whether the gallery should emit verbose browser logging.
func isPageDebugLoggingEnabled() bool {
	switch normalizeLowercase(getPageQueryValue("debug")) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// decodeCatalogJSON validates the fetched JSON payload before the UI reads from it.
func decodeCatalogJSON(parsePayload []byte) (docsCatalog, error) {
	var parseCatalog docsCatalog
	if parseErr := json.Unmarshal(parsePayload, &parseCatalog); parseErr != nil {
		return docsCatalog{}, fmt.Errorf("invalid catalog.json: %w", parseErr)
	}
	if len(parseCatalog.Modules) == 0 {
		return docsCatalog{}, fmt.Errorf("catalog.json must define modules")
	}
	if len(parseCatalog.Statuses) == 0 {
		return docsCatalog{}, fmt.Errorf("catalog.json must define statuses")
	}
	if len(parseCatalog.Levels) == 0 {
		return docsCatalog{}, fmt.Errorf("catalog.json must define levels")
	}
	if len(parseCatalog.Filters) == 0 {
		return docsCatalog{}, fmt.Errorf("catalog.json must define filters")
	}
	if len(parseCatalog.SortOptions) == 0 {
		return docsCatalog{}, fmt.Errorf("catalog.json must define sortOptions")
	}
	if len(parseCatalog.Items) == 0 {
		return docsCatalog{}, fmt.Errorf("catalog.json must define at least one item")
	}
	for _, parseItem := range parseCatalog.Items {
		if !containsCatalogValue(parseCatalog.Filters, parseItem.Type) {
			return docsCatalog{}, fmt.Errorf("catalog.json item %d type %q must appear in filters", parseItem.ID, parseItem.Type)
		}
		if !containsCatalogValue(parseCatalog.Statuses, parseItem.Status) {
			return docsCatalog{}, fmt.Errorf("catalog.json item %d status %q must appear in statuses", parseItem.ID, parseItem.Status)
		}
		if !containsCatalogValue(parseCatalog.Levels, parseItem.Level) {
			return docsCatalog{}, fmt.Errorf("catalog.json item %d level %q must appear in levels", parseItem.ID, parseItem.Level)
		}
		if !containsCatalogValue(parseCatalog.Modules, parseItem.Module) {
			return docsCatalog{}, fmt.Errorf("catalog.json item %d module %q must appear in modules", parseItem.ID, parseItem.Module)
		}
	}
	return parseCatalog, nil
}

func containsCatalogValue(parseValues []string, parseExpected string) bool {
	for _, parseValue := range parseValues {
		if parseValue == parseExpected {
			return true
		}
	}
	return false
}

// decodeFetchedCatalog converts the low-level fetch payload into a validated catalog.
func decodeFetchedCatalog(parsePayload interface{}) (docsCatalog, error) {
	parseTextPayload, parseOk := parsePayload.(string)
	if !parseOk {
		return docsCatalog{}, fmt.Errorf("catalog response must be text")
	}
	return decodeCatalogJSON([]byte(parseTextPayload))
}

// decodeFetchedText validates plain text fetch responses used by markdown documents.
func decodeFetchedText(parsePayload interface{}) (string, error) {
	parseTextPayload, parseOk := parsePayload.(string)
	if !parseOk {
		return "", fmt.Errorf("document response must be text")
	}
	return parseTextPayload, nil
}

// catalogCacheKey returns the cache key used to share catalog data across renders.
func catalogCacheKey(parseUrl string) string {
	return catalogCacheKeyPrefix + parseUrl
}

// markdownCacheKey returns the cache key used to share markdown document loads.
func markdownCacheKey(parseUrl string) string {
	return markdownCacheKeyPrefix + parseUrl
}

// loadCatalogResource fetches and validates the catalog through the cached-resource loader path.
func loadCatalogResource(parseCtx context.Context, parseUrl string) (docsCatalog, error) {
	parseResultCh := fetch.Fetch(parseUrl, fetch.Options{})
	select {
	case <-parseCtx.Done():
		return docsCatalog{}, parseCtx.Err()
	case parseResult := <-parseResultCh:
		if parseResult.Err != nil {
			return docsCatalog{}, parseResult.Err
		}
		return decodeFetchedCatalog(parseResult.Data)
	}
}

// loadMarkdownResource fetches and validates a markdown document as plain text.
func loadMarkdownResource(parseCtx context.Context, parseUrl string) (string, error) {
	if strings.TrimSpace(parseUrl) == "" {
		return "", nil
	}
	parseResultCh := fetch.Fetch(parseUrl, fetch.Options{})
	select {
	case <-parseCtx.Done():
		return "", parseCtx.Err()
	case parseResult := <-parseResultCh:
		if parseResult.Err != nil {
			return "", parseResult.Err
		}
		return decodeFetchedText(parseResult.Data)
	}
}

func errorString(parseErr error) string {
	if parseErr == nil {
		return ""
	}
	return parseErr.Error()
}

// normalizeLowercase trims and lowercases values before they are used in comparisons.
func normalizeLowercase(parseValue string) string {
	return strings.TrimSpace(strings.ToLower(parseValue))
}

// filterItems applies the active query and select filters to the catalog list.
func filterItems(parseItems []docsItem, parseQuery, parseActiveFilter, parseStatusFilter, parseLevelFilter, parseModuleFilter string) []docsItem {
	parseNormalizedQuery := normalizeLowercase(parseQuery)
	parseFiltered := make([]docsItem, 0, len(parseItems))
	for _, parseItem := range parseItems {
		isParseTypeMatches := parseActiveFilter == filterAll || parseItem.Type == parseActiveFilter
		isParseStatusMatches := parseStatusFilter == allFilterValue || parseItem.Status == parseStatusFilter
		isParseLevelMatches := parseLevelFilter == allFilterValue || parseItem.Level == parseLevelFilter
		isParseModuleMatches := parseModuleFilter == allFilterValue || parseItem.Module == parseModuleFilter
		parseSearchParts := append([]string{parseItem.Title, parseItem.Type, parseItem.Level, parseItem.Status, parseItem.Module, parseItem.Blurb, parseItem.ReadTime}, parseItem.Tags...)
		parseSearchParts = append(parseSearchParts, parseItem.SearchTags...)
		parseSearchable := strings.ToLower(strings.Join(parseSearchParts, " "))
		isParseQueryMatches := parseNormalizedQuery == "" || strings.Contains(parseSearchable, parseNormalizedQuery)
		if isParseTypeMatches && isParseStatusMatches && isParseLevelMatches && isParseModuleMatches && isParseQueryMatches {
			parseFiltered = append(parseFiltered, parseItem)
		}
	}
	return parseFiltered
}

// sortItems returns a stable copy of the filtered list ordered by the active sort option.
func sortItems(parseItems []docsItem, parseSortBy string) []docsItem {
	parseSorted := append([]docsItem(nil), parseItems...)
	parseLevelRank := map[string]int{levelBeginner: 0, levelCore: 1, levelIntermediate: 2, levelAdvanced: 3}
	switch parseSortBy {
	case sortAlpha:
		sort.SliceStable(parseSorted, func(parseI, parseJ int) bool { return parseSorted[parseI].Title < parseSorted[parseJ].Title })
	case sortLevel:
		sort.SliceStable(parseSorted, func(parseI2, parseJ2 int) bool {
			parseLeft := 999
			parseRight := 999
			if parseValue, parseOk := parseLevelRank[parseSorted[parseI2].Level]; parseOk {
				parseLeft = parseValue
			}
			if parseValue2, parseOk2 := parseLevelRank[parseSorted[parseJ2].Level]; parseOk2 {
				parseRight = parseValue2
			}
			if parseLeft != parseRight {
				return parseLeft < parseRight
			}
			return parseSorted[parseI2].Title < parseSorted[parseJ2].Title
		})
	}
	return parseSorted
}

// countItemsByType counts how many catalog items belong to a single display category.
func countItemsByType(parseItems []docsItem, parseKind string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if parseItem.Type == parseKind {
			parseCount++
		}
	}
	return parseCount
}

// findSelectedItem resolves the currently selected item from the active filtered list.
func findSelectedItem(parseItems []docsItem, parseSelectedID int) (docsItem, bool) {
	for _, parseItem := range parseItems {
		if parseItem.ID == parseSelectedID {
			return parseItem, true
		}
	}
	return docsItem{}, false
}

// getCatalogItemIDByTitle resolves one catalog item ID by its display title.
func getCatalogItemIDByTitle(parseItems []docsItem, parseTitle string) int {
	parseExpectedTitle := normalizeLowercase(parseTitle)
	if parseExpectedTitle == "" {
		return 0
	}
	for _, parseItem := range parseItems {
		if normalizeLowercase(parseItem.Title) == parseExpectedTitle {
			return parseItem.ID
		}
	}
	return 0
}

// filteredItemsSignature produces a compact dependency key for filtered item lists.
func filteredItemsSignature(parseItems []docsItem) string {
	parseParts := make([]string, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseParts = append(parseParts, fmt.Sprintf("%d", parseItem.ID))
	}
	return strings.Join(parseParts, ",")
}
