SELECT
    memory_key,
    category,
    summary,
    detail,
    source_message,
    usefulness_score,
    confidence_score,
    rubric_reason,
    updated_at
FROM user_memory
WHERE user_id = ?
ORDER BY usefulness_score DESC, updated_at DESC, id DESC
