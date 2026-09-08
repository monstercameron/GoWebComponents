package services

import (
	"context"
	"errors"
	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// selectFileDialog shares the same gated adapter across legacy and lab entry points.
func selectFileDialog(parseContext context.Context, parseHost *desktop.FileDialogHost, parseKind string, parseOptions desktop.FileDialogOptions) (desktop.FileSelection, error) {
	parseReply := parseHost.SelectPaths(parseContext, desktop.FileDialogRequest{Version: 1, Kind: parseKind, Options: parseOptions})
	if parseReply.Code != "" {
		return desktop.FileSelection{}, &interop.Error{Op: "desktop", Target: desktop.FileDialogMethod, Code: parseReply.Code, Err: errors.New(parseReply.Message)}
	}
	return parseReply.Selection, nil
}
