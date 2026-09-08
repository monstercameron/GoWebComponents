//go:build !windows

package wails

import "github.com/monstercameron/GoWebComponents/v5/desktop"

// NewFileDialogs returns nil on unverified platforms so hosts advertise no support.
func NewFileDialogs() desktop.FileDialogBackend { return nil }
