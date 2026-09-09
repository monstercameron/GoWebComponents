package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// SystemEnvironment identifies read-only OS, architecture, theme, and accent inspection.
const SystemEnvironment Feature = "system-environment"

// ExternalURLs identifies opening validated HTTP(S) URLs in the system browser.
const ExternalURLs Feature = "external-urls"

// Autostart identifies explicit application login-startup inspection and control.
const Autostart Feature = "autostart"

// FileManager identifies revealing trusted local paths in the system file manager.
const FileManager Feature = "file-manager"

const (
	SystemEnvironmentMethod = "desktop.system.environment"
	ExternalURLOpenMethod   = "desktop.url.open"
	AutostartStatusMethod   = "desktop.autostart.status"
	AutostartEnableMethod   = "desktop.autostart.enable"
	AutostartDisableMethod  = "desktop.autostart.disable"
	FileManagerRevealMethod = "desktop.file-manager.reveal"
)

// SystemEnvironmentInfo reports bounded, portable Windows environment metadata.
type SystemEnvironmentInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Theme        string `json:"theme"`
	AccentColor  string `json:"accentColor"`
}

// ExternalURLOpenRequest carries one absolute HTTP(S) URL.
type ExternalURLOpenRequest struct {
	URL string `json:"url"`
}

// FileManagerRevealRequest describes one absolute local path to reveal.
type FileManagerRevealRequest struct {
	Path       string `json:"path"`
	SelectFile bool   `json:"selectFile,omitempty"`
}

// AutostartStatus reports whether the current application launches at login.
type AutostartStatus struct {
	Enabled  bool   `json:"enabled"`
	Path     string `json:"path,omitempty"`
	Strategy string `json:"strategy,omitempty"`
}

// SystemEnvironmentBackend is the optional backend extension for read-only environment inspection.
type SystemEnvironmentBackend interface {
	SystemEnvironment(context.Context) (SystemEnvironmentInfo, error)
}

// ExternalURLBackend is the optional backend extension for opening validated web URLs.
type ExternalURLBackend interface {
	OpenExternalURL(context.Context, ExternalURLOpenRequest) error
}

// AutostartBackend is the optional backend extension for explicit login-startup control.
type AutostartBackend interface {
	AutostartStatus(context.Context) (AutostartStatus, error)
	AutostartEnable(context.Context) error
	AutostartDisable(context.Context) error
}

// FileManagerBackend is the optional backend extension for revealing trusted local paths.
type FileManagerBackend interface {
	RevealPath(context.Context, FileManagerRevealRequest) error
}

// InspectSystemEnvironment returns typed host environment information.
func (parseHost *NativeHost) InspectSystemEnvironment(parseContext context.Context) (SystemEnvironmentInfo, error) {
	if parseErr := parseHost.Require(SystemEnvironment); parseErr != nil {
		return SystemEnvironmentInfo{}, parseErr
	}
	parseBackend, parseOK := parseHost.parseBackend.(SystemEnvironmentBackend)
	if !parseOK {
		return SystemEnvironmentInfo{}, getError(SystemEnvironmentMethod, interop.CodeUnavailable, errors.New("system environment backend unavailable"))
	}
	if parseErr := validateSystemContext(parseContext, SystemEnvironmentMethod); parseErr != nil {
		return SystemEnvironmentInfo{}, parseErr
	}
	parseInfo, parseErr := parseBackend.SystemEnvironment(parseContext)
	if parseContext.Err() != nil {
		return SystemEnvironmentInfo{}, getContextError(SystemEnvironmentMethod, parseContext.Err())
	}
	if parseErr != nil {
		return SystemEnvironmentInfo{}, parseErr
	}
	if parseErr = validateSystemEnvironment(parseInfo); parseErr != nil {
		return SystemEnvironmentInfo{}, getError(SystemEnvironmentMethod, interop.CodeDecode, parseErr)
	}
	return parseInfo, nil
}

// OpenExternalURL opens an explicitly requested absolute HTTP(S) URL.
func (parseHost *NativeHost) OpenExternalURL(parseContext context.Context, parseRequest ExternalURLOpenRequest) error {
	if parseErr := parseHost.Require(ExternalURLs); parseErr != nil {
		return parseErr
	}
	parseBackend, parseOK := parseHost.parseBackend.(ExternalURLBackend)
	if !parseOK {
		return getError(ExternalURLOpenMethod, interop.CodeUnavailable, errors.New("external URL backend unavailable"))
	}
	if parseErr := validateSystemContext(parseContext, ExternalURLOpenMethod); parseErr != nil {
		return parseErr
	}
	if parseErr := validateExternalURL(parseRequest.URL); parseErr != nil {
		return getError(ExternalURLOpenMethod, interop.CodeInvalid, parseErr)
	}
	parseErr := parseBackend.OpenExternalURL(parseContext, parseRequest)
	if parseContext.Err() != nil {
		return getContextError(ExternalURLOpenMethod, parseContext.Err())
	}
	return parseErr
}

// RevealPath reveals an existing absolute local Windows path in the system file manager.
func (parseHost *NativeHost) RevealPath(parseContext context.Context, parseRequest FileManagerRevealRequest) error {
	if parseErr := parseHost.Require(FileManager); parseErr != nil {
		return parseErr
	}
	parseBackend, parseOK := parseHost.parseBackend.(FileManagerBackend)
	if !parseOK {
		return getError(FileManagerRevealMethod, interop.CodeUnavailable, errors.New("file manager backend unavailable"))
	}
	if parseErr := validateSystemContext(parseContext, FileManagerRevealMethod); parseErr != nil {
		return parseErr
	}
	if parseErr := validateFileManagerPath(parseRequest.Path); parseErr != nil {
		return getError(FileManagerRevealMethod, interop.CodeInvalid, parseErr)
	}
	parseInfo, parseErr := os.Stat(parseRequest.Path)
	if parseErr != nil {
		return getError(FileManagerRevealMethod, interop.CodeInvalid, errors.New("file manager path does not exist"))
	}
	if parseRequest.SelectFile && parseInfo.IsDir() {
		return getError(FileManagerRevealMethod, interop.CodeInvalid, errors.New("SelectFile requires a file path"))
	}
	parseErr = parseBackend.RevealPath(parseContext, parseRequest)
	if parseContext.Err() != nil {
		return getContextError(FileManagerRevealMethod, parseContext.Err())
	}
	return parseErr
}

// GetAutostartStatus inspects the current application's login-startup registration.
func (parseHost *NativeHost) GetAutostartStatus(parseContext context.Context) (AutostartStatus, error) {
	if parseErr := parseHost.Require(Autostart); parseErr != nil {
		return AutostartStatus{}, parseErr
	}
	parseBackend, parseOK := parseHost.parseBackend.(AutostartBackend)
	if !parseOK {
		return AutostartStatus{}, getError(AutostartStatusMethod, interop.CodeUnavailable, errors.New("autostart backend unavailable"))
	}
	if parseErr := validateSystemContext(parseContext, AutostartStatusMethod); parseErr != nil {
		return AutostartStatus{}, parseErr
	}
	parseStatus, parseErr := parseBackend.AutostartStatus(parseContext)
	if parseContext.Err() != nil {
		return AutostartStatus{}, getContextError(AutostartStatusMethod, parseContext.Err())
	}
	if parseErr != nil {
		return AutostartStatus{}, parseErr
	}
	if parseErr = validateAutostartStatus(parseStatus); parseErr != nil {
		return AutostartStatus{}, getError(AutostartStatusMethod, interop.CodeDecode, parseErr)
	}
	return parseStatus, nil
}

// EnableAutostart explicitly registers the current application to launch at login.
func (parseHost *NativeHost) EnableAutostart(parseContext context.Context) error {
	return parseHost.setAutostart(parseContext, true)
}

// DisableAutostart explicitly removes the current application's login-startup registration.
func (parseHost *NativeHost) DisableAutostart(parseContext context.Context) error {
	return parseHost.setAutostart(parseContext, false)
}

// setAutostart applies one explicitly requested autostart state change.
func (parseHost *NativeHost) setAutostart(parseContext context.Context, isEnabled bool) error {
	if parseErr := parseHost.Require(Autostart); parseErr != nil {
		return parseErr
	}
	parseBackend, parseOK := parseHost.parseBackend.(AutostartBackend)
	if !parseOK {
		return getError(AutostartEnableMethod, interop.CodeUnavailable, errors.New("autostart backend unavailable"))
	}
	parseMethod := AutostartDisableMethod
	if isEnabled {
		parseMethod = AutostartEnableMethod
	}
	if parseErr := validateSystemContext(parseContext, parseMethod); parseErr != nil {
		return parseErr
	}
	var parseErr error
	if isEnabled {
		parseErr = parseBackend.AutostartEnable(parseContext)
	} else {
		parseErr = parseBackend.AutostartDisable(parseContext)
	}
	if parseContext.Err() != nil {
		return getContextError(parseMethod, parseContext.Err())
	}
	return parseErr
}

// InspectSystemEnvironment invokes typed host environment inspection.
func (parseClient Client) InspectSystemEnvironment(parseContext context.Context) (SystemEnvironmentInfo, error) {
	if parseErr := parseClient.Require(SystemEnvironment); parseErr != nil {
		return SystemEnvironmentInfo{}, parseErr
	}
	parseReply, parseErr := parseClient.getNativeReply(parseContext, SystemEnvironmentMethod)
	if parseErr != nil {
		return SystemEnvironmentInfo{}, parseErr
	}
	var parseInfo SystemEnvironmentInfo
	if parseErr = json.Unmarshal(parseReply.Data, &parseInfo); parseErr != nil {
		return SystemEnvironmentInfo{}, getError(SystemEnvironmentMethod, interop.CodeDecode, parseErr)
	}
	if parseErr = validateSystemEnvironment(parseInfo); parseErr != nil {
		return SystemEnvironmentInfo{}, getError(SystemEnvironmentMethod, interop.CodeDecode, parseErr)
	}
	return parseInfo, nil
}

// OpenExternalURL invokes the typed external URL host method.
func (parseClient Client) OpenExternalURL(parseContext context.Context, parseURL string) error {
	if parseErr := validateExternalURL(parseURL); parseErr != nil {
		return getError(ExternalURLOpenMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := parseClient.Require(ExternalURLs); parseErr != nil {
		return parseErr
	}
	_, parseErr := parseClient.getNativeReply(parseContext, ExternalURLOpenMethod, ExternalURLOpenRequest{URL: parseURL})
	return parseErr
}

// RevealPath asks the host to reveal an existing absolute local Windows path.
func (parseClient Client) RevealPath(parseContext context.Context, parsePath string, isSelectFile bool) error {
	if parseErr := validateFileManagerPath(parsePath); parseErr != nil {
		return getError(FileManagerRevealMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := parseClient.Require(FileManager); parseErr != nil {
		return parseErr
	}
	_, parseErr := parseClient.getNativeReply(parseContext, FileManagerRevealMethod, FileManagerRevealRequest{Path: parsePath, SelectFile: isSelectFile})
	return parseErr
}

// GetAutostartStatus invokes typed host autostart inspection.
func (parseClient Client) GetAutostartStatus(parseContext context.Context) (AutostartStatus, error) {
	if parseErr := parseClient.Require(Autostart); parseErr != nil {
		return AutostartStatus{}, parseErr
	}
	parseReply, parseErr := parseClient.getNativeReply(parseContext, AutostartStatusMethod)
	if parseErr != nil {
		return AutostartStatus{}, parseErr
	}
	var parseStatus AutostartStatus
	if parseErr = json.Unmarshal(parseReply.Data, &parseStatus); parseErr != nil {
		return AutostartStatus{}, getError(AutostartStatusMethod, interop.CodeDecode, parseErr)
	}
	if parseErr = validateAutostartStatus(parseStatus); parseErr != nil {
		return AutostartStatus{}, getError(AutostartStatusMethod, interop.CodeDecode, parseErr)
	}
	return parseStatus, nil
}

// EnableAutostart invokes the explicit host autostart enable operation.
func (parseClient Client) EnableAutostart(parseContext context.Context) error {
	return parseClient.setAutostart(parseContext, AutostartEnableMethod)
}

// DisableAutostart invokes the explicit host autostart disable operation.
func (parseClient Client) DisableAutostart(parseContext context.Context) error {
	return parseClient.setAutostart(parseContext, AutostartDisableMethod)
}

// setAutostart invokes one explicit no-argument autostart method.
func (parseClient Client) setAutostart(parseContext context.Context, parseMethod string) error {
	if parseErr := parseClient.Require(Autostart); parseErr != nil {
		return parseErr
	}
	_, parseErr := parseClient.getNativeReply(parseContext, parseMethod)
	return parseErr
}

// validateSystemContext rejects absent and completed caller contexts before backend work.
func validateSystemContext(parseContext context.Context, parseMethod string) error {
	if parseContext == nil {
		return getError(parseMethod, interop.CodeInvalid, errors.New("calling context required"))
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return getContextError(parseMethod, parseErr)
	}
	return nil
}

// validateExternalURL permits only bounded, absolute HTTP(S) URLs without credentials.
func validateExternalURL(parseValue string) error {
	if parseValue == "" || len(parseValue) > 8192 || strings.ContainsRune(parseValue, 0) {
		return errors.New("invalid or oversized external URL")
	}
	parseURL, parseErr := url.ParseRequestURI(parseValue)
	if parseErr != nil || parseURL.IsAbs() == false || (parseURL.Scheme != "https" && parseURL.Scheme != "http") || parseURL.Host == "" || parseURL.User != nil {
		return errors.New("external URL must be an absolute HTTP(S) URL without credentials")
	}
	return nil
}

// validateFileManagerPath accepts only bounded drive-qualified local Windows paths.
func validateFileManagerPath(parseValue string) error {
	if parseValue == "" || len(parseValue) > 32768 || strings.ContainsRune(parseValue, 0) || strings.HasPrefix(parseValue, `\\`) || strings.HasPrefix(parseValue, "//") || strings.Contains(parseValue, "://") {
		return errors.New("file manager path must be an absolute local Windows path")
	}
	if len(parseValue) < 3 || !isWindowsDriveLetter(parseValue[0]) || parseValue[1] != ':' || (parseValue[2] != '\\' && parseValue[2] != '/') || strings.Contains(parseValue[2:], ":") {
		return errors.New("file manager path must be an absolute local Windows path")
	}
	parsePath := strings.ReplaceAll(parseValue[3:], `\`, "/")
	for _, parseSegment := range strings.Split(parsePath, "/") {
		if parseSegment == "." || parseSegment == ".." {
			return errors.New("file manager path traversal is not allowed")
		}
	}
	return nil
}

// isWindowsDriveLetter reports whether a byte can prefix a local drive path.
func isWindowsDriveLetter(parseValue byte) bool {
	return parseValue >= 'A' && parseValue <= 'Z' || parseValue >= 'a' && parseValue <= 'z'
}

// validateSystemEnvironment rejects malformed backend environment metadata.
func validateSystemEnvironment(parseInfo SystemEnvironmentInfo) error {
	if parseInfo.OS != "windows" || parseInfo.Architecture == "" || len(parseInfo.Architecture) > 64 || (parseInfo.Theme != "light" && parseInfo.Theme != "dark") || len(parseInfo.AccentColor) > 64 || strings.ContainsRune(parseInfo.AccentColor, 0) {
		return errors.New("invalid system environment information")
	}
	return nil
}

// validateAutostartStatus rejects inconsistent or unbounded backend state.
func validateAutostartStatus(parseStatus AutostartStatus) error {
	if len(parseStatus.Path) > 32768 || len(parseStatus.Strategy) > 128 || strings.ContainsRune(parseStatus.Path, 0) || strings.ContainsRune(parseStatus.Strategy, 0) || (!parseStatus.Enabled && (parseStatus.Path != "" || parseStatus.Strategy != "")) {
		return errors.New("invalid autostart status")
	}
	return nil
}
