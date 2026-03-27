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
