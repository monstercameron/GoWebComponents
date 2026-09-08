package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

type desktopFakeTransport struct {
	parseCapabilities  Capabilities
	parseStart         func(string, json.RawMessage) (string, error)
	parsePoll          func(string) (Reply, error)
	parseCancelCount   int
	parseListenID      string
	parseNext          func(string) (Reply, error)
	parseUnlistenCount int
	parseMutex         sync.Mutex
}

// Capabilities supports the transport contract test.
func (parseFake *desktopFakeTransport) Capabilities() (Capabilities, error) {
	return parseFake.parseCapabilities, nil
}

// Start supports the transport contract test.
func (parseFake *desktopFakeTransport) Start(parseName string, parseData json.RawMessage) (string, error) {
	if parseFake.parseStart != nil {
		return parseFake.parseStart(parseName, parseData)
	}
	return "request-1", nil
}

// Poll supports the transport contract test.
func (parseFake *desktopFakeTransport) Poll(parseID string) (Reply, error) {
	if parseFake.parsePoll != nil {
		return parseFake.parsePoll(parseID)
	}
	return Reply{Done: true, Data: json.RawMessage(`{"value":7}`)}, nil
}

// Cancel supports the transport contract test.
func (parseFake *desktopFakeTransport) Cancel(string) error {
	parseFake.parseMutex.Lock()
	parseFake.parseCancelCount++
	parseFake.parseMutex.Unlock()
	return nil
}

// Listen supports the transport contract test.
func (parseFake *desktopFakeTransport) Listen(string) (string, error) {
	parseFake.parseMutex.Lock()
	defer parseFake.parseMutex.Unlock()
	parseFake.parseListenID = "listener-1"
	return parseFake.parseListenID, nil
}

// Next supports the transport contract test.
func (parseFake *desktopFakeTransport) Next(parseID string) (Reply, error) {
	if parseFake.parseNext != nil {
		return parseFake.parseNext(parseID)
	}
	return Reply{}, nil
}

// Unlisten supports the transport contract test.
func (parseFake *desktopFakeTransport) Unlisten(string) error {
	parseFake.parseMutex.Lock()
	parseFake.parseUnlistenCount++
	parseFake.parseMutex.Unlock()
	return nil
}

// getTestTransport supports the transport contract test.
func getTestTransport() *desktopFakeTransport {
	return &desktopFakeTransport{parseCapabilities: Capabilities{Protocol: ProtocolVersion, Methods: []string{"value"}, Topics: []string{"counter.progress"}}}
}

// TestClientCallRoundTripAndCancellation supports the transport contract test.
func TestClientCallRoundTripAndCancellation(parseT *testing.T) {
	parseTransport := getTestTransport()
	parseValue, parseErr := Call[struct {
		Value int `json:"value"`
	}](context.Background(), NewClient(parseTransport), "value")
	if parseErr != nil || parseValue.Value != 7 {
		parseT.Fatalf("round trip = %#v, %v", parseValue, parseErr)
	}
	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	if _, parseErr = Call[struct{}](parseCtx, NewClient(parseTransport), "value"); !interop.IsCode(parseErr, interop.CodeCancelled) {
		parseT.Fatalf("cancel-before-call = %v", parseErr)
	}
	parseCtx2, parseCancel2 := context.WithCancel(context.Background())
	parseTransport.parsePoll = func(string) (Reply, error) { parseCancel2(); return Reply{}, nil }
	if _, parseErr = Call[struct{}](parseCtx2, NewClient(parseTransport), "value"); !interop.IsCode(parseErr, interop.CodeCancelled) {
		parseT.Fatalf("in-flight cancellation = %v", parseErr)
	}
	parseTransport.parseMutex.Lock()
	parseCancelCount := parseTransport.parseCancelCount
	parseTransport.parseMutex.Unlock()
	if parseCancelCount == 0 {
		parseT.Fatal("in-flight cancellation did not release request")
	}
}

// TestClientCallErrorsAndProtocol supports the transport contract test.
func TestClientCallErrorsAndProtocol(parseT *testing.T) {
	parseTransport := getTestTransport()
	parseTransport.parseCapabilities.Protocol++
	if _, parseErr := Call[string](context.Background(), NewClient(parseTransport), "value"); !interop.IsCode(parseErr, interop.CodeInvalid) {
		parseT.Fatalf("protocol mismatch = %v", parseErr)
	}
	parseTransport = getTestTransport()
	if _, parseErr := Call[string](context.Background(), NewClient(parseTransport), "missing"); !interop.IsCode(parseErr, interop.CodeMissingExport) {
		parseT.Fatalf("missing method = %v", parseErr)
	}
	parseTransport.parsePoll = func(string) (Reply, error) { return Reply{Done: true, Data: json.RawMessage(`{"value":`)}, nil }
	if _, parseErr := Call[struct{ Value int }](context.Background(), NewClient(parseTransport), "value"); !interop.IsCode(parseErr, interop.CodeDecode) {
		parseT.Fatalf("decode error = %v", parseErr)
	}
	parseTransport.parseStart = func(string, json.RawMessage) (string, error) { return "", errors.New("remote failed") }
	if _, parseErr := Call[string](context.Background(), NewClient(parseTransport), "value"); parseErr == nil {
		parseT.Fatal("remote start failure was swallowed")
	}
	parseTransport = getTestTransport()
	parseCtx, parseCancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer parseCancel()
	// Timer delivery may be delayed under concurrent test load; wait for the
	// cancelled-context precondition instead of assuming a short sleep establishes it.
	select {
	case <-parseCtx.Done():
	case <-time.After(time.Second):
		parseT.Fatal("deadline notification was not delivered")
	}
	if _, parseErr := Call[string](parseCtx, NewClient(parseTransport), "value"); !interop.IsCode(parseErr, interop.CodeTimeout) {
		parseT.Fatalf("deadline = %v", parseErr)
	}
}

// TestClientCancellationDuringPollRejectsLateSuccess ensures cancellation wins at the return boundary.
func TestClientCancellationDuringPollRejectsLateSuccess(parseT *testing.T) {
	parseTransport := getTestTransport()
	parseContext, parseCancel := context.WithCancel(context.Background())
	defer parseCancel()
	parseTransport.parsePoll = func(string) (Reply, error) {
		parseCancel()
		return Reply{Done: true, Data: json.RawMessage("7")}, nil
	}
	if _, parseErr := Call[int](parseContext, NewClient(parseTransport), "value"); !interop.IsCode(parseErr, interop.CodeCancelled) {
		parseT.Fatalf("late success escaped cancellation: %v", parseErr)
	}
}

// TestClientWireRejectsUnsafeIntegers prevents irreversible precision loss before JavaScript receives a payload.
func TestClientWireRejectsUnsafeIntegers(parseT *testing.T) {
	parseTransport := getTestTransport()
	parseTransport.parseStart = func(string, json.RawMessage) (string, error) {
		parseT.Error("unsafe input reached transport")
		return "", nil
	}
	for _, parseValue := range []any{int64(9007199254740993), uint64(18446744073709551615), map[string]any{"nested": []any{int64(-9007199254740993)}}} {
		if _, parseErr := Call[any](context.Background(), NewClient(parseTransport), "value", parseValue); !interop.IsCode(parseErr, interop.CodeEncode) {
			parseT.Fatalf("unsafe wire input %v: %v", parseValue, parseErr)
		}
	}
}

// TestSubscribeReceivesAndStops checks actual delivery before idempotent cancellation.
func TestSubscribeReceivesAndStops(parseT *testing.T) {
	parseTransport := getTestTransport()
	parseTransport.parseNext = func(string) (Reply, error) { return Reply{Done: true, Data: json.RawMessage("50")}, nil }
	parseReceived := make(chan int, 8)
	parseStop, parseErr := Subscribe[int](context.Background(), NewClient(parseTransport), "counter.progress", func(parseValue int, parseErr error) {
		if parseErr != nil {
			parseT.Errorf("event: %v", parseErr)
			return
		}
		select {
		case parseReceived <- parseValue:
		default:
		}
	})
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	defer parseStop()
	select {
	case parseValue := <-parseReceived:
		if parseValue != 50 {
			parseT.Fatalf("event = %d", parseValue)
		}
	case <-time.After(time.Second):
		parseT.Fatal("no event delivered")
	}
	parseStop()
	parseStop()
	parseTransport.parseMutex.Lock()
	parseCount := parseTransport.parseUnlistenCount
	parseTransport.parseMutex.Unlock()
	if parseCount != 1 {
		parseT.Fatalf("unlisten=%d", parseCount)
	}
}

// TestSubscribeContainsHandlerPanicAndReleases verifies failure isolation also performs native unsubscribe.
func TestSubscribeContainsHandlerPanicAndReleases(parseT *testing.T) {
	parseTransport := getTestTransport()
	parseTransport.parseNext = func(string) (Reply, error) { return Reply{Done: true, Data: json.RawMessage("1")}, nil }
	parseStop, parseErr := Subscribe[int](context.Background(), NewClient(parseTransport), "counter.progress", func(int, error) { panic("test subscription panic") })
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	defer parseStop()
	parseDeadline := time.Now().Add(time.Second)
	for {
		parseTransport.parseMutex.Lock()
		parseCount := parseTransport.parseUnlistenCount
		parseTransport.parseMutex.Unlock()
		if parseCount == 1 {
			return
		}
		if time.Now().After(parseDeadline) {
			parseT.Fatal("panicking handler did not release its subscription")
		}
		time.Sleep(time.Millisecond)
	}
}
