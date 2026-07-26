package ui

import (
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

// SSRObservabilityOptions configures per-operation SSR observability metadata.
type SSRObservabilityOptions struct {
	CorrelationID string
	OnEvent       func(SSRObservation)
}

// SSRObservation describes one SSR or hydration observability event.
type SSRObservation struct {
	Name          string
	Domain        string
	Phase         string
	Timestamp     time.Time
	CorrelationID string
	Render        *SSRRenderMetrics
	Bootstrap     *SSRBootstrapMetrics
	Hydration     *SSRHydrationMetrics
}

// SSRRenderMetrics captures one server render duration sample.
type SSRRenderMetrics struct {
	Duration   time.Duration
	DurationNs int64
}

// SSRBootstrapMetrics captures one bootstrap serialization size sample.
type SSRBootstrapMetrics struct {
	Format       string
	PayloadBytes int
	ScriptBytes  int
}

// SSRHydrationMetrics captures one client hydration pass summary.
type SSRHydrationMetrics struct {
	StartedAt            time.Time
	FinishedAt           time.Time
	Duration             time.Duration
	DurationNs           int64
	ExistingDOMNodeCount int
	FallbackCount        int
	MismatchCount        int
	DiscardedNodeCount   int
	Strict               bool
	Failed               bool
	Failure              string
}

type ssrObserver struct {
	id     int
	notify func(SSRObservation)
}

var (
	ssrObserversMu   sync.Mutex
	nextSSRObserver  int
	ssrObserversList []ssrObserver
)

// SSRObserverSubscription is the cleanup handle returned by RegisterSSRObserver.
type SSRObserverSubscription struct {
	cancel func()
}

// Cancel unregisters the SSR observer installed by RegisterSSRObserver.
func (parseS SSRObserverSubscription) Cancel() {
	if parseS.cancel != nil {
		parseS.cancel()
	}
}

// RegisterSSRObserver subscribes to framework-owned SSR and hydration observability events.
// Call Cancel on the returned subscription to unsubscribe.
func RegisterSSRObserver(parseNotify func(SSRObservation)) SSRObserverSubscription {
	if parseNotify == nil {
		return SSRObserverSubscription{}
	}

	ssrObserversMu.Lock()
	nextSSRObserver++
	parseObserverID := nextSSRObserver
	ssrObserversList = append(ssrObserversList, ssrObserver{id: parseObserverID, notify: parseNotify})
	ssrObserversMu.Unlock()

	return SSRObserverSubscription{cancel: func() {
		ssrObserversMu.Lock()
		defer ssrObserversMu.Unlock()
		for parseIndex, parseObserver := range ssrObserversList {
			if parseObserver.id != parseObserverID {
				continue
			}
			// Release the slot the shift vacates: an observer holds a notify
			// closure, which the compaction would otherwise leave reachable
			// past the new length until another subscriber overwrote it.
			copy(ssrObserversList[parseIndex:], ssrObserversList[parseIndex+1:])
			ssrObserversList[len(ssrObserversList)-1] = ssrObserver{}
			ssrObserversList = ssrObserversList[:len(ssrObserversList)-1]
			return
		}
	}}
}

// dispatchSSRObservation is a core package helper.
func dispatchSSRObservation(parseOptions SSRObservabilityOptions, parseObservation SSRObservation) {
	if parseObservation.Timestamp.IsZero() {
		parseObservation.Timestamp = time.Now().UTC()
	}
	if parseObservation.CorrelationID == "" {
		parseObservation.CorrelationID = strings.TrimSpace(parseOptions.CorrelationID)
	}

	ssrObserversMu.Lock()
	parseObservers := append([]ssrObserver(nil), ssrObserversList...)
	ssrObserversMu.Unlock()

	for _, parseObserver := range parseObservers {
		if parseObserver.notify != nil {
			parseObserver.notify(parseObservation)
		}
	}
	if parseOptions.OnEvent != nil {
		parseOptions.OnEvent(parseObservation)
	}
}

// newSSRRenderObservation is a core package helper.
func newSSRRenderObservation(parseOptions SSRObservabilityOptions, parseDuration time.Duration, parseErr error) SSRObservation {
	parsePhase := "finish"
	if parseErr != nil {
		parsePhase = "error"
	}
	return SSRObservation{
		Name:          "ssr.render",
		Domain:        "ssr",
		Phase:         parsePhase,
		CorrelationID: strings.TrimSpace(parseOptions.CorrelationID),
		Render: &SSRRenderMetrics{
			Duration:   parseDuration,
			DurationNs: parseDuration.Nanoseconds(),
		},
	}
}

// newSSRBootstrapObservation is a core package helper.
func newSSRBootstrapObservation(parseOptions SSRObservabilityOptions, format string, parsePayloadBytes int, parseScriptBytes int, parseErr error) SSRObservation {
	parsePhase := "finish"
	if parseErr != nil {
		parsePhase = "error"
	}
	return SSRObservation{
		Name:          "ssr.bootstrap",
		Domain:        "ssr",
		Phase:         parsePhase,
		CorrelationID: strings.TrimSpace(parseOptions.CorrelationID),
		Bootstrap: &SSRBootstrapMetrics{
			Format:       format,
			PayloadBytes: parsePayloadBytes,
			ScriptBytes:  parseScriptBytes,
		},
	}
}

// newSSRHydrationObservation is a core package helper.
func newSSRHydrationObservation(parseMetrics runtime.HydrationMetrics) SSRObservation {
	parsePhase := "finish"
	if parseMetrics.Failed {
		parsePhase = "error"
	}
	return SSRObservation{
		Name:          "runtime.hydration",
		Domain:        "runtime",
		Phase:         parsePhase,
		Timestamp:     parseMetrics.FinishedAt,
		CorrelationID: parseMetrics.CorrelationID,
		Hydration: &SSRHydrationMetrics{
			StartedAt:            parseMetrics.StartedAt,
			FinishedAt:           parseMetrics.FinishedAt,
			Duration:             parseMetrics.Duration,
			DurationNs:           parseMetrics.DurationNs,
			ExistingDOMNodeCount: parseMetrics.ExistingDOMNodeCount,
			FallbackCount:        parseMetrics.FallbackCount,
			MismatchCount:        parseMetrics.MismatchCount,
			DiscardedNodeCount:   parseMetrics.DiscardedNodeCount,
			Strict:               parseMetrics.Strict,
			Failed:               parseMetrics.Failed,
			Failure:              parseMetrics.Failure,
		},
	}
}

var _ = newSSRHydrationObservation

// renderToStringObserved is a core package helper.
func renderToStringObserved(parseRoot Node, parseOptions SSRObservabilityOptions) (string, error) {
	parseStart := time.Now()
	parseMarkup, parseErr := runtime.RenderToString(parseRoot)
	dispatchSSRObservation(parseOptions, newSSRRenderObservation(parseOptions, time.Since(parseStart), parseErr))
	return parseMarkup, parseErr
}
