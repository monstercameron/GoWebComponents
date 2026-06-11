// Package manualdata embeds the reference-manual chapters so the docs site
// wasm app ships them inside the binary instead of fetching files at runtime.
package manualdata

import "embed"

// Chapters holds every numbered reference-manual chapter markdown document.
//
//go:embed *.md
var Chapters embed.FS
