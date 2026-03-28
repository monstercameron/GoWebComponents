SELECT
    id,
    incident_key,
    slo_key,
    severity,
    status,
    title,
    summary,
    started_at,
    resolved_at,
    postmortem_url,
    updated_at
FROM incidents
ORDER BY id DESC
LIMIT ?;
