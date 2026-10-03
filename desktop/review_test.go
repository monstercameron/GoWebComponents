package desktop

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// parsePanickingBackend embeds the clipboard fake but panics on write, the way a
// faulty host adapter would.
type parsePanickingBackend struct{ *parseNativeFake }

// ClipboardWrite panics to model a backend defect.
func (parseBackend parsePanickingBackend) ClipboardWrite(context.Context, ClipboardWriteRequest) error {
	panic("adapter exploded: secret-internal-path")
}

// TestNativeExecuteContainsBackendPanic pins the "always returns a wire-safe
// reply" contract: a panicking adapter becomes a typed failure envelope, and its
// panic text (which may name internals) does not cross the wire.
func TestNativeExecuteContainsBackendPanic(parseTest *testing.T) {
	parseFake := &parseNativeFake{parseFeatures: []Feature{Clipboard}}
	parsePolicy, _ := ParseFeaturePolicy("all")
	parseHost := NewNativeHost(parsePanickingBackend{parseFake}, parsePolicy)
	parseArgs, _ := json.Marshal(ClipboardWriteRequest{Text: "hi"})

	parseReply := parseHost.Execute(context.Background(), NativeRequest{Version: NativeContractVersion, Method: ClipboardWriteMethod, Args: parseArgs})
	if parseReply.Code != interop.CodeRemote {
		parseTest.Fatalf("reply code = %q, want %q", parseReply.Code, interop.CodeRemote)
	}
	if parseReply.Message == "" || len(parseReply.Data) != 0 {
		parseTest.Fatalf("expected a failure envelope, got %+v", parseReply)
	}
	if strings.Contains(parseReply.Message, "secret") {
		parseTest.Fatalf("panic text leaked to the wire: %q", parseReply.Message)
	}
}
