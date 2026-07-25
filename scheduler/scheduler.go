package scheduler

import (
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/devtools"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

type Scheduler = runtime.Scheduler
type Deadline = runtime.Deadline

// Metrics is a concurrency-safe snapshot of scheduler ergonomics and backpressure.
type Metrics struct {
	IdleScheduled       int
	IdleExecuted        int
	TimeoutScheduled    int
	TimeoutExecuted     int
	InlineFallbacks     int
	LastDelayMs         int
	MaxDelayMs          int
	TotalQueueLatencyNs int64
}

// Instrumented wraps any runtime scheduler and records scheduling latency.
type Instrumented struct {
	next Scheduler
	mu   sync.Mutex
	m    Metrics
}

// NewInstrumented wraps next. A nil next is allowed and executes callbacks inline.
func NewInstrumented(parseNext Scheduler) *Instrumented {
	return &Instrumented{next: parseNext}
}

func (parseS *Instrumented) RequestIdleCallback(parseCallback func(Deadline)) {
	if parseCallback == nil {
		return
	}
	parseStart := time.Now()
	parseS.mu.Lock()
	parseS.m.IdleScheduled++
	parseS.mu.Unlock()
	parseRun := func(parseDeadline Deadline) {
		parseS.recordExecuted(parseStart, true)
		parseCallback(parseDeadline)
	}
	if parseS.next == nil {
		parseS.recordInline()
		parseRun(inlineDeadline{})
		return
	}
	parseS.next.RequestIdleCallback(parseRun)
}

func (parseS *Instrumented) SetTimeout(parseCallback func(), parseDelay int) {
	if parseCallback == nil {
		return
	}
	parseStart := time.Now()
	parseS.mu.Lock()
	parseS.m.TimeoutScheduled++
	parseS.m.LastDelayMs = parseDelay
	if parseDelay > parseS.m.MaxDelayMs {
		parseS.m.MaxDelayMs = parseDelay
	}
	parseS.mu.Unlock()
	parseRun := func() {
		parseS.recordExecuted(parseStart, false)
		parseCallback()
	}
	if parseS.next == nil {
		parseS.recordInline()
		parseRun()
		return
	}
	parseS.next.SetTimeout(parseRun, parseDelay)
}

// Snapshot returns the current scheduler metrics.
func (parseS *Instrumented) Snapshot() Metrics {
	if parseS == nil {
		return Metrics{}
	}
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	return parseS.m
}

// DevtoolsSection returns a companion devtools section for scheduler telemetry.
func (parseS *Instrumented) DevtoolsSection() devtools.ExtensionSection {
	parseM := parseS.Snapshot()
	return devtools.ExtensionSection{
		Name: "Scheduler",
		Summary: map[string]string{
			"idleScheduled":    intString(parseM.IdleScheduled),
			"idleExecuted":     intString(parseM.IdleExecuted),
			"timeoutScheduled": intString(parseM.TimeoutScheduled),
			"timeoutExecuted":  intString(parseM.TimeoutExecuted),
			"inlineFallbacks":  intString(parseM.InlineFallbacks),
			"maxDelayMs":       intString(parseM.MaxDelayMs),
		},
	}
}

func (parseS *Instrumented) recordExecuted(parseStart time.Time, isIdle bool) {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	if isIdle {
		parseS.m.IdleExecuted++
	} else {
		parseS.m.TimeoutExecuted++
	}
	parseS.m.TotalQueueLatencyNs += time.Since(parseStart).Nanoseconds()
}

func (parseS *Instrumented) recordInline() {
	parseS.mu.Lock()
	parseS.m.InlineFallbacks++
	parseS.mu.Unlock()
}

type inlineDeadline struct{}

func (inlineDeadline) TimeRemaining() float64 { return 1000 }
func (inlineDeadline) DidTimeout() bool       { return false }

func intString(parseValue int) string {
	if parseValue == 0 {
		return "0"
	}
	parseDigits := make([]byte, 0, 12)
	parseNegative := parseValue < 0
	if parseNegative {
		parseValue = -parseValue
	}
	for parseValue > 0 {
		parseDigits = append(parseDigits, byte('0'+parseValue%10))
		parseValue /= 10
	}
	if parseNegative {
		parseDigits = append(parseDigits, '-')
	}
	for parseLeft, parseRight := 0, len(parseDigits)-1; parseLeft < parseRight; parseLeft, parseRight = parseLeft+1, parseRight-1 {
		parseDigits[parseLeft], parseDigits[parseRight] = parseDigits[parseRight], parseDigits[parseLeft]
	}
	return string(parseDigits)
}
