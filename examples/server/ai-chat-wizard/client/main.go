//go:build js && wasm

package main

import wizardapp "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/client/app"

func main() {
	wizardapp.ParseRun()
}
