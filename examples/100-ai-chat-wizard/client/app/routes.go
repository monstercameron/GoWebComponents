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
const marketingSignupRoute = "/signup"

func parseChatThreadPath(parsePublicID string) string {
	parsePublicID = strings.TrimSpace(parsePublicID)
	if parsePublicID == "" {
		return chatRouteRoot
	}
	return fmt.Sprintf("/app/thread/%s", parsePublicID)
}

func parseCurrentThreadRoutePublicID() string {
	return strings.TrimSpace(router.UseParams().Get("publicID"))
}

func parseCurrentCanvasRouteID() string {
	return strings.TrimSpace(router.UseParams().Get("canvasID"))
}

func parseChatCanvasPath(parsePublicID, parseCanvasID string) string {
	parsePublicID = strings.TrimSpace(parsePublicID)
	parseCanvasID = strings.TrimSpace(parseCanvasID)
	if parsePublicID == "" || parseCanvasID == "" {
		return parseChatThreadPath(parsePublicID)
	}
	return fmt.Sprintf("/app/thread/%s/canvas/%s", parsePublicID, parseCanvasID)
}

func isLandingRoute(parsePath string) bool {
	switch strings.TrimSpace(parsePath) {
	case authLandingRoute, marketingHomeRoute, marketingCapabilitiesRoute, marketingPricingRoute, marketingSignupRoute:
		return true
	default:
		return false
	}
}

func isChatRoute(parsePath string) bool {
	parsePath = strings.TrimSpace(parsePath)
	return parsePath == chatRouteRoot || strings.HasPrefix(parsePath, chatRouteRoot+"/")
}

func parseThreadRoutePublicIDFromPath(parsePath string) string {
	parsePath = strings.TrimSpace(parsePath)
	if parsePath == "" {
		return ""
	}
	if parseIdx := strings.Index(parsePath, "#"); parseIdx >= 0 {
		parsePath = parsePath[:parseIdx]
	}
	if parseIdx2 := strings.Index(parsePath, "?"); parseIdx2 >= 0 {
		parsePath = parsePath[:parseIdx2]
	}
	parseTrimmed := strings.Trim(strings.TrimSpace(parsePath), "/")
	if parseTrimmed == "" {
		return ""
	}
	parseSegments := strings.Split(parseTrimmed, "/")
	if len(parseSegments) < 3 || parseSegments[0] != "app" || parseSegments[1] != "thread" {
		return ""
	}
	return strings.TrimSpace(parseSegments[2])
}

func parseFindConversationSummaryByID(parseConversations []convSummary, parseId int64) (convSummary, bool) {
	for _, parseSummary := range parseConversations {
		if parseSummary.ID == parseId {
			return parseSummary, true
		}
	}
	return convSummary{}, false
}

func parseSummaryPublicIDForID(parseConversations []convSummary, parseId int64) string {
	parseSummary, parseOk := parseFindConversationSummaryByID(parseConversations, parseId)
	if !parseOk {
		return ""
	}
	return strings.TrimSpace(parseSummary.PublicID)
}

func parseFindConversationSummaryByPublicID(parseConversations []convSummary, parsePublicID string) (convSummary, bool) {
	parsePublicID = strings.TrimSpace(parsePublicID)
	if parsePublicID == "" {
		return convSummary{}, false
	}
	for _, parseSummary := range parseConversations {
		if parseSummary.PublicID == parsePublicID {
			return parseSummary, true
		}
	}
	return convSummary{}, false
}
