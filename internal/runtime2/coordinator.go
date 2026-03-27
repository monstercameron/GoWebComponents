package runtime2

import (
	"fmt"
	"sync"
)

// Coordinator scopes future multithreaded runtime state away from the shipped runtime singleton.
type Coordinator struct {
	storeMu      sync.RWMutex
	storeEntries map[RegionInstanceID]*CoordinatorEntry
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
	return &Coordinator{storeEntries: map[RegionInstanceID]*CoordinatorEntry{}}
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
	parseEntry, parseHasEntry := parseCoordinator.storeEntries[parseRegionInstanceID]
	parseCoordinator.storeMu.RUnlock()
	if !parseHasEntry || parseEntry == nil {
		return CoordinatorEntry{}, false
	}
	return *parseEntry, true
}

// GetEntrySnapshotFields reads only the Epoch, RendererID, and SourceIDs from one coordinator entry.
// It avoids copying the full CoordinatorEntry value and is optimized for the hot update-snapshot path.
func (parseCoordinator *Coordinator) GetEntrySnapshotFields(parseRegionInstanceID RegionInstanceID) (getEpoch uint64, getRendererID RendererID, getSourceIDs []string, ok bool) {
	if parseCoordinator == nil {
		return 0, "", nil, false
	}
	parseCoordinator.storeMu.RLock()
	parseEntry, parseHasEntry := parseCoordinator.storeEntries[parseRegionInstanceID]
	if !parseHasEntry || parseEntry == nil {
		parseCoordinator.storeMu.RUnlock()
		return 0, "", nil, false
	}
	getEpoch = parseEntry.Epoch
	getRendererID = parseEntry.RendererID
	getSourceIDs = parseEntry.SourceIDs
	parseCoordinator.storeMu.RUnlock()
	return getEpoch, getRendererID, getSourceIDs, true
}

// GetEntryDispatchValidation reads only the IsFallback, LastSnapshotVersion, and LastDispatchedVersion from one coordinator entry.
// It avoids copying the full CoordinatorEntry value and is optimized for the hot dispatch-validation path.
func (parseCoordinator *Coordinator) GetEntryDispatchValidation(parseRegionInstanceID RegionInstanceID) (isFallback bool, lastSnapshotVersion uint64, lastDispatchedVersion uint64, ok bool) {
	if parseCoordinator == nil {
		return false, 0, 0, false
	}
	parseCoordinator.storeMu.RLock()
	parseEntry, parseHasEntry := parseCoordinator.storeEntries[parseRegionInstanceID]
	if !parseHasEntry || parseEntry == nil {
		parseCoordinator.storeMu.RUnlock()
		return false, 0, 0, false
	}
	isFallback = parseEntry.IsFallback
	lastSnapshotVersion = parseEntry.LastSnapshotVersion
	lastDispatchedVersion = parseEntry.LastDispatchedVersion
	parseCoordinator.storeMu.RUnlock()
	return isFallback, lastSnapshotVersion, lastDispatchedVersion, true
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
	parseCoordinator.storeEntries[parseEntry.RegionInstanceID] = &parseEntry
	return nil
}

// UpdateRegion records a dispatched region update.
func (parseCoordinator *Coordinator) UpdateRegion(parseRegionInstanceID RegionInstanceID, parseInputVersion uint64) error {
	if parseCoordinator == nil {
		return fmt.Errorf("runtime2: coordinator is required")
	}
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return parseErr
	}
	parseCoordinator.storeMu.Lock()
	parseEntry, parseEntryErr := parseCoordinator.getMutableCoordinatorEntryLocked(parseRegionInstanceID)
	if parseEntryErr != nil {
		parseCoordinator.storeMu.Unlock()
		return parseEntryErr
	}
	if parseEntry.IsFallback {
		parseCoordinator.storeMu.Unlock()
		return fmt.Errorf("runtime2: coordinator entry %q is in fallback mode", parseRegionInstanceID)
	}
	if parseErr := ValidateMonotonicInputVersion(parseEntry.LastDispatchedVersion, parseInputVersion); parseErr != nil {
		parseCoordinator.storeMu.Unlock()
		return parseErr
	}
	parseEntry.LastDispatchedVersion = parseInputVersion
	parseEntry.CurrentState = CoordinatorStateActive
	parseCoordinator.storeMu.Unlock()
	return nil
}

// UpdateRegionAndGetEntry records a dispatched region update and returns the updated entry.
func (parseCoordinator *Coordinator) UpdateRegionAndGetEntry(parseRegionInstanceID RegionInstanceID, parseInputVersion uint64) (CoordinatorEntry, error) {
	if parseCoordinator == nil {
		return CoordinatorEntry{}, fmt.Errorf("runtime2: coordinator is required")
	}
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return CoordinatorEntry{}, parseErr
	}
	parseCoordinator.storeMu.Lock()
	parseEntry, parseEntryErr := parseCoordinator.getMutableCoordinatorEntryLocked(parseRegionInstanceID)
	if parseEntryErr != nil {
		parseCoordinator.storeMu.Unlock()
		return CoordinatorEntry{}, parseEntryErr
	}
	if parseEntry.IsFallback {
		parseCoordinator.storeMu.Unlock()
		return CoordinatorEntry{}, fmt.Errorf("runtime2: coordinator entry %q is in fallback mode", parseRegionInstanceID)
	}
	if parseErr := ValidateMonotonicInputVersion(parseEntry.LastDispatchedVersion, parseInputVersion); parseErr != nil {
		parseCoordinator.storeMu.Unlock()
		return CoordinatorEntry{}, parseErr
	}
	parseEntry.LastDispatchedVersion = parseInputVersion
	parseEntry.CurrentState = CoordinatorStateActive
	parseCoordinator.storeMu.Unlock()
	return *parseEntry, nil
}

// getMutableCoordinatorEntryLocked loads one mutable coordinator entry pointer while the write lock is held.
func (parseCoordinator *Coordinator) getMutableCoordinatorEntryLocked(parseRegionInstanceID RegionInstanceID) (*CoordinatorEntry, error) {
	parseEntry, parseHasEntry := parseCoordinator.storeEntries[parseRegionInstanceID]
	if !parseHasEntry || parseEntry == nil {
		return nil, fmt.Errorf("runtime2: coordinator entry %q is not mounted", parseRegionInstanceID)
	}
	return parseEntry, nil
}

// StoreRegionSnapshotAndDispatchedVersion stores monotonic snapshot and dispatched versions in one transaction and returns the updated entry.
func (parseCoordinator *Coordinator) StoreRegionSnapshotAndDispatchedVersion(
	parseRegionInstanceID RegionInstanceID,
	parseSnapshotVersion uint64,
	parseDispatchedVersion uint64,
) (CoordinatorEntry, error) {
	return parseCoordinator.StoreRegionSnapshotDispatchState(
		parseRegionInstanceID,
		parseSnapshotVersion,
		parseDispatchedVersion,
		nil,
		false,
	)
}

// StoreRegionSnapshotDispatchState stores monotonic snapshot and dispatched versions and optionally canonical source IDs in one transaction.
func (parseCoordinator *Coordinator) StoreRegionSnapshotDispatchState(
	parseRegionInstanceID RegionInstanceID,
	parseSnapshotVersion uint64,
	parseDispatchedVersion uint64,
	parseSourceIDs []string,
	parseShouldStoreSourceIDs bool,
) (CoordinatorEntry, error) {
	if parseCoordinator == nil {
		return CoordinatorEntry{}, fmt.Errorf("runtime2: coordinator is required")
	}
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return CoordinatorEntry{}, parseErr
	}
	if parseSnapshotVersion == 0 {
		return CoordinatorEntry{}, fmt.Errorf("runtime2: snapshot version is required")
	}
	if parseDispatchedVersion == 0 {
		return CoordinatorEntry{}, fmt.Errorf("runtime2: dispatched version is required")
	}
	parseNormalizedSourceIDs := []string(nil)
	if parseShouldStoreSourceIDs {
		getNormalizedSourceIDs, parseNormalizeErr := NormalizeSourceIDs(parseSourceIDs)
		if parseNormalizeErr != nil {
			return CoordinatorEntry{}, parseNormalizeErr
		}
		parseNormalizedSourceIDs = getNormalizedSourceIDs
	}
	parseCoordinator.storeMu.Lock()
	defer parseCoordinator.storeMu.Unlock()
	parseEntry, parseEntryErr := parseCoordinator.getMutableCoordinatorEntryLocked(parseRegionInstanceID)
	if parseEntryErr != nil {
		return CoordinatorEntry{}, parseEntryErr
	}
	if parseEntry.IsFallback {
		return CoordinatorEntry{}, fmt.Errorf("runtime2: coordinator entry %q is in fallback mode", parseRegionInstanceID)
	}
	if parseErr := ValidateMonotonicInputVersion(parseEntry.LastSnapshotVersion, parseSnapshotVersion); parseErr != nil {
		return CoordinatorEntry{}, parseErr
	}
	if parseErr := ValidateMonotonicInputVersion(parseEntry.LastDispatchedVersion, parseDispatchedVersion); parseErr != nil {
		return CoordinatorEntry{}, parseErr
	}
	parseEntry.LastSnapshotVersion = parseSnapshotVersion
	parseEntry.LastDispatchedVersion = parseDispatchedVersion
	if parseShouldStoreSourceIDs {
		parseEntry.SourceIDs = parseNormalizedSourceIDs
	}
	parseEntry.CurrentState = CoordinatorStateActive
	return *parseEntry, nil
}

// CommitRegion records a committed region patch version.
func (parseCoordinator *Coordinator) CommitRegion(parseRegionInstanceID RegionInstanceID, parseCommittedVersion uint64) error {
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return parseErr
	}
	return parseCoordinator.handleCoordinatorCommitRegionTrusted(parseRegionInstanceID, parseCommittedVersion)
}

// CancelRegion marks one region's in-flight work as canceled.
func (parseCoordinator *Coordinator) CancelRegion(parseRegionInstanceID RegionInstanceID) error {
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return parseErr
	}
	parseEntry, parseErr := parseCoordinator.getMutableEntryTrusted(parseRegionInstanceID)
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
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return parseErr
	}
	return parseCoordinator.handleCoordinatorFallbackRegionTrusted(parseRegionInstanceID)
}

// RestartRegion bumps one region's epoch and clears version state for a fresh mount cycle.
func (parseCoordinator *Coordinator) RestartRegion(parseRegionInstanceID RegionInstanceID, parseEpoch uint64) error {
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return parseErr
	}
	return parseCoordinator.handleCoordinatorRestartRegionTrusted(parseRegionInstanceID, parseEpoch)
}

// SetRegionAttached stores one coordinator attached-state transition for a mounted region.
func (parseCoordinator *Coordinator) SetRegionAttached(parseRegionInstanceID RegionInstanceID, parseIsAttached bool) error {
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return parseErr
	}
	return parseCoordinator.handleCoordinatorSetRegionAttachedTrusted(parseRegionInstanceID, parseIsAttached)
}

// SetRegionSourceIDs stores one canonical declared-source set for a mounted region.
func (parseCoordinator *Coordinator) SetRegionSourceIDs(parseRegionInstanceID RegionInstanceID, parseSourceIDs []string) error {
	if parseCoordinator == nil {
		return fmt.Errorf("runtime2: coordinator is required")
	}
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return parseErr
	}
	parseNormalizedSourceIDs, parseNormalizeErr := NormalizeSourceIDs(parseSourceIDs)
	if parseNormalizeErr != nil {
		return parseNormalizeErr
	}
	parseCoordinator.storeMu.Lock()
	defer parseCoordinator.storeMu.Unlock()
	parseEntry, parseEntryErr := parseCoordinator.getMutableCoordinatorEntryLocked(parseRegionInstanceID)
	if parseEntryErr != nil {
		return parseEntryErr
	}
	if parseEntry == nil {
		return fmt.Errorf("runtime2: coordinator entry %q is not mounted", parseRegionInstanceID)
	}
	parseEntry.SourceIDs = parseNormalizedSourceIDs
	return nil
}

// SetRegionSnapshotState stores one monotonic snapshot version and optionally canonical source IDs in one transaction.
func (parseCoordinator *Coordinator) SetRegionSnapshotState(
	parseRegionInstanceID RegionInstanceID,
	parseSnapshotVersion uint64,
	parseSourceIDs []string,
	parseShouldStoreSourceIDs bool,
) (CoordinatorEntry, error) {
	if parseCoordinator == nil {
		return CoordinatorEntry{}, fmt.Errorf("runtime2: coordinator is required")
	}
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return CoordinatorEntry{}, parseErr
	}
	if parseSnapshotVersion == 0 {
		return CoordinatorEntry{}, fmt.Errorf("runtime2: snapshot version is required")
	}
	parseNormalizedSourceIDs := []string(nil)
	if parseShouldStoreSourceIDs {
		getNormalizedSourceIDs, parseNormalizeErr := NormalizeSourceIDs(parseSourceIDs)
		if parseNormalizeErr != nil {
			return CoordinatorEntry{}, parseNormalizeErr
		}
		parseNormalizedSourceIDs = getNormalizedSourceIDs
	}
	parseCoordinator.storeMu.Lock()
	defer parseCoordinator.storeMu.Unlock()
	parseEntry, parseEntryErr := parseCoordinator.getMutableCoordinatorEntryLocked(parseRegionInstanceID)
	if parseEntryErr != nil {
		return CoordinatorEntry{}, parseEntryErr
	}
	if parseVersionErr := ValidateMonotonicInputVersion(parseEntry.LastSnapshotVersion, parseSnapshotVersion); parseVersionErr != nil {
		return CoordinatorEntry{}, parseVersionErr
	}
	parseEntry.LastSnapshotVersion = parseSnapshotVersion
	if parseShouldStoreSourceIDs {
		parseEntry.SourceIDs = parseNormalizedSourceIDs
	}
	return *parseEntry, nil
}

// SetRegionLastSnapshotVersion stores one monotonic snapshot version for a mounted region.
func (parseCoordinator *Coordinator) SetRegionLastSnapshotVersion(parseRegionInstanceID RegionInstanceID, parseSnapshotVersion uint64) error {
	_, parseErr := parseCoordinator.SetRegionSnapshotState(parseRegionInstanceID, parseSnapshotVersion, nil, false)
	return parseErr
}

// applyRegionSnapshotState stores one monotonic snapshot version and optionally canonical source IDs in one transaction.
// Unlike SetRegionSnapshotState it returns only an error to eliminate the CoordinatorEntry copy for callers that discard the entry.
func (parseCoordinator *Coordinator) applyRegionSnapshotState(
	parseRegionInstanceID RegionInstanceID,
	parseSnapshotVersion uint64,
	parseSourceIDs []string,
	parseShouldStoreSourceIDs bool,
) error {
	if parseCoordinator == nil {
		return fmt.Errorf("runtime2: coordinator is required")
	}
	if parseSnapshotVersion == 0 {
		return fmt.Errorf("runtime2: snapshot version is required")
	}
	parseNormalizedSourceIDs := []string(nil)
	if parseShouldStoreSourceIDs {
		getNormalizedSourceIDs, parseNormalizeErr := NormalizeSourceIDs(parseSourceIDs)
		if parseNormalizeErr != nil {
			return parseNormalizeErr
		}
		parseNormalizedSourceIDs = getNormalizedSourceIDs
	}
	parseCoordinator.storeMu.Lock()
	defer parseCoordinator.storeMu.Unlock()
	parseEntry, parseEntryErr := parseCoordinator.getMutableCoordinatorEntryLocked(parseRegionInstanceID)
	if parseEntryErr != nil {
		return parseEntryErr
	}
	if parseVersionErr := ValidateMonotonicInputVersion(parseEntry.LastSnapshotVersion, parseSnapshotVersion); parseVersionErr != nil {
		return parseVersionErr
	}
	parseEntry.LastSnapshotVersion = parseSnapshotVersion
	if parseShouldStoreSourceIDs {
		parseEntry.SourceIDs = parseNormalizedSourceIDs
	}
	return nil
}

// applyRegionSnapshotDispatchState stores monotonic snapshot and dispatched versions and optionally canonical source IDs in one transaction.
// Unlike StoreRegionSnapshotDispatchState it returns only an error to eliminate the CoordinatorEntry copy for callers that discard the entry.
func (parseCoordinator *Coordinator) applyRegionSnapshotDispatchState(
	parseRegionInstanceID RegionInstanceID,
	parseSnapshotVersion uint64,
	parseDispatchedVersion uint64,
	parseSourceIDs []string,
	parseShouldStoreSourceIDs bool,
) error {
	if parseCoordinator == nil {
		return fmt.Errorf("runtime2: coordinator is required")
	}
	if parseSnapshotVersion == 0 {
		return fmt.Errorf("runtime2: snapshot version is required")
	}
	if parseDispatchedVersion == 0 {
		return fmt.Errorf("runtime2: dispatched version is required")
	}
	parseNormalizedSourceIDs := []string(nil)
	if parseShouldStoreSourceIDs {
		getNormalizedSourceIDs, parseNormalizeErr := NormalizeSourceIDs(parseSourceIDs)
		if parseNormalizeErr != nil {
			return parseNormalizeErr
		}
		parseNormalizedSourceIDs = getNormalizedSourceIDs
	}
	parseCoordinator.storeMu.Lock()
	defer parseCoordinator.storeMu.Unlock()
	parseEntry, parseEntryErr := parseCoordinator.getMutableCoordinatorEntryLocked(parseRegionInstanceID)
	if parseEntryErr != nil {
		return parseEntryErr
	}
	if parseEntry.IsFallback {
		return fmt.Errorf("runtime2: coordinator entry %q is in fallback mode", parseRegionInstanceID)
	}
	if parseVersionErr := ValidateMonotonicInputVersion(parseEntry.LastSnapshotVersion, parseSnapshotVersion); parseVersionErr != nil {
		return parseVersionErr
	}
	if parseVersionErr := ValidateMonotonicInputVersion(parseEntry.LastDispatchedVersion, parseDispatchedVersion); parseVersionErr != nil {
		return parseVersionErr
	}
	parseEntry.LastSnapshotVersion = parseSnapshotVersion
	parseEntry.LastDispatchedVersion = parseDispatchedVersion
	if parseShouldStoreSourceIDs {
		parseEntry.SourceIDs = parseNormalizedSourceIDs
	}
	parseEntry.CurrentState = CoordinatorStateActive
	return nil
}

// IncrementRegionDroppedStalePatchCount increments one region's stale patch-drop counter.
func (parseCoordinator *Coordinator) IncrementRegionDroppedStalePatchCount(parseRegionInstanceID RegionInstanceID) (uint64, error) {
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return 0, parseErr
	}
	return parseCoordinator.handleCoordinatorIncrementRegionDroppedStalePatchCountTrusted(parseRegionInstanceID)
}

// IncrementRegionIgnoredStaleDiagnosticCount increments one region's stale-diagnostic ignore counter.
func (parseCoordinator *Coordinator) IncrementRegionIgnoredStaleDiagnosticCount(parseRegionInstanceID RegionInstanceID) (uint64, error) {
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return 0, parseErr
	}
	return parseCoordinator.handleCoordinatorIncrementRegionIgnoredStaleDiagnosticCountTrusted(parseRegionInstanceID)
}

// IncrementRegionRepairRemountCount increments one region's successful repair-remount counter.
func (parseCoordinator *Coordinator) IncrementRegionRepairRemountCount(parseRegionInstanceID RegionInstanceID) (uint64, error) {
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return 0, parseErr
	}
	return parseCoordinator.handleCoordinatorIncrementRegionRepairRemountCountTrusted(parseRegionInstanceID)
}

// handleCoordinatorCommitRegionTrusted records one committed region patch version for one trusted region identifier.
func (parseCoordinator *Coordinator) handleCoordinatorCommitRegionTrusted(parseRegionInstanceID RegionInstanceID, parseCommittedVersion uint64) error {
	parseEntry, parseErr := parseCoordinator.getMutableEntryTrusted(parseRegionInstanceID)
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

// handleCoordinatorFallbackRegionTrusted marks one region as fallback state for one trusted region identifier.
func (parseCoordinator *Coordinator) handleCoordinatorFallbackRegionTrusted(parseRegionInstanceID RegionInstanceID) error {
	parseEntry, parseErr := parseCoordinator.getMutableEntryTrusted(parseRegionInstanceID)
	if parseErr != nil {
		return parseErr
	}
	parseEntry.IsAttached = false
	parseEntry.IsFallback = true
	parseEntry.CurrentState = CoordinatorStateFallback
	return parseCoordinator.storeMutableEntry(parseEntry)
}

// handleCoordinatorRestartRegionTrusted bumps one region epoch and clears version state for one trusted region identifier.
func (parseCoordinator *Coordinator) handleCoordinatorRestartRegionTrusted(parseRegionInstanceID RegionInstanceID, parseEpoch uint64) error {
	parseEntry, parseErr := parseCoordinator.getMutableEntryTrusted(parseRegionInstanceID)
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

// handleCoordinatorSetRegionAttachedTrusted stores one attached-state transition for one trusted region identifier.
func (parseCoordinator *Coordinator) handleCoordinatorSetRegionAttachedTrusted(parseRegionInstanceID RegionInstanceID, parseIsAttached bool) error {
	parseEntry, parseErr := parseCoordinator.getMutableEntryTrusted(parseRegionInstanceID)
	if parseErr != nil {
		return parseErr
	}
	parseEntry.IsAttached = parseIsAttached
	return parseCoordinator.storeMutableEntry(parseEntry)
}

// handleCoordinatorIncrementRegionDroppedStalePatchCountTrusted increments one stale patch-drop counter for one trusted region identifier.
func (parseCoordinator *Coordinator) handleCoordinatorIncrementRegionDroppedStalePatchCountTrusted(parseRegionInstanceID RegionInstanceID) (uint64, error) {
	parseEntry, parseErr := parseCoordinator.getMutableEntryTrusted(parseRegionInstanceID)
	if parseErr != nil {
		return 0, parseErr
	}
	parseEntry.DroppedStalePatchCount++
	if parseStoreErr := parseCoordinator.storeMutableEntry(parseEntry); parseStoreErr != nil {
		return 0, parseStoreErr
	}
	return parseEntry.DroppedStalePatchCount, nil
}

// handleCoordinatorIncrementRegionIgnoredStaleDiagnosticCountTrusted increments one stale-diagnostic ignore counter for one trusted region identifier.
func (parseCoordinator *Coordinator) handleCoordinatorIncrementRegionIgnoredStaleDiagnosticCountTrusted(parseRegionInstanceID RegionInstanceID) (uint64, error) {
	parseEntry, parseErr := parseCoordinator.getMutableEntryTrusted(parseRegionInstanceID)
	if parseErr != nil {
		return 0, parseErr
	}
	parseEntry.IgnoredStaleDiagnosticCount++
	if parseStoreErr := parseCoordinator.storeMutableEntry(parseEntry); parseStoreErr != nil {
		return 0, parseStoreErr
	}
	return parseEntry.IgnoredStaleDiagnosticCount, nil
}

// handleCoordinatorIncrementRegionRepairRemountCountTrusted increments one repair-remount counter for one trusted region identifier.
func (parseCoordinator *Coordinator) handleCoordinatorIncrementRegionRepairRemountCountTrusted(parseRegionInstanceID RegionInstanceID) (uint64, error) {
	parseEntry, parseErr := parseCoordinator.getMutableEntryTrusted(parseRegionInstanceID)
	if parseErr != nil {
		return 0, parseErr
	}
	parseEntry.RepairRemountCount++
	if parseStoreErr := parseCoordinator.storeMutableEntry(parseEntry); parseStoreErr != nil {
		return 0, parseStoreErr
	}
	return parseEntry.RepairRemountCount, nil
}

// getMutableEntry loads one mutable coordinator entry after validating the region identifier.
func (parseCoordinator *Coordinator) getMutableEntry(parseRegionInstanceID RegionInstanceID) (CoordinatorEntry, error) {
	if parseCoordinator == nil {
		return CoordinatorEntry{}, fmt.Errorf("runtime2: coordinator is required")
	}
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return CoordinatorEntry{}, parseErr
	}
	return parseCoordinator.getMutableEntryTrusted(parseRegionInstanceID)
}

// getMutableEntryTrusted loads one mutable coordinator entry for one trusted region identifier.
func (parseCoordinator *Coordinator) getMutableEntryTrusted(parseRegionInstanceID RegionInstanceID) (CoordinatorEntry, error) {
	if parseCoordinator == nil {
		return CoordinatorEntry{}, fmt.Errorf("runtime2: coordinator is required")
	}
	parseCoordinator.storeMu.RLock()
	defer parseCoordinator.storeMu.RUnlock()
	parseEntry, parseHasEntry := parseCoordinator.storeEntries[parseRegionInstanceID]
	if !parseHasEntry || parseEntry == nil {
		return CoordinatorEntry{}, fmt.Errorf("runtime2: coordinator entry %q is not mounted", parseRegionInstanceID)
	}
	return *parseEntry, nil
}

// storeMutableEntry writes one mutated coordinator entry back into the coordinator store.
func (parseCoordinator *Coordinator) storeMutableEntry(parseEntry CoordinatorEntry) error {
	if parseCoordinator == nil {
		return fmt.Errorf("runtime2: coordinator is required")
	}
	parseCoordinator.storeMu.Lock()
	defer parseCoordinator.storeMu.Unlock()
	parseMutableEntry, parseHasEntry := parseCoordinator.storeEntries[parseEntry.RegionInstanceID]
	if parseHasEntry && parseMutableEntry != nil {
		*parseMutableEntry = parseEntry
		return nil
	}
	parseEntryCopy := parseEntry
	parseCoordinator.storeEntries[parseEntry.RegionInstanceID] = &parseEntryCopy
	return nil
}
