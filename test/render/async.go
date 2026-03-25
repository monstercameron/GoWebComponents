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

func BuildFailureError(code string, message string) error {
	return base.BuildFailureError(code, message)
}

func BuildHydrationMismatchError(path string, reason string) error {
	return base.BuildHydrationMismatchError(path, reason)
}

func BuildLoaderFailureError(path string, reason string) error {
	return base.BuildLoaderFailureError(path, reason)
}

func BuildRouteGuardFailureError(path string, reason string) error {
	return base.BuildRouteGuardFailureError(path, reason)
}

func BuildCacheConflictError(entity string) error {
	return base.BuildCacheConflictError(entity)
}

func BuildOfflineReplayError(entity string, reason string) error {
	return base.BuildOfflineReplayError(entity, reason)
}
