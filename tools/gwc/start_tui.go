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
	enterpriseSections := normalizeStartEnterpriseSections(startEnterpriseScaffoldSections)
	inputs := newStartInputs()
	model := startModel{
		presets:            defaultStartPresets(),
		step:               startStepPreset,
		inputs:             inputs,
		enterpriseSections: enterpriseSections,
		enterpriseEnabled:  make([]bool, len(enterpriseSections)),
	}

	finalModel, err := startProgramRunner(model)
	if err != nil {
		return nil, err
	}
	final, ok := finalModel.(startModel)
	if !ok || !final.confirmed {
		return nil, nil
	}
	selection := final.currentSelection()
	return &selection, nil
}

func runStartPostTUI(selection startSelection, result *scaffoldResult, generationErr error) (*startPostResult, error) {
	model := startPostModel{
		selection: selection,
		result:    result,
		options:   []startPostChoice{startPostChoiceExit, startPostChoiceRunDev},
	}
	if generationErr != nil {
		model.errText = generationErr.Error()
	}

	finalModel, err := startProgramRunner(model)
	if err != nil {
		return nil, err
	}
	final, ok := finalModel.(startPostModel)
	if !ok || !final.confirmed || generationErr != nil {
		return nil, nil
	}
	return &startPostResult{RunDev: final.selectedChoice() == startPostChoiceRunDev}, nil
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
	nameInput := textinput.New()
	nameInput.Prompt = "> "
	nameInput.Placeholder = "my-app"
	nameInput.CharLimit = 80
	nameInput.Width = 48

	moduleInput := textinput.New()
	moduleInput.Prompt = "> "
	moduleInput.Placeholder = "github.com/your-org/my-app"
	moduleInput.CharLimit = 120
	moduleInput.Width = 48

	authorInput := textinput.New()
	authorInput.Prompt = "> "
	authorInput.Placeholder = "Your Name"
	authorInput.CharLimit = 120
	authorInput.Width = 48

	versionInput := textinput.New()
	versionInput.Prompt = "> "
	versionInput.Placeholder = "0.1.0"
	versionInput.CharLimit = 40
	versionInput.Width = 48

	descriptionInput := textinput.New()
	descriptionInput.Prompt = "> "
	descriptionInput.Placeholder = "Short project description"
	descriptionInput.CharLimit = 160
	descriptionInput.Width = 64

	nameInput.Focus()
	return []textinput.Model{nameInput, moduleInput, authorInput, versionInput, descriptionInput}
}

func (m startModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m startModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		return m, nil
	case tea.KeyMsg:
		switch typed.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
	}

	switch m.step {
	case startStepPreset:
		return m.updatePresetStep(msg)
	case startStepEnterprise:
		return m.updateEnterpriseStep(msg)
	case startStepProject:
		return m.updateProjectStep(msg)
	case startStepConfirm:
		return m.updateConfirmStep(msg)
	default:
		return m, nil
	}
}

func (m startPostModel) Init() tea.Cmd {
	return nil
}

func (m startPostModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.errText == "" && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.errText == "" && m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "esc":
			m.confirmed = true
			return m, tea.Quit
		case "enter":
			m.confirmed = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m startPostModel) View() string {
	if m.quitting {
		return "\n"
	}
	if m.errText != "" {
		return m.renderGenerationError()
	}
	return m.renderGenerationSuccess()
}

func (m startModel) updatePresetStep(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		switch typed.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.presets)-1 {
				m.cursor++
			}
		case "enter":
			preset := m.presets[m.cursor]
			m.selection.Preset = preset
			m.seedProjectDefaults(preset)
			if m.hasEnterpriseSections() {
				m.step = startStepEnterprise
			} else {
				m.step = startStepProject
			}
			m.errText = ""
			return m, textinput.Blink
		}
	}
	return m, nil
}

func (m startModel) updateEnterpriseStep(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			if m.hasEnterpriseSections() {
				m.step = startStepEnterprise
			} else {
				m.step = startStepPreset
			}
			m.errText = ""
			return m, nil
		case "up", "k":
			if m.enterpriseCursor > 0 {
				m.enterpriseCursor--
			}
			return m, nil
		case "down", "j":
			if m.enterpriseCursor < len(m.enterpriseSections)-1 {
				m.enterpriseCursor++
			}
			return m, nil
		case " ", "x", "enter":
			if key.String() == "enter" {
				m.selection.EnterpriseSections = append([]launcherPluginScaffoldSection(nil), m.enterpriseSections...)
				m.selection.EnabledEnterpriseSections = m.selectedEnterpriseSectionTitles()
				m.selection.EnterpriseFeatures = m.selectedEnterpriseFeatures()
				m.step = startStepProject
				return m, nil
			}
			if len(m.enterpriseEnabled) > 0 {
				m.enterpriseEnabled[m.enterpriseCursor] = !m.enterpriseEnabled[m.enterpriseCursor]
			}
			return m, nil
		}
	}
	return m, nil
}

func (m startModel) updateProjectStep(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.step = startStepPreset
			m.errText = ""
			return m, nil
		case "shift+tab", "up":
			m.focusInput(m.inputIndex - 1)
			return m, nil
		case "tab", "down":
			m.focusInput(m.inputIndex + 1)
			return m, nil
		case "enter":
			if m.inputIndex < len(m.inputs)-1 {
				m.focusInput(m.inputIndex + 1)
				return m, nil
			}
			selection, err := m.buildSelection()
			if err != nil {
				m.errText = err.Error()
				return m, nil
			}
			m.selection = selection
			m.step = startStepConfirm
			m.errText = ""
			return m, nil
		}
	}

	cmds := make([]tea.Cmd, 0, len(m.inputs))
	for i := range m.inputs {
		updated, cmd := m.inputs[i].Update(msg)
		m.inputs[i] = updated
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m startModel) updateConfirmStep(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "backspace":
			m.step = startStepProject
			m.errText = ""
			return m, nil
		case "enter":
			m.confirmed = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m startModel) View() string {
	if m.quitting {
		return "\n"
	}

	switch m.step {
	case startStepPreset:
		return m.renderPresetPicker()
	case startStepEnterprise:
		return m.renderEnterpriseSections()
	case startStepProject:
		return m.renderProjectForm()
	case startStepConfirm:
		return m.renderConfirmation()
	default:
		return "\n"
	}
}

func (m *startModel) seedProjectDefaults(preset startPreset) {
	defaultName := defaultProjectName(preset)
	m.inputs[0].SetValue(defaultName)
	m.inputs[1].SetValue(defaultModulePath(defaultName))
	m.inputs[2].SetValue(defaultAuthor())
	m.inputs[3].SetValue(defaultVersion())
	m.inputs[4].SetValue(defaultDescription(preset))
	m.focusInput(0)
}

func (m *startModel) focusInput(index int) {
	if len(m.inputs) == 0 {
		return
	}
	if index < 0 {
		index = len(m.inputs) - 1
	}
	if index >= len(m.inputs) {
		index = 0
	}
	for i := range m.inputs {
		if i == index {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	m.inputIndex = index
}

func (m startModel) currentSelection() startSelection {
	selection := m.selection
	if selection.ProjectMode == "" {
		selection.ProjectMode = startDefaultProjectMode
	}
	selection.ProjectName = strings.TrimSpace(m.inputs[0].Value())
	selection.ModulePath = strings.TrimSpace(m.inputs[1].Value())
	selection.Author = strings.TrimSpace(m.inputs[2].Value())
	selection.Version = strings.TrimSpace(m.inputs[3].Value())
	selection.Description = strings.TrimSpace(m.inputs[4].Value())
	selection.TargetDir = defaultTargetDir(selection.ProjectName)
	return selection
}

func (m startModel) buildSelection() (startSelection, error) {
	selection := m.currentSelection()
	if selection.Preset.Key == "" {
		return startSelection{}, fmt.Errorf("choose a preset first")
	}
	if selection.ProjectName == "" {
		return startSelection{}, fmt.Errorf("project name is required")
	}
	if selection.ModulePath == "" {
		return startSelection{}, fmt.Errorf("module path is required")
	}
	if selection.Author == "" {
		return startSelection{}, fmt.Errorf("author is required")
	}
	if selection.Version == "" {
		return startSelection{}, fmt.Errorf("version is required")
	}
	if selection.Description == "" {
		return startSelection{}, fmt.Errorf("description is required")
	}
	if err := validateProjectFolderName(selection.ProjectName); err != nil {
		return startSelection{}, err
	}
	if err := validateGeneratedTargetDir(selection.TargetDir); err != nil {
		return startSelection{}, err
	}
	return selection, nil
}

func (m startModel) renderPresetPicker() string {
	var lines []string
	lines = append(lines, "GWC Start")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Step 1 of %d: choose a starter preset.", m.totalSteps()))
	lines = append(lines, "Keep it small first; customize features next.")
	lines = append(lines, "")

	for index, preset := range m.presets {
		cursor := "  "
		if index == m.cursor {
			cursor = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s", cursor, preset.Name))
		lines = append(lines, fmt.Sprintf("   %s", preset.Summary))
	}

	lines = append(lines, "")
	lines = append(lines, "Keys: up/down or j/k to move, enter to continue, q to quit")
	return strings.Join(lines, "\n")
}

func (m startModel) renderEnterpriseSections() string {
	var lines []string
	lines = append(lines, "GWC Start")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Step 2 of %d: optional enterprise plugin sections.", m.totalSteps()))
	lines = append(lines, "These sections come from organization plugins and are separate from built-in preset features.")
	lines = append(lines, "")

	if len(m.enterpriseSections) == 0 {
		lines = append(lines, "No enterprise plugin sections were contributed.")
	} else {
		for index, section := range m.enterpriseSections {
			cursor := "  "
			if index == m.enterpriseCursor {
				cursor = "> "
			}
			marker := "[ ]"
			if index < len(m.enterpriseEnabled) && m.enterpriseEnabled[index] {
				marker = "[x]"
			}
			lines = append(lines, fmt.Sprintf("%s%s %s", cursor, marker, section.Title))
			if strings.TrimSpace(section.Summary) != "" {
				lines = append(lines, fmt.Sprintf("   %s", strings.TrimSpace(section.Summary)))
			}
			if len(section.Features) > 0 {
				lines = append(lines, fmt.Sprintf("   Features: %s", strings.Join(section.Features, ", ")))
			}
		}
	}

	lines = append(lines, "")
	lines = append(lines, "Keys: up/down or j/k to move, space/x to toggle, enter to continue, esc to go back, q to quit")
	return strings.Join(lines, "\n")
}

func (m startModel) renderProjectForm() string {
	preset := m.selection.Preset
	var lines []string
	lines = append(lines, "GWC Start")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Step %d of %d: project identity.", m.projectStepNumber(), m.totalSteps()))
	lines = append(lines, fmt.Sprintf("Preset: %s", preset.Name))
	lines = append(lines, "")
	lines = append(lines, "Project name")
	lines = append(lines, m.inputs[0].View())
	lines = append(lines, "")
	lines = append(lines, "Module path")
	lines = append(lines, m.inputs[1].View())
	lines = append(lines, "")
	lines = append(lines, "Author")
	lines = append(lines, m.inputs[2].View())
	lines = append(lines, "")
	lines = append(lines, "Version")
	lines = append(lines, m.inputs[3].View())
	lines = append(lines, "")
	lines = append(lines, "Description")
	lines = append(lines, m.inputs[4].View())
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Scaffold mode: %s", selectionProjectModeLabel(m.currentSelection().ProjectMode)))
	lines = append(lines, fmt.Sprintf("Scaffold folder: %s", defaultTargetDir(strings.TrimSpace(m.inputs[0].Value()))))
	if m.errText != "" {
		lines = append(lines, "")
		lines = append(lines, "Error: "+m.errText)
	}
	lines = append(lines, "")
	lines = append(lines, "Keys: tab/shift+tab to move, enter to continue, esc to go back, q to quit")
	return strings.Join(lines, "\n")
}

func (m startModel) renderConfirmation() string {
	selection := m.currentSelection()
	var lines []string
	lines = append(lines, "GWC Start")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Step %d of %d: confirm scaffold plan.", m.confirmStepNumber(), m.totalSteps()))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Preset:         %s", selection.Preset.Name))
	lines = append(lines, fmt.Sprintf("Project name:   %s", selection.ProjectName))
	lines = append(lines, fmt.Sprintf("Module path:    %s", selection.ModulePath))
	lines = append(lines, fmt.Sprintf("Author:         %s", selection.Author))
	lines = append(lines, fmt.Sprintf("Version:        %s", selection.Version))
	lines = append(lines, fmt.Sprintf("Mode:           %s", selectionProjectModeLabel(selection.ProjectMode)))
	lines = append(lines, fmt.Sprintf("Target dir:     %s", selection.TargetDir))
	lines = append(lines, fmt.Sprintf("Output:         %s", selectionProjectModeOutput(selection.ProjectMode)))
	lines = append(lines, "")
	lines = append(lines, selection.Description)
	lines = append(lines, "")
	lines = append(lines, "Included baseline features:")
	for _, feature := range selection.Preset.Features {
		lines = append(lines, fmt.Sprintf("  - %s", feature))
	}
	if len(selection.EnterpriseSections) > 0 {
		selectedSections := map[string]struct{}{}
		for _, title := range selection.EnabledEnterpriseSections {
			selectedSections[title] = struct{}{}
		}
		lines = append(lines, "")
		lines = append(lines, "Optional enterprise sections:")
		for _, section := range selection.EnterpriseSections {
			marker := "[ ]"
			if _, ok := selectedSections[section.Title]; ok {
				marker = "[x]"
			}
			lines = append(lines, fmt.Sprintf("  %s %s", marker, section.Title))
		}
		if len(selection.EnterpriseFeatures) > 0 {
			lines = append(lines, fmt.Sprintf("  Selected enterprise features: %s", strings.Join(selection.EnterpriseFeatures, ", ")))
		}
	}
	lines = append(lines, "")
	lines = append(lines, selectionProjectModeDescription(selection.ProjectMode))
	lines = append(lines, "")
	lines = append(lines, "Keys: enter to confirm, esc to go back, q to quit")
	return strings.Join(lines, "\n")
}

func (m startModel) hasEnterpriseSections() bool {
	return len(m.enterpriseSections) > 0
}

func (m startModel) totalSteps() int {
	if m.hasEnterpriseSections() {
		return 4
	}
	return 3
}

func (m startModel) projectStepNumber() int {
	if m.hasEnterpriseSections() {
		return 3
	}
	return 2
}

func (m startModel) confirmStepNumber() int {
	if m.hasEnterpriseSections() {
		return 4
	}
	return 3
}

func (m startModel) selectedEnterpriseSectionTitles() []string {
	selected := []string{}
	for index, section := range m.enterpriseSections {
		if index >= len(m.enterpriseEnabled) || !m.enterpriseEnabled[index] {
			continue
		}
		title := strings.TrimSpace(section.Title)
		if title == "" {
			continue
		}
		selected = append(selected, title)
	}
	return selected
}

func (m startModel) selectedEnterpriseFeatures() []string {
	features := []string{}
	for index, section := range m.enterpriseSections {
		if index >= len(m.enterpriseEnabled) || !m.enterpriseEnabled[index] {
			continue
		}
		features = append(features, section.Features...)
	}
	return normalizeScaffoldFeatureKeys(features)
}

func normalizeStartEnterpriseSections(sections []launcherPluginScaffoldSection) []launcherPluginScaffoldSection {
	normalized := []launcherPluginScaffoldSection{}
	seen := map[string]struct{}{}
	for _, section := range sections {
		title := strings.TrimSpace(section.Title)
		if title == "" {
			continue
		}
		key := strings.ToLower(title)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, launcherPluginScaffoldSection{
			Title:    title,
			Summary:  strings.TrimSpace(section.Summary),
			Features: normalizeScaffoldFeatureKeys(section.Features),
		})
	}
	return normalized
}

func (m startPostModel) renderGenerationSuccess() string {
	var lines []string
	lines = append(lines, "GWC Start")
	lines = append(lines, "")
	lines = append(lines, "Scaffold generated successfully.")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Preset:         %s", m.selection.Preset.Name))
	lines = append(lines, fmt.Sprintf("Project name:   %s", m.selection.ProjectName))
	lines = append(lines, fmt.Sprintf("Module path:    %s", m.selection.ModulePath))
	lines = append(lines, fmt.Sprintf("Author:         %s", m.selection.Author))
	lines = append(lines, fmt.Sprintf("Version:        %s", m.selection.Version))
	if m.result != nil {
		lines = append(lines, fmt.Sprintf("Target dir:     %s", m.result.TargetDir))
		lines = append(lines, fmt.Sprintf("App entry:      %s", m.result.AppPath))
		lines = append(lines, fmt.Sprintf("HTML shell:     %s", m.result.HTMLPath))
	}
	lines = append(lines, "")
	lines = append(lines, "Choose the next action:")
	for index, option := range m.options {
		cursor := "  "
		if index == m.cursor {
			cursor = "> "
		}
		lines = append(lines, cursor+m.optionLabel(option))
	}
	lines = append(lines, "")
	lines = append(lines, "Keys: up/down or j/k to move, enter to confirm, esc to exit")
	return strings.Join(lines, "\n")
}

func (m startPostModel) renderGenerationError() string {
	var lines []string
	lines = append(lines, "GWC Start")
	lines = append(lines, "")
	lines = append(lines, "Scaffold generation failed.")
	lines = append(lines, "")
	lines = append(lines, m.errText)
	lines = append(lines, "")
	lines = append(lines, "Press enter or esc to close.")
	return strings.Join(lines, "\n")
}

func (m startPostModel) optionLabel(choice startPostChoice) string {
	switch choice {
	case startPostChoiceRunDev:
		return "Run the generated scaffold in the dev server now"
	default:
		return "Exit and return to the shell"
	}
}

func (m startPostModel) selectedChoice() startPostChoice {
	if len(m.options) == 0 {
		return startPostChoiceExit
	}
	if m.cursor < 0 || m.cursor >= len(m.options) {
		return startPostChoiceExit
	}
	return m.options[m.cursor]
}

func defaultProjectName(preset startPreset) string {
	return projectNameWithWordSelector(preset, newRandomWordSelector())
}

func projectNameWithWordSelector(preset startPreset, selector func(int) int) string {
	base := strings.Trim(strings.ReplaceAll(strings.ToLower(strings.TrimSpace(preset.Key)), "_", "-"), "-")
	if base == "" {
		base = "app"
	}
	return fmt.Sprintf("%s-%s-%s", pickProjectWord(techProjectWords, selector), pickProjectWord(salesProjectWords, selector), base)
}

func pickProjectWord(words []string, selector func(int) int) string {
	if len(words) == 0 {
		return "app"
	}
	if selector == nil {
		return words[0]
	}
	index := selector(len(words))
	if index < 0 {
		index = -index
	}
	return words[index%len(words)]
}

func newRandomWordSelector() func(int) int {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return func(limit int) int {
		if limit <= 0 {
			return 0
		}
		return rng.Intn(limit)
	}
}

func defaultModulePath(projectName string) string {
	projectName = strings.TrimSpace(projectName)
	if projectName == "" {
		projectName = "my-app"
	}
	return fmt.Sprintf("github.com/your-org/%s", projectName)
}

func defaultAuthor() string {
	for _, key := range []string{"GIT_AUTHOR_NAME", "GITHUB_USER", "USERNAME", "USER"} {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return "Your Name"
}

func defaultVersion() string {
	return "0.1.0"
}

func defaultDescription(preset startPreset) string {
	if strings.TrimSpace(preset.Summary) != "" {
		return strings.TrimSpace(preset.Summary)
	}
	return "Starter app generated by gwc start"
}

func defaultTargetDir(projectName string) string {
	projectName = strings.TrimSpace(projectName)
	if projectName == "" {
		projectName = "my-app"
	}
	return filepath.Join(defaultGeneratedScaffoldRoot(), projectName)
}

func normalizeScaffoldProjectMode(value string) (scaffoldProjectMode, bool) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", string(scaffoldProjectModeStandalone):
		return scaffoldProjectModeStandalone, true
	case string(scaffoldProjectModeContributorLinked), "linked", "contributor":
		return scaffoldProjectModeContributorLinked, true
	default:
		return "", false
	}
}

func selectionProjectModeLabel(mode scaffoldProjectMode) string {
	switch mode {
	case scaffoldProjectModeContributorLinked:
		return "Contributor-linked"
	default:
		return "Standalone"
	}
}

func selectionProjectModeOutput(mode scaffoldProjectMode) string {
	switch mode {
	case scaffoldProjectModeContributorLinked:
		return "contributor-linked project that keeps a local replace to the current framework checkout"
	default:
		return fmt.Sprintf("standalone project under %s", defaultGeneratedScaffoldRoot())
	}
}

func selectionProjectModeDescription(mode scaffoldProjectMode) string {
	switch mode {
	case scaffoldProjectModeContributorLinked:
		return "This scaffold will keep a deliberate local source link back to the current framework checkout for contributor work."
	default:
		return "This scaffold will be generated as a user-owned project outside the framework repo by default."
	}
}

func selectionProjectOwnership(mode scaffoldProjectMode) string {
	switch mode {
	case scaffoldProjectModeContributorLinked:
		return "framework-coupled"
	default:
		return "standalone"
	}
}

func selectionFrameworkSourceMode(mode scaffoldProjectMode) string {
	switch mode {
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
	overrides, configPath, ok, err := loadLauncherOverridesForCurrentContext()
	if err == nil && ok {
		if overrideRoot, resolveErr := resolveLauncherOverrideValue(configPath, overrides.Paths.GeneratedProjectRoot); resolveErr == nil && strings.TrimSpace(overrideRoot) != "" {
			return overrideRoot
		}
	}
	homeDir, err := startUserHomeDir()
	if err == nil && strings.TrimSpace(homeDir) != "" {
		return defaultGeneratedScaffoldRootForOS(runtime.GOOS, homeDir, startPathExists)
	}
	workingDir, workingDirErr := os.Getwd()
	if workingDirErr != nil {
		return filepath.Join("gwc-projects")
	}
	return filepath.Join(workingDir, "gwc-projects")
}

func defaultGeneratedScaffoldRootForOS(goos string, homeDir string, pathExists func(string) bool) string {
	homeDir = strings.TrimSpace(homeDir)
	if homeDir == "" {
		return filepath.Join("gwc-projects")
	}
	documentsDir := filepath.Join(homeDir, "Documents")
	projectsDir := filepath.Join(homeDir, "Projects")
	switch goos {
	case "windows", "darwin":
		return documentsDir
	default:
		if pathExists != nil && pathExists(documentsDir) {
			return documentsDir
		}
		if pathExists != nil && pathExists(projectsDir) {
			return projectsDir
		}
		return homeDir
	}
}

func validateGeneratedTargetDir(targetDir string) error {
	resolvedTarget, err := filepath.Abs(strings.TrimSpace(targetDir))
	if err != nil {
		return fmt.Errorf("resolve target directory: %w", err)
	}
	volumeName := filepath.VolumeName(resolvedTarget)
	trimmedTarget := strings.TrimPrefix(resolvedTarget, volumeName)
	trimmedTarget = strings.TrimSpace(trimmedTarget)
	if trimmedTarget == "" || trimmedTarget == string(os.PathSeparator) {
		return fmt.Errorf("target directory must not be the filesystem root")
	}
	return nil
}

func validateProjectFolderName(projectName string) error {
	projectName = strings.TrimSpace(projectName)
	if projectName == "" {
		return fmt.Errorf("project name is required")
	}
	if projectName == "." || projectName == ".." {
		return fmt.Errorf("project name must produce a normal folder name")
	}
	if strings.ContainsAny(projectName, `<>:"/\\|?*`) {
		return fmt.Errorf("project name contains characters that cannot be used for the scaffold folder")
	}
	return nil
}
