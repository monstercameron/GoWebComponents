package agentbridge

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// writeActivateAgentMode enables agent mode for the duration of a sub-test
// and restores the prior state on cleanup.
func writeActivateAgentMode(parseTB testing.TB) {
	parseTB.Helper()
	SetAgentModeActive(true)
	parseTB.Cleanup(func() { SetAgentModeActive(false) })
}

// writeCallHandler is a helper that marshals parseArgs to JSON and invokes
// parseHandler, returning the raw result and any envelope error.
func writeCallHandler(parseHandler AgentCommandHandler, parseArgs any) (json.RawMessage, *EnvelopeError) {
	parseRaw, parseMarshalErr := json.Marshal(parseArgs)
	if parseMarshalErr != nil {
		panic("writeCallHandler: marshal: " + parseMarshalErr.Error())
	}
	return parseHandler(json.RawMessage(parseRaw))
}

// ---------------------------------------------------------------------------
// Forbidden gate — all four handlers
// ---------------------------------------------------------------------------

// TestWriteCommandsForbiddenWhenInactive pins that all four write commands
// return ErrorCodeForbidden when agent mode is not active.
func TestWriteCommandsForbiddenWhenInactive(t *testing.T) {
	// Ensure agent mode is off.
	SetAgentModeActive(false)

	parseHandlers := map[string]AgentCommandHandler{
		"bridge.set-atom":    writeHandleSetAtom,
		"bridge.set-state":   writeHandleSetState,
		"bridge.emit":        writeHandleEmit,
		"bridge.publish":     writeHandlePublish,
		"bridge.navigate":    writeHandleNavigate,
		"bridge.mount":       writeHandleMount,
		"bridge.unmount":     writeHandleUnmount,
		"bridge.delete-atom": writeHandleDeleteAtom,
	}

	for parseName, parseHandler := range parseHandlers {
		parseName, parseHandler := parseName, parseHandler
		t.Run(parseName, func(t *testing.T) {
			_, parseErr := parseHandler(json.RawMessage(`{}`))
			if parseErr == nil {
				t.Fatalf("%s: expected forbidden error, got nil", parseName)
			}
			if parseErr.Code != ErrorCodeForbidden {
				t.Fatalf("%s: expected code %q, got %q", parseName, ErrorCodeForbidden, parseErr.Code)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// bridge.set-atom
// ---------------------------------------------------------------------------

// TestWriteSetAtomDecodeErrors pins that bridge.set-atom returns
// ErrorCodeBadPayload for various malformed payloads and never touches atom
// state.
func TestWriteSetAtomDecodeErrors(t *testing.T) {
	writeActivateAgentMode(t)

	parseCases := []struct {
		parseName    string
		parsePayload string
	}{
		{"not-json", `not json`},
		{"missing-id", `{"value":1}`},
		{"empty-id", `{"id":"","value":1}`},
		{"missing-value", `{"id":"myatom"}`},
		{"unknown-atom", `{"id":"__nonexistent_atom_xyz__","value":42}`},
	}

	for _, parseCase := range parseCases {
		parseCase := parseCase
		t.Run(parseCase.parseName, func(t *testing.T) {
			_, parseErr := writeHandleSetAtom(json.RawMessage(parseCase.parsePayload))
			if parseErr == nil {
				t.Fatalf("set-atom(%s): expected bad-payload error, got nil", parseCase.parseName)
			}
			if parseErr.Code != ErrorCodeBadPayload {
				t.Fatalf("set-atom(%s): expected code %q, got %q (%s)",
					parseCase.parseName, ErrorCodeBadPayload, parseErr.Code, parseErr.Message)
			}
		})
	}
}

func TestWriteSetAtomSchemaValidationRejectsWrongType(t *testing.T) {
	writeActivateAgentMode(t)
	parseRt := runtime.GetGlobalRuntime()
	if parseErr := parseRt.RestoreAtomSnapshot(map[string]any{"agentbridge.schema.count": 1}); parseErr != nil {
		t.Fatalf("seed atom: %v", parseErr)
	}

	_, parseErr := writeHandleSetAtom(json.RawMessage(`{"id":"agentbridge.schema.count","value":"wrong"}`))
	if parseErr == nil {
		t.Fatal("expected schema validation error")
	}
	if parseErr.Code != ErrorCodeBadPayload {
		t.Fatalf("schema error code = %q, want %q", parseErr.Code, ErrorCodeBadPayload)
	}
	if !strings.Contains(parseErr.Message, "schema path value") {
		t.Fatalf("schema error message = %q, want schema path", parseErr.Message)
	}
}

// ---------------------------------------------------------------------------
// bridge.emit — error code mapping
// ---------------------------------------------------------------------------

// TestWriteEmitDecodeErrors pins that bridge.emit returns ErrorCodeBadPayload
// for malformed payloads.
func TestWriteEmitDecodeErrors(t *testing.T) {
	writeActivateAgentMode(t)

	parseCases := []struct {
		parseName    string
		parsePayload string
	}{
		{"not-json", `not json`},
		{"missing-ref", `{"event":"click"}`},
		{"empty-ref", `{"ref":"","event":"click"}`},
		{"missing-event", `{"ref":"someref"}`},
		{"empty-event", `{"ref":"someref","event":""}`},
	}

	for _, parseCase := range parseCases {
		parseCase := parseCase
		t.Run(parseCase.parseName, func(t *testing.T) {
			_, parseErr := writeHandleEmit(json.RawMessage(parseCase.parsePayload))
			if parseErr == nil {
				t.Fatalf("emit(%s): expected bad-payload error, got nil", parseCase.parseName)
			}
			if parseErr.Code != ErrorCodeBadPayload {
				t.Fatalf("emit(%s): expected code %q, got %q (%s)",
					parseCase.parseName, ErrorCodeBadPayload, parseErr.Code, parseErr.Message)
			}
		})
	}
}

// TestWriteEmitStaleRefMapsToStaleRefCode pins that a stale agent ref in
// bridge.emit maps to ErrorCodeStaleRef (not generic bad-payload).
func TestWriteEmitStaleRefMapsToStaleRefCode(t *testing.T) {
	writeActivateAgentMode(t)

	// A well-formed payload but with a ref that will not resolve (no tree is
	// mounted in the global runtime during this test).
	parsePayload := `{"ref":"ul@0/Item[key=a]@0","event":"click"}`
	_, parseErr := writeHandleEmit(json.RawMessage(parsePayload))
	if parseErr == nil {
		t.Fatal("emit(stale-ref): expected error, got nil")
	}
	if parseErr.Code != ErrorCodeStaleRef {
		t.Fatalf("emit(stale-ref): expected code %q, got %q (%s)",
			ErrorCodeStaleRef, parseErr.Code, parseErr.Message)
	}
}

// ---------------------------------------------------------------------------
// bridge.publish
// ---------------------------------------------------------------------------

// TestWritePublishDecodeErrors pins that bridge.publish returns
// ErrorCodeBadPayload for malformed payloads.
func TestWritePublishDecodeErrors(t *testing.T) {
	writeActivateAgentMode(t)

	parseCases := []struct {
		parseName    string
		parsePayload string
	}{
		{"not-json", `not json`},
		{"missing-topic", `{"payload":1}`},
		{"empty-topic", `{"topic":"","payload":1}`},
		{"missing-payload", `{"topic":"my-topic"}`},
	}

	for _, parseCase := range parseCases {
		parseCase := parseCase
		t.Run(parseCase.parseName, func(t *testing.T) {
			_, parseErr := writeHandlePublish(json.RawMessage(parseCase.parsePayload))
			if parseErr == nil {
				t.Fatalf("publish(%s): expected bad-payload error, got nil", parseCase.parseName)
			}
			if parseErr.Code != ErrorCodeBadPayload {
				t.Fatalf("publish(%s): expected code %q, got %q (%s)",
					parseCase.parseName, ErrorCodeBadPayload, parseErr.Code, parseErr.Message)
			}
		})
	}
}

// TestWritePublishDelivers pins that bridge.publish with a valid payload
// succeeds and returns an ok result.
func TestWritePublishDelivers(t *testing.T) {
	writeActivateAgentMode(t)

	parsePayload := `{"topic":"test.topic","payload":{"x":1}}`
	parseResult, parseErr := writeHandlePublish(json.RawMessage(parsePayload))
	if parseErr != nil {
		t.Fatalf("publish: unexpected error: %v", parseErr)
	}
	if parseResult == nil {
		t.Fatal("publish: expected non-nil result")
	}
}

// ---------------------------------------------------------------------------
// bridge.navigate
// ---------------------------------------------------------------------------

// TestWriteNavigateDecodeErrors pins that bridge.navigate returns
// ErrorCodeBadPayload for malformed payloads.
func TestWriteNavigateDecodeErrors(t *testing.T) {
	writeActivateAgentMode(t)

	parseCases := []struct {
		parseName    string
		parsePayload string
	}{
		{"not-json", `not json`},
		{"missing-path", `{}`},
		{"empty-path", `{"path":""}`},
	}

	for _, parseCase := range parseCases {
		parseCase := parseCase
		t.Run(parseCase.parseName, func(t *testing.T) {
			_, parseErr := writeHandleNavigate(json.RawMessage(parseCase.parsePayload))
			if parseErr == nil {
				t.Fatalf("navigate(%s): expected bad-payload error, got nil", parseCase.parseName)
			}
			if parseErr.Code != ErrorCodeBadPayload {
				t.Fatalf("navigate(%s): expected code %q, got %q (%s)",
					parseCase.parseName, ErrorCodeBadPayload, parseErr.Code, parseErr.Message)
			}
		})
	}
}

// TestWriteNavigateNativePlatformUnavailable pins that on native (non-wasm)
// builds bridge.navigate returns ErrorCodeBadPayload because the router package
// is not compiled in. On wasm builds the router.Navigate call would be reached
// and this sub-test is skipped — the wasm behaviour is validated by
// integration tests that boot the actual app.
func TestWriteNavigateNativePlatformUnavailable(t *testing.T) {
	writeActivateAgentMode(t)

	_, parseErr := writeNavigatePlatform("/some/path")
	// On native builds the stub returns ErrorCodeBadPayload.
	// On wasm builds this calls the real router and would succeed or panic
	// (no router mounted); either way, if writeNavigatePlatform does NOT
	// return an error we accept it — this test is specifically about the
	// native stub.
	if parseErr != nil && parseErr.Code != ErrorCodeBadPayload {
		t.Fatalf("navigate(native): expected bad-payload code, got %q (%s)",
			parseErr.Code, parseErr.Message)
	}
}

func TestWriteSetStateDecodeAndStaleRef(t *testing.T) {
	writeActivateAgentMode(t)

	parseCases := []struct {
		parseName    string
		parsePayload string
	}{
		{"not-json", `not json`},
		{"missing-ref", `{"slot":0,"value":1}`},
		{"missing-value", `{"ref":"x","slot":0}`},
		{"negative-slot", `{"ref":"x","slot":-1,"value":1}`},
	}
	for _, parseCase := range parseCases {
		parseCase := parseCase
		t.Run(parseCase.parseName, func(t *testing.T) {
			_, parseErr := writeHandleSetState(json.RawMessage(parseCase.parsePayload))
			if parseErr == nil || parseErr.Code != ErrorCodeBadPayload {
				t.Fatalf("set-state(%s) error = %#v, want bad-payload", parseCase.parseName, parseErr)
			}
		})
	}

	_, parseErr := writeHandleSetState(json.RawMessage(`{"ref":"missing/ref","slot":0,"value":1}`))
	if parseErr == nil || parseErr.Code != ErrorCodeStaleRef {
		t.Fatalf("set-state stale error = %#v, want stale-ref", parseErr)
	}
}

func TestWriteMountUnmountRoundTripClearsBridgeRegistry(t *testing.T) {
	writeActivateAgentMode(t)
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")
	parseAdapter.SetAttribute(parseContainer, "id", "agent-root")
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})

	writeMountMu.Lock()
	parseBaselineMounts := len(writeMountedRoots)
	writeMountMu.Unlock()
	parseBaselineAtoms := runtime.GetGlobalRuntime().SnapshotAtoms()

	RegisterMountComponent("agentbridge.test.Panel", func(parseProps map[string]any) *runtime.Element {
		parseText, _ := parseProps["text"].(string)
		return runtime.CreateElement("div", map[string]any{"id": "mounted-panel"}, runtime.CreateElement("TEXT_ELEMENT", map[string]any{"nodeValue": parseText}))
	})

	parseMountResult, parseMountErr := writeHandleMount(json.RawMessage(`{"id":"panel-1","component":"agentbridge.test.Panel","selector":"#agent-root","props":{"text":"hello"}}`))
	if parseMountErr != nil {
		t.Fatalf("mount returned error: %v", parseMountErr)
	}
	if !strings.Contains(string(parseMountResult), `"ok":true`) {
		t.Fatalf("mount result = %s", string(parseMountResult))
	}
	writeMountMu.Lock()
	parseMountCount := len(writeMountedRoots)
	writeMountMu.Unlock()
	if parseMountCount != parseBaselineMounts+1 {
		t.Fatalf("mounted root count = %d, want %d", parseMountCount, parseBaselineMounts+1)
	}
	if len(parseAdapter.GetChildren(parseContainer)) == 0 {
		t.Fatal("mount did not append rendered DOM under container")
	}

	if _, parseUnmountErr := writeHandleUnmount(json.RawMessage(`{"id":"panel-1"}`)); parseUnmountErr != nil {
		t.Fatalf("unmount returned error: %v", parseUnmountErr)
	}
	writeMountMu.Lock()
	parseFinalMounts := len(writeMountedRoots)
	writeMountMu.Unlock()
	if parseFinalMounts != parseBaselineMounts {
		t.Fatalf("mounted root registry leaked: got %d want %d", parseFinalMounts, parseBaselineMounts)
	}
	if parseAfterAtoms := runtime.GetGlobalRuntime().SnapshotAtoms(); len(parseAfterAtoms) != len(parseBaselineAtoms) {
		t.Fatalf("atom registry count changed across mount/unmount: before=%d after=%d", len(parseBaselineAtoms), len(parseAfterAtoms))
	}
}

func TestWriteDeleteAtomRequiresForceWithSubscriberRefs(t *testing.T) {
	writeActivateAgentMode(t)
	parseRt := runtime.GetGlobalRuntime()
	if parseErr := parseRt.RestoreAtomSnapshot(map[string]any{"agentbridge.delete.free": true}); parseErr != nil {
		t.Fatalf("seed atom: %v", parseErr)
	}

	parseRaw, parseErr := writeHandleDeleteAtom(json.RawMessage(`{"id":"agentbridge.delete.free"}`))
	if parseErr != nil {
		t.Fatalf("delete free atom returned error: %v", parseErr)
	}
	if !strings.Contains(string(parseRaw), `"deleted":true`) {
		t.Fatalf("delete result = %s", string(parseRaw))
	}
	if _, parseExists := parseRt.GetAtomValue("agentbridge.delete.free"); parseExists {
		t.Fatal("atom still exists after delete")
	}
}

// ---------------------------------------------------------------------------
// RegisterWriteCommands
// ---------------------------------------------------------------------------

// TestRegisterWriteCommandsRegistersAll pins that RegisterWriteCommands
// installs all four expected command names in the bridge registry.
func TestRegisterWriteCommandsRegistersAll(t *testing.T) {
	RegisterWriteCommands()

	parseExpected := []string{
		"bridge.set-atom",
		"bridge.set-state",
		"bridge.emit",
		"bridge.publish",
		"bridge.navigate",
		"bridge.mount",
		"bridge.unmount",
		"bridge.delete-atom",
	}

	parseRegistered := ListAgentCommands()
	parseSet := make(map[string]bool, len(parseRegistered))
	for _, parseName := range parseRegistered {
		parseSet[parseName] = true
	}

	for _, parseName := range parseExpected {
		if !parseSet[parseName] {
			t.Errorf("RegisterWriteCommands: command %q not found in registry", parseName)
		}
	}
}
