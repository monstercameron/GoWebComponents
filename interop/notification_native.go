//go:build !(js && wasm)

package interop

import "context"

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

// NotificationPermissionState is unavailable on non-browser builds.
func NotificationPermissionState() (NotificationPermission, error) {
	return NotificationDefault, unavailable("Notification", "")
}

// RequestNotificationPermission is unavailable on non-browser builds.
func RequestNotificationPermission(parseCtx context.Context) (NotificationPermission, error) {
	_ = parseCtx
	return NotificationDefault, unavailable("RequestNotificationPermission", "")
}

// PostNotification is unavailable on non-browser builds.
func PostNotification(parseTitle string, parseBody string) error {
	_ = parseTitle
	_ = parseBody
	return unavailable("PostNotification", "")
}
