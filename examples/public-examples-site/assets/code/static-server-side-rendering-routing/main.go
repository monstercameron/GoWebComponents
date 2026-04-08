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

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
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
	parseBootstrapReference, parseErr := ui.ReadBootstrapReferenceScript("")
	if parseErr != nil {
		return defaultBootstrapPayload()
	}
	parseBootstrapPayload, parseErr := ui.ReadBootstrapReference(parseBootstrapReference)
	if parseErr != nil {
		return defaultBootstrapPayload()
	}
	return parseBootstrapPayload
}

func homePage(parseProps router.Attrs) ui.Node {
	return renderDemoShell(demoShellView{
		Mode:          modeHome,
		ActivePath:    "/",
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		LoadRevision:  1,
		Notice:        "The docs route is server-rendered first, while the rest of the routes are client-driven after hydration.",
	})
}

func docsPage(parseProps router.Attrs) ui.Node {
	parseParams := router.UseParams()
	parseQuery := router.UseQuery()
	parseSearch := router.UseSearchParams()
	parseRevalidator := router.UseRevalidator()

	parseArticle := articleForSection(parseParams.Get("section"))
	if parseValue, parseOk := parseProps["section"].(string); parseOk && parseValue != "" {
		parseArticle = articleForSection(parseValue)
	}
	parseTab := parseQuery.Get("tab")
	if parseTab == "" {
		parseTab = tabOverview
	}
	parseRevision, _ := parseProps["revision"].(int)

	return renderDemoShell(demoShellView{
		Mode:          modeDocs,
		ActivePath:    "/docs/" + parseArticle.ID,
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		SectionID:     parseArticle.ID,
		SectionTitle:  parseArticle.Title,
		SectionBody:   parseArticle.Summary,
		CurrentTab:    parseTab,
		LoadRevision:  parseRevision,
		Notice:        fmt.Sprintf("Route params and query state resolved this docs view. The current query string is %q.", parseSearch.Encode()),
	},
		uiButton("Show overview", func() { parseSearch.Replace("tab", tabOverview) }),
		uiButton("Show loader", func() { parseSearch.Replace("tab", tabLoader) }),
		uiButton("Revalidate", func() { parseRevalidator.Revalidate() }),
	)
}

func searchPage(parseProps router.Attrs) ui.Node {
	parseSearch := router.UseSearchParams()
	parseQuery := router.UseQuery()
	parseRevalidator := router.UseRevalidator()

	parseResults, _ := parseProps["results"].([]guideArticle)
	parseRevision, _ := parseProps["revision"].(int)
	parseSearchQuery := parseQuery.Get("q")

	return renderDemoShell(demoShellView{
		Mode:          modeSearch,
		ActivePath:    "/search",
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		SearchQuery:   parseSearchQuery,
		SearchResults: parseResults,
		LoadRevision:  parseRevision,
		Notice:        "The search route uses query-keyed loader results and route-level revalidation.",
	},
		uiButton("Filter SSR", func() { parseSearch.Replace("q", searchQuerySSR) }),
		uiButton("Filter routing", func() { parseSearch.Replace("q", searchQueryRouting) }),
		uiButton("Revalidate", func() { parseRevalidator.Revalidate() }),
	)
}

func filterCatalog(parseQuery string) []guideArticle {
	parseTrimmed := strings.TrimSpace(strings.ToLower(parseQuery))
	if parseTrimmed == "" {
		return catalogList()
	}

	parseMatches := make([]guideArticle, 0, len(guideCatalog))
	for _, parseArticle := range catalogList() {
		if strings.Contains(strings.ToLower(parseArticle.Title), parseTrimmed) || strings.Contains(strings.ToLower(parseArticle.Summary), parseTrimmed) {
			parseMatches = append(parseMatches, parseArticle)
		}
	}
	return parseMatches
}

func signInPage(parseProps router.Attrs) ui.Node {
	parseNav := router.UseNavigate()
	parseFrom := router.UseQuery().Get("from")
	parseNotice := "The secure route redirected here because the auth query flag was missing."
	if parseFrom != "" {
		parseNotice = "The secure route redirected here from " + parseFrom + "."
	}

	return renderDemoShell(demoShellView{
		Mode:          modeSignIn,
		ActivePath:    "/signin",
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		LoadRevision:  1,
		Notice:        parseNotice,
	},
		uiButton("Grant access", func() { parseNav.Replace("/secure?auth=true&role=" + secureRoleMaintainer) }),
	)
}

func securePage(parseProps router.Attrs) ui.Node {
	parseRevalidator := router.UseRevalidator()
	parseNav := router.UseNavigate()
	parseRole, _ := parseProps["role"].(string)
	parseUser, _ := parseProps["user"].(string)
	parseRevision, _ := parseProps["revision"].(int)

	return renderDemoShell(demoShellView{
		Mode:          modeSecure,
		ActivePath:    "/secure",
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		SecureRole:    parseRole,
		SecureUser:    parseUser,
		LoadRevision:  parseRevision,
		Notice:        "A before-enter redirect protected the route and the loader returned role-specific content.",
	},
		uiButton("Switch role", func() { parseNav.Replace("/secure?auth=true&role=auditor") }),
		uiButton("Revalidate", func() { parseRevalidator.Revalidate() }),
	)
}

func docsLoadingPage(parseProps router.Attrs) ui.Node {
	parseSection, _ := parseProps["section"].(string)
	parseArticle := articleForSection(parseSection)
	parseView := defaultServerView()
	parseView.SectionID = parseArticle.ID
	parseView.SectionTitle = "Loading " + parseArticle.Title
	parseView.SectionBody = "Route loader is resolving the current docs section before hydration settles."
	parseView.ActivePath = "/docs/" + parseArticle.ID
	return renderDemoShell(parseView)
}

func docsErrorPage(parseProps router.Attrs) ui.Node {
	parseMessage, _ := parseProps["error"].(string)
	parseView := defaultServerView()
	parseView.SectionTitle = "Docs route failed"
	parseView.SectionBody = parseMessage
	return renderDemoShell(parseView)
}

func searchLoadingPage(parseProps router.Attrs) ui.Node {
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

func searchErrorPage(parseProps router.Attrs) ui.Node {
	parseMessage, _ := parseProps["error"].(string)
	return renderDemoShell(demoShellView{
		Mode:          modeSearch,
		ActivePath:    "/search",
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		SearchQuery:   "",
		LoadRevision:  0,
		Notice:        parseMessage,
	})
}

func secureLoadingPage(parseProps router.Attrs) ui.Node {
	return renderDemoShell(demoShellView{
		Mode:          modeSecure,
		ActivePath:    "/secure",
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		Notice:        "Checking guarded session state through the route loader.",
	})
}

func secureErrorPage(parseProps router.Attrs) ui.Node {
	parseMessage, _ := parseProps["error"].(string)
	return renderDemoShell(demoShellView{
		Mode:          modeSignIn,
		ActivePath:    "/signin",
		BootstrapPath: hydratedBootstrap.Route.Path,
		Transport:     bootstrapTransport(hydratedBootstrap),
		Notice:        parseMessage,
	})
}

func uiButton(parseLabel string, parseOnClick func()) ui.Node {
	return html.Button(html.Props{OnClick: ui.UseEvent(func() { parseOnClick() }), Class: "inline-flex rounded-full border border-cyan-900/80 bg-cyan-950/70 px-4 py-2 text-sm font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text(parseLabel))
}

func routedApp(parseR *router.Router) ui.Node {
	parseHashTick := ui.UseState(0)
	_ = parseHashTick.Get()

	ui.UseEffect(func() func() {
		parseWindow := js.Global().Get("window")
		if !parseWindow.Truthy() {
			return nil
		}

		parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseHashTick.Update(func(parsePrev int) int { return parsePrev + 1 })
			return nil
		})
		parseWindow.Call("addEventListener", "hashchange", parseListener)
		return func() {
			parseWindow.Call("removeEventListener", "hashchange", parseListener)
			parseListener.Release()
		}
	}, true)

	ui.UseEffect(func() func() {
		parseTicker := time.NewTicker(75 * time.Millisecond)
		parseStop := make(chan struct{})

		go func() {
			for {
				select {
				case <-parseStop:
					return
				case <-parseTicker.C:
					parseHashTick.Update(func(parsePrev2 int) int { return parsePrev2 + 1 })
				}
			}
		}()

		return func() {
			close(parseStop)
			parseTicker.Stop()
		}
	}, true)

	return parseR.Current()
}

func main() {
	hydratedBootstrap = loadBootstrapPayload()

	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/docs/" + guideSectionSSR})
	parseR.Register("/", homePage, router.Options{Title: "SSR Routing Demo"})
	parseR.Register("/docs/:section", docsPage, router.Options{
		Title:        "SSR Routing Docs",
		Description:  "Server-rendered shell with route params, query-aware loaders, and manual revalidation.",
		CanonicalURL: "https://example.local/ssr-routing/docs",
		Loader: func(parseCtx context.Context, parseRouteCtx router.RouteContext) (router.Attrs, error) {
			select {
			case <-parseCtx.Done():
				return nil, parseCtx.Err()
			case <-time.After(40 * time.Millisecond):
			}

			parseArticle := articleForSection(parseRouteCtx.Params.Get("section"))
			docsRevision++
			return router.Attrs{
				"section":  parseArticle.ID,
				"revision": docsRevision,
			}, nil
		},
		Loading: docsLoadingPage,
		Error:   docsErrorPage,
	})
	parseR.Register("/search", searchPage, router.Options{
		Title:        "SSR Routing Search",
		Description:  "Query-aware route loader results inside a hydrated routed shell.",
		CanonicalURL: "https://example.local/ssr-routing/search",
		Loader: func(parseCtx2 context.Context, parseRouteCtx2 router.RouteContext) (router.Attrs, error) {
			select {
			case <-parseCtx2.Done():
				return nil, parseCtx2.Err()
			case <-time.After(35 * time.Millisecond):
			}

			searchRevision++
			parseQuery := strings.TrimSpace(strings.ToLower(parseRouteCtx2.Query.Get("q")))
			return router.Attrs{
				"revision": searchRevision,
				"results":  filterCatalog(parseQuery),
			}, nil
		},
		Loading: searchLoadingPage,
		Error:   searchErrorPage,
	})
	parseR.Register("/signin", signInPage, router.Options{Title: "SSR Routing Sign In"})
	parseR.Register("/secure", securePage, router.Options{
		Title:        "SSR Routing Secure",
		Description:  "Guarded route rendered after a before-enter redirect and route loader approval.",
		CanonicalURL: "https://example.local/ssr-routing/secure",
		BeforeEnter: func(parseCtx3 router.RouteContext) router.GuardResult {
			if parseCtx3.Query.Get("auth") != "true" {
				return router.RedirectNavigation("/signin?from=secure")
			}
			return router.AllowNavigation()
		},
		Loader: func(parseCtx4 context.Context, parseRouteCtx3 router.RouteContext) (router.Attrs, error) {
			select {
			case <-parseCtx4.Done():
				return nil, parseCtx4.Err()
			case <-time.After(30 * time.Millisecond):
			}

			secureRevision++
			parseRole := parseRouteCtx3.Query.Get("role")
			if parseRole == "" {
				parseRole = secureRoleMaintainer
			}
			return router.Attrs{
				"revision": secureRevision,
				"role":     parseRole,
				"user":     "Morgan Reconciler",
			}, nil
		},
		Loading: secureLoadingPage,
		Error:   secureErrorPage,
	})
	parseR.Register("/legacy", docsPage, router.Options{
		Redirect:     "/docs/routing?tab=loader",
		Title:        "SSR Routing Legacy",
		Description:  "Redirect route for the SSR routing demo.",
		CanonicalURL: "https://example.local/ssr-routing/legacy",
	})
	parseR.Register("*", homePage, router.Options{Title: "SSR Routing Demo"})

	parseRoot := ui.CreateElement(func() ui.Node { return routedApp(parseR) })
	if _, parseErr := exampleboot.ApplyExampleHydration(parseRoot, ui.HydrationOptions{Bootstrap: hydratedBootstrap}); parseErr != nil {
		exampleboot.RenderExampleRoot(parseRoot)
	}
	exampleboot.WaitExampleRuntime()
}
