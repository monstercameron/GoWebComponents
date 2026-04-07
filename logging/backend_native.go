//go:build !js || !wasm
// +build !js !wasm

package logging

import (
	"context"
	"encoding/json"
	"fmt"
)

// writeStructuredContext writes one JSON-encoded structured log record to stdout.
func writeStructuredContext(parseLogContext context.Context, parseLogLevel, parseLogScope, parseLogMessage string, parseLogFields map[string]interface{}) {
	parseRecord := buildLogRecord(parseLogContext, parseLogLevel, parseLogScope, parseLogMessage, parseLogFields)
	parseEncodedRecord, parseErr := json.Marshal(parseRecord)
	if parseErr != nil {
		fmt.Printf("{\"level\":\"error\",\"message\":\"logging marshal failed\",\"error\":%q}\n", parseErr.Error())
		return
	}
	fmt.Println(string(parseEncodedRecord))
}
