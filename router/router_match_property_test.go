//go:build js && wasm

package router

import (
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"testing"
)

// TestMatchRoutePatternProperties drives matchRoutePattern with randomized
// patterns and paths and checks invariants rather than examples:
//  1. round-trip: a path built from a pattern and params must match that
//     pattern and recover exactly those params;
//  2. decoded params never contain a path separator;
//  3. no input combination panics;
//  4. segment-count mismatches never match.
func TestMatchRoutePatternProperties(parseT *testing.T) {
	parseRng := rand.New(rand.NewSource(7))
	parseSegments := []string{"users", "posts", "a", "x-y_z", "v2"}
	parseParamValues := []string{"42", "hello world", "naïve", "a.b-c_d", "%", "ok"}

	for parseTrial := 0; parseTrial < 400; parseTrial++ {
		// Build a random pattern of 1-4 segments, some parameterized.
		parseCount := 1 + parseRng.Intn(4)
		parsePatternParts := make([]string, 0, parseCount)
		parseParamNames := make([]string, 0, parseCount)
		for parseIdx := 0; parseIdx < parseCount; parseIdx++ {
			if parseRng.Intn(2) == 0 {
				parseName := fmt.Sprintf("p%d", parseIdx)
				parsePatternParts = append(parsePatternParts, ":"+parseName)
				parseParamNames = append(parseParamNames, parseName)
			} else {
				parsePatternParts = append(parsePatternParts, parseSegments[parseRng.Intn(len(parseSegments))])
			}
		}
		parsePattern := "/" + strings.Join(parsePatternParts, "/")

		// Build a matching path with encoded param values.
		parseWantParams := map[string]string{}
		parsePathParts := make([]string, 0, parseCount)
		parseParamIdx := 0
		for _, parsePart := range parsePatternParts {
			if strings.HasPrefix(parsePart, ":") {
				parseValue := parseParamValues[parseRng.Intn(len(parseParamValues))]
				parseWantParams[parseParamNames[parseParamIdx]] = parseValue
				parsePathParts = append(parsePathParts, url.PathEscape(parseValue))
				parseParamIdx++
			} else {
				parsePathParts = append(parsePathParts, parsePart)
			}
		}
		parsePath := "/" + strings.Join(parsePathParts, "/")

		parseGotParams, parseOk := matchRoutePattern(parsePattern, parsePath)
		if !parseOk {
			parseT.Fatalf("trial %d: round-trip failed: pattern=%q path=%q", parseTrial, parsePattern, parsePath)
		}
		if len(parseGotParams) != len(parseWantParams) {
			parseT.Fatalf("trial %d: param count mismatch: got %v want %v", parseTrial, parseGotParams, parseWantParams)
		}
		for parseName, parseWant := range parseWantParams {
			if parseGotParams[parseName] != parseWant {
				parseT.Fatalf("trial %d: param %s = %q, want %q (pattern=%q path=%q)",
					parseTrial, parseName, parseGotParams[parseName], parseWant, parsePattern, parsePath)
			}
			if strings.Contains(parseGotParams[parseName], "/") {
				parseT.Fatalf("trial %d: decoded param contains separator: %q", parseTrial, parseGotParams[parseName])
			}
		}

		// Segment-count mismatch must never match.
		if _, parseWrong := matchRoutePattern(parsePattern, parsePath+"/extra"); parseWrong {
			parseT.Fatalf("trial %d: pattern %q matched longer path", parseTrial, parsePattern)
		}

		// Garbage inputs must not panic (and %2F must not smuggle separators).
		parseGarbage := []string{"", "///", "/" + url.PathEscape("a/b"), "/%zz", parsePath + "%2F.."}
		for _, parseBad := range parseGarbage {
			func() {
				defer func() {
					if parseRecovered := recover(); parseRecovered != nil {
						parseT.Fatalf("trial %d: panic on input %q: %v", parseTrial, parseBad, parseRecovered)
					}
				}()
				if parseParams, parseOk := matchRoutePattern(parsePattern, parseBad); parseOk {
					for _, parseVal := range parseParams {
						if strings.Contains(parseVal, "/") {
							parseT.Fatalf("trial %d: separator smuggled via %q -> %v", parseTrial, parseBad, parseParams)
						}
					}
				}
			}()
		}
	}
}
