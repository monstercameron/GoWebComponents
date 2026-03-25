//go:build js && wasm
// +build js,wasm

package routertest

import base "github.com/monstercameron/GoWebComponents/testkit/router"
import appRouter "github.com/monstercameron/GoWebComponents/router"

type LoaderAttempt = base.LoaderAttempt
type LoaderController = base.LoaderController

func NewLoaderController() *LoaderController {
	return base.NewLoaderController()
}

func BuildGuardBlocked(reason string) appRouter.GuardFunc {
	return base.BuildGuardBlocked(reason)
}

func BuildGuardRedirect(path string) appRouter.GuardFunc {
	return base.BuildGuardRedirect(path)
}

func BuildAsyncGuardDenied(reason string) appRouter.AsyncGuardFunc {
	return base.BuildAsyncGuardDenied(reason)
}
