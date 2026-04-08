INSERT INTO user_profile (user_id, selected_thinking_enabled) VALUES (?, ?)
ON CONFLICT(user_id) DO UPDATE SET selected_thinking_enabled=excluded.selected_thinking_enabled
