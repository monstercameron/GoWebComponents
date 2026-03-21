package ui

import (
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
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

// ObserveSSR subscribes to framework-owned SSR and hydration observability events.
func ObserveSSR(notify func(SSRObservation)) func() {
	if notify == nil {
		return func() {}
	}

	ssrObserversMu.Lock()
	nextSSRObserver++
	observerID := nextSSRObserver
	ssrObserversList = append(ssrObserversList, ssrObserver{id: observerID, notify: notify})
	ssrObserversMu.Unlock()

	return func() {
		ssrObserversMu.Lock()
		defer ssrObserversMu.Unlock()
		for index, observer := range ssrObserversList {
			if observer.id != observerID {
				continue
			}
			ssrObserversList = append(ssrObserversList[:index], ssrObserversList[index+1:]...)
			return
		}
	}
}

func dispatchSSRObservation(options SSRObservabilityOptions, observation SSRObservation) {
	if observation.Timestamp.IsZero() {
		observation.Timestamp = time.Now().UTC()
	}
	if observation.CorrelationID == "" {
		observation.CorrelationID = strings.TrimSpace(options.CorrelationID)
	}

	ssrObserversMu.Lock()
	observers := append([]ssrObserver(nil), ssrObserversList...)
	ssrObserversMu.Unlock()

	for _, observer := range observers {
		if observer.notify != nil {
			observer.notify(observation)
		}
	}
	if options.OnEvent != nil {
		options.OnEvent(observation)
	}
}

func newSSRRenderObservation(options SSRObservabilityOptions, duration time.Duration, err error) SSRObservation {
	phase := "finish"
	if err != nil {
		phase = "error"
	}
	return SSRObservation{
		Name:          "ssr.render",
		Domain:        "ssr",
		Phase:         phase,
		CorrelationID: strings.TrimSpace(options.CorrelationID),
		Render: &SSRRenderMetrics{
			Duration:   duration,
			DurationNs: duration.Nanoseconds(),
		},
	}
}

func newSSRBootstrapObservation(options SSRObservabilityOptions, format string, payloadBytes int, scriptBytes int, err error) SSRObservation {
	phase := "finish"
	if err != nil {
		phase = "error"
	}
	return SSRObservation{
		Name:          "ssr.bootstrap",
		Domain:        "ssr",
		Phase:         phase,
		CorrelationID: strings.TrimSpace(options.CorrelationID),
		Bootstrap: &SSRBootstrapMetrics{
			Format:       format,
			PayloadBytes: payloadBytes,
			ScriptBytes:  scriptBytes,
		},
	}
}

func newSSRHydrationObservation(metrics runtime.HydrationMetrics) SSRObservation {
	phase := "finish"
	if metrics.Failed {
		phase = "error"
	}
	return SSRObservation{
		Name:          "runtime.hydration",
		Domain:        "runtime",
		Phase:         phase,
		Timestamp:     metrics.FinishedAt,
		CorrelationID: metrics.CorrelationID,
		Hydration: &SSRHydrationMetrics{
			StartedAt:            metrics.StartedAt,
			FinishedAt:           metrics.FinishedAt,
			Duration:             metrics.Duration,
			DurationNs:           metrics.DurationNs,
			ExistingDOMNodeCount: metrics.ExistingDOMNodeCount,
			FallbackCount:        metrics.FallbackCount,
			MismatchCount:        metrics.MismatchCount,
			DiscardedNodeCount:   metrics.DiscardedNodeCount,
			Strict:               metrics.Strict,
			Failed:               metrics.Failed,
			Failure:              metrics.Failure,
		},
	}
}

func renderToStringObserved(root Node, options SSRObservabilityOptions) (string, error) {
	start := time.Now()
	markup, err := runtime.RenderToString(root)
	dispatchSSRObservation(options, newSSRRenderObservation(options, time.Since(start), err))
	return markup, err
}
