//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/devtools"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func diagnosticsExample() ui.Node {
	snapshot := devtools.UseSnapshot(800 * time.Millisecond)
	message := "No diagnostics yet"
	if len(snapshot.Diagnostics) > 0 {
		message = snapshot.Diagnostics[0].Message
	}

	return shared.ExamplePage(
		"devtools diagnostics",
		"Read structured runtime and router diagnostics from snapshots",
		"This example intentionally triggers a router warning by registering the same exact route twice. The devtools snapshot then exposes the diagnostic payload inside normal UI.",
		shared.ExamplePanel("Diagnostics summary",
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Count", fmt.Sprintf("%d", len(snapshot.Diagnostics))),
				shared.ExampleStat("Source", firstDiagnosticSource(snapshot)),
				shared.ExampleStat("Severity", firstDiagnosticSeverity(snapshot)),
			),
			html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text(message)),
		),
	)
}

func firstDiagnosticSource(snapshot devtools.Snapshot) string {
	if len(snapshot.Diagnostics) == 0 {
		return "-"
	}
	return snapshot.Diagnostics[0].Source
}

func firstDiagnosticSeverity(snapshot devtools.Snapshot) string {
	if len(snapshot.Diagnostics) == 0 {
		return "-"
	}
	return string(snapshot.Diagnostics[0].Severity)
}

func main() {
	utils.DisableAllDebug()
	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	r.Register("/", func(router.Attrs) *router.Element { return ui.CreateElement(diagnosticsExample) })
	r.Register("/", func(router.Attrs) *router.Element { return ui.CreateElement(diagnosticsExample) })
	r.Mount("#app")
	select {}
}
