//go:build !windows

package wails

import "github.com/monstercameron/GoWebComponents/v8/desktop"

// NewNativeBackend returns no backend on unverified platforms.
func NewNativeBackend() desktop.NativeBackend { return nil }
