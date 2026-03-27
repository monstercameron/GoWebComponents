package runtime2

import (
	"fmt"
	"sort"
	"strings"
)

var storeRenderAllowedHostTagSet = map[string]struct{}{
	"article": {},
	"aside":   {},
	"div":     {},
	"em":      {},
	"footer":  {},
	"h1":      {},
	"h2":      {},
	"h3":      {},
	"header":  {},
	"li":      {},
	"main":    {},
	"nav":     {},
	"ol":      {},
	"p":       {},
	"section": {},
	"small":   {},
	"span":    {},
	"strong":  {},
	"ul":      {},
}

// GetRenderAllowedHostTags returns the explicit first-slice allowed host-tag set in stable sorted order.
func GetRenderAllowedHostTags() []string {
	getRenderAllowedHostTags := make([]string, 0, len(storeRenderAllowedHostTagSet))
	for getRenderAllowedHostTag := range storeRenderAllowedHostTagSet {
		getRenderAllowedHostTags = append(getRenderAllowedHostTags, getRenderAllowedHostTag)
	}
	sort.Strings(getRenderAllowedHostTags)
	return getRenderAllowedHostTags
}

// ValidateWorkerRenderableHostTag verifies one host tag is explicitly allowed for first-slice worker-renderable regions.
func ValidateWorkerRenderableHostTag(parseHostTag string) error {
	parseTrimmedHostTag := strings.TrimSpace(parseHostTag)
	if parseTrimmedHostTag == "" {
		return fmt.Errorf("runtime2: host tag is required")
	}
	if parseTrimmedHostTag != parseHostTag {
		return fmt.Errorf("runtime2: host tag %q must not contain surrounding whitespace", parseHostTag)
	}
	if _, hasRenderAllowedHostTag := storeRenderAllowedHostTagSet[parseHostTag]; !hasRenderAllowedHostTag {
		return fmt.Errorf("runtime2: host tag %q is not allowed in first-slice worker-renderable regions", parseHostTag)
	}
	return nil
}
