package atlas

import (
	"encoding/json"
	"maps"
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/ui"
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

func SupportedLocales() []string {
	return []string{"en", "fr", "ar"}
}

func LocaleDirection(parseLocale string) string {
	if strings.EqualFold(strings.TrimSpace(parseLocale), "ar") {
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
	parsePrefs := DefaultPreferences()
	return ThemeState{Mode: parsePrefs.Theme, PrefersReducedMotion: false}
}

func DefaultI18n(parseLocale string) I18nState {
	parseTrimmed := strings.TrimSpace(parseLocale)
	if parseTrimmed == "" {
		parseTrimmed = DefaultPreferences().Locale
	}
	return I18nState{
		Locale:           parseTrimmed,
		SupportedLocales: SupportedLocales(),
		Direction:        LocaleDirection(parseTrimmed),
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

func (parseP Payload) ToSSRBootstrap() ui.SSRBootstrap {
	return ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{
			Path:   parseP.Route.Path,
			Query:  cloneQuery(parseP.Route.Query),
			Params: cloneParams(parseP.Route.Params),
		},
		I18n: ui.SSRI18nBootstrap{
			Locale:         parseP.I18n.Locale,
			FallbackLocale: "en",
			Direction:      parseP.I18n.Direction,
		},
		Data: map[string]any{
			atlasBootstrapDataKey: bootstrapData{
				Route:       parseP.Route,
				Preferences: parseP.Preferences,
				I18n:        parseP.I18n,
				Theme:       parseP.Theme,
				Data:        cloneData(parseP.Data),
				Requests:    cloneRequestsForBootstrap(parseP.Data, parseP.Requests),
				SavedViews:  append([]SavedViewPayload(nil), parseP.SavedViews...),
				CSRF:        parseP.CSRF,
				User:        parseP.User,
			},
		},
	}
}

func PayloadFromSSRBootstrap(parseInput ui.SSRBootstrap) Payload {
	parsePayload := Payload{
		Route: RouteBootstrap{
			Path:   parseInput.Route.Path,
			Query:  cloneQuery(parseInput.Route.Query),
			Params: cloneParams(parseInput.Route.Params),
		},
		Preferences: DefaultPreferences(),
		I18n:        DefaultI18n(parseInput.I18n.Locale),
		Theme:       DefaultTheme(),
		Data:        map[string]any{},
		Requests:    map[string]Request{},
	}
	if parseRaw, parseOk := parseInput.Data[atlasBootstrapDataKey]; parseOk {
		parseDecoded := bootstrapData{}
		if decodeInto(parseRaw, &parseDecoded) == nil {
			if parseDecoded.Route.Path != "" {
				parsePayload.Route.Path = parseDecoded.Route.Path
			}
			parsePayload.Route.Surface = parseDecoded.Route.Surface
			parsePayload.Route.Screen = parseDecoded.Route.Screen
			parsePayload.Route.Title = parseDecoded.Route.Title
			parsePayload.Route.Description = parseDecoded.Route.Description
			parsePayload.Route.Canonical = parseDecoded.Route.Canonical
			parsePayload.Preferences = parseDecoded.Preferences
			parsePayload.I18n = parseDecoded.I18n
			parsePayload.Theme = parseDecoded.Theme
			parsePayload.Data = cloneData(parseDecoded.Data)
			parsePayload.Requests = cloneRequests(parseDecoded.Requests)
			parsePayload.SavedViews = append([]SavedViewPayload(nil), parseDecoded.SavedViews...)
			parsePayload.CSRF = parseDecoded.CSRF
			parsePayload.User = parseDecoded.User
		}
	}
	if parsePayload.I18n.Locale == "" {
		parsePayload.I18n = DefaultI18n(parsePayload.Preferences.Locale)
	}
	return parsePayload
}

func decodeInto(parseInput any, parseTarget any) error {
	parseEncoded, parseErr := json.Marshal(parseInput)
	if parseErr != nil {
		return parseErr
	}
	return json.Unmarshal(parseEncoded, parseTarget)
}

func cloneQuery(parseInput map[string][]string) map[string][]string {
	if len(parseInput) == 0 {
		return map[string][]string{}
	}
	parseClone := make(map[string][]string, len(parseInput))
	for parseKey, parseValues := range parseInput {
		parseClone[parseKey] = append([]string(nil), parseValues...)
	}
	return parseClone
}

func cloneParams(parseInput map[string]string) map[string]string {
	if len(parseInput) == 0 {
		return map[string]string{}
	}
	parseClone := make(map[string]string, len(parseInput))
	maps.Copy(parseClone, parseInput)
	return parseClone
}

func cloneData(parseInput map[string]any) map[string]any {
	if len(parseInput) == 0 {
		return map[string]any{}
	}
	parseClone := make(map[string]any, len(parseInput))
	maps.Copy(parseClone, parseInput)
	return parseClone
}

func cloneRequests(parseInput map[string]Request) map[string]Request {
	if len(parseInput) == 0 {
		return map[string]Request{}
	}
	parseClone := make(map[string]Request, len(parseInput))
	for parseKey, parseValue := range parseInput {
		parseClone[parseKey] = Request{
			Method: parseValue.Method,
			URL:    parseValue.URL,
			Status: parseValue.Status,
			Data:   cloneData(parseValue.Data),
		}
	}
	return parseClone
}

func cloneRequestsForBootstrap(parseRouteData map[string]any, parseInput map[string]Request) map[string]Request {
	if len(parseInput) == 0 {
		return map[string]Request{}
	}
	parseClone := make(map[string]Request, len(parseInput))
	for parseKey, parseValue := range parseInput {
		parseRequest := Request{
			Method: parseValue.Method,
			URL:    parseValue.URL,
			Status: parseValue.Status,
		}
		if len(parseValue.Data) > 0 {
			parseData := map[string]any{}
			for parseDataKey, parseItem := range parseValue.Data {
				if _, parseDuplicated := parseRouteData[parseDataKey]; parseDuplicated {
					continue
				}
				parseData[parseDataKey] = parseItem
			}
			if len(parseData) > 0 {
				parseRequest.Data = cloneData(parseData)
			}
		}
		parseClone[parseKey] = parseRequest
	}
	return parseClone
}

func StartupRequest(parsePayload Payload, parseKey string) (Request, bool) {
	if len(parsePayload.Requests) == 0 {
		return Request{}, false
	}
	parseRequest, parseOk := parsePayload.Requests[strings.TrimSpace(parseKey)]
	if !parseOk {
		return Request{}, false
	}
	parseRequest.Data = cloneData(parseRequest.Data)
	return parseRequest, true
}
