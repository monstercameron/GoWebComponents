//go:build js && wasm
// +build js,wasm

package render_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/html"
	render "github.com/monstercameron/GoWebComponents/test/render"
	"github.com/monstercameron/GoWebComponents/ui"
)

func asyncResourceApp(controller *render.ResourceController[string]) ui.Node {
	resource := fetch.UseResource(controller.Loader(), "resource")
	state := resource.Get()
	label := "idle"
	switch {
	case state.Loading:
		label = "loading"
	case state.Error != nil:
		label = "error:" + state.Error.Error()
	default:
		label = fmt.Sprintf("value:%s", state.Value)
	}

	return html.Div(html.Props{ID: "resource-root"},
		html.P(html.Props{ID: "resource-state"}, html.Text(label)),
		html.Button(html.Props{ID: "resource-reload", Type: "button", OnClick: ui.UseEvent(func() {
			resource.Reload()
		})}, html.Text("Reload")),
		html.Button(html.Props{ID: "resource-cancel", Type: "button", OnClick: ui.UseEvent(func() {
			resource.Cancel()
		})}, html.Text("Cancel")),
	)
}

func TestResourceControllerDrivesResolveRejectCancelAndRetry(t *testing.T) {
	controller := render.NewResourceController[string]()
	fixture := render.New(t, render.WithQueuedScheduler())
	fixture.Render(ui.CreateElement(func() ui.Node {
		return asyncResourceApp(controller)
	}))

	if got := <-controller.Started(); got != 1 {
		t.Fatalf("expected first attempt index 1, got %d", got)
	}
	if !controller.Pending() || fixture.ByID("resource-state").Text() != "loading" {
		t.Fatalf("expected initial pending resource state, got pending=%t text=%q", controller.Pending(), fixture.ByID("resource-state").Text())
	}

	controller.Resolve("ready")
	fixture.Stabilize()
	if got := fixture.ByID("resource-state").Text(); got != "value:ready" {
		t.Fatalf("expected resolved resource value, got %q", got)
	}

	fixture.ClickByID("resource-reload")
	if got := <-controller.Started(); got != 2 {
		t.Fatalf("expected second attempt index 2, got %d", got)
	}
	controller.Reject(errors.New("boom"))
	fixture.Stabilize()
	if got := fixture.ByID("resource-state").Text(); got != "error:boom" {
		t.Fatalf("expected rejected resource state, got %q", got)
	}

	fixture.ClickByID("resource-reload")
	if got := <-controller.Started(); got != 3 {
		t.Fatalf("expected third attempt index 3, got %d", got)
	}
	fixture.ClickByID("resource-cancel")
	fixture.Stabilize()
	if attempts := controller.Attempts(); len(attempts) != 3 || !attempts[2].Cancelled {
		t.Fatalf("expected cancelled third attempt, got %+v", attempts)
	}

	fixture.ClickByID("resource-reload")
	if got := <-controller.Started(); got != 4 {
		t.Fatalf("expected retry attempt index 4, got %d", got)
	}
	controller.Resolve("retried")
	fixture.Stabilize()
	if got := fixture.ByID("resource-state").Text(); got != "value:retried" {
		t.Fatalf("expected retried resource value, got %q", got)
	}
}

func TestResourceControllerAwaitReturnsContextCancellation(t *testing.T) {
	controller := render.NewResourceController[string]()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := controller.Await(ctx)
		done <- err
	}()
	<-controller.Started()
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
