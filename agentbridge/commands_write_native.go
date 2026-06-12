//go:build !(js && wasm && gwcagent)

package agentbridge

import "encoding/json"

// writeNavigatePlatform is the stub used on every build except js+wasm+gwcagent.
// router-backed navigation is linked only into agent-mode wasm builds, so the
// bridge stays a no-op (and pulls in no router dependency) elsewhere; here it
// returns ErrorCodeBadPayload with an explanatory message.
func writeNavigatePlatform(parsePath string) (json.RawMessage, *EnvelopeError) {
	return nil, &EnvelopeError{
		Code:    ErrorCodeBadPayload,
		Message: "navigate: router is not available on this platform (js+wasm only)",
	}
}
