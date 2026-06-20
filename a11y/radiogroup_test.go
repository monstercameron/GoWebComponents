package a11y_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/a11y"
	"github.com/monstercameron/GoWebComponents/ui"
)

func renderRadioGroup(t *testing.T, parseProps a11y.RadioGroupProps) string {
	t.Helper()
	parseMarkup, parseErr := ui.RenderToString(a11y.RadioGroup(parseProps))
	if parseErr != nil {
		t.Fatalf("render: %v", parseErr)
	}
	return parseMarkup
}

func TestRadioGroupEmitsARIAAndRovingTabIndex(t *testing.T) {
	parseOut := renderRadioGroup(t, a11y.RadioGroupProps{
		ID:    "size",
		Label: "Size",
		Items: []a11y.Item{
			{ID: "s", Label: "Small"},
			{ID: "m", Label: "Medium", Selected: true},
			{ID: "l", Label: "Large"},
		},
	})
	for _, parseWant := range []string{
		`role="radiogroup"`, `aria-label="Size"`, `aria-orientation="horizontal"`, `role="radio"`,
	} {
		if !strings.Contains(parseOut, parseWant) {
			t.Fatalf("missing %q in:\n%s", parseWant, parseOut)
		}
	}
	// Exactly one tab stop, and it is the selected radio "m" (attrs are serialized
	// in sorted order: …id role tabindex).
	if parseN := strings.Count(parseOut, `tabindex="0"`); parseN != 1 {
		t.Fatalf("expected exactly one tabindex=0, got %d:\n%s", parseN, parseOut)
	}
	if !strings.Contains(parseOut, `id="m" role="radio" tabindex="0"`) {
		t.Fatalf("selected radio should carry the tab stop:\n%s", parseOut)
	}
	if strings.Count(parseOut, `aria-checked="true"`) != 1 {
		t.Fatalf("expected exactly one checked radio:\n%s", parseOut)
	}
}

func TestRadioGroupNoSelectionTabsFirstEnabled(t *testing.T) {
	parseOut := renderRadioGroup(t, a11y.RadioGroupProps{
		ID: "g",
		Items: []a11y.Item{
			{ID: "a", Label: "A"},
			{ID: "b", Label: "B"},
		},
	})
	if strings.Count(parseOut, `tabindex="0"`) != 1 {
		t.Fatalf("expected one tab stop when nothing selected:\n%s", parseOut)
	}
	if !strings.Contains(parseOut, `id="a" role="radio" tabindex="0"`) {
		t.Fatalf("first enabled radio should be the tab stop:\n%s", parseOut)
	}
	if strings.Contains(parseOut, `aria-checked="true"`) {
		t.Fatalf("no radio should be checked when none selected:\n%s", parseOut)
	}
}

func TestRadioGroupDisabledIsSkippedForTabStop(t *testing.T) {
	parseOut := renderRadioGroup(t, a11y.RadioGroupProps{
		ID: "g",
		Items: []a11y.Item{
			{ID: "a", Label: "A", Disabled: true},
			{ID: "b", Label: "B"},
		},
	})
	// The disabled first item must NOT be the tab stop; b is.
	if !strings.Contains(parseOut, `id="b" role="radio" tabindex="0"`) {
		t.Fatalf("disabled item should be skipped for the tab stop:\n%s", parseOut)
	}
	if strings.Count(parseOut, `tabindex="0"`) != 1 {
		t.Fatalf("expected exactly one tab stop:\n%s", parseOut)
	}
}

func TestRadioGroupSelectedButDisabledFallsBack(t *testing.T) {
	// A selected-but-disabled radio is not a valid tab stop; the first enabled
	// radio takes it instead.
	parseOut := renderRadioGroup(t, a11y.RadioGroupProps{
		ID: "g",
		Items: []a11y.Item{
			{ID: "a", Label: "A"},
			{ID: "b", Label: "B", Selected: true, Disabled: true},
		},
	})
	if !strings.Contains(parseOut, `id="a" role="radio" tabindex="0"`) {
		t.Fatalf("selected-but-disabled radio must not be the tab stop:\n%s", parseOut)
	}
}

func TestRadioGroupVerticalOrientationAndEmpty(t *testing.T) {
	parseOut := renderRadioGroup(t, a11y.RadioGroupProps{
		ID: "g", Orientation: "vertical",
		Items: []a11y.Item{{ID: "a", Label: "A"}},
	})
	if !strings.Contains(parseOut, `aria-orientation="vertical"`) {
		t.Fatalf("vertical orientation not honored:\n%s", parseOut)
	}
	// Empty group must not panic and still be a radiogroup.
	parseEmpty := renderRadioGroup(t, a11y.RadioGroupProps{ID: "e"})
	if !strings.Contains(parseEmpty, `role="radiogroup"`) {
		t.Fatalf("empty group should still render the container:\n%s", parseEmpty)
	}
	if strings.Contains(parseEmpty, `tabindex="0"`) {
		t.Fatalf("empty group should have no tab stop:\n%s", parseEmpty)
	}
}
