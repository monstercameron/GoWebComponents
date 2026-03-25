//go:build js && wasm

package app

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/internal/markdownrender"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/ui"
)

var chatLog = logging.New("chat-wizard")

// ─── markdown ─────────────────────────────────────────────────────────────────

// renderedMarkdownCache stores previously rendered markdown so that unchanged
// completed messages are not re-converted on every streaming token delta.
var renderedMarkdownCache = map[string]string{}
var thoughtSectionsCache = map[string][]thoughtSection{}
var threadCostSummaryCache = map[string]threadCostSummary{}

func cachedRenderedMarkdown(markdown string) (string, bool) {
	renderedHTML, ok := renderedMarkdownCache[markdown]
	return renderedHTML, ok
}

func cacheRenderedMarkdown(markdown, renderedHTML string) {
	if markdown == "" || renderedHTML == "" {
		return
	}
	renderedMarkdownCache[markdown] = renderedHTML
}

func renderMarkdownSync(markdown string) string {
	if renderedHTML, ok := renderedMarkdownCache[markdown]; ok {
		return renderedHTML
	}
	renderedHTML, err := markdownrender.Render(markdown)
	if err != nil {
		return markdown
	}
	renderedMarkdownCache[markdown] = renderedHTML
	return renderedHTML
}

func completedAssistantMessagesMarkdownSignature(messages []message) string {
	var builder strings.Builder
	for _, messageItem := range messages {
		if messageItem.Role != roleAssistant || messageItem.Pending {
			continue
		}
		if strings.TrimSpace(messageItem.Content) == "" {
			continue
		}
		builder.WriteString(messageItem.Content)
		builder.WriteString("\n\x1f\n")
	}
	return builder.String()
}

// ─── scroll ───────────────────────────────────────────────────────────────────

func scrollMessageListToBottom() {
	doc, err := interop.CurrentDocument()
	if err != nil {
		return
	}
	messageListElement, foundMessageList, err := doc.ElementByID(idMessageList)
	if err != nil || !foundMessageList {
		return
	}
	_, scrollHeight, _, err := messageListElement.ScrollMetrics()
	if err != nil {
		return
	}
	_ = messageListElement.SetScrollTop(scrollHeight)
}

func scrollStreamingAssistantBubbleIntoView(behavior string) {
	doc, err := interop.CurrentDocument()
	if err != nil {
		return
	}
	streamingBubbleElement, foundStreamingBubble, err := doc.ElementByID(idStreamingBubble)
	if err != nil || !foundStreamingBubble {
		scrollMessageListToBottom()
		return
	}
	options := interop.ScrollIntoViewOptions{
		Behavior: behavior,
		Block:    "start",
	}
	if err := streamingBubbleElement.ScrollIntoView(options); err != nil {
		scrollMessageListToBottom()
	}
}

// isMessageListAtScrollBottom returns true when the message list scroll container is
// within scrollThresholdPx of the bottom — the threshold below which
// auto-scroll is active.
func isMessageListAtScrollBottom() bool {
	doc, err := interop.CurrentDocument()
	if err != nil {
		return true
	}
	messageListElement, foundMessageList, err := doc.ElementByID(idMessageList)
	if err != nil || !foundMessageList {
		return true
	}
	scrollTop, scrollHeight, clientHeight, err := messageListElement.ScrollMetrics()
	if err != nil {
		return true
	}
	return scrollHeight-scrollTop-clientHeight < scrollThresholdPx
}

func messageListScrollTop() (float64, bool) {
	doc, err := interop.CurrentDocument()
	if err != nil {
		return 0, false
	}
	messageListElement, foundMessageList, err := doc.ElementByID(idMessageList)
	if err != nil || !foundMessageList {
		return 0, false
	}
	scrollTop, _, _, err := messageListElement.ScrollMetrics()
	if err != nil {
		return 0, false
	}
	return scrollTop, true
}

func setMessageListScrollTop(scrollTop float64) bool {
	doc, err := interop.CurrentDocument()
	if err != nil {
		return false
	}
	messageListElement, foundMessageList, err := doc.ElementByID(idMessageList)
	if err != nil || !foundMessageList {
		return false
	}
	if err := messageListElement.SetScrollTop(scrollTop); err != nil {
		return false
	}
	return true
}

func focusChatInput() {
	doc, err := interop.CurrentDocument()
	if err != nil {
		return
	}
	chatInputElement, foundChatInput, err := doc.ElementByID(idChatInput)
	if err != nil || !foundChatInput {
		return
	}
	_ = chatInputElement.Focus()
}

func currentWASMQuerySuffix() string {
	env := interop.SharedWindowEnv()
	suffix, ok := env.LookupString("__gwc_wasm_query")
	if !ok {
		return ""
	}
	suffix = strings.TrimSpace(suffix)
	if suffix == "?br=true" {
		return suffix
	}
	return ""
}

// ─── text ─────────────────────────────────────────────────────────────────────

// displayNameInitials returns up to 2 uppercase initials from a display name.
func displayNameInitials(displayName string) string {
	nameParts := strings.Fields(displayName)
	initialsText := ""
	for _, part := range nameParts {
		runes := []rune(part)
		if len(runes) > 0 {
			initialsText += strings.ToUpper(string(runes[0]))
		}
		if len(initialsText) >= 2 {
			break
		}
	}
	if initialsText == "" {
		return "?"
	}
	return initialsText
}

const managedUserNameMemoryKey = "profile.display_name"

func managedUserNameMemory(displayName string) editableUserMemory {
	return editableUserMemory{
		Key:          managedUserNameMemoryKey,
		Category:     "identity",
		Summary:      strings.TrimSpace(displayName),
		Detail:       "Primary display name used for this account.",
		RubricReason: "Pinned from the profile display name so the assistant sees it first.",
	}
}

func isManagedUserNameMemory(memory editableUserMemory) bool {
	return strings.TrimSpace(memory.Key) == managedUserNameMemoryKey
}

func ensureManagedUserNameMemory(displayName string, memories []editableUserMemory) []editableUserMemory {
	filtered := make([]editableUserMemory, 0, len(memories)+1)
	for _, memory := range memories {
		if isManagedUserNameMemory(memory) {
			continue
		}
		filtered = append(filtered, memory)
	}
	trimmedName := strings.TrimSpace(displayName)
	if trimmedName == "" {
		return filtered
	}
	return append([]editableUserMemory{managedUserNameMemory(trimmedName)}, filtered...)
}

func defaultModelCatalog() modelCatalog {
	return modelCatalog{
		DefaultModel: defaultModel,
		Models:       append([]modelOption(nil), availableModels...),
	}
}

func modelOptionByID(modelID string, models []modelOption) (modelOption, bool) {
	trimmedModelID := strings.TrimSpace(modelID)
	for _, option := range models {
		if option.ID == trimmedModelID {
			return option, true
		}
	}
	return modelOption{}, false
}

func providerOptionsForModels(models []modelOption) []providerOption {
	if len(models) == 0 {
		models = availableModels
	}
	options := make([]providerOption, 0, len(models))
	seenProviders := map[string]struct{}{}
	for _, option := range models {
		providerID := strings.TrimSpace(option.Capabilities.ProviderID)
		if providerID == "" {
			providerID = "default"
		}
		if _, seen := seenProviders[providerID]; seen {
			continue
		}
		providerLabel := strings.TrimSpace(option.Capabilities.ProviderLabel)
		if providerLabel == "" {
			providerLabel = strings.ToUpper(providerID)
		}
		options = append(options, providerOption{ID: providerID, Label: providerLabel})
		seenProviders[providerID] = struct{}{}
	}
	return options
}

func providerForModel(modelID string, models []modelOption, fallback string) providerOption {
	resolvedModelID := normalizeSelectedModelID(modelID, models, fallback)
	option, ok := modelOptionByID(resolvedModelID, models)
	if !ok {
		return providerOption{}
	}
	providerID := strings.TrimSpace(option.Capabilities.ProviderID)
	if providerID == "" {
		providerID = "default"
	}
	providerLabel := strings.TrimSpace(option.Capabilities.ProviderLabel)
	if providerLabel == "" {
		providerLabel = strings.ToUpper(providerID)
	}
	return providerOption{ID: providerID, Label: providerLabel}
}

func modelsForProvider(models []modelOption, providerID string) []modelOption {
	if len(models) == 0 {
		models = availableModels
	}
	trimmedProviderID := strings.TrimSpace(providerID)
	if trimmedProviderID == "" {
		return append([]modelOption(nil), models...)
	}
	filtered := make([]modelOption, 0, len(models))
	for _, option := range models {
		candidateProviderID := strings.TrimSpace(option.Capabilities.ProviderID)
		if candidateProviderID == "" {
			candidateProviderID = "default"
		}
		if candidateProviderID == trimmedProviderID {
			filtered = append(filtered, option)
		}
	}
	if len(filtered) == 0 {
		return append([]modelOption(nil), models...)
	}
	return filtered
}

func defaultModelForProvider(providerID string, models []modelOption, fallback string) string {
	filtered := modelsForProvider(models, providerID)
	if len(filtered) == 0 {
		return normalizeSelectedModelID(fallback, models, fallback)
	}
	resolvedFallback := normalizeSelectedModelID(fallback, models, fallback)
	if option, ok := modelOptionByID(resolvedFallback, models); ok {
		candidateProviderID := strings.TrimSpace(option.Capabilities.ProviderID)
		if candidateProviderID == "" {
			candidateProviderID = "default"
		}
		if candidateProviderID == strings.TrimSpace(providerID) {
			return option.ID
		}
	}
	return filtered[0].ID
}

func normalizeSelectedModelID(modelID string, models []modelOption, fallback string) string {
	modelID = strings.TrimSpace(modelID)
	switch modelID {
	case "gpt-5.4-2026-03-17":
		modelID = "gpt-5.4"
	case "gpt-5.4-mini-2026-03-17":
		modelID = "gpt-5.4-mini"
	case "gpt-5.4-nano-2026-03-17":
		modelID = "gpt-5.4-nano"
	}
	if len(models) == 0 {
		models = availableModels
	}
	if strings.TrimSpace(fallback) == "" {
		fallback = defaultModel
	}
	if modelID == "" {
		if _, ok := modelOptionByID(fallback, models); ok {
			return fallback
		}
		return models[0].ID
	}
	if _, ok := modelOptionByID(modelID, models); ok {
		return modelID
	}
	if _, ok := modelOptionByID(fallback, models); ok {
		return fallback
	}
	return models[0].ID
}

func selectedModelForConversation(messages []message, models []modelOption, fallback string) string {
	for idx := len(messages) - 1; idx >= 0; idx-- {
		messageItem := messages[idx]
		if messageItem.Role == roleSwitch {
			switchModel := strings.TrimSpace(messageItem.Content)
			if switchModel != "" {
				return normalizeSelectedModelID(switchModel, models, fallback)
			}
		}
		if messageItem.Role != roleAssistant {
			continue
		}
		modelID := strings.TrimSpace(messageItem.ModelID)
		if modelID != "" {
			return normalizeSelectedModelID(modelID, models, fallback)
		}
	}
	return normalizeSelectedModelID(fallback, models, fallback)
}

func normalizeSelectedToneID(toneID string) string {
	toneID = strings.TrimSpace(toneID)
	if toneID == "" {
		return defaultTone
	}
	for _, toneOption := range availableTones {
		if toneOption.ID == toneID {
			return toneID
		}
	}
	return defaultTone
}

func normalizeSelectedThinkingEffort(effort string) string {
	effort = strings.TrimSpace(strings.ToLower(effort))
	if effort == "" {
		return defaultThinkingEffort
	}
	for _, option := range availableThinkingEfforts {
		if option.ID == effort {
			return effort
		}
	}
	return defaultThinkingEffort
}

func modelLabelForID(modelID string, models []modelOption) string {
	if option, ok := modelOptionByID(modelID, models); ok {
		return option.Label
	}
	if strings.TrimSpace(modelID) == "" {
		return ""
	}
	return modelID
}

func modelSupportsThinking(modelID string, models []modelOption, fallback string) bool {
	resolvedModelID := normalizeSelectedModelID(modelID, models, fallback)
	option, ok := modelOptionByID(resolvedModelID, models)
	return ok && option.Capabilities.SupportsThinking
}

func modelSupportsSpeech(modelID string, models []modelOption, fallback string) bool {
	resolvedModelID := normalizeSelectedModelID(modelID, models, fallback)
	option, ok := modelOptionByID(resolvedModelID, models)
	return ok && option.Capabilities.SupportsSpeech
}

func sameModelOptions(left, right []modelOption) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func previewLogText(text string, maxLen int) string {
	trimmedText := strings.TrimSpace(text)
	if maxLen <= 0 || len(trimmedText) <= maxLen {
		return trimmedText
	}
	if maxLen <= 1 {
		return trimmedText[:maxLen]
	}
	return trimmedText[:maxLen-1] + "…"
}

func thoughtSectionKey(messageIndex, sectionIndex int, heading string) string {
	return fmt.Sprintf("%d:%d:%s", messageIndex, sectionIndex, strings.TrimSpace(heading))
}

func materializeThoughtSections(messageIndex int, cachedSections []thoughtSection) []thoughtSection {
	if len(cachedSections) == 0 {
		return nil
	}
	sections := make([]thoughtSection, len(cachedSections))
	for idx, section := range cachedSections {
		sections[idx] = thoughtSection{
			Key:     thoughtSectionKey(messageIndex, idx, section.Heading),
			Heading: section.Heading,
			Body:    section.Body,
		}
	}
	return sections
}

func parseThoughtHeading(line string) (string, bool) {
	trimmedLine := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmedLine, "**") || !strings.HasSuffix(trimmedLine, "**") || len(trimmedLine) <= 4 {
		return "", false
	}
	heading := strings.TrimSpace(trimmedLine[2 : len(trimmedLine)-2])
	return heading, heading != ""
}

func parseThoughtSections(messageIndex int, thoughtText string) []thoughtSection {
	normalizedText := strings.TrimSpace(strings.ReplaceAll(thoughtText, "\r\n", "\n"))
	if normalizedText == "" {
		return nil
	}
	if cachedSections, ok := thoughtSectionsCache[normalizedText]; ok {
		return materializeThoughtSections(messageIndex, cachedSections)
	}

	lines := strings.Split(normalizedText, "\n")
	sections := make([]thoughtSection, 0, 4)
	currentHeading := ""
	currentBodyLines := make([]string, 0, len(lines))

	flushCurrent := func() {
		if currentHeading == "" && len(currentBodyLines) == 0 {
			return
		}
		heading := strings.TrimSpace(currentHeading)
		body := strings.TrimSpace(strings.Join(currentBodyLines, "\n"))
		if heading == "" {
			heading = "Thinking"
		}
		sections = append(sections, thoughtSection{
			Heading: heading,
			Body:    body,
		})
		currentHeading = ""
		currentBodyLines = currentBodyLines[:0]
	}

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if len(sections) == 0 && currentHeading == "" && len(currentBodyLines) == 0 && strings.EqualFold(trimmedLine, "thinking") {
			continue
		}
		if heading, ok := parseThoughtHeading(trimmedLine); ok {
			flushCurrent()
			currentHeading = heading
			continue
		}
		currentBodyLines = append(currentBodyLines, line)
	}
	flushCurrent()

	if len(sections) == 0 {
		sections = []thoughtSection{{
			Heading: "Thinking",
			Body:    normalizedText,
		}}
	}

	thoughtSectionsCache[normalizedText] = sections
	return materializeThoughtSections(messageIndex, sections)
}

func exactAssistantMessageCost(modelID string, promptTokens, completionTokens int) (assistantMessageCost, bool) {
	trimmedModelID := strings.TrimSpace(modelID)
	pricing, foundPricing := availableModelPricing[trimmedModelID]
	if !foundPricing || (promptTokens <= 0 && completionTokens <= 0) {
		return assistantMessageCost{
			ModelID:          trimmedModelID,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
		}, false
	}
	cost := (float64(promptTokens) * pricing.InputDollarsPerMillion / 1_000_000) +
		(float64(completionTokens) * pricing.OutputDollarsPerMillion / 1_000_000)
	return assistantMessageCost{
		ModelID:          trimmedModelID,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		Cost:             cost,
	}, true
}

func threadCostSummarySignature(messages []message) string {
	var builder strings.Builder
	for messageIndex, messageItem := range messages {
		if messageItem.Role != roleAssistant || messageItem.Pending {
			continue
		}
		if strings.TrimSpace(messageItem.Content) == "" {
			continue
		}
		builder.WriteString(fmt.Sprintf("%d|%s|%d|%d\n", messageIndex, messageItem.ModelID, messageItem.PromptTokens, messageItem.CompletionTokens))
	}
	return builder.String()
}

func deriveThreadCostSummary(messages []message) threadCostSummary {
	signature := threadCostSummarySignature(messages)
	if cachedSummary, ok := threadCostSummaryCache[signature]; ok {
		return cachedSummary
	}
	summary := threadCostSummary{
		AssistantMessageCosts:  make(map[int]assistantMessageCost),
		AllAssistantCostsExact: true,
	}
	assistantMessageCount := 0

	for messageIndex, messageItem := range messages {
		if messageItem.Role != roleAssistant || messageItem.Pending {
			continue
		}
		if strings.TrimSpace(messageItem.Content) == "" {
			continue
		}
		assistantMessageCount++
		messageCost, hasExactCost := exactAssistantMessageCost(messageItem.ModelID, messageItem.PromptTokens, messageItem.CompletionTokens)
		if !hasExactCost {
			summary.AllAssistantCostsExact = false
			continue
		}
		summary.AssistantMessageCosts[messageIndex] = messageCost
		summary.TotalCost += messageCost.Cost
		summary.HasAnyExactCosts = true
	}
	if assistantMessageCount == 0 {
		summary.AllAssistantCostsExact = false
	}

	threadCostSummaryCache[signature] = summary
	return summary
}

func formatCostUSD(cost float64) string {
	switch {
	case cost >= 1:
		return fmt.Sprintf("$%.2f", cost)
	case cost >= 0.01:
		return fmt.Sprintf("$%.3f", cost)
	case cost >= 0.001:
		return fmt.Sprintf("$%.4f", cost)
	default:
		return fmt.Sprintf("$%.5f", cost)
	}
}

// ─── state helpers ────────────────────────────────────────────────────────────

func cloneMessages(previousMessages []message) []message {
	return append([]message(nil), previousMessages...)
}

func lastPendingMessageIndex(messages []message) int {
	for index := len(messages) - 1; index >= 0; index-- {
		if messages[index].Pending {
			return index
		}
	}
	return -1
}

func appendAssistantMessageDeltaValue(previousMessages []message, deltaText string) []message {
	if len(previousMessages) == 0 {
		return previousMessages
	}
	lastIndex := len(previousMessages) - 1
	updatedMessages := cloneMessages(previousMessages)
	lastMessage := updatedMessages[lastIndex]
	lastMessage.Content += deltaText
	updatedMessages[lastIndex] = lastMessage
	return updatedMessages
}

func appendAssistantMessageDelta(messagesState ui.State[[]message], deltaText string) {
	messagesState.Update(func(previousMessages []message) []message {
		return appendAssistantMessageDeltaValue(previousMessages, deltaText)
	})
}

func appendAssistantThoughtDeltaValue(previousMessages []message, deltaText string) []message {
	if len(previousMessages) == 0 {
		return previousMessages
	}
	lastIndex := len(previousMessages) - 1
	updatedMessages := cloneMessages(previousMessages)
	lastMessage := updatedMessages[lastIndex]
	lastMessage.Thought += deltaText
	lastMessage.ThoughtPending = true
	updatedMessages[lastIndex] = lastMessage
	return updatedMessages
}

func appendAssistantThoughtDelta(messagesState ui.State[[]message], deltaText string) {
	messagesState.Update(func(previousMessages []message) []message {
		return appendAssistantThoughtDeltaValue(previousMessages, deltaText)
	})
}

func markPendingAssistantThoughtCompleteValue(previousMessages []message) []message {
	pendingIndex := lastPendingMessageIndex(previousMessages)
	if pendingIndex < 0 {
		return previousMessages
	}
	updatedMessages := cloneMessages(previousMessages)
	updatedMessages[pendingIndex].ThoughtPending = false
	return updatedMessages
}

func markPendingAssistantThoughtComplete(messagesState ui.State[[]message]) {
	messagesState.Update(func(previousMessages []message) []message {
		return markPendingAssistantThoughtCompleteValue(previousMessages)
	})
}

func finalizePendingMessagesValue(previousMessages []message) []message {
	pendingIndex := lastPendingMessageIndex(previousMessages)
	if pendingIndex < 0 {
		return previousMessages
	}
	updatedMessages := cloneMessages(previousMessages)
	updatedMessages[pendingIndex].Pending = false
	updatedMessages[pendingIndex].ThoughtPending = false
	return updatedMessages
}

func finalizePendingMessages(messagesState ui.State[[]message]) {
	messagesState.Update(func(previousMessages []message) []message {
		return finalizePendingMessagesValue(previousMessages)
	})
}

func finalizePendingMessageWithStatsValue(previousMessages []message, timeToFirstTokenSeconds, tokensPerSecond float64, totalTokenCount int, modelID string, promptTokens, completionTokens int) []message {
	pendingIndex := lastPendingMessageIndex(previousMessages)
	if pendingIndex < 0 {
		return previousMessages
	}
	updatedMessages := cloneMessages(previousMessages)
	updatedMessages[pendingIndex].Pending = false
	updatedMessages[pendingIndex].ThoughtPending = false
	updatedMessages[pendingIndex].ModelID = modelID
	updatedMessages[pendingIndex].PromptTokens = promptTokens
	updatedMessages[pendingIndex].CompletionTokens = completionTokens
	updatedMessages[pendingIndex].TTFT = timeToFirstTokenSeconds
	updatedMessages[pendingIndex].TKPS = tokensPerSecond
	updatedMessages[pendingIndex].Tokens = totalTokenCount
	if completionTokens > 0 {
		updatedMessages[pendingIndex].Tokens = completionTokens
	}
	return updatedMessages
}

func finalizePendingMessageWithStats(messagesState ui.State[[]message], timeToFirstTokenSeconds, tokensPerSecond float64, totalTokenCount int, modelID string, promptTokens, completionTokens int) {
	messagesState.Update(func(previousMessages []message) []message {
		return finalizePendingMessageWithStatsValue(previousMessages, timeToFirstTokenSeconds, tokensPerSecond, totalTokenCount, modelID, promptTokens, completionTokens)
	})
}

func replacePendingMessageWithErrorValue(previousMessages []message, errorMessage string) []message {
	pendingIndex := lastPendingMessageIndex(previousMessages)
	if pendingIndex < 0 {
		updatedMessages := cloneMessages(previousMessages)
		return append(updatedMessages, message{Role: roleAssistant, Content: errorMessage})
	}
	updatedMessages := cloneMessages(previousMessages)
	updatedMessages[pendingIndex] = message{Role: roleAssistant, Content: errorMessage}
	return updatedMessages
}

func replacePendingMessageWithError(messagesState ui.State[[]message], errorMessage string) {
	messagesState.Update(func(previousMessages []message) []message {
		return replacePendingMessageWithErrorValue(previousMessages, errorMessage)
	})
}
