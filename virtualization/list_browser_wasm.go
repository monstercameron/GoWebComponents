//go:build js && wasm
// +build js,wasm

package virtualization

import (
	"encoding/json"

	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
)

// buildListViewportEffect wires the browser-owned viewport subscription and restoration lifecycle for one list instance.
func buildListViewportEffect(parseListID string, parseItemCount int, parseHeight float64, parseRowHeight float64, parseConfig ViewportConfig, parseKeysRef ui.Ref[[]string], parseKeyIndexRef ui.Ref[map[string]int], parseRestoreRef ui.Ref[restorationSnapshot], parseViewport ui.State[ViewportState], parsePublishDiagnostics func(ViewportState)) func() func() {
	return func() func() {
		parseDocument, parseErr := interop.GetDocument()
		if parseErr != nil {
			return nil
		}
		parseElement, parseOk, parseErr := parseDocument.ElementByID(parseListID)
		if parseErr != nil || !parseOk {
			return nil
		}
		if parseSnapshot, parseSnapshotOK := loadRestorationSnapshot(parseListID); parseSnapshotOK {
			parseRestoreRef.Set(parseSnapshot)
			_ = restoreElementScrollTop(parseElement, parseSnapshot, parseKeyIndexRef.Get(), parseRowHeight, parseItemCount, parseHeight)
		}
		parseSub, parseErr := ObserveOwnedViewport(parseElement, parseConfig, func(parseNext ViewportState) {
			parseSnapshot := restorationSnapshot{ScrollTop: parseNext.ScrollTop}
			parseAnchorKeys := parseKeysRef.Get()
			if parseNext.Visible.Start >= 0 && parseNext.Visible.Start < len(parseAnchorKeys) {
				parseSnapshot.AnchorKey = parseAnchorKeys[parseNext.Visible.Start]
			}
			parseRestoreRef.Set(parseSnapshot)
			storeRestorationSnapshot(parseListID, parseSnapshot)
			persistRestorationSnapshot(parseListID, parseSnapshot)
			parseViewport.Set(parseNext)
			parsePublishDiagnostics(parseNext)
		})
		if parseErr != nil {
			return nil
		}
		return func() {
			storeRestorationSnapshot(parseListID, parseRestoreRef.Get())
			persistRestorationSnapshot(parseListID, parseRestoreRef.Get())
			parseSub.Cancel()
		}
	}
}

// buildListRestorationEffect reapplies the latest stored anchor when the item-key mapping changes.
func buildListRestorationEffect(parseListID string, parseItemCount int, parseHeight float64, parseRowHeight float64, parseKeyIndex map[string]int, parseRestoreRef ui.Ref[restorationSnapshot]) func() func() {
	return func() func() {
		parseDocument, parseErr := interop.GetDocument()
		if parseErr != nil {
			return nil
		}
		parseElement, parseOk, parseErr := parseDocument.ElementByID(parseListID)
		if parseErr != nil || !parseOk {
			return nil
		}
		parseSnapshot, parseSnapshotOK := loadRestorationSnapshot(parseListID)
		if !parseSnapshotOK {
			return nil
		}
		parseRestoreRef.Set(parseSnapshot)
		_ = restoreElementScrollTop(parseElement, parseSnapshot, parseKeyIndex, parseRowHeight, parseItemCount, parseHeight)
		return nil
	}
}

// persistRestorationSnapshot stores one browser restoration payload in sessionStorage when storage is available.
func persistRestorationSnapshot(parseId string, parseSnapshot restorationSnapshot) {
	if parseId == "" {
		return
	}
	parseStorage, parseErr := interop.GetSessionStorage()
	if parseErr != nil {
		return
	}
	parsePayload, parseErr := json.Marshal(parseSnapshot)
	if parseErr != nil {
		return
	}
	_ = parseStorage.SetItem(restorationStoragePrefix+parseId, string(parsePayload))
}

// loadPersistedRestorationSnapshot loads one browser restoration payload from sessionStorage when storage is available.
func loadPersistedRestorationSnapshot(parseId string) (restorationSnapshot, bool) {
	if parseId == "" {
		return restorationSnapshot{}, false
	}
	parseStorage, parseErr := interop.GetSessionStorage()
	if parseErr != nil {
		return restorationSnapshot{}, false
	}
	parseRaw, parseOk, parseErr := parseStorage.GetItem(restorationStoragePrefix + parseId)
	if parseErr != nil || !parseOk || parseRaw == "" {
		return restorationSnapshot{}, false
	}
	var parseSnapshot restorationSnapshot
	if parseErr2 := json.Unmarshal([]byte(parseRaw), &parseSnapshot); parseErr2 != nil {
		return restorationSnapshot{}, false
	}
	return parseSnapshot, parseOk
}
