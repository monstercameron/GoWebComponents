package shared

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func ExamplePage(title, feature, summary string, content ...ui.Node) ui.Node {
	documentation := ExampleDocumentation(title, feature, summary)
	content = append(content, documentation...)

	children := []ui.Node{
		html.Div(html.Props{Class: "mx-auto max-w-5xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-[radial-gradient(circle_at_top,_rgba(59,130,246,0.22),_transparent_45%),rgba(15,23,42,0.92)] p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text(feature)),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text(title)),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(summary)),
			),
			html.Div(html.Props{Class: "mt-8 grid gap-6"}, content...),
		),
	}

	return html.Div(html.Props{Class: "min-h-screen bg-[#08111d] text-slate-100"}, children...)
}

func ExampleDocumentation(title, feature, summary string) []ui.Node {
	purposePanel := ExamplePurpose(title, feature)

	functionalDetails := ExamplePanel("Functional details",
		html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text(summary)),
		html.Ul(html.Props{Class: "mt-5 grid gap-3 text-sm leading-7 text-slate-300"},
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text("This page isolates "+title+" so you can see the user-facing behavior without unrelated framework concerns in the way.")),
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text("Use the interactive controls in the panels above, then watch the stats and rendered output change as the example exercises "+feature+".")),
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text("The intent is to show both the functional result and the small amount of state, routing, async work, or DOM wiring needed to make that behavior happen.")),
		),
	)

	implementationDetails := ExamplePanel("Implementation details",
		html.Ul(html.Props{Class: "mt-3 grid gap-3 text-sm leading-7 text-slate-300"},
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text("Each catalog example is a focused Go js/wasm package that mounts a single component tree into #app with ui.Render(ui.CreateElement(...), \"#app\").")),
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text("The visible cards on the page intentionally expose internal state, computed values, or current route information so the implementation mechanics stay inspectable while you interact with the example.")),
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text("The compiled wasm output is generated under examples/static/bin/, and the examples dev server serves the html entrypoint, shared static assets, and wasm bundle together for local testing.")),
		),
		ExampleCode(
			"func main() {",
			"    ui.Render(ui.CreateElement(exampleComponent), \"#app\")",
			"    select {}",
			"}",
		),
	)

	return []ui.Node{purposePanel, functionalDetails, implementationDetails}
}

func ExamplePurpose(title, feature string) ui.Node {
	lead, bullets := purposeCopy(title, feature)
	items := make([]ui.Node, 0, len(bullets))
	for _, bullet := range bullets {
		items = append(items,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text(bullet)),
		)
	}

	return ExamplePanel("Tool purpose",
		html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text(lead)),
		html.Ul(html.Props{Class: "mt-5 grid gap-3 text-sm leading-7 text-slate-300"}, items...),
	)
}

func purposeCopy(title, feature string) (string, []string) {
	switch {
	case feature == "ui.UseState":
		return "Use ui.UseState when a single component owns interactive local state and should rerender itself when that state changes.", []string{
			"Reach for this when the value only matters inside one feature surface, such as counters, toggles, local form draft state, or small view modes.",
			"Keep shared or cross-route data out of ui.UseState and move that into atoms, loaders, or another shared mechanism when multiple parts of the app need the same source of truth.",
		}
	case feature == "ui.UseEffect":
		return "Use ui.UseEffect when a component must synchronize with work outside the pure render path, such as timers, DOM APIs, subscriptions, or document metadata.", []string{
			"The effect body should connect the external system, and the cleanup should disconnect it so rerenders and unmounts do not leak work.",
			"This tool exists for side effects, not for computing display values that can be derived directly during render.",
		}
	case feature == "ui.Render":
		return "Use ui.Render to mount a component tree into a real DOM container and start the browser-facing app surface.", []string{
			"This is the normal client entrypoint for focused examples and browser-only applications.",
			"It answers the question: what do I mount, and where in the document do I mount it?",
		}
	case feature == "ui.RenderToString":
		return "Use ui.RenderToString when you need server-generated HTML before the wasm client starts, such as SSR previews, crawlers, or fast first paint flows.", []string{
			"It is about producing HTML output, not attaching interactivity by itself.",
			"Pair it with hydration when the server HTML should later resume into a live client tree.",
		}
	case feature == "ui.Hydrate":
		return "Use ui.Hydrate when matching HTML already exists in the DOM and the client should attach behavior without replacing that markup wholesale.", []string{
			"It is the right tool for SSR and prerender flows where the first paint comes from HTML that shipped before wasm execution.",
			"Hydration is valuable when preserving existing DOM matters for startup speed, user perception, or route-specific server output.",
		}
	case feature == "ui.RenderBootstrapScript, ui.ReadBootstrapScript":
		return "Use bootstrap script helpers when server-rendered HTML needs structured startup data embedded directly into the page and restored during hydration.", []string{
			"This keeps the initial client resume aligned with the exact server-rendered route or state payload.",
			"It is most relevant when the server already knows the first route data and you want to avoid an immediate duplicate client fetch.",
		}
	case feature == "router.HydrateMount":
		return "Use router.HydrateMount when a route was hydrated first and the router should attach listeners without forcing an immediate replacement render of that initial route.", []string{
			"This exists to preserve the prerendered or server-rendered first route while still enabling normal client navigation afterward.",
			"It is specifically about router startup behavior in hydration flows, not general routing.",
		}
	case strings.HasPrefix(feature, "ui."):
		return "Use " + feature + " when the concern belongs inside the component tree itself: local state, rendering, effects, composition, or UI-specific event flow.", []string{
			"These examples are meant to answer where the API fits in a component's lifecycle and what kind of UI problem it solves.",
			"If the behavior is local to one view and primarily changes rendering, this family of tools is usually the right place to start.",
		}
	case strings.HasPrefix(feature, "state."):
		return "Use " + feature + " when state must outlive one component, be derived from shared values, or be snapshotted, restored, or persisted.", []string{
			"These examples focus on shared state coordination rather than local component-only interactions.",
			"The purpose of the tool is to make state relationships inspectable and reusable across multiple views.",
		}
	case strings.HasPrefix(feature, "fetch."):
		return "Use " + feature + " when the UI needs to load, cache, retry, or imperatively request async data and expose that lifecycle to the page.", []string{
			"These examples show what the user sees while work is pending, successful, retried, cancelled, or failed.",
			"The tool purpose is not just fetching data, but shaping how async work participates in rendering.",
		}
	case strings.HasPrefix(feature, "router."):
		return "Use " + feature + " when the URL should drive view state, navigation, route data loading, or hydration-aware startup behavior.", []string{
			"These examples demonstrate why a route tool exists by tying a visible screen change to path, params, query, redirect, guard, or loader behavior.",
			"If the state must be shareable through navigation and browser history, this is usually the right family of tools.",
		}
	case strings.HasPrefix(feature, "html."):
		return "Use " + feature + " when you want direct, semantic DOM construction helpers without dropping into string-based templates.", []string{
			"These examples show how the helper affects markup shape, semantics, and typed props rather than app-wide state flow.",
			"The tool purpose is clearer when you inspect the resulting HTML structure alongside the rendered output.",
		}
	case strings.HasPrefix(feature, "devtools."):
		return "Use " + feature + " when you need runtime inspection, snapshots, profiling hints, or diagnostics while developing GoWebComponents applications.", []string{
			"These tools are for understanding app behavior, not for driving user-facing product features directly.",
			"The examples focus on what diagnostic surface the tool exposes and when that surface is valuable during debugging.",
		}
	default:
		return "Use this example to understand the purpose of " + title + " in isolation before mixing it into a larger integrated app.", []string{
			"The page is intentionally scoped so the API or tool's job is visible without unrelated framework behavior hiding the core idea.",
			"Read the live stats and output changes as the explanation of what the tool is for, not just as decorative UI.",
		}
	}
}

func ExamplePanel(title string, body ...ui.Node) ui.Node {
	children := append([]ui.Node{
		html.H2(html.Props{Class: "text-xl font-bold text-white"}, html.Text(title)),
	}, body...)
	return html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"}, children...)
}

func ExampleButton(label string, handler ui.Handler) ui.Node {
	return html.Button(
		html.Props{
			OnClick: handler,
			Class:   "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80",
		},
		html.Text(label),
	)
}

func ExampleStat(label, value string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/60 p-4"},
		html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(label)),
		html.P(html.Props{Class: "mt-3 text-3xl font-black text-white"}, html.Text(value)),
	)
}

func ExampleCode(lines ...string) ui.Node {
	children := make([]ui.Node, 0, len(lines))
	for _, line := range lines {
		children = append(children, html.Text(line+"\n"))
	}
	return html.Pre(html.Props{Class: "overflow-x-auto rounded-2xl border border-white/10 bg-black/40 p-4 text-sm text-slate-300"},
		html.Code(html.Props{}, children...),
	)
}
