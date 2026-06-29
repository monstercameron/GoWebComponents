package runtime2

import "fmt"

// WorkerControlDispatchResult reports one worker-side control-envelope dispatch outcome.
type WorkerControlDispatchResult struct {
	HasMountResult   bool
	HasUpdateResult  bool
	HasEventResult   bool
	HasCancelResult  bool
	HasDisposeResult bool
	HasRestartResult bool

	GetMountState    WorkerRegionState
	GetUpdateResult  WorkerRegionUpdateResult
	GetEventResult   WorkerRegionEventResult
	GetCancelResult  WorkerRegionCancelResult
	GetDisposeResult WorkerRegionDisposeResult
	GetRestartResult WorkerRegionRestartResult
}

// HandleWorkerControlEnvelope validates and routes one worker-side control envelope.
func HandleWorkerControlEnvelope(
	parseWorkerRegionRuntime *WorkerRegionRuntime,
	parseEnvelope ControlEnvelope,
) (WorkerControlDispatchResult, error) {
	if parseWorkerRegionRuntime == nil {
		return WorkerControlDispatchResult{}, fmt.Errorf("runtime2: worker region runtime is nil")
	}
	switch parseEnvelope.Kind {
	case ControlKindMount, ControlKindUpdate, ControlKindEvent, ControlKindCancel, ControlKindDispose, ControlKindRestart:
	default:
		if _, parseKindErr := ParseControlKind(string(parseEnvelope.Kind)); parseKindErr != nil {
			return WorkerControlDispatchResult{}, parseKindErr
		}
		return WorkerControlDispatchResult{}, fmt.Errorf("runtime2: control kind %q is unsupported for worker dispatch", parseEnvelope.Kind)
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return WorkerControlDispatchResult{}, parseErr
	}
	getRegionID := string(parseEnvelope.RegionInstanceID)
	getRendererID := string(parseEnvelope.RendererID)
	getSnapshot := parseEnvelope.Snapshot
	switch parseEnvelope.Kind {
	case ControlKindMount:
		if getSnapshot == nil {
			return WorkerControlDispatchResult{}, fmt.Errorf("runtime2: mount snapshot is required")
		}
		parseMountState, parseMountErr := parseWorkerRegionRuntime.handleWorkerRegionMount(WorkerRegionMountSpec{
			RegionID:     getRegionID,
			RendererID:   getRendererID,
			Epoch:        getSnapshot.Epoch,
			InputVersion: getSnapshot.InputVersion,
			Snapshot:     *getSnapshot,
		}, true)
		if parseMountErr != nil {
			return WorkerControlDispatchResult{}, parseMountErr
		}
		return WorkerControlDispatchResult{
			HasMountResult: true,
			GetMountState:  parseMountState,
		}, nil
	case ControlKindUpdate:
		if getSnapshot == nil {
			return WorkerControlDispatchResult{}, fmt.Errorf("runtime2: update snapshot is required")
		}
		parseUpdateResult, parseUpdateErr := parseWorkerRegionRuntime.handleWorkerRegionUpdate(WorkerRegionUpdateSpec{
			RegionID:     getRegionID,
			RendererID:   getRendererID,
			Epoch:        getSnapshot.Epoch,
			InputVersion: parseEnvelope.InputVersion,
			Snapshot:     *getSnapshot,
		}, true)
		if parseUpdateErr != nil {
			return WorkerControlDispatchResult{}, parseUpdateErr
		}
		return WorkerControlDispatchResult{
			HasUpdateResult: true,
			GetUpdateResult: parseUpdateResult,
		}, nil
	case ControlKindEvent:
		if parseEnvelope.EventSlot == nil {
			return WorkerControlDispatchResult{}, fmt.Errorf("runtime2: event-slot payload is required")
		}
		parseEventResult, parseEventErr := parseWorkerRegionRuntime.HandleWorkerRegionEvent(WorkerRegionEventSpec{
			RegionID:  getRegionID,
			EventSlot: *parseEnvelope.EventSlot,
		})
		if parseEventErr != nil {
			return WorkerControlDispatchResult{}, parseEventErr
		}
		return WorkerControlDispatchResult{
			HasEventResult: true,
			GetEventResult: parseEventResult,
		}, nil
	case ControlKindCancel:
		parseCancelResult, parseCancelErr := parseWorkerRegionRuntime.HandleWorkerRegionCancel(WorkerRegionCancelSpec{
			RegionID:     getRegionID,
			InputVersion: parseEnvelope.InputVersion,
		})
		if parseCancelErr != nil {
			return WorkerControlDispatchResult{}, parseCancelErr
		}
		return WorkerControlDispatchResult{
			HasCancelResult: true,
			GetCancelResult: parseCancelResult,
		}, nil
	case ControlKindDispose:
		parseDisposeResult := parseWorkerRegionRuntime.HandleWorkerRegionDispose(getRegionID)
		return WorkerControlDispatchResult{
			HasDisposeResult: true,
			GetDisposeResult: parseDisposeResult,
		}, nil
	case ControlKindRestart:
		parseRestartResult, parseRestartErr := parseWorkerRegionRuntime.HandleWorkerRegionRestart(WorkerRegionRestartSpec{
			RegionID: getRegionID,
			Epoch:    parseEnvelope.Epoch,
		})
		if parseRestartErr != nil {
			return WorkerControlDispatchResult{}, parseRestartErr
		}
		return WorkerControlDispatchResult{
			HasRestartResult: true,
			GetRestartResult: parseRestartResult,
		}, nil
	default:
		return WorkerControlDispatchResult{}, fmt.Errorf("runtime2: control kind %q is unsupported for worker dispatch", parseEnvelope.Kind)
	}
}
