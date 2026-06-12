package agentbridge

import "sort"

// ListMountComponents returns the sorted names of every component registered
// for bridge.mount via RegisterMountComponent. It is the discovery surface an
// agent uses to learn what bridge.mount accepts, instead of guessing component
// names.
func ListMountComponents() []string {
	writeMountMu.Lock()
	defer writeMountMu.Unlock()
	parseNames := make([]string, 0, len(writeMountComponents))
	for parseName := range writeMountComponents {
		parseNames = append(parseNames, parseName)
	}
	sort.Strings(parseNames)
	return parseNames
}
