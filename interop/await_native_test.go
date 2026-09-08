//go:build !(js && wasm)

package interop_test

import (
	"context"
	"errors"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// TestAwaitNativeStubsReturnUnavailable verifies the Promise bridge surface fails
// loudly (structured CodeUnavailable) on non-browser builds rather than silently
// succeeding — so native unit tests of code paths that call Await catch misuse.
func TestAwaitNativeStubsReturnUnavailable(t *testing.T) {
	var parseValue interop.Value

	if _, parseErr := parseValue.Await(context.Background()); parseErr == nil {
		t.Fatal("expected Await to be unavailable on native")
	} else if !errorHasCode(parseErr, interop.CodeUnavailable) {
		t.Fatalf("expected CodeUnavailable, got %v", parseErr)
	}

	if _, parseErr := parseValue.AwaitCall(context.Background(), "estimate"); parseErr == nil {
		t.Fatal("expected AwaitCall to be unavailable on native")
	}

	if _, parseErr := interop.RequestNotificationPermission(context.Background()); parseErr == nil {
		t.Fatal("expected RequestNotificationPermission to be unavailable on native")
	}

	if parseErr := interop.PostNotification("title", "body"); parseErr == nil {
		t.Fatal("expected PostNotification to be unavailable on native")
	}

	if parseState, parseErr := interop.NotificationPermissionState(); parseErr == nil {
		t.Fatal("expected NotificationPermissionState to be unavailable on native")
	} else if parseState != interop.NotificationDefault {
		t.Fatalf("expected default permission on native, got %q", parseState)
	}
}

// TestAwaitNilContextNative confirms a nil context does not panic the stub.
func TestAwaitNilContextNative(t *testing.T) {
	var parseValue interop.Value
	//nolint:staticcheck // intentionally passing nil to prove it is tolerated.
	if _, parseErr := parseValue.Await(nil); parseErr == nil {
		t.Fatal("expected unavailable error")
	}
}

func errorHasCode(parseErr error, parseCode interop.ErrorCode) bool {
	var parseTyped *interop.Error
	if errors.As(parseErr, &parseTyped) {
		return parseTyped.Code == parseCode
	}
	return false
}
