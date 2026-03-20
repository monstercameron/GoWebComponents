package runtime

import (
	"fmt"
	"reflect"
	goRuntime "runtime"
	"strings"
)

// ComponentSignature captures the dev-time identity and hook shape of a component.
type ComponentSignature struct {
	Kind          string
	Name          string
	QualifiedName string
	Key           string
	HookKinds     []string
}

// CompatibleWith reports whether two component signatures can safely preserve state.
func (signature ComponentSignature) CompatibleWith(other ComponentSignature) bool {
	if signature.Kind != other.Kind {
		return false
	}
	if signature.identityKey() != other.identityKey() {
		return false
	}
	if signature.Key != other.Key {
		return false
	}
	if len(signature.HookKinds) != len(other.HookKinds) {
		return false
	}
	for index, kind := range signature.HookKinds {
		if other.HookKinds[index] != kind {
			return false
		}
	}
	return true
}

// Summary renders a compact human-readable description of the signature.
func (signature ComponentSignature) Summary() string {
	label := signature.Name
	if strings.TrimSpace(label) == "" {
		label = signature.QualifiedName
	}
	if strings.TrimSpace(label) == "" {
		label = "unknown"
	}

	parts := []string{label}
	if strings.TrimSpace(signature.Key) != "" {
		parts = append(parts, "key="+signature.Key)
	}
	if len(signature.HookKinds) > 0 {
		parts = append(parts, "hooks="+strings.Join(signature.HookKinds, " > "))
	}
	return strings.Join(parts, " | ")
}

func (signature ComponentSignature) identityKey() string {
	if strings.TrimSpace(signature.QualifiedName) != "" {
		return signature.QualifiedName
	}
	return signature.Name
}

func recordHookSignature(hooks *Hooks, kind string) {
	if hooks == nil || strings.TrimSpace(kind) == "" {
		return
	}
	hooks.signature = append(hooks.signature, kind)
}

func buildComponentSignature(fiber *Fiber, hooks *Hooks) *ComponentSignature {
	if fiber == nil {
		return nil
	}

	kind, name := describeFiber(fiber)
	if kind != "component" {
		return nil
	}

	prettyName, qualifiedName := describeCallableIdentity(fiber.typeOf)
	if strings.TrimSpace(prettyName) == "" {
		prettyName = name
	}
	if strings.TrimSpace(qualifiedName) == "" {
		qualifiedName = prettyName
	}

	signature := &ComponentSignature{
		Kind:          kind,
		Name:          prettyName,
		QualifiedName: qualifiedName,
		Key:           describeFiberKey(fiber),
	}
	if hooks != nil && len(hooks.signature) > 0 {
		signature.HookKinds = append([]string(nil), hooks.signature...)
	}
	return signature
}

func describeCallableIdentity(value interface{}) (string, string) {
	rv := reflect.ValueOf(value)
	if rv.IsValid() && rv.Kind() == reflect.Func {
		if fn := goRuntime.FuncForPC(rv.Pointer()); fn != nil {
			qualified := fn.Name()
			return trimCallableName(qualified), qualified
		}
	}
	if value == nil {
		return "nil", ""
	}
	rendered := reflect.TypeOf(value).String()
	return rendered, rendered
}

func trimCallableName(name string) string {
	if index := strings.LastIndex(name, "/"); index >= 0 {
		name = name[index+1:]
	}
	if index := strings.LastIndex(name, "."); index >= 0 {
		name = name[index+1:]
	}
	return name
}

func describeFiberKey(fiber *Fiber) string {
	if fiber == nil || fiber.props == nil {
		return ""
	}

	key, ok := fiber.props["key"]
	if !ok || key == nil {
		return ""
	}
	return fmt.Sprint(key)
}
