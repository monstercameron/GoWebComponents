package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	gwccapabilities "github.com/monstercameron/GoWebComponents/docs/capabilities"
	gwcerrorcodes "github.com/monstercameron/GoWebComponents/docs/errorcodes"
	playwright "github.com/playwright-community/playwright-go"
)

var runModelCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runModel(parseArgs)
}

var runRenderCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runRender(parseArgs)
}

var runProbeCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runProbe(parseArgs)
}

var runExplainCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runExplain(parseArgs)
}

var runSearchCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runSearch(parseArgs)
}

var runInspectImpactCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runInspectImpact(parseArgs)
}

var agenticRenderGoTest = runAgenticRenderGoTest

var agenticRunBrowserProbe = runAgenticProbeWithPlaywright

type agenticSearchConfig struct {
	rootPath string
	query    string
	limit    int
	json     bool
}

type agenticSearchReport struct {
	OK      bool                  `json:"ok"`
	Root    string                `json:"root"`
	Query   string                `json:"query"`
	Count   int                   `json:"count"`
	Results []agenticSearchResult `json:"results,omitempty"`
}

type agenticSearchResult struct {
	Symbol         string   `json:"symbol"`
	Name           string   `json:"name"`
	Kind           string   `json:"kind"`
	Package        string   `json:"package,omitempty"`
	File           string   `json:"file"`
	Line           int      `json:"line"`
	Signature      string   `json:"signature,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	CapabilityTags []string `json:"capabilityTags,omitempty"`
	Score          int      `json:"score"`
	UsagePointer   string   `json:"usagePointer,omitempty"`
}

type agenticExplainReport struct {
	OK         bool                      `json:"ok"`
	Query      string                    `json:"query"`
	Kind       string                    `json:"kind,omitempty"`
	ErrorCode  *agenticExplainErrorCode  `json:"errorCode,omitempty"`
	Capability *agenticExplainCapability `json:"capability,omitempty"`
	NotFound   bool                      `json:"notFound,omitempty"`
	Message    string                    `json:"message,omitempty"`
}

type agenticExplainErrorCode struct {
	Code        string `json:"code"`
	Docs        string `json:"docs,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

type agenticExplainCapability struct {
	Name        string   `json:"name"`
	Packages    []string `json:"packages,omitempty"`
	ExampleSlug string   `json:"exampleSlug,omitempty"`
	Chapter     string   `json:"chapter,omitempty"`
}

type agenticRenderConfig struct {
	rootPath  string
	component string
	propsJSON string
	json      bool
}

type agenticRenderReport struct {
	OK          bool                    `json:"ok"`
	Root        string                  `json:"root"`
	Component   string                  `json:"component"`
	Package     string                  `json:"package,omitempty"`
	File        string                  `json:"file,omitempty"`
	Line        int                     `json:"line,omitempty"`
	Props       string                  `json:"props,omitempty"`
	HTML        string                  `json:"html,omitempty"`
	Error       string                  `json:"error,omitempty"`
	Diagnostics []agenticReadDiagnostic `json:"diagnostics,omitempty"`
}

type agenticProbeConfig struct {
	rootPath string
	target   string
	url      string
	timeout  time.Duration
	json     bool
}

type agenticProbeReport struct {
	OK            bool     `json:"ok"`
	Target        string   `json:"target"`
	URL           string   `json:"url,omitempty"`
	Skipped       bool     `json:"skipped,omitempty"`
	Reason        string   `json:"reason,omitempty"`
	Status        int      `json:"status,omitempty"`
	Title         string   `json:"title,omitempty"`
	BodyText      string   `json:"bodyText,omitempty"`
	ConsoleErrors []string `json:"consoleErrors,omitempty"`
	PageErrors    []string `json:"pageErrors,omitempty"`
	DurationMs    int64    `json:"durationMs,omitempty"`
}

type agenticRenderComponentTarget struct {
	component   agenticComponent
	packageDir  string
	packageName string
	propsType   string
}

func (parseL launcher) runModel(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"root"}, []string{"json"})
	parseFlags := flag.NewFlagSet("model", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Root directory to model; defaults to the current working directory")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON manifest")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseConfig, parseErr := resolveAgenticReadConfig(agenticReadConfig{rootPath: *parseRoot, json: *parseJSON})
	if parseErr != nil {
		return parseErr
	}
	parseManifest, parseErr := buildAgenticModelManifest(parseConfig.rootPath)
	if parseErr != nil {
		return parseErr
	}
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseManifest)
	}
	renderAgenticModelSummary(parseManifest)
	return nil
}

func (parseL launcher) runSearch(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"root", "limit"}, []string{"json"})
	parseFlags := flag.NewFlagSet("search", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Root directory to index; defaults to the current working directory")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON report")
	parseLimit := parseFlags.Int("limit", 10, "Maximum result count")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseQuery := strings.TrimSpace(strings.Join(parseFlags.Args(), " "))
	parseConfig, parseErr := resolveAgenticSearchConfig(agenticSearchConfig{
		rootPath: *parseRoot,
		query:    parseQuery,
		limit:    *parseLimit,
		json:     *parseJSON,
	})
	if parseErr != nil {
		return parseErr
	}
	parseReport, parseErr := buildAgenticSearchReport(parseConfig)
	if parseErr != nil {
		return parseErr
	}
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseReport)
	}
	renderAgenticSearchReport(parseReport)
	return nil
}

func (parseL launcher) runExplain(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"root"}, []string{"json"})
	parseFlags := flag.NewFlagSet("explain", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", firstNonEmpty(parseL.repoRoot, ""), "Repository root used for generated error-code metadata")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON report")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseQuery := strings.TrimSpace(strings.Join(parseFlags.Args(), " "))
	if parseQuery == "" {
		return errors.New("explain requires an error code or capability query")
	}
	parseConfig, parseErr := resolveAgenticReadConfig(agenticReadConfig{rootPath: *parseRoot, json: *parseJSON})
	if parseErr != nil {
		return parseErr
	}
	parseReport := buildAgenticExplainReport(parseConfig.rootPath, parseQuery)
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseReport)
	}
	renderAgenticExplainReport(parseReport)
	return nil
}

func (parseL launcher) runInspectImpact(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"root", "impact"}, []string{"json"})
	parseFlags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Root directory to inspect; defaults to the current working directory")
	parseImpact := parseFlags.String("impact", "", "Symbol to query for direct and transitive dependents")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON report")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseConfig, parseErr := resolveAgenticReadConfig(agenticReadConfig{rootPath: *parseRoot, json: *parseJSON})
	if parseErr != nil {
		return parseErr
	}
	parseReport, parseErr := buildAgenticImpactReport(parseConfig.rootPath, *parseImpact)
	if parseErr != nil {
		return parseErr
	}
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseReport)
	}
	printAgenticHumanSummary("GWC inspect impact", parseReport.OK, []string{
		fmt.Sprintf("symbol: %s", parseReport.Symbol),
		fmt.Sprintf("direct: %d", len(parseReport.DirectDependents)),
		fmt.Sprintf("transitive: %d", len(parseReport.TransitiveDependents)),
	})
	return nil
}

func (parseL launcher) runRender(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"root", "props"}, []string{"json"})
	parseFlags := flag.NewFlagSet("render", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Root directory containing the target Go module; defaults to the current working directory")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON report")
	parseProps := parseFlags.String("props", "", "JSON props object to pass to a one-argument component")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseComponent := strings.TrimSpace(strings.Join(parseFlags.Args(), " "))
	parseConfig, parseErr := resolveAgenticRenderConfig(agenticRenderConfig{
		rootPath:  *parseRoot,
		component: parseComponent,
		propsJSON: *parseProps,
		json:      *parseJSON,
	})
	if parseErr != nil {
		return parseErr
	}
	parseReport := buildAgenticRenderReport(parseConfig)
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseReport)
	}
	renderAgenticRenderReport(parseReport)
	return nil
}

func (parseL launcher) runProbe(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"root", "url", "timeout"}, []string{"json"})
	parseFlags := flag.NewFlagSet("probe", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", firstNonEmpty(parseL.repoRoot, ""), "Repository root for example lookup")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON report")
	parseURL := parseFlags.String("url", "", "URL to probe, or base URL for a public example slug")
	parseTimeout := parseFlags.Duration("timeout", 15*time.Second, "Browser navigation timeout")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseTarget := strings.TrimSpace(strings.Join(parseFlags.Args(), " "))
	parseConfig, parseErr := resolveAgenticProbeConfig(agenticProbeConfig{
		rootPath: *parseRoot,
		target:   parseTarget,
		url:      *parseURL,
		timeout:  *parseTimeout,
		json:     *parseJSON,
	})
	if parseErr != nil {
		return parseErr
	}
	parseReport := buildAgenticProbeReport(parseConfig)
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseReport)
	}
	renderAgenticProbeReport(parseReport)
	return nil
}

func resolveAgenticSearchConfig(parseConfig agenticSearchConfig) (agenticSearchConfig, error) {
	parseReadConfig, parseErr := resolveAgenticReadConfig(agenticReadConfig{rootPath: parseConfig.rootPath, json: parseConfig.json})
	if parseErr != nil {
		return agenticSearchConfig{}, parseErr
	}
	parseQuery := strings.TrimSpace(parseConfig.query)
	if parseQuery == "" {
		return agenticSearchConfig{}, errors.New("search requires a query")
	}
	parseLimit := parseConfig.limit
	if parseLimit <= 0 {
		parseLimit = 10
	}
	if parseLimit > 100 {
		parseLimit = 100
	}
	return agenticSearchConfig{
		rootPath: parseReadConfig.rootPath,
		query:    parseQuery,
		limit:    parseLimit,
		json:     parseConfig.json,
	}, nil
}

func buildAgenticSearchReport(parseConfig agenticSearchConfig) (agenticSearchReport, error) {
	parseManifest, parseErr := buildAgenticModelManifest(parseConfig.rootPath)
	if parseErr != nil {
		return agenticSearchReport{}, parseErr
	}
	parseQueryTokens := agenticSearchTokens(parseConfig.query)
	parseResults := []agenticSearchResult{}
	for _, parseSymbol := range parseManifest.Symbols {
		if !parseSymbol.Exported {
			continue
		}
		parseScore := scoreAgenticSearchSymbol(parseQueryTokens, parseConfig.query, parseSymbol)
		if parseScore <= 0 {
			continue
		}
		parseResults = append(parseResults, agenticSearchResult{
			Symbol:         parseSymbol.ID,
			Name:           parseSymbol.Name,
			Kind:           parseSymbol.Kind,
			Package:        parseSymbol.Package,
			File:           parseSymbol.File,
			Line:           parseSymbol.Line,
			Signature:      parseSymbol.Signature,
			Summary:        parseSymbol.Summary,
			CapabilityTags: append([]string(nil), parseSymbol.CapabilityTags...),
			Score:          parseScore,
			UsagePointer:   fmt.Sprintf("%s:%d", parseSymbol.File, parseSymbol.Line),
		})
	}
	sort.Slice(parseResults, func(parseLeft int, parseRight int) bool {
		if parseResults[parseLeft].Score != parseResults[parseRight].Score {
			return parseResults[parseLeft].Score > parseResults[parseRight].Score
		}
		return parseResults[parseLeft].Symbol < parseResults[parseRight].Symbol
	})
	if len(parseResults) > parseConfig.limit {
		parseResults = parseResults[:parseConfig.limit]
	}
	return agenticSearchReport{
		OK:      true,
		Root:    parseConfig.rootPath,
		Query:   parseConfig.query,
		Count:   len(parseResults),
		Results: parseResults,
	}, nil
}

func agenticSearchTokens(parseQuery string) []string {
	parseTokenSet := map[string]struct{}{}
	parseLower := strings.ToLower(parseQuery)
	parseFields := strings.FieldsFunc(parseLower, func(parseRune rune) bool {
		return !(parseRune >= 'a' && parseRune <= 'z') && !(parseRune >= '0' && parseRune <= '9')
	})
	for _, parseField := range parseFields {
		parseField = strings.TrimSpace(parseField)
		if len(parseField) < 2 {
			continue
		}
		parseTokenSet[parseField] = struct{}{}
	}
	return sortedAgenticStringSet(parseTokenSet)
}

func scoreAgenticSearchSymbol(parseTokens []string, parseQuery string, parseSymbol agenticSymbol) int {
	parseQueryLower := strings.ToLower(strings.TrimSpace(parseQuery))
	parseNameLower := strings.ToLower(parseSymbol.Name)
	parseHaystacks := []struct {
		text   string
		weight int
	}{
		{parseNameLower, 35},
		{strings.ToLower(parseSymbol.Signature), 18},
		{strings.ToLower(parseSymbol.Summary), 12},
		{strings.ToLower(strings.Join(parseSymbol.CapabilityTags, " ")), 10},
		{strings.ToLower(parseSymbol.Package), 5},
	}
	parseScore := 0
	if parseNameLower == parseQueryLower {
		parseScore += 120
	} else if strings.Contains(parseNameLower, parseQueryLower) {
		parseScore += 60
	}
	for _, parseToken := range parseTokens {
		for _, parseHaystack := range parseHaystacks {
			if strings.Contains(parseHaystack.text, parseToken) {
				parseScore += parseHaystack.weight
			}
		}
	}
	return parseScore
}

func buildAgenticExplainReport(parseRootPath string, parseQuery string) agenticExplainReport {
	parseQuery = strings.TrimSpace(parseQuery)
	parseCodes := loadAgenticErrorCodes(parseRootPath)
	for _, parseCode := range parseCodes {
		if strings.EqualFold(parseCode.Code, parseQuery) {
			return agenticExplainReport{
				OK:    true,
				Query: parseQuery,
				Kind:  "errorcode",
				ErrorCode: &agenticExplainErrorCode{
					Code:        parseCode.Code,
					Docs:        parseCode.Docs,
					Remediation: parseCode.Remediation,
				},
			}
		}
	}
	for _, parseCap := range gwccapabilities.Capabilities() {
		if strings.EqualFold(parseCap.Name, parseQuery) || strings.Contains(strings.ToLower(parseCap.Name), strings.ToLower(parseQuery)) {
			return agenticExplainReport{
				OK:    true,
				Query: parseQuery,
				Kind:  "capability",
				Capability: &agenticExplainCapability{
					Name:        parseCap.Name,
					Packages:    append([]string(nil), parseCap.Packages...),
					ExampleSlug: parseCap.ExampleSlug,
					Chapter:     parseCap.Chapter,
				},
			}
		}
	}
	return agenticExplainReport{
		OK:       false,
		Query:    parseQuery,
		NotFound: true,
		Message:  "no matching error code or capability",
	}
}

func loadAgenticErrorCodes(parseRootPath string) []gwcerrorcodes.ErrorCode {
	parseContent, parseErr := os.ReadFile(filepath.Join(parseRootPath, "internal", "runtime", "diagnostic_metadata.go"))
	if parseErr != nil {
		return nil
	}
	return gwcerrorcodes.ExtractCodes(string(parseContent))
}

func resolveAgenticRenderConfig(parseConfig agenticRenderConfig) (agenticRenderConfig, error) {
	parseReadConfig, parseErr := resolveAgenticReadConfig(agenticReadConfig{rootPath: parseConfig.rootPath, json: parseConfig.json})
	if parseErr != nil {
		return agenticRenderConfig{}, parseErr
	}
	parseComponent := strings.TrimSpace(parseConfig.component)
	if parseComponent == "" {
		return agenticRenderConfig{}, errors.New("render requires a component name or manifest id")
	}
	parseProps := strings.TrimSpace(parseConfig.propsJSON)
	if parseProps != "" && !json.Valid([]byte(parseProps)) {
		return agenticRenderConfig{}, errors.New("render props must be valid JSON")
	}
	return agenticRenderConfig{
		rootPath:  parseReadConfig.rootPath,
		component: parseComponent,
		propsJSON: parseProps,
		json:      parseConfig.json,
	}, nil
}

func buildAgenticRenderReport(parseConfig agenticRenderConfig) agenticRenderReport {
	parseTarget, parseErr := resolveAgenticRenderTarget(parseConfig.rootPath, parseConfig.component)
	if parseErr != nil {
		return agenticRenderReport{
			OK:        false,
			Root:      parseConfig.rootPath,
			Component: parseConfig.component,
			Error:     parseErr.Error(),
		}
	}
	parsePropsJSON := strings.TrimSpace(parseConfig.propsJSON)
	if parseTarget.propsType != "" && parsePropsJSON == "" {
		parsePropsJSON = "{}"
	}
	if parseTarget.propsType == "" && parsePropsJSON != "" {
		return agenticRenderReport{
			OK:        false,
			Root:      parseConfig.rootPath,
			Component: parseConfig.component,
			Package:   parseTarget.component.Package,
			File:      parseTarget.component.File,
			Line:      parseTarget.component.Line,
			Error:     "props were provided for a zero-argument component",
		}
	}
	parseReport, parseErr := agenticRenderGoTest(parseConfig.rootPath, parseTarget, parsePropsJSON)
	if parseErr != nil {
		parseReport.OK = false
		parseReport.Error = parseErr.Error()
	}
	parseReport.Root = parseConfig.rootPath
	parseReport.Component = parseTarget.component.Name
	parseReport.Package = parseTarget.component.Package
	parseReport.File = parseTarget.component.File
	parseReport.Line = parseTarget.component.Line
	parseReport.Props = parseTarget.propsType
	return parseReport
}

func resolveAgenticRenderTarget(parseRootPath string, parseComponent string) (agenticRenderComponentTarget, error) {
	parseManifest, parseErr := buildAgenticModelManifest(parseRootPath)
	if parseErr != nil {
		return agenticRenderComponentTarget{}, parseErr
	}
	parseMatches := []agenticComponent{}
	for _, parseCandidate := range parseManifest.Components {
		if parseCandidate.Name == parseComponent || parseCandidate.ID == parseComponent || strings.HasSuffix(parseCandidate.ID, "."+parseComponent) {
			parseMatches = append(parseMatches, parseCandidate)
		}
	}
	if len(parseMatches) == 0 {
		return agenticRenderComponentTarget{}, fmt.Errorf("component %q not found", parseComponent)
	}
	if len(parseMatches) > 1 {
		parseIDs := make([]string, 0, len(parseMatches))
		for _, parseMatch := range parseMatches {
			parseIDs = append(parseIDs, parseMatch.ID)
		}
		sort.Strings(parseIDs)
		return agenticRenderComponentTarget{}, fmt.Errorf("component %q is ambiguous: %s", parseComponent, strings.Join(parseIDs, ", "))
	}
	parseComponentRecord := parseMatches[0]
	parsePropsType := strings.TrimSpace(parseComponentRecord.Props)
	if strings.Contains(parsePropsType, ",") {
		return agenticRenderComponentTarget{}, fmt.Errorf("component %q has multiple props parameters; render supports zero or one", parseComponent)
	}
	if strings.Contains(parsePropsType, ".") && !strings.HasPrefix(parsePropsType, "*") {
		return agenticRenderComponentTarget{}, fmt.Errorf("component %q has selector props type %q; render currently supports local props types", parseComponent, parsePropsType)
	}
	return agenticRenderComponentTarget{
		component:   parseComponentRecord,
		packageDir:  filepath.Join(parseRootPath, filepath.FromSlash(filepath.Dir(parseComponentRecord.File))),
		packageName: parseComponentRecord.PackageName,
		propsType:   parsePropsType,
	}, nil
}

func runAgenticRenderGoTest(parseRootPath string, parseTarget agenticRenderComponentTarget, parsePropsJSON string) (agenticRenderReport, error) {
	parsePackageDir := parseTarget.packageDir
	if strings.HasSuffix(parsePackageDir, string(filepath.Separator)+".") {
		parsePackageDir = filepath.Dir(parsePackageDir)
	}
	parseTestSource := buildAgenticRenderTestSource(parseTarget, parsePropsJSON)
	parseTempDir, parseErr := os.MkdirTemp("", "gwc-render-overlay-*")
	if parseErr != nil {
		return agenticRenderReport{}, fmt.Errorf("create render overlay temp dir: %w", parseErr)
	}
	defer os.RemoveAll(parseTempDir)

	parseBackingPath := filepath.Join(parseTempDir, "gwc_render_oracle_test.go")
	if parseErr = os.WriteFile(parseBackingPath, []byte(parseTestSource), 0644); parseErr != nil {
		return agenticRenderReport{}, fmt.Errorf("write render overlay test: %w", parseErr)
	}
	parseSyntheticPath := filepath.Join(parsePackageDir, "gwc_render_oracle_test.go")
	parseOverlayPath := filepath.Join(parseTempDir, "overlay.json")
	parseOverlayBytes, parseErr := json.Marshal(map[string]map[string]string{
		"Replace": {
			parseSyntheticPath: parseBackingPath,
		},
	})
	if parseErr != nil {
		return agenticRenderReport{}, fmt.Errorf("encode render overlay: %w", parseErr)
	}
	if parseErr = os.WriteFile(parseOverlayPath, parseOverlayBytes, 0644); parseErr != nil {
		return agenticRenderReport{}, fmt.Errorf("write render overlay: %w", parseErr)
	}

	parseCmd := exec.Command("go", "test", "-overlay", parseOverlayPath, "-run", "^TestGWCReadSurfaceRender$", "-count=1", "-v", ".")
	parseCmd.Dir = parsePackageDir
	parseOutput, parseErr := parseCmd.CombinedOutput()
	parseReport, parseReportErr := parseAgenticRenderOutput(string(parseOutput))
	if parseReportErr != nil {
		if parseErr != nil {
			return agenticRenderReport{}, fmt.Errorf("go test render oracle failed: %s", strings.TrimSpace(string(parseOutput)))
		}
		return agenticRenderReport{}, parseReportErr
	}
	if parseErr != nil && parseReport.OK {
		return parseReport, fmt.Errorf("go test render oracle failed after output: %s", strings.TrimSpace(string(parseOutput)))
	}
	return parseReport, nil
}

func buildAgenticRenderTestSource(parseTarget agenticRenderComponentTarget, parsePropsJSON string) string {
	parsePackageName := firstNonEmpty(parseTarget.packageName, "main")
	var parseBuilder strings.Builder
	parseBuilder.WriteString("package " + parsePackageName + "\n\n")
	parseBuilder.WriteString("import (\n")
	parseBuilder.WriteString("\t\"encoding/json\"\n")
	parseBuilder.WriteString("\t\"fmt\"\n")
	parseBuilder.WriteString("\t\"testing\"\n\n")
	parseBuilder.WriteString("\tgwcui \"github.com/monstercameron/GoWebComponents/ui\"\n")
	parseBuilder.WriteString(")\n\n")
	parseBuilder.WriteString("func TestGWCReadSurfaceRender(parseT *testing.T) {\n")
	parseBuilder.WriteString("\tparseReport := struct {\n")
	parseBuilder.WriteString("\t\tOK bool `json:\"ok\"`\n")
	parseBuilder.WriteString("\t\tHTML string `json:\"html,omitempty\"`\n")
	parseBuilder.WriteString("\t\tError string `json:\"error,omitempty\"`\n")
	parseBuilder.WriteString("\t}{}\n")
	parseBuilder.WriteString("\tparseEmit := func() { parseBytes, _ := json.Marshal(parseReport); fmt.Println(\"GWC_RENDER_JSON:\" + string(parseBytes)) }\n")
	parseBuilder.WriteString("\tdefer func() { if parseRecovered := recover(); parseRecovered != nil { parseReport.OK = false; parseReport.Error = fmt.Sprint(parseRecovered); parseEmit() } }()\n")
	if parseTarget.propsType == "" {
		parseBuilder.WriteString("\tparseNode := gwcui.CreateElement(" + parseTarget.component.Name + ")\n")
	} else {
		parsePropsType := strings.TrimSpace(parseTarget.propsType)
		parsePassExpr := "parseProps"
		if strings.HasPrefix(parsePropsType, "*") {
			parsePropsType = strings.TrimPrefix(parsePropsType, "*")
			parsePassExpr = "&parseProps"
		}
		parseBuilder.WriteString("\tvar parseProps " + parsePropsType + "\n")
		parseBuilder.WriteString("\tif parseErr := json.Unmarshal([]byte(" + strconvQuote(parsePropsJSON) + "), &parseProps); parseErr != nil { parseReport.OK = false; parseReport.Error = \"decode props: \" + parseErr.Error(); parseEmit(); return }\n")
		parseBuilder.WriteString("\tparseNode := gwcui.CreateElement(" + parseTarget.component.Name + ", " + parsePassExpr + ")\n")
	}
	parseBuilder.WriteString("\tparseHTML, parseErr := gwcui.RenderToString(parseNode)\n")
	parseBuilder.WriteString("\tif parseErr != nil { parseReport.OK = false; parseReport.Error = parseErr.Error(); parseEmit(); return }\n")
	parseBuilder.WriteString("\tparseReport.OK = true\n")
	parseBuilder.WriteString("\tparseReport.HTML = parseHTML\n")
	parseBuilder.WriteString("\tparseEmit()\n")
	parseBuilder.WriteString("}\n")
	return parseBuilder.String()
}

func strconvQuote(parseValue string) string {
	parseBytes, parseErr := json.Marshal(parseValue)
	if parseErr != nil {
		return "\"\""
	}
	return string(parseBytes)
}

func parseAgenticRenderOutput(parseOutput string) (agenticRenderReport, error) {
	for _, parseLine := range strings.Split(parseOutput, "\n") {
		parseLine = strings.TrimSpace(parseLine)
		if !strings.HasPrefix(parseLine, "GWC_RENDER_JSON:") {
			continue
		}
		var parseReport agenticRenderReport
		if parseErr := json.Unmarshal([]byte(strings.TrimPrefix(parseLine, "GWC_RENDER_JSON:")), &parseReport); parseErr != nil {
			return agenticRenderReport{}, fmt.Errorf("decode render oracle output: %w", parseErr)
		}
		return parseReport, nil
	}
	return agenticRenderReport{}, errors.New("render oracle did not emit a structured result")
}

func resolveAgenticProbeConfig(parseConfig agenticProbeConfig) (agenticProbeConfig, error) {
	parseRootPath := strings.TrimSpace(parseConfig.rootPath)
	if parseRootPath != "" {
		parseReadConfig, parseErr := resolveAgenticReadConfig(agenticReadConfig{rootPath: parseRootPath, json: parseConfig.json})
		if parseErr != nil {
			return agenticProbeConfig{}, parseErr
		}
		parseRootPath = parseReadConfig.rootPath
	}
	parseTarget := strings.TrimSpace(parseConfig.target)
	if parseTarget == "" {
		return agenticProbeConfig{}, errors.New("probe requires a URL or public example slug")
	}
	parseTimeout := parseConfig.timeout
	if parseTimeout <= 0 {
		parseTimeout = 15 * time.Second
	}
	parseProbeURL, parseErr := resolveAgenticProbeURL(parseTarget, strings.TrimSpace(parseConfig.url))
	if parseErr != nil {
		return agenticProbeConfig{}, parseErr
	}
	return agenticProbeConfig{
		rootPath: parseRootPath,
		target:   parseTarget,
		url:      parseProbeURL,
		timeout:  parseTimeout,
		json:     parseConfig.json,
	}, nil
}

func resolveAgenticProbeURL(parseTarget string, parseURL string) (string, error) {
	if isAgenticHTTPURL(parseTarget) {
		return parseTarget, nil
	}
	if parseURL == "" {
		return "", nil
	}
	if strings.Contains(parseURL, "{example}") {
		return strings.ReplaceAll(parseURL, "{example}", url.PathEscape(parseTarget)), nil
	}
	if isAgenticHTTPURL(parseURL) {
		parseBase, parseErr := url.Parse(parseURL)
		if parseErr != nil {
			return "", parseErr
		}
		if parseBase.Path == "" || parseBase.Path == "/" || strings.HasSuffix(parseBase.Path, "/") {
			parseBase.Path = strings.TrimRight(parseBase.Path, "/") + "/examples/public/" + url.PathEscape(parseTarget) + "/"
		}
		return parseBase.String(), nil
	}
	return "", fmt.Errorf("probe url must be an http(s) URL, got %q", parseURL)
}

func isAgenticHTTPURL(parseValue string) bool {
	parseURL, parseErr := url.Parse(strings.TrimSpace(parseValue))
	return parseErr == nil && (parseURL.Scheme == "http" || parseURL.Scheme == "https") && parseURL.Host != ""
}

func buildAgenticProbeReport(parseConfig agenticProbeConfig) agenticProbeReport {
	if parseConfig.url == "" {
		parseReport := agenticProbeReport{
			OK:      false,
			Target:  parseConfig.target,
			Skipped: true,
			Reason:  "browser probe requires a URL or -url base; no browser was launched",
		}
		if parseConfig.rootPath != "" {
			parseExamplePath := filepath.Join(parseConfig.rootPath, "examples", "public", filepath.FromSlash(parseConfig.target))
			if _, parseErr := os.Stat(parseExamplePath); parseErr == nil {
				parseReport.Reason = "public example exists, but browser probe requires a running server URL"
			}
		}
		return parseReport
	}
	parseReport := agenticRunBrowserProbe(parseConfig)
	parseReport.Target = parseConfig.target
	parseReport.URL = parseConfig.url
	return parseReport
}

func runAgenticProbeWithPlaywright(parseConfig agenticProbeConfig) agenticProbeReport {
	parseStarted := time.Now()
	parseReport := agenticProbeReport{
		Target: parseConfig.target,
		URL:    parseConfig.url,
	}
	parsePw, parseErr := playwright.Run()
	if parseErr != nil {
		parseReport.Skipped = true
		parseReport.Reason = fmt.Sprintf("playwright unavailable: %v", parseErr)
		parseReport.DurationMs = time.Since(parseStarted).Milliseconds()
		return parseReport
	}
	defer parsePw.Stop()

	parseBrowser, parseErr := parsePw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	if parseErr != nil {
		parseReport.Skipped = true
		parseReport.Reason = fmt.Sprintf("chromium unavailable: %v", parseErr)
		parseReport.DurationMs = time.Since(parseStarted).Milliseconds()
		return parseReport
	}
	defer parseBrowser.Close()

	parsePage, parseErr := parseBrowser.NewPage()
	if parseErr != nil {
		parseReport.Reason = fmt.Sprintf("create page: %v", parseErr)
		parseReport.DurationMs = time.Since(parseStarted).Milliseconds()
		return parseReport
	}
	parsePage.On("console", func(parseMsg playwright.ConsoleMessage) {
		if parseMsg.Type() == "error" {
			parseReport.ConsoleErrors = append(parseReport.ConsoleErrors, parseMsg.Text())
		}
	})
	parseResponse, parseErr := parsePage.Goto(parseConfig.url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(float64(parseConfig.timeout.Milliseconds())),
	})
	if parseErr != nil {
		parseReport.Reason = fmt.Sprintf("navigate: %v", parseErr)
		parseReport.DurationMs = time.Since(parseStarted).Milliseconds()
		return parseReport
	}
	if parseResponse != nil {
		parseReport.Status = parseResponse.Status()
	}
	if parseTitle, parseTitleErr := parsePage.Title(); parseTitleErr == nil {
		parseReport.Title = parseTitle
	}
	if parseBody, parseBodyErr := parsePage.TextContent("body"); parseBodyErr == nil {
		parseReport.BodyText = strings.TrimSpace(parseBody)
		if len(parseReport.BodyText) > 2000 {
			parseReport.BodyText = parseReport.BodyText[:2000]
		}
	}
	parseReport.DurationMs = time.Since(parseStarted).Milliseconds()
	parseReport.OK = parseReport.Status >= 200 && parseReport.Status < 400 && len(parseReport.ConsoleErrors) == 0 && len(parseReport.PageErrors) == 0
	return parseReport
}

func renderAgenticModelSummary(parseManifest agenticModelManifest) {
	fmt.Println("GWC model")
	fmt.Printf("  root:       %s\n", parseManifest.Root)
	fmt.Printf("  module:     %s\n", firstNonEmpty(parseManifest.ModulePath, "<unknown>"))
	fmt.Printf("  packages:   %d\n", len(parseManifest.Packages))
	fmt.Printf("  components: %d\n", len(parseManifest.Components))
	fmt.Printf("  symbols:    %d\n", len(parseManifest.Symbols))
	fmt.Printf("  edges:      %d\n", len(parseManifest.Edges))
	for _, parseComponent := range parseManifest.Components {
		fmt.Printf("  component:  %s (%s:%d)\n", parseComponent.ID, parseComponent.File, parseComponent.Line)
	}
}

func renderAgenticSearchReport(parseReport agenticSearchReport) {
	fmt.Println("GWC search")
	fmt.Printf("  query:   %s\n", parseReport.Query)
	fmt.Printf("  results: %d\n", parseReport.Count)
	for _, parseResult := range parseReport.Results {
		fmt.Printf("  - %s [%s] score=%d %s:%d\n", parseResult.Symbol, parseResult.Kind, parseResult.Score, parseResult.File, parseResult.Line)
		if parseResult.Summary != "" {
			fmt.Printf("    %s\n", parseResult.Summary)
		}
	}
}

func renderAgenticExplainReport(parseReport agenticExplainReport) {
	fmt.Println("GWC explain")
	if parseReport.NotFound {
		fmt.Printf("  not found: %s\n", parseReport.Query)
		return
	}
	switch parseReport.Kind {
	case "errorcode":
		fmt.Printf("  code: %s\n", parseReport.ErrorCode.Code)
		fmt.Printf("  docs: %s\n", firstNonEmpty(parseReport.ErrorCode.Docs, "<none>"))
		fmt.Printf("  next: %s\n", firstNonEmpty(parseReport.ErrorCode.Remediation, "<none>"))
	case "capability":
		fmt.Printf("  capability: %s\n", parseReport.Capability.Name)
		fmt.Printf("  packages:   %s\n", strings.Join(parseReport.Capability.Packages, ", "))
		fmt.Printf("  example:    %s\n", firstNonEmpty(parseReport.Capability.ExampleSlug, "<none>"))
		fmt.Printf("  chapter:    %s\n", firstNonEmpty(parseReport.Capability.Chapter, "<none>"))
	}
}

func renderAgenticRenderReport(parseReport agenticRenderReport) {
	fmt.Println("GWC render")
	fmt.Printf("  component: %s\n", parseReport.Component)
	if parseReport.File != "" {
		fmt.Printf("  source:    %s:%d\n", parseReport.File, parseReport.Line)
	}
	if !parseReport.OK {
		fmt.Printf("  error:     %s\n", parseReport.Error)
		return
	}
	fmt.Println(parseReport.HTML)
}

func renderAgenticProbeReport(parseReport agenticProbeReport) {
	fmt.Println("GWC probe")
	fmt.Printf("  target: %s\n", parseReport.Target)
	if parseReport.URL != "" {
		fmt.Printf("  url:    %s\n", parseReport.URL)
	}
	if parseReport.Skipped {
		fmt.Printf("  skipped: %s\n", parseReport.Reason)
		return
	}
	fmt.Printf("  ok:     %t\n", parseReport.OK)
	fmt.Printf("  status: %d\n", parseReport.Status)
	fmt.Printf("  title:  %s\n", firstNonEmpty(parseReport.Title, "<none>"))
	if len(parseReport.ConsoleErrors) > 0 {
		fmt.Println("  console errors:")
		for _, parseConsoleErr := range parseReport.ConsoleErrors {
			fmt.Printf("    - %s\n", parseConsoleErr)
		}
	}
}
