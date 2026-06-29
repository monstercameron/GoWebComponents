package runtime2

import (
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var storeCanonicalPropRecordsBenchmarkSink []RenderPropRecord

// parseBuildCanonicalPropRecordLegacyBenchmark preserves the previous prop-record key normalization path that always trimmed keys.
func parseBuildCanonicalPropRecordLegacyBenchmark(parseKey string, parseRawValue any) (RenderPropRecord, bool, error) {
	buildKey := strings.TrimSpace(parseKey)
	if buildKey == "" {
		return RenderPropRecord{}, false, nil
	}
	buildPropRecord := RenderPropRecord{
		Key: buildKey,
	}
	switch {
	case buildKey == "class":
		buildPropRecord.Kind = RenderPropKindClass
		buildPropRecord.Value = parseGetCanonicalStringValue(parseRawValue)
	case buildKey == "style":
		buildStyleValue, parseStyleErr := FormatRenderStyleValue(parseRawValue)
		if parseStyleErr != nil {
			return RenderPropRecord{}, false, parseStyleErr
		}
		buildPropRecord.Kind = RenderPropKindStyle
		buildPropRecord.Value = buildStyleValue
	case strings.HasPrefix(buildKey, "aria-"):
		buildPropRecord.Kind = RenderPropKindAria
		buildPropRecord.Value = parseGetCanonicalStringValue(parseRawValue)
	case strings.HasPrefix(buildKey, "data-"):
		buildPropRecord.Kind = RenderPropKindData
		buildPropRecord.Value = parseGetCanonicalStringValue(parseRawValue)
	case buildKey == "text":
		buildPropRecord.Kind = RenderPropKindTextAdjacent
		buildPropRecord.Value = parseGetCanonicalStringValue(parseRawValue)
	default:
		return RenderPropRecord{}, false, nil
	}
	buildPropFamily := "class"
	switch buildPropRecord.Kind {
	case RenderPropKindClass:
		buildPropFamily = "class"
	case RenderPropKindStyle:
		buildPropFamily = "style"
	case RenderPropKindAria:
		buildPropFamily = "aria"
	case RenderPropKindData:
		buildPropFamily = "data"
	}
	if buildPropRecord.Kind != RenderPropKindTextAdjacent {
		if parseFamilyErr := ValidateWorkerRenderablePropFamily(buildPropFamily); parseFamilyErr != nil {
			return RenderPropRecord{}, false, parseFamilyErr
		}
	}
	return buildPropRecord, true, nil
}

// parseBuildCanonicalPropsLegacyBenchmark preserves the previous prop extraction/sort behavior for compare benchmarks.
func parseBuildCanonicalPropsLegacyBenchmark(parseMapValue map[string]any) ([]RenderPropRecord, error) {
	buildRawValueByKey := make(map[string]any, len(parseMapValue))
	if parsePropsValue, hasPropsValue := parseMapValue["props"]; hasPropsValue {
		switch getPropsValue := parsePropsValue.(type) {
		case map[string]any:
			maps.Copy(buildRawValueByKey, getPropsValue)
		case map[string]string:
			for getPropKey, getPropValue := range getPropsValue {
				buildRawValueByKey[getPropKey] = getPropValue
			}
		default:
			parseReflectProps := reflect.ValueOf(parsePropsValue)
			buildPropsValue, parsePropsErr := parseBuildCanonicalMapValue(parseReflectProps)
			if parsePropsErr == nil {
				maps.Copy(buildRawValueByKey, buildPropsValue)
			}
		}
	}
	for getKey, getValue := range parseMapValue {
		if _, hasReservedKey := storeCanonicalReservedFieldSet[getKey]; hasReservedKey {
			continue
		}
		buildRawValueByKey[getKey] = getValue
	}
	buildPropRecords := make([]RenderPropRecord, 0, len(buildRawValueByKey))
	for getPropKey, getPropRawValue := range buildRawValueByKey {
		buildPropRecord, hasPropRecord, parsePropErr := parseBuildCanonicalPropRecordLegacyBenchmark(getPropKey, getPropRawValue)
		if parsePropErr != nil {
			return nil, parsePropErr
		}
		if !hasPropRecord {
			continue
		}
		buildPropRecords = append(buildPropRecords, buildPropRecord)
	}
	sort.Slice(buildPropRecords, func(parseLeftIndex int, parseRightIndex int) bool {
		parseLeftRecord := buildPropRecords[parseLeftIndex]
		parseRightRecord := buildPropRecords[parseRightIndex]
		if parseLeftRecord.Key == parseRightRecord.Key {
			return parseLeftRecord.Kind < parseRightRecord.Kind
		}
		return parseLeftRecord.Key < parseRightRecord.Key
	})
	for parseIndex := 1; parseIndex < len(buildPropRecords); parseIndex++ {
		if buildPropRecords[parseIndex-1].Key == buildPropRecords[parseIndex].Key {
			return nil, fmt.Errorf("runtime2: duplicate prop key %q", buildPropRecords[parseIndex].Key)
		}
	}
	return buildPropRecords, nil
}

// buildRenderPropsBenchmarkPayloadSingle builds one single-prop host payload used by prop extraction compare benchmarks.
func buildRenderPropsBenchmarkPayloadSingle() map[string]any {
	return map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"class": "card",
		},
	}
}

// buildRenderPropsBenchmarkPayloadMany builds one multi-prop host payload used by prop extraction compare benchmarks.
func buildRenderPropsBenchmarkPayloadMany() map[string]any {
	return map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"class":      "card primary",
			"style":      map[string]any{"display": "block", "gap": "8px"},
			"aria-label": "panel",
			"data-id":    "42",
			"text":       "adjacent",
		},
	}
}

// BenchmarkParseBuildCanonicalPropsCurrentVsLegacy compares current prop extraction/sort against the previous trim-and-sort behavior.
func BenchmarkParseBuildCanonicalPropsCurrentVsLegacy(parseB *testing.B) {
	buildPayloads := []struct {
		getName    string
		getPayload map[string]any
	}{
		{getName: "single_prop", getPayload: buildRenderPropsBenchmarkPayloadSingle()},
		{getName: "many_props", getPayload: buildRenderPropsBenchmarkPayloadMany()},
	}
	for _, getPayload := range buildPayloads {
		parseB.Run(getPayload.getName+"/legacy", func(parseB *testing.B) {
			parseB.ReportAllocs()
			parseB.ResetTimer()
			for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
				buildProps, parseErr := parseBuildCanonicalPropsLegacyBenchmark(getPayload.getPayload)
				if parseErr != nil {
					parseB.Fatalf("parseBuildCanonicalPropsLegacyBenchmark returned error: %v", parseErr)
				}
				storeCanonicalPropRecordsBenchmarkSink = buildProps
			}
		})
		parseB.Run(getPayload.getName+"/current", func(parseB *testing.B) {
			parseB.ReportAllocs()
			parseB.ResetTimer()
			for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
				buildProps, parseErr := parseBuildCanonicalProps(getPayload.getPayload)
				if parseErr != nil {
					parseB.Fatalf("parseBuildCanonicalProps returned error: %v", parseErr)
				}
				storeCanonicalPropRecordsBenchmarkSink = buildProps
			}
		})
	}
}
