//go:build js && wasm
// +build js,wasm

package fiber

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"reflect"
	"syscall/js"
	"time"
)

// StateExportVersion tracks the format version for backwards compatibility
const StateExportVersion = "1.0.0"

// StateSnapshot represents a complete snapshot of the application state
type StateSnapshot struct {
	Version     string                 `json:"version"`
	Timestamp   int64                  `json:"timestamp"`
	FiberStates []FiberStateSnapshot   `json:"fiberStates"`
	GlobalState map[string]interface{} `json:"globalState"`
	DOMState    DOMStateSnapshot       `json:"domState"`
	MemoryStats MemoryStatsSnapshot    `json:"memoryStats"`
}

// FiberStateSnapshot represents the state of a single fiber component
type FiberStateSnapshot struct {
	ID            string                 `json:"id"`
	ComponentType string                 `json:"componentType"`
	HookStates    []interface{}          `json:"hookStates"`
	EffectDeps    [][]interface{}        `json:"effectDeps"`
	MemoValues    []MemoSnapshot         `json:"memoValues"`
	Props         map[string]interface{} `json:"props"`
	Position      ComponentPosition      `json:"position"`
}

// MemoSnapshot represents a memoized value with its dependencies
type MemoSnapshot struct {
	Value interface{}   `json:"value"`
	Deps  []interface{} `json:"deps"`
}

// ComponentPosition helps identify component location in the tree
type ComponentPosition struct {
	Depth    int    `json:"depth"`
	Index    int    `json:"index"`
	ParentID string `json:"parentId"`
	Path     string `json:"path"` // e.g., "root.0.1.2"
}

// DOMStateSnapshot captures DOM-related state
type DOMStateSnapshot struct {
	ScrollPosition ScrollPosition `json:"scrollPosition"`
	FocusedElement string         `json:"focusedElement"`
	FormStates     []FormState    `json:"formStates"`
}

// ScrollPosition captures scroll state
type ScrollPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// FormState captures form input states
type FormState struct {
	ID    string `json:"id"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// MemoryStatsSnapshot captures memory usage stats
type MemoryStatsSnapshot struct {
	FiberPoolSize    int32   `json:"fiberPoolSize"`
	HooksPoolSize    int32   `json:"hooksPoolSize"`
	ElementPoolSize  int32   `json:"elementPoolSize"`
	TotalAllocations int64   `json:"totalAllocations"`
	PoolHitRate      float64 `json:"poolHitRate"`
}

// Global state storage for cross-component data
var globalAppState = make(map[string]interface{})

// Fiber ID tracking for state mapping
var fiberIDCounter int
var fiberIDMap = make(map[*Fiber]string)

// ExportAppState exports the complete application state as binary data to JavaScript
//
//go:export exportAppState
func ExportAppState() js.Value {
	// Only export state if hot reload is enabled
	if !IsHotReloadEnabled() {
		debugf("STATE", "⚠️ ExportAppState: Hot reload disabled, skipping state export\n")
		return js.ValueOf(map[string]interface{}{
			"disabled": true,
			"message":  "Hot reload is disabled",
		})
	}

	debugf("STATE", "🔄 ExportAppState: Starting binary state export...\n")

	snapshot := StateSnapshot{
		Version:     StateExportVersion,
		Timestamp:   time.Now().UnixMilli(),
		GlobalState: make(map[string]interface{}),
	}

	// Export global state
	for k, v := range globalAppState {
		snapshot.GlobalState[k] = v
	}

	// Export fiber states
	snapshot.FiberStates = exportFiberStates()

	// Export DOM state
	snapshot.DOMState = exportDOMState()

	// Export memory stats
	snapshot.MemoryStats = exportMemoryStats()

	// Register types for gob encoding
	registerGobTypes()

	// Encode to binary using gob
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(snapshot)
	if err != nil {
		debugf("STATE", "🚨 ExportAppState: Failed to encode state: %v\n", err)
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Failed to export state: %v", err),
		})
	}

	// Convert to base64 for JavaScript transfer
	binaryData := buf.Bytes()
	base64Data := base64.StdEncoding.EncodeToString(binaryData)

	debugf("STATE", "✅ ExportAppState: Exported %d bytes of binary state data (base64: %d chars)\n",
		len(binaryData), len(base64Data))

	// Return as JavaScript object with metadata
	return js.ValueOf(map[string]interface{}{
		"version":   StateExportVersion,
		"timestamp": time.Now().UnixMilli(),
		"format":    "gob_base64",
		"data":      base64Data,
		"size":      len(binaryData),
	})
}

// ImportAppState imports and restores application state from binary JavaScript data
//
//go:export importAppState
func ImportAppState(jsState js.Value) {
	// Only import state if hot reload is enabled
	if !IsHotReloadEnabled() {
		debugf("STATE", "⚠️ ImportAppState: Hot reload disabled, skipping state import\n")
		return
	}

	debugf("STATE", "🔄 ImportAppState: Starting binary state import...\n")

	if jsState.IsNull() || jsState.IsUndefined() {
		debugf("STATE", "⚠️ ImportAppState: No state provided\n")
		return
	}

	// Extract metadata and data from the JS object
	format := jsState.Get("format").String()
	if format != "gob_base64" {
		debugf("STATE", "🚨 ImportAppState: Unsupported format: %s\n", format)
		return
	}

	version := jsState.Get("version").String()
	if version != StateExportVersion {
		debugf("STATE", "⚠️ ImportAppState: Version mismatch. Expected %s, got %s\n",
			StateExportVersion, version)
		// Continue anyway - might be compatible
	}

	// Decode base64 data
	base64Data := jsState.Get("data").String()
	binaryData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		debugf("STATE", "🚨 ImportAppState: Failed to decode base64 data: %v\n", err)
		return
	}

	// Register types for gob decoding
	registerGobTypes()

	// Decode binary data using gob
	buf := bytes.NewReader(binaryData)
	decoder := gob.NewDecoder(buf)

	var snapshot StateSnapshot
	if err := decoder.Decode(&snapshot); err != nil {
		debugf("STATE", "🚨 ImportAppState: Failed to decode state: %v\n", err)
		return
	}

	debugf("STATE", "✅ ImportAppState: Decoded %d bytes of binary state data\n", len(binaryData))

	// Import global state
	for k, v := range snapshot.GlobalState {
		globalAppState[k] = v
	}

	// Import fiber states
	importFiberStates(snapshot.FiberStates)

	// Import DOM state
	importDOMState(snapshot.DOMState)

	debugf("STATE", "✅ ImportAppState: Binary state import completed\n")

	// Schedule a re-render to apply the imported state
	scheduleUpdateAtRoot()
}

// exportFiberStates exports all fiber component states
func exportFiberStates() []FiberStateSnapshot {
	var states []FiberStateSnapshot

	if currentRoot == nil {
		debugf("STATE", "⚠️ exportFiberStates: No current root fiber\n")
		return states
	}

	// Walk the fiber tree and export states
	walkFiberTree(currentRoot, func(fiber *Fiber, position ComponentPosition) {
		if fiber.hooks != nil && len(fiber.hooks.state) > 0 {
			fiberID := getFiberID(fiber)

			snapshot := FiberStateSnapshot{
				ID:            fiberID,
				ComponentType: getComponentTypeName(fiber.typeOf),
				HookStates:    make([]interface{}, len(fiber.hooks.state)),
				EffectDeps:    make([][]interface{}, len(fiber.hooks.deps)),
				MemoValues:    exportMemoValues(fiber.hooks.memos),
				Props:         copyProps(fiber.props),
				Position:      position,
			}

			// Copy hook states (deep copy to avoid references)
			copy(snapshot.HookStates, fiber.hooks.state)

			// Copy effect dependencies
			for i, deps := range fiber.hooks.deps {
				if deps != nil {
					snapshot.EffectDeps[i] = make([]interface{}, len(deps))
					copy(snapshot.EffectDeps[i], deps)
				}
			}

			states = append(states, snapshot)
			debugf("STATE", "📊 Exported fiber state: %s (%d hooks)\n",
				fiberID, len(fiber.hooks.state))
		}
	})

	return states
}

// importFiberStates imports and restores fiber component states
func importFiberStates(states []FiberStateSnapshot) {
	debugf("STATE", "🔄 importFiberStates: Importing %d fiber states\n", len(states))

	if currentRoot == nil {
		debugf("STATE", "⚠️ importFiberStates: No current root fiber\n")
		return
	}

	// Create a map of state snapshots by component type and position
	stateMap := make(map[string]FiberStateSnapshot)
	for _, state := range states {
		key := fmt.Sprintf("%s_%s", state.ComponentType, state.Position.Path)
		stateMap[key] = state
	}

	// Walk the current fiber tree and restore states
	walkFiberTree(currentRoot, func(fiber *Fiber, position ComponentPosition) {
		componentType := getComponentTypeName(fiber.typeOf)
		key := fmt.Sprintf("%s_%s", componentType, position.Path)

		if snapshot, exists := stateMap[key]; exists {
			debugf("STATE", "🔄 Restoring state for fiber: %s\n", key)

			// Initialize hooks if needed
			if fiber.hooks == nil {
				fiber.hooks = getHooksFromPool()
			}

			// Restore hook states
			if len(snapshot.HookStates) > 0 {
				// Ensure capacity
				if cap(fiber.hooks.state) < len(snapshot.HookStates) {
					fiber.hooks.state = make([]interface{}, len(snapshot.HookStates))
				} else {
					fiber.hooks.state = fiber.hooks.state[:len(snapshot.HookStates)]
				}
				copy(fiber.hooks.state, snapshot.HookStates)
			}

			// Restore effect dependencies
			if len(snapshot.EffectDeps) > 0 {
				fiber.hooks.deps = make([][]interface{}, len(snapshot.EffectDeps))
				for i, deps := range snapshot.EffectDeps {
					if deps != nil {
						fiber.hooks.deps[i] = make([]interface{}, len(deps))
						copy(fiber.hooks.deps[i], deps)
					}
				}
			}

			// Restore memoized values
			if len(snapshot.MemoValues) > 0 {
				fiber.hooks.memos = make([]memoizedValue, len(snapshot.MemoValues))
				for i, memo := range snapshot.MemoValues {
					fiber.hooks.memos[i] = memoizedValue{
						value: memo.Value,
						deps:  memo.Deps,
					}
				}
			}

			debugf("STATE", "✅ Restored fiber state: %s (%d hooks)\n",
				key, len(snapshot.HookStates))
		}
	})
}

// exportDOMState exports DOM-related state
func exportDOMState() DOMStateSnapshot {
	global := js.Global()
	window := global.Get("window")
	document := global.Get("document")

	domState := DOMStateSnapshot{
		ScrollPosition: ScrollPosition{
			X: window.Get("scrollX").Int(),
			Y: window.Get("scrollY").Int(),
		},
		FormStates: []FormState{},
	}

	// Get focused element
	activeElement := document.Get("activeElement")
	if !activeElement.IsNull() {
		id := activeElement.Get("id").String()
		if id != "" {
			domState.FocusedElement = id
		}
	}

	// Export form states
	inputs := document.Call("querySelectorAll", "input, textarea, select")
	length := inputs.Get("length").Int()

	for i := 0; i < length; i++ {
		input := inputs.Call("item", i)
		id := input.Get("id").String()
		if id != "" {
			formState := FormState{
				ID:    id,
				Value: input.Get("value").String(),
				Type:  input.Get("type").String(),
			}
			domState.FormStates = append(domState.FormStates, formState)
		}
	}

	return domState
}

// importDOMState imports and restores DOM-related state
func importDOMState(domState DOMStateSnapshot) {
	global := js.Global()
	window := global.Get("window")
	document := global.Get("document")

	// Restore scroll position
	window.Call("scrollTo", domState.ScrollPosition.X, domState.ScrollPosition.Y)

	// Restore form states
	for _, formState := range domState.FormStates {
		element := document.Call("getElementById", formState.ID)
		if !element.IsNull() {
			element.Set("value", formState.Value)
		}
	}

	// Restore focus
	if domState.FocusedElement != "" {
		element := document.Call("getElementById", domState.FocusedElement)
		if !element.IsNull() {
			element.Call("focus")
		}
	}
}

// exportMemoryStats exports current memory statistics
func exportMemoryStats() MemoryStatsSnapshot {
	return MemoryStatsSnapshot{
		FiberPoolSize:    poolSizes.fiber,
		HooksPoolSize:    poolSizes.hooks,
		ElementPoolSize:  poolSizes.element,
		TotalAllocations: poolUtilization.totalAllocations,
		PoolHitRate:      poolUtilization.poolHitRate,
	}
}

// Helper functions

// walkFiberTree walks the fiber tree and calls the callback for each fiber
func walkFiberTree(fiber *Fiber, callback func(*Fiber, ComponentPosition)) {
	walkFiberTreeHelper(fiber, callback, ComponentPosition{
		Depth:    0,
		Index:    0,
		ParentID: "",
		Path:     "root",
	})
}

func walkFiberTreeHelper(fiber *Fiber, callback func(*Fiber, ComponentPosition), position ComponentPosition) {
	if fiber == nil {
		return
	}

	callback(fiber, position)

	// Walk children
	if fiber.child != nil {
		childPosition := ComponentPosition{
			Depth:    position.Depth + 1,
			Index:    0,
			ParentID: getFiberID(fiber),
			Path:     position.Path + ".0",
		}
		walkFiberTreeHelper(fiber.child, callback, childPosition)
	}

	// Walk siblings
	if fiber.sibling != nil {
		siblingPosition := ComponentPosition{
			Depth:    position.Depth,
			Index:    position.Index + 1,
			ParentID: position.ParentID,
			Path:     fmt.Sprintf("%s.%d", position.Path[:len(position.Path)-1], position.Index+1),
		}
		walkFiberTreeHelper(fiber.sibling, callback, siblingPosition)
	}
}

// getFiberID gets or creates a unique ID for a fiber
func getFiberID(fiber *Fiber) string {
	if id, exists := fiberIDMap[fiber]; exists {
		return id
	}

	fiberIDCounter++
	id := fmt.Sprintf("fiber_%d", fiberIDCounter)
	fiberIDMap[fiber] = id
	return id
}

// getComponentTypeName gets a string representation of the component type
func getComponentTypeName(typeOf interface{}) string {
	if typeOf == nil {
		return "unknown"
	}

	switch t := typeOf.(type) {
	case string:
		return t
	case func(map[string]interface{}) *Element:
		return fmt.Sprintf("func_%p", t)
	default:
		return reflect.TypeOf(typeOf).String()
	}
}

// copyProps creates a deep copy of props map
func copyProps(props map[string]interface{}) map[string]interface{} {
	if props == nil {
		return nil
	}

	copied := make(map[string]interface{}, len(props))
	for k, v := range props {
		copied[k] = v
	}
	return copied
}

// exportMemoValues exports memoized values
func exportMemoValues(memos []memoizedValue) []MemoSnapshot {
	snapshots := make([]MemoSnapshot, len(memos))
	for i, memo := range memos {
		snapshots[i] = MemoSnapshot{
			Value: memo.value,
			Deps:  memo.deps,
		}
	}
	return snapshots
}

// SetGlobalState sets a global state value that persists across hot reloads
func SetGlobalState(key string, value interface{}) {
	globalAppState[key] = value
	debugf("STATE", "🌐 SetGlobalState: %s = %+v\n", key, value)
}

// GetGlobalState gets a global state value
func GetGlobalState(key string) (interface{}, bool) {
	value, exists := globalAppState[key]
	debugf("STATE", "🌐 GetGlobalState: %s = %+v (exists: %v)\n", key, value, exists)
	return value, exists
}

// ClearGlobalState clears all global state
func ClearGlobalState() {
	globalAppState = make(map[string]interface{})
	debugf("STATE", "🌐 ClearGlobalState: All global state cleared\n")
}

// ExportStateSnapshot exports the state to a binary format that can be saved to disk
func ExportStateSnapshot() ([]byte, error) {
	debugf("STATE", "📤 ExportStateSnapshot: Creating binary snapshot...\n")

	snapshot := StateSnapshot{
		Version:     StateExportVersion,
		Timestamp:   time.Now().UnixMilli(),
		GlobalState: make(map[string]interface{}),
	}

	// Export global state
	for k, v := range globalAppState {
		snapshot.GlobalState[k] = v
	}

	// Export fiber states
	snapshot.FiberStates = exportFiberStates()

	// Export DOM state
	snapshot.DOMState = exportDOMState()

	// Export memory stats
	snapshot.MemoryStats = exportMemoryStats()

	// Register types for gob encoding
	registerGobTypes()

	// Encode to binary using gob
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(snapshot)
	if err != nil {
		debugf("STATE", "🚨 ExportStateSnapshot: Failed to encode snapshot: %v\n", err)
		return nil, fmt.Errorf("failed to encode state snapshot: %w", err)
	}

	binaryData := buf.Bytes()
	debugf("STATE", "✅ ExportStateSnapshot: Created %d bytes binary snapshot\n", len(binaryData))

	return binaryData, nil
}

// ImportStateSnapshot imports state from a binary format (e.g., loaded from disk)
func ImportStateSnapshot(data []byte) error {
	debugf("STATE", "📥 ImportStateSnapshot: Loading binary snapshot (%d bytes)...\n", len(data))

	if len(data) == 0 {
		return fmt.Errorf("no data provided")
	}

	// Register types for gob decoding
	registerGobTypes()

	// Decode binary data using gob
	buf := bytes.NewReader(data)
	decoder := gob.NewDecoder(buf)

	var snapshot StateSnapshot
	if err := decoder.Decode(&snapshot); err != nil {
		debugf("STATE", "🚨 ImportStateSnapshot: Failed to decode snapshot: %v\n", err)
		return fmt.Errorf("failed to decode state snapshot: %w", err)
	}

	// Validate version compatibility
	if snapshot.Version != StateExportVersion {
		debugf("STATE", "⚠️ ImportStateSnapshot: Version mismatch. Expected %s, got %s\n",
			StateExportVersion, snapshot.Version)
		// Continue anyway - might be compatible
	}

	// Import global state
	for k, v := range snapshot.GlobalState {
		globalAppState[k] = v
	}

	// Import fiber states
	importFiberStates(snapshot.FiberStates)

	// Import DOM state
	importDOMState(snapshot.DOMState)

	debugf("STATE", "✅ ImportStateSnapshot: Binary snapshot import completed\n")

	// Schedule a re-render to apply the imported state
	scheduleUpdateAtRoot()

	return nil
}

// SaveStateToFile saves the current state to a binary file (for debugging/development)
func SaveStateToFile(filename string) error {
	data, err := ExportStateSnapshot()
	if err != nil {
		return err
	}

	// Note: In WASM, we can't directly write files, but this function signature
	// allows for future file system API integration or download functionality
	debugf("STATE", "💾 SaveStateToFile: Would save %d bytes to %s\n", len(data), filename)

	// For now, just log the base64 representation for manual saving
	base64Data := base64.StdEncoding.EncodeToString(data)
	debugf("STATE", "📋 SaveStateToFile: Base64 data (for manual saving):\n%s\n", base64Data)

	return nil
}

// LoadStateFromBase64 loads state from a base64 string (for debugging/development)
func LoadStateFromBase64(base64Data string) error {
	debugf("STATE", "📋 LoadStateFromBase64: Decoding base64 data...\n")

	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return fmt.Errorf("failed to decode base64 data: %w", err)
	}

	return ImportStateSnapshot(data)
}

// HotReloadWasm performs a hot reload of the WASM module while preserving state
//
//go:export hotReloadWasm
func HotReloadWasm() {
	debugf("STATE", "🔥 HotReloadWasm: Starting hot reload...\n")

	// Export current state
	stateJS := ExportAppState()

	// Store state in the live reload client script for persistence
	global := js.Global()
	if !global.Get("GoLiveReload").IsUndefined() {
		liveReload := global.Get("GoLiveReload")
		if !liveReload.Get("storeState").IsUndefined() {
			liveReload.Call("storeState", stateJS)
			debugf("STATE", "✅ HotReloadWasm: State saved to live reload client\n")
		} else {
			debugf("STATE", "⚠️ HotReloadWasm: Live reload client storeState not available\n")
		}
	} else {
		debugf("STATE", "⚠️ HotReloadWasm: Live reload client not available\n")
	}

	// Trigger page reload - the live reload script will handle state restoration
	global.Get("location").Call("reload")
}

// registerGobTypes registers all types used in state snapshots for gob encoding/decoding
func registerGobTypes() {
	// Register basic types that might be stored in state
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
	gob.Register(StateSnapshot{})
	gob.Register(FiberStateSnapshot{})
	gob.Register(MemoSnapshot{})
	gob.Register(ComponentPosition{})
	gob.Register(DOMStateSnapshot{})
	gob.Register(ScrollPosition{})
	gob.Register(FormState{})
	gob.Register(MemoryStatsSnapshot{})

	// Register common Go types that might appear in state
	gob.Register(int(0))
	gob.Register(int32(0))
	gob.Register(int64(0))
	gob.Register(float32(0))
	gob.Register(float64(0))
	gob.Register(bool(false))
	gob.Register(string(""))
	gob.Register([]string{})
	gob.Register([]int{})
	gob.Register([]float64{})
}

// RestoreStateFromStorage restores state from the live reload client script
func RestoreStateFromStorage() {
	// Only restore state if hot reload is enabled
	if !IsHotReloadEnabled() {
		debugf("STATE", "⚠️ RestoreStateFromStorage: Hot reload disabled, skipping state restoration\n")
		return
	}

	global := js.Global()

	// Check if the live reload client has stored state
	if !global.Get("GoLiveReload").IsUndefined() {
		liveReload := global.Get("GoLiveReload")
		if !liveReload.Get("getStoredState").IsUndefined() {
			storedState := liveReload.Call("getStoredState")
			if !storedState.IsNull() && !storedState.IsUndefined() {
				debugf("STATE", "ℹ️ RestoreStateFromStorage: Found state in live reload client\n")
				ImportAppState(storedState)

				// Clear the stored state
				liveReload.Call("clearStoredState")
				return
			}
		}
	}

	debugf("STATE", "ℹ️ RestoreStateFromStorage: No saved state found\n")
}
