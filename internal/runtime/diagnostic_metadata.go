package runtime

import (
	"fmt"
	"strings"
)

const actionableErrorsDoc = "ACTIONABLE_ERRORS.md"

type diagnosticDetails struct {
	Code        string
	Docs        string
	Remediation string
	Recoverable bool
}

// diagnosticMetadata is a core package helper.
func diagnosticMetadata(parseSource string, parseSeverity DiagnosticSeverity, parseClassification DiagnosticClassification, parseMessage string) diagnosticDetails {
	parseLower := strings.ToLower(strings.TrimSpace(parseMessage))
	parseDetails := diagnosticDetails{
		Recoverable: parseClassification == DiagnosticRecovered || parseClassification == DiagnosticUnsupportedRecover,
	}

	switch {
	case strings.Contains(parseLower, "uncaught render panic in "):
		parseDetails.Code = "GWC-RUNTIME-PANIC-RENDER"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-runtime-panic-render"
		parseDetails.Remediation = "Inspect the component render path named in the diagnostic and replace panic-based control flow with guarded branches, fallback UI, or an error boundary around the failing subtree."
	case strings.Contains(parseLower, "uncaught event panic in "):
		parseDetails.Code = "GWC-RUNTIME-PANIC-EVENT"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-runtime-panic-event"
		parseDetails.Remediation = "Inspect the event handler named in the diagnostic, remove panic-based control flow, and return explicit errors or guarded updates instead of crashing the runtime callback."
	case strings.Contains(parseLower, "uncaught effect panic in "):
		parseDetails.Code = "GWC-RUNTIME-PANIC-EFFECT"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-runtime-panic-effect"
		parseDetails.Remediation = "Inspect the effect body for the named component and move failure-prone work behind validation, explicit error handling, or an error boundary-friendly fallback path."
	case strings.Contains(parseLower, "uncaught cleanup panic in "):
		parseDetails.Code = "GWC-RUNTIME-PANIC-CLEANUP"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-runtime-panic-cleanup"
		parseDetails.Remediation = "Inspect the cleanup function for the named component and make teardown idempotent so unmount or dependency changes do not panic mid-cleanup."
	case strings.Contains(parseLower, "uncaught loader panic in "):
		parseDetails.Code = "GWC-RUNTIME-PANIC-LOADER"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-runtime-panic-loader"
		parseDetails.Remediation = "Inspect the route loader or async data callback named in the diagnostic, remove panic-based control flow, and return explicit errors so route error UI can recover without a fatal exit."
	case strings.Contains(parseLower, "uncaught hydration panic in "):
		parseDetails.Code = "GWC-RUNTIME-PANIC-HYDRATION"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-runtime-panic-hydration"
		parseDetails.Remediation = "Inspect the first client render inputs and server markup, then fix the mismatch or hydration-time panic before relying on strict resume paths."
	case strings.Contains(parseLower, "uncaught startup panic in "):
		parseDetails.Code = "GWC-RUNTIME-PANIC-STARTUP"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-runtime-panic-startup"
		parseDetails.Remediation = "Verify startup selectors, bootstrap inputs, and runtime initialization so mount-time failures do not terminate before the app can render."
	case strings.Contains(parseLower, "uncaught deferred panic in "):
		parseDetails.Code = "GWC-RUNTIME-PANIC-DEFERRED"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-runtime-panic-deferred"
		parseDetails.Remediation = "Inspect deferred callbacks, transition work, and scheduled runtime continuations so delayed work returns explicit errors instead of panicking."
	case strings.Contains(parseLower, "uncaught ssr panic in "):
		parseDetails.Code = "GWC-RUNTIME-PANIC-SSR"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-runtime-panic-ssr"
		parseDetails.Remediation = "Inspect the server render path and bootstrap generation code first, then surface the returned error instead of letting request-time rendering fail invisibly."
	case strings.Contains(parseLower, "called outside component context"):
		parseDetails.Code = "GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-runtime-hook-outside-component"
		parseDetails.Remediation = "Call framework hooks only while rendering a component through ui.CreateElement(...). Move the hook call out of package init code, route factories, and ordinary helpers that are not rendering components."
	case strings.Contains(parseLower, "gousefunc requires a function"):
		parseDetails.Code = "GWC-RUNTIME-HOOK-FUNC-TYPE"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-runtime-hook-func-type"
		parseDetails.Remediation = "Pass a real function to GoUseFunc and move any non-callable config or data values outside the event-hook wrapper."
	case strings.Contains(parseLower, "gousefunc dom adapter is nil"):
		parseDetails.Code = "GWC-RUNTIME-DOM-ADAPTER-NIL"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-runtime-dom-adapter-nil"
		parseDetails.Remediation = "Initialize the runtime with a DOM adapter before rendering components that call GoUseFunc or wire event handlers."
	case strings.Contains(parseLower, "runtime atom registry not initialized"):
		parseDetails.Code = "GWC-RUNTIME-ATOM-REGISTRY-NIL"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-runtime-atom-registry-nil"
		parseDetails.Remediation = "Create or initialize the runtime before calling GoUseAtom so the shared atom registry exists for subscriptions and updates."
	case strings.Contains(parseLower, "gouseatom accessor cache type mismatch"):
		parseDetails.Code = "GWC-RUNTIME-ATOM-ACCESSOR-MISMATCH"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-runtime-atom-accessor-mismatch"
		parseDetails.Remediation = "Do not reuse one hook slot for different atom value types. Keep GoUseAtom call order stable and preserve a single value shape per atom hook position."
	case strings.Contains(parseLower, "called with nil context descriptor"):
		parseDetails.Code = "GWC-UI-CONTEXT-NIL"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-ui-context-nil"
		parseDetails.Remediation = "Pass the descriptor returned by ui.CreateContext(...) and avoid nil placeholder contexts."
	case strings.Contains(parseLower, "ui.createelement requires a component function or ui.node") ||
		strings.Contains(parseLower, "ui.createelement components may accept at most one props argument") ||
		strings.Contains(parseLower, "ui.createelement components must return ui.node"):
		parseDetails.Code = "GWC-UI-CREATE-ELEMENT-TYPE"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-ui-create-element-type"
		parseDetails.Remediation = "Pass a component function or ui.Node to ui.CreateElement, keep component signatures to zero or one props argument, and return exactly one ui.Node tree."
	case strings.Contains(parseLower, "is not available on non-js/wasm builds in the current ssr slice"):
		parseDetails.Code = "GWC-UI-UNSUPPORTED-ON-SERVER"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-ui-unsupported-on-server"
		parseDetails.Remediation = "Use the SSR-safe alternative documented for this API, or call the browser-only API only from js/wasm builds after the client runtime is active."
	case strings.Contains(parseLower, "route component cannot be nil"):
		parseDetails.Code = "GWC-ROUTER-COMPONENT-NIL"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-router-component-nil"
		parseDetails.Remediation = "Register a concrete component function or static node for the route instead of leaving a nil placeholder."
	case strings.Contains(parseLower, "unsupported route component type"):
		parseDetails.Code = "GWC-ROUTER-COMPONENT-TYPE"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-router-component-type"
		parseDetails.Remediation = "Register a component function, ui.Node, or route-compatible element producer instead of a raw config or data value."
	case strings.Contains(parseLower, "route component must return exactly one element"):
		parseDetails.Code = "GWC-ROUTER-COMPONENT-ARITY"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-router-component-arity"
		parseDetails.Remediation = "Return exactly one element tree from the route component. Wrap siblings in ui.Fragment(...) when needed."
	case strings.Contains(parseLower, "replacing existing catch-all route registration") ||
		strings.Contains(parseLower, "replacing existing pattern route registration") ||
		strings.Contains(parseLower, "replacing existing route registration"):
		parseDetails.Code = "GWC-ROUTER-DUPLICATE-ROUTE"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-router-duplicate-route"
		parseDetails.Remediation = "Remove duplicate route registrations or confirm that last-write-wins replacement is intentional."
		parseDetails.Recoverable = true
	case strings.Contains(parseLower, "redirect loop"):
		parseDetails.Code = "GWC-ROUTER-REDIRECT-LOOP"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-router-redirect-loop"
		parseDetails.Remediation = "Compare redirect source and target normalization and ensure guards or default routes do not bounce between the same paths."
		parseDetails.Recoverable = parseSeverity != DiagnosticError
	case strings.Contains(parseLower, "route loader failed"):
		parseDetails.Code = "GWC-ROUTER-LOADER-FAILED"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-router-loader-failed"
		parseDetails.Remediation = "Inspect the loader inputs, network failure, and route-local error UI. Supply an Options.Error fallback when the route depends on remote data."
	case strings.Contains(parseLower, "hydration text mismatch"):
		parseDetails.Code = "GWC-HYDRATION-TEXT-MISMATCH"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-hydration-text-mismatch"
		parseDetails.Remediation = "Make the first client render deterministic with the server HTML. Recheck IDs, params, query-derived branches, time-dependent strings, and bootstrap data reuse."
		parseDetails.Recoverable = parseSeverity != DiagnosticError
	case strings.Contains(parseLower, "hydration attribute mismatch"):
		parseDetails.Code = "GWC-HYDRATION-ATTRIBUTE-MISMATCH"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-hydration-attribute-mismatch"
		parseDetails.Remediation = "Compare server-rendered attributes with the first client render inputs and verify metadata, params, query state, and SSR bootstrap reuse."
		parseDetails.Recoverable = parseSeverity != DiagnosticError
	case strings.Contains(parseLower, "hydration discarded unexpected dom nodes"):
		parseDetails.Code = "GWC-HYDRATION-DISCARDED-NODES"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-hydration-discarded-nodes"
		parseDetails.Remediation = "Ensure the server document does not inject extra nodes into the hydrated subtree and keep client-only placeholders outside the hydrated tree."
		parseDetails.Recoverable = parseSeverity != DiagnosticError
	case strings.Contains(parseLower, "hydration fell back to client rendering"):
		parseDetails.Code = "GWC-HYDRATION-FALLBACK"
		parseDetails.Docs = actionableErrorsDoc + "#gwc-hydration-fallback"
		parseDetails.Remediation = "Fix the underlying server-client mismatch instead of relying on fallback rendering. In strict hydration flows, treat the output as correctness-threatening until the mismatch is resolved."
		parseDetails.Recoverable = parseSeverity != DiagnosticError
	case strings.Contains(parseLower, "target container selector was not found"):
		parseDetails.Code = "GWC-RUNTIME-CONTAINER-NOT-FOUND"
		parseDetails.Docs = "TROUBLESHOOTING.md#broken-example-serving"
		parseDetails.Remediation = "Verify the target selector exists before calling ui.Render(...) or ui.Hydrate(...), and confirm the page mounted the expected root container."
	}

	_ = parseSource
	return parseDetails
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

// panicDiagnosticCode is a core package helper.
func panicDiagnosticCode(parsePhase boundaryPhase) string {
	switch parsePhase {
	case boundaryPhaseRender:
		return "GWC-RUNTIME-PANIC-RENDER"
	case boundaryPhaseEvent:
		return "GWC-RUNTIME-PANIC-EVENT"
	case boundaryPhaseEffect:
		return "GWC-RUNTIME-PANIC-EFFECT"
	case boundaryPhaseCleanup:
		return "GWC-RUNTIME-PANIC-CLEANUP"
	case PanicPhaseLoader:
		return "GWC-RUNTIME-PANIC-LOADER"
	case PanicPhaseHydration:
		return "GWC-RUNTIME-PANIC-HYDRATION"
	case PanicPhaseStartup:
		return "GWC-RUNTIME-PANIC-STARTUP"
	case PanicPhaseDeferred:
		return "GWC-RUNTIME-PANIC-DEFERRED"
	case PanicPhaseSSR:
		return "GWC-RUNTIME-PANIC-SSR"
	case PanicPhaseAsync:
		return "GWC-RUNTIME-PANIC-ASYNC"
	default:
		return "GWC-RUNTIME-PANIC"
	}
}

// panicDiagnosticDocs is a core package helper.
func panicDiagnosticDocs(parsePhase boundaryPhase) string {
	switch parsePhase {
	case boundaryPhaseRender:
		return actionableErrorsDoc + "#gwc-runtime-panic-render"
	case boundaryPhaseEvent:
		return actionableErrorsDoc + "#gwc-runtime-panic-event"
	case boundaryPhaseEffect:
		return actionableErrorsDoc + "#gwc-runtime-panic-effect"
	case boundaryPhaseCleanup:
		return actionableErrorsDoc + "#gwc-runtime-panic-cleanup"
	case PanicPhaseLoader:
		return actionableErrorsDoc + "#gwc-runtime-panic-loader"
	case PanicPhaseHydration:
		return actionableErrorsDoc + "#gwc-runtime-panic-hydration"
	case PanicPhaseStartup:
		return actionableErrorsDoc + "#gwc-runtime-panic-startup"
	case PanicPhaseDeferred:
		return actionableErrorsDoc + "#gwc-runtime-panic-deferred"
	case PanicPhaseSSR:
		return actionableErrorsDoc + "#gwc-runtime-panic-ssr"
	case PanicPhaseAsync:
		return actionableErrorsDoc + "#gwc-runtime-panic-async"
	default:
		return actionableErrorsDoc
	}
}

// panicDiagnosticRemediation is a core package helper.
func panicDiagnosticRemediation(parsePhase boundaryPhase) string {
	parseCode := panicDiagnosticCode(parsePhase)
	if strings.TrimSpace(parseCode) == "" {
		parseCode = "GWC-RUNTIME-PANIC"
	}
	parsePrefix := fmt.Sprintf("Match code %s in automation; ", parseCode)
	switch parsePhase {
	case PanicPhaseRender:
		return parsePrefix + "inspect the component render path named by where/path first; replace panic-based control flow with guarded branches, fallback UI, or an ErrorBoundary when local recovery is expected."
	case PanicPhaseEvent:
		return parsePrefix + "inspect the event handler named by where/path first; return explicit errors or guarded updates instead of panicking from runtime callbacks."
	case PanicPhaseEffect:
		return parsePrefix + "inspect the effect body named by where/path first; move failure-prone work behind validation, explicit errors, or a recoverable fallback path."
	case PanicPhaseCleanup:
		return parsePrefix + "inspect the cleanup path named by where/path first; make teardown idempotent and avoid panicking during unmount or dependency changes."
	case PanicPhaseLoader:
		return parsePrefix + "inspect the loader or async data path named by where/path first; return explicit errors so route error UI can recover instead of crashing route resolution."
	case PanicPhaseHydration:
		return parsePrefix + "compare server markup with the first client render at where/path first; fix the mismatch or hydration panic before relying on strict resume."
	case PanicPhaseStartup:
		return parsePrefix + "verify the startup target and bootstrap inputs named by where/path first; ensure mount selectors and initialization state exist before rendering."
	case PanicPhaseDeferred:
		return parsePrefix + "inspect the deferred callback or transition work named by where/path first; replace panic-based control flow with explicit errors or guarded branches."
	case PanicPhaseSSR:
		return parsePrefix + "inspect the server render path named by where/path first; surface the returned error to the request handler instead of treating it as a transport-level panic."
	case PanicPhaseAsync:
		return parsePrefix + "inspect the async task named by where/path first; the panicking goroutine was contained and abandoned, so fix its body to return explicit errors instead of panicking."
	default:
		return parsePrefix + "inspect the where/path fields first, then replace panic-based control flow with guarded branches, explicit errors, fallback UI, or a boundary when recovery is expected."
	}
}

// panicSummary is a core package helper.
func panicSummary(parseRecovered interface{}) string {
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
func panicDiagnosticMessage(parseFiber *Fiber, parsePhase boundaryPhase, parseRecovered interface{}) string {
	return fmt.Sprintf("uncaught %s panic in %s: %s", parsePhase, panicSubject(parseFiber), panicSummary(parseRecovered))
}

// reportUnhandledPanic is a core package helper.
func reportUnhandledPanic(parseFiber *Fiber, parsePhase boundaryPhase, parseRecovered interface{}) string {
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
func markUnhandledPanic(parseFiber *Fiber, parsePhase boundaryPhase, parseRecovered interface{}) interface{} {
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
func panicFinalUnhandledPanic(parseFiber *Fiber, parsePhase boundaryPhase, parseRecovered interface{}) {
	panicFinalUnhandledPanicContext(
		"runtime",
		parsePhase,
		panicSubject(parseFiber),
		diagnosticPathForFiber(parseFiber),
		diagnosticComponentStack(parseFiber),
		parseRecovered,
	)
}
