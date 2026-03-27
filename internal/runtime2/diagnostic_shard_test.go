package runtime2

import "testing"

// TestValidateDiagnosticShardIDAcceptsNormalizedShard verifies normalized shard identities pass diagnostic validation.
func TestValidateDiagnosticShardIDAcceptsNormalizedShard(parseT *testing.T) {
	if parseErr := ValidateDiagnosticShardID(SchedulerShardID("shard-3")); parseErr != nil {
		parseT.Fatalf("ValidateDiagnosticShardID(valid) returned error: %v", parseErr)
	}
}

// TestValidateDiagnosticShardIDRejectsWhitespaceWrappedShard verifies shard identities with surrounding whitespace are rejected.
func TestValidateDiagnosticShardIDRejectsWhitespaceWrappedShard(parseT *testing.T) {
	if parseErr := ValidateDiagnosticShardID(SchedulerShardID(" shard-3 ")); parseErr == nil {
		parseT.Fatal("expected whitespace-wrapped shard ID to fail")
	}
}
