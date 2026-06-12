//go:build !js || !wasm

package virtualization

import "github.com/monstercameron/GoWebComponents/ui"

// buildListViewportEffect returns one native no-op effect because owned viewport observation only runs in browser builds.
func buildListViewportEffect(parseListID string, parseItemCount int, parseHeight float64, parseRowHeight float64, parseConfig ViewportConfig, parseKeysRef ui.Ref[[]string], parseKeyIndexRef ui.Ref[map[string]int], parseRestoreRef ui.Ref[restorationSnapshot], parseViewport ui.State[ViewportState], parsePublishDiagnostics func(ViewportState)) func() func() {
	_ = parseListID
	_ = parseItemCount
	_ = parseHeight
	_ = parseRowHeight
	_ = parseConfig
	_ = parseKeysRef
	_ = parseKeyIndexRef
	_ = parseRestoreRef
	_ = parseViewport
	_ = parsePublishDiagnostics
	return func() func() {
		return nil
	}
}

// buildListRestorationEffect returns one native no-op effect because browser restoration only runs in browser builds.
func buildListRestorationEffect(parseListID string, parseItemCount int, parseHeight float64, parseRowHeight float64, parseKeyIndex map[string]int, parseRestoreRef ui.Ref[restorationSnapshot]) func() func() {
	_ = parseListID
	_ = parseItemCount
	_ = parseHeight
	_ = parseRowHeight
	_ = parseKeyIndex
	_ = parseRestoreRef
	return func() func() {
		return nil
	}
}

// persistRestorationSnapshot is a native no-op because sessionStorage is unavailable outside browser builds.
func persistRestorationSnapshot(parseId string, parseSnapshot restorationSnapshot) {
	_ = parseId
	_ = parseSnapshot
}

// loadPersistedRestorationSnapshot always misses on native builds because sessionStorage is unavailable.
func loadPersistedRestorationSnapshot(parseId string) (restorationSnapshot, bool) {
	_ = parseId
	return restorationSnapshot{}, false
}
