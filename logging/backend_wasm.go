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
			// Lead with the message string: the structured record is a JS
			// object whose console preview shows only a few properties in
			// nondeterministic (Go map iteration) order, so without this the
			// message may be invisible in console text — unreadable in
			// devtools and unmatchable by browser-test log assertions.
			if parseLogMessage != "" {
				parseMethod.Invoke(js.ValueOf(parseLogMessage), js.ValueOf(parseRecord))
			} else {
				parseMethod.Invoke(js.ValueOf(parseRecord))
			}
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
