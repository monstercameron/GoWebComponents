//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/head"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/plugin"
	"github.com/monstercameron/GoWebComponents/v5/router"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

const pluginLogKey = "catalog-plugin-example-log"

const companionHostStory = "This example demonstrates the public companion host, not the internal framework plugin kernel."

func appendHostLog(parseHost *plugin.Host, parseMessage string) {
	parseCurrent, parseOk := parseHost.Value(pluginLogKey)
	parseEntries, _ := parseCurrent.([]string)
	if !parseOk {
		parseEntries = nil
	}
	parseNext := append(append([]string(nil), parseEntries...), strings.TrimSpace(parseMessage))
	parseHost.SetValue(pluginLogKey, parseNext)
}

func hostLogs(parseHost *plugin.Host) []string {
	parseCurrent, parseOk := parseHost.Value(pluginLogKey)
	parseEntries, _ := parseCurrent.([]string)
	if !parseOk {
		return nil
	}
	return append([]string(nil), parseEntries...)
}

func seoPlugin() plugin.Plugin {
	return plugin.Define(plugin.Manifest{
		ID:          "seo-companion",
		Version:     "0.1.0",
		Description: "Provides SSR head nodes, a devtools panel, and bootstrap metadata.",
		Tier:        plugin.TierSupportedCompanion,
		Requires:    []plugin.Capability{plugin.CapabilitySSR, plugin.CapabilityDevtools},
	}, func(parseHost *plugin.Host) (plugin.CleanupFunc, error) {
		if parseErr := parseHost.AddHeadProvider(func() ui.Node {
			return head.Compose(
				router.Metadata{
					Title:        "Plugin Host",
					Description:  "Compose explicit plugin contributions without privileged runtime access.",
					CanonicalURL: "https://example.local/plugins",
				},
				head.Robots("index,follow"),
				head.SocialTags(head.SocialMetadata{
					Type:        "website",
					Title:       "Plugin Host",
					Description: "Supported companion plugin host layered on public APIs.",
					URL:         "https://example.local/plugins",
					TwitterCard: "summary",
				}),
			)
		}); parseErr != nil {
			return nil, parseErr
		}
		if parseErr2 := parseHost.AddBootstrapProvider(func() plugin.BootstrapPayload {
			return plugin.BootstrapPayload{
				Namespace: "seo",
				Data: map[string]interface{}{
					"canonical": "https://example.local/plugins",
					"robots":    "index,follow",
				},
			}
		}); parseErr2 != nil {
			return nil, parseErr2
		}
		if parseErr3 := parseHost.AddPanelProvider(func() plugin.Panel {
			return plugin.Panel{ID: "seo-preview", Title: "SEO Preview", Summary: "Shows head and crawl metadata exported by a plugin."}
		}); parseErr3 != nil {
			return nil, parseErr3
		}
		appendHostLog(parseHost, "seo-companion registered head and bootstrap providers")
		return func() error {
			appendHostLog(parseHost, "seo-companion cleaned up")
			return nil
		}, nil
	})
}

func policyPlugin() plugin.Plugin {
	return plugin.Define(plugin.Manifest{
		ID:          "policy-guard",
		Version:     "0.1.0",
		Description: "Adds route guard and form validation rules.",
		Tier:        plugin.TierExperimental,
		Requires:    []plugin.Capability{plugin.CapabilityRouter, plugin.CapabilityForms},
	}, func(parseHost *plugin.Host) (plugin.CleanupFunc, error) {
		if parseErr := parseHost.AddRouteGuard(func(parseRequest plugin.RouteRequest) plugin.GuardDecision {
			if parseRequest.Path == "/admin" {
				return plugin.Redirect("/signin", "Admin routes require a signed-in operator.")
			}
			return plugin.Allow("Public route stays available.")
		}); parseErr != nil {
			return nil, parseErr
		}
		if parseErr2 := parseHost.AddFormValidator(func(parseSubmission plugin.FormSubmission) []plugin.ValidationIssue {
			parseIssues := make([]plugin.ValidationIssue, 0, 2)
			if strings.TrimSpace(parseSubmission.Values["quantity"]) == "0" || strings.TrimSpace(parseSubmission.Values["quantity"]) == "" {
				parseIssues = append(parseIssues, plugin.ValidationIssue{Field: "quantity", Message: "Quantity must be greater than zero."})
			}
			if strings.TrimSpace(parseSubmission.Values["workflow"]) == "" {
				parseIssues = append(parseIssues, plugin.ValidationIssue{Field: "workflow", Message: "Choose a workflow before submit."})
			}
			return parseIssues
		}); parseErr2 != nil {
			return nil, parseErr2
		}
		if parseErr3 := parseHost.AddSubmitObserver(func(parseSubmission2 plugin.FormSubmission) {
			appendHostLog(parseHost, fmt.Sprintf("policy-guard observed submit intent=%s form=%s", parseSubmission2.Intent, parseSubmission2.ID))
		}); parseErr3 != nil {
			return nil, parseErr3
		}
		appendHostLog(parseHost, "policy-guard installed route and form rules")
		return func() error {
			appendHostLog(parseHost, "policy-guard cleaned up")
			return nil
		}, nil
	})
}

func telemetryPlugin() plugin.Plugin {
	return plugin.Define(plugin.Manifest{
		ID:          "telemetry-audit",
		Version:     "0.1.0",
		Description: "Decorates cache keys and records navigation or request events.",
		Tier:        plugin.TierExperimental,
		Requires:    []plugin.Capability{plugin.CapabilityRouter, plugin.CapabilityAsyncData, plugin.CapabilityDevtools},
	}, func(parseHost *plugin.Host) (plugin.CleanupFunc, error) {
		if parseErr := parseHost.AddCacheKeyDecorator(func(parseKey string) string {
			return "plugins:" + strings.TrimSpace(parseKey)
		}); parseErr != nil {
			return nil, parseErr
		}
		if parseErr2 := parseHost.AddNavigationObserver(func(parseEvent plugin.NavigationEvent) {
			appendHostLog(parseHost, fmt.Sprintf("telemetry-audit saw navigation to %s via %s", parseEvent.Path, parseEvent.Source))
		}); parseErr2 != nil {
			return nil, parseErr2
		}
		if parseErr3 := parseHost.AddRequestObserver(func(parseEvent2 plugin.RequestEvent) {
			appendHostLog(parseHost, fmt.Sprintf("telemetry-audit saw request %s phase=%s", parseEvent2.Key, parseEvent2.Phase))
		}); parseErr3 != nil {
			return nil, parseErr3
		}
		if parseErr4 := parseHost.AddPanelProvider(func() plugin.Panel {
			return plugin.Panel{ID: "telemetry-audit", Title: "Telemetry Audit", Summary: "Tracks navigation and async-data events through plugin hooks."}
		}); parseErr4 != nil {
			return nil, parseErr4
		}
		appendHostLog(parseHost, "telemetry-audit installed route and async-data observers")
		return func() error {
			appendHostLog(parseHost, "telemetry-audit cleaned up")
			return nil
		}, nil
	})
}

func buildHost() (*plugin.Host, error) {
	parseHost := plugin.NewHost(plugin.HostOptions{Capabilities: []plugin.Capability{
		plugin.CapabilityRouter,
		plugin.CapabilityAsyncData,
		plugin.CapabilityDevtools,
		plugin.CapabilitySSR,
		plugin.CapabilityForms,
	}})
	parseHost.SetValue(pluginLogKey, []string{"plugin host created with router, async-data, devtools, SSR, and forms capabilities"})
	for _, parseCurrent := range []plugin.Plugin{seoPlugin(), policyPlugin(), telemetryPlugin()} {
		if parseErr := parseHost.Register(parseCurrent); parseErr != nil {
			return nil, parseErr
		}
	}
	return parseHost, nil
}

func renderHeadPreview(parseHost *plugin.Host) string {
	parseMarkup, parseErr := ui.RenderToString(ui.Fragment(parseHost.HeadNodes()...))
	if parseErr != nil {
		return "Failed to render head preview: " + parseErr.Error()
	}
	return parseMarkup
}

func bootstrapLines(parseHost *plugin.Host) []string {
	parsePayloads := parseHost.BootstrapData()
	parseKeys := make([]string, 0, len(parsePayloads))
	for parseKey := range parsePayloads {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	parseLines := make([]string, 0, len(parseKeys))
	for _, parseKey2 := range parseKeys {
		parseFields := make([]string, 0, len(parsePayloads[parseKey2]))
		for parseField := range parsePayloads[parseKey2] {
			parseFields = append(parseFields, parseField)
		}
		sort.Strings(parseFields)
		parseLines = append(parseLines, fmt.Sprintf("%s: %s", parseKey2, strings.Join(parseFields, ", ")))
	}
	return parseLines
}

func formatDecision(parseDecision plugin.GuardDecision) string {
	switch parseDecision.Outcome {
	case plugin.GuardRedirect:
		return fmt.Sprintf("redirect to %s (%s)", parseDecision.Redirect, parseDecision.Reason)
	case plugin.GuardBlock:
		return "blocked: " + parseDecision.Reason
	default:
		return "allowed: " + parseDecision.Reason
	}
}

func formatIssues(parseIssues []plugin.ValidationIssue) string {
	if len(parseIssues) == 0 {
		return "No validation issues. The form plugin would allow this submit."
	}
	parseParts := make([]string, 0, len(parseIssues))
	for _, parseIssue := range parseIssues {
		parseParts = append(parseParts, parseIssue.Field+": "+parseIssue.Message)
	}
	return strings.Join(parseParts, " | ")
}

func pluginExample() ui.Node {
	parseHostRef := ui.UseRef[*plugin.Host](nil)
	parseStartupError := ""
	if parseHostRef.Get() == nil {
		parseHost, parseErr := buildHost()
		if parseErr != nil {
			parseStartupError = parseErr.Error()
		} else {
			parseHostRef.Set(parseHost)
		}
	}
	parseHost2 := parseHostRef.Get()

	parseRouteResult := ui.UseState("No route decision evaluated yet.")
	parseDecoratedKey := ui.UseState("Click a request action to decorate a cache key.")
	parseValidation := ui.UseState("No validation run yet.")
	parseLogLines := ui.UseState([]string{"The plugin host will append lifecycle and observer events here."})

	ui.UseEffect(func() func() {
		if parseHost2 == nil {
			return nil
		}
		parseLogLines.Set(hostLogs(parseHost2))
		return func() {
			_ = parseHost2.Close()
		}
	}, parseHost2)

	parseTriggerRoute := func(parsePath string) ui.Handler {
		return ui.UseEvent(func() {
			if parseHost2 == nil {
				return
			}
			parseDecision := parseHost2.EvaluateRoute(plugin.RouteRequest{Path: parsePath})
			parseHost2.NotifyNavigation(plugin.NavigationEvent{Path: parsePath, Source: "plugin-example"})
			parseRouteResult.Set(formatDecision(parseDecision))
			parseLogLines.Set(hostLogs(parseHost2))
		})
	}

	parseTriggerRequest := func(parseKey string) ui.Handler {
		return ui.UseEvent(func() {
			if parseHost2 == nil {
				return
			}
			parseDecorated := parseHost2.DecorateCacheKey(parseKey)
			parseHost2.NotifyRequest(plugin.RequestEvent{Key: parseDecorated, Phase: "ready", Source: "plugin-example"})
			parseDecoratedKey.Set(parseDecorated)
			parseLogLines.Set(hostLogs(parseHost2))
		})
	}

	parseValidate := func(parseValues map[string]string, parseIntent string) ui.Handler {
		return ui.UseEvent(func() {
			if parseHost2 == nil {
				return
			}
			parseSubmission := plugin.FormSubmission{ID: "purchase-order", Intent: parseIntent, Values: parseValues}
			parseIssues := parseHost2.ValidateForm(parseSubmission)
			parseHost2.NotifySubmit(parseSubmission)
			parseValidation.Set(formatIssues(parseIssues))
			parseLogLines.Set(hostLogs(parseHost2))
		})
	}

	parseRegistered := []ui.Node{}
	parseCapabilities := []ui.Node{}
	parsePanels := []ui.Node{}
	parseBootstrap := []ui.Node{}
	parseHeadPreview := "Plugin host unavailable."
	if parseStartupError != "" {
		parseHeadPreview = "Startup error: " + parseStartupError
	}
	if parseHost2 != nil {
		for _, parseCapability := range parseHost2.Capabilities() {
			parseCapabilities = append(parseCapabilities, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text(string(parseCapability))))
		}
		for _, parseManifest := range parseHost2.Plugins() {
			parseRequires := make([]string, 0, len(parseManifest.Requires))
			for _, parseCapability2 := range parseManifest.Requires {
				parseRequires = append(parseRequires, string(parseCapability2))
			}
			parseRegistered = append(parseRegistered, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/60 p-4"},
				html.H3(html.Props{Class: "text-lg font-bold text-white"}, html.Text(parseManifest.ID)),
				html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(parseManifest.Description)),
				html.P(html.Props{Class: "mt-3 text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text(string(parseManifest.Tier))),
				html.P(html.Props{Class: "mt-2 text-sm text-slate-400"}, html.Text("Requires: "+strings.Join(parseRequires, ", "))),
			))
		}
		for _, parsePanel := range parseHost2.Panels() {
			parsePanels = append(parsePanels, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text(parsePanel.Title+": "+parsePanel.Summary)))
		}
		for _, parseLine := range bootstrapLines(parseHost2) {
			parseBootstrap = append(parseBootstrap, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text(parseLine)))
		}
		parseHeadPreview = renderHeadPreview(parseHost2)
	}

	return shared.ExamplePage(
		"Plugin Host",
		"plugin.Host, plugin.Plugin",
		"Register explicit companion plugins, declare capabilities, contribute subsystem hooks, and keep registration or cleanup in normal application code instead of hidden runtime discovery. "+companionHostStory,
		shared.ExamplePanel("Host summary",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300", ID: "plugin-host-summary"}, html.Text("The host exposes explicit capabilities, manifest metadata, and ordered registration without giving plugins privileged runtime access.")),
			html.Div(html.Props{Class: "mt-4 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Capabilities", fmt.Sprintf("%d", len(parseCapabilities))),
				shared.ExampleStat("Registered plugins", fmt.Sprintf("%d", len(parseRegistered))),
				shared.ExampleStat("Devtools panels", fmt.Sprintf("%d", len(parsePanels))),
			),
			html.Ul(html.Props{Class: "mt-4 grid gap-3 text-sm text-slate-300 md:grid-cols-2"}, parseRegistered...),
			html.H3(html.Props{Class: "mt-6 text-lg font-bold text-white"}, html.Text("Enabled capabilities")),
			html.Ul(html.Props{Class: "mt-3 grid gap-3 text-sm text-slate-300 md:grid-cols-3"}, parseCapabilities...),
		),
		shared.ExamplePanel("Route and async hooks",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("One plugin guards admin routes, and another decorates async-data keys plus logs navigation or request events. The host keeps those hooks ordered and explicit.")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Evaluate pricing route", parseTriggerRoute("/pricing")),
				shared.ExampleButton("Evaluate admin route", parseTriggerRoute("/admin")),
				shared.ExampleButton("Decorate inventory request", parseTriggerRequest("inventory:list")),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300", ID: "plugin-route-result"}, html.Text(parseRouteResult.Get())),
			html.P(html.Props{Class: "mt-2 text-sm leading-7 text-slate-300", ID: "plugin-cache-key"}, html.Text(parseDecoratedKey.Get())),
		),
		shared.ExamplePanel("Form and lifecycle hooks",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Form validators and submit observers are contributed the same way: explicit setup plus explicit cleanup, with no internal hook-slot access.")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Validate invalid order", parseValidate(map[string]string{"quantity": "0", "workflow": ""}, "submit")),
				shared.ExampleButton("Validate ready order", parseValidate(map[string]string{"quantity": "5", "workflow": "expedite"}, "submit")),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300", ID: "plugin-validation-result"}, html.Text(parseValidation.Get())),
			html.H3(html.Props{Class: "mt-6 text-lg font-bold text-white"}, html.Text("Plugin activity log")),
			html.Ul(html.Props{Class: "mt-3 grid gap-3 text-sm text-slate-300"}, func() []ui.Node {
				parseEntries := parseLogLines.Get()
				parseChildren := make([]ui.Node, 0, len(parseEntries))
				for _, parseEntry := range parseEntries {
					parseChildren = append(parseChildren, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text(parseEntry)))
				}
				return parseChildren
			}()...),
		),
		shared.ExamplePanel("SSR and devtools contributions",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("The SEO companion plugin contributes head nodes and bootstrap metadata through the host's SSR hooks, while plugins also expose devtools panels without changing core router ownership.")),
			html.H3(html.Props{Class: "mt-4 text-lg font-bold text-white"}, html.Text("Head preview")),
			html.Pre(html.Props{Class: "mt-3 overflow-x-auto rounded-2xl border border-white/10 bg-black/40 p-4 text-sm text-slate-300", ID: "plugin-head-preview"}, html.Text(parseHeadPreview)),
			html.H3(html.Props{Class: "mt-6 text-lg font-bold text-white"}, html.Text("Bootstrap namespaces")),
			html.Ul(html.Props{Class: "mt-3 grid gap-3 text-sm text-slate-300"}, parseBootstrap...),
			html.H3(html.Props{Class: "mt-6 text-lg font-bold text-white"}, html.Text("Panel providers")),
			html.Ul(html.Props{Class: "mt-3 grid gap-3 text-sm text-slate-300"}, parsePanels...),
		),
		shared.ExamplePanel("Implementation shape",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("The plugin package is a supported companion API. Plugins declare a manifest, require explicit capabilities, register through normal code, and contribute only the documented hooks the host enables.")),
			shared.ExampleCode(
				`host := plugin.NewHost(plugin.HostOptions{Capabilities: []plugin.Capability{...}})`,
				`host.Register(seoPlugin())`,
				`host.Register(policyPlugin())`,
				`host.Register(telemetryPlugin())`,
				`decision := host.EvaluateRoute(plugin.RouteRequest{Path: "/admin"})`,
				`headMarkup, _ := ui.RenderToString(ui.Fragment(host.HeadNodes()...))`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(pluginExample))
	exampleboot.WaitExampleRuntime()
}
