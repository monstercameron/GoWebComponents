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
CREATE TABLE IF NOT EXISTS model_catalog (
    id                             TEXT PRIMARY KEY,
    provider_id                    TEXT NOT NULL,
    provider_label                 TEXT NOT NULL,
    label                          TEXT NOT NULL,
    note                           TEXT NOT NULL DEFAULT '',
    description                    TEXT NOT NULL DEFAULT '',
    supports_thinking              INTEGER NOT NULL DEFAULT 0,
    supports_speech                INTEGER NOT NULL DEFAULT 0,
    input_cost_per_million_usd     REAL NOT NULL DEFAULT 0,
    output_cost_per_million_usd    REAL NOT NULL DEFAULT 0,
    pricing_currency               TEXT NOT NULL DEFAULT 'USD',
    max_output_tokens              INTEGER NOT NULL DEFAULT 0,
    throughput_tokens_per_second   REAL NOT NULL DEFAULT 0,
    onboarding_ready               INTEGER NOT NULL DEFAULT 1,
    is_default                     INTEGER NOT NULL DEFAULT 0,
    use_for_title_generation       INTEGER NOT NULL DEFAULT 0,
    use_for_memory_extraction      INTEGER NOT NULL DEFAULT 0,
    sort_order                     INTEGER NOT NULL DEFAULT 0
);

INSERT OR IGNORE INTO model_catalog (
    id, provider_id, provider_label, label, note, description,
    supports_thinking, supports_speech,
    input_cost_per_million_usd, output_cost_per_million_usd, pricing_currency,
    max_output_tokens, throughput_tokens_per_second, onboarding_ready,
    is_default, use_for_title_generation, use_for_memory_extraction, sort_order
) VALUES
    ('gpt-5.4', 'openai', 'OpenAI', 'GPT-5.4', 'Best', 'General-purpose frontier reasoning and coding model.', 1, 1, 1.25, 10.00, 'USD', 0, 0, 1, 0, 0, 1, 10),
    ('gpt-5.4-mini', 'openai', 'OpenAI', 'GPT-5.4 mini', 'Fast', 'Balanced default chat model for fast interactive use.', 1, 1, 0.25, 2.00, 'USD', 0, 0, 1, 1, 0, 0, 20),
    ('gpt-5.4-nano', 'openai', 'OpenAI', 'GPT-5.4 nano', 'Cheap', 'Lowest-cost OpenAI option for lightweight background tasks.', 1, 1, 0.05, 0.40, 'USD', 0, 0, 1, 0, 1, 0, 30),
    ('claude-sonnet-4-5', 'anthropic', 'Anthropic', 'Claude Sonnet 4.5', 'Reasoning', 'Balanced Anthropic reasoning model for general chat and coding flows.', 1, 0, 0, 0, 'USD', 4096, 0, 1, 0, 0, 0, 40),
    ('claude-haiku-4-5', 'anthropic', 'Anthropic', 'Claude Haiku 4.5', 'Fast', 'Faster Anthropic model for lightweight generation tasks.', 1, 0, 0, 0, 'USD', 4096, 0, 1, 0, 1, 0, 50),
    ('gpt-oss-120b', 'cerebras', 'Cerebras', 'GPT OSS 120B', 'Reasoning', 'Cerebras-hosted reasoning model with strong coding and general chat performance.', 1, 0, 0, 0, 'USD', 4096, 3000, 1, 0, 0, 0, 60),
    ('llama3.1-8b', 'cerebras', 'Cerebras', 'Llama 3.1 8B', 'Fast', 'Fast Cerebras-hosted general model for lightweight generation tasks.', 0, 0, 0, 0, 'USD', 4096, 2200, 1, 0, 1, 0, 70),
    ('qwen-3-235b-a22b-instruct-2507', 'cerebras', 'Cerebras', 'Qwen 3 235B Instruct', 'Preview', 'Preview Cerebras-hosted large model for evaluation.', 0, 0, 0, 0, 'USD', 4096, 1400, 1, 0, 0, 0, 80),
    ('zai-glm-4.7', 'cerebras', 'Cerebras', 'Z.ai GLM 4.7', 'Preview', 'Preview Cerebras-hosted reasoning-capable model for evaluation.', 1, 0, 0, 0, 'USD', 4096, 1000, 1, 0, 0, 0, 90);

-- Keep capability flags accurate for existing databases that already contain
-- earlier catalog rows inserted before these defaults were updated.
UPDATE model_catalog
SET supports_thinking = 0,
    description = 'Preview Cerebras-hosted large model for evaluation.'
WHERE id = 'qwen-3-235b-a22b-instruct-2507';
