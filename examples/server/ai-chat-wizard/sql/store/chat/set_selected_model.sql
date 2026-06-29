INSERT INTO user_profile (user_id, selected_model) VALUES (?, ?)
ON CONFLICT(user_id) DO UPDATE SET selected_model=excluded.selected_model
