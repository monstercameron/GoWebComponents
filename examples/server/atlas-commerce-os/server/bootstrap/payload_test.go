package bootstrap

import "testing"

func TestPayloadToSSRBootstrap(parseT *testing.T) {
	parsePayload := Payload{
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

	parseGot := parsePayload.ToSSRBootstrap()

	if parseGot.Route.Path != parsePayload.Route.Path {
		parseT.Fatalf("Route.Path mismatch: got %q want %q", parseGot.Route.Path, parsePayload.Route.Path)
	}
	if parseGot.Route.Query["warehouse"][0] != "east" {
		parseT.Fatalf("Route.Query mismatch: got %#v", parseGot.Route.Query)
	}
	if parseGot.Route.Params["sku"] != "desk-1" {
		parseT.Fatalf("Route.Params mismatch: got %#v", parseGot.Route.Params)
	}
	if parseGot.I18n.Locale != parsePayload.I18n.Locale || parseGot.I18n.Direction != parsePayload.I18n.Direction {
		parseT.Fatalf("I18n mismatch: got %#v want locale=%q direction=%q", parseGot.I18n, parsePayload.I18n.Locale, parsePayload.I18n.Direction)
	}

	parseDataRoute, parseOk := parseGot.Data["route"].(RouteBootstrap)
	if !parseOk || parseDataRoute.Title != parsePayload.Route.Title || parseDataRoute.Surface != parsePayload.Route.Surface {
		parseT.Fatalf("route payload mismatch: %#v", parseGot.Data["route"])
	}
	parseDataPreferences, parseOk := parseGot.Data["preferences"].(PreferencesState)
	if !parseOk || parseDataPreferences.DefaultWarehouse != parsePayload.Preferences.DefaultWarehouse {
		parseT.Fatalf("preferences payload mismatch: %#v", parseGot.Data["preferences"])
	}
	parseDataTheme, parseOk := parseGot.Data["theme"].(ThemeState)
	if !parseOk || parseDataTheme.Mode != parsePayload.Theme.Mode || !parseDataTheme.PrefersReducedMotion {
		parseT.Fatalf("theme payload mismatch: %#v", parseGot.Data["theme"])
	}
	parseDataViews, parseOk := parseGot.Data["savedViews"].([]SavedViewPayload)
	if !parseOk || len(parseDataViews) != 1 || parseDataViews[0].Name != parsePayload.SavedViews[0].Name {
		parseT.Fatalf("savedViews payload mismatch: %#v", parseGot.Data["savedViews"])
	}
	parseDataUser, parseOk := parseGot.Data["user"].(*UserSession)
	if !parseOk || parseDataUser.ID != parsePayload.User.ID {
		parseT.Fatalf("user payload mismatch: %#v", parseGot.Data["user"])
	}
	parseDataWorkspace, parseOk := parseGot.Data["workspace"].(map[string]any)
	if !parseOk || parseDataWorkspace["activeTab"] != "alerts" {
		parseT.Fatalf("workspace payload mismatch: %#v", parseGot.Data["workspace"])
	}
	if parseGot.Data["csrf"] != parsePayload.CSRF {
		parseT.Fatalf("csrf mismatch: got %#v want %q", parseGot.Data["csrf"], parsePayload.CSRF)
	}
	parseDataPayload, parseOk := parseGot.Data["payload"].(map[string]any)
	if !parseOk || parseDataPayload["cards"] != 4 {
		parseT.Fatalf("payload data mismatch: %#v", parseGot.Data["payload"])
	}
}
