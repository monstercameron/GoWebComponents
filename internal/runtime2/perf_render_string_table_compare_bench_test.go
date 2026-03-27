package runtime2

import (
	"sort"
	"strconv"
	"testing"
)

var storeRenderStringTableBenchmarkSink RenderStringTable
var storeRenderStringRefBenchmarkSink uint32
var hasRenderStringRefBenchmarkSink bool

// buildRenderStringTableBenchmarkValues builds one duplicate-heavy string slice used by string-table build benchmarks.
func buildRenderStringTableBenchmarkValues() []string {
	buildValues := make([]string, 0, 4096)
	for parseIndex := 0; parseIndex < 2048; parseIndex++ {
		getValue := "v-" + strconv.Itoa(parseIndex%512)
		buildValues = append(buildValues, getValue)
	}
	return buildValues
}

// buildRenderStringTableTinyBenchmarkValues builds one tiny string slice used to benchmark BuildRenderStringTable tiny fast paths.
func buildRenderStringTableTinyBenchmarkValues() []string {
	return []string{"solo"}
}

// buildRenderStringTableLegacyBenchmark preserves the previous string-table build path that populated the reverse map in a second pass.
func buildRenderStringTableLegacyBenchmark(parseValues []string) RenderStringTable {
	parseEntries := append(make([]string, 0, len(parseValues)), parseValues...)
	sort.Strings(parseEntries)
	buildWrite := 0
	for parseRead := 0; parseRead < len(parseEntries); parseRead++ {
		if parseRead == 0 || parseEntries[parseRead] != parseEntries[parseRead-1] {
			parseEntries[buildWrite] = parseEntries[parseRead]
			buildWrite++
		}
	}
	parseEntries = parseEntries[:buildWrite]
	if len(parseEntries) <= getRenderStringTableInlineLookupEntryLimit {
		return RenderStringTable{
			Entries: parseEntries,
		}
	}
	parseRefByString := make(map[string]uint32, len(parseEntries))
	for parseIndex, parseValue := range parseEntries {
		parseRefByString[parseValue] = uint32(parseIndex)
	}
	return RenderStringTable{
		Entries:                      parseEntries,
		storeRenderStringRefByString: parseRefByString,
	}
}

// getRenderStringRefLegacyBenchmark preserves the previous no-map lookup path that always used binary search.
func getRenderStringRefLegacyBenchmark(parseStringTable RenderStringTable, parseValue string) (uint32, bool) {
	if parseStringTable.storeRenderStringRefByString != nil {
		getRenderStringRef, hasRenderStringRef := parseStringTable.storeRenderStringRefByString[parseValue]
		return getRenderStringRef, hasRenderStringRef
	}
	parseIndex := sort.SearchStrings(parseStringTable.Entries, parseValue)
	if parseIndex >= len(parseStringTable.Entries) {
		return 0, false
	}
	if parseStringTable.Entries[parseIndex] == parseValue {
		return uint32(parseIndex), true
	}
	return 0, false
}

// BenchmarkBuildRenderStringTableCurrentVsLegacy compares current tiny-table fast path behavior against the previous always-sort path.
func BenchmarkBuildRenderStringTableCurrentVsLegacy(parseB *testing.B) {
	parseTinyValues := buildRenderStringTableTinyBenchmarkValues()
	parseLargeValues := buildRenderStringTableBenchmarkValues()
	parseB.Run("tiny_table/legacy_always_sort", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			storeRenderStringTableBenchmarkSink = buildRenderStringTableLegacyBenchmark(parseTinyValues)
		}
	})
	parseB.Run("tiny_table/current_fast_path", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			storeRenderStringTableBenchmarkSink = BuildRenderStringTable(parseTinyValues)
		}
	})
	parseB.Run("large_table/legacy_second_pass_map", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			storeRenderStringTableBenchmarkSink = buildRenderStringTableLegacyBenchmark(parseLargeValues)
		}
	})
	parseB.Run("large_table/current_map_build", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			storeRenderStringTableBenchmarkSink = BuildRenderStringTable(parseLargeValues)
		}
	})
}

// BenchmarkGetRenderStringRefCurrentVsLegacy compares current tiny-table linear fallback against the previous binary-search-only fallback.
func BenchmarkGetRenderStringRefCurrentVsLegacy(parseB *testing.B) {
	getSmallEntries := make([]string, 0, 4)
	for parseIndex := 0; parseIndex < 4; parseIndex++ {
		getSmallEntries = append(getSmallEntries, "s-"+strconv.Itoa(parseIndex))
	}
	sort.Strings(getSmallEntries)
	getSmallTable, parseSmallErr := ParseRenderStringTable(getSmallEntries)
	if parseSmallErr != nil {
		parseB.Fatalf("ParseRenderStringTable(small) returned error: %v", parseSmallErr)
	}
	getLargeEntries := make([]string, 0, 512)
	for parseIndex := 0; parseIndex < 512; parseIndex++ {
		getLargeEntries = append(getLargeEntries, "l-"+strconv.Itoa(parseIndex))
	}
	sort.Strings(getLargeEntries)
	getLargeTable, parseLargeErr := ParseRenderStringTable(getLargeEntries)
	if parseLargeErr != nil {
		parseB.Fatalf("ParseRenderStringTable(large) returned error: %v", parseLargeErr)
	}
	parseB.Run("small_table/legacy_binary", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			storeRenderStringRefBenchmarkSink, hasRenderStringRefBenchmarkSink = getRenderStringRefLegacyBenchmark(getSmallTable, "s-2")
		}
	})
	parseB.Run("small_table/current_inline_linear", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			storeRenderStringRefBenchmarkSink, hasRenderStringRefBenchmarkSink = getSmallTable.GetRenderStringRef("s-2")
		}
	})
	parseB.Run("large_table/legacy_binary", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			storeRenderStringRefBenchmarkSink, hasRenderStringRefBenchmarkSink = getRenderStringRefLegacyBenchmark(getLargeTable, "l-377")
		}
	})
	parseB.Run("large_table/current_binary", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			storeRenderStringRefBenchmarkSink, hasRenderStringRefBenchmarkSink = getLargeTable.GetRenderStringRef("l-377")
		}
	})
}
