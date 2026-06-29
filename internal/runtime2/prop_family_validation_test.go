package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestValidateWorkerRenderablePropFamilyAcceptsAllowedFamily verifies first-slice allowed prop families pass validation.
func TestValidateWorkerRenderablePropFamilyAcceptsAllowedFamily(parseT *testing.T) {
	if parseErr := runtime2.ValidateWorkerRenderablePropFamily("class"); parseErr != nil {
		parseT.Fatalf("ValidateWorkerRenderablePropFamily(class) returned error: %v", parseErr)
	}
}

// TestValidateWorkerRenderablePropFamilyRejectsUnsupportedFamily verifies unsupported prop families are rejected.
func TestValidateWorkerRenderablePropFamilyRejectsUnsupportedFamily(parseT *testing.T) {
	if parseErr := runtime2.ValidateWorkerRenderablePropFamily("event"); parseErr == nil {
		parseT.Fatal("expected unsupported prop family to fail")
	}
}

// TestValidateWorkerRenderablePropFamilyRejectsWhitespaceWrappedFamily verifies surrounding whitespace is rejected.
func TestValidateWorkerRenderablePropFamilyRejectsWhitespaceWrappedFamily(parseT *testing.T) {
	if parseErr := runtime2.ValidateWorkerRenderablePropFamily(" data "); parseErr == nil {
		parseT.Fatal("expected whitespace-wrapped prop family to fail")
	}
}
