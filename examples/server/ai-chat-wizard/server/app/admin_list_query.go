package app

import (
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
)

type parseAdminListQueryShape struct {
	parseLimit           int32
	parseOffset          int32
	parseSearch          string
	parseSortBy          string
	isParseSortAscending bool
}

// parseBuildAdminListQueryShape resolves one typed list-query contract with legacy limit fallback and safe defaults.
func parseBuildAdminListQueryShape(parseLegacyLimit int32, parseQuery *chatpb.AdminListQuery) parseAdminListQueryShape {
	parseLimit := parseLegacyLimit
	if parseQuery != nil && parseQuery.GetLimit() > 0 {
		parseLimit = parseQuery.GetLimit()
	}
	parseLimit = parseClampAdminListLimit(parseLimit)

	parseOffset := int32(0)
	if parseQuery != nil && parseQuery.GetOffset() > 0 {
		parseOffset = parseQuery.GetOffset()
	}
	parseSearch := ""
	parseSortBy := ""
	isParseSortAscending := false
	if parseQuery != nil {
		parseSearch = strings.TrimSpace(parseQuery.GetSearch())
		parseSortBy = strings.TrimSpace(strings.ToLower(parseQuery.GetSortBy()))
		isParseSortAscending = parseIsAdminSortDirectionAscending(parseQuery.GetSortDirection())
	}
	return parseAdminListQueryShape{
		parseLimit:           parseLimit,
		parseOffset:          parseOffset,
		parseSearch:          parseSearch,
		parseSortBy:          parseSortBy,
		isParseSortAscending: isParseSortAscending,
	}
}

// parseIsAdminSortDirectionAscending reports whether one list sort direction requests ascending order.
func parseIsAdminSortDirectionAscending(parseSortDirection string) bool {
	switch strings.TrimSpace(strings.ToLower(parseSortDirection)) {
	case "asc", "ascending":
		return true
	default:
		return false
	}
}

// parseApplyAdminSliceWindow applies one offset+limit window to one typed row slice.
func parseApplyAdminSliceWindow[parseT any](parseRows []parseT, parseOffset int32, parseLimit int32) []parseT {
	if len(parseRows) == 0 {
		return parseRows
	}
	if parseOffset < 0 {
		parseOffset = 0
	}
	if parseLimit <= 0 {
		parseLimit = defaultAdminListLimit
	}
	if parseOffset >= int32(len(parseRows)) {
		return []parseT{}
	}
	parseEnd := min(int(parseOffset+parseLimit), len(parseRows))
	return parseRows[parseOffset:parseEnd]
}
