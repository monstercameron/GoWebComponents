package app

func hasScrollSpaceBelow(parseScrollTop, parseScrollHeight, parseClientHeight, parseThreshold float64) bool {
	return parseScrollHeight-parseScrollTop-parseClientHeight > parseThreshold
}
