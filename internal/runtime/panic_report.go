package runtime

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

const frameworkModulePath = "github.com/monstercameron/GoWebComponents"
const frameworkWorkspaceName = "GoWebComponents"

const (
	visibleAppFrameLimit       = 3
	visibleFrameworkFrameLimit = 4
	visiblePlatformFrameLimit  = 1
)

type panicFrame struct {
	Function string
	File     string
	Line     int
}

type panicReportContext struct {
	Source          string
	Phase           PanicPhase
	Subject         string
	Path            string
	ComponentStack  []string
	Summary         string
	Code            string
	Docs            string
	Remediation     string
	Consequence     string
	TopFrame        string
	AppFrames       []string
	FrameworkFrames []string
	PlatformFrames  []string
	Artifact        WASMArtifactMetadata
}

// PanicReport describes one wrapped framework-owned fatal panic report.
type PanicReport struct {
	Source          string
	Phase           PanicPhase
	Subject         string
	Where           string
	Path            string
	ComponentStack  []string
	Summary         string
	Code            string
	Docs            string
	Remediation     string
	Consequence     string
	TopFrame        string
	AppFrames       []string
	FrameworkFrames []string
	PlatformFrames  []string
	Artifact        WASMArtifactMetadata
	Formatted       string
}

type reportedPanic struct {
	Original interface{}
}

type PanicLoggingOptions struct {
	HideRawPanicOutput bool
	OnReport           func(PanicReport)
}

type ActionablePanicOptions struct {
	Source         string
	Subject        string
	Message        string
	Path           string
	ComponentStack []string
	Consequence    string
}

var hideRawPanicOutput atomic.Bool

var (
	panicLoggingHookMu sync.RWMutex
	panicLoggingHook   func(PanicReport)
)

// init is a core package helper.
func init() {
	hideRawPanicOutput.Store(true)
}

// ConfigureUnhandledPanicLogging sets the panic logging options including raw output visibility and the report hook.
func ConfigureUnhandledPanicLogging(parseOptions PanicLoggingOptions) {
	hideRawPanicOutput.Store(parseOptions.HideRawPanicOutput)
	panicLoggingHookMu.Lock()
	panicLoggingHook = parseOptions.OnReport
	panicLoggingHookMu.Unlock()
}

// CurrentUnhandledPanicLoggingOptions returns a snapshot of the current panic logging configuration.
func CurrentUnhandledPanicLoggingOptions() PanicLoggingOptions {
	panicLoggingHookMu.RLock()
	parseHook := panicLoggingHook
	panicLoggingHookMu.RUnlock()
	return PanicLoggingOptions{HideRawPanicOutput: hideRawPanicOutput.Load(), OnReport: parseHook}
}

// shouldHideRawPanicOutput is a core package helper.
func shouldHideRawPanicOutput() bool {
	return hideRawPanicOutput.Load()
}

// wrappedPanicString is a core package helper.
func wrappedPanicString(parseRecovered interface{}) (string, bool) {
	parseMessage, parseOk := parseRecovered.(string)
	if !parseOk {
		return "", false
	}
	parseTrimmed := strings.TrimSpace(parseMessage)
	if parseTrimmed == "" {
		return "", false
	}
	if strings.Contains(parseTrimmed, "\n[GWC-") || strings.HasPrefix(parseTrimmed, "[GWC-") {
		return parseMessage, true
	}
	return "", false
}

// mergePanicReportFields is a core package helper.
func mergePanicReportFields(parsePrimary map[string]string, parseExtra map[string]string) map[string]string {
	if len(parsePrimary) == 0 && len(parseExtra) == 0 {
		return nil
	}
	parseMerged := map[string]string{}
	for parseKey, parseValue := range parsePrimary {
		parseMerged[parseKey] = parseValue
	}
	for parseKey2, parseValue2 := range parseExtra {
		parseMerged[parseKey2] = parseValue2
	}
	return parseMerged
}

// unwrapReportedPanic is a core package helper.
func unwrapReportedPanic(parseRecovered interface{}) (interface{}, bool) {
	parseCurrent := parseRecovered
	isParseUnwrapped := false
	for {
		switch parseTyped := parseCurrent.(type) {
		case reportedPanic:
			isParseUnwrapped = true
			parseCurrent = parseTyped.Original
		case *reportedPanic:
			isParseUnwrapped = true
			if parseTyped == nil {
				return nil, true
			}
			parseCurrent = parseTyped.Original
		default:
			if !isParseUnwrapped {
				return nil, false
			}
			return parseCurrent, true
		}
	}
}

// currentPanicReportHook is a core package helper.
func currentPanicReportHook() func(PanicReport) {
	panicLoggingHookMu.RLock()
	defer panicLoggingHookMu.RUnlock()
	return panicLoggingHook
}

// clonePanicReport is a core package helper.
func clonePanicReport(parseReport PanicReport) PanicReport {
	parseCloned := parseReport
	parseCloned.ComponentStack = append([]string(nil), parseReport.ComponentStack...)
	parseCloned.AppFrames = append([]string(nil), parseReport.AppFrames...)
	parseCloned.FrameworkFrames = append([]string(nil), parseReport.FrameworkFrames...)
	parseCloned.PlatformFrames = append([]string(nil), parseReport.PlatformFrames...)
	return parseCloned
}

// emitWrappedPanicReport is a core package helper.
func emitWrappedPanicReport(parseReport PanicReport) {
	parseReport.Formatted = strings.TrimSpace(parseReport.Formatted)
	if parseReport.Formatted == "" {
		return
	}
	if parseHook := currentPanicReportHook(); parseHook != nil {
		func() {
			defer func() { _ = recover() }()
			parseHook(clonePanicReport(parseReport))
		}()
	}
	if emitBrowserPanicReport(parseReport) {
		return
	}
	fmt.Println(parseReport.Formatted)
}

// parsePanicFrames is a core package helper.
func parsePanicFrames(parseStack []byte) []panicFrame {
	parseLines := strings.Split(strings.ReplaceAll(string(parseStack), "\r\n", "\n"), "\n")
	parseFrames := make([]panicFrame, 0, len(parseLines)/2)
	for parseIndex := 1; parseIndex+1 < len(parseLines); parseIndex += 2 {
		parseFunction := strings.TrimSpace(parseLines[parseIndex])
		parseLocation := strings.TrimSpace(parseLines[parseIndex+1])
		if parseFunction == "" || parseLocation == "" {
			continue
		}
		if strings.Contains(parseFunction, "runtime/debug.Stack") ||
			strings.Contains(parseFunction, "github.com/monstercameron/GoWebComponents/internal/runtime.parsePanicFrames") ||
			strings.Contains(parseFunction, "github.com/monstercameron/GoWebComponents/internal/runtime.buildPanicReportContext") ||
			strings.Contains(parseFunction, "github.com/monstercameron/GoWebComponents/internal/runtime.ReportUnhandledPanicContext") ||
			strings.Contains(parseFunction, "github.com/monstercameron/GoWebComponents/internal/runtime.reportUnhandledPanic") {
			continue
		}

		parseLocation = strings.TrimSpace(strings.SplitN(parseLocation, " +", 2)[0])
		parseLine := 0
		if parseLastColon := strings.LastIndex(parseLocation, ":"); parseLastColon > 0 {
			if parseParsed, parseErr := strconv.Atoi(parseLocation[parseLastColon+1:]); parseErr == nil {
				parseLine = parseParsed
				parseLocation = parseLocation[:parseLastColon]
			}
		}

		parseFrames = append(parseFrames, translateWASMStackFrame(panicFrame{Function: parseFunction, File: parseLocation, Line: parseLine}))
	}
	return parseFrames
}

// sanitizePanicFunction is a core package helper.
func sanitizePanicFunction(parseFunction string) string {
	parseTrimmed := strings.TrimSpace(parseFunction)
	if parseTrimmed == "" {
		return ""
	}
	if strings.HasSuffix(parseTrimmed, ")") {
		if parseIndex := strings.LastIndex(parseTrimmed, "("); parseIndex > 0 {
			parseTrimmed = parseTrimmed[:parseIndex]
		}
	}
	return strings.TrimSpace(parseTrimmed)
}

// classifyPanicFrame is a core package helper.
func classifyPanicFrame(parseFrame panicFrame) string {
	parseFunction := strings.ReplaceAll(sanitizePanicFunction(parseFrame.Function), "\\", "/")
	parseFile := strings.ReplaceAll(parseFrame.File, "\\", "/")
	if strings.HasSuffix(parseFile, "_test.go") || strings.Contains(parseFunction, ".Test") {
		return "app"
	}

	if strings.Contains(parseFunction, frameworkModulePath+"/") {
		if strings.Contains(parseFunction, frameworkModulePath+"/examples/") || strings.Contains(parseFunction, frameworkModulePath+"/test/") {
			return "app"
		}
		return "framework"
	}

	if strings.Contains(parseFile, "/"+frameworkWorkspaceName+"/") {
		if strings.Contains(parseFile, "/"+frameworkWorkspaceName+"/examples/") || strings.Contains(parseFile, "/"+frameworkWorkspaceName+"/test/") {
			return "app"
		}
		return "framework"
	}

	if strings.HasPrefix(parseFunction, "runtime.") ||
		strings.HasPrefix(parseFunction, "syscall/js.") ||
		strings.HasPrefix(parseFunction, "testing.") ||
		strings.Contains(parseFile, "/src/runtime/") ||
		strings.Contains(parseFile, "/src/testing/") ||
		strings.Contains(parseFile, "/src/syscall/js/") {
		return "platform"
	}

	return "app"
}

// lowSignalPanicFrame is a core package helper.
func lowSignalPanicFrame(parseFrame panicFrame, parseBucket string) bool {
	parseFunction := strings.ReplaceAll(sanitizePanicFunction(parseFrame.Function), "\\", "/")
	if parseFunction == "" {
		return true
	}
	switch parseBucket {
	case "framework":
		for _, parseToken := range []string{
			"internal/runtime.buildUnhandledPanicReport",
			"internal/runtime.markUnhandledPanicContext",
			"internal/runtime.markUnhandledPanic",
			"internal/runtime.finalizeUnhandledPanicContext",
			"internal/runtime.panicFinalUnhandledPanicContext",
			"internal/runtime.panicFinalUnhandledPanic",
			"internal/runtime.reportUnhandledPanic",
			"internal/runtime.ReportUnhandledPanicContext",
			"internal/runtime.(*Runtime).continueWorkLoop",
			"internal/platform/jsdom.(*WASMScheduler).SetTimeout.func1",
		} {
			if strings.Contains(parseFunction, parseToken) {
				return true
			}
		}
	case "platform":
		if parseFunction == "panic" || strings.HasPrefix(parseFunction, "runtime.panic") {
			return true
		}
	}
	return false
}

// shortenPanicFilePath is a core package helper.
func shortenPanicFilePath(parsePath string) string {
	parseNormalized := strings.ReplaceAll(strings.TrimSpace(parsePath), "\\", "/")
	if parseNormalized == "" {
		return ""
	}
	parseMarker := "/" + frameworkWorkspaceName + "/"
	if parseIndex := strings.Index(parseNormalized, parseMarker); parseIndex >= 0 {
		return parseNormalized[parseIndex+len(parseMarker):]
	}
	parseParts := strings.Split(parseNormalized, "/")
	if len(parseParts) <= 3 {
		return parseNormalized
	}
	return strings.Join(parseParts[len(parseParts)-3:], "/")
}

// formatPanicFrame is a core package helper.
func formatPanicFrame(parseFrame panicFrame) string {
	parseFunction := sanitizePanicFunction(parseFrame.Function)
	parseLocation := shortenPanicFilePath(parseFrame.File)
	if parseLocation == "" {
		return parseFunction
	}
	if parseFrame.Line > 0 {
		return fmt.Sprintf("%s at %s:%d", parseFunction, parseLocation, parseFrame.Line)
	}
	return fmt.Sprintf("%s at %s", parseFunction, parseLocation)
}

// appendPanicFrame is a core package helper.
func appendPanicFrame(parseSection []string, parseFrame panicFrame) []string {
	parseFormatted := formatPanicFrame(parseFrame)
	if parseFormatted == "" {
		return parseSection
	}
	return append(parseSection, parseFormatted)
}

// limitPanicFrames is a core package helper.
func limitPanicFrames(parseValues []string, parseLimit int, parseLabel string) []string {
	if parseLimit <= 0 || len(parseValues) <= parseLimit {
		return parseValues
	}
	parseTrimmed := append([]string(nil), parseValues[:parseLimit]...)
	parseTrimmed = append(parseTrimmed, fmt.Sprintf("... %d more %s frames omitted", len(parseValues)-parseLimit, parseLabel))
	return parseTrimmed
}

// panicConsequence is a core package helper.
func panicConsequence(parsePhase PanicPhase) string {
	switch parsePhase {
	case PanicPhaseRender:
		return "no error boundary recovered this render panic; render work was aborted and the app state should be treated as failed."
	case PanicPhaseEvent:
		return "no error boundary recovered this event panic; the callback rethrew and the app state may no longer be trustworthy."
	case PanicPhaseEffect:
		return "no error boundary recovered this effect panic; commit work stopped and follow-up renders should not be trusted."
	case PanicPhaseCleanup:
		return "no error boundary recovered this cleanup panic; teardown stopped mid-flight and the app state may be inconsistent."
	case PanicPhaseLoader:
		return "no route boundary recovered this loader panic; route data resolution stopped and pending navigation state is not trustworthy."
	case PanicPhaseHydration:
		return "hydration could not continue safely; resume work stopped and the runtime rethrew instead of mutating a mismatched tree."
	case PanicPhaseStartup:
		return "startup failed before the app could mount safely; no further render work continued after this panic."
	case PanicPhaseDeferred:
		return "deferred runtime work panicked outside a recovering boundary; the scheduled task aborted and the app should be treated as failed."
	case PanicPhaseSSR:
		return "server rendering failed before HTML could be returned; treat the request as failed and surface the error upstream."
	default:
		return "no recovering boundary handled this panic; the runtime stopped and the app should be treated as failed."
	}
}

// panicPhaseMayRecoverWithBoundary is a core package helper.
func panicPhaseMayRecoverWithBoundary(parsePhase PanicPhase) bool {
	switch parsePhase {
	case PanicPhaseRender, PanicPhaseEvent, PanicPhaseEffect, PanicPhaseCleanup:
		return true
	default:
		return false
	}
}

// buildPanicReportContext is a core package helper.
func buildPanicReportContext(parseSource string, parsePhase PanicPhase, parseSubject string, parsePath string, parseComponentStack []string, parseRecovered interface{}) panicReportContext {
	parseFrames := parsePanicFrames(debug.Stack())
	parseContext := panicReportContext{
		Source:         strings.TrimSpace(parseSource),
		Phase:          parsePhase,
		Subject:        strings.TrimSpace(parseSubject),
		Path:           strings.TrimSpace(parsePath),
		ComponentStack: append([]string(nil), parseComponentStack...),
		Summary:        panicSummary(parseRecovered),
		Code:           panicDiagnosticCode(parsePhase),
		Docs:           panicDiagnosticDocs(parsePhase),
		Remediation:    panicDiagnosticRemediation(parsePhase),
		Consequence:    panicConsequence(parsePhase),
		Artifact:       currentWASMArtifactMetadata(),
	}
	if parseContext.Source == "" {
		parseContext.Source = "runtime"
	}
	if parseContext.Subject == "" {
		parseContext.Subject = "application"
	}

	for _, parseFrame := range parseFrames {
		parseBucket := classifyPanicFrame(parseFrame)
		if lowSignalPanicFrame(parseFrame, parseBucket) {
			continue
		}
		switch parseBucket {
		case "app":
			parseContext.AppFrames = appendPanicFrame(parseContext.AppFrames, parseFrame)
		case "framework":
			parseContext.FrameworkFrames = appendPanicFrame(parseContext.FrameworkFrames, parseFrame)
		default:
			parseContext.PlatformFrames = appendPanicFrame(parseContext.PlatformFrames, parseFrame)
		}
	}

	if len(parseContext.AppFrames) > 0 {
		parseContext.TopFrame = parseContext.AppFrames[0]
	}
	if parseContext.Path == "" && len(parseContext.ComponentStack) > 0 {
		parseContext.Path = strings.Join(parseContext.ComponentStack, " > ")
	}
	if parseContext.Path == "" {
		parseContext.Path = parseContext.Subject
	}
	parseContext.AppFrames = limitPanicFrames(parseContext.AppFrames, visibleAppFrameLimit, "app")
	parseContext.FrameworkFrames = limitPanicFrames(parseContext.FrameworkFrames, visibleFrameworkFrameLimit, "framework")
	parseContext.PlatformFrames = limitPanicFrames(parseContext.PlatformFrames, visiblePlatformFrameLimit, "platform")
	return parseContext
}

// formatPanicStackSection is a core package helper.
func formatPanicStackSection(parseLines []string, parseValues []string) []string {
	for _, parseValue := range parseValues {
		parseLines = append(parseLines, "  "+parseValue)
	}
	return parseLines
}

// buildPanicReport is a core package helper.
func buildPanicReport(parseContext panicReportContext) PanicReport {
	parseWhere := parseContext.TopFrame
	if parseWhere == "" {
		parseWhere = parseContext.Path
	}
	parseLines := []string{
		parseContext.Summary,
		fmt.Sprintf("[%s] uncaught %s panic in %s", parseContext.Code, parseContext.Phase, parseContext.Subject),
		"where: " + parseWhere,
		"path: " + parseContext.Path,
		"error: " + parseContext.Summary,
		"runtime: " + parseContext.Consequence,
		"next: " + parseContext.Remediation,
		"docs: " + parseContext.Docs,
	}
	if parseArtifactFields := panicArtifactFields(parseContext.Artifact); len(parseArtifactFields) > 0 {
		parseParts := make([]string, 0, len(parseArtifactFields))
		for _, parseKey := range []string{"artifact_build_id", "artifact_path", "artifact_sha256", "artifact_manifest", "artifact_symbols", "artifact_version"} {
			if parseValue := strings.TrimSpace(parseArtifactFields[parseKey]); parseValue != "" {
				parseParts = append(parseParts, parseKey+"="+parseValue)
			}
		}
		if len(parseParts) > 0 {
			parseLines = append(parseLines, "artifact: "+strings.Join(parseParts, " | "))
		}
	}
	if len(parseContext.AppFrames) > 0 || len(parseContext.FrameworkFrames) > 0 || len(parseContext.PlatformFrames) > 0 {
		parseLines = append(parseLines, "stack:")
		if len(parseContext.AppFrames) > 0 {
			parseLines = append(parseLines, "app:")
			parseLines = formatPanicStackSection(parseLines, parseContext.AppFrames)
		}
		if len(parseContext.FrameworkFrames) > 0 {
			parseLines = append(parseLines, "framework: GWC")
			parseLines = formatPanicStackSection(parseLines, parseContext.FrameworkFrames)
		}
		if len(parseContext.PlatformFrames) > 0 {
			parseLines = append(parseLines, "platform: GOLANG")
			parseLines = formatPanicStackSection(parseLines, parseContext.PlatformFrames)
		}
	}
	return PanicReport{
		Source:          parseContext.Source,
		Phase:           parseContext.Phase,
		Subject:         parseContext.Subject,
		Where:           parseWhere,
		Path:            parseContext.Path,
		ComponentStack:  append([]string(nil), parseContext.ComponentStack...),
		Summary:         parseContext.Summary,
		Code:            parseContext.Code,
		Docs:            parseContext.Docs,
		Remediation:     parseContext.Remediation,
		Consequence:     parseContext.Consequence,
		TopFrame:        parseContext.TopFrame,
		AppFrames:       append([]string(nil), parseContext.AppFrames...),
		FrameworkFrames: append([]string(nil), parseContext.FrameworkFrames...),
		PlatformFrames:  append([]string(nil), parseContext.PlatformFrames...),
		Artifact:        parseContext.Artifact,
		Formatted:       strings.Join(parseLines, "\n"),
	}
}

// formatPanicReport is a core package helper.
func formatPanicReport(parseContext panicReportContext) string {
	return buildPanicReport(parseContext).Formatted
}

// defaultActionablePanicConsequence is a core package helper.
func defaultActionablePanicConsequence(parseSubject string) string {
	if strings.TrimSpace(parseSubject) == "" {
		return "framework API validation failed; the current call aborted before runtime state could continue changing."
	}
	return fmt.Sprintf("framework API validation failed in %s; the current call aborted before runtime state could continue changing.", strings.TrimSpace(parseSubject))
}

// buildActionablePanicContext is a core package helper.
func buildActionablePanicContext(parseOptions ActionablePanicOptions) panicReportContext {
	parseSource := strings.TrimSpace(parseOptions.Source)
	if parseSource == "" {
		parseSource = "runtime"
	}
	parseSubject := strings.TrimSpace(parseOptions.Subject)
	if parseSubject == "" {
		parseSubject = parseSource
	}
	parseSummary := strings.TrimSpace(parseOptions.Message)
	if parseSummary == "" {
		parseSummary = "framework misuse without message"
	}
	parseDetails := diagnosticMetadata(parseSource, DiagnosticError, DiagnosticCorrectness, parseSummary)
	if strings.TrimSpace(parseDetails.Code) == "" {
		parseDetails.Code = "GWC-FRAMEWORK-USAGE"
		parseDetails.Docs = actionableErrorsDoc
		parseDetails.Remediation = "Inspect the where/path fields first, then correct the invalid framework API usage before retrying."
	}
	parseContext := panicReportContext{
		Source:         parseSource,
		Subject:        parseSubject,
		Path:           strings.TrimSpace(parseOptions.Path),
		ComponentStack: append([]string(nil), parseOptions.ComponentStack...),
		Summary:        parseSummary,
		Code:           parseDetails.Code,
		Docs:           parseDetails.Docs,
		Remediation:    parseDetails.Remediation,
		Consequence:    strings.TrimSpace(parseOptions.Consequence),
	}
	if parseContext.Consequence == "" {
		parseContext.Consequence = defaultActionablePanicConsequence(parseSubject)
	}
	if parseContext.Path == "" && len(parseContext.ComponentStack) > 0 {
		parseContext.Path = strings.Join(parseContext.ComponentStack, " > ")
	}
	if parseContext.Path == "" {
		parseContext.Path = parseSubject
	}
	for _, parseFrame := range parsePanicFrames(debug.Stack()) {
		parseBucket := classifyPanicFrame(parseFrame)
		if lowSignalPanicFrame(parseFrame, parseBucket) {
			continue
		}
		switch parseBucket {
		case "app":
			parseContext.AppFrames = appendPanicFrame(parseContext.AppFrames, parseFrame)
		case "framework":
			parseContext.FrameworkFrames = appendPanicFrame(parseContext.FrameworkFrames, parseFrame)
		default:
			parseContext.PlatformFrames = appendPanicFrame(parseContext.PlatformFrames, parseFrame)
		}
	}
	if len(parseContext.AppFrames) > 0 {
		parseContext.TopFrame = parseContext.AppFrames[0]
	}
	parseContext.AppFrames = limitPanicFrames(parseContext.AppFrames, visibleAppFrameLimit, "app")
	parseContext.FrameworkFrames = limitPanicFrames(parseContext.FrameworkFrames, visibleFrameworkFrameLimit, "framework")
	parseContext.PlatformFrames = limitPanicFrames(parseContext.PlatformFrames, visiblePlatformFrameLimit, "platform")
	return parseContext
}

// formatActionablePanicReport is a core package helper.
func formatActionablePanicReport(parseContext panicReportContext) string {
	parseWhere := parseContext.TopFrame
	if parseWhere == "" {
		parseWhere = parseContext.Path
	}
	parseLines := []string{
		parseContext.Summary,
		fmt.Sprintf("[%s] framework misuse in %s", parseContext.Code, parseContext.Subject),
		"where: " + parseWhere,
		"path: " + parseContext.Path,
		"error: " + parseContext.Summary,
		"runtime: " + parseContext.Consequence,
		"next: " + parseContext.Remediation,
		"docs: " + parseContext.Docs,
	}
	if len(parseContext.AppFrames) > 0 || len(parseContext.FrameworkFrames) > 0 || len(parseContext.PlatformFrames) > 0 {
		parseLines = append(parseLines, "stack:")
		if len(parseContext.AppFrames) > 0 {
			parseLines = append(parseLines, "app:")
			parseLines = formatPanicStackSection(parseLines, parseContext.AppFrames)
		}
		if len(parseContext.FrameworkFrames) > 0 {
			parseLines = append(parseLines, "framework: GWC")
			parseLines = formatPanicStackSection(parseLines, parseContext.FrameworkFrames)
		}
		if len(parseContext.PlatformFrames) > 0 {
			parseLines = append(parseLines, "platform: GOLANG")
			parseLines = formatPanicStackSection(parseLines, parseContext.PlatformFrames)
		}
	}
	return strings.Join(parseLines, "\n")
}

// ActionableFrameworkPanic is a core package helper.
func ActionableFrameworkPanic(parseOptions ActionablePanicOptions) string {
	parseContext := buildActionablePanicContext(parseOptions)
	reportDiagnosticWithContextDetails(
		parseContext.Source,
		DiagnosticError,
		parseContext.Summary,
		parseContext.Path,
		parseContext.ComponentStack,
		parseContext.TopFrame,
		parseContext.Consequence,
		nil,
	)
	return formatActionablePanicReport(parseContext)
}

// buildUnhandledPanicReport is a core package helper.
func buildUnhandledPanicReport(parseSource string, parsePhase PanicPhase, parseSubject string, parsePath string, parseComponentStack []string, parseRecovered interface{}) PanicReport {
	parseContext := buildPanicReportContext(parseSource, parsePhase, parseSubject, parsePath, parseComponentStack, parseRecovered)
	reportDiagnosticWithContextDetails(
		parseContext.Source,
		DiagnosticError,
		fmt.Sprintf("uncaught %s panic in %s: %s", parseContext.Phase, parseContext.Subject, parseContext.Summary),
		parseContext.Path,
		parseContext.ComponentStack,
		parseContext.TopFrame,
		parseContext.Consequence,
		mergePanicReportFields(map[string]string{
			"phase":   string(parseContext.Phase),
			"where":   parseContext.Subject,
			"summary": parseContext.Summary,
		}, panicArtifactFields(parseContext.Artifact)),
	)
	return buildPanicReport(parseContext)
}

// ReportUnhandledPanicContext is a core package helper.
func ReportUnhandledPanicContext(parseSource string, parsePhase PanicPhase, parseSubject string, parsePath string, parseComponentStack []string, parseRecovered interface{}) string {
	if parseMessage, parseOk := wrappedPanicString(parseRecovered); parseOk {
		return parseMessage
	}
	return buildUnhandledPanicReport(parseSource, parsePhase, parseSubject, parsePath, parseComponentStack, parseRecovered).Formatted
}

// recoveredAsError is a core package helper.
func recoveredAsError(parseRecovered interface{}) error {
	switch parseTyped := parseRecovered.(type) {
	case nil:
		return nil
	case error:
		return parseTyped
	case string:
		return fmt.Errorf("%s", parseTyped)
	default:
		return fmt.Errorf("%v", parseTyped)
	}
}

// finalizeUnhandledPanicContext is a core package helper.
func finalizeUnhandledPanicContext(parseSource string, parsePhase PanicPhase, parseSubject string, parsePath string, parseComponentStack []string, parseRecovered interface{}) (interface{}, bool) {
	if parseOriginal, parseOk := unwrapReportedPanic(parseRecovered); parseOk {
		if shouldHideRawPanicOutput() {
			return parseOriginal, true
		}
		panic(parseOriginal)
	}
	parseReport := buildUnhandledPanicReport(parseSource, parsePhase, parseSubject, parsePath, parseComponentStack, parseRecovered)
	emitWrappedPanicReport(parseReport)
	if shouldHideRawPanicOutput() {
		return parseRecovered, true
	}
	panic(parseRecovered)
}

// markUnhandledPanicContext is a core package helper.
func markUnhandledPanicContext(parseSource string, parsePhase PanicPhase, parseSubject string, parsePath string, parseComponentStack []string, parseRecovered interface{}) interface{} {
	if parseOriginal, parseOk := unwrapReportedPanic(parseRecovered); parseOk {
		return reportedPanic{Original: parseOriginal}
	}
	parseReport := buildUnhandledPanicReport(parseSource, parsePhase, parseSubject, parsePath, parseComponentStack, parseRecovered)
	emitWrappedPanicReport(parseReport)
	return reportedPanic{Original: parseRecovered}
}

// panicFinalUnhandledPanicContext is a core package helper.
func panicFinalUnhandledPanicContext(parseSource string, parsePhase PanicPhase, parseSubject string, parsePath string, parseComponentStack []string, parseRecovered interface{}) {
	_, _ = finalizeUnhandledPanicContext(parseSource, parsePhase, parseSubject, parsePath, parseComponentStack, parseRecovered)
}

// FinalizeUnhandledPanicContext is a core package helper.
func FinalizeUnhandledPanicContext(parseSource string, parsePhase PanicPhase, parseSubject string, parsePath string, parseComponentStack []string, parseRecovered interface{}) (interface{}, bool) {
	return finalizeUnhandledPanicContext(parseSource, parsePhase, parseSubject, parsePath, parseComponentStack, parseRecovered)
}
