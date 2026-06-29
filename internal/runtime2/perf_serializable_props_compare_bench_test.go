package runtime2

import (
	"reflect"
	"testing"
)

var storeSerializablePropsBenchmarkErrSink error

// validateSerializablePropsLegacyBenchmark preserves the previous serializable-props fast-path behavior for compare benchmarks.
func validateSerializablePropsLegacyBenchmark(parseProps any) error {
	if parseProps == nil {
		return nil
	}
	if isSerializableAnyFastLegacyBenchmark(parseProps) {
		return nil
	}
	parseValue := reflectValueOfSerializablePropsBenchmark(parseProps)
	if isSerializableValueFast(parseValue) {
		return nil
	}
	return validateSerializableValue(parseValue, "props")
}

// isSerializableAnyFastLegacyBenchmark preserves the previous exact-type fast-path set from before typed scalar slice/map exits were added.
func isSerializableAnyFastLegacyBenchmark(parseValue any) bool {
	switch getValue := parseValue.(type) {
	case nil:
		return true
	case bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64,
		string:
		return true
	case []any:
		for _, getItem := range getValue {
			if !isSerializableAnyFastLegacyBenchmark(getItem) {
				return false
			}
		}
		return true
	case map[string]any:
		for getKey, getItem := range getValue {
			if !hasSerializableSafeMapKey(getKey) {
				getNormalizedKey := getSerializableNormalizedName(getKey)
				if hasSerializableRefName(getNormalizedKey) {
					return false
				}
				if hasSerializableDOMInteropName(getNormalizedKey) {
					return false
				}
				if hasSerializableEventClosureName(getNormalizedKey) && hasSerializableFunctionValueAnyFast(getItem) {
					return false
				}
			}
			if !isSerializableAnyFastLegacyBenchmark(getItem) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// reflectValueOfSerializablePropsBenchmark wraps reflect.ValueOf through a noinline helper to keep compare benchmarks honest.
//
//go:noinline
func reflectValueOfSerializablePropsBenchmark(parseProps any) reflect.Value {
	return reflect.ValueOf(parseProps)
}

// BenchmarkValidateSerializablePropsCurrentVsLegacy compares typed scalar container validation against the previous reflect-heavy fallback.
func BenchmarkValidateSerializablePropsCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("typed_string_map/current", func(parseB *testing.B) {
		parseProps := map[string]string{
			"title": "Orders",
			"state": "open",
			"owner": "ops",
			"team":  "a",
		}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			storeSerializablePropsBenchmarkErrSink = ValidateSerializableProps(parseProps)
		}
	})
	parseB.Run("typed_string_map/legacy", func(parseB *testing.B) {
		parseProps := map[string]string{
			"title": "Orders",
			"state": "open",
			"owner": "ops",
			"team":  "a",
		}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			storeSerializablePropsBenchmarkErrSink = validateSerializablePropsLegacyBenchmark(parseProps)
		}
	})
	parseB.Run("typed_int_slice/current", func(parseB *testing.B) {
		parseProps := []int{1, 2, 3, 4, 5, 6, 7, 8}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			storeSerializablePropsBenchmarkErrSink = ValidateSerializableProps(parseProps)
		}
	})
	parseB.Run("typed_int_slice/legacy", func(parseB *testing.B) {
		parseProps := []int{1, 2, 3, 4, 5, 6, 7, 8}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			storeSerializablePropsBenchmarkErrSink = validateSerializablePropsLegacyBenchmark(parseProps)
		}
	})
}
