//go:build js && wasm

package interop

import (
	"context"
	"errors"
	"syscall/js"
)

// NotificationPermission is the browser Notification permission state.
type NotificationPermission string

const (
	// NotificationDefault means the user has not yet granted or denied permission.
	NotificationDefault NotificationPermission = "default"
	// NotificationGranted means notifications are allowed.
	NotificationGranted NotificationPermission = "granted"
	// NotificationDenied means notifications are blocked.
	NotificationDenied NotificationPermission = "denied"
)

// notificationCtor returns the global Notification constructor, or a structured
// error when the API is unavailable (insecure context, unsupported browser).
func notificationCtor() (js.Value, error) {
	parseCtor := js.Global().Get("Notification")
	if parseCtor.IsUndefined() || parseCtor.IsNull() {
		return js.Value{}, unavailable("Notification", "")
	}
	return parseCtor, nil
}

// NotificationPermissionState reports the current permission without prompting.
func NotificationPermissionState() (NotificationPermission, error) {
	parseCtor, parseErr := notificationCtor()
	if parseErr != nil {
		return NotificationDefault, parseErr
	}
	parsePerm := parseCtor.Get("permission")
	if parsePerm.Type() != js.TypeString {
		return NotificationDefault, nil
	}
	return NotificationPermission(parsePerm.String()), nil
}

// RequestNotificationPermission prompts the user (if needed) and resolves to the
// resulting permission. It awaits the requestPermission() promise through the
// shared Promise→Go bridge, so callers never touch js.FuncOf. Call it from a
// goroutine, not directly inside an event callback.
func RequestNotificationPermission(parseCtx context.Context) (NotificationPermission, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	parseCtor, parseErr := notificationCtor()
	if parseErr != nil {
		return NotificationDefault, parseErr
	}
	parseRequest := parseCtor.Get("requestPermission")
	if parseRequest.Type() != js.TypeFunction {
		// Some very old browsers expose only the synchronous permission field.
		return NotificationPermissionState()
	}
	parsePromise := parseCtor.Call("requestPermission")
	parseResolved, parseErr := awaitValue(parseCtx, "RequestNotificationPermission", "", parsePromise)
	if parseErr != nil {
		return NotificationDefault, parseErr
	}
	if parseResolved.Type() != js.TypeString {
		return NotificationPermissionState()
	}
	return NotificationPermission(parseResolved.String()), nil
}

// PostNotification displays a notification. It returns a structured error when
// the API is unavailable or permission has not been granted — it never silently
// drops, and never prompts (call RequestNotificationPermission first).
func PostNotification(parseTitle string, parseBody string) error {
	parseCtor, parseErr := notificationCtor()
	if parseErr != nil {
		return parseErr
	}
	parsePerm := NotificationPermission(parseCtor.Get("permission").String())
	if parsePerm != NotificationGranted {
		return wrapError("PostNotification", "", CodeUnavailable, errors.New("notification permission not granted"))
	}
	parseOptions := js.Global().Get("Object").New()
	parseOptions.Set("body", parseBody)
	parseCtor.New(parseTitle, parseOptions)
	return nil
}
