package pluginruntime

import "fmt"

// guardedResult stores the result of one guarded plugin operation.
type guardedResult struct {
	getErr       error
	getRecovered any
}

// runGuardedCall executes one plugin-owned callback and converts panics into errors.
func runGuardedCall(parseOperation string, parseCall func() error) (parseResult guardedResult) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseResult.getRecovered = parseRecovered
			parseResult.getErr = fmt.Errorf("pluginruntime: panic during %s: %v", parseOperation, parseRecovered)
		}
	}()
	if parseCall == nil {
		return guardedResult{}
	}
	parseResult.getErr = parseCall()
	return parseResult
}
