package agentbridge

import (
	"net/url"
	"strings"
)

// buildBridgeWSURL returns the agent hub WebSocket URL for a page. The scheme
// follows the page: an https page must dial wss (a plain ws dial from a secure
// page is blocked as mixed content), every other page dials ws. A non-empty
// parseOverride (taken only from the page-controlled __GWC_AGENT_BRIDGE
// bootstrap object, never from the address bar, so a crafted link cannot
// redirect page state to another host) replaces the derived endpoint when it
// is an absolute ws or wss URL; the token is always attached. It returns false
// when no usable endpoint can be built.
func buildBridgeWSURL(parsePageProtocol string, parseHost string, parseToken string, parseOverride string) (string, bool) {
	if parseOverride = strings.TrimSpace(parseOverride); parseOverride != "" {
		parseURL, parseErr := url.Parse(parseOverride)
		if parseErr != nil || (parseURL.Scheme != "ws" && parseURL.Scheme != "wss") || parseURL.Host == "" {
			return "", false
		}
		parseQuery := parseURL.Query()
		parseQuery.Set("token", parseToken)
		parseURL.RawQuery = parseQuery.Encode()
		return parseURL.String(), true
	}
	if parseHost == "" {
		return "", false
	}
	parseScheme := "ws"
	if strings.EqualFold(strings.TrimSuffix(parsePageProtocol, ":"), "https") {
		parseScheme = "wss"
	}
	return parseScheme + "://" + parseHost + "/gwc-agent?token=" + url.QueryEscape(parseToken), true
}
