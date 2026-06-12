package runtime2

import (
	"reflect"
	"testing"
)

var storeCanonicalChildrenIRBenchmarkSink []*canonicalRenderNode
var storeCanonicalMapValueIRBenchmarkSink map[string]any

// buildCanonicalChildrenBenchmarkPayload builds one representative mixed-children payload for normalization benchmarks.
func buildCanonicalChildrenBenchmarkPayload() []any {
	buildChildren := make([]any, 0, 48)
	for parseIndex := range 48 {
		buildChildren = append(buildChildren, map[string]any{
			"kind": "host-element",
			"tag":  "li",
			"key":  "item-" + parseFormatCanonicalScalarValue(parseIndex),
			"children": []any{
				map[string]any{
					"kind": "text",
					"text": "value-" + parseFormatCanonicalScalarValue(parseIndex),
				},
			},
		})
	}
	return buildChildren
}

// buildLegacyCanonicalChildrenForBenchmark preserves the previous child normalization path for benchmark comparison.
func buildLegacyCanonicalChildrenForBenchmark(parseChildrenValue any) ([]*canonicalRenderNode, error) {
	if parseChildrenValue == nil {
		return nil, nil
	}
	switch getChildrenValue := parseChildrenValue.(type) {
	case []any:
		buildChildren := make([]*canonicalRenderNode, 0, len(getChildrenValue))
		for _, getChildValue := range getChildrenValue {
			buildNode, parseNodeErr := parseBuildCanonicalRenderNode(getChildValue)
			if parseNodeErr != nil {
				return nil, parseNodeErr
			}
			buildChildren = append(buildChildren, buildNode)
		}
		return buildChildren, nil
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

// buildLegacyCanonicalMapValueForBenchmark preserves the previous map normalization loop for benchmark comparison.
func buildLegacyCanonicalMapValueForBenchmark(parseMapValue reflect.Value) (map[string]any, error) {
	if !parseMapValue.IsValid() {
		return nil, nil
	}
	for parseMapValue.IsValid() && (parseMapValue.Kind() == reflect.Interface || parseMapValue.Kind() == reflect.Pointer) {
		if parseMapValue.IsNil() {
			return nil, nil
		}
		parseMapValue = parseMapValue.Elem()
	}
	if !parseMapValue.IsValid() || parseMapValue.Kind() != reflect.Map {
		return nil, nil
	}
	buildMapValue := make(map[string]any, parseMapValue.Len())
	parseKeyKind := parseMapValue.Type().Key().Kind()
	parseMapIterator := parseMapValue.MapRange()
	for parseMapIterator.Next() {
		getMapKey := parseMapIterator.Key()
		buildKey := ""
		if parseKeyKind == reflect.String {
			buildKey = getMapKey.String()
		} else {
			buildKey = parseFormatCanonicalScalarValue(getMapKey.Interface())
		}
		buildMapValue[buildKey] = parseMapIterator.Value().Interface()
	}
	return buildMapValue, nil
}

// BenchmarkParseBuildCanonicalChildrenIRCurrentVsLegacy compares current child normalization against the previous path.
func BenchmarkParseBuildCanonicalChildrenIRCurrentVsLegacy(parseB *testing.B) {
	getChildren := buildCanonicalChildrenBenchmarkPayload()
	parseB.Run("legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getChildrenNodes, parseErr := buildLegacyCanonicalChildrenForBenchmark(getChildren)
			if parseErr != nil {
				parseB.Fatalf("buildLegacyCanonicalChildrenForBenchmark returned error: %v", parseErr)
			}
			storeCanonicalChildrenIRBenchmarkSink = getChildrenNodes
		}
	})
	parseB.Run("current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getChildrenNodes, parseErr := parseBuildCanonicalChildren(getChildren)
			if parseErr != nil {
				parseB.Fatalf("parseBuildCanonicalChildren returned error: %v", parseErr)
			}
			storeCanonicalChildrenIRBenchmarkSink = getChildrenNodes
		}
	})
}

// BenchmarkParseBuildCanonicalMapValueIRCurrentVsLegacy compares current map normalization against the previous per-entry branch path.
func BenchmarkParseBuildCanonicalMapValueIRCurrentVsLegacy(parseB *testing.B) {
	getStringKeyMap := map[string]any{
		"class": "dashboard",
		"style": "display:grid",
		"text":  "summary",
	}
	getNumericKeyMap := map[uint64]any{
		1: "a",
		2: "b",
		3: "c",
		4: "d",
	}
	parseB.Run("string_keys/legacy", func(parseB *testing.B) {
		parseMapValue := reflect.ValueOf(getStringKeyMap)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getMapValue, parseErr := buildLegacyCanonicalMapValueForBenchmark(parseMapValue)
			if parseErr != nil {
				parseB.Fatalf("buildLegacyCanonicalMapValueForBenchmark returned error: %v", parseErr)
			}
			storeCanonicalMapValueIRBenchmarkSink = getMapValue
		}
	})
	parseB.Run("string_keys/current", func(parseB *testing.B) {
		parseMapValue := reflect.ValueOf(getStringKeyMap)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getMapValue, parseErr := parseBuildCanonicalMapValue(parseMapValue)
			if parseErr != nil {
				parseB.Fatalf("parseBuildCanonicalMapValue returned error: %v", parseErr)
			}
			storeCanonicalMapValueIRBenchmarkSink = getMapValue
		}
	})
	parseB.Run("numeric_keys/legacy", func(parseB *testing.B) {
		parseMapValue := reflect.ValueOf(getNumericKeyMap)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getMapValue, parseErr := buildLegacyCanonicalMapValueForBenchmark(parseMapValue)
			if parseErr != nil {
				parseB.Fatalf("buildLegacyCanonicalMapValueForBenchmark returned error: %v", parseErr)
			}
			storeCanonicalMapValueIRBenchmarkSink = getMapValue
		}
	})
	parseB.Run("numeric_keys/current", func(parseB *testing.B) {
		parseMapValue := reflect.ValueOf(getNumericKeyMap)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getMapValue, parseErr := parseBuildCanonicalMapValue(parseMapValue)
			if parseErr != nil {
				parseB.Fatalf("parseBuildCanonicalMapValue returned error: %v", parseErr)
			}
			storeCanonicalMapValueIRBenchmarkSink = getMapValue
		}
	})
}
