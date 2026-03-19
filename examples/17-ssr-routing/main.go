//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

var hydratedBootstrap = defaultBootstrapPayload()

const (
	modeHome   = "home"
	modeSearch = "search"
	modeSecure = "secure"
	modeSignIn = "signin"
)

var (
	docsRevision   int
	searchRevision int
	secureRevision int
)

func loadBootstrapPayload() ui.SSRBootstrap {
	bootstrapReference, err := ui.ReadBootstrapReferenceScript("")
	if err != nil {
		return defaultBootstrapPayload()
	}
	bootstrapPayload, err := ui.ReadBootstrapReference(bootstrapReference)
	if err != nil {
		return defaultBootstrapPayload()
	}
	return bootstrapPayload
}

func homePage(props router.Attrs) ui.Node {
	return renderDemoShell(demoShellView{
		Mode:          modeHome,
		ActivePath:    "/",
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		LoadRevision:  1,
		Notice:        "The docs route is server-rendered first, while the rest of the routes are client-driven after hydration.",
	})
}

func docsPage(props router.Attrs) ui.Node {
	params := router.UseParams()
	query := router.UseQuery()
	search := router.UseSearchParams()
	revalidator := router.UseRevalidator()

	article := articleForSection(params.Get("section"))
	if value, ok := props["section"].(string); ok && value != "" {
		article = articleForSection(value)
	}
	tab := query.Get("tab")
	if tab == "" {
		tab = tabOverview
	}
	revision, _ := props["revision"].(int)

	return renderDemoShell(demoShellView{
		Mode:          modeDocs,
		ActivePath:    "/docs/" + article.ID,
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		SectionID:     article.ID,
		SectionTitle:  article.Title,
		SectionBody:   article.Summary,
		CurrentTab:    tab,
		LoadRevision:  revision,
		Notice:        fmt.Sprintf("Route params and query state resolved this docs view. The current query string is %q.", search.Encode()),
	},
		uiButton("Show overview", func() { search.Replace("tab", tabOverview) }),
		uiButton("Show loader", func() { search.Replace("tab", tabLoader) }),
		uiButton("Revalidate", func() { revalidator.Revalidate() }),
	)
}

func searchPage(props router.Attrs) ui.Node {
	search := router.UseSearchParams()
	query := router.UseQuery()
	revalidator := router.UseRevalidator()

	results, _ := props["results"].([]guideArticle)
	revision, _ := props["revision"].(int)
	searchQuery := query.Get("q")

	return renderDemoShell(demoShellView{
		Mode:          modeSearch,
		ActivePath:    "/search",
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		SearchQuery:   searchQuery,
		SearchResults: results,
		LoadRevision:  revision,
		Notice:        "The search route uses query-keyed loader results and route-level revalidation.",
	},
		uiButton("Filter SSR", func() { search.Replace("q", searchQuerySSR) }),
		uiButton("Filter routing", func() { search.Replace("q", searchQueryRouting) }),
		uiButton("Revalidate", func() { revalidator.Revalidate() }),
	)
}

func filterCatalog(query string) []guideArticle {
	trimmed := strings.TrimSpace(strings.ToLower(query))
	if trimmed == "" {
		return catalogList()
	}

	matches := make([]guideArticle, 0, len(guideCatalog))
	for _, article := range catalogList() {
		if strings.Contains(strings.ToLower(article.Title), trimmed) || strings.Contains(strings.ToLower(article.Summary), trimmed) {
			matches = append(matches, article)
		}
	}
	return matches
}

func signInPage(props router.Attrs) ui.Node {
	nav := router.UseNavigate()
	from := router.UseQuery().Get("from")
	notice := "The secure route redirected here because the auth query flag was missing."
	if from != "" {
		notice = "The secure route redirected here from " + from + "."
	}

	return renderDemoShell(demoShellView{
		Mode:          modeSignIn,
		ActivePath:    "/signin",
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		LoadRevision:  1,
		Notice:        notice,
	},
		uiButton("Grant access", func() { nav.Replace("/secure?auth=true&role=" + secureRoleMaintainer) }),
	)
}

func securePage(props router.Attrs) ui.Node {
	revalidator := router.UseRevalidator()
	nav := router.UseNavigate()
	role, _ := props["role"].(string)
	user, _ := props["user"].(string)
	revision, _ := props["revision"].(int)

	return renderDemoShell(demoShellView{
		Mode:          modeSecure,
		ActivePath:    "/secure",
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		SecureRole:    role,
		SecureUser:    user,
		LoadRevision:  revision,
		Notice:        "A before-enter redirect protected the route and the loader returned role-specific content.",
	},
		uiButton("Switch role", func() { nav.Replace("/secure?auth=true&role=auditor") }),
		uiButton("Revalidate", func() { revalidator.Revalidate() }),
	)
}

func docsLoadingPage(props router.Attrs) ui.Node {
	section, _ := props["section"].(string)
	article := articleForSection(section)
	view := defaultServerView()
	view.SectionID = article.ID
	view.SectionTitle = "Loading " + article.Title
	view.SectionBody = "Route loader is resolving the current docs section before hydration settles."
	view.ActivePath = "/docs/" + article.ID
	return renderDemoShell(view)
}

func docsErrorPage(props router.Attrs) ui.Node {
	message, _ := props["error"].(string)
	view := defaultServerView()
	view.SectionTitle = "Docs route failed"
	view.SectionBody = message
	return renderDemoShell(view)
}

func searchLoadingPage(props router.Attrs) ui.Node {
	return renderDemoShell(demoShellView{
		Mode:          modeSearch,
		ActivePath:    "/search",
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		SectionTitle:  "Loading search",
		LoadRevision:  0,
		Notice:        "Route loader is resolving search results for the current query.",
	})
}

func searchErrorPage(props router.Attrs) ui.Node {
	message, _ := props["error"].(string)
	return renderDemoShell(demoShellView{
		Mode:          modeSearch,
		ActivePath:    "/search",
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		SearchQuery:   "",
		LoadRevision:  0,
		Notice:        message,
	})
}

func secureLoadingPage(props router.Attrs) ui.Node {
	return renderDemoShell(demoShellView{
		Mode:          modeSecure,
		ActivePath:    "/secure",
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		Notice:        "Checking guarded session state through the route loader.",
	})
}

func secureErrorPage(props router.Attrs) ui.Node {
	message, _ := props["error"].(string)
	return renderDemoShell(demoShellView{
		Mode:          modeSignIn,
		ActivePath:    "/signin",
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		Notice:        message,
	})
}

func uiButton(label string, onClick func()) ui.Node {
	return html.Button(html.Props{OnClick: ui.UseEvent(func() { onClick() }), Class: "inline-flex rounded-full border border-cyan-900/80 bg-cyan-950/70 px-4 py-2 text-sm font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text(label))
}

func routedApp(r *router.Router) ui.Node {
	hashTick := ui.UseState(0)
	_ = hashTick.Get()

	ui.UseEffect(func() func() {
		window := js.Global().Get("window")
		if !window.Truthy() {
			return nil
		}

		listener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			hashTick.Update(func(prev int) int { return prev + 1 })
			return nil
		})
		window.Call("addEventListener", "hashchange", listener)
		return func() {
			window.Call("removeEventListener", "hashchange", listener)
			listener.Release()
		}
	}, true)

	ui.UseEffect(func() func() {
		ticker := time.NewTicker(75 * time.Millisecond)
		stop := make(chan struct{})

		go func() {
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					hashTick.Update(func(prev int) int { return prev + 1 })
				}
			}
		}()

		return func() {
			close(stop)
			ticker.Stop()
		}
	}, true)

	return r.Current()
}

func main() {
	hydratedBootstrap = loadBootstrapPayload()

	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/docs/" + guideSectionSSR})
	r.Register("/", homePage, router.Options{Title: "SSR Routing Demo"})
	r.Register("/docs/:section", docsPage, router.Options{
		Title:        "SSR Routing Docs",
		Description:  "Server-rendered shell with route params, query-aware loaders, and manual revalidation.",
		CanonicalURL: "https://example.local/ssr-routing/docs",
		Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(40 * time.Millisecond):
			}

			article := articleForSection(routeCtx.Params.Get("section"))
			docsRevision++
			return router.Attrs{
				"section":  article.ID,
				"revision": docsRevision,
			}, nil
		},
		Loading: docsLoadingPage,
		Error:   docsErrorPage,
	})
	r.Register("/search", searchPage, router.Options{
		Title:        "SSR Routing Search",
		Description:  "Query-aware route loader results inside a hydrated routed shell.",
		CanonicalURL: "https://example.local/ssr-routing/search",
		Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(35 * time.Millisecond):
			}

			searchRevision++
			query := strings.TrimSpace(strings.ToLower(routeCtx.Query.Get("q")))
			return router.Attrs{
				"revision": searchRevision,
				"results":  filterCatalog(query),
			}, nil
		},
		Loading: searchLoadingPage,
		Error:   searchErrorPage,
	})
	r.Register("/signin", signInPage, router.Options{Title: "SSR Routing Sign In"})
	r.Register("/secure", securePage, router.Options{
		Title:        "SSR Routing Secure",
		Description:  "Guarded route rendered after a before-enter redirect and route loader approval.",
		CanonicalURL: "https://example.local/ssr-routing/secure",
		BeforeEnter: func(ctx router.RouteContext) router.GuardResult {
			if ctx.Query.Get("auth") != "true" {
				return router.RedirectNavigation("/signin?from=secure")
			}
			return router.AllowNavigation()
		},
		Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(30 * time.Millisecond):
			}

			secureRevision++
			role := routeCtx.Query.Get("role")
			if role == "" {
				role = secureRoleMaintainer
			}
			return router.Attrs{
				"revision": secureRevision,
				"role":     role,
				"user":     "Morgan Reconciler",
			}, nil
		},
		Loading: secureLoadingPage,
		Error:   secureErrorPage,
	})
	r.Register("/legacy", docsPage, router.Options{
		Redirect:     "/docs/routing?tab=loader",
		Title:        "SSR Routing Legacy",
		Description:  "Redirect route for the SSR routing demo.",
		CanonicalURL: "https://example.local/ssr-routing/legacy",
	})
	r.Register("*", homePage, router.Options{Title: "SSR Routing Demo"})

	root := ui.CreateElement(func() ui.Node { return routedApp(r) })
	if _, err := ui.Hydrate(root, "#app", ui.HydrationOptions{Bootstrap: hydratedBootstrap}); err != nil {
		ui.Render(root, "#app")
	}

	select {}
}
