package runtime2

import "fmt"

// DiagnosticEventKind identifies one stable runtime2 lifecycle diagnostic event kind.
type DiagnosticEventKind string

const (
	// DiagnosticEventKindMount identifies one mount lifecycle diagnostic.
	DiagnosticEventKindMount DiagnosticEventKind = "mount"
	// DiagnosticEventKindUpdate identifies one update lifecycle diagnostic.
	DiagnosticEventKindUpdate DiagnosticEventKind = "update"
	// DiagnosticEventKindCancel identifies one cancel lifecycle diagnostic.
	DiagnosticEventKindCancel DiagnosticEventKind = "cancel"
	// DiagnosticEventKindDispose identifies one dispose lifecycle diagnostic.
	DiagnosticEventKindDispose DiagnosticEventKind = "dispose"
	// DiagnosticEventKindRestart identifies one restart lifecycle diagnostic.
	DiagnosticEventKindRestart DiagnosticEventKind = "restart"
	// DiagnosticEventKindPatchReady identifies one patch-ready lifecycle diagnostic.
	DiagnosticEventKindPatchReady DiagnosticEventKind = "patch-ready"
	// DiagnosticEventKindFallback identifies one fallback lifecycle diagnostic.
	DiagnosticEventKindFallback DiagnosticEventKind = "fallback"
	// DiagnosticEventKindRepair identifies one repair lifecycle diagnostic.
	DiagnosticEventKindRepair DiagnosticEventKind = "repair"
)

// ParseDiagnosticEventKind validates one raw diagnostic event kind value.
func ParseDiagnosticEventKind(parseRaw string) (DiagnosticEventKind, error) {
	parseKind := DiagnosticEventKind(parseRaw)
	switch parseKind {
	case DiagnosticEventKindMount,
		DiagnosticEventKindUpdate,
		DiagnosticEventKindCancel,
		DiagnosticEventKindDispose,
		DiagnosticEventKindRestart,
		DiagnosticEventKindPatchReady,
		DiagnosticEventKindFallback,
		DiagnosticEventKindRepair:
		return parseKind, nil
	case "":
		return "", fmt.Errorf("runtime2: diagnostic event kind is required")
	default:
		return "", fmt.Errorf("runtime2: diagnostic event kind %q is unsupported", parseRaw)
	}
}

// GetDiagnosticEventKinds returns the stable lifecycle diagnostic event kinds in deterministic order.
func GetDiagnosticEventKinds() []DiagnosticEventKind {
	return []DiagnosticEventKind{
		DiagnosticEventKindMount,
		DiagnosticEventKindUpdate,
		DiagnosticEventKindCancel,
		DiagnosticEventKindDispose,
		DiagnosticEventKindRestart,
		DiagnosticEventKindPatchReady,
		DiagnosticEventKindFallback,
		DiagnosticEventKindRepair,
	}
}
