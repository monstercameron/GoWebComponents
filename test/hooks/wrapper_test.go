package hooks

import "testing"

func TestRenderHookPanicsWithNilTestingTB(parseT *testing.T) {
	defer func() {
		if recover() == nil {
			parseT.Fatalf("expected panic when RenderHook receives nil testing.TB")
		}
	}()

	_ = RenderHook[int](nil, func() int { return 42 })
}
