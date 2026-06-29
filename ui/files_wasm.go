//go:build js && wasm

package ui

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
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
	var parseChangeFn js.Func
	parseChangeFn = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		defer runtime.RecoverContainedPanic("ui", "PickFile change")
		defer parseChangeFn.Release()
		parseFiles := parseInput.Get("files")
		if !parseFiles.Truthy() || parseFiles.Get("length").Int() == 0 {
			return nil
		}
		parseFile := parseFiles.Call("item", 0)
		parseName := parseFile.Get("name").String()
		parseReader := js.Global().Get("FileReader").New()
		var parseLoadFn js.Func
		parseLoadFn = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) any {
			defer runtime.RecoverContainedPanic("ui", "PickFile read")
			defer parseLoadFn.Release()
			parseBuf := parseReader.Get("result")
			parseBytes := js.Global().Get("Uint8Array").New(parseBuf)
			parseData := make([]byte, parseBytes.Get("length").Int())
			js.CopyBytesToGo(parseData, parseBytes)
			parseOnPick(PickedFile{Name: parseName, Data: parseData})
			return nil
		})
		parseReader.Call("addEventListener", "load", parseLoadFn)
		parseReader.Call("readAsArrayBuffer", parseFile)
		return nil
	})
	parseInput.Call("addEventListener", "change", parseChangeFn)
	parseInput.Call("click")
}
