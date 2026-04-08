//go:build js && wasm

package app

import (
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
)

// shouldRedirectUnauthenticatedRouteToLanding returns whether the client should send a resolved unauthenticated app route back to landing.
func shouldRedirectUnauthenticatedRouteToLanding(parseCurrentPath string, isAuthResolved bool, isAuthenticated bool) bool {
	if !isAuthResolved || isAuthenticated {
		return false
	}
	return isChatRoute(parseCurrentPath)
}

// parseResolvePostLoginRoute resolves one auth-success route target from intent and role capability.
func parseResolvePostLoginRoute(parseIntentPath string, parseRoleSummary *chatpb.AuthRoleSummary) string {
	parseIntentPath = parseNormalizePostLoginRouteIntent(parseIntentPath)
	if parseIntentPath == "" {
		return chatRouteRoot
	}
	if parseIsAdminRouteIntentPath(parseIntentPath) && !parseCanAccessAdminFromRoleSummary(parseRoleSummary) {
		return chatRouteRoot
	}
	return parseIntentPath
}

// shouldSyncConvRoute reports whether one route should participate in active-conversation route normalization.
func shouldSyncConvRoute(parseCurrentPath string) bool {
	parseCurrentPath = strings.TrimSpace(parseCurrentPath)
	if parseCurrentPath == "" {
		return false
	}
	return parseCurrentPath == chatRouteRoot ||
		parseThreadRoutePublicIDFromPath(parseCurrentPath) != ""
}

// shouldResetDraftForRootRoute reports whether one true chat-root route should clear the loaded conversation draft.
func shouldResetDraftForRootRoute(parseCurrentPath string, parseThreadRoutePublicID string, parseActiveConvID int64, parseActiveConvPublicID string) bool {
	if !shouldSyncConvRoute(parseCurrentPath) || strings.TrimSpace(parseCurrentPath) != chatRouteRoot {
		return false
	}
	if strings.TrimSpace(parseThreadRoutePublicID) != "" || parseActiveConvID <= 0 {
		return false
	}
	return strings.TrimSpace(parseActiveConvPublicID) != ""
}

// shouldWarnPendingRootRoute reports whether one true chat-root route should warn about an unresolved conversation public id.
func shouldWarnPendingRootRoute(parseCurrentPath string, parseThreadRoutePublicID string, parseActiveConvID int64, parseActiveConvPublicID string) bool {
	if !shouldSyncConvRoute(parseCurrentPath) || strings.TrimSpace(parseCurrentPath) != chatRouteRoot {
		return false
	}
	if strings.TrimSpace(parseThreadRoutePublicID) != "" || parseActiveConvID <= 0 {
		return false
	}
	return strings.TrimSpace(parseActiveConvPublicID) == ""
}

func shouldResolveConversationRoute(parseThreadRoutePublicID, parseActiveConvPublicID string) bool {
	parseRequestedPublicID := strings.TrimSpace(parseThreadRoutePublicID)
	if parseRequestedPublicID == "" {
		return false
	}
	return parseRequestedPublicID != strings.TrimSpace(parseActiveConvPublicID)
}

// shouldNormalizeActiveConversationRoute reports whether one chat-root or thread route should normalize to the active conversation route.
func shouldNormalizeActiveConversationRoute(parseCurrentPath string, parseThreadRoutePublicID, parseActiveConvPublicID string) bool {
	if !shouldSyncConvRoute(parseCurrentPath) {
		return false
	}
	parseActivePublicID := strings.TrimSpace(parseActiveConvPublicID)
	if parseActivePublicID == "" {
		return false
	}
	parseRequestedPublicID := strings.TrimSpace(parseThreadRoutePublicID)
	if parseRequestedPublicID != "" && parseRequestedPublicID != parseActivePublicID {
		return false
	}
	return true
}

func parseRouteSyncMismatchKey(parseThreadRoutePublicID, parseActiveConvPublicID string) string {
	parseRequestedPublicID := strings.TrimSpace(parseThreadRoutePublicID)
	parseActivePublicID := strings.TrimSpace(parseActiveConvPublicID)
	if parseRequestedPublicID == "" || parseActivePublicID == "" || parseRequestedPublicID == parseActivePublicID {
		return ""
	}
	return parseRequestedPublicID + "!=" + parseActivePublicID
}
