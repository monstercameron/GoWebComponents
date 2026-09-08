//go:build !js || !wasm

package main

import (
	"fmt"
	"maps"
	"net/url"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

const (
	bootstrapEndpointPath  = "/_gwc/bootstrap"
	serverCanonicalBaseURL = "http://127.0.0.1:8079"
)

type resolvedRoute struct {
	Status       int
	Redirect     string
	Path         string
	Title        string
	Description  string
	CanonicalURL string
	Bootstrap    ui.SSRBootstrap
	View         demoShellView
}

func buildBootstrap(parsePath string, parseQuery url.Values, parseParams map[string]string, parseRouteData bootstrapRouteData) ui.SSRBootstrap {
	parseQueryMap := make(map[string][]string, len(parseQuery))
	for parseKey, parseValues := range parseQuery {
		parseQueryMap[parseKey] = append([]string(nil), parseValues...)
	}

	return ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{
			Path:   parsePath,
			Query:  parseQueryMap,
			Params: cloneParams(parseParams),
		},
		Data: map[string]any{
			"transport": serverBootstrapTransport,
			"routeData": parseRouteData,
		},
		IDSeed: parseRouteData.Revision,
	}
}

func cloneParams(parseParams map[string]string) map[string]string {
	if len(parseParams) == 0 {
		return map[string]string{}
	}
	parseClone := make(map[string]string, len(parseParams))
	maps.Copy(parseClone, parseParams)
	return parseClone
}

func resolveRoute(parsePath string, parseQuery url.Values) resolvedRoute {
	parseNormalizedPath := normalizePath(parsePath)
	switch {
	case parseNormalizedPath == "/":
		parseData := bootstrapRouteData{
			Page:     serverPageHome,
			Notice:   "This page is rendered by the Go HTTP server on every request. Use the links below to trigger new server-side renders over real URLs.",
			Revision: revisionFromQuery(parseQuery),
		}
		parseBootstrap := buildBootstrap(parseNormalizedPath, parseQuery, nil, parseData)
		return resolvedRoute{
			Status:       200,
			Path:         parseNormalizedPath,
			Title:        "GWC Server SSR Demo",
			Description:  "Request-time SSR shell rendered by a Go server and hydrated by the wasm client.",
			CanonicalURL: serverCanonicalBaseURL + "/",
			Bootstrap:    parseBootstrap,
			View:         viewFromRouteData(parseNormalizedPath, transportFromBootstrap(parseBootstrap), parseData),
		}
	case parseNormalizedPath == "/legacy":
		return resolvedRoute{Status: 302, Redirect: legacyRedirectPath}
	case strings.HasPrefix(parseNormalizedPath, "/docs/"):
		parseSection := strings.TrimPrefix(parseNormalizedPath, "/docs/")
		parseArticle := articleForSection(parseSection)
		parseCurrentTab := emptyFallback(strings.TrimSpace(parseQuery.Get("tab")), serverTabOverview)
		parseData2 := bootstrapRouteData{
			Page:         serverPageDocs,
			SectionID:    parseArticle.ID,
			SectionTitle: parseArticle.Title,
			SectionBody:  parseArticle.Summary,
			CurrentTab:   parseCurrentTab,
			StreamMode:   normalizeStreamMode(parseQuery.Get("stream")),
			Notice:       fmt.Sprintf("Server-rendered docs response for %s with tab=%s.", parseArticle.ID, parseCurrentTab),
			Revision:     revisionFromQuery(parseQuery),
		}
		parseParams := map[string]string{"section": parseArticle.ID}
		parseBootstrap2 := buildBootstrap(parseNormalizedPath, parseQuery, parseParams, parseData2)
		return resolvedRoute{
			Status:       200,
			Path:         parseNormalizedPath,
			Title:        parseArticle.Title + " | GWC Server SSR Demo",
			Description:  parseArticle.Summary,
			CanonicalURL: serverCanonicalBaseURL + buildPathWithQuery(parseNormalizedPath, parseQuery),
			Bootstrap:    parseBootstrap2,
			View:         viewFromRouteData(parseNormalizedPath, transportFromBootstrap(parseBootstrap2), parseData2),
		}
	case parseNormalizedPath == "/search":
		parseSearchQuery := strings.TrimSpace(parseQuery.Get("q"))
		parseData3 := bootstrapRouteData{
			Page:          serverPageSearch,
			SearchQuery:   parseSearchQuery,
			SearchResults: filterCatalog(parseSearchQuery),
			Notice:        "Every direct search navigation renders fresh HTML and a fresh bootstrap payload on the server.",
			Revision:      revisionFromQuery(parseQuery),
		}
		parseBootstrap3 := buildBootstrap(parseNormalizedPath, parseQuery, nil, parseData3)
		return resolvedRoute{
			Status:       200,
			Path:         parseNormalizedPath,
			Title:        "Search | GWC Server SSR Demo",
			Description:  "Server-rendered search results over real URLs.",
			CanonicalURL: serverCanonicalBaseURL + buildPathWithQuery(parseNormalizedPath, parseQuery),
			Bootstrap:    parseBootstrap3,
			View:         viewFromRouteData(parseNormalizedPath, transportFromBootstrap(parseBootstrap3), parseData3),
		}
	case parseNormalizedPath == "/secure":
		if parseQuery.Get("auth") != "true" {
			return resolvedRoute{Status: 302, Redirect: secureRedirectPath}
		}
		parseRole := emptyFallback(strings.TrimSpace(parseQuery.Get("role")), serverSecureRoleMaintainer)
		parseData4 := bootstrapRouteData{
			Page:       serverPageSecure,
			SecureRole: parseRole,
			SecureUser: serverSecureUserDefault,
			Notice:     "The secure route was allowed by the server and mirrored by the client browser router after hydration.",
			Revision:   revisionFromQuery(parseQuery),
		}
		parseBootstrap4 := buildBootstrap(parseNormalizedPath, parseQuery, nil, parseData4)
		return resolvedRoute{
			Status:       200,
			Path:         parseNormalizedPath,
			Title:        "Secure | GWC Server SSR Demo",
			Description:  "Protected route rendered on the server and hydrated on the client.",
			CanonicalURL: serverCanonicalBaseURL + buildPathWithQuery(parseNormalizedPath, parseQuery),
			Bootstrap:    parseBootstrap4,
			View:         viewFromRouteData(parseNormalizedPath, transportFromBootstrap(parseBootstrap4), parseData4),
		}
	case parseNormalizedPath == "/signin":
		parseFrom := strings.TrimSpace(parseQuery.Get("from"))
		parseNotice := "The protected route redirected here because auth=true was not present."
		if parseFrom != "" {
			parseNotice = "The protected route redirected here from " + parseFrom + "."
		}
		parseData5 := bootstrapRouteData{
			Page:     serverPageSignIn,
			Notice:   parseNotice,
			Revision: revisionFromQuery(parseQuery),
		}
		parseBootstrap5 := buildBootstrap(parseNormalizedPath, parseQuery, nil, parseData5)
		return resolvedRoute{
			Status:       200,
			Path:         parseNormalizedPath,
			Title:        "Sign In | GWC Server SSR Demo",
			Description:  "Guard redirect target for the protected server-rendered route.",
			CanonicalURL: serverCanonicalBaseURL + buildPathWithQuery(parseNormalizedPath, parseQuery),
			Bootstrap:    parseBootstrap5,
			View:         viewFromRouteData(parseNormalizedPath, transportFromBootstrap(parseBootstrap5), parseData5),
		}
	default:
		parseData6 := bootstrapRouteData{
			Page:     serverPageNotFound,
			Notice:   "This path is not registered by the server SSR demo.",
			Revision: 1,
		}
		parseBootstrap6 := buildBootstrap(parseNormalizedPath, parseQuery, nil, parseData6)
		return resolvedRoute{
			Status:       404,
			Path:         parseNormalizedPath,
			Title:        "Not Found | GWC Server SSR Demo",
			Description:  "Unknown route for the server-rendered SSR demo.",
			CanonicalURL: serverCanonicalBaseURL + buildPathWithQuery(parseNormalizedPath, parseQuery),
			Bootstrap:    parseBootstrap6,
			View:         viewFromRouteData(parseNormalizedPath, transportFromBootstrap(parseBootstrap6), parseData6),
		}
	}
}

func buildPathWithQuery(parsePath string, parseQuery url.Values) string {
	parseEncoded := parseQuery.Encode()
	if parseEncoded == "" {
		return parsePath
	}
	return parsePath + "?" + parseEncoded
}

func bootstrapReferenceURL(parsePath string, parseQuery url.Values) string {
	parseValues := url.Values{}
	parseValues.Set("path", parsePath)
	for parseKey, parseItems := range parseQuery {
		for _, parseItem := range parseItems {
			parseValues.Add(parseKey, parseItem)
		}
	}
	return bootstrapEndpointPath + "?" + parseValues.Encode()
}
