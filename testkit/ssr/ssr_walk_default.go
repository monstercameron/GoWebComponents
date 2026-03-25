//go:build !js || !wasm
// +build !js !wasm

package ssr

import (
	"os"
	"path/filepath"
	"strings"
)

func collectStaticExportFiles(root string, rel string, export *StaticExport) error {
	dirPath := root
	if rel != "" {
		dirPath = filepath.Join(root, rel)
	}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		childRel := name
		if rel != "" {
			childRel = filepath.Join(rel, name)
		}
		if entry.IsDir() {
			if err := collectStaticExportFiles(root, childRel, export); err != nil {
				return err
			}
			continue
		}
		path := filepath.Join(root, childRel)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		relative := filepath.ToSlash(childRel)
		if strings.HasSuffix(relative, ".html") {
			export.HTMLFiles[relative] = Snapshot{HTML: string(data)}
			continue
		}
		if strings.HasPrefix(relative, "bootstrap/") {
			export.Bootstrap[relative] = append([]byte(nil), data...)
		}
	}
	return nil
}
