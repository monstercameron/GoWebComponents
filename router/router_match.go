//go:build js && wasm
// +build js,wasm

package router

import (
	"net/url"
	"strings"
)

func routeIDExact(parsePath string) string {
	return "exact:" + parsePath
}

// routeIDPattern is an internal router helper.
func routeIDPattern(parsePattern string) string {
	return "pattern:" + parsePattern
}

// routeIDNotFound is an internal router helper.
func routeIDNotFound() string {
	return "notfound:*"
}

// expandPathPrefixes is an internal router helper.
func expandPathPrefixes(parsePath string) []string {
	parseParts := splitPath(parsePath)
	if len(parseParts) == 0 {
		return []string{"/"}
	}
	parsePrefixes := make([]string, 0, len(parseParts)+1)
	parsePrefixes = append(parsePrefixes, "/")
	for parseIndex := range parseParts {
		parsePrefixes = append(parsePrefixes, "/"+strings.Join(parseParts[:parseIndex+1], "/"))
	}
	return parsePrefixes
}

// matchRoutePattern is an internal router helper.
func matchRoutePattern(parsePattern, parsePath string) (map[string]string, bool) {
	if strings.HasSuffix(parsePattern, "*") {
		parsePrefix := strings.TrimSuffix(parsePattern, "*")
		parsePrefix = strings.TrimSuffix(parsePrefix, "/")
		if parsePrefix == "" {
			return map[string]string{}, true
		}
		if parsePath == parsePrefix || strings.HasPrefix(parsePath, parsePrefix+"/") {
			return map[string]string{}, true
		}
		return nil, false
	}

	parsePatternParts := splitPath(parsePattern)
	parsePathParts := splitPath(parsePath)
	if len(parsePatternParts) != len(parsePathParts) {
		return nil, false
	}

	parseParams := make(map[string]string)
	for parseIndex, parsePart := range parsePatternParts {
		parseCandidate := parsePathParts[parseIndex]
		if strings.HasPrefix(parsePart, ":") {
			parseName := strings.TrimPrefix(parsePart, ":")
			if parseName == "" || strings.ContainsAny(parseName, "?*") {
				return nil, false
			}
			parseDecoded, parseErr := url.PathUnescape(parseCandidate)
			if parseErr != nil || parseDecoded == "" {
				return nil, false
			}
			parseParams[parseName] = parseDecoded
			continue
		}
		if parsePart != parseCandidate {
			return nil, false
		}
	}

	return parseParams, true
}

// splitPath is an internal router helper.
func splitPath(parsePath string) []string {
	parseTrimmed := strings.Trim(strings.TrimSpace(parsePath), "/")
	if parseTrimmed == "" {
		return []string{}
	}
	return strings.Split(parseTrimmed, "/")
}

// copyParams is an internal router helper.
func copyParams(parseParams map[string]string) map[string]string {
	if len(parseParams) == 0 {
		return map[string]string{}
	}
	parseClone := make(map[string]string, len(parseParams))
	for parseKey, parseValue := range parseParams {
		parseClone[parseKey] = parseValue
	}
	return parseClone
}

// copyParamsToAttrs is an internal router helper.
func copyParamsToAttrs(parseParams map[string]string) Attrs {
	if len(parseParams) == 0 {
		return nil
	}
	parseAttrs := make(Attrs, len(parseParams))
	for parseKey, parseValue := range parseParams {
		parseAttrs[parseKey] = parseValue
	}
	return parseAttrs
}

// copyAttrs is an internal router helper.
func copyAttrs(parseAttrs Attrs) Attrs {
	if len(parseAttrs) == 0 {
		return nil
	}
	parseClone := make(Attrs, len(parseAttrs))
	for parseKey, parseValue := range parseAttrs {
		parseClone[parseKey] = parseValue
	}
	return parseClone
}

// copyQueryValues is an internal router helper.
func copyQueryValues(parseValues url.Values) url.Values {
	parseClone := make(url.Values, len(parseValues))
	for parseKey, parseValue := range parseValues {
		parseClone[parseKey] = append([]string(nil), parseValue...)
	}
	return parseClone
}

// mergeAttrs is an internal router helper.
func mergeAttrs(parseBase Attrs, parseExtra Attrs) Attrs {
	if len(parseBase) == 0 && len(parseExtra) == 0 {
		return nil
	}
	parseMerged := make(Attrs, len(parseBase)+len(parseExtra))
	for parseKey, parseValue := range parseBase {
		parseMerged[parseKey] = parseValue
	}
	for parseKey2, parseValue2 := range parseExtra {
		parseMerged[parseKey2] = parseValue2
	}
	return parseMerged
}
