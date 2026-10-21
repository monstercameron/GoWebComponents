package telemetry

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/monstercameron/GoWebComponents/v7/interop"
)

// Defaults applied by NewBatchExporter and NewEventQueue for unset fields.
const (
	DefaultMaxBatchSize = 100
	DefaultTimeout      = 10 * time.Second
	DefaultMaxAttempts  = 4
	DefaultBackoffBase  = 200 * time.Millisecond
	DefaultBackoffMax   = 5 * time.Second
	DefaultQueueLimit   = 1000
	// maxRetryAfter caps a server supplied Retry-After so a hostile collector
	// cannot stall the exporter.
	maxRetryAfter = 30 * time.Second
)

// BatchExporterConfig configures a BatchExporter. Zero fields take the Default*
// values above.
type BatchExporterConfig struct {
	// Endpoint is the OTLP/HTTP JSON traces URL.
	Endpoint string
	// Client sends the requests; http.DefaultClient when nil.
	Client *http.Client
	// Options sets the OTLP resource and scope.
	Options ExportOptions
	// MaxBatchSize caps the events in one request.
	MaxBatchSize int
	// Timeout bounds each request attempt.
	Timeout time.Duration
	// MaxAttempts is the total tries per batch, including the first.
	MaxAttempts int
	// BackoffBase is the delay before the first retry; it doubles per retry.
	BackoffBase time.Duration
	// BackoffMax caps the retry delay.
	BackoffMax time.Duration
	// Wait pauses between attempts and returns early with the context error when
	// parseCtx ends. Tests replace it; nil waits on a real timer.
	Wait func(parseCtx context.Context, parseDelay time.Duration) error
}

// BatchExporter posts RUM events to an OTLP/HTTP JSON collector in bounded
// batches. It retries 5xx and 429 responses and network errors with exponential
// backoff, honours Retry-After, and counts events it gave up on instead of
// blocking the page. It is safe for concurrent use.
type BatchExporter struct {
	config  BatchExporterConfig
	sent    atomic.Uint64
	dropped atomic.Uint64
	retries atomic.Uint64
}

// NewBatchExporter builds a BatchExporter, filling unset config fields with the
// package defaults.
func NewBatchExporter(parseConfig BatchExporterConfig) *BatchExporter {
	if parseConfig.Client == nil {
		parseConfig.Client = http.DefaultClient
	}
	if parseConfig.MaxBatchSize <= 0 {
		parseConfig.MaxBatchSize = DefaultMaxBatchSize
	}
	if parseConfig.Timeout <= 0 {
		parseConfig.Timeout = DefaultTimeout
	}
	if parseConfig.MaxAttempts <= 0 {
		parseConfig.MaxAttempts = DefaultMaxAttempts
	}
	if parseConfig.BackoffBase <= 0 {
		parseConfig.BackoffBase = DefaultBackoffBase
	}
	if parseConfig.BackoffMax <= 0 {
		parseConfig.BackoffMax = DefaultBackoffMax
	}
	if parseConfig.Wait == nil {
		parseConfig.Wait = waitForDelay
	}
	return &BatchExporter{config: parseConfig}
}

// waitForDelay sleeps for parseDelay or until parseCtx ends.
func waitForDelay(parseCtx context.Context, parseDelay time.Duration) error {
	parseTimer := time.NewTimer(parseDelay)
	defer parseTimer.Stop()
	select {
	case <-parseTimer.C:
		return nil
	case <-parseCtx.Done():
		return parseCtx.Err()
	}
}

// Sent reports how many events were accepted by the collector.
func (parseExporter *BatchExporter) Sent() uint64 { return parseExporter.sent.Load() }

// Dropped reports how many events were discarded after retries were exhausted,
// after a permanent failure, or because a queue overflowed.
func (parseExporter *BatchExporter) Dropped() uint64 { return parseExporter.dropped.Load() }

// Retries reports how many retry attempts were made.
func (parseExporter *BatchExporter) Retries() uint64 { return parseExporter.retries.Load() }

// Export sends parseEvents in batches of at most MaxBatchSize. A batch that
// fails for good is counted as dropped and the remaining batches are still
// attempted; the returned error joins every batch failure. When parseCtx ends,
// the events not yet sent are counted as dropped and the context error returned.
func (parseExporter *BatchExporter) Export(parseCtx context.Context, parseEvents []RUMEvent) error {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	var parseErrs []error
	for parseStart := 0; parseStart < len(parseEvents); parseStart += parseExporter.config.MaxBatchSize {
		parseEnd := min(parseStart+parseExporter.config.MaxBatchSize, len(parseEvents))
		parseBatch := parseEvents[parseStart:parseEnd]
		if parseErr := parseCtx.Err(); parseErr != nil {
			parseExporter.dropped.Add(uint64(len(parseEvents) - parseStart))
			return errors.Join(append(parseErrs, parseErr)...)
		}
		if parseErr := parseExporter.sendBatch(parseCtx, parseBatch); parseErr != nil {
			parseExporter.dropped.Add(uint64(len(parseBatch)))
			parseErrs = append(parseErrs, parseErr)
			continue
		}
		parseExporter.sent.Add(uint64(len(parseBatch)))
	}
	return errors.Join(parseErrs...)
}

// sendBatch posts one batch, retrying retryable failures with backoff.
func (parseExporter *BatchExporter) sendBatch(parseCtx context.Context, parseBatch []RUMEvent) error {
	parseBody, parseErr := BuildOTLPJSON(parseBatch, parseExporter.config.Options)
	if parseErr != nil {
		return parseErr
	}
	parseDelay := parseExporter.config.BackoffBase
	var parseLast error
	for parseAttempt := 1; parseAttempt <= parseExporter.config.MaxAttempts; parseAttempt++ {
		parseRetryAfter, parseRetryable, parseErr := parseExporter.postOnce(parseCtx, parseBody)
		if parseErr == nil {
			return nil
		}
		parseLast = parseErr
		if !parseRetryable || parseCtx.Err() != nil || parseAttempt == parseExporter.config.MaxAttempts {
			break
		}
		parseWait := parseDelay
		if parseRetryAfter > 0 {
			parseWait = min(parseRetryAfter, maxRetryAfter)
		}
		parseExporter.retries.Add(1)
		if parseWaitErr := parseExporter.config.Wait(parseCtx, parseWait); parseWaitErr != nil {
			return errors.Join(parseLast, parseWaitErr)
		}
		parseDelay = min(parseDelay*2, parseExporter.config.BackoffMax)
	}
	return parseLast
}

// postOnce makes one bounded request. It reports the server's Retry-After, if
// any, and whether the failure is worth retrying.
func (parseExporter *BatchExporter) postOnce(parseCtx context.Context, parseBody []byte) (time.Duration, bool, error) {
	parseAttemptCtx, parseCancel := context.WithTimeout(parseCtx, parseExporter.config.Timeout)
	defer parseCancel()
	parseReq, parseErr := http.NewRequestWithContext(parseAttemptCtx, http.MethodPost, parseExporter.config.Endpoint, bytes.NewReader(parseBody))
	if parseErr != nil {
		return 0, false, parseErr
	}
	parseReq.Header.Set("Content-Type", "application/json")
	parseResp, parseErr := parseExporter.config.Client.Do(parseReq)
	if parseErr != nil {
		// Transport failures and per-attempt timeouts are transient.
		return 0, true, parseErr
	}
	if parseCloseErr := parseResp.Body.Close(); parseCloseErr != nil {
		return 0, true, parseCloseErr
	}
	if parseResp.StatusCode < 300 {
		return 0, false, nil
	}
	parseStatusErr := &HTTPStatusError{StatusCode: parseResp.StatusCode}
	parseRetryable := parseResp.StatusCode == http.StatusTooManyRequests || parseResp.StatusCode >= 500
	return getRetryAfter(parseResp.Header.Get("Retry-After")), parseRetryable, parseStatusErr
}

// getRetryAfter reads a delta-seconds Retry-After value; anything else yields 0.
func getRetryAfter(parseValue string) time.Duration {
	parseSeconds, parseErr := strconv.Atoi(parseValue)
	if parseErr != nil || parseSeconds <= 0 {
		return 0
	}
	return time.Duration(parseSeconds) * time.Second
}

// EventQueue buffers RUM events and sends them through a BatchExporter on Flush.
// When full it drops the oldest event and counts the loss on the exporter. It is
// safe for concurrent use.
type EventQueue struct {
	mu       sync.Mutex
	events   []RUMEvent
	limit    int
	exporter *BatchExporter
}

// NewEventQueue builds a queue holding at most parseLimit events (DefaultQueueLimit
// when not positive) that flushes through parseExporter.
func NewEventQueue(parseExporter *BatchExporter, parseLimit int) *EventQueue {
	if parseLimit <= 0 {
		parseLimit = DefaultQueueLimit
	}
	return &EventQueue{limit: parseLimit, exporter: parseExporter}
}

// Add appends parseEvent, dropping the oldest queued event when the queue is full.
func (parseQueue *EventQueue) Add(parseEvent RUMEvent) {
	parseQueue.mu.Lock()
	defer parseQueue.mu.Unlock()
	if len(parseQueue.events) >= parseQueue.limit {
		parseQueue.events = parseQueue.events[1:]
		parseQueue.exporter.dropped.Add(1)
	}
	parseQueue.events = append(parseQueue.events, parseEvent)
}

// Len reports the number of queued events.
func (parseQueue *EventQueue) Len() int {
	parseQueue.mu.Lock()
	defer parseQueue.mu.Unlock()
	return len(parseQueue.events)
}

// Flush exports and clears the queued events. Events that fail to send are
// counted as dropped by the exporter, not requeued, so a dead collector cannot
// grow memory.
func (parseQueue *EventQueue) Flush(parseCtx context.Context) error {
	parseQueue.mu.Lock()
	parseEvents := parseQueue.events
	parseQueue.events = nil
	parseQueue.mu.Unlock()
	if len(parseEvents) == 0 {
		return nil
	}
	return parseQueue.exporter.Export(parseCtx, parseEvents)
}

// PageHideTarget is the part of interop.EventTarget the flush binding needs.
type PageHideTarget interface {
	Listen(parseName string, parseHandler func(interop.BrowserEvent)) (interop.Subscription, error)
}

// BindFlushOnPageHide flushes parseQueue whenever parseTarget fires one of
// parseEvents (pagehide when none are given). The flush runs on its own goroutine
// so the browser event handler never blocks on the network. The returned function
// removes every listener.
func BindFlushOnPageHide(parseQueue *EventQueue, parseTarget PageHideTarget, parseEvents ...string) (func(), error) {
	if len(parseEvents) == 0 {
		parseEvents = []string{"pagehide"}
	}
	var parseSubs []interop.Subscription
	parseCancel := func() {
		for _, parseSub := range parseSubs {
			parseSub.Cancel()
		}
	}
	for _, parseName := range parseEvents {
		parseSub, parseErr := parseTarget.Listen(parseName, func(interop.BrowserEvent) {
			go func() { _ = parseQueue.Flush(context.Background()) }()
		})
		if parseErr != nil {
			parseCancel()
			return func() {}, parseErr
		}
		parseSubs = append(parseSubs, parseSub)
	}
	return parseCancel, nil
}

// BindBrowserFlushOnPageHide binds parseQueue to the window pagehide event. It
// returns an unavailable error outside a browser.
func BindBrowserFlushOnPageHide(parseQueue *EventQueue) (func(), error) {
	parseWindow, parseErr := interop.GetWindowEvents()
	if parseErr != nil {
		return func() {}, parseErr
	}
	return BindFlushOnPageHide(parseQueue, parseWindow, "pagehide")
}
