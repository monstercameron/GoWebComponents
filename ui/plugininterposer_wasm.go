//go:build js && wasm

package ui

import (
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"strconv"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/pluginruntime"
)

func init() {
	_ = pluginruntime.RegisterBuiltinService(pluginruntime.ServiceRegistration{
		Key:   pluginruntime.ServiceKeyDOM,
		Value: buildUIDOMService{},
	})
	_ = pluginruntime.RegisterBuiltinService(pluginruntime.ServiceRegistration{
		Key:   pluginruntime.ServiceKeyStyle,
		Value: buildUIStyleService{},
	})
	_ = pluginruntime.RegisterBuiltinService(pluginruntime.ServiceRegistration{
		Key:   pluginruntime.ServiceKeyEvents,
		Value: buildUIEventService{},
	})
}

type buildUIDOMService struct{}

type buildUIStyleService struct{}

type buildUIStylePatchHandle struct {
	getRoot     js.Value
	getPrevious map[string]string
	getDidExist map[string]bool
}

type buildUIEventService struct{}

var storeUIEventState struct {
	getMu        sync.Mutex
	hasStarted   bool
	getListeners []js.Func
	getRecords   []pluginruntime.EventRecord
}

const maxUIEventRecords = 128

// GetDOMSnapshot returns one browser DOM snapshot rooted at document.body.
func (buildUIDOMService) GetDOMSnapshot(parseBudget pluginruntime.QueryBudget) (pluginruntime.DOMSnapshot, error) {
	getBody := getUIDocumentBody()
	if !getBody.Truthy() {
		return pluginruntime.DOMSnapshot{Meta: pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDRuntime1, false)}, nil
	}
	buildLimit := parseBudget.MaxItems
	if buildLimit <= 0 {
		buildLimit = 500
	}
	buildCount := 0
	isBuildTruncated := false
	getRoot := buildUIDOMNodeSnapshot(getBody, "0", &buildCount, buildLimit, &isBuildTruncated)
	return pluginruntime.DOMSnapshot{
		Meta: pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDRuntime1, isBuildTruncated),
		Root: getRoot,
	}, nil
}

// HighlightNode applies a visible outline to one resolved DOM node.
func (buildUIDOMService) HighlightNode(parseNodeID string, parseOptions pluginruntime.CommandOptions) error {
	getNode := getUIDOMNodeByID(parseNodeID)
	if !getNode.Truthy() {
		return nil
	}
	getStyle := getNode.Get("style")
	if getStyle.Truthy() {
		getStyle.Call("setProperty", "outline", "2px solid #67e8f9")
		getStyle.Call("setProperty", "outline-offset", "2px")
	}
	return nil
}

// ScrollNodeIntoView scrolls one resolved DOM node into view.
func (buildUIDOMService) ScrollNodeIntoView(parseNodeID string, parseOptions pluginruntime.CommandOptions) error {
	getNode := getUIDOMNodeByID(parseNodeID)
	if !getNode.Truthy() || !getNode.Get("scrollIntoView").Truthy() {
		return nil
	}
	getNode.Call("scrollIntoView", map[string]any{"block": "center", "inline": "nearest"})
	return nil
}

// GetStyleSnapshot returns one browser style snapshot for the document root.
func (buildUIStyleService) GetStyleSnapshot(parseBudget pluginruntime.QueryBudget) (pluginruntime.StyleSnapshot, error) {
	getRoot := getUIDocumentRoot()
	buildSnapshot := pluginruntime.StyleSnapshot{
		Meta:      pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDRuntime1, false),
		Variables: map[string]string{},
	}
	if !getRoot.Truthy() {
		return buildSnapshot, nil
	}
	getStyle := getRoot.Get("style")
	if getStyle.Truthy() {
		getLength := getStyle.Get("length").Int()
		for parseIndex := 0; parseIndex < getLength; parseIndex++ {
			getName := getStyle.Call("item", parseIndex).String()
			if !strings.HasPrefix(strings.TrimSpace(getName), "--") {
				continue
			}
			buildSnapshot.Variables[getName] = strings.TrimSpace(getStyle.Call("getPropertyValue", getName).String())
		}
	}
	getDocument := getUIDocument()
	if getDocument.Truthy() {
		getSheets := getDocument.Get("styleSheets")
		getLength := getSheets.Length()
		for parseIndex := 0; parseIndex < getLength; parseIndex++ {
			getSheet := getSheets.Index(parseIndex)
			getHref := strings.TrimSpace(getSheet.Get("href").String())
			if getHref == "" {
				getHref = "inline"
			}
			buildSnapshot.Stylesheet = append(buildSnapshot.Stylesheet, getHref)
		}
	}
	return buildSnapshot, nil
}

// ApplyStyleVariables applies one reversible set of CSS custom properties at document root.
func (buildUIStyleService) ApplyStyleVariables(parseVariables map[string]string, parseOptions pluginruntime.CommandOptions) (pluginruntime.StylePatchHandle, error) {
	getRoot := getUIDocumentRoot()
	if !getRoot.Truthy() {
		return buildUIStylePatchHandle{}, nil
	}
	getStyle := getRoot.Get("style")
	buildHandle := buildUIStylePatchHandle{
		getRoot:     getRoot,
		getPrevious: map[string]string{},
		getDidExist: map[string]bool{},
	}
	for parseName, parseValue := range parseVariables {
		getTrimmed := strings.TrimSpace(parseName)
		if !strings.HasPrefix(getTrimmed, "--") {
			continue
		}
		getCurrent := strings.TrimSpace(getStyle.Call("getPropertyValue", getTrimmed).String())
		buildHandle.getPrevious[getTrimmed] = getCurrent
		buildHandle.getDidExist[getTrimmed] = getCurrent != ""
		getStyle.Call("setProperty", getTrimmed, parseValue)
	}
	return buildHandle, nil
}

// RemoveStylePatch restores the previous document-root CSS custom properties.
func (parseHandle buildUIStylePatchHandle) RemoveStylePatch() error {
	if !parseHandle.getRoot.Truthy() {
		return nil
	}
	getStyle := parseHandle.getRoot.Get("style")
	for parseName, parseValue := range parseHandle.getPrevious {
		if parseHandle.getDidExist[parseName] {
			getStyle.Call("setProperty", parseName, parseValue)
			continue
		}
		getStyle.Call("removeProperty", parseName)
	}
	return nil
}

// GetEventSnapshot returns one bounded browser event-ring snapshot.
func (buildUIEventService) GetEventSnapshot(parseBudget pluginruntime.QueryBudget) (pluginruntime.EventSnapshot, error) {
	ensureUIEventListeners()
	storeUIEventState.getMu.Lock()
	defer storeUIEventState.getMu.Unlock()
	buildEvents := append([]pluginruntime.EventRecord(nil), storeUIEventState.getRecords...)
	if parseBudget.MaxItems > 0 && len(buildEvents) > parseBudget.MaxItems {
		buildEvents = append([]pluginruntime.EventRecord(nil), buildEvents[len(buildEvents)-parseBudget.MaxItems:]...)
	}
	return pluginruntime.EventSnapshot{
		Meta:   pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDRuntime1, false),
		Events: buildEvents,
	}, nil
}

// getUIDocument returns the current browser document value.
func getUIDocument() js.Value {
	return js.Global().Get("document")
}

// getUIDocumentBody returns the current browser document body value.
func getUIDocumentBody() js.Value {
	getDocument := getUIDocument()
	if !getDocument.Truthy() {
		return js.Null()
	}
	return getDocument.Get("body")
}

// getUIDocumentRoot returns the current browser document element value.
func getUIDocumentRoot() js.Value {
	getDocument := getUIDocument()
	if !getDocument.Truthy() {
		return js.Null()
	}
	return getDocument.Get("documentElement")
}

// buildUIDOMNodeSnapshot builds one DOM node snapshot recursively.
func buildUIDOMNodeSnapshot(parseNode js.Value, parseID string, parseCount *int, parseLimit int, isParseTruncated *bool) *pluginruntime.DOMNodeSnapshot {
	if !parseNode.Truthy() {
		return nil
	}
	if parseCount != nil {
		if *parseCount >= parseLimit {
			*isParseTruncated = true
			return nil
		}
		*parseCount++
	}
	buildSnapshot := &pluginruntime.DOMNodeSnapshot{
		ID:         parseID,
		Tag:        strings.ToLower(strings.TrimSpace(parseNode.Get("nodeName").String())),
		Text:       strings.TrimSpace(parseNode.Get("textContent").String()),
		Attributes: map[string]string{},
	}
	getAttributes := parseNode.Get("attributes")
	if getAttributes.Truthy() {
		getLength := getAttributes.Length()
		for parseIndex := 0; parseIndex < getLength; parseIndex++ {
			getAttr := getAttributes.Index(parseIndex)
			buildSnapshot.Attributes[getAttr.Get("name").String()] = getAttr.Get("value").String()
		}
	}
	getChildren := parseNode.Get("childNodes")
	if getChildren.Truthy() {
		getLength := getChildren.Length()
		for parseIndex := 0; parseIndex < getLength; parseIndex++ {
			getChild := getChildren.Index(parseIndex)
			buildChildID := parseID + "." + strconv.Itoa(parseIndex)
			getChildSnapshot := buildUIDOMNodeSnapshot(getChild, buildChildID, parseCount, parseLimit, isParseTruncated)
			if getChildSnapshot != nil {
				buildSnapshot.Children = append(buildSnapshot.Children, *getChildSnapshot)
			}
		}
	}
	return buildSnapshot
}

// getUIDOMNodeByID resolves one DOM node from the current document body using a path ID.
func getUIDOMNodeByID(parseID string) js.Value {
	getBody := getUIDocumentBody()
	if !getBody.Truthy() {
		return js.Null()
	}
	if parseID == "" || parseID == "0" {
		return getBody
	}
	buildParts := strings.Split(strings.TrimPrefix(parseID, "0."), ".")
	getCurrent := getBody
	for _, parsePart := range buildParts {
		if strings.TrimSpace(parsePart) == "" {
			continue
		}
		getIndex, parseErr := strconv.Atoi(parsePart)
		if parseErr != nil {
			return js.Null()
		}
		getChildren := getCurrent.Get("childNodes")
		if !getChildren.Truthy() || getChildren.Length() <= getIndex {
			return js.Null()
		}
		getCurrent = getChildren.Index(getIndex)
	}
	return getCurrent
}

// ensureUIEventListeners installs one bounded browser event ring on first use.
func ensureUIEventListeners() {
	storeUIEventState.getMu.Lock()
	defer storeUIEventState.getMu.Unlock()
	if storeUIEventState.hasStarted {
		return
	}
	getDocument := getUIDocument()
	if !getDocument.Truthy() || !getDocument.Get("addEventListener").Truthy() {
		return
	}
	for _, parseType := range []string{"click", "input", "change", "submit", "keydown"} {
		buildType := parseType
		getListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
			defer runtime.RecoverContainedPanic("ui", "ensureUIEventListeners callback")
			if len(parseArgs) == 0 {
				return nil
			}
			getEvent := parseArgs[0]
			getTarget := getEvent.Get("target")
			buildTarget := strings.ToLower(strings.TrimSpace(getTarget.Get("nodeName").String()))
			if getID := strings.TrimSpace(getTarget.Get("id").String()); getID != "" {
				buildTarget += "#" + getID
			}
			storeUIEventState.getMu.Lock()
			defer storeUIEventState.getMu.Unlock()
			storeUIEventState.getRecords = append(storeUIEventState.getRecords, pluginruntime.EventRecord{
				Type:       buildType,
				Phase:      "capture",
				Target:     buildTarget,
				Current:    "document",
				OccurredAt: time.Now().UTC(),
			})
			if len(storeUIEventState.getRecords) > maxUIEventRecords {
				storeUIEventState.getRecords = append([]pluginruntime.EventRecord(nil), storeUIEventState.getRecords[len(storeUIEventState.getRecords)-maxUIEventRecords:]...)
			}
			return nil
		})
		getDocument.Call("addEventListener", buildType, getListener, true)
		storeUIEventState.getListeners = append(storeUIEventState.getListeners, getListener)
	}
	storeUIEventState.hasStarted = true
}
