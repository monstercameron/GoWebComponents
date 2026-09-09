//go:build !windows

package wails

import (
	"errors"

	"github.com/monstercameron/GoWebComponents/v6/desktop"
)

// NewNativeBackendWithWindowTemplates reports that the Windows-only adapter is unavailable.
func NewNativeBackendWithWindowTemplates(map[string]WindowTemplate) (desktop.NativeBackend, error) {
	return nil, errors.New("Wails child windows are unavailable on this platform")
}
