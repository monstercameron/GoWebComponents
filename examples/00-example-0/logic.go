//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/utils"
)

// catalogDataURL resolves the catalog asset relative to the current document URL.
func catalogDataURL() string {
	return utils.ResolveDocumentURL(catalogDataRelativeURL)
}

// docsSourceURL resolves a markdown source asset relative to the current document URL.
func docsSourceURL(sourcePath string) string {
	if sourcePath == "" {
		return ""
	}
	return utils.ResolveDocumentURL(sourcePath)
}

// decodeCatalogJSON validates the fetched JSON payload before the UI reads from it.
func decodeCatalogJSON(payload []byte) (docsCatalog, error) {
	var catalog docsCatalog
	if err := json.Unmarshal(payload, &catalog); err != nil {
		return docsCatalog{}, fmt.Errorf("invalid catalog.json: %w", err)
	}
	if len(catalog.Modules) == 0 {
		return docsCatalog{}, fmt.Errorf("catalog.json must define modules")
	}
	if len(catalog.Statuses) == 0 {
		return docsCatalog{}, fmt.Errorf("catalog.json must define statuses")
	}
	if len(catalog.Levels) == 0 {
		return docsCatalog{}, fmt.Errorf("catalog.json must define levels")
	}
	if len(catalog.Filters) == 0 {
		return docsCatalog{}, fmt.Errorf("catalog.json must define filters")
	}
	if len(catalog.SortOptions) == 0 {
		return docsCatalog{}, fmt.Errorf("catalog.json must define sortOptions")
	}
	if len(catalog.Items) == 0 {
		return docsCatalog{}, fmt.Errorf("catalog.json must define at least one item")
	}
	for _, item := range catalog.Items {
		if !containsCatalogValue(catalog.Filters, item.Type) {
			return docsCatalog{}, fmt.Errorf("catalog.json item %d type %q must appear in filters", item.ID, item.Type)
		}
		if !containsCatalogValue(catalog.Statuses, item.Status) {
			return docsCatalog{}, fmt.Errorf("catalog.json item %d status %q must appear in statuses", item.ID, item.Status)
		}
		if !containsCatalogValue(catalog.Levels, item.Level) {
			return docsCatalog{}, fmt.Errorf("catalog.json item %d level %q must appear in levels", item.ID, item.Level)
		}
		if !containsCatalogValue(catalog.Modules, item.Module) {
			return docsCatalog{}, fmt.Errorf("catalog.json item %d module %q must appear in modules", item.ID, item.Module)
		}
	}
	return catalog, nil
}

func containsCatalogValue(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

// decodeFetchedCatalog converts the low-level fetch payload into a validated catalog.
func decodeFetchedCatalog(payload interface{}) (docsCatalog, error) {
	textPayload, ok := payload.(string)
	if !ok {
		return docsCatalog{}, fmt.Errorf("catalog response must be text")
	}
	return decodeCatalogJSON([]byte(textPayload))
}

// decodeFetchedText validates plain text fetch responses used by markdown documents.
func decodeFetchedText(payload interface{}) (string, error) {
	textPayload, ok := payload.(string)
	if !ok {
		return "", fmt.Errorf("document response must be text")
	}
	return textPayload, nil
}

// catalogCacheKey returns the cache key used to share catalog data across renders.
func catalogCacheKey(url string) string {
	return catalogCacheKeyPrefix + url
}

// markdownCacheKey returns the cache key used to share markdown document loads.
func markdownCacheKey(url string) string {
	return markdownCacheKeyPrefix + url
}

// loadCatalogResource fetches and validates the catalog through the cached-resource loader path.
func loadCatalogResource(ctx context.Context, url string) (docsCatalog, error) {
	resultCh := fetch.Fetch(url, fetch.Options{})
	select {
	case <-ctx.Done():
		return docsCatalog{}, ctx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			return docsCatalog{}, result.Err
		}
		return decodeFetchedCatalog(result.Data)
	}
}

// loadMarkdownResource fetches and validates a markdown document as plain text.
func loadMarkdownResource(ctx context.Context, url string) (string, error) {
	resultCh := fetch.Fetch(url, fetch.Options{})
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			return "", result.Err
		}
		return decodeFetchedText(result.Data)
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// normalizeLowercase trims and lowercases values before they are used in comparisons.
func normalizeLowercase(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

// filterItems applies the active query and select filters to the catalog list.
func filterItems(items []docsItem, query, activeFilter, statusFilter, levelFilter, moduleFilter string) []docsItem {
	normalizedQuery := normalizeLowercase(query)
	filtered := make([]docsItem, 0, len(items))
	for _, item := range items {
		typeMatches := activeFilter == filterAll || item.Type == activeFilter
		statusMatches := statusFilter == allFilterValue || item.Status == statusFilter
		levelMatches := levelFilter == allFilterValue || item.Level == levelFilter
		moduleMatches := moduleFilter == allFilterValue || item.Module == moduleFilter
		searchParts := append([]string{item.Title, item.Type, item.Level, item.Status, item.Module, item.Blurb, item.ReadTime}, item.Tags...)
		searchParts = append(searchParts, item.SearchTags...)
		searchable := strings.ToLower(strings.Join(searchParts, " "))
		queryMatches := normalizedQuery == "" || strings.Contains(searchable, normalizedQuery)
		if typeMatches && statusMatches && levelMatches && moduleMatches && queryMatches {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// sortItems returns a stable copy of the filtered list ordered by the active sort option.
func sortItems(items []docsItem, sortBy string) []docsItem {
	sorted := append([]docsItem(nil), items...)
	levelRank := map[string]int{levelBeginner: 0, levelCore: 1, levelIntermediate: 2, levelAdvanced: 3}
	switch sortBy {
	case sortAlpha:
		sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Title < sorted[j].Title })
	case sortLevel:
		sort.SliceStable(sorted, func(i, j int) bool {
			left := 999
			right := 999
			if value, ok := levelRank[sorted[i].Level]; ok {
				left = value
			}
			if value, ok := levelRank[sorted[j].Level]; ok {
				right = value
			}
			if left != right {
				return left < right
			}
			return sorted[i].Title < sorted[j].Title
		})
	}
	return sorted
}

// countItemsByType counts how many catalog items belong to a single display category.
func countItemsByType(items []docsItem, kind string) int {
	count := 0
	for _, item := range items {
		if item.Type == kind {
			count++
		}
	}
	return count
}

// findSelectedItem resolves the currently selected item from the active filtered list.
func findSelectedItem(items []docsItem, selectedID int) (docsItem, bool) {
	for _, item := range items {
		if item.ID == selectedID {
			return item, true
		}
	}
	return docsItem{}, false
}

// filteredItemsSignature produces a compact dependency key for filtered item lists.
func filteredItemsSignature(items []docsItem) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%d", item.ID))
	}
	return strings.Join(parts, ",")
}
