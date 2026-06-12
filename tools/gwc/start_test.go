package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDefaultTargetDirUsesGeneratedRoot(parseT *testing.T) {
	parseProjectName := "starter-app"
	parseTargetDir := defaultTargetDir(parseProjectName)
	parseGeneratedRoot := defaultGeneratedScaffoldRoot()

	if !strings.HasPrefix(filepath.Clean(parseTargetDir), filepath.Clean(parseGeneratedRoot)) {
		parseT.Fatalf("expected target dir %q to live under generated root %q", parseTargetDir, parseGeneratedRoot)
	}
	if filepath.Base(parseTargetDir) != parseProjectName {
		parseT.Fatalf("expected target dir base %q to match project name %q", filepath.Base(parseTargetDir), parseProjectName)
	}
}

func TestScaffoldMetadataJSONWireShapePreservesNestedObjects(parseT *testing.T) {
	parseEncoded, parseErr := json.Marshal(scaffoldMetadata{})
	if parseErr != nil {
		parseT.Fatalf("marshal scaffold metadata: %v", parseErr)
	}
	parseText := string(parseEncoded)
	for _, parseExpected := range []string{`"preset":{}`, `"enterprise":{}`, `"ownership":{}`, `"tooling":{}`} {
		if !strings.Contains(parseText, parseExpected) {
			parseT.Fatalf("expected scaffold metadata to preserve %s, got %s", parseExpected, parseText)
		}
	}
}

func TestDefaultGeneratedScaffoldRootForWindowsUsesDocuments(parseT *testing.T) {
	parseHomeDir := filepath.Join("C:\\Users", "Cam")
	parseGot := defaultGeneratedScaffoldRootForOS("windows", parseHomeDir, nil)
	parseWant := filepath.Join(parseHomeDir, "Documents")
	if parseGot != parseWant {
		parseT.Fatalf("expected windows scaffold root %q, got %q", parseWant, parseGot)
	}
}

func TestDefaultGeneratedScaffoldRootForDarwinUsesDocuments(parseT *testing.T) {
	parseHomeDir := filepath.Join("/Users", "cam")
	parseGot := defaultGeneratedScaffoldRootForOS("darwin", parseHomeDir, nil)
	parseWant := filepath.Join(parseHomeDir, "Documents")
	if parseGot != parseWant {
		parseT.Fatalf("expected darwin scaffold root %q, got %q", parseWant, parseGot)
	}
}

func TestDefaultGeneratedScaffoldRootForLinuxFallsBackFromDocuments(parseT *testing.T) {
	parseHomeDir := filepath.Join("/home", "cam")
	parseDocumentsDir := filepath.Join(parseHomeDir, "Documents")
	parseProjectsDir := filepath.Join(parseHomeDir, "Projects")

	parseGot := defaultGeneratedScaffoldRootForOS("linux", parseHomeDir, func(parsePath string) bool {
		return parsePath == parseDocumentsDir
	})
	if parseGot != parseDocumentsDir {
		parseT.Fatalf("expected linux scaffold root to prefer Documents, got %q", parseGot)
	}

	parseGot = defaultGeneratedScaffoldRootForOS("linux", parseHomeDir, func(parsePath2 string) bool {
		return parsePath2 == parseProjectsDir
	})
	if parseGot != parseProjectsDir {
		parseT.Fatalf("expected linux scaffold root to fall back to Projects, got %q", parseGot)
	}

	parseGot = defaultGeneratedScaffoldRootForOS("linux", parseHomeDir, func(parsePath3 string) bool { return false })
	if parseGot != parseHomeDir {
		parseT.Fatalf("expected linux scaffold root to fall back to home dir, got %q", parseGot)
	}
}

func TestValidateProjectFolderNameRejectsInvalidCharacters(parseT *testing.T) {
	if parseErr := validateProjectFolderName(`bad:name`); parseErr == nil {
		parseT.Fatal("expected invalid project folder name to be rejected")
	}
}

func TestValidateProjectFolderNameRejectsDotNames(parseT *testing.T) {
	for _, parseProjectName := range []string{".", ".."} {
		if parseErr := validateProjectFolderName(parseProjectName); parseErr == nil {
			parseT.Fatalf("expected project name %q to be rejected", parseProjectName)
		}
	}
}

func TestDefaultDescriptionUsesPresetSummary(parseT *testing.T) {
	parsePreset := startPreset{Summary: "Small scaffold summary"}
	if parseGot := defaultDescription(parsePreset); parseGot != parsePreset.Summary {
		parseT.Fatalf("expected default description %q, got %q", parsePreset.Summary, parseGot)
	}
}

func TestDefaultStartPresetsCoverMajorAdoptionModes(parseT *testing.T) {
	parsePresets := defaultStartPresets()
	parseByKey := make(map[string]startPreset, len(parsePresets))
	for _, parsePreset := range parsePresets {
		parseByKey[parsePreset.Key] = parsePreset
	}

	parseRequired := map[string][]string{
		"minimal-client":   {"ui", "html"},
		"routed-spa":       {"router", "browser-tests"},
		"ssr-app":          {"ssr", "hydration"},
		"reference-app":    {"router", "forms", "fetch", "state", "browser-tests"},
		"dashboard-app":    {"router", "fetch", "state", "forms", "browser-tests"},
		"marketing-site":   {"ssr", "hydration", "release-profile"},
		"content-blog":     {"router", "ssr", "hydration", "fetch"},
		"authed-app-shell": {"router", "forms", "fetch", "state", "browser-tests"},
	}
	for parseKey, parseExpectedFeatures := range parseRequired {
		parsePreset2, parseOk := parseByKey[parseKey]
		if !parseOk {
			parseT.Fatalf("expected preset %q to exist", parseKey)
		}
		parseFeatureSet := make(map[string]struct{}, len(parsePreset2.Features))
		for _, parseFeature := range parsePreset2.Features {
			parseFeatureSet[parseFeature] = struct{}{}
		}
		for _, parseExpectedFeature := range parseExpectedFeatures {
			if _, parseOk2 := parseFeatureSet[parseExpectedFeature]; !parseOk2 {
				parseT.Fatalf("expected preset %q to include feature %q", parseKey, parseExpectedFeature)
			}
		}
	}
}

func TestProjectNameWithWordSelectorPrefixesPresetSlug(parseT *testing.T) {
	parsePreset := startPreset{Key: "minimal-client"}
	parseSelector := sequentialWordSelector(2, 5)
	parseGot := projectNameWithWordSelector(parsePreset, parseSelector)
	parseWant := techProjectWords[2] + "-" + salesProjectWords[5] + "-minimal-client"
	if parseGot != parseWant {
		parseT.Fatalf("expected generated project name %q, got %q", parseWant, parseGot)
	}
}

func TestProjectNameWithWordSelectorFallsBackToAppBase(parseT *testing.T) {
	parseGot := projectNameWithWordSelector(startPreset{}, sequentialWordSelector(0, 0))
	parseWant := techProjectWords[0] + "-" + salesProjectWords[0] + "-app"
	if parseGot != parseWant {
		parseT.Fatalf("expected fallback generated project name %q, got %q", parseWant, parseGot)
	}
}

func TestValidateStartTerminalRequiresInteractiveTTY(parseT *testing.T) {
	parseTests := []struct {
		name             string
		stdinIsTerminal  bool
		stdoutIsTerminal bool
		wantErr          bool
	}{
		{name: "stdin and stdout interactive", stdinIsTerminal: true, stdoutIsTerminal: true},
		{name: "stdin not interactive", stdoutIsTerminal: true, wantErr: true},
		{name: "stdout not interactive", stdinIsTerminal: true, wantErr: true},
		{name: "neither interactive", wantErr: true},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			parseErr := validateStartTerminal(parseTest.stdinIsTerminal, parseTest.stdoutIsTerminal)
			if parseTest.wantErr {
				if parseErr == nil {
					parseT2.Fatal("expected non-interactive terminal validation to fail")
				}
				if !strings.Contains(parseErr.Error(), "interactive terminal") {
					parseT2.Fatalf("expected actionable interactive terminal error, got %v", parseErr)
				}
				return
			}
			if parseErr != nil {
				parseT2.Fatalf("expected interactive terminal validation to pass, got %v", parseErr)
			}
		})
	}
}

func TestStartPostModelSelectedChoiceDefaultsToExit(parseT *testing.T) {
	parseModel := startPostModel{}
	if parseChoice := parseModel.selectedChoice(); parseChoice != startPostChoiceExit {
		parseT.Fatalf("expected default post-start choice to be exit, got %v", parseChoice)
	}

	parseModel.options = []startPostChoice{startPostChoiceExit, startPostChoiceRunDev}
	parseModel.cursor = 1
	if parseChoice2 := parseModel.selectedChoice(); parseChoice2 != startPostChoiceRunDev {
		parseT.Fatalf("expected second post-start choice to be run-dev, got %v", parseChoice2)
	}
}

func TestStartModelInitAndViewRouting(parseT *testing.T) {
	parseModel := newTestStartModel()
	if parseCmd := parseModel.Init(); parseCmd == nil {
		parseT.Fatal("expected start model init command")
	}
	if parseView := parseModel.View(); !strings.Contains(parseView, "Step 1 of 3") || !strings.Contains(parseView, parseModel.presets[0].Name) {
		parseT.Fatalf("expected preset picker view, got %q", parseView)
	}

	parseModel.step = startStepProject
	parseModel.selection.Preset = parseModel.presets[0]
	if parseView2 := parseModel.View(); !strings.Contains(parseView2, "Step 2 of 3") || !strings.Contains(parseView2, "Project name") {
		parseT.Fatalf("expected project form view, got %q", parseView2)
	}

	parseModel.step = startStepConfirm
	if parseView3 := parseModel.View(); !strings.Contains(parseView3, "Step 3 of 3") || !strings.Contains(parseView3, "Included baseline features") {
		parseT.Fatalf("expected confirmation view, got %q", parseView3)
	}

	parseModel.quitting = true
	if parseView4 := parseModel.View(); parseView4 != "\n" {
		parseT.Fatalf("expected quitting view newline, got %q", parseView4)
	}
}

func TestStartModelEnterpriseStepSelection(parseT *testing.T) {
	parseModel := newTestStartModel()
	parseModel.step = startStepEnterprise
	parseModel.enterpriseSections = []launcherPluginScaffoldSection{
		{Title: "Org baseline", Summary: "Enterprise profile", Features: []string{"release-profile", "browser-tests"}},
	}
	parseModel.enterpriseEnabled = []bool{false}

	if parseView := parseModel.renderEnterpriseSections(); !strings.Contains(parseView, "optional enterprise plugin sections") {
		parseT.Fatalf("expected enterprise sections view, got %q", parseView)
	}

	parseUpdatedModel, _ := parseModel.updateEnterpriseStep(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	parseUpdated := parseUpdatedModel.(startModel)
	if !parseUpdated.enterpriseEnabled[0] {
		parseT.Fatal("expected space to toggle enterprise section selection")
	}

	parseUpdatedModel, _ = parseUpdated.updateEnterpriseStep(tea.KeyMsg{Type: tea.KeyEnter})
	parseUpdated = parseUpdatedModel.(startModel)
	if parseUpdated.step != startStepProject {
		parseT.Fatalf("expected enter to advance from enterprise step to project step, got %v", parseUpdated.step)
	}
	if len(parseUpdated.selection.EnabledEnterpriseSections) != 1 || parseUpdated.selection.EnabledEnterpriseSections[0] != "Org baseline" {
		parseT.Fatalf("expected enabled enterprise section to be stored, got %#v", parseUpdated.selection.EnabledEnterpriseSections)
	}
	if parseGot := strings.Join(parseUpdated.selection.EnterpriseFeatures, ","); parseGot != "release-profile,browser-tests" {
		parseT.Fatalf("expected selected enterprise features to be captured, got %q", parseGot)
	}
}

func TestStartModelUpdateHandlesWindowAndQuit(parseT *testing.T) {
	parseModel := newTestStartModel()
	parseUpdatedModel, parseCmd := parseModel.Update(tea.WindowSizeMsg{Width: 123})
	parseUpdated := parseUpdatedModel.(startModel)
	if parseUpdated.width != 123 {
		parseT.Fatalf("expected width 123, got %d", parseUpdated.width)
	}
	if parseCmd != nil {
		parseT.Fatalf("expected no command for window resize, got %#v", parseCmd)
	}

	parseUpdatedModel, parseCmd = parseModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	parseUpdated = parseUpdatedModel.(startModel)
	if !parseUpdated.quitting {
		parseT.Fatal("expected q to mark model quitting")
	}
	if parseCmd == nil {
		parseT.Fatal("expected quit command for q")
	}
}

func TestStartModelUpdateDispatchAndFallbackBranches(parseT *testing.T) {
	parseModel := newTestStartModel()
	parseModel.step = startStepProject
	parseModel.selection.Preset = parseModel.presets[0]
	parseUpdatedModel, _ := parseModel.Update(tea.KeyMsg{Type: tea.KeyTab})
	parseUpdated := parseUpdatedModel.(startModel)
	if parseUpdated.inputIndex != 1 {
		parseT.Fatalf("expected project-step dispatch to advance input focus, got %d", parseUpdated.inputIndex)
	}

	parseModel = newTestStartModel()
	parseModel.step = startStepConfirm
	parseUpdatedModel, parseCmd := parseModel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	parseUpdated = parseUpdatedModel.(startModel)
	if !parseUpdated.confirmed || parseCmd == nil {
		parseT.Fatal("expected confirm-step dispatch to confirm and quit")
	}

	parseModel = newTestStartModel()
	parseModel.step = startStep(99)
	parseUpdatedModel, parseCmd = parseModel.Update(tea.MouseMsg{})
	parseUpdated = parseUpdatedModel.(startModel)
	if parseUpdated.step != startStep(99) || parseCmd != nil {
		parseT.Fatalf("expected unknown step fallback to return unchanged model and nil cmd, got step=%v cmd=%#v", parseUpdated.step, parseCmd)
	}

	parseModel = newTestStartModel()
	parseUpdatedModel, parseCmd = parseModel.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	parseUpdated = parseUpdatedModel.(startModel)
	if !parseUpdated.quitting || parseCmd == nil {
		parseT.Fatal("expected ctrl+c to quit start model")
	}
}

func TestUpdatePresetStepNavigationAndEnter(parseT *testing.T) {
	parseModel := newTestStartModel()
	parseModel.cursor = 1
	parseUpdatedModel, _ := parseModel.updatePresetStep(tea.KeyMsg{Type: tea.KeyUp})
	parseUpdated := parseUpdatedModel.(startModel)
	if parseUpdated.cursor != 0 {
		parseT.Fatalf("expected cursor to move up to 0, got %d", parseUpdated.cursor)
	}

	parseUpdatedModel, _ = parseUpdated.updatePresetStep(tea.KeyMsg{Type: tea.KeyDown})
	parseUpdated = parseUpdatedModel.(startModel)
	if parseUpdated.cursor != 1 {
		parseT.Fatalf("expected cursor to move down to 1, got %d", parseUpdated.cursor)
	}

	parseT.Setenv("GIT_AUTHOR_NAME", "Preset Author")

	parseUpdated.errText = "stale"
	parseUpdatedModel, parseCmd := parseUpdated.updatePresetStep(tea.KeyMsg{Type: tea.KeyEnter})
	parseUpdated = parseUpdatedModel.(startModel)
	if parseUpdated.step != startStepProject {
		parseT.Fatalf("expected enter to advance to project step, got %v", parseUpdated.step)
	}
	if parseUpdated.selection.Preset.Key != parseUpdated.presets[parseUpdated.cursor].Key {
		parseT.Fatalf("expected selected preset to be stored, got %#v", parseUpdated.selection.Preset)
	}
	if !strings.HasSuffix(parseUpdated.inputs[0].Value(), "-"+parseUpdated.selection.Preset.Key) {
		parseT.Fatalf("expected seeded project name to include preset key, got %q", parseUpdated.inputs[0].Value())
	}
	if parseUpdated.inputs[1].Value() != "github.com/your-org/"+parseUpdated.inputs[0].Value() {
		parseT.Fatalf("expected seeded module path to derive from project name, got %q", parseUpdated.inputs[1].Value())
	}
	if parseUpdated.inputs[2].Value() != "Preset Author" || parseUpdated.inputs[3].Value() != "0.1.0" {
		parseT.Fatalf("expected seeded author/version defaults, got author=%q version=%q", parseUpdated.inputs[2].Value(), parseUpdated.inputs[3].Value())
	}
	if parseUpdated.inputs[4].Value() != parseUpdated.selection.Preset.Summary {
		parseT.Fatalf("expected seeded description from preset summary, got %q", parseUpdated.inputs[4].Value())
	}
	if parseUpdated.errText != "" {
		parseT.Fatalf("expected errText to clear, got %q", parseUpdated.errText)
	}
	if parseCmd == nil {
		parseT.Fatal("expected blink command after preset selection")
	}
}

func TestUpdateProjectStepNavigationAndSubmission(parseT *testing.T) {
	parseModel := newTestStartModel()
	parseModel.step = startStepProject
	parseModel.selection.Preset = parseModel.presets[0]
	parseModel.focusInput(0)

	parseUpdatedModel, _ := parseModel.updateProjectStep(tea.KeyMsg{Type: tea.KeyTab})
	parseUpdated := parseUpdatedModel.(startModel)
	if parseUpdated.inputIndex != 1 {
		parseT.Fatalf("expected tab to advance focus to 1, got %d", parseUpdated.inputIndex)
	}

	parseUpdatedModel, _ = parseUpdated.updateProjectStep(tea.KeyMsg{Type: tea.KeyShiftTab})
	parseUpdated = parseUpdatedModel.(startModel)
	if parseUpdated.inputIndex != 0 {
		parseT.Fatalf("expected shift+tab to move focus back to 0, got %d", parseUpdated.inputIndex)
	}

	parseUpdated.errText = "problem"
	parseUpdatedModel, _ = parseUpdated.updateProjectStep(tea.KeyMsg{Type: tea.KeyEsc})
	parseUpdated = parseUpdatedModel.(startModel)
	if parseUpdated.step != startStepPreset || parseUpdated.errText != "" {
		parseT.Fatalf("expected esc to return to preset step and clear error, got step=%v err=%q", parseUpdated.step, parseUpdated.errText)
	}

	parseModel = newTestStartModel()
	parseModel.step = startStepProject
	parseModel.selection.Preset = parseModel.presets[0]
	parseModel.inputs[0].SetValue("app")
	parseModel.inputs[1].SetValue("example.com/app")
	parseModel.inputs[2].SetValue("Cam")
	parseModel.inputs[3].SetValue("1.0.0")
	parseModel.inputs[4].SetValue("Desc")
	parseModel.focusInput(4)
	parseUpdatedModel, _ = parseModel.updateProjectStep(tea.KeyMsg{Type: tea.KeyEnter})
	parseUpdated = parseUpdatedModel.(startModel)
	if parseUpdated.step != startStepConfirm {
		parseT.Fatalf("expected enter on final input to advance to confirm, got %v", parseUpdated.step)
	}
	if parseUpdated.selection.ProjectName != "app" {
		parseT.Fatalf("expected built selection to be stored, got %#v", parseUpdated.selection)
	}

	parseModel = newTestStartModel()
	parseModel.step = startStepProject
	parseModel.selection.Preset = parseModel.presets[0]
	parseModel.inputs[0].SetValue("")
	parseModel.focusInput(4)
	parseUpdatedModel, _ = parseModel.updateProjectStep(tea.KeyMsg{Type: tea.KeyEnter})
	parseUpdated = parseUpdatedModel.(startModel)
	if !strings.Contains(parseUpdated.errText, "project name is required") {
		parseT.Fatalf("expected validation error, got %q", parseUpdated.errText)
	}
}

func TestUpdateConfirmStepAndViews(parseT *testing.T) {
	parseModel := newTestStartModel()
	parseModel.step = startStepConfirm
	parseModel.selection.Preset = parseModel.presets[0]
	parseModel.errText = "stale"
	parseUpdatedModel, _ := parseModel.updateConfirmStep(tea.KeyMsg{Type: tea.KeyEsc})
	parseUpdated := parseUpdatedModel.(startModel)
	if parseUpdated.step != startStepProject || parseUpdated.errText != "" {
		parseT.Fatalf("expected esc to return to project step and clear error, got step=%v err=%q", parseUpdated.step, parseUpdated.errText)
	}

	parseUpdatedModel, parseCmd := parseModel.updateConfirmStep(tea.KeyMsg{Type: tea.KeyEnter})
	parseUpdated = parseUpdatedModel.(startModel)
	if !parseUpdated.confirmed {
		parseT.Fatal("expected enter to confirm start model")
	}
	if parseCmd == nil {
		parseT.Fatal("expected quit command on confirm enter")
	}
}

func TestStartPostModelUpdateAndView(parseT *testing.T) {
	parseModel := startPostModel{
		selection: startSelection{Preset: startPreset{Name: "Minimal"}, ProjectName: "app", ModulePath: "example.com/app", Author: "Cam", Version: "1.0.0"},
		result:    &scaffoldResult{TargetDir: `C:\tmp\app`, AppPath: `C:\tmp\app\main.go`, HTMLPath: `C:\tmp\app\index.html`},
		options:   []startPostChoice{startPostChoiceExit, startPostChoiceRunDev},
	}
	if parseCmd := parseModel.Init(); parseCmd != nil {
		parseT.Fatalf("expected nil init command, got %#v", parseCmd)
	}
	if parseView := parseModel.View(); !strings.Contains(parseView, "Scaffold generated successfully.") || !strings.Contains(parseView, "Run the generated scaffold in the dev server now") {
		parseT.Fatalf("expected success view, got %q", parseView)
	}
	if parseGot := parseModel.optionLabel(startPostChoiceExit); !strings.Contains(parseGot, "Exit") {
		parseT.Fatalf("expected exit label, got %q", parseGot)
	}
	if parseGot2 := parseModel.optionLabel(startPostChoiceRunDev); !strings.Contains(parseGot2, "dev server") {
		parseT.Fatalf("expected run-dev label, got %q", parseGot2)
	}

	parseUpdatedModel, _ := parseModel.Update(tea.KeyMsg{Type: tea.KeyDown})
	parseUpdated := parseUpdatedModel.(startPostModel)
	if parseUpdated.cursor != 1 {
		parseT.Fatalf("expected down to move cursor to 1, got %d", parseUpdated.cursor)
	}
	parseUpdatedModel, _ = parseUpdated.Update(tea.KeyMsg{Type: tea.KeyUp})
	parseUpdated = parseUpdatedModel.(startPostModel)
	if parseUpdated.cursor != 0 {
		parseT.Fatalf("expected up to move cursor to 0, got %d", parseUpdated.cursor)
	}
	parseUpdatedModel, parseCmd2 := parseUpdated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	parseUpdated = parseUpdatedModel.(startPostModel)
	if !parseUpdated.confirmed || parseCmd2 == nil {
		parseT.Fatal("expected enter to confirm and quit post model")
	}

	parseModel.errText = "failed"
	if parseView2 := parseModel.View(); !strings.Contains(parseView2, "Scaffold generation failed.") {
		parseT.Fatalf("expected error view, got %q", parseView2)
	}
	parseUpdatedModel, parseCmd2 = parseModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	parseUpdated = parseUpdatedModel.(startPostModel)
	if !parseUpdated.confirmed || parseCmd2 == nil {
		parseT.Fatal("expected esc to confirm-close post model")
	}

	parseModel.quitting = true
	if parseView3 := parseModel.View(); parseView3 != "\n" {
		parseT.Fatalf("expected quitting newline view, got %q", parseView3)
	}
}

func TestStartPostModelUpdateAdditionalBranches(parseT *testing.T) {
	parseModel := startPostModel{options: []startPostChoice{startPostChoiceExit, startPostChoiceRunDev}, cursor: 1}
	parseUpdatedModel, _ := parseModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	parseUpdated := parseUpdatedModel.(startPostModel)
	if parseUpdated.cursor != 0 {
		parseT.Fatalf("expected k to move cursor up, got %d", parseUpdated.cursor)
	}

	parseUpdatedModel, _ = parseUpdated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	parseUpdated = parseUpdatedModel.(startPostModel)
	if parseUpdated.cursor != 1 {
		parseT.Fatalf("expected j to move cursor down, got %d", parseUpdated.cursor)
	}

	parseUpdated.errText = "failed"
	parseUpdatedModel, _ = parseUpdated.Update(tea.KeyMsg{Type: tea.KeyDown})
	parseUpdated = parseUpdatedModel.(startPostModel)
	if parseUpdated.cursor != 1 {
		parseT.Fatalf("expected errText branch to suppress movement, got %d", parseUpdated.cursor)
	}

	parseUpdatedModel, parseCmd := parseUpdated.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	parseUpdated = parseUpdatedModel.(startPostModel)
	if !parseUpdated.quitting || parseCmd == nil {
		parseT.Fatal("expected ctrl+c to quit post model")
	}

	parseUpdatedModel, parseCmd = startPostModel{}.Update(tea.MouseMsg{})
	parseUpdated = parseUpdatedModel.(startPostModel)
	if parseUpdated.cursor != 0 || parseCmd != nil {
		parseT.Fatalf("expected non-key post update to no-op, got %#v cmd=%#v", parseUpdated, parseCmd)
	}
}

func TestStartHelperDefaultsAndUtilities(parseT *testing.T) {
	if isInteractiveFile(nil) {
		parseT.Fatal("expected nil file to be non-interactive")
	}
	if parseGot := defaultModulePath(""); parseGot != "github.com/your-org/my-app" {
		parseT.Fatalf("expected default module path fallback, got %q", parseGot)
	}
	parseT.Setenv("GIT_AUTHOR_NAME", "Git Author")
	if parseGot2 := defaultAuthor(); parseGot2 != "Git Author" {
		parseT.Fatalf("expected env-based author, got %q", parseGot2)
	}
	if parseGot3 := defaultVersion(); parseGot3 != "0.1.0" {
		parseT.Fatalf("expected default version, got %q", parseGot3)
	}
	if parseGot4 := defaultProjectName(startPreset{Key: "minimal-client"}); !strings.HasSuffix(parseGot4, "-minimal-client") {
		parseT.Fatalf("expected random project name to use preset key, got %q", parseGot4)
	}
	parseSelector := newRandomWordSelector()
	if parseSelector(0) != 0 {
		parseT.Fatal("expected random selector with zero limit to return 0")
	}
	if parseGot5 := pickProjectWord(nil, nil); parseGot5 != "app" {
		parseT.Fatalf("expected empty word list fallback, got %q", parseGot5)
	}
	if parseGot6 := pickProjectWord([]string{"alpha", "beta"}, func(parseLimit int) int { return -1 }); parseGot6 != "beta" {
		parseT.Fatalf("expected negative index normalization, got %q", parseGot6)
	}

	parseReader, parseWriter, parseErr := os.Pipe()
	if parseErr != nil {
		parseT.Fatalf("pipe: %v", parseErr)
	}
	defer parseReader.Close()
	defer parseWriter.Close()
	if isInteractiveFile(parseReader) {
		parseT.Fatal("expected pipe file to be non-interactive")
	}

	parseModel := newTestStartModel()
	parseModel.selection.Preset = parseModel.presets[0]
	parseModel.inputs[0].SetValue("project")
	parseModel.inputs[1].SetValue("example.com/project")
	parseModel.inputs[2].SetValue("Cam")
	parseModel.inputs[3].SetValue("1.0.0")
	parseModel.inputs[4].SetValue("Desc")
	parseModel.focusInput(-1)
	if parseModel.inputIndex != len(parseModel.inputs)-1 {
		parseT.Fatalf("expected negative focus index to wrap, got %d", parseModel.inputIndex)
	}
	parseModel.focusInput(len(parseModel.inputs))
	if parseModel.inputIndex != 0 {
		parseT.Fatalf("expected overflow focus index to wrap to 0, got %d", parseModel.inputIndex)
	}
	if parseView := parseModel.renderPresetPicker(); !strings.Contains(parseView, "Keys: up/down") {
		parseT.Fatalf("expected preset picker help text, got %q", parseView)
	}
	parseModel.errText = "bad input"
	if parseView2 := parseModel.renderProjectForm(); !strings.Contains(parseView2, "Error: bad input") {
		parseT.Fatalf("expected project form error text, got %q", parseView2)
	}
	if parseView3 := parseModel.renderConfirmation(); !strings.Contains(parseView3, "user-owned project outside the framework repo") {
		parseT.Fatalf("expected confirmation copy, got %q", parseView3)
	}
}

func TestStartHelperAdditionalFallbacks(parseT *testing.T) {
	for _, parseKey := range []string{"GIT_AUTHOR_NAME", "GITHUB_USER", "USERNAME", "USER"} {
		parseT.Setenv(parseKey, "")
	}
	if parseGot := defaultAuthor(); parseGot != "Your Name" {
		parseT.Fatalf("expected default author fallback, got %q", parseGot)
	}
	if parseGot2 := defaultDescription(startPreset{}); parseGot2 != "Starter app generated by gwc start" {
		parseT.Fatalf("expected default description fallback, got %q", parseGot2)
	}
	if parseGot3 := pickProjectWord([]string{"alpha", "beta"}, nil); parseGot3 != "alpha" {
		parseT.Fatalf("expected nil selector to pick first word, got %q", parseGot3)
	}
	if parseGot4 := (startPostModel{options: []startPostChoice{startPostChoiceExit}, cursor: 9}).selectedChoice(); parseGot4 != startPostChoiceExit {
		parseT.Fatalf("expected out-of-range selected choice to fall back to exit, got %v", parseGot4)
	}
	if parseGot5 := (startPostModel{options: []startPostChoice{startPostChoiceExit}, cursor: -1}).selectedChoice(); parseGot5 != startPostChoiceExit {
		parseT.Fatalf("expected negative selected choice to fall back to exit, got %v", parseGot5)
	}
	if parseGot6 := defaultGeneratedScaffoldRootForOS("plan9", "", nil); parseGot6 != filepath.Join("gwc-projects") {
		parseT.Fatalf("expected empty-home scaffold root fallback, got %q", parseGot6)
	}
}

func TestDefaultGeneratedScaffoldRootUsesFallbackSources(parseT *testing.T) {
	parseOriginalHomeDir := startUserHomeDir
	parseT.Cleanup(func() {
		startUserHomeDir = parseOriginalHomeDir
	})

	parseT.Run("cwd fallback", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseOriginalWD, parseErr := os.Getwd()
		if parseErr != nil {
			parseT2.Fatalf("get working dir: %v", parseErr)
		}
		if parseErr2 := os.Chdir(parseRoot); parseErr2 != nil {
			parseT2.Fatalf("chdir temp root: %v", parseErr2)
		}
		defer func() {
			_ = os.Chdir(parseOriginalWD)
		}()
		startUserHomeDir = func() (string, error) { return "", errors.New("no home") }
		parseGot := defaultGeneratedScaffoldRoot()
		parseWant := filepath.Join(parseRoot, "gwc-projects")
		if parseGot != parseWant {
			parseT2.Fatalf("expected cwd fallback %q, got %q", parseWant, parseGot)
		}
	})
}

func TestLoadScaffoldMetadataRejectsInvalidJSON(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-start.json"), []byte(`{"tooling":`), 0644); parseErr != nil {
		parseT.Fatalf("write invalid metadata: %v", parseErr)
	}
	_, parseOk, parseErr2 := loadScaffoldMetadata(parseRoot)
	if parseOk || parseErr2 == nil || !strings.Contains(parseErr2.Error(), "parse scaffold metadata") {
		parseT.Fatalf("expected invalid metadata parse error, got ok=%t err=%v", parseOk, parseErr2)
	}
}

func TestLoadScaffoldMetadataUpgradesLegacySchemaDefaults(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseLegacy := `{
  "projectName": "legacy-app",
  "modulePath": "example.com/legacy-app"
}
`
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-start.json"), []byte(parseLegacy), 0644); parseErr != nil {
		parseT.Fatalf("write legacy metadata: %v", parseErr)
	}

	parseMetadata, parseOk, parseErr2 := loadScaffoldMetadata(parseRoot)
	if parseErr2 != nil || !parseOk {
		parseT.Fatalf("expected legacy metadata to load, got metadata=%#v ok=%t err=%v", parseMetadata, parseOk, parseErr2)
	}
	if parseMetadata.SchemaVersion != currentScaffoldMetadataSchemaVersion {
		parseT.Fatalf("expected legacy metadata to upgrade schema version, got %#v", parseMetadata)
	}
	if parseMetadata.Ownership.ProjectOwnership != "standalone" || parseMetadata.Ownership.FrameworkSourceMode != "module-proxy" {
		parseT.Fatalf("expected legacy metadata ownership defaults, got %#v", parseMetadata.Ownership)
	}
}

func TestLoadScaffoldMetadataRejectsUnknownSchemaVersion(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-start.json"), []byte(`{"schemaVersion":99}`), 0644); parseErr != nil {
		parseT.Fatalf("write future metadata: %v", parseErr)
	}
	_, parseOk, parseErr2 := loadScaffoldMetadata(parseRoot)
	if parseOk || parseErr2 == nil || !strings.Contains(parseErr2.Error(), "unsupported scaffold metadata schema version") {
		parseT.Fatalf("expected future metadata schema rejection, got ok=%t err=%v", parseOk, parseErr2)
	}
}

func TestLoadScaffoldMetadataMissingPathReturnsZeroValues(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseMetadata, parseOk, parseErr := loadScaffoldMetadata(parseRoot)
	if parseErr != nil || parseOk {
		parseT.Fatalf("expected missing metadata to return zero values, got metadata=%#v ok=%t err=%v", parseMetadata, parseOk, parseErr)
	}
	if parseMetadata.ProjectName != "" || parseMetadata.Tooling.AppPath != "" {
		parseT.Fatalf("expected missing metadata to leave zero-value fields, got %#v", parseMetadata)
	}
}

func TestNormalizePathBlankInputReturnsEmpty(parseT *testing.T) {
	parsePath, parseErr := normalizePath(parseT.TempDir(), "   ")
	if parseErr != nil {
		parseT.Fatalf("normalize blank path: %v", parseErr)
	}
	if parsePath != "" {
		parseT.Fatalf("expected blank normalized path, got %q", parsePath)
	}
}

func TestStartProjectStepAdditionalBranches(parseT *testing.T) {
	parseModel := newTestStartModel()
	parseModel.selection.Preset = parseModel.presets[0]
	parseModel.step = startStepProject
	parseModel.inputIndex = 1

	parseUpdatedModel, parseCmd := parseModel.updateProjectStep(tea.KeyMsg{Type: tea.KeyEsc})
	parseUpdated := parseUpdatedModel.(startModel)
	if parseUpdated.step != startStepPreset || parseCmd != nil {
		parseT.Fatalf("expected esc to return to preset step, got %#v cmd=%#v", parseUpdated, parseCmd)
	}

	parseModel = newTestStartModel()
	parseModel.selection.Preset = parseModel.presets[0]
	parseModel.step = startStepProject
	parseModel.inputIndex = 1
	parseUpdatedModel, _ = parseModel.updateProjectStep(tea.KeyMsg{Type: tea.KeyShiftTab})
	parseUpdated = parseUpdatedModel.(startModel)
	if parseUpdated.inputIndex != 0 {
		parseT.Fatalf("expected shift+tab to move focus backward, got %d", parseUpdated.inputIndex)
	}

	parseUpdatedModel, _ = parseUpdated.updateProjectStep(tea.KeyMsg{Type: tea.KeyTab})
	parseUpdated = parseUpdatedModel.(startModel)
	if parseUpdated.inputIndex != 1 {
		parseT.Fatalf("expected tab to move focus forward, got %d", parseUpdated.inputIndex)
	}

	parseUpdatedModel, _ = parseUpdated.updateProjectStep(tea.KeyMsg{Type: tea.KeyEnter})
	parseUpdated = parseUpdatedModel.(startModel)
	if parseUpdated.inputIndex != 2 || parseUpdated.step != startStepProject {
		parseT.Fatalf("expected enter before last input to advance focus, got %#v", parseUpdated)
	}

	parseModel = newTestStartModel()
	parseModel.selection.Preset = parseModel.presets[0]
	parseModel.step = startStepProject
	parseModel.inputIndex = len(parseModel.inputs) - 1
	parseUpdatedModel, _ = parseModel.updateProjectStep(tea.KeyMsg{Type: tea.KeyEnter})
	parseUpdated = parseUpdatedModel.(startModel)
	if parseUpdated.errText == "" || parseUpdated.step != startStepProject {
		parseT.Fatalf("expected invalid selection to keep project step with error, got %#v", parseUpdated)
	}

	parseModel = newTestStartModel()
	parseModel.selection.Preset = parseModel.presets[0]
	parseModel.step = startStepProject
	parseModel.inputIndex = len(parseModel.inputs) - 1
	parseModel.inputs[0].SetValue("starter-app")
	parseModel.inputs[1].SetValue("example.com/starter-app")
	parseModel.inputs[2].SetValue("Cam")
	parseModel.inputs[3].SetValue("1.0.0")
	parseModel.inputs[4].SetValue("Starter app")
	parseUpdatedModel, _ = parseModel.updateProjectStep(tea.KeyMsg{Type: tea.KeyEnter})
	parseUpdated = parseUpdatedModel.(startModel)
	if parseUpdated.step != startStepConfirm || parseUpdated.selection.ProjectName != "starter-app" {
		parseT.Fatalf("expected valid selection to advance to confirmation, got %#v", parseUpdated)
	}
}

func TestUpdateProjectStepCoversArrowKeysAndInputUpdate(parseT *testing.T) {
	parseModel := newTestStartModel()
	parseModel.selection.Preset = parseModel.presets[0]
	parseModel.step = startStepProject
	parseModel.focusInput(0)

	parseUpdatedModel, _ := parseModel.updateProjectStep(tea.KeyMsg{Type: tea.KeyUp})
	parseUpdated := parseUpdatedModel.(startModel)
	if parseUpdated.inputIndex != len(parseUpdated.inputs)-1 {
		parseT.Fatalf("expected up arrow to wrap focus to last input, got %d", parseUpdated.inputIndex)
	}

	parseUpdatedModel, _ = parseUpdated.updateProjectStep(tea.KeyMsg{Type: tea.KeyDown})
	parseUpdated = parseUpdatedModel.(startModel)
	if parseUpdated.inputIndex != 0 {
		parseT.Fatalf("expected down arrow to wrap focus to first input, got %d", parseUpdated.inputIndex)
	}

	parseUpdatedModel, _ = parseUpdated.updateProjectStep(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	parseUpdated = parseUpdatedModel.(startModel)
	if parseUpdated.inputs[0].Value() != "x" {
		parseT.Fatalf("expected rune input to update focused field, got %q", parseUpdated.inputs[0].Value())
	}
}

func TestUpdateConfirmStepBackspaceAndNoop(parseT *testing.T) {
	parseModel := newTestStartModel()
	parseModel.step = startStepConfirm
	parseModel.selection.Preset = parseModel.presets[0]
	parseModel.errText = "stale"

	parseUpdatedModel, _ := parseModel.updateConfirmStep(tea.KeyMsg{Type: tea.KeyBackspace})
	parseUpdated := parseUpdatedModel.(startModel)
	if parseUpdated.step != startStepProject || parseUpdated.errText != "" {
		parseT.Fatalf("expected backspace to return to project step and clear error, got step=%v err=%q", parseUpdated.step, parseUpdated.errText)
	}

	parseUpdatedModel, parseCmd := parseUpdated.updateConfirmStep(tea.MouseMsg{})
	parseUpdated = parseUpdatedModel.(startModel)
	if parseUpdated.step != startStepProject || parseCmd != nil {
		parseT.Fatalf("expected non-key confirm update to no-op, got step=%v cmd=%#v", parseUpdated.step, parseCmd)
	}
	if parseUpdated.confirmed {
		parseT.Fatal("expected non-key confirm update to leave confirmation false")
	}
}

func TestStartViewAndPathHelpersAdditionalBranches(parseT *testing.T) {
	if parseGot := (startModel{step: 999}).View(); parseGot != "\n" {
		parseT.Fatalf("expected unknown start view to return newline, got %q", parseGot)
	}

	parseFile, parseErr := os.CreateTemp(parseT.TempDir(), "closed-*.txt")
	if parseErr != nil {
		parseT.Fatalf("create temp file: %v", parseErr)
	}
	parsePath := parseFile.Name()
	if parseErr2 := parseFile.Close(); parseErr2 != nil {
		parseT.Fatalf("close temp file: %v", parseErr2)
	}
	parseOpened, parseErr := os.Open(parsePath)
	if parseErr != nil {
		parseT.Fatalf("open temp file: %v", parseErr)
	}
	if parseErr3 := parseOpened.Close(); parseErr3 != nil {
		parseT.Fatalf("close opened file: %v", parseErr3)
	}
	if isInteractiveFile(parseOpened) {
		parseT.Fatal("expected closed file stat failure to be treated as non-interactive")
	}

	parseRoot := parseT.TempDir()
	parseRelativePath, parseErr := normalizePath(parseRoot, filepath.Join("nested", "file.txt"))
	if parseErr != nil {
		parseT.Fatalf("normalize relative path: %v", parseErr)
	}
	if parseRelativePath != filepath.Join(parseRoot, "nested", "file.txt") {
		parseT.Fatalf("expected normalized relative path, got %q", parseRelativePath)
	}
	parseAbsolutePath, parseErr := normalizePath(parseRoot, filepath.Join(parseRoot, "already.txt"))
	if parseErr != nil {
		parseT.Fatalf("normalize absolute path: %v", parseErr)
	}
	if parseAbsolutePath != filepath.Join(parseRoot, "already.txt") {
		parseT.Fatalf("expected normalized absolute path, got %q", parseAbsolutePath)
	}
	if _, parseErr4 := normalizeExistingPath(parseRoot, filepath.Join(parseRoot, "missing.txt")); parseErr4 == nil {
		parseT.Fatal("expected missing existing path to fail")
	}
	if parseErr5 := os.WriteFile(filepath.Join(parseRoot, "existing.txt"), []byte("ok"), 0644); parseErr5 != nil {
		parseT.Fatalf("write existing path: %v", parseErr5)
	}
	parseResolvedExisting, parseErr := normalizeExistingPath(parseRoot, filepath.Join(parseRoot, "existing.txt"))
	if parseErr != nil {
		parseT.Fatalf("normalize existing path: %v", parseErr)
	}
	if parseResolvedExisting != filepath.Join(parseRoot, "existing.txt") {
		parseT.Fatalf("expected resolved existing path, got %q", parseResolvedExisting)
	}
}

func TestRunStartTUIUsesProgramRunnerResult(parseT *testing.T) {
	parseOriginalProgramRunner := startProgramRunner
	parseOriginalSections := startEnterpriseScaffoldSections
	parseT.Cleanup(func() {
		startProgramRunner = parseOriginalProgramRunner
		startEnterpriseScaffoldSections = parseOriginalSections
	})
	startEnterpriseScaffoldSections = []launcherPluginScaffoldSection{
		{
			Title:    "Org Security Baseline",
			Summary:  "Organization-required secure defaults.",
			Features: []string{"release-profile", "browser-tests"},
		},
	}

	startProgramRunner = func(parseModel tea.Model) (tea.Model, error) {
		parseStartModelValue, parseOk := parseModel.(startModel)
		if !parseOk {
			parseT.Fatalf("expected start model, got %T", parseModel)
		}
		if len(parseStartModelValue.enterpriseSections) != 1 {
			parseT.Fatalf("expected enterprise sections to be present, got %#v", parseStartModelValue.enterpriseSections)
		}
		parseStartModelValue.confirmed = true
		parseStartModelValue.selection.Preset = parseStartModelValue.presets[1]
		parseStartModelValue.selection.EnterpriseSections = append([]launcherPluginScaffoldSection(nil), parseStartModelValue.enterpriseSections...)
		parseStartModelValue.selection.EnabledEnterpriseSections = []string{parseStartModelValue.enterpriseSections[0].Title}
		parseStartModelValue.selection.EnterpriseFeatures = append([]string(nil), parseStartModelValue.enterpriseSections[0].Features...)
		parseStartModelValue.inputs[0].SetValue("starter-app")
		parseStartModelValue.inputs[1].SetValue("example.com/starter-app")
		parseStartModelValue.inputs[2].SetValue("Cam")
		parseStartModelValue.inputs[3].SetValue("1.2.3")
		parseStartModelValue.inputs[4].SetValue("Starter description")
		return parseStartModelValue, nil
	}

	parseSelection, parseErr := runStartTUI()
	if parseErr != nil {
		parseT.Fatalf("run start tui: %v", parseErr)
	}
	if parseSelection == nil {
		parseT.Fatal("expected confirmed selection")
	}
	if parseSelection.Preset.Key != "routed-spa" || parseSelection.ProjectName != "starter-app" {
		parseT.Fatalf("expected selection from final start model, got %#v", parseSelection)
	}
	if parseSelection.ModulePath != "example.com/starter-app" || parseSelection.Author != "Cam" || parseSelection.Version != "1.2.3" {
		parseT.Fatalf("expected populated selection values, got %#v", parseSelection)
	}
	if len(parseSelection.EnabledEnterpriseSections) != 1 || parseSelection.EnabledEnterpriseSections[0] != "Org Security Baseline" {
		parseT.Fatalf("expected selected enterprise section to round-trip, got %#v", parseSelection)
	}
	if parseGot := strings.Join(parseSelection.EnterpriseFeatures, ","); parseGot != "release-profile,browser-tests" {
		parseT.Fatalf("expected enterprise features to round-trip, got %q", parseGot)
	}
	if parseSelection.ProjectMode != scaffoldProjectModeStandalone {
		parseT.Fatalf("expected default project mode to be standalone, got %#v", parseSelection)
	}
	if filepath.Base(parseSelection.TargetDir) != "starter-app" {
		parseT.Fatalf("expected target dir derived from project name, got %#v", parseSelection)
	}
}

func TestRunStartTUIHandlesCancelAndProgramError(parseT *testing.T) {
	parseOriginalProgramRunner := startProgramRunner
	parseT.Cleanup(func() { startProgramRunner = parseOriginalProgramRunner })

	startProgramRunner = func(parseModel tea.Model) (tea.Model, error) {
		return startModel{}, nil
	}
	parseSelection, parseErr := runStartTUI()
	if parseErr != nil {
		parseT.Fatalf("run start tui cancel path: %v", parseErr)
	}
	if parseSelection != nil {
		parseT.Fatalf("expected nil selection on unconfirmed model, got %#v", parseSelection)
	}

	startProgramRunner = func(parseModel2 tea.Model) (tea.Model, error) {
		return nil, errors.New("bubbletea failed")
	}
	parseSelection, parseErr = runStartTUI()
	if parseErr == nil || !strings.Contains(parseErr.Error(), "bubbletea failed") {
		parseT.Fatalf("expected bubbled program error, got selection=%#v err=%v", parseSelection, parseErr)
	}
}

func TestRunStartPostTUIUsesProgramRunnerResult(parseT *testing.T) {
	parseOriginalProgramRunner := startProgramRunner
	parseT.Cleanup(func() { startProgramRunner = parseOriginalProgramRunner })

	startProgramRunner = func(parseModel tea.Model) (tea.Model, error) {
		parsePostModel, parseOk := parseModel.(startPostModel)
		if !parseOk {
			parseT.Fatalf("expected start post model, got %T", parseModel)
		}
		parsePostModel.confirmed = true
		parsePostModel.cursor = 1
		return parsePostModel, nil
	}

	parseResult, parseErr := runStartPostTUI(startSelection{ProjectName: "starter-app"}, &scaffoldResult{TargetDir: `C:\tmp\starter-app`}, nil)
	if parseErr != nil {
		parseT.Fatalf("run start post tui: %v", parseErr)
	}
	if parseResult == nil || !parseResult.RunDev {
		parseT.Fatalf("expected confirmed run-dev post result, got %#v", parseResult)
	}
}

func TestRunStartPostTUIHandlesCancelErrorAndGenerationFailure(parseT *testing.T) {
	parseOriginalProgramRunner := startProgramRunner
	parseT.Cleanup(func() { startProgramRunner = parseOriginalProgramRunner })

	startProgramRunner = func(parseModel tea.Model) (tea.Model, error) {
		return startPostModel{}, nil
	}
	parseResult, parseErr := runStartPostTUI(startSelection{ProjectName: "starter-app"}, &scaffoldResult{}, nil)
	if parseErr != nil {
		parseT.Fatalf("run start post tui cancel path: %v", parseErr)
	}
	if parseResult != nil {
		parseT.Fatalf("expected nil post result on unconfirmed model, got %#v", parseResult)
	}

	startProgramRunner = func(parseModel2 tea.Model) (tea.Model, error) {
		return nil, errors.New("post bubbletea failed")
	}
	parseResult, parseErr = runStartPostTUI(startSelection{ProjectName: "starter-app"}, &scaffoldResult{}, nil)
	if parseErr == nil || !strings.Contains(parseErr.Error(), "post bubbletea failed") {
		parseT.Fatalf("expected bubbled post-program error, got result=%#v err=%v", parseResult, parseErr)
	}

	startProgramRunner = func(parseModel3 tea.Model) (tea.Model, error) {
		parsePostModel, parseOk := parseModel3.(startPostModel)
		if !parseOk {
			parseT.Fatalf("expected start post model, got %T", parseModel3)
		}
		parsePostModel.confirmed = true
		parsePostModel.cursor = 1
		return parsePostModel, nil
	}
	parseResult, parseErr = runStartPostTUI(startSelection{ProjectName: "starter-app"}, nil, errors.New("generation failed"))
	if parseErr != nil {
		parseT.Fatalf("run start post tui generation failure path: %v", parseErr)
	}
	if parseResult != nil {
		parseT.Fatalf("expected generation failure path to suppress post result, got %#v", parseResult)
	}
}

func TestRunStartUsesInjectedCollaborators(parseT *testing.T) {
	parseOriginalTerminalValidator := startTerminalValidator
	parseOriginalSelectionRunner := startSelectionRunner
	parseOriginalPostRunner := startPostRunner
	parseOriginalGenerateScaffold := startGenerateScaffold
	parseOriginalInitGit := startInitGit
	parseOriginalRunDev := startRunDev
	parseOriginalResolveSections := startResolveScaffoldSections
	parseT.Cleanup(func() {
		startTerminalValidator = parseOriginalTerminalValidator
		startSelectionRunner = parseOriginalSelectionRunner
		startPostRunner = parseOriginalPostRunner
		startGenerateScaffold = parseOriginalGenerateScaffold
		startInitGit = parseOriginalInitGit
		startRunDev = parseOriginalRunDev
		startResolveScaffoldSections = parseOriginalResolveSections
	})
	startResolveScaffoldSections = func(parseRepoRoot string) ([]launcherPluginScaffoldSection, error) { return nil, nil }

	parseSelection := startSelection{ProjectName: "starter-app", Preset: startPreset{Key: "minimal-client"}}
	parseResult := scaffoldResult{TargetDir: `C:\tmp\starter-app`, AppPath: `C:\tmp\starter-app\main.go`, HTMLPath: `C:\tmp\starter-app\index.html`}
	parseAppLauncher := launcher{}
	parseStartArgs := []string{"-skip-prereq-checks"}

	parseT.Run("terminal validation failure", func(parseT2 *testing.T) {
		startTerminalValidator = func(bool, bool) error { return errors.New("not interactive") }
		parseErr := parseAppLauncher.runStart(parseStartArgs)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "not interactive") {
			parseT2.Fatalf("expected terminal validation error, got %v", parseErr)
		}
	})

	parseT.Run("selection canceled", func(parseT3 *testing.T) {
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return nil, nil }
		parseErr2 := parseAppLauncher.runStart(parseStartArgs)
		if parseErr2 != nil {
			parseT3.Fatalf("expected nil error on canceled selection, got %v", parseErr2)
		}
	})

	parseT.Run("selection runner error", func(parseT4 *testing.T) {
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return nil, errors.New("selection failed") }
		parseErr3 := parseAppLauncher.runStart(parseStartArgs)
		if parseErr3 == nil || !strings.Contains(parseErr3.Error(), "selection failed") {
			parseT4.Fatalf("expected selection runner error, got %v", parseErr3)
		}
	})

	parseT.Run("generation failure still shows post tui", func(parseT5 *testing.T) {
		isParsePostCalled := false
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &parseSelection, nil }
		startGenerateScaffold = func(parseL launcher, parseGot startSelection) (scaffoldResult, error) {
			if parseGot.ProjectName != parseSelection.ProjectName {
				parseT5.Fatalf("expected selection to flow into generation, got %#v", parseGot)
			}
			return scaffoldResult{}, errors.New("generation failed")
		}
		startPostRunner = func(parseGot2 startSelection, parseGenerated *scaffoldResult, parseGenerationErr error) (*startPostResult, error) {
			isParsePostCalled = true
			if parseGenerated != nil || parseGenerationErr == nil || parseGenerationErr.Error() != "generation failed" {
				parseT5.Fatalf("expected generation error to flow into post runner, got generated=%#v err=%v", parseGenerated, parseGenerationErr)
			}
			return nil, nil
		}
		parseErr4 := parseAppLauncher.runStart(parseStartArgs)
		if parseErr4 != nil {
			parseT5.Fatalf("expected generation failure path to return nil after post tui, got %v", parseErr4)
		}
		if !isParsePostCalled {
			parseT5.Fatal("expected post tui to run after generation failure")
		}
	})

	parseT.Run("post tui error bubbles", func(parseT6 *testing.T) {
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &parseSelection, nil }
		startGenerateScaffold = func(parseL2 launcher, parseGot3 startSelection) (scaffoldResult, error) { return parseResult, nil }
		startPostRunner = func(parseGot4 startSelection, parseGenerated2 *scaffoldResult, parseGenerationErr2 error) (*startPostResult, error) {
			return nil, errors.New("post failed")
		}
		parseErr5 := parseAppLauncher.runStart(parseStartArgs)
		if parseErr5 == nil || !strings.Contains(parseErr5.Error(), "post failed") {
			parseT6.Fatalf("expected post runner error, got %v", parseErr5)
		}
	})

	parseT.Run("run dev path uses scaffold args", func(parseT7 *testing.T) {
		isParseRunDevCalled := false
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &parseSelection, nil }
		startGenerateScaffold = func(parseL3 launcher, parseGot5 startSelection) (scaffoldResult, error) { return parseResult, nil }
		startPostRunner = func(parseGot6 startSelection, parseGenerated3 *scaffoldResult, parseGenerationErr3 error) (*startPostResult, error) {
			return &startPostResult{RunDev: true}, nil
		}
		startRunDev = func(parseL4 launcher, parseArgs []string) error {
			isParseRunDevCalled = true
			if fmt.Sprint(parseArgs) != fmt.Sprint(devArgsFromScaffold(parseResult)) {
				parseT7.Fatalf("expected scaffold dev args, got %#v", parseArgs)
			}
			return nil
		}
		parseErr6 := parseAppLauncher.runStart(parseStartArgs)
		if parseErr6 != nil {
			parseT7.Fatalf("expected run dev path to succeed, got %v", parseErr6)
		}
		if !isParseRunDevCalled {
			parseT7.Fatal("expected runDev to be called")
		}
	})

	parseT.Run("optional setup flags flow into scaffold selection", func(parseT8 *testing.T) {
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &parseSelection, nil }
		startGenerateScaffold = func(parseL5 launcher, parseGot7 startSelection) (scaffoldResult, error) {
			if !parseGot7.SkipGoModTidy || !parseGot7.SkipRuntimeAssets {
				parseT8.Fatalf("expected skip flags on scaffold selection, got %+v", parseGot7)
			}
			return parseResult, nil
		}
		startPostRunner = func(parseGot8 startSelection, parseGenerated4 *scaffoldResult, parseGenerationErr4 error) (*startPostResult, error) {
			return nil, nil
		}
		parseErr7 := parseAppLauncher.runStart([]string{"-skip-prereq-checks", "-skip-tidy", "-skip-runtime-assets"})
		if parseErr7 != nil {
			parseT8.Fatalf("expected optional setup flag path to succeed, got %v", parseErr7)
		}
	})

	parseT.Run("mode flag flows into scaffold selection", func(parseT9 *testing.T) {
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &parseSelection, nil }
		startGenerateScaffold = func(parseL6 launcher, parseGot9 startSelection) (scaffoldResult, error) {
			if parseGot9.ProjectMode != scaffoldProjectModeContributorLinked {
				parseT9.Fatalf("expected contributor-linked mode on scaffold selection, got %+v", parseGot9)
			}
			return parseResult, nil
		}
		startPostRunner = func(parseGot10 startSelection, parseGenerated5 *scaffoldResult, parseGenerationErr5 error) (*startPostResult, error) {
			return nil, nil
		}
		if parseErr8 := parseAppLauncher.runStart([]string{"-skip-prereq-checks", "-mode", "contributor-linked"}); parseErr8 != nil {
			parseT9.Fatalf("expected mode flag path to succeed, got %v", parseErr8)
		}
	})

	parseT.Run("init-git flag initializes repository after generation", func(parseT10 *testing.T) {
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &parseSelection, nil }
		startGenerateScaffold = func(parseL7 launcher, parseGot11 startSelection) (scaffoldResult, error) {
			if !parseGot11.InitGit {
				parseT10.Fatalf("expected init-git flag on scaffold selection, got %+v", parseGot11)
			}
			return parseResult, nil
		}
		isParseGitInitCalled := false
		startInitGit = func(parseTargetDir string) error {
			isParseGitInitCalled = true
			if parseTargetDir != parseResult.TargetDir {
				parseT10.Fatalf("expected git init target %q, got %q", parseResult.TargetDir, parseTargetDir)
			}
			return nil
		}
		startPostRunner = func(parseGot12 startSelection, parseGenerated6 *scaffoldResult, parseGenerationErr6 error) (*startPostResult, error) {
			return nil, nil
		}
		if parseErr9 := parseAppLauncher.runStart([]string{"-skip-prereq-checks", "-init-git"}); parseErr9 != nil {
			parseT10.Fatalf("expected init-git path to succeed, got %v", parseErr9)
		}
		if !isParseGitInitCalled {
			parseT10.Fatal("expected git init to be called")
		}
	})

	parseT.Run("git init failure still shows post tui", func(parseT11 *testing.T) {
		isParsePostCalled2 := false
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &parseSelection, nil }
		startGenerateScaffold = func(parseL8 launcher, parseGot13 startSelection) (scaffoldResult, error) { return parseResult, nil }
		startInitGit = func(parseTargetDir2 string) error { return errors.New("git init failed") }
		startPostRunner = func(parseGot14 startSelection, parseGenerated7 *scaffoldResult, parseGenerationErr7 error) (*startPostResult, error) {
			isParsePostCalled2 = true
			if parseGenerated7 == nil || parseGenerationErr7 == nil || parseGenerationErr7.Error() != "git init failed" {
				parseT11.Fatalf("expected git init error to flow into post runner, got generated=%#v err=%v", parseGenerated7, parseGenerationErr7)
			}
			return nil, nil
		}
		if parseErr10 := parseAppLauncher.runStart([]string{"-skip-prereq-checks", "-init-git"}); parseErr10 != nil {
			parseT11.Fatalf("expected git init failure path to return nil after post tui, got %v", parseErr10)
		}
		if !isParsePostCalled2 {
			parseT11.Fatal("expected post tui after git init failure")
		}
	})

	parseT.Run("invalid flag parse error bubbles", func(parseT12 *testing.T) {
		startTerminalValidator = func(bool, bool) error {
			parseT12.Fatal("expected parse failure before terminal validation")
			return nil
		}
		parseErr11 := parseAppLauncher.runStart([]string{"-definitely-invalid"})
		if parseErr11 == nil || !strings.Contains(parseErr11.Error(), "flag provided but not defined") {
			parseT12.Fatalf("expected invalid flag parse error, got %v", parseErr11)
		}
	})

	parseT.Run("invalid mode parse error bubbles", func(parseT13 *testing.T) {
		startTerminalValidator = func(bool, bool) error {
			parseT13.Fatal("expected mode validation before terminal validation")
			return nil
		}
		parseErr12 := parseAppLauncher.runStart([]string{"-mode", "not-a-real-mode"})
		if parseErr12 == nil || !strings.Contains(parseErr12.Error(), "unknown scaffold mode") {
			parseT13.Fatalf("expected invalid mode error, got %v", parseErr12)
		}
	})

	parseT.Run("post tui nil result skips run dev", func(parseT14 *testing.T) {
		isParseRunDevCalled2 := false
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &parseSelection, nil }
		startGenerateScaffold = func(parseL9 launcher, parseGot15 startSelection) (scaffoldResult, error) { return parseResult, nil }
		startPostRunner = func(parseGot16 startSelection, parseGenerated8 *scaffoldResult, parseGenerationErr8 error) (*startPostResult, error) {
			return nil, nil
		}
		startRunDev = func(parseL10 launcher, parseArgs2 []string) error {
			isParseRunDevCalled2 = true
			return nil
		}
		parseErr13 := parseAppLauncher.runStart(parseStartArgs)
		if parseErr13 != nil {
			parseT14.Fatalf("expected nil post result path to succeed, got %v", parseErr13)
		}
		if isParseRunDevCalled2 {
			parseT14.Fatal("expected runDev to be skipped when post result is nil")
		}
	})

	parseT.Run("post tui exit choice skips run dev", func(parseT15 *testing.T) {
		isParseRunDevCalled3 := false
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &parseSelection, nil }
		startGenerateScaffold = func(parseL11 launcher, parseGot17 startSelection) (scaffoldResult, error) { return parseResult, nil }
		startPostRunner = func(parseGot18 startSelection, parseGenerated9 *scaffoldResult, parseGenerationErr9 error) (*startPostResult, error) {
			return &startPostResult{RunDev: false}, nil
		}
		startRunDev = func(parseL12 launcher, parseArgs3 []string) error {
			isParseRunDevCalled3 = true
			return nil
		}
		parseErr14 := parseAppLauncher.runStart(parseStartArgs)
		if parseErr14 != nil {
			parseT15.Fatalf("expected exit choice path to succeed, got %v", parseErr14)
		}
		if isParseRunDevCalled3 {
			parseT15.Fatal("expected runDev to be skipped when RunDev is false")
		}
	})

	parseT.Run("run dev error bubbles", func(parseT16 *testing.T) {
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &parseSelection, nil }
		startGenerateScaffold = func(parseL13 launcher, parseGot19 startSelection) (scaffoldResult, error) { return parseResult, nil }
		startPostRunner = func(parseGot20 startSelection, parseGenerated10 *scaffoldResult, parseGenerationErr10 error) (*startPostResult, error) {
			return &startPostResult{RunDev: true}, nil
		}
		startRunDev = func(parseL14 launcher, parseArgs4 []string) error {
			return errors.New("dev launch failed")
		}
		parseErr15 := parseAppLauncher.runStart(parseStartArgs)
		if parseErr15 == nil || !strings.Contains(parseErr15.Error(), "dev launch failed") {
			parseT16.Fatalf("expected runDev error to bubble, got %v", parseErr15)
		}
	})
}

func TestResolveStartScaffoldPluginSectionsCollectsContributions(parseT *testing.T) {
	parsePluginPath := filepath.Join(parseT.TempDir(), "scaffold-plugin.exe")
	if parseErr := os.WriteFile(parsePluginPath, []byte(""), 0644); parseErr != nil {
		parseT.Fatalf("write plugin placeholder: %v", parseErr)
	}

	parseOriginalConfig := launcherActiveEnterpriseConfig
	parseOriginalSources := launcherActiveEnterpriseSources
	parseOriginalPluginProcess := launcherRunPluginProcess
	parseT.Cleanup(func() {
		launcherActiveEnterpriseConfig = parseOriginalConfig
		launcherActiveEnterpriseSources = parseOriginalSources
		launcherRunPluginProcess = parseOriginalPluginProcess
	})
	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Plugins: []launcherExecutablePlugin{
			{
				Name:         "scaffold-plugin",
				Path:         parsePluginPath,
				Capabilities: []string{"scaffold_feature"},
			},
		},
	}
	launcherActiveEnterpriseSources = launcherEnterpriseConfigSources{FrameworkDefaults: true}
	launcherRunPluginProcess = func(parsePath string, parseArgs []string, parseEnv []string, parseStdin []byte, parseTimeout time.Duration) (string, string, error) {
		return `{"scaffoldSections":[{"title":"Org defaults","summary":"Enterprise baseline","features":["release-profile","release-profile","browser-tests"]}]}`, "", nil
	}

	parseSections, parseErr2 := resolveStartScaffoldPluginSections(`C:\repo`)
	if parseErr2 != nil {
		parseT.Fatalf("resolve scaffold plugin sections: %v", parseErr2)
	}
	if len(parseSections) != 1 || parseSections[0].Title != "Org defaults" {
		parseT.Fatalf("expected scaffold plugin section to be collected, got %#v", parseSections)
	}
	if parseGot := strings.Join(parseSections[0].Features, ","); parseGot != "release-profile,browser-tests" {
		parseT.Fatalf("expected normalized scaffold section features, got %q", parseGot)
	}
}

func TestBuildSelectionRequiresFields(parseT *testing.T) {
	parseTests := []struct {
		name        string
		projectName string
		modulePath  string
		author      string
		version     string
		description string
		wantErr     string
	}{
		{name: "missing project name", modulePath: "github.com/test/app", author: "Cam", version: "0.1.0", description: "desc", wantErr: "project name is required"},
		{name: "missing module path", projectName: "app", author: "Cam", version: "0.1.0", description: "desc", wantErr: "module path is required"},
		{name: "missing author", projectName: "app", modulePath: "github.com/test/app", version: "0.1.0", description: "desc", wantErr: "author is required"},
		{name: "missing version", projectName: "app", modulePath: "github.com/test/app", author: "Cam", description: "desc", wantErr: "version is required"},
		{name: "missing description", projectName: "app", modulePath: "github.com/test/app", author: "Cam", version: "0.1.0", wantErr: "description is required"},
		{name: "invalid folder name", projectName: `bad:name`, modulePath: "github.com/test/app", author: "Cam", version: "0.1.0", description: "desc", wantErr: "project name contains characters"},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			parseModel := newTestStartModel()
			parseModel.inputs[0].SetValue(parseTest.projectName)
			parseModel.inputs[1].SetValue(parseTest.modulePath)
			parseModel.inputs[2].SetValue(parseTest.author)
			parseModel.inputs[3].SetValue(parseTest.version)
			parseModel.inputs[4].SetValue(parseTest.description)

			_, parseErr := parseModel.buildSelection()
			if parseErr == nil || !strings.Contains(parseErr.Error(), parseTest.wantErr) {
				parseT2.Fatalf("expected error containing %q, got %v", parseTest.wantErr, parseErr)
			}
		})
	}
}

func TestBuildSelectionDerivesTargetDirFromProjectName(parseT *testing.T) {
	parseModel := newTestStartModel()
	parseModel.inputs[0].SetValue("derived-app")
	parseModel.inputs[1].SetValue("github.com/test/derived-app")
	parseModel.inputs[2].SetValue("Cam")
	parseModel.inputs[3].SetValue("0.1.0")
	parseModel.inputs[4].SetValue("desc")

	parseSelection, parseErr := parseModel.buildSelection()
	if parseErr != nil {
		parseT.Fatalf("build selection: %v", parseErr)
	}
	if filepath.Base(parseSelection.TargetDir) != "derived-app" {
		parseT.Fatalf("expected target dir base to match project name, got %q", parseSelection.TargetDir)
	}
}

func TestValidateGeneratedTargetDirAllowsUserWorkspacePath(parseT *testing.T) {
	parseTargetDir := filepath.Join(parseT.TempDir(), "starter-app")
	if parseErr := validateGeneratedTargetDir(parseTargetDir); parseErr != nil {
		parseT.Fatalf("expected user workspace path %q to be allowed, got %v", parseTargetDir, parseErr)
	}
}

func TestValidateGeneratedTargetDirRejectsFilesystemRoot(parseT *testing.T) {
	parseRootPath := string(os.PathSeparator)
	if parseVolume := filepath.VolumeName(parseRootPath); parseVolume != "" {
		parseRootPath = parseVolume + string(os.PathSeparator)
	}
	if parseErr := validateGeneratedTargetDir(parseRootPath); parseErr == nil {
		parseT.Fatal("expected filesystem root to be rejected")
	}
}

func TestEnsureEmptyDirRejectsNonEmptyDirectory(parseT *testing.T) {
	parseTargetDir := filepath.Join(defaultGeneratedScaffoldRoot(), "test-non-empty-dir")
	_ = os.RemoveAll(parseTargetDir)
	if parseErr := os.MkdirAll(parseTargetDir, 0755); parseErr != nil {
		parseT.Fatalf("create target dir: %v", parseErr)
	}
	parseT.Cleanup(func() {
		_ = os.RemoveAll(parseTargetDir)
	})
	if parseErr2 := os.WriteFile(filepath.Join(parseTargetDir, "existing.txt"), []byte("occupied"), 0644); parseErr2 != nil {
		parseT.Fatalf("seed target dir: %v", parseErr2)
	}

	if parseErr3 := ensureEmptyDir(parseTargetDir); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "not empty") {
		parseT.Fatalf("expected non-empty directory error, got %v", parseErr3)
	}
}

func TestEnsureEmptyDirRejectsExistingFile(parseT *testing.T) {
	parseTargetPath := filepath.Join(defaultGeneratedScaffoldRoot(), "test-existing-file")
	_ = os.Remove(parseTargetPath)
	if parseErr := os.MkdirAll(filepath.Dir(parseTargetPath), 0755); parseErr != nil {
		parseT.Fatalf("create parent dir: %v", parseErr)
	}
	parseT.Cleanup(func() {
		_ = os.Remove(parseTargetPath)
	})
	if parseErr2 := os.WriteFile(parseTargetPath, []byte("file"), 0644); parseErr2 != nil {
		parseT.Fatalf("seed target path: %v", parseErr2)
	}

	if parseErr3 := ensureEmptyDir(parseTargetPath); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "not a directory") {
		parseT.Fatalf("expected existing file error, got %v", parseErr3)
	}
}

func TestEnsureEmptyDirAllowsCreateAndEmptyDirectory(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseNewDir := filepath.Join(parseRoot, "new-target")
	if parseErr := ensureEmptyDir(parseNewDir); parseErr != nil {
		parseT.Fatalf("expected missing directory to be created, got %v", parseErr)
	}
	if parseInfo, parseErr2 := os.Stat(parseNewDir); parseErr2 != nil || !parseInfo.IsDir() {
		parseT.Fatalf("expected created directory to exist, info=%#v err=%v", parseInfo, parseErr2)
	}
	if parseErr3 := ensureEmptyDir(parseNewDir); parseErr3 != nil {
		parseT.Fatalf("expected existing empty directory to remain valid, got %v", parseErr3)
	}
}

func TestDetectHTMLPathPrefersIndexHTML(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseIndexPath := filepath.Join(parseRoot, "index.html")
	parseOtherPath := filepath.Join(parseRoot, "other.html")
	if parseErr := os.WriteFile(parseOtherPath, []byte("other"), 0644); parseErr != nil {
		parseT.Fatalf("write other html: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseIndexPath, []byte("index"), 0644); parseErr2 != nil {
		parseT.Fatalf("write index html: %v", parseErr2)
	}

	if parseGot := detectHTMLPath(parseRoot); parseGot != parseIndexPath {
		parseT.Fatalf("expected index.html to be preferred, got %q", parseGot)
	}
}

func TestDetectHTMLPathFallsBackToFirstHTMLFile(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseFallbackPath := filepath.Join(parseRoot, "app-shell.html")
	if parseErr := os.WriteFile(parseFallbackPath, []byte("shell"), 0644); parseErr != nil {
		parseT.Fatalf("write fallback html: %v", parseErr)
	}

	if parseGot := detectHTMLPath(parseRoot); parseGot != parseFallbackPath {
		parseT.Fatalf("expected fallback html path %q, got %q", parseFallbackPath, parseGot)
	}
}

func TestDetectHTMLPathReturnsEmptyForNonDirectoryRoot(parseT *testing.T) {
	parseRootFile := filepath.Join(parseT.TempDir(), "not-a-dir")
	if parseErr := os.WriteFile(parseRootFile, []byte("occupied"), 0644); parseErr != nil {
		parseT.Fatalf("write root file: %v", parseErr)
	}

	if parseGot := detectHTMLPath(parseRootFile); parseGot != "" {
		parseT.Fatalf("expected empty html path for non-directory root, got %q", parseGot)
	}
}

func TestRenderScaffoldGoModUsesStandaloneLayout(parseT *testing.T) {
	parseSelection := startSelection{ModulePath: "github.com/test/app"}
	parseContent := renderScaffoldGoMod(parseSelection, "github.com/monstercameron/GoWebComponents", `C:\repo\GoWebComponents`)
	for _, parseExpected := range []string{"module github.com/test/app", "go 1.25.0"} {
		if !strings.Contains(parseContent, parseExpected) {
			parseT.Fatalf("expected go.mod content to contain %q", parseExpected)
		}
	}
	for _, parseUnexpected := range []string{"require github.com/monstercameron/GoWebComponents", "replace github.com/monstercameron/GoWebComponents"} {
		if strings.Contains(parseContent, parseUnexpected) {
			parseT.Fatalf("expected standalone go.mod content to omit %q", parseUnexpected)
		}
	}
}

func TestRenderScaffoldGoModUsesContributorLinkedReplace(parseT *testing.T) {
	parseSelection := startSelection{
		ModulePath:  "github.com/test/app",
		ProjectMode: scaffoldProjectModeContributorLinked,
	}
	parseContent := renderScaffoldGoMod(parseSelection, "github.com/monstercameron/GoWebComponents", `C:\repo\GoWebComponents`)
	for _, parseExpected := range []string{
		"module github.com/test/app",
		"go 1.25.0",
		"replace github.com/monstercameron/GoWebComponents => C:/repo/GoWebComponents",
	} {
		if !strings.Contains(parseContent, parseExpected) {
			parseT.Fatalf("expected contributor-linked go.mod content to contain %q", parseExpected)
		}
	}
}

func TestRenderScaffoldMetadataIncludesToolingDefaults(parseT *testing.T) {
	parseSelection := startSelection{
		Preset: startPreset{
			Key:         "minimal-client",
			Name:        "Minimal Client",
			Summary:     "Starter summary",
			Description: "Starter description",
			Features:    []string{"ui", "html"},
		},
		ProjectName: "starter-app",
		ModulePath:  "example.com/starter-app",
		Author:      "Cam",
		Version:     "0.1.0",
		Description: "starter",
		TargetDir:   filepath.Join("C:\\tmp", "starter-app"),
	}

	parseContent := renderScaffoldMetadata(parseSelection)
	var parseMetadata scaffoldMetadata
	if parseErr := json.Unmarshal([]byte(parseContent), &parseMetadata); parseErr != nil {
		parseT.Fatalf("unmarshal scaffold metadata: %v\n%s", parseErr, parseContent)
	}
	if parseMetadata.ProjectName != parseSelection.ProjectName || parseMetadata.ModulePath != parseSelection.ModulePath {
		parseT.Fatalf("expected project metadata to round-trip, got %#v", parseMetadata)
	}
	if parseMetadata.SchemaVersion != currentScaffoldMetadataSchemaVersion {
		parseT.Fatalf("expected scaffold metadata schema version %d, got %#v", currentScaffoldMetadataSchemaVersion, parseMetadata)
	}
	if parseMetadata.Tooling.AppPath != "main.go" || parseMetadata.Tooling.HTMLPath != "index.html" || parseMetadata.Tooling.WASMPath != filepath.ToSlash(filepath.Join("bin", "main.wasm")) {
		parseT.Fatalf("expected default tooling paths, got %#v", parseMetadata.Tooling)
	}
	if parseMetadata.Tooling.ReleaseOutDir != filepath.ToSlash(filepath.Join("bin", "wasm-release")) || parseMetadata.Tooling.ReleaseBinaryName != "app.wasm" || parseMetadata.Tooling.ReleaseCompression != "gzip+brotli" {
		parseT.Fatalf("expected release tooling defaults, got %#v", parseMetadata.Tooling)
	}
	if parseMetadata.Ownership.ProjectOwnership != "standalone" || parseMetadata.Ownership.FrameworkSourceMode != "module-proxy" {
		parseT.Fatalf("expected standalone ownership metadata, got %#v", parseMetadata.Ownership)
	}
	if parseMetadata.Preset.Key != parseSelection.Preset.Key || len(parseMetadata.Preset.Features) != 2 {
		parseT.Fatalf("expected preset metadata to round-trip, got %#v", parseMetadata.Preset)
	}
	if !strings.HasSuffix(parseContent, "\n") {
		parseT.Fatalf("expected scaffold metadata to end with newline, got %q", parseContent)
	}
}

func TestRenderScaffoldMetadataRecordsContributorLinkedOwnership(parseT *testing.T) {
	parseSelection := startSelection{
		Preset: startPreset{
			Key:         "minimal-client",
			Name:        "Minimal Client",
			Summary:     "Starter summary",
			Description: "Starter description",
			Features:    []string{"ui", "html"},
		},
		ProjectMode: scaffoldProjectModeContributorLinked,
		ProjectName: "starter-app",
		ModulePath:  "example.com/starter-app",
		Author:      "Cam",
		Version:     "0.1.0",
		Description: "starter",
		TargetDir:   filepath.Join("C:\\tmp", "starter-app"),
	}

	parseContent := renderScaffoldMetadata(parseSelection)
	var parseMetadata scaffoldMetadata
	if parseErr := json.Unmarshal([]byte(parseContent), &parseMetadata); parseErr != nil {
		parseT.Fatalf("unmarshal scaffold metadata: %v\n%s", parseErr, parseContent)
	}
	if parseMetadata.Ownership.ProjectOwnership != "framework-coupled" || parseMetadata.Ownership.FrameworkSourceMode != "local-replace" {
		parseT.Fatalf("expected contributor-linked ownership metadata, got %#v", parseMetadata.Ownership)
	}
}

func TestRenderScaffoldMetadataFallsBackWhenMarshalFails(parseT *testing.T) {
	parseOriginalMarshal := scaffoldMarshalIndent
	parseT.Cleanup(func() { scaffoldMarshalIndent = parseOriginalMarshal })
	scaffoldMarshalIndent = func(parseV any, parsePrefix string, parseIndent string) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}

	if parseGot := renderScaffoldMetadata(startSelection{}); parseGot != "{}\n" {
		parseT.Fatalf("expected marshal failure fallback payload, got %q", parseGot)
	}
}

func TestRenderScaffoldFeatureMatrixTracksSelectedCapabilities(parseT *testing.T) {
	parseSelection := startSelection{
		Preset: startPreset{
			Key:      "reference-app",
			Name:     "Reference App",
			Features: []string{"router", "fetch", "browser-tests", "router"},
		},
	}

	parseContent := renderScaffoldFeatureMatrix(parseSelection)
	for _, parseExpected := range []string{
		"- [x] `router`",
		"- [x] `fetch`",
		"- [x] `browser-tests`",
		"- [ ] `ssr`",
		"- `browser-tests`: Browser Tests",
	} {
		if !strings.Contains(parseContent, parseExpected) {
			parseT.Fatalf("expected feature matrix to contain %q, got:\n%s", parseExpected, parseContent)
		}
	}
}

func TestRenderScaffoldExtraFilesAddsBrowserTestScaffold(parseT *testing.T) {
	parseSelection := startSelection{
		ProjectName: "starter-app",
		Preset: startPreset{
			Key:      "reference-app",
			Name:     "Reference App",
			Features: []string{"router", "forms", "fetch", "hot-reload", "ui", "browser-tests"},
		},
	}

	parseFiles := renderScaffoldExtraFiles(parseSelection)
	for _, parseExpected := range []string{"FEATURE_MATRIX.md", "starter_test.go", ".github/workflows/ci.yml", "test/playwrightgo/README.md", "test/playwrightgo/smoke_test.go"} {
		if _, parseOk := parseFiles[parseExpected]; !parseOk {
			parseT.Fatalf("expected scaffold extra file %q to be generated", parseExpected)
		}
	}
	parseTestSource := string(parseFiles["starter_test.go"])
	for _, parseExpected2 := range []string{`"router"`, `"forms"`, `"fetch"`, `"hot-reload"`, `Active route: %s`, `Last submit marked complete.`, `Data status: %s`, `test/playwrightgo/smoke_test.go`} {
		if !strings.Contains(parseTestSource, parseExpected2) {
			parseT.Fatalf("expected generated starter_test.go to contain %q", parseExpected2)
		}
	}
	parseWorkflowSource := string(parseFiles[".github/workflows/ci.yml"])
	for _, parseExpected3 := range []string{"actions/checkout@v6", "actions/setup-go@v6", "go test ./...", "mkdir -p bin", "go build -o bin/main.wasm .", "GOOS: js", "GOARCH: wasm"} {
		if !strings.Contains(parseWorkflowSource, parseExpected3) {
			parseT.Fatalf("expected generated ci workflow to contain %q", parseExpected3)
		}
	}

	parseFiles = renderScaffoldExtraFiles(startSelection{Preset: startPreset{Features: []string{"ui"}}})
	if _, parseOk2 := parseFiles["starter_test.go"]; !parseOk2 {
		parseT.Fatal("expected baseline starter test scaffold to be generated")
	}
	if _, parseOk3 := parseFiles[".github/workflows/ci.yml"]; !parseOk3 {
		parseT.Fatal("expected baseline starter workflow to be generated")
	}
	if _, parseOk4 := parseFiles["test/playwrightgo/smoke_test.go"]; parseOk4 {
		parseT.Fatal("expected browser smoke test scaffold to be skipped without browser-tests capability")
	}
}

func TestRenderScaffoldOutputGolden(parseT *testing.T) {
	const repoModulePath = "github.com/monstercameron/GoWebComponents"
	parseRepoRoot := filepath.FromSlash("/repo/GoWebComponents")

	parseTests := []struct {
		name      string
		selection startSelection
	}{
		{
			name: "minimal-client-standalone",
			selection: startSelection{
				Preset: startPreset{
					Key:         "minimal-client",
					Name:        "Minimal Client App",
					Summary:     "Starter for pure client-side wasm apps.",
					Description: "Generated in golden tests.",
					Features:    []string{"ui", "html"},
				},
				ProjectName:   "golden-minimal-client",
				ModulePath:    "github.com/example/golden-minimal-client",
				Author:        "Golden Test",
				Version:       "1.0.0",
				Description:   "Golden output for the minimal client scaffold.",
				TargetDir:     filepath.FromSlash("/generated/golden-minimal-client"),
				SkipGoModTidy: true,
			},
		},
		{
			name: "ssr-app",
			selection: startSelection{
				Preset: startPreset{
					Key:         "ssr-app",
					Name:        "SSR App",
					Summary:     "Starter for request-time HTML rendering.",
					Description: "Generated in golden tests.",
					Features:    []string{"ssr", "hydration"},
				},
				ProjectName:   "golden-ssr-app",
				ModulePath:    "github.com/example/golden-ssr-app",
				Author:        "Golden Test",
				Version:       "1.0.0",
				Description:   "Golden output for the SSR scaffold.",
				TargetDir:     filepath.FromSlash("/generated/golden-ssr-app"),
				SkipGoModTidy: true,
			},
		},
		{
			name: "reference-app-browser-tests",
			selection: startSelection{
				Preset: startPreset{
					Key:         "reference-app",
					Name:        "Reference App",
					Summary:     "Starter for mixed routing, forms, and data flows.",
					Description: "Generated in golden tests.",
					Features:    []string{"router", "forms", "fetch", "state", "browser-tests"},
				},
				ProjectName:   "golden-reference-app",
				ModulePath:    "github.com/example/golden-reference-app",
				Author:        "Golden Test",
				Version:       "1.0.0",
				Description:   "Golden output for the reference app scaffold.",
				TargetDir:     filepath.FromSlash("/generated/golden-reference-app"),
				SkipGoModTidy: true,
			},
		},
		{
			name: "minimal-client-contributor-linked",
			selection: startSelection{
				Preset: startPreset{
					Key:         "minimal-client",
					Name:        "Minimal Client App",
					Summary:     "Starter for pure client-side wasm apps.",
					Description: "Generated in golden tests.",
					Features:    []string{"ui", "html"},
				},
				ProjectMode:   scaffoldProjectModeContributorLinked,
				ProjectName:   "golden-linked-client",
				ModulePath:    "github.com/example/golden-linked-client",
				Author:        "Golden Test",
				Version:       "1.0.0",
				Description:   "Golden output for the contributor-linked scaffold.",
				TargetDir:     filepath.FromSlash("/generated/golden-linked-client"),
				SkipGoModTidy: true,
			},
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			parseRendered := renderScaffoldGoldenFiles(parseTest.selection, repoModulePath, parseRepoRoot)
			parsePaths := make([]string, 0, len(parseRendered))
			for parsePath := range parseRendered {
				parsePaths = append(parsePaths, parsePath)
			}
			sort.Strings(parsePaths)
			for _, parseRelativePath := range parsePaths {
				parseGoldenPath := filepath.Join("testdata", "start_golden", parseTest.name, filepath.FromSlash(parseRelativePath))
				assertScaffoldGoldenFile(parseT2, parseGoldenPath, parseRendered[parseRelativePath])
			}
		})
	}
}

func renderScaffoldGoldenFiles(parseSelection startSelection, parseRepoModulePath string, parseRepoRoot string) map[string]string {
	parseFiles := map[string]string{
		"go.mod":         renderScaffoldGoMod(parseSelection, parseRepoModulePath, parseRepoRoot),
		"main.go":        renderScaffoldMain(parseSelection, parseRepoModulePath),
		"index.html":     renderScaffoldHTML(parseSelection),
		"gwc-start.json": renderScaffoldMetadata(parseSelection),
		"README.md":      renderScaffoldREADME(parseSelection),
	}
	for parsePath, parseContent := range renderScaffoldExtraFiles(parseSelection) {
		parseFiles[parsePath] = string(parseContent)
	}
	return parseFiles
}

func assertScaffoldGoldenFile(parseT *testing.T, parseGoldenPath string, parseGot string) {
	parseT.Helper()

	parseNormalized := normalizeScaffoldGoldenContent(parseGot)
	if os.Getenv("GWC_UPDATE_GOLDEN") == "1" {
		if parseErr := os.MkdirAll(filepath.Dir(parseGoldenPath), 0755); parseErr != nil {
			parseT.Fatalf("create golden dir %q: %v", filepath.Dir(parseGoldenPath), parseErr)
		}
		if parseErr2 := os.WriteFile(parseGoldenPath, []byte(parseNormalized), 0644); parseErr2 != nil {
			parseT.Fatalf("write golden file %q: %v", parseGoldenPath, parseErr2)
		}
		return
	}

	parseWantBytes, parseErr3 := os.ReadFile(parseGoldenPath)
	if parseErr3 != nil {
		parseT.Fatalf("read golden file %q: %v", parseGoldenPath, parseErr3)
	}
	if parseWant := normalizeScaffoldGoldenContent(string(parseWantBytes)); parseWant != parseNormalized {
		parseT.Fatalf("scaffold golden mismatch for %s", filepath.ToSlash(parseGoldenPath))
	}
}

func normalizeScaffoldGoldenContent(parseValue string) string {
	parseValue = strings.ReplaceAll(parseValue, "\r\n", "\n")
	return strings.ReplaceAll(parseValue, "\\", "/")
}

func TestRenderScaffoldMainIncludesFeatureCardsFromSelection(parseT *testing.T) {
	parseSelection := startSelection{
		Preset:      startPreset{Name: "Reference App", Features: []string{"router", "forms"}},
		ProjectName: "starter-app",
		Description: "starter description",
		Author:      "Cam",
		Version:     "1.0.0",
		ModulePath:  "example.com/starter-app",
	}

	parseMain := renderScaffoldMain(parseSelection, "github.com/monstercameron/GoWebComponents")
	for _, parseExpected := range []string{
		"Starter capability matrix",
		"Route shell and navigation affordances are scaffolded in the starter layout.",
		"Starter form interactions are scaffolded so teams can extend typed form state deliberately.",
	} {
		if !strings.Contains(parseMain, parseExpected) {
			parseT.Fatalf("expected generated main scaffold to contain %q", parseExpected)
		}
	}
}

func TestRenderScaffoldMainReferenceAppIncludesCommonPathWidgets(parseT *testing.T) {
	parseSelection := startSelection{
		Preset: startPreset{
			Key:      "reference-app",
			Name:     "Reference App",
			Features: []string{"router", "fetch", "state", "forms", "browser-tests", "release-profile"},
		},
		ProjectName: "starter-app",
		Description: "starter description",
		Author:      "Cam",
		Version:     "1.0.0",
		ModulePath:  "example.com/starter-app",
		TargetDir:   filepath.Join("C:\\tmp", "starter-app"),
	}

	parseMain := renderScaffoldMain(parseSelection, "github.com/monstercameron/GoWebComponents")
	for _, parseExpected := range []string{
		`html.Text("Routing")`,
		`html.Text("Async Data")`,
		`html.Text("Shared State")`,
		`html.Text("Forms")`,
		`Active route: %s`,
		`Data status: %s`,
		`Team members tracked: %d`,
	} {
		if !strings.Contains(parseMain, parseExpected) {
			parseT.Fatalf("expected common-path scaffold section %q in generated main.go", parseExpected)
		}
	}

	parseReadme := renderScaffoldREADME(parseSelection)
	for _, parseExpected2 := range []string{"go test ./...", "go test -tags playwrightgo ./test/playwrightgo -run TestMainSuite -v"} {
		if !strings.Contains(parseReadme, parseExpected2) {
			parseT.Fatalf("expected reference-app README to include %q, got:\n%s", parseExpected2, parseReadme)
		}
	}
}

func TestRenderScaffoldHTMLIncludesFirstPixelFallback(parseT *testing.T) {
	parseHTML := renderScaffoldHTML(startSelection{ProjectName: "starter-app"})
	for _, parseExpected := range []string{
		`<div id="gwc-first-pixel"`,
		`<div id="app"></div>`,
		`data-gwc-first-pixel="true"`,
		`role="status"`,
		`Preparing starter-app`,
		`Starting the wasm app...`,
		`new MutationObserver(removeFirstPixel).observe(appRoot, { childList: true });`,
		`WebAssembly.instantiateStreaming(fetch('./bin/main.wasm')`,
	} {
		if !strings.Contains(parseHTML, parseExpected) {
			parseT.Fatalf("expected scaffold HTML to contain %q, got:\n%s", parseExpected, parseHTML)
		}
	}
}

func TestSeedScaffoldGoSumAllowsMissingRepoFile(parseT *testing.T) {
	parseTargetDir := parseT.TempDir()
	parseLauncher := launcher{repoRoot: parseT.TempDir()}
	if parseErr := parseLauncher.seedScaffoldGoSum(parseTargetDir); parseErr != nil {
		parseT.Fatalf("expected missing repo go.sum to be ignored, got %v", parseErr)
	}
	if _, parseErr2 := os.Stat(filepath.Join(parseTargetDir, "go.sum")); !os.IsNotExist(parseErr2) {
		parseT.Fatalf("expected no scaffold go.sum to be written, got err=%v", parseErr2)
	}
}

func TestSeedScaffoldGoSumReadAndWriteErrors(parseT *testing.T) {
	parseRepoRoot := parseT.TempDir()
	if parseErr := os.Mkdir(filepath.Join(parseRepoRoot, "go.sum"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir repo go.sum dir: %v", parseErr)
	}
	if parseErr2 := (launcher{repoRoot: parseRepoRoot}).seedScaffoldGoSum(parseT.TempDir()); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "read repo go.sum") {
		parseT.Fatalf("expected repo go.sum read error, got %v", parseErr2)
	}

	parseRepoRoot = parseT.TempDir()
	if parseErr3 := os.WriteFile(filepath.Join(parseRepoRoot, "go.sum"), []byte("sum data"), 0644); parseErr3 != nil {
		parseT.Fatalf("write repo go.sum: %v", parseErr3)
	}
	parseTargetDir := parseT.TempDir()
	if parseErr4 := os.Mkdir(filepath.Join(parseTargetDir, "go.sum"), 0755); parseErr4 != nil {
		parseT.Fatalf("mkdir target go.sum dir: %v", parseErr4)
	}
	if parseErr5 := (launcher{repoRoot: parseRepoRoot}).seedScaffoldGoSum(parseTargetDir); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "write scaffold go.sum") {
		parseT.Fatalf("expected scaffold go.sum write error, got %v", parseErr5)
	}
}

func TestNormalizeExistingPathBlankInputReturnsStatError(parseT *testing.T) {
	parseResolved, parseErr := normalizeExistingPath(parseT.TempDir(), "   ")
	if parseErr == nil {
		parseT.Fatalf("expected blank existing path to fail stat lookup, got resolved=%q", parseResolved)
	}
}

func TestDevArgsFromScaffoldIncludesGeneratedPaths(parseT *testing.T) {
	parseResult := scaffoldResult{
		TargetDir: filepath.Join("tmp", "gwc-start", "app"),
		AppPath:   filepath.Join("tmp", "gwc-start", "app", "main.go"),
		HTMLPath:  filepath.Join("tmp", "gwc-start", "app", "index.html"),
	}

	parseGot := devArgsFromScaffold(parseResult)
	parseWant := []string{"-app", parseResult.AppPath, "-root", parseResult.TargetDir, "-html", parseResult.HTMLPath, "-wasm", scaffoldWASMOutputPath()}
	if len(parseGot) != len(parseWant) {
		parseT.Fatalf("expected %d dev args, got %d: %v", len(parseWant), len(parseGot), parseGot)
	}
	for parseI := range parseWant {
		if parseGot[parseI] != parseWant[parseI] {
			parseT.Fatalf("expected dev arg %d to be %q, got %q", parseI, parseWant[parseI], parseGot[parseI])
		}
	}
}

func TestSeedScaffoldGoSumCopiesRootGoSum(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{repoRoot: parseRepoRoot}
	parseTargetDir := parseT.TempDir()

	if parseErr2 := parseLauncher.seedScaffoldGoSum(parseTargetDir); parseErr2 != nil {
		parseT.Fatalf("seed scaffold go.sum: %v", parseErr2)
	}

	parseRootGoSum, parseErr := os.ReadFile(filepath.Join(parseRepoRoot, "go.sum"))
	if parseErr != nil {
		parseT.Fatalf("read repo go.sum: %v", parseErr)
	}
	parseTargetGoSum, parseErr := os.ReadFile(filepath.Join(parseTargetDir, "go.sum"))
	if parseErr != nil {
		parseT.Fatalf("read target go.sum: %v", parseErr)
	}
	if string(parseRootGoSum) != string(parseTargetGoSum) {
		parseT.Fatal("expected scaffold go.sum to match repo go.sum")
	}
}

func TestGenerateStartScaffoldWritesStarterFiles(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{repoRoot: parseRepoRoot}
	parseTargetDir := filepath.Join(defaultGeneratedScaffoldRoot(), "test-generate-start-scaffold")
	_ = os.RemoveAll(parseTargetDir)
	parseT.Cleanup(func() {
		_ = os.RemoveAll(parseTargetDir)
	})

	parseSelection := startSelection{
		Preset: startPreset{
			Key:         "minimal-client",
			Name:        "Minimal Client App",
			Summary:     "Test preset",
			Description: "Generated in tests.",
			Features:    []string{"ui", "html"},
		},
		ProjectName:   "test-generate-start-scaffold",
		ModulePath:    "github.com/example/test-generate-start-scaffold",
		Author:        "Test Author",
		Version:       "1.2.3",
		Description:   "Generated metadata test app.",
		TargetDir:     parseTargetDir,
		SkipGoModTidy: true,
	}

	parseResult, parseErr := parseLauncher.generateStartScaffold(parseSelection)
	if parseErr != nil {
		parseT.Fatalf("generate scaffold: %v", parseErr)
	}

	for _, parsePath := range []string{
		filepath.Join(parseTargetDir, "go.mod"),
		filepath.Join(parseTargetDir, "go.sum"),
		filepath.Join(parseTargetDir, "main.go"),
		filepath.Join(parseTargetDir, "index.html"),
		filepath.Join(parseTargetDir, "gwc-start.json"),
		filepath.Join(parseTargetDir, "FEATURE_MATRIX.md"),
		filepath.Join(parseTargetDir, "starter_test.go"),
		filepath.Join(parseTargetDir, ".github", "workflows", "ci.yml"),
		filepath.Join(parseTargetDir, "README.md"),
		filepath.Join(parseTargetDir, "wasm_exec.js"),
	} {
		if _, parseErr2 := os.Stat(parsePath); parseErr2 != nil {
			parseT.Fatalf("expected generated file %q: %v", parsePath, parseErr2)
		}
	}

	if parseResult.TargetDir != parseTargetDir {
		parseT.Fatalf("expected result target dir %q, got %q", parseTargetDir, parseResult.TargetDir)
	}
	if parseResult.AppPath != filepath.Join(parseTargetDir, "main.go") {
		parseT.Fatalf("expected result app path to point at generated main.go, got %q", parseResult.AppPath)
	}
	if parseResult.HTMLPath != filepath.Join(parseTargetDir, "index.html") {
		parseT.Fatalf("expected result html path to point at generated index.html, got %q", parseResult.HTMLPath)
	}
	ensureScaffoldUsesLocalRepoModule(parseT, parseRepoRoot, parseTargetDir)

	parseMetadataBytes, parseErr := os.ReadFile(filepath.Join(parseTargetDir, "gwc-start.json"))
	if parseErr != nil {
		parseT.Fatalf("read generated metadata file: %v", parseErr)
	}
	parseMetadataText := string(parseMetadataBytes)
	for _, parseExpected := range []string{"Test Author", "1.2.3", "Generated metadata test app."} {
		if !strings.Contains(parseMetadataText, parseExpected) {
			parseT.Fatalf("expected metadata file to contain %q", parseExpected)
		}
	}
	var parseMetadata scaffoldMetadata
	if parseErr3 := json.Unmarshal(parseMetadataBytes, &parseMetadata); parseErr3 != nil {
		parseT.Fatalf("unmarshal generated metadata: %v\n%s", parseErr3, string(parseMetadataBytes))
	}
	if parseMetadata.SchemaVersion != currentScaffoldMetadataSchemaVersion {
		parseT.Fatalf("expected generated metadata schema version %d, got %#v", currentScaffoldMetadataSchemaVersion, parseMetadata)
	}
	if parseMetadata.Tooling.AppPath != "main.go" {
		parseT.Fatalf("expected metadata app path main.go, got %#v", parseMetadata.Tooling)
	}
	if parseMetadata.Tooling.HTMLPath != "index.html" {
		parseT.Fatalf("expected metadata html path index.html, got %#v", parseMetadata.Tooling)
	}
	if parseMetadata.Tooling.WASMPath != filepath.ToSlash(filepath.Join("bin", "main.wasm")) {
		parseT.Fatalf("expected metadata wasm path bin/main.wasm, got %#v", parseMetadata.Tooling)
	}
	if parseMetadata.Tooling.DefaultBuildProfile != "development" {
		parseT.Fatalf("expected default build profile metadata, got %#v", parseMetadata.Tooling)
	}
	if parseMetadata.Tooling.ReleaseOutDir != filepath.ToSlash(filepath.Join("bin", "wasm-release")) {
		parseT.Fatalf("expected default release out dir metadata, got %#v", parseMetadata.Tooling)
	}
	if parseMetadata.Tooling.ReleaseBinaryName != "app.wasm" {
		parseT.Fatalf("expected default release binary metadata, got %#v", parseMetadata.Tooling)
	}
	if parseMetadata.Tooling.ReleaseCompression != "gzip+brotli" {
		parseT.Fatalf("expected default release compression metadata, got %#v", parseMetadata.Tooling)
	}
	if parseMetadata.Ownership.ProjectOwnership != "standalone" || parseMetadata.Ownership.FrameworkSourceMode != "module-proxy" {
		parseT.Fatalf("expected standalone ownership metadata, got %#v", parseMetadata.Ownership)
	}

	parseReadmeBytes, parseErr := os.ReadFile(filepath.Join(parseTargetDir, "README.md"))
	if parseErr != nil {
		parseT.Fatalf("read generated README file: %v", parseErr)
	}
	if !strings.Contains(string(parseReadmeBytes), "-wasm \"bin/main.wasm\"") {
		parseT.Fatalf("expected generated README to include explicit wasm output flag, got:\n%s", string(parseReadmeBytes))
	}
	for _, parseExpected2 := range []string{"## Starter Output Rules", "treat this scaffold as disposable starter code", "FEATURE_MATRIX.md", "## Verify", "go test ./..."} {
		if !strings.Contains(string(parseReadmeBytes), parseExpected2) {
			parseT.Fatalf("expected generated README to contain %q", parseExpected2)
		}
	}

	parseWorkflowBytes, parseErr := os.ReadFile(filepath.Join(parseTargetDir, ".github", "workflows", "ci.yml"))
	if parseErr != nil {
		parseT.Fatalf("read generated workflow file: %v", parseErr)
	}
	for _, parseExpected3 := range []string{"actions/checkout@v6", "actions/setup-go@v6", "go test ./...", "mkdir -p bin", "go build -o bin/main.wasm ."} {
		if !strings.Contains(string(parseWorkflowBytes), parseExpected3) {
			parseT.Fatalf("expected generated workflow to contain %q", parseExpected3)
		}
	}

	parseFeatureMatrixBytes, parseErr := os.ReadFile(filepath.Join(parseTargetDir, "FEATURE_MATRIX.md"))
	if parseErr != nil {
		parseT.Fatalf("read generated feature matrix file: %v", parseErr)
	}
	parseFeatureMatrixText := string(parseFeatureMatrixBytes)
	for _, parseExpected4 := range []string{"- [x] `ui`", "- [x] `html`", "- [ ] `router`"} {
		if !strings.Contains(parseFeatureMatrixText, parseExpected4) {
			parseT.Fatalf("expected generated feature matrix to contain %q", parseExpected4)
		}
	}
	if _, parseErr4 := os.Stat(filepath.Join(parseTargetDir, "test", "playwrightgo", "smoke_test.go")); !os.IsNotExist(parseErr4) {
		parseT.Fatalf("expected browser smoke test file to be absent for minimal scaffold, got err=%v", parseErr4)
	}

	parseHtmlBytes, parseErr := os.ReadFile(filepath.Join(parseTargetDir, "index.html"))
	if parseErr != nil {
		parseT.Fatalf("read generated html file: %v", parseErr)
	}
	parseHtmlText := string(parseHtmlBytes)
	for _, parseExpected5 := range []string{"color-scheme: dark", "radial-gradient(circle at top", "linear-gradient(160deg"} {
		if !strings.Contains(parseHtmlText, parseExpected5) {
			parseT.Fatalf("expected generated html to contain %q", parseExpected5)
		}
	}

	parseMainBytes, parseErr := os.ReadFile(filepath.Join(parseTargetDir, "main.go"))
	if parseErr != nil {
		parseT.Fatalf("read generated main.go file: %v", parseErr)
	}
	parseMainText := string(parseMainBytes)
	for _, parseExpected6 := range []string{"GoWebComponents Starter", "Dark-mode starter scaffold generated by gwc start"} {
		if !strings.Contains(parseMainText, parseExpected6) {
			parseT.Fatalf("expected generated main.go to contain %q", parseExpected6)
		}
	}

	if parseErr5 := os.MkdirAll(filepath.Join(parseTargetDir, "bin"), 0755); parseErr5 != nil {
		parseT.Fatalf("create scaffold bin directory: %v", parseErr5)
	}
	buildCmd := exec.Command("go", "build", "-o", filepath.Join("bin", "main.wasm"), ".")
	buildCmd.Dir = parseTargetDir
	buildCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	buildOutput, parseErr := buildCmd.CombinedOutput()
	if parseErr != nil {
		parseT.Fatalf("expected generated scaffold to build for wasm, got error: %v\n%s", parseErr, string(buildOutput))
	}

	parseTestCmd := exec.Command("go", "test", "./...")
	parseTestCmd.Dir = parseTargetDir
	parseTestCmd.Env = os.Environ()
	parseTestOutput, parseErr := parseTestCmd.CombinedOutput()
	if parseErr != nil {
		parseT.Fatalf("expected generated scaffold baseline tests to pass, got error: %v\n%s", parseErr, string(parseTestOutput))
	}
}

func TestGenerateStartScaffoldCIWorkflowMatchesStarterOutputs(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{repoRoot: parseRepoRoot}

	parseTests := []struct {
		name              string
		projectName       string
		preset            startPreset
		expectBrowserTest bool
		expectedFeatures  []string
	}{
		{
			name:        "minimal",
			projectName: "test-starter-ci-minimal",
			preset: startPreset{
				Key:         "minimal-client",
				Name:        "Minimal Client App",
				Summary:     "Test preset",
				Description: "Generated in tests.",
				Features:    []string{"ui", "html"},
			},
			expectedFeatures: []string{"ui", "html"},
		},
		{
			name:        "reference",
			projectName: "test-starter-ci-reference",
			preset: startPreset{
				Key:         "reference-app",
				Name:        "Reference App",
				Summary:     "Test preset",
				Description: "Generated in tests.",
				Features:    []string{"router", "forms", "fetch", "state", "browser-tests"},
			},
			expectBrowserTest: true,
			expectedFeatures:  []string{"router", "forms", "fetch", "state", "browser-tests"},
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			parseTargetDir := filepath.Join(parseT2.TempDir(), parseTest.projectName)
			parseResult, parseErr2 := parseLauncher.generateStartScaffold(startSelection{
				Preset:        parseTest.preset,
				ProjectName:   parseTest.projectName,
				ModulePath:    "github.com/example/" + parseTest.projectName,
				Author:        "Test Author",
				Version:       "1.2.3",
				Description:   "Generated metadata test app.",
				TargetDir:     parseTargetDir,
				SkipGoModTidy: true,
			})
			if parseErr2 != nil {
				parseT2.Fatalf("generate scaffold: %v", parseErr2)
			}

			if parseResult.TargetDir != parseTargetDir {
				parseT2.Fatalf("expected target dir %q, got %q", parseTargetDir, parseResult.TargetDir)
			}

			parseWorkflowBytes, parseErr2 := os.ReadFile(filepath.Join(parseTargetDir, ".github", "workflows", "ci.yml"))
			if parseErr2 != nil {
				parseT2.Fatalf("read workflow file: %v", parseErr2)
			}
			parseWorkflowText := string(parseWorkflowBytes)
			for _, parseExpected := range []string{"go test ./...", "mkdir -p bin", "go build -o bin/main.wasm .", "GOOS: js", "GOARCH: wasm"} {
				if !strings.Contains(parseWorkflowText, parseExpected) {
					parseT2.Fatalf("expected workflow to contain %q", parseExpected)
				}
			}

			for _, parseRequiredPath := range []string{
				filepath.Join(parseTargetDir, "starter_test.go"),
				filepath.Join(parseTargetDir, "main.go"),
				filepath.Join(parseTargetDir, "FEATURE_MATRIX.md"),
			} {
				if _, parseErr3 := os.Stat(parseRequiredPath); parseErr3 != nil {
					parseT2.Fatalf("expected starter output %q: %v", parseRequiredPath, parseErr3)
				}
			}

			parseStarterTestBytes, parseErr2 := os.ReadFile(filepath.Join(parseTargetDir, "starter_test.go"))
			if parseErr2 != nil {
				parseT2.Fatalf("read starter_test.go: %v", parseErr2)
			}
			parseStarterTestText := string(parseStarterTestBytes)
			for _, parseFeature := range parseTest.expectedFeatures {
				if !strings.Contains(parseStarterTestText, fmt.Sprintf("%q", parseFeature)) {
					parseT2.Fatalf("expected starter_test.go to mention feature %q", parseFeature)
				}
			}

			parseBrowserSmokePath := filepath.Join(parseTargetDir, "test", "playwrightgo", "smoke_test.go")
			_, parseBrowserErr := os.Stat(parseBrowserSmokePath)
			if parseTest.expectBrowserTest && parseBrowserErr != nil {
				parseT2.Fatalf("expected browser smoke test scaffold: %v", parseBrowserErr)
			}
			if !parseTest.expectBrowserTest && !os.IsNotExist(parseBrowserErr) {
				parseT2.Fatalf("expected no browser smoke test scaffold, got err=%v", parseBrowserErr)
			}
		})
	}
}

func TestDefaultStarterTemplatesScaffoldTidyTestAndBuild(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{repoRoot: parseRepoRoot}
	parsePresets := defaultStartPresets()
	if len(parsePresets) == 0 {
		parseT.Fatal("expected at least one starter preset")
	}

	for _, parsePreset := range parsePresets {
		parseT.Run(parsePreset.Key, func(parseT2 *testing.T) {
			parseProjectName := "test-starter-" + parsePreset.Key
			parseTargetDir := filepath.Join(parseT2.TempDir(), parseProjectName)
			parseResult, parseErr2 := parseLauncher.generateStartScaffold(startSelection{
				Preset:        parsePreset,
				ProjectMode:   scaffoldProjectModeContributorLinked,
				ProjectName:   parseProjectName,
				ModulePath:    "github.com/example/" + parseProjectName,
				Author:        "Test Author",
				Version:       "1.2.3",
				Description:   parsePreset.Summary,
				TargetDir:     parseTargetDir,
				SkipGoModTidy: false,
			})
			if parseErr2 != nil {
				parseT2.Fatalf("generate starter scaffold: %v", parseErr2)
			}

			parseTestCmd := exec.Command("go", "test", "./...")
			parseTestCmd.Dir = parseTargetDir
			parseTestCmd.Env = os.Environ()
			parseTestOutput, parseErr2 := parseTestCmd.CombinedOutput()
			if parseErr2 != nil {
				parseT2.Fatalf("generated starter tests failed: %v\n%s", parseErr2, string(parseTestOutput))
			}

			parseBuildErr := parseLauncher.runBuild([]string{
				"-app", parseResult.AppPath,
				"-root", parseResult.TargetDir,
				"-out", filepath.Join(parseResult.TargetDir, filepath.FromSlash(scaffoldWASMOutputPath())),
				"-profile", "development",
			})
			if parseBuildErr != nil {
				parseT2.Fatalf("gwc build failed for starter %q: %v", parsePreset.Key, parseBuildErr)
			}
			if parseInfo, parseErr3 := os.Stat(filepath.Join(parseTargetDir, filepath.FromSlash(scaffoldWASMOutputPath()))); parseErr3 != nil || parseInfo.Size() == 0 {
				parseT2.Fatalf("expected non-empty wasm output for starter %q, info=%v err=%v", parsePreset.Key, parseInfo, parseErr3)
			}
		})
	}
}

func TestGenerateStartScaffoldWritesContributorLinkedGoMod(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{repoRoot: parseRepoRoot}
	parseTargetDir := filepath.Join(parseT.TempDir(), "test-linked-starter")

	_, parseErr = parseLauncher.generateStartScaffold(startSelection{
		Preset: startPreset{
			Key:         "minimal-client",
			Name:        "Minimal Client App",
			Summary:     "Test preset",
			Description: "Generated in tests.",
			Features:    []string{"ui", "html"},
		},
		ProjectMode:   scaffoldProjectModeContributorLinked,
		ProjectName:   "test-linked-starter",
		ModulePath:    "github.com/example/test-linked-starter",
		Author:        "Test Author",
		Version:       "1.2.3",
		Description:   "Generated metadata test app.",
		TargetDir:     parseTargetDir,
		SkipGoModTidy: true,
	})
	if parseErr != nil {
		parseT.Fatalf("generate contributor-linked scaffold: %v", parseErr)
	}

	parseGoModBytes, parseErr := os.ReadFile(filepath.Join(parseTargetDir, "go.mod"))
	if parseErr != nil {
		parseT.Fatalf("read contributor-linked go.mod: %v", parseErr)
	}
	parseExpectedReplace := "replace github.com/monstercameron/GoWebComponents => " + filepath.ToSlash(filepath.Clean(parseRepoRoot))
	if !strings.Contains(string(parseGoModBytes), parseExpectedReplace) {
		parseT.Fatalf("expected contributor-linked go.mod to contain %q, got:\n%s", parseExpectedReplace, string(parseGoModBytes))
	}

	parseMetadataBytes, parseErr := os.ReadFile(filepath.Join(parseTargetDir, "gwc-start.json"))
	if parseErr != nil {
		parseT.Fatalf("read contributor-linked metadata: %v", parseErr)
	}
	var parseMetadata scaffoldMetadata
	if parseErr2 := json.Unmarshal(parseMetadataBytes, &parseMetadata); parseErr2 != nil {
		parseT.Fatalf("unmarshal contributor-linked metadata: %v\n%s", parseErr2, string(parseMetadataBytes))
	}
	if parseMetadata.Ownership.ProjectOwnership != "framework-coupled" || parseMetadata.Ownership.FrameworkSourceMode != "local-replace" {
		parseT.Fatalf("expected contributor-linked ownership metadata, got %#v", parseMetadata.Ownership)
	}
}

func TestRunDevDryRunJSONResolvesGeneratedScaffoldPlan(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{repoRoot: parseRepoRoot}
	parseTargetDir := filepath.Join(defaultGeneratedScaffoldRoot(), "test-run-dev-dry-run-json")
	_ = os.RemoveAll(parseTargetDir)
	parseT.Cleanup(func() {
		_ = os.RemoveAll(parseTargetDir)
	})

	parseSelection := startSelection{
		Preset: startPreset{
			Key:         "minimal-client",
			Name:        "Minimal Client App",
			Summary:     "Test preset",
			Description: "Generated in tests.",
			Features:    []string{"ui", "html"},
		},
		ProjectName:   "test-run-dev-dry-run-json",
		ModulePath:    "github.com/example/test-run-dev-dry-run-json",
		Author:        "Test Author",
		Version:       "1.2.3",
		Description:   "Generated metadata test app.",
		TargetDir:     parseTargetDir,
		SkipGoModTidy: true,
	}

	parseResult, parseErr := parseLauncher.generateStartScaffold(parseSelection)
	if parseErr != nil {
		parseT.Fatalf("generate scaffold: %v", parseErr)
	}

	parseWorkingDir, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatalf("get working dir: %v", parseErr)
	}
	if parseErr2 := os.Chdir(parseRepoRoot); parseErr2 != nil {
		parseT.Fatalf("chdir repo root: %v", parseErr2)
	}
	parseT.Cleanup(func() {
		_ = os.Chdir(parseWorkingDir)
	})

	parseStdout, parseRestoreStdout, parseErr := captureStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	parseArgs := append(devArgsFromScaffold(parseResult), "-dry-run", "-json", "-port", "8137")
	if parseErr3 := parseLauncher.runDev(parseArgs); parseErr3 != nil {
		parseT.Fatalf("run dev dry-run json: %v", parseErr3)
	}

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr)
	}

	var parsePlan struct {
		App  string `json:"app"`
		Root string `json:"root"`
		HTML string `json:"html"`
		WASM string `json:"wasm"`
		Host string `json:"host"`
		Port string `json:"port"`
		Hot  bool   `json:"hot"`
	}
	if parseErr4 := json.Unmarshal([]byte(parseOutput), &parsePlan); parseErr4 != nil {
		parseT.Fatalf("unmarshal dev plan json: %v\noutput: %s", parseErr4, parseOutput)
	}

	if parsePlan.App != parseResult.AppPath {
		parseT.Fatalf("expected app path %q, got %q", parseResult.AppPath, parsePlan.App)
	}
	if parsePlan.Root != parseResult.TargetDir {
		parseT.Fatalf("expected root path %q, got %q", parseResult.TargetDir, parsePlan.Root)
	}
	if parsePlan.HTML != parseResult.HTMLPath {
		parseT.Fatalf("expected html path %q, got %q", parseResult.HTMLPath, parsePlan.HTML)
	}
	if parsePlan.WASM != scaffoldWASMOutputPath() {
		parseT.Fatalf("expected wasm path %q, got %q", scaffoldWASMOutputPath(), parsePlan.WASM)
	}
	if parsePlan.Host != "127.0.0.1" {
		parseT.Fatalf("expected default host 127.0.0.1, got %q", parsePlan.Host)
	}
	if parsePlan.Port != "8137" {
		parseT.Fatalf("expected port 8137, got %q", parsePlan.Port)
	}
	if !parsePlan.Hot {
		parseT.Fatal("expected hot reload to remain enabled in dry-run plan")
	}
}

func TestRunDevFailsForMissingAppPath(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseLauncher := launcher{repoRoot: parseRoot}

	parseErr := parseLauncher.runDev([]string{"-app", filepath.Join(parseRoot, "missing.go"), "-root", parseRoot, "-dry-run"})
	if parseErr == nil {
		parseT.Fatal("expected missing app path to fail")
	}
	if !strings.Contains(parseErr.Error(), "resolve app path") {
		parseT.Fatalf("expected missing app path error, got %v", parseErr)
	}
}

func TestRunDevExecutesForwardedCommandAndSurfacesFailure(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseAppDir := filepath.Join(parseRoot, "app")
	if parseErr := os.MkdirAll(parseAppDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir app dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Join(parseRoot, "tools", "livereload"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir livereload dir: %v", parseErr2)
	}
	parseAppPath := filepath.Join(parseAppDir, "main.go")
	parseHtmlPath := filepath.Join(parseAppDir, "index.html")
	if parseErr3 := os.WriteFile(parseAppPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write main.go: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(parseHtmlPath, []byte("<html></html>\n"), 0644); parseErr4 != nil {
		parseT.Fatalf("write index.html: %v", parseErr4)
	}

	parseBinDir := filepath.Join(parseRoot, "fake-bin")
	if parseErr5 := os.MkdirAll(parseBinDir, 0755); parseErr5 != nil {
		parseT.Fatalf("mkdir fake bin: %v", parseErr5)
	}
	parseArgsPath := filepath.Join(parseRoot, "go-args.txt")
	parseGoBatPath := filepath.Join(parseBinDir, "go.bat")
	parseGoBat := "@echo off\r\n" +
		"echo %*>\"" + parseArgsPath + "\"\r\n" +
		"if /I \"%1\"==\"fail-now\" exit /b 9\r\n" +
		"exit /b 0\r\n"
	if parseErr6 := os.WriteFile(parseGoBatPath, []byte(parseGoBat), 0644); parseErr6 != nil {
		parseT.Fatalf("write fake go.bat: %v", parseErr6)
	}
	parseT.Setenv("PATH", parseBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	parseDevLauncher := launcher{repoRoot: parseRoot}
	if parseErr7 := parseDevLauncher.runDev([]string{"-main", parseAppPath, "-root", parseAppDir, "-index", parseHtmlPath, "-output", "dist/app.wasm", "-host", "0.0.0.0", "-port", "9123", "-hot=false", "-client-script", "custom-client.js"}); parseErr7 != nil {
		parseT.Fatalf("run dev forwarded command: %v", parseErr7)
	}
	parseArgsBytes, parseErr8 := os.ReadFile(parseArgsPath)
	if parseErr8 != nil {
		parseT.Fatalf("read forwarded args: %v", parseErr8)
	}
	parseArgsText := string(parseArgsBytes)
	for _, parseExpected := range []string{"run tools/livereload/client_script.go tools/livereload/livereload.go tools/livereload/livereload_http.go tools/livereload/livereload_paths.go tools/livereload/livereload_support.go", "-app " + parseAppPath, "-root " + parseAppDir, "-html " + parseHtmlPath, "-wasm dist/app.wasm", "-host 0.0.0.0", "-port 9123", "-hot false", "-client-script custom-client.js"} {
		if !strings.Contains(parseArgsText, parseExpected) {
			parseT.Fatalf("expected forwarded args to contain %q, got %q", parseExpected, parseArgsText)
		}
	}

	if parseErr9 := os.WriteFile(parseGoBatPath, []byte("@echo off\r\nexit /b 9\r\n"), 0644); parseErr9 != nil {
		parseT.Fatalf("rewrite failing go.bat: %v", parseErr9)
	}
	parseErr8 = parseDevLauncher.runDev([]string{"-app", parseAppPath, "-root", parseAppDir, "-html", parseHtmlPath})
	if parseErr8 == nil {
		parseT.Fatal("expected forwarded go run failure")
	}
}

func TestScaffoldHelperFailureBranches(parseT *testing.T) {
	parseOriginalResolveWasmExecPath := scaffoldResolveWasmExecPath
	parseOriginalScaffoldWriteFile := scaffoldWriteFile
	parseOriginalScaffoldReadFile := scaffoldReadFile
	parseOriginalScaffoldFormatMain := scaffoldFormatMain
	parseT.Cleanup(func() {
		scaffoldResolveWasmExecPath = parseOriginalResolveWasmExecPath
		scaffoldWriteFile = parseOriginalScaffoldWriteFile
		scaffoldReadFile = parseOriginalScaffoldReadFile
		scaffoldFormatMain = parseOriginalScaffoldFormatMain
	})

	parseT.Run("seed scaffold go.sum tolerates missing root file", func(parseT2 *testing.T) {
		parseTestLauncher := launcher{repoRoot: parseT2.TempDir()}
		if parseErr := parseTestLauncher.seedScaffoldGoSum(parseT2.TempDir()); parseErr != nil {
			parseT2.Fatalf("expected missing repo go.sum to be ignored, got %v", parseErr)
		}
	})

	parseT.Run("seed scaffold go.sum read error", func(parseT3 *testing.T) {
		parseRepoRoot := parseT3.TempDir()
		if parseErr2 := os.MkdirAll(filepath.Join(parseRepoRoot, "go.sum"), 0755); parseErr2 != nil {
			parseT3.Fatalf("mkdir go.sum dir: %v", parseErr2)
		}
		parseTestLauncher2 := launcher{repoRoot: parseRepoRoot}
		parseErr3 := parseTestLauncher2.seedScaffoldGoSum(parseT3.TempDir())
		if parseErr3 == nil || !strings.Contains(parseErr3.Error(), "read repo go.sum") {
			parseT3.Fatalf("expected repo go.sum read error, got %v", parseErr3)
		}
	})

	parseT.Run("read repo module path failures", func(parseT4 *testing.T) {
		parseTestLauncher3 := launcher{repoRoot: parseT4.TempDir()}
		if _, parseErr4 := parseTestLauncher3.readRepoModulePath(); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "read repo go.mod") {
			parseT4.Fatalf("expected missing go.mod error, got %v", parseErr4)
		}

		parseRepoRoot2 := parseT4.TempDir()
		if parseErr5 := os.WriteFile(filepath.Join(parseRepoRoot2, "go.mod"), []byte("go 1.25.0\n"), 0644); parseErr5 != nil {
			parseT4.Fatalf("write go.mod: %v", parseErr5)
		}
		parseTestLauncher3 = launcher{repoRoot: parseRepoRoot2}
		if _, parseErr6 := parseTestLauncher3.readRepoModulePath(); parseErr6 == nil || !strings.Contains(parseErr6.Error(), "repo module path not found") {
			parseT4.Fatalf("expected missing module line error, got %v", parseErr6)
		}
	})

	parseT.Run("generate scaffold fails before file writes when repo metadata is missing", func(parseT5 *testing.T) {
		parseTargetDir := filepath.Join(parseT5.TempDir(), "starter-app")
		parseSelection := startSelection{
			Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
			ProjectName: "starter-app",
			ModulePath:  "example.com/starter-app",
			Author:      "Cam",
			Version:     "0.1.0",
			Description: "starter",
			TargetDir:   parseTargetDir,
		}
		parseTestLauncher4 := launcher{repoRoot: parseT5.TempDir()}
		_, parseErr7 := parseTestLauncher4.generateStartScaffold(parseSelection)
		if parseErr7 == nil || !strings.Contains(parseErr7.Error(), "read repo go.mod") {
			parseT5.Fatalf("expected missing repo module path error, got %v", parseErr7)
		}
	})

	parseT.Run("generate scaffold fails when scaffold module cannot be prepared", func(parseT6 *testing.T) {
		parseOriginalGetwd := launcherConfigGetwd
		parseT6.Cleanup(func() { launcherConfigGetwd = parseOriginalGetwd })
		parseRepoRoot3 := parseT6.TempDir()
		if parseErr8 := os.WriteFile(filepath.Join(parseRepoRoot3, "go.mod"), []byte("module example.com/repo\n\ngo 1.25.0\n"), 0644); parseErr8 != nil {
			parseT6.Fatalf("write repo go.mod: %v", parseErr8)
		}
		launcherConfigGetwd = func() (string, error) { return parseRepoRoot3, nil }
		parseT6.Setenv(launcherOverrideEnvVar, "")
		parseSelection2 := startSelection{
			Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
			ProjectName: "starter-app",
			ModulePath:  "example.com/starter-app",
			Author:      "Cam",
			Version:     "0.1.0",
			Description: "starter",
			TargetDir:   filepath.Join(parseT6.TempDir(), "starter-app"),
		}
		parseTestLauncher5 := launcher{repoRoot: parseRepoRoot3}
		_, parseErr9 := parseTestLauncher5.generateStartScaffold(parseSelection2)
		if parseErr9 == nil || !strings.Contains(parseErr9.Error(), "prepare scaffold module") {
			parseT6.Fatalf("expected scaffold module preparation error, got %v", parseErr9)
		}
	})

	parseT.Run("tidy scaffold module surfaces blank output failures", func(parseT7 *testing.T) {
		parseBinDir := filepath.Join(parseT7.TempDir(), "bin")
		if parseErr10 := os.MkdirAll(parseBinDir, 0755); parseErr10 != nil {
			parseT7.Fatalf("mkdir fake bin: %v", parseErr10)
		}
		parseGoBatPath := filepath.Join(parseBinDir, "go.bat")
		if parseErr11 := os.WriteFile(parseGoBatPath, []byte("@echo off\r\nexit /b 7\r\n"), 0644); parseErr11 != nil {
			parseT7.Fatalf("write fake go.bat: %v", parseErr11)
		}
		parseT7.Setenv("PATH", parseBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		parseErr12 := (launcher{}).tidyScaffoldModule(parseT7.TempDir())
		if parseErr12 == nil || !strings.Contains(parseErr12.Error(), "prepare scaffold module: exit status") {
			parseT7.Fatalf("expected blank-output tidy error, got %v", parseErr12)
		}
	})

	parseT.Run("generate scaffold fails when wasm_exec override does not point to a file", func(parseT8 *testing.T) {
		parseRepoRoot4 := parseT8.TempDir()
		if parseErr13 := os.WriteFile(filepath.Join(parseRepoRoot4, "go.mod"), []byte("module example.com/repo\n\ngo 1.25.0\n"), 0644); parseErr13 != nil {
			parseT8.Fatalf("write repo go.mod: %v", parseErr13)
		}

		parseOverrideDir := filepath.Join(parseT8.TempDir(), "vendor", "wasm_exec.js")
		if parseErr14 := os.MkdirAll(parseOverrideDir, 0755); parseErr14 != nil {
			parseT8.Fatalf("mkdir override dir: %v", parseErr14)
		}
		parseOverrides := launcherOverrides{Paths: launcherOverridePaths{WASMExecJS: parseOverrideDir}}
		parseOverrideBytes, parseErr15 := json.Marshal(parseOverrides)
		if parseErr15 != nil {
			parseT8.Fatalf("marshal overrides: %v", parseErr15)
		}
		parseConfigPath := filepath.Join(parseT8.TempDir(), "runner.json")
		if parseErr16 := os.WriteFile(parseConfigPath, parseOverrideBytes, 0644); parseErr16 != nil {
			parseT8.Fatalf("write runner config: %v", parseErr16)
		}
		parseT8.Setenv(launcherOverrideEnvVar, parseConfigPath)

		parseSelection3 := startSelection{
			Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
			ProjectName: "starter-app",
			ModulePath:  "example.com/starter-app",
			Author:      "Cam",
			Version:     "0.1.0",
			Description: "starter",
			TargetDir:   filepath.Join(parseT8.TempDir(), "starter-app"),
		}
		parseTestLauncher6 := launcher{repoRoot: parseRepoRoot4}
		_, parseErr15 = parseTestLauncher6.generateStartScaffold(parseSelection3)
		if parseErr15 == nil || !strings.Contains(parseErr15.Error(), "configured wasmExecJS path does not exist") {
			parseT8.Fatalf("expected missing wasm_exec.js override error, got %v", parseErr15)
		}
	})

	parseT.Run("generate scaffold fails when gofmt fails", func(parseT9 *testing.T) {
		parseRepoRoot5 := parseT9.TempDir()
		if parseErr17 := os.WriteFile(filepath.Join(parseRepoRoot5, "go.mod"), []byte("module example.com/repo\n\ngo 1.25.0\n"), 0644); parseErr17 != nil {
			parseT9.Fatalf("write repo go.mod: %v", parseErr17)
		}

		parseWasmExecPath := filepath.Join(parseT9.TempDir(), "wasm_exec.js")
		if parseErr18 := os.WriteFile(parseWasmExecPath, []byte("console.log('wasm');\n"), 0644); parseErr18 != nil {
			parseT9.Fatalf("write wasm_exec.js: %v", parseErr18)
		}
		parseOverrides2 := launcherOverrides{Paths: launcherOverridePaths{WASMExecJS: parseWasmExecPath}}
		parseOverrideBytes2, parseErr19 := json.Marshal(parseOverrides2)
		if parseErr19 != nil {
			parseT9.Fatalf("marshal overrides: %v", parseErr19)
		}
		parseConfigPath2 := filepath.Join(parseT9.TempDir(), "runner.json")
		if parseErr20 := os.WriteFile(parseConfigPath2, parseOverrideBytes2, 0644); parseErr20 != nil {
			parseT9.Fatalf("write runner config: %v", parseErr20)
		}
		parseT9.Setenv(launcherOverrideEnvVar, parseConfigPath2)

		parseBinDir2 := filepath.Join(parseT9.TempDir(), "bin")
		if parseErr21 := os.MkdirAll(parseBinDir2, 0755); parseErr21 != nil {
			parseT9.Fatalf("mkdir fake bin: %v", parseErr21)
		}
		if parseErr22 := os.WriteFile(filepath.Join(parseBinDir2, "go.bat"), []byte("@echo off\r\nexit /b 0\r\n"), 0644); parseErr22 != nil {
			parseT9.Fatalf("write fake go.bat: %v", parseErr22)
		}
		if parseErr23 := os.WriteFile(filepath.Join(parseBinDir2, "gofmt.bat"), []byte("@echo off\r\nexit /b 9\r\n"), 0644); parseErr23 != nil {
			parseT9.Fatalf("write fake gofmt.bat: %v", parseErr23)
		}
		parseT9.Setenv("PATH", parseBinDir2+string(os.PathListSeparator)+os.Getenv("PATH"))

		parseSelection4 := startSelection{
			Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
			ProjectName: "starter-app",
			ModulePath:  "example.com/starter-app",
			Author:      "Cam",
			Version:     "0.1.0",
			Description: "starter",
			TargetDir:   filepath.Join(parseT9.TempDir(), "starter-app"),
		}
		_, parseErr19 = (launcher{repoRoot: parseRepoRoot5}).generateStartScaffold(parseSelection4)
		if parseErr19 == nil || !strings.Contains(parseErr19.Error(), "format generated main.go") {
			parseT9.Fatalf("expected gofmt failure, got %v", parseErr19)
		}
	})

	parseT.Run("generate scaffold injected write failures", func(parseT10 *testing.T) {
		parseRepoRoot6 := parseT10.TempDir()
		if parseErr24 := os.WriteFile(filepath.Join(parseRepoRoot6, "go.mod"), []byte("module example.com/repo\n\ngo 1.25.0\n"), 0644); parseErr24 != nil {
			parseT10.Fatalf("write repo go.mod: %v", parseErr24)
		}
		parseWasmExecPath2 := filepath.Join(parseT10.TempDir(), "wasm_exec.js")
		if parseErr25 := os.WriteFile(parseWasmExecPath2, []byte("console.log('wasm');\n"), 0644); parseErr25 != nil {
			parseT10.Fatalf("write wasm_exec.js: %v", parseErr25)
		}
		scaffoldResolveWasmExecPath = func() (string, error) { return parseWasmExecPath2, nil }
		scaffoldFormatMain = func(string) error { return nil }

		parseNewSelection := func() startSelection {
			return startSelection{
				Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
				ProjectName: "starter-app",
				ModulePath:  "example.com/starter-app",
				Author:      "Cam",
				Version:     "0.1.0",
				Description: "starter",
				TargetDir:   filepath.Join(parseT10.TempDir(), "starter-app"),
			}
		}

		parseWriteFailures := []struct {
			name    string
			suffix  string
			wantErr string
		}{
			{name: "go mod", suffix: "go.mod", wantErr: "write go.mod"},
			{name: "main go", suffix: "main.go", wantErr: "write main.go"},
			{name: "index html", suffix: "index.html", wantErr: "write index.html"},
			{name: "metadata", suffix: "gwc-start.json", wantErr: "write gwc-start.json"},
			{name: "readme", suffix: "README.md", wantErr: "write README.md"},
			{name: "wasm exec write", suffix: "wasm_exec.js", wantErr: "write wasm_exec.js"},
		}
		for _, parseTestCase := range parseWriteFailures {
			parseT10.Run(parseTestCase.name, func(parseT11 *testing.T) {
				scaffoldWriteFile = func(parseName2 string, parseData []byte, parsePerm os.FileMode) error {
					if strings.HasSuffix(filepath.ToSlash(parseName2), parseTestCase.suffix) {
						return errors.New("write failed")
					}
					return parseOriginalScaffoldWriteFile(parseName2, parseData, parsePerm)
				}
				defer func() { scaffoldWriteFile = parseOriginalScaffoldWriteFile }()

				_, parseErr26 := (launcher{repoRoot: parseRepoRoot6}).generateStartScaffold(parseNewSelection())
				if parseErr26 == nil || !strings.Contains(parseErr26.Error(), parseTestCase.wantErr) {
					parseT11.Fatalf("expected %s failure, got %v", parseTestCase.wantErr, parseErr26)
				}
			})
		}
	})

	parseT.Run("generate scaffold injected read and seed failures", func(parseT12 *testing.T) {
		parseRepoRoot7 := parseT12.TempDir()
		if parseErr27 := os.WriteFile(filepath.Join(parseRepoRoot7, "go.mod"), []byte("module example.com/repo\n\ngo 1.25.0\n"), 0644); parseErr27 != nil {
			parseT12.Fatalf("write repo go.mod: %v", parseErr27)
		}
		if parseErr28 := os.MkdirAll(filepath.Join(parseRepoRoot7, "go.sum"), 0755); parseErr28 != nil {
			parseT12.Fatalf("mkdir repo go.sum dir: %v", parseErr28)
		}
		parseWasmExecPath3 := filepath.Join(parseT12.TempDir(), "wasm_exec.js")
		if parseErr29 := os.WriteFile(parseWasmExecPath3, []byte("console.log('wasm');\n"), 0644); parseErr29 != nil {
			parseT12.Fatalf("write wasm_exec.js: %v", parseErr29)
		}
		scaffoldResolveWasmExecPath = func() (string, error) { return parseWasmExecPath3, nil }
		scaffoldFormatMain = func(string) error { return nil }

		parseNewSelection2 := func() startSelection {
			return startSelection{
				Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
				ProjectName: "starter-app",
				ModulePath:  "example.com/starter-app",
				Author:      "Cam",
				Version:     "0.1.0",
				Description: "starter",
				TargetDir:   filepath.Join(parseT12.TempDir(), "starter-app"),
			}
		}

		scaffoldReadFile = func(parseName3 string) ([]byte, error) {
			if filepath.Clean(parseName3) == filepath.Clean(parseWasmExecPath3) {
				return nil, errors.New("read failed")
			}
			return parseOriginalScaffoldReadFile(parseName3)
		}
		_, parseErr30 := (launcher{repoRoot: parseRepoRoot7}).generateStartScaffold(parseNewSelection2())
		if parseErr30 == nil || !strings.Contains(parseErr30.Error(), "read wasm_exec.js") {
			parseT12.Fatalf("expected wasm_exec read failure, got %v", parseErr30)
		}
		scaffoldReadFile = parseOriginalScaffoldReadFile

		_, parseErr30 = (launcher{repoRoot: parseRepoRoot7}).generateStartScaffold(parseNewSelection2())
		if parseErr30 == nil || !strings.Contains(parseErr30.Error(), "read repo go.sum") {
			parseT12.Fatalf("expected seed scaffold go.sum failure, got %v", parseErr30)
		}
	})
}

func TestGenerateStartScaffoldEarlyValidationBranches(parseT *testing.T) {
	parseT.Run("rejects filesystem root target", func(parseT2 *testing.T) {
		parseTargetDir := string(os.PathSeparator)
		if parseVolume := filepath.VolumeName(parseTargetDir); parseVolume != "" {
			parseTargetDir = parseVolume + string(os.PathSeparator)
		}
		parseSelection := startSelection{
			Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
			ProjectName: "starter-app",
			ModulePath:  "example.com/starter-app",
			Author:      "Cam",
			Version:     "0.1.0",
			Description: "starter",
			TargetDir:   parseTargetDir,
		}
		_, parseErr := (launcher{repoRoot: parseT2.TempDir()}).generateStartScaffold(parseSelection)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "filesystem root") {
			parseT2.Fatalf("expected target root validation error, got %v", parseErr)
		}
	})

	parseT.Run("rejects existing file target", func(parseT3 *testing.T) {
		parseTargetDir2 := filepath.Join(parseT3.TempDir(), "existing-target")
		if parseErr2 := os.WriteFile(parseTargetDir2, []byte("occupied"), 0644); parseErr2 != nil {
			parseT3.Fatalf("write target file: %v", parseErr2)
		}
		parseSelection2 := startSelection{
			Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
			ProjectName: "starter-app",
			ModulePath:  "example.com/starter-app",
			Author:      "Cam",
			Version:     "0.1.0",
			Description: "starter",
			TargetDir:   parseTargetDir2,
		}
		_, parseErr3 := (launcher{repoRoot: parseT3.TempDir()}).generateStartScaffold(parseSelection2)
		if parseErr3 == nil || !strings.Contains(parseErr3.Error(), "not a directory") {
			parseT3.Fatalf("expected existing file target error, got %v", parseErr3)
		}
	})

	parseT.Run("rejects target under file parent", func(parseT4 *testing.T) {
		parseParentFile := filepath.Join(parseT4.TempDir(), "occupied")
		if parseErr4 := os.WriteFile(parseParentFile, []byte("occupied"), 0644); parseErr4 != nil {
			parseT4.Fatalf("write parent file: %v", parseErr4)
		}
		parseSelection3 := startSelection{
			Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
			ProjectName: "starter-app",
			ModulePath:  "example.com/starter-app",
			Author:      "Cam",
			Version:     "0.1.0",
			Description: "starter",
			TargetDir:   filepath.Join(parseParentFile, "starter-app"),
		}
		_, parseErr5 := (launcher{repoRoot: parseT4.TempDir()}).generateStartScaffold(parseSelection3)
		if parseErr5 == nil || !strings.Contains(parseErr5.Error(), "create target directory") {
			parseT4.Fatalf("expected target creation error, got %v", parseErr5)
		}
	})
}

func TestRunStartHelpReturnsNil(parseT *testing.T) {
	if parseErr := (launcher{}).run([]string{"start", "-help"}); parseErr != nil {
		parseT.Fatalf("expected start help to succeed, got %v", parseErr)
	}
}

func TestRunBootstrapRoutesToStartAndExamplesOnHealthyDoctor(parseT *testing.T) {
	parseOriginalDoctorLookPath := doctorLookPath
	parseOriginalDoctorCommandOutput := doctorCommandOutput
	parseOriginalDoctorResolveWasmExec := doctorResolveWasmExec
	parseOriginalDoctorGetwd := doctorGetwd
	parseOriginalRunStartCommand := runStartCommand
	parseOriginalRunExamplesCommand := runExamplesCommand
	parseT.Cleanup(func() {
		doctorLookPath = parseOriginalDoctorLookPath
		doctorCommandOutput = parseOriginalDoctorCommandOutput
		doctorResolveWasmExec = parseOriginalDoctorResolveWasmExec
		doctorGetwd = parseOriginalDoctorGetwd
		runStartCommand = parseOriginalRunStartCommand
		runExamplesCommand = parseOriginalRunExamplesCommand
	})

	doctorLookPath = func(parseFile string) (string, error) { return filepath.Join(`C:\tools`, parseFile), nil }
	doctorCommandOutput = func(parseName string, parseArgs ...string) (string, error) { return "v1.0.0", nil }
	doctorResolveWasmExec = func() (string, error) { return filepath.Join(`C:\tools`, "wasm_exec.js"), nil }
	doctorGetwd = func() (string, error) { return parseT.TempDir(), nil }

	isParseStartCalled := false
	isParseExamplesCalled := false
	runStartCommand = func(parseL launcher, parseArgs2 []string) error {
		isParseStartCalled = true
		return nil
	}
	runExamplesCommand = func(parseL2 launcher, parseArgs3 []string) error {
		isParseExamplesCalled = true
		return nil
	}

	parseFirstPort, parseErr := reserveTCPPort()
	if parseErr != nil {
		parseT.Fatalf("reserve first bootstrap port: %v", parseErr)
	}
	if parseErr2 := (launcher{repoRoot: parseT.TempDir()}).runBootstrap([]string{"-port", parseFirstPort}); parseErr2 != nil {
		parseT.Fatalf("bootstrap start mode: %v", parseErr2)
	}
	if !isParseStartCalled {
		parseT.Fatal("expected bootstrap default mode to call start command")
	}
	if isParseExamplesCalled {
		parseT.Fatal("expected bootstrap default mode to skip examples command")
	}

	isParseStartCalled = false
	isParseExamplesCalled = false
	parseSecondPort, parseErr := reserveTCPPort()
	if parseErr != nil {
		parseT.Fatalf("reserve second bootstrap port: %v", parseErr)
	}
	if parseErr3 := (launcher{repoRoot: parseT.TempDir()}).runBootstrap([]string{"-examples", "-port", parseSecondPort}); parseErr3 != nil {
		parseT.Fatalf("bootstrap examples mode: %v", parseErr3)
	}
	if !isParseExamplesCalled {
		parseT.Fatal("expected bootstrap examples mode to call examples command")
	}
	if isParseStartCalled {
		parseT.Fatal("expected bootstrap examples mode to skip start command")
	}
}

func TestRunBootstrapBlocksWhenDoctorFails(parseT *testing.T) {
	parseOriginalDoctorLookPath := doctorLookPath
	parseOriginalDoctorGetwd := doctorGetwd
	parseOriginalRunStartCommand := runStartCommand
	parseT.Cleanup(func() {
		doctorLookPath = parseOriginalDoctorLookPath
		doctorGetwd = parseOriginalDoctorGetwd
		runStartCommand = parseOriginalRunStartCommand
	})

	doctorLookPath = func(parseFile string) (string, error) {
		return "", errors.New("missing tool")
	}
	doctorGetwd = func() (string, error) { return parseT.TempDir(), nil }

	isParseStartCalled := false
	runStartCommand = func(parseL launcher, parseArgs []string) error {
		isParseStartCalled = true
		return nil
	}

	parsePort, parseErr := reserveTCPPort()
	if parseErr != nil {
		parseT.Fatalf("reserve bootstrap port: %v", parseErr)
	}
	parseErr = (launcher{repoRoot: parseT.TempDir()}).runBootstrap([]string{"-port", parsePort})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "bootstrap blocked by doctor checks") {
		parseT.Fatalf("expected bootstrap doctor failure, got %v", parseErr)
	}
	if isParseStartCalled {
		parseT.Fatal("expected bootstrap to skip start when doctor checks fail")
	}
}

func TestValidateStartPrerequisitesChecksRuntimeAndBrowserDependencies(parseT *testing.T) {
	parseOriginalDoctorLookPath := doctorLookPath
	parseOriginalDoctorCommandOutput := doctorCommandOutput
	parseOriginalDoctorResolveWasmExec := doctorResolveWasmExec
	parseT.Cleanup(func() {
		doctorLookPath = parseOriginalDoctorLookPath
		doctorCommandOutput = parseOriginalDoctorCommandOutput
		doctorResolveWasmExec = parseOriginalDoctorResolveWasmExec
	})

	parseWasmChecks := 0
	doctorLookPath = func(parseCommand string) (string, error) {
		return filepath.Join(`C:\tools`, parseCommand), nil
	}
	doctorCommandOutput = func(parseName string, parseArgs ...string) (string, error) {
		return "v1.0.0", nil
	}
	doctorResolveWasmExec = func() (string, error) {
		parseWasmChecks++
		return filepath.Join(`C:\tools`, "wasm_exec.js"), nil
	}

	parseLauncher := launcher{repoRoot: parseT.TempDir()}
	if parseErr := parseLauncher.validateStartPrerequisites(startSelection{
		Preset:            startPreset{Features: []string{"ui"}},
		SkipRuntimeAssets: true,
	}); parseErr != nil {
		parseT.Fatalf("expected runtime-asset-skipped prerequisite validation to pass, got %v", parseErr)
	}
	if parseWasmChecks != 0 {
		parseT.Fatalf("expected runtime asset check to be skipped, got %d wasm checks", parseWasmChecks)
	}

	parseErr2 := parseLauncher.validateStartPrerequisites(startSelection{
		Preset: startPreset{Features: []string{"browser-tests"}},
	})
	if parseErr2 != nil {
		parseT.Fatalf("expected browser prerequisite validation to avoid hard JavaScript tooling dependencies, got %v", parseErr2)
	}
}

func TestGenerateStartScaffoldCanSkipOptionalPostInitSteps(parseT *testing.T) {
	parseOriginalResolveWasmExecPath := scaffoldResolveWasmExecPath
	parseOriginalScaffoldTidyModule := scaffoldTidyModule
	parseOriginalScaffoldFormatMain := scaffoldFormatMain
	parseT.Cleanup(func() {
		scaffoldResolveWasmExecPath = parseOriginalResolveWasmExecPath
		scaffoldTidyModule = parseOriginalScaffoldTidyModule
		scaffoldFormatMain = parseOriginalScaffoldFormatMain
	})

	scaffoldResolveWasmExecPath = func() (string, error) {
		return "", errors.New("wasm exec should not be resolved when runtime assets are skipped")
	}
	scaffoldTidyModule = func(parseL launcher, parseTargetDir2 string) error {
		return errors.New("tidy should not run when skip-go-mod-tidy is enabled")
	}
	scaffoldFormatMain = func(string) error { return nil }

	parseRepoRoot := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseRepoRoot, "go.mod"), []byte("module example.com/repo\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write repo go.mod: %v", parseErr)
	}

	parseTargetDir := filepath.Join(parseT.TempDir(), "starter-app")
	parseSelection := startSelection{
		Preset:            startPreset{Key: "minimal-client", Name: "Minimal Client", Features: []string{"ui", "html"}},
		ProjectName:       "starter-app",
		ModulePath:        "example.com/starter-app",
		Author:            "Cam",
		Version:           "0.1.0",
		Description:       "starter",
		TargetDir:         parseTargetDir,
		SkipGoModTidy:     true,
		SkipRuntimeAssets: true,
	}

	parseResult, parseErr2 := (launcher{repoRoot: parseRepoRoot}).generateStartScaffold(parseSelection)
	if parseErr2 != nil {
		parseT.Fatalf("expected scaffold generation with optional setup skipped to pass, got %v", parseErr2)
	}
	if parseResult.TargetDir != parseTargetDir {
		parseT.Fatalf("expected scaffold target dir %q, got %q", parseTargetDir, parseResult.TargetDir)
	}
	if _, parseErr3 := os.Stat(filepath.Join(parseTargetDir, "wasm_exec.js")); !os.IsNotExist(parseErr3) {
		parseT.Fatalf("expected wasm_exec.js to be absent when runtime assets are skipped, got err=%v", parseErr3)
	}
}

func TestGeneratedScaffoldServesOverDevServer(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("skipping dev server smoke test in short mode")
	}

	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{repoRoot: parseRepoRoot}
	parseTargetDir := filepath.Join(defaultGeneratedScaffoldRoot(), "test-dev-server-smoke")
	_ = os.RemoveAll(parseTargetDir)
	parseT.Cleanup(func() {
		_ = os.RemoveAll(parseTargetDir)
	})

	parseSelection := startSelection{
		Preset: startPreset{
			Key:         "minimal-client",
			Name:        "Minimal Client App",
			Summary:     "Test preset",
			Description: "Generated in tests.",
			Features:    []string{"ui", "html"},
		},
		ProjectName:   "test-dev-server-smoke",
		ModulePath:    "github.com/example/test-dev-server-smoke",
		Author:        "Test Author",
		Version:       "1.2.3",
		Description:   "Generated metadata test app.",
		TargetDir:     parseTargetDir,
		SkipGoModTidy: true,
	}

	parseResult, parseErr := parseLauncher.generateStartScaffold(parseSelection)
	if parseErr != nil {
		parseT.Fatalf("generate scaffold: %v", parseErr)
	}
	ensureScaffoldUsesLocalRepoModule(parseT, parseRepoRoot, parseTargetDir)

	parsePort, parseErr := reserveTCPPort()
	if parseErr != nil {
		parseT.Fatalf("reserve tcp port: %v", parseErr)
	}

	parseCtx, parseCancel := context.WithCancel(context.Background())
	defer parseCancel()
	parseCmd := exec.CommandContext(parseCtx, "go", append([]string{"run", "./tools/gwc", "dev"}, append(devArgsFromScaffold(parseResult), "-host", "127.0.0.1", "-port", parsePort)...)...)
	parseCmd.Dir = parseRepoRoot
	var parseOutput bytes.Buffer
	parseCmd.Stdout = &parseOutput
	parseCmd.Stderr = &parseOutput
	if parseErr2 := parseCmd.Start(); parseErr2 != nil {
		parseT.Fatalf("start dev server: %v", parseErr2)
	}
	parseProcessExited := make(chan struct{})
	var parseProcessErr error
	go func() {
		parseProcessErr = parseCmd.Wait()
		close(parseProcessExited)
	}()
	parseT.Cleanup(func() {
		parseCancel()
		terminateProcessTree(parseCmd)
		select {
		case <-parseProcessExited:
			if parseProcessErr != nil && !strings.Contains(parseProcessErr.Error(), "signal: killed") && !strings.Contains(parseProcessErr.Error(), "exit status 1") {
				parseT.Logf("dev server shutdown cleanup note: %v\n%s", parseProcessErr, parseOutput.String())
			}
		case <-time.After(5 * time.Second):
			if parseCmd.Process != nil {
				_ = parseCmd.Process.Kill()
			}
			select {
			case <-parseProcessExited:
				if parseProcessErr != nil && !strings.Contains(parseProcessErr.Error(), "signal: killed") {
					parseT.Logf("dev server shutdown cleanup note: %v\n%s", parseProcessErr, parseOutput.String())
				}
			case <-time.After(5 * time.Second):
				parseT.Logf("dev server shutdown cleanup note: timed out waiting for process exit\n%s", parseOutput.String())
			}
		}
	})

	parseRootURL := "http://127.0.0.1:" + parsePort + "/"
	parseWasmURL := "http://127.0.0.1:" + parsePort + "/" + scaffoldWASMOutputPath()

	parseRootBody := waitForHTTPBodyWithProcess(parseT, parseRootURL, 180*time.Second, parseProcessExited, &parseProcessErr, &parseOutput, func(parseResp *http.Response, parseBody string) error {
		if parseResp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status %d", parseResp.StatusCode)
		}
		if !strings.Contains(parseBody, parseSelection.ProjectName) {
			return fmt.Errorf("html shell missing project name")
		}
		if !strings.Contains(parseBody, "__GWC_LIVERELOAD_CONFIG") {
			return fmt.Errorf("html shell missing livereload config injection")
		}
		return nil
	})
	if !strings.Contains(parseRootBody, "wasm_exec.js") {
		parseT.Fatalf("expected served HTML to include wasm runtime bootstrap, got:\n%s", parseRootBody)
	}

	waitForHTTPBodyWithProcess(parseT, parseWasmURL, 180*time.Second, parseProcessExited, &parseProcessErr, &parseOutput, func(parseResp2 *http.Response, parseBody2 string) error {
		if parseResp2.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status %d", parseResp2.StatusCode)
		}
		if parseContentType := parseResp2.Header.Get("Content-Type"); !strings.Contains(parseContentType, "application/wasm") {
			return fmt.Errorf("unexpected content type %q", parseContentType)
		}
		if len(parseBody2) == 0 {
			return fmt.Errorf("empty wasm response")
		}
		return nil
	})
}

func TestGeneratedScaffoldLauncherBuildTestVerifyReleaseRoundTrip(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("skipping launcher round-trip test in short mode")
	}

	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{repoRoot: parseRepoRoot}
	parseTargetDir := filepath.Join(parseT.TempDir(), "test-launcher-roundtrip")

	parseSelection := startSelection{
		Preset: startPreset{
			Key:         "minimal-client",
			Name:        "Minimal Client App",
			Summary:     "Test preset",
			Description: "Generated in tests.",
			Features:    []string{"ui", "html"},
		},
		ProjectName:   "test-launcher-roundtrip",
		ModulePath:    "github.com/example/test-launcher-roundtrip",
		Author:        "Test Author",
		Version:       "1.2.3",
		Description:   "Generated launcher round-trip test app.",
		TargetDir:     parseTargetDir,
		SkipGoModTidy: true,
	}

	parseResult, parseErr := parseLauncher.generateStartScaffold(parseSelection)
	if parseErr != nil {
		parseT.Fatalf("generate scaffold: %v", parseErr)
	}
	ensureScaffoldUsesLocalRepoModule(parseT, parseRepoRoot, parseTargetDir)

	buildOut := filepath.Join(parseTargetDir, "bin", "ci", "app.wasm")
	var build buildSummary
	runLauncherJSONCommand(parseT, parseRepoRoot, &build, "build", "-app", parseResult.AppPath, "-root", parseResult.TargetDir, "-out", buildOut, "-profile", "ci", "-json")
	if !build.OK {
		parseT.Fatalf("expected build summary to succeed, got %#v", build)
	}
	if build.OutputPath != buildOut {
		parseT.Fatalf("expected build output path %q, got %#v", buildOut, build)
	}
	if _, parseErr2 := os.Stat(buildOut); parseErr2 != nil {
		parseT.Fatalf("expected build artifact %q: %v", buildOut, parseErr2)
	}

	var parseTests testSummary
	runLauncherJSONCommand(parseT, parseRepoRoot, &parseTests, "test", "-app", parseResult.AppPath, "-root", parseResult.TargetDir, "-lane", "unit", "-json")
	if !parseTests.OK {
		parseT.Fatalf("expected gwc test summary to succeed, got %#v", parseTests)
	}
	if len(parseTests.Lanes) != 1 || parseTests.Lanes[0].Name != "unit" || parseTests.Lanes[0].OK != true {
		parseT.Fatalf("expected unit test lane success, got %#v", parseTests)
	}

	var parseVerify verifySummary
	runLauncherJSONCommand(parseT, parseRepoRoot, &parseVerify, "verify", "-app", parseResult.AppPath, "-root", parseResult.TargetDir, "-json")
	if !parseVerify.OK {
		parseT.Fatalf("expected verify summary to succeed, got %#v", parseVerify)
	}
	if !parseVerify.Tests.Ran || parseVerify.Build.OK != true {
		parseT.Fatalf("expected verify to run tests and build successfully, got %#v", parseVerify)
	}
	if _, parseErr3 := os.Stat(parseVerify.Build.OutputPath); parseErr3 != nil {
		parseT.Fatalf("expected verify build artifact %q: %v", parseVerify.Build.OutputPath, parseErr3)
	}

	parseReleaseOut := filepath.Join(parseTargetDir, "bin", "release-e2e")
	var parseRelease releaseSummary
	runLauncherJSONCommand(parseT, parseRepoRoot, &parseRelease, "release", "-app", parseResult.AppPath, "-root", parseResult.TargetDir, "-out-dir", parseReleaseOut, "-binary-name", "app.wasm", "-compression", "none", "-json")
	if !parseRelease.OK {
		parseT.Fatalf("expected release summary to succeed, got %#v", parseRelease)
	}
	if parseRelease.OutDir != parseReleaseOut {
		parseT.Fatalf("expected release output dir %q, got %#v", parseReleaseOut, parseRelease)
	}
	parseWasmArtifact, parseOk := parseRelease.Artifacts["wasm"]
	if !parseOk || parseWasmArtifact.Bytes <= 0 {
		parseT.Fatalf("expected wasm release artifact, got %#v", parseRelease)
	}
	if _, parseErr4 := os.Stat(filepath.Join(parseReleaseOut, "app.wasm")); parseErr4 != nil {
		parseT.Fatalf("expected released wasm artifact: %v", parseErr4)
	}
	if _, parseErr5 := os.Stat(filepath.Join(parseReleaseOut, "wasm-release-manifest.json")); parseErr5 != nil {
		parseT.Fatalf("expected release manifest: %v", parseErr5)
	}
}

func newTestStartModel() startModel {
	parseModel := startModel{
		presets: defaultStartPresets(),
		inputs:  newStartInputs(),
	}
	parseModel.selection.Preset = parseModel.presets[0]
	return parseModel
}

func ensureScaffoldUsesLocalRepoModule(parseT *testing.T, parseRepoRoot string, parseTargetDir string) {
	parseT.Helper()
	parseModulePath, parseErr := (launcher{repoRoot: parseRepoRoot}).readRepoModulePath()
	if parseErr != nil {
		parseT.Fatalf("read repo module path: %v", parseErr)
	}

	parseGoModPath := filepath.Join(parseTargetDir, "go.mod")
	parseGoModBytes, parseErr := os.ReadFile(parseGoModPath)
	if parseErr != nil {
		parseT.Fatalf("read scaffold go.mod: %v", parseErr)
	}
	parseGoModText := strings.TrimSpace(string(parseGoModBytes)) + "\n\n" +
		"require " + parseModulePath + " v0.0.0\n\n" +
		"replace " + parseModulePath + " => " + filepath.ToSlash(parseRepoRoot) + "\n"
	if parseErr2 := os.WriteFile(parseGoModPath, []byte(parseGoModText), 0644); parseErr2 != nil {
		parseT.Fatalf("write scaffold go.mod: %v", parseErr2)
	}

	parseTidyCmd := exec.Command("go", "mod", "tidy")
	parseTidyCmd.Dir = parseTargetDir
	parseTidyCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	parseTidyOutput, parseErr := parseTidyCmd.CombinedOutput()
	if parseErr != nil {
		parseT.Fatalf("tidy scaffold go.mod: %v\n%s", parseErr, string(parseTidyOutput))
	}
}

func runLauncherJSONCommand(parseT *testing.T, parseRepoRoot string, parseTarget any, parseArgs ...string) {
	parseT.Helper()

	parseCmd := exec.Command("go", append([]string{"run", "./tools/gwc"}, parseArgs...)...)
	parseCmd.Dir = parseRepoRoot
	parseOutput, parseErr := parseCmd.CombinedOutput()
	if parseErr != nil {
		parseT.Fatalf("run launcher command %q: %v\n%s", strings.Join(parseArgs, " "), parseErr, string(parseOutput))
	}
	parsePayload := bytes.TrimSpace(parseOutput)
	if len(parsePayload) > 0 && parsePayload[0] != '{' {
		parseStart := bytes.IndexByte(parsePayload, '{')
		if parseStart >= 0 {
			parsePayload = parsePayload[parseStart:]
		}
	}
	if parseErr2 := json.Unmarshal(parsePayload, parseTarget); parseErr2 != nil {
		parseT.Fatalf("unmarshal launcher JSON for %q: %v\n%s", strings.Join(parseArgs, " "), parseErr2, string(parseOutput))
	}
}

func captureStdout() (func() (string, error), func(), error) {
	parseOriginalStdout := os.Stdout
	parseReader, parseWriter, parseErr := os.Pipe()
	if parseErr != nil {
		return nil, nil, parseErr
	}
	os.Stdout = parseWriter

	parseReadOutput := func() (string, error) {
		if parseErr2 := parseWriter.Close(); parseErr2 != nil {
			return "", parseErr2
		}
		parseBytes, parseErr3 := io.ReadAll(parseReader)
		if parseErr3 != nil {
			return "", parseErr3
		}
		return string(parseBytes), nil
	}
	parseRestore := func() {
		os.Stdout = parseOriginalStdout
		_ = parseWriter.Close()
		_ = parseReader.Close()
	}
	return parseReadOutput, parseRestore, nil
}

func reserveTCPPort() (string, error) {
	parseListener, parseErr := net.Listen("tcp", "127.0.0.1:0")
	if parseErr != nil {
		return "", parseErr
	}
	defer parseListener.Close()
	_, parsePort, parseErr := net.SplitHostPort(parseListener.Addr().String())
	if parseErr != nil {
		return "", parseErr
	}
	return parsePort, nil
}

func waitForHTTPBodyWithProcess(parseT *testing.T, parseUrl string, parseTimeout time.Duration, parseProcessExited <-chan struct{}, parseProcessErr *error, parseOutput *bytes.Buffer, parseValidate func(resp *http.Response, body string) error) string {
	parseT.Helper()

	parseClient := &http.Client{Timeout: 5 * time.Second}
	parseDeadline := time.Now().Add(parseTimeout)
	var parseLastErr error
	for time.Now().Before(parseDeadline) {
		select {
		case <-parseProcessExited:
			parseT.Fatalf("dev server exited before %s became ready: %v\n%s", parseUrl, *parseProcessErr, parseOutput.String())
		default:
		}

		parseResp, parseErr := parseClient.Get(parseUrl)
		if parseErr != nil {
			parseLastErr = parseErr
			time.Sleep(500 * time.Millisecond)
			continue
		}
		parseBodyBytes, parseReadErr := io.ReadAll(parseResp.Body)
		parseResp.Body.Close()
		if parseReadErr != nil {
			parseLastErr = parseReadErr
			time.Sleep(500 * time.Millisecond)
			continue
		}
		parseBody := string(parseBodyBytes)
		if parseErr2 := parseValidate(parseResp, parseBody); parseErr2 != nil {
			parseLastErr = parseErr2
			time.Sleep(500 * time.Millisecond)
			continue
		}
		return parseBody
	}
	parseT.Fatalf("timed out waiting for %s: %v\n%s", parseUrl, parseLastErr, parseOutput.String())
	return ""
}

func terminateProcessTree(parseCmd *exec.Cmd) {
	if parseCmd == nil || parseCmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", parseCmd.Process.Pid)).Run()
		return
	}
	_ = parseCmd.Process.Kill()
}

func sequentialWordSelector(parseIndices ...int) func(int) int {
	parsePosition := 0
	return func(parseLimit int) int {
		if len(parseIndices) == 0 {
			return 0
		}
		if parsePosition >= len(parseIndices) {
			return parseIndices[len(parseIndices)-1]
		}
		parseSelected := parseIndices[parsePosition]
		parsePosition++
		return parseSelected
	}
}
