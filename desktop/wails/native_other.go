//go:build !windows

package wails

import "github.com/monstercameron/GoWebComponents/v5/desktop"

// NewNativeBackend returns no backend on unverified platforms.
func NewNativeBackend() desktop.NativeBackend { return nil }
