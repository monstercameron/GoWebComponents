package agentbridge

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// replayMu guards the replay recording flag and the captured event buffer.
var replayMu sync.Mutex

// replayRecording reports whether deterministic update recording is active.
var replayRecording bool

// replayCaptured holds the most recently stopped recording for replay.
var replayCaptured []runtime.ReplayEvent

// RegisterReplayCommands installs bridge.replay, the agent-facing wrapper over
// the runtime's deterministic update recorder. An agent records a window of
// scheduling updates, then replays them to reproduce a bug deterministically —
// the local stand-in for the prod→plan loop.
func RegisterReplayCommands() {
	RegisterAgentCommand("bridge.replay", replayHandle)
}

// replayPayload is the decoded bridge.replay request. Action is one of
// start / stop / status / replay; an empty action is treated as status.
type replayPayload struct {
	Action string `json:"action"`
}

// replayResult is the bridge.replay response.
type replayResult struct {
	Action    string `json:"action"`
	Recording bool   `json:"recording"`
	Captured  int    `json:"captured"`
	Replayed  int    `json:"replayed,omitempty"`
}

// replayHandle handles bridge.replay. start/replay mutate runtime scheduling and
// require agent mode; stop/status are read-only.
func replayHandle(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	parseReq := replayPayload{}
	if len(parsePayload) > 0 {
		if parseErr := json.Unmarshal(parsePayload, &parseReq); parseErr != nil {
			return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "replay: malformed payload: " + parseErr.Error()}
		}
	}
	parseRt := runtime.GetGlobalRuntime()

	replayMu.Lock()
	defer replayMu.Unlock()

	switch strings.TrimSpace(parseReq.Action) {
	case "start":
		if !IsAgentModeActive() {
			return nil, &EnvelopeError{Code: ErrorCodeForbidden, Message: "agent mode is not active"}
		}
		// Refuse a double-start: starting again silently resets the recorder and
		// drops the in-progress capture window with no signal. Make the caller
		// stop first so a lost window is never invisible.
		if replayRecording {
			return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "replay: already recording; stop before starting a new capture"}
		}
		parseRt.StartReplayRecording()
		replayRecording = true
		return replayMarshal(replayResult{Action: "start", Recording: true, Captured: len(replayCaptured)})
	case "stop":
		// stop mutates the recorder (ends recording), so gate it like start —
		// status below is a pure read and stays ungated.
		if !IsAgentModeActive() {
			return nil, &EnvelopeError{Code: ErrorCodeForbidden, Message: "agent mode is not active"}
		}
		parseEvents := parseRt.StopReplayRecording()
		replayRecording = false
		replayCaptured = parseEvents
		return replayMarshal(replayResult{Action: "stop", Recording: false, Captured: len(parseEvents)})
	case "status", "":
		return replayMarshal(replayResult{Action: "status", Recording: replayRecording, Captured: len(replayCaptured)})
	case "replay":
		if !IsAgentModeActive() {
			return nil, &EnvelopeError{Code: ErrorCodeForbidden, Message: "agent mode is not active"}
		}
		parseCount := len(replayCaptured)
		parseRt.ReplayUpdates(replayCaptured)
		// Replaying re-applies recorded updates to live state; record it so the
		// audit trail is not blind to it. It is not reversible (the updates are
		// re-applied, not diffed), so no undo closure.
		RecordAgentMutation("bridge.replay", "replay "+itoa(uint64(parseCount))+" recorded updates", nil)
		return replayMarshal(replayResult{Action: "replay", Recording: replayRecording, Captured: parseCount, Replayed: parseCount})
	default:
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "replay: unknown action " + parseReq.Action}
	}
}

// replayMarshal serializes a replay result into a wire payload.
func replayMarshal(parseValue replayResult) (json.RawMessage, *EnvelopeError) {
	parseBytes, parseErr := json.Marshal(parseValue)
	if parseErr != nil {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: parseErr.Error()}
	}
	return parseBytes, nil
}

// resetReplayState clears recording state for tests.
func resetReplayState() {
	replayMu.Lock()
	defer replayMu.Unlock()
	replayRecording = false
	replayCaptured = nil
}
