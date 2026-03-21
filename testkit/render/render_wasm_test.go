//go:build js && wasm
// +build js,wasm

package render

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func staticHarnessApp() ui.Node {
	return html.Div(html.Props{ID: "fixture-root"},
		html.H1(html.Props{ID: "main-heading"}, html.Text("Hello Harness")),
		html.Button(html.Props{ID: "primary-button", Type: "button"}, html.Text("Save")),
		html.Span(html.Props{ID: "search-label"}, html.Text("Search Catalog")),
		html.Input(html.Props{ID: "search-input", Type: "text", Raw: map[string]interface{}{"aria-labelledby": "search-label"}}),
	)
}

func counterHarnessApp() ui.Node {
	count := ui.UseState(0)
	increment := ui.UseEvent(func() {
		count.Update(func(previous int) int {
			return previous + 1
		})
	})

	return html.Div(html.Props{ID: "counter-root"},
		html.P(html.Props{ID: "count-label"}, html.Text(fmt.Sprintf("Count: %d", count.Get()))),
		html.Button(html.Props{ID: "increment", Type: "button", OnClick: increment}, html.Text("Increment")),
	)
}

func inputHarnessApp() ui.Node {
	value := ui.UseState("")
	handleInput := ui.UseEvent(func(event ui.InputEvent) {
		value.Set(event.GetValue())
	})

	return html.Div(html.Props{ID: "input-root"},
		html.Input(html.Props{ID: "name-input", Type: "text", OnInput: handleInput}),
		html.P(html.Props{ID: "name-value"}, html.Text("Value: "+value.Get())),
	)
}

func transitionHarnessApp() ui.Node {
	count := ui.UseState(0)
	handleClick := ui.UseEvent(func() {
		ui.StartTransition(func() {
			count.Update(func(previous int) int {
				return previous + 1
			})
		})
	})

	return html.Div(html.Props{ID: "transition-root"},
		html.P(html.Props{ID: "transition-count"}, html.Text(fmt.Sprintf("Transition Count: %d", count.Get()))),
		html.Button(html.Props{ID: "transition-increment", Type: "button", OnClick: handleClick}, html.Text("Increment Later")),
	)
}

func TestFixtureRenderAndQuery(t *testing.T) {
	fixture := New(t)
	fixture.Render(ui.CreateElement(staticHarnessApp))

	heading := fixture.ByID("main-heading")
	if heading == nil || heading.Text() != "Hello Harness" {
		t.Fatalf("expected heading text, got %#v", heading)
	}
	if match := fixture.ByText("Save"); match == nil || match.Tag() != "button" {
		t.Fatalf("expected to find button by text, got %#v", match)
	}
	buttons := fixture.AllByTag("button")
	if len(buttons) != 1 {
		t.Fatalf("expected one button, got %d", len(buttons))
	}
	if buttons[0].Attr("id") != "primary-button" {
		t.Fatalf("expected queried button id to be preserved, got %q", buttons[0].Attr("id"))
	}
	if button := fixture.ByRole("button", "Save"); button == nil || button.Attr("id") != "primary-button" {
		t.Fatalf("expected role query to find the button, got %#v", button)
	}
	if field := fixture.ByRole("textbox", "Search Catalog"); field == nil || field.Attr("id") != "search-input" {
		t.Fatalf("expected role query to resolve aria-labelledby accessible name, got %#v", field)
	}
	if name := fixture.ByID("search-input").Name(); name != "Search Catalog" {
		t.Fatalf("expected stable accessible name, got %q", name)
	}
	if textboxes := fixture.AllByRole("textbox"); len(textboxes) != 1 {
		t.Fatalf("expected one textbox role match, got %d", len(textboxes))
	}
}

func TestFixtureExposesRenderedPropertiesForStateUpdates(t *testing.T) {
	fixture := New(t)
	fixture.Render(ui.CreateElement(counterHarnessApp))

	if got := fixture.ByID("count-label").Text(); got != "Count: 0" {
		t.Fatalf("expected initial count label, got %q", got)
	}

	handler, ok := fixture.ByID("increment").Property("onclick").(func())
	if !ok {
		t.Fatalf("expected onclick property with func() signature, got %T", fixture.ByID("increment").Property("onclick"))
	}
	handler()

	if got := fixture.ByID("count-label").Text(); got != "Count: 1" {
		t.Fatalf("expected updated count label after invoking onclick, got %q", got)
	}
}

func TestFixtureRerenderReplacesTree(t *testing.T) {
	fixture := New(t)
	fixture.Render(ui.CreateElement(staticHarnessApp))
	fixture.Rerender(html.Div(html.Props{ID: "replacement"}, html.Text("Replaced")))

	if fixture.ByID("main-heading") != nil {
		t.Fatal("expected original tree to be replaced")
	}
	replacement := fixture.ByID("replacement")
	if replacement == nil || replacement.Text() != "Replaced" {
		t.Fatalf("expected rerendered replacement tree, got %#v", replacement)
	}
}

func TestFixtureInputHelperDispatchesAndSettles(t *testing.T) {
	fixture := New(t)
	fixture.Render(ui.CreateElement(inputHarnessApp))

	fixture.InputByID("name-input", "Cam")

	if got := fixture.ByID("name-value").Text(); got != "Value: Cam" {
		t.Fatalf("expected input helper to update rendered value, got %q", got)
	}
}

func TestFixtureQueuedSchedulerClickHelperStabilizesTransitionWork(t *testing.T) {
	fixture := New(t, WithQueuedScheduler())
	fixture.Render(ui.CreateElement(transitionHarnessApp))

	fixture.ClickByID("transition-increment")

	if got := fixture.ByID("transition-count").Text(); got != "Transition Count: 1" {
		t.Fatalf("expected queued click helper to stabilize transition work, got %q", got)
	}
}
