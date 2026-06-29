package runtime2_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

type parseSpecProps struct {
	Title  string
	Counts []int
}

type parseDOMInteropProps struct {
	DOMNode string
}

type parseEventClosureProps struct {
	OnClick func()
}

// TestValidateParallelRegionSpecAcceptsMinimalValidSpec verifies the smallest valid spec passes.
func TestValidateParallelRegionSpecAcceptsMinimalValidSpec(parseT *testing.T) {
	parseSpec := runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
	}
	if parseErr := runtime2.ValidateParallelRegionSpec(parseSpec); parseErr != nil {
		parseT.Fatalf("ValidateParallelRegionSpec returned error: %v", parseErr)
	}
}

// TestValidateParallelRegionSpecRejectsMissingRendererID verifies the renderer ID is required.
func TestValidateParallelRegionSpecRejectsMissingRendererID(parseT *testing.T) {
	parseSpec := runtime2.ParallelRegionSpec{RegionInstanceID: runtime2.RegionInstanceID("region-1")}
	if parseErr := runtime2.ValidateParallelRegionSpec(parseSpec); parseErr == nil {
		parseT.Fatal("expected missing renderer ID to fail")
	}
}

// TestValidateParallelRegionSpecRejectsMissingRegionInstanceID verifies the region instance ID is required.
func TestValidateParallelRegionSpecRejectsMissingRegionInstanceID(parseT *testing.T) {
	parseSpec := runtime2.ParallelRegionSpec{RendererID: runtime2.RendererID("dashboard.hot-panel")}
	if parseErr := runtime2.ValidateParallelRegionSpec(parseSpec); parseErr == nil {
		parseT.Fatal("expected missing region instance ID to fail")
	}
}

// TestValidateSerializablePropsAcceptsNilProps verifies nil props are allowed.
func TestValidateSerializablePropsAcceptsNilProps(parseT *testing.T) {
	if parseErr := runtime2.ValidateSerializableProps(nil); parseErr != nil {
		parseT.Fatalf("ValidateSerializableProps(nil) returned error: %v", parseErr)
	}
}

// TestValidateSerializablePropsAcceptsPrimitiveAndNestedValues verifies supported prop graphs pass validation.
func TestValidateSerializablePropsAcceptsPrimitiveAndNestedValues(parseT *testing.T) {
	parseProps := map[string]any{
		"title": "Orders",
		"items": []map[string]any{
			{"count": 3},
		},
		"meta": parseSpecProps{
			Title:  "Hot Panel",
			Counts: []int{1, 2, 3},
		},
	}
	if parseErr := runtime2.ValidateSerializableProps(parseProps); parseErr != nil {
		parseT.Fatalf("ValidateSerializableProps returned error: %v", parseErr)
	}
}

// TestValidateSerializablePropsAcceptsTypedScalarContainers verifies typed scalar slices and maps stay on the fast serializable path.
func TestValidateSerializablePropsAcceptsTypedScalarContainers(parseT *testing.T) {
	parseProps := map[string]any{
		"labels": []string{"a", "b", "c"},
		"counts": []int{1, 2, 3},
		"meta":   map[string]string{"title": "Orders", "state": "open"},
	}
	if parseErr := runtime2.ValidateSerializableProps(parseProps); parseErr != nil {
		parseT.Fatalf("ValidateSerializableProps returned error: %v", parseErr)
	}
}

// TestValidateSerializablePropsRejectsTypedScalarMapRefMarker verifies typed scalar maps still reject ref-like key names.
func TestValidateSerializablePropsRejectsTypedScalarMapRefMarker(parseT *testing.T) {
	parseErr := runtime2.ValidateSerializableProps(map[string]string{
		"ref": "node-1",
	})
	if parseErr == nil {
		parseT.Fatal("expected typed scalar map ref marker to fail")
	}
}

// TestValidateSerializablePropsRejectsFunctionPath verifies unsupported function props fail with a path.
func TestValidateSerializablePropsRejectsFunctionPath(parseT *testing.T) {
	parseProps := map[string]any{
		"nested": map[string]any{
			"callback": func() {},
		},
	}
	parseErr := runtime2.ValidateSerializableProps(parseProps)
	if parseErr == nil {
		parseT.Fatal("expected function props to fail")
	}
	if !strings.Contains(parseErr.Error(), "props.nested.callback") {
		parseT.Fatalf("expected function-prop error to include offending path, got %v", parseErr)
	}
}

// TestValidateSerializablePropsRejectsUnsupportedInterfaceValue verifies unsupported interface payloads fail with a path.
func TestValidateSerializablePropsRejectsUnsupportedInterfaceValue(parseT *testing.T) {
	parseProps := map[string]any{
		"nested": any(make(chan int)),
	}
	parseErr := runtime2.ValidateSerializableProps(parseProps)
	if parseErr == nil {
		parseT.Fatal("expected unsupported interface payload to fail")
	}
	if !strings.Contains(parseErr.Error(), "props.nested") {
		parseT.Fatalf("expected unsupported interface error to include offending path, got %v", parseErr)
	}
}

// TestValidateSerializablePropsRejectsUnsupportedMapKeyKind verifies bad map keys fail before transport.
func TestValidateSerializablePropsRejectsUnsupportedMapKeyKind(parseT *testing.T) {
	parseProps := map[int]string{1: "bad"}
	parseErr := runtime2.ValidateSerializableProps(parseProps)
	if parseErr == nil {
		parseT.Fatal("expected unsupported map key kind to fail")
	}
	if !strings.Contains(parseErr.Error(), "unsupported map key kind") {
		parseT.Fatalf("expected map-key error, got %v", parseErr)
	}
}

// TestNormalizeSourceIDsAcceptsEmptySourceSet verifies empty source lists normalize consistently.
func TestNormalizeSourceIDsAcceptsEmptySourceSet(parseT *testing.T) {
	parseSourceIDs, parseErr := runtime2.NormalizeSourceIDs(nil)
	if parseErr != nil {
		parseT.Fatalf("NormalizeSourceIDs returned error: %v", parseErr)
	}
	if len(parseSourceIDs) != 0 {
		parseT.Fatalf("expected empty source IDs, got %+v", parseSourceIDs)
	}
}

// TestNormalizeSourceIDsDeduplicatesAndSorts verifies duplicate source IDs normalize consistently.
func TestNormalizeSourceIDsDeduplicatesAndSorts(parseT *testing.T) {
	parseSourceIDs, parseErr := runtime2.NormalizeSourceIDs([]string{"b", "a", "b"})
	if parseErr != nil {
		parseT.Fatalf("NormalizeSourceIDs returned error: %v", parseErr)
	}
	if len(parseSourceIDs) != 2 || parseSourceIDs[0] != "a" || parseSourceIDs[1] != "b" {
		parseT.Fatalf("expected normalized sorted source IDs, got %+v", parseSourceIDs)
	}
}

// TestNormalizeSourceIDsRejectsInvalidFormat verifies invalid source IDs fail clearly.
func TestNormalizeSourceIDsRejectsInvalidFormat(parseT *testing.T) {
	if _, parseErr := runtime2.NormalizeSourceIDs([]string{"bad source"}); parseErr == nil {
		parseT.Fatal("expected invalid source ID format to fail")
	}
}

// TestNormalizeParallelRegionSpecCanonicalizesSources verifies the spec normalizer keeps source ordering stable.
func TestNormalizeParallelRegionSpecCanonicalizesSources(parseT *testing.T) {
	parseSpec, parseErr := runtime2.NormalizeParallelRegionSpec(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		SourceIDs:        []string{"status", "count", "status"},
	})
	if parseErr != nil {
		parseT.Fatalf("NormalizeParallelRegionSpec returned error: %v", parseErr)
	}
	if len(parseSpec.SourceIDs) != 2 || parseSpec.SourceIDs[0] != "count" || parseSpec.SourceIDs[1] != "status" {
		parseT.Fatalf("expected canonical source ordering, got %+v", parseSpec.SourceIDs)
	}
}

// TestValidateParallelRegionSpecRejectsRefLikeProps verifies ref-like props are rejected in first-slice worker-renderable specs.
func TestValidateParallelRegionSpecRejectsRefLikeProps(parseT *testing.T) {
	parseSpec := runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Props: map[string]any{
			"ref": "node-1",
		},
	}
	if parseErr := runtime2.ValidateParallelRegionSpec(parseSpec); parseErr == nil {
		parseT.Fatal("expected ref-like props to fail spec validation")
	}
}

// TestValidateSerializablePropsRejectsDirectDOMInteropMapKey verifies direct DOM interop markers are rejected in input map keys.
func TestValidateSerializablePropsRejectsDirectDOMInteropMapKey(parseT *testing.T) {
	parseProps := map[string]any{
		"dom_ref": "node-1",
	}
	parseErr := runtime2.ValidateSerializableProps(parseProps)
	if parseErr == nil {
		parseT.Fatal("expected direct DOM interop markers to fail")
	}
	if !strings.Contains(parseErr.Error(), "direct DOM interop marker") {
		parseT.Fatalf("expected direct DOM interop error details, got %v", parseErr)
	}
}

// TestValidateSerializablePropsRejectsDirectDOMInteropStructField verifies direct DOM interop markers are rejected in exported struct fields.
func TestValidateSerializablePropsRejectsDirectDOMInteropStructField(parseT *testing.T) {
	parseErr := runtime2.ValidateSerializableProps(parseDOMInteropProps{
		DOMNode: "node-1",
	})
	if parseErr == nil {
		parseT.Fatal("expected direct DOM interop struct marker to fail")
	}
	if !strings.Contains(parseErr.Error(), "direct DOM interop marker") {
		parseT.Fatalf("expected direct DOM interop error details, got %v", parseErr)
	}
}

// TestValidateSerializablePropsRejectsEventClosureMapProp verifies event-closure map props are rejected with explicit guidance.
func TestValidateSerializablePropsRejectsEventClosureMapProp(parseT *testing.T) {
	parseProps := map[string]any{
		"onClick": func() {},
	}
	parseErr := runtime2.ValidateSerializableProps(parseProps)
	if parseErr == nil {
		parseT.Fatal("expected event-closure map prop to fail")
	}
	if !strings.Contains(parseErr.Error(), "event-closure prop") {
		parseT.Fatalf("expected event-closure error details, got %v", parseErr)
	}
}

// TestValidateSerializablePropsRejectsEventClosureStructField verifies event-closure struct props are rejected with explicit guidance.
func TestValidateSerializablePropsRejectsEventClosureStructField(parseT *testing.T) {
	parseErr := runtime2.ValidateSerializableProps(parseEventClosureProps{
		OnClick: func() {},
	})
	if parseErr == nil {
		parseT.Fatal("expected event-closure struct field to fail")
	}
	if !strings.Contains(parseErr.Error(), "event-closure prop") {
		parseT.Fatalf("expected event-closure error details, got %v", parseErr)
	}
}
