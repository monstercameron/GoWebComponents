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

func loadStoreQueries() (storeQueries, error) {
	var queries storeQueries
	var err error

	if queries.schema, err = sqlfiles.Load("store/schema.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.migrations, err = sqlfiles.Load("store/migrations.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.createUser, err = sqlfiles.Load("store/create_user.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.upsertUserProfileName, err = sqlfiles.Load("store/upsert_user_profile_name.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.getUserAuthByEmail, err = sqlfiles.Load("store/get_user_auth_by_email.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.userExists, err = sqlfiles.Load("store/user_exists.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.conversationOwnedByUser, err = sqlfiles.Load("store/conversation_owned_by_user.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.resolveConversationRoute, err = sqlfiles.Load("store/resolve_conversation_route.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.createConversation, err = sqlfiles.Load("store/create_conversation.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.saveConversationMessage, err = sqlfiles.Load("store/save_conversation_message.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.saveConversationTitle, err = sqlfiles.Load("store/save_conversation_title.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.listConversations, err = sqlfiles.Load("store/list_conversations.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.loadConversation, err = sqlfiles.Load("store/load_conversation.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.deleteConversationMessages, err = sqlfiles.Load("store/delete_conversation_messages.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.deleteConversation, err = sqlfiles.Load("store/delete_conversation.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.getUserName, err = sqlfiles.Load("store/get_user_name.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.setSelectedModel, err = sqlfiles.Load("store/set_selected_model.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.getSelectedModel, err = sqlfiles.Load("store/get_selected_model.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.setSelectedTone, err = sqlfiles.Load("store/set_selected_tone.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.getSelectedTone, err = sqlfiles.Load("store/get_selected_tone.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.setSelectedThinkingEnabled, err = sqlfiles.Load("store/set_selected_thinking_enabled.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.getSelectedThinkingEnabled, err = sqlfiles.Load("store/get_selected_thinking_enabled.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.setSelectedThinkingEffort, err = sqlfiles.Load("store/set_selected_thinking_effort.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.getSelectedThinkingEffort, err = sqlfiles.Load("store/get_selected_thinking_effort.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.setSelectedSystemPrompt, err = sqlfiles.Load("store/set_selected_system_prompt.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.getSelectedSystemPrompt, err = sqlfiles.Load("store/get_selected_system_prompt.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.upsertUserMemory, err = sqlfiles.Load("store/upsert_user_memory.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.listUserMemories, err = sqlfiles.Load("store/list_user_memories.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.deleteUserMemory, err = sqlfiles.Load("store/delete_user_memory.sql"); err != nil {
		return storeQueries{}, err
	}
	if queries.listModelCatalog, err = sqlfiles.Load("store/list_model_catalog.sql"); err != nil {
		return storeQueries{}, err
	}

	return queries, nil
}
