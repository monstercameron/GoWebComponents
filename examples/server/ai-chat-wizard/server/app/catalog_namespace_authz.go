package app

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type parseCatalogNamespaceClass string

const (
	parseCatalogNamespaceClassUnknown       parseCatalogNamespaceClass = "unknown"
	parseCatalogNamespaceClassPublic        parseCatalogNamespaceClass = "public"
	parseCatalogNamespaceClassAuthenticated parseCatalogNamespaceClass = "authenticated"
	parseCatalogNamespaceClassAdmin         parseCatalogNamespaceClass = "admin"
)

type parseCatalogNamespaceReadScope struct {
	namespace    string
	class        parseCatalogNamespaceClass
	userID       int64
	adminScope   parseAdminAccessScope
	adminSliceID string
}

// parseAuthorizeCatalogNamespaceRead enforces read access rules for one server-owned catalog namespace.
func (parseS *chatServer) parseAuthorizeCatalogNamespaceRead(parseCtx context.Context, parseNamespace string) (parseCatalogNamespaceReadScope, error) {
	parseNamespace = strings.TrimSpace(parseNamespace)
	if parseNamespace == "" {
		return parseCatalogNamespaceReadScope{}, status.Error(codes.InvalidArgument, "catalog namespace is required")
	}
	parseNamespaceClass := parseResolveCatalogNamespaceClass(parseNamespace)
	switch parseNamespaceClass {
	case parseCatalogNamespaceClassPublic:
		return parseCatalogNamespaceReadScope{
			namespace: parseNamespace,
			class:     parseNamespaceClass,
		}, nil
	case parseCatalogNamespaceClassAuthenticated:
		parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
		if parseErr != nil {
			return parseCatalogNamespaceReadScope{}, parseErr
		}
		return parseCatalogNamespaceReadScope{
			namespace: parseNamespace,
			class:     parseNamespaceClass,
			userID:    parseUserID,
		}, nil
	case parseCatalogNamespaceClassAdmin:
		parseSliceID, parseErr := parseResolveCatalogAdminSliceID(parseNamespace)
		if parseErr != nil {
			return parseCatalogNamespaceReadScope{}, parseErr
		}
		parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, parseSliceID)
		if parseErr != nil {
			return parseCatalogNamespaceReadScope{}, parseErr
		}
		return parseCatalogNamespaceReadScope{
			namespace:    parseNamespace,
			class:        parseNamespaceClass,
			userID:       parseScope.adminUserID,
			adminScope:   parseScope,
			adminSliceID: parseSliceID,
		}, nil
	default:
		return parseCatalogNamespaceReadScope{}, status.Errorf(codes.PermissionDenied, "catalog namespace %q is not allowlisted", parseNamespace)
	}
}

// parseResolveCatalogNamespaceClass classifies one namespace into public, authenticated, admin, or unknown.
func parseResolveCatalogNamespaceClass(parseNamespace string) parseCatalogNamespaceClass {
	parseRoot, _ := parseSplitCatalogNamespace(parseNamespace)
	switch parseRoot {
	case "marketing", "auth":
		return parseCatalogNamespaceClassPublic
	case "chat", "settings", "billing":
		return parseCatalogNamespaceClassAuthenticated
	case "admin", "dashboard", "business", "customers", "chats", "providers", "ops":
		return parseCatalogNamespaceClassAdmin
	default:
		return parseCatalogNamespaceClassUnknown
	}
}

// parseResolveCatalogAdminSliceID maps one admin namespace to one dashboard surface slice key.
func parseResolveCatalogAdminSliceID(parseNamespace string) (string, error) {
	parseRoot, parseRest := parseSplitCatalogNamespace(parseNamespace)
	switch parseRoot {
	case "business":
		return "dashboard.business.catalog", nil
	case "customers":
		return "dashboard.customers.catalog", nil
	case "chats":
		return "dashboard.chats.catalog", nil
	case "providers":
		return "dashboard.providers.catalog", nil
	case "ops":
		return "dashboard.ops.catalog", nil
	case "dashboard", "admin":
		parseSurfaceRoot, _ := parseSplitCatalogNamespace(parseRest)
		switch parseSurfaceRoot {
		case "":
			return "dashboard.home", nil
		case "business":
			return "dashboard.business.catalog", nil
		case "customers", "users", "user", "workspace", "support", "billing":
			return "dashboard.customers.catalog", nil
		case "chats", "usage", "conversations":
			return "dashboard.chats.catalog", nil
		case "providers", "provider", "routing", "cost_guardrail":
			return "dashboard.providers.catalog", nil
		case "ops", "incident", "feature_flag", "experiment":
			return "dashboard.ops.catalog", nil
		default:
			return "", status.Errorf(codes.PermissionDenied, "catalog admin namespace %q is not allowlisted", strings.TrimSpace(parseNamespace))
		}
	default:
		return "", status.Errorf(codes.PermissionDenied, "catalog admin namespace %q is not allowlisted", strings.TrimSpace(parseNamespace))
	}
}

// parseSplitCatalogNamespace splits one namespace into its first segment and remaining path.
func parseSplitCatalogNamespace(parseNamespace string) (string, string) {
	parseNamespace = strings.TrimSpace(strings.ToLower(parseNamespace))
	parseNamespace = strings.ReplaceAll(parseNamespace, "/", ".")
	parseNamespace = strings.Trim(parseNamespace, ".")
	if parseNamespace == "" {
		return "", ""
	}
	parseParts := strings.SplitN(parseNamespace, ".", 2)
	parseRoot := strings.TrimSpace(parseParts[0])
	if len(parseParts) == 1 {
		return parseRoot, ""
	}
	return parseRoot, strings.TrimSpace(parseParts[1])
}
