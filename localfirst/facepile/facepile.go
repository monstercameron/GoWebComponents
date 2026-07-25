// Package facepile renders collaboration presence as a GoWebComponents component (FC6) — the
// "who's here right now" surface over a localfirst.PresenceSet. It is the visual half of the
// collaboration story whose engine (convergent document store + presence) ships in
// localfirst; the framework renders its own multiplayer UI.
package facepile

import (
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/localfirst"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// Facepile renders the live peers in a presence set as a horizontal pile: one marker per
// peer, labeled by client id, with the peer's opaque state carried as a title for hover
// detail. Restyle via the gwc-facepile classes.
func Facepile(parsePresence *localfirst.PresenceSet) ui.Node {
	if parsePresence == nil {
		return html.Div(html.Props{Class: "gwc-facepile"})
	}

	parsePeers := parsePresence.Live()
	parseFaces := make([]ui.Node, 0, len(parsePeers))
	for _, parsePeer := range parsePeers {
		parseFaces = append(parseFaces, html.Tag("li", html.Props{
			Class: "gwc-facepile-face",
			Title: parsePeer.State,
			Raw:   map[string]any{"data-client": parsePeer.ClientID},
		}, html.Text(parsePeer.ClientID)))
	}

	return html.Div(html.Props{
		Class: "gwc-facepile",
		Role:  "group",
		Aria:  map[string]string{"label": presenceLabel(len(parsePeers))},
	},
		html.Tag("ul", html.Props{Class: "gwc-facepile-faces"}, parseFaces...),
	)
}

// presenceLabel renders an accessible summary of how many peers are present.
func presenceLabel(parseCount int) string {
	switch parseCount {
	case 0:
		return "no one else here"
	case 1:
		return "1 person here"
	default:
		return itoa(parseCount) + " people here"
	}
}

// itoa is a tiny dependency-free integer formatter for the small counts a facepile shows.
func itoa(parseN int) string {
	if parseN == 0 {
		return "0"
	}
	parseDigits := []byte{}
	for parseN > 0 {
		parseDigits = append([]byte{byte('0' + parseN%10)}, parseDigits...)
		parseN /= 10
	}
	return string(parseDigits)
}
