INSERT INTO user_memory (
    user_id,
    memory_key,
    category,
    summary,
    detail,
    source_message,
    usefulness_score,
    confidence_score,
    rubric_reason,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id, memory_key) DO UPDATE SET
    category=excluded.category,
    summary=excluded.summary,
    detail=excluded.detail,
    source_message=excluded.source_message,
    usefulness_score=excluded.usefulness_score,
    confidence_score=excluded.confidence_score,
    rubric_reason=excluded.rubric_reason,
    updated_at=excluded.updated_at
