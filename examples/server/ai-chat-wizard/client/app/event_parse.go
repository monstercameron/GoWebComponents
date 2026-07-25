//go:build js && wasm

package app

import (
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func parseEventDatasetValue(parseE ui.Event, parseKey string) string {
	return strings.TrimSpace(parseE.JSValue().Get("currentTarget").Get("dataset").Get(parseKey).String())
}

func parseEventValueOrDataset(parseE ui.Event, parseKey string) string {
	if parseValue := strings.TrimSpace(parseE.GetValue()); parseValue != "" {
		return parseValue
	}
	return parseEventDatasetValue(parseE, parseKey)
}

func parseEventDatasetInt(parseE ui.Event, parseKey string) (int, bool) {
	parseValue, parseErr := strconv.Atoi(parseEventDatasetValue(parseE, parseKey))
	if parseErr != nil {
		return 0, false
	}
	return parseValue, true
}

func parseEventDatasetInt64(parseE ui.Event, parseKey string) (int64, bool) {
	parseValue, parseErr := strconv.ParseInt(parseEventDatasetValue(parseE, parseKey), 10, 64)
	if parseErr != nil {
		return 0, false
	}
	return parseValue, true
}
