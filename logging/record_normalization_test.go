package logging

import (
	"log/slog"
	"reflect"
	"testing"
	"time"
)

// TestNormalizeLogValueSemantics pins down the JSON-safety contract for every
// value shape that can reach a log backend: scalars pass through, times format
// as RFC3339Nano UTC, durations stringify, maps and slices recurse with
// normalized members, and unknown types degrade to printable forms.
func TestNormalizeLogValueSemantics(parseT *testing.T) {
	parseWhen := time.Date(2026, 6, 11, 15, 4, 5, 123456789, time.FixedZone("X", 3600))

	parseCases := []struct {
		Name string
		In   any
		Want any
	}{
		{"nil", nil, nil},
		{"string", "s", "s"},
		{"bool", true, true},
		{"int", 7, 7},
		{"int64", int64(-9), int64(-9)},
		{"uint8", uint8(255), uint8(255)},
		{"float64", 2.5, 2.5},
		{"duration", 1500 * time.Millisecond, "1.5s"},
		{"time-utc-rfc3339nano", parseWhen, "2026-06-11T14:04:05.123456789Z"},
		{"error", errFixture{"boom"}, "boom"},
		{"string-map", map[string]any{"a": 1, "t": parseWhen}, map[string]any{"a": 1, "t": "2026-06-11T14:04:05.123456789Z"}},
		{"int-keyed-map", map[int]string{3: "x"}, map[string]any{"3": "x"}},
		{"slice", []any{1, "two", 3 * time.Second}, []any{1, "two", "3s"}},
		{"int-slice", []int{1, 2}, []any{1, 2}},
		{"nested", map[string]any{"inner": []any{map[string]any{"d": time.Second}}},
			map[string]any{"inner": []any{map[string]any{"d": "1s"}}}},
	}
	for _, parseCase := range parseCases {
		parseT.Run(parseCase.Name, func(parseT2 *testing.T) {
			parseGot := normalizeLogValue(parseCase.In)
			if !reflect.DeepEqual(parseGot, parseCase.Want) {
				parseT2.Fatalf("normalizeLogValue(%#v) = %#v, want %#v", parseCase.In, parseGot, parseCase.Want)
			}
		})
	}
}

type errFixture struct{ parseMsg string }

func (parseE errFixture) Error() string { return parseE.parseMsg }

// TestBuildSlogValueKinds verifies every slog kind resolves to the same
// JSON-safe shapes, including nested groups.
func TestBuildSlogValueKinds(parseT *testing.T) {
	parseWhen := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	parseCases := []struct {
		Name string
		In   slog.Value
		Want any
	}{
		{"bool", slog.BoolValue(true), true},
		{"duration", slog.DurationValue(2 * time.Second), "2s"},
		{"float", slog.Float64Value(1.25), 1.25},
		{"int64", slog.Int64Value(42), int64(42)},
		{"string", slog.StringValue("v"), "v"},
		{"time", slog.TimeValue(parseWhen), "2026-01-02T03:04:05Z"},
		{"uint64", slog.Uint64Value(9), uint64(9)},
		{"any", slog.AnyValue(3 * time.Minute), "3m0s"},
		{"group", slog.GroupValue(slog.String("a", "1"), slog.Int("b", 2), slog.Attr{Key: " ", Value: slog.StringValue("dropped")}),
			map[string]any{"a": "1", "b": int64(2)}},
	}
	for _, parseCase := range parseCases {
		parseT.Run(parseCase.Name, func(parseT2 *testing.T) {
			parseGot := buildSlogValue(parseCase.In)
			if !reflect.DeepEqual(parseGot, parseCase.Want) {
				parseT2.Fatalf("buildSlogValue(%v) = %#v, want %#v", parseCase.In, parseGot, parseCase.Want)
			}
		})
	}
}

// TestFieldKeyProtection verifies reserved record keys cannot be clobbered by
// user fields and positional arguments get stable keys.
func TestFieldKeyProtection(parseT *testing.T) {
	if parseGot := buildArgumentFieldKey(3); parseGot != "arg_3" {
		parseT.Fatalf("argument key = %q", parseGot)
	}
	if parseGot := protectLogFieldKey("  "); parseGot != "" {
		parseT.Fatalf("blank key should empty, got %q", parseGot)
	}
	if parseGot := protectLogFieldKey("custom"); parseGot != "custom" {
		parseT.Fatalf("plain key mangled: %q", parseGot)
	}
	parseProtected := 0
	for parseReserved := range reservedRecordKeys {
		parseGot := protectLogFieldKey(parseReserved)
		if parseGot == parseReserved {
			parseT.Fatalf("reserved key %q not protected", parseReserved)
		}
		if parseGot != "field_"+parseReserved {
			parseT.Fatalf("reserved key %q protected as %q", parseReserved, parseGot)
		}
		parseProtected++
	}
	if parseProtected == 0 {
		parseT.Fatal("no reserved keys exist — protection untested")
	}
}

// TestBuildMapAndSliceValueEdges covers nil and empty reflected containers.
func TestBuildMapAndSliceValueEdges(parseT *testing.T) {
	var parseNilMap map[string]int
	if parseGot := buildMapValue(reflect.ValueOf(parseNilMap), 0); parseGot != nil {
		parseT.Fatalf("nil map should normalize to nil, got %#v", parseGot)
	}
	var parseNilSlice []int
	if parseGot := buildSliceValue(reflect.ValueOf(parseNilSlice), 0); parseGot != nil {
		parseT.Fatalf("nil slice should normalize to nil, got %#v", parseGot)
	}
	if parseGot := buildSliceValue(reflect.ValueOf([2]string{"a", "b"}), 0); !reflect.DeepEqual(parseGot, []any{"a", "b"}) {
		parseT.Fatalf("array normalization: %#v", parseGot)
	}
	if parseGot := buildMapValue(reflect.ValueOf(map[string]int{"": 1, "k": 2}), 0); !reflect.DeepEqual(parseGot, map[string]any{"k": 2}) {
		parseT.Fatalf("empty-key map entry should drop: %#v", parseGot)
	}
}
