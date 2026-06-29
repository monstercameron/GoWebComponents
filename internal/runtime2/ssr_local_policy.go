package runtime2

import "fmt"

// SSRShellAttachPolicy identifies runtime2 SSR attach policy modes.
type SSRShellAttachPolicy string

const (
	// SSRShellAttachPolicyLocalOnly keeps runtime2 SSR strictly local-only in the first slice.
	SSRShellAttachPolicyLocalOnly SSRShellAttachPolicy = "local-only"
)

// ParseSSRShellAttachPolicy validates one runtime2 SSR attach policy value.
func ParseSSRShellAttachPolicy(parseRaw string) (SSRShellAttachPolicy, error) {
	parsePolicy := SSRShellAttachPolicy(parseRaw)
	switch parsePolicy {
	case SSRShellAttachPolicyLocalOnly:
		return parsePolicy, nil
	case "":
		return "", fmt.Errorf("runtime2: SSR shell attach policy is required")
	default:
		return "", fmt.Errorf("runtime2: SSR shell attach policy %q is unsupported", parseRaw)
	}
}

// GetSSRShellAttachPolicy reports the runtime2 SSR attach policy for the current slice.
func GetSSRShellAttachPolicy() SSRShellAttachPolicy {
	return SSRShellAttachPolicyLocalOnly
}
