//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	bootstrapEndpointPath = "/_gwc/bootstrap"
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

func buildBootstrap(path string, query url.Values, params map[string]string, routeData bootstrapRouteData) ui.SSRBootstrap {
	queryMap := make(map[string][]string, len(query))
	for key, values := range query {
		queryMap[key] = append([]string(nil), values...)
	}

	return ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{
			Path:   path,
			Query:  queryMap,
			Params: cloneParams(params),
		},
		Data: map[string]interface{}{
			"transport": serverBootstrapTransport,
			"routeData": routeData,
		},
		IDSeed: routeData.Revision,
	}
}

func cloneParams(params map[string]string) map[string]string {
	if len(params) == 0 {
		return map[string]string{}
	}
	clone := make(map[string]string, len(params))
	for key, value := range params {
		clone[key] = value
	}
	return clone
}

func resolveRoute(path string, query url.Values) resolvedRoute {
	normalizedPath := normalizePath(path)
	switch {
	case normalizedPath == "/":
		data := bootstrapRouteData{
			Page:     serverPageHome,
			Notice:   "This page is rendered by the Go HTTP server on every request. Use the links below to trigger new server-side renders over real URLs.",
			Revision: revisionFromQuery(query),
		}
		bootstrap := buildBootstrap(normalizedPath, query, nil, data)
		return resolvedRoute{
			Status:       200,
			Path:         normalizedPath,
			Title:        "GWC Server SSR Demo",
			Description:  "Request-time SSR shell rendered by a Go server and hydrated by the wasm client.",
			CanonicalURL: serverCanonicalBaseURL + "/",
			Bootstrap:    bootstrap,
			View:         viewFromRouteData(normalizedPath, transportFromBootstrap(bootstrap), data),
		}
	case normalizedPath == "/legacy":
		return resolvedRoute{Status: 302, Redirect: legacyRedirectPath}
	case strings.HasPrefix(normalizedPath, "/docs/"):
		section := strings.TrimPrefix(normalizedPath, "/docs/")
		article := articleForSection(section)
		currentTab := emptyFallback(strings.TrimSpace(query.Get("tab")), serverTabOverview)
		data := bootstrapRouteData{
			Page:         serverPageDocs,
			SectionID:    article.ID,
			SectionTitle: article.Title,
			SectionBody:  article.Summary,
			CurrentTab:   currentTab,
			Notice:       fmt.Sprintf("Server-rendered docs response for %s with tab=%s.", article.ID, currentTab),
			Revision:     revisionFromQuery(query),
		}
		params := map[string]string{"section": article.ID}
		bootstrap := buildBootstrap(normalizedPath, query, params, data)
		return resolvedRoute{
			Status:       200,
			Path:         normalizedPath,
			Title:        article.Title + " | GWC Server SSR Demo",
			Description:  article.Summary,
			CanonicalURL: serverCanonicalBaseURL + buildPathWithQuery(normalizedPath, query),
			Bootstrap:    bootstrap,
			View:         viewFromRouteData(normalizedPath, transportFromBootstrap(bootstrap), data),
		}
	case normalizedPath == "/search":
		searchQuery := strings.TrimSpace(query.Get("q"))
		data := bootstrapRouteData{
			Page:          serverPageSearch,
			SearchQuery:   searchQuery,
			SearchResults: filterCatalog(searchQuery),
			Notice:        "Every direct search navigation renders fresh HTML and a fresh bootstrap payload on the server.",
			Revision:      revisionFromQuery(query),
		}
		bootstrap := buildBootstrap(normalizedPath, query, nil, data)
		return resolvedRoute{
			Status:       200,
			Path:         normalizedPath,
			Title:        "Search | GWC Server SSR Demo",
			Description:  "Server-rendered search results over real URLs.",
			CanonicalURL: serverCanonicalBaseURL + buildPathWithQuery(normalizedPath, query),
			Bootstrap:    bootstrap,
			View:         viewFromRouteData(normalizedPath, transportFromBootstrap(bootstrap), data),
		}
	case normalizedPath == "/secure":
		if query.Get("auth") != "true" {
			return resolvedRoute{Status: 302, Redirect: secureRedirectPath}
		}
		role := emptyFallback(strings.TrimSpace(query.Get("role")), serverSecureRoleMaintainer)
		data := bootstrapRouteData{
			Page:       serverPageSecure,
			SecureRole: role,
			SecureUser: serverSecureUserDefault,
			Notice:     "The secure route was allowed by the server and mirrored by the client browser router after hydration.",
			Revision:   revisionFromQuery(query),
		}
		bootstrap := buildBootstrap(normalizedPath, query, nil, data)
		return resolvedRoute{
			Status:       200,
			Path:         normalizedPath,
			Title:        "Secure | GWC Server SSR Demo",
			Description:  "Protected route rendered on the server and hydrated on the client.",
			CanonicalURL: serverCanonicalBaseURL + buildPathWithQuery(normalizedPath, query),
			Bootstrap:    bootstrap,
			View:         viewFromRouteData(normalizedPath, transportFromBootstrap(bootstrap), data),
		}
	case normalizedPath == "/signin":
		from := strings.TrimSpace(query.Get("from"))
		notice := "The protected route redirected here because auth=true was not present."
		if from != "" {
			notice = "The protected route redirected here from " + from + "."
		}
		data := bootstrapRouteData{
			Page:     serverPageSignIn,
			Notice:   notice,
			Revision: revisionFromQuery(query),
		}
		bootstrap := buildBootstrap(normalizedPath, query, nil, data)
		return resolvedRoute{
			Status:       200,
			Path:         normalizedPath,
			Title:        "Sign In | GWC Server SSR Demo",
			Description:  "Guard redirect target for the protected server-rendered route.",
			CanonicalURL: serverCanonicalBaseURL + buildPathWithQuery(normalizedPath, query),
			Bootstrap:    bootstrap,
			View:         viewFromRouteData(normalizedPath, transportFromBootstrap(bootstrap), data),
		}
	default:
		data := bootstrapRouteData{
			Page:     serverPageNotFound,
			Notice:   "This path is not registered by the server SSR demo.",
			Revision: 1,
		}
		bootstrap := buildBootstrap(normalizedPath, query, nil, data)
		return resolvedRoute{
			Status:       404,
			Path:         normalizedPath,
			Title:        "Not Found | GWC Server SSR Demo",
			Description:  "Unknown route for the server-rendered SSR demo.",
			CanonicalURL: serverCanonicalBaseURL + buildPathWithQuery(normalizedPath, query),
			Bootstrap:    bootstrap,
			View:         viewFromRouteData(normalizedPath, transportFromBootstrap(bootstrap), data),
		}
	}
}

func buildPathWithQuery(path string, query url.Values) string {
	encoded := query.Encode()
	if encoded == "" {
		return path
	}
	return path + "?" + encoded
}

func bootstrapReferenceURL(path string, query url.Values) string {
	values := url.Values{}
	values.Set("path", path)
	for key, items := range query {
		for _, item := range items {
			values.Add(key, item)
		}
	}
	return bootstrapEndpointPath + "?" + values.Encode()
}
