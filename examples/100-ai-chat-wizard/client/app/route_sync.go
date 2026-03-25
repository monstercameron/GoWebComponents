package app

import "strings"

func shouldResetDraftForRootRoute(threadRoutePublicID string, activeConvID int64, activeConvPublicID string) bool {
	if strings.TrimSpace(threadRoutePublicID) != "" || activeConvID <= 0 {
		return false
	}
	return strings.TrimSpace(activeConvPublicID) != ""
}

func shouldWarnPendingRootRoute(threadRoutePublicID string, activeConvID int64, activeConvPublicID string) bool {
	if strings.TrimSpace(threadRoutePublicID) != "" || activeConvID <= 0 {
		return false
	}
	return strings.TrimSpace(activeConvPublicID) == ""
}

func shouldResolveConversationRoute(threadRoutePublicID, activeConvPublicID string) bool {
	requestedPublicID := strings.TrimSpace(threadRoutePublicID)
	if requestedPublicID == "" {
		return false
	}
	return requestedPublicID != strings.TrimSpace(activeConvPublicID)
}

func shouldNormalizeActiveConversationRoute(threadRoutePublicID, activeConvPublicID string) bool {
	activePublicID := strings.TrimSpace(activeConvPublicID)
	if activePublicID == "" {
		return false
	}
	requestedPublicID := strings.TrimSpace(threadRoutePublicID)
	if requestedPublicID != "" && requestedPublicID != activePublicID {
		return false
	}
	return true
}

func routeSyncMismatchKey(threadRoutePublicID, activeConvPublicID string) string {
	requestedPublicID := strings.TrimSpace(threadRoutePublicID)
	activePublicID := strings.TrimSpace(activeConvPublicID)
	if requestedPublicID == "" || activePublicID == "" || requestedPublicID == activePublicID {
		return ""
	}
	return requestedPublicID + "!=" + activePublicID
}
