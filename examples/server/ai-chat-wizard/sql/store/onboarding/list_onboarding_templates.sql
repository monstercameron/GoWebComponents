SELECT id, template_key, title, category, prompt_text, checklist_json, is_default, sort_order, updated_at
FROM onboarding_templates
ORDER BY sort_order ASC, id ASC
LIMIT ?;
