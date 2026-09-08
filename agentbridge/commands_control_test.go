package agentbridge

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

func TestRegisterControlCommandsRegistersWaitForAndDescribe(t *testing.T) {
	RegisterControlCommands()
	parseNames := ListAgentCommands()
	parseSet := map[string]bool{}
	for _, parseName := range parseNames {
		parseSet[parseName] = true
	}
	for _, parseWant := range []string{"bridge.wait-for", "bridge.describe"} {
		if !parseSet[parseWant] {
			t.Fatalf("expected %s to be registered; got %v", parseWant, parseNames)
		}
	}
}

func TestControlDescribeReportsAtomsCommandsAndSchemas(t *testing.T) {
	RegisterControlCommands()
	parseRt := runtime.GetGlobalRuntime()
	if parseErr := parseRt.RestoreAtomSnapshot(map[string]any{
		"agentbridge.describe.count": 3,
		"agentbridge.describe.name":  "Atlas",
	}); parseErr != nil {
		t.Fatalf("restore atoms: %v", parseErr)
	}
	parseRt.AdvanceAgentStateVersion()

	parseRaw, parseEnvelopeErr := controlHandleDescribe(json.RawMessage(`{}`))
	if parseEnvelopeErr != nil {
		t.Fatalf("describe error: %v", parseEnvelopeErr)
	}
	var parseResult controlDescribeResult
	if parseErr := json.Unmarshal(parseRaw, &parseResult); parseErr != nil {
		t.Fatalf("decode describe: %v", parseErr)
	}
	if parseResult.StateVersion == 0 {
		t.Fatal("expected non-zero stateVersion")
	}
	parseHasDescribe := false
	for _, parseCommand := range parseResult.Commands {
		if parseCommand == "bridge.describe" {
			parseHasDescribe = true
			break
		}
	}
	if !parseHasDescribe {
		t.Fatalf("commands missing bridge.describe: %v", parseResult.Commands)
	}
	parseSeen := map[string]controlDescribeAtom{}
	for _, parseAtom := range parseResult.Atoms {
		parseSeen[parseAtom.ID] = parseAtom
	}
	if parseSeen["agentbridge.describe.count"].Schema["type"] != "number" {
		t.Fatalf("count schema = %#v", parseSeen["agentbridge.describe.count"].Schema)
	}
	if parseSeen["agentbridge.describe.name"].Schema["type"] != "string" {
		t.Fatalf("name schema = %#v", parseSeen["agentbridge.describe.name"].Schema)
	}
}

func TestControlWaitForAtomEqualityAndTimeout(t *testing.T) {
	parseRt := runtime.GetGlobalRuntime()
	if parseErr := parseRt.RestoreAtomSnapshot(map[string]any{"agentbridge.wait.ready": false}); parseErr != nil {
		t.Fatalf("seed atom: %v", parseErr)
	}
	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = parseRt.RestoreAtomSnapshot(map[string]any{"agentbridge.wait.ready": true})
		parseRt.AdvanceAgentStateVersion()
	}()

	parseRaw, parseEnvelopeErr := controlHandleWaitFor(json.RawMessage(`{"timeoutMs":250,"atom":{"id":"agentbridge.wait.ready","equals":true}}`))
	if parseEnvelopeErr != nil {
		t.Fatalf("wait-for atom returned error: %v", parseEnvelopeErr)
	}
	if !strings.Contains(string(parseRaw), `"ok":true`) {
		t.Fatalf("wait-for payload = %s", string(parseRaw))
	}

	_, parseTimeoutErr := controlHandleWaitFor(json.RawMessage(`{"timeoutMs":20,"atom":{"id":"agentbridge.wait.never","equals":true}}`))
	if parseTimeoutErr == nil {
		t.Fatal("expected timeout error")
	}
	if parseTimeoutErr.Code != ErrorCodeTimeout {
		t.Fatalf("timeout code = %q, want %q", parseTimeoutErr.Code, ErrorCodeTimeout)
	}
}

func TestControlWaitForMalformedPayloadAndQueryMatch(t *testing.T) {
	if _, parseErr := controlHandleWaitFor(json.RawMessage(`not json`)); parseErr == nil || parseErr.Code != ErrorCodeBadPayload {
		t.Fatalf("malformed wait-for error = %#v, want bad-payload", parseErr)
	}

	parseAdapter := mockdom.NewMockDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")
	parseAdapter.SetAttribute(parseContainer, "id", "agent-control-query")
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRt := runtime.GetGlobalRuntime()
	parseRt.Render(runtime.Button(map[string]any{"aria-label": "Save changes"}, "Save"), parseContainer)

	parseRaw, parseErr := controlHandleWaitFor(json.RawMessage(`{"timeoutMs":25,"query":{"role":"button","label":"save","min":1}}`))
	if parseErr != nil {
		t.Fatalf("wait-for query returned error: %v", parseErr)
	}
	var parseResult controlWaitResult
	if parseDecodeErr := json.Unmarshal(parseRaw, &parseResult); parseDecodeErr != nil {
		t.Fatalf("decode wait-for query result: %v", parseDecodeErr)
	}
	if !parseResult.OK || parseResult.Reason != "query" {
		t.Fatalf("wait-for query result = %+v, want ok query", parseResult)
	}

	_, parseTimeoutErr := controlHandleWaitFor(json.RawMessage(`{"timeoutMs":10,"query":{"role":"button","label":"never","min":2}}`))
	if parseTimeoutErr == nil || parseTimeoutErr.Code != ErrorCodeTimeout {
		t.Fatalf("wait-for query timeout error = %#v, want timeout", parseTimeoutErr)
	}
	if !strings.Contains(parseTimeoutErr.Message, "query") {
		t.Fatalf("timeout message = %q, want query reason", parseTimeoutErr.Message)
	}
}

func TestControlDescribeRejectsMalformedPayload(t *testing.T) {
	_, parseErr := controlHandleDescribe(json.RawMessage(`{"unterminated"`))
	if parseErr == nil {
		t.Fatal("expected malformed describe payload error")
	}
	if parseErr.Code != ErrorCodeBadPayload || !strings.Contains(parseErr.Message, "malformed payload") {
		t.Fatalf("describe malformed error = %#v, want bad-payload malformed", parseErr)
	}
}

func TestControlJSONHelpersCoverMismatchAndSchemaTypes(t *testing.T) {
	if controlJSONEqual(map[string]any{"a": 1}, json.RawMessage(`{"a":2}`)) {
		t.Fatal("controlJSONEqual reported mismatched JSON objects equal")
	}
	if controlJSONEqual("value", json.RawMessage(`{`)) {
		t.Fatal("controlJSONEqual reported malformed wanted JSON equal")
	}

	parseCases := []struct {
		value any
		want  map[string]any
	}{
		{nil, map[string]any{"type": "null"}},
		{true, map[string]any{"type": "boolean"}},
		{"x", map[string]any{"type": "string"}},
		{[]string{"a"}, map[string]any{"type": "array"}},
		{map[string]int{"a": 1}, map[string]any{"type": "object"}},
		{struct{ Name string }{Name: "x"}, map[string]any{"type": "object"}},
		{chan int(nil), map[string]any{"type": "string"}},
	}
	for _, parseCase := range parseCases {
		if parseGot := controlJSONSchema(parseCase.value); !reflect.DeepEqual(parseGot, parseCase.want) {
			t.Fatalf("controlJSONSchema(%T) = %#v, want %#v", parseCase.value, parseGot, parseCase.want)
		}
	}
	if parseGot := controlTypeName(nil); parseGot != "nil" {
		t.Fatalf("controlTypeName(nil) = %q, want nil", parseGot)
	}
}
