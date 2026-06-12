//go:build js && wasm

package routertest

import base "github.com/monstercameron/GoWebComponents/testkit/router"
import appRouter "github.com/monstercameron/GoWebComponents/router"

type LoaderAttempt = base.LoaderAttempt
type LoaderController = base.LoaderController

func NewLoaderController() *LoaderController {
	return base.NewLoaderController()
}

func BuildGuardBlocked(parseReason string) appRouter.GuardFunc {
	return base.BuildGuardBlocked(parseReason)
}

func BuildGuardRedirect(parsePath string) appRouter.GuardFunc {
	return base.BuildGuardRedirect(parsePath)
}

func BuildAsyncGuardDenied(parseReason string) appRouter.AsyncGuardFunc {
	return base.BuildAsyncGuardDenied(parseReason)
}
