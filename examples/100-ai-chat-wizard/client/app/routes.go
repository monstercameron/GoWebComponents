//go:build js && wasm

package app

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/router"
)

const chatRouteRoot = "/"
const chatRouteThreadPattern = "/thread/:publicID"
const chatRouteCanvasPattern = "/thread/:publicID/canvas/:canvasID"

func chatThreadPath(publicID string) string {
	publicID = strings.TrimSpace(publicID)
	if publicID == "" {
		return chatRouteRoot
	}
	return fmt.Sprintf("/thread/%s", publicID)
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
	return fmt.Sprintf("/thread/%s/canvas/%s", publicID, canvasID)
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
