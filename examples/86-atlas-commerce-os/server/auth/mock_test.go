package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMockSessionManagerResolveAndCookies(t *testing.T) {
	manager := NewMockSessionManager()

	if session := manager.Resolve(httptest.NewRequest(http.MethodGet, "http://example.com/app/dashboard", nil)); session != nil {
		t.Fatalf("Resolve(no cookie) = %+v, want nil", session)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/app/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: MockSessionCookieName, Value: " warehouse_supervisor "})
	session := manager.Resolve(req)
	if session == nil {
		t.Fatal("Resolve(valid cookie) = nil, want session")
	}
	if session.UserID != "demo-operator" || session.DisplayName != "Atlas Demo Operator" || session.Role != "warehouse_supervisor" || session.DefaultWarehouse != "illinois-hub" {
		t.Fatalf("Resolve(valid cookie) = %+v", session)
	}

	invalidReq := httptest.NewRequest(http.MethodGet, "http://example.com/app/dashboard", nil)
	invalidReq.AddCookie(&http.Cookie{Name: MockSessionCookieName, Value: "admin"})
	if session := manager.Resolve(invalidReq); session != nil {
		t.Fatalf("Resolve(invalid cookie) = %+v, want nil", session)
	}

	startCookie := manager.StartCookie(" OPS_LEAD ")
	if startCookie.Name != MockSessionCookieName || startCookie.Value != "ops_lead" || startCookie.Path != "/" || !startCookie.HttpOnly || startCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("StartCookie() = %+v", startCookie)
	}
	blankCookie := manager.StartCookie("unknown")
	if blankCookie.Value != "" {
		t.Fatalf("StartCookie(unknown) value = %q, want empty", blankCookie.Value)
	}

	clearCookie := manager.ClearCookie()
	if clearCookie.Name != MockSessionCookieName || clearCookie.MaxAge != -1 || clearCookie.Value != "" || clearCookie.Path != "/" || !clearCookie.HttpOnly || clearCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("ClearCookie() = %+v", clearCookie)
	}
}

func TestMockSessionManagerRequireInternalSession(t *testing.T) {
	manager := NewMockSessionManager()

	t.Run("redirect for browser route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/app/orders?tab=late", nil)
		req.Header.Set("Referer", "http://example.com/app/inventory?filter=critical")
		res := httptest.NewRecorder()

		session := manager.RequireInternalSession(res, req)
		if session != nil {
			t.Fatalf("RequireInternalSession(browser) = %+v, want nil", session)
		}
		if res.Code != http.StatusSeeOther {
			t.Fatalf("RequireInternalSession(browser) status = %d, want 303", res.Code)
		}
		location := res.Header().Get("Location")
		if !strings.Contains(location, MockSignInPath) || !strings.Contains(location, "next=%2Fapp%2Finventory%3Ffilter%3Dcritical") {
			t.Fatalf("RequireInternalSession(browser) location = %q", location)
		}
	})

	t.Run("json for api route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/orders?tab=late", nil)
		res := httptest.NewRecorder()

		session := manager.RequireInternalSession(res, req)
		if session != nil {
			t.Fatalf("RequireInternalSession(api) = %+v, want nil", session)
		}
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("RequireInternalSession(api) status = %d, want 401", res.Code)
		}
		if contentType := res.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
			t.Fatalf("RequireInternalSession(api) content type = %q", contentType)
		}
		var payload map[string]any
		if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal(api response): %v", err)
		}
		if payload["error"] != "mock_sign_in_required" || !strings.Contains(payload["recovery"].(string), MockSignInPath) {
			t.Fatalf("RequireInternalSession(api) payload = %+v", payload)
		}
	})

	t.Run("returns existing session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/app/dashboard", nil)
		req.AddCookie(&http.Cookie{Name: MockSessionCookieName, Value: "inventory_manager"})
		res := httptest.NewRecorder()

		session := manager.RequireInternalSession(res, req)
		if session == nil || session.Role != "inventory_manager" || session.DefaultWarehouse != "new-jersey-hub" {
			t.Fatalf("RequireInternalSession(existing) = %+v", session)
		}
		if res.Code != http.StatusOK {
			t.Fatalf("RequireInternalSession(existing) recorder status = %d, want default 200", res.Code)
		}
	})
}

func TestMockAuthHelperFunctions(t *testing.T) {
	roles := AllowedRoles()
	if strings.Join(roles, ",") != "inventory_manager,warehouse_supervisor,ops_lead" {
		t.Fatalf("AllowedRoles() = %#v", roles)
	}

	if got := normalizeRole(" INVENTORY_MANAGER "); got != "inventory_manager" {
		t.Fatalf("normalizeRole(valid) = %q", got)
	}
	if got := normalizeRole("admin"); got != "" {
		t.Fatalf("normalizeRole(invalid) = %q, want empty", got)
	}

	reqWithReferer := httptest.NewRequest(http.MethodGet, "http://example.com/auth/mock-sign-in", nil)
	reqWithReferer.Header.Set("Referer", "http://example.com/app/ops?view=all")
	if got := requestNextPath(reqWithReferer); got != "/app/ops?view=all" {
		t.Fatalf("requestNextPath(referer) = %q", got)
	}

	reqAtSignIn := httptest.NewRequest(http.MethodGet, "http://example.com/auth/mock-sign-in", nil)
	if got := requestNextPath(reqAtSignIn); got != "/app/dashboard" {
		t.Fatalf("requestNextPath(sign-in) = %q, want /app/dashboard", got)
	}

	reqAtRoute := httptest.NewRequest(http.MethodGet, "http://example.com/app/queue?status=late", nil)
	if got := requestNextPath(reqAtRoute); got != "/app/queue?status=late" {
		t.Fatalf("requestNextPath(route) = %q", got)
	}

	if got := mockSignInURL("http://evil.example"); got != MockSignInPath+"?next=%2Fapp%2Fdashboard" {
		t.Fatalf("mockSignInURL(external) = %q", got)
	}
	if got := mockSignInURL("/app/inventory?filter=late"); got != MockSignInPath+"?next=%2Fapp%2Finventory%3Ffilter%3Dlate" {
		t.Fatalf("mockSignInURL(valid) = %q", got)
	}

	tests := map[string]string{
		"":                        "/app/dashboard",
		"   ":                     "/app/dashboard",
		"dashboard":               "/app/dashboard",
		"//evil.example/path":     "/app/dashboard",
		"/app/inventory":          "/app/inventory",
		"/app/orders?status=late": "/app/orders?status=late",
	}
	for input, want := range tests {
		if got := sanitizeNextPath(input); got != want {
			t.Fatalf("sanitizeNextPath(%q) = %q, want %q", input, got, want)
		}
	}
}
