//go:build js && wasm

package app

import (
	"context"
	"encoding/base64"
	"io"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
)

type ttsClip struct {
	DataURL  string
	MimeType string
	Script   string
}

type ttsPlaybackState struct {
	ActiveKey  string
	LoadingKey string
	IsPlaying  bool
	Error      string
}

type ttsClipStatus struct {
	Supported bool
	IsLoading bool
	IsPlaying bool
	CanStop   bool
	Error     string
}

type ttsAudioController struct {
	clipStatus  func(string, string) ttsClipStatus
	toggle      func(string, string, string)
	stop        func(string)
	stopCurrent func()
}

func (c ttsAudioController) Status(key, model string) ttsClipStatus {
	if c.clipStatus == nil {
		return ttsClipStatus{}
	}
	return c.clipStatus(key, model)
}

func (c ttsAudioController) Toggle(key, text, model string) {
	if c.toggle != nil {
		c.toggle(key, text, model)
	}
}

func (c ttsAudioController) Stop(key string) {
	if c.stop != nil {
		c.stop(key)
	}
}

func (c ttsAudioController) StopCurrent() {
	if c.stopCurrent != nil {
		c.stopCurrent()
	}
}

func useTTSAudio(activeConvID int64, catalog modelCatalog, chatClientRef ui.Ref[chatpb.ChatServiceClient]) ttsAudioController {
	intl := i18n.UseI18n()
	playbackState := ui.UseState(ttsPlaybackState{})
	audioElementRef := ui.UseRef(interop.Value{})
	audioSubscriptionRef := ui.UseRef([]interop.Subscription{})
	clipCacheRef := ui.UseRef(map[string]ttsClip{})
	requestIDRef := ui.UseRef(0)
	requestCancelRef := ui.UseRef(context.CancelFunc(nil))

	setPlaybackState := func(next ttsPlaybackState) {
		playbackState.Set(next)
	}

	clearPendingRequest := func() {
		cancel := requestCancelRef.Get()
		if cancel == nil {
			return
		}
		cancel()
		requestCancelRef.Set(nil)
	}

	createAudioElement := func() (interop.Value, bool) {
		global, err := interop.GlobalThis()
		if err != nil {
			return interop.Value{}, false
		}

		audioFactory := global.Get("Audio")
		if audioFactory.Present() {
			if audioElement, invokeErr := audioFactory.Invoke(); invokeErr == nil && audioElement.Present() {
				return audioElement, true
			}
		}

		document := global.Get("document")
		if !document.Present() {
			return interop.Value{}, false
		}
		audioElement, err := document.Call("createElement", "audio")
		if err != nil || !audioElement.Present() {
			return interop.Value{}, false
		}
		return audioElement, true
	}

	ensureAudioElement := func() (interop.Value, bool) {
		audioElement := audioElementRef.Get()
		if audioElement.Present() {
			return audioElement, true
		}

		audioElement, ok := createAudioElement()
		if !ok {
			return interop.Value{}, false
		}
		_ = audioElement.Set("preload", "auto")
		_ = audioElement.Set("playsInline", true)

		subscriptions := make([]interop.Subscription, 0, 4)
		if sub, subErr := audioElement.SetFunction("onplay", func(args ...interop.Value) any {
			current := playbackState.Get()
			current.IsPlaying = true
			current.Error = ""
			setPlaybackState(current)
			return nil
		}); subErr == nil {
			subscriptions = append(subscriptions, sub)
		}
		if sub, subErr := audioElement.SetFunction("onpause", func(args ...interop.Value) any {
			current := playbackState.Get()
			current.IsPlaying = false
			setPlaybackState(current)
			return nil
		}); subErr == nil {
			subscriptions = append(subscriptions, sub)
		}
		if sub, subErr := audioElement.SetFunction("onended", func(args ...interop.Value) any {
			current := playbackState.Get()
			current.IsPlaying = false
			current.LoadingKey = ""
			setPlaybackState(current)
			return nil
		}); subErr == nil {
			subscriptions = append(subscriptions, sub)
		}
		if sub, subErr := audioElement.SetFunction("onerror", func(args ...interop.Value) any {
			current := playbackState.Get()
			current.IsPlaying = false
			current.LoadingKey = ""
			current.Error = intl.T(chatI18nNamespace, "tts.audioPlaybackFailed")
			setPlaybackState(current)
			return nil
		}); subErr == nil {
			subscriptions = append(subscriptions, sub)
		}

		audioSubscriptionRef.Set(subscriptions)
		audioElementRef.Set(audioElement)
		return audioElement, true
	}

	stopPlayback := func(clearActiveKey bool) {
		requestIDRef.Set(requestIDRef.Get() + 1)
		clearPendingRequest()
		if audioElement, ok := ensureAudioElement(); ok {
			_, _ = audioElement.Call("pause")
			_ = audioElement.Set("currentTime", 0)
		}
		current := playbackState.Get()
		current.IsPlaying = false
		current.LoadingKey = ""
		if clearActiveKey {
			current.ActiveKey = ""
			current.Error = ""
		}
		setPlaybackState(current)
	}

	playClip := func(key string, clip ttsClip) {
		audioElement, ok := ensureAudioElement()
		if !ok {
			setPlaybackState(ttsPlaybackState{ActiveKey: key, Error: intl.T(chatI18nNamespace, "tts.audioUnavailable")})
			return
		}
		_ = audioElement.Set("src", clip.DataURL)
		_ = audioElement.Set("currentTime", 0)
		if _, err := audioElement.Call("play"); err != nil {
			setPlaybackState(ttsPlaybackState{ActiveKey: key, Error: intl.T(chatI18nNamespace, "tts.audioCouldNotStart")})
			return
		}
		setPlaybackState(ttsPlaybackState{ActiveKey: key, IsPlaying: true})
	}

	toggle := func(key, text, model string) {
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			setPlaybackState(ttsPlaybackState{ActiveKey: key, Error: intl.T(chatI18nNamespace, "tts.noText")})
			return
		}

		resolvedModel := normalizeSelectedModelID(model, catalog.Models, catalog.DefaultModel)
		if !modelSupportsSpeech(resolvedModel, catalog.Models, catalog.DefaultModel) {
			setPlaybackState(ttsPlaybackState{ActiveKey: key, Error: intl.T(chatI18nNamespace, "assistant.speechUnavailable")})
			return
		}

		current := playbackState.Get()
		if current.LoadingKey == key {
			return
		}

		audioElement, ok := ensureAudioElement()
		if !ok {
			setPlaybackState(ttsPlaybackState{ActiveKey: key, Error: intl.T(chatI18nNamespace, "tts.audioUnavailable")})
			return
		}

		if current.ActiveKey == key && current.IsPlaying {
			_, _ = audioElement.Call("pause")
			current.IsPlaying = false
			setPlaybackState(current)
			return
		}

		if current.ActiveKey == key && !current.IsPlaying && current.LoadingKey == "" {
			if _, err := audioElement.Call("play"); err != nil {
				setPlaybackState(ttsPlaybackState{ActiveKey: key, Error: intl.T(chatI18nNamespace, "tts.audioCouldNotResume")})
				return
			}
			current.IsPlaying = true
			current.Error = ""
			setPlaybackState(current)
			return
		}

		stopPlayback(false)
		if clip, ok := clipCacheRef.Get()[key]; ok {
			playClip(key, clip)
			return
		}

		client := chatClientRef.Get()
		if client == nil {
			setPlaybackState(ttsPlaybackState{ActiveKey: key, Error: intl.T(chatI18nNamespace, "tts.connectionNotReady")})
			return
		}

		requestID := requestIDRef.Get() + 1
		requestIDRef.Set(requestID)
		setPlaybackState(ttsPlaybackState{ActiveKey: key, LoadingKey: key})

		go func(currentRequestID int, responseKey string, responseText string, responseModel string) {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			requestCancelRef.Set(cancel)
			defer func() {
				cancel()
				if requestIDRef.Get() == currentRequestID {
					requestCancelRef.Set(nil)
				}
			}()

			stream, err := client.SynthesizeSpeech(ctx, &chatpb.SynthesizeSpeechRequest{Text: responseText, Model: responseModel})
			if currentRequestID != requestIDRef.Get() {
				return
			}
			if err != nil {
				setPlaybackState(ttsPlaybackState{ActiveKey: responseKey, Error: intl.T(chatI18nNamespace, "tts.speechSynthesisFailed")})
				return
			}

			resolvedMimeType := "audio/mpeg"
			resolvedScript := ""
			audioBytes := make([]byte, 0, 64*1024)
			streamDone := false
			for {
				chunk, recvErr := stream.Recv()
				if recvErr == io.EOF {
					break
				}
				if currentRequestID != requestIDRef.Get() {
					return
				}
				if recvErr != nil {
					setPlaybackState(ttsPlaybackState{ActiveKey: responseKey, Error: intl.T(chatI18nNamespace, "tts.speechSynthesisFailed")})
					return
				}
				if chunk.GetError() != "" {
					setPlaybackState(ttsPlaybackState{ActiveKey: responseKey, Error: chunk.GetError()})
					return
				}
				if mimeType := strings.TrimSpace(chunk.GetMimeType()); mimeType != "" {
					resolvedMimeType = mimeType
				}
				if resolvedScript == "" {
					resolvedScript = chunk.GetScript()
				}
				if audioChunk := chunk.GetAudioChunk(); len(audioChunk) > 0 {
					audioBytes = append(audioBytes, audioChunk...)
				}
				if chunk.GetDone() {
					streamDone = true
					break
				}
			}
			if currentRequestID != requestIDRef.Get() {
				return
			}
			if !streamDone || len(audioBytes) == 0 {
				setPlaybackState(ttsPlaybackState{ActiveKey: responseKey, Error: intl.T(chatI18nNamespace, "tts.speechSynthesisReturnedNoAudio")})
				return
			}

			clipCache := clipCacheRef.Get()
			clipCache[responseKey] = ttsClip{
				DataURL:  "data:" + resolvedMimeType + ";base64," + base64.StdEncoding.EncodeToString(audioBytes),
				MimeType: resolvedMimeType,
				Script:   resolvedScript,
			}
			clipCacheRef.Set(clipCache)
			playClip(responseKey, clipCache[responseKey])
		}(requestID, key, trimmed, resolvedModel)
	}

	stop := func(key string) {
		current := playbackState.Get()
		if key != "" && current.ActiveKey != key && current.LoadingKey != key {
			return
		}
		stopPlayback(true)
	}

	ui.UseEffect(func() func() {
		_, _ = ensureAudioElement()
		return nil
	}, true)

	ui.UseEffect(func() func() {
		stopPlayback(true)
		return nil
	}, activeConvID)

	ui.UseEffect(func() func() {
		return func() {
			stopPlayback(true)
			for _, sub := range audioSubscriptionRef.Get() {
				sub.Cancel()
			}
			audioSubscriptionRef.Set(nil)
		}
	}, true)

	return ttsAudioController{
		clipStatus: func(key, model string) ttsClipStatus {
			current := playbackState.Get()
			status := ttsClipStatus{
				Supported: modelSupportsSpeech(model, catalog.Models, catalog.DefaultModel),
				IsLoading: current.LoadingKey == key,
				IsPlaying: current.ActiveKey == key && current.IsPlaying,
				CanStop:   current.ActiveKey == key || current.LoadingKey == key,
			}
			if current.ActiveKey == key || current.LoadingKey == key {
				status.Error = current.Error
			}
			return status
		},
		toggle:      toggle,
		stop:        stop,
		stopCurrent: func() { stopPlayback(true) },
	}
}
