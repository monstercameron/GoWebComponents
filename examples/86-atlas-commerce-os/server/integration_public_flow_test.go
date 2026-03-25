package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPublicBrowsingFlowSSRBootstrapContinuity verifies a landing-to-catalog-to-product public flow keeps SSR and bootstrap continuity for hydration.
func TestPublicBrowsingFlowSSRBootstrapContinuity(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseCases := []struct {
		name               string
		path               string
		expectedRoutePath  string
		expectedRequestURL []string
	}{
		{
			name:               "landing",
			path:               "/",
			expectedRoutePath:  `"path":"/"`,
			expectedRequestURL: nil,
		},
		{
			name:              "catalog",
			path:              "/shop?q=frame&warehouse=new-jersey-hub&sort=warehouse",
			expectedRoutePath: `"path":"/shop"`,
			expectedRequestURL: []string{
				`"url":"/api/public/catalog`,
				`q=frame`,
				`sort=warehouse`,
				`warehouse=new-jersey-hub`,
			},
		},
		{
			name:               "product",
			path:               "/shop/frame-desk",
			expectedRoutePath:  `"path":"/shop/frame-desk"`,
			expectedRequestURL: []string{`"url":"/api/public/products/frame-desk"`},
		},
	}

	for _, parseTc := range parseCases {
		parseReq := httptest.NewRequest(http.MethodGet, parseTc.path, nil)
		parseRes := httptest.NewRecorder()
		parseServer.routes().ServeHTTP(parseRes, parseReq)
		if parseRes.Code != http.StatusOK {
			parseT.Fatalf("%s expected %d, got %d", parseTc.name, http.StatusOK, parseRes.Code)
		}

		parseBody := parseRes.Body.String()
		for _, parseExpected := range []string{
			`<div id="app"></div>`,
			atlasBootstrapScriptID,
			parseTc.expectedRoutePath,
		} {
			if parseExpected == "" {
				continue
			}
			if !strings.Contains(parseBody, parseExpected) {
				parseT.Fatalf("%s expected response to contain %q, got %q", parseTc.name, parseExpected, parseBody)
			}
		}
		for _, parseExpected := range parseTc.expectedRequestURL {
			if strings.TrimSpace(parseExpected) == "" {
				continue
			}
			if !strings.Contains(parseBody, parseExpected) {
				parseT.Fatalf("%s expected startup request fragment %q, got %q", parseTc.name, parseExpected, parseBody)
			}
		}
	}
}
