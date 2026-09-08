//go:build !js || !wasm

package fetch

import (
	"errors"
	"testing"
	"time"
)

type realtimeBlockingSend struct {
	realtimeTestTransport
	parseEntered chan struct{}
	parseRelease chan struct{}
}

// send exposes a deterministic heartbeat/stop overlap without blocking close.
func (parseT *realtimeBlockingSend) send(string) error {
	close(parseT.parseEntered)
	<-parseT.parseRelease
	return errors.New("late send error")
}

// TestRealtimeConstructorOverlapStopRejectsLateTransport ensures a constructor
// returning after unmount cannot publish a transport or revive its callbacks.
func TestRealtimeConstructorOverlapStopRejectsLateTransport(parseT *testing.T) {
	parseController, parseState, _ := newRealtimeTestController("WebSocket", resolveWebSocketOptions(nil))
	parseEntered := make(chan realtimeTransportCallbacks, 1)
	parseRelease := make(chan struct{})
	parseFinished := make(chan struct{})
	parseTransport := &realtimeTestTransport{}
	parseController.openTransport = func(_ string, _ realtimeResolvedOptions, parseCallbacks realtimeTransportCallbacks) (realtimeTransport, error) {
		parseEntered <- parseCallbacks
		<-parseRelease
		return parseTransport, nil
	}
	go func() { parseController.start(); close(parseFinished) }()
	parseCallbacks := <-parseEntered
	parseController.stop(true)
	close(parseRelease)
	select {
	case <-parseFinished:
	case <-time.After(time.Second):
		parseT.Fatal("constructor failed to finish")
	}
	parseCallbacks.handleOpen()
	parseCallbacks.handleMessage(RealtimeMessage{Data: "stale"})
	parseCallbacks.handleClose()
	if parseController.getTransport() != nil || parseTransport.closeCount() != 1 {
		parseT.Fatal("late transport leaked or was not closed exactly once")
	}
	if parseGot := parseState.Get(); !parseGot.Closed || parseGot.Open || len(parseGot.Messages) != 0 {
		parseT.Fatalf("stale constructor callback revived closed state: %+v", parseGot)
	}
}

// TestRealtimeHeartbeatOverlapStopCannotInstallTimer forces stop between an
// timer callback's send and completion, then joins it before checking cleanup.
func TestRealtimeHeartbeatOverlapStopCannotInstallTimer(parseT *testing.T) {
	parseController, parseState, _ := newRealtimeTestController("WebSocket", resolveWebSocketOptions([]WebSocketOptions{{HeartbeatInterval: time.Hour}}))
	parseController.options.shouldSendHeartbeat = true
	parseTransport := &realtimeBlockingSend{parseEntered: make(chan struct{}), parseRelease: make(chan struct{})}
	parseController.openTransport = func(_ string, _ realtimeResolvedOptions, parseCallbacks realtimeTransportCallbacks) (realtimeTransport, error) {
		parseCallbacks.handleOpen() // Synchronous callbacks must not deadlock.
		return parseTransport, nil
	}
	parseController.start()
	parseFinished := make(chan struct{})
	go func() { parseController.handleHeartbeat(); close(parseFinished) }()
	select {
	case <-parseTransport.parseEntered:
	case <-time.After(time.Second):
		parseT.Fatal("heartbeat did not send")
	}
	parseController.stop(true)
	close(parseTransport.parseRelease)
	select {
	case <-parseFinished:
	case <-time.After(time.Second):
		parseT.Fatal("stale heartbeat failed to finish")
	}
	parseController.lifecycleMu.Lock()
	defer parseController.lifecycleMu.Unlock()
	parseController.timerMu.Lock()
	defer parseController.timerMu.Unlock()
	if parseController.heartbeatTimer != nil || parseController.reconnectTimer != nil {
		parseT.Fatal("completed stale heartbeat reinstalled a timer after stop")
	}
	if parseGot := parseState.Get(); !parseGot.Closed || parseGot.Error != nil || !parseGot.LastHeartbeatAt.IsZero() {
		parseT.Fatalf("late heartbeat mutated terminal state: %+v", parseGot)
	}
}

// TestRealtimeStaleGenerationCallbacksCannotCloseRestartedConnection verifies
// old open/message/error/close callbacks cannot affect a new live generation.
func TestRealtimeStaleGenerationCallbacksCannotCloseRestartedConnection(parseT *testing.T) {
	parseController, parseState, parseCalls := newRealtimeTestController("WebSocket", resolveWebSocketOptions(nil))
	parseController.start()
	parseOld := parseCalls.get(0).parseCallbacks
	parseOldID := int(parseController.connectionID.Load())
	parseController.stop(true)
	parseController.start()
	defer parseController.stop(true)
	parseCalls.get(1).parseCallbacks.handleOpen()
	parseOld.handleOpen()
	parseOld.handleMessage(RealtimeMessage{Data: "stale"})
	parseOld.handleError(errors.New("stale"))
	parseOld.handleClose()
	parseController.handleReconnectTimer(parseOldID)
	parseController.handleHeartbeatTimer(parseOldID)
	if parseGot := parseState.Get(); !parseGot.Open || parseGot.Error != nil || len(parseGot.Messages) != 0 {
		parseT.Fatalf("stale callback changed new connection: %+v", parseGot)
	}
	if parseController.getTransport() != parseCalls.get(1).parseTransport {
		parseT.Fatal("stale close removed the new transport")
	}
	if parseCalls.count() != 2 || !parseState.Get().LastHeartbeatAt.IsZero() {
		parseT.Fatal("stale timer operated on the new generation")
	}
}
