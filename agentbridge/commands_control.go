package agentbridge

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/events"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// RegisterControlCommands installs bridge.wait-for and bridge.describe.
func RegisterControlCommands() {
	RegisterAgentCommand("bridge.wait-for", controlHandleWaitFor)
	RegisterAgentCommand("bridge.describe", controlHandleDescribe)
}

type controlWaitPayload struct {
	TimeoutMs int                      `json:"timeoutMs"`
	Idle      bool                     `json:"idle,omitempty"`
	Atom      *controlWaitAtomPayload  `json:"atom,omitempty"`
	Query     *controlWaitQueryPayload `json:"query,omitempty"`
}

type controlWaitAtomPayload struct {
	ID     string          `json:"id"`
	Equals json.RawMessage `json:"equals"`
}

type controlWaitQueryPayload struct {
	Role  string `json:"role,omitempty"`
	Label string `json:"label,omitempty"`
	Text  string `json:"text,omitempty"`
	ID    string `json:"id,omitempty"`
	Tag   string `json:"tag,omitempty"`
	Min   int    `json:"min,omitempty"`
}

type controlWaitResult struct {
	OK           bool   `json:"ok"`
	StateVersion uint64 `json:"stateVersion"`
	Reason       string `json:"reason,omitempty"`
}

type controlDescribeResult struct {
	StateVersion uint64                 `json:"stateVersion"`
	Commands     []string               `json:"commands"`
	Atoms        []controlDescribeAtom  `json:"atoms,omitempty"`
	Events       []controlDescribeEvent `json:"events,omitempty"`
}

type controlDescribeAtom struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Schema map[string]any `json:"schema"`
}

// controlDescribeEvent is one live pub/sub topic in the describe manifest. The
// subscriber count lets an agent see that a bridge.publish target has zero
// listeners (or whether the in-app subscribers are typed such that an untyped
// bridge publish may not reach them) before publishing blindly.
type controlDescribeEvent struct {
	Topic       string `json:"topic"`
	Subscribers int    `json:"subscribers"`
}

func controlHandleWaitFor(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	var parseReq controlWaitPayload
	if len(parsePayload) > 0 {
		if parseErr := json.Unmarshal(parsePayload, &parseReq); parseErr != nil {
			return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: fmt.Sprintf("wait-for: malformed payload: %v", parseErr)}
		}
	}
	parseTimeout := time.Duration(parseReq.TimeoutMs) * time.Millisecond
	if parseTimeout <= 0 {
		parseTimeout = 250 * time.Millisecond
	}
	parseDeadline := time.Now().Add(parseTimeout)
	parseRt := runtime.GetGlobalRuntime()
	for {
		parseOK, parseReason := controlWaitSatisfied(parseRt, parseReq)
		if parseOK {
			return controlMarshal(controlWaitResult{OK: true, StateVersion: parseRt.AgentStateVersion(), Reason: parseReason})
		}
		if time.Now().After(parseDeadline) {
			return nil, &EnvelopeError{Code: ErrorCodeTimeout, Message: "wait-for: timed out waiting for " + parseReason}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func controlWaitSatisfied(parseRt *runtime.Runtime, parseReq controlWaitPayload) (bool, string) {
	parseReasons := []string{}
	if parseReq.Idle {
		parseReasons = append(parseReasons, "idle")
	}
	if parseReq.Atom != nil {
		parseReasons = append(parseReasons, "atom "+parseReq.Atom.ID)
		parseValue, parseOK := parseRt.GetAtomValue(parseReq.Atom.ID)
		if !parseOK || !controlJSONEqual(parseValue, parseReq.Atom.Equals) {
			return false, strings.Join(parseReasons, ", ")
		}
	}
	if parseReq.Query != nil {
		parseReasons = append(parseReasons, "query")
		parseMin := parseReq.Query.Min
		if parseMin <= 0 {
			parseMin = 1
		}
		parseMatches := runtime.QueryAgentNodes(parseRt, runtime.AgentQuerySelector{
			Role:  parseReq.Query.Role,
			Label: parseReq.Query.Label,
			Text:  parseReq.Query.Text,
			ID:    parseReq.Query.ID,
			Tag:   parseReq.Query.Tag,
		})
		if len(parseMatches) < parseMin {
			return false, strings.Join(parseReasons, ", ")
		}
	}
	if len(parseReasons) == 0 {
		return true, "no conditions"
	}
	return true, strings.Join(parseReasons, ", ")
}

func controlHandleDescribe(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	if len(strings.TrimSpace(string(parsePayload))) > 0 && strings.TrimSpace(string(parsePayload)) != "{}" {
		var parseDiscard map[string]any
		if parseErr := json.Unmarshal(parsePayload, &parseDiscard); parseErr != nil {
			return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: fmt.Sprintf("describe: malformed payload: %v", parseErr)}
		}
	}
	parseRt := runtime.GetGlobalRuntime()
	parseSnapshot := parseRt.SnapshotAtoms()
	parseAtomIDs := make([]string, 0, len(parseSnapshot))
	for parseID := range parseSnapshot {
		parseAtomIDs = append(parseAtomIDs, parseID)
	}
	sort.Strings(parseAtomIDs)
	parseAtoms := make([]controlDescribeAtom, 0, len(parseAtomIDs))
	for _, parseID := range parseAtomIDs {
		parseAtoms = append(parseAtoms, controlDescribeAtom{
			ID:     parseID,
			Type:   controlTypeName(parseSnapshot[parseID]),
			Schema: controlJSONSchema(parseSnapshot[parseID]),
		})
	}
	parseTopics := events.Topics()
	parseEvents := make([]controlDescribeEvent, 0, len(parseTopics))
	for _, parseTopic := range parseTopics {
		parseEvents = append(parseEvents, controlDescribeEvent{
			Topic:       parseTopic.Topic,
			Subscribers: parseTopic.Subscribers,
		})
	}

	return controlMarshal(controlDescribeResult{
		StateVersion: parseRt.AgentStateVersion(),
		Commands:     ListAgentCommands(),
		Atoms:        parseAtoms,
		Events:       parseEvents,
	})
}

func controlMarshal(parseValue any) (json.RawMessage, *EnvelopeError) {
	parseBytes, parseErr := json.Marshal(parseValue)
	if parseErr != nil {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: parseErr.Error()}
	}
	return parseBytes, nil
}

func controlJSONEqual(parseValue any, parseWant json.RawMessage) bool {
	if len(parseWant) == 0 {
		return false
	}
	parseHaveBytes, parseErr := json.Marshal(parseValue)
	if parseErr != nil {
		return false
	}
	var parseHave any
	var parseWantValue any
	if parseErr = json.Unmarshal(parseHaveBytes, &parseHave); parseErr != nil {
		return false
	}
	if parseErr = json.Unmarshal(parseWant, &parseWantValue); parseErr != nil {
		return false
	}
	return reflect.DeepEqual(parseHave, parseWantValue)
}

func controlTypeName(parseValue any) string {
	if parseValue == nil {
		return "nil"
	}
	return reflect.TypeOf(parseValue).String()
}

func controlJSONSchema(parseValue any) map[string]any {
	switch parseValue.(type) {
	case nil:
		return map[string]any{"type": "null"}
	case bool:
		return map[string]any{"type": "boolean"}
	case string:
		return map[string]any{"type": "string"}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return map[string]any{"type": "number"}
	case []any:
		return map[string]any{"type": "array"}
	case map[string]any:
		return map[string]any{"type": "object"}
	default:
		parseKind := reflect.TypeOf(parseValue).Kind()
		if parseKind == reflect.Slice || parseKind == reflect.Array {
			return map[string]any{"type": "array"}
		}
		if parseKind == reflect.Map || parseKind == reflect.Struct {
			return map[string]any{"type": "object"}
		}
		return map[string]any{"type": "string"}
	}
}
