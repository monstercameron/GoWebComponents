package router

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

const (
	routeContractRootPath = "/"
	routeContractCatchAll = "*"
)

// RouteParamsProvider exposes typed route params for reverse routing helpers.
type RouteParamsProvider interface {
	RouteParams() map[string]string
}

// RouteQueryProvider exposes typed query values for reverse routing helpers.
type RouteQueryProvider interface {
	RouteQuery() url.Values
}

// RouteContract stores a validated route pattern together with reverse-routing helpers.
type RouteContract struct {
	pattern    string
	segments   []routeContractSegment
	paramNames []string
}

type routeContractSegment struct {
	literal string
	param   string
}

// DefineRoute validates a route pattern and returns a reusable reverse-routing contract.
func DefineRoute(parsePattern string) (RouteContract, error) {
	parseNormalized := normalizeRouteContractPattern(parsePattern)
	if parseNormalized == "" {
		return RouteContract{}, fmt.Errorf("router: route pattern is required")
	}
	if parseNormalized == routeContractCatchAll {
		return RouteContract{}, fmt.Errorf("router: route contracts do not support catch-all pattern %q", parseNormalized)
	}

	parseTrimmed := strings.Trim(parseNormalized, "/")
	if parseTrimmed == "" {
		return RouteContract{pattern: routeContractRootPath}, nil
	}

	parseRawSegments := strings.Split(parseTrimmed, "/")
	parseSegments := make([]routeContractSegment, 0, len(parseRawSegments))
	parseParamNames := make([]string, 0, len(parseRawSegments))
	parseSeen := map[string]struct{}{}

	for _, parseSegment := range parseRawSegments {
		if parseSegment == "" {
			return RouteContract{}, fmt.Errorf("router: route pattern %q contains an empty segment", parseNormalized)
		}
		if parseSegment == routeContractCatchAll {
			return RouteContract{}, fmt.Errorf("router: route contracts do not support catch-all segment in %q", parseNormalized)
		}
		if after, ok := strings.CutPrefix(parseSegment, ":"); ok {
			parseName := strings.TrimSpace(after)
			if parseName == "" {
				return RouteContract{}, fmt.Errorf("router: route pattern %q contains an unnamed param segment", parseNormalized)
			}
			if strings.HasSuffix(parseName, "?") {
				return RouteContract{}, fmt.Errorf("router: route contracts do not support optional params in %q", parseNormalized)
			}
			if _, parseExists := parseSeen[parseName]; parseExists {
				return RouteContract{}, fmt.Errorf("router: route pattern %q repeats param %q", parseNormalized, parseName)
			}
			parseSeen[parseName] = struct{}{}
			parseParamNames = append(parseParamNames, parseName)
			parseSegments = append(parseSegments, routeContractSegment{param: parseName})
			continue
		}
		if strings.Contains(parseSegment, "*") {
			return RouteContract{}, fmt.Errorf("router: route contracts do not support wildcard segment %q in %q", parseSegment, parseNormalized)
		}
		parseSegments = append(parseSegments, routeContractSegment{literal: parseSegment})
	}

	return RouteContract{
		pattern:    parseNormalized,
		segments:   parseSegments,
		paramNames: parseParamNames,
	}, nil
}

// MustDefineRoute validates a route pattern and panics if it is invalid.
func MustDefineRoute(parsePattern string) RouteContract {
	parseContract, parseErr := DefineRoute(parsePattern)
	if parseErr != nil {
		panic(parseErr)
	}
	return parseContract
}

// Pattern returns the normalized route pattern used by the contract.
func (parseC RouteContract) Pattern() string {
	return parseC.pattern
}

// ParamNames returns the ordered param names required by the contract.
func (parseC RouteContract) ParamNames() []string {
	if len(parseC.paramNames) == 0 {
		return nil
	}
	parseNames := make([]string, len(parseC.paramNames))
	copy(parseNames, parseC.paramNames)
	return parseNames
}

// Path builds a validated path for the contract from raw param values.
func (parseC RouteContract) Path(parseParams map[string]string) (string, error) {
	if strings.TrimSpace(parseC.pattern) == "" {
		return "", fmt.Errorf("router: route contract is not initialized")
	}
	if parseErr := parseC.validateParams(parseParams); parseErr != nil {
		return "", parseErr
	}
	if len(parseC.segments) == 0 {
		return routeContractRootPath, nil
	}

	parseParts := make([]string, 0, len(parseC.segments))
	for _, parseSegment := range parseC.segments {
		if parseSegment.param == "" {
			parseParts = append(parseParts, parseSegment.literal)
			continue
		}
		parseValue := strings.TrimSpace(parseParams[parseSegment.param])
		if parseValue == "" {
			return "", fmt.Errorf("router: route %q requires non-empty param %q", parseC.pattern, parseSegment.param)
		}
		parseParts = append(parseParts, url.PathEscape(parseValue))
	}
	return routeContractRootPath + strings.Join(parseParts, "/"), nil
}

// MustPath builds a path and panics if params do not satisfy the contract.
func (parseC RouteContract) MustPath(parseParams map[string]string) string {
	parsePath, parseErr := parseC.Path(parseParams)
	if parseErr != nil {
		panic(parseErr)
	}
	return parsePath
}

// Href builds a validated navigation target for the contract from raw params and query values.
func (parseC RouteContract) Href(parseParams map[string]string, parseQuery url.Values) (string, error) {
	parsePath, parseErr := parseC.Path(parseParams)
	if parseErr != nil {
		return "", parseErr
	}
	parseEncoded := parseQuery.Encode()
	if parseEncoded == "" {
		return parsePath, nil
	}
	return parsePath + "?" + parseEncoded, nil
}

// MustHref builds a navigation target and panics if params do not satisfy the contract.
func (parseC RouteContract) MustHref(parseParams map[string]string, parseQuery url.Values) string {
	parseHref, parseErr := parseC.Href(parseParams, parseQuery)
	if parseErr != nil {
		panic(parseErr)
	}
	return parseHref
}

// PathFor builds a validated path from a typed param provider.
func (parseC RouteContract) PathFor(parseParams RouteParamsProvider) (string, error) {
	return parseC.Path(routeParamsFromProvider(parseParams))
}

// MustPathFor builds a path from a typed param provider and panics if the contract fails validation.
func (parseC RouteContract) MustPathFor(parseParams RouteParamsProvider) string {
	parsePath, parseErr := parseC.PathFor(parseParams)
	if parseErr != nil {
		panic(parseErr)
	}
	return parsePath
}

// HrefFor builds a validated navigation target from typed param and query providers.
func (parseC RouteContract) HrefFor(parseParams RouteParamsProvider, parseQuery RouteQueryProvider) (string, error) {
	return parseC.Href(routeParamsFromProvider(parseParams), routeQueryFromProvider(parseQuery))
}

// MustHrefFor builds a navigation target from typed param and query providers and panics on validation failure.
func (parseC RouteContract) MustHrefFor(parseParams RouteParamsProvider, parseQuery RouteQueryProvider) string {
	parseHref, parseErr := parseC.HrefFor(parseParams, parseQuery)
	if parseErr != nil {
		panic(parseErr)
	}
	return parseHref
}

// validateParams is a core package helper.
func (parseC RouteContract) validateParams(parseParams map[string]string) error {
	if len(parseParams) == 0 || len(parseC.paramNames) == 0 {
		if len(parseParams) == 0 {
			return nil
		}
	}

	parseAllowed := map[string]struct{}{}
	for _, parseName := range parseC.paramNames {
		parseAllowed[parseName] = struct{}{}
	}

	parseExtra := make([]string, 0)
	for parseKey := range parseParams {
		parseKey = strings.TrimSpace(parseKey)
		if parseKey == "" {
			parseExtra = append(parseExtra, parseKey)
			continue
		}
		if _, parseOk := parseAllowed[parseKey]; !parseOk {
			parseExtra = append(parseExtra, parseKey)
		}
	}
	if len(parseExtra) > 0 {
		sort.Strings(parseExtra)
		return fmt.Errorf("router: route %q does not define params %s", parseC.pattern, strings.Join(parseExtra, ", "))
	}
	return nil
}

// normalizeRouteContractPattern is a core package helper.
func normalizeRouteContractPattern(parsePattern string) string {
	parseTrimmed := strings.TrimSpace(parsePattern)
	if parseTrimmed == "" {
		return ""
	}
	if parseTrimmed == "#" {
		return routeContractRootPath
	}
	parseTrimmed = strings.TrimPrefix(parseTrimmed, "#")
	if parseIdx := strings.Index(parseTrimmed, "?"); parseIdx >= 0 {
		parseTrimmed = parseTrimmed[:parseIdx]
	}
	if parseTrimmed == "" {
		return routeContractRootPath
	}
	if !strings.HasPrefix(parseTrimmed, routeContractRootPath) {
		parseTrimmed = routeContractRootPath + parseTrimmed
	}
	if parseTrimmed != routeContractRootPath {
		parseTrimmed = strings.TrimRight(parseTrimmed, routeContractRootPath)
		if parseTrimmed == "" {
			return routeContractRootPath
		}
	}
	return parseTrimmed
}

// routeParamsFromProvider is a core package helper.
func routeParamsFromProvider(parseProvider RouteParamsProvider) map[string]string {
	if parseProvider == nil {
		return nil
	}
	return parseProvider.RouteParams()
}

// routeQueryFromProvider is a core package helper.
func routeQueryFromProvider(parseProvider RouteQueryProvider) url.Values {
	if parseProvider == nil {
		return nil
	}
	return parseProvider.RouteQuery()
}
