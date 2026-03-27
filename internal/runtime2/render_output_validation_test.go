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
