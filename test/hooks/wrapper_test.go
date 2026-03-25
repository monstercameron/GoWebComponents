package hooks

import "testing"

func TestRenderHookPanicsWithNilTestingTB(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic when RenderHook receives nil testing.TB")
		}
	}()

	_ = RenderHook[int](nil, func() int { return 42 })
}

