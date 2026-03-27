package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestValidateWorkerRenderableHostTagAcceptsAllowedTag verifies first-slice allowed host tags pass validation.
func TestValidateWorkerRenderableHostTagAcceptsAllowedTag(parseT *testing.T) {
	if parseErr := runtime2.ValidateWorkerRenderableHostTag("div"); parseErr != nil {
		parseT.Fatalf("ValidateWorkerRenderableHostTag(div) returned error: %v", parseErr)
	}
}

// TestValidateWorkerRenderableHostTagRejectsUnsupportedTag verifies unsupported host tags are rejected.
func TestValidateWorkerRenderableHostTagRejectsUnsupportedTag(parseT *testing.T) {
	if parseErr := runtime2.ValidateWorkerRenderableHostTag("iframe"); parseErr == nil {
		parseT.Fatal("expected unsupported host tag to fail")
	}
}

// TestValidateWorkerRenderableHostTagRejectsWhitespaceWrappedTag verifies surrounding whitespace is rejected.
func TestValidateWorkerRenderableHostTagRejectsWhitespaceWrappedTag(parseT *testing.T) {
	if parseErr := runtime2.ValidateWorkerRenderableHostTag(" div "); parseErr == nil {
		parseT.Fatal("expected whitespace-wrapped host tag to fail")
	}
}
