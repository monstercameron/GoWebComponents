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

func (m *MockSessionManager) Resolve(r *http.Request) *Session {
	cookie, err := r.Cookie(MockSessionCookieName)
	if err != nil {
		return nil
	}
	role := normalizeRole(cookie.Value)
	if role == "" {
		return nil
	}
	return &Session{
		UserID:           "demo-operator",
		DisplayName:      "Atlas Demo Operator",
		Role:             role,
		DefaultWarehouse: mockRoleWarehouses[role],
	}
}

func (m *MockSessionManager) RequireInternalSession(w http.ResponseWriter, r *http.Request) *Session {
	session := m.Resolve(r)
	if session == nil {
		recoveryURL := mockSignInURL(requestNextPath(r))
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			http.Redirect(w, r, recoveryURL, http.StatusSeeOther)
			return nil
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":    "mock_sign_in_required",
			"message":  "Start a mock Atlas internal session to access this route.",
			"recovery": recoveryURL,
		})
		return nil
	}
	return session
}

func AllowedRoles() []string {
	return []string{"inventory_manager", "warehouse_supervisor", "ops_lead"}
}

func (m *MockSessionManager) StartCookie(role string) *http.Cookie {
	return &http.Cookie{
		Name:     MockSessionCookieName,
		Value:    normalizeRole(role),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func (m *MockSessionManager) ClearCookie() *http.Cookie {
	return &http.Cookie{
		Name:     MockSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
}

func normalizeRole(role string) string {
	trimmed := strings.TrimSpace(strings.ToLower(role))
	if _, ok := mockRoleWarehouses[trimmed]; ok {
		return trimmed
	}
	return ""
}

func requestNextPath(r *http.Request) string {
	referer := strings.TrimSpace(r.Header.Get("Referer"))
	if referer != "" {
		if parsed, err := url.Parse(referer); err == nil {
			if next := sanitizeNextPath(parsed.RequestURI()); next != "" {
				return next
			}
		}
	}
	if next := sanitizeNextPath(r.URL.RequestURI()); next != "" && next != MockSignInPath {
		return next
	}
	return "/app/dashboard"
}

func mockSignInURL(next string) string {
	values := url.Values{}
	values.Set("next", sanitizeNextPath(next))
	return MockSignInPath + "?" + values.Encode()
}

func sanitizeNextPath(next string) string {
	trimmed := strings.TrimSpace(next)
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "//") {
		return "/app/dashboard"
	}
	if parsed, err := url.Parse(trimmed); err == nil {
		candidate := parsed.RequestURI()
		if strings.HasPrefix(candidate, "/") && !strings.HasPrefix(candidate, "//") {
			return candidate
		}
	}
	return "/app/dashboard"
}
