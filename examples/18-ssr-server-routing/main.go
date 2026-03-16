//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

var initialBootstrap = ui.SSRBootstrap{}
var initialRouteData = bootstrapRouteData{}
var initialRouteDataAvailable bool
var initialRouteDataConsumed bool

func loadBootstrapPayload() ui.SSRBootstrap {
	ref, err := ui.ReadBootstrapReferenceScript("")
	if err != nil {
		return ui.SSRBootstrap{}
	}
	payload, err := ui.ReadBootstrapReference(ref)
	if err != nil {
		return ui.SSRBootstrap{}
	}
	return payload
}

func consumeInitialRouteData(routeCtx router.RouteContext, page string) (bootstrapRouteData, bool) {
	if !initialRouteDataAvailable || initialRouteDataConsumed {
		return bootstrapRouteData{}, false
	}
	if normalizePath(initialBootstrap.Route.Path) != normalizePath(routeCtx.Path) {
		return bootstrapRouteData{}, false
	}
	if initialRouteData.Page != page {
		return bootstrapRouteData{}, false
	}
	if page == "docs" && initialBootstrap.Route.Params["section"] != routeCtx.Params.Get("section") {
		return bootstrapRouteData{}, false
	}
	if page == "search" && strings.TrimSpace(initialRouteData.SearchQuery) != strings.TrimSpace(routeCtx.Query.Get("q")) {
		return bootstrapRouteData{}, false
	}
	if page == "secure" && strings.TrimSpace(initialRouteData.SecureRole) != emptyFallback(strings.TrimSpace(routeCtx.Query.Get("role")), "maintainer") {
		return bootstrapRouteData{}, false
	}
	initialRouteDataConsumed = true
	return initialRouteData, true
}

func homePage(props router.Attrs) ui.Node {
	data := bootstrapRouteData{
		Page:     "home",
		Notice:   "The client is now running on a real browser route after the server generated the initial HTML.",
		Revision: 1,
	}
	return renderDemoShell(viewFromRouteData("/", transportFromBootstrap(initialBootstrap), data))
}

func docsPage(props router.Attrs) ui.Node {
	params := router.UseParams()
	query := router.UseQuery()
	section := params.Get("section")
	article := articleForSection(section)
	revision, _ := props["revision"].(int)
	tab := emptyFallback(strings.TrimSpace(query.Get("tab")), "overview")
	data := bootstrapRouteData{
		Page:         "docs",
		SectionID:    article.ID,
		SectionTitle: article.Title,
		SectionBody:  article.Summary,
		CurrentTab:   tab,
		Notice:       fmt.Sprintf("Client router resumed on %s with tab=%s.", article.ID, tab),
		Revision:     maxInt(revision, 1),
	}
	return renderDemoShell(viewFromRouteData("/docs/"+article.ID, transportFromBootstrap(initialBootstrap), data))
}

func searchPage(props router.Attrs) ui.Node {
	query := router.UseQuery()
	results, _ := props["results"].([]guideArticle)
	revision, _ := props["revision"].(int)
	searchQuery := strings.TrimSpace(query.Get("q"))
	data := bootstrapRouteData{
		Page:          "search",
		SearchQuery:   searchQuery,
		SearchResults: results,
		Notice:        "Client browser routing mirrors the same search route while direct links still trigger full server SSR navigations.",
		Revision:      maxInt(revision, 1),
	}
	return renderDemoShell(viewFromRouteData("/search", transportFromBootstrap(initialBootstrap), data))
}

func signInPage(props router.Attrs) ui.Node {
	from := strings.TrimSpace(router.UseQuery().Get("from"))
	notice := "The server redirected this protected request into the sign-in route."
	if from != "" {
		notice = "The server redirected this protected request from " + from + "."
	}
	data := bootstrapRouteData{Page: "signin", Notice: notice, Revision: 1}
	return renderDemoShell(viewFromRouteData("/signin", transportFromBootstrap(initialBootstrap), data))
}

func securePage(props router.Attrs) ui.Node {
	role, _ := props["role"].(string)
	user, _ := props["user"].(string)
	revision, _ := props["revision"].(int)
	data := bootstrapRouteData{
		Page:       "secure",
		SecureRole: emptyFallback(role, "maintainer"),
		SecureUser: emptyFallback(user, "Morgan Reconciler"),
		Notice:     "The client resumed the protected route using the same role payload that the server rendered.",
		Revision:   maxInt(revision, 1),
	}
	return renderDemoShell(viewFromRouteData("/secure", transportFromBootstrap(initialBootstrap), data))
}

func notFoundPage(props router.Attrs) ui.Node {
	data := bootstrapRouteData{Page: "not-found", Notice: "Unknown route in the browser router.", Revision: 1}
	return renderDemoShell(viewFromRouteData(router.GetCurrentPath(), transportFromBootstrap(initialBootstrap), data))
}

func main() {
	initialBootstrap = loadBootstrapPayload()
	decoded, ok := decodeBootstrapRouteData(initialBootstrap)
	initialRouteData = decoded
	initialRouteDataAvailable = ok

	r := router.NewRouter(router.RouterOptions{DefaultRoute: "/"})
	r.Register("/", homePage, router.Options{Title: "GWC Server SSR Demo"})
	r.Register("/docs/:section", docsPage, router.Options{
		Title: "GWC Server SSR Demo Docs",
		Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
			if data, ok := consumeInitialRouteData(routeCtx, "docs"); ok {
				return router.Attrs{"revision": data.Revision}, nil
			}
			return router.Attrs{"revision": revisionFromQuery(routeCtx.Query.Values())}, nil
		},
	})
	r.Register("/search", searchPage, router.Options{
		Title: "GWC Server SSR Demo Search",
		Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
			if data, ok := consumeInitialRouteData(routeCtx, "search"); ok {
				return router.Attrs{"revision": data.Revision, "results": data.SearchResults}, nil
			}
			searchQuery := strings.TrimSpace(routeCtx.Query.Get("q"))
			return router.Attrs{"revision": revisionFromQuery(routeCtx.Query.Values()), "results": filterCatalog(searchQuery)}, nil
		},
	})
	r.Register("/signin", signInPage, router.Options{Title: "GWC Server SSR Demo Sign In"})
	r.Register("/secure", securePage, router.Options{
		Title: "GWC Server SSR Demo Secure",
		BeforeEnter: func(ctx router.RouteContext) router.GuardResult {
			if ctx.Query.Get("auth") != "true" {
				return router.RedirectNavigation("/signin?from=secure")
			}
			return router.AllowNavigation()
		},
		Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
			if data, ok := consumeInitialRouteData(routeCtx, "secure"); ok {
				return router.Attrs{"revision": data.Revision, "role": data.SecureRole, "user": data.SecureUser}, nil
			}
			role := emptyFallback(strings.TrimSpace(routeCtx.Query.Get("role")), "maintainer")
			return router.Attrs{"revision": revisionFromQuery(routeCtx.Query.Values()), "role": role, "user": "Morgan Reconciler"}, nil
		},
	})
	r.Register("/legacy", docsPage, router.Options{Redirect: "/docs/routing?tab=loader", Title: "GWC Server SSR Demo Legacy"})
	r.Register("*", notFoundPage, router.Options{Title: "GWC Server SSR Demo Not Found"})

	root := ui.CreateElement(func() ui.Node { return r.GoGetRoute() })
	_, _ = ui.Hydrate(root, "#app", ui.HydrationOptions{Bootstrap: initialBootstrap})
	r.HydrateMount("#app")

	select {}
}
