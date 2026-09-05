package html

import "testing"

// TestDataAttrAndDataMapCollisionResolvesToDataMap pins that when Props.DataAttr and
// the Props.Data map target the same data-* name, exactly ONE data-<name> attribute
// is emitted and the Data map wins — consistently across BOTH build paths. The
// fast-lane slice path previously appended both, emitting a DUPLICATE data-foo, while
// the map path silently let Data overwrite DataAttr; the two paths now agree.
func TestDataAttrAndDataMapCollisionResolvesToDataMap(parseT *testing.T) {
	parseProps := Props{
		DataAttr: DataAttribute{Name: "foo", Value: "from-attr"},
		Data:     map[string]string{"foo": "from-map"},
	}

	// Fast-lane (slice) path.
	_, parseAttrs, parseOk := toRuntimeCompactProps(&parseProps, nil)
	if !parseOk {
		parseT.Fatal("expected DataAttr+Data props to be fast-lane eligible")
	}
	parseCount := 0
	parseValue := ""
	for _, parseAttr := range parseAttrs {
		if parseAttr.Name == "data-foo" {
			parseCount++
			parseValue = parseAttr.Value
		}
	}
	if parseCount != 1 {
		parseT.Fatalf("expected exactly one data-foo attr (no duplicate), got %d: %#v", parseCount, parseAttrs)
	}
	if parseValue != "from-map" {
		parseT.Fatalf("Data map must win the collision in the fast-lane path, got %q", parseValue)
	}

	// Map path must resolve the collision the same way.
	parseMap := toRuntimePropsWithEvents(&parseProps, nil)
	if parseMap["data-foo"] != "from-map" {
		parseT.Fatalf("map path must also resolve to the Data map value, got %#v", parseMap["data-foo"])
	}
}

// TestDataAttrWithoutCollisionStillEmitted pins that a DataAttr with a distinct name
// (no Data-map collision) is still emitted — the collision guard must not drop it.
func TestDataAttrWithoutCollisionStillEmitted(parseT *testing.T) {
	parseProps := Props{
		DataAttr: DataAttribute{Name: "only-attr", Value: "v"},
		Data:     map[string]string{"other": "w"},
	}
	_, parseAttrs, parseOk := toRuntimeCompactProps(&parseProps, nil)
	if !parseOk {
		parseT.Fatal("expected fast-lane eligibility")
	}
	parseSeen := map[string]string{}
	for _, parseAttr := range parseAttrs {
		parseSeen[parseAttr.Name] = parseAttr.Value
	}
	if parseSeen["data-only-attr"] != "v" {
		parseT.Fatalf("non-colliding DataAttr must still be emitted, got %#v", parseSeen)
	}
	if parseSeen["data-other"] != "w" {
		parseT.Fatalf("Data map entry must still be emitted, got %#v", parseSeen)
	}
}

func TestDataAttrsUseLastValueAndDataMapWins(parseT *testing.T) {
	parseProps := Props{
		DataAttr: DataAttribute{Name: "state", Value: "first"},
		DataAttrs: []DataAttribute{
			{Name: "state", Value: "second"},
			{Name: "row-id", Value: "7"},
		},
		Data: map[string]string{"state": "map"},
	}
	_, parseAttrs, parseOK := toRuntimeCompactProps(&parseProps, nil)
	if !parseOK {
		parseT.Fatal("expected DataAttrs to remain compact-lane eligible")
	}
	parseSeen := map[string]string{}
	parseCounts := map[string]int{}
	for _, parseAttr := range parseAttrs {
		parseSeen[parseAttr.Name] = parseAttr.Value
		parseCounts[parseAttr.Name]++
	}
	if parseCounts["data-state"] != 1 || parseSeen["data-state"] != "map" {
		parseT.Fatalf("Data map must win without duplicates, attrs=%#v", parseAttrs)
	}
	if parseSeen["data-row-id"] != "7" {
		parseT.Fatalf("DataAttrs entry missing, attrs=%#v", parseAttrs)
	}

	parseMap := toRuntimePropsWithEvents(&parseProps, nil)
	if parseMap["data-state"] != "map" || parseMap["data-row-id"] != "7" {
		parseT.Fatalf("map lane collision semantics diverged: %#v", parseMap)
	}
}
