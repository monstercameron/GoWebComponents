//go:build !js || !wasm

package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// fakeCollector answers each request with the next scripted status and records
// the span count of every request it receives.
type fakeCollector struct {
	mu          sync.Mutex
	statuses    []int
	retryAfter  string
	spanCounts  []int
	requests    int
	contentType string
}

func (parseC *fakeCollector) ServeHTTP(parseW http.ResponseWriter, parseR *http.Request) {
	var parseBody struct {
		ResourceSpans []struct {
			ScopeSpans []struct {
				Spans []json.RawMessage `json:"spans"`
			} `json:"scopeSpans"`
		} `json:"resourceSpans"`
	}
	_ = json.NewDecoder(parseR.Body).Decode(&parseBody)
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	parseC.requests++
	parseC.contentType = parseR.Header.Get("Content-Type")
	parseCount := 0
	for _, parseRS := range parseBody.ResourceSpans {
		for _, parseSS := range parseRS.ScopeSpans {
			parseCount += len(parseSS.Spans)
		}
	}
	parseC.spanCounts = append(parseC.spanCounts, parseCount)
	parseStatus := http.StatusOK
	if len(parseC.statuses) > 0 {
		parseStatus = parseC.statuses[0]
		parseC.statuses = parseC.statuses[1:]
	}
	if parseStatus >= 400 && parseC.retryAfter != "" {
		parseW.Header().Set("Retry-After", parseC.retryAfter)
	}
	parseW.WriteHeader(parseStatus)
}

func buildEvents(parseN int) []RUMEvent {
	parseEvents := make([]RUMEvent, parseN)
	for parseIndex := range parseEvents {
		parseEvents[parseIndex] = RUMEvent{Name: "e" + strconv.Itoa(parseIndex), Type: "profile"}
	}
	return parseEvents
}

// TestBatchExporterTable covers batching, retry classification and drop counting.
func TestBatchExporterTable(parseT *testing.T) {
	for _, parseCase := range []struct {
		name         string
		events       int
		batch        int
		statuses     []int
		attempts     int
		wantRequests int
		wantSpans    []int
		wantSent     uint64
		wantDropped  uint64
		wantRetries  uint64
		wantErr      bool
	}{
		{name: "single batch ok", events: 3, batch: 10, wantRequests: 1, wantSpans: []int{3}, wantSent: 3},
		{name: "splits into batches", events: 5, batch: 2, wantRequests: 3, wantSpans: []int{2, 2, 1}, wantSent: 5},
		{name: "retry 503 then ok", events: 2, batch: 10, statuses: []int{503, 200}, wantRequests: 2, wantSpans: []int{2, 2}, wantSent: 2, wantRetries: 1},
		{name: "retry 429 then ok", events: 1, batch: 10, statuses: []int{429, 429, 200}, wantRequests: 3, wantSpans: []int{1, 1, 1}, wantSent: 1, wantRetries: 2},
		{name: "exhausts attempts", events: 2, batch: 10, statuses: []int{500, 500, 500}, attempts: 3, wantRequests: 3, wantSpans: []int{2, 2, 2}, wantDropped: 2, wantRetries: 2, wantErr: true},
		{name: "4xx is permanent", events: 2, batch: 10, statuses: []int{400}, wantRequests: 1, wantSpans: []int{2}, wantDropped: 2, wantErr: true},
		{name: "failed batch does not stop later ones", events: 4, batch: 2, statuses: []int{400, 200}, wantRequests: 2, wantSpans: []int{2, 2}, wantSent: 2, wantDropped: 2, wantErr: true},
		{name: "empty input sends nothing", events: 0, batch: 10},
	} {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			parseCollector := &fakeCollector{statuses: parseCase.statuses}
			parseSrv := httptest.NewServer(parseCollector)
			defer parseSrv.Close()
			var parseWaits []time.Duration
			parseExporter := NewBatchExporter(BatchExporterConfig{
				Endpoint: parseSrv.URL, MaxBatchSize: parseCase.batch, MaxAttempts: parseCase.attempts,
				Wait: func(_ context.Context, parseDelay time.Duration) error {
					parseWaits = append(parseWaits, parseDelay)
					return nil
				},
			})
			parseErr := parseExporter.Export(context.Background(), buildEvents(parseCase.events))
			if (parseErr != nil) != parseCase.wantErr {
				parseT.Fatalf("err = %v, wantErr %v", parseErr, parseCase.wantErr)
			}
			if parseCollector.requests != parseCase.wantRequests {
				parseT.Fatalf("requests = %d, want %d", parseCollector.requests, parseCase.wantRequests)
			}
			if len(parseCollector.spanCounts) != len(parseCase.wantSpans) {
				parseT.Fatalf("span counts = %v, want %v", parseCollector.spanCounts, parseCase.wantSpans)
			}
			for parseIndex, parseWant := range parseCase.wantSpans {
				if parseCollector.spanCounts[parseIndex] != parseWant {
					parseT.Fatalf("span counts = %v, want %v", parseCollector.spanCounts, parseCase.wantSpans)
				}
			}
			if parseExporter.Sent() != parseCase.wantSent || parseExporter.Dropped() != parseCase.wantDropped || parseExporter.Retries() != parseCase.wantRetries {
				parseT.Fatalf("sent/dropped/retries = %d/%d/%d, want %d/%d/%d", parseExporter.Sent(), parseExporter.Dropped(), parseExporter.Retries(), parseCase.wantSent, parseCase.wantDropped, parseCase.wantRetries)
			}
			if parseCase.wantRequests > 0 && parseCollector.contentType != "application/json" {
				parseT.Fatalf("content type = %q", parseCollector.contentType)
			}
		})
	}
}

// TestBatchExporterBackoffDoublesAndHonoursRetryAfter checks the wait schedule.
func TestBatchExporterBackoffDoublesAndHonoursRetryAfter(parseT *testing.T) {
	parseCollector := &fakeCollector{statuses: []int{500, 500, 500, 500}}
	parseSrv := httptest.NewServer(parseCollector)
	defer parseSrv.Close()
	var parseWaits []time.Duration
	parseExporter := NewBatchExporter(BatchExporterConfig{
		Endpoint: parseSrv.URL, MaxAttempts: 4, BackoffBase: 100 * time.Millisecond, BackoffMax: 300 * time.Millisecond,
		Wait: func(_ context.Context, parseDelay time.Duration) error { parseWaits = append(parseWaits, parseDelay); return nil },
	})
	_ = parseExporter.Export(context.Background(), buildEvents(1))
	parseWant := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond}
	if len(parseWaits) != len(parseWant) {
		parseT.Fatalf("waits = %v, want %v", parseWaits, parseWant)
	}
	for parseIndex := range parseWant {
		if parseWaits[parseIndex] != parseWant[parseIndex] {
			parseT.Fatalf("waits = %v, want %v", parseWaits, parseWant)
		}
	}

	parseCollector = &fakeCollector{statuses: []int{429, 200}, retryAfter: "2"}
	parseSrv2 := httptest.NewServer(parseCollector)
	defer parseSrv2.Close()
	parseWaits = nil
	parseExporter = NewBatchExporter(BatchExporterConfig{
		Endpoint: parseSrv2.URL,
		Wait:     func(_ context.Context, parseDelay time.Duration) error { parseWaits = append(parseWaits, parseDelay); return nil },
	})
	if parseErr := parseExporter.Export(context.Background(), buildEvents(1)); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if len(parseWaits) != 1 || parseWaits[0] != 2*time.Second {
		parseT.Fatalf("Retry-After wait = %v", parseWaits)
	}
}

// TestBatchExporterTimeoutRetriesThenDrops proves a stuck collector is bounded by
// the per-attempt timeout and counted as dropped.
func TestBatchExporterTimeoutRetriesThenDrops(parseT *testing.T) {
	parseRelease := make(chan struct{})
	parseSrv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-parseRelease }))
	defer parseSrv.Close()
	defer close(parseRelease)
	parseExporter := NewBatchExporter(BatchExporterConfig{
		Endpoint: parseSrv.URL, Timeout: 30 * time.Millisecond, MaxAttempts: 2,
		Wait: func(context.Context, time.Duration) error { return nil },
	})
	parseErr := parseExporter.Export(context.Background(), buildEvents(2))
	if !errors.Is(parseErr, context.DeadlineExceeded) {
		parseT.Fatalf("err = %v, want deadline exceeded", parseErr)
	}
	if parseExporter.Dropped() != 2 || parseExporter.Retries() != 1 {
		parseT.Fatalf("dropped/retries = %d/%d", parseExporter.Dropped(), parseExporter.Retries())
	}
}

// TestBatchExporterContextCancelDropsRemaining proves cancellation drops what is unsent.
func TestBatchExporterContextCancelDropsRemaining(parseT *testing.T) {
	parseCollector := &fakeCollector{}
	parseSrv := httptest.NewServer(parseCollector)
	defer parseSrv.Close()
	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	parseExporter := NewBatchExporter(BatchExporterConfig{Endpoint: parseSrv.URL, MaxBatchSize: 2})
	if parseErr := parseExporter.Export(parseCtx, buildEvents(5)); !errors.Is(parseErr, context.Canceled) {
		parseT.Fatalf("err = %v", parseErr)
	}
	if parseCollector.requests != 0 || parseExporter.Dropped() != 5 {
		parseT.Fatalf("requests=%d dropped=%d", parseCollector.requests, parseExporter.Dropped())
	}
}

// TestEventQueueOverflowDropsOldestAndFlushes checks the queue bound and flush.
func TestEventQueueOverflowDropsOldestAndFlushes(parseT *testing.T) {
	parseCollector := &fakeCollector{}
	parseSrv := httptest.NewServer(parseCollector)
	defer parseSrv.Close()
	parseExporter := NewBatchExporter(BatchExporterConfig{Endpoint: parseSrv.URL})
	parseQueue := NewEventQueue(parseExporter, 3)
	for _, parseEvent := range buildEvents(5) {
		parseQueue.Add(parseEvent)
	}
	if parseQueue.Len() != 3 || parseExporter.Dropped() != 2 {
		parseT.Fatalf("len=%d dropped=%d", parseQueue.Len(), parseExporter.Dropped())
	}
	if parseErr := parseQueue.Flush(context.Background()); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if parseQueue.Len() != 0 || parseExporter.Sent() != 3 || parseCollector.spanCounts[0] != 3 {
		parseT.Fatalf("len=%d sent=%d counts=%v", parseQueue.Len(), parseExporter.Sent(), parseCollector.spanCounts)
	}
	if parseErr := parseQueue.Flush(context.Background()); parseErr != nil || parseCollector.requests != 1 {
		parseT.Fatalf("empty flush sent a request: err=%v requests=%d", parseErr, parseCollector.requests)
	}
}

// fakePageTarget records listeners and lets a test fire events.
type fakePageTarget struct {
	handlers map[string]func(interop.BrowserEvent)
	failOn   string
}

func (parseF *fakePageTarget) Listen(parseName string, parseHandler func(interop.BrowserEvent)) (interop.Subscription, error) {
	if parseName == parseF.failOn {
		return interop.Subscription{}, errors.New("listen failed")
	}
	if parseF.handlers == nil {
		parseF.handlers = map[string]func(interop.BrowserEvent){}
	}
	parseF.handlers[parseName] = parseHandler
	return interop.Subscription{}, nil
}

// TestBindFlushOnPageHideFlushesOnEvent proves pagehide triggers a flush and that
// a listener failure is reported.
func TestBindFlushOnPageHideFlushesOnEvent(parseT *testing.T) {
	parseCollector := &fakeCollector{}
	parseSrv := httptest.NewServer(parseCollector)
	defer parseSrv.Close()
	parseQueue := NewEventQueue(NewBatchExporter(BatchExporterConfig{Endpoint: parseSrv.URL}), 0)
	parseQueue.Add(buildEvents(1)[0])
	parseTarget := &fakePageTarget{}
	if _, parseErr := BindFlushOnPageHide(parseQueue, parseTarget); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseTarget.handlers["pagehide"](interop.BrowserEvent{})
	parseDeadline := time.Now().Add(2 * time.Second)
	for parseQueue.exporter.Sent() == 0 && time.Now().Before(parseDeadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if parseQueue.exporter.Sent() != 1 {
		parseT.Fatalf("pagehide did not flush: sent=%d", parseQueue.exporter.Sent())
	}
	if _, parseErr := BindFlushOnPageHide(parseQueue, &fakePageTarget{failOn: "hidden"}, "pagehide", "hidden"); parseErr == nil {
		parseT.Fatal("listen failure not reported")
	}
	if _, parseErr := BindBrowserFlushOnPageHide(parseQueue); parseErr == nil {
		parseT.Skip("running in a browser")
	}
}
