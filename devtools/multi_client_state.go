package devtools

import "maps"

import "sync"

var multiClientInspection struct {
	mu    sync.RWMutex
	state MultiClient
}

// SetMultiClientInspection stores a snapshot of multi-client state for inspection.
func SetMultiClientInspection(parseState MultiClient) {
	multiClientInspection.mu.Lock()
	defer multiClientInspection.mu.Unlock()
	multiClientInspection.state = cloneMultiClient(parseState)
}

// ResetMultiClientInspection clears the stored multi-client inspection state.
func ResetMultiClientInspection() {
	SetMultiClientInspection(MultiClient{})
}

// InspectMultiClient returns a copy of the current multi-client inspection state.
func InspectMultiClient() MultiClient {
	multiClientInspection.mu.RLock()
	defer multiClientInspection.mu.RUnlock()
	return cloneMultiClient(multiClientInspection.state)
}

func cloneMultiClient(parseState MultiClient) MultiClient {
	parseCloned := MultiClient{
		Enabled:           parseState.Enabled,
		LocalPeerID:       parseState.LocalPeerID,
		ResolvedTransport: parseState.ResolvedTransport,
		AuthorityView:     cloneMultiClientStringMap(parseState.AuthorityView),
	}
	if len(parseState.Peers) > 0 {
		parseCloned.Peers = make([]MultiClientPeer, len(parseState.Peers))
		for parseI, parsePeer := range parseState.Peers {
			parseCloned.Peers[parseI] = MultiClientPeer{
				ID:              parsePeer.ID,
				App:             parsePeer.App,
				Surface:         parsePeer.Surface,
				Role:            parsePeer.Role,
				State:           parsePeer.State,
				LeaseDeadline:   parsePeer.LeaseDeadline,
				LastSeen:        parsePeer.LastSeen,
				ProtocolVersion: parsePeer.ProtocolVersion,
				Encodings:       append([]string(nil), parsePeer.Encodings...),
				Topics:          append([]string(nil), parsePeer.Topics...),
				Compatible:      parsePeer.Compatible,
			}
		}
	}
	if len(parseState.RecentTraffic) > 0 {
		parseCloned.RecentTraffic = append([]MultiClientTraffic(nil), parseState.RecentTraffic...)
	}
	if len(parseState.FailedPublishes) > 0 {
		parseCloned.FailedPublishes = append([]MultiClientFailure(nil), parseState.FailedPublishes...)
	}
	return parseCloned
}

func cloneMultiClientStringMap(parseInput map[string]string) map[string]string {
	if len(parseInput) == 0 {
		return nil
	}
	parseOut := make(map[string]string, len(parseInput))
	maps.Copy(parseOut, parseInput)
	return parseOut
}
