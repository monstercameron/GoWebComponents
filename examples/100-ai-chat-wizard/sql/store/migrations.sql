ALTER TABLE conversations ADD COLUMN title TEXT;
ALTER TABLE conversations ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE conversations ADD COLUMN public_id TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN model_id TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN prompt_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE messages ADD COLUMN completion_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE user_profile ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE user_profile ADD COLUMN selected_model TEXT NOT NULL DEFAULT '';
ALTER TABLE user_profile ADD COLUMN selected_tone TEXT NOT NULL DEFAULT '';
ALTER TABLE user_profile ADD COLUMN selected_thinking_enabled INTEGER NOT NULL DEFAULT 1;
ALTER TABLE user_profile ADD COLUMN selected_thinking_effort TEXT NOT NULL DEFAULT 'medium';
ALTER TABLE user_profile ADD COLUMN selected_system_prompt TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_profile_user_id ON user_profile(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id, id);
CREATE TABLE IF NOT EXISTS user_memory (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id           INTEGER NOT NULL REFERENCES users(id),
    memory_key        TEXT NOT NULL,
    category          TEXT NOT NULL,
    summary           TEXT NOT NULL,
    detail            TEXT NOT NULL DEFAULT '',
    source_message    TEXT NOT NULL DEFAULT '',
    usefulness_score  INTEGER NOT NULL DEFAULT 0,
    confidence_score  REAL NOT NULL DEFAULT 0,
    rubric_reason     TEXT NOT NULL DEFAULT '',
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL,
    UNIQUE(user_id, memory_key)
);
CREATE INDEX IF NOT EXISTS idx_user_memory_user_id ON user_memory(user_id, usefulness_score DESC, updated_at DESC);
