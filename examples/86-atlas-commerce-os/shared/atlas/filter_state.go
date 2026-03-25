package atlas

import (
	"net/url"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/ui"
)

const atlasFilterSyncDelay = 280 * time.Millisecond

type atlasListFilterState struct {
	Query     string
	Category  string
	Warehouse string
	Status    string
	Sort      string
}

func atlasCatalogFilterState(parseQuery catalogQueryState) atlasListFilterState {
	return atlasListFilterState{
		Query:     parseQuery.Search,
		Category:  fallback(parseQuery.Category, "all"),
		Warehouse: fallback(parseQuery.Warehouse, "all"),
		Sort:      parseQuery.Sort,
	}
}

func atlasMapFilterState(parseFilters map[string]string) atlasListFilterState {
	if parseFilters == nil {
		parseFilters = map[string]string{}
	}
	return atlasListFilterState{
		Query:     parseFilters["q"],
		Category:  fallback(parseFilters["category"], "all"),
		Warehouse: fallback(parseFilters["warehouse"], "all"),
		Status:    fallback(parseFilters["status"], "all"),
		Sort:      parseFilters["sort"],
	}
}

func atlasSavedViewFilterState(parseSaved SavedViewPayload) atlasListFilterState {
	parseState := atlasMapFilterState(parseSaved.Filters)
	if strings.TrimSpace(parseSaved.SortKey) != "" {
		parseState.Sort = parseSaved.SortKey
	}
	return parseState
}

func atlasSameListFilterState(parseLeft, parseRight atlasListFilterState) bool {
	return atlasNormalizedFilterValue(parseLeft.Query) == atlasNormalizedFilterValue(parseRight.Query) &&
		atlasNormalizedFilterValue(parseLeft.Category) == atlasNormalizedFilterValue(parseRight.Category) &&
		atlasNormalizedFilterValue(parseLeft.Warehouse) == atlasNormalizedFilterValue(parseRight.Warehouse) &&
		atlasNormalizedFilterValue(parseLeft.Status) == atlasNormalizedFilterValue(parseRight.Status) &&
		atlasNormalizedFilterValue(parseLeft.Sort) == atlasNormalizedFilterValue(parseRight.Sort)
}

func atlasBuildListFilterQuery(parseBase url.Values, parseState atlasListFilterState) url.Values {
	parseNext := url.Values{}
	for parseKey, parseValues := range parseBase {
		parseNext[parseKey] = append([]string(nil), parseValues...)
	}
	atlasAssignFilterQuery(parseNext, "q", parseState.Query)
	atlasAssignFilterQuery(parseNext, "category", parseState.Category)
	atlasAssignFilterQuery(parseNext, "warehouse", parseState.Warehouse)
	atlasAssignFilterQuery(parseNext, "status", parseState.Status)
	atlasAssignFilterQuery(parseNext, "sort", parseState.Sort)
	return parseNext
}

func atlasAssignFilterQuery(parseValues url.Values, parseKey, parseValue string) {
	parseTrimmed := atlasNormalizedFilterValue(parseValue)
	if parseTrimmed == "" || parseTrimmed == "all" {
		parseValues.Del(parseKey)
		return
	}
	parseValues.Set(parseKey, parseTrimmed)
}

func atlasNormalizedFilterValue(parseValue string) string {
	return strings.TrimSpace(strings.ToLower(parseValue))
}

func atlasFilterSubmitLabel(isSyncing bool, parseIdle string) string {
	if isSyncing {
		return "Updating..."
	}
	return parseIdle
}

func atlasSetFormFieldInTransition[T any](parseForm ui.Form[T], parseField, parseValue string) {
	startAtlasTransition(func() {
		parseForm.SetField(parseField, parseValue)
	})
}

func atlasSetFormInTransition[T any](parseForm ui.Form[T], parseValue T) {
	startAtlasTransition(func() {
		parseForm.Set(parseValue)
	})
}
