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

// getComponentHandle is a core package helper.
func getComponentHandle(parseComponent interface{}) *runtime.ComponentType {
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
		handle.SetImplementation(parseComponent)
		return handle
	}

	handle := runtime.NewComponentType(parseIdentity, parsePrettyName, parseQualifiedName, parseComponent, func(parseImplementation interface{}, parseRawProps map[string]interface{}) *runtime.Element {
		return renderComponent(parseImplementation, parseRawProps)
	})
	parseStored, _ := componentHandleCache.LoadOrStore(parseIdentity, handle)
	parseResolved := parseStored.(*runtime.ComponentType)
	parseResolved.SetImplementation(parseComponent)
	return parseResolved
}

// describeComponentIdentity is a core package helper.
func describeComponentIdentity(parseComponent interface{}) (string, string) {
	if parseComponent == nil {
		return "", ""
	}

	parseValue := reflect.ValueOf(parseComponent)
	if parseValue.IsValid() && parseValue.Kind() == reflect.Func {
		if parseFn := goRuntime.FuncForPC(parseValue.Pointer()); parseFn != nil {
			parseQualified := parseFn.Name()
			return trimComponentName(parseQualified), parseQualified
		}
		return reflect.TypeOf(parseComponent).String(), fmt.Sprintf("%s@%x", reflect.TypeOf(parseComponent).String(), parseValue.Pointer())
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
