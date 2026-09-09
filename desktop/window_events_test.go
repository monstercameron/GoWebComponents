package desktop

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

type parseWindowEventTransport struct {
	parseCapabilities Capabilities
	parseNext         Reply
	parseUnlistened   chan string
	parseOnce         sync.Once
	parseListenCalls  int
}

func (parseTransport *parseWindowEventTransport) Capabilities() (Capabilities, error) {
	return parseTransport.parseCapabilities, nil
}
func (*parseWindowEventTransport) Start(string, json.RawMessage) (string, error) { return "", nil }
func (*parseWindowEventTransport) Poll(string) (Reply, error)                    { return Reply{}, nil }
func (*parseWindowEventTransport) Cancel(string) error                           { return nil }
func (parseTransport *parseWindowEventTransport) Listen(parseTopic string) (string, error) {
	parseTransport.parseListenCalls++
	return parseTopic, nil
}
func (parseTransport *parseWindowEventTransport) Next(string) (Reply, error) {
	parseReply := Reply{}
	parseTransport.parseOnce.Do(func() { parseReply = parseTransport.parseNext })
	return parseReply, nil
}
func (parseTransport *parseWindowEventTransport) Unlisten(parseID string) error {
	select {
	case parseTransport.parseUnlistened <- parseID:
	default:
	}
	return nil
}

// TestSubscribeWindowEventsRequiresFeatureAndTopic verifies both independent capability gates.
func TestSubscribeWindowEventsRequiresFeatureAndTopic(parseTest *testing.T) {
	parseTransport := &parseWindowEventTransport{parseCapabilities: Capabilities{Protocol: ProtocolVersion, Topics: []string{WindowEventTopic}}, parseUnlistened: make(chan string, 1)}
	if _, parseErr := NewClient(parseTransport).SubscribeWindowEvents(context.Background(), func(WindowEvent, error) {}); !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseTest.Fatalf("feature gate error=%v", parseErr)
	}
	parseTransport.parseCapabilities.Features = []Feature{WindowEvents}
	parseTransport.parseCapabilities.Topics = nil
	if _, parseErr := NewClient(parseTransport).SubscribeWindowEvents(context.Background(), func(WindowEvent, error) {}); !interop.IsCode(parseErr, interop.CodeMissingExport) {
		parseTest.Fatalf("topic gate error=%v", parseErr)
	}
}

// TestSubscribeWindowDropsRequiresExplicitTopic verifies drag/drop cannot be inferred from the base feature.
func TestSubscribeWindowDropsRequiresExplicitTopic(parseTest *testing.T) {
	parseTransport := &parseWindowEventTransport{parseCapabilities: Capabilities{Protocol: ProtocolVersion, Features: []Feature{WindowEvents}, Topics: []string{WindowEventTopic}}, parseUnlistened: make(chan string, 1)}
	if _, parseErr := NewClient(parseTransport).SubscribeWindowDrops(context.Background(), func(WindowEvent, error) {}); !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseTest.Fatalf("drop gate error=%v", parseErr)
	}
}

// TestSubscribeWindowEventsDeliversTypedPayloadAndCleansUp verifies decoding and idempotent lifecycle cleanup.
func TestSubscribeWindowEventsDeliversTypedPayloadAndCleansUp(parseTest *testing.T) {
	parseData, _ := json.Marshal(WindowEvent{Kind: WindowResized, WindowID: "caller", Width: 640, Height: 480})
	parseTransport := &parseWindowEventTransport{parseCapabilities: Capabilities{Protocol: ProtocolVersion, Features: []Feature{WindowEvents}, Topics: []string{WindowEventTopic}}, parseNext: Reply{Done: true, Data: parseData}, parseUnlistened: make(chan string, 2)}
	parseDelivered := make(chan WindowEvent, 1)
	parseStop, parseErr := NewClient(parseTransport).SubscribeWindowEvents(context.Background(), func(parseEvent WindowEvent, parseEventErr error) {
		if parseEventErr == nil {
			parseDelivered <- parseEvent
		}
	})
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	select {
	case parseEvent := <-parseDelivered:
		if parseEvent.WindowID != "caller" || parseEvent.Kind != WindowResized || parseEvent.Width != 640 {
			parseTest.Fatalf("event=%+v", parseEvent)
		}
	case <-time.After(time.Second):
		parseTest.Fatal("event not delivered")
	}
	parseStop()
	parseStop()
	select {
	case parseID := <-parseTransport.parseUnlistened:
		if parseID != WindowEventTopic {
			parseTest.Fatalf("unlisten=%q", parseID)
		}
	case <-time.After(time.Second):
		parseTest.Fatal("subscription not cleaned up")
	}
	select {
	case parseID := <-parseTransport.parseUnlistened:
		parseTest.Fatalf("duplicate unlisten=%q", parseID)
	case <-time.After(40 * time.Millisecond):
	}
}

// TestWindowEventTopicsReturnsFreshExplicitAllowlist verifies callers cannot mutate later capability results.
func TestWindowEventTopicsReturnsFreshExplicitAllowlist(parseTest *testing.T) {
	parseTopics := WindowEventTopics(true)
	parseTopics[0] = "raw.event"
	parseNext := WindowEventTopics(false)
	if len(parseNext) != 1 || parseNext[0] != WindowEventTopic {
		parseTest.Fatalf("topics=%v", parseNext)
	}
}

// TestValidateWindowEventRejectsRawKindsAndCrossTopicFiles verifies native payloads cannot expand the allowlist.
func TestValidateWindowEventRejectsRawKindsAndCrossTopicFiles(parseTest *testing.T) {
	parseCases := []WindowEvent{
		{Kind: WindowEventKind("windows:1243"), WindowID: "caller"},
		{Kind: WindowFocused, WindowID: "caller", Files: []string{"secret.txt"}},
		{Kind: WindowResized, WindowID: "", Width: 100, Height: 100},
	}
	for _, parseEvent := range parseCases {
		if parseErr := validateWindowEvent(parseEvent, false); !interop.IsCode(parseErr, interop.CodeDecode) {
			parseTest.Fatalf("event=%+v error=%v", parseEvent, parseErr)
		}
	}
	if parseErr := validateWindowEvent(WindowEvent{Kind: WindowFilesDropped, WindowID: "caller", Files: make([]string, 33)}, true); !interop.IsCode(parseErr, interop.CodeDecode) {
		parseTest.Fatalf("oversized drop error=%v", parseErr)
	}
}

// TestWindowEventSubscriptionsRejectNilHandlersBeforeListening verifies wrapper handlers cannot panic later.
func TestWindowEventSubscriptionsRejectNilHandlersBeforeListening(parseTest *testing.T) {
	parseTransport := &parseWindowEventTransport{parseCapabilities: Capabilities{Protocol: ProtocolVersion, Features: []Feature{WindowEvents}, Topics: WindowEventTopics(true)}, parseUnlistened: make(chan string, 1)}
	parseClient := NewClient(parseTransport)
	if _, parseErr := parseClient.SubscribeWindowEvents(context.Background(), nil); !interop.IsCode(parseErr, interop.CodeInvalid) {
		parseTest.Fatalf("window event nil handler error=%v", parseErr)
	}
	if _, parseErr := parseClient.SubscribeWindowDrops(context.Background(), nil); !interop.IsCode(parseErr, interop.CodeInvalid) {
		parseTest.Fatalf("window drop nil handler error=%v", parseErr)
	}
	if parseTransport.parseListenCalls != 0 {
		parseTest.Fatalf("Listen called %d times", parseTransport.parseListenCalls)
	}
}
