//go:build js && wasm

package app

import (
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
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

func shouldResetDraftForRootRoute(parseThreadRoutePublicID string, parseActiveConvID int64, parseActiveConvPublicID string) bool {
	if strings.TrimSpace(parseThreadRoutePublicID) != "" || parseActiveConvID <= 0 {
		return false
	}
	return strings.TrimSpace(parseActiveConvPublicID) != ""
}

func shouldWarnPendingRootRoute(parseThreadRoutePublicID string, parseActiveConvID int64, parseActiveConvPublicID string) bool {
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

func shouldNormalizeActiveConversationRoute(parseThreadRoutePublicID, parseActiveConvPublicID string) bool {
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
