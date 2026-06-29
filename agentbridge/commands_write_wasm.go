//go:build js && wasm && gwcagent

package agentbridge

import (
	"encoding/json"

	"github.com/monstercameron/GoWebComponents/v4/router"
)

// writeNavigatePlatform calls router.Navigate on wasm builds where the global
// router is available. The router package's Navigate function panics if the
// global router is nil; such panics are not expected in normal usage and are
// allowed to propagate as unhandled errors.
func writeNavigatePlatform(parsePath string) (json.RawMessage, *EnvelopeError) {
	router.Navigate(parsePath)
	return writeEncodeOK()
}
