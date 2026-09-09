package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"example.com/gwc-wails-counter/contracts"
	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	apiMaxResults       = 256
	apiMaxPaths         = 32
	apiMaxPathLen       = 4096
	apiClipboardFixture = "GoWebComponents API clipboard fixture"
)

// APIService exposes explicit, user-invoked native capability probes.
type APIService struct {
	Files        *desktop.FileDialogHost
	Native       *desktop.NativeHost
	Policy       desktop.FeaturePolicy
	parseMutex   sync.Mutex
	parseFixture string
	parseResults []contracts.APIResult
	parseClosed  bool
}

// NewAPIService creates a private fixture directory for picker actions.
func NewAPIService() (*APIService, error) {
	parsePolicy, _ := desktop.ParseFeaturePolicy("all")
	parseFixture, parseErr := os.MkdirTemp("", "gwc-wails-api-")
	if parseErr != nil {
		return nil, parseErr
	}
	for _, parseName := range []string{"sample-a.txt", "sample-b.txt", "sample-unicode-雪.txt"} {
		if parseErr = os.WriteFile(filepath.Join(parseFixture, parseName), []byte("GoWebComponents API fixture\n"), 0o600); parseErr != nil {
			_ = os.RemoveAll(parseFixture)
			return nil, parseErr
		}
	}
	if parseErr = os.Mkdir(filepath.Join(parseFixture, "folder"), 0o700); parseErr != nil {
		_ = os.RemoveAll(parseFixture)
		return nil, parseErr
	}
	return &APIService{parseFixture: parseFixture, Policy: parsePolicy}, nil
}

// ServiceShutdown closes the service and removes only its private fixture directory.
func (parseService *APIService) ServiceShutdown() error {
	if parseService == nil {
		return nil
	}
	parseService.parseMutex.Lock()
	if parseService.parseClosed {
		parseService.parseMutex.Unlock()
		return nil
	}
	parseService.parseClosed = true
	parseFixture := parseService.parseFixture
	parseService.parseMutex.Unlock()
	return os.RemoveAll(parseFixture)
}

// Run performs one explicitly named native capability action.
func (parseService *APIService) Run(parseContext context.Context, parseAction string) (contracts.APIResult, error) {
	parseResult := parseService.newResult(parseAction)
	if parseContext == nil {
		return parseResult, errors.New("context is required")
	}
	if parseErr := parseService.checkOpen(parseContext); parseErr != nil {
		parseResult.Outcome = "cancelled"
		return parseResult, parseErr
	}
	if parseAction == "" {
		return parseService.finishResult(parseResult, "error", errors.New("action is required"))
	}
	if !isAPIAction(parseAction) {
		return parseService.finishResult(parseResult, "error", errors.New("unknown action"))
	}
	if parseFeature := apiActionFeature(parseAction); parseFeature != "" {
		if !parseService.Policy.Allows(parseFeature) {
			return parseService.finishResult(parseResult, "error", errors.New("desktop feature disabled or unavailable"))
		}
		if parseFeature != desktop.FileDialogs {
			if parseService.Native == nil {
				return parseService.finishResult(parseResult, "error", errors.New("desktop feature disabled or unavailable"))
			}
			if parseErr := parseService.Native.Require(parseFeature); parseErr != nil {
				return parseService.finishResult(parseResult, "error", parseErr)
			}
		}
	}
	parseWindow, parseErr := getCallingWindow(parseContext)
	if parseErr != nil {
		return parseService.finishResult(parseResult, "error", parseErr)
	}
	parseResult.WindowID = fmt.Sprint(parseWindow.ID())
	switch parseAction {
	case "open-file", "open-files", "open-directory", "save-path":
		parseKind := parseAction
		parseOptions := desktop.FileDialogOptions{Title: "API Lab — " + parseAction, Directory: parseService.parseFixture, Filters: []desktop.FileFilter{{Name: "Text files", Pattern: "*.txt"}}}
		if parseAction == "open-directory" {
			parseOptions.Filters = nil
		}
		if parseAction == "save-path" {
			parseKind = "save-file"
			parseOptions.Filename = "api-report.json"
			parseOptions.Filters = []desktop.FileFilter{{Name: "JSON files", Pattern: "*.json"}}
		}
		parseSelection, parseDialogErr := selectFileDialog(parseContext, parseService.Files, parseKind, parseOptions)
		if parseDialogErr != nil {
			return parseService.finishPickerError(parseResult, parseDialogErr)
		}
		if applyPickerSelection(&parseResult, parseAction, parseSelection) == "cancelled" {
			return parseService.finishResult(parseResult, "cancelled", nil)
		}
	case "message-info":
		parseReply, parseNativeErr := parseService.Native.ShowMessage(parseContext, desktop.MessageRequest{Kind: "info", Title: "API Lab — Information", Message: "Native information dialog"})
		if parseNativeErr != nil {
			return parseService.finishResult(parseResult, "error", parseNativeErr)
		}
		parseResult.Detail = parseReply.Button
	case "message-question":
		parseReply, parseNativeErr := parseService.Native.ShowMessage(parseContext, desktop.MessageRequest{Kind: "question", Title: "API Lab — Question", Message: "Continue with the native question?"})
		if parseNativeErr != nil {
			return parseService.finishResult(parseResult, "error", parseNativeErr)
		}
		parseResult.Detail = parseReply.Button
	case "window-info":
		parseInfo, parseNativeErr := parseService.Native.WindowControl(parseContext, desktop.WindowRequest{Action: "info"})
		if parseNativeErr != nil {
			return parseService.finishResult(parseResult, "error", parseNativeErr)
		}
		parseResult.Detail = fmt.Sprintf("id=%s name=%s size=%dx%d", parseInfo.ID, parseInfo.Name, parseInfo.Width, parseInfo.Height)
	case "window-resize":
		parseInfo, parseNativeErr := parseService.Native.WindowControl(parseContext, desktop.WindowRequest{Action: "resize", Width: 720, Height: 520})
		if parseNativeErr != nil {
			return parseService.finishResult(parseResult, "error", parseNativeErr)
		}
		parseResult.Detail = fmt.Sprintf("%dx%d", parseInfo.Width, parseInfo.Height)
	case "window-maximize":
		_, parseNativeErr := parseService.Native.WindowControl(parseContext, desktop.WindowRequest{Action: "maximize"})
		if parseNativeErr != nil {
			return parseService.finishResult(parseResult, "error", parseNativeErr)
		}
		parseResult.Detail = "maximised"
	case "window-restore":
		_, parseNativeErr := parseService.Native.WindowControl(parseContext, desktop.WindowRequest{Action: "restore"})
		if parseNativeErr != nil {
			return parseService.finishResult(parseResult, "error", parseNativeErr)
		}
		parseResult.Detail = "restored"
	case "window-fullscreen":
		_, parseNativeErr := parseService.Native.WindowControl(parseContext, desktop.WindowRequest{Action: "fullscreen"})
		if parseNativeErr != nil {
			return parseService.finishResult(parseResult, "error", parseNativeErr)
		}
		parseResult.Detail = "fullscreen"
	case "window-unfullscreen":
		_, parseNativeErr := parseService.Native.WindowControl(parseContext, desktop.WindowRequest{Action: "unfullscreen"})
		if parseNativeErr != nil {
			return parseService.finishResult(parseResult, "error", parseNativeErr)
		}
		parseResult.Detail = "unfullscreen"
	case "clipboard-write":
		if parseNativeErr := parseService.Native.ClipboardWrite(parseContext, desktop.ClipboardWriteRequest{Text: apiClipboardFixture}); parseNativeErr != nil {
			return parseService.finishResult(parseResult, "error", parseNativeErr)
		}
		parseResult.Detail = "synthetic text written"
	case "clipboard-read":
		parseValue, parseNativeErr := parseService.Native.ClipboardRead(parseContext)
		if parseNativeErr != nil {
			return parseService.finishResult(parseResult, "error", parseNativeErr)
		}
		parseResult.Detail = fmt.Sprintf("matches=%t chars=%d", parseValue == apiClipboardFixture, len([]rune(parseValue)))
	case "screens":
		parseScreens, parseNativeErr := parseService.Native.ListScreens(parseContext)
		if parseNativeErr != nil {
			return parseService.finishResult(parseResult, "error", parseNativeErr)
		}
		parseJSON, parseMarshalErr := json.Marshal(parseScreens)
		if parseMarshalErr != nil {
			return parseService.finishResult(parseResult, "error", parseMarshalErr)
		}
		parseResult.Detail = string(parseJSON)
	}
	if parseResult.Detail == "unknown" {
		return parseService.finishResult(parseResult, "error", errors.New("native dialog button result unavailable"))
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return parseService.finishResult(parseResult, "cancelled", parseErr)
	}
	return parseService.finishResult(parseResult, "completed", nil)
}

// applyPickerSelection adds selection-only detail after distinguishing cancellation.
func applyPickerSelection(parseResult *contracts.APIResult, parseAction string, parseSelection desktop.FileSelection) string {
	if parseSelection.Cancelled {
		parseResult.Detail = ""
		parseResult.Paths = nil
		return "cancelled"
	}
	parseResult.Paths = boundPaths(parseSelection.Paths)
	if parseAction == "save-path" {
		parseResult.Detail = "path selected; no file written"
	}
	return "completed"
}

func apiActionFeature(parseAction string) desktop.Feature {
	switch parseAction {
	case "open-file", "open-files", "open-directory", "save-path":
		return desktop.FileDialogs
	case "clipboard-write", "clipboard-read":
		return desktop.Clipboard
	case "message-info", "message-question":
		return desktop.MessageDialogs
	case "window-info", "window-resize", "window-maximize", "window-restore", "window-fullscreen", "window-unfullscreen":
		return desktop.WindowControls
	case "screens":
		return desktop.Screens
	default:
		return ""
	}
}

// ExportReport writes a report through a native save dialog without overwriting an existing file.
func (parseService *APIService) ExportReport(parseContext context.Context) (contracts.APIResult, error) {
	if parseContext == nil {
		return contracts.APIResult{}, errors.New("context is required")
	}
	if parseErr := parseService.checkOpen(parseContext); parseErr != nil {
		return contracts.APIResult{}, parseErr
	}
	if !parseService.Policy.Allows(desktop.ReportExport) {
		return contracts.APIResult{ID: "export-report", Outcome: "error", Detail: "report export is disabled or unavailable"}, errors.New("report export is disabled or unavailable")
	}
	if parseService.Files == nil || len(parseService.Files.GetMethods()) == 0 {
		return contracts.APIResult{ID: "export-report", Outcome: "error", Detail: "file dialogs are disabled or unavailable"}, errors.New("file dialogs are disabled or unavailable")
	}
	parseResult := parseService.newResult("export-report")
	parseWindow, parseErr := getCallingWindow(parseContext)
	if parseErr != nil {
		return parseResult, parseErr
	}
	parseResult.WindowID = fmt.Sprint(parseWindow.ID())
	parseApp := application.Get()
	if parseApp == nil {
		return parseService.finishResult(parseResult, "error", errors.New("application unavailable"))
	}
	parseSave := parseApp.Dialog.SaveFile()
	parseOptions := &application.SaveFileDialogOptions{Title: "API Lab — Export Report"}
	if parseHome, parseHomeErr := os.UserHomeDir(); parseHomeErr == nil {
		parseOptions.Directory = parseHome
	}
	parseSave.SetOptions(parseOptions)
	parsePath, parseDialogErr := parseSave.SetFilename("api-report.json").AddFilter("JSON files", "*.json").AttachToWindow(parseWindow).PromptForSingleSelection()
	if parseDialogErr != nil {
		return parseService.finishPickerError(parseResult, parseDialogErr)
	}
	if parsePath == "" {
		return parseService.finishResult(parseResult, "cancelled", nil)
	}
	parseReport, parseReportErr := parseService.GetReport(parseContext)
	if parseReportErr != nil {
		return parseService.finishResult(parseResult, "error", parseReportErr)
	}
	parseData, parseMarshalErr := json.MarshalIndent(parseReport, "", "  ")
	if parseMarshalErr != nil {
		return parseService.finishResult(parseResult, "error", parseMarshalErr)
	}
	if parseWriteErr := writeAPIReport(parseContext, parseService.parseFixture, parsePath, append(parseData, '\n')); parseWriteErr != nil {
		return parseService.finishResult(parseResult, "error", parseWriteErr)
	}
	parseResult.Paths = boundPaths([]string{parsePath})
	parseResult.Detail = "report exported"
	return parseService.finishResult(parseResult, "completed", nil)
}

// createExclusive creates a new file and refuses to replace an existing file.
func createExclusive(parsePath string) (*os.File, error) {
	return os.OpenFile(parsePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
}

// GetReport returns a bounded deep-copy snapshot of observations.
func (parseService *APIService) GetReport(parseContext context.Context) (contracts.APIReport, error) {
	if parseContext == nil {
		return contracts.APIReport{}, errors.New("context is required")
	}
	if parseErr := parseService.checkOpen(parseContext); parseErr != nil {
		return contracts.APIReport{}, parseErr
	}
	parseService.parseMutex.Lock()
	defer parseService.parseMutex.Unlock()
	if parseService.parseClosed {
		return contracts.APIReport{}, errors.New("service is closed")
	}
	parseCopy := make([]contracts.APIResult, len(parseService.parseResults))
	for parseIndex, parseValue := range parseService.parseResults {
		parseCopy[parseIndex] = parseValue
		parseCopy[parseIndex].Paths = append([]string(nil), parseValue.Paths...)
	}
	return contracts.APIReport{Platform: runtime.GOOS, WailsVersion: "v3.0.0-beta.17", FixtureDir: parseService.parseFixture, Results: parseCopy}, nil
}

// RecordObservation stores an explicit human-observed result.
func (parseService *APIService) RecordObservation(parseContext context.Context, parseID, parseOutcome, parseDetail string) (contracts.APIResult, error) {
	if parseContext == nil {
		return contracts.APIResult{}, errors.New("context is required")
	}
	if parseErr := parseService.checkOpen(parseContext); parseErr != nil {
		return contracts.APIResult{}, parseErr
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return contracts.APIResult{}, parseErr
	}
	if parseOutcome != "observed-pass" && parseOutcome != "observed-fail" && parseOutcome != "not-tested" {
		return contracts.APIResult{}, errors.New("invalid observation outcome")
	}
	if strings.TrimSpace(parseID) == "" {
		return contracts.APIResult{}, errors.New("observation case is required")
	}
	if len(parseID) > 128 || len(parseDetail) > 2048 {
		return contracts.APIResult{}, errors.New("observation too long")
	}
	parseResult := contracts.APIResult{ID: parseID, Outcome: parseOutcome, Detail: parseDetail}
	if parseWindow, isWindow := parseContext.Value(application.WindowKey).(application.Window); isWindow && parseWindow != nil {
		parseResult.WindowID = fmt.Sprint(parseWindow.ID())
	}
	return parseService.appendAPIResult(parseResult), nil
}

// appendAPIResult records a native menu or shortcut observation in newest-first bounded storage.
func (parseService *APIService) appendAPIResult(parseResult contracts.APIResult) contracts.APIResult {
	parseResult = boundResult(parseResult)
	parseService.parseMutex.Lock()
	defer parseService.parseMutex.Unlock()
	if parseService.parseClosed {
		return parseResult
	}
	if len(parseService.parseResults) == apiMaxResults {
		copy(parseService.parseResults, parseService.parseResults[1:])
		parseService.parseResults[len(parseService.parseResults)-1] = parseResult
	} else {
		parseService.parseResults = append(parseService.parseResults, parseResult)
	}
	return boundResult(parseResult)
}

// newResult creates a timestamped result shell.
func (parseService *APIService) newResult(parseID string) contracts.APIResult {
	return boundResult(contracts.APIResult{ID: parseID})
}

// finishResult records a bounded result and returns its original error.
func (parseService *APIService) finishResult(parseResult contracts.APIResult, parseOutcome string, parseErr error) (contracts.APIResult, error) {
	parseResult.Outcome = parseOutcome
	if parseErr != nil {
		parseResult.Detail = parseErr.Error()
	}
	return parseService.appendAPIResult(parseResult), parseErr
}

// boundResult applies wire-size limits and copies paths.
func boundResult(parseResult contracts.APIResult) contracts.APIResult {
	if parseResult.At == "" {
		parseResult.At = time.Now().UTC().Format(time.RFC3339Nano)
	}
	parseResult.ID = boundAPIText(parseResult.ID, 128)
	parseResult.Detail = boundAPIText(parseResult.Detail, 2048)
	parseResult.Paths = boundPaths(parseResult.Paths)
	return parseResult
}

// boundPaths applies count and path-length limits.
func boundPaths(parsePaths []string) []string {
	if len(parsePaths) > apiMaxPaths {
		parsePaths = parsePaths[:apiMaxPaths]
	}
	parseCopy := make([]string, 0, len(parsePaths))
	for _, parsePath := range parsePaths {
		parsePath = boundAPIText(parsePath, apiMaxPathLen)
		if parsePath != "" {
			parseCopy = append(parseCopy, parsePath)
		}
	}
	return parseCopy
}

// boundAPIText truncates at a valid UTF-8 boundary within the byte quota.
func boundAPIText(parseText string, parseLimit int) string {
	parseText = strings.ToValidUTF8(parseText, "\uFFFD")
	if len(parseText) <= parseLimit {
		return parseText
	}
	parseText = parseText[:parseLimit]
	for !utf8.ValidString(parseText) {
		parseText = parseText[:len(parseText)-1]
	}
	return parseText
}

// isAPIPickerCancelled recognizes beta.17's private Windows CFD cancellation.
// Wails propagates an internal sentinel but exposes no public typed equivalent.
// Match only its exact leaf message; never swallow a joined error or substring.
func isAPIPickerCancelled(parseErr error) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	for parseErr != nil {
		if _, isJoined := parseErr.(interface{ Unwrap() []error }); isJoined {
			return false
		}
		parseNext := errors.Unwrap(parseErr)
		if parseNext == nil {
			return parseErr.Error() == "cancelled by user"
		}
		parseErr = parseNext
	}
	return false
}

// finishPickerError normalizes explicit native dismissal without hiding other failures.
func (parseService *APIService) finishPickerError(parseResult contracts.APIResult, parseErr error) (contracts.APIResult, error) {
	if isAPIPickerCancelled(parseErr) {
		return parseService.finishResult(parseResult, "cancelled", nil)
	}
	return parseService.finishResult(parseResult, "error", parseErr)
}

// validateAPIReportPath rejects temporary-fixture aliases and unsafe Windows file names.
func validateAPIReportPath(parseFixture, parsePath string) (string, error) {
	if !filepath.IsAbs(parsePath) || parseFixture == "" {
		return "", errors.New("report path must be absolute")
	}
	parseName := filepath.Base(parsePath)
	if !filepath.IsLocal(parseName) || isAPIWindowsDeviceName(parseName) || strings.ContainsAny(parseName, ":\x00") || strings.TrimRight(parseName, " .") != parseName || !strings.EqualFold(filepath.Ext(parseName), ".json") {
		return "", errors.New("report path must be a regular .json filename")
	}
	parseFixturePath, parseErr := filepath.EvalSymlinks(parseFixture)
	if parseErr != nil {
		return "", fmt.Errorf("resolve fixture: %w", parseErr)
	}
	parseParent, parseErr := filepath.EvalSymlinks(filepath.Dir(parsePath))
	if parseErr != nil {
		return "", fmt.Errorf("resolve report directory: %w", parseErr)
	}
	parseFixturePath, parseErr = filepath.Abs(parseFixturePath)
	if parseErr != nil {
		return "", parseErr
	}
	parseParent, parseErr = filepath.Abs(parseParent)
	if parseErr != nil {
		return "", parseErr
	}
	parseCompareFixture, parseCompareParent := filepath.Clean(parseFixturePath), filepath.Clean(parseParent)
	if runtime.GOOS == "windows" {
		parseCompareFixture, parseCompareParent = strings.ToLower(parseCompareFixture), strings.ToLower(parseCompareParent)
	}
	if parseCompareParent == parseCompareFixture || strings.HasPrefix(parseCompareParent, parseCompareFixture+string(filepath.Separator)) {
		return "", errors.New("report path cannot be inside temporary fixture")
	}
	return filepath.Join(parseParent, parseName), nil
}

// isAPIWindowsDeviceName rejects legacy DOS device aliases even with a JSON extension.
func isAPIWindowsDeviceName(parseName string) bool {
	parseStem := strings.ToUpper(strings.TrimRight(strings.SplitN(parseName, ".", 2)[0], " "))
	switch parseStem {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$", "CLOCK$":
		return true
	}
	parseRunes := []rune(parseStem)
	return len(parseRunes) == 4 && (strings.HasPrefix(parseStem, "COM") || strings.HasPrefix(parseStem, "LPT")) && strings.ContainsRune("123456789¹²³", parseRunes[3])
}

// writeAPIReport creates only an explicitly selected new durable file and removes partial output.
func writeAPIReport(parseContext context.Context, parseFixture, parsePath string, parseData []byte) error {
	if parseContext == nil {
		return errors.New("context is required")
	}
	parseResolved, parseErr := validateAPIReportPath(parseFixture, parsePath)
	if parseErr != nil {
		return parseErr
	}
	if parseErr = parseContext.Err(); parseErr != nil {
		return parseErr
	}
	parseFile, parseErr := createExclusive(parseResolved)
	if parseErr != nil {
		return parseErr
	}
	isComplete := false
	defer func() {
		_ = parseFile.Close()
		if !isComplete {
			_ = os.Remove(parseResolved)
		}
	}()
	if parseErr = parseContext.Err(); parseErr != nil {
		return parseErr
	}
	if _, parseErr = parseFile.Write(parseData); parseErr != nil {
		return parseErr
	}
	if parseErr = parseFile.Sync(); parseErr != nil {
		return parseErr
	}
	if parseErr = parseFile.Close(); parseErr != nil {
		return parseErr
	}
	isComplete = true
	return nil
}

// selectionOutcome maps an empty native picker result to cancellation.
func selectionOutcome(parsePaths []string) string {
	if len(parsePaths) == 0 {
		return "cancelled"
	}
	return "completed"
}

// checkOpen validates service ownership and context state.
func (parseService *APIService) checkOpen(parseContext context.Context) error {
	if parseService == nil {
		return errors.New("service is nil")
	}
	if parseContext == nil {
		return errors.New("context is required")
	}
	parseService.parseMutex.Lock()
	parseClosed := parseService.parseClosed
	parseService.parseMutex.Unlock()
	if parseClosed {
		return errors.New("service is closed")
	}
	return parseContext.Err()
}

// isAPIAction reports whether an action is supported.
func isAPIAction(parseAction string) bool {
	switch parseAction {
	case "open-file", "open-files", "open-directory", "save-path", "message-info", "message-question", "window-info", "window-resize", "window-maximize", "window-restore", "window-fullscreen", "window-unfullscreen", "clipboard-write", "clipboard-read", "screens":
		return true
	default:
		return false
	}
}
