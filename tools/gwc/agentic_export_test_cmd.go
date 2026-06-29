package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"os"
	"sort"
	"strconv"
	"strings"
)

// runExportTestCommand routes recording-to-testkit test generation.
var runExportTestCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runExportTest(parseArgs)
}

type exportTestConfig struct {
	recordingPath string
	outPath       string
	packageName   string
	component     string
	testName      string
	json          bool
}

type exportTestReport struct {
	OK            bool     `json:"ok"`
	RecordingPath string   `json:"recordingPath"`
	OutPath       string   `json:"outPath,omitempty"`
	PackageName   string   `json:"packageName"`
	Component     string   `json:"component"`
	TestName      string   `json:"testName"`
	CommandCount  int      `json:"commandCount"`
	Assertions    []string `json:"assertions,omitempty"`
	Source        string   `json:"source,omitempty"`
}

type exportTestRecording struct {
	Name          string                       `json:"name"`
	PackageName   string                       `json:"package"`
	Component     string                       `json:"component"`
	TestName      string                       `json:"testName"`
	Commands      []exportTestRecordingCommand `json:"commands"`
	FinalSnapshot any                          `json:"finalSnapshot"`
}

type exportTestRecordingCommand struct {
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"payload"`
}

func (parseL launcher) runExportTest(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"recording", "out", "package", "component", "test-name"}, []string{"json"})
	parseFlags := flag.NewFlagSet("export-test", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRecording := parseFlags.String("recording", "", "Path to a gwc live-session recording JSON file")
	parseOut := parseFlags.String("out", "", "Optional Go test file path to write")
	parsePackage := parseFlags.String("package", "", "Package name for the generated test")
	parseComponent := parseFlags.String("component", "", "Root component expression to render")
	parseTestName := parseFlags.String("test-name", "", "Generated Go test name")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON envelope")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseConfig := exportTestConfig{
		recordingPath: *parseRecording,
		outPath:       *parseOut,
		packageName:   *parsePackage,
		component:     *parseComponent,
		testName:      *parseTestName,
		json:          *parseJSON,
	}
	parseReport, parseErr := buildExportTestReport(parseConfig)
	if parseConfig.json {
		parseDiagnostics := []agenticDiagnostic(nil)
		if parseErr != nil {
			parseDiagnostics = append(parseDiagnostics, buildAgenticCommandError("GWC-EXPORT-TEST", parseErr))
		}
		if parseWriteErr := writeAgenticEnvelope("export-test", parseErr == nil, parseReport, parseDiagnostics, parseErr); parseWriteErr != nil {
			return parseWriteErr
		}
	}
	if parseErr != nil {
		return parseErr
	}
	if !parseConfig.json {
		if strings.TrimSpace(parseReport.OutPath) == "" {
			fmt.Print(parseReport.Source)
		} else {
			printAgenticHumanSummary("GWC export-test", true, []string{"out: " + parseReport.OutPath})
		}
	}
	return nil
}

func buildExportTestReport(parseConfig exportTestConfig) (exportTestReport, error) {
	parseReport := exportTestReport{RecordingPath: strings.TrimSpace(parseConfig.recordingPath), OutPath: strings.TrimSpace(parseConfig.outPath)}
	if parseReport.RecordingPath == "" {
		return parseReport, errors.New("export-test requires -recording")
	}
	parseRecording, parseErr := readExportTestRecording(parseReport.RecordingPath)
	if parseErr != nil {
		return parseReport, parseErr
	}
	parsePackage := firstNonEmpty(parseConfig.packageName, parseRecording.PackageName, "main")
	parseComponent := firstNonEmpty(parseConfig.component, parseRecording.Component)
	if parseComponent == "" {
		return parseReport, errors.New("export-test requires -component or recording.component")
	}
	parseTestName := firstNonEmpty(parseConfig.testName, parseRecording.TestName, "TestGWCRecording"+goIdentifierSuffix(firstNonEmpty(parseRecording.Name, "Flow")))
	parseSource, parseAssertions, parseErr := buildExportTestSource(parsePackage, parseComponent, parseTestName, parseRecording)
	if parseErr != nil {
		return parseReport, parseErr
	}
	if parseReport.OutPath != "" {
		if parseErr := os.WriteFile(parseReport.OutPath, []byte(parseSource), 0o644); parseErr != nil {
			return parseReport, fmt.Errorf("write generated test: %w", parseErr)
		}
	}
	parseReport.OK = true
	parseReport.PackageName = parsePackage
	parseReport.Component = parseComponent
	parseReport.TestName = parseTestName
	parseReport.CommandCount = len(parseRecording.Commands)
	parseReport.Assertions = parseAssertions
	parseReport.Source = parseSource
	return parseReport, nil
}

func readExportTestRecording(parsePath string) (exportTestRecording, error) {
	parseRaw, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return exportTestRecording{}, fmt.Errorf("read recording %s: %w", parsePath, parseErr)
	}
	parseRecording := exportTestRecording{}
	if parseErr := json.Unmarshal(parseRaw, &parseRecording); parseErr != nil {
		return exportTestRecording{}, fmt.Errorf("decode recording %s: %w", parsePath, parseErr)
	}
	return parseRecording, nil
}

func buildExportTestSource(parsePackage string, parseComponent string, parseTestName string, parseRecording exportTestRecording) (string, []string, error) {
	parseAssertions := []string{}
	parseUsesStrings := false
	var parseBody bytes.Buffer
	parseBody.WriteString("\tparseFixture := render.New(parseT)\n")
	parseBody.WriteString("\tparseFixture.Render(" + renderRootExpression(parseComponent) + ")\n")
	for _, parseCommand := range parseRecording.Commands {
		parseLine, parseAssert, parseUsesString := exportTestCommandLine(parseCommand)
		if parseLine == "" {
			continue
		}
		parseBody.WriteString(parseLine)
		if parseAssert != "" {
			parseAssertions = append(parseAssertions, parseAssert)
		}
		if parseUsesString {
			parseUsesStrings = true
		}
	}
	for _, parseText := range exportTestSnapshotTexts(parseRecording.FinalSnapshot) {
		parseUsesStrings = true
		parseAssertions = append(parseAssertions, "text:"+parseText)
		parseBody.WriteString("\tif !strings.Contains(parseFixture.Text(), " + strconv.Quote(parseText) + ") {\n")
		parseBody.WriteString("\t\tparseT.Fatalf(\"expected final rendered text to contain %q, got %q\", " + strconv.Quote(parseText) + ", parseFixture.Text())\n")
		parseBody.WriteString("\t}\n")
	}

	var parseSource bytes.Buffer
	parseSource.WriteString("package " + sanitizePackageName(parsePackage) + "\n\n")
	parseSource.WriteString("import (\n")
	if parseUsesStrings {
		parseSource.WriteString("\t\"strings\"\n")
	}
	parseSource.WriteString("\t\"testing\"\n\n")
	parseSource.WriteString("\t\"github.com/monstercameron/GoWebComponents/v4/testkit/render\"\n")
	parseSource.WriteString("\t\"github.com/monstercameron/GoWebComponents/v4/ui\"\n")
	parseSource.WriteString(")\n\n")
	parseSource.WriteString("func " + sanitizeTestName(parseTestName) + "(parseT *testing.T) {\n")
	parseSource.Write(parseBody.Bytes())
	parseSource.WriteString("}\n")
	parseFormatted, parseErr := format.Source(parseSource.Bytes())
	if parseErr != nil {
		return "", nil, fmt.Errorf("format generated test: %w", parseErr)
	}
	return string(parseFormatted), parseAssertions, nil
}

func exportTestCommandLine(parseCommand exportTestRecordingCommand) (string, string, bool) {
	parseName := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(parseCommand.Name)), "bridge.")
	var parsePayload map[string]any
	if len(parseCommand.Payload) > 0 {
		_ = json.Unmarshal(parseCommand.Payload, &parsePayload)
	}
	switch parseName {
	case "emit":
		parseID := firstNonEmpty(exportTestPayloadString(parsePayload, "id", "targetID"), exportTestPayloadStringMap(parsePayload, "selector", "id"))
		parseEvent := firstNonEmpty(exportTestPayloadString(parsePayload, "event", "type"), "click")
		if parseID == "" {
			return "\tparseFixture.Stabilize()\n", "", false
		}
		parseEventLiteral := exportTestEventLiteral(parsePayload)
		return "\tparseFixture.DispatchByID(" + strconv.Quote(parseID) + ", " + strconv.Quote(parseEvent) + ", " + parseEventLiteral + ")\n\tparseFixture.Stabilize()\n", "dispatch:" + parseID + ":" + parseEvent, false
	case "wait-for":
		parseSelector := exportTestPayloadMap(parsePayload, "query", "selector")
		parseLine, parseAssert := exportTestSelectorAssertion(parseSelector)
		if parseLine != "" {
			return parseLine, parseAssert, false
		}
		return "\tparseFixture.Stabilize()\n", "settle", false
	case "query":
		parseLine, parseAssert := exportTestSelectorAssertion(parsePayload)
		return parseLine, parseAssert, false
	default:
		return "\tparseFixture.Stabilize()\n", "", false
	}
}

func exportTestSelectorAssertion(parseSelector map[string]any) (string, string) {
	parseID := exportTestPayloadString(parseSelector, "id", "targetID")
	if parseID != "" {
		return "\tif parseFixture.ByID(" + strconv.Quote(parseID) + ") == nil {\n\t\tparseT.Fatalf(\"expected node id %q\", " + strconv.Quote(parseID) + ")\n\t}\n", "id:" + parseID
	}
	parseText := exportTestPayloadString(parseSelector, "text")
	if parseText != "" {
		return "\tif parseFixture.ByText(" + strconv.Quote(parseText) + ") == nil {\n\t\tparseT.Fatalf(\"expected text %q\", " + strconv.Quote(parseText) + ")\n\t}\n", "text:" + parseText
	}
	parseRole := exportTestPayloadString(parseSelector, "role")
	if parseRole != "" {
		parseName := exportTestPayloadString(parseSelector, "name", "label")
		return "\tif parseFixture.ByRole(" + strconv.Quote(parseRole) + ", " + strconv.Quote(parseName) + ") == nil {\n\t\tparseT.Fatalf(\"expected role %q named %q\", " + strconv.Quote(parseRole) + ", " + strconv.Quote(parseName) + ")\n\t}\n", "role:" + parseRole + ":" + parseName
	}
	return "", ""
}

func exportTestEventLiteral(parsePayload map[string]any) string {
	parseFields := []string{}
	if parseValue := exportTestPayloadString(parsePayload, "value"); parseValue != "" {
		parseFields = append(parseFields, "Value: "+strconv.Quote(parseValue))
	}
	if parseChecked, parseOK := exportTestPayloadBool(parsePayload, "checked"); parseOK {
		parseFields = append(parseFields, fmt.Sprintf("Checked: %t", parseChecked))
	}
	if parseKey := exportTestPayloadString(parsePayload, "key"); parseKey != "" {
		parseFields = append(parseFields, "Key: "+strconv.Quote(parseKey))
	}
	if parseKeyCode, parseOK := exportTestPayloadInt(parsePayload, "keyCode"); parseOK {
		parseFields = append(parseFields, fmt.Sprintf("KeyCode: %d", parseKeyCode))
	}
	if len(parseFields) == 0 {
		return "render.Event{}"
	}
	return "render.Event{" + strings.Join(parseFields, ", ") + "}"
}

func exportTestPayloadMap(parsePayload map[string]any, parseKeys ...string) map[string]any {
	for _, parseKey := range parseKeys {
		if parseValue, parseOK := parsePayload[parseKey].(map[string]any); parseOK {
			return parseValue
		}
	}
	return map[string]any{}
}

func exportTestPayloadStringMap(parsePayload map[string]any, parseMapKey string, parseKey string) string {
	return exportTestPayloadString(exportTestPayloadMap(parsePayload, parseMapKey), parseKey)
}

func exportTestPayloadString(parsePayload map[string]any, parseKeys ...string) string {
	for _, parseKey := range parseKeys {
		if parseValue, parseOK := parsePayload[parseKey].(string); parseOK && strings.TrimSpace(parseValue) != "" {
			return strings.TrimSpace(parseValue)
		}
	}
	return ""
}

func exportTestPayloadBool(parsePayload map[string]any, parseKey string) (bool, bool) {
	parseValue, parseOK := parsePayload[parseKey].(bool)
	return parseValue, parseOK
}

func exportTestPayloadInt(parsePayload map[string]any, parseKey string) (int, bool) {
	switch parseValue := parsePayload[parseKey].(type) {
	case float64:
		return int(parseValue), true
	case int:
		return parseValue, true
	default:
		return 0, false
	}
}

func exportTestSnapshotTexts(parseValue any) []string {
	parseSeen := map[string]bool{}
	var parseTexts []string
	var parseWalk func(any)
	parseWalk = func(parseCurrent any) {
		switch parseTyped := parseCurrent.(type) {
		case map[string]any:
			if parseText, parseOK := parseTyped["text"].(string); parseOK && strings.TrimSpace(parseText) != "" && !parseSeen[parseText] {
				parseSeen[parseText] = true
				parseTexts = append(parseTexts, parseText)
			}
			for _, parseChild := range parseTyped {
				parseWalk(parseChild)
			}
		case []any:
			for _, parseChild := range parseTyped {
				parseWalk(parseChild)
			}
		}
	}
	parseWalk(parseValue)
	sort.Strings(parseTexts)
	if len(parseTexts) > 8 {
		parseTexts = parseTexts[:8]
	}
	return parseTexts
}

func renderRootExpression(parseComponent string) string {
	parseComponent = strings.TrimSpace(parseComponent)
	if strings.HasPrefix(parseComponent, "ui.CreateElement(") {
		return parseComponent
	}
	return "ui.CreateElement(" + parseComponent + ")"
}

func sanitizePackageName(parsePackage string) string {
	parsePackage = strings.TrimSpace(parsePackage)
	if parsePackage == "" {
		return "main"
	}
	var parseBuilder strings.Builder
	for parseIndex, parseRune := range parsePackage {
		if parseRune == '_' || parseRune >= 'A' && parseRune <= 'Z' || parseRune >= 'a' && parseRune <= 'z' || parseIndex > 0 && parseRune >= '0' && parseRune <= '9' {
			parseBuilder.WriteRune(parseRune)
		}
	}
	if parseBuilder.Len() == 0 {
		return "main"
	}
	return parseBuilder.String()
}

func sanitizeTestName(parseName string) string {
	parseName = goIdentifierSuffix(parseName)
	if parseName == "" || !strings.HasPrefix(parseName, "Test") {
		parseName = "Test" + parseName
	}
	return parseName
}

func goIdentifierSuffix(parseValue string) string {
	parseParts := strings.FieldsFunc(parseValue, func(parseRune rune) bool {
		return !(parseRune >= 'A' && parseRune <= 'Z' || parseRune >= 'a' && parseRune <= 'z' || parseRune >= '0' && parseRune <= '9')
	})
	var parseBuilder strings.Builder
	for _, parsePart := range parseParts {
		if parsePart == "" {
			continue
		}
		parseBuilder.WriteString(strings.ToUpper(parsePart[:1]))
		if len(parsePart) > 1 {
			parseBuilder.WriteString(parsePart[1:])
		}
	}
	if parseBuilder.Len() == 0 {
		return "Flow"
	}
	return parseBuilder.String()
}
