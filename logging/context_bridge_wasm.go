//go:build js && wasm

package logging

import "context"

// resolveExternalContextDetails returns no external context metadata on js/wasm builds.
func resolveExternalContextDetails(parseCtx context.Context) logContextDetails {
	_ = parseCtx
	return logContextDetails{}
}
