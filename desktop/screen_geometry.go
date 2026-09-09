package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// ScreenGeometry identifies optional DIP/physical display coordinate operations.
const ScreenGeometry Feature = "screen-geometry"

// ScreenGeometryMethod is the versioned native method for display transforms.
const ScreenGeometryMethod = "desktop.screens.geometry"

// ScreenGeometryOperation identifies one ScreenManager operation.
type ScreenGeometryOperation string

const (
	ScreenDipToPhysicalPoint   ScreenGeometryOperation = "dip-to-physical-point"
	ScreenPhysicalToDipPoint   ScreenGeometryOperation = "physical-to-dip-point"
	ScreenDipToPhysicalRect    ScreenGeometryOperation = "dip-to-physical-rect"
	ScreenPhysicalToDipRect    ScreenGeometryOperation = "physical-to-dip-rect"
	ScreenNearestDipPoint      ScreenGeometryOperation = "nearest-dip-point"
	ScreenNearestPhysicalPoint ScreenGeometryOperation = "nearest-physical-point"
	ScreenNearestDipRect       ScreenGeometryOperation = "nearest-dip-rect"
	ScreenNearestPhysicalRect  ScreenGeometryOperation = "nearest-physical-rect"
)

// ScreenPoint is a point in either DIP or physical screen coordinates.
type ScreenPoint struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// ScreenRect is a non-empty rectangle in either DIP or physical coordinates.
type ScreenRect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ScreenGeometryRequest carries one bounded ScreenManager operation.
type ScreenGeometryRequest struct {
	Operation ScreenGeometryOperation `json:"operation"`
	Point     ScreenPoint             `json:"point,omitempty"`
	Rect      ScreenRect              `json:"rect,omitempty"`
}

// ScreenGeometryReply carries a transformed value or the nearest screen ID.
type ScreenGeometryReply struct {
	Point    ScreenPoint `json:"point,omitempty"`
	Rect     ScreenRect  `json:"rect,omitempty"`
	ScreenID string      `json:"screenId,omitempty"`
}

// ScreenGeometryBackend is an optional backend extension for typed screen transforms.
type ScreenGeometryBackend interface {
	ScreenGeometry(context.Context, ScreenGeometryRequest) (ScreenGeometryReply, error)
}

const screenGeometryCoordinateLimit = 1 << 26

// ScreenGeometry invokes one optional typed screen transform on the native host.
func (parseHost *NativeHost) ScreenGeometry(parseContext context.Context, parseRequest ScreenGeometryRequest) (ScreenGeometryReply, error) {
	if parseErr := parseHost.Require(ScreenGeometry); parseErr != nil {
		return ScreenGeometryReply{}, parseErr
	}
	parseBackend, parseOK := parseHost.parseBackend.(ScreenGeometryBackend)
	if !parseOK {
		return ScreenGeometryReply{}, getError(ScreenGeometryMethod, interop.CodeUnavailable, errors.New("screen geometry backend unavailable"))
	}
	if parseErr := validateSystemContext(parseContext, ScreenGeometryMethod); parseErr != nil {
		return ScreenGeometryReply{}, parseErr
	}
	if parseErr := validateScreenGeometryRequest(parseRequest); parseErr != nil {
		return ScreenGeometryReply{}, getError(ScreenGeometryMethod, interop.CodeInvalid, parseErr)
	}
	parseReply, parseErr := parseBackend.ScreenGeometry(parseContext, parseRequest)
	if parseContext.Err() != nil {
		return ScreenGeometryReply{}, getContextError(ScreenGeometryMethod, parseContext.Err())
	}
	if parseErr != nil {
		return ScreenGeometryReply{}, parseErr
	}
	if parseErr = validateScreenGeometryReply(parseRequest, parseReply); parseErr != nil {
		return ScreenGeometryReply{}, getError(ScreenGeometryMethod, interop.CodeDecode, parseErr)
	}
	return parseReply, nil
}

// ScreenGeometry invokes the connected host's typed screen transform method.
func (parseClient Client) ScreenGeometry(parseContext context.Context, parseRequest ScreenGeometryRequest) (ScreenGeometryReply, error) {
	if parseErr := parseClient.Require(ScreenGeometry); parseErr != nil {
		return ScreenGeometryReply{}, parseErr
	}
	if parseErr := validateScreenGeometryRequest(parseRequest); parseErr != nil {
		return ScreenGeometryReply{}, getError(ScreenGeometryMethod, interop.CodeInvalid, parseErr)
	}
	parseReply, parseErr := parseClient.getNativeReply(parseContext, ScreenGeometryMethod, parseRequest)
	if parseErr != nil {
		return ScreenGeometryReply{}, parseErr
	}
	var parseValue ScreenGeometryReply
	if parseErr = json.Unmarshal(parseReply.Data, &parseValue); parseErr != nil {
		return ScreenGeometryReply{}, getError(ScreenGeometryMethod, interop.CodeDecode, parseErr)
	}
	if parseErr = validateScreenGeometryReply(parseRequest, parseValue); parseErr != nil {
		return ScreenGeometryReply{}, getError(ScreenGeometryMethod, interop.CodeDecode, parseErr)
	}
	return parseValue, nil
}

// validateScreenGeometryRequest rejects unbounded coordinates and mixed operation inputs.
func validateScreenGeometryRequest(parseRequest ScreenGeometryRequest) error {
	if !isScreenGeometryPoint(parseRequest.Point) || !isScreenGeometryRect(parseRequest.Rect) {
		return errors.New("screen geometry coordinates are out of bounds")
	}
	switch parseRequest.Operation {
	case ScreenDipToPhysicalPoint, ScreenPhysicalToDipPoint, ScreenNearestDipPoint, ScreenNearestPhysicalPoint:
		if parseRequest.Rect != (ScreenRect{}) {
			return errors.New("point operation cannot include a rectangle")
		}
		return nil
	case ScreenDipToPhysicalRect, ScreenPhysicalToDipRect, ScreenNearestDipRect, ScreenNearestPhysicalRect:
		if parseRequest.Point != (ScreenPoint{}) {
			return errors.New("rectangle operation cannot include a point")
		}
		if parseRequest.Rect.Width <= 0 || parseRequest.Rect.Height <= 0 {
			return errors.New("screen geometry rectangle must be non-empty")
		}
		return nil
	default:
		return errors.New("unknown screen geometry operation")
	}
}

// validateScreenGeometryReply rejects malformed backend results while preserving operation units.
func validateScreenGeometryReply(parseRequest ScreenGeometryRequest, parseReply ScreenGeometryReply) error {
	switch parseRequest.Operation {
	case ScreenDipToPhysicalPoint, ScreenPhysicalToDipPoint:
		if !isScreenGeometryPoint(parseReply.Point) || parseReply.ScreenID != "" || parseReply.Rect != (ScreenRect{}) {
			return errors.New("invalid screen geometry point reply")
		}
	case ScreenDipToPhysicalRect, ScreenPhysicalToDipRect:
		if !isScreenGeometryRect(parseReply.Rect) || parseReply.Rect.Width <= 0 || parseReply.Rect.Height <= 0 || parseReply.ScreenID != "" || parseReply.Point != (ScreenPoint{}) {
			return errors.New("invalid screen geometry rectangle reply")
		}
	case ScreenNearestDipPoint, ScreenNearestPhysicalPoint, ScreenNearestDipRect, ScreenNearestPhysicalRect:
		if !isScreenGeometryID(parseReply.ScreenID) || parseReply.Point != (ScreenPoint{}) || parseReply.Rect != (ScreenRect{}) {
			return errors.New("invalid nearest screen reply")
		}
	}
	return nil
}

// isScreenGeometryPoint bounds integer coordinates against native API limits.
func isScreenGeometryPoint(parsePoint ScreenPoint) bool {
	return absScreenGeometryCoordinate(parsePoint.X) && absScreenGeometryCoordinate(parsePoint.Y)
}

// isScreenGeometryRect bounds rectangle origin and dimensions against native API limits.
func isScreenGeometryRect(parseRect ScreenRect) bool {
	return absScreenGeometryCoordinate(parseRect.X) && absScreenGeometryCoordinate(parseRect.Y) && parseRect.Width >= 0 && parseRect.Width <= screenGeometryCoordinateLimit && parseRect.Height >= 0 && parseRect.Height <= screenGeometryCoordinateLimit
}

// absScreenGeometryCoordinate checks finite portable integer coordinates without overflow.
func absScreenGeometryCoordinate(parseValue int) bool {
	return parseValue >= -screenGeometryCoordinateLimit && parseValue <= screenGeometryCoordinateLimit
}

// isScreenGeometryID bounds a backend-provided display identifier.
func isScreenGeometryID(parseValue string) bool {
	return len(parseValue) > 0 && len(parseValue) <= 256 && utf8.ValidString(parseValue) && !strings.ContainsRune(parseValue, 0)
}
