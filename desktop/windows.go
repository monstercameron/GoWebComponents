package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/monstercameron/GoWebComponents/v8/interop"
)

// ChildWindows identifies caller-owned, host-templated child-window lifecycle operations.
const ChildWindows Feature = "child-windows"

const (
	ChildWindowCreateMethod  = "desktop.child-window.create"
	ChildWindowListMethod    = "desktop.child-window.list"
	ChildWindowInspectMethod = "desktop.child-window.inspect"
	ChildWindowControlMethod = "desktop.child-window.control"
)

// ChildWindowCreateRequest requests a host-approved template without accepting a URL.
type ChildWindowCreateRequest struct {
	ID         string `json:"id"`
	TemplateID string `json:"templateId"`
	Title      string `json:"title,omitempty"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
}

// ChildWindowRequest identifies one child in the calling window's ownership scope.
type ChildWindowRequest struct {
	ID string `json:"id"`
}

// ChildWindowControlRequest applies a bounded action to an owned child.
type ChildWindowControlRequest struct {
	ID     string `json:"id"`
	Action string `json:"action"`
	Title  string `json:"title,omitempty"`
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// ChildWindowInfo reports portable metadata for a caller-owned child.
type ChildWindowInfo struct {
	ID         string `json:"id"`
	TemplateID string `json:"templateId"`
	Title      string `json:"title"`
	X          int    `json:"x,omitempty"`
	Y          int    `json:"y,omitempty"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Visible    bool   `json:"visible"`
	Focused    bool   `json:"focused,omitempty"`
	Minimised  bool   `json:"minimised,omitempty"`
	Maximised  bool   `json:"maximised,omitempty"`
	Closing    bool   `json:"closing,omitempty"`
}

// ChildWindowBackend is an optional adapter extension; NativeBackend remains source-compatible.
type ChildWindowBackend interface {
	CreateChildWindow(context.Context, ChildWindowCreateRequest) (ChildWindowInfo, error)
	ListChildWindows(context.Context) ([]ChildWindowInfo, error)
	InspectChildWindow(context.Context, ChildWindowRequest) (ChildWindowInfo, error)
	ControlChildWindow(context.Context, ChildWindowControlRequest) (ChildWindowInfo, error)
}

// CreateChildWindow validates and delegates creation to the optional backend.
func (parseHost *NativeHost) CreateChildWindow(parseContext context.Context, parseRequest ChildWindowCreateRequest) (ChildWindowInfo, error) {
	if parseErr := parseHost.Require(ChildWindows); parseErr != nil {
		return ChildWindowInfo{}, parseErr
	}
	parseBackend, parseOK := parseHost.parseBackend.(ChildWindowBackend)
	if !parseOK {
		return ChildWindowInfo{}, getError(ChildWindowCreateMethod, interop.CodeUnavailable, errors.New("child windows are unavailable"))
	}
	if parseErr := validateChildWindowContext(parseContext, ChildWindowCreateMethod); parseErr != nil {
		return ChildWindowInfo{}, parseErr
	}
	if parseErr := validateChildWindowCreateRequest(parseRequest); parseErr != nil {
		return ChildWindowInfo{}, getError(ChildWindowCreateMethod, interop.CodeInvalid, parseErr)
	}
	parseInfo, parseErr := parseBackend.CreateChildWindow(parseContext, parseRequest)
	return validateChildWindowResult(parseContext, ChildWindowCreateMethod, parseInfo, parseErr)
}

// ListChildWindows returns only children resolved by the backend for this caller.
func (parseHost *NativeHost) ListChildWindows(parseContext context.Context) ([]ChildWindowInfo, error) {
	if parseErr := parseHost.Require(ChildWindows); parseErr != nil {
		return nil, parseErr
	}
	parseBackend, parseOK := parseHost.parseBackend.(ChildWindowBackend)
	if !parseOK {
		return nil, getError(ChildWindowListMethod, interop.CodeUnavailable, errors.New("child windows are unavailable"))
	}
	if parseErr := validateChildWindowContext(parseContext, ChildWindowListMethod); parseErr != nil {
		return nil, parseErr
	}
	parseResult, parseErr := parseBackend.ListChildWindows(parseContext)
	if parseContext.Err() != nil {
		return nil, getContextError(ChildWindowListMethod, parseContext.Err())
	}
	if parseErr != nil {
		return nil, parseErr
	}
	for _, parseInfo := range parseResult {
		if parseErr = validateChildWindowInfo(parseInfo); parseErr != nil {
			return nil, getError(ChildWindowListMethod, interop.CodeDecode, parseErr)
		}
	}
	return parseResult, nil
}

// InspectChildWindow inspects one caller-owned child.
func (parseHost *NativeHost) InspectChildWindow(parseContext context.Context, parseRequest ChildWindowRequest) (ChildWindowInfo, error) {
	if parseErr := validateChildWindowID(parseRequest.ID); parseErr != nil {
		return ChildWindowInfo{}, getError(ChildWindowInspectMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := parseHost.Require(ChildWindows); parseErr != nil {
		return ChildWindowInfo{}, parseErr
	}
	parseBackend, parseOK := parseHost.parseBackend.(ChildWindowBackend)
	if !parseOK {
		return ChildWindowInfo{}, getError(ChildWindowInspectMethod, interop.CodeUnavailable, errors.New("child windows are unavailable"))
	}
	if parseErr := validateChildWindowContext(parseContext, ChildWindowInspectMethod); parseErr != nil {
		return ChildWindowInfo{}, parseErr
	}
	parseInfo, parseErr := parseBackend.InspectChildWindow(parseContext, parseRequest)
	return validateChildWindowResult(parseContext, ChildWindowInspectMethod, parseInfo, parseErr)
}

// ControlChildWindow shows, hides, or closes one caller-owned child.
func (parseHost *NativeHost) ControlChildWindow(parseContext context.Context, parseRequest ChildWindowControlRequest) (ChildWindowInfo, error) {
	if parseErr := validateChildWindowControlRequest(parseRequest); parseErr != nil {
		return ChildWindowInfo{}, getError(ChildWindowControlMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := parseHost.Require(ChildWindows); parseErr != nil {
		return ChildWindowInfo{}, parseErr
	}
	parseBackend, parseOK := parseHost.parseBackend.(ChildWindowBackend)
	if !parseOK {
		return ChildWindowInfo{}, getError(ChildWindowControlMethod, interop.CodeUnavailable, errors.New("child windows are unavailable"))
	}
	if parseErr := validateChildWindowContext(parseContext, ChildWindowControlMethod); parseErr != nil {
		return ChildWindowInfo{}, parseErr
	}
	parseInfo, parseErr := parseBackend.ControlChildWindow(parseContext, parseRequest)
	return validateChildWindowResult(parseContext, ChildWindowControlMethod, parseInfo, parseErr)
}

// validateChildWindowResult applies cancellation and adapter-result checks consistently.
func validateChildWindowResult(parseContext context.Context, parseMethod string, parseInfo ChildWindowInfo, parseErr error) (ChildWindowInfo, error) {
	if parseContext.Err() != nil {
		return ChildWindowInfo{}, getContextError(parseMethod, parseContext.Err())
	}
	if parseErr != nil {
		return ChildWindowInfo{}, parseErr
	}
	if parseErr = validateChildWindowInfo(parseInfo); parseErr != nil {
		return ChildWindowInfo{}, getError(parseMethod, interop.CodeDecode, parseErr)
	}
	return parseInfo, nil
}

// validateChildWindowContext prevents native work after cancellation.
func validateChildWindowContext(parseContext context.Context, parseMethod string) error {
	if parseContext == nil {
		return getError(parseMethod, interop.CodeInvalid, errors.New("calling context required"))
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return getContextError(parseMethod, parseErr)
	}
	return nil
}

// CreateChildWindow creates a child from a host-registered same-origin template.
func (parseClient Client) CreateChildWindow(parseContext context.Context, parseRequest ChildWindowCreateRequest) (ChildWindowInfo, error) {
	if parseErr := validateChildWindowCreateRequest(parseRequest); parseErr != nil {
		return ChildWindowInfo{}, getError(ChildWindowCreateMethod, interop.CodeInvalid, parseErr)
	}
	return getChildWindowReply(parseContext, parseClient, ChildWindowCreateMethod, parseRequest)
}

// ListChildWindows lists only children owned by the calling window.
func (parseClient Client) ListChildWindows(parseContext context.Context) ([]ChildWindowInfo, error) {
	if parseErr := parseClient.Require(ChildWindows); parseErr != nil {
		return nil, parseErr
	}
	parseReply, parseErr := parseClient.getNativeReply(parseContext, ChildWindowListMethod)
	if parseErr != nil {
		return nil, parseErr
	}
	var parseResult []ChildWindowInfo
	if parseErr = json.Unmarshal(parseReply.Data, &parseResult); parseErr != nil {
		return nil, getError(ChildWindowListMethod, interop.CodeDecode, parseErr)
	}
	for _, parseInfo := range parseResult {
		if parseErr = validateChildWindowInfo(parseInfo); parseErr != nil {
			return nil, getError(ChildWindowListMethod, interop.CodeDecode, parseErr)
		}
	}
	return parseResult, nil
}

// InspectChildWindow inspects one child in the calling window's ownership scope.
func (parseClient Client) InspectChildWindow(parseContext context.Context, parseID string) (ChildWindowInfo, error) {
	parseRequest := ChildWindowRequest{ID: parseID}
	if parseErr := validateChildWindowID(parseID); parseErr != nil {
		return ChildWindowInfo{}, getError(ChildWindowInspectMethod, interop.CodeInvalid, parseErr)
	}
	return getChildWindowReply(parseContext, parseClient, ChildWindowInspectMethod, parseRequest)
}

// ControlChildWindow shows, hides, or closes one caller-owned child.
func (parseClient Client) ControlChildWindow(parseContext context.Context, parseRequest ChildWindowControlRequest) (ChildWindowInfo, error) {
	if parseErr := validateChildWindowControlRequest(parseRequest); parseErr != nil {
		return ChildWindowInfo{}, getError(ChildWindowControlMethod, interop.CodeInvalid, parseErr)
	}
	return getChildWindowReply(parseContext, parseClient, ChildWindowControlMethod, parseRequest)
}

// ShowChildWindow makes one caller-owned child visible.
func (parseClient Client) ShowChildWindow(parseContext context.Context, parseID string) (ChildWindowInfo, error) {
	return parseClient.ControlChildWindow(parseContext, ChildWindowControlRequest{ID: parseID, Action: "show"})
}

// HideChildWindow hides one caller-owned child without destroying it.
func (parseClient Client) HideChildWindow(parseContext context.Context, parseID string) (ChildWindowInfo, error) {
	return parseClient.ControlChildWindow(parseContext, ChildWindowControlRequest{ID: parseID, Action: "hide"})
}

// CloseChildWindow permanently closes one caller-owned child, never its parent.
func (client Client) CloseChildWindow(ctx context.Context, id string) (ChildWindowInfo, error) {
	return client.ControlChildWindow(ctx, ChildWindowControlRequest{ID: id, Action: "close"})
}

// FocusChildWindow focuses one caller-owned child window.
func (client Client) FocusChildWindow(ctx context.Context, id string) (ChildWindowInfo, error) {
	return client.ControlChildWindow(ctx, ChildWindowControlRequest{ID: id, Action: "focus"})
}

// ResizeChildWindow resizes one caller-owned child window within portable bounds.
func (client Client) ResizeChildWindow(ctx context.Context, id string, width, height int) (ChildWindowInfo, error) {
	return client.ControlChildWindow(ctx, ChildWindowControlRequest{ID: id, Action: "resize", Width: width, Height: height})
}

// SetChildWindowTitle updates the title of one caller-owned child window.
func (client Client) SetChildWindowTitle(ctx context.Context, id, title string) (ChildWindowInfo, error) {
	return client.ControlChildWindow(ctx, ChildWindowControlRequest{ID: id, Action: "set-title", Title: title})
}

// getChildWindowReply invokes and validates a single child-window result.
func getChildWindowReply(parseContext context.Context, parseClient Client, parseMethod string, parseRequest any) (ChildWindowInfo, error) {
	if parseErr := parseClient.Require(ChildWindows); parseErr != nil {
		return ChildWindowInfo{}, parseErr
	}
	parseReply, parseErr := parseClient.getNativeReply(parseContext, parseMethod, parseRequest)
	if parseErr != nil {
		return ChildWindowInfo{}, parseErr
	}
	var parseInfo ChildWindowInfo
	if parseErr = json.Unmarshal(parseReply.Data, &parseInfo); parseErr != nil {
		return ChildWindowInfo{}, getError(parseMethod, interop.CodeDecode, parseErr)
	}
	if parseErr = validateChildWindowInfo(parseInfo); parseErr != nil {
		return ChildWindowInfo{}, getError(parseMethod, interop.CodeDecode, parseErr)
	}
	return parseInfo, nil
}

// decodeChildWindowRequest rejects unknown fields so a URL cannot be smuggled into create.
func decodeChildWindowRequest(parseData json.RawMessage, parseTarget any) error {
	parseDecoder := json.NewDecoder(bytes.NewReader(parseData))
	parseDecoder.DisallowUnknownFields()
	if parseErr := parseDecoder.Decode(parseTarget); parseErr != nil {
		return getError("child-window request", interop.CodeInvalid, parseErr)
	}
	if parseErr := parseDecoder.Decode(&struct{}{}); parseErr != io.EOF {
		return getError("child-window request", interop.CodeInvalid, errors.New("unexpected data after child-window request"))
	}
	return nil
}

// validateChildWindowCreateRequest bounds caller-selected presentation and identifiers.
func validateChildWindowCreateRequest(parseRequest ChildWindowCreateRequest) error {
	if parseErr := validateChildWindowID(parseRequest.ID); parseErr != nil {
		return parseErr
	}
	if parseErr := validateChildWindowID(parseRequest.TemplateID); parseErr != nil {
		return errors.New("invalid child-window template id")
	}
	if !utf8.ValidString(parseRequest.Title) || strings.ContainsRune(parseRequest.Title, 0) || len(parseRequest.Title) > 256 {
		return errors.New("invalid child-window title")
	}
	if parseRequest.Width < 0 || parseRequest.Height < 0 || parseRequest.Width > 4096 || parseRequest.Height > 4096 {
		return errors.New("child-window dimensions must be zero or between 1 and 4096")
	}
	if (parseRequest.Width == 0) != (parseRequest.Height == 0) {
		return errors.New("child-window width and height must be supplied together")
	}
	return nil
}

// validateChildWindowControlRequest permits bounded caller-owned child operations only.
func validateChildWindowControlRequest(request ChildWindowControlRequest) error {
	if err := validateChildWindowID(request.ID); err != nil {
		return err
	}
	switch request.Action {
	case "show", "hide", "close", "focus", "minimize", "unminimize", "maximize", "unmaximize", "restore", "center":
		if request.Title != "" || request.X != 0 || request.Y != 0 || request.Width != 0 || request.Height != 0 {
			return errors.New("child-window action does not accept arguments")
		}
		return nil
	case "set-title":
		if !utf8.ValidString(request.Title) || strings.ContainsRune(request.Title, 0) || len(request.Title) > 256 {
			return errors.New("invalid child-window title")
		}
		if request.X != 0 || request.Y != 0 || request.Width != 0 || request.Height != 0 {
			return errors.New("child-window set-title accepts only title")
		}
		return nil
	case "resize":
		if request.Width < 1 || request.Width > 4096 || request.Height < 1 || request.Height > 4096 {
			return errors.New("child-window dimensions must be between 1 and 4096")
		}
		if request.Title != "" || request.X != 0 || request.Y != 0 {
			return errors.New("child-window resize accepts only width and height")
		}
		return nil
	case "set-bounds":
		if request.Width < 1 || request.Width > 4096 || request.Height < 1 || request.Height > 4096 {
			return errors.New("child-window dimensions must be between 1 and 4096")
		}
		if request.X < -100000 || request.X > 100000 || request.Y < -100000 || request.Y > 100000 || request.Title != "" {
			return errors.New("child-window bounds out of range")
		}
		return nil
	default:
		return errors.New("unknown child-window action")
	}
}

// validateChildWindowID accepts compact caller-scoped identifiers.
func validateChildWindowID(id string) error {
	if id == "" || len(id) > 128 || !utf8.ValidString(id) || strings.ContainsRune(id, 0) {
		return errors.New("invalid child-window id")
	}
	return nil
}

// validateChildWindowInfo rejects malformed adapter results.
func validateChildWindowInfo(info ChildWindowInfo) error {
	if err := validateChildWindowID(info.ID); err != nil {
		return err
	}
	if err := validateChildWindowID(info.TemplateID); err != nil {
		return errors.New("invalid child-window template id")
	}
	if !utf8.ValidString(info.Title) || strings.ContainsRune(info.Title, 0) || len(info.Title) > 256 || info.X < -100000 || info.X > 100000 || info.Y < -100000 || info.Y > 100000 || info.Width <= 0 || info.Height <= 0 || info.Width > 4096 || info.Height > 4096 {
		return errors.New("invalid child-window metadata")
	}
	return nil
}
