package head

import "testing"

func TestDiffHeadStaleTagsFindsOmittedRobots(parseT *testing.T) {
	parsePrev := buildHeadTags(Document{Robots: "noindex", Social: SocialMetadata{Title: "A"}})
	parseNext := buildHeadTags(Document{Social: SocialMetadata{Title: "B"}})
	parseStale := diffHeadStaleTags(parsePrev, parseNext)
	if len(parseStale) != 1 || parseStale[0].SelectorVal != "robots" {
		parseT.Fatalf("expected only robots stale, got %+v", parseStale)
	}
	if parseStale := diffHeadStaleTags(parseNext, parseNext); len(parseStale) != 0 {
		parseT.Fatalf("identical heads must have no stale tags, got %+v", parseStale)
	}
	if parseStale := diffHeadStaleTags(nil, parseNext); len(parseStale) != 0 {
		parseT.Fatalf("first apply has nothing stale, got %+v", parseStale)
	}
}