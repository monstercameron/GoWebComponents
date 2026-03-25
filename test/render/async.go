package render

import base "github.com/monstercameron/GoWebComponents/testkit/render"

type ResourceAttempt = base.ResourceAttempt
type ResourceController[T any] = base.ResourceController[T]
type FailureError = base.FailureError

const (
	FailureCodeHydrationMismatch = base.FailureCodeHydrationMismatch
	FailureCodeLoaderFailure     = base.FailureCodeLoaderFailure
	FailureCodeRouteGuardFailure = base.FailureCodeRouteGuardFailure
	FailureCodeCacheConflict     = base.FailureCodeCacheConflict
	FailureCodeOfflineReplay     = base.FailureCodeOfflineReplay
)

func NewResourceController[T any]() *ResourceController[T] {
	return base.NewResourceController[T]()
}

func BuildFailureError(parseCode string, parseMessage string) error {
	return base.BuildFailureError(parseCode, parseMessage)
}

func BuildHydrationMismatchError(parsePath string, parseReason string) error {
	return base.BuildHydrationMismatchError(parsePath, parseReason)
}

func BuildLoaderFailureError(parsePath string, parseReason string) error {
	return base.BuildLoaderFailureError(parsePath, parseReason)
}

func BuildRouteGuardFailureError(parsePath string, parseReason string) error {
	return base.BuildRouteGuardFailureError(parsePath, parseReason)
}

func BuildCacheConflictError(parseEntity string) error {
	return base.BuildCacheConflictError(parseEntity)
}

func BuildOfflineReplayError(parseEntity string, parseReason string) error {
	return base.BuildOfflineReplayError(parseEntity, parseReason)
}
