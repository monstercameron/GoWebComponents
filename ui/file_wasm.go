//go:build js && wasm
// +build js,wasm

package ui

import "syscall/js"

// File wraps a browser File object exposed through a file input event.
type File struct {
	value js.Value
}

// FileFromJSValue wraps a browser File object that was obtained from direct JS interop.
func FileFromJSValue(value js.Value) File {
	return File{value: value}
}

// Name returns the browser file name.
func (f File) Name() string {
	if f.value.IsUndefined() || f.value.IsNull() {
		return ""
	}
	name := f.value.Get("name")
	if name.IsUndefined() || name.IsNull() {
		return ""
	}
	return name.String()
}

// Type returns the browser file MIME type when available.
func (f File) Type() string {
	if f.value.IsUndefined() || f.value.IsNull() {
		return ""
	}
	kind := f.value.Get("type")
	if kind.IsUndefined() || kind.IsNull() {
		return ""
	}
	return kind.String()
}

// Size returns the browser file size in bytes.
func (f File) Size() int64 {
	if f.value.IsUndefined() || f.value.IsNull() {
		return 0
	}
	size := f.value.Get("size")
	if size.IsUndefined() || size.IsNull() {
		return 0
	}
	return int64(size.Float())
}

// LastModified returns the browser file last-modified timestamp in milliseconds since epoch.
func (f File) LastModified() int64 {
	if f.value.IsUndefined() || f.value.IsNull() {
		return 0
	}
	lastModified := f.value.Get("lastModified")
	if lastModified.IsUndefined() || lastModified.IsNull() {
		return 0
	}
	return int64(lastModified.Float())
}

// JSValue returns the underlying browser File object.
func (f File) JSValue() js.Value {
	return f.value
}

// ExtractFiles returns the file list from an input or change event target.
func ExtractFiles(event Event) []File {
	jsEvent := event.JSValue()
	if jsEvent.IsUndefined() || jsEvent.IsNull() {
		return nil
	}
	target := jsEvent.Get("target")
	if target.IsUndefined() || target.IsNull() {
		return nil
	}
	files := target.Get("files")
	if files.IsUndefined() || files.IsNull() {
		return nil
	}
	length := files.Get("length")
	if length.IsUndefined() || length.IsNull() {
		return nil
	}
	count := length.Int()
	result := make([]File, 0, count)
	for index := 0; index < count; index++ {
		file := files.Index(index)
		if file.IsUndefined() || file.IsNull() {
			continue
		}
		result = append(result, FileFromJSValue(file))
	}
	return result
}