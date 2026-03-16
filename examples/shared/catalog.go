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
		html.Div(html.Props{Class: "mx-auto max-w-6xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2.25rem] border border-cyan-300/15 bg-[linear-gradient(135deg,rgba(15,23,42,0.96),rgba(15,23,42,0.76))] p-8 shadow-[0_24px_60px_rgba(2,6,23,0.45)] backdrop-blur-sm"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text(feature)),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text(title)),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(summary)),
			),
			html.Div(html.Props{Class: "mt-8 grid gap-6"}, content...),
		),
	}

	return html.Div(html.Props{Class: "min-h-screen text-slate-100"}, children...)
}

func ExampleDocumentation(title, feature, summary string) []ui.Node {
	overviewLead, overviewBullets := overviewCopy(title, feature)
	functionalBullets := functionalCopy(title, feature)
	implementationLead, implementationBullets, codeLines := implementationCopy(title, feature)

	overviewPanel := ExamplePanel("Overview",
		html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text(overviewLead)),
		ExampleBulletList(overviewBullets...),
	)

	functionalDetails := ExamplePanel("Functional",
		html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text(summary)),
		ExampleBulletList(functionalBullets...),
	)

	implementationChildren := []ui.Node{
		html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text(implementationLead)),
		ExampleBulletList(implementationBullets...),
	}
	if len(codeLines) > 0 {
		implementationChildren = append(implementationChildren, ExampleCode(codeLines...))
	}
	implementationDetails := ExamplePanel("Implementation", implementationChildren...)

	return []ui.Node{overviewPanel, functionalDetails, implementationDetails}
}

func overviewCopy(title, feature string) (string, []string) {
	subject := exampleSubject(title, feature)
	switch {
	case subject == "ui.UseState":
		return "Use ui.UseState when a single component owns interactive local state and should rerender itself when that state changes.", []string{
			"Reach for this when the value only matters inside one feature surface, such as counters, toggles, local form draft state, or small view modes.",
			"Keep shared or cross-route data out of ui.UseState and move that into atoms, loaders, or another shared mechanism when multiple parts of the app need the same source of truth.",
		}
	case subject == "ui.UseEffect":
		return "Use ui.UseEffect when a component must synchronize with work outside the pure render path, such as timers, DOM APIs, subscriptions, or document metadata.", []string{
			"The effect body should connect the external system, and the cleanup should disconnect it so rerenders and unmounts do not leak work.",
			"This tool exists for side effects, not for computing display values that can be derived directly during render.",
		}
	case subject == "ui.UseReducer":
		return "Use ui.UseReducer when local state has named transitions, branch-heavy updates, or rules that read more clearly as actions than as scattered set calls.", []string{
			"Reducers work best when the legal state changes are important enough to name, test, and reason about directly.",
			"If the state is a single value with trivial updates, ui.UseState usually stays simpler and easier to scan.",
		}
	case subject == "ui.Render":
		return "Use ui.Render to mount a component tree into a real DOM container and start the browser-facing app surface.", []string{
			"This is the normal client entrypoint for focused examples and browser-only applications.",
			"It answers the question: what do I mount, and where in the document do I mount it?",
		}
	case subject == "ui.RenderToString":
		return "Use ui.RenderToString when you need server-generated HTML before the wasm client starts, such as SSR previews, crawlers, or fast first paint flows.", []string{
			"It is about producing HTML output, not attaching interactivity by itself.",
			"Pair it with hydration when the server HTML should later resume into a live client tree.",
		}
	case subject == "ui.Hydrate":
		return "Use ui.Hydrate when matching HTML already exists in the DOM and the client should attach behavior without replacing that markup wholesale.", []string{
			"It is the right tool for SSR and prerender flows where the first paint comes from HTML that shipped before wasm execution.",
			"Hydration is valuable when preserving existing DOM matters for startup speed, user perception, or route-specific server output.",
		}
	case subject == "ui.RenderBootstrapScript, ui.ReadBootstrapScript":
		return "Use bootstrap script helpers when server-rendered HTML needs structured startup data embedded directly into the page and restored during hydration.", []string{
			"This keeps the initial client resume aligned with the exact server-rendered route or state payload.",
			"It is most relevant when the server already knows the first route data and you want to avoid an immediate duplicate client fetch.",
		}
	case subject == "router.HydrateMount":
		return "Use router.HydrateMount when a route was hydrated first and the router should attach listeners without forcing an immediate replacement render of that initial route.", []string{
			"This exists to preserve the prerendered or server-rendered first route while still enabling normal client navigation afterward.",
			"It is specifically about router startup behavior in hydration flows, not general routing.",
		}
	case strings.HasPrefix(subject, "ui."):
		return "Use " + subject + " when the concern belongs inside the component tree itself: local state, rendering, effects, composition, or UI-specific event flow.", []string{
			"These examples are meant to answer where the API fits in a component's lifecycle and what kind of UI problem it solves.",
			"If the behavior is local to one view and primarily changes rendering, this family of tools is usually the right place to start.",
		}
	case strings.HasPrefix(subject, "state."):
		return "Use " + subject + " when state must outlive one component, be derived from shared values, or be snapshotted, restored, or persisted.", []string{
			"These examples focus on shared state coordination rather than local component-only interactions.",
			"The purpose of the tool is to make state relationships inspectable and reusable across multiple views.",
		}
	case strings.HasPrefix(subject, "fetch."):
		return "Use " + subject + " when the UI needs to load, cache, retry, or imperatively request async data and expose that lifecycle to the page.", []string{
			"These examples show what the user sees while work is pending, successful, retried, cancelled, or failed.",
			"The tool purpose is not just fetching data, but shaping how async work participates in rendering.",
		}
	case strings.HasPrefix(subject, "router.") || strings.Contains(subject, "router") || strings.Contains(strings.ToLower(feature), "route"):
		return "Use " + subject + " when the URL should drive view state, navigation, route data loading, or hydration-aware startup behavior.", []string{
			"These examples demonstrate why a route tool exists by tying a visible screen change to path, params, query, redirect, guard, or loader behavior.",
			"If the state must be shareable through navigation and browser history, this is usually the right family of tools.",
		}
	case strings.HasPrefix(subject, "html."):
		return "Use " + subject + " when you want direct, semantic DOM construction helpers without dropping into string-based templates.", []string{
			"These examples show how the helper affects markup shape, semantics, and typed props rather than app-wide state flow.",
			"The tool purpose is clearer when you inspect the resulting HTML structure alongside the rendered output.",
		}
	case strings.HasPrefix(subject, "devtools."):
		return "Use " + subject + " when you need runtime inspection, snapshots, profiling hints, or diagnostics while developing GoWebComponents applications.", []string{
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

func functionalCopy(title, feature string) []string {
	subject := exampleSubject(title, feature)
	bullets := []string{
		"This page isolates " + title + " so you can see the user-facing behavior without unrelated framework concerns in the way.",
		"Use the interactive controls in the panels above, then watch the stats and rendered output change as the example exercises " + feature + ".",
	}

	switch {
	case subject == "ui.UseReducer":
		bullets = append(bullets,
			"Reach for reducers when multiple buttons, events, or async results all need to drive the same state machine in a predictable way.",
			"The payoff is that the UI reads in terms of actions and outcomes instead of manual state patching spread across handlers.",
		)
	case strings.HasPrefix(subject, "ui."):
		bullets = append(bullets,
			"Use this family of tools when the behavior is local to one rendered surface and you need the UI tree itself to explain the state change.",
			"Prefer the smallest hook or primitive that matches the job so the example stays easy to reason about and reuse.",
		)
	case strings.HasPrefix(subject, "state."):
		bullets = append(bullets,
			"Use these controls to see how shared values, computed values, or snapshots propagate across multiple readers without duplicating the write path.",
			"Choose the state package when the data relationship matters more than the local widget that happens to display it.",
		)
	case strings.HasPrefix(subject, "fetch."):
		bullets = append(bullets,
			"Watch how loading, ready, retry, cancellation, or error states affect what the user sees before choosing a higher-level data pattern.",
			"Prefer the lowest fetch abstraction that still makes the lifecycle readable for your screen.",
		)
	case strings.HasPrefix(subject, "router.") || strings.Contains(strings.ToLower(feature), "route"):
		bullets = append(bullets,
			"Use the navigation controls and visible route stats together so the path, params, query, or loader output stays tied to the rendered screen.",
			"Choose router primitives when the state should survive refreshes, deep links, or browser history changes.",
		)
	case strings.HasPrefix(subject, "html."):
		bullets = append(bullets,
			"Inspect both the rendered structure and the user-facing output so the DOM shape stays as important as the visual result.",
			"These helpers are best when semantic markup clarity matters more than abstract component state transitions.",
		)
	case strings.HasPrefix(subject, "devtools."):
		bullets = append(bullets,
			"Use the example to compare visible app behavior with the diagnostic surface so you can tell what the tool adds during debugging.",
			"The value here is observability: choose these APIs when understanding runtime behavior is the primary goal.",
		)
	default:
		bullets = append(bullets,
			"The intent is to show both the functional result and the small amount of state, routing, async work, or DOM wiring needed to make that behavior happen.",
		)
	}

	return bullets
}

func implementationCopy(title, feature string) (string, []string, []string) {
	subject := exampleSubject(title, feature)
	lead := "This example keeps the implementation small enough that the rendered behavior maps directly to the Go code driving the component tree, hook state, and event handlers."
	bullets := []string{
		"The compiled wasm output is generated under examples/static/bin/, and the examples dev server serves the HTML entrypoint, shared static assets, and wasm bundle together for local testing.",
	}
	code := []string{
		"func main() {",
		"    ui.Render(ui.CreateElement(exampleComponent), \"#app\")",
		"    select {}",
		"}",
	}

	switch {
	case subject == "ui.UseReducer":
		lead = "ui.UseReducer stores one reducer state cell in the component and routes every change through Dispatch, so the reducer function becomes the single transition table for that local state machine."
		bullets = append([]string{
			"Keep the reducer pure: it should derive the next state from the current state and action only, because it runs during render scheduling and must stay deterministic.",
			"Use reducers when several handlers need to converge on the same state rules. For a single scalar or a trivial toggle, ui.UseState usually allocates less code and less mental overhead.",
			"Watch for large state structs copied on every dispatch. If transitions become hot and the state payload grows, split state or move heavier data behind references instead of repeatedly cloning it.",
		}, bullets...)
		code = []string{
			"type action string",
			"func reducer(state workflowState, action action) workflowState { ... }",
			"workflow := ui.UseReducer(reducer, workflowState{Step: \"Draft\"})",
			"workflow.Dispatch(actionReview)",
		}
	case subject == "ui.Render":
		lead = "ui.Render is the client entrypoint: it creates the initial runtime root for a selector and mounts the component tree returned by ui.CreateElement into that DOM target."
		bullets = append([]string{
			"The selector must already exist in the document. If the mount target is missing, the runtime has nowhere to attach the tree.",
			"Render is a full client mount, not hydration. Use ui.Hydrate when the DOM already contains matching SSR or prerendered markup that should be resumed instead of replaced.",
			"Keep main small: initialize dependencies, mount once, and block with select {} so the Go wasm runtime stays alive for events and rerenders.",
		}, bullets...)
	case subject == "ui.RenderToString":
		lead = "This example centers on server-side HTML generation, so the important implementation detail is when markup is produced versus when interactivity is attached later."
		bullets = append([]string{
			"Use render-to-string output for first paint, previews, or non-interactive delivery, then pair it with hydration only when the client must resume the same tree.",
			"Watch out for assuming RenderToString adds event handlers by itself; it only produces markup and serializes the current component output.",
			"Server-side rendering cost scales with the component tree you serialize. Keep request-time render work predictable and avoid burying expensive I/O directly inside render code.",
		}, bullets...)
		code = []string{
			"markup, err := ui.RenderToString(ui.CreateElement(exampleComponent))",
			"if err != nil {",
			"    panic(err)",
			"}",
		}
	case subject == "ui.Hydrate" || subject == "router.HydrateMount" || subject == "ui.RenderBootstrapScript, ui.ReadBootstrapScript":
		lead = "This example relies on existing HTML already being in the document, so the main implementation concern is resuming the correct DOM without replacing useful server output."
		bullets = append([]string{
			"Hydration works best when the client tree matches the server HTML and any bootstrap payload stays aligned with the initial route or state.",
			"Watch out for mismatched initial markup or stale bootstrap data, because those are the fastest ways to lose the benefit of hydration.",
			"The performance win comes from preserving the first paint and avoiding an immediate full rerender. That win disappears if the client has to throw away the prerendered subtree.",
		}, bullets...)
		code = []string{
			"func main() {",
			"    _, _ = ui.Hydrate(ui.CreateElement(exampleComponent), \"#app\")",
			"    select {}",
			"}",
		}
	case strings.HasPrefix(subject, "router.") || strings.Contains(strings.ToLower(feature), "route"):
		lead = "These router examples are implemented as small route tables over the Go router, where Register wires path patterns to page factories and Mount or HydrateMount binds updates to the browser location."
		bullets = append([]string{
			"Hash routers derive route state from window.location.hash and avoid server rewrite requirements. Browser routers read pathname and need the server to return the app entrypoint for deep links.",
			"Route loaders should return compact, serializable attrs and respect context cancellation. If a user navigates away, the loader must stop work quickly instead of continuing background fetches.",
			"Avoid putting large opaque objects into route attrs or query-derived state. Loader output is read on rerender, so keep it small, typed, and easy to inspect.",
		}, bullets...)
		code = []string{
			"func main() {",
			"    r := router.NewHashRouter(router.RouterOptions{DefaultRoute: \"/\"})",
			"    r.Register(\"/\", page)",
			"    r.Mount(\"#app\")",
			"    select {}",
			"}",
		}
	case strings.HasPrefix(subject, "state."):
		lead = "These state examples use stable atom IDs and explicit derived or snapshot APIs so shared values live outside one component and can be observed from multiple render paths."
		bullets = append([]string{
			"Choose stable atom IDs and avoid constructing them from unstable render-time values. Changing the ID effectively changes the storage location and subscription graph.",
			"Derived and computed state should stay cheap. If a derived function becomes expensive, memoize the inputs or reduce the amount of shared state fan-out instead of recomputing large graphs on every update.",
			"Snapshot export and restore are best for tooling, persistence, and controlled handoff points. Be careful with very large atom graphs because serialization and restore both have real runtime cost.",
		}, bullets...)
		if subject == "state.UseAtom" {
			code = []string{
				"count := state.UseAtom(\"catalog-counter\", 3)",
				"count.Update(func(previous int) int { return previous + 1 })",
				"value := count.Get()",
			}
		}
	case strings.HasPrefix(subject, "fetch."):
		lead = "These fetch examples expose the async lifecycle directly from Go hooks, so loaders, reloads, cancellation, cache state, and typed results stay visible in the component rather than hidden behind ad hoc goroutines."
		bullets = append([]string{
			"Use context-aware loaders and honor ctx.Done() promptly. Cancellation is part of the mechanism, not an optional extra, especially when requests can be superseded by newer input.",
			"Keep request triggers, active status, and response payloads close together so the reason for each screen state stays obvious during rerender and retry flows.",
			"Watch for unnecessary reload churn from unstable dependency values. If the dependency changes every render, the fetch hook will appear to thrash even when the UI looks simple.",
		}, bullets...)
		if subject == "fetch.UseResource" {
			code = []string{
				"resource := fetch.UseResource(func(ctx context.Context) (deploymentPreview, error) { ... }, environment.Get())",
				"state := resource.Get()",
				"resource.Reload()",
			}
		}
	case strings.HasPrefix(subject, "html."):
		lead = "These examples are implemented with typed html helpers that emit explicit element trees, so the Go call structure stays close to the final DOM structure the browser receives."
		bullets = append([]string{
			"Keep the markup tree readable enough that a developer can map helper calls back to the resulting semantic HTML quickly during inspection and debugging.",
			"Prefer dedicated semantic helpers over html.Tag when the element has a first-class wrapper. Generic tags are useful escape hatches, not the default.",
			"Deeply nested helper trees can become expensive to reason about long before they become expensive to render. Optimize first for maintainable structure and only then for node count.",
		}, bullets...)
	case strings.HasPrefix(subject, "devtools."):
		lead = "These devtools examples pair a small interactive surface with runtime inspection APIs so the instrumentation cost and the observed state stay visible together."
		bullets = append([]string{
			"Keep the demo state intentionally small so the inspection output is easy to correlate with the actual component tree and hook values.",
			"Panel refresh cadence matters: shorter intervals give fresher diagnostics but increase snapshot work and browser churn. Tune polling to the debugging task instead of always sampling aggressively.",
			"Watch out for noisy demo logic that makes it harder to see what the devtools API is actually adding beyond normal UI state changes.",
		}, bullets...)
	default:
		bullets = append([]string{
			"Each catalog example is a focused Go js/wasm package that mounts a single component tree into #app with ui.Render(ui.CreateElement(...), \"#app\").",
			"Component functions rerun on state changes, so hook order and dependency stability matter. Keep render logic deterministic and move imperative work into the appropriate hook or event path.",
			"Watch out for adding extra framework concerns before the core teaching point is clear; the best examples stay narrow until the primitive's job is obvious.",
		}, bullets...)
	}

	return lead, bullets, code
}

func exampleSubject(title, feature string) string {
	for _, candidate := range []string{title, feature} {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, ".") {
			return trimmed
		}
		if strings.Contains(strings.ToLower(trimmed), "router") || strings.Contains(strings.ToLower(trimmed), "hydrate") {
			return trimmed
		}
	}
	return strings.TrimSpace(title)
}

func ExampleBulletList(items ...string) ui.Node {
	children := make([]ui.Node, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		children = append(children,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text(trimmed)),
		)
	}
	return html.Ul(html.Props{Class: "mt-5 grid gap-3 text-sm leading-7 text-slate-300"}, children...)
}

func ExamplePanel(title string, body ...ui.Node) ui.Node {
	children := append([]ui.Node{
		html.H2(html.Props{Class: "text-xl font-bold text-white"}, html.Text(title)),
	}, body...)
	return html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-950/55 p-6 backdrop-blur-sm shadow-[0_18px_36px_rgba(2,6,23,0.32)]"}, children...)
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
