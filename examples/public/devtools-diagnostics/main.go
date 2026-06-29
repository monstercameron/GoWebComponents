//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/devtools"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

func diagnosticsExample() ui.Node {
	parseSnapshot := devtools.UseSnapshot(800 * time.Millisecond)
	parseMessage := "No diagnostics yet"
	if len(parseSnapshot.Diagnostics) > 0 {
		parseMessage = parseSnapshot.Diagnostics[0].Message
	}

	return shared.ExamplePage(
		"devtools diagnostics",
		"Read structured runtime and router diagnostics from snapshots",
		"This example intentionally triggers a router warning by registering the same exact route twice. The devtools snapshot then exposes the diagnostic payload inside normal UI.",
		shared.ExamplePanel("Diagnostics summary",
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Count", fmt.Sprintf("%d", len(parseSnapshot.Diagnostics))),
				shared.ExampleStat("Source", firstDiagnosticSource(parseSnapshot)),
				shared.ExampleStat("Severity", firstDiagnosticSeverity(parseSnapshot)),
			),
			html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text(parseMessage)),
		),
	)
}

func firstDiagnosticSource(parseSnapshot devtools.Snapshot) string {
	if len(parseSnapshot.Diagnostics) == 0 {
		return "-"
	}
	return parseSnapshot.Diagnostics[0].Source
}

func firstDiagnosticSeverity(parseSnapshot devtools.Snapshot) string {
	if len(parseSnapshot.Diagnostics) == 0 {
		return "-"
	}
	return string(parseSnapshot.Diagnostics[0].Severity)
}

func main() {
	utils.DisableAllDebug()
	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	parseR.Register("/", func(router.Attrs) *router.Element { return ui.CreateElement(diagnosticsExample) })
	parseR.Register("/", func(router.Attrs) *router.Element { return ui.CreateElement(diagnosticsExample) })
	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
