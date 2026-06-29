package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/server/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestSendDeniedForDisabledUser verifies chat send fails closed when the authenticated user is explicitly disabled.
func TestSendDeniedForDisabledUser(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "runtime-disabled@example.com")
	parseMustAssignBillingPlan(parseT, parseStore, parseUser.ID, "free")
	if parseErr := parseStore.parseUpsertUserAccessState(parseUserAccessStateWrite{
		UserID:           parseUser.ID,
		Status:           "disabled",
		Reason:           "runtime-policy-test",
		DisabledByUserID: parseUser.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertUserAccessState: %v", parseErr)
	}

	parseFake := parseNewFakeProvider()
	parseFake.streamChat = func(_ context.Context, _ provider.ChatRequest, _ func(provider.ChatEvent) error) (provider.ChatResult, error) {
		parseT.Fatal("streamChat should not be called for disabled users")
		return provider.ChatResult{}, nil
	}
	parseServer := parseNewFakeChatServer(parseStore, parseFake)
	parseCtx := parseBindAuthUser(parseServer, "peer-runtime-disabled", parseUser.ID, parseUser.Email)
	parseStream := &fakeChatSendStream{ctx: parseCtx}

	parseErr := parseServer.Send(&chatpb.SendRequest{Message: "hello", Model: modelGPT54Mini}, parseStream)
	if status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("Send disabled user status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}

// TestSendDeniedForSuspendedWorkspace verifies chat send fails closed when the user's active workspace memberships are all suspended.
func TestSendDeniedForSuspendedWorkspace(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "runtime-suspended@example.com")
	parseMustAssignBillingPlan(parseT, parseStore, parseUser.ID, "free")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "ws-runtime-suspend-send")
	if parseErr := parseStore.parseSetWorkspaceStatusByID(parseWorkspaceID, "suspended"); parseErr != nil {
		parseT.Fatalf("parseSetWorkspaceStatusByID: %v", parseErr)
	}

	parseFake := parseNewFakeProvider()
	parseFake.streamChat = func(_ context.Context, _ provider.ChatRequest, _ func(provider.ChatEvent) error) (provider.ChatResult, error) {
		parseT.Fatal("streamChat should not be called for suspended workspace access")
		return provider.ChatResult{}, nil
	}
	parseServer := parseNewFakeChatServer(parseStore, parseFake)
	parseCtx := parseBindAuthUser(parseServer, "peer-runtime-suspended", parseUser.ID, parseUser.Email)
	parseStream := &fakeChatSendStream{ctx: parseCtx}

	parseErr := parseServer.Send(&chatpb.SendRequest{Message: "hello", Model: modelGPT54Mini}, parseStream)
	if status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("Send suspended workspace status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}

// TestWorkspaceScopedWritesBlockedWhenSuspended verifies workspace-scoped API key, webhook, support, and notification writes fail when workspace status is suspended.
func TestWorkspaceScopedWritesBlockedWhenSuspended(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "runtime-workspace-write@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "ws-runtime-write-block")
	if parseErr := parseStore.parseSetWorkspaceStatusByID(parseWorkspaceID, "suspended"); parseErr != nil {
		parseT.Fatalf("parseSetWorkspaceStatusByID: %v", parseErr)
	}

	if _, parseErr := parseStore.parseCreateAPIKey(parseAPIKeyWrite{
		KeyID:       "runtime-api-key",
		WorkspaceID: parseWorkspaceID,
		UserID:      parseUser.ID,
		Label:       "Runtime Key",
		KeyPrefix:   "rk_",
		SecretHash:  "hash",
		ScopesJSON:  `["chat:send"]`,
	}); !errors.Is(parseErr, errStoreWorkspaceSuspended) {
		parseT.Fatalf("parseCreateAPIKey error=%v want=%v", parseErr, errStoreWorkspaceSuspended)
	}

	if parseErr := parseStore.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID: parseWorkspaceID,
		Label:       "Runtime Hook",
		TargetURL:   "https://example.com/hook",
		SecretHash:  "hash",
		EventsJSON:  `["chat.completed"]`,
		IsEnabled:   true,
	}); !errors.Is(parseErr, errStoreWorkspaceSuspended) {
		parseT.Fatalf("parseUpsertWebhookEndpoint error=%v want=%v", parseErr, errStoreWorkspaceSuspended)
	}

	if parseErr := parseStore.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      "runtime-ticket",
		WorkspaceID:    parseWorkspaceID,
		UserID:         parseUser.ID,
		Status:         "open",
		Priority:       "normal",
		Subject:        "Runtime support",
		Body:           "Need assistance",
		AssigneeUserID: 0,
	}); !errors.Is(parseErr, errStoreWorkspaceSuspended) {
		parseT.Fatalf("parseUpsertSupportTicket error=%v want=%v", parseErr, errStoreWorkspaceSuspended)
	}

	if _, parseErr := parseStore.parseCreateNotificationOutbox(context.Background(), parseNotificationOutboxWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseUser.ID,
		NotificationKey: "runtime-notify",
		ChannelKey:      "email",
		TemplateKey:     "runtime-template",
		Status:          "pending",
		Subject:         "Runtime Notification",
		BodyText:        "Body",
	}); !errors.Is(parseErr, errStoreWorkspaceSuspended) {
		parseT.Fatalf("parseCreateNotificationOutbox error=%v want=%v", parseErr, errStoreWorkspaceSuspended)
	}
}

// TestUserScopedWritesBlockedWhenAuthBlocked verifies user-scoped operational writes fail when auth-blocks remain after reactivation.
func TestUserScopedWritesBlockedWhenAuthBlocked(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "runtime-user-write@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "ws-runtime-user-write-block")

	if parseErr := parseStore.parseUpsertUserAccessState(parseUserAccessStateWrite{
		UserID:           parseUser.ID,
		Status:           "disabled",
		Reason:           "runtime-policy-test",
		DisabledByUserID: parseUser.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertUserAccessState: %v", parseErr)
	}
	if _, parseErr := parseStore.parseCreateAPIKey(parseAPIKeyWrite{
		KeyID:       "runtime-user-disabled-api-key",
		WorkspaceID: parseWorkspaceID,
		UserID:      parseUser.ID,
		Label:       "Runtime Key",
		KeyPrefix:   "rk_",
		SecretHash:  "hash",
		ScopesJSON:  `["chat:send"]`,
	}); !errors.Is(parseErr, errStoreUserDisabled) {
		parseT.Fatalf("parseCreateAPIKey disabled user error=%v want=%v", parseErr, errStoreUserDisabled)
	}
	if parseErr := parseStore.parseDeleteUserAuthBlock(parseUser.ID, parseUserAuthBlockKeyUserDisabled); parseErr != nil {
		parseT.Fatalf("parseDeleteUserAuthBlock: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertUserAccessState(parseUserAccessStateWrite{
		UserID:           parseUser.ID,
		Status:           "active",
		Reason:           "runtime-policy-reset",
		DisabledByUserID: parseUser.ID,
		DisabledAt:       "",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertUserAccessState active: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertUserAuthBlock(parseUser.ID, "runtime.user.blocked", "runtime-test", "runtime block"); parseErr != nil {
		parseT.Fatalf("parseUpsertUserAuthBlock: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      "runtime-user-auth-blocked-ticket",
		WorkspaceID:    parseWorkspaceID,
		UserID:         parseUser.ID,
		Status:         "open",
		Priority:       "normal",
		Subject:        "Runtime support",
		Body:           "Need assistance",
		AssigneeUserID: parseUser.ID,
	}); !errors.Is(parseErr, errStoreUserAuthBlocked) {
		parseT.Fatalf("parseUpsertSupportTicket auth-blocked user error=%v want=%v", parseErr, errStoreUserAuthBlocked)
	}
}

// TestWebhookAndSupportMessageBlockedWhenWorkspaceSuspended verifies webhook deliveries and support ticket replies fail once the workspace is suspended.
func TestWebhookAndSupportMessageBlockedWhenWorkspaceSuspended(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "runtime-webhook-support@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "ws-runtime-webhook-support")

	if parseErr := parseStore.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID: parseWorkspaceID,
		Label:       "Runtime Hook",
		TargetURL:   "https://example.com/hook",
		SecretHash:  "hash",
		EventsJSON:  `["chat.completed"]`,
		IsEnabled:   true,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookEndpoint baseline: %v", parseErr)
	}
	parseEndpointRows, parseErr := parseStore.parseListWebhookEndpoints(10)
	if parseErr != nil {
		parseT.Fatalf("parseListWebhookEndpoints baseline: %v", parseErr)
	}
	if len(parseEndpointRows) == 0 {
		parseT.Fatal("expected seeded webhook endpoint")
	}
	if parseErr = parseStore.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      "runtime-support-message-ticket",
		WorkspaceID:    parseWorkspaceID,
		UserID:         parseUser.ID,
		Status:         "open",
		Priority:       "normal",
		Subject:        "Runtime support",
		Body:           "Need assistance",
		AssigneeUserID: parseUser.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSupportTicket baseline: %v", parseErr)
	}
	parseTicketRows, parseErr := parseStore.parseListSupportTickets(10)
	if parseErr != nil {
		parseT.Fatalf("parseListSupportTickets baseline: %v", parseErr)
	}
	if len(parseTicketRows) == 0 {
		parseT.Fatal("expected seeded support ticket")
	}

	if parseErr = parseStore.parseSetWorkspaceStatusByID(parseWorkspaceID, "suspended"); parseErr != nil {
		parseT.Fatalf("parseSetWorkspaceStatusByID: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWebhookDelivery(context.Background(), parseWebhookDeliveryWrite{
		EndpointID:   parseEndpointRows[0].ID,
		EventType:    "chat.completed",
		DeliveryKey:  "runtime-delivery-suspended",
		ResponseBody: "blocked",
	}); !errors.Is(parseErr, errStoreWorkspaceSuspended) {
		parseT.Fatalf("parseUpsertWebhookDelivery suspended workspace error=%v want=%v", parseErr, errStoreWorkspaceSuspended)
	}
	if _, parseErr = parseStore.parseCreateSupportTicketMessage(parseSupportTicketMessageWrite{
		TicketID:     parseTicketRows[0].ID,
		AuthorUserID: parseUser.ID,
		MessageType:  "reply",
		Body:         "follow up",
		IsInternal:   false,
	}); !errors.Is(parseErr, errStoreWorkspaceSuspended) {
		parseT.Fatalf("parseCreateSupportTicketMessage suspended workspace error=%v want=%v", parseErr, errStoreWorkspaceSuspended)
	}
}

// TestDispatchNotificationOutboxPendingFailsSuspendedWorkspaceRows verifies pending notification dispatch marks suspended-workspace rows failed without invoking delivery callbacks.
func TestDispatchNotificationOutboxPendingFailsSuspendedWorkspaceRows(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "runtime-notify-dispatch@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "ws-runtime-notify-dispatch")
	parseNow := time.Now().UTC().Format(time.RFC3339)

	if _, parseErr := parseStore.parseCreateNotificationOutbox(context.Background(), parseNotificationOutboxWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseUser.ID,
		NotificationKey: "runtime-notify-dispatch",
		ChannelKey:      "email",
		TemplateKey:     "runtime-template",
		Status:          "pending",
		Subject:         "Dispatch Notification",
		BodyText:        "Body",
		ScheduledAt:     parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateNotificationOutbox: %v", parseErr)
	}
	if parseErr := parseStore.parseSetWorkspaceStatusByID(parseWorkspaceID, "suspended"); parseErr != nil {
		parseT.Fatalf("parseSetWorkspaceStatusByID: %v", parseErr)
	}

	parseDeliverCount := 0
	parseProcessed, parseErr := parseDispatchNotificationOutboxPending(context.Background(), parseStore, parseNow, 10, func(_ context.Context, parseRow parseNotificationOutboxRow) error {
		parseDeliverCount++
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("parseDispatchNotificationOutboxPending: %v", parseErr)
	}
	if parseProcessed != 1 || parseDeliverCount != 0 {
		parseT.Fatalf("expected processed=1 and deliver_count=0, got processed=%d deliver_count=%d", parseProcessed, parseDeliverCount)
	}

	parseRows, parseErr := parseStore.parseListNotificationOutbox(10)
	if parseErr != nil {
		parseT.Fatalf("parseListNotificationOutbox: %v", parseErr)
	}
	if len(parseRows) != 1 || parseRows[0].Status != "failed" || !strings.Contains(parseRows[0].ErrorMessage, "workspace suspended") {
		parseT.Fatalf("expected failed suspended-workspace notification row, got %+v", parseRows)
	}
}

// TestDispatchNotificationOutboxPendingFailsDisabledUserRows verifies pending notification dispatch marks disabled-user rows failed without invoking delivery callbacks.
func TestDispatchNotificationOutboxPendingFailsDisabledUserRows(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseMustCreateUser(parseT, parseStore, "runtime-notify-owner@example.com")
	parseDisabledUser := parseMustCreateUser(parseT, parseStore, "runtime-notify-disabled@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseOwner.ID, "ws-runtime-notify-disabled")
	parseNow := time.Now().UTC().Format(time.RFC3339)

	if _, parseErr := parseStore.parseCreateNotificationOutbox(context.Background(), parseNotificationOutboxWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseDisabledUser.ID,
		NotificationKey: "runtime-notify-dispatch-disabled-user",
		ChannelKey:      "email",
		TemplateKey:     "runtime-template",
		Status:          "pending",
		Subject:         "Dispatch Notification",
		BodyText:        "Body",
		ScheduledAt:     parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateNotificationOutbox: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertUserAccessState(parseUserAccessStateWrite{
		UserID:           parseDisabledUser.ID,
		Status:           "disabled",
		Reason:           "runtime-dispatch-disabled-user",
		DisabledByUserID: parseOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertUserAccessState: %v", parseErr)
	}

	parseDeliverCount := 0
	parseProcessed, parseErr := parseDispatchNotificationOutboxPending(context.Background(), parseStore, parseNow, 10, func(_ context.Context, parseRow parseNotificationOutboxRow) error {
		parseDeliverCount++
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("parseDispatchNotificationOutboxPending: %v", parseErr)
	}
	if parseProcessed != 1 || parseDeliverCount != 0 {
		parseT.Fatalf("expected processed=1 and deliver_count=0, got processed=%d deliver_count=%d", parseProcessed, parseDeliverCount)
	}

	parseRows, parseErr := parseStore.parseListNotificationOutbox(10)
	if parseErr != nil {
		parseT.Fatalf("parseListNotificationOutbox: %v", parseErr)
	}
	if len(parseRows) != 1 || parseRows[0].Status != "failed" || !strings.Contains(parseRows[0].ErrorMessage, "user disabled") {
		parseT.Fatalf("expected failed disabled-user notification row, got %+v", parseRows)
	}
}

// TestHandleBackgroundJobsFailsSuspendedWorkspaceQueue verifies workspace-queued jobs fail closed when their workspace is suspended.
func TestHandleBackgroundJobsFailsSuspendedWorkspaceQueue(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "runtime-jobs@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "ws-runtime-jobs")
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseQueueKey := fmt.Sprintf("workspace:%d", parseWorkspaceID)

	if parseErr := parseStore.parseUpsertBackgroundJob(parseBackgroundJobWrite{
		JobKey:       "runtime-job-suspended",
		JobType:      backgroundJobTypeWeeklySummary,
		QueueKey:     parseQueueKey,
		Status:       "pending",
		AttemptCount: 0,
		MaxAttempts:  3,
		PayloadJSON:  `{}`,
		RunAfter:     parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertBackgroundJob: %v", parseErr)
	}
	if parseErr := parseStore.parseSetWorkspaceStatusByID(parseWorkspaceID, "suspended"); parseErr != nil {
		parseT.Fatalf("parseSetWorkspaceStatusByID: %v", parseErr)
	}

	parseProcessed, parseErr := parseHandleBackgroundJobs(context.Background(), parseStore, parseNow, 10, parseBackgroundJobHandlers{
		HandleWeeklySummary: func(_ context.Context, parseRow parseBackgroundJobRow) error {
			parseT.Fatalf("unexpected handler invocation for suspended workspace job: %+v", parseRow)
			return nil
		},
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleBackgroundJobs: %v", parseErr)
	}
	if parseProcessed != 1 {
		parseT.Fatalf("expected one processed suspended job, got %d", parseProcessed)
	}

	parseRows, parseErr := parseStore.parseListBackgroundJobs(10)
	if parseErr != nil {
		parseT.Fatalf("parseListBackgroundJobs: %v", parseErr)
	}
	if len(parseRows) != 1 || parseRows[0].Status != "failed" || !strings.Contains(parseRows[0].ErrorMessage, "workspace suspended") {
		parseT.Fatalf("expected suspended workspace job failure row, got %+v", parseRows)
	}
}

// TestHandleBackgroundJobsFailsBlockedPayloadScope verifies payload-scoped user/workspace policy blocks pending jobs without invoking handlers.
func TestHandleBackgroundJobsFailsBlockedPayloadScope(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "runtime-jobs-payload@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "ws-runtime-jobs-payload")
	parseNow := time.Now().UTC().Format(time.RFC3339)

	if parseErr := parseStoreWeeklySummaryJob(
		context.Background(),
		parseStore,
		"runtime-job-payload-disabled-user",
		fmt.Sprintf(`{"workspace_id":%d,"user_id":%d}`, parseWorkspaceID, parseUser.ID),
		parseNow,
	); parseErr != nil {
		parseT.Fatalf("parseStoreWeeklySummaryJob baseline: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertUserAccessState(parseUserAccessStateWrite{
		UserID:           parseUser.ID,
		Status:           "disabled",
		Reason:           "runtime-policy-test",
		DisabledByUserID: parseUser.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertUserAccessState: %v", parseErr)
	}

	parseProcessed, parseErr := parseHandleBackgroundJobs(context.Background(), parseStore, parseNow, 10, parseBackgroundJobHandlers{
		HandleWeeklySummary: func(_ context.Context, parseRow parseBackgroundJobRow) error {
			parseT.Fatalf("unexpected handler invocation for blocked payload scope: %+v", parseRow)
			return nil
		},
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleBackgroundJobs: %v", parseErr)
	}
	if parseProcessed != 1 {
		parseT.Fatalf("expected one processed blocked payload job, got %d", parseProcessed)
	}

	parseRows, parseErr := parseStore.parseListBackgroundJobs(10)
	if parseErr != nil {
		parseT.Fatalf("parseListBackgroundJobs: %v", parseErr)
	}
	if len(parseRows) != 1 || parseRows[0].Status != "failed" || !strings.Contains(parseRows[0].ErrorMessage, "store user disabled") {
		parseT.Fatalf("expected blocked payload job failure row, got %+v", parseRows)
	}
}
