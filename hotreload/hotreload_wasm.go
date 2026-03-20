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
func Configure(config Config) {
	normalized := normalizeConfig(config)
	if enabled && bridgeInstalled && configEqual(currentConfig, normalized) {
		restorePendingSnapshot()
		return
	}

	enabled = true
	currentConfig = normalized
	uninstallBridge()
	installBridge(normalized)
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

// ExportSnapshot returns the current serializable hot reload snapshot payload.
func ExportSnapshot() (string, error) {
	snapshot := bridgeSnapshot{
		ResetKey:   currentConfig.ResetKey,
		State:      state.ExportSnapshot().Select(currentConfig.AtomIDs...),
		Components: runtimepkg.GetGlobalRuntime().CaptureHotReloadSnapshot().Components,
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ImportSnapshot restores a previously exported hot reload snapshot payload.
func ImportSnapshot(payload string) error {
	_, err := importSnapshot(payload)
	return err
}

func importSnapshot(payload string) (hotReloadRestoreResult, error) {
	if strings.TrimSpace(payload) == "" {
		result := hotReloadRestoreResult{Outcome: "skipped-empty", Message: "No hot reload snapshot was available to restore."}
		lastRestoreResult = result
		return result, nil
	}

	var snapshot bridgeSnapshot
	if err := json.Unmarshal([]byte(payload), &snapshot); err == nil && (len(snapshot.State) > 0 || len(snapshot.Components) > 0) {
		if !snapshotResetKeyMatches(snapshot.ResetKey) {
			result := hotReloadRestoreResult{Outcome: "skipped-reset-key", Message: "Hot reload snapshot was discarded because ResetKey changed."}
			lastRestoreResult = result
			return result, nil
		}
		if snapshot.State != nil {
			normalized := make(state.Snapshot, len(snapshot.State))
			for key, value := range snapshot.State {
				normalized[key] = runtimepkg.NormalizeHotReloadValue(value)
			}
			snapshot.State = normalized
		}
		if snapshot.State != nil {
			if err := state.ImportSnapshot(snapshot.State); err != nil {
				result := hotReloadRestoreResult{Outcome: "error", Message: err.Error()}
				lastRestoreResult = result
				return result, err
			}
		}
		decision := runtimepkg.GetGlobalRuntime().RestoreHotReloadSnapshotWithPlan(runtimepkg.HotReloadSnapshot{Components: snapshot.Components}, runtimepkg.HotReloadRestorePlan{
			Selective:         strings.TrimSpace(snapshot.RestoreMode) == "selective",
			ChangedIdentities: append([]string(nil), snapshot.ChangedComponents...),
		})
		result := hotReloadRestoreResult{Outcome: "restored", Message: "Hot reload snapshot was queued for restore."}
		if decision.Strategy == "selective" {
			result.Outcome = "restored-selective"
			result.Message = fmt.Sprintf("Selective hot reload restore queued with %d changed component identities remounted.", len(snapshot.ChangedComponents))
		} else if strings.TrimSpace(snapshot.RestoreMode) == "selective" && strings.TrimSpace(decision.UnsafeReason) != "" {
			result.Outcome = "restored-legacy"
			result.Message = "Selective hot reload fell back to full compatible restore: " + decision.UnsafeReason
		}
		lastRestoreResult = result
		return result, nil
	}

	legacyState, err := state.UnmarshalSnapshotJSON([]byte(payload))
	if err != nil {
		result := hotReloadRestoreResult{Outcome: "error", Message: err.Error()}
		lastRestoreResult = result
		return result, err
	}
	if err := state.ImportSnapshot(legacyState); err != nil {
		result := hotReloadRestoreResult{Outcome: "error", Message: err.Error()}
		lastRestoreResult = result
		return result, err
	}
	result := hotReloadRestoreResult{Outcome: "restored-legacy", Message: "Legacy hot reload state was restored."}
	lastRestoreResult = result
	return result, nil
}

// Prepare runs cleanup needed before the current runtime instance is replaced.
func Prepare() {
	runtimepkg.GetGlobalRuntime().PrepareForHotReload()
}

func normalizeConfig(config Config) Config {
	normalized := Config{ResetKey: strings.TrimSpace(config.ResetKey)}
	if len(config.AtomIDs) > 0 {
		normalized.AtomIDs = append([]string(nil), config.AtomIDs...)
	}
	return normalized
}

func configEqual(left, right Config) bool {
	if left.ResetKey != right.ResetKey {
		return false
	}
	if len(left.AtomIDs) != len(right.AtomIDs) {
		return false
	}
	for index := range left.AtomIDs {
		if left.AtomIDs[index] != right.AtomIDs[index] {
			return false
		}
	}
	return true
}

func installBridge(config Config) {
	global, err := interop.GlobalThis()
	if err != nil {
		return
	}
	if err := global.Set(appBridgeGlobal, map[string]any{}); err != nil {
		return
	}
	bridge := global.Get(appBridgeGlobal)
	if !bridge.Present() {
		return
	}

	exportSub, err = bridge.SetFunction("captureSnapshot", func(args ...interop.Value) any {
		payload, err := ExportSnapshot()
		if err != nil {
			return ""
		}
		return payload
	})
	if err != nil {
		_ = global.Delete(appBridgeGlobal)
		return
	}

	prepareSub, err = bridge.SetFunction("prepare", func(args ...interop.Value) any {
		Prepare()
		return nil
	})
	if err != nil {
		exportSub.Cancel()
		_ = global.Delete(appBridgeGlobal)
		return
	}

	importSub, err = bridge.SetFunction("restoreSnapshot", func(args ...interop.Value) any {
		if len(args) == 0 || !args[0].Present() {
			result := hotReloadRestoreResult{Outcome: "skipped-empty", Message: "No hot reload snapshot was available to restore."}
			lastRestoreResult = result
			return map[string]any{
				"outcome": result.Outcome,
				"message": result.Message,
			}
		}
		result, err := importSnapshot(args[0].String())
		response := map[string]any{
			"outcome": result.Outcome,
			"message": result.Message,
		}
		if len(result.Diagnostics) > 0 {
			response["diagnostics"] = result.Diagnostics
		}
		if err != nil {
			response["error"] = err.Error()
		}
		return response
	})
	if err != nil {
		exportSub.Cancel()
		prepareSub.Cancel()
		_ = global.Delete(appBridgeGlobal)
		return
	}

	getLastRestoreResultSub, err = bridge.SetFunction("getLastRestoreResult", func(args ...interop.Value) any {
		return map[string]any{
			"outcome":     lastRestoreResult.Outcome,
			"message":     lastRestoreResult.Message,
			"diagnostics": lastRestoreResult.Diagnostics,
		}
	})
	if err != nil {
		exportSub.Cancel()
		prepareSub.Cancel()
		importSub.Cancel()
		_ = global.Delete(appBridgeGlobal)
		return
	}

	getDiagnosticsSub, err = bridge.SetFunction("getHotReloadDiagnostics", func(args ...interop.Value) any {
		return hotReloadDiagnostics()
	})
	if err != nil {
		exportSub.Cancel()
		prepareSub.Cancel()
		importSub.Cancel()
		getLastRestoreResultSub.Cancel()
		_ = global.Delete(appBridgeGlobal)
		return
	}

	getActivitySub, err = bridge.SetFunction("getHotReloadActivity", func(args ...interop.Value) any {
		return hotReloadActivity()
	})
	if err != nil {
		exportSub.Cancel()
		prepareSub.Cancel()
		importSub.Cancel()
		getLastRestoreResultSub.Cancel()
		getDiagnosticsSub.Cancel()
		_ = global.Delete(appBridgeGlobal)
		return
	}

	_ = bridge.Set("enabled", true)
	if config.ResetKey != "" {
		_ = bridge.Set("resetKey", config.ResetKey)
	}
	if len(config.AtomIDs) > 0 {
		_ = bridge.Set("atomIDs", append([]string(nil), config.AtomIDs...))
	}
	bridgeInstalled = true
}

func snapshotResetKeyMatches(snapshotResetKey string) bool {
	currentResetKey := strings.TrimSpace(currentConfig.ResetKey)
	snapshotResetKey = strings.TrimSpace(snapshotResetKey)
	if currentResetKey == "" && snapshotResetKey == "" {
		return true
	}
	return currentResetKey == snapshotResetKey
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

	global, err := interop.GlobalThis()
	if err != nil {
		return
	}
	_ = global.Delete(appBridgeGlobal)
}

func restorePendingSnapshot() {
	global, err := interop.GlobalThis()
	if err != nil {
		return
	}

	liveReload := global.Get(liveReloadGlobal)
	if !liveReload.Present() {
		return
	}
	getStoredState := liveReload.Get("getStoredState")
	if !getStoredState.Present() {
		return
	}

	saved, err := getStoredState.Invoke()
	if err != nil || !saved.Present() || strings.TrimSpace(saved.String()) == "" {
		return
	}

	if err := ImportSnapshot(saved.String()); err != nil {
		return
	}
	clearStoredState := liveReload.Get("clearStoredState")
	if clearStoredState.Present() {
		_, _ = clearStoredState.Invoke()
	}
}

func hotReloadDiagnostics() []hotReloadDiagnostic {
	diagnostics := runtimepkg.GetDiagnostics()
	filtered := make([]hotReloadDiagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if !strings.Contains(strings.ToLower(diagnostic.Message), "hot reload") {
			continue
		}
		filtered = append(filtered, hotReloadDiagnostic{
			Severity:       string(diagnostic.Severity),
			Classification: string(diagnostic.Classification),
			Message:        diagnostic.Message,
			Path:           diagnostic.Path,
			ComponentStack: append([]string(nil), diagnostic.ComponentStack...),
		})
	}
	lastRestoreResult.Diagnostics = filtered
	return filtered
}

func hotReloadActivity() []hotReloadActivityEntry {
	logs := runtimepkg.GetLogs()
	filtered := make([]hotReloadActivityEntry, 0, len(logs))
	for _, entry := range logs {
		domain := strings.TrimSpace(entry.Domain)
		message := strings.TrimSpace(entry.Message)
		if domain == "" || message == "" {
			continue
		}
		include := domain == "hotreload"
		if !include && domain == "router" {
			lower := strings.ToLower(message)
			include = strings.Contains(lower, "route loader") || strings.Contains(lower, "before-enter") || strings.Contains(lower, "before-leave") || strings.Contains(lower, "navigation")
		}
		if !include {
			continue
		}
		filtered = append(filtered, hotReloadActivityEntry{
			Domain:         domain,
			Level:          string(entry.Level),
			Classification: string(entry.Classification),
			Message:        message,
			Timestamp:      entry.Timestamp,
			Fields:         entry.Fields,
		})
	}
	if len(filtered) > 6 {
		filtered = append([]hotReloadActivityEntry(nil), filtered[len(filtered)-6:]...)
	}
	return filtered
}

func appBridge() (interop.Value, error) {
	global, err := interop.GlobalThis()
	if err != nil {
		return interop.Value{}, err
	}
	bridge := global.Get(appBridgeGlobal)
	if !bridge.Present() {
		return interop.Value{}, errors.New("hot reload bridge not installed")
	}
	return bridge, nil
}
