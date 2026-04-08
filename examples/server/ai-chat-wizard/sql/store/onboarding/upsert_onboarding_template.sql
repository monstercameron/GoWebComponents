INSERT INTO onboarding_templates (
    template_key,
    title,
    category,
    prompt_text,
    checklist_json,
    is_default,
    sort_order,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(template_key) DO UPDATE SET
    title = excluded.title,
    category = excluded.category,
    prompt_text = excluded.prompt_text,
    checklist_json = excluded.checklist_json,
    is_default = excluded.is_default,
    sort_order = excluded.sort_order,
    updated_at = excluded.updated_at;
