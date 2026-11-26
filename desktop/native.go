package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/monstercameron/GoWebComponents/v8/interop"
)

// Clipboard identifies explicit text clipboard operations.
const Clipboard Feature = "clipboard"

// MessageDialogs identifies native information and question dialogs.
const MessageDialogs Feature = "message-dialogs"

// WindowControls identifies caller-owned window inspection and controls.
const WindowControls Feature = "window-controls"

// WindowPrinting identifies explicit caller-window print requests.
const WindowPrinting Feature = "window-printing"

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
	WindowPrintMethod     = "desktop.window.print"
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

var nativeFeatures = []Feature{FileDialogs, Clipboard, MessageDialogs, WindowControls, WindowEvents, ChildWindows, WindowPrinting, Screens, ScreenGeometry, NativeMenus, RuntimeMenus, SystemTray, GlobalShortcuts, PersistentStorage, ReportExport, SystemEnvironment, ExternalURLs, Autostart, FileManager}

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
	Kind          string   `json:"kind"`
	Title         string   `json:"title"`
	Message       string   `json:"message"`
	Buttons       []string `json:"buttons,omitempty"`
	DefaultButton string   `json:"defaultButton,omitempty"`
	CancelButton  string   `json:"cancelButton,omitempty"`
}

// MessageReply identifies the button selected by the operator.
type MessageReply struct {
	Button string `json:"button"`
}

// validateMessageRequest bounds portable message-dialog controls to native Windows capabilities.
func validateMessageRequest(request MessageRequest) error {
	if len(request.Buttons) > 3 {
		return errors.New("too many message buttons")
	}
	seenButtons := make(map[string]bool, len(request.Buttons))
	for _, button := range request.Buttons {
		if strings.TrimSpace(button) == "" || len(button) > 64 || !utf8.ValidString(button) || strings.ContainsRune(button, 0) {
			return errors.New("invalid message button text")
		}
		if seenButtons[button] {
			return errors.New("duplicate message button label")
		}
		seenButtons[button] = true
	}
	buttons := request.Buttons
	switch request.Kind {
	case "info", "warning", "error":
		if len(buttons) == 0 {
			buttons = []string{"Ok"}
		}
		if len(buttons) != 1 || buttons[0] != "Ok" {
			return errors.New("information, warning, and error dialogs support only the Ok button")
		}
	case "question":
		if len(buttons) == 0 {
			buttons = []string{"Yes", "No"}
		}
	default:
		return errors.New("unknown message kind")
	}
	if request.DefaultButton != "" && !messageHasButton(buttons, request.DefaultButton) {
		return errors.New("default message button is not present")
	}
	if request.CancelButton != "" && !messageHasButton(buttons, request.CancelButton) {
		return errors.New("cancel message button is not present")
	}
	return nil
}

// messageHasButton reports whether a bounded dialog button is present.
func messageHasButton(parseButtons []string, parseButton string) bool {
	for _, parseValue := range parseButtons {
		if parseValue == parseButton {
			return true
		}
	}
	return false
}

// validateMessageReply checks that a native result belongs to its requested button family.
func validateMessageReply(parseRequest MessageRequest, parseReply MessageReply) error {
	parseButtons := parseRequest.Buttons
	if len(parseButtons) == 0 {
		if parseRequest.Kind == "question" {
			parseButtons = []string{"Yes", "No"}
		} else {
			parseButtons = []string{"Ok"}
		}
	}
	if !messageHasButton(parseButtons, parseReply.Button) {
		return errors.New("invalid message button")
	}
	return nil
}

// WindowRequest describes one caller-owned window operation.
type WindowRequest struct {
	Action   string  `json:"action"`
	Title    string  `json:"title,omitempty"`
	ScreenID string  `json:"screenId,omitempty"`
	X        int     `json:"x,omitempty"`
	Y        int     `json:"y,omitempty"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
	Enabled  bool    `json:"enabled,omitempty"`
	Zoom     float64 `json:"zoom,omitempty"`
	Red      int     `json:"red,omitempty"`
	Green    int     `json:"green,omitempty"`
	Blue     int     `json:"blue,omitempty"`
	Alpha    int     `json:"alpha,omitempty"`
	State    string  `json:"state,omitempty"`
}

// WindowBorderSizes reports non-client border thicknesses in pixels.
type WindowBorderSizes struct {
	Top    int `json:"top"`
	Left   int `json:"left"`
	Bottom int `json:"bottom"`
	Right  int `json:"right"`
}

// WindowInfo reports stable caller-window metadata.
type WindowInfo struct {
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	X                 int                `json:"x"`
	Y                 int                `json:"y"`
	RelativeX         int                `json:"relativeX"`
	RelativeY         int                `json:"relativeY"`
	Width             int                `json:"width"`
	Height            int                `json:"height"`
	Focused           bool               `json:"focused"`
	Minimised         bool               `json:"minimised"`
	Maximised         bool               `json:"maximised"`
	Fullscreen        bool               `json:"fullscreen"`
	Visible           bool               `json:"visible"`
	Resizable         bool               `json:"resizable"`
	Zoom              float64            `json:"zoom"`
	IgnoreMouseEvents bool               `json:"ignoreMouseEvents,omitempty"`
	ScreenID          string             `json:"screenId,omitempty"`
	BorderSizes       *WindowBorderSizes `json:"borderSizes,omitempty"`
}

// ScreenInfo reports a display without exposing backend-specific screen objects.
type ScreenInfo struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Primary          bool         `json:"primary"`
	Scale            float32      `json:"scale"`
	X                int          `json:"x"`
	Y                int          `json:"y"`
	Width            int          `json:"width"`
	Height           int          `json:"height"`
	WorkArea         ScreenBounds `json:"workArea"`
	PhysicalBounds   ScreenBounds `json:"physicalBounds"`
	PhysicalWorkArea ScreenBounds `json:"physicalWorkArea"`
	Rotation         float32      `json:"rotation"`
}

// ScreenBounds describes monitor geometry in the units named by its enclosing field.
type ScreenBounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
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

// WindowPrintBackend is implemented only by hosts supporting explicit native printing.
type WindowPrintBackend interface {
	PrintWindow(context.Context) error
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
	if _, parseOK := parseHost.parseBackend.(ChildWindowBackend); parseOK && parseHost.parsePolicy.Allows(ChildWindows) {
		parseMethods = append(parseMethods, ChildWindowCreateMethod, ChildWindowListMethod, ChildWindowInspectMethod, ChildWindowControlMethod)
	}
	if _, parseOK := parseHost.parseBackend.(WindowPrintBackend); parseOK && parseHost.parsePolicy.Allows(WindowPrinting) {
		parseMethods = append(parseMethods, WindowPrintMethod)
	}
	if parseHost.parsePolicy.Allows(Screens) {
		parseMethods = append(parseMethods, ScreensMethod)
	}
	if _, parseOK := parseHost.parseBackend.(ScreenGeometryBackend); parseOK && parseHost.parsePolicy.Allows(ScreenGeometry) {
		parseMethods = append(parseMethods, ScreenGeometryMethod)
	}
	if _, parseOK := parseHost.parseBackend.(RuntimeMenuBackend); parseOK && parseHost.parsePolicy.Allows(RuntimeMenus) {
		parseMethods = append(parseMethods, MenuReplaceMethod)
	}
	if _, parseOK := parseHost.parseBackend.(ContextMenuBackend); parseOK && parseHost.parsePolicy.Allows(RuntimeMenus) {
		parseMethods = append(parseMethods, ContextMenuInstallMethod, ContextMenuShowMethod, ContextMenuRemoveMethod)
	}
	if _, parseOK := parseHost.parseBackend.(TrayBackend); parseOK && parseHost.parsePolicy.Allows(SystemTray) {
		parseMethods = append(parseMethods, TrayConfigureMethod)
	}
	if _, parseOK := parseHost.parseBackend.(ShortcutBackend); parseOK && parseHost.parsePolicy.Allows(GlobalShortcuts) {
		parseMethods = append(parseMethods, ShortcutConfigureMethod)
	}
	if _, parseOK := parseHost.parseBackend.(SystemEnvironmentBackend); parseOK && parseHost.parsePolicy.Allows(SystemEnvironment) {
		parseMethods = append(parseMethods, SystemEnvironmentMethod)
	}
	if _, parseOK := parseHost.parseBackend.(ExternalURLBackend); parseOK && parseHost.parsePolicy.Allows(ExternalURLs) {
		parseMethods = append(parseMethods, ExternalURLOpenMethod)
	}
	if _, parseOK := parseHost.parseBackend.(AutostartBackend); parseOK && parseHost.parsePolicy.Allows(Autostart) {
		parseMethods = append(parseMethods, AutostartStatusMethod, AutostartEnableMethod, AutostartDisableMethod)
	}
	if _, parseOK := parseHost.parseBackend.(FileManagerBackend); parseOK && parseHost.parsePolicy.Allows(FileManager) {
		parseMethods = append(parseMethods, FileManagerRevealMethod)
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
func (parseHost *NativeHost) Execute(parseContext context.Context, parseRequest NativeRequest) (parseReply NativeReply) {
	// A faulty host adapter must not unwind through the transport: contain its
	// panic as a typed failure. The panic value stays off the wire because it can
	// name internal paths and state; the diagnostic belongs in the host's logs.
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			log.Printf("desktop: native method %q panicked: %v", parseRequest.Method, parseRecovered)
			parseReply = nativeFailure(interop.CodeRemote, "native method failed unexpectedly")
		}
	}()
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
	case ChildWindowCreateMethod:
		var parseArgs ChildWindowCreateRequest
		parseErr = decodeChildWindowRequest(parseRequest.Args, &parseArgs)
		var parseValue ChildWindowInfo
		if parseErr == nil {
			parseValue, parseErr = parseHost.CreateChildWindow(parseContext, parseArgs)
		}
		parseResult = parseValue
	case ChildWindowListMethod:
		var parseValue []ChildWindowInfo
		parseValue, parseErr = parseHost.ListChildWindows(parseContext)
		parseResult = parseValue
	case ChildWindowInspectMethod:
		var parseArgs ChildWindowRequest
		parseErr = decodeChildWindowRequest(parseRequest.Args, &parseArgs)
		var parseValue ChildWindowInfo
		if parseErr == nil {
			parseValue, parseErr = parseHost.InspectChildWindow(parseContext, parseArgs)
		}
		parseResult = parseValue
	case ChildWindowControlMethod:
		var parseArgs ChildWindowControlRequest
		parseErr = decodeChildWindowRequest(parseRequest.Args, &parseArgs)
		var parseValue ChildWindowInfo
		if parseErr == nil {
			parseValue, parseErr = parseHost.ControlChildWindow(parseContext, parseArgs)
		}
		parseResult = parseValue
	case WindowPrintMethod:
		parseErr = parseHost.PrintWindow(parseContext)
		parseResult = struct{}{}
	case ScreensMethod:
		var parseValue []ScreenInfo
		parseValue, parseErr = parseHost.ListScreens(parseContext)
		parseResult = parseValue
	case ScreenGeometryMethod:
		var parseArgs ScreenGeometryRequest
		parseErr = json.Unmarshal(parseRequest.Args, &parseArgs)
		var parseValue ScreenGeometryReply
		if parseErr == nil {
			parseValue, parseErr = parseHost.ScreenGeometry(parseContext, parseArgs)
		}
		parseResult = parseValue
	case TrayConfigureMethod:
		var parseArgs TrayRequest
		parseErr = json.Unmarshal(parseRequest.Args, &parseArgs)
		if parseErr == nil {
			parseErr = parseHost.ConfigureTray(parseContext, parseArgs)
		}
		parseResult = struct{}{}
	case ShortcutConfigureMethod:
		var parseArgs ShortcutRequest
		parseErr = json.Unmarshal(parseRequest.Args, &parseArgs)
		if parseErr == nil {
			parseErr = parseHost.ConfigureShortcut(parseContext, parseArgs)
		}
		parseResult = struct{}{}
	case SystemEnvironmentMethod:
		var parseValue SystemEnvironmentInfo
		parseValue, parseErr = parseHost.InspectSystemEnvironment(parseContext)
		parseResult = parseValue
	case ExternalURLOpenMethod:
		var parseArgs ExternalURLOpenRequest
		parseErr = json.Unmarshal(parseRequest.Args, &parseArgs)
		if parseErr == nil {
			parseErr = parseHost.OpenExternalURL(parseContext, parseArgs)
		}
		parseResult = struct{}{}
	case AutostartStatusMethod:
		var parseValue AutostartStatus
		parseValue, parseErr = parseHost.GetAutostartStatus(parseContext)
		parseResult = parseValue
	case AutostartEnableMethod:
		parseErr = parseHost.EnableAutostart(parseContext)
		parseResult = struct{}{}
	case AutostartDisableMethod:
		parseErr = parseHost.DisableAutostart(parseContext)
		parseResult = struct{}{}
	case FileManagerRevealMethod:
		var parseArgs FileManagerRevealRequest
		parseErr = json.Unmarshal(parseRequest.Args, &parseArgs)
		if parseErr == nil {
			parseErr = parseHost.RevealPath(parseContext, parseArgs)
		}
		parseResult = struct{}{}
	case MenuReplaceMethod:
		var parseArgs Menu
		parseErr = json.Unmarshal(parseRequest.Args, &parseArgs)
		if parseErr == nil {
			parseErr = parseHost.ReplaceMenu(parseContext, parseArgs)
		}
		parseResult = struct{}{}
	case ContextMenuInstallMethod:
		var parseArgs ContextMenuRequest
		parseErr = json.Unmarshal(parseRequest.Args, &parseArgs)
		if parseErr == nil {
			parseErr = parseHost.InstallContextMenu(parseContext, parseArgs)
		}
		parseResult = struct{}{}
	case ContextMenuShowMethod:
		var parseArgs ContextMenuShowRequest
		parseErr = json.Unmarshal(parseRequest.Args, &parseArgs)
		if parseErr == nil {
			parseErr = parseHost.ShowContextMenu(parseContext, parseArgs)
		}
		parseResult = struct{}{}
	case ContextMenuRemoveMethod:
		var parseArgs struct {
			ID string `json:"id"`
		}
		parseErr = json.Unmarshal(parseRequest.Args, &parseArgs)
		if parseErr == nil {
			parseErr = parseHost.RemoveContextMenu(parseContext, parseArgs.ID)
		}
		parseResult = struct{}{}
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
	_, parseJSONType := errors.AsType[*json.UnmarshalTypeError](parseErr)
	_, parseJSONSyntax := errors.AsType[*json.SyntaxError](parseErr)
	if parseJSONType || parseJSONSyntax {
		parseCode = interop.CodeInvalid
	}
	if parseTyped, parseOk := errors.AsType[*interop.Error](parseErr); parseOk {
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
	if parseErr := validateMessageRequest(parseRequest); parseErr != nil {
		return MessageReply{}, getError(MessageMethod, interop.CodeInvalid, parseErr)
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
	if parseErr = validateMessageReply(parseRequest, parseReply); parseErr != nil {
		return MessageReply{}, getError(MessageMethod, interop.CodeDecode, parseErr)
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

// PrintWindow opens the native print workflow for the explicit caller window.
func (parseHost *NativeHost) PrintWindow(parseContext context.Context) error {
	if parseErr := parseHost.Require(WindowPrinting); parseErr != nil {
		return parseErr
	}
	if parseContext == nil {
		return getError(WindowPrintMethod, interop.CodeInvalid, errors.New("calling context required"))
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return getContextError(WindowPrintMethod, parseErr)
	}
	parseBackend, parseOK := parseHost.parseBackend.(WindowPrintBackend)
	if !parseOK {
		return getError(WindowPrintMethod, interop.CodeUnavailable, errors.New("window printing unavailable"))
	}
	parseErr := parseBackend.PrintWindow(parseContext)
	if parseContext.Err() != nil {
		return getContextError(WindowPrintMethod, parseContext.Err())
	}
	return parseErr
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
	if parseErr := validateMessageRequest(parseRequest); parseErr != nil {
		return MessageReply{}, getError(MessageMethod, interop.CodeInvalid, parseErr)
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
	if parseErr = validateMessageReply(parseRequest, parseValue); parseErr != nil {
		return MessageReply{}, getError(MessageMethod, interop.CodeDecode, parseErr)
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

// PrintWindow invokes the separately gated native caller-window print workflow.
func (parseClient Client) PrintWindow(parseContext context.Context) error {
	if parseErr := parseClient.Require(WindowPrinting); parseErr != nil {
		return parseErr
	}
	_, parseErr := parseClient.getNativeReplyTimeout(parseContext, 5*time.Minute, WindowPrintMethod)
	return parseErr
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
func validateWindowRequest(request WindowRequest) error {
	if (request.Zoom != 0 || request.Red != 0 || request.Green != 0 || request.Blue != 0 || request.Alpha != 0 || request.State != "") && request.Action != "set-zoom" && request.Action != "set-background-color" && !isWindowButtonStateAction(request.Action) {
		return errors.New("window style arguments do not apply to action")
	}
	switch request.Action {
	case "info", "show", "hide", "close", "reload", "force-reload", "snap-assist", "center", "enable-size-constraints", "disable-size-constraints", "minimize", "unminimize", "maximize", "unmaximize", "toggle-maximize", "restore", "fullscreen", "unfullscreen", "toggle-fullscreen", "focus", "show-menu-bar", "hide-menu-bar", "toggle-menu-bar", "toggle-frameless", "zoom-in", "zoom-out", "zoom-reset":
		if request.Title != "" || request.ScreenID != "" || request.X != 0 || request.Y != 0 || request.Width != 0 || request.Height != 0 || request.Enabled {
			return errors.New("window action does not accept arguments")
		}
	case "set-title":
		if err := validateNativeText(request.Title, "window title"); err != nil {
			return err
		}
		if request.ScreenID != "" || request.X != 0 || request.Y != 0 || request.Width != 0 || request.Height != 0 || request.Enabled {
			return errors.New("set-title accepts only title")
		}
	case "set-screen":
		if request.ScreenID == "" {
			return errors.New("screen id required")
		}
		if err := validateNativeText(request.ScreenID, "screen id"); err != nil {
			return err
		}
		if request.Title != "" || request.X != 0 || request.Y != 0 || request.Width != 0 || request.Height != 0 || request.Enabled {
			return errors.New("set-screen accepts only screen id")
		}
	case "set-position", "set-relative-position":
		if request.X < -100000 || request.X > 100000 || request.Y < -100000 || request.Y > 100000 || request.Title != "" || request.ScreenID != "" || request.Width != 0 || request.Height != 0 || request.Enabled {
			return errors.New("window position out of bounds")
		}
	case "resize", "set-min-size", "set-max-size", "set-bounds":
		if request.Width < 1 || request.Width > 8192 || request.Height < 1 || request.Height > 8192 {
			return errors.New("window dimensions out of bounds")
		}
		if request.X < -100000 || request.X > 100000 || request.Y < -100000 || request.Y > 100000 || request.Title != "" || request.ScreenID != "" || (request.Action != "set-bounds" && (request.X != 0 || request.Y != 0)) || request.Enabled {
			return errors.New("size action accepts only dimensions")
		}
	case "set-always-on-top", "set-resizable", "set-frameless", "flash", "set-content-protection", "set-ignore-mouse-events", "set-enabled":
		if request.Title != "" || request.ScreenID != "" || request.X != 0 || request.Y != 0 || request.Width != 0 || request.Height != 0 {
			return errors.New("boolean window action accepts only enabled")
		}
	case "set-background-color":
		if request.Red < 0 || request.Red > 255 || request.Green < 0 || request.Green > 255 || request.Blue < 0 || request.Blue > 255 || request.Alpha < 0 || request.Alpha > 255 || request.Zoom != 0 || request.State != "" || hasWindowBaseArguments(request) {
			return errors.New("window color out of bounds")
		}
	case "set-zoom":
		if request.Zoom < 0.25 || request.Zoom > 5 || math.IsNaN(request.Zoom) || math.IsInf(request.Zoom, 0) || request.Red != 0 || request.Green != 0 || request.Blue != 0 || request.Alpha != 0 || request.State != "" || hasWindowBaseArguments(request) {
			return errors.New("window zoom out of bounds")
		}
	case "set-minimize-button-state", "set-maximize-button-state", "set-close-button-state", "set-fullscreen-button-state":
		if (request.State != "enabled" && request.State != "disabled" && request.State != "hidden") || request.Zoom != 0 || request.Red != 0 || request.Green != 0 || request.Blue != 0 || request.Alpha != 0 || hasWindowBaseArguments(request) {
			return errors.New("invalid window button state")
		}
	default:
		return errors.New("unknown window action")
	}
	return nil
}

// isWindowButtonStateAction reports whether an action consumes the state field.
func isWindowButtonStateAction(action string) bool {
	return action == "set-minimize-button-state" || action == "set-maximize-button-state" || action == "set-close-button-state" || action == "set-fullscreen-button-state"
}

// hasWindowBaseArguments reports arguments not used by color, zoom, or button-state actions.
func hasWindowBaseArguments(request WindowRequest) bool {
	return request.Title != "" || request.ScreenID != "" || request.X != 0 || request.Y != 0 || request.Width != 0 || request.Height != 0 || request.Enabled
}

// validateWindowInfo rejects malformed backend window metadata.
func validateWindowInfo(info WindowInfo) error {
	if info.ID == "" || len(info.ID) > 256 || !utf8.ValidString(info.ID) || strings.ContainsRune(info.ID, 0) || len(info.Name) > 4096 || !utf8.ValidString(info.Name) || strings.ContainsRune(info.Name, 0) || len(info.ScreenID) > 256 || !utf8.ValidString(info.ScreenID) || strings.ContainsRune(info.ScreenID, 0) || info.X < -100000 || info.X > 100000 || info.Y < -100000 || info.Y > 100000 || info.RelativeX < -100000 || info.RelativeX > 100000 || info.RelativeY < -100000 || info.RelativeY > 100000 || info.Width < 0 || info.Width > 32768 || info.Height < 0 || info.Height > 32768 || math.IsNaN(info.Zoom) || math.IsInf(info.Zoom, 0) || info.Zoom < 0 || info.Zoom > 100 {
		return errors.New("invalid window information")
	}
	if info.BorderSizes != nil {
		b := info.BorderSizes
		if b.Top < 0 || b.Top > 4096 || b.Left < 0 || b.Left > 4096 || b.Bottom < 0 || b.Bottom > 4096 || b.Right < 0 || b.Right > 4096 {
			return errors.New("invalid window border sizes")
		}
	}
	return nil
}

// validateScreens rejects malformed backend display metadata.
func validateScreens(parseScreens []ScreenInfo) error {
	if len(parseScreens) > 64 {
		return errors.New("too many screens")
	}
	for _, parseScreen := range parseScreens {
		for _, parseBounds := range []ScreenBounds{parseScreen.WorkArea, parseScreen.PhysicalBounds, parseScreen.PhysicalWorkArea} {
			if parseBounds.X < -100000 || parseBounds.X > 100000 || parseBounds.Y < -100000 || parseBounds.Y > 100000 || parseBounds.Width < 0 || parseBounds.Width > 100000 || parseBounds.Height < 0 || parseBounds.Height > 100000 {
				return errors.New("invalid screen bounds")
			}
		}
		if math.IsNaN(float64(parseScreen.Rotation)) || math.IsInf(float64(parseScreen.Rotation), 0) || parseScreen.Rotation < 0 || parseScreen.Rotation >= 360 {
			return errors.New("invalid screen rotation")
		}
		if parseScreen.ID == "" || len(parseScreen.ID) > 256 || !utf8.ValidString(parseScreen.ID) || strings.ContainsRune(parseScreen.ID, 0) || len(parseScreen.Name) > 4096 || !utf8.ValidString(parseScreen.Name) || strings.ContainsRune(parseScreen.Name, 0) || parseScreen.Width < 0 || parseScreen.Width > 100000 || parseScreen.Height < 0 || parseScreen.Height > 100000 || math.IsNaN(float64(parseScreen.Scale)) || math.IsInf(float64(parseScreen.Scale), 0) || parseScreen.Scale < 0 || parseScreen.Scale > 100 {
			return errors.New("invalid screen information")
		}
	}
	return nil
}
