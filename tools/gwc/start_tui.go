package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var techProjectWords = []string{
	"quantum",
	"signal",
	"vector",
	"pixel",
	"atlas",
	"circuit",
	"neon",
	"nova",
}

var salesProjectWords = []string{
	"boost",
	"pipeline",
	"revenue",
	"growth",
	"velocity",
	"funnel",
	"launch",
	"momentum",
}

type startPreset struct {
	Key         string
	Name        string
	Summary     string
	Description string
	Features    []string
}

type scaffoldProjectMode string

const (
	scaffoldProjectModeStandalone        scaffoldProjectMode = "standalone"
	scaffoldProjectModeContributorLinked scaffoldProjectMode = "contributor-linked"
)

type startSelection struct {
	Preset                    startPreset
	EnterpriseSections        []launcherPluginScaffoldSection
	EnabledEnterpriseSections []string
	EnterpriseFeatures        []string
	ProjectMode               scaffoldProjectMode
	ProjectName               string
	ModulePath                string
	Author                    string
	Version                   string
	Description               string
	TargetDir                 string
	InitGit                   bool
	SkipGoModTidy             bool
	SkipRuntimeAssets         bool
}

type startStep int

const (
	startStepPreset startStep = iota
	startStepEnterprise
	startStepProject
	startStepConfirm
)

type startModel struct {
	presets            []startPreset
	cursor             int
	step               startStep
	quitting           bool
	confirmed          bool
	width              int
	inputs             []textinput.Model
	inputIndex         int
	enterpriseSections []launcherPluginScaffoldSection
	enterpriseEnabled  []bool
	enterpriseCursor   int
	selection          startSelection
	errText            string
}

type startPostChoice int

const (
	startPostChoiceExit startPostChoice = iota
	startPostChoiceRunDev
)

type startPostResult struct {
	RunDev bool
}

type startPostModel struct {
	selection startSelection
	result    *scaffoldResult
	errText   string
	cursor    int
	quitting  bool
	confirmed bool
	options   []startPostChoice
}

var startProgramRunner = func(model tea.Model) (tea.Model, error) {
	return tea.NewProgram(model, tea.WithAltScreen()).Run()
}

var startEnterpriseScaffoldSections []launcherPluginScaffoldSection
var startDefaultProjectMode = scaffoldProjectModeStandalone

func runStartTUI() (*startSelection, error) {
	parseEnterpriseSections := normalizeStartEnterpriseSections(startEnterpriseScaffoldSections)
	parseInputs := newStartInputs()
	parseModel := startModel{
		presets:            defaultStartPresets(),
		step:               startStepPreset,
		inputs:             parseInputs,
		enterpriseSections: parseEnterpriseSections,
		enterpriseEnabled:  make([]bool, len(parseEnterpriseSections)),
	}

	parseFinalModel, parseErr := startProgramRunner(parseModel)
	if parseErr != nil {
		return nil, parseErr
	}
	parseFinal, parseOk := parseFinalModel.(startModel)
	if !parseOk || !parseFinal.confirmed {
		return nil, nil
	}
	parseSelection := parseFinal.currentSelection()
	return &parseSelection, nil
}

func runStartPostTUI(parseSelection startSelection, parseResult *scaffoldResult, parseGenerationErr error) (*startPostResult, error) {
	parseModel := startPostModel{
		selection: parseSelection,
		result:    parseResult,
		options:   []startPostChoice{startPostChoiceExit, startPostChoiceRunDev},
	}
	if parseGenerationErr != nil {
		parseModel.errText = parseGenerationErr.Error()
	}

	parseFinalModel, parseErr := startProgramRunner(parseModel)
	if parseErr != nil {
		return nil, parseErr
	}
	parseFinal, parseOk := parseFinalModel.(startPostModel)
	if !parseOk || !parseFinal.confirmed || parseGenerationErr != nil {
		return nil, nil
	}
	return &startPostResult{RunDev: parseFinal.selectedChoice() == startPostChoiceRunDev}, nil
}

func defaultStartPresets() []startPreset {
	return []startPreset{
		{
			Key:         "minimal-client",
			Name:        "Minimal Client App",
			Summary:     "Smallest browser-rendered starter with a clean wasm mount path.",
			Description: "Start here when you want a tiny app you can understand and replace quickly.",
			Features:    []string{"ui", "html", "dev-profile", "browser-mount"},
		},
		{
			Key:         "routed-spa",
			Name:        "Routed SPA",
			Summary:     "Client-rendered app with router wiring and a realistic page shell.",
			Description: "Use this when you already know you need routes, navigation, and app-level structure.",
			Features:    []string{"ui", "html", "router", "dev-profile", "browser-tests"},
		},
		{
			Key:         "ssr-app",
			Name:        "SSR App",
			Summary:     "Server-rendered starter with hydration entrypoints and bootstrap structure.",
			Description: "Choose this when request-time HTML and hydration are part of the product from day one.",
			Features:    []string{"ui", "html", "router", "ssr", "hydration", "release-profile"},
		},
		{
			Key:         "reference-app",
			Name:        "Reference App",
			Summary:     "Larger baseline with routing, forms, async data, and production-minded defaults.",
			Description: "Use this when you want a more complete app shell that still stays understandable and disposable.",
			Features:    []string{"ui", "html", "router", "forms", "fetch", "state", "browser-tests", "release-profile"},
		},
	}
}

func newStartInputs() []textinput.Model {
	parseNameInput := textinput.New()
	parseNameInput.Prompt = "> "
	parseNameInput.Placeholder = "my-app"
	parseNameInput.CharLimit = 80
	parseNameInput.Width = 48

	parseModuleInput := textinput.New()
	parseModuleInput.Prompt = "> "
	parseModuleInput.Placeholder = "github.com/your-org/my-app"
	parseModuleInput.CharLimit = 120
	parseModuleInput.Width = 48

	parseAuthorInput := textinput.New()
	parseAuthorInput.Prompt = "> "
	parseAuthorInput.Placeholder = "Your Name"
	parseAuthorInput.CharLimit = 120
	parseAuthorInput.Width = 48

	parseVersionInput := textinput.New()
	parseVersionInput.Prompt = "> "
	parseVersionInput.Placeholder = "0.1.0"
	parseVersionInput.CharLimit = 40
	parseVersionInput.Width = 48

	parseDescriptionInput := textinput.New()
	parseDescriptionInput.Prompt = "> "
	parseDescriptionInput.Placeholder = "Short project description"
	parseDescriptionInput.CharLimit = 160
	parseDescriptionInput.Width = 64

	parseNameInput.Focus()
	return []textinput.Model{parseNameInput, parseModuleInput, parseAuthorInput, parseVersionInput, parseDescriptionInput}
}

func (parseM startModel) Init() tea.Cmd {
	return textinput.Blink
}

func (parseM startModel) Update(parseMsg tea.Msg) (tea.Model, tea.Cmd) {
	switch parseTyped := parseMsg.(type) {
	case tea.WindowSizeMsg:
		parseM.width = parseTyped.Width
		return parseM, nil
	case tea.KeyMsg:
		switch parseTyped.String() {
		case "ctrl+c", "q":
			parseM.quitting = true
			return parseM, tea.Quit
		}
	}

	switch parseM.step {
	case startStepPreset:
		return parseM.updatePresetStep(parseMsg)
	case startStepEnterprise:
		return parseM.updateEnterpriseStep(parseMsg)
	case startStepProject:
		return parseM.updateProjectStep(parseMsg)
	case startStepConfirm:
		return parseM.updateConfirmStep(parseMsg)
	default:
		return parseM, nil
	}
}

func (parseM startPostModel) Init() tea.Cmd {
	return nil
}

func (parseM startPostModel) Update(parseMsg tea.Msg) (tea.Model, tea.Cmd) {
	if parseKey, parseOk := parseMsg.(tea.KeyMsg); parseOk {
		switch parseKey.String() {
		case "ctrl+c", "q":
			parseM.quitting = true
			return parseM, tea.Quit
		case "up", "k":
			if parseM.errText == "" && parseM.cursor > 0 {
				parseM.cursor--
			}
		case "down", "j":
			if parseM.errText == "" && parseM.cursor < len(parseM.options)-1 {
				parseM.cursor++
			}
		case "esc":
			parseM.confirmed = true
			return parseM, tea.Quit
		case "enter":
			parseM.confirmed = true
			return parseM, tea.Quit
		}
	}
	return parseM, nil
}

func (parseM startPostModel) View() string {
	if parseM.quitting {
		return "\n"
	}
	if parseM.errText != "" {
		return parseM.renderGenerationError()
	}
	return parseM.renderGenerationSuccess()
}

func (parseM startModel) updatePresetStep(parseMsg tea.Msg) (tea.Model, tea.Cmd) {
	switch parseTyped := parseMsg.(type) {
	case tea.KeyMsg:
		switch parseTyped.String() {
		case "up", "k":
			if parseM.cursor > 0 {
				parseM.cursor--
			}
		case "down", "j":
			if parseM.cursor < len(parseM.presets)-1 {
				parseM.cursor++
			}
		case "enter":
			parsePreset := parseM.presets[parseM.cursor]
			parseM.selection.Preset = parsePreset
			parseM.seedProjectDefaults(parsePreset)
			if parseM.hasEnterpriseSections() {
				parseM.step = startStepEnterprise
			} else {
				parseM.step = startStepProject
			}
			parseM.errText = ""
			return parseM, textinput.Blink
		}
	}
	return parseM, nil
}

func (parseM startModel) updateEnterpriseStep(parseMsg tea.Msg) (tea.Model, tea.Cmd) {
	if parseKey, parseOk := parseMsg.(tea.KeyMsg); parseOk {
		switch parseKey.String() {
		case "esc":
			if parseM.hasEnterpriseSections() {
				parseM.step = startStepEnterprise
			} else {
				parseM.step = startStepPreset
			}
			parseM.errText = ""
			return parseM, nil
		case "up", "k":
			if parseM.enterpriseCursor > 0 {
				parseM.enterpriseCursor--
			}
			return parseM, nil
		case "down", "j":
			if parseM.enterpriseCursor < len(parseM.enterpriseSections)-1 {
				parseM.enterpriseCursor++
			}
			return parseM, nil
		case " ", "x", "enter":
			if parseKey.String() == "enter" {
				parseM.selection.EnterpriseSections = append([]launcherPluginScaffoldSection(nil), parseM.enterpriseSections...)
				parseM.selection.EnabledEnterpriseSections = parseM.selectedEnterpriseSectionTitles()
				parseM.selection.EnterpriseFeatures = parseM.selectedEnterpriseFeatures()
				parseM.step = startStepProject
				return parseM, nil
			}
			if len(parseM.enterpriseEnabled) > 0 {
				parseM.enterpriseEnabled[parseM.enterpriseCursor] = !parseM.enterpriseEnabled[parseM.enterpriseCursor]
			}
			return parseM, nil
		}
	}
	return parseM, nil
}

func (parseM startModel) updateProjectStep(parseMsg tea.Msg) (tea.Model, tea.Cmd) {
	if parseKey, parseOk := parseMsg.(tea.KeyMsg); parseOk {
		switch parseKey.String() {
		case "esc":
			parseM.step = startStepPreset
			parseM.errText = ""
			return parseM, nil
		case "shift+tab", "up":
			parseM.focusInput(parseM.inputIndex - 1)
			return parseM, nil
		case "tab", "down":
			parseM.focusInput(parseM.inputIndex + 1)
			return parseM, nil
		case "enter":
			if parseM.inputIndex < len(parseM.inputs)-1 {
				parseM.focusInput(parseM.inputIndex + 1)
				return parseM, nil
			}
			parseSelection, parseErr := parseM.buildSelection()
			if parseErr != nil {
				parseM.errText = parseErr.Error()
				return parseM, nil
			}
			parseM.selection = parseSelection
			parseM.step = startStepConfirm
			parseM.errText = ""
			return parseM, nil
		}
	}

	parseCmds := make([]tea.Cmd, 0, len(parseM.inputs))
	for parseI := range parseM.inputs {
		parseUpdated, parseCmd := parseM.inputs[parseI].Update(parseMsg)
		parseM.inputs[parseI] = parseUpdated
		parseCmds = append(parseCmds, parseCmd)
	}
	return parseM, tea.Batch(parseCmds...)
}

func (parseM startModel) updateConfirmStep(parseMsg tea.Msg) (tea.Model, tea.Cmd) {
	if parseKey, parseOk := parseMsg.(tea.KeyMsg); parseOk {
		switch parseKey.String() {
		case "esc", "backspace":
			parseM.step = startStepProject
			parseM.errText = ""
			return parseM, nil
		case "enter":
			parseM.confirmed = true
			return parseM, tea.Quit
		}
	}
	return parseM, nil
}

func (parseM startModel) View() string {
	if parseM.quitting {
		return "\n"
	}

	switch parseM.step {
	case startStepPreset:
		return parseM.renderPresetPicker()
	case startStepEnterprise:
		return parseM.renderEnterpriseSections()
	case startStepProject:
		return parseM.renderProjectForm()
	case startStepConfirm:
		return parseM.renderConfirmation()
	default:
		return "\n"
	}
}

func (parseM *startModel) seedProjectDefaults(parsePreset startPreset) {
	parseDefaultName := defaultProjectName(parsePreset)
	parseM.inputs[0].SetValue(parseDefaultName)
	parseM.inputs[1].SetValue(defaultModulePath(parseDefaultName))
	parseM.inputs[2].SetValue(defaultAuthor())
	parseM.inputs[3].SetValue(defaultVersion())
	parseM.inputs[4].SetValue(defaultDescription(parsePreset))
	parseM.focusInput(0)
}

func (parseM *startModel) focusInput(parseIndex int) {
	if len(parseM.inputs) == 0 {
		return
	}
	if parseIndex < 0 {
		parseIndex = len(parseM.inputs) - 1
	}
	if parseIndex >= len(parseM.inputs) {
		parseIndex = 0
	}
	for parseI := range parseM.inputs {
		if parseI == parseIndex {
			parseM.inputs[parseI].Focus()
		} else {
			parseM.inputs[parseI].Blur()
		}
	}
	parseM.inputIndex = parseIndex
}

func (parseM startModel) currentSelection() startSelection {
	parseSelection := parseM.selection
	if parseSelection.ProjectMode == "" {
		parseSelection.ProjectMode = startDefaultProjectMode
	}
	parseSelection.ProjectName = strings.TrimSpace(parseM.inputs[0].Value())
	parseSelection.ModulePath = strings.TrimSpace(parseM.inputs[1].Value())
	parseSelection.Author = strings.TrimSpace(parseM.inputs[2].Value())
	parseSelection.Version = strings.TrimSpace(parseM.inputs[3].Value())
	parseSelection.Description = strings.TrimSpace(parseM.inputs[4].Value())
	parseSelection.TargetDir = defaultTargetDir(parseSelection.ProjectName)
	return parseSelection
}

func (parseM startModel) buildSelection() (startSelection, error) {
	parseSelection := parseM.currentSelection()
	if parseSelection.Preset.Key == "" {
		return startSelection{}, fmt.Errorf("choose a preset first")
	}
	if parseSelection.ProjectName == "" {
		return startSelection{}, fmt.Errorf("project name is required")
	}
	if parseSelection.ModulePath == "" {
		return startSelection{}, fmt.Errorf("module path is required")
	}
	if parseSelection.Author == "" {
		return startSelection{}, fmt.Errorf("author is required")
	}
	if parseSelection.Version == "" {
		return startSelection{}, fmt.Errorf("version is required")
	}
	if parseSelection.Description == "" {
		return startSelection{}, fmt.Errorf("description is required")
	}
	if parseErr := validateProjectFolderName(parseSelection.ProjectName); parseErr != nil {
		return startSelection{}, parseErr
	}
	if parseErr2 := validateGeneratedTargetDir(parseSelection.TargetDir); parseErr2 != nil {
		return startSelection{}, parseErr2
	}
	return parseSelection, nil
}

func (parseM startModel) renderPresetPicker() string {
	var parseLines []string
	parseLines = append(parseLines, "GWC Start")
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, fmt.Sprintf("Step 1 of %d: choose a starter preset.", parseM.totalSteps()))
	parseLines = append(parseLines, "Keep it small first; customize features next.")
	parseLines = append(parseLines, "")

	for parseIndex, parsePreset := range parseM.presets {
		parseCursor := "  "
		if parseIndex == parseM.cursor {
			parseCursor = "> "
		}
		parseLines = append(parseLines, fmt.Sprintf("%s%s", parseCursor, parsePreset.Name))
		parseLines = append(parseLines, fmt.Sprintf("   %s", parsePreset.Summary))
	}

	parseLines = append(parseLines, "")
	parseLines = append(parseLines, "Keys: up/down or j/k to move, enter to continue, q to quit")
	return strings.Join(parseLines, "\n")
}

func (parseM startModel) renderEnterpriseSections() string {
	var parseLines []string
	parseLines = append(parseLines, "GWC Start")
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, fmt.Sprintf("Step 2 of %d: optional enterprise plugin sections.", parseM.totalSteps()))
	parseLines = append(parseLines, "These sections come from organization plugins and are separate from built-in preset features.")
	parseLines = append(parseLines, "")

	if len(parseM.enterpriseSections) == 0 {
		parseLines = append(parseLines, "No enterprise plugin sections were contributed.")
	} else {
		for parseIndex, parseSection := range parseM.enterpriseSections {
			parseCursor := "  "
			if parseIndex == parseM.enterpriseCursor {
				parseCursor = "> "
			}
			parseMarker := "[ ]"
			if parseIndex < len(parseM.enterpriseEnabled) && parseM.enterpriseEnabled[parseIndex] {
				parseMarker = "[x]"
			}
			parseLines = append(parseLines, fmt.Sprintf("%s%s %s", parseCursor, parseMarker, parseSection.Title))
			if strings.TrimSpace(parseSection.Summary) != "" {
				parseLines = append(parseLines, fmt.Sprintf("   %s", strings.TrimSpace(parseSection.Summary)))
			}
			if len(parseSection.Features) > 0 {
				parseLines = append(parseLines, fmt.Sprintf("   Features: %s", strings.Join(parseSection.Features, ", ")))
			}
		}
	}

	parseLines = append(parseLines, "")
	parseLines = append(parseLines, "Keys: up/down or j/k to move, space/x to toggle, enter to continue, esc to go back, q to quit")
	return strings.Join(parseLines, "\n")
}

func (parseM startModel) renderProjectForm() string {
	parsePreset := parseM.selection.Preset
	var parseLines []string
	parseLines = append(parseLines, "GWC Start")
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, fmt.Sprintf("Step %d of %d: project identity.", parseM.projectStepNumber(), parseM.totalSteps()))
	parseLines = append(parseLines, fmt.Sprintf("Preset: %s", parsePreset.Name))
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, "Project name")
	parseLines = append(parseLines, parseM.inputs[0].View())
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, "Module path")
	parseLines = append(parseLines, parseM.inputs[1].View())
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, "Author")
	parseLines = append(parseLines, parseM.inputs[2].View())
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, "Version")
	parseLines = append(parseLines, parseM.inputs[3].View())
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, "Description")
	parseLines = append(parseLines, parseM.inputs[4].View())
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, fmt.Sprintf("Scaffold mode: %s", selectionProjectModeLabel(parseM.currentSelection().ProjectMode)))
	parseLines = append(parseLines, fmt.Sprintf("Scaffold folder: %s", defaultTargetDir(strings.TrimSpace(parseM.inputs[0].Value()))))
	if parseM.errText != "" {
		parseLines = append(parseLines, "")
		parseLines = append(parseLines, "Error: "+parseM.errText)
	}
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, "Keys: tab/shift+tab to move, enter to continue, esc to go back, q to quit")
	return strings.Join(parseLines, "\n")
}

func (parseM startModel) renderConfirmation() string {
	parseSelection := parseM.currentSelection()
	var parseLines []string
	parseLines = append(parseLines, "GWC Start")
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, fmt.Sprintf("Step %d of %d: confirm scaffold plan.", parseM.confirmStepNumber(), parseM.totalSteps()))
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, fmt.Sprintf("Preset:         %s", parseSelection.Preset.Name))
	parseLines = append(parseLines, fmt.Sprintf("Project name:   %s", parseSelection.ProjectName))
	parseLines = append(parseLines, fmt.Sprintf("Module path:    %s", parseSelection.ModulePath))
	parseLines = append(parseLines, fmt.Sprintf("Author:         %s", parseSelection.Author))
	parseLines = append(parseLines, fmt.Sprintf("Version:        %s", parseSelection.Version))
	parseLines = append(parseLines, fmt.Sprintf("Mode:           %s", selectionProjectModeLabel(parseSelection.ProjectMode)))
	parseLines = append(parseLines, fmt.Sprintf("Target dir:     %s", parseSelection.TargetDir))
	parseLines = append(parseLines, fmt.Sprintf("Output:         %s", selectionProjectModeOutput(parseSelection.ProjectMode)))
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, parseSelection.Description)
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, "Included baseline features:")
	for _, parseFeature := range parseSelection.Preset.Features {
		parseLines = append(parseLines, fmt.Sprintf("  - %s", parseFeature))
	}
	if len(parseSelection.EnterpriseSections) > 0 {
		parseSelectedSections := map[string]struct{}{}
		for _, parseTitle := range parseSelection.EnabledEnterpriseSections {
			parseSelectedSections[parseTitle] = struct{}{}
		}
		parseLines = append(parseLines, "")
		parseLines = append(parseLines, "Optional enterprise sections:")
		for _, parseSection := range parseSelection.EnterpriseSections {
			parseMarker := "[ ]"
			if _, parseOk := parseSelectedSections[parseSection.Title]; parseOk {
				parseMarker = "[x]"
			}
			parseLines = append(parseLines, fmt.Sprintf("  %s %s", parseMarker, parseSection.Title))
		}
		if len(parseSelection.EnterpriseFeatures) > 0 {
			parseLines = append(parseLines, fmt.Sprintf("  Selected enterprise features: %s", strings.Join(parseSelection.EnterpriseFeatures, ", ")))
		}
	}
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, selectionProjectModeDescription(parseSelection.ProjectMode))
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, "Keys: enter to confirm, esc to go back, q to quit")
	return strings.Join(parseLines, "\n")
}

func (parseM startModel) hasEnterpriseSections() bool {
	return len(parseM.enterpriseSections) > 0
}

func (parseM startModel) totalSteps() int {
	if parseM.hasEnterpriseSections() {
		return 4
	}
	return 3
}

func (parseM startModel) projectStepNumber() int {
	if parseM.hasEnterpriseSections() {
		return 3
	}
	return 2
}

func (parseM startModel) confirmStepNumber() int {
	if parseM.hasEnterpriseSections() {
		return 4
	}
	return 3
}

func (parseM startModel) selectedEnterpriseSectionTitles() []string {
	parseSelected := []string{}
	for parseIndex, parseSection := range parseM.enterpriseSections {
		if parseIndex >= len(parseM.enterpriseEnabled) || !parseM.enterpriseEnabled[parseIndex] {
			continue
		}
		parseTitle := strings.TrimSpace(parseSection.Title)
		if parseTitle == "" {
			continue
		}
		parseSelected = append(parseSelected, parseTitle)
	}
	return parseSelected
}

func (parseM startModel) selectedEnterpriseFeatures() []string {
	parseFeatures := []string{}
	for parseIndex, parseSection := range parseM.enterpriseSections {
		if parseIndex >= len(parseM.enterpriseEnabled) || !parseM.enterpriseEnabled[parseIndex] {
			continue
		}
		parseFeatures = append(parseFeatures, parseSection.Features...)
	}
	return normalizeScaffoldFeatureKeys(parseFeatures)
}

func normalizeStartEnterpriseSections(parseSections []launcherPluginScaffoldSection) []launcherPluginScaffoldSection {
	parseNormalized := []launcherPluginScaffoldSection{}
	parseSeen := map[string]struct{}{}
	for _, parseSection := range parseSections {
		parseTitle := strings.TrimSpace(parseSection.Title)
		if parseTitle == "" {
			continue
		}
		parseKey := strings.ToLower(parseTitle)
		if _, parseExists := parseSeen[parseKey]; parseExists {
			continue
		}
		parseSeen[parseKey] = struct{}{}
		parseNormalized = append(parseNormalized, launcherPluginScaffoldSection{
			Title:    parseTitle,
			Summary:  strings.TrimSpace(parseSection.Summary),
			Features: normalizeScaffoldFeatureKeys(parseSection.Features),
		})
	}
	return parseNormalized
}

func (parseM startPostModel) renderGenerationSuccess() string {
	var parseLines []string
	parseLines = append(parseLines, "GWC Start")
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, "Scaffold generated successfully.")
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, fmt.Sprintf("Preset:         %s", parseM.selection.Preset.Name))
	parseLines = append(parseLines, fmt.Sprintf("Project name:   %s", parseM.selection.ProjectName))
	parseLines = append(parseLines, fmt.Sprintf("Module path:    %s", parseM.selection.ModulePath))
	parseLines = append(parseLines, fmt.Sprintf("Author:         %s", parseM.selection.Author))
	parseLines = append(parseLines, fmt.Sprintf("Version:        %s", parseM.selection.Version))
	if parseM.result != nil {
		parseLines = append(parseLines, fmt.Sprintf("Target dir:     %s", parseM.result.TargetDir))
		parseLines = append(parseLines, fmt.Sprintf("App entry:      %s", parseM.result.AppPath))
		parseLines = append(parseLines, fmt.Sprintf("HTML shell:     %s", parseM.result.HTMLPath))
	}
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, "Choose the next action:")
	for parseIndex, parseOption := range parseM.options {
		parseCursor := "  "
		if parseIndex == parseM.cursor {
			parseCursor = "> "
		}
		parseLines = append(parseLines, parseCursor+parseM.optionLabel(parseOption))
	}
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, "Keys: up/down or j/k to move, enter to confirm, esc to exit")
	return strings.Join(parseLines, "\n")
}

func (parseM startPostModel) renderGenerationError() string {
	var parseLines []string
	parseLines = append(parseLines, "GWC Start")
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, "Scaffold generation failed.")
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, parseM.errText)
	parseLines = append(parseLines, "")
	parseLines = append(parseLines, "Press enter or esc to close.")
	return strings.Join(parseLines, "\n")
}

func (parseM startPostModel) optionLabel(parseChoice startPostChoice) string {
	switch parseChoice {
	case startPostChoiceRunDev:
		return "Run the generated scaffold in the dev server now"
	default:
		return "Exit and return to the shell"
	}
}

func (parseM startPostModel) selectedChoice() startPostChoice {
	if len(parseM.options) == 0 {
		return startPostChoiceExit
	}
	if parseM.cursor < 0 || parseM.cursor >= len(parseM.options) {
		return startPostChoiceExit
	}
	return parseM.options[parseM.cursor]
}

func defaultProjectName(parsePreset startPreset) string {
	return projectNameWithWordSelector(parsePreset, newRandomWordSelector())
}

func projectNameWithWordSelector(parsePreset startPreset, parseSelector func(int) int) string {
	parseBase := strings.Trim(strings.ReplaceAll(strings.ToLower(strings.TrimSpace(parsePreset.Key)), "_", "-"), "-")
	if parseBase == "" {
		parseBase = "app"
	}
	return fmt.Sprintf("%s-%s-%s", pickProjectWord(techProjectWords, parseSelector), pickProjectWord(salesProjectWords, parseSelector), parseBase)
}

func pickProjectWord(parseWords []string, parseSelector func(int) int) string {
	if len(parseWords) == 0 {
		return "app"
	}
	if parseSelector == nil {
		return parseWords[0]
	}
	parseIndex := parseSelector(len(parseWords))
	if parseIndex < 0 {
		parseIndex = -parseIndex
	}
	return parseWords[parseIndex%len(parseWords)]
}

func newRandomWordSelector() func(int) int {
	parseRng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return func(parseLimit int) int {
		if parseLimit <= 0 {
			return 0
		}
		return parseRng.Intn(parseLimit)
	}
}

func defaultModulePath(parseProjectName string) string {
	parseProjectName = strings.TrimSpace(parseProjectName)
	if parseProjectName == "" {
		parseProjectName = "my-app"
	}
	return fmt.Sprintf("github.com/your-org/%s", parseProjectName)
}

func defaultAuthor() string {
	for _, parseKey := range []string{"GIT_AUTHOR_NAME", "GITHUB_USER", "USERNAME", "USER"} {
		parseValue := strings.TrimSpace(os.Getenv(parseKey))
		if parseValue != "" {
			return parseValue
		}
	}
	return "Your Name"
}

func defaultVersion() string {
	return "0.1.0"
}

func defaultDescription(parsePreset startPreset) string {
	if strings.TrimSpace(parsePreset.Summary) != "" {
		return strings.TrimSpace(parsePreset.Summary)
	}
	return "Starter app generated by gwc start"
}

func defaultTargetDir(parseProjectName string) string {
	parseProjectName = strings.TrimSpace(parseProjectName)
	if parseProjectName == "" {
		parseProjectName = "my-app"
	}
	return filepath.Join(defaultGeneratedScaffoldRoot(), parseProjectName)
}

func normalizeScaffoldProjectMode(parseValue string) (scaffoldProjectMode, bool) {
	switch strings.TrimSpace(strings.ToLower(parseValue)) {
	case "", string(scaffoldProjectModeStandalone):
		return scaffoldProjectModeStandalone, true
	case string(scaffoldProjectModeContributorLinked), "linked", "contributor":
		return scaffoldProjectModeContributorLinked, true
	default:
		return "", false
	}
}

func selectionProjectModeLabel(parseMode scaffoldProjectMode) string {
	switch parseMode {
	case scaffoldProjectModeContributorLinked:
		return "Contributor-linked"
	default:
		return "Standalone"
	}
}

func selectionProjectModeOutput(parseMode scaffoldProjectMode) string {
	switch parseMode {
	case scaffoldProjectModeContributorLinked:
		return "contributor-linked project that keeps a local replace to the current framework checkout"
	default:
		return fmt.Sprintf("standalone project under %s", defaultGeneratedScaffoldRoot())
	}
}

func selectionProjectModeDescription(parseMode scaffoldProjectMode) string {
	switch parseMode {
	case scaffoldProjectModeContributorLinked:
		return "This scaffold will keep a deliberate local source link back to the current framework checkout for contributor work."
	default:
		return "This scaffold will be generated as a user-owned project outside the framework repo by default."
	}
}

func selectionProjectOwnership(parseMode scaffoldProjectMode) string {
	switch parseMode {
	case scaffoldProjectModeContributorLinked:
		return "framework-coupled"
	default:
		return "standalone"
	}
}

func selectionFrameworkSourceMode(parseMode scaffoldProjectMode) string {
	switch parseMode {
	case scaffoldProjectModeContributorLinked:
		return "local-replace"
	default:
		return "module-proxy"
	}
}

var startUserHomeDir = os.UserHomeDir

var startPathExists = func(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func defaultGeneratedScaffoldRoot() string {
	parseOverrides, parseConfigPath, parseOk, parseErr := loadLauncherOverridesForCurrentContext()
	if parseErr == nil && parseOk {
		if parseOverrideRoot, parseResolveErr := resolveLauncherOverrideValue(parseConfigPath, parseOverrides.Paths.GeneratedProjectRoot); parseResolveErr == nil && strings.TrimSpace(parseOverrideRoot) != "" {
			return parseOverrideRoot
		}
	}
	parseHomeDir, parseErr := startUserHomeDir()
	if parseErr == nil && strings.TrimSpace(parseHomeDir) != "" {
		return defaultGeneratedScaffoldRootForOS(runtime.GOOS, parseHomeDir, startPathExists)
	}
	parseWorkingDir, parseWorkingDirErr := os.Getwd()
	if parseWorkingDirErr != nil {
		return filepath.Join("gwc-projects")
	}
	return filepath.Join(parseWorkingDir, "gwc-projects")
}

func defaultGeneratedScaffoldRootForOS(parseGoos string, parseHomeDir string, parsePathExists func(string) bool) string {
	parseHomeDir = strings.TrimSpace(parseHomeDir)
	if parseHomeDir == "" {
		return filepath.Join("gwc-projects")
	}
	parseDocumentsDir := filepath.Join(parseHomeDir, "Documents")
	parseProjectsDir := filepath.Join(parseHomeDir, "Projects")
	switch parseGoos {
	case "windows", "darwin":
		return parseDocumentsDir
	default:
		if parsePathExists != nil && parsePathExists(parseDocumentsDir) {
			return parseDocumentsDir
		}
		if parsePathExists != nil && parsePathExists(parseProjectsDir) {
			return parseProjectsDir
		}
		return parseHomeDir
	}
}

func validateGeneratedTargetDir(parseTargetDir string) error {
	parseResolvedTarget, parseErr := filepath.Abs(strings.TrimSpace(parseTargetDir))
	if parseErr != nil {
		return fmt.Errorf("resolve target directory: %w", parseErr)
	}
	parseVolumeName := filepath.VolumeName(parseResolvedTarget)
	parseTrimmedTarget := strings.TrimPrefix(parseResolvedTarget, parseVolumeName)
	parseTrimmedTarget = strings.TrimSpace(parseTrimmedTarget)
	if parseTrimmedTarget == "" || parseTrimmedTarget == string(os.PathSeparator) {
		return fmt.Errorf("target directory must not be the filesystem root")
	}
	return nil
}

func validateProjectFolderName(parseProjectName string) error {
	parseProjectName = strings.TrimSpace(parseProjectName)
	if parseProjectName == "" {
		return fmt.Errorf("project name is required")
	}
	if parseProjectName == "." || parseProjectName == ".." {
		return fmt.Errorf("project name must produce a normal folder name")
	}
	if strings.ContainsAny(parseProjectName, `<>:"/\\|?*`) {
		return fmt.Errorf("project name contains characters that cannot be used for the scaffold folder")
	}
	return nil
}
