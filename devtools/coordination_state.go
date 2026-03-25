package devtools

import "sync"

var coordinationInspection struct {
	mu    sync.RWMutex
	state Coordination
}

// SetCoordinationInspection stores app-owned worker, sync, and replay inspection state.
func SetCoordinationInspection(parseState Coordination) {
	coordinationInspection.mu.Lock()
	defer coordinationInspection.mu.Unlock()
	coordinationInspection.state = cloneCoordination(parseState)
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

func cloneCoordination(parseState Coordination) Coordination {
	parseCloned := Coordination{}
	if len(parseState.Workers) > 0 {
		parseCloned.Workers = append([]WorkerJob(nil), parseState.Workers...)
	}
	if len(parseState.SyncEvents) > 0 {
		parseCloned.SyncEvents = append([]SyncEvent(nil), parseState.SyncEvents...)
	}
	if len(parseState.Replay) > 0 {
		parseCloned.Replay = append([]ReplayEntry(nil), parseState.Replay...)
	}
	if len(parseState.QueueEntries) > 0 {
		parseCloned.QueueEntries = append([]SyncQueueEntry(nil), parseState.QueueEntries...)
	}
	if len(parseState.SyncHealth) > 0 {
		parseCloned.SyncHealth = append([]SyncHealthEntry(nil), parseState.SyncHealth...)
	}
	parseCloned.Reconnect = parseState.Reconnect
	parseCloned.Conflict = parseState.Conflict
	parseCloned.LastReplayError = parseState.LastReplayError
	return parseCloned
}
