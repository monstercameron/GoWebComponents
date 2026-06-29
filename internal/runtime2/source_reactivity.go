package runtime2

import (
	"sort"
	"strings"
)

// SourceReactivity stores declared source bindings and queued region updates.
type SourceReactivity struct {
	storeSourceIDsByRegionID map[RegionInstanceID][]string
	storeRegionIDsBySourceID map[string]map[RegionInstanceID]struct{}
	storeQueuedRegionIDs     map[RegionInstanceID]struct{}
	storeQueuedRegionOrder   []RegionInstanceID
	hasQueuedRegionSorted    bool
}

// BuildSourceReactivity creates a source-reactivity tracker with empty bindings and queue state.
func BuildSourceReactivity() *SourceReactivity {
	return &SourceReactivity{
		storeSourceIDsByRegionID: make(map[RegionInstanceID][]string),
		storeRegionIDsBySourceID: make(map[string]map[RegionInstanceID]struct{}),
		storeQueuedRegionIDs:     make(map[RegionInstanceID]struct{}),
		storeQueuedRegionOrder:   make([]RegionInstanceID, 0),
		hasQueuedRegionSorted:    true,
	}
}

// SetRegionDeclaredSources sets or replaces declared source IDs for one region instance.
func (parseSourceReactivity *SourceReactivity) SetRegionDeclaredSources(parseRegionInstanceID RegionInstanceID, parseSourceIDs []string) error {
	if parseSourceReactivity == nil {
		return nil
	}
	getRegionInstanceID, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID))
	if parseErr != nil {
		return parseErr
	}
	getSourceIDs, parseErr := NormalizeSourceIDs(parseSourceIDs)
	if parseErr != nil {
		return parseErr
	}
	if hasSourceReactivityExactSourceIDs(parseSourceReactivity.storeSourceIDsByRegionID[getRegionInstanceID], getSourceIDs) {
		return nil
	}
	parseSourceReactivity.clearRegionDeclaredSources(getRegionInstanceID)
	if len(getSourceIDs) == 0 {
		delete(parseSourceReactivity.storeSourceIDsByRegionID, getRegionInstanceID)
		return nil
	}
	parseSourceReactivity.storeSourceIDsByRegionID[getRegionInstanceID] = append([]string(nil), getSourceIDs...)
	for _, getSourceID := range getSourceIDs {
		getRegionIDs, hasSourceID := parseSourceReactivity.storeRegionIDsBySourceID[getSourceID]
		if !hasSourceID {
			getRegionIDs = make(map[RegionInstanceID]struct{})
			parseSourceReactivity.storeRegionIDsBySourceID[getSourceID] = getRegionIDs
		}
		getRegionIDs[getRegionInstanceID] = struct{}{}
	}
	return nil
}

// HandleSourceChange enqueues updates for regions that declared one changed source ID.
func (parseSourceReactivity *SourceReactivity) HandleSourceChange(parseSourceID string) bool {
	if parseSourceReactivity == nil {
		return false
	}
	getSourceID := strings.TrimSpace(parseSourceID)
	if getSourceID == "" || getSourceID != parseSourceID {
		return false
	}
	getRegionIDs, hasSourceID := parseSourceReactivity.storeRegionIDsBySourceID[getSourceID]
	if !hasSourceID {
		return false
	}
	for getRegionInstanceID := range getRegionIDs {
		parseSourceReactivity.storeSourceReactivityQueueRegionUpdate(getRegionInstanceID)
	}
	return true
}

// GetRegionUpdateQueue returns coalesced region updates in stable order and clears queued state.
func (parseSourceReactivity *SourceReactivity) GetRegionUpdateQueue() []RegionInstanceID {
	if parseSourceReactivity == nil {
		return nil
	}
	if len(parseSourceReactivity.storeQueuedRegionOrder) == 0 {
		return nil
	}
	if !parseSourceReactivity.hasQueuedRegionSorted && len(parseSourceReactivity.storeQueuedRegionOrder) > 1 {
		sort.Slice(parseSourceReactivity.storeQueuedRegionOrder, func(parseLeftIndex int, parseRightIndex int) bool {
			return string(parseSourceReactivity.storeQueuedRegionOrder[parseLeftIndex]) < string(parseSourceReactivity.storeQueuedRegionOrder[parseRightIndex])
		})
	}
	getRegionUpdates := append([]RegionInstanceID(nil), parseSourceReactivity.storeQueuedRegionOrder...)
	parseSourceReactivity.storeQueuedRegionOrder = parseSourceReactivity.storeQueuedRegionOrder[:0]
	parseSourceReactivity.hasQueuedRegionSorted = true
	clear(parseSourceReactivity.storeQueuedRegionIDs)
	return getRegionUpdates
}

// hasSourceReactivityExactSourceIDs reports whether two normalized source-ID slices already match exactly.
func hasSourceReactivityExactSourceIDs(parseCurrent []string, parseNext []string) bool {
	if len(parseCurrent) != len(parseNext) {
		return false
	}
	for parseIndex := range parseCurrent {
		if parseCurrent[parseIndex] != parseNext[parseIndex] {
			return false
		}
	}
	return true
}

// storeSourceReactivityQueueRegionUpdate enqueues one region update once and marks queue sort state lazily.
func (parseSourceReactivity *SourceReactivity) storeSourceReactivityQueueRegionUpdate(parseRegionInstanceID RegionInstanceID) {
	if parseSourceReactivity == nil {
		return
	}
	if _, hasRegionID := parseSourceReactivity.storeQueuedRegionIDs[parseRegionInstanceID]; hasRegionID {
		return
	}
	getQueuedRegionOrderCount := len(parseSourceReactivity.storeQueuedRegionOrder)
	if getQueuedRegionOrderCount > 0 {
		getQueuedRegionLast := parseSourceReactivity.storeQueuedRegionOrder[getQueuedRegionOrderCount-1]
		if string(getQueuedRegionLast) > string(parseRegionInstanceID) {
			parseSourceReactivity.hasQueuedRegionSorted = false
		}
	}
	parseSourceReactivity.storeQueuedRegionOrder = append(parseSourceReactivity.storeQueuedRegionOrder, parseRegionInstanceID)
	parseSourceReactivity.storeQueuedRegionIDs[parseRegionInstanceID] = struct{}{}
}

// clearRegionDeclaredSources removes reverse source bindings for one region.
func (parseSourceReactivity *SourceReactivity) clearRegionDeclaredSources(parseRegionInstanceID RegionInstanceID) {
	getSourceIDs, hasRegionSourceIDs := parseSourceReactivity.storeSourceIDsByRegionID[parseRegionInstanceID]
	if !hasRegionSourceIDs {
		return
	}
	for _, getSourceID := range getSourceIDs {
		getRegionIDs, hasSourceID := parseSourceReactivity.storeRegionIDsBySourceID[getSourceID]
		if !hasSourceID {
			continue
		}
		delete(getRegionIDs, parseRegionInstanceID)
		if len(getRegionIDs) == 0 {
			delete(parseSourceReactivity.storeRegionIDsBySourceID, getSourceID)
		}
	}
	delete(parseSourceReactivity.storeSourceIDsByRegionID, parseRegionInstanceID)
}
