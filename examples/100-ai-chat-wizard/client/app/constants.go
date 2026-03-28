//go:build js && wasm

package app

import (
	"time"

	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/internal/buildinfo"
)

// ─── reconciler ──────────────────────────────────────────────────────────────

// innerHTMLProp is the reconciler key for raw HTML injection.
const innerHTMLProp = "__gwc_prop__:innerHTML"

// ─── i18n ────────────────────────────────────────────────────────────────────

const (
	chatI18nNamespace        = "chat"
	marketingI18nNamespace   = "marketing"
	chatLocalePersistenceKey = "chat-wizard:locale"
)

// ─── branding ────────────────────────────────────────────────────────────────

const appBrandName = "RelayDesk"
const assistantBadgeText = "GWC"

// getAppVersion returns the shared RelayDesk version string for all visible app badges.
func getAppVersion() string {
	return buildinfo.GetBuildAppVersion()
}

// ─── DOM element IDs ─────────────────────────────────────────────────────────

const (
	idScrollAnchor            = "scroll-anchor"
	idScrollToBottomBtn       = "scroll-to-bottom-btn"
	idMessageList             = "message-list"
	idThreadScreen            = "thread-screen"
	idStreamingBubble         = "streaming-assistant-bubble"
	idCanvasWorkspace         = "canvas-workspace"
	idCanvasFrame             = "canvas-frame"
	idCanvasConsole           = "canvas-console"
	idCanvasSplitHandle       = "canvas-split-handle"
	idCanvasFocusEditor       = "canvas-focus-editor"
	idConvList                = "conversation-list"
	idChatInput               = "chat-input"
	idChatInputWrap           = "chat-input-wrap"
	idNameInput               = "name-input"
	idAuthNameInput           = "auth-name-input"
	idAuthEmailInput          = "auth-email-input"
	idAuthPasswordInput       = "auth-password-input"
	idSendBtn                 = "send-btn"
	idEmptyState              = "empty-state"
	idBillingPlatformFeeValue = "billing-platform-fee-value"
	idBillingUsageValue       = "billing-usage-value"
	idBillingPremiumValue     = "billing-premium-value"
	idBillingTotalValue       = "billing-total-value"
	idQuotePrompt             = "quote-selection-prompt"
	idQuoteSpinner            = "quote-selection-spinner"
	appSelector               = "#app"
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
	dataTTSProvider     = "ttsprovider"
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
	dataStarterPrompt   = "starterprompt"
	dataAdminUserID        = "adminuserid"
	dataAdminAction        = "adminaction"
	dataAdminWorkspaceID   = "adminwsid"
)

// ─── gRPC ────────────────────────────────────────────────────────────────────

const grpcEndpoint = "/socket"
const authMetadataKey = "authorization"
const clientMetadataKey = "x-chat-client-id"
const requestIDMetadataKey = "x-request-id"
const correlationIDMetadataKey = "x-correlation-id"
const traceParentMetadataKey = "traceparent"
const traceStateMetadataKey = "tracestate"

const thoughtChunkModelPrefix = "__thought_delta__:"
const thoughtChunkModelDone = "__thought_done__"

const cacheKeySelectedModel = "chat-wizard:selected-model"
const cacheKeySelectedTone = "chat-wizard:selected-tone"
const cacheKeySelectedThinkingEnabled = "chat-wizard:selected-thinking-enabled"
const cacheKeySelectedThinkingEffort = "chat-wizard:selected-thinking-effort"
const cacheKeyCustomSystemPrompt = "chat-wizard:custom-system-prompt"
const crossTabChannelSelectedModel = "chat-wizard:selected-model"
const storageKeyAuthToken = "chat-wizard:auth-token"
const storageKeyPostLoginRoute = "chat-wizard:post-login-route"
const storageKeyClientIdentity = "chat-wizard:client-id"
const storageKeyCorrelationIdentity = "chat-wizard:correlation-id"
const storageKeyCanvasSplit = "chat-wizard:canvas-split"

const brandLogoURL = "/static/images/relaydesk-logo.png"
const brandChatIconURL = "/static/images/relaydesk-chat-icon.png"
const backgroundWorkerRuntimeURL = "/static/script/wasm_exec.js"
const backgroundWorkerWASMURL = "/worker/background-worker.wasm"
const backgroundWorkerRenderPoolSize = 4

const backgroundWorkerRequestRenderMarkdownBatch = "render-markdown-batch"
const backgroundWorkerRequestRenderMessageMetadataBatch = "render-message-metadata-batch"
const backgroundWorkerRequestRenderThreadCostSummary = "render-thread-cost-summary"
const backgroundWorkerRequestRenderSignatures = "render-signatures"
const backgroundWorkerCommandStartTicker = "start-ticker"
const backgroundWorkerCommandStopTicker = "stop-ticker"
const backgroundWorkerCommandStartMaintenanceLoop = "start-maintenance-loop"
const backgroundWorkerCommandStopMaintenanceLoop = "stop-maintenance-loop"
const backgroundWorkerEventTick = "tick"
const backgroundWorkerEventMaintenanceBatch = "maintenance-batch"
const usagePremiumWindowKey = "__relaydesk_usage_premium_percent"
const platformFeeWindowKey = "__relaydesk_platform_fee_usd"
const defaultUsagePremiumPercent = 5.0
const defaultPlatformFeeUSD = 29.0

// ─── scroll ───────────────────────────────────────────────────────────────────

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

type accountCostSummary struct {
	ThreadCount          int
	PlatformFee          float64
	UsageCost            float64
	PremiumPercent       float64
	PremiumCost          float64
	TotalCost            float64
	HasAnyExactCosts     bool
	AllThreadCostsExact  bool
	HasCoverageGaps      bool
	FailedThreadLookups  int
	ExactThreadCostCount int
}

type markdownRenderResult struct {
	GetSourceBytes []byte `json:"sourceBytes"`
	GetHTMLBytes   []byte `json:"htmlBytes"`
}

type markdownRenderBatchRequest struct {
	GetSourceBytesList [][]byte `json:"sourceBytesList"`
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
const ttsProviderOpenAI = "openai"
const defaultTTSProvider = ttsProviderOpenAI

const cacheKeyModelCatalog = "chat-wizard:model-catalog"

// ─── AI tone catalogue ────────────────────────────────────────────────────────

const defaultTone = "balanced"
const defaultThinkingEffort = "medium"
const defaultThinkingEnabled = true
const authModeLogin = "login"
const authModeSignup = "signup"
const authModeReset = "reset"
const authModeUpdatePassword = "update_password"

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
