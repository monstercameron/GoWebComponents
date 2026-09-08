package agentbridge

import (
	"encoding/json"
	"fmt"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// readSnapshotPayload is the decoded payload for the bridge.snapshot command.
type readSnapshotPayload struct {
	MaxDepth int `json:"maxDepth"`
	MaxNodes int `json:"maxNodes"`
}

// readQueryPayload is the decoded payload for the bridge.query command.
type readQueryPayload struct {
	Role  string `json:"role,omitempty"`
	Label string `json:"label,omitempty"`
	Text  string `json:"text,omitempty"`
	ID    string `json:"id,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

// readSnapshotResult is the JSON-encoded payload returned by bridge.snapshot.
type readSnapshotResult struct {
	Snapshot runtime.AgentSnapshot `json:"snapshot"`
}

// readQueryResult is the JSON-encoded payload returned by bridge.query.
type readQueryResult struct {
	Matches []runtime.AgentQueryMatch `json:"matches"`
}

// RegisterReadCommands installs the bridge.snapshot and
// bridge.query command handlers into the process-wide bridge command registry
// via RegisterAgentCommand. It should be called once, typically from
// application bootstrap or from an init() guarded by an appropriate build tag.
// Calling it multiple times is safe (last registration wins per
// RegisterAgentCommand semantics).
//
// Both commands are read-only and work regardless of whether agent mode is
// active (reads are always allowed in agent builds).
func RegisterReadCommands() {
	RegisterAgentCommand("bridge.snapshot", readHandleSnapshot)
	RegisterAgentCommand("bridge.query", readHandleQuery)
}

// readHandleSnapshot handles the bridge.snapshot command. It decodes an
// optional {maxDepth, maxNodes} payload, calls BuildAgentSnapshot against the
// global runtime, and returns the serialised AgentSnapshot. A nil or no-tree
// runtime returns an empty ok result rather than an error, so agents can safely
// snapshot before the app has mounted.
func readHandleSnapshot(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	var parsePay readSnapshotPayload
	if len(parsePayload) > 0 {
		if parseErr := json.Unmarshal(parsePayload, &parsePay); parseErr != nil {
			return nil, &EnvelopeError{
				Code:    ErrorCodeBadPayload,
				Message: fmt.Sprintf("bridge.snapshot: bad payload: %v", parseErr),
			}
		}
	}

	parseRt := runtime.GetGlobalRuntime()
	parseSnap := runtime.BuildAgentSnapshot(parseRt, parsePay.MaxDepth, parsePay.MaxNodes)

	parseResult := readSnapshotResult{Snapshot: parseSnap}
	parseBytes, parseErr := json.Marshal(parseResult)
	if parseErr != nil {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: fmt.Sprintf("bridge.snapshot: marshal result: %v", parseErr),
		}
	}
	return parseBytes, nil
}

// readHandleQuery handles the bridge.query command. It decodes an optional
// selector payload ({role, label, text, id, tag}), calls QueryAgentNodes
// against the global runtime, and returns the serialised match list. An empty
// or nil tree returns an empty ok result.
func readHandleQuery(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	var parsePay readQueryPayload
	if len(parsePayload) > 0 {
		if parseErr := json.Unmarshal(parsePayload, &parsePay); parseErr != nil {
			return nil, &EnvelopeError{
				Code:    ErrorCodeBadPayload,
				Message: fmt.Sprintf("bridge.query: bad payload: %v", parseErr),
			}
		}
	}

	parseRt := runtime.GetGlobalRuntime()
	parseSel := runtime.AgentQuerySelector{
		Role:  parsePay.Role,
		Label: parsePay.Label,
		Text:  parsePay.Text,
		ID:    parsePay.ID,
		Tag:   parsePay.Tag,
	}
	parseMatches := runtime.QueryAgentNodes(parseRt, parseSel)
	if parseMatches == nil {
		parseMatches = []runtime.AgentQueryMatch{}
	}

	parseResult := readQueryResult{Matches: parseMatches}
	parseBytes, parseErr := json.Marshal(parseResult)
	if parseErr != nil {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: fmt.Sprintf("bridge.query: marshal result: %v", parseErr),
		}
	}
	return parseBytes, nil
}

// readDecodeSelector is an unexported helper that decodes a raw JSON selector
// payload into a readQueryPayload. Prefixed with "read" to avoid collisions
// with other command files added to this package concurrently.
func readDecodeSelector(parseRaw json.RawMessage) (readQueryPayload, error) {
	var parseSel readQueryPayload
	if len(parseRaw) == 0 {
		return parseSel, nil
	}
	if parseErr := json.Unmarshal(parseRaw, &parseSel); parseErr != nil {
		return parseSel, fmt.Errorf("readDecodeSelector: %w", parseErr)
	}
	return parseSel, nil
}
