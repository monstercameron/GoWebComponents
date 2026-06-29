package ui

import "github.com/monstercameron/GoWebComponents/v4/internal/runtime"

// reportParallelRegionDiagnosticError emits one parallel-region diagnostic message through the shared runtime channel.
func reportParallelRegionDiagnosticError(parseMessage string) {
	runtime.ReportDiagnostic("ui", runtime.DiagnosticError, parseMessage)
}

// renderParallelRegionShellNode builds one parallel-region shell host node with the provided marker props.
func renderParallelRegionShellNode(parseShellProps map[string]any, parseChild Node) Node {
	if parseChild == nil {
		return runtime.CreateElement("div", parseShellProps)
	}
	return runtime.CreateElement("div", parseShellProps, parseChild)
}

// isParallelRegionTransitionUpdate reports whether the current render pass is running in the transition lane.
func isParallelRegionTransitionUpdate() bool {
	return runtime.IsCurrentFiberTransitionUpdate()
}

// getParallelRegionSourceAtomValue reads one declared source atom value from the shared runtime store.
func getParallelRegionSourceAtomValue(parseSourceID string) (any, bool) {
	return runtime.GetGlobalRuntime().GetAtomValue(parseSourceID)
}
