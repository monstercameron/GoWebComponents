//go:build !js || !wasm

package fetch

import (
	"context"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// TestScopeCacheKey documents the prefix convention that keeps unrelated
// components out of each other's cache entries in the app-global namespace.
func TestScopeCacheKey(parseT *testing.T) {
	if parseGot := ScopeCacheKey("billing", "invoices"); parseGot != "billing:invoices" {
		parseT.Fatalf("expected billing:invoices, got %q", parseGot)
	}
	if parseGot := ScopeCacheKey("", "users", "  ", "list"); parseGot != "users:list" {
		parseT.Fatalf("expected empty segments dropped, got %q", parseGot)
	}
	// Two features that both pick the bare key "list" collide; scoping splits
	// them into distinct keys.
	parseA := ScopeCacheKey("billing", "list")
	parseB := ScopeCacheKey("inventory", "list")
	if parseA == parseB {
		parseT.Fatalf("expected scoped keys to differ, both were %q", parseA)
	}
}

// TestUseCachedResourceConflictingTypesWarns pins the documented safety net for
// the global-namespace sharp edge: requesting one key with two value types is
// surfaced as a diagnostic warning, not silently shared.
func TestUseCachedResourceConflictingTypesWarns(parseT *testing.T) {
	runtime.SetCurrentFiber(&runtime.Fiber{})
	defer runtime.SetCurrentFiber(nil)
	runtime.ClearDiagnostics()

	parseKey := ScopeCacheKey("conflicttest", "shared")
	_ = UseCachedResource(parseKey, func(parseCtx context.Context) (string, error) {
		return "", nil
	})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	_ = UseCachedResource(parseKey, func(parseCtx context.Context) (int, error) {
		return 0, nil
	})

	parseFound := false
	for _, parseDiag := range runtime.GetDiagnostics() {
		if strings.Contains(parseDiag.Message, "conflicting value types") && strings.Contains(parseDiag.Message, parseKey) {
			parseFound = true
			break
		}
	}
	if !parseFound {
		parseT.Fatalf("expected a conflicting-value-types diagnostic for key %q, got %+v", parseKey, runtime.GetDiagnostics())
	}
}
