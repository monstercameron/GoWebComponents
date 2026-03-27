package runtime2

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"testing"
)

// TestRuntime2PackageDoesNotImportCurrentRuntimePackage verifies the package boundary stays independent from internal/runtime.
func TestRuntime2PackageDoesNotImportCurrentRuntimePackage(parseT *testing.T) {
	parseFileSet := token.NewFileSet()
	parsePackages, parseErr := parser.ParseDir(parseFileSet, ".", nil, parser.ImportsOnly)
	if parseErr != nil {
		parseT.Fatalf("parse runtime2 package: %v", parseErr)
	}
	parsePackage, parseOk := parsePackages["runtime2"]
	if !parseOk {
		parseT.Fatal("expected runtime2 package")
	}
	for parsePath, parseFile := range parsePackage.Files {
		parseBase := filepath.Base(parsePath)
		if filepath.Ext(parseBase) != ".go" {
			continue
		}
		for _, parseImport := range parseFile.Imports {
			parseImportPath, parseErr := strconv.Unquote(parseImport.Path.Value)
			if parseErr != nil {
				parseT.Fatalf("unquote import path from %s: %v", parseBase, parseErr)
			}
			if parseImportPath == "github.com/monstercameron/GoWebComponents/internal/runtime" {
				parseT.Fatalf("expected runtime2 package to avoid internal/runtime import in %s", parseBase)
			}
		}
	}
}
