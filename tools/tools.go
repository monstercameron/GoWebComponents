//go:build tools

// Package tools pins dependencies that are executed by root-module tooling but
// are not imported by ordinary root packages.
package tools

import (
	// tools/gwc launches tools/livereload by file path from the root module.
	_ "github.com/fsnotify/fsnotify"
)
