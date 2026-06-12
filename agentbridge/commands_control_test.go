package agentbridge

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
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
