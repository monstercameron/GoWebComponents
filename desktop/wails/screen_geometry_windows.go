//go:build windows

package wails

import (
	"context"
	"errors"

	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ScreenGeometry performs the exact pinned Wails ScreenManager transform or nearest-screen operation.
func (nativeBackend) ScreenGeometry(ctx context.Context, req desktop.ScreenGeometryRequest) (desktop.ScreenGeometryReply, error) {
	if err := validateCaller(ctx); err != nil {
		return desktop.ScreenGeometryReply{}, err
	}
	app := application.Get()
	if app == nil || app.Screen == nil {
		return desktop.ScreenGeometryReply{}, unavailable("Wails screen manager unavailable")
	}
	point := application.Point{X: req.Point.X, Y: req.Point.Y}
	rect := application.Rect{X: req.Rect.X, Y: req.Rect.Y, Width: req.Rect.Width, Height: req.Rect.Height}
	switch req.Operation {
	case desktop.ScreenDipToPhysicalPoint:
		return desktop.ScreenGeometryReply{Point: getScreenPoint(app.Screen.DipToPhysicalPoint(point))}, nil
	case desktop.ScreenPhysicalToDipPoint:
		return desktop.ScreenGeometryReply{Point: getScreenPoint(app.Screen.PhysicalToDipPoint(point))}, nil
	case desktop.ScreenDipToPhysicalRect:
		return desktop.ScreenGeometryReply{Rect: getScreenRect(app.Screen.DipToPhysicalRect(rect))}, nil
	case desktop.ScreenPhysicalToDipRect:
		return desktop.ScreenGeometryReply{Rect: getScreenRect(app.Screen.PhysicalToDipRect(rect))}, nil
	case desktop.ScreenNearestDipPoint:
		return buildNearestScreenReply(app.Screen.ScreenNearestDipPoint(point))
	case desktop.ScreenNearestPhysicalPoint:
		return buildNearestScreenReply(app.Screen.ScreenNearestPhysicalPoint(point))
	case desktop.ScreenNearestDipRect:
		return buildNearestScreenReply(app.Screen.ScreenNearestDipRect(rect))
	case desktop.ScreenNearestPhysicalRect:
		return buildNearestScreenReply(app.Screen.ScreenNearestPhysicalRect(rect))
	default:
		return desktop.ScreenGeometryReply{}, invalid("unknown screen geometry operation")
	}
}

// getScreenPoint converts one Wails point without exposing application types.
func getScreenPoint(point application.Point) desktop.ScreenPoint {
	return desktop.ScreenPoint{X: point.X, Y: point.Y}
}

// getScreenRect converts one Wails rectangle without exposing application types.
func getScreenRect(rect application.Rect) desktop.ScreenRect {
	return desktop.ScreenRect{X: rect.X, Y: rect.Y, Width: rect.Width, Height: rect.Height}
}

// buildNearestScreenReply converts a Wails nearest-screen result.
func buildNearestScreenReply(screen *application.Screen) (desktop.ScreenGeometryReply, error) {
	if screen == nil || screen.ID == "" {
		return desktop.ScreenGeometryReply{}, errors.New("nearest screen unavailable")
	}
	return desktop.ScreenGeometryReply{ScreenID: screen.ID}, nil
}
