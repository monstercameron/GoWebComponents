//go:build js && wasm

package app

import "time"

// ─── reconciler ──────────────────────────────────────────────────────────────

// innerHTMLProp is the reconciler key for raw HTML injection.
const innerHTMLProp = "__gwc_prop__:innerHTML"

// ─── branding ────────────────────────────────────────────────────────────────

const appBrandName = "RelayDesk"
const appVersion = "v2026.03.24.1"
const assistantBadgeText = "GWC"

// ─── DOM element IDs ─────────────────────────────────────────────────────────

const (
	idScrollAnchor       = "scroll-anchor"
	idScrollToBottomBtn  = "scroll-to-bottom-btn"
	idMessageList        = "message-list"
	idThreadScreen       = "thread-screen"
	idStreamingBubble    = "streaming-assistant-bubble"
	idCanvasPreviewPane  = "canvas-preview-pane"
	idCanvasPreviewFrame = "canvas-preview-frame"
	idCanvasWorkspace    = "canvas-workspace"
	idCanvasFrame        = "canvas-frame"
	idCanvasConsole      = "canvas-console"
	idCanvasSplitHandle  = "canvas-split-handle"
	idCanvasFocusEditor  = "canvas-focus-editor"
	idConvList           = "conversation-list"
	idChatInput          = "chat-input"
	idChatInputWrap      = "chat-input-wrap"
	idNameInput          = "name-input"
	idAuthNameInput      = "auth-name-input"
	idAuthEmailInput     = "auth-email-input"
	idAuthPasswordInput  = "auth-password-input"
	idSendBtn            = "send-btn"
	idEmptyState         = "empty-state"
	idQuotePrompt        = "quote-selection-prompt"
	idQuoteSpinner       = "quote-selection-spinner"
	appSelector          = "#app"
)

// ─── message roles ────────────────────────────────────────────────────────────

const (
	roleUser      = "user"
	roleAssistant = "assistant"
	roleSwitch    = "_switch" // visual-only model-switch divider, never sent to server
)

// ─── dataset attribute keys ───────────────────────────────────────────────────

const (
	dataIdx             = "idx"
	dataConvID          = "convid"
	dataProvider        = "provider"
	dataModel           = "model"
	dataTone            = "tone"
	dataLocale          = "locale"
	dataConvRow         = "convrow"
	dataThoughtSection  = "thoughtsection"
	dataThinkingEffort  = "thinkingeffort"
	dataMemoryIndex     = "memoryindex"
	dataMemoryField     = "memoryfield"
	dataCanvasID        = "canvasid"
	dataCanvasFocus     = "canvasfocus"
	dataSettingsSection = "settingssection"
)

// ─── gRPC ────────────────────────────────────────────────────────────────────

const grpcEndpoint = "/socket"
const authMetadataKey = "authorization"

const thoughtChunkModelPrefix = "__thought_delta__:"
const thoughtChunkModelDone = "__thought_done__"

const cacheKeySelectedModel = "chat-wizard:selected-model"
const cacheKeySelectedTone = "chat-wizard:selected-tone"
const cacheKeySelectedThinkingEnabled = "chat-wizard:selected-thinking-enabled"
const cacheKeySelectedThinkingEffort = "chat-wizard:selected-thinking-effort"
const cacheKeyCustomSystemPrompt = "chat-wizard:custom-system-prompt"
const storageKeyAuthToken = "chat-wizard:auth-token"
const storageKeyCanvasSplit = "chat-wizard:canvas-split"

const backgroundWorkerRuntimeURL = "/static/script/wasm_exec.js"
const backgroundWorkerWASMURL = "/worker/background-worker.wasm"

const backgroundWorkerRequestRenderMarkdown = "render-markdown"
const backgroundWorkerRequestRenderMarkdownBatch = "render-markdown-batch"
const backgroundWorkerCommandStartTicker = "start-ticker"
const backgroundWorkerCommandStopTicker = "stop-ticker"
const backgroundWorkerEventTick = "tick"

// ─── scroll ───────────────────────────────────────────────────────────────────

const scrollBehaviorInstant = "instant"
const scrollBehaviorSmooth = "smooth"

// ─── cache TTLs ───────────────────────────────────────────────────────────────

const (
	userNameTTL = 5 * time.Minute
	convListTTL = 10 * time.Minute // midpoint of 5-15 min idle window
	modelTTL    = 24 * time.Hour
	toneTTL     = 24 * time.Hour
)

// ─── timing ───────────────────────────────────────────────────────────────────

const (
	bgRefreshInterval = 60 * time.Second
	focusDelay        = 80 * time.Millisecond
	forkFocusDelay    = 50 * time.Millisecond
	scrollSettleDelay = 180 * time.Millisecond
	streamFollowDelay = 45 * time.Millisecond
	streamFollowHold  = 260 * time.Millisecond
	grpcDialTimeout   = 8 * time.Second
	grpcStatePoll     = 750 * time.Millisecond
	grpcStallLimit    = 12 * time.Second
	grpcSleepAfter    = 4 * time.Minute
	grpcSleepRetry    = 30 * time.Second
	grpcBackoffBase   = 500 * time.Millisecond
	grpcBackoffMax    = 12 * time.Second
)

// ─── layout / behaviour thresholds ───────────────────────────────────────────

const (
	scrollThresholdPx  = 80.0 // px from bottom; within this = at bottom
	sidebarPreviewLen  = 34   // max byte length before truncating sidebar preview
	canvasSplitMin     = 0.2
	canvasSplitMax     = 0.8
	canvasSplitDefault = 0.56
)

// ─── domain types ─────────────────────────────────────────────────────────────

type message struct {
	Role           string
	Content        string
	Pending        bool
	Thought        string
	ThoughtPending bool
	ModelID        string
	// Exact assistant usage from the server when available.
	PromptTokens     int
	CompletionTokens int
	// Streaming performance stats — populated on the assistant message once
	// streaming completes. Zero values mean stats are not available.
	TTFT   float64 // seconds from request send to first token
	TKPS   float64 // tokens per second during the stream body
	Tokens int     // total token-delta count
}

type thoughtSection struct {
	Key     string
	Heading string
	Body    string
}

type modelPricing struct {
	InputDollarsPerMillion  float64
	OutputDollarsPerMillion float64
	Currency                string
}

type assistantMessageCost struct {
	ModelID          string
	PromptTokens     int
	CompletionTokens int
	Cost             float64
}

type threadCostSummary struct {
	TotalCost              float64
	AssistantMessageCosts  map[int]assistantMessageCost
	HasAnyExactCosts       bool
	AllAssistantCostsExact bool
}

type markdownRenderRequest struct {
	Source string `json:"source"`
}

type markdownRenderResult struct {
	Source string `json:"source"`
	HTML   string `json:"html"`
}

type markdownRenderBatchRequest struct {
	Sources []string `json:"sources"`
}

type markdownRenderBatchResult struct {
	Results []markdownRenderResult `json:"results"`
}

type modelOption struct {
	ID           string
	Label        string
	Note         string
	Capabilities modelCapabilities
	Pricing      modelPricing
}

type modelCapabilities struct {
	ProviderID       string
	ProviderLabel    string
	SupportsThinking bool
	SupportsSpeech   bool
}

type modelCatalog struct {
	DefaultModel string
	Models       []modelOption
}

type providerOption struct {
	ID    string
	Label string
}

type toneOption struct {
	ID string
}

type thinkingEffortOption struct {
	ID string
}

// convSummary is the lightweight sidebar entry for a stored conversation.
type convSummary struct {
	ID        int64
	PublicID  string
	StartedAt string
	Preview   string
}

// ─── AI model catalogue ───────────────────────────────────────────────────────

const defaultModel = ""

const cacheKeyModelCatalog = "chat-wizard:model-catalog"

// ─── AI tone catalogue ────────────────────────────────────────────────────────

const defaultTone = "balanced"
const defaultThinkingEffort = "medium"
const defaultThinkingEnabled = true
const authModeLogin = "login"
const authModeSignup = "signup"

var availableTones = []toneOption{
	{ID: "balanced"},
	{ID: "friendly"},
	{ID: "professional"},
	{ID: "concise"},
}

var availableThinkingEfforts = []thinkingEffortOption{
	{ID: "off"},
	{ID: "low"},
	{ID: "medium"},
	{ID: "high"},
}
