package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	serverauth "github.com/monstercameron/GoWebComponents/v5/examples/server/atlas-commerce-os/server/auth"
)

// TestThemeAndLocalePersistenceIntegration verifies first paint defaults, preference save, and SSR-stable preference resume across reloads.
func TestThemeAndLocalePersistenceIntegration(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseInitialReq := httptest.NewRequest(http.MethodGet, "/app/settings", nil)
	parseInitialReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseInitialRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseInitialRes, parseInitialReq)
	if parseInitialRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseInitialRes.Code)
	}
	parseInitialBody := parseInitialRes.Body.String()
	for _, parseExpected := range []string{
		`<html lang="en" class="atlas-theme-dark atlas-density-compact"`,
		`id="__ATLAS_BOOTSTRAP__"`,
		`"locale":"en"`,
		`"direction":"ltr"`,
		`"theme":{"mode":"dark"`,
	} {
		if !strings.Contains(parseInitialBody, parseExpected) {
			parseT.Fatalf("expected initial SSR response to contain %q, got %q", parseExpected, parseInitialBody)
		}
	}

	parseCSRFTok, parseCSRFCookie := loadCSRFFromPage(parseT, parseServer, "/app/settings")
	parseSaveForm := url.Values{
		"csrf_token":           {parseCSRFTok},
		"theme":                {"light"},
		"locale":               {"ar"},
		"density":              {"comfortable"},
		"default_warehouse_id": {"illinois-hub"},
	}
	parseSaveReq := httptest.NewRequest(http.MethodPost, "/api/app/preferences", strings.NewReader(parseSaveForm.Encode()))
	parseSaveReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseSaveReq.Header.Set("Referer", "http://example.com/app/settings")
	parseSaveReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseSaveReq.AddCookie(parseCSRFCookie)
	parseSaveRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseSaveRes, parseSaveReq)
	if parseSaveRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d body=%q", http.StatusSeeOther, parseSaveRes.Code, parseSaveRes.Body.String())
	}

	parsePreferencesReq := httptest.NewRequest(http.MethodGet, "/api/app/preferences", nil)
	parsePreferencesReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parsePreferencesRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parsePreferencesRes, parsePreferencesReq)
	if parsePreferencesRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parsePreferencesRes.Code)
	}
	var parsePreferencesPayload struct {
		Theme              string `json:"theme"`
		Locale             string `json:"locale"`
		Density            string `json:"density"`
		DefaultWarehouseID string `json:"defaultWarehouseId"`
	}
	if parseErr := json.Unmarshal(parsePreferencesRes.Body.Bytes(), &parsePreferencesPayload); parseErr != nil {
		parseT.Fatalf("decode preferences payload: %v", parseErr)
	}
	if parsePreferencesPayload.Theme != "light" || parsePreferencesPayload.Locale != "ar" || parsePreferencesPayload.Density != "comfortable" || parsePreferencesPayload.DefaultWarehouseID != "illinois-hub" {
		parseT.Fatalf("expected saved preferences to persist, got %+v", parsePreferencesPayload)
	}

	parseReloadReq := httptest.NewRequest(http.MethodGet, "/app/settings", nil)
	parseReloadReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseReloadRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseReloadRes, parseReloadReq)
	if parseReloadRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseReloadRes.Code)
	}
	parseReloadBody := parseReloadRes.Body.String()
	for _, parseExpected := range []string{
		`<html lang="ar" class="atlas-theme-light atlas-density-comfortable"`,
		`id="__ATLAS_BOOTSTRAP__"`,
		`"locale":"ar"`,
		`"direction":"rtl"`,
		`"defaultWarehouse":"illinois-hub"`,
		`"theme":{"mode":"light"`,
	} {
		if !strings.Contains(parseReloadBody, parseExpected) {
			parseT.Fatalf("expected reloaded SSR response to contain %q, got %q", parseExpected, parseReloadBody)
		}
	}

	parseReloadAgainReq := httptest.NewRequest(http.MethodGet, "/app/settings", nil)
	parseReloadAgainReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseReloadAgainRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseReloadAgainRes, parseReloadAgainReq)
	if parseReloadAgainRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseReloadAgainRes.Code)
	}
	parseReloadAgainBody := parseReloadAgainRes.Body.String()
	for _, parseExpected := range []string{
		`<html lang="ar" class="atlas-theme-light atlas-density-comfortable"`,
		`"locale":"ar"`,
		`"direction":"rtl"`,
		`"theme":{"mode":"light"`,
	} {
		if !strings.Contains(parseReloadAgainBody, parseExpected) {
			parseT.Fatalf("expected stable reload SSR response to contain %q, got %q", parseExpected, parseReloadAgainBody)
		}
	}
}
