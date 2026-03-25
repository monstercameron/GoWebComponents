//go:build js && wasm
// +build js,wasm

package logging

import "github.com/monstercameron/GoWebComponents/utils"

func writeStructured(parseLogLevel, parseLogScope, parseLogMessage string, parseLogFields map[string]interface{}) {
	utils.WriteConsoleStructured(parseLogLevel, parseLogScope, parseLogMessage, parseLogFields)
}
