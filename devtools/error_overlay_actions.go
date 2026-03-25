package devtools

import "sync"

var errorOverlayActions struct {
	mu      sync.RWMutex
	actions []ErrorOverlayAction
}

// SetErrorOverlayActions replaces the current app-owned recovery actions for the error overlay.
func SetErrorOverlayActions(actions []ErrorOverlayAction) {
	errorOverlayActions.mu.Lock()
	defer errorOverlayActions.mu.Unlock()
	errorOverlayActions.actions = cloneErrorOverlayActions(actions)
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

func cloneErrorOverlayActions(actions []ErrorOverlayAction) []ErrorOverlayAction {
	if len(actions) == 0 {
		return nil
	}
	cloned := make([]ErrorOverlayAction, len(actions))
	for i, action := range actions {
		cloned[i] = ErrorOverlayAction{
			Label:        action.Label,
			MatchCodes:   append([]string(nil), action.MatchCodes...),
			MatchSources: append([]string(nil), action.MatchSources...),
			Run:          action.Run,
		}
	}
	return cloned
}
