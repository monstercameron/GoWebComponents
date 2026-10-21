package ui

import "fmt"

// GetSSRObservationAttributes maps one SSRObservation to OTel-compatible span attribute
// key-value pairs.
//
// The returned map uses stable "gwc.*" attribute keys following OpenTelemetry naming
// conventions. Pass them to your OTel SDK with SetAttributes or equivalent:
//
//	attrs := ui.GetSSRObservationAttributes(event)
//	for k, v := range attrs {
//	    span.SetAttributes(attribute.String(k, v))
//	}
//
// Keys are guaranteed stable across patch releases once the observability surface
// is marked stable.
//
// This mapping is pure Go and lives in a tag-free file so the telemetry package
// compiles under js/wasm as well as on the host.
func GetSSRObservationAttributes(parseObservation SSRObservation) map[string]string {
	parseAttrs := map[string]string{
		"gwc.ssr.event.name":      parseObservation.Name,
		"gwc.ssr.event.domain":    parseObservation.Domain,
		"gwc.ssr.event.phase":     parseObservation.Phase,
		"gwc.ssr.correlation_id":  parseObservation.CorrelationID,
		"gwc.ssr.event.timestamp": parseObservation.Timestamp.UTC().Format("2006-01-02T15:04:05.999999999Z"),
	}
	if parseR := parseObservation.Render; parseR != nil {
		parseAttrs["gwc.ssr.render.duration_ms"] = fmt.Sprintf("%.3f", float64(parseR.DurationNs)/1e6)
		parseAttrs["gwc.ssr.render.duration_ns"] = fmt.Sprintf("%d", parseR.DurationNs)
	}
	if parseB := parseObservation.Bootstrap; parseB != nil {
		parseAttrs["gwc.ssr.bootstrap.format"] = parseB.Format
		parseAttrs["gwc.ssr.bootstrap.payload_bytes"] = fmt.Sprintf("%d", parseB.PayloadBytes)
		parseAttrs["gwc.ssr.bootstrap.script_bytes"] = fmt.Sprintf("%d", parseB.ScriptBytes)
	}
	if parseH := parseObservation.Hydration; parseH != nil {
		parseAttrs["gwc.ssr.hydration.duration_ms"] = fmt.Sprintf("%.3f", float64(parseH.DurationNs)/1e6)
		parseAttrs["gwc.ssr.hydration.duration_ns"] = fmt.Sprintf("%d", parseH.DurationNs)
		parseAttrs["gwc.ssr.hydration.existing_dom_nodes"] = fmt.Sprintf("%d", parseH.ExistingDOMNodeCount)
		parseAttrs["gwc.ssr.hydration.fallback_count"] = fmt.Sprintf("%d", parseH.FallbackCount)
		parseAttrs["gwc.ssr.hydration.mismatch_count"] = fmt.Sprintf("%d", parseH.MismatchCount)
		parseAttrs["gwc.ssr.hydration.discarded_node_count"] = fmt.Sprintf("%d", parseH.DiscardedNodeCount)
		parseAttrs["gwc.ssr.hydration.strict"] = fmt.Sprintf("%t", parseH.Strict)
		parseAttrs["gwc.ssr.hydration.failed"] = fmt.Sprintf("%t", parseH.Failed)
		if parseH.Failure != "" {
			parseAttrs["gwc.ssr.hydration.failure"] = parseH.Failure
		}
	}
	return parseAttrs
}
