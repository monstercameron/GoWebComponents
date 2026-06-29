package agentbridge

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestEnvelopeRoundTripPerKind builds one envelope of every kind, formats it,
// parses it back, and requires the decoded value to match the original.
func TestEnvelopeRoundTripPerKind(t *testing.T) {
	parsePayload := json.RawMessage(`{"appID":"fixture","buildID":"abc123"}`)
	parseCases := []struct {
		name     string
		envelope Envelope
	}{
		{"hello", BuildHelloEnvelope(1, parsePayload)},
		{"command", BuildCommandEnvelope(2, "sess-1", "bridge.snapshot", json.RawMessage(`{"budget":1024}`))},
		{"ack", BuildAckEnvelope(3, "sess-1", 2, 7, json.RawMessage(`{"nodes":[]}`))},
		{"error-ack", BuildErrorAckEnvelope(4, "sess-1", 2, ErrorCodeStaleRef, "node unmounted")},
		{"event", BuildEventEnvelope(5, "sess-1", "log", json.RawMessage(`{"severity":"error"}`))},
	}
	for _, parseCase := range parseCases {
		t.Run(parseCase.name, func(t *testing.T) {
			parseWire, parseErr := FormatEnvelopeJSON(parseCase.envelope)
			if parseErr != nil {
				t.Fatalf("format: %v", parseErr)
			}
			parseDecoded, parseErr := ParseEnvelope(parseWire)
			if parseErr != nil {
				t.Fatalf("parse: %v", parseErr)
			}
			if !reflect.DeepEqual(parseDecoded, parseCase.envelope) {
				t.Fatalf("round trip mismatch:\n got %#v\nwant %#v", parseDecoded, parseCase.envelope)
			}
		})
	}
}

// TestFormatEnvelopeJSONGolden pins the exact wire JSON of a command frame so
// field renames or marshal-order drift break loudly.
func TestFormatEnvelopeJSONGolden(t *testing.T) {
	parseWire, parseErr := FormatEnvelopeJSON(BuildCommandEnvelope(2, "sess-1", "bridge.set-atom", json.RawMessage(`{"id":"theme","value":"dark"}`)))
	if parseErr != nil {
		t.Fatalf("format: %v", parseErr)
	}
	parseWant := `{"protocol":"gwc.agentbridge","version":1,"kind":"command","seq":2,"session":"sess-1","name":"bridge.set-atom","payload":{"id":"theme","value":"dark"}}`
	if parseWire != parseWant {
		t.Fatalf("golden mismatch:\n got %s\nwant %s", parseWire, parseWant)
	}
}

// TestFormatEnvelopeJSONOmitsAckFieldsOffAcks pins that non-ack frames carry
// no ok/error/ackSeq noise on the wire.
func TestFormatEnvelopeJSONOmitsAckFieldsOffAcks(t *testing.T) {
	parseWire, parseErr := FormatEnvelopeJSON(BuildEventEnvelope(9, "sess-1", "lifecycle", nil))
	if parseErr != nil {
		t.Fatalf("format: %v", parseErr)
	}
	for _, parseField := range []string{`"ok"`, `"error"`, `"ackSeq"`, `"stateVersion"`, `"payload"`} {
		if strings.Contains(parseWire, parseField) {
			t.Fatalf("event frame leaked field %s: %s", parseField, parseWire)
		}
	}
}

// TestParseEnvelopeRejections walks every protocol-level rejection path and
// requires an actionable error mentioning the violated rule.
func TestParseEnvelopeRejections(t *testing.T) {
	parseCases := []struct {
		name    string
		wire    string
		wantSub string
	}{
		{"malformed JSON", `{"protocol":`, "malformed envelope JSON"},
		{"wrong protocol", `{"protocol":"gwc.other","version":1,"kind":"hello","seq":1}`, "unsupported protocol"},
		{"missing version", `{"protocol":"gwc.agentbridge","kind":"hello","seq":1}`, "invalid protocol version"},
		{"future version", `{"protocol":"gwc.agentbridge","version":2,"kind":"hello","seq":1}`, "newer than supported"},
		{"missing seq", `{"protocol":"gwc.agentbridge","version":1,"kind":"hello"}`, "missing seq"},
		{"unknown kind", `{"protocol":"gwc.agentbridge","version":1,"kind":"mystery","seq":1}`, "unknown envelope kind"},
		{"command without name", `{"protocol":"gwc.agentbridge","version":1,"kind":"command","seq":1}`, "command frame missing name"},
		{"event without name", `{"protocol":"gwc.agentbridge","version":1,"kind":"event","seq":1}`, "event frame missing name"},
		{"ack without ackSeq", `{"protocol":"gwc.agentbridge","version":1,"kind":"ack","seq":1,"ok":true}`, "missing ackSeq"},
		{"ack without ok", `{"protocol":"gwc.agentbridge","version":1,"kind":"ack","seq":1,"ackSeq":1}`, "missing ok"},
		{"failed ack without code", `{"protocol":"gwc.agentbridge","version":1,"kind":"ack","seq":1,"ackSeq":1,"ok":false}`, "missing structured error code"},
		{"ok ack with error", `{"protocol":"gwc.agentbridge","version":1,"kind":"ack","seq":1,"ackSeq":1,"ok":true,"error":{"code":"x"}}`, "must not carry an error"},
	}
	for _, parseCase := range parseCases {
		t.Run(parseCase.name, func(t *testing.T) {
			_, parseErr := ParseEnvelope(parseCase.wire)
			if parseErr == nil {
				t.Fatalf("expected rejection, got success")
			}
			if !strings.Contains(parseErr.Error(), parseCase.wantSub) {
				t.Fatalf("error %q does not mention %q", parseErr.Error(), parseCase.wantSub)
			}
		})
	}
}

// TestFormatEnvelopeJSONRejectsInvalid pins that the format path enforces the
// same invariants as the parse path (a hand-built bad envelope never ships).
func TestFormatEnvelopeJSONRejectsInvalid(t *testing.T) {
	parseBad := Envelope{Protocol: ProtocolName, Version: ProtocolVersion, Kind: KindCommand, Seq: 1}
	if _, parseErr := FormatEnvelopeJSON(parseBad); parseErr == nil {
		t.Fatalf("expected format rejection for command without name")
	}
}
