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

func init() {
	hideRawPanicOutput.Store(true)
}

// ConfigureUnhandledPanicLogging sets the panic logging options including raw output visibility and the report hook.
func ConfigureUnhandledPanicLogging(options PanicLoggingOptions) {
	hideRawPanicOutput.Store(options.HideRawPanicOutput)
	panicLoggingHookMu.Lock()
	panicLoggingHook = options.OnReport
	panicLoggingHookMu.Unlock()
}

// CurrentUnhandledPanicLoggingOptions returns a snapshot of the current panic logging configuration.
func CurrentUnhandledPanicLoggingOptions() PanicLoggingOptions {
	panicLoggingHookMu.RLock()
	hook := panicLoggingHook
	panicLoggingHookMu.RUnlock()
	return PanicLoggingOptions{HideRawPanicOutput: hideRawPanicOutput.Load(), OnReport: hook}
}

func shouldHideRawPanicOutput() bool {
	return hideRawPanicOutput.Load()
}

func wrappedPanicString(recovered interface{}) (string, bool) {
	message, ok := recovered.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "", false
	}
	if strings.Contains(trimmed, "\n[GWC-") || strings.HasPrefix(trimmed, "[GWC-") {
		return message, true
	}
	return "", false
}

func mergePanicReportFields(primary map[string]string, extra map[string]string) map[string]string {
	if len(primary) == 0 && len(extra) == 0 {
		return nil
	}
	merged := map[string]string{}
	for key, value := range primary {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func unwrapReportedPanic(recovered interface{}) (interface{}, bool) {
	current := recovered
	unwrapped := false
	for {
		switch typed := current.(type) {
		case reportedPanic:
			unwrapped = true
			current = typed.Original
		case *reportedPanic:
			unwrapped = true
			if typed == nil {
				return nil, true
			}
			current = typed.Original
		default:
			if !unwrapped {
				return nil, false
			}
			return current, true
		}
	}
}

func currentPanicReportHook() func(PanicReport) {
	panicLoggingHookMu.RLock()
	defer panicLoggingHookMu.RUnlock()
	return panicLoggingHook
}

func clonePanicReport(report PanicReport) PanicReport {
	cloned := report
	cloned.ComponentStack = append([]string(nil), report.ComponentStack...)
	cloned.AppFrames = append([]string(nil), report.AppFrames...)
	cloned.FrameworkFrames = append([]string(nil), report.FrameworkFrames...)
	cloned.PlatformFrames = append([]string(nil), report.PlatformFrames...)
	return cloned
}

func emitWrappedPanicReport(report PanicReport) {
	report.Formatted = strings.TrimSpace(report.Formatted)
	if report.Formatted == "" {
		return
	}
	if hook := currentPanicReportHook(); hook != nil {
		func() {
			defer func() { _ = recover() }()
			hook(clonePanicReport(report))
		}()
	}
	if emitBrowserPanicReport(report) {
		return
	}
	fmt.Println(report.Formatted)
}

func parsePanicFrames(stack []byte) []panicFrame {
	lines := strings.Split(strings.ReplaceAll(string(stack), "\r\n", "\n"), "\n")
	frames := make([]panicFrame, 0, len(lines)/2)
	for index := 1; index+1 < len(lines); index += 2 {
		function := strings.TrimSpace(lines[index])
		location := strings.TrimSpace(lines[index+1])
		if function == "" || location == "" {
			continue
		}
		if strings.Contains(function, "runtime/debug.Stack") ||
			strings.Contains(function, "github.com/monstercameron/GoWebComponents/internal/runtime.parsePanicFrames") ||
			strings.Contains(function, "github.com/monstercameron/GoWebComponents/internal/runtime.buildPanicReportContext") ||
			strings.Contains(function, "github.com/monstercameron/GoWebComponents/internal/runtime.ReportUnhandledPanicContext") ||
			strings.Contains(function, "github.com/monstercameron/GoWebComponents/internal/runtime.reportUnhandledPanic") {
			continue
		}

		location = strings.TrimSpace(strings.SplitN(location, " +", 2)[0])
		line := 0
		if lastColon := strings.LastIndex(location, ":"); lastColon > 0 {
			if parsed, err := strconv.Atoi(location[lastColon+1:]); err == nil {
				line = parsed
				location = location[:lastColon]
			}
		}

		frames = append(frames, translateWASMStackFrame(panicFrame{Function: function, File: location, Line: line}))
	}
	return frames
}

func sanitizePanicFunction(function string) string {
	trimmed := strings.TrimSpace(function)
	if trimmed == "" {
		return ""
	}
	if strings.HasSuffix(trimmed, ")") {
		if index := strings.LastIndex(trimmed, "("); index > 0 {
			trimmed = trimmed[:index]
		}
	}
	return strings.TrimSpace(trimmed)
}

func classifyPanicFrame(frame panicFrame) string {
	function := strings.ReplaceAll(sanitizePanicFunction(frame.Function), "\\", "/")
	file := strings.ReplaceAll(frame.File, "\\", "/")
	if strings.HasSuffix(file, "_test.go") || strings.Contains(function, ".Test") {
		return "app"
	}

	if strings.Contains(function, frameworkModulePath+"/") {
		if strings.Contains(function, frameworkModulePath+"/examples/") || strings.Contains(function, frameworkModulePath+"/test/") {
			return "app"
		}
		return "framework"
	}

	if strings.Contains(file, "/"+frameworkWorkspaceName+"/") {
		if strings.Contains(file, "/"+frameworkWorkspaceName+"/examples/") || strings.Contains(file, "/"+frameworkWorkspaceName+"/test/") {
			return "app"
		}
		return "framework"
	}

	if strings.HasPrefix(function, "runtime.") ||
		strings.HasPrefix(function, "syscall/js.") ||
		strings.HasPrefix(function, "testing.") ||
		strings.Contains(file, "/src/runtime/") ||
		strings.Contains(file, "/src/testing/") ||
		strings.Contains(file, "/src/syscall/js/") {
		return "platform"
	}

	return "app"
}

func lowSignalPanicFrame(frame panicFrame, bucket string) bool {
	function := strings.ReplaceAll(sanitizePanicFunction(frame.Function), "\\", "/")
	if function == "" {
		return true
	}
	switch bucket {
	case "framework":
		for _, token := range []string{
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
			if strings.Contains(function, token) {
				return true
			}
		}
	case "platform":
		if function == "panic" || strings.HasPrefix(function, "runtime.panic") {
			return true
		}
	}
	return false
}

func shortenPanicFilePath(path string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(path), "\\", "/")
	if normalized == "" {
		return ""
	}
	marker := "/" + frameworkWorkspaceName + "/"
	if index := strings.Index(normalized, marker); index >= 0 {
		return normalized[index+len(marker):]
	}
	parts := strings.Split(normalized, "/")
	if len(parts) <= 3 {
		return normalized
	}
	return strings.Join(parts[len(parts)-3:], "/")
}

func formatPanicFrame(frame panicFrame) string {
	function := sanitizePanicFunction(frame.Function)
	location := shortenPanicFilePath(frame.File)
	if location == "" {
		return function
	}
	if frame.Line > 0 {
		return fmt.Sprintf("%s at %s:%d", function, location, frame.Line)
	}
	return fmt.Sprintf("%s at %s", function, location)
}

func appendPanicFrame(section []string, frame panicFrame) []string {
	formatted := formatPanicFrame(frame)
	if formatted == "" {
		return section
	}
	return append(section, formatted)
}

func limitPanicFrames(values []string, limit int, label string) []string {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	trimmed := append([]string(nil), values[:limit]...)
	trimmed = append(trimmed, fmt.Sprintf("... %d more %s frames omitted", len(values)-limit, label))
	return trimmed
}

func panicConsequence(phase PanicPhase) string {
	switch phase {
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

func panicPhaseMayRecoverWithBoundary(phase PanicPhase) bool {
	switch phase {
	case PanicPhaseRender, PanicPhaseEvent, PanicPhaseEffect, PanicPhaseCleanup:
		return true
	default:
		return false
	}
}

func buildPanicReportContext(source string, phase PanicPhase, subject string, path string, componentStack []string, recovered interface{}) panicReportContext {
	frames := parsePanicFrames(debug.Stack())
	context := panicReportContext{
		Source:         strings.TrimSpace(source),
		Phase:          phase,
		Subject:        strings.TrimSpace(subject),
		Path:           strings.TrimSpace(path),
		ComponentStack: append([]string(nil), componentStack...),
		Summary:        panicSummary(recovered),
		Code:           panicDiagnosticCode(phase),
		Docs:           panicDiagnosticDocs(phase),
		Remediation:    panicDiagnosticRemediation(phase),
		Consequence:    panicConsequence(phase),
		Artifact:       currentWASMArtifactMetadata(),
	}
	if context.Source == "" {
		context.Source = "runtime"
	}
	if context.Subject == "" {
		context.Subject = "application"
	}

	for _, frame := range frames {
		bucket := classifyPanicFrame(frame)
		if lowSignalPanicFrame(frame, bucket) {
			continue
		}
		switch bucket {
		case "app":
			context.AppFrames = appendPanicFrame(context.AppFrames, frame)
		case "framework":
			context.FrameworkFrames = appendPanicFrame(context.FrameworkFrames, frame)
		default:
			context.PlatformFrames = appendPanicFrame(context.PlatformFrames, frame)
		}
	}

	if len(context.AppFrames) > 0 {
		context.TopFrame = context.AppFrames[0]
	}
	if context.Path == "" && len(context.ComponentStack) > 0 {
		context.Path = strings.Join(context.ComponentStack, " > ")
	}
	if context.Path == "" {
		context.Path = context.Subject
	}
	context.AppFrames = limitPanicFrames(context.AppFrames, visibleAppFrameLimit, "app")
	context.FrameworkFrames = limitPanicFrames(context.FrameworkFrames, visibleFrameworkFrameLimit, "framework")
	context.PlatformFrames = limitPanicFrames(context.PlatformFrames, visiblePlatformFrameLimit, "platform")
	return context
}

func formatPanicStackSection(lines []string, values []string) []string {
	for _, value := range values {
		lines = append(lines, "  "+value)
	}
	return lines
}

func buildPanicReport(context panicReportContext) PanicReport {
	where := context.TopFrame
	if where == "" {
		where = context.Path
	}
	lines := []string{
		context.Summary,
		fmt.Sprintf("[%s] uncaught %s panic in %s", context.Code, context.Phase, context.Subject),
		"where: " + where,
		"path: " + context.Path,
		"error: " + context.Summary,
		"runtime: " + context.Consequence,
		"next: " + context.Remediation,
		"docs: " + context.Docs,
	}
	if artifactFields := panicArtifactFields(context.Artifact); len(artifactFields) > 0 {
		parts := make([]string, 0, len(artifactFields))
		for _, key := range []string{"artifact_build_id", "artifact_path", "artifact_sha256", "artifact_manifest", "artifact_symbols", "artifact_version"} {
			if value := strings.TrimSpace(artifactFields[key]); value != "" {
				parts = append(parts, key+"="+value)
			}
		}
		if len(parts) > 0 {
			lines = append(lines, "artifact: "+strings.Join(parts, " | "))
		}
	}
	if len(context.AppFrames) > 0 || len(context.FrameworkFrames) > 0 || len(context.PlatformFrames) > 0 {
		lines = append(lines, "stack:")
		if len(context.AppFrames) > 0 {
			lines = append(lines, "app:")
			lines = formatPanicStackSection(lines, context.AppFrames)
		}
		if len(context.FrameworkFrames) > 0 {
			lines = append(lines, "framework: GWC")
			lines = formatPanicStackSection(lines, context.FrameworkFrames)
		}
		if len(context.PlatformFrames) > 0 {
			lines = append(lines, "platform: GOLANG")
			lines = formatPanicStackSection(lines, context.PlatformFrames)
		}
	}
	return PanicReport{
		Source:          context.Source,
		Phase:           context.Phase,
		Subject:         context.Subject,
		Where:           where,
		Path:            context.Path,
		ComponentStack:  append([]string(nil), context.ComponentStack...),
		Summary:         context.Summary,
		Code:            context.Code,
		Docs:            context.Docs,
		Remediation:     context.Remediation,
		Consequence:     context.Consequence,
		TopFrame:        context.TopFrame,
		AppFrames:       append([]string(nil), context.AppFrames...),
		FrameworkFrames: append([]string(nil), context.FrameworkFrames...),
		PlatformFrames:  append([]string(nil), context.PlatformFrames...),
		Artifact:        context.Artifact,
		Formatted:       strings.Join(lines, "\n"),
	}
}

func formatPanicReport(context panicReportContext) string {
	return buildPanicReport(context).Formatted
}

func defaultActionablePanicConsequence(subject string) string {
	if strings.TrimSpace(subject) == "" {
		return "framework API validation failed; the current call aborted before runtime state could continue changing."
	}
	return fmt.Sprintf("framework API validation failed in %s; the current call aborted before runtime state could continue changing.", strings.TrimSpace(subject))
}

func buildActionablePanicContext(options ActionablePanicOptions) panicReportContext {
	source := strings.TrimSpace(options.Source)
	if source == "" {
		source = "runtime"
	}
	subject := strings.TrimSpace(options.Subject)
	if subject == "" {
		subject = source
	}
	summary := strings.TrimSpace(options.Message)
	if summary == "" {
		summary = "framework misuse without message"
	}
	details := diagnosticMetadata(source, DiagnosticError, DiagnosticCorrectness, summary)
	if strings.TrimSpace(details.Code) == "" {
		details.Code = "GWC-FRAMEWORK-USAGE"
		details.Docs = actionableErrorsDoc
		details.Remediation = "Inspect the where/path fields first, then correct the invalid framework API usage before retrying."
	}
	context := panicReportContext{
		Source:         source,
		Subject:        subject,
		Path:           strings.TrimSpace(options.Path),
		ComponentStack: append([]string(nil), options.ComponentStack...),
		Summary:        summary,
		Code:           details.Code,
		Docs:           details.Docs,
		Remediation:    details.Remediation,
		Consequence:    strings.TrimSpace(options.Consequence),
	}
	if context.Consequence == "" {
		context.Consequence = defaultActionablePanicConsequence(subject)
	}
	if context.Path == "" && len(context.ComponentStack) > 0 {
		context.Path = strings.Join(context.ComponentStack, " > ")
	}
	if context.Path == "" {
		context.Path = subject
	}
	for _, frame := range parsePanicFrames(debug.Stack()) {
		bucket := classifyPanicFrame(frame)
		if lowSignalPanicFrame(frame, bucket) {
			continue
		}
		switch bucket {
		case "app":
			context.AppFrames = appendPanicFrame(context.AppFrames, frame)
		case "framework":
			context.FrameworkFrames = appendPanicFrame(context.FrameworkFrames, frame)
		default:
			context.PlatformFrames = appendPanicFrame(context.PlatformFrames, frame)
		}
	}
	if len(context.AppFrames) > 0 {
		context.TopFrame = context.AppFrames[0]
	}
	context.AppFrames = limitPanicFrames(context.AppFrames, visibleAppFrameLimit, "app")
	context.FrameworkFrames = limitPanicFrames(context.FrameworkFrames, visibleFrameworkFrameLimit, "framework")
	context.PlatformFrames = limitPanicFrames(context.PlatformFrames, visiblePlatformFrameLimit, "platform")
	return context
}

func formatActionablePanicReport(context panicReportContext) string {
	where := context.TopFrame
	if where == "" {
		where = context.Path
	}
	lines := []string{
		context.Summary,
		fmt.Sprintf("[%s] framework misuse in %s", context.Code, context.Subject),
		"where: " + where,
		"path: " + context.Path,
		"error: " + context.Summary,
		"runtime: " + context.Consequence,
		"next: " + context.Remediation,
		"docs: " + context.Docs,
	}
	if len(context.AppFrames) > 0 || len(context.FrameworkFrames) > 0 || len(context.PlatformFrames) > 0 {
		lines = append(lines, "stack:")
		if len(context.AppFrames) > 0 {
			lines = append(lines, "app:")
			lines = formatPanicStackSection(lines, context.AppFrames)
		}
		if len(context.FrameworkFrames) > 0 {
			lines = append(lines, "framework: GWC")
			lines = formatPanicStackSection(lines, context.FrameworkFrames)
		}
		if len(context.PlatformFrames) > 0 {
			lines = append(lines, "platform: GOLANG")
			lines = formatPanicStackSection(lines, context.PlatformFrames)
		}
	}
	return strings.Join(lines, "\n")
}

func ActionableFrameworkPanic(options ActionablePanicOptions) string {
	context := buildActionablePanicContext(options)
	reportDiagnosticWithContextDetails(
		context.Source,
		DiagnosticError,
		context.Summary,
		context.Path,
		context.ComponentStack,
		context.TopFrame,
		context.Consequence,
		nil,
	)
	return formatActionablePanicReport(context)
}

func buildUnhandledPanicReport(source string, phase PanicPhase, subject string, path string, componentStack []string, recovered interface{}) PanicReport {
	context := buildPanicReportContext(source, phase, subject, path, componentStack, recovered)
	reportDiagnosticWithContextDetails(
		context.Source,
		DiagnosticError,
		fmt.Sprintf("uncaught %s panic in %s: %s", context.Phase, context.Subject, context.Summary),
		context.Path,
		context.ComponentStack,
		context.TopFrame,
		context.Consequence,
		mergePanicReportFields(map[string]string{
			"phase":   string(context.Phase),
			"where":   context.Subject,
			"summary": context.Summary,
		}, panicArtifactFields(context.Artifact)),
	)
	return buildPanicReport(context)
}

func ReportUnhandledPanicContext(source string, phase PanicPhase, subject string, path string, componentStack []string, recovered interface{}) string {
	if message, ok := wrappedPanicString(recovered); ok {
		return message
	}
	return buildUnhandledPanicReport(source, phase, subject, path, componentStack, recovered).Formatted
}

func recoveredAsError(recovered interface{}) error {
	switch typed := recovered.(type) {
	case nil:
		return nil
	case error:
		return typed
	case string:
		return fmt.Errorf("%s", typed)
	default:
		return fmt.Errorf("%v", typed)
	}
}

func finalizeUnhandledPanicContext(source string, phase PanicPhase, subject string, path string, componentStack []string, recovered interface{}) (interface{}, bool) {
	if original, ok := unwrapReportedPanic(recovered); ok {
		if shouldHideRawPanicOutput() {
			return original, true
		}
		panic(original)
	}
	report := buildUnhandledPanicReport(source, phase, subject, path, componentStack, recovered)
	emitWrappedPanicReport(report)
	if shouldHideRawPanicOutput() {
		return recovered, true
	}
	panic(recovered)
}

func markUnhandledPanicContext(source string, phase PanicPhase, subject string, path string, componentStack []string, recovered interface{}) interface{} {
	if original, ok := unwrapReportedPanic(recovered); ok {
		return reportedPanic{Original: original}
	}
	report := buildUnhandledPanicReport(source, phase, subject, path, componentStack, recovered)
	emitWrappedPanicReport(report)
	return reportedPanic{Original: recovered}
}

func panicFinalUnhandledPanicContext(source string, phase PanicPhase, subject string, path string, componentStack []string, recovered interface{}) {
	_, _ = finalizeUnhandledPanicContext(source, phase, subject, path, componentStack, recovered)
}

func FinalizeUnhandledPanicContext(source string, phase PanicPhase, subject string, path string, componentStack []string, recovered interface{}) (interface{}, bool) {
	return finalizeUnhandledPanicContext(source, phase, subject, path, componentStack, recovered)
}
