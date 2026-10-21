package interop

import (
	"errors"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v7/internal/errcode"
)

// TestInteropGWCCodeMapsEveryErrorCode verifies each interop failure class
// resolves to a registered GWC-INTEROP-* code with remediation text.
func TestInteropGWCCodeMapsEveryErrorCode(parseT *testing.T) {
	parseCodes := []ErrorCode{
		CodeUnavailable, CodePromiseRejected, CodeDisposed, CodeEncode, CodeDecode,
		CodeMissingExport, CodeNotFunction, CodeInvalid, CodeBlocked, CodeQuotaExceeded,
		CodeUnauthorized, CodeCancelled, CodeTimeout, CodeRemote,
	}
	for _, parseCode := range parseCodes {
		parseErr := &Error{Op: "Test", Code: parseCode, Err: errors.New("boom")}
		parseGWC := parseErr.GWCCode()
		if !strings.HasPrefix(string(parseGWC), "GWC-INTEROP-") {
			parseT.Fatalf("code %q mapped to %q, want a GWC-INTEROP-* code", parseCode, parseGWC)
		}
		if parseEntry, parseOk := errcode.Lookup(parseGWC); !parseOk || parseEntry.Remediation == "" {
			parseT.Fatalf("code %q mapped to %q which is not registered with remediation", parseCode, parseGWC)
		}
	}
	if (*Error)(nil).GWCCode() != "" || (&Error{Code: "bogus"}).GWCCode() != "" {
		parseT.Fatalf("nil and unknown interop errors must have no GWC code")
	}
	parseMessage := (&Error{Op: "Open", Code: CodeBlocked, Err: errors.New("x")}).Error()
	if !strings.Contains(parseMessage, "error-codes.md#gwc-interop-persistent-store-blocked") || strings.Contains(parseMessage, "..") {
		parseT.Fatalf("blocked message should cite the registry page once: %q", parseMessage)
	}
}
