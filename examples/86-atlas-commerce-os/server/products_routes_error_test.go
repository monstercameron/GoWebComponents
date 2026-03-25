package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	serverauth "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/auth"
)

func TestInternalProductDetailAndEditorNotFoundPaths(t *testing.T) {
	t.Run("api detail not found", func(t *testing.T) {
		server, cleanup := newTestAtlasServer(t)
		defer cleanup()

		req := httptest.NewRequest(http.MethodGet, "/api/app/products/not-a-real-product", nil)
		req.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
		res := httptest.NewRecorder()

		server.routes().ServeHTTP(res, req)

		if res.Code != http.StatusNotFound {
			t.Fatalf("expected %d, got %d", http.StatusNotFound, res.Code)
		}
		if !strings.Contains(res.Body.String(), "product_admin_not_found") {
			t.Fatalf("expected product_admin_not_found payload, got %q", res.Body.String())
		}
	})

	t.Run("editor page not found", func(t *testing.T) {
		server, cleanup := newTestAtlasServer(t)
		defer cleanup()

		req := httptest.NewRequest(http.MethodGet, "/app/products/not-a-real-product", nil)
		req.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
		res := httptest.NewRecorder()

		server.routes().ServeHTTP(res, req)

		if res.Code != http.StatusNotFound {
			t.Fatalf("expected %d, got %d", http.StatusNotFound, res.Code)
		}
		body := res.Body.String()
		if !strings.Contains(body, "Atlas Internal Route Not Found") {
			t.Fatalf("expected internal route recovery content, got %q", body)
		}
		if !strings.Contains(body, "Back to dashboard") {
			t.Fatalf("expected dashboard recovery link, got %q", body)
		}
	})
}

func TestInternalProductsQueryFailurePaths(t *testing.T) {
	t.Run("api list query failure", func(t *testing.T) {
		server, cleanup := newTestAtlasServer(t)
		cleanup()

		req := httptest.NewRequest(http.MethodGet, "/api/app/products", nil)
		req.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
		res := httptest.NewRecorder()

		server.routes().ServeHTTP(res, req)

		if res.Code != http.StatusInternalServerError {
			t.Fatalf("expected %d, got %d", http.StatusInternalServerError, res.Code)
		}
		if !strings.Contains(res.Body.String(), "product_admin_query_failed") {
			t.Fatalf("expected product_admin_query_failed payload, got %q", res.Body.String())
		}
	})

	t.Run("page list query failure", func(t *testing.T) {
		server, cleanup := newTestAtlasServer(t)
		cleanup()

		req := httptest.NewRequest(http.MethodGet, "/app/products", nil)
		req.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
		res := httptest.NewRecorder()

		server.routes().ServeHTTP(res, req)

		if res.Code != http.StatusInternalServerError {
			t.Fatalf("expected %d, got %d", http.StatusInternalServerError, res.Code)
		}
		if !strings.Contains(res.Body.String(), "product_admin_query_failed") {
			t.Fatalf("expected product_admin_query_failed payload, got %q", res.Body.String())
		}
	})
}
