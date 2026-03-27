package runtime2

import (
	"fmt"
	"sort"
)

// RenderStringTable stores canonicalized string entries for render IR.
type RenderStringTable struct {
	Entries                      []string
	storeRenderStringRefByString map[string]uint32
}

// BuildRenderStringTable builds a deduplicated canonical string table.
func BuildRenderStringTable(parseValues []string) RenderStringTable {
	parseUniqueStringSet := make(map[string]struct{}, len(parseValues))
	for _, parseValue := range parseValues {
		parseUniqueStringSet[parseValue] = struct{}{}
	}
	parseEntries := make([]string, 0, len(parseUniqueStringSet))
	for parseValue := range parseUniqueStringSet {
		parseEntries = append(parseEntries, parseValue)
	}
	sort.Strings(parseEntries)
	parseRefByString := make(map[string]uint32, len(parseEntries))
	for parseIndex, parseValue := range parseEntries {
		parseRefByString[parseValue] = uint32(parseIndex)
	}
	return RenderStringTable{
		Entries:                      parseEntries,
		storeRenderStringRefByString: parseRefByString,
	}
}

// ParseRenderStringTable decodes one raw string table and validates canonical ordering plus uniqueness.
func ParseRenderStringTable(parseEntries []string) (RenderStringTable, error) {
	buildEntries := append([]string(nil), parseEntries...)
	for parseIndex := 1; parseIndex < len(buildEntries); parseIndex++ {
		if buildEntries[parseIndex-1] > buildEntries[parseIndex] {
			return RenderStringTable{}, fmt.Errorf("runtime2: string table is not canonical at index %d", parseIndex)
		}
		if buildEntries[parseIndex-1] == buildEntries[parseIndex] {
			return RenderStringTable{}, fmt.Errorf("runtime2: string table duplicates value %q", buildEntries[parseIndex])
		}
	}
	buildRefByString := make(map[string]uint32, len(buildEntries))
	for parseIndex, getValue := range buildEntries {
		buildRefByString[getValue] = uint32(parseIndex)
	}
	return RenderStringTable{
		Entries:                      buildEntries,
		storeRenderStringRefByString: buildRefByString,
	}, nil
}

// GetRenderStringRef returns the table reference for one string value.
func (parseStringTable RenderStringTable) GetRenderStringRef(parseValue string) (uint32, bool) {
	getRenderStringRef, hasRenderStringRef := parseStringTable.storeRenderStringRefByString[parseValue]
	return getRenderStringRef, hasRenderStringRef
}

// GetRenderStringByRef resolves one string-table reference.
func (parseStringTable RenderStringTable) GetRenderStringByRef(parseRef uint32) (string, error) {
	if parseRef >= uint32(len(parseStringTable.Entries)) {
		return "", fmt.Errorf("runtime2: string reference %d is out of range", parseRef)
	}
	return parseStringTable.Entries[parseRef], nil
}
