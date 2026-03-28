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

// ParseStatus returns the current TTS controller status.
func (parseC ttsAudioController) ParseStatus(parseKey, parseModel string) ttsClipStatus {
	if parseC.clipStatus == nil {
		return ttsClipStatus{}
	}
	return parseC.clipStatus(parseKey, parseModel)
}

// ParseToggle toggles speech playback.
func (parseC ttsAudioController) ParseToggle(parseKey, parseText, parseModel string) {
	if parseC.toggle != nil {
		parseC.toggle(parseKey, parseText, parseModel)
	}
}

// ParseStop stops active playback.
func (parseC ttsAudioController) ParseStop(parseKey string) {
	if parseC.stop != nil {
		parseC.stop(parseKey)
	}
}

// ParseStopCurrent stops the current playback session.
func (parseC ttsAudioController) ParseStopCurrent() {
	if parseC.stopCurrent != nil {
		parseC.stopCurrent()
	}
}

func parseUseTTSAudio(parseActiveConvID int64, parseCatalog modelCatalog, parseChatClientRef ui.Ref[chatpb.ChatServiceClient], parseSelectedTTSProvider string) ttsAudioController {
	parseIntl := i18n.UseI18n()
	parsePlaybackState := ui.UseState(ttsPlaybackState{})
	parseAudioElementRef := ui.UseRef(interop.Value{})
	parseAudioSubscriptionRef := ui.UseRef([]interop.Subscription{})
	parseClipCacheRef := ui.UseRef(map[string]ttsClip{})
	parseRequestIDRef := ui.UseRef(0)
	parseRequestCancelRef := ui.UseRef(context.CancelFunc(nil))

	setPlaybackState := func(parseNext ttsPlaybackState) {
		parsePlaybackState.Set(parseNext)
	}

	clearPendingRequest := func() {
		parseCancel := parseRequestCancelRef.Get()
		if parseCancel == nil {
			return
		}
		parseCancel()
		parseRequestCancelRef.Set(nil)
	}

	parseCreateAudioElement := func() (interop.Value, bool) {
		parseGlobal, parseErr := interop.GetGlobalThis()
		if parseErr != nil {
			return interop.Value{}, false
		}

		parseAudioFactory := parseGlobal.Get("Audio")
		if parseAudioFactory.Present() {
			if parseAudioElement, parseInvokeErr := parseAudioFactory.Invoke(); parseInvokeErr == nil && parseAudioElement.Present() {
				return parseAudioElement, true
			}
		}

		parseDocument := parseGlobal.Get("document")
		if !parseDocument.Present() {
			return interop.Value{}, false
		}
		parseAudioElement2, parseErr := parseDocument.Call("createElement", "audio")
		if parseErr != nil || !parseAudioElement2.Present() {
			return interop.Value{}, false
		}
		return parseAudioElement2, true
	}

	parseEnsureAudioElement := func() (interop.Value, bool) {
		parseAudioElement3 := parseAudioElementRef.Get()
		if parseAudioElement3.Present() {
			return parseAudioElement3, true
		}

		parseAudioElement3, parseOk := parseCreateAudioElement()
		if !parseOk {
			return interop.Value{}, false
		}
		_ = parseAudioElement3.Set("preload", "auto")
		_ = parseAudioElement3.Set("playsInline", true)

		parseSubscriptions := make([]interop.Subscription, 0, 4)
		if parseSub, parseSubErr := parseAudioElement3.SetFunction("onplay", func(parseArgs ...interop.Value) any {
			parseCurrent := parsePlaybackState.Get()
			parseCurrent.IsPlaying = true
			parseCurrent.Error = ""
			setPlaybackState(parseCurrent)
			return nil
		}); parseSubErr == nil {
			parseSubscriptions = append(parseSubscriptions, parseSub)
		}
		if parseSub2, parseSubErr2 := parseAudioElement3.SetFunction("onpause", func(parseArgs2 ...interop.Value) any {
			parseCurrent2 := parsePlaybackState.Get()
			parseCurrent2.IsPlaying = false
			setPlaybackState(parseCurrent2)
			return nil
		}); parseSubErr2 == nil {
			parseSubscriptions = append(parseSubscriptions, parseSub2)
		}
		if parseSub3, parseSubErr3 := parseAudioElement3.SetFunction("onended", func(parseArgs3 ...interop.Value) any {
			parseCurrent3 := parsePlaybackState.Get()
			parseCurrent3.IsPlaying = false
			parseCurrent3.LoadingKey = ""
			setPlaybackState(parseCurrent3)
			return nil
		}); parseSubErr3 == nil {
			parseSubscriptions = append(parseSubscriptions, parseSub3)
		}
		if parseSub4, parseSubErr4 := parseAudioElement3.SetFunction("onerror", func(parseArgs4 ...interop.Value) any {
			parseCurrent4 := parsePlaybackState.Get()
			parseCurrent4.IsPlaying = false
			parseCurrent4.LoadingKey = ""
			parseCurrent4.Error = parseIntl.T(chatI18nNamespace, "tts.audioPlaybackFailed")
			setPlaybackState(parseCurrent4)
			return nil
		}); parseSubErr4 == nil {
			parseSubscriptions = append(parseSubscriptions, parseSub4)
		}

		parseAudioSubscriptionRef.Set(parseSubscriptions)
		parseAudioElementRef.Set(parseAudioElement3)
		return parseAudioElement3, true
	}

	parseStopPlayback := func(isClearActiveKey bool) {
		parseRequestIDRef.Set(parseRequestIDRef.Get() + 1)
		clearPendingRequest()
		if parseAudioElement4, parseOk2 := parseEnsureAudioElement(); parseOk2 {
			_, _ = parseAudioElement4.Call("pause")
			_ = parseAudioElement4.Set("currentTime", 0)
		}
		parseCurrent5 := parsePlaybackState.Get()
		parseCurrent5.IsPlaying = false
		parseCurrent5.LoadingKey = ""
		if isClearActiveKey {
			parseCurrent5.ActiveKey = ""
			parseCurrent5.Error = ""
		}
		setPlaybackState(parseCurrent5)
	}

	parsePlayClip := func(parseKey string, parseClip2 ttsClip) {
		parseAudioElement5, parseOk3 := parseEnsureAudioElement()
		if !parseOk3 {
			setPlaybackState(ttsPlaybackState{ActiveKey: parseKey, Error: parseIntl.T(chatI18nNamespace, "tts.audioUnavailable")})
			return
		}
		_ = parseAudioElement5.Set("src", parseClip2.DataURL)
		_ = parseAudioElement5.Set("currentTime", 0)
		if _, parseErr2 := parseAudioElement5.Call("play"); parseErr2 != nil {
			setPlaybackState(ttsPlaybackState{ActiveKey: parseKey, Error: parseIntl.T(chatI18nNamespace, "tts.audioCouldNotStart")})
			return
		}
		setPlaybackState(ttsPlaybackState{ActiveKey: parseKey, IsPlaying: true})
	}

	parseToggle := func(parseKey2, parseText, parseModel string) {
		parseTrimmed := strings.TrimSpace(parseText)
		if parseTrimmed == "" {
			setPlaybackState(ttsPlaybackState{ActiveKey: parseKey2, Error: parseIntl.T(chatI18nNamespace, "tts.noText")})
			return
		}

		parseResolvedModel, parseSupported := parseResolveSpeechSynthesisModelForProvider(parseModel, parseCatalog.Models, parseCatalog.DefaultModel, parseSelectedTTSProvider)
		if !parseSupported {
			setPlaybackState(ttsPlaybackState{ActiveKey: parseKey2, Error: parseIntl.T(chatI18nNamespace, "assistant.speechUnavailable")})
			return
		}

		parseCurrent6 := parsePlaybackState.Get()
		if parseCurrent6.LoadingKey == parseKey2 {
			return
		}

		parseAudioElement6, parseOk4 := parseEnsureAudioElement()
		if !parseOk4 {
			setPlaybackState(ttsPlaybackState{ActiveKey: parseKey2, Error: parseIntl.T(chatI18nNamespace, "tts.audioUnavailable")})
			return
		}

		if parseCurrent6.ActiveKey == parseKey2 && parseCurrent6.IsPlaying {
			_, _ = parseAudioElement6.Call("pause")
			parseCurrent6.IsPlaying = false
			setPlaybackState(parseCurrent6)
			return
		}

		if parseCurrent6.ActiveKey == parseKey2 && !parseCurrent6.IsPlaying && parseCurrent6.LoadingKey == "" {
			if _, parseErr3 := parseAudioElement6.Call("play"); parseErr3 != nil {
				setPlaybackState(ttsPlaybackState{ActiveKey: parseKey2, Error: parseIntl.T(chatI18nNamespace, "tts.audioCouldNotResume")})
				return
			}
			parseCurrent6.IsPlaying = true
			parseCurrent6.Error = ""
			setPlaybackState(parseCurrent6)
			return
		}

		parseStopPlayback(false)
		if parseClip, parseOk5 := parseClipCacheRef.Get()[parseKey2]; parseOk5 {
			parsePlayClip(parseKey2, parseClip)
			return
		}

		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			setPlaybackState(ttsPlaybackState{ActiveKey: parseKey2, Error: parseIntl.T(chatI18nNamespace, "tts.connectionNotReady")})
			return
		}

		parseRequestID := parseRequestIDRef.Get() + 1
		parseRequestIDRef.Set(parseRequestID)
		setPlaybackState(ttsPlaybackState{ActiveKey: parseKey2, LoadingKey: parseKey2})

		go func(parseCurrentRequestID int, parseResponseKey string, parseResponseText string, parseResponseModel string) {
			parseCtx, parseCancel2 := context.WithTimeout(context.Background(), 45*time.Second)
			parseRequestCancelRef.Set(parseCancel2)
			defer func() {
				parseCancel2()
				if parseRequestIDRef.Get() == parseCurrentRequestID {
					parseRequestCancelRef.Set(nil)
				}
			}()

			parseStream, parseErr4 := parseClient.SynthesizeSpeech(parseCtx, &chatpb.SynthesizeSpeechRequest{Text: parseResponseText, Model: parseResponseModel})
			if parseCurrentRequestID != parseRequestIDRef.Get() {
				return
			}
			if parseErr4 != nil {
				setPlaybackState(ttsPlaybackState{ActiveKey: parseResponseKey, Error: parseIntl.T(chatI18nNamespace, "tts.speechSynthesisFailed")})
				return
			}

			parseResolvedMimeType := "audio/mpeg"
			parseResolvedScript := ""
			parseAudioBytes := make([]byte, 0, 64*1024)
			isParseStreamDone := false
			for {
				parseChunk, parseRecvErr := parseStream.Recv()
				if parseRecvErr == io.EOF {
					break
				}
				if parseCurrentRequestID != parseRequestIDRef.Get() {
					return
				}
				if parseRecvErr != nil {
					setPlaybackState(ttsPlaybackState{ActiveKey: parseResponseKey, Error: parseIntl.T(chatI18nNamespace, "tts.speechSynthesisFailed")})
					return
				}
				if parseChunk.GetError() != "" {
					setPlaybackState(ttsPlaybackState{ActiveKey: parseResponseKey, Error: parseChunk.GetError()})
					return
				}
				if parseMimeType := strings.TrimSpace(parseChunk.GetMimeType()); parseMimeType != "" {
					parseResolvedMimeType = parseMimeType
				}
				if parseResolvedScript == "" {
					parseResolvedScript = parseChunk.GetScript()
				}
				if parseAudioChunk := parseChunk.GetAudioChunk(); len(parseAudioChunk) > 0 {
					parseAudioBytes = append(parseAudioBytes, parseAudioChunk...)
				}
				if parseChunk.GetDone() {
					isParseStreamDone = true
					break
				}
			}
			if parseCurrentRequestID != parseRequestIDRef.Get() {
				return
			}
			if !isParseStreamDone || len(parseAudioBytes) == 0 {
				setPlaybackState(ttsPlaybackState{ActiveKey: parseResponseKey, Error: parseIntl.T(chatI18nNamespace, "tts.speechSynthesisReturnedNoAudio")})
				return
			}

			parseClipCache := parseClipCacheRef.Get()
			parseClipCache[parseResponseKey] = ttsClip{
				DataURL:  "data:" + parseResolvedMimeType + ";base64," + base64.StdEncoding.EncodeToString(parseAudioBytes),
				MimeType: parseResolvedMimeType,
				Script:   parseResolvedScript,
			}
			parseClipCacheRef.Set(parseClipCache)
			parsePlayClip(parseResponseKey, parseClipCache[parseResponseKey])
		}(parseRequestID, parseKey2, parseTrimmed, parseResolvedModel)
	}

	parseStop := func(parseKey3 string) {
		parseCurrent7 := parsePlaybackState.Get()
		if parseKey3 != "" && parseCurrent7.ActiveKey != parseKey3 && parseCurrent7.LoadingKey != parseKey3 {
			return
		}
		parseStopPlayback(true)
	}

	ui.UseEffect(func() func() {
		_, _ = parseEnsureAudioElement()
		return nil
	}, true)

	ui.UseEffect(func() func() {
		parseStopPlayback(true)
		return nil
	}, parseActiveConvID)

	ui.UseEffect(func() func() {
		return func() {
			parseStopPlayback(true)
			for _, parseSub5 := range parseAudioSubscriptionRef.Get() {
				parseSub5.Cancel()
			}
			parseAudioSubscriptionRef.Set(nil)
		}
	}, true)

	return ttsAudioController{
		clipStatus: func(parseKey4, parseModel2 string) ttsClipStatus {
			parseCurrent8 := parsePlaybackState.Get()
			_, parseSupported2 := parseResolveSpeechSynthesisModelForProvider(parseModel2, parseCatalog.Models, parseCatalog.DefaultModel, parseSelectedTTSProvider)
			parseStatus := ttsClipStatus{
				Supported: parseSupported2,
				IsLoading: parseCurrent8.LoadingKey == parseKey4,
				IsPlaying: parseCurrent8.ActiveKey == parseKey4 && parseCurrent8.IsPlaying,
				CanStop:   parseCurrent8.ActiveKey == parseKey4 || parseCurrent8.LoadingKey == parseKey4,
			}
			if parseCurrent8.ActiveKey == parseKey4 || parseCurrent8.LoadingKey == parseKey4 {
				parseStatus.Error = parseCurrent8.Error
			}
			return parseStatus
		},
		toggle:      parseToggle,
		stop:        parseStop,
		stopCurrent: func() { parseStopPlayback(true) },
	}
}
