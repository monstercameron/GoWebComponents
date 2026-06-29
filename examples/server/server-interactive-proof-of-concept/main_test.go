package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// buildServerInteractiveNoFlushRecorder captures handler output without implementing http.Flusher.
type buildServerInteractiveNoFlushRecorder struct {
	buildHeader http.Header
	buildBody   strings.Builder
	buildStatus int
}

// Header returns the response headers for the non-flushing recorder.
func (buildR *buildServerInteractiveNoFlushRecorder) Header() http.Header {
	if buildR.buildHeader == nil {
		buildR.buildHeader = make(http.Header)
	}
	return buildR.buildHeader
}

// Write stores response body bytes for the non-flushing recorder.
func (buildR *buildServerInteractiveNoFlushRecorder) Write(buildBytes []byte) (int, error) {
	if buildR.buildStatus == 0 {
		buildR.buildStatus = http.StatusOK
	}
	return buildR.buildBody.Write(buildBytes)
}

// WriteHeader stores the response status for the non-flushing recorder.
func (buildR *buildServerInteractiveNoFlushRecorder) WriteHeader(buildStatus int) {
	buildR.buildStatus = buildStatus
}

// TestBuildServerInteractiveSnapshotIncludesRenderedState verifies initial hub snapshot rendering.
func TestBuildServerInteractiveSnapshotIncludesRenderedState(parseT *testing.T) {
	parseHub := buildServerInteractiveHub()

	parseSnapshotBytes, parseErr := parseHub.buildServerInteractiveSnapshot()
	if parseErr != nil {
		parseT.Fatalf("buildServerInteractiveSnapshot: %v", parseErr)
	}

	var parseSnapshot serverInteractiveSnapshot
	if parseErr2 := json.Unmarshal(parseSnapshotBytes, &parseSnapshot); parseErr2 != nil {
		parseT.Fatalf("Unmarshal snapshot: %v", parseErr2)
	}
	if parseSnapshot.Version != 1 {
		parseT.Fatalf("expected version 1, got %d", parseSnapshot.Version)
	}
	for _, parseNeedle := range []string{
		"Active Users",
		"Pending Approvals",
		"Incidents Today",
		"Initial dashboard snapshot seeded by server.",
		`data-action="add-user"`,
	} {
		if !strings.Contains(parseSnapshot.HTML, parseNeedle) {
			parseT.Fatalf("expected snapshot HTML to contain %q", parseNeedle)
		}
	}
	if parseSnapshot.UpdatedAt == "" {
		parseT.Fatal("expected updatedAt to be populated")
	}
}

// TestApplyServerInteractiveActionMutatesState verifies server-owned dashboard state transitions.
func TestApplyServerInteractiveActionMutatesState(parseT *testing.T) {
	parseHub := buildServerInteractiveHub()

	if parseEvent := parseHub.applyServerInteractiveAction(serverInteractiveAction{Action: "add-user"}); !strings.Contains(parseEvent, "added one active user") {
		parseT.Fatalf("unexpected add-user event %q", parseEvent)
	}
	if parseHub.storeState.ActiveUsers != 20 {
		parseT.Fatalf("expected active users to increment, got %d", parseHub.storeState.ActiveUsers)
	}

	parseHub.storeState.PendingApprovals = 1
	if parseEvent := parseHub.applyServerInteractiveAction(serverInteractiveAction{Action: "resolve-approval"}); !strings.Contains(parseEvent, "resolved one pending approval") {
		parseT.Fatalf("unexpected resolve-approval event %q", parseEvent)
	}
	if parseHub.storeState.PendingApprovals != 0 {
		parseT.Fatalf("expected pending approvals to decrement, got %d", parseHub.storeState.PendingApprovals)
	}

	parseHub.applyServerInteractiveAction(serverInteractiveAction{Action: "add-incident"})
	if parseHub.storeState.IncidentsToday != 3 {
		parseT.Fatalf("expected incidents to increment, got %d", parseHub.storeState.IncidentsToday)
	}
	parseHub.applyServerInteractiveAction(serverInteractiveAction{Action: "clear-incidents"})
	if parseHub.storeState.IncidentsToday != 0 {
		parseT.Fatalf("expected incidents to clear, got %d", parseHub.storeState.IncidentsToday)
	}

	parseUnknown := parseHub.applyServerInteractiveAction(serverInteractiveAction{Action: "unknown"})
	if !strings.Contains(parseUnknown, "unknown action") {
		parseT.Fatalf("expected unknown action event, got %q", parseUnknown)
	}
	if parseHub.storeState.Version != 6 {
		parseT.Fatalf("expected version to reflect actions, got %d", parseHub.storeState.Version)
	}

	for range 10 {
		parseHub.applyServerInteractiveAction(serverInteractiveAction{Action: "add-user"})
	}
	if len(parseHub.storeState.RecentEvents) != 6 {
		parseT.Fatalf("expected recent events to cap at 6, got %d", len(parseHub.storeState.RecentEvents))
	}
}

// TestBroadcastServerInteractiveSnapshotFanout verifies client registration, delivery, and cleanup.
func TestBroadcastServerInteractiveSnapshotFanout(parseT *testing.T) {
	parseHub := buildServerInteractiveHub()
	parseReadyClient := make(chan []byte, 1)
	parseFullClient := make(chan []byte, 1)
	parseFullClient <- []byte("already-full")

	parseHub.storeServerInteractiveClient(parseReadyClient)
	parseHub.storeServerInteractiveClient(parseFullClient)
	parseHub.broadcastServerInteractiveSnapshot([]byte(`{"ok":true}`))

	select {
	case parsePayload := <-parseReadyClient:
		if string(parsePayload) != `{"ok":true}` {
			parseT.Fatalf("unexpected broadcast payload %q", string(parsePayload))
		}
	default:
		parseT.Fatal("expected ready client to receive snapshot")
	}

	select {
	case parsePayload := <-parseFullClient:
		if string(parsePayload) != "already-full" {
			parseT.Fatalf("expected full client payload to be preserved, got %q", string(parsePayload))
		}
	default:
		parseT.Fatal("expected full client payload to remain queued")
	}

	parseHub.clearServerInteractiveClient(parseReadyClient)
	if len(parseHub.storeClients) != 1 {
		parseT.Fatalf("expected one registered client after cleanup, got %d", len(parseHub.storeClients))
	}
}

// TestHandleServerInteractiveIndexAndActionHandlers verifies method guards and accepted action responses.
func TestHandleServerInteractiveIndexAndActionHandlers(parseT *testing.T) {
	parseHub := buildServerInteractiveHub()

	parseIndexRecorder := httptest.NewRecorder()
	parseHub.handleServerInteractiveIndex(parseIndexRecorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if parseIndexRecorder.Code != http.StatusOK {
		parseT.Fatalf("expected index status 200, got %d", parseIndexRecorder.Code)
	}
	if parseGot := parseIndexRecorder.Header().Get("Content-Type"); parseGot != "text/html; charset=utf-8" {
		parseT.Fatalf("unexpected index content type %q", parseGot)
	}
	if !strings.Contains(parseIndexRecorder.Body.String(), "Server-Interactive Dashboard POC") {
		parseT.Fatalf("expected shell HTML, got %q", parseIndexRecorder.Body.String())
	}

	parseMethodRecorder := httptest.NewRecorder()
	parseHub.handleServerInteractiveAction(parseMethodRecorder, httptest.NewRequest(http.MethodGet, "/action", nil))
	if parseMethodRecorder.Code != http.StatusMethodNotAllowed {
		parseT.Fatalf("expected action method guard, got %d", parseMethodRecorder.Code)
	}

	parseBadPayloadRecorder := httptest.NewRecorder()
	parseBadPayloadRequest := httptest.NewRequest(http.MethodPost, "/action", strings.NewReader("{"))
	parseHub.handleServerInteractiveAction(parseBadPayloadRecorder, parseBadPayloadRequest)
	if parseBadPayloadRecorder.Code != http.StatusBadRequest {
		parseT.Fatalf("expected bad payload status 400, got %d", parseBadPayloadRecorder.Code)
	}

	parseActionRecorder := httptest.NewRecorder()
	parseActionRequest := httptest.NewRequest(http.MethodPost, "/action", strings.NewReader(`{"action":"add-user"}`))
	parseHub.handleServerInteractiveAction(parseActionRecorder, parseActionRequest)
	if parseActionRecorder.Code != http.StatusAccepted {
		parseT.Fatalf("expected accepted action status, got %d", parseActionRecorder.Code)
	}
	if parseGot := parseActionRecorder.Header().Get("Content-Type"); parseGot != "application/json" {
		parseT.Fatalf("unexpected action content type %q", parseGot)
	}
	if strings.TrimSpace(parseActionRecorder.Body.String()) != `{"ok":true}` {
		parseT.Fatalf("unexpected action response %q", parseActionRecorder.Body.String())
	}
}

// TestHandleServerInteractiveEventsBranches verifies SSE setup, method guards, and flusher requirements.
func TestHandleServerInteractiveEventsBranches(parseT *testing.T) {
	parseHub := buildServerInteractiveHub()

	parseMethodRecorder := httptest.NewRecorder()
	parseHub.handleServerInteractiveEvents(parseMethodRecorder, httptest.NewRequest(http.MethodPost, "/events", nil))
	if parseMethodRecorder.Code != http.StatusMethodNotAllowed {
		parseT.Fatalf("expected events method guard, got %d", parseMethodRecorder.Code)
	}

	parseNoFlushRecorder := &buildServerInteractiveNoFlushRecorder{}
	parseHub.handleServerInteractiveEvents(parseNoFlushRecorder, httptest.NewRequest(http.MethodGet, "/events", nil))
	if parseNoFlushRecorder.buildStatus != http.StatusInternalServerError {
		parseT.Fatalf("expected non-flusher branch, got %d", parseNoFlushRecorder.buildStatus)
	}

	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	parseEventRecorder := httptest.NewRecorder()
	parseEventRequest := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(parseCtx)
	parseHub.handleServerInteractiveEvents(parseEventRecorder, parseEventRequest)
	if parseEventRecorder.Code != http.StatusOK {
		parseT.Fatalf("expected events status 200, got %d", parseEventRecorder.Code)
	}
	for _, parseNeedle := range []string{"text/event-stream", "no-store", "keep-alive", "data: ", "\"version\":1"} {
		if !strings.Contains(parseEventRecorder.Body.String()+parseEventRecorder.Header().Get("Content-Type")+parseEventRecorder.Header().Get("Cache-Control")+parseEventRecorder.Header().Get("Connection"), parseNeedle) {
			parseT.Fatalf("expected events response to contain %q", parseNeedle)
		}
	}
	if len(parseHub.storeClients) != 0 {
		parseT.Fatalf("expected events client cleanup, got %d clients", len(parseHub.storeClients))
	}
}
