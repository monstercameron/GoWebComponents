INSERT INTO user_profile (user_id, selected_tone) VALUES (?, ?)
ON CONFLICT(user_id) DO UPDATE SET selected_tone=excluded.selected_tone
