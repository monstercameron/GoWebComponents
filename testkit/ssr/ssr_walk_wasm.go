//go:build js && wasm
// +build js,wasm

package ssr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall/js"
)

func collectStaticExportFiles(root string, rel string, export *StaticExport) error {
	dirPath := root
	if rel != "" {
		dirPath = filepath.Join(root, rel)
	}
	names, err := nodeReadDirNames(dirPath)
	if err != nil {
		return err
	}
	for _, name := range names {
		childRel := name
		if rel != "" {
			childRel = filepath.Join(rel, name)
		}
		path := filepath.Join(root, childRel)
		isDir, dirErr := nodeIsDir(path)
		if dirErr != nil {
			return dirErr
		}
		if isDir {
			if err := collectStaticExportFiles(root, childRel, export); err != nil {
				return err
			}
			continue
		}
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

func nodeReadDirNames(path string) ([]string, error) {
	fs, err := nodeFS()
	if err != nil {
		return nil, err
	}
	var (
		result  js.Value
		readErr error
	)
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				readErr = fmt.Errorf("node readdirSync %q: %v", path, recovered)
			}
		}()
		result = fs.Call("readdirSync", path)
	}()
	if readErr != nil {
		return nil, readErr
	}
	length := result.Length()
	names := make([]string, 0, length)
	for index := 0; index < length; index++ {
		names = append(names, result.Index(index).String())
	}
	return names, nil
}

func nodeIsDir(path string) (bool, error) {
	fs, err := nodeFS()
	if err != nil {
		return false, err
	}
	var (
		statValue js.Value
		statErr   error
	)
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				statErr = fmt.Errorf("node statSync %q: %v", path, recovered)
			}
		}()
		statValue = fs.Call("statSync", path)
	}()
	if statErr != nil {
		return false, statErr
	}
	return statValue.Call("isDirectory").Bool(), nil
}

func nodeFS() (js.Value, error) {
	global := js.Global()
	require := global.Get("require")
	if require.Type() == js.TypeFunction {
		fs := require.Invoke("fs")
		if fs.IsUndefined() || fs.IsNull() {
			return js.Undefined(), fmt.Errorf("node fs module is unavailable")
		}
		return fs, nil
	}
	fs := global.Get("fs")
	if fs.IsUndefined() || fs.IsNull() {
		return js.Undefined(), fmt.Errorf("node require is unavailable")
	}
	return fs, nil
}
