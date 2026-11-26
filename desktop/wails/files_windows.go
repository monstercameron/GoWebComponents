//go:build windows

// Package wails implements optional native backends for portable GWC desktop APIs.
package wails

import (
	"context"
	"errors"

	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/monstercameron/GoWebComponents/v8/interop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type fileDialogs struct{}

// NewFileDialogs returns a Wails backend; wrap it in an explicitly enabled FileDialogHost.
func NewFileDialogs() desktop.FileDialogBackend { return fileDialogs{} }

// SelectPaths translates portable options and uses the RPC caller as native owner.
func (fileDialogs) SelectPaths(ctx context.Context, req desktop.FileDialogRequest) (desktop.FileSelection, error) {
	failUnavailable := func(message string) (desktop.FileSelection, error) {
		return desktop.FileSelection{}, &interop.Error{
			Op:     "desktop",
			Target: desktop.FileDialogMethod,
			Code:   interop.CodeUnavailable,
			Err:    errors.New(message),
		}
	}
	if ctx == nil {
		return failUnavailable("calling window context required")
	}
	if err := ctx.Err(); err != nil {
		return desktop.FileSelection{}, err
	}
	win, isWindow := ctx.Value(application.WindowKey).(application.Window)
	if !isWindow || win == nil {
		return failUnavailable("calling window unavailable")
	}
	app := application.Get()
	if app == nil {
		return failUnavailable("Wails application unavailable")
	}
	opts := req.Options
	// The Wails Windows common-file-dialog adapter does not expose these controls;
	// reject them instead of silently dropping caller intent.
	if opts.ShowHiddenFiles || opts.ButtonText != "" {
		return desktop.FileSelection{}, &interop.Error{
			Op:     "desktop",
			Target: desktop.FileDialogMethod,
			Code:   interop.CodeInvalid,
			Err:    errors.New("Windows file dialogs do not support hidden-file visibility or custom button text"),
		}
	}
	var paths []string
	var err error
	if req.Kind == "save-file" {
		dialog := app.Dialog.SaveFile()
		dialog.SetOptions(&application.SaveFileDialogOptions{Title: opts.Title})
		dialog.SetDirectory(opts.Directory).SetFilename(opts.Filename).AttachToWindow(win)
		for _, filter := range opts.Filters {
			dialog.AddFilter(filter.Name, filter.Pattern)
		}
		var path string
		path, err = dialog.PromptForSingleSelection()
		if path != "" {
			paths = []string{path}
		}
	} else {
		dialog := app.Dialog.OpenFile().SetTitle(opts.Title).SetDirectory(opts.Directory).AttachToWindow(win)
		isMultiple := false
		switch req.Kind {
		case "open-file":
			dialog.CanChooseFiles(true).CanChooseDirectories(false)
		case "open-files":
			dialog.CanChooseFiles(true).CanChooseDirectories(false)
			isMultiple = true
		case "open-directory":
			dialog.CanChooseFiles(false).CanChooseDirectories(true)
		case "open-directories":
			dialog.CanChooseFiles(false).CanChooseDirectories(true)
			isMultiple = true
		default:
			return desktop.FileSelection{}, &interop.Error{
				Op:   "desktop",
				Code: interop.CodeInvalid,
				Err:  errors.New("unknown file-dialog operation"),
			}
		}
		for _, filter := range opts.Filters {
			dialog.AddFilter(filter.Name, filter.Pattern)
		}
		if isMultiple {
			paths, err = dialog.PromptForMultipleSelection()
		} else {
			var path string
			path, err = dialog.PromptForSingleSelection()
			if path != "" {
				paths = []string{path}
			}
		}
	}
	if isPickerCancelled(err) {
		return desktop.FileSelection{Paths: []string{}, Cancelled: true}, nil
	}
	if err != nil {
		return desktop.FileSelection{}, err
	}
	return desktop.FileSelection{Paths: paths, Cancelled: len(paths) == 0}, nil
}

// isPickerCancelled isolates Wails v3's private CFD cancellation sentinel.
// The upstream internal package cannot be imported. Match only a single exact leaf;
// mixed joined errors must remain failures.
func isPickerCancelled(err error) bool {
	if err == nil {
		return false
	}
	for {
		if _, isJoined := err.(interface{ Unwrap() []error }); isJoined {
			return false
		}
		next := errors.Unwrap(err)
		if next == nil {
			return err.Error() == "cancelled by user"
		}
		err = next
	}
}
