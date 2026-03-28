package cachepolicy

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/client/cachecore"
)

const canvasSessionResourcePrefix = "canvas.session"

// CanvasSessionSnapshotPayload stores lightweight non-secret canvas/session UI state for refresh restore.
type CanvasSessionSnapshotPayload struct {
	PaneWidthRatio float64 `json:"pane_width_ratio"`
	SelectedItemID string  `json:"selected_item_id"`
	SessionState   []byte  `json:"session_state"`
	DebugState     []byte  `json:"-"`
}

// BuildCanvasSessionSnapshotPolicy returns one canvas/session snapshot cache policy.
func BuildCanvasSessionSnapshotPolicy() cachecore.CachePolicy {
	return cachecore.BuildCachePolicy(cachecore.PolicyClassSession, 45*time.Second, 24*time.Hour, true, true, false)
}

// BuildCanvasSessionResourceKey builds one canvas/session snapshot resource key for one route context.
func BuildCanvasSessionResourceKey(parseRouteKey string) string {
	return canvasSessionResourcePrefix + "|route=" + strings.TrimSpace(parseRouteKey)
}

// BuildCanvasSessionSnapshotPayloadJSON builds one sanitized canvas/session payload JSON contract without debug state.
func BuildCanvasSessionSnapshotPayloadJSON(parsePayload CanvasSessionSnapshotPayload) ([]byte, error) {
	parsePayload.SelectedItemID = strings.TrimSpace(parsePayload.SelectedItemID)
	parsePayload.SessionState = append([]byte(nil), parsePayload.SessionState...)
	parsePayload.DebugState = nil
	if parsePayload.PaneWidthRatio <= 0 {
		parsePayload.PaneWidthRatio = 0.5
	}
	if parsePayload.PaneWidthRatio < 0.2 {
		parsePayload.PaneWidthRatio = 0.2
	}
	if parsePayload.PaneWidthRatio > 0.8 {
		parsePayload.PaneWidthRatio = 0.8
	}
	return json.Marshal(parsePayload)
}

// ParseCanvasSessionSnapshotPayloadJSON parses one sanitized canvas/session payload JSON contract.
func ParseCanvasSessionSnapshotPayloadJSON(parseRaw []byte) (CanvasSessionSnapshotPayload, error) {
	parsePayload := CanvasSessionSnapshotPayload{}
	if parseErr := json.Unmarshal(parseRaw, &parsePayload); parseErr != nil {
		return CanvasSessionSnapshotPayload{}, parseErr
	}
	parsePayload.SelectedItemID = strings.TrimSpace(parsePayload.SelectedItemID)
	parsePayload.SessionState = append([]byte(nil), parsePayload.SessionState...)
	parsePayload.DebugState = nil
	return parsePayload, nil
}

// StoreCanvasSessionSnapshot stores one lightweight canvas/session snapshot payload.
func StoreCanvasSessionSnapshot(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseRouteKey string, parsePayload CanvasSessionSnapshotPayload) error {
	if parseStorage == nil {
		return nil
	}
	parsePolicy := BuildCanvasSessionSnapshotPolicy()
	parsePayloadJSON, parseErr := BuildCanvasSessionSnapshotPayloadJSON(parsePayload)
	if parseErr != nil {
		return parseErr
	}
	parseRecord := cachecore.BuildCacheRecordEnvelope(
		strings.TrimSpace(parseScopeKey),
		BuildCanvasSessionResourceKey(parseRouteKey),
		"",
		"",
		time.Now().UTC(),
		parsePolicy.StaleAfter,
		parsePolicy.ExpiresAfter,
		parsePayloadJSON,
		cachecore.CacheStatusReady,
		"",
	)
	return parseStorage.Set(parseCtx, parseRecord)
}

// ReadCanvasSessionSnapshot reads one canvas/session snapshot with snapshot-first SWR behavior.
func ReadCanvasSessionSnapshot(
	parseCtx context.Context,
	parseAPI *cachecore.UIAPI,
	parseScopeKey string,
	parseRouteKey string,
	parseRefresh func(context.Context, string, string) (cachecore.CacheRecordEnvelope, error),
) (cachecore.CachedResourceView, error) {
	if parseAPI == nil {
		return cachecore.CachedResourceView{}, nil
	}
	return parseAPI.ReadCachedResource(
		parseCtx,
		strings.TrimSpace(parseScopeKey),
		BuildCanvasSessionResourceKey(parseRouteKey),
		BuildCanvasSessionSnapshotPolicy(),
		parseRefresh,
	)
}
