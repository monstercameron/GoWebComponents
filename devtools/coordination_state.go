package devtools

import "sync"

var coordinationInspection struct {
	mu    sync.RWMutex
	state Coordination
}

// SetCoordinationInspection stores app-owned worker, sync, and replay inspection state.
func SetCoordinationInspection(state Coordination) {
	coordinationInspection.mu.Lock()
	defer coordinationInspection.mu.Unlock()
	coordinationInspection.state = cloneCoordination(state)
}

// ResetCoordinationInspection clears the app-owned coordination inspection state.
func ResetCoordinationInspection() {
	SetCoordinationInspection(Coordination{})
}

// InspectCoordination returns the current app-owned coordination inspection state.
func InspectCoordination() Coordination {
	coordinationInspection.mu.RLock()
	defer coordinationInspection.mu.RUnlock()
	return cloneCoordination(coordinationInspection.state)
}

func cloneCoordination(state Coordination) Coordination {
	cloned := Coordination{}
	if len(state.Workers) > 0 {
		cloned.Workers = append([]WorkerJob(nil), state.Workers...)
	}
	if len(state.SyncEvents) > 0 {
		cloned.SyncEvents = append([]SyncEvent(nil), state.SyncEvents...)
	}
	if len(state.Replay) > 0 {
		cloned.Replay = append([]ReplayEntry(nil), state.Replay...)
	}
	return cloned
}
