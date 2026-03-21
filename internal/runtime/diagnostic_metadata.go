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
	case strings.Contains(lower, "called outside component context"):
		details.Code = "GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT"
		details.Docs = actionableErrorsDoc + "#gwc-runtime-hook-outside-component"
		details.Remediation = "Call framework hooks only while rendering a component through ui.CreateElement(...). Move the hook call out of package init code, route factories, and ordinary helpers that are not rendering components."
	case strings.Contains(lower, "called with nil context descriptor"):
		details.Code = "GWC-UI-CONTEXT-NIL"
		details.Docs = actionableErrorsDoc + "#gwc-ui-context-nil"
		details.Remediation = "Pass the descriptor returned by ui.CreateContext(...) and avoid nil placeholder contexts."
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
	return fmt.Sprintf("%s called outside component context (GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT). Call hooks only while rendering a component through ui.CreateElement(...). Do not call hooks in package init code, route factories, or ordinary helpers. See %s#gwc-runtime-hook-outside-component.", strings.TrimSpace(name), actionableErrorsDoc)
}

func actionableContextDescriptorNilPanic(name string) string {
	return fmt.Sprintf("%s called with nil context descriptor (GWC-UI-CONTEXT-NIL). Pass the descriptor returned by ui.CreateContext(...) and avoid nil placeholder contexts. See %s#gwc-ui-context-nil.", strings.TrimSpace(name), actionableErrorsDoc)
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
	default:
		return actionableErrorsDoc
	}
}

func panicDiagnosticRemediation(phase boundaryPhase) string {
	switch phase {
	case boundaryPhaseRender:
		return "Inspect the component render path and replace panic-based control flow with guarded branches, fallback UI, or an error boundary around the failing subtree."
	case boundaryPhaseEvent:
		return "Inspect the event handler, remove panic-based control flow, and return explicit errors or guarded updates instead of crashing the callback."
	case boundaryPhaseEffect:
		return "Inspect the effect body and move failure-prone work behind validation, explicit error handling, or an error boundary-friendly fallback path."
	case boundaryPhaseCleanup:
		return "Inspect the cleanup function and make teardown idempotent so unmount or dependency changes do not panic mid-cleanup."
	default:
		return "Inspect the failing user code path and replace panic-based control flow with explicit error handling where possible."
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

func actionablePanicMessage(fiber *Fiber, phase boundaryPhase, recovered interface{}) string {
	subject := panicSubject(fiber)
	summary := panicSummary(recovered)
	code := panicDiagnosticCode(phase)
	docs := panicDiagnosticDocs(phase)
	path := strings.TrimSpace(diagnosticPathForFiber(fiber))
	if path == "" {
		path = subject
	}

	return fmt.Sprintf("%s\n[%s] uncaught %s panic in %s\nwhere: %s\nerror: %s\nnext: %s\ndocs: %s", summary, code, phase, subject, path, summary, panicDiagnosticRemediation(phase), docs)
}

func reportUnhandledPanic(fiber *Fiber, phase boundaryPhase, recovered interface{}) string {
	ReportDiagnosticWithContext(
		"runtime",
		DiagnosticError,
		panicDiagnosticMessage(fiber, phase, recovered),
		diagnosticPathForFiber(fiber),
		diagnosticComponentStack(fiber),
	)
	return actionablePanicMessage(fiber, phase, recovered)
}
