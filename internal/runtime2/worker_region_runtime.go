package runtime2

import (
	"fmt"
	"strings"
)

// WorkerRegionRenderer renders one mounted region snapshot into worker-local render IR.
type WorkerRegionRenderer func(parseMount WorkerRegionMountSpec) (any, error)

// WorkerRegionMountSpec describes one worker-side mount request.
type WorkerRegionMountSpec struct {
	RegionID     string
	RendererID   string
	Epoch        uint64
	InputVersion uint64
	Snapshot     SnapshotEnvelope
	RenderInput  WorkerRenderInput
}

// WorkerRegionState stores one mounted worker region and its cached render output.
type WorkerRegionState struct {
	RegionID     string
	RendererID   string
	Epoch        uint64
	InputVersion uint64
	Snapshot     SnapshotEnvelope
	SourceIDs    []string
	RenderIR     CanonicalRenderIR
}

// WorkerRegionUpdateSpec describes one worker-side update request for an existing mounted region.
type WorkerRegionUpdateSpec struct {
	RegionID     string
	RendererID   string
	Epoch        uint64
	InputVersion uint64
	Snapshot     SnapshotEnvelope
}

// WorkerRegionEventSpec describes one worker-side semantic event dispatch for an existing mounted region.
type WorkerRegionEventSpec struct {
	RegionID     string
	InputVersion uint64
	EventSlot    EventSlotDispatch
}

// WorkerRegionUpdateResult reports whether an update produced a patch-ready or explicit no-op outcome.
type WorkerRegionUpdateResult struct {
	HasPatchReady bool
	IsNoOp        bool
	IsCanceled    bool
	PatchIR       PatchStreamRaw
}

// WorkerRegionEventResult reports whether a semantic event dispatch produced a patch-ready or explicit no-op outcome.
type WorkerRegionEventResult struct {
	HasPatchReady bool
	IsNoOp        bool
	PatchIR       PatchStreamRaw
}

// WorkerRegionCancelSpec describes one worker-side cancel request.
type WorkerRegionCancelSpec struct {
	RegionID     string
	InputVersion uint64
}

// WorkerRegionCancelResult reports cancel-state handling for one region.
type WorkerRegionCancelResult struct {
	HasRegion         bool
	HasCancelRecorded bool
}

// WorkerRegionDisposeResult reports whether dispose cleared any worker-region state.
type WorkerRegionDisposeResult struct {
	HasStateCleared bool
}

// WorkerRegionRestartSpec describes one worker-side restart request with a fresh epoch.
type WorkerRegionRestartSpec struct {
	RegionID string
	Epoch    uint64
}

// WorkerRegionRestartResult reports whether a restart advanced epoch state for one region.
type WorkerRegionRestartResult struct {
	HasEpochAdvanced bool
}

// WorkerRegionRuntime stores worker-side renderer registrations and mounted region state.
type WorkerRegionRuntime struct {
	storeWorkerRegionRendererByID                        map[string]WorkerRegionRenderer
	storeWorkerRegionRendererMetadataByID                map[string]RendererMetadata
	storeWorkerRegionRendererTrustedByID                 map[string]bool
	storeWorkerRegionRendererUpdateValidationEnabledByID map[string]bool
	storeWorkerRegionStateByID                           map[string]WorkerRegionState
	storeWorkerRegionCanceledByID                        map[string]uint64
	storeWorkerRegionEpochFloorByID                      map[string]uint64
	storeWorkerRegionPatchVersionByID                    map[string]uint64
	isWorkerRegionUpdateValidationEnabled                bool
}

// BuildWorkerRegionRuntime creates a worker-region runtime with empty renderer and region state maps.
func BuildWorkerRegionRuntime() *WorkerRegionRuntime {
	return &WorkerRegionRuntime{
		storeWorkerRegionRendererByID:                        make(map[string]WorkerRegionRenderer),
		storeWorkerRegionRendererMetadataByID:                make(map[string]RendererMetadata),
		storeWorkerRegionRendererTrustedByID:                 make(map[string]bool),
		storeWorkerRegionRendererUpdateValidationEnabledByID: make(map[string]bool),
		storeWorkerRegionStateByID:                           make(map[string]WorkerRegionState),
		storeWorkerRegionCanceledByID:                        make(map[string]uint64),
		storeWorkerRegionEpochFloorByID:                      make(map[string]uint64),
		storeWorkerRegionPatchVersionByID:                    make(map[string]uint64),
		isWorkerRegionUpdateValidationEnabled:                true,
	}
}

// RegisterWorkerRegionRenderer registers one worker-side renderer function by stable renderer ID.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) RegisterWorkerRegionRenderer(parseRendererID string, parseRender WorkerRegionRenderer) error {
	return parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithMetadata(
		parseRendererID,
		parseRender,
		buildWorkerRegionDisplayRendererMetadata(),
	)
}

// RegisterWorkerRegionRendererWithMetadata registers one worker-side renderer function together with capability metadata.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) RegisterWorkerRegionRendererWithMetadata(
	parseRendererID string,
	parseRender WorkerRegionRenderer,
	parseMetadata RendererMetadata,
) error {
	if parseWorkerRegionRuntime == nil {
		return fmt.Errorf("runtime2: worker region runtime is nil")
	}
	if strings.TrimSpace(parseRendererID) == "" {
		return fmt.Errorf("runtime2: renderer ID is required")
	}
	if parseRender == nil {
		return fmt.Errorf("runtime2: renderer function is required")
	}
	if parseMetadataErr := ValidateRendererMetadata(parseMetadata); parseMetadataErr != nil {
		return parseMetadataErr
	}
	if _, hasWorkerRegionRenderer := parseWorkerRegionRuntime.storeWorkerRegionRendererByID[parseRendererID]; hasWorkerRegionRenderer {
		return fmt.Errorf("runtime2: renderer ID %q is already registered", parseRendererID)
	}
	parseWorkerRegionRuntime.storeWorkerRegionRendererByID[parseRendererID] = parseRender
	parseWorkerRegionRuntime.storeWorkerRegionRendererMetadataByID[parseRendererID] = buildRendererMetadataCopy(parseMetadata)
	return nil
}

// RegisterWorkerRegionRendererWithRegisteredMetadata registers one worker-side renderer function using metadata resolved from the shared runtime2 registry.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) RegisterWorkerRegionRendererWithRegisteredMetadata(
	parseRendererID string,
	parseRender WorkerRegionRenderer,
) error {
	getRendererID, parseRendererIDErr := ParseRendererID(parseRendererID)
	if parseRendererIDErr != nil {
		return parseRendererIDErr
	}
	getRendererMetadata, parseResolveErr := ResolveRendererMetadata(getRendererID)
	if parseResolveErr != nil {
		return parseResolveErr
	}
	return parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithMetadata(
		string(getRendererID),
		parseRender,
		getRendererMetadata,
	)
}

// SetWorkerRegionRendererMetadata updates capability metadata for one registered worker renderer.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) SetWorkerRegionRendererMetadata(
	parseRendererID string,
	parseMetadata RendererMetadata,
) error {
	if parseWorkerRegionRuntime == nil {
		return fmt.Errorf("runtime2: worker region runtime is nil")
	}
	if strings.TrimSpace(parseRendererID) == "" {
		return fmt.Errorf("runtime2: renderer ID is required")
	}
	if parseMetadataErr := ValidateRendererMetadata(parseMetadata); parseMetadataErr != nil {
		return parseMetadataErr
	}
	if _, hasWorkerRegionRenderer := parseWorkerRegionRuntime.storeWorkerRegionRendererByID[parseRendererID]; !hasWorkerRegionRenderer {
		return fmt.Errorf("runtime2: renderer ID %q is not registered", parseRendererID)
	}
	parseWorkerRegionRuntime.storeWorkerRegionRendererMetadataByID[parseRendererID] = buildRendererMetadataCopy(parseMetadata)
	return nil
}

// SetWorkerRegionRendererTrusted marks one registered renderer as trusted or untrusted for update-path validation gating.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) SetWorkerRegionRendererTrusted(parseRendererID string, parseIsTrusted bool) error {
	if parseWorkerRegionRuntime == nil {
		return fmt.Errorf("runtime2: worker region runtime is nil")
	}
	if strings.TrimSpace(parseRendererID) == "" {
		return fmt.Errorf("runtime2: renderer ID is required")
	}
	if _, hasWorkerRegionRenderer := parseWorkerRegionRuntime.storeWorkerRegionRendererByID[parseRendererID]; !hasWorkerRegionRenderer {
		return fmt.Errorf("runtime2: renderer ID %q is not registered", parseRendererID)
	}
	if parseIsTrusted {
		parseWorkerRegionRuntime.storeWorkerRegionRendererTrustedByID[parseRendererID] = true
		return nil
	}
	delete(parseWorkerRegionRuntime.storeWorkerRegionRendererTrustedByID, parseRendererID)
	return nil
}

// SetWorkerRegionUpdateValidationEnabled enables or disables update-path render-output validation for trusted renderers.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) SetWorkerRegionUpdateValidationEnabled(parseIsEnabled bool) {
	if parseWorkerRegionRuntime == nil {
		return
	}
	parseWorkerRegionRuntime.isWorkerRegionUpdateValidationEnabled = parseIsEnabled
}

// SetWorkerRegionRendererUpdateValidationEnabled enables or disables update-path render-output validation for one registered renderer.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) SetWorkerRegionRendererUpdateValidationEnabled(parseRendererID string, parseIsEnabled bool) error {
	if parseWorkerRegionRuntime == nil {
		return fmt.Errorf("runtime2: worker region runtime is nil")
	}
	if strings.TrimSpace(parseRendererID) == "" {
		return fmt.Errorf("runtime2: renderer ID is required")
	}
	if _, hasWorkerRegionRenderer := parseWorkerRegionRuntime.storeWorkerRegionRendererByID[parseRendererID]; !hasWorkerRegionRenderer {
		return fmt.Errorf("runtime2: renderer ID %q is not registered", parseRendererID)
	}
	parseWorkerRegionRuntime.storeWorkerRegionRendererUpdateValidationEnabledByID[parseRendererID] = parseIsEnabled
	return nil
}

// HandleWorkerRegionMount resolves one renderer, builds initial render IR, and stores worker region state.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) HandleWorkerRegionMount(parseMount WorkerRegionMountSpec) (WorkerRegionState, error) {
	return parseWorkerRegionRuntime.handleWorkerRegionMount(parseMount, false)
}

// handleWorkerRegionMount resolves one renderer, builds initial render IR, and optionally skips snapshot-envelope validation when already validated by the caller.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) handleWorkerRegionMount(parseMount WorkerRegionMountSpec, parseHasMountSnapshotValidated bool) (WorkerRegionState, error) {
	if parseWorkerRegionRuntime == nil {
		return WorkerRegionState{}, fmt.Errorf("runtime2: worker region runtime is nil")
	}
	if strings.TrimSpace(parseMount.RegionID) == "" {
		return WorkerRegionState{}, fmt.Errorf("runtime2: region ID is required")
	}
	if strings.TrimSpace(parseMount.RendererID) == "" {
		return WorkerRegionState{}, fmt.Errorf("runtime2: renderer ID is required")
	}
	if getEpochFloor, hasEpochFloor := parseWorkerRegionRuntime.storeWorkerRegionEpochFloorByID[parseMount.RegionID]; hasEpochFloor && parseMount.Epoch < getEpochFloor {
		return WorkerRegionState{}, fmt.Errorf("runtime2: stale epoch %d for region %q (required>=%d)", parseMount.Epoch, parseMount.RegionID, getEpochFloor)
	}
	buildMountEpoch := parseMount.Epoch
	if buildMountEpoch == 0 {
		buildMountEpoch = 1
	}
	if !parseHasMountSnapshotValidated && parseMount.Snapshot.RegionInstanceID != "" {
		if parseSnapshotErr := ValidateSnapshotEnvelope(parseMount.Snapshot); parseSnapshotErr != nil {
			return WorkerRegionState{}, fmt.Errorf("runtime2: mount snapshot is invalid: %w", parseSnapshotErr)
		}
		if parseMount.Snapshot.RegionInstanceID != RegionInstanceID(parseMount.RegionID) {
			return WorkerRegionState{}, fmt.Errorf("runtime2: mount snapshot region %q does not match mount region %q", parseMount.Snapshot.RegionInstanceID, parseMount.RegionID)
		}
		if parseMount.Snapshot.Epoch != buildMountEpoch {
			return WorkerRegionState{}, fmt.Errorf("runtime2: mount snapshot epoch %d does not match mount epoch %d", parseMount.Snapshot.Epoch, buildMountEpoch)
		}
		if parseMount.Snapshot.InputVersion != parseMount.InputVersion {
			return WorkerRegionState{}, fmt.Errorf("runtime2: mount snapshot input version %d does not match mount input version %d", parseMount.Snapshot.InputVersion, parseMount.InputVersion)
		}
	}
	getWorkerRegionRenderer, hasWorkerRegionRenderer := parseWorkerRegionRuntime.storeWorkerRegionRendererByID[parseMount.RendererID]
	if !hasWorkerRegionRenderer {
		return WorkerRegionState{}, fmt.Errorf("runtime2: unknown renderer ID %q", parseMount.RendererID)
	}
	buildRenderInput, buildSourceIDs, parseRenderInputErr := buildWorkerRenderInputWithSourceOrderFromValidatedSnapshot(parseMount.Snapshot, nil)
	if parseRenderInputErr != nil {
		return WorkerRegionState{}, fmt.Errorf("runtime2: mount render input is invalid: %w", parseRenderInputErr)
	}
	parseRenderMountSpec := parseMount
	parseRenderMountSpec.RenderInput = buildRenderInput
	parseRenderIR, parseRenderErr := getWorkerRegionRenderer(parseRenderMountSpec)
	if parseRenderErr != nil {
		return WorkerRegionState{}, fmt.Errorf("runtime2: render mount region %q with renderer %q: %w", parseMount.RegionID, parseMount.RendererID, parseRenderErr)
	}
	if parseErr := ValidateWorkerRenderableRenderOutput(parseRenderIR); parseErr != nil {
		return WorkerRegionState{}, fmt.Errorf("runtime2: render mount region %q with renderer %q produced unsupported output: %w", parseMount.RegionID, parseMount.RendererID, parseErr)
	}
	buildCanonicalRenderIR, parseCanonicalErr := BuildCanonicalRenderIR(parseRenderIR)
	if parseCanonicalErr != nil {
		return WorkerRegionState{}, fmt.Errorf("runtime2: canonicalize mount render output for region %q: %w", parseMount.RegionID, parseCanonicalErr)
	}
	buildWorkerRegionState := WorkerRegionState{
		RegionID:     parseMount.RegionID,
		RendererID:   parseMount.RendererID,
		Epoch:        buildMountEpoch,
		InputVersion: parseMount.InputVersion,
		Snapshot:     parseMount.Snapshot,
		SourceIDs:    buildSourceIDs,
		RenderIR:     buildCanonicalRenderIR,
	}
	parseWorkerRegionRuntime.storeWorkerRegionStateByID[parseMount.RegionID] = buildWorkerRegionState
	parseWorkerRegionRuntime.storeWorkerRegionEpochFloorByID[parseMount.RegionID] = buildMountEpoch
	parseWorkerRegionRuntime.storeWorkerRegionPatchVersionByID[parseMount.RegionID] = parseMount.InputVersion
	return buildWorkerRegionState, nil
}

// HandleWorkerRegionUpdate renders one updated region snapshot and reports patch-ready or no-op results.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) HandleWorkerRegionUpdate(parseUpdate WorkerRegionUpdateSpec) (WorkerRegionUpdateResult, error) {
	return parseWorkerRegionRuntime.handleWorkerRegionUpdate(parseUpdate, false)
}

// handleWorkerRegionUpdate renders one updated region snapshot and optionally skips snapshot-envelope validation when already validated by the caller.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) handleWorkerRegionUpdate(parseUpdate WorkerRegionUpdateSpec, parseHasUpdateSnapshotValidated bool) (WorkerRegionUpdateResult, error) {
	if parseWorkerRegionRuntime == nil {
		return WorkerRegionUpdateResult{}, fmt.Errorf("runtime2: worker region runtime is nil")
	}
	if strings.TrimSpace(parseUpdate.RegionID) == "" {
		return WorkerRegionUpdateResult{}, fmt.Errorf("runtime2: region ID is required")
	}
	if getEpochFloor, hasEpochFloor := parseWorkerRegionRuntime.storeWorkerRegionEpochFloorByID[parseUpdate.RegionID]; hasEpochFloor && parseUpdate.Epoch > 0 && parseUpdate.Epoch < getEpochFloor {
		return WorkerRegionUpdateResult{}, fmt.Errorf("runtime2: stale epoch %d for region %q (required>=%d)", parseUpdate.Epoch, parseUpdate.RegionID, getEpochFloor)
	}
	getWorkerRegionState, hasWorkerRegionState := parseWorkerRegionRuntime.storeWorkerRegionStateByID[parseUpdate.RegionID]
	if !hasWorkerRegionState {
		return WorkerRegionUpdateResult{}, fmt.Errorf("runtime2: unknown region ID %q", parseUpdate.RegionID)
	}
	if strings.TrimSpace(parseUpdate.RendererID) != "" && parseUpdate.RendererID != getWorkerRegionState.RendererID {
		return WorkerRegionUpdateResult{}, fmt.Errorf(
			"runtime2: update renderer %q does not match mounted renderer %q for region %q",
			parseUpdate.RendererID,
			getWorkerRegionState.RendererID,
			parseUpdate.RegionID,
		)
	}
	parseUpdateEpoch := parseUpdate.Epoch
	if parseUpdateEpoch == 0 {
		parseUpdateEpoch = getWorkerRegionState.Epoch
	}
	if !parseHasUpdateSnapshotValidated && parseUpdate.Snapshot.RegionInstanceID != "" {
		if parseSnapshotErr := ValidateSnapshotEnvelope(parseUpdate.Snapshot); parseSnapshotErr != nil {
			return WorkerRegionUpdateResult{}, fmt.Errorf("runtime2: update snapshot is invalid: %w", parseSnapshotErr)
		}
		if parseUpdate.Snapshot.RegionInstanceID != RegionInstanceID(parseUpdate.RegionID) {
			return WorkerRegionUpdateResult{}, fmt.Errorf("runtime2: update snapshot region %q does not match update region %q", parseUpdate.Snapshot.RegionInstanceID, parseUpdate.RegionID)
		}
		if parseUpdate.Snapshot.Epoch != parseUpdateEpoch {
			return WorkerRegionUpdateResult{}, fmt.Errorf("runtime2: update snapshot epoch %d does not match update epoch %d", parseUpdate.Snapshot.Epoch, parseUpdateEpoch)
		}
		if parseUpdate.Snapshot.InputVersion != parseUpdate.InputVersion {
			return WorkerRegionUpdateResult{}, fmt.Errorf("runtime2: update snapshot input version %d does not match update input version %d", parseUpdate.Snapshot.InputVersion, parseUpdate.InputVersion)
		}
	}
	parseEffectiveSnapshot := parseResolveWorkerUpdateSnapshot(parseUpdate, getWorkerRegionState, parseUpdateEpoch)
	if parseUpdateEpoch != getWorkerRegionState.Epoch {
		return WorkerRegionUpdateResult{}, fmt.Errorf("runtime2: stale epoch %d for region %q (latest=%d)", parseUpdateEpoch, parseUpdate.RegionID, getWorkerRegionState.Epoch)
	}
	if getCanceledInputVersion, hasCanceledInputVersion := parseWorkerRegionRuntime.storeWorkerRegionCanceledByID[parseUpdate.RegionID]; hasCanceledInputVersion && parseUpdate.InputVersion <= getCanceledInputVersion {
		return WorkerRegionUpdateResult{
			IsCanceled: true,
		}, nil
	}
	if parseUpdate.InputVersion <= getWorkerRegionState.InputVersion {
		return WorkerRegionUpdateResult{}, fmt.Errorf("runtime2: stale input version %d for region %q (latest=%d)", parseUpdate.InputVersion, parseUpdate.RegionID, getWorkerRegionState.InputVersion)
	}
	getWorkerRegionRenderer, hasWorkerRegionRenderer := parseWorkerRegionRuntime.storeWorkerRegionRendererByID[getWorkerRegionState.RendererID]
	if !hasWorkerRegionRenderer {
		return WorkerRegionUpdateResult{}, fmt.Errorf("runtime2: unknown renderer ID %q", getWorkerRegionState.RendererID)
	}
	buildRenderInput, buildSourceIDs, parseRenderInputErr := buildWorkerRenderInputWithSourceOrderFromValidatedSnapshot(
		parseEffectiveSnapshot,
		getWorkerRegionState.SourceIDs,
	)
	if parseRenderInputErr != nil {
		return WorkerRegionUpdateResult{}, fmt.Errorf("runtime2: update render input is invalid: %w", parseRenderInputErr)
	}
	parseRenderIR, parseRenderErr := getWorkerRegionRenderer(WorkerRegionMountSpec{
		RegionID:     parseUpdate.RegionID,
		RendererID:   getWorkerRegionState.RendererID,
		Epoch:        parseUpdateEpoch,
		InputVersion: parseUpdate.InputVersion,
		Snapshot:     parseEffectiveSnapshot,
		RenderInput:  buildRenderInput,
	})
	if parseRenderErr != nil {
		return WorkerRegionUpdateResult{}, fmt.Errorf("runtime2: render update region %q with renderer %q: %w", parseUpdate.RegionID, getWorkerRegionState.RendererID, parseRenderErr)
	}
	if parseWorkerRegionRuntime.shouldWorkerRegionValidateUpdateRenderOutput(getWorkerRegionState.RendererID) {
		if parseErr := ValidateWorkerRenderableRenderOutput(parseRenderIR); parseErr != nil {
			return WorkerRegionUpdateResult{}, fmt.Errorf("runtime2: render update region %q with renderer %q produced unsupported output: %w", parseUpdate.RegionID, getWorkerRegionState.RendererID, parseErr)
		}
	}
	buildCanonicalRenderIR, parseCanonicalErr := BuildCanonicalRenderIR(parseRenderIR)
	if parseCanonicalErr != nil {
		return WorkerRegionUpdateResult{}, fmt.Errorf("runtime2: canonicalize update render output for region %q: %w", parseUpdate.RegionID, parseCanonicalErr)
	}
	if IsCanonicalRenderIREqual(getWorkerRegionState.RenderIR, buildCanonicalRenderIR) {
		getWorkerRegionState.InputVersion = parseUpdate.InputVersion
		getWorkerRegionState.Snapshot = parseEffectiveSnapshot
		getWorkerRegionState.SourceIDs = buildSourceIDs
		parseWorkerRegionRuntime.storeWorkerRegionStateByID[parseUpdate.RegionID] = getWorkerRegionState
		return WorkerRegionUpdateResult{
			IsNoOp: true,
		}, nil
	}
	buildPatchVersion := parseWorkerRegionRuntime.storeWorkerRegionPatchVersionByID[parseUpdate.RegionID] + 1
	buildWorkerRegionPatch, buildNoOp, parsePatchErr := buildWorkerRegionUpdatePatch(
		parseUpdate.RegionID,
		parseUpdateEpoch,
		parseUpdate.InputVersion,
		buildPatchVersion,
		getWorkerRegionState.RenderIR,
		buildCanonicalRenderIR,
	)
	if parsePatchErr != nil {
		return WorkerRegionUpdateResult{}, parsePatchErr
	}
	getWorkerRegionState.InputVersion = parseUpdate.InputVersion
	getWorkerRegionState.Snapshot = parseEffectiveSnapshot
	getWorkerRegionState.SourceIDs = buildSourceIDs
	getWorkerRegionState.RenderIR = buildCanonicalRenderIR
	parseWorkerRegionRuntime.storeWorkerRegionStateByID[parseUpdate.RegionID] = getWorkerRegionState
	parseWorkerRegionRuntime.storeWorkerRegionPatchVersionByID[parseUpdate.RegionID] = buildPatchVersion
	if buildNoOp {
		return WorkerRegionUpdateResult{
			IsNoOp: true,
		}, nil
	}
	return WorkerRegionUpdateResult{
		HasPatchReady: true,
		PatchIR:       buildWorkerRegionPatch,
	}, nil
}

// HandleWorkerRegionEvent renders one semantic event-slot dispatch against mounted worker state and reports patch-ready or no-op results.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) HandleWorkerRegionEvent(parseEvent WorkerRegionEventSpec) (WorkerRegionEventResult, error) {
	return parseWorkerRegionRuntime.handleWorkerRegionEvent(parseEvent)
}

// handleWorkerRegionEvent renders one semantic event-slot dispatch against mounted worker state and reports patch-ready or no-op results.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) handleWorkerRegionEvent(parseEvent WorkerRegionEventSpec) (WorkerRegionEventResult, error) {
	if parseWorkerRegionRuntime == nil {
		return WorkerRegionEventResult{}, fmt.Errorf("runtime2: worker region runtime is nil")
	}
	if strings.TrimSpace(parseEvent.RegionID) == "" {
		return WorkerRegionEventResult{}, fmt.Errorf("runtime2: region ID is required")
	}
	parseNormalizedEventSlot, parseEventSlotErr := buildEventSlotDispatchNormalized(parseEvent.EventSlot)
	if parseEventSlotErr != nil {
		return WorkerRegionEventResult{}, parseEventSlotErr
	}
	getWorkerRegionState, hasWorkerRegionState := parseWorkerRegionRuntime.storeWorkerRegionStateByID[parseEvent.RegionID]
	if !hasWorkerRegionState {
		return WorkerRegionEventResult{}, fmt.Errorf("runtime2: unknown region ID %q", parseEvent.RegionID)
	}
	getEventInputVersion := parseEvent.InputVersion
	if getEventInputVersion == 0 {
		getEventInputVersion = getWorkerRegionState.InputVersion
	}
	if getEventInputVersion < getWorkerRegionState.InputVersion {
		return WorkerRegionEventResult{}, fmt.Errorf(
			"runtime2: stale event input version %d for region %q (latest=%d)",
			getEventInputVersion,
			parseEvent.RegionID,
			getWorkerRegionState.InputVersion,
		)
	}
	parseRendererMetadata, hasRendererMetadata := parseWorkerRegionRuntime.storeWorkerRegionRendererMetadataByID[getWorkerRegionState.RendererID]
	if !hasRendererMetadata {
		return WorkerRegionEventResult{}, fmt.Errorf("runtime2: renderer metadata for %q is not registered", getWorkerRegionState.RendererID)
	}
	if !hasEventSlotMetadataDispatch(parseRendererMetadata.EventSlotMetadata, parseNormalizedEventSlot) {
		return WorkerRegionEventResult{}, fmt.Errorf(
			"runtime2: renderer %q does not declare event slot %q for event %q",
			getWorkerRegionState.RendererID,
			parseNormalizedEventSlot.SlotID,
			parseNormalizedEventSlot.EventType,
		)
	}
	getWorkerRegionRenderer, hasWorkerRegionRenderer := parseWorkerRegionRuntime.storeWorkerRegionRendererByID[getWorkerRegionState.RendererID]
	if !hasWorkerRegionRenderer {
		return WorkerRegionEventResult{}, fmt.Errorf("runtime2: unknown renderer ID %q", getWorkerRegionState.RendererID)
	}
	buildRenderInput, _, parseRenderInputErr := buildWorkerRenderInputWithSourceOrderFromValidatedSnapshot(
		getWorkerRegionState.Snapshot,
		getWorkerRegionState.SourceIDs,
	)
	if parseRenderInputErr != nil {
		return WorkerRegionEventResult{}, fmt.Errorf("runtime2: event render input is invalid: %w", parseRenderInputErr)
	}
	buildRenderInput.GetEventSlot = &EventSlotDispatch{
		SlotID:    parseNormalizedEventSlot.SlotID,
		EventType: parseNormalizedEventSlot.EventType,
		Payload:   parseNormalizedEventSlot.Payload,
	}
	parseRenderIR, parseRenderErr := getWorkerRegionRenderer(WorkerRegionMountSpec{
		RegionID:     getWorkerRegionState.RegionID,
		RendererID:   getWorkerRegionState.RendererID,
		Epoch:        getWorkerRegionState.Epoch,
		InputVersion: getEventInputVersion,
		Snapshot:     getWorkerRegionState.Snapshot,
		RenderInput:  buildRenderInput,
	})
	if parseRenderErr != nil {
		return WorkerRegionEventResult{}, fmt.Errorf(
			"runtime2: render event region %q with renderer %q: %w",
			parseEvent.RegionID,
			getWorkerRegionState.RendererID,
			parseRenderErr,
		)
	}
	if parseWorkerRegionRuntime.shouldWorkerRegionValidateUpdateRenderOutput(getWorkerRegionState.RendererID) {
		if parseErr := ValidateWorkerRenderableRenderOutput(parseRenderIR); parseErr != nil {
			return WorkerRegionEventResult{}, fmt.Errorf(
				"runtime2: render event region %q with renderer %q produced unsupported output: %w",
				parseEvent.RegionID,
				getWorkerRegionState.RendererID,
				parseErr,
			)
		}
	}
	buildCanonicalRenderIR, parseCanonicalErr := BuildCanonicalRenderIR(parseRenderIR)
	if parseCanonicalErr != nil {
		return WorkerRegionEventResult{}, fmt.Errorf("runtime2: canonicalize event render output for region %q: %w", parseEvent.RegionID, parseCanonicalErr)
	}
	if IsCanonicalRenderIREqual(getWorkerRegionState.RenderIR, buildCanonicalRenderIR) {
		getWorkerRegionState.InputVersion = getEventInputVersion
		getWorkerRegionState.Snapshot.InputVersion = getEventInputVersion
		parseWorkerRegionRuntime.storeWorkerRegionStateByID[parseEvent.RegionID] = getWorkerRegionState
		return WorkerRegionEventResult{
			IsNoOp: true,
		}, nil
	}
	buildPatchVersion := parseWorkerRegionRuntime.storeWorkerRegionPatchVersionByID[parseEvent.RegionID] + 1
	buildWorkerRegionPatch, buildNoOp, parsePatchErr := buildWorkerRegionUpdatePatch(
		parseEvent.RegionID,
		getWorkerRegionState.Epoch,
		getEventInputVersion,
		buildPatchVersion,
		getWorkerRegionState.RenderIR,
		buildCanonicalRenderIR,
	)
	if parsePatchErr != nil {
		return WorkerRegionEventResult{}, parsePatchErr
	}
	getWorkerRegionState.InputVersion = getEventInputVersion
	getWorkerRegionState.Snapshot.InputVersion = getEventInputVersion
	getWorkerRegionState.RenderIR = buildCanonicalRenderIR
	parseWorkerRegionRuntime.storeWorkerRegionStateByID[parseEvent.RegionID] = getWorkerRegionState
	parseWorkerRegionRuntime.storeWorkerRegionPatchVersionByID[parseEvent.RegionID] = buildPatchVersion
	if buildNoOp {
		return WorkerRegionEventResult{
			IsNoOp: true,
		}, nil
	}
	return WorkerRegionEventResult{
		HasPatchReady: true,
		PatchIR:       buildWorkerRegionPatch,
	}, nil
}

// buildWorkerRegionUpdatePatch builds one worker update patch, using a single-text-node fast path when safe.
func buildWorkerRegionUpdatePatch(
	parseRegionID string,
	parseEpoch uint64,
	parseInputVersion uint64,
	parsePatchVersion uint64,
	parsePreviousIR CanonicalRenderIR,
	parseNextIR CanonicalRenderIR,
) (PatchStreamRaw, bool, error) {
	buildFastPatch, hasFastPatch, parseFastPatchErr := buildWorkerRegionFastSetTextPatch(
		parseRegionID,
		parseEpoch,
		parseInputVersion,
		parsePatchVersion,
		parsePreviousIR,
		parseNextIR,
	)
	if parseFastPatchErr != nil {
		return PatchStreamRaw{}, false, parseFastPatchErr
	}
	if hasFastPatch {
		return buildFastPatch, false, nil
	}
	return BuildCanonicalPatchStream(
		parseRegionID,
		parseEpoch,
		parseInputVersion,
		parsePatchVersion,
		parsePreviousIR,
		parseNextIR,
	)
}

// buildWorkerRegionFastSetTextPatch builds one set-text patch when both IR payloads are single text nodes with matching IDs.
func buildWorkerRegionFastSetTextPatch(
	parseRegionID string,
	parseEpoch uint64,
	parseInputVersion uint64,
	parsePatchVersion uint64,
	parsePreviousIR CanonicalRenderIR,
	parseNextIR CanonicalRenderIR,
) (PatchStreamRaw, bool, error) {
	parsePreviousNodeRecord, parsePreviousTextValue, hasPreviousTextRecord := parseResolveWorkerSingleTextNodeRecord(parsePreviousIR)
	if !hasPreviousTextRecord {
		return PatchStreamRaw{}, false, nil
	}
	parseNextNodeRecord, parseNextTextValue, hasNextTextRecord := parseResolveWorkerSingleTextNodeRecord(parseNextIR)
	if !hasNextTextRecord {
		return PatchStreamRaw{}, false, nil
	}
	if parsePreviousNodeRecord.NodeID != parseNextNodeRecord.NodeID {
		return PatchStreamRaw{}, false, nil
	}
	if parsePreviousTextValue == parseNextTextValue {
		return PatchStreamRaw{}, false, nil
	}
	buildPatchStringTable := BuildRenderStringTable([]string{parseNextTextValue})
	buildTextRef, hasTextRef := buildPatchStringTable.GetRenderStringRef(parseNextTextValue)
	if !hasTextRef {
		return PatchStreamRaw{}, false, fmt.Errorf("runtime2: set-text payload %q missing from patch table", parseNextTextValue)
	}
	buildHeader := PatchStreamHeaderRaw{
		ProtocolVersion: PatchStreamProtocolVersion,
		RegionID:        parseRegionID,
		Epoch:           parseEpoch,
		InputVersion:    parseInputVersion,
		PatchVersion:    parsePatchVersion,
	}
	buildOps := []PatchStreamOpRaw{
		{
			GetOpCode: uint8(PatchOpCodeSetText),
			GetSetTextOp: &PatchSetTextOpRaw{
				TargetNodeID: parsePreviousNodeRecord.NodeID,
				TextRef:      buildTextRef,
			},
		},
	}
	buildPatchStringEntries := append([]string(nil), buildPatchStringTable.Entries...)
	buildPatchIdentity, parseIdentityErr := buildPatchStreamIdentityFromParts(buildHeader, buildPatchStringEntries, buildOps)
	if parseIdentityErr != nil {
		return PatchStreamRaw{}, false, parseIdentityErr
	}
	buildPatchStreamRaw, parsePatchStreamErr := buildPatchStreamRawWithIdentity(
		buildHeader,
		buildPatchStringEntries,
		buildOps,
		buildPatchIdentity,
	)
	if parsePatchStreamErr != nil {
		return PatchStreamRaw{}, false, parsePatchStreamErr
	}
	return buildPatchStreamRaw, true, nil
}

// parseResolveWorkerSingleTextNodeRecord resolves one single-node text IR record and its text payload.
func parseResolveWorkerSingleTextNodeRecord(parseIR CanonicalRenderIR) (RenderNodeRecordRaw, string, bool) {
	if len(parseIR.GetNodeRecords) != 1 || len(parseIR.GetPropRecords) != 0 {
		return RenderNodeRecordRaw{}, "", false
	}
	parseNodeRecord := parseIR.GetNodeRecords[0]
	if RenderNodeKind(parseNodeRecord.Kind) != RenderNodeKindText {
		return RenderNodeRecordRaw{}, "", false
	}
	if parseNodeRecord.ChildCount != 0 || parseNodeRecord.PropCount != 0 {
		return RenderNodeRecordRaw{}, "", false
	}
	if int(parseNodeRecord.TextRef) >= len(parseIR.GetStringTable.Entries) {
		return RenderNodeRecordRaw{}, "", false
	}
	return parseNodeRecord, parseIR.GetStringTable.Entries[parseNodeRecord.TextRef], true
}

// shouldWorkerRegionValidateUpdateRenderOutput reports whether update-path render-output validation should run for one renderer.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) shouldWorkerRegionValidateUpdateRenderOutput(parseRendererID string) bool {
	if parseWorkerRegionRuntime == nil {
		return true
	}
	if parseIsEnabled, hasRendererOverride := parseWorkerRegionRuntime.storeWorkerRegionRendererUpdateValidationEnabledByID[parseRendererID]; hasRendererOverride {
		return parseIsEnabled
	}
	if parseWorkerRegionRuntime.isWorkerRegionUpdateValidationEnabled {
		return true
	}
	parseRendererMetadata, hasRendererMetadata := parseWorkerRegionRuntime.storeWorkerRegionRendererMetadataByID[parseRendererID]
	if !hasRendererMetadata || !HasRendererFeatureFlag(parseRendererMetadata, "display-only") {
		return true
	}
	return !parseWorkerRegionRuntime.storeWorkerRegionRendererTrustedByID[parseRendererID]
}

// buildWorkerRegionDisplayRendererMetadata returns the default worker renderer capability metadata for first-slice display-only regions.
func buildWorkerRegionDisplayRendererMetadata() RendererMetadata {
	return RendererMetadata{
		FeatureFlags: []string{"display-only"},
	}
}

// parseResolveWorkerUpdateSnapshot resolves the snapshot attached to one update, falling back to the cached snapshot when absent.
func parseResolveWorkerUpdateSnapshot(parseUpdate WorkerRegionUpdateSpec, parseWorkerRegionState WorkerRegionState, parseUpdateEpoch uint64) SnapshotEnvelope {
	if parseUpdate.Snapshot.RegionInstanceID != "" {
		return parseUpdate.Snapshot
	}
	if parseWorkerRegionState.Snapshot.RegionInstanceID == "" {
		return SnapshotEnvelope{}
	}
	parseSnapshot := parseWorkerRegionState.Snapshot
	parseSnapshot.Epoch = parseUpdateEpoch
	parseSnapshot.InputVersion = parseUpdate.InputVersion
	return parseSnapshot
}

// HandleWorkerRegionCancel records a cancel watermark for one region to suppress stale or canceled patch-ready output.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) HandleWorkerRegionCancel(parseCancel WorkerRegionCancelSpec) (WorkerRegionCancelResult, error) {
	if parseWorkerRegionRuntime == nil {
		return WorkerRegionCancelResult{}, fmt.Errorf("runtime2: worker region runtime is nil")
	}
	if strings.TrimSpace(parseCancel.RegionID) == "" {
		return WorkerRegionCancelResult{}, fmt.Errorf("runtime2: region ID is required")
	}
	if _, hasWorkerRegionState := parseWorkerRegionRuntime.storeWorkerRegionStateByID[parseCancel.RegionID]; !hasWorkerRegionState {
		return WorkerRegionCancelResult{}, nil
	}
	getCanceledInputVersion, hasCanceledInputVersion := parseWorkerRegionRuntime.storeWorkerRegionCanceledByID[parseCancel.RegionID]
	if !hasCanceledInputVersion || parseCancel.InputVersion > getCanceledInputVersion {
		parseWorkerRegionRuntime.storeWorkerRegionCanceledByID[parseCancel.RegionID] = parseCancel.InputVersion
	}
	return WorkerRegionCancelResult{
		HasRegion:         true,
		HasCancelRecorded: true,
	}, nil
}

// HandleWorkerRegionDispose clears one region's cached worker state and cancel watermark.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) HandleWorkerRegionDispose(parseRegionID string) WorkerRegionDisposeResult {
	if parseWorkerRegionRuntime == nil {
		return WorkerRegionDisposeResult{}
	}
	if strings.TrimSpace(parseRegionID) == "" {
		return WorkerRegionDisposeResult{}
	}
	_, hasWorkerRegionState := parseWorkerRegionRuntime.storeWorkerRegionStateByID[parseRegionID]
	delete(parseWorkerRegionRuntime.storeWorkerRegionStateByID, parseRegionID)
	delete(parseWorkerRegionRuntime.storeWorkerRegionCanceledByID, parseRegionID)
	delete(parseWorkerRegionRuntime.storeWorkerRegionEpochFloorByID, parseRegionID)
	delete(parseWorkerRegionRuntime.storeWorkerRegionPatchVersionByID, parseRegionID)
	return WorkerRegionDisposeResult{
		HasStateCleared: hasWorkerRegionState,
	}
}

// HandleWorkerRegionRestart advances one region epoch and clears stale worker-side state.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) HandleWorkerRegionRestart(parseRestart WorkerRegionRestartSpec) (WorkerRegionRestartResult, error) {
	if parseWorkerRegionRuntime == nil {
		return WorkerRegionRestartResult{}, fmt.Errorf("runtime2: worker region runtime is nil")
	}
	if strings.TrimSpace(parseRestart.RegionID) == "" {
		return WorkerRegionRestartResult{}, fmt.Errorf("runtime2: region ID is required")
	}
	if parseRestart.Epoch == 0 {
		return WorkerRegionRestartResult{}, fmt.Errorf("runtime2: restart epoch is required")
	}
	getEpochFloor := parseWorkerRegionRuntime.storeWorkerRegionEpochFloorByID[parseRestart.RegionID]
	if parseRestart.Epoch < getEpochFloor {
		return WorkerRegionRestartResult{}, fmt.Errorf("runtime2: stale epoch %d for region %q (required>=%d)", parseRestart.Epoch, parseRestart.RegionID, getEpochFloor)
	}
	parseWorkerRegionRuntime.storeWorkerRegionEpochFloorByID[parseRestart.RegionID] = parseRestart.Epoch
	delete(parseWorkerRegionRuntime.storeWorkerRegionStateByID, parseRestart.RegionID)
	delete(parseWorkerRegionRuntime.storeWorkerRegionCanceledByID, parseRestart.RegionID)
	delete(parseWorkerRegionRuntime.storeWorkerRegionPatchVersionByID, parseRestart.RegionID)
	return WorkerRegionRestartResult{
		HasEpochAdvanced: parseRestart.Epoch > getEpochFloor,
	}, nil
}

// GetWorkerRegionState reports cached worker region state for one region ID.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) GetWorkerRegionState(parseRegionID string) (WorkerRegionState, bool) {
	if parseWorkerRegionRuntime == nil {
		return WorkerRegionState{}, false
	}
	getWorkerRegionState, hasWorkerRegionState := parseWorkerRegionRuntime.storeWorkerRegionStateByID[parseRegionID]
	if !hasWorkerRegionState {
		return WorkerRegionState{}, false
	}
	return getWorkerRegionState, true
}
