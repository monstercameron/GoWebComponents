// Package design is Atlas Commerce OS's design system, authored entirely in Go on
// top of the repo's typed-CSS package (github.com/monstercameron/GoWebComponents/v5/css).
// There is no Tailwind build step, no .css file, and no CDN: every rule in the
// running app is a Go value that the css package folds into a hashed class and
// emits through its Sink — the in-memory buffer on native (harvested into the SSR
// <style> block) and a managed <style> element on wasm. Because both lanes go
// through the same authoring surface, the design system is written exactly once.
//
// # Read this file first if you are learning v5
//
// Atlas is the reference example, so this package is deliberately written to be
// read. The rationale comments are the point: they say what failure each rule
// prevents. If you are copying this into a new app, copy the *shape* — tokens,
// then base, then a small closed set of primitives — not the hex values.
//
// # The visual direction: "Dock Manifest"
//
// Atlas is warehouse-aware commerce. Its real subject is whether a thing can be
// where you need it, when you need it, and its vocabulary is hub codes, freight
// lanes, promise dates, discrepancies, closeouts and SKUs. Freight paperwork —
// dock tags, manifests, lane placards — is therefore the native visual language:
// light industrial paper and ink, one heavy signature device (the lane placard),
// and everything else quiet.
//
// What this replaced, and why the replacement is shaped the way it is:
//
//   - The old look was near-black rounded boxes nested four deep, all at the same
//     visual weight, with three unrelated accent hues used ornamentally (an orange
//     eyebrow, a teal badge, a yellow button). Nothing was more important than
//     anything else, so nothing read as important at all.
//   - So: exactly one flat surface primitive and NO nested-card primitive (see
//     [Surface]); separation comes from a hairline rule or from space.
//   - So: the three saturated hues are status semantics only, and the API names
//     them so ornamental use reads wrong at the call site (see [StatusException],
//     [PrimaryActionOnly]).
//   - So: hierarchy is carried by type role and weight, not by another box.
//
// # The three type roles ARE the information architecture
//
// [Display], [Prose] and [Data] are not three fonts, they are three claims about
// what a string IS:
//
//   - [Data] (mono) — a machine fact: SKU, hub code, ETA, promise date, quantity,
//     price, status code, any id. Mono is not decoration here. It gives fixed
//     advance width, so a column of SKUs or quantities aligns without a table, and
//     it makes a transposed digit visible. Combined with tabular figures it means
//     "1,041" under "1,042" differs in exactly one glyph slot.
//   - [Prose] (proportional) — a sentence. Anything a human wrote to be read.
//   - [Display] (condensed, heavy, uppercase) — a name for a region of the page:
//     page titles, section titles, eyebrows, button labels, table headers.
//
// The rule to hold: every machine fact is [Data] and every sentence is [Prose].
// When those two get mixed, a table stops being scannable and a paragraph starts
// looking like a log file. There is no fourth role, and there is no "just make it
// bold" — reach for a role and a step from [TypeStep].
//
// # How to use it
//
// Call [Install] exactly once at startup, in both lanes, before the first render.
// It emits the :root token block, the dark-mode override, the framework reset and
// the accessibility floor. Everything else is a *bundle* — a []css.Rule that you
// fold into a class:
//
//	design.Install()
//
//	html.Div(html.Props{Class: design.Class(design.Surface())},
//	    html.Div(html.Props{Class: design.Class(design.Eyebrow())}, "RECEIVING"),
//	    html.H2(html.Props{Class: design.Class(design.Display(design.StepSubhead))}, "Open discrepancies"),
//	    html.Span(html.Props{Class: design.Class(design.Data(design.StepFine))}, "SKU-40192"),
//	)
//
// Bundles compose left-to-right, and later bundles win on conflicting properties,
// because [Class] concatenates them into one rule-set and the css package resolves
// a repeated property last-write-wins within a scope:
//
//	design.Class(design.Prose(design.StepBase), design.Data(design.StepBase)) // ends up mono
//
// That is why primitives return []css.Rule rather than a pre-folded class name: two
// class names on one element would resolve by stylesheet emission order, which is
// not something a caller can see or reason about. One folded class has no such
// ambiguity.
//
// # Why bundles are package-level vars
//
// [Class] is called inside render loops. css.New memoizes on the canonical
// serialization of the rule-set, so folding a bundle a second time is a map
// lookup — but building the rule slice again is not free. The bundles are
// therefore built once at package init and returned by value, and every one is
// length-clipped (see clip) so a caller who writes append(design.Surface(), x)
// gets a copy instead of silently clobbering the shared bundle for the whole
// process. Treat every returned slice as read-only anyway.
//
// Emission stays lazy: nothing is written to the Sink until a bundle is folded (or
// [Install] runs). That is what makes the package survive css.Reset() in tests —
// an init-time fold would hand out class names whose CSS had been thrown away.
//
// [Class] still canonicalizes on every call, which is a few microseconds. For a call
// site in a genuinely hot loop — a table cell rendered a thousand times — hoist the
// folded STRING at the caller, which turns the per-row cost into a field read:
//
//	var skuCellClass = design.Class(design.NumericCell())   // in the calling package
//
// # Three cascade traps this package works around
//
// All three cost real debugging time, so they are called out where they occur:
//
//  1. Within one folded class, blocks are emitted in *sorted selector order*, not
//     source order (see canonicalize in css/rule.go). You cannot break a tie by
//     writing one rule after another. [Table] wins its zebra-vs-hover tie on
//     specificity instead.
//  2. Declarations within one block are also sorted, by property name. That makes a
//     shorthand-then-longhand pair (border, then border-top in [Divider]) work
//     reliably, and it makes border-bottom-beats-border-top impossible in one block.
//  3. A descendant rule inside a bundle (& tbody td, 0-1-2) outranks a bare class
//     applied to that descendant (.numeric-cell, 0-1-0). The cell modifiers
//     [NumericCell], [ProseCell] and [CellMeta] therefore emit through a
//     specificity-doubling variant (&& -> .c-x.c-x, 0-2-0), pinned by
//     TestCellModifiersOutrankTable.
//
// # Two upstream bugs this package found
//
// Both were found by asserting on emitted output rather than by reading the API, which
// is the argument for the tests in this package being shaped the way they are:
//
//   - css.Transition with a multi-property css.TransitionProperty (PropColors) emits an
//     invalid shorthand where only the last property gets the timing. See withMotion.
//   - html.Props{TabIndex: 0} renders no attribute at all, so a scroll region declared
//     that way is unreachable by keyboard. See [TableScroll].
//
// # The catalog is a manifest, not a card grid
//
// The absence of a Card (below) needs a positive answer for the one surface that is
// neither a queue nor a document: a product catalog. Without one, /shop gets built out
// of Surface + Cluster and comes out as two dozen bordered boxes at identical weight —
// the disease, repainted. [Catalog] is that answer: a list of full-width manifest lines
// on a shared CSS grid, with a labelled column header above it, a thumbnail column, and
// availability promoted to its own column because availability is the product's thesis.
//
// It is a list rather than a <table> on purpose (a buyer reads across a row, an operator
// reads down a column, and only a list can restack its tracks on a phone), and it is
// full-width rows rather than a grid because four products in a three-column grid is an
// orphan row with two holes and no fix. The whole argument is in catalog.go; read it
// before changing the shape, because the shape is the argument.
//
// # Theming, and the three themes
//
// Every color, font stack and the rail width are CSS custom properties emitted
// into :root, so a theme overrides values without regenerating a single class. The
// dark theme is nothing but a second :root block inside a prefers-color-scheme
// media query that rewrites the same names. No primitive knows a theme exists.
//
// PRINT IS THE THIRD :ROOT BLOCK. Atlas's whole metaphor is printed paperwork, so a
// receipt, a transfer and a purchase order print as the documents they imitate: the rail
// leaves, chrome leaves, tokens become ink on white, a table header repeats on every page
// and a row does not shear across the fold. It costs one more token block plus a handful
// of per-primitive overrides precisely BECAUSE theming is token-shaped. See print.go —
// including the trap that browsers do not print background-color, which turns every
// filled element (the exception chip, the placard) invisible unless it is re-expressed as
// an outline.
//
// # Contrast is measured, not eyeballed
//
// Token POLARITY between themes is not legibility. contrast_test.go computes WCAG 2.1
// ratios in Go for every (foreground, background) pair the primitives actually paint, in
// all three themes, and it found three real failures that review had not: a 1.36:1 dark
// primary button, a 1.50:1 input border, and a 2.11:1 focus ring inside the light
// placard. Each was fixed with a token or a scoped rule, never with a lowered threshold.
// Adding a color token now fails the build until it is measured.
//
// This is also why the token names are semantic rather than literal: --atlas-ink
// means "the color you write with", not "black". In dark mode ink becomes light and
// paper becomes dark, and every primitive stays correct without being touched.
//
// Do NOT reintroduce what the old example-shell.css did: a hard `color-scheme: dark`
// plus `!important` gradients. That locked the app to a dark rendering while the
// app itself reported THEME: light, because the override could not be beaten by
// anything the app emitted. [Install] declares `color-scheme: light dark` (both are
// supported, follow the user) and this package emits no !important anywhere.
//
// # What is deliberately absent
//
// A design system is defined as much by what it refuses to provide. There is no
// Card, Panel, SurfaceInner or nested-surface primitive; no card GRID (see [Catalog]);
// no free-form color argument anywhere; no fourth button; no shadow scale; no radius
// above 2px; no linear gradient; no webfont. Each absence is documented at the place a
// caller would go looking for it.
//
// The two gradients that do exist are both perforations — the placard's tear edge and the
// rail's punched trailing edge — and both are radial-gradients tiling a single disc.
// TestNoThemeLockRegression matches every gradient function by name so a third one has to
// be argued for.
//
// # The frame participates
//
// One more absence was closed rather than kept: the console rail used to be a generic
// left rail, which meant the largest persistent element on every internal screen carried
// none of the identity. It now has a perforated trailing edge and a column of mono route
// codes, so the frame reads as a placard column — see the section header in shell.go,
// including what came OFF to pay for it (the rail's rounded pill items).
package design
