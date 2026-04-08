package app

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestCreateAuditLogRedactsSecrets verifies audit payload persistence uses the shared secret scrubber.
func TestCreateAuditLogRedactsSecrets(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "audit-redaction@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "ws-audit-redaction")

	parsePayloadJSON := `{"authorization":"Bearer abc.def.ghi","api_key":"sk-live-123","path":"C:\\secrets\\config.json","nested":{"cookie":"session=abc"}}`
	if _, parseErr := parseStore.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseUser.ID,
		WorkspaceID: parseWorkspaceID,
		EventType:   "admin.redaction.test",
		TargetType:  "payload",
		TargetID:    "payload-1",
		Summary:     `Bearer abc.def.ghi and C:\secrets\config.json`,
		PayloadJSON: parsePayloadJSON,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAuditLog: %v", parseErr)
	}

	parseRows, parseErr := parseStore.parseListAuditLogs(1)
	if parseErr != nil {
		parseT.Fatalf("parseListAuditLogs: %v", parseErr)
	}
	if len(parseRows) != 1 {
		parseT.Fatalf("expected one audit row, got %d", len(parseRows))
	}
	parseRow := parseRows[0]
	if strings.Contains(parseRow.Summary, "abc.def.ghi") || strings.Contains(parseRow.Summary, "C:\\secrets\\config.json") {
		parseT.Fatalf("expected summary redaction, got %q", parseRow.Summary)
	}
	var parsePayload map[string]any
	if parseErr = json.Unmarshal([]byte(parseRow.PayloadJSON), &parsePayload); parseErr != nil {
		parseT.Fatalf("json.Unmarshal(payload): %v", parseErr)
	}
	if parsePayload["authorization"] != parseLogRedactionText || parsePayload["api_key"] != parseLogRedactionText || parsePayload["path"] != parseLogRedactionText {
		parseT.Fatalf("expected redacted payload fields, got %+v", parsePayload)
	}
	parseNested, parseOk := parsePayload["nested"].(map[string]any)
	if !parseOk {
		parseT.Fatalf("expected nested payload map, got %+v", parsePayload)
	}
	if parseNested["cookie"] != parseLogRedactionText {
		parseT.Fatalf("expected nested cookie redaction, got %+v", parseNested)
	}
}
