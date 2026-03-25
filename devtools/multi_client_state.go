package devtools

import "sync"

var multiClientInspection struct {
	mu    sync.RWMutex
	state MultiClient
}

// SetMultiClientInspection stores a snapshot of multi-client state for inspection.
func SetMultiClientInspection(state MultiClient) {
	multiClientInspection.mu.Lock()
	defer multiClientInspection.mu.Unlock()
	multiClientInspection.state = cloneMultiClient(state)
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

func cloneMultiClient(state MultiClient) MultiClient {
	cloned := MultiClient{
		Enabled:           state.Enabled,
		LocalPeerID:       state.LocalPeerID,
		ResolvedTransport: state.ResolvedTransport,
		AuthorityView:     cloneMultiClientStringMap(state.AuthorityView),
	}
	if len(state.Peers) > 0 {
		cloned.Peers = make([]MultiClientPeer, len(state.Peers))
		for i, peer := range state.Peers {
			cloned.Peers[i] = MultiClientPeer{
				ID:              peer.ID,
				App:             peer.App,
				Surface:         peer.Surface,
				Role:            peer.Role,
				State:           peer.State,
				LeaseDeadline:   peer.LeaseDeadline,
				LastSeen:        peer.LastSeen,
				ProtocolVersion: peer.ProtocolVersion,
				Encodings:       append([]string(nil), peer.Encodings...),
				Topics:          append([]string(nil), peer.Topics...),
				Compatible:      peer.Compatible,
			}
		}
	}
	if len(state.RecentTraffic) > 0 {
		cloned.RecentTraffic = append([]MultiClientTraffic(nil), state.RecentTraffic...)
	}
	if len(state.FailedPublishes) > 0 {
		cloned.FailedPublishes = append([]MultiClientFailure(nil), state.FailedPublishes...)
	}
	return cloned
}

func cloneMultiClientStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
