package ui

import (
	"fmt"
	"reflect"
	goRuntime "runtime"
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

var componentHandleCache sync.Map
var getComponentIdentityCache sync.Map
var getComponentRenderCache sync.Map

type componentIdentityCacheKey struct {
	getType    reflect.Type
	getPointer uintptr
}

type componentIdentityCacheValue struct {
	getPrettyName    string
	getQualifiedName string
}

// getComponentHandle is a core package helper.
func getComponentHandle(parseComponent any) *runtime.ComponentType {
	parsePrettyName, parseQualifiedName := describeComponentIdentity(parseComponent)
	parseIdentity := parseQualifiedName
	if strings.TrimSpace(parseIdentity) == "" {
		parseIdentity = parsePrettyName
	}
	if strings.TrimSpace(parseIdentity) == "" {
		parseIdentity = reflect.TypeOf(parseComponent).String()
	}

	if parseCached, parseOk := componentHandleCache.Load(parseIdentity); parseOk {
		handle := parseCached.(*runtime.ComponentType)
		handle.SetImplementationRenderer(parseComponent, buildComponentRenderer(parseComponent))
		return handle
	}

	handle := runtime.NewComponentType(parseIdentity, parsePrettyName, parseQualifiedName, parseComponent, buildComponentRenderer(parseComponent))
	parseStored, _ := componentHandleCache.LoadOrStore(parseIdentity, handle)
	parseResolved := parseStored.(*runtime.ComponentType)
	parseResolved.SetImplementationRenderer(parseComponent, buildComponentRenderer(parseComponent))
	return parseResolved
}

// buildComponentRenderer prepares one reusable renderer closure for a component implementation signature.
func buildComponentRenderer(parseComponent any) func(any, map[string]any) *runtime.Element {
	if parseComponent == nil {
		return nil
	}

	switch parseComponent.(type) {
	case func() Node:
		return func(parseImplementation any, parseRawProps map[string]any) *runtime.Element {
			parseTypedImplementation, parseOk := parseImplementation.(func() Node)
			if !parseOk {
				return renderComponent(parseImplementation, parseRawProps)
			}
			return parseTypedImplementation()
		}
	case func(map[string]any) Node:
		return func(parseImplementation any, parseRawProps map[string]any) *runtime.Element {
			parseTypedImplementation, parseOk := parseImplementation.(func(map[string]any) Node)
			if !parseOk {
				return renderComponent(parseImplementation, parseRawProps)
			}
			return parseTypedImplementation(getComponentMapProps(parseRawProps))
		}
	case func(runtime.Attrs) Node:
		return func(parseImplementation any, parseRawProps map[string]any) *runtime.Element {
			parseTypedImplementation, parseOk := parseImplementation.(func(runtime.Attrs) Node)
			if !parseOk {
				return renderComponent(parseImplementation, parseRawProps)
			}
			return parseTypedImplementation(getComponentAttrsProps(parseRawProps))
		}
	}

	parseComponentType := reflect.TypeOf(parseComponent)
	if parseComponentType == nil || parseComponentType.Kind() != reflect.Func {
		return func(parseImplementation any, parseRawProps map[string]any) *runtime.Element {
			return renderComponent(parseImplementation, parseRawProps)
		}
	}
	if parseCached, parseOk := getComponentRenderCache.Load(parseComponentType); parseOk {
		return parseCached.(func(any, map[string]any) *runtime.Element)
	}

	parseMeta := getComponentMeta(parseComponentType)
	parseRenderer := func(parseImplementation any, parseRawProps map[string]any) *runtime.Element {
		parseImplementationValue := reflect.ValueOf(parseImplementation)
		if !parseImplementationValue.IsValid() || parseImplementationValue.Kind() != reflect.Func {
			panic(actionableCreateElementPanic("ui.CreateElement requires a component function or ui.Node"))
		}

		var parseResults []reflect.Value
		if parseMeta.hasArg {
			var parseArgBuf [1]reflect.Value
			parseArgBuf[0] = parseMeta.getArgValue(parseRawProps)
			parseResults = parseImplementationValue.Call(parseArgBuf[:])
		} else {
			parseResults = parseImplementationValue.Call(nil)
		}

		if len(parseResults) == 0 || !parseResults[0].IsValid() || parseResults[0].IsNil() {
			return nil
		}

		parseElement, _ := parseResults[0].Interface().(*runtime.Element)
		return parseElement
	}

	parseStored, _ := getComponentRenderCache.LoadOrStore(parseComponentType, parseRenderer)
	return parseStored.(func(any, map[string]any) *runtime.Element)
}

// describeComponentIdentity is a core package helper.
func describeComponentIdentity(parseComponent any) (string, string) {
	if parseComponent == nil {
		return "", ""
	}

	parseValue := reflect.ValueOf(parseComponent)
	if parseValue.IsValid() && parseValue.Kind() == reflect.Func {
		getCacheKey := componentIdentityCacheKey{
			getType:    parseValue.Type(),
			getPointer: parseValue.Pointer(),
		}
		if parseCached, hasParseCached := getComponentIdentityCache.Load(getCacheKey); hasParseCached {
			getCached := parseCached.(componentIdentityCacheValue)
			return getCached.getPrettyName, getCached.getQualifiedName
		}
		if parseFn := goRuntime.FuncForPC(parseValue.Pointer()); parseFn != nil {
			parseQualified := parseFn.Name()
			getIdentity := componentIdentityCacheValue{
				getPrettyName:    trimComponentName(parseQualified),
				getQualifiedName: parseQualified,
			}
			getComponentIdentityCache.Store(getCacheKey, getIdentity)
			return getIdentity.getPrettyName, getIdentity.getQualifiedName
		}
		parseQualified := fmt.Sprintf("%s@%x", reflect.TypeOf(parseComponent).String(), parseValue.Pointer())
		getIdentity := componentIdentityCacheValue{
			getPrettyName:    reflect.TypeOf(parseComponent).String(),
			getQualifiedName: parseQualified,
		}
		getComponentIdentityCache.Store(getCacheKey, getIdentity)
		return getIdentity.getPrettyName, getIdentity.getQualifiedName
	}

	parseRendered := reflect.TypeOf(parseComponent).String()
	return parseRendered, parseRendered
}

// trimComponentName is a core package helper.
func trimComponentName(parseName string) string {
	if parseIndex := strings.LastIndex(parseName, "/"); parseIndex >= 0 {
		parseName = parseName[parseIndex+1:]
	}
	if parseIndex2 := strings.LastIndex(parseName, "."); parseIndex2 >= 0 {
		parseName = parseName[parseIndex2+1:]
	}
	return parseName
}
