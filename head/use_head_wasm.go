//go:build js && wasm

package head

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// UseHead applies a Document's title and head tags (description, canonical, robots, OpenGraph,
// Twitter) to the LIVE document on the client, re-applying whenever any of those fields change. It
// is the client-side complement to RenderToString (which is server-only): call it from your root
// component so client-side navigation updates the title and social/SEO meta, not just the title.
// On native/SSR builds it is a no-op (see use_head_native.go).
func UseHead(parseDoc Document) {
	ui.UseEffect(func() func() {
		applyHead(parseDoc)
		return nil
	},
		parseDoc.Metadata.Title, parseDoc.Metadata.Description, parseDoc.Metadata.CanonicalURL,
		parseDoc.Robots, parseDoc.Social.Type, parseDoc.Social.Title, parseDoc.Social.Description,
		parseDoc.Social.ImageURL, parseDoc.Social.URL, parseDoc.Social.TwitterCard,
	)
}

// applyHead writes the document title and head tags. Empty fields are skipped (an existing tag is
// left untouched rather than blanked).
func applyHead(parseDoc Document) {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	if parseDoc.Metadata.Title != "" {
		parseDocument.Set("title", parseDoc.Metadata.Title)
	}
	upsertHeadTag(parseDocument, "meta", "name", "description", "content", parseDoc.Metadata.Description)
	upsertHeadTag(parseDocument, "meta", "name", "robots", "content", parseDoc.Robots)
	upsertHeadTag(parseDocument, "link", "rel", "canonical", "href", parseDoc.Metadata.CanonicalURL)
	upsertHeadTag(parseDocument, "meta", "property", "og:type", "content", parseDoc.Social.Type)
	upsertHeadTag(parseDocument, "meta", "property", "og:title", "content", parseDoc.Social.Title)
	upsertHeadTag(parseDocument, "meta", "property", "og:description", "content", parseDoc.Social.Description)
	upsertHeadTag(parseDocument, "meta", "property", "og:image", "content", parseDoc.Social.ImageURL)
	upsertHeadTag(parseDocument, "meta", "property", "og:url", "content", parseDoc.Social.URL)
	upsertHeadTag(parseDocument, "meta", "name", "twitter:card", "content", parseDoc.Social.TwitterCard)
}

// upsertHeadTag finds <parseTag parseSelectorAttr="parseSelectorVal"> in the document and sets
// parseValueAttr to parseValue, creating the element in <head> if it does not exist. A blank value
// is a no-op (it never blanks an existing tag). The selector attribute/value are framework-owned
// constants (never user input), so the querySelector string is safe.
func upsertHeadTag(parseDocument js.Value, parseTag, parseSelectorAttr, parseSelectorVal, parseValueAttr, parseValue string) {
	if parseValue == "" {
		return
	}
	parseSelector := parseTag + `[` + parseSelectorAttr + `="` + parseSelectorVal + `"]`
	parseExisting := parseDocument.Call("querySelector", parseSelector)
	if parseExisting.Truthy() {
		parseExisting.Call("setAttribute", parseValueAttr, parseValue)
		return
	}
	parseEl := parseDocument.Call("createElement", parseTag)
	parseEl.Call("setAttribute", parseSelectorAttr, parseSelectorVal)
	parseEl.Call("setAttribute", parseValueAttr, parseValue)
	if parseHead := parseDocument.Get("head"); parseHead.Truthy() {
		parseHead.Call("appendChild", parseEl)
	}
}
