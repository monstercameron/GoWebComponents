package a11y_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/a11y"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func TestAlertDialogEmitsARIA(t *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(a11y.AlertDialog(a11y.AlertDialogProps{
		ID:      "wipe",
		Title:   "Delete everything?",
		Message: "This cannot be undone.",
		Buttons: []a11y.DialogButton{
			{ID: "cancel", Label: "Cancel"},
			{ID: "ok", Label: "Delete", Primary: true},
		},
	}))
	if parseErr != nil {
		t.Fatalf("render: %v", parseErr)
	}
	for _, parseWant := range []string{
		`role="alertdialog"`, `aria-modal="true"`,
		`aria-labelledby="wipe-title"`, `aria-describedby="wipe-desc"`,
		`id="wipe-title"`, `Delete everything?`,
		`id="wipe-desc"`, `This cannot be undone.`,
		`id="cancel"`, `id="ok"`, `Delete`,
	} {
		if !strings.Contains(parseMarkup, parseWant) {
			t.Fatalf("missing %q in:\n%s", parseWant, parseMarkup)
		}
	}
}
