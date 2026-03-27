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
	// Sort and deduplicate in-place to avoid the intermediate uniqueness-set allocation.
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
	// Parsed tables are frequently consumed via GetRenderStringByRef only; skip reverse-map allocation here.
	return RenderStringTable{
		Entries: buildEntries,
	}, nil
}

// GetRenderStringRef returns the table reference for one string value.
func (parseStringTable RenderStringTable) GetRenderStringRef(parseValue string) (uint32, bool) {
	if parseStringTable.storeRenderStringRefByString != nil {
		getRenderStringRef, hasRenderStringRef := parseStringTable.storeRenderStringRefByString[parseValue]
		return getRenderStringRef, hasRenderStringRef
	}
	for parseIndex, getValue := range parseStringTable.Entries {
		if getValue == parseValue {
			return uint32(parseIndex), true
		}
	}
	return 0, false
}

// GetRenderStringByRef resolves one string-table reference.
func (parseStringTable RenderStringTable) GetRenderStringByRef(parseRef uint32) (string, error) {
	if parseRef >= uint32(len(parseStringTable.Entries)) {
		return "", fmt.Errorf("runtime2: string reference %d is out of range", parseRef)
	}
	return parseStringTable.Entries[parseRef], nil
}
