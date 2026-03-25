//go:build !js || !wasm
// +build !js !wasm

package logging

import (
	"fmt"
	"strings"
)

func writeStructured(parseLogLevel, parseLogScope, parseLogMessage string, parseLogFields map[string]interface{}) {
	parseLogPrefix := ""
	if parseLogScope != "" {
		parseLogPrefix = "[" + parseLogScope + "] "
	}
	if len(parseLogFields) == 0 {
		fmt.Printf("%s%s: %s\n", parseLogPrefix, strings.ToUpper(parseLogLevel), parseLogMessage)
		return
	}
	fmt.Printf("%s%s: %s %v\n", parseLogPrefix, strings.ToUpper(parseLogLevel), parseLogMessage, parseLogFields)
}
