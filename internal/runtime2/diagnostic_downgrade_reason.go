package runtime2

import (
	"fmt"
	"strings"
)

// DiagnosticDowngradePath identifies one transport path that can downgrade.
type DiagnosticDowngradePath string

const (
	diagnosticDowngradePathInvalid DiagnosticDowngradePath = ""

	DiagnosticDowngradePathSharedMemory DiagnosticDowngradePath = "shared-memory"
	DiagnosticDowngradePathBinary       DiagnosticDowngradePath = "binary"
)

// DiagnosticDowngradeReason stores structured downgrade diagnostics for binary and shared-memory fallback paths.
type DiagnosticDowngradeReason struct {
	Path   DiagnosticDowngradePath `json:"path,omitempty"`
	Reason string                  `json:"reason,omitempty"`
}

// ParseDiagnosticDowngradePath validates one downgrade path identifier.
func ParseDiagnosticDowngradePath(parseRaw string) (DiagnosticDowngradePath, error) {
	parsePath := DiagnosticDowngradePath(parseRaw)
	switch parsePath {
	case DiagnosticDowngradePathSharedMemory, DiagnosticDowngradePathBinary:
		return parsePath, nil
	case diagnosticDowngradePathInvalid:
		return "", fmt.Errorf("runtime2: diagnostic downgrade path is required")
	default:
		return "", fmt.Errorf("runtime2: diagnostic downgrade path %q is unsupported", parseRaw)
	}
}

// ValidateDiagnosticDowngradeReason verifies downgrade diagnostics align with binary and shared-memory fallback reason sets.
func ValidateDiagnosticDowngradeReason(parseDowngradeReason DiagnosticDowngradeReason) error {
	parsePath, parsePathErr := ParseDiagnosticDowngradePath(string(parseDowngradeReason.Path))
	if parsePathErr != nil {
		return parsePathErr
	}
	parseReason := strings.TrimSpace(parseDowngradeReason.Reason)
	if parseReason == "" {
		return fmt.Errorf("runtime2: diagnostic downgrade reason is required")
	}
	if parseReason != parseDowngradeReason.Reason {
		return fmt.Errorf("runtime2: diagnostic downgrade reason must not contain surrounding whitespace")
	}
	switch parsePath {
	case DiagnosticDowngradePathSharedMemory:
		switch parseReason {
		case "unavailable", "invalid-shared-page", "capability-contract-disallow":
			return nil
		default:
			return fmt.Errorf("runtime2: diagnostic shared-memory downgrade reason %q is unsupported", parseReason)
		}
	case DiagnosticDowngradePathBinary:
		switch parseReason {
		case "encode-failed", "decode-failed", "capability-contract-disallow":
			return nil
		default:
			return fmt.Errorf("runtime2: diagnostic binary downgrade reason %q is unsupported", parseReason)
		}
	default:
		return fmt.Errorf("runtime2: diagnostic downgrade path %q is unsupported", parsePath)
	}
}
