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
	return catalog, nil
}

// decodeFetchedCatalog converts the low-level fetch payload into a validated catalog.
func decodeFetchedCatalog(payload interface{}) (docsCatalog, error) {
	textPayload, ok := payload.(string)
	if !ok {
		return docsCatalog{}, fmt.Errorf("catalog response must be text")
	}
	return decodeCatalogJSON([]byte(textPayload))
}

// catalogCacheKey returns the cache key used to share catalog data across renders.
func catalogCacheKey(url string) string {
	return catalogCacheKeyPrefix + url
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
		searchable := strings.ToLower(strings.Join(append([]string{item.Title, item.Type, item.Level, item.Status, item.Module, item.Blurb, item.ReadTime}, item.Tags...), " "))
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