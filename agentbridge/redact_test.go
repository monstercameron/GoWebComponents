package agentbridge

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v7/internal/runtime"
)

// configureTestRedaction installs a redaction policy for one test.
func configureTestRedaction(parseTB testing.TB) {
	parseTB.Helper()
	runtime.ConfigurePanicReportRedaction(runtime.PanicReportRedactionPolicy{Fields: []string{"token"}})
	parseTB.Cleanup(func() { runtime.ConfigurePanicReportRedaction(runtime.PanicReportRedactionPolicy{}) })
}

// TestConsolePayloadRedactsSecretsUnderPolicy pins that a secret-bearing console
// message and window error never reach the payload when a policy is active.
func TestConsolePayloadRedactsSecretsUnderPolicy(t *testing.T) {
	parseArgs := []string{"login failed token=sk-live-123", "extra sk-live-123"}
	parseNow := time.Unix(0, 0)

	parseOpen := string(buildConsolePayload("console.error", "", parseArgs, parseNow))
	if !strings.Contains(parseOpen, "sk-live-123") {
		t.Fatalf("without a policy the message passes through unchanged: %s", parseOpen)
	}

	configureTestRedaction(t)
	for _, parseMessage := range []string{"", "window error sk-live-123"} {
		parseRaw := buildConsolePayload("window.onerror", parseMessage, parseArgs, parseNow)
		if strings.Contains(string(parseRaw), "sk-live-123") {
			t.Fatalf("secret leaked through console payload: %s", parseRaw)
		}
		var parseDecoded struct {
			Source    string   `json:"source"`
			Message   string   `json:"message"`
			Arguments []string `json:"arguments"`
		}
		if parseErr := json.Unmarshal(parseRaw, &parseDecoded); parseErr != nil {
			t.Fatal(parseErr)
		}
		if parseDecoded.Message != redactedPlaceholder || len(parseDecoded.Arguments) != 2 || parseDecoded.Arguments[0] != redactedPlaceholder || parseDecoded.Source != "window.onerror" {
			t.Fatalf("unexpected redacted payload: %s", parseRaw)
		}
	}
}

// TestWriteSetAtomDryRunRedactsCurrentValue pins that a dry run does not echo
// the atom's current (possibly secret) value when a policy is active.
func TestWriteSetAtomDryRunRedactsCurrentValue(t *testing.T) {
	writeActivateAgentMode(t)
	seedAtom(t, "agenttest.apikey", "sk-live-123")

	parseDry := func() string {
		parseRaw, parseErr := writeCallHandler(writeHandleSetAtom, map[string]any{
			"id": "agenttest.apikey", "value": "next", "dryRun": true,
		})
		if parseErr != nil {
			t.Fatalf("dry run: %v", parseErr)
		}
		return string(parseRaw)
	}
	if parseOpen := parseDry(); !strings.Contains(parseOpen, "sk-live-123") {
		t.Fatalf("without a policy the dry run reports the prior value: %s", parseOpen)
	}
	configureTestRedaction(t)
	parseRedacted := parseDry()
	if strings.Contains(parseRedacted, "sk-live-123") || !strings.Contains(parseRedacted, redactedPlaceholder) {
		t.Fatalf("dry run leaked the current value: %s", parseRedacted)
	}
	if got := currentAtom(t, "agenttest.apikey"); got != "sk-live-123" {
		t.Fatalf("dry run mutated the atom: %#v", got)
	}
}
