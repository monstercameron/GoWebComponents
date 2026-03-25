package app

import "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/internal/sqlfiles"

type storeQueries struct {
	schema                     string
	migrations                 string
	createUser                 string
	upsertUserProfileName      string
	getUserAuthByEmail         string
	userExists                 string
	conversationOwnedByUser    string
	resolveConversationRoute   string
	createConversation         string
	saveConversationMessage    string
	saveConversationTitle      string
	listConversations          string
	loadConversation           string
	deleteConversationMessages string
	deleteConversation         string
	getUserName                string
	setSelectedModel           string
	getSelectedModel           string
	setSelectedTone            string
	getSelectedTone            string
	setSelectedThinkingEnabled string
	getSelectedThinkingEnabled string
	setSelectedThinkingEffort  string
	getSelectedThinkingEffort  string
	setSelectedSystemPrompt    string
	getSelectedSystemPrompt    string
	upsertUserMemory           string
	listUserMemories           string
	deleteUserMemory           string
	listModelCatalog           string
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
	if parseQueries.parseCreateUser, parseErr = sqlfiles.ParseLoad("store/create_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.upsertUserProfileName, parseErr = sqlfiles.ParseLoad("store/upsert_user_profile_name.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.getUserAuthByEmail, parseErr = sqlfiles.ParseLoad("store/get_user_auth_by_email.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.parseUserExists, parseErr = sqlfiles.ParseLoad("store/user_exists.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.parseConversationOwnedByUser, parseErr = sqlfiles.ParseLoad("store/conversation_owned_by_user.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.parseResolveConversationRoute, parseErr = sqlfiles.ParseLoad("store/resolve_conversation_route.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.parseCreateConversation, parseErr = sqlfiles.ParseLoad("store/create_conversation.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.parseSaveConversationMessage, parseErr = sqlfiles.ParseLoad("store/save_conversation_message.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.parseSaveConversationTitle, parseErr = sqlfiles.ParseLoad("store/save_conversation_title.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.parseListConversations, parseErr = sqlfiles.ParseLoad("store/list_conversations.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.parseLoadConversation, parseErr = sqlfiles.ParseLoad("store/load_conversation.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.deleteConversationMessages, parseErr = sqlfiles.ParseLoad("store/delete_conversation_messages.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.parseDeleteConversation, parseErr = sqlfiles.ParseLoad("store/delete_conversation.sql"); parseErr != nil {
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
	if parseQueries.parseUpsertUserMemory, parseErr = sqlfiles.ParseLoad("store/upsert_user_memory.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.parseListUserMemories, parseErr = sqlfiles.ParseLoad("store/list_user_memories.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.parseDeleteUserMemory, parseErr = sqlfiles.ParseLoad("store/delete_user_memory.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}
	if parseQueries.parseListModelCatalog, parseErr = sqlfiles.ParseLoad("store/list_model_catalog.sql"); parseErr != nil {
		return storeQueries{}, parseErr
	}

	return parseQueries, nil
}
