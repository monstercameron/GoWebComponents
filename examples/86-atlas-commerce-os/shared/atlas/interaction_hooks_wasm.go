//go:build js && wasm
// +build js,wasm

package atlas

import "github.com/monstercameron/GoWebComponents/ui"

type atlasLocalState[T any] struct {
	value ui.State[T]
}

func useAtlasState[T any](initial T) *atlasLocalState[T] {
	return &atlasLocalState[T]{value: ui.UseState(initial)}
}

func (s *atlasLocalState[T]) Get() T {
	return s.value.Get()
}

func (s *atlasLocalState[T]) Set(value T) {
	s.value.Set(value)
}

func useAtlasEffect(effect func() func(), deps ...interface{}) {
	ui.UseEffect(effect, deps...)
}
