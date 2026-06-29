//go:build js && wasm && !production

package hotreload

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/v4/deprecation"
	runtimepkg "github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/interop"
	"github.com/monstercameron/GoWebComponents/v4/state"
)

const appBridgeGlobal = "GoWebComponentsHotReloadApp"
const liveReloadGlobal = "GoLiveReload"

const (
	hotReloadSnapshotProtocol       = "gwc.hotreload.snapshot"
	currentHotReloadSnapshotVersion = 1
	defaultHotReloadSchemaVersion   = 1
)

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
	Protocol          string                                  `json:"protocol,omitempty"`
	Version           int                                     `json:"version,omitempty"`
	SnapshotVersion   int                                     `json:"snapshotVersion,omitempty"`
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
		currentConfig = parseNormalized
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

// IsEnabled reports whether the public hot reload bridge is enabled.
//
// Deprecated: use Enabled.
func IsEnabled() bool {
	deprecation.Warn("hotreload.IsEnabled", "hotreload.Enabled")
	return Enabled()
}

// GetSnapshot returns the current serializable hot reload snapshot payload.
func GetSnapshot() (string, error) {
	parseStateSnapshot, parseErr := state.GetSnapshot()
	if parseErr != nil {
		return "", parseErr
	}
	parseSnapshot := bridgeSnapshot{
		Protocol:        hotReloadSnapshotProtocol,
		Version:         currentHotReloadSnapshotVersion,
		SnapshotVersion: currentConfig.SnapshotVersion,
		ResetKey:        currentConfig.ResetKey,
		State:           parseStateSnapshot.Select(currentConfig.AtomIDs...),
		Components:      runtimepkg.GetGlobalRuntime().CaptureHotReloadSnapshot().Components,
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
	if parseErr := json.Unmarshal([]byte(parsePayload), &parseSnapshot); parseErr == nil {
		parseNormalized, parseStructured, parseVersionErr := normalizeBridgeSnapshot(parseSnapshot)
		if parseVersionErr != nil {
			parseResult2 := hotReloadRestoreResult{Outcome: "error", Message: parseVersionErr.Error()}
			lastRestoreResult = parseResult2
			runtimepkg.ReportDiagnostic("hotreload", runtimepkg.DiagnosticError, "hot reload snapshot protocol mismatch: "+parseVersionErr.Error())
			return parseResult2, parseVersionErr
		}
		if !parseStructured {
			goto legacySnapshot
		}
		parseSnapshot = parseNormalized
		if !snapshotResetKeyMatches(parseSnapshot.ResetKey) {
			parseResult2 := hotReloadRestoreResult{Outcome: "skipped-reset-key", Message: "Hot reload snapshot was discarded because ResetKey changed."}
			lastRestoreResult = parseResult2
			return parseResult2, nil
		}
		parseMigrated, parseMigrationErr := migrateBridgeSnapshot(parseSnapshot)
		if parseMigrationErr != nil {
			parseResultMigration := hotReloadRestoreResult{Outcome: "error", Message: parseMigrationErr.Error()}
			lastRestoreResult = parseResultMigration
			runtimepkg.ReportDiagnostic("hotreload", runtimepkg.DiagnosticError, "hot reload snapshot migration failed: "+parseMigrationErr.Error())
			return parseResultMigration, parseMigrationErr
		}
		parseSnapshot = parseMigrated
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

legacySnapshot:
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

func normalizeBridgeSnapshot(parseSnapshot bridgeSnapshot) (bridgeSnapshot, bool, error) {
	parseProtocol := strings.TrimSpace(parseSnapshot.Protocol)
	parseStructured := parseProtocol != "" ||
		parseSnapshot.Version != 0 ||
		parseSnapshot.SnapshotVersion != 0 ||
		strings.TrimSpace(parseSnapshot.ResetKey) != "" ||
		strings.TrimSpace(parseSnapshot.RestoreMode) != "" ||
		len(parseSnapshot.ChangedComponents) > 0 ||
		parseSnapshot.State != nil ||
		len(parseSnapshot.Components) > 0
	if !parseStructured {
		return parseSnapshot, false, nil
	}
	if parseProtocol != "" && parseProtocol != hotReloadSnapshotProtocol {
		return parseSnapshot, true, fmt.Errorf("hotreload: unsupported snapshot protocol %q", parseProtocol)
	}
	parseVersion, parseErr := normalizeBridgeSnapshotVersion(parseSnapshot.Version)
	if parseErr != nil {
		return parseSnapshot, true, parseErr
	}
	parseSnapshot.Protocol = hotReloadSnapshotProtocol
	parseSnapshot.Version = parseVersion
	parseSchemaVersion, parseSchemaErr := normalizeBridgeSchemaVersion(parseSnapshot.SnapshotVersion)
	if parseSchemaErr != nil {
		return parseSnapshot, true, parseSchemaErr
	}
	parseSnapshot.SnapshotVersion = parseSchemaVersion
	return parseSnapshot, true, nil
}

func normalizeBridgeSnapshotVersion(parseVersion int) (int, error) {
	if parseVersion < 0 {
		return 0, fmt.Errorf("hotreload: unsupported snapshot version %d", parseVersion)
	}
	if parseVersion == 0 {
		return currentHotReloadSnapshotVersion, nil
	}
	if parseVersion > currentHotReloadSnapshotVersion {
		return 0, fmt.Errorf("hotreload: unsupported snapshot version %d", parseVersion)
	}
	return parseVersion, nil
}

func normalizeBridgeSchemaVersion(parseVersion int) (int, error) {
	if parseVersion < 0 {
		return 0, fmt.Errorf("hotreload: unsupported snapshot schema version %d", parseVersion)
	}
	if parseVersion == 0 {
		return defaultHotReloadSchemaVersion, nil
	}
	return parseVersion, nil
}

// Prepare runs cleanup needed before the current runtime instance is replaced.
func Prepare() {
	runtimepkg.GetGlobalRuntime().PrepareForHotReload()
}

func normalizeConfig(parseConfig Config) Config {
	parseNormalized := Config{
		ResetKey:        strings.TrimSpace(parseConfig.ResetKey),
		SnapshotVersion: normalizeConfigSchemaVersion(parseConfig.SnapshotVersion),
	}
	if len(parseConfig.AtomIDs) > 0 {
		parseNormalized.AtomIDs = append([]string(nil), parseConfig.AtomIDs...)
	}
	if len(parseConfig.SnapshotMigrations) > 0 {
		parseNormalized.SnapshotMigrations = make([]SnapshotMigration, 0, len(parseConfig.SnapshotMigrations))
		for _, parseMigration := range parseConfig.SnapshotMigrations {
			parseNormalized.SnapshotMigrations = append(parseNormalized.SnapshotMigrations, cloneSnapshotMigration(parseMigration))
		}
	}
	return parseNormalized
}

func configEqual(parseLeft, parseRight Config) bool {
	if parseLeft.ResetKey != parseRight.ResetKey {
		return false
	}
	if parseLeft.SnapshotVersion != parseRight.SnapshotVersion {
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

func normalizeConfigSchemaVersion(parseVersion int) int {
	if parseVersion <= 0 {
		return defaultHotReloadSchemaVersion
	}
	return parseVersion
}

func cloneSnapshotMigration(parseMigration SnapshotMigration) SnapshotMigration {
	parseClone := parseMigration
	parseClone.FromVersion = normalizeConfigSchemaVersion(parseMigration.FromVersion)
	if len(parseMigration.ComponentPathAliases) > 0 {
		parseClone.ComponentPathAliases = make(map[string]string, len(parseMigration.ComponentPathAliases))
		for parseFrom, parseTo := range parseMigration.ComponentPathAliases {
			parseClone.ComponentPathAliases[strings.TrimSpace(parseFrom)] = strings.TrimSpace(parseTo)
		}
	}
	if len(parseMigration.ComponentIdentityAliases) > 0 {
		parseClone.ComponentIdentityAliases = make(map[string]string, len(parseMigration.ComponentIdentityAliases))
		for parseFrom, parseTo := range parseMigration.ComponentIdentityAliases {
			parseClone.ComponentIdentityAliases[strings.TrimSpace(parseFrom)] = strings.TrimSpace(parseTo)
		}
	}
	return parseClone
}

func migrateBridgeSnapshot(parseSnapshot bridgeSnapshot) (bridgeSnapshot, error) {
	parseFromVersion := parseSnapshot.SnapshotVersion
	if parseFromVersion <= 0 {
		parseFromVersion = defaultHotReloadSchemaVersion
	}
	parseTargetVersion := normalizeConfigSchemaVersion(currentConfig.SnapshotVersion)
	if parseFromVersion == parseTargetVersion {
		parseSnapshot.SnapshotVersion = parseTargetVersion
		return parseSnapshot, nil
	}
	if parseFromVersion > parseTargetVersion {
		return parseSnapshot, fmt.Errorf("hotreload: snapshot schema version %d is newer than configured version %d", parseFromVersion, parseTargetVersion)
	}

	parseCurrent := parseSnapshot
	for parseCurrent.SnapshotVersion < parseTargetVersion {
		parseMigration, parseFound := nextSnapshotMigration(parseCurrent.SnapshotVersion, parseTargetVersion, currentConfig.SnapshotMigrations)
		if !parseFound {
			return parseSnapshot, fmt.Errorf("hotreload: no snapshot migration from version %d to %d", parseCurrent.SnapshotVersion, parseTargetVersion)
		}
		parseMigrated, parseErr := applySnapshotMigration(parseCurrent, parseMigration)
		if parseErr != nil {
			return parseSnapshot, parseErr
		}
		parseCurrent = parseMigrated
	}
	return parseCurrent, nil
}

func nextSnapshotMigration(parseFromVersion int, parseTargetVersion int, parseMigrations []SnapshotMigration) (SnapshotMigration, bool) {
	var parseBest SnapshotMigration
	parseBestVersion := 0
	for _, parseMigration := range parseMigrations {
		parseMigrationFrom := normalizeConfigSchemaVersion(parseMigration.FromVersion)
		parseMigrationTo := parseMigration.ToVersion
		if parseMigrationFrom != parseFromVersion || parseMigrationTo <= parseMigrationFrom || parseMigrationTo > parseTargetVersion {
			continue
		}
		if parseMigrationTo == parseTargetVersion {
			return parseMigration, true
		}
		if parseBestVersion == 0 || parseMigrationTo < parseBestVersion {
			parseBest = parseMigration
			parseBestVersion = parseMigrationTo
		}
	}
	return parseBest, parseBestVersion != 0
}

func applySnapshotMigration(parseSnapshot bridgeSnapshot, parseMigration SnapshotMigration) (bridgeSnapshot, error) {
	parseFromVersion := normalizeConfigSchemaVersion(parseMigration.FromVersion)
	parseToVersion := parseMigration.ToVersion
	if parseToVersion <= parseFromVersion {
		return parseSnapshot, fmt.Errorf("hotreload: invalid snapshot migration from version %d to %d", parseFromVersion, parseToVersion)
	}

	if parseMigration.MigrateState != nil {
		parseState, parseErr := parseMigration.MigrateState(SnapshotMigrationContext{
			FromVersion: parseFromVersion,
			ToVersion:   parseToVersion,
			State:       cloneStateSnapshot(parseSnapshot.State),
		})
		if parseErr != nil {
			return parseSnapshot, fmt.Errorf("hotreload: snapshot state migration from version %d to %d failed: %w", parseFromVersion, parseToVersion, parseErr)
		}
		if parseState == nil {
			parseState = state.Snapshot{}
		}
		parseSnapshot.State = parseState
	}

	applyComponentPathAliases(parseSnapshot.Components, parseMigration.ComponentPathAliases)
	applyComponentIdentityAliases(parseSnapshot.Components, parseMigration.ComponentIdentityAliases)
	parseSnapshot.SnapshotVersion = parseToVersion
	return parseSnapshot, nil
}

func cloneStateSnapshot(parseSnapshot state.Snapshot) state.Snapshot {
	if parseSnapshot == nil {
		return state.Snapshot{}
	}
	parseClone := make(state.Snapshot, len(parseSnapshot))
	for parseKey, parseValue := range parseSnapshot {
		parseClone[parseKey] = parseValue
	}
	return parseClone
}

func applyComponentPathAliases(parseComponents []runtimepkg.HotReloadComponentSnapshot, parseAliases map[string]string) {
	if len(parseAliases) == 0 {
		return
	}
	for parseIndex := range parseComponents {
		parsePath := strings.TrimSpace(parseComponents[parseIndex].Path)
		if parseReplacement := strings.TrimSpace(parseAliases[parsePath]); parseReplacement != "" {
			parseComponents[parseIndex].Path = parseReplacement
		}
	}
}

func applyComponentIdentityAliases(parseComponents []runtimepkg.HotReloadComponentSnapshot, parseAliases map[string]string) {
	if len(parseAliases) == 0 {
		return
	}
	for parseIndex := range parseComponents {
		for parseTrailIndex, parseIdentity := range parseComponents[parseIndex].IdentityTrail {
			if parseReplacement := strings.TrimSpace(parseAliases[strings.TrimSpace(parseIdentity)]); parseReplacement != "" {
				parseComponents[parseIndex].IdentityTrail[parseTrailIndex] = parseReplacement
			}
		}
		if parseReplacement := strings.TrimSpace(parseAliases[strings.TrimSpace(parseComponents[parseIndex].Signature.QualifiedName)]); parseReplacement != "" {
			parseComponents[parseIndex].Signature.QualifiedName = parseReplacement
		}
		if parseReplacement := strings.TrimSpace(parseAliases[strings.TrimSpace(parseComponents[parseIndex].Signature.Name)]); parseReplacement != "" {
			parseComponents[parseIndex].Signature.Name = parseReplacement
		}
	}
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

	parseRegisterBridgeFunction := func(parseName string, parseHandler func(...interop.Value) any, parseSubscription *interop.Subscription) bool {
		parseRegistered, parseSetErr := parseBridge.SetFunction(parseName, parseHandler)
		if parseSetErr != nil {
			uninstallBridge()
			return false
		}
		*parseSubscription = parseRegistered
		return true
	}

	if !parseRegisterBridgeFunction("captureSnapshot", func(parseArgs ...interop.Value) any {
		parsePayload, parseErr3 := GetSnapshot()
		if parseErr3 != nil {
			return ""
		}
		return parsePayload
	}, &exportSub) {
		return
	}

	if !parseRegisterBridgeFunction("prepare", func(parseArgs2 ...interop.Value) any {
		Prepare()
		return nil
	}, &prepareSub) {
		return
	}

	if !parseRegisterBridgeFunction("restoreSnapshot", func(parseArgs3 ...interop.Value) any {
		if len(parseArgs3) == 0 || !parseArgs3[0].Present() {
			parseResult := hotReloadRestoreResult{Outcome: "skipped-empty", Message: "No hot reload snapshot was available to restore."}
			lastRestoreResult = parseResult
			return map[string]any{
				"outcome": parseResult.Outcome,
				"message": parseResult.Message,
			}
		}
		parseResult2, parseErr4 := importSnapshot(parseArgs3[0].String())
		if strings.HasPrefix(parseResult2.Outcome, "restored") {
			parseResult2.Diagnostics = hotReloadDiagnostics()
		}
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
	}, &importSub) {
		return
	}

	if !parseRegisterBridgeFunction("getLastRestoreResult", func(parseArgs4 ...interop.Value) any {
		return map[string]any{
			"outcome":     lastRestoreResult.Outcome,
			"message":     lastRestoreResult.Message,
			"diagnostics": lastRestoreResult.Diagnostics,
		}
	}, &getLastRestoreResultSub) {
		return
	}

	if !parseRegisterBridgeFunction("getHotReloadDiagnostics", func(parseArgs5 ...interop.Value) any {
		return hotReloadDiagnostics()
	}, &getDiagnosticsSub) {
		return
	}

	if !parseRegisterBridgeFunction("getHotReloadActivity", func(parseArgs6 ...interop.Value) any {
		return hotReloadActivity()
	}, &getActivitySub) {
		return
	}

	_ = parseBridge.Set("enabled", true)
	_ = parseBridge.Set("protocol", hotReloadSnapshotProtocol)
	_ = parseBridge.Set("version", currentHotReloadSnapshotVersion)
	_ = parseBridge.Set("snapshotVersion", parseConfig.SnapshotVersion)
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
