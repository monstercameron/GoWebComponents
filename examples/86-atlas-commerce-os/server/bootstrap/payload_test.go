package bootstrap

import "testing"

func TestPayloadToSSRBootstrap(t *testing.T) {
	payload := Payload{
		Route: RouteBootstrap{
			Path:        "/inventory",
			Query:       map[string][]string{"warehouse": {"east"}},
			Params:      map[string]string{"sku": "desk-1"},
			Surface:     "console",
			Screen:      "inventory",
			Title:       "Inventory",
			Description: "Inventory workspace",
			Canonical:   "/inventory",
		},
		Preferences: PreferencesState{
			Theme:            "dark",
			Locale:           "en-US",
			Density:          "compact",
			DefaultWarehouse: "east",
		},
		I18n: I18nState{
			Locale:           "en-US",
			SupportedLocales: []string{"en-US", "fr-FR"},
			Direction:        "ltr",
		},
		Theme: ThemeState{
			Mode:                 "dark",
			PrefersReducedMotion: true,
		},
		Data: map[string]any{
			"cards": 4,
		},
		SavedViews: []SavedViewPayload{{
			Name:          "Low stock",
			Scope:         "inventory",
			SortKey:       "available",
			SortDirection: "asc",
			Filters:       map[string]string{"warehouse": "east"},
		}},
		User: &UserSession{
			ID:               "u1",
			DisplayName:      "Cam",
			Role:             "buyer",
			DefaultWarehouse: "east",
		},
		Workspace: map[string]any{
			"activeTab": "alerts",
		},
		CSRF: "token-123",
	}

	got := payload.ToSSRBootstrap()

	if got.Route.Path != payload.Route.Path {
		t.Fatalf("Route.Path mismatch: got %q want %q", got.Route.Path, payload.Route.Path)
	}
	if got.Route.Query["warehouse"][0] != "east" {
		t.Fatalf("Route.Query mismatch: got %#v", got.Route.Query)
	}
	if got.Route.Params["sku"] != "desk-1" {
		t.Fatalf("Route.Params mismatch: got %#v", got.Route.Params)
	}
	if got.I18n.Locale != payload.I18n.Locale || got.I18n.Direction != payload.I18n.Direction {
		t.Fatalf("I18n mismatch: got %#v want locale=%q direction=%q", got.I18n, payload.I18n.Locale, payload.I18n.Direction)
	}

	dataRoute, ok := got.Data["route"].(RouteBootstrap)
	if !ok || dataRoute.Title != payload.Route.Title || dataRoute.Surface != payload.Route.Surface {
		t.Fatalf("route payload mismatch: %#v", got.Data["route"])
	}
	dataPreferences, ok := got.Data["preferences"].(PreferencesState)
	if !ok || dataPreferences.DefaultWarehouse != payload.Preferences.DefaultWarehouse {
		t.Fatalf("preferences payload mismatch: %#v", got.Data["preferences"])
	}
	dataTheme, ok := got.Data["theme"].(ThemeState)
	if !ok || dataTheme.Mode != payload.Theme.Mode || !dataTheme.PrefersReducedMotion {
		t.Fatalf("theme payload mismatch: %#v", got.Data["theme"])
	}
	dataViews, ok := got.Data["savedViews"].([]SavedViewPayload)
	if !ok || len(dataViews) != 1 || dataViews[0].Name != payload.SavedViews[0].Name {
		t.Fatalf("savedViews payload mismatch: %#v", got.Data["savedViews"])
	}
	dataUser, ok := got.Data["user"].(*UserSession)
	if !ok || dataUser.ID != payload.User.ID {
		t.Fatalf("user payload mismatch: %#v", got.Data["user"])
	}
	dataWorkspace, ok := got.Data["workspace"].(map[string]any)
	if !ok || dataWorkspace["activeTab"] != "alerts" {
		t.Fatalf("workspace payload mismatch: %#v", got.Data["workspace"])
	}
	if got.Data["csrf"] != payload.CSRF {
		t.Fatalf("csrf mismatch: got %#v want %q", got.Data["csrf"], payload.CSRF)
	}
	dataPayload, ok := got.Data["payload"].(map[string]any)
	if !ok || dataPayload["cards"] != 4 {
		t.Fatalf("payload data mismatch: %#v", got.Data["payload"])
	}
}
