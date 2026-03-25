//go:build js && wasm

package app

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/ui"
)

// chatStreamController owns the composer and streaming message workflow.
//
// This keeps the assistant send loop, stream bookkeeping, edit-resend path, and
// fork flow out of App() so the render function only wires the resulting
// handlers into the component tree.
type chatStreamController struct {
	StartEdit        ui.Handler
	CancelEdit       ui.Handler
	HandleEditChange ui.Handler
	SubmitEdit       ui.Handler
	HandleEditKey    ui.Handler
	HandleInput      ui.Handler
	HandleKey        ui.Handler
	Send             ui.Handler
	Fork             ui.Handler
}

// useChatStream hides the gRPC send/stream lifecycle and related composer
// events behind one feature hook built from ordinary public hooks.
func useChatStream(
	intl i18n.Runtime,
	app ui.Reducer[appState, appAction],
	chatClientRef ui.Ref[chatpb.ChatServiceClient],
	scrollMemory threadScrollMemory,
	onResetDraftModel func(),
	onConversationChange func(int64),
	onNavigateToConversation func(string),
	handleAuthFailure func(error) bool,
) chatStreamController {
	triggerSend := func(priorMsgs []message, userText string) {
		client := chatClientRef.Get()
		if client == nil {
			chatLog.Warn("gRPC client not yet ready", nil)
			requestGRPCReconnect("send requested without an active bridge")
			return
		}
		currentState := app.Get()
		model := currentState.SelectedModel
		tone := currentState.SelectedTone
		thinkingEnabled := currentState.SelectedThinkingEnabled && modelSupportsThinking(model, currentState.ModelOptions, currentState.DefaultModel)
		thinkingEffort := currentState.SelectedThinkingEffort

		history := make([]*chatpb.ChatMessage, 0, len(priorMsgs))
		for _, msg := range priorMsgs {
			if msg.Role == roleSwitch {
				continue
			}
			history = append(history, &chatpb.ChatMessage{
				Role:             msg.Role,
				Content:          msg.Content,
				ModelId:          msg.ModelID,
				PromptTokens:     int64(msg.PromptTokens),
				CompletionTokens: int64(msg.CompletionTokens),
			})
		}

		app.Dispatch(appAction{
			Type: appActionSetMessages,
			Messages: append(append([]message{}, priorMsgs...),
				message{Role: roleUser, Content: userText},
				message{Role: roleAssistant, Content: "", Pending: true, ThoughtPending: true, ModelID: model},
			),
		})
		scrollMemory.ResetToBottomMode()
		app.Dispatch(appAction{Type: appActionSetStreaming, Streaming: true})
		// On submit, bring the newest assistant bubble to the top of view.
		scrollMemory.FollowStream()

		convID := currentState.ActiveConvID
		chatLog.Info("send", logging.Fields{
			"conv_id":          convID,
			"history":          len(history),
			"model":            model,
			"tone":             tone,
			"thinking_enabled": thinkingEnabled,
			"thinking_effort":  thinkingEffort,
			"text":             previewLogText(userText, 56),
		})
		go func() {
			defer app.Dispatch(appAction{Type: appActionSetStreaming, Streaming: false})
			sendTime := time.Now()
			var firstVisibleAt time.Time
			var firstAnswerTokenAt time.Time
			var tokenCount int
			var responseModelID string
			var promptTokens int
			var completionTokens int

			stream, err := client.Send(context.Background(), &chatpb.SendRequest{
				History:         history,
				Message:         userText,
				Model:           model,
				Tone:            tone,
				ThinkingEnabled: thinkingEnabled,
				ThinkingEffort:  thinkingEffort,
				ConversationId:  convID,
			})
			if err != nil {
				if handleAuthFailure != nil && handleAuthFailure(err) {
					return
				}
				chatLog.Error("send failed", logging.Fields{"error": err, "conv_id": convID})
				requestGRPCReconnect("send rpc failed")
				app.Dispatch(appAction{
					Type: appActionUpdateMessages,
					UpdateMessages: func(previousMessages []message) []message {
						return replacePendingMessageWithErrorValue(previousMessages, fmt.Sprintf("%s %v", intl.T(chatI18nNamespace, "error.genericPrefix"), err))
					},
				})
				return
			}

			var newConvID int64
			for {
				chunk, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					if handleAuthFailure != nil && handleAuthFailure(err) {
						return
					}
					chatLog.Error("stream recv failed", logging.Fields{"error": err, "conv_id": convID})
					requestGRPCReconnect("stream recv failed")
					app.Dispatch(appAction{
						Type: appActionUpdateMessages,
						UpdateMessages: func(previousMessages []message) []message {
							return replacePendingMessageWithErrorValue(previousMessages, fmt.Sprintf("%s %v", intl.T(chatI18nNamespace, "error.receivePrefix"), err))
						},
					})
					return
				}
				if chunk.GetError() != "" {
					chatLog.Error("stream error", logging.Fields{"error": chunk.GetError(), "conv_id": convID})
					app.Dispatch(appAction{
						Type: appActionUpdateMessages,
						UpdateMessages: func(previousMessages []message) []message {
							return replacePendingMessageWithErrorValue(previousMessages, intl.T(chatI18nNamespace, "error.genericPrefix")+" "+chunk.GetError())
						},
					})
					return
				}
				if chunkModel := chunk.GetModel(); chunkModel != "" {
					switch {
					case chunkModel == thoughtChunkModelDone:
						app.Dispatch(appAction{
							Type:           appActionUpdateMessages,
							UpdateMessages: markPendingAssistantThoughtCompleteValue,
						})
						continue
					case strings.HasPrefix(chunkModel, thoughtChunkModelPrefix):
						thoughtDelta := strings.TrimPrefix(chunkModel, thoughtChunkModelPrefix)
						if thoughtDelta != "" {
							if firstVisibleAt.IsZero() {
								firstVisibleAt = time.Now()
							}
							app.Dispatch(appAction{
								Type: appActionUpdateMessages,
								UpdateMessages: func(previousMessages []message) []message {
									return appendAssistantThoughtDeltaValue(previousMessages, thoughtDelta)
								},
							})
							if scrollMemory.ShouldAutoScroll() {
								scrollMemory.FollowStream()
							}
						}
						continue
					default:
						responseModelID = chunkModel
					}
				}
				if chunk.GetPromptTokens() > 0 {
					promptTokens = int(chunk.GetPromptTokens())
				}
				if chunk.GetCompletionTokens() > 0 {
					completionTokens = int(chunk.GetCompletionTokens())
				}
				if chunk.GetDone() {
					newConvID = chunk.GetConversationId()
					break
				}
				if delta := chunk.GetDelta(); delta != "" {
					if firstVisibleAt.IsZero() {
						firstVisibleAt = time.Now()
					}
					if firstAnswerTokenAt.IsZero() {
						firstAnswerTokenAt = time.Now()
					}
					tokenCount++
					app.Dispatch(appAction{
						Type: appActionUpdateMessages,
						UpdateMessages: func(previousMessages []message) []message {
							return appendAssistantMessageDeltaValue(previousMessages, delta)
						},
					})
					if scrollMemory.ShouldAutoScroll() {
						scrollMemory.FollowStream()
					}
				}
			}

			var ttft float64
			var tkps float64
			if !firstVisibleAt.IsZero() {
				ttft = firstVisibleAt.Sub(sendTime).Seconds()
			}
			if !firstAnswerTokenAt.IsZero() {
				if bodyDuration := time.Since(firstAnswerTokenAt).Seconds(); bodyDuration > 0 {
					tkps = float64(tokenCount) / bodyDuration
				}
			}
			if responseModelID == "" {
				responseModelID = model
			}
			chatLog.Info("reply", logging.Fields{
				"conv_id":        newConvID,
				"model":          responseModelID,
				"prompt_tok":     promptTokens,
				"completion_tok": completionTokens,
				"stream_tok":     tokenCount,
				"ttft_ms":        int(ttft * 1000),
			})
			app.Dispatch(appAction{
				Type: appActionUpdateMessages,
				UpdateMessages: func(previousMessages []message) []message {
					return finalizePendingMessageWithStatsValue(previousMessages, ttft, tkps, tokenCount, responseModelID, promptTokens, completionTokens)
				},
			})
			if newConvID > 0 {
				currentState := app.Get()
				resolvedPublicID := summaryPublicIDForID(currentState.ConversationList, newConvID)
				if resolvedPublicID == "" {
					chatLog.Warn("reply completed before conversation route resolved", logging.Fields{
						"conv_id":           newConvID,
						"conversation_list": len(currentState.ConversationList),
					})
				}
				app.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: newConvID, ActiveConvPublicID: resolvedPublicID})
			}
			if onConversationChange != nil {
				onConversationChange(newConvID)
			}
		}()
	}

	performSend := func() {
		currentState := app.Get()
		text := strings.TrimSpace(currentState.InputText)
		if text == "" || currentState.Streaming {
			return
		}
		if chatClientRef.Get() == nil {
			chatLog.Warn("gRPC client not yet ready", nil)
			requestGRPCReconnect("composer send requested while bridge unavailable")
			return
		}
		current := currentState.Messages
		app.Dispatch(appAction{Type: appActionSetInputText, InputText: ""})
		triggerSend(current, text)
		scheduleFocusChatInput(focusDelay)
	}

	submitEditFlow := func() {
		currentState := app.Get()
		idx := currentState.EditIdx
		if idx < 0 {
			return
		}
		current := currentState.Messages
		if idx >= len(current) {
			return
		}
		text := strings.TrimSpace(currentState.EditText)
		if text == "" {
			return
		}
		chatLog.Info("edit resend", logging.Fields{"idx": idx, "text": previewLogText(text, 56)})
		priorMsgs := append([]message{}, current[:idx]...)
		app.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: -1})
		app.Dispatch(appAction{Type: appActionSetEditText, EditText: ""})
		triggerSend(priorMsgs, text)
	}

	startEdit := ui.UseEvent(func(e ui.Event) {
		if app.Get().Streaming {
			return
		}
		idx, ok := eventDatasetInt(e, dataIdx)
		if !ok || idx < 0 {
			return
		}
		current := app.Get().Messages
		if idx >= len(current) {
			return
		}
		app.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: idx})
		app.Dispatch(appAction{Type: appActionSetEditText, EditText: current[idx].Content})
	})

	cancelEdit := ui.UseEvent(func() {
		app.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: -1})
		app.Dispatch(appAction{Type: appActionSetEditText, EditText: ""})
	})

	handleEditChange := ui.UseEvent(func(e ui.Event) {
		app.Dispatch(appAction{Type: appActionSetEditText, EditText: e.GetValue()})
	})

	submitEdit := ui.UseEvent(func() {
		submitEditFlow()
	})

	handleEditKey := ui.UseEvent(func(e ui.Event) {
		if e.GetKey() == "Enter" && !e.JSValue().Get("shiftKey").Bool() {
			e.PreventDefault()
			submitEditFlow()
		}
	})

	send := ui.UseEvent(func() {
		performSend()
	})

	handleKey := ui.UseEvent(func(e ui.Event) {
		if e.GetKey() == "Enter" && !e.JSValue().Get("shiftKey").Bool() {
			e.PreventDefault()
			performSend()
		}
	})

	handleInput := ui.UseEvent(func(e ui.Event) {
		app.Dispatch(appAction{Type: appActionSetInputText, InputText: e.GetValue()})
	})

	fork := ui.UseEvent(func(e ui.Event) {
		if app.Get().Streaming {
			return
		}
		idx, ok := eventDatasetInt(e, dataIdx)
		if !ok || idx < 0 {
			return
		}
		current := app.Get().Messages
		if idx >= len(current) {
			return
		}
		scrollMemory.CancelPendingPersist()
		scrollMemory.PersistNow(app.Get().ActiveConvID)
		scrollMemory.PrepareRestore(0)
		chatLog.Info("fork", logging.Fields{"idx": idx, "remaining": idx + 1})
		app.Dispatch(appAction{Type: appActionSetMessages, Messages: append([]message{}, current[:idx+1]...)})
		app.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: 0, ActiveConvPublicID: ""})
		app.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: -1})
		app.Dispatch(appAction{Type: appActionSetEditText, EditText: ""})
		if onResetDraftModel != nil {
			onResetDraftModel()
		}
		if onNavigateToConversation != nil {
			onNavigateToConversation("")
		}
		scheduleFocusChatInput(forkFocusDelay)
	})

	return chatStreamController{
		StartEdit:        startEdit,
		CancelEdit:       cancelEdit,
		HandleEditChange: handleEditChange,
		SubmitEdit:       submitEdit,
		HandleEditKey:    handleEditKey,
		HandleInput:      handleInput,
		HandleKey:        handleKey,
		Send:             send,
		Fork:             fork,
	}
}
