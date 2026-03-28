package app

import "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/internal/sqlfiles"

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
}

func parseLoadStoreQueries() (storeQueries, error) {
	var parseQueries storeQueries
	var parseErr error

	if parseQueries.schema, parseErr = sqlfiles.ParseLoad("store/schema.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.migrations, parseErr = sqlfiles.ParseLoad("store/migrations.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.createUser, parseErr = sqlfiles.ParseLoad("store/create_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertUserProfileName, parseErr = sqlfiles.ParseLoad("store/upsert_user_profile_name.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getUserAuthByEmail, parseErr = sqlfiles.ParseLoad("store/get_user_auth_by_email.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.userExists, parseErr = sqlfiles.ParseLoad("store/user_exists.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertUserAccessState, parseErr = sqlfiles.ParseLoad("store/upsert_user_access_state.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getUserAccessState, parseErr = sqlfiles.ParseLoad("store/get_user_access_state.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertAuthTokenVersion, parseErr = sqlfiles.ParseLoad("store/upsert_auth_token_version.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getAuthTokenVersion, parseErr = sqlfiles.ParseLoad("store/get_auth_token_version.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertAuthIdentity, parseErr = sqlfiles.ParseLoad("store/upsert_auth_identity.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getAuthIdentityByProviderSubject, parseErr = sqlfiles.ParseLoad("store/get_auth_identity_by_provider_subject.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listAuthIdentitiesByUser, parseErr = sqlfiles.ParseLoad("store/list_auth_identities_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteAuthIdentityByScope, parseErr = sqlfiles.ParseLoad("store/delete_auth_identity_by_scope.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.createAuthOIDCState, parseErr = sqlfiles.ParseLoad("store/create_auth_oidc_state.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getAuthOIDCStateByStateTokenHash, parseErr = sqlfiles.ParseLoad("store/get_auth_oidc_state_by_state_token_hash.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.consumeAuthOIDCStateByStateTokenHash, parseErr = sqlfiles.ParseLoad("store/consume_auth_oidc_state_by_state_token_hash.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteExpiredAuthOIDCStates, parseErr = sqlfiles.ParseLoad("store/delete_expired_auth_oidc_states.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertWorkspaceAuthPolicy, parseErr = sqlfiles.ParseLoad("store/upsert_workspace_auth_policy.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getWorkspaceAuthPolicyByWorkspace, parseErr = sqlfiles.ParseLoad("store/get_workspace_auth_policy_by_workspace.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertUserAuthBlock, parseErr = sqlfiles.ParseLoad("store/upsert_user_auth_block.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.countUserAuthBlocksByUser, parseErr = sqlfiles.ParseLoad("store/count_user_auth_blocks_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteUserAuthBlock, parseErr = sqlfiles.ParseLoad("store/delete_user_auth_block.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteUserAuthBlocksByKey, parseErr = sqlfiles.ParseLoad("store/delete_user_auth_blocks_by_key.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertAuthSession, parseErr = sqlfiles.ParseLoad("store/upsert_auth_session.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getAuthSessionBySessionID, parseErr = sqlfiles.ParseLoad("store/get_auth_session_by_session_id.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.touchAuthSessionLastSeen, parseErr = sqlfiles.ParseLoad("store/touch_auth_session_last_seen.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.revokeAuthSession, parseErr = sqlfiles.ParseLoad("store/revoke_auth_session.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.revokeAuthSessionsByUser, parseErr = sqlfiles.ParseLoad("store/revoke_auth_sessions_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.updateUserPasswordHash, parseErr = sqlfiles.ParseLoad("store/update_user_password_hash.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.createEmailVerificationToken, parseErr = sqlfiles.ParseLoad("store/create_email_verification_token.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getEmailVerificationTokenByHash, parseErr = sqlfiles.ParseLoad("store/get_email_verification_token_by_hash.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.markEmailVerificationTokenVerified, parseErr = sqlfiles.ParseLoad("store/mark_email_verification_token_verified.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.createPasswordResetToken, parseErr = sqlfiles.ParseLoad("store/create_password_reset_token.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getPasswordResetTokenByHash, parseErr = sqlfiles.ParseLoad("store/get_password_reset_token_by_hash.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.consumePasswordResetToken, parseErr = sqlfiles.ParseLoad("store/consume_password_reset_token.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.conversationOwnedByUser, parseErr = sqlfiles.ParseLoad("store/conversation_owned_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.resolveConversationRoute, parseErr = sqlfiles.ParseLoad("store/resolve_conversation_route.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.createConversation, parseErr = sqlfiles.ParseLoad("store/create_conversation.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.saveConversationMessage, parseErr = sqlfiles.ParseLoad("store/save_conversation_message.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.saveConversationTitle, parseErr = sqlfiles.ParseLoad("store/save_conversation_title.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listConversations, parseErr = sqlfiles.ParseLoad("store/list_conversations.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.loadConversation, parseErr = sqlfiles.ParseLoad("store/load_conversation.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteConversationMessages, parseErr = sqlfiles.ParseLoad("store/delete_conversation_messages.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteConversation, parseErr = sqlfiles.ParseLoad("store/delete_conversation.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getUserName, parseErr = sqlfiles.ParseLoad("store/get_user_name.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.setSelectedModel, parseErr = sqlfiles.ParseLoad("store/set_selected_model.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getSelectedModel, parseErr = sqlfiles.ParseLoad("store/get_selected_model.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.setSelectedTone, parseErr = sqlfiles.ParseLoad("store/set_selected_tone.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getSelectedTone, parseErr = sqlfiles.ParseLoad("store/get_selected_tone.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.setSelectedThinkingEnabled, parseErr = sqlfiles.ParseLoad("store/set_selected_thinking_enabled.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getSelectedThinkingEnabled, parseErr = sqlfiles.ParseLoad("store/get_selected_thinking_enabled.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.setSelectedThinkingEffort, parseErr = sqlfiles.ParseLoad("store/set_selected_thinking_effort.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getSelectedThinkingEffort, parseErr = sqlfiles.ParseLoad("store/get_selected_thinking_effort.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.setSelectedSystemPrompt, parseErr = sqlfiles.ParseLoad("store/set_selected_system_prompt.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getSelectedSystemPrompt, parseErr = sqlfiles.ParseLoad("store/get_selected_system_prompt.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertUserMemory, parseErr = sqlfiles.ParseLoad("store/upsert_user_memory.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listUserMemories, parseErr = sqlfiles.ParseLoad("store/list_user_memories.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteUserMemory, parseErr = sqlfiles.ParseLoad("store/delete_user_memory.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listModelCatalog, parseErr = sqlfiles.ParseLoad("store/list_model_catalog.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.saveUsageEvent, parseErr = sqlfiles.ParseLoad("store/save_usage_event.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listUsageEvents, parseErr = sqlfiles.ParseLoad("store/list_usage_events.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.sumUsageTokensSince, parseErr = sqlfiles.ParseLoad("store/sum_usage_tokens_since.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getModelPricing, parseErr = sqlfiles.ParseLoad("store/get_model_pricing.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertBillingCustomer, parseErr = sqlfiles.ParseLoad("store/upsert_billing_customer.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getBillingCustomerByUser, parseErr = sqlfiles.ParseLoad("store/get_billing_customer_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertBillingSubscription, parseErr = sqlfiles.ParseLoad("store/upsert_billing_subscription.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getBillingSubscriptionByProvider, parseErr = sqlfiles.ParseLoad("store/get_billing_subscription_by_provider.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listBillingSubscriptionsByCustomer, parseErr = sqlfiles.ParseLoad("store/list_billing_subscriptions_by_customer.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertBillingInvoice, parseErr = sqlfiles.ParseLoad("store/upsert_billing_invoice.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getBillingInvoiceByProvider, parseErr = sqlfiles.ParseLoad("store/get_billing_invoice_by_provider.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listBillingInvoicesByCustomer, parseErr = sqlfiles.ParseLoad("store/list_billing_invoices_by_customer.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.createBillingInvoiceLineItem, parseErr = sqlfiles.ParseLoad("store/create_billing_invoice_line_item.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listBillingInvoiceLineItems, parseErr = sqlfiles.ParseLoad("store/list_billing_invoice_line_items.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertBillingAccessOverride, parseErr = sqlfiles.ParseLoad("store/upsert_billing_access_override.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listBillingAccessOverridesByCustomer, parseErr = sqlfiles.ParseLoad("store/list_billing_access_overrides_by_customer.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.createBillingEvent, parseErr = sqlfiles.ParseLoad("store/create_billing_event.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listBillingEventsByCustomer, parseErr = sqlfiles.ParseLoad("store/list_billing_events_by_customer.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listBillingDunningEventsByCustomer, parseErr = sqlfiles.ParseLoad("store/list_billing_dunning_events_by_customer.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getBillingDunningEventByID, parseErr = sqlfiles.ParseLoad("store/get_billing_dunning_event_by_id.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertBillingDunningEvent, parseErr = sqlfiles.ParseLoad("store/upsert_billing_dunning_event.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteBillingDunningEvent, parseErr = sqlfiles.ParseLoad("store/delete_billing_dunning_event.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.updateBillingDunningEventResolved, parseErr = sqlfiles.ParseLoad("store/update_billing_dunning_event_resolved.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.updateBillingInvoiceResolution, parseErr = sqlfiles.ParseLoad("store/update_billing_invoice_resolution.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listBillingPlans, parseErr = sqlfiles.ParseLoad("store/list_billing_plans.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertBillingPlan, parseErr = sqlfiles.ParseLoad("store/upsert_billing_plan.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteBillingPlan, parseErr = sqlfiles.ParseLoad("store/delete_billing_plan.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listBillingPlanEntitlements, parseErr = sqlfiles.ParseLoad("store/list_billing_plan_entitlements.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertBillingPlanEntitlement, parseErr = sqlfiles.ParseLoad("store/upsert_billing_plan_entitlement.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteBillingPlanEntitlement, parseErr = sqlfiles.ParseLoad("store/delete_billing_plan_entitlement.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listBillingPlanOverages, parseErr = sqlfiles.ParseLoad("store/list_billing_plan_overages.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertBillingPlanOverage, parseErr = sqlfiles.ParseLoad("store/upsert_billing_plan_overage.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteBillingPlanOverage, parseErr = sqlfiles.ParseLoad("store/delete_billing_plan_overage.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listBillingQuotaPolicies, parseErr = sqlfiles.ParseLoad("store/list_billing_quota_policies.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertBillingQuotaPolicy, parseErr = sqlfiles.ParseLoad("store/upsert_billing_quota_policy.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteBillingQuotaPolicy, parseErr = sqlfiles.ParseLoad("store/delete_billing_quota_policy.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listBillingUpgradeTriggers, parseErr = sqlfiles.ParseLoad("store/list_billing_upgrade_triggers.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertBillingUpgradeTrigger, parseErr = sqlfiles.ParseLoad("store/upsert_billing_upgrade_trigger.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteBillingUpgradeTrigger, parseErr = sqlfiles.ParseLoad("store/delete_billing_upgrade_trigger.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listBillingEffectiveAccessByUser, parseErr = sqlfiles.ParseLoad("store/list_billing_effective_access_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listBillingEffectiveModelAccessByUser, parseErr = sqlfiles.ParseLoad("store/list_billing_effective_model_access_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertSURole, parseErr = sqlfiles.ParseLoad("store/upsert_su_role.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listSURoles, parseErr = sqlfiles.ParseLoad("store/list_su_roles.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteSURolePermissions, parseErr = sqlfiles.ParseLoad("store/delete_su_role_permissions.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.insertSURolePermission, parseErr = sqlfiles.ParseLoad("store/insert_su_role_permission.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listSURolePermissions, parseErr = sqlfiles.ParseLoad("store/list_su_role_permissions.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertSUUserRole, parseErr = sqlfiles.ParseLoad("store/upsert_su_user_role.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listSUUserRoles, parseErr = sqlfiles.ParseLoad("store/list_su_user_roles.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.userHasSURole, parseErr = sqlfiles.ParseLoad("store/user_has_su_role.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertSiteConfig, parseErr = sqlfiles.ParseLoad("store/upsert_site_config.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listSiteConfigs, parseErr = sqlfiles.ParseLoad("store/list_site_configs.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.createServerToolPolicyHistory, parseErr = sqlfiles.ParseLoad("store/create_server_tool_policy_history.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listServerToolPolicyHistory, parseErr = sqlfiles.ParseLoad("store/list_server_tool_policy_history.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertFeatureFlag, parseErr = sqlfiles.ParseLoad("store/upsert_feature_flag.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listFeatureFlags, parseErr = sqlfiles.ParseLoad("store/list_feature_flags.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertWorkspace, parseErr = sqlfiles.ParseLoad("store/upsert_workspace.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getWorkspaceByID, parseErr = sqlfiles.ParseLoad("store/get_workspace_by_id.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listWorkspaces, parseErr = sqlfiles.ParseLoad("store/list_workspaces.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertWorkspaceMembership, parseErr = sqlfiles.ParseLoad("store/upsert_workspace_membership.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listWorkspaceMemberships, parseErr = sqlfiles.ParseLoad("store/list_workspace_memberships.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listWorkspaceMembershipsByUser, parseErr = sqlfiles.ParseLoad("store/list_workspace_memberships_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listWorkspaceMembershipsByWorkspace, parseErr = sqlfiles.ParseLoad("store/list_workspace_memberships_by_workspace.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.countWorkspaceMembershipsByUser, parseErr = sqlfiles.ParseLoad("store/count_workspace_memberships_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.countActiveWorkspaceMembershipsByUser, parseErr = sqlfiles.ParseLoad("store/count_active_workspace_memberships_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listWorkspaceSSOConfigs, parseErr = sqlfiles.ParseLoad("store/list_workspace_sso_configs.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getWorkspaceSSOConfigByScope, parseErr = sqlfiles.ParseLoad("store/get_workspace_sso_config_by_scope.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertWorkspaceSSOConfig, parseErr = sqlfiles.ParseLoad("store/upsert_workspace_sso_config.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteWorkspaceSSOConfig, parseErr = sqlfiles.ParseLoad("store/delete_workspace_sso_config.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listDataRetentionPolicies, parseErr = sqlfiles.ParseLoad("store/list_data_retention_policies.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getDataRetentionPolicyByScope, parseErr = sqlfiles.ParseLoad("store/get_data_retention_policy_by_scope.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertDataRetentionPolicy, parseErr = sqlfiles.ParseLoad("store/upsert_data_retention_policy.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteDataRetentionPolicy, parseErr = sqlfiles.ParseLoad("store/delete_data_retention_policy.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listComplianceControls, parseErr = sqlfiles.ParseLoad("store/list_compliance_controls.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getComplianceControlByScope, parseErr = sqlfiles.ParseLoad("store/get_compliance_control_by_scope.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertComplianceControl, parseErr = sqlfiles.ParseLoad("store/upsert_compliance_control.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteComplianceControl, parseErr = sqlfiles.ParseLoad("store/delete_compliance_control.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listServiceLevelObjectives, parseErr = sqlfiles.ParseLoad("store/list_service_level_objectives.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getServiceLevelObjectiveByKey, parseErr = sqlfiles.ParseLoad("store/get_service_level_objective_by_key.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertServiceLevelObjective, parseErr = sqlfiles.ParseLoad("store/upsert_service_level_objective.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteServiceLevelObjective, parseErr = sqlfiles.ParseLoad("store/delete_service_level_objective.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listAuthSessions, parseErr = sqlfiles.ParseLoad("store/list_auth_sessions.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertWorkspaceInvitation, parseErr = sqlfiles.ParseLoad("store/upsert_workspace_invitation.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listWorkspaceInvitations, parseErr = sqlfiles.ParseLoad("store/list_workspace_invitations.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.createAPIKey, parseErr = sqlfiles.ParseLoad("store/create_api_key.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listAPIKeys, parseErr = sqlfiles.ParseLoad("store/list_api_keys.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.revokeAPIKey, parseErr = sqlfiles.ParseLoad("store/revoke_api_key.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertWebhookEndpoint, parseErr = sqlfiles.ParseLoad("store/upsert_webhook_endpoint.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listWebhookEndpoints, parseErr = sqlfiles.ParseLoad("store/list_webhook_endpoints.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertWebhookDelivery, parseErr = sqlfiles.ParseLoad("store/upsert_webhook_delivery.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listWebhookDeliveries, parseErr = sqlfiles.ParseLoad("store/list_webhook_deliveries.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listWebhookDeliveriesPendingRetry, parseErr = sqlfiles.ParseLoad("store/list_webhook_deliveries_pending_retry.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.updateWebhookDeliveryAttempt, parseErr = sqlfiles.ParseLoad("store/update_webhook_delivery_attempt.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.updateWebhookDeliveryDelivered, parseErr = sqlfiles.ParseLoad("store/update_webhook_delivery_delivered.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.createAuditLog, parseErr = sqlfiles.ParseLoad("store/create_audit_log.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listAuditLogs, parseErr = sqlfiles.ParseLoad("store/list_audit_logs.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertSupportTicket, parseErr = sqlfiles.ParseLoad("store/upsert_support_ticket.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listSupportTickets, parseErr = sqlfiles.ParseLoad("store/list_support_tickets.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.createSupportTicketMessage, parseErr = sqlfiles.ParseLoad("store/create_support_ticket_message.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listSupportTicketMessages, parseErr = sqlfiles.ParseLoad("store/list_support_ticket_messages.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.createIncidentUpdate, parseErr = sqlfiles.ParseLoad("store/create_incident_update.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listIncidentUpdates, parseErr = sqlfiles.ParseLoad("store/list_incident_updates.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getIncidentUpdateByID, parseErr = sqlfiles.ParseLoad("store/get_incident_update_by_id.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertIncidentUpdate, parseErr = sqlfiles.ParseLoad("store/upsert_incident_update.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteIncidentUpdate, parseErr = sqlfiles.ParseLoad("store/delete_incident_update.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.updateIncidentStatus, parseErr = sqlfiles.ParseLoad("store/update_incident_status.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listIncidents, parseErr = sqlfiles.ParseLoad("store/list_incidents.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getIncidentByKey, parseErr = sqlfiles.ParseLoad("store/get_incident_by_key.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertIncident, parseErr = sqlfiles.ParseLoad("store/upsert_incident.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteIncident, parseErr = sqlfiles.ParseLoad("store/delete_incident.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listWorkspaceCostGuardrails, parseErr = sqlfiles.ParseLoad("store/list_workspace_cost_guardrails.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.createNotificationOutbox, parseErr = sqlfiles.ParseLoad("store/create_notification_outbox.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listNotificationOutbox, parseErr = sqlfiles.ParseLoad("store/list_notification_outbox.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listNotificationOutboxPending, parseErr = sqlfiles.ParseLoad("store/list_notification_outbox_pending.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.updateNotificationOutboxStatus, parseErr = sqlfiles.ParseLoad("store/update_notification_outbox_status.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertBackgroundJob, parseErr = sqlfiles.ParseLoad("store/upsert_background_job.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listBackgroundJobs, parseErr = sqlfiles.ParseLoad("store/list_background_jobs.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertOnboardingTemplate, parseErr = sqlfiles.ParseLoad("store/upsert_onboarding_template.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listOnboardingTemplates, parseErr = sqlfiles.ParseLoad("store/list_onboarding_templates.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertUserActivationMilestone, parseErr = sqlfiles.ParseLoad("store/upsert_user_activation_milestone.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listUserActivationMilestones, parseErr = sqlfiles.ParseLoad("store/list_user_activation_milestones.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertSavedWorkflow, parseErr = sqlfiles.ParseLoad("store/upsert_saved_workflow.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listSavedWorkflows, parseErr = sqlfiles.ParseLoad("store/list_saved_workflows.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertPromptLibraryItem, parseErr = sqlfiles.ParseLoad("store/upsert_prompt_library_item.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listPromptLibraryItems, parseErr = sqlfiles.ParseLoad("store/list_prompt_library_items.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertWeeklyValueSummary, parseErr = sqlfiles.ParseLoad("store/upsert_weekly_value_summary.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listWeeklyValueSummaries, parseErr = sqlfiles.ParseLoad("store/list_weekly_value_summaries.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.createProductAnalyticsEvent, parseErr = sqlfiles.ParseLoad("store/create_product_analytics_event.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listProductAnalyticsEvents, parseErr = sqlfiles.ParseLoad("store/list_product_analytics_events.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listProductAnalyticsEventsByUser, parseErr = sqlfiles.ParseLoad("store/list_product_analytics_events_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertExperimentAssignment, parseErr = sqlfiles.ParseLoad("store/upsert_experiment_assignment.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listExperimentAssignments, parseErr = sqlfiles.ParseLoad("store/list_experiment_assignments.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.createSubscriptionChurnFeedback, parseErr = sqlfiles.ParseLoad("store/create_subscription_churn_feedback.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listSubscriptionChurnFeedback, parseErr = sqlfiles.ParseLoad("store/list_subscription_churn_feedback.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listSubscriptionChurnFeedbackByCustomer, parseErr = sqlfiles.ParseLoad("store/list_subscription_churn_feedback_by_customer.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertExperiment, parseErr = sqlfiles.ParseLoad("store/upsert_experiment.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listExperiments, parseErr = sqlfiles.ParseLoad("store/list_experiments.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getAdminDashboardSummary, parseErr = sqlfiles.ParseLoad("store/get_admin_dashboard_summary.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listAdminDashboardDailyUsage, parseErr = sqlfiles.ParseLoad("store/list_admin_dashboard_daily_usage.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listAdminDashboardProviderUsage, parseErr = sqlfiles.ParseLoad("store/list_admin_dashboard_provider_usage.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listAdminDashboardModelUsage, parseErr = sqlfiles.ParseLoad("store/list_admin_dashboard_model_usage.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listAdminDashboardUserUsage, parseErr = sqlfiles.ParseLoad("store/list_admin_dashboard_user_usage.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listAdminUsageEvents, parseErr = sqlfiles.ParseLoad("store/list_admin_usage_events.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listAdminUsers, parseErr = sqlfiles.ParseLoad("store/list_admin_users.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listAdminConversations, parseErr = sqlfiles.ParseLoad("store/list_admin_conversations.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}

	return parseQueries, nil
}
