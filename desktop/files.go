package desktop

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/monstercameron/GoWebComponents/v8/interop"
)

// Feature identifies a portable desktop workflow, not a UI rollout flag.
type Feature string

// FileDialogs enables path selection, never reading or writing selected files.
const FileDialogs Feature = "file-dialogs"

// FileDialogMethod is the adapter registration name; application callers use typed methods.
const FileDialogMethod = "desktop.files.select"

// FileFilter describes an OS file-picker filter without backend-specific types.
type FileFilter struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
}

// FileDialogOptions configures a caller-owned native picker.
type FileDialogOptions struct {
	Title           string       `json:"title"`
	Directory       string       `json:"directory"`
	Filename        string       `json:"filename"` // SaveFile only.
	ButtonText      string       `json:"buttonText,omitempty"`
	ShowHiddenFiles bool         `json:"showHiddenFiles,omitempty"`
	Filters         []FileFilter `json:"filters"` // File pickers only, not OpenDirectory.
}

// FileSelection distinguishes operator cancellation from successful path selection.
// Selecting a save path does not create or overwrite a file.
type FileSelection struct {
	Paths     []string `json:"paths"`
	Cancelled bool     `json:"cancelled"`
}

// FileDialogRequest is the versioned backend contract used by registered host services.
type FileDialogRequest struct {
	Version int               `json:"version"`
	Kind    string            `json:"kind"`
	Options FileDialogOptions `json:"options"`
}

// FileDialogReply preserves portable error categories across generated bindings.
type FileDialogReply struct {
	Selection FileSelection     `json:"selection"`
	Code      interop.ErrorCode `json:"code,omitempty"`
	Message   string            `json:"message,omitempty"`
}

// FileDialogBackend is implemented by native adapters or application test doubles.
// Implementations must resolve the invoking window from context, never current focus.
type FileDialogBackend interface {
	SelectPaths(context.Context, FileDialogRequest) (FileSelection, error)
}

// FileDialogHost enforces immutable opt-in before invoking a native backend.
// Its zero value denies access. This policy covers this service, not unrelated host code.
type FileDialogHost struct {
	parseBackend FileDialogBackend
	isEnabled    bool
}

// NewFileDialogHost constructs a host service with explicit file-dialog permission.
func NewFileDialogHost(parseBackend FileDialogBackend, isEnabled bool) *FileDialogHost {
	return &FileDialogHost{parseBackend: parseBackend, isEnabled: isEnabled}
}

// GetMethods advertises only operations enabled in this host configuration.
func (parseHost *FileDialogHost) GetMethods() []string {
	if parseHost == nil || !parseHost.isEnabled || parseHost.parseBackend == nil {
		return nil
	}
	return []string{FileDialogMethod}
}

// SelectPaths validates every direct native invocation before opening a dialog.
func (parseHost *FileDialogHost) SelectPaths(parseContext context.Context, parseRequest FileDialogRequest) FileDialogReply {
	parseFail := func(parseCode interop.ErrorCode, parseErr error) FileDialogReply {
		return FileDialogReply{Code: parseCode, Message: parseErr.Error()}
	}
	if len(parseHost.GetMethods()) == 0 {
		return parseFail(interop.CodeUnavailable, errors.New("file dialogs are disabled or unavailable"))
	}
	if parseContext == nil {
		return parseFail(interop.CodeInvalid, errors.New("calling context required"))
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return getFileDialogFailure(getContextError(FileDialogMethod, parseErr))
	}
	if parseErr := validateFileDialogRequest(parseRequest); parseErr != nil {
		return parseFail(interop.CodeInvalid, parseErr)
	}
	// Copy options so a backend cannot mutate the caller's shared filter slice.
	parseRequest.Options.Filters = append([]FileFilter(nil), parseRequest.Options.Filters...)
	parseSelection, parseErr := parseHost.parseBackend.SelectPaths(parseContext, parseRequest)
	if parseContext.Err() != nil {
		return getFileDialogFailure(getContextError(FileDialogMethod, parseContext.Err()))
	}
	if parseErr != nil {
		return getFileDialogFailure(parseErr)
	}
	if parseErr = validateFileSelection(parseRequest.Kind, parseSelection); parseErr != nil {
		return parseFail(interop.CodeDecode, parseErr)
	}
	parseSelection.Paths = append([]string{}, parseSelection.Paths...)
	return FileDialogReply{Selection: parseSelection}
}

// getFileDialogFailure translates typed backend failures into the wire contract.
func getFileDialogFailure(parseErr error) FileDialogReply {
	if errors.Is(parseErr, context.Canceled) || errors.Is(parseErr, context.DeadlineExceeded) {
		parseErr = getContextError(FileDialogMethod, parseErr)
	}
	parseCode := interop.CodeRemote
	parseTyped, parseOk := errors.AsType[*interop.Error](parseErr)
	if parseOk {
		parseCode = parseTyped.Code
	}
	return FileDialogReply{Code: parseCode, Message: parseErr.Error()}
}

// Supports reports effective workflow availability for optional controls.
func (parseClient Client) Supports(parseFeatures ...Feature) bool {
	return parseClient.Require(parseFeatures...) == nil
}

// Require checks a live bridge and every requested feature; it never grants host permission.
func (parseClient Client) Require(parseFeatures ...Feature) error {
	parseCaps, parseErr := parseClient.GetCapabilities()
	if parseErr != nil {
		return parseErr
	}
	for _, parseFeature := range parseFeatures {
		if parseFeature == NativeMenus || parseFeature == WindowEvents {
			hasMenus := false
			for _, parseAvailable := range parseCaps.Features {
				if parseAvailable == parseFeature {
					hasMenus = true
					break
				}
			}
			if !hasMenus {
				return getError(string(parseFeature), interop.CodeUnavailable, errors.New("desktop feature is disabled or unavailable"))
			}
			continue
		}
		parseMethods := featureMethods(parseFeature)
		if parseMethods == nil {
			return getError(string(parseFeature), interop.CodeInvalid, errors.New("unknown desktop feature"))
		}
		parseAvailable := true
		for _, parseMethod := range parseMethods {
			if !hasName(parseCaps.Methods, parseMethod) {
				parseAvailable = false
				break
			}
		}
		if !parseAvailable {
			return getError(string(parseFeature), interop.CodeUnavailable, errors.New("desktop feature is disabled or unavailable"))
		}
	}
	return nil
}

// featureMethods lists every method required for a complete portable feature.
func featureMethods(parseFeature Feature) []string {
	switch parseFeature {
	case FileDialogs:
		return []string{FileDialogMethod}
	case Clipboard:
		return []string{ClipboardWriteMethod, ClipboardReadMethod}
	case MessageDialogs:
		return []string{MessageMethod}
	case WindowControls:
		return []string{WindowMethod}
	case ChildWindows:
		return []string{ChildWindowCreateMethod, ChildWindowListMethod, ChildWindowInspectMethod, ChildWindowControlMethod}
	case WindowPrinting:
		return []string{WindowPrintMethod}
	case WindowEvents:
		return []string{}
	case Screens:
		return []string{ScreensMethod}
	case ScreenGeometry:
		return []string{ScreenGeometryMethod}
	case RuntimeMenus:
		return []string{MenuReplaceMethod, ContextMenuInstallMethod, ContextMenuShowMethod, ContextMenuRemoveMethod}
	case SystemTray:
		return []string{TrayConfigureMethod}
	case GlobalShortcuts:
		return []string{ShortcutConfigureMethod}
	case PersistentStorage:
		return []string{StorageLoadMethod, StorageSaveMethod, StorageDeleteMethod, StorageKeysMethod}
	case ReportExport:
		return []string{ReportExportMethod}
	case SystemEnvironment:
		return []string{SystemEnvironmentMethod}
	case ExternalURLs:
		return []string{ExternalURLOpenMethod}
	case Autostart:
		return []string{AutostartStatusMethod, AutostartEnableMethod, AutostartDisableMethod}
	case FileManager:
		return []string{FileManagerRevealMethod}
	default:
		return nil
	}
}

// OpenFile opens a single-file picker using the connected desktop host.
func OpenFile(parseContext context.Context, parseOptions FileDialogOptions) (FileSelection, error) {
	parseClient, parseErr := Connect()
	if parseErr != nil {
		return FileSelection{}, parseErr
	}
	return parseClient.OpenFile(parseContext, parseOptions)
}

// OpenFiles opens a multiple-file picker using the connected desktop host.
func OpenFiles(parseContext context.Context, parseOptions FileDialogOptions) (FileSelection, error) {
	parseClient, parseErr := Connect()
	if parseErr != nil {
		return FileSelection{}, parseErr
	}
	return parseClient.OpenFiles(parseContext, parseOptions)
}

// OpenDirectory opens a directory picker using the connected desktop host.
func OpenDirectory(ctx context.Context, options FileDialogOptions) (FileSelection, error) {
	client, err := Connect()
	if err != nil {
		return FileSelection{}, err
	}
	return client.OpenDirectory(ctx, options)
}

// OpenDirectories opens a multiple-directory picker using the connected desktop host.
func OpenDirectories(ctx context.Context, options FileDialogOptions) (FileSelection, error) {
	client, err := Connect()
	if err != nil {
		return FileSelection{}, err
	}
	return client.OpenDirectories(ctx, options)
}

// SaveFile selects a save destination without creating or overwriting a file.
func SaveFile(parseContext context.Context, parseOptions FileDialogOptions) (FileSelection, error) {
	parseClient, parseErr := Connect()
	if parseErr != nil {
		return FileSelection{}, parseErr
	}
	return parseClient.SaveFile(parseContext, parseOptions)
}

// OpenFile selects one file through this explicitly injected client.
func (parseClient Client) OpenFile(parseContext context.Context, parseOptions FileDialogOptions) (FileSelection, error) {
	return parseClient.getFileSelection(parseContext, "open-file", parseOptions)
}

// OpenFiles selects multiple files through this explicitly injected client.
func (parseClient Client) OpenFiles(parseContext context.Context, parseOptions FileDialogOptions) (FileSelection, error) {
	return parseClient.getFileSelection(parseContext, "open-files", parseOptions)
}

// OpenDirectory selects one directory through this explicitly injected client.
func (client Client) OpenDirectory(ctx context.Context, options FileDialogOptions) (FileSelection, error) {
	return client.getFileSelection(ctx, "open-directory", options)
}

// OpenDirectories selects multiple directories through this explicitly injected client.
func (client Client) OpenDirectories(ctx context.Context, options FileDialogOptions) (FileSelection, error) {
	return client.getFileSelection(ctx, "open-directories", options)
}

// SaveFile selects a save destination; no file is written by this operation.
func (parseClient Client) SaveFile(parseContext context.Context, parseOptions FileDialogOptions) (FileSelection, error) {
	return parseClient.getFileSelection(parseContext, "save-file", parseOptions)
}

// getFileSelection hides RPC names, bounded interactive deadlines and wire errors.
func (parseClient Client) getFileSelection(parseContext context.Context, parseKind string, parseOptions FileDialogOptions) (FileSelection, error) {
	parseRequest := FileDialogRequest{Version: 1, Kind: parseKind, Options: parseOptions}
	if parseErr := validateFileDialogRequest(parseRequest); parseErr != nil {
		return FileSelection{}, getError(FileDialogMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := parseClient.Require(FileDialogs); parseErr != nil {
		return FileSelection{}, parseErr
	}
	parseReply, parseErr := CallWithTimeout[FileDialogReply](parseContext, parseClient, 5*time.Minute, FileDialogMethod, parseRequest)
	if parseErr != nil {
		return FileSelection{}, parseErr
	}
	if parseReply.Code != "" {
		return FileSelection{}, getError(FileDialogMethod, parseReply.Code, errors.New(parseReply.Message))
	}
	if parseErr = validateFileSelection(parseKind, parseReply.Selection); parseErr != nil {
		return FileSelection{}, getError(FileDialogMethod, interop.CodeDecode, parseErr)
	}
	return parseReply.Selection, nil
}

// validateFileDialogRequest bounds portable input before either transport or native work.
func validateFileDialogRequest(request FileDialogRequest) error {
	if request.Version != 1 {
		return errors.New("unsupported file-dialog contract version")
	}
	switch request.Kind {
	case "open-file", "open-files", "open-directory", "open-directories", "save-file":
	default:
		return errors.New("unknown file-dialog operation")
	}
	options := request.Options
	if request.Kind != "save-file" && options.Filename != "" {
		return errors.New("filename applies only to SaveFile")
	}
	if (request.Kind == "open-directory" || request.Kind == "open-directories") && len(options.Filters) != 0 {
		return errors.New("file filters do not apply to directory pickers")
	}
	if len(options.Filters) > 16 {
		return errors.New("too many file filters")
	}
	values := []string{options.Title, options.Directory, options.Filename, options.ButtonText}
	for _, filter := range options.Filters {
		if filter.Name == "" || filter.Pattern == "" {
			return errors.New("file filter name and pattern required")
		}
		values = append(values, filter.Name, filter.Pattern)
	}
	for _, value := range values {
		if len(value) > 4096 || !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
			return errors.New("invalid or oversized dialog text")
		}
	}
	return nil
}

// validateFileSelection rejects ambiguous cancellation and malformed backend results.
func validateFileSelection(kind string, selection FileSelection) error {
	if selection.Cancelled {
		if len(selection.Paths) != 0 {
			return errors.New("cancelled selection contains paths")
		}
		return nil
	}
	isMultiple := kind == "open-files" || kind == "open-directories"
	if len(selection.Paths) == 0 || len(selection.Paths) > 256 || (!isMultiple && len(selection.Paths) != 1) {
		return errors.New("invalid file selection count")
	}
	for _, path := range selection.Paths {
		if path == "" || len(path) > 32768 || !utf8.ValidString(path) || strings.ContainsRune(path, 0) {
			return errors.New("invalid selected path")
		}
	}
	return nil
}
