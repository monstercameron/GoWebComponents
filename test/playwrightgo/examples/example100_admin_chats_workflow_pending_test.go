//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	playwright "github.com/playwright-community/playwright-go"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TestExample100AdminChatsWorkflowRegression validates the chats workflow from anomaly summary through inspector/detail and settings-entry return-to-queue behavior.
func TestExample100AdminChatsWorkflowRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL, _ := startExample100AdminListMechanicsServer(parseT, parseRepoRoot, "18118")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
		parseSuperuserToken := loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)

		parseConn := parseOpenExample100AdminMutationConn(parseT, parseBaseURL, parseSuperuserToken)
		defer parseConn.Close()
		parseClient := chatpb.NewChatServiceClient(parseConn)

		parseSummaryCtx, parseSummaryCancel := parseBuildExample100AdminMutationCallContext()
		parseSummaryResp, parseErr := parseClient.GetAdminDashboard(parseSummaryCtx, &chatpb.GetAdminDashboardRequest{
			LookbackDays: 30,
			TopLimit:     8,
			RecentLimit:  12,
		})
		parseSummaryCancel()
		if parseErr != nil {
			parseT.Fatalf("GetAdminDashboard chats workflow summary: %v", parseErr)
		}
		if parseSummaryResp.GetSummary() == nil {
			parseT.Fatalf("GetAdminDashboard returned nil summary in chats workflow")
		}

		parseAnomalyStatus := "failed"
		parseAnomalyCtx, parseAnomalyCancel := parseBuildExample100AdminMutationCallContext()
		parseAnomalyResp, parseErr := parseClient.ListAdminUsageEvents(parseAnomalyCtx, &chatpb.ListAdminUsageEventsRequest{
			LookbackDays: 30,
			Limit:        25,
			Status:       "failed",
			ListQuery: &chatpb.AdminListQuery{
				Limit:         25,
				Offset:        0,
				SortBy:        "created_at",
				SortDirection: "desc",
			},
		})
		parseAnomalyCancel()
		if parseErr != nil {
			parseT.Fatalf("ListAdminUsageEvents failed anomaly queue: %v", parseErr)
		}
		if len(parseAnomalyResp.GetEvents()) == 0 {
			parseAnomalyStatus = "all"
			parseFallbackCtx, parseFallbackCancel := parseBuildExample100AdminMutationCallContext()
			parseAnomalyResp, parseErr = parseClient.ListAdminUsageEvents(parseFallbackCtx, &chatpb.ListAdminUsageEventsRequest{
				LookbackDays: 30,
				Limit:        25,
				ListQuery: &chatpb.AdminListQuery{
					Limit:         25,
					Offset:        0,
					SortBy:        "created_at",
					SortDirection: "desc",
				},
			})
			parseFallbackCancel()
			if parseErr != nil {
				parseT.Fatalf("ListAdminUsageEvents fallback anomaly queue: %v", parseErr)
			}
		}
		if len(parseAnomalyResp.GetEvents()) == 0 {
			parseT.Fatalf("expected anomaly queue usage rows for chats workflow, got none")
		}
		parseAnomalyEvent := parseAnomalyResp.GetEvents()[0]
		if parseAnomalyEvent.GetConversationId() <= 0 {
			parseT.Fatalf("anomaly queue row missing conversation id: %+v", parseAnomalyEvent)
		}

		parseThreadInspectorCtx, parseThreadInspectorCancel := parseBuildExample100AdminMutationCallContext()
		parseThreadInspectorResp, parseErr := parseClient.ListAdminConversations(parseThreadInspectorCtx, &chatpb.ListAdminConversationsRequest{
			ListQuery: &chatpb.AdminListQuery{
				Limit:         20,
				Offset:        0,
				Search:        fmt.Sprintf("%d", parseAnomalyEvent.GetConversationId()),
				SortBy:        "last_activity_at",
				SortDirection: "desc",
			},
		})
		parseThreadInspectorCancel()
		if parseErr != nil {
			parseT.Fatalf("ListAdminConversations thread inspector: %v", parseErr)
		}
		if len(parseThreadInspectorResp.GetConversations()) == 0 {
			parseT.Fatalf("expected thread inspector conversation rows, got none")
		}

		parseRunDetailCtx, parseRunDetailCancel := parseBuildExample100AdminMutationCallContext()
		parseRunDetailResp, parseErr := parseClient.ListAdminUsageEvents(parseRunDetailCtx, &chatpb.ListAdminUsageEventsRequest{
			LookbackDays: 30,
			Limit:        25,
			ProviderId:   parseAnomalyEvent.GetProviderId(),
			ListQuery: &chatpb.AdminListQuery{
				Limit:         25,
				Offset:        0,
				Search:        strings.TrimSpace(parseAnomalyEvent.GetEventId()),
				SortBy:        "created_at",
				SortDirection: "desc",
			},
		})
		parseRunDetailCancel()
		if parseErr != nil {
			parseT.Fatalf("ListAdminUsageEvents run detail: %v", parseErr)
		}
		if len(parseRunDetailResp.GetEvents()) == 0 {
			parseT.Fatalf("expected run detail usage rows for event id %q, got none", parseAnomalyEvent.GetEventId())
		}

		parseModelsCtx, parseModelsCancel := parseBuildExample100AdminMutationCallContext()
		parseModelsResp, parseErr := parseClient.ListModelOptions(parseModelsCtx, &chatpb.ListModelOptionsRequest{})
		parseModelsCancel()
		if parseErr != nil {
			parseT.Fatalf("ListModelOptions chats workflow settings entry: %v", parseErr)
		}
		if len(parseModelsResp.GetModels()) == 0 {
			parseT.Fatalf("expected model options for settings-entry adjustment, got none")
		}
		parseModelID := strings.TrimSpace(parseModelsResp.GetModels()[0].GetId())
		if parseModelID == "" {
			parseT.Fatalf("settings-entry model id was blank in ListModelOptions response")
		}
		parseSetModelCtx, parseSetModelCancel := parseBuildExample100AdminMutationCallContext()
		if _, parseErr := parseClient.SetSelectedModel(parseSetModelCtx, wrapperspb.String(parseModelID)); parseErr != nil {
			parseSetModelCancel()
			parseT.Fatalf("SetSelectedModel chats workflow settings entry: %v", parseErr)
		}
		parseSetModelCancel()
		parseGetModelCtx, parseGetModelCancel := parseBuildExample100AdminMutationCallContext()
		parseModelResp, parseErr := parseClient.GetSelectedModel(parseGetModelCtx, &emptypb.Empty{})
		parseGetModelCancel()
		if parseErr != nil {
			parseT.Fatalf("GetSelectedModel chats workflow settings entry: %v", parseErr)
		}
		if parseModelResp.GetValue() != parseModelID {
			parseT.Fatalf("GetSelectedModel=%q want=%q", parseModelResp.GetValue(), parseModelID)
		}

		parseAssertExample100AdminChatsWorkflowRoutes(parseT, parsePage, parseBaseURL, parseAnomalyStatus, parseAnomalyEvent.GetConversationId(), parseAnomalyEvent.GetEventId())
		assertExample100AdminGuardNoRuntimeErrors(parseT, "admin-chats workflow regression flow", parseEvidence)
	})
}

// parseAssertExample100AdminChatsWorkflowRoutes verifies anomaly queue context survives inspector/detail/settings transitions and returns to the same queue state.
func parseAssertExample100AdminChatsWorkflowRoutes(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseStatus string, parseConversationID int64, parseEventID string) {
	parseT.Helper()
	parseStatus = strings.TrimSpace(parseStatus)
	if parseStatus == "" {
		parseStatus = "all"
	}
	parseQueueRoute := fmt.Sprintf("/app/dashboard/usage?tab=chats&lookback_days=30&status=%s&sort_by=created_at&sort_direction=desc&limit=25&offset=0", parseStatus)
	parseRoutes := []string{
		parseQueueRoute,
		fmt.Sprintf("%s&view=thread-inspector&conversation_id=%d", parseQueueRoute, parseConversationID),
		fmt.Sprintf("%s&view=run-detail&event_id=%s", parseQueueRoute, parseEventID),
		"/app/settings?panel=settings-model&source=chat-anomaly",
		parseQueueRoute,
	}
	for _, parseRoute := range parseRoutes {
		if _, parseErr := parsePage.Goto(parseBaseURL+parseRoute, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto chats workflow route %s: %v", parseRoute, parseErr)
		}
		parseWaitJS := fmt.Sprintf(`() => (window.location.pathname + window.location.search) === %q`, parseRoute)
		if _, parseErr := parsePage.WaitForFunction(parseWaitJS, nil); parseErr != nil {
			parseT.Fatalf("wait chats workflow route %s: %v", parseRoute, parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
			parseT.Fatalf("wait chat input on chats workflow route %s: %v", parseRoute, parseErr)
		}
	}
	parseAssertExample100AdminListURLStateNavigation(parseT, parsePage, parseBaseURL, parseRoutes)
}
