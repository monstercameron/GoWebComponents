package main

import "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/internal/sqlfiles"

type seedQueries struct {
	schema             string
	parseCreateUser    string
	upsertBillingUser  string
	insertUserProfile  string
	insertConversation string
	insertMessage      string
}

func parseLoadSeedQueries() (seedQueries, error) {
	var parseQueries seedQueries
	var parseErr error

	if parseQueries.schema, parseErr = sqlfiles.ParseLoad("store/schema.sql"); parseErr != nil {
		return seedQueries{}, parseErr
	}
	if parseQueries.parseCreateUser, parseErr = sqlfiles.ParseLoad("store/create_user.sql"); parseErr != nil {
		return seedQueries{}, parseErr
	}
	if parseQueries.upsertBillingUser, parseErr = sqlfiles.ParseLoad("store/upsert_billing_customer.sql"); parseErr != nil {
		return seedQueries{}, parseErr
	}
	if parseQueries.insertUserProfile, parseErr = sqlfiles.ParseLoad("seed/insert_user_profile.sql"); parseErr != nil {
		return seedQueries{}, parseErr
	}
	if parseQueries.insertConversation, parseErr = sqlfiles.ParseLoad("seed/insert_conversation.sql"); parseErr != nil {
		return seedQueries{}, parseErr
	}
	if parseQueries.insertMessage, parseErr = sqlfiles.ParseLoad("store/save_conversation_message.sql"); parseErr != nil {
		return seedQueries{}, parseErr
	}

	return parseQueries, nil
}
