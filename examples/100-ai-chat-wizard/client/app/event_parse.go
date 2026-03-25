//go:build js && wasm

package app

import (
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/ui"
)

func eventDatasetValue(e ui.Event, key string) string {
	return strings.TrimSpace(e.JSValue().Get("currentTarget").Get("dataset").Get(key).String())
}

func eventValueOrDataset(e ui.Event, key string) string {
	if value := strings.TrimSpace(e.GetValue()); value != "" {
		return value
	}
	return eventDatasetValue(e, key)
}

func eventDatasetInt(e ui.Event, key string) (int, bool) {
	value, err := strconv.Atoi(eventDatasetValue(e, key))
	if err != nil {
		return 0, false
	}
	return value, true
}

func eventDatasetInt64(e ui.Event, key string) (int64, bool) {
	value, err := strconv.ParseInt(eventDatasetValue(e, key), 10, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}
