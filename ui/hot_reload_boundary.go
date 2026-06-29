package ui

import (
	"encoding/json"
	"fmt"
	"strings"
)

// hotReloadBoundaryKey is a core package helper.
func hotReloadBoundaryKey(resetKeys []any) string {
	if len(resetKeys) == 0 {
		return "__gwc_hotreload_boundary__"
	}

	parseBoundaryData, parseBoundaryErr := json.Marshal(resetKeys)
	if parseBoundaryErr == nil {
		return "__gwc_hotreload_boundary__:" + string(parseBoundaryData)
	}

	parseBoundaryParts := make([]string, 0, len(resetKeys))
	for _, parseBoundaryKey := range resetKeys {
		parseBoundaryParts = append(parseBoundaryParts, fmt.Sprintf("%#v", parseBoundaryKey))
	}
	return "__gwc_hotreload_boundary__:" + strings.Join(parseBoundaryParts, "|")
}
