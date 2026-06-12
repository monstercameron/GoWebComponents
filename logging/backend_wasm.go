//go:build js && wasm

package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"syscall/js"
)

// writeStructuredContext writes one structured log object to the browser console when available.
func writeStructuredContext(parseLogContext context.Context, parseLogLevel, parseLogScope, parseLogMessage string, parseLogFields map[string]interface{}) {
	parseRecord := buildLogRecord(parseLogContext, parseLogLevel, parseLogScope, parseLogMessage, parseLogFields)
	parseConsole := js.Global().Get("console")
	if parseConsole.Truthy() {
		parseMethodName := buildLogLevelDetails(parseLogLevel).consoleMethod
		parseMethod := parseConsole.Get(parseMethodName)
		if parseMethod.Type() != js.TypeFunction {
			parseMethod = parseConsole.Get("log")
		}
		if parseMethod.Type() == js.TypeFunction {
			parseMethod.Invoke(js.ValueOf(parseRecord))
			return
		}
	}

	parseEncodedRecord, parseErr := json.Marshal(parseRecord)
	if parseErr != nil {
		fmt.Printf("{\"level\":\"error\",\"message\":\"logging marshal failed\",\"error\":%q}\n", parseErr.Error())
		return
	}
	fmt.Println(string(parseEncodedRecord))
}
