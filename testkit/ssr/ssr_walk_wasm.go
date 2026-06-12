//go:build js && wasm

package ssr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall/js"
)

func collectStaticExportFiles(parseRoot string, parseRel string, parseExport *StaticExport) error {
	parseDirPath := parseRoot
	if parseRel != "" {
		parseDirPath = filepath.Join(parseRoot, parseRel)
	}
	parseNames, parseErr := nodeReadDirNames(parseDirPath)
	if parseErr != nil {
		return parseErr
	}
	for _, parseName := range parseNames {
		parseChildRel := parseName
		if parseRel != "" {
			parseChildRel = filepath.Join(parseRel, parseName)
		}
		parsePath := filepath.Join(parseRoot, parseChildRel)
		isDir, parseDirErr := nodeIsDir(parsePath)
		if parseDirErr != nil {
			return parseDirErr
		}
		if isDir {
			if parseErr2 := collectStaticExportFiles(parseRoot, parseChildRel, parseExport); parseErr2 != nil {
				return parseErr2
			}
			continue
		}
		parseData, parseReadErr := os.ReadFile(parsePath)
		if parseReadErr != nil {
			return parseReadErr
		}
		parseRelative := filepath.ToSlash(parseChildRel)
		if strings.HasSuffix(parseRelative, ".html") {
			parseExport.HTMLFiles[parseRelative] = Snapshot{HTML: string(parseData)}
			continue
		}
		if strings.HasPrefix(parseRelative, "bootstrap/") {
			parseExport.Bootstrap[parseRelative] = append([]byte(nil), parseData...)
		}
	}
	return nil
}

func nodeReadDirNames(parsePath string) ([]string, error) {
	parseFs, parseErr := nodeFS()
	if parseErr != nil {
		return nil, parseErr
	}
	var (
		parseResult  js.Value
		parseReadErr error
	)
	func() {
		defer func() {
			if parseRecovered := recover(); parseRecovered != nil {
				parseReadErr = fmt.Errorf("node readdirSync %q: %v", parsePath, parseRecovered)
			}
		}()
		parseResult = parseFs.Call("readdirSync", parsePath)
	}()
	if parseReadErr != nil {
		return nil, parseReadErr
	}
	parseLength := parseResult.Length()
	parseNames := make([]string, 0, parseLength)
	for parseIndex := 0; parseIndex < parseLength; parseIndex++ {
		parseNames = append(parseNames, parseResult.Index(parseIndex).String())
	}
	return parseNames, nil
}

func nodeIsDir(parsePath string) (bool, error) {
	parseFs, parseErr := nodeFS()
	if parseErr != nil {
		return false, parseErr
	}
	var (
		parseStatValue js.Value
		parseStatErr   error
	)
	func() {
		defer func() {
			if parseRecovered := recover(); parseRecovered != nil {
				parseStatErr = fmt.Errorf("node statSync %q: %v", parsePath, parseRecovered)
			}
		}()
		parseStatValue = parseFs.Call("statSync", parsePath)
	}()
	if parseStatErr != nil {
		return false, parseStatErr
	}
	return parseStatValue.Call("isDirectory").Bool(), nil
}

func nodeFS() (js.Value, error) {
	parseGlobal := js.Global()
	parseRequire := parseGlobal.Get("require")
	if parseRequire.Type() == js.TypeFunction {
		parseFs := parseRequire.Invoke("fs")
		if parseFs.IsUndefined() || parseFs.IsNull() {
			return js.Undefined(), fmt.Errorf("node fs module is unavailable")
		}
		return parseFs, nil
	}
	parseFs2 := parseGlobal.Get("fs")
	if parseFs2.IsUndefined() || parseFs2.IsNull() {
		return js.Undefined(), fmt.Errorf("node require is unavailable")
	}
	return parseFs2, nil
}
