//go:build !js || !wasm

package fetch

import (
	"context"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/pluginruntime"
)

func TestFetchServiceSnapshotBudgetAndCommands(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	parseQuery := UseQuery("svc/users", func(context.Context) (string, error) {
		return "server", nil
	}, QueryOptions{Tags: []string{"svc"}})
	parseQuery.Set("Ada")

	parseService := BuildFetchService()
	parseSnapshot, parseErr := parseService.GetFetchSnapshot(pluginruntime.QueryBudget{MaxItems: 1})
	if parseErr != nil {
		parseT.Fatalf("GetFetchSnapshot: %v", parseErr)
	}
	if parseSnapshot.Meta != pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDRuntime1, false) {
		parseT.Fatalf("unexpected fetch snapshot meta: %#v", parseSnapshot.Meta)
	}
	if len(parseSnapshot.Entries) != 1 || parseSnapshot.Entries[0].Key != "svc/users" || !parseSnapshot.Entries[0].Ready {
		parseT.Fatalf("unexpected fetch snapshot entries: %#v", parseSnapshot.Entries)
	}
	parseSnapshot.Entries[0].OwnerPaths = append(parseSnapshot.Entries[0].OwnerPaths, "mutated")
	parseSnapshot2, parseErr := parseService.GetFetchSnapshot(pluginruntime.QueryBudget{MaxItems: 1})
	if parseErr != nil {
		parseT.Fatalf("GetFetchSnapshot second read: %v", parseErr)
	}
	if len(parseSnapshot2.Entries[0].OwnerPaths) != 0 {
		parseT.Fatalf("snapshot owner paths were aliased: %#v", parseSnapshot2.Entries[0].OwnerPaths)
	}

	if parseErr := parseService.RevalidateFetchEntry("", pluginruntime.CommandOptions{}); parseErr == nil || !strings.Contains(parseErr.Error(), "cache key is required") {
		parseT.Fatalf("empty revalidate error = %v", parseErr)
	}
	if parseErr := parseService.RevalidateFetchEntry("svc/users", pluginruntime.CommandOptions{}); parseErr != nil {
		parseT.Fatalf("RevalidateFetchEntry: %v", parseErr)
	}
	if parseState := parseQuery.Get(); !parseState.Stale {
		parseT.Fatalf("revalidate did not mark query stale: %+v", parseState)
	}

	if parseErr := parseService.ClearFetchEntry("", pluginruntime.CommandOptions{}); parseErr == nil || !strings.Contains(parseErr.Error(), "cache key is required") {
		parseT.Fatalf("empty clear error = %v", parseErr)
	}
	if parseErr := parseService.ClearFetchEntry("svc/users", pluginruntime.CommandOptions{}); parseErr != nil {
		parseT.Fatalf("ClearFetchEntry: %v", parseErr)
	}
	parseSnapshot3, parseErr := parseService.GetFetchSnapshot(pluginruntime.QueryBudget{})
	if parseErr != nil {
		parseT.Fatalf("GetFetchSnapshot after clear: %v", parseErr)
	}
	if len(parseSnapshot3.Entries) != 0 {
		parseT.Fatalf("expected cleared fetch snapshot, got %#v", parseSnapshot3.Entries)
	}
}
