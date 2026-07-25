package projection_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/projection"
)

// The wire envelope between app.wasm and services.wasm.
//
// Two binaries have to agree on this, so the risk is drift: a field renamed on
// one side, a kind added on the other. These tests pin the shape and, more
// importantly, pin what happens when the two sides DISAGREE — because that is
// what a version skew between artifacts actually produces.

// envelopePoster encodes requests and hands them straight to a worker function,
// exercising the full encode/decode path rather than the client alone.
type envelopePoster struct {
	client *projection.WorkerClient
	handle func(projection.RequestEnvelope) projection.ReplyEnvelope
	// lastEncoded keeps the bytes so a test can inspect the wire form.
	lastEncoded []byte
}

func (parsePoster *envelopePoster) Post(parseRequestID uint64, parseName string, parseRequest []byte) error {
	parseEncoded, parseErr := projection.EncodeRequest(parseRequestID, parseName, parseRequest)
	if parseErr != nil {
		return parseErr
	}
	parsePoster.lastEncoded = parseEncoded

	go func() {
		parseDecoded, parseDecodeErr := projection.DecodeRequest(parseEncoded)
		if parseDecodeErr != nil {
			return
		}
		parseReply := parsePoster.handle(parseDecoded)
		parseReplyBytes, parseReplyErr := projection.EncodeReply(parseReply)
		if parseReplyErr != nil {
			return
		}
		parsePoster.client.DeliverEncodedReply(parseReplyBytes)
	}()
	return nil
}

// TestARequestSurvivesTheRoundTrip is the happy path across both binaries.
func TestARequestSurvivesTheRoundTrip(parseT *testing.T) {
	parsePoster := &envelopePoster{
		handle: func(parseRequest projection.RequestEnvelope) projection.ReplyEnvelope {
			return projection.ReplyEnvelope{
				RequestID: parseRequest.RequestID,
				Kind:      projection.ReplyOK,
				Payload:   []byte(`{"id":11}`),
			}
		},
	}
	parseClient, parseErr := projection.NewWorkerClient(parsePoster)
	if parseErr != nil {
		parseT.Fatalf("NewWorkerClient: %v", parseErr)
	}
	parsePoster.client = parseClient

	parseResult, parseInvokeErr := addItem.Invoke(context.Background(), parseClient, jsonCodec{}, addItemArgs{Name: "widget"})
	if parseInvokeErr != nil {
		parseT.Fatalf("Invoke: %v", parseInvokeErr)
	}
	if parseResult.ID != 11 {
		parseT.Errorf("result = %+v, want id 11", parseResult)
	}

	// The command name and the arguments must both reach the worker.
	var parseSent projection.RequestEnvelope
	if parseErr := json.Unmarshal(parsePoster.lastEncoded, &parseSent); parseErr != nil {
		parseT.Fatalf("the wire form does not decode: %v", parseErr)
	}
	if parseSent.Command != "addItem" {
		parseT.Errorf("command = %q, want addItem", parseSent.Command)
	}
	if parseSent.RequestID == 0 {
		parseT.Error("a request with no id can be answered but never matched")
	}
}

// TestReplyKindsMapToFailureKinds is the mapping that must exist in exactly one
// place, because getting it wrong turns a refusal into a retry storm.
func TestReplyKindsMapToFailureKinds(parseT *testing.T) {
	for _, parseCase := range []struct {
		label string
		kind  projection.ReplyKind
		want  projection.FailureKind
	}{
		{"rejection", projection.ReplyRejected, projection.FailureRejected},
		{"contained panic", projection.ReplyPanic, projection.FailureRejected},
	} {
		parsePoster := &envelopePoster{
			handle: func(parseRequest projection.RequestEnvelope) projection.ReplyEnvelope {
				return projection.ReplyEnvelope{
					RequestID: parseRequest.RequestID,
					Kind:      parseCase.kind,
					Message:   "the domain said no",
				}
			},
		}
		parseClient, _ := projection.NewWorkerClient(parsePoster)
		parsePoster.client = parseClient

		_, parseErr := parseClient.Send(context.Background(), "cmd", nil)
		if parseErr == nil {
			parseT.Errorf("%s: expected a failure", parseCase.label)
			continue
		}
		if parseKind := projection.Classify(parseErr); parseKind != parseCase.want {
			parseT.Errorf("%s: kind = %s, want %s", parseCase.label, parseKind, parseCase.want)
		}
	}
}

// TestAnUnknownReplyKindIsNotTreatedAsSuccess is the version-skew case.
//
// A worker speaking a newer protocol would otherwise have its failures read as
// results, and the caller would act on an empty payload as though the command
// had worked.
func TestAnUnknownReplyKindIsNotTreatedAsSuccess(parseT *testing.T) {
	parsePoster := &envelopePoster{
		handle: func(parseRequest projection.RequestEnvelope) projection.ReplyEnvelope {
			return projection.ReplyEnvelope{
				RequestID: parseRequest.RequestID,
				Kind:      projection.ReplyKind("throttled"),
				Message:   "slow down",
			}
		},
	}
	parseClient, _ := projection.NewWorkerClient(parsePoster)
	parsePoster.client = parseClient

	if _, parseErr := parseClient.Send(context.Background(), "cmd", nil); parseErr == nil {
		parseT.Fatal("an unrecognized reply kind must not be read as success")
	}
}

// TestAnUndecodableReplyIsReported: it cannot be routed to anyone, so nobody can
// be told their command failed. Surfacing it at the decode is the only chance to
// notice.
func TestAnUndecodableReplyIsReported(parseT *testing.T) {
	parseClient, _ := projection.NewWorkerClient(&recordingPoster{})

	hasDelivered, parseErr := parseClient.DeliverEncodedReply([]byte("not json"))
	if parseErr == nil {
		parseT.Error("an undecodable reply must be reported")
	}
	if hasDelivered {
		parseT.Error("nothing was delivered")
	}

	if _, parseErr := parseClient.DeliverEncodedReply([]byte(`{"k":"ok"}`)); parseErr == nil {
		parseT.Error("a reply with no request id cannot be matched and must be reported")
	}
}

// TestTheWorkerSideRejectsUnmatchableRequests is the same guard from the other
// direction: a request with no id can be answered but never matched, so the
// caller waits for a reply that arrives and is discarded.
func TestTheWorkerSideRejectsUnmatchableRequests(parseT *testing.T) {
	if _, parseErr := projection.DecodeRequest([]byte(`{"cmd":"doThing"}`)); parseErr == nil {
		parseT.Error("a request with no id must be rejected on the worker side")
	}
	if _, parseErr := projection.DecodeRequest([]byte(`{"rid":7}`)); parseErr == nil {
		parseT.Error("a request naming no command must be rejected")
	}
	if _, parseErr := projection.DecodeRequest([]byte("garbage")); parseErr == nil {
		parseT.Error("an undecodable request must be rejected")
	}
}

func TestEncodeRejectsUnmatchableEnvelopes(parseT *testing.T) {
	if _, parseErr := projection.EncodeRequest(1, "", nil); parseErr == nil {
		parseT.Error("a request with no command must be rejected at encode")
	}
	if _, parseErr := projection.EncodeReply(projection.ReplyEnvelope{Kind: projection.ReplyOK}); parseErr == nil {
		parseT.Error("a reply that echoes no request id must be rejected at encode")
	}
}

// TestAnEmptyReplyKindDefaultsToOK keeps a minimal worker honest: omitting the
// kind on a successful reply is the obvious thing to do and must mean success
// rather than an unrecognized kind.
func TestAnEmptyReplyKindDefaultsToOK(parseT *testing.T) {
	parseEncoded, parseErr := projection.EncodeReply(projection.ReplyEnvelope{RequestID: 3, Payload: []byte("x")})
	if parseErr != nil {
		parseT.Fatalf("EncodeReply: %v", parseErr)
	}

	var parseDecoded projection.ReplyEnvelope
	if parseErr := json.Unmarshal(parseEncoded, &parseDecoded); parseErr != nil {
		parseT.Fatalf("unmarshal: %v", parseErr)
	}
	if parseDecoded.Kind != projection.ReplyOK {
		parseT.Errorf("kind = %q, want it defaulted to ok", parseDecoded.Kind)
	}
}

// TestPayloadsAreOpaqueToTheEnvelope: the envelope must not need to understand
// what it carries, or every new command shape would require a protocol change.
func TestPayloadsAreOpaqueToTheEnvelope(parseT *testing.T) {
	parseBinary := []byte{0, 1, 2, 255, 254}

	parseEncoded, parseErr := projection.EncodeRequest(5, "cmd", parseBinary)
	if parseErr != nil {
		parseT.Fatalf("EncodeRequest: %v", parseErr)
	}
	parseDecoded, parseDecodeErr := projection.DecodeRequest(parseEncoded)
	if parseDecodeErr != nil {
		parseT.Fatalf("DecodeRequest: %v", parseDecodeErr)
	}
	if len(parseDecoded.Payload) != len(parseBinary) {
		parseT.Fatalf("payload length = %d, want %d", len(parseDecoded.Payload), len(parseBinary))
	}
	for parseIndex := range parseBinary {
		if parseDecoded.Payload[parseIndex] != parseBinary[parseIndex] {
			parseT.Fatalf("payload byte %d = %d, want %d",
				parseIndex, parseDecoded.Payload[parseIndex], parseBinary[parseIndex])
		}
	}
}
