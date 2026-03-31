package app

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/internal/sqlfiles"
)

// parseLoadStoreQuery loads one SQL file into one store query field.
func parseLoadStoreQuery(parseTarget *string, parsePath string) error {
	parseQueryText, parseErr := sqlfiles.ParseLoad(parsePath)
	if parseErr != nil {
		return fmt.Errorf("load store query %q: %w", parsePath, parseErr)
	}
	*parseTarget = parseQueryText
	return nil
}

type storeQueries struct {
	schema                                  string
	migrations                              string
	createUser                              string
	upsertUserProfileName                   string
	getUserAuthByEmail                      string
	userExists                              string
	upsertUserAccessState                   string
	getUserAccessState                      string
	upsertAuthTokenVersion                  string
	getAuthTokenVersion                     string
	upsertAuthIdentity                      string
	getAuthIdentityByProviderSubject        string
	listAuthIdentitiesByUser                string
	deleteAuthIdentityByScope               string
	createAuthOIDCState                     string
	getAuthOIDCStateByStateTokenHash        string
	consumeAuthOIDCStateByStateTokenHash    string
	deleteExpiredAuthOIDCStates             string
	upsertWorkspaceAuthPolicy               string
	getWorkspaceAuthPolicyByWorkspace       string
	upsertUserAuthBlock                     string
	countUserAuthBlocksByUser               string
	deleteUserAuthBlock                     string
	deleteUserAuthBlocksByKey               string
	upsertAuthSession                       string
	getAuthSessionBySessionID               string
	touchAuthSessionLastSeen                string
	revokeAuthSession                       string
	revokeAuthSessionsByUser                string
	updateUserPasswordHash                  string
	createEmailVerificationToken            string
	getEmailVerificationTokenByHash         string
	markEmailVerificationTokenVerified      string
	createPasswordResetToken                string
	getPasswordResetTokenByHash             string
	consumePasswordResetToken               string
	conversationOwnedByUser                 string
	resolveConversationRoute                string
	createConversation                      string
	saveConversationMessage                 string
	saveConversationTitle                   string
	listConversations                       string
	loadConversation                        string
	deleteConversationMessages              string
	deleteConversationUsageEvents           string
	deleteConversation                      string
	getUserName                             string
	setSelectedModel                        string
	getSelectedModel                        string
	setSelectedTone                         string
	getSelectedTone                         string
	setSelectedThinkingEnabled              string
	getSelectedThinkingEnabled              string
	setSelectedThinkingEffort               string
	getSelectedThinkingEffort               string
	setSelectedSystemPrompt                 string
	getSelectedSystemPrompt                 string
	upsertUserMemory                        string
	listUserMemories                        string
	deleteUserMemory                        string
	listModelCatalog                        string
	saveUsageEvent                          string
	listUsageEvents                         string
	sumUsageTokensSince                     string
	getModelPricing                         string
	upsertBillingCustomer                   string
	getBillingCustomerByUser                string
	upsertBillingSubscription               string
	getBillingSubscriptionByProvider        string
	listBillingSubscriptionsByCustomer      string
	upsertBillingInvoice                    string
	getBillingInvoiceByProvider             string
	listBillingInvoicesByCustomer           string
	createBillingInvoiceLineItem            string
	listBillingInvoiceLineItems             string
	upsertBillingAccessOverride             string
	listBillingAccessOverridesByCustomer    string
	createBillingEvent                      string
	listBillingEventsByCustomer             string
	listBillingDunningEventsByCustomer      string
	getBillingDunningEventByID              string
	upsertBillingDunningEvent               string
	deleteBillingDunningEvent               string
	updateBillingDunningEventResolved       string
	updateBillingInvoiceResolution          string
	listBillingPlans                        string
	upsertBillingPlan                       string
	deleteBillingPlan                       string
	listBillingPlanEntitlements             string
	upsertBillingPlanEntitlement            string
	deleteBillingPlanEntitlement            string
	listBillingPlanOverages                 string
	upsertBillingPlanOverage                string
	deleteBillingPlanOverage                string
	listBillingQuotaPolicies                string
	upsertBillingQuotaPolicy                string
	deleteBillingQuotaPolicy                string
	listBillingUpgradeTriggers              string
	upsertBillingUpgradeTrigger             string
	deleteBillingUpgradeTrigger             string
	listBillingEffectiveAccessByUser        string
	listBillingEffectiveModelAccessByUser   string
	upsertSURole                            string
	listSURoles                             string
	deleteSURolePermissions                 string
	insertSURolePermission                  string
	listSURolePermissions                   string
	upsertSUUserRole                        string
	listSUUserRoles                         string
	userHasSURole                           string
	upsertSiteConfig                        string
	listSiteConfigs                         string
	createServerToolPolicyHistory           string
	listServerToolPolicyHistory             string
	upsertFeatureFlag                       string
	listFeatureFlags                        string
	upsertWorkspace                         string
	getWorkspaceByID                        string
	listWorkspaces                          string
	upsertWorkspaceMembership               string
	listWorkspaceMemberships                string
	listWorkspaceMembershipsByUser          string
	listWorkspaceMembershipsByWorkspace     string
	countWorkspaceMembershipsByUser         string
	countActiveWorkspaceMembershipsByUser   string
	listWorkspaceSSOConfigs                 string
	getWorkspaceSSOConfigByScope            string
	upsertWorkspaceSSOConfig                string
	deleteWorkspaceSSOConfig                string
	listDataRetentionPolicies               string
	getDataRetentionPolicyByScope           string
	upsertDataRetentionPolicy               string
	deleteDataRetentionPolicy               string
	listComplianceControls                  string
	getComplianceControlByScope             string
	upsertComplianceControl                 string
	deleteComplianceControl                 string
	listServiceLevelObjectives              string
	getServiceLevelObjectiveByKey           string
	upsertServiceLevelObjective             string
	deleteServiceLevelObjective             string
	listAuthSessions                        string
	upsertWorkspaceInvitation               string
	listWorkspaceInvitations                string
	createAPIKey                            string
	listAPIKeys                             string
	revokeAPIKey                            string
	upsertWebhookEndpoint                   string
	listWebhookEndpoints                    string
	upsertWebhookDelivery                   string
	listWebhookDeliveries                   string
	listWebhookDeliveriesPendingRetry       string
	updateWebhookDeliveryAttempt            string
	updateWebhookDeliveryDelivered          string
	createAuditLog                          string
	listAuditLogs                           string
	upsertSupportTicket                     string
	listSupportTickets                      string
	createSupportTicketMessage              string
	listSupportTicketMessages               string
	createIncidentUpdate                    string
	listIncidentUpdates                     string
	getIncidentUpdateByID                   string
	upsertIncidentUpdate                    string
	deleteIncidentUpdate                    string
	updateIncidentStatus                    string
	listIncidents                           string
	getIncidentByKey                        string
	upsertIncident                          string
	deleteIncident                          string
	upsertWorkspaceModelRoutingPolicy       string
	listWorkspaceModelRoutingPolicies       string
	upsertWorkspaceCostGuardrail            string
	listWorkspaceCostGuardrails             string
	createNotificationOutbox                string
	listNotificationOutbox                  string
	listNotificationOutboxPending           string
	updateNotificationOutboxStatus          string
	upsertBackgroundJob                     string
	listBackgroundJobs                      string
	upsertOnboardingTemplate                string
	listOnboardingTemplates                 string
	upsertUserActivationMilestone           string
	listUserActivationMilestones            string
	upsertSavedWorkflow                     string
	listSavedWorkflows                      string
	upsertPromptLibraryItem                 string
	listPromptLibraryItems                  string
	upsertWeeklyValueSummary                string
	listWeeklyValueSummaries                string
	createProductAnalyticsEvent             string
	listProductAnalyticsEvents              string
	listProductAnalyticsEventsByUser        string
	upsertExperimentAssignment              string
	listExperimentAssignments               string
	createSubscriptionChurnFeedback         string
	listSubscriptionChurnFeedback           string
	listSubscriptionChurnFeedbackByCustomer string
	upsertExperiment                        string
	listExperiments                         string
	getAdminDashboardSummary                string
	listAdminDashboardDailyUsage            string
	listAdminDashboardProviderUsage         string
	listAdminDashboardModelUsage            string
	listAdminDashboardUserUsage             string
	listAdminUsageEvents                    string
	listAdminUsers                          string
	listAdminConversations                  string
	listAdminChatFailedReplies              string
	listAdminChatSlowReplies                string
	listAdminChatHighCostThreads            string
	listAdminChatFeatureUsageSlices         string
	listAdminCustomerAccountTimelineEvents  string
	listAdminProviderHealthTrends           string
	listAdminProviderFallbackEvents         string
	listAdminFailedBackgroundJobs           string
	listAdminFailedNotifications            string
	listAdminFailedWebhookDeliveries        string
	listAdminIncidentTimeline               string
	listAdminRecentOpsActions               string
	getAdminBusinessMutationPreview         string
	getAdminProviderMutationPreview         string
	getAdminOpsMutationPreview              string
	listAdminFailedPaymentBillingEvents     string
	listAdminBillingDunningTimeline         string
	refreshProviderUsageDailyRollups        string
	listProviderUsageDailyRollups           string
}

func parseLoadStoreQueries() (storeQueries, error) {
	var parseQueries storeQueries
	var parseErr error

	if parseErr = parseLoadStoreQuery(&parseQueries.schema, "store/schema.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.migrations, "store/migrations.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.createUser, "store/ops/create_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertUserProfileName, "store/ops/upsert_user_profile_name.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getUserAuthByEmail, "store/ops/get_user_auth_by_email.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.userExists, "store/ops/user_exists.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertUserAccessState, "store/auth/upsert_user_access_state.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getUserAccessState, "store/ops/get_user_access_state.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertAuthTokenVersion, "store/ops/upsert_auth_token_version.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getAuthTokenVersion, "store/ops/get_auth_token_version.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertAuthIdentity, "store/ops/upsert_auth_identity.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getAuthIdentityByProviderSubject, "store/ops/get_auth_identity_by_provider_subject.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAuthIdentitiesByUser, "store/ops/list_auth_identities_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteAuthIdentityByScope, "store/ops/delete_auth_identity_by_scope.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.createAuthOIDCState, "store/ops/create_auth_oidc_state.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getAuthOIDCStateByStateTokenHash, "store/ops/get_auth_oidc_state_by_state_token_hash.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.consumeAuthOIDCStateByStateTokenHash, "store/ops/consume_auth_oidc_state_by_state_token_hash.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteExpiredAuthOIDCStates, "store/auth/delete_expired_auth_oidc_states.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertWorkspaceAuthPolicy, "store/ops/upsert_workspace_auth_policy.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getWorkspaceAuthPolicyByWorkspace, "store/ops/get_workspace_auth_policy_by_workspace.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertUserAuthBlock, "store/auth/upsert_user_auth_block.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.countUserAuthBlocksByUser, "store/auth/count_user_auth_blocks_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteUserAuthBlock, "store/auth/delete_user_auth_block.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteUserAuthBlocksByKey, "store/auth/delete_user_auth_blocks_by_key.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertAuthSession, "store/ops/upsert_auth_session.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getAuthSessionBySessionID, "store/ops/get_auth_session_by_session_id.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.touchAuthSessionLastSeen, "store/ops/touch_auth_session_last_seen.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.revokeAuthSession, "store/ops/revoke_auth_session.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.revokeAuthSessionsByUser, "store/ops/revoke_auth_sessions_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.updateUserPasswordHash, "store/ops/update_user_password_hash.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.createEmailVerificationToken, "store/auth/create_email_verification_token.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getEmailVerificationTokenByHash, "store/auth/get_email_verification_token_by_hash.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.markEmailVerificationTokenVerified, "store/auth/mark_email_verification_token_verified.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.createPasswordResetToken, "store/auth/create_password_reset_token.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getPasswordResetTokenByHash, "store/auth/get_password_reset_token_by_hash.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.consumePasswordResetToken, "store/auth/consume_password_reset_token.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.conversationOwnedByUser, "store/chat/conversation_owned_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.resolveConversationRoute, "store/ops/resolve_conversation_route.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.createConversation, "store/ops/create_conversation.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.saveConversationMessage, "store/chat/save_conversation_message.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.saveConversationTitle, "store/chat/save_conversation_title.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listConversations, "store/ops/list_conversations.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.loadConversation, "store/ops/load_conversation.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteConversationMessages, "store/ops/delete_conversation_messages.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteConversationUsageEvents, "store/ops/delete_conversation_usage_events.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteConversation, "store/ops/delete_conversation.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getUserName, "store/ops/get_user_name.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.setSelectedModel, "store/chat/set_selected_model.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getSelectedModel, "store/chat/get_selected_model.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.setSelectedTone, "store/chat/set_selected_tone.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getSelectedTone, "store/chat/get_selected_tone.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.setSelectedThinkingEnabled, "store/chat/set_selected_thinking_enabled.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getSelectedThinkingEnabled, "store/chat/get_selected_thinking_enabled.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.setSelectedThinkingEffort, "store/chat/set_selected_thinking_effort.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getSelectedThinkingEffort, "store/chat/get_selected_thinking_effort.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.setSelectedSystemPrompt, "store/chat/set_selected_system_prompt.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getSelectedSystemPrompt, "store/chat/get_selected_system_prompt.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertUserMemory, "store/chat/upsert_user_memory.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listUserMemories, "store/chat/list_user_memories.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteUserMemory, "store/chat/delete_user_memory.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listModelCatalog, "store/ops/list_model_catalog.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.saveUsageEvent, "store/ops/save_usage_event.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listUsageEvents, "store/ops/list_usage_events.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.sumUsageTokensSince, "store/ops/sum_usage_tokens_since.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getModelPricing, "store/ops/get_model_pricing.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertBillingCustomer, "store/billing/upsert_billing_customer.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getBillingCustomerByUser, "store/billing/get_billing_customer_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertBillingSubscription, "store/billing/upsert_billing_subscription.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getBillingSubscriptionByProvider, "store/billing/get_billing_subscription_by_provider.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listBillingSubscriptionsByCustomer, "store/billing/list_billing_subscriptions_by_customer.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertBillingInvoice, "store/billing/upsert_billing_invoice.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getBillingInvoiceByProvider, "store/billing/get_billing_invoice_by_provider.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listBillingInvoicesByCustomer, "store/billing/list_billing_invoices_by_customer.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.createBillingInvoiceLineItem, "store/billing/create_billing_invoice_line_item.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listBillingInvoiceLineItems, "store/billing/list_billing_invoice_line_items.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertBillingAccessOverride, "store/billing/upsert_billing_access_override.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listBillingAccessOverridesByCustomer, "store/billing/list_billing_access_overrides_by_customer.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.createBillingEvent, "store/billing/create_billing_event.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listBillingEventsByCustomer, "store/billing/list_billing_events_by_customer.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listBillingDunningEventsByCustomer, "store/billing/list_billing_dunning_events_by_customer.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getBillingDunningEventByID, "store/billing/get_billing_dunning_event_by_id.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertBillingDunningEvent, "store/billing/upsert_billing_dunning_event.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteBillingDunningEvent, "store/billing/delete_billing_dunning_event.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.updateBillingDunningEventResolved, "store/billing/update_billing_dunning_event_resolved.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.updateBillingInvoiceResolution, "store/billing/update_billing_invoice_resolution.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listBillingPlans, "store/billing/list_billing_plans.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertBillingPlan, "store/billing/upsert_billing_plan.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteBillingPlan, "store/billing/delete_billing_plan.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listBillingPlanEntitlements, "store/billing/list_billing_plan_entitlements.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertBillingPlanEntitlement, "store/billing/upsert_billing_plan_entitlement.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteBillingPlanEntitlement, "store/billing/delete_billing_plan_entitlement.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listBillingPlanOverages, "store/billing/list_billing_plan_overages.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertBillingPlanOverage, "store/billing/upsert_billing_plan_overage.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteBillingPlanOverage, "store/billing/delete_billing_plan_overage.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listBillingQuotaPolicies, "store/billing/list_billing_quota_policies.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertBillingQuotaPolicy, "store/billing/upsert_billing_quota_policy.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteBillingQuotaPolicy, "store/billing/delete_billing_quota_policy.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listBillingUpgradeTriggers, "store/billing/list_billing_upgrade_triggers.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertBillingUpgradeTrigger, "store/billing/upsert_billing_upgrade_trigger.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteBillingUpgradeTrigger, "store/billing/delete_billing_upgrade_trigger.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listBillingEffectiveAccessByUser, "store/ops/list_billing_effective_access_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listBillingEffectiveModelAccessByUser, "store/ops/list_billing_effective_model_access_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertSURole, "store/admin/upsert_su_role.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listSURoles, "store/admin/list_su_roles.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteSURolePermissions, "store/admin/delete_su_role_permissions.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.insertSURolePermission, "store/admin/insert_su_role_permission.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listSURolePermissions, "store/admin/list_su_role_permissions.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertSUUserRole, "store/admin/upsert_su_user_role.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listSUUserRoles, "store/admin/list_su_user_roles.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.userHasSURole, "store/ops/user_has_su_role.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertSiteConfig, "store/admin/upsert_site_config.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listSiteConfigs, "store/admin/list_site_configs.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.createServerToolPolicyHistory, "store/ops/create_server_tool_policy_history.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listServerToolPolicyHistory, "store/ops/list_server_tool_policy_history.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertFeatureFlag, "store/admin/upsert_feature_flag.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listFeatureFlags, "store/admin/list_feature_flags.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertWorkspace, "store/ops/upsert_workspace.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getWorkspaceByID, "store/ops/get_workspace_by_id.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listWorkspaces, "store/ops/list_workspaces.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertWorkspaceMembership, "store/admin/upsert_workspace_membership.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listWorkspaceMemberships, "store/admin/list_workspace_memberships.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listWorkspaceMembershipsByUser, "store/admin/list_workspace_memberships_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listWorkspaceMembershipsByWorkspace, "store/admin/list_workspace_memberships_by_workspace.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.countWorkspaceMembershipsByUser, "store/admin/count_workspace_memberships_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.countActiveWorkspaceMembershipsByUser, "store/ops/count_active_workspace_memberships_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listWorkspaceSSOConfigs, "store/ops/list_workspace_sso_configs.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getWorkspaceSSOConfigByScope, "store/ops/get_workspace_sso_config_by_scope.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertWorkspaceSSOConfig, "store/ops/upsert_workspace_sso_config.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteWorkspaceSSOConfig, "store/ops/delete_workspace_sso_config.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listDataRetentionPolicies, "store/ops/list_data_retention_policies.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getDataRetentionPolicyByScope, "store/ops/get_data_retention_policy_by_scope.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertDataRetentionPolicy, "store/ops/upsert_data_retention_policy.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteDataRetentionPolicy, "store/ops/delete_data_retention_policy.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listComplianceControls, "store/ops/list_compliance_controls.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getComplianceControlByScope, "store/ops/get_compliance_control_by_scope.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertComplianceControl, "store/ops/upsert_compliance_control.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteComplianceControl, "store/ops/delete_compliance_control.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listServiceLevelObjectives, "store/ops/list_service_level_objectives.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getServiceLevelObjectiveByKey, "store/ops/get_service_level_objective_by_key.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertServiceLevelObjective, "store/ops/upsert_service_level_objective.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteServiceLevelObjective, "store/ops/delete_service_level_objective.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAuthSessions, "store/ops/list_auth_sessions.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertWorkspaceInvitation, "store/admin/upsert_workspace_invitation.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listWorkspaceInvitations, "store/admin/list_workspace_invitations.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.createAPIKey, "store/ops/create_api_key.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAPIKeys, "store/ops/list_api_keys.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.revokeAPIKey, "store/ops/revoke_api_key.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertWebhookEndpoint, "store/ops/upsert_webhook_endpoint.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listWebhookEndpoints, "store/ops/list_webhook_endpoints.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertWebhookDelivery, "store/ops/upsert_webhook_delivery.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listWebhookDeliveries, "store/ops/list_webhook_deliveries.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listWebhookDeliveriesPendingRetry, "store/ops/list_webhook_deliveries_pending_retry.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.updateWebhookDeliveryAttempt, "store/ops/update_webhook_delivery_attempt.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.updateWebhookDeliveryDelivered, "store/ops/update_webhook_delivery_delivered.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.createAuditLog, "store/ops/create_audit_log.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAuditLogs, "store/ops/list_audit_logs.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertSupportTicket, "store/ops/upsert_support_ticket.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listSupportTickets, "store/ops/list_support_tickets.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.createSupportTicketMessage, "store/ops/create_support_ticket_message.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listSupportTicketMessages, "store/ops/list_support_ticket_messages.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.createIncidentUpdate, "store/ops/create_incident_update.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listIncidentUpdates, "store/ops/list_incident_updates.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getIncidentUpdateByID, "store/ops/get_incident_update_by_id.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertIncidentUpdate, "store/ops/upsert_incident_update.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteIncidentUpdate, "store/ops/delete_incident_update.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.updateIncidentStatus, "store/ops/update_incident_status.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listIncidents, "store/ops/list_incidents.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getIncidentByKey, "store/ops/get_incident_by_key.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertIncident, "store/ops/upsert_incident.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.deleteIncident, "store/ops/delete_incident.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertWorkspaceModelRoutingPolicy, "store/ops/upsert_workspace_model_routing_policy.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listWorkspaceModelRoutingPolicies, "store/ops/list_workspace_model_routing_policies.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertWorkspaceCostGuardrail, "store/ops/upsert_workspace_cost_guardrail.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listWorkspaceCostGuardrails, "store/ops/list_workspace_cost_guardrails.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.createNotificationOutbox, "store/ops/create_notification_outbox.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listNotificationOutbox, "store/ops/list_notification_outbox.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listNotificationOutboxPending, "store/ops/list_notification_outbox_pending.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.updateNotificationOutboxStatus, "store/ops/update_notification_outbox_status.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertBackgroundJob, "store/ops/upsert_background_job.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listBackgroundJobs, "store/ops/list_background_jobs.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertOnboardingTemplate, "store/onboarding/upsert_onboarding_template.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listOnboardingTemplates, "store/onboarding/list_onboarding_templates.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertUserActivationMilestone, "store/onboarding/upsert_user_activation_milestone.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listUserActivationMilestones, "store/onboarding/list_user_activation_milestones.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertSavedWorkflow, "store/onboarding/upsert_saved_workflow.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listSavedWorkflows, "store/onboarding/list_saved_workflows.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertPromptLibraryItem, "store/ops/upsert_prompt_library_item.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listPromptLibraryItems, "store/ops/list_prompt_library_items.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertWeeklyValueSummary, "store/ops/upsert_weekly_value_summary.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listWeeklyValueSummaries, "store/ops/list_weekly_value_summaries.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.createProductAnalyticsEvent, "store/growth/create_product_analytics_event.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listProductAnalyticsEvents, "store/growth/list_product_analytics_events.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listProductAnalyticsEventsByUser, "store/growth/list_product_analytics_events_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertExperimentAssignment, "store/admin/upsert_experiment_assignment.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listExperimentAssignments, "store/admin/list_experiment_assignments.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.createSubscriptionChurnFeedback, "store/growth/create_subscription_churn_feedback.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listSubscriptionChurnFeedback, "store/growth/list_subscription_churn_feedback.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listSubscriptionChurnFeedbackByCustomer, "store/growth/list_subscription_churn_feedback_by_customer.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.upsertExperiment, "store/admin/upsert_experiment.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listExperiments, "store/admin/list_experiments.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getAdminDashboardSummary, "store/admin/get_admin_dashboard_summary.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminDashboardDailyUsage, "store/admin/list_admin_dashboard_daily_usage.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminDashboardProviderUsage, "store/admin/list_admin_dashboard_provider_usage.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminDashboardModelUsage, "store/admin/list_admin_dashboard_model_usage.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminDashboardUserUsage, "store/admin/list_admin_dashboard_user_usage.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminUsageEvents, "store/admin/list_admin_usage_events.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminUsers, "store/admin/list_admin_users.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminConversations, "store/admin/list_admin_conversations.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminChatFailedReplies, "store/admin/list_admin_chat_failed_replies.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminChatSlowReplies, "store/admin/list_admin_chat_slow_replies.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminChatHighCostThreads, "store/admin/list_admin_chat_high_cost_threads.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminChatFeatureUsageSlices, "store/admin/list_admin_chat_feature_usage_slices.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminCustomerAccountTimelineEvents, "store/admin/list_admin_customer_account_timeline_events.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminProviderHealthTrends, "store/admin/list_admin_provider_health_trends.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminProviderFallbackEvents, "store/admin/list_admin_provider_fallback_events.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminFailedBackgroundJobs, "store/admin/list_admin_failed_background_jobs.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminFailedNotifications, "store/admin/list_admin_failed_notifications.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminFailedWebhookDeliveries, "store/admin/list_admin_failed_webhook_deliveries.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminIncidentTimeline, "store/admin/list_admin_incident_timeline.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminRecentOpsActions, "store/admin/list_admin_recent_ops_actions.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getAdminBusinessMutationPreview, "store/admin/get_admin_business_mutation_preview.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getAdminProviderMutationPreview, "store/admin/get_admin_provider_mutation_preview.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.getAdminOpsMutationPreview, "store/admin/get_admin_ops_mutation_preview.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminFailedPaymentBillingEvents, "store/admin/list_admin_failed_payment_billing_events.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listAdminBillingDunningTimeline, "store/admin/list_admin_billing_dunning_timeline.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.refreshProviderUsageDailyRollups, "store/ops/refresh_provider_usage_daily_rollups.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseErr = parseLoadStoreQuery(&parseQueries.listProviderUsageDailyRollups, "store/ops/list_provider_usage_daily_rollups.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}

	return parseQueries, nil
}
