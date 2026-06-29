//go:build !js || !wasm

package runtime

import "fmt"

// writeInvokeHandlerPlatform is a native stub. On non-wasm builds the
// platform-specific handler types (func(js.Value), func(GoEvent)) do not
// exist, so any value reaching this point has an unsupported type.
func writeInvokeHandlerPlatform(parseHandler any, parseEventName string, parseRef string) error {
	return fmt.Errorf("handler %q on ref %q has unsupported type %T: %w",
		parseEventName, parseRef, parseHandler, ErrAgentNoHandler)
}
