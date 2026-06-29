//go:build !js || !wasm

package fetch

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/ui"
)

type realtimeTestTransport struct {
	parseMu      sync.Mutex
	parseSent    []string
	parseSendErr error
	parseClosed  int
}

func (parseT *realtimeTestTransport) send(parseMessage string) error {
	parseT.parseMu.Lock()
	defer parseT.parseMu.Unlock()
	if parseT.parseSendErr != nil {
		return parseT.parseSendErr
	}
	parseT.parseSent = append(parseT.parseSent, parseMessage)
	return nil
}

func (parseT *realtimeTestTransport) close() error {
	parseT.parseMu.Lock()
	defer parseT.parseMu.Unlock()
	parseT.parseClosed++
	return nil
}

func (parseT *realtimeTestTransport) sent() []string {
	parseT.parseMu.Lock()
	defer parseT.parseMu.Unlock()
	return append([]string(nil), parseT.parseSent...)
}

func (parseT *realtimeTestTransport) closeCount() int {
	parseT.parseMu.Lock()
	defer parseT.parseMu.Unlock()
	return parseT.parseClosed
}

type realtimeTestOpenCall struct {
	parseURL       string
	parseOptions   realtimeResolvedOptions
	parseCallbacks realtimeTransportCallbacks
	parseTransport *realtimeTestTransport
}

func newRealtimeTestController(parseAPI string, parseOptions realtimeResolvedOptions) (*realtimeConnectionController, ui.State[RealtimeState], *[]realtimeTestOpenCall) {
	parseState := ui.UseState(RealtimeState{Status: RealtimeIdle, Supported: true})
	parseCalls := []realtimeTestOpenCall{}
	parseController := &realtimeConnectionController{
		api:     parseAPI,
		url:     "wss://example.test/live",
		options: parseOptions,
		state:   parseState,
		openTransport: func(parseURL string, parseOptions realtimeResolvedOptions, parseCallbacks realtimeTransportCallbacks) (realtimeTransport, error) {
			parseTransport := &realtimeTestTransport{}
			parseCalls = append(parseCalls, realtimeTestOpenCall{
				parseURL:       parseURL,
				parseOptions:   parseOptions,
				parseCallbacks: parseCallbacks,
				parseTransport: parseTransport,
			})
			return parseTransport, nil
		},
	}
	return parseController, parseState, &parseCalls
}

func TestRealtimeControllerStateMachineCallbacksAndStaleEvents(parseT *testing.T) {
	parseNow := time.Date(2026, time.June, 13, 9, 0, 0, 0, time.UTC)
	parseController, parseState, parseCalls := newRealtimeTestController("WebSocket", realtimeResolvedOptions{
		reconnect:      true,
		maxReconnects:  2,
		initialBackoff: time.Hour,
		maxBackoff:     time.Hour,
		backoffFactor:  2,
		maxMessages:    2,
		maxErrors:      2,
		now:            func() time.Time { return parseNow },
	})

	parseController.start()
	if len(*parseCalls) != 1 {
		parseT.Fatalf("expected one transport open, got %d", len(*parseCalls))
	}
	if parseState.Get().Status != RealtimeConnecting || !parseState.Get().Connecting || parseState.Get().ConnectAttempts != 1 {
		parseT.Fatalf("expected connecting state after start, got %+v", parseState.Get())
	}

	parseCallbacks := (*parseCalls)[0].parseCallbacks
	parseCallbacks.handleOpen()
	if parseGot := parseState.Get(); parseGot.Status != RealtimeOpen || !parseGot.Open || parseGot.LastOpenAt != parseNow {
		parseT.Fatalf("expected open state, got %+v", parseGot)
	}

	parseCallbacks.handleMessage(RealtimeMessage{Data: "one", Type: "text"})
	parseCallbacks.handleMessage(RealtimeMessage{Data: "two", Type: "text"})
	parseCallbacks.handleMessage(RealtimeMessage{Data: "three", Type: "text"})
	parseGot := parseState.Get()
	if parseGot.LastMessage.Data != "three" || parseGot.LastMessage.ReceivedAt != parseNow {
		parseT.Fatalf("expected last message timestamp to be filled, got %+v", parseGot.LastMessage)
	}
	if parseMessages := []string{parseGot.Messages[0].Data, parseGot.Messages[1].Data}; !reflect.DeepEqual(parseMessages, []string{"two", "three"}) {
		parseT.Fatalf("expected bounded newest messages, got %+v", parseGot.Messages)
	}

	parseCallbacks.handleError(errors.New("transient"))
	parseCallbacks.handleError(nil)
	if parseGot := parseState.Get(); len(parseGot.Errors) != 1 || parseGot.Error == nil || parseGot.Errors[0].Message != "transient" {
		parseT.Fatalf("expected one recorded transport error, got %+v", parseGot)
	}

	parseController.start()
	if len(*parseCalls) != 2 {
		parseT.Fatalf("expected restart to open a second transport, got %d", len(*parseCalls))
	}
	parseCallbacks.handleMessage(RealtimeMessage{Data: "stale"})
	if parseGot := parseState.Get(); parseGot.LastMessage.Data != "three" {
		parseT.Fatalf("expected stale callback to be ignored, got %+v", parseGot.LastMessage)
	}
	if (*parseCalls)[0].parseTransport.closeCount() != 1 {
		parseT.Fatalf("expected restart to close stale transport once, got %d", (*parseCalls)[0].parseTransport.closeCount())
	}
}

func TestRealtimeControllerReconnectBackoffAndMaxAttempts(parseT *testing.T) {
	parseNow := time.Date(2026, time.June, 13, 10, 0, 0, 0, time.UTC)
	parseController, parseState, parseCalls := newRealtimeTestController("WebSocket", realtimeResolvedOptions{
		reconnect:      true,
		maxReconnects:  2,
		initialBackoff: 5 * time.Millisecond,
		maxBackoff:     20 * time.Millisecond,
		backoffFactor:  2,
		maxMessages:    4,
		maxErrors:      4,
		now:            func() time.Time { return parseNow },
	})
	parseT.Cleanup(func() { parseController.stop(true) })

	parseController.start()
	(*parseCalls)[0].parseCallbacks.handleOpen()
	(*parseCalls)[0].parseCallbacks.handleClose()
	parseGot := parseState.Get()
	if parseGot.Status != RealtimeReconnecting || parseGot.ReconnectAttempts != 1 || parseGot.Reconnects != 1 || parseGot.NextReconnectAt != parseNow.Add(5*time.Millisecond) {
		parseT.Fatalf("expected first reconnect with initial backoff, got %+v", parseGot)
	}

	waitFetchTestCondition(parseT, time.Second, func() bool {
		return len(*parseCalls) == 2
	})
	(*parseCalls)[1].parseCallbacks.handleOpen()
	(*parseCalls)[1].parseCallbacks.handleClose()
	parseGot = parseState.Get()
	if parseGot.Status != RealtimeReconnecting || parseGot.ReconnectAttempts != 1 || parseGot.Reconnects != 2 || parseGot.NextReconnectAt != parseNow.Add(5*time.Millisecond) {
		parseT.Fatalf("expected reconnect attempts to reset after open, got %+v", parseGot)
	}

	waitFetchTestCondition(parseT, time.Second, func() bool {
		return len(*parseCalls) == 3
	})
	(*parseCalls)[2].parseCallbacks.handleClose()
	parseGot = parseState.Get()
	if parseGot.Status != RealtimeReconnecting || parseGot.ReconnectAttempts != 2 || parseGot.NextReconnectAt != parseNow.Add(10*time.Millisecond) {
		parseT.Fatalf("expected second consecutive reconnect to use exponential backoff, got %+v", parseGot)
	}

	waitFetchTestCondition(parseT, time.Second, func() bool {
		return len(*parseCalls) == 4
	})
	parseErr := errors.New("open failed")
	parseController.scheduleReconnect(parseErr)
	parseGot = parseState.Get()
	if parseGot.Status != RealtimeClosed || !parseGot.Closed || parseGot.Error != parseErr || !parseGot.NextReconnectAt.IsZero() {
		parseT.Fatalf("expected max reconnect attempts to close permanently, got %+v", parseGot)
	}
}

func TestRealtimeControllerStopAndCloseCallbacksAreIdempotent(parseT *testing.T) {
	parseController, parseState, parseCalls := newRealtimeTestController("WebSocket", realtimeResolvedOptions{
		reconnect:      true,
		maxReconnects:  1,
		initialBackoff: time.Hour,
		maxBackoff:     time.Hour,
		backoffFactor:  2,
		maxMessages:    4,
		maxErrors:      4,
		now:            time.Now,
	})

	parseController.start()
	(*parseCalls)[0].parseCallbacks.handleOpen()
	parseController.stop(true)
	parseController.stop(true)
	if parseClosed := (*parseCalls)[0].parseTransport.closeCount(); parseClosed != 1 {
		parseT.Fatalf("expected manual stop to close transport once, got %d", parseClosed)
	}
	if parseGot := parseState.Get(); parseGot.Status != RealtimeClosed || !parseGot.Closed || !parseGot.NextReconnectAt.IsZero() {
		parseT.Fatalf("expected closed state after manual stop, got %+v", parseGot)
	}

	parseController.start()
	if len(*parseCalls) != 2 {
		parseT.Fatalf("expected reopen after manual stop, got %d opens", len(*parseCalls))
	}
	(*parseCalls)[1].parseCallbacks.handleOpen()
	(*parseCalls)[1].parseCallbacks.handleClose()
	(*parseCalls)[1].parseCallbacks.handleClose()
	waitFetchTestCondition(parseT, time.Second, func() bool {
		return (*parseCalls)[1].parseTransport.closeCount() == 1
	})
	if parseGot := parseState.Get(); parseGot.Reconnects != 1 {
		parseT.Fatalf("expected duplicate close callback to schedule one reconnect, got %+v", parseGot)
	}
	parseController.stop(true)
}

func TestRealtimeControllerSendAndHeartbeatErrorPaths(parseT *testing.T) {
	parseSendErr := errors.New("write failed")
	parseNow := time.Date(2026, time.June, 13, 11, 0, 0, 0, time.UTC)
	parseController, parseState, parseCalls := newRealtimeTestController("WebSocket", realtimeResolvedOptions{
		reconnect:           true,
		maxReconnects:       1,
		initialBackoff:      time.Hour,
		maxBackoff:          time.Hour,
		backoffFactor:       2,
		heartbeatInterval:   time.Hour,
		heartbeatTimeout:    time.Hour,
		heartbeatMessage:    "keepalive",
		shouldSendHeartbeat: true,
		maxMessages:         4,
		maxErrors:           2,
		now:                 func() time.Time { return parseNow },
	})
	if parseErr := parseController.send("before-open"); parseErr == nil || !strings.Contains(parseErr.Error(), "not open") {
		parseT.Fatalf("expected send before open to fail, got %v", parseErr)
	}

	parseController.start()
	(*parseCalls)[0].parseCallbacks.handleOpen()
	if parseErr := parseController.send("hello"); parseErr != nil {
		parseT.Fatalf("expected send through open transport to succeed, got %v", parseErr)
	}
	if parseSent := (*parseCalls)[0].parseTransport.sent(); !reflect.DeepEqual(parseSent, []string{"hello"}) {
		parseT.Fatalf("expected sent message, got %v", parseSent)
	}

	(*parseCalls)[0].parseTransport.parseSendErr = parseSendErr
	if parseErr := parseController.send("boom"); parseErr != parseSendErr {
		parseT.Fatalf("expected send error to be returned, got %v", parseErr)
	}
	if parseGot := parseState.Get(); parseGot.Error != parseSendErr || len(parseGot.Errors) != 1 || parseGot.Errors[0].Message != "write failed" {
		parseT.Fatalf("expected send error to be recorded, got %+v", parseGot)
	}

	parseController.handleHeartbeat()
	parseGot := parseState.Get()
	if len(parseGot.Errors) != 2 || parseGot.LastHeartbeatAt != parseNow {
		parseT.Fatalf("expected heartbeat send failure to be bounded and timestamped, got %+v", parseGot)
	}
	parseController.stop(true)
}

func TestRealtimeControllerHeartbeatTimeoutClosesAndReconnects(parseT *testing.T) {
	parseNow := time.Date(2026, time.June, 13, 12, 0, 0, 0, time.UTC)
	parseController, parseState, parseCalls := newRealtimeTestController("EventSource", realtimeResolvedOptions{
		reconnect:         true,
		maxReconnects:     1,
		initialBackoff:    time.Hour,
		maxBackoff:        time.Hour,
		backoffFactor:     2,
		heartbeatInterval: time.Hour,
		heartbeatTimeout:  time.Millisecond,
		maxMessages:       4,
		maxErrors:         4,
		now:               func() time.Time { return parseNow },
	})

	parseController.start()
	(*parseCalls)[0].parseCallbacks.handleOpen()
	parseNow = parseNow.Add(2 * time.Millisecond)
	parseController.handleHeartbeat()
	parseGot := parseState.Get()
	if parseGot.Status != RealtimeReconnecting || parseGot.HeartbeatMisses != 1 || parseGot.Error == nil || !strings.Contains(parseGot.Error.Error(), "heartbeat timed out") {
		parseT.Fatalf("expected heartbeat timeout to schedule reconnect and record miss, got %+v", parseGot)
	}
	waitFetchTestCondition(parseT, time.Second, func() bool {
		return (*parseCalls)[0].parseTransport.closeCount() == 1
	})
	parseController.stop(true)
}

func TestRealtimeHandlesZeroValueNoopMethods(parseT *testing.T) {
	var parseSocket WebSocket
	if parseGot := parseSocket.Get(); !reflect.DeepEqual(parseGot, RealtimeState{}) {
		parseT.Fatalf("expected zero websocket state, got %+v", parseGot)
	}
	if parseErr := parseSocket.Send("ping"); parseErr == nil || !strings.Contains(parseErr.Error(), "not open") {
		parseT.Fatalf("expected zero websocket send error, got %v", parseErr)
	}
	parseSocket.Open()
	parseSocket.Close()

	var parseEvents EventSource
	if parseGot := parseEvents.Get(); !reflect.DeepEqual(parseGot, RealtimeState{}) {
		parseT.Fatalf("expected zero eventsource state, got %+v", parseGot)
	}
	parseEvents.Open()
	parseEvents.Close()
}
