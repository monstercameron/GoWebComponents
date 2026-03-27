package runtime2

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// CapabilitySource describes the raw capability inputs used to build a runtime2 capability report.
type CapabilitySource struct {
	HasWorkerSupport                bool
	HasMessagePortSupport           bool
	HasStructuredCloneSupport       bool
	HasBinaryTransportSupport       bool
	HasSharedBufferSupport          bool
	HasSharedMemoryTransportSupport bool
}

// CapabilityReport reports the currently usable multithreaded runtime capabilities.
type CapabilityReport struct {
	HasWorkerSupport                bool
	HasMessagePortSupport           bool
	HasStructuredCloneSupport       bool
	HasBinaryTransportSupport       bool
	HasSharedBufferSupport          bool
	HasSharedMemoryTransportSupport bool
}

var storeCapabilityReportValue atomic.Value
var isCapabilityReportInitialized atomic.Bool
var runCapabilityDetectOnce sync.Once
var storeCapabilityDetectSource CapabilitySource

// BuildCapabilityReport normalizes raw capability inputs into a usable runtime2 capability report.
func BuildCapabilityReport(parseSource CapabilitySource) CapabilityReport {
	if !parseSource.HasWorkerSupport {
		return CapabilityReport{}
	}
	parseReport := CapabilityReport{
		HasWorkerSupport:          true,
		HasMessagePortSupport:     parseSource.HasMessagePortSupport,
		HasStructuredCloneSupport: parseSource.HasStructuredCloneSupport,
		HasBinaryTransportSupport: parseSource.HasBinaryTransportSupport,
		HasSharedBufferSupport:    parseSource.HasSharedBufferSupport,
	}
	if parseSource.HasSharedBufferSupport && parseSource.HasSharedMemoryTransportSupport {
		parseReport.HasSharedMemoryTransportSupport = true
	}
	return parseReport
}

// ValidateCapabilityReport verifies an explicit capability report is internally consistent.
func ValidateCapabilityReport(parseReport CapabilityReport) error {
	if !parseReport.HasWorkerSupport &&
		(parseReport.HasMessagePortSupport ||
			parseReport.HasStructuredCloneSupport ||
			parseReport.HasBinaryTransportSupport ||
			parseReport.HasSharedBufferSupport ||
			parseReport.HasSharedMemoryTransportSupport) {
		return fmt.Errorf("runtime2: worker support is required before dependent capabilities")
	}
	if parseReport.HasSharedMemoryTransportSupport && !parseReport.HasSharedBufferSupport {
		return fmt.Errorf("runtime2: shared-memory transport requires shared-buffer support")
	}
	return nil
}

// InitCapabilityReport initializes package-level capability state from one detected capability source.
func InitCapabilityReport(parseSource CapabilitySource) (CapabilityReport, error) {
	parseReport := BuildCapabilityReport(parseSource)
	if parseErr := ValidateCapabilityReport(parseReport); parseErr != nil {
		return CapabilityReport{}, parseErr
	}
	storeCapabilityReportValue.Store(parseReport)
	isCapabilityReportInitialized.Store(true)
	return parseReport, nil
}

// SetCapabilityReportOverride stores one explicit package-level capability override.
func SetCapabilityReportOverride(parseReport CapabilityReport) error {
	if parseErr := ValidateCapabilityReport(parseReport); parseErr != nil {
		return parseErr
	}
	storeCapabilityReportValue.Store(parseReport)
	isCapabilityReportInitialized.Store(true)
	return nil
}

// ResetCapabilityReport clears package-level capability overrides and restores default disabled capability reporting.
func ResetCapabilityReport() {
	storeCapabilityReportValue.Store(CapabilityReport{})
	isCapabilityReportInitialized.Store(false)
}

// InitCapabilityReportFromRuntime initializes package-level capability state from live runtime feature detection.
func InitCapabilityReportFromRuntime() (CapabilityReport, error) {
	runCapabilityDetectOnce.Do(func() {
		storeCapabilityDetectSource = DetectCapabilitySource()
	})
	return InitCapabilityReport(storeCapabilityDetectSource)
}

// GetCapabilityReport reports the currently known package-level multithreaded runtime capabilities.
func GetCapabilityReport() CapabilityReport {
	if isCapabilityReportInitialized.Load() {
		getReportValue := storeCapabilityReportValue.Load()
		if getReport, hasReport := getReportValue.(CapabilityReport); hasReport {
			return getReport
		}
	}
	return BuildCapabilityReport(CapabilitySource{})
}
