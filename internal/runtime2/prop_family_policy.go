package runtime2

import (
	"fmt"
	"sort"
	"strings"
)

var storeRenderAllowedPropFamilySet = map[string]struct{}{
	"aria":  {},
	"class": {},
	"data":  {},
	"style": {},
}

// GetRenderAllowedPropFamilies returns the explicit first-slice allowed prop-family set in stable sorted order.
func GetRenderAllowedPropFamilies() []string {
	getRenderAllowedPropFamilies := make([]string, 0, len(storeRenderAllowedPropFamilySet))
	for getRenderAllowedPropFamily := range storeRenderAllowedPropFamilySet {
		getRenderAllowedPropFamilies = append(getRenderAllowedPropFamilies, getRenderAllowedPropFamily)
	}
	sort.Strings(getRenderAllowedPropFamilies)
	return getRenderAllowedPropFamilies
}

// ValidateWorkerRenderablePropFamily verifies one prop family is explicitly allowed for first-slice worker-renderable regions.
func ValidateWorkerRenderablePropFamily(parsePropFamily string) error {
	parseTrimmedPropFamily := strings.TrimSpace(parsePropFamily)
	if parseTrimmedPropFamily == "" {
		return fmt.Errorf("runtime2: prop family is required")
	}
	if parseTrimmedPropFamily != parsePropFamily {
		return fmt.Errorf("runtime2: prop family %q must not contain surrounding whitespace", parsePropFamily)
	}
	if _, hasRenderAllowedPropFamily := storeRenderAllowedPropFamilySet[parsePropFamily]; !hasRenderAllowedPropFamily {
		return fmt.Errorf("runtime2: prop family %q is not allowed in first-slice worker-renderable regions", parsePropFamily)
	}
	return nil
}
