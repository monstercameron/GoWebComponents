package virtualization

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
)

// RowRenderProps describes the data exposed to one rendered row callback.
type RowRenderProps[T any] struct {
	Index         int
	Item          T
	Key           string
	VisibleRange  Range
	RenderedRange Range
}

// ListProps describes the first public fixed-height virtualized list surface.
type ListProps[T any] struct {
	OuterProps       html.Props
	ID               string
	Items            []T
	Height           float64
	RowHeight        float64
	Overscan         int
	Class            string
	Style            map[string]string
	ItemKey          func(T) string
	RenderRow        func(RowRenderProps[T]) ui.Node
	OnViewportChange func(ViewportDiagnostics)
	Empty            ui.Node
	InnerClass       string
}

type rowLifecycleCounts struct {
	Mounts   int
	Unmounts int
}

type rowBodyProps[T any] struct {
	Row       RowRenderProps[T]
	Render    func(RowRenderProps[T]) ui.Node
	OnMount   func()
	OnUnmount func()
}

type restorationSnapshot struct {
	ScrollTop float64
	AnchorKey string
}

type restorationStoreState struct {
	mu        sync.Mutex
	snapshots map[string]restorationSnapshot
}

var restorationStore = restorationStoreState{
	snapshots: map[string]restorationSnapshot{},
}

const restorationStoragePrefix = "gwc:virtualization:restore:"

// List renders one fixed-height vertical virtualized list with an owned scroll
// container.
func List[T any](props ListProps[T]) ui.Node {
	validateListProps(props)
	listID := resolveListID(props)

	config := ViewportConfig{
		TotalItems: len(props.Items),
		RowHeight:  props.RowHeight,
		Overscan:   props.Overscan,
	}
	initialState, err := ComputeViewportState(config, 0, props.Height)
	if err != nil {
		panic(err)
	}
	keys, keyIndex := collectItemKeys(props.Items, props.ItemKey)
	viewport := ui.UseState(initialState)
	lifecycleCounts := ui.UseRef(rowLifecycleCounts{})
	keysRef := ui.UseRef(keys)
	keyIndexRef := ui.UseRef(keyIndex)
	keysRef.Set(keys)
	keyIndexRef.Set(keyIndex)
	restoreRef := ui.UseRef(restorationSnapshot{})
	publishDiagnostics := func(state ViewportState) {
		if props.OnViewportChange == nil {
			return
		}
		counts := lifecycleCounts.Get()
		props.OnViewportChange(state.Diagnostics().WithRowLifecycle(counts.Mounts, counts.Unmounts))
	}

	ui.UseEffect(func() func() {
		document, err := interop.GetDocument()
		if err != nil {
			return nil
		}
		element, ok, err := document.ElementByID(listID)
		if err != nil || !ok {
			return nil
		}
		if snapshot, ok := loadRestorationSnapshot(listID); ok {
			restoreRef.Set(snapshot)
			_ = restoreElementScrollTop(element, snapshot, keyIndexRef.Get(), props.RowHeight, len(props.Items), props.Height)
		}
		sub, err := ObserveOwnedViewport(element, config, func(next ViewportState) {
			snapshot := restorationSnapshot{ScrollTop: next.ScrollTop}
			anchorKeys := keysRef.Get()
			if next.Visible.Start >= 0 && next.Visible.Start < len(anchorKeys) {
				snapshot.AnchorKey = anchorKeys[next.Visible.Start]
			}
			restoreRef.Set(snapshot)
			storeRestorationSnapshot(listID, snapshot)
			persistRestorationSnapshot(listID, snapshot)
			viewport.Set(next)
			publishDiagnostics(next)
		})
		if err != nil {
			return nil
		}
		return func() {
			storeRestorationSnapshot(listID, restoreRef.Get())
			persistRestorationSnapshot(listID, restoreRef.Get())
			sub.Cancel()
		}
	}, listID, len(props.Items), props.Height, props.RowHeight, props.Overscan)

	ui.UseEffect(func() func() {
		document, err := interop.GetDocument()
		if err != nil {
			return nil
		}
		element, ok, err := document.ElementByID(listID)
		if err != nil || !ok {
			return nil
		}
		snapshot, ok := loadRestorationSnapshot(listID)
		if !ok {
			return nil
		}
		restoreRef.Set(snapshot)
		_ = restoreElementScrollTop(element, snapshot, keyIndex, props.RowHeight, len(props.Items), props.Height)
		return nil
	}, listID, props.Height, props.RowHeight, keySignature(keys))

	state := viewport.Get()
	rendered := clampRange(state.Rendered, len(props.Items))
	content := renderRows(rendered, state.Visible, props.Items, props.RowHeight, props.ItemKey, props.RenderRow, func() {
		counts := lifecycleCounts.Get()
		counts.Mounts++
		lifecycleCounts.Set(counts)
		publishDiagnostics(viewport.Get())
	}, func() {
		counts := lifecycleCounts.Get()
		counts.Unmounts++
		lifecycleCounts.Set(counts)
		publishDiagnostics(viewport.Get())
	})

	if len(props.Items) == 0 && props.Empty != nil {
		content = []ui.Node{props.Empty}
	}

	innerChildren := make([]ui.Node, 0, len(content)+2)
	if rendered.Start > 0 {
		innerChildren = append(innerChildren, html.Div(html.Props{
			Style: map[string]string{"height": px(float64(rendered.Start) * props.RowHeight)},
			Aria:  map[string]string{"hidden": "true"},
		}))
	}
	innerChildren = append(innerChildren, content...)
	if rendered.End < len(props.Items) {
		innerChildren = append(innerChildren, html.Div(html.Props{
			Style: map[string]string{"height": px(float64(len(props.Items)-rendered.End) * props.RowHeight)},
			Aria:  map[string]string{"hidden": "true"},
		}))
	}

	return html.Div(buildListOuterProps(props, listID), html.Div(html.Props{
		Class: props.InnerClass,
		Style: map[string]string{
			"height":        px(state.TotalHeight),
			"boxSizing":     "border-box",
			"overflow":      "hidden",
			"position":      "relative",
			"paddingTop":    "0",
			"paddingBottom": "0",
		},
	}, innerChildren...))
}

func validateListProps[T any](props ListProps[T]) {
	if resolveListID(props) == "" {
		panic("virtualization.List requires a non-empty ID")
	}
	if props.Height <= 0 {
		panic("virtualization.List requires Height > 0")
	}
	if props.RowHeight <= 0 {
		panic("virtualization.List requires RowHeight > 0")
	}
	if props.ItemKey == nil {
		panic("virtualization.List requires ItemKey")
	}
	if props.RenderRow == nil {
		panic("virtualization.List requires RenderRow")
	}
}

func resolveListID[T any](props ListProps[T]) string {
	if id := strings.TrimSpace(props.ID); id != "" {
		return id
	}
	return strings.TrimSpace(props.OuterProps.ID)
}

func buildListOuterProps[T any](props ListProps[T], listID string) html.Props {
	outer := props.OuterProps
	outer.ID = listID
	outer.Class = mergeClassNames(outer.Class, props.Class)
	outer.Style = mergeStyle(
		mergeStyle(map[string]string{
			"height":    px(props.Height),
			"overflowY": "auto",
		}, outer.Style),
		props.Style,
	)
	return outer
}

func mergeClassNames(values ...string) string {
	classes := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			classes = append(classes, trimmed)
		}
	}
	return strings.Join(classes, " ")
}

func clampRange(r Range, total int) Range {
	start := clampIndex(r.Start, total)
	end := clampIndex(r.End, total)
	if end < start {
		end = start
	}
	return Range{Start: start, End: end}
}

func renderRows[T any](rendered, visible Range, items []T, rowHeight float64, itemKey func(T) string, renderRow func(RowRenderProps[T]) ui.Node, onMount func(), onUnmount func()) []ui.Node {
	children := make([]ui.Node, 0, rendered.Len())
	for index := rendered.Start; index < rendered.End; index++ {
		indexValue := index
		itemValue := items[index]
		keyValue := itemKey(itemValue)
		rowProps := RowRenderProps[T]{
			Index:         indexValue,
			Item:          itemValue,
			Key:           keyValue,
			VisibleRange:  visible,
			RenderedRange: rendered,
		}
		children = append(children, html.Div(html.Props{
			Key: keyValue,
			Style: map[string]string{
				"height":    px(rowHeight),
				"boxSizing": "border-box",
			},
		}, ui.CreateElement(func(props rowBodyProps[T]) ui.Node {
			ui.UseEffect(func() func() {
				if props.OnMount != nil {
					props.OnMount()
				}
				return func() {
					if props.OnUnmount != nil {
						props.OnUnmount()
					}
				}
			}, props.Row.Key)
			return props.Render(props.Row)
		}, rowBodyProps[T]{
			Row:       rowProps,
			Render:    renderRow,
			OnMount:   onMount,
			OnUnmount: onUnmount,
		})))
	}
	return children
}

func mergeStyle(base, extra map[string]string) map[string]string {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	merged := make(map[string]string, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func px(value float64) string {
	return fmt.Sprintf("%.0fpx", value)
}

func collectItemKeys[T any](items []T, itemKey func(T) string) ([]string, map[string]int) {
	keys := make([]string, 0, len(items))
	index := make(map[string]int, len(items))
	for itemIndex, item := range items {
		key := itemKey(item)
		keys = append(keys, key)
		index[key] = itemIndex
	}
	return keys, index
}

func keySignature(keys []string) string {
	return fmt.Sprintf("%q", keys)
}

func storeRestorationSnapshot(id string, snapshot restorationSnapshot) {
	if id == "" {
		return
	}
	restorationStore.mu.Lock()
	defer restorationStore.mu.Unlock()
	restorationStore.snapshots[id] = snapshot
}

func loadRestorationSnapshot(id string) (restorationSnapshot, bool) {
	if id == "" {
		return restorationSnapshot{}, false
	}
	restorationStore.mu.Lock()
	defer restorationStore.mu.Unlock()
	snapshot, ok := restorationStore.snapshots[id]
	if ok {
		return snapshot, true
	}
	browserSnapshot, browserOK := loadPersistedRestorationSnapshot(id)
	if browserOK {
		restorationStore.snapshots[id] = browserSnapshot
		return browserSnapshot, true
	}
	return restorationSnapshot{}, false
}

func persistRestorationSnapshot(id string, snapshot restorationSnapshot) {
	if id == "" {
		return
	}
	storage, err := interop.GetSessionStorage()
	if err != nil {
		return
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	_ = storage.SetItem(restorationStoragePrefix+id, string(payload))
}

func loadPersistedRestorationSnapshot(id string) (restorationSnapshot, bool) {
	if id == "" {
		return restorationSnapshot{}, false
	}
	storage, err := interop.GetSessionStorage()
	if err != nil {
		return restorationSnapshot{}, false
	}
	raw, ok, err := storage.GetItem(restorationStoragePrefix + id)
	if err != nil || !ok || raw == "" {
		return restorationSnapshot{}, false
	}
	var snapshot restorationSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return restorationSnapshot{}, false
	}
	return snapshot, ok
}

func restoreElementScrollTop(element interop.Element, snapshot restorationSnapshot, keyIndex map[string]int, rowHeight float64, totalItems int, viewportHeight float64) error {
	target := snapshot.ScrollTop
	if snapshot.AnchorKey != "" {
		if anchorIndex, ok := keyIndex[snapshot.AnchorKey]; ok {
			target = float64(anchorIndex) * rowHeight
		}
	}
	maxScrollTop := math.Max(float64(totalItems)*rowHeight-viewportHeight, 0)
	if target < 0 {
		target = 0
	}
	if target > maxScrollTop {
		target = maxScrollTop
	}
	currentScrollTop, _, _, err := element.ScrollMetrics()
	if err != nil {
		return err
	}
	if math.Abs(currentScrollTop-target) < 0.5 {
		return nil
	}
	return element.SetScrollTop(target)
}
