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
const authExpiredMessage = "Session expired. Sign in again."

type authSessionController struct {
	HandleModeToggle       ui.Handler
	HandleEmailInput       ui.Handler
	HandlePasswordInput    ui.Handler
	HandleDisplayNameInput ui.Handler
	HandleSubmit           ui.Handler
	HandlePasswordKey      ui.Handler
	Logout                 ui.Handler
}

func handleUnauthenticatedRPC(app ui.Reducer[appState, appAction], userNameState state.Atom[string], err error) bool {
	if status.Code(err) != codes.Unauthenticated {
		return false
	}
	sessionEmail := app.Get().SessionEmail
	clearPersistedAuthToken()
	userNameState.Set("User")
	app.Dispatch(appAction{Type: appActionResetWorkspace})
	app.Dispatch(appAction{Type: appActionSetAuthenticated, Authenticated: false})
	app.Dispatch(appAction{Type: appActionSetAuthResolved, AuthResolved: true})
	app.Dispatch(appAction{Type: appActionSetAuthMode, AuthMode: authModeLogin})
	app.Dispatch(appAction{Type: appActionSetAuthSubmitting, AuthSubmitting: false})
	app.Dispatch(appAction{Type: appActionSetAuthError, AuthError: authExpiredMessage})
	app.Dispatch(appAction{Type: appActionSetAuthEmail, AuthEmail: sessionEmail})
	app.Dispatch(appAction{Type: appActionSetAuthPassword, AuthPassword: ""})
	app.Dispatch(appAction{Type: appActionSetAuthDisplayName, AuthDisplayName: ""})
	app.Dispatch(appAction{Type: appActionSetSessionEmail, SessionEmail: ""})
	return true
}

func loadPersistedAuthToken() string {
	storage, err := interop.LocalStorage()
	if err != nil {
		return ""
	}
	value, ok, err := storage.GetItem(storageKeyAuthToken)
	if err != nil || !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func persistAuthToken(token string) {
	storage, err := interop.LocalStorage()
	if err != nil {
		return
	}
	token = strings.TrimSpace(token)
	if token == "" {
		_ = storage.RemoveItem(storageKeyAuthToken)
		return
	}
	_ = storage.SetItem(storageKeyAuthToken, token)
}

func clearPersistedAuthToken() {
	storage, err := interop.LocalStorage()
	if err != nil {
		return
	}
	_ = storage.RemoveItem(storageKeyAuthToken)
}

func useAuthSession(
	app ui.Reducer[appState, appAction],
	userNameState state.Atom[string],
	chatClientRef ui.Ref[chatpb.ChatServiceClient],
	onAuthenticated func(),
	onLogout func(),
) authSessionController {
	lastRefreshAt := ui.UseRef(time.Time{})
	refreshInFlight := ui.UseRef(false)

	refreshSession := func(force bool, reason string) {
		if !app.Get().Authenticated || !app.Get().GRPCReady {
			return
		}
		client := chatClientRef.Get()
		if client == nil {
			return
		}
		token := loadPersistedAuthToken()
		if token == "" {
			return
		}
		now := time.Now()
		if !force && !authTokenExpiresWithin(token, authRefreshLeadTime, now) {
			return
		}
		if !force && !lastRefreshAt.Get().IsZero() && now.Sub(lastRefreshAt.Get()) < authRefreshMinInterval {
			return
		}
		if refreshInFlight.Get() {
			return
		}
		refreshInFlight.Set(true)
		go func() {
			defer refreshInFlight.Set(false)
			resp, err := client.RefreshSession(context.Background(), &emptypb.Empty{})
			if err != nil {
				if handleUnauthenticatedRPC(app, userNameState, err) {
					chatLog.Warn("auth refresh expired session", logging.Fields{"reason": reason})
					if onLogout != nil {
						onLogout()
					}
					return
				}
				chatLog.Warn("auth refresh failed", logging.Fields{"error": err, "reason": reason})
				return
			}
			if strings.TrimSpace(resp.GetAuthToken()) == "" {
				return
			}
			persistAuthToken(resp.GetAuthToken())
			lastRefreshAt.Set(time.Now())
			if displayName := strings.TrimSpace(resp.GetDisplayName()); displayName != "" {
				userNameState.Set(displayName)
			}
			app.Dispatch(appAction{Type: appActionSetSessionEmail, SessionEmail: resp.GetEmail()})
			chatLog.Info("auth refresh", logging.Fields{"reason": reason, "email": resp.GetEmail()})
		}()
	}

	ui.UseEffect(func() func() {
		if !app.Get().GRPCReady {
			return nil
		}
		token := loadPersistedAuthToken()
		if token == "" {
			app.Dispatch(appAction{Type: appActionSetAuthenticated, Authenticated: false})
			app.Dispatch(appAction{Type: appActionSetAuthResolved, AuthResolved: true})
			app.Dispatch(appAction{Type: appActionSetSessionEmail, SessionEmail: ""})
			return nil
		}
		client := chatClientRef.Get()
		if client == nil {
			return nil
		}
		go func() {
			session, err := client.GetSession(context.Background(), &emptypb.Empty{})
			if err != nil || !session.GetAuthenticated() {
				clearPersistedAuthToken()
				app.Dispatch(appAction{Type: appActionSetAuthenticated, Authenticated: false})
				app.Dispatch(appAction{Type: appActionSetAuthResolved, AuthResolved: true})
				app.Dispatch(appAction{Type: appActionSetAuthError, AuthError: ""})
				app.Dispatch(appAction{Type: appActionSetSessionEmail, SessionEmail: ""})
				userNameState.Set("User")
				if onLogout != nil {
					onLogout()
				}
				return
			}
			userNameState.Set(session.GetDisplayName())
			app.Dispatch(appAction{Type: appActionSetAuthenticated, Authenticated: true})
			app.Dispatch(appAction{Type: appActionSetAuthResolved, AuthResolved: true})
			app.Dispatch(appAction{Type: appActionSetAuthMode, AuthMode: authModeLogin})
			app.Dispatch(appAction{Type: appActionSetAuthError, AuthError: ""})
			app.Dispatch(appAction{Type: appActionSetSessionEmail, SessionEmail: session.GetEmail()})
			lastRefreshAt.Set(time.Now())
			if onAuthenticated != nil {
				onAuthenticated()
			}
		}()
		return nil
	}, app.Get().GRPCReady)

	ui.UseEffect(func() func() {
		if !app.Get().Authenticated || !app.Get().GRPCReady {
			return nil
		}
		window := js.Global().Get("window")
		document := js.Global().Get("document")
		callback := func(reason string) func() {
			return func() {
				refreshSession(false, reason)
			}
		}
		cleanupFns := []func(){
			attachJSEventListener(window, "pointerdown", callback("pointerdown")),
			attachJSEventListener(window, "keydown", callback("keydown")),
			attachJSEventListener(window, "focus", callback("focus")),
			attachJSEventListener(document, "visibilitychange", func() {
				if document.Truthy() && document.Get("visibilityState").String() == "visible" {
					refreshSession(false, "visible")
				}
			}),
		}
		return func() {
			for _, cleanup := range cleanupFns {
				cleanup()
			}
		}
	}, app.Get().Authenticated, app.Get().GRPCReady)

	handleModeToggle := ui.UseEvent(func() {
		nextMode := authModeSignup
		if app.Get().AuthMode == authModeSignup {
			nextMode = authModeLogin
		}
		app.Dispatch(appAction{Type: appActionSetAuthMode, AuthMode: nextMode})
		app.Dispatch(appAction{Type: appActionSetAuthError, AuthError: ""})
		app.Dispatch(appAction{Type: appActionSetAuthPassword, AuthPassword: ""})
	})

	handleEmailInput := ui.UseEvent(func(e ui.Event) {
		app.Dispatch(appAction{Type: appActionSetAuthEmail, AuthEmail: e.GetValue()})
	})

	handlePasswordInput := ui.UseEvent(func(e ui.Event) {
		app.Dispatch(appAction{Type: appActionSetAuthPassword, AuthPassword: e.GetValue()})
	})

	handleDisplayNameInput := ui.UseEvent(func(e ui.Event) {
		app.Dispatch(appAction{Type: appActionSetAuthDisplayName, AuthDisplayName: e.GetValue()})
	})

	submit := func() {
		currentState := app.Get()
		if currentState.AuthSubmitting || !currentState.GRPCReady {
			return
		}
		client := chatClientRef.Get()
		if client == nil {
			app.Dispatch(appAction{Type: appActionSetAuthError, AuthError: "Connection is not ready yet."})
			return
		}
		email := strings.TrimSpace(currentState.AuthEmail)
		password := currentState.AuthPassword
		displayName := strings.TrimSpace(currentState.AuthDisplayName)
		if email == "" || strings.TrimSpace(password) == "" {
			app.Dispatch(appAction{Type: appActionSetAuthError, AuthError: "Email and password are required."})
			return
		}
		if currentState.AuthMode == authModeSignup && len(strings.TrimSpace(password)) < 8 {
			app.Dispatch(appAction{Type: appActionSetAuthError, AuthError: "Password must be at least 8 characters."})
			return
		}
		app.Dispatch(appAction{Type: appActionSetAuthSubmitting, AuthSubmitting: true})
		app.Dispatch(appAction{Type: appActionSetAuthError, AuthError: ""})
		go func(mode string) {
			defer app.Dispatch(appAction{Type: appActionSetAuthSubmitting, AuthSubmitting: false})

			var (
				resp *chatpb.AuthResponse
				err  error
			)
			switch mode {
			case authModeSignup:
				resp, err = client.Signup(context.Background(), &chatpb.SignupRequest{
					Email:       email,
					Password:    password,
					DisplayName: displayName,
				})
			default:
				resp, err = client.Login(context.Background(), &chatpb.LoginRequest{
					Email:    email,
					Password: password,
				})
			}
			if err != nil {
				app.Dispatch(appAction{Type: appActionSetAuthError, AuthError: authErrorMessage(mode, err)})
				return
			}
			persistAuthToken(resp.GetAuthToken())
			userNameState.Set(resp.GetDisplayName())
			app.Dispatch(appAction{Type: appActionResetWorkspace})
			app.Dispatch(appAction{Type: appActionSetAuthenticated, Authenticated: true})
			app.Dispatch(appAction{Type: appActionSetAuthResolved, AuthResolved: true})
			app.Dispatch(appAction{Type: appActionSetAuthMode, AuthMode: authModeLogin})
			app.Dispatch(appAction{Type: appActionSetAuthError, AuthError: ""})
			app.Dispatch(appAction{Type: appActionSetSessionEmail, SessionEmail: resp.GetEmail()})
			app.Dispatch(appAction{Type: appActionSetAuthDisplayName, AuthDisplayName: ""})
			app.Dispatch(appAction{Type: appActionSetAuthPassword, AuthPassword: ""})
			lastRefreshAt.Set(time.Now())
			if onAuthenticated != nil {
				onAuthenticated()
			}
			chatLog.Info("auth success", logging.Fields{"mode": mode, "email": resp.GetEmail()})
		}(currentState.AuthMode)
	}

	handleSubmit := ui.UseEvent(func() {
		submit()
	})

	handlePasswordKey := ui.UseEvent(func(e ui.Event) {
		if e.GetKey() == "Enter" && !e.JSValue().Get("shiftKey").Bool() {
			e.PreventDefault()
			submit()
		}
	})

	logout := ui.UseEvent(func() {
		if client := chatClientRef.Get(); client != nil {
			go func() {
				_, _ = client.Logout(context.Background(), &emptypb.Empty{})
			}()
		}
		clearPersistedAuthToken()
		userNameState.Set("User")
		app.Dispatch(appAction{Type: appActionResetWorkspace})
		app.Dispatch(appAction{Type: appActionSetAuthenticated, Authenticated: false})
		app.Dispatch(appAction{Type: appActionSetAuthResolved, AuthResolved: true})
		app.Dispatch(appAction{Type: appActionSetAuthMode, AuthMode: authModeLogin})
		app.Dispatch(appAction{Type: appActionSetAuthError, AuthError: ""})
		app.Dispatch(appAction{Type: appActionSetAuthDisplayName, AuthDisplayName: ""})
		app.Dispatch(appAction{Type: appActionSetSessionEmail, SessionEmail: ""})
		app.Dispatch(appAction{Type: appActionSetAuthPassword, AuthPassword: ""})
		if onLogout != nil {
			onLogout()
		}
	})

	return authSessionController{
		HandleModeToggle:       handleModeToggle,
		HandleEmailInput:       handleEmailInput,
		HandlePasswordInput:    handlePasswordInput,
		HandleDisplayNameInput: handleDisplayNameInput,
		HandleSubmit:           handleSubmit,
		HandlePasswordKey:      handlePasswordKey,
		Logout:                 logout,
	}
}

func authErrorMessage(mode string, err error) string {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		if mode == authModeSignup {
			return "Could not create the account."
		}
		return "Could not sign in."
	}
	return message
}

func authTokenExpiresWithin(token string, window time.Duration, now time.Time) bool {
	expiry, ok := authTokenExpiry(token)
	if !ok {
		return true
	}
	return !expiry.After(now.Add(window))
}

func authTokenExpiry(token string) (time.Time, bool) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		ExpiresAt int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.ExpiresAt <= 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.ExpiresAt, 0), true
}

func attachJSEventListener(target js.Value, eventName string, handler func()) func() {
	if !target.Truthy() || target.Get("addEventListener").Type() != js.TypeFunction {
		return func() {}
	}
	listener := js.FuncOf(func(js.Value, []js.Value) interface{} {
		handler()
		return nil
	})
	target.Call("addEventListener", eventName, listener)
	return func() {
		target.Call("removeEventListener", eventName, listener)
		listener.Release()
	}
}
