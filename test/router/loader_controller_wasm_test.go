//go:build js && wasm
// +build js,wasm

package routertest_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

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
			errText := ""
			switch typed := props["error"].(type) {
			case error:
				errText = typed.Error()
			case string:
				errText = typed
			default:
				errText = fmt.Sprint(typed)
			}
			return html.Div(html.Props{ID: "loader-error"}, html.Text(errText))
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
	if got := waitForNodeText(t, "ready", fixture.Render, func() string {
		node := fixture.ByID("loader-route")
		if node == nil {
			return ""
		}
		return node.Text()
	}); got != "ready" {
		t.Fatalf("expected resolved route data, got %q", got)
	}

	fixture.Navigate("/users/8?q=focus")
	if got := <-controller.Started(); got != 2 {
		t.Fatalf("expected second loader attempt index 2, got %d", got)
	}
	controller.Reject(errors.New("boom"))
	if got := waitForNodeText(t, "boom", fixture.Render, func() string {
		node := fixture.ByID("loader-error")
		if node == nil {
			return ""
		}
		return node.Text()
	}); got != "boom" {
		t.Fatalf("expected loader error route, got %q", got)
	}

	fixture.Navigate("/users/9?q=focus")
	if got := <-controller.Started(); got != 3 {
		t.Fatalf("expected third loader attempt index 3, got %d", got)
	}
	controller.Cancel()
	fixture.Render()
	if attempts := controller.Attempts(); len(attempts) != 3 || !attempts[2].Cancelled || controller.Pending() {
		t.Fatalf("expected cancelled attempt to be recorded and pending state cleared, got attempts=%+v pending=%t", attempts, controller.Pending())
	}

	fixture.Navigate("/users/10?q=focus")
	if got := <-controller.Started(); got != 4 {
		t.Fatalf("expected retry loader attempt index 4, got %d", got)
	}
	controller.Resolve(appRouter.Attrs{"message": "retried"})
	if got := waitForNodeText(t, "retried", fixture.Render, func() string {
		node := fixture.ByID("loader-route")
		if node == nil {
			return ""
		}
		return node.Text()
	}); got != "retried" {
		t.Fatalf("expected retried loader route data, got %q", got)
	}
}

func waitForNodeText(t *testing.T, expected string, rerender func(), read func() string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	last := ""
	for time.Now().Before(deadline) {
		rerender()
		last = read()
		if last == expected {
			return last
		}
		time.Sleep(10 * time.Millisecond)
	}
	return last
}
