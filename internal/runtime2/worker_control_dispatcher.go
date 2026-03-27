package runtime2

import "fmt"

// WorkerControlDispatchResult reports one worker-side control-envelope dispatch outcome.
type WorkerControlDispatchResult struct {
	HasMountResult   bool
	HasUpdateResult  bool
	HasCancelResult  bool
	HasDisposeResult bool
	HasRestartResult bool

	GetMountState   WorkerRegionState
	GetUpdateResult WorkerRegionUpdateResult
	GetCancelResult WorkerRegionCancelResult
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
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return WorkerControlDispatchResult{}, parseErr
	}
	switch parseEnvelope.Kind {
	case ControlKindMount:
		parseMountState, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
			RegionID:     string(parseEnvelope.RegionInstanceID),
			RendererID:   string(parseEnvelope.RendererID),
			Epoch:        parseEnvelope.Snapshot.Epoch,
			InputVersion: parseEnvelope.Snapshot.InputVersion,
		})
		if parseMountErr != nil {
			return WorkerControlDispatchResult{}, parseMountErr
		}
		return WorkerControlDispatchResult{
			HasMountResult: true,
			GetMountState:  parseMountState,
		}, nil
	case ControlKindUpdate:
		parseUpdateResult, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
			RegionID:     string(parseEnvelope.RegionInstanceID),
			Epoch:        parseEnvelope.Snapshot.Epoch,
			InputVersion: parseEnvelope.InputVersion,
		})
		if parseUpdateErr != nil {
			return WorkerControlDispatchResult{}, parseUpdateErr
		}
		return WorkerControlDispatchResult{
			HasUpdateResult: true,
			GetUpdateResult: parseUpdateResult,
		}, nil
	case ControlKindCancel:
		parseCancelResult, parseCancelErr := parseWorkerRegionRuntime.HandleWorkerRegionCancel(WorkerRegionCancelSpec{
			RegionID:     string(parseEnvelope.RegionInstanceID),
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
		parseDisposeResult := parseWorkerRegionRuntime.HandleWorkerRegionDispose(string(parseEnvelope.RegionInstanceID))
		return WorkerControlDispatchResult{
			HasDisposeResult: true,
			GetDisposeResult: parseDisposeResult,
		}, nil
	case ControlKindRestart:
		parseRestartResult, parseRestartErr := parseWorkerRegionRuntime.HandleWorkerRegionRestart(WorkerRegionRestartSpec{
			RegionID: string(parseEnvelope.RegionInstanceID),
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
