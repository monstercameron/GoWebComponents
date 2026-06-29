package render

import (
	"strings"
	"testing"
)

// TestRenderConvenienceHelpersExposeBaseContracts verifies the thin render wrapper helpers.
func TestRenderConvenienceHelpersExposeBaseContracts(parseT *testing.T) {
	if NewResourceController[string]() == nil {
		parseT.Fatal("expected resource controller wrapper to return a value")
	}
	if strings.TrimSpace(ParallelSafetyContract()) == "" {
		parseT.Fatal("expected non-empty parallel safety contract")
	}

	for _, parseErr := range []error{
		BuildFailureError(FailureCodeLoaderFailure, "loader broke"),
		BuildHydrationMismatchError("/app", "mismatch"),
		BuildLoaderFailureError("/catalog", "timeout"),
		BuildRouteGuardFailureError("/admin", "forbidden"),
		BuildCacheConflictError("product"),
		BuildOfflineReplayError("transfer", "offline"),
	} {
		if parseErr == nil || strings.TrimSpace(parseErr.Error()) == "" {
			parseT.Fatalf("expected non-empty error from helper, got %v", parseErr)
		}
	}
}
