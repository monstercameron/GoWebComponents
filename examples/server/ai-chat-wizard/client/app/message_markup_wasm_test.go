//go:build js && wasm

package app

import (
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/testkit/render"
)

// setMessageMarkupTestRuntime provides the DOM adapter required by Wasm event hooks.
func setMessageMarkupTestRuntime(parseT *testing.T) {
	render.New(parseT)
}

// TestMessageCodeCopyUnavailable reports clipboard failure without removing the code or duplicating controls.
func TestMessageCodeCopyUnavailable(parseT *testing.T) {
	parseFixture := render.New(parseT)
	parseFixture.Render(renderMessageMarkup(`<pre><code>copy this &lt; literally</code></pre>`))
	parseButton := parseFixture.ByRole("button", "Copy")
	if !parseButton.Exists() {
		parseT.Fatal("GWC copy control missing")
	}
	parseButton.Click()
	parseDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadline) && !parseFixture.ByRole("button", "Copy failed — retry").Exists() {
		time.Sleep(time.Millisecond)
		parseFixture.Flush()
	}
	if !parseFixture.ByRole("button", "Copy failed — retry").Exists() {
		parseT.Fatalf("clipboard unavailability was hidden: %s", parseFixture.Text())
	}
	parseFixture.Rerender(renderMessageMarkup(`<pre><code>replacement</code></pre>`))
	if len(parseFixture.AllByRole("button")) != 1 || !parseFixture.ByRole("button", "Copy").Exists() || strings.Contains(parseFixture.Text(), "copy this") || !strings.Contains(parseFixture.Text(), "replacement") {
		parseT.Fatalf("code replacement corrupted ownership: %s", parseFixture.Text())
	}
	parseFixture.Rerender(renderMessageMarkup(`<pre><code>same</code></pre><pre><code>same</code></pre>`))
	if len(parseFixture.AllByRole("button")) != 2 {
		parseT.Fatalf("identical sibling samples lost a copy control: %s", parseFixture.Text())
	}
}
