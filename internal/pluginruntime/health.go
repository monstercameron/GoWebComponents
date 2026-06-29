package pluginruntime

import (
	"sort"
	"sync"
	"time"
)

// HealthState describes one plugin runtime health state.
type HealthState string

const (
	HealthStateStarting    HealthState = "starting"
	HealthStateHealthy     HealthState = "healthy"
	HealthStateDegraded    HealthState = "degraded"
	HealthStateQuarantined HealthState = "quarantined"
	HealthStateStopped     HealthState = "stopped"
)

// HealthReason describes the cause of one health transition.
type HealthReason string

const (
	HealthReasonStartupError         HealthReason = "startup-error"
	HealthReasonPanic                HealthReason = "panic"
	HealthReasonCompatibilityFailure HealthReason = "compatibility-failure"
	HealthReasonRepeatedErrors       HealthReason = "repeated-errors"
	HealthReasonBudgetOverrun        HealthReason = "budget-overrun"
	HealthReasonStopped              HealthReason = "stopped"
)

// HealthReport stores one stable health snapshot for one plugin.
type HealthReport struct {
	PluginID   string
	State      HealthState
	Reason     HealthReason
	Message    string
	ErrorCount int
	UpdatedAt  time.Time
}

type healthStateStore struct {
	getMu      sync.RWMutex
	getReports map[string]HealthReport
}

// newHealthStateStore builds one empty health report store.
func newHealthStateStore() *healthStateStore {
	return &healthStateStore{getReports: map[string]HealthReport{}}
}

// setHealthReport stores one health report snapshot.
func (parseStore *healthStateStore) setHealthReport(parseReport HealthReport) {
	if parseStore == nil {
		return
	}
	parseStore.getMu.Lock()
	defer parseStore.getMu.Unlock()
	parseStore.getReports[parseReport.PluginID] = parseReport
}

// getHealthReport returns one cloned health report snapshot.
func (parseStore *healthStateStore) getHealthReport(parsePluginID string) (HealthReport, bool) {
	if parseStore == nil {
		return HealthReport{}, false
	}
	parseStore.getMu.RLock()
	defer parseStore.getMu.RUnlock()
	getReport, hasReport := parseStore.getReports[parsePluginID]
	return getReport, hasReport
}

// listHealthReports returns sorted health reports for every tracked plugin.
func (parseStore *healthStateStore) listHealthReports() []HealthReport {
	if parseStore == nil {
		return nil
	}
	parseStore.getMu.RLock()
	defer parseStore.getMu.RUnlock()
	buildReports := make([]HealthReport, 0, len(parseStore.getReports))
	for _, buildReport := range parseStore.getReports {
		buildReports = append(buildReports, buildReport)
	}
	sort.Slice(buildReports, func(parseI int, parseJ int) bool {
		return buildReports[parseI].PluginID < buildReports[parseJ].PluginID
	})
	return buildReports
}
