package shared

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func ExamplePage(parseTitle, parseFeature, parseSummary string, parseContent ...ui.Node) ui.Node {
	parseDocumentation := ExampleDocumentation(parseTitle, parseFeature, parseSummary)
	parseContent = append(parseContent, parseDocumentation...)

	parseChildren := []ui.Node{
		html.Div(html.Props{Class: "mx-auto max-w-6xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2.25rem] border border-cyan-300/15 bg-[linear-gradient(135deg,rgba(15,23,42,0.96),rgba(15,23,42,0.76))] p-8 shadow-[0_24px_60px_rgba(2,6,23,0.45)] backdrop-blur-sm"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text(parseFeature)),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text(parseTitle)),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(parseSummary)),
			),
			html.Div(html.Props{Class: "mt-8 grid gap-6"}, parseContent...),
		),
	}

	return html.Div(html.Props{Class: "min-h-screen text-slate-100"}, parseChildren...)
}

func ExampleDocumentation(parseTitle, parseFeature, parseSummary string) []ui.Node {
	parseOverviewLead, parseOverviewBullets := overviewCopy(parseTitle, parseFeature)
	parseFunctionalBullets := functionalCopy(parseTitle, parseFeature)
	parseImplementationLead, parseImplementationBullets, parseCodeLines := implementationCopy(parseTitle, parseFeature)

	parseOverviewPanel := ExamplePanel("Overview",
		html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text(parseOverviewLead)),
		ExampleBulletList(parseOverviewBullets...),
	)

	parseFunctionalDetails := ExamplePanel("Functional",
		html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text(parseSummary)),
		ExampleBulletList(parseFunctionalBullets...),
	)

	parseImplementationChildren := []ui.Node{
		html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text(parseImplementationLead)),
		ExampleBulletList(parseImplementationBullets...),
	}
	if len(parseCodeLines) > 0 {
		parseImplementationChildren = append(parseImplementationChildren, ExampleCode(parseCodeLines...))
	}
	parseImplementationDetails := ExamplePanel("Implementation", parseImplementationChildren...)

	return []ui.Node{parseOverviewPanel, parseFunctionalDetails, parseImplementationDetails}
}

func overviewCopy(parseTitle, parseFeature string) (string, []string) {
	parseSubject := exampleSubject(parseTitle, parseFeature)
	switch {
	case parseSubject == "ui.UseState":
		return "Use ui.UseState when a single component owns interactive local state and should rerender itself when that state changes.", []string{
			"Reach for this when the value only matters inside one feature surface, such as counters, toggles, local form draft state, or small view modes.",
			"Keep shared or cross-route data out of ui.UseState and move that into atoms, loaders, or another shared mechanism when multiple parts of the app need the same source of truth.",
		}
	case parseSubject == "ui.UseEffect":
		return "Use ui.UseEffect when a component must synchronize with work outside the pure render path, such as timers, DOM APIs, subscriptions, or document metadata.", []string{
			"The effect body should connect the external system, and the cleanup should disconnect it so rerenders and unmounts do not leak work.",
			"This tool exists for side effects, not for computing display values that can be derived directly during render.",
		}
	case parseSubject == "ui.UseReducer":
		return "Use ui.UseReducer when local state has named transitions, branch-heavy updates, or rules that read more clearly as actions than as scattered set calls.", []string{
			"Reducers work best when the legal state changes are important enough to name, test, and reason about directly.",
			"If the state is a single value with trivial updates, ui.UseState usually stays simpler and easier to scan.",
		}
	case parseSubject == "ui.Render":
		return "Use ui.Render to mount a component tree into a real DOM container and start the browser-facing app surface.", []string{
			"This is the normal client entrypoint for focused examples and browser-only applications.",
			"It answers the question: what do I mount, and where in the document do I mount it?",
		}
	case parseSubject == "ui.RenderToString":
		return "Use ui.RenderToString when you need server-generated HTML before the wasm client starts, such as SSR previews, crawlers, or fast first paint flows.", []string{
			"It is about producing HTML output, not attaching interactivity by itself.",
			"Pair it with hydration when the server HTML should later resume into a live client tree.",
		}
	case parseSubject == "ui.Hydrate":
		return "Use ui.Hydrate when matching HTML already exists in the DOM and the client should attach behavior without replacing that markup wholesale.", []string{
			"It is the right tool for SSR and prerender flows where the first paint comes from HTML that shipped before wasm execution.",
			"Hydration is valuable when preserving existing DOM matters for startup speed, user perception, or route-specific server output.",
		}
	case parseSubject == "ui.RenderBootstrapScript, ui.ReadBootstrapScript":
		return "Use bootstrap script helpers when server-rendered HTML needs structured startup data embedded directly into the page and restored during hydration.", []string{
			"This keeps the initial client resume aligned with the exact server-rendered route or state payload.",
			"It is most relevant when the server already knows the first route data and you want to avoid an immediate duplicate client fetch.",
		}
	case parseSubject == "router.HydrateMount":
		return "Use router.HydrateMount when a route was hydrated first and the router should attach listeners without forcing an immediate replacement render of that initial route.", []string{
			"This exists to preserve the prerendered or server-rendered first route while still enabling normal client navigation afterward.",
			"It is specifically about router startup behavior in hydration flows, not general routing.",
		}
	case strings.HasPrefix(parseSubject, "ui."):
		return "Use " + parseSubject + " when the concern belongs inside the component tree itself: local state, rendering, effects, composition, or UI-specific event flow.", []string{
			"These examples are meant to answer where the API fits in a component's lifecycle and what kind of UI problem it solves.",
			"If the behavior is local to one view and primarily changes rendering, this family of tools is usually the right place to start.",
		}
	case strings.HasPrefix(parseSubject, "state."):
		return "Use " + parseSubject + " when state must outlive one component, be derived from shared values, or be snapshotted, restored, or persisted.", []string{
			"These examples focus on shared state coordination rather than local component-only interactions.",
			"The purpose of the tool is to make state relationships inspectable and reusable across multiple views.",
		}
	case strings.HasPrefix(parseSubject, "fetch."):
		return "Use " + parseSubject + " when the UI needs to load, cache, retry, or imperatively request async data and expose that lifecycle to the page.", []string{
			"These examples show what the user sees while work is pending, successful, retried, cancelled, or failed.",
			"The tool purpose is not just fetching data, but shaping how async work participates in rendering.",
		}
	case strings.HasPrefix(parseSubject, "router.") || strings.Contains(parseSubject, "router") || strings.Contains(strings.ToLower(parseFeature), "route"):
		return "Use " + parseSubject + " when the URL should drive view state, navigation, route data loading, or hydration-aware startup behavior.", []string{
			"These examples demonstrate why a route tool exists by tying a visible screen change to path, params, query, redirect, guard, or loader behavior.",
			"If the state must be shareable through navigation and browser history, this is usually the right family of tools.",
		}
	case strings.HasPrefix(parseSubject, "html."):
		return "Use " + parseSubject + " when you want direct, semantic DOM construction helpers without dropping into string-based templates.", []string{
			"These examples show how the helper affects markup shape, semantics, and typed props rather than app-wide state flow.",
			"The tool purpose is clearer when you inspect the resulting HTML structure alongside the rendered output.",
		}
	case strings.HasPrefix(parseSubject, "devtools."):
		return "Use " + parseSubject + " when you need runtime inspection, snapshots, profiling hints, or diagnostics while developing GoWebComponents applications.", []string{
			"These tools are for understanding app behavior, not for driving user-facing product features directly.",
			"The examples focus on what diagnostic surface the tool exposes and when that surface is valuable during debugging.",
		}
	case strings.HasPrefix(parseSubject, "pwa."):
		return "Use " + parseSubject + " when installability, service-worker lifecycle, Cache Storage, or offline diagnostics should stay explicit in application code instead of hiding behind runtime side effects.", []string{
			"These examples focus on reviewable ownership: manifest, service worker, cache warming, and diagnostics are surfaced as deliberate app choices.",
			"Choose the pwa package when you want first-class offline and installability helpers without giving up control of deployment policy.",
		}
	case strings.HasPrefix(parseSubject, "plugin."):
		return "Use " + parseSubject + " when an application or companion package needs explicit manifest-based extension points without reaching into framework internals.", []string{
			"This family is about controlled integration boundaries: capabilities, registration, cleanup, and hook contribution should stay visible in normal application code.",
			"Choose it when you want reusable extension structure, but still want the owning app to decide exactly which hooks are enabled.",
		}
	default:
		return "Use this example to understand the purpose of " + parseTitle + " in isolation before mixing it into a larger integrated app.", []string{
			"The page is intentionally scoped so the API or tool's job is visible without unrelated framework behavior hiding the core idea.",
			"Read the live stats and output changes as the explanation of what the tool is for, not just as decorative UI.",
		}
	}
}

func functionalCopy(parseTitle, parseFeature string) []string {
	parseSubject := exampleSubject(parseTitle, parseFeature)
	parseBullets := []string{
		"This page isolates " + parseTitle + " so you can see the user-facing behavior without unrelated framework concerns in the way.",
		"Use the interactive controls in the panels above, then watch the stats and rendered output change as the example exercises " + parseFeature + ".",
	}

	switch {
	case parseSubject == "ui.UseReducer":
		parseBullets = append(parseBullets,
			"Reach for reducers when multiple buttons, events, or async results all need to drive the same state machine in a predictable way.",
			"The payoff is that the UI reads in terms of actions and outcomes instead of manual state patching spread across handlers.",
		)
	case strings.HasPrefix(parseSubject, "ui."):
		parseBullets = append(parseBullets,
			"Use this family of tools when the behavior is local to one rendered surface and you need the UI tree itself to explain the state change.",
			"Prefer the smallest hook or primitive that matches the job so the example stays easy to reason about and reuse.",
		)
	case strings.HasPrefix(parseSubject, "state."):
		parseBullets = append(parseBullets,
			"Use these controls to see how shared values, computed values, or snapshots propagate across multiple readers without duplicating the write path.",
			"Choose the state package when the data relationship matters more than the local widget that happens to display it.",
		)
	case strings.HasPrefix(parseSubject, "fetch."):
		parseBullets = append(parseBullets,
			"Watch how loading, ready, retry, cancellation, or error states affect what the user sees before choosing a higher-level data pattern.",
			"Prefer the lowest fetch abstraction that still makes the lifecycle readable for your screen.",
		)
	case strings.HasPrefix(parseSubject, "router.") || strings.Contains(strings.ToLower(parseFeature), "route"):
		parseBullets = append(parseBullets,
			"Use the navigation controls and visible route stats together so the path, params, query, or loader output stays tied to the rendered screen.",
			"Choose router primitives when the state should survive refreshes, deep links, or browser history changes.",
		)
	case strings.HasPrefix(parseSubject, "html."):
		parseBullets = append(parseBullets,
			"Inspect both the rendered structure and the user-facing output so the DOM shape stays as important as the visual result.",
			"These helpers are best when semantic markup clarity matters more than abstract component state transitions.",
		)
	case strings.HasPrefix(parseSubject, "devtools."):
		parseBullets = append(parseBullets,
			"Use the example to compare visible app behavior with the diagnostic surface so you can tell what the tool adds during debugging.",
			"The value here is observability: choose these APIs when understanding runtime behavior is the primary goal.",
		)
	case strings.HasPrefix(parseSubject, "pwa."):
		parseBullets = append(parseBullets,
			"Use the controls to separate installability, service-worker registration, cache warming, and diagnostics inspection into explicit operations the app owns directly.",
			"Choose these helpers when offline support matters, but correctness still depends on visible cache policy and update behavior rather than framework magic.",
		)
	case strings.HasPrefix(parseSubject, "plugin."):
		parseBullets = append(parseBullets,
			"Use these controls to see how plugins contribute route policy, async-data decoration, SSR head output, and form rules through one explicit host.",
			"The point is not hidden discovery. The point is flexible but reviewable extension wiring that stays outside the framework's internal runtime state.",
		)
	default:
		parseBullets = append(parseBullets,
			"The intent is to show both the functional result and the small amount of state, routing, async work, or DOM wiring needed to make that behavior happen.",
		)
	}

	return parseBullets
}

func implementationCopy(parseTitle, parseFeature string) (string, []string, []string) {
	parseSubject := exampleSubject(parseTitle, parseFeature)
	parseLead := "This example keeps the implementation small enough that the rendered behavior maps directly to the Go code driving the component tree, hook state, and event handlers."
	parseBullets := []string{
		"The compiled wasm output is generated under bin/examples/, and the examples dev server serves that build directory at /static/bin/ alongside the HTML entrypoint and shared static assets for local testing.",
	}
	parseCode := []string{
		"func main() {",
		"    ui.Render(ui.CreateElement(exampleComponent), \"#app\")",
		"    select {}",
		"}",
	}

	switch {
	case parseSubject == "ui.UseReducer":
		parseLead = "ui.UseReducer stores one reducer state cell in the component and routes every change through Dispatch, so the reducer function becomes the single transition table for that local state machine."
		parseBullets = append([]string{
			"Keep the reducer pure: it should derive the next state from the current state and action only, because it runs during render scheduling and must stay deterministic.",
			"Use reducers when several handlers need to converge on the same state rules. For a single scalar or a trivial toggle, ui.UseState usually allocates less code and less mental overhead.",
			"Watch for large state structs copied on every dispatch. If transitions become hot and the state payload grows, split state or move heavier data behind references instead of repeatedly cloning it.",
		}, parseBullets...)
		parseCode = []string{
			"type action string",
			"func reducer(state workflowState, action action) workflowState { ... }",
			"workflow := ui.UseReducer(reducer, workflowState{Step: \"Draft\"})",
			"workflow.Dispatch(actionReview)",
		}
	case parseSubject == "ui.Render":
		parseLead = "ui.Render is the client entrypoint: it creates the initial runtime root for a selector and mounts the component tree returned by ui.CreateElement into that DOM target."
		parseBullets = append([]string{
			"The selector must already exist in the document. If the mount target is missing, the runtime has nowhere to attach the tree.",
			"Render is a full client mount, not hydration. Use ui.Hydrate when the DOM already contains matching SSR or prerendered markup that should be resumed instead of replaced.",
			"Keep main small: initialize dependencies, mount once, and block with select {} so the Go wasm runtime stays alive for events and rerenders.",
		}, parseBullets...)
	case parseSubject == "ui.RenderToString":
		parseLead = "This example centers on server-side HTML generation, so the important implementation detail is when markup is produced versus when interactivity is attached later."
		parseBullets = append([]string{
			"Use render-to-string output for first paint, previews, or non-interactive delivery, then pair it with hydration only when the client must resume the same tree.",
			"Watch out for assuming RenderToString adds event handlers by itself; it only produces markup and serializes the current component output.",
			"Server-side rendering cost scales with the component tree you serialize. Keep request-time render work predictable and avoid burying expensive I/O directly inside render code.",
		}, parseBullets...)
		parseCode = []string{
			"markup, err := ui.RenderToString(ui.CreateElement(exampleComponent))",
			"if err != nil {",
			"    panic(err)",
			"}",
		}
	case parseSubject == "ui.Hydrate" || parseSubject == "router.HydrateMount" || parseSubject == "ui.RenderBootstrapScript, ui.ReadBootstrapScript":
		parseLead = "This example relies on existing HTML already being in the document, so the main implementation concern is resuming the correct DOM without replacing useful server output."
		parseBullets = append([]string{
			"Hydration works best when the client tree matches the server HTML and any bootstrap payload stays aligned with the initial route or state.",
			"Watch out for mismatched initial markup or stale bootstrap data, because those are the fastest ways to lose the benefit of hydration.",
			"The performance win comes from preserving the first paint and avoiding an immediate full rerender. That win disappears if the client has to throw away the prerendered subtree.",
		}, parseBullets...)
		parseCode = []string{
			"func main() {",
			"    _, _ = ui.Hydrate(ui.CreateElement(exampleComponent), \"#app\")",
			"    select {}",
			"}",
		}
	case strings.HasPrefix(parseSubject, "router.") || strings.Contains(strings.ToLower(parseFeature), "route"):
		parseLead = "These router examples are implemented as small route tables over the Go router, where Register wires path patterns to page factories and Mount or HydrateMount binds updates to the browser location."
		parseBullets = append([]string{
			"Hash routers derive route state from window.location.hash and avoid server rewrite requirements. Browser routers read pathname and need the server to return the app entrypoint for deep links.",
			"Route loaders should return compact, serializable attrs and respect context cancellation. If a user navigates away, the loader must stop work quickly instead of continuing background fetches.",
			"Avoid putting large opaque objects into route attrs or query-derived state. Loader output is read on rerender, so keep it small, typed, and easy to inspect.",
		}, parseBullets...)
		parseCode = []string{
			"func main() {",
			"    r := router.NewHashRouter(router.RouterOptions{DefaultRoute: \"/\"})",
			"    r.Register(\"/\", page)",
			"    r.Mount(\"#app\")",
			"    select {}",
			"}",
		}
	case strings.HasPrefix(parseSubject, "state."):
		parseLead = "These state examples use stable atom IDs and explicit derived or snapshot APIs so shared values live outside one component and can be observed from multiple render paths."
		parseBullets = append([]string{
			"Choose stable atom IDs and avoid constructing them from unstable render-time values. Changing the ID effectively changes the storage location and subscription graph.",
			"Derived and computed state should stay cheap. If a derived function becomes expensive, memoize the inputs or reduce the amount of shared state fan-out instead of recomputing large graphs on every update.",
			"Snapshot export and restore are best for tooling, persistence, and controlled handoff points. Be careful with very large atom graphs because serialization and restore both have real runtime cost.",
		}, parseBullets...)
		if parseSubject == "state.UseAtom" {
			parseCode = []string{
				"count := state.UseAtom(\"catalog-counter\", 3)",
				"count.Update(func(previous int) int { return previous + 1 })",
				"value := count.Get()",
			}
		}
	case strings.HasPrefix(parseSubject, "fetch."):
		parseLead = "These fetch examples expose the async lifecycle directly from Go hooks, so loaders, reloads, cancellation, cache state, and typed results stay visible in the component rather than hidden behind ad hoc goroutines."
		parseBullets = append([]string{
			"Use context-aware loaders and honor ctx.Done() promptly. Cancellation is part of the mechanism, not an optional extra, especially when requests can be superseded by newer input.",
			"Keep request triggers, active status, and response payloads close together so the reason for each screen state stays obvious during rerender and retry flows.",
			"Watch for unnecessary reload churn from unstable dependency values. If the dependency changes every render, the fetch hook will appear to thrash even when the UI looks simple.",
		}, parseBullets...)
		if parseSubject == "fetch.UseResource" {
			parseCode = []string{
				"resource := fetch.UseResource(func(ctx context.Context) (deploymentPreview, error) { ... }, environment.Get())",
				"state := resource.Get()",
				"resource.Reload()",
			}
		}
	case strings.HasPrefix(parseSubject, "html."):
		parseLead = "These examples are implemented with typed html helpers that emit explicit element trees, so the Go call structure stays close to the final DOM structure the browser receives."
		parseBullets = append([]string{
			"Keep the markup tree readable enough that a developer can map helper calls back to the resulting semantic HTML quickly during inspection and debugging.",
			"Prefer dedicated semantic helpers over html.Tag when the element has a first-class wrapper. Generic tags are useful escape hatches, not the default.",
			"Deeply nested helper trees can become expensive to reason about long before they become expensive to render. Optimize first for maintainable structure and only then for node count.",
		}, parseBullets...)
	case strings.HasPrefix(parseSubject, "devtools."):
		parseLead = "These devtools examples pair a small interactive surface with runtime inspection APIs so the instrumentation cost and the observed state stay visible together."
		parseBullets = append([]string{
			"Keep the demo state intentionally small so the inspection output is easy to correlate with the actual component tree and hook values.",
			"Panel refresh cadence matters: shorter intervals give fresher diagnostics but increase snapshot work and browser churn. Tune polling to the debugging task instead of always sampling aggressively.",
			"Watch out for noisy demo logic that makes it harder to see what the devtools API is actually adding beyond normal UI state changes.",
		}, parseBullets...)
	case strings.HasPrefix(parseSubject, "pwa."):
		parseLead = "These PWA examples keep installability, service-worker lifecycle, Cache Storage, and structured diagnostics under explicit app control so offline behavior remains reviewable rather than implicit."
		parseBullets = append([]string{
			"Prefer versioned cache plans and typed diagnostics over console-only debugging. The point is to inspect what was cached, queued, or registered in a form that can graduate into real support tooling.",
			"Keep the manifest, service-worker URL, and cache namespace obvious in code. PWA bugs are operational bugs as much as UI bugs, so hidden defaults are expensive to debug later.",
			"Treat offline writes and cached shells differently. Read caches are reconstructible, while queued writes and retained snapshots need explicit product policy and purge behavior.",
		}, parseBullets...)
		parseCode = []string{
			"manager, _ := pwa.OpenCacheStorageManager()",
			"registration, _ := pwa.RegisterServiceWorker(ctx, pwa.ServiceWorkerOptions{URL: \"/sw.js\", Scope: \"/app/\"})",
			"snapshot, _ := pwa.InspectDiagnostics(ctx, pwa.DiagnosticsOptions{...})",
		}
	case strings.HasPrefix(parseSubject, "plugin."):
		parseLead = "These plugin examples create one explicit host, register manifests in order, and let companion code contribute only the hooks the host enables for that application surface."
		parseBullets = append([]string{
			"Keep registration explicit and early in app setup so capability mismatches fail in one obvious place instead of during unrelated renders later.",
			"Prefer package-owned hooks and stable public APIs underneath the plugin host. The host is an integration organizer, not permission to reach into framework internals.",
			"Cleanup should stay scoped to the plugin's own subscriptions, observers, or external resources. Do not assume a hidden runtime shutdown channel exists unless the host documents one.",
		}, parseBullets...)
		parseCode = []string{
			"host := plugin.NewHost(plugin.HostOptions{Capabilities: []plugin.Capability{...}})",
			"host.Register(examplePlugin())",
			"decision := host.EvaluateRoute(plugin.RouteRequest{Path: \"/admin\"})",
			"issues := host.ValidateForm(plugin.FormSubmission{ID: \"purchase-order\"})",
		}
	default:
		parseBullets = append([]string{
			"Each catalog example is a focused Go js/wasm package that mounts a single component tree into #app with ui.Render(ui.CreateElement(...), \"#app\").",
			"Component functions rerun on state changes, so hook order and dependency stability matter. Keep render logic deterministic and move imperative work into the appropriate hook or event path.",
			"Watch out for adding extra framework concerns before the core teaching point is clear; the best examples stay narrow until the primitive's job is obvious.",
		}, parseBullets...)
	}

	return parseLead, parseBullets, parseCode
}

func exampleSubject(parseTitle, parseFeature string) string {
	for _, parseCandidate := range []string{parseTitle, parseFeature} {
		parseTrimmed := strings.TrimSpace(parseCandidate)
		if parseTrimmed == "" {
			continue
		}
		if strings.Contains(parseTrimmed, ".") {
			return parseTrimmed
		}
		if strings.Contains(strings.ToLower(parseTrimmed), "router") || strings.Contains(strings.ToLower(parseTrimmed), "hydrate") {
			return parseTrimmed
		}
	}
	return strings.TrimSpace(parseTitle)
}

func ExampleBulletList(parseItems ...string) ui.Node {
	parseChildren := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseTrimmed := strings.TrimSpace(parseItem)
		if parseTrimmed == "" {
			continue
		}
		parseChildren = append(parseChildren,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text(parseTrimmed)),
		)
	}
	return html.Ul(html.Props{Class: "mt-5 grid gap-3 text-sm leading-7 text-slate-300"}, parseChildren...)
}

func ExamplePanel(parseTitle string, parseBody ...ui.Node) ui.Node {
	parseChildren := append([]ui.Node{
		html.H2(html.Props{Class: "text-xl font-bold text-white"}, html.Text(parseTitle)),
	}, parseBody...)
	return html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-950/55 p-6 backdrop-blur-sm shadow-[0_18px_36px_rgba(2,6,23,0.32)]"}, parseChildren...)
}

func ExampleButton(parseLabel string, parseHandler ui.Handler) ui.Node {
	return html.Button(
		html.Props{
			OnClick: parseHandler,
			Class:   "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80",
		},
		html.Text(parseLabel),
	)
}

func ExampleStat(parseLabel, parseValue string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/60 p-4"},
		html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "mt-3 text-3xl font-black text-white"}, html.Text(parseValue)),
	)
}

func ExampleCode(parseLines ...string) ui.Node {
	parseChildren := make([]ui.Node, 0, len(parseLines))
	for _, parseLine := range parseLines {
		parseChildren = append(parseChildren, html.Text(parseLine+"\n"))
	}
	return html.Pre(html.Props{Class: "overflow-x-auto rounded-2xl border border-white/10 bg-black/40 p-4 text-sm text-slate-300"},
		html.Code(html.Props{}, parseChildren...),
	)
}
