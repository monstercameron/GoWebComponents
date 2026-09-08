package bootstrap

import "github.com/monstercameron/GoWebComponents/v6/ui"

type Payload struct {
	Route       RouteBootstrap     `json:"route"`
	Preferences PreferencesState   `json:"preferences"`
	I18n        I18nState          `json:"i18n"`
	Theme       ThemeState         `json:"theme"`
	Data        map[string]any     `json:"data"`
	SavedViews  []SavedViewPayload `json:"savedViews"`
	User        *UserSession       `json:"user,omitempty"`
	Workspace   map[string]any     `json:"workspace,omitempty"`
	CSRF        string             `json:"csrf,omitempty"`
}

type RouteBootstrap struct {
	Path        string              `json:"path"`
	Query       map[string][]string `json:"query,omitempty"`
	Params      map[string]string   `json:"params,omitempty"`
	Surface     string              `json:"surface"`
	Screen      string              `json:"screen"`
	Title       string              `json:"title"`
	Description string              `json:"description,omitempty"`
	Canonical   string              `json:"canonical,omitempty"`
}

type PreferencesState struct {
	Theme            string `json:"theme"`
	Locale           string `json:"locale"`
	Density          string `json:"density"`
	DefaultWarehouse string `json:"defaultWarehouse"`
}

type I18nState struct {
	Locale           string   `json:"locale"`
	SupportedLocales []string `json:"supportedLocales"`
	Direction        string   `json:"direction"`
}

type ThemeState struct {
	Mode                 string `json:"mode"`
	PrefersReducedMotion bool   `json:"prefersReducedMotion"`
}

type SavedViewPayload struct {
	Name          string            `json:"name"`
	Scope         string            `json:"scope"`
	SortKey       string            `json:"sortKey"`
	SortDirection string            `json:"sortDirection"`
	Filters       map[string]string `json:"filters"`
}

type UserSession struct {
	ID               string `json:"id"`
	DisplayName      string `json:"displayName"`
	Role             string `json:"role"`
	DefaultWarehouse string `json:"defaultWarehouse"`
}

func (parseP Payload) ToSSRBootstrap() ui.SSRBootstrap {
	return ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{
			Path:   parseP.Route.Path,
			Query:  parseP.Route.Query,
			Params: parseP.Route.Params,
		},
		Data: map[string]any{
			"route":       parseP.Route,
			"preferences": parseP.Preferences,
			"theme":       parseP.Theme,
			"savedViews":  parseP.SavedViews,
			"user":        parseP.User,
			"workspace":   parseP.Workspace,
			"csrf":        parseP.CSRF,
			"payload":     parseP.Data,
		},
		I18n: ui.SSRI18nBootstrap{
			Locale:    parseP.I18n.Locale,
			Direction: parseP.I18n.Direction,
		},
	}
}
