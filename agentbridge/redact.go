package agentbridge

import (
	"encoding/json"
	"time"

	"github.com/monstercameron/GoWebComponents/v7/internal/runtime"
)

// redactedPlaceholder replaces any value withheld by the snapshot redaction
// policy; it matches the marker BuildAgentSnapshot uses.
const redactedPlaceholder = "[redacted]"

// redactText returns parseText, or the placeholder when a snapshot redaction
// policy is active and the text is non-empty (fail-closed, as in snapshots).
func redactText(parseText string) string {
	if parseText != "" && runtime.AgentRedactionActive() {
		return redactedPlaceholder
	}
	return parseText
}

// redactValue returns parseValue, or the placeholder when a snapshot redaction
// policy is active and the value is non-nil.
func redactValue(parseValue any) any {
	if parseValue != nil && runtime.AgentRedactionActive() {
		return redactedPlaceholder
	}
	return parseValue
}

// buildConsolePayload encodes a captured console or window-error event. The
// message and every argument pass through the snapshot redaction policy, so a
// secret logged to the console cannot reach the agent when the same secret is
// hidden from snapshots.
func buildConsolePayload(parseSource string, parseMessage string, parseArgsText []string, parseNow time.Time) json.RawMessage {
	if parseMessage == "" && len(parseArgsText) > 0 {
		parseMessage = parseArgsText[0]
	}
	parseSafeArgs := make([]string, len(parseArgsText))
	for parseIdx, parseArg := range parseArgsText {
		parseSafeArgs[parseIdx] = redactText(parseArg)
	}
	parsePayload, _ := json.Marshal(map[string]any{
		"source":    parseSource,
		"severity":  "error",
		"message":   redactText(parseMessage),
		"arguments": parseSafeArgs,
		"time":      parseNow.UTC().Format(time.RFC3339Nano),
	})
	return parsePayload
}
