//go:build js && wasm

package app

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

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

type selectedModelCrossTabMessage struct {
	Model string `json:"model"`
}

func cachedRenderedMarkdown(parseMarkdown string) (string, bool) {
	parseRenderedHTML, parseOk := renderedMarkdownCache[parseMarkdown]
	return parseRenderedHTML, parseOk
}

func cacheRenderedMarkdown(parseMarkdown, parseRenderedHTML string) {
	if parseMarkdown == "" || parseRenderedHTML == "" {
		return
	}
	renderedMarkdownCache[parseMarkdown] = parseRenderedHTML
}

func renderMarkdownSync(parseMarkdown string) string {
	if parseRenderedHTML, parseOk := renderedMarkdownCache[parseMarkdown]; parseOk {
		return parseRenderedHTML
	}
	parseRenderedHTML2, parseErr := markdownrender.Render(parseMarkdown)
	if parseErr != nil {
		return parseMarkdown
	}
	renderedMarkdownCache[parseMarkdown] = parseRenderedHTML2
	return parseRenderedHTML2
}

func parseCompletedAssistantMessagesMarkdownSignature(parseMessages []message) string {
	var parseBuilder strings.Builder
	for _, parseMessageItem := range parseMessages {
		if parseMessageItem.Role != roleAssistant || parseMessageItem.Pending {
			continue
		}
		if strings.TrimSpace(parseMessageItem.Content) == "" {
			continue
		}
		parseBuilder.WriteString(parseMessageItem.Content)
		parseBuilder.WriteString("\n\x1f\n")
	}
	return parseBuilder.String()
}

// ─── scroll ───────────────────────────────────────────────────────────────────

func parseScrollMessageListToBottom(parseBehavior ...string) {
	parseDoc, parseErr := interop.GetDocument()
	if parseErr != nil {
		return
	}
	parseResolvedBehavior := ""
	if len(parseBehavior) > 0 {
		parseResolvedBehavior = strings.TrimSpace(parseBehavior[0])
	}
	if parseResolvedBehavior != "" {
		parseScrollAnchorElement, parseFoundScrollAnchor, parseAnchorErr := parseDoc.ElementByID(idScrollAnchor)
		if parseAnchorErr == nil && parseFoundScrollAnchor {
			parseOptions := interop.ScrollIntoViewOptions{
				Behavior: parseResolvedBehavior,
				Block:    "end",
			}
			if parseErr2 := parseScrollAnchorElement.ScrollIntoView(parseOptions); parseErr2 == nil {
				return
			}
		}
	}
	parseMessageListElement, parseFoundMessageList, parseErr := parseDoc.ElementByID(idMessageList)
	if parseErr != nil || !parseFoundMessageList {
		return
	}
	_, parseScrollHeight, _, parseErr := parseMessageListElement.ScrollMetrics()
	if parseErr != nil {
		return
	}
	_ = parseMessageListElement.SetScrollTop(parseScrollHeight)
}

func parseScrollStreamingAssistantBubbleIntoView(parseBehavior string) {
	parseDoc, parseErr := interop.GetDocument()
	if parseErr != nil {
		return
	}
	parseStreamingBubbleElement, parseFoundStreamingBubble, parseErr := parseDoc.ElementByID(idStreamingBubble)
	if parseErr != nil || !parseFoundStreamingBubble {
		parseScrollMessageListToBottom(parseBehavior)
		return
	}
	parseOptions := interop.ScrollIntoViewOptions{
		Behavior: parseBehavior,
		Block:    "start",
	}
	if parseErr2 := parseStreamingBubbleElement.ScrollIntoView(parseOptions); parseErr2 != nil {
		parseScrollMessageListToBottom(parseBehavior)
	}
}

// isMessageListAtScrollBottom returns true when the message list scroll container is
// within scrollThresholdPx of the bottom — the threshold below which
// auto-scroll is active.
func isMessageListAtScrollBottom() bool {
	parseDoc, parseErr := interop.GetDocument()
	if parseErr != nil {
		return true
	}
	parseMessageListElement, parseFoundMessageList, parseErr := parseDoc.ElementByID(idMessageList)
	if parseErr != nil || !parseFoundMessageList {
		return true
	}
	parseScrollTop, parseScrollHeight, parseClientHeight, parseErr := parseMessageListElement.ScrollMetrics()
	if parseErr != nil {
		return true
	}
	return !hasScrollSpaceBelow(parseScrollTop, parseScrollHeight, parseClientHeight, scrollThresholdPx)
}

func parseMessageListHasScrollBelow() bool {
	parseDoc, parseErr := interop.GetDocument()
	if parseErr != nil {
		return false
	}
	parseMessageListElement, parseFoundMessageList, parseErr := parseDoc.ElementByID(idMessageList)
	if parseErr != nil || !parseFoundMessageList {
		return false
	}
	parseScrollTop, parseScrollHeight, parseClientHeight, parseErr := parseMessageListElement.ScrollMetrics()
	if parseErr != nil {
		return false
	}
	return hasScrollSpaceBelow(parseScrollTop, parseScrollHeight, parseClientHeight, scrollThresholdPx)
}

func parseMessageListScrollTop() (float64, bool) {
	parseDoc, parseErr := interop.GetDocument()
	if parseErr != nil {
		return 0, false
	}
	parseMessageListElement, parseFoundMessageList, parseErr := parseDoc.ElementByID(idMessageList)
	if parseErr != nil || !parseFoundMessageList {
		return 0, false
	}
	parseScrollTop, _, _, parseErr := parseMessageListElement.ScrollMetrics()
	if parseErr != nil {
		return 0, false
	}
	return parseScrollTop, true
}

func setMessageListScrollTop(parseScrollTop float64) bool {
	parseDoc, parseErr := interop.GetDocument()
	if parseErr != nil {
		return false
	}
	parseMessageListElement, parseFoundMessageList, parseErr := parseDoc.ElementByID(idMessageList)
	if parseErr != nil || !parseFoundMessageList {
		return false
	}
	if parseErr2 := parseMessageListElement.SetScrollTop(parseScrollTop); parseErr2 != nil {
		return false
	}
	return true
}

func parseFocusChatInput() {
	parseDoc, parseErr := interop.GetDocument()
	if parseErr != nil {
		return
	}
	parseChatInputElement, parseFoundChatInput, parseErr := parseDoc.ElementByID(idChatInput)
	if parseErr != nil || !parseFoundChatInput {
		return
	}
	_ = parseChatInputElement.Focus()
}

func parseScheduleFocusChatInput(parseDelay time.Duration) {
	if _, parseErr := interop.ScheduleTimeout(parseDelay, func() {
		parseFocusChatInput()
	}); parseErr != nil {
		go func() {
			time.Sleep(parseDelay)
			parseFocusChatInput()
		}()
	}
}

func parseCurrentWASMQuerySuffix() string {
	parseEnv, _ := interop.GetWindowEnv()
	parseSuffix, parseOk := parseEnv.LookupString("__gwc_wasm_query")
	if !parseOk {
		return ""
	}
	parseSuffix = strings.TrimSpace(parseSuffix)
	if parseSuffix == "?br=true" {
		return parseSuffix
	}
	return ""
}

// ─── text ─────────────────────────────────────────────────────────────────────

// displayNameInitials returns up to 2 uppercase initials from a display name.
func parseDisplayNameInitials(parseDisplayName string) string {
	parseNameParts := strings.Fields(parseDisplayName)
	parseInitialsText := ""
	for _, parsePart := range parseNameParts {
		parseRunes := []rune(parsePart)
		if len(parseRunes) > 0 {
			parseInitialsText += strings.ToUpper(string(parseRunes[0]))
		}
		if len(parseInitialsText) >= 2 {
			break
		}
	}
	if parseInitialsText == "" {
		return "?"
	}
	return parseInitialsText
}

const managedUserNameMemoryKey = "profile.display_name"

func parseManagedUserNameMemory(parseDisplayName string) editableUserMemory {
	return editableUserMemory{
		Key:          managedUserNameMemoryKey,
		Category:     "identity",
		Summary:      strings.TrimSpace(parseDisplayName),
		Detail:       "Primary display name used for this account.",
		RubricReason: "Pinned from the profile display name so the assistant sees it first.",
	}
}

func isManagedUserNameMemory(parseMemory editableUserMemory) bool {
	return strings.TrimSpace(parseMemory.Key) == managedUserNameMemoryKey
}

func parseEnsureManagedUserNameMemory(parseDisplayName string, parseMemories []editableUserMemory) []editableUserMemory {
	parseFiltered := make([]editableUserMemory, 0, len(parseMemories)+1)
	for _, parseMemory := range parseMemories {
		if isManagedUserNameMemory(parseMemory) {
			continue
		}
		parseFiltered = append(parseFiltered, parseMemory)
	}
	parseTrimmedName := strings.TrimSpace(parseDisplayName)
	if parseTrimmedName == "" {
		return parseFiltered
	}
	return append([]editableUserMemory{parseManagedUserNameMemory(parseTrimmedName)}, parseFiltered...)
}

func parseDefaultModelCatalog() modelCatalog {
	return modelCatalog{DefaultModel: defaultModel}
}

func parseModelOptionByID(parseModelID string, parseModels []modelOption) (modelOption, bool) {
	parseTrimmedModelID := strings.TrimSpace(parseModelID)
	for _, parseOption := range parseModels {
		if parseOption.ID == parseTrimmedModelID {
			return parseOption, true
		}
	}
	return modelOption{}, false
}

func parseProviderOptionsForModels(parseModels []modelOption) []providerOption {
	parseOptions := make([]providerOption, 0, len(parseModels))
	parseSeenProviders := map[string]struct{}{}
	for _, parseOption := range parseModels {
		parseProviderID := strings.TrimSpace(parseOption.Capabilities.ProviderID)
		if parseProviderID == "" {
			parseProviderID = "default"
		}
		if _, parseSeen := parseSeenProviders[parseProviderID]; parseSeen {
			continue
		}
		parseProviderLabel := strings.TrimSpace(parseOption.Capabilities.ProviderLabel)
		if parseProviderLabel == "" {
			parseProviderLabel = strings.ToUpper(parseProviderID)
		}
		parseOptions = append(parseOptions, providerOption{ID: parseProviderID, Label: parseProviderLabel})
		parseSeenProviders[parseProviderID] = struct{}{}
	}
	return parseOptions
}

func parseProviderForModel(parseModelID string, parseModels []modelOption, parseFallback string) providerOption {
	parseResolvedModelID := parseNormalizeSelectedModelID(parseModelID, parseModels, parseFallback)
	parseOption, parseOk := parseModelOptionByID(parseResolvedModelID, parseModels)
	if !parseOk {
		return providerOption{}
	}
	parseProviderID := strings.TrimSpace(parseOption.Capabilities.ProviderID)
	if parseProviderID == "" {
		parseProviderID = "default"
	}
	parseProviderLabel := strings.TrimSpace(parseOption.Capabilities.ProviderLabel)
	if parseProviderLabel == "" {
		parseProviderLabel = strings.ToUpper(parseProviderID)
	}
	return providerOption{ID: parseProviderID, Label: parseProviderLabel}
}

func parseModelsForProvider(parseModels []modelOption, parseProviderID string) []modelOption {
	parseTrimmedProviderID := strings.TrimSpace(parseProviderID)
	if parseTrimmedProviderID == "" {
		return append([]modelOption(nil), parseModels...)
	}
	parseFiltered := make([]modelOption, 0, len(parseModels))
	for _, parseOption := range parseModels {
		parseCandidateProviderID := strings.TrimSpace(parseOption.Capabilities.ProviderID)
		if parseCandidateProviderID == "" {
			parseCandidateProviderID = "default"
		}
		if parseCandidateProviderID == parseTrimmedProviderID {
			parseFiltered = append(parseFiltered, parseOption)
		}
	}
	if len(parseFiltered) == 0 {
		return append([]modelOption(nil), parseModels...)
	}
	return parseFiltered
}

func parseDefaultModelForProvider(parseProviderID string, parseModels []modelOption, parseFallback string) string {
	parseFiltered := parseModelsForProvider(parseModels, parseProviderID)
	if len(parseFiltered) == 0 {
		return parseNormalizeSelectedModelID(parseFallback, parseModels, parseFallback)
	}
	parseResolvedFallback := parseNormalizeSelectedModelID(parseFallback, parseModels, parseFallback)
	if parseOption, parseOk := parseModelOptionByID(parseResolvedFallback, parseModels); parseOk {
		parseCandidateProviderID := strings.TrimSpace(parseOption.Capabilities.ProviderID)
		if parseCandidateProviderID == "" {
			parseCandidateProviderID = "default"
		}
		if parseCandidateProviderID == strings.TrimSpace(parseProviderID) {
			return parseOption.ID
		}
	}
	return parseFiltered[0].ID
}

func parseNormalizeSelectedModelID(parseModelID string, parseModels []modelOption, parseFallback string) string {
	parseModelID = strings.TrimSpace(parseModelID)
	switch parseModelID {
	case "gpt-5.4-2026-03-17":
		parseModelID = "gpt-5.4"
	case "gpt-5.4-mini-2026-03-17":
		parseModelID = "gpt-5.4-mini"
	case "gpt-5.4-nano-2026-03-17":
		parseModelID = "gpt-5.4-nano"
	}
	parseFallback = strings.TrimSpace(parseFallback)
	if parseModelID == "" {
		if _, parseOk := parseModelOptionByID(parseFallback, parseModels); parseOk {
			return parseFallback
		}
		if parseFallback != "" && len(parseModels) == 0 {
			return parseFallback
		}
		if len(parseModels) == 0 {
			return ""
		}
		return parseModels[0].ID
	}
	if len(parseModels) == 0 {
		return parseModelID
	}
	if _, parseOk2 := parseModelOptionByID(parseModelID, parseModels); parseOk2 {
		return parseModelID
	}
	if _, parseOk3 := parseModelOptionByID(parseFallback, parseModels); parseOk3 {
		return parseFallback
	}
	if parseFallback != "" {
		return parseFallback
	}
	return parseModels[0].ID
}

func parseRecoverPersistedModelSelection(parsePersistedModel string, parseModels []modelOption) (string, bool) {
	parseCanonicalModel := parseNormalizeSelectedModelID(parsePersistedModel, nil, "")
	if parseCanonicalModel != "" {
		if _, parseOk := parseModelOptionByID(parseCanonicalModel, parseModels); parseOk {
			return parseCanonicalModel, false
		}
	}
	if len(parseModels) == 0 {
		return parseCanonicalModel, false
	}
	return parseModels[0].ID, true
}

func parseSelectedModelCrossTabChannelName(parseSessionEmail string) string {
	parseNormalizedEmail := strings.TrimSpace(strings.ToLower(parseSessionEmail))
	if parseNormalizedEmail == "" {
		return crossTabChannelSelectedModel
	}
	var parseBuilder strings.Builder
	parseBuilder.WriteString(crossTabChannelSelectedModel)
	parseBuilder.WriteString(":")
	for _, parseR := range parseNormalizedEmail {
		switch {
		case parseR >= 'a' && parseR <= 'z':
			parseBuilder.WriteRune(parseR)
		case parseR >= '0' && parseR <= '9':
			parseBuilder.WriteRune(parseR)
		default:
			parseBuilder.WriteRune('-')
		}
	}
	return parseBuilder.String()
}

func parseSelectedModelForConversation(parseMessages []message, parseModels []modelOption, parseFallback string) string {
	for parseIdx := len(parseMessages) - 1; parseIdx >= 0; parseIdx-- {
		parseMessageItem := parseMessages[parseIdx]
		if parseMessageItem.Role == roleSwitch {
			parseSwitchModel := strings.TrimSpace(parseMessageItem.Content)
			if parseSwitchModel != "" {
				return parseNormalizeSelectedModelID(parseSwitchModel, parseModels, parseFallback)
			}
		}
		if parseMessageItem.Role != roleAssistant {
			continue
		}
		parseModelID := strings.TrimSpace(parseMessageItem.ModelID)
		if parseModelID != "" {
			return parseNormalizeSelectedModelID(parseModelID, parseModels, parseFallback)
		}
	}
	return parseNormalizeSelectedModelID(parseFallback, parseModels, parseFallback)
}

func parseNormalizeSelectedToneID(parseToneID string) string {
	parseToneID = strings.TrimSpace(parseToneID)
	if parseToneID == "" {
		return defaultTone
	}
	for _, parseToneOption := range availableTones {
		if parseToneOption.ID == parseToneID {
			return parseToneID
		}
	}
	return defaultTone
}

func parseNormalizeSelectedThinkingEffort(parseEffort string) string {
	parseEffort = strings.TrimSpace(strings.ToLower(parseEffort))
	if parseEffort == "" {
		return defaultThinkingEffort
	}
	for _, parseOption := range availableThinkingEfforts {
		if parseOption.ID == parseEffort {
			return parseEffort
		}
	}
	return defaultThinkingEffort
}

func parseModelLabelForID(parseModelID string, parseModels []modelOption) string {
	if parseOption, parseOk := parseModelOptionByID(parseModelID, parseModels); parseOk {
		return parseOption.Label
	}
	if strings.TrimSpace(parseModelID) == "" {
		return ""
	}
	return parseModelID
}

func parseModelSupportsThinking(parseModelID string, parseModels []modelOption, parseFallback string) bool {
	parseResolvedModelID := parseNormalizeSelectedModelID(parseModelID, parseModels, parseFallback)
	parseOption, parseOk := parseModelOptionByID(parseResolvedModelID, parseModels)
	return parseOk && parseOption.Capabilities.SupportsThinking
}

func parseModelSupportsSpeech(parseModelID string, parseModels []modelOption, parseFallback string) bool {
	parseResolvedModelID := parseNormalizeSelectedModelID(parseModelID, parseModels, parseFallback)
	parseOption, parseOk := parseModelOptionByID(parseResolvedModelID, parseModels)
	return parseOk && parseOption.Capabilities.SupportsSpeech
}

func parseModelSupportsCapability(parseModelID string, parseModels []modelOption, parseFallback string, parseCapability string) bool {
	switch strings.TrimSpace(strings.ToLower(parseCapability)) {
	case "thinking":
		return parseModelSupportsThinking(parseModelID, parseModels, parseFallback)
	case "speech":
		return parseModelSupportsSpeech(parseModelID, parseModels, parseFallback)
	default:
		return true
	}
}

func filterModelsByCapability(parseModels []modelOption, parseCapability string) []modelOption {
	parseCapability = strings.TrimSpace(strings.ToLower(parseCapability))
	if parseCapability == "" {
		return append([]modelOption(nil), parseModels...)
	}
	parseFiltered := make([]modelOption, 0, len(parseModels))
	for _, parseOption := range parseModels {
		if parseModelSupportsCapability(parseOption.ID, parseModels, "", parseCapability) {
			parseFiltered = append(parseFiltered, parseOption)
		}
	}
	if len(parseFiltered) == 0 {
		return append([]modelOption(nil), parseModels...)
	}
	return parseFiltered
}

func parseSameModelOptions(parseLeft, parseRight []modelOption) bool {
	if len(parseLeft) != len(parseRight) {
		return false
	}
	for parseIdx := range parseLeft {
		if parseLeft[parseIdx] != parseRight[parseIdx] {
			return false
		}
	}
	return true
}

func parsePreviewLogText(parseText string, parseMaxLen int) string {
	parseTrimmedText := strings.TrimSpace(parseText)
	if parseMaxLen <= 0 || len(parseTrimmedText) <= parseMaxLen {
		return parseTrimmedText
	}
	if parseMaxLen <= 1 {
		return parseTrimmedText[:parseMaxLen]
	}
	return parseTrimmedText[:parseMaxLen-1] + "…"
}

func parseThoughtSectionKey(parseMessageIndex, parseSectionIndex int, parseHeading string) string {
	return fmt.Sprintf("%d:%d:%s", parseMessageIndex, parseSectionIndex, strings.TrimSpace(parseHeading))
}

func parseMaterializeThoughtSections(parseMessageIndex int, parseCachedSections []thoughtSection) []thoughtSection {
	if len(parseCachedSections) == 0 {
		return nil
	}
	parseSections := make([]thoughtSection, len(parseCachedSections))
	for parseIdx, parseSection := range parseCachedSections {
		parseSections[parseIdx] = thoughtSection{
			Key:     parseThoughtSectionKey(parseMessageIndex, parseIdx, parseSection.Heading),
			Heading: parseSection.Heading,
			Body:    parseSection.Body,
		}
	}
	return parseSections
}

func parseThoughtHeading(parseLine string) (string, bool) {
	parseTrimmedLine := strings.TrimSpace(parseLine)
	if !strings.HasPrefix(parseTrimmedLine, "**") || !strings.HasSuffix(parseTrimmedLine, "**") || len(parseTrimmedLine) <= 4 {
		return "", false
	}
	parseHeading := strings.TrimSpace(parseTrimmedLine[2 : len(parseTrimmedLine)-2])
	return parseHeading, parseHeading != ""
}

func parseThoughtSections(parseMessageIndex int, parseThoughtText string) []thoughtSection {
	parseNormalizedText := strings.TrimSpace(strings.ReplaceAll(parseThoughtText, "\r\n", "\n"))
	if parseNormalizedText == "" {
		return nil
	}
	if parseCachedSections, parseOk := thoughtSectionsCache[parseNormalizedText]; parseOk {
		return parseMaterializeThoughtSections(parseMessageIndex, parseCachedSections)
	}

	parseLines := strings.Split(parseNormalizedText, "\n")
	parseSections := make([]thoughtSection, 0, 4)
	parseCurrentHeading := ""
	parseCurrentBodyLines := make([]string, 0, len(parseLines))

	parseFlushCurrent := func() {
		if parseCurrentHeading == "" && len(parseCurrentBodyLines) == 0 {
			return
		}
		parseHeading := strings.TrimSpace(parseCurrentHeading)
		parseBody := strings.TrimSpace(strings.Join(parseCurrentBodyLines, "\n"))
		if parseHeading == "" {
			parseHeading = "Thinking"
		}
		parseSections = append(parseSections, thoughtSection{
			Heading: parseHeading,
			Body:    parseBody,
		})
		parseCurrentHeading = ""
		parseCurrentBodyLines = parseCurrentBodyLines[:0]
	}

	for _, parseLine := range parseLines {
		parseTrimmedLine := strings.TrimSpace(parseLine)
		if len(parseSections) == 0 && parseCurrentHeading == "" && len(parseCurrentBodyLines) == 0 && strings.EqualFold(parseTrimmedLine, "thinking") {
			continue
		}
		if parseHeading2, parseOk2 := parseThoughtHeading(parseTrimmedLine); parseOk2 {
			parseFlushCurrent()
			parseCurrentHeading = parseHeading2
			continue
		}
		parseCurrentBodyLines = append(parseCurrentBodyLines, parseLine)
	}
	parseFlushCurrent()

	if len(parseSections) == 0 {
		parseSections = []thoughtSection{{
			Heading: "Thinking",
			Body:    parseNormalizedText,
		}}
	}

	thoughtSectionsCache[parseNormalizedText] = parseSections
	return parseMaterializeThoughtSections(parseMessageIndex, parseSections)
}

func parseExactAssistantMessageCost(parseModelID string, parseModels []modelOption, parsePromptTokens, parseCompletionTokens int) (assistantMessageCost, bool) {
	parseTrimmedModelID := strings.TrimSpace(parseModelID)
	parseOption, parseFoundOption := parseModelOptionByID(parseTrimmedModelID, parseModels)
	if !parseFoundOption || (parsePromptTokens <= 0 && parseCompletionTokens <= 0) {
		return assistantMessageCost{
			ModelID:          parseTrimmedModelID,
			PromptTokens:     parsePromptTokens,
			CompletionTokens: parseCompletionTokens,
		}, false
	}
	parsePricing := parseOption.Pricing
	parseCost := (float64(parsePromptTokens) * parsePricing.InputDollarsPerMillion / 1_000_000) +
		(float64(parseCompletionTokens) * parsePricing.OutputDollarsPerMillion / 1_000_000)
	return assistantMessageCost{
		ModelID:          parseTrimmedModelID,
		PromptTokens:     parsePromptTokens,
		CompletionTokens: parseCompletionTokens,
		Cost:             parseCost,
	}, true
}

func parseThreadCostSummarySignature(parseMessages []message, parseModels []modelOption) string {
	var parseBuilder strings.Builder
	for _, parseOption := range parseModels {
		parseBuilder.WriteString(fmt.Sprintf("model|%s|%.6f|%.6f|%s\n", parseOption.ID, parseOption.Pricing.InputDollarsPerMillion, parseOption.Pricing.OutputDollarsPerMillion, parseOption.Pricing.Currency))
	}
	parseBuilder.WriteString("--\n")
	for parseMessageIndex, parseMessageItem := range parseMessages {
		if parseMessageItem.Role != roleAssistant || parseMessageItem.Pending {
			continue
		}
		if strings.TrimSpace(parseMessageItem.Content) == "" {
			continue
		}
		parseBuilder.WriteString(fmt.Sprintf("%d|%s|%d|%d\n", parseMessageIndex, parseMessageItem.ModelID, parseMessageItem.PromptTokens, parseMessageItem.CompletionTokens))
	}
	return parseBuilder.String()
}

func parseDeriveThreadCostSummary(parseMessages []message, parseModels []modelOption) threadCostSummary {
	parseSignature := parseThreadCostSummarySignature(parseMessages, parseModels)
	if parseCachedSummary, parseOk := threadCostSummaryCache[parseSignature]; parseOk {
		return parseCachedSummary
	}
	parseSummary := threadCostSummary{
		AssistantMessageCosts:  make(map[int]assistantMessageCost),
		AllAssistantCostsExact: true,
	}
	parseAssistantMessageCount := 0

	for parseMessageIndex, parseMessageItem := range parseMessages {
		if parseMessageItem.Role != roleAssistant || parseMessageItem.Pending {
			continue
		}
		if strings.TrimSpace(parseMessageItem.Content) == "" {
			continue
		}
		parseAssistantMessageCount++
		parseMessageCost, hasExactCost := parseExactAssistantMessageCost(parseMessageItem.ModelID, parseModels, parseMessageItem.PromptTokens, parseMessageItem.CompletionTokens)
		if !hasExactCost {
			parseSummary.AllAssistantCostsExact = false
			continue
		}
		parseSummary.AssistantMessageCosts[parseMessageIndex] = parseMessageCost
		parseSummary.TotalCost += parseMessageCost.Cost
		parseSummary.HasAnyExactCosts = true
	}
	if parseAssistantMessageCount == 0 {
		parseSummary.AllAssistantCostsExact = false
	}

	threadCostSummaryCache[parseSignature] = parseSummary
	return parseSummary
}

func formatCostUSD(parseCost float64) string {
	switch {
	case parseCost >= 1:
		return fmt.Sprintf("$%.2f", parseCost)
	case parseCost >= 0.01:
		return fmt.Sprintf("$%.3f", parseCost)
	case parseCost >= 0.001:
		return fmt.Sprintf("$%.4f", parseCost)
	default:
		return fmt.Sprintf("$%.5f", parseCost)
	}
}

func formatPercentValue(parseValue float64) string {
	parseRounded := math.Round(parseValue)
	if math.Abs(parseValue-parseRounded) < 0.000001 {
		return fmt.Sprintf("%.0f", parseRounded)
	}
	if parseValue >= 10 {
		return fmt.Sprintf("%.1f", parseValue)
	}
	return fmt.Sprintf("%.2f", parseValue)
}

func parseSanitizeUsagePremiumPercent(parseValue, parseFallback float64) float64 {
	if math.IsNaN(parseValue) || math.IsInf(parseValue, 0) || parseValue < 0 {
		return parseFallback
	}
	if parseValue > 1000 {
		return 1000
	}
	return parseValue
}

func parseConfiguredUsagePremiumPercent() float64 {
	parseFallback := parseSanitizeUsagePremiumPercent(defaultUsagePremiumPercent, 5.0)
	parseEnv, parseErr := interop.GetWindowEnv()
	if parseErr != nil {
		return parseFallback
	}
	parseRawValue, parseOk := parseEnv.LookupString(usagePremiumWindowKey)
	if !parseOk {
		return parseFallback
	}
	parseParsed, parseErr := strconv.ParseFloat(strings.TrimSpace(parseRawValue), 64)
	if parseErr != nil {
		return parseFallback
	}
	return parseSanitizeUsagePremiumPercent(parseParsed, parseFallback)
}

func applyUsagePremium(parseUsageCost, parsePremiumPercent float64) (parsePremiumCost float64, parseTotalCost float64) {
	parseNormalizedUsageCost := parseUsageCost
	if parseNormalizedUsageCost < 0 {
		parseNormalizedUsageCost = 0
	}
	parseNormalizedPremiumPct := parseSanitizeUsagePremiumPercent(parsePremiumPercent, defaultUsagePremiumPercent)
	parsePremiumCost = parseNormalizedUsageCost * parseNormalizedPremiumPct / 100
	parseTotalCost = parseNormalizedUsageCost + parsePremiumCost
	return parsePremiumCost, parseTotalCost
}

func parseDeriveAccountCostSummary(parseThreadSummaries []threadCostSummary, parsePremiumPercent float64, parseFailedThreadLookups int) accountCostSummary {
	parseNormalizedPremiumPct := parseSanitizeUsagePremiumPercent(parsePremiumPercent, defaultUsagePremiumPercent)
	parseSummary := accountCostSummary{
		ThreadCount:         len(parseThreadSummaries) + parseMaxInt(0, parseFailedThreadLookups),
		PremiumPercent:      parseNormalizedPremiumPct,
		AllThreadCostsExact: parseFailedThreadLookups == 0,
		FailedThreadLookups: parseMaxInt(0, parseFailedThreadLookups),
	}
	if parseSummary.FailedThreadLookups > 0 {
		parseSummary.HasCoverageGaps = true
	}
	for _, parseThreadSummary := range parseThreadSummaries {
		if parseThreadSummary.HasAnyExactCosts {
			parseSummary.HasAnyExactCosts = true
			parseSummary.ExactThreadCostCount++
			parseSummary.UsageCost += parseThreadSummary.TotalCost
		}
		if !parseThreadSummary.AllAssistantCostsExact {
			parseSummary.AllThreadCostsExact = false
			parseSummary.HasCoverageGaps = true
		}
	}
	if parseSummary.ThreadCount == 0 {
		parseSummary.AllThreadCostsExact = false
	}
	parseSummary.PremiumCost, parseSummary.TotalCost = applyUsagePremium(parseSummary.UsageCost, parseNormalizedPremiumPct)
	return parseSummary
}

// ─── state helpers ────────────────────────────────────────────────────────────

func parseCloneMessages(parsePreviousMessages []message) []message {
	return append([]message(nil), parsePreviousMessages...)
}

func parseLastPendingMessageIndex(parseMessages []message) int {
	for parseIndex := len(parseMessages) - 1; parseIndex >= 0; parseIndex-- {
		if parseMessages[parseIndex].Pending {
			return parseIndex
		}
	}
	return -1
}

func parseAppendAssistantMessageDeltaValue(parsePreviousMessages []message, parseDeltaText string) []message {
	if len(parsePreviousMessages) == 0 {
		return parsePreviousMessages
	}
	parseLastIndex := len(parsePreviousMessages) - 1
	parseUpdatedMessages := parseCloneMessages(parsePreviousMessages)
	parseLastMessage := parseUpdatedMessages[parseLastIndex]
	parseLastMessage.Content += parseDeltaText
	parseUpdatedMessages[parseLastIndex] = parseLastMessage
	return parseUpdatedMessages
}

func parseAppendAssistantMessageDelta(parseMessagesState ui.State[[]message], parseDeltaText string) {
	parseMessagesState.Update(func(parsePreviousMessages []message) []message {
		return parseAppendAssistantMessageDeltaValue(parsePreviousMessages, parseDeltaText)
	})
}

func parseAppendAssistantThoughtDeltaValue(parsePreviousMessages []message, parseDeltaText string) []message {
	if len(parsePreviousMessages) == 0 {
		return parsePreviousMessages
	}
	parseLastIndex := len(parsePreviousMessages) - 1
	parseUpdatedMessages := parseCloneMessages(parsePreviousMessages)
	parseLastMessage := parseUpdatedMessages[parseLastIndex]
	parseLastMessage.Thought += parseDeltaText
	parseLastMessage.ThoughtPending = true
	parseUpdatedMessages[parseLastIndex] = parseLastMessage
	return parseUpdatedMessages
}

func parseAppendAssistantThoughtDelta(parseMessagesState ui.State[[]message], parseDeltaText string) {
	parseMessagesState.Update(func(parsePreviousMessages []message) []message {
		return parseAppendAssistantThoughtDeltaValue(parsePreviousMessages, parseDeltaText)
	})
}

func parseMarkPendingAssistantThoughtCompleteValue(parsePreviousMessages []message) []message {
	parsePendingIndex := parseLastPendingMessageIndex(parsePreviousMessages)
	if parsePendingIndex < 0 {
		return parsePreviousMessages
	}
	parseUpdatedMessages := parseCloneMessages(parsePreviousMessages)
	parseUpdatedMessages[parsePendingIndex].ThoughtPending = false
	return parseUpdatedMessages
}

func parseMarkPendingAssistantThoughtComplete(parseMessagesState ui.State[[]message]) {
	parseMessagesState.Update(func(parsePreviousMessages []message) []message {
		return parseMarkPendingAssistantThoughtCompleteValue(parsePreviousMessages)
	})
}

func parseFinalizePendingMessagesValue(parsePreviousMessages []message) []message {
	parsePendingIndex := parseLastPendingMessageIndex(parsePreviousMessages)
	if parsePendingIndex < 0 {
		return parsePreviousMessages
	}
	parseUpdatedMessages := parseCloneMessages(parsePreviousMessages)
	parseUpdatedMessages[parsePendingIndex].Pending = false
	parseUpdatedMessages[parsePendingIndex].ThoughtPending = false
	return parseUpdatedMessages
}

func parseFinalizePendingMessages(parseMessagesState ui.State[[]message]) {
	parseMessagesState.Update(func(parsePreviousMessages []message) []message {
		return parseFinalizePendingMessagesValue(parsePreviousMessages)
	})
}

func parseFinalizePendingMessageWithStatsValue(parsePreviousMessages []message, parseTimeToFirstTokenSeconds, parseTokensPerSecond float64, parseTotalTokenCount int, parseModelID string, parsePromptTokens, parseCompletionTokens int) []message {
	parsePendingIndex := parseLastPendingMessageIndex(parsePreviousMessages)
	if parsePendingIndex < 0 {
		return parsePreviousMessages
	}
	parseUpdatedMessages := parseCloneMessages(parsePreviousMessages)
	parseUpdatedMessages[parsePendingIndex].Pending = false
	parseUpdatedMessages[parsePendingIndex].ThoughtPending = false
	parseUpdatedMessages[parsePendingIndex].ModelID = parseModelID
	parseUpdatedMessages[parsePendingIndex].PromptTokens = parsePromptTokens
	parseUpdatedMessages[parsePendingIndex].CompletionTokens = parseCompletionTokens
	parseUpdatedMessages[parsePendingIndex].TTFT = parseTimeToFirstTokenSeconds
	parseUpdatedMessages[parsePendingIndex].TKPS = parseTokensPerSecond
	parseUpdatedMessages[parsePendingIndex].Tokens = parseTotalTokenCount
	if parseCompletionTokens > 0 {
		parseUpdatedMessages[parsePendingIndex].Tokens = parseCompletionTokens
	}
	return parseUpdatedMessages
}

func parseFinalizePendingMessageWithStats(parseMessagesState ui.State[[]message], parseTimeToFirstTokenSeconds, parseTokensPerSecond float64, parseTotalTokenCount int, parseModelID string, parsePromptTokens, parseCompletionTokens int) {
	parseMessagesState.Update(func(parsePreviousMessages []message) []message {
		return parseFinalizePendingMessageWithStatsValue(parsePreviousMessages, parseTimeToFirstTokenSeconds, parseTokensPerSecond, parseTotalTokenCount, parseModelID, parsePromptTokens, parseCompletionTokens)
	})
}

func parseReplacePendingMessageWithErrorValue(parsePreviousMessages []message, parseErrorMessage string) []message {
	parsePendingIndex := parseLastPendingMessageIndex(parsePreviousMessages)
	if parsePendingIndex < 0 {
		parseUpdatedMessages := parseCloneMessages(parsePreviousMessages)
		return append(parseUpdatedMessages, message{Role: roleAssistant, Content: parseErrorMessage})
	}
	parseUpdatedMessages2 := parseCloneMessages(parsePreviousMessages)
	parseUpdatedMessages2[parsePendingIndex] = message{Role: roleAssistant, Content: parseErrorMessage}
	return parseUpdatedMessages2
}

func parseReplacePendingMessageWithError(parseMessagesState ui.State[[]message], parseErrorMessage string) {
	parseMessagesState.Update(func(parsePreviousMessages []message) []message {
		return parseReplacePendingMessageWithErrorValue(parsePreviousMessages, parseErrorMessage)
	})
}
