//go:build js && wasm && !production
// +build js,wasm,!production

package hotreload

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	runtimepkg "github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/state"
)

const appBridgeGlobal = "GoWebComponentsHotReloadApp"
const liveReloadGlobal = "GoLiveReload"

var enabled bool
var bridgeInstalled bool
var currentConfig Config
var lastRestoreResult hotReloadRestoreResult
var exportSub interop.Subscription
var prepareSub interop.Subscription
var importSub interop.Subscription
var getLastRestoreResultSub interop.Subscription
var getDiagnosticsSub interop.Subscription
var getActivitySub interop.Subscription

type bridgeSnapshot struct {
	ResetKey          string                                  `json:"resetKey,omitempty"`
	RestoreMode       string                                  `json:"restoreMode,omitempty"`
	ChangedComponents []string                                `json:"changedComponents,omitempty"`
	State             state.Snapshot                          `json:"state,omitempty"`
	Components        []runtimepkg.HotReloadComponentSnapshot `json:"components,omitempty"`
}

type hotReloadRestoreResult struct {
	Outcome     string                `json:"outcome"`
	Message     string                `json:"message,omitempty"`
	Diagnostics []hotReloadDiagnostic `json:"diagnostics,omitempty"`
}

type hotReloadDiagnostic struct {
	Severity       string   `json:"severity,omitempty"`
	Classification string   `json:"classification,omitempty"`
	Message        string   `json:"message,omitempty"`
	Path           string   `json:"path,omitempty"`
	ComponentStack []string `json:"componentStack,omitempty"`
}

type hotReloadActivityEntry struct {
	Domain         string            `json:"domain,omitempty"`
	Level          string            `json:"level,omitempty"`
	Classification string            `json:"classification,omitempty"`
	Message        string            `json:"message,omitempty"`
	Timestamp      string            `json:"timestamp,omitempty"`
	Fields         map[string]string `json:"fields,omitempty"`
}

// Configure installs or reconfigures the development hot reload bridge.
func Configure(parseConfig Config) {
	parseNormalized := normalizeConfig(parseConfig)
	if enabled && bridgeInstalled && configEqual(currentConfig, parseNormalized) {
		restorePendingSnapshot()
		return
	}

	enabled = true
	currentConfig = parseNormalized
	uninstallBridge()
	installBridge(parseNormalized)
	restorePendingSnapshot()
}

// Disable removes the development hot reload bridge from the page.
func Disable() {
	enabled = false
	currentConfig = Config{}
	lastRestoreResult = hotReloadRestoreResult{}
	uninstallBridge()
}

// Enabled reports whether the public hot reload bridge is enabled.
func Enabled() bool {
	return enabled
}

// IsEnabled is a compatibility wrapper around Enabled.
func IsEnabled() bool {
	return Enabled()
}

// GetSnapshot returns the current serializable hot reload snapshot payload.
func GetSnapshot() (string, error) {
	parseStateSnapshot, parseErr := state.GetSnapshot()
	if parseErr != nil {
		return "", parseErr
	}
	parseSnapshot := bridgeSnapshot{
		ResetKey:   currentConfig.ResetKey,
		State:      parseStateSnapshot.Select(currentConfig.AtomIDs...),
		Components: runtimepkg.GetGlobalRuntime().CaptureHotReloadSnapshot().Components,
	}
	parseData, parseErr := json.Marshal(parseSnapshot)
	if parseErr != nil {
		return "", parseErr
	}
	return string(parseData), nil
}

// ApplySnapshot restores a previously exported hot reload snapshot payload.
func ApplySnapshot(parsePayload string) error {
	_, parseErr := importSnapshot(parsePayload)
	return parseErr
}

func importSnapshot(parsePayload string) (hotReloadRestoreResult, error) {
	if strings.TrimSpace(parsePayload) == "" {
		parseResult := hotReloadRestoreResult{Outcome: "skipped-empty", Message: "No hot reload snapshot was available to restore."}
		lastRestoreResult = parseResult
		return parseResult, nil
	}

	var parseSnapshot bridgeSnapshot
	if parseErr := json.Unmarshal([]byte(parsePayload), &parseSnapshot); parseErr == nil && (len(parseSnapshot.State) > 0 || len(parseSnapshot.Components) > 0) {
		if !snapshotResetKeyMatches(parseSnapshot.ResetKey) {
			parseResult2 := hotReloadRestoreResult{Outcome: "skipped-reset-key", Message: "Hot reload snapshot was discarded because ResetKey changed."}
			lastRestoreResult = parseResult2
			return parseResult2, nil
		}
		if parseSnapshot.State != nil {
			parseNormalized := make(state.Snapshot, len(parseSnapshot.State))
			for parseKey, parseValue := range parseSnapshot.State {
				parseNormalized[parseKey] = runtimepkg.NormalizeHotReloadValue(parseValue)
			}
			parseSnapshot.State = parseNormalized
		}
		if parseSnapshot.State != nil {
			if parseErr2 := state.ApplySnapshot(parseSnapshot.State); parseErr2 != nil {
				parseResult3 := hotReloadRestoreResult{Outcome: "error", Message: parseErr2.Error()}
				lastRestoreResult = parseResult3
				return parseResult3, parseErr2
			}
		}
		parseDecision := runtimepkg.GetGlobalRuntime().RestoreHotReloadSnapshotWithPlan(runtimepkg.HotReloadSnapshot{Components: parseSnapshot.Components}, runtimepkg.HotReloadRestorePlan{
			Selective:         strings.TrimSpace(parseSnapshot.RestoreMode) == "selective",
			ChangedIdentities: append([]string(nil), parseSnapshot.ChangedComponents...),
		})
		parseResult4 := hotReloadRestoreResult{Outcome: "restored", Message: "Hot reload snapshot was queued for restore."}
		if parseDecision.Strategy == "selective" {
			parseResult4.Outcome = "restored-selective"
			parseResult4.Message = fmt.Sprintf("Selective hot reload restore queued with %d changed component identities remounted.", len(parseSnapshot.ChangedComponents))
		} else if strings.TrimSpace(parseSnapshot.RestoreMode) == "selective" && strings.TrimSpace(parseDecision.UnsafeReason) != "" {
			parseResult4.Outcome = "restored-legacy"
			parseResult4.Message = "Selective hot reload fell back to full compatible restore: " + parseDecision.UnsafeReason
		}
		lastRestoreResult = parseResult4
		return parseResult4, nil
	}

	parseLegacyState, parseErr3 := state.UnmarshalSnapshotJSON([]byte(parsePayload))
	if parseErr3 != nil {
		parseResult5 := hotReloadRestoreResult{Outcome: "error", Message: parseErr3.Error()}
		lastRestoreResult = parseResult5
		return parseResult5, parseErr3
	}
	if parseErr4 := state.ApplySnapshot(parseLegacyState); parseErr4 != nil {
		parseResult6 := hotReloadRestoreResult{Outcome: "error", Message: parseErr4.Error()}
		lastRestoreResult = parseResult6
		return parseResult6, parseErr4
	}
	parseResult7 := hotReloadRestoreResult{Outcome: "restored-legacy", Message: "Legacy hot reload state was restored."}
	lastRestoreResult = parseResult7
	return parseResult7, nil
}

// Prepare runs cleanup needed before the current runtime instance is replaced.
func Prepare() {
	runtimepkg.GetGlobalRuntime().PrepareForHotReload()
}

func normalizeConfig(parseConfig Config) Config {
	parseNormalized := Config{ResetKey: strings.TrimSpace(parseConfig.ResetKey)}
	if len(parseConfig.AtomIDs) > 0 {
		parseNormalized.AtomIDs = append([]string(nil), parseConfig.AtomIDs...)
	}
	return parseNormalized
}

func configEqual(parseLeft, parseRight Config) bool {
	if parseLeft.ResetKey != parseRight.ResetKey {
		return false
	}
	if len(parseLeft.AtomIDs) != len(parseRight.AtomIDs) {
		return false
	}
	for parseIndex := range parseLeft.AtomIDs {
		if parseLeft.AtomIDs[parseIndex] != parseRight.AtomIDs[parseIndex] {
			return false
		}
	}
	return true
}

func installBridge(parseConfig Config) {
	parseGlobal, parseErr := interop.GetGlobalThis()
	if parseErr != nil {
		return
	}
	if parseErr2 := parseGlobal.Set(appBridgeGlobal, map[string]any{}); parseErr2 != nil {
		return
	}
	parseBridge := parseGlobal.Get(appBridgeGlobal)
	if !parseBridge.Present() {
		return
	}

	exportSub, parseErr = parseBridge.SetFunction("captureSnapshot", func(parseArgs ...interop.Value) any {
		parsePayload, parseErr3 := GetSnapshot()
		if parseErr3 != nil {
			return ""
		}
		return parsePayload
	})
	if parseErr != nil {
		_ = parseGlobal.Delete(appBridgeGlobal)
		return
	}

	prepareSub, parseErr = parseBridge.SetFunction("prepare", func(parseArgs2 ...interop.Value) any {
		Prepare()
		return nil
	})
	if parseErr != nil {
		exportSub.Cancel()
		_ = parseGlobal.Delete(appBridgeGlobal)
		return
	}

	importSub, parseErr = parseBridge.SetFunction("restoreSnapshot", func(parseArgs3 ...interop.Value) any {
		if len(parseArgs3) == 0 || !parseArgs3[0].Present() {
			parseResult := hotReloadRestoreResult{Outcome: "skipped-empty", Message: "No hot reload snapshot was available to restore."}
			lastRestoreResult = parseResult
			return map[string]any{
				"outcome": parseResult.Outcome,
				"message": parseResult.Message,
			}
		}
		parseResult2, parseErr4 := importSnapshot(parseArgs3[0].String())
		parseResponse := map[string]any{
			"outcome": parseResult2.Outcome,
			"message": parseResult2.Message,
		}
		if len(parseResult2.Diagnostics) > 0 {
			parseResponse["diagnostics"] = parseResult2.Diagnostics
		}
		if parseErr4 != nil {
			parseResponse["error"] = parseErr4.Error()
		}
		return parseResponse
	})
	if parseErr != nil {
		exportSub.Cancel()
		prepareSub.Cancel()
		_ = parseGlobal.Delete(appBridgeGlobal)
		return
	}

	getLastRestoreResultSub, parseErr = parseBridge.SetFunction("getLastRestoreResult", func(parseArgs4 ...interop.Value) any {
		return map[string]any{
			"outcome":     lastRestoreResult.Outcome,
			"message":     lastRestoreResult.Message,
			"diagnostics": lastRestoreResult.Diagnostics,
		}
	})
	if parseErr != nil {
		exportSub.Cancel()
		prepareSub.Cancel()
		importSub.Cancel()
		_ = parseGlobal.Delete(appBridgeGlobal)
		return
	}

	getDiagnosticsSub, parseErr = parseBridge.SetFunction("getHotReloadDiagnostics", func(parseArgs5 ...interop.Value) any {
		return hotReloadDiagnostics()
	})
	if parseErr != nil {
		exportSub.Cancel()
		prepareSub.Cancel()
		importSub.Cancel()
		getLastRestoreResultSub.Cancel()
		_ = parseGlobal.Delete(appBridgeGlobal)
		return
	}

	getActivitySub, parseErr = parseBridge.SetFunction("getHotReloadActivity", func(parseArgs6 ...interop.Value) any {
		return hotReloadActivity()
	})
	if parseErr != nil {
		exportSub.Cancel()
		prepareSub.Cancel()
		importSub.Cancel()
		getLastRestoreResultSub.Cancel()
		getDiagnosticsSub.Cancel()
		_ = parseGlobal.Delete(appBridgeGlobal)
		return
	}

	_ = parseBridge.Set("enabled", true)
	if parseConfig.ResetKey != "" {
		_ = parseBridge.Set("resetKey", parseConfig.ResetKey)
	}
	if len(parseConfig.AtomIDs) > 0 {
		_ = parseBridge.Set("atomIDs", append([]string(nil), parseConfig.AtomIDs...))
	}
	bridgeInstalled = true
}

func snapshotResetKeyMatches(parseSnapshotResetKey string) bool {
	parseCurrentResetKey := strings.TrimSpace(currentConfig.ResetKey)
	parseSnapshotResetKey = strings.TrimSpace(parseSnapshotResetKey)
	if parseCurrentResetKey == "" && parseSnapshotResetKey == "" {
		return true
	}
	return parseCurrentResetKey == parseSnapshotResetKey
}

func uninstallBridge() {
	exportSub.Cancel()
	prepareSub.Cancel()
	importSub.Cancel()
	getLastRestoreResultSub.Cancel()
	getDiagnosticsSub.Cancel()
	getActivitySub.Cancel()
	exportSub = interop.Subscription{}
	prepareSub = interop.Subscription{}
	importSub = interop.Subscription{}
	getLastRestoreResultSub = interop.Subscription{}
	getDiagnosticsSub = interop.Subscription{}
	getActivitySub = interop.Subscription{}
	bridgeInstalled = false

	parseGlobal, parseErr := interop.GetGlobalThis()
	if parseErr != nil {
		return
	}
	_ = parseGlobal.Delete(appBridgeGlobal)
}

func restorePendingSnapshot() {
	parseGlobal, parseErr := interop.GetGlobalThis()
	if parseErr != nil {
		return
	}

	parseLiveReload := parseGlobal.Get(liveReloadGlobal)
	if !parseLiveReload.Present() {
		return
	}
	getStoredState := parseLiveReload.Get("getStoredState")
	if !getStoredState.Present() {
		return
	}

	parseSaved, parseErr := getStoredState.Invoke()
	if parseErr != nil || !parseSaved.Present() || strings.TrimSpace(parseSaved.String()) == "" {
		return
	}

	if parseErr2 := ApplySnapshot(parseSaved.String()); parseErr2 != nil {
		return
	}
	clearStoredState := parseLiveReload.Get("clearStoredState")
	if clearStoredState.Present() {
		_, _ = clearStoredState.Invoke()
	}
}

func hotReloadDiagnostics() []hotReloadDiagnostic {
	parseDiagnostics := runtimepkg.GetDiagnostics()
	parseFiltered := make([]hotReloadDiagnostic, 0, len(parseDiagnostics))
	for _, parseDiagnostic := range parseDiagnostics {
		if !strings.Contains(strings.ToLower(parseDiagnostic.Message), "hot reload") {
			continue
		}
		parseFiltered = append(parseFiltered, hotReloadDiagnostic{
			Severity:       string(parseDiagnostic.Severity),
			Classification: string(parseDiagnostic.Classification),
			Message:        parseDiagnostic.Message,
			Path:           parseDiagnostic.Path,
			ComponentStack: append([]string(nil), parseDiagnostic.ComponentStack...),
		})
	}
	lastRestoreResult.Diagnostics = parseFiltered
	return parseFiltered
}

func hotReloadActivity() []hotReloadActivityEntry {
	parseLogs := runtimepkg.GetLogs()
	parseFiltered := make([]hotReloadActivityEntry, 0, len(parseLogs))
	for _, parseEntry := range parseLogs {
		parseDomain := strings.TrimSpace(parseEntry.Domain)
		parseMessage := strings.TrimSpace(parseEntry.Message)
		if parseDomain == "" || parseMessage == "" {
			continue
		}
		isParseInclude := parseDomain == "hotreload"
		if !isParseInclude && parseDomain == "router" {
			parseLower := strings.ToLower(parseMessage)
			isParseInclude = strings.Contains(parseLower, "route loader") || strings.Contains(parseLower, "before-enter") || strings.Contains(parseLower, "before-leave") || strings.Contains(parseLower, "navigation")
		}
		if !isParseInclude {
			continue
		}
		parseFiltered = append(parseFiltered, hotReloadActivityEntry{
			Domain:         parseDomain,
			Level:          string(parseEntry.Level),
			Classification: string(parseEntry.Classification),
			Message:        parseMessage,
			Timestamp:      parseEntry.Timestamp,
			Fields:         parseEntry.Fields,
		})
	}
	if len(parseFiltered) > 6 {
		parseFiltered = append([]hotReloadActivityEntry(nil), parseFiltered[len(parseFiltered)-6:]...)
	}
	return parseFiltered
}

func appBridge() (interop.Value, error) {
	parseGlobal, parseErr := interop.GetGlobalThis()
	if parseErr != nil {
		return interop.Value{}, parseErr
	}
	parseBridge := parseGlobal.Get(appBridgeGlobal)
	if !parseBridge.Present() {
		return interop.Value{}, errors.New("hot reload bridge not installed")
	}
	return parseBridge, nil
}
