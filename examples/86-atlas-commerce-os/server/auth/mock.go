package auth

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

const MockSessionCookieName = "atlas_mock_role"

const MockSignInPath = "/auth/mock-sign-in"

type Session struct {
	UserID           string `json:"id"`
	DisplayName      string `json:"displayName"`
	Role             string `json:"role"`
	DefaultWarehouse string `json:"defaultWarehouse"`
}

type MockSessionManager struct{}

var mockRoleWarehouses = map[string]string{
	"inventory_manager":    "new-jersey-hub",
	"warehouse_supervisor": "illinois-hub",
	"ops_lead":             "nevada-hub",
}

func NewMockSessionManager() *MockSessionManager {
	return &MockSessionManager{}
}

func (parseM *MockSessionManager) Resolve(parseR *http.Request) *Session {
	parseCookie, parseErr := parseR.Cookie(MockSessionCookieName)
	if parseErr != nil {
		return nil
	}
	parseRole := normalizeRole(parseCookie.Value)
	if parseRole == "" {
		return nil
	}
	return &Session{
		UserID:           "demo-operator",
		DisplayName:      "Atlas Demo Operator",
		Role:             parseRole,
		DefaultWarehouse: mockRoleWarehouses[parseRole],
	}
}

func (parseM *MockSessionManager) RequireInternalSession(parseW http.ResponseWriter, parseR *http.Request) *Session {
	parseSession := parseM.Resolve(parseR)
	if parseSession == nil {
		parseRecoveryURL := mockSignInURL(requestNextPath(parseR))
		if !strings.HasPrefix(parseR.URL.Path, "/api/") {
			http.Redirect(parseW, parseR, parseRecoveryURL, http.StatusSeeOther)
			return nil
		}
		parseW.Header().Set("Content-Type", "application/json; charset=utf-8")
		parseW.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(parseW).Encode(map[string]any{
			"error":    "mock_sign_in_required",
			"message":  "Start a mock Atlas internal session to access this route.",
			"recovery": parseRecoveryURL,
		})
		return nil
	}
	return parseSession
}

func AllowedRoles() []string {
	return []string{"inventory_manager", "warehouse_supervisor", "ops_lead"}
}

func (parseM *MockSessionManager) StartCookie(parseRole string) *http.Cookie {
	return &http.Cookie{
		Name:     MockSessionCookieName,
		Value:    normalizeRole(parseRole),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func (parseM *MockSessionManager) ClearCookie() *http.Cookie {
	return &http.Cookie{
		Name:     MockSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
}

func normalizeRole(parseRole string) string {
	parseTrimmed := strings.TrimSpace(strings.ToLower(parseRole))
	if _, parseOk := mockRoleWarehouses[parseTrimmed]; parseOk {
		return parseTrimmed
	}
	return ""
}

func requestNextPath(parseR *http.Request) string {
	parseReferer := strings.TrimSpace(parseR.Header.Get("Referer"))
	if parseReferer != "" {
		if parseParsed, parseErr := url.Parse(parseReferer); parseErr == nil {
			if parseNext := sanitizeNextPath(parseParsed.RequestURI()); parseNext != "" {
				return parseNext
			}
		}
	}
	if parseNext2 := sanitizeNextPath(parseR.URL.RequestURI()); parseNext2 != "" && parseNext2 != MockSignInPath {
		return parseNext2
	}
	return "/app/dashboard"
}

func mockSignInURL(parseNext string) string {
	parseValues := url.Values{}
	parseValues.Set("next", sanitizeNextPath(parseNext))
	return MockSignInPath + "?" + parseValues.Encode()
}

func sanitizeNextPath(parseNext string) string {
	parseTrimmed := strings.TrimSpace(parseNext)
	if parseTrimmed == "" || !strings.HasPrefix(parseTrimmed, "/") || strings.HasPrefix(parseTrimmed, "//") {
		return "/app/dashboard"
	}
	if parseParsed, parseErr := url.Parse(parseTrimmed); parseErr == nil {
		parseCandidate := parseParsed.RequestURI()
		if strings.HasPrefix(parseCandidate, "/") && !strings.HasPrefix(parseCandidate, "//") {
			return parseCandidate
		}
	}
	return "/app/dashboard"
}
