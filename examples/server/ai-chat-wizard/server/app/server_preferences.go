package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/server/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

func parseCapabilityStatusError(parseErr error) error {
	var parseCapabilityErr *provider.UnsupportedCapabilityError
	if !errors.As(parseErr, &parseCapabilityErr) {
		return nil
	}
	return status.Errorf(codes.FailedPrecondition, "model %q does not support %s", parseCapabilityErr.Model, parseCapabilityErr.Capability)
}

func parseUserFacingStreamError(parseProviderID string, parseModelID string, parseErr error) string {
	parseProviderID = strings.TrimSpace(parseProviderID)
	parseModelID = strings.TrimSpace(parseModelID)
	parseRaw := strings.TrimSpace(parseErr.Error())
	if parseRaw == "" {
		if parseProviderID == "" {
			return "model provider stream failed"
		}
		return fmt.Sprintf("%s stream failed", parseProviderID)
	}
	if parseProviderID == "cerebras" && strings.Contains(parseRaw, "404 Not Found") {
		if parseModelID != "" {
			return fmt.Sprintf("Cerebras model %q is unavailable for this API key. Try llama3.1-8b or another available Cerebras model.", parseModelID)
		}
		return "Selected Cerebras model is unavailable for this API key. Try llama3.1-8b or another available Cerebras model."
	}
	return parseSanitizeProviderErrorMessage(parseProviderID, parseRaw)
}

func parseNormalizeSelectedModelID(parseModelID string) string {
	parseModelID = strings.TrimSpace(parseModelID)
	switch parseModelID {
	case "", modelGPT54Mini, "gpt-5.4-mini-2026-03-17":
		return modelGPT54Mini
	case modelGPT54, "gpt-5.4-2026-03-17":
		return modelGPT54
	case modelGPT54Nano, "gpt-5.4-nano-2026-03-17":
		return modelGPT54Nano
	default:
		return parseModelID
	}
}

// parseNormalizeOptionalSelectedModelID normalizes aliases while preserving explicit-empty values.
func parseNormalizeOptionalSelectedModelID(parseModelID string) string {
	if strings.TrimSpace(parseModelID) == "" {
		return ""
	}
	return parseNormalizeSelectedModelID(parseModelID)
}

// SetSelectedTone stores the current user's selected tone.
func (parseS *chatServer) SetSelectedTone(parseCtx context.Context, parseReq *wrapperspb.StringValue) (*emptypb.Empty, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "SetSelectedTone"))
	parseLogger = parseLogger.With(parseResolveTraceabilityAttrs(parseCtx)...)
	parseSelectedTone := parseNormalizeSelectedToneID(parseReq.GetValue())
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Error("rpc.SetSelectedTone: store unavailable; rejecting settings write")
		return nil, status.Error(codes.Unavailable, "settings persistence unavailable")
	}
	if parseErr2 := parseS.store.setSelectedTone(parseUserID, parseSelectedTone); parseErr2 != nil {
		parseLogger.Error("rpc.SetSelectedTone: db upsert failed", slog.String("error", parseErr2.Error()))
		return nil, parseBuildSanitizedInternalStatus(parseCtx, "set selected tone", parseErr2)
	}
	parseLogger.Info("rpc.SetSelectedTone: complete", slog.String("tone", parseSelectedTone))
	return &emptypb.Empty{}, nil
}

// GetSelectedTone returns the current user's selected tone.
func (parseS *chatServer) GetSelectedTone(parseCtx context.Context, _ *emptypb.Empty) (*wrapperspb.StringValue, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetSelectedTone"))
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetSelectedTone: store unavailable - returning default")
		return wrapperspb.String(defaultToneID), nil
	}
	parseSelectedTone, parseErr := parseS.store.getSelectedTone(parseUserID, defaultToneID)
	if parseErr != nil {
		parseLogger.Error("rpc.GetSelectedTone: db query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "get selected tone: %v", parseErr)
	}
	parseSelectedTone = parseNormalizeSelectedToneID(parseSelectedTone)
	parseLogger.Info("rpc.GetSelectedTone: complete", slog.String("tone", parseSelectedTone))
	return wrapperspb.String(parseSelectedTone), nil
}

// SetSelectedThinkingEnabled stores the current user's thinking toggle.
func (parseS *chatServer) SetSelectedThinkingEnabled(parseCtx context.Context, parseReq *wrapperspb.BoolValue) (*emptypb.Empty, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "SetSelectedThinkingEnabled"))
	parseLogger = parseLogger.With(parseResolveTraceabilityAttrs(parseCtx)...)
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Error("rpc.SetSelectedThinkingEnabled: store unavailable; rejecting settings write")
		return nil, status.Error(codes.Unavailable, "settings persistence unavailable")
	}
	if parseErr2 := parseS.store.setSelectedThinkingEnabled(parseUserID, parseReq.GetValue()); parseErr2 != nil {
		parseLogger.Error("rpc.SetSelectedThinkingEnabled: db upsert failed", slog.String("error", parseErr2.Error()))
		return nil, parseBuildSanitizedInternalStatus(parseCtx, "set selected thinking enabled", parseErr2)
	}
	parseLogger.Info("rpc.SetSelectedThinkingEnabled: complete", slog.Bool("enabled", parseReq.GetValue()))
	return &emptypb.Empty{}, nil
}

// GetSelectedThinkingEnabled returns the current user's thinking toggle.
func (parseS *chatServer) GetSelectedThinkingEnabled(parseCtx context.Context, _ *emptypb.Empty) (*wrapperspb.BoolValue, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetSelectedThinkingEnabled"))
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetSelectedThinkingEnabled: store unavailable - returning default")
		return wrapperspb.Bool(defaultThinkingEnabled), nil
	}
	parseEnabled, parseErr := parseS.store.getSelectedThinkingEnabled(parseUserID, defaultThinkingEnabled)
	if parseErr != nil {
		parseLogger.Error("rpc.GetSelectedThinkingEnabled: db query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "get selected thinking enabled: %v", parseErr)
	}
	parseLogger.Info("rpc.GetSelectedThinkingEnabled: complete", slog.Bool("enabled", parseEnabled))
	return wrapperspb.Bool(parseEnabled), nil
}

// SetSelectedThinkingEffort stores the current user's thinking-effort preference.
func (parseS *chatServer) SetSelectedThinkingEffort(parseCtx context.Context, parseReq *wrapperspb.StringValue) (*emptypb.Empty, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "SetSelectedThinkingEffort"))
	parseLogger = parseLogger.With(parseResolveTraceabilityAttrs(parseCtx)...)
	parseSelectedThinkingEffort := parseNormalizeSelectedThinkingEffort(parseReq.GetValue())
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Error("rpc.SetSelectedThinkingEffort: store unavailable; rejecting settings write")
		return nil, status.Error(codes.Unavailable, "settings persistence unavailable")
	}
	if parseErr2 := parseS.store.setSelectedThinkingEffort(parseUserID, parseSelectedThinkingEffort); parseErr2 != nil {
		parseLogger.Error("rpc.SetSelectedThinkingEffort: db upsert failed", slog.String("error", parseErr2.Error()))
		return nil, parseBuildSanitizedInternalStatus(parseCtx, "set selected thinking effort", parseErr2)
	}
	parseLogger.Info("rpc.SetSelectedThinkingEffort: complete", slog.String("effort", parseSelectedThinkingEffort))
	return &emptypb.Empty{}, nil
}

// GetSelectedThinkingEffort returns the current user's thinking-effort preference.
func (parseS *chatServer) GetSelectedThinkingEffort(parseCtx context.Context, _ *emptypb.Empty) (*wrapperspb.StringValue, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetSelectedThinkingEffort"))
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetSelectedThinkingEffort: store unavailable - returning default")
		return wrapperspb.String(defaultThinkingEffort), nil
	}
	parseSelectedThinkingEffort, parseErr := parseS.store.getSelectedThinkingEffort(parseUserID, defaultThinkingEffort)
	if parseErr != nil {
		parseLogger.Error("rpc.GetSelectedThinkingEffort: db query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "get selected thinking effort: %v", parseErr)
	}
	parseSelectedThinkingEffort = parseNormalizeSelectedThinkingEffort(parseSelectedThinkingEffort)
	parseLogger.Info("rpc.GetSelectedThinkingEffort: complete", slog.String("effort", parseSelectedThinkingEffort))
	return wrapperspb.String(parseSelectedThinkingEffort), nil
}

// SetCustomSystemPrompt stores the current user's custom system prompt.
func (parseS *chatServer) SetCustomSystemPrompt(parseCtx context.Context, parseReq *wrapperspb.StringValue) (*emptypb.Empty, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "SetCustomSystemPrompt"))
	parseLogger = parseLogger.With(parseResolveTraceabilityAttrs(parseCtx)...)
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Error("rpc.SetCustomSystemPrompt: store unavailable; rejecting prompt write")
		return nil, parseBuildStoreUnavailableRPCStatus(parseCtx, "/chat.ChatService/SetCustomSystemPrompt", "system prompt persistence unavailable")
	}
	parseCustomPrompt := parseNormalizeCustomSystemPrompt(parseReq.GetValue())
	if parseErr2 := parseS.store.setSelectedSystemPrompt(parseUserID, parseCustomPrompt); parseErr2 != nil {
		parseLogger.Error("rpc.SetCustomSystemPrompt: db upsert failed", slog.String("error", parseErr2.Error()))
		return nil, parseBuildSanitizedInternalStatus(parseCtx, "set custom system prompt", parseErr2)
	}
	parseLogger.Info("rpc.SetCustomSystemPrompt: complete", slog.Int("runes", len([]rune(parseCustomPrompt))))
	return &emptypb.Empty{}, nil
}

// GetCustomSystemPrompt returns the current user's custom system prompt.
func (parseS *chatServer) GetCustomSystemPrompt(parseCtx context.Context, _ *emptypb.Empty) (*wrapperspb.StringValue, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetCustomSystemPrompt"))
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetCustomSystemPrompt: store unavailable - returning default")
		return wrapperspb.String(defaultCustomSystemPromptTemplate), nil
	}
	parseCustomPrompt, parseErr := parseS.store.getSelectedSystemPrompt(parseUserID, defaultCustomSystemPromptTemplate)
	if parseErr != nil {
		parseLogger.Error("rpc.GetCustomSystemPrompt: db query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "get custom system prompt: %v", parseErr)
	}
	parseCustomPrompt = parseNormalizeCustomSystemPrompt(parseCustomPrompt)
	parseLogger.Info("rpc.GetCustomSystemPrompt: complete", slog.Int("runes", len([]rune(parseCustomPrompt))))
	return wrapperspb.String(parseCustomPrompt), nil
}

// baseAssistantSystemPrompt is the foundation injected on every request. Extend this
// to shape the assistant's personality, knowledge scope, and safety rails.
const baseAssistantSystemPrompt = `You are a helpful AI assistant built into a Go WebAssembly chat application.
You have broad general knowledge and can help with coding, writing, research, and everyday questions.
Always be accurate; if you are uncertain, say so rather than guessing.
Format responses with Markdown when it improves readability (code blocks, lists, headers).`

// toneInstructionByID maps the client-selected tone to an additional system directive.
var toneInstructionByID = map[string]string{
	"balanced":     "Communicate in a warm, conversational tone - friendly but informative.",
	"friendly":     "Be upbeat, approachable, and kind. Prefer plain language and a supportive tone while staying accurate.",
	"professional": "Communicate formally and precisely. Avoid casual language. Prefer structured, direct responses.",
	"concise":      "Be as brief as possible. Omit pleasantries and filler. Lead with the answer, then add detail only if essential.",
}

func parseNormalizeSelectedToneID(parseToneID string) string {
	parseToneID = strings.TrimSpace(parseToneID)
	if parseToneID == "" {
		return defaultToneID
	}
	if _, parseOk := toneInstructionByID[parseToneID]; parseOk {
		return parseToneID
	}
	return defaultToneID
}

func parseNormalizeSelectedThinkingEffort(parseEffort string) string {
	parseEffort = strings.TrimSpace(strings.ToLower(parseEffort))
	switch parseEffort {
	case "low", "medium", "high":
		return parseEffort
	default:
		return defaultThinkingEffort
	}
}

var fencedCodeBlockPattern = regexp.MustCompile("(?s)```.*?```|~~~.*?~~~")
var inlineCodePattern = regexp.MustCompile("`[^`]+`")
var markdownLinkPattern = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
var markdownFormattingPattern = regexp.MustCompile(`(?m)^#{1,6}\s+|[*_~]+|^>\s?`)
var htmlCommentPattern = regexp.MustCompile(`(?s)<!--.*?-->`)
var blankLinePattern = regexp.MustCompile(`\n{3,}`)

func parseSanitizeTextForTTS(parseSource string) string {
	parseTrimmed := strings.TrimSpace(parseSource)
	if parseTrimmed == "" {
		return ""
	}
	parseTrimmed = htmlCommentPattern.ReplaceAllString(parseTrimmed, " ")
	parseTrimmed = fencedCodeBlockPattern.ReplaceAllString(parseTrimmed, " ")
	parseTrimmed = inlineCodePattern.ReplaceAllString(parseTrimmed, " ")
	parseTrimmed = markdownLinkPattern.ReplaceAllString(parseTrimmed, "$1")
	parseTrimmed = markdownFormattingPattern.ReplaceAllString(parseTrimmed, "")

	parseLines := strings.Split(parseTrimmed, "\n")
	parseKept := make([]string, 0, len(parseLines))
	for _, parseLine := range parseLines {
		parseResolved := strings.TrimSpace(parseLine)
		if parseResolved == "" {
			parseKept = append(parseKept, "")
			continue
		}
		if strings.HasPrefix(parseResolved, "//") || strings.HasPrefix(parseResolved, "/*") || strings.HasPrefix(parseResolved, "*") || strings.HasPrefix(parseResolved, "*/") {
			continue
		}
		parseKept = append(parseKept, parseResolved)
	}

	parseTrimmed = strings.TrimSpace(strings.Join(parseKept, "\n"))
	parseTrimmed = blankLinePattern.ReplaceAllString(parseTrimmed, "\n\n")
	parseTrimmed = strings.Join(strings.Fields(parseTrimmed), " ")
	if parseTrimmed == "" {
		return ""
	}
	parseRunes := []rune(parseTrimmed)
	if len(parseRunes) > maxTTSScriptRunes {
		parseTrimmed = strings.TrimSpace(string(parseRunes[:maxTTSScriptRunes]))
	}
	return parseTrimmed
}

// buildSystemPrompt combines the base prompt with the tone directive for the
// given tone ID. Falls back to balanced when the ID is unrecognised.
func buildSystemPrompt(parseTone string, parseCustomPrompt string, parseMemories []userMemoryRow) string {
	parseInstruction := toneInstructionByID[parseNormalizeSelectedToneID(parseTone)]
	parseCustomPrompt = parseResolveCustomSystemPrompt(parseCustomPrompt)
	parseMemoryBlock := buildUserMemoryPromptBlock(parseMemories)
	parseCustomPromptUsesMemories := strings.Contains(parseCustomPrompt, "{{memories}}")
	parseCustomPrompt = parseResolveSystemPromptTemplate(parseCustomPrompt, parseMemoryBlock, time.Now())
	parsePrompt := baseAssistantSystemPrompt + "\n\n" + parseInstruction
	if parseCustomPrompt != "" {
		parsePrompt += "\n\nAdditional user-configured instructions:\n" + parseCustomPrompt
	}
	if parseMemoryBlock != "" && !parseCustomPromptUsesMemories {
		parsePrompt += "\n\nKnown user context:\n" + parseMemoryBlock
	}
	return parsePrompt
}

func parseResolveSystemPromptTemplate(parsePrompt, parseMemoryBlock string, parseNow time.Time) string {
	parseResolved := strings.TrimSpace(parsePrompt)
	if parseResolved == "" {
		return ""
	}
	parseResolvedMemories := strings.TrimSpace(parseMemoryBlock)
	if parseResolvedMemories == "" {
		parseResolvedMemories = "- No stored memories yet."
	}
	parseReplacer := strings.NewReplacer(
		"{{date}}", parseNow.Format("2006-01-02"),
		"{{time}}", parseNow.Format("15:04:05 -0700"),
		"{{memories}}", parseResolvedMemories,
	)
	return parseReplacer.Replace(parseResolved)
}

// parseResolveCustomSystemPrompt returns the normalized user prompt or the system-default template.
func parseResolveCustomSystemPrompt(parsePrompt string) string {
	parseNormalizedPrompt := parseNormalizeCustomSystemPrompt(parsePrompt)
	if parseNormalizedPrompt != "" {
		return parseNormalizedPrompt
	}
	return defaultCustomSystemPromptTemplate
}

func parseNormalizeCustomSystemPrompt(parsePrompt string) string {
	parseTrimmed := strings.TrimSpace(parsePrompt)
	if parseTrimmed == "" {
		return ""
	}
	parseRunes := []rune(parseTrimmed)
	if len(parseRunes) > maxCustomSystemPromptRunes {
		parseTrimmed = strings.TrimSpace(string(parseRunes[:maxCustomSystemPromptRunes]))
	}
	return parseTrimmed
}

func (parseS *chatServer) parseExtractAndStoreUserMemories(parseCtx context.Context, parseUserID int64, parseUserMessage string) {
	if parseS == nil || parseS.store == nil || parseS.providerRegistry == nil {
		return
	}
	parseStartedAt := time.Now()
	parseTrimmedMessage := strings.TrimSpace(parseUserMessage)
	if parseTrimmedMessage == "" {
		parseS.logger.Debug("memory extraction skipped: blank message", slog.Int64("user_id", parseUserID))
		return
	}
	if parseSlots := parseS.memoryExtractionSlots; parseSlots != nil {
		select {
		case parseSlots <- struct{}{}:
			defer func() { <-parseSlots }()
		default:
			parseS.logger.Info("memory extraction skipped: queue full",
				slog.Int64("user_id", parseUserID),
				slog.Int("message_chars", len([]rune(parseTrimmedMessage))),
			)
			return
		}
	}

	parseExtractionModel := parseNormalizeSelectedModelID(parseS.memoryExtractionModel)
	if parseExtractionModel == "" {
		parseExtractionModel = parseS.defaultModel
	}
	parseExtractionProvider, parseResolvedModel, parseErr := parseS.providerRegistry.ParseResolve(parseExtractionModel)
	if parseErr != nil {
		parseS.logger.Warn("memory extraction skipped: provider unavailable",
			slog.Int64("user_id", parseUserID),
			slog.String("requested_model", parseExtractionModel),
			slog.String("error", parseErr.Error()),
			slog.Int64("duration_ms", time.Since(parseStartedAt).Milliseconds()),
		)
		return
	}
	parseS.logger.Debug("memory extraction started",
		slog.Int64("user_id", parseUserID),
		slog.String("provider", parseExtractionProvider.ParseID()),
		slog.String("model", parseResolvedModel),
		slog.Int("message_chars", len([]rune(parseTrimmedMessage))),
	)

	if parseCtx == nil {
		parseCtx = context.Background()
	}
	parseCtx, parseCancel := context.WithTimeout(parseCtx, userMemoryExtractionTimeout)
	defer parseCancel()

	parseCandidates, parseErr := parseExtractionProvider.ParseExtractUserMemories(parseCtx, provider.MemoryExtractionRequest{
		Model:       parseResolvedModel,
		UserMessage: parseTrimmedMessage,
	})
	if parseErr != nil {
		parseS.logger.Warn("memory extraction failed",
			slog.Int64("user_id", parseUserID),
			slog.String("provider", parseExtractionProvider.ParseID()),
			slog.String("model", parseResolvedModel),
			slog.String("error", parseErr.Error()),
			slog.Int64("duration_ms", time.Since(parseStartedAt).Milliseconds()),
		)
		return
	}

	parseUsefulCandidates := filterUsefulUserMemories(parseCandidates)
	parseExistingMemoryKeyBySignature := map[string]string{}
	parseExistingMemories, parseErr := parseS.store.parseListUserMemories(parseUserID)
	if parseErr != nil {
		parseS.logger.Warn("memory extraction existing-memory lookup failed",
			slog.Int64("user_id", parseUserID),
			slog.String("provider", parseExtractionProvider.ParseID()),
			slog.String("model", parseResolvedModel),
			slog.String("error", parseErr.Error()),
		)
	} else {
		parseExistingMemoryKeyBySignature = parseBuildUserMemorySignatureKeyMap(parseExistingMemories)
	}
	parseSavedCount := 0
	parseSaveFailureCount := 0
	for _, parseCandidate := range parseUsefulCandidates {
		parseMemoryKey := parseNormalizeUserMemoryKey(parseCandidate.Key, parseCandidate.Category, parseCandidate.Summary)
		parseMemorySignature := parseBuildUserMemorySignature(parseCandidate.Category, parseCandidate.Summary, parseCandidate.Detail)
		if parseMemorySignature != "" {
			if parseExistingMemoryKey, hasParseExistingMemoryKey := parseExistingMemoryKeyBySignature[parseMemorySignature]; hasParseExistingMemoryKey && strings.TrimSpace(parseExistingMemoryKey) != "" {
				parseMemoryKey = strings.TrimSpace(parseExistingMemoryKey)
			} else {
				parseExistingMemoryKeyBySignature[parseMemorySignature] = parseMemoryKey
			}
		}
		if parseErr2 := parseS.store.parseUpsertUserMemory(parseUserID, userMemoryRow{
			Key:             parseMemoryKey,
			Category:        parseNormalizeUserMemoryCategory(parseCandidate.Category),
			Summary:         strings.TrimSpace(parseCandidate.Summary),
			Detail:          strings.TrimSpace(parseCandidate.Detail),
			SourceMessage:   parseTrimmedMessage,
			UsefulnessScore: parseClampUsefulnessScore(parseCandidate.UsefulnessScore),
			ConfidenceScore: parseClampConfidenceScore(parseCandidate.ConfidenceScore),
			RubricReason:    strings.TrimSpace(parseCandidate.RubricReason),
		}); parseErr2 != nil {
			parseSaveFailureCount++
			parseS.logger.Warn("memory extraction save failed",
				slog.Int64("user_id", parseUserID),
				slog.String("provider", parseExtractionProvider.ParseID()),
				slog.String("model", parseResolvedModel),
				slog.String("memory_key", parseCandidate.Key),
				slog.String("error", parseErr2.Error()),
			)
			continue
		}
		parseSavedCount++
	}
	parseS.logger.Info("memory extraction completed",
		slog.Int64("user_id", parseUserID),
		slog.String("provider", parseExtractionProvider.ParseID()),
		slog.String("model", parseResolvedModel),
		slog.Int("candidate_count", len(parseCandidates)),
		slog.Int("useful_candidate_count", len(parseUsefulCandidates)),
		slog.Int("saved_count", parseSavedCount),
		slog.Int("save_failure_count", parseSaveFailureCount),
		slog.Int64("duration_ms", time.Since(parseStartedAt).Milliseconds()),
	)
}

func filterUsefulUserMemories(parseCandidates []provider.UserMemoryCandidate) []provider.UserMemoryCandidate {
	parseFiltered := make([]provider.UserMemoryCandidate, 0, len(parseCandidates))
	parseSeenOrder := make([]string, 0, len(parseCandidates))
	parseDedupedCandidates := make(map[string]provider.UserMemoryCandidate, len(parseCandidates))
	for _, parseCandidate := range parseCandidates {
		parseCandidate.Category = parseNormalizeUserMemoryCategory(parseCandidate.Category)
		parseCandidate.Key = parseNormalizeUserMemoryKey(parseCandidate.Key, parseCandidate.Category, parseCandidate.Summary)
		parseCandidate.Summary = strings.TrimSpace(parseCandidate.Summary)
		parseCandidate.Detail = strings.TrimSpace(parseCandidate.Detail)
		parseCandidate.RubricReason = strings.TrimSpace(parseCandidate.RubricReason)
		parseCandidate.UsefulnessScore = parseClampUsefulnessScore(parseCandidate.UsefulnessScore)
		parseCandidate.ConfidenceScore = parseClampConfidenceScore(parseCandidate.ConfidenceScore)
		if parseCandidate.Key == "" || parseCandidate.Summary == "" {
			continue
		}
		if parseCandidate.UsefulnessScore < userMemoryUsefulnessThreshold || parseCandidate.ConfidenceScore < userMemoryConfidenceThreshold {
			continue
		}
		parseSignature := parseBuildUserMemorySignature(parseCandidate.Category, parseCandidate.Summary, parseCandidate.Detail)
		if parseSignature == "" {
			parseSignature = parseCandidate.Key
		}
		parseExistingCandidate, hasParseExistingCandidate := parseDedupedCandidates[parseSignature]
		if !hasParseExistingCandidate {
			parseDedupedCandidates[parseSignature] = parseCandidate
			parseSeenOrder = append(parseSeenOrder, parseSignature)
			continue
		}
		if parseShouldReplaceUserMemoryCandidate(parseExistingCandidate, parseCandidate) {
			parseDedupedCandidates[parseSignature] = parseCandidate
		}
	}
	for _, parseSignature := range parseSeenOrder {
		parseFiltered = append(parseFiltered, parseDedupedCandidates[parseSignature])
	}
	return parseFiltered
}

// parseShouldReplaceUserMemoryCandidate reports whether one candidate should replace an existing deduplicated candidate.
func parseShouldReplaceUserMemoryCandidate(parseCurrent provider.UserMemoryCandidate, parseCandidate provider.UserMemoryCandidate) bool {
	if parseCandidate.UsefulnessScore != parseCurrent.UsefulnessScore {
		return parseCandidate.UsefulnessScore > parseCurrent.UsefulnessScore
	}
	if parseCandidate.ConfidenceScore != parseCurrent.ConfidenceScore {
		return parseCandidate.ConfidenceScore > parseCurrent.ConfidenceScore
	}
	parseCurrentDetail := strings.TrimSpace(parseCurrent.Detail)
	parseCandidateDetail := strings.TrimSpace(parseCandidate.Detail)
	if parseCurrentDetail == "" && parseCandidateDetail != "" {
		return true
	}
	if parseCurrentDetail != "" && parseCandidateDetail == "" {
		return false
	}
	return len([]rune(parseCandidate.Summary))+len([]rune(parseCandidate.Detail)) > len([]rune(parseCurrent.Summary))+len([]rune(parseCurrent.Detail))
}

// parseBuildUserMemorySignatureKeyMap indexes existing memory keys by normalized signature for stable extraction upserts.
func parseBuildUserMemorySignatureKeyMap(parseMemories []userMemoryRow) map[string]string {
	parseMemoryKeyBySignature := make(map[string]string, len(parseMemories))
	for _, parseMemory := range parseMemories {
		parseSignature := parseBuildUserMemorySignature(parseMemory.Category, parseMemory.Summary, parseMemory.Detail)
		if parseSignature == "" {
			continue
		}
		if _, hasParseSignature := parseMemoryKeyBySignature[parseSignature]; hasParseSignature {
			continue
		}
		parseMemoryKeyBySignature[parseSignature] = strings.TrimSpace(parseMemory.Key)
	}
	return parseMemoryKeyBySignature
}

// parseBuildUserMemorySignature builds one stable dedupe signature for memory records.
func parseBuildUserMemorySignature(parseCategory, parseSummary, parseDetail string) string {
	parseCategory = parseNormalizeUserMemoryCategory(parseCategory)
	parseSummary = parseNormalizeUserMemorySignaturePart(parseSummary)
	if parseSummary == "" {
		return ""
	}
	parseDetail = parseNormalizeUserMemorySignaturePart(parseDetail)
	if parseDetail == "" || strings.EqualFold(parseDetail, parseSummary) {
		return parseCategory + "|" + parseSummary
	}
	return parseCategory + "|" + parseSummary + "|" + parseDetail
}

// parseNormalizeUserMemorySignaturePart canonicalizes one signature fragment for dedupe comparisons.
func parseNormalizeUserMemorySignaturePart(parseValue string) string {
	parseValue = strings.TrimSpace(strings.ToLower(parseValue))
	if parseValue == "" {
		return ""
	}
	return strings.Join(strings.Fields(parseValue), " ")
}

func parseNormalizeUserMemoryCategory(parseCategory string) string {
	switch strings.TrimSpace(strings.ToLower(parseCategory)) {
	case "preference", "profile", "constraint", "project":
		return strings.TrimSpace(strings.ToLower(parseCategory))
	default:
		return "other"
	}
}

func parseNormalizeUserMemoryKey(parseKey, parseCategory, parseSummary string) string {
	parseBase := strings.TrimSpace(strings.ToLower(parseKey))
	if parseBase == "" {
		parseBase = parseNormalizeUserMemoryCategory(parseCategory) + "-" + strings.TrimSpace(strings.ToLower(parseSummary))
	}
	var parseBuilder strings.Builder
	isParseLastDash := false
	for _, parseCurrentRune := range parseBase {
		switch {
		case parseCurrentRune >= 'a' && parseCurrentRune <= 'z', parseCurrentRune >= '0' && parseCurrentRune <= '9':
			parseBuilder.WriteRune(parseCurrentRune)
			isParseLastDash = false
		default:
			if !isParseLastDash {
				parseBuilder.WriteRune('-')
				isParseLastDash = true
			}
		}
	}
	return strings.Trim(parseBuilder.String(), "-")
}

func parseClampUsefulnessScore(parseScore int) int {
	switch {
	case parseScore < 0:
		return 0
	case parseScore > 100:
		return 100
	default:
		return parseScore
	}
}

func parseClampConfidenceScore(parseScore float64) float64 {
	switch {
	case parseScore < 0:
		return 0
	case parseScore > 1:
		return 1
	default:
		return parseScore
	}
}

func buildUserMemoryPromptBlock(parseMemories []userMemoryRow) string {
	if len(parseMemories) == 0 {
		return ""
	}
	var parseBuilder strings.Builder
	parseCount := 0
	parseSeenSignatures := make(map[string]struct{}, len(parseMemories))
	for _, parseMemory := range parseMemories {
		if parseCount >= maxInjectedUserMemoryCount {
			break
		}
		parseSummary := strings.TrimSpace(parseMemory.Summary)
		if parseSummary == "" {
			continue
		}
		parseSignature := parseBuildUserMemorySignature(parseMemory.Category, parseMemory.Summary, parseMemory.Detail)
		if parseSignature != "" {
			if _, hasParseSignature := parseSeenSignatures[parseSignature]; hasParseSignature {
				continue
			}
			parseSeenSignatures[parseSignature] = struct{}{}
		}
		parseBuilder.WriteString("- ")
		parseBuilder.WriteString(parseSummary)
		if parseDetail := strings.TrimSpace(parseMemory.Detail); parseDetail != "" && !strings.EqualFold(parseDetail, parseSummary) {
			parseBuilder.WriteString(" (")
			parseBuilder.WriteString(parseDetail)
			parseBuilder.WriteString(")")
		}
		parseBuilder.WriteString("\n")
		parseCount++
		if parseBuilder.Len() >= maxInjectedUserMemoryRunes {
			break
		}
	}
	parseBlock := strings.TrimSpace(parseBuilder.String())
	if parseBlock == "" {
		return ""
	}
	parseRunes := []rune(parseBlock)
	if len(parseRunes) > maxInjectedUserMemoryRunes {
		parseBlock = strings.TrimSpace(string(parseRunes[:maxInjectedUserMemoryRunes]))
	}
	return parseBlock
}
