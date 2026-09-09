//go:build windows

package wails

import (
	"context"
	"errors"

	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ScreenGeometry performs the exact pinned Wails ScreenManager transform or nearest-screen operation.
func (parseBackend nativeBackend) ScreenGeometry(parseContext context.Context, parseRequest desktop.ScreenGeometryRequest) (desktop.ScreenGeometryReply, error) {
	if _, parseErr := getCaller(parseContext); parseErr != nil {
		return desktop.ScreenGeometryReply{}, parseErr
	}
	parseApp := application.Get()
	if parseApp == nil || parseApp.Screen == nil {
		return desktop.ScreenGeometryReply{}, unavailable("Wails screen manager unavailable")
	}
	parsePoint := application.Point{X: parseRequest.Point.X, Y: parseRequest.Point.Y}
	parseRect := application.Rect{X: parseRequest.Rect.X, Y: parseRequest.Rect.Y, Width: parseRequest.Rect.Width, Height: parseRequest.Rect.Height}
	switch parseRequest.Operation {
	case desktop.ScreenDipToPhysicalPoint:
		return desktop.ScreenGeometryReply{Point: getScreenPoint(parseApp.Screen.DipToPhysicalPoint(parsePoint))}, nil
	case desktop.ScreenPhysicalToDipPoint:
		return desktop.ScreenGeometryReply{Point: getScreenPoint(parseApp.Screen.PhysicalToDipPoint(parsePoint))}, nil
	case desktop.ScreenDipToPhysicalRect:
		return desktop.ScreenGeometryReply{Rect: getScreenRect(parseApp.Screen.DipToPhysicalRect(parseRect))}, nil
	case desktop.ScreenPhysicalToDipRect:
		return desktop.ScreenGeometryReply{Rect: getScreenRect(parseApp.Screen.PhysicalToDipRect(parseRect))}, nil
	case desktop.ScreenNearestDipPoint:
		return getNearestScreenPointReply(parseApp.Screen.ScreenNearestDipPoint(parsePoint))
	case desktop.ScreenNearestPhysicalPoint:
		return getNearestScreenPointReply(parseApp.Screen.ScreenNearestPhysicalPoint(parsePoint))
	case desktop.ScreenNearestDipRect:
		return getNearestScreenRectReply(parseApp.Screen.ScreenNearestDipRect(parseRect))
	case desktop.ScreenNearestPhysicalRect:
		return getNearestScreenRectReply(parseApp.Screen.ScreenNearestPhysicalRect(parseRect))
	default:
		return desktop.ScreenGeometryReply{}, invalid("unknown screen geometry operation")
	}
}

// getScreenPoint converts one Wails point without exposing application types.
func getScreenPoint(parsePoint application.Point) desktop.ScreenPoint {
	return desktop.ScreenPoint{X: parsePoint.X, Y: parsePoint.Y}
}

// getScreenRect converts one Wails rectangle without exposing application types.
func getScreenRect(parseRect application.Rect) desktop.ScreenRect {
	return desktop.ScreenRect{X: parseRect.X, Y: parseRect.Y, Width: parseRect.Width, Height: parseRect.Height}
}

// getNearestScreenPointReply converts a Wails nearest-screen result.
func getNearestScreenPointReply(parseScreen *application.Screen) (desktop.ScreenGeometryReply, error) {
	if parseScreen == nil || parseScreen.ID == "" {
		return desktop.ScreenGeometryReply{}, errors.New("nearest screen unavailable")
	}
	return desktop.ScreenGeometryReply{ScreenID: parseScreen.ID}, nil
}

// getNearestScreenRectReply converts a Wails nearest-screen result.
func getNearestScreenRectReply(parseScreen *application.Screen) (desktop.ScreenGeometryReply, error) {
	if parseScreen == nil || parseScreen.ID == "" {
		return desktop.ScreenGeometryReply{}, errors.New("nearest screen unavailable")
	}
	return desktop.ScreenGeometryReply{ScreenID: parseScreen.ID}, nil
}
