package agentbridge

import "testing"

// TestBuildBridgeWSURLFollowsPageScheme pins ws/wss selection and the override.
func TestBuildBridgeWSURLFollowsPageScheme(t *testing.T) {
	parseCases := []struct {
		name, protocol, host, override, want string
		ok                                   bool
	}{
		{"http page dials ws", "http:", "localhost:8080", "", "ws://localhost:8080/gwc-agent?token=a+b", true},
		{"https page dials wss", "https:", "dev.example.com", "", "wss://dev.example.com/gwc-agent?token=a+b", true},
		{"unknown protocol falls back to ws", "", "h:1", "", "ws://h:1/gwc-agent?token=a+b", true},
		{"override wins and gets token", "https:", "ignored", "wss://hub.example.com/x", "wss://hub.example.com/x?token=a+b", true},
		{"override replaces an existing token", "http:", "h", "ws://hub/x?token=old&k=v", "ws://hub/x?k=v&token=a+b", true},
		{"override must be ws or wss", "https:", "h", "https://hub/x", "", false},
		{"override must have a host", "https:", "h", "ws:///x", "", false},
		{"missing host without override", "https:", "", "", "", false},
	}
	for _, parseCase := range parseCases {
		parseGot, parseOK := buildBridgeWSURL(parseCase.protocol, parseCase.host, "a b", parseCase.override)
		if parseOK != parseCase.ok || parseGot != parseCase.want {
			t.Errorf("%s: got (%q, %v), want (%q, %v)", parseCase.name, parseGot, parseOK, parseCase.want, parseCase.ok)
		}
	}
}
