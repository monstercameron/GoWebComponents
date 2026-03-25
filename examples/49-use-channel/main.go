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
	parseSource := ui.UseState((<-chan string)(nil))
	parseStarted := ui.UseState(0)
	parseChannel := ui.UseChannel(parseSource.Get())

	parseStart := ui.UseEvent(func() {
		parseMessages := make(chan string)
		parseSource.Set(parseMessages)
		parseStarted.Update(func(parsePrevious int) int { return parsePrevious + 1 })

		go func(parseOut chan<- string) {
			defer close(parseOut)
			parseSteps := []string{"Boot worker", "Read queue", "Transform payload", "Publish result"}
			for _, parseStep := range parseSteps {
				time.Sleep(300 * time.Millisecond)
				parseOut <- parseStep
			}
		}(parseMessages)
	})
	reset := ui.UseEvent(func() { parseSource.Set(nil) })

	parseLatest := "No value yet"
	if parseChannel.Ok() {
		parseLatest = parseChannel.Get()
	}

	return shared.ExamplePage(
		"ui.UseChannel",
		"Subscribe to the latest value from a Go channel",
		"UseChannel exposes the last received value together with availability and closure state, which keeps streamed goroutine output readable from regular component render logic.",
		shared.ExamplePanel("Channel stream",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Start new stream", parseStart),
				shared.ExampleButton("Reset source", reset),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Started", fmt.Sprintf("%d", parseStarted.Get())),
				shared.ExampleStat("Latest", parseLatest),
				shared.ExampleStat("Has value", map[bool]string{true: "Yes", false: "No"}[parseChannel.Ok()]),
				shared.ExampleStat("Closed", map[bool]string{true: "Yes", false: "No"}[parseChannel.Closed()]),
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
