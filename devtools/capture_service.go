package devtools

import "github.com/monstercameron/GoWebComponents/v6/internal/pluginruntime"

func init() {
	_ = pluginruntime.RegisterBuiltinService(pluginruntime.ServiceRegistration{
		Key:   pluginruntime.ServiceKeyCapture,
		Value: buildCaptureService{},
	})
}

type buildCaptureService struct{}

// GetCaptureSnapshot returns the current trace-replay capture summary.
func (buildCaptureService) GetCaptureSnapshot(parseBudget pluginruntime.QueryBudget) (pluginruntime.CaptureSnapshot, error) {
	buildSnapshot := pluginruntime.CaptureSnapshot{
		Meta: pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDRuntime1, false),
	}
	if getReplay, hasReplay := CurrentTraceReplay(); hasReplay {
		buildSnapshot.Sessions = append(buildSnapshot.Sessions, getReplay.Label)
	}
	return buildSnapshot, nil
}
