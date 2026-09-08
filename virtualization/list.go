package virtualization

import (
	"fmt"
	"maps"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/interop"
	"github.com/monstercameron/GoWebComponents/v6/ui"
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
	ScrollTop float64 `json:"scrollTop"`
	AnchorKey string  `json:"anchorKey,omitempty"`
}

// maxRestorationSnapshots bounds the process-global scroll-restoration store. The
// store must outlive a list's unmount (that is the whole point — restore scroll
// after navigating away and back), so entries can't be dropped on unmount; without
// a cap, an app with dynamically-keyed lists (e.g. "list-user-<id>" across a long
// session) would grow the map without bound. A snapshot is tiny (a float64 + a
// short key), so a generous LRU cap covers realistic navigation while bounding
// memory: the least-recently-stored snapshot is evicted past the cap.
const maxRestorationSnapshots = 256

type restorationStoreState struct {
	mu        sync.Mutex
	snapshots map[string]restorationSnapshot
	// order tracks store recency (least-recent first) for LRU eviction.
	order []string
}

var restorationStore = restorationStoreState{
	snapshots: map[string]restorationSnapshot{},
}

// touchAndEvictLocked records parseId as most-recently-stored and evicts the
// oldest snapshots once the store exceeds maxRestorationSnapshots. Caller holds mu.
func (parseStore *restorationStoreState) touchAndEvictLocked(parseId string) {
	for parseIndex, parseExisting := range parseStore.order {
		if parseExisting == parseId {
			parseStore.order = append(parseStore.order[:parseIndex], parseStore.order[parseIndex+1:]...)
			break
		}
	}
	parseStore.order = append(parseStore.order, parseId)
	for len(parseStore.order) > maxRestorationSnapshots {
		parseEvict := parseStore.order[0]
		parseStore.order = parseStore.order[1:]
		delete(parseStore.snapshots, parseEvict)
	}
}

// List renders one fixed-height vertical virtualized list with an owned scroll
// container.
func List[T any](parseProps ListProps[T]) ui.Node {
	validateListProps(parseProps)
	parseListID := resolveListID(parseProps)

	parseConfig := ViewportConfig{
		TotalItems: len(parseProps.Items),
		RowHeight:  parseProps.RowHeight,
		Overscan:   parseProps.Overscan,
	}
	parseInitialState, parseErr := ComputeViewportState(parseConfig, 0, parseProps.Height)
	if parseErr != nil {
		panic(parseErr)
	}
	// Rebuild the O(N) item-key bundle only when the item set changes (a new
	// slice identity), not on scroll-driven re-renders.
	parseKeyBundle := ui.UseMemoOf(func(itemsIdentity) itemKeysBundle {
		parseK, parseKI := collectItemKeys(parseProps.Items, parseProps.ItemKey)
		return itemKeysBundle{keys: parseK, index: parseKI, signature: keySignature(parseK)}
	}, listItemsIdentity(parseProps.Items))
	parseKeys := parseKeyBundle.keys
	parseKeyIndex := parseKeyBundle.index
	parseViewport := ui.UseState(parseInitialState)
	parseLifecycleCounts := ui.UseRef(rowLifecycleCounts{})
	parseKeysRef := ui.UseRef(parseKeys)
	parseKeyIndexRef := ui.UseRef(parseKeyIndex)
	parseKeysRef.Set(parseKeys)
	parseKeyIndexRef.Set(parseKeyIndex)
	parseRestoreRef := ui.UseRef(restorationSnapshot{})
	parsePublishDiagnostics := func(parseState2 ViewportState) {
		if parseProps.OnViewportChange == nil {
			return
		}
		parseCounts := parseLifecycleCounts.Get()
		parseProps.OnViewportChange(parseState2.Diagnostics().WithRowLifecycle(parseCounts.Mounts, parseCounts.Unmounts))
	}

	ui.UseEffect(buildListViewportEffect(
		parseListID,
		len(parseProps.Items),
		parseProps.Height,
		parseProps.RowHeight,
		parseConfig,
		parseKeysRef,
		parseKeyIndexRef,
		parseRestoreRef,
		parseViewport,
		parsePublishDiagnostics,
	), parseListID, len(parseProps.Items), parseProps.Height, parseProps.RowHeight, parseProps.Overscan)

	ui.UseEffect(buildListRestorationEffect(
		parseListID,
		len(parseProps.Items),
		parseProps.Height,
		parseProps.RowHeight,
		parseKeyIndex,
		parseRestoreRef,
	), parseListID, parseProps.Height, parseProps.RowHeight, parseKeyBundle.signature)

	parseState := parseViewport.Get()
	parseRendered := clampRange(parseState.Rendered, len(parseProps.Items))
	parseContent := renderRows(parseRendered, parseState.Visible, parseProps.Items, parseProps.RowHeight, parseProps.ItemKey, parseProps.RenderRow, func() {
		parseCounts2 := parseLifecycleCounts.Get()
		parseCounts2.Mounts++
		parseLifecycleCounts.Set(parseCounts2)
		parsePublishDiagnostics(parseViewport.Get())
	}, func() {
		parseCounts3 := parseLifecycleCounts.Get()
		parseCounts3.Unmounts++
		parseLifecycleCounts.Set(parseCounts3)
		parsePublishDiagnostics(parseViewport.Get())
	})

	if len(parseProps.Items) == 0 && parseProps.Empty != nil {
		parseContent = []ui.Node{parseProps.Empty}
	}

	parseInnerChildren := make([]ui.Node, 0, len(parseContent)+2)
	if parseRendered.Start > 0 {
		parseInnerChildren = append(parseInnerChildren, html.Div(html.Props{
			Style: map[string]string{"height": px(float64(parseRendered.Start) * parseProps.RowHeight)},
			Aria:  map[string]string{"hidden": "true"},
		}))
	}
	parseInnerChildren = append(parseInnerChildren, parseContent...)
	if parseRendered.End < len(parseProps.Items) {
		parseInnerChildren = append(parseInnerChildren, html.Div(html.Props{
			Style: map[string]string{"height": px(float64(len(parseProps.Items)-parseRendered.End) * parseProps.RowHeight)},
			Aria:  map[string]string{"hidden": "true"},
		}))
	}

	return html.Div(buildListOuterProps(parseProps, parseListID), html.Div(html.Props{
		Class: parseProps.InnerClass,
		Style: map[string]string{
			"height":        px(parseState.TotalHeight),
			"boxSizing":     "border-box",
			"overflow":      "hidden",
			"position":      "relative",
			"paddingTop":    "0",
			"paddingBottom": "0",
		},
	}, parseInnerChildren...))
}

func validateListProps[T any](parseProps ListProps[T]) {
	if resolveListID(parseProps) == "" {
		panic("virtualization.List requires a non-empty ID")
	}
	if parseProps.Height <= 0 {
		panic("virtualization.List requires Height > 0")
	}
	if parseProps.RowHeight <= 0 {
		panic("virtualization.List requires RowHeight > 0")
	}
	if parseProps.ItemKey == nil {
		panic("virtualization.List requires ItemKey")
	}
	if parseProps.RenderRow == nil {
		panic("virtualization.List requires RenderRow")
	}
}

func resolveListID[T any](parseProps ListProps[T]) string {
	if parseId := strings.TrimSpace(parseProps.ID); parseId != "" {
		return parseId
	}
	return strings.TrimSpace(parseProps.OuterProps.ID)
}

func buildListOuterProps[T any](parseProps ListProps[T], parseListID string) html.Props {
	parseOuter := parseProps.OuterProps
	parseOuter.ID = parseListID
	parseOuter.Class = mergeClassNames(parseOuter.Class, parseProps.Class)
	parseOuter.Style = mergeStyle(
		mergeStyle(map[string]string{
			"height":    px(parseProps.Height),
			"overflowY": "auto",
		}, parseOuter.Style),
		parseProps.Style,
	)
	return parseOuter
}

func mergeClassNames(parseValues ...string) string {
	parseClasses := make([]string, 0, len(parseValues))
	for _, parseValue := range parseValues {
		parseTrimmed := strings.TrimSpace(parseValue)
		if parseTrimmed != "" {
			parseClasses = append(parseClasses, parseTrimmed)
		}
	}
	return strings.Join(parseClasses, " ")
}

func clampRange(parseR Range, parseTotal int) Range {
	parseStart := clampIndex(parseR.Start, parseTotal)
	parseEnd := max(clampIndex(parseR.End, parseTotal), parseStart)
	return Range{Start: parseStart, End: parseEnd}
}

// rowBodyComponent renders one row body and ties its mount/unmount callbacks
// to the row's effect lifecycle.
func rowBodyComponent[T any](parseProps rowBodyProps[T]) ui.Node {
	ui.UseEffect(func() func() {
		if parseProps.OnMount != nil {
			parseProps.OnMount()
		}
		return func() {
			if parseProps.OnUnmount != nil {
				parseProps.OnUnmount()
			}
		}
	}, parseProps.Row.Key)
	return parseProps.Render(parseProps.Row)
}

func renderRows[T any](parseRendered, parseVisible Range, parseItems []T, parseRowHeight float64, parseItemKey func(T) string, renderRow func(RowRenderProps[T]) ui.Node, parseOnMount func(), parseOnUnmount func()) []ui.Node {
	parseChildren := make([]ui.Node, 0, parseRendered.Len())
	for parseIndex := parseRendered.Start; parseIndex < parseRendered.End; parseIndex++ {
		parseIndexValue := parseIndex
		parseItemValue := parseItems[parseIndex]
		parseKeyValue := parseItemKey(parseItemValue)
		parseRowProps := RowRenderProps[T]{
			Index:         parseIndexValue,
			Item:          parseItemValue,
			Key:           parseKeyValue,
			VisibleRange:  parseVisible,
			RenderedRange: parseRendered,
		}
		parseChildren = append(parseChildren, html.Div(html.Props{
			Key: parseKeyValue,
			Style: map[string]string{
				"height":    px(parseRowHeight),
				"boxSizing": "border-box",
			},
		}, ui.CreateElement(rowBodyComponent[T], rowBodyProps[T]{
			Row:       parseRowProps,
			Render:    renderRow,
			OnMount:   parseOnMount,
			OnUnmount: parseOnUnmount,
		})))
	}
	return parseChildren
}

func mergeStyle(parseBase, parseExtra map[string]string) map[string]string {
	if len(parseBase) == 0 && len(parseExtra) == 0 {
		return nil
	}
	parseMerged := make(map[string]string, len(parseBase)+len(parseExtra))
	maps.Copy(parseMerged, parseBase)
	maps.Copy(parseMerged, parseExtra)
	return parseMerged
}

func px(parseValue float64) string {
	return fmt.Sprintf("%.0fpx", parseValue)
}

func collectItemKeys[T any](parseItems []T, parseItemKey func(T) string) ([]string, map[string]int) {
	parseKeys := make([]string, 0, len(parseItems))
	parseIndex := make(map[string]int, len(parseItems))
	for parseItemIndex, parseItem := range parseItems {
		parseKey := parseItemKey(parseItem)
		parseKeys = append(parseKeys, parseKey)
		parseIndex[parseKey] = parseItemIndex
	}
	return parseKeys, parseIndex
}

// itemKeysBundle memoizes the per-item key slice, key->index map, and the effect
// signature so the O(N) key walk (collectItemKeys + keySignature) runs once per
// item-set change instead of once per render — a render happens on every scroll
// frame, so recomputing them there made scrolling O(N) per frame and defeated
// virtualization.
type itemKeysBundle struct {
	keys      []string
	index     map[string]int
	signature string
}

// itemsIdentity is a cheap O(1) identity for an items slice: the backing-array
// pointer plus length. It is comparable, so it can key ui.UseMemoOf. A caller
// that follows the immutable-update idiom (a new slice when items change) gets a
// new identity and a rebuild; in-place mutation of a same-length slice keeps the
// identity and would reuse the cached keys, so items must be treated as immutable.
type itemsIdentity struct {
	ptr uintptr
	len int
}

func listItemsIdentity[T any](parseItems []T) itemsIdentity {
	if len(parseItems) == 0 {
		return itemsIdentity{}
	}
	return itemsIdentity{ptr: reflect.ValueOf(parseItems).Pointer(), len: len(parseItems)}
}

// keySignature returns a cheap change-detection signature of the item keys, used
// as a UseEffect dependency. It replaces a fmt.Sprintf("%q", keys) that allocated
// a full quoted copy of every key on every render (and a render happens on every
// scroll frame) — for a large list that was a per-frame multi-hundred-KB alloc.
// This FNV-1a fold over the key bytes allocates only the small final string.
func keySignature(parseKeys []string) string {
	const parseOffset = uint64(14695981039346656037)
	const parsePrime = uint64(1099511628211)
	parseHash := parseOffset
	for _, parseKey := range parseKeys {
		for parseI := 0; parseI < len(parseKey); parseI++ {
			parseHash ^= uint64(parseKey[parseI])
			parseHash *= parsePrime
		}
		// Separator byte so ["ab","c"] and ["a","bc"] hash differently.
		parseHash ^= 0x1f
		parseHash *= parsePrime
	}
	return strconv.FormatUint(parseHash, 16) + ":" + strconv.Itoa(len(parseKeys))
}

func storeRestorationSnapshot(parseId string, parseSnapshot restorationSnapshot) {
	if parseId == "" {
		return
	}
	restorationStore.mu.Lock()
	defer restorationStore.mu.Unlock()
	restorationStore.snapshots[parseId] = parseSnapshot
	restorationStore.touchAndEvictLocked(parseId)
}

func loadRestorationSnapshot(parseId string) (restorationSnapshot, bool) {
	if parseId == "" {
		return restorationSnapshot{}, false
	}
	restorationStore.mu.Lock()
	defer restorationStore.mu.Unlock()
	parseSnapshot, parseOk := restorationStore.snapshots[parseId]
	if parseOk {
		return parseSnapshot, true
	}
	parseBrowserSnapshot, parseBrowserOK := loadPersistedRestorationSnapshot(parseId)
	if parseBrowserOK {
		restorationStore.snapshots[parseId] = parseBrowserSnapshot
		restorationStore.touchAndEvictLocked(parseId)
		return parseBrowserSnapshot, true
	}
	return restorationSnapshot{}, false
}

func restoreElementScrollTop(parseElement interop.Element, parseSnapshot restorationSnapshot, parseKeyIndex map[string]int, parseRowHeight float64, parseTotalItems int, parseViewportHeight float64) error {
	parseTarget := parseSnapshot.ScrollTop
	if parseSnapshot.AnchorKey != "" {
		if parseAnchorIndex, parseOk := parseKeyIndex[parseSnapshot.AnchorKey]; parseOk {
			parseTarget = float64(parseAnchorIndex) * parseRowHeight
		}
	}
	parseMaxScrollTop := math.Max(float64(parseTotalItems)*parseRowHeight-parseViewportHeight, 0)
	if parseTarget < 0 {
		parseTarget = 0
	}
	if parseTarget > parseMaxScrollTop {
		parseTarget = parseMaxScrollTop
	}
	parseCurrentScrollTop, _, _, parseErr := parseElement.ScrollMetrics()
	if parseErr != nil {
		return parseErr
	}
	if math.Abs(parseCurrentScrollTop-parseTarget) < 0.5 {
		return nil
	}
	return parseElement.SetScrollTop(parseTarget)
}
