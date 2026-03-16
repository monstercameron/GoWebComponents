//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func useChannelExample() ui.Node {
	source := ui.UseState((<-chan string)(nil))
	started := ui.UseState(0)
	channel := ui.UseChannel(source.Get())

	start := ui.UseEvent(func() {
		messages := make(chan string)
		source.Set(messages)
		started.Update(func(previous int) int { return previous + 1 })

		go func(out chan<- string) {
			defer close(out)
			steps := []string{"Boot worker", "Read queue", "Transform payload", "Publish result"}
			for _, step := range steps {
				time.Sleep(300 * time.Millisecond)
				out <- step
			}
		}(messages)
	})
	reset := ui.UseEvent(func() { source.Set(nil) })

	latest := "No value yet"
	if channel.Ok() {
		latest = channel.Get()
	}

	return shared.ExamplePage(
		"ui.UseChannel",
		"Subscribe to the latest value from a Go channel",
		"UseChannel exposes the last received value together with availability and closure state, which keeps streamed goroutine output readable from regular component render logic.",
		shared.ExamplePanel("Channel stream",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Start new stream", start),
				shared.ExampleButton("Reset source", reset),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Started", fmt.Sprintf("%d", started.Get())),
				shared.ExampleStat("Latest", latest),
				shared.ExampleStat("Has value", map[bool]string{true: "Yes", false: "No"}[channel.Ok()]),
				shared.ExampleStat("Closed", map[bool]string{true: "Yes", false: "No"}[channel.Closed()]),
			),
			shared.ExampleCode(
				`stream := ui.UseChannel(source)`,
				`if stream.Ok() { latest := stream.Get() }`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useChannelExample), "#app")
	select {}
}
