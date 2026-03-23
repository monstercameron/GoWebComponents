//go:build js && wasm
// +build js,wasm

package logging

import "github.com/monstercameron/GoWebComponents/utils"

func writeStructured(level, scope, message string, fields map[string]interface{}) {
	utils.ConsoleStructured(level, scope, message, fields)
}