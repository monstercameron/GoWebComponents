package runtime2

import (
	"fmt"
	"sync"
)

// Coordinator scopes future multithreaded runtime state away from the shipped runtime singleton.
type Coordinator struct {
	storeMu      sync.RWMutex
	storeEntries map[RegionInstanceID]CoordinatorEntry
}

// Status reports the current coordinator configuration state.
type Status struct {
	IsConfigured bool
}

// CoordinatorState identifies one coordinator lifecycle state.
type CoordinatorState string

const (
	// CoordinatorStateMounted reports a mounted region awaiting or holding worker ownership.
	CoordinatorStateMounted CoordinatorState = "mounted"
	// CoordinatorStateActive reports a region with active worker-backed updates.
	CoordinatorStateActive CoordinatorState = "active"
	// CoordinatorStateCanceled reports a region with canceled in-flight work.
	CoordinatorStateCanceled CoordinatorState = "canceled"
	// CoordinatorStateFallback reports a region that has exited worker-backed mode.
	CoordinatorStateFallback CoordinatorState = "fallback"
)

// CoordinatorEntry stores one live region's coordinator state.
type CoordinatorEntry struct {
	RegionInstanceID      RegionInstanceID
	RendererID            RendererID
	Epoch                 uint64
	AssignedWorkerShard   string
	CurrentState          CoordinatorState
	LastDispatchedVersion uint64
	LastCommittedVersion  uint64
	IsFallback            bool
}

// BuildCoordinator creates an isolated coordinator for the multithreaded runtime package.
func BuildCoordinator() *Coordinator {
	return &Coordinator{storeEntries: map[RegionInstanceID]CoordinatorEntry{}}
}

// GetStatus reports whether the coordinator has been configured.
func (parseCoordinator *Coordinator) GetStatus() Status {
	if parseCoordinator == nil {
		return Status{}
	}
	return Status{}
}

// GetEntry reports the current coordinator entry for one region instance.
func (parseCoordinator *Coordinator) GetEntry(parseRegionInstanceID RegionInstanceID) (CoordinatorEntry, bool) {
	if parseCoordinator == nil {
		return CoordinatorEntry{}, false
	}
	parseCoordinator.storeMu.RLock()
	defer parseCoordinator.storeMu.RUnlock()
	parseEntry, parseHasEntry := parseCoordinator.storeEntries[parseRegionInstanceID]
	return parseEntry, parseHasEntry
}

// MountRegion creates one live coordinator entry for a mounted region.
func (parseCoordinator *Coordinator) MountRegion(parseEntry CoordinatorEntry) error {
	if parseCoordinator == nil {
		return fmt.Errorf("runtime2: coordinator is required")
	}
	if _, parseErr := ParseRegionInstanceID(string(parseEntry.RegionInstanceID)); parseErr != nil {
		return parseErr
	}
	if _, parseErr := ParseRendererID(string(parseEntry.RendererID)); parseErr != nil {
		return parseErr
	}
	if parseEntry.Epoch == 0 {
		return fmt.Errorf("runtime2: coordinator epoch is required")
	}
	parseCoordinator.storeMu.Lock()
	defer parseCoordinator.storeMu.Unlock()
	if _, parseHasEntry := parseCoordinator.storeEntries[parseEntry.RegionInstanceID]; parseHasEntry {
		return fmt.Errorf("runtime2: coordinator entry %q already exists", parseEntry.RegionInstanceID)
	}
	parseEntry.CurrentState = CoordinatorStateMounted
	parseEntry.IsFallback = false
	parseCoordinator.storeEntries[parseEntry.RegionInstanceID] = parseEntry
	return nil
}

// UpdateRegion records a dispatched region update.
func (parseCoordinator *Coordinator) UpdateRegion(parseRegionInstanceID RegionInstanceID, parseInputVersion uint64) error {
	parseEntry, parseErr := parseCoordinator.getMutableEntry(parseRegionInstanceID)
	if parseErr != nil {
		return parseErr
	}
	if parseEntry.IsFallback {
		return fmt.Errorf("runtime2: coordinator entry %q is in fallback mode", parseRegionInstanceID)
	}
	if parseErr := ValidateMonotonicInputVersion(parseEntry.LastDispatchedVersion, parseInputVersion); parseErr != nil {
		return parseErr
	}
	parseEntry.LastDispatchedVersion = parseInputVersion
	parseEntry.CurrentState = CoordinatorStateActive
	return parseCoordinator.storeMutableEntry(parseEntry)
}

// CommitRegion records a committed region patch version.
func (parseCoordinator *Coordinator) CommitRegion(parseRegionInstanceID RegionInstanceID, parseCommittedVersion uint64) error {
	parseEntry, parseErr := parseCoordinator.getMutableEntry(parseRegionInstanceID)
	if parseErr != nil {
		return parseErr
	}
	if parseEntry.IsFallback {
		return fmt.Errorf("runtime2: coordinator entry %q is in fallback mode", parseRegionInstanceID)
	}
	if parseErr := ValidateMonotonicInputVersion(parseEntry.LastCommittedVersion, parseCommittedVersion); parseErr != nil {
		return parseErr
	}
	parseEntry.LastCommittedVersion = parseCommittedVersion
	parseEntry.CurrentState = CoordinatorStateActive
	return parseCoordinator.storeMutableEntry(parseEntry)
}

// CancelRegion marks one region's in-flight work as canceled.
func (parseCoordinator *Coordinator) CancelRegion(parseRegionInstanceID RegionInstanceID) error {
	parseEntry, parseErr := parseCoordinator.getMutableEntry(parseRegionInstanceID)
	if parseErr != nil {
		return parseErr
	}
	parseEntry.CurrentState = CoordinatorStateCanceled
	return parseCoordinator.storeMutableEntry(parseEntry)
}

// DisposeRegion removes one live coordinator entry.
func (parseCoordinator *Coordinator) DisposeRegion(parseRegionInstanceID RegionInstanceID) error {
	if parseCoordinator == nil {
		return fmt.Errorf("runtime2: coordinator is required")
	}
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return parseErr
	}
	parseCoordinator.storeMu.Lock()
	defer parseCoordinator.storeMu.Unlock()
	if _, parseHasEntry := parseCoordinator.storeEntries[parseRegionInstanceID]; !parseHasEntry {
		return fmt.Errorf("runtime2: coordinator entry %q is not mounted", parseRegionInstanceID)
	}
	delete(parseCoordinator.storeEntries, parseRegionInstanceID)
	return nil
}

// FallbackRegion marks one region as locally owned fallback state.
func (parseCoordinator *Coordinator) FallbackRegion(parseRegionInstanceID RegionInstanceID) error {
	parseEntry, parseErr := parseCoordinator.getMutableEntry(parseRegionInstanceID)
	if parseErr != nil {
		return parseErr
	}
	parseEntry.IsFallback = true
	parseEntry.CurrentState = CoordinatorStateFallback
	return parseCoordinator.storeMutableEntry(parseEntry)
}

// RestartRegion bumps one region's epoch and clears version state for a fresh mount cycle.
func (parseCoordinator *Coordinator) RestartRegion(parseRegionInstanceID RegionInstanceID, parseEpoch uint64) error {
	parseEntry, parseErr := parseCoordinator.getMutableEntry(parseRegionInstanceID)
	if parseErr != nil {
		return parseErr
	}
	if parseEpoch == 0 {
		return fmt.Errorf("runtime2: restart epoch is required")
	}
	if parseEpoch <= parseEntry.Epoch {
		return fmt.Errorf("runtime2: restart epoch must advance current=%d next=%d", parseEntry.Epoch, parseEpoch)
	}
	parseEntry.Epoch = parseEpoch
	parseEntry.LastDispatchedVersion = 0
	parseEntry.LastCommittedVersion = 0
	parseEntry.IsFallback = false
	parseEntry.CurrentState = CoordinatorStateMounted
	return parseCoordinator.storeMutableEntry(parseEntry)
}

// getMutableEntry loads one mutable coordinator entry.
func (parseCoordinator *Coordinator) getMutableEntry(parseRegionInstanceID RegionInstanceID) (CoordinatorEntry, error) {
	if parseCoordinator == nil {
		return CoordinatorEntry{}, fmt.Errorf("runtime2: coordinator is required")
	}
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return CoordinatorEntry{}, parseErr
	}
	parseCoordinator.storeMu.RLock()
	defer parseCoordinator.storeMu.RUnlock()
	parseEntry, parseHasEntry := parseCoordinator.storeEntries[parseRegionInstanceID]
	if !parseHasEntry {
		return CoordinatorEntry{}, fmt.Errorf("runtime2: coordinator entry %q is not mounted", parseRegionInstanceID)
	}
	return parseEntry, nil
}

// storeMutableEntry writes one mutated coordinator entry back into the coordinator store.
func (parseCoordinator *Coordinator) storeMutableEntry(parseEntry CoordinatorEntry) error {
	if parseCoordinator == nil {
		return fmt.Errorf("runtime2: coordinator is required")
	}
	parseCoordinator.storeMu.Lock()
	defer parseCoordinator.storeMu.Unlock()
	parseCoordinator.storeEntries[parseEntry.RegionInstanceID] = parseEntry
	return nil
}
