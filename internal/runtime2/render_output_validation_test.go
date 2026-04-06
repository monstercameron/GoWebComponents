package runtime2

import (
	"strings"
	"testing"
)

// TestValidateWorkerRenderableRenderOutputAcceptsDisplayOnlyShape verifies display-only render output passes validation.
func TestValidateWorkerRenderableRenderOutputAcceptsDisplayOnlyShape(parseT *testing.T) {
	parseRenderOutput := map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"children": []any{
			map[string]any{
				"kind": "text",
				"text": "hello",
			},
		},
	}
	if parseErr := ValidateWorkerRenderableRenderOutput(parseRenderOutput); parseErr != nil {
		parseT.Fatalf("ValidateWorkerRenderableRenderOutput(display-only) returned error: %v", parseErr)
	}
}

// TestValidateWorkerRenderableRenderOutputRejectsPortalMarkerKey verifies portal-like keys fail output validation.
func TestValidateWorkerRenderableRenderOutputRejectsPortalMarkerKey(parseT *testing.T) {
	parseRenderOutput := map[string]any{
		"portal": map[string]any{
			"target": "modal-root",
		},
	}
	parseErr := ValidateWorkerRenderableRenderOutput(parseRenderOutput)
	if parseErr == nil {
		parseT.Fatal("expected portal marker key to fail render-output validation")
	}
	if !strings.Contains(parseErr.Error(), "unsupported portal marker") {
		parseT.Fatalf("expected portal-marker error details, got %v", parseErr)
	}
}

// TestValidateWorkerRenderableRenderOutputRejectsPortalKind verifies portal node-kind values fail output validation.
func TestValidateWorkerRenderableRenderOutputRejectsPortalKind(parseT *testing.T) {
	parseRenderOutput := map[string]any{
		"kind": "portal",
	}
	parseErr := ValidateWorkerRenderableRenderOutput(parseRenderOutput)
	if parseErr == nil {
		parseT.Fatal("expected portal node kind to fail render-output validation")
	}
	if !strings.Contains(parseErr.Error(), "unsupported portal node kind") {
		parseT.Fatalf("expected portal-kind error details, got %v", parseErr)
	}
}

// TestValidateWorkerRenderableRenderOutputRejectsDirectDOMInteropMarkerKey verifies direct DOM interop keys fail output validation.
func TestValidateWorkerRenderableRenderOutputRejectsDirectDOMInteropMarkerKey(parseT *testing.T) {
	parseRenderOutput := map[string]any{
		"dom_node": "node-1",
	}
	parseErr := ValidateWorkerRenderableRenderOutput(parseRenderOutput)
	if parseErr == nil {
		parseT.Fatal("expected direct DOM interop marker key to fail render-output validation")
	}
	if !strings.Contains(parseErr.Error(), "direct DOM interop marker") {
		parseT.Fatalf("expected direct DOM marker error details, got %v", parseErr)
	}
}

// TestValidateWorkerRenderableRenderOutputRejectsDirectDOMInteropKind verifies direct DOM interop kind values fail output validation.
func TestValidateWorkerRenderableRenderOutputRejectsDirectDOMInteropKind(parseT *testing.T) {
	parseRenderOutput := map[string]any{
		"kind": "dom-interop",
	}
	parseErr := ValidateWorkerRenderableRenderOutput(parseRenderOutput)
	if parseErr == nil {
		parseT.Fatal("expected direct DOM interop kind to fail render-output validation")
	}
	if !strings.Contains(parseErr.Error(), "direct DOM interop kind") {
		parseT.Fatalf("expected direct DOM kind error details, got %v", parseErr)
	}
}

// TestValidateWorkerRenderableRenderOutputTraversesNilPointersSlicesStructsAndMaps verifies the recursive validator accepts plain data, skips nil references, and rejects non-string-key portal and DOM interop markers.
func TestValidateWorkerRenderableRenderOutputTraversesNilPointersSlicesStructsAndMaps(parseT *testing.T) {
	type parseNestedRenderOutput struct {
		Text string
		Node *struct {
			Kind string
		}
		Items []any
	}

	if parseErr := ValidateWorkerRenderableRenderOutput(nil); parseErr != nil {
		parseT.Fatalf("ValidateWorkerRenderableRenderOutput(nil) returned error: %v", parseErr)
	}
	if parseErr := ValidateWorkerRenderableRenderOutput(17); parseErr != nil {
		parseT.Fatalf("ValidateWorkerRenderableRenderOutput(primitive) returned error: %v", parseErr)
	}

	parseNested := parseNestedRenderOutput{
		Text: "ok",
		Node: &struct {
			Kind string
		}{Kind: "text"},
		Items: []any{
			nil,
			"child",
			[]string{"grandchild"},
		},
	}
	if parseErr := ValidateWorkerRenderableRenderOutput(parseNested); parseErr != nil {
		parseT.Fatalf("ValidateWorkerRenderableRenderOutput(nested safe output) returned error: %v", parseErr)
	}

	if parseErr := ValidateWorkerRenderableRenderOutput([]map[string]any{{"kind": "text"}}); parseErr != nil {
		parseT.Fatalf("ValidateWorkerRenderableRenderOutput(slice of maps) returned error: %v", parseErr)
	}

	parsePortalMarker := map[int]any{
		1: map[string]any{"kind": "portal"},
	}
	if parseErr := ValidateWorkerRenderableRenderOutput(parsePortalMarker); parseErr == nil {
		parseT.Fatal("expected non-string-key portal marker to fail render-output validation")
	}

	parseDirectDOMInteropMarker := map[int]any{
		1: map[string]any{"kind": "dom-interop"},
	}
	if parseErr := ValidateWorkerRenderableRenderOutput(parseDirectDOMInteropMarker); parseErr == nil {
		parseT.Fatal("expected non-string-key direct DOM interop marker to fail render-output validation")
	}

	parsePortalStruct := struct {
		Portal string
	}{
		Portal: "modal-root",
	}
	if parseErr := ValidateWorkerRenderableRenderOutput(parsePortalStruct); parseErr == nil {
		parseT.Fatal("expected portal field name to fail render-output validation")
	}
}
