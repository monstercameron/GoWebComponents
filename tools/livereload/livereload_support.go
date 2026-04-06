package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// resetDebounceState resets the debouncing state after a build.
func (parseLrs *LiveReloadServer) resetDebounceState() {
	parseLrs.firstChangeTime = time.Time{}
	parseLrs.changeCount = 0
	if parseLrs.maxDebounceTimer != nil {
		parseLrs.maxDebounceTimer.Stop()
		parseLrs.maxDebounceTimer = nil
	}
}

// classifyUpdate determines whether the changes require a hot reload or full page reload.
func (parseLrs *LiveReloadServer) classifyUpdate() UpdateClassification {
	parseLrs.mutex.Lock()
	defer parseLrs.mutex.Unlock()

	var parseChangedFiles []string
	for parseFile := range parseLrs.changedFiles {
		parseChangedFiles = append(parseChangedFiles, parseFile)
	}

	parseLrs.changedFiles = make(map[string]time.Time)

	if parseLrs.alwaysHotReload {
		parseReason := "App dev server hot reload"
		if len(parseChangedFiles) > 0 {
			parseReason = "Hot reload for app changes"
		}
		return newUpdateClassification("small", "hot", parseReason, parseChangedFiles)
	}

	if len(parseChangedFiles) == 0 {
		return newUpdateClassification("small", "hot", "No files changed", parseChangedFiles)
	}

	var parseHotReloadReasons []string
	for _, parseFile2 := range parseChangedFiles {
		parseRelPath, _ := filepath.Rel(parseLrs.projectRoot, parseFile2)
		parseRelPath = filepath.ToSlash(parseRelPath)

		if strings.Contains(parseRelPath, "main.go") {
			return newUpdateClassification("big", "full", "Main function or entry point changed", parseChangedFiles)
		}

		if strings.Contains(parseRelPath, "fiber/fiber.go") ||
			strings.Contains(parseRelPath, "fiber/hooks.go") ||
			strings.Contains(parseRelPath, "fiber/types.go") ||
			strings.Contains(parseRelPath, "fiber/state_management.go") {
			return newUpdateClassification("big", "full", "Core fiber system changed", parseChangedFiles)
		}

		if strings.Contains(parseRelPath, "go.mod") || strings.Contains(parseRelPath, "go.sum") {
			return newUpdateClassification("big", "full", "Package dependencies changed", parseChangedFiles)
		}

		if strings.Contains(parseRelPath, "examples/") {
			parseHotReloadReasons = append(parseHotReloadReasons, "example components")
		}
		if strings.Contains(parseRelPath, "website/") {
			parseHotReloadReasons = append(parseHotReloadReasons, "website components")
		}
	}

	parseUniqueReasons := make(map[string]bool)
	var parseFinalReasons []string
	for _, parseReason2 := range parseHotReloadReasons {
		if !parseUniqueReasons[parseReason2] {
			parseUniqueReasons[parseReason2] = true
			parseFinalReasons = append(parseFinalReasons, parseReason2)
		}
	}

	if len(parseFinalReasons) > 0 {
		return newUpdateClassification("small", "hot", "UI changes: "+strings.Join(parseFinalReasons, ", "), parseChangedFiles)
	}

	return newUpdateClassification("big", "full", "Logic changes detected, using full reload for safety", parseChangedFiles)
}

func (parseLrs *LiveReloadServer) requestStateSnapshot() {
	parseLrs.broadcastMessage(MessageTypeStateExport, map[string]string{
		"reason": "hot_reload",
	})
}

func (parseLrs *LiveReloadServer) takePendingStateSnapshot() string {
	parseLrs.stateSnapshotMu.Lock()
	defer parseLrs.stateSnapshotMu.Unlock()
	parseSnapshot := parseLrs.pendingStateSnapshot
	parseLrs.pendingStateSnapshot = ""
	return parseSnapshot
}

func (parseLrs *LiveReloadServer) clearPendingStateSnapshot() {
	parseLrs.stateSnapshotMu.Lock()
	parseLrs.pendingStateSnapshot = ""
	parseLrs.stateSnapshotMu.Unlock()
}

func (parseLrs *LiveReloadServer) buildChangedComponentManifest(parseClassification UpdateClassification) (*ChangedComponentManifest, error) {
	parseManifest := &ChangedComponentManifest{
		GeneratedAt:  time.Now(),
		ReloadType:   parseClassification.ReloadType,
		Reason:       parseClassification.Reason,
		ChangedFiles: make([]string, 0, len(parseClassification.ChangedFiles)),
	}

	parseComponentsByQualifiedName := make(map[string]ChangedComponent)
	for _, parseFile := range parseClassification.ChangedFiles {
		parseRelFile := parseFile
		if parseRelative, parseErr := filepath.Rel(parseLrs.watchRoot, parseFile); parseErr == nil {
			parseRelFile = filepath.ToSlash(parseRelative)
		}
		parseManifest.ChangedFiles = append(parseManifest.ChangedFiles, parseRelFile)

		parseComponents, parseErr2 := parseLrs.extractChangedComponents(parseFile)
		if parseErr2 != nil {
			return nil, parseErr2
		}
		for _, parseComponent := range parseComponents {
			parseComponentsByQualifiedName[parseComponent.QualifiedName] = parseComponent
		}
	}

	if len(parseComponentsByQualifiedName) > 0 {
		parseQualifiedNames := make([]string, 0, len(parseComponentsByQualifiedName))
		for parseQualifiedName := range parseComponentsByQualifiedName {
			parseQualifiedNames = append(parseQualifiedNames, parseQualifiedName)
		}
		sort.Strings(parseQualifiedNames)
		parseManifest.Components = make([]ChangedComponent, 0, len(parseQualifiedNames))
		for _, parseQualifiedName2 := range parseQualifiedNames {
			parseManifest.Components = append(parseManifest.Components, parseComponentsByQualifiedName[parseQualifiedName2])
		}
	}

	return parseManifest, nil
}

func isComponentValueSpec(parseSpec *ast.ValueSpec, parseIndex int) bool {
	if parseSpec == nil {
		return false
	}
	if parseFuncType, parseOk := parseSpec.Type.(*ast.FuncType); parseOk {
		return returnsComponentNode(parseFuncType)
	}
	if parseIndex >= len(parseSpec.Values) {
		return false
	}
	parseFuncLiteral, parseOk2 := parseSpec.Values[parseIndex].(*ast.FuncLit)
	if !parseOk2 {
		return false
	}
	return returnsComponentNode(parseFuncLiteral.Type)
}

func returnsComponentNode(parseFuncType *ast.FuncType) bool {
	if parseFuncType == nil || parseFuncType.Results == nil || len(parseFuncType.Results.List) != 1 {
		return false
	}
	return isComponentResultExpr(parseFuncType.Results.List[0].Type)
}

func isComponentResultExpr(parseExpr ast.Expr) bool {
	switch parseTyped := parseExpr.(type) {
	case *ast.Ident:
		return parseTyped.Name == "Node" || parseTyped.Name == "Element"
	case *ast.SelectorExpr:
		parsePackageIdent, parseOk := parseTyped.X.(*ast.Ident)
		if !parseOk {
			return false
		}
		return (parsePackageIdent.Name == "ui" && (parseTyped.Sel.Name == "Node" || parseTyped.Sel.Name == "Element")) ||
			(parsePackageIdent.Name == "runtime" && parseTyped.Sel.Name == "Element")
	case *ast.StarExpr:
		return isRuntimeElementExpr(parseTyped.X)
	default:
		return false
	}
}

func isRuntimeElementExpr(parseExpr ast.Expr) bool {
	switch parseTyped := parseExpr.(type) {
	case *ast.Ident:
		return parseTyped.Name == "Element"
	case *ast.SelectorExpr:
		parsePackageIdent, parseOk := parseTyped.X.(*ast.Ident)
		if !parseOk {
			return false
		}
		return (parsePackageIdent.Name == "ui" || parsePackageIdent.Name == "runtime") && parseTyped.Sel.Name == "Element"
	default:
		return false
	}
}

func qualifyComponentName(parsePackagePath string, parseName string) string {
	if strings.TrimSpace(parsePackagePath) == "" {
		return parseName
	}
	return parsePackagePath + "." + parseName
}

func resolvePackagePath(parseModulePath string, parseWatchRoot string, parseFilePath string) string {
	parseDirectory := filepath.Dir(parseFilePath)
	parseRelDirectory, parseErr := filepath.Rel(parseWatchRoot, parseDirectory)
	if parseErr != nil || parseRelDirectory == "." {
		return parseModulePath
	}
	parseRelDirectory = filepath.ToSlash(parseRelDirectory)
	if strings.TrimSpace(parseModulePath) == "" {
		return parseRelDirectory
	}
	return parseModulePath + "/" + parseRelDirectory
}

func resolveModulePath(parseRoot string) string {
	parseGoModPath := filepath.Join(parseRoot, "go.mod")
	parseContent, parseErr := os.ReadFile(parseGoModPath)
	if parseErr != nil {
		return ""
	}
	for _, parseLine := range strings.Split(string(parseContent), "\n") {
		parseTrimmed := strings.TrimSpace(parseLine)
		if strings.HasPrefix(parseTrimmed, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(parseTrimmed, "module "))
		}
	}
	return ""
}

func (parseLrs *LiveReloadServer) writeChangedComponentManifest(parseManifest *ChangedComponentManifest) error {
	if parseManifest == nil || strings.TrimSpace(parseLrs.manifestPath) == "" {
		return nil
	}
	if parseErr := os.MkdirAll(filepath.Dir(parseLrs.manifestPath), 0o755); parseErr != nil {
		return fmt.Errorf("create manifest dir: %w", parseErr)
	}
	parseData, parseErr2 := json.MarshalIndent(parseManifest, "", "  ")
	if parseErr2 != nil {
		return fmt.Errorf("marshal manifest: %w", parseErr2)
	}
	if parseErr3 := os.WriteFile(parseLrs.manifestPath, parseData, 0o644); parseErr3 != nil {
		return fmt.Errorf("write manifest: %w", parseErr3)
	}
	return nil
}
