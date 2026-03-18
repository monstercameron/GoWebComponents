//go:build !js || !wasm
// +build !js !wasm

package ui

// File is the native no-op counterpart of the browser File wrapper.
type File struct{}

func (File) Name() string { return "" }

func (File) Type() string { return "" }

func (File) Size() int64 { return 0 }

func (File) LastModified() int64 { return 0 }

// ExtractFiles returns no files outside the browser wasm runtime.
func ExtractFiles(event Event) []File {
	_ = event
	return nil
}