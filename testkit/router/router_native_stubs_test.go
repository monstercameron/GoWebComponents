//go:build !js || !wasm
// +build !js !wasm

package routertest

import "testing"

func TestNativeRouterFixtureStubMethods(t *testing.T) {
	fixture := &Fixture{}

	fixture.Register("/users/:id", nil)
	fixture.SetPath("/users/1")
	fixture.Render()
	fixture.Navigate("/users/2")
	fixture.Replace("/users/2?tab=history")
	fixture.Cleanup()

	inspection := fixture.Inspect()
	if inspection.Path != "" || inspection.Query != nil || inspection.Params != nil {
		t.Fatalf("expected zero-value inspection in native stub, got %+v", inspection)
	}
	if fixture.Path() != "" || fixture.Query() != nil || fixture.Params() != nil || fixture.Router() != nil {
		t.Fatalf("expected zero-value navigation state in native stub")
	}
	if fixture.ByID("route") != nil || fixture.ByText("route") != nil {
		t.Fatalf("expected no route nodes in native stub")
	}
	if fixture.Text() != "" {
		t.Fatalf("expected empty fixture text in native stub")
	}
}

