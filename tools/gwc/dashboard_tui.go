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

func (l launcher) runDashboard(args []string) error {
	config, err := l.resolveDashboardConfig(args)
	if err != nil {
		return err
	}
	if strings.TrimSpace(config.projectRoot) == "" && strings.TrimSpace(config.statusURL) == "" {
		return nil
	}

	snapshot := collectDashboardSnapshot(config.projectRoot, config.statusURL)
	if config.json {
		encoded, err := json.MarshalIndent(snapshot, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	}

	if !isInteractiveFile(os.Stdin) || !isInteractiveFile(os.Stdout) {
		printDashboardSnapshot(snapshot)
		return nil
	}
	return runDashboardTUI(config, snapshot)
}

func (l launcher) resolveDashboardConfig(args []string) (dashboardConfig, error) {
	fs := flag.NewFlagSet("dashboard", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	root := fs.String("root", "", "Project root used for provider inventory and dev status defaults")
	statusURL := fs.String("status-url", "", "Explicit dev status endpoint to monitor")
	host := fs.String("host", defaultHost, "Host for the default dev status endpoint")
	port := fs.String("port", defaultPort, "Port for the default dev status endpoint")
	jsonOutput := fs.Bool("json", false, "Emit the dashboard snapshot as JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return dashboardConfig{}, nil
		}
		return dashboardConfig{}, err
	}

	resolvedRoot := strings.TrimSpace(*root)
	if resolvedRoot == "" {
		cwd, err := launcherConfigGetwd()
		if err != nil {
			return dashboardConfig{}, err
		}
		resolvedRoot = cwd
	}
	absRoot, err := filepath.Abs(resolvedRoot)
	if err != nil {
		return dashboardConfig{}, err
	}

	resolvedStatusURL := strings.TrimSpace(*statusURL)
	if resolvedStatusURL == "" {
		resolvedStatusURL = "http://" + joinHostPort(strings.TrimSpace(*host), strings.TrimSpace(*port)) + "/__gwc/status"
	}
	return dashboardConfig{
		projectRoot:   absRoot,
		statusURL:     resolvedStatusURL,
		disconnectURL: dashboardDisconnectURL(resolvedStatusURL),
		json:          *jsonOutput,
	}, nil
}

func runDashboardTUI(config dashboardConfig, initial dashboardSnapshot) error {
	model := dashboardModel{
		projectRoot:   config.projectRoot,
		statusURL:     config.statusURL,
		disconnectURL: config.disconnectURL,
		snapshot:      &initial,
	}
	_, err := dashboardProgramRunner(model)
	return err
}

func (m dashboardModel) Init() tea.Cmd {
	return tea.Batch(m.pollSnapshotCmd(), dashboardTick())
}

func (m dashboardModel) pollSnapshotCmd() tea.Cmd {
	projectRoot := m.projectRoot
	statusURL := m.statusURL
	return func() tea.Msg {
		return dashboardSnapshotMsg{snapshot: collectDashboardSnapshot(projectRoot, statusURL)}
	}
}

func (m dashboardModel) disconnectClientCmd(clientID string, disconnectAll bool) tea.Cmd {
	disconnectURL := m.disconnectURL
	return func() tea.Msg {
		requestBody, err := json.Marshal(dashboardDisconnectRequest{ClientID: clientID, All: disconnectAll})
		if err != nil {
			return dashboardDisconnectMsg{err: err}
		}
		req, err := http.NewRequest(http.MethodPost, disconnectURL, strings.NewReader(string(requestBody)))
		if err != nil {
			return dashboardDisconnectMsg{err: err}
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
		if err != nil {
			return dashboardDisconnectMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return dashboardDisconnectMsg{err: fmt.Errorf("disconnect endpoint returned %s", resp.Status)}
		}
		var payload dashboardDisconnectResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return dashboardDisconnectMsg{err: err}
		}
		return dashboardDisconnectMsg{response: payload}
	}
}

func dashboardTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return struct{}{} })
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		switch typed.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			return m, m.pollSnapshotCmd()
		case "up", "k":
			if m.selectedClient > 0 {
				m.selectedClient--
			}
			return m, nil
		case "down", "j":
			if clients := m.currentClients(); m.selectedClient < len(clients)-1 {
				m.selectedClient++
			}
			return m, nil
		case "x":
			clientID := m.currentClientID()
			if clientID == "" {
				m.lastAction = "No client selected to disconnect."
				return m, nil
			}
			m.lastAction = "Disconnecting selected client..."
			return m, m.disconnectClientCmd(clientID, false)
		case "X":
			m.lastAction = "Disconnecting all live-reload clients..."
			return m, m.disconnectClientCmd("", true)
		}
	case dashboardSnapshotMsg:
		m.snapshot = &typed.snapshot
		if clients := m.currentClients(); len(clients) == 0 {
			m.selectedClient = 0
		} else if m.selectedClient >= len(clients) {
			m.selectedClient = len(clients) - 1
		}
		return m, nil
	case dashboardDisconnectMsg:
		if typed.err != nil {
			m.lastAction = typed.err.Error()
			return m, nil
		}
		m.lastAction = fmt.Sprintf("Disconnected %d client(s); %d remaining.", typed.response.Disconnected, typed.response.Remaining)
		return m, m.pollSnapshotCmd()
	case struct{}:
		return m, tea.Batch(m.pollSnapshotCmd(), dashboardTick())
	}
	return m, nil
}

func (m dashboardModel) View() string {
	var b strings.Builder
	b.WriteString("GWC Dashboard\n\n")
	b.WriteString(fmt.Sprintf("Project root: %s\n", m.projectRoot))
	b.WriteString(fmt.Sprintf("Status URL:   %s\n", m.statusURL))
	b.WriteString(fmt.Sprintf("Updated:      %s\n", dashboardUpdatedAt(m.snapshot)))
	b.WriteString("\n")

	if m.snapshot != nil && m.snapshot.Status != nil {
		status := m.snapshot.Status
		b.WriteString("Dev server\n")
		b.WriteString(fmt.Sprintf("  listening:    %s\n", status.ListeningURL))
		b.WriteString(fmt.Sprintf("  hot reload:   enabled=%t eligible=%t\n", status.HotReloadEnabled, status.HotReloadEligible))
		if strings.TrimSpace(status.ServedWASMPath) != "" {
			b.WriteString(fmt.Sprintf("  wasm:         %s\n", status.ServedWASMPath))
		}
		b.WriteString(fmt.Sprintf("  clients:      %d\n", status.ClientCount))
	} else {
		b.WriteString("Dev server\n")
		b.WriteString("  waiting for status endpoint\n")
		if m.snapshot != nil && strings.TrimSpace(m.snapshot.StatusError) != "" {
			b.WriteString(fmt.Sprintf("  detail:       %s\n", m.snapshot.StatusError))
		}
	}

	b.WriteString("\nClients\n")
	clients := m.currentClients()
	if len(clients) == 0 {
		b.WriteString("  no live-reload clients connected\n")
	} else {
		for index, client := range clients {
			prefix := " "
			if index == m.selectedClient {
				prefix = ">"
			}
			b.WriteString(fmt.Sprintf("%s %s  %s\n", prefix, client.ID, dashboardClientLabel(client)))
			b.WriteString(fmt.Sprintf("    connected %s, seen %s\n", dashboardFormatTime(client.ConnectedAt), dashboardFormatTime(client.LastSeenAt)))
		}
	}

	b.WriteString("\nProviders\n")
	providers := []dashboardProviderStatus(nil)
	if m.snapshot != nil {
		providers = m.snapshot.Providers
	}
	for _, provider := range providers {
		b.WriteString(fmt.Sprintf("  %-11s %s\n", provider.Label, dashboardProviderSummary(provider)))
	}

	if strings.TrimSpace(m.lastAction) != "" {
		b.WriteString("\nAction\n")
		b.WriteString("  " + m.lastAction + "\n")
	}

	b.WriteString("\nKeys: up/down select client, x disconnect selected, X disconnect all, r refresh, q quit.\n")
	return b.String()
}

func (m dashboardModel) currentClients() []dashboardClientSession {
	if m.snapshot == nil || m.snapshot.Status == nil {
		return nil
	}
	return m.snapshot.Status.Clients
}

func (m dashboardModel) currentClientID() string {
	clients := m.currentClients()
	if len(clients) == 0 || m.selectedClient < 0 || m.selectedClient >= len(clients) {
		return ""
	}
	return strings.TrimSpace(clients[m.selectedClient].ID)
}

func collectDashboardSnapshot(projectRoot string, statusURL string) dashboardSnapshot {
	snapshot := dashboardSnapshot{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		ProjectRoot: projectRoot,
		StatusURL:   statusURL,
		Providers:   scanDashboardProviders(projectRoot),
	}
	if strings.TrimSpace(statusURL) == "" {
		return snapshot
	}

	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Get(statusURL)
	if err != nil {
		snapshot.StatusError = err.Error()
		return snapshot
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		snapshot.StatusError = fmt.Sprintf("status endpoint returned %s", resp.Status)
		return snapshot
	}

	var payload dashboardStatusPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		snapshot.StatusError = err.Error()
		return snapshot
	}
	snapshot.Status = &payload
	return snapshot
}

func scanDashboardProviders(projectRoot string) []dashboardProviderStatus {
	values := loadDashboardEnvValues(projectRoot)
	providers := make([]dashboardProviderStatus, 0, len(dashboardProviderCatalog))
	for _, descriptor := range dashboardProviderCatalog {
		apiValue, apiName, apiSource := resolveDashboardEnvValue(values, descriptor.APIKeyEnv)
		baseValue, _, baseSource := resolveDashboardEnvValue(values, descriptor.BaseURLEnv)
		modelValue, _, modelSource := resolveDashboardEnvValue(values, descriptor.ModelEnv)
		if modelValue == "" && len(descriptor.DefaultModels) > 0 {
			modelValue = descriptor.DefaultModels[0]
			modelSource = "catalog default"
		}
		providers = append(providers, dashboardProviderStatus{
			ID:              descriptor.ID,
			Label:           descriptor.Label,
			APIKeyEnv:       apiName,
			AuthConfigured:  strings.TrimSpace(apiValue) != "",
			AuthSource:      apiSource,
			BaseURL:         baseValue,
			BaseURLSource:   baseSource,
			DefaultModel:    modelValue,
			ModelSource:     modelSource,
			Available:       strings.TrimSpace(apiValue) != "",
			ManagementNotes: descriptor.Notes,
		})
	}
	return providers
}

func loadDashboardEnvValues(projectRoot string) map[string]dashboardEnvValue {
	values := map[string]dashboardEnvValue{}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || strings.TrimSpace(value) == "" {
			continue
		}
		values[key] = dashboardEnvValue{Value: value, Source: "process environment"}
	}

	for _, path := range dashboardEnvCandidatePaths(projectRoot) {
		fileValues, err := parseDashboardEnvFile(path)
		if err != nil {
			continue
		}
		for key, value := range fileValues {
			if _, exists := values[key]; exists {
				continue
			}
			values[key] = dashboardEnvValue{Value: value, Source: filepath.Base(path)}
		}
	}
	return values
}

func dashboardEnvCandidatePaths(projectRoot string) []string {
	return []string{
		filepath.Join(projectRoot, ".env.local"),
		filepath.Join(projectRoot, ".env.development.local"),
		filepath.Join(projectRoot, ".env.development"),
		filepath.Join(projectRoot, ".env"),
	}
}

func parseDashboardEnvFile(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	lines := strings.Split(string(content), "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(strings.TrimPrefix(rawLine, "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}
		values[key] = value
	}
	return values, nil
}

func resolveDashboardEnvValue(values map[string]dashboardEnvValue, keys []string) (string, string, string) {
	for _, key := range keys {
		entry, ok := values[key]
		if !ok || strings.TrimSpace(entry.Value) == "" {
			continue
		}
		return entry.Value, key, entry.Source
	}
	if len(keys) == 0 {
		return "", "", ""
	}
	return "", keys[0], ""
}

func dashboardDisconnectURL(statusURL string) string {
	trimmed := strings.TrimSpace(statusURL)
	if trimmed == "" {
		return ""
	}
	if strings.HasSuffix(trimmed, "/__gwc/status") {
		return strings.TrimSuffix(trimmed, "/status") + "/clients/disconnect"
	}
	return strings.TrimRight(trimmed, "/") + "/__gwc/clients/disconnect"
}

func printDashboardSnapshot(snapshot dashboardSnapshot) {
	fmt.Println("GWC Dashboard")
	fmt.Printf("  project root: %s\n", snapshot.ProjectRoot)
	fmt.Printf("  status url:   %s\n", snapshot.StatusURL)
	if snapshot.Status != nil {
		fmt.Printf("  clients:      %d\n", snapshot.Status.ClientCount)
	} else if strings.TrimSpace(snapshot.StatusError) != "" {
		fmt.Printf("  status:       %s\n", snapshot.StatusError)
	}
	for _, provider := range snapshot.Providers {
		fmt.Printf("  provider:     %s\n", dashboardProviderSummary(provider))
	}
}

func dashboardUpdatedAt(snapshot *dashboardSnapshot) string {
	if snapshot == nil || strings.TrimSpace(snapshot.GeneratedAt) == "" {
		return "pending"
	}
	return snapshot.GeneratedAt
}

func dashboardClientLabel(client dashboardClientSession) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(client.RemoteAddr) != "" {
		parts = append(parts, client.RemoteAddr)
	}
	if strings.TrimSpace(client.UserAgent) != "" {
		parts = append(parts, dashboardTruncate(strings.TrimSpace(client.UserAgent), 64))
	}
	if len(parts) == 0 {
		return "anonymous websocket client"
	}
	return strings.Join(parts, " | ")
}

func dashboardFormatTime(value time.Time) string {
	if value.IsZero() {
		return "unknown"
	}
	return value.Local().Format("15:04:05")
}

func dashboardProviderSummary(provider dashboardProviderStatus) string {
	status := "missing credentials"
	if provider.AuthConfigured {
		status = "configured"
	}
	parts := []string{fmt.Sprintf("%s (%s)", provider.Label, status)}
	if strings.TrimSpace(provider.APIKeyEnv) != "" {
		if provider.AuthConfigured {
			parts = append(parts, fmt.Sprintf("key=%s via %s", provider.APIKeyEnv, provider.AuthSource))
		} else {
			parts = append(parts, fmt.Sprintf("needs %s", provider.APIKeyEnv))
		}
	}
	if strings.TrimSpace(provider.DefaultModel) != "" {
		parts = append(parts, fmt.Sprintf("model=%s", provider.DefaultModel))
	}
	if strings.TrimSpace(provider.BaseURL) != "" {
		parts = append(parts, fmt.Sprintf("base=%s", provider.BaseURL))
	}
	return strings.Join(parts, "; ")
}

func dashboardTruncate(text string, limit int) string {
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return text[:limit-3] + "..."
}

func init() {
	slices.SortFunc(dashboardProviderCatalog, func(a dashboardProviderDescriptor, b dashboardProviderDescriptor) int {
		return strings.Compare(a.Label, b.Label)
	})
}
