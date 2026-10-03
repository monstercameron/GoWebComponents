package head

// headTag describes one client-managed <meta>/<link> element: the selector that
// finds it and the attribute that carries its value.
type headTag struct {
	Tag, SelectorAttr, SelectorVal, ValueAttr, Value string
}

// buildHeadTags lists the head tags a Document wants set, skipping empty values.
// It is pure so the stale-tag diff can be tested without a DOM.
func buildHeadTags(parseDoc Document) []headTag {
	parseAll := []headTag{
		{"meta", "name", "description", "content", parseDoc.Metadata.Description},
		{"meta", "name", "robots", "content", parseDoc.Robots},
		{"link", "rel", "canonical", "href", parseDoc.Metadata.CanonicalURL},
		{"meta", "property", "og:type", "content", parseDoc.Social.Type},
		{"meta", "property", "og:title", "content", parseDoc.Social.Title},
		{"meta", "property", "og:description", "content", parseDoc.Social.Description},
		{"meta", "property", "og:image", "content", parseDoc.Social.ImageURL},
		{"meta", "property", "og:url", "content", parseDoc.Social.URL},
		{"meta", "name", "twitter:card", "content", parseDoc.Social.TwitterCard},
		{"meta", "name", "twitter:title", "content", parseDoc.Social.Title},
		{"meta", "name", "twitter:description", "content", parseDoc.Social.Description},
		{"meta", "name", "twitter:image", "content", parseDoc.Social.ImageURL},
	}
	parseTags := parseAll[:0]
	for _, parseTag := range parseAll {
		if parseTag.Value != "" {
			parseTags = append(parseTags, parseTag)
		}
	}
	return parseTags
}

// diffHeadStaleTags returns the tags present in parsePrev whose selector is
// absent from parseNext: the previous page set them and the next page omitted
// them, so they must be removed or a stale robots=noindex would persist.
func diffHeadStaleTags(parsePrev, parseNext []headTag) []headTag {
	parseKept := make(map[[3]string]bool, len(parseNext))
	for _, parseTag := range parseNext {
		parseKept[[3]string{parseTag.Tag, parseTag.SelectorAttr, parseTag.SelectorVal}] = true
	}
	var parseStale []headTag
	for _, parseTag := range parsePrev {
		if !parseKept[[3]string{parseTag.Tag, parseTag.SelectorAttr, parseTag.SelectorVal}] {
			parseStale = append(parseStale, parseTag)
		}
	}
	return parseStale
}