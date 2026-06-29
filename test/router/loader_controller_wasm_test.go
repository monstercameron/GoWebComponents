//go:build js && wasm

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
	parseData := appRouter.UseRouteData()
	if parseData["message"] == nil {
		return html.Div(html.Props{ID: "loader-route"}, html.Text("missing"))
	}
	return html.Div(html.Props{ID: "loader-route"}, html.Text(parseData["message"].(string)))
}

func TestLoaderControllerDrivesResolveRejectCancelAndRetry(parseT *testing.T) {
	parseController := routertest.NewLoaderController()
	parseFixture := routertest.NewHistory(parseT)
	parseFixture.Register("/users/:id", controlledLoaderRoute, appRouter.Options{
		Loader:  parseController.Loader(),
		Loading: html.Div(html.Props{ID: "loader-loading"}, html.Text("loading")),
		Error: func(parseProps appRouter.Attrs) *appRouter.Element {
			parseErrText := ""
			switch parseTyped := parseProps["error"].(type) {
			case error:
				parseErrText = parseTyped.Error()
			case string:
				parseErrText = parseTyped
			default:
				parseErrText = fmt.Sprint(parseTyped)
			}
			return html.Div(html.Props{ID: "loader-error"}, html.Text(parseErrText))
		},
	})

	parseFixture.SetPath("/users/7?q=focus")
	if parseGot := <-parseController.Started(); parseGot != 1 {
		parseT.Fatalf("expected first loader attempt index 1, got %d", parseGot)
	}
	if !parseController.Pending() || parseFixture.ByID("loader-loading") == nil {
		parseT.Fatalf("expected loader to remain pending, got pending=%t", parseController.Pending())
	}
	parseFirst := parseController.Attempts()[0]
	if parseFirst.Path != "/users/7" || parseFirst.Params["id"] != "7" || parseFirst.Query.Get("q") != "focus" {
		parseT.Fatalf("expected route context to be captured, got %+v", parseFirst)
	}

	parseController.Resolve(appRouter.Attrs{"message": "ready"})
	if parseGot2 := waitForNodeText(parseT, "ready", parseFixture.Render, func() string {
		parseNode := parseFixture.ByID("loader-route")
		if parseNode == nil {
			return ""
		}
		return parseNode.Text()
	}); parseGot2 != "ready" {
		parseT.Fatalf("expected resolved route data, got %q", parseGot2)
	}

	parseFixture.Navigate("/users/8?q=focus")
	if parseGot3 := <-parseController.Started(); parseGot3 != 2 {
		parseT.Fatalf("expected second loader attempt index 2, got %d", parseGot3)
	}
	parseController.Reject(errors.New("boom"))
	if parseGot4 := waitForNodeText(parseT, "boom", parseFixture.Render, func() string {
		parseNode2 := parseFixture.ByID("loader-error")
		if parseNode2 == nil {
			return ""
		}
		return parseNode2.Text()
	}); parseGot4 != "boom" {
		parseT.Fatalf("expected loader error route, got %q", parseGot4)
	}

	parseFixture.Navigate("/users/9?q=focus")
	if parseGot5 := <-parseController.Started(); parseGot5 != 3 {
		parseT.Fatalf("expected third loader attempt index 3, got %d", parseGot5)
	}
	parseController.Cancel()
	parseFixture.Render()
	if parseAttempts := parseController.Attempts(); len(parseAttempts) != 3 || !parseAttempts[2].Cancelled || parseController.Pending() {
		parseT.Fatalf("expected cancelled attempt to be recorded and pending state cleared, got attempts=%+v pending=%t", parseAttempts, parseController.Pending())
	}

	parseFixture.Navigate("/users/10?q=focus")
	if parseGot6 := <-parseController.Started(); parseGot6 != 4 {
		parseT.Fatalf("expected retry loader attempt index 4, got %d", parseGot6)
	}
	parseController.Resolve(appRouter.Attrs{"message": "retried"})
	if parseGot7 := waitForNodeText(parseT, "retried", parseFixture.Render, func() string {
		parseNode3 := parseFixture.ByID("loader-route")
		if parseNode3 == nil {
			return ""
		}
		return parseNode3.Text()
	}); parseGot7 != "retried" {
		parseT.Fatalf("expected retried loader route data, got %q", parseGot7)
	}
}

func waitForNodeText(parseT *testing.T, parseExpected string, parseRerender func(), parseRead func() string) string {
	parseT.Helper()
	parseDeadline := time.Now().Add(2 * time.Second)
	parseLast := ""
	for time.Now().Before(parseDeadline) {
		parseRerender()
		parseLast = parseRead()
		if parseLast == parseExpected {
			return parseLast
		}
		time.Sleep(10 * time.Millisecond)
	}
	return parseLast
}
