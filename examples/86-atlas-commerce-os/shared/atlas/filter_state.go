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

func atlasCatalogFilterState(query catalogQueryState) atlasListFilterState {
	return atlasListFilterState{
		Query:     query.Search,
		Category:  fallback(query.Category, "all"),
		Warehouse: fallback(query.Warehouse, "all"),
		Sort:      query.Sort,
	}
}

func atlasMapFilterState(filters map[string]string) atlasListFilterState {
	if filters == nil {
		filters = map[string]string{}
	}
	return atlasListFilterState{
		Query:     filters["q"],
		Category:  fallback(filters["category"], "all"),
		Warehouse: fallback(filters["warehouse"], "all"),
		Status:    fallback(filters["status"], "all"),
		Sort:      filters["sort"],
	}
}

func atlasSavedViewFilterState(saved SavedViewPayload) atlasListFilterState {
	state := atlasMapFilterState(saved.Filters)
	if strings.TrimSpace(saved.SortKey) != "" {
		state.Sort = saved.SortKey
	}
	return state
}

func atlasSameListFilterState(left, right atlasListFilterState) bool {
	return atlasNormalizedFilterValue(left.Query) == atlasNormalizedFilterValue(right.Query) &&
		atlasNormalizedFilterValue(left.Category) == atlasNormalizedFilterValue(right.Category) &&
		atlasNormalizedFilterValue(left.Warehouse) == atlasNormalizedFilterValue(right.Warehouse) &&
		atlasNormalizedFilterValue(left.Status) == atlasNormalizedFilterValue(right.Status) &&
		atlasNormalizedFilterValue(left.Sort) == atlasNormalizedFilterValue(right.Sort)
}

func atlasBuildListFilterQuery(base url.Values, state atlasListFilterState) url.Values {
	next := url.Values{}
	for key, values := range base {
		next[key] = append([]string(nil), values...)
	}
	atlasAssignFilterQuery(next, "q", state.Query)
	atlasAssignFilterQuery(next, "category", state.Category)
	atlasAssignFilterQuery(next, "warehouse", state.Warehouse)
	atlasAssignFilterQuery(next, "status", state.Status)
	atlasAssignFilterQuery(next, "sort", state.Sort)
	return next
}

func atlasAssignFilterQuery(values url.Values, key, value string) {
	trimmed := atlasNormalizedFilterValue(value)
	if trimmed == "" || trimmed == "all" {
		values.Del(key)
		return
	}
	values.Set(key, trimmed)
}

func atlasNormalizedFilterValue(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func atlasFilterSubmitLabel(syncing bool, idle string) string {
	if syncing {
		return "Updating..."
	}
	return idle
}

func atlasSetFormFieldInTransition[T any](form ui.Form[T], field, value string) {
	startAtlasTransition(func() {
		form.SetField(field, value)
	})
}

func atlasSetFormInTransition[T any](form ui.Form[T], value T) {
	startAtlasTransition(func() {
		form.Set(value)
	})
}
