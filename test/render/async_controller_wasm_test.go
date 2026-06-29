//go:build js && wasm

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

func asyncResourceApp(parseController *render.ResourceController[string]) ui.Node {
	parseResource := fetch.UseResource(parseController.Loader(), "resource")
	parseState := parseResource.Get()
	parseLabel := "idle"
	switch {
	case parseState.Loading:
		parseLabel = "loading"
	case parseState.Error != nil:
		parseLabel = "error:" + parseState.Error.Error()
	default:
		parseLabel = fmt.Sprintf("value:%s", parseState.Value)
	}

	return html.Div(html.Props{ID: "resource-root"},
		html.P(html.Props{ID: "resource-state"}, html.Text(parseLabel)),
		html.Button(html.Props{ID: "resource-reload", Type: "button", OnClick: ui.UseEvent(func() {
			parseResource.Reload()
		})}, html.Text("Reload")),
		html.Button(html.Props{ID: "resource-cancel", Type: "button", OnClick: ui.UseEvent(func() {
			parseResource.Cancel()
		})}, html.Text("Cancel")),
	)
}

func TestResourceControllerDrivesResolveRejectCancelAndRetry(parseT *testing.T) {
	parseController := render.NewResourceController[string]()
	parseFixture := render.New(parseT, render.WithQueuedScheduler())
	parseFixture.Render(ui.CreateElement(func() ui.Node {
		return asyncResourceApp(parseController)
	}))

	if parseGot := <-parseController.Started(); parseGot != 1 {
		parseT.Fatalf("expected first attempt index 1, got %d", parseGot)
	}
	if !parseController.Pending() || parseFixture.ByID("resource-state").Text() != "loading" {
		parseT.Fatalf("expected initial pending resource state, got pending=%t text=%q", parseController.Pending(), parseFixture.ByID("resource-state").Text())
	}

	parseController.Resolve("ready")
	parseFixture.Stabilize()
	if parseGot2 := parseFixture.ByID("resource-state").Text(); parseGot2 != "value:ready" {
		parseT.Fatalf("expected resolved resource value, got %q", parseGot2)
	}

	parseFixture.ClickByID("resource-reload")
	if parseGot3 := <-parseController.Started(); parseGot3 != 2 {
		parseT.Fatalf("expected second attempt index 2, got %d", parseGot3)
	}
	parseController.Reject(errors.New("boom"))
	parseFixture.Stabilize()
	if parseGot4 := parseFixture.ByID("resource-state").Text(); parseGot4 != "error:boom" {
		parseT.Fatalf("expected rejected resource state, got %q", parseGot4)
	}

	parseFixture.ClickByID("resource-reload")
	if parseGot5 := <-parseController.Started(); parseGot5 != 3 {
		parseT.Fatalf("expected third attempt index 3, got %d", parseGot5)
	}
	parseFixture.ClickByID("resource-cancel")
	parseFixture.Stabilize()
	if parseAttempts := parseController.Attempts(); len(parseAttempts) != 3 || !parseAttempts[2].Cancelled {
		parseT.Fatalf("expected cancelled third attempt, got %+v", parseAttempts)
	}

	parseFixture.ClickByID("resource-reload")
	if parseGot6 := <-parseController.Started(); parseGot6 != 4 {
		parseT.Fatalf("expected retry attempt index 4, got %d", parseGot6)
	}
	parseController.Resolve("retried")
	parseFixture.Stabilize()
	if parseGot7 := parseFixture.ByID("resource-state").Text(); parseGot7 != "value:retried" {
		parseT.Fatalf("expected retried resource value, got %q", parseGot7)
	}
}

func TestResourceControllerAwaitReturnsContextCancellation(parseT *testing.T) {
	parseController := render.NewResourceController[string]()
	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseDone := make(chan error, 1)
	go func() {
		_, parseErr := parseController.Await(parseCtx)
		parseDone <- parseErr
	}()
	<-parseController.Started()
	parseCancel()
	if parseErr2 := <-parseDone; !errors.Is(parseErr2, context.Canceled) {
		parseT.Fatalf("expected context cancellation, got %v", parseErr2)
	}
}
