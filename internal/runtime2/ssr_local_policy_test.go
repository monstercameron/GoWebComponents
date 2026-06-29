package runtime2

import "testing"

// TestGetSSRShellAttachPolicyLocalOnly verifies runtime2 keeps SSR local-only in the first slice.
func TestGetSSRShellAttachPolicyLocalOnly(parseT *testing.T) {
	if GetSSRShellAttachPolicy() != SSRShellAttachPolicyLocalOnly {
		parseT.Fatalf("GetSSRShellAttachPolicy() = %q, want %q", GetSSRShellAttachPolicy(), SSRShellAttachPolicyLocalOnly)
	}
}

// TestParseSSRShellAttachPolicyRejectsNonLocalModes verifies non-local SSR attach policies are rejected in the first slice.
func TestParseSSRShellAttachPolicyRejectsNonLocalModes(parseT *testing.T) {
	if _, parseErr := ParseSSRShellAttachPolicy("worker-first"); parseErr == nil {
		parseT.Fatal("expected unsupported SSR shell attach policy to fail")
	}
}

// TestParseSSRShellAttachPolicyAcceptsLocalOnlyAndRejectsEmpty verifies the current SSR attach policy parses successfully while empty input remains invalid.
func TestParseSSRShellAttachPolicyAcceptsLocalOnlyAndRejectsEmpty(parseT *testing.T) {
	parsePolicy, parseErr := ParseSSRShellAttachPolicy(string(SSRShellAttachPolicyLocalOnly))
	if parseErr != nil {
		parseT.Fatalf("ParseSSRShellAttachPolicy(local-only) returned error: %v", parseErr)
	}
	if parsePolicy != SSRShellAttachPolicyLocalOnly {
		parseT.Fatalf("ParseSSRShellAttachPolicy(local-only) = %q, want %q", parsePolicy, SSRShellAttachPolicyLocalOnly)
	}
	if _, parseErr := ParseSSRShellAttachPolicy(""); parseErr == nil {
		parseT.Fatal("expected empty SSR shell attach policy to fail")
	}
}
