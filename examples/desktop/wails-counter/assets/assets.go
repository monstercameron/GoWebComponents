package assets

import "embed"

// Files contains the packaged frontend and local runtime assets.
//
//go:embed all:dist
var Files embed.FS
