package runtime

import (
	"reflect"
	"unsafe"
)

// propsEqual compares two property maps for equality
func propsEqual(parseA, parseB map[string]any) bool {
	if samePropsMap(parseA, parseB) {
		return true
	}
	if len(parseA) == 0 && len(parseB) == 0 {
		return true
	}

	// Fast path: different lengths
	parseALen := len(parseA)
	if parseALen != len(parseB) {
		return false
	}

	for parseK, parseV1 := range parseA {
		parseV2, parseOk := parseB[parseK]
		if !parseOk {
			return false
		}

		if parseK == "children" {
			// Avoid deep comparison for children
			// Check if they are the same slice reference
			// If not, assume they are different to avoid O(Subtree) traversal
			if parseV1 == nil && parseV2 == nil {
				continue
			}
			if parseV1 == nil || parseV2 == nil {
				return false
			}

			// Fast path for the common children representation.
			if parseC1, parseOk1 := parseV1.([]any); parseOk1 {
				if parseC2, parseOk2 := parseV2.([]any); parseOk2 {
					if len(parseC1) == len(parseC2) {
						if len(parseC1) == 0 {
							continue
						}
						if &parseC1[0] == &parseC2[0] {
							continue
						}
					}
					return false
				}
			}

			// Fallback for non-standard slice types.
			parseT1 := reflect.TypeOf(parseV1)
			parseT2 := reflect.TypeOf(parseV2)
			if parseT1 != nil && parseT2 != nil && parseT1.Kind() == reflect.Slice && parseT2.Kind() == reflect.Slice {
				parseRv1 := reflect.ValueOf(parseV1)
				parseRv2 := reflect.ValueOf(parseV2)
				if parseRv1.Pointer() == parseRv2.Pointer() && parseRv1.Len() == parseRv2.Len() {
					continue
				}
			}

			// If pointers differ, assume different.
			// This skips DeepEqual.
			return false
		}

		if !fastEqual(parseV1, parseV2) {
			return false
		}
	}

	return true
}

// propsEqualIgnoringChildren compares two property maps while ignoring child slices.
func propsEqualIgnoringChildren(parseA, parseB map[string]any) bool {
	if samePropsMap(parseA, parseB) {
		return true
	}
	if len(parseA) == 0 && len(parseB) == 0 {
		return true
	}

	parseALen := len(parseA)
	parseBLen := len(parseB)
	if _, hasParseChildrenA := parseA["children"]; hasParseChildrenA {
		parseALen--
	}
	if _, hasParseChildrenB := parseB["children"]; hasParseChildrenB {
		parseBLen--
	}
	if parseALen != parseBLen {
		return false
	}

	// One pass suffices: the children-adjusted lengths already matched, and
	// every non-children key of A was found in B, so B cannot hold an extra
	// non-children key.
	for parseK, parseV1 := range parseA {
		if parseK == "children" {
			continue
		}
		parseV2, parseOk := parseB[parseK]
		if !parseOk {
			return false
		}
		if !fastEqual(parseV1, parseV2) {
			return false
		}
	}

	return true
}

// samePropsMap compares the one-word Go map headers. Host fibers retain the
// element-owned props map, so memoized/reused elements commonly present the
// exact same immutable map on both sides; iterating and probing every key
// cannot discover a difference. In-place mutation was already invisible to
// the old fiber because it aliases this same map, so the fast path does not
// weaken the existing semantics.
func samePropsMap(parseA, parseB map[string]any) bool {
	return *(*unsafe.Pointer)(unsafe.Pointer(&parseA)) == *(*unsafe.Pointer)(unsafe.Pointer(&parseB))
}

// isSameType checks if two component types are the same
func isSameType(parseType1, parseType2 any) bool {
	// String types (HTML tags)
	if parseS1, parseOk1 := parseType1.(string); parseOk1 {
		if parseS2, parseOk2 := parseType2.(string); parseOk2 {
			return parseS1 == parseS2
		}
		return false
	}

	if parseComponent1, parseOk12 := componentTypeFromFiberType(parseType1); parseOk12 {
		parseComponent2, parseOk22 := componentTypeFromFiberType(parseType2)
		if !parseOk22 {
			return false
		}
		return parseComponent1.IdentityKey() != "" && parseComponent1.IdentityKey() == parseComponent2.IdentityKey()
	}

	parseV1 := reflect.ValueOf(parseType1)
	parseV2 := reflect.ValueOf(parseType2)

	if !parseV1.IsValid() || !parseV2.IsValid() {
		return false
	}

	if parseV1.Kind() == reflect.Func && parseV2.Kind() == reflect.Func {
		return sameFunctionIdentity(parseType1, parseType2)
	}

	if parseV1.Type().Comparable() {
		return parseType1 == parseType2
	}

	return reflect.DeepEqual(parseType1, parseType2)
}
