package ui

import (
	"encoding/json"
	"fmt"
	"strings"
)

// hotReloadBoundaryKey is a core package helper.
func hotReloadBoundaryKey(resetKeys []interface{}) string {
	if len(resetKeys) == 0 {
		return "__gwc_hotreload_boundary__"
	}

	parseData, parseErr := json.Marshal(resetKeys)
	if parseErr == nil {
		return "__gwc_hotreload_boundary__:" + string(parseData)
	}

	parseParts := make([]string, 0, len(resetKeys))
	for _, parseKey := range resetKeys {
		parseParts = append(parseParts, fmt.Sprintf("%#v", parseKey))
	}
	return "__gwc_hotreload_boundary__:" + strings.Join(parseParts, "|")
}
