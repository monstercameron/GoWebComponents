//go:build !js || !wasm
// +build !js !wasm

package logging

import (
	"fmt"
	"strings"
)

func writeStructured(parseLevel, parseScope, parseMessage string, parseFields map[string]interface{}) {
	parsePrefix := ""
	if parseScope != "" {
		parsePrefix = "[" + parseScope + "] "
	}
	if len(parseFields) == 0 {
		fmt.Printf("%s%s: %s\n", parsePrefix, strings.ToUpper(parseLevel), parseMessage)
		return
	}
	fmt.Printf("%s%s: %s %v\n", parsePrefix, strings.ToUpper(parseLevel), parseMessage, parseFields)
}
