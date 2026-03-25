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
func DefineRoute(pattern string) (RouteContract, error) {
	normalized := normalizeRouteContractPattern(pattern)
	if normalized == "" {
		return RouteContract{}, fmt.Errorf("router: route pattern is required")
	}
	if normalized == routeContractCatchAll {
		return RouteContract{}, fmt.Errorf("router: route contracts do not support catch-all pattern %q", normalized)
	}

	trimmed := strings.Trim(normalized, "/")
	if trimmed == "" {
		return RouteContract{pattern: routeContractRootPath}, nil
	}

	rawSegments := strings.Split(trimmed, "/")
	segments := make([]routeContractSegment, 0, len(rawSegments))
	paramNames := make([]string, 0, len(rawSegments))
	seen := map[string]struct{}{}

	for _, segment := range rawSegments {
		if segment == "" {
			return RouteContract{}, fmt.Errorf("router: route pattern %q contains an empty segment", normalized)
		}
		if segment == routeContractCatchAll {
			return RouteContract{}, fmt.Errorf("router: route contracts do not support catch-all segment in %q", normalized)
		}
		if strings.HasPrefix(segment, ":") {
			name := strings.TrimSpace(strings.TrimPrefix(segment, ":"))
			if name == "" {
				return RouteContract{}, fmt.Errorf("router: route pattern %q contains an unnamed param segment", normalized)
			}
			if strings.HasSuffix(name, "?") {
				return RouteContract{}, fmt.Errorf("router: route contracts do not support optional params in %q", normalized)
			}
			if _, exists := seen[name]; exists {
				return RouteContract{}, fmt.Errorf("router: route pattern %q repeats param %q", normalized, name)
			}
			seen[name] = struct{}{}
			paramNames = append(paramNames, name)
			segments = append(segments, routeContractSegment{param: name})
			continue
		}
		if strings.Contains(segment, "*") {
			return RouteContract{}, fmt.Errorf("router: route contracts do not support wildcard segment %q in %q", segment, normalized)
		}
		segments = append(segments, routeContractSegment{literal: segment})
	}

	return RouteContract{
		pattern:    normalized,
		segments:   segments,
		paramNames: paramNames,
	}, nil
}

// MustDefineRoute validates a route pattern and panics if it is invalid.
func MustDefineRoute(pattern string) RouteContract {
	contract, err := DefineRoute(pattern)
	if err != nil {
		panic(err)
	}
	return contract
}

// Pattern returns the normalized route pattern used by the contract.
func (c RouteContract) Pattern() string {
	return c.pattern
}

// ParamNames returns the ordered param names required by the contract.
func (c RouteContract) ParamNames() []string {
	if len(c.paramNames) == 0 {
		return nil
	}
	names := make([]string, len(c.paramNames))
	copy(names, c.paramNames)
	return names
}

// Path builds a validated path for the contract from raw param values.
func (c RouteContract) Path(params map[string]string) (string, error) {
	if strings.TrimSpace(c.pattern) == "" {
		return "", fmt.Errorf("router: route contract is not initialized")
	}
	if err := c.validateParams(params); err != nil {
		return "", err
	}
	if len(c.segments) == 0 {
		return routeContractRootPath, nil
	}

	parts := make([]string, 0, len(c.segments))
	for _, segment := range c.segments {
		if segment.param == "" {
			parts = append(parts, segment.literal)
			continue
		}
		value := strings.TrimSpace(params[segment.param])
		if value == "" {
			return "", fmt.Errorf("router: route %q requires non-empty param %q", c.pattern, segment.param)
		}
		parts = append(parts, url.PathEscape(value))
	}
	return routeContractRootPath + strings.Join(parts, "/"), nil
}

// MustPath builds a path and panics if params do not satisfy the contract.
func (c RouteContract) MustPath(params map[string]string) string {
	path, err := c.Path(params)
	if err != nil {
		panic(err)
	}
	return path
}

// Href builds a validated navigation target for the contract from raw params and query values.
func (c RouteContract) Href(params map[string]string, query url.Values) (string, error) {
	path, err := c.Path(params)
	if err != nil {
		return "", err
	}
	encoded := query.Encode()
	if encoded == "" {
		return path, nil
	}
	return path + "?" + encoded, nil
}

// MustHref builds a navigation target and panics if params do not satisfy the contract.
func (c RouteContract) MustHref(params map[string]string, query url.Values) string {
	href, err := c.Href(params, query)
	if err != nil {
		panic(err)
	}
	return href
}

// PathFor builds a validated path from a typed param provider.
func (c RouteContract) PathFor(params RouteParamsProvider) (string, error) {
	return c.Path(routeParamsFromProvider(params))
}

// MustPathFor builds a path from a typed param provider and panics if the contract fails validation.
func (c RouteContract) MustPathFor(params RouteParamsProvider) string {
	path, err := c.PathFor(params)
	if err != nil {
		panic(err)
	}
	return path
}

// HrefFor builds a validated navigation target from typed param and query providers.
func (c RouteContract) HrefFor(params RouteParamsProvider, query RouteQueryProvider) (string, error) {
	return c.Href(routeParamsFromProvider(params), routeQueryFromProvider(query))
}

// MustHrefFor builds a navigation target from typed param and query providers and panics on validation failure.
func (c RouteContract) MustHrefFor(params RouteParamsProvider, query RouteQueryProvider) string {
	href, err := c.HrefFor(params, query)
	if err != nil {
		panic(err)
	}
	return href
}

func (c RouteContract) validateParams(params map[string]string) error {
	if len(params) == 0 || len(c.paramNames) == 0 {
		if len(params) == 0 {
			return nil
		}
	}

	allowed := map[string]struct{}{}
	for _, name := range c.paramNames {
		allowed[name] = struct{}{}
	}

	extra := make([]string, 0)
	for key := range params {
		key = strings.TrimSpace(key)
		if key == "" {
			extra = append(extra, key)
			continue
		}
		if _, ok := allowed[key]; !ok {
			extra = append(extra, key)
		}
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		return fmt.Errorf("router: route %q does not define params %s", c.pattern, strings.Join(extra, ", "))
	}
	return nil
}

func normalizeRouteContractPattern(pattern string) string {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return ""
	}
	if trimmed == "#" {
		return routeContractRootPath
	}
	trimmed = strings.TrimPrefix(trimmed, "#")
	if idx := strings.Index(trimmed, "?"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	if trimmed == "" {
		return routeContractRootPath
	}
	if !strings.HasPrefix(trimmed, routeContractRootPath) {
		trimmed = routeContractRootPath + trimmed
	}
	if trimmed != routeContractRootPath {
		trimmed = strings.TrimRight(trimmed, routeContractRootPath)
		if trimmed == "" {
			return routeContractRootPath
		}
	}
	return trimmed
}

func routeParamsFromProvider(provider RouteParamsProvider) map[string]string {
	if provider == nil {
		return nil
	}
	return provider.RouteParams()
}

func routeQueryFromProvider(provider RouteQueryProvider) url.Values {
	if provider == nil {
		return nil
	}
	return provider.RouteQuery()
}
