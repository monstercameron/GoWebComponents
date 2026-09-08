//go:build windows

package wails

import (
	"context"
	"errors"
	"fmt"
	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/monstercameron/GoWebComponents/v6/interop"
	"testing"
)

// TestPickerCancellation tests the narrow pinned-version compatibility boundary.
func TestPickerCancellation(parseT *testing.T) {
	for _, parseCase := range []struct {
		parseErr    error
		isCancelled bool
	}{
		{nil, false}, {errors.New("cancelled by user"), true},
		{fmt.Errorf("wrapped: %w", errors.New("cancelled by user")), true},
		{errors.New("disk error: cancelled by user"), false},
		{errors.Join(errors.New("cancelled by user"), errors.New("disk error")), false},
		{context.Canceled, false},
	} {
		if isPickerCancelled(parseCase.parseErr) != parseCase.isCancelled {
			parseT.Fatalf("classification %v", parseCase.parseErr)
		}
	}
}

// TestMissingCallerNeverOpensDialog verifies no focused-window fallback exists.
func TestMissingCallerNeverOpensDialog(parseT *testing.T) {
	parseHost := desktop.NewFileDialogHost(NewFileDialogs(), true)
	parseReply := parseHost.SelectPaths(context.Background(), desktop.FileDialogRequest{Version: 1, Kind: "open-file"})
	if parseReply.Code != interop.CodeUnavailable {
		parseT.Fatalf("missing owner: %+v", parseReply)
	}
}
