package localfirst

// Cursor is a typed collaboration cursor/selection — where a peer is looking and what (if
// anything) they have selected, for live multiplayer awareness. It is presence-shaped
// (ephemeral, last-write, never conflict-merged): "where the cursor is right now" has no
// history. Carry it as the State payload of a Presence (JSON-encode it) so cursors ride the
// existing PresenceSet without a new transport.
type Cursor struct {
	// ClientID identifies the peer the cursor belongs to.
	ClientID string `json:"client"`
	// Anchor is the primary caret position (an app-defined offset/index).
	Anchor int `json:"anchor"`
	// Head is the moving end of a selection; equal to Anchor when there is no selection.
	Head int `json:"head"`
	// Label is an optional display name shown next to the cursor.
	Label string `json:"label,omitempty"`
}

// HasSelection reports whether the cursor spans a range (Anchor != Head).
func (parseC Cursor) HasSelection() bool {
	return parseC.Anchor != parseC.Head
}

// Start returns the lower of Anchor/Head — the selection's start regardless of drag
// direction.
func (parseC Cursor) Start() int {
	if parseC.Anchor <= parseC.Head {
		return parseC.Anchor
	}
	return parseC.Head
}

// End returns the higher of Anchor/Head — the selection's end regardless of drag direction.
func (parseC Cursor) End() int {
	if parseC.Anchor >= parseC.Head {
		return parseC.Anchor
	}
	return parseC.Head
}
