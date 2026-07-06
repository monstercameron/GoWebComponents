package router

import "strings"

// Route-shadowing detection. matchPattern tries registered patterns in
// REGISTRATION ORDER and returns the first match, so a pattern registered after a
// broader one that already covers all of its paths is UNREACHABLE (e.g. "/users/*"
// before "/users/new", or "/users/:id" before "/users/admin"). Register warns when
// it detects this. The logic is build-tag-free (the matcher itself is js,wasm only)
// so the subsumption rules can be unit-tested natively.

// Distinctive placeholder tokens for a pattern's variable segments — a NUL-wrapped
// sentinel cannot collide with a real static path segment, so "earlier matches the
// concrete instance of later" holds IFF earlier subsumes every path later matches.
const (
	shadowParamToken = "\x00gwc-param\x00"
	shadowWildToken  = "\x00gwc-wild\x00"
)

// patternShadows reports whether the earlier pattern fully subsumes the later
// pattern — i.e. every path that later could match is also matched by earlier, so
// later is unreachable when earlier is registered first. Identical patterns are not
// treated as shadowing (that is the caller's duplicate-registration concern).
func patternShadows(parseEarlier, parseLater string) bool {
	if parseEarlier == parseLater {
		return false
	}
	// A trailing-* wildcard matches paths of UNBOUNDED depth. A fixed-arity
	// pattern (only static/:param segments) can never subsume all of them, so it
	// cannot shadow a wildcard — only another wildcard can. Without this guard the
	// single-representative concrete path (one extra segment) makes e.g. "/a/:x"
	// look like it shadows "/a/*", a false unreachable warning for a route that
	// still matches "/a/b/c".
	if strings.HasSuffix(parseLater, "*") && !strings.HasSuffix(parseEarlier, "*") {
		return false
	}
	return shadowPatternMatches(parseEarlier, shadowConcretePath(parseLater))
}

// shadowConcretePath builds a representative concrete path for a pattern, filling
// :param segments and a trailing * with distinctive sentinels.
func shadowConcretePath(parsePattern string) string {
	if strings.HasSuffix(parsePattern, "*") {
		parsePrefix := strings.TrimSuffix(parsePattern, "*")
		parsePrefix = strings.TrimSuffix(parsePrefix, "/")
		return parsePrefix + "/" + shadowWildToken
	}
	parseParts := shadowSplitPath(parsePattern)
	for parseIndex, parsePart := range parseParts {
		if strings.HasPrefix(parsePart, ":") {
			parseParts[parseIndex] = shadowParamToken
		}
	}
	return "/" + strings.Join(parseParts, "/")
}

// shadowPatternMatches mirrors matchRoutePattern's match decision (bool only),
// build-tag-free so it can run under both native and wasm.
func shadowPatternMatches(parsePattern, parsePath string) bool {
	if strings.HasSuffix(parsePattern, "*") {
		parsePrefix := strings.TrimSuffix(parsePattern, "*")
		parsePrefix = strings.TrimSuffix(parsePrefix, "/")
		if parsePrefix == "" {
			return true
		}
		return parsePath == parsePrefix || strings.HasPrefix(parsePath, parsePrefix+"/")
	}
	parsePatternParts := shadowSplitPath(parsePattern)
	parsePathParts := shadowSplitPath(parsePath)
	if len(parsePatternParts) != len(parsePathParts) {
		return false
	}
	for parseIndex, parsePart := range parsePatternParts {
		if strings.HasPrefix(parsePart, ":") {
			continue // a :param matches any single non-empty segment
		}
		if parsePart != parsePathParts[parseIndex] {
			return false
		}
	}
	return true
}

// shadowSplitPath mirrors splitPath (which lives in the js,wasm-only matcher file).
func shadowSplitPath(parsePath string) []string {
	parseTrimmed := strings.Trim(strings.TrimSpace(parsePath), "/")
	if parseTrimmed == "" {
		return []string{}
	}
	return strings.Split(parseTrimmed, "/")
}
