//go:build js && wasm
// +build js,wasm

package state

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

type noOpScheduler struct{}

func (noOpScheduler) RequestIdleCallback(callback func(runtime.Deadline)) {}

func (noOpScheduler) SetTimeout(callback func(), delay int) {}

func installStateHookContext(t *testing.T) {
	t.Helper()
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	t.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})
}

func TestUseAtomSharesGlobalState(t *testing.T) {
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})

	atomID := "state-test-shared"

	runtime.SetCurrentFiber(&runtime.Fiber{})
	first := UseAtom(atomID, "Guest")
	if first.Get() != "Guest" {
		t.Fatalf("expected initial atom value, got %q", first.Get())
	}
	first.Set("Alice")
	if first.Get() != "Alice" {
		t.Fatalf("expected updated atom value, got %q", first.Get())
	}

	runtime.SetCurrentFiber(&runtime.Fiber{})
	second := UseAtom(atomID, "Ignored")
	if second.Get() != "Alice" {
		t.Fatalf("expected second hook to reuse shared atom state, got %q", second.Get())
	}
	second.Update(func(prev string) string { return prev + " Smith" })
	if second.Get() != "Alice Smith" {
		t.Fatalf("expected updater to modify shared atom state, got %q", second.Get())
	}

	runtime.SetCurrentFiber(nil)
}

func TestUseAtomWrapper(t *testing.T) {
	installStateHookContext(t)

	atom := UseAtom("state-test-wrapper", 1)
	if atom.Get() != 1 {
		t.Fatalf("expected initial wrapper atom value, got %d", atom.Get())
	}
	atom.Set(3)
	atom.Update(func(prev int) int { return prev * 2 })
	if atom.Get() != 6 {
		t.Fatalf("expected wrapper methods to update atom value, got %d", atom.Get())
	}
}