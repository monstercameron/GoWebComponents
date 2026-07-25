package projection

import (
	"encoding/json"
	"errors"
	"fmt"
)

// The wire envelope between app.wasm and services.wasm.
//
// WorkerClient owns the correlation table; this file owns what actually travels.
// They are separate because the table is where the bugs are and is worth testing
// natively, while the envelope is a shape both sides have to agree on and is
// worth stating once, here, rather than twice in two binaries that can drift.

// RequestEnvelope is one command on the wire.
type RequestEnvelope struct {
	// RequestID is what the reply must echo. Correlation depends on it entirely,
	// so a worker that drops it produces replies nobody can match.
	RequestID uint64 `json:"rid"`
	Command   string `json:"cmd"`
	// Payload is the encoded arguments, opaque to the envelope.
	Payload []byte `json:"p,omitempty"`
}

// ReplyKind tells the client which of its constructors to use.
//
// Carried explicitly rather than inferred from whether an error string is
// present, because "rejected", "panicked", and "the worker is dying" need
// different handling and only one of them is retryable. Inferring would collapse
// them into one, which is what FailureKind exists to prevent.
type ReplyKind string

const (
	// ReplyOK carries a successful result.
	ReplyOK ReplyKind = "ok"
	// ReplyRejected means the domain refused the command.
	ReplyRejected ReplyKind = "rejected"
	// ReplyPanic means a domain handler panicked and the worker contained it.
	ReplyPanic ReplyKind = "panic"
)

// ReplyEnvelope is one command's result on the wire.
type ReplyEnvelope struct {
	RequestID uint64    `json:"rid"`
	Kind      ReplyKind `json:"k"`
	Payload   []byte    `json:"p,omitempty"`
	Message   string    `json:"m,omitempty"`
}

// EncodeRequest renders a request for postMessage.
func EncodeRequest(parseRequestID uint64, parseCommand string, parsePayload []byte) ([]byte, error) {
	if parseCommand == "" {
		return nil, errors.New("projection: a command name is required")
	}
	return json.Marshal(RequestEnvelope{RequestID: parseRequestID, Command: parseCommand, Payload: parsePayload})
}

// DecodeRequest parses a request on the worker side.
func DecodeRequest(parseBytes []byte) (RequestEnvelope, error) {
	var parseEnvelope RequestEnvelope
	if parseErr := json.Unmarshal(parseBytes, &parseEnvelope); parseErr != nil {
		return RequestEnvelope{}, fmt.Errorf("projection: decoding a request: %w", parseErr)
	}
	if parseEnvelope.RequestID == 0 {
		// A request with no id can be answered but never matched, so the caller
		// would wait for a reply that arrives and is discarded.
		return RequestEnvelope{}, errors.New("projection: a request arrived with no request id")
	}
	if parseEnvelope.Command == "" {
		return RequestEnvelope{}, fmt.Errorf("projection: request %d names no command", parseEnvelope.RequestID)
	}
	return parseEnvelope, nil
}

// EncodeReply renders a reply for postMessage.
func EncodeReply(parseReply ReplyEnvelope) ([]byte, error) {
	if parseReply.RequestID == 0 {
		return nil, errors.New("projection: a reply must echo its request id")
	}
	if parseReply.Kind == "" {
		parseReply.Kind = ReplyOK
	}
	return json.Marshal(parseReply)
}

// DeliverEncodedReply decodes a reply and routes it to its waiting request.
//
// The single place that maps a wire kind to a failure kind, so the mapping
// exists once rather than in every transport. Reports whether anything was
// waiting, which is false for a request that timed out in flight — a normal
// outcome, not an error.
func (parseClient *WorkerClient) DeliverEncodedReply(parseBytes []byte) (bool, error) {
	if parseClient == nil {
		return false, errors.New("projection: worker client is nil")
	}

	var parseReply ReplyEnvelope
	if parseErr := json.Unmarshal(parseBytes, &parseReply); parseErr != nil {
		// An undecodable reply cannot be routed to anyone, so nobody can be told
		// their command failed. Surfacing it here is the only chance to notice.
		return false, fmt.Errorf("projection: decoding a reply: %w", parseErr)
	}
	if parseReply.RequestID == 0 {
		return false, errors.New("projection: a reply arrived with no request id and cannot be matched")
	}

	switch parseReply.Kind {
	case ReplyOK, "":
		return parseClient.Deliver(parseReply.RequestID, parseReply.Payload, nil), nil
	case ReplyRejected:
		return parseClient.DeliverRejection(parseReply.RequestID, parseReply.Message), nil
	case ReplyPanic:
		return parseClient.DeliverPanic(parseReply.RequestID, parseReply.Message), nil
	default:
		// An unknown kind is not treated as success. A worker speaking a newer
		// protocol would otherwise have its failures read as results.
		return parseClient.DeliverRejection(parseReply.RequestID,
			fmt.Sprintf("the worker replied with an unrecognized kind %q", parseReply.Kind)), nil
	}
}
