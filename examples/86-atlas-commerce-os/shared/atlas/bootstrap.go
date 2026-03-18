package atlas

import (
	"encoding/json"
	"strings"

	"github.com/monstercameron/GoWebComponents/ui"
)

const atlasBootstrapDataKey = "atlas"

type Payload struct {
	Route       RouteBootstrap     `json:"route"`
	Preferences PreferencesState   `json:"preferences"`
	I18n        I18nState          `json:"i18n"`
	Theme       ThemeState         `json:"theme"`
	Data        map[string]any     `json:"data"`
	Requests    map[string]Request `json:"requests,omitempty"`
	SavedViews  []SavedViewPayload `json:"savedViews"`
	CSRF        string             `json:"csrf,omitempty"`
	User        *UserSession       `json:"user,omitempty"`
}

type Request struct {
	Method string         `json:"method"`
	URL    string         `json:"url"`
	Status int            `json:"status"`
	Data   map[string]any `json:"data,omitempty"`
}

type RouteBootstrap struct {
	Path    string              `json:"path"`
	Query   map[string][]string `json:"query,omitempty"`
	Params  map[string]string   `json:"params,omitempty"`
	Surface string              `json:"surface"`
	Screen  string              `json:"screen"`
	Title   string              `json:"title"`
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

func SupportedLocales() []string {
	return []string{"en", "fr", "ar"}
}

func LocaleDirection(locale string) string {
	if strings.EqualFold(strings.TrimSpace(locale), "ar") {
		return "rtl"
	}
	return "ltr"
}

func DefaultPreferences() PreferencesState {
	return PreferencesState{
		Theme:            "dark",
		Locale:           "en",
		Density:          "compact",
		DefaultWarehouse: "new-jersey-hub",
	}
}

func DefaultTheme() ThemeState {
	prefs := DefaultPreferences()
	return ThemeState{Mode: prefs.Theme, PrefersReducedMotion: false}
}

func DefaultI18n(locale string) I18nState {
	trimmed := strings.TrimSpace(locale)
	if trimmed == "" {
		trimmed = DefaultPreferences().Locale
	}
	return I18nState{
		Locale:           trimmed,
		SupportedLocales: SupportedLocales(),
		Direction:        LocaleDirection(trimmed),
	}
}

type bootstrapData struct {
	Route       RouteBootstrap     `json:"route"`
	Preferences PreferencesState   `json:"preferences"`
	I18n        I18nState          `json:"i18n"`
	Theme       ThemeState         `json:"theme"`
	Data        map[string]any     `json:"data"`
	Requests    map[string]Request `json:"requests,omitempty"`
	SavedViews  []SavedViewPayload `json:"savedViews"`
	CSRF        string             `json:"csrf,omitempty"`
	User        *UserSession       `json:"user,omitempty"`
}

func (p Payload) ToSSRBootstrap() ui.SSRBootstrap {
	return ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{
			Path:   p.Route.Path,
			Query:  cloneQuery(p.Route.Query),
			Params: cloneParams(p.Route.Params),
		},
		I18n: ui.SSRI18nBootstrap{
			Locale:         p.I18n.Locale,
			FallbackLocale: "en",
			Direction:      p.I18n.Direction,
		},
		Data: map[string]any{
			atlasBootstrapDataKey: bootstrapData{
				Route:       p.Route,
				Preferences: p.Preferences,
				I18n:        p.I18n,
				Theme:       p.Theme,
				Data:        cloneData(p.Data),
				Requests:    cloneRequests(p.Requests),
				SavedViews:  append([]SavedViewPayload(nil), p.SavedViews...),
				CSRF:        p.CSRF,
				User:        p.User,
			},
		},
	}
}

func PayloadFromSSRBootstrap(input ui.SSRBootstrap) Payload {
	payload := Payload{
		Route: RouteBootstrap{
			Path:   input.Route.Path,
			Query:  cloneQuery(input.Route.Query),
			Params: cloneParams(input.Route.Params),
		},
		Preferences: DefaultPreferences(),
		I18n:        DefaultI18n(input.I18n.Locale),
		Theme:       DefaultTheme(),
		Data:        map[string]any{},
		Requests:    map[string]Request{},
	}
	if raw, ok := input.Data[atlasBootstrapDataKey]; ok {
		decoded := bootstrapData{}
		if decodeInto(raw, &decoded) == nil {
			if decoded.Route.Path != "" {
				payload.Route.Surface = decoded.Route.Surface
				payload.Route.Screen = decoded.Route.Screen
				payload.Route.Title = decoded.Route.Title
			}
			payload.Preferences = decoded.Preferences
			payload.I18n = decoded.I18n
			payload.Theme = decoded.Theme
			payload.Data = cloneData(decoded.Data)
			payload.Requests = cloneRequests(decoded.Requests)
			payload.SavedViews = append([]SavedViewPayload(nil), decoded.SavedViews...)
			payload.CSRF = decoded.CSRF
			payload.User = decoded.User
		}
	}
	if payload.I18n.Locale == "" {
		payload.I18n = DefaultI18n(payload.Preferences.Locale)
	}
	return payload
}

func decodeInto(input any, target any) error {
	encoded, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, target)
}

func cloneQuery(input map[string][]string) map[string][]string {
	if len(input) == 0 {
		return map[string][]string{}
	}
	clone := make(map[string][]string, len(input))
	for key, values := range input {
		clone[key] = append([]string(nil), values...)
	}
	return clone
}

func cloneParams(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	clone := make(map[string]string, len(input))
	for key, value := range input {
		clone[key] = value
	}
	return clone
}

func cloneData(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	clone := make(map[string]any, len(input))
	for key, value := range input {
		clone[key] = value
	}
	return clone
}

func cloneRequests(input map[string]Request) map[string]Request {
	if len(input) == 0 {
		return map[string]Request{}
	}
	clone := make(map[string]Request, len(input))
	for key, value := range input {
		clone[key] = Request{
			Method: value.Method,
			URL:    value.URL,
			Status: value.Status,
			Data:   cloneData(value.Data),
		}
	}
	return clone
}

func StartupRequest(payload Payload, key string) (Request, bool) {
	if len(payload.Requests) == 0 {
		return Request{}, false
	}
	request, ok := payload.Requests[strings.TrimSpace(key)]
	if !ok {
		return Request{}, false
	}
	request.Data = cloneData(request.Data)
	return request, true
}
