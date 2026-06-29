package agentbridge

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// fakeSocket is an in-memory AgentSocket for testing the client loop. Frames
// written by the client appear on the written channel; callers push frames for
// the client to read via the readable channel.
type fakeSocket struct {
	readable chan string // frames the client will read
	written  chan string // frames the client has written
	closed   chan struct{}
}

// newFakeSocket constructs a fakeSocket with buffered channels.
func newFakeSocket(parseReadBuf int, parseWriteBuf int) *fakeSocket {
	return &fakeSocket{
		readable: make(chan string, parseReadBuf),
		written:  make(chan string, parseWriteBuf),
		closed:   make(chan struct{}),
	}
}

// ReadFrame returns the next queued frame or io.EOF when the socket is closed.
func (parseFake *fakeSocket) ReadFrame() (string, error) {
	select {
	case parseFrame, parseOk := <-parseFake.readable:
		if !parseOk {
			return "", io.EOF
		}
		return parseFrame, nil
	case <-parseFake.closed:
		return "", io.EOF
	}
}

// WriteFrame records a frame in the written channel.
func (parseFake *fakeSocket) WriteFrame(parseFrame string) error {
	select {
	case <-parseFake.closed:
		return io.EOF
	default:
	}
	parseFake.written <- parseFrame
	return nil
}

// Close signals the socket as closed.
func (parseFake *fakeSocket) Close() error {
	select {
	case <-parseFake.closed:
	default:
		close(parseFake.closed)
	}
	return nil
}

// pushRead sends a frame for the client to consume.
func (parseFake *fakeSocket) pushRead(parseFrame string) {
	parseFake.readable <- parseFrame
}

// readWritten reads one written frame; blocks if none is available yet.
func (parseFake *fakeSocket) readWritten() string {
	return <-parseFake.written
}

// buildTestCommandFrame builds a valid KindCommand frame string.
func buildTestCommandFrame(parseSeq uint64, parseName string, parsePayload json.RawMessage) string {
	parseEnv := BuildCommandEnvelope(parseSeq, "", parseName, parsePayload)
	parseJSON, parseErr := FormatEnvelopeJSON(parseEnv)
	if parseErr != nil {
		panic("buildTestCommandFrame: " + parseErr.Error())
	}
	return parseJSON
}

// cleanRegistry removes all commands registered by a test.
func cleanRegistry(parseNames ...string) {
	commandRegistryMu.Lock()
	defer commandRegistryMu.Unlock()
	for _, parseName := range parseNames {
		delete(commandRegistry, parseName)
	}
}

// TestHelloIsFirstFrame verifies the hello frame is the very first frame
// written and carries the expected structure.
func TestHelloIsFirstFrame(t *testing.T) {
	parseCmd := "test.hello-probe"
	RegisterAgentCommand(parseCmd, func(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
		return json.RawMessage(`{}`), nil
	})
	defer cleanRegistry(parseCmd)

	parseSock := newFakeSocket(8, 16)
	parseClient := NewBridgeClient("app-test", "build-001")

	parseDone := make(chan struct{})
	go func() {
		defer close(parseDone)
		parseClient.RunLoop(parseSock)
	}()

	// The first written frame must be hello.
	parseFirstFrame := parseSock.readWritten()
	parseEnv, parseErr := ParseEnvelope(parseFirstFrame)
	if parseErr != nil {
		t.Fatalf("hello frame failed to parse: %v", parseErr)
	}
	if parseEnv.Kind != KindHello {
		t.Fatalf("first frame kind = %q, want %q", parseEnv.Kind, KindHello)
	}

	// Payload should decode as helloPayload with at least one command name.
	var parseHello helloPayload
	if parseDecodeErr := json.Unmarshal(parseEnv.Payload, &parseHello); parseDecodeErr != nil {
		t.Fatalf("hello payload decode: %v", parseDecodeErr)
	}
	parseCmdFound := false
	for _, parseName := range parseHello.Commands {
		if parseName == parseCmd {
			parseCmdFound = true
			break
		}
	}
	if !parseCmdFound {
		t.Errorf("hello payload.commands does not include %q; got %v", parseCmd, parseHello.Commands)
	}

	parseSock.Close()
	<-parseDone
}

// TestEventQueuedBeforeHello verifies console/error events captured before
// the socket is ready flush immediately after the hello frame.
func TestEventQueuedBeforeHello(t *testing.T) {
	parseSock := newFakeSocket(4, 16)
	parseClient := NewBridgeClient("app-test", "build-queued")
	if parseErr := parseClient.SendEvent("", "console.error", json.RawMessage(`{"message":"boom"}`)); parseErr != nil {
		t.Fatalf("SendEvent before loop returned error: %v", parseErr)
	}

	parseDone := make(chan struct{})
	go func() {
		defer close(parseDone)
		parseClient.RunLoop(parseSock)
	}()

	parseHello, parseErr := ParseEnvelope(parseSock.readWritten())
	if parseErr != nil {
		t.Fatalf("parse hello: %v", parseErr)
	}
	if parseHello.Kind != KindHello {
		t.Fatalf("first frame kind = %q, want hello", parseHello.Kind)
	}
	parseEvent, parseErr := ParseEnvelope(parseSock.readWritten())
	if parseErr != nil {
		t.Fatalf("parse queued event: %v", parseErr)
	}
	if parseEvent.Kind != KindEvent || parseEvent.Name != "console.error" {
		t.Fatalf("queued event = %#v", parseEvent)
	}
	parseSock.Close()
	<-parseDone
}

func TestSendEventConnectedWritesImmediatelyAndReportsFormatErrors(t *testing.T) {
	parseSock := newFakeSocket(1, 4)
	parseClient := NewBridgeClient("app-test", "build-live")
	parseClient.eventMu.Lock()
	parseClient.eventSock = parseSock
	parseClient.eventMu.Unlock()

	if parseErr := parseClient.SendEvent("session-1", "console.info", json.RawMessage(`{"message":"ready"}`)); parseErr != nil {
		t.Fatalf("SendEvent connected returned error: %v", parseErr)
	}
	parseEventFrame := parseSock.readWritten()
	parseEvent, parseErr := ParseEnvelope(parseEventFrame)
	if parseErr != nil {
		t.Fatalf("parse connected event: %v", parseErr)
	}
	if parseEvent.Kind != KindEvent || parseEvent.Session != "session-1" || parseEvent.Name != "console.info" {
		t.Fatalf("connected event = %#v", parseEvent)
	}

	parseErr = parseClient.SendEvent("", "", json.RawMessage(`{}`))
	if parseErr == nil {
		t.Fatal("expected SendEvent format error for empty event name")
	}
	if !strings.Contains(parseErr.Error(), "agentbridge: SendEvent") {
		t.Fatalf("SendEvent format error = %q, want wrapped SendEvent error", parseErr.Error())
	}
}

func TestSendEventBacklogDropsOldestAtCapacity(t *testing.T) {
	parseClient := NewBridgeClient("app-test", "build-backlog")
	for parseIndex := 0; parseIndex < 65; parseIndex++ {
		parsePayload := json.RawMessage(`{"index":` + strconv.Itoa(parseIndex) + `}`)
		if parseErr := parseClient.SendEvent("", "console.log", parsePayload); parseErr != nil {
			t.Fatalf("SendEvent queued index %d returned error: %v", parseIndex, parseErr)
		}
	}

	parseClient.eventMu.Lock()
	parseBacklog := append([]Envelope(nil), parseClient.eventBacklog...)
	parseClient.eventMu.Unlock()

	if len(parseBacklog) != 64 {
		t.Fatalf("backlog len = %d, want 64", len(parseBacklog))
	}
	if parseBacklog[0].Seq != 2 {
		t.Fatalf("oldest retained seq = %d, want 2 after dropping seq 1", parseBacklog[0].Seq)
	}
	if string(parseBacklog[0].Payload) != `{"index":1}` {
		t.Fatalf("oldest retained payload = %s, want index 1", string(parseBacklog[0].Payload))
	}
	if parseBacklog[len(parseBacklog)-1].Seq != 65 {
		t.Fatalf("newest retained seq = %d, want 65", parseBacklog[len(parseBacklog)-1].Seq)
	}
	if string(parseBacklog[len(parseBacklog)-1].Payload) != `{"index":64}` {
		t.Fatalf("newest retained payload = %s, want index 64", string(parseBacklog[len(parseBacklog)-1].Payload))
	}
}

// TestCommandRoundTrip verifies that a registered command produces a KindAck
// with ok=true and matching ackSeq.
func TestCommandRoundTrip(t *testing.T) {
	parseCmd := "test.echo"
	parseEchoPayload := json.RawMessage(`{"msg":"hello"}`)
	RegisterAgentCommand(parseCmd, func(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
		return parsePayload, nil
	})
	defer cleanRegistry(parseCmd)

	parseSock := newFakeSocket(4, 16)
	parseClient := NewBridgeClient("", "")
	parseCommandSeq := uint64(99)

	parseDone := make(chan struct{})
	go func() {
		defer close(parseDone)
		parseClient.RunLoop(parseSock)
	}()

	// Consume hello, then send a command and collect the ack.
	parseSock.readWritten() // hello
	parseSock.pushRead(buildTestCommandFrame(parseCommandSeq, parseCmd, parseEchoPayload))
	parseAckFrame := parseSock.readWritten()

	parseAck, parseErr := ParseEnvelope(parseAckFrame)
	if parseErr != nil {
		t.Fatalf("ack parse error: %v", parseErr)
	}
	if parseAck.Kind != KindAck {
		t.Fatalf("reply kind = %q, want KindAck", parseAck.Kind)
	}
	if parseAck.OK == nil || !*parseAck.OK {
		t.Fatalf("ack ok = %v, want true", parseAck.OK)
	}
	if parseAck.AckSeq != parseCommandSeq {
		t.Errorf("ack ackSeq = %d, want %d", parseAck.AckSeq, parseCommandSeq)
	}
	parseSock.Close()
	<-parseDone
}

// TestCommandAckCarriesRuntimeStateVersion verifies command acks report the
// bridge-visible runtime state version instead of the old zero placeholder.
func TestCommandAckCarriesRuntimeStateVersion(t *testing.T) {
	parseCmd := "test.state-version"
	parseRt := runtime.GetGlobalRuntime()
	parseBefore := parseRt.AgentStateVersion()
	RegisterAgentCommand(parseCmd, func(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
		parseRt.AdvanceAgentStateVersion()
		return json.RawMessage(`{"ok":true}`), nil
	})
	defer cleanRegistry(parseCmd)

	parseSock := newFakeSocket(4, 16)
	parseClient := NewBridgeClient("", "")
	parseCommandSeq := uint64(101)

	parseDone := make(chan struct{})
	go func() {
		defer close(parseDone)
		parseClient.RunLoop(parseSock)
	}()

	parseSock.readWritten() // hello
	parseSock.pushRead(buildTestCommandFrame(parseCommandSeq, parseCmd, nil))
	parseAckFrame := parseSock.readWritten()

	parseAck, parseErr := ParseEnvelope(parseAckFrame)
	if parseErr != nil {
		t.Fatalf("ack parse error: %v", parseErr)
	}
	if parseAck.StateVersion <= parseBefore {
		t.Fatalf("ack stateVersion = %d, want > %d", parseAck.StateVersion, parseBefore)
	}
	parseSock.Close()
	<-parseDone
}

// TestUnknownCommandAcksError verifies that an unregistered command produces a
// KindAck with ok=false and code unknown-command.
func TestUnknownCommandAcksError(t *testing.T) {
	parseSock := newFakeSocket(4, 16)
	parseClient := NewBridgeClient("", "")
	parseCommandSeq := uint64(7)
	parseUnknownCmd := "does.not.exist.xyz.agentbridge.test"

	parseDone := make(chan struct{})
	go func() {
		defer close(parseDone)
		parseClient.RunLoop(parseSock)
	}()

	parseSock.readWritten() // hello
	parseSock.pushRead(buildTestCommandFrame(parseCommandSeq, parseUnknownCmd, nil))
	parseAckFrame := parseSock.readWritten()

	parseAck, parseErr := ParseEnvelope(parseAckFrame)
	if parseErr != nil {
		t.Fatalf("parse: %v", parseErr)
	}
	if parseAck.Kind != KindAck {
		t.Fatalf("kind = %q, want KindAck", parseAck.Kind)
	}
	if parseAck.OK == nil || *parseAck.OK {
		t.Fatalf("ok = %v, want false", parseAck.OK)
	}
	if parseAck.Error == nil || parseAck.Error.Code != ErrorCodeUnknownCommand {
		t.Errorf("error.code = %v, want %q", parseAck.Error, ErrorCodeUnknownCommand)
	}
	if parseAck.AckSeq != parseCommandSeq {
		t.Errorf("ackSeq = %d, want %d", parseAck.AckSeq, parseCommandSeq)
	}
	parseSock.Close()
	<-parseDone
}

// TestMalformedFrameDoesNotStopLoop verifies that a malformed inbound frame
// does not crash or stop the loop; a subsequent valid command still gets acked.
func TestMalformedFrameDoesNotStopLoop(t *testing.T) {
	parseCmd := "test.after-malformed"
	RegisterAgentCommand(parseCmd, func(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
		return json.RawMessage(`"ok"`), nil
	})
	defer cleanRegistry(parseCmd)

	parseSock := newFakeSocket(8, 16)
	parseClient := NewBridgeClient("", "")
	parseCommandSeq := uint64(3)

	parseDone := make(chan struct{})
	go func() {
		defer close(parseDone)
		parseClient.RunLoop(parseSock)
	}()

	parseSock.readWritten() // hello
	// Push a malformed frame followed by a valid command.
	parseSock.pushRead(`not valid json at all {{{{`)
	parseSock.pushRead(buildTestCommandFrame(parseCommandSeq, parseCmd, nil))
	// Expect exactly one ack for the valid command.
	parseAckFrame := parseSock.readWritten()

	parseAck, parseErr := ParseEnvelope(parseAckFrame)
	if parseErr != nil {
		t.Fatalf("ack parse error: %v", parseErr)
	}
	if parseAck.Kind != KindAck {
		t.Fatalf("kind = %q, want KindAck", parseAck.Kind)
	}
	if parseAck.OK == nil || !*parseAck.OK {
		t.Fatalf("ok = %v, want true", parseAck.OK)
	}
	parseSock.Close()
	<-parseDone
}

// TestOutboundSeqStrictlyIncreasing verifies that across the hello frame and
// all ack frames the sequence numbers are strictly increasing.
func TestOutboundSeqStrictlyIncreasing(t *testing.T) {
	parseCmd := "test.seq-check"
	RegisterAgentCommand(parseCmd, func(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
		return nil, nil
	})
	defer cleanRegistry(parseCmd)

	parseSock := newFakeSocket(8, 32)
	parseClient := NewBridgeClient("", "")
	parseNCommands := 4

	parseDone := make(chan struct{})
	go func() {
		defer close(parseDone)
		parseClient.RunLoop(parseSock)
	}()

	// Collect hello + parseNCommands acks.
	parseCollected := make([]string, 0, parseNCommands+1)
	parseCollected = append(parseCollected, parseSock.readWritten()) // hello
	for parseI := range parseNCommands {
		parseSock.pushRead(buildTestCommandFrame(uint64(parseI+1), parseCmd, nil))
		parseCollected = append(parseCollected, parseSock.readWritten())
	}
	parseSock.Close()
	<-parseDone

	parsePrevSeq := uint64(0)
	for parseIdx, parseRawFrame := range parseCollected {
		parseEnv, parseParseErr := ParseEnvelope(parseRawFrame)
		if parseParseErr != nil {
			t.Fatalf("frame %d parse: %v", parseIdx, parseParseErr)
		}
		if parseEnv.Seq <= parsePrevSeq {
			t.Errorf("frame %d: seq %d not strictly greater than previous seq %d",
				parseIdx, parseEnv.Seq, parsePrevSeq)
		}
		parsePrevSeq = parseEnv.Seq
	}
}
