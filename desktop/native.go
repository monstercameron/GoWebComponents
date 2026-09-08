package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/monstercameron/GoWebComponents/v5/interop"
)

// Clipboard identifies explicit text clipboard operations.
const Clipboard Feature = "clipboard"

// MessageDialogs identifies native information and question dialogs.
const MessageDialogs Feature = "message-dialogs"

// WindowControls identifies caller-owned window inspection and controls.
const WindowControls Feature = "window-controls"

// Screens identifies display enumeration.
const Screens Feature = "screens"

// NativeMenus identifies native application menu and shortcut callbacks.
const NativeMenus Feature = "native-menus"

// PersistentStorage identifies the host-owned durable storage service.
const PersistentStorage Feature = "persistent-storage"

// ReportExport identifies explicit native report export.
const ReportExport Feature = "report-export"

const (
	ClipboardWriteMethod  = "desktop.clipboard.write"
	ClipboardReadMethod   = "desktop.clipboard.read"
	MessageMethod         = "desktop.message.show"
	WindowMethod          = "desktop.window.control"
	ScreensMethod         = "desktop.screens.list"
	NativeContractVersion = 1
	ReportExportMethod    = "desktop.report.export"
	StorageLoadMethod     = "storage.load"
	StorageSaveMethod     = "storage.save"
	StorageDeleteMethod   = "storage.delete"
	StorageKeysMethod     = "storage.keys"
)

// NativeRequest is the versioned envelope used by generated host bindings.
type NativeRequest struct {
	Version int             `json:"version"`
	Method  string          `json:"method"`
	Args    json.RawMessage `json:"args"`
}

// NativeReply is the structured result envelope; native failures never rely on thrown errors.
type NativeReply struct {
	Version int               `json:"version"`
	Data    json.RawMessage   `json:"data,omitempty"`
	Code    interop.ErrorCode `json:"code,omitempty"`
	Message string            `json:"message,omitempty"`
}

// FeaturePolicy is an immutable allowlist parsed from host configuration.
type FeaturePolicy struct {
	parseAllowed map[Feature]bool
	parseAll     bool
}

// ParseFeaturePolicy parses all, none, or a comma-separated feature allowlist.
func ParseFeaturePolicy(parseValue string) (FeaturePolicy, error) {
	parsePolicy := FeaturePolicy{parseAllowed: map[Feature]bool{}}
	parseValue = strings.TrimSpace(strings.ToLower(parseValue))
	if parseValue == "" || parseValue == "none" {
		return parsePolicy, nil
	}
	if parseValue == "all" {
		parsePolicy.parseAll = true
		return parsePolicy, nil
	}
	for _, parseName := range strings.Split(parseValue, ",") {
		parseFeature := Feature(strings.TrimSpace(parseName))
		if !isNativeFeature(parseFeature) {
			return FeaturePolicy{}, fmt.Errorf("unknown desktop feature %q", parseName)
		}
		parsePolicy.parseAllowed[parseFeature] = true
	}
	return parsePolicy, nil
}

// Allows reports whether this policy permits one feature.
func (parsePolicy FeaturePolicy) Allows(parseFeature Feature) bool {
	if !isNativeFeature(parseFeature) {
		return false
	}
	return parsePolicy.parseAll || parsePolicy.parseAllowed[parseFeature]
}

// Intersect combines policy and backend availability without granting either side new access.
func (parsePolicy FeaturePolicy) Intersect(parseFeatures []Feature) FeaturePolicy {
	parseResult := FeaturePolicy{parseAllowed: map[Feature]bool{}}
	for _, parseFeature := range parseFeatures {
		if parsePolicy.Allows(parseFeature) {
			parseResult.parseAllowed[parseFeature] = true
		}
	}
	return parseResult
}

// FeatureNames returns a stable copy of allowed feature names.
func (parsePolicy FeaturePolicy) FeatureNames() []Feature {
	parseNames := make([]Feature, 0, len(nativeFeatures))
	for _, parseFeature := range nativeFeatures {
		if parsePolicy.Allows(parseFeature) {
			parseNames = append(parseNames, parseFeature)
		}
	}
	return parseNames
}

var nativeFeatures = []Feature{FileDialogs, Clipboard, MessageDialogs, WindowControls, Screens, NativeMenus, PersistentStorage, ReportExport}

// isNativeFeature reports whether a feature is known to this contract.
func isNativeFeature(parseFeature Feature) bool {
	for _, parseKnown := range nativeFeatures {
		if parseKnown == parseFeature {
			return true
		}
	}
	return false
}

// ClipboardWriteRequest carries explicit text to the native clipboard.
type ClipboardWriteRequest struct {
	Text string `json:"text"`
}

// MessageRequest describes a bounded native dialog.
type MessageRequest struct {
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

// MessageReply identifies the button selected by the operator.
type MessageReply struct {
	Button string `json:"button"`
}

// WindowRequest describes one caller-owned window operation.
type WindowRequest struct {
	Action string `json:"action"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// WindowInfo reports stable caller-window metadata.
type WindowInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Maximised  bool   `json:"maximised"`
	Fullscreen bool   `json:"fullscreen"`
}

// ScreenInfo reports a display without exposing backend-specific screen objects.
type ScreenInfo struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Primary bool    `json:"primary"`
	Scale   float32 `json:"scale"`
	X       int     `json:"x"`
	Y       int     `json:"y"`
	Width   int     `json:"width"`
	Height  int     `json:"height"`
}

// NativeBackend is implemented by a host adapter and must resolve caller windows from context.
type NativeBackend interface {
	Features() []Feature
	ClipboardWrite(context.Context, ClipboardWriteRequest) error
	ClipboardRead(context.Context) (string, error)
	ShowMessage(context.Context, MessageRequest) (MessageReply, error)
	Window(context.Context, WindowRequest) (WindowInfo, error)
	Screens(context.Context) ([]ScreenInfo, error)
}

// NativeHost enforces policy and backend intersection before native work.
type NativeHost struct {
	parseBackend NativeBackend
	parsePolicy  FeaturePolicy
}

// NewNativeHost constructs a host whose effective policy is policy intersected with backend features.
func NewNativeHost(parseBackend NativeBackend, parsePolicy FeaturePolicy) *NativeHost {
	return &NativeHost{parseBackend: parseBackend, parsePolicy: parsePolicy.Intersect(func() []Feature {
		if parseBackend == nil {
			return nil
		}
		return parseBackend.Features()
	}())}
}

// GetFeatures returns the effective immutable feature set.
func (parseHost *NativeHost) GetFeatures() []Feature {
	if parseHost == nil {
		return nil
	}
	return parseHost.parsePolicy.FeatureNames()
}

// GetMethods returns only methods enabled by effective policy.
func (parseHost *NativeHost) GetMethods() []string {
	if parseHost == nil || parseHost.parseBackend == nil {
		return nil
	}
	parseMethods := []string{}
	if parseHost.parsePolicy.Allows(Clipboard) {
		parseMethods = append(parseMethods, ClipboardWriteMethod, ClipboardReadMethod)
	}
	if parseHost.parsePolicy.Allows(MessageDialogs) {
		parseMethods = append(parseMethods, MessageMethod)
	}
	if parseHost.parsePolicy.Allows(WindowControls) {
		parseMethods = append(parseMethods, WindowMethod)
	}
	if parseHost.parsePolicy.Allows(Screens) {
		parseMethods = append(parseMethods, ScreensMethod)
	}
	return parseMethods
}

// Require checks policy and backend support for one feature.
func (parseHost *NativeHost) Require(parseFeature Feature) error {
	if parseHost == nil || parseHost.parseBackend == nil || !parseHost.parsePolicy.Allows(parseFeature) {
		return getError(string(parseFeature), interop.CodeUnavailable, errors.New("desktop feature disabled or unavailable"))
	}
	return nil
}

// Execute handles one versioned native request and always returns a wire-safe reply.
func (parseHost *NativeHost) Execute(parseContext context.Context, parseRequest NativeRequest) NativeReply {
	if parseRequest.Version != NativeContractVersion {
		return nativeFailure(interop.CodeInvalid, "unsupported native contract version")
	}
	if parseContext == nil {
		return nativeFailure(interop.CodeInvalid, "calling context required")
	}
	if !json.Valid(parseRequest.Args) {
		return nativeFailure(interop.CodeInvalid, "invalid native request arguments")
	}
	var parseResult any
	var parseErr error
	switch parseRequest.Method {
	case ClipboardWriteMethod:
		var parseArgs ClipboardWriteRequest
		parseErr = json.Unmarshal(parseRequest.Args, &parseArgs)
		if parseErr == nil {
			parseErr = parseHost.ClipboardWrite(parseContext, parseArgs)
		}
		parseResult = struct{}{}
	case ClipboardReadMethod:
		var parseValue string
		parseValue, parseErr = parseHost.ClipboardRead(parseContext)
		parseResult = parseValue
	case MessageMethod:
		var parseArgs MessageRequest
		parseErr = json.Unmarshal(parseRequest.Args, &parseArgs)
		var parseValue MessageReply
		if parseErr == nil {
			parseValue, parseErr = parseHost.ShowMessage(parseContext, parseArgs)
		}
		parseResult = parseValue
	case WindowMethod:
		var parseArgs WindowRequest
		parseErr = json.Unmarshal(parseRequest.Args, &parseArgs)
		var parseValue WindowInfo
		if parseErr == nil {
			parseValue, parseErr = parseHost.WindowControl(parseContext, parseArgs)
		}
		parseResult = parseValue
	case ScreensMethod:
		var parseValue []ScreenInfo
		parseValue, parseErr = parseHost.ListScreens(parseContext)
		parseResult = parseValue
	default:
		return nativeFailure(interop.CodeMissingExport, "native method not registered")
	}
	if parseErr != nil {
		return nativeErrorReply(parseErr)
	}
	parseData, parseMarshalErr := json.Marshal(parseResult)
	if parseMarshalErr != nil {
		return nativeFailure(interop.CodeEncode, parseMarshalErr.Error())
	}
	return NativeReply{Version: NativeContractVersion, Data: parseData}
}

// nativeFailure creates a typed failure envelope.
func nativeFailure(parseCode interop.ErrorCode, parseMessage string) NativeReply {
	return NativeReply{Version: NativeContractVersion, Code: parseCode, Message: parseMessage}
}

// nativeErrorReply preserves an interop classification in a wire envelope.
func nativeErrorReply(parseErr error) NativeReply {
	parseCode := interop.CodeRemote
	var parseJSONType *json.UnmarshalTypeError
	var parseJSONSyntax *json.SyntaxError
	if errors.As(parseErr, &parseJSONType) || errors.As(parseErr, &parseJSONSyntax) {
		parseCode = interop.CodeInvalid
	}
	var parseTyped *interop.Error
	if errors.As(parseErr, &parseTyped) {
		parseCode = parseTyped.Code
	}
	return nativeFailure(parseCode, parseErr.Error())
}

// ClipboardWrite performs an explicitly requested clipboard write.
func (parseHost *NativeHost) ClipboardWrite(parseContext context.Context, parseRequest ClipboardWriteRequest) error {
	if parseErr := parseHost.Require(Clipboard); parseErr != nil {
		return parseErr
	}
	if parseContext == nil {
		return getError(ClipboardWriteMethod, interop.CodeInvalid, errors.New("calling context required"))
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return getContextError(ClipboardWriteMethod, parseErr)
	}
	if parseErr := validateNativeText(parseRequest.Text, "clipboard text"); parseErr != nil {
		return getError(ClipboardWriteMethod, interop.CodeInvalid, parseErr)
	}
	parseErr := parseHost.parseBackend.ClipboardWrite(parseContext, parseRequest)
	if parseContext.Err() != nil {
		return getContextError(ClipboardWriteMethod, parseContext.Err())
	}
	return parseErr
}

// ClipboardRead performs an explicitly requested clipboard read.
func (parseHost *NativeHost) ClipboardRead(parseContext context.Context) (string, error) {
	if parseErr := parseHost.Require(Clipboard); parseErr != nil {
		return "", parseErr
	}
	if parseContext == nil {
		return "", getError(ClipboardReadMethod, interop.CodeInvalid, errors.New("calling context required"))
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return "", getContextError(ClipboardReadMethod, parseErr)
	}
	parseValue, parseErr := parseHost.parseBackend.ClipboardRead(parseContext)
	if parseContext.Err() != nil {
		return "", getContextError(ClipboardReadMethod, parseContext.Err())
	}
	if parseErr != nil {
		return "", parseErr
	}
	if parseErr = validateNativeText(parseValue, "clipboard text"); parseErr != nil {
		return "", getError(ClipboardReadMethod, interop.CodeDecode, parseErr)
	}
	return parseValue, nil
}

// ShowMessage displays one typed native dialog.
func (parseHost *NativeHost) ShowMessage(parseContext context.Context, parseRequest MessageRequest) (MessageReply, error) {
	if parseErr := parseHost.Require(MessageDialogs); parseErr != nil {
		return MessageReply{}, parseErr
	}
	if parseContext == nil {
		return MessageReply{}, getError(MessageMethod, interop.CodeInvalid, errors.New("calling context required"))
	}
	if parseRequest.Kind != "info" && parseRequest.Kind != "question" {
		return MessageReply{}, getError(MessageMethod, interop.CodeInvalid, errors.New("unknown message kind"))
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return MessageReply{}, getContextError(MessageMethod, parseErr)
	}
	if parseErr := validateNativeText(parseRequest.Title, "message title"); parseErr != nil {
		return MessageReply{}, getError(MessageMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := validateNativeText(parseRequest.Message, "message text"); parseErr != nil {
		return MessageReply{}, getError(MessageMethod, interop.CodeInvalid, parseErr)
	}
	parseReply, parseErr := parseHost.parseBackend.ShowMessage(parseContext, parseRequest)
	if parseContext.Err() != nil {
		return MessageReply{}, getContextError(MessageMethod, parseContext.Err())
	}
	if parseErr != nil {
		return MessageReply{}, parseErr
	}
	if parseReply.Button != "Ok" && parseReply.Button != "Yes" && parseReply.Button != "No" {
		return MessageReply{}, getError(MessageMethod, interop.CodeDecode, errors.New("invalid message button"))
	}
	if parseRequest.Kind == "info" && parseReply.Button != "Ok" {
		return MessageReply{}, getError(MessageMethod, interop.CodeDecode, errors.New("invalid information button"))
	}
	if parseRequest.Kind == "question" && parseReply.Button == "Ok" {
		return MessageReply{}, getError(MessageMethod, interop.CodeDecode, errors.New("invalid question button"))
	}
	return parseReply, nil
}

// WindowControl performs one typed caller-window operation.
func (parseHost *NativeHost) WindowControl(parseContext context.Context, parseRequest WindowRequest) (WindowInfo, error) {
	if parseErr := parseHost.Require(WindowControls); parseErr != nil {
		return WindowInfo{}, parseErr
	}
	if parseContext == nil {
		return WindowInfo{}, getError(WindowMethod, interop.CodeInvalid, errors.New("calling context required"))
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return WindowInfo{}, getContextError(WindowMethod, parseErr)
	}
	if parseErr := validateWindowRequest(parseRequest); parseErr != nil {
		return WindowInfo{}, getError(WindowMethod, interop.CodeInvalid, parseErr)
	}
	parseInfo, parseErr := parseHost.parseBackend.Window(parseContext, parseRequest)
	if parseContext.Err() != nil {
		return WindowInfo{}, getContextError(WindowMethod, parseContext.Err())
	}
	if parseErr != nil {
		return WindowInfo{}, parseErr
	}
	if parseErr = validateWindowInfo(parseInfo); parseErr != nil {
		return WindowInfo{}, getError(WindowMethod, interop.CodeDecode, parseErr)
	}
	return parseInfo, nil
}

// ListScreens returns typed display information.
func (parseHost *NativeHost) ListScreens(parseContext context.Context) ([]ScreenInfo, error) {
	if parseErr := parseHost.Require(Screens); parseErr != nil {
		return nil, parseErr
	}
	if parseContext == nil {
		return nil, getError(ScreensMethod, interop.CodeInvalid, errors.New("calling context required"))
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return nil, getContextError(ScreensMethod, parseErr)
	}
	parseResult, parseErr := parseHost.parseBackend.Screens(parseContext)
	if parseContext.Err() != nil {
		return nil, getContextError(ScreensMethod, parseContext.Err())
	}
	if parseErr != nil {
		return nil, parseErr
	}
	if parseErr = validateScreens(parseResult); parseErr != nil {
		return nil, getError(ScreensMethod, interop.CodeDecode, parseErr)
	}
	return parseResult, nil
}

// WriteClipboard invokes the typed clipboard client method.
func (parseClient Client) WriteClipboard(parseContext context.Context, parseText string) error {
	if parseErr := validateNativeText(parseText, "clipboard text"); parseErr != nil {
		return getError(ClipboardWriteMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := parseClient.Require(Clipboard); parseErr != nil {
		return parseErr
	}
	_, parseErr := parseClient.getNativeReply(parseContext, ClipboardWriteMethod, ClipboardWriteRequest{Text: parseText})
	return parseErr
}

// ReadClipboard invokes the typed clipboard client method.
func (parseClient Client) ReadClipboard(parseContext context.Context) (string, error) {
	if parseErr := parseClient.Require(Clipboard); parseErr != nil {
		return "", parseErr
	}
	parseReply, parseErr := parseClient.getNativeReply(parseContext, ClipboardReadMethod)
	if parseErr != nil {
		return "", parseErr
	}
	var parseValue string
	if parseErr = json.Unmarshal(parseReply.Data, &parseValue); parseErr != nil {
		return "", getError(ClipboardReadMethod, interop.CodeDecode, parseErr)
	}
	if parseErr = validateNativeText(parseValue, "clipboard text"); parseErr != nil {
		return "", getError(ClipboardReadMethod, interop.CodeDecode, parseErr)
	}
	return parseValue, nil
}

// ShowMessage invokes the typed message client method.
func (parseClient Client) ShowMessage(parseContext context.Context, parseRequest MessageRequest) (MessageReply, error) {
	if parseRequest.Kind != "info" && parseRequest.Kind != "question" {
		return MessageReply{}, getError(MessageMethod, interop.CodeInvalid, errors.New("unknown message kind"))
	}
	if parseErr := validateNativeText(parseRequest.Title, "message title"); parseErr != nil {
		return MessageReply{}, getError(MessageMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := validateNativeText(parseRequest.Message, "message text"); parseErr != nil {
		return MessageReply{}, getError(MessageMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := parseClient.Require(MessageDialogs); parseErr != nil {
		return MessageReply{}, parseErr
	}
	parseReply, parseErr := parseClient.getNativeReplyTimeout(parseContext, 5*time.Minute, MessageMethod, parseRequest)
	if parseErr != nil {
		return MessageReply{}, parseErr
	}
	var parseValue MessageReply
	if parseErr = json.Unmarshal(parseReply.Data, &parseValue); parseErr != nil {
		return MessageReply{}, getError(MessageMethod, interop.CodeDecode, parseErr)
	}
	if parseValue.Button != "Ok" && parseValue.Button != "Yes" && parseValue.Button != "No" {
		return MessageReply{}, getError(MessageMethod, interop.CodeDecode, errors.New("invalid message button"))
	}
	if parseRequest.Kind == "info" && parseValue.Button != "Ok" {
		return MessageReply{}, getError(MessageMethod, interop.CodeDecode, errors.New("invalid information button"))
	}
	if parseRequest.Kind == "question" && parseValue.Button == "Ok" {
		return MessageReply{}, getError(MessageMethod, interop.CodeDecode, errors.New("invalid question button"))
	}
	return parseValue, nil
}

// ControlWindow invokes the typed window client method.
func (parseClient Client) ControlWindow(parseContext context.Context, parseRequest WindowRequest) (WindowInfo, error) {
	if parseErr := validateWindowRequest(parseRequest); parseErr != nil {
		return WindowInfo{}, getError(WindowMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := parseClient.Require(WindowControls); parseErr != nil {
		return WindowInfo{}, parseErr
	}
	parseReply, parseErr := parseClient.getNativeReply(parseContext, WindowMethod, parseRequest)
	if parseErr != nil {
		return WindowInfo{}, parseErr
	}
	var parseValue WindowInfo
	if parseErr = json.Unmarshal(parseReply.Data, &parseValue); parseErr != nil {
		return WindowInfo{}, getError(WindowMethod, interop.CodeDecode, parseErr)
	}
	if parseErr = validateWindowInfo(parseValue); parseErr != nil {
		return WindowInfo{}, getError(WindowMethod, interop.CodeDecode, parseErr)
	}
	return parseValue, nil
}

// ListScreens invokes the typed screen client method.
func (parseClient Client) ListScreens(parseContext context.Context) ([]ScreenInfo, error) {
	if parseErr := parseClient.Require(Screens); parseErr != nil {
		return nil, parseErr
	}
	parseReply, parseErr := parseClient.getNativeReply(parseContext, ScreensMethod)
	if parseErr != nil {
		return nil, parseErr
	}
	var parseValue []ScreenInfo
	if parseErr = json.Unmarshal(parseReply.Data, &parseValue); parseErr != nil {
		return nil, getError(ScreensMethod, interop.CodeDecode, parseErr)
	}
	if parseErr = validateScreens(parseValue); parseErr != nil {
		return nil, getError(ScreensMethod, interop.CodeDecode, parseErr)
	}
	return parseValue, nil
}

// getNativeReply sends one versioned request and preserves the host envelope.
func (parseClient Client) getNativeReply(parseContext context.Context, parseMethod string, parseArgs ...any) (NativeReply, error) {
	return parseClient.getNativeReplyTimeout(parseContext, RequestTimeout, parseMethod, parseArgs...)
}

// getNativeReplyTimeout sends one versioned request with an operation-specific ceiling.
func (parseClient Client) getNativeReplyTimeout(parseContext context.Context, parseTimeout time.Duration, parseMethod string, parseArgs ...any) (NativeReply, error) {
	var parseValue any = map[string]any{}
	if len(parseArgs) == 1 {
		parseValue = parseArgs[0]
	}
	parseData, parseErr := json.Marshal(parseValue)
	if parseErr != nil {
		return NativeReply{}, getError(parseMethod, interop.CodeEncode, parseErr)
	}
	parseReply, parseErr := CallWithTimeout[NativeReply](parseContext, parseClient, parseTimeout, parseMethod, NativeRequest{Version: NativeContractVersion, Method: parseMethod, Args: parseData})
	if parseErr != nil {
		return NativeReply{}, parseErr
	}
	if parseReply.Version != NativeContractVersion {
		return NativeReply{}, getError(parseMethod, interop.CodeDecode, errors.New("unsupported native reply version"))
	}
	if parseReply.Code != "" {
		return NativeReply{}, getError(parseMethod, parseReply.Code, errors.New(parseReply.Message))
	}
	return parseReply, nil
}

// validateNativeText bounds native text and rejects malformed strings.
func validateNativeText(parseValue, parseLabel string) error {
	if len(parseValue) > 4096 || !utf8.ValidString(parseValue) || strings.ContainsRune(parseValue, 0) {
		return fmt.Errorf("invalid or oversized %s", parseLabel)
	}
	return nil
}

// validateWindowRequest bounds the explicit window operation allowlist.
func validateWindowRequest(parseRequest WindowRequest) error {
	switch parseRequest.Action {
	case "info", "maximize", "restore", "fullscreen", "unfullscreen":
		if parseRequest.Width != 0 || parseRequest.Height != 0 {
			return errors.New("dimensions apply only to resize")
		}
	case "resize":
		if parseRequest.Width < 1 || parseRequest.Width > 8192 || parseRequest.Height < 1 || parseRequest.Height > 8192 {
			return errors.New("window dimensions out of bounds")
		}
	default:
		return errors.New("unknown window action")
	}
	return nil
}

// validateWindowInfo rejects malformed backend window metadata.
func validateWindowInfo(parseInfo WindowInfo) error {
	if parseInfo.ID == "" || len(parseInfo.ID) > 256 || !utf8.ValidString(parseInfo.ID) || strings.ContainsRune(parseInfo.ID, 0) || len(parseInfo.Name) > 4096 || !utf8.ValidString(parseInfo.Name) || strings.ContainsRune(parseInfo.Name, 0) || parseInfo.Width < 0 || parseInfo.Width > 32768 || parseInfo.Height < 0 || parseInfo.Height > 32768 {
		return errors.New("invalid window information")
	}
	return nil
}

// validateScreens rejects malformed backend display metadata.
func validateScreens(parseScreens []ScreenInfo) error {
	if len(parseScreens) > 64 {
		return errors.New("too many screens")
	}
	for _, parseScreen := range parseScreens {
		if parseScreen.ID == "" || len(parseScreen.ID) > 256 || !utf8.ValidString(parseScreen.ID) || strings.ContainsRune(parseScreen.ID, 0) || len(parseScreen.Name) > 4096 || !utf8.ValidString(parseScreen.Name) || strings.ContainsRune(parseScreen.Name, 0) || parseScreen.Width < 0 || parseScreen.Width > 100000 || parseScreen.Height < 0 || parseScreen.Height > 100000 || math.IsNaN(float64(parseScreen.Scale)) || math.IsInf(float64(parseScreen.Scale), 0) || parseScreen.Scale < 0 || parseScreen.Scale > 100 {
			return errors.New("invalid screen information")
		}
	}
	return nil
}
