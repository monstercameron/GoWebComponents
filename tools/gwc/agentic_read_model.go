package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type agenticReadConfig struct {
	rootPath string
	json     bool
}

type agenticModelManifest struct {
	OK          bool                    `json:"ok"`
	Root        string                  `json:"root"`
	ModulePath  string                  `json:"modulePath,omitempty"`
	Packages    []agenticPackageRecord  `json:"packages,omitempty"`
	Components  []agenticComponent      `json:"components,omitempty"`
	Symbols     []agenticSymbol         `json:"symbols,omitempty"`
	Edges       []agenticReferenceEdge  `json:"edges,omitempty"`
	Diagnostics []agenticReadDiagnostic `json:"diagnostics,omitempty"`
}

type agenticPackageRecord struct {
	ImportPath string   `json:"importPath,omitempty"`
	Name       string   `json:"name"`
	Dir        string   `json:"dir"`
	Files      []string `json:"files,omitempty"`
}

type agenticComponent struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Package     string   `json:"package,omitempty"`
	PackageName string   `json:"packageName,omitempty"`
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Props       string   `json:"props,omitempty"`
	Hooks       []string `json:"hooks,omitempty"`
	AtomReads   []string `json:"atomReads,omitempty"`
	AtomWrites  []string `json:"atomWrites,omitempty"`
	Events      []string `json:"events,omitempty"`
	Routes      []string `json:"routes,omitempty"`
	Calls       []string `json:"calls,omitempty"`
}

type agenticSymbol struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Kind           string   `json:"kind"`
	Package        string   `json:"package,omitempty"`
	PackageName    string   `json:"packageName,omitempty"`
	File           string   `json:"file"`
	Line           int      `json:"line"`
	Signature      string   `json:"signature,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	Exported       bool     `json:"exported"`
	CapabilityTags []string `json:"capabilityTags,omitempty"`
	References     []string `json:"references,omitempty"`
}

type agenticReferenceEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind,omitempty"`
}

type agenticReadDiagnostic struct {
	Severity string `json:"severity"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
}

type agenticImpactReport struct {
	OK                   bool                    `json:"ok"`
	Symbol               string                  `json:"symbol"`
	Root                 string                  `json:"root,omitempty"`
	DirectDependents     []agenticImpactTarget   `json:"directDependents,omitempty"`
	TransitiveDependents []agenticImpactTarget   `json:"transitiveDependents,omitempty"`
	Tests                []string                `json:"tests,omitempty"`
	Docs                 []string                `json:"docs,omitempty"`
	Diagnostics          []agenticReadDiagnostic `json:"diagnostics,omitempty"`
}

type agenticImpactTarget struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Package  string `json:"package,omitempty"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Distance int    `json:"distance"`
}

type agenticParsedFile struct {
	relPath     string
	absPath     string
	dir         string
	importPath  string
	packageName string
	fileSet     *token.FileSet
	file        *ast.File
}

type agenticDeclAnalysis struct {
	hooks      map[string]struct{}
	atomReads  map[string]struct{}
	atomWrites map[string]struct{}
	events     map[string]struct{}
	routes     map[string]struct{}
	calls      map[string]struct{}
	refs       map[string]struct{}
}

func resolveAgenticReadConfig(parseConfig agenticReadConfig) (agenticReadConfig, error) {
	parseRootPath := strings.TrimSpace(parseConfig.rootPath)
	if parseRootPath == "" {
		parseCwd, parseErr := os.Getwd()
		if parseErr != nil {
			return agenticReadConfig{}, fmt.Errorf("resolve root from cwd: %w", parseErr)
		}
		parseRootPath = parseCwd
	}
	parseAbsRoot, parseErr := filepath.Abs(parseRootPath)
	if parseErr != nil {
		return agenticReadConfig{}, fmt.Errorf("resolve root: %w", parseErr)
	}
	parseInfo, parseErr := os.Stat(parseAbsRoot)
	if parseErr != nil {
		return agenticReadConfig{}, fmt.Errorf("stat root: %w", parseErr)
	}
	if !parseInfo.IsDir() {
		return agenticReadConfig{}, fmt.Errorf("root is not a directory: %s", parseAbsRoot)
	}
	return agenticReadConfig{rootPath: parseAbsRoot, json: parseConfig.json}, nil
}

func buildAgenticModelManifest(parseRootPath string) (agenticModelManifest, error) {
	parseRootPath = strings.TrimSpace(parseRootPath)
	if parseRootPath == "" {
		return agenticModelManifest{}, errors.New("model root is required")
	}
	parseModulePath := readAgenticModulePath(parseRootPath)
	parseFiles, parseDiagnostics, parseErr := parseAgenticGoFiles(parseRootPath, parseModulePath)
	if parseErr != nil {
		return agenticModelManifest{}, parseErr
	}

	parseManifest := agenticModelManifest{
		OK:          len(parseDiagnostics) == 0,
		Root:        parseRootPath,
		ModulePath:  parseModulePath,
		Diagnostics: parseDiagnostics,
	}
	parsePackageFiles := map[string]map[string]struct{}{}
	parsePackageRecords := map[string]agenticPackageRecord{}
	parseSymbolsByName := map[string][]agenticSymbol{}
	parseSymbolRefsByID := map[string][]string{}

	for _, parseFile := range parseFiles {
		parsePkgKey := parseFile.importPath
		if parsePkgKey == "" {
			parsePkgKey = parseFile.packageName + ":" + parseFile.dir
		}
		if _, parseOk := parsePackageFiles[parsePkgKey]; !parseOk {
			parsePackageFiles[parsePkgKey] = map[string]struct{}{}
		}
		parsePackageFiles[parsePkgKey][parseFile.relPath] = struct{}{}
		parsePackageRecords[parsePkgKey] = agenticPackageRecord{
			ImportPath: parseFile.importPath,
			Name:       parseFile.packageName,
			Dir:        parseFile.dir,
		}

		for _, parseDecl := range parseFile.file.Decls {
			switch parseTypedDecl := parseDecl.(type) {
			case *ast.FuncDecl:
				parseSymbol := buildAgenticFuncSymbol(parseFile, parseTypedDecl)
				parseAnalysis := analyzeAgenticDecl(parseTypedDecl.Body)
				parseSymbol.References = sortedAgenticSet(parseAnalysis.refs)
				parseSymbol.CapabilityTags = agenticCapabilityTagsForPackage(parseSymbol.Package)
				parseManifest.Symbols = append(parseManifest.Symbols, parseSymbol)
				parseSymbolsByName[parseSymbol.Name] = append(parseSymbolsByName[parseSymbol.Name], parseSymbol)
				parseSymbolRefsByID[parseSymbol.ID] = parseSymbol.References

				if isAgenticComponentDecl(parseTypedDecl) {
					parseManifest.Components = append(parseManifest.Components, buildAgenticComponent(parseFile, parseTypedDecl, parseAnalysis))
				}
			case *ast.GenDecl:
				for _, parseSpec := range parseTypedDecl.Specs {
					parseSymbol, parseOk := buildAgenticGenSymbol(parseFile, parseTypedDecl, parseSpec)
					if !parseOk {
						continue
					}
					parseSymbol.CapabilityTags = agenticCapabilityTagsForPackage(parseSymbol.Package)
					parseManifest.Symbols = append(parseManifest.Symbols, parseSymbol)
					parseSymbolsByName[parseSymbol.Name] = append(parseSymbolsByName[parseSymbol.Name], parseSymbol)
				}
			}
		}
	}

	for parsePkgKey, parseRecord := range parsePackageRecords {
		parseRecord.Files = sortedAgenticStringSet(parsePackageFiles[parsePkgKey])
		parseManifest.Packages = append(parseManifest.Packages, parseRecord)
	}
	sort.Slice(parseManifest.Packages, func(parseLeft int, parseRight int) bool {
		parseA := firstNonEmpty(parseManifest.Packages[parseLeft].ImportPath, parseManifest.Packages[parseLeft].Dir, parseManifest.Packages[parseLeft].Name)
		parseB := firstNonEmpty(parseManifest.Packages[parseRight].ImportPath, parseManifest.Packages[parseRight].Dir, parseManifest.Packages[parseRight].Name)
		return parseA < parseB
	})

	sort.Slice(parseManifest.Components, func(parseLeft int, parseRight int) bool {
		return parseManifest.Components[parseLeft].ID < parseManifest.Components[parseRight].ID
	})
	sort.Slice(parseManifest.Symbols, func(parseLeft int, parseRight int) bool {
		return parseManifest.Symbols[parseLeft].ID < parseManifest.Symbols[parseRight].ID
	})

	parseEdgeSeen := map[string]struct{}{}
	for parseFromID, parseRefs := range parseSymbolRefsByID {
		for _, parseRef := range parseRefs {
			for _, parseTo := range parseSymbolsByName[parseRef] {
				if parseTo.ID == parseFromID {
					continue
				}
				parseKey := parseFromID + "\x00" + parseTo.ID
				if _, parseExists := parseEdgeSeen[parseKey]; parseExists {
					continue
				}
				parseEdgeSeen[parseKey] = struct{}{}
				parseManifest.Edges = append(parseManifest.Edges, agenticReferenceEdge{
					From: parseFromID,
					To:   parseTo.ID,
					Kind: "static-reference",
				})
			}
		}
	}
	sort.Slice(parseManifest.Edges, func(parseLeft int, parseRight int) bool {
		if parseManifest.Edges[parseLeft].From != parseManifest.Edges[parseRight].From {
			return parseManifest.Edges[parseLeft].From < parseManifest.Edges[parseRight].From
		}
		return parseManifest.Edges[parseLeft].To < parseManifest.Edges[parseRight].To
	})

	return parseManifest, nil
}

func parseAgenticGoFiles(parseRootPath string, parseModulePath string) ([]agenticParsedFile, []agenticReadDiagnostic, error) {
	parseFiles := []agenticParsedFile{}
	parseDiagnostics := []agenticReadDiagnostic{}
	parseWalkErr := filepath.WalkDir(parseRootPath, func(parsePath string, parseEntry fs.DirEntry, parseErr error) error {
		if parseErr != nil {
			return parseErr
		}
		if parseEntry.IsDir() {
			if shouldSkipDoctorAuditDir(parseEntry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(parseEntry.Name()) != ".go" {
			return nil
		}
		parseRelPath, parseRelErr := filepath.Rel(parseRootPath, parsePath)
		if parseRelErr != nil {
			return parseRelErr
		}
		parseRelPath = filepath.ToSlash(parseRelPath)
		parseFileSet := token.NewFileSet()
		parseFile, parseParseErr := parser.ParseFile(parseFileSet, parsePath, nil, parser.ParseComments)
		if parseParseErr != nil {
			parseDiagnostics = append(parseDiagnostics, agenticReadDiagnostic{
				Severity: "error",
				Code:     "parse_go_file",
				Message:  parseParseErr.Error(),
				File:     parseRelPath,
			})
			return nil
		}
		parseDir := filepath.ToSlash(filepath.Dir(parseRelPath))
		if parseDir == "." {
			parseDir = ""
		}
		parseFiles = append(parseFiles, agenticParsedFile{
			relPath:     parseRelPath,
			absPath:     parsePath,
			dir:         parseDir,
			importPath:  agenticImportPathForDir(parseModulePath, parseDir),
			packageName: parseFile.Name.Name,
			fileSet:     parseFileSet,
			file:        parseFile,
		})
		return nil
	})
	if parseWalkErr != nil {
		return nil, nil, fmt.Errorf("walk model root: %w", parseWalkErr)
	}
	sort.Slice(parseFiles, func(parseLeft int, parseRight int) bool {
		return parseFiles[parseLeft].relPath < parseFiles[parseRight].relPath
	})
	return parseFiles, parseDiagnostics, nil
}

func readAgenticModulePath(parseRootPath string) string {
	parseContent, parseErr := os.ReadFile(filepath.Join(parseRootPath, "go.mod"))
	if parseErr != nil {
		return ""
	}
	for _, parseLine := range strings.Split(string(parseContent), "\n") {
		parseFields := strings.Fields(strings.TrimSpace(parseLine))
		if len(parseFields) >= 2 && parseFields[0] == "module" {
			return parseFields[1]
		}
	}
	return ""
}

func agenticImportPathForDir(parseModulePath string, parseDir string) string {
	parseModulePath = strings.TrimSpace(parseModulePath)
	parseDir = strings.Trim(strings.TrimSpace(filepath.ToSlash(parseDir)), "/")
	if parseModulePath == "" {
		return parseDir
	}
	if parseDir == "" || parseDir == "." {
		return parseModulePath
	}
	return parseModulePath + "/" + parseDir
}

func buildAgenticFuncSymbol(parseFile agenticParsedFile, parseDecl *ast.FuncDecl) agenticSymbol {
	parseName := parseDecl.Name.Name
	parseKind := "func"
	if parseDecl.Recv != nil && len(parseDecl.Recv.List) > 0 {
		parseKind = "method"
		parseName = agenticReceiverName(parseDecl.Recv.List[0].Type) + "." + parseName
	}
	return agenticSymbol{
		ID:          agenticSymbolID(parseFile.importPath, parseFile.packageName, parseName),
		Name:        parseName,
		Kind:        parseKind,
		Package:     parseFile.importPath,
		PackageName: parseFile.packageName,
		File:        parseFile.relPath,
		Line:        parseFile.fileSet.Position(parseDecl.Pos()).Line,
		Signature:   "func " + parseDecl.Name.Name + agenticFuncTypeString(parseDecl.Type),
		Summary:     agenticCommentSummary(parseDecl.Doc),
		Exported:    ast.IsExported(parseDecl.Name.Name),
	}
}

func buildAgenticGenSymbol(parseFile agenticParsedFile, parseDecl *ast.GenDecl, parseSpec ast.Spec) (agenticSymbol, bool) {
	var parseName string
	var parseSignature string
	var parseKind string
	var parsePos token.Pos
	var parseComment *ast.CommentGroup
	switch parseTypedSpec := parseSpec.(type) {
	case *ast.TypeSpec:
		parseName = parseTypedSpec.Name.Name
		parseKind = "type"
		parsePos = parseTypedSpec.Pos()
		parseComment = firstAgenticComment(parseTypedSpec.Doc, parseDecl.Doc)
		parseSignature = "type " + parseName
		if parseTypedSpec.Type != nil {
			parseSignature += " " + agenticExprString(parseTypedSpec.Type)
		}
	case *ast.ValueSpec:
		if len(parseTypedSpec.Names) == 0 {
			return agenticSymbol{}, false
		}
		parseKind = strings.ToLower(parseDecl.Tok.String())
		parsePos = parseTypedSpec.Pos()
		parseComment = firstAgenticComment(parseTypedSpec.Doc, parseDecl.Doc)
		parseName = parseTypedSpec.Names[0].Name
		parseNames := make([]string, 0, len(parseTypedSpec.Names))
		for _, parseIdent := range parseTypedSpec.Names {
			parseNames = append(parseNames, parseIdent.Name)
		}
		parseSignature = parseKind + " " + strings.Join(parseNames, ", ")
		if parseTypedSpec.Type != nil {
			parseSignature += " " + agenticExprString(parseTypedSpec.Type)
		}
	default:
		return agenticSymbol{}, false
	}
	return agenticSymbol{
		ID:          agenticSymbolID(parseFile.importPath, parseFile.packageName, parseName),
		Name:        parseName,
		Kind:        parseKind,
		Package:     parseFile.importPath,
		PackageName: parseFile.packageName,
		File:        parseFile.relPath,
		Line:        parseFile.fileSet.Position(parsePos).Line,
		Signature:   parseSignature,
		Summary:     agenticCommentSummary(parseComment),
		Exported:    ast.IsExported(parseName),
	}, true
}

func buildAgenticComponent(parseFile agenticParsedFile, parseDecl *ast.FuncDecl, parseAnalysis agenticDeclAnalysis) agenticComponent {
	return agenticComponent{
		ID:          agenticSymbolID(parseFile.importPath, parseFile.packageName, parseDecl.Name.Name),
		Name:        parseDecl.Name.Name,
		Package:     parseFile.importPath,
		PackageName: parseFile.packageName,
		File:        parseFile.relPath,
		Line:        parseFile.fileSet.Position(parseDecl.Pos()).Line,
		Props:       agenticComponentProps(parseDecl),
		Hooks:       sortedAgenticSet(parseAnalysis.hooks),
		AtomReads:   sortedAgenticSet(parseAnalysis.atomReads),
		AtomWrites:  sortedAgenticSet(parseAnalysis.atomWrites),
		Events:      sortedAgenticSet(parseAnalysis.events),
		Routes:      sortedAgenticSet(parseAnalysis.routes),
		Calls:       sortedAgenticSet(parseAnalysis.calls),
	}
}

func analyzeAgenticDecl(parseNode ast.Node) agenticDeclAnalysis {
	parseAnalysis := agenticDeclAnalysis{
		hooks:      map[string]struct{}{},
		atomReads:  map[string]struct{}{},
		atomWrites: map[string]struct{}{},
		events:     map[string]struct{}{},
		routes:     map[string]struct{}{},
		calls:      map[string]struct{}{},
		refs:       map[string]struct{}{},
	}
	if parseNode == nil {
		return parseAnalysis
	}
	ast.Inspect(parseNode, func(parseCurrent ast.Node) bool {
		switch parseTyped := parseCurrent.(type) {
		case *ast.CallExpr:
			parseName := agenticCallName(parseTyped.Fun)
			if parseName != "" {
				parseAnalysis.calls[parseName] = struct{}{}
				parseAnalysis.refs[agenticLastNameSegment(parseName)] = struct{}{}
				if strings.HasPrefix(agenticLastNameSegment(parseName), "Use") {
					parseAnalysis.hooks[parseName] = struct{}{}
				}
				switch agenticLastNameSegment(parseName) {
				case "UseAtom", "UseDerived", "UseSelector", "UseComputed":
					if parseAtom := agenticFirstStringArg(parseTyped.Args); parseAtom != "" {
						parseAnalysis.atomReads[parseAtom] = struct{}{}
					}
				case "Set", "SetValue", "Update":
					parseAnalysis.atomWrites[parseName] = struct{}{}
				case "Emit", "EmitEvent", "Dispatch", "DispatchEvent":
					parseAnalysis.events[parseName] = struct{}{}
				case "MustDefineRoute", "Register":
					if parseRoute := agenticFirstRouteArg(parseTyped.Args); parseRoute != "" {
						parseAnalysis.routes[parseRoute] = struct{}{}
					}
				}
			}
		case *ast.Ident:
			parseAnalysis.refs[parseTyped.Name] = struct{}{}
		case *ast.SelectorExpr:
			parseAnalysis.refs[parseTyped.Sel.Name] = struct{}{}
		case *ast.KeyValueExpr:
			if parseEvent := agenticEventKey(parseTyped.Key); parseEvent != "" {
				parseAnalysis.events[parseEvent] = struct{}{}
			}
		}
		return true
	})
	return parseAnalysis
}

func isAgenticComponentDecl(parseDecl *ast.FuncDecl) bool {
	if parseDecl == nil || parseDecl.Recv != nil || parseDecl.Type == nil || parseDecl.Type.Results == nil {
		return false
	}
	if len(parseDecl.Type.Results.List) != 1 {
		return false
	}
	return strings.Contains(agenticExprString(parseDecl.Type.Results.List[0].Type), "Node")
}

func agenticComponentProps(parseDecl *ast.FuncDecl) string {
	if parseDecl == nil || parseDecl.Type == nil || parseDecl.Type.Params == nil || len(parseDecl.Type.Params.List) == 0 {
		return ""
	}
	parseParts := []string{}
	for _, parseField := range parseDecl.Type.Params.List {
		parseType := agenticExprString(parseField.Type)
		if len(parseField.Names) == 0 {
			parseParts = append(parseParts, parseType)
			continue
		}
		for range parseField.Names {
			parseParts = append(parseParts, parseType)
		}
	}
	return strings.Join(parseParts, ", ")
}

func agenticCallName(parseExpr ast.Expr) string {
	switch parseTyped := parseExpr.(type) {
	case *ast.Ident:
		return parseTyped.Name
	case *ast.SelectorExpr:
		parsePrefix := agenticCallName(parseTyped.X)
		if parsePrefix == "" {
			return parseTyped.Sel.Name
		}
		return parsePrefix + "." + parseTyped.Sel.Name
	case *ast.IndexExpr:
		return agenticCallName(parseTyped.X)
	case *ast.IndexListExpr:
		return agenticCallName(parseTyped.X)
	}
	return ""
}

func agenticLastNameSegment(parseName string) string {
	parseName = strings.TrimSpace(parseName)
	if parseName == "" {
		return ""
	}
	if parseDot := strings.LastIndex(parseName, "."); parseDot >= 0 {
		return parseName[parseDot+1:]
	}
	return parseName
}

func agenticFirstStringArg(parseArgs []ast.Expr) string {
	for _, parseArg := range parseArgs {
		if parseValue := agenticStringLiteralValue(parseArg); parseValue != "" {
			return parseValue
		}
	}
	return ""
}

func agenticFirstRouteArg(parseArgs []ast.Expr) string {
	for _, parseArg := range parseArgs {
		parseValue := agenticStringLiteralValue(parseArg)
		if strings.HasPrefix(parseValue, "/") {
			return parseValue
		}
	}
	return ""
}

func agenticStringLiteralValue(parseExpr ast.Expr) string {
	parseLiteral, parseOk := parseExpr.(*ast.BasicLit)
	if !parseOk || parseLiteral.Kind != token.STRING {
		return ""
	}
	parseValue, parseErr := strconv.Unquote(parseLiteral.Value)
	if parseErr != nil {
		return ""
	}
	return parseValue
}

func agenticEventKey(parseExpr ast.Expr) string {
	var parseName string
	switch parseTyped := parseExpr.(type) {
	case *ast.Ident:
		parseName = parseTyped.Name
	case *ast.SelectorExpr:
		parseName = parseTyped.Sel.Name
	}
	if strings.HasPrefix(parseName, "On") && len(parseName) > 2 {
		return parseName
	}
	return ""
}

func agenticSymbolID(parseImportPath string, parsePackageName string, parseName string) string {
	parsePrefix := firstNonEmpty(strings.TrimSpace(parseImportPath), strings.TrimSpace(parsePackageName), "package")
	return parsePrefix + "." + strings.TrimSpace(parseName)
}

func agenticFuncTypeString(parseType *ast.FuncType) string {
	parseRendered := agenticNodeString(parseType)
	return strings.TrimPrefix(parseRendered, "func")
}

func agenticExprString(parseExpr ast.Expr) string {
	return agenticNodeString(parseExpr)
}

func agenticNodeString(parseNode any) string {
	if parseNode == nil {
		return ""
	}
	var parseBuffer bytes.Buffer
	if parseErr := printer.Fprint(&parseBuffer, token.NewFileSet(), parseNode); parseErr != nil {
		return ""
	}
	return strings.TrimSpace(parseBuffer.String())
}

func agenticReceiverName(parseExpr ast.Expr) string {
	switch parseTyped := parseExpr.(type) {
	case *ast.Ident:
		return parseTyped.Name
	case *ast.StarExpr:
		return agenticReceiverName(parseTyped.X)
	case *ast.IndexExpr:
		return agenticReceiverName(parseTyped.X)
	case *ast.IndexListExpr:
		return agenticReceiverName(parseTyped.X)
	case *ast.SelectorExpr:
		return parseTyped.Sel.Name
	default:
		return agenticExprString(parseExpr)
	}
}

func agenticCommentSummary(parseComment *ast.CommentGroup) string {
	if parseComment == nil {
		return ""
	}
	parseText := strings.TrimSpace(parseComment.Text())
	if parseText == "" {
		return ""
	}
	parseText = strings.Join(strings.Fields(parseText), " ")
	parseEnd := len(parseText)
	for _, parseMarker := range []string{". ", "\n"} {
		if parseIndex := strings.Index(parseText, parseMarker); parseIndex >= 0 && parseIndex+1 < parseEnd {
			parseEnd = parseIndex + 1
		}
	}
	return strings.TrimSpace(parseText[:parseEnd])
}

func firstAgenticComment(parseComments ...*ast.CommentGroup) *ast.CommentGroup {
	for _, parseComment := range parseComments {
		if parseComment != nil {
			return parseComment
		}
	}
	return nil
}

func sortedAgenticSet(parseSet map[string]struct{}) []string {
	return sortedAgenticStringSet(parseSet)
}

func sortedAgenticStringSet(parseSet map[string]struct{}) []string {
	if len(parseSet) == 0 {
		return nil
	}
	parseValues := make([]string, 0, len(parseSet))
	for parseValue := range parseSet {
		if strings.TrimSpace(parseValue) != "" {
			parseValues = append(parseValues, parseValue)
		}
	}
	sort.Strings(parseValues)
	return parseValues
}

func agenticCapabilityTagsForPackage(parseImportPath string) []string {
	parseImportPath = strings.Trim(parseImportPath, "/")
	if parseImportPath == "" {
		return nil
	}
	parseSuffix := parseImportPath
	if parseSlash := strings.LastIndex(parseImportPath, "/"); parseSlash >= 0 {
		parseSuffix = parseImportPath[parseSlash+1:]
	}
	parseTags := map[string]struct{}{}
	switch parseSuffix {
	case "ui":
		parseTags["components-hooks"] = struct{}{}
		parseTags["ssr-hydration"] = struct{}{}
		parseTags["forms-accessibility"] = struct{}{}
	case "html", "shorthand":
		parseTags["html-authoring"] = struct{}{}
	case "state":
		parseTags["shared-state-reactivity"] = struct{}{}
	case "fetch":
		parseTags["data-loading-mutations"] = struct{}{}
		parseTags["realtime-data"] = struct{}{}
	case "router":
		parseTags["routing"] = struct{}{}
	case "interop":
		parseTags["browser-interop-workers"] = struct{}{}
	case "i18n":
		parseTags["internationalization"] = struct{}{}
	case "devtools":
		parseTags["devtools-diagnostics"] = struct{}{}
	case "pwa":
		parseTags["pwa-offline"] = struct{}{}
	case "flags":
		parseTags["feature-flags"] = struct{}{}
	}
	return sortedAgenticStringSet(parseTags)
}

func buildAgenticImpactReport(parseRootPath string, parseSymbol string) (agenticImpactReport, error) {
	parseSymbol = strings.TrimSpace(parseSymbol)
	if parseSymbol == "" {
		return agenticImpactReport{}, errors.New("impact symbol is required")
	}
	parseManifest, parseErr := buildAgenticModelManifest(parseRootPath)
	if parseErr != nil {
		return agenticImpactReport{}, parseErr
	}
	parseReport := agenticImpactReport{
		OK:          true,
		Symbol:      parseSymbol,
		Root:        parseRootPath,
		Diagnostics: append([]agenticReadDiagnostic(nil), parseManifest.Diagnostics...),
	}

	parseReverse := map[string][]agenticSymbol{}
	parseByID := map[string]agenticSymbol{}
	for _, parseSymbolRecord := range parseManifest.Symbols {
		parseByID[parseSymbolRecord.ID] = parseSymbolRecord
		for _, parseRef := range parseSymbolRecord.References {
			parseReverse[strings.ToLower(parseRef)] = append(parseReverse[strings.ToLower(parseRef)], parseSymbolRecord)
		}
	}
	for _, parseEdge := range parseManifest.Edges {
		parseTo := parseByID[parseEdge.To]
		parseFrom := parseByID[parseEdge.From]
		if parseTo.Name != "" && parseFrom.Name != "" {
			parseReverse[strings.ToLower(parseTo.Name)] = append(parseReverse[strings.ToLower(parseTo.Name)], parseFrom)
		}
	}

	parseDirectSeen := map[string]struct{}{}
	for _, parseDependent := range parseReverse[strings.ToLower(agenticImpactLookupName(parseSymbol))] {
		if _, parseExists := parseDirectSeen[parseDependent.ID]; parseExists {
			continue
		}
		parseDirectSeen[parseDependent.ID] = struct{}{}
		parseReport.DirectDependents = append(parseReport.DirectDependents, agenticImpactTargetForSymbol(parseDependent, 1))
	}
	sortAgenticImpactTargets(parseReport.DirectDependents)

	parseVisited := map[string]struct{}{}
	parseQueue := append([]agenticImpactTarget(nil), parseReport.DirectDependents...)
	for len(parseQueue) > 0 {
		parseCurrent := parseQueue[0]
		parseQueue = parseQueue[1:]
		if _, parseExists := parseVisited[parseCurrent.ID]; parseExists {
			continue
		}
		parseVisited[parseCurrent.ID] = struct{}{}
		parseReport.TransitiveDependents = append(parseReport.TransitiveDependents, parseCurrent)
		for _, parseNext := range parseReverse[strings.ToLower(parseCurrent.Name)] {
			if _, parseDone := parseVisited[parseNext.ID]; parseDone {
				continue
			}
			parseQueue = append(parseQueue, agenticImpactTargetForSymbol(parseNext, parseCurrent.Distance+1))
		}
	}
	sortAgenticImpactTargets(parseReport.TransitiveDependents)

	parseTests, parseDocs, parseScanErr := scanAgenticImpactSurfaces(parseRootPath, parseSymbol)
	if parseScanErr != nil {
		parseReport.OK = false
		parseReport.Diagnostics = append(parseReport.Diagnostics, agenticReadDiagnostic{
			Severity: "error",
			Code:     "impact_surface_scan",
			Message:  parseScanErr.Error(),
		})
	}
	parseReport.Tests = parseTests
	parseReport.Docs = parseDocs
	return parseReport, nil
}

func agenticImpactLookupName(parseSymbol string) string {
	parseSymbol = strings.TrimSpace(parseSymbol)
	if parseSymbol == "" {
		return ""
	}
	if parseDot := strings.LastIndex(parseSymbol, "."); parseDot >= 0 {
		return parseSymbol[parseDot+1:]
	}
	return parseSymbol
}

func agenticImpactTargetForSymbol(parseSymbol agenticSymbol, parseDistance int) agenticImpactTarget {
	return agenticImpactTarget{
		ID:       parseSymbol.ID,
		Name:     parseSymbol.Name,
		Kind:     parseSymbol.Kind,
		Package:  parseSymbol.Package,
		File:     parseSymbol.File,
		Line:     parseSymbol.Line,
		Distance: parseDistance,
	}
}

func sortAgenticImpactTargets(parseTargets []agenticImpactTarget) {
	sort.Slice(parseTargets, func(parseLeft int, parseRight int) bool {
		if parseTargets[parseLeft].Distance != parseTargets[parseRight].Distance {
			return parseTargets[parseLeft].Distance < parseTargets[parseRight].Distance
		}
		return parseTargets[parseLeft].ID < parseTargets[parseRight].ID
	})
}

func scanAgenticImpactSurfaces(parseRootPath string, parseSymbol string) ([]string, []string, error) {
	parseTests := map[string]struct{}{}
	parseDocs := map[string]struct{}{}
	parseNeedle := []byte(parseSymbol)
	parseShortNeedle := []byte(agenticImpactLookupName(parseSymbol))
	parseErr := filepath.WalkDir(parseRootPath, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return parseWalkErr
		}
		if parseEntry.IsDir() {
			if shouldSkipDoctorAuditDir(parseEntry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		parseName := parseEntry.Name()
		parseExt := strings.ToLower(filepath.Ext(parseName))
		parseIsTest := strings.HasSuffix(parseName, "_test.go")
		parseIsDoc := parseExt == ".md" || parseExt == ".mdx" || parseExt == ".txt"
		if !parseIsTest && !parseIsDoc {
			return nil
		}
		parseContent, parseReadErr := os.ReadFile(parsePath)
		if parseReadErr != nil {
			return parseReadErr
		}
		if !bytes.Contains(parseContent, parseNeedle) && !bytes.Contains(parseContent, parseShortNeedle) {
			return nil
		}
		parseRelPath, parseRelErr := filepath.Rel(parseRootPath, parsePath)
		if parseRelErr != nil {
			parseRelPath = parsePath
		}
		parseRelPath = filepath.ToSlash(parseRelPath)
		if parseIsTest {
			parseTests[parseRelPath] = struct{}{}
		}
		if parseIsDoc {
			parseDocs[parseRelPath] = struct{}{}
		}
		return nil
	})
	if parseErr != nil {
		return nil, nil, parseErr
	}
	return sortedAgenticStringSet(parseTests), sortedAgenticStringSet(parseDocs), nil
}
