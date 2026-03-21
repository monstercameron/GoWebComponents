//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"sort"
	"strings"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/head"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/plugin"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const pluginLogKey = "catalog-plugin-example-log"

func appendHostLog(host *plugin.Host, message string) {
	current, ok := host.Value(pluginLogKey)
	entries, _ := current.([]string)
	if !ok {
		entries = nil
	}
	next := append(append([]string(nil), entries...), strings.TrimSpace(message))
	host.SetValue(pluginLogKey, next)
}

func hostLogs(host *plugin.Host) []string {
	current, ok := host.Value(pluginLogKey)
	entries, _ := current.([]string)
	if !ok {
		return nil
	}
	return append([]string(nil), entries...)
}

func seoPlugin() plugin.Plugin {
	return plugin.Define(plugin.Manifest{
		ID:          "seo-companion",
		Version:     "0.1.0",
		Description: "Provides SSR head nodes, a devtools panel, and bootstrap metadata.",
		Tier:        plugin.TierSupportedCompanion,
		Requires:    []plugin.Capability{plugin.CapabilitySSR, plugin.CapabilityDevtools},
	}, func(host *plugin.Host) (plugin.CleanupFunc, error) {
		if err := host.AddHeadProvider(func() ui.Node {
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
					Description: "Experimental plugin host layered on companion APIs.",
					URL:         "https://example.local/plugins",
					TwitterCard: "summary",
				}),
			)
		}); err != nil {
			return nil, err
		}
		if err := host.AddBootstrapProvider(func() plugin.BootstrapPayload {
			return plugin.BootstrapPayload{
				Namespace: "seo",
				Data: map[string]interface{}{
					"canonical": "https://example.local/plugins",
					"robots":    "index,follow",
				},
			}
		}); err != nil {
			return nil, err
		}
		if err := host.AddPanelProvider(func() plugin.Panel {
			return plugin.Panel{ID: "seo-preview", Title: "SEO Preview", Summary: "Shows head and crawl metadata exported by a plugin."}
		}); err != nil {
			return nil, err
		}
		appendHostLog(host, "seo-companion registered head and bootstrap providers")
		return func() error {
			appendHostLog(host, "seo-companion cleaned up")
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
	}, func(host *plugin.Host) (plugin.CleanupFunc, error) {
		if err := host.AddRouteGuard(func(request plugin.RouteRequest) plugin.GuardDecision {
			if request.Path == "/admin" {
				return plugin.Redirect("/signin", "Admin routes require a signed-in operator.")
			}
			return plugin.Allow("Public route stays available.")
		}); err != nil {
			return nil, err
		}
		if err := host.AddFormValidator(func(submission plugin.FormSubmission) []plugin.ValidationIssue {
			issues := make([]plugin.ValidationIssue, 0, 2)
			if strings.TrimSpace(submission.Values["quantity"]) == "0" || strings.TrimSpace(submission.Values["quantity"]) == "" {
				issues = append(issues, plugin.ValidationIssue{Field: "quantity", Message: "Quantity must be greater than zero."})
			}
			if strings.TrimSpace(submission.Values["workflow"]) == "" {
				issues = append(issues, plugin.ValidationIssue{Field: "workflow", Message: "Choose a workflow before submit."})
			}
			return issues
		}); err != nil {
			return nil, err
		}
		if err := host.AddSubmitObserver(func(submission plugin.FormSubmission) {
			appendHostLog(host, fmt.Sprintf("policy-guard observed submit intent=%s form=%s", submission.Intent, submission.ID))
		}); err != nil {
			return nil, err
		}
		appendHostLog(host, "policy-guard installed route and form rules")
		return func() error {
			appendHostLog(host, "policy-guard cleaned up")
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
	}, func(host *plugin.Host) (plugin.CleanupFunc, error) {
		if err := host.AddCacheKeyDecorator(func(key string) string {
			return "plugins:" + strings.TrimSpace(key)
		}); err != nil {
			return nil, err
		}
		if err := host.AddNavigationObserver(func(event plugin.NavigationEvent) {
			appendHostLog(host, fmt.Sprintf("telemetry-audit saw navigation to %s via %s", event.Path, event.Source))
		}); err != nil {
			return nil, err
		}
		if err := host.AddRequestObserver(func(event plugin.RequestEvent) {
			appendHostLog(host, fmt.Sprintf("telemetry-audit saw request %s phase=%s", event.Key, event.Phase))
		}); err != nil {
			return nil, err
		}
		if err := host.AddPanelProvider(func() plugin.Panel {
			return plugin.Panel{ID: "telemetry-audit", Title: "Telemetry Audit", Summary: "Tracks navigation and async-data events through plugin hooks."}
		}); err != nil {
			return nil, err
		}
		appendHostLog(host, "telemetry-audit installed route and async-data observers")
		return func() error {
			appendHostLog(host, "telemetry-audit cleaned up")
			return nil
		}, nil
	})
}

func buildHost() (*plugin.Host, error) {
	host := plugin.NewHost(plugin.HostOptions{Capabilities: []plugin.Capability{
		plugin.CapabilityRouter,
		plugin.CapabilityAsyncData,
		plugin.CapabilityDevtools,
		plugin.CapabilitySSR,
		plugin.CapabilityForms,
	}})
	host.SetValue(pluginLogKey, []string{"plugin host created with router, async-data, devtools, SSR, and forms capabilities"})
	for _, current := range []plugin.Plugin{seoPlugin(), policyPlugin(), telemetryPlugin()} {
		if err := host.Register(current); err != nil {
			return nil, err
		}
	}
	return host, nil
}

func renderHeadPreview(host *plugin.Host) string {
	markup, err := ui.RenderToString(ui.Fragment(host.HeadNodes()...))
	if err != nil {
		return "Failed to render head preview: " + err.Error()
	}
	return markup
}

func bootstrapLines(host *plugin.Host) []string {
	payloads := host.BootstrapData()
	keys := make([]string, 0, len(payloads))
	for key := range payloads {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		fields := make([]string, 0, len(payloads[key]))
		for field := range payloads[key] {
			fields = append(fields, field)
		}
		sort.Strings(fields)
		lines = append(lines, fmt.Sprintf("%s: %s", key, strings.Join(fields, ", ")))
	}
	return lines
}

func formatDecision(decision plugin.GuardDecision) string {
	switch decision.Outcome {
	case plugin.GuardRedirect:
		return fmt.Sprintf("redirect to %s (%s)", decision.Redirect, decision.Reason)
	case plugin.GuardBlock:
		return "blocked: " + decision.Reason
	default:
		return "allowed: " + decision.Reason
	}
}

func formatIssues(issues []plugin.ValidationIssue) string {
	if len(issues) == 0 {
		return "No validation issues. The form plugin would allow this submit."
	}
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		parts = append(parts, issue.Field+": "+issue.Message)
	}
	return strings.Join(parts, " | ")
}

func pluginExample() ui.Node {
	hostRef := ui.UseRef[*plugin.Host](nil)
	startupError := ""
	if hostRef.Get() == nil {
		host, err := buildHost()
		if err != nil {
			startupError = err.Error()
		} else {
			hostRef.Set(host)
		}
	}
	host := hostRef.Get()

	routeResult := ui.UseState("No route decision evaluated yet.")
	decoratedKey := ui.UseState("Click a request action to decorate a cache key.")
	validation := ui.UseState("No validation run yet.")
	logLines := ui.UseState([]string{"The plugin host will append lifecycle and observer events here."})

	ui.UseEffect(func() func() {
		if host == nil {
			return nil
		}
		logLines.Set(hostLogs(host))
		return func() {
			_ = host.Close()
		}
	}, host)

	triggerRoute := func(path string) ui.Handler {
		return ui.UseEvent(func() {
			if host == nil {
				return
			}
			decision := host.EvaluateRoute(plugin.RouteRequest{Path: path})
			host.NotifyNavigation(plugin.NavigationEvent{Path: path, Source: "plugin-example"})
			routeResult.Set(formatDecision(decision))
			logLines.Set(hostLogs(host))
		})
	}

	triggerRequest := func(key string) ui.Handler {
		return ui.UseEvent(func() {
			if host == nil {
				return
			}
			decorated := host.DecorateCacheKey(key)
			host.NotifyRequest(plugin.RequestEvent{Key: decorated, Phase: "ready", Source: "plugin-example"})
			decoratedKey.Set(decorated)
			logLines.Set(hostLogs(host))
		})
	}

	validate := func(values map[string]string, intent string) ui.Handler {
		return ui.UseEvent(func() {
			if host == nil {
				return
			}
			submission := plugin.FormSubmission{ID: "purchase-order", Intent: intent, Values: values}
			issues := host.ValidateForm(submission)
			host.NotifySubmit(submission)
			validation.Set(formatIssues(issues))
			logLines.Set(hostLogs(host))
		})
	}

	registered := []ui.Node{}
	capabilities := []ui.Node{}
	panels := []ui.Node{}
	bootstrap := []ui.Node{}
	headPreview := "Plugin host unavailable."
	if startupError != "" {
		headPreview = "Startup error: " + startupError
	}
	if host != nil {
		for _, capability := range host.Capabilities() {
			capabilities = append(capabilities, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text(string(capability))))
		}
		for _, manifest := range host.Plugins() {
			requires := make([]string, 0, len(manifest.Requires))
			for _, capability := range manifest.Requires {
				requires = append(requires, string(capability))
			}
			registered = append(registered, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/60 p-4"},
				html.H3(html.Props{Class: "text-lg font-bold text-white"}, html.Text(manifest.ID)),
				html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(manifest.Description)),
				html.P(html.Props{Class: "mt-3 text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text(string(manifest.Tier))),
				html.P(html.Props{Class: "mt-2 text-sm text-slate-400"}, html.Text("Requires: "+strings.Join(requires, ", "))),
			))
		}
		for _, panel := range host.Panels() {
			panels = append(panels, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text(panel.Title+": "+panel.Summary)))
		}
		for _, line := range bootstrapLines(host) {
			bootstrap = append(bootstrap, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text(line)))
		}
		headPreview = renderHeadPreview(host)
	}

	return shared.ExamplePage(
		"Plugin Host",
		"plugin.Host, plugin.Plugin",
		"Register explicit companion plugins, declare capabilities, contribute subsystem hooks, and keep registration or cleanup in normal application code instead of hidden runtime discovery.",
		shared.ExamplePanel("Host summary",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300", ID: "plugin-host-summary"}, html.Text("The host exposes explicit capabilities, manifest metadata, and ordered registration without giving plugins privileged runtime access.")),
			html.Div(html.Props{Class: "mt-4 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Capabilities", fmt.Sprintf("%d", len(capabilities))),
				shared.ExampleStat("Registered plugins", fmt.Sprintf("%d", len(registered))),
				shared.ExampleStat("Devtools panels", fmt.Sprintf("%d", len(panels))),
			),
			html.Ul(html.Props{Class: "mt-4 grid gap-3 text-sm text-slate-300 md:grid-cols-2"}, registered...),
			html.H3(html.Props{Class: "mt-6 text-lg font-bold text-white"}, html.Text("Enabled capabilities")),
			html.Ul(html.Props{Class: "mt-3 grid gap-3 text-sm text-slate-300 md:grid-cols-3"}, capabilities...),
		),
		shared.ExamplePanel("Route and async hooks",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("One plugin guards admin routes, and another decorates async-data keys plus logs navigation or request events. The host keeps those hooks ordered and explicit.")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Evaluate pricing route", triggerRoute("/pricing")),
				shared.ExampleButton("Evaluate admin route", triggerRoute("/admin")),
				shared.ExampleButton("Decorate inventory request", triggerRequest("inventory:list")),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300", ID: "plugin-route-result"}, html.Text(routeResult.Get())),
			html.P(html.Props{Class: "mt-2 text-sm leading-7 text-slate-300", ID: "plugin-cache-key"}, html.Text(decoratedKey.Get())),
		),
		shared.ExamplePanel("Form and lifecycle hooks",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Form validators and submit observers are contributed the same way: explicit setup plus explicit cleanup, with no internal hook-slot access.")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Validate invalid order", validate(map[string]string{"quantity": "0", "workflow": ""}, "submit")),
				shared.ExampleButton("Validate ready order", validate(map[string]string{"quantity": "5", "workflow": "expedite"}, "submit")),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300", ID: "plugin-validation-result"}, html.Text(validation.Get())),
			html.H3(html.Props{Class: "mt-6 text-lg font-bold text-white"}, html.Text("Plugin activity log")),
			html.Ul(html.Props{Class: "mt-3 grid gap-3 text-sm text-slate-300"}, func() []ui.Node {
				entries := logLines.Get()
				children := make([]ui.Node, 0, len(entries))
				for _, entry := range entries {
					children = append(children, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text(entry)))
				}
				return children
			}()...),
		),
		shared.ExamplePanel("SSR and devtools contributions",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("The SEO companion plugin contributes head nodes and bootstrap metadata through the host's SSR hooks, while plugins also expose devtools panels without changing core router ownership.")),
			html.H3(html.Props{Class: "mt-4 text-lg font-bold text-white"}, html.Text("Head preview")),
			html.Pre(html.Props{Class: "mt-3 overflow-x-auto rounded-2xl border border-white/10 bg-black/40 p-4 text-sm text-slate-300", ID: "plugin-head-preview"}, html.Text(headPreview)),
			html.H3(html.Props{Class: "mt-6 text-lg font-bold text-white"}, html.Text("Bootstrap namespaces")),
			html.Ul(html.Props{Class: "mt-3 grid gap-3 text-sm text-slate-300"}, bootstrap...),
			html.H3(html.Props{Class: "mt-6 text-lg font-bold text-white"}, html.Text("Panel providers")),
			html.Ul(html.Props{Class: "mt-3 grid gap-3 text-sm text-slate-300"}, panels...),
		),
		shared.ExamplePanel("Implementation shape",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("The plugin package is an experimental companion API. Plugins declare a manifest, require explicit capabilities, register through normal code, and contribute only the documented hooks the host enables.")),
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
	ui.Render(ui.CreateElement(pluginExample), "#app")
	select {}
}
