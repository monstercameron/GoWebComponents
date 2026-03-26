//go:build js && wasm

package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"syscall/js"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

const authRefreshLeadTime = 15 * time.Minute
const authRefreshMinInterval = 1 * time.Minute
const authExpiredMessage = "Your session expired. Please sign in again."

type authSessionController struct {
	HandleModeToggle       ui.Handler
	HandleForgotPassword   ui.Handler
	HandleUpdatePassword   ui.Handler
	HandleEmailInput       ui.Handler
	HandlePasswordInput    ui.Handler
	HandleDisplayNameInput ui.Handler
	HandleSubmit           ui.Handler
	HandlePasswordKey      ui.Handler
	Logout                 ui.Handler
}

func handleUnauthenticatedRPC(parseApp ui.Reducer[appState, appAction], parseUserNameState state.Atom[string], parseErr error) bool {
	if status.Code(parseErr) != codes.Unauthenticated {
		return false
	}
	parseSessionEmail := parseApp.Get().SessionEmail
	clearPersistedAuthToken()
	parseUserNameState.Set("User")
	parseApp.Dispatch(appAction{Type: appActionResetWorkspace})
	parseApp.Dispatch(appAction{Type: appActionSetAuthenticated, Authenticated: false})
	parseApp.Dispatch(appAction{Type: appActionSetAuthResolved, AuthResolved: true})
	parseApp.Dispatch(appAction{Type: appActionSetAuthMode, AuthMode: authModeLogin})
	parseApp.Dispatch(appAction{Type: appActionSetAuthSubmitting, AuthSubmitting: false})
	parseApp.Dispatch(appAction{Type: appActionSetAuthError, AuthError: authExpiredMessage})
	parseApp.Dispatch(appAction{Type: appActionSetAuthEmail, AuthEmail: parseSessionEmail})
	parseApp.Dispatch(appAction{Type: appActionSetAuthPassword, AuthPassword: ""})
	parseApp.Dispatch(appAction{Type: appActionSetAuthDisplayName, AuthDisplayName: ""})
	parseApp.Dispatch(appAction{Type: appActionSetSessionEmail, SessionEmail: ""})
	return true
}

func parseLoadPersistedAuthToken() string {
	parseStorage, parseErr := interop.GetLocalStorage()
	if parseErr != nil {
		return ""
	}
	parseValue, parseOk, parseErr := parseStorage.GetItem(storageKeyAuthToken)
	if parseErr != nil || !parseOk {
		return ""
	}
	return strings.TrimSpace(parseValue)
}

func parsePersistAuthToken(parseToken string) {
	parseStorage, parseErr := interop.GetLocalStorage()
	if parseErr != nil {
		return
	}
	parseToken = strings.TrimSpace(parseToken)
	if parseToken == "" {
		_ = parseStorage.RemoveItem(storageKeyAuthToken)
		return
	}
	_ = parseStorage.SetItem(storageKeyAuthToken, parseToken)
}

func clearPersistedAuthToken() {
	parseStorage, parseErr := interop.GetLocalStorage()
	if parseErr != nil {
		return
	}
	_ = parseStorage.RemoveItem(storageKeyAuthToken)
}

func parseUseAuthSession(
	parseApp ui.Reducer[appState, appAction],
	parseUserNameState state.Atom[string],
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	parseOnAuthenticated func(),
	parseOnLogout func(),
) authSessionController {
	parseLastRefreshAt := ui.UseRef(time.Time{})
	parseRefreshInFlight := ui.UseRef(false)

	parseRefreshSession := func(isForce bool, parseReason string) {
		if !parseApp.Get().Authenticated || !parseApp.Get().GRPCReady {
			return
		}
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			return
		}
		parseToken := parseLoadPersistedAuthToken()
		if parseToken == "" {
			return
		}
		parseNow := time.Now()
		if !isForce && !parseAuthTokenExpiresWithin(parseToken, authRefreshLeadTime, parseNow) {
			return
		}
		if !isForce && !parseLastRefreshAt.Get().IsZero() && parseNow.Sub(parseLastRefreshAt.Get()) < authRefreshMinInterval {
			return
		}
		if parseRefreshInFlight.Get() {
			return
		}
		parseRefreshInFlight.Set(true)
		go func() {
			defer parseRefreshInFlight.Set(false)
			parseResp, parseErr := parseClient.RefreshSession(context.Background(), &emptypb.Empty{})
			if parseErr != nil {
				if handleUnauthenticatedRPC(parseApp, parseUserNameState, parseErr) {
					chatLog.Warn("auth refresh expired session", logging.Fields{"reason": parseReason})
					if parseOnLogout != nil {
						parseOnLogout()
					}
					return
				}
				chatLog.Warn("auth refresh failed", logging.Fields{"error": parseErr, "reason": parseReason})
				return
			}
			if strings.TrimSpace(parseResp.GetAuthToken()) == "" {
				return
			}
			parsePersistAuthToken(parseResp.GetAuthToken())
			parseLastRefreshAt.Set(time.Now())
			if parseDisplayName := strings.TrimSpace(parseResp.GetDisplayName()); parseDisplayName != "" {
				parseUserNameState.Set(parseDisplayName)
			}
			parseApp.Dispatch(appAction{Type: appActionSetSessionEmail, SessionEmail: parseResp.GetEmail()})
			chatLog.Info("auth refresh", logging.Fields{"reason": parseReason, "email": parseResp.GetEmail()})
		}()
	}

	ui.UseEffect(func() func() {
		if !parseApp.Get().GRPCReady {
			return nil
		}
		parseToken2 := parseLoadPersistedAuthToken()
		if parseToken2 == "" {
			parseApp.Dispatch(appAction{Type: appActionSetAuthenticated, Authenticated: false})
			parseApp.Dispatch(appAction{Type: appActionSetAuthResolved, AuthResolved: true})
			parseApp.Dispatch(appAction{Type: appActionSetSessionEmail, SessionEmail: ""})
			return nil
		}
		parseClient2 := parseChatClientRef.Get()
		if parseClient2 == nil {
			return nil
		}
		go func() {
			parseSession, parseErr2 := parseClient2.GetSession(context.Background(), &emptypb.Empty{})
			if parseErr2 != nil || !parseSession.GetAuthenticated() {
				clearPersistedAuthToken()
				parseApp.Dispatch(appAction{Type: appActionSetAuthenticated, Authenticated: false})
				parseApp.Dispatch(appAction{Type: appActionSetAuthResolved, AuthResolved: true})
				parseApp.Dispatch(appAction{Type: appActionSetAuthError, AuthError: ""})
				parseApp.Dispatch(appAction{Type: appActionSetSessionEmail, SessionEmail: ""})
				parseUserNameState.Set("User")
				if parseOnLogout != nil {
					parseOnLogout()
				}
				return
			}
			parseUserNameState.Set(parseSession.GetDisplayName())
			parseApp.Dispatch(appAction{Type: appActionSetAuthenticated, Authenticated: true})
			parseApp.Dispatch(appAction{Type: appActionSetAuthResolved, AuthResolved: true})
			parseApp.Dispatch(appAction{Type: appActionSetAuthMode, AuthMode: authModeLogin})
			parseApp.Dispatch(appAction{Type: appActionSetAuthError, AuthError: ""})
			parseApp.Dispatch(appAction{Type: appActionSetSessionEmail, SessionEmail: parseSession.GetEmail()})
			parseLastRefreshAt.Set(time.Now())
			if parseOnAuthenticated != nil {
				parseOnAuthenticated()
			}
		}()
		return nil
	}, parseApp.Get().GRPCReady)

	ui.UseEffect(func() func() {
		if !parseApp.Get().Authenticated || !parseApp.Get().GRPCReady {
			return nil
		}
		parseWindow := js.Global().Get("window")
		parseDocument := js.Global().Get("document")
		parseCallback := func(parseReason2 string) func() {
			return func() {
				parseRefreshSession(false, parseReason2)
			}
		}
		parseCleanupFns := []func(){
			parseAttachJSEventListener(parseWindow, "pointerdown", parseCallback("pointerdown")),
			parseAttachJSEventListener(parseWindow, "keydown", parseCallback("keydown")),
			parseAttachJSEventListener(parseWindow, "focus", parseCallback("focus")),
			parseAttachJSEventListener(parseDocument, "visibilitychange", func() {
				if parseDocument.Truthy() && parseDocument.Get("visibilityState").String() == "visible" {
					parseRefreshSession(false, "visible")
				}
			}),
		}
		return func() {
			for _, parseCleanup := range parseCleanupFns {
				parseCleanup()
			}
		}
	}, parseApp.Get().Authenticated, parseApp.Get().GRPCReady)

	handleModeToggle := ui.UseEvent(func() {
		parseNextMode := authModeSignup
		if parseApp.Get().AuthMode == authModeSignup || parseApp.Get().AuthMode == authModeReset || parseApp.Get().AuthMode == authModeUpdatePassword {
			parseNextMode = authModeLogin
		}
		parseApp.Dispatch(appAction{Type: appActionSetAuthMode, AuthMode: parseNextMode})
		parseApp.Dispatch(appAction{Type: appActionSetAuthError, AuthError: ""})
		parseApp.Dispatch(appAction{Type: appActionSetAuthPassword, AuthPassword: ""})
	})

	handleForgotPassword := ui.UseEvent(func() {
		parseApp.Dispatch(appAction{Type: appActionSetAuthMode, AuthMode: authModeReset})
		parseApp.Dispatch(appAction{Type: appActionSetAuthError, AuthError: ""})
		parseApp.Dispatch(appAction{Type: appActionSetAuthPassword, AuthPassword: ""})
	})

	handleUpdatePassword := ui.UseEvent(func() {
		parseApp.Dispatch(appAction{Type: appActionSetAuthMode, AuthMode: authModeUpdatePassword})
		parseApp.Dispatch(appAction{Type: appActionSetAuthError, AuthError: ""})
		parseApp.Dispatch(appAction{Type: appActionSetAuthPassword, AuthPassword: ""})
	})

	handleEmailInput := ui.UseEvent(func(parseE ui.Event) {
		parseApp.Dispatch(appAction{Type: appActionSetAuthEmail, AuthEmail: parseE.GetValue()})
	})

	handlePasswordInput := ui.UseEvent(func(parseE2 ui.Event) {
		parseApp.Dispatch(appAction{Type: appActionSetAuthPassword, AuthPassword: parseE2.GetValue()})
	})

	handleDisplayNameInput := ui.UseEvent(func(parseE3 ui.Event) {
		parseApp.Dispatch(appAction{Type: appActionSetAuthDisplayName, AuthDisplayName: parseE3.GetValue()})
	})

	parseSubmit := func() {
		parseCurrentState := parseApp.Get()
		if parseCurrentState.AuthSubmitting || !parseCurrentState.GRPCReady {
			return
		}
		parseClient3 := parseChatClientRef.Get()
		if parseClient3 == nil {
			parseApp.Dispatch(appAction{Type: appActionSetAuthError, AuthError: "Connection is not ready yet."})
			return
		}
		parseEmail := strings.TrimSpace(parseCurrentState.AuthEmail)
		parsePassword := parseCurrentState.AuthPassword
		parseDisplayName2 := strings.TrimSpace(parseCurrentState.AuthDisplayName)
		if parseEmail == "" || strings.TrimSpace(parsePassword) == "" {
			parseApp.Dispatch(appAction{Type: appActionSetAuthError, AuthError: "Email and password are required."})
			return
		}
		if parseCurrentState.AuthMode == authModeSignup && len(strings.TrimSpace(parsePassword)) < 8 {
			parseApp.Dispatch(appAction{Type: appActionSetAuthError, AuthError: "Password must be at least 8 characters."})
			return
		}
		parseApp.Dispatch(appAction{Type: appActionSetAuthSubmitting, AuthSubmitting: true})
		parseApp.Dispatch(appAction{Type: appActionSetAuthError, AuthError: ""})
		go func(parseMode string) {
			defer parseApp.Dispatch(appAction{Type: appActionSetAuthSubmitting, AuthSubmitting: false})

			var (
				parseResp2 *chatpb.AuthResponse
				parseErr3  error
			)
			switch parseMode {
			case authModeSignup:
				parseResp2, parseErr3 = parseClient3.Signup(context.Background(), &chatpb.SignupRequest{
					Email:       parseEmail,
					Password:    parsePassword,
					DisplayName: parseDisplayName2,
				})
			default:
				parseResp2, parseErr3 = parseClient3.Login(context.Background(), &chatpb.LoginRequest{
					Email:    parseEmail,
					Password: parsePassword,
				})
			}
			if parseErr3 != nil {
				parseApp.Dispatch(appAction{Type: appActionSetAuthError, AuthError: parseAuthErrorMessage(parseMode, parseErr3)})
				return
			}
			parsePersistAuthToken(parseResp2.GetAuthToken())
			parseUserNameState.Set(parseResp2.GetDisplayName())
			parseApp.Dispatch(appAction{Type: appActionResetWorkspace})
			parseApp.Dispatch(appAction{Type: appActionSetAuthenticated, Authenticated: true})
			parseApp.Dispatch(appAction{Type: appActionSetAuthResolved, AuthResolved: true})
			parseApp.Dispatch(appAction{Type: appActionSetAuthMode, AuthMode: authModeLogin})
			parseApp.Dispatch(appAction{Type: appActionSetAuthError, AuthError: ""})
			parseApp.Dispatch(appAction{Type: appActionSetSessionEmail, SessionEmail: parseResp2.GetEmail()})
			parseApp.Dispatch(appAction{Type: appActionSetAuthDisplayName, AuthDisplayName: ""})
			parseApp.Dispatch(appAction{Type: appActionSetAuthPassword, AuthPassword: ""})
			parseLastRefreshAt.Set(time.Now())
			if parseOnAuthenticated != nil {
				parseOnAuthenticated()
			}
			chatLog.Info("auth success", logging.Fields{"mode": parseMode, "email": parseResp2.GetEmail()})
		}(parseCurrentState.AuthMode)
	}

	handleSubmit := ui.UseEvent(func() {
		parseSubmit()
	})

	handlePasswordKey := ui.UseEvent(func(parseE4 ui.Event) {
		if parseE4.GetKey() == "Enter" && !parseE4.JSValue().Get("shiftKey").Bool() {
			parseE4.PreventDefault()
			parseSubmit()
		}
	})

	parseLogout := ui.UseEvent(func() {
		if parseClient4 := parseChatClientRef.Get(); parseClient4 != nil {
			go func() {
				_, _ = parseClient4.Logout(context.Background(), &emptypb.Empty{})
			}()
		}
		clearPersistedAuthToken()
		parseUserNameState.Set("User")
		parseApp.Dispatch(appAction{Type: appActionResetWorkspace})
		parseApp.Dispatch(appAction{Type: appActionSetAuthenticated, Authenticated: false})
		parseApp.Dispatch(appAction{Type: appActionSetAuthResolved, AuthResolved: true})
		parseApp.Dispatch(appAction{Type: appActionSetAuthMode, AuthMode: authModeLogin})
		parseApp.Dispatch(appAction{Type: appActionSetAuthError, AuthError: ""})
		parseApp.Dispatch(appAction{Type: appActionSetAuthDisplayName, AuthDisplayName: ""})
		parseApp.Dispatch(appAction{Type: appActionSetSessionEmail, SessionEmail: ""})
		parseApp.Dispatch(appAction{Type: appActionSetAuthPassword, AuthPassword: ""})
		if parseOnLogout != nil {
			parseOnLogout()
		}
	})

	return authSessionController{
		HandleModeToggle:       handleModeToggle,
		HandleForgotPassword:   handleForgotPassword,
		HandleUpdatePassword:   handleUpdatePassword,
		HandleEmailInput:       handleEmailInput,
		HandlePasswordInput:    handlePasswordInput,
		HandleDisplayNameInput: handleDisplayNameInput,
		HandleSubmit:           handleSubmit,
		HandlePasswordKey:      handlePasswordKey,
		Logout:                 parseLogout,
	}
}

func parseAuthErrorMessage(parseMode string, parseErr error) string {
	if parseErr == nil {
		return parseAuthDefaultFailureMessage(parseMode)
	}
	if parseRpcStatus, parseOk := status.FromError(parseErr); parseOk {
		return parseAuthStatusMessage(parseMode, parseRpcStatus.Code(), parseRpcStatus.Message())
	}
	parseMessage := parseSanitizeRPCErrorText(parseErr.Error())
	if parseMessage == "" {
		return parseAuthDefaultFailureMessage(parseMode)
	}
	return parseMessage
}

func parseAuthStatusMessage(parseMode string, parseCode codes.Code, parseMessage string) string {
	parseNormalizedMessage := strings.ToLower(strings.TrimSpace(parseMessage))

	switch parseCode {
	case codes.Unauthenticated:
		switch {
		case strings.Contains(parseNormalizedMessage, "invalid email or password"), strings.Contains(parseNormalizedMessage, "invalid credentials"):
			return "That email and password didn't match. Try again."
		case strings.Contains(parseNormalizedMessage, "sign in again"), strings.Contains(parseNormalizedMessage, "session expired"), strings.Contains(parseNormalizedMessage, "authentication required"):
			return authExpiredMessage
		default:
			return "Your session is no longer valid. Please sign in again."
		}
	case codes.AlreadyExists:
		return "An account with that email already exists. Sign in instead or use another email."
	case codes.InvalidArgument:
		switch {
		case strings.Contains(parseNormalizedMessage, "email and password are required"):
			return "Enter your email and password to continue."
		case strings.Contains(parseNormalizedMessage, "password must be at least 8 characters"):
			return "Use at least 8 characters for your password."
		case strings.Contains(parseNormalizedMessage, "memory"), strings.Contains(parseNormalizedMessage, "model"), strings.Contains(parseNormalizedMessage, "thinking"), strings.Contains(parseNormalizedMessage, "tone"):
			return parseSanitizeRPCErrorText(parseMessage)
		default:
			if parseTrimmed := parseSanitizeRPCErrorText(parseMessage); parseTrimmed != "" {
				return parseTrimmed
			}
			return parseAuthDefaultFailureMessage(parseMode)
		}
	case codes.Unavailable:
		return "The service is temporarily unavailable. Try again in a moment."
	case codes.DeadlineExceeded:
		return "That took too long. Please try again."
	case codes.Canceled:
		return "The request was canceled. Please try again."
	case codes.Internal:
		return "Something went wrong on our side. Please try again."
	default:
		if parseTrimmed2 := parseSanitizeRPCErrorText(parseMessage); parseTrimmed2 != "" {
			return parseTrimmed2
		}
		return parseAuthDefaultFailureMessage(parseMode)
	}
}

func parseAuthDefaultFailureMessage(parseMode string) string {
	if parseMode == authModeSignup {
		return "Could not create your account right now."
	}
	return "Could not sign you in right now."
}

func parseSanitizeRPCErrorText(parseMessage string) string {
	parseTrimmed := strings.TrimSpace(parseMessage)
	if parseTrimmed == "" {
		return ""
	}
	parseLower := strings.ToLower(parseTrimmed)
	if strings.HasPrefix(parseLower, "rpc error:") {
		if parseIdx := strings.Index(parseLower, "desc ="); parseIdx >= 0 {
			parseTrimmed = strings.TrimSpace(parseTrimmed[parseIdx+len("desc ="):])
		}
	}
	switch parseTrimmed {
	case "issue auth token":
		return "Could not start your session right now."
	case "auth unavailable":
		return "Sign-in is unavailable right now."
	}
	return parseTrimmed
}

func parseAuthTokenExpiresWithin(parseToken string, parseWindow time.Duration, parseNow time.Time) bool {
	parseExpiry, parseOk := parseAuthTokenExpiry(parseToken)
	if !parseOk {
		return true
	}
	return !parseExpiry.After(parseNow.Add(parseWindow))
}

func parseAuthTokenExpiry(parseToken string) (time.Time, bool) {
	parseParts := strings.Split(strings.TrimSpace(parseToken), ".")
	if len(parseParts) != 3 {
		return time.Time{}, false
	}
	parsePayload, parseErr := base64.RawURLEncoding.DecodeString(parseParts[1])
	if parseErr != nil {
		return time.Time{}, false
	}
	var parseClaims struct {
		ExpiresAt int64 `json:"exp"`
	}
	if parseErr2 := json.Unmarshal(parsePayload, &parseClaims); parseErr2 != nil || parseClaims.ExpiresAt <= 0 {
		return time.Time{}, false
	}
	return time.Unix(parseClaims.ExpiresAt, 0), true
}

func parseAttachJSEventListener(parseTarget js.Value, parseEventName string, parseHandler func()) func() {
	if !parseTarget.Truthy() || parseTarget.Get("addEventListener").Type() != js.TypeFunction {
		return func() {}
	}
	parseListener := js.FuncOf(func(js.Value, []js.Value) interface{} {
		parseHandler()
		return nil
	})
	parseTarget.Call("addEventListener", parseEventName, parseListener)
	return func() {
		parseTarget.Call("removeEventListener", parseEventName, parseListener)
		parseListener.Release()
	}
}
