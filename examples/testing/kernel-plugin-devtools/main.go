//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/devtools"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/internal/pluginruntime"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const (
	getKernelPluginDevtoolsRendererID = "examples.kernel-plugin-devtools.region"
	getKernelPluginDevtoolsRegionID   = "examples.kernel-plugin-devtools.region.primary"
)

type renderKernelPluginDevtoolsRegionProps struct {
	Count int
	Theme string
}

type buildKernelPluginDevtoolsPlugin struct{}

// Manifest returns the example kernel plugin manifest.
func (buildKernelPluginDevtoolsPlugin) Manifest() pluginruntime.Manifest {
	return pluginruntime.Manifest{
		ID:               "examples.kernel-plugin-devtools",
		Version:          "1.0.0",
		Description:      "example kernel plugin section",
		OptionalServices: []pluginruntime.ServiceKey{pluginruntime.ServiceKeyDOM, pluginruntime.ServiceKeyStyle, pluginruntime.ServiceKeyEvents, pluginruntime.ServiceKeyRuntime2Meta},
		ActivationPolicy: pluginruntime.ActivationPolicyBoot,
	}
}

// StartPlugin registers the example kernel-backed devtools section.
func (buildKernelPluginDevtoolsPlugin) StartPlugin(parseContext pluginruntime.Context) (pluginruntime.Handle, error) {
	parseErr := parseContext.RegisterContribution(pluginruntime.ContributionRegistration{
		Metadata: pluginruntime.ContributionMetadata{
			ID:               "example-kernel-plugin",
			Kind:             pluginruntime.ContributionKindDevtoolsSection,
			ExecutionClass:   pluginruntime.ExecutionClassWarm,
			ActivationPolicy: pluginruntime.ActivationPolicyBoot,
			Order:            220,
		},
		Value: pluginruntime.DevtoolsSectionProviderFunc(func() ([]pluginruntime.DevtoolsSection, error) {
			buildSummary := map[string]string{}
			buildLines := make([]string, 0, 12)

			if getDOMService, hasDOMService := parseContext.ResolveService(pluginruntime.ServiceKeyDOM); hasDOMService {
				if getDOM, hasTypedDOM := getDOMService.(pluginruntime.DOMService); hasTypedDOM {
					getDOMSnapshot, parseDOMErr := getDOM.GetDOMSnapshot(pluginruntime.QueryBudget{MaxItems: 400})
					if parseDOMErr == nil {
						getNodeCount := buildKernelPluginDevtoolsNodeCount(getDOMSnapshot.Root)
						buildSummary["dom"] = fmt.Sprintf("%d", getNodeCount)
						buildLines = append(buildLines, fmt.Sprintf("dom nodes: %d", getNodeCount))
						buildLines = append(buildLines, fmt.Sprintf("theme: %s", findKernelPluginDevtoolsTheme(getDOMSnapshot.Root)))
						if getDOMSnapshot.Meta.Truncated {
							buildLines = append(buildLines, "dom snapshot truncated: true")
						}
					}
				}
			}

			if getStyleService, hasStyleService := parseContext.ResolveService(pluginruntime.ServiceKeyStyle); hasStyleService {
				if getStyle, hasTypedStyle := getStyleService.(pluginruntime.StyleService); hasTypedStyle {
					getStyleSnapshot, parseStyleErr := getStyle.GetStyleSnapshot(pluginruntime.QueryBudget{})
					if parseStyleErr == nil {
						buildSummary["vars"] = fmt.Sprintf("%d", len(getStyleSnapshot.Variables))
						buildLines = append(buildLines, fmt.Sprintf("style variables: %d", len(getStyleSnapshot.Variables)))
						buildLines = append(buildLines, fmt.Sprintf("stylesheets: %d", len(getStyleSnapshot.Stylesheet)))
					}
				}
			}

			if getEventService, hasEventService := parseContext.ResolveService(pluginruntime.ServiceKeyEvents); hasEventService {
				if getEvents, hasTypedEvents := getEventService.(pluginruntime.EventService); hasTypedEvents {
					getEventSnapshot, parseEventErr := getEvents.GetEventSnapshot(pluginruntime.QueryBudget{MaxItems: 16})
					if parseEventErr == nil {
						buildSummary["events"] = fmt.Sprintf("%d", len(getEventSnapshot.Events))
						buildLines = append(buildLines, fmt.Sprintf("events: %d", len(getEventSnapshot.Events)))
						buildLines = append(buildLines, fmt.Sprintf("latest event: %s", formatKernelPluginDevtoolsLatestEvent(getEventSnapshot.Events)))
					}
				}
			}

			if getRuntime2MetaService, hasRuntime2MetaService := parseContext.ResolveService(pluginruntime.ServiceKeyRuntime2Meta); hasRuntime2MetaService {
				if getRuntime2Meta, hasTypedRuntime2Meta := getRuntime2MetaService.(pluginruntime.Runtime2MetaService); hasTypedRuntime2Meta {
					getRuntime2Snapshot, parseRuntime2Err := getRuntime2Meta.GetRuntime2MetaSnapshot(pluginruntime.QueryBudget{MaxItems: 4})
					if parseRuntime2Err == nil {
						buildSummary["runtime2"] = fmt.Sprintf("%d", len(getRuntime2Snapshot.Regions))
						buildLines = append(buildLines, fmt.Sprintf("runtime2 regions: %d", len(getRuntime2Snapshot.Regions)))
						for _, getRegion := range getRuntime2Snapshot.Regions {
							buildLines = append(buildLines, fmt.Sprintf(
								"%s => %s, transport=%s, diagnostics=%d",
								getRegion.RegionInstanceID,
								getRegion.RegionMode,
								getRegion.TransportTier,
								getRegion.DiagnosticCount,
							))
						}
					}
				}
			}

			if len(buildLines) == 0 {
				buildLines = append(buildLines, "example kernel plugin is waiting on kernel services")
			}
			return []pluginruntime.DevtoolsSection{{
				Name:    "Example Kernel Plugin",
				Summary: buildSummary,
				Lines:   buildLines,
			}}, nil
		}),
	})
	if parseErr != nil {
		return nil, parseErr
	}
	return nil, nil
}

// renderKernelPluginDevtoolsRegion renders the runtime2-backed region card.
func renderKernelPluginDevtoolsRegion(parseProps renderKernelPluginDevtoolsRegionProps) ui.Node {
	return html.Div(html.Props{Class: "rounded-[1.75rem] border border-cyan-300/20 bg-[linear-gradient(140deg,rgba(8,47,73,0.92),rgba(15,23,42,0.9))] p-6 shadow-[0_22px_50px_rgba(8,47,73,0.38)]"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.28em] text-cyan-200"}, html.Text("Runtime2 Region")),
		html.P(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text(fmt.Sprintf("%d", parseProps.Count))),
		html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-200"}, html.Text("Theme: "+parseProps.Theme)),
		html.P(html.Props{Class: "mt-3 text-xs uppercase tracking-[0.24em] text-cyan-100/80"}, html.Text("Region ID: "+getKernelPluginDevtoolsRegionID)),
	)
}

// renderKernelPluginDevtoolsButton renders one labeled example action button.
func renderKernelPluginDevtoolsButton(parseID string, parseLabel string, parseHandler ui.Handler) ui.Node {
	return html.Button(
		html.Props{
			ID:      parseID,
			OnClick: parseHandler,
			Class:   "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80",
		},
		html.Text(parseLabel),
	)
}

// renderKernelPluginDevtoolsExample renders the example page and embedded devtools surface.
func renderKernelPluginDevtoolsExample() ui.Node {
	parseCount := ui.UseState(1)
	parseTheme := ui.UseState("ocean")

	handleKernelPluginDevtoolsThemeEffect(parseTheme.Get())

	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrevious int) int { return parsePrevious + 1 })
	})
	parseToggleTheme := ui.UseEvent(func() {
		parseTheme.Update(func(parsePrevious string) string {
			if parsePrevious == "ocean" {
				return "midnight"
			}
			return "ocean"
		})
	})
	parseReset := ui.UseEvent(func() {
		parseCount.Set(1)
		parseTheme.Set("ocean")
	})

	return html.Div(
		html.Props{
			ID:    "kernel-plugin-root",
			Class: "bg-[radial-gradient(circle_at_top_left,rgba(34,211,238,0.12),transparent_28%),linear-gradient(180deg,#07111f_0%,#08111d_44%,#111827_100%)]",
			Data:  map[string]string{"theme": parseTheme.Get()},
		},
		shared.ExamplePage(
			"Kernel Plugin Devtools",
			"pluginruntime + devtools.Panel",
			"This example uses the internal kernel directly. A repo-local plugin contributes one live devtools section by reading DOM, style, event, and runtime2 services through the core plugin layer while the page itself drives a real runtime2-backed ui.ParallelRegion.",
			shared.ExamplePanel(
				"Interaction Surface",
				html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Use the buttons below, then open the devtools panel to confirm the kernel plugin section updates event counts, style variable counts, DOM theme state, and runtime2 region status.")),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					renderKernelPluginDevtoolsButton("kernel-plugin-increment", "Increment region", parseIncrement),
					renderKernelPluginDevtoolsButton("kernel-plugin-theme", "Toggle theme", parseToggleTheme),
					renderKernelPluginDevtoolsButton("kernel-plugin-reset", "Reset", parseReset),
				),
				html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
					shared.ExampleStat("Theme", strings.Title(parseTheme.Get())),
					shared.ExampleStat("Count", fmt.Sprintf("%d", parseCount.Get())),
					shared.ExampleStat("Region", "1 active"),
				),
			),
			shared.ExamplePanel(
				"Runtime2 Surface",
				html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("The card below is rendered through ui.ParallelRegion, so the runtime2 metadata surfaced in devtools is backed by a live tracked host adapter rather than synthetic fixtures.")),
				html.Div(html.Props{Class: "mt-6"},
					ui.ParallelRegion(ui.ParallelRegionSpec[renderKernelPluginDevtoolsRegionProps]{
						RendererID:       getKernelPluginDevtoolsRendererID,
						RegionInstanceID: getKernelPluginDevtoolsRegionID,
						Props: renderKernelPluginDevtoolsRegionProps{
							Count: parseCount.Get(),
							Theme: parseTheme.Get(),
						},
					}),
				),
			),
			ui.CreateElement(devtools.Panel, devtools.PanelProps{
				Title:           "Kernel Plugin Devtools",
				InitiallyOpen:   false,
				RefreshInterval: 250 * time.Millisecond,
				MaxDepth:        5,
			}),
		),
	)
}

// handleKernelPluginDevtoolsThemeEffect syncs example theme CSS variables onto the document root.
func handleKernelPluginDevtoolsThemeEffect(parseTheme string) {
	ui.UseEffect(func() func() {
		getDocument := js.Global().Get("document")
		if !getDocument.Truthy() {
			return nil
		}
		getRoot := getDocument.Get("documentElement")
		if !getRoot.Truthy() {
			return nil
		}
		getStyle := getRoot.Get("style")
		switch parseTheme {
		case "midnight":
			getStyle.Call("setProperty", "--kernel-plugin-accent", "#f97316")
			getStyle.Call("setProperty", "--kernel-plugin-surface", "#431407")
		default:
			getStyle.Call("setProperty", "--kernel-plugin-accent", "#22d3ee")
			getStyle.Call("setProperty", "--kernel-plugin-surface", "#083344")
		}
		return nil
	}, parseTheme)
}

// buildKernelPluginDevtoolsNodeCount counts one DOM snapshot subtree.
func buildKernelPluginDevtoolsNodeCount(parseRoot *pluginruntime.DOMNodeSnapshot) int {
	if parseRoot == nil {
		return 0
	}
	buildCount := 1
	for parseChildIndex := range parseRoot.Children {
		parseChild := parseRoot.Children[parseChildIndex]
		buildCount += buildKernelPluginDevtoolsNodeCount(&parseChild)
	}
	return buildCount
}

// findKernelPluginDevtoolsTheme finds the current example theme from the DOM snapshot tree.
func findKernelPluginDevtoolsTheme(parseRoot *pluginruntime.DOMNodeSnapshot) string {
	if parseRoot == nil {
		return "unknown"
	}
	if parseRoot.Attributes["id"] == "kernel-plugin-root" {
		getTheme := strings.TrimSpace(parseRoot.Attributes["data-theme"])
		if getTheme == "" {
			return "unknown"
		}
		return getTheme
	}
	for parseChildIndex := range parseRoot.Children {
		parseChild := parseRoot.Children[parseChildIndex]
		getTheme := findKernelPluginDevtoolsTheme(&parseChild)
		if getTheme != "unknown" {
			return getTheme
		}
	}
	return "unknown"
}

// formatKernelPluginDevtoolsLatestEvent formats the latest event-ring record for the devtools section.
func formatKernelPluginDevtoolsLatestEvent(parseEvents []pluginruntime.EventRecord) string {
	if len(parseEvents) == 0 {
		return "none"
	}
	getLatest := parseEvents[len(parseEvents)-1]
	buildTarget := strings.TrimSpace(getLatest.Target)
	if buildTarget == "" {
		buildTarget = "unknown-target"
	}
	return fmt.Sprintf("%s on %s", getLatest.Type, buildTarget)
}

// registerKernelPluginDevtoolsRenderer registers the runtime2 region renderer used by the example.
func registerKernelPluginDevtoolsRenderer() {
	if parseErr := ui.RegisterParallelRegion(getKernelPluginDevtoolsRendererID, renderKernelPluginDevtoolsRegion); parseErr != nil {
		panic(parseErr)
	}
}

// registerKernelPluginDevtoolsPlugin registers the example kernel plugin before UI bootstrap.
func registerKernelPluginDevtoolsPlugin() {
	if parseErr := pluginruntime.RegisterBuiltinPlugin(pluginruntime.PluginRegistration{
		Factory: func() pluginruntime.Plugin {
			return buildKernelPluginDevtoolsPlugin{}
		},
	}); parseErr != nil {
		panic(parseErr)
	}
}

// main registers the example renderer and kernel plugin, then mounts the example UI.
func main() {
	utils.DisableAllDebug()
	registerKernelPluginDevtoolsRenderer()
	registerKernelPluginDevtoolsPlugin()
	ui.Render(ui.CreateElement(renderKernelPluginDevtoolsExample), "#app")
	select {}
}
