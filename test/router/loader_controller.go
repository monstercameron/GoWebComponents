//go:build js && wasm
// +build js,wasm

package routertest

import base "github.com/monstercameron/GoWebComponents/testkit/router"

type LoaderAttempt = base.LoaderAttempt
type LoaderController = base.LoaderController

func NewLoaderController() *LoaderController {
	return base.NewLoaderController()
}
