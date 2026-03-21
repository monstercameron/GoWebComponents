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

func diagnosticMetadata(source string, severity DiagnosticSeverity, classification DiagnosticClassification, message string) diagnosticDetails {
	lower := strings.ToLower(strings.TrimSpace(message))
	details := diagnosticDetails{
		Recoverable: classification == DiagnosticRecovered || classification == DiagnosticUnsupportedRecover,
	}

	switch {
	case strings.Contains(lower, "uncaught render panic in "):
		details.Code = "GWC-RUNTIME-PANIC-RENDER"
		details.Docs = actionableErrorsDoc + "#gwc-runtime-panic-render"
		details.Remediation = "Inspect the component render path named in the diagnostic and replace panic-based control flow with guarded branches, fallback UI, or an error boundary around the failing subtree."
	case strings.Contains(lower, "uncaught event panic in "):
		details.Code = "GWC-RUNTIME-PANIC-EVENT"
		details.Docs = actionableErrorsDoc + "#gwc-runtime-panic-event"
		details.Remediation = "Inspect the event handler named in the diagnostic, remove panic-based control flow, and return explicit errors or guarded updates instead of crashing the runtime callback."
	case strings.Contains(lower, "uncaught effect panic in "):
		details.Code = "GWC-RUNTIME-PANIC-EFFECT"
		details.Docs = actionableErrorsDoc + "#gwc-runtime-panic-effect"
		details.Remediation = "Inspect the effect body for the named component and move failure-prone work behind validation, explicit error handling, or an error boundary-friendly fallback path."
	case strings.Contains(lower, "uncaught cleanup panic in "):
		details.Code = "GWC-RUNTIME-PANIC-CLEANUP"
		details.Docs = actionableErrorsDoc + "#gwc-runtime-panic-cleanup"
		details.Remediation = "Inspect the cleanup function for the named component and make teardown idempotent so unmount or dependency changes do not panic mid-cleanup."
	case strings.Contains(lower, "uncaught loader panic in "):
		details.Code = "GWC-RUNTIME-PANIC-LOADER"
		details.Docs = actionableErrorsDoc + "#gwc-runtime-panic-loader"
		details.Remediation = "Inspect the route loader or async data callback named in the diagnostic, remove panic-based control flow, and return explicit errors so route error UI can recover without a fatal exit."
	case strings.Contains(lower, "uncaught hydration panic in "):
		details.Code = "GWC-RUNTIME-PANIC-HYDRATION"
		details.Docs = actionableErrorsDoc + "#gwc-runtime-panic-hydration"
		details.Remediation = "Inspect the first client render inputs and server markup, then fix the mismatch or hydration-time panic before relying on strict resume paths."
	case strings.Contains(lower, "uncaught startup panic in "):
		details.Code = "GWC-RUNTIME-PANIC-STARTUP"
		details.Docs = actionableErrorsDoc + "#gwc-runtime-panic-startup"
		details.Remediation = "Verify startup selectors, bootstrap inputs, and runtime initialization so mount-time failures do not terminate before the app can render."
	case strings.Contains(lower, "uncaught deferred panic in "):
		details.Code = "GWC-RUNTIME-PANIC-DEFERRED"
		details.Docs = actionableErrorsDoc + "#gwc-runtime-panic-deferred"
		details.Remediation = "Inspect deferred callbacks, transition work, and scheduled runtime continuations so delayed work returns explicit errors instead of panicking."
	case strings.Contains(lower, "uncaught ssr panic in "):
		details.Code = "GWC-RUNTIME-PANIC-SSR"
		details.Docs = actionableErrorsDoc + "#gwc-runtime-panic-ssr"
		details.Remediation = "Inspect the server render path and bootstrap generation code first, then surface the returned error instead of letting request-time rendering fail invisibly."
	case strings.Contains(lower, "called outside component context"):
		details.Code = "GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT"
		details.Docs = actionableErrorsDoc + "#gwc-runtime-hook-outside-component"
		details.Remediation = "Call framework hooks only while rendering a component through ui.CreateElement(...). Move the hook call out of package init code, route factories, and ordinary helpers that are not rendering components."
	case strings.Contains(lower, "gousefunc requires a function"):
		details.Code = "GWC-RUNTIME-HOOK-FUNC-TYPE"
		details.Docs = actionableErrorsDoc + "#gwc-runtime-hook-func-type"
		details.Remediation = "Pass a real function to GoUseFunc and move any non-callable config or data values outside the event-hook wrapper."
	case strings.Contains(lower, "gousefunc dom adapter is nil"):
		details.Code = "GWC-RUNTIME-DOM-ADAPTER-NIL"
		details.Docs = actionableErrorsDoc + "#gwc-runtime-dom-adapter-nil"
		details.Remediation = "Initialize the runtime with a DOM adapter before rendering components that call GoUseFunc or wire event handlers."
	case strings.Contains(lower, "runtime atom registry not initialized"):
		details.Code = "GWC-RUNTIME-ATOM-REGISTRY-NIL"
		details.Docs = actionableErrorsDoc + "#gwc-runtime-atom-registry-nil"
		details.Remediation = "Create or initialize the runtime before calling GoUseAtom so the shared atom registry exists for subscriptions and updates."
	case strings.Contains(lower, "gouseatom accessor cache type mismatch"):
		details.Code = "GWC-RUNTIME-ATOM-ACCESSOR-MISMATCH"
		details.Docs = actionableErrorsDoc + "#gwc-runtime-atom-accessor-mismatch"
		details.Remediation = "Do not reuse one hook slot for different atom value types. Keep GoUseAtom call order stable and preserve a single value shape per atom hook position."
	case strings.Contains(lower, "called with nil context descriptor"):
		details.Code = "GWC-UI-CONTEXT-NIL"
		details.Docs = actionableErrorsDoc + "#gwc-ui-context-nil"
		details.Remediation = "Pass the descriptor returned by ui.CreateContext(...) and avoid nil placeholder contexts."
	case strings.Contains(lower, "ui.createelement requires a component function or ui.node") ||
		strings.Contains(lower, "ui.createelement components may accept at most one props argument") ||
		strings.Contains(lower, "ui.createelement components must return ui.node"):
		details.Code = "GWC-UI-CREATE-ELEMENT-TYPE"
		details.Docs = actionableErrorsDoc + "#gwc-ui-create-element-type"
		details.Remediation = "Pass a component function or ui.Node to ui.CreateElement, keep component signatures to zero or one props argument, and return exactly one ui.Node tree."
	case strings.Contains(lower, "is not available on non-js/wasm builds in the current ssr slice"):
		details.Code = "GWC-UI-UNSUPPORTED-ON-SERVER"
		details.Docs = actionableErrorsDoc + "#gwc-ui-unsupported-on-server"
		details.Remediation = "Use the SSR-safe alternative documented for this API, or call the browser-only API only from js/wasm builds after the client runtime is active."
	case strings.Contains(lower, "route component cannot be nil"):
		details.Code = "GWC-ROUTER-COMPONENT-NIL"
		details.Docs = actionableErrorsDoc + "#gwc-router-component-nil"
		details.Remediation = "Register a concrete component function or static node for the route instead of leaving a nil placeholder."
	case strings.Contains(lower, "unsupported route component type"):
		details.Code = "GWC-ROUTER-COMPONENT-TYPE"
		details.Docs = actionableErrorsDoc + "#gwc-router-component-type"
		details.Remediation = "Register a component function, ui.Node, or route-compatible element producer instead of a raw config or data value."
	case strings.Contains(lower, "route component must return exactly one element"):
		details.Code = "GWC-ROUTER-COMPONENT-ARITY"
		details.Docs = actionableErrorsDoc + "#gwc-router-component-arity"
		details.Remediation = "Return exactly one element tree from the route component. Wrap siblings in ui.Fragment(...) when needed."
	case strings.Contains(lower, "replacing existing catch-all route registration") ||
		strings.Contains(lower, "replacing existing pattern route registration") ||
		strings.Contains(lower, "replacing existing route registration"):
		details.Code = "GWC-ROUTER-DUPLICATE-ROUTE"
		details.Docs = actionableErrorsDoc + "#gwc-router-duplicate-route"
		details.Remediation = "Remove duplicate route registrations or confirm that last-write-wins replacement is intentional."
		details.Recoverable = true
	case strings.Contains(lower, "redirect loop"):
		details.Code = "GWC-ROUTER-REDIRECT-LOOP"
		details.Docs = actionableErrorsDoc + "#gwc-router-redirect-loop"
		details.Remediation = "Compare redirect source and target normalization and ensure guards or default routes do not bounce between the same paths."
		details.Recoverable = severity != DiagnosticError
	case strings.Contains(lower, "route loader failed"):
		details.Code = "GWC-ROUTER-LOADER-FAILED"
		details.Docs = actionableErrorsDoc + "#gwc-router-loader-failed"
		details.Remediation = "Inspect the loader inputs, network failure, and route-local error UI. Supply an Options.Error fallback when the route depends on remote data."
	case strings.Contains(lower, "hydration text mismatch"):
		details.Code = "GWC-HYDRATION-TEXT-MISMATCH"
		details.Docs = actionableErrorsDoc + "#gwc-hydration-text-mismatch"
		details.Remediation = "Make the first client render deterministic with the server HTML. Recheck IDs, params, query-derived branches, time-dependent strings, and bootstrap data reuse."
		details.Recoverable = severity != DiagnosticError
	case strings.Contains(lower, "hydration attribute mismatch"):
		details.Code = "GWC-HYDRATION-ATTRIBUTE-MISMATCH"
		details.Docs = actionableErrorsDoc + "#gwc-hydration-attribute-mismatch"
		details.Remediation = "Compare server-rendered attributes with the first client render inputs and verify metadata, params, query state, and SSR bootstrap reuse."
		details.Recoverable = severity != DiagnosticError
	case strings.Contains(lower, "hydration discarded unexpected dom nodes"):
		details.Code = "GWC-HYDRATION-DISCARDED-NODES"
		details.Docs = actionableErrorsDoc + "#gwc-hydration-discarded-nodes"
		details.Remediation = "Ensure the server document does not inject extra nodes into the hydrated subtree and keep client-only placeholders outside the hydrated tree."
		details.Recoverable = severity != DiagnosticError
	case strings.Contains(lower, "hydration fell back to client rendering"):
		details.Code = "GWC-HYDRATION-FALLBACK"
		details.Docs = actionableErrorsDoc + "#gwc-hydration-fallback"
		details.Remediation = "Fix the underlying server-client mismatch instead of relying on fallback rendering. In strict hydration flows, treat the output as correctness-threatening until the mismatch is resolved."
		details.Recoverable = severity != DiagnosticError
	case strings.Contains(lower, "target container selector was not found"):
		details.Code = "GWC-RUNTIME-CONTAINER-NOT-FOUND"
		details.Docs = "TROUBLESHOOTING.md#broken-example-serving"
		details.Remediation = "Verify the target selector exists before calling ui.Render(...) or ui.Hydrate(...), and confirm the page mounted the expected root container."
	}

	_ = source
	return details
}

func logSeverity(level LogLevel) DiagnosticSeverity {
	switch level {
	case LogError:
		return DiagnosticError
	case LogWarn:
		return DiagnosticWarning
	default:
		return DiagnosticInfo
	}
}

func actionableHookUsagePanic(name string) string {
	trimmed := strings.TrimSpace(name)
	return ActionableFrameworkPanic(ActionablePanicOptions{
		Source:  "runtime",
		Subject: trimmed,
		Message: fmt.Sprintf("%s called outside component context", trimmed),
		Path:    trimmed,
	})
}

func actionableContextDescriptorNilPanic(name string) string {
	trimmed := strings.TrimSpace(name)
	return ActionableFrameworkPanic(ActionablePanicOptions{
		Source:  "runtime",
		Subject: trimmed,
		Message: fmt.Sprintf("%s called with nil context descriptor", trimmed),
		Path:    trimmed,
	})
}

func actionableGoUseFuncTypePanic() string {
	return ActionableFrameworkPanic(ActionablePanicOptions{
		Source:  "runtime",
		Subject: "GoUseFunc",
		Message: "GoUseFunc requires a function",
		Path:    "GoUseFunc",
	})
}

func actionableGoUseFuncDOMAdapterPanic() string {
	return ActionableFrameworkPanic(ActionablePanicOptions{
		Source:  "runtime",
		Subject: "GoUseFunc",
		Message: "GoUseFunc dom adapter is nil",
		Path:    "GoUseFunc",
	})
}

func actionableGoUseAtomRegistryPanic() string {
	return ActionableFrameworkPanic(ActionablePanicOptions{
		Source:  "runtime",
		Subject: "GoUseAtom",
		Message: "Runtime atom registry not initialized",
		Path:    "GoUseAtom",
	})
}

func actionableGoUseAtomAccessorPanic() string {
	return ActionableFrameworkPanic(ActionablePanicOptions{
		Source:  "runtime",
		Subject: "GoUseAtom",
		Message: "GoUseAtom accessor cache type mismatch",
		Path:    "GoUseAtom",
	})
}

func panicDiagnosticCode(phase boundaryPhase) string {
	switch phase {
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
	default:
		return "GWC-RUNTIME-PANIC"
	}
}

func panicDiagnosticDocs(phase boundaryPhase) string {
	switch phase {
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
	default:
		return actionableErrorsDoc
	}
}

func panicDiagnosticRemediation(phase boundaryPhase) string {
	code := panicDiagnosticCode(phase)
	if strings.TrimSpace(code) == "" {
		code = "GWC-RUNTIME-PANIC"
	}
	prefix := fmt.Sprintf("Match code %s in automation; ", code)
	switch phase {
	case PanicPhaseRender:
		return prefix + "inspect the component render path named by where/path first; replace panic-based control flow with guarded branches, fallback UI, or an ErrorBoundary when local recovery is expected."
	case PanicPhaseEvent:
		return prefix + "inspect the event handler named by where/path first; return explicit errors or guarded updates instead of panicking from runtime callbacks."
	case PanicPhaseEffect:
		return prefix + "inspect the effect body named by where/path first; move failure-prone work behind validation, explicit errors, or a recoverable fallback path."
	case PanicPhaseCleanup:
		return prefix + "inspect the cleanup path named by where/path first; make teardown idempotent and avoid panicking during unmount or dependency changes."
	case PanicPhaseLoader:
		return prefix + "inspect the loader or async data path named by where/path first; return explicit errors so route error UI can recover instead of crashing route resolution."
	case PanicPhaseHydration:
		return prefix + "compare server markup with the first client render at where/path first; fix the mismatch or hydration panic before relying on strict resume."
	case PanicPhaseStartup:
		return prefix + "verify the startup target and bootstrap inputs named by where/path first; ensure mount selectors and initialization state exist before rendering."
	case PanicPhaseDeferred:
		return prefix + "inspect the deferred callback or transition work named by where/path first; replace panic-based control flow with explicit errors or guarded branches."
	case PanicPhaseSSR:
		return prefix + "inspect the server render path named by where/path first; surface the returned error to the request handler instead of treating it as a transport-level panic."
	default:
		return prefix + "inspect the where/path fields first, then replace panic-based control flow with guarded branches, explicit errors, fallback UI, or a boundary when recovery is expected."
	}
}

func panicSummary(recovered interface{}) string {
	err := normalizeBoundaryError(recovered)
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "panic without message"
	}
	return message
}

func panicSubject(fiber *Fiber) string {
	stack := diagnosticComponentStack(fiber)
	if len(stack) > 0 {
		return stack[len(stack)-1]
	}
	return "application"
}

func panicDiagnosticMessage(fiber *Fiber, phase boundaryPhase, recovered interface{}) string {
	return fmt.Sprintf("uncaught %s panic in %s: %s", phase, panicSubject(fiber), panicSummary(recovered))
}

func reportUnhandledPanic(fiber *Fiber, phase boundaryPhase, recovered interface{}) string {
	return ReportUnhandledPanicContext(
		"runtime",
		phase,
		panicSubject(fiber),
		diagnosticPathForFiber(fiber),
		diagnosticComponentStack(fiber),
		recovered,
	)
}

func markUnhandledPanic(fiber *Fiber, phase boundaryPhase, recovered interface{}) interface{} {
	return markUnhandledPanicContext(
		"runtime",
		phase,
		panicSubject(fiber),
		diagnosticPathForFiber(fiber),
		diagnosticComponentStack(fiber),
		recovered,
	)
}

func panicFinalUnhandledPanic(fiber *Fiber, phase boundaryPhase, recovered interface{}) {
	panicFinalUnhandledPanicContext(
		"runtime",
		phase,
		panicSubject(fiber),
		diagnosticPathForFiber(fiber),
		diagnosticComponentStack(fiber),
		recovered,
	)
}
