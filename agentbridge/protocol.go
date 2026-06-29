package agentbridge

import (
	"encoding/json"
	"fmt"
)

// ProtocolName identifies the agent bridge wire protocol in every envelope.
const ProtocolName = "gwc.agentbridge"

// ProtocolVersion is the newest envelope schema version this build speaks.
// Parsers reject newer versions loudly instead of guessing.
const ProtocolVersion = 1

// Envelope kinds. Every frame on the wire is exactly one of these.
const (
	// KindHello is the first frame an app sends after connecting: app id,
	// build id, and capabilities ride in the payload.
	KindHello = "hello"
	// KindCommand is a hub-to-app instruction (snapshot, query, set-atom, ...).
	KindCommand = "command"
	// KindAck is the app's reply to exactly one command, matched by AckSeq.
	KindAck = "ack"
	// KindEvent is an unsolicited app-to-hub push (log, diagnostic, lifecycle).
	KindEvent = "event"
)

// Command-layer error codes carried in EnvelopeError.Code. These are part of
// the wire contract: agents branch on them, so they are stable strings.
const (
	// ErrorCodeStaleRef means a node ref no longer resolves; the agent should
	// re-query instead of retrying the same ref.
	ErrorCodeStaleRef = "stale-ref"
	// ErrorCodeUnknownCommand means the app build does not implement the
	// requested command name.
	ErrorCodeUnknownCommand = "unknown-command"
	// ErrorCodeBadPayload means the command payload failed validation; state
	// was not touched.
	ErrorCodeBadPayload = "bad-payload"
	// ErrorCodeForbidden means the app is not in agent mode or the caller
	// lacks the session write lease.
	ErrorCodeForbidden = "forbidden"
	// ErrorCodeTimeout means a wait command reached its deadline before the
	// requested runtime condition became true.
	ErrorCodeTimeout = "timeout"
)

// EnvelopeError is the structured failure attached to a non-ok ack.
type EnvelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

// Envelope is one frame of the gwc.agentbridge protocol. Each side numbers
// its own outbound frames with a monotonic Seq starting at 1; an ack points
// back at the command it answers via AckSeq.
type Envelope struct {
	// Protocol is the wire-protocol identifier (always ProtocolName) — receivers reject mismatches.
	Protocol string `json:"protocol"`
	// Version is the protocol version (always ProtocolVersion) for compatibility negotiation.
	Version int `json:"version"`
	// Kind is the frame type: KindHello, KindCommand, KindAck, or KindEvent.
	Kind string `json:"kind"`
	// Seq is this sender's monotonic frame number, starting at 1.
	Seq uint64 `json:"seq"`
	// Session names the app session a frame belongs to. It is empty on the
	// app's hello (the hub assigns the session id in its reply) and may be
	// empty on app<->hub frames where the connection implies the session.
	Session string `json:"session,omitempty"`
	// Name is the command name for KindCommand and the topic for KindEvent.
	Name string `json:"name,omitempty"`
	// Payload is the kind-specific body, left raw so the protocol layer stays
	// decoupled from individual command schemas.
	Payload json.RawMessage `json:"payload,omitempty"`
	// AckSeq is the Seq of the command a KindAck frame answers.
	AckSeq uint64 `json:"ackSeq,omitempty"`
	// OK reports command success on a KindAck frame. It is a pointer so acks
	// are explicit on the wire and other kinds omit it entirely.
	OK *bool `json:"ok,omitempty"`
	// Error carries the structured failure when OK is false.
	Error *EnvelopeError `json:"error,omitempty"`
	// StateVersion is the app's post-commit state version on a successful
	// ack, giving callers read-your-writes ordering without polling.
	StateVersion uint64 `json:"stateVersion,omitempty"`
}

// NewHelloEnvelope builds the first frame an app sends after connecting.
func NewHelloEnvelope(parseSeq uint64, parsePayload json.RawMessage) Envelope {
	return Envelope{
		Protocol: ProtocolName,
		Version:  ProtocolVersion,
		Kind:     KindHello,
		Seq:      parseSeq,
		Payload:  parsePayload,
	}
}

// NewCommandEnvelope builds a hub-to-app command frame.
func NewCommandEnvelope(parseSeq uint64, parseSession string, parseName string, parsePayload json.RawMessage) Envelope {
	return Envelope{
		Protocol: ProtocolName,
		Version:  ProtocolVersion,
		Kind:     KindCommand,
		Seq:      parseSeq,
		Session:  parseSession,
		Name:     parseName,
		Payload:  parsePayload,
	}
}

// NewAckEnvelope builds a successful reply to the command numbered parseAckSeq.
func NewAckEnvelope(parseSeq uint64, parseSession string, parseAckSeq uint64, parseStateVersion uint64, parsePayload json.RawMessage) Envelope {
	parseOK := true
	return Envelope{
		Protocol:     ProtocolName,
		Version:      ProtocolVersion,
		Kind:         KindAck,
		Seq:          parseSeq,
		Session:      parseSession,
		AckSeq:       parseAckSeq,
		OK:           &parseOK,
		StateVersion: parseStateVersion,
		Payload:      parsePayload,
	}
}

// NewErrorAckEnvelope builds a failed reply to the command numbered
// parseAckSeq, carrying a stable error code the agent can branch on.
func NewErrorAckEnvelope(parseSeq uint64, parseSession string, parseAckSeq uint64, parseCode string, parseMessage string) Envelope {
	parseOK := false
	return Envelope{
		Protocol: ProtocolName,
		Version:  ProtocolVersion,
		Kind:     KindAck,
		Seq:      parseSeq,
		Session:  parseSession,
		AckSeq:   parseAckSeq,
		OK:       &parseOK,
		Error:    &EnvelopeError{Code: parseCode, Message: parseMessage},
	}
}

// NewEventEnvelope builds an unsolicited app-to-hub push frame.
func NewEventEnvelope(parseSeq uint64, parseSession string, parseName string, parsePayload json.RawMessage) Envelope {
	return Envelope{
		Protocol: ProtocolName,
		Version:  ProtocolVersion,
		Kind:     KindEvent,
		Seq:      parseSeq,
		Session:  parseSession,
		Name:     parseName,
		Payload:  parsePayload,
	}
}

// BuildHelloEnvelope builds the first frame an app sends after connecting.
//
// Deprecated: use NewHelloEnvelope (New* matches the module-wide constructor convention).
func BuildHelloEnvelope(parseSeq uint64, parsePayload json.RawMessage) Envelope {
	return NewHelloEnvelope(parseSeq, parsePayload)
}

// BuildCommandEnvelope builds a hub-to-app command frame.
//
// Deprecated: use NewCommandEnvelope.
func BuildCommandEnvelope(parseSeq uint64, parseSession string, parseName string, parsePayload json.RawMessage) Envelope {
	return NewCommandEnvelope(parseSeq, parseSession, parseName, parsePayload)
}

// BuildAckEnvelope builds a successful reply to the command numbered parseAckSeq.
//
// Deprecated: use NewAckEnvelope.
func BuildAckEnvelope(parseSeq uint64, parseSession string, parseAckSeq uint64, parseStateVersion uint64, parsePayload json.RawMessage) Envelope {
	return NewAckEnvelope(parseSeq, parseSession, parseAckSeq, parseStateVersion, parsePayload)
}

// BuildErrorAckEnvelope builds a failed reply to the command numbered parseAckSeq.
//
// Deprecated: use NewErrorAckEnvelope.
func BuildErrorAckEnvelope(parseSeq uint64, parseSession string, parseAckSeq uint64, parseCode string, parseMessage string) Envelope {
	return NewErrorAckEnvelope(parseSeq, parseSession, parseAckSeq, parseCode, parseMessage)
}

// BuildEventEnvelope builds an unsolicited app-to-hub push frame.
//
// Deprecated: use NewEventEnvelope.
func BuildEventEnvelope(parseSeq uint64, parseSession string, parseName string, parsePayload json.RawMessage) Envelope {
	return NewEventEnvelope(parseSeq, parseSession, parseName, parsePayload)
}

// FormatEnvelopeJSON validates an envelope and serializes it for the wire.
func FormatEnvelopeJSON(parseEnvelope Envelope) (string, error) {
	if parseErr := validateEnvelope(parseEnvelope); parseErr != nil {
		return "", parseErr
	}
	parseBytes, parseErr := json.Marshal(parseEnvelope)
	if parseErr != nil {
		return "", fmt.Errorf("agentbridge: marshal envelope: %w", parseErr)
	}
	return string(parseBytes), nil
}

// ParseEnvelope decodes one wire frame and rejects anything this build does
// not speak: wrong protocol, future version, unknown kind, or a frame whose
// kind-specific required fields are missing.
func ParseEnvelope(parsePayload string) (Envelope, error) {
	var parseEnvelope Envelope
	if parseErr := json.Unmarshal([]byte(parsePayload), &parseEnvelope); parseErr != nil {
		return Envelope{}, fmt.Errorf("agentbridge: malformed envelope JSON: %w", parseErr)
	}
	if parseErr := validateEnvelope(parseEnvelope); parseErr != nil {
		return Envelope{}, parseErr
	}
	return parseEnvelope, nil
}

// validateEnvelope enforces the protocol-level invariants shared by the
// format and parse paths; command payload schemas are validated downstream.
func validateEnvelope(parseEnvelope Envelope) error {
	if parseEnvelope.Protocol != ProtocolName {
		return fmt.Errorf("agentbridge: unsupported protocol %q (want %q)", parseEnvelope.Protocol, ProtocolName)
	}
	if parseEnvelope.Version < 1 {
		return fmt.Errorf("agentbridge: missing or invalid protocol version %d", parseEnvelope.Version)
	}
	if parseEnvelope.Version > ProtocolVersion {
		return fmt.Errorf("agentbridge: protocol version %d is newer than supported version %d", parseEnvelope.Version, ProtocolVersion)
	}
	if parseEnvelope.Seq == 0 {
		return fmt.Errorf("agentbridge: missing seq (frames number from 1)")
	}
	switch parseEnvelope.Kind {
	case KindHello:
		// Hello carries identity in the payload; no protocol-level fields.
	case KindCommand:
		if parseEnvelope.Name == "" {
			return fmt.Errorf("agentbridge: command frame missing name")
		}
	case KindEvent:
		if parseEnvelope.Name == "" {
			return fmt.Errorf("agentbridge: event frame missing name")
		}
	case KindAck:
		if parseEnvelope.AckSeq == 0 {
			return fmt.Errorf("agentbridge: ack frame missing ackSeq")
		}
		if parseEnvelope.OK == nil {
			return fmt.Errorf("agentbridge: ack frame missing ok")
		}
		if !*parseEnvelope.OK {
			if parseEnvelope.Error == nil || parseEnvelope.Error.Code == "" {
				return fmt.Errorf("agentbridge: failed ack missing structured error code")
			}
		} else if parseEnvelope.Error != nil {
			return fmt.Errorf("agentbridge: successful ack must not carry an error")
		}
	default:
		return fmt.Errorf("agentbridge: unknown envelope kind %q", parseEnvelope.Kind)
	}
	return nil
}
