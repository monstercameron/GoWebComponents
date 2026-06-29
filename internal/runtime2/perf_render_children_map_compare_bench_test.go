package runtime2

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

var storeCanonicalChildrenBenchmarkSink []*canonicalRenderNode
var storeCanonicalMapBenchmarkSink map[string]any

// parseBuildCanonicalChildrenLegacyBenchmark preserves the previous reflect-first children normalization path for compare benchmarks.
func parseBuildCanonicalChildrenLegacyBenchmark(parseChildrenValue any) ([]*canonicalRenderNode, error) {
	if parseChildrenValue == nil {
		return nil, nil
	}
	parseReflectValue := reflect.ValueOf(parseChildrenValue)
	for parseReflectValue.IsValid() && (parseReflectValue.Kind() == reflect.Interface || parseReflectValue.Kind() == reflect.Pointer) {
		if parseReflectValue.IsNil() {
			return nil, nil
		}
		parseReflectValue = parseReflectValue.Elem()
	}
	if !parseReflectValue.IsValid() {
		return nil, nil
	}
	if parseReflectValue.Kind() != reflect.Slice && parseReflectValue.Kind() != reflect.Array {
		buildNode, parseNodeErr := parseBuildCanonicalRenderNode(parseChildrenValue)
		if parseNodeErr != nil {
			return nil, parseNodeErr
		}
		return []*canonicalRenderNode{buildNode}, nil
	}
	buildChildren := make([]*canonicalRenderNode, 0, parseReflectValue.Len())
	for parseIndex := 0; parseIndex < parseReflectValue.Len(); parseIndex++ {
		buildNode, parseNodeErr := parseBuildCanonicalRenderNode(parseReflectValue.Index(parseIndex).Interface())
		if parseNodeErr != nil {
			return nil, parseNodeErr
		}
		buildChildren = append(buildChildren, buildNode)
	}
	return buildChildren, nil
}

// parseBuildCanonicalMapValueLegacyBenchmark preserves the previous MapKeys-based map normalization behavior for compare benchmarks.
func parseBuildCanonicalMapValueLegacyBenchmark(parseMapValue reflect.Value) (map[string]any, error) {
	if !parseMapValue.IsValid() {
		return nil, fmt.Errorf("runtime2: map value is invalid")
	}
	for parseMapValue.IsValid() && (parseMapValue.Kind() == reflect.Interface || parseMapValue.Kind() == reflect.Pointer) {
		if parseMapValue.IsNil() {
			return nil, fmt.Errorf("runtime2: map value is nil")
		}
		parseMapValue = parseMapValue.Elem()
	}
	if !parseMapValue.IsValid() || parseMapValue.Kind() != reflect.Map {
		return nil, fmt.Errorf("runtime2: value %T is not a map", parseMapValue.Interface())
	}
	buildMapValue := make(map[string]any, parseMapValue.Len())
	for _, getMapKey := range parseMapValue.MapKeys() {
		buildKey := parseFormatCanonicalScalarValue(getMapKey.Interface())
		buildMapValue[buildKey] = parseMapValue.MapIndex(getMapKey).Interface()
	}
	return buildMapValue, nil
}

// buildRenderChildrenBenchmarkPayload builds one mixed children payload used by compare benchmarks.
func buildRenderChildrenBenchmarkPayload() []any {
	buildChildren := make([]any, 0, 128)
	for parseIndex := range 128 {
		buildChildren = append(buildChildren, map[string]any{
			"kind": "host-element",
			"tag":  "span",
			"key":  "k-" + strconv.Itoa(parseIndex),
			"children": []any{
				map[string]any{
					"kind": "text",
					"text": "t-" + strconv.Itoa(parseIndex),
				},
			},
		})
	}
	return buildChildren
}

// buildRenderMapNormalizationBenchmarkPayload builds one map payload used by map normalization compare benchmarks.
func buildRenderMapNormalizationBenchmarkPayload() map[string]any {
	buildMapValue := make(map[string]any, 256)
	for parseIndex := range 256 {
		buildMapValue["k-"+strconv.Itoa(parseIndex)] = "v-" + strconv.Itoa(parseIndex)
	}
	return buildMapValue
}

// BenchmarkParseBuildCanonicalChildrenCurrentVsLegacy compares current []any fast-path children normalization against the previous reflect-first path.
func BenchmarkParseBuildCanonicalChildrenCurrentVsLegacy(parseB *testing.B) {
	parseChildrenValue := buildRenderChildrenBenchmarkPayload()
	parseB.Run("legacy_reflect_first", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			buildChildren, parseErr := parseBuildCanonicalChildrenLegacyBenchmark(parseChildrenValue)
			if parseErr != nil {
				parseB.Fatalf("parseBuildCanonicalChildrenLegacyBenchmark returned error: %v", parseErr)
			}
			storeCanonicalChildrenBenchmarkSink = buildChildren
		}
	})
	parseB.Run("current_any_slice_fast_path", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			buildChildren, parseErr := parseBuildCanonicalChildren(parseChildrenValue)
			if parseErr != nil {
				parseB.Fatalf("parseBuildCanonicalChildren returned error: %v", parseErr)
			}
			storeCanonicalChildrenBenchmarkSink = buildChildren
		}
	})
}

// BenchmarkParseBuildCanonicalMapValueCurrentVsLegacy compares current MapRange normalization against the previous MapKeys walk.
func BenchmarkParseBuildCanonicalMapValueCurrentVsLegacy(parseB *testing.B) {
	parseMapValue := reflect.ValueOf(buildRenderMapNormalizationBenchmarkPayload())
	parseB.Run("legacy_map_keys", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			buildMapValue, parseErr := parseBuildCanonicalMapValueLegacyBenchmark(parseMapValue)
			if parseErr != nil {
				parseB.Fatalf("parseBuildCanonicalMapValueLegacyBenchmark returned error: %v", parseErr)
			}
			storeCanonicalMapBenchmarkSink = buildMapValue
		}
	})
	parseB.Run("current_map_range", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			buildMapValue, parseErr := parseBuildCanonicalMapValue(parseMapValue)
			if parseErr != nil {
				parseB.Fatalf("parseBuildCanonicalMapValue returned error: %v", parseErr)
			}
			storeCanonicalMapBenchmarkSink = buildMapValue
		}
	})
}
