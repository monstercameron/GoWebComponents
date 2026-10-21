//go:build !windows

package wails

import "github.com/monstercameron/GoWebComponents/v7/desktop"

// NewFileDialogs returns nil on unverified platforms so hosts advertise no support.
func NewFileDialogs() desktop.FileDialogBackend { return nil }
