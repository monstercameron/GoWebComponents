package desktop

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/interop"
)

type parseNativeTransport struct {
	parseMethods []string
	parseReplies map[string]Reply
}

// Capabilities returns all native methods used by the envelope tests.
func (parseTransport *parseNativeTransport) Capabilities() (Capabilities, error) {
	return Capabilities{Protocol: ProtocolVersion, Methods: parseTransport.parseMethods}, nil
}

// Start records a request and prepares its deterministic reply.
func (parseTransport *parseNativeTransport) Start(parseMethod string, parseData json.RawMessage) (string, error) {
	var parseArguments []json.RawMessage
	if parseErr := json.Unmarshal(parseData, &parseArguments); parseErr != nil || len(parseArguments) != 1 {
		return "", parseErr
	}
	var parseRequest NativeRequest
	if parseErr := json.Unmarshal(parseArguments[0], &parseRequest); parseErr != nil {
		return "", parseErr
	}
	parseReply, parseFound := parseTransport.parseReplies[parseMethod]
	if !parseFound {
		parseReply = Reply{Done: true, Code: interop.CodeRemote, Message: "missing"}
	}
	parseTransport.parseReplies[parseMethod] = parseReply
	return parseMethod, nil
}

// Poll returns the prepared reply.
func (parseTransport *parseNativeTransport) Poll(parseID string) (Reply, error) {
	return parseTransport.parseReplies[parseID], nil
}

// Cancel completes test transport cleanup.
func (*parseNativeTransport) Cancel(string) error { return nil }

// Listen rejects unused event operations.
func (*parseNativeTransport) Listen(string) (string, error) { return "", nil }

// Next returns an empty event reply.
func (*parseNativeTransport) Next(string) (Reply, error) { return Reply{}, nil }

// Unlisten completes test transport cleanup.
func (*parseNativeTransport) Unlisten(string) error { return nil }

// TestNativeClientEnvelopeWorkflows verifies typed wrappers decode host envelopes and preserve codes.
func TestNativeClientEnvelopeWorkflows(parseTest *testing.T) {
	parseJSON := func(parseValue any) json.RawMessage { parseData, _ := json.Marshal(parseValue); return parseData }
	parseTransport := &parseNativeTransport{parseMethods: []string{ClipboardWriteMethod, ClipboardReadMethod, MessageMethod, WindowMethod, ScreensMethod}, parseReplies: map[string]Reply{
		ClipboardWriteMethod: {Done: true, Data: parseJSON(NativeReply{Version: NativeContractVersion})},
		ClipboardReadMethod:  {Done: true, Data: parseJSON(NativeReply{Version: NativeContractVersion, Data: parseJSON("fixture")})},
		MessageMethod:        {Done: true, Data: parseJSON(NativeReply{Version: NativeContractVersion, Data: parseJSON(MessageReply{Button: "Yes"})})},
		WindowMethod:         {Done: true, Data: parseJSON(NativeReply{Version: NativeContractVersion, Data: parseJSON(WindowInfo{ID: "id", Width: 10, Height: 10})})},
		ScreensMethod:        {Done: true, Data: parseJSON(NativeReply{Version: NativeContractVersion, Data: parseJSON([]ScreenInfo{{ID: "screen"}})})},
	}}
	parseClient := NewClient(parseTransport)
	if parseErr := parseClient.WriteClipboard(context.Background(), "text"); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseValue, parseErr := parseClient.ReadClipboard(context.Background()); parseErr != nil || parseValue != "fixture" {
		parseTest.Fatalf("clipboard=%q err=%v", parseValue, parseErr)
	}
	if parseValue, parseErr := parseClient.ShowMessage(context.Background(), MessageRequest{Kind: "question"}); parseErr != nil || parseValue.Button != "Yes" {
		parseTest.Fatalf("message=%+v err=%v", parseValue, parseErr)
	}
	if parseValue, parseErr := parseClient.ControlWindow(context.Background(), WindowRequest{Action: "info"}); parseErr != nil || parseValue.ID != "id" {
		parseTest.Fatalf("window=%+v err=%v", parseValue, parseErr)
	}
	if parseValue, parseErr := parseClient.ListScreens(context.Background()); parseErr != nil || len(parseValue) != 1 {
		parseTest.Fatalf("screens=%+v err=%v", parseValue, parseErr)
	}
}

// TestNativeClientDisabledFeature verifies Require denies direct unavailable calls.
func TestNativeClientDisabledFeature(parseTest *testing.T) {
	parseClient := NewClient(&parseNativeTransport{parseMethods: nil, parseReplies: map[string]Reply{}})
	if _, parseErr := parseClient.ReadClipboard(context.Background()); !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseTest.Fatalf("error=%v", parseErr)
	}
}
