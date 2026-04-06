//go:build !js || !wasm
// +build !js !wasm

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

// buildListTestNode renders List through a component wrapper so list tests stay within a valid hook context on both native and wasm targets.
func buildListTestNode[T any](parseProps ListProps[T]) ui.Node {
	return ui.CreateElement(func() ui.Node {
		return List(parseProps)
	})
}

func setInteropElementField(parseT *testing.T, parseElement *interop.Element, parseField string, parseValue interface{}) {
	parseT.Helper()
	parseStructValue := reflect.ValueOf(parseElement).Elem()
	parseTarget := parseStructValue.FieldByName(parseField)
	if !parseTarget.IsValid() {
		parseT.Fatalf("missing interop.Element field %q", parseField)
	}
	reflect.NewAt(parseTarget.Type(), unsafe.Pointer(parseTarget.UnsafeAddr())).Elem().Set(reflect.ValueOf(parseValue))
}

func TestListRendersRowsAndEmptyState(parseT *testing.T) {
	parseRows := []string{"a", "b", "c"}
	parseDiagnosticCalls := 0
	parseNode := buildListTestNode(ListProps[string]{
		ID:         "list-a",
		Items:      parseRows,
		Height:     120,
		RowHeight:  24,
		Overscan:   2,
		Class:      "outer",
		InnerClass: "inner",
		ItemKey:    func(parseItem string) string { return parseItem },
		RenderRow: func(parseProps RowRenderProps[string]) ui.Node {
			return html.Div(html.Props{Class: "row"}, html.Text(parseProps.Key))
		},
		OnViewportChange: func(ViewportDiagnostics) {
			parseDiagnosticCalls++
		},
	})

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("render list rows: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "outer") || !strings.Contains(parseMarkup, "inner") {
		parseT.Fatalf("expected outer and inner classes in rendered markup")
	}
	if !strings.Contains(parseMarkup, ">a<") || !strings.Contains(parseMarkup, ">b<") {
		parseT.Fatalf("expected row markup content in rendered output")
	}
	if parseDiagnosticCalls != 0 {
		parseT.Fatalf("expected no viewport diagnostics in native SSR path, got %d", parseDiagnosticCalls)
	}

	parseAccessibleMarkup, parseErr := ui.RenderToString(buildListTestNode(ListProps[string]{
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
		Items:     parseRows,
		Height:    120,
		RowHeight: 24,
		Class:     "outer",
		ItemKey:   func(parseItem2 string) string { return parseItem2 },
		RenderRow: func(parseProps2 RowRenderProps[string]) ui.Node {
			return html.Div(html.Props{ID: "queue-row-" + parseProps2.Key, Role: "option"}, html.Text(parseProps2.Key))
		},
	}))
	if parseErr != nil {
		parseT.Fatalf("render list with outer props: %v", parseErr)
	}
	for _, parseSnippet := range []string{
		`id="list-accessible"`,
		`role="listbox"`,
		`tabindex="0"`,
		`aria-label="Virtualized queue"`,
		`aria-activedescendant="queue-row-b"`,
		`data-lane="keyboard"`,
		`outer-props outer`,
	} {
		if !strings.Contains(parseAccessibleMarkup, parseSnippet) {
			parseT.Fatalf("expected %q in accessible list markup: %s", parseSnippet, parseAccessibleMarkup)
		}
	}

	parseEmptyMarkup, parseErr := ui.RenderToString(buildListTestNode(ListProps[string]{
		ID:        "list-empty",
		Items:     nil,
		Height:    120,
		RowHeight: 24,
		ItemKey:   func(parseItem3 string) string { return parseItem3 },
		RenderRow: func(RowRenderProps[string]) ui.Node { return html.Div(html.Props{}, html.Text("row")) },
		Empty:     html.Div(html.Props{Class: "empty"}, html.Text("No rows")),
	}))
	if parseErr != nil {
		parseT.Fatalf("render list empty: %v", parseErr)
	}
	if !strings.Contains(parseEmptyMarkup, "No rows") {
		parseT.Fatalf("expected empty-state markup in output")
	}
}

func TestListHelpersAndRestorationSnapshot(parseT *testing.T) {
	parseAssertPanics := func(parseName string, parseFn func()) {
		parseT.Helper()
		defer func() {
			if recover() == nil {
				parseT.Fatalf("%s: expected panic", parseName)
			}
		}()
		parseFn()
	}

	parseAssertPanics("missing id", func() {
		validateListProps(ListProps[string]{
			Height:    100,
			RowHeight: 20,
			ItemKey:   func(parseItem string) string { return parseItem },
			RenderRow: func(RowRenderProps[string]) ui.Node { return nil },
		})
	})
	validateListProps(ListProps[string]{
		OuterProps: html.Props{ID: "outer-id"},
		Height:     100,
		RowHeight:  20,
		ItemKey:    func(parseItem2 string) string { return parseItem2 },
		RenderRow:  func(RowRenderProps[string]) ui.Node { return nil },
	})
	parseAssertPanics("missing item key", func() {
		validateListProps(ListProps[string]{
			ID:        "x",
			Height:    100,
			RowHeight: 20,
			RenderRow: func(RowRenderProps[string]) ui.Node { return nil },
		})
	})
	parseAssertPanics("missing height", func() {
		validateListProps(ListProps[string]{
			ID:        "x",
			Height:    0,
			RowHeight: 20,
			ItemKey:   func(parseItem3 string) string { return parseItem3 },
			RenderRow: func(RowRenderProps[string]) ui.Node {
				return nil
			},
		})
	})
	parseAssertPanics("missing row height", func() {
		validateListProps(ListProps[string]{
			ID:        "x",
			Height:    100,
			RowHeight: 0,
			ItemKey:   func(parseItem4 string) string { return parseItem4 },
			RenderRow: func(RowRenderProps[string]) ui.Node {
				return nil
			},
		})
	})
	parseAssertPanics("missing render row", func() {
		validateListProps(ListProps[string]{
			ID:        "x",
			Height:    100,
			RowHeight: 20,
			ItemKey:   func(parseItem5 string) string { return parseItem5 },
		})
	})
	if _, parseErr := ui.RenderToString(buildListTestNode(ListProps[string]{
		ID:        "x",
		Items:     []string{"a"},
		Height:    100,
		RowHeight: 20,
		Overscan:  -1,
		ItemKey:   func(parseItem6 string) string { return parseItem6 },
		RenderRow: func(RowRenderProps[string]) ui.Node { return nil },
	})); parseErr == nil || !strings.Contains(parseErr.Error(), "overscan") {
		parseT.Fatalf("expected overscan render error, got %v", parseErr)
	}

	parseR := clampRange(Range{Start: -3, End: 20}, 5)
	if parseR.Start != 0 || parseR.End != 5 {
		parseT.Fatalf("unexpected clamped range: %+v", parseR)
	}
	parseR = clampRange(Range{Start: 4, End: 1}, 5)
	if parseR.Start != 4 || parseR.End != 4 {
		parseT.Fatalf("expected end clamp to start, got %+v", parseR)
	}

	if px(12.4) != "12px" {
		parseT.Fatalf("unexpected px output")
	}
	parseMerged := mergeStyle(map[string]string{"a": "1"}, map[string]string{"b": "2"})
	if parseMerged["a"] != "1" || parseMerged["b"] != "2" {
		parseT.Fatalf("unexpected merged style: %#v", parseMerged)
	}
	if mergeStyle(nil, nil) != nil {
		parseT.Fatalf("expected nil mergeStyle for nil inputs")
	}

	parseKeys, parseIndex := collectItemKeys([]string{"aa", "bb"}, func(parseV string) string { return parseV })
	if len(parseKeys) != 2 || parseIndex["bb"] != 1 {
		parseT.Fatalf("unexpected key collection: keys=%v index=%v", parseKeys, parseIndex)
	}
	if keySignature(parseKeys) == "" {
		parseT.Fatalf("expected non-empty key signature")
	}
	if parseGot := resolveListID(ListProps[string]{OuterProps: html.Props{ID: "outer-id"}}); parseGot != "outer-id" {
		parseT.Fatalf("expected outer props id, got %q", parseGot)
	}
	if parseGot2 := resolveListID(ListProps[string]{ID: "explicit", OuterProps: html.Props{ID: "outer-id"}}); parseGot2 != "explicit" {
		parseT.Fatalf("expected explicit id to win, got %q", parseGot2)
	}
	parseOuter := buildListOuterProps(ListProps[string]{
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
	if parseOuter.ID != "resolved-id" || parseOuter.Role != "listbox" {
		parseT.Fatalf("unexpected outer props: %+v", parseOuter)
	}
	if parseOuter.Class != "keyboard viewport" {
		parseT.Fatalf("expected merged class names, got %q", parseOuter.Class)
	}
	if parseOuter.Raw["tabindex"] != 0 {
		parseT.Fatalf("expected raw tabindex to survive merge, got %#v", parseOuter.Raw)
	}
	if parseOuter.Style["height"] != "140px" || parseOuter.Style["overflowY"] != "auto" || parseOuter.Style["outline"] != "none" || parseOuter.Style["border"] != "1px solid" {
		parseT.Fatalf("unexpected merged style: %#v", parseOuter.Style)
	}
	if mergeClassNames("", "one", " two ", "three") != "one two three" {
		parseT.Fatalf("expected merged class names helper to trim blanks")
	}

	restorationStore.snapshots = map[string]restorationSnapshot{}
	storeRestorationSnapshot("", restorationSnapshot{ScrollTop: 1})
	if len(restorationStore.snapshots) != 0 {
		parseT.Fatalf("expected empty snapshot store id to be ignored, got %+v", restorationStore.snapshots)
	}
	storeRestorationSnapshot("demo", restorationSnapshot{ScrollTop: 44, AnchorKey: "bb"})
	parseLoaded, parseOk := loadRestorationSnapshot("demo")
	if !parseOk || parseLoaded.ScrollTop != 44 || parseLoaded.AnchorKey != "bb" {
		parseT.Fatalf("unexpected loaded restoration snapshot: %+v ok=%t", parseLoaded, parseOk)
	}
	if _, parseOk2 := loadRestorationSnapshot("missing"); parseOk2 {
		parseT.Fatal("expected missing restoration snapshot lookup to miss")
	}
	if _, parseOk3 := loadRestorationSnapshot(""); parseOk3 {
		parseT.Fatalf("expected empty id restoration lookup miss")
	}
	// Exercise no-op persistence branches in native builds.
	persistRestorationSnapshot("", restorationSnapshot{})
	persistRestorationSnapshot("demo", restorationSnapshot{ScrollTop: 50})
	_, _ = loadPersistedRestorationSnapshot("demo")
	_, _ = loadPersistedRestorationSnapshot("")
}

func TestRenderRowsHelperHandlesEmptyAndPopulatedRanges(parseT *testing.T) {
	parseRows := renderRows(Range{Start: 0, End: 0}, Range{Start: 0, End: 0}, []string{"a"}, 20, func(parseItem string) string {
		return parseItem
	}, func(RowRenderProps[string]) ui.Node {
		return html.Div(html.Props{}, html.Text("row"))
	}, nil, nil)
	if len(parseRows) != 0 {
		parseT.Fatalf("expected empty rendered range to produce no rows, got %d", len(parseRows))
	}

	parseMarkup, parseErr := ui.RenderToString(ui.CreateElement(func() ui.Node {
		parseRows2 := renderRows(Range{Start: 0, End: 2}, Range{Start: 0, End: 1}, []string{"a", "b"}, 20, func(parseItem2 string) string {
			return "key-" + parseItem2
		}, func(parseProps RowRenderProps[string]) ui.Node {
			return html.Div(html.Props{}, html.Text(parseProps.Key))
		}, nil, nil)
		return html.Div(html.Props{}, parseRows2...)
	}))
	if parseErr != nil {
		parseT.Fatalf("render helper rows markup: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "key-a") || !strings.Contains(parseMarkup, "key-b") {
		parseT.Fatalf("expected helper rows markup to include rendered keys, got %q", parseMarkup)
	}
}

func TestRestoreElementScrollTopAndObserveOwnedViewport(parseT *testing.T) {
	// In native tests interop.Element has no browser bindings, so this path
	// exercises the scroll-metrics error branch.
	parseErr := restoreElementScrollTop(interop.Element{}, restorationSnapshot{ScrollTop: 48}, map[string]int{}, 24, 20, 200)
	if parseErr == nil {
		parseT.Fatalf("expected scroll metrics error in native interop build")
	}

	parseConfig := ViewportConfig{TotalItems: 10, RowHeight: 20, Overscan: 1}
	parseSub, parseErr := ObserveOwnedViewport(interop.Element{}, parseConfig, func(ViewportState) {})
	if parseErr == nil {
		parseT.Fatalf("expected unavailable native observation error")
	}
	isParseCalled := false
	parseSub.cancel = func() { isParseCalled = true }
	parseSub.Cancel()
	if !isParseCalled {
		parseT.Fatalf("expected subscription cancel callback to run")
	}

	_, parseErr = ObserveOwnedViewport(interop.Element{}, ViewportConfig{TotalItems: -1, RowHeight: 20}, nil)
	if parseErr == nil {
		parseT.Fatalf("expected invalid config error")
	}
}

func TestRestoreElementScrollTopAnchorsClampsAndNoOps(parseT *testing.T) {
	parseElement := interop.Element{}
	parseCurrentScrollTop := 40.0
	parseApplied := make([]float64, 0, 4)
	setInteropElementField(parseT, &parseElement, "scrollMetrics", func() (float64, float64, float64, error) {
		return parseCurrentScrollTop, 500, 120, nil
	})
	setInteropElementField(parseT, &parseElement, "setScrollTop", func(parseValue float64) error {
		parseCurrentScrollTop = parseValue
		parseApplied = append(parseApplied, parseValue)
		return nil
	})

	// Negative offsets clamp to zero.
	if parseErr := restoreElementScrollTop(parseElement, restorationSnapshot{ScrollTop: -20}, map[string]int{}, 20, 20, 120); parseErr != nil {
		parseT.Fatalf("restoreElementScrollTop negative clamp: %v", parseErr)
	}
	if len(parseApplied) != 1 || parseApplied[0] != 0 {
		parseT.Fatalf("expected one applied scrollTop of 0, got %v", parseApplied)
	}

	// Anchor key overrides scrollTop and clamps to max scroll range.
	parseKeyIndex := map[string]int{"item-19": 19}
	if parseErr2 := restoreElementScrollTop(parseElement, restorationSnapshot{ScrollTop: 5, AnchorKey: "item-19"}, parseKeyIndex, 20, 20, 120); parseErr2 != nil {
		parseT.Fatalf("restoreElementScrollTop anchor clamp: %v", parseErr2)
	}
	parseMaxScroll := float64(20)*20 - 120 // 280
	if len(parseApplied) != 2 || parseApplied[1] != parseMaxScroll {
		parseT.Fatalf("expected anchor restore to apply max scroll %v, got %v", parseMaxScroll, parseApplied)
	}

	// Near-equal current and target scrollTop should no-op.
	parseCurrentScrollTop = parseMaxScroll + 0.2
	if parseErr3 := restoreElementScrollTop(parseElement, restorationSnapshot{ScrollTop: parseMaxScroll}, parseKeyIndex, 20, 20, 120); parseErr3 != nil {
		parseT.Fatalf("restoreElementScrollTop noop threshold: %v", parseErr3)
	}
	if len(parseApplied) != 2 {
		parseT.Fatalf("expected no additional setScrollTop call when within epsilon, got %v", parseApplied)
	}
}

func TestRestoreElementScrollTopSetErrorIsReturned(parseT *testing.T) {
	parseElement := interop.Element{}
	setInteropElementField(parseT, &parseElement, "scrollMetrics", func() (float64, float64, float64, error) {
		return 0, 300, 100, nil
	})
	setInteropElementField(parseT, &parseElement, "setScrollTop", func(float64) error {
		return errors.New("set failed")
	})

	parseErr := restoreElementScrollTop(parseElement, restorationSnapshot{ScrollTop: 200}, map[string]int{}, 20, 20, 100)
	if parseErr == nil || !strings.Contains(parseErr.Error(), "set failed") {
		parseT.Fatalf("expected setScrollTop error to be returned, got %v", parseErr)
	}
}
