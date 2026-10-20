package runtime

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/v7/internal/errcode"
)

type diagnosticDetails struct {
	Code        string
	Docs        string
	Remediation string
	Recoverable bool
}

// diagnosticMatcher maps message fragments to a registered code.
type diagnosticMatcher struct {
	Code    errcode.Code
	Needles []string
}

// diagnosticMatchers lists the message fragments recognised by the fallback
// matcher. Order matters: the first matcher with any fragment present wins.
var diagnosticMatchers = []diagnosticMatcher{
	{errcode.RuntimePanicRender, []string{"uncaught render panic in "}},
	{errcode.RuntimePanicEvent, []string{"uncaught event panic in "}},
	{errcode.RuntimePanicEffect, []string{"uncaught effect panic in "}},
	{errcode.RuntimePanicCleanup, []string{"uncaught cleanup panic in "}},
	{errcode.RuntimePanicLoader, []string{"uncaught loader panic in "}},
	{errcode.RuntimePanicHydration, []string{"uncaught hydration panic in "}},
	{errcode.RuntimePanicStartup, []string{"uncaught startup panic in "}},
	{errcode.RuntimePanicDeferred, []string{"uncaught deferred panic in "}},
	{errcode.RuntimePanicSSR, []string{"uncaught ssr panic in "}},
	{errcode.RuntimeHookOutsideComponent, []string{"called outside component context"}},
	{errcode.RuntimeHookThreading, []string{"called from a goroutine outside the render goroutine"}},
	{errcode.RuntimeHookFuncType, []string{"gousefunc requires a function"}},
	{errcode.RuntimeDOMAdapterNil, []string{"gousefunc dom adapter is nil"}},
	{errcode.RuntimeAtomRegistryNil, []string{"runtime atom registry not initialized"}},
	{errcode.RuntimeAtomAccessorMismatch, []string{"gouseatom accessor cache type mismatch"}},
	{errcode.UIContextNil, []string{"called with nil context descriptor"}},
	{errcode.UICreateElementType, []string{
		"ui.createelement requires a component function or ui.node",
		"ui.createelement components may accept at most one props argument",
		"ui.createelement components must return ui.node",
	}},
	{errcode.UIUnsupportedOnServer, []string{"is not available on non-js/wasm builds in the current ssr slice"}},
	{errcode.RouterComponentNil, []string{"route component cannot be nil"}},
	{errcode.RouterComponentType, []string{"unsupported route component type"}},
	{errcode.RouterComponentArity, []string{"route component must return exactly one element"}},
	{errcode.RouterDuplicateRoute, []string{
		"replacing existing catch-all route registration",
		"replacing existing pattern route registration",
		"replacing existing route registration",
	}},
	{errcode.RouterRedirectLoop, []string{"redirect loop"}},
	{errcode.RouterLoaderFailed, []string{"route loader failed"}},
	{errcode.HydrationTextMismatch, []string{"hydration text mismatch"}},
	{errcode.HydrationAttributeMismatch, []string{"hydration attribute mismatch"}},
	{errcode.HydrationDiscardedNodes, []string{"hydration discarded unexpected dom nodes"}},
	{errcode.HydrationFallback, []string{"hydration fell back to client rendering"}},
	{errcode.RuntimeContainerNotFound, []string{"target container selector was not found"}},
}

// matchDiagnosticCode returns the registered code whose fragment appears in a
// lowercased message, or "" when nothing matches.
func matchDiagnosticCode(parseLower string) errcode.Code {
	for _, parseMatcher := range diagnosticMatchers {
		for _, parseNeedle := range parseMatcher.Needles {
			if strings.Contains(parseLower, parseNeedle) {
				return parseMatcher.Code
			}
		}
	}
	return ""
}

// getDiagnosticDetails builds the report details for a registered code.
func getDiagnosticDetails(parseCode errcode.Code, parseSeverity DiagnosticSeverity, parseRecoverable bool) diagnosticDetails {
	parseEntry, parseOk := errcode.Lookup(parseCode)
	if !parseOk {
		return diagnosticDetails{Recoverable: parseRecoverable}
	}
	parseDetails := diagnosticDetails{
		Code:        string(parseEntry.Code),
		Docs:        parseEntry.Docs,
		Remediation: parseEntry.Remediation,
		Recoverable: parseRecoverable,
	}
	if parseEntry.Recoverable {
		parseDetails.Recoverable = true
	}
	if parseEntry.RecoverableUnlessError {
		parseDetails.Recoverable = parseSeverity != DiagnosticError
	}
	return parseDetails
}

// diagnosticMetadata resolves a message to its registered code details, using
// the fragment matcher over the registry.
func diagnosticMetadata(parseSource string, parseSeverity DiagnosticSeverity, parseClassification DiagnosticClassification, parseMessage string) diagnosticDetails {
	_ = parseSource
	parseLower := strings.ToLower(strings.TrimSpace(parseMessage))
	parseRecoverable := parseClassification == DiagnosticRecovered || parseClassification == DiagnosticUnsupportedRecover
	// A message produced by errcode.Error carries its registered code in a
	// leading "[GWC-...]" token; trust it before the fragment matcher.
	if parseCode := parseEmbeddedDiagnosticCode(parseMessage); parseCode != "" {
		return getDiagnosticDetails(parseCode, parseSeverity, parseRecoverable)
	}
	return getDiagnosticDetails(matchDiagnosticCode(parseLower), parseSeverity, parseRecoverable)
}

// logSeverity is a core package helper.
func logSeverity(parseLevel LogLevel) DiagnosticSeverity {
	switch parseLevel {
	case LogError:
		return DiagnosticError
	case LogWarn:
		return DiagnosticWarning
	default:
		return DiagnosticInfo
	}
}

// actionableHookUsagePanic is a core package helper.
func actionableHookUsagePanic(parseName string) string {
	parseTrimmed := strings.TrimSpace(parseName)
	return ActionableFrameworkPanic(ActionablePanicOptions{
		Source:  "runtime",
		Subject: parseTrimmed,
		Message: fmt.Sprintf("%s called outside component context", parseTrimmed),
		Path:    parseTrimmed,
	})
}

// actionableHookThreadingPanic is a core package helper.
func actionableHookThreadingPanic(parseName string, parseFiber *Fiber) string {
	parseTrimmed := strings.TrimSpace(parseName)
	return ActionableFrameworkPanic(ActionablePanicOptions{
		Source:         "runtime",
		Subject:        parseTrimmed,
		Message:        hookThreadingViolationMessage(parseTrimmed),
		Path:           diagnosticPathForFiber(parseFiber),
		ComponentStack: diagnosticComponentStack(parseFiber),
		Consequence:    "hook state is owned by the active render goroutine; continuing from another goroutine would corrupt component state",
	})
}

// hookThreadingViolationMessage is a core package helper.
func hookThreadingViolationMessage(parseName string) string {
	parseTrimmed := strings.TrimSpace(parseName)
	if parseTrimmed == "" {
		parseTrimmed = "hook"
	}
	return fmt.Sprintf("%s called from a goroutine outside the render goroutine", parseTrimmed)
}

// actionableContextDescriptorNilPanic is a core package helper.
func actionableContextDescriptorNilPanic(parseName string) string {
	parseTrimmed := strings.TrimSpace(parseName)
	return ActionableFrameworkPanic(ActionablePanicOptions{
		Source:  "runtime",
		Subject: parseTrimmed,
		Message: fmt.Sprintf("%s called with nil context descriptor", parseTrimmed),
		Path:    parseTrimmed,
	})
}

// actionableGoUseFuncTypePanic is a core package helper.
func actionableGoUseFuncTypePanic() string {
	return ActionableFrameworkPanic(ActionablePanicOptions{
		Source:  "runtime",
		Subject: "GoUseFunc",
		Message: "GoUseFunc requires a function",
		Path:    "GoUseFunc",
	})
}

// actionableGoUseFuncDOMAdapterPanic is a core package helper.
func actionableGoUseFuncDOMAdapterPanic() string {
	return ActionableFrameworkPanic(ActionablePanicOptions{
		Source:  "runtime",
		Subject: "GoUseFunc",
		Message: "GoUseFunc dom adapter is nil",
		Path:    "GoUseFunc",
	})
}

// actionableRuntimeDOMAdapterPanic is a core package helper.
func actionableRuntimeDOMAdapterPanic(parseSubject string) string {
	return ActionableFrameworkPanic(ActionablePanicOptions{
		Source:  "runtime",
		Subject: parseSubject,
		Message: parseSubject + " requires a DOM adapter",
		Path:    parseSubject,
	})
}

// actionableGoUseAtomRegistryPanic is a core package helper.
func actionableGoUseAtomRegistryPanic() string {
	return ActionableFrameworkPanic(ActionablePanicOptions{
		Source:  "runtime",
		Subject: "GoUseAtom",
		Message: "Runtime atom registry not initialized",
		Path:    "GoUseAtom",
	})
}

// actionableGoUseAtomAccessorPanic is a core package helper.
func actionableGoUseAtomAccessorPanic() string {
	return ActionableFrameworkPanic(ActionablePanicOptions{
		Source:  "runtime",
		Subject: "GoUseAtom",
		Message: "GoUseAtom accessor cache type mismatch",
		Path:    "GoUseAtom",
	})
}

// getPanicPhaseCode maps a panic phase to its registered code.
func getPanicPhaseCode(parsePhase boundaryPhase) errcode.Code {
	switch parsePhase {
	case boundaryPhaseRender:
		return errcode.RuntimePanicRender
	case boundaryPhaseEvent:
		return errcode.RuntimePanicEvent
	case boundaryPhaseEffect:
		return errcode.RuntimePanicEffect
	case boundaryPhaseCleanup:
		return errcode.RuntimePanicCleanup
	case PanicPhaseLoader:
		return errcode.RuntimePanicLoader
	case PanicPhaseHydration:
		return errcode.RuntimePanicHydration
	case PanicPhaseStartup:
		return errcode.RuntimePanicStartup
	case PanicPhaseDeferred:
		return errcode.RuntimePanicDeferred
	case PanicPhaseSSR:
		return errcode.RuntimePanicSSR
	case PanicPhaseAsync:
		return errcode.RuntimePanicAsync
	default:
		return errcode.RuntimePanic
	}
}

// panicDiagnosticCode is a core package helper.
func panicDiagnosticCode(parsePhase boundaryPhase) string {
	return string(getPanicPhaseCode(parsePhase))
}

// panicDiagnosticDocs is a core package helper.
func panicDiagnosticDocs(parsePhase boundaryPhase) string {
	parseEntry, _ := errcode.Lookup(getPanicPhaseCode(parsePhase))
	return parseEntry.Docs
}

// panicDiagnosticRemediation is a core package helper.
func panicDiagnosticRemediation(parsePhase boundaryPhase) string {
	parseEntry, _ := errcode.Lookup(getPanicPhaseCode(parsePhase))
	return fmt.Sprintf("Match code %s in automation; ", parseEntry.Code) + parseEntry.PhaseRemediation
}

// panicSummary is a core package helper.
func panicSummary(parseRecovered any) string {
	parseErr := normalizeBoundaryError(parseRecovered)
	parseMessage := strings.TrimSpace(parseErr.Error())
	if parseMessage == "" {
		return "panic without message"
	}
	return parseMessage
}

// panicSubject is a core package helper.
func panicSubject(parseFiber *Fiber) string {
	parseStack := diagnosticComponentStack(parseFiber)
	if len(parseStack) > 0 {
		return parseStack[len(parseStack)-1]
	}
	return "application"
}

// panicDiagnosticMessage is a core package helper.
func panicDiagnosticMessage(parseFiber *Fiber, parsePhase boundaryPhase, parseRecovered any) string {
	return fmt.Sprintf("uncaught %s panic in %s: %s", parsePhase, panicSubject(parseFiber), panicSummary(parseRecovered))
}

// reportUnhandledPanic is a core package helper.
func reportUnhandledPanic(parseFiber *Fiber, parsePhase boundaryPhase, parseRecovered any) string {
	return ReportUnhandledPanicContext(
		"runtime",
		parsePhase,
		panicSubject(parseFiber),
		diagnosticPathForFiber(parseFiber),
		diagnosticComponentStack(parseFiber),
		parseRecovered,
	)
}

// markUnhandledPanic is a core package helper.
func markUnhandledPanic(parseFiber *Fiber, parsePhase boundaryPhase, parseRecovered any) any {
	return markUnhandledPanicContext(
		"runtime",
		parsePhase,
		panicSubject(parseFiber),
		diagnosticPathForFiber(parseFiber),
		diagnosticComponentStack(parseFiber),
		parseRecovered,
	)
}

// panicFinalUnhandledPanic is a core package helper.
func panicFinalUnhandledPanic(parseFiber *Fiber, parsePhase boundaryPhase, parseRecovered any) {
	panicFinalUnhandledPanicContext(
		"runtime",
		parsePhase,
		panicSubject(parseFiber),
		diagnosticPathForFiber(parseFiber),
		diagnosticComponentStack(parseFiber),
		parseRecovered,
	)
}

// parseEmbeddedDiagnosticCode returns the registered code named by a leading
// "[GWC-...]" token in a message, or "" when absent or unregistered.
func parseEmbeddedDiagnosticCode(parseMessage string) errcode.Code {
	parseTrimmed := strings.TrimSpace(parseMessage)
	if !strings.HasPrefix(parseTrimmed, "[GWC-") {
		return ""
	}
	parseEnd := strings.IndexByte(parseTrimmed, ']')
	if parseEnd < 0 {
		return ""
	}
	parseCode := errcode.Code(parseTrimmed[1:parseEnd])
	if _, parseOk := errcode.Lookup(parseCode); !parseOk {
		return ""
	}
	return parseCode
}
