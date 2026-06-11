package runtime

import (
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	diagnosticsMu   sync.Mutex
	diagnosticIndex = map[string]int{}
	diagnostics     []Diagnostic
	logsMu          sync.Mutex
	logBuffer       []LogEntry
)

const maxLogEntries = 200

// ReportDiagnostic records or increments a runtime diagnostic entry.
func ReportDiagnostic(parseSource string, parseSeverity DiagnosticSeverity, parseMessage string) {
	ReportDiagnosticWithContext(parseSource, parseSeverity, parseMessage, "", nil)
}

// ReportDiagnosticWithContext records or increments a runtime diagnostic entry
// and optionally attaches fiber-path context for devtools and debugging.
func ReportDiagnosticWithContext(parseSource string, parseSeverity DiagnosticSeverity, parseMessage string, parsePath string, parseComponentStack []string) {
	reportDiagnosticWithContextDetails(parseSource, parseSeverity, parseMessage, parsePath, parseComponentStack, "", "", nil)
}

// reportDiagnosticWithContextDetails is a core package helper.
func reportDiagnosticWithContextDetails(parseSource string, parseSeverity DiagnosticSeverity, parseMessage string, parsePath string, parseComponentStack []string, parseTopFrame string, parseConsequence string, parseExtraFields map[string]string) {
	parseTrimmedSource := strings.TrimSpace(parseSource)
	if parseTrimmedSource == "" {
		parseTrimmedSource = "runtime"
	}
	parseTrimmedMessage := strings.TrimSpace(parseMessage)
	if parseTrimmedMessage == "" {
		return
	}
	parseTrimmedPath := strings.TrimSpace(parsePath)
	parseStackKey := strings.Join(parseComponentStack, " > ")
	parseClassification := classifyDiagnostic(parseTrimmedSource, parseSeverity, parseTrimmedMessage)
	parseDetails := diagnosticMetadata(parseTrimmedSource, parseSeverity, parseClassification, parseTrimmedMessage)
	if shouldEscalateDiagnosticStrictly(parseTrimmedSource, parseSeverity, parseClassification, parseDetails) {
		escalateStrictDiagnostic(parseTrimmedSource, parseDetails, parseTrimmedMessage, parseTrimmedPath, parseComponentStack)
	}

	// Dedup-first: the key is derived from the same inputs the context-fields
	// map is built from, so repeat diagnostics (e.g. a missing-key warning
	// firing every render) increment a counter without paying the field-map,
	// clone, and log construction below.
	parseKey := string(parseSeverity) + "|" + parseTrimmedSource + "|" + parseTrimmedMessage + "|" + parseTrimmedPath + "|" + parseStackKey + "|" +
		strings.TrimSpace(parseTopFrame) + "|" + strings.TrimSpace(parseConsequence) + "|" + diagnosticFieldsKey(parseExtraFields)

	diagnosticsMu.Lock()
	defer diagnosticsMu.Unlock()
	if parseIndex, parseOk := diagnosticIndex[parseKey]; parseOk {
		diagnostics[parseIndex].Count++
		return
	}
	parseFields := diagnosticContextFields(parseTrimmedPath, parseComponentStack, parseTopFrame, parseConsequence, parseExtraFields)

	diagnosticIndex[parseKey] = len(diagnostics)
	diagnostics = append(diagnostics, Diagnostic{
		Source:         parseTrimmedSource,
		Severity:       parseSeverity,
		Classification: parseClassification,
		Code:           parseDetails.Code,
		Docs:           parseDetails.Docs,
		Remediation:    parseDetails.Remediation,
		Recoverable:    parseDetails.Recoverable,
		TopFrame:       strings.TrimSpace(parseTopFrame),
		Consequence:    strings.TrimSpace(parseConsequence),
		Message:        parseTrimmedMessage,
		Count:          1,
		Path:           parseTrimmedPath,
		ComponentStack: append([]string(nil), parseComponentStack...),
		Fields:         cloneLogFields(parseFields),
	})

	reportDiagnosticLogDetails(parseTrimmedSource, parseSeverity, parseTrimmedMessage, parseTopFrame, parseConsequence, parseFields)
}

// GetDiagnostics returns a copy of the current diagnostic list.
func GetDiagnostics() []Diagnostic {
	diagnosticsMu.Lock()
	defer diagnosticsMu.Unlock()
	parseClone := make([]Diagnostic, len(diagnostics))
	for parseIndex, parseDiagnostic := range diagnostics {
		parseClone[parseIndex] = parseDiagnostic
		parseClone[parseIndex].ComponentStack = append([]string(nil), parseDiagnostic.ComponentStack...)
		parseClone[parseIndex].Fields = cloneLogFields(parseDiagnostic.Fields)
	}
	return parseClone
}

// ReportLog records one structured framework log entry.
func ReportLog(parseDomain string, parseLevel LogLevel, parseMessage string) {
	ReportLogWithFields(parseDomain, parseLevel, DiagnosticInformational, parseMessage, "", nil)
}

// ReportLogWithFields records one structured framework log entry with optional
// correlation id and fields.
func ReportLogWithFields(parseDomain string, parseLevel LogLevel, parseClassification DiagnosticClassification, parseMessage string, parseCorrelationID string, parseFields map[string]string) {
	reportLogWithFieldsDetails(parseDomain, parseLevel, parseClassification, parseMessage, parseCorrelationID, parseFields, "", "")
}

// reportLogWithFieldsDetails is a core package helper.
func reportLogWithFieldsDetails(parseDomain string, parseLevel LogLevel, parseClassification DiagnosticClassification, parseMessage string, parseCorrelationID string, parseFields map[string]string, parseTopFrame string, parseConsequence string) {
	parseTrimmedDomain := strings.TrimSpace(parseDomain)
	if parseTrimmedDomain == "" {
		parseTrimmedDomain = "runtime"
	}
	parseTrimmedMessage := strings.TrimSpace(parseMessage)
	if parseTrimmedMessage == "" {
		return
	}
	if parseClassification == "" {
		parseClassification = DiagnosticInformational
	}
	if parseLevel == "" {
		parseLevel = LogInfo
	}
	parseDetails := diagnosticMetadata(parseTrimmedDomain, logSeverity(parseLevel), parseClassification, parseTrimmedMessage)

	parseEntry := LogEntry{
		Domain:         parseTrimmedDomain,
		Level:          parseLevel,
		Classification: parseClassification,
		Code:           parseDetails.Code,
		Docs:           parseDetails.Docs,
		Remediation:    parseDetails.Remediation,
		Recoverable:    parseDetails.Recoverable,
		TopFrame:       strings.TrimSpace(parseTopFrame),
		Consequence:    strings.TrimSpace(parseConsequence),
		Message:        parseTrimmedMessage,
		Timestamp:      time.Now().UTC().Format(timeFormatRFC3339Milli),
		CorrelationID:  strings.TrimSpace(parseCorrelationID),
		Fields:         cloneLogFields(parseFields),
	}

	logsMu.Lock()
	defer logsMu.Unlock()
	logBuffer = append(logBuffer, parseEntry)
	if len(logBuffer) > maxLogEntries {
		logBuffer = append([]LogEntry(nil), logBuffer[len(logBuffer)-maxLogEntries:]...)
	}
}

// GetLogs returns a copy of the current in-memory log buffer.
func GetLogs() []LogEntry {
	logsMu.Lock()
	defer logsMu.Unlock()
	parseClone := make([]LogEntry, len(logBuffer))
	for parseIndex, parseEntry := range logBuffer {
		parseClone[parseIndex] = parseEntry
		parseClone[parseIndex].Fields = cloneLogFields(parseEntry.Fields)
	}
	return parseClone
}

// ClearDiagnostics removes all recorded diagnostics.
func ClearDiagnostics() {
	diagnosticsMu.Lock()
	defer diagnosticsMu.Unlock()
	diagnosticIndex = map[string]int{}
	diagnostics = nil
}

// ClearLogs removes all buffered framework logs.
func ClearLogs() {
	logsMu.Lock()
	defer logsMu.Unlock()
	logBuffer = nil
}

// Inspect captures a snapshot of the current runtime tree, profiling state, and diagnostics.
func (parseRt *Runtime) Inspect() InspectionSnapshot {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()

	parseSnapshot := InspectionSnapshot{
		Diagnostics: GetDiagnostics(),
		Logs:        GetLogs(),
	}
	if parseRt == nil {
		return parseSnapshot
	}

	parseEvents := make([]ProfilingEvent, len(parseRt.profiling.events))
	for parseIndex, parseEvent := range parseRt.profiling.events {
		parseEvents[parseIndex] = parseEvent
		parseEvents[parseIndex].Fields = cloneLogFields(parseEvent.Fields)
	}
	var parseRoot *FiberSnapshot
	var parseStats InspectionStats
	if parseRt.currentRoot != nil {
		parseRoot, parseStats = inspectFiberTree(parseRt.currentRoot)
	}
	parseStartedAt := ""
	if !parseRt.profiling.startupStartedAt.IsZero() {
		parseStartedAt = parseRt.profiling.startupStartedAt.UTC().Format(timeFormatRFC3339Milli)
	}
	parseSnapshot.Root = parseRoot
	parseSnapshot.Stats = parseStats
	parseSnapshot.Profiling = ProfilingSnapshot{
		RenderCalls:                      parseRt.profiling.renderCalls,
		ScheduledRootUpdates:             parseRt.profiling.scheduledRootUpdates,
		ScheduledFiberMarks:              parseRt.profiling.scheduledFiberMarks,
		ScheduledGranularMarks:           parseRt.profiling.scheduledGranularMarks,
		WorkLoopPasses:                   parseRt.profiling.workLoopPasses,
		ProcessedUnits:                   parseRt.profiling.processedUnits,
		CommitCount:                      parseRt.profiling.commitCount,
		FineGrainedCommits:               parseRt.profiling.fineGrainedCommits,
		FineGrainedDescendantHostCommits: parseRt.profiling.fineGrainedDescendantHostCommits,
		FineGrainedDescendantTextCommits: parseRt.profiling.fineGrainedDescendantTextCommits,
		EffectExecutions:                 parseRt.profiling.effectExecutions,
		CleanupExecutions:                parseRt.profiling.cleanupExecutions,
		LastRenderDurationNs:             parseRt.profiling.lastRenderDurationNs,
		LastCommitDurationNs:             parseRt.profiling.lastCommitDurationNs,
		LastEffectDurationNs:             parseRt.profiling.lastEffectDurationNs,
		LastCleanupDurationNs:            parseRt.profiling.lastCleanupDurationNs,
		PhaseTotals: ProfilingPhaseTotalsSnapshot{
			RenderDurationNs:  parseRt.profiling.totalRenderDurationNs,
			DiffDurationNs:    parseRt.profiling.totalDiffDurationNs,
			CommitDurationNs:  parseRt.profiling.totalCommitDurationNs,
			EffectDurationNs:  parseRt.profiling.totalEffectDurationNs,
			CleanupDurationNs: parseRt.profiling.totalCleanupDurationNs,
		},
		ComponentRenders: collectComponentRenderTraces(parseRt.profiling.componentRenders, 30),
		RecentEvents:     parseEvents,
		FlamegraphFrames: collectFlamegraphFrames(parseRoot, 256),
		Startup: StartupProfilingSnapshot{
			Mode:                       parseRt.profiling.startupMode,
			StartedAt:                  parseStartedAt,
			BootstrapReadDurationNs:    parseRt.profiling.bootstrapReadDurationNs,
			WASMTransferBytes:          parseRt.profiling.startupWASMTransferBytes,
			WASMDecodedBytes:           parseRt.profiling.startupWASMDecodedBytes,
			BootstrapDecodedBytes:      parseRt.profiling.startupBootstrapDecodedBytes,
			CacheWarmupDurationNs:      parseRt.profiling.startupCacheWarmupDurationNs,
			ServiceWorkerOverheadNs:    parseRt.profiling.startupServiceWorkerOverheadNs,
			InitialRouteDataBytes:      parseRt.profiling.startupInitialRouteDataBytes,
			HydrationDurationNs:        parseRt.profiling.hydrationDurationNs,
			StartupCommitDurationNs:    parseRt.profiling.startupCommitDurationNs,
			FirstInteractionDurationNs: parseRt.profiling.firstInteractionDurationNs,
			FirstInteractionCaptured:   parseRt.profiling.firstInteractionCaptured,
			FirstInteractionEvent:      parseRt.profiling.firstInteractionEvent,
			RouteBudgets:               buildRouteStartupBudgets(parseRt.profiling.routeStartupBudgets, 12),
		},
		HotBranches: collectHotBranches(parseRoot, 5),
	}
	parseSnapshot.Hydration = inspectHydrationDebugSnapshot(parseRt.lastHydrationMetrics, parseSnapshot.Diagnostics)
	return parseSnapshot
}

const timeFormatRFC3339Milli = "2006-01-02T15:04:05.000Z07:00"

// classifyDiagnostic is a core package helper.
func classifyDiagnostic(parseSource string, parseSeverity DiagnosticSeverity, parseMessage string) DiagnosticClassification {
	parseTrimmed := strings.ToLower(strings.TrimSpace(parseMessage))
	switch parseSeverity {
	case DiagnosticInfo:
		return DiagnosticInformational
	case DiagnosticError:
		return DiagnosticCorrectness
	case DiagnosticWarning:
		if strings.Contains(parseTrimmed, "slow ") {
			return DiagnosticPerformance
		}
		if strings.Contains(parseTrimmed, "fell back") ||
			strings.Contains(parseTrimmed, "recovered") ||
			strings.Contains(parseTrimmed, "ignoring ") ||
			strings.Contains(parseTrimmed, "discarded unexpected") ||
			(strings.Contains(parseTrimmed, "redirect loop") && parseSource == "router") {
			return DiagnosticUnsupportedRecover
		}
		return DiagnosticCorrectness
	default:
		return DiagnosticInformational
	}
}

// reportDiagnosticLogDetails is a core package helper.
func reportDiagnosticLogDetails(parseSource string, parseSeverity DiagnosticSeverity, parseMessage string, parseTopFrame string, parseConsequence string, parseFields map[string]string) {
	reportLogWithFieldsDetails(
		parseSource,
		logLevelForSeverity(parseSeverity),
		classifyDiagnostic(parseSource, parseSeverity, parseMessage),
		parseMessage,
		"",
		parseFields,
		parseTopFrame,
		parseConsequence,
	)
}

// diagnosticContextFields is a core package helper.
func diagnosticContextFields(parsePath string, parseComponentStack []string, parseTopFrame string, parseConsequence string, parseExtraFields map[string]string) map[string]string {
	parseFields := cloneLogFields(parseExtraFields)
	if len(parseFields) == 0 {
		parseFields = map[string]string{}
	}
	if strings.TrimSpace(parsePath) != "" {
		parseFields["path"] = strings.TrimSpace(parsePath)
	}
	if len(parseComponentStack) > 0 {
		parseFields["component_stack"] = strings.Join(parseComponentStack, " > ")
	}
	if strings.TrimSpace(parseTopFrame) != "" {
		parseFields["top_frame"] = strings.TrimSpace(parseTopFrame)
	}
	if strings.TrimSpace(parseConsequence) != "" {
		parseFields["runtime"] = strings.TrimSpace(parseConsequence)
	}
	if len(parseFields) == 0 {
		return nil
	}
	return parseFields
}

// diagnosticFieldsKey is a core package helper.
func diagnosticFieldsKey(parseFields map[string]string) string {
	if len(parseFields) == 0 {
		return ""
	}
	parseKeys := make([]string, 0, len(parseFields))
	for parseKey := range parseFields {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	parseParts := make([]string, 0, len(parseKeys))
	for _, parseKey2 := range parseKeys {
		parseParts = append(parseParts, parseKey2+"="+parseFields[parseKey2])
	}
	return strings.Join(parseParts, "|")
}

// logLevelForSeverity is a core package helper.
func logLevelForSeverity(parseSeverity DiagnosticSeverity) LogLevel {
	switch parseSeverity {
	case DiagnosticError:
		return LogError
	case DiagnosticWarning:
		return LogWarn
	default:
		return LogInfo
	}
}

// cloneLogFields is a core package helper.
func cloneLogFields(parseFields map[string]string) map[string]string {
	if len(parseFields) == 0 {
		return nil
	}
	parseClone := make(map[string]string, len(parseFields))
	for parseKey, parseValue := range parseFields {
		parseClone[parseKey] = parseValue
	}
	return parseClone
}
