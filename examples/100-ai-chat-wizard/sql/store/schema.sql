CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    email         TEXT NOT NULL COLLATE NOCASE UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS conversations (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    public_id  TEXT NOT NULL,
    started_at TEXT NOT NULL,
    title      TEXT
);
CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id, id DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_public_id ON conversations(public_id);
CREATE TABLE IF NOT EXISTS messages (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id   INTEGER NOT NULL REFERENCES conversations(id),
    role              TEXT    NOT NULL,
    content           TEXT    NOT NULL,
    model_id          TEXT    NOT NULL DEFAULT '',
    prompt_tokens     INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    saved_at          TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id, id);
CREATE TABLE IF NOT EXISTS user_profile (
    user_id                   INTEGER PRIMARY KEY REFERENCES users(id),
    name                      TEXT NOT NULL DEFAULT 'User',
    updated_at                INTEGER NOT NULL DEFAULT 0,
    selected_model            TEXT NOT NULL DEFAULT '',
    selected_tone             TEXT NOT NULL DEFAULT '',
    selected_thinking_enabled INTEGER NOT NULL DEFAULT 1,
    selected_thinking_effort  TEXT NOT NULL DEFAULT 'medium',
    selected_system_prompt    TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_profile_user_id ON user_profile(user_id);
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
