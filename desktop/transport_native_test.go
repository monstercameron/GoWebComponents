//go:build !js || !wasm

package desktop

import (
	"context"
	"github.com/monstercameron/GoWebComponents/v6/interop"
	"testing"
)

// TestConnectIsUnavailableOnNative verifies SSR does not discover a native Wails host.
func TestConnectIsUnavailableOnNative(parseT *testing.T) {
	_, parseErr := Connect()
	if !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseT.Fatalf("Connect error = %v", parseErr)
	}
}

// TestFileWorkflowsUnavailableOnNative verifies web/SSR callers need no native dependency.
func TestFileWorkflowsUnavailableOnNative(parseT *testing.T) {
	for _, parseCall := range []func(context.Context, FileDialogOptions) (FileSelection, error){OpenFile, OpenFiles, OpenDirectory, SaveFile} {
		if _, parseErr := parseCall(context.Background(), FileDialogOptions{}); !interop.IsCode(parseErr, interop.CodeUnavailable) {
			parseT.Fatalf("unexpected error: %v", parseErr)
		}
	}
}
