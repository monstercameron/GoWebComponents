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
func (parseSignature ComponentSignature) CompatibleWith(parseOther ComponentSignature) bool {
	if parseSignature.Kind != parseOther.Kind {
		return false
	}
	if parseSignature.identityKey() != parseOther.identityKey() {
		return false
	}
	if parseSignature.Key != parseOther.Key {
		return false
	}
	if len(parseSignature.HookKinds) != len(parseOther.HookKinds) {
		return false
	}
	for parseIndex, parseKind := range parseSignature.HookKinds {
		if parseOther.HookKinds[parseIndex] != parseKind {
			return false
		}
	}
	return true
}

// Summary renders a compact human-readable description of the signature.
func (parseSignature ComponentSignature) Summary() string {
	parseLabel := parseSignature.Name
	if strings.TrimSpace(parseLabel) == "" {
		parseLabel = parseSignature.QualifiedName
	}
	if strings.TrimSpace(parseLabel) == "" {
		parseLabel = "unknown"
	}

	parseParts := []string{parseLabel}
	if strings.TrimSpace(parseSignature.Key) != "" {
		parseParts = append(parseParts, "key="+parseSignature.Key)
	}
	if len(parseSignature.HookKinds) > 0 {
		parseParts = append(parseParts, "hooks="+strings.Join(parseSignature.HookKinds, " > "))
	}
	return strings.Join(parseParts, " | ")
}

// identityKey is a core package helper.
func (parseSignature ComponentSignature) identityKey() string {
	if strings.TrimSpace(parseSignature.QualifiedName) != "" {
		return parseSignature.QualifiedName
	}
	return parseSignature.Name
}

// recordHookSignature is a core package helper.
func recordHookSignature(parseHooks *Hooks, parseKind string) {
	// Callers pass constant kind literals; no sanitization needed on this
	// per-hook-call hot path.
	if parseHooks == nil || parseKind == "" {
		return
	}
	parseHooks.signature = append(parseHooks.signature, parseKind)
}

// buildComponentSignature is a core package helper.
func buildComponentSignature(parseFiber *Fiber, parseHooks *Hooks) *ComponentSignature {
	if parseFiber == nil {
		return nil
	}

	parseKind, parseName := describeFiber(parseFiber)
	if parseKind != "component" {
		return nil
	}

	parsePrettyName, parseQualifiedName := describeCallableIdentity(parseFiber.typeOf)
	if strings.TrimSpace(parsePrettyName) == "" {
		parsePrettyName = parseName
	}
	if strings.TrimSpace(parseQualifiedName) == "" {
		parseQualifiedName = parsePrettyName
	}

	parseSignature := &ComponentSignature{
		Kind:          parseKind,
		Name:          parsePrettyName,
		QualifiedName: parseQualifiedName,
		Key:           describeFiberKey(parseFiber),
	}
	if parseHooks != nil && len(parseHooks.signature) > 0 {
		parseSignature.HookKinds = append([]string(nil), parseHooks.signature...)
	}
	return parseSignature
}

// describeCallableIdentity is a core package helper.
func describeCallableIdentity(parseValue any) (string, string) {
	if parseComponent, parseOk := parseValue.(*ComponentType); parseOk && parseComponent != nil {
		parsePretty := strings.TrimSpace(parseComponent.Name)
		parseQualified := strings.TrimSpace(parseComponent.IdentityKey())
		if parsePretty == "" {
			parsePretty = trimCallableName(parseQualified)
		}
		return parsePretty, parseQualified
	}

	parseRv := reflect.ValueOf(parseValue)
	if parseRv.IsValid() && parseRv.Kind() == reflect.Func {
		if parseFn := goRuntime.FuncForPC(parseRv.Pointer()); parseFn != nil {
			parseQualified2 := parseFn.Name()
			return trimCallableName(parseQualified2), parseQualified2
		}
	}
	if parseValue == nil {
		return "nil", ""
	}
	parseRendered := reflect.TypeOf(parseValue).String()
	return parseRendered, parseRendered
}

// trimCallableName is a core package helper.
func trimCallableName(parseName string) string {
	if parseIndex := strings.LastIndex(parseName, "/"); parseIndex >= 0 {
		parseName = parseName[parseIndex+1:]
	}
	if parseIndex2 := strings.LastIndex(parseName, "."); parseIndex2 >= 0 {
		parseName = parseName[parseIndex2+1:]
	}
	return parseName
}

// describeFiberKey is a core package helper.
func describeFiberKey(parseFiber *Fiber) string {
	if parseFiber == nil {
		return ""
	}
	if parseFiber.key != "" {
		return parseFiber.key
	}
	if parseFiber.props == nil {
		return ""
	}

	parseKey, parseOk := parseFiber.props["key"]
	if !parseOk || parseKey == nil {
		return ""
	}
	return fmt.Sprint(parseKey)
}
