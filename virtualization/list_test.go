package virtualization

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
)

func setInteropElementField(t *testing.T, element *interop.Element, field string, value interface{}) {
	t.Helper()
	structValue := reflect.ValueOf(element).Elem()
	target := structValue.FieldByName(field)
	if !target.IsValid() {
		t.Fatalf("missing interop.Element field %q", field)
	}
	reflect.NewAt(target.Type(), unsafe.Pointer(target.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

func TestListRendersRowsAndEmptyState(t *testing.T) {
	rows := []string{"a", "b", "c"}
	diagnosticCalls := 0
	node := List(ListProps[string]{
		ID:         "list-a",
		Items:      rows,
		Height:     120,
		RowHeight:  24,
		Overscan:   2,
		Class:      "outer",
		InnerClass: "inner",
		ItemKey:    func(item string) string { return item },
		RenderRow: func(props RowRenderProps[string]) ui.Node {
			return html.Div(html.Props{Class: "row"}, html.Text(props.Key))
		},
		OnViewportChange: func(ViewportDiagnostics) {
			diagnosticCalls++
		},
	})

	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("render list rows: %v", err)
	}
	if !strings.Contains(markup, "outer") || !strings.Contains(markup, "inner") {
		t.Fatalf("expected outer and inner classes in rendered markup")
	}
	if !strings.Contains(markup, ">a<") || !strings.Contains(markup, ">b<") {
		t.Fatalf("expected row markup content in rendered output")
	}
	if diagnosticCalls != 0 {
		t.Fatalf("expected no viewport diagnostics in native SSR path, got %d", diagnosticCalls)
	}

	accessibleMarkup, err := ui.RenderToString(List(ListProps[string]{
		OuterProps: html.Props{
			ID:    "list-accessible",
			Role:  "listbox",
			Class: "outer-props",
			Aria: map[string]string{
				"label":            "Virtualized queue",
				"activedescendant": "queue-row-b",
			},
			Data: map[string]string{"lane": "keyboard"},
			Raw:  map[string]interface{}{"tabindex": 0},
		},
		Items:     rows,
		Height:    120,
		RowHeight: 24,
		Class:     "outer",
		ItemKey:   func(item string) string { return item },
		RenderRow: func(props RowRenderProps[string]) ui.Node {
			return html.Div(html.Props{ID: "queue-row-" + props.Key, Role: "option"}, html.Text(props.Key))
		},
	}))
	if err != nil {
		t.Fatalf("render list with outer props: %v", err)
	}
	for _, snippet := range []string{
		`id="list-accessible"`,
		`role="listbox"`,
		`tabindex="0"`,
		`aria-label="Virtualized queue"`,
		`aria-activedescendant="queue-row-b"`,
		`data-lane="keyboard"`,
		`outer-props outer`,
	} {
		if !strings.Contains(accessibleMarkup, snippet) {
			t.Fatalf("expected %q in accessible list markup: %s", snippet, accessibleMarkup)
		}
	}

	emptyMarkup, err := ui.RenderToString(List(ListProps[string]{
		ID:        "list-empty",
		Items:     nil,
		Height:    120,
		RowHeight: 24,
		ItemKey:   func(item string) string { return item },
		RenderRow: func(RowRenderProps[string]) ui.Node { return html.Div(html.Props{}, html.Text("row")) },
		Empty:     html.Div(html.Props{Class: "empty"}, html.Text("No rows")),
	}))
	if err != nil {
		t.Fatalf("render list empty: %v", err)
	}
	if !strings.Contains(emptyMarkup, "No rows") {
		t.Fatalf("expected empty-state markup in output")
	}
}

func TestListHelpersAndRestorationSnapshot(t *testing.T) {
	assertPanics := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("%s: expected panic", name)
			}
		}()
		fn()
	}

	assertPanics("missing id", func() {
		validateListProps(ListProps[string]{
			Height:    100,
			RowHeight: 20,
			ItemKey:   func(item string) string { return item },
			RenderRow: func(RowRenderProps[string]) ui.Node { return nil },
		})
	})
	validateListProps(ListProps[string]{
		OuterProps: html.Props{ID: "outer-id"},
		Height:     100,
		RowHeight:  20,
		ItemKey:    func(item string) string { return item },
		RenderRow:  func(RowRenderProps[string]) ui.Node { return nil },
	})
	assertPanics("missing item key", func() {
		validateListProps(ListProps[string]{
			ID:        "x",
			Height:    100,
			RowHeight: 20,
			RenderRow: func(RowRenderProps[string]) ui.Node { return nil },
		})
	})
	assertPanics("missing height", func() {
		validateListProps(ListProps[string]{
			ID:       "x",
			Height:   0,
			RowHeight: 20,
			ItemKey:  func(item string) string { return item },
			RenderRow: func(RowRenderProps[string]) ui.Node {
				return nil
			},
		})
	})
	assertPanics("missing row height", func() {
		validateListProps(ListProps[string]{
			ID:       "x",
			Height:   100,
			RowHeight: 0,
			ItemKey:  func(item string) string { return item },
			RenderRow: func(RowRenderProps[string]) ui.Node {
				return nil
			},
		})
	})
	assertPanics("missing render row", func() {
		validateListProps(ListProps[string]{
			ID:       "x",
			Height:   100,
			RowHeight: 20,
			ItemKey:  func(item string) string { return item },
		})
	})

	r := clampRange(Range{Start: -3, End: 20}, 5)
	if r.Start != 0 || r.End != 5 {
		t.Fatalf("unexpected clamped range: %+v", r)
	}
	r = clampRange(Range{Start: 4, End: 1}, 5)
	if r.Start != 4 || r.End != 4 {
		t.Fatalf("expected end clamp to start, got %+v", r)
	}

	if px(12.4) != "12px" {
		t.Fatalf("unexpected px output")
	}
	merged := mergeStyle(map[string]string{"a": "1"}, map[string]string{"b": "2"})
	if merged["a"] != "1" || merged["b"] != "2" {
		t.Fatalf("unexpected merged style: %#v", merged)
	}
	if mergeStyle(nil, nil) != nil {
		t.Fatalf("expected nil mergeStyle for nil inputs")
	}

	keys, index := collectItemKeys([]string{"aa", "bb"}, func(v string) string { return v })
	if len(keys) != 2 || index["bb"] != 1 {
		t.Fatalf("unexpected key collection: keys=%v index=%v", keys, index)
	}
	if keySignature(keys) == "" {
		t.Fatalf("expected non-empty key signature")
	}
	if got := resolveListID(ListProps[string]{OuterProps: html.Props{ID: "outer-id"}}); got != "outer-id" {
		t.Fatalf("expected outer props id, got %q", got)
	}
	if got := resolveListID(ListProps[string]{ID: "explicit", OuterProps: html.Props{ID: "outer-id"}}); got != "explicit" {
		t.Fatalf("expected explicit id to win, got %q", got)
	}
	outer := buildListOuterProps(ListProps[string]{
		OuterProps: html.Props{
			ID:    "outer-id",
			Class: "keyboard",
			Style: map[string]string{"outline": "none"},
			Role:  "listbox",
			Raw:   map[string]interface{}{"tabindex": 0},
		},
		Class:  "viewport",
		Height: 140,
		Style:  map[string]string{"border": "1px solid"},
	}, "resolved-id")
	if outer.ID != "resolved-id" || outer.Role != "listbox" {
		t.Fatalf("unexpected outer props: %+v", outer)
	}
	if outer.Class != "keyboard viewport" {
		t.Fatalf("expected merged class names, got %q", outer.Class)
	}
	if outer.Raw["tabindex"] != 0 {
		t.Fatalf("expected raw tabindex to survive merge, got %#v", outer.Raw)
	}
	if outer.Style["height"] != "140px" || outer.Style["overflowY"] != "auto" || outer.Style["outline"] != "none" || outer.Style["border"] != "1px solid" {
		t.Fatalf("unexpected merged style: %#v", outer.Style)
	}
	if mergeClassNames("", "one", " two ", "three") != "one two three" {
		t.Fatalf("expected merged class names helper to trim blanks")
	}

	restorationStore.snapshots = map[string]restorationSnapshot{}
	storeRestorationSnapshot("", restorationSnapshot{ScrollTop: 1})
	if len(restorationStore.snapshots) != 0 {
		t.Fatalf("expected empty snapshot store id to be ignored, got %+v", restorationStore.snapshots)
	}
	storeRestorationSnapshot("demo", restorationSnapshot{ScrollTop: 44, AnchorKey: "bb"})
	loaded, ok := loadRestorationSnapshot("demo")
	if !ok || loaded.ScrollTop != 44 || loaded.AnchorKey != "bb" {
		t.Fatalf("unexpected loaded restoration snapshot: %+v ok=%t", loaded, ok)
	}
	if _, ok := loadRestorationSnapshot("missing"); ok {
		t.Fatal("expected missing restoration snapshot lookup to miss")
	}
	if _, ok := loadRestorationSnapshot(""); ok {
		t.Fatalf("expected empty id restoration lookup miss")
	}
	// Exercise no-op persistence branches in native builds.
	persistRestorationSnapshot("", restorationSnapshot{})
	persistRestorationSnapshot("demo", restorationSnapshot{ScrollTop: 50})
	_, _ = loadPersistedRestorationSnapshot("demo")
	_, _ = loadPersistedRestorationSnapshot("")
}

func TestRenderRowsHelperHandlesEmptyAndPopulatedRanges(t *testing.T) {
	rows := renderRows(Range{Start: 0, End: 0}, Range{Start: 0, End: 0}, []string{"a"}, 20, func(item string) string {
		return item
	}, func(RowRenderProps[string]) ui.Node {
		return html.Div(html.Props{}, html.Text("row"))
	}, nil, nil)
	if len(rows) != 0 {
		t.Fatalf("expected empty rendered range to produce no rows, got %d", len(rows))
	}

	rows = renderRows(Range{Start: 0, End: 2}, Range{Start: 0, End: 1}, []string{"a", "b"}, 20, func(item string) string {
		return "key-" + item
	}, func(props RowRenderProps[string]) ui.Node {
		return html.Div(html.Props{}, html.Text(props.Key))
	}, nil, nil)
	markup, err := ui.RenderToString(html.Div(html.Props{}, rows...))
	if err != nil {
		t.Fatalf("render helper rows markup: %v", err)
	}
	if !strings.Contains(markup, "key-a") || !strings.Contains(markup, "key-b") {
		t.Fatalf("expected helper rows markup to include rendered keys, got %q", markup)
	}
}

func TestRestoreElementScrollTopAndObserveOwnedViewport(t *testing.T) {
	// In native tests interop.Element has no browser bindings, so this path
	// exercises the scroll-metrics error branch.
	err := restoreElementScrollTop(interop.Element{}, restorationSnapshot{ScrollTop: 48}, map[string]int{}, 24, 20, 200)
	if err == nil {
		t.Fatalf("expected scroll metrics error in native interop build")
	}

	config := ViewportConfig{TotalItems: 10, RowHeight: 20, Overscan: 1}
	sub, err := ObserveOwnedViewport(interop.Element{}, config, func(ViewportState) {})
	if err == nil {
		t.Fatalf("expected unavailable native observation error")
	}
	called := false
	sub.cancel = func() { called = true }
	sub.Cancel()
	if !called {
		t.Fatalf("expected subscription cancel callback to run")
	}

	_, err = ObserveOwnedViewport(interop.Element{}, ViewportConfig{TotalItems: -1, RowHeight: 20}, nil)
	if err == nil {
		t.Fatalf("expected invalid config error")
	}
}

func TestRestoreElementScrollTopAnchorsClampsAndNoOps(t *testing.T) {
	element := interop.Element{}
	currentScrollTop := 40.0
	applied := make([]float64, 0, 4)
	setInteropElementField(t, &element, "scrollMetrics", func() (float64, float64, float64, error) {
		return currentScrollTop, 500, 120, nil
	})
	setInteropElementField(t, &element, "setScrollTop", func(value float64) error {
		currentScrollTop = value
		applied = append(applied, value)
		return nil
	})

	// Negative offsets clamp to zero.
	if err := restoreElementScrollTop(element, restorationSnapshot{ScrollTop: -20}, map[string]int{}, 20, 20, 120); err != nil {
		t.Fatalf("restoreElementScrollTop negative clamp: %v", err)
	}
	if len(applied) != 1 || applied[0] != 0 {
		t.Fatalf("expected one applied scrollTop of 0, got %v", applied)
	}

	// Anchor key overrides scrollTop and clamps to max scroll range.
	keyIndex := map[string]int{"item-19": 19}
	if err := restoreElementScrollTop(element, restorationSnapshot{ScrollTop: 5, AnchorKey: "item-19"}, keyIndex, 20, 20, 120); err != nil {
		t.Fatalf("restoreElementScrollTop anchor clamp: %v", err)
	}
	maxScroll := float64(20)*20 - 120 // 280
	if len(applied) != 2 || applied[1] != maxScroll {
		t.Fatalf("expected anchor restore to apply max scroll %v, got %v", maxScroll, applied)
	}

	// Near-equal current and target scrollTop should no-op.
	currentScrollTop = maxScroll + 0.2
	if err := restoreElementScrollTop(element, restorationSnapshot{ScrollTop: maxScroll}, keyIndex, 20, 20, 120); err != nil {
		t.Fatalf("restoreElementScrollTop noop threshold: %v", err)
	}
	if len(applied) != 2 {
		t.Fatalf("expected no additional setScrollTop call when within epsilon, got %v", applied)
	}
}

func TestRestoreElementScrollTopSetErrorIsReturned(t *testing.T) {
	element := interop.Element{}
	setInteropElementField(t, &element, "scrollMetrics", func() (float64, float64, float64, error) {
		return 0, 300, 100, nil
	})
	setInteropElementField(t, &element, "setScrollTop", func(float64) error {
		return errors.New("set failed")
	})

	err := restoreElementScrollTop(element, restorationSnapshot{ScrollTop: 200}, map[string]int{}, 20, 20, 100)
	if err == nil || !strings.Contains(err.Error(), "set failed") {
		t.Fatalf("expected setScrollTop error to be returned, got %v", err)
	}
}
