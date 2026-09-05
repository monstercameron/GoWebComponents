package runtime

import (
	"reflect"
	"testing"
)

func TestWrapEventHandlerCellNilAndNonFunctionContracts(parseT *testing.T) {
	parseRt := NewRuntime(Config{})
	if parseRt.wrapEventHandlerCell(nil) != nil {
		parseT.Fatal("nil handler cell should wrap to nil")
	}
	if parseRt.wrapEventHandlerCell(&funcHandlerCell{}) != nil {
		parseT.Fatal("nil handler function should wrap to nil")
	}
	parseCell := &funcHandlerCell{fn: "not-a-func"}
	if parseGot := parseRt.wrapEventHandlerCell(parseCell); parseGot != "not-a-func" {
		parseT.Fatalf("non-function handler should pass through, got %#v", parseGot)
	}
}

func TestWrapEventHandlerCellCommonNoArgDispatchDoesNotAllocate(parseT *testing.T) {
	parseRt := NewRuntime(Config{})
	parseCalls := 0
	parseFn := func() { parseCalls++ }
	parseCell := &funcHandlerCell{fn: parseFn, fnVal: reflect.ValueOf(parseFn)}
	parseWrapped, parseOK := parseRt.wrapEventHandlerCell(parseCell).(func())
	if !parseOK {
		parseT.Fatalf("wrapped handler has unexpected type %T", parseRt.wrapEventHandlerCell(parseCell))
	}
	parseAllocs := testing.AllocsPerRun(1000, parseWrapped)
	if parseAllocs != 0 {
		parseT.Fatalf("common no-arg event dispatch allocated %.2f objects per call", parseAllocs)
	}
	if parseCalls == 0 {
		parseT.Fatal("wrapped handler was not invoked")
	}
}

func TestWrapEventHandlerCellCallsCachedFunctionAndReturnsZerosWhenInvalid(parseT *testing.T) {
	parseRt := NewRuntime(Config{})
	parseCalledWith := ""
	parseFn := func(parseValue string) (string, bool) {
		parseCalledWith = parseValue
		return "seen:" + parseValue, true
	}
	parseCell := &funcHandlerCell{
		fn:    parseFn,
		fnVal: reflect.ValueOf(parseFn),
	}
	parseWrapped, parseOK := parseRt.wrapEventHandlerCell(parseCell).(func(string) (string, bool))
	if !parseOK {
		parseT.Fatalf("wrapped handler has unexpected type %T", parseRt.wrapEventHandlerCell(parseCell))
	}
	parseValue, parseBool := parseWrapped("click")
	if parseValue != "seen:click" || !parseBool || parseCalledWith != "click" {
		parseT.Fatalf("wrapped handler returned (%q, %v), calledWith=%q", parseValue, parseBool, parseCalledWith)
	}

	parseCell.fnVal = reflect.Value{}
	parseValue, parseBool = parseWrapped("ignored")
	if parseValue != "" || parseBool {
		parseT.Fatalf("invalid cached function should return zero values, got (%q, %v)", parseValue, parseBool)
	}
}

func TestBuildEventResultValues(parseT *testing.T) {
	if parseGot := buildEventResultValues(reflect.TypeOf(func() {})); parseGot != nil {
		parseT.Fatalf("zero-result function should return nil result values, got %#v", parseGot)
	}
	parseValues := buildEventResultValues(reflect.TypeOf(func() (int, string, bool) { return 0, "", false }))
	if len(parseValues) != 3 {
		parseT.Fatalf("result value count = %d, want 3", len(parseValues))
	}
	if parseValues[0].Int() != 0 || parseValues[1].String() != "" || parseValues[2].Bool() {
		parseT.Fatalf("result values should be zeroed, got %#v", parseValues)
	}
}
