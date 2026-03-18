//go:build !js || !wasm
// +build !js !wasm

package atlas

type atlasLocalState[T any] struct {
	value T
}

func useAtlasState[T any](initial T) *atlasLocalState[T] {
	return &atlasLocalState[T]{value: initial}
}

func (s *atlasLocalState[T]) Get() T {
	return s.value
}

func (s *atlasLocalState[T]) Set(value T) {
	s.value = value
}

func useAtlasEffect(effect func() func(), deps ...interface{}) {
}
