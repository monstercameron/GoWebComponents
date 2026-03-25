package devtools

import "sync"

var errorOverlayActions struct {
	mu      sync.RWMutex
	actions []ErrorOverlayAction
}

// SetErrorOverlayActions replaces the current app-owned recovery actions for the error overlay.
func SetErrorOverlayActions(parseActions []ErrorOverlayAction) {
	errorOverlayActions.mu.Lock()
	defer errorOverlayActions.mu.Unlock()
	errorOverlayActions.actions = cloneErrorOverlayActions(parseActions)
}

// ResetErrorOverlayActions clears the current app-owned recovery actions for the error overlay.
func ResetErrorOverlayActions() {
	SetErrorOverlayActions(nil)
}

// InspectErrorOverlayActions returns the current app-owned recovery actions for the error overlay.
func InspectErrorOverlayActions() []ErrorOverlayAction {
	errorOverlayActions.mu.RLock()
	defer errorOverlayActions.mu.RUnlock()
	return cloneErrorOverlayActions(errorOverlayActions.actions)
}

func cloneErrorOverlayActions(parseActions []ErrorOverlayAction) []ErrorOverlayAction {
	if len(parseActions) == 0 {
		return nil
	}
	parseCloned := make([]ErrorOverlayAction, len(parseActions))
	for parseI, parseAction := range parseActions {
		parseCloned[parseI] = ErrorOverlayAction{
			Label:        parseAction.Label,
			MatchCodes:   append([]string(nil), parseAction.MatchCodes...),
			MatchSources: append([]string(nil), parseAction.MatchSources...),
			Run:          parseAction.Run,
		}
	}
	return parseCloned
}
