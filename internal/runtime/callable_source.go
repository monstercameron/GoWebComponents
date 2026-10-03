package runtime

import (
	"reflect"
	goRuntime "runtime"
	"strconv"
	"strings"
	"sync"
)

// callableSource is the cached declaration site of one component function.
type callableSource struct {
	file string
	line int
}

// callableSourceCache maps a function entry address to its declaration site so
// diagnostics resolve file:line once per component type, never per render.
var callableSourceCache sync.Map

// describeCallableSource returns the source file and line of a component
// function, or "" and 0 when unknown or when development metadata is compiled
// out. It is for diagnostics only: file:line shifts with every edit, so it must
// never feed an identity key.
func describeCallableSource(parseValue any) (string, int) {
	if !hookSignatureRecordingEnabled {
		return "", 0
	}
	if parseComponent, parseOk := componentTypeFromFiberType(parseValue); parseOk {
		parseStateValue := parseComponent.getState.Load()
		if parseStateValue == nil {
			return "", 0
		}
		parseValue = parseStateValue.(componentRenderState).getImplementation
	}
	parseRv := reflect.ValueOf(parseValue)
	if !parseRv.IsValid() || parseRv.Kind() != reflect.Func || parseRv.IsNil() {
		return "", 0
	}
	parseFn := goRuntime.FuncForPC(parseRv.Pointer())
	if parseFn == nil {
		return "", 0
	}
	parseEntry := parseFn.Entry()
	if parseCached, parseOk := callableSourceCache.Load(parseEntry); parseOk {
		parseSource := parseCached.(callableSource)
		return parseSource.file, parseSource.line
	}
	parseFile, parseLine := parseFn.FileLine(parseEntry)
	callableSourceCache.Store(parseEntry, callableSource{file: parseFile, line: parseLine})
	return parseFile, parseLine
}

// describeFiberSource formats "file:line" for a component fiber, or "".
func describeFiberSource(parseFiber *Fiber) string {
	if parseFiber == nil {
		return ""
	}
	if parseKind, _ := describeFiber(parseFiber); parseKind != "component" {
		return ""
	}
	parseFile, parseLine := describeCallableSource(parseFiber.typeOf)
	if parseFile == "" {
		return ""
	}
	return parseFile + ":" + strconv.Itoa(parseLine)
}

// diagnosticComponentStackWithSource returns the component stack with each
// component's declaration site appended as " (file:line)". Plain
// diagnosticComponentStack stays name-only because consumers match on names.
func diagnosticComponentStackWithSource(parseFiber *Fiber) []string {
	parseStack := make([]string, 0, 8)
	for parseCurrent := parseFiber; parseCurrent != nil; parseCurrent = parseCurrent.parent {
		parseKind, parseName := describeFiber(parseCurrent)
		if parseKind == "root" || parseKind == "text" || strings.TrimSpace(parseName) == "" {
			continue
		}
		if parseSource := describeFiberSource(parseCurrent); parseSource != "" {
			parseName += " (" + parseSource + ")"
		}
		parseStack = append(parseStack, parseName)
	}
	for parseLeft, parseRight := 0, len(parseStack)-1; parseLeft < parseRight; parseLeft, parseRight = parseLeft+1, parseRight-1 {
		parseStack[parseLeft], parseStack[parseRight] = parseStack[parseRight], parseStack[parseLeft]
	}
	return parseStack
}

// diagnosticPathWithSource is diagnosticPathForFiber with each component's
// declaration site attached; use it only for human-facing "where" text, never
// as a lookup key (hot reload matches snapshots by the plain path).
func diagnosticPathWithSource(parseFiber *Fiber) string {
	parseStack := diagnosticComponentStackWithSource(parseFiber)
	if len(parseStack) == 0 {
		return ""
	}
	return strings.Join(parseStack, " > ")
}
