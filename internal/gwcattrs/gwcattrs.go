// Package gwcattrs holds the canonical telemetry attribute key strings shared by
// the ui, logging and telemetry packages. It is stdlib-only and tag-free so it
// compiles on both native and js/wasm builds without pulling in OpenTelemetry.
package gwcattrs

// Error and exception attribute keys.
const (
	// KeyErrorCode is the stable GWC diagnostic code of an error.
	KeyErrorCode = "gwc.error.code"
	// KeyComponent is the component name an event belongs to.
	KeyComponent = "gwc.component"
	// KeyComponentStack is the component ancestry at the time of an event.
	KeyComponentStack = "gwc.component_stack"
	// KeyRenderMode is the render mode (client, ssr, hydrate).
	KeyRenderMode = "gwc.render.mode"
	// KeyLane is the scheduler lane an update ran in.
	KeyLane = "gwc.lane"
	// KeyHydrationStrict reports whether strict hydration was enabled.
	KeyHydrationStrict = "gwc.hydration.strict"
	// KeyExceptionType is the OpenTelemetry exception type key.
	KeyExceptionType = "exception.type"
	// KeyExceptionMessage is the OpenTelemetry exception message key.
	KeyExceptionMessage = "exception.message"
	// KeyExceptionStacktrace is the OpenTelemetry exception stacktrace key.
	KeyExceptionStacktrace = "exception.stacktrace"
)

// SSR observation attribute keys, identical to those emitted by
// ui.GetSSRObservationAttributes.
const (
	KeySSREventName                   = "gwc.ssr.event.name"
	KeySSREventDomain                 = "gwc.ssr.event.domain"
	KeySSREventPhase                  = "gwc.ssr.event.phase"
	KeySSRCorrelationID               = "gwc.ssr.correlation_id"
	KeySSREventTimestamp              = "gwc.ssr.event.timestamp"
	KeySSRRenderDurationMs            = "gwc.ssr.render.duration_ms"
	KeySSRRenderDurationNs            = "gwc.ssr.render.duration_ns"
	KeySSRBootstrapFormat             = "gwc.ssr.bootstrap.format"
	KeySSRBootstrapPayloadBytes       = "gwc.ssr.bootstrap.payload_bytes"
	KeySSRBootstrapScriptBytes        = "gwc.ssr.bootstrap.script_bytes"
	KeySSRHydrationDurationMs         = "gwc.ssr.hydration.duration_ms"
	KeySSRHydrationDurationNs         = "gwc.ssr.hydration.duration_ns"
	KeySSRHydrationExistingDOMNodes   = "gwc.ssr.hydration.existing_dom_nodes"
	KeySSRHydrationFallbackCount      = "gwc.ssr.hydration.fallback_count"
	KeySSRHydrationMismatchCount      = "gwc.ssr.hydration.mismatch_count"
	KeySSRHydrationDiscardedNodeCount = "gwc.ssr.hydration.discarded_node_count"
	KeySSRHydrationStrict             = "gwc.ssr.hydration.strict"
	KeySSRHydrationFailed             = "gwc.ssr.hydration.failed"
	KeySSRHydrationFailure            = "gwc.ssr.hydration.failure"
)
