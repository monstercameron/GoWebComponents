package projection

import (
	"errors"
	"fmt"
)

// SSR bootstrap — plan item P3.11.
//
// Criterion: a server-rendered route is Ready() on first render.
//
// Without this, a server-rendered page paints its markup instantly and then the
// client mounts, finds an empty projection, renders a loading state or an empty
// list, and only fills in after the first domain round trip. The user sees
// content, then sees it vanish, then sees it return. Server rendering made the
// first paint fast and the experience worse.
//
// The fix is to ship the projection's initial contents alongside the markup, so
// the client's first render has the same data the server rendered from.
//
// Ready() is the part that is easy to get wrong. It is NOT Len() > 0: a
// projection legitimately holds zero rows, and a UI that treats empty as
// "still loading" shows a spinner forever on a route with no results, while one
// that treats it as loaded shows "no results found" before anything has been
// fetched. Those are different states and this distinguishes them.

// Bootstrap is a projection's initial contents, serialized for transport with
// server-rendered markup.
//
// The field names are short because this is embedded in HTML and paid for on
// every page load. The row order is the published order; keys and payloads are
// parallel arrays rather than a slice of structs for the same reason — it
// avoids repeating two field names per row.
type Bootstrap struct {
	// Keys are the row keys in published order.
	Keys []Key `json:"k,omitempty"`
	// Payloads are the encoded rows, parallel to Keys.
	Payloads [][]byte `json:"p,omitempty"`
	// Complete reports whether these rows are the whole projection.
	//
	// False means the server sent a prefix — the first screenful of a large
	// table — and the client should expect more. A client that assumed
	// completeness would stop paginating at the fold.
	Complete bool `json:"c,omitempty"`
}

// Len reports how many rows the bootstrap carries.
func (parseBootstrap Bootstrap) Len() int {
	return len(parseBootstrap.Keys)
}

// Validate reports whether the bootstrap is internally consistent.
//
// Parallel arrays are compact and can desynchronize, which would pair a key
// with another row's payload — data corruption that renders without error. It
// is checked once at hydration rather than trusted.
func (parseBootstrap Bootstrap) Validate() error {
	if len(parseBootstrap.Keys) != len(parseBootstrap.Payloads) {
		return fmt.Errorf("projection: bootstrap has %d keys and %d payloads",
			len(parseBootstrap.Keys), len(parseBootstrap.Payloads))
	}
	parseSeen := make(map[Key]bool, len(parseBootstrap.Keys))
	for parseIndex, parseKey := range parseBootstrap.Keys {
		if parseKey == "" {
			return fmt.Errorf("projection: bootstrap row %d has no key", parseIndex)
		}
		if parseSeen[parseKey] {
			return fmt.Errorf("projection: bootstrap key %q appears more than once", parseKey)
		}
		parseSeen[parseKey] = true
	}
	return nil
}

// BuildBootstrap builds a bootstrap from rows the server already has.
//
// Encoding is the caller's, matching the Decoder the client will hydrate with.
// A mismatch between them is the one failure this cannot catch, and it surfaces
// immediately at hydration rather than later.
func BuildBootstrap[T any](parseKeys []Key, parseRows []T, parseEncode func(T) ([]byte, error), parseComplete bool) (Bootstrap, error) {
	if parseEncode == nil {
		return Bootstrap{}, errors.New("projection: an encoder is required")
	}
	if len(parseKeys) != len(parseRows) {
		return Bootstrap{}, fmt.Errorf("projection: %d keys for %d rows", len(parseKeys), len(parseRows))
	}

	parseBootstrap := Bootstrap{
		Keys:     make([]Key, 0, len(parseKeys)),
		Payloads: make([][]byte, 0, len(parseRows)),
		Complete: parseComplete,
	}
	for parseIndex, parseRow := range parseRows {
		if parseKeys[parseIndex] == "" {
			return Bootstrap{}, fmt.Errorf("projection: row %d has no key", parseIndex)
		}
		parsePayload, parseErr := parseEncode(parseRow)
		if parseErr != nil {
			return Bootstrap{}, fmt.Errorf("projection: encoding row %q: %w", parseKeys[parseIndex], parseErr)
		}
		parseBootstrap.Keys = append(parseBootstrap.Keys, parseKeys[parseIndex])
		parseBootstrap.Payloads = append(parseBootstrap.Payloads, parsePayload)
	}
	return parseBootstrap, nil
}

// Hydrate seeds a projection from a server bootstrap.
//
// After it returns, Ready() is true — including for a bootstrap carrying zero
// rows, which is the case that makes Ready() worth having at all: a route with
// no results is loaded, not loading.
//
// Hydrating a projection that already holds rows is refused. It would either
// duplicate keys or silently drop the bootstrap, and both are worse than an
// error at a point where the caller can still do something about it.
func (parseProjection *Projection[T]) Hydrate(parseBootstrap Bootstrap) error {
	if parseProjection == nil {
		return errors.New("projection: projection is nil")
	}
	if parseProjection.ready {
		return errors.New("projection: already hydrated; a second bootstrap would duplicate or discard rows")
	}
	if len(parseProjection.entries) > 0 {
		return errors.New("projection: cannot hydrate a projection that already holds rows")
	}
	if parseErr := parseBootstrap.Validate(); parseErr != nil {
		return parseErr
	}

	// Built as ops rather than written directly so hydration goes through the
	// same residency cap, decoding, and ordering as a live delta. A second path
	// into the same state is a second place for them to diverge.
	parseOps := make([]Op, 0, parseBootstrap.Len())
	var parseAnchor Key
	for parseIndex, parseKey := range parseBootstrap.Keys {
		parseOps = append(parseOps, Op{
			Kind:     OpInsert,
			Key:      parseKey,
			AfterKey: parseAnchor,
			Payload:  parseBootstrap.Payloads[parseIndex],
		})
		parseAnchor = parseKey
	}
	if parseErr := parseProjection.Apply(parseOps); parseErr != nil {
		return fmt.Errorf("projection: hydrating: %w", parseErr)
	}

	parseProjection.ready = true
	parseProjection.complete = parseBootstrap.Complete
	return nil
}

// MarkReady declares a projection loaded without a bootstrap.
//
// For a client-rendered route that has finished its first fetch. Without it,
// only server-rendered routes could ever be Ready, and a UI branching on
// Ready() would show a spinner forever on every other route.
func (parseProjection *Projection[T]) MarkReady(parseComplete bool) {
	if parseProjection == nil {
		return
	}
	parseProjection.ready = true
	parseProjection.complete = parseComplete
}

// Ready reports whether the projection holds its initial contents.
//
// The distinction this exists for: Ready() is NOT Len() > 0. A projection with
// zero rows may be loaded — a search with no matches, a new account with no
// transactions — and a UI that reads emptiness as loading shows a spinner
// forever on exactly those routes. It may equally be unloaded, and a UI that
// reads emptiness as loaded shows "nothing here" before anything was fetched.
func (parseProjection *Projection[T]) Ready() bool {
	if parseProjection == nil {
		return false
	}
	return parseProjection.ready
}

// Complete reports whether the projection holds the whole dataset rather than a
// server-sent prefix.
//
// Distinct from Ready: a route can be ready to render — enough rows for the
// first screen — while more remain. A client that conflated them would stop
// paginating at the fold.
func (parseProjection *Projection[T]) Complete() bool {
	if parseProjection == nil {
		return false
	}
	return parseProjection.complete
}
