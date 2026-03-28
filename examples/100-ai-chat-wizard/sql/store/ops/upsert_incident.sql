INSERT INTO incidents (
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
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM service_level_objectives WHERE slo_key = ?)
ON CONFLICT(incident_key) DO UPDATE SET
    slo_key = excluded.slo_key,
    severity = excluded.severity,
    status = excluded.status,
    title = excluded.title,
    summary = excluded.summary,
    started_at = excluded.started_at,
    resolved_at = excluded.resolved_at,
    postmortem_url = excluded.postmortem_url,
    updated_at = excluded.updated_at;
