package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	serverauth "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/auth"
)

func TestInternalProductDetailAndEditorNotFoundPaths(parseT *testing.T) {
	parseT.Run("api detail not found", func(parseT2 *testing.T) {
		parseServer, parseCleanup := newTestAtlasServer(parseT2)
		defer parseCleanup()

		parseReq := httptest.NewRequest(http.MethodGet, "/api/app/products/not-a-real-product", nil)
		parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
		parseRes := httptest.NewRecorder()

		parseServer.routes().ServeHTTP(parseRes, parseReq)

		if parseRes.Code != http.StatusNotFound {
			parseT2.Fatalf("expected %d, got %d", http.StatusNotFound, parseRes.Code)
		}
		if !strings.Contains(parseRes.Body.String(), "product_admin_not_found") {
			parseT2.Fatalf("expected product_admin_not_found payload, got %q", parseRes.Body.String())
		}
	})

	parseT.Run("editor page not found", func(parseT3 *testing.T) {
		parseServer2, parseCleanup2 := newTestAtlasServer(parseT3)
		defer parseCleanup2()

		parseReq2 := httptest.NewRequest(http.MethodGet, "/app/products/not-a-real-product", nil)
		parseReq2.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
		parseRes2 := httptest.NewRecorder()

		parseServer2.routes().ServeHTTP(parseRes2, parseReq2)

		if parseRes2.Code != http.StatusNotFound {
			parseT3.Fatalf("expected %d, got %d", http.StatusNotFound, parseRes2.Code)
		}
		parseBody := parseRes2.Body.String()
		if !strings.Contains(parseBody, "Atlas Internal Route Not Found") {
			parseT3.Fatalf("expected internal route recovery content, got %q", parseBody)
		}
		if !strings.Contains(parseBody, "Back to dashboard") {
			parseT3.Fatalf("expected dashboard recovery link, got %q", parseBody)
		}
	})
}

func TestInternalProductsQueryFailurePaths(parseT *testing.T) {
	parseT.Run("api list query failure", func(parseT2 *testing.T) {
		parseServer, parseCleanup := newTestAtlasServer(parseT2)
		parseCleanup()

		parseReq := httptest.NewRequest(http.MethodGet, "/api/app/products", nil)
		parseReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
		parseRes := httptest.NewRecorder()

		parseServer.routes().ServeHTTP(parseRes, parseReq)

		if parseRes.Code != http.StatusInternalServerError {
			parseT2.Fatalf("expected %d, got %d", http.StatusInternalServerError, parseRes.Code)
		}
		if !strings.Contains(parseRes.Body.String(), "product_admin_query_failed") {
			parseT2.Fatalf("expected product_admin_query_failed payload, got %q", parseRes.Body.String())
		}
	})

	parseT.Run("page list query failure", func(parseT3 *testing.T) {
		parseServer2, parseCleanup2 := newTestAtlasServer(parseT3)
		parseCleanup2()

		parseReq2 := httptest.NewRequest(http.MethodGet, "/app/products", nil)
		parseReq2.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
		parseRes2 := httptest.NewRecorder()

		parseServer2.routes().ServeHTTP(parseRes2, parseReq2)

		if parseRes2.Code != http.StatusInternalServerError {
			parseT3.Fatalf("expected %d, got %d", http.StatusInternalServerError, parseRes2.Code)
		}
		if !strings.Contains(parseRes2.Body.String(), "product_admin_query_failed") {
			parseT3.Fatalf("expected product_admin_query_failed payload, got %q", parseRes2.Body.String())
		}
	})
}
