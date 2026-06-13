//go:build js && wasm

package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	sidebarDragKindThread = "thread"
	sidebarDragKindFolder = "folder"
)

type sidebarThreadFolder struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type sidebarOrganizationState struct {
	Folders       []sidebarThreadFolder `json:"folders"`
	ThreadFolders map[int64]string      `json:"threadFolders"`
	ThreadOrder   []int64               `json:"threadOrder"`
}

type sidebarThreadGroup struct {
	Folder  sidebarThreadFolder
	Threads []convSummary
}

func parseNormalizeSidebarFolderName(parseName string, parseFallback string) string {
	parseTrimmed := strings.TrimSpace(parseName)
	if parseTrimmed == "" {
		parseTrimmed = strings.TrimSpace(parseFallback)
	}
	if parseTrimmed == "" {
		parseTrimmed = "Folder"
	}
	if len(parseTrimmed) > 48 {
		parseTrimmed = strings.TrimSpace(parseTrimmed[:48])
	}
	return parseTrimmed
}

func parseNewSidebarFolderID() string {
	return "folder-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func parseSanitizeSidebarOrganization(parseOrg sidebarOrganizationState, parseConversations []convSummary) sidebarOrganizationState {
	parseNext := sidebarOrganizationState{
		Folders:       make([]sidebarThreadFolder, 0, len(parseOrg.Folders)),
		ThreadFolders: map[int64]string{},
		ThreadOrder:   make([]int64, 0, len(parseConversations)),
	}
	parseFolderIDs := map[string]struct{}{}
	for parseIndex, parseFolder := range parseOrg.Folders {
		parseID := strings.TrimSpace(parseFolder.ID)
		if parseID == "" {
			continue
		}
		if _, parseSeen := parseFolderIDs[parseID]; parseSeen {
			continue
		}
		parseFolderIDs[parseID] = struct{}{}
		parseNext.Folders = append(parseNext.Folders, sidebarThreadFolder{
			ID:   parseID,
			Name: parseNormalizeSidebarFolderName(parseFolder.Name, fmt.Sprintf("Folder %d", parseIndex+1)),
		})
	}
	if parseConversations == nil {
		for parseThreadID, parseFolderID := range parseOrg.ThreadFolders {
			if parseThreadID <= 0 {
				continue
			}
			parseFolderID = strings.TrimSpace(parseFolderID)
			if parseFolderID == "" {
				continue
			}
			if _, parseKnownFolder := parseFolderIDs[parseFolderID]; parseKnownFolder {
				parseNext.ThreadFolders[parseThreadID] = parseFolderID
			}
		}
		parseSeenThreads := map[int64]struct{}{}
		for _, parseID := range parseOrg.ThreadOrder {
			if parseID <= 0 {
				continue
			}
			if _, parseSeen := parseSeenThreads[parseID]; parseSeen {
				continue
			}
			parseSeenThreads[parseID] = struct{}{}
			parseNext.ThreadOrder = append(parseNext.ThreadOrder, parseID)
		}
		return parseNext
	}
	parseConversationIDs := map[int64]convSummary{}
	for _, parseSummary := range parseConversations {
		if parseSummary.ID <= 0 {
			continue
		}
		parseConversationIDs[parseSummary.ID] = parseSummary
		if parseFolderID := strings.TrimSpace(parseOrg.ThreadFolders[parseSummary.ID]); parseFolderID != "" {
			if _, parseKnownFolder := parseFolderIDs[parseFolderID]; parseKnownFolder {
				parseNext.ThreadFolders[parseSummary.ID] = parseFolderID
			}
		}
	}
	parseSeenThreads := map[int64]struct{}{}
	for _, parseID := range parseOrg.ThreadOrder {
		if parseID <= 0 {
			continue
		}
		if _, parseExists := parseConversationIDs[parseID]; !parseExists {
			continue
		}
		if _, parseSeen := parseSeenThreads[parseID]; parseSeen {
			continue
		}
		parseSeenThreads[parseID] = struct{}{}
		parseNext.ThreadOrder = append(parseNext.ThreadOrder, parseID)
	}
	for _, parseSummary := range parseConversations {
		if parseSummary.ID <= 0 {
			continue
		}
		if _, parseSeen := parseSeenThreads[parseSummary.ID]; parseSeen {
			continue
		}
		parseSeenThreads[parseSummary.ID] = struct{}{}
		parseNext.ThreadOrder = append(parseNext.ThreadOrder, parseSummary.ID)
	}
	return parseNext
}

func parseSidebarOrderedConversations(parseConversations []convSummary, parseThreadOrder []int64) []convSummary {
	parseByID := map[int64]convSummary{}
	for _, parseSummary := range parseConversations {
		if parseSummary.ID > 0 {
			parseByID[parseSummary.ID] = parseSummary
		}
	}
	parseOrdered := make([]convSummary, 0, len(parseConversations))
	parseSeen := map[int64]struct{}{}
	for _, parseID := range parseThreadOrder {
		parseSummary, parseOk := parseByID[parseID]
		if !parseOk {
			continue
		}
		parseOrdered = append(parseOrdered, parseSummary)
		parseSeen[parseID] = struct{}{}
	}
	for _, parseSummary := range parseConversations {
		if parseSummary.ID <= 0 {
			continue
		}
		if _, parseOk := parseSeen[parseSummary.ID]; parseOk {
			continue
		}
		parseOrdered = append(parseOrdered, parseSummary)
	}
	return parseOrdered
}

func parseReplaceConversationPreview(parseConversations []convSummary, parseConversationID int64, parsePreview string) []convSummary {
	parseNext := append([]convSummary(nil), parseConversations...)
	for parseIndex, parseSummary := range parseNext {
		if parseSummary.ID == parseConversationID {
			parseNext[parseIndex].Preview = strings.TrimSpace(parsePreview)
			break
		}
	}
	return parseNext
}

func parseBuildSidebarThreadGroups(parseConversations []convSummary, parseOrg sidebarOrganizationState) ([]sidebarThreadGroup, []convSummary) {
	parseClean := parseSanitizeSidebarOrganization(parseOrg, parseConversations)
	parseGroups := make([]sidebarThreadGroup, 0, len(parseClean.Folders))
	parseThreadsByFolder := map[string][]convSummary{}
	parseUnfiled := make([]convSummary, 0)
	for _, parseSummary := range parseSidebarOrderedConversations(parseConversations, parseClean.ThreadOrder) {
		if parseFolderID := parseClean.ThreadFolders[parseSummary.ID]; parseFolderID != "" {
			parseThreadsByFolder[parseFolderID] = append(parseThreadsByFolder[parseFolderID], parseSummary)
			continue
		}
		parseUnfiled = append(parseUnfiled, parseSummary)
	}
	for _, parseFolder := range parseClean.Folders {
		parseGroups = append(parseGroups, sidebarThreadGroup{
			Folder:  parseFolder,
			Threads: parseThreadsByFolder[parseFolder.ID],
		})
	}
	return parseGroups, parseUnfiled
}

func parseSidebarCreateFolder(parseOrg sidebarOrganizationState, parseName string) sidebarOrganizationState {
	parseNext := parseSanitizeSidebarOrganization(parseOrg, nil)
	parseNext.Folders = append(parseNext.Folders, sidebarThreadFolder{
		ID:   parseNewSidebarFolderID(),
		Name: parseNormalizeSidebarFolderName(parseName, fmt.Sprintf("Folder %d", len(parseNext.Folders)+1)),
	})
	return parseNext
}

func parseSidebarRenameFolder(parseOrg sidebarOrganizationState, parseFolderID string, parseName string) sidebarOrganizationState {
	parseNext := parseSanitizeSidebarOrganization(parseOrg, nil)
	parseFolderID = strings.TrimSpace(parseFolderID)
	for parseIndex, parseFolder := range parseNext.Folders {
		if parseFolder.ID == parseFolderID {
			parseNext.Folders[parseIndex].Name = parseNormalizeSidebarFolderName(parseName, parseFolder.Name)
			break
		}
	}
	return parseNext
}

func parseSidebarMoveThreadToFolder(parseOrg sidebarOrganizationState, parseConversations []convSummary, parseThreadID int64, parseFolderID string) sidebarOrganizationState {
	parseNext := parseSanitizeSidebarOrganization(parseOrg, parseConversations)
	if parseThreadID <= 0 {
		return parseNext
	}
	parseFolderID = strings.TrimSpace(parseFolderID)
	if parseFolderID == "" {
		delete(parseNext.ThreadFolders, parseThreadID)
		return parseNext
	}
	for _, parseFolder := range parseNext.Folders {
		if parseFolder.ID == parseFolderID {
			parseNext.ThreadFolders[parseThreadID] = parseFolderID
			return parseNext
		}
	}
	return parseNext
}

func parseSidebarReorderThread(parseOrg sidebarOrganizationState, parseConversations []convSummary, parseThreadID, parseBeforeThreadID int64, parseFolderID string) sidebarOrganizationState {
	parseNext := parseSidebarMoveThreadToFolder(parseOrg, parseConversations, parseThreadID, parseFolderID)
	if parseThreadID <= 0 || parseThreadID == parseBeforeThreadID {
		return parseNext
	}
	parseOrder := make([]int64, 0, len(parseNext.ThreadOrder))
	for _, parseID := range parseNext.ThreadOrder {
		if parseID != parseThreadID {
			parseOrder = append(parseOrder, parseID)
		}
	}
	parseInserted := false
	parseReordered := make([]int64, 0, len(parseOrder)+1)
	for _, parseID := range parseOrder {
		if !parseInserted && parseBeforeThreadID > 0 && parseID == parseBeforeThreadID {
			parseReordered = append(parseReordered, parseThreadID)
			parseInserted = true
		}
		parseReordered = append(parseReordered, parseID)
	}
	if !parseInserted {
		parseReordered = append(parseReordered, parseThreadID)
	}
	parseNext.ThreadOrder = parseReordered
	return parseSanitizeSidebarOrganization(parseNext, parseConversations)
}

func parseSidebarReorderFolder(parseOrg sidebarOrganizationState, parseFolderID, parseBeforeFolderID string) sidebarOrganizationState {
	parseNext := parseSanitizeSidebarOrganization(parseOrg, nil)
	parseFolderID = strings.TrimSpace(parseFolderID)
	parseBeforeFolderID = strings.TrimSpace(parseBeforeFolderID)
	if parseFolderID == "" || parseFolderID == parseBeforeFolderID {
		return parseNext
	}
	var parseMoved sidebarThreadFolder
	hasMoved := false
	parseFolders := make([]sidebarThreadFolder, 0, len(parseNext.Folders))
	for _, parseFolder := range parseNext.Folders {
		if parseFolder.ID == parseFolderID {
			parseMoved = parseFolder
			hasMoved = true
			continue
		}
		parseFolders = append(parseFolders, parseFolder)
	}
	if !hasMoved {
		return parseNext
	}
	parseInserted := false
	parseReordered := make([]sidebarThreadFolder, 0, len(parseFolders)+1)
	for _, parseFolder := range parseFolders {
		if !parseInserted && parseBeforeFolderID != "" && parseFolder.ID == parseBeforeFolderID {
			parseReordered = append(parseReordered, parseMoved)
			parseInserted = true
		}
		parseReordered = append(parseReordered, parseFolder)
	}
	if !parseInserted {
		parseReordered = append(parseReordered, parseMoved)
	}
	parseNext.Folders = parseReordered
	return parseNext
}
