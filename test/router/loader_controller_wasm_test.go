//go:build js && wasm
// +build js,wasm

package routertest_test

import (
	"errors"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	appRouter "github.com/monstercameron/GoWebComponents/router"
	routertest "github.com/monstercameron/GoWebComponents/test/router"
)

func controlledLoaderRoute(_ appRouter.Attrs) *appRouter.Element {
	data := appRouter.UseRouteData()
	if data["message"] == nil {
		return html.Div(html.Props{ID: "loader-route"}, html.Text("missing"))
	}
	return html.Div(html.Props{ID: "loader-route"}, html.Text(data["message"].(string)))
}

func TestLoaderControllerDrivesResolveRejectCancelAndRetry(t *testing.T) {
	controller := routertest.NewLoaderController()
	fixture := routertest.NewHistory(t)
	fixture.Register("/users/:id", controlledLoaderRoute, appRouter.Options{
		Loader:  controller.Loader(),
		Loading: html.Div(html.Props{ID: "loader-loading"}, html.Text("loading")),
		Error: func(props appRouter.Attrs) *appRouter.Element {
			return html.Div(html.Props{ID: "loader-error"}, html.Text(props["error"].(error).Error()))
		},
	})

	fixture.SetPath("/users/7?q=focus")
	if got := <-controller.Started(); got != 1 {
		t.Fatalf("expected first loader attempt index 1, got %d", got)
	}
	if !controller.Pending() || fixture.ByID("loader-loading") == nil {
		t.Fatalf("expected loader to remain pending, got pending=%t", controller.Pending())
	}
	first := controller.Attempts()[0]
	if first.Path != "/users/7" || first.Params["id"] != "7" || first.Query.Get("q") != "focus" {
		t.Fatalf("expected route context to be captured, got %+v", first)
	}

	controller.Resolve(appRouter.Attrs{"message": "ready"})
	fixture.Render()
	if got := fixture.ByID("loader-route").Text(); got != "ready" {
		t.Fatalf("expected resolved route data, got %q", got)
	}

	fixture.Router().Revalidate()
	fixture.Render()
	if got := <-controller.Started(); got != 2 {
		t.Fatalf("expected second loader attempt index 2, got %d", got)
	}
	controller.Reject(errors.New("boom"))
	fixture.Render()
	if got := fixture.ByID("loader-error").Text(); got != "boom" {
		t.Fatalf("expected loader error route, got %q", got)
	}

	fixture.Router().Revalidate()
	fixture.Render()
	if got := <-controller.Started(); got != 3 {
		t.Fatalf("expected third loader attempt index 3, got %d", got)
	}
	controller.Cancel()
	fixture.Render()
	if attempts := controller.Attempts(); len(attempts) != 3 || !controller.Pending() {
		t.Fatalf("expected cancelled attempt to remain recorded while awaiting retry, got attempts=%+v pending=%t", attempts, controller.Pending())
	}

	fixture.Router().Revalidate()
	fixture.Render()
	if got := <-controller.Started(); got != 4 {
		t.Fatalf("expected retry loader attempt index 4, got %d", got)
	}
	controller.Resolve(appRouter.Attrs{"message": "retried"})
	fixture.Render()
	if got := fixture.ByID("loader-route").Text(); got != "retried" {
		t.Fatalf("expected retried loader route data, got %q", got)
	}
}
