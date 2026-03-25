package main

import "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/internal/sqlfiles"

type seedQueries struct {
	schema             string
	createUser         string
	insertUserProfile  string
	insertConversation string
	insertMessage      string
}

func loadSeedQueries() (seedQueries, error) {
	var queries seedQueries
	var err error

	if queries.schema, err = sqlfiles.Load("store/schema.sql"); err != nil {
		return seedQueries{}, err
	}
	if queries.createUser, err = sqlfiles.Load("store/create_user.sql"); err != nil {
		return seedQueries{}, err
	}
	if queries.insertUserProfile, err = sqlfiles.Load("seed/insert_user_profile.sql"); err != nil {
		return seedQueries{}, err
	}
	if queries.insertConversation, err = sqlfiles.Load("seed/insert_conversation.sql"); err != nil {
		return seedQueries{}, err
	}
	if queries.insertMessage, err = sqlfiles.Load("store/save_conversation_message.sql"); err != nil {
		return seedQueries{}, err
	}

	return queries, nil
}
