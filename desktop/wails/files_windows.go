//go:build windows

// Package wails implements optional native backends for portable GWC desktop APIs.
package wails

import (
	"context"
	"errors"

	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/monstercameron/GoWebComponents/v6/interop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type fileDialogs struct{}

// NewFileDialogs returns a Wails backend; wrap it in an explicitly enabled FileDialogHost.
func NewFileDialogs() desktop.FileDialogBackend { return fileDialogs{} }

// SelectPaths translates portable options and uses the RPC caller as native owner.
func (fileDialogs) SelectPaths(parseContext context.Context, parseRequest desktop.FileDialogRequest) (desktop.FileSelection, error) {
	parseUnavailable := func(parseMessage string) (desktop.FileSelection, error) {
		return desktop.FileSelection{}, &interop.Error{Op: "desktop", Target: desktop.FileDialogMethod, Code: interop.CodeUnavailable, Err: errors.New(parseMessage)}
	}
	if parseContext == nil {
		return parseUnavailable("calling window context required")
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return desktop.FileSelection{}, parseErr
	}
	parseWindow, isWindow := parseContext.Value(application.WindowKey).(application.Window)
	if !isWindow || parseWindow == nil {
		return parseUnavailable("calling window unavailable")
	}
	parseApp := application.Get()
	if parseApp == nil {
		return parseUnavailable("Wails application unavailable")
	}
	parseOptions := parseRequest.Options
	var parsePaths []string
	var parseErr error
	if parseRequest.Kind == "save-file" {
		parseDialog := parseApp.Dialog.SaveFile()
		parseDialog.SetOptions(&application.SaveFileDialogOptions{Title: parseOptions.Title})
		parseDialog.SetDirectory(parseOptions.Directory).SetFilename(parseOptions.Filename).AttachToWindow(parseWindow)
		for _, parseFilter := range parseOptions.Filters {
			parseDialog.AddFilter(parseFilter.Name, parseFilter.Pattern)
		}
		var parsePath string
		parsePath, parseErr = parseDialog.PromptForSingleSelection()
		if parsePath != "" {
			parsePaths = []string{parsePath}
		}
	} else {
		parseDialog := parseApp.Dialog.OpenFile().SetTitle(parseOptions.Title).SetDirectory(parseOptions.Directory).AttachToWindow(parseWindow)
		switch parseRequest.Kind {
		case "open-file", "open-files":
			parseDialog.CanChooseFiles(true).CanChooseDirectories(false)
		case "open-directory":
			parseDialog.CanChooseFiles(false).CanChooseDirectories(true)
		default:
			return desktop.FileSelection{}, &interop.Error{Op: "desktop", Code: interop.CodeInvalid, Err: errors.New("unknown file-dialog operation")}
		}
		for _, parseFilter := range parseOptions.Filters {
			parseDialog.AddFilter(parseFilter.Name, parseFilter.Pattern)
		}
		if parseRequest.Kind == "open-files" {
			parsePaths, parseErr = parseDialog.PromptForMultipleSelection()
		} else {
			var parsePath string
			parsePath, parseErr = parseDialog.PromptForSingleSelection()
			if parsePath != "" {
				parsePaths = []string{parsePath}
			}
		}
	}
	if isPickerCancelled(parseErr) {
		return desktop.FileSelection{Paths: []string{}, Cancelled: true}, nil
	}
	if parseErr != nil {
		return desktop.FileSelection{}, parseErr
	}
	return desktop.FileSelection{Paths: parsePaths, Cancelled: len(parsePaths) == 0}, nil
}

// isPickerCancelled isolates beta.17's private cancellation sentinel compatibility.
// The upstream internal package cannot be imported. Match only a single exact leaf;
// mixed joined errors must remain failures. Remove this shim when upstream exports it.
func isPickerCancelled(parseErr error) bool {
	if parseErr == nil {
		return false
	}
	for {
		if _, isJoined := parseErr.(interface{ Unwrap() []error }); isJoined {
			return false
		}
		parseNext := errors.Unwrap(parseErr)
		if parseNext == nil {
			return parseErr.Error() == "cancelled by user"
		}
		parseErr = parseNext
	}
}
