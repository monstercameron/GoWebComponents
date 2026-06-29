//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

var initialBootstrap = ui.SSRBootstrap{}
var initialRouteData = bootstrapRouteData{}
var initialRouteDataAvailable bool
var initialRouteDataConsumed bool

func loadBootstrapPayload() ui.SSRBootstrap {
	parseBootstrapReference, parseErr := ui.ReadBootstrapReferenceScript("")
	if parseErr != nil {
		return ui.SSRBootstrap{}
	}
	parseBootstrapPayload, parseErr := ui.ReadBootstrapReference(parseBootstrapReference)
	if parseErr != nil {
		return ui.SSRBootstrap{}
	}
	return parseBootstrapPayload
}

func decodeBootstrapRouteData(parsePayload ui.SSRBootstrap) (bootstrapRouteData, bool) {
	if parsePayload.Data == nil {
		return bootstrapRouteData{}, false
	}
	parseRaw, parseOk := parsePayload.Data["routeData"]
	if !parseOk {
		return bootstrapRouteData{}, false
	}
	parseData, parseErr := json.Marshal(parseRaw)
	if parseErr != nil {
		return bootstrapRouteData{}, false
	}
	var parseDecoded bootstrapRouteData
	if parseErr2 := json.Unmarshal(parseData, &parseDecoded); parseErr2 != nil {
		return bootstrapRouteData{}, false
	}
	return parseDecoded, true
}

func consumeInitialRouteData(parseRouteCtx router.RouteContext, parsePageName string) (bootstrapRouteData, bool) {
	if !initialRouteDataAvailable || initialRouteDataConsumed {
		return bootstrapRouteData{}, false
	}
	if normalizePath(initialBootstrap.Route.Path) != normalizePath(parseRouteCtx.Path) {
		return bootstrapRouteData{}, false
	}
	if initialRouteData.Page != parsePageName {
		return bootstrapRouteData{}, false
	}
	if parsePageName == serverPageDocs && initialBootstrap.Route.Params["section"] != parseRouteCtx.Params.Get("section") {
		return bootstrapRouteData{}, false
	}
	if parsePageName == serverPageSearch && strings.TrimSpace(initialRouteData.SearchQuery) != strings.TrimSpace(parseRouteCtx.Query.Get("q")) {
		return bootstrapRouteData{}, false
	}
	if parsePageName == serverPageSecure && strings.TrimSpace(initialRouteData.SecureRole) != emptyFallback(strings.TrimSpace(parseRouteCtx.Query.Get("role")), serverSecureRoleMaintainer) {
		return bootstrapRouteData{}, false
	}
	initialRouteDataConsumed = true
	return initialRouteData, true
}

func homePage(parseProps router.Attrs) ui.Node {
	parseData := bootstrapRouteData{
		Page:     serverPageHome,
		Notice:   "The client is now running on a real browser route after the server generated the initial HTML.",
		Revision: 1,
	}
	return renderDemoShell(viewFromRouteData("/", transportFromBootstrap(initialBootstrap), parseData))
}

func docsPage(parseProps router.Attrs) ui.Node {
	parseParams := router.UseParams()
	parseQuery := router.UseQuery()
	parseSection := parseParams.Get("section")
	parseArticle := articleForSection(parseSection)
	parseRevision, _ := parseProps["revision"].(int)
	parseTab := emptyFallback(strings.TrimSpace(parseQuery.Get("tab")), serverTabOverview)
	parseStreamMode := normalizeStreamMode(parseQuery.Get("stream"))
	parseData := bootstrapRouteData{
		Page:         serverPageDocs,
		SectionID:    parseArticle.ID,
		SectionTitle: parseArticle.Title,
		SectionBody:  parseArticle.Summary,
		CurrentTab:   parseTab,
		StreamMode:   parseStreamMode,
		Notice:       fmt.Sprintf("Client router resumed on %s with tab=%s.", parseArticle.ID, parseTab),
		Revision:     maxInt(parseRevision, 1),
	}
	return renderDemoShell(viewFromRouteData("/docs/"+parseArticle.ID, transportFromBootstrap(initialBootstrap), parseData))
}

func searchPage(parseProps router.Attrs) ui.Node {
	parseQuery := router.UseQuery()
	parseResults, _ := parseProps["results"].([]guideArticle)
	parseRevision, _ := parseProps["revision"].(int)
	parseSearchQuery := strings.TrimSpace(parseQuery.Get("q"))
	parseData := bootstrapRouteData{
		Page:          serverPageSearch,
		SearchQuery:   parseSearchQuery,
		SearchResults: parseResults,
		Notice:        "Client browser routing mirrors the same search route while direct links still trigger full server SSR navigations.",
		Revision:      maxInt(parseRevision, 1),
	}
	return renderDemoShell(viewFromRouteData("/search", transportFromBootstrap(initialBootstrap), parseData))
}

func signInPage(parseProps router.Attrs) ui.Node {
	parseFrom := strings.TrimSpace(router.UseQuery().Get("from"))
	parseNotice := "The server redirected this protected request into the sign-in route."
	if parseFrom != "" {
		parseNotice = "The server redirected this protected request from " + parseFrom + "."
	}
	parseData := bootstrapRouteData{Page: serverPageSignIn, Notice: parseNotice, Revision: 1}
	return renderDemoShell(viewFromRouteData("/signin", transportFromBootstrap(initialBootstrap), parseData))
}

func securePage(parseProps router.Attrs) ui.Node {
	parseRole, _ := parseProps["role"].(string)
	parseUser, _ := parseProps["user"].(string)
	parseRevision, _ := parseProps["revision"].(int)
	parseData := bootstrapRouteData{
		Page:       serverPageSecure,
		SecureRole: emptyFallback(parseRole, serverSecureRoleMaintainer),
		SecureUser: emptyFallback(parseUser, serverSecureUserDefault),
		Notice:     "The client resumed the protected route using the same role payload that the server rendered.",
		Revision:   maxInt(parseRevision, 1),
	}
	return renderDemoShell(viewFromRouteData("/secure", transportFromBootstrap(initialBootstrap), parseData))
}

func notFoundPage(parseProps router.Attrs) ui.Node {
	parseData := bootstrapRouteData{Page: serverPageNotFound, Notice: "Unknown route in the browser router.", Revision: 1}
	return renderDemoShell(viewFromRouteData(router.GetCurrentPath(), transportFromBootstrap(initialBootstrap), parseData))
}

func main() {
	initialBootstrap = loadBootstrapPayload()
	parseDecoded, parseOk := decodeBootstrapRouteData(initialBootstrap)
	initialRouteData = parseDecoded
	initialRouteDataAvailable = parseOk

	parseR := router.NewHistoryRouter(router.RouterOptions{DefaultRoute: "/"})
	parseR.Register("/", homePage, router.Options{Title: "GWC Server SSR Demo"})
	parseR.Register("/docs/:section", docsPage, router.Options{
		Title: "GWC Server SSR Demo Docs",
		Loader: func(parseCtx context.Context, parseRouteCtx router.RouteContext) (router.Attrs, error) {
			if parseData, parseOk2 := consumeInitialRouteData(parseRouteCtx, serverPageDocs); parseOk2 {
				return router.Attrs{"revision": parseData.Revision}, nil
			}
			return router.Attrs{"revision": revisionFromQuery(parseRouteCtx.Query.Values())}, nil
		},
	})
	parseR.Register("/search", searchPage, router.Options{
		Title: "GWC Server SSR Demo Search",
		Loader: func(parseCtx2 context.Context, parseRouteCtx2 router.RouteContext) (router.Attrs, error) {
			if parseData2, parseOk3 := consumeInitialRouteData(parseRouteCtx2, serverPageSearch); parseOk3 {
				return router.Attrs{"revision": parseData2.Revision, "results": parseData2.SearchResults}, nil
			}
			parseSearchQuery := strings.TrimSpace(parseRouteCtx2.Query.Get("q"))
			return router.Attrs{"revision": revisionFromQuery(parseRouteCtx2.Query.Values()), "results": filterCatalog(parseSearchQuery)}, nil
		},
	})
	parseR.Register("/signin", signInPage, router.Options{Title: "GWC Server SSR Demo Sign In"})
	parseR.Register("/secure", securePage, router.Options{
		Title: "GWC Server SSR Demo Secure",
		BeforeEnter: func(parseCtx3 router.RouteContext) router.GuardResult {
			if parseCtx3.Query.Get("auth") != "true" {
				return router.RedirectNavigation(secureRedirectPath)
			}
			return router.AllowNavigation()
		},
		Loader: func(parseCtx4 context.Context, parseRouteCtx3 router.RouteContext) (router.Attrs, error) {
			if parseData3, parseOk4 := consumeInitialRouteData(parseRouteCtx3, serverPageSecure); parseOk4 {
				return router.Attrs{"revision": parseData3.Revision, "role": parseData3.SecureRole, "user": parseData3.SecureUser}, nil
			}
			parseRole := emptyFallback(strings.TrimSpace(parseRouteCtx3.Query.Get("role")), serverSecureRoleMaintainer)
			return router.Attrs{"revision": revisionFromQuery(parseRouteCtx3.Query.Values()), "role": parseRole, "user": serverSecureUserDefault}, nil
		},
	})
	parseR.Register("/legacy", docsPage, router.Options{Redirect: legacyRedirectPath, Title: "GWC Server SSR Demo Legacy"})
	parseR.Register("*", notFoundPage, router.Options{Title: "GWC Server SSR Demo Not Found"})

	parseRoot := ui.CreateElement(func() ui.Node { return parseR.Current() })
	_, _ = ui.Hydrate(parseRoot, "#app", ui.HydrationOptions{Bootstrap: initialBootstrap})
	parseR.HydrateMount("#app")

	select {}
}
