package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestAgentDevWebSocketRemainsIdleAndCancels verifies idle reads neither panic nor reconnect and cancellation reaps the reader.
func TestAgentDevWebSocketRemainsIdleAndCancels(parseTest *testing.T) {
	var parseConnections atomic.Int32
	parseAccepted := make(chan struct{}, 4)
	parseDisconnected := make(chan struct{}, 4)
	parseServer := httptest.NewServer(http.HandlerFunc(func(parseWriter http.ResponseWriter, parseRequest *http.Request) {
		parseUpgrader := websocket.Upgrader{}
		parseConn, parseErr := parseUpgrader.Upgrade(parseWriter, parseRequest, nil)
		if parseErr != nil {
			return
		}
		defer parseConn.Close()
		parseConnections.Add(1)
		parseAccepted <- struct{}{}
		_, _, _ = parseConn.ReadMessage()
		parseDisconnected <- struct{}{}
	}))
	defer parseServer.Close()
	parseContext, parseCancel := context.WithCancel(context.Background())
	defer parseCancel()
	parseDone := make(chan struct{})
	go streamAgentDevWebSocket(parseContext, newAgentEventStream(io.Discard, "dev"), "ws"+strings.TrimPrefix(parseServer.URL, "http"), parseDone)
	select {
	case <-parseAccepted:
	case <-time.After(3 * time.Second):
		parseTest.Fatal("websocket did not connect")
	}
	// More than twice the previous 500ms deadline reproduces its poisoned-read panic.
	time.Sleep(1200 * time.Millisecond)
	if parseConnections.Load() != 1 {
		parseTest.Fatalf("idle websocket reconnected %d times", parseConnections.Load())
	}
	parseCancel()
	select {
	case <-parseDone:
	case <-time.After(time.Second):
		parseTest.Fatal("cancellation did not reap websocket reader")
	}
	select {
	case <-parseDisconnected:
	case <-time.After(time.Second):
		parseTest.Fatal("cancellation did not close underlying connection")
	}
}
