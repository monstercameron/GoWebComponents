// Package thirdpartyjs is a worked example of integrating a third-party JavaScript library
// behind a TYPED Go bridge using interop.ImportModule, with a pure-Go fallback so the exact same
// code runs on the server (SSR/native) where no JS module loader exists. It wraps the popular
// `@sindresorhus/slugify` ESM module: in the browser it calls the real library; everywhere else
// it transparently falls back to an equivalent Go implementation. This is the F5 pattern —
// dynamic-import a JS module, expose it as a typed Go API, and keep native-stub parity.
package thirdpartyjs

import (
	"context"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// slugifyModuleSpecifier is the ESM URL the browser dynamically imports. Any CDN that serves the
// module as an ES module works; the bridge does not care which.
const slugifyModuleSpecifier = "https://esm.sh/@sindresorhus/slugify@2"

// SlugifyBridge is a typed wrapper over the imported JS module. The zero value is usable: it
// behaves as "module not loaded" and routes through the Go fallback.
type SlugifyBridge struct {
	module interop.Module
	loaded bool
}

// LoadSlugify dynamically imports the slugify module and returns a bridge bound to it. When the
// import is unavailable (native/server, or an offline browser), it returns an UNLOADED bridge
// that still works via the Go fallback — so callers never have to special-case the platform.
func LoadSlugify(parseCtx context.Context) SlugifyBridge {
	parseModule, parseErr := interop.ImportModule(parseCtx, slugifyModuleSpecifier)
	if parseErr != nil {
		return SlugifyBridge{}
	}
	return SlugifyBridge{module: parseModule, loaded: true}
}

// Loaded reports whether the real JS module backs this bridge (false → the Go fallback is used).
func (parseB SlugifyBridge) Loaded() bool { return parseB.loaded }

// Slugify converts text to a URL slug. When the JS module is loaded it calls the library's
// default export; otherwise — and whenever the call fails or returns an unexpected type — it
// falls back to the equivalent Go implementation, guaranteeing the same well-formed slug on
// every platform.
func (parseB SlugifyBridge) Slugify(parseCtx context.Context, parseText string) string {
	if parseB.loaded {
		if parseResult, parseErr := parseB.module.CallDefault(parseCtx, parseText); parseErr == nil {
			if parseSlug, parseOk := parseResult.(string); parseOk && parseSlug != "" {
				return parseSlug
			}
		}
	}
	return fallbackSlugify(parseText)
}

// Dispose releases the imported module's resources, if any. Safe to call on an unloaded bridge.
func (parseB SlugifyBridge) Dispose() error {
	if !parseB.loaded {
		return nil
	}
	return parseB.module.Dispose()
}

// fallbackSlugify is the pure-Go equivalent of the JS library's core behavior: lowercase, then
// collapse every run of non-alphanumeric characters into a single hyphen, trimming hyphens at
// the ends. It is what gives the bridge native parity.
func fallbackSlugify(parseText string) string {
	var parseBuilder strings.Builder
	parsePrevHyphen := false
	for _, parseRune := range strings.ToLower(parseText) {
		switch {
		case parseRune >= 'a' && parseRune <= 'z', parseRune >= '0' && parseRune <= '9':
			parseBuilder.WriteRune(parseRune)
			parsePrevHyphen = false
		default:
			if !parsePrevHyphen && parseBuilder.Len() > 0 {
				parseBuilder.WriteByte('-')
				parsePrevHyphen = true
			}
		}
	}
	return strings.Trim(parseBuilder.String(), "-")
}
