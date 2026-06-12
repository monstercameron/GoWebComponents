//go:build !js || !wasm

package ui

// File is the native no-op counterpart of the browser File wrapper.
type File struct{}

// Name is a core package helper.
func (File) Name() string { return "" }

// Type is a core package helper.
func (File) Type() string { return "" }

// Size is a core package helper.
func (File) Size() int64 { return 0 }

// LastModified is a core package helper.
func (File) LastModified() int64 { return 0 }

// GetFiles returns no files outside the browser wasm runtime.
func GetFiles(parseFileEvent Event) []File {
	_ = parseFileEvent
	return nil
}
