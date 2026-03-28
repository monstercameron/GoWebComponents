INSERT INTO user_profile (user_id, selected_thinking_effort) VALUES (?, ?)
ON CONFLICT(user_id) DO UPDATE SET selected_thinking_effort=excluded.selected_thinking_effort
