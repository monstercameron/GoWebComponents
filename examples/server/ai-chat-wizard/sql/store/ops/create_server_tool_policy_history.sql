INSERT INTO server_tool_policy_history (
    is_enabled,
    max_session_seconds,
    max_output_bytes,
    approved_tools_json,
    updated_by_user_id,
    source,
    created_at
)
VALUES (?, ?, ?, ?, ?, ?, ?);
