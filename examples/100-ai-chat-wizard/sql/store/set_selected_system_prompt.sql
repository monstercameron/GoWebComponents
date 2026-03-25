INSERT INTO user_profile (user_id, selected_system_prompt) VALUES (?, ?)
ON CONFLICT(user_id) DO UPDATE SET selected_system_prompt=excluded.selected_system_prompt
