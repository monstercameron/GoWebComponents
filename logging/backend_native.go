//go:build !js || !wasm
// +build !js !wasm

package logging

import (
	"fmt"
	"strings"
)

func writeStructured(level, scope, message string, fields map[string]interface{}) {
	prefix := ""
	if scope != "" {
		prefix = "[" + scope + "] "
	}
	if len(fields) == 0 {
		fmt.Printf("%s%s: %s\n", prefix, strings.ToUpper(level), message)
		return
	}
	fmt.Printf("%s%s: %s %v\n", prefix, strings.ToUpper(level), message, fields)
}