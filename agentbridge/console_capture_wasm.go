//go:build js && wasm && gwcagent

package agentbridge

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"
)

type consoleCaptureRelease func()

func installConsoleCapture(parseClient *BridgeClient) consoleCaptureRelease {
	parseGlobal := js.Global()
	parseConsole := parseGlobal.Get("console")
	parseWindow := parseGlobal.Get("window")
	parseReleaseFns := []func(){}

	if parseConsole.Type() == js.TypeObject {
		parseOriginalError := parseConsole.Get("error")
		if parseOriginalError.Type() == js.TypeFunction {
			parseFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
				parsePayload := consoleCapturePayload("console.error", "", parseArgs)
				_ = parseClient.SendEvent("", "console.error", parsePayload)
				return parseOriginalError.Invoke(parseArgs...)
			})
			parseConsole.Set("error", parseFn)
			parseReleaseFns = append(parseReleaseFns, func() {
				parseConsole.Set("error", parseOriginalError)
				parseFn.Release()
			})
		}
	}

	if parseWindow.Type() == js.TypeObject {
		parseErrorFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseMessage := ""
			if len(parseArgs) > 0 {
				parseMessage = jsValueSummary(parseArgs[0])
			}
			_ = parseClient.SendEvent("", "window.onerror", consoleCapturePayload("window.onerror", parseMessage, parseArgs))
			return nil
		})
		parseWindow.Call("addEventListener", "error", parseErrorFn)
		parseReleaseFns = append(parseReleaseFns, func() {
			parseWindow.Call("removeEventListener", "error", parseErrorFn)
			parseErrorFn.Release()
		})

		parseRejectFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseMessage := ""
			if len(parseArgs) > 0 {
				parseReason := parseArgs[0].Get("reason")
				if parseReason.Type() != js.TypeUndefined {
					parseMessage = jsValueSummary(parseReason)
				}
			}
			_ = parseClient.SendEvent("", "unhandledrejection", consoleCapturePayload("unhandledrejection", parseMessage, parseArgs))
			return nil
		})
		parseWindow.Call("addEventListener", "unhandledrejection", parseRejectFn)
		parseReleaseFns = append(parseReleaseFns, func() {
			parseWindow.Call("removeEventListener", "unhandledrejection", parseRejectFn)
			parseRejectFn.Release()
		})
	}

	return func() {
		for parseIdx := len(parseReleaseFns) - 1; parseIdx >= 0; parseIdx-- {
			func() {
				defer func() { recover() }()
				parseReleaseFns[parseIdx]()
			}()
		}
	}
}

func consoleCapturePayload(parseSource string, parseMessage string, parseArgs []js.Value) json.RawMessage {
	parseArgsText := make([]string, 0, len(parseArgs))
	for _, parseArg := range parseArgs {
		parseArgsText = append(parseArgsText, jsValueSummary(parseArg))
	}
	if parseMessage == "" && len(parseArgsText) > 0 {
		parseMessage = parseArgsText[0]
	}
	parsePayload, _ := json.Marshal(map[string]any{
		"source":    parseSource,
		"severity":  "error",
		"message":   parseMessage,
		"arguments": parseArgsText,
		"time":      time.Now().UTC().Format(time.RFC3339Nano),
	})
	return parsePayload
}

func jsValueSummary(parseValue js.Value) (parseResult string) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseResult = fmt.Sprintf("%v", parseRecovered)
		}
	}()
	if parseValue.Type() == js.TypeString {
		return parseValue.String()
	}
	if parseValue.Type() == js.TypeObject {
		parseMessage := parseValue.Get("message")
		if parseMessage.Type() == js.TypeString && parseMessage.String() != "" {
			return parseMessage.String()
		}
	}
	return parseValue.String()
}
