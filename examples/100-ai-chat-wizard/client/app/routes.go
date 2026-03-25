//go:build js && wasm

package app

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/router"
)

const chatRouteRoot = "/app"
const chatRouteThreadPattern = "/app/thread/:publicID"
const chatRouteCanvasPattern = "/app/thread/:publicID/canvas/:canvasID"
const authLandingRoute = "/"
const marketingHomeRoute = "/home"
const marketingCapabilitiesRoute = "/capabilities"
const marketingPricingRoute = "/pricing"

func chatThreadPath(publicID string) string {
	publicID = strings.TrimSpace(publicID)
	if publicID == "" {
		return chatRouteRoot
	}
	return fmt.Sprintf("/app/thread/%s", publicID)
}

func currentThreadRoutePublicID() string {
	return strings.TrimSpace(router.UseParams().Get("publicID"))
}

func currentCanvasRouteID() string {
	return strings.TrimSpace(router.UseParams().Get("canvasID"))
}

func chatCanvasPath(publicID, canvasID string) string {
	publicID = strings.TrimSpace(publicID)
	canvasID = strings.TrimSpace(canvasID)
	if publicID == "" || canvasID == "" {
		return chatThreadPath(publicID)
	}
	return fmt.Sprintf("/app/thread/%s/canvas/%s", publicID, canvasID)
}

func isLandingRoute(path string) bool {
	switch strings.TrimSpace(path) {
	case authLandingRoute, marketingHomeRoute, marketingCapabilitiesRoute, marketingPricingRoute:
		return true
	default:
		return false
	}
}

func isChatRoute(path string) bool {
	path = strings.TrimSpace(path)
	return path == chatRouteRoot || strings.HasPrefix(path, chatRouteRoot+"/")
}

func threadRoutePublicIDFromPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if idx := strings.Index(path, "#"); idx >= 0 {
		path = path[:idx]
	}
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	if trimmed == "" {
		return ""
	}
	segments := strings.Split(trimmed, "/")
	if len(segments) < 3 || segments[0] != "app" || segments[1] != "thread" {
		return ""
	}
	return strings.TrimSpace(segments[2])
}

func findConversationSummaryByID(conversations []convSummary, id int64) (convSummary, bool) {
	for _, summary := range conversations {
		if summary.ID == id {
			return summary, true
		}
	}
	return convSummary{}, false
}

func summaryPublicIDForID(conversations []convSummary, id int64) string {
	summary, ok := findConversationSummaryByID(conversations, id)
	if !ok {
		return ""
	}
	return strings.TrimSpace(summary.PublicID)
}

func findConversationSummaryByPublicID(conversations []convSummary, publicID string) (convSummary, bool) {
	publicID = strings.TrimSpace(publicID)
	if publicID == "" {
		return convSummary{}, false
	}
	for _, summary := range conversations {
		if summary.PublicID == publicID {
			return summary, true
		}
	}
	return convSummary{}, false
}
