//go:build js && wasm

package app

import (
	"context"
	"io"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/v5/i18n"
	"github.com/monstercameron/GoWebComponents/v5/logging"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// chatStreamController owns the composer and streaming message workflow.
//
// This keeps the assistant send loop, stream bookkeeping, edit-resend path, and
// fork flow out of App() so the render function only wires the resulting
// handlers into the component tree.
type chatStreamController struct {
	StartEdit          ui.Handler
	CancelEdit         ui.Handler
	HandleEditChange   ui.Handler
	SubmitEdit         ui.Handler
	HandleEditKey      ui.Handler
	HandleInput        ui.Handler
	ApplyStarterPrompt ui.Handler
	HandleKey          ui.Handler
	Send               ui.Handler
	Fork               ui.Handler
}

const chatSendRPCDeadline = 90 * time.Second

// useChatStream hides the gRPC send/stream lifecycle and related composer
// events behind one feature hook built from ordinary public hooks.
func parseUseChatStream(
	parseIntl i18n.Runtime,
	parseApp ui.Reducer[appState, appAction],
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	parseScrollMemory threadScrollMemory,
	parseOnResetDraftModel func(),
	parseOnConversationChange func(int64),
	parseOnNavigateToConversation func(string),
	handleAuthFailure func(error) bool,
) chatStreamController {
	_ = parseIntl
	parseTriggerSend := func(parsePriorMsgs2 []message, parseUserText string) {
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			chatLog.Warn("gRPC client not yet ready", nil)
			parseRequestGRPCReconnect("send requested without an active bridge")
			return
		}
		parseCurrentState := parseApp.Get()
		parseModel := parseCurrentState.SelectedModel
		parseTone := parseCurrentState.SelectedTone
		isParseThinkingEnabled := parseCurrentState.SelectedThinkingEnabled && parseModelSupportsThinking(parseModel, parseCurrentState.ModelOptions, parseCurrentState.DefaultModel)
		parseThinkingEffort := parseCurrentState.SelectedThinkingEffort

		parseHistory := make([]*chatpb.ChatMessage, 0, len(parsePriorMsgs2))
		// Rebuild the wire transcript from local messages so the server sees the same turn history the user edited.
		for _, parseMsg := range parsePriorMsgs2 {
			if parseMsg.Role == roleSwitch {
				continue
			}
			parseHistory = append(parseHistory, &chatpb.ChatMessage{
				Role:             parseMsg.Role,
				Content:          parseMsg.Content,
				ModelId:          parseMsg.ModelID,
				PromptTokens:     int64(parseMsg.PromptTokens),
				CompletionTokens: int64(parseMsg.CompletionTokens),
			})
		}

		// Optimistically show the pending assistant turn immediately so the UI stays responsive while the provider runs.
		parseApp.Dispatch(appAction{
			Type: appActionSetMessages,
			Messages: append(append([]message{}, parsePriorMsgs2...),
				message{Role: roleUser, Content: parseUserText},
				message{Role: roleAssistant, Content: "", Pending: true, ThoughtPending: true, ModelID: parseModel},
			),
		})
		parseScrollMemory.ResetToBottomMode()
		parseApp.Dispatch(appAction{Type: appActionSetStreaming, Streaming: true})
		// On submit, bring the newest assistant bubble to the top of view.
		parseScrollMemory.ParseFollowStream()

		parseConvID := parseCurrentState.ActiveConvID
		chatLog.Info("send", logging.Fields{
			"conv_id":          parseConvID,
			"history":          len(parseHistory),
			"model":            parseModel,
			"tone":             parseTone,
			"thinking_enabled": isParseThinkingEnabled,
			"thinking_effort":  parseThinkingEffort,
			"text":             parsePreviewLogText(parseUserText, 56),
		})
		// Move the network call off the render path so the composer stays interactive while tokens stream in.
		go func() {
			defer parseApp.Dispatch(appAction{Type: appActionSetStreaming, Streaming: false})
			parseSendTime := time.Now()
			var parseFirstVisibleAt time.Time
			var parseFirstAnswerTokenAt time.Time
			var parseTokenCount int
			var parseResponseModelID string
			var parsePromptTokens int
			var parseCompletionTokens int

			parseSendCtx, parseSendCancel := context.WithTimeout(context.Background(), chatSendRPCDeadline)
			defer parseSendCancel()
			parseStream, parseErr := parseClient.Send(parseSendCtx, &chatpb.SendRequest{
				History:         parseHistory,
				Message:         parseUserText,
				Model:           parseModel,
				Tone:            parseTone,
				ThinkingEnabled: isParseThinkingEnabled,
				ThinkingEffort:  parseThinkingEffort,
				ConversationId:  parseConvID,
			})
			if parseErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr) {
					return
				}
				chatLog.Error("send failed", logging.Fields{"error": parseErr, "conv_id": parseConvID})
				parseRequestGRPCReconnect("send rpc failed")
				parseApp.Dispatch(appAction{
					Type: appActionUpdateMessages,
					UpdateMessages: func(parsePreviousMessages []message) []message {
						return parseReplacePendingMessageWithErrorValue(parsePreviousMessages, parseBuildUserErrorText(userErrorScopeChat, parseErr))
					},
				})
				return
			}

			var parseNewConvID int64
			for {
				parseChunk, parseErr2 := parseStream.Recv()
				if parseErr2 == io.EOF {
					break
				}
				if parseErr2 != nil {
					if handleAuthFailure != nil && handleAuthFailure(parseErr2) {
						return
					}
					chatLog.Error("stream recv failed", logging.Fields{"error": parseErr2, "conv_id": parseConvID})
					parseRequestGRPCReconnect("stream recv failed")
					parseApp.Dispatch(appAction{
						Type: appActionUpdateMessages,
						UpdateMessages: func(parsePreviousMessages2 []message) []message {
							return parseReplacePendingMessageWithErrorValue(parsePreviousMessages2, parseBuildUserErrorText(userErrorScopeChat, parseErr2))
						},
					})
					return
				}
				if parseChunk.GetError() != "" {
					chatLog.Error("stream error", logging.Fields{"error": parseChunk.GetError(), "conv_id": parseConvID})
					parseApp.Dispatch(appAction{
						Type: appActionUpdateMessages,
						UpdateMessages: func(parsePreviousMessages3 []message) []message {
							return parseReplacePendingMessageWithErrorValue(parsePreviousMessages3, parseBuildUserErrorText(userErrorScopeChat, nil))
						},
					})
					return
				}
				if parseChunkModel := parseChunk.GetModel(); parseChunkModel != "" {
					switch {
					case parseChunkModel == thoughtChunkModelDone:
						parseApp.Dispatch(appAction{
							Type:           appActionUpdateMessages,
							UpdateMessages: parseMarkPendingAssistantThoughtCompleteValue,
						})
						continue
					case strings.HasPrefix(parseChunkModel, thoughtChunkModelPrefix):
						parseThoughtDelta := strings.TrimPrefix(parseChunkModel, thoughtChunkModelPrefix)
						if parseThoughtDelta != "" {
							if parseFirstVisibleAt.IsZero() {
								parseFirstVisibleAt = time.Now()
							}
							parseApp.Dispatch(appAction{
								Type: appActionUpdateMessages,
								UpdateMessages: func(parsePreviousMessages4 []message) []message {
									return parseAppendAssistantThoughtDeltaValue(parsePreviousMessages4, parseThoughtDelta)
								},
							})
							if parseScrollMemory.ShouldAutoScroll() {
								parseScrollMemory.ParseFollowStream()
							}
						}
						continue
					default:
						parseResponseModelID = parseChunkModel
					}
				}
				if parseChunk.GetPromptTokens() > 0 {
					parsePromptTokens = int(parseChunk.GetPromptTokens())
				}
				if parseChunk.GetCompletionTokens() > 0 {
					parseCompletionTokens = int(parseChunk.GetCompletionTokens())
				}
				if parseChunk.GetDone() {
					parseNewConvID = parseChunk.GetConversationId()
					break
				}
				if parseDelta := parseChunk.GetDelta(); parseDelta != "" {
					if parseFirstVisibleAt.IsZero() {
						parseFirstVisibleAt = time.Now()
					}
					if parseFirstAnswerTokenAt.IsZero() {
						parseFirstAnswerTokenAt = time.Now()
					}
					parseTokenCount++
					parseApp.Dispatch(appAction{
						Type: appActionUpdateMessages,
						UpdateMessages: func(parsePreviousMessages5 []message) []message {
							return parseAppendAssistantMessageDeltaValue(parsePreviousMessages5, parseDelta)
						},
					})
					if parseScrollMemory.ShouldAutoScroll() {
						parseScrollMemory.ParseFollowStream()
					}
				}
			}

			var parseTtft float64
			var parseTkps float64
			if !parseFirstVisibleAt.IsZero() {
				parseTtft = parseFirstVisibleAt.Sub(parseSendTime).Seconds()
			}
			if !parseFirstAnswerTokenAt.IsZero() {
				if parseBodyDuration := time.Since(parseFirstAnswerTokenAt).Seconds(); parseBodyDuration > 0 {
					parseTkps = float64(parseTokenCount) / parseBodyDuration
				}
			}
			if parseResponseModelID == "" {
				parseResponseModelID = parseModel
			}
			parseCurrentStateAfterReply := parseApp.Get()
			parseResponseModelID = parseNormalizeSelectedModelID(parseResponseModelID, parseCurrentStateAfterReply.ModelOptions, parseCurrentStateAfterReply.DefaultModel)
			if parseResponseModelID != "" && parseResponseModelID != parseCurrentStateAfterReply.SelectedModel {
				parsePriorSelectedModel := parseCurrentStateAfterReply.SelectedModel
				parseApp.Dispatch(appAction{Type: appActionSetSelectedModel, SelectedModel: parseResponseModelID})
				if parseModelSyncClient := parseChatClientRef.Get(); parseModelSyncClient != nil {
					go func(parseModelID, parsePreviousModelID string) {
						_, parseSyncErr := parseModelSyncClient.SetSelectedModel(context.Background(), wrapperspb.String(parseModelID))
						if parseSyncErr != nil {
							if handleAuthFailure != nil && handleAuthFailure(parseSyncErr) {
								return
							}
							chatLog.Warn("sync selected model after reply failed", logging.Fields{
								"error":          parseSyncErr,
								"model":          parseModelID,
								"previous_model": parsePreviousModelID,
							})
							return
						}
						chatLog.Info("synced selected model from reply", logging.Fields{
							"model":          parseModelID,
							"previous_model": parsePreviousModelID,
						})
					}(parseResponseModelID, parsePriorSelectedModel)
				}
			}
			chatLog.Info("reply", logging.Fields{
				"conv_id":        parseNewConvID,
				"model":          parseResponseModelID,
				"prompt_tok":     parsePromptTokens,
				"completion_tok": parseCompletionTokens,
				"stream_tok":     parseTokenCount,
				"ttft_ms":        int(parseTtft * 1000),
			})
			parseApp.Dispatch(appAction{
				Type: appActionUpdateMessages,
				UpdateMessages: func(parsePreviousMessages6 []message) []message {
					return parseFinalizePendingMessageWithStatsValue(parsePreviousMessages6, parseTtft, parseTkps, parseTokenCount, parseResponseModelID, parsePromptTokens, parseCompletionTokens)
				},
			})
			if parseNewConvID > 0 {
				parseCurrentState2 := parseApp.Get()
				parseResolvedPublicID := parseSummaryPublicIDForID(parseCurrentState2.ConversationList, parseNewConvID)
				if parseResolvedPublicID == "" {
					chatLog.Warn("reply completed before conversation route resolved", logging.Fields{
						"conv_id":           parseNewConvID,
						"conversation_list": len(parseCurrentState2.ConversationList),
					})
				}
				parseApp.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: parseNewConvID, ActiveConvPublicID: parseResolvedPublicID})
			}
			if parseOnConversationChange != nil {
				parseOnConversationChange(parseNewConvID)
			}
		}()
	}

	parsePerformSend := func() {
		parseCurrentState3 := parseApp.Get()
		parseText := strings.TrimSpace(parseCurrentState3.InputText)
		if parseText == "" || parseCurrentState3.Streaming {
			return
		}
		if parseChatClientRef.Get() == nil {
			chatLog.Warn("gRPC client not yet ready", nil)
			parseRequestGRPCReconnect("composer send requested while bridge unavailable")
			parseApp.Dispatch(appAction{Type: appActionSetInputText, InputText: ""})
			parseApp.Dispatch(appAction{
				Type: appActionUpdateMessages,
				UpdateMessages: func(parsePreviousMessages []message) []message {
					return append(
						append(append([]message{}, parsePreviousMessages...), message{Role: roleUser, Content: parseText}),
						message{Role: roleAssistant, Content: parseBuildUserErrorText(userErrorScopeChat, nil)},
					)
				},
			})
			parseScrollMemory.ResetToBottomMode()
			return
		}
		parseCurrent := parseCurrentState3.Messages
		parseApp.Dispatch(appAction{Type: appActionSetInputText, InputText: ""})
		parseTriggerSend(parseCurrent, parseText)
		parseScheduleFocusChatInput(focusDelay)
	}

	parseSubmitEditFlow := func() {
		parseCurrentState4 := parseApp.Get()
		parseIdx := parseCurrentState4.EditIdx
		if parseIdx < 0 {
			return
		}
		parseCurrent2 := parseCurrentState4.Messages
		if parseIdx >= len(parseCurrent2) {
			return
		}
		parseText2 := strings.TrimSpace(parseCurrentState4.EditText)
		if parseText2 == "" {
			return
		}
		chatLog.Info("edit resend", logging.Fields{"idx": parseIdx, "text": parsePreviewLogText(parseText2, 56)})
		parsePriorMsgs := append([]message{}, parseCurrent2[:parseIdx]...)
		parseApp.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: -1})
		parseApp.Dispatch(appAction{Type: appActionSetEditText, EditText: ""})
		parseTriggerSend(parsePriorMsgs, parseText2)
	}

	parseStartEdit := ui.UseEvent(func(parseE ui.Event) {
		if parseApp.Get().Streaming {
			return
		}
		parseIdx2, parseOk := parseEventDatasetInt(parseE, dataIdx)
		if !parseOk || parseIdx2 < 0 {
			return
		}
		parseCurrent3 := parseApp.Get().Messages
		if parseIdx2 >= len(parseCurrent3) {
			return
		}
		parseApp.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: parseIdx2})
		parseApp.Dispatch(appAction{Type: appActionSetEditText, EditText: parseCurrent3[parseIdx2].Content})
	})

	parseCancelEdit := ui.UseEvent(func() {
		parseApp.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: -1})
		parseApp.Dispatch(appAction{Type: appActionSetEditText, EditText: ""})
	})

	handleEditChange := ui.UseEvent(func(parseE2 ui.Event) {
		parseApp.Dispatch(appAction{Type: appActionSetEditText, EditText: parseE2.GetValue()})
	})

	parseSubmitEdit := ui.UseEvent(func() {
		parseSubmitEditFlow()
	})

	handleEditKey := ui.UseEvent(func(parseE3 ui.Event) {
		if parseE3.GetKey() == "Enter" && !parseE3.JSValue().Get("shiftKey").Bool() {
			parseE3.PreventDefault()
			parseSubmitEditFlow()
		}
	})

	parseSend := ui.UseEvent(func() {
		parsePerformSend()
	})

	handleKey := ui.UseEvent(func(parseE4 ui.Event) {
		if parseE4.GetKey() == "Enter" && !parseE4.JSValue().Get("shiftKey").Bool() {
			parseE4.PreventDefault()
			parsePerformSend()
		}
	})

	handleInput := ui.UseEvent(func(parseE5 ui.Event) {
		parseApp.Dispatch(appAction{Type: appActionSetInputText, InputText: parseE5.GetValue()})
	})

	parseApplyStarterPrompt := ui.UseEvent(func(parseE5 ui.Event) {
		if parseApp.Get().Streaming {
			return
		}
		parsePromptText := strings.TrimSpace(parseEventDatasetValue(parseE5, dataStarterPrompt))
		if parsePromptText == "" {
			return
		}
		parseApp.Dispatch(appAction{Type: appActionSetInputText, InputText: parsePromptText})
		parseScheduleFocusChatInput(focusDelay)
	})

	parseFork := ui.UseEvent(func(parseE6 ui.Event) {
		if parseApp.Get().Streaming {
			return
		}
		parseIdx3, parseOk2 := parseEventDatasetInt(parseE6, dataIdx)
		if !parseOk2 || parseIdx3 < 0 {
			return
		}
		parseCurrent4 := parseApp.Get().Messages
		if parseIdx3 >= len(parseCurrent4) {
			return
		}
		parseScrollMemory.CancelPendingPersist()
		parseScrollMemory.ParsePersistNow(parseApp.Get().ActiveConvID)
		parseScrollMemory.ParsePrepareRestore(0)
		chatLog.Info("fork", logging.Fields{"idx": parseIdx3, "remaining": parseIdx3 + 1})
		parseApp.Dispatch(appAction{Type: appActionSetMessages, Messages: append([]message{}, parseCurrent4[:parseIdx3+1]...)})
		parseApp.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: 0, ActiveConvPublicID: ""})
		parseApp.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: -1})
		parseApp.Dispatch(appAction{Type: appActionSetEditText, EditText: ""})
		if parseOnResetDraftModel != nil {
			parseOnResetDraftModel()
		}
		if parseOnNavigateToConversation != nil {
			parseOnNavigateToConversation("")
		}
		parseScheduleFocusChatInput(forkFocusDelay)
	})

	return chatStreamController{
		StartEdit:          parseStartEdit,
		CancelEdit:         parseCancelEdit,
		HandleEditChange:   handleEditChange,
		SubmitEdit:         parseSubmitEdit,
		HandleEditKey:      handleEditKey,
		HandleInput:        handleInput,
		ApplyStarterPrompt: parseApplyStarterPrompt,
		HandleKey:          handleKey,
		Send:               parseSend,
		Fork:               parseFork,
	}
}
