//go:build js && wasm

package main

import wizardapp "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/client/app"

func main() {
	wizardapp.ParseRun()
}
