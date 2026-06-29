UPDATE incidents
SET
    status = ?,
    resolved_at = ?,
    updated_at = ?
WHERE id = ?;
