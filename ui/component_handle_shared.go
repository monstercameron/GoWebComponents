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

func getComponentHandle(component interface{}) *runtime.ComponentType {
	prettyName, qualifiedName := describeComponentIdentity(component)
	identity := qualifiedName
	if strings.TrimSpace(identity) == "" {
		identity = prettyName
	}
	if strings.TrimSpace(identity) == "" {
		identity = reflect.TypeOf(component).String()
	}

	if cached, ok := componentHandleCache.Load(identity); ok {
		handle := cached.(*runtime.ComponentType)
		handle.SetImplementation(component)
		return handle
	}

	handle := runtime.NewComponentType(identity, prettyName, qualifiedName, component, func(implementation interface{}, rawProps map[string]interface{}) *runtime.Element {
		return renderComponent(implementation, rawProps)
	})
	stored, _ := componentHandleCache.LoadOrStore(identity, handle)
	resolved := stored.(*runtime.ComponentType)
	resolved.SetImplementation(component)
	return resolved
}

func describeComponentIdentity(component interface{}) (string, string) {
	if component == nil {
		return "", ""
	}

	value := reflect.ValueOf(component)
	if value.IsValid() && value.Kind() == reflect.Func {
		if fn := goRuntime.FuncForPC(value.Pointer()); fn != nil {
			qualified := fn.Name()
			return trimComponentName(qualified), qualified
		}
		return reflect.TypeOf(component).String(), fmt.Sprintf("%s@%x", reflect.TypeOf(component).String(), value.Pointer())
	}

	rendered := reflect.TypeOf(component).String()
	return rendered, rendered
}

func trimComponentName(name string) string {
	if index := strings.LastIndex(name, "/"); index >= 0 {
		name = name[index+1:]
	}
	if index := strings.LastIndex(name, "."); index >= 0 {
		name = name[index+1:]
	}
	return name
}
