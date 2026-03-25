//go:build js && wasm
// +build js,wasm

package logging

import "github.com/monstercameron/GoWebComponents/utils"

func writeStructured(parseLevel, parseScope, parseMessage string, parseFields map[string]interface{}) {
	utils.WriteConsoleStructured(parseLevel, parseScope, parseMessage, parseFields)
}
