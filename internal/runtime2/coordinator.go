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
	RegionInstanceID            RegionInstanceID
	RendererID                  RendererID
	SourceIDs                   []string
	Epoch                       uint64
	AssignedWorkerShard         string
	CurrentState                CoordinatorState
	LastSnapshotVersion         uint64
	LastDispatchedVersion       uint64
	LastCommittedVersion        uint64
	DroppedStalePatchCount      uint64
	IgnoredStaleDiagnosticCount uint64
	RepairRemountCount          uint64
	IsAttached                  bool
	IsFallback                  bool
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
	parseSourceIDs, parseSourceIDsErr := NormalizeSourceIDs(parseEntry.SourceIDs)
	if parseSourceIDsErr != nil {
		return parseSourceIDsErr
	}
	parseEntry.SourceIDs = parseSourceIDs
	if parseEntry.Epoch == 0 {
		return fmt.Errorf("runtime2: coordinator epoch is required")
	}
	parseCoordinator.storeMu.Lock()
	defer parseCoordinator.storeMu.Unlock()
	if _, parseHasEntry := parseCoordinator.storeEntries[parseEntry.RegionInstanceID]; parseHasEntry {
		return fmt.Errorf("runtime2: coordinator entry %q already exists", parseEntry.RegionInstanceID)
	}
	parseEntry.CurrentState = CoordinatorStateMounted
	parseEntry.IsAttached = false
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
	parseEntry.IsAttached = false
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
	parseEntry.LastSnapshotVersion = 0
	parseEntry.DroppedStalePatchCount = 0
	parseEntry.IgnoredStaleDiagnosticCount = 0
	parseEntry.RepairRemountCount = 0
	parseEntry.IsAttached = false
	parseEntry.IsFallback = false
	parseEntry.CurrentState = CoordinatorStateMounted
	return parseCoordinator.storeMutableEntry(parseEntry)
}

// SetRegionAttached stores one coordinator attached-state transition for a mounted region.
func (parseCoordinator *Coordinator) SetRegionAttached(parseRegionInstanceID RegionInstanceID, parseIsAttached bool) error {
	parseEntry, parseErr := parseCoordinator.getMutableEntry(parseRegionInstanceID)
	if parseErr != nil {
		return parseErr
	}
	parseEntry.IsAttached = parseIsAttached
	return parseCoordinator.storeMutableEntry(parseEntry)
}

// SetRegionSourceIDs stores one canonical declared-source set for a mounted region.
func (parseCoordinator *Coordinator) SetRegionSourceIDs(parseRegionInstanceID RegionInstanceID, parseSourceIDs []string) error {
	parseEntry, parseErr := parseCoordinator.getMutableEntry(parseRegionInstanceID)
	if parseErr != nil {
		return parseErr
	}
	parseNormalizedSourceIDs, parseNormalizeErr := NormalizeSourceIDs(parseSourceIDs)
	if parseNormalizeErr != nil {
		return parseNormalizeErr
	}
	parseEntry.SourceIDs = parseNormalizedSourceIDs
	return parseCoordinator.storeMutableEntry(parseEntry)
}

// SetRegionLastSnapshotVersion stores one monotonic snapshot version for a mounted region.
func (parseCoordinator *Coordinator) SetRegionLastSnapshotVersion(parseRegionInstanceID RegionInstanceID, parseSnapshotVersion uint64) error {
	if parseSnapshotVersion == 0 {
		return fmt.Errorf("runtime2: snapshot version is required")
	}
	parseEntry, parseErr := parseCoordinator.getMutableEntry(parseRegionInstanceID)
	if parseErr != nil {
		return parseErr
	}
	if parseVersionErr := ValidateMonotonicInputVersion(parseEntry.LastSnapshotVersion, parseSnapshotVersion); parseVersionErr != nil {
		return parseVersionErr
	}
	parseEntry.LastSnapshotVersion = parseSnapshotVersion
	return parseCoordinator.storeMutableEntry(parseEntry)
}

// IncrementRegionDroppedStalePatchCount increments one region's stale patch-drop counter.
func (parseCoordinator *Coordinator) IncrementRegionDroppedStalePatchCount(parseRegionInstanceID RegionInstanceID) (uint64, error) {
	parseEntry, parseErr := parseCoordinator.getMutableEntry(parseRegionInstanceID)
	if parseErr != nil {
		return 0, parseErr
	}
	parseEntry.DroppedStalePatchCount++
	if parseStoreErr := parseCoordinator.storeMutableEntry(parseEntry); parseStoreErr != nil {
		return 0, parseStoreErr
	}
	return parseEntry.DroppedStalePatchCount, nil
}

// IncrementRegionIgnoredStaleDiagnosticCount increments one region's stale-diagnostic ignore counter.
func (parseCoordinator *Coordinator) IncrementRegionIgnoredStaleDiagnosticCount(parseRegionInstanceID RegionInstanceID) (uint64, error) {
	parseEntry, parseErr := parseCoordinator.getMutableEntry(parseRegionInstanceID)
	if parseErr != nil {
		return 0, parseErr
	}
	parseEntry.IgnoredStaleDiagnosticCount++
	if parseStoreErr := parseCoordinator.storeMutableEntry(parseEntry); parseStoreErr != nil {
		return 0, parseStoreErr
	}
	return parseEntry.IgnoredStaleDiagnosticCount, nil
}

// IncrementRegionRepairRemountCount increments one region's successful repair-remount counter.
func (parseCoordinator *Coordinator) IncrementRegionRepairRemountCount(parseRegionInstanceID RegionInstanceID) (uint64, error) {
	parseEntry, parseErr := parseCoordinator.getMutableEntry(parseRegionInstanceID)
	if parseErr != nil {
		return 0, parseErr
	}
	parseEntry.RepairRemountCount++
	if parseStoreErr := parseCoordinator.storeMutableEntry(parseEntry); parseStoreErr != nil {
		return 0, parseStoreErr
	}
	return parseEntry.RepairRemountCount, nil
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
