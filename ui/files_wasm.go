//go:build js && wasm

package ui

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

func defaultDownload(parseData []byte, parseName, parseMime string) {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseArray := js.Global().Get("Uint8Array").New(len(parseData))
	js.CopyBytesToJS(parseArray, parseData)
	parseParts := js.Global().Get("Array").New(1)
	parseParts.SetIndex(0, parseArray)
	parseOpts := js.Global().Get("Object").New()
	if parseMime != "" {
		parseOpts.Set("type", parseMime)
	}
	parseBlob := js.Global().Get("Blob").New(parseParts, parseOpts)
	parseURL := js.Global().Get("URL").Call("createObjectURL", parseBlob)
	parseAnchor := parseDocument.Call("createElement", "a")
	parseAnchor.Set("href", parseURL)
	parseAnchor.Set("download", parseName)
	parseAnchor.Call("click")
	js.Global().Get("URL").Call("revokeObjectURL", parseURL)
}

func defaultPickFile(parseAccept string, parseOnPick func(PickedFile)) {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseInput := parseDocument.Call("createElement", "input")
	parseInput.Set("type", "file")
	if parseAccept != "" {
		parseInput.Set("accept", parseAccept)
	}
	// Both input listeners release both funcs exactly once: cancelling the
	// native dialog fires "cancel" (not "change"), which previously leaked
	// parseChangeFn on every cancelled pick.
	var parseChangeFn, parseCancelFn js.Func
	isInputReleased := false
	parseReleaseInputFns := func() {
		if isInputReleased {
			return
		}
		isInputReleased = true
		parseChangeFn.Release()
		parseCancelFn.Release()
	}
	parseChangeFn = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		defer runtime.RecoverContainedPanic("ui", "PickFile change")
		defer parseReleaseInputFns()
		parseFiles := parseInput.Get("files")
		if !parseFiles.Truthy() || parseFiles.Get("length").Int() == 0 {
			return nil
		}
		parseFile := parseFiles.Call("item", 0)
		parseName := parseFile.Get("name").String()
		parseReader := js.Global().Get("FileReader").New()
		// load/error release both reader funcs once; a failed read previously
		// leaked parseLoadFn and silently never reported anything.
		var parseLoadFn, parseErrorFn js.Func
		isReaderReleased := false
		parseReleaseReaderFns := func() {
			if isReaderReleased {
				return
			}
			isReaderReleased = true
			parseLoadFn.Release()
			parseErrorFn.Release()
		}
		parseLoadFn = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) any {
			defer runtime.RecoverContainedPanic("ui", "PickFile read")
			defer parseReleaseReaderFns()
			parseBuf := parseReader.Get("result")
			parseBytes := js.Global().Get("Uint8Array").New(parseBuf)
			parseData := make([]byte, parseBytes.Get("length").Int())
			js.CopyBytesToGo(parseData, parseBytes)
			parseOnPick(PickedFile{Name: parseName, Data: parseData})
			return nil
		})
		parseErrorFn = js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) any {
			defer runtime.RecoverContainedPanic("ui", "PickFile read error")
			defer parseReleaseReaderFns()
			runtime.ReportDiagnostic("ui", runtime.DiagnosticWarning, "PickFile failed to read "+parseName)
			return nil
		})
		parseReader.Call("addEventListener", "load", parseLoadFn)
		parseReader.Call("addEventListener", "error", parseErrorFn)
		parseReader.Call("readAsArrayBuffer", parseFile)
		return nil
	})
	parseCancelFn = js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) any {
		defer runtime.RecoverContainedPanic("ui", "PickFile cancel")
		defer parseReleaseInputFns()
		return nil
	})
	parseInput.Call("addEventListener", "change", parseChangeFn)
	parseInput.Call("addEventListener", "cancel", parseCancelFn)
	parseInput.Call("click")
}
