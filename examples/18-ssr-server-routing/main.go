//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"strings"

	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

var initialBootstrap = ui.SSRBootstrap{}
var initialRouteData = bootstrapRouteData{}
var initialRouteDataAvailable bool
var initialRouteDataConsumed bool

func loadBootstrapPayload() ui.SSRBootstrap {
	bootstrapReference, err := ui.ReadBootstrapReferenceScript("")
	if err != nil {
		return ui.SSRBootstrap{}
	}
	bootstrapPayload, err := ui.ReadBootstrapReference(bootstrapReference)
	if err != nil {
		return ui.SSRBootstrap{}
	}
	return bootstrapPayload
}

func consumeInitialRouteData(routeCtx router.RouteContext, pageName string) (bootstrapRouteData, bool) {
	if !initialRouteDataAvailable || initialRouteDataConsumed {
		return bootstrapRouteData{}, false
	}
	if normalizePath(initialBootstrap.Route.Path) != normalizePath(routeCtx.Path) {
		return bootstrapRouteData{}, false
	}
	if initialRouteData.Page != pageName {
		return bootstrapRouteData{}, false
	}
	if pageName == serverPageDocs && initialBootstrap.Route.Params["section"] != routeCtx.Params.Get("section") {
		return bootstrapRouteData{}, false
	}
	if pageName == serverPageSearch && strings.TrimSpace(initialRouteData.SearchQuery) != strings.TrimSpace(routeCtx.Query.Get("q")) {
		return bootstrapRouteData{}, false
	}
	if pageName == serverPageSecure && strings.TrimSpace(initialRouteData.SecureRole) != emptyFallback(strings.TrimSpace(routeCtx.Query.Get("role")), serverSecureRoleMaintainer) {
		return bootstrapRouteData{}, false
	}
	initialRouteDataConsumed = true
	return initialRouteData, true
}

func homePage(props router.Attrs) ui.Node {
	data := bootstrapRouteData{
		Page:     serverPageHome,
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
	tab := emptyFallback(strings.TrimSpace(query.Get("tab")), serverTabOverview)
	data := bootstrapRouteData{
		Page:         serverPageDocs,
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
		Page:          serverPageSearch,
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
	data := bootstrapRouteData{Page: serverPageSignIn, Notice: notice, Revision: 1}
	return renderDemoShell(viewFromRouteData("/signin", transportFromBootstrap(initialBootstrap), data))
}

func securePage(props router.Attrs) ui.Node {
	role, _ := props["role"].(string)
	user, _ := props["user"].(string)
	revision, _ := props["revision"].(int)
	data := bootstrapRouteData{
		Page:       serverPageSecure,
		SecureRole: emptyFallback(role, serverSecureRoleMaintainer),
		SecureUser: emptyFallback(user, serverSecureUserDefault),
		Notice:     "The client resumed the protected route using the same role payload that the server rendered.",
		Revision:   maxInt(revision, 1),
	}
	return renderDemoShell(viewFromRouteData("/secure", transportFromBootstrap(initialBootstrap), data))
}

func notFoundPage(props router.Attrs) ui.Node {
	data := bootstrapRouteData{Page: serverPageNotFound, Notice: "Unknown route in the browser router.", Revision: 1}
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
			if data, ok := consumeInitialRouteData(routeCtx, serverPageDocs); ok {
				return router.Attrs{"revision": data.Revision}, nil
			}
			return router.Attrs{"revision": revisionFromQuery(routeCtx.Query.Values())}, nil
		},
	})
	r.Register("/search", searchPage, router.Options{
		Title: "GWC Server SSR Demo Search",
		Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
			if data, ok := consumeInitialRouteData(routeCtx, serverPageSearch); ok {
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
				return router.RedirectNavigation(secureRedirectPath)
			}
			return router.AllowNavigation()
		},
		Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
			if data, ok := consumeInitialRouteData(routeCtx, serverPageSecure); ok {
				return router.Attrs{"revision": data.Revision, "role": data.SecureRole, "user": data.SecureUser}, nil
			}
			role := emptyFallback(strings.TrimSpace(routeCtx.Query.Get("role")), serverSecureRoleMaintainer)
			return router.Attrs{"revision": revisionFromQuery(routeCtx.Query.Values()), "role": role, "user": serverSecureUserDefault}, nil
		},
	})
	r.Register("/legacy", docsPage, router.Options{Redirect: legacyRedirectPath, Title: "GWC Server SSR Demo Legacy"})
	r.Register("*", notFoundPage, router.Options{Title: "GWC Server SSR Demo Not Found"})

	root := ui.CreateElement(func() ui.Node { return r.Current() })
	_, _ = ui.Hydrate(root, "#app", ui.HydrationOptions{Bootstrap: initialBootstrap})
	r.HydrateMount("#app")

	select {}
}
