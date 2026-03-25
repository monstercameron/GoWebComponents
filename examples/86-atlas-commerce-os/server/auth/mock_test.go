package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMockSessionManagerResolveAndCookies(parseT *testing.T) {
	parseManager := NewMockSessionManager()

	if parseSession := parseManager.Resolve(httptest.NewRequest(http.MethodGet, "http://example.com/app/dashboard", nil)); parseSession != nil {
		parseT.Fatalf("Resolve(no cookie) = %+v, want nil", parseSession)
	}

	parseReq := httptest.NewRequest(http.MethodGet, "http://example.com/app/dashboard", nil)
	parseReq.AddCookie(&http.Cookie{Name: MockSessionCookieName, Value: " warehouse_supervisor "})
	parseSession2 := parseManager.Resolve(parseReq)
	if parseSession2 == nil {
		parseT.Fatal("Resolve(valid cookie) = nil, want session")
	}
	if parseSession2.UserID != "demo-operator" || parseSession2.DisplayName != "Atlas Demo Operator" || parseSession2.Role != "warehouse_supervisor" || parseSession2.DefaultWarehouse != "illinois-hub" {
		parseT.Fatalf("Resolve(valid cookie) = %+v", parseSession2)
	}

	parseInvalidReq := httptest.NewRequest(http.MethodGet, "http://example.com/app/dashboard", nil)
	parseInvalidReq.AddCookie(&http.Cookie{Name: MockSessionCookieName, Value: "admin"})
	if parseSession3 := parseManager.Resolve(parseInvalidReq); parseSession3 != nil {
		parseT.Fatalf("Resolve(invalid cookie) = %+v, want nil", parseSession3)
	}

	parseStartCookie := parseManager.StartCookie(" OPS_LEAD ")
	if parseStartCookie.Name != MockSessionCookieName || parseStartCookie.Value != "ops_lead" || parseStartCookie.Path != "/" || !parseStartCookie.HttpOnly || parseStartCookie.SameSite != http.SameSiteLaxMode {
		parseT.Fatalf("StartCookie() = %+v", parseStartCookie)
	}
	parseBlankCookie := parseManager.StartCookie("unknown")
	if parseBlankCookie.Value != "" {
		parseT.Fatalf("StartCookie(unknown) value = %q, want empty", parseBlankCookie.Value)
	}

	clearCookie := parseManager.ClearCookie()
	if clearCookie.Name != MockSessionCookieName || clearCookie.MaxAge != -1 || clearCookie.Value != "" || clearCookie.Path != "/" || !clearCookie.HttpOnly || clearCookie.SameSite != http.SameSiteLaxMode {
		parseT.Fatalf("ClearCookie() = %+v", clearCookie)
	}
}

func TestMockSessionManagerRequireInternalSession(parseT *testing.T) {
	parseManager := NewMockSessionManager()

	parseT.Run("redirect for browser route", func(parseT2 *testing.T) {
		parseReq := httptest.NewRequest(http.MethodGet, "http://example.com/app/orders?tab=late", nil)
		parseReq.Header.Set("Referer", "http://example.com/app/inventory?filter=critical")
		parseRes := httptest.NewRecorder()

		parseSession := parseManager.RequireInternalSession(parseRes, parseReq)
		if parseSession != nil {
			parseT2.Fatalf("RequireInternalSession(browser) = %+v, want nil", parseSession)
		}
		if parseRes.Code != http.StatusSeeOther {
			parseT2.Fatalf("RequireInternalSession(browser) status = %d, want 303", parseRes.Code)
		}
		parseLocation := parseRes.Header().Get("Location")
		if !strings.Contains(parseLocation, MockSignInPath) || !strings.Contains(parseLocation, "next=%2Fapp%2Finventory%3Ffilter%3Dcritical") {
			parseT2.Fatalf("RequireInternalSession(browser) location = %q", parseLocation)
		}
	})

	parseT.Run("json for api route", func(parseT3 *testing.T) {
		parseReq2 := httptest.NewRequest(http.MethodGet, "http://example.com/api/orders?tab=late", nil)
		parseRes2 := httptest.NewRecorder()

		parseSession2 := parseManager.RequireInternalSession(parseRes2, parseReq2)
		if parseSession2 != nil {
			parseT3.Fatalf("RequireInternalSession(api) = %+v, want nil", parseSession2)
		}
		if parseRes2.Code != http.StatusUnauthorized {
			parseT3.Fatalf("RequireInternalSession(api) status = %d, want 401", parseRes2.Code)
		}
		if parseContentType := parseRes2.Header().Get("Content-Type"); !strings.Contains(parseContentType, "application/json") {
			parseT3.Fatalf("RequireInternalSession(api) content type = %q", parseContentType)
		}
		var parsePayload map[string]any
		if parseErr := json.Unmarshal(parseRes2.Body.Bytes(), &parsePayload); parseErr != nil {
			parseT3.Fatalf("json.Unmarshal(api response): %v", parseErr)
		}
		if parsePayload["error"] != "mock_sign_in_required" || !strings.Contains(parsePayload["recovery"].(string), MockSignInPath) {
			parseT3.Fatalf("RequireInternalSession(api) payload = %+v", parsePayload)
		}
	})

	parseT.Run("returns existing session", func(parseT4 *testing.T) {
		parseReq3 := httptest.NewRequest(http.MethodGet, "http://example.com/app/dashboard", nil)
		parseReq3.AddCookie(&http.Cookie{Name: MockSessionCookieName, Value: "inventory_manager"})
		parseRes3 := httptest.NewRecorder()

		parseSession3 := parseManager.RequireInternalSession(parseRes3, parseReq3)
		if parseSession3 == nil || parseSession3.Role != "inventory_manager" || parseSession3.DefaultWarehouse != "new-jersey-hub" {
			parseT4.Fatalf("RequireInternalSession(existing) = %+v", parseSession3)
		}
		if parseRes3.Code != http.StatusOK {
			parseT4.Fatalf("RequireInternalSession(existing) recorder status = %d, want default 200", parseRes3.Code)
		}
	})
}

func TestMockAuthHelperFunctions(parseT *testing.T) {
	parseRoles := AllowedRoles()
	if strings.Join(parseRoles, ",") != "inventory_manager,warehouse_supervisor,ops_lead" {
		parseT.Fatalf("AllowedRoles() = %#v", parseRoles)
	}

	if parseGot := normalizeRole(" INVENTORY_MANAGER "); parseGot != "inventory_manager" {
		parseT.Fatalf("normalizeRole(valid) = %q", parseGot)
	}
	if parseGot2 := normalizeRole("admin"); parseGot2 != "" {
		parseT.Fatalf("normalizeRole(invalid) = %q, want empty", parseGot2)
	}

	parseReqWithReferer := httptest.NewRequest(http.MethodGet, "http://example.com/auth/mock-sign-in", nil)
	parseReqWithReferer.Header.Set("Referer", "http://example.com/app/ops?view=all")
	if parseGot3 := requestNextPath(parseReqWithReferer); parseGot3 != "/app/ops?view=all" {
		parseT.Fatalf("requestNextPath(referer) = %q", parseGot3)
	}

	parseReqAtSignIn := httptest.NewRequest(http.MethodGet, "http://example.com/auth/mock-sign-in", nil)
	if parseGot4 := requestNextPath(parseReqAtSignIn); parseGot4 != "/app/dashboard" {
		parseT.Fatalf("requestNextPath(sign-in) = %q, want /app/dashboard", parseGot4)
	}

	parseReqAtRoute := httptest.NewRequest(http.MethodGet, "http://example.com/app/queue?status=late", nil)
	if parseGot5 := requestNextPath(parseReqAtRoute); parseGot5 != "/app/queue?status=late" {
		parseT.Fatalf("requestNextPath(route) = %q", parseGot5)
	}

	if parseGot6 := mockSignInURL("http://evil.example"); parseGot6 != MockSignInPath+"?next=%2Fapp%2Fdashboard" {
		parseT.Fatalf("mockSignInURL(external) = %q", parseGot6)
	}
	if parseGot7 := mockSignInURL("/app/inventory?filter=late"); parseGot7 != MockSignInPath+"?next=%2Fapp%2Finventory%3Ffilter%3Dlate" {
		parseT.Fatalf("mockSignInURL(valid) = %q", parseGot7)
	}

	parseTests := map[string]string{
		"":                        "/app/dashboard",
		"   ":                     "/app/dashboard",
		"dashboard":               "/app/dashboard",
		"//evil.example/path":     "/app/dashboard",
		"/app/inventory":          "/app/inventory",
		"/app/orders?status=late": "/app/orders?status=late",
	}
	for parseInput, parseWant := range parseTests {
		if parseGot8 := sanitizeNextPath(parseInput); parseGot8 != parseWant {
			parseT.Fatalf("sanitizeNextPath(%q) = %q, want %q", parseInput, parseGot8, parseWant)
		}
	}
}
