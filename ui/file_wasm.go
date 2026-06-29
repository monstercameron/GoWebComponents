//go:build js && wasm

package ui

import "syscall/js"

// File wraps a browser File object exposed through a file input event.
type File struct {
	value js.Value
}

// WrapJSFile wraps a browser File object that was obtained from direct JS interop.
func WrapJSFile(parseValue js.Value) File {
	return File{value: parseValue}
}

// Name returns the browser file name.
func (parseF File) Name() string {
	if parseF.value.IsUndefined() || parseF.value.IsNull() {
		return ""
	}
	parseName := parseF.value.Get("name")
	if parseName.IsUndefined() || parseName.IsNull() {
		return ""
	}
	return parseName.String()
}

// Type returns the browser file MIME type when available.
func (parseF File) Type() string {
	if parseF.value.IsUndefined() || parseF.value.IsNull() {
		return ""
	}
	parseKind := parseF.value.Get("type")
	if parseKind.IsUndefined() || parseKind.IsNull() {
		return ""
	}
	return parseKind.String()
}

// Size returns the browser file size in bytes.
func (parseF File) Size() int64 {
	if parseF.value.IsUndefined() || parseF.value.IsNull() {
		return 0
	}
	parseSize := parseF.value.Get("size")
	if parseSize.IsUndefined() || parseSize.IsNull() {
		return 0
	}
	return int64(parseSize.Float())
}

// LastModified returns the browser file last-modified timestamp in milliseconds since epoch.
func (parseF File) LastModified() int64 {
	if parseF.value.IsUndefined() || parseF.value.IsNull() {
		return 0
	}
	parseLastModified := parseF.value.Get("lastModified")
	if parseLastModified.IsUndefined() || parseLastModified.IsNull() {
		return 0
	}
	return int64(parseLastModified.Float())
}

// JSValue returns the underlying browser File object.
func (parseF File) JSValue() js.Value {
	return parseF.value
}

// GetFiles returns the file list from an input or change event target.
func GetFiles(parseEvent Event) []File {
	parseJsEvent := parseEvent.JSValue()
	if parseJsEvent.IsUndefined() || parseJsEvent.IsNull() {
		return nil
	}
	parseTarget := parseJsEvent.Get("target")
	if parseTarget.IsUndefined() || parseTarget.IsNull() {
		return nil
	}
	parseFiles := parseTarget.Get("files")
	if parseFiles.IsUndefined() || parseFiles.IsNull() {
		return nil
	}
	parseLength := parseFiles.Get("length")
	if parseLength.IsUndefined() || parseLength.IsNull() {
		return nil
	}
	parseCount := parseLength.Int()
	parseResult := make([]File, 0, parseCount)
	for parseIndex := 0; parseIndex < parseCount; parseIndex++ {
		parseFile := parseFiles.Index(parseIndex)
		if parseFile.IsUndefined() || parseFile.IsNull() {
			continue
		}
		parseResult = append(parseResult, WrapJSFile(parseFile))
	}
	return parseResult
}
