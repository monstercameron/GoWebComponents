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
}

// BuildSourceReactivity creates a source-reactivity tracker with empty bindings and queue state.
func BuildSourceReactivity() *SourceReactivity {
	return &SourceReactivity{
		storeSourceIDsByRegionID: make(map[RegionInstanceID][]string),
		storeRegionIDsBySourceID: make(map[string]map[RegionInstanceID]struct{}),
		storeQueuedRegionIDs:     make(map[RegionInstanceID]struct{}),
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
		parseSourceReactivity.storeQueuedRegionIDs[getRegionInstanceID] = struct{}{}
	}
	return true
}

// GetRegionUpdateQueue returns coalesced region updates in stable order and clears queued state.
func (parseSourceReactivity *SourceReactivity) GetRegionUpdateQueue() []RegionInstanceID {
	if parseSourceReactivity == nil {
		return nil
	}
	getRegionUpdates := make([]RegionInstanceID, 0, len(parseSourceReactivity.storeQueuedRegionIDs))
	for getRegionInstanceID := range parseSourceReactivity.storeQueuedRegionIDs {
		getRegionUpdates = append(getRegionUpdates, getRegionInstanceID)
	}
	sort.Slice(getRegionUpdates, func(parseLeftIndex int, parseRightIndex int) bool {
		return string(getRegionUpdates[parseLeftIndex]) < string(getRegionUpdates[parseRightIndex])
	})
	parseSourceReactivity.storeQueuedRegionIDs = make(map[RegionInstanceID]struct{})
	return getRegionUpdates
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
