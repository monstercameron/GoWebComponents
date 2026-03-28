package app

import "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/internal/sqlfiles"

type storeQueries struct {
	schema                                string
	migrations                            string
	createUser                            string
	upsertUserProfileName                 string
	getUserAuthByEmail                    string
	userExists                            string
	upsertAuthTokenVersion                string
	getAuthTokenVersion                   string
	upsertAuthSession                     string
	getAuthSessionBySessionID             string
	touchAuthSessionLastSeen              string
	revokeAuthSession                     string
	conversationOwnedByUser               string
	resolveConversationRoute              string
	createConversation                    string
	saveConversationMessage               string
	saveConversationTitle                 string
	listConversations                     string
	loadConversation                      string
	deleteConversationMessages            string
	deleteConversation                    string
	getUserName                           string
	setSelectedModel                      string
	getSelectedModel                      string
	setSelectedTone                       string
	getSelectedTone                       string
	setSelectedThinkingEnabled            string
	getSelectedThinkingEnabled            string
	setSelectedThinkingEffort             string
	getSelectedThinkingEffort             string
	setSelectedSystemPrompt               string
	getSelectedSystemPrompt               string
	upsertUserMemory                      string
	listUserMemories                      string
	deleteUserMemory                      string
	listModelCatalog                      string
	saveUsageEvent                        string
	listUsageEvents                       string
	getModelPricing                       string
	upsertBillingCustomer                 string
	getBillingCustomerByUser              string
	upsertBillingSubscription             string
	getBillingSubscriptionByProvider      string
	listBillingSubscriptionsByCustomer    string
	upsertBillingInvoice                  string
	getBillingInvoiceByProvider           string
	listBillingInvoicesByCustomer         string
	createBillingInvoiceLineItem          string
	listBillingInvoiceLineItems           string
	upsertBillingAccessOverride           string
	listBillingAccessOverridesByCustomer  string
	createBillingEvent                    string
	listBillingEventsByCustomer           string
	listBillingPlanEntitlements           string
	listBillingEffectiveAccessByUser      string
	listBillingEffectiveModelAccessByUser string
	upsertSURole                          string
	listSURoles                           string
	deleteSURolePermissions               string
	insertSURolePermission                string
	listSURolePermissions                 string
	upsertSUUserRole                      string
	listSUUserRoles                       string
	userHasSURole                         string
	upsertSiteConfig                      string
	listSiteConfigs                       string
	upsertFeatureFlag                     string
	listFeatureFlags                      string
	upsertWorkspace                       string
	listWorkspaces                        string
	upsertWorkspaceMembership             string
	listWorkspaceMemberships              string
	listAuthSessions                      string
	upsertWorkspaceInvitation             string
	listWorkspaceInvitations              string
	createAPIKey                          string
	listAPIKeys                           string
	revokeAPIKey                          string
	upsertWebhookEndpoint                 string
	listWebhookEndpoints                  string
	upsertWebhookDelivery                 string
	listWebhookDeliveries                 string
	createAuditLog                        string
	listAuditLogs                         string
	upsertSupportTicket                   string
	listSupportTickets                    string
	createSupportTicketMessage            string
	listSupportTicketMessages             string
	createIncidentUpdate                  string
	listIncidentUpdates                   string
	createNotificationOutbox              string
	listNotificationOutbox                string
	upsertBackgroundJob                   string
	listBackgroundJobs                    string
	upsertOnboardingTemplate              string
	listOnboardingTemplates               string
	upsertUserActivationMilestone         string
	listUserActivationMilestones          string
	upsertSavedWorkflow                   string
	listSavedWorkflows                    string
	upsertPromptLibraryItem               string
	listPromptLibraryItems                string
	upsertWeeklyValueSummary              string
	listWeeklyValueSummaries              string
	createProductAnalyticsEvent           string
	listProductAnalyticsEvents            string
	upsertExperimentAssignment            string
	listExperimentAssignments             string
	createSubscriptionChurnFeedback       string
	listSubscriptionChurnFeedback         string
	upsertExperiment                      string
	listExperiments                       string
	getAdminDashboardSummary              string
	listAdminDashboardDailyUsage          string
	listAdminDashboardProviderUsage       string
	listAdminDashboardModelUsage          string
	listAdminDashboardUserUsage           string
	listAdminUsageEvents                  string
	listAdminUsers                        string
	listAdminConversations                string
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
	if parseQueries.upsertAuthTokenVersion, parseErr = sqlfiles.ParseLoad("store/upsert_auth_token_version.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getAuthTokenVersion, parseErr = sqlfiles.ParseLoad("store/get_auth_token_version.sql"); parseErr != nil {
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
	if parseQueries.listBillingPlanEntitlements, parseErr = sqlfiles.ParseLoad("store/list_billing_plan_entitlements.sql"); parseErr != nil {
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
	if parseQueries.upsertFeatureFlag, parseErr = sqlfiles.ParseLoad("store/upsert_feature_flag.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listFeatureFlags, parseErr = sqlfiles.ParseLoad("store/list_feature_flags.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertWorkspace, parseErr = sqlfiles.ParseLoad("store/upsert_workspace.sql"); parseErr != nil {
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
	if parseQueries.createNotificationOutbox, parseErr = sqlfiles.ParseLoad("store/create_notification_outbox.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.listNotificationOutbox, parseErr = sqlfiles.ParseLoad("store/list_notification_outbox.sql"); parseErr != nil {
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
