package runtime2_test

import (
	"reflect"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHandleHostRegionDeclaredSourceLookupUsesSourceBridge verifies host adapters can look up declared source values and versions through a shipped-runtime bridge.
func TestHandleHostRegionDeclaredSourceLookupUsesSourceBridge(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	var getLookupSourceIDs []string
	parseErr = buildHostRegionAdapter.SetHostRegionSourceLookup(
		func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
			getLookupSourceIDs = append([]string(nil), parseSourceIDs...)
			return map[string]any{
					"count":  9,
					"status": "healthy",
				},
				map[string]uint64{
					"count":  4,
					"status": 4,
				},
				nil
		},
	)
	if parseErr != nil {
		parseT.Fatalf("SetHostRegionSourceLookup returned error: %v", parseErr)
	}
	getSourceSnapshot, parseErr := buildHostRegionAdapter.HandleHostRegionDeclaredSourceLookup([]string{"status", "count", "count"})
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionDeclaredSourceLookup returned error: %v", parseErr)
	}
	if !reflect.DeepEqual(getLookupSourceIDs, []string{"count", "status"}) {
		parseT.Fatalf("expected canonical source lookup IDs [count status], got %+v", getLookupSourceIDs)
	}
	if getSourceSnapshot.GetSourceValues["count"] != 9 {
		parseT.Fatalf("expected source value for count=9, got %+v", getSourceSnapshot.GetSourceValues)
	}
	if getSourceSnapshot.GetSourceVersions["status"] != 4 {
		parseT.Fatalf("expected source version for status=4, got %+v", getSourceSnapshot.GetSourceVersions)
	}
}

// TestHandleHostRegionDeclaredSourceLookupRejectsMissingSourceBridge verifies host adapters fail declared-source lookup when no shipped-runtime bridge is configured.
func TestHandleHostRegionDeclaredSourceLookupRejectsMissingSourceBridge(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionDeclaredSourceLookup([]string{"count"})
	if parseErr == nil {
		parseT.Fatal("expected declared-source lookup without source bridge to fail")
	}
}

// TestHandleHostRegionDeclaredSourceLookupRejectsMissingDeclaredSourceValue verifies host source lookup fails when one declared source value is missing.
func TestHandleHostRegionDeclaredSourceLookupRejectsMissingDeclaredSourceValue(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	parseErr = buildHostRegionAdapter.SetHostRegionSourceLookup(
		func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
			return map[string]any{
					"count": 1,
				},
				map[string]uint64{
					"count":  4,
					"status": 4,
				},
				nil
		},
	)
	if parseErr != nil {
		parseT.Fatalf("SetHostRegionSourceLookup returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionDeclaredSourceLookup([]string{"count", "status"})
	if parseErr == nil {
		parseT.Fatal("expected missing declared source value to fail")
	}
}
