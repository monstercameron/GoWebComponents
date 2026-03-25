package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type dashboardConfig struct {
	projectRoot   string
	statusURL     string
	disconnectURL string
	json          bool
}

type dashboardSnapshot struct {
	GeneratedAt string                    `json:"generatedAt"`
	ProjectRoot string                    `json:"projectRoot"`
	StatusURL   string                    `json:"statusURL"`
	Status      *dashboardStatusPayload   `json:"status,omitempty"`
	StatusError string                    `json:"statusError,omitempty"`
	Providers   []dashboardProviderStatus `json:"providers,omitempty"`
}

type dashboardStatusPayload struct {
	Mode              string                   `json:"mode"`
	ListeningURL      string                   `json:"listeningURL"`
	StatusURL         string                   `json:"statusURL"`
	WebSocketURL      string                   `json:"websocketURL"`
	ProjectRoot       string                   `json:"projectRoot"`
	ServedWASMPath    string                   `json:"servedWasmPath"`
	HotReloadEnabled  bool                     `json:"hotReloadEnabled"`
	HotReloadEligible bool                     `json:"hotReloadEligible"`
	ClientCount       int                      `json:"clientCount"`
	Clients           []dashboardClientSession `json:"clients,omitempty"`
}

type dashboardClientSession struct {
	ID          string    `json:"id"`
	RemoteAddr  string    `json:"remoteAddr,omitempty"`
	UserAgent   string    `json:"userAgent,omitempty"`
	ConnectedAt time.Time `json:"connectedAt"`
	LastSeenAt  time.Time `json:"lastSeenAt"`
}

type dashboardProviderStatus struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	APIKeyEnv       string   `json:"apiKeyEnv,omitempty"`
	AuthConfigured  bool     `json:"authConfigured"`
	AuthSource      string   `json:"authSource,omitempty"`
	BaseURL         string   `json:"baseURL,omitempty"`
	BaseURLSource   string   `json:"baseURLSource,omitempty"`
	DefaultModel    string   `json:"defaultModel,omitempty"`
	ModelSource     string   `json:"modelSource,omitempty"`
	Available       bool     `json:"available"`
	ManagementNotes []string `json:"managementNotes,omitempty"`
}

type dashboardProviderDescriptor struct {
	ID            string
	Label         string
	APIKeyEnv     []string
	BaseURLEnv    []string
	DefaultModels []string
	ModelEnv      []string
	Notes         []string
}

type dashboardEnvValue struct {
	Value  string
	Source string
}

type dashboardSnapshotMsg struct {
	snapshot dashboardSnapshot
}

type dashboardDisconnectRequest struct {
	ClientID string `json:"clientID,omitempty"`
	All      bool   `json:"all,omitempty"`
}

type dashboardDisconnectResponse struct {
	DisconnectedIDs []string `json:"disconnectedIDs,omitempty"`
	Disconnected    int      `json:"disconnected"`
	Remaining       int      `json:"remaining"`
}

type dashboardDisconnectMsg struct {
	response dashboardDisconnectResponse
	err      error
}

type dashboardModel struct {
	projectRoot    string
	statusURL      string
	disconnectURL  string
	snapshot       *dashboardSnapshot
	selectedClient int
	lastAction     string
}

var dashboardProgramRunner = func(model tea.Model) (tea.Model, error) {
	return tea.NewProgram(model, tea.WithAltScreen()).Run()
}

var dashboardProviderCatalog = []dashboardProviderDescriptor{
	{ID: "openai", Label: "OpenAI", APIKeyEnv: []string{"OPENAI_API_KEY"}, BaseURLEnv: []string{"OPENAI_BASE_URL"}, ModelEnv: []string{"OPENAI_MODEL"}, DefaultModels: []string{"gpt-5.4-mini", "gpt-5.4"}, Notes: []string{"Configure an OpenAI API key to enable GPT inference."}},
	{ID: "anthropic", Label: "Anthropic", APIKeyEnv: []string{"ANTHROPIC_API_KEY"}, BaseURLEnv: []string{"ANTHROPIC_BASE_URL"}, ModelEnv: []string{"ANTHROPIC_MODEL"}, DefaultModels: []string{"claude-sonnet-4-5"}, Notes: []string{"Useful when you need Claude family models or reasoning-heavy prompts."}},
	{ID: "cerebras", Label: "Cerebras", APIKeyEnv: []string{"CEREBRAS_API_KEY"}, BaseURLEnv: []string{"CEREBRAS_BASE_URL"}, ModelEnv: []string{"CEREBRAS_MODEL"}, DefaultModels: []string{"gpt-oss-120b"}, Notes: []string{"Good fit when you want fast open-model inference."}},
	{ID: "gemini", Label: "Gemini", APIKeyEnv: []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"}, BaseURLEnv: []string{"GEMINI_BASE_URL", "GOOGLE_BASE_URL"}, ModelEnv: []string{"GEMINI_MODEL", "GOOGLE_MODEL"}, DefaultModels: []string{"gemini-2.5-flash"}, Notes: []string{"Supports Google-hosted Gemini deployments."}},
	{ID: "groq", Label: "Groq", APIKeyEnv: []string{"GROQ_API_KEY"}, BaseURLEnv: []string{"GROQ_BASE_URL"}, ModelEnv: []string{"GROQ_MODEL"}, DefaultModels: []string{"llama-3.3-70b-versatile"}, Notes: []string{"Useful for low-latency open-model inference."}},
	{ID: "mistral", Label: "Mistral", APIKeyEnv: []string{"MISTRAL_API_KEY"}, BaseURLEnv: []string{"MISTRAL_BASE_URL"}, ModelEnv: []string{"MISTRAL_MODEL"}, DefaultModels: []string{"mistral-large-latest"}, Notes: []string{"Configure this for Mistral-hosted models."}},
	{ID: "together", Label: "Together", APIKeyEnv: []string{"TOGETHER_API_KEY"}, BaseURLEnv: []string{"TOGETHER_BASE_URL"}, ModelEnv: []string{"TOGETHER_MODEL"}, DefaultModels: []string{"meta-llama/Llama-3.3-70B-Instruct-Turbo"}, Notes: []string{"Covers Together-hosted open-model catalogs."}},
	{ID: "openrouter", Label: "OpenRouter", APIKeyEnv: []string{"OPENROUTER_API_KEY"}, BaseURLEnv: []string{"OPENROUTER_BASE_URL"}, ModelEnv: []string{"OPENROUTER_MODEL"}, DefaultModels: []string{"openai/gpt-5-mini"}, Notes: []string{"Works well when routing multiple providers behind one API surface."}},
}

func (parseL launcher) runDashboard(parseArgs []string) error {
	parseConfig, parseErr := parseL.resolveDashboardConfig(parseArgs)
	if parseErr != nil {
		return parseErr
	}
	if strings.TrimSpace(parseConfig.projectRoot) == "" && strings.TrimSpace(parseConfig.statusURL) == "" {
		return nil
	}

	parseSnapshot := collectDashboardSnapshot(parseConfig.projectRoot, parseConfig.statusURL)
	if parseConfig.json {
		parseEncoded, parseErr2 := json.MarshalIndent(parseSnapshot, "", "  ")
		if parseErr2 != nil {
			return parseErr2
		}
		fmt.Println(string(parseEncoded))
		return nil
	}

	if !isInteractiveFile(os.Stdin) || !isInteractiveFile(os.Stdout) {
		printDashboardSnapshot(parseSnapshot)
		return nil
	}
	return runDashboardTUI(parseConfig, parseSnapshot)
}

func (parseL launcher) resolveDashboardConfig(parseArgs []string) (dashboardConfig, error) {
	parseFs := flag.NewFlagSet("dashboard", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseRoot := parseFs.String("root", "", "Project root used for provider inventory and dev status defaults")
	parseStatusURL := parseFs.String("status-url", "", "Explicit dev status endpoint to monitor")
	parseHost := parseFs.String("host", defaultHost, "Host for the default dev status endpoint")
	parsePort := parseFs.String("port", defaultPort, "Port for the default dev status endpoint")
	parseJsonOutput := parseFs.Bool("json", false, "Emit the dashboard snapshot as JSON")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return dashboardConfig{}, nil
		}
		return dashboardConfig{}, parseErr
	}

	parseResolvedRoot := strings.TrimSpace(*parseRoot)
	if parseResolvedRoot == "" {
		parseCwd, parseErr2 := launcherConfigGetwd()
		if parseErr2 != nil {
			return dashboardConfig{}, parseErr2
		}
		parseResolvedRoot = parseCwd
	}
	parseAbsRoot, parseErr3 := filepath.Abs(parseResolvedRoot)
	if parseErr3 != nil {
		return dashboardConfig{}, parseErr3
	}

	parseResolvedStatusURL := strings.TrimSpace(*parseStatusURL)
	if parseResolvedStatusURL == "" {
		parseResolvedStatusURL = "http://" + joinHostPort(strings.TrimSpace(*parseHost), strings.TrimSpace(*parsePort)) + "/__gwc/status"
	}
	return dashboardConfig{
		projectRoot:   parseAbsRoot,
		statusURL:     parseResolvedStatusURL,
		disconnectURL: dashboardDisconnectURL(parseResolvedStatusURL),
		json:          *parseJsonOutput,
	}, nil
}

func runDashboardTUI(parseConfig dashboardConfig, parseInitial dashboardSnapshot) error {
	parseModel := dashboardModel{
		projectRoot:   parseConfig.projectRoot,
		statusURL:     parseConfig.statusURL,
		disconnectURL: parseConfig.disconnectURL,
		snapshot:      &parseInitial,
	}
	_, parseErr := dashboardProgramRunner(parseModel)
	return parseErr
}

func (parseM dashboardModel) Init() tea.Cmd {
	return tea.Batch(parseM.pollSnapshotCmd(), dashboardTick())
}

func (parseM dashboardModel) pollSnapshotCmd() tea.Cmd {
	parseProjectRoot := parseM.projectRoot
	parseStatusURL := parseM.statusURL
	return func() tea.Msg {
		return dashboardSnapshotMsg{snapshot: collectDashboardSnapshot(parseProjectRoot, parseStatusURL)}
	}
}

func (parseM dashboardModel) disconnectClientCmd(parseClientID string, isDisconnectAll bool) tea.Cmd {
	parseDisconnectURL := parseM.disconnectURL
	return func() tea.Msg {
		parseRequestBody, parseErr := json.Marshal(dashboardDisconnectRequest{ClientID: parseClientID, All: isDisconnectAll})
		if parseErr != nil {
			return dashboardDisconnectMsg{err: parseErr}
		}
		parseReq, parseErr := http.NewRequest(http.MethodPost, parseDisconnectURL, strings.NewReader(string(parseRequestBody)))
		if parseErr != nil {
			return dashboardDisconnectMsg{err: parseErr}
		}
		parseReq.Header.Set("Content-Type", "application/json")
		parseResp, parseErr := (&http.Client{Timeout: 2 * time.Second}).Do(parseReq)
		if parseErr != nil {
			return dashboardDisconnectMsg{err: parseErr}
		}
		defer parseResp.Body.Close()
		if parseResp.StatusCode != http.StatusOK {
			return dashboardDisconnectMsg{err: fmt.Errorf("disconnect endpoint returned %s", parseResp.Status)}
		}
		var parsePayload dashboardDisconnectResponse
		if parseErr2 := json.NewDecoder(parseResp.Body).Decode(&parsePayload); parseErr2 != nil {
			return dashboardDisconnectMsg{err: parseErr2}
		}
		return dashboardDisconnectMsg{response: parsePayload}
	}
}

func dashboardTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return struct{}{} })
}

func (parseM dashboardModel) Update(parseMsg tea.Msg) (tea.Model, tea.Cmd) {
	switch parseTyped := parseMsg.(type) {
	case tea.KeyMsg:
		switch parseTyped.String() {
		case "q", "ctrl+c":
			return parseM, tea.Quit
		case "r":
			return parseM, parseM.pollSnapshotCmd()
		case "up", "k":
			if parseM.selectedClient > 0 {
				parseM.selectedClient--
			}
			return parseM, nil
		case "down", "j":
			if parseClients := parseM.currentClients(); parseM.selectedClient < len(parseClients)-1 {
				parseM.selectedClient++
			}
			return parseM, nil
		case "x":
			parseClientID := parseM.currentClientID()
			if parseClientID == "" {
				parseM.lastAction = "No client selected to disconnect."
				return parseM, nil
			}
			parseM.lastAction = "Disconnecting selected client..."
			return parseM, parseM.disconnectClientCmd(parseClientID, false)
		case "X":
			parseM.lastAction = "Disconnecting all live-reload clients..."
			return parseM, parseM.disconnectClientCmd("", true)
		}
	case dashboardSnapshotMsg:
		parseM.snapshot = &parseTyped.snapshot
		if parseClients2 := parseM.currentClients(); len(parseClients2) == 0 {
			parseM.selectedClient = 0
		} else if parseM.selectedClient >= len(parseClients2) {
			parseM.selectedClient = len(parseClients2) - 1
		}
		return parseM, nil
	case dashboardDisconnectMsg:
		if parseTyped.err != nil {
			parseM.lastAction = parseTyped.err.Error()
			return parseM, nil
		}
		parseM.lastAction = fmt.Sprintf("Disconnected %d client(s); %d remaining.", parseTyped.response.Disconnected, parseTyped.response.Remaining)
		return parseM, parseM.pollSnapshotCmd()
	case struct{}:
		return parseM, tea.Batch(parseM.pollSnapshotCmd(), dashboardTick())
	}
	return parseM, nil
}

func (parseM dashboardModel) View() string {
	var parseB strings.Builder
	parseB.WriteString("GWC Dashboard\n\n")
	parseB.WriteString(fmt.Sprintf("Project root: %s\n", parseM.projectRoot))
	parseB.WriteString(fmt.Sprintf("Status URL:   %s\n", parseM.statusURL))
	parseB.WriteString(fmt.Sprintf("Updated:      %s\n", dashboardUpdatedAt(parseM.snapshot)))
	parseB.WriteString("\n")

	if parseM.snapshot != nil && parseM.snapshot.Status != nil {
		parseStatus := parseM.snapshot.Status
		parseB.WriteString("Dev server\n")
		parseB.WriteString(fmt.Sprintf("  listening:    %s\n", parseStatus.ListeningURL))
		parseB.WriteString(fmt.Sprintf("  hot reload:   enabled=%t eligible=%t\n", parseStatus.HotReloadEnabled, parseStatus.HotReloadEligible))
		if strings.TrimSpace(parseStatus.ServedWASMPath) != "" {
			parseB.WriteString(fmt.Sprintf("  wasm:         %s\n", parseStatus.ServedWASMPath))
		}
		parseB.WriteString(fmt.Sprintf("  clients:      %d\n", parseStatus.ClientCount))
	} else {
		parseB.WriteString("Dev server\n")
		parseB.WriteString("  waiting for status endpoint\n")
		if parseM.snapshot != nil && strings.TrimSpace(parseM.snapshot.StatusError) != "" {
			parseB.WriteString(fmt.Sprintf("  detail:       %s\n", parseM.snapshot.StatusError))
		}
	}

	parseB.WriteString("\nClients\n")
	parseClients := parseM.currentClients()
	if len(parseClients) == 0 {
		parseB.WriteString("  no live-reload clients connected\n")
	} else {
		for parseIndex, parseClient := range parseClients {
			parsePrefix := " "
			if parseIndex == parseM.selectedClient {
				parsePrefix = ">"
			}
			parseB.WriteString(fmt.Sprintf("%s %s  %s\n", parsePrefix, parseClient.ID, dashboardClientLabel(parseClient)))
			parseB.WriteString(fmt.Sprintf("    connected %s, seen %s\n", dashboardFormatTime(parseClient.ConnectedAt), dashboardFormatTime(parseClient.LastSeenAt)))
		}
	}

	parseB.WriteString("\nProviders\n")
	parseProviders := []dashboardProviderStatus(nil)
	if parseM.snapshot != nil {
		parseProviders = parseM.snapshot.Providers
	}
	for _, parseProvider := range parseProviders {
		parseB.WriteString(fmt.Sprintf("  %-11s %s\n", parseProvider.Label, dashboardProviderSummary(parseProvider)))
	}

	if strings.TrimSpace(parseM.lastAction) != "" {
		parseB.WriteString("\nAction\n")
		parseB.WriteString("  " + parseM.lastAction + "\n")
	}

	parseB.WriteString("\nKeys: up/down select client, x disconnect selected, X disconnect all, r refresh, q quit.\n")
	return parseB.String()
}

func (parseM dashboardModel) currentClients() []dashboardClientSession {
	if parseM.snapshot == nil || parseM.snapshot.Status == nil {
		return nil
	}
	return parseM.snapshot.Status.Clients
}

func (parseM dashboardModel) currentClientID() string {
	parseClients := parseM.currentClients()
	if len(parseClients) == 0 || parseM.selectedClient < 0 || parseM.selectedClient >= len(parseClients) {
		return ""
	}
	return strings.TrimSpace(parseClients[parseM.selectedClient].ID)
}

func collectDashboardSnapshot(parseProjectRoot string, parseStatusURL string) dashboardSnapshot {
	parseSnapshot := dashboardSnapshot{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		ProjectRoot: parseProjectRoot,
		StatusURL:   parseStatusURL,
		Providers:   scanDashboardProviders(parseProjectRoot),
	}
	if strings.TrimSpace(parseStatusURL) == "" {
		return parseSnapshot
	}

	parseClient := &http.Client{Timeout: 1500 * time.Millisecond}
	parseResp, parseErr := parseClient.Get(parseStatusURL)
	if parseErr != nil {
		parseSnapshot.StatusError = parseErr.Error()
		return parseSnapshot
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusOK {
		parseSnapshot.StatusError = fmt.Sprintf("status endpoint returned %s", parseResp.Status)
		return parseSnapshot
	}

	var parsePayload dashboardStatusPayload
	if parseErr2 := json.NewDecoder(parseResp.Body).Decode(&parsePayload); parseErr2 != nil {
		parseSnapshot.StatusError = parseErr2.Error()
		return parseSnapshot
	}
	parseSnapshot.Status = &parsePayload
	return parseSnapshot
}

func scanDashboardProviders(parseProjectRoot string) []dashboardProviderStatus {
	parseValues := loadDashboardEnvValues(parseProjectRoot)
	parseProviders := make([]dashboardProviderStatus, 0, len(dashboardProviderCatalog))
	for _, parseDescriptor := range dashboardProviderCatalog {
		parseApiValue, parseApiName, parseApiSource := resolveDashboardEnvValue(parseValues, parseDescriptor.APIKeyEnv)
		parseBaseValue, _, parseBaseSource := resolveDashboardEnvValue(parseValues, parseDescriptor.BaseURLEnv)
		parseModelValue, _, parseModelSource := resolveDashboardEnvValue(parseValues, parseDescriptor.ModelEnv)
		if parseModelValue == "" && len(parseDescriptor.DefaultModels) > 0 {
			parseModelValue = parseDescriptor.DefaultModels[0]
			parseModelSource = "catalog default"
		}
		parseProviders = append(parseProviders, dashboardProviderStatus{
			ID:              parseDescriptor.ID,
			Label:           parseDescriptor.Label,
			APIKeyEnv:       parseApiName,
			AuthConfigured:  strings.TrimSpace(parseApiValue) != "",
			AuthSource:      parseApiSource,
			BaseURL:         parseBaseValue,
			BaseURLSource:   parseBaseSource,
			DefaultModel:    parseModelValue,
			ModelSource:     parseModelSource,
			Available:       strings.TrimSpace(parseApiValue) != "",
			ManagementNotes: parseDescriptor.Notes,
		})
	}
	return parseProviders
}

func loadDashboardEnvValues(parseProjectRoot string) map[string]dashboardEnvValue {
	parseValues := map[string]dashboardEnvValue{}
	for _, parseEntry := range os.Environ() {
		parseKey, parseValue, parseOk := strings.Cut(parseEntry, "=")
		if !parseOk {
			continue
		}
		parseKey = strings.TrimSpace(parseKey)
		if parseKey == "" || strings.TrimSpace(parseValue) == "" {
			continue
		}
		parseValues[parseKey] = dashboardEnvValue{Value: parseValue, Source: "process environment"}
	}

	for _, parsePath := range dashboardEnvCandidatePaths(parseProjectRoot) {
		parseFileValues, parseErr := parseDashboardEnvFile(parsePath)
		if parseErr != nil {
			continue
		}
		for parseKey2, parseValue2 := range parseFileValues {
			if _, parseExists := parseValues[parseKey2]; parseExists {
				continue
			}
			parseValues[parseKey2] = dashboardEnvValue{Value: parseValue2, Source: filepath.Base(parsePath)}
		}
	}
	return parseValues
}

func dashboardEnvCandidatePaths(parseProjectRoot string) []string {
	return []string{
		filepath.Join(parseProjectRoot, ".env.local"),
		filepath.Join(parseProjectRoot, ".env.development.local"),
		filepath.Join(parseProjectRoot, ".env.development"),
		filepath.Join(parseProjectRoot, ".env"),
	}
}

func parseDashboardEnvFile(parsePath string) (map[string]string, error) {
	parseContent, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return nil, parseErr
	}
	parseValues := map[string]string{}
	parseLines := strings.Split(string(parseContent), "\n")
	for _, parseRawLine := range parseLines {
		parseLine := strings.TrimSpace(strings.TrimPrefix(parseRawLine, "\ufeff"))
		if parseLine == "" || strings.HasPrefix(parseLine, "#") {
			continue
		}
		parseLine = strings.TrimPrefix(parseLine, "export ")
		parseKey, parseValue, parseOk := strings.Cut(parseLine, "=")
		if !parseOk {
			continue
		}
		parseKey = strings.TrimSpace(parseKey)
		parseValue = strings.TrimSpace(parseValue)
		parseValue = strings.Trim(parseValue, `"'`)
		if parseKey == "" {
			continue
		}
		parseValues[parseKey] = parseValue
	}
	return parseValues, nil
}

func resolveDashboardEnvValue(parseValues map[string]dashboardEnvValue, parseKeys []string) (string, string, string) {
	for _, parseKey := range parseKeys {
		parseEntry, parseOk := parseValues[parseKey]
		if !parseOk || strings.TrimSpace(parseEntry.Value) == "" {
			continue
		}
		return parseEntry.Value, parseKey, parseEntry.Source
	}
	if len(parseKeys) == 0 {
		return "", "", ""
	}
	return "", parseKeys[0], ""
}

func dashboardDisconnectURL(parseStatusURL string) string {
	parseTrimmed := strings.TrimSpace(parseStatusURL)
	if parseTrimmed == "" {
		return ""
	}
	if strings.HasSuffix(parseTrimmed, "/__gwc/status") {
		return strings.TrimSuffix(parseTrimmed, "/status") + "/clients/disconnect"
	}
	return strings.TrimRight(parseTrimmed, "/") + "/__gwc/clients/disconnect"
}

func printDashboardSnapshot(parseSnapshot dashboardSnapshot) {
	fmt.Println("GWC Dashboard")
	fmt.Printf("  project root: %s\n", parseSnapshot.ProjectRoot)
	fmt.Printf("  status url:   %s\n", parseSnapshot.StatusURL)
	if parseSnapshot.Status != nil {
		fmt.Printf("  clients:      %d\n", parseSnapshot.Status.ClientCount)
	} else if strings.TrimSpace(parseSnapshot.StatusError) != "" {
		fmt.Printf("  status:       %s\n", parseSnapshot.StatusError)
	}
	for _, parseProvider := range parseSnapshot.Providers {
		fmt.Printf("  provider:     %s\n", dashboardProviderSummary(parseProvider))
	}
}

func dashboardUpdatedAt(parseSnapshot *dashboardSnapshot) string {
	if parseSnapshot == nil || strings.TrimSpace(parseSnapshot.GeneratedAt) == "" {
		return "pending"
	}
	return parseSnapshot.GeneratedAt
}

func dashboardClientLabel(parseClient dashboardClientSession) string {
	parseParts := make([]string, 0, 2)
	if strings.TrimSpace(parseClient.RemoteAddr) != "" {
		parseParts = append(parseParts, parseClient.RemoteAddr)
	}
	if strings.TrimSpace(parseClient.UserAgent) != "" {
		parseParts = append(parseParts, dashboardTruncate(strings.TrimSpace(parseClient.UserAgent), 64))
	}
	if len(parseParts) == 0 {
		return "anonymous websocket client"
	}
	return strings.Join(parseParts, " | ")
}

func dashboardFormatTime(parseValue time.Time) string {
	if parseValue.IsZero() {
		return "unknown"
	}
	return parseValue.Local().Format("15:04:05")
}

func dashboardProviderSummary(parseProvider dashboardProviderStatus) string {
	parseStatus := "missing credentials"
	if parseProvider.AuthConfigured {
		parseStatus = "configured"
	}
	parseParts := []string{fmt.Sprintf("%s (%s)", parseProvider.Label, parseStatus)}
	if strings.TrimSpace(parseProvider.APIKeyEnv) != "" {
		if parseProvider.AuthConfigured {
			parseParts = append(parseParts, fmt.Sprintf("key=%s via %s", parseProvider.APIKeyEnv, parseProvider.AuthSource))
		} else {
			parseParts = append(parseParts, fmt.Sprintf("needs %s", parseProvider.APIKeyEnv))
		}
	}
	if strings.TrimSpace(parseProvider.DefaultModel) != "" {
		parseParts = append(parseParts, fmt.Sprintf("model=%s", parseProvider.DefaultModel))
	}
	if strings.TrimSpace(parseProvider.BaseURL) != "" {
		parseParts = append(parseParts, fmt.Sprintf("base=%s", parseProvider.BaseURL))
	}
	return strings.Join(parseParts, "; ")
}

func dashboardTruncate(parseText string, parseLimit int) string {
	if parseLimit <= 0 || len(parseText) <= parseLimit {
		return parseText
	}
	return parseText[:parseLimit-3] + "..."
}

func init() {
	slices.SortFunc(dashboardProviderCatalog, func(parseA dashboardProviderDescriptor, parseB dashboardProviderDescriptor) int {
		return strings.Compare(parseA.Label, parseB.Label)
	})
}
