package app

func hasScrollSpaceBelow(scrollTop, scrollHeight, clientHeight, threshold float64) bool {
	return scrollHeight-scrollTop-clientHeight > threshold
}
