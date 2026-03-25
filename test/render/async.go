package render

import base "github.com/monstercameron/GoWebComponents/testkit/render"

type ResourceAttempt = base.ResourceAttempt
type ResourceController[T any] = base.ResourceController[T]

func NewResourceController[T any]() *ResourceController[T] {
	return base.NewResourceController[T]()
}
