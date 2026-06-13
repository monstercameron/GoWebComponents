//go:build js && wasm

package app

import "testing"

func TestSidebarOrganizationSanitizeAndGroup(parseT *testing.T) {
	parseConversations := []convSummary{
		{ID: 1, Preview: "Alpha"},
		{ID: 2, Preview: "Beta"},
		{ID: 3, Preview: "Gamma"},
	}
	parseOrg := sidebarOrganizationState{
		Folders: []sidebarThreadFolder{
			{ID: "f-work", Name: " Work "},
			{ID: "f-work", Name: "Duplicate"},
			{ID: "", Name: "Missing"},
		},
		ThreadFolders: map[int64]string{
			1: "f-work",
			2: "missing-folder",
			9: "f-work",
		},
		ThreadOrder: []int64{3, 1, 3, 9},
	}

	parseClean := parseSanitizeSidebarOrganization(parseOrg, parseConversations)
	if len(parseClean.Folders) != 1 || parseClean.Folders[0].ID != "f-work" || parseClean.Folders[0].Name != "Work" {
		parseT.Fatalf("unexpected sanitized folders: %+v", parseClean.Folders)
	}
	if parseClean.ThreadFolders[1] != "f-work" {
		parseT.Fatalf("expected thread 1 in f-work, got %+v", parseClean.ThreadFolders)
	}
	if _, parseOk := parseClean.ThreadFolders[2]; parseOk {
		parseT.Fatalf("expected invalid folder assignment dropped, got %+v", parseClean.ThreadFolders)
	}
	parseWantOrder := []int64{3, 1, 2}
	for parseIndex, parseWant := range parseWantOrder {
		if parseClean.ThreadOrder[parseIndex] != parseWant {
			parseT.Fatalf("thread order[%d]=%d want %d; full=%+v", parseIndex, parseClean.ThreadOrder[parseIndex], parseWant, parseClean.ThreadOrder)
		}
	}

	parseGroups, parseUnfiled := parseBuildSidebarThreadGroups(parseConversations, parseClean)
	if len(parseGroups) != 1 || len(parseGroups[0].Threads) != 1 || parseGroups[0].Threads[0].ID != 1 {
		parseT.Fatalf("unexpected folder groups: %+v", parseGroups)
	}
	if len(parseUnfiled) != 2 || parseUnfiled[0].ID != 3 || parseUnfiled[1].ID != 2 {
		parseT.Fatalf("unexpected unfiled order: %+v", parseUnfiled)
	}
}

func TestSidebarOrganizationMoveAndReorderThreads(parseT *testing.T) {
	parseConversations := []convSummary{
		{ID: 1, Preview: "Alpha"},
		{ID: 2, Preview: "Beta"},
		{ID: 3, Preview: "Gamma"},
	}
	parseOrg := sidebarOrganizationState{
		Folders:       []sidebarThreadFolder{{ID: "f-work", Name: "Work"}},
		ThreadFolders: map[int64]string{},
		ThreadOrder:   []int64{1, 2, 3},
	}

	parseMoved := parseSidebarMoveThreadToFolder(parseOrg, parseConversations, 2, "f-work")
	if parseMoved.ThreadFolders[2] != "f-work" {
		parseT.Fatalf("expected thread 2 moved to folder, got %+v", parseMoved.ThreadFolders)
	}

	parseReordered := parseSidebarReorderThread(parseMoved, parseConversations, 3, 2, "f-work")
	if parseReordered.ThreadFolders[3] != "f-work" {
		parseT.Fatalf("expected thread 3 assigned to target folder, got %+v", parseReordered.ThreadFolders)
	}
	parseWantOrder := []int64{1, 3, 2}
	for parseIndex, parseWant := range parseWantOrder {
		if parseReordered.ThreadOrder[parseIndex] != parseWant {
			parseT.Fatalf("thread order[%d]=%d want %d; full=%+v", parseIndex, parseReordered.ThreadOrder[parseIndex], parseWant, parseReordered.ThreadOrder)
		}
	}

	parseUnfiled := parseSidebarMoveThreadToFolder(parseReordered, parseConversations, 2, "")
	if _, parseOk := parseUnfiled.ThreadFolders[2]; parseOk {
		parseT.Fatalf("expected thread 2 removed from folder, got %+v", parseUnfiled.ThreadFolders)
	}
}

func TestSidebarOrganizationReorderFoldersAndReplacePreview(parseT *testing.T) {
	parseOrg := sidebarOrganizationState{
		Folders: []sidebarThreadFolder{
			{ID: "a", Name: "A"},
			{ID: "b", Name: "B"},
			{ID: "c", Name: "C"},
		},
		ThreadFolders: map[int64]string{7: "b"},
		ThreadOrder:   []int64{7},
	}

	parseReordered := parseSidebarReorderFolder(parseOrg, "c", "a")
	parseWantFolders := []string{"c", "a", "b"}
	for parseIndex, parseWant := range parseWantFolders {
		if parseReordered.Folders[parseIndex].ID != parseWant {
			parseT.Fatalf("folder order[%d]=%q want %q; full=%+v", parseIndex, parseReordered.Folders[parseIndex].ID, parseWant, parseReordered.Folders)
		}
	}
	if parseReordered.ThreadFolders[7] != "b" || len(parseReordered.ThreadOrder) != 1 || parseReordered.ThreadOrder[0] != 7 {
		parseT.Fatalf("folder reorder should preserve thread metadata, got %+v", parseReordered)
	}

	parseUpdated := parseReplaceConversationPreview([]convSummary{{ID: 7, Preview: "Old"}, {ID: 8, Preview: "Other"}}, 7, " New title ")
	if parseUpdated[0].Preview != "New title" || parseUpdated[1].Preview != "Other" {
		parseT.Fatalf("unexpected preview replacement: %+v", parseUpdated)
	}
}
