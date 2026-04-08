package cachecore

import (
	"fmt"
	"strings"
)

// ScopeKey stores domain-agnostic scope dimensions for cache isolation.
type ScopeKey struct {
	AppVersion string
	SessionID  string
	UserID     string
	Workspace  string
	Locale     string
	RouteKey   string
	ThreadKey  string
}

// BuildScopeKey formats one stable scope key for cache and outbox records.
func BuildScopeKey(parseScope ScopeKey) string {
	return fmt.Sprintf(
		"app=%s|sid=%s|uid=%s|wid=%s|locale=%s|route=%s|thread=%s",
		parseNormalizeScopePart(parseScope.AppVersion),
		parseNormalizeScopePart(parseScope.SessionID),
		parseNormalizeScopePart(parseScope.UserID),
		parseNormalizeScopePart(parseScope.Workspace),
		parseNormalizeScopePart(parseScope.Locale),
		parseNormalizeScopePart(parseScope.RouteKey),
		parseNormalizeScopePart(parseScope.ThreadKey),
	)
}

// BuildScopedResourceKey combines one scope key with one resource key for lookups and de-duplication.
func BuildScopedResourceKey(parseScopeKey string, parseResourceKey string) string {
	parseScopeKey = strings.TrimSpace(parseScopeKey)
	parseResourceKey = strings.TrimSpace(parseResourceKey)
	if parseScopeKey == "" {
		return parseResourceKey
	}
	if parseResourceKey == "" {
		return parseScopeKey
	}
	return parseScopeKey + "::" + parseResourceKey
}

// BuildScopePrefixForLogout returns one deterministic scope prefix for logout/session-eviction events.
func BuildScopePrefixForLogout(parseScope ScopeKey) string {
	parseScope.SessionID = "-"
	parseScope.UserID = "-"
	parseScope.Workspace = "-"
	parseScope.RouteKey = "-"
	parseScope.ThreadKey = "-"
	return BuildScopeKey(parseScope)
}

// BuildScopePrefixForAccountSwitch returns one deterministic scope prefix for account-switch eviction.
func BuildScopePrefixForAccountSwitch(parseScope ScopeKey) string {
	parseScope.UserID = "-"
	parseScope.Workspace = "-"
	parseScope.RouteKey = "-"
	parseScope.ThreadKey = "-"
	return BuildScopeKey(parseScope)
}

// BuildScopePrefixForWorkspaceSwitch returns one deterministic scope prefix for workspace-switch eviction.
func BuildScopePrefixForWorkspaceSwitch(parseScope ScopeKey) string {
	parseScope.Workspace = "-"
	parseScope.RouteKey = "-"
	parseScope.ThreadKey = "-"
	return BuildScopeKey(parseScope)
}

// BuildScopePrefixForLocaleSwitch returns one deterministic scope prefix for locale-switch eviction.
func BuildScopePrefixForLocaleSwitch(parseScope ScopeKey) string {
	parseScope.Locale = "-"
	return BuildScopeKey(parseScope)
}

// parseNormalizeScopePart normalizes one scope-key segment into a deterministic safe token.
func parseNormalizeScopePart(parseValue string) string {
	parseValue = strings.TrimSpace(parseValue)
	if parseValue == "" {
		return "-"
	}
	parseValue = strings.NewReplacer("|", "_", "=", "_", ":", "_", "\n", "_", "\r", "_", "\t", "_").Replace(parseValue)
	return parseValue
}
