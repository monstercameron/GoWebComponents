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
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDefaultTargetDirUsesGeneratedRoot(t *testing.T) {
	projectName := "starter-app"
	targetDir := defaultTargetDir(projectName)
	generatedRoot := defaultGeneratedScaffoldRoot()

	if !strings.HasPrefix(filepath.Clean(targetDir), filepath.Clean(generatedRoot)) {
		t.Fatalf("expected target dir %q to live under generated root %q", targetDir, generatedRoot)
	}
	if filepath.Base(targetDir) != projectName {
		t.Fatalf("expected target dir base %q to match project name %q", filepath.Base(targetDir), projectName)
	}
}

func TestDefaultGeneratedScaffoldRootForWindowsUsesDocuments(t *testing.T) {
	homeDir := filepath.Join("C:\\Users", "Cam")
	got := defaultGeneratedScaffoldRootForOS("windows", homeDir, nil)
	want := filepath.Join(homeDir, "Documents")
	if got != want {
		t.Fatalf("expected windows scaffold root %q, got %q", want, got)
	}
}

func TestDefaultGeneratedScaffoldRootForDarwinUsesDocuments(t *testing.T) {
	homeDir := filepath.Join("/Users", "cam")
	got := defaultGeneratedScaffoldRootForOS("darwin", homeDir, nil)
	want := filepath.Join(homeDir, "Documents")
	if got != want {
		t.Fatalf("expected darwin scaffold root %q, got %q", want, got)
	}
}

func TestDefaultGeneratedScaffoldRootForLinuxFallsBackFromDocuments(t *testing.T) {
	homeDir := filepath.Join("/home", "cam")
	documentsDir := filepath.Join(homeDir, "Documents")
	projectsDir := filepath.Join(homeDir, "Projects")

	got := defaultGeneratedScaffoldRootForOS("linux", homeDir, func(path string) bool {
		return path == documentsDir
	})
	if got != documentsDir {
		t.Fatalf("expected linux scaffold root to prefer Documents, got %q", got)
	}

	got = defaultGeneratedScaffoldRootForOS("linux", homeDir, func(path string) bool {
		return path == projectsDir
	})
	if got != projectsDir {
		t.Fatalf("expected linux scaffold root to fall back to Projects, got %q", got)
	}

	got = defaultGeneratedScaffoldRootForOS("linux", homeDir, func(path string) bool { return false })
	if got != homeDir {
		t.Fatalf("expected linux scaffold root to fall back to home dir, got %q", got)
	}
}

func TestValidateProjectFolderNameRejectsInvalidCharacters(t *testing.T) {
	if err := validateProjectFolderName(`bad:name`); err == nil {
		t.Fatal("expected invalid project folder name to be rejected")
	}
}

func TestValidateProjectFolderNameRejectsDotNames(t *testing.T) {
	for _, projectName := range []string{".", ".."} {
		if err := validateProjectFolderName(projectName); err == nil {
			t.Fatalf("expected project name %q to be rejected", projectName)
		}
	}
}

func TestDefaultDescriptionUsesPresetSummary(t *testing.T) {
	preset := startPreset{Summary: "Small scaffold summary"}
	if got := defaultDescription(preset); got != preset.Summary {
		t.Fatalf("expected default description %q, got %q", preset.Summary, got)
	}
}

func TestProjectNameWithWordSelectorPrefixesPresetSlug(t *testing.T) {
	preset := startPreset{Key: "minimal-client"}
	selector := sequentialWordSelector(2, 5)
	got := projectNameWithWordSelector(preset, selector)
	want := techProjectWords[2] + "-" + salesProjectWords[5] + "-minimal-client"
	if got != want {
		t.Fatalf("expected generated project name %q, got %q", want, got)
	}
}

func TestProjectNameWithWordSelectorFallsBackToAppBase(t *testing.T) {
	got := projectNameWithWordSelector(startPreset{}, sequentialWordSelector(0, 0))
	want := techProjectWords[0] + "-" + salesProjectWords[0] + "-app"
	if got != want {
		t.Fatalf("expected fallback generated project name %q, got %q", want, got)
	}
}

func TestValidateStartTerminalRequiresInteractiveTTY(t *testing.T) {
	tests := []struct {
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateStartTerminal(test.stdinIsTerminal, test.stdoutIsTerminal)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected non-interactive terminal validation to fail")
				}
				if !strings.Contains(err.Error(), "interactive terminal") {
					t.Fatalf("expected actionable interactive terminal error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected interactive terminal validation to pass, got %v", err)
			}
		})
	}
}

func TestStartPostModelSelectedChoiceDefaultsToExit(t *testing.T) {
	model := startPostModel{}
	if choice := model.selectedChoice(); choice != startPostChoiceExit {
		t.Fatalf("expected default post-start choice to be exit, got %v", choice)
	}

	model.options = []startPostChoice{startPostChoiceExit, startPostChoiceRunDev}
	model.cursor = 1
	if choice := model.selectedChoice(); choice != startPostChoiceRunDev {
		t.Fatalf("expected second post-start choice to be run-dev, got %v", choice)
	}
}

func TestStartModelInitAndViewRouting(t *testing.T) {
	model := newTestStartModel()
	if cmd := model.Init(); cmd == nil {
		t.Fatal("expected start model init command")
	}
	if view := model.View(); !strings.Contains(view, "Step 1 of 3") || !strings.Contains(view, model.presets[0].Name) {
		t.Fatalf("expected preset picker view, got %q", view)
	}

	model.step = startStepProject
	model.selection.Preset = model.presets[0]
	if view := model.View(); !strings.Contains(view, "Step 2 of 3") || !strings.Contains(view, "Project name") {
		t.Fatalf("expected project form view, got %q", view)
	}

	model.step = startStepConfirm
	if view := model.View(); !strings.Contains(view, "Step 3 of 3") || !strings.Contains(view, "Included baseline features") {
		t.Fatalf("expected confirmation view, got %q", view)
	}

	model.quitting = true
	if view := model.View(); view != "\n" {
		t.Fatalf("expected quitting view newline, got %q", view)
	}
}

func TestStartModelUpdateHandlesWindowAndQuit(t *testing.T) {
	model := newTestStartModel()
	updatedModel, cmd := model.Update(tea.WindowSizeMsg{Width: 123})
	updated := updatedModel.(startModel)
	if updated.width != 123 {
		t.Fatalf("expected width 123, got %d", updated.width)
	}
	if cmd != nil {
		t.Fatalf("expected no command for window resize, got %#v", cmd)
	}

	updatedModel, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated = updatedModel.(startModel)
	if !updated.quitting {
		t.Fatal("expected q to mark model quitting")
	}
	if cmd == nil {
		t.Fatal("expected quit command for q")
	}
}

func TestStartModelUpdateDispatchAndFallbackBranches(t *testing.T) {
	model := newTestStartModel()
	model.step = startStepProject
	model.selection.Preset = model.presets[0]
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated := updatedModel.(startModel)
	if updated.inputIndex != 1 {
		t.Fatalf("expected project-step dispatch to advance input focus, got %d", updated.inputIndex)
	}

	model = newTestStartModel()
	model.step = startStepConfirm
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(startModel)
	if !updated.confirmed || cmd == nil {
		t.Fatal("expected confirm-step dispatch to confirm and quit")
	}

	model = newTestStartModel()
	model.step = startStep(99)
	updatedModel, cmd = model.Update(tea.MouseMsg{})
	updated = updatedModel.(startModel)
	if updated.step != startStep(99) || cmd != nil {
		t.Fatalf("expected unknown step fallback to return unchanged model and nil cmd, got step=%v cmd=%#v", updated.step, cmd)
	}

	model = newTestStartModel()
	updatedModel, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	updated = updatedModel.(startModel)
	if !updated.quitting || cmd == nil {
		t.Fatal("expected ctrl+c to quit start model")
	}
}

func TestUpdatePresetStepNavigationAndEnter(t *testing.T) {
	model := newTestStartModel()
	model.cursor = 1
	updatedModel, _ := model.updatePresetStep(tea.KeyMsg{Type: tea.KeyUp})
	updated := updatedModel.(startModel)
	if updated.cursor != 0 {
		t.Fatalf("expected cursor to move up to 0, got %d", updated.cursor)
	}

	updatedModel, _ = updated.updatePresetStep(tea.KeyMsg{Type: tea.KeyDown})
	updated = updatedModel.(startModel)
	if updated.cursor != 1 {
		t.Fatalf("expected cursor to move down to 1, got %d", updated.cursor)
	}

	t.Setenv("GIT_AUTHOR_NAME", "Preset Author")

	updated.errText = "stale"
	updatedModel, cmd := updated.updatePresetStep(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(startModel)
	if updated.step != startStepProject {
		t.Fatalf("expected enter to advance to project step, got %v", updated.step)
	}
	if updated.selection.Preset.Key != updated.presets[updated.cursor].Key {
		t.Fatalf("expected selected preset to be stored, got %#v", updated.selection.Preset)
	}
	if !strings.HasSuffix(updated.inputs[0].Value(), "-"+updated.selection.Preset.Key) {
		t.Fatalf("expected seeded project name to include preset key, got %q", updated.inputs[0].Value())
	}
	if updated.inputs[1].Value() != "github.com/your-org/"+updated.inputs[0].Value() {
		t.Fatalf("expected seeded module path to derive from project name, got %q", updated.inputs[1].Value())
	}
	if updated.inputs[2].Value() != "Preset Author" || updated.inputs[3].Value() != "0.1.0" {
		t.Fatalf("expected seeded author/version defaults, got author=%q version=%q", updated.inputs[2].Value(), updated.inputs[3].Value())
	}
	if updated.inputs[4].Value() != updated.selection.Preset.Summary {
		t.Fatalf("expected seeded description from preset summary, got %q", updated.inputs[4].Value())
	}
	if updated.errText != "" {
		t.Fatalf("expected errText to clear, got %q", updated.errText)
	}
	if cmd == nil {
		t.Fatal("expected blink command after preset selection")
	}
}

func TestUpdateProjectStepNavigationAndSubmission(t *testing.T) {
	model := newTestStartModel()
	model.step = startStepProject
	model.selection.Preset = model.presets[0]
	model.focusInput(0)

	updatedModel, _ := model.updateProjectStep(tea.KeyMsg{Type: tea.KeyTab})
	updated := updatedModel.(startModel)
	if updated.inputIndex != 1 {
		t.Fatalf("expected tab to advance focus to 1, got %d", updated.inputIndex)
	}

	updatedModel, _ = updated.updateProjectStep(tea.KeyMsg{Type: tea.KeyShiftTab})
	updated = updatedModel.(startModel)
	if updated.inputIndex != 0 {
		t.Fatalf("expected shift+tab to move focus back to 0, got %d", updated.inputIndex)
	}

	updated.errText = "problem"
	updatedModel, _ = updated.updateProjectStep(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(startModel)
	if updated.step != startStepPreset || updated.errText != "" {
		t.Fatalf("expected esc to return to preset step and clear error, got step=%v err=%q", updated.step, updated.errText)
	}

	model = newTestStartModel()
	model.step = startStepProject
	model.selection.Preset = model.presets[0]
	model.inputs[0].SetValue("app")
	model.inputs[1].SetValue("example.com/app")
	model.inputs[2].SetValue("Cam")
	model.inputs[3].SetValue("1.0.0")
	model.inputs[4].SetValue("Desc")
	model.focusInput(4)
	updatedModel, _ = model.updateProjectStep(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(startModel)
	if updated.step != startStepConfirm {
		t.Fatalf("expected enter on final input to advance to confirm, got %v", updated.step)
	}
	if updated.selection.ProjectName != "app" {
		t.Fatalf("expected built selection to be stored, got %#v", updated.selection)
	}

	model = newTestStartModel()
	model.step = startStepProject
	model.selection.Preset = model.presets[0]
	model.inputs[0].SetValue("")
	model.focusInput(4)
	updatedModel, _ = model.updateProjectStep(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(startModel)
	if !strings.Contains(updated.errText, "project name is required") {
		t.Fatalf("expected validation error, got %q", updated.errText)
	}
}

func TestUpdateConfirmStepAndViews(t *testing.T) {
	model := newTestStartModel()
	model.step = startStepConfirm
	model.selection.Preset = model.presets[0]
	model.errText = "stale"
	updatedModel, _ := model.updateConfirmStep(tea.KeyMsg{Type: tea.KeyEsc})
	updated := updatedModel.(startModel)
	if updated.step != startStepProject || updated.errText != "" {
		t.Fatalf("expected esc to return to project step and clear error, got step=%v err=%q", updated.step, updated.errText)
	}

	updatedModel, cmd := model.updateConfirmStep(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(startModel)
	if !updated.confirmed {
		t.Fatal("expected enter to confirm start model")
	}
	if cmd == nil {
		t.Fatal("expected quit command on confirm enter")
	}
}

func TestStartPostModelUpdateAndView(t *testing.T) {
	model := startPostModel{
		selection: startSelection{Preset: startPreset{Name: "Minimal"}, ProjectName: "app", ModulePath: "example.com/app", Author: "Cam", Version: "1.0.0"},
		result:    &scaffoldResult{TargetDir: `C:\tmp\app`, AppPath: `C:\tmp\app\main.go`, HTMLPath: `C:\tmp\app\index.html`},
		options:   []startPostChoice{startPostChoiceExit, startPostChoiceRunDev},
	}
	if cmd := model.Init(); cmd != nil {
		t.Fatalf("expected nil init command, got %#v", cmd)
	}
	if view := model.View(); !strings.Contains(view, "Scaffold generated successfully.") || !strings.Contains(view, "Run the generated scaffold in the dev server now") {
		t.Fatalf("expected success view, got %q", view)
	}
	if got := model.optionLabel(startPostChoiceExit); !strings.Contains(got, "Exit") {
		t.Fatalf("expected exit label, got %q", got)
	}
	if got := model.optionLabel(startPostChoiceRunDev); !strings.Contains(got, "dev server") {
		t.Fatalf("expected run-dev label, got %q", got)
	}

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := updatedModel.(startPostModel)
	if updated.cursor != 1 {
		t.Fatalf("expected down to move cursor to 1, got %d", updated.cursor)
	}
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated = updatedModel.(startPostModel)
	if updated.cursor != 0 {
		t.Fatalf("expected up to move cursor to 0, got %d", updated.cursor)
	}
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(startPostModel)
	if !updated.confirmed || cmd == nil {
		t.Fatal("expected enter to confirm and quit post model")
	}

	model.errText = "failed"
	if view := model.View(); !strings.Contains(view, "Scaffold generation failed.") {
		t.Fatalf("expected error view, got %q", view)
	}
	updatedModel, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(startPostModel)
	if !updated.confirmed || cmd == nil {
		t.Fatal("expected esc to confirm-close post model")
	}

	model.quitting = true
	if view := model.View(); view != "\n" {
		t.Fatalf("expected quitting newline view, got %q", view)
	}
}

func TestStartPostModelUpdateAdditionalBranches(t *testing.T) {
	model := startPostModel{options: []startPostChoice{startPostChoiceExit, startPostChoiceRunDev}, cursor: 1}
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	updated := updatedModel.(startPostModel)
	if updated.cursor != 0 {
		t.Fatalf("expected k to move cursor up, got %d", updated.cursor)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated = updatedModel.(startPostModel)
	if updated.cursor != 1 {
		t.Fatalf("expected j to move cursor down, got %d", updated.cursor)
	}

	updated.errText = "failed"
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated = updatedModel.(startPostModel)
	if updated.cursor != 1 {
		t.Fatalf("expected errText branch to suppress movement, got %d", updated.cursor)
	}

	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	updated = updatedModel.(startPostModel)
	if !updated.quitting || cmd == nil {
		t.Fatal("expected ctrl+c to quit post model")
	}

	updatedModel, cmd = startPostModel{}.Update(tea.MouseMsg{})
	updated = updatedModel.(startPostModel)
	if updated.cursor != 0 || cmd != nil {
		t.Fatalf("expected non-key post update to no-op, got %#v cmd=%#v", updated, cmd)
	}
}

func TestStartHelperDefaultsAndUtilities(t *testing.T) {
	if isInteractiveFile(nil) {
		t.Fatal("expected nil file to be non-interactive")
	}
	if got := defaultModulePath(""); got != "github.com/your-org/my-app" {
		t.Fatalf("expected default module path fallback, got %q", got)
	}
	t.Setenv("GIT_AUTHOR_NAME", "Git Author")
	if got := defaultAuthor(); got != "Git Author" {
		t.Fatalf("expected env-based author, got %q", got)
	}
	if got := defaultVersion(); got != "0.1.0" {
		t.Fatalf("expected default version, got %q", got)
	}
	if got := defaultProjectName(startPreset{Key: "minimal-client"}); !strings.HasSuffix(got, "-minimal-client") {
		t.Fatalf("expected random project name to use preset key, got %q", got)
	}
	selector := newRandomWordSelector()
	if selector(0) != 0 {
		t.Fatal("expected random selector with zero limit to return 0")
	}
	if got := pickProjectWord(nil, nil); got != "app" {
		t.Fatalf("expected empty word list fallback, got %q", got)
	}
	if got := pickProjectWord([]string{"alpha", "beta"}, func(limit int) int { return -1 }); got != "beta" {
		t.Fatalf("expected negative index normalization, got %q", got)
	}

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer reader.Close()
	defer writer.Close()
	if isInteractiveFile(reader) {
		t.Fatal("expected pipe file to be non-interactive")
	}

	model := newTestStartModel()
	model.selection.Preset = model.presets[0]
	model.inputs[0].SetValue("project")
	model.inputs[1].SetValue("example.com/project")
	model.inputs[2].SetValue("Cam")
	model.inputs[3].SetValue("1.0.0")
	model.inputs[4].SetValue("Desc")
	model.focusInput(-1)
	if model.inputIndex != len(model.inputs)-1 {
		t.Fatalf("expected negative focus index to wrap, got %d", model.inputIndex)
	}
	model.focusInput(len(model.inputs))
	if model.inputIndex != 0 {
		t.Fatalf("expected overflow focus index to wrap to 0, got %d", model.inputIndex)
	}
	if view := model.renderPresetPicker(); !strings.Contains(view, "Keys: up/down") {
		t.Fatalf("expected preset picker help text, got %q", view)
	}
	model.errText = "bad input"
	if view := model.renderProjectForm(); !strings.Contains(view, "Error: bad input") {
		t.Fatalf("expected project form error text, got %q", view)
	}
	if view := model.renderConfirmation(); !strings.Contains(view, "user-owned project outside the framework repo") {
		t.Fatalf("expected confirmation copy, got %q", view)
	}
}

func TestStartHelperAdditionalFallbacks(t *testing.T) {
	for _, key := range []string{"GIT_AUTHOR_NAME", "GITHUB_USER", "USERNAME", "USER"} {
		t.Setenv(key, "")
	}
	if got := defaultAuthor(); got != "Your Name" {
		t.Fatalf("expected default author fallback, got %q", got)
	}
	if got := defaultDescription(startPreset{}); got != "Starter app generated by gwc start" {
		t.Fatalf("expected default description fallback, got %q", got)
	}
	if got := pickProjectWord([]string{"alpha", "beta"}, nil); got != "alpha" {
		t.Fatalf("expected nil selector to pick first word, got %q", got)
	}
	if got := (startPostModel{options: []startPostChoice{startPostChoiceExit}, cursor: 9}).selectedChoice(); got != startPostChoiceExit {
		t.Fatalf("expected out-of-range selected choice to fall back to exit, got %v", got)
	}
	if got := (startPostModel{options: []startPostChoice{startPostChoiceExit}, cursor: -1}).selectedChoice(); got != startPostChoiceExit {
		t.Fatalf("expected negative selected choice to fall back to exit, got %v", got)
	}
	if got := defaultGeneratedScaffoldRootForOS("plan9", "", nil); got != filepath.Join("gwc-projects") {
		t.Fatalf("expected empty-home scaffold root fallback, got %q", got)
	}
}

func TestDefaultGeneratedScaffoldRootUsesFallbackSources(t *testing.T) {
	originalHomeDir := startUserHomeDir
	t.Cleanup(func() {
		startUserHomeDir = originalHomeDir
	})

	t.Run("cwd fallback", func(t *testing.T) {
		root := t.TempDir()
		originalWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("get working dir: %v", err)
		}
		if err := os.Chdir(root); err != nil {
			t.Fatalf("chdir temp root: %v", err)
		}
		defer func() {
			_ = os.Chdir(originalWD)
		}()
		startUserHomeDir = func() (string, error) { return "", errors.New("no home") }
		got := defaultGeneratedScaffoldRoot()
		want := filepath.Join(root, "gwc-projects")
		if got != want {
			t.Fatalf("expected cwd fallback %q, got %q", want, got)
		}
	})
}

func TestLoadScaffoldMetadataRejectsInvalidJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "gwc-start.json"), []byte(`{"tooling":`), 0644); err != nil {
		t.Fatalf("write invalid metadata: %v", err)
	}
	_, ok, err := loadScaffoldMetadata(root)
	if ok || err == nil || !strings.Contains(err.Error(), "parse scaffold metadata") {
		t.Fatalf("expected invalid metadata parse error, got ok=%t err=%v", ok, err)
	}
}

func TestLoadScaffoldMetadataMissingPathReturnsZeroValues(t *testing.T) {
	root := t.TempDir()
	metadata, ok, err := loadScaffoldMetadata(root)
	if err != nil || ok {
		t.Fatalf("expected missing metadata to return zero values, got metadata=%#v ok=%t err=%v", metadata, ok, err)
	}
	if metadata.ProjectName != "" || metadata.Tooling.AppPath != "" {
		t.Fatalf("expected missing metadata to leave zero-value fields, got %#v", metadata)
	}
}

func TestNormalizePathBlankInputReturnsEmpty(t *testing.T) {
	path, err := normalizePath(t.TempDir(), "   ")
	if err != nil {
		t.Fatalf("normalize blank path: %v", err)
	}
	if path != "" {
		t.Fatalf("expected blank normalized path, got %q", path)
	}
}

func TestStartProjectStepAdditionalBranches(t *testing.T) {
	model := newTestStartModel()
	model.selection.Preset = model.presets[0]
	model.step = startStepProject
	model.inputIndex = 1

	updatedModel, cmd := model.updateProjectStep(tea.KeyMsg{Type: tea.KeyEsc})
	updated := updatedModel.(startModel)
	if updated.step != startStepPreset || cmd != nil {
		t.Fatalf("expected esc to return to preset step, got %#v cmd=%#v", updated, cmd)
	}

	model = newTestStartModel()
	model.selection.Preset = model.presets[0]
	model.step = startStepProject
	model.inputIndex = 1
	updatedModel, _ = model.updateProjectStep(tea.KeyMsg{Type: tea.KeyShiftTab})
	updated = updatedModel.(startModel)
	if updated.inputIndex != 0 {
		t.Fatalf("expected shift+tab to move focus backward, got %d", updated.inputIndex)
	}

	updatedModel, _ = updated.updateProjectStep(tea.KeyMsg{Type: tea.KeyTab})
	updated = updatedModel.(startModel)
	if updated.inputIndex != 1 {
		t.Fatalf("expected tab to move focus forward, got %d", updated.inputIndex)
	}

	updatedModel, _ = updated.updateProjectStep(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(startModel)
	if updated.inputIndex != 2 || updated.step != startStepProject {
		t.Fatalf("expected enter before last input to advance focus, got %#v", updated)
	}

	model = newTestStartModel()
	model.selection.Preset = model.presets[0]
	model.step = startStepProject
	model.inputIndex = len(model.inputs) - 1
	updatedModel, _ = model.updateProjectStep(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(startModel)
	if updated.errText == "" || updated.step != startStepProject {
		t.Fatalf("expected invalid selection to keep project step with error, got %#v", updated)
	}

	model = newTestStartModel()
	model.selection.Preset = model.presets[0]
	model.step = startStepProject
	model.inputIndex = len(model.inputs) - 1
	model.inputs[0].SetValue("starter-app")
	model.inputs[1].SetValue("example.com/starter-app")
	model.inputs[2].SetValue("Cam")
	model.inputs[3].SetValue("1.0.0")
	model.inputs[4].SetValue("Starter app")
	updatedModel, _ = model.updateProjectStep(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(startModel)
	if updated.step != startStepConfirm || updated.selection.ProjectName != "starter-app" {
		t.Fatalf("expected valid selection to advance to confirmation, got %#v", updated)
	}
}

func TestUpdateProjectStepCoversArrowKeysAndInputUpdate(t *testing.T) {
	model := newTestStartModel()
	model.selection.Preset = model.presets[0]
	model.step = startStepProject
	model.focusInput(0)

	updatedModel, _ := model.updateProjectStep(tea.KeyMsg{Type: tea.KeyUp})
	updated := updatedModel.(startModel)
	if updated.inputIndex != len(updated.inputs)-1 {
		t.Fatalf("expected up arrow to wrap focus to last input, got %d", updated.inputIndex)
	}

	updatedModel, _ = updated.updateProjectStep(tea.KeyMsg{Type: tea.KeyDown})
	updated = updatedModel.(startModel)
	if updated.inputIndex != 0 {
		t.Fatalf("expected down arrow to wrap focus to first input, got %d", updated.inputIndex)
	}

	updatedModel, _ = updated.updateProjectStep(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	updated = updatedModel.(startModel)
	if updated.inputs[0].Value() != "x" {
		t.Fatalf("expected rune input to update focused field, got %q", updated.inputs[0].Value())
	}
}

func TestUpdateConfirmStepBackspaceAndNoop(t *testing.T) {
	model := newTestStartModel()
	model.step = startStepConfirm
	model.selection.Preset = model.presets[0]
	model.errText = "stale"

	updatedModel, _ := model.updateConfirmStep(tea.KeyMsg{Type: tea.KeyBackspace})
	updated := updatedModel.(startModel)
	if updated.step != startStepProject || updated.errText != "" {
		t.Fatalf("expected backspace to return to project step and clear error, got step=%v err=%q", updated.step, updated.errText)
	}

	updatedModel, cmd := updated.updateConfirmStep(tea.MouseMsg{})
	updated = updatedModel.(startModel)
	if updated.step != startStepProject || cmd != nil {
		t.Fatalf("expected non-key confirm update to no-op, got step=%v cmd=%#v", updated.step, cmd)
	}
	if updated.confirmed {
		t.Fatal("expected non-key confirm update to leave confirmation false")
	}
}

func TestStartViewAndPathHelpersAdditionalBranches(t *testing.T) {
	if got := (startModel{step: 999}).View(); got != "\n" {
		t.Fatalf("expected unknown start view to return newline, got %q", got)
	}

	file, err := os.CreateTemp(t.TempDir(), "closed-*.txt")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}
	opened, err := os.Open(path)
	if err != nil {
		t.Fatalf("open temp file: %v", err)
	}
	if err := opened.Close(); err != nil {
		t.Fatalf("close opened file: %v", err)
	}
	if isInteractiveFile(opened) {
		t.Fatal("expected closed file stat failure to be treated as non-interactive")
	}

	root := t.TempDir()
	relativePath, err := normalizePath(root, filepath.Join("nested", "file.txt"))
	if err != nil {
		t.Fatalf("normalize relative path: %v", err)
	}
	if relativePath != filepath.Join(root, "nested", "file.txt") {
		t.Fatalf("expected normalized relative path, got %q", relativePath)
	}
	absolutePath, err := normalizePath(root, filepath.Join(root, "already.txt"))
	if err != nil {
		t.Fatalf("normalize absolute path: %v", err)
	}
	if absolutePath != filepath.Join(root, "already.txt") {
		t.Fatalf("expected normalized absolute path, got %q", absolutePath)
	}
	if _, err := normalizeExistingPath(root, filepath.Join(root, "missing.txt")); err == nil {
		t.Fatal("expected missing existing path to fail")
	}
	if err := os.WriteFile(filepath.Join(root, "existing.txt"), []byte("ok"), 0644); err != nil {
		t.Fatalf("write existing path: %v", err)
	}
	resolvedExisting, err := normalizeExistingPath(root, filepath.Join(root, "existing.txt"))
	if err != nil {
		t.Fatalf("normalize existing path: %v", err)
	}
	if resolvedExisting != filepath.Join(root, "existing.txt") {
		t.Fatalf("expected resolved existing path, got %q", resolvedExisting)
	}
}

func TestRunStartTUIUsesProgramRunnerResult(t *testing.T) {
	originalProgramRunner := startProgramRunner
	t.Cleanup(func() { startProgramRunner = originalProgramRunner })

	startProgramRunner = func(model tea.Model) (tea.Model, error) {
		startModelValue, ok := model.(startModel)
		if !ok {
			t.Fatalf("expected start model, got %T", model)
		}
		startModelValue.confirmed = true
		startModelValue.selection.Preset = startModelValue.presets[1]
		startModelValue.inputs[0].SetValue("starter-app")
		startModelValue.inputs[1].SetValue("example.com/starter-app")
		startModelValue.inputs[2].SetValue("Cam")
		startModelValue.inputs[3].SetValue("1.2.3")
		startModelValue.inputs[4].SetValue("Starter description")
		return startModelValue, nil
	}

	selection, err := runStartTUI()
	if err != nil {
		t.Fatalf("run start tui: %v", err)
	}
	if selection == nil {
		t.Fatal("expected confirmed selection")
	}
	if selection.Preset.Key != "routed-spa" || selection.ProjectName != "starter-app" {
		t.Fatalf("expected selection from final start model, got %#v", selection)
	}
	if selection.ModulePath != "example.com/starter-app" || selection.Author != "Cam" || selection.Version != "1.2.3" {
		t.Fatalf("expected populated selection values, got %#v", selection)
	}
	if filepath.Base(selection.TargetDir) != "starter-app" {
		t.Fatalf("expected target dir derived from project name, got %#v", selection)
	}
}

func TestRunStartTUIHandlesCancelAndProgramError(t *testing.T) {
	originalProgramRunner := startProgramRunner
	t.Cleanup(func() { startProgramRunner = originalProgramRunner })

	startProgramRunner = func(model tea.Model) (tea.Model, error) {
		return startModel{}, nil
	}
	selection, err := runStartTUI()
	if err != nil {
		t.Fatalf("run start tui cancel path: %v", err)
	}
	if selection != nil {
		t.Fatalf("expected nil selection on unconfirmed model, got %#v", selection)
	}

	startProgramRunner = func(model tea.Model) (tea.Model, error) {
		return nil, errors.New("bubbletea failed")
	}
	selection, err = runStartTUI()
	if err == nil || !strings.Contains(err.Error(), "bubbletea failed") {
		t.Fatalf("expected bubbled program error, got selection=%#v err=%v", selection, err)
	}
}

func TestRunStartPostTUIUsesProgramRunnerResult(t *testing.T) {
	originalProgramRunner := startProgramRunner
	t.Cleanup(func() { startProgramRunner = originalProgramRunner })

	startProgramRunner = func(model tea.Model) (tea.Model, error) {
		postModel, ok := model.(startPostModel)
		if !ok {
			t.Fatalf("expected start post model, got %T", model)
		}
		postModel.confirmed = true
		postModel.cursor = 1
		return postModel, nil
	}

	result, err := runStartPostTUI(startSelection{ProjectName: "starter-app"}, &scaffoldResult{TargetDir: `C:\tmp\starter-app`}, nil)
	if err != nil {
		t.Fatalf("run start post tui: %v", err)
	}
	if result == nil || !result.RunDev {
		t.Fatalf("expected confirmed run-dev post result, got %#v", result)
	}
}

func TestRunStartPostTUIHandlesCancelErrorAndGenerationFailure(t *testing.T) {
	originalProgramRunner := startProgramRunner
	t.Cleanup(func() { startProgramRunner = originalProgramRunner })

	startProgramRunner = func(model tea.Model) (tea.Model, error) {
		return startPostModel{}, nil
	}
	result, err := runStartPostTUI(startSelection{ProjectName: "starter-app"}, &scaffoldResult{}, nil)
	if err != nil {
		t.Fatalf("run start post tui cancel path: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil post result on unconfirmed model, got %#v", result)
	}

	startProgramRunner = func(model tea.Model) (tea.Model, error) {
		return nil, errors.New("post bubbletea failed")
	}
	result, err = runStartPostTUI(startSelection{ProjectName: "starter-app"}, &scaffoldResult{}, nil)
	if err == nil || !strings.Contains(err.Error(), "post bubbletea failed") {
		t.Fatalf("expected bubbled post-program error, got result=%#v err=%v", result, err)
	}

	startProgramRunner = func(model tea.Model) (tea.Model, error) {
		postModel, ok := model.(startPostModel)
		if !ok {
			t.Fatalf("expected start post model, got %T", model)
		}
		postModel.confirmed = true
		postModel.cursor = 1
		return postModel, nil
	}
	result, err = runStartPostTUI(startSelection{ProjectName: "starter-app"}, nil, errors.New("generation failed"))
	if err != nil {
		t.Fatalf("run start post tui generation failure path: %v", err)
	}
	if result != nil {
		t.Fatalf("expected generation failure path to suppress post result, got %#v", result)
	}
}

func TestRunStartUsesInjectedCollaborators(t *testing.T) {
	originalTerminalValidator := startTerminalValidator
	originalSelectionRunner := startSelectionRunner
	originalPostRunner := startPostRunner
	originalGenerateScaffold := startGenerateScaffold
	originalRunDev := startRunDev
	t.Cleanup(func() {
		startTerminalValidator = originalTerminalValidator
		startSelectionRunner = originalSelectionRunner
		startPostRunner = originalPostRunner
		startGenerateScaffold = originalGenerateScaffold
		startRunDev = originalRunDev
	})

	selection := startSelection{ProjectName: "starter-app", Preset: startPreset{Key: "minimal-client"}}
	result := scaffoldResult{TargetDir: `C:\tmp\starter-app`, AppPath: `C:\tmp\starter-app\main.go`, HTMLPath: `C:\tmp\starter-app\index.html`}
	appLauncher := launcher{}

	t.Run("terminal validation failure", func(t *testing.T) {
		startTerminalValidator = func(bool, bool) error { return errors.New("not interactive") }
		err := appLauncher.runStart(nil)
		if err == nil || !strings.Contains(err.Error(), "not interactive") {
			t.Fatalf("expected terminal validation error, got %v", err)
		}
	})

	t.Run("selection canceled", func(t *testing.T) {
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return nil, nil }
		err := appLauncher.runStart(nil)
		if err != nil {
			t.Fatalf("expected nil error on canceled selection, got %v", err)
		}
	})

	t.Run("selection runner error", func(t *testing.T) {
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return nil, errors.New("selection failed") }
		err := appLauncher.runStart(nil)
		if err == nil || !strings.Contains(err.Error(), "selection failed") {
			t.Fatalf("expected selection runner error, got %v", err)
		}
	})

	t.Run("generation failure still shows post tui", func(t *testing.T) {
		postCalled := false
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &selection, nil }
		startGenerateScaffold = func(l launcher, got startSelection) (scaffoldResult, error) {
			if got.ProjectName != selection.ProjectName {
				t.Fatalf("expected selection to flow into generation, got %#v", got)
			}
			return scaffoldResult{}, errors.New("generation failed")
		}
		startPostRunner = func(got startSelection, generated *scaffoldResult, generationErr error) (*startPostResult, error) {
			postCalled = true
			if generated != nil || generationErr == nil || generationErr.Error() != "generation failed" {
				t.Fatalf("expected generation error to flow into post runner, got generated=%#v err=%v", generated, generationErr)
			}
			return nil, nil
		}
		err := appLauncher.runStart(nil)
		if err != nil {
			t.Fatalf("expected generation failure path to return nil after post tui, got %v", err)
		}
		if !postCalled {
			t.Fatal("expected post tui to run after generation failure")
		}
	})

	t.Run("post tui error bubbles", func(t *testing.T) {
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &selection, nil }
		startGenerateScaffold = func(l launcher, got startSelection) (scaffoldResult, error) { return result, nil }
		startPostRunner = func(got startSelection, generated *scaffoldResult, generationErr error) (*startPostResult, error) {
			return nil, errors.New("post failed")
		}
		err := appLauncher.runStart(nil)
		if err == nil || !strings.Contains(err.Error(), "post failed") {
			t.Fatalf("expected post runner error, got %v", err)
		}
	})

	t.Run("run dev path uses scaffold args", func(t *testing.T) {
		runDevCalled := false
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &selection, nil }
		startGenerateScaffold = func(l launcher, got startSelection) (scaffoldResult, error) { return result, nil }
		startPostRunner = func(got startSelection, generated *scaffoldResult, generationErr error) (*startPostResult, error) {
			return &startPostResult{RunDev: true}, nil
		}
		startRunDev = func(l launcher, args []string) error {
			runDevCalled = true
			if fmt.Sprint(args) != fmt.Sprint(devArgsFromScaffold(result)) {
				t.Fatalf("expected scaffold dev args, got %#v", args)
			}
			return nil
		}
		err := appLauncher.runStart(nil)
		if err != nil {
			t.Fatalf("expected run dev path to succeed, got %v", err)
		}
		if !runDevCalled {
			t.Fatal("expected runDev to be called")
		}
	})

	t.Run("invalid flag parse error bubbles", func(t *testing.T) {
		startTerminalValidator = func(bool, bool) error {
			t.Fatal("expected parse failure before terminal validation")
			return nil
		}
		err := appLauncher.runStart([]string{"-definitely-invalid"})
		if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
			t.Fatalf("expected invalid flag parse error, got %v", err)
		}
	})

	t.Run("post tui nil result skips run dev", func(t *testing.T) {
		runDevCalled := false
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &selection, nil }
		startGenerateScaffold = func(l launcher, got startSelection) (scaffoldResult, error) { return result, nil }
		startPostRunner = func(got startSelection, generated *scaffoldResult, generationErr error) (*startPostResult, error) {
			return nil, nil
		}
		startRunDev = func(l launcher, args []string) error {
			runDevCalled = true
			return nil
		}
		err := appLauncher.runStart(nil)
		if err != nil {
			t.Fatalf("expected nil post result path to succeed, got %v", err)
		}
		if runDevCalled {
			t.Fatal("expected runDev to be skipped when post result is nil")
		}
	})

	t.Run("post tui exit choice skips run dev", func(t *testing.T) {
		runDevCalled := false
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &selection, nil }
		startGenerateScaffold = func(l launcher, got startSelection) (scaffoldResult, error) { return result, nil }
		startPostRunner = func(got startSelection, generated *scaffoldResult, generationErr error) (*startPostResult, error) {
			return &startPostResult{RunDev: false}, nil
		}
		startRunDev = func(l launcher, args []string) error {
			runDevCalled = true
			return nil
		}
		err := appLauncher.runStart(nil)
		if err != nil {
			t.Fatalf("expected exit choice path to succeed, got %v", err)
		}
		if runDevCalled {
			t.Fatal("expected runDev to be skipped when RunDev is false")
		}
	})

	t.Run("run dev error bubbles", func(t *testing.T) {
		startTerminalValidator = func(bool, bool) error { return nil }
		startSelectionRunner = func() (*startSelection, error) { return &selection, nil }
		startGenerateScaffold = func(l launcher, got startSelection) (scaffoldResult, error) { return result, nil }
		startPostRunner = func(got startSelection, generated *scaffoldResult, generationErr error) (*startPostResult, error) {
			return &startPostResult{RunDev: true}, nil
		}
		startRunDev = func(l launcher, args []string) error {
			return errors.New("dev launch failed")
		}
		err := appLauncher.runStart(nil)
		if err == nil || !strings.Contains(err.Error(), "dev launch failed") {
			t.Fatalf("expected runDev error to bubble, got %v", err)
		}
	})
}

func TestBuildSelectionRequiresFields(t *testing.T) {
	tests := []struct {
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := newTestStartModel()
			model.inputs[0].SetValue(test.projectName)
			model.inputs[1].SetValue(test.modulePath)
			model.inputs[2].SetValue(test.author)
			model.inputs[3].SetValue(test.version)
			model.inputs[4].SetValue(test.description)

			_, err := model.buildSelection()
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
			}
		})
	}
}

func TestBuildSelectionDerivesTargetDirFromProjectName(t *testing.T) {
	model := newTestStartModel()
	model.inputs[0].SetValue("derived-app")
	model.inputs[1].SetValue("github.com/test/derived-app")
	model.inputs[2].SetValue("Cam")
	model.inputs[3].SetValue("0.1.0")
	model.inputs[4].SetValue("desc")

	selection, err := model.buildSelection()
	if err != nil {
		t.Fatalf("build selection: %v", err)
	}
	if filepath.Base(selection.TargetDir) != "derived-app" {
		t.Fatalf("expected target dir base to match project name, got %q", selection.TargetDir)
	}
}

func TestValidateGeneratedTargetDirAllowsUserWorkspacePath(t *testing.T) {
	targetDir := filepath.Join(t.TempDir(), "starter-app")
	if err := validateGeneratedTargetDir(targetDir); err != nil {
		t.Fatalf("expected user workspace path %q to be allowed, got %v", targetDir, err)
	}
}

func TestValidateGeneratedTargetDirRejectsFilesystemRoot(t *testing.T) {
	rootPath := string(os.PathSeparator)
	if volume := filepath.VolumeName(rootPath); volume != "" {
		rootPath = volume + string(os.PathSeparator)
	}
	if err := validateGeneratedTargetDir(rootPath); err == nil {
		t.Fatal("expected filesystem root to be rejected")
	}
}

func TestEnsureEmptyDirRejectsNonEmptyDirectory(t *testing.T) {
	targetDir := filepath.Join(defaultGeneratedScaffoldRoot(), "test-non-empty-dir")
	_ = os.RemoveAll(targetDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("create target dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(targetDir)
	})
	if err := os.WriteFile(filepath.Join(targetDir, "existing.txt"), []byte("occupied"), 0644); err != nil {
		t.Fatalf("seed target dir: %v", err)
	}

	if err := ensureEmptyDir(targetDir); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("expected non-empty directory error, got %v", err)
	}
}

func TestEnsureEmptyDirRejectsExistingFile(t *testing.T) {
	targetPath := filepath.Join(defaultGeneratedScaffoldRoot(), "test-existing-file")
	_ = os.Remove(targetPath)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		t.Fatalf("create parent dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(targetPath)
	})
	if err := os.WriteFile(targetPath, []byte("file"), 0644); err != nil {
		t.Fatalf("seed target path: %v", err)
	}

	if err := ensureEmptyDir(targetPath); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected existing file error, got %v", err)
	}
}

func TestEnsureEmptyDirAllowsCreateAndEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	newDir := filepath.Join(root, "new-target")
	if err := ensureEmptyDir(newDir); err != nil {
		t.Fatalf("expected missing directory to be created, got %v", err)
	}
	if info, err := os.Stat(newDir); err != nil || !info.IsDir() {
		t.Fatalf("expected created directory to exist, info=%#v err=%v", info, err)
	}
	if err := ensureEmptyDir(newDir); err != nil {
		t.Fatalf("expected existing empty directory to remain valid, got %v", err)
	}
}

func TestDetectHTMLPathPrefersIndexHTML(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "index.html")
	otherPath := filepath.Join(root, "other.html")
	if err := os.WriteFile(otherPath, []byte("other"), 0644); err != nil {
		t.Fatalf("write other html: %v", err)
	}
	if err := os.WriteFile(indexPath, []byte("index"), 0644); err != nil {
		t.Fatalf("write index html: %v", err)
	}

	if got := detectHTMLPath(root); got != indexPath {
		t.Fatalf("expected index.html to be preferred, got %q", got)
	}
}

func TestDetectHTMLPathFallsBackToFirstHTMLFile(t *testing.T) {
	root := t.TempDir()
	fallbackPath := filepath.Join(root, "app-shell.html")
	if err := os.WriteFile(fallbackPath, []byte("shell"), 0644); err != nil {
		t.Fatalf("write fallback html: %v", err)
	}

	if got := detectHTMLPath(root); got != fallbackPath {
		t.Fatalf("expected fallback html path %q, got %q", fallbackPath, got)
	}
}

func TestDetectHTMLPathReturnsEmptyForNonDirectoryRoot(t *testing.T) {
	rootFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(rootFile, []byte("occupied"), 0644); err != nil {
		t.Fatalf("write root file: %v", err)
	}

	if got := detectHTMLPath(rootFile); got != "" {
		t.Fatalf("expected empty html path for non-directory root, got %q", got)
	}
}

func TestRenderScaffoldGoModIncludesReplaceDirective(t *testing.T) {
	selection := startSelection{ModulePath: "github.com/test/app"}
	content := renderScaffoldGoMod(selection, "github.com/monstercameron/GoWebComponents", "../../..")
	for _, expected := range []string{"module github.com/test/app", "require github.com/monstercameron/GoWebComponents v0.0.0", "replace github.com/monstercameron/GoWebComponents => ../../.."} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected go.mod content to contain %q", expected)
		}
	}
}

func TestRenderScaffoldMetadataIncludesToolingDefaults(t *testing.T) {
	selection := startSelection{
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

	content := renderScaffoldMetadata(selection)
	var metadata scaffoldMetadata
	if err := json.Unmarshal([]byte(content), &metadata); err != nil {
		t.Fatalf("unmarshal scaffold metadata: %v\n%s", err, content)
	}
	if metadata.ProjectName != selection.ProjectName || metadata.ModulePath != selection.ModulePath {
		t.Fatalf("expected project metadata to round-trip, got %#v", metadata)
	}
	if metadata.Tooling.AppPath != "main.go" || metadata.Tooling.HTMLPath != "index.html" || metadata.Tooling.WASMPath != "main.wasm" {
		t.Fatalf("expected default tooling paths, got %#v", metadata.Tooling)
	}
	if metadata.Tooling.ReleaseOutDir != filepath.ToSlash(filepath.Join("dist", "wasm-release")) || metadata.Tooling.ReleaseBinaryName != "app.wasm" || metadata.Tooling.ReleaseCompression != "gzip+brotli" {
		t.Fatalf("expected release tooling defaults, got %#v", metadata.Tooling)
	}
	if metadata.Preset.Key != selection.Preset.Key || len(metadata.Preset.Features) != 2 {
		t.Fatalf("expected preset metadata to round-trip, got %#v", metadata.Preset)
	}
	if !strings.HasSuffix(content, "\n") {
		t.Fatalf("expected scaffold metadata to end with newline, got %q", content)
	}
}

func TestRenderScaffoldMetadataFallsBackWhenMarshalFails(t *testing.T) {
	originalMarshal := scaffoldMarshalIndent
	t.Cleanup(func() { scaffoldMarshalIndent = originalMarshal })
	scaffoldMarshalIndent = func(v interface{}, prefix string, indent string) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}

	if got := renderScaffoldMetadata(startSelection{}); got != "{}\n" {
		t.Fatalf("expected marshal failure fallback payload, got %q", got)
	}
}

func TestSeedScaffoldGoSumAllowsMissingRepoFile(t *testing.T) {
	targetDir := t.TempDir()
	launcher := launcher{repoRoot: t.TempDir()}
	if err := launcher.seedScaffoldGoSum(targetDir); err != nil {
		t.Fatalf("expected missing repo go.sum to be ignored, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "go.sum")); !os.IsNotExist(err) {
		t.Fatalf("expected no scaffold go.sum to be written, got err=%v", err)
	}
}

func TestSeedScaffoldGoSumReadAndWriteErrors(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, "go.sum"), 0755); err != nil {
		t.Fatalf("mkdir repo go.sum dir: %v", err)
	}
	if err := (launcher{repoRoot: repoRoot}).seedScaffoldGoSum(t.TempDir()); err == nil || !strings.Contains(err.Error(), "read repo go.sum") {
		t.Fatalf("expected repo go.sum read error, got %v", err)
	}

	repoRoot = t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "go.sum"), []byte("sum data"), 0644); err != nil {
		t.Fatalf("write repo go.sum: %v", err)
	}
	targetDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(targetDir, "go.sum"), 0755); err != nil {
		t.Fatalf("mkdir target go.sum dir: %v", err)
	}
	if err := (launcher{repoRoot: repoRoot}).seedScaffoldGoSum(targetDir); err == nil || !strings.Contains(err.Error(), "write scaffold go.sum") {
		t.Fatalf("expected scaffold go.sum write error, got %v", err)
	}
}

func TestNormalizeExistingPathBlankInputReturnsStatError(t *testing.T) {
	resolved, err := normalizeExistingPath(t.TempDir(), "   ")
	if err == nil {
		t.Fatalf("expected blank existing path to fail stat lookup, got resolved=%q", resolved)
	}
}

func TestDevArgsFromScaffoldIncludesGeneratedPaths(t *testing.T) {
	result := scaffoldResult{
		TargetDir: filepath.Join("tmp", "gwc-start", "app"),
		AppPath:   filepath.Join("tmp", "gwc-start", "app", "main.go"),
		HTMLPath:  filepath.Join("tmp", "gwc-start", "app", "index.html"),
	}

	got := devArgsFromScaffold(result)
	want := []string{"-app", result.AppPath, "-root", result.TargetDir, "-html", result.HTMLPath, "-wasm", scaffoldWASMOutputPath()}
	if len(got) != len(want) {
		t.Fatalf("expected %d dev args, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected dev arg %d to be %q, got %q", i, want[i], got[i])
		}
	}
}

func TestSeedScaffoldGoSumCopiesRootGoSum(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	launcher := launcher{repoRoot: repoRoot}
	targetDir := t.TempDir()

	if err := launcher.seedScaffoldGoSum(targetDir); err != nil {
		t.Fatalf("seed scaffold go.sum: %v", err)
	}

	rootGoSum, err := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if err != nil {
		t.Fatalf("read repo go.sum: %v", err)
	}
	targetGoSum, err := os.ReadFile(filepath.Join(targetDir, "go.sum"))
	if err != nil {
		t.Fatalf("read target go.sum: %v", err)
	}
	if string(rootGoSum) != string(targetGoSum) {
		t.Fatal("expected scaffold go.sum to match repo go.sum")
	}
}

func TestGenerateStartScaffoldWritesStarterFiles(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	launcher := launcher{repoRoot: repoRoot}
	targetDir := filepath.Join(defaultGeneratedScaffoldRoot(), "test-generate-start-scaffold")
	_ = os.RemoveAll(targetDir)
	t.Cleanup(func() {
		_ = os.RemoveAll(targetDir)
	})

	selection := startSelection{
		Preset: startPreset{
			Key:         "minimal-client",
			Name:        "Minimal Client App",
			Summary:     "Test preset",
			Description: "Generated in tests.",
			Features:    []string{"ui", "html"},
		},
		ProjectName: "test-generate-start-scaffold",
		ModulePath:  "github.com/example/test-generate-start-scaffold",
		Author:      "Test Author",
		Version:     "1.2.3",
		Description: "Generated metadata test app.",
		TargetDir:   targetDir,
	}

	result, err := launcher.generateStartScaffold(selection)
	if err != nil {
		t.Fatalf("generate scaffold: %v", err)
	}

	for _, path := range []string{
		filepath.Join(targetDir, "go.mod"),
		filepath.Join(targetDir, "go.sum"),
		filepath.Join(targetDir, "main.go"),
		filepath.Join(targetDir, "index.html"),
		filepath.Join(targetDir, "gwc-start.json"),
		filepath.Join(targetDir, "README.md"),
		filepath.Join(targetDir, "wasm_exec.js"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected generated file %q: %v", path, err)
		}
	}

	if result.TargetDir != targetDir {
		t.Fatalf("expected result target dir %q, got %q", targetDir, result.TargetDir)
	}
	if result.AppPath != filepath.Join(targetDir, "main.go") {
		t.Fatalf("expected result app path to point at generated main.go, got %q", result.AppPath)
	}
	if result.HTMLPath != filepath.Join(targetDir, "index.html") {
		t.Fatalf("expected result html path to point at generated index.html, got %q", result.HTMLPath)
	}

	metadataBytes, err := os.ReadFile(filepath.Join(targetDir, "gwc-start.json"))
	if err != nil {
		t.Fatalf("read generated metadata file: %v", err)
	}
	metadataText := string(metadataBytes)
	for _, expected := range []string{"Test Author", "1.2.3", "Generated metadata test app."} {
		if !strings.Contains(metadataText, expected) {
			t.Fatalf("expected metadata file to contain %q", expected)
		}
	}
	var metadata scaffoldMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		t.Fatalf("unmarshal generated metadata: %v\n%s", err, string(metadataBytes))
	}
	if metadata.Tooling.AppPath != "main.go" {
		t.Fatalf("expected metadata app path main.go, got %#v", metadata.Tooling)
	}
	if metadata.Tooling.HTMLPath != "index.html" {
		t.Fatalf("expected metadata html path index.html, got %#v", metadata.Tooling)
	}
	if metadata.Tooling.WASMPath != "main.wasm" {
		t.Fatalf("expected metadata wasm path main.wasm, got %#v", metadata.Tooling)
	}
	if metadata.Tooling.DefaultBuildProfile != "development" {
		t.Fatalf("expected default build profile metadata, got %#v", metadata.Tooling)
	}
	if metadata.Tooling.ReleaseOutDir != filepath.ToSlash(filepath.Join("dist", "wasm-release")) {
		t.Fatalf("expected default release out dir metadata, got %#v", metadata.Tooling)
	}
	if metadata.Tooling.ReleaseBinaryName != "app.wasm" {
		t.Fatalf("expected default release binary metadata, got %#v", metadata.Tooling)
	}
	if metadata.Tooling.ReleaseCompression != "gzip+brotli" {
		t.Fatalf("expected default release compression metadata, got %#v", metadata.Tooling)
	}

	readmeBytes, err := os.ReadFile(filepath.Join(targetDir, "README.md"))
	if err != nil {
		t.Fatalf("read generated README file: %v", err)
	}
	if !strings.Contains(string(readmeBytes), "-wasm \"main.wasm\"") {
		t.Fatalf("expected generated README to include explicit wasm output flag, got:\n%s", string(readmeBytes))
	}

	htmlBytes, err := os.ReadFile(filepath.Join(targetDir, "index.html"))
	if err != nil {
		t.Fatalf("read generated html file: %v", err)
	}
	htmlText := string(htmlBytes)
	for _, expected := range []string{"color-scheme: dark", "radial-gradient(circle at top", "linear-gradient(160deg"} {
		if !strings.Contains(htmlText, expected) {
			t.Fatalf("expected generated html to contain %q", expected)
		}
	}

	mainBytes, err := os.ReadFile(filepath.Join(targetDir, "main.go"))
	if err != nil {
		t.Fatalf("read generated main.go file: %v", err)
	}
	mainText := string(mainBytes)
	for _, expected := range []string{"GoWebComponents Placeholder Brand", "Dark-mode starter scaffold generated by gwc start"} {
		if !strings.Contains(mainText, expected) {
			t.Fatalf("expected generated main.go to contain %q", expected)
		}
	}

	buildCmd := exec.Command("go", "build", "-o", "main.wasm", ".")
	buildCmd.Dir = targetDir
	buildCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected generated scaffold to build for wasm, got error: %v\n%s", err, string(buildOutput))
	}
}

func TestRunDevDryRunJSONResolvesGeneratedScaffoldPlan(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	launcher := launcher{repoRoot: repoRoot}
	targetDir := filepath.Join(defaultGeneratedScaffoldRoot(), "test-run-dev-dry-run-json")
	_ = os.RemoveAll(targetDir)
	t.Cleanup(func() {
		_ = os.RemoveAll(targetDir)
	})

	selection := startSelection{
		Preset: startPreset{
			Key:         "minimal-client",
			Name:        "Minimal Client App",
			Summary:     "Test preset",
			Description: "Generated in tests.",
			Features:    []string{"ui", "html"},
		},
		ProjectName: "test-run-dev-dry-run-json",
		ModulePath:  "github.com/example/test-run-dev-dry-run-json",
		Author:      "Test Author",
		Version:     "1.2.3",
		Description: "Generated metadata test app.",
		TargetDir:   targetDir,
	}

	result, err := launcher.generateStartScaffold(selection)
	if err != nil {
		t.Fatalf("generate scaffold: %v", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(workingDir)
	})

	stdout, restoreStdout, err := captureStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	args := append(devArgsFromScaffold(result), "-dry-run", "-json", "-port", "8137")
	if err := launcher.runDev(args); err != nil {
		t.Fatalf("run dev dry-run json: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}

	var plan struct {
		App  string `json:"app"`
		Root string `json:"root"`
		HTML string `json:"html"`
		WASM string `json:"wasm"`
		Host string `json:"host"`
		Port string `json:"port"`
		Hot  bool   `json:"hot"`
	}
	if err := json.Unmarshal([]byte(output), &plan); err != nil {
		t.Fatalf("unmarshal dev plan json: %v\noutput: %s", err, output)
	}

	if plan.App != result.AppPath {
		t.Fatalf("expected app path %q, got %q", result.AppPath, plan.App)
	}
	if plan.Root != result.TargetDir {
		t.Fatalf("expected root path %q, got %q", result.TargetDir, plan.Root)
	}
	if plan.HTML != result.HTMLPath {
		t.Fatalf("expected html path %q, got %q", result.HTMLPath, plan.HTML)
	}
	if plan.WASM != scaffoldWASMOutputPath() {
		t.Fatalf("expected wasm path %q, got %q", scaffoldWASMOutputPath(), plan.WASM)
	}
	if plan.Host != "127.0.0.1" {
		t.Fatalf("expected default host 127.0.0.1, got %q", plan.Host)
	}
	if plan.Port != "8137" {
		t.Fatalf("expected port 8137, got %q", plan.Port)
	}
	if !plan.Hot {
		t.Fatal("expected hot reload to remain enabled in dry-run plan")
	}
}

func TestRunDevFailsForMissingAppPath(t *testing.T) {
	root := t.TempDir()
	launcher := launcher{repoRoot: root}

	err := launcher.runDev([]string{"-app", filepath.Join(root, "missing.go"), "-root", root, "-dry-run"})
	if err == nil {
		t.Fatal("expected missing app path to fail")
	}
	if !strings.Contains(err.Error(), "resolve app path") {
		t.Fatalf("expected missing app path error, got %v", err)
	}
}

func TestRunDevExecutesForwardedCommandAndSurfacesFailure(t *testing.T) {
	root := t.TempDir()
	appDir := filepath.Join(root, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "tools", "livereload"), 0755); err != nil {
		t.Fatalf("mkdir livereload dir: %v", err)
	}
	appPath := filepath.Join(appDir, "main.go")
	htmlPath := filepath.Join(appDir, "index.html")
	if err := os.WriteFile(appPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(htmlPath, []byte("<html></html>\n"), 0644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	binDir := filepath.Join(root, "fake-bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	argsPath := filepath.Join(root, "go-args.txt")
	goBatPath := filepath.Join(binDir, "go.bat")
	goBat := "@echo off\r\n" +
		"echo %*>\"" + argsPath + "\"\r\n" +
		"if /I \"%1\"==\"fail-now\" exit /b 9\r\n" +
		"exit /b 0\r\n"
	if err := os.WriteFile(goBatPath, []byte(goBat), 0644); err != nil {
		t.Fatalf("write fake go.bat: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	devLauncher := launcher{repoRoot: root}
	if err := devLauncher.runDev([]string{"-main", appPath, "-root", appDir, "-index", htmlPath, "-output", "dist/app.wasm", "-host", "0.0.0.0", "-port", "9123", "-hot=false", "-client-script", "custom-client.js"}); err != nil {
		t.Fatalf("run dev forwarded command: %v", err)
	}
	argsBytes, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read forwarded args: %v", err)
	}
	argsText := string(argsBytes)
	for _, expected := range []string{"run .", "-app " + appPath, "-root " + appDir, "-html " + htmlPath, "-wasm dist/app.wasm", "-host 0.0.0.0", "-port 9123", "-hot false", "-client-script custom-client.js"} {
		if !strings.Contains(argsText, expected) {
			t.Fatalf("expected forwarded args to contain %q, got %q", expected, argsText)
		}
	}

	if err := os.WriteFile(goBatPath, []byte("@echo off\r\nexit /b 9\r\n"), 0644); err != nil {
		t.Fatalf("rewrite failing go.bat: %v", err)
	}
	err = devLauncher.runDev([]string{"-app", appPath, "-root", appDir, "-html", htmlPath})
	if err == nil {
		t.Fatal("expected forwarded go run failure")
	}
}

func TestScaffoldHelperFailureBranches(t *testing.T) {
	originalScaffoldRel := scaffoldRel
	originalResolveWasmExecPath := scaffoldResolveWasmExecPath
	originalScaffoldWriteFile := scaffoldWriteFile
	originalScaffoldReadFile := scaffoldReadFile
	originalScaffoldFormatMain := scaffoldFormatMain
	t.Cleanup(func() {
		scaffoldRel = originalScaffoldRel
		scaffoldResolveWasmExecPath = originalResolveWasmExecPath
		scaffoldWriteFile = originalScaffoldWriteFile
		scaffoldReadFile = originalScaffoldReadFile
		scaffoldFormatMain = originalScaffoldFormatMain
	})

	t.Run("seed scaffold go.sum tolerates missing root file", func(t *testing.T) {
		testLauncher := launcher{repoRoot: t.TempDir()}
		if err := testLauncher.seedScaffoldGoSum(t.TempDir()); err != nil {
			t.Fatalf("expected missing repo go.sum to be ignored, got %v", err)
		}
	})

	t.Run("seed scaffold go.sum read error", func(t *testing.T) {
		repoRoot := t.TempDir()
		if err := os.MkdirAll(filepath.Join(repoRoot, "go.sum"), 0755); err != nil {
			t.Fatalf("mkdir go.sum dir: %v", err)
		}
		testLauncher := launcher{repoRoot: repoRoot}
		err := testLauncher.seedScaffoldGoSum(t.TempDir())
		if err == nil || !strings.Contains(err.Error(), "read repo go.sum") {
			t.Fatalf("expected repo go.sum read error, got %v", err)
		}
	})

	t.Run("read repo module path failures", func(t *testing.T) {
		testLauncher := launcher{repoRoot: t.TempDir()}
		if _, err := testLauncher.readRepoModulePath(); err == nil || !strings.Contains(err.Error(), "read repo go.mod") {
			t.Fatalf("expected missing go.mod error, got %v", err)
		}

		repoRoot := t.TempDir()
		if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("go 1.25.0\n"), 0644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
		testLauncher = launcher{repoRoot: repoRoot}
		if _, err := testLauncher.readRepoModulePath(); err == nil || !strings.Contains(err.Error(), "repo module path not found") {
			t.Fatalf("expected missing module line error, got %v", err)
		}
	})

	t.Run("generate scaffold fails before file writes when repo metadata is missing", func(t *testing.T) {
		targetDir := filepath.Join(t.TempDir(), "starter-app")
		selection := startSelection{
			Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
			ProjectName: "starter-app",
			ModulePath:  "example.com/starter-app",
			Author:      "Cam",
			Version:     "0.1.0",
			Description: "starter",
			TargetDir:   targetDir,
		}
		testLauncher := launcher{repoRoot: t.TempDir()}
		_, err := testLauncher.generateStartScaffold(selection)
		if err == nil || !strings.Contains(err.Error(), "read repo go.mod") {
			t.Fatalf("expected missing repo module path error, got %v", err)
		}
	})

	t.Run("generate scaffold fails when scaffold module cannot be prepared", func(t *testing.T) {
		originalGetwd := launcherConfigGetwd
		t.Cleanup(func() { launcherConfigGetwd = originalGetwd })
		repoRoot := t.TempDir()
		if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/repo\n\ngo 1.25.0\n"), 0644); err != nil {
			t.Fatalf("write repo go.mod: %v", err)
		}
		launcherConfigGetwd = func() (string, error) { return repoRoot, nil }
		t.Setenv(launcherOverrideEnvVar, "")
		selection := startSelection{
			Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
			ProjectName: "starter-app",
			ModulePath:  "example.com/starter-app",
			Author:      "Cam",
			Version:     "0.1.0",
			Description: "starter",
			TargetDir:   filepath.Join(t.TempDir(), "starter-app"),
		}
		testLauncher := launcher{repoRoot: repoRoot}
		_, err := testLauncher.generateStartScaffold(selection)
		if err == nil || !strings.Contains(err.Error(), "prepare scaffold module") {
			t.Fatalf("expected scaffold module preparation error, got %v", err)
		}
	})

	t.Run("tidy scaffold module surfaces blank output failures", func(t *testing.T) {
		binDir := filepath.Join(t.TempDir(), "bin")
		if err := os.MkdirAll(binDir, 0755); err != nil {
			t.Fatalf("mkdir fake bin: %v", err)
		}
		goBatPath := filepath.Join(binDir, "go.bat")
		if err := os.WriteFile(goBatPath, []byte("@echo off\r\nexit /b 7\r\n"), 0644); err != nil {
			t.Fatalf("write fake go.bat: %v", err)
		}
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		err := (launcher{}).tidyScaffoldModule(t.TempDir())
		if err == nil || !strings.Contains(err.Error(), "prepare scaffold module: exit status") {
			t.Fatalf("expected blank-output tidy error, got %v", err)
		}
	})

	t.Run("generate scaffold fails when wasm_exec override does not point to a file", func(t *testing.T) {
		repoRoot := t.TempDir()
		if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/repo\n\ngo 1.25.0\n"), 0644); err != nil {
			t.Fatalf("write repo go.mod: %v", err)
		}

		overrideDir := filepath.Join(t.TempDir(), "vendor", "wasm_exec.js")
		if err := os.MkdirAll(overrideDir, 0755); err != nil {
			t.Fatalf("mkdir override dir: %v", err)
		}
		overrides := launcherOverrides{Paths: launcherOverridePaths{WASMExecJS: overrideDir}}
		overrideBytes, err := json.Marshal(overrides)
		if err != nil {
			t.Fatalf("marshal overrides: %v", err)
		}
		configPath := filepath.Join(t.TempDir(), "runner.json")
		if err := os.WriteFile(configPath, overrideBytes, 0644); err != nil {
			t.Fatalf("write runner config: %v", err)
		}
		t.Setenv(launcherOverrideEnvVar, configPath)

		selection := startSelection{
			Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
			ProjectName: "starter-app",
			ModulePath:  "example.com/starter-app",
			Author:      "Cam",
			Version:     "0.1.0",
			Description: "starter",
			TargetDir:   filepath.Join(t.TempDir(), "starter-app"),
		}
		testLauncher := launcher{repoRoot: repoRoot}
		_, err = testLauncher.generateStartScaffold(selection)
		if err == nil || !strings.Contains(err.Error(), "configured wasmExecJS path does not exist") {
			t.Fatalf("expected missing wasm_exec.js override error, got %v", err)
		}
	})

	t.Run("generate scaffold fails when gofmt fails", func(t *testing.T) {
		repoRoot := t.TempDir()
		if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/repo\n\ngo 1.25.0\n"), 0644); err != nil {
			t.Fatalf("write repo go.mod: %v", err)
		}

		wasmExecPath := filepath.Join(t.TempDir(), "wasm_exec.js")
		if err := os.WriteFile(wasmExecPath, []byte("console.log('wasm');\n"), 0644); err != nil {
			t.Fatalf("write wasm_exec.js: %v", err)
		}
		overrides := launcherOverrides{Paths: launcherOverridePaths{WASMExecJS: wasmExecPath}}
		overrideBytes, err := json.Marshal(overrides)
		if err != nil {
			t.Fatalf("marshal overrides: %v", err)
		}
		configPath := filepath.Join(t.TempDir(), "runner.json")
		if err := os.WriteFile(configPath, overrideBytes, 0644); err != nil {
			t.Fatalf("write runner config: %v", err)
		}
		t.Setenv(launcherOverrideEnvVar, configPath)

		binDir := filepath.Join(t.TempDir(), "bin")
		if err := os.MkdirAll(binDir, 0755); err != nil {
			t.Fatalf("mkdir fake bin: %v", err)
		}
		if err := os.WriteFile(filepath.Join(binDir, "go.bat"), []byte("@echo off\r\nexit /b 0\r\n"), 0644); err != nil {
			t.Fatalf("write fake go.bat: %v", err)
		}
		if err := os.WriteFile(filepath.Join(binDir, "gofmt.bat"), []byte("@echo off\r\nexit /b 9\r\n"), 0644); err != nil {
			t.Fatalf("write fake gofmt.bat: %v", err)
		}
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		selection := startSelection{
			Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
			ProjectName: "starter-app",
			ModulePath:  "example.com/starter-app",
			Author:      "Cam",
			Version:     "0.1.0",
			Description: "starter",
			TargetDir:   filepath.Join(t.TempDir(), "starter-app"),
		}
		_, err = (launcher{repoRoot: repoRoot}).generateStartScaffold(selection)
		if err == nil || !strings.Contains(err.Error(), "format generated main.go") {
			t.Fatalf("expected gofmt failure, got %v", err)
		}
	})

	t.Run("generate scaffold injected rel and write failures", func(t *testing.T) {
		repoRoot := t.TempDir()
		if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/repo\n\ngo 1.25.0\n"), 0644); err != nil {
			t.Fatalf("write repo go.mod: %v", err)
		}
		wasmExecPath := filepath.Join(t.TempDir(), "wasm_exec.js")
		if err := os.WriteFile(wasmExecPath, []byte("console.log('wasm');\n"), 0644); err != nil {
			t.Fatalf("write wasm_exec.js: %v", err)
		}
		scaffoldResolveWasmExecPath = func() (string, error) { return wasmExecPath, nil }
		scaffoldFormatMain = func(string) error { return nil }

		newSelection := func() startSelection {
			return startSelection{
				Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
				ProjectName: "starter-app",
				ModulePath:  "example.com/starter-app",
				Author:      "Cam",
				Version:     "0.1.0",
				Description: "starter",
				TargetDir:   filepath.Join(t.TempDir(), "starter-app"),
			}
		}

		scaffoldRel = func(base string, target string) (string, error) {
			return "", errors.New("rel failed")
		}
		_, err := (launcher{repoRoot: repoRoot}).generateStartScaffold(newSelection())
		if err == nil || !strings.Contains(err.Error(), "resolve repo replace path") {
			t.Fatalf("expected rel failure, got %v", err)
		}
		scaffoldRel = originalScaffoldRel

		writeFailures := []struct {
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
		for _, testCase := range writeFailures {
			t.Run(testCase.name, func(t *testing.T) {
				scaffoldWriteFile = func(name string, data []byte, perm os.FileMode) error {
					if strings.HasSuffix(filepath.ToSlash(name), testCase.suffix) {
						return errors.New("write failed")
					}
					return originalScaffoldWriteFile(name, data, perm)
				}
				defer func() { scaffoldWriteFile = originalScaffoldWriteFile }()

				_, err := (launcher{repoRoot: repoRoot}).generateStartScaffold(newSelection())
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("expected %s failure, got %v", testCase.wantErr, err)
				}
			})
		}
	})

	t.Run("generate scaffold injected read and seed failures", func(t *testing.T) {
		repoRoot := t.TempDir()
		if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/repo\n\ngo 1.25.0\n"), 0644); err != nil {
			t.Fatalf("write repo go.mod: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(repoRoot, "go.sum"), 0755); err != nil {
			t.Fatalf("mkdir repo go.sum dir: %v", err)
		}
		wasmExecPath := filepath.Join(t.TempDir(), "wasm_exec.js")
		if err := os.WriteFile(wasmExecPath, []byte("console.log('wasm');\n"), 0644); err != nil {
			t.Fatalf("write wasm_exec.js: %v", err)
		}
		scaffoldResolveWasmExecPath = func() (string, error) { return wasmExecPath, nil }
		scaffoldFormatMain = func(string) error { return nil }

		newSelection := func() startSelection {
			return startSelection{
				Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
				ProjectName: "starter-app",
				ModulePath:  "example.com/starter-app",
				Author:      "Cam",
				Version:     "0.1.0",
				Description: "starter",
				TargetDir:   filepath.Join(t.TempDir(), "starter-app"),
			}
		}

		scaffoldReadFile = func(name string) ([]byte, error) {
			if filepath.Clean(name) == filepath.Clean(wasmExecPath) {
				return nil, errors.New("read failed")
			}
			return originalScaffoldReadFile(name)
		}
		_, err := (launcher{repoRoot: repoRoot}).generateStartScaffold(newSelection())
		if err == nil || !strings.Contains(err.Error(), "read wasm_exec.js") {
			t.Fatalf("expected wasm_exec read failure, got %v", err)
		}
		scaffoldReadFile = originalScaffoldReadFile

		_, err = (launcher{repoRoot: repoRoot}).generateStartScaffold(newSelection())
		if err == nil || !strings.Contains(err.Error(), "read repo go.sum") {
			t.Fatalf("expected seed scaffold go.sum failure, got %v", err)
		}
	})
}

func TestGenerateStartScaffoldEarlyValidationBranches(t *testing.T) {
	t.Run("rejects filesystem root target", func(t *testing.T) {
		targetDir := string(os.PathSeparator)
		if volume := filepath.VolumeName(targetDir); volume != "" {
			targetDir = volume + string(os.PathSeparator)
		}
		selection := startSelection{
			Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
			ProjectName: "starter-app",
			ModulePath:  "example.com/starter-app",
			Author:      "Cam",
			Version:     "0.1.0",
			Description: "starter",
			TargetDir:   targetDir,
		}
		_, err := (launcher{repoRoot: t.TempDir()}).generateStartScaffold(selection)
		if err == nil || !strings.Contains(err.Error(), "filesystem root") {
			t.Fatalf("expected target root validation error, got %v", err)
		}
	})

	t.Run("rejects existing file target", func(t *testing.T) {
		targetDir := filepath.Join(t.TempDir(), "existing-target")
		if err := os.WriteFile(targetDir, []byte("occupied"), 0644); err != nil {
			t.Fatalf("write target file: %v", err)
		}
		selection := startSelection{
			Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
			ProjectName: "starter-app",
			ModulePath:  "example.com/starter-app",
			Author:      "Cam",
			Version:     "0.1.0",
			Description: "starter",
			TargetDir:   targetDir,
		}
		_, err := (launcher{repoRoot: t.TempDir()}).generateStartScaffold(selection)
		if err == nil || !strings.Contains(err.Error(), "not a directory") {
			t.Fatalf("expected existing file target error, got %v", err)
		}
	})

	t.Run("rejects target under file parent", func(t *testing.T) {
		parentFile := filepath.Join(t.TempDir(), "occupied")
		if err := os.WriteFile(parentFile, []byte("occupied"), 0644); err != nil {
			t.Fatalf("write parent file: %v", err)
		}
		selection := startSelection{
			Preset:      startPreset{Key: "minimal-client", Name: "Minimal Client"},
			ProjectName: "starter-app",
			ModulePath:  "example.com/starter-app",
			Author:      "Cam",
			Version:     "0.1.0",
			Description: "starter",
			TargetDir:   filepath.Join(parentFile, "starter-app"),
		}
		_, err := (launcher{repoRoot: t.TempDir()}).generateStartScaffold(selection)
		if err == nil || !strings.Contains(err.Error(), "create target directory") {
			t.Fatalf("expected target creation error, got %v", err)
		}
	})
}

func TestRunStartHelpReturnsNil(t *testing.T) {
	if err := (launcher{}).run([]string{"start", "-help"}); err != nil {
		t.Fatalf("expected start help to succeed, got %v", err)
	}
}

func TestGeneratedScaffoldServesOverDevServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping dev server smoke test in short mode")
	}

	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	launcher := launcher{repoRoot: repoRoot}
	targetDir := filepath.Join(defaultGeneratedScaffoldRoot(), "test-dev-server-smoke")
	_ = os.RemoveAll(targetDir)
	t.Cleanup(func() {
		_ = os.RemoveAll(targetDir)
	})

	selection := startSelection{
		Preset: startPreset{
			Key:         "minimal-client",
			Name:        "Minimal Client App",
			Summary:     "Test preset",
			Description: "Generated in tests.",
			Features:    []string{"ui", "html"},
		},
		ProjectName: "test-dev-server-smoke",
		ModulePath:  "github.com/example/test-dev-server-smoke",
		Author:      "Test Author",
		Version:     "1.2.3",
		Description: "Generated metadata test app.",
		TargetDir:   targetDir,
	}

	result, err := launcher.generateStartScaffold(selection)
	if err != nil {
		t.Fatalf("generate scaffold: %v", err)
	}

	port, err := reserveTCPPort()
	if err != nil {
		t.Fatalf("reserve tcp port: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", append([]string{"run", "./tools/gwc", "dev"}, append(devArgsFromScaffold(result), "-host", "127.0.0.1", "-port", port)...)...)
	cmd.Dir = repoRoot
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatalf("start dev server: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		terminateProcessTree(cmd)
		waitErr := waitForCommandExit(cmd, 5*time.Second)
		if waitErr != nil && !strings.Contains(waitErr.Error(), "signal: killed") && !strings.Contains(waitErr.Error(), "exit status 1") {
			t.Logf("dev server shutdown cleanup note: %v", waitErr)
		}
	})

	rootURL := "http://127.0.0.1:" + port + "/"
	wasmURL := "http://127.0.0.1:" + port + "/" + scaffoldWASMOutputPath()

	rootBody := waitForHTTPBody(t, rootURL, 60*time.Second, func(resp *http.Response, body string) error {
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status %d", resp.StatusCode)
		}
		if !strings.Contains(body, selection.ProjectName) {
			return fmt.Errorf("html shell missing project name")
		}
		if !strings.Contains(body, "__GWC_LIVERELOAD_CONFIG") {
			return fmt.Errorf("html shell missing livereload config injection")
		}
		return nil
	})
	if !strings.Contains(rootBody, "wasm_exec.js") {
		t.Fatalf("expected served HTML to include wasm runtime bootstrap, got:\n%s", rootBody)
	}

	waitForHTTPBody(t, wasmURL, 60*time.Second, func(resp *http.Response, body string) error {
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status %d", resp.StatusCode)
		}
		if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "application/wasm") {
			return fmt.Errorf("unexpected content type %q", contentType)
		}
		if len(body) == 0 {
			return fmt.Errorf("empty wasm response")
		}
		return nil
	})
}

func newTestStartModel() startModel {
	model := startModel{
		presets: defaultStartPresets(),
		inputs:  newStartInputs(),
	}
	model.selection.Preset = model.presets[0]
	return model
}

func captureStdout() (func() (string, error), func(), error) {
	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	os.Stdout = writer

	readOutput := func() (string, error) {
		if err := writer.Close(); err != nil {
			return "", err
		}
		bytes, err := io.ReadAll(reader)
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}
	restore := func() {
		os.Stdout = originalStdout
		_ = writer.Close()
		_ = reader.Close()
	}
	return readOutput, restore, nil
}

func reserveTCPPort() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		return "", err
	}
	return port, nil
}

func waitForHTTPBody(t *testing.T, url string, timeout time.Duration, validate func(resp *http.Response, body string) error) string {
	t.Helper()

	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		bodyBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			time.Sleep(500 * time.Millisecond)
			continue
		}
		body := string(bodyBytes)
		if err := validate(resp, body); err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		return body
	}
	t.Fatalf("timed out waiting for %s: %v", url, lastErr)
	return ""
}

func terminateProcessTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", cmd.Process.Pid)).Run()
		return
	}
	_ = cmd.Process.Kill()
}

func waitForCommandExit(cmd *exec.Cmd, timeout time.Duration) error {
	if cmd == nil {
		return nil
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timed out waiting for process %d to exit", cmd.Process.Pid)
	}
}

func sequentialWordSelector(indices ...int) func(int) int {
	position := 0
	return func(limit int) int {
		if len(indices) == 0 {
			return 0
		}
		if position >= len(indices) {
			return indices[len(indices)-1]
		}
		selected := indices[position]
		position++
		return selected
	}
}
