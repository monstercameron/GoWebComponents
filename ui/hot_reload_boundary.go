package ui

import (
	"encoding/json"
	"fmt"
	"strings"
)

func hotReloadBoundaryKey(resetKeys []interface{}) string {
	if len(resetKeys) == 0 {
		return "__gwc_hotreload_boundary__"
	}

	data, err := json.Marshal(resetKeys)
	if err == nil {
		return "__gwc_hotreload_boundary__:" + string(data)
	}

	parts := make([]string, 0, len(resetKeys))
	for _, key := range resetKeys {
		parts = append(parts, fmt.Sprintf("%#v", key))
	}
	return "__gwc_hotreload_boundary__:" + strings.Join(parts, "|")
}
